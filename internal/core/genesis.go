package core

import (
	"bytes"
	"encoding/binary"
	"sort"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// GenesisConfig holds the parameters used to build the genesis block.
type GenesisConfig struct {
	ChainID       int
	Timestamp     int64
	ValidatorAddr [crypto.AddressSize]byte
	Allocations   map[[crypto.AddressSize]byte]uint64
}

// DefaultGenesisConfig returns a sensible genesis config for a single-validator testnet.
func DefaultGenesisConfig(validatorAddr [crypto.AddressSize]byte) *GenesisConfig {
	alloc := map[[crypto.AddressSize]byte]uint64{
		validatorAddr: 10_000_000 * OneQBC, // 10M QBC pre-mine to validator
	}
	return &GenesisConfig{
		ChainID:       ChainID,
		Timestamp:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		ValidatorAddr: validatorAddr,
		Allocations:   alloc,
	}
}

// Build creates and returns the genesis block (height 0, no transactions).
func (cfg *GenesisConfig) Build() *Block {
	var prevHash [crypto.HashSize]byte
	stateRoot := genesisStateRoot(cfg.Allocations)

	hdr := BlockHeader{
		Version:       ProtocolVersion,
		Height:        0,
		PrevHash:      prevHash,
		MerkleRoot:    EmptyMerkleRoot(),
		StateRoot:     stateRoot,
		Timestamp:     cfg.Timestamp,
		ValidatorAddr: cfg.ValidatorAddr,
		GasUsed:       0,
		GasLimit:      BlockGasLimit,
		BaseFee:       InitialBaseFee, // seed the EIP-1559 fee market
		BurnedFees:    0,
	}
	blk := &Block{Header: hdr, Txs: nil}
	blk.Hash = blk.ComputeHash()
	return blk
}

// genesisStateRoot produces a deterministic hash from the allocation map.
func genesisStateRoot(allocs map[[crypto.AddressSize]byte]uint64) [crypto.HashSize]byte {
	keys := make([][crypto.AddressSize]byte, 0, len(allocs))
	for k := range allocs {
		keys = append(keys, k)
	}
	// sort.Slice with bytes.Compare for deterministic ordering.
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})

	h := make([]byte, 0, len(allocs)*(crypto.AddressSize+8))
	b8 := make([]byte, 8)
	for _, k := range keys {
		h = append(h, k[:]...)
		binary.BigEndian.PutUint64(b8, allocs[k])
		h = append(h, b8...)
	}
	return crypto.Hash256(h)
}
