package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/storage"
)

func tempDB(t *testing.T) *storage.DB {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "testdb")
	db, err := storage.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ─────────────────────────────────────────────────────────────────────────────
// DB primitives
// ─────────────────────────────────────────────────────────────────────────────

func TestDB_PutGet(t *testing.T) {
	db := tempDB(t)
	if err := db.Put([]byte("hello"), []byte("world")); err != nil {
		t.Fatal(err)
	}
	val, err := db.Get([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "world" {
		t.Fatalf("want %q got %q", "world", val)
	}
}

func TestDB_GetMissing(t *testing.T) {
	db := tempDB(t)
	val, err := db.Get([]byte("missing"))
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Fatalf("want nil for missing key, got %v", val)
	}
}

func TestDB_Has(t *testing.T) {
	db := tempDB(t)
	_ = db.Put([]byte("k"), []byte("v"))
	ok, err := db.Has([]byte("k"))
	if err != nil || !ok {
		t.Fatalf("Has: err=%v ok=%v", err, ok)
	}
	ok, err = db.Has([]byte("missing"))
	if err != nil || ok {
		t.Fatalf("Has missing: err=%v ok=%v", err, ok)
	}
}

func TestDB_Delete(t *testing.T) {
	db := tempDB(t)
	_ = db.Put([]byte("k"), []byte("v"))
	_ = db.Delete([]byte("k"))
	val, _ := db.Get([]byte("k"))
	if val != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDB_IterPrefix(t *testing.T) {
	db := tempDB(t)
	_ = db.Put([]byte("p:1"), []byte("a"))
	_ = db.Put([]byte("p:2"), []byte("b"))
	_ = db.Put([]byte("q:3"), []byte("c")) // different prefix

	var keys []string
	err := db.IterPrefix([]byte("p:"), func(k, _ []byte) error {
		keys = append(keys, string(k))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("want 2 keys, got %d: %v", len(keys), keys)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BlockStore
// ─────────────────────────────────────────────────────────────────────────────

func makeBlock(t *testing.T, height uint64, prev [crypto.HashSize]byte) *core.Block {
	t.Helper()
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	blk, err := core.NewBlock(height, prev, crypto.ZeroHash,
		time.Now().UnixNano(), w.Address, nil, 0, core.InitialBaseFee, 0)
	if err != nil {
		t.Fatalf("NewBlock h=%d: %v", height, err)
	}
	if err := blk.SignHeader(w.PrivateKey); err != nil {
		t.Fatalf("SignHeader: %v", err)
	}
	return blk
}

func TestBlockStore_PutAndGet(t *testing.T) {
	db := tempDB(t)
	bs := storage.NewBlockStore(db)

	blk := makeBlock(t, 1, crypto.ZeroHash)
	if err := bs.PutBlock(blk); err != nil {
		t.Fatal(err)
	}

	got, err := bs.GetBlock(blk.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != blk.Hash {
		t.Fatalf("hash mismatch: want %s got %s", crypto.ToHex(blk.Hash), crypto.ToHex(got.Hash))
	}
}

func TestBlockStore_GetByHeight(t *testing.T) {
	db := tempDB(t)
	bs := storage.NewBlockStore(db)

	blk := makeBlock(t, 42, crypto.ZeroHash)
	_ = bs.PutBlock(blk)

	got, err := bs.GetBlockByHeight(42)
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.Height != 42 {
		t.Fatalf("wrong height: %d", got.Header.Height)
	}
}

func TestBlockStore_GetMissing(t *testing.T) {
	db := tempDB(t)
	bs := storage.NewBlockStore(db)
	_, err := bs.GetBlockByHeight(999)
	if err != storage.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBlockStore_Tip(t *testing.T) {
	db := tempDB(t)
	bs := storage.NewBlockStore(db)

	// Empty DB — tip returns (0, zeroHash, nil).
	h, hash, err := bs.GetTip()
	if err != nil {
		t.Fatal(err)
	}
	if h != 0 || hash != (crypto.ZeroHash) {
		t.Fatalf("unexpected initial tip: h=%d hash=%v", h, hash)
	}

	blk := makeBlock(t, 5, crypto.ZeroHash)
	_ = bs.PutBlock(blk)
	_ = bs.UpdateTip(5, blk.Hash)

	h, hash, err = bs.GetTip()
	if err != nil {
		t.Fatal(err)
	}
	if h != 5 || hash != blk.Hash {
		t.Fatalf("tip mismatch: h=%d hash=%s", h, crypto.ToHex(hash))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// StateStore
// ─────────────────────────────────────────────────────────────────────────────

func TestStateStore_SaveLoad(t *testing.T) {
	db := tempDB(t)
	ss := storage.NewStateStore(db)

	w, _ := crypto.NewWallet()
	st := state.NewStateDB()
	st.SetAccount(w.Address, &state.Account{Balance: 42_000, Nonce: 3})

	if err := ss.SaveState(st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	loaded, err := ss.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	acc := loaded.GetAccount(w.Address)
	if acc.Balance != 42_000 {
		t.Errorf("balance: want 42000, got %d", acc.Balance)
	}
	if acc.Nonce != 3 {
		t.Errorf("nonce: want 3, got %d", acc.Nonce)
	}
}

func TestStateStore_MultipleAccounts(t *testing.T) {
	db := tempDB(t)
	ss := storage.NewStateStore(db)

	const N = 10
	wallets := make([]*crypto.Wallet, N)
	st := state.NewStateDB()
	for i := 0; i < N; i++ {
		w, _ := crypto.NewWallet()
		wallets[i] = w
		st.SetAccount(w.Address, &state.Account{Balance: uint64(i+1) * 1_000})
	}

	_ = ss.SaveState(st)
	loaded, err := ss.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded.Len() != N {
		t.Errorf("want %d accounts, got %d", N, loaded.Len())
	}
	for i, w := range wallets {
		acc := loaded.GetAccount(w.Address)
		want := uint64(i+1) * 1_000
		if acc.Balance != want {
			t.Errorf("account %d: want balance %d, got %d", i, want, acc.Balance)
		}
	}
}

// Keep the OS path separator happy on Windows.
var _ = os.PathSeparator
