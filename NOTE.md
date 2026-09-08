# QubitsCoin (QBC) — Architecture & Code Notes

> **Purpose of this document**: A complete explanation of every package,
> file, function, and how they connect — written for any developer who
> wants to understand the project without reading every line of source code.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Logical Architecture Diagram](#2-logical-architecture-diagram)
3. [Data-Flow Diagram — Block Production](#3-data-flow-diagram--block-production)
4. [Data-Flow Diagram — P2P Handshake](#4-data-flow-diagram--p2p-handshake)
5. [Package-by-Package Notes](#5-package-by-package-notes)
   - [internal/crypto](#51-internalcrypto)
   - [internal/core](#52-internalcore)
   - [internal/state](#53-internalstate)
   - [internal/mempool](#54-internalmempool)
   - [internal/consensus](#55-internalconsensus)
   - [internal/p2p](#56-internalp2p)
   - [internal/vm](#57-internalvm)
   - [internal/upgrade](#58-internalupgrade)
   - [cmd/node](#59-cmdnode)
6. [Key Algorithms Explained](#6-key-algorithms-explained)
7. [Fee Model Deep Dive](#7-fee-model-deep-dive)
8. [Cryptography Reference](#8-cryptography-reference)
9. [Constants Quick-Reference](#9-constants-quick-reference)

---

## 1. Project Overview

QubitsCoin (QBC) is a **quantum-resistant Layer-1 blockchain** written in Go.
Every cryptographic operation uses **NIST Post-Quantum Cryptography (PQC)**
standards — no RSA, no ECDSA, no secp256k1.

| Property        | Value                                           |
|-----------------|-------------------------------------------------|
| Signing         | ML-DSA-65 (FIPS 204) via Cloudflare CIRCL       |
| Key Exchange    | ML-KEM-768 (FIPS 203) for P2P handshake         |
| Hashing         | SHA-3-256 (FIPS 202) for all hashes             |
| Encryption      | AES-256-GCM for post-handshake P2P channels     |
| Smart Contracts | WASM via wazero (pure-Go, no cgo)               |
| Block time      | 2 seconds                                       |
| Fee model       | EIP-1559 (dynamic base fee, burned + tip split) |
| Currency unit   | 1 QBC = 1,000,000,000 qubits (9 decimals)       |
| Hard cap        | 100,000,000 QBC                                 |

---

## 2. Logical Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          cmd/node/main.go                               │
│  Entry point: wires all subsystems together and runs the demo loop      │
└───────┬──────────────┬──────────────┬────────────────┬──────────────────┘
        │              │              │                │
        ▼              ▼              ▼                ▼
┌──────────────┐ ┌──────────┐ ┌──────────────┐ ┌────────────────┐
│   consensus  │ │   p2p    │ │   upgrade    │ │      vm        │
│   (engine)   │ │  (node)  │ │  (manager)   │ │  (wazero WASM) │
└──────┬───────┘ └────┬─────┘ └──────┬───────┘ └───────┬────────┘
       │              │              │                  │
       ▼              │              ▼                  │
┌──────────────┐      │     ┌────────────────┐          │
│    state     │      │     │  upgrade/      │          │
│  (StateDB)   │      │     │  scheduler     │          │
│  transition  │      │     │  downloader    │          │
│  block_proc  │      │     │  applier       │          │
└──────┬───────┘      │     └────────────────┘          │
       │              │                                 │
       ▼              ▼                                 │
┌──────────────┐ ┌──────────────┐                       │
│    core      │ │   crypto     │◄──────────────────────┘
│  types       │ │  hash.go     │
│  block       │ │  signing.go  │
│  transaction │ │  wallet.go   │
│  fee         │ │              │
│  tokenomics  │ └──────────────┘
│  genesis     │
│  merkle      │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   mempool    │
│  (priority   │
│   queue)     │
└──────────────┘
```

**Dependency rule**: lower boxes never import upper boxes.
`core` and `crypto` are the foundation — they import nothing from this project.

---

## 3. Data-Flow Diagram — Block Production

```
Every 2 seconds (BlockInterval):
─────────────────────────────────────────────────────────────────

  Mempool                Consensus Engine              State DB
     │                        │                           │
     │  pool.Pending(10_000)  │                           │
     │───────────────────────►│                           │
     │                        │  state.Snapshot()         │
     │                        │──────────────────────────►│
     │                        │◄──── snap (deep copy) ────│
     │                        │                           │
     │            For each tx in pending:                  │
     │                        │  ApplyTransaction(snap,tx,│
     │                        │    remainingGas, vm,      │
     │                        │    baseFee)               │
     │                        │──────────────────────────►│
     │                        │◄── TxResult{GasUsed,      │
     │                        │    BurnedFee,ValidatorTip}│
     │                        │                           │
     │                        │  Accumulate:              │
     │                        │   totalGas += GasUsed     │
     │                        │   totalBurned += BurnedFee│
     │                        │   totalTip += ValidatorTip│
     │                        │                           │
     │                        │  Credit validator:        │
     │                        │  balance += tip + reward  │
     │                        │                           │
     │                        │  snap.CommitRoot()        │
     │                        │──────────────────────────►│
     │                        │◄── stateRoot [32]byte ────│
     │                        │                           │
     │                        │  NextBaseFee(baseFee,     │
     │                        │    totalGas) → nextFee    │
     │                        │                           │
     │                        │  NewBlock(height,         │
     │                        │   prevHash, stateRoot,    │
     │                        │   ts, validatorAddr,      │
     │                        │   txs, totalGas,          │
     │                        │   nextFee, totalBurned)   │
     │                        │                           │
     │                        │  blk.SignHeader(privKey)  │
     │                        │                           │
     │                        │  state.Apply(snap)        │
     │                        │──────────────────────────►│
     │  pool.PurgeCommitted() │                           │
     │◄───────────────────────│                           │
     │                        │                           │
     │                        │  commitCh ← blk           │
     │                        │  upgradeMgr.OnBlock(h)    │
     │                        │  p2pNode.BroadcastBlock() │
```

---

## 4. Data-Flow Diagram — P2P Handshake

```
  Initiator (dialer)                    Responder (listener)
        │                                       │
        │── TCP connect ────────────────────────►│
        │                                       │
        │── Hello{NodeID, PubKey, Addr} ────────►│
        │                                       │
        │◄── HelloResp{NodeID, PubKey, ok} ──────│
        │                                       │
        │  [KEM encapsulate using resp pubkey]   │
        │  ct, ss = KEM.Encapsulate(respPubKey)  │
        │  transcript = SHA-3(ct ‖ respNodeID)  │
        │  sig = ML-DSA.Sign(privKey,transcript)│
        │                                       │
        │── KEMInit{senderID, ct, sig} ──────────►│
        │                                       │  [Verify initiator sig]
        │                                       │  ss = KEM.Decapsulate(ct)
        │                                       │  sig2 = ML-DSA.Sign(
        │                                       │    privKey, SHA-3(ss[:16]‖initID))
        │◄── KEMDone{receiverID, sig2} ──────────│
        │                                       │
        │  [Verify responder sig]               │
        │                                       │
        │  sessionKey = SHA-3(ss ‖ sortedIDs)   │  sessionKey = SHA-3(ss ‖ sortedIDs)
        │                                       │
        │════ AES-256-GCM encrypted channel ════│
        │  All subsequent messages encrypted    │
        │  Nonce = counter (8-byte big-endian)  │
```

---

## 5. Package-by-Package Notes

---

### 5.1 `internal/crypto`

**Purpose**: All cryptographic primitives. Nothing in this project does
raw crypto outside this package.

#### `hash.go`

```
Hash256(data []byte) → [32]byte
```
- Computes **SHA-3-256** (FIPS 202) of arbitrary data.
- Used everywhere: block hashes, address derivation, state roots, tx hashes.

```
HashMany(parts ...[]byte) → [32]byte
```
- Hashes multiple byte slices as if concatenated, without allocating a combined
  slice. Used for multi-field transcript hashes (e.g., KEM handshake, Merkle).

```
ToHex(h [32]byte) → string
```
- Encodes a 32-byte array to lowercase hex. Used for logging and map keys.

```
var ZeroHash [32]byte
```
- The all-zero hash. Used as a sentinel for "no code" (EOA vs contract)
  and as the `To` field of deployment transactions (no destination address).

---

#### `signing.go`

```
Sign(privKeyBytes, msg []byte) → (sig []byte, err error)
```
- Unmarshals an ML-DSA-65 private key (4032 bytes) from raw bytes.
- Signs `msg` using `mldsa65.Scheme().Sign()` — deterministic signing.
- Returns a 3309-byte signature.

```
Verify(pubKeyBytes, msg, sig []byte) → (bool, error)
```
- Unmarshals an ML-DSA-65 public key (1952 bytes).
- Verifies `sig` over `msg`. Returns `(false, nil)` for bad signatures
  (not an error), reserving errors for key-parsing failures.

---

#### `wallet.go`

```
type Wallet struct {
    PublicKey  []byte  // 1952 bytes — ML-DSA-65
    PrivateKey []byte  // 4032 bytes — ML-DSA-65
    Address    [32]byte // SHA-3-256(PublicKey)
}
```

- **Address derivation**: `Address = SHA-3-256(PublicKey)` — quantum-safe
  since SHA-3 preimage resistance is unaffected by Grover's algorithm at
  this key size.

```
NewWallet() → (*Wallet, error)
```
- Generates a fresh ML-DSA-65 key pair using `crypto/rand`.
- Derives the address from the public key.

```
DeriveAddress(pubKeyBytes []byte) → [32]byte
```
- Used by transaction verification to check `From == SHA-3-256(PublicKey)`.

```
Zeroize()
```
- Overwrites private key bytes with zeros. Call when done with a wallet
  to protect against memory scraping.

---

### 5.2 `internal/core`

**Purpose**: Blockchain data structures and pure-logic functions.
No I/O, no goroutines, no external dependencies except `internal/crypto`.

#### `types.go` — Constants

| Constant           | Value                | Meaning                                  |
|--------------------|----------------------|------------------------------------------|
| `GasTransfer`      | 21                   | Gas cost of a simple transfer            |
| `GasDeploy`        | 5,000                | Gas cost of deploying a WASM contract    |
| `GasCall`          | 500                  | Base gas cost of calling a contract      |
| `BlockGasLimit`    | 500,000,000          | Max gas per block (~23M transfers/block) |
| `MinGasPrice`      | 1                    | Absolute floor: 1 qubit/gas              |
| `OneQBC`           | 1,000,000,000        | Qubits in one QBC (9 decimal places)     |
| `MaxSupply`        | 100,000,000 × OneQBC | Hard cap: 100M QBC                       |
| `ProtocolVersion`  | 1                    | Block header version                     |
| `ChainID`          | 1                    | Mainnet chain identifier                 |
| `BlockIntervalSec` | 2                    | Target block time in seconds             |

`TxType` enumerates transaction types: `TxTransfer` (0x01), `TxDeploy` (0x02),
`TxCall` (0x03), `TxStake` (0x04), `TxUnstake` (0x05).

---

#### `fee.go` — EIP-1559 Fee Model

```
const (
    InitialBaseFee        = 10  // qubits/gas at genesis
    MinBaseFee            = 1   // floor: 1 qubit/gas
    MaxBaseFeeChangeDenom = 8   // ±12.5% max change per block
    TargetGasRatioDenom   = 2   // target = 50% of BlockGasLimit
)
```

```
NextBaseFee(current, gasUsed uint64) → uint64
```
**How it works**:
1. `target = BlockGasLimit / 2` (250,000,000 gas)
2. If `gasUsed == target` → return `current` unchanged
3. If `gasUsed > target` (congested):
   - `delta = current × (gasUsed − target) / target / 8`
   - Clamped to `[1, current/8]` (never more than +12.5%)
   - Return `current + delta`
4. If `gasUsed < target` (idle):
   - `delta = current × (target − gasUsed) / target / 8`
   - Return `max(MinBaseFee, current − delta)` (never below 1)

```
FeeEstimate(baseFee, tier) → (maxFeePerGas, priorityTip)
```
| Tier              | maxFeePerGas         | priorityTip |
|-------------------|----------------------|-------------|
| `FeeTierUltraLow` | baseFee              | 0           |
| `FeeTierStandard` | baseFee + baseFee/10 | baseFee/10  |
| `FeeTierFast`     | baseFee + baseFee/2  | baseFee/2   |

```
TransferCostQubits(baseFee, tip) → uint64
```
- Returns `GasTransfer × (baseFee + tip) = 21 × (baseFee + tip)`.
- At genesis baseFee=10, no tip: **210 qubits = $0.00000021** (at $1/QBC).

```
FeeComparisonTable() → []FeeComparison
```
- Returns benchmark data: ETH ($1.50), BNB ($0.05), Avalanche ($0.02),
  Polygon ($0.002), Solana ($0.00025), Sui ($0.00002), **QBC ($0.00000021)**.

---

#### `block.go` — Block Structure

```
type BlockHeader struct {
    Version       uint32           // protocol version (currently 1)
    Height        uint64           // block number (0 = genesis)
    PrevHash      [32]byte         // SHA-3-256 of parent block header
    MerkleRoot    [32]byte         // Merkle root of all tx hashes
    StateRoot     [32]byte         // SHA-3-256 of sorted account state
    Timestamp     int64            // Unix nanoseconds
    ValidatorAddr [32]byte         // block proposer address
    GasUsed       uint64           // total gas consumed by all txs
    GasLimit      uint64           // always BlockGasLimit
    BaseFee       uint64           // this block's base fee (qubits/gas)
    BurnedFees    uint64           // total qubits burned in this block
}

type Block struct {
    Header    BlockHeader
    Txs       []*Transaction
    Signature []byte         // ML-DSA-65 over Header.Encode()
    Hash      [32]byte       // SHA-3-256 of Header.Encode()
}
```

**`Header.Encode()`** — produces 160 bytes in canonical order:
`Version(4) + Height(8) + PrevHash(32) + MerkleRoot(32) + StateRoot(32)
+ Timestamp(8) + ValidatorAddr(32) + GasUsed(8) + GasLimit(8)
+ BaseFee(8) + BurnedFees(8)`

**`NewBlock(...)`** — constructor that:
1. Calls `tx.BasicValidate()` on every transaction
2. Computes the Merkle root from tx hashes
3. Builds the header (baseFee, burnedFees included)
4. Computes the block hash

**`SignHeader(privKey)`** — ML-DSA-65 signs `Header.Encode()`,
stores the 3309-byte signature, then updates `Hash`.

**`VerifyValidatorSig(pubKey)`** — verifies the stored signature.

---

#### `transaction.go` — Transaction Structure

```
type Transaction struct {
    Version   uint8            // must be 1
    Type      TxType           // Transfer / Deploy / Call / Stake / Unstake
    Nonce     uint64           // sender nonce (replay protection)
    From      [32]byte         // sender address = SHA-3-256(PublicKey)
    To        [32]byte         // recipient / contract address
    Amount    uint64           // qubits to send
    GasLimit  uint64           // max gas this tx may use
    GasPrice  uint64           // qubits per gas (must be ≥ baseFee)
    Timestamp int64            // Unix nanoseconds
    Data      []byte           // contract bytecode (Deploy) or call payload (Call)
    PublicKey []byte           // sender's ML-DSA-65 public key (1952 bytes)
    Signature []byte           // ML-DSA-65 signature (3309 bytes)
    Hash      [32]byte         // SHA-3-256(signingPayload + signature)
}
```

**Signing payload** (fields hashed for signature):
`Version + Type + Nonce + From + To + Amount + GasLimit
+ GasPrice + Timestamp + len(Data) + Data + PublicKey`

**`BasicValidate()`** checks:
- Version == 1
- GasLimit > 0
- GasPrice ≥ MinGasPrice (1)
- Timestamp > 0
- PublicKey is exactly 1952 bytes
- Signature is exactly 3309 bytes
- `From == SHA-3-256(PublicKey)` (prevents key substitution)
- Signature is cryptographically valid

**`NewTransfer(...)`** — convenience constructor for simple transfers.
Sets `GasLimit = GasTransfer`, `Type = TxTransfer`.

---

#### `tokenomics.go` — Emission Schedule

```
const (
    InitialBlockReward = 45 × OneQBC   // 45 QBC per block in Era 0
    HalvingInterval    = 1,000,000     // ~23 days at 2s/block
    GenesisPremine     = 10,000,000 × OneQBC
)
```

```
BlockReward(height) → uint64
```
- `era = (height - 1) / HalvingInterval`
- `reward = InitialBlockReward >> era` (right-shift = divide by 2 each era)
- Height 0 (genesis): returns 0 (no reward for genesis block)

| Era | Block range           | Reward/block |
|-----|-----------------------|--------------|
| 0   | 1 – 1,000,000         | 45 QBC       |
| 1   | 1,000,001 – 2,000,000 | 22.5 QBC     |
| 2   | 2,000,001 – 3,000,000 | 11.25 QBC    |
| ... | ...                   | ...          |
| Max | ~32M+                 | 0 QBC        |

```
TotalEmissionAt(height) → uint64   // total block rewards up to height
CirculatingSupply(height) → uint64 // GenesisPremine + TotalEmissionAt(height)
```

---

#### `genesis.go` — Genesis Block

```
type GenesisConfig struct {
    ChainID       int
    Timestamp     int64
    ValidatorAddr [32]byte
    Allocations   map[[32]byte]uint64  // address → initial balance
}
```

**`DefaultGenesisConfig(validatorAddr)`** — creates a config with
10,000,000 QBC pre-mined to the validator.

**`Build()`** — creates the genesis block (height 0):
- `BaseFee = InitialBaseFee` (10 qubits/gas) — seeds the fee market
- `BurnedFees = 0` — nothing burned in genesis
- `StateRoot = genesisStateRoot(allocations)` — deterministic hash of
  all pre-mine allocations sorted by address
- No transactions, no signature required

---

#### `merkle.go` — Binary Merkle Tree

```
ComputeMerkleRoot(hashes [][32]byte) → [32]byte
EmptyMerkleRoot() → [32]byte
```

- **Algorithm**: Binary tree, leaves are tx hashes. If an odd number of
  leaves, the last leaf is duplicated (standard Bitcoin-style).
- Each parent = `SHA-3-256(leftChild ‖ rightChild)`.
- Empty tree returns `SHA-3-256([]byte{})`.
- Used in `NewBlock()` to create the block's `MerkleRoot` field.

---

### 5.3 `internal/state`

**Purpose**: Account state management, transaction execution, and block application.

#### `account.go`

```
type Account struct {
    Nonce       uint64    // incremented each tx from this address
    Balance     uint64    // balance in qubits
    CodeHash    [32]byte  // zero for Externally Owned Accounts (EOAs)
    StorageRoot [32]byte  // zero for EOAs
}
```

- `IsContract()` returns true if `CodeHash != ZeroHash`.
- `Clone()` returns a deep copy — critical for snapshot-based rollback.

---

#### `statedb.go` — In-Memory State Database

```
type StateDB struct {
    mu       sync.RWMutex
    accounts map[string]*Account  // hex address → account
}
```

**Key methods**:

| Method                  | Description                                              |
|-------------------------|----------------------------------------------------------|
| `GetAccount(addr)`      | Returns a clone (safe to modify without affecting state) |
| `SetAccount(addr, acc)` | Stores a clone (thread-safe write)                       |
| `GetBalance(addr)`      | Shortcut for `GetAccount(addr).Balance`                  |
| `GetNonce(addr)`        | Shortcut for `GetAccount(addr).Nonce`                    |
| `Credit(addr, amount)`  | Adds qubits (block reward credits)                       |
| `CommitRoot()`          | Deterministic SHA-3-256 of all accounts (sorted)         |
| `Snapshot()`            | Deep copy — used before executing a block                |
| `Apply(other)`          | Replaces this state with `other` (post-block commit)     |

**Snapshot pattern** used by the consensus engine:
```
snap := state.Snapshot()       // cheap deep copy before block
// ... apply transactions to snap ...
state.Apply(snap)              // atomically commit if no errors
```

---

#### `transition.go` — Transaction Execution

```
ApplyTransaction(st, tx, remainingBlockGas, execVM, baseFee) → (*TxResult, error)
```

**Execution steps**:
1. **Intrinsic gas check**: `IntrinsicGas(tx) ≤ tx.GasLimit`
2. **Block gas check**: `tx.GasLimit ≤ remainingBlockGas`
3. **Base-fee check**: `tx.GasPrice ≥ baseFee` (EIP-1559 rejection)
4. **Balance check**: `sender.Balance ≥ tx.GasLimit × tx.GasPrice + tx.Amount`
5. **Nonce check**: `sender.Nonce == tx.Nonce`
6. **Pre-deduct**: `sender.Balance -= tx.GasLimit × tx.GasPrice`; `sender.Nonce++`
7. **Dispatch** by `tx.Type`:
   - `TxTransfer` → `applyTransfer` (moves `Amount` from sender to recipient)
   - `TxDeploy` → `applyDeploy` (compiles WASM, stores `CodeHash`, calls `_init`)
   - `TxCall` → `applyCall` (parses function name from `Data`, calls via VM)
8. **Gas refund**: `sender.Balance += (tx.GasLimit − gasUsed) × tx.GasPrice`
9. **Fee split**:
   - `BurnedFee  = gasUsed × min(baseFee, tx.GasPrice)`
   - `ValidatorTip = gasUsed × tx.GasPrice − BurnedFee`
   - Burned fees are **not credited to anyone** — they vanish (deflation)

```
type TxResult struct {
    GasUsed      uint64
    FeeCollected uint64  // total deducted from sender (gasUsed × gasPrice)
    BurnedFee    uint64  // base-fee portion destroyed
    ValidatorTip uint64  // tip credited to block validator
    Success      bool
    Error        error
    ReturnData   []byte  // contract address (Deploy) or return values (Call)
}
```

```
IntrinsicGas(tx) → uint64
```
| Type     | Formula                      |
|----------|------------------------------|
| Transfer | `GasTransfer` (21)           |
| Deploy   | `GasDeploy + len(Data) × 68` |
| Call     | `GasCall + len(Data) × 16`   |

---

#### `block_processor.go` — Full Block Application

```
ApplyBlock(st, blk, validatorAddr, execVM) → (*BlockResult, error)
```

1. Reads `baseFee` from `blk.Header.BaseFee`
2. Calls `ApplyTransaction` for each tx, accumulating:
   - `GasUsed`, `FeeCollected`, `BurnedFees`, `ValidatorTip`
3. Computes `BlockReward(blk.Header.Height)`
4. Credits `ValidatorTip + BlockReward` to validator's balance
5. **BurnedFees are NOT credited** — they reduce circulating supply

```
type BlockResult struct {
    GasUsed      uint64
    FeeCollected uint64  // total paid by all senders
    BurnedFees   uint64  // base-fee portion destroyed (deflationary)
    ValidatorTip uint64  // tip portion credited to validator
    BlockReward  uint64  // coinbase subsidy
    TxResults    []*TxResult
}

TotalValidatorIncome() → ValidatorTip + BlockReward
// Note: BurnedFees is NOT income — it is destroyed
```

---

### 5.4 `internal/mempool`

**Purpose**: Thread-safe, priority-ordered pool of unconfirmed transactions.

#### `mempool.go`

```
type Mempool struct {
    mu       sync.RWMutex
    txs      map[string]*Transaction      // hash-hex → tx
    bySender map[string][]*Transaction    // addr-hex → sender's txs
    heap     txHeap                       // max-heap by GasPrice
    maxSize  int
}
```

```
New(maxSize int) → *Mempool
```
- `maxSize ≤ 0` defaults to `DefaultMaxSize` (10,000).

```
Add(tx) → error
```
1. `tx.BasicValidate()` — full signature + format check
2. `len(tx.Data) ≤ MaxTxDataSize` (1 MiB)
3. No duplicate hash
4. Pool doesn’t full (`len(txs) < maxSize`)
5. Sender queue not full (max 64 txs per sender)
6. Inserts into `txs` map, `bySender` map, and `heap`

```
Pending(n int) → []*Transaction
```
- Returns up to `n` highest-`GasPrice` txs **without removing them**.
- Copies the heap to avoid mutation during iteration.

```
PurgeCommitted(committed []*Transaction)
```
- Removes all committed txs from the pool after a block is finalised.

#### `priorityqueue.go` — `txHeap`

Implements `container/heap.Interface` for a **max-heap** ordered by `GasPrice`.
Higher gas price = higher priority = included first in a block.

---

### 5.5 `internal/consensus`

**Purpose**: Block production engine (single-validator BFT) and validator set management.

#### `validator.go`

```
type Validator struct {
    Address     [32]byte
    PublicKey   []byte
    VotingPower uint64
}

type ValidatorSet struct {
    validators   []*Validator
    totalPower   uint64
    addressIndex map[string]int  // hex → slice index
}
```

```
NewValidatorSet(validators) → (*ValidatorSet, error)
```
- Validates no duplicate addresses, builds the index.

```
HasQuorum(addresses [][32]byte) → bool
```
- Returns true if the sum of voting power for unique, known addresses
  is **> 2/3 of totalPower** (Byzantine fault-tolerant threshold).

```
Proposer(height uint64) → *Validator
```
- Round-robin: `validators[height % len(validators)]`.

```
Get(addr) → *Validator   // nil if not in set
Contains(addr) → bool
All() → []*Validator     // defensive copy
TotalPower() → uint64
```

---

#### `vote.go`

```
type Vote struct {
    Height    uint64
    BlockHash [32]byte
    Voter     [32]byte
    PublicKey []byte
    Signature []byte
}
```

```
Vote.Sign(privKey) → error       // ML-DSA-65 sign Vote payload
Vote.Verify() → error            // verifies Voter == SHA-3(PubKey) + sig valid
```

---

#### `engine.go` — Block Production Loop

```
type Engine struct {
    validatorAddr, validatorPub, validatorPriv  // this node's identity
    validatorSet  *ValidatorSet
    state         *state.StateDB
    pool          *mempool.Mempool
    execVM        *vm.VM
    upgradeMgr    *upgrade.Manager
    chain         []*core.Block    // in-memory chain (index = height)
    commitCh      chan *core.Block  // buffered(64); send committed blocks
}
```

```
Run(ctx) — blocks until ctx cancelled
```
- Ticks every `BlockInterval` (2s).
- Calls `produceBlock(ctx)` each tick.

```
produceBlock(ctx) — called under mutex
```
1. `height = len(chain)` (next block height)
2. `Proposer(height) == validatorAddr?` → skip if not our turn
3. `buildBlock(height)` → `(blk, snapState, err)`
4. `commit(blk, snapState)`

```
buildBlock(height) → (*Block, *StateDB, error)
```
- See [Data-Flow Diagram — Block Production](#3-data-flow-diagram--block-production)
- Inherits `baseFee` from parent block header
- Computes `NextBaseFee` after processing all txs
- Credits `totalTip + BlockReward` to validator on the snapshot

```
commit(blk, snap)
```
1. `state.Apply(snap)` — persists the snapshot
2. `pool.PurgeCommitted(blk.Txs)` — removes included txs from mempool
3. `chain = append(chain, blk)` — extends an in-memory chain
4. Logs block details (height, hash, gas, baseFee, burned, reward)
5. Non-blocking send to `commitCh`
6. `upgradeMgr.OnBlock(height)` — notifies upgrade scheduler

```
BlockByHeight(h) → *Block    // query API: returns block or nil
Height() → uint64            // current chain length
AcceptVote(v) → error        // extension point for multi-validator mode
```

---

### 5.6 `internal/p2p`

**Purpose**: Quantum-safe peer-to-peer networking using ML-KEM-768 key exchange
and AES-256-GCM encrypted channels.

#### `identity.go`

```
type Identity struct {
    NodeID     [32]byte  // SHA-3-256(PublicKey) — this node's P2P ID
    PublicKey  []byte    // ML-DSA-65 public key
    PrivateKey []byte    // ML-DSA-65 private key
    ListenAddr string    // "host:port"
}

NewIdentity(pubKey, privKey, listenAddr) → *Identity
```
- Derives `NodeID = SHA-3-256(pubKey)`.

```
type PeerInfo struct {
    NodeID     [32]byte
    PublicKey  []byte
    ListenAddr string
}
```

---

#### `handshake.go` — ML-KEM-768 Key Exchange

**Protocol** (see [Data-Flow Diagram — P2P Handshake](#4-data-flow-diagram--p2p-handshake)):

```
InitiatorHandshake(conn, local, remoteInfo) → (*SecureConn, error)
ResponderHandshake(conn, local) → (*SecureConn, *PeerInfo, error)
```

**Security properties**:
- `ss` (shared secret) is 32 bytes from ML-KEM-768 encapsulation.
- Both sides sign transcripts with ML-DSA-65 to authenticate.
- Session key = `SHA-3-256(ss ‖ sorted(nodeIDA, nodeIDB))` — the sort
  makes `sessionKey` identical regardless of who initiated.
- Forward-secrecy is partial: KEM ciphertext is sent once; if `ss` is
  later compromised, past traffic is exposed. Full forward-secrecy
  requires ephemeral keys (future work).

---

#### `secure.go` — AES-256-GCM Channel

```
type SecureConn struct {
    conn    net.Conn
    gcm     cipher.AEAD     // AES-256-GCM
    sendMu, recvMu sync.Mutex
    sendSeq, recvSeq uint64  // counter-based nonces
}
```

```
Send(plaintext []byte) → error
```
1. Lock `sendMu`
2. `nonce = nonce12(sendSeq)` — 12-byte nonce, 4 zero bytes + 8-byte counter
3. `sendSeq++`
4. `ciphertext = AES-GCM.Seal(nonce, plaintext)`
5. Write as a length-prefixed frame

```
Recv() → ([]byte, error)
```
1. Lock `recvMu`
2. Read length-prefixed frame
3. `nonce = nonce12(recvSeq)`, `recvSeq++`
4. `AES-GCM.Open(nonce, frame)` — authenticates and decrypts

**Counter-nonces prevent nonce reuse** (the most critical AES-GCM requirement).

---

#### `message.go` — Wire Protocol

**Message types**:
| Code | Name | Used for |
|---|---|---|
| 0x01 | `MsgHello` | Handshake initiation |
| 0x02 | `MsgHelloResp` | Handshake response |
| 0x03 | `MsgKEMInit` | KEM ciphertext + signature |
| 0x04 | `MsgKEMDone` | KEM completion + responder sig |
| 0x10 | `MsgTx` | Gossip a transaction |
| 0x11 | `MsgBlock` | Gossip a block |
| 0x12 | `MsgGetBlocks` | Request blocks by height |
| 0x13 | `MsgBlocks` | Respond with blocks |
| 0x14 | `MsgPeerList` | Share known peers |
| 0x20 | `MsgPing` | Keepalive ping |
| 0x21 | `MsgPong` | Keepalive response |

**Wire framing**: `[4-byte big-endian length][payload bytes]`
Max frame: 8 MiB.

**Serialization**: Go `encoding/gob` (compact binary, no schema definition needed).

```
NewTxGossip(tx) → *GossipMsg
NewBlockGossip(blk) → *GossipMsg
```
- Wraps data in `GossipMsg{MsgID: hash, Hops: 0, Type, Data}`.
- `MsgID` is the tx/block hash — used for deduplication.

---

#### `gossip.go` — Duplicate Suppression

```
type GossipCache struct {
    seen  map[[32]byte]struct{}   // set of seen MsgIDs
    order [][32]byte              // FIFO for eviction
    cap   int                    // default: 32,768
}
```

```
MarkSeen(msgID) → bool
```
- Returns `true` if **first time seen** (should forward).
- Returns `false` if already seen (discard — prevents gossip loops).
- When cache is full, evicts the oldest **half** in one pass.

---

#### `peer.go` — Connected Peer

```
type Peer struct {
    Info  *PeerInfo
    conn  *SecureConn
    inbox chan []byte   // buffered(256)
    quit  chan struct{}
}
```

```
run(ctx, onMsg func([]byte)) — goroutine
```
- Continuously reads decrypted frames from `SecureConn.Recv()`.
- Sets a 60-second read deadline per message (keepalive enforcement).
- Calls `onMsg(frame)` for each received frame.
- Stops on context cancellation or read error.

```
Send(data []byte) → error  // encrypts + sends via SecureConn
Close()                    // closes the underlying connection
```

---

#### `node.go` — P2P Node Orchestrator

```
type Node struct {
    identity         *Identity
    server           *tcpServer
    gossip           *GossipCache
    peers            map[string]*Peer    // nodeID-hex → peer
    OnPeerConnect    func(*PeerInfo)     // callback hooks
    OnPeerDisconnect func(*PeerInfo)
    OnTxReceived     func(*core.Transaction)
    OnBlockReceived  func(*core.Block)
}
```

```
NewNode(local) → (*Node, error)
```
- Creates a `tcpServer` (binds TCP socket on `local.ListenAddr`).
- Initializes gossip cache and empty peer map.

```
Start(ctx)
```
- Launches `server.acceptLoop` in a goroutine.
- Launches `pingLoop` in a goroutine.

```
Connect(ctx, addr) → error
```
- Dials `addr`, runs `InitiatorHandshake`, calls `addPeer`.

```
addPeer(ctx, p)
```
1. Reject duplicate NodeIDs and connections over `defaultMaxPeers` (25).
2. Add to `peers` map.
3. Call `OnPeerConnect` hook.
4. Launch `p.run(ctx, dispatch)` goroutine.
5. Send our peer list to new peer.

```
BroadcastBlock(blk)   // gossip a block to all peers
BroadcastTx(tx)       // gossip a transaction to all peers
```
Both check `gossip.MarkSeen` first to prevent re-broadcasting
messages this node already knows about.

**`dispatch(sender, frame)`** — routes received frames by `MsgType`:
- `MsgTx` → decode tx, call `OnTxReceived`, re-gossip with `Hops++`
- `MsgBlock` → decode block, call `OnBlockReceived`, re-gossip
- `MsgPeerList` → connect to unknown peers
- `MsgPing` → send `MsgPong`

```
PeerCount() → int
```

---

#### `server.go` — TCP Listener

```
newTCPServer(local) → (*tcpServer, error)
```
- `net.Listen("tcp", local.ListenAddr)`.

```
acceptLoop(ctx, onPeer)
```
- Calls `listener.Accept()` in a loop.
- On context cancellation, closes listener (unblocks Accept).
- For each accepted connection, spawns `handleConn` goroutine.

```
handleConn(ctx, conn, onPeer)
```
- Runs `ResponderHandshake` on the raw TCP connection.
- On success: logs the remote address via `sc.RemoteAddr()`, creates a `Peer`.
- On failure: logs the error, closes the connection.

---

### 5.7 `internal/vm`

**Purpose**: WASM smart-contract execution via wazero (pure Go, no cgo).

#### `vm.go`

```
type VM struct {
    rt      wazero.Runtime
    modules map[string]wazero.CompiledModule  // codeHash-hex → compiled
    storage *ContractStorage
    seq     atomic.Uint64   // unique instance name suffix
}
```

```
NewVM(ctx) → (*VM, error)
```
- Creates a wazero runtime.
- Registers the QBC host module (`"env"`) with these host functions:

| Export             | Parameters            | Description                        |
|--------------------|-----------------------|------------------------------------|
| `qbc_get`          | `(slot i32) → i64`    | Read from contract storage         |
| `qbc_set`          | `(slot i32, val i64)` | Write to contract storage          |
| `qbc_log`          | `(ptr i32, len i32)`  | Log a message from contract        |
| `qbc_caller`       | `() → i64`            | Get caller address (first 8 bytes) |
| `qbc_block_height` | `() → i64`            | Get current block height           |
| `qbc_value`        | `() → i64`            | Get qubits sent with the call      |

```
Deploy(ctx, contractAddr, code) → error
```
- Computes `codeHash = SHA-3-256(code)`.
- Compiles WASM bytecode using `wazero.Runtime.CompileModule`.
- Caches the compiled module by `codeHash`.
- If the same code is deployed again, returns immediately (idempotent).

```
Call(ctx, ec, codeHash, funcName, params...) → ([]uint64, error)
```
1. Looks up the compiled module by `codeHash`.
2. Instantiates a **fresh module instance** per call (memory isolation).
3. Calls the named exported function with `params`.
4. Returns `[]uint64` result values.
5. **Closes the module instance** after the call (cleanup).

Fresh instances prevent state leakage between calls. Contract persistent
state lives in `ContractStorage`, not in WASM linear memory.

---

#### `context.go` — Execution Environment

```
type ExecutionContext struct {
    ContractAddr [32]byte
    CallerAddr   [32]byte
    Value        uint64    // qubits sent with call
    BlockHeight  uint64
    BlockTime    int64
    GasLimit     uint64
    GasUsed      uint64
    ReadOnly     bool
}
```

```
UseGas(amount) → error   // returns ErrOutOfGas if budget exceeded
WithExecCtx(ctx, ec) → context.Context
GetExecCtx(ctx) → *ExecutionContext
```

The `ExecutionContext` is attached to a `context.Context` via a typed key
(`ExecCtxKey{}`). Host functions retrieve it with `GetExecCtx` to read
`CallerAddr`, `Value`, etc.

---

#### `storage.go` — Contract Storage

```
type ContractStorage struct {
    mu   sync.RWMutex
    data map[[32]byte]map[uint32]uint64  // contractAddr → slot → value
}
```

- Simple key-value store: `(contractAddress, uint32 slot) → uint64 value`.
- Not persistent — in-memory only (production would use a Merkle trie).
- Host functions `qbc_get` / `qbc_set` read/write this storage.

---

#### `host.go` — Host Function Implementations

```
hostGet(storage) → api.GoModuleFunction
hostSet(storage) → api.GoModuleFunction
hostLog()        → api.GoModuleFunction
hostCaller()     → api.GoModuleFunction
hostBlockHeight()→ api.GoModuleFunction
hostValue()      → api.GoModuleFunction
```

Each function returns a closure that captures the storage or the
`ExecutionContext` from the call's `context.Context`.

---

#### `contract.go` — CounterContract (Built-in Test Contract)

```
var CounterContractWASM []byte
```

A compiled WASM module embedded at compile time (via a build tag or
`go:generate`). Exports:
- `increment()` — adds 1 to a counter in slot 0
- `get_count() → i64` — reads the counter from slot 0
- `_init()` — called once on deployment; initializes counter to 0

Used by `cmd/node/main.go` to demonstrate contract deployment and calling.

---

### 5.8 `internal/upgrade`

**Purpose**: Automatic binary self-update and on-chain protocol upgrade scheduling.

#### `version.go`

```
type Version struct { Major, Minor, Patch int }
```

```
Current() → Version          // reads buildVersion variable (set by -ldflags)
ParseVersion(s) → Version    // parses "vX.Y.Z" or "X.Y.Z"
v.String() → string          // "vX.Y.Z"
v.After(other) → bool        // strict semantic version comparison
v.Equal(other) → bool
```

`buildVersion` defaults to `"0.4.0"`. Override at build time:
```
go build -ldflags "-X github.com/qbc-qubitscoin/qubitscoin/internal/upgrade.buildVersion=1.0.0"
```

---

#### `release.go` — GitHub Releases API Client

```
FetchLatestRelease(ctx, apiURL) → (*Release, error)
```
1. GETs the GitHub Releases API endpoint.
2. Parses the JSON response.
3. Selects the asset matching `binaryAssetName()`:
   - Linux: `node-linux-amd64`
   - Windows: `node-windows-amd64.exe`
   - macOS: `node-darwin-arm64`
4. Fetches the companion `.sha3sum` asset to get the expected checksum.

```
type Release struct {
    Version     Version
    DownloadURL string   // binary download URL
    Checksum    string   // expected SHA-3-256 hex
    ReleaseURL  string   // GitHub release page URL
}
```

---

#### `downloader.go` — Binary Download + Verification

```
Download(ctx, rel) → (tmpPath string, error)
```
1. Creates a temp file in the same directory as the running binary.
2. Streams the binary via HTTP, writing to the temp file.
3. Simultaneously hashes the stream with SHA-3-256.
4. Verifies the hash against `rel.Checksum` (if provided).
5. `chmod 0755` the temp file.

```
ComputeFileHash(path) → (string, error)
```
- Reads a file and returns its SHA-3-256 hex digest.
- Called by `Manager.applyRelease` for **on-disk re-verification** after
  download, before replacing the running binary (defense-in-depth).

---

#### `applier.go` — Binary Replacement

```
Apply(newBinaryPath) → error
```
**Cross-platform atomic binary replacement**:
1. Rename current exe → `<exe>.old` (Windows allows renaming running executables)
2. Rename `newBinaryPath` → current exe path
3. `reExec(exePath, os.Args)`:
   - **POSIX**: `syscall.Exec` — replaces the process image
   - **Windows**: `os.StartProcess` + `os.Exit(0)` — spawns new process

On any error, attempts rollback by reversing the renames.

```
cleanOldBinary()
```
- Removes `<exe>.old` if it exists from a previous upgrade.
- Called at startup.

---

#### `scheduler.go` — On-Chain Upgrade Coordination

```
type Proposal struct {
    TargetVersion Version
    TargetHeight  uint64
    Proposer      [32]byte
    PublicKey     []byte
    Signature     []byte  // ML-DSA-65 over payload()
}
```

```
type Scheduler struct {
    proposals   map[string]*Proposal       // version → proposal
    votes        map[string]map[string]bool // version → voterHex → voted
    scheduled   *Proposal                  // proposal at quorum
    totalVoters int
    onReady     ReadyFunc
}
```

```
AddProposal(p) → (bool, error)
```
1. Verifies signature: `Proposer == SHA-3(PubKey)` and sig valid.
2. Records the vote.
3. If `3 × voteCount > 2 × totalVoters` → quorum reached → `scheduled = p`.
4. Returns `(true, nil)` if quorum was reached.

```
OnBlock(height)
```
- If `scheduled != nil` and `height ≥ scheduled.TargetHeight`:
  - Calls `onReady(p)` in a goroutine.
  - Clears `scheduled` to prevent repeated firing.

```
Manager.Run(ctx)
```
- Periodic release check loop (default: every hour).
- Calls `check(ctx)` immediately on startup.

```
Manager.applyRelease(ctx, rel)
```
- Protected by `sync.Once` — only applies once per process lifetime.
- Downloads → `ComputeFileHash` re-verify → `Apply`.

---

### 5.9 `cmd/node`

**Purpose**: Entry point. Wires all subsystems into a runnable demo node.

#### `main.go` — Step-by-Step Startup

| Step | Action                                                                          |
|------|---------------------------------------------------------------------------------|
| 1    | Generate two ML-DSA-65 wallets (validator + deployer)                           |
| 2    | Build genesis block with 10M + 1M QBC pre-mine                                  |
| 3    | Bootstrap StateDB from genesis allocations; create mempool and P2P node         |
| 4    | Print fee comparison table (QBC vs ETH/Solana/etc.)                             |
| 5    | Print halving emission schedule (6 eras)                                        |
| 5b   | Print `BlockReward()` spot-checks at key heights                                |
| 6    | Deploy `CounterContract` WASM via `ApplyTransaction`                            |
| 7    | Call `increment()` × 5; verify `get_count() == 5`                               |
| 8    | Run `ApplyBlock` on a simulated block; verify reward + fee split                |
| 9    | Start upgrade manager; start consensus engine (runs 3 blocks)                   |
| 10   | Print final summary (chain height, genesis hash, validator balance, peer count) |

**P2P wiring in main**:
```go
p2pNode.OnTxReceived = func(tx *core.Transaction) {
    if pool.Add(tx) == nil {
        p2pNode.BroadcastTx(tx) // re-gossip valid txs
    }
}
p2pNode.Start(ctx)
```

**Engine commit loop**:
```go
case blk := <-engine.CommitCh():
    p2pNode.BroadcastBlock(blk)   // gossip to peers
    // ... log / display block ...
```

```go
func totalAllocQBC(allocs map[[32]byte]uint64) uint64
```
- Helper that sums all genesis allocations and returns the total in QBC.

---

## 6. Key Algorithms Explained

### Nonce-based replay protection

Every account has a `Nonce` that starts at 0. Each transaction from that
account must have `tx.Nonce == account.Nonce`. After a successful transaction,
`account.Nonce` is incremented. This prevents:
- **Replay attacks**: submitting the same signed transaction twice.
- **Transaction ordering** within the mempool (higher nonce = later tx).

### State root computation

```
StateRoot = SHA-3-256(
    ∀ addr ∈ sorted(accounts):
        addr_hex || nonce_be8 || balance_be8 || codeHash || storageRoot
)
```

Sorting addresses ensures determinism regardless of Go's random map iteration.
This root is stored in `BlockHeader.StateRoot` and committed by every validator.

### Merkle root computation

```
if len(leaves) == 0:   return SHA-3-256([])
if len(leaves) is odd: leaves.append(leaves[-1])   // duplicate last
while len(leaves) > 1:
    parents = []
    for i in range(0, len(leaves), 2):
        parents.append(SHA-3-256(leaves[i] || leaves[i+1]))
    leaves = parents
return leaves[0]
```

---

## 7. Fee Model Deep Dive

### Why burn the base fee?

Burning `BaseFee` (instead of giving it to the validator) solves the
**incentive-alignment problem**: if validators received the base fee, they
would have an incentive to artificially inflate block gas usage to collect
more fees. By burning it, validators only benefit from the `PriorityTip`,
which users voluntarily add. This makes the base fee a neutral market signal.

### Fee split example

```
Block baseFee = 10 qubits/gas
Tx:  gasLimit=1000, gasPrice=15, gasUsed=500

Pre-deduct from sender: 1000 × 15 = 15,000 qubits
Refund unused gas:      (1000 - 500) × 15 = 7,500 qubits

FeeCollected = 500 × 15 = 7,500 qubits   (net deduction)
BurnedFee    = 500 × 10 = 5,000 qubits   (base fee destroyed)
ValidatorTip = 500 × 5  = 2,500 qubits   (priority tip to validator)
```

### Base fee adjustment

```
At 100% full block (500M gas used, target 250M):
  over = 500M - 250M = 250M
  delta = baseFee × 250M / 250M / 8 = baseFee / 8 = +12.5%

At empty block (0 gas used):
  under = 250M - 0 = 250M
  delta = baseFee × 250M / 250M / 8 = baseFee / 8 = -12.5%
```

---

## 8. Cryptography Reference

| Algorithm   | Key sizes          | Signature/CT size | NIST standard   |
|-------------|--------------------|-------------------|-----------------|
| ML-DSA-65   | PK=1952B, SK=4032B | Sig=3309B         | FIPS 204        |
| ML-KEM-768  | PK=1184B, SK=2400B | CT=1088B, SS=32B  | FIPS 203        |
| SHA-3-256   | N/A                | Digest=32B        | FIPS 202        |
| AES-256-GCM | Key=32B, Nonce=12B | Tag=16B           | NIST SP 800-38D |

### Why ML-DSA-65 instead of ECDSA?

ECDSA relies on the **discrete logarithm problem** on elliptic curves.
Shor's algorithm on a large enough quantum computer can break this
in polynomial time. ML-DSA-65 (Module Lattice Digital Signature Algorithm)
is based on the **Module Learning With Errors (MLWE)** hardness assumption,
which has no known quantum speedup.

### Why SHA-3-256 instead of SHA-256?

SHA-256's security against collision attacks is 128 bits classically.
Grover's algorithm provides a quadratic speedup for preimage attacks,
reducing effective security to ~128 bits classically but ~85 bits for
a quantum adversary. SHA-3-256 has the same collision resistance profile,
but its **sponge construction** (Keccak) is fundamentally different from
SHA-2's Merkle-Damgård, providing better resistance against length-extension
attacks and maintaining 128-bit post-quantum security for preimages.

---

## 9. Constants Quick-Reference

### Gas constants (`internal/core/types.go`)

| Name            | Value       | Meaning                               |
|-----------------|-------------|---------------------------------------|
| `GasTransfer`   | 21          | Gas for a simple value transfer       |
| `GasDeploy`     | 5,000       | Base gas for WASM contract deployment |
| `GasCall`       | 500         | Base gas for contract function call   |
| `BlockGasLimit` | 500,000,000 | Max gas per block                     |
| `MinGasPrice`   | 1           | Minimum 1 qubit per gas unit          |

### Fee constants (`internal/core/fee.go`)

| Name                    | Value | Meaning                        |
|-------------------------|-------|--------------------------------|
| `InitialBaseFee`        | 10    | Genesis base fee in qubits/gas |
| `MinBaseFee`            | 1     | Absolute floor for base fee    |
| `MaxBaseFeeChangeDenom` | 8     | Denominator for ±12.5% cap     |
| `TargetGasRatioDenom`   | 2     | Target = 50% of BlockGasLimit  |

### Tokenomics constants (`internal/core/tokenomics.go`)

| Name                 | Value               | Meaning                            |
|----------------------|---------------------|------------------------------------|
| `InitialBlockReward` | 45 × OneQBC         | 45 QBC per block (Era 0)           |
| `HalvingInterval`    | 1,000,000           | Blocks between halvings (~23 days) |
| `GenesisPremine`     | 10,000,000 × OneQBC | Initial supply allocation          |

### P2P constants (`internal/p2p/`)

| Name               | Value  | Meaning                                    |
|--------------------|--------|--------------------------------------------|
| `defaultMaxPeers`  | 25     | Max simultaneous connections               |
| `dialTimeout`      | 10s    | Timeout for outbound connection            |
| `pingInterval`     | 30s    | Keepalive ping period                      |
| `peerReadDeadline` | 60s    | Max time to wait for a message             |
| `gossipCacheSize`  | 32,768 | Dedup cache capacity                       |
| `maxGossipHops`    | 3      | Max relay hops before a message is dropped |
| `maxFrameSize`     | 8 MiB  | Max wire frame size                        |

### Consensus constants (`internal/consensus/engine.go`)

| Name            | Value  | Meaning                              |
|-----------------|--------|--------------------------------------|
| `BlockInterval` | 2s     | Target block time                    |
| `RoundTimeout`  | 4s     | Timeout per consensus round          |
| `MaxTxPerBlock` | 10,000 | Max transactions pulled from mempool |

---

*Generated: 2026-05-20 — QubitsCoin v0.4.0*
