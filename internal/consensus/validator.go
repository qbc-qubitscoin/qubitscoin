package consensus

import (
	"errors"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// Validator represents a single consensus participant.
type Validator struct {
	Address     [crypto.AddressSize]byte
	PublicKey   []byte // ML-DSA-65 public key
	VotingPower uint64
}

// ValidatorSet is an immutable set of validators for a given epoch.
type ValidatorSet struct {
	validators   []*Validator
	totalPower   uint64
	addressIndex map[string]int
}

// NewValidatorSet creates a ValidatorSet from a slice of Validators.
func NewValidatorSet(validators []*Validator) (*ValidatorSet, error) {
	if len(validators) == 0 {
		return nil, errors.New("validator set must not be empty")
	}
	var total uint64
	idx := make(map[string]int, len(validators))
	for i, v := range validators {
		key := crypto.ToHex(v.Address)
		if _, dup := idx[key]; dup {
			return nil, errors.New("duplicate validator address")
		}
		idx[key] = i
		total += v.VotingPower
	}
	return &ValidatorSet{
		validators:   validators,
		totalPower:   total,
		addressIndex: idx,
	}, nil
}

// HasQuorum returns true if the given set of addresses holds > 2/3 of total power.
func (vs *ValidatorSet) HasQuorum(addresses [][crypto.AddressSize]byte) bool {
	var power uint64
	seen := make(map[string]bool)
	for _, addr := range addresses {
		key := crypto.ToHex(addr)
		if seen[key] {
			continue
		}
		seen[key] = true
		if i, ok := vs.addressIndex[key]; ok {
			power += vs.validators[i].VotingPower
		}
	}
	return 3*power > 2*vs.totalPower
}

// Proposer returns the validator responsible for proposing at the given height
// using round-robin selection.
func (vs *ValidatorSet) Proposer(height uint64) *Validator {
	idx := height % uint64(len(vs.validators))
	return vs.validators[idx]
}

// Get returns the validator with the given address or nil.
func (vs *ValidatorSet) Get(addr [crypto.AddressSize]byte) *Validator {
	key := crypto.ToHex(addr)
	if i, ok := vs.addressIndex[key]; ok {
		return vs.validators[i]
	}
	return nil
}

// Contains returns true if the address belongs to the validator set.
func (vs *ValidatorSet) Contains(addr [crypto.AddressSize]byte) bool {
	return vs.Get(addr) != nil
}

// All returns all validators.
func (vs *ValidatorSet) All() []*Validator {
	out := make([]*Validator, len(vs.validators))
	copy(out, vs.validators)
	return out
}

// TotalPower returns the aggregate voting power.
func (vs *ValidatorSet) TotalPower() uint64 { return vs.totalPower }
