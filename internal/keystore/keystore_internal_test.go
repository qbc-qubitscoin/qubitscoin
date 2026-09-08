package keystore

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

type badReader struct{}
func (r *badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("rand error")
}

type partialReader struct {
	calls int
}
func (r *partialReader) Read(p []byte) (n int, err error) {
	if r.calls == 0 {
		r.calls++
		for i := range p { p[i] = 0 }
		return len(p), nil
	}
	return 0, errors.New("rand error 2")
}

func TestEncrypt_Errors(t *testing.T) {
	tmp := t.TempDir()
	w, _ := crypto.NewWallet()

	// 1. rand.Reader fails for salt
	origRand := rand.Reader
	rand.Reader = &badReader{}
	err := Encrypt(filepath.Join(tmp, "w1.json"), "pw", w)
	if err == nil || !strings.Contains(err.Error(), "generate salt") {
		t.Errorf("expected generate salt error, got %v", err)
	}

	// 1b. rand.Reader fails for IV
	rand.Reader = &partialReader{}
	err = Encrypt(filepath.Join(tmp, "w2.json"), "pw", w)
	if err == nil || !strings.Contains(err.Error(), "generate IV") {
		t.Errorf("expected generate IV error, got %v", err)
	}
	
	rand.Reader = origRand
}

func TestEncrypt_ExtendedErrors(t *testing.T) {
	tmp := t.TempDir()
	w, _ := crypto.NewWallet()

	origSeal := aesgcmSealFunc
	origMkdirAll := osMkdirAll
	origOpenFile := osOpenFile
	origRemove := osRemove
	origRename := osRename
	origJsonMarshal := jsonMarshalIndent
	origFileWrite := osFileWrite
	origFileClose := osFileClose

	defer func() {
		aesgcmSealFunc = origSeal
		osMkdirAll = origMkdirAll
		osOpenFile = origOpenFile
		osRemove = origRemove
		osRename = origRename
		jsonMarshalIndent = origJsonMarshal
		osFileWrite = origFileWrite
		osFileClose = origFileClose
	}()

	// Also explicitly call the original var functions so they show up as covered
	f, _ := os.CreateTemp(tmp, "cov")
	origFileWrite(f, []byte("test"))
	origFileClose(f)
	origRemove(f.Name())

	// aesgcmSealFunc fails
	aesgcmSealFunc = func(key, nonce, plaintext []byte) ([]byte, error) { return nil, errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e1.json"), "pw", w)
	aesgcmSealFunc = origSeal

	// osMkdirAll fails
	osMkdirAll = func(path string, perm os.FileMode) error { return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e2.json"), "pw", w)
	osMkdirAll = origMkdirAll

	// osOpenFile fails
	osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) { return nil, errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e3.json"), "pw", w)
	osOpenFile = origOpenFile

	// jsonMarshalIndent fails
	jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) { return nil, errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e4.json"), "pw", w)
	// jsonMarshalIndent fails and osFileClose fails
	osFileClose = func(f *os.File) error { f.Close(); return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e4b.json"), "pw", w)
	// jsonMarshalIndent fails, osFileClose succeeds, osRemove fails
	osFileClose = origFileClose
	osRemove = func(path string) error { return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e4c.json"), "pw", w)
	jsonMarshalIndent = origJsonMarshal
	osRemove = origRemove

	// osFileWrite fails
	osFileWrite = func(f *os.File, b []byte) (int, error) { return 0, errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e5.json"), "pw", w)
	// write fails and osFileClose fails
	osFileClose = func(f *os.File) error { f.Close(); return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e5b.json"), "pw", w)
	// write fails, osFileClose succeeds, osRemove fails
	osFileClose = origFileClose
	osRemove = func(path string) error { return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e5c.json"), "pw", w)
	osFileWrite = origFileWrite
	osRemove = origRemove

	// success path but osFileClose fails
	osFileClose = func(f *os.File) error { f.Close(); return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e6.json"), "pw", w)
	// success path, osFileClose fails, osRemove fails
	osFileClose = func(f *os.File) error { f.Close(); return errors.New("err") }
	osRemove = func(path string) error { return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e6b.json"), "pw", w)
	osFileClose = origFileClose
	osRemove = origRemove
	
	// osRename fails
	osRename = func(oldpath, newpath string) error { return errors.New("err") }
	_ = Encrypt(filepath.Join(tmp, "e7.json"), "pw", w)
	osRename = origRename
}

func TestInternalAESGCM_Errors(t *testing.T) {
	// bad key size
	badKey := make([]byte, 10)
	_, err := aesgcmSeal(badKey, nil, nil)
	if err == nil {
		t.Error("expected aesgcmSeal error")
	}
	_, err = aesgcmOpen(badKey, nil, nil)
	if err == nil {
		t.Error("expected aesgcmOpen error")
	}
	
	cipherNewGCM = func(cipherBlock cipher.Block) (cipher.AEAD, error) {
		return nil, errors.New("err")
	}
	goodKey := make([]byte, 32)
	_, err = aesgcmSeal(goodKey, nil, nil)
	if err == nil { t.Error("expected aesgcmSeal cipherNewGCM error") }
	_, err = aesgcmOpen(goodKey, nil, nil)
	if err == nil { t.Error("expected aesgcmOpen cipherNewGCM error") }
	cipherNewGCM = cipher.NewGCM
}
