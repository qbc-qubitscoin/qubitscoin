package oracle

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// OracleNode represents an off-chain client that fetches real-world data
// and submits it to the QubitsCoin blockchain.
type OracleNode struct {
	NodeID      string
	StakeAmount uint64
	RPCEndpoint string
}

// NewNode initializes a new oracle client worker.
func NewNode(id string, stake uint64, rpc string) *OracleNode {
	return &OracleNode{
		NodeID:      id,
		StakeAmount: stake,
		RPCEndpoint: rpc,
	}
}

// FetchFinancialData simulates fetching price data (e.g., from Binance or CoinGecko).
func (n *OracleNode) FetchFinancialData(ticker string) (uint64, error) {
	// In a real implementation, this would make an HTTP GET request to a price API.
	// We simulate a price response with a small random jitter to represent different
	// oracle nodes querying at slightly different milliseconds.
	
	basePrice := int64(2000)
	jitter, _ := rand.Int(rand.Reader, big.NewInt(5)) // 0 to 4
	
	simulatedPrice := uint64(basePrice + jitter.Int64())
	return simulatedPrice, nil
}

// FetchESGScore simulates fetching specialized QESG data from an accredited registry.
// This supports Phase 10's requirement for sustainability verification.
func (n *OracleNode) FetchESGScore(companyID string) (uint64, error) {
	// Specialized ESG score retrieval (0-100 scale).
	// Real implementation requires verifying the TLS certificate of the registry.
	return 85, nil
}

// SubmitTransaction simulates broadcasting the fetched data to the smart contract.
func (n *OracleNode) SubmitTransaction(contractAddress string, data uint64) error {
	// Calls internal/core transaction signing logic and broadcasts via RPC.
	fmt.Printf("[%s] Submitting data %d to contract %s at %s\n", 
		n.NodeID, data, contractAddress, time.Now().Format(time.RFC3339))
	return nil
}
