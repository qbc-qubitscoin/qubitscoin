package keystore_test

import (
	"path/filepath"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/keystore"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}

	path := filepath.Join(t.TempDir(), "wallet.json")
	const password = "correct-horse-battery-staple"

	if err := keystore.Encrypt(path, password, w); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	loaded, err := keystore.Decrypt(path, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if loaded.Address != w.Address {
		t.Errorf("address mismatch")
	}
	if string(loaded.PrivateKey) != string(w.PrivateKey) {
		t.Errorf("private key mismatch")
	}
	if string(loaded.PublicKey) != string(w.PublicKey) {
		t.Errorf("public key mismatch")
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	w, _ := crypto.NewWallet()
	path := filepath.Join(t.TempDir(), "wallet.json")
	_ = keystore.Encrypt(path, "correct-password", w)

	_, err := keystore.Decrypt(path, "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestPeekAddress(t *testing.T) {
	w, _ := crypto.NewWallet()
	path := filepath.Join(t.TempDir(), "wallet.json")
	_ = keystore.Encrypt(path, "secret", w)

	addr, err := keystore.PeekAddress(path)
	if err != nil {
		t.Fatalf("PeekAddress: %v", err)
	}
	if addr != crypto.AddressToHex(w.Address) {
		t.Errorf("address mismatch: want %s got %s", crypto.AddressToHex(w.Address), addr)
	}
}

func TestEncrypt_FilePermissions(t *testing.T) {
	w, _ := crypto.NewWallet()
	path := filepath.Join(t.TempDir(), "wallet.json")
	if err := keystore.Encrypt(path, "pw", w); err != nil {
		t.Fatal(err)
	}
	// Calling Encrypt again on the same path should overwrite.
	if err := keystore.Encrypt(path, "pw2", w); err != nil {
		t.Fatalf("overwrite keystore: %v", err)
	}
	_, err := keystore.Decrypt(path, "pw2")
	if err != nil {
		t.Fatalf("Decrypt after overwrite: %v", err)
	}
}
