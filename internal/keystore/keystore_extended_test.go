package keystore_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/keystore"
)

func TestDecrypt_Extended_Errors(t *testing.T) {
	tmp := t.TempDir()

	// 1. file not found
	_, err := keystore.Decrypt(filepath.Join(tmp, "nonexistent.json"), "pw")
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Errorf("expected read error, got %v", err)
	}

	// 2. bad JSON
	badJSONPath := filepath.Join(tmp, "bad.json")
	os.WriteFile(badJSONPath, []byte("invalid json"), 0600)
	_, err = keystore.Decrypt(badJSONPath, "pw")
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got %v", err)
	}

	validWallet, _ := crypto.NewWallet()
	validPath := filepath.Join(tmp, "valid.json")
	keystore.Encrypt(validPath, "pw", validWallet)
	validData, _ := os.ReadFile(validPath)

	tests := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{
			name: "wrong version",
			mutate: func(m map[string]any) {
				m["version"] = 2
			},
			wantErr: "unsupported version",
		},
		{
			name: "wrong kdf",
			mutate: func(m map[string]any) {
				m["kdf"] = "scrypt"
			},
			wantErr: "unsupported KDF",
		},
		{
			name: "wrong cipher",
			mutate: func(m map[string]any) {
				m["cipher"] = "aes128cbc"
			},
			wantErr: "unsupported cipher",
		},
		{
			name: "bad base64 salt",
			mutate: func(m map[string]any) {
				kdfParams := m["kdf_params"].(map[string]any)
				kdfParams["salt"] = "---"
			},
			wantErr: "decode salt",
		},
		{
			name: "bad base64 iv",
			mutate: func(m map[string]any) {
				cipherParams := m["cipher_params"].(map[string]any)
				cipherParams["iv"] = "---"
			},
			wantErr: "decode IV",
		},
		{
			name: "bad base64 ciphertext",
			mutate: func(m map[string]any) {
				cipherParams := m["cipher_params"].(map[string]any)
				cipherParams["ciphertext"] = "---"
			},
			wantErr: "decode ciphertext",
		},
		{
			name: "bad base64 public key",
			mutate: func(m map[string]any) {
				m["public_key"] = "---"
			},
			wantErr: "decode public key",
		},
		{
			name: "corrupt address mismatch", // corrupt pubkey would cause address mismatch
			mutate: func(m map[string]any) {
				// hex has to be valid (32 bytes = 64 chars) to pass HexToAddress, but won't match pubkey
				wrongWallet, _ := crypto.NewWallet()
				m["address"] = crypto.AddressToHex(wrongWallet.Address)
			},
			wantErr: "address mismatch",
		},
		{
			name: "invalid stored address hex",
			mutate: func(m map[string]any) {
				m["address"] = "zzz" // invalid hex
			},
			wantErr: "invalid stored address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			json.Unmarshal(validData, &m)
			tc.mutate(m)
			mutatedData, _ := json.Marshal(m)
			path := filepath.Join(tmp, "mutated.json")
			os.WriteFile(path, mutatedData, 0600)

			_, err := keystore.Decrypt(path, "pw")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPeekAddress_Extended_Errors(t *testing.T) {
	tmp := t.TempDir()

	// 1. file not found
	_, err := keystore.PeekAddress(filepath.Join(tmp, "nonexistent.json"))
	if err == nil {
		t.Error("expected error for missing file")
	}

	// 2. bad json
	badJSONPath := filepath.Join(tmp, "bad.json")
	os.WriteFile(badJSONPath, []byte("invalid json"), 0600)
	_, err = keystore.PeekAddress(badJSONPath)
	if err == nil {
		t.Error("expected error for bad json")
	}

	// 3. valid file
	validWallet, _ := crypto.NewWallet()
	validPath := filepath.Join(tmp, "valid.json")
	keystore.Encrypt(validPath, "pw", validWallet)
	addr, err := keystore.PeekAddress(validPath)
	if err != nil {
		t.Errorf("expected no error for valid file, got %v", err)
	}
	if addr != crypto.AddressToHex(validWallet.Address) {
		t.Errorf("address mismatch")
	}
}

func TestDecrypt_Extended_WrongPassword(t *testing.T) {
	tmp := t.TempDir()
	validWallet, _ := crypto.NewWallet()
	validPath := filepath.Join(tmp, "valid.json")
	keystore.Encrypt(validPath, "pw", validWallet)

	_, err := keystore.Decrypt(validPath, "wrong")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}
