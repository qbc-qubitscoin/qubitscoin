package p2p

import "github.com/qbc-qubitscoin/qubitscoin/internal/crypto"

// Identity holds the local node's long-term ML-DSA-65 identity.
type Identity struct {
	NodeID     [crypto.AddressSize]byte // SHA-3-256(PublicKey)
	PublicKey  []byte
	PrivateKey []byte
	ListenAddr string
}

// NewIdentity creates an Identity from existing key material.
func NewIdentity(pubKey, privKey []byte, listenAddr string) *Identity {
	nodeID := crypto.DeriveAddress(pubKey)
	return &Identity{
		NodeID:     nodeID,
		PublicKey:  pubKey,
		PrivateKey: privKey,
		ListenAddr: listenAddr,
	}
}

// PeerInfo is the shareable descriptor of a remote peer.
type PeerInfo struct {
	NodeID     [crypto.AddressSize]byte
	PublicKey  []byte
	ListenAddr string
}
