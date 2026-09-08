// cmd/node is the QubitsCoin node binary.
//
// Usage:
//
//	qbc-node start # run a full node
//	qbc-node wallet new # generate a new wallet
//	qbc-node wallet show # display the address in the keystore
//	qbc-node tx send --to <addr> --amount X # broadcast a signed transfer
//	qbc-node query balance --address <addr> # query an address balance via RPC
//	qbc-node query chain # query chain status via RPC
//	qbc-node version # print version information
package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/qbc-qubitscoin/qubitscoin/internal/config"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/keystore"
	"github.com/qbc-qubitscoin/qubitscoin/internal/node"
	"github.com/qbc-qubitscoin/qubitscoin/internal/upgrade"
)

// ─────────────────────────────────────────────────────────────────────────────
// Root command
// ─────────────────────────────────────────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "qbc-node",
	Short: "QubitsCoin (QBC) full node — quantum-resistant Layer-1 blockchain",
	Long: `QubitsCoin (QBC) is a quantum-resistant Layer-1 blockchain.

Cryptography: ML-DSA-65 (FIPS 204) | ML-KEM-768 (FIPS 203) | SHA-3-256 (FIPS 202) | AES-256-GCM
Fee model: EIP-1559 ultra-low (burn + tip)
Block time: 2 seconds
Hard cap: 100,000,000 QBC`,
}

// ─────────────────────────────────────────────────────────────────────────────
// Shared flags
// ─────────────────────────────────────────────────────────────────────────────

var (
	flagConfig   string
	flagDataDir  string
	flagPassword string
	flagRPCAddr  string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "path to a TOML config file")
	rootCmd.PersistentFlags().StringVar(&flagDataDir, "datadir", "", "override data directory (default: ~/.qbc)")
	rootCmd.PersistentFlags().StringVar(&flagPassword, "password", "", "keystore password (avoid using on CLI; prefer QBC_PASSWORD env var)")
	rootCmd.PersistentFlags().StringVar(&flagRPCAddr, "rpc", "http://127.0.0.1:8545", "RPC endpoint for query/tx commands")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(walletCmd)
	rootCmd.AddCommand(txCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(versionCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// qbc-node start
// ─────────────────────────────────────────────────────────────────────────────

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the full node",
	Long: `Start a QubitsCoin full node.

The node will:
  • Connect to bootstrap peers and sync the chain
  • Validate and relay transactions and blocks
  • Optionally mine blocks (--miner)
  • Serve a JSON-RPC API (--rpc-listen)
  • Expose Prometheus metrics (--metrics)`,
	RunE: runStart,
}

var (
	flagMiner      bool
	flagRPCListen  string
	flagP2PListen  string
	flagMetrics    bool
	flagMetricAddr string
	flagTestnet    bool
)

func init() {
	startCmd.Flags().BoolVar(&flagMiner, "miner", false, "enable block production (validator mode)")
	startCmd.Flags().StringVar(&flagRPCListen, "rpc-listen", "", "JSON-RPC listen address (default: 127.0.0.1:8545)")
	startCmd.Flags().StringVar(&flagP2PListen, "p2p-listen", "", "P2P TCP listen address (default: 0.0.0.0:8765)")
	startCmd.Flags().BoolVar(&flagMetrics, "metrics", false, "enable Prometheus /metrics endpoint")
	startCmd.Flags().StringVar(&flagMetricAddr, "metrics-addr", "0.0.0.0:9090", "Prometheus listen address")
	startCmd.Flags().BoolVar(&flagTestnet, "testnet", false, "use testnet defaults")
}

func runStart(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	// CLI flag overrides.
	if flagDataDir != "" {
		cfg.Node.DataDir = flagDataDir
	}
	if flagMiner {
		cfg.Node.MinerEnabled = true
	}
	if flagRPCListen != "" {
		cfg.RPC.ListenAddr = flagRPCListen
	}
	if flagP2PListen != "" {
		cfg.P2P.ListenAddr = flagP2PListen
	}
	if flagMetrics {
		cfg.Metrics.Enabled = true
		if flagMetricAddr != "" {
			cfg.Metrics.ListenAddr = flagMetricAddr
		}
	}

	password := resolvePassword()

	n, err := node.New(cfg, password)
	if err != nil {
		return fmt.Errorf("init node: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	printBanner()
	n.Start(ctx)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// qbc-node wallet
// ─────────────────────────────────────────────────────────────────────────────

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Wallet management commands",
}

var walletNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a new ML-DSA-65 wallet and save an encrypted keystore file",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if flagDataDir != "" {
			cfg.Node.DataDir = flagDataDir
		}

		password := resolvePassword()
		if password == "" {
			return fmt.Errorf("password required (--password or QBC_PASSWORD env var)")
		}

		ksPath := resolvePath(cfg.Node.DataDir, cfg.Node.KeystoreFile)
		if _, statErr := os.Stat(ksPath); statErr == nil {
			return fmt.Errorf("keystore already exists at %s — delete it first if you want a new wallet", ksPath)
		}

		w, err := crypto.NewWallet()
		if err != nil {
			return fmt.Errorf("generate wallet: %w", err)
		}
		if err := keystore.Encrypt(ksPath, password, w); err != nil {
			return fmt.Errorf("save keystore: %w", err)
		}

		fmt.Printf("✓ New wallet created\n")
		fmt.Printf("  Address: %s\n", crypto.AddressToHex(w.Address))
		fmt.Printf("  Keystore: %s\n", ksPath)
		fmt.Println()
		fmt.Println("⚠ Back up your keystore file and remember your password.")
		fmt.Println("   There is NO recovery mechanism — lost keys = lost funds.")
		return nil
	},
}

var walletShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the address stored in the keystore file",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if flagDataDir != "" {
			cfg.Node.DataDir = flagDataDir
		}
		ksPath := resolvePath(cfg.Node.DataDir, cfg.Node.KeystoreFile)
		addr, err := keystore.PeekAddress(ksPath)
		if err != nil {
			return fmt.Errorf("read keystore %s: %w", ksPath, err)
		}
		fmt.Printf("Address: %s\n", addr)
		fmt.Printf("Keystore: %s\n", ksPath)
		return nil
	},
}

func init() {
	walletCmd.AddCommand(walletNewCmd)
	walletCmd.AddCommand(walletShowCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// qbc-node tx send
// ─────────────────────────────────────────────────────────────────────────────

var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Transaction commands",
}

var (
	flagTxTo     string
	flagTxAmount uint64
	flagTxNonce  uint64
)

var txSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Sign and broadcast a QBC transfer",
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagTxTo == "" {
			return fmt.Errorf("--to address required")
		}
		if flagTxAmount == 0 {
			return fmt.Errorf("--amount must be > 0 (in qubits)")
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if flagDataDir != "" {
			cfg.Node.DataDir = flagDataDir
		}

		password := resolvePassword()
		if password == "" {
			return fmt.Errorf("password required (--password or QBC_PASSWORD env var)")
		}

		ksPath := resolvePath(cfg.Node.DataDir, cfg.Node.KeystoreFile)
		w, err := keystore.Decrypt(ksPath, password)
		if err != nil {
			return fmt.Errorf("unlock wallet: %w", err)
		}

		toAddr, err := crypto.HexToAddress(flagTxTo)
		if err != nil {
			return fmt.Errorf("invalid --to address: %w", err)
		}

		// Fetch current base fee for gas price.
		baseFee, err := fetchBaseFee(flagRPCAddr)
		if err != nil {
			log.Printf("[warn] could not fetch base fee: %v — using InitialBaseFee=%d", err, core.InitialBaseFee)
			baseFee = core.InitialBaseFee
		}

		tx := core.NewTransfer(w.Address, toAddr, w.PublicKey, flagTxNonce, flagTxAmount, baseFee)
		if err := tx.Sign(w.PrivateKey); err != nil {
			return fmt.Errorf("sign tx: %w", err)
		}

		raw, err := gobEncodeTx(tx)
		if err != nil {
			return fmt.Errorf("encode tx: %w", err)
		}
		rawHex := hex.EncodeToString(raw)

		result, err := rpcCall(flagRPCAddr, "qbc_sendRawTransaction", []string{rawHex})
		if err != nil {
			return fmt.Errorf("broadcast: %w", err)
		}

		fmt.Printf("✓ Transaction broadcast\n")
		fmt.Printf("  Hash : %s\n", extractString(result, "tx_hash"))
		fmt.Printf("  From : %s\n", crypto.AddressToHex(w.Address))
		fmt.Printf("  To   : %s\n", flagTxTo)
		fmt.Printf("  Amt  : %d qubits (%g QBC)\n", flagTxAmount, float64(flagTxAmount)/float64(core.OneQBC))
		return nil
	},
}

func init() {
	txSendCmd.Flags().StringVar(&flagTxTo, "to", "", "destination address (hex)")
	txSendCmd.Flags().Uint64Var(&flagTxAmount, "amount", 0, "amount in qubits (1 QBC = 1 000 000 000 qubits)")
	txSendCmd.Flags().Uint64Var(&flagTxNonce, "nonce", 0, "sender nonce (use qbc-node query nonce to get the current value)")
	txCmd.AddCommand(txSendCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// qbc-node query
// ─────────────────────────────────────────────────────────────────────────────

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query chain state via RPC",
}

var queryBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Query the balance of an address",
	RunE: func(_ *cobra.Command, args []string) error {
		addr := flagQueryAddr
		if addr == "" && len(args) > 0 {
			addr = args[0]
		}
		if addr == "" {
			return fmt.Errorf("address required (--address or positional arg)")
		}
		result, err := rpcCall(flagRPCAddr, "qbc_getBalance", []string{addr})
		if err != nil {
			return err
		}
		qubits := extractUint64(result, "balance_qubits")
		fmt.Printf("Address : %s\n", addr)
		fmt.Printf("Balance : %d qubits  (%g QBC)\n", qubits, float64(qubits)/float64(core.OneQBC))
		return nil
	},
}

var queryChainCmd = &cobra.Command{
	Use:   "chain",
	Short: "Display current chain status",
	RunE: func(_ *cobra.Command, _ []string) error {
		result, err := rpcCall(flagRPCAddr, "qbc_chainInfo", nil)
		if err != nil {
			return err
		}
		printMap(result)
		return nil
	},
}

var queryBlockCmd = &cobra.Command{
	Use:   "block",
	Short: "Display a block by height",
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagQueryHeight == 0 && !flagQueryHeightSet {
			return fmt.Errorf("--height required")
		}
		result, err := rpcCall(flagRPCAddr, "qbc_blockByHeight", []uint64{flagQueryHeight})
		if err != nil {
			return err
		}
		printMap(result)
		return nil
	},
}

var queryFeeCmd = &cobra.Command{
	Use:   "fee",
	Short: "Display current fee estimate for all tiers",
	RunE: func(_ *cobra.Command, _ []string) error {
		result, err := rpcCall(flagRPCAddr, "qbc_feeEstimate", nil)
		if err != nil {
			return err
		}
		printMap(result)
		return nil
	},
}

var (
	flagQueryAddr      string
	flagQueryHeight    uint64
	flagQueryHeightSet bool
)

func init() {
	queryBalanceCmd.Flags().StringVar(&flagQueryAddr, "address", "", "address to query")
	queryBlockCmd.Flags().Uint64Var(&flagQueryHeight, "height", 0, "block height to retrieve")
	_ = queryBlockCmd.Flags().Lookup("height")
	flagQueryHeightSet = false // reset; cobra will set it when --height is provided

	queryCmd.AddCommand(queryBalanceCmd)
	queryCmd.AddCommand(queryChainCmd)
	queryCmd.AddCommand(queryBlockCmd)
	queryCmd.AddCommand(queryFeeCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// qbc-node version
// ─────────────────────────────────────────────────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the node version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("qbc-node %s\n", upgrade.Current())
		fmt.Println("Chain ID: 1  (QubitsCoin Mainnet)")
		fmt.Println("Crypto  : ML-DSA-65 (FIPS 204) | ML-KEM-768 (FIPS 203) | SHA-3-256 (FIPS 202)")
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// Silence the default log timestamp prefix, so our package-level prefixes
	// are the sole source of context.
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func loadConfig() (*config.Config, error) {
	if flagConfig != "" {
		return config.Load(flagConfig)
	}
	if flagTestnet {
		return config.TestnetDefault(), nil
	}
	return config.Default(), nil
}

func resolvePassword() string {
	if flagPassword != "" {
		return flagPassword
	}
	return os.Getenv("QBC_PASSWORD")
}

func resolvePath(dataDir, rel string) string {
	if rel == "" || rel[0] == '/' {
		return rel
	}
	if len(dataDir) >= 2 && dataDir[:2] == "~/" {
		home, _ := os.UserHomeDir()
		dataDir = home + dataDir[1:]
	}
	return dataDir + "/" + rel
}

func printBanner() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Printf(" ║  QubitsCoin (QBC) %-47s 										    ║"+"\n", upgrade.Current())
	fmt.Println("║  ML-DSA-65 / ML-KEM-768 / SHA-3-256 | EIP-1559 Ultra-Low Fees    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}

// ─────────────────────────────────────────────────────────────────────────────
// JSON-RPC client helpers (used by query and tx commands)
// ─────────────────────────────────────────────────────────────────────────────

type rpcReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// rpcCall sends a JSON-RPC 2.0 request and returns the decoded Result.
func rpcCall(endpoint, method string, params interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(rpcReq{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}

	httpResp, err := http.Post(endpoint, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", endpoint, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("WARN: could not close HTTP response body: %v\n", err)
		}
	}(httpResp.Body)

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp rpcResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		// Might be a scalar (e.g. qbc_blockHeight returns uint64).
		return map[string]interface{}{"value": string(resp.Result)}, nil
	}
	return result, nil
}

func fetchBaseFee(endpoint string) (uint64, error) {
	result, err := rpcCall(endpoint, "qbc_gasPrice", nil)
	if err != nil {
		return 0, err
	}
	return extractUint64(result, "base_fee"), nil
}

func extractString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractUint64(m map[string]interface{}, key string) uint64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return uint64(n)
		case uint64:
			return n
		}
	}
	return 0
}

func printMap(m map[string]interface{}) {
	for k, v := range m {
		fmt.Printf("  %-25s: %v\n", k, v)
	}
}

func gobEncodeTx(tx *core.Transaction) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(tx); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Ensure time is used (it's transitively used in loadConfig / node).
var _ = time.Second

// Ensure json is used for string parsing.
var _ = strings.TrimSpace
