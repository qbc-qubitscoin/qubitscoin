package p2p

import (
	"sync"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

const (
	gossipCacheSize = 32768
	maxGossipHops   = 3
)

// GossipCache tracks recently seen gossip message IDs to prevent re-broadcast.
type GossipCache struct {
	mu    sync.Mutex
	seen  map[[crypto.HashSize]byte]struct{}
	order [][crypto.HashSize]byte
	cap   int
}

// NewGossipCache creates a cache with the given capacity.
func NewGossipCache(capacity int) *GossipCache {
	if capacity <= 0 {
		capacity = gossipCacheSize
	}
	return &GossipCache{
		seen:  make(map[[crypto.HashSize]byte]struct{}, capacity),
		order: make([][crypto.HashSize]byte, 0, capacity),
		cap:   capacity,
	}
}

// MarkSeen records msgID as seen and returns true if it was NOT seen before
// (i.e., true means the message is new and should be forwarded).
func (gc *GossipCache) MarkSeen(msgID [crypto.HashSize]byte) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if _, exists := gc.seen[msgID]; exists {
		return false
	}

	// Evict oldest half when full.
	if len(gc.order) >= gc.cap {
		half := gc.cap / 2
		for _, old := range gc.order[:half] {
			delete(gc.seen, old)
		}
		gc.order = gc.order[half:]
	}

	gc.seen[msgID] = struct{}{}
	gc.order = append(gc.order, msgID)
	return true
}
