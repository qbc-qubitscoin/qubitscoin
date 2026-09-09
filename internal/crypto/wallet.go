package crypto

import (
	"crypto/rand"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

const (
	PublicKeySize  = 1952
	PrivateKeySize = 4032
	SignatureSize  = 3309
	AddressSize    = 32 // SHA-3-256 of a public key
)

// Wallet holds an ML-DSA-65 key pair.
type Wallet struct {
	PublicKey  []byte // 1952 bytes
	PrivateKey []byte // 4032 bytes
	Address    [AddressSize]byte
}

// NewWallet generates a fresh ML-DSA-65 key pair using secure randomness.
func NewWallet() (*Wallet, error) {
	pub, priv, err := mldsa65.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return walletFromKeys(pub, priv)
}

func walletFromKeys(pub *mldsa65.PublicKey, priv *mldsa65.PrivateKey) (*Wallet, error) {
	pubBytes, _ := pub.MarshalBinary()
	privBytes, _ := priv.MarshalBinary()
	addr := DeriveAddress(pubBytes)
	return &Wallet{
		PublicKey:  pubBytes,
		PrivateKey: privBytes,
		Address:    addr,
	}, nil
}

// DeriveAddress computes Address = SHA-3-256(pubKeyBytes).
func DeriveAddress(pubKeyBytes []byte) [AddressSize]byte {
	return Hash256(pubKeyBytes)
}

// Zeroize overwrites private key material in memory.
func (w *Wallet) Zeroize() {
	for i := range w.PrivateKey {
		w.PrivateKey[i] = 0
	}
}
