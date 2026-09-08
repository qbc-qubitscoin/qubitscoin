# QubitsCoin Developer Quickstart & Runbook

Welcome to the QubitsCoin core repository. This project comprises a custom blockchain built in Go, featuring a post-quantum cryptographic layer (ML-DSA-65), a WASM smart contract virtual machine (QubitVM), and a comprehensive ecosystem of decentralized applications (Phases 1-32).

This document outlines the step-by-step process for running the project locally for development, and the strategy for moving to a production server environment.

---

## Part 1: Running Locally (Development Mode)

Local development runs a single-node "devnet" or local testnet. This allows you to deploy WASM smart contracts, test the JSON-RPC interface, and interact with the web dashboards without requiring a full peer-to-peer network.

### Step 1: Prerequisites
Ensure you have the following installed on your local machine:
1. **Go (1.21+)**: The core node is written in Go.
2. **TinyGo**: Required to compile Go smart contracts into WebAssembly (WASM) for the QubitVM.
3. **Make** (optional, but recommended for build scripts).

### Step 2: Build the Core Node
Compile the main blockchain node executable.
```bash
# Navigate to the root directory
cd qubitscoin

# Build the node executable
go build -o qbcd ./cmd/node
```

### Step 3: Run the Local Node
Start the node in standalone/dev mode.
```bash
# Run the node
./qbcd
```
*Note: In local mode, the node will begin producing blocks automatically and expose a JSON-RPC server on `http://localhost:8545`.*

### Step 4: Run the Test Suites
The repository contains comprehensive unit tests for all 32 phases (Core, Crypto, QubitVM, Oracles, DeFi, DAOs, etc.).
```bash
# Run all tests recursively
go test -v ./...
```

### Step 5: Start the Web Dashboards
The project contains several frontend web interfaces (e.g., the Web Wallet, GreenDAO).
You can run a simple local HTTP server to view them.
```bash
# Using Python's built-in HTTP server
cd web
python -m http.server 8080
```
Open your browser to `http://localhost:8080/wallet/` or `http://localhost:8080/greendao/`.

---

## Part 2: Production Server Deployment

Moving to production requires transitioning from a local devnet to a distributed Peer-to-Peer (P2P) network. 

### 1. Infrastructure Preparation
- **Servers**: Provision cloud instances (AWS EC2, Google Compute Engine, or bare metal) with at least 4 Cores, 16GB RAM, and NVMe SSDs for fast state I/O.
- **Networking**: Open port `30303` (TCP/UDP) for the P2P network layer, and optionally port `8545` if the node is intended to be a public RPC endpoint.

### 2. Compilation and Binary Distribution
- Do not build on the production server. Use a CI/CD pipeline (e.g., GitHub Actions) to compile static Linux binaries (`GOOS=linux GOARCH=amd64`).
- Distribute the compiled `qbcd` binary to your server nodes.

### 3. Bootstrap Nodes (Seed Nodes)
A blockchain needs initial connection points.
1. Deploy 3 to 5 highly available "Seed Nodes" across different geographic regions.
2. Note their P2P addresses (e.g., `enode://<pubkey>@<ip>:30303`).

### 4. Running the Mainnet Validator Node
On a production server, run the node pointing to the seed nodes and using a secure production configuration.

```bash
# Example Systemd execution command
./qbcd \
  --network mainnet \
  --bootnodes "enode://pubkey1@ip1:30303,enode://pubkey2@ip2:30303" \
  --validator-key /etc/qubitscoin/keys/validator.key \
  --datadir /var/lib/qubitscoin
```

### 5. Process Management and Monitoring
- **Systemd/Docker**: Wrap the execution in a Systemd service file or a Docker container ensuring the process auto-restarts on failure.
- **Telemetry**: Hook the node logs into a monitoring stack (Prometheus + Grafana). Monitor metric endpoints for block propagation times, ML-DSA verification latencies, and memory usage.
- **Security**: Keep validator private keys secure, ideally utilizing Hardware Security Modules (HSMs) or secure cloud enclaves (e.g., AWS KMS) via QubitsCoin's Phase 21 custody integrations.
