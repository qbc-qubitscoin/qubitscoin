package state

import "github.com/qbc-qubitscoin/qubitscoin/internal/crypto"

// Account holds the on-chain state for a single address.
type Account struct {
	Nonce       uint64
	Balance     uint64
	CodeHash    [crypto.HashSize]byte // zero for EOA
	StorageRoot [crypto.HashSize]byte // zero for EOA
}

// IsContract returns true if this account has associated code.
func (a *Account) IsContract() bool {
	return a.CodeHash != crypto.ZeroHash
}

// Clone returns a deep copy of the account.
func (a *Account) Clone() *Account {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}
