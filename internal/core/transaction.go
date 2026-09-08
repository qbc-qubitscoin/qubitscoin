package core

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// Transaction represents a signed QubitsCoin transaction.
type Transaction struct {
	Version   uint8
	Type      TxType
	Nonce     uint64
	From      [crypto.AddressSize]byte
	To        [crypto.AddressSize]byte
	Amount    uint64
	GasLimit  uint64
	GasPrice  uint64
	Timestamp int64
	Data      []byte // contract bytecode (Deploy) or call payload (Call)
	PublicKey []byte // ML-DSA-65 public key (1952 bytes)
	Signature []byte // ML-DSA-65 signature (3309 bytes)
	Hash      [crypto.HashSize]byte
}

// signingPayload returns the canonical bytes that are signed.
func (tx *Transaction) signingPayload() []byte {
	buf := make([]byte, 0, 256+len(tx.Data)+len(tx.PublicKey))
	buf = append(buf, tx.Version, uint8(tx.Type))
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, tx.Nonce)
	buf = append(buf, b8...)
	buf = append(buf, tx.From[:]...)
	buf = append(buf, tx.To[:]...)
	binary.BigEndian.PutUint64(b8, tx.Amount)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, tx.GasLimit)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, tx.GasPrice)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, uint64(tx.Timestamp))
	buf = append(buf, b8...)
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(len(tx.Data)))
	buf = append(buf, b4...)
	buf = append(buf, tx.Data...)
	buf = append(buf, tx.PublicKey...)
	return buf
}

// Sign signs the transaction with the given ML-DSA-65 private key bytes.
func (tx *Transaction) Sign(privKeyBytes []byte) error {
	sig, err := crypto.Sign(privKeyBytes, tx.signingPayload())
	if err != nil {
		return err
	}
	tx.Signature = sig
	tx.Hash = tx.ComputeHash()
	return nil
}

// Verify checks the transaction's signature and that From == SHA-3-256(PublicKey).
func (tx *Transaction) Verify() error {
	derivedAddr := crypto.DeriveAddress(tx.PublicKey)
	if derivedAddr != tx.From {
		return errors.New("from address does not match the public key")
	}
	ok, err := crypto.Verify(tx.PublicKey, tx.signingPayload(), tx.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid transaction signature")
	}
	return nil
}

// ComputeHash returns SHA-3-256 of the signing payload + signature.
func (tx *Transaction) ComputeHash() [crypto.HashSize]byte {
	return crypto.HashMany(tx.signingPayload(), tx.Signature)
}

// BasicValidate performs stateless validation.
func (tx *Transaction) BasicValidate() error {
	if tx.Version != 1 {
		return errors.New("unsupported tx version")
	}
	if tx.GasLimit == 0 {
		return errors.New("gas limit must be > 0")
	}
	if tx.GasPrice < MinGasPrice {
		return errors.New("gas price below a minimum")
	}
	if tx.Timestamp <= 0 {
		return errors.New("invalid timestamp")
	}
	if len(tx.PublicKey) != crypto.PublicKeySize {
		return errors.New("invalid public key size")
	}
	if len(tx.Signature) != crypto.SignatureSize {
		return errors.New("invalid signature size")
	}
	return tx.Verify()
}

// NewTransfer creates a transfer transaction (not yet signed).
func NewTransfer(from [crypto.AddressSize]byte, to [crypto.AddressSize]byte,
	pubKey []byte, nonce, amount, gasPrice uint64) *Transaction {
	return &Transaction{
		Version:   1,
		Type:      TxTransfer,
		Nonce:     nonce,
		From:      from,
		To:        to,
		Amount:    amount,
		GasLimit:  GasTransfer,
		GasPrice:  gasPrice,
		Timestamp: time.Now().UnixNano(),
		PublicKey: pubKey,
	}
}
