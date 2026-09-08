package core

import (
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestBlockHeader_Encode_Deterministic(t *testing.T) {
	h := BlockHeader{
		Version:   ProtocolVersion,
		Height:    42,
		Timestamp: time.Now().UnixNano(),
		GasLimit:  BlockGasLimit,
	}
	e1 := h.Encode()
	e2 := h.Encode()
	if string(e1) != string(e2) {
		t.Fatal("Encode is not deterministic")
	}
}

func TestBlockHeader_Encode_ChangesWithHeight(t *testing.T) {
	h1 := BlockHeader{Version: ProtocolVersion, Height: 1}
	h2 := BlockHeader{Version: ProtocolVersion, Height: 2}
	if string(h1.Encode()) == string(h2.Encode()) {
		t.Fatal("different heights should produce different encodings")
	}
}

func TestBlock_ComputeHash_Deterministic(t *testing.T) {
	blk := &Block{Header: BlockHeader{Height: 1, Version: ProtocolVersion}}
	h1 := blk.ComputeHash()
	h2 := blk.ComputeHash()
	if h1 != h2 {
		t.Fatal("ComputeHash is not deterministic")
	}
}

func TestBlock_SignHeader_Verify(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}
	blk := &Block{
		Header: BlockHeader{
			Version:       ProtocolVersion,
			Height:        1,
			ValidatorAddr: w.Address,
		},
	}
	if err := blk.SignHeader(w.PrivateKey); err != nil {
		t.Fatalf("SignHeader: %v", err)
	}
	if len(blk.Signature) == 0 {
		t.Fatal("signature should not be empty after SignHeader")
	}
	if err := blk.VerifyValidatorSig(w.PublicKey); err != nil {
		t.Fatalf("VerifyValidatorSig: %v", err)
	}
}

func TestBlock_VerifyValidatorSig_WrongKey(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	blk := &Block{Header: BlockHeader{Version: ProtocolVersion, Height: 5}}
	_ = blk.SignHeader(w1.PrivateKey)
	if err := blk.VerifyValidatorSig(w2.PublicKey); err == nil {
		t.Fatal("verification with the wrong public key should fail")
	}
}

func TestNewBlock_EmptyTxs(t *testing.T) {
	var prevHash, stateRoot [crypto.HashSize]byte
	var valAddr [crypto.AddressSize]byte
	blk, err := NewBlock(1, prevHash, stateRoot, time.Now().UnixNano(), valAddr, nil, 0, InitialBaseFee, 0)
	if err != nil {
		t.Fatalf("NewBlock: %v", err)
	}
	if blk.Header.Height != 1 {
		t.Errorf("height: want 1, got %d", blk.Header.Height)
	}
	var zeroHash [crypto.HashSize]byte
	if blk.Hash == zeroHash {
		t.Fatal("block hash should not be zero")
	}
}

func TestNewBlock_WithTransactions(t *testing.T) {
	w, _ := crypto.NewWallet()
	tx := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)

	var prevHash, stateRoot [crypto.HashSize]byte
	blk, err := NewBlock(1, prevHash, stateRoot, time.Now().UnixNano(), w.Address, []*Transaction{tx}, GasTransfer, InitialBaseFee, 0)
	if err != nil {
		t.Fatalf("NewBlock: %v", err)
	}
	if len(blk.Txs) != 1 {
		t.Errorf("want 1 tx, got %d", len(blk.Txs))
	}
	var emptyRoot [crypto.HashSize]byte
	if blk.Header.MerkleRoot == emptyRoot {
		t.Fatal("merkle root should not be zero with transactions")
	}
}

func TestNewBlock_MerkleRootChangesWithTxs(t *testing.T) {
	tx1 := newSignedTransfer(t, testWallet, testWallet2.Address, 0, 1)
	tx2 := newSignedTransfer(t, testWallet, testWallet2.Address, 1, 2)

	var prevHash, stateRoot [crypto.HashSize]byte
	var valAddr [crypto.AddressSize]byte
	ts := time.Now().UnixNano()

	blk1, _ := NewBlock(1, prevHash, stateRoot, ts, valAddr, []*Transaction{tx1}, GasTransfer, InitialBaseFee, 0)
	blk2, _ := NewBlock(1, prevHash, stateRoot, ts, valAddr, []*Transaction{tx1, tx2}, GasTransfer*2, InitialBaseFee, 0)
	if blk1.Header.MerkleRoot == blk2.Header.MerkleRoot {
		t.Fatal("different tx sets should produce different merkle roots")
	}
}
