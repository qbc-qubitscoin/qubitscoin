package carbonx

import (
	"errors"
	"fmt"
	"time"
)

type CreditStatus string

const (
	StatusActive  CreditStatus = "ACTIVE"
	StatusRetired CreditStatus = "RETIRED" // Burned
)

// CarbonCredit represents a tokenized carbon offset.
type CarbonCredit struct {
	TokenID       uint64
	Owner         string
	RegistryName  string
	SerialNumber  string
	VintageYear   uint16
	TonsCO2       uint64
	Status        CreditStatus
	RetirementDoc string // Stores the on-chain certificate data once retired
}

// CarbonXMarketplace manages the tokenization and lifecycle of carbon credits.
type CarbonXMarketplace struct {
	Credits      map[uint64]*CarbonCredit
	NextTokenID  uint64
	OwnerBalance map[string]uint64
}

func NewCarbonXMarketplace() *CarbonXMarketplace {
	return &CarbonXMarketplace{
		Credits:      make(map[uint64]*CarbonCredit),
		NextTokenID:  1,
		OwnerBalance: make(map[string]uint64),
	}
}

// MintCredit tokenizes a verified registry claim.
// In production, this requires cryptographic proof from the Phase 10 QESG Oracle.
func (cx *CarbonXMarketplace) MintCredit(receiver, registry, serial string, vintage uint16, tons uint64) uint64 {
	id := cx.NextTokenID
	cx.Credits[id] = &CarbonCredit{
		TokenID:      id,
		Owner:        receiver,
		RegistryName: registry,
		SerialNumber: serial,
		VintageYear:  vintage,
		TonsCO2:      tons,
		Status:       StatusActive,
	}
	cx.OwnerBalance[receiver] += tons
	cx.NextTokenID++
	return id
}

// TransferCredit moves an active credit to a new owner (Trading).
func (cx *CarbonXMarketplace) TransferCredit(tokenID uint64, from, to string) error {
	credit, exists := cx.Credits[tokenID]
	if !exists {
		return errors.New("credit not found")
	}
	if credit.Status != StatusActive {
		return errors.New("cannot transfer a retired credit")
	}
	if credit.Owner != from {
		return errors.New("unauthorized sender")
	}

	credit.Owner = to
	cx.OwnerBalance[from] -= credit.TonsCO2
	cx.OwnerBalance[to] += credit.TonsCO2

	return nil
}

// RetireCredit permanently burns the token to generate a public offset certificate.
func (cx *CarbonXMarketplace) RetireCredit(tokenID uint64, owner, beneficiary string) (string, error) {
	credit, exists := cx.Credits[tokenID]
	if !exists {
		return "", errors.New("credit not found")
	}
	if credit.Status != StatusActive {
		return "", errors.New("credit is already retired")
	}
	if credit.Owner != owner {
		return "", errors.New("unauthorized sender")
	}

	// Burn the token (mark as retired, remove from liquid balance)
	credit.Status = StatusRetired
	cx.OwnerBalance[owner] -= credit.TonsCO2

	// Generate public retirement certificate
	cert := fmt.Sprintf("CERTIFICATE OF RETIREMENT | Beneficiary: %s | Tons CO2: %d | Registry: %s | Serial: %s | Date: %s",
		beneficiary, credit.TonsCO2, credit.RegistryName, credit.SerialNumber, time.Now().Format(time.RFC3339))
	
	credit.RetirementDoc = cert

	return cert, nil
}
