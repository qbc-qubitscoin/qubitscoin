package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/config"
)

func TestDefault_Fields(t *testing.T) {
	cfg := config.Default()

	if cfg.Chain.NetworkID != 1 {
		t.Errorf("NetworkID: want 1, got %d", cfg.Chain.NetworkID)
	}
	if cfg.Node.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.P2P.MaxPeers <= 0 {
		t.Error("MaxPeers should be positive")
	}
	if cfg.RPC.ListenAddr == "" {
		t.Error("RPC.ListenAddr should not be empty")
	}
	if cfg.Upgrade.CheckInterval.Duration < time.Minute {
		t.Error("Upgrade.CheckInterval should be >= 1 minute")
	}
}

func TestTestnetDefault_NetworkID(t *testing.T) {
	cfg := config.TestnetDefault()
	if cfg.Chain.NetworkID != 2 {
		t.Errorf("testnet NetworkID: want 2, got %d", cfg.Chain.NetworkID)
	}
}

func TestLoad_FromFile(t *testing.T) {
	const toml = `
[chain]
network_id = 42
name = "MyNet"

[node]
data_dir = "/tmp/mynode"
miner_enabled = true
log_level = "debug"

[p2p]
listen_addr = "0.0.0.0:9999"
max_peers = 10

[rpc]
enabled = true
listen_addr = "0.0.0.0:8080"

[upgrade]
check_interval = "30m"
`
	path := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Chain.NetworkID != 42 {
		t.Errorf("NetworkID: want 42, got %d", cfg.Chain.NetworkID)
	}
	if cfg.Chain.Name != "MyNet" {
		t.Errorf("Name: want MyNet, got %q", cfg.Chain.Name)
	}
	if !cfg.Node.MinerEnabled {
		t.Error("MinerEnabled should be true")
	}
	if cfg.Node.LogLevel != "debug" {
		t.Errorf("LogLevel: want debug, got %q", cfg.Node.LogLevel)
	}
	if cfg.P2P.ListenAddr != "0.0.0.0:9999" {
		t.Errorf("P2P.ListenAddr: want 0.0.0.0:9999, got %q", cfg.P2P.ListenAddr)
	}
	if cfg.P2P.MaxPeers != 10 {
		t.Errorf("MaxPeers: want 10, got %d", cfg.P2P.MaxPeers)
	}
	if cfg.Upgrade.CheckInterval.Duration != 30*time.Minute {
		t.Errorf("CheckInterval: want 30m, got %v", cfg.Upgrade.CheckInterval.Duration)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for a missing file, got nil")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	_ = os.WriteFile(path, []byte("not valid toml ][[["), 0o600)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}
