package multisig

import (
	"bytes"
	"encoding/binary"
)



// State Keys
var (
	KeyThreshold = []byte("m_threshold")
	KeyOwners    = []byte("owners_count")
	// owners are stored as "owner_0", "owner_1", etc.
)

// InitMultisig initializes the M-of-N multisig wallet.
// Payload format: [threshold uint32] [owner_count uint32] [pubkey1 1952 bytes] [pubkey2 1952 bytes] ...
func InitMultisig(payloadPtr, payloadLen uint32) int32 {
	payload := PtrToBytes(payloadPtr, payloadLen)
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
	WriteState(KeyThreshold, payload[0:4])
	WriteState(KeyOwners, payload[4:8])

	// Write each owner's public key
	offset := uint32(8)
	for i := uint32(0); i < ownerCount; i++ {
		keyName := append([]byte("owner_"), byte(i))
		pubKey := payload[offset : offset+1952]
		WriteState(keyName, pubKey)
		offset += 1952
	}

	return 0
}

// ExecuteTransfer proposes/executes a transfer.
// Payload format: [to_address 32 bytes] [amount uint64] [num_signatures uint32] [sig1 3309 bytes] [sig2 3309 bytes] ...
func ExecuteTransfer(payloadPtr, payloadLen uint32) int32 {
	payload := PtrToBytes(payloadPtr, payloadLen)
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
	threshBytes, err := ReadState(KeyThreshold, 4)
	if err != nil {
		return -3
	}
	threshold := binary.BigEndian.Uint32(threshBytes)

	if numSigs < threshold {
		return -4 // Not enough signatures
	}

	// Read owner count
	ownerCountBytes, err := ReadState(KeyOwners, 4)
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
			pubKey, err := ReadState(keyName, 1952)
			if err == nil {
				if VerifySig(pubKey, msgToSign, sig) {
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
	if success := DoTransfer(toAddr, amount); !success {
		return -7
	}

	return 0
}

// --- Helper Functions ---

var PtrToBytes = func(ptr, len uint32) []byte {
	return make([]byte, len) 
}

var WriteState = func(key, val []byte) {
}

var ReadState = func(key []byte, maxLen uint32) ([]byte, error) {
	val := make([]byte, maxLen)
	return val, nil
}

var VerifySig = func(pubKey, msg, sig []byte) bool {
	return bytes.Equal(pubKey, sig[:len(pubKey)]) // mock logic
}

var DoTransfer = func(to []byte, amount uint64) bool {
	if amount == 0 { return false }
	return true
}

