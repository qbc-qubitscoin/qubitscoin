package consensus

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func makeValidator(seed byte, power uint64) *Validator {
	var addr [crypto.AddressSize]byte
	addr[0] = seed
	return &Validator{Address: addr, VotingPower: power}
}

func TestNewValidatorSet_Empty(t *testing.T) {
	_, err := NewValidatorSet(nil)
	if err == nil {
		t.Fatal("an empty validator set should return an error")
	}
}

func TestNewValidatorSet_Duplicate(t *testing.T) {
	v := makeValidator(1, 10)
	_, err := NewValidatorSet([]*Validator{v, v})
	if err == nil {
		t.Fatal("duplicate validator address should return an error")
	}
}

func TestNewValidatorSet_Valid(t *testing.T) {
	vs, err := NewValidatorSet([]*Validator{
		makeValidator(1, 10),
		makeValidator(2, 20),
	})
	if err != nil {
		t.Fatalf("NewValidatorSet: %v", err)
	}
	if vs.TotalPower() != 30 {
		t.Errorf("TotalPower: want 30, got %d", vs.TotalPower())
	}
}

func TestValidatorSet_Contains(t *testing.T) {
	v := makeValidator(1, 10)
	vs, _ := NewValidatorSet([]*Validator{v})
	if !vs.Contains(v.Address) {
		t.Fatal("Contains: known validator not found")
	}
	var unknown [crypto.AddressSize]byte
	unknown[0] = 0xFF
	if vs.Contains(unknown) {
		t.Fatal("Contains: unknown address should return false")
	}
}

func TestValidatorSet_Get(t *testing.T) {
	v := makeValidator(2, 50)
	vs, _ := NewValidatorSet([]*Validator{v})
	got := vs.Get(v.Address)
	if got == nil {
		t.Fatal("Get: validator not found")
	}
	if got.VotingPower != 50 {
		t.Errorf("VotingPower: want 50, got %d", got.VotingPower)
	}
}

func TestValidatorSet_HasQuorum_SingleValidator(t *testing.T) {
	v := makeValidator(1, 1)
	vs, _ := NewValidatorSet([]*Validator{v})
	// Single validator: always quorum if they sign.
	if !vs.HasQuorum([][crypto.AddressSize]byte{v.Address}) {
		t.Fatal("a single validator with a self-vote should have quorum")
	}
}

func TestValidatorSet_HasQuorum_ThreeValidators(t *testing.T) {
	v1 := makeValidator(1, 1)
	v2 := makeValidator(2, 1)
	v3 := makeValidator(3, 1)
	vs, _ := NewValidatorSet([]*Validator{v1, v2, v3})

	// 1 of 3 — no quorum (1/3 ≤ 2/3).
	if vs.HasQuorum([][crypto.AddressSize]byte{v1.Address}) {
		t.Fatal("1/3 should not be quorum")
	}
	// 2 of 3 — quorum (2/3 > 2/3 is false, need strictly >).
	// 3*2 > 2*3 → 6 > 6 → false. So 2/3 is NOT quorum.
	if vs.HasQuorum([][crypto.AddressSize]byte{v1.Address, v2.Address}) {
		t.Fatal("exactly 2/3 should NOT be quorum (need strictly >)")
	}
	// 3 of 3 — quorum.
	if !vs.HasQuorum([][crypto.AddressSize]byte{v1.Address, v2.Address, v3.Address}) {
		t.Fatal("3/3 should have quorum")
	}
}

func TestValidatorSet_HasQuorum_WeightedPower(t *testing.T) {
	v1 := makeValidator(1, 10)
	v2 := makeValidator(2, 5)
	vs, _ := NewValidatorSet([]*Validator{v1, v2})
	// v1 has 10/15 power — 3*10 > 2*15 → 30 > 30 → false (not quorum alone).
	if vs.HasQuorum([][crypto.AddressSize]byte{v1.Address}) {
		t.Fatal("10/15 power should not be quorum (need strictly > 2/3)")
	}
	// Both: 15/15 — quorum.
	if !vs.HasQuorum([][crypto.AddressSize]byte{v1.Address, v2.Address}) {
		t.Fatal("full power should have quorum")
	}
}

func TestValidatorSet_HasQuorum_DuplicatesIgnored(t *testing.T) {
	// 3-validator set: one validator has 1/3 power — not quorum alone.
	// Repeating their address many times must not manufacture quorum.
	v1 := makeValidator(1, 1)
	v2 := makeValidator(2, 1)
	v3 := makeValidator(3, 1)
	vs, _ := NewValidatorSet([]*Validator{v1, v2, v3})

	addrs := make([][crypto.AddressSize]byte, 10)
	for i := range addrs {
		addrs[i] = v1.Address // same address 10×
	}
	if vs.HasQuorum(addrs) {
		t.Fatal("duplicate addresses must not multiply voting power")
	}
}

func TestValidatorSet_Proposer_RoundRobin(t *testing.T) {
	v1 := makeValidator(1, 1)
	v2 := makeValidator(2, 1)
	vs, _ := NewValidatorSet([]*Validator{v1, v2})

	// Height 0 → index 0, height 1 → index 1, height 2 → index 0 again.
	if vs.Proposer(0).Address != v1.Address {
		t.Error("height 0 should propose v1")
	}
	if vs.Proposer(1).Address != v2.Address {
		t.Error("height 1 should propose v2")
	}
	if vs.Proposer(2).Address != v1.Address {
		t.Error("height 2 should wrap to v1")
	}
}

func TestValidatorSet_All(t *testing.T) {
	vs, _ := NewValidatorSet([]*Validator{
		makeValidator(1, 1),
		makeValidator(2, 2),
	})
	all := vs.All()
	if len(all) != 2 {
		t.Errorf("All: want 2, got %d", len(all))
	}
}
