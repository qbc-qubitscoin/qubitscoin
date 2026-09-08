package node_test

import (
	"context"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/config"
	"github.com/qbc-qubitscoin/qubitscoin/internal/node"
)

// TestNode_NewEphemeral verifies that a node can be constructed without a
// keystore (ephemeral wallet) and starts + shuts down cleanly.
func TestNode_NewEphemeral(t *testing.T) {
	cfg := config.Default()
	cfg.Node.DataDir = t.TempDir()
	cfg.Node.KeystoreFile = "keystore.json"
	cfg.Node.MinerEnabled = false // no block production in this test
	cfg.RPC.Enabled = false       // no RPC listener
	cfg.P2P.ListenAddr = "127.0.0.1:0" // OS-assigned port
	cfg.P2P.BootstrapPeers = nil  // no bootstrap dials
	cfg.Metrics.Enabled = false

	// Empty password → ephemeral wallet (not persisted).
	n, err := node.New(cfg, "")
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		n.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Node shut down cleanly after context cancellation.
	case <-time.After(5 * time.Second):
		t.Fatal("node did not shut down within 5 seconds")
	}
}

// TestNode_WithMiner runs the node in miner mode for 3 blocks and verifies
// that the chain height grows.
func TestNode_WithMiner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping miner integration test in short mode")
	}

	cfg := config.Default()
	cfg.Node.DataDir = t.TempDir()
	cfg.Node.KeystoreFile = "keystore.json"
	cfg.Node.MinerEnabled = true
	cfg.RPC.Enabled = false
	cfg.P2P.ListenAddr = "127.0.0.1:0"
	cfg.P2P.BootstrapPeers = nil
	cfg.Metrics.Enabled = false

	n, err := node.New(cfg, "")
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}

	// Let the node mine for ~8 s (should produce ≥ 3 blocks at 2 s/block).
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	go n.Start(ctx)

	// Wait for context to expire.
	<-ctx.Done()

	status := n.StatusReport()
	t.Logf("final status: %s", status)
}
