package oracle

import (
	"sort"
)

// OracleAggregator simulates the on-chain WASM smart contract state.
type OracleAggregator struct {
	// ActiveEpoch tracks the current epoch ID
	ActiveEpoch uint64
	// Submissions maps a node ID to the value they submitted for the active epoch
	Submissions map[string]uint64
	// FinalizedData maps epoch ID to the finalized median value
	FinalizedData map[uint64]uint64
}

func NewAggregator() *OracleAggregator {
	return &OracleAggregator{
		ActiveEpoch:   1,
		Submissions:   make(map[string]uint64),
		FinalizedData: make(map[uint64]uint64),
	}
}

// SubmitData allows an authorized node to submit a data point.
// Returns false if the node already submitted this epoch.
func (a *OracleAggregator) SubmitData(nodeID string, value uint64) bool {
	if _, exists := a.Submissions[nodeID]; exists {
		return false // Prevent duplicate submissions in same epoch
	}
	a.Submissions[nodeID] = value
	return true
}

// FinalizeEpoch calculates the median of all submissions, records it,
// and increments the epoch. This is the core anti-manipulation mechanism.
func (a *OracleAggregator) FinalizeEpoch() uint64 {
	if len(a.Submissions) == 0 {
		a.ActiveEpoch++
		return 0 // No data
	}

	values := make([]uint64, 0, len(a.Submissions))
	for _, v := range a.Submissions {
		values = append(values, v)
	}

	// Sort ascending
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })

	var median uint64
	n := len(values)
	if n%2 == 1 {
		median = values[n/2]
	} else {
		// Average of the two middle values
		median = (values[n/2-1] + values[n/2]) / 2
	}

	a.FinalizedData[a.ActiveEpoch] = median
	
	// Reset for next epoch
	a.Submissions = make(map[string]uint64)
	a.ActiveEpoch++

	return median
}

// GetLatestValue allows downstream contracts to read the most recently finalized data.
func (a *OracleAggregator) GetLatestValue() (uint64, bool) {
	if a.ActiveEpoch <= 1 {
		return 0, false
	}
	val, exists := a.FinalizedData[a.ActiveEpoch-1]
	return val, exists
}
