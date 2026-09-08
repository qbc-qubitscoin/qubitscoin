package mempool

import (
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// Wallets generated once to avoid paying key-gen cost per test.
var (
	walletA *crypto.Wallet
	walletB *crypto.Wallet
)

func init() {
	var err error
	walletA, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
	walletB, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
}

// signedTx creates, signs, and returns a transfer transaction.
func signedTx(t *testing.T, w *crypto.Wallet, to [crypto.AddressSize]byte, nonce, gasPrice uint64) *core.Transaction {
	t.Helper()
	tx := &core.Transaction{
		Version:   1,
		Type:      core.TxTransfer,
		Nonce:     nonce,
		From:      w.Address,
		To:        to,
		Amount:    1,
		GasLimit:  core.GasTransfer,
		GasPrice:  gasPrice,
		Timestamp: time.Now().UnixNano(),
		PublicKey: w.PublicKey,
	}
	if err := tx.Sign(w.PrivateKey); err != nil {
		t.Fatalf("sign tx: %v", err)
	}
	return tx
}

func TestMempool_Add_Valid(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	if err := mp.Add(tx); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if mp.Size() != 1 {
		t.Errorf("Size: want 1, got %d", mp.Size())
	}
}

func TestMempool_Add_Duplicate(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	_ = mp.Add(tx)
	if err := mp.Add(tx); err == nil {
		t.Fatal("expected error for duplicate transaction")
	}
}

func TestMempool_Add_InvalidTx_BadVersion(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	tx.Version = 99
	if err := mp.Add(tx); err == nil {
		t.Fatal("expected error for an invalid tx version")
	}
}

func TestMempool_Pending_HighestGasPriceFirst(t *testing.T) {
	mp := New(0)
	// Add 3 txs with different gas prices from different nonce.
	prices := []uint64{core.MinGasPrice, core.MinGasPrice * 3, core.MinGasPrice * 2}
	for i, p := range prices {
		tx := signedTx(t, walletA, walletB.Address, uint64(i), p)
		if err := mp.Add(tx); err != nil {
			t.Fatalf("Add tx %d: %v", i, err)
		}
	}
	pending := mp.Pending(10)
	if len(pending) != 3 {
		t.Fatalf("Pending: want 3, got %d", len(pending))
	}
	// Should be in descending gas price order.
	for i := 1; i < len(pending); i++ {
		if pending[i].GasPrice > pending[i-1].GasPrice {
			t.Errorf("pending[%d].GasPrice=%d > pending[%d].GasPrice=%d: not sorted",
				i, pending[i].GasPrice, i-1, pending[i-1].GasPrice)
		}
	}
}

func TestMempool_Pending_LimitN(t *testing.T) {
	mp := New(0)
	for i := 0; i < 5; i++ {
		tx := signedTx(t, walletA, walletB.Address, uint64(i), core.MinGasPrice)
		_ = mp.Add(tx)
	}
	pending := mp.Pending(3)
	if len(pending) != 3 {
		t.Errorf("Pending(3): want 3, got %d", len(pending))
	}
}

func TestMempool_Pending_DoesNotRemove(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	_ = mp.Add(tx)
	_ = mp.Pending(10)
	if mp.Size() != 1 {
		t.Error("Pending should not remove transactions")
	}
}

func TestMempool_Get(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	_ = mp.Add(tx)
	hashHex := crypto.ToHex(tx.Hash)
	got, ok := mp.Get(hashHex)
	if !ok {
		t.Fatal("Get: tx not found")
	}
	if got.Hash != tx.Hash {
		t.Fatal("Get: wrong transaction returned")
	}
}

func TestMempool_Remove(t *testing.T) {
	mp := New(0)
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	_ = mp.Add(tx)
	mp.Remove(crypto.ToHex(tx.Hash))
	if mp.Size() != 0 {
		t.Errorf("after Remove: want size 0, got %d", mp.Size())
	}
}

func TestMempool_PurgeCommitted(t *testing.T) {
	mp := New(0)
	tx1 := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	tx2 := signedTx(t, walletA, walletB.Address, 1, core.MinGasPrice)
	_ = mp.Add(tx1)
	_ = mp.Add(tx2)

	mp.PurgeCommitted([]*core.Transaction{tx1})
	if mp.Size() != 1 {
		t.Errorf("after PurgeCommitted: want size 1, got %d", mp.Size())
	}
	if _, ok := mp.Get(crypto.ToHex(tx1.Hash)); ok {
		t.Fatal("purged tx should no longer be in the pool")
	}
}

func TestMempool_Full(t *testing.T) {
	mp := New(2)
	tx1 := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	tx2 := signedTx(t, walletA, walletB.Address, 1, core.MinGasPrice)
	tx3 := signedTx(t, walletA, walletB.Address, 2, core.MinGasPrice)

	_ = mp.Add(tx1)
	_ = mp.Add(tx2)
	if err := mp.Add(tx3); err == nil {
		t.Fatal("expected error when mempool is full")
	}
}

func TestMempool_Size_AfterOperations(t *testing.T) {
	mp := New(0)
	if mp.Size() != 0 {
		t.Fatalf("initial size: want 0, got %d", mp.Size())
	}
	tx := signedTx(t, walletA, walletB.Address, 0, core.MinGasPrice)
	_ = mp.Add(tx)
	if mp.Size() != 1 {
		t.Fatalf("after Add: want 1, got %d", mp.Size())
	}
	mp.Remove(crypto.ToHex(tx.Hash))
	if mp.Size() != 0 {
		t.Fatalf("after Remove: want 0, got %d", mp.Size())
	}
}
