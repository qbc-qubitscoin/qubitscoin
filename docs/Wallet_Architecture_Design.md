# QubitsCoin Wallet Architecture Design

**Version**: v1.0
**Phase**: 8

## 1. Overview
The QubitsCoin Wallet ecosystem encompasses Mobile (iOS/Android), Web, and Desktop applications. Given QubitsCoin's quantum-safe mandate, the primary challenge is securely storing and managing large keys (ML-DSA-65 private keys are 4,032 bytes, signatures are 3,309 bytes).

## 2. Platform Architectures

### 2.1 Mobile Wallets (iOS & Android)
- **Framework**: React Native or Flutter, bridging to a core Rust/Go cryptographic library.
- **Key Storage**: 
  - iOS: Secure Enclave (wrapped using a randomly generated AES-256-GCM key, where the AES key is protected by biometric Face ID/Touch ID).
  - Android: Android Keystore System (wrapped by BiometricPrompt).
- **Communication**: JSON-RPC over HTTPS/WSS to QubitsCoin nodes.

### 2.2 Desktop Wallets (macOS, Windows, Linux)
- **Framework**: Electron or Tauri.
- **Key Storage**: Keychain Access (macOS), DPAPI (Windows), Secret Service API (Linux).
- **Node Integration**: Desktop wallets will have the option to run a light client or a full bundled `qbc-node` in the background for enhanced privacy.

### 2.3 Web Wallets (Browser Extension & Web App)
- **Framework**: React/TypeScript.
- **Key Storage**: IndexedDB wrapped with a user-provided AES-256-GCM passphrase. For browser extensions, background service workers will manage key lifecycle.

## 3. Advanced Key Management

### 3.1 MPC (Multi-Party Computation)
QubitsCoin integrates Threshold Signature Schemes (TSS) adapted for lattice-based cryptography. 
- **Protocol**: A distributed key generation (DKG) protocol ensures the ML-DSA-65 private key is never assembled in a single location.
- **Use Case**: Institutional custody, minimizing single points of failure.

### 3.2 Hardware Wallet Integration
Current hardware wallets (Ledger, Trezor) have limited memory (RAM) which restricts their ability to compute ML-DSA-65 signatures directly. 
- **Interim Solution**: "Smart Card" hardware models capable of PQC, or utilizing the WebAuthn standard for passkey-based transaction signing at the multi-sig contract layer.
- **Long-term**: Integration with next-generation PQC-compliant hardware devices via APDU commands.

## 4. Multi-Signature Smart Contracts
On-chain multi-sig is implemented via QubitVM (WASM).
- A wallet is deployed as a smart contract storing `N` authorized ML-DSA-65 public keys.
- Transactions require `M` distinct signatures submitted as contract calls before the contract executes the internal asset transfer.
