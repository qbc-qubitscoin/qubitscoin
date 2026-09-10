package carbonx

import (
	"testing"
)

func TestCarbonX_TransferCredit_Errors(t *testing.T) {
	cx := NewCarbonXMarketplace()
	
	// Mint a credit
	id := cx.MintCredit("alice", "reg", "ser", 2023, 100)
	
	// Transfer when credit not found
	err := cx.TransferCredit(999, "alice", "bob")
	if err == nil || err.Error() != "credit not found" {
		t.Fatalf("expected credit not found, got %v", err)
	}
	
	// Transfer unauthorized sender
	err = cx.TransferCredit(id, "eve", "bob")
	if err == nil || err.Error() != "unauthorized sender" {
		t.Fatalf("expected unauthorized sender, got %v", err)
	}
	
	// Retire the credit
	_, err = cx.RetireCredit(id, "alice", "beneficiary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Transfer credit retired
	err = cx.TransferCredit(id, "alice", "bob")
	if err == nil || err.Error() != "cannot transfer a retired credit" {
		t.Fatalf("expected cannot transfer a retired credit, got %v", err)
	}
}

func TestCarbonX_RetireCredit_Errors(t *testing.T) {
	cx := NewCarbonXMarketplace()
	id := cx.MintCredit("alice", "reg", "ser", 2023, 100)
	
	// Retire when credit not found
	_, err := cx.RetireCredit(999, "alice", "ben")
	if err == nil || err.Error() != "credit not found" {
		t.Fatalf("expected credit not found, got %v", err)
	}
	
	// Retire unauthorized sender
	_, err = cx.RetireCredit(id, "eve", "ben")
	if err == nil || err.Error() != "unauthorized sender" {
		t.Fatalf("expected unauthorized sender, got %v", err)
	}
	
	// Retire once
	_, err = cx.RetireCredit(id, "alice", "ben")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Retire already retired
	_, err = cx.RetireCredit(id, "alice", "ben2")
	if err == nil || err.Error() != "credit is already retired" {
		t.Fatalf("expected credit is already retired, got %v", err)
	}
}

func TestCarbonX_MintCredit_Sequencing(t *testing.T) {
	cx := NewCarbonXMarketplace()
	id1 := cx.MintCredit("a", "r", "s", 2020, 10)
	id2 := cx.MintCredit("b", "r", "s", 2020, 20)
	
	if id1 != 1 {
		t.Errorf("expected id 1, got %d", id1)
	}
	if id2 != 2 {
		t.Errorf("expected id 2, got %d", id2)
	}
	if cx.NextTokenID != 3 {
		t.Errorf("expected next id 3, got %d", cx.NextTokenID)
	}
}
