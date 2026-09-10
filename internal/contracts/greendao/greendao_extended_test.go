package greendao

import (
	"testing"
)

// ── SubmitProposal errors ─────────────────────────────────────────────────────

func TestSubmitProposal_ExceedsTreasury(t *testing.T) {
	dao := NewGreenDAO(100, 1000)
	_, err := dao.SubmitProposal("alice", "alice", "project", 200)
	if err == nil {
		t.Error("should fail when amount exceeds treasury")
	}
}

func TestSubmitProposal_IDIncrements(t *testing.T) {
	dao := NewGreenDAO(10_000, 1000)
	id1, _ := dao.SubmitProposal("a", "a", "p1", 100)
	id2, _ := dao.SubmitProposal("b", "b", "p2", 100)
	if id2 != id1+1 {
		t.Errorf("IDs should increment: want %d, got %d", id1+1, id2)
	}
}

// ── Vote errors ───────────────────────────────────────────────────────────────

func TestVote_ProposalNotFound(t *testing.T) {
	dao := NewGreenDAO(1000, 100)
	if err := dao.Vote(99, "voter", 10, true); err == nil {
		t.Error("voting on non-existent proposal should fail")
	}
}

func TestVote_ProposalNotActive(t *testing.T) {
	dao := NewGreenDAO(10_000, 10)
	id, _ := dao.SubmitProposal("p", "p", "desc", 100)

	// Vote enough to pass immediately (10 total power, need 20% quorum=2, 60% threshold)
	dao.Vote(id, "v1", 10, true) // 10 votes for → quorum(2) met, passing(6) met → PASSED

	// Now try voting on a PASSED proposal
	if err := dao.Vote(id, "v2", 5, false); err == nil {
		t.Error("voting on a non-active proposal should fail")
	}
}

func TestVote_DuplicateVote(t *testing.T) {
	dao := NewGreenDAO(10_000, 1000)
	id, _ := dao.SubmitProposal("p", "p", "desc", 100)
	dao.Vote(id, "voter", 5, true)
	if err := dao.Vote(id, "voter", 5, true); err == nil {
		t.Error("duplicate vote should fail")
	}
}

// ── Vote outcomes ─────────────────────────────────────────────────────────────

func TestVote_PassesProposal(t *testing.T) {
	// totalPower=100, quorum=20, supermajority=60%
	dao := NewGreenDAO(10_000, 100)
	id, _ := dao.SubmitProposal("dev", "dev", "solar farm", 1000)

	// 70 votes for → quorum met (70≥20), passing threshold met (42≤70)
	dao.Vote(id, "v1", 70, true)

	if dao.Proposals[id].Status != StatusPassed {
		t.Errorf("proposal should be PASSED, got %s", dao.Proposals[id].Status)
	}
}

func TestVote_RejectsProposal(t *testing.T) {
	// totalPower=100, quorum=20, passingThreshold=60%
	dao := NewGreenDAO(10_000, 100)
	id, _ := dao.SubmitProposal("dev", "dev", "wind farm", 1000)

	// 70 votes against + 10 for = 80 total, for=10 < 48 (60%), against=70 > 32 (40%) → REJECTED
	dao.Vote(id, "v1", 10, true)
	dao.Vote(id, "v2", 70, false)

	if dao.Proposals[id].Status != StatusRejected {
		t.Errorf("proposal should be REJECTED, got %s", dao.Proposals[id].Status)
	}
}

func TestVote_BelowQuorum_StaysActive(t *testing.T) {
	dao := NewGreenDAO(10_000, 100) // quorum = 20
	id, _ := dao.SubmitProposal("dev", "dev", "hydro", 500)
	dao.Vote(id, "v1", 10, true) // only 10 votes < quorum(20) → still ACTIVE
	if dao.Proposals[id].Status != StatusActive {
		t.Errorf("proposal should remain ACTIVE, got %s", dao.Proposals[id].Status)
	}
}

func TestVote_AgainstDoesNotTriggerRejection_BelowQuorum(t *testing.T) {
	dao := NewGreenDAO(10_000, 100) // quorum=20
	id, _ := dao.SubmitProposal("dev", "dev", "wind", 200)
	// 15 against — below quorum, no status change
	dao.Vote(id, "v1", 15, false)
	if dao.Proposals[id].Status != StatusActive {
		t.Errorf("want ACTIVE, got %s", dao.Proposals[id].Status)
	}
}

// ── ExecuteProposal errors ────────────────────────────────────────────────────

func TestExecuteProposal_NotFound(t *testing.T) {
	dao := NewGreenDAO(10_000, 100)
	if err := dao.ExecuteProposal(99); err == nil {
		t.Error("executing non-existent proposal should fail")
	}
}

func TestExecuteProposal_NotPassed(t *testing.T) {
	dao := NewGreenDAO(10_000, 100)
	id, _ := dao.SubmitProposal("p", "p", "d", 100)
	// proposal is ACTIVE — cannot execute
	if err := dao.ExecuteProposal(id); err == nil {
		t.Error("executing an ACTIVE proposal should fail")
	}
}

func TestExecuteProposal_InsufficientFunds(t *testing.T) {
	// Treasury starts at 50 but proposal asks for 100
	dao := NewGreenDAO(50, 10)
	id, _ := dao.SubmitProposal("p", "p", "d", 50) // 50 ≤ 50, passes submission
	// Pass it by voting
	dao.Vote(id, "v1", 10, true)
	// Now drain treasury before execution
	dao.TreasuryBalance = 0
	if err := dao.ExecuteProposal(id); err == nil {
		t.Error("execution with empty treasury should fail")
	}
}

func TestExecuteProposal_Success(t *testing.T) {
	dao := NewGreenDAO(10_000, 100)
	id, _ := dao.SubmitProposal("dev", "dev", "solar", 1000)
	dao.Vote(id, "v1", 70, true) // passes

	prev := dao.TreasuryBalance
	if err := dao.ExecuteProposal(id); err != nil {
		t.Fatalf("ExecuteProposal: %v", err)
	}
	if dao.TreasuryBalance != prev-1000 {
		t.Errorf("treasury: want %d, got %d", prev-1000, dao.TreasuryBalance)
	}
	if dao.Proposals[id].Status != StatusExecuted {
		t.Errorf("status: want EXECUTED, got %s", dao.Proposals[id].Status)
	}
}
