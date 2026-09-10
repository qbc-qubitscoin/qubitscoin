package state

import (
	"sync"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

func randomAddr(t *testing.T) [crypto.AddressSize]byte {
	t.Helper()
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	return w.Address
}

// ── Snapshot & Apply ──────────────────────────────────────────────────────────

func TestSnapshot_IndependentFromOriginal(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)

	db.Credit(addr, 1000)
	snap := db.Snapshot()

	// Modify original after snapshot
	db.Credit(addr, 500)

	// Snapshot should still reflect original state
	if snap.GetBalance(addr) != 1000 {
		t.Errorf("Snapshot balance: want 1000, got %d", snap.GetBalance(addr))
	}
}

func TestApply_ReplacesAllState(t *testing.T) {
	db1 := NewStateDB()
	db2 := NewStateDB()

	addr1 := randomAddr(t)
	addr2 := randomAddr(t)

	db1.Credit(addr1, 100)
	db2.Credit(addr2, 200)

	// Apply db2 onto db1
	db1.Apply(db2)

	// db1 should now have addr2's balance
	if db1.GetBalance(addr2) != 200 {
		t.Errorf("after Apply, addr2 balance: want 200, got %d", db1.GetBalance(addr2))
	}
	// addr1 should be gone (full replacement)
	if db1.GetBalance(addr1) != 0 {
		t.Errorf("after Apply, addr1 balance should be 0, got %d", db1.GetBalance(addr1))
	}
}

func TestApply_SnapshotRoundTrip(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 777)

	snap := db.Snapshot()

	// Corrupt the original
	db.Credit(addr, 9999)

	// Restore from snapshot
	db.Apply(snap)

	if db.GetBalance(addr) != 777 {
		t.Errorf("after Apply snapshot, balance: want 777, got %d", db.GetBalance(addr))
	}
}

// ── CommitRoot determinism ────────────────────────────────────────────────────

func TestCommitRoot_Deterministic(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 12345)

	r1 := db.CommitRoot()
	r2 := db.CommitRoot()
	if r1 != r2 {
		t.Error("CommitRoot is not deterministic")
	}
}

func TestCommitRoot_ChangesAfterUpdate(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 100)
	r1 := db.CommitRoot()

	db.Credit(addr, 1)
	r2 := db.CommitRoot()

	if r1 == r2 {
		t.Error("CommitRoot should change when state is updated")
	}
}

func TestCommitRoot_EmptyState(t *testing.T) {
	db1 := NewStateDB()
	db2 := NewStateDB()
	if db1.CommitRoot() != db2.CommitRoot() {
		t.Error("two empty DBs should have identical commit roots")
	}
}

func TestCommitRoot_InsertOrderIndependent(t *testing.T) {
	// Two DBs with same accounts but inserted in different order
	addr1 := randomAddr(t)
	addr2 := randomAddr(t)

	db1 := NewStateDB()
	db1.Credit(addr1, 100)
	db1.Credit(addr2, 200)

	db2 := NewStateDB()
	db2.Credit(addr2, 200)
	db2.Credit(addr1, 100)

	if db1.CommitRoot() != db2.CommitRoot() {
		t.Error("CommitRoot should be independent of insertion order")
	}
}

// ── GetAccount / SetAccount ───────────────────────────────────────────────────

func TestGetAccount_UnknownAddrReturnsZero(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	acc := db.GetAccount(addr)
	if acc.Balance != 0 || acc.Nonce != 0 {
		t.Errorf("unknown account should be zero, got balance=%d nonce=%d", acc.Balance, acc.Nonce)
	}
}

func TestSetAccount_PersistsAndRetrieves(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)

	acc := &Account{Balance: 500, Nonce: 3}
	db.SetAccount(addr, acc)

	retrieved := db.GetAccount(addr)
	if retrieved.Balance != 500 {
		t.Errorf("Balance: want 500, got %d", retrieved.Balance)
	}
	if retrieved.Nonce != 3 {
		t.Errorf("Nonce: want 3, got %d", retrieved.Nonce)
	}
}

func TestGetAccount_ReturnsCopy(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 1000)

	acc := db.GetAccount(addr)
	acc.Balance = 0 // Mutate returned copy

	// Original in DB should be unchanged
	if db.GetBalance(addr) != 1000 {
		t.Error("GetAccount returned a reference, not a copy — mutation affected stored state")
	}
}

// ── Nonce tracking ────────────────────────────────────────────────────────────

func TestGetNonce_DefaultZero(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	if db.GetNonce(addr) != 0 {
		t.Error("nonce of unknown account should be 0")
	}
}

func TestGetNonce_AfterSet(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.SetAccount(addr, &Account{Nonce: 42, Balance: 0})
	if db.GetNonce(addr) != 42 {
		t.Errorf("Nonce: want 42, got %d", db.GetNonce(addr))
	}
}

// ── Len ───────────────────────────────────────────────────────────────────────

func TestLen_TracksAccounts(t *testing.T) {
	db := NewStateDB()
	if db.Len() != 0 {
		t.Errorf("empty DB should have Len 0, got %d", db.Len())
	}

	addr1 := randomAddr(t)
	addr2 := randomAddr(t)
	db.Credit(addr1, 1)
	db.Credit(addr2, 1)

	if db.Len() != 2 {
		t.Errorf("after 2 unique credits, Len should be 2, got %d", db.Len())
	}
}

// ── ForEach ───────────────────────────────────────────────────────────────────

func TestForEach_VisitsAllAccounts(t *testing.T) {
	db := NewStateDB()
	addrs := make([][crypto.AddressSize]byte, 5)
	for i := range addrs {
		addrs[i] = randomAddr(t)
		db.Credit(addrs[i], uint64(i+1)*100)
	}

	visited := 0
	db.ForEach(func(_ [crypto.AddressSize]byte, _ *Account) {
		visited++
	})

	if visited != 5 {
		t.Errorf("ForEach visited %d accounts, want 5", visited)
	}
}

// ── Concurrent access (race detector) ────────────────────────────────────────

func TestConcurrentCredit(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)

	var wg sync.WaitGroup
	const goroutines = 50
	const creditPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Credit(addr, creditPerGoroutine)
		}()
	}
	wg.Wait()

	expected := uint64(goroutines * creditPerGoroutine)
	if db.GetBalance(addr) != expected {
		t.Errorf("concurrent Credit: want %d, got %d", expected, db.GetBalance(addr))
	}
}

func TestConcurrentGetSet(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 1000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = db.GetBalance(addr)
		}()
		go func() {
			defer wg.Done()
			db.Credit(addr, 1)
		}()
	}
	wg.Wait()
}

func TestConcurrentSnapshotAndApply(t *testing.T) {
	db := NewStateDB()
	addr := randomAddr(t)
	db.Credit(addr, 500)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap := db.Snapshot()
			_ = snap.CommitRoot()
		}()
	}
	wg.Wait()
}
