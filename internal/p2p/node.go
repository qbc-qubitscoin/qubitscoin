package p2p

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"sync"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

const (
	defaultMaxPeers = 25
	dialTimeout     = 10 * time.Second
	pingInterval    = 30 * time.Second
)

// Node is the top-level P2P network participant.
type Node struct {
	identity *Identity
	server   *tcpServer
	gossip   *GossipCache

	peersMu sync.RWMutex
	peers   map[string]*Peer // nodeID hex -> Peer

	// Hooks — set by the application layer before calling Start.
	OnPeerConnect    func(*PeerInfo)
	OnPeerDisconnect func(*PeerInfo)
	OnTxReceived     func(*core.Transaction)
	OnBlockReceived  func(*core.Block)
}

// NewNode creates a P2P node from a wallet identity and listen address.
func NewNode(local *Identity) (*Node, error) {
	srv, err := newTCPServer(local)
	if err != nil {
		return nil, err
	}
	return &Node{
		identity: local,
		server:   srv,
		gossip:   NewGossipCache(0),
		peers:    make(map[string]*Peer),
	}, nil
}

// Start begins listening for inbound connections.
func (n *Node) Start(ctx context.Context) {
	log.Printf("[p2p] node started, listening on %s (nodeID=%s)",
		n.identity.ListenAddr, crypto.ToHex(n.identity.NodeID))
	go n.server.acceptLoop(ctx, func(p *Peer) {
		n.addPeer(ctx, p)
	})
	go n.pingLoop(ctx)
}

// Connect dials a remote address and performs the initiator handshake.
func (n *Node) Connect(ctx context.Context, addr string) error {
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return err
	}

	// We need the remote's public key for KEM — use a placeholder PeerInfo
	// the server will populate it during handshake.
	remotePeerInfo := &PeerInfo{ListenAddr: addr}
	sc, err := InitiatorHandshake(conn, n.identity, remotePeerInfo)
	if err != nil {
		err := conn.Close()
		if err != nil {
			return err
		}
		return err
	}

	peer := newPeer(remotePeerInfo, sc)
	n.addPeer(ctx, peer)
	return nil
}

func (n *Node) addPeer(ctx context.Context, p *Peer) {
	n.peersMu.Lock()
	key := crypto.ToHex(p.Info.NodeID)
	if _, exists := n.peers[key]; exists {
		n.peersMu.Unlock()
		p.Close()
		return
	}
	if len(n.peers) >= defaultMaxPeers {
		n.peersMu.Unlock()
		p.Close()
		return
	}
	n.peers[key] = p
	n.peersMu.Unlock()

	log.Printf("[p2p] peer connected: %s (%s)", p.Info.ListenAddr, crypto.ToHex(p.Info.NodeID))
	if n.OnPeerConnect != nil {
		n.OnPeerConnect(p.Info)
	}

	go func() {
		p.run(ctx, func(msg []byte) { n.dispatch(p, msg) })
		n.removePeer(p)
	}()

	// Send our peer list.
	go n.sendPeerList(p)
}

func (n *Node) removePeer(p *Peer) {
	n.peersMu.Lock()
	delete(n.peers, crypto.ToHex(p.Info.NodeID))
	n.peersMu.Unlock()

	log.Printf("[p2p] peer disconnected: %s", p.Info.ListenAddr)
	if n.OnPeerDisconnect != nil {
		n.OnPeerDisconnect(p.Info)
	}
}

// BroadcastTx gossips a transaction to all peers.
func (n *Node) BroadcastTx(tx *core.Transaction) {
	msg, err := NewTxGossip(tx)
	if err != nil {
		return
	}
	if !n.gossip.MarkSeen(msg.MsgID) {
		return
	}
	n.broadcast(msg)
}

// BroadcastBlock gossips a block to all peers.
func (n *Node) BroadcastBlock(blk *core.Block) {
	msg, err := NewBlockGossip(blk)
	if err != nil {
		return
	}
	if !n.gossip.MarkSeen(msg.MsgID) {
		return
	}
	n.broadcast(msg)
}

func (n *Node) broadcast(msg *GossipMsg) {
	if msg.Hops >= maxGossipHops {
		return
	}
	data, err := gobEncode(msg)
	if err != nil {
		return
	}
	frame := append([]byte{byte(msg.Type)}, data...)

	n.peersMu.RLock()
	defer n.peersMu.RUnlock()
	for _, p := range n.peers {
		err := p.Send(frame)
		if err != nil {
			return
		}
	}
}

func (n *Node) dispatch(sender *Peer, frame []byte) {
	if len(frame) < 1 {
		return
	}
	mt := MsgType(frame[0])
	payload := frame[1:]

	switch mt {
	case MsgTx:
		var gossip GossipMsg
		if err := gobDecode(payload, &gossip); err != nil {
			return
		}
		if !n.gossip.MarkSeen(gossip.MsgID) {
			return
		}
		var tx core.Transaction
		if err := gobDecode(gossip.Data, &tx); err != nil {
			return
		}
		if n.OnTxReceived != nil {
			n.OnTxReceived(&tx)
		}
		// Re-gossip with incremented hop count.
		gossip.Hops++
		n.broadcast(&gossip)

	case MsgBlock:
		var gossip GossipMsg
		if err := gobDecode(payload, &gossip); err != nil {
			return
		}
		if !n.gossip.MarkSeen(gossip.MsgID) {
			return
		}
		var blk core.Block
		if err := gobDecode(gossip.Data, &blk); err != nil {
			return
		}
		if n.OnBlockReceived != nil {
			n.OnBlockReceived(&blk)
		}
		gossip.Hops++
		n.broadcast(&gossip)

	case MsgPeerList:
		var pl PeerListPayload
		if err := gobDecode(payload, &pl); err != nil {
			return
		}
		// Connect to unknown peers.
		for _, info := range pl.Peers {
			if info.NodeID == n.identity.NodeID {
				continue
			}
			key := crypto.ToHex(info.NodeID)
			n.peersMu.RLock()
			_, known := n.peers[key]
			n.peersMu.RUnlock()
			if !known {
				go func() {
					err := n.Connect(context.Background(), info.ListenAddr)
					if err != nil {
						log.Printf("[p2p] failed to connect to peer %s: %v", info.ListenAddr, err)
					}
				}()
			}
		}

	case MsgPing:
		pong := []byte{byte(MsgPong), 0, 0, 0, 0, 0, 0, 0, 0}
		err := sender.Send(pong)
		if err != nil {
			return
		}

	case MsgPong:
		// no-op
	}
}

func (n *Node) sendPeerList(p *Peer) {
	n.peersMu.RLock()
	peers := make([]PeerInfo, 0, len(n.peers))
	for _, peer := range n.peers {
		if peer.Info.NodeID != p.Info.NodeID {
			peers = append(peers, *peer.Info)
		}
	}
	n.peersMu.RUnlock()

	if len(peers) == 0 {
		return
	}
	pl := PeerListPayload{Peers: peers}
	data, err := gobEncode(pl)
	if err != nil {
		return
	}
	frame := append([]byte{byte(MsgPeerList)}, data...)
	err = p.Send(frame)
	if err != nil {
		return
	}
}

func (n *Node) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping := make([]byte, 9)
			ping[0] = byte(MsgPing)
			binary.BigEndian.PutUint64(ping[1:], uint64(time.Now().UnixNano()))
			n.peersMu.RLock()
			for _, p := range n.peers {
				err := p.Send(ping)
				if err != nil {
					return
				}
			}
			n.peersMu.RUnlock()
		}
	}
}

// PeerCount returns the number of connected peers.
func (n *Node) PeerCount() int {
	n.peersMu.RLock()
	defer n.peersMu.RUnlock()
	return len(n.peers)
}

// BroadcastRaw sends a raw byte frame to all connected peers.
// Used by the sync layer to send MsgGetBlocks without gossip dedup.
func (n *Node) BroadcastRaw(frame []byte) {
	n.peersMu.RLock()
	defer n.peersMu.RUnlock()
	for _, p := range n.peers {
		if err := p.Send(frame); err != nil {
			log.Printf("[p2p] BroadcastRaw send error to %s: %v", p.Info.ListenAddr, err)
		}
	}
}
