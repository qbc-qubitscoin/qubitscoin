package config

import (
	"testing"
	"time"
)

func TestDuration_UnmarshalText_Error(t *testing.T) {
	var d duration
	if err := d.UnmarshalText([]byte("invalid-duration")); err == nil {
		t.Fatal("expected error for invalid duration text, got nil")
	}
}

func TestDuration_MarshalText(t *testing.T) {
	d := duration{Duration: 15 * time.Minute}
	b, err := d.MarshalText()
	if err != nil {
		t.Fatalf("unexpected error marshaling duration: %v", err)
	}
	if string(b) != "15m0s" {
		t.Fatalf("expected '15m0s', got %q", string(b))
	}
}
