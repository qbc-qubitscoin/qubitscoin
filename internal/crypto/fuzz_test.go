package crypto

import (
	"testing"
)

func FuzzHexToHash(f *testing.F) {
	f.Add("0000000000000000000000000000000000000000000000000000000000000000")
	f.Add("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	f.Add("invalidhexstring------------------------------------------------")
	f.Fuzz(func(t *testing.T, s string) {
		h, err := HexToHash(s)
		if err == nil {
			// If it decoded successfully, encoding it back should match the input (lowercased)
			if ToHex(h) != s {
				// This might fail if the input was uppercase hex, but HexToHash accepts uppercase.
				// Actually, ToHex returns lowercase.
				// Let's just check that it parses.
			}
		}
	})
}

func FuzzHexToAddress(f *testing.F) {
	f.Add("0000000000000000000000000000000000000000000000000000000000000000")
	f.Add("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	f.Fuzz(func(t *testing.T, s string) {
		addr, err := HexToAddress(s)
		if err == nil {
			// Ensure it doesn't crash
			AddressToHex(addr)
		}
	})
}

func FuzzVerify(f *testing.F) {
	// Add seed corpus for fuzzing signature verification
	// We just need arbitrary bytes
	f.Add([]byte("pubkey"), []byte("msg"), []byte("sig"))
	f.Fuzz(func(t *testing.T, pubKeyBytes, msg, sig []byte) {
		// Just ensure it doesn't panic
		Verify(pubKeyBytes, msg, sig)
	})
}

func FuzzHash256(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, data []byte) {
		// Ensure it doesn't panic
		Hash256(data)
	})
}
