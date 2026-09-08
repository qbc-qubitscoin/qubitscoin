package core

import (
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func FuzzTransactionValidate(f *testing.F) {
	// Seed with a valid-looking but incorrect signature tx
	f.Add(
		uint8(1),                 // Version
		uint8(TxTransfer),        // Type
		uint64(1),                // Nonce
		uint64(100),              // Amount
		uint64(21000),            // GasLimit
		uint64(10),               // GasPrice
		int64(time.Now().Unix()), // Timestamp
		[]byte{},                 // Data
		make([]byte, crypto.PublicKeySize), // PublicKey
		make([]byte, crypto.SignatureSize), // Signature
	)

	f.Fuzz(func(t *testing.T, version, typ uint8, nonce, amount, gasLimit, gasPrice uint64, ts int64, data, pubKey, sig []byte) {
		tx := &Transaction{
			Version:   version,
			Type:      TxType(typ),
			Nonce:     nonce,
			Amount:    amount,
			GasLimit:  gasLimit,
			GasPrice:  gasPrice,
			Timestamp: ts,
			Data:      data,
			PublicKey: pubKey,
			Signature: sig,
		}

		// Ensure BasicValidate doesn't panic on malformed inputs
		_ = tx.BasicValidate()
	})
}

func FuzzNextBaseFee(f *testing.F) {
	f.Add(uint64(10), uint64(250000000))
	f.Add(uint64(1), uint64(0))
	f.Add(uint64(1000), uint64(500000000))
	
	f.Fuzz(func(t *testing.T, current, gasUsed uint64) {
		// Ensure no panic
		next := NextBaseFee(current, gasUsed)
		if next < MinBaseFee {
			t.Errorf("NextBaseFee %d is less than MinBaseFee %d", next, MinBaseFee)
		}
	})
}
