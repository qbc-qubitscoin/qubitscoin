# Developer Note 06: Networking, Mempool & JSON-RPC Layer

## 1. Why P2P & JSON-RPC Were Structured This Way

Blockchains must communicate peer-to-peer while exposing standard interfaces to wallets, block explorers, and dApps:
- **P2P Layer**: Uses explicit binary framing over TCP, authenticated by post-quantum ML-KEM-768 key encapsulation and AES-256-GCM encryption.
- **Mempool Layer**: Thread-safe priority queue ordered by effective tip (`GasPrice - BaseFee`), ensuring highest-paying transactions are mined first.
- **JSON-RPC 2.0 Layer**: Standard JSON-RPC over HTTP, compatible with standard Ethereum tooling patterns (`qbc_getBalance`, `qbc_sendRawTransaction`, etc.).

---

## 2. P2P Frame Protocol & Wire Format

Located in `internal/p2p/message.go` and `internal/p2p/secure.go`:

```
+------------------+-------------------+--------------------+------------------------+
| Magic (4 bytes)  | MsgType (1 byte)  | Length (4 bytes)   | Encrypted GCM Payload  |
| 0x51 42 43 01    | 0x01..0x0A        | Big-Endian uint32  | Ciphertext + 16B Tag   |
+------------------+-------------------+--------------------+------------------------+
```

### Handshake Protocol (Post-Quantum Key Exchange)
1. **Hello**: Node sends `[NodeID, PublicKey, ListenAddr]`.
2. **HelloResp**: Remote accepts connection.
3. **KEM Encapsulation**: Initiator calls `mlkem768.Encapsulate(remotePubKEM)` and generates 32-byte shared secret.
4. **Signature**: Initiator signs transcript `SHA3-256(ciphertext ‖ remoteNodeID)`.
5. **Secure Connection Established**: Both sides derive session keys for AES-256-GCM authenticated encryption.

---

## 3. Mempool Priority Heap

Located in `internal/mempool/mempool.go`:
- Built using Go's `container/heap`.
- **Ordering**: Max-heap ordered by `tx.GasPrice` (highest gas price at top).
- **Secondary Ordering**: FIFO order by timestamp when gas prices are equal.
- **Per-Sender Queue Limit**: At most 64 pending transactions per sender to prevent spam DoS attacks.
- **Mempool Eviction**: When pool reaches `maxSize` (default 50,000 txs), lowest fee transactions are evicted.

---

## 4. JSON-RPC 2.0 API Endpoints

Located in `internal/rpc/api.go` and `internal/rpc/server.go`:

| Method | Parameters | Returns | Description |
|---|---|---|---|
| `qbc_chainInfo` | `[]` | `ChainInfo` | Current chain height, tip block hash, base fee, mempool size, peer count |
| `qbc_blockHeight` | `[]` | `uint64` | Current chain height |
| `qbc_blockByHeight` | `[height uint64]` | `BlockInfo` | Block header, transactions, hash at specific height |
| `qbc_blockByHash` | `[hashHex string]` | `BlockInfo` | Block matching 64-char hash |
| `qbc_getBalance` | `[addrHex string]` | `uint64` | Qubit balance of account |
| `qbc_getTransactionCount` | `[addrHex string]` | `uint64` | Nonce of account |
| `qbc_sendRawTransaction` | `[rawTxHex string]` | `string` (txHash) | Validates and submits signed transaction to mempool |
| `qbc_feeEstimate` | `[]` | `FeeEstimate` | Low, standard, and fast gas price recommendations |
| `qbc_gasPrice` | `[]` | `uint64` | Current minimum gas price |

- Supports single requests and JSON-RPC 2.0 batch requests (`[ {...}, {...} ]`).
- Includes `/healthz` HTTP health check endpoint for container probes.
