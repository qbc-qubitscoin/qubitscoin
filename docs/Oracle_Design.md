# QubitOracle & QESG Oracle Design

**Version**: v1.0
**Phase**: 10

## Overview
The QubitsCoin Oracle Network bridges off-chain real-world data to the deterministic on-chain QubitVM. It is divided into two logical segments:
1. **QubitOracle**: General-purpose financial, crypto, and weather data feeds.
2. **QESG Oracle**: Specialized ESG (Environmental, Social, and Governance) scoring, carbon metrics, and physical climate risk data, directly integrated with accredited registries (e.g., Verra, Gold Standard).

## Architecture

### 1. Staked Oracle Nodes
Oracle nodes must stake `QBC` to participate in data reporting. The stake acts as a security bond that can be slashed if the node is proven to have submitted maliciously manipulated data.

### 2. The Aggregator Smart Contract
On-chain contracts do not trust any single oracle node. Instead, the `oracle.go` smart contract implements an aggregator pattern:
- The contract defines an "Epoch" (e.g., every 5 minutes).
- Authorized, staked nodes submit their fetched data points during the Epoch.
- At the end of the Epoch, the contract computes the **Median** of all submitted values.
- The Median is stored in the contract's state as the canonical value for that time period.

Using a median naturally drops statistical outliers, rendering single-node or minority-coalition data poisoning ineffective.

### 3. QESG Oracle Specialization
Unlike standard price feeds, ESG data is highly specialized. QESG Oracle nodes do not scrape self-reported company PDFs. Instead, they strictly query APIs of authorized third-party ESG auditors and carbon registries. The aggregator verifies that the data originates from these specific, cryptographically signed endpoints using TLSNotary or similar proofs.
