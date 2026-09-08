package dex

// QubitSwap represents a basic Automated Market Maker (AMM) Liquidity Pool.
// It implements the constant product formula: x * y = k.
type QubitSwap struct {
	ReserveA uint64 // Reserve of Token A
	ReserveB uint64 // Reserve of Token B
}

func NewQubitSwap() *QubitSwap {
	return &QubitSwap{}
}

// AddLiquidity adds tokens to the reserves. In a real contract, this would
// mint LP tokens to the provider based on their proportional share.
func (qs *QubitSwap) AddLiquidity(amountA, amountB uint64) {
	qs.ReserveA += amountA
	qs.ReserveB += amountB
}

// SwapAforB allows a user to trade Token A for Token B.
// It calculates the output amount ensuring (ReserveA + amountA) * (ReserveB - amountBOut) >= ReserveA * ReserveB.
// It applies a 0.3% fee to amountA.
func (qs *QubitSwap) SwapAforB(amountAIn uint64) (amountBOut uint64, success bool) {
	if amountAIn == 0 || qs.ReserveA == 0 || qs.ReserveB == 0 {
		return 0, false
	}

	// Apply 0.3% fee: amountAInWithFee = amountAIn * 997 / 1000
	amountAInWithFee := (amountAIn * 997) / 1000
	
	// yOut = (y * xIn) / (x + xIn)
	numerator := qs.ReserveB * amountAInWithFee
	denominator := qs.ReserveA + amountAInWithFee
	
	amountBOut = numerator / denominator

	// Ensure we have enough reserves
	if amountBOut >= qs.ReserveB {
		return 0, false
	}

	qs.ReserveA += amountAIn
	qs.ReserveB -= amountBOut

	return amountBOut, true
}
