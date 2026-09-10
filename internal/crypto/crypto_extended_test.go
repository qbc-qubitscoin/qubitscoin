package crypto_test

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestHash256(t *testing.T) {
	// empty
	h1 := crypto.Hash256(nil)
	var z [crypto.HashSize]byte
	if h1 == z {
		t.Errorf("empty hash should not be zero hash")
	}
	// non-empty
	h2 := crypto.Hash256([]byte("hello"))
	if h1 == h2 {
		t.Errorf("different inputs should have different hashes")
	}
}

func TestHashMany(t *testing.T) {
	h1 := crypto.HashMany([]byte("a"), []byte("b"))
	h2 := crypto.Hash256([]byte("ab"))
	if h1 != h2 {
		t.Errorf("HashMany doesn't match Hash256 of concatenated bytes")
	}
}

func TestToHex_HexToHash(t *testing.T) {
	h := crypto.Hash256([]byte("test"))
	hexStr := crypto.ToHex(h)
	
	h2, err := crypto.HexToHash(hexStr)
	if err != nil {
		t.Fatalf("HexToHash failed: %v", err)
	}
	if h != h2 {
		t.Errorf("round trip failed")
	}

	// bad hex
	_, err = crypto.HexToHash("not hex characters")
	if err == nil {
		t.Errorf("expected error for bad hex")
	}

	// wrong length
	shortHex := strings.Repeat("a", 62)
	_, err = crypto.HexToHash(shortHex)
	if err == nil {
		t.Errorf("expected error for wrong length")
	}
}

func TestHexToAddress_AddressToHex(t *testing.T) {
	addrBytes := crypto.Hash256([]byte("addr"))
	var addr [crypto.AddressSize]byte
	copy(addr[:], addrBytes[:])

	hexStr := crypto.AddressToHex(addr)
	
	addr2, err := crypto.HexToAddress(hexStr)
	if err != nil {
		t.Fatalf("HexToAddress failed: %v", err)
	}
	if addr != addr2 {
		t.Errorf("round trip failed")
	}

	// bad hex
	_, err = crypto.HexToAddress("not hex characters")
	if err == nil {
		t.Errorf("expected error for bad hex")
	}

	// wrong length
	shortHex := strings.Repeat("a", 62)
	_, err = crypto.HexToAddress(shortHex)
	if err == nil {
		t.Errorf("expected error for wrong length")
	}
}

func TestZeroHash(t *testing.T) {
	var z [crypto.HashSize]byte
	if crypto.ZeroHash != z {
		t.Errorf("ZeroHash is not all zeros")
	}
}

func TestDeriveAddress(t *testing.T) {
	pubKey := []byte("some public key")
	addr := crypto.DeriveAddress(pubKey)
	expected := crypto.Hash256(pubKey)
	if addr != expected {
		t.Errorf("DeriveAddress did not match Hash256 of pubkey")
	}
}

func TestSign_BadKey(t *testing.T) {
	_, err := crypto.Sign([]byte("bad key"), []byte("msg"))
	if err == nil {
		t.Errorf("expected error for bad private key")
	}
}

func TestVerify_BadKey(t *testing.T) {
	_, err := crypto.Verify([]byte("bad key"), []byte("msg"), make([]byte, crypto.SignatureSize))
	if err == nil {
		t.Errorf("expected error for bad public key")
	}
}

func TestVerify_BadSigLength(t *testing.T) {
	// Need a valid public key to pass the first check.
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}
	_, err = crypto.Verify(w.PublicKey, []byte("msg"), []byte("short"))
	if err == nil {
		t.Errorf("expected error for bad signature length")
	}
}

type failReader struct{}
func (f failReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("failed")
}

func TestNewWallet_Fail(t *testing.T) {
	// Save the original and restore it after the test.
	orig := rand.Reader
	defer func() { rand.Reader = orig }()

	// Overwrite with a reader that fails.
	rand.Reader = failReader{}

	_, err := crypto.NewWallet()
	if err == nil {
		t.Errorf("expected error when rand.Reader fails")
	}
}

