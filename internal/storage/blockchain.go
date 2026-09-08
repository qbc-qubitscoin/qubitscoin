package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ErrNotFound is returned when a requested block is absent.
var ErrNotFound = errors.New("storage: not found")

// Key prefixes — one byte + payload keeps iteration efficient.
var (
	keyPrefixBlock  = []byte("b:") // "b:" + 32-byte hash  → gob(Block)
	keyPrefixHeight = []byte("h:") // "h:" + 8-byte height → 32-byte hash
	keyTip          = []byte("tip")
)

// BlockStore persists and retrieves full blocks using a DB.
type BlockStore struct{ db *DB }

// NewBlockStore wraps db in a BlockStore.
func NewBlockStore(db *DB) *BlockStore { return &BlockStore{db: db} }

// PutBlock writes a block and updates the height→hash index.
// It does NOT update the chain tip; call UpdateTip separately after commit.
func (s *BlockStore) PutBlock(blk *core.Block) error {
	data, err := encodeBlock(blk)
	if err != nil {
		return err
	}
	batch := s.db.NewBatch()
	batch.Put(blockKey(blk.Hash), data)
	batch.Put(heightKey(blk.Header.Height), blk.Hash[:])
	return s.db.Write(batch)
}

// UpdateTip atomically records height + hash as the chain tip.
func (s *BlockStore) UpdateTip(height uint64, hash [crypto.HashSize]byte) error {
	buf := make([]byte, 8+crypto.HashSize)
	binary.BigEndian.PutUint64(buf[:8], height)
	copy(buf[8:], hash[:])
	return s.db.Put(keyTip, buf)
}

// GetTip returns the persisted chain tip (height, hash).
// Returns (0, zero, nil) when the database is empty.
func (s *BlockStore) GetTip() (height uint64, hash [crypto.HashSize]byte, err error) {
	val, err := s.db.Get(keyTip)
	if err != nil || val == nil {
		return 0, [crypto.HashSize]byte{}, err
	}
	height = binary.BigEndian.Uint64(val[:8])
	copy(hash[:], val[8:])
	return height, hash, nil
}

// GetBlock retrieves a block by its hash.
func (s *BlockStore) GetBlock(hash [crypto.HashSize]byte) (*core.Block, error) {
	val, err := s.db.Get(blockKey(hash))
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, ErrNotFound
	}
	return decodeBlock(val)
}

// GetBlockByHeight retrieves a block by height.
func (s *BlockStore) GetBlockByHeight(height uint64) (*core.Block, error) {
	hashBytes, err := s.db.Get(heightKey(height))
	if err != nil {
		return nil, err
	}
	if hashBytes == nil {
		return nil, ErrNotFound
	}
	var hash [crypto.HashSize]byte
	copy(hash[:], hashBytes)
	return s.GetBlock(hash)
}

// HasBlock reports whether a block with hash is stored.
func (s *BlockStore) HasBlock(hash [crypto.HashSize]byte) (bool, error) {
	return s.db.Has(blockKey(hash))
}

// ─────────────────────────────────────────────────────────────────────────────

func blockKey(hash [crypto.HashSize]byte) []byte {
	k := make([]byte, len(keyPrefixBlock)+crypto.HashSize)
	copy(k, keyPrefixBlock)
	copy(k[len(keyPrefixBlock):], hash[:])
	return k
}

func heightKey(h uint64) []byte {
	k := make([]byte, len(keyPrefixHeight)+8)
	copy(k, keyPrefixHeight)
	binary.BigEndian.PutUint64(k[len(keyPrefixHeight):], h)
	return k
}

func encodeBlock(blk *core.Block) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(blk); err != nil {
		return nil, fmt.Errorf("encode block h=%d: %w", blk.Header.Height, err)
	}
	return buf.Bytes(), nil
}

func decodeBlock(data []byte) (*core.Block, error) {
	var blk core.Block
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&blk); err != nil {
		return nil, fmt.Errorf("decode block: %w", err)
	}
	return &blk, nil
}
