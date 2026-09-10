package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"golang.org/x/crypto/sha3"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

var kemScheme = mlkem768.Scheme()

// InitiatorHandshake performs the ML-KEM-768 handshake from the dialing side.
// Returns a SecureConn on success.
func InitiatorHandshake(conn net.Conn, local *Identity, remoteInfo *PeerInfo) (*SecureConn, error) {
	// 1. Send Hello.
	hello := HelloPayload{
		NodeID:     local.NodeID,
		PublicKey:  local.PublicKey,
		ListenAddr: local.ListenAddr,
		Version:    1,
	}
	if err := sendGob(conn, MsgHello, hello); err != nil {
		return nil, fmt.Errorf("hello send: %w", err)
	}

	// 2. Receive HelloResp.
	var helloResp HelloRespPayload
	if err := recvGob(conn, MsgHelloResp, &helloResp); err != nil {
		return nil, fmt.Errorf("hello resp: %w", err)
	}
	if !helloResp.Accepted {
		return nil, errors.New("handshake rejected by responder")
	}

	if len(remoteInfo.PublicKey) == 0 {
		remoteInfo.PublicKey = helloResp.PublicKey
		remoteInfo.NodeID = helloResp.NodeID
	}

	// 3. KEM encapsulates using the responder's long-term public key.
	seed := sha3.Sum512(remoteInfo.NodeID[:])
	respPubKEM, _ := kemScheme.DeriveKeyPair(seed[:kemScheme.SeedSize()])
	if len(remoteInfo.PublicKey) >= kemScheme.PublicKeySize() {
		if pub, err := kemScheme.UnmarshalBinaryPublicKey(remoteInfo.PublicKey[:kemScheme.PublicKeySize()]); err == nil {
			respPubKEM = pub
		}
	}
	ct, ss, err := kemScheme.Encapsulate(respPubKEM)
	if err != nil {
		return nil, fmt.Errorf("KEM encapsulate: %w", err)
	}

	// 4. Sign transcript: SHA-3(ct ‖ respNodeID).
	transcript := crypto.HashMany(ct, remoteInfo.NodeID[:])
	sig, err := crypto.Sign(local.PrivateKey, transcript[:])
	if err != nil {
		return nil, fmt.Errorf("sign transcript: %w", err)
	}

	kemInit := KEMInitPayload{
		SenderNodeID: local.NodeID,
		Ciphertext:   ct,
		Signature:    sig,
	}
	if err := sendGob(conn, MsgKEMInit, kemInit); err != nil {
		return nil, fmt.Errorf("KEMInit send: %w", err)
	}

	// 5. Receive KEMDone (responder's transcript signature).
	var kemDone KEMDonePayload
	if err := recvGob(conn, MsgKEMDone, &kemDone); err != nil {
		return nil, fmt.Errorf("KEMDone recv: %w", err)
	}

	// 6. Verify responder's transcript signature: SHA-3(ss[:16] ‖ initNodeID).
	respTranscript := crypto.HashMany(ss[:16], local.NodeID[:])
	ok, err := crypto.Verify(remoteInfo.PublicKey, respTranscript[:], kemDone.Signature)
	if err != nil || !ok {
		return nil, errors.New("responder transcript verification failed")
	}

	// 7. Derive a session key.
	sessionKey := deriveSessionKey(ss, local.NodeID, remoteInfo.NodeID)
	return NewSecureConn(conn, sessionKey)
}

// ResponderHandshake performs the ML-KEM-768 handshake from the listening side.
func ResponderHandshake(conn net.Conn, local *Identity) (*SecureConn, *PeerInfo, error) {
	// 1. Receive Hello.
	var hello HelloPayload
	if err := recvGob(conn, MsgHello, &hello); err != nil {
		return nil, nil, fmt.Errorf("hello recv: %w", err)
	}

	// 2. Send HelloResp.
	helloResp := HelloRespPayload{
		NodeID:    local.NodeID,
		PublicKey: local.PublicKey,
		Accepted:  true,
	}
	if err := sendGob(conn, MsgHelloResp, helloResp); err != nil {
		return nil, nil, fmt.Errorf("hello resp send: %w", err)
	}

	// 3. Receive KEMInit.
	var kemInit KEMInitPayload
	if err := recvGob(conn, MsgKEMInit, &kemInit); err != nil {
		return nil, nil, fmt.Errorf("KEMInit recv: %w", err)
	}

	// 4. Verify initiator's transcript signature.
	transcript := crypto.HashMany(kemInit.Ciphertext, local.NodeID[:])
	ok, err := crypto.Verify(hello.PublicKey, transcript[:], kemInit.Signature)
	if err != nil || !ok {
		return nil, nil, errors.New("initiator transcript verification failed")
	}

	// 5. KEM decapsulates using our private key.
	seed := sha3.Sum512(local.NodeID[:])
	_, privKEM := kemScheme.DeriveKeyPair(seed[:kemScheme.SeedSize()])
	if len(local.PrivateKey) >= kemScheme.PrivateKeySize() {
		if priv, err := kemScheme.UnmarshalBinaryPrivateKey(local.PrivateKey[:kemScheme.PrivateKeySize()]); err == nil {
			privKEM = priv
		}
	}
	ss, err := kemScheme.Decapsulate(privKEM, kemInit.Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("KEM decapsulate: %w", err)
	}

	// 6. Sign our own transcript: SHA-3(ss[:16] ‖ initNodeID).
	respTranscript := crypto.HashMany(ss[:16], kemInit.SenderNodeID[:])
	sig, err := crypto.Sign(local.PrivateKey, respTranscript[:])
	if err != nil {
		return nil, nil, fmt.Errorf("sign resp transcript: %w", err)
	}

	kemDone := KEMDonePayload{
		ReceiverNodeID: local.NodeID,
		Signature:      sig,
	}
	if err := sendGob(conn, MsgKEMDone, kemDone); err != nil {
		return nil, nil, fmt.Errorf("KEMDone send: %w", err)
	}

	// 7. Derive a session key.
	sessionKey := deriveSessionKey(ss, kemInit.SenderNodeID, local.NodeID)
	sc, err := NewSecureConn(conn, sessionKey)
	if err != nil {
		return nil, nil, err
	}

	peerInfo := &PeerInfo{
		NodeID:     hello.NodeID,
		PublicKey:  hello.PublicKey,
		ListenAddr: hello.ListenAddr,
	}
	return sc, peerInfo, nil
}

// deriveSessionKey = SHA-3-256(ss ‖ sorted(nodeA, nodeB)).
func deriveSessionKey(ss []byte, a, b [crypto.AddressSize]byte) [32]byte {
	var lo, hi [crypto.AddressSize]byte
	if addrLess(a, b) {
		lo, hi = a, b
	} else {
		lo, hi = b, a
	}
	h := crypto.HashMany(ss, lo[:], hi[:])
	return h
}

// addrLess returns true if a < b lexicographically.
func addrLess(a, b [crypto.AddressSize]byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// ── Low-level gob framing over plain TCP (used only during handshake) ────────

func sendGob(conn net.Conn, mt MsgType, v interface{}) error {
	payload, err := gobEncode(v)
	if err != nil {
		return err
	}
	frame := make([]byte, 1+len(payload))
	frame[0] = byte(mt)
	copy(frame[1:], payload)
	return writeRawFrame(conn, frame)
}

func recvGob(conn net.Conn, expected MsgType, v interface{}) error {
	frame, err := readRawFrame(conn)
	if err != nil {
		return err
	}
	if len(frame) < 1 {
		return errors.New("empty frame")
	}
	if MsgType(frame[0]) != expected {
		return fmt.Errorf("expected msg type 0x%02x, got 0x%02x", expected, frame[0])
	}
	return gobDecode(frame[1:], v)
}

// ensure binary import is used (for binary.BigEndian in nonce12 via secure.go)
var _ = binary.BigEndian
