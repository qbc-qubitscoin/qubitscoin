# Wallet Security Audit Report

**Target**: QubitsCoin Client-Side Key Handling & MPC Architecture
**Phase**: 8

## 1. Scope
This audit focuses on the theoretical attack surfaces of the QubitsCoin wallet architecture, specifically client-side ML-DSA-65 key handling, storage, and transaction signing.

## 2. Threat Model
- **Physical Device Compromise**: Attacker gains access to an unlocked phone/desktop.
- **Malware/Keylogger**: Device is infected with data-exfiltrating malware.
- **Supply Chain Attack**: Compromised dependencies in wallet frontend frameworks.
- **Quantum Harvest-and-Decrypt**: Future quantum computers attempting to extract keys from network traffic.

## 3. Vulnerability Assessment & Mitigations

### 3.1 Large Key Memory Exfiltration
- **Risk**: ML-DSA-65 keys are large (~4KB). During the signing process, the unencrypted key resides in RAM. Malware could dump the application memory to extract it.
- **Mitigation**: Implement `mlock()` (or OS equivalent) to prevent key material from being paged to disk. Ensure explicit zeroization of memory buffers immediately after the `crypto.Sign()` operation completes. 

### 3.2 Secure Enclave Limitations
- **Risk**: Modern Secure Enclaves (iOS/Android) only natively support elliptic curves (Secp256r1, Ed25519) and RSA. They cannot natively generate or sign with ML-DSA-65.
- **Mitigation**: The wallet must generate the ML-DSA-65 key in user-space, generate a symmetric AES-256-GCM key in the Secure Enclave, and use the enclave to encrypt/decrypt the ML-DSA-65 key. While the key is briefly in user-space during signing, it remains hardware-bound at rest.

### 3.3 Random Number Generation (RNG) Failures
- **Risk**: Lattice-based cryptography requires exceptionally high-entropy randomness. Weak PRNGs completely compromise ML-DSA-65.
- **Mitigation**: Wallet implementations MUST use OS-level cryptographically secure pseudo-random number generators (CSPRNG) (e.g., `/dev/urandom`, `SecRandomCopyBytes`, `Crypto.getRandomValues`).

### 3.4 Web Wallet LocalStorage Extraction
- **Risk**: Browser extensions or web wallets storing keys in `localStorage` are vulnerable to XSS attacks.
- **Mitigation**: Keys must never touch `localStorage`. They must be encrypted via WebCrypto API (AES-GCM) with a strong passphrase-derived key (Argon2id) and stored in `IndexedDB`.

## 4. Conclusion
The proposed wallet architecture adequately addresses the unique challenges of large PQC keys. By leveraging OS-level hardware-backed symmetric encryption and strict memory zeroization, the client-side attack surface is minimized.
