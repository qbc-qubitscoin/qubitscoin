# QubitsCoin (QBC) — Quantum-Resistant Layer-1 Blockchain

A production-grade, post-quantum Layer-1 blockchain written in Go, built exclusively on NIST Post-Quantum Cryptography (PQC) standards and featuring a high-performance WebAssembly (WASM) virtual machine and modern reactive UI portal.

[![Go](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![NIST PQC](https://img.shields.io/badge/crypto-NIST%20PQC%20Only-blueviolet)](https://csrc.nist.gov/projects/post-quantum-cryptography)
[![Coverage](https://img.shields.io/badge/coverage-100%25%20Statements-brightgreen)](TASK_RECORD.md)
[![UI](https://img.shields.io/badge/UI-React%2018%20%7C%20TypeScript%205%20%7C%20Vite-61DAFB?logo=react)](web)

---

## 1. Post-Quantum Cryptography

All cryptographic operations in QubitsCoin strictly conform to NIST Post-Quantum Cryptography standards. Traditional asymmetric cryptography (RSA, ECDSA, Ed25519, secp256k1) is completely omitted.

| Primitive | Standard | Specifications | Usage |
|-----------|----------|----------------|-------|
| **ML-DSA-65** | FIPS 204 | Public Key: 1,952 B<br>Signature: 3,309 B | Transaction & block signing, validator consensus votes, upgrade proposals |
| **ML-KEM-768** | FIPS 203 | Ciphertext: 1,088 B<br>Shared Secret: 32 B | Ephemeral quantum-resistant P2P key encapsulation handshake |
| **SHA-3-256** | FIPS 202 | 32-byte digest | All cryptographic hashing — addresses, block headers, transaction IDs, Merkle roots |
| **AES-256-GCM** | NIST SP 800-38D | 32-byte key, 12-byte IV | Post-handshake P2P transport encryption with monotonic counter nonces |
| **Argon2id** | RFC 9106 | 64 MB memory, 3 iterations | Keystore key derivation function (KDF) for private key encryption at rest |

---

## 2. Core Architecture & Repository Structure

```
qubitscoin/
├── cmd/
│   ├── node/          # Full node entry point (CLI: qbc-node start / wallet / tx / query)
│   ├── loadtest/      # High-throughput load testing and benchmarking tool
│   ├── multisig/      # Standalone CLI tool for quantum-safe M-of-N multisig operations
│   └── qubitid/       # CLI utility for QubitID decentralized identity credentials
├── configs/           # Network presets (mainnet.toml, testnet.toml, prometheus.yml, systemd)
├── docs/              # Comprehensive architectural designs, whitepapers, and phase specs (1-32)
├── developer_notes/   # In-depth engineering notes (TDD, PQC, VM, Networking, UI, Upgrades)
├── internal/
│   ├── config/        # TOML configuration parser and validator
│   ├── consensus/     # Single-validator BFT consensus engine, proposer rotation, validator sets
│   ├── contracts/     # Native and WASM smart contract modules (Phases 14–32):
│   │   ├── aimarket/      # AI model marketplace, prompt monetization & inference settlements
│   │   ├── autonomous/    # Autonomous AI agent execution, tasks & governance
│   │   ├── carbonx/       # Carbon credit registry, verifications & tokenization
│   │   ├── cbdc/          # Central Bank Digital Currency cross-border settlement rail
│   │   ├── compute/       # Decentralized cloud compute task matching and settlement
│   │   ├── custody/       # Institutional multi-tier custody, timelocks & emergency freezes
│   │   ├── dex/           # QubitSwap automated market maker (AMM) pools & swaps
│   │   ├── energyx/       # Peer-to-peer renewable energy trading & grid settlement
│   │   ├── esgmarket/     # ESG credit marketplace & compliance validation
│   │   ├── esgnetwork/    # Global ESG tracking, auditor sign-offs & emissions ledger
│   │   ├── global/        # Global multi-region jurisdiction routing & state anchors
│   │   ├── govpartner/    # Government & institutional partner identity and permissions
│   │   ├── greendao/      # DAO governance, token voting, timelocks & treasury grants
│   │   ├── hydrochain/    # Green hydrogen supply chain lifecycle tracking
│   │   ├── iso20022/      # ISO 20022 financial messaging translation (pacs.008, pain.001)
│   │   ├── lending/       # Decentralized collateralized lending & borrowing markets
│   │   ├── mainnet/       # Mainnet deployment parameters, staking rules & genesis allocations
│   │   ├── multisig/      # Quantum-resistant M-of-N multi-signature smart contract
│   │   ├── oracle/        # Decentralized price feeds, cross-chain attestation & quorum feeds
│   │   ├── quantumnet/    # Quantum network simulation, qubit teleportation & entanglement
│   │   ├── qubitid/       # Decentralized Identity (DID) & Verifiable Credentials (VC)
│   │   ├── rollups/       # Layer-2 optimistic & validity rollup state anchors
│   │   ├── rwa/           # Real-World Asset tokenization, dividends & compliance
│   │   ├── sovereign/     # Sovereign wealth fund reserves & allocation management
│   │   ├── storage/       # Decentralized storage contracts with storage proofs
│   │   └── superapp/      # SuperApp multi-service aggregation contract
│   ├── core/          # Block, Transaction, Genesis, Merkle tree, Tokenomics, EIP-1559 fee model
│   ├── crypto/        # ML-DSA-65 key management, ML-KEM-768 key encapsulation, SHA-3-256
│   ├── identity/      # W3C-compliant DID and Verifiable Credential primitives
│   ├── keystore/      # Encrypted disk keystores using Argon2id + AES-256-GCM
│   ├── mempool/       # Gas-price priority queue with sender account throttling
│   ├── metrics/       # Prometheus telemetry metrics and /healthz endpoint
│   ├── node/          # Full node lifecycle coordinator wiring all subsystems
│   ├── oracle/        # Oracle node backend service
│   ├── p2p/           # Encrypted P2P network, KEM handshake, gossip routing, peer discovery
│   ├── rpc/           # JSON-RPC 2.0 server (9 standard methods, batch queries)
│   ├── state/         # Account StateDB, block processor (ApplyBlock), gas metering
│   ├── storage/       # Embedded LevelDB database persistence for blocks and states
│   ├── sync/          # Initial Block Download (IBD) and batch chain synchronizer
│   ├── upgrade/       # Self-updating binary downloader, verification, and on-chain scheduler
│   ├── vm/            # WebAssembly virtual machine powered by wazero (pure-Go, zero CGo)
│   └── web/           # Embedded HTTP web server serving JSON-RPC and the static UI bundle
├── test/
│   └── bdd/           # Behavior-Driven Development (BDD) end-to-end scenario suites
└── web/               # Standalone React 18 + TypeScript 5 + Vite web dashboard
```

---

## 3. Currency & Tokenomics

| Parameter | Specification |
|-----------|---------------|
| **Asset Symbol** | **QBC** |
| **Atomic Unit** | **qubit** (1 QBC = 1,000,000,000 qubits; 9 decimal places) |
| **Hard Cap** | **100,000,000 QBC** |
| **Genesis Pre-mine** | 10,000,000 QBC (10% allocated across core ecosystem funds) |
| **Mining Subsidies** | 90,000,000 QBC emitted over 32 halving eras |
| **Initial Era 0 Reward** | 45.000000000 QBC per block |
| **Halving Interval** | Every 1,000,000 blocks (~23.1 days at 2 s/block) |
| **Block Time Target** | 2.0 seconds |
| **Block Gas Limit** | 30,000,000 gas |
| **Fee Mechanism** | EIP-1559 dynamic base fee (burned) + priority tip (awarded to validator) |

---

## 4. Building & Running

### Prerequisites
- **Go 1.22+** (tested and verified on Go 1.26+)
- **Node.js 18+ & npm** (optional, only for building or editing the web frontend)

### A. Build the Blockchain Node Executable
Always build the binary with a distinct name such as `qbc-node` or `qbc-node.exe` so your terminal does not confuse it with Node.js:

```powershell
# In PowerShell / Command Prompt:
go build -o qbc-node.exe ./cmd/node

# In Linux / macOS / Git Bash:
go build -o qbc-node ./cmd/node
```

### B. Start the Full Node
```powershell
# In Windows PowerShell:
.\qbc-node.exe start

# In Linux / macOS / Git Bash:
./qbc-node start

# Or run directly via Go without building an executable:
go run ./cmd/node start
```

> [!NOTE]
> Running `node start` on Windows will invoke the Node.js JavaScript interpreter from your `PATH`. Always use `.\qbc-node.exe start` or `go run ./cmd/node start`.

The node starts up, initializes the state database at `~/.qbc`, begins producing blocks every 2 seconds, and serves:
- **JSON-RPC 2.0 Endpoint**: `http://127.0.0.1:8545`
- **Prometheus Metrics & Healthz**: `http://127.0.0.1:9100/metrics`, `/healthz`
- **Interactive Web Portal**: `http://127.0.0.1:8545`

### C. Run the Web Dashboard (Frontend Development)
The frontend dashboard is built with React 18, TypeScript 5, and Vite:

```bash
cd web
npm install
npm run dev
```

To build production static assets that are embedded into the Go binary:
```bash
cd web
npm run build
```

---

## 5. Testing & Quality Assurance

QubitsCoin enforces strict quality and correctness requirements. **Every internal package and test suite has achieved 100.0% statement coverage**:

```bash
# Run all unit and package tests
go test ./...

# Run targeted package with coverage report
go test -cover ./internal/consensus ./internal/p2p ./internal/core

# Run BDD integration test scenarios
go test -v ./test/bdd

# Run frontend unit tests (Vitest)
cd web && npm test -- --run
```

Refer to [`TASK_RECORD.md`](TASK_RECORD.md) for the verified 100% coverage audit breakdown across all 44 internal packages.

---

## 6. Development Phases (1 – 32)

- [x] **Phase 1** — Core primitives (ML-DSA-65, ML-KEM-768, SHA-3-256, Merkle root, Block & Tx)
- [x] **Phase 2** — Account state machine (StateDB), EIP-1559 gas accounting, mempool priority queue
- [x] **Phase 3** — BFT consensus engine, validator sets, P2P encrypted transport & gossip routing
- [x] **Phase 4** — Wazero pure-Go WASM virtual machine & on-chain auto-upgrade scheduler
- [x] **Phase 5** — Tokenomics, halving emission schedule, fee burn mechanics
- [x] **Phase 6** — Cobra CLI commands, Docker deployment, testnet preparation
- [x] **Phase 7** — Embedded React 18 & TypeScript 5 web dashboard, explorer, and wallet portal
- [x] **Phase 8–13** — DeFi ecosystem: QubitSwap AMM, Lending & Borrowing, Oracle feeds, LevelDB storage
- [x] **Phase 14** — HydroChain green hydrogen supply chain infrastructure
- [x] **Phase 15** — Real World Asset (RWA) tokenization and compliance
- [x] **Phase 16–17** — Decentralized Storage network & distributed compute task scheduler
- [x] **Phase 18–21** — AI Marketplace, CBDC cross-border settlements, ISO 20022 messaging, Custody
- [x] **Phase 22–25** — SuperApp client protocol, Global multi-region anchors, Mainnet staging, L2 Rollups
- [x] **Phase 26–29** — ESG credit marketplace, Government partnerships, EnergyX grid, Sovereign funds
- [x] **Phase 30–32** — Quantum Internet simulation, Global ESG Network, Autonomous AI execution

---

## 7. License

Distributed under the [MIT License](LICENSE).
