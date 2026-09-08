package p2p

import (
	"context"
	"log"
	"time"
)

const peerReadDeadline = 60 * time.Second

// Peer represents a connected remote node.
type Peer struct {
	Info  *PeerInfo
	conn  *SecureConn
	inbox chan []byte
	quit  chan struct{}
}

// newPeer creates a Peer from an established SecureConn.
func newPeer(info *PeerInfo, conn *SecureConn) *Peer {
	return &Peer{
		Info:  info,
		conn:  conn,
		inbox: make(chan []byte, 256),
		quit:  make(chan struct{}),
	}
}

// run starts the receiver pump; call in a goroutine.
func (p *Peer) run(ctx context.Context, onMsg func([]byte)) {
	defer close(p.quit)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := p.conn.SetReadDeadline(time.Now().Add(peerReadDeadline))
		if err != nil {
			return
		}
		msg, err := p.conn.Recv()
		if err != nil {
			log.Printf("[p2p] peer %s read error: %v", p.Info.ListenAddr, err)
			return
		}
		onMsg(msg)
	}
}

// Send encrypts and sends a message to this peer.
func (p *Peer) Send(data []byte) error {
	return p.conn.Send(data)
}

// Close terminates the peer connection.
func (p *Peer) Close() {
	err := p.conn.Close()
	if err != nil {
		return
	}
}
