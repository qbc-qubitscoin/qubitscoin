package core

import (
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// testWallet is a package-level wallet reused across tests to save key-gen time.
var testWallet *crypto.Wallet
var testWallet2 *crypto.Wallet

func init() {
	var err error
	testWallet, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
	testWallet2, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
}

func newSignedTransfer(t *testing.T, from *crypto.Wallet, to [crypto.AddressSize]byte, nonce, amount uint64) *Transaction {
	t.Helper()
	tx := NewTransfer(from.Address, to, from.PublicKey, nonce, amount, MinGasPrice)
	if err := tx.Sign(from.PrivateKey); err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tx
}

func TestTransaction_SignVerify(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 100*OneQBC)
	if err := tx.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestTransaction_TamperedAmountFails(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 100*OneQBC)
	tx.Amount = 999 * OneQBC // tamper after signing
	if err := tx.Verify(); err == nil {
		t.Fatal("tampered tx should fail Verify")
	}
}

func TestTransaction_WrongPublicKeyFails(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)
	tx.PublicKey = testWallet2.PublicKey // swap key
	if err := tx.Verify(); err == nil {
		t.Fatal("the wrong public key should fail Verify")
	}
}

func TestTransaction_ComputeHash_Deterministic(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 1, 50*OneQBC)
	h1 := tx.ComputeHash()
	h2 := tx.ComputeHash()
	if h1 != h2 {
		t.Fatal("ComputeHash is not deterministic")
	}
}

func TestTransaction_DifferentNonce_DifferentHash(t *testing.T) {
	tx1 := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)
	tx2 := newSignedTransfer(t, testWallet, testWallet2.Address, 1, 1)
	if tx1.Hash == tx2.Hash {
		t.Fatal("different nonces should produce different hashes")
	}
}

func TestTransaction_BasicValidate_Valid(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)
	if err := tx.BasicValidate(); err != nil {
		t.Fatalf("BasicValidate: %v", err)
	}
}

func TestTransaction_BasicValidate_BadVersion(t *testing.T) {
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)
	tx.Version = 99
	if err := tx.BasicValidate(); err == nil {
		t.Fatal("a bad version should fail BasicValidate")
	}
}

func TestTransaction_BasicValidate_ZeroGasLimit(t *testing.T) {
	tx := &Transaction{
		Version:   1,
		Type:      TxTransfer,
		GasLimit:  0,
		GasPrice:  MinGasPrice,
		Timestamp: time.Now().UnixNano(),
		PublicKey: testWallet.PublicKey,
	}
	if err := tx.BasicValidate(); err == nil {
		t.Fatal("zero gas limit should fail BasicValidate")
	}
}

func TestTransaction_BasicValidate_LowGasPrice(t *testing.T) {
	tx := &Transaction{
		Version:   1,
		Type:      TxTransfer,
		GasLimit:  GasTransfer,
		GasPrice:  MinGasPrice - 1,
		Timestamp: time.Now().UnixNano(),
		PublicKey: testWallet.PublicKey,
	}
	if err := tx.BasicValidate(); err == nil {
		t.Fatal("gas price below a minimum should fail BasicValidate")
	}
}

func TestNewTransfer_Fields(t *testing.T) {
	tx := NewTransfer(testWallet.Address, testWallet2.Address, testWallet.PublicKey, 5, 100, MinGasPrice)
	if tx.Version != 1 {
		t.Errorf("Version: want 1, got %d", tx.Version)
	}
	if tx.Type != TxTransfer {
		t.Errorf("Type: want TxTransfer, got %d", tx.Type)
	}
	if tx.Nonce != 5 {
		t.Errorf("Nonce: want 5, got %d", tx.Nonce)
	}
	if tx.Amount != 100 {
		t.Errorf("Amount: want 100, got %d", tx.Amount)
	}
	if tx.GasLimit != GasTransfer {
		t.Errorf("GasLimit: want %d, got %d", GasTransfer, tx.GasLimit)
	}
}
