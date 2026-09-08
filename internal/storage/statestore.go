package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
)

// keyPrefixAccount is the DB key prefix for account records.
var keyPrefixAccount = []byte("a:")

// StateStore persists and loads account state to/from a DB.
type StateStore struct{ db *DB }

// NewStateStore wraps db in a StateStore.
func NewStateStore(db *DB) *StateStore { return &StateStore{db: db} }

// SaveState writes every account from st into the database atomically.
// Existing accounts are overwritten; accounts not present in st are untouched
// (use a fresh DB or call this after loading the full desired state).
func (s *StateStore) SaveState(st *state.DB) error {
	batch := s.db.NewBatch()
	st.ForEach(func(addr [crypto.AddressSize]byte, acc *state.Account) {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(acc); err != nil {
			return
		}
		key := accountKey(crypto.AddressToHex(addr))
		batch.Put(key, buf.Bytes())
	})
	return s.db.Write(batch)
}

// LoadState reads all persisted accounts and returns a populated StateDB.
func (s *StateStore) LoadState() (*state.DB, error) {
	st := state.NewStateDB()
	err := s.db.IterPrefix(keyPrefixAccount, func(key, val []byte) error {
		hexAddr := string(key[len(keyPrefixAccount):])
		addr, err := crypto.HexToAddress(hexAddr)
		if err != nil {
			return fmt.Errorf("invalid stored address %q: %w", hexAddr, err)
		}
		var acc state.Account
		if err := gob.NewDecoder(bytes.NewReader(val)).Decode(&acc); err != nil {
			return fmt.Errorf("decode account %s: %w", hexAddr, err)
		}
		st.SetAccount(addr, &acc)
		return nil
	})
	return st, err
}

func accountKey(hexAddr string) []byte {
	k := make([]byte, len(keyPrefixAccount)+len(hexAddr))
	copy(k, keyPrefixAccount)
	copy(k[len(keyPrefixAccount):], hexAddr)
	return k
}
