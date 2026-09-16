package mempool

import (
	"testing"
	
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestMempool_PurgeCommitted_EmptyAndNotInPool(t *testing.T) {
	mp := New(10)
	
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	tx1 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 1, 10, core.MinGasPrice)
	tx1.Sign(w1.PrivateKey)
	
	err := mp.Add(tx1)
	if err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	
	// Empty list
	mp.PurgeCommitted(nil)
	if mp.Size() != 1 {
		t.Errorf("expected size 1, got %d", mp.Size())
	}
	
	// Tx not in pool
	tx2 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 2, 10, core.MinGasPrice)
	tx2.Sign(w1.PrivateKey)
	mp.PurgeCommitted([]*core.Transaction{tx2})
	if mp.Size() != 1 {
		t.Errorf("expected size 1, got %d", mp.Size())
	}
}

func TestMempool_Pending_Zero(t *testing.T) {
	mp := New(10)
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	tx1 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 1, 10, core.MinGasPrice)
	tx1.Sign(w1.PrivateKey)
	_ = mp.Add(tx1)
	
	res := mp.Pending(0)
	if len(res) != 0 {
		t.Errorf("expected 0 pending, got %d", len(res))
	}
}

func TestMempool_Add_MaxSizeEdgeCase(t *testing.T) {
	// What happens when maxSize is explicitly 0?
	// The constructor handles it by setting it to DefaultMaxSize, so we can't test maxSize=0 in Add directly unless we set it manually, which is unexported.
	// We'll test maxSize = 1
	mp := New(1)
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	tx1 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 1, 10, core.MinGasPrice)
	tx1.Sign(w1.PrivateKey)
	err := mp.Add(tx1)
	if err != nil {
		t.Fatalf("failed to add tx1: %v", err)
	}
	
	tx2 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 2, 10, core.MinGasPrice)
	tx2.Sign(w1.PrivateKey)
	err = mp.Add(tx2)
	if err == nil || err.Error() != "mempool full" {
		t.Errorf("expected 'mempool full', got %v", err)
	}
	
	// Test max tx data size
	tx3 := core.NewTransfer(w1.Address, w2.Address, w1.PublicKey, 3, 10, core.MinGasPrice)
	tx3.Data = make([]byte, MaxTxDataSize+1)
	tx3.Sign(w1.PrivateKey)
	err = mp.Add(tx3)
	if err == nil || err.Error() != "tx data exceeds maximum size" {
		t.Errorf("expected 'tx data exceeds maximum size', got %v", err)
	}
}
