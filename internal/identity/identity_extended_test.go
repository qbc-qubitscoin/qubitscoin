package identity_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/identity"
)

func TestIssueKYCCredential_Extended(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}

	issuerDID := "did:qbc:issuer-123"
	subjectDID := "did:qbc:subject-456"

	vc, err := identity.IssueKYCCredential(
		w.PrivateKey,
		issuerDID,
		subjectDID,
		"approved",
		"low",
		"US",
	)
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}
	if vc == nil {
		t.Fatal("expected VerifiableCredential, got nil")
	}

	// Check ProofValue is base64 (non-empty)
	if vc.Proof == nil {
		t.Fatal("Proof is nil")
	}
	if vc.Proof.JWS == "" {
		t.Error("Proof.JWS is empty")
	}
	
	// Base64 check
	_, err = base64.StdEncoding.DecodeString(vc.Proof.JWS)
	if err != nil {
		t.Errorf("Proof.JWS is not valid base64. error: %v", err)
	}

	if vc.Proof.VerificationMethod == "" {
		t.Error("expected verification method")
	}

	// Test crypto.Sign error
	badKey := make([]byte, 10) // Invalid key size for ML-DSA-65
	_, err = identity.IssueKYCCredential(badKey, "did:qbc:issuer", "did:qbc:subj", "VERIFIED", "LOW", "US")
	if err == nil {
		t.Error("expected sign error")
	}

	// Test json.Marshal error
	origMarshal := identity.JsonMarshal
	identity.JsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("marshal error")
	}
	_, err = identity.IssueKYCCredential(w.PrivateKey, "did:qbc:issuer", "did:qbc:subj", "VERIFIED", "LOW", "US")
	if err == nil {
		t.Error("expected marshal error")
	}
	identity.JsonMarshal = origMarshal

	if !strings.Contains(vc.Proof.VerificationMethod, issuerDID) {
		t.Errorf("VerificationMethod %q does not contain issuer %q", vc.Proof.VerificationMethod, issuerDID)
	}
}
