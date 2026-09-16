package p2p

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── Identity tests ────────────────────────────────────────────────────────────

func newTestWallet(t *testing.T) *crypto.Wallet {
	t.Helper()
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	return w
}

func TestNewIdentity_NodeIDDerived(t *testing.T) {
	w := newTestWallet(t)
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:7777")
	expected := crypto.DeriveAddress(w.PublicKey)
	if id.NodeID != expected {
		t.Errorf("NodeID mismatch: want %x, got %x", expected, id.NodeID)
	}
}

func TestNewIdentity_FieldsPreserved(t *testing.T) {
	w := newTestWallet(t)
	addr := "0.0.0.0:9000"
	id := NewIdentity(w.PublicKey, w.PrivateKey, addr)

	if !bytes.Equal(id.PublicKey, w.PublicKey) {
		t.Error("PublicKey mismatch")
	}
	if !bytes.Equal(id.PrivateKey, w.PrivateKey) {
		t.Error("PrivateKey mismatch")
	}
	if id.ListenAddr != addr {
		t.Errorf("ListenAddr: want %s, got %s", addr, id.ListenAddr)
	}
}

func TestNewIdentity_EmptyListenAddr(t *testing.T) {
	w := newTestWallet(t)
	id := NewIdentity(w.PublicKey, w.PrivateKey, "")
	if id.ListenAddr != "" {
		t.Errorf("expected empty ListenAddr, got %s", id.ListenAddr)
	}
}

// ── Message framing tests ─────────────────────────────────────────────────────

func TestGobEncodeDecode_HelloPayload(t *testing.T) {
	w := newTestWallet(t)
	original := HelloPayload{
		NodeID:     crypto.DeriveAddress(w.PublicKey),
		PublicKey:  w.PublicKey,
		ListenAddr: "192.168.1.1:8080",
		Version:    1,
	}

	data, err := gobEncode(original)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("gobEncode returned empty bytes")
	}

	var decoded HelloPayload
	if err := gobDecode(data, &decoded); err != nil {
		t.Fatalf("gobDecode: %v", err)
	}

	if decoded.NodeID != original.NodeID {
		t.Errorf("NodeID mismatch after encode/decode")
	}
	if !bytes.Equal(decoded.PublicKey, original.PublicKey) {
		t.Error("PublicKey mismatch after encode/decode")
	}
	if decoded.ListenAddr != original.ListenAddr {
		t.Errorf("ListenAddr: want %s, got %s", original.ListenAddr, decoded.ListenAddr)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version: want %d, got %d", original.Version, decoded.Version)
	}
}

func TestGobEncodeDecode_HelloRespPayload(t *testing.T) {
	w := newTestWallet(t)
	original := HelloRespPayload{
		NodeID:    crypto.DeriveAddress(w.PublicKey),
		PublicKey: w.PublicKey,
		Accepted:  true,
	}

	data, err := gobEncode(original)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}

	var decoded HelloRespPayload
	if err := gobDecode(data, &decoded); err != nil {
		t.Fatalf("gobDecode: %v", err)
	}

	if decoded.Accepted != original.Accepted {
		t.Errorf("Accepted: want %v, got %v", original.Accepted, decoded.Accepted)
	}
	if decoded.NodeID != original.NodeID {
		t.Error("NodeID mismatch")
	}
}

func TestGobEncodeDecode_KEMInitPayload(t *testing.T) {
	w := newTestWallet(t)
	original := KEMInitPayload{
		SenderNodeID: crypto.DeriveAddress(w.PublicKey),
		Ciphertext:   []byte("mock_ciphertext_bytes"),
		Signature:    []byte("mock_signature_bytes"),
	}

	data, err := gobEncode(original)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}

	var decoded KEMInitPayload
	if err := gobDecode(data, &decoded); err != nil {
		t.Fatalf("gobDecode: %v", err)
	}

	if decoded.SenderNodeID != original.SenderNodeID {
		t.Error("SenderNodeID mismatch")
	}
	if !bytes.Equal(decoded.Ciphertext, original.Ciphertext) {
		t.Error("Ciphertext mismatch")
	}
	if !bytes.Equal(decoded.Signature, original.Signature) {
		t.Error("Signature mismatch")
	}
}

func TestGobEncodeDecode_GetBlocksPayload(t *testing.T) {
	original := GetBlocksPayload{FromHeight: 42, MaxCount: 100}
	data, err := gobEncode(original)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}

	var decoded GetBlocksPayload
	if err := gobDecode(data, &decoded); err != nil {
		t.Fatalf("gobDecode: %v", err)
	}

	if decoded.FromHeight != original.FromHeight {
		t.Errorf("FromHeight: want %d, got %d", original.FromHeight, decoded.FromHeight)
	}
	if decoded.MaxCount != original.MaxCount {
		t.Errorf("MaxCount: want %d, got %d", original.MaxCount, decoded.MaxCount)
	}
}

// ── GossipMsg tests ───────────────────────────────────────────────────────────

func TestNewTxGossip_Fields(t *testing.T) {
	w := newTestWallet(t)
	w2, _ := crypto.NewWallet()

	tx := &core.Transaction{
		Version:   1,
		Type:      core.TxTransfer,
		From:      w.Address,
		To:        w2.Address,
		Amount:    100,
		GasLimit:  core.GasTransfer,
		GasPrice:  core.MinGasPrice,
		Timestamp: 1_000_000,
		PublicKey: w.PublicKey,
		Signature: make([]byte, crypto.SignatureSize),
	}
	tx.Hash = tx.ComputeHash()

	msg, err := NewTxGossip(tx)
	if err != nil {
		t.Fatalf("NewTxGossip: %v", err)
	}
	if msg.MsgID != tx.Hash {
		t.Error("MsgID should equal transaction hash")
	}
	if msg.Type != MsgTx {
		t.Errorf("Type: want MsgTx (0x%02x), got 0x%02x", MsgTx, msg.Type)
	}
	if msg.Hops != 0 {
		t.Errorf("Hops: want 0, got %d", msg.Hops)
	}
	if len(msg.Data) == 0 {
		t.Error("Data must not be empty")
	}
}

func TestNewTxGossip_DataDecodesBackToTx(t *testing.T) {
	w, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	tx := &core.Transaction{
		Version:   1,
		Type:      core.TxTransfer,
		From:      w.Address,
		To:        w2.Address,
		Amount:    999,
		GasLimit:  core.GasTransfer,
		GasPrice:  core.MinGasPrice,
		Timestamp: 9_999_999,
		PublicKey: w.PublicKey,
		Signature: make([]byte, crypto.SignatureSize),
	}
	tx.Hash = tx.ComputeHash()

	msg, err := NewTxGossip(tx)
	if err != nil {
		t.Fatalf("NewTxGossip: %v", err)
	}

	var recovered core.Transaction
	if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&recovered); err != nil {
		t.Fatalf("decode gossip data: %v", err)
	}
	if recovered.Amount != tx.Amount {
		t.Errorf("Amount: want %d, got %d", tx.Amount, recovered.Amount)
	}
}

// ── addrLess helper tests ─────────────────────────────────────────────────────

func TestAddrLess_Symmetric(t *testing.T) {
	var a, b [crypto.AddressSize]byte
	a[0] = 1
	b[0] = 2
	if !addrLess(a, b) {
		t.Error("a[0]=1 should be less than b[0]=2")
	}
	if addrLess(b, a) {
		t.Error("b[0]=2 should not be less than a[0]=1")
	}
}

func TestAddrLess_Equal(t *testing.T) {
	var a [crypto.AddressSize]byte
	a[5] = 0x42
	b := a
	if addrLess(a, b) {
		t.Error("equal addresses: addrLess should return false")
	}
}

func TestDeriveSessionKey_Commutative(t *testing.T) {
	ss := []byte("shared_secret_bytes_here")
	var a, b [crypto.AddressSize]byte
	a[0] = 0xAB
	b[0] = 0xCD

	k1 := deriveSessionKey(ss, a, b)
	k2 := deriveSessionKey(ss, b, a)
	if k1 != k2 {
		t.Error("deriveSessionKey must be commutative (same key regardless of node order)")
	}
}

// ── MsgType constants sanity checks ──────────────────────────────────────────

func TestMsgTypeConstants(t *testing.T) {
	// Ensure no two msg types share the same byte value
	types := map[MsgType]string{
		MsgHello:     "MsgHello",
		MsgHelloResp: "MsgHelloResp",
		MsgKEMInit:   "MsgKEMInit",
		MsgKEMDone:   "MsgKEMDone",
		MsgTx:        "MsgTx",
		MsgBlock:     "MsgBlock",
		MsgGetBlocks: "MsgGetBlocks",
		MsgBlocks:    "MsgBlocks",
		MsgPeerList:  "MsgPeerList",
		MsgPing:      "MsgPing",
		MsgPong:      "MsgPong",
	}
	if len(types) != 11 {
		t.Errorf("expected 11 unique msg types, got %d (duplicate detected)", len(types))
	}
}
