package oracle

import (
	"testing"
)

func TestOracle_FinalizeEpoch_Empty(t *testing.T) {
	a := NewAggregator()
	med := a.FinalizeEpoch()
	if med != 0 {
		t.Errorf("expected 0, got %d", med)
	}
	if a.ActiveEpoch != 2 {
		t.Errorf("expected epoch 2, got %d", a.ActiveEpoch)
	}
}

func TestOracle_GetLatestValue_NoData(t *testing.T) {
	a := NewAggregator()
	// ActiveEpoch is 1, so GetLatestValue should return 0, false
	val, ok := a.GetLatestValue()
	if ok || val != 0 {
		t.Errorf("expected 0, false, got %d, %v", val, ok)
	}
}

func TestOracle_FinalizeEpoch_EvenOdd(t *testing.T) {
	// Even number of submissions
	a1 := NewAggregator()
	a1.SubmitData("n1", 10)
	a1.SubmitData("n2", 20)
	a1.SubmitData("n3", 30)
	a1.SubmitData("n4", 40)
	med1 := a1.FinalizeEpoch()
	if med1 != 25 { // (20+30)/2
		t.Errorf("expected 25, got %d", med1)
	}
	
	// Odd number of submissions
	a2 := NewAggregator()
	a2.SubmitData("n1", 10)
	if !a2.SubmitData("n2", 20) {
		t.Errorf("expected true")
	}
	if a2.SubmitData("n2", 30) {
		t.Errorf("expected false for duplicate")
	}
	a2.SubmitData("n3", 50)
	med2 := a2.FinalizeEpoch()
	if med2 != 20 {
		t.Errorf("expected 20, got %d", med2)
	}
}
