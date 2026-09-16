package mempool

import (
	"sync"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

var (
	concWallet1 *crypto.Wallet
	concWallet2 *crypto.Wallet
)

func init() {
	var err error
	concWallet1, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
	concWallet2, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
}

func makeSignedTx(t *testing.T, from *crypto.Wallet, to [crypto.AddressSize]byte, nonce, amount, gasPrice uint64) *core.Transaction {
	t.Helper()
	tx := &core.Transaction{
		Version:   1,
		Type:      core.TxTransfer,
		Nonce:     nonce,
		From:      from.Address,
		To:        to,
		Amount:    amount,
		GasLimit:  core.GasTransfer,
		GasPrice:  gasPrice,
		Timestamp: time.Now().UnixNano(),
		PublicKey: from.PublicKey,
	}
	if err := tx.Sign(from.PrivateKey); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return tx
}

// ── Priority ordering ─────────────────────────────────────────────────────────

// TestPending_OrderByGasPrice verifies that Pending() returns txs sorted by highest gas price first.
func TestPending_OrderByGasPrice(t *testing.T) {
	mp := New(100)

	// Add txs with different gas prices from different nonces
	lowTx := makeSignedTx(t, concWallet1, concWallet2.Address, 10, 1, core.MinGasPrice)
	highTx := makeSignedTx(t, concWallet1, concWallet2.Address, 11, 1, core.MinGasPrice*10)

	if err := mp.Add(lowTx); err != nil {
		t.Fatalf("Add lowTx: %v", err)
	}
	if err := mp.Add(highTx); err != nil {
		t.Fatalf("Add highTx: %v", err)
	}

	pending := mp.Pending(10)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending txs, got %d", len(pending))
	}
	if pending[0].GasPrice < pending[1].GasPrice {
		t.Errorf("Pending() not sorted by gas price desc: first=%d, second=%d",
			pending[0].GasPrice, pending[1].GasPrice)
	}
}

// TestPending_LimitRespected verifies Pending(n) returns at most n txs.
func TestPending_LimitRespected(t *testing.T) {
	mp := New(100)
	for i := uint64(0); i < 5; i++ {
		tx := makeSignedTx(t, concWallet1, concWallet2.Address, 100+i, 1, core.MinGasPrice)
		if err := mp.Add(tx); err != nil {
			t.Fatalf("Add tx %d: %v", i, err)
		}
	}

	pending := mp.Pending(3)
	if len(pending) != 3 {
		t.Errorf("Pending(3): want 3 txs, got %d", len(pending))
	}
}

// TestPending_DoesNotMutatePool verifies that calling Pending() doesn't remove txs.
func TestPending_DoesNotMutatePool(t *testing.T) {
	mp := New(100)
	tx := makeSignedTx(t, concWallet1, concWallet2.Address, 200, 1, core.MinGasPrice)
	if err := mp.Add(tx); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_ = mp.Pending(10)
	if mp.Size() != 1 {
		t.Errorf("Pending() should not remove txs: size want 1, got %d", mp.Size())
	}
}

// ── Concurrent access (race detector) ────────────────────────────────────────

// TestConcurrentAdd verifies that concurrent Add calls are safe under -race.
func TestConcurrentAdd(t *testing.T) {
	mp := New(10_000)
	var wg sync.WaitGroup

	// Use wallets to avoid same-hash collisions; each goroutine uses unique nonces
	wallets := make([]*crypto.Wallet, 4)
	for i := range wallets {
		w, err := crypto.NewWallet()
		if err != nil {
			t.Fatalf("NewWallet: %v", err)
		}
		wallets[i] = w
	}

	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(walletIdx int) {
			defer wg.Done()
			w := wallets[walletIdx]
			for nonce := uint64(0); nonce < 10; nonce++ {
				tx := makeSignedTx(t, w, concWallet2.Address, nonce, 1, core.MinGasPrice)
				_ = mp.Add(tx) // errors are acceptable (pool full, etc.)
			}
		}(g)
	}
	wg.Wait()

	// Pool should have some transactions
	if mp.Size() == 0 {
		t.Error("after concurrent adds, pool should not be empty")
	}
}

// TestConcurrentPurgeCommitted verifies thread-safety of PurgeCommitted.
func TestConcurrentPurgeCommitted(t *testing.T) {
	mp := New(10_000)
	committed := make([]*core.Transaction, 0, 20)

	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	// Pre-fill pool
	for nonce := uint64(0); nonce < 20; nonce++ {
		tx := makeSignedTx(t, w, w2.Address, nonce, 1, core.MinGasPrice)
		if err := mp.Add(tx); err == nil {
			committed = append(committed, tx)
		}
	}

	var wg sync.WaitGroup

	// Purge from one goroutine, Size() from another
	wg.Add(2)
	go func() {
		defer wg.Done()
		mp.PurgeCommitted(committed)
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = mp.Size()
		}
	}()
	wg.Wait()
}

// TestConcurrentGetAndRemove verifies thread-safety of Get and Remove.
func TestConcurrentGetAndRemove(t *testing.T) {
	mp := New(1000)
	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	tx := makeSignedTx(t, w, w2.Address, 999, 1, core.MinGasPrice)
	if err := mp.Add(tx); err != nil {
		t.Fatalf("Add: %v", err)
	}
	hashHex := crypto.ToHex(tx.Hash)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = mp.Get(hashHex)
			mp.Remove(hashHex)
		}()
	}
	wg.Wait()
}

// ── Sender queue limit ────────────────────────────────────────────────────────

func TestAdd_SenderQueueFull(t *testing.T) {
	mp := New(10_000)
	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	// Fill sender queue to MaxSenderQueueLen
	var lastErr error
	for nonce := uint64(0); nonce <= uint64(MaxSenderQueueLen); nonce++ {
		tx := makeSignedTx(t, w, w2.Address, nonce, 1, core.MinGasPrice)
		lastErr = mp.Add(tx)
	}
	if lastErr == nil {
		t.Errorf("expected 'sender queue full' error after %d txs from one sender", MaxSenderQueueLen+1)
	}
}

// ── Len vs Size consistency ───────────────────────────────────────────────────

func TestLen_EqualsSize(t *testing.T) {
	mp := New(100)
	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	tx := makeSignedTx(t, w, w2.Address, 0, 1, core.MinGasPrice)
	_ = mp.Add(tx)

	if mp.Len() != mp.Size() {
		t.Errorf("Len()=%d != Size()=%d", mp.Len(), mp.Size())
	}
}

// ── Remove non-existent ───────────────────────────────────────────────────────

func TestRemove_NonExistentIsNoOp(t *testing.T) {
	mp := New(100)
	// Should not panic
	mp.Remove("nonexistent_hash")
	if mp.Size() != 0 {
		t.Error("pool should be empty after removing non-existent tx")
	}
}

// ── Get tests ─────────────────────────────────────────────────────────────────

func TestGet_FindsAddedTx(t *testing.T) {
	mp := New(100)
	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	tx := makeSignedTx(t, w, w2.Address, 0, 42, core.MinGasPrice)
	_ = mp.Add(tx)

	found, ok := mp.Get(crypto.ToHex(tx.Hash))
	if !ok {
		t.Fatal("Get: expected to find tx")
	}
	if found.Amount != tx.Amount {
		t.Errorf("Amount: want %d, got %d", tx.Amount, found.Amount)
	}
}

func TestGet_MissingReturnsFalse(t *testing.T) {
	mp := New(100)
	_, ok := mp.Get("missing_hash_hex")
	if ok {
		t.Error("Get on missing hash should return false")
	}
}

// ── New() with zero/negative maxSize uses default ─────────────────────────────

func TestNew_DefaultMaxSize(t *testing.T) {
	mp := New(0)
	if mp.maxSize != DefaultMaxSize {
		t.Errorf("expected default max size %d, got %d", DefaultMaxSize, mp.maxSize)
	}
}

func TestNew_NegativeMaxSize(t *testing.T) {
	mp := New(-1)
	if mp.maxSize != DefaultMaxSize {
		t.Errorf("expected default max size %d, got %d", DefaultMaxSize, mp.maxSize)
	}
}
