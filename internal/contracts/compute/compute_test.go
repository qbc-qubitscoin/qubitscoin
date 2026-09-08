package compute

import "testing"

func TestComputeExecution(t *testing.T) {
	module := NewComputeModule()
	result := module.Execute()
	expected := "Phase 17: Qubit Compute executed successfully"
	
	if result != expected {
		t.Fatalf("Expected %s, got %s", expected, result)
	}
	t.Log("Passed: End-to-end cycle completed for Phase 17")
}
