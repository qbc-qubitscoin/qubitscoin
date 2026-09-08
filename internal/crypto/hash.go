package crypto

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

const HashSize = 32

// Hash256 returns the SHA-3-256 digest of data.
func Hash256(data []byte) [HashSize]byte {
	return sha3.Sum256(data)
}

// HashMany hashes a sequence of byte slices concatenated together.
func HashMany(parts ...[]byte) [HashSize]byte {
	h := sha3.New256()
	for _, p := range parts {
		h.Write(p)
	}
	var out [HashSize]byte
	h.Sum(out[:0])
	return out
}

// ToHex encodes a hash to a lowercase hex string.
func ToHex(h [HashSize]byte) string {
	return hex.EncodeToString(h[:])
}

// ZeroHash is the all-zero hash value.
var ZeroHash [HashSize]byte

// HexToHash decodes a 64-character hex string into a [HashSize]byte.
// Returns an error if the input is not exactly 64 hex characters.
func HexToHash(s string) ([HashSize]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return [HashSize]byte{}, err
	}
	if len(b) != HashSize {
		return [HashSize]byte{}, fmt.Errorf("hex hash must be %d bytes, got %d", HashSize, len(b))
	}
	var out [HashSize]byte
	copy(out[:], b)
	return out, nil
}

// HexToAddress decodes a hex string into an [AddressSize]byte.
func HexToAddress(s string) ([AddressSize]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return [AddressSize]byte{}, err
	}
	if len(b) != AddressSize {
		return [AddressSize]byte{}, fmt.Errorf("hex address must be %d bytes, got %d", AddressSize, len(b))
	}
	var out [AddressSize]byte
	copy(out[:], b)
	return out, nil
}

// AddressToHex encodes an address to a lowercase hex string.
func AddressToHex(addr [AddressSize]byte) string {
	return hex.EncodeToString(addr[:])
}
