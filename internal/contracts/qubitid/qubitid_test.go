package qubitid

import (
	"testing"
)

func TestRegisterDid(t *testing.T) {
	// Test invalid payload length
	originalPtrToBytes := ptrToBytes
	defer func() { ptrToBytes = originalPtrToBytes }()
	
	ptrToBytes = func(ptr, len uint32) []byte {
		return make([]byte, len)
	}

	res := register_did(0, 10)
	if res != -1 {
		t.Fatalf("expected -1, got %d", res)
	}

	// Test valid payload length
	written := false
	originalWriteState := writeState
	defer func() { writeState = originalWriteState }()
	writeState = func(key, val []byte) {
		written = true
	}

	res = register_did(0, 32)
	if res != 0 {
		t.Fatalf("expected 0, got %d", res)
	}
	if !written {
		t.Fatal("expected writeState to be called")
	}
}

func TestVerifyKycStatus(t *testing.T) {
	originalReadState := readState
	defer func() { readState = originalReadState }()

	// Test read success
	readState = func(key []byte, maxLen uint32) ([]byte, error) {
		return make([]byte, 32), nil
	}
	res := verify_kyc_status()
	if res != 1 {
		t.Fatalf("expected 1, got %d", res)
	}

	// Test read error
	readState = func(key []byte, maxLen uint32) ([]byte, error) {
		return nil, dummyError{}
	}
	res = verify_kyc_status()
	if res != 0 {
		t.Fatalf("expected 0, got %d", res)
	}
}

func TestDummyError(t *testing.T) {
	e := dummyError{}
	if e.Error() != "error" {
		t.Fatal("expected 'error'")
	}
}

func TestDefaultMocks(t *testing.T) {
	// Test the default mock implementations that exist in the file
	b := ptrToBytes(0, 5)
	if len(b) != 5 {
		t.Fatal("expected length 5")
	}
	
	writeState(nil, nil) // just ensure it doesn't panic
	
	r, err := readState(nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 5 {
		t.Fatal("expected length 5")
	}

	qbcStateRead(0, 0, 0, 0)
	qbcStateWrite(0, 0, 0, 0)
}
