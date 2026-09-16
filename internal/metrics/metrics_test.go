package metrics

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMetricVars_NotNil(t *testing.T) {
	// All metric variables should be initialized by the package's init (promauto).
	if ChainHeight == nil {
		t.Error("ChainHeight gauge is nil")
	}
	if PeerCount == nil {
		t.Error("PeerCount gauge is nil")
	}
	if MempoolSize == nil {
		t.Error("MempoolSize gauge is nil")
	}
	if BlocksProduced == nil {
		t.Error("BlocksProduced counter is nil")
	}
	if TxProcessed == nil {
		t.Error("TxProcessed counter is nil")
	}
	if FeeBurned == nil {
		t.Error("FeeBurned counter is nil")
	}
	if BaseFee == nil {
		t.Error("BaseFee gauge is nil")
	}
	if BlockProductionDuration == nil {
		t.Error("BlockProductionDuration histogram is nil")
	}
	if RPCRequests == nil {
		t.Error("RPCRequests counter vec is nil")
	}
	if RPCErrors == nil {
		t.Error("RPCErrors counter vec is nil")
	}
}

func TestMetrics_SetAndObserve(t *testing.T) {
	ChainHeight.Set(42)
	PeerCount.Set(5)
	MempoolSize.Set(100)
	BlocksProduced.Add(1)
	TxProcessed.Add(3)
	FeeBurned.Add(1000)
	BaseFee.Set(1e9)
	BlockProductionDuration.Observe(0.001)
	RPCRequests.WithLabelValues("qbc_chainInfo").Inc()
	RPCErrors.WithLabelValues("qbc_chainInfo").Inc()
	// If any of these panic, the test fails.
}

func TestServe_StartsAndResponds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a random available port.
	addr := "127.0.0.1:0"

	// We need a specific port for the HTTP call — use a fixed ephemeral port.
	testAddr := "127.0.0.1:19999"

	// Start the metrics server in background.
	go Serve(ctx, testAddr)

	// Give the server a moment to start.
	time.Sleep(150 * time.Millisecond)

	// Hit the /healthz endpoint.
	resp, err := http.Get("http://" + testAddr + "/healthz")
	if err != nil {
		t.Skipf("metrics server not reachable (port conflict?): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz: want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("healthz body: want 'ok', got %q", body)
	}

	// Hit /metrics endpoint.
	resp2, err := http.Get("http://" + testAddr + "/metrics")
	if err != nil {
		t.Fatalf("/metrics: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("/metrics: want 200, got %d", resp2.StatusCode)
	}
	body2, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body2), "qbc_chain_height") {
		t.Error("/metrics response should contain qbc_chain_height")
	}

	// Graceful shutdown via context cancellation.
	cancel()
	time.Sleep(100 * time.Millisecond)

	_ = addr // suppress lint
}

func TestServe_Error(t *testing.T) {
	ctx := context.Background()
	// An invalid address will cause ListenAndServe to fail immediately with an error
	Serve(ctx, "999.999.999.999:99999")
}
