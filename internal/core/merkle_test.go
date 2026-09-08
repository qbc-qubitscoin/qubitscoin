package core

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func makeHashes(n int) [][crypto.HashSize]byte {
	hashes := make([][crypto.HashSize]byte, n)
	for i := range hashes {
		hashes[i] = crypto.Hash256([]byte{byte(i), byte(i >> 8)})
	}
	return hashes
}

func TestEmptyMerkleRoot_NotZero(t *testing.T) {
	r := EmptyMerkleRoot()
	var zero [crypto.HashSize]byte
	if r == zero {
		t.Fatal("an empty merkle root should not be zero")
	}
}

func TestEmptyMerkleRoot_Deterministic(t *testing.T) {
	if EmptyMerkleRoot() != EmptyMerkleRoot() {
		t.Fatal("EmptyMerkleRoot is not deterministic")
	}
}

func TestComputeMerkleRoot_Empty(t *testing.T) {
	r := ComputeMerkleRoot(nil)
	if r != EmptyMerkleRoot() {
		t.Fatal("an empty slice should return EmptyMerkleRoot()")
	}
}

func TestComputeMerkleRoot_SingleLeaf(t *testing.T) {
	leaves := makeHashes(1)
	r := ComputeMerkleRoot(leaves)
	if r != leaves[0] {
		t.Fatal("single leaf: the root should equal that leaf")
	}
}

func TestComputeMerkleRoot_TwoLeaves(t *testing.T) {
	leaves := makeHashes(2)
	r := ComputeMerkleRoot(leaves)
	expected := crypto.HashMany(leaves[0][:], leaves[1][:])
	if r != expected {
		t.Fatal("two leaves: the root should be hash(leaf0||leaf1)")
	}
}

func TestComputeMerkleRoot_OddLeaves(t *testing.T) {
	// Odd count: last leaf is duplicated, then paired.
	leaves := makeHashes(3)
	r := ComputeMerkleRoot(leaves)
	var zero [crypto.HashSize]byte
	if r == zero {
		t.Fatal("merkle root should not be zero")
	}
	// Result must be deterministic.
	if ComputeMerkleRoot(leaves) != r {
		t.Fatal("not deterministic")
	}
}

func TestComputeMerkleRoot_Deterministic(t *testing.T) {
	leaves := makeHashes(8)
	r1 := ComputeMerkleRoot(leaves)
	r2 := ComputeMerkleRoot(leaves)
	if r1 != r2 {
		t.Fatal("ComputeMerkleRoot is not deterministic")
	}
}

func TestComputeMerkleRoot_OrderMatters(t *testing.T) {
	leaves := makeHashes(4)
	r1 := ComputeMerkleRoot(leaves)
	// Swap first two leaves.
	leaves[0], leaves[1] = leaves[1], leaves[0]
	r2 := ComputeMerkleRoot(leaves)
	if r1 == r2 {
		t.Fatal("changing leaf order should change the root")
	}
}

func TestNewMerkleTree_RootMatchesCompute(t *testing.T) {
	leaves := makeHashes(6)
	tree := NewMerkleTree(leaves)
	expected := ComputeMerkleRoot(leaves)
	if tree.Root != expected {
		t.Fatal("NewMerkleTree.Root should match ComputeMerkleRoot")
	}
}
