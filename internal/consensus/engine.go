package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
	"github.com/qbc-qubitscoin/qubitscoin/internal/upgrade"
	"github.com/qbc-qubitscoin/qubitscoin/internal/vm"
)

const (
	BlockInterval = 2 * time.Second
	RoundTimeout  = 4 * time.Second
	MaxTxPerBlock = 10_000 // raised to match the higher block gas limit
)

// Engine drives single-validator BFT block production.
type Engine struct {
	mu sync.Mutex

	validatorAddr [crypto.AddressSize]byte
	validatorPub  []byte
	validatorPriv []byte

	validatorSet *ValidatorSet
	state        *state.DB
	pool         *mempool.Mempool
	execVM       *vm.VM           // WASM VM (maybe nil)
	upgradeMgr   *upgrade.Manager // auto-upgrade watcher (maybe nil)

	chain    []*core.Block // in-memory chain, index = height
	commitCh chan *core.Block
}

// NewEngine creates a consensus engine.
// execVM and upgradeMgr may be nil if those features are not needed.
func NewEngine(
	addr [crypto.AddressSize]byte,
	pubKey, privKey []byte,
	vs *ValidatorSet,
	st *state.DB,
	pool *mempool.Mempool,
	genesis *core.Block,
	execVM *vm.VM,
	upgradeMgr *upgrade.Manager,
) *Engine {
	return &Engine{
		validatorAddr: addr,
		validatorPub:  pubKey,
		validatorPriv: privKey,
		validatorSet:  vs,
		state:         st,
		pool:          pool,
		execVM:        execVM,
		upgradeMgr:    upgradeMgr,
		chain:         []*core.Block{genesis},
		commitCh:      make(chan *core.Block, 64),
	}
}

// Run starts the block-production loop; it blocks until ctx is canceled.
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(BlockInterval)
	defer ticker.Stop()
	log.Printf("[consensus] engine started, validator=%s", crypto.ToHex(e.validatorAddr))
	for {
		select {
		case <-ctx.Done():
			log.Println("[consensus] engine stopped")
			return
		case <-ticker.C:
			if err := e.produceBlock(ctx); err != nil {
				log.Printf("[consensus] block production error: %v", err)
			}
		}
	}
}

func (e *Engine) produceBlock(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	height := uint64(len(e.chain))
	proposer := e.validatorSet.Proposer(height)
	if proposer.Address != e.validatorAddr {
		return nil // not our turn
	}

	blk, newState, err := e.buildBlock(height)
	if err != nil {
		return fmt.Errorf("buildBlock: %w", err)
	}
	if err := e.commit(blk, newState); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// buildBlock assembles a new block on a state snapshot.
// Fee model: baseFee is burned; only priorityTip + blockReward go to validator.
func (e *Engine) buildBlock(height uint64) (*core.Block, *state.DB, error) {
	prev := e.chain[height-1]
	prevHash := prev.Hash
	baseFee := prev.Header.BaseFee // inherit parent's base fee
	snap := e.state.Snapshot()

	pending := e.pool.Pending(MaxTxPerBlock)
	var included []*core.Transaction
	var totalGas, totalBurned, totalTip uint64

	for _, tx := range pending {
		result, err := state.ApplyTransaction(snap, tx, core.BlockGasLimit-totalGas, e.execVM, baseFee)
		if err != nil {
			continue // skip txs that can't pay baseFee or are otherwise invalid
		}
		totalGas += result.GasUsed
		totalBurned += result.BurnedFee
		totalTip += result.ValidatorTip
		included = append(included, tx)
		if totalGas >= core.BlockGasLimit {
			break
		}
	}

	// ── Credit validator: tip + block subsidy (NOT burned fees) ─────────────
	reward := core.BlockReward(height)
	income := totalTip + reward
	if income > 0 {
		validator := snap.GetAccount(e.validatorAddr)
		validator.Balance += income
		snap.SetAccount(e.validatorAddr, validator)
	}

	stateRoot := snap.CommitRoot()

	// ── Compute next block's base fee ────────────────────────────────────────
	nextBaseFee := core.NextBaseFee(baseFee, totalGas)

	blk, err := core.NewBlock(
		height, prevHash, stateRoot,
		time.Now().UnixNano(),
		e.validatorAddr, included, totalGas, nextBaseFee, totalBurned,
	)
	if err != nil {
		return nil, nil, err
	}
	if err := blk.SignHeader(e.validatorPriv); err != nil {
		return nil, nil, err
	}
	return blk, snap, nil
}

// commit finalizes a block: applies state, purges mempool, appends a chain.
func (e *Engine) commit(blk *core.Block, snap *state.DB) error {
	e.state.Apply(snap)
	e.pool.PurgeCommitted(blk.Txs)
	e.chain = append(e.chain, blk)

	reward := core.BlockReward(blk.Header.Height)
	log.Printf("[consensus] block height=%d hash=%s txs=%d gas=%d baseFee=%d burned=%d qubits reward=%d QBC",
		blk.Header.Height,
		crypto.ToHex(blk.Hash)[:16]+"…",
		len(blk.Txs),
		blk.Header.GasUsed,
		blk.Header.BaseFee,
		blk.Header.BurnedFees,
		reward/core.OneQBC,
	)

	select {
	case e.commitCh <- blk:
	default:
	}

	if e.upgradeMgr != nil {
		e.upgradeMgr.OnBlock(blk.Header.Height)
	}
	return nil
}

// CommitCh returns a channel that receives every committed block.
func (e *Engine) CommitCh() <-chan *core.Block { return e.commitCh }

// InjectBlock appends a block that was loaded from persistent storage during
// node startup.  It does NOT re-apply state — the caller must ensure state
// is already consistent with the injected blocks.
func (e *Engine) InjectBlock(blk *core.Block) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Extend the chain slice to the required height, filling gaps with nil.
	for uint64(len(e.chain)) <= blk.Header.Height {
		e.chain = append(e.chain, nil)
	}
	e.chain[blk.Header.Height] = blk
}

// BlockByHeight returns the block at the given height or nil.
func (e *Engine) BlockByHeight(h uint64) *core.Block {
	e.mu.Lock()
	defer e.mu.Unlock()
	if h >= uint64(len(e.chain)) {
		return nil
	}
	return e.chain[h]
}

// Height returns the current chain height (number of blocks including genesis).
func (e *Engine) Height() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return uint64(len(e.chain))
}

// AcceptVote processes an incoming vote received from a peer validator.
// Extension point: not exercised in single-validator mode; called by the
// P2P layer when vote messages arrive from other validators.
func (e *Engine) AcceptVote(v *Vote) error {
	if err := v.Verify(); err != nil {
		return err
	}
	if !e.validatorSet.Contains(v.Voter) {
		return fmt.Errorf("unknown voter: %s", crypto.ToHex(v.Voter))
	}
	return nil
}
