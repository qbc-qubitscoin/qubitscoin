# CarbonX Registry Integration Architecture

**Version**: v1.0
**Phase**: 13 (CarbonX Marketplace)

## 1. Source of Truth
QubitsCoin **does not** perform novel carbon accounting or attempt to independently verify the scientific validity of a carbon offset project (e.g., measuring tree girth via satellites).

The sole source of truth for the physical existence and validity of a carbon credit relies on **Accredited Carbon Registries** (e.g., Verra, Gold Standard, American Carbon Registry).

## 2. Technical Integration via QESG Oracle
The bridging of assets relies heavily on the **Phase 10 QESG Oracle**.

1. **Data Ingestion**: The QESG Oracle utilizes API connections to the respective carbon registries.
2. **Data Verification**: Using TLSNotary (or similar verifiable web protocols), the oracle nodes generate a cryptographic proof that a specific registry ID (e.g., VCU-12345) exists, is active, and is held in the CarbonX Custody account.
3. **Smart Contract Minting**: The `CarbonX` tokenization smart contract requires this cryptographic proof from the Oracle Aggregator before it will mint the corresponding on-chain tokens.

## 3. Data Schema Mapping
Each tokenized carbon credit on QubitsCoin contains metadata inherently mapping it back to the source:
- `RegistryName` (e.g., "Verra")
- `ProjectID` (e.g., "VCS-105")
- `VintageYear` (e.g., 2021)
- `SerialNumber` (The exact block of credits bridged)
- `Methodology` (e.g., "VM0015 - Avoided Deforestation")

This transparent 1:1 mapping ensures that auditors, buyers, and regulators can seamlessly trace an on-chain token back to its real-world physical verification documents.
