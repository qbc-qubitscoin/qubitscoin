package p2p

import (
	"context"
	"log"
	"net"
)

// tcpServer accepts inbound TCP connections and runs the responder handshake.
type tcpServer struct {
	listener net.Listener
	local    *Identity
}

func newTCPServer(local *Identity) (*tcpServer, error) {
	ln, err := net.Listen("tcp", local.ListenAddr)
	if err != nil {
		return nil, err
	}
	local.ListenAddr = ln.Addr().String()
	return &tcpServer{listener: ln, local: local}, nil
}

// acceptLoop runs until ctx is canceled, calling onPeer for each new peer.
func (s *tcpServer) acceptLoop(ctx context.Context, onPeer func(*Peer)) {
	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[p2p] accept error: %v", err)
				continue
			}
		}
		go s.handleConn(ctx, conn, onPeer)
	}
}

func (s *tcpServer) handleConn(ctx context.Context, conn net.Conn, onPeer func(*Peer)) {
	sc, info, err := ResponderHandshake(conn, s.local)
	if err != nil {
		log.Printf("[p2p] responder handshake failed from %s: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}
	log.Printf("[p2p] inbound secure connection established from %s", sc.RemoteAddr())
	peer := newPeer(info, sc)
	onPeer(peer)
}
