# QubitsCoin (QBC) — Developer Notes Directory

Welcome to the **QubitsCoin Developer Notes** folder. This directory contains the complete technical architecture notes explaining **WHY** each system was designed and **HOW** it works under the hood.

---

## Directory Index

| File | Subsystem | Description |
|---|---|---|
| [`01_TDD_AND_TESTING_GUIDE.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/01_TDD_AND_TESTING_GUIDE.md) | Quality & Engineering | Test-Driven Development (TDD) workflow, Ginkgo BDD specs, fuzzing, and concurrency testing |
| [`02_CORE_BLOCKCHAIN_AND_TOKENOMICS.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/02_CORE_BLOCKCHAIN_AND_TOKENOMICS.md) | Core Blockchain | Transaction lifecycle, EIP-1559 dynamic base fee, Merkle trees, StateDB, and halving emission curve |
| [`03_POST_QUANTUM_SECURITY.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/03_POST_QUANTUM_SECURITY.md) | Cryptography & Keystore | NIST FIPS 204 ML-DSA-65 digital signatures, ML-KEM-768 quantum key encapsulation, SHA-3-256, and Argon2id wallet encryption |
| [`04_CONSENSUS_ENGINE.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/04_CONSENSUS_ENGINE.md) | Consensus | Deterministic round-robin proposer selection, BFT quorum voting, 2-second block intervals, and state commit pipeline |
| [`05_SMART_CONTRACTS_AND_VM.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/05_SMART_CONTRACTS_AND_VM.md) | WebAssembly VM & DeFi | Wazero pure-Go WASM host runtime, gas metering, storage slots, QubitSwap AMM ($x \cdot y = k$), Lending, CarbonX ESG, GreenDAO, and QubitID |
| [`06_NETWORKING_AND_RPC.md`](file:///C:/workspace/repo/qubitscoin/developer_notes/06_NETWORKING_AND_RPC.md) | P2P & JSON-RPC | Frame wire protocol, AES-256-GCM secure connections, priority Mempool heap, and JSON-RPC 2.0 dispatch engine |

---

## The Rule for Developers

Before implementing any new feature, fix, or smart contract:
1. **Read the corresponding note** to understand current state invariants.
2. **Write unit tests first** in accordance with [`PROJECT_RULES.md`](file:///C:/workspace/repo/qubitscoin/PROJECT_RULES.md).
3. **Update or add to these notes** so that the *Why* and *How* of your changes are permanently documented for future developers.
