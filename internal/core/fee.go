package core

// ─────────────────────────────────────────────────────────────────────────────
// QBC Fee Model — World's Cheapest On-Chain Fees
//
// EIP-1559 style two-component fee:
//
//   EffectiveFeePerGas = BaseFee + PriorityTip
//   TotalFee           = GasUsed × EffectiveFeePerGas
//
//   BaseFee    → BURNED (reduces circulating supply, deflationary)
//   PriorityTip → credited to the block validator (incentive)
//
// The BaseFee auto-adjusts each block:
//   • Block used > TargetGas  → BaseFee rises  (max +12.5%)
//   • Block used < TargetGas  → BaseFee drops  (max −12.5%)
//   • Target utilization = 50% of BlockGasLimit
//
// This keeps fees near their minimum during normal load while preventing
// spam during congestion — all without ever going above zero for idle blocks.
// ─────────────────────────────────────────────────────────────────────────────

const (
	// InitialBaseFee is the base fee at genesis (qubits per gas).
	// Transfer fee at genesis = GasTransfer × InitialBaseFee = 21 × 10 = 210 qubits.
	InitialBaseFee uint64 = 10

	// MinBaseFee is the lowest the base fee can ever fall.
	// At 1 qubit/gas a transfer costs 21 qubits = 0.000000021 QBC.
	MinBaseFee uint64 = 1

	// MaxBaseFeeChangeDenom is the denominator for the ±12.5% cap per block.
	// delta = baseFee / MaxBaseFeeChangeDenom  (= 12.5% when denom=8)
	MaxBaseFeeChangeDenom uint64 = 8

	// TargetGasRatioDenom splits BlockGasLimit into target.
	// target = BlockGasLimit / TargetGasRatioDenom  (50% when denom=2)
	TargetGasRatioDenom uint64 = 2
)

// TargetBlockGas returns the ideal gas usage per block (50% utilization).
func TargetBlockGas() uint64 { return BlockGasLimit / TargetGasRatioDenom }

// NextBaseFee computes the base fee for the next block given the current
// base fee and the gas used in the current block (EIP-1559 adjustment).
//
//   - gasUsed == target  → no change
//   - gasUsed > target   → increase, capped at +12.5%
//   - gasUsed < target   → decrease, capped at −12.5% (floor: MinBaseFee)
func NextBaseFee(current, gasUsed uint64) uint64 {
	target := TargetBlockGas()
	if gasUsed == target {
		return current
	}

	maxDelta := current / MaxBaseFeeChangeDenom
	if maxDelta == 0 {
		maxDelta = 1
	}

	if gasUsed > target {
		// Scale delta proportionally to how over-target we are, cap at maxDelta.
		over := gasUsed - target
		delta := current * over / target / MaxBaseFeeChangeDenom
		if delta == 0 {
			delta = 1
		}
		if delta > maxDelta {
			delta = maxDelta
		}
		return current + delta
	}

	// gasUsed < target — decrease.
	under := target - gasUsed
	delta := current * under / target / MaxBaseFeeChangeDenom
	if delta >= current-MinBaseFee {
		return MinBaseFee
	}
	return current - delta
}

// ─────────────────────────────────────────────────────────────────────────────
// Fee tiers — convenience helpers for wallets / RPC clients
// ─────────────────────────────────────────────────────────────────────────────

// FeeTier represents a transaction urgency level.
type FeeTier uint8

const (
	FeeTierUltraLow FeeTier = iota // baseFee only, zero tip — cheapest possible
	FeeTierStandard                // baseFee + small tip
	FeeTierFast                    // baseFee + generous tip — prioritized in mempool
)

// FeeEstimate returns (maxFeePerGas, priorityTipPerGas) for a given tier
// and current base fee.
func FeeEstimate(baseFee uint64, tier FeeTier) (maxFeePerGas, priorityTip uint64) {
	switch tier {
	case FeeTierUltraLow:
		return baseFee, 0
	case FeeTierStandard:
		tip := baseFee / 10 // 10% of base fee as tip
		if tip == 0 {
			tip = 1
		}
		return baseFee + tip, tip
	case FeeTierFast:
		tip := baseFee / 2 // 50% of base fee as tip
		if tip == 0 {
			tip = 1
		}
		return baseFee + tip, tip
	default:
		return baseFee, 0
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TransferCostQubits returns the total qubit cost of a simple transfer
// at the given base fee and priority tip per gas.
// ─────────────────────────────────────────────────────────────────────────────
func TransferCostQubits(baseFee, priorityTip uint64) uint64 {
	return GasTransfer * (baseFee + priorityTip)
}

// ─────────────────────────────────────────────────────────────────────────────
// FeeComparison holds the fee data for one chain (for display purposes).
// ─────────────────────────────────────────────────────────────────────────────
type FeeComparison struct {
	Chain       string
	TransferUSD float64 // average transfer fee in USD
	TPSLimit    uint64  // approx peak TPS
}

// FeeComparisonTable returns a list of chains for comparison.
// QBC figures assume InitialBaseFee and a token price of $1.00.
func FeeComparisonTable() []FeeComparison {
	// QBC transfer fee at genesis base fee, assuming $1/QBC:
	//   21 gas × 10 qubits/gas = 210 qubits = 0.00000021 QBC = $0.00000021
	qbcFeeUSD := float64(TransferCostQubits(InitialBaseFee, 0)) / float64(OneQBC)

	return []FeeComparison{
		{Chain: "Ethereum (L1)", TransferUSD: 1.50, TPSLimit: 15},
		{Chain: "BNB Smart Chain", TransferUSD: 0.05, TPSLimit: 100},
		{Chain: "Avalanche", TransferUSD: 0.02, TPSLimit: 4_500},
		{Chain: "Polygon", TransferUSD: 0.002, TPSLimit: 7_000},
		{Chain: "Solana", TransferUSD: 0.00025, TPSLimit: 65_000},
		{Chain: "Sui", TransferUSD: 0.00002, TPSLimit: 120_000},
		{Chain: "QBC (genesis base fee, $1/QBC)", TransferUSD: qbcFeeUSD, TPSLimit: 23_000_000},
	}
}
