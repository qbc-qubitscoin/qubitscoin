package rpc_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func buildRealServer(t *testing.T) (*httptest.Server, *rpc.Server) {
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
	srv := rpc.NewServer("127.0.0.1:0", api, 5*time.Second, 5*time.Second)
	
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, srv
}

func TestRPCServer_HTTPError(t *testing.T) {
	ts, _ := buildRealServer(t)
	// Test GET request
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}

	// Test GET /ui/ returns 200 OK
	uiResp, err := http.Get(ts.URL + "/ui/")
	if err != nil {
		t.Fatal(err)
	}
	if uiResp.StatusCode != http.StatusOK {
		t.Errorf("GET /ui/: want 200, got %d", uiResp.StatusCode)
	}

	// Test GET /dashboard redirects to /ui/
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirect so we can inspect status
		},
	}
	dashResp, err := client.Get(ts.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if dashResp.StatusCode != http.StatusFound {
		t.Errorf("GET /dashboard: want 302, got %d", dashResp.StatusCode)
	}
	
	// Test bad JSON
	resp2, _ := http.Post(ts.URL, "application/json", bytes.NewReader([]byte("{bad json")))
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp2.StatusCode) // rpc server returns 200 with error JSON
	}
	var out map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&out)
	if out["error"] == nil {
		t.Errorf("expected error for bad JSON")
	}
}

func TestRPCServer_Batch(t *testing.T) {
	ts, _ := buildRealServer(t)
	req := []map[string]interface{}{
		{"jsonrpc": "2.0", "id": 1, "method": "qbc_chainInfo"},
		{"jsonrpc": "2.0", "id": 2, "method": "qbc_gasPrice"},
	}
	body, _ := json.Marshal(req)
	resp, _ := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	var out []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out) != 2 {
		t.Errorf("expected 2 responses, got %d", len(out))
	}
}

func TestRPCServer_Batch_Misc(t *testing.T) {
	ts, _ := buildRealServer(t)
	// Test empty batch
	req := []map[string]interface{}{}
	body, _ := json.Marshal(req)
	resp, _ := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	var out map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out)
	if out["error"] == nil {
		t.Errorf("expected error for empty batch")
	}
	
	// Test batch with invalid inner request and one error request
	body2 := []byte(`[123, {"jsonrpc": "2.0", "id": 2, "method": "qbc_unknown"}]`)
	resp2, _ := http.Post(ts.URL, "application/json", bytes.NewReader(body2))
	var out2 []map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&out2)
	if len(out2) != 2 || out2[0]["error"] == nil || out2[1]["error"] == nil {
		t.Errorf("expected error for invalid inner request")
	}

	// Test bad batch json (starts with [, but invalid)
	body3 := []byte(`[bad json`)
	resp3, _ := http.Post(ts.URL, "application/json", bytes.NewReader(body3))
	var out3 map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&out3)
	if out3["error"] == nil {
		t.Errorf("expected error for bad batch json")
	}
}

// errReader simulates a read error
type errReader struct{}
func (errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}
func (errReader) Close() error { return nil }

func TestRPCServer_ReadError(t *testing.T) {
	_, srv := buildRealServer(t)
	req := httptest.NewRequest("POST", "/", errReader{})
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 200 { // rpc returns 200 with error JSON
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRPCServer_ZeroTimeouts(t *testing.T) {
	srv := rpc.NewServer("127.0.0.1:0", nil, 0, 0)
	if srv == nil {
		t.Errorf("expected server")
	}
}

func TestRPCError_Error(t *testing.T) {
	err := &rpc.RPCError{Code: 123, Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestRPCServer_Healthz(t *testing.T) {
	ts, _ := buildRealServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRPCServer_Start(t *testing.T) {
	_, srv := buildRealServer(t)
	// We can't use ts.URL because it uses httptest.Server, but we can call Start on srv directly.
	// But it will try to listen on the addr.
	// We gave it "127.0.0.1:0" so it should bind to a random port.
	ctx, cancel := context.WithCancel(context.Background())
	srv.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Let it start
	cancel()                          // Shutdown
	time.Sleep(50 * time.Millisecond) // Let it stop
}

func TestRPCServer_InvalidJSONRPC2(t *testing.T) {
	ts, _ := buildRealServer(t)
	req := map[string]interface{}{
		"jsonrpc": "1.0", "id": 1, "method": "qbc_chainInfo",
	}
	body, _ := json.Marshal(req)
	resp, _ := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	var out map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out)
	if out["error"] == nil {
		t.Errorf("expected error for missing JSONRPC 2.0")
	}
	
	// Test empty body
	resp2, _ := http.Post(ts.URL, "application/json", bytes.NewReader([]byte("   \n ")))
	var out2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&out2)
	if out2["error"] == nil {
		t.Errorf("expected error for empty body")
	}
}

func TestRPC_BlockByHash(t *testing.T) {
	ts, _ := buildTestHandler(t) // we can use the handler from rpc_test.go
	// bad param
	resp := rpcCallErr(t, ts, "qbc_blockByHash", []string{"nothex"})
	if resp == nil {
		t.Errorf("expected error")
	}
	
	// good param but not found
	hash := crypto.Hash256([]byte("dummy"))
	_ = rpcCallErr(t, ts, "qbc_blockByHash", []string{crypto.ToHex(hash)})
}

func TestRPC_MissingParams(t *testing.T) {
	ts, _ := buildTestHandler(t)
	methods := []string{
		"qbc_blockByHeight",
		"qbc_blockByHash",
		"qbc_getBalance",
		"qbc_getTransactionCount",
		"qbc_sendRawTransaction",
	}
	for _, m := range methods {
		if err := rpcCallErr(t, ts, m, nil); err == nil {
			t.Errorf("expected error for missing params in %s", m)
		}
	}
}

func TestRPC_BlockByHeight_BadParams(t *testing.T) {
	ts, _ := buildTestHandler(t)
	resp := rpcCallErr(t, ts, "qbc_blockByHeight", []string{"notanumber"})
	if resp == nil {
		t.Errorf("expected error")
	}
	// Test out of bounds
	resp2 := rpcCallErr(t, ts, "qbc_blockByHeight", []uint64{9999})
	if resp2 == nil {
		t.Errorf("expected error for not found")
	}
}

func TestRPC_SendRawTransaction_Success(t *testing.T) {
	ts, w := buildTestHandler(t)
	tx := core.NewTransfer(w.Address, w.Address, w.PublicKey, 1, 100, core.MinGasPrice)
	tx.Sign(w.PrivateKey)
	raw, _ := rpc.GobEncodeTx(tx)
	
	resp := rpcCallErr(t, ts, "qbc_sendRawTransaction", []string{hex.EncodeToString(raw)})
	if resp != nil {
		t.Errorf("expected success, got error: %v", resp)
	}
}

func TestRPC_BlockByHash_Success(t *testing.T) {
	ts, _ := buildTestHandler(t)
	// get height 0 hash first
	resp := rpcCall(t, ts, "qbc_blockByHeight", []uint64{0})
	hash := resp["hash"].(string)
	
	resp2 := rpcCall(t, ts, "qbc_blockByHash", []string{hash})
	if resp2["hash"] != hash {
		t.Errorf("expected hash %s, got %v", hash, resp2["hash"])
	}
}

func TestRPC_GetBalance_BadParams(t *testing.T) {
	ts, _ := buildTestHandler(t)
	resp := rpcCallErr(t, ts, "qbc_getBalance", []string{"nothex"})
	if resp == nil {
		t.Errorf("expected error")
	}
}

func TestRPC_GetTransactionCount_BadParams(t *testing.T) {
	ts, _ := buildTestHandler(t)
	resp := rpcCallErr(t, ts, "qbc_getTransactionCount", []string{"nothex"})
	if resp == nil {
		t.Errorf("expected error")
	}
}

// Minimal RPC Error struct for decoding
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func rpcCallErr(t *testing.T, ts *httptest.Server, method string, params interface{}) *rpcError {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Error
}

func TestRPC_SendRawTransaction(t *testing.T) {
	ts, _ := buildTestHandler(t)
	
	err := rpcCallErr(t, ts, "qbc_sendRawTransaction", []string{"badhex"})
	if err == nil {
		t.Errorf("expected error for bad hex")
	}
	
	// test decode tx fail
	err2 := rpcCallErr(t, ts, "qbc_sendRawTransaction", []string{"000000"})
	if err2 == nil {
		t.Errorf("expected error for bad gob")
	}
	
	// test pool reject (invalid signature)
	tx := &core.Transaction{Nonce: 1}
	raw, _ := rpc.GobEncodeTx(tx) // Also tests GobEncodeTx
	err3 := rpcCallErr(t, ts, "qbc_sendRawTransaction", []string{hex.EncodeToString(raw)})
	if err3 == nil {
		t.Errorf("expected error for pool reject")
	}
}

func TestRPC_FeeEstimate(t *testing.T) {
	ts, _ := buildTestHandler(t)
	
	err := rpcCallErr(t, ts, "qbc_feeEstimate", []int{0})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
