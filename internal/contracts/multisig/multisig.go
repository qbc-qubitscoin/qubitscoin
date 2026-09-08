package main

import (
	"bytes"
	"encoding/binary"
)

// QubitsCoin WASM Host Functions
// In a real environment, these are provided by the QubitVM runtime (wazero).
// We stub them here to allow the contract to compile.
//go:wasmimport env qbc_state_read
func qbcStateRead(keyPtr, keyLen, valPtr, valMaxLen uint32) uint32 { return 0 }

//go:wasmimport env qbc_state_write
func qbcStateWrite(keyPtr, keyLen, valPtr, valLen uint32) {}

//go:wasmimport env qbc_transfer
func qbcTransfer(toPtr, toLen uint32, amount uint64) uint32 { return 0 }

//go:wasmimport env qbc_crypto_verify
func qbcCryptoVerify(pubPtr, pubLen, msgPtr, msgLen, sigPtr, sigLen uint32) uint32 { return 0 }

// State Keys
var (
	KeyThreshold = []byte("m_threshold")
	KeyOwners    = []byte("owners_count")
	// owners are stored as "owner_0", "owner_1", etc.
)

// init_multisig initializes the M-of-N multisig wallet.
// Payload format: [threshold uint32] [owner_count uint32] [pubkey1 1952 bytes] [pubkey2 1952 bytes] ...
//export init_multisig
func init_multisig(payloadPtr, payloadLen uint32) int32 {
	payload := ptrToBytes(payloadPtr, payloadLen)
	if len(payload) < 8 {
		return -1 // Invalid payload
	}

	// threshold := binary.BigEndian.Uint32(payload[0:4])
	ownerCount := binary.BigEndian.Uint32(payload[4:8])

	expectedLen := 8 + (ownerCount * 1952) // 1952 is ML-DSA-65 public key size
	if uint32(len(payload)) != expectedLen {
		return -2 // Payload length mismatch
	}

	// Write threshold and owner count to state
	writeState(KeyThreshold, payload[0:4])
	writeState(KeyOwners, payload[4:8])

	// Write each owner's public key
	offset := uint32(8)
	for i := uint32(0); i < ownerCount; i++ {
		keyName := append([]byte("owner_"), byte(i))
		pubKey := payload[offset : offset+1952]
		writeState(keyName, pubKey)
		offset += 1952
	}

	return 0
}

// execute_transfer proposes/executes a transfer.
// Payload format: [to_address 32 bytes] [amount uint64] [num_signatures uint32] [sig1 3309 bytes] [sig2 3309 bytes] ...
//export execute_transfer
func execute_transfer(payloadPtr, payloadLen uint32) int32 {
	payload := ptrToBytes(payloadPtr, payloadLen)
	if len(payload) < 44 {
		return -1
	}

	toAddr := payload[0:32]
	amount := binary.BigEndian.Uint64(payload[32:40])
	numSigs := binary.BigEndian.Uint32(payload[40:44])

	expectedLen := 44 + (numSigs * 3309) // 3309 is ML-DSA-65 signature size
	if uint32(len(payload)) != expectedLen {
		return -2
	}

	// Read threshold
	threshBytes, err := readState(KeyThreshold, 4)
	if err != nil {
		return -3
	}
	threshold := binary.BigEndian.Uint32(threshBytes)

	if numSigs < threshold {
		return -4 // Not enough signatures
	}

	// Read owner count
	ownerCountBytes, err := readState(KeyOwners, 4)
	if err != nil {
		return -5
	}
	ownerCount := binary.BigEndian.Uint32(ownerCountBytes)

	// Construct the message that should have been signed
	// msg = [to_address] + [amount]
	msgToSign := append(toAddr, payload[32:40]...)

	// Verify signatures against owner public keys
	validSigs := uint32(0)
	sigOffset := uint32(44)
	
	for s := uint32(0); s < numSigs; s++ {
		sig := payload[sigOffset : sigOffset+3309]
		
		for o := uint32(0); o < ownerCount; o++ {
			keyName := append([]byte("owner_"), byte(o))
			pubKey, err := readState(keyName, 1952)
			if err == nil {
				if verifySig(pubKey, msgToSign, sig) {
					validSigs++
					break // Sig matches this owner
				}
			}
		}
		sigOffset += 3309
	}

	if validSigs < threshold {
		return -6 // Signature verification failed
	}

	// Signatures valid, execute transfer
	if success := doTransfer(toAddr, amount); !success {
		return -7
	}

	return 0
}

// --- Helper Functions ---

func ptrToBytes(ptr, len uint32) []byte {
	return make([]byte, len) 
}

func writeState(key, val []byte) {
}

type dummyError struct{}
func (e dummyError) Error() string { return "error" }

func readState(key []byte, maxLen uint32) ([]byte, error) {
	val := make([]byte, maxLen)
	return val, nil
}

func verifySig(pubKey, msg, sig []byte) bool {
	return bytes.Equal(pubKey, sig[:len(pubKey)]) // mock logic
}

func doTransfer(to []byte, amount uint64) bool {
	if amount == 0 { return false }
	return true
}

func main() {}
