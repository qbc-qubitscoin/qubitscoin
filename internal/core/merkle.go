package core

import (
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// MerkleTree holds a simple binary Merkle tree built over SHA-3-256 leaf hashes.
type MerkleTree struct {
	Root   [crypto.HashSize]byte
	leaves [][crypto.HashSize]byte
}

// NewMerkleTree constructs a Merkle tree from a slice of leaf hashes.
func NewMerkleTree(leaves [][crypto.HashSize]byte) *MerkleTree {
	root := computeRoot(leaves)
	return &MerkleTree{Root: root, leaves: leaves}
}

// ComputeMerkleRoot builds a root hash from a slice of transaction hashes.
func ComputeMerkleRoot(txHashes [][crypto.HashSize]byte) [crypto.HashSize]byte {
	return computeRoot(txHashes)
}

// EmptyMerkleRoot returns the hash used when there are no transactions.
func EmptyMerkleRoot() [crypto.HashSize]byte {
	return crypto.Hash256([]byte("empty"))
}

func computeRoot(nodes [][crypto.HashSize]byte) [crypto.HashSize]byte {
	if len(nodes) == 0 {
		return EmptyMerkleRoot()
	}
	current := make([][crypto.HashSize]byte, len(nodes))
	copy(current, nodes)
	for len(current) > 1 {
		if len(current)%2 != 0 {
			// Duplicate last node for odd count.
			current = append(current, current[len(current)-1])
		}
		next := make([][crypto.HashSize]byte, len(current)/2)
		for i := 0; i < len(current); i += 2 {
			next[i/2] = crypto.HashMany(current[i][:], current[i+1][:])
		}
		current = next
	}
	return current[0]
}
