# QubitID Verifiable Credential Schema

**Version**: v1.0
**Phase**: 9 (QubitID Development)

## Overview
The QubitID system uses W3C-compatible Verifiable Credentials (VCs) to represent identity, KYC/AML status, and investor accreditation. To preserve privacy and maintain GDPR compliance, personally identifiable information (PII) is **never** stored on the QubitsCoin blockchain. Instead, off-chain VCs contain the PII, signed by trusted issuers (e.g., regulated KYC providers) using ML-DSA-65 keys. The blockchain only stores Decentralized Identifiers (DIDs) and cryptographic proofs (e.g., zero-knowledge proofs or credential hashes).

## 1. KYC / AML Credential Schema

This schema defines a credential asserting that a user has passed KYC/AML checks by a trusted provider.

```json
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://schema.qubitscoin.org/credentials/kyc/v1"
  ],
  "id": "urn:uuid:3978344f-8596-4c3a-a978-8fcaba3903c5",
  "type": ["VerifiableCredential", "KYCCredential"],
  "issuer": "did:qbc:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "issuanceDate": "2026-09-01T12:00:00Z",
  "expirationDate": "2027-09-01T12:00:00Z",
  "credentialSubject": {
    "id": "did:qbc:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
    "kycStatus": "APPROVED",
    "amlRiskLevel": "LOW",
    "jurisdiction": "US",
    "verificationMethod": "PASSPORT_AND_LIVENESS"
  },
  "proof": {
    "type": "MLDSA65Signature2026",
    "created": "2026-09-01T12:00:05Z",
    "proofPurpose": "assertionMethod",
    "verificationMethod": "did:qbc:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef#keys-1",
    "jws": "eyJhbGciOiJNTERTQTY1Ii...[3309-byte-signature-encoded]..."
  }
}
```

## 2. Investor Accreditation Schema

This schema defines a credential asserting that an entity meets the legal requirements for accredited or professional investor status, required for participation in Phase 10 (GreenDAO) and Phase 14 (HydroChain).

```json
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://schema.qubitscoin.org/credentials/accreditation/v1"
  ],
  "id": "urn:uuid:98765432-1234-5678-abcd-ef0123456789",
  "type": ["VerifiableCredential", "AccreditedInvestorCredential"],
  "issuer": "did:qbc:auditor-address-hex",
  "issuanceDate": "2026-09-02T09:30:00Z",
  "credentialSubject": {
    "id": "did:qbc:user-address-hex",
    "accreditationStatus": "VERIFIED",
    "criteriaMet": ["NetWorthLimit", "IncomeLimit"],
    "jurisdiction": "EU"
  },
  "proof": {
    "type": "MLDSA65Signature2026",
    "proofPurpose": "assertionMethod",
    "verificationMethod": "did:qbc:auditor-address-hex#keys-1",
    "jws": "..."
  }
}
```

## 3. Selective Disclosure (ZK Tooling)
In preparation for Phase 10, QubitID relies on cryptographic commitments. Rather than presenting the full KYC VC to a smart contract, the user presents a Zero-Knowledge Proof (ZKP) that they possess a valid `KYCCredential` signed by an authorized issuer, proving `kycStatus == APPROVED` without revealing the `jurisdiction` or the credential's unique ID.
