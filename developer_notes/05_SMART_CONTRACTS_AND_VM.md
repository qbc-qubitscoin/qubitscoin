# Developer Note 05: Smart Contracts & WebAssembly VM Runtime

## 1. Why Wazero WebAssembly Was Chosen Over EVM

Ethereum Virtual Machine (EVM) suffers from 256-bit word overhead, expensive gas modeling, and non-standard bytecode. QubitsCoin uses **WebAssembly (WASM)** powered by **Tetratelabs Wazero**:
1. **100% Pure Go (Zero CGO)**: Wazero does not require C libraries, gcc, or Rust linkers to run WebAssembly. It runs across Windows, Linux, and macOS out of the box.
2. **Standard Compiler Support**: Contracts can be compiled from Go, Rust, C, or AssemblyScript to standard WASM.
3. **Memory Isolation & Safety**: WASM sandboxes each contract execution in isolated linear memory.

---

## 2. VM Host Functions & ABI

Located in `internal/vm/host.go`:
When a WASM module runs on QBC, the node exposes 6 host functions into the module under the `qbc` namespace:

| Host Function | Signature | Purpose | Gas Cost |
|---|---|---|---|
| `qbc_get(slot uint32)` | `uint64` | Reads a 64-bit value from contract storage slot | 200 gas |
| `qbc_set(slot uint32, val uint64)` | `void` | Writes a 64-bit value into contract storage slot (fails in ReadOnly mode) | 5,000 gas |
| `qbc_caller()` | `uint64` (ptr) | Writes the 32-byte address of the transaction sender into WASM linear memory | 50 gas |
| `qbc_value()` | `uint64` | Returns the amount of qubits transferred with the call | 20 gas |
| `qbc_block_height()` | `uint64` | Returns the current chain height | 20 gas |
| `qbc_log(ptr uint32, len uint32)` | `void` | Emits a debug/event log string from contract linear memory | 500 gas |

### Gas Metering Protection
Every host call checks `ec.UseGas(cost)`. If gas exceeds the transaction's gas limit, the VM immediately aborts with `ErrOutOfGas`, triggering state rollback.

---

## 3. Core Built-In Smart Contracts

### 1. QubitSwap AMM (`internal/contracts/dex/qubitswap.go`)
Implements the constant product formula:
$$x \cdot y = k$$
- `AddLiquidity(amountA, amountB)`: Increases reserve balances.
- `SwapAforB(amountAIn)`:
  - Deducts a **0.3% protocol fee**:
    $$\text{amountInWithFee} = \frac{\text{amountAIn} \cdot 997}{1000}$$
  - Computes output amount:
    $$\text{amountBOut} = \frac{\text{ReserveB} \cdot \text{amountInWithFee}}{\text{ReserveA} + \text{amountInWithFee}}$$
  - Invariant: Reserves can never be fully drained; zero input returns `(0, false)`.

### 2. GreenDAO Governance (`internal/contracts/greendao/greendao.go`)
- `SubmitProposal(title, description, requestedQBC, target)`
- `Vote(proposalID, voter, power, support)`
  - Enforces **20% quorum** ($\ge 20\%$ of total power must vote).
  - Enforces **60% supermajority approval** of cast votes.
- `ExecuteProposal(proposalID)`: Releases treasury funds only when passed.

### 3. CarbonX ESG Registry (`internal/contracts/carbonx/carbonx.go`)
- `MintCredit(owner, tonnesCO2, standard, projectID)`: Generates verified carbon token IDs.
- `TransferCredit(tokenID, from, to)`: Transfers credit ownership.
- `RetireCredit(tokenID, owner)`: Permanently burns the credit from circulation, ensuring credits cannot be double-counted.

### 4. QubitID Sovereign Identity (`internal/contracts/qubitid/qubitid.go`)
- Decentralized Identity (DID) registration mapped to ML-DSA-65 post-quantum addresses.
- Verifiable Credentials (KYC, carbon offsets, governance credentials).

### 5. Multi-Signature Treasury (`internal/contracts/multisig/multisig.go`)
- $M$-of-$N$ threshold multi-signature contract for high-value treasury operations.
