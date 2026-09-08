package upgrade

import "testing"

func TestParseVersion_WithV(t *testing.T) {
	v, err := ParseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("ParseVersion: %v", err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Errorf("want 1.2.3, got %v", v)
	}
}

func TestParseVersion_WithoutV(t *testing.T) {
	v, err := ParseVersion("0.4.0")
	if err != nil {
		t.Fatalf("ParseVersion: %v", err)
	}
	if v.Major != 0 || v.Minor != 4 || v.Patch != 0 {
		t.Errorf("want 0.4.0, got %v", v)
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	cases := []string{"", "1.2", "abc", "1.2.x", "1.2.3.4"}
	for _, c := range cases {
		if _, err := ParseVersion(c); err == nil {
			t.Errorf("ParseVersion(%q): expected error, got nil", c)
		}
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{1, 2, 3}
	if s := v.String(); s != "v1.2.3" {
		t.Errorf("String: want v1.2.3, got %s", s)
	}
}

func TestVersion_After(t *testing.T) {
	cases := []struct {
		a, b  Version
		after bool
	}{
		{Version{1, 0, 0}, Version{0, 9, 9}, true},
		{Version{0, 5, 0}, Version{0, 4, 9}, true},
		{Version{0, 4, 2}, Version{0, 4, 1}, true},
		{Version{1, 0, 0}, Version{1, 0, 0}, false},
		{Version{0, 3, 9}, Version{0, 4, 0}, false},
	}
	for _, c := range cases {
		got := c.a.After(c.b)
		if got != c.after {
			t.Errorf("%v.After(%v): want %v, got %v", c.a, c.b, c.after, got)
		}
	}
}

func TestVersion_Equal(t *testing.T) {
	va := Version{1, 2, 3}
	vb := Version{1, 2, 3}
	vc := Version{1, 2, 4}
	if !va.Equal(vb) {
		t.Error("equal versions should be Equal")
	}
	if va.Equal(vc) {
		t.Error("different versions should not be Equal")
	}
}

func TestCurrent_Parseable(t *testing.T) {
	v := Current()
	if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		t.Error("the current version has a negative component")
	}
	// Round-trip through String.
	parsed, err := ParseVersion(v.String())
	if err != nil {
		t.Fatalf("re-parse current version: %v", err)
	}
	if !parsed.Equal(v) {
		t.Errorf("round-trip mismatch: %v vs %v", v, parsed)
	}
}
