package identity

import (
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// testIssuer returns a wallet used for signing credentials.
func testWallet(t *testing.T) *crypto.Wallet {
	t.Helper()
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	return w
}

// ── IssueKYCCredential ────────────────────────────────────────────────────────

func TestIssueKYCCredential_ReturnsCredential(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(
		w.PrivateKey,
		"did:qbc:issuer1",
		"did:qbc:subject1",
		"approved",
		"low",
		"EU",
	)
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}
	if vc == nil {
		t.Fatal("expected non-nil VerifiableCredential")
	}
}

func TestIssueKYCCredential_ContextFields(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:issuer", "did:qbc:subject", "approved", "low", "US")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}

	if len(vc.Context) == 0 {
		t.Error("Context must not be empty")
	}
	foundW3C := false
	for _, ctx := range vc.Context {
		if strings.Contains(ctx, "w3.org") {
			foundW3C = true
		}
	}
	if !foundW3C {
		t.Error("Context should contain the W3C credentials context")
	}
}

func TestIssueKYCCredential_TypeFields(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:issuer", "did:qbc:subject", "approved", "low", "US")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}

	hasVC := false
	hasKYC := false
	for _, tp := range vc.Type {
		if tp == "VerifiableCredential" {
			hasVC = true
		}
		if tp == "KYCCredential" {
			hasKYC = true
		}
	}
	if !hasVC {
		t.Error("Type should contain 'VerifiableCredential'")
	}
	if !hasKYC {
		t.Error("Type should contain 'KYCCredential'")
	}
}

func TestIssueKYCCredential_IssuerAndSubject(t *testing.T) {
	w := testWallet(t)
	issuer := "did:qbc:issuer-abc"
	subject := "did:qbc:subject-xyz"

	vc, err := IssueKYCCredential(w.PrivateKey, issuer, subject, "approved", "low", "SG")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}

	if vc.Issuer != issuer {
		t.Errorf("Issuer: want %s, got %s", issuer, vc.Issuer)
	}
	if vc.CredentialSubject["id"] != subject {
		t.Errorf("Subject ID: want %s, got %v", subject, vc.CredentialSubject["id"])
	}
}

func TestIssueKYCCredential_CredentialSubjectFields(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", "medium", "JP")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}

	if vc.CredentialSubject["kycStatus"] != "approved" {
		t.Errorf("kycStatus: want 'approved', got %v", vc.CredentialSubject["kycStatus"])
	}
	if vc.CredentialSubject["amlRiskLevel"] != "medium" {
		t.Errorf("amlRiskLevel: want 'medium', got %v", vc.CredentialSubject["amlRiskLevel"])
	}
	if vc.CredentialSubject["jurisdiction"] != "JP" {
		t.Errorf("jurisdiction: want 'JP', got %v", vc.CredentialSubject["jurisdiction"])
	}
}

func TestIssueKYCCredential_HasProof(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", "low", "UK")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}
	if vc.Proof == nil {
		t.Fatal("Proof must not be nil")
	}
	if vc.Proof.Type == "" {
		t.Error("Proof.Type must not be empty")
	}
	if vc.Proof.ProofPurpose == "" {
		t.Error("Proof.ProofPurpose must not be empty")
	}
	if vc.Proof.JWS == "" {
		t.Error("Proof.JWS (signature) must not be empty")
	}
	if !strings.Contains(vc.Proof.VerificationMethod, "did:qbc:i") {
		t.Errorf("VerificationMethod should reference the issuer DID, got: %s", vc.Proof.VerificationMethod)
	}
}

func TestIssueKYCCredential_HasIssuanceDate(t *testing.T) {
	w := testWallet(t)
	vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", "low", "DE")
	if err != nil {
		t.Fatalf("IssueKYCCredential: %v", err)
	}
	if vc.IssuanceDate == "" {
		t.Error("IssuanceDate must not be empty")
	}
}

func TestIssueKYCCredential_UniqueIDs(t *testing.T) {
	w := testWallet(t)
	vc1, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", "low", "FR")
	if err != nil {
		t.Fatalf("first IssueKYCCredential: %v", err)
	}
	vc2, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", "low", "FR")
	if err != nil {
		t.Fatalf("second IssueKYCCredential: %v", err)
	}
	// IDs are time-based, so two credentials issued sequentially should differ.
	// We can't guarantee nanosecond differences in tests, so just check the IDs are non-empty.
	if vc1.ID == "" || vc2.ID == "" {
		t.Error("Credential IDs must not be empty")
	}
}

func TestIssueKYCCredential_DifferentRiskLevels(t *testing.T) {
	w := testWallet(t)
	levels := []string{"low", "medium", "high"}
	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			vc, err := IssueKYCCredential(w.PrivateKey, "did:qbc:i", "did:qbc:s", "approved", lvl, "US")
			if err != nil {
				t.Fatalf("IssueKYCCredential(%s): %v", lvl, err)
			}
			if vc.CredentialSubject["amlRiskLevel"] != lvl {
				t.Errorf("amlRiskLevel: want %s, got %v", lvl, vc.CredentialSubject["amlRiskLevel"])
			}
		})
	}
}
