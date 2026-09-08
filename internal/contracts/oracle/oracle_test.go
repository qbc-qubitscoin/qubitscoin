package oracle

import (
	"testing"
)

func TestOracleManipulationResistance(t *testing.T) {
	agg := NewAggregator()

	// Scenario: 5 Oracle Nodes fetching ETH/USD price.
	// True market price is roughly $2000.
	
	// 4 honest nodes submit accurate data
	agg.SubmitData("node_1", 2001)
	agg.SubmitData("node_2", 1999)
	agg.SubmitData("node_3", 2005)
	agg.SubmitData("node_4", 2000)

	// 1 malicious node attempts to poison the data feed to trigger a false liquidation
	// by submitting an astronomically high price.
	agg.SubmitData("node_evil", 9999999)

	// The epoch is finalized, contract calculates the median.
	median := agg.FinalizeEpoch()

	// The expected sorted array: 1999, 2000, 2001, 2005, 9999999
	// The median should be the middle value: 2001
	expected := uint64(2001)

	if median != expected {
		t.Fatalf("Manipulation resistance failed! Expected median %d, but got %d", expected, median)
	}

	// Verify downstream contracts get the correct, uncorrupted value
	latest, ok := agg.GetLatestValue()
	if !ok || latest != expected {
		t.Fatalf("Downstream value incorrect. Expected %d, got %d", expected, latest)
	}

	t.Logf("Passed: Median aggregation successfully ignored the outlier (%d). Final price: %d", 9999999, latest)
}

func TestOracleEvenNodeCount(t *testing.T) {
	agg := NewAggregator()

	agg.SubmitData("node_1", 100)
	agg.SubmitData("node_2", 110)
	agg.SubmitData("node_3", 105)
	agg.SubmitData("node_4", 200) // Outlier

	// Sorted: 100, 105, 110, 200
	// Median of even count is average of middle two: (105 + 110) / 2 = 107
	median := agg.FinalizeEpoch()

	if median != 107 {
		t.Fatalf("Expected 107, got %d", median)
	}
}
