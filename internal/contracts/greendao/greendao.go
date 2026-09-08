package greendao

import (
	"errors"
)

type ProposalStatus string

const (
	StatusActive    ProposalStatus = "ACTIVE"
	StatusPassed    ProposalStatus = "PASSED"
	StatusRejected  ProposalStatus = "REJECTED"
	StatusExecuted  ProposalStatus = "EXECUTED"
)

// Proposal represents a request for funding a renewable energy project.
type Proposal struct {
	ID             uint64
	Proposer       string
	Recipient      string
	AmountRequested uint64
	Description    string
	VotesFor       uint64
	VotesAgainst   uint64
	Status         ProposalStatus
}

// GreenDAO manages proposals and treasury funds.
type GreenDAO struct {
	TreasuryBalance uint64
	Proposals       map[uint64]*Proposal
	NextProposalID  uint64
	HasVoted        map[uint64]map[string]bool // ProposalID -> User -> true
	TotalVotingPower uint64 // Mock of total circulating supply
}

func NewGreenDAO(initialTreasury, totalPower uint64) *GreenDAO {
	return &GreenDAO{
		TreasuryBalance:  initialTreasury,
		Proposals:        make(map[uint64]*Proposal),
		NextProposalID:   1,
		HasVoted:         make(map[uint64]map[string]bool),
		TotalVotingPower: totalPower,
	}
}

// SubmitProposal creates a new funding request.
func (dao *GreenDAO) SubmitProposal(proposer, recipient, description string, amount uint64) (uint64, error) {
	if amount > dao.TreasuryBalance {
		return 0, errors.New("requested amount exceeds treasury balance")
	}

	id := dao.NextProposalID
	dao.Proposals[id] = &Proposal{
		ID:              id,
		Proposer:        proposer,
		Recipient:       recipient,
		AmountRequested: amount,
		Description:     description,
		Status:          StatusActive,
	}
	dao.HasVoted[id] = make(map[string]bool)
	dao.NextProposalID++
	return id, nil
}

// Vote casts a vote on an active proposal.
func (dao *GreenDAO) Vote(proposalID uint64, voter string, votingPower uint64, support bool) error {
	p, exists := dao.Proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}
	if p.Status != StatusActive {
		return errors.New("proposal is not active")
	}
	if dao.HasVoted[proposalID][voter] {
		return errors.New("voter has already voted")
	}

	if support {
		p.VotesFor += votingPower
	} else {
		p.VotesAgainst += votingPower
	}

	dao.HasVoted[proposalID][voter] = true

	// Check if quorum (20%) and supermajority (60%) are met
	totalVotes := p.VotesFor + p.VotesAgainst
	quorum := dao.TotalVotingPower / 5 // 20%
	
	if totalVotes >= quorum {
		// Calculate percentage of passing votes
		passingThreshold := (totalVotes * 60) / 100
		if p.VotesFor >= passingThreshold {
			p.Status = StatusPassed
		} else if p.VotesAgainst > (totalVotes * 40) / 100 {
			// If enough against to mathematically prevent 60%
			p.Status = StatusRejected
		}
	}

	return nil
}

// ExecuteProposal disburses funds if the proposal passed.
func (dao *GreenDAO) ExecuteProposal(proposalID uint64) error {
	p, exists := dao.Proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}
	if p.Status != StatusPassed {
		return errors.New("proposal must be PASSED to execute")
	}
	if dao.TreasuryBalance < p.AmountRequested {
		return errors.New("insufficient treasury funds")
	}

	dao.TreasuryBalance -= p.AmountRequested
	p.Status = StatusExecuted

	// In a real system, we invoke the WASM host function qbc_transfer here
	// qbcTransfer(p.Recipient, p.AmountRequested)

	return nil
}
