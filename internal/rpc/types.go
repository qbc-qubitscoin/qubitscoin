// Package rpc provides a JSON-RPC 2.0 HTTP API for the QBC node.
package rpc

import "encoding/json"

// ─────────────────────────────────────────────────────────────────────────────
// JSON-RPC 2.0 wire types
// ─────────────────────────────────────────────────────────────────────────────

// Request is an inbound JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response is an outbound JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError carries a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

func errResponse(id json.RawMessage, code int, msg string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
}

func okResponse(id json.RawMessage, result interface{}) *Response {
	return &Response{JSONRPC: "2.0", ID: id, Result: result}
}

// ─────────────────────────────────────────────────────────────────────────────
// Domain-level response types
// ─────────────────────────────────────────────────────────────────────────────

// BlockInfo is the JSON representation of a block header returned by the API.
type BlockInfo struct {
	Height        uint64 `json:"height"`
	Hash          string `json:"hash"`
	PrevHash      string `json:"prev_hash"`
	StateRoot     string `json:"state_root"`
	MerkleRoot    string `json:"merkle_root"`
	Timestamp     int64  `json:"timestamp"`
	ValidatorAddr string `json:"validator_addr"`
	GasUsed       uint64 `json:"gas_used"`
	GasLimit      uint64 `json:"gas_limit"`
	BaseFee       uint64 `json:"base_fee"`
	BurnedFees    uint64 `json:"burned_fees"`
	TxCount       int    `json:"tx_count"`
}

// TxInfo is the JSON representation of a transaction returned by the API.
type TxInfo struct {
	Hash      string `json:"hash"`
	Type      uint8  `json:"type"`
	Version   uint8  `json:"version"`
	Nonce     uint64 `json:"nonce"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	GasLimit  uint64 `json:"gas_limit"`
	GasPrice  uint64 `json:"gas_price"`
	Timestamp int64  `json:"timestamp"`
	DataHex   string `json:"data,omitempty"`
}

// ChainInfo carries high-level node status.
type ChainInfo struct {
	ChainID    uint32 `json:"chain_id"`
	NetworkID  uint32 `json:"network_id"`
	Height     uint64 `json:"height"`
	TipHash    string `json:"tip_hash"`
	PeerCount  int    `json:"peer_count"`
	MempoolLen int    `json:"mempool_len"`
	Version    string `json:"version"`
}

// FeeEstimateResult holds the fee estimate for each tier.
type FeeEstimateResult struct {
	BaseFee         uint64 `json:"base_fee"`
	UltraLowTip     uint64 `json:"ultra_low_tip"`
	StandardTip     uint64 `json:"standard_tip"`
	FastTip         uint64 `json:"fast_tip"`
	TransferUltraLow uint64 `json:"transfer_ultra_low_qubits"`
	TransferStandard uint64 `json:"transfer_standard_qubits"`
	TransferFast     uint64 `json:"transfer_fast_qubits"`
}
