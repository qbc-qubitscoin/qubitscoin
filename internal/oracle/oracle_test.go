package oracle

import (
	"testing"
)

func TestNewNode(t *testing.T) {
	n := NewNode("node-1", 1000, "http://localhost:8545")
	if n.NodeID != "node-1" {
		t.Errorf("NodeID: want node-1, got %s", n.NodeID)
	}
	if n.StakeAmount != 1000 {
		t.Errorf("StakeAmount: want 1000, got %d", n.StakeAmount)
	}
	if n.RPCEndpoint != "http://localhost:8545" {
		t.Errorf("RPCEndpoint: want http://localhost:8545, got %s", n.RPCEndpoint)
	}
}

func TestFetchFinancialData(t *testing.T) {
	n := NewNode("oracle-1", 500, "http://rpc")
	price, err := n.FetchFinancialData("BTC")
	if err != nil {
		t.Fatalf("FetchFinancialData error: %v", err)
	}
	// Simulated price is 2000 + jitter(0-4)
	if price < 2000 || price > 2004 {
		t.Errorf("price %d out of expected range [2000, 2004]", price)
	}
}

func TestFetchESGScore(t *testing.T) {
	n := NewNode("oracle-2", 100, "http://esg")
	score, err := n.FetchESGScore("company-abc")
	if err != nil {
		t.Fatalf("FetchESGScore error: %v", err)
	}
	if score != 85 {
		t.Errorf("ESG score: want 85, got %d", score)
	}
}

func TestSubmitTransaction(t *testing.T) {
	n := NewNode("oracle-3", 200, "http://rpc")
	err := n.SubmitTransaction("0xcontract", 42)
	if err != nil {
		t.Fatalf("SubmitTransaction should not return error, got: %v", err)
	}
}
