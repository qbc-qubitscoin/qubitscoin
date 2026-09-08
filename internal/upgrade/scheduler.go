package upgrade

import (
	"encoding/binary"
	"log"
	"sync"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ─────────────────────────────────────────────────────────────────────────────
// On-chain upgrade proposal
// ─────────────────────────────────────────────────────────────────────────────

// Proposal is a signed, gossip-able message that a validator emits to propose
// upgrading all nodes to a specific software version at a specific block height.
type Proposal struct {
	TargetVersion Version
	TargetHeight  uint64
	Proposer      [crypto.AddressSize]byte
	PublicKey     []byte
	Signature     []byte // ML-DSA-65 over payload()
}

// payload returns the canonical bytes that are signed.
func (p *Proposal) payload() []byte {
	var buf []byte
	buf = append(buf, []byte(p.TargetVersion.String())...)
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, p.TargetHeight)
	buf = append(buf, b8...)
	buf = append(buf, p.Proposer[:]...)
	return buf
}

// Sign signs the proposal with the given ML-DSA-65 private key bytes.
func (p *Proposal) Sign(privKeyBytes []byte) error {
	sig, err := crypto.Sign(privKeyBytes, p.payload())
	if err != nil {
		return err
	}
	p.Signature = sig
	return nil
}

// Verify checks the proposal signature.
func (p *Proposal) Verify() (bool, error) {
	if len(p.PublicKey) != crypto.PublicKeySize {
		return false, nil
	}
	addr := crypto.DeriveAddress(p.PublicKey)
	if addr != p.Proposer {
		return false, nil
	}
	return crypto.Verify(p.PublicKey, p.payload(), p.Signature)
}

// ─────────────────────────────────────────────────────────────────────────────
// Scheduler
// ─────────────────────────────────────────────────────────────────────────────

// ReadyFunc The Scheduler calls Upgrade ReadyFunc when a pending upgrade should be
// applied.  The caller should shut down gracefully then invoke Apply().
type ReadyFunc func(proposal *Proposal)

// Scheduler watches committed block heights and fires UpgradeReadyFunc when a
// scheduled upgrade height is reached.  It also tracks validator quorum for
// upgrade proposals.
type Scheduler struct {
	mu          sync.Mutex
	proposals   map[string]*Proposal       // version string → proposal
	votes       map[string]map[string]bool // version string → voterAddrHex → bool
	scheduled   *Proposal                  // proposal that reached quorum
	totalVoters int
	onReady     ReadyFunc
}

// NewScheduler creates a Scheduler.
// totalVoters is the total number of validators in the current epoch.
func NewScheduler(totalVoters int, onReady ReadyFunc) *Scheduler {
	return &Scheduler{
		proposals:   make(map[string]*Proposal),
		votes:       make(map[string]map[string]bool),
		totalVoters: totalVoters,
		onReady:     onReady,
	}
}

// AddProposal records a signed upgrade proposal.
// Returns true if this proposal now has quorum (> 2/3 of validators).
func (s *Scheduler) AddProposal(p *Proposal) (bool, error) {
	ok, err := p.Verify()
	if err != nil || !ok {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := p.TargetVersion.String()
	if _, exists := s.proposals[key]; !exists {
		s.proposals[key] = p
		s.votes[key] = make(map[string]bool)
	}

	voterKey := crypto.ToHex(p.Proposer)
	s.votes[key][voterKey] = true

	quorum := s.hasQuorum(key)
	if quorum && s.scheduled == nil {
		s.scheduled = p
		log.Printf("[upgrade] quorum reached for %s at height %d",
			p.TargetVersion, p.TargetHeight)
	}
	return quorum, nil
}

// hasQuorum returns true if > 2/3 of validators have signed this proposal.
// Must be called with s.mu held.
func (s *Scheduler) hasQuorum(versionKey string) bool {
	count := len(s.votes[versionKey])
	return 3*count > 2*s.totalVoters
}

// OnBlock is called by the node for each committed block height.
// If the scheduled upgrade height is reached, onReady is invoked in a goroutine.
func (s *Scheduler) OnBlock(height uint64) {
	s.mu.Lock()
	p := s.scheduled
	s.mu.Unlock()

	if p == nil {
		return
	}
	if height >= p.TargetHeight {
		log.Printf("[upgrade] block %d reached upgrade target %d — triggering %s",
			height, p.TargetHeight, p.TargetVersion)
		if s.onReady != nil {
			go s.onReady(p)
		}
		// Clear to prevent repeated firings.
		s.mu.Lock()
		s.scheduled = nil
		s.mu.Unlock()
	}
}

// Scheduled returns the current pending upgrade proposal or nil.
func (s *Scheduler) Scheduled() *Proposal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduled
}
