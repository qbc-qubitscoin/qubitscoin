package crypto

import (
	"strings"
	"testing"
)

func TestHash256_Length(t *testing.T) {
	h := Hash256([]byte("qubitscoin"))
	if len(h) != HashSize {
		t.Fatalf("expected %d bytes, got %d", HashSize, len(h))
	}
}

func TestHash256_Deterministic(t *testing.T) {
	a := Hash256([]byte("hello"))
	b := Hash256([]byte("hello"))
	if a != b {
		t.Fatal("Hash256 is not deterministic")
	}
}

func TestHash256_DifferentInputs(t *testing.T) {
	a := Hash256([]byte("hello"))
	b := Hash256([]byte("world"))
	if a == b {
		t.Fatal("different inputs produced the same hash")
	}
}

func TestHash256_EmptyInput(t *testing.T) {
	h := Hash256([]byte{})
	var zero [HashSize]byte
	if h == zero {
		t.Fatal("hash of empty input should not be zero")
	}
}

func TestHashMany_ConcatenatesCorrectly(t *testing.T) {
	combined := Hash256(append([]byte("foo"), []byte("bar")...))
	many := HashMany([]byte("foo"), []byte("bar"))
	if combined != many {
		t.Fatal("HashMany should equal Hash256(a||b)")
	}
}

func TestHashMany_SinglePart(t *testing.T) {
	a := Hash256([]byte("single"))
	b := HashMany([]byte("single"))
	if a != b {
		t.Fatal("HashMany with one part should match Hash256")
	}
}

func TestZeroHash_IsAllZeros(t *testing.T) {
	var expected [HashSize]byte
	if ZeroHash != expected {
		t.Fatal("ZeroHash should be all zeros")
	}
}

func TestToHex_Length(t *testing.T) {
	var addr [AddressSize]byte
	addr[0] = 0xab
	hex := ToHex(addr)
	if len(hex) != AddressSize*2 {
		t.Fatalf("expected hex length %d, got %d", AddressSize*2, len(hex))
	}
}

func TestToHex_Content(t *testing.T) {
	var addr [AddressSize]byte
	addr[0] = 0xde
	addr[1] = 0xad
	hex := ToHex(addr)
	if !strings.HasPrefix(hex, "dead") {
		t.Fatalf("expected hex to start with 'dead', got %s", hex[:8])
	}
}

func TestToHex_LowercaseOnly(t *testing.T) {
	var addr [AddressSize]byte
	for i := range addr {
		addr[i] = 0xff
	}
	hex := ToHex(addr)
	if hex != strings.ToLower(hex) {
		t.Fatal("ToHex should return lowercase hex")
	}
}
