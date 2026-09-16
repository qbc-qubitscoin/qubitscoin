// Package sync provides initial block download (IBD) and chain catch-up.
//
// The Syncer periodically checks whether the local chain is behind any
// connected peer and, if so, requests the missing blocks in batches.
//
// Protocol (uses existing P2P message types MsgGetBlocks / MsgBlocks):
//
//  1. Every SyncInterval the Syncer compares the local height to the height
//     advertised in the peer's latest MsgBlock gossip.
//  2. If behind, it sends MsgGetBlocks{FromHeight: localTip+1, MaxCount: 64}.
//  3. The responding peer sends back an encoded []core.Block slice.
//  4. Blocks are validated and applied sequentially.  The engine and state
//     are updated in-place; committed blocks are written to the BlockStore.
package sync

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"sync"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/p2p"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/storage"
	"github.com/qbc-qubitscoin/qubitscoin/internal/vm"
)

var (
	// SyncInterval is how often the syncer polls for new blocks from peers.
	SyncInterval = 5 * time.Second
	// BatchSize is the maximum number of blocks requested per round.
	BatchSize = 64
)

// Syncer drives initial block download and ongoing chain catch-up.
type Syncer struct {
	engine       *consensus.Engine
	st           *state.DB
	store        *storage.BlockStore
	node         *p2p.Node
	execVM       *vm.VM
	validatorPub []byte // for block-signature verification

	mu          sync.Mutex
	peerHeights map[string]uint64 // nodeID hex → last seen height from peer
}

// New creates a Syncer. execVM may be nil if smart-contract replay is not needed.
func New(
	engine *consensus.Engine,
	st *state.DB,
	store *storage.BlockStore,
	node *p2p.Node,
	execVM *vm.VM,
	validatorPub []byte,
) *Syncer {
	return &Syncer{
		engine:       engine,
		st:           st,
		store:        store,
		node:         node,
		execVM:       execVM,
		validatorPub: validatorPub,
		peerHeights:  make(map[string]uint64),
	}
}

// ObservePeerHeight records the latest height advertised by a peer.
// Called by the P2P block-gossip handler.
func (s *Syncer) ObservePeerHeight(nodeIDHex string, height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if height > s.peerHeights[nodeIDHex] {
		s.peerHeights[nodeIDHex] = height
	}
}

// BestPeerHeight returns the highest height seen across all peers.
func (s *Syncer) BestPeerHeight() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best uint64
	for _, h := range s.peerHeights {
		if h > best {
			best = h
		}
	}
	return best
}

// Run starts the periodic sync loop; blocks until ctx is canceled.
func (s *Syncer) Run(ctx context.Context) {
	ticker := time.NewTicker(SyncInterval)
	defer ticker.Stop()

	log.Println("[sync] syncer started")
	for {
		select {
		case <-ctx.Done():
			log.Println("[sync] syncer stopped")
			return
		case <-ticker.C:
			s.trySync()
		}
	}
}

// trySync performs one catch-up attempt if the chain is behind.
func (s *Syncer) trySync() {
	local := s.engine.Height()
	best := s.BestPeerHeight()
	if best <= local {
		return // already up to date or no peers
	}

	want := local // next height to request
	log.Printf("[sync] behind peers: local=%d best_peer=%d, catching up…", local, best)

	// Request blocks in batches until caught up or no more peers.
	for want < best {
		count := uint32(BatchSize)
		remaining := best - want
		if remaining < uint64(count) {
			count = uint32(remaining)
		}

		payload := p2p.GetBlocksPayload{
			FromHeight: want,
			MaxCount:   count,
		}
		data, _ := gobEncode(payload)
		frame := append([]byte{byte(p2p.MsgGetBlocks)}, data...)

		// Broadcast to all peers — the first responder wins.
		s.node.BroadcastRaw(frame)
		want += uint64(count)
	}
}

// ApplyBlocks validates and applies a slice of blocks received from a peer.
// Blocks outside the expected range or failing validation are discarded.
func (s *Syncer) ApplyBlocks(blocks []*core.Block) {
	for _, blk := range blocks {
		localHeight := s.engine.Height()
		if blk.Header.Height != localHeight {
			// Already have this block, or it's ahead of what we expect.
			continue
		}
		// Basic structural validation.
		if err := blk.VerifyValidatorSig(s.validatorPub); err != nil {
			log.Printf("[sync] invalid block sig at h=%d: %v", blk.Header.Height, err)
			continue
		}
		// Apply the block to state.
		prev := s.engine.BlockByHeight(localHeight - 1)
		if blk.Header.PrevHash != prev.Hash {
			log.Printf("[sync] prev-hash mismatch at h=%d", blk.Header.Height)
			continue
		}
		result, err := state.ApplyBlock(s.st, blk, blk.Header.ValidatorAddr, s.execVM)
		if err != nil {
			log.Printf("[sync] ApplyBlock h=%d: %v", blk.Header.Height, err)
			continue
		}
		log.Printf("[sync] applied block h=%d hash=%s txs=%d burned=%d q reward=%d q",
			blk.Header.Height,
			crypto.ToHex(blk.Hash)[:12]+"…",
			len(blk.Txs),
			result.BurnedFees,
			result.BlockReward,
		)
		// Persist block and state.
		if s.store != nil {
			if err := s.store.PutBlock(blk); err != nil {
				log.Printf("[sync] store block h=%d: %v", blk.Header.Height, err)
			} else {
				_ = s.store.UpdateTip(blk.Header.Height, blk.Hash)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func gobEncode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
