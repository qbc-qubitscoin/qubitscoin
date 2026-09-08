# GreenDAO Governance Framework

**Version**: v1.0
**Phase**: 12 (GreenDAO Launch)

## 1. Overview
The QubitsCoin GreenDAO is an on-chain treasury managed by token holders, dedicated exclusively to funding, developing, and tokenizing renewable energy projects (solar, wind, hydropower, green hydrogen, and EV charging infrastructure). 

## 2. Funding Criteria
To be eligible for GreenDAO treasury disbursement, a project must meet the following criteria:
1. **Sustainability Threshold**: Must demonstrably result in net-zero or negative carbon emissions over its operational lifecycle.
2. **Economic Viability**: Must present a clear revenue model (e.g., Power Purchase Agreements (PPAs) with local grids) to yield returns to the QToken Engine (Phase 15).
3. **Phase 10 Oracle Verification**: Must integrate with smart meters capable of feeding real-time production data to the QESG Oracle.

## 3. Due Diligence Process
1. **Initial Submission**: Project developers submit a proposal via the GreenDAO portal.
2. **Phase 9 Verification**: The proposing entity must hold a valid `KYCCredential` (Phase 9) to prevent anonymous scams.
3. **Technical Audit**: An independent engineering firm (appointed by the DAO) reviews the feasibility study.
4. **On-Chain Vote**: The community votes on the proposal. A 60% supermajority with a 20% quorum is required.
5. **Disbursement**: Funds are released via milestone-based smart contract tranches.

## 4. Proposal Lifecycle
- `Draft` -> `Active` (Voting Period: 7 days) -> `Passed`/`Rejected` -> `Executing` -> `Completed`.
