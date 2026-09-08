package core

import (
	"encoding/binary"
	"errors"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// BlockHeader contains all consensus-critical header fields.
type BlockHeader struct {
	Version       uint32
	Height        uint64
	PrevHash      [crypto.HashSize]byte
	MerkleRoot    [crypto.HashSize]byte
	StateRoot     [crypto.HashSize]byte
	Timestamp     int64
	ValidatorAddr [crypto.AddressSize]byte
	GasUsed       uint64
	GasLimit      uint64
	BaseFee       uint64 // qubits per gas — burned portion of the fee
	BurnedFees    uint64 // total qubits burned in this block (BaseFee × GasUsed)
}

// Block is a header plus its transactions and validator signature.
type Block struct {
	Header    BlockHeader
	Txs       []*Transaction
	Signature []byte // ML-DSA-65 signature over Encode()
	Hash      [crypto.HashSize]byte
}

// Encode returns the canonical byte encoding of the header (for hashing / signing).
func (h *BlockHeader) Encode() []byte {
	buf := make([]byte, 0, 160)
	b4 := make([]byte, 4)
	b8 := make([]byte, 8)

	binary.BigEndian.PutUint32(b4, h.Version)
	buf = append(buf, b4...)
	binary.BigEndian.PutUint64(b8, h.Height)
	buf = append(buf, b8...)
	buf = append(buf, h.PrevHash[:]...)
	buf = append(buf, h.MerkleRoot[:]...)
	buf = append(buf, h.StateRoot[:]...)
	binary.BigEndian.PutUint64(b8, uint64(h.Timestamp))
	buf = append(buf, b8...)
	buf = append(buf, h.ValidatorAddr[:]...)
	binary.BigEndian.PutUint64(b8, h.GasUsed)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, h.GasLimit)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, h.BaseFee)
	buf = append(buf, b8...)
	binary.BigEndian.PutUint64(b8, h.BurnedFees)
	buf = append(buf, b8...)
	return buf
}

// ComputeHash returns SHA-3-256 of the encoded header.
func (b *Block) ComputeHash() [crypto.HashSize]byte {
	return crypto.Hash256(b.Header.Encode())
}

// SignHeader signs the encoded header with the validator's ML-DSA-65 private key.
func (b *Block) SignHeader(privKeyBytes []byte) error {
	sig, err := crypto.Sign(privKeyBytes, b.Header.Encode())
	if err != nil {
		return err
	}
	b.Signature = sig
	b.Hash = b.ComputeHash()
	return nil
}

// VerifyValidatorSig verifies the block's validator signature.
func (b *Block) VerifyValidatorSig(pubKeyBytes []byte) error {
	ok, err := crypto.Verify(pubKeyBytes, b.Header.Encode(), b.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid block validator signature")
	}
	return nil
}

// NewBlock constructs a block, validates all transactions, and computes the Merkle root.
// stateRoot must be computed by the caller after applying transactions.
// baseFee is the current block's base fee (qubits per gas).
// burnedFees is the total base fee burned across all transactions.
func NewBlock(
	height uint64,
	prevHash, stateRoot [crypto.HashSize]byte,
	timestamp int64,
	validatorAddr [crypto.AddressSize]byte,
	txs []*Transaction,
	gasUsed, baseFee, burnedFees uint64,
) (*Block, error) {

	txHashes := make([][crypto.HashSize]byte, len(txs))
	for i, tx := range txs {
		if err := tx.BasicValidate(); err != nil {
			return nil, err
		}
		txHashes[i] = tx.Hash
	}

	merkleRoot := ComputeMerkleRoot(txHashes)

	hdr := BlockHeader{
		Version:       ProtocolVersion,
		Height:        height,
		PrevHash:      prevHash,
		MerkleRoot:    merkleRoot,
		StateRoot:     stateRoot,
		Timestamp:     timestamp,
		ValidatorAddr: validatorAddr,
		GasUsed:       gasUsed,
		GasLimit:      BlockGasLimit,
		BaseFee:       baseFee,
		BurnedFees:    burnedFees,
	}
	blk := &Block{Header: hdr, Txs: txs}
	blk.Hash = blk.ComputeHash()
	return blk, nil
}
