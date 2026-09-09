package state

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func makeAddr(seed byte) [crypto.AddressSize]byte {
	var addr [crypto.AddressSize]byte
	addr[0] = seed
	return addr
}

func TestStateDB_GetAccount_Unknown(t *testing.T) {
	st := NewStateDB()
	acc := st.GetAccount(makeAddr(1))
	if acc.Balance != 0 || acc.Nonce != 0 {
		t.Fatal("an unknown account should return a zero account")
	}
}

func TestStateDB_SetGet_Roundtrip(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(2)
	st.SetAccount(addr, &Account{Balance: 500, Nonce: 3})
	acc := st.GetAccount(addr)
	if acc.Balance != 500 {
		t.Errorf("Balance: want 500, got %d", acc.Balance)
	}
	if acc.Nonce != 3 {
		t.Errorf("Nonce: want 3, got %d", acc.Nonce)
	}
}

func TestStateDB_GetAccount_ReturnsCopy(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(3)
	st.SetAccount(addr, &Account{Balance: 100})
	acc := st.GetAccount(addr)
	acc.Balance = 9999 // mutate returned copy
	if st.GetAccount(addr).Balance != 100 {
		t.Fatal("GetAccount should return an independent copy")
	}
}

func TestStateDB_Len(t *testing.T) {
	st := NewStateDB()
	if st.Len() != 0 {
		t.Fatal("new StateDB should be empty")
	}
	st.SetAccount(makeAddr(1), &Account{Balance: 1})
	st.SetAccount(makeAddr(2), &Account{Balance: 2})
	if st.Len() != 2 {
		t.Errorf("Len: want 2, got %d", st.Len())
	}
}

func TestStateDB_GetBalance(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(4)
	st.SetAccount(addr, &Account{Balance: 777})
	if st.GetBalance(addr) != 777 {
		t.Errorf("GetBalance: want 777, got %d", st.GetBalance(addr))
	}
}

func TestStateDB_GetNonce(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(5)
	st.SetAccount(addr, &Account{Nonce: 12})
	if st.GetNonce(addr) != 12 {
		t.Errorf("GetNonce: want 12, got %d", st.GetNonce(addr))
	}
}

func TestStateDB_Credit(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(6)
	st.SetAccount(addr, &Account{Balance: 100})
	st.Credit(addr, 50)
	if st.GetBalance(addr) != 150 {
		t.Errorf("after Credit: want 150, got %d", st.GetBalance(addr))
	}
}

func TestStateDB_Credit_NewAccount(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(7)
	st.Credit(addr, 200)
	if st.GetBalance(addr) != 200 {
		t.Errorf("Credit on a new account: want 200, got %d", st.GetBalance(addr))
	}
}

func TestStateDB_CommitRoot_Deterministic(t *testing.T) {
	st := NewStateDB()
	st.SetAccount(makeAddr(1), &Account{Balance: 100})
	st.SetAccount(makeAddr(2), &Account{Balance: 200})
	r1 := st.CommitRoot()
	r2 := st.CommitRoot()
	if r1 != r2 {
		t.Fatal("CommitRoot is not deterministic")
	}
}

func TestStateDB_CommitRoot_ChangesWithState(t *testing.T) {
	st := NewStateDB()
	st.SetAccount(makeAddr(1), &Account{Balance: 100})
	r1 := st.CommitRoot()
	st.SetAccount(makeAddr(1), &Account{Balance: 200})
	r2 := st.CommitRoot()
	if r1 == r2 {
		t.Fatal("CommitRoot should change when the state changes")
	}
}

func TestStateDB_Snapshot_Independence(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(8)
	st.SetAccount(addr, &Account{Balance: 1000})
	snap := st.Snapshot()
	snap.SetAccount(addr, &Account{Balance: 9999})
	if st.GetBalance(addr) != 1000 {
		t.Fatal("modifying a snapshot should not affect the original")
	}
}

func TestStateDB_Apply(t *testing.T) {
	st := NewStateDB()
	addr := makeAddr(9)
	st.SetAccount(addr, &Account{Balance: 100})

	snap := st.Snapshot()
	snap.SetAccount(addr, &Account{Balance: 500})

	st.Apply(snap)
	if st.GetBalance(addr) != 500 {
		t.Errorf("after Apply: want 500, got %d", st.GetBalance(addr))
	}
}

func TestStateDB_CommitRoot_EmptyIsConsistent(t *testing.T) {
	st1 := NewStateDB()
	st2 := NewStateDB()
	if st1.CommitRoot() != st2.CommitRoot() {
		t.Fatal("two empty state DBs should have the same root")
	}
}

func TestStateDB_ForEach_InvalidKey(t *testing.T) {
	st := NewStateDB()
	st.accounts["not_hex_!@#$"] = &Account{Balance: 1}
	
	addr := makeAddr(99)
	st.SetAccount(addr, &Account{Balance: 100})
	
	count := 0
	st.ForEach(func(a [crypto.AddressSize]byte, acc *Account) {
		count++
	})
	if count != 1 {
		t.Errorf("ForEach visited %d accounts, expected 1", count)
	}
}
