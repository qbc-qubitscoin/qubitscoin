// Package keystore provides encrypted on-disk storage for ML-DSA-65 wallets.
//
// File format (JSON):
//
//	{
//	  "version": 1,
//	  "address": "<hex>",          // SHA-3-256(pubKey)
//	  "kdf": "argon2id",
//	  "kdf_params": { "memory": 65536, "time": 3, "threads": 4, "salt": "<base64>" },
//	  "cipher": "aes256gcm",
//	  "cipher_params": { "iv": "<base64>", "ciphertext": "<base64>" },
//	  "public_key": "<base64>"     // ML-DSA-65 public key, unencrypted
//	}
//
// All cryptography is quantum-resistant or symmetric:
//   - KDF: Argon2id (memory-hard, side-channel resistant)
//   - Encryption: AES-256-GCM (NIST approved symmetric AEAD)
//   - No RSA, no ECDSA, no secp256k1 anywhere.
package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"golang.org/x/crypto/argon2"
)

const fileVersion = 1

// argon2idParams defines the Argon2id work factors.
// These are intentionally high to resist offline brute-force attacks.
type argon2idParams struct {
	Memory  uint32 `json:"memory"`  // KiB
	Time    uint32 `json:"time"`    // iterations
	Threads uint8  `json:"threads"` // parallelism
	Salt    string `json:"salt"`    // base64-encoded 16-byte random salt
}

// cipherParams holds the AES-256-GCM parameters.
type cipherParams struct {
	IV         string `json:"iv"`         // base64-encoded 12-byte nonce
	Ciphertext string `json:"ciphertext"` // base64-encoded (encrypted privKey + GCM tag)
}

// keystoreFile is the on-disk JSON layout.
type keystoreFile struct {
	Version      int            `json:"version"`
	Address      string         `json:"address"` // hex — SHA-3-256(pubKey)
	KDF          string         `json:"kdf"`     // always "argon2id"
	KDFParams    argon2idParams `json:"kdf_params"`
	Cipher       string         `json:"cipher"` // always "aes256gcm"
	CipherParams cipherParams   `json:"cipher_params"`
	PublicKey    string         `json:"public_key"` // base64 ML-DSA-65 public key
}

// ─────────────────────────────────────────────────────────────────────────────
// Encrypt / Decrypt
// ─────────────────────────────────────────────────────────────────────────────

// Encrypt creates an encrypted keystore file at the path using password.
// If the file already exists, it is overwritten.
func Encrypt(path, password string, w *crypto.Wallet) error {
	// Generate random Argon2id salt.
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("keystore: generate salt: %w", err)
	}

	kdfP := argon2idParams{
		Memory:  64 * 1024, // 64 MiB
		Time:    3,
		Threads: 4,
		Salt:    base64.StdEncoding.EncodeToString(salt),
	}

	// Derive a 32-byte AES key from the password.
	key := deriveKey(password, salt, kdfP)

	// Generate random 12-byte AES-GCM nonce.
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return fmt.Errorf("keystore: generate IV: %w", err)
	}

	// Encrypt the private key.
	ciphertext, err := aesgcmSeal(key, iv, w.PrivateKey)
	if err != nil {
		return fmt.Errorf("keystore: encrypt: %w", err)
	}

	kf := keystoreFile{
		Version:   fileVersion,
		Address:   crypto.AddressToHex(w.Address),
		KDF:       "argon2id",
		KDFParams: kdfP,
		Cipher:    "aes256gcm",
		CipherParams: cipherParams{
			IV:         base64.StdEncoding.EncodeToString(iv),
			Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		},
		PublicKey: base64.StdEncoding.EncodeToString(w.PublicKey),
	}

	// Write atomically via temp file.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("keystore: create a temp file: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(kf); err != nil {
		err := f.Close()
		if err != nil {
			return err
		}
		err = os.Remove(tmp)
		if err != nil {
			return err
		}
		return fmt.Errorf("keystore: marshal: %w", err)
	}
	if err := f.Close(); err != nil {
		err := os.Remove(tmp)
		if err != nil {
			return err
		}
		return err
	}
	return os.Rename(tmp, path)
}

// Decrypt loads an encrypted keystore file and decrypts it with a password.
func Decrypt(path, password string) (*crypto.Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("keystore: read %s: %w", path, err)
	}
	var kf keystoreFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("keystore: parse: %w", err)
	}
	if kf.Version != fileVersion {
		return nil, fmt.Errorf("keystore: unsupported version %d", kf.Version)
	}
	if kf.KDF != "argon2id" {
		return nil, fmt.Errorf("keystore: unsupported KDF %q", kf.KDF)
	}
	if kf.Cipher != "aes256gcm" {
		return nil, fmt.Errorf("keystore: unsupported cipher %q", kf.Cipher)
	}

	salt, err := base64.StdEncoding.DecodeString(kf.KDFParams.Salt)
	if err != nil {
		return nil, fmt.Errorf("keystore: decode salt: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(kf.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("keystore: decode IV: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(kf.CipherParams.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("keystore: decode ciphertext: %w", err)
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(kf.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("keystore: decode public key: %w", err)
	}

	key := deriveKey(password, salt, kf.KDFParams)
	privKey, err := aesgcmOpen(key, iv, ciphertext)
	if err != nil {
		return nil, errors.New("keystore: decryption failed — wrong password")
	}

	addr, err := crypto.HexToAddress(kf.Address)
	if err != nil {
		return nil, fmt.Errorf("keystore: invalid stored address: %w", err)
	}

	// Verify the derived address matches the stored one.
	derived := crypto.DeriveAddress(pubKeyBytes)
	if derived != addr {
		return nil, errors.New("keystore: address mismatch — a file may be corrupt")
	}

	return &crypto.Wallet{
		PublicKey:  pubKeyBytes,
		PrivateKey: privKey,
		Address:    addr,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// List returns the address stored in a keystore file without decrypting it.
// ─────────────────────────────────────────────────────────────────────────────

// PeekAddress returns the address field from a keystore file (no password needed).
func PeekAddress(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var kf struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(data, &kf); err != nil {
		return "", err
	}
	return kf.Address, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func deriveKey(password string, salt []byte, p argon2idParams) []byte {
	return argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, 32)
}

func aesgcmSeal(key, nonce, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

func aesgcmOpen(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
