package vm

import (
	"sync"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ContractStorage provides per-contract, per-slot uint64 storage.
type ContractStorage struct {
	mu   sync.RWMutex
	data map[string]map[uint32]uint64 // addrHex -> slotID -> value
}

// NewContractStorage creates an empty ContractStorage.
func NewContractStorage() *ContractStorage {
	return &ContractStorage{data: make(map[string]map[uint32]uint64)}
}

// Get returns the value stored at (contractAddr, slot), or 0 if unset.
func (s *ContractStorage) Get(contractAddr [crypto.AddressSize]byte, slot uint32) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := crypto.ToHex(contractAddr)
	if slots, ok := s.data[key]; ok {
		return slots[slot]
	}
	return 0
}

// Set stores value at (contractAddr, slot).
func (s *ContractStorage) Set(contractAddr [crypto.AddressSize]byte, slot uint32, value uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := crypto.ToHex(contractAddr)
	if _, ok := s.data[key]; !ok {
		s.data[key] = make(map[uint32]uint64)
	}
	s.data[key][slot] = value
}
