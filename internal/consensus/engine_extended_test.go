package consensus_test

import (
	"context"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
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

func TestEngine_Basics(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}

	genesis, st := newTestGenesis(w.Address)
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}

	pool := mempool.New(100)
	eng := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	if eng.Height() != 1 {
		t.Errorf("Height: want 1, got %d", eng.Height())
	}
	if b := eng.BlockByHeight(0); b == nil || b.Hash != genesis.Hash {
		t.Errorf("BlockByHeight(0) did not return genesis block")
	}
	if b := eng.BlockByHeight(999); b != nil {
		t.Errorf("BlockByHeight(999): want nil, got %v", b)
	}
	if eng.CommitCh() == nil {
		t.Error("CommitCh should not be nil")
	}
}

func TestEngine_InjectBlock(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	genesis, st := newTestGenesis(w.Address)
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}

	pool := mempool.New(100)
	eng := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	injected, err := core.NewBlock(3, genesis.Hash, genesis.Header.StateRoot, time.Now().UnixNano(), w.Address, nil, 0, 1000, 0)
	if err != nil {
		t.Fatalf("NewBlock: %v", err)
	}
	_ = injected.SignHeader(w.PrivateKey)

	eng.InjectBlock(injected)

	if eng.Height() != 4 {
		t.Errorf("Height: want 4 after inject, got %d", eng.Height())
	}
	if eng.BlockByHeight(1) != nil || eng.BlockByHeight(2) != nil {
		t.Errorf("gaps should be nil")
	}
	if b := eng.BlockByHeight(3); b == nil || b.Hash != injected.Hash {
		t.Errorf("BlockByHeight(3) mismatch")
	}
}

func TestEngine_AcceptVote(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	genesis, st := newTestGenesis(w.Address)
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}

	pool := mempool.New(100)
	eng := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	// 1. Valid vote
	v := &consensus.Vote{
		Type:      consensus.VotePrevote,
		Height:    1,
		Round:     0,
		BlockHash: genesis.Hash,
		Voter:     w.Address,
		PublicKey: w.PublicKey,
	}
	if err := v.Sign(w.PrivateKey); err != nil {
		t.Fatalf("v.Sign: %v", err)
	}
	if err := eng.AcceptVote(v); err != nil {
		t.Errorf("AcceptVote: unexpected error for valid vote: %v", err)
	}

	// 2. Invalid vote signature
	tampered := *v
	tampered.Signature = make([]byte, crypto.SignatureSize)
	if err := eng.AcceptVote(&tampered); err == nil {
		t.Error("AcceptVote: expected error for invalid signature, got nil")
	}

	// 3. Unknown voter
	otherWallet, _ := crypto.NewWallet()
	vUnknown := &consensus.Vote{
		Type:      consensus.VotePrevote,
		Height:    1,
		Round:     0,
		BlockHash: genesis.Hash,
		Voter:     otherWallet.Address,
		PublicKey: otherWallet.PublicKey,
	}
	_ = vUnknown.Sign(otherWallet.PrivateKey)
	if err := eng.AcceptVote(vUnknown); err == nil {
		t.Error("AcceptVote: expected error for unknown voter, got nil")
	}
}

func TestEngine_ProduceBlock_And_Run(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	genesis, st := newTestGenesis(w.Address)
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{
		{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 100},
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}

	pool := mempool.New(100)
	eng := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)

	// Add a transaction to pool
	tx := core.NewTransfer(w.Address, [crypto.AddressSize]byte{1}, w.PublicKey, 100, 1, core.MinGasPrice)
	_ = tx.Sign(w.PrivateKey)
	_ = pool.Add(tx)

	// Run for a short period then cancel
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		eng.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("engine Run did not exit within timeout")
	}

	// Test non-proposer node does not produce block
	otherWallet, _ := crypto.NewWallet()
	nonProposerEng := consensus.NewEngine(otherWallet.Address, otherWallet.PublicKey, otherWallet.PrivateKey, vs, st, pool, genesis, nil, nil)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	nonProposerEng.Run(ctx2)
}
