# Legal and Privacy Review: QubitID Data Handling

**Target**: QubitID Decentralized Identity System
**Phase**: 9
**Regulatory Context**: GDPR (EU), CCPA (California), and equivalent global privacy regimes.

## 1. Scope of Review
This document provides a simulated compliance review of the QubitID architecture regarding the handling of Personally Identifiable Information (PII) for KYC/AML and credentialing workflows.

## 2. Architectural Findings

### 2.1 On-Chain Data Minimization
**Design**: The QubitsCoin blockchain only stores Decentralized Identifiers (DIDs, which resolve to public keys) and cryptographic proofs. No plaintext or encrypted PII (names, dates of birth, national ID numbers) is ever stored on the public ledger.
**Legal Conclusion**: COMPLIANT. By keeping PII strictly off-chain, the network avoids the inherent conflict between blockchain immutability and data privacy laws.

### 2.2 The "Right to be Forgotten" (GDPR Article 17)
**Design**: If a user invokes their Right to Erasure, the off-chain KYC provider (the data controller) deletes the user's PII from their centralized databases. The user also deletes the Verifiable Credential from their local wallet. The blockchain retains the DID and transaction history, but these can no longer be linked to the real-world identity by the KYC provider.
**Legal Conclusion**: COMPLIANT. The blockchain records remain intact (necessary for financial ledger integrity), but the link to PII is severed, satisfying the erasure requirement in the context of decentralized systems.

### 2.3 Cross-Border Data Transfers (GDPR Chapter V)
**Design**: Verifiable Credentials are held locally on the user's device. The user actively chooses when and with whom to share their credential (or a Zero-Knowledge Proof derived from it).
**Legal Conclusion**: COMPLIANT. The user acts as the custodian of their own data, transferring it on a consent-driven, peer-to-peer basis.

## 3. KYC/AML Regulatory Alignment

### 3.1 Financial Action Task Force (FATF) Recommendations
The QubitID framework supports FATF Travel Rule compliance by allowing Virtual Asset Service Providers (VASPs) to exchange verifiable credentials establishing the identities of the originator and beneficiary, *prior* to executing the on-chain transfer on QubitsCoin.

## 4. Exit Criteria Status
- **Legal Sign-Off**: Simulated sign-off achieved. The architecture structurally prevents on-chain GDPR violations.
- **Data Handling**: Approved.
