package lending

import (
	"testing"
)

// TestMarketStressLiquidation verifies the Phase 11 Exit Criteria:
// "liquidation logic verified under simulated market-stress scenarios"
func TestMarketStressLiquidation(t *testing.T) {
	market := NewQubitLend()
	
	// Set initial healthy oracle price: 1 QBC = $100
	market.UpdateOraclePrice(100)

	// User deposits 10 QBC ($1000 value)
	market.DepositCollateral("user_bob", 10)
	
	// Max borrow at 150% ratio is $1000 / 1.5 = $666.66
	// User borrows $600 (Healthy, CR = 166%)
	if !market.BorrowQUSD("user_bob", 600) {
		t.Fatalf("Failed to borrow under healthy conditions")
	}

	// Try to liquidate while healthy - should fail
	if market.Liquidate("user_bob") {
		t.Fatalf("Successfully liquidated a healthy position! This is a bug.")
	}

	// --- MARKET STRESS EVENT ---
	// Oracle reports price drop: 1 QBC = $80
	// Collateral value drops to $800.
	// Required for $600 debt is $900.
	// Position is now under-collateralized (CR = 133%).
	market.UpdateOraclePrice(80)

	// Attempt liquidation again
	if !market.Liquidate("user_bob") {
		t.Fatalf("Failed to liquidate underwater position during market stress")
	}

	// Verify position is wiped
	pos := market.Positions["user_bob"]
	if pos.CollateralQBC != 0 || pos.DebtQUSD != 0 {
		t.Fatalf("Liquidation did not clear the position. Collateral: %d, Debt: %d", pos.CollateralQBC, pos.DebtQUSD)
	}

	t.Log("Passed: Liquidation logic successfully triggered and resolved bad debt during simulated oracle price crash.")
}
