package state

import (
	"encoding/binary"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// DB StateDB is an in-memory trie-like account store protected by a mutex.
type DB struct {
	mu       sync.RWMutex
	accounts map[string]*Account // address hex -> account
}

// NewStateDB creates an empty state database.
func NewStateDB() *DB {
	return &DB{accounts: make(map[string]*Account)}
}

// GetAccount returns the account for the given address, or a zero account if absent.
func (s *DB) GetAccount(addr [crypto.AddressSize]byte) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := crypto.ToHex(addr)
	if a, ok := s.accounts[key]; ok {
		return a.Clone()
	}
	return &Account{}
}

// SetAccount stores the account for the given address.
func (s *DB) SetAccount(addr [crypto.AddressSize]byte, acc *Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[crypto.ToHex(addr)] = acc.Clone()
}

// GetBalance returns the qubit balance of an address.
func (s *DB) GetBalance(addr [crypto.AddressSize]byte) uint64 {
	return s.GetAccount(addr).Balance
}

// GetNonce returns the nonce of an address.
func (s *DB) GetNonce(addr [crypto.AddressSize]byte) uint64 {
	return s.GetAccount(addr).Nonce
}

// Credit adds amount to the balance of addr (used for block rewards).
func (s *DB) Credit(addr [crypto.AddressSize]byte, amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := crypto.ToHex(addr)
	acc := s.accounts[key]
	if acc == nil {
		acc = &Account{}
	} else {
		acc = acc.Clone()
	}
	acc.Balance += amount
	s.accounts[key] = acc
}

// CommitRoot computes a deterministic state root = SHA-3-256 (sorted account entries).
func (s *DB) CommitRoot() [crypto.HashSize]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.accounts))
	for k := range s.accounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf []byte
	b8 := make([]byte, 8)
	for _, k := range keys {
		a := s.accounts[k]
		buf = append(buf, []byte(k)...)
		binary.BigEndian.PutUint64(b8, a.Nonce)
		buf = append(buf, b8...)
		binary.BigEndian.PutUint64(b8, a.Balance)
		buf = append(buf, b8...)
		buf = append(buf, a.CodeHash[:]...)
		buf = append(buf, a.StorageRoot[:]...)
	}
	return crypto.Hash256(buf)
}

// Snapshot returns a deep copy of the current state (for rollback).
func (s *DB) Snapshot() *DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := NewStateDB()
	for k, v := range s.accounts {
		snap.accounts[k] = v.Clone()
	}
	return snap
}

// Len returns the number of accounts.
func (s *DB) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.accounts)
}

// ForEach calls fn for every account in the database (read-locked).
// The callback must not call any StateDB method that acquires the write lock.
func (s *DB) ForEach(fn func(addr [crypto.AddressSize]byte, acc *Account)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for hexKey, acc := range s.accounts {
		b, err := hex.DecodeString(hexKey)
		if err != nil || len(b) != crypto.AddressSize {
			continue
		}
		var addr [crypto.AddressSize]byte
		copy(addr[:], b)
		fn(addr, acc.Clone())
	}
}

// Apply replaces this state with the contents of another (used after dry-run).
func (s *DB) Apply(other *DB) {
	other.mu.RLock()
	defer other.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts = make(map[string]*Account, len(other.accounts))
	for k, v := range other.accounts {
		s.accounts[k] = v.Clone()
	}
}
