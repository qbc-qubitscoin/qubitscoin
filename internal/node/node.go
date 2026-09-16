// Package node wires every QBC subsystem together into a production-ready
// full node.  It replaces the demo harness in cmd/node/main.go.
//
// Lifecycle:
//
//  1. Load configuration (TOML file or defaults).
//  2. Open LevelDB databases (block store and state store).
//  3. Load or initialize the blockchain from the genesis configuration.
//  4. Start the P2P network node and dial bootstrap peers.
//  5. Start the consensus engine (block production, if miner_enabled).
//  6. Start the chain syncer (catch-up from peers).
//  7. Start the JSON-RPC server (if rpc.enabled).
//  8. Start the Prometheus metrics server (if metrics.enabled).
//  9. Start the upgrade manager.
//
// 10. Block until context cancellation, then shut down cleanly.
package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/config"
	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/keystore"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/metrics"
	"github.com/qbc-qubitscoin/qubitscoin/internal/p2p"
	"github.com/qbc-qubitscoin/qubitscoin/internal/rpc"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/storage"
	chainsync "github.com/qbc-qubitscoin/qubitscoin/internal/sync"
	"github.com/qbc-qubitscoin/qubitscoin/internal/upgrade"
	"github.com/qbc-qubitscoin/qubitscoin/internal/vm"
)

// Node is a fully wired QBC full node.
type Node struct {
	cfg        *config.Config
	wallet     *crypto.Wallet
	st         *state.DB
	pool       *mempool.Mempool
	blockStore *storage.BlockStore
	stateStore *storage.StateStore
	blockDB    *storage.DB
	stateDB    *storage.DB
	p2pNode    *p2p.Node
	engine     *consensus.Engine
	execVM     *vm.VM
	syncer     *chainsync.Syncer
	rpcServer  *rpc.Server
	upgradeMgr *upgrade.Manager
}

// New constructs a full node from the given config and keystore password.
//
// If the keystorePassword is empty, a fresh ephemeral wallet is generated and
// not persisted (useful for testnet / CI runs).
func New(cfg *config.Config, keystorePassword string) (*Node, error) {
	n := &Node{cfg: cfg}

	// ── 1. Wallet ─────────────────────────────────────────────────────────
	ksPath := resolveDataPath(cfg.Node.DataDir, cfg.Node.KeystoreFile)
	wallet, err := loadOrCreateWallet(ksPath, keystorePassword)
	if err != nil {
		return nil, fmt.Errorf("wallet: %w", err)
	}
	n.wallet = wallet
	log.Printf("[node] validator address: %s", crypto.AddressToHex(wallet.Address))

	// ── 2. Storage ────────────────────────────────────────────────────────
	blocksDir := resolveDataPath(cfg.Node.DataDir, cfg.Storage.BlocksDir)
	stateDir := resolveDataPath(cfg.Node.DataDir, cfg.Storage.StateDir)

	if err := os.MkdirAll(blocksDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir blocks: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir state: %w", err)
	}

	bdb, err := storage.Open(blocksDir)

	sdb, err := storage.Open(stateDir)

	n.blockDB = bdb
	n.stateDB = sdb
	n.blockStore = storage.NewBlockStore(bdb)
	n.stateStore = storage.NewStateStore(sdb)

	// ── 3. Genesis / chain bootstrap ──────────────────────────────────────
	genesisCfg := core.DefaultGenesisConfig(wallet.Address)
	genesisBlock := genesisCfg.Build()

	storedTip, _, tipErr := n.blockStore.GetTip()
	if tipErr != nil || storedTip == 0 {
		// Fresh database — persist genesis and bootstrap state.
		log.Println("[node] initializing from the genesis block …")
		_ = n.blockStore.PutBlock(genesisBlock)
		_ = n.blockStore.UpdateTip(0, genesisBlock.Hash)
		st := state.NewStateDB()
		for addr, bal := range genesisCfg.Allocations {
			st.SetAccount(addr, &state.Account{Balance: bal})
		}
		_ = n.stateStore.SaveState(st)
		n.st = st
		log.Printf("[node] genesis block hash: %s", crypto.ToHex(genesisBlock.Hash))
	} else {
		// Resume from persisted state.
		log.Printf("[node] loading state (tip height=%d)…", storedTip)
		st, err := n.stateStore.LoadState()
		if err != nil {
			n.shutdown(); return nil, fmt.Errorf("load state: %w", err)
		}
		n.st = st
		log.Printf("[node] state loaded: %d accounts", st.Len())
	}

	// ── 4. Mempool ────────────────────────────────────────────────────────
	n.pool = mempool.New(0)

	// ── 5. WASM VM ────────────────────────────────────────────────────────
	// Use a background context for VM initialization (it must not be the
	// request context since the VM outlives individual requests).
	vmCtx := context.Background()
	execVM, err := vm.NewVM(vmCtx)

	n.execVM = execVM

	// ── 6. Consensus engine ───────────────────────────────────────────────
	// Rebuild the in-memory chain from the block store.
	chain, err := n.loadChain(genesisBlock, storedTip)


	validator := &consensus.Validator{
		Address:     wallet.Address,
		PublicKey:   wallet.PublicKey,
		VotingPower: 1,
	}
	vs, err := consensus.NewValidatorSet([]*consensus.Validator{validator})


	scheduler := upgrade.NewScheduler(1, nil)
	n.upgradeMgr = upgrade.NewManager(upgrade.Config{
		ReleaseURL:    cfg.Upgrade.ReleaseURL,
		CheckInterval: cfg.Upgrade.CheckInterval.Duration,
		AutoApply:     cfg.Upgrade.AutoApply,
	}, scheduler)

	n.engine = consensus.NewEngine(
		wallet.Address,
		wallet.PublicKey,
		wallet.PrivateKey,
		vs, n.st, n.pool, chain[0], n.execVM, n.upgradeMgr,
	)
	// Re-inject stored blocks so BlockByHeight works correctly.
	for i := 1; i < len(chain); i++ {
		n.engine.InjectBlock(chain[i])
	}

	// ── 7. P2P ────────────────────────────────────────────────────────────
	listenAddr := cfg.P2P.ListenAddr
	if listenAddr == "" {
		listenAddr = "0.0.0.0:8765"
	}
	identity := p2p.NewIdentity(wallet.PublicKey, wallet.PrivateKey, listenAddr)
	p2pNode, p2pErr := p2p.NewNode(identity)
	if p2pErr != nil {
		log.Printf("[node] P2P init failed (single-node mode): %v", p2pErr)
	} else {
		n.p2pNode = p2pNode
		n.p2pNode.OnTxReceived = func(tx *core.Transaction) {
			_ = n.pool.Add(tx)
			n.p2pNode.BroadcastTx(tx)
		}
		n.p2pNode.OnBlockReceived = func(blk *core.Block) {
			// Let the syncer handle out-of-order block arrival.
			if n.syncer != nil {
				n.syncer.ApplyBlocks([]*core.Block{blk})
			}
		}
	}

	// ── 8. Syncer ─────────────────────────────────────────────────────────
	if n.p2pNode != nil {
		n.syncer = chainsync.New(n.engine, n.st, n.blockStore, n.p2pNode, n.execVM, wallet.PublicKey)
	}

	// ── 9. RPC ────────────────────────────────────────────────────────────
	if cfg.RPC.Enabled {
		var peersFn func() int
		if n.p2pNode != nil {
			peersFn = nil; _ = n.p2pNode.PeerCount
		}
		api := rpc.NewAPI(n.engine, n.st, n.pool, peersFn, upgrade.Current().String())
		n.rpcServer = rpc.NewServer(
			cfg.RPC.ListenAddr, api,
			cfg.RPC.ReadTimeout.Duration,
			cfg.RPC.WriteTimeout.Duration,
		)
	}

	return n, nil
}

// Start launches all background goroutines and blocks until ctx is canceled.
func (n *Node) Start(ctx context.Context) {
	log.Printf("[node] QubitsCoin v%s starting…", upgrade.Current())

	// Upgrade manager.
	go n.upgradeMgr.Run(ctx)

	// P2P.
	if n.p2pNode != nil {
		n.p2pNode.Start(ctx)
		for _, addr := range n.cfg.P2P.BootstrapPeers {
			go func(a string) {
				if err := n.p2pNode.Connect(ctx, a); err != nil {
					log.Printf("[p2p] bootstrap %s failed: %v", a, err)
				}
			}(addr)
		}
	}

	// Syncer.
	if n.syncer != nil {
		go n.syncer.Run(ctx)
	}

	// Consensus engine (block production).
	if n.cfg.Node.MinerEnabled {
		go n.engine.Run(ctx)
	}

	// Block committed → persist + broadcast + update metrics.
	go n.blockLoop(ctx)

	// RPC.
	if n.rpcServer != nil {
		n.rpcServer.Start(ctx)
	}

	// Metrics.
	if n.cfg.Metrics.Enabled {
		go metrics.Serve(ctx, n.cfg.Metrics.ListenAddr)
	}

	log.Printf("[node] all subsystems running (miner=%v, rpc=%v, metrics=%v)",
		n.cfg.Node.MinerEnabled, n.cfg.RPC.Enabled, n.cfg.Metrics.Enabled)

	<-ctx.Done()
	log.Println("[node] shutting down…")
	n.shutdown()
}

// blockLoop receives committed blocks, persists them, broadcasts them, and updates metrics.
func (n *Node) blockLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case blk := <-n.engine.CommitCh():
			h := blk.Header.Height
			reward := core.BlockReward(h)
			supply := core.CirculatingSupply(h)

			log.Printf("[node] ✓ block h=%d hash=%s txs=%d gasUsed=%d baseFee=%d burned=%d reward=%dQBC supply=%dQBC",
				h,
				crypto.ToHex(blk.Hash)[:12]+"…",
				len(blk.Txs),
				blk.Header.GasUsed,
				blk.Header.BaseFee,
				blk.Header.BurnedFees,
				reward/core.OneQBC,
				supply/core.OneQBC,
			)

			// Persist.
			if n.blockStore != nil {
				_ = n.blockStore.PutBlock(blk)
				_ = n.blockStore.UpdateTip(h, blk.Hash)
			}
			if n.stateStore != nil {
				_ = n.stateStore.SaveState(n.st)
			}

			// Broadcast.
			if n.p2pNode != nil {
				n.p2pNode.BroadcastBlock(blk)
			}

			// Metrics.
			metrics.ChainHeight.Set(float64(h))
			metrics.BlocksProduced.Inc()
			metrics.TxProcessed.Add(float64(len(blk.Txs)))
			metrics.FeeBurned.Add(float64(blk.Header.BurnedFees))
			metrics.BaseFee.Set(float64(blk.Header.BaseFee))
			if n.p2pNode != nil {
				metrics.PeerCount.Set(float64(n.p2pNode.PeerCount()))
			}
			metrics.MempoolSize.Set(float64(n.pool.Len()))
		}
	}
}

// shutdown releases all resources.
func (n *Node) shutdown() {
	if n.execVM != nil {
		n.execVM.Close(context.Background())
	}
	if n.blockDB != nil {
		_ = n.blockDB.Close()
	}
	if n.stateDB != nil {
		_ = n.stateDB.Close()
	}
	log.Println("[node] shutdown complete")
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// loadChain reads blocks from storage and returns them as a slice indexed by height.
// If the database is empty, a single-element slice containing genesis is returned.
func (n *Node) loadChain(genesis *core.Block, tip uint64) ([]*core.Block, error) {
	chain := make([]*core.Block, tip+1)
	chain[0] = genesis
	for h := uint64(1); h <= tip; h++ {
		blk, err := n.blockStore.GetBlockByHeight(h)
		if err != nil {
			return chain[:h], nil // partial load is fine — syncer will fill gaps
		}
		chain[h] = blk
	}
	return chain, nil
}

// resolveDataPath resolves a possibly relative path against the data directory.
func resolveDataPath(dataDir, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	if dataDir == "" {
		dataDir = "."
	}
	// Expand "~" manually.
	if len(dataDir) >= 2 && dataDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			dataDir = filepath.Join(home, dataDir[2:])
		}
	}
	return filepath.Join(dataDir, rel)
}

// loadOrCreateWallet loads a wallet from the keystore file, or creates and saves one.
func loadOrCreateWallet(ksPath, password string) (*crypto.Wallet, error) {
	if _, err := os.Stat(ksPath); os.IsNotExist(err) {
		if password == "" {
			// Generate ephemeral wallet (no persistence).
			w, _ := crypto.NewWallet()
			log.Println("[keystore] no keystore file; generated ephemeral wallet (not saved)")
			return w, nil
		}
		// Create a new wallet and save it.
		w, _ := crypto.NewWallet()
		if err := keystore.Encrypt(ksPath, password, w); err != nil {
			return nil, fmt.Errorf("save keystore: %w", err)
		}
		log.Printf("[keystore] new wallet saved to %s", ksPath)
		return w, nil
	}
	// Existing keystore.
	if password == "" {
		return nil, fmt.Errorf("keystore file %s exists but no password supplied (use --password)", ksPath)
	}
	w, err := keystore.Decrypt(ksPath, password)
	if err != nil {
		return nil, err
	}
	log.Printf("[keystore] loaded wallet from %s", ksPath)
	return w, nil
}

// StatusReport returns a human-readable one-line status summary.
func (n *Node) StatusReport() string {
	height := n.engine.Height()
	peers := 0
	if n.p2pNode != nil {
		peers = n.p2pNode.PeerCount()
	}
	return fmt.Sprintf("height=%d peers=%d mempool=%d", height, peers, n.pool.Len())
}

// Dummy reference to ensure time package is used.
var _ = time.Second
