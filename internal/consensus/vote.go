package consensus

import (
	"encoding/binary"
	"errors"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

const (
	VotePrevote   uint8 = 0x01
	VotePrecommit uint8 = 0x02
)

// Vote is a signed BFT vote from a validator.
type Vote struct {
	Type      uint8
	Height    uint64
	Round     uint32
	BlockHash [crypto.HashSize]byte
	Voter     [crypto.AddressSize]byte
	PublicKey []byte // ML-DSA-65 public key
	Signature []byte
}

// payload returns the canonical bytes that are signed.
func (v *Vote) payload() []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, v.Type)
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, v.Height)
	buf = append(buf, b8...)
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, v.Round)
	buf = append(buf, b4...)
	buf = append(buf, v.BlockHash[:]...)
	buf = append(buf, v.Voter[:]...)
	return buf
}

// Sign signs the vote with the given ML-DSA-65 private key bytes.
func (v *Vote) Sign(privKeyBytes []byte) error {
	sig, err := crypto.Sign(privKeyBytes, v.payload())
	if err != nil {
		return err
	}
	v.Signature = sig
	return nil
}

// Verify checks the vote's ML-DSA-65 signature.
func (v *Vote) Verify() error {
	if len(v.PublicKey) != crypto.PublicKeySize {
		return errors.New("invalid public key size in a vote")
	}
	if len(v.Signature) != crypto.SignatureSize {
		return errors.New("invalid signature size in a vote")
	}
	derivedAddr := crypto.DeriveAddress(v.PublicKey)
	if derivedAddr != v.Voter {
		return errors.New("vote public key does not match voter address")
	}
	ok, _ := crypto.Verify(v.PublicKey, v.payload(), v.Signature)
	if !ok {
		return errors.New("invalid vote signature")
	}
	return nil
}
