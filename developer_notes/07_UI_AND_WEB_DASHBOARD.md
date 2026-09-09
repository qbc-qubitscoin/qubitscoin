# Developer Note 07: Web UI, Block Explorer & Node Dashboard

## 1. Why an Embedded Web Dashboard is Built Into QBC

Traditional blockchain nodes require running a separate web server, configuring reverse proxies, or downloading bulky desktop apps just to inspect local node status or submit a transaction.

QubitsCoin provides an **embedded, zero-configuration Web Portal** directly out of the node binary:
- **Zero Dependencies**: Uses pure Go `embed.FS` to bundle HTML5/CSS3/JavaScript directly into the compiled binary.
- **Unified Port**: Served directly on the node's HTTP server (`http://127.0.0.1:8545/ui/` and `/dashboard`), sharing the same port as the JSON-RPC engine.
- **Quantum-Safe Wallet**: Allows generating ML-DSA-65 keys and submitting post-quantum transactions directly from the browser.
- **DeFi & Governance Portal**: Native interface for QubitSwap AMM, GreenDAO governance proposals, and CarbonX ESG credits.

---

## 2. How the Embedded Web Server Works

Located in `internal/web/server.go`:
```go
//go:embed static/*
var staticFS embed.FS

func Handler() http.Handler {
    sub, _ := fs.Sub(staticFS, "static")
    fileServer := http.FileServer(http.FS(sub))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet && r.Method != http.MethodHead {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

### Route Multiplexing in `internal/rpc/server.go`:
- `POST /`: Dispatches to JSON-RPC 2.0 handler (`handleRPC`).
- `GET /ui/`: Serves embedded static web assets (`Handler()`).
- `GET /dashboard`: Redirects (HTTP 302) to `/ui/`.
- `GET /healthz`: Returns `200 ok` for container liveness and health probes.
- `GET /`: Returns `405 Method Not Allowed` for RPC protocol conformance.

---

## 3. Web Application Components & Features

### 1. Real-Time Network Dashboard
- Queries `qbc_chainInfo` every 2 seconds.
- Displays:
  - Current Block Height & Tip Block Hash
  - EIP-1559 Dynamic Base Fee (qubits & QBC)
  - Mempool Pending Transaction Count
  - Connected P2P Peer Count
  - Total Fee Burned & Block Reward Subsidies

### 2. Block & Transaction Explorer
- Interactive search box: Look up any block by height or 64-character hash (`qbc_blockByHeight`, `qbc_blockByHash`).
- Inspects block headers: `PrevHash`, `MerkleRoot`, `StateRoot`, `ValidatorAddr`, `GasUsed`, `Timestamp`.
- Displays transaction table with sender, recipient, amount, and fee tips.

### 3. Quantum-Safe Web Wallet
- Generates post-quantum ML-DSA-65 key pairs.
- Derives 32-byte address using SHA-3-256:
  $$\text{Address} = \text{SHA-3-256}(\text{PublicKey})$$
- Inspects account balance and nonce via `qbc_getBalance` and `qbc_getTransactionCount`.
- Transfer dispatcher with dynamic fee tier recommendations from `qbc_feeEstimate`.

### 4. DeFi & Contracts Portal
- **QubitSwap AMM**: Live liquidity pool reserves, swap simulator with constant product $x \cdot y = k$ and 0.3% fee.
- **GreenDAO**: Proposal browser with 20% quorum meter and voting action.
- **CarbonX ESG**: Verified carbon credit registry and retirement certificate generator.
- **RPC Console**: Interactive JSON-RPC 2.0 debugging shell.
