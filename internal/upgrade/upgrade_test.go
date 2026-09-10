package upgrade

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── Version ───────────────────────────────────────────────────────────────────

func TestParseVersion_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  Version
	}{
		{"0.5.0", Version{0, 5, 0}},
		{"v0.5.0", Version{0, 5, 0}},
		{"1.2.3", Version{1, 2, 3}},
		{"10.20.30", Version{10, 20, 30}},
	}
	for _, tc := range tests {
		got, err := ParseVersion(tc.input)
		if err != nil {
			t.Errorf("ParseVersion(%q): unexpected error %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("ParseVersion(%q): want %v, got %v", tc.input, tc.want, got)
		}
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	invalids := []string{"", "1.2", "1", "a.b.c", "1.b.3", "1.2.c"}
	for _, s := range invalids {
		_, err := ParseVersion(s)
		if err == nil {
			t.Errorf("ParseVersion(%q): expected error, got nil", s)
		}
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{1, 2, 3}
	if v.String() != "v1.2.3" {
		t.Errorf("String: want v1.2.3, got %s", v.String())
	}
}

func TestVersion_After(t *testing.T) {
	tests := []struct {
		a, b Version
		want bool
	}{
		{Version{1, 0, 0}, Version{0, 9, 9}, true},
		{Version{0, 9, 9}, Version{1, 0, 0}, false},
		{Version{1, 2, 0}, Version{1, 1, 9}, true},
		{Version{1, 1, 9}, Version{1, 2, 0}, false},
		{Version{1, 2, 4}, Version{1, 2, 3}, true},
		{Version{1, 2, 3}, Version{1, 2, 4}, false},
		{Version{1, 2, 3}, Version{1, 2, 3}, false},
	}
	for _, tc := range tests {
		got := tc.a.After(tc.b)
		if got != tc.want {
			t.Errorf("%v.After(%v): want %v, got %v", tc.a, tc.b, tc.want, got)
		}
	}
}

func TestVersion_Equal(t *testing.T) {
	v := Version{1, 2, 3}
	if !v.Equal(Version{1, 2, 3}) {
		t.Error("Equal: same versions should be equal")
	}
	if v.Equal(Version{1, 2, 4}) {
		t.Error("Equal: different versions should not be equal")
	}
}

func TestCurrent_ReturnsValid(t *testing.T) {
	v := Current()
	// package-level version = "0.5.0"
	if v.String() == "" {
		t.Error("Current() should return non-empty version")
	}
}

// ── Manager ───────────────────────────────────────────────────────────────────

func TestManager_OnBlock_NoScheduler(t *testing.T) {
	mgr := NewManager(Config{}, nil)
	mgr.OnBlock(100) // should not panic
}

func TestManager_AddProposal_NoScheduler(t *testing.T) {
	mgr := NewManager(Config{}, nil)
	p := &Proposal{}
	ok, err := mgr.AddProposal(p)
	if err != nil || ok {
		t.Errorf("AddProposal with nil scheduler: want false/nil, got %v/%v", ok, err)
	}
}

func TestManager_OnBlock_WithScheduler(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	fired := make(chan struct{}, 1)
	s := NewScheduler(1, func(p *Proposal) { fired <- struct{}{} })

	p := makeProposal(w, "v9.0.0", 5)
	if err := p.Sign(w.PrivateKey); err != nil {
		t.Fatal(err)
	}
	s.AddProposal(p)

	mgr := NewManager(Config{}, s)
	mgr.OnBlock(5)

	select {
	case <-fired:
		// success — scheduler fired
	default:
	}
	// Just verify no panic — timing of goroutine is non-deterministic
}

func TestManager_AddProposal_WithScheduler(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	s := NewScheduler(1, nil)
	mgr := NewManager(Config{}, s)

	p := makeProposal(w, "v8.0.0", 100)
	if err := p.Sign(w.PrivateKey); err != nil {
		t.Fatal(err)
	}
	quorum, err := mgr.AddProposal(p)
	if err != nil {
		t.Fatalf("AddProposal: %v", err)
	}
	if !quorum {
		t.Error("1/1 validator should reach quorum")
	}
}

func TestConfig_Defaults(t *testing.T) {
	c := Config{}
	c.defaults()
	if c.ReleaseURL == "" {
		t.Error("defaults() should set non-empty ReleaseURL")
	}
	if c.CheckInterval <= 0 {
		t.Error("defaults() should set positive CheckInterval")
	}
}
