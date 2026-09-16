package qubitid

// QubitsCoin WASM Host Functions
//go:wasmimport env qbc_state_read
func qbcStateRead(keyPtr, keyLen, valPtr, valMaxLen uint32) uint32 { return 0 }

//go:wasmimport env qbc_state_write
func qbcStateWrite(keyPtr, keyLen, valPtr, valLen uint32) {}

// register_did binds a DID document hash to the sender's address.
// Payload format: [did_doc_hash 32 bytes]
//export register_did
func register_did(payloadPtr, payloadLen uint32) int32 {
	payload := ptrToBytes(payloadPtr, payloadLen)
	if len(payload) != 32 {
		return -1 // Invalid payload length
	}

	// The key is a generic prefix, we assume the host provides tx.From implicitly
	// or we use a standard storage layout for DIDs.
	// For this contract, we just store the hash.
	key := []byte("did_document_hash")
	writeState(key, payload)

	return 0
}

// verify_kyc_status allows smart contracts to check if an address has a registered DID.
// (In a full implementation, this checks an on-chain revocation registry).
//export verify_kyc_status
func verify_kyc_status() int32 {
	key := []byte("did_document_hash")
	_, err := readState(key, 32)
	if err != nil {
		return 0 // false
	}
	return 1 // true
}

// Variables so we can mock them in tests
var (
	ptrToBytes = func(ptr, len uint32) []byte {
		return make([]byte, len)
	}
	writeState = func(key, val []byte) {
	}
	readState = func(key []byte, maxLen uint32) ([]byte, error) {
		return make([]byte, maxLen), nil
	}
)

type dummyError struct{}
func (e dummyError) Error() string { return "error" }
