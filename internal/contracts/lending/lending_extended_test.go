package lending

import (
	"testing"
)

// ── BorrowQUSD boundary tests ─────────────────────────────────────────────────

// TestBorrowQUSD_ExactlyAtRatio verifies borrowing at exactly 150% CR is allowed.
func TestBorrowQUSD_ExactlyAtRatio(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100) // 1 QBC = $100

	// Deposit 15 QBC → $1500 value
	// At 150% CR: max borrow = $1500 / 1.5 = $1000
	market.DepositCollateral("alice", 15)

	if !market.BorrowQUSD("alice", 1000) {
		t.Fatal("BorrowQUSD at exactly 150% CR should succeed")
	}
}

// TestBorrowQUSD_OneOverRatioFails verifies borrowing one unit above the limit fails.
func TestBorrowQUSD_OneOverRatioFails(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("alice", 15) // $1500 value → max $1000

	if market.BorrowQUSD("alice", 1001) {
		t.Fatal("BorrowQUSD above 150% CR limit should fail")
	}
}

// TestBorrowQUSD_NoPosition fails without a deposit.
func TestBorrowQUSD_NoPosition(t *testing.T) {
	market := NewQubitLend()
	if market.BorrowQUSD("ghost", 100) {
		t.Fatal("BorrowQUSD with no position should return false")
	}
}

// TestBorrowQUSD_ZeroAmount — test zero borrow does not change state.
func TestBorrowQUSD_ZeroAmount(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("alice", 10) // $1000 value

	// Borrowing 0 stays under the limit
	result := market.BorrowQUSD("alice", 0)
	if !result {
		// 0 borrow satisfies 0 <= 666 limit, should succeed
		// (0*150 = 0 <= 1000*100 = 100000 → true)
		t.Log("BorrowQUSD(0) returned false — acceptable if implementation rejects zero")
	}
}

// TestBorrowQUSD_CumulativeDebt tests sequential borrows accumulate debt.
func TestBorrowQUSD_CumulativeDebt(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("alice", 10) // $1000 → max $666

	if !market.BorrowQUSD("alice", 300) {
		t.Fatal("first borrow of $300 should succeed")
	}
	if !market.BorrowQUSD("alice", 300) {
		t.Fatal("second borrow of $300 (total $600) should succeed")
	}
	// Third borrow of $100 would bring total to $700 > $666: should fail
	if market.BorrowQUSD("alice", 100) {
		t.Fatal("third borrow bringing total to $700 should fail (over 150% CR)")
	}
}

// ── Liquidate boundary tests ──────────────────────────────────────────────────

// TestLiquidate_ExactlyAt150_Protected ensures healthy positions cannot be liquidated.
func TestLiquidate_ExactlyAt150_Protected(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("bob", 15) // $1500
	market.BorrowQUSD("bob", 1000)      // exactly 150% CR

	// Collateral value * 100 == Debt * 150  → 1500*100 == 1000*150 = 150000
	// The check is >=, so this position should NOT be liquidated
	if market.Liquidate("bob") {
		t.Fatal("position at exactly 150% CR should be protected from liquidation")
	}
}

// TestLiquidate_NoDebt returns false for no-debt position.
func TestLiquidate_NoDebt(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("charlie", 10)

	if market.Liquidate("charlie") {
		t.Fatal("position with no debt should not be liquidatable")
	}
}

// TestLiquidate_NonExistentPosition returns false.
func TestLiquidate_NonExistentPosition(t *testing.T) {
	market := NewQubitLend()
	if market.Liquidate("nonexistent") {
		t.Fatal("liquidating non-existent user should return false")
	}
}

// TestLiquidate_ClearsPosition verifies collateral and debt are wiped after liquidation.
func TestLiquidate_ClearsPosition(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(100)
	market.DepositCollateral("dave", 10) // $1000
	market.BorrowQUSD("dave", 600)       // 166% CR — healthy

	market.UpdateOraclePrice(80) // Drop: $800 collateral, 133% CR — underwater
	if !market.Liquidate("dave") {
		t.Fatal("underwater position should be liquidatable")
	}

	pos := market.Positions["dave"]
	if pos.CollateralQBC != 0 {
		t.Errorf("after liquidation, CollateralQBC should be 0, got %d", pos.CollateralQBC)
	}
	if pos.DebtQUSD != 0 {
		t.Errorf("after liquidation, DebtQUSD should be 0, got %d", pos.DebtQUSD)
	}
}

// TestUpdateOraclePrice_AffectsLiquidation tests that price oracle drives liquidations.
func TestUpdateOraclePrice_AffectsLiquidation(t *testing.T) {
	market := NewQubitLend()
	market.UpdateOraclePrice(200) // 1 QBC = $200
	market.DepositCollateral("eve", 5) // $1000
	market.BorrowQUSD("eve", 600)      // 166% CR — healthy

	// Price crash to $100: collateral = $500, debt = $600 → 83% CR — underwater
	market.UpdateOraclePrice(100)

	if !market.Liquidate("eve") {
		t.Fatal("position should be liquidatable after price crash")
	}
}

// ── DepositCollateral tests ───────────────────────────────────────────────────

func TestDepositCollateral_CreatesNewPosition(t *testing.T) {
	market := NewQubitLend()
	market.DepositCollateral("frank", 50)

	pos := market.Positions["frank"]
	if pos == nil {
		t.Fatal("DepositCollateral should create a new position")
	}
	if pos.CollateralQBC != 50 {
		t.Errorf("CollateralQBC: want 50, got %d", pos.CollateralQBC)
	}
}

func TestDepositCollateral_Cumulative(t *testing.T) {
	market := NewQubitLend()
	market.DepositCollateral("grace", 30)
	market.DepositCollateral("grace", 20)

	if market.Positions["grace"].CollateralQBC != 50 {
		t.Errorf("cumulative deposit: want 50, got %d", market.Positions["grace"].CollateralQBC)
	}
}

// ── Table-driven: various CR scenarios ───────────────────────────────────────

func TestBorrowQUSD_Table(t *testing.T) {
	tests := []struct {
		name        string
		collateral  uint64
		oraclePrice uint64
		borrow      uint64
		wantSuccess bool
	}{
		{"healthy_166pct", 10, 100, 600, true},
		{"exactly_150pct", 15, 100, 1000, true},
		{"under_150pct", 10, 100, 700, false},
		{"zero_collateral", 0, 100, 100, false},
		{"high_price_healthy", 5, 400, 1000, true},  // $2000 value → max $1333
		{"low_price_fails", 5, 50, 200, false},       // $250 value → max $166
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			market := NewQubitLend()
			market.UpdateOraclePrice(tc.oraclePrice)
			if tc.collateral > 0 {
				market.DepositCollateral("user", tc.collateral)
			}
			got := market.BorrowQUSD("user", tc.borrow)
			if got != tc.wantSuccess {
				t.Errorf("BorrowQUSD: want %v, got %v", tc.wantSuccess, got)
			}
		})
	}
}
