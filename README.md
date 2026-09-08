# QubitsCoin (QBC) Node

A quantum-resistant Layer 1 blockchain written in Go, built exclusively on NIST Post-Quantum Cryptography standards.

[![Go](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![PQC](https://img.shields.io/badge/crypto-PQC%20only-blueviolet)](https://csrc.nist.gov/projects/post-quantum-cryptography)

---

## Cryptography

All cryptography uses NIST PQC standards **exclusively**. Traditional asymmetric cryptography (RSA, ECC, ECDSA) is strictly prohibited.

| Primitive   | Standard | Key / Sig sizes          | Usage                                                              |
|-------------|----------|--------------------------|--------------------------------------------------------------------|
| ML-DSA-65   | FIPS 204 | PK=1952B, Sig=3309B      | Transaction & block signatures, validator votes, upgrade proposals |
| ML-KEM-768  | FIPS 203 | CT=1088B, SS=32B         | P2P key encapsulation (handshake)                                  |
| SHA-3-256   | FIPS 202 | 32-byte digest           | All hashing — addresses, block headers, Merkle trees               |
| AES-256-GCM | —        | 32-byte key, 12-byte IV  | Post-handshake P2P channel encryption (counter nonces)             |

---

## Architecture

```
internal/
  core/         Block, Transaction, Genesis, Merkle tree, Tokenomics
  crypto/       ML-DSA-65 wallet & signing, SHA-3-256, ML-KEM-768
  state/        Account model (StateDB), state transitions,
                block processor (ApplyBlock), gas accounting
  mempool/      Gas-price priority queue
  consensus/    BFT engine, validator set, ML-DSA-65 signed votes
  p2p/          ML-KEM-768 handshake, AES-GCM encrypted transport,
                gossip cache, peer exchange
  vm/           WASM smart contracts via wazero (pure-Go runtime)
  upgrade/      Binary self-updater + on-chain upgrade scheduler
cmd/
  node/         Entry point — Phases 1-5 integration demo
```

---

## Currency & Tokenomics

| Property          | Value                                        |
|-------------------|----------------------------------------------|
| Symbol            | QBC                                          |
| Smallest unit     | qubit  (1 QBC = 1,000,000,000 qubits)        |
| Hard cap          | **100,000,000 QBC**                          |
| Genesis pre-mine  | 10,000,000 QBC (10 %)                        |
| Block rewards     | 90,000,000 QBC emitted via halving schedule  |
| Initial reward    | 45 QBC / block (Era 0)                       |
| Halving interval  | 1,000,000 blocks (~23 days at 2 s/block)     |
| Block time target | 2 seconds                                    |
| Block gas limit   | 30,000,000 gas                               |

### Emission Schedule

| Era | Block Range           | Reward / Block  | Era Total      |
|-----|-----------------------|-----------------|----------------|
| 0   | 1 – 1,000,000         | 45.0000 QBC     | 45,000,000 QBC |
| 1   | 1,000,001 – 2,000,000 | 22.5000 QBC     | 22,500,000 QBC |
| 2   | 2,000,001 – 3,000,000 | 11.2500 QBC     | 11,250,000 QBC |
| 3   | 3,000,001 – 4,000,000 | 5.6250 QBC      | 5,625,000 QBC  |
| …   | …                     | halves each era | …              |
| 32+ | —                     | 0 QBC           | —              |

> Validator income per block = **block reward + gas fees collected from all transactions in the block**.

---

## Gas

| Operation             | Base gas cost                  |
|-----------------------|--------------------------------|
| Transfer (TxTransfer) | 21,000                         |
| Deploy  (TxDeploy)    | 200,000 + 68 × bytecode length |
| Call    (TxCall)      | 50,000  + 16 × calldata length |
| Minimum gas price     | 1,000 qubits / gas             |

---

## P2P Handshake (ML-KEM-768 + ML-DSA-65)

1. Each node holds a persistent ML-DSA-65 identity keypair.
2. In connection, the initiator sends its ML-DSA-65 public key.
3. The responder encapsulates a shared secret with ML-KEM-768 and signs the ciphertext.
4. Both sides derive a 256-bit AES-GCM session key from the shared secret.
5. All later messages are encrypted with AES-256-GCM (counter nonce).

---

## Smart Contracts (WASM via wazero)

- Contracts are compiled to WebAssembly and stored on-chain by hash.
- The pure-Go [wazero](https://github.com/tetratelabs/wazero) runtime executes contracts with no CGo dependency.
- Host functions exposed to contracts: `storage_get`, `storage_set`, `emit_log`, `get_caller`, `get_block_height`, `get_value`.
- Gas is metered per host-function call.
- Call data layout: `[1-byte funcNameLen][funcName bytes][8-byte big-endian uint64 params…]`

---

## Auto-Upgrade System

**Binary self-updater**
- Polls GitHub Releases API on a configurable interval.
- Downloads the new binary, verifies SHA-3-256 checksum.
- Atomically rename + re-exec (POSIX) / restart (Windows).

**On-chain upgrade scheduler**
- Validators sign `Proposal{targetVersion, targetHeight}` with ML-DSA-65.
- Upgrade fires when > 2/3 of voting power has signed.
- Consensus engine calls `OnBlock(height)` to trigger at the right height.

---

## Dependencies

| Package                           | Version  | Purpose                        |
|-----------------------------------|----------|--------------------------------|
| `github.com/cloudflare/circl`     | v1.6.3   | ML-DSA-65 + ML-KEM-768 (CIRCL) |
| `github.com/tetratelabs/wazero`   | v1.11.0  | Pure-Go WASM runtime           |
| `golang.org/x/crypto`             | v0.51.0  | SHA-3-256 (FIPS 202)           |

---

## Build & Run

```bash
# Build
go build ./cmd/node

# Run demo (Phases 1-5)
./node
```

Expected output includes:
- ML-DSA-65 wallet generation
- Genesis block construction
- Emission schedule table
- `BlockReward()` spot-checks
- Circulating supply at various heights
- CounterContract WASM deploy + 5 `increment()` calls
- `ApplyBlock` reward crediting verification
- Live consensus blocks production with reward logging

---

## Development Phases

- [x] **Phase 1** — Core primitives (crypto, block, transaction, genesis, Merkle)
- [x] **Phase 2** — State & mempool (StateDB, gas-price priority queue)
- [x] **Phase 3** — BFT consensus engine & P2P networking (ML-KEM-768 handshake)
- [x] **Phase 4** — WASM VM (wazero) + binary & on-chain auto-upgrade system
- [x] **Phase 5** — Tokenomics: halving schedule, fee collection, `ApplyBlock` reward crediting
- [x] **Phase 6** — CLI (Cobra), testnet preparation, fuzz testing

---

## License

MIT


## Advanced Development Phases (14 - 32)
- [x] **Phase 14** — HydroChain Infrastructure
- [x] **Phase 15** — RWA Tokenization
- [x] **Phase 16-17** — Decentralized Storage & Compute
- [x] **Phase 18-21** — AI Marketplace, CBDC, ISO20022, Custody
- [x] **Phase 22-25** — SuperApp, Global Expansion, Mainnet, Layer-2 Rollups
- [x] **Phase 26-29** — ESG Markets, Gov Partnerships, Energy Exchange, Sovereign Funds
- [x] **Phase 30-32** — Quantum Internet, Global ESG Network, Autonomous AI
