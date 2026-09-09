# Developer Note 07: Universal React + TypeScript UI & Embedded Web Server

## 1. Why a Universal React + TypeScript Architecture

As the QubitsCoin ecosystem scales with post-quantum cryptography, Wazero smart contracts, and real-time node telemetry, the web interface requires enterprise-grade modularity, type safety, and automated component verification.

### Key Architectural Pillars
- **Strict TypeScript 5 & React 18**: Complete type coverage mirroring QBC core Go structures (`ChainInfo`, `BlockInfo`, `TransactionInfo`, `FeeEstimate`).
- **Test-Driven Development (TDD)**: Every client method and UI component is backed by Vitest and React Testing Library unit tests before implementation.
- **Embedded Zero-Configuration Delivery**: Vite compiles production bundles directly into `internal/web/static/`, allowing the Go node binary to bundle and serve the entire SPA via pure Go `embed.FS` with zero runtime dependencies.
- **Universal Development**: Can be developed standalone (`npm run dev` with hot module reloading) or accessed directly through any active node at `http://127.0.0.1:8545/ui/` and `/dashboard`.

---

## 2. Directory & Component Structure

```text
web/
├── package.json               # Vite, React 18, TypeScript, Vitest, Testing Library
├── tsconfig.json              # Strict TypeScript configuration (bundler resolution)
├── vite.config.ts             # Vite build pipeline targeting ../internal/web/static
├── index.html                 # Single-page app root with theme FOUC prevention
└── src/
    ├── main.tsx               # React DOM entry point
    ├── App.tsx                # App root, active tab state, theme manager, node poller
    ├── App.css                # Glassmorphism dark/light CSS with color-scheme support
    ├── types/
    │   └── rpc.ts             # TypeScript interfaces for JSON-RPC 2.0 requests & data models
    ├── services/
    │   └── rpcClient.ts       # Typed QBCClient with single & batch dispatch, RPCError
    ├── components/
    │   ├── Header.tsx         # Node liveness indicator, height pill, theme toggle
    │   ├── Tabs.tsx           # Tab navigation (Dashboard, Explorer, Wallet, AMM, DAO, ESG, RPC)
    │   ├── DashboardTab.tsx   # Live node metrics (Height, Validators, BaseFee, Gas target)
    │   ├── ExplorerTab.tsx    # Search blocks by height/hash, inspect transactions
    │   ├── WalletTab.tsx      # NIST ML-DSA-65 post-quantum address generator & transfers
    │   ├── DexTab.tsx         # QubitSwap constant product AMM ($x \cdot y = k$) calculator
    │   ├── GreenDaoTab.tsx    # Environmental quadratic governance voting
    │   ├── CarbonXTab.tsx     # ESG carbon offset credit registry & permanent retirement
    │   └── RpcConsoleTab.tsx  # Interactive JSON-RPC 2.0 console with pre-filled templates
    └── test/
        ├── setup.ts           # Vitest environment setup (@testing-library/jest-dom)
        ├── rpcClient.test.ts  # TDD tests for typed RPC client & batch calls
        └── components.test.tsx# TDD tests for React UI components & interactions
```

---

## 3. Typed JSON-RPC 2.0 Client (`services/rpcClient.ts`)

The `QBCClient` class provides type-safe methods over the node's JSON-RPC 2.0 engine:

```typescript
export class QBCClient {
  constructor(endpoint: string = 'http://localhost:8545')
  async getChainInfo(): Promise<ChainInfo>
  async getBlockByHeight(height: number): Promise<BlockInfo>
  async getBlockByHash(hash: string): Promise<BlockInfo>
  async getBalance(address: string): Promise<string>
  async getTransactionCount(address: string): Promise<number>
  async sendRawTransaction(rawHex: string): Promise<string>
  async getFeeEstimate(): Promise<FeeEstimate>
  async getGasPrice(): Promise<number>
  async dispatchBatch(calls: { method: string; params: any[] }[]): Promise<any[]>
}
```

### Error Handling & Protocol Conformance
- If the node responds with `{ "error": { "code": -32601, "message": "Method not found" } }`, `dispatch` parses the error and throws an instance of `RPCError`.
- Network and HTTP errors (non-200 responses) throw informative network errors.
- Dynamic endpoint switching allows pointing the dashboard at remote testnet or mainnet nodes without page reload.

---

## 4. Modern Web Guidance & Styling Architecture

The UI adheres strictly to modern web standards:
- **`color-scheme: light dark`**: Native OS theme inheritance with smooth fallback.
- **FOUC Prevention**: Inline `<script>` in `index.html` loads saved theme from `localStorage` before the first paint.
- **Accessible ARIA Semantics**: Form controls, tab lists, and status indicators have explicit `role`, `aria-selected`, `aria-label`, and keyboard focus styles.
- **Design Tokens**: Defined in CSS variables (`--bg-primary`, `--bg-card`, `--accent-cyan`, `--border-color`), using subtle Glassmorphism backdrop filters for a futuristic quantum appearance.

---

## 5. Automated Testing & Verification

### Vitest & React Testing Library (Frontend)
Run the complete frontend test suite:
```bash
cd web
npm test
```
- Tests JSON-RPC client responses, error cases, batching, and endpoint mutations.
- Tests React component state transitions, block searching, wallet generation, AMM arithmetic, governance voting, and RPC execution.

### Go Web Server Testing (`internal/web`)
The embedded Go server continues to be tested with 100% statement coverage:
```bash
go test -v ./internal/web
```
- Tests root `/` serving `index.html`.
- Tests head request handling, `405 Method Not Allowed` on POST, and `404 Not Found` on non-existent assets.

---

## 6. Build & Deployment Workflow

To build and package the React frontend into the Go binary:
1. `cd web`
2. `npm run build`
   - Compiles TypeScript and runs Vite bundler.
   - Emits optimized single-page assets directly into `internal/web/static/`.
3. Re-run `go test ./internal/web` to ensure embed compatibility.
4. When compiling `cmd/node`, `embed.FS` packages the built React SPA directly inside the resulting executable.
