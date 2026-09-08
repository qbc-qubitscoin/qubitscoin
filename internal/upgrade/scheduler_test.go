package upgrade

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

var schedWallet *crypto.Wallet

func init() {
	var err error
	schedWallet, err = crypto.NewWallet()
	if err != nil {
		panic(err)
	}
}

func makeProposal(w *crypto.Wallet, version string, height uint64) *Proposal {
	v, _ := ParseVersion(version)
	return &Proposal{
		TargetVersion: v,
		TargetHeight:  height,
		Proposer:      w.Address,
		PublicKey:     w.PublicKey,
	}
}

func TestProposal_Sign_Verify(t *testing.T) {
	p := makeProposal(schedWallet, "v1.0.0", 100)
	if err := p.Sign(schedWallet.PrivateKey); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, err := p.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("valid proposal should verify")
	}
}

func TestProposal_Verify_Tampered(t *testing.T) {
	p := makeProposal(schedWallet, "v1.0.0", 100)
	_ = p.Sign(schedWallet.PrivateKey)
	p.TargetHeight = 999 // tamper after signing
	ok, _ := p.Verify()
	if ok {
		t.Fatal("tampered proposal should not verify")
	}
}

func TestProposal_Verify_WrongKey(t *testing.T) {
	w2, _ := crypto.NewWallet()
	p := makeProposal(schedWallet, "v1.0.0", 50)
	_ = p.Sign(schedWallet.PrivateKey)
	p.PublicKey = w2.PublicKey // swap key
	ok, _ := p.Verify()
	if ok {
		t.Fatal("the wrong public key should fail Verify")
	}
}

func TestScheduler_AddProposal_SingleValidator(t *testing.T) {
	s := NewScheduler(1, nil)
	p := makeProposal(schedWallet, "v1.0.0", 100)
	_ = p.Sign(schedWallet.PrivateKey)

	quorum, err := s.AddProposal(p)
	if err != nil {
		t.Fatalf("AddProposal: %v", err)
	}
	if !quorum {
		t.Fatal("a single validator signing should reach quorum immediately")
	}
}

func TestScheduler_AddProposal_InvalidSignature(t *testing.T) {
	s := NewScheduler(1, nil)
	p := makeProposal(schedWallet, "v1.0.0", 100)
	// Do not sign → invalid signature.
	p.Signature = make([]byte, crypto.SignatureSize) // wrong sig
	_, err := s.AddProposal(p)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if s.Scheduled() != nil {
		t.Fatal("an invalid proposal should not schedule an upgrade")
	}
}

func TestScheduler_Scheduled_SetAfterQuorum(t *testing.T) {
	s := NewScheduler(1, nil)
	p := makeProposal(schedWallet, "v2.0.0", 200)
	_ = p.Sign(schedWallet.PrivateKey)
	_, _ = s.AddProposal(p)

	sched := s.Scheduled()
	if sched == nil {
		t.Fatal("Scheduled() should be non-nil after quorum")
	}
	if sched.TargetHeight != 200 {
		t.Errorf("TargetHeight: want 200, got %d", sched.TargetHeight)
	}
}

func TestScheduler_OnBlock_FiresAtTargetHeight(t *testing.T) {
	var fired int32
	s := NewScheduler(1, func(p *Proposal) {
		atomic.AddInt32(&fired, 1)
	})

	p := makeProposal(schedWallet, "v1.5.0", 50)
	_ = p.Sign(schedWallet.PrivateKey)
	_, _ = s.AddProposal(p)

	s.OnBlock(49) // not yet
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt32(&fired) != 0 {
		t.Fatal("onReady should not fire before target height")
	}

	s.OnBlock(50) // trigger
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&fired) != 1 {
		t.Fatalf("onReady should fire exactly once at target height, got %d", atomic.LoadInt32(&fired))
	}
}

func TestScheduler_OnBlock_DoesNotFireTwice(t *testing.T) {
	var fired int32
	s := NewScheduler(1, func(p *Proposal) {
		atomic.AddInt32(&fired, 1)
	})

	p := makeProposal(schedWallet, "v1.6.0", 10)
	_ = p.Sign(schedWallet.PrivateKey)
	_, _ = s.AddProposal(p)

	s.OnBlock(10)
	s.OnBlock(11)
	s.OnBlock(12)
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&fired) != 1 {
		t.Fatalf("onReady should fire exactly once, got %d", atomic.LoadInt32(&fired))
	}
}

func TestScheduler_NoQuorum_DoesNotSchedule(t *testing.T) {
	// 3 total validators — single proposal doesn't reach >2/3.
	s := NewScheduler(3, nil)
	p := makeProposal(schedWallet, "v3.0.0", 100)
	_ = p.Sign(schedWallet.PrivateKey)

	quorum, _ := s.AddProposal(p)
	if quorum {
		t.Fatal("1/3 should not reach quorum")
	}
	if s.Scheduled() != nil {
		t.Fatal("upgrade should not be scheduled without a quorum")
	}
}
