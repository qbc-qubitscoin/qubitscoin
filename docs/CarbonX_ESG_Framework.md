# CarbonX ESG Framework: Double-Counting Prevention

**Version**: v1.0
**Phase**: 13 (CarbonX Marketplace)

## 1. The Double-Counting Problem
"Double-counting" occurs when a single ton of reduced/removed CO2 is claimed by two different entities. In blockchain tokenization, this risk manifests severely if a carbon credit exists simultaneously as an active, tradable asset on a legacy registry (e.g., Verra) AND as an active, tradable token on the blockchain.

If unmitigated, Company A could buy and retire the token on-chain, while Company B buys and retires the underlying traditional asset off-chain, destroying the environmental integrity of the system.

## 2. Prevention Mechanism (The "Bridge-and-Retire" Model)
QubitsCoin CarbonX employs a strict **Two-Way Bridge Protocol** to ensure 1:1 parity and absolute prevention of double-counting:

1. **Tokenization (Bridging On-Chain)**: 
   When a carbon credit is bridged to QubitsCoin, the underlying asset on the traditional registry (Verra/Gold Standard) must be immediately transferred to a dedicated, audited "Custody Wallet/Account" and mathematically locked (or explicitly marked as "Bridged to QubitsCoin"). It cannot be traded off-chain while the token exists.
   
2. **On-Chain Retirement (Burning)**: 
   When a corporate entity buys the tokenized carbon credit on the QubitsCoin DEX and wishes to claim the offset, they execute the `RetireCredit` smart contract function.
   - The on-chain token is permanently burned.
   - A cryptographic "Retirement Certificate" is generated on-chain.
   - The off-chain custodial agent receives the retirement proof and executes the final, permanent retirement of the underlying asset on the traditional registry, mirroring the on-chain action.

## 3. Exit Criteria Attestation
By structurally enforcing that a credit is "locked" off-chain while it is liquid on-chain, and permanently retired off-chain when burned on-chain, registry partners can mathematically confirm there is zero double-counting risk in the CarbonX integration design.
