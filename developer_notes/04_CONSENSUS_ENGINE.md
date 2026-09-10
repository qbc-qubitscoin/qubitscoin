# Developer Note 04: Consensus Engine & Validator Architecture

## 1. Why the Consensus Engine Was Designed This Way

Blockchains require consensus to order transactions and prevent double-spending. Proof-of-Work (PoW) is energy-intensive and slow, while complex multi-round PBFT protocols often suffer from communication overhead ($\mathcal{O}(n^2)$ network messages per block).

QubitsCoin uses a **Deterministic Round-Robin Single-Proposer with BFT Quorum Voting** model:
- Fixed **2-second block intervals** (`BlockInterval = 2 * time.Second`).
- Deterministic proposer selection for round-robin fairness.
- Supermajority quorum requirement for multi-validator networks:
  $$\text{Quorum} \ge \left\lfloor \frac{2 \cdot \text{TotalPower}}{3} \right\rfloor + 1$$
- Post-quantum ML-DSA-65 signatures on both block headers and consensus votes.

---

## 2. How Block Production Operates

### The Engine Loop (`Run` in `internal/consensus/engine.go`)
1. **Ticker Activation**: Every 2 seconds, the ticker fires.
2. **Proposer Check**:
   $$\text{proposerIndex} = \text{height} \pmod{|\text{ValidatorSet}|}$$
   If `proposer.Address != node.ValidatorAddress`, the node quietly skips production.
3. **Mempool Ingestion**:
   The engine pulls up to `MaxTxPerBlock` (10,000 txs) from the mempool priority queue.
4. **State Dry-Run**:
   Transactions are executed against an isolated `state.DB.Snapshot()`.
5. **Reward & Fee Calculation**:
   - `BurnedFees` are destroyed.
   - `ValidatorTip` + `BlockReward(height)` are credited to the block proposer.
6. **Block Construction & Signature**:
   The validator signs the block header using their ML-DSA-65 private key.
7. **Commit Pipeline**:
   - State is committed: `e.state.Apply(snap)`.
   - Committed transactions are purged from the mempool: `e.pool.PurgeCommitted(blk.Txs)`.
   - The block is emitted onto `commitCh` (buffer size 64) for P2P gossiping and RPC subscribers.
   - Upgrade manager hook is triggered: `e.upgradeMgr.OnBlock(height)`.

---

## 3. Validator Voting & Fork Prevention

When operating in multi-validator environments:
- **Vote Types**: `VotePrevote (1)` and `VotePrecommit (2)`.
- **Vote Payload**:
  $$\text{Payload} = \text{Type} \parallel \text{BigEndian}(Height) \parallel \text{BigEndian}(Round) \parallel \text{BlockHash}$$
- **Quorum Enforcement**: A block is not final until $\ge 2/3 + 1$ of total voting power has signed precommit votes for the block hash.
- **Vote Rejection**: Any vote with a voter not in the current validator set or with an invalid ML-DSA-65 signature is rejected immediately with an error.
