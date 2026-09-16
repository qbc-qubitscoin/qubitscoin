package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/upgrade"
)

func newTestGenesis(valAddr [crypto.AddressSize]byte) (*core.Block, *state.DB) {
	st := state.NewStateDB()
	cfg := core.DefaultGenesisConfig(valAddr)
	cfg.Allocations = map[[crypto.AddressSize]byte]uint64{
		valAddr: 1_000_000 * core.OneQBC,
	}
	st.SetAccount(valAddr, &state.Account{Balance: 1_000_000 * core.OneQBC})
	blk := cfg.Build()
	return blk, st
}

func TestEngine_ProduceBlock_Internal(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	st := state.NewStateDB()
	st.SetAccount(w.Address, &state.Account{
		Balance: 1_000_000 * core.OneQBC,
		Nonce:   0,
	})

	cfg := core.DefaultGenesisConfig(w.Address)
	cfg.Allocations = map[[crypto.AddressSize]byte]uint64{
		w.Address: 1_000_000 * core.OneQBC,
	}
	genesis := cfg.Build()

	vs, err := NewValidatorSet([]*Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}

	pool := mempool.New(100)

	// Add a valid transaction with gasPrice >= baseFee (nonce=0, amount=1000, gasPrice=5000)
	recipient := [crypto.AddressSize]byte{9, 9, 9}
	txValid := core.NewTransfer(w.Address, recipient, w.PublicKey, 0, 1000, 5000)
	txValid.GasLimit = core.BlockGasLimit // Triggers totalGas >= core.BlockGasLimit -> break
	if err := txValid.Sign(w.PrivateKey); err != nil {
		t.Fatalf("sign txValid: %v", err)
	}
	if err := pool.Add(txValid); err != nil {
		t.Fatalf("pool.Add txValid: %v", err)
	}

	// Add an invalid transaction to exercise error continue in buildBlock
	sender2, _ := crypto.NewWallet()
	txInvalid := core.NewTransfer(sender2.Address, recipient, sender2.PublicKey, 1000, 999, 5000)
	_ = txInvalid.Sign(sender2.PrivateKey)
	_ = pool.Add(txInvalid)

	upgSched := upgrade.NewScheduler(1, nil)
	upgMgr := upgrade.NewManager(upgrade.Config{CheckInterval: time.Hour}, upgSched)

	eng := NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, upgMgr)

	// 1. Successful block production
	ctx := context.Background()
	if err := eng.produceBlock(ctx); err != nil {
		t.Fatalf("produceBlock: %v", err)
	}

	if eng.Height() != 2 {
		t.Errorf("expected height 2, got %d", eng.Height())
	}

	// 2. Produce block when not proposer
	otherWallet, _ := crypto.NewWallet()
	nonProposerEng := NewEngine(otherWallet.Address, otherWallet.PublicKey, otherWallet.PrivateKey, vs, st, pool, genesis, nil, nil)
	if err := nonProposerEng.produceBlock(ctx); err != nil {
		t.Errorf("produceBlock when not proposer should return nil, got %v", err)
	}

	// 3. Error in signing header (corrupt validator private key)
	badKeyEng := NewEngine(w.Address, w.PublicKey, []byte("bad-priv-key"), vs, st, pool, genesis, nil, nil)
	if err := badKeyEng.produceBlock(ctx); err == nil {
		t.Error("expected error when private key is corrupt, got nil")
	}
}

func TestVote_Sign_Verify_ErrorBranches(t *testing.T) {
	v := &Vote{
		Type:      VotePrevote,
		Height:    1,
		Round:     0,
		BlockHash: [crypto.HashSize]byte{1},
		Voter:     [crypto.AddressSize]byte{2},
		PublicKey: []byte("bad-pub-key"),
	}

	// Sign with bad private key
	if err := v.Sign([]byte("bad-priv-key")); err == nil {
		t.Error("expected error signing with bad key, got nil")
	}

	// Verify with invalid public key size
	v.Signature = make([]byte, crypto.SignatureSize)
	if err := v.Verify(); err == nil {
		t.Error("expected error verifying with bad public key, got nil")
	}

	// Verify with invalid signature size
	v.PublicKey = make([]byte, crypto.PublicKeySize)
	v.Signature = []byte("short")
	if err := v.Verify(); err == nil {
		t.Error("expected error with short signature, got nil")
	}

	// Verify with voter address mismatch
	v.Signature = make([]byte, crypto.SignatureSize)
	if err := v.Verify(); err == nil {
		t.Error("expected error with voter address mismatch, got nil")
	}

	// Verify with wrong vote signature (ok == false)
	w, _ := crypto.NewWallet()
	vCorrect := &Vote{
		Type:      VotePrevote,
		Height:    1,
		Round:     0,
		BlockHash: [crypto.HashSize]byte{1},
		Voter:     w.Address,
		PublicKey: w.PublicKey,
	}
	_ = vCorrect.Sign(w.PrivateKey)
	vTampered := *vCorrect
	vTampered.Height = 999 // height changed so signature doesn't match
	if err := vTampered.Verify(); err == nil {
		t.Error("expected error for wrong signature on vote, got nil")
	}
}

func TestEngine_CommitChannel_Full(t *testing.T) {
	w, _ := crypto.NewWallet()
	genesis, st := newTestGenesis(w.Address)
	vs, _ := NewValidatorSet([]*Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	pool := mempool.New(100)
	eng := NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	// Fill commitCh buffer (capacity 64)
	for i := 0; i < 64; i++ {
		eng.commitCh <- genesis
	}

	// Commit should not block when commitCh is full (hits default: case)
	if err := eng.commit(genesis, st.Snapshot()); err != nil {
		t.Fatalf("commit failed when channel full: %v", err)
	}
}

func TestEngine_Run_TickerFires(t *testing.T) {
	w, _ := crypto.NewWallet()
	genesis, st := newTestGenesis(w.Address)
	vs, _ := NewValidatorSet([]*Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	pool := mempool.New(100)
	eng := NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	// Run for 2.2s so ticker (2s) fires at least once
	ctx, cancel := context.WithTimeout(context.Background(), 2200*time.Millisecond)
	defer cancel()

	eng.Run(ctx)
}

