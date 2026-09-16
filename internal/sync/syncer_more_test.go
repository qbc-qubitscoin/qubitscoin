package sync

import (
	"context"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/p2p"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
)

func TestSyncer_Run_ContextCancel(t *testing.T) {
	st := state.NewStateDB()
	
	w, _ := crypto.NewWallet()
	val := &consensus.Validator{
		Address:     w.Address,
		PublicKey:   w.PublicKey,
		VotingPower: 1,
	}
	vs, _ := consensus.NewValidatorSet([]*consensus.Validator{val})
	genesis := &core.Block{Header: core.BlockHeader{Height: 0}}
	
	engine := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, nil, genesis, nil, nil)
	
	ident := p2p.NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	node, _ := p2p.NewNode(ident)
	
	s := New(engine, st, nil, node, nil, nil)
	
	// Speed up the tick for testing
	oldInterval := SyncInterval
	SyncInterval = 1 * time.Millisecond
	defer func() { SyncInterval = oldInterval }()
	
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond) // Let it tick a few times
	cancel()
	
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit on context cancel")
	}
}

func TestSyncer_trySync(t *testing.T) {
	// Set up a mock engine and node
	st := state.NewStateDB()
	
	// Create validator key
	w, _ := crypto.NewWallet()
	val := &consensus.Validator{
		Address:     w.Address,
		PublicKey:   w.PublicKey,
		VotingPower: 1,
	}
	vs, _ := consensus.NewValidatorSet([]*consensus.Validator{val})
	genesis := &core.Block{Header: core.BlockHeader{Height: 0}}
	
	engine := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, nil, genesis, nil, nil)
	
	// mock node needs a network
	ident := p2p.NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	node, _ := p2p.NewNode(ident)
	
	s := New(engine, st, nil, node, nil, nil)
	
	// test already up to date
	s.trySync() // best <= local (0 <= 0)
	
	// test behind
	s.ObservePeerHeight("peer", 100)
	
	// trySync should encode a GetBlocks message
	// we just ensure it doesn't panic
	s.trySync()
}

func TestSyncer_ApplyBlocks(t *testing.T) {
	st := state.NewStateDB()
	
	// Create validator key
	w, _ := crypto.NewWallet()
	val := &consensus.Validator{
		Address:     w.Address,
		PublicKey:   w.PublicKey,
		VotingPower: 1,
	}
	vs, _ := consensus.NewValidatorSet([]*consensus.Validator{val})
	genesis := &core.Block{Header: core.BlockHeader{Height: 0}, Hash: [32]byte{1}}
	engine := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, nil, genesis, nil, nil)
	
	s := New(engine, st, nil, nil, nil, w.PublicKey)
	
	// 1. Empty slice
	s.ApplyBlocks([]*core.Block{})
	
	// 2. Wrong height
	blkWrongHeight := &core.Block{
		Header: core.BlockHeader{Height: 99},
	}
	s.ApplyBlocks([]*core.Block{blkWrongHeight})
	
	// 3. Bad signature
	blkBadSig := &core.Block{
		Header: core.BlockHeader{Height: 0},
	}
	// Sign with a different key
	badW, _ := crypto.NewWallet()
	blkBadSig.SignHeader(badW.PrivateKey)
	
	s.ApplyBlocks([]*core.Block{blkBadSig})
	
	// 4. Missing parent (height 1, but prevHash mismatch or parent missing if height is ahead)
	blkMissingParent := &core.Block{
		Header: core.BlockHeader{Height: 1},
	}
	blkMissingParent.SignHeader(w.PrivateKey)
	
	s.ApplyBlocks([]*core.Block{blkMissingParent})
	
	// To fully cover, we need a block with valid signature and prevHash mismatch,
	// and a block with valid signature and prevHash match (to hit ApplyBlock).
	// Since height 0 is genesis, wait, localHeight starts at 0 or 1?
	// engine.Height() returns the tip height.
	// If it returns 0, then localHeight-1 would underflow!
	// Let's check engine.Height().
}
