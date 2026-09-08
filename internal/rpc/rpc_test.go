package rpc_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/rpc"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
)

// buildTestServer creates a fully wired RPC server backed by a minimal in-memory engine.
func buildTestServer(t *testing.T) (*httptest.Server, *crypto.Wallet) {
	t.Helper()

	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}

	// Bootstrap genesis.
	genCfg := core.DefaultGenesisConfig(w.Address)
	genesis := genCfg.Build()

	st := state.NewStateDB()
	for addr, bal := range genCfg.Allocations {
		st.SetAccount(addr, &state.Account{Balance: bal})
	}

	pool := mempool.New(0)

	validator := &consensus.Validator{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 1}
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{validator})
	if err != nil {
		t.Fatalf("ValidatorSet: %v", err)
	}
	engine := consensus.NewEngine(
		w.Address, w.PublicKey, w.PrivateKey,
		vs, st, pool, genesis, nil, nil,
	)

	api := rpc.NewAPI(engine, st, pool, nil, "test")
	srv := rpc.NewServer("", api, 5*time.Second, 5*time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Inline the handleRPC logic by using the exported handler.
		srv.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts, w
}

func rpcPost(t *testing.T, ts *httptest.Server, method string, params interface{}) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("WARN: could not close HTTP response body: %v\n", err)
		}
	}(resp.Body)
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("RPC error %d: %s", out.Error.Code, out.Error.Message)
	}
	var result map[string]interface{}
	_ = json.Unmarshal(out.Result, &result)
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Server needs a ServeHTTP method — wrap NewServer to expose it.
// ─────────────────────────────────────────────────────────────────────────────

// serveHTTPHandler wraps the API directly for testing.
type serveHTTPHandler struct{ api *rpc.API }

func (h *serveHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method isn't allowed", http.StatusMethodNotAllowed)
		return
	}
	var req rpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": nil,
			"error": map[string]interface{}{"code": -32700, "message": err.Error()},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.api.Dispatch(&req))
}

func buildTestHandler(t *testing.T) (*httptest.Server, *crypto.Wallet) {
	t.Helper()
	w, _ := crypto.NewWallet()
	genCfg := core.DefaultGenesisConfig(w.Address)
	genesis := genCfg.Build()
	st := state.NewStateDB()
	for addr, bal := range genCfg.Allocations {
		st.SetAccount(addr, &state.Account{Balance: bal})
	}
	pool := mempool.New(0)
	validator := &consensus.Validator{Address: w.Address, PublicKey: w.PublicKey, VotingPower: 1}
	vs, _ := consensus.NewValidatorSet([]*consensus.Validator{validator})
	engine := consensus.NewEngine(w.Address, w.PublicKey, w.PrivateKey, vs, st, pool, genesis, nil, nil)
	api := rpc.NewAPI(engine, st, pool, nil, "test")
	ts := httptest.NewServer(&serveHTTPHandler{api: api})
	t.Cleanup(ts.Close)
	return ts, w
}

func rpcCall(t *testing.T, ts *httptest.Server, method string, params interface{}) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("WARN: could not close HTTP response body: %v\n", err)
		}
	}(resp.Body)
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Error != nil {
		t.Fatalf("RPC error %d: %s", out.Error.Code, out.Error.Message)
	}
	var result map[string]interface{}
	_ = json.Unmarshal(out.Result, &result)
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestRPC_BlockHeight(t *testing.T) {
	ts, _ := buildTestHandler(t)
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "qbc_blockHeight", "params": nil,
	})
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("WARN: could not close HTTP response body: %v\n", err)
		}
	}(resp.Body)
	var out struct {
		Result float64 `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	// Genesis engine starts at height 1 (genesis at 0, chain length = 1).
	if out.Result < 1 {
		t.Errorf("expected height >= 1, got %v", out.Result)
	}
}

func TestRPC_GetBalance(t *testing.T) {
	ts, w := buildTestHandler(t)
	result := rpcCall(t, ts, "qbc_getBalance", []string{crypto.AddressToHex(w.Address)})
	bal, ok := result["balance_qubits"].(float64)
	if !ok {
		t.Fatalf("missing balance_qubits in %v", result)
	}
	if bal <= 0 {
		t.Errorf("expected positive balance, got %v", bal)
	}
}

func TestRPC_GetTransactionCount(t *testing.T) {
	ts, w := buildTestHandler(t)
	result := rpcCall(t, ts, "qbc_getTransactionCount", []string{crypto.AddressToHex(w.Address)})
	_, ok := result["nonce"]
	if !ok {
		t.Fatalf("missing nonce in %v", result)
	}
}

func TestRPC_ChainInfo(t *testing.T) {
	ts, _ := buildTestHandler(t)
	result := rpcCall(t, ts, "qbc_chainInfo", nil)
	if _, ok := result["height"]; !ok {
		t.Errorf("missing height in chain info: %v", result)
	}
	if _, ok := result["chain_id"]; !ok {
		t.Errorf("missing chain_id in chain info: %v", result)
	}
}

func TestRPC_BlockByHeight(t *testing.T) {
	ts, _ := buildTestHandler(t)
	result := rpcCall(t, ts, "qbc_blockByHeight", []uint64{0})
	if _, ok := result["hash"]; !ok {
		t.Errorf("missing hash in block: %v", result)
	}
}

func TestRPC_GasPrice(t *testing.T) {
	ts, _ := buildTestHandler(t)
	result := rpcCall(t, ts, "qbc_gasPrice", nil)
	if _, ok := result["base_fee"]; !ok {
		t.Errorf("missing base_fee in gas price: %v", result)
	}
}

func TestRPC_MethodNotFound(t *testing.T) {
	ts, _ := buildTestHandler(t)
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "qbc_nonExistent", "params": nil,
	})
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("WARN: could not close HTTP response body: %v\n", err)
		}
	}(resp.Body)
	var out struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Error == nil || out.Error.Code != -32601 {
		t.Errorf("expected -32601 MethodNotFound, got %+v", out.Error)
	}
}
