package consensus

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

var testVoteWallet *crypto.Wallet

func init() {
	var err error
	testVoteWallet, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
}

func makeVote(w *crypto.Wallet, voteType uint8, height uint64, blockHash [crypto.HashSize]byte) *Vote {
	return &Vote{
		Type:      voteType,
		Height:    height,
		Round:     0,
		BlockHash: blockHash,
		Voter:     w.Address,
		PublicKey: w.PublicKey,
	}
}

func TestVote_Sign_Verify(t *testing.T) {
	blockHash := crypto.Hash256([]byte("block"))
	v := makeVote(testVoteWallet, VotePrevote, 1, blockHash)
	if err := v.Sign(testVoteWallet.PrivateKey); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := v.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVote_Verify_TamperedHeight(t *testing.T) {
	blockHash := crypto.Hash256([]byte("block"))
	v := makeVote(testVoteWallet, VotePrevote, 1, blockHash)
	_ = v.Sign(testVoteWallet.PrivateKey)
	v.Height = 99 // tamper after signing
	if err := v.Verify(); err == nil {
		t.Fatal("tampered height should fail Verify")
	}
}

func TestVote_Verify_TamperedBlockHash(t *testing.T) {
	blockHash := crypto.Hash256([]byte("block"))
	v := makeVote(testVoteWallet, VotePrecommit, 5, blockHash)
	_ = v.Sign(testVoteWallet.PrivateKey)
	v.BlockHash = crypto.Hash256([]byte("different"))
	if err := v.Verify(); err == nil {
		t.Fatal("tampered block hash should fail Verify")
	}
}

func TestVote_Verify_WrongPublicKey(t *testing.T) {
	other, _ := crypto.NewWallet()
	blockHash := crypto.Hash256([]byte("block"))
	v := makeVote(testVoteWallet, VotePrevote, 1, blockHash)
	_ = v.Sign(testVoteWallet.PrivateKey)
	v.PublicKey = other.PublicKey // swap key
	if err := v.Verify(); err == nil {
		t.Fatal("the wrong public key should fail Verify")
	}
}

func TestVote_Verify_MissingSignature(t *testing.T) {
	blockHash := crypto.Hash256([]byte("block"))
	v := makeVote(testVoteWallet, VotePrevote, 1, blockHash)
	// Do not sign — Signature is nil / wrong length.
	if err := v.Verify(); err == nil {
		t.Fatal("unsigned vote should fail Verify")
	}
}

func TestVote_Prevote_And_Precommit_DifferentSignatures(t *testing.T) {
	blockHash := crypto.Hash256([]byte("block"))
	prevote := makeVote(testVoteWallet, VotePrevote, 1, blockHash)
	precommit := makeVote(testVoteWallet, VotePrecommit, 1, blockHash)
	_ = prevote.Sign(testVoteWallet.PrivateKey)
	_ = precommit.Sign(testVoteWallet.PrivateKey)
	// Both should verify successfully.
	if err := prevote.Verify(); err != nil {
		t.Fatalf("prevote Verify: %v", err)
	}
	if err := precommit.Verify(); err != nil {
		t.Fatalf("precommit Verify: %v", err)
	}
	// Signatures should differ (different type byte in payload).
	if string(prevote.Signature) == string(precommit.Signature) {
		t.Fatal("prevote and precommit should produce different signatures")
	}
}

func TestVote_DifferentHeights_DifferentSignatures(t *testing.T) {
	bh := crypto.Hash256([]byte("block"))
	v1 := makeVote(testVoteWallet, VotePrevote, 1, bh)
	v2 := makeVote(testVoteWallet, VotePrevote, 2, bh)
	_ = v1.Sign(testVoteWallet.PrivateKey)
	_ = v2.Sign(testVoteWallet.PrivateKey)
	if string(v1.Signature) == string(v2.Signature) {
		t.Fatal("votes at different heights should produce different signatures")
	}
}
