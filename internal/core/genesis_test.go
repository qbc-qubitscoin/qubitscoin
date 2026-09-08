package core

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestDefaultGenesisConfig_Allocations(t *testing.T) {
	w, _ := crypto.NewWallet()
	cfg := DefaultGenesisConfig(w.Address)

	if cfg.ChainID != ChainID {
		t.Errorf("ChainID: want %d, got %d", ChainID, cfg.ChainID)
	}
	bal, ok := cfg.Allocations[w.Address]
	if !ok {
		t.Fatal("validator address not in allocations")
	}
	if bal == 0 {
		t.Fatal("validator allocation must be > 0")
	}
}

func TestGenesisConfig_Build_Height(t *testing.T) {
	w, _ := crypto.NewWallet()
	cfg := DefaultGenesisConfig(w.Address)
	blk := cfg.Build()
	if blk.Header.Height != 0 {
		t.Errorf("genesis height: want 0, got %d", blk.Header.Height)
	}
}

func TestGenesisConfig_Build_NilTxs(t *testing.T) {
	w, _ := crypto.NewWallet()
	blk := DefaultGenesisConfig(w.Address).Build()
	if blk.Txs != nil {
		t.Error("genesis block should have a nil tx list")
	}
}

func TestGenesisConfig_Build_HashNotZero(t *testing.T) {
	w, _ := crypto.NewWallet()
	blk := DefaultGenesisConfig(w.Address).Build()
	var zero [crypto.HashSize]byte
	if blk.Hash == zero {
		t.Fatal("genesis block hash should not be zero")
	}
}

func TestGenesisConfig_Build_StateRootDeterministic(t *testing.T) {
	w, _ := crypto.NewWallet()
	cfg := DefaultGenesisConfig(w.Address)
	b1 := cfg.Build()
	b2 := cfg.Build()
	if b1.Header.StateRoot != b2.Header.StateRoot {
		t.Fatal("genesis state root is not deterministic")
	}
}

func TestGenesisConfig_Build_DifferentValidators_DifferentRoots(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	r1 := DefaultGenesisConfig(w1.Address).Build().Header.StateRoot
	r2 := DefaultGenesisConfig(w2.Address).Build().Header.StateRoot
	if r1 == r2 {
		t.Fatal("different validators should produce different state roots")
	}
}

func TestGenesisConfig_Build_GasLimit(t *testing.T) {
	w, _ := crypto.NewWallet()
	blk := DefaultGenesisConfig(w.Address).Build()
	if blk.Header.GasLimit != BlockGasLimit {
		t.Errorf("gas limit: want %d, got %d", BlockGasLimit, blk.Header.GasLimit)
	}
}

func TestGenesisConfig_Build_PrevHashIsZero(t *testing.T) {
	w, _ := crypto.NewWallet()
	blk := DefaultGenesisConfig(w.Address).Build()
	var zero [crypto.HashSize]byte
	if blk.Header.PrevHash != zero {
		t.Fatal("genesis prevHash should be all zeros")
	}
}
