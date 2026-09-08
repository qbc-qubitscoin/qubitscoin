package core

import (
	"testing"
	"time"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestCore_TransactionBasicValidate(t *testing.T) {
	w, _ := crypto.NewWallet()
	tx := NewTransfer(w.Address, w.Address, w.PublicKey, 1, 10, MinGasPrice)
	tx.Sign(w.PrivateKey)
	
	if err := tx.BasicValidate(); err != nil {
		t.Fatalf("expected valid tx, got %v", err)
	}

	// 1. Version
	tx.Version = 2
	if err := tx.BasicValidate(); err == nil || err.Error() != "unsupported tx version" {
		t.Errorf("expected unsupported tx version, got %v", err)
	}
	tx.Version = 1

	// 2. GasLimit
	tx.GasLimit = 0
	if err := tx.BasicValidate(); err == nil || err.Error() != "gas limit must be > 0" {
		t.Errorf("expected gas limit > 0, got %v", err)
	}
	tx.GasLimit = GasTransfer

	// 3. GasPrice
	tx.GasPrice = MinGasPrice - 1
	if err := tx.BasicValidate(); err == nil || err.Error() != "gas price below a minimum" {
		t.Errorf("expected gas price below minimum, got %v", err)
	}
	tx.GasPrice = MinGasPrice

	// 4. Timestamp
	tx.Timestamp = 0
	if err := tx.BasicValidate(); err == nil || err.Error() != "invalid timestamp" {
		t.Errorf("expected invalid timestamp, got %v", err)
	}
	tx.Timestamp = time.Now().UnixNano()

	// 5. PublicKey size
	tx.PublicKey = make([]byte, crypto.PublicKeySize-1)
	if err := tx.BasicValidate(); err == nil || err.Error() != "invalid public key size" {
		t.Errorf("expected invalid public key size, got %v", err)
	}
	tx.PublicKey = w.PublicKey

	// 6. Signature size
	tx.Signature = make([]byte, crypto.SignatureSize-1)
	if err := tx.BasicValidate(); err == nil || err.Error() != "invalid signature size" {
		t.Errorf("expected invalid signature size, got %v", err)
	}
	tx.Signature = make([]byte, crypto.SignatureSize)
	
	// 7. Signature verify err -> from does not match public key
	tx.From[0] ^= 0xFF
	if err := tx.BasicValidate(); err == nil || err.Error() != "from address does not match the public key" {
		t.Errorf("expected address mismatch, got %v", err)
	}
}

func TestCore_GenesisStateRootSort(t *testing.T) {
	var k1, k2 [crypto.AddressSize]byte
	k1[0] = 2
	k2[0] = 1
	root := genesisStateRoot(map[[crypto.AddressSize]byte]uint64{
		k1: 100,
		k2: 200,
	})
	if root == [crypto.HashSize]byte{} {
		t.Errorf("expected non-empty root for empty allocs")
	}
}
