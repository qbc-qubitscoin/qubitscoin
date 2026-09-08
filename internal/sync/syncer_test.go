package sync_test

import (
	"testing"

	chainsync "github.com/qbc-qubitscoin/qubitscoin/internal/sync"
)

func TestSyncer_ObservePeerHeight(t *testing.T) {
	s := chainsync.New(nil, nil, nil, nil, nil, nil)

	s.ObservePeerHeight("peer1", 10)
	s.ObservePeerHeight("peer2", 5)
	s.ObservePeerHeight("peer1", 20) // update upward

	if best := s.BestPeerHeight(); best != 20 {
		t.Errorf("BestPeerHeight: want 20, got %d", best)
	}
}

func TestSyncer_BestPeerHeight_NoPeers(t *testing.T) {
	s := chainsync.New(nil, nil, nil, nil, nil, nil)
	if h := s.BestPeerHeight(); h != 0 {
		t.Errorf("want 0 with no peers, got %d", h)
	}
}

func TestSyncer_ObservePeerHeight_NoRegress(t *testing.T) {
	s := chainsync.New(nil, nil, nil, nil, nil, nil)
	s.ObservePeerHeight("p", 100)
	s.ObservePeerHeight("p", 50) // lower height should be ignored
	if best := s.BestPeerHeight(); best != 100 {
		t.Errorf("height should not regress: want 100, got %d", best)
	}
}
