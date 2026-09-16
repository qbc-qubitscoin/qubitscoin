package p2p

import (
	"bytes"
	"context"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cloudflare/circl/kem"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"golang.org/x/crypto/sha3"
)

func TestGossipCache_Full(t *testing.T) {
	// 1. Default capacity
	gcDef := NewGossipCache(0)
	if gcDef.cap != gossipCacheSize {
		t.Fatalf("expected default cap %d, got %d", gossipCacheSize, gcDef.cap)
	}

	// 2. Custom capacity and eviction
	gc := NewGossipCache(4)
	var ids [5][crypto.HashSize]byte
	for i := 0; i < 5; i++ {
		ids[i][0] = byte(i + 1)
	}

	// Add 4 items
	for i := 0; i < 4; i++ {
		if !gc.MarkSeen(ids[i]) {
			t.Fatalf("item %d should be new", i)
		}
		// Second time should return false
		if gc.MarkSeen(ids[i]) {
			t.Fatalf("item %d should be seen", i)
		}
	}

	// 5th item triggers eviction of oldest half (items 0 and 1)
	if !gc.MarkSeen(ids[4]) {
		t.Fatal("5th item should be new")
	}

	// Items 0 and 1 were evicted, so marking them seen now returns true
	if !gc.MarkSeen(ids[0]) {
		t.Fatal("evicted item 0 should be treated as new")
	}
	// Item 2 was NOT evicted
	if gc.MarkSeen(ids[2]) {
		t.Fatal("retained item 2 should be seen")
	}
}

func TestSecureConn_Lifecycle(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	var key [32]byte
	for i := range key {
		key[i] = byte(i + 1)
	}

	sc1, err := NewSecureConn(c1, key)
	if err != nil {
		t.Fatal(err)
	}
	sc2, err := NewSecureConn(c2, key)
	if err != nil {
		t.Fatal(err)
	}

	// SetReadDeadline
	if err := sc1.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	// RemoteAddr
	_ = sc1.RemoteAddr()

	// Send / Recv
	msg := []byte("secret-quantum-message")
	go func() {
		_ = sc1.Send(msg)
	}()

	recv, err := sc2.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if !bytes.Equal(recv, msg) {
		t.Fatalf("mismatch: got %s, want %s", recv, msg)
	}

	// Tampered message
	go func() {
		_ = writeRawFrame(c1, []byte("corrupt-invalid-ciphertext-bytes-here"))
	}()
	_, err = sc2.Recv()
	if err == nil {
		t.Fatal("expected decryption error on corrupt frame")
	}

	// Close
	_ = sc1.Close()
	_ = sc2.Close()

	// Recv on closed
	_, err = sc2.Recv()
	if err == nil {
		t.Fatal("expected error reading from closed connection")
	}
}

func TestHandshake_HelpersAndEdgeCases(t *testing.T) {
	// addrLess
	var a, b [crypto.AddressSize]byte
	a[0] = 1
	b[0] = 2
	if !addrLess(a, b) {
		t.Fatal("expected a < b")
	}
	if addrLess(b, a) {
		t.Fatal("expected b not < a")
	}
	if addrLess(a, a) {
		t.Fatal("expected a not < a")
	}

	// deriveSessionKey determinism
	var ss = []byte("shared-secret-bytes-16")
	k1 := deriveSessionKey(ss, a, b)
	k2 := deriveSessionKey(ss, b, a)
	if k1 != k2 {
		t.Fatal("session keys should match regardless of node order")
	}

	// sendGob / recvGob errors
	c1, c2 := net.Pipe()
	_ = c1.Close()
	_ = c2.Close()

	if err := sendGob(c1, MsgPing, "data"); err == nil {
		t.Fatal("expected sendGob to fail on closed pipe")
	}
	var res string
	if err := recvGob(c2, MsgPing, &res); err == nil {
		t.Fatal("expected recvGob to fail on closed pipe")
	}

	// recvGob empty frame
	p1, p2 := net.Pipe()
	go func() {
		_ = writeRawFrame(p1, []byte{})
	}()
	if err := recvGob(p2, MsgPing, &res); err == nil {
		t.Fatal("expected error on empty frame")
	}
	_ = p1.Close()
	_ = p2.Close()

	// recvGob unexpected message type
	p3, p4 := net.Pipe()
	go func() {
		_ = writeRawFrame(p3, []byte{byte(MsgPong), 1, 2, 3})
	}()
	if err := recvGob(p4, MsgPing, &res); err == nil {
		t.Fatal("expected error on mismatched message type")
	}
	_ = p3.Close()
	_ = p4.Close()
}

func TestHandshake_FailureModes(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:9101")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:9102")

	// 1. Responder rejects handshake
	c1, c2 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c2, MsgHello, &hello)
		_ = sendGob(c2, MsgHelloResp, HelloRespPayload{
			NodeID:   id2.NodeID,
			Accepted: false,
		})
	}()

	remote := &PeerInfo{ListenAddr: id2.ListenAddr}
	_, err := InitiatorHandshake(c1, id1, remote)
	if err == nil {
		t.Fatal("expected error when responder rejects handshake")
	}
	_ = c1.Close()
	_ = c2.Close()

	// 2. Initiator sends invalid signature in KEMInit
	c3, c4 := net.Pipe()
	go func() {
		_ = sendGob(c3, MsgHello, HelloPayload{
			NodeID:    id1.NodeID,
			PublicKey: id1.PublicKey,
		})
		var resp HelloRespPayload
		_ = recvGob(c3, MsgHelloResp, &resp)
		_ = sendGob(c3, MsgKEMInit, KEMInitPayload{
			SenderNodeID: id1.NodeID,
			Ciphertext:   make([]byte, 1088),
			Signature:    make([]byte, crypto.SignatureSize),
		})
	}()

	_, _, err = ResponderHandshake(c4, id2)
	if err == nil {
		t.Fatal("expected error on invalid initiator signature")
	}
	_ = c3.Close()
	_ = c4.Close()

	// 3. Responder sends corrupted signature in KEMDone
	c5, c6 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c6, MsgHello, &hello)
		_ = sendGob(c6, MsgHelloResp, HelloRespPayload{
			NodeID:    id2.NodeID,
			PublicKey: id2.PublicKey,
			Accepted:  true,
		})
		var kemInit KEMInitPayload
		_ = recvGob(c6, MsgKEMInit, &kemInit)
		_ = sendGob(c6, MsgKEMDone, KEMDonePayload{
			ReceiverNodeID: id2.NodeID,
			Signature:      make([]byte, crypto.SignatureSize),
		})
	}()

	remote2 := &PeerInfo{ListenAddr: id2.ListenAddr}
	_, err = InitiatorHandshake(c5, id1, remote2)
	if err == nil {
		t.Fatal("expected error on corrupted responder signature in KEMDone")
	}
	_ = c5.Close()
	_ = c6.Close()
}

func TestPeer_RunAndClose(t *testing.T) {
	c1, c2 := net.Pipe()
	var key [32]byte
	key[0] = 77
	sc1, _ := NewSecureConn(c1, key)
	sc2, _ := NewSecureConn(c2, key)

	peerInfo := &PeerInfo{ListenAddr: "127.0.0.1:9200"}
	peer := newPeer(peerInfo, sc1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan []byte, 1)
	go peer.run(ctx, func(msg []byte) {
		received <- msg
	})

	// Send message to peer from sc2
	msg := []byte("hello-peer")
	_ = sc2.Send(msg)

	select {
	case got := <-received:
		if !bytes.Equal(got, msg) {
			t.Fatalf("got %s, want %s", got, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for peer message")
	}

	// Test peer Send with concurrent reader
	go func() {
		_, _ = sc2.Recv()
	}()
	_ = peer.Send([]byte("peer-response"))

	// Close context and connection to terminate peer.run cleanly
	cancel()
	peer.Close()
	_ = sc2.Close()
	select {
	case <-peer.quit:
		// quit closed
	case <-time.After(time.Second):
		t.Fatal("quit channel not closed")
	}
}

func TestNode_FullNetwork_Lifecycle(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	// Invalid listen address
	badID := NewIdentity(w1.PublicKey, w1.PrivateKey, "999.999.999.999:1")
	if _, err := NewNode(badID); err == nil {
		t.Fatal("expected NewNode to fail with invalid address")
	}

	node1, err := NewNode(id1)
	if err != nil {
		t.Fatal(err)
	}
	node2, err := NewNode(id2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var connected1, connected2 sync.WaitGroup
	connected1.Add(1)
	connected2.Add(1)

	node1.OnPeerConnect = func(pi *PeerInfo) {
		connected1.Done()
	}
	node2.OnPeerConnect = func(pi *PeerInfo) {
		connected2.Done()
	}

	node1.Start(ctx)
	node2.Start(ctx)

	// Connect node1 to node2
	if err := node1.Connect(ctx, id2.ListenAddr); err != nil {
		t.Fatalf("failed to connect node1 -> node2: %v", err)
	}

	// Wait for connection to establish on both sides
	connected1.Wait()
	connected2.Wait()

	if node1.PeerCount() != 1 {
		t.Fatalf("expected node1 PeerCount 1, got %d", node1.PeerCount())
	}
	if node2.PeerCount() != 1 {
		t.Fatalf("expected node2 PeerCount 1, got %d", node2.PeerCount())
	}

	// Duplicate connect (should be rejected in addPeer)
	_ = node1.Connect(ctx, id2.ListenAddr)

	// Connect to non-existent address
	if err := node1.Connect(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("expected connect to closed port to fail")
	}

	// BroadcastRaw
	node1.BroadcastRaw([]byte("raw-sync-frame"))

	// Test Transaction Gossip
	txReceived := make(chan *core.Transaction, 1)
	node2.OnTxReceived = func(tx *core.Transaction) {
		txReceived <- tx
	}

	tx := &core.Transaction{
		Type:   core.TxTransfer,
		From:   w1.Address,
		To:     w2.Address,
		Amount: 1000,
		Nonce:  1,
	}
	tx.Hash = tx.ComputeHash()

	node1.BroadcastTx(tx)

	select {
	case gotTx := <-txReceived:
		if gotTx.Hash != tx.Hash {
			t.Fatalf("tx hash mismatch: got %v, want %v", gotTx.Hash, tx.Hash)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for gossiped tx")
	}

	// Re-broadcast same tx (should be deduped by gossip cache)
	node1.BroadcastTx(tx)

	// Test Block Gossip
	blkReceived := make(chan *core.Block, 1)
	node2.OnBlockReceived = func(blk *core.Block) {
		blkReceived <- blk
	}

	var dummyHash [crypto.HashSize]byte
	blk, err := core.NewBlock(1, dummyHash, dummyHash, time.Now().UnixNano(), w1.Address, []*core.Transaction{}, 0, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = blk.SignHeader(w1.PrivateKey)

	node1.BroadcastBlock(blk)

	select {
	case gotBlk := <-blkReceived:
		if gotBlk.Hash != blk.Hash {
			t.Fatalf("block hash mismatch: got %v, want %v", gotBlk.Hash, blk.Hash)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for gossiped block")
	}

	// Re-broadcast same block (should be deduped)
	node1.BroadcastBlock(blk)

	// Dispatch edge cases
	p := node1.peers[crypto.ToHex(id2.NodeID)]
	if p != nil {
		// Empty frame
		node1.dispatch(p, []byte{})

		// MsgTx with bad gob
		node1.dispatch(p, []byte{byte(MsgTx), 0xff, 0xff})

		// MsgBlock with bad gob
		node1.dispatch(p, []byte{byte(MsgBlock), 0xff, 0xff})

		// MsgPing -> sends pong
		node1.dispatch(p, []byte{byte(MsgPing), 0, 0, 0, 0, 0, 0, 0, 0})

		// MsgPong -> no-op
		node1.dispatch(p, []byte{byte(MsgPong), 0, 0, 0, 0, 0, 0, 0, 0})

		// Unknown message type
		node1.dispatch(p, []byte{0xfe, 1, 2, 3})

		// MsgPeerList with self NodeID (ignored)
		selfPL := PeerListPayload{Peers: []PeerInfo{{NodeID: id1.NodeID}}}
		selfPLData, _ := gobEncode(selfPL)
		node1.dispatch(p, append([]byte{byte(MsgPeerList)}, selfPLData...))

		// MsgPeerList bad gob
		node1.dispatch(p, []byte{byte(MsgPeerList), 0xff})
	}

	// Close context to terminate servers
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestWireFraming_Errors(t *testing.T) {
	c1, c2 := net.Pipe()
	_ = c1.Close()
	_ = c2.Close()

	// writeRawFrame on closed pipe
	if err := writeRawFrame(c1, []byte("data")); err == nil {
		t.Fatal("expected writeRawFrame to fail on closed pipe")
	}

	// readRawFrame on closed pipe (header read error)
	if _, err := readRawFrame(c2); err == nil {
		t.Fatal("expected readRawFrame to fail on closed pipe")
	}

	// Truncated payload read error
	p1, p2 := net.Pipe()
	go func() {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], 100)
		_, _ = p1.Write(hdr[:])
		_, _ = p1.Write([]byte("short"))
		_ = p1.Close()
	}()

	if _, err := readRawFrame(p2); err == nil {
		t.Fatal("expected readRawFrame to fail on truncated payload")
	}
	_ = p2.Close()
}

func TestNewSecureConn_Errors(t *testing.T) {
	origCipher := newCipherFunc
	origGCM := newGCMFunc
	defer func() {
		newCipherFunc = origCipher
		newGCMFunc = origGCM
	}()

	var key [32]byte
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	// 1. newCipher error
	newCipherFunc = func(key []byte) (cipher.Block, error) {
		return nil, errors.New("cipher error")
	}
	if _, err := NewSecureConn(c1, key); err == nil {
		t.Fatal("expected error from newCipherFunc")
	}

	// 2. newGCM error
	newCipherFunc = origCipher
	newGCMFunc = func(cipher cipher.Block) (cipher.AEAD, error) {
		return nil, errors.New("gcm error")
	}
	if _, err := NewSecureConn(c1, key); err == nil {
		t.Fatal("expected error from newGCMFunc")
	}
}

func TestGossipWrappers_Errors(t *testing.T) {
	origEncode := gobEncodeFunc
	defer func() { gobEncodeFunc = origEncode }()

	gobEncodeFunc = func(v interface{}) ([]byte, error) {
		return nil, errors.New("encode error")
	}

	tx := &core.Transaction{}
	if _, err := NewTxGossip(tx); err == nil {
		t.Fatal("expected NewTxGossip to fail when encode fails")
	}

	blk := &core.Block{}
	if _, err := NewBlockGossip(blk); err == nil {
		t.Fatal("expected NewBlockGossip to fail when encode fails")
	}
}

func TestHandshake_MoreErrors(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")

	// 1. InitiatorHandshake: invalid local.PrivateKey causes Sign to fail
	c1, c2 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c2, MsgHello, &hello)
		_ = sendGob(c2, MsgHelloResp, HelloRespPayload{
			NodeID:    id.NodeID,
			PublicKey: id.PublicKey,
			Accepted:  true,
		})
	}()

	badLocal := NewIdentity(w.PublicKey, []byte("short-invalid-priv-key"), "127.0.0.1:0")
	remote := &PeerInfo{ListenAddr: "127.0.0.1:0"}
	if _, err := InitiatorHandshake(c1, badLocal, remote); err == nil {
		t.Fatal("expected InitiatorHandshake to fail on invalid private key")
	}
	_ = c1.Close()
	_ = c2.Close()

	// 2. InitiatorHandshake: network closed during send of KEMInit
	c3, c4 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c4, MsgHello, &hello)
		_ = sendGob(c4, MsgHelloResp, HelloRespPayload{
			NodeID:    id.NodeID,
			PublicKey: id.PublicKey,
			Accepted:  true,
		})
		_ = c4.Close()
	}()

	remote2 := &PeerInfo{ListenAddr: "127.0.0.1:0"}
	_, _ = InitiatorHandshake(c3, id, remote2)
	_ = c3.Close()

	// 3. ResponderHandshake: closed pipe during Hello
	c5, c6 := net.Pipe()
	_ = c5.Close()
	_ = c6.Close()
	if _, _, err := ResponderHandshake(c5, id); err == nil {
		t.Fatal("expected ResponderHandshake to fail on closed pipe")
	}

	// 4. ResponderHandshake: closed pipe during HelloResp send
	c7, c8 := net.Pipe()
	go func() {
		_ = sendGob(c7, MsgHello, HelloPayload{
			NodeID:    id.NodeID,
			PublicKey: id.PublicKey,
		})
		_ = c7.Close()
	}()
	_, _, _ = ResponderHandshake(c8, id)
	_ = c8.Close()

	// 5. ResponderHandshake: invalid local.PrivateKey causes Sign to fail
	c9, c10 := net.Pipe()
	go func() {
		_ = sendGob(c9, MsgHello, HelloPayload{
			NodeID:    id.NodeID,
			PublicKey: id.PublicKey,
		})
		var resp HelloRespPayload
		_ = recvGob(c9, MsgHelloResp, &resp)
		pub, _, _ := kemScheme.GenerateKeyPair()
		ct, _, _ := kemScheme.Encapsulate(pub)
		transcript := crypto.HashMany(ct, id.NodeID[:])
		sig, _ := crypto.Sign(w.PrivateKey, transcript[:])
		_ = sendGob(c9, MsgKEMInit, KEMInitPayload{
			SenderNodeID: id.NodeID,
			Ciphertext:   ct,
			Signature:    sig,
		})
	}()

	badResponder := NewIdentity(w.PublicKey, []byte("short-invalid-priv-key"), "127.0.0.1:0")
	if _, _, err := ResponderHandshake(c10, badResponder); err == nil {
		t.Fatal("expected ResponderHandshake to fail on invalid private key")
	}
	_ = c9.Close()
	_ = c10.Close()
}

func TestNode_AdditionalBranches(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	n1, err := NewNode(id1)
	if err != nil {
		t.Fatal(err)
	}

	// 1. broadcast with max hops
	n1.broadcast(&GossipMsg{Hops: maxGossipHops})

	// 2. addPeer when max peers reached
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	var key [32]byte
	sc1, _ := NewSecureConn(c1, key)
	sc2, _ := NewSecureConn(c2, key)

	for i := 0; i < defaultMaxPeers; i++ {
		var dummyID [crypto.AddressSize]byte
		dummyID[0] = byte(i + 1)
		n1.peers[crypto.ToHex(dummyID)] = newPeer(&PeerInfo{NodeID: dummyID}, sc1)
	}

	extraPeer := newPeer(&PeerInfo{ListenAddr: "127.0.0.1:9999"}, sc2)
	n1.addPeer(context.Background(), extraPeer)

	// Clear dummy peers
	n1.peers = make(map[string]*Peer)

	// 3. sendPeerList when multiple peers exist
	p1 := newPeer(&PeerInfo{NodeID: id1.NodeID}, sc1)
	p2 := newPeer(&PeerInfo{NodeID: id2.NodeID}, sc2)
	n1.peers[crypto.ToHex(id1.NodeID)] = p1
	n1.peers[crypto.ToHex(id2.NodeID)] = p2

	go func() {
		_, _ = sc2.Recv()
	}()
	n1.sendPeerList(p1)

	// 4. removePeer and OnPeerDisconnect
	disconnected := false
	n1.OnPeerDisconnect = func(pi *PeerInfo) {
		disconnected = true
	}
	n1.removePeer(p1)
	if !disconnected {
		t.Fatal("expected OnPeerDisconnect to be called")
	}

	// 5. pingLoop
	origPing := pingInterval
	defer func() { pingInterval = origPing }()
	pingInterval = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()
	n1.pingLoop(ctx)
}

func TestP2P_RemainingBranches(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")

	// 1. InitiatorHandshake on closed connection at start
	cClosed1, cClosed2 := net.Pipe()
	_ = cClosed1.Close()
	_ = cClosed2.Close()
	remote := &PeerInfo{ListenAddr: "127.0.0.1:0"}
	if _, err := InitiatorHandshake(cClosed1, id, remote); err == nil {
		t.Fatal("expected error on closed pipe in InitiatorHandshake")
	}

	// 2. InitiatorHandshake closed during recv HelloResp
	c1, c2 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c2, MsgHello, &hello)
		_ = c2.Close()
	}()
	_, _ = InitiatorHandshake(c1, id, remote)
	_ = c1.Close()

	// 3. InitiatorHandshake closed during recv KEMDone
	c3, c4 := net.Pipe()
	go func() {
		var hello HelloPayload
		_ = recvGob(c4, MsgHello, &hello)
		_ = sendGob(c4, MsgHelloResp, HelloRespPayload{NodeID: id.NodeID, PublicKey: id.PublicKey, Accepted: true})
		var kemInit KEMInitPayload
		_ = recvGob(c4, MsgKEMInit, &kemInit)
		_ = c4.Close()
	}()
	_, _ = InitiatorHandshake(c3, id, remote)
	_ = c3.Close()

	// 4. ResponderHandshake closed during recv KEMInit
	c5, c6 := net.Pipe()
	go func() {
		_ = sendGob(c5, MsgHello, HelloPayload{NodeID: id.NodeID, PublicKey: id.PublicKey})
		var resp HelloRespPayload
		_ = recvGob(c5, MsgHelloResp, &resp)
		_ = c5.Close()
	}()
	_, _, _ = ResponderHandshake(c6, id)
	_ = c6.Close()

	// 5. ResponderHandshake closed during send KEMDone
	c7, c8 := net.Pipe()
	go func() {
		_ = sendGob(c7, MsgHello, HelloPayload{NodeID: id.NodeID, PublicKey: id.PublicKey})
		var resp HelloRespPayload
		_ = recvGob(c7, MsgHelloResp, &resp)
		seed := sha3.Sum512(id.NodeID[:])
		pub, _ := kemScheme.DeriveKeyPair(seed[:kemScheme.SeedSize()])
		ct, _, _ := kemScheme.Encapsulate(pub)
		tr := crypto.HashMany(ct, id.NodeID[:])
		sig, _ := crypto.Sign(w.PrivateKey, tr[:])
		_ = sendGob(c7, MsgKEMInit, KEMInitPayload{SenderNodeID: id.NodeID, Ciphertext: ct, Signature: sig})
		_ = c7.Close()
	}()
	_, _, _ = ResponderHandshake(c8, id)
	_ = c8.Close()

	// 6. Node Connect handshake failure
	srv, err := NewNode(id)
	if err != nil {
		t.Fatal(err)
	}
	lnFail, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lnFail.Close()
	go func() {
		conn, err := lnFail.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()
	_ = srv.Connect(context.Background(), lnFail.Addr().String())

	// 7. Server handleConn handshake failure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.Start(ctx)
	dialerConn, err := net.Dial("tcp", srv.server.listener.Addr().String())
	if err == nil {
		_ = dialerConn.Close()
	}
	time.Sleep(20 * time.Millisecond)

	// 8. Dispatch error cases: MsgTx / MsgBlock with corrupt payload
	cPipe1, cPipe2 := net.Pipe()
	defer cPipe1.Close()
	defer cPipe2.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			if _, err := cPipe2.Read(buf); err != nil {
				return
			}
		}
	}()
	sc, _ := NewSecureConn(cPipe1, [32]byte{1})
	p := newPeer(&PeerInfo{NodeID: id.NodeID}, sc)

	badTxGossip, _ := gobEncode(&GossipMsg{MsgID: [32]byte{10}, Type: MsgTx, Data: []byte("not-a-tx")})
	srv.dispatch(p, append([]byte{byte(MsgTx)}, badTxGossip...))

	badBlkGossip, _ := gobEncode(&GossipMsg{MsgID: [32]byte{11}, Type: MsgBlock, Data: []byte("not-a-blk")})
	srv.dispatch(p, append([]byte{byte(MsgBlock)}, badBlkGossip...))

	// MsgPeerList with unknown peer triggers go n.Connect
	unknownPL, _ := gobEncode(&PeerListPayload{Peers: []PeerInfo{{NodeID: [32]byte{99}, ListenAddr: "127.0.0.1:1"}}})
	srv.dispatch(p, append([]byte{byte(MsgPeerList)}, unknownPL...))

	// MsgPing on closed peer
	badSc, _ := NewSecureConn(cClosed1, [32]byte{2})
	badPeer := newPeer(&PeerInfo{}, badSc)
	srv.dispatch(badPeer, []byte{byte(MsgPing)})

	// BroadcastRaw and broadcast with error sending to a peer
	srv.peers["bad"] = badPeer
	srv.BroadcastRaw([]byte("raw"))
	srv.broadcast(&GossipMsg{Type: MsgTx})
	delete(srv.peers, "bad")

	// Broadcast and sendPeerList with gobEncode error
	origEncode := gobEncodeFunc
	defer func() { gobEncodeFunc = origEncode }()
	// sendGob with un-encodable type
	_ = sendGob(cPipe1, MsgHello, make(chan int))

	// sendPeerList with peer whose send fails
	srv.peers["bad"] = badPeer
	srv.sendPeerList(p)
	delete(srv.peers, "bad")
}

func TestHandshake_DecapsulateError(t *testing.T) {
	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	// Short ciphertext in KEMInit with valid ML-DSA signature
	c1, c2 := net.Pipe()
	go func() {
		_ = sendGob(c1, MsgHello, HelloPayload{NodeID: id1.NodeID, PublicKey: id1.PublicKey})
		var resp HelloRespPayload
		_ = recvGob(c1, MsgHelloResp, &resp)
		shortCT := []byte("too-short")
		tr := crypto.HashMany(shortCT, id2.NodeID[:])
		sig, _ := crypto.Sign(w1.PrivateKey, tr[:])
		_ = sendGob(c1, MsgKEMInit, KEMInitPayload{
			SenderNodeID: id1.NodeID,
			Ciphertext:   shortCT,
			Signature:    sig,
		})
	}()

	_, _, err := ResponderHandshake(c2, id2)
	if err == nil {
		t.Fatal("expected ResponderHandshake to fail on short ciphertext")
	}
	_ = c1.Close()
	_ = c2.Close()
}

func TestHandshake_NewSecureConnError(t *testing.T) {
	origCipher := newCipherFunc
	defer func() { newCipherFunc = origCipher }()

	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	newCipherFunc = func(key []byte) (cipher.Block, error) {
		return nil, errors.New("cipher-fail")
	}

	c1, c2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		remote := &PeerInfo{ListenAddr: "127.0.0.1:0"}
		_, _ = InitiatorHandshake(c1, id1, remote)
	}()

	go func() {
		defer wg.Done()
		_, _, _ = ResponderHandshake(c2, id2)
	}()

	wg.Wait()
	_ = c1.Close()
	_ = c2.Close()
}

func TestHandshake_ValidKEMBytes(t *testing.T) {
	pub, priv, _ := kemScheme.GenerateKeyPair()
	pubBytes, _ := pub.MarshalBinary()
	privBytes, _ := priv.MarshalBinary()

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()

	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	// Pre-populate valid KEM key bytes
	id2.PublicKey = pubBytes
	id2.PrivateKey = privBytes

	remote := &PeerInfo{ListenAddr: "127.0.0.1:0", PublicKey: pubBytes, NodeID: id2.NodeID}

	go func() {
		_, _ = InitiatorHandshake(c1, id1, remote)
	}()

	_, _, _ = ResponderHandshake(c2, id2)
}

func TestPeer_RunContextDone(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	sc, _ := NewSecureConn(c1, [32]byte{1})
	peer := newPeer(&PeerInfo{}, sc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before run starts

	peer.run(ctx, func(msg []byte) {})
}

func TestNode_PingLoop_Error(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	node, _ := NewNode(id)

	c1, c2 := net.Pipe()
	_ = c1.Close()
	_ = c2.Close()

	badSc, _ := NewSecureConn(c1, [32]byte{1})
	node.peers["bad"] = newPeer(&PeerInfo{}, badSc)

	origPing := pingInterval
	defer func() { pingInterval = origPing }()
	pingInterval = 2 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	node.pingLoop(ctx)
}

func TestServer_AcceptError_Branch(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	srv, _ := newTCPServer(id)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = srv.listener.Close()
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	srv.acceptLoop(ctx, func(p *Peer) {})
}

func TestNode_AddPeer_RemovalOnQuit(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	node, _ := NewNode(id)

	c1, c2 := net.Pipe()
	sc1, _ := NewSecureConn(c1, [32]byte{1})
	sc2, _ := NewSecureConn(c2, [32]byte{1})

	var removed sync.WaitGroup
	removed.Add(1)
	node.OnPeerDisconnect = func(pi *PeerInfo) {
		removed.Done()
	}

	p := newPeer(&PeerInfo{NodeID: id.NodeID}, sc1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node.addPeer(ctx, p)

	// Close peer connection to trigger removePeer via p.run exit
	_ = sc1.Close()
	_ = sc2.Close()

	removed.Wait()
}

func TestPeer_Send_Nil(t *testing.T) {
	var p *Peer
	_ = p.Send([]byte("test"))
	p2 := &Peer{}
	_ = p2.Send([]byte("test"))
	p.Close()
	p2.Close()
}

func TestNode_Broadcast_EncodeErrors(t *testing.T) {
	w, _ := crypto.NewWallet()
	id := NewIdentity(w.PublicKey, w.PrivateKey, "127.0.0.1:0")
	node, _ := NewNode(id)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	sc, _ := NewSecureConn(c1, [32]byte{1})
	p := newPeer(&PeerInfo{NodeID: [crypto.AddressSize]byte{1}}, sc)
	node.peers["peer1"] = p

	orig := gobEncodeFunc
	defer func() { gobEncodeFunc = orig }()

	gobEncodeFunc = func(v interface{}) ([]byte, error) {
		return nil, errors.New("encode error")
	}

	node.BroadcastTx(&core.Transaction{})
	node.BroadcastBlock(&core.Block{})
	node.broadcast(&GossipMsg{})

	p2 := newPeer(&PeerInfo{NodeID: [crypto.AddressSize]byte{2}}, sc)
	node.peers["peer2"] = p2
	node.sendPeerList(p)
}

type mockFailingKEM struct {
	kem.Scheme
}

func (m *mockFailingKEM) Encapsulate(pk kem.PublicKey) (ct, ss []byte, err error) {
	return nil, nil, errors.New("encapsulate fail")
}

func TestInitiatorHandshake_EncapsulateError(t *testing.T) {
	origScheme := kemScheme
	defer func() { kemScheme = origScheme }()
	kemScheme = &mockFailingKEM{Scheme: origScheme}

	w1, _ := crypto.NewWallet()
	w2, _ := crypto.NewWallet()
	id1 := NewIdentity(w1.PublicKey, w1.PrivateKey, "127.0.0.1:0")
	id2 := NewIdentity(w2.PublicKey, w2.PrivateKey, "127.0.0.1:0")

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		_ = recvGob(c2, MsgHello, &HelloPayload{})
		_ = sendGob(c2, MsgHelloResp, HelloRespPayload{
			NodeID:    id2.NodeID,
			PublicKey: id2.PublicKey,
			Accepted:  true,
		})
	}()

	remote := &PeerInfo{ListenAddr: "127.0.0.1:0", PublicKey: id2.PublicKey, NodeID: id2.NodeID}
	_, err := InitiatorHandshake(c1, id1, remote)
	if err == nil {
		t.Fatal("expected error from encapsulate failure")
	}
}
