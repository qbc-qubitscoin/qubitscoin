// Package config provides TOML-based configuration for the QBC node.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the root configuration structure; maps directly to a TOML file.
type Config struct {
	Chain   ChainConfig   `toml:"chain"`
	Node    NodeConfig    `toml:"node"`
	P2P     P2PConfig     `toml:"p2p"`
	RPC     RPCConfig     `toml:"rpc"`
	Storage StorageConfig `toml:"storage"`
	Metrics MetricsConfig `toml:"metrics"`
	Upgrade UpgradeConfig `toml:"upgrade"`
}

// ChainConfig holds chain-identity settings (cannot be changed after genesis).
type ChainConfig struct {
	// NetworkID distinguishes mainnet (1) from testnet (2) etc.
	NetworkID uint32 `toml:"network_id"`
	// Name is a human-readable label displayed in logs.
	Name string `toml:"name"`
}

// NodeConfig holds validator and key-management settings.
type NodeConfig struct {
	// DataDir is the root directory for all persistent data.
	DataDir string `toml:"data_dir"`
	// KeystoreFile is the path to the encrypted wallet file.
	// Relative paths are resolved against DataDir.
	KeystoreFile string `toml:"keystore_file"`
	// MinerEnabled enables or disables block production.
	MinerEnabled bool `toml:"miner_enabled"`
	// LogLevel controls logging verbosity: "debug", "info", "warn", "error".
	LogLevel string `toml:"log_level"`
}

// P2PConfig controls the libp2p / TCP networking layer.
type P2PConfig struct {
	// ListenAddr is the TCP address the node listens on, e.g. "0.0.0.0:8765".
	ListenAddr string `toml:"listen_addr"`
	// ExternalAddr is the externally reachable address announced to peers.
	// Leave empty to derive from ListenAddr.
	ExternalAddr string `toml:"external_addr"`
	// BootstrapPeers is a list of initial peers to dial on startup.
	BootstrapPeers []string `toml:"bootstrap_peers"`
	// MaxPeers is the upper bound on simultaneous peer connections.
	MaxPeers int `toml:"max_peers"`
}

// RPCConfig configures the JSON-RPC HTTP server.
type RPCConfig struct {
	// Enabled toggles the RPC server.
	Enabled bool `toml:"enabled"`
	// ListenAddr is the HTTP bind address, e.g. "127.0.0.1:8545".
	ListenAddr string `toml:"listen_addr"`
	// CORSOrigins is a list of allowed CORS origins (empty = all).
	CORSOrigins []string `toml:"cors_origins"`
	// ReadTimeout is the per-request read deadline.
	ReadTimeout duration `toml:"read_timeout"`
	// WriteTimeout is the per-request write deadline.
	WriteTimeout duration `toml:"write_timeout"`
}

// StorageConfig points to the on-disk databases.
type StorageConfig struct {
	// BlocksDir is the LevelDB directory for block storage.
	BlocksDir string `toml:"blocks_dir"`
	// StateDir is the LevelDB directory for account-state storage.
	StateDir string `toml:"state_dir"`
}

// MetricsConfig controls the Prometheus /metrics endpoint.
type MetricsConfig struct {
	// Enabled toggles the metrics endpoint.
	Enabled bool `toml:"enabled"`
	// ListenAddr is the bind address, e.g. "0.0.0.0:9090".
	ListenAddr string `toml:"listen_addr"`
}

// UpgradeConfig controls the auto-upgrade manager.
type UpgradeConfig struct {
	// ReleaseURL is the GitHub Releases API endpoint for version checks.
	ReleaseURL string `toml:"release_url"`
	// CheckInterval is how frequent the node polls for new releases.
	CheckInterval duration `toml:"check_interval"`
	// AutoApply: if true the node downloads and restarts automatically.
	AutoApply bool `toml:"auto_apply"`
}

// ─────────────────────────────────────────────────────────────────────────────
// duration is a TOML-friendly time.Duration (stored as a string like "1h").
// ─────────────────────────────────────────────────────────────────────────────

type duration struct{ time.Duration }

func (d *duration) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

func (d duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Load reads and parses a TOML config file. Missing fields fall back to
// defaults provided by Default().
// ─────────────────────────────────────────────────────────────────────────────

// Load reads a TOML file at a path and merges it over Default().
func Load(path string) (*Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %s: %w", path, err)
	}
	defer f.Close()
	if _, err := toml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Default returns a Config pre-filled with sane production defaults.
func Default() *Config {
	return &Config{
		Chain: ChainConfig{
			NetworkID: 1,
			Name:      "QubitsCoin Mainnet",
		},
		Node: NodeConfig{
			DataDir:      "~/.qbc",
			KeystoreFile: "keystore.json",
			MinerEnabled: false,
			LogLevel:     "info",
		},
		P2P: P2PConfig{
			ListenAddr: "0.0.0.0:8765",
			MaxPeers:   25,
			BootstrapPeers: []string{
				"qbc-seed1.qubitscoin.io:8765",
				"qbc-seed2.qubitscoin.io:8765",
			},
		},
		RPC: RPCConfig{
			Enabled:      true,
			ListenAddr:   "127.0.0.1:8545",
			ReadTimeout:  duration{30 * time.Second},
			WriteTimeout: duration{30 * time.Second},
		},
		Storage: StorageConfig{
			BlocksDir: "blocks",
			StateDir:  "state",
		},
		Metrics: MetricsConfig{
			Enabled:    false,
			ListenAddr: "0.0.0.0:9090",
		},
		Upgrade: UpgradeConfig{
			ReleaseURL:    "https://api.github.com/repos/qbc-qubitscoin/qubitscoin/releases/latest",
			CheckInterval: duration{time.Hour},
			AutoApply:     false,
		},
	}
}

// TestnetDefault returns config pre-filled for the public testnet.
func TestnetDefault() *Config {
	cfg := Default()
	cfg.Chain.NetworkID = 2
	cfg.Chain.Name = "QubitsCoin Testnet"
	cfg.P2P.BootstrapPeers = []string{
		"qbc-testnet-seed1.qubitscoin.io:8765",
		"qbc-testnet-seed2.qubitscoin.io:8765",
	}
	cfg.Node.DataDir = "~/.qbc-testnet"
	return cfg
}
