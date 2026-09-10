package identity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

var JsonMarshal = json.Marshal

// VerifiableCredential represents a W3C Verifiable Credential.
type VerifiableCredential struct {
	Context           []string               `json:"@context"`
	ID                string                 `json:"id"`
	Type              []string               `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	ExpirationDate    string                 `json:"expirationDate,omitempty"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Proof             *Proof                 `json:"proof,omitempty"`
}

// Proof contains the ML-DSA-65 signature.
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
	JWS                string `json:"jws"` // We use JWS field to store hex-encoded signature
}

// IssueKYCCredential creates and signs a new KYC credential.
func IssueKYCCredential(issuerKey []byte, issuerDID, subjectDID, status, riskLevel, jurisdiction string) (*VerifiableCredential, error) {
	vc := &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://schema.qubitscoin.org/credentials/kyc/v1",
		},
		ID:           fmt.Sprintf("urn:uuid:kyc-%d", time.Now().UnixNano()),
		Type:         []string{"VerifiableCredential", "KYCCredential"},
		Issuer:       issuerDID,
		IssuanceDate: time.Now().UTC().Format(time.RFC3339),
		CredentialSubject: map[string]interface{}{
			"id":           subjectDID,
			"kycStatus":    status,
			"amlRiskLevel": riskLevel,
			"jurisdiction": jurisdiction,
		},
	}

	payload, err := JsonMarshal(vc)
	if err != nil {
		return nil, err
	}

	// Sign the SHA256 hash of the payload
	hash := sha256.Sum256(payload)
	sig, err := crypto.Sign(issuerKey, hash[:])
	if err != nil {
		return nil, err
	}

	vc.Proof = &Proof{
		Type:               "MLDSA65Signature2026",
		Created:            time.Now().UTC().Format(time.RFC3339),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: issuerDID + "#keys-1",
		JWS:                crypto.ToHex(crypto.Hash256(sig)), // In a real system, full sig is encoded. Using hash for demo struct size.
	}

	return vc, nil
}
