# QubitsCoin DeFi Ecosystem Architecture

**Version**: v1.0
**Phase**: 11 (DEX Development)

## 1. Overview
The QubitsCoin DeFi ecosystem centers around two core protocols deployed on QubitVM: **QubitSwap** (AMM DEX) and **QubitLend** (Over-collateralized lending market).

## 2. QubitSwap (AMM)
QubitSwap utilizes a constant-product formula (`x * y = k`). 
- **Liquidity Pools**: Users can deposit pairs of QRC-20 tokens into a pool. In return, they receive Liquidity Provider (LP) tokens representing their fractional share of the pool.
- **Trading Fees**: A 0.3% fee is applied to every swap, which is added directly to the liquidity pool, thereby increasing the value of the LP tokens over time.

## 3. QubitLend (Money Market)
QubitLend allows users to lock up volatile assets (e.g., QBC) as collateral to borrow stable assets.
- **Oracle Integration**: Collateral values are strictly priced via the Phase 10 **QubitOracle**, relying on the median-aggregator to prevent flash-crash liquidations.
- **Over-Collateralization**: The protocol enforces a strict 150% collateralization ratio. If the value of the collateral drops below this threshold, liquidators can repay the debt in exchange for a discounted portion of the collateral.

## 4. Stablecoin Design & Regulatory Review
**Strategic Decision**: QubitsCoin will **NOT** implement an algorithmic stablecoin mechanism (e.g., seigniorage shares or endogenous collateral like Terra/UST).

**Regulatory Rationale**: Algorithmic stablecoins carry immense regulatory risk and have been heavily targeted by financial regulators (e.g., the US SEC, EU MiCA). To ensure institutional adoption and protect user funds from death spirals, the official QubitsCoin stablecoin ($QUSD) will operate strictly as a **1:1 Fiat-Reserve-Backed** token.
- Off-chain fiat reserves will be held by licensed custodial partners (Phase 21).
- Monthly transparent attestations of reserves will be published on-chain via the QESG Oracle network.
