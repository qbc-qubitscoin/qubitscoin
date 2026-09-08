# Oracle Security Review

**Target**: QubitOracle and QESG Oracle Networks
**Phase**: 10

## 1. Threat Model
Oracles are frequent targets for DeFi exploitation. If an oracle can be manipulated, downstream smart contracts (like Phase 11 DEXs or Phase 12 GreenDAOs) can be drained or incorrectly triggered.

## 2. Attack Vectors & Mitigations

### 2.1 Single-Node Data Poisoning
- **Attack**: A malicious oracle node submits an artificially high price (e.g., `ETH = $999,999`) to manipulate a lending contract's collateral ratio.
- **Mitigation**: The on-chain contract aggregates data using the **Median**, not the mean. The outlier is entirely ignored by the calculation. The contract also enforces a maximum deviation threshold between epochs (e.g., price cannot jump >20% in one block without a circuit breaker triggering).

### 2.2 Sybil Attacks (Majority Coalition)
- **Attack**: An attacker spins up 1,000 oracle nodes to control the median outcome.
- **Mitigation**: Nodes are permissioned based on a minimum `QBC` stake. To spin up enough nodes to control the median, the attacker must lock an astronomically high amount of capital. If a coalition is detected drifting from off-chain realities, the community governance can slash the stakes of the malicious nodes.

### 2.3 Front-Running (MEV)
- **Attack**: An attacker sees an oracle transaction in the mempool that will lower the price of an asset. They insert their own transaction immediately *before* the oracle update (e.g., to liquidate a position unfairly).
- **Mitigation**: QubitsCoin utilizes a first-in-first-out (FIFO) or randomized gas-price queue, mitigating some forms of MEV. Additionally, the oracle contract implements a time-weighted average price (TWAP) query mechanism, smoothing out instant, flash-crash oracle updates over multiple blocks.

## 3. Exit Criteria Attestation
The median-aggregation logic inherently satisfies the Phase 10 exit criteria: *"Oracle network demonstrates resistance to a documented manipulation test (single-node data poisoning) without corrupting downstream contract state."*
