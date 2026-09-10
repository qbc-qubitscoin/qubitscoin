# Developer Note 03: Post-Quantum Security & Cryptography

## 1. Why Post-Quantum Cryptography Matters

Most modern blockchains (Bitcoin, Ethereum, Solana) rely on elliptic curve cryptography:
- **ECDSA (secp256k1)** or **Ed25519**.
- Shor's algorithm running on a quantum computer of sufficient qubits can calculate private keys from public keys in polynomial time:
  $$\mathcal{O}(n^3)$$
  This completely breaks the signature security of existing networks.

QubitsCoin is built **post-quantum by default**, implementing standard NIST Post-Quantum Cryptography (PQC) standards finalized in August 2024 (FIPS 204 & FIPS 203).

---

## 2. Cryptographic Primitives in QBC

| Primitive | Implementation | Standard | Key Size | Security Level |
|---|---|---|---|---|
| **Digital Signatures** | Cloudflare Circl `mldsa65` | NIST FIPS 204 (ML-DSA-65) | Pub: 1,952 bytes, Priv: 4,032 bytes, Sig: 3,309 bytes | Category 3 (AES-192 equivalent quantum hardness) |
| **Key Exchange (KEM)** | Cloudflare Circl `mlkem768` | NIST FIPS 203 (ML-KEM-768) | Pub: 1,184 bytes, Ciphertext: 1,088 bytes | Category 3 (Kyber-768 equivalent) |
| **Hashing** | `golang.org/x/crypto/sha3` | NIST FIPS 202 (SHA-3-256) | Digest: 32 bytes | Pre-image resistance: 256 bits, Collision: 128 bits |
| **Key Derivation** | `golang.org/x/crypto/argon2` | Argon2id | 64MB memory, 3 iterations, 4 threads | Memory-hard password hashing |
| **Local Encryption** | `crypto/aes` + `cipher.NewGCM` | AES-256-GCM | Key: 32 bytes, Nonce: 12 bytes | Authenticated Encryption with Associated Data (AEAD) |

---

## 3. How Address Derivation Works

Unlike Ethereum (which takes Keccak-256 and trims to 20 bytes), QBC addresses are the full 32-byte SHA-3-256 hash of the ML-DSA-65 public key:

$$\text{Address} = \text{SHA-3-256}(\text{PublicKey}_{\text{ML-DSA-65}})$$

- Total Address Length: **32 bytes** (256 bits).
- Representation: 64-character lowercase hex string.
- Address Collision Resistance: $2^{128}$ operations, making address collisions practically impossible even for quantum computers (Grover's algorithm gives $2^{128}$ security on 256-bit hashes).

---

## 4. Keystore Encryption Architecture

Located in `internal/keystore/keystore.go`:
Wallets are stored in password-protected JSON files (`keystore.json`):

1. **Password Salt**: 16 cryptographically secure random bytes (`crypto/rand`).
2. **Argon2id Key Derivation**:
   ```go
   derivedKey := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
   ```
3. **AES-256-GCM Encryption**:
   - Random 12-byte initialization vector (IV).
   - Encrypts `[1952 bytes pubkey] ‖ [4032 bytes privkey]`.
4. **Atomic Write Guarantee**:
   - The file is written to `.tmp` first, synced via `f.Sync()`, and renamed via `os.Rename()` to prevent wallet file corruption on unexpected power loss.
