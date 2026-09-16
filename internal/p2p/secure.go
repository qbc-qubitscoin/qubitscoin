package p2p

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// SecureConn wraps a TCP connection with AES-256-GCM encryption.
// Nonce is counter-based: [4-byte zero pad][8-byte big-endian sequence number].
type SecureConn struct {
	conn    net.Conn
	gcm     cipher.AEAD
	sendMu  sync.Mutex
	recvMu  sync.Mutex
	sendSeq uint64
	recvSeq uint64
}

var (
	newCipherFunc = aes.NewCipher
	newGCMFunc    = cipher.NewGCM
)

// NewSecureConn creates a SecureConn from a raw connection and a 32-byte session key.
func NewSecureConn(conn net.Conn, sessionKey [32]byte) (*SecureConn, error) {
	block, err := newCipherFunc(sessionKey[:])
	if err != nil {
		return nil, err
	}
	gcm, err := newGCMFunc(block)
	if err != nil {
		return nil, err
	}
	return &SecureConn{conn: conn, gcm: gcm}, nil
}

// nonce12 converts an uint64 counter to a 12-byte GCM nonce.
func nonce12(seq uint64) [12]byte {
	var n [12]byte
	binary.BigEndian.PutUint64(n[4:], seq)
	return n
}

// Send encrypts and sends a message frame.
func (sc *SecureConn) Send(plaintext []byte) error {
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()

	n := nonce12(sc.sendSeq)
	sc.sendSeq++
	ciphertext := sc.gcm.Seal(nil, n[:], plaintext, nil)
	return writeRawFrame(sc.conn, ciphertext)
}

// Recv receives and decrypts one message frame.
func (sc *SecureConn) Recv() ([]byte, error) {
	sc.recvMu.Lock()
	defer sc.recvMu.Unlock()

	frame, err := readRawFrame(sc.conn)
	if err != nil {
		return nil, err
	}
	n := nonce12(sc.recvSeq)
	sc.recvSeq++
	plain, err := sc.gcm.Open(nil, n[:], frame, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plain, nil
}

// SetReadDeadline sets a deadline on the underlying connection.
func (sc *SecureConn) SetReadDeadline(t time.Time) error {
	return sc.conn.SetReadDeadline(t)
}

// Close closes the underlying connection.
func (sc *SecureConn) Close() error {
	return sc.conn.Close()
}

// RemoteAddr returns the remote network address.
func (sc *SecureConn) RemoteAddr() net.Addr {
	return sc.conn.RemoteAddr()
}
