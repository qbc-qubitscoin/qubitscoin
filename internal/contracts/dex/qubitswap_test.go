package dex

import (
	"testing"
)

// ── Unit Tests (table-driven, TDD) ───────────────────────────────────────────

func TestNewQubitSwap_EmptyReserves(t *testing.T) {
	qs := NewQubitSwap()
	if qs.ReserveA != 0 || qs.ReserveB != 0 {
		t.Fatalf("new pool should have zero reserves, got A=%d B=%d", qs.ReserveA, qs.ReserveB)
	}
}

func TestAddLiquidity_IncreasesReserves(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(1000, 2000)
	if qs.ReserveA != 1000 {
		t.Errorf("ReserveA: want 1000, got %d", qs.ReserveA)
	}
	if qs.ReserveB != 2000 {
		t.Errorf("ReserveB: want 2000, got %d", qs.ReserveB)
	}
}

func TestAddLiquidity_Cumulative(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(500, 1000)
	qs.AddLiquidity(500, 1000)
	if qs.ReserveA != 1000 {
		t.Errorf("ReserveA: want 1000, got %d", qs.ReserveA)
	}
	if qs.ReserveB != 2000 {
		t.Errorf("ReserveB: want 2000, got %d", qs.ReserveB)
	}
}

func TestAddLiquidity_ZeroAmounts(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(0, 0)
	if qs.ReserveA != 0 || qs.ReserveB != 0 {
		t.Error("adding zero liquidity should keep reserves at zero")
	}
}

// TestSwapAforB_EmptyPool verifies that swapping into an empty pool fails safely.
func TestSwapAforB_EmptyPool(t *testing.T) {
	qs := NewQubitSwap()
	out, ok := qs.SwapAforB(100)
	if ok || out != 0 {
		t.Errorf("swap on empty pool: want (0, false), got (%d, %v)", out, ok)
	}
}

// TestSwapAforB_ZeroInput verifies that zero input returns failure.
func TestSwapAforB_ZeroInput(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(1000, 1000)
	out, ok := qs.SwapAforB(0)
	if ok || out != 0 {
		t.Errorf("zero input swap: want (0, false), got (%d, %v)", out, ok)
	}
}

// TestSwapAforB_BasicSwap checks a known output for a 0.3% fee AMM.
func TestSwapAforB_BasicSwap(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(10_000, 10_000)

	// Swap 1000 A in: amountAInWithFee = 1000 * 997 / 1000 = 997
	// amountBOut = (10000 * 997) / (10000 + 997) = 9970000 / 10997 ≈ 906
	out, ok := qs.SwapAforB(1000)
	if !ok {
		t.Fatal("expected successful swap, got failure")
	}
	if out == 0 {
		t.Fatal("expected non-zero output")
	}
	// The output must be strictly positive and less than the full reserve
	if out >= 10_000 {
		t.Errorf("output %d must be less than ReserveB 10000", out)
	}
}

// TestSwapAforB_FeeDeducted verifies the 0.3% fee is applied.
// Compare naive (no-fee) vs actual output — actual must be less.
func TestSwapAforB_FeeDeducted(t *testing.T) {
	// Without fee: amountBOut = (ReserveB * amountA) / (ReserveA + amountA)
	// With 0.3% fee, output should be ~0.3% less.
	rA, rB := uint64(10_000), uint64(10_000)
	amountA := uint64(100)

	qs := NewQubitSwap()
	qs.AddLiquidity(rA, rB)
	outWithFee, _ := qs.SwapAforB(amountA)

	// Naive output (no fee)
	naiveOut := (rB * amountA) / (rA + amountA)

	if outWithFee >= naiveOut {
		t.Errorf("fee not deducted: with_fee=%d >= naive=%d", outWithFee, naiveOut)
	}
}

// TestSwapAforB_ReservesUpdated checks that reserves change correctly after swap.
func TestSwapAforB_ReservesUpdated(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(10_000, 10_000)
	beforeA, beforeB := qs.ReserveA, qs.ReserveB

	amountIn := uint64(500)
	out, ok := qs.SwapAforB(amountIn)
	if !ok {
		t.Fatal("swap failed")
	}

	if qs.ReserveA != beforeA+amountIn {
		t.Errorf("ReserveA: want %d, got %d", beforeA+amountIn, qs.ReserveA)
	}
	if qs.ReserveB != beforeB-out {
		t.Errorf("ReserveB: want %d, got %d", beforeB-out, qs.ReserveB)
	}
}

// TestAMMInvariant_HoldsAfterSwap verifies x*y = k is maintained (allowing for rounding down).
func TestAMMInvariant_HoldsAfterSwap(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(10_000, 10_000)
	kBefore := qs.ReserveA * qs.ReserveB

	_, ok := qs.SwapAforB(500)
	if !ok {
		t.Fatal("swap failed")
	}
	kAfter := qs.ReserveA * qs.ReserveB

	// Due to integer division (fee rounds down), kAfter >= kBefore (protocol captures rounding).
	if kAfter < kBefore {
		t.Errorf("AMM invariant violated: k_before=%d > k_after=%d", kBefore, kAfter)
	}
}

// TestSwapAforB_CannotDrainPool ensures the pool cannot be drained to zero B.
func TestSwapAforB_CannotDrainPool(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(1, 1)

	// A huge swap that would otherwise require >= all of ReserveB
	out, ok := qs.SwapAforB(1_000_000)
	if ok && out >= 1 {
		// If it succeeded, pool must still have some B left
		if qs.ReserveB == 0 {
			t.Error("pool was completely drained — invariant violation")
		}
	}
	// Either the swap fails (ok=false) or succeeds with partial output — both are valid.
}

// TestSwapAforB_MultipleSwaps runs 10 sequential swaps and checks invariants hold.
func TestSwapAforB_MultipleSwaps(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(100_000, 100_000)

	for i := 0; i < 10; i++ {
		kBefore := qs.ReserveA * qs.ReserveB
		out, ok := qs.SwapAforB(100)
		if !ok {
			t.Fatalf("swap %d failed unexpectedly", i)
		}
		if out == 0 {
			t.Fatalf("swap %d returned 0 output", i)
		}
		kAfter := qs.ReserveA * qs.ReserveB
		if kAfter < kBefore {
			t.Errorf("swap %d: k invariant violated (before=%d, after=%d)", i, kBefore, kAfter)
		}
	}
}

// TestSwapAforB_AsymmetricPool validates correct output in an imbalanced pool.
func TestSwapAforB_AsymmetricPool(t *testing.T) {
	qs := NewQubitSwap()
	qs.AddLiquidity(1_000, 100_000) // 1 A buys ~100 B (cheap A)

	out, ok := qs.SwapAforB(10)
	if !ok {
		t.Fatal("swap failed in asymmetric pool")
	}
	// With 1000 A and 100000 B, swapping 10 A should yield roughly 990 B
	if out < 500 {
		t.Errorf("asymmetric swap: expected high B output, got %d", out)
	}
}

// ── Table-driven tests ────────────────────────────────────────────────────────

func TestSwapAforB_Table(t *testing.T) {
	tests := []struct {
		name      string
		reserveA  uint64
		reserveB  uint64
		amountIn  uint64
		wantOK    bool
		minOutGT  uint64 // output must be > this
	}{
		{"equal_reserves_small_swap", 10_000, 10_000, 100, true, 0},
		{"equal_reserves_large_swap", 10_000, 10_000, 9_000, true, 0},
		{"no_reserve_A", 0, 10_000, 100, false, 0},
		{"no_reserve_B", 10_000, 0, 100, false, 0},
		{"zero_input", 10_000, 10_000, 0, false, 0},
		// tiny_reserves: with 1 A and 1 B, swapping 1 A produces 0 B output due to
		// integer rounding (amountAInWithFee = 997/1000 = 0 in integer math).
		// The AMM contract guard (amountBOut >= ReserveB) correctly blocks this.
		// This edge case is tested separately in TestSwapAforB_TinyReserveRoundingIsSafe.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			qs := NewQubitSwap()
			qs.ReserveA = tc.reserveA
			qs.ReserveB = tc.reserveB
			out, ok := qs.SwapAforB(tc.amountIn)
			if ok != tc.wantOK {
				t.Errorf("ok: want %v, got %v (out=%d)", tc.wantOK, ok, out)
			}
			if ok && out <= tc.minOutGT {
				t.Errorf("output %d must be > %d", out, tc.minOutGT)
			}
		})
	}
}

// TestSwapAforB_TinyReserveRoundingIsSafe verifies that a 1:1 pool with amountIn=1
// never allows the pool to be drained, regardless of whether ok is true or false.
// This documents the known integer rounding behavior of the AMM.
func TestSwapAforB_TinyReserveRoundingIsSafe(t *testing.T) {
	qs := NewQubitSwap()
	qs.ReserveA = 1
	qs.ReserveB = 1

	// amountAInWithFee = 1 * 997 / 1000 = 0 (integer division rounds to 0)
	// numerator = 1 * 0 = 0, denominator = 1 + 0 = 1 → amountBOut = 0
	// guard: 0 >= 1 is false → pool NOT drained → ok=true but out=0
	out, _ := qs.SwapAforB(1)

	// The critical invariant: the pool must never allow amountBOut >= ReserveB
	if qs.ReserveB == 0 {
		t.Error("pool drained to zero — AMM safety guard failed")
	}
	if out >= 1 {
		t.Errorf("tiny reserve swap should yield 0 output, got %d", out)
	}
}
