package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// MsgType lists P2P message type codes.
type MsgType uint8

const (
	MsgHello     MsgType = 0x01
	MsgHelloResp MsgType = 0x02
	MsgKEMInit   MsgType = 0x03
	MsgKEMDone   MsgType = 0x04

	MsgTx        MsgType = 0x10
	MsgBlock     MsgType = 0x11
	MsgGetBlocks MsgType = 0x12
	MsgBlocks    MsgType = 0x13
	MsgPeerList  MsgType = 0x14

	MsgPing MsgType = 0x20
	MsgPong MsgType = 0x21
)

// ── Payload structs ──────────────────────────────────────────────────────────

type HelloPayload struct {
	NodeID     [crypto.AddressSize]byte
	PublicKey  []byte
	ListenAddr string
	Version    uint32
}

type HelloRespPayload struct {
	NodeID    [crypto.AddressSize]byte
	PublicKey []byte
	Accepted  bool
}

type KEMInitPayload struct {
	SenderNodeID [crypto.AddressSize]byte
	Ciphertext   []byte // ML-KEM-768 ciphertext
	Signature    []byte // ML-DSA-65 over SHA-3(ct ‖ respNodeID)
}

type KEMDonePayload struct {
	ReceiverNodeID [crypto.AddressSize]byte
	Signature      []byte // ML-DSA-65 over SHA-3(ss[:16] ‖ initNodeID)
}

type GetBlocksPayload struct {
	FromHeight uint64
	MaxCount   uint32
}

type PeerListPayload struct {
	Peers []PeerInfo
}

// RawMessage is the wire frame before encryption: [1-byte type][gob-encoded payload].
type RawMessage struct {
	Type    MsgType
	Payload []byte
}

// ── Wire framing ─────────────────────────────────────────────────────────────

const maxFrameSize = 8 << 20 // 8 MiB

// writeRawFrame writes [4-byte big-endian length][data] to conn.
func writeRawFrame(conn net.Conn, data []byte) error {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := conn.Write(hdr[:]); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}

// readRawFrame reads one length-prefixed frame from conn.
func readRawFrame(conn net.Conn) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(hdr[:])
	if size > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ── Gob helpers ──────────────────────────────────────────────────────────────

func gobEncode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobDecode(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}

// ── Gossip message wrapper ────────────────────────────────────────────────────

// GossipMsg wraps a transaction or block for gossip propagation.
type GossipMsg struct {
	MsgID [crypto.HashSize]byte
	Hops  uint8
	Type  MsgType
	Data  []byte
}

var gobEncodeFunc = gobEncode

// NewTxGossip creates a gossip wrapper for a transaction.
func NewTxGossip(tx *core.Transaction) (*GossipMsg, error) {
	data, err := gobEncodeFunc(tx)
	if err != nil {
		return nil, err
	}
	return &GossipMsg{
		MsgID: tx.Hash,
		Hops:  0,
		Type:  MsgTx,
		Data:  data,
	}, nil
}

// NewBlockGossip creates a gossip wrapper for a block.
func NewBlockGossip(blk *core.Block) (*GossipMsg, error) {
	data, err := gobEncodeFunc(blk)
	if err != nil {
		return nil, err
	}
	return &GossipMsg{
		MsgID: blk.Hash,
		Hops:  0,
		Type:  MsgBlock,
		Data:  data,
	}, nil
}
