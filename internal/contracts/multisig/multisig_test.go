package multisig

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

var (
	origPtr        = PtrToBytes
	origWrite      = WriteState
	origRead       = ReadState
	origVerify     = VerifySig
	origDoTransfer = DoTransfer
)

func resetMocks() {
	PtrToBytes = origPtr
	WriteState = origWrite
	ReadState = origRead
	VerifySig = origVerify
	DoTransfer = origDoTransfer
}

func TestInitMultisig(t *testing.T) {
	defer resetMocks()
	// ... rest of TestInitMultisig
	// Payload < 8 bytes
	if res := InitMultisig(0, 4); res != -1 {
		t.Errorf("expected -1, got %d", res)
	}

	// Payload length mismatch
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[4:8], 2) // ownerCount = 2, expected 8 + 2*1952 = 3912
	
	PtrToBytes = func(ptr, l uint32) []byte {
		return payload
	}
	if res := InitMultisig(0, 12); res != -2 {
		t.Errorf("expected -2, got %d", res)
	}

	// Valid payload
	validPayload := make([]byte, 8+2*1952)
	binary.BigEndian.PutUint32(validPayload[0:4], 2) // threshold
	binary.BigEndian.PutUint32(validPayload[4:8], 2) // ownerCount
	
	PtrToBytes = func(ptr, l uint32) []byte {
		return validPayload
	}
	
	written := make(map[string][]byte)
	WriteState = func(key, val []byte) {
		written[string(key)] = append([]byte(nil), val...)
	}

	if res := InitMultisig(0, uint32(len(validPayload))); res != 0 {
		t.Errorf("expected 0, got %d", res)
	}
	
	if len(written) != 4 {
		t.Errorf("expected 4 states written (threshold, owners, owner_0, owner_1), got %d", len(written))
	}
}

func TestExecuteTransfer(t *testing.T) {
	defer resetMocks()
	// ... rest of TestExecuteTransfer
	// payload < 44
	PtrToBytes = func(ptr, l uint32) []byte {
		return make([]byte, 40)
	}
	if res := ExecuteTransfer(0, 40); res != -1 {
		t.Errorf("expected -1, got %d", res)
	}

	// payload len mismatch
	payload := make([]byte, 44)
	binary.BigEndian.PutUint32(payload[40:44], 1) // numSigs = 1, expected 44 + 3309
	PtrToBytes = func(ptr, l uint32) []byte {
		return payload
	}
	if res := ExecuteTransfer(0, 44); res != -2 {
		t.Errorf("expected -2, got %d", res)
	}

	// threshold read fail
	validPayload := make([]byte, 44+3309)
	binary.BigEndian.PutUint32(validPayload[40:44], 1)
	PtrToBytes = func(ptr, l uint32) []byte { return validPayload }
	
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		return nil, errors.New("read err")
	}
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -3 {
		t.Errorf("expected -3, got %d", res)
	}

	// not enough signatures
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		if bytes.Equal(key, KeyThreshold) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 2)
			return b, nil
		}
		return nil, nil
	}
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -4 {
		t.Errorf("expected -4, got %d", res)
	}

	// owner count read fail
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		if bytes.Equal(key, KeyThreshold) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		if bytes.Equal(key, KeyOwners) {
			return nil, errors.New("read err")
		}
		return nil, nil
	}
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -5 {
		t.Errorf("expected -5, got %d", res)
	}

	// Invalid signature
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		if bytes.Equal(key, KeyThreshold) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		if bytes.Equal(key, KeyOwners) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		if bytes.HasPrefix(key, []byte("owner_")) {
			return make([]byte, 1952), nil
		}
		return nil, nil
	}
	VerifySig = func(pubKey, msg, sig []byte) bool { return false }
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -6 {
		t.Errorf("expected -6, got %d", res)
	}

	// Owner read fail (simulate loop reading error)
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		if bytes.Equal(key, KeyThreshold) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		if bytes.Equal(key, KeyOwners) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		return nil, errors.New("read err")
	}
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -6 {
		t.Errorf("expected -6, got %d", res)
	}
	
	// Valid signature, but transfer fails
	ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
		if bytes.Equal(key, KeyThreshold) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		if bytes.Equal(key, KeyOwners) {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, 1)
			return b, nil
		}
		return make([]byte, 1952), nil
	}
	VerifySig = func(pubKey, msg, sig []byte) bool { return true }
	DoTransfer = func(to []byte, amount uint64) bool { return false }
	
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != -7 {
		t.Errorf("expected -7, got %d", res)
	}

	// Valid signature, transfer succeeds
	DoTransfer = func(to []byte, amount uint64) bool { return true }
	if res := ExecuteTransfer(0, uint32(len(validPayload))); res != 0 {
		t.Errorf("expected 0, got %d", res)
	}
}

func TestMockFunctions(t *testing.T) {
	resetMocks()
	origPtr(0, 10)
	origWrite(nil, nil)
	origRead(nil, 10)
	origVerify(make([]byte, 10), nil, make([]byte, 10))
	origDoTransfer(nil, 1)
	origDoTransfer(nil, 0)

	// Also cover the WASM stubs
	qbcStateRead(0, 0, 0, 0)
	qbcStateWrite(0, 0, 0, 0)
	qbcTransfer(0, 0, 0)
	qbcCryptoVerify(0, 0, 0, 0, 0, 0)
}
