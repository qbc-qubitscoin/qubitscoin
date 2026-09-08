# QubitsCoin (QBC) — Production Deployment Guide

> **Version**: v0.5.0
> **Last updated**: 2026-05-21
> **Binary**: `qbc-node`
> **Module**: `github.com/qbc-qubitscoin/qubitscoin`

---

## Table of Contents

1. [Production Readiness Audit](#1-production-readiness-audit)
2. [Do You Need a Server?](#2-do-you-need-a-server)
3. [Server Requirements](#3-server-requirements)
4. [Installation Methods](#4-installation-methods)
   - [A. Build from Source](#a-build-from-source)
   - [B. Docker (Recommended)](#b-docker-recommended)
   - [C. Pre-built Binary](#c-pre-built-binary)
5. [First-Time Setup](#5-first-time-setup)
6. [Configuration Reference](#6-configuration-reference)
7. [Running the Node](#7-running-the-node)
   - [7.1 Bare Metal with systemd](#71-bare-metal-with-systemd)
   - [7.2 Docker Compose](#72-docker-compose)
   - [7.3 Docker (single container)](#73-docker-single-container)
8. [Validator / Miner Mode](#8-validator--miner-mode)
9. [JSON-RPC API Usage](#9-json-rpc-api-usage)
10. [Monitoring (Prometheus + Grafana)](#10-monitoring-prometheus--grafana)
11. [Security Hardening](#11-security-hardening)
12. [Backup and Recovery](#12-backup-and-recovery)
13. [Upgrading the Node](#13-upgrading-the-node)
14. [Firewall & Port Reference](#14-firewall--port-reference)
15. [Troubleshooting](#15-troubleshooting)
16. [Known Limitations & Roadmap](#16-known-limitations--roadmap)
17. [Quick Reference Cheat Sheet](#17-quick-reference-cheat-sheet)

---

## 1. Production Readiness Audit

Honest assessment of every subsystem.
`✅ Ready` = tested, stable, can run in production.
`⚠️ Caution` = works but needs extra attention before mainnet.
`🔲 Planned` = not yet implemented, required before mainnet.

| Subsystem                               | Status     | Notes                                                   |
|-----------------------------------------|------------|---------------------------------------------------------|
| **ML-DSA-65 signing** (FIPS 204)        | ✅ Ready   | Cloudflare CIRCL v1.6.3                                 |
| **ML-KEM-768 P2P handshake** (FIPS 203) | ✅ Ready   | Full KEM exchange per connection                        |
| **SHA-3-256 hashing** (FIPS 202)        | ✅ Ready   | All hashes use SHA-3-256                                |
| **AES-256-GCM P2P encryption**          | ✅ Ready   | Post-handshake channel                                  |
| **EIP-1559 fee model**                  | ✅ Ready   | Burn + tip, ±12.5%/block adjustment                     |
| **LevelDB persistence**                 | ✅ Ready   | Blocks + state survive restarts                         |
| **Keystore (Argon2id + AES-256-GCM)**   | ✅ Ready   | Encrypted at rest, atomic writes                        |
| **JSON-RPC API**                        | ✅ Ready   | 9 methods, batch support                                |
| **Prometheus metrics**                  | ✅ Ready   | 10 metrics, /healthz endpoint                           |
| **TOML config system**                  | ✅ Ready   | Mainnet + testnet defaults                              |
| **Cobra CLI**                           | ✅ Ready   | start / wallet / tx / query / version                   |
| **Docker image**                        | ✅ Ready   | Scratch image, ~16 MB                                   |
| **systemd service unit**                | ✅ Ready   | Hardened, auto-restart                                  |
| **Auto-upgrade manager**                | ✅ Ready   | GitHub Releases polling, SHA-3 verify                   |
| **P2P gossip**                          | ✅ Ready   | Dedup cache, hop limiting                               |
| **Chain sync (IBD)**                    | ✅ Ready   | Batch block download from peers                         |
| **Block production**                    | ✅ Ready   | 2-second intervals                                      |
| **WASM smart contracts**                | ✅ Ready   | wazero pure-Go runtime                                  |
| **Multi-validator BFT**                 | ⚠️ Caution | Single-validator only; multi-sig BFT planned for v0.7.0 |
| **Mempool spam protection**             | ⚠️ Caution | MinGasPrice floor + sender limit; no rate-limit yet     |
| **RPC authentication**                  | ⚠️ Caution | No built-in auth; use nginx/Caddy reverse proxy         |
| **State trie (MPT)**                    | ⚠️ Caution | Simple sorted-hash root; Merkle Patricia Trie planned   |
| **Light client**                        | 🔲 Planned | v0.6.0                                                  |
| **Staking / PoS**                       | 🔲 Planned | v0.7.0                                                  |
| **Cross-chain bridge**                  | 🔲 Planned | v1.0.0                                                  |

> **Priority items**: Multi-validator BFT and Merkle Patricia Trie should be completed before mainnet launch.

---

## 2. Do You Need a Server?

**Short answer: Yes, for production. No, for development/testing.**

`qbc-node` is a single static binary with no external runtime dependencies — no JVM, no Python, no external database process. LevelDB is embedded entirely in-process.

| Use Case                      | Server needed?            | Minimum setup                            |
|-------------------------------|---------------------------|------------------------------------------|
| Development / testing         | ❌ No                     | Your laptop is fine                      |
| Running a testnet node        | ✅ Yes (cloud VM)         | 1 vCPU, 1 GB RAM, 20 GB SSD              |
| Running a validator (mainnet) | ✅ Yes (dedicated server) | 4 vCPU, 8 GB RAM, 500 GB SSD             |
| Running a public RPC endpoint | ✅ Yes + load balancer    | 8 vCPU, 16 GB RAM, 1 TB NVMe             |
| Wallet use only               | ❌ No                     | Laptop with RPC pointed at a public node |

---

## 3. Server Requirements

### Minimum (Testnet Node)

| Resource   | Requirement                                       |
|------------|---------------------------------------------------|
| CPU        | 1 vCPU (x86-64 or ARM64)                          |
| RAM        | 1 GB                                              |
| Disk       | 20 GB SSD (chain grows ~1 GB/day at full load)    |
| OS         | Linux (Ubuntu 22.04+ recommended), macOS, Windows |
| Network    | 10 Mbps stable uplink, public IP for P2P          |
| Ports open | 8765/TCP (P2P)                                    |

### Recommended (Validator / Mainnet Node)

| Resource   | Requirement                                                  |
|------------|--------------------------------------------------------------|
| CPU        | 4 vCPU (x86-64)                                              |
| RAM        | 8 GB                                                         |
| Disk       | 500 GB NVMe SSD (RAID-1 preferred)                           |
| OS         | Ubuntu 22.04 LTS or Debian 12                                |
| Network    | 100 Mbps stable, static public IP                            |
| Ports open | 8765/TCP (P2P), 8545/TCP (RPC — internal or reverse-proxied) |

### Recommended Cloud Providers

| Provider     | Instance                                       | Monthly cost (est.) |
|--------------|------------------------------------------------|---------------------|
| AWS          | t3.medium (testnet) / c5.xlarge (mainnet)      | $15–$150            |
| DigitalOcean | s-2vcpu-4gb (testnet) / g-4vcpu-16gb (mainnet) | $24–$126            |
| Hetzner      | CX22 (testnet) / CCX23 (mainnet)               | €4–€45              |
| Google Cloud | e2-medium (testnet) / c2-standard-4 (mainnet)  | $24–$180            |

> **Tip**: Hetzner (EU/US) offers the best price/performance ratio for blockchain nodes.

---

## 4. Installation Methods

### A. Build from Source

**Requirements**: Go 1.22+ (pure-Go, no CGO needed)

```bash
# 1. Clone the repository
git clone https://github.com/qbc-qubitscoin/qubitscoin.git
cd qubitscoin

# 2. Build the release binary
make build

# The binary is now at bin/qbc-node
./bin/qbc-node version
```

**Cross-compile for other platforms**:

```bash
make build-linux      # → bin/qbc-node-linux-amd64
make build-darwin     # → bin/qbc-node-darwin-arm64
make build-windows    # → bin/qbc-node-windows-amd64.exe
make build-all        # all three platforms at once
```

**Install system-wide (Linux)**:

```bash
sudo cp bin/qbc-node-linux-amd64 /usr/local/bin/qbc-node
sudo chmod +x /usr/local/bin/qbc-node
qbc-node version
```

---

### B. Docker (Recommended)

```bash
# Build the image locally
docker build -t qbc-node:v0.5.0 .

# Or pull from registry (when published)
docker pull ghcr.io/qbc-qubitscoin/qbc-node:v0.5.0
```

---

### C. Pre-built Binary

```bash
# Linux amd64
curl -LO https://github.com/qbc-qubitscoin/qubitscoin/releases/download/v0.5.0/qbc-node-linux-amd64
chmod +x qbc-node-linux-amd64
sudo mv qbc-node-linux-amd64 /usr/local/bin/qbc-node

# Verify the SHA-3-256 checksum published on the release page
qbc-node version
```

---

## 5. First-Time Setup

Run these steps **once** before starting the node for the first time.

### Step 1 — Create the data directory

```bash
mkdir -p ~/.qbc
```

### Step 2 — Copy the config file

```bash
# Mainnet
cp configs/mainnet.toml ~/.qbc/config.toml

# Or for testnet
cp configs/testnet.toml ~/.qbc/config.toml
```

Edit `~/.qbc/config.toml` — at minimum set:

```toml
[node]
data_dir      = "/home/YOUR_USER/.qbc"
miner_enabled = false          # true only for validators
keystore_file = "keystore.json"
```

### Step 3 — Create a wallet (keystore)

```bash
# Always pass the password via env var — never via --password on the CLI
export QBC_PASSWORD="your-strong-password-here"

qbc-node wallet new --config ~/.qbc/config.toml
```

Expected output:

```
✓ New wallet created
  Address  : 3f4a8b2c1d...
  Keystore : /home/user/.qbc/keystore.json

⚠  Back up your keystore file and remember your password.
   There is NO recovery mechanism — lost keys = lost funds.
```

> ⚠️ **CRITICAL**: Back up `~/.qbc/keystore.json` to an offline location immediately
> (USB drive, encrypted cloud storage). If you lose this file or forget the password,
> your funds are permanently inaccessible.

### Step 4 — Verify the wallet address

```bash
qbc-node wallet show --config ~/.qbc/config.toml
```

### Step 5 — Start the node (sync only — no mining)

```bash
qbc-node start --config ~/.qbc/config.toml
```

The node will:
1. Load and decrypt the keystore
2. Open LevelDB block and state databases
3. Apply the genesis block
4. Connect to bootstrap peers
5. Begin syncing the chain
6. Serve JSON-RPC at `127.0.0.1:8545`

---

## 6. Configuration Reference

Full annotated config (`~/.qbc/config.toml`):

```toml
# ── Chain identity ────────────────────────────────────────────────────────────
[chain]
network_id = 1          # 1 = mainnet, 2 = testnet
name       = "QubitsCoin Mainnet"

# ── Node behaviour ────────────────────────────────────────────────────────────
[node]
data_dir      = "~/.qbc"          # all data stored here
keystore_file = "keystore.json"   # relative to data_dir
miner_enabled = false             # true = produce blocks (validators only)
log_level     = "info"            # debug | info | warn | error

# ── P2P networking ────────────────────────────────────────────────────────────
[p2p]
listen_addr   = "0.0.0.0:8765"   # bind all interfaces
external_addr = "1.2.3.4:8765"   # your public IP:port (leave empty to auto-detect)
max_peers     = 25
bootstrap_peers = [
  "qbc-seed1.qubitscoin.io:8765",
  "qbc-seed2.qubitscoin.io:8765",
]

# ── JSON-RPC API ──────────────────────────────────────────────────────────────
[rpc]
enabled       = true
listen_addr   = "127.0.0.1:8545"  # localhost only; use a reverse proxy for public access
cors_origins  = []                 # restrict in production
read_timeout  = "30s"
write_timeout = "30s"

# ── LevelDB storage paths (relative to data_dir) ─────────────────────────────
[storage]
blocks_dir = "blocks"
state_dir  = "state"

# ── Prometheus metrics ────────────────────────────────────────────────────────
[metrics]
enabled     = false               # true = expose /metrics
listen_addr = "0.0.0.0:9090"     # restrict to internal network in production

# ── Auto-upgrade ──────────────────────────────────────────────────────────────
[upgrade]
release_url    = "https://api.github.com/repos/qbc-qubitscoin/qubitscoin/releases/latest"
check_interval = "1h"
auto_apply     = false            # NEVER set true on mainnet validators
```

### Environment Variables

| Variable           | Description                          | Example                            |
|--------------------|--------------------------------------|------------------------------------|
| `QBC_PASSWORD`     | Keystore decryption password         | `export QBC_PASSWORD="hunter2"`    |
| `GRAFANA_PASSWORD` | Grafana admin password (Docker only) | `export GRAFANA_PASSWORD="secret"` |

> **Security rule**: never pass `--password` on the command line in production.
> It appears in `ps aux` and shell history. Always use `QBC_PASSWORD`.

---

## 7. Running the Node

### 7.1 Bare Metal with systemd

**This is the recommended method for production validators.**

**Install the binary**:

```bash
sudo cp bin/qbc-node-linux-amd64 /usr/local/bin/qbc-node
sudo chmod +x /usr/local/bin/qbc-node
```

**Create the service user** (never run as root):

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin qbc
sudo mkdir -p /var/lib/qbc /etc/qbc
sudo chown qbc:qbc /var/lib/qbc
```

**Install config and keystore**:

```bash
sudo cp configs/mainnet.toml /etc/qbc/config.toml
# Edit /etc/qbc/config.toml — set data_dir = "/var/lib/qbc"

# Create the keystore as the service user
sudo -u qbc QBC_PASSWORD="$YOUR_PASSWORD" qbc-node wallet new \
  --config /etc/qbc/config.toml
```

**Store the password securely**:

```bash
# Write the env file, readable only by the qbc user
sudo bash -c 'echo "QBC_PASSWORD=your-strong-password" > /etc/qbc/environment'
sudo chown qbc:qbc /etc/qbc/environment
sudo chmod 400 /etc/qbc/environment
```

**Install and enable the systemd unit**:

```bash
sudo cp configs/qbc-node.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable qbc-node
sudo systemctl start qbc-node
```

**Check status**:

```bash
sudo systemctl status qbc-node
journalctl -u qbc-node -f                   # follow live logs
journalctl -u qbc-node --since "1h ago"     # last hour of logs
```

---

### 7.2 Docker Compose

**Prerequisites**: Docker 24+ and Docker Compose v2

```bash
# 1. Copy and fill in the environment file
cp .env.example .env
nano .env          # set QBC_PASSWORD at minimum

# 2. Create the data directory and config
mkdir -p data
cp configs/mainnet.toml data/config.toml
# Edit data/config.toml — set data_dir = "/data"

# 3. Start the node
docker compose up -d

# 4. Follow logs
docker compose logs -f qbc-node

# 5. Verify the node is live
curl -s http://localhost:8545/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qbc_chainInfo","params":null}'
```

**With Prometheus + Grafana monitoring**:

```bash
docker compose --profile monitoring up -d

# Grafana:    http://localhost:3000  (admin / $GRAFANA_PASSWORD)
# Prometheus: http://localhost:9091
# Metrics:    http://localhost:9090/metrics
```

**Stop / restart**:

```bash
docker compose down            # stop containers (data preserved on disk)
docker compose down -v         # stop AND delete all volumes (DESTRUCTIVE)
docker compose restart qbc-node
```

---

### 7.3 Docker (single container)

```bash
docker run -d \
  --name qbc-node \
  --restart unless-stopped \
  -p 8765:8765 \
  -p 127.0.0.1:8545:8545 \
  -p 9090:9090 \
  -v /path/to/data:/data \
  -v /path/to/config.toml:/data/config.toml:ro \
  -e QBC_PASSWORD="your-password" \
  qbc-node:v0.5.0 \
  start --config /data/config.toml --datadir /data --metrics
```

---

## 8. Validator / Miner Mode

A **validator** is a node that produces blocks and earns block rewards.

> ⚠️ In v0.5.0 the chain uses **single-validator consensus**. Only the node whose
> address matches the genesis validator set entry will produce blocks.
> Multi-validator BFT is planned for v0.7.0.

### Enable mining

```toml
# config.toml
[node]
miner_enabled = true
```

Or via the CLI flag:

```bash
qbc-node start --config ~/.qbc/config.toml --miner
```

### Block reward schedule

| Era             | Block range           | Reward/block           | Era total           |
|-----------------|-----------------------|------------------------|---------------------|
| 0               | 1 – 1,000,000         | 45 QBC                 | 45,000,000 QBC      |
| 1               | 1,000,001 – 2,000,000 | 22.5 QBC               | 22,500,000 QBC      |
| 2               | 2,000,001 – 3,000,000 | 11.25 QBC              | 11,250,000 QBC      |
| 3               | 3,000,001 – 4,000,000 | 5.625 QBC              | 5,625,000 QBC       |
| 4               | 4,000,001 – 5,000,000 | 2.8125 QBC             | 2,812,500 QBC       |
| 5+              | …                     | Halves every 1M blocks | …                   |
| **Mined total** |                       |                        | **90,000,000 QBC**  |
| **Pre-mine**    |                       |                        | 10,000,000 QBC      |
| **Hard cap**    |                       |                        | **100,000,000 QBC** |

### Check validator earnings

```bash
ADDR=$(qbc-node wallet show --config ~/.qbc/config.toml | grep Address | awk '{print $2}')
qbc-node query balance --address "$ADDR" --rpc http://127.0.0.1:8545
```

---

## 9. JSON-RPC API Usage

The node exposes a **JSON-RPC 2.0** HTTP API. All requests are `POST` to the root path `/`.

### Available methods

| Method                    | Description                          | Parameters          |
|---------------------------|--------------------------------------|---------------------|
| `qbc_chainInfo`           | Node status, height, peer count      | none                |
| `qbc_blockHeight`         | Current chain height (integer)       | none                |
| `qbc_blockByHeight`       | Block by height                      | `[height: uint64]`  |
| `qbc_blockByHash`         | Block by hash                        | `["<hex-hash>"]`    |
| `qbc_getBalance`          | Account balance in qubits            | `["<hex-address>"]` |
| `qbc_getTransactionCount` | Account nonce                        | `["<hex-address>"]` |
| `qbc_sendRawTransaction`  | Broadcast a signed transaction       | `["<hex-gob-tx>"]`  |
| `qbc_feeEstimate`         | Fee estimates for all priority tiers | none                |
| `qbc_gasPrice`            | Current base fee                     | none                |

### Example: chain info

```bash
curl -s http://127.0.0.1:8545/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qbc_chainInfo","params":null}'
```

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "chain_id": 1,
    "height": 4521,
    "mempool_len": 3,
    "network_id": 1,
    "peer_count": 8,
    "tip_hash": "a1b2c3d4...",
    "version": "v0.5.0"
  }
}
```

### Example: get balance

```bash
curl -s http://127.0.0.1:8545/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qbc_getBalance","params":["3f4a8b2c..."]}'
```

### Example: send a transaction (CLI)

```bash
export QBC_PASSWORD="your-password"

qbc-node tx send \
  --to     3f4a8b2c1d... \
  --amount 1000000000 \
  --nonce  0 \
  --config ~/.qbc/config.toml \
  --rpc    http://127.0.0.1:8545
```

### Example: fee estimate

```bash
curl -s http://127.0.0.1:8545/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qbc_feeEstimate","params":null}'
```

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "base_fee": 10,
    "ultra_low_tip": 0,
    "standard_tip": 1,
    "fast_tip": 5,
    "transfer_ultra_low_qubits": 210,
    "transfer_standard_qubits": 231,
    "transfer_fast_qubits": 315
  }
}
```

### Batch request

```bash
curl -s http://127.0.0.1:8545/ \
  -H "Content-Type: application/json" \
  -d '[
    {"jsonrpc":"2.0","id":1,"method":"qbc_blockHeight","params":null},
    {"jsonrpc":"2.0","id":2,"method":"qbc_gasPrice","params":null}
  ]'
```

---

## 10. Monitoring (Prometheus + Grafana)

### Available metrics

All metrics are prefixed with `qbc_`.

| Metric                                  | Type      | Description                     |
|-----------------------------------------|-----------|---------------------------------|
| `qbc_chain_height`                      | Gauge     | Current confirmed block height  |
| `qbc_peer_count`                        | Gauge     | Connected P2P peers             |
| `qbc_mempool_size`                      | Gauge     | Pending transactions in mempool |
| `qbc_base_fee_qubits`                   | Gauge     | Current block base fee          |
| `qbc_blocks_produced_total`             | Counter   | Blocks produced since startup   |
| `qbc_transactions_processed_total`      | Counter   | Transactions included in blocks |
| `qbc_fee_burned_qubits_total`           | Counter   | Cumulative qubits burned        |
| `qbc_block_production_duration_seconds` | Histogram | Block build latency             |
| `qbc_rpc_requests_total{method}`        | Counter   | RPC requests by method          |
| `qbc_rpc_errors_total{method}`          | Counter   | RPC errors by method            |

### Enable metrics

```toml
# config.toml
[metrics]
enabled     = true
listen_addr = "0.0.0.0:9090"
```

Or via flag:

```bash
qbc-node start --metrics --metrics-addr 0.0.0.0:9090
```

### Start Prometheus + Grafana (Docker)

```bash
docker compose --profile monitoring up -d

# Grafana:     http://localhost:3000  (admin / $GRAFANA_PASSWORD)
# Prometheus:  http://localhost:9091
# Raw metrics: http://localhost:9090/metrics
```

In Grafana, add a Prometheus data source (`http://prometheus:9090`) and build
dashboards using the `qbc_*` metric names above.

### Health check endpoint

```bash
curl http://localhost:9090/healthz
# → ok
```

---

## 11. Security Hardening

### 11.1 Always put a reverse proxy in front of RPC

The RPC server has **no built-in authentication**. Never bind it to a public IP directly.

**Caddy** (`/etc/caddy/Caddyfile`):

```
rpc.yourdomain.com {
    reverse_proxy 127.0.0.1:8545
    basicauth * {
        api_user $2a$14$...   # bcrypt hash of API password
    }
    tls your@email.com
}
```

**nginx** (`/etc/nginx/sites-available/qbc-rpc`):

```nginx
server {
    listen 443 ssl;
    server_name rpc.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/rpc.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/rpc.yourdomain.com/privkey.pem;

    location / {
        proxy_pass         http://127.0.0.1:8545;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;

        limit_req          zone=rpc burst=20 nodelay;
        limit_req_status   429;
    }
}
```

### 11.2 Firewall rules (UFW)

```bash
sudo ufw allow 8765/tcp comment "QBC P2P"     # P2P must be open to the internet
sudo ufw deny  8545/tcp                        # RPC — reverse proxy handles external access
sudo ufw deny  9090/tcp                        # Metrics — internal network only
sudo ufw allow 22/tcp                          # SSH
sudo ufw enable
```

### 11.3 Keystore security

```bash
# Use a password of at least 24 random characters
# Back up to OFFLINE storage immediately after creation
cp ~/.qbc/keystore.json /mnt/usb/qbc-keystore-$(date +%Y%m%d).json

# Prevent the password from appearing in shell history
unset HISTFILE
export QBC_PASSWORD="your-password"
qbc-node start ...

# In systemd, store the password in an EnvironmentFile with 400 permissions
sudo chmod 400 /etc/qbc/environment
sudo chown qbc:qbc /etc/qbc/environment
```

### 11.4 System hardening

```bash
# Disable core dumps (prevents private key extraction from crash dumps)
echo "* hard core 0" | sudo tee -a /etc/security/limits.conf

# Lock down the data directory
sudo chmod 700 /var/lib/qbc
sudo chown -R qbc:qbc /var/lib/qbc
```

### 11.5 Cryptography summary

QBC uses **exclusively NIST PQC algorithms**. No legacy asymmetric crypto exists anywhere in the codebase.

| Algorithm   | Standard        | Purpose                                     |
|-------------|-----------------|---------------------------------------------|
| ML-DSA-65   | FIPS 204        | Transaction signing, block signing          |
| ML-KEM-768  | FIPS 203        | P2P key encapsulation (handshake)           |
| SHA-3-256   | FIPS 202        | All hashes: blocks, transactions, addresses |
| AES-256-GCM | NIST SP 800-38D | P2P channel encryption, keystore encryption |
| Argon2id    | RFC 9106        | Keystore password key derivation            |

**Prohibited and not present**: RSA, ECDSA, secp256k1, Ed25519, SHA-2, MD5.

---

## 12. Backup and Recovery

### What to back up

| Path                   | Criticality     | Contents                                            |
|------------------------|-----------------|-----------------------------------------------------|
| `~/.qbc/keystore.json` | 🔴 **CRITICAL** | Encrypted private key — cannot recover without this |
| `~/.qbc/config.toml`   | 🟡 Important    | Node config — easy to recreate from template        |
| `~/.qbc/blocks/`       | 🟢 Optional     | Block database — re-syncs from network peers        |
| `~/.qbc/state/`        | 🟢 Optional     | State database — rebuilt from block history         |

> **Rule**: back up only `keystore.json`. Everything else re-syncs from the network automatically.

### Backup commands

```bash
# Back up keystore (do this FIRST, before anything else)
cp ~/.qbc/keystore.json ~/keystore-backup-$(date +%Y%m%d-%H%M%S).json
# Move the file to offline storage immediately

# Full data backup (stop the node first for a consistent snapshot)
sudo systemctl stop qbc-node
tar -czf qbc-backup-$(date +%Y%m%d).tar.gz ~/.qbc/
sudo systemctl start qbc-node
```

### Recovery

```bash
# If you have your keystore, recovery is simple
mkdir -p ~/.qbc
cp /path/to/keystore-backup.json ~/.qbc/keystore.json
cp configs/mainnet.toml ~/.qbc/config.toml

# Start the node — it re-syncs the full chain from peers automatically
qbc-node start --config ~/.qbc/config.toml
```

---

## 13. Upgrading the Node

### Manual upgrade (recommended for mainnet validators)

```bash
# 1. Download the new binary
curl -LO https://github.com/qbc-qubitscoin/qubitscoin/releases/download/v0.6.0/qbc-node-linux-amd64

# 2. Verify the SHA-3-256 checksum published on the release page
# sha3sum qbc-node-linux-amd64

# 3. Stop the running node
sudo systemctl stop qbc-node

# 4. Swap the binary (keep the old one as a rollback target)
sudo mv /usr/local/bin/qbc-node /usr/local/bin/qbc-node.bak
sudo mv qbc-node-linux-amd64 /usr/local/bin/qbc-node
sudo chmod +x /usr/local/bin/qbc-node

# 5. Confirm the version
qbc-node version

# 6. Start and monitor
sudo systemctl start qbc-node
journalctl -u qbc-node -f
```

### Rollback

```bash
sudo systemctl stop qbc-node
sudo mv /usr/local/bin/qbc-node.bak /usr/local/bin/qbc-node
sudo systemctl start qbc-node
```

### Auto-upgrade (testnet only)

```toml
[upgrade]
auto_apply     = true
check_interval = "1h"
```

> ⚠️ Never set `auto_apply = true` on a mainnet validator. Always read the release
> notes and verify checksums manually before upgrading.

### Docker upgrade

```bash
docker compose down
docker pull qbc-node:v0.6.0   # or: make docker VERSION=v0.6.0
docker compose up -d
```

---

## 14. Firewall & Port Reference

### Ports the node listens on

| Port   | Protocol | Direction          | Purpose                               | Exposure                                  |
|--------|----------|--------------------|---------------------------------------|-------------------------------------------|
| `8765` | TCP      | Inbound + Outbound | P2P gossip, block/tx relay, peer sync | Open to internet                          |
| `8545` | TCP      | Inbound            | JSON-RPC API                          | Localhost only — reverse proxy for public |
| `9090` | TCP      | Inbound            | Prometheus metrics + /healthz         | Internal network only                     |

### Outbound connections the node makes

| Destination     | Port   | Purpose                                  |
|-----------------|--------|------------------------------------------|
| Bootstrap peers | `8765` | Initial peer discovery                   |
| Network peers   | `8765` | Ongoing gossip and block sync            |
| GitHub API      | `443`  | Release version checks (upgrade manager) |

---

## 15. Troubleshooting

### "keystore file exists but no password supplied"

```bash
export QBC_PASSWORD="your-password"
qbc-node start --config ~/.qbc/config.toml
```

### "open leveldb: …/blocks: resource temporarily unavailable"

Another `qbc-node` process already holds the database lock. Stop it first:

```bash
sudo systemctl stop qbc-node
pkill qbc-node
```

### Stuck at zero peers

1. Confirm your firewall allows **outbound** TCP on port 8765.
2. Check that `external_addr` in config matches your real public IP.
3. Test bootstrap peer reachability:
   ```bash
   nc -zv qbc-seed1.qubitscoin.io 8765
   ```

### `qbc_blockByHeight` returns "block not found" for height 0

The node is still initializing. Wait a few seconds for the genesis block to load and retry.

### High memory usage

The node keeps all block headers in memory for fast height-based lookups.
At ~200 bytes per block header, this is negligible up to millions of blocks.
A pruning cache will replace the slice approach in v0.6.0.

### "baseFee dropped to minimum" in logs

Expected when the mempool is empty. The floor is 1 qubit/gas (`MinBaseFee`).
The fee rises automatically as transactions arrive.

### Syncing slowly

```bash
# Check peer count
qbc-node query chain --rpc http://127.0.0.1:8545

# Increase max_peers in config (up to ~50)
# Check disk I/O on Linux
iostat -x 1
```

### "decryption failed — wrong password?"

The Argon2id KDF takes ~2 seconds on first unlocking by design. If you are certain
the password is correct, wait. If not, there is no bypass.

---

## 16. Known Limitations & Roadmap

### Current limitations (v0.5.0)

| Limitation                        | Impact                                    | Fix in                       |
|-----------------------------------|-------------------------------------------|------------------------------|
| Single-validator consensus        | Chain halts if the validator goes offline | v0.7.0 (multi-validator BFT) |
| Simple state root (sorted hash)   | Not a full Merkle Patricia Trie           | v0.6.0                       |
| No built-in RPC authentication    | Must use a reverse proxy                  | v0.6.0                       |
| `blockByHash` does a linear scan  | Slow on long chains                       | v0.6.0 (hash index in DB)    |
| Mempool has no TTL-based eviction | Can grow without bound under spam         | v0.6.0                       |
| In-memory chain slice             | ~200 bytes/block; ~2 GB at 10M blocks     | v0.6.0 (pruning)             |
| No light client                   | Full node required to verify state        | v0.8.0                       |
| No staking / PoS                  | No stake-based validator rotation         | v0.7.0                       |

### Roadmap

```
v0.5.0  (current) — Production infrastructure complete
  ✅ Persistent storage (LevelDB)
  ✅ JSON-RPC API (9 methods)
  ✅ Cobra CLI
  ✅ Encrypted keystore (Argon2id + AES-256-GCM)
  ✅ Docker + systemd
  ✅ Prometheus metrics

v0.6.0  — Chain hardening
  🔲 Merkle Patricia Trie state root
  🔲 RPC: optional JWT authentication
  🔲 RPC: hash-indexed block lookup
  🔲 Mempool: TTL-based eviction
  🔲 Consensus: block pruning + height→hash index

v0.7.0  — Multi-validator BFT
  🔲 Tendermint-style dBFT with ≥ 3 validators
  🔲 On-chain staking contract
  🔲 Validator rotation

v0.8.0  — Ecosystem
  🔲 Light client (SPV proofs)
  🔲 EVM-compatible ABI encoding for WASM
  🔲 WebSocket RPC (real-time subscriptions)

v1.0.0  — Mainnet launch
  🔲 Third-party security audit
  🔲 Cross-chain bridge (IBC or custom)
  🔲 Block explorer API
```

---

## 17. Quick Reference Cheat Sheet

```bash
# ── Build & install ───────────────────────────────────────────────────────────
git clone https://github.com/qbc-qubitscoin/qubitscoin.git && cd qubitscoin
make build                                   # → bin/qbc-node
sudo cp bin/qbc-node-linux-amd64 /usr/local/bin/qbc-node

# ── First-time setup ──────────────────────────────────────────────────────────
mkdir -p ~/.qbc
cp configs/mainnet.toml ~/.qbc/config.toml
export QBC_PASSWORD="strong-password"
qbc-node wallet new  --config ~/.qbc/config.toml
qbc-node wallet show --config ~/.qbc/config.toml

# ── Start node ────────────────────────────────────────────────────────────────
qbc-node start --config ~/.qbc/config.toml                    # sync only
qbc-node start --config ~/.qbc/config.toml --miner            # + produce blocks
qbc-node start --config ~/.qbc/config.toml --metrics          # + Prometheus
qbc-node start --testnet                                       # testnet defaults

# ── Query ─────────────────────────────────────────────────────────────────────
qbc-node query chain                         # chain status + peer count
qbc-node query balance --address <hex>       # account balance
qbc-node query block   --height 100          # block at height 100
qbc-node query fee                           # current fee tiers
qbc-node version                             # node version

# ── Send a transaction ────────────────────────────────────────────────────────
qbc-node tx send --to <hex> --amount 1000000000 --nonce 0

# ── Docker ────────────────────────────────────────────────────────────────────
docker compose up -d                         # start node
docker compose --profile monitoring up -d   # + Prometheus + Grafana
docker compose logs -f qbc-node             # follow logs
docker compose down                          # stop (data preserved)

# ── systemd ───────────────────────────────────────────────────────────────────
sudo systemctl start   qbc-node
sudo systemctl stop    qbc-node
sudo systemctl restart qbc-node
sudo systemctl status  qbc-node
journalctl -u qbc-node -f                   # follow live logs

# ── RPC via curl ──────────────────────────────────────────────────────────────
curl -s http://127.0.0.1:8545/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qbc_chainInfo","params":null}'

# ── Makefile targets ──────────────────────────────────────────────────────────
make build           # build for current platform
make build-all       # linux + darwin + windows
make test            # all unit tests
make test-race       # tests with race detector
make test-cover      # tests with HTML coverage report
make vet             # go vet all packages
make docker          # build Docker image
make run-node        # build + run local testnet node
make release         # vet + test + build-all
make help            # list all targets
```

---

*QubitsCoin is quantum-resistant by design. All cryptography is NIST PQC:
ML-DSA-65 (FIPS 204) · ML-KEM-768 (FIPS 203) · SHA-3-256 (FIPS 202) · AES-256-GCM.
No RSA, no ECDSA, no secp256k1 anywhere in the codebase.*
