package greendao

import (
	"testing"
)

// TestPilotFundingCycle simulates a full proposal lifecycle to satisfy Phase 12 exit criteria.
func TestPilotFundingCycle(t *testing.T) {
	// Initialize DAO with $1M in treasury and 100,000 total voting power.
	dao := NewGreenDAO(1000000, 100000)

	// 1. Proposal Submission
	// "Texas Solar Farm Expansion" asks for $200k.
	proposalID, err := dao.SubmitProposal("developer_alice", "spv_texas_solar", "Build a 50MW Solar Array", 200000)
	if err != nil {
		t.Fatalf("Failed to submit proposal: %v", err)
	}

	// 2. Voting
	// Quorum is 20k (20%). We need 60% approval.
	
	// Bob votes Yes (weight: 15,000)
	err = dao.Vote(proposalID, "voter_bob", 15000, true)
	if err != nil { t.Fatal(err) }

	// Charlie votes No (weight: 5,000)
	err = dao.Vote(proposalID, "voter_charlie", 5000, false)
	if err != nil { t.Fatal(err) }

	// At this point: Total = 20k (Quorum met). Yes = 15k (75% > 60% threshold).
	// Proposal should be PASSED.
	p := dao.Proposals[proposalID]
	if p.Status != StatusPassed {
		t.Fatalf("Expected proposal to be PASSED, got %s", p.Status)
	}

	// 3. Disbursement (Execution)
	err = dao.ExecuteProposal(proposalID)
	if err != nil {
		t.Fatalf("Failed to execute passed proposal: %v", err)
	}

	// 4. Verification
	if p.Status != StatusExecuted {
		t.Fatalf("Expected proposal to be EXECUTED, got %s", p.Status)
	}

	if dao.TreasuryBalance != 800000 { // 1,000,000 - 200,000
		t.Fatalf("Expected treasury to be 800,000, got %d", dao.TreasuryBalance)
	}

	t.Log("Passed: Pilot funding cycle (proposal → vote → disbursement) completed successfully.")
}
