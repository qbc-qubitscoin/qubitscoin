package state

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestAccount_IsContract_EOA(t *testing.T) {
	a := &Account{}
	if a.IsContract() {
		t.Fatal("an empty account should not be a contract")
	}
}

func TestAccount_IsContract_WithCodeHash(t *testing.T) {
	a := &Account{CodeHash: crypto.Hash256([]byte("code"))}
	if !a.IsContract() {
		t.Fatal("an account with CodeHash should be a contract")
	}
}

func TestAccount_Clone_IndependentBalance(t *testing.T) {
	orig := &Account{Balance: 1000, Nonce: 5}
	clone := orig.Clone()
	clone.Balance = 9999
	if orig.Balance != 1000 {
		t.Fatal("modifying clone should not affect the original")
	}
}

func TestAccount_Clone_IndependentCodeHash(t *testing.T) {
	orig := &Account{CodeHash: crypto.Hash256([]byte("original"))}
	clone := orig.Clone()
	clone.CodeHash = crypto.Hash256([]byte("changed"))
	if orig.CodeHash == clone.CodeHash {
		t.Fatal("modifying a clone's CodeHash should not affect the original")
	}
}

func TestAccount_Clone_NilReturnsNil(t *testing.T) {
	var a *Account
	if a.Clone() != nil {
		t.Fatal("Clone of nil should return nil")
	}
}

func TestAccount_Clone_PreservesAllFields(t *testing.T) {
	orig := &Account{
		Nonce:    42,
		Balance:  1_000_000,
		CodeHash: crypto.Hash256([]byte("code")),
	}
	c := orig.Clone()
	if c.Nonce != orig.Nonce {
		t.Errorf("Nonce: want %d, got %d", orig.Nonce, c.Nonce)
	}
	if c.Balance != orig.Balance {
		t.Errorf("Balance: want %d, got %d", orig.Balance, c.Balance)
	}
	if c.CodeHash != orig.CodeHash {
		t.Error("CodeHash not preserved in Clone")
	}
}
