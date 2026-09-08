package lending

// Position represents a user's collateralized debt position.
type Position struct {
	CollateralQBC uint64
	DebtQUSD      uint64 // USD stablecoin debt
}

// QubitLend is the money market contract.
type QubitLend struct {
	Positions map[string]*Position
	// Price of 1 QBC in QUSD (e.g., from QubitOracle)
	OraclePrice uint64
}

func NewQubitLend() *QubitLend {
	return &QubitLend{
		Positions: make(map[string]*Position),
		OraclePrice: 100, // Default to $100 per QBC
	}
}

// UpdateOraclePrice receives the median price from the Phase 10 Oracle.
func (ql *QubitLend) UpdateOraclePrice(newPrice uint64) {
	ql.OraclePrice = newPrice
}

// DepositCollateral adds QBC to a user's position.
func (ql *QubitLend) DepositCollateral(user string, amount uint64) {
	if ql.Positions[user] == nil {
		ql.Positions[user] = &Position{}
	}
	ql.Positions[user].CollateralQBC += amount
}

// BorrowQUSD borrows QUSD against deposited QBC.
// Requires a 150% Collateralization Ratio.
func (ql *QubitLend) BorrowQUSD(user string, borrowAmount uint64) bool {
	pos := ql.Positions[user]
	if pos == nil {
		return false
	}

	collateralValueUSD := pos.CollateralQBC * ql.OraclePrice
	newTotalDebt := pos.DebtQUSD + borrowAmount

	// Check if (Collateral Value / New Debt) >= 1.5
	// Rewritten as integers: Collateral Value * 100 >= New Debt * 150
	if collateralValueUSD*100 < newTotalDebt*150 {
		return false // Under-collateralized
	}

	pos.DebtQUSD = newTotalDebt
	return true
}

// Liquidate allows any user to liquidate a position if the collateral ratio falls below 150%.
// For simplicity, we just zero out the debt and take the collateral.
func (ql *QubitLend) Liquidate(targetUser string) bool {
	pos := ql.Positions[targetUser]
	if pos == nil || pos.DebtQUSD == 0 {
		return false // Nothing to liquidate
	}

	collateralValueUSD := pos.CollateralQBC * ql.OraclePrice
	
	// If ratio >= 150%, cannot be liquidated
	if collateralValueUSD*100 >= pos.DebtQUSD*150 {
		return false 
	}

	// Position is underwater. Liquidate it.
	pos.CollateralQBC = 0
	pos.DebtQUSD = 0

	return true
}
