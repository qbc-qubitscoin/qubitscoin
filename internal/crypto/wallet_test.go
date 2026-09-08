package crypto

import (
	"testing"
)

// sharedWallet is generated once per test run to avoid paying key-gen cost N times.
var sharedWallet *Wallet

func init() {
	var err error
	sharedWallet, err = NewWallet()
	if err != nil {
		panic("failed to generate a test wallet: " + err.Error())
	}
}

func TestNewWallet_KeySizes(t *testing.T) {
	w := sharedWallet
	if len(w.PublicKey) != PublicKeySize {
		t.Errorf("public key: want %d bytes, got %d", PublicKeySize, len(w.PublicKey))
	}
	if len(w.PrivateKey) != PrivateKeySize {
		t.Errorf("private key: want %d bytes, got %d", PrivateKeySize, len(w.PrivateKey))
	}
}

func TestNewWallet_AddressSize(t *testing.T) {
	if len(sharedWallet.Address) != AddressSize {
		t.Errorf("address: want %d bytes, got %d", AddressSize, len(sharedWallet.Address))
	}
}

func TestNewWallet_AddressIsHashOfPubKey(t *testing.T) {
	w := sharedWallet
	derived := DeriveAddress(w.PublicKey)
	if derived != w.Address {
		t.Fatal("wallet address does not equal SHA-3-256 (public key)")
	}
}

func TestNewWallet_UniqueEachCall(t *testing.T) {
	w1, _ := NewWallet()
	w2, _ := NewWallet()
	if w1.Address == w2.Address {
		t.Fatal("two wallets should have different addresses")
	}
}

func TestSign_Verify_Roundtrip(t *testing.T) {
	w := sharedWallet
	msg := []byte("QubitsCoin test message")
	sig, err := Sign(w.PrivateKey, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != SignatureSize {
		t.Errorf("signature size: want %d, got %d", SignatureSize, len(sig))
	}
	ok, err := Verify(w.PublicKey, msg, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("a valid signature was rejected")
	}
}

func TestVerify_WrongMessage(t *testing.T) {
	w := sharedWallet
	sig, _ := Sign(w.PrivateKey, []byte("original"))
	ok, err := Verify(w.PublicKey, []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("signature over the wrong message should be invalid")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	w1 := sharedWallet
	w2, _ := NewWallet()
	sig, _ := Sign(w1.PrivateKey, []byte("hello"))
	ok, _ := Verify(w2.PublicKey, []byte("hello"), sig)
	if ok {
		t.Fatal("signature verified with the wrong public key")
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	w := sharedWallet
	msg := []byte("important data")
	sig, _ := Sign(w.PrivateKey, msg)
	sig[0] ^= 0xff // flip bits
	ok, _ := Verify(w.PublicKey, msg, sig)
	if ok {
		t.Fatal("the tampered signature should be invalid")
	}
}

func TestZeroize(t *testing.T) {
	w, _ := NewWallet()
	w.Zeroize()
	for _, b := range w.PrivateKey {
		if b != 0 {
			t.Fatal("private key was not zeroed after Zeroize()")
		}
	}
}

func TestDeriveAddress_Deterministic(t *testing.T) {
	a1 := DeriveAddress(sharedWallet.PublicKey)
	a2 := DeriveAddress(sharedWallet.PublicKey)
	if a1 != a2 {
		t.Fatal("DeriveAddress is not deterministic")
	}
}
