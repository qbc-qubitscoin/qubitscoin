package carbonx

import (
	"testing"
	"strings"
)

// TestPilotIssuanceToRetirementCycle simulates the Phase 13 Exit Criteria:
// "pilot carbon credit issuance-to-retirement cycle completed end-to-end."
func TestPilotIssuanceToRetirementCycle(t *testing.T) {
	market := NewCarbonXMarketplace()

	// 1. Issuance (Minting)
	// QESG Oracle confirms 1000 tons locked in Verra custody account.
	tokenID := market.MintCredit("project_developer", "Verra", "VCU-999-888-2021", 2021, 1000)

	// Verify Developer received the tokens
	if market.OwnerBalance["project_developer"] != 1000 {
		t.Fatalf("Developer should have 1000 tons in balance")
	}

	// 2. Trading
	// Developer sells the credit to MegaCorp on the QubitSwap DEX.
	err := market.TransferCredit(tokenID, "project_developer", "mega_corp")
	if err != nil {
		t.Fatalf("Failed to trade credit: %v", err)
	}

	// Verify balances updated
	if market.OwnerBalance["project_developer"] != 0 {
		t.Fatalf("Developer balance should be 0 after trade")
	}
	if market.OwnerBalance["mega_corp"] != 1000 {
		t.Fatalf("MegaCorp balance should be 1000 after trade")
	}

	// 3. Retirement (Burn)
	// MegaCorp retires the credit to claim their ESG offset.
	cert, err := market.RetireCredit(tokenID, "mega_corp", "MegaCorp Inc. ESG Report 2026")
	if err != nil {
		t.Fatalf("Failed to retire credit: %v", err)
	}

	// Verify the credit is mathematically burned and unusable
	if market.OwnerBalance["mega_corp"] != 0 {
		t.Fatalf("MegaCorp balance should be 0 after retirement")
	}
	
	// Attempting to trade a retired credit must fail to prevent double-counting
	err = market.TransferCredit(tokenID, "mega_corp", "scammer_corp")
	if err == nil {
		t.Fatalf("CRITICAL BUG: Successfully traded a retired credit! Double-counting risk!")
	}

	// Verify certificate generation
	if !strings.Contains(cert, "CERTIFICATE OF RETIREMENT") {
		t.Fatalf("Invalid certificate format generated")
	}

	t.Logf("Passed: Pilot cycle completed successfully.\nGenerated Certificate: %s", cert)
}
