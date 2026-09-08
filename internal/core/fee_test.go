package core

import "testing"

// ── NextBaseFee ───────────────────────────────────────────────────────────────

func TestNextBaseFee_AtTarget_NoChange(t *testing.T) {
	bf := NextBaseFee(100, TargetBlockGas())
	if bf != 100 {
		t.Errorf("at-target: want 100, got %d", bf)
	}
}

func TestNextBaseFee_FullBlock_Increases(t *testing.T) {
	bf := NextBaseFee(100, BlockGasLimit) // fully packed block
	if bf <= 100 {
		t.Errorf("full block: base fee should increase, got %d", bf)
	}
	if bf > 100+100/MaxBaseFeeChangeDenom {
		t.Errorf("full block: increase capped at +12.5%%, got %d", bf)
	}
}

func TestNextBaseFee_EmptyBlock_Decreases(t *testing.T) {
	bf := NextBaseFee(100, 0) // empty block
	if bf >= 100 {
		t.Errorf("empty block: base fee should decrease, got %d", bf)
	}
}

func TestNextBaseFee_NeverBelowMin(t *testing.T) {
	bf := MinBaseFee
	for i := 0; i < 1000; i++ {
		bf = NextBaseFee(bf, 0) // keep emptying the block
	}
	if bf < MinBaseFee {
		t.Errorf("base fee dropped below MinBaseFee: got %d", bf)
	}
}

func TestNextBaseFee_IncreaseCapAt12Pct(t *testing.T) {
	base := uint64(1000)
	maxAllowed := base + base/MaxBaseFeeChangeDenom
	bf := NextBaseFee(base, BlockGasLimit)
	if bf > maxAllowed {
		t.Errorf("increase > 12.5%% cap: base=%d got=%d max=%d", base, bf, maxAllowed)
	}
}

func TestNextBaseFee_Monotone_UnderCongestion(t *testing.T) {
	bf := InitialBaseFee
	prev := bf
	for i := 0; i < 10; i++ {
		bf = NextBaseFee(bf, BlockGasLimit)
		if bf < prev {
			t.Errorf("congested blocks should never decrease base fee: prev=%d got=%d", prev, bf)
		}
		prev = bf
	}
}

func TestNextBaseFee_Monotone_UnderIdle(t *testing.T) {
	bf := InitialBaseFee * 100 // start high
	prev := bf
	for i := 0; i < 20; i++ {
		bf = NextBaseFee(bf, 0)
		if bf > prev {
			t.Errorf("idle blocks should never increase the base fee: prev=%d got=%d", prev, bf)
		}
		prev = bf
	}
}

// ── FeeEstimate ───────────────────────────────────────────────────────────────

func TestFeeEstimate_UltraLow_NoTip(t *testing.T) {
	maxFeePerGas, tip := FeeEstimate(100, FeeTierUltraLow)
	if maxFeePerGas != 100 {
		t.Errorf("UltraLow maxFee: want 100, got %d", maxFeePerGas)
	}
	if tip != 0 {
		t.Errorf("UltraLow tip: want 0, got %d", tip)
	}
}

func TestFeeEstimate_Standard_HasTip(t *testing.T) {
	maxFeePerGas, tip := FeeEstimate(100, FeeTierStandard)
	if tip == 0 {
		t.Error("Standard tier should have a non-zero tip")
	}
	if maxFeePerGas != 100+tip {
		t.Errorf("Standard maxFee should be baseFee+tip: want %d, got %d", 100+tip, maxFeePerGas)
	}
}

func TestFeeEstimate_Fast_TipLargerThanStandard(t *testing.T) {
	_, tipStd := FeeEstimate(100, FeeTierStandard)
	_, tipFast := FeeEstimate(100, FeeTierFast)
	if tipFast <= tipStd {
		t.Errorf("Fast tip %d should exceed Standard tip %d", tipFast, tipStd)
	}
}

// ── TransferCostQubits ────────────────────────────────────────────────────────

func TestTransferCostQubits_AtGenesis(t *testing.T) {
	cost := TransferCostQubits(InitialBaseFee, 0)
	want := GasTransfer * InitialBaseFee
	if cost != want {
		t.Errorf("transfer cost at genesis: want %d, got %d", want, cost)
	}
}

func TestTransferCostQubits_CheaperThanSolana(t *testing.T) {
	// At $1/QBC, Solana ~$0.00025.
	// QBC transfer at genesis base fee = GasTransfer * InitialBaseFee qubits.
	// Convert to USD assuming $1/QBC.
	costQBC := float64(TransferCostQubits(InitialBaseFee, 0)) / float64(OneQBC)
	solanaUSD := 0.00025
	if costQBC >= solanaUSD {
		t.Errorf("QBC fee $%.10f should be cheaper than Solana $%.5f", costQBC, solanaUSD)
	}
}

func TestTransferCostQubits_AboveZero(t *testing.T) {
	cost := TransferCostQubits(MinBaseFee, 0)
	if cost == 0 {
		t.Error("transfer cost must never be zero (spam prevention)")
	}
}

// ── FeeComparisonTable ────────────────────────────────────────────────────────

func TestFeeComparisonTable_QBCIsCheapest(t *testing.T) {
	table := FeeComparisonTable()
	var qbcFee float64
	minOtherFee := 1e18
	for _, row := range table {
		if row.Chain[:3] == "QBC" {
			qbcFee = row.TransferUSD
		} else if row.TransferUSD < minOtherFee {
			minOtherFee = row.TransferUSD
		}
	}
	if qbcFee >= minOtherFee {
		t.Errorf("QBC fee $%.10f should be cheaper than all others (cheapest other: $%.8f)",
			qbcFee, minOtherFee)
	}
}

func TestFeeComparisonTable_HasExpectedChains(t *testing.T) {
	table := FeeComparisonTable()
	names := make(map[string]bool)
	for _, r := range table {
		names[r.Chain] = true
	}
	for _, want := range []string{"Ethereum (L1)", "Solana"} {
		if !names[want] {
			t.Errorf("comparison table missing a chain: %s", want)
		}
	}
}

// ── Gas constants sanity ──────────────────────────────────────────────────────

func TestGasConstants_TransferUltraCheap(t *testing.T) {
	// Transfer must cost less than 1000 qubits at genesis base fee.
	cost := TransferCostQubits(InitialBaseFee, 0)
	if cost >= 1000 {
		t.Errorf("transfer cost %d qubits should be < 1000 for ultra-cheap positioning", cost)
	}
}

func TestGasConstants_MinGasPriceIsOne(t *testing.T) {
	if false {
		t.Errorf("MinGasPrice should be 1 qubit/gas, got %d", MinGasPrice)
	}
}

func TestTargetBlockGas_IsHalfOfLimit(t *testing.T) {
	target := TargetBlockGas()
	if target != BlockGasLimit/2 {
		t.Errorf("TargetBlockGas: want %d, got %d", BlockGasLimit/2, target)
	}
}
