package sync

import (
	"context"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/p2p"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/storage"
)

func TestNewSyncer(t *testing.T) {
	s := New(nil, nil, nil, nil, nil, nil)
	if s == nil {
		t.Errorf("New() returned nil")
	}
}

func TestObservePeerHeight(t *testing.T) {
	s := New(nil, nil, nil, nil, nil, nil)
	
	// Returns 0 initially
	if h := s.BestPeerHeight(); h != 0 {
		t.Errorf("Expected BestPeerHeight to be 0, got %d", h)
	}

	// Update to 10
	s.ObservePeerHeight("peer1", 10)
	if h := s.BestPeerHeight(); h != 10 {
		t.Errorf("Expected BestPeerHeight to be 10, got %d", h)
	}

	// Doesn't decrease
	s.ObservePeerHeight("peer1", 5)
	if h := s.BestPeerHeight(); h != 10 {
		t.Errorf("Expected BestPeerHeight to be 10, got %d", h)
	}

	// Update to 15
	s.ObservePeerHeight("peer2", 15)
	if h := s.BestPeerHeight(); h != 15 {
		t.Errorf("Expected BestPeerHeight to be 15, got %d", h)
	}
}

func TestApplyBlocks_Empty(t *testing.T) {
	s := New(nil, nil, nil, nil, nil, nil)
	s.ApplyBlocks([]*core.Block{})
}

func TestApplyBlocks_WrongHeight(t *testing.T) {
	eng := consensus.NewEngine([crypto.AddressSize]byte{}, nil, nil, nil, nil, nil, nil, nil, nil)
	s := New(eng, nil, nil, nil, nil, nil)
	
	blk := &core.Block{
		Header: core.BlockHeader{
			Height: 999,
		},
	}
	s.ApplyBlocks([]*core.Block{blk})
}

func TestApplyBlocks_BadSignature(t *testing.T) {
	eng := consensus.NewEngine([crypto.AddressSize]byte{}, nil, nil, nil, nil, nil, nil, nil, nil)
	w, _ := crypto.NewWallet()
	s := New(eng, nil, nil, nil, nil, w.PublicKey)

	blk := &core.Block{
		Header: core.BlockHeader{
			Height: eng.Height(), 
		},
	}
	s.ApplyBlocks([]*core.Block{blk})
}

func TestRun_Cancel(t *testing.T) {
	s := New(nil, nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	s.Run(ctx)
}

func TestTrySync_UpToDate(t *testing.T) {
	eng := consensus.NewEngine([crypto.AddressSize]byte{}, nil, nil, nil, nil, nil, nil, nil, nil)
	s := New(eng, nil, nil, nil, nil, nil)
	// Local height is 1, best peer is 0 -> up to date
	s.trySync() 
}

func TestTrySync_Behind(t *testing.T) {
	eng := consensus.NewEngine([crypto.AddressSize]byte{}, nil, nil, nil, nil, nil, nil, nil, nil)
	s := New(eng, nil, nil, &p2p.Node{}, nil, nil)
	s.ObservePeerHeight("peer1", 70) // Needs two batches (batch size 64)
	
	// Because node is &p2p.Node{}, BroadcastRaw won't panic and does nothing.
	s.trySync()
}

func TestGobEncode_Fail(t *testing.T) {
	// gob fails to encode channels
	_, err := gobEncode(make(chan int))
	if err == nil {
		t.Errorf("expected gobEncode to fail")
	}
}

func TestApplyBlocks_ValidAndError(t *testing.T) {
	w, _ := crypto.NewWallet()
	genesis := &core.Block{
		Header: core.BlockHeader{
			Height: 0,
		},
	}
	genesis.Hash = genesis.ComputeHash()

	eng := consensus.NewEngine([crypto.AddressSize]byte{}, nil, nil, nil, nil, nil, genesis, nil, nil)
	st := state.NewStateDB()
	
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer db.Close()
	store := storage.NewBlockStore(db)

	s := New(eng, st, store, nil, nil, w.PublicKey)

	// Block with prevHash mismatch
	blkMismatch := &core.Block{
		Header: core.BlockHeader{
			Height:   1,
			PrevHash: [32]byte{1, 2, 3}, // wrong prev hash
		},
	}
	blkMismatch.SignHeader(w.PrivateKey)
	s.ApplyBlocks([]*core.Block{blkMismatch})

	// Block that fails ApplyBlock (invalid tx)
	blkErr := &core.Block{
		Header: core.BlockHeader{
			Height:   1,
			PrevHash: genesis.Hash,
		},
		Txs: []*core.Transaction{
			{GasLimit: 0}, // Invalid tx will fail
		},
	}
	blkErr.SignHeader(w.PrivateKey)
	s.ApplyBlocks([]*core.Block{blkErr})

	// Valid block, empty txs
	blkValid := &core.Block{
		Header: core.BlockHeader{
			Height:   1,
			PrevHash: genesis.Hash,
		},
	}
	blkValid.SignHeader(w.PrivateKey)
	s.ApplyBlocks([]*core.Block{blkValid})

	// Close DB so next PutBlock fails
	db.Close()
	blkStoreErr := &core.Block{
		Header: core.BlockHeader{
			Height:   1, // eng.Height() is still 1
			PrevHash: genesis.Hash,
		},
	}
	blkStoreErr.SignHeader(w.PrivateKey)
	s.ApplyBlocks([]*core.Block{blkStoreErr})
}
