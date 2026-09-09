package upgrade

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestCurrent_Fallback(t *testing.T) {
	orig := version
	defer func() { version = orig }()

	version = "invalid_version_string"
	cur := Current()
	if cur.Major != 0 || cur.Minor != 5 || cur.Patch != 0 {
		t.Errorf("expected fallback 0.5.0, got %v", cur)
	}
}

func TestProposal_SignAndVerify_EdgeCases(t *testing.T) {
	p := &Proposal{
		TargetHeight: 100,
	}

	// Sign with short key
	err := p.Sign([]byte("short"))
	if err == nil {
		t.Fatal("expected error signing with short key")
	}

	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	p.Proposer = w.Address
	p.PublicKey = w.PublicKey

	// Sign with valid key
	if err := p.Sign(w.PrivateKey); err != nil {
		t.Fatal(err)
	}

	// Verify valid
	ok, err := p.Verify()
	if err != nil || !ok {
		t.Fatalf("expected verify true, got %v, %v", ok, err)
	}

	// Verify short public key
	p.PublicKey = []byte("short")
	ok, _ = p.Verify()
	if ok {
		t.Fatal("expected false for short public key")
	}

	// Verify proposer address mismatch
	p.PublicKey = w.PublicKey
	var fakeAddr [crypto.AddressSize]byte
	fakeAddr[0] = 99
	p.Proposer = fakeAddr
	ok, _ = p.Verify()
	if ok {
		t.Fatal("expected false for mismatched proposer address")
	}
}

func TestDownloader_ComputeFileHash(t *testing.T) {
	tmp, err := os.CreateTemp("", "hash-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := []byte("hello-quantum-upgrade")
	_, _ = tmp.Write(content)
	_ = tmp.Close()

	hash, err := ComputeFileHash(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := sha3.Sum256(content)
	if hash != hex.EncodeToString(expected[:]) {
		t.Fatalf("expected %s got %s", hex.EncodeToString(expected[:]), hash)
	}

	// Non-existent file
	_, err = ComputeFileHash("non_existent_file_path_12345.bin")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}

	// Directory path (fails on read)
	_, _ = ComputeFileHash(os.TempDir())
}

func TestDownloader_Download_Scenarios(t *testing.T) {
	binaryContent := []byte("mock-node-binary-content-12345")
	h := sha3.Sum256(binaryContent)
	checksumHex := hex.EncodeToString(h[:])

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(binaryContent)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	ctx := context.Background()

	// 1. Success with checksum
	rel := &Release{
		DownloadURL: ts.URL + "/binary",
		Checksum:    checksumHex,
	}
	path, err := Download(ctx, rel)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer os.Remove(path)

	// 2. Success with empty checksum
	relEmpty := &Release{
		DownloadURL: ts.URL + "/binary",
	}
	path2, err := Download(ctx, relEmpty)
	if err != nil {
		t.Fatalf("Download without checksum failed: %v", err)
	}
	defer os.Remove(path2)

	// 3. Checksum mismatch
	relMismatch := &Release{
		DownloadURL: ts.URL + "/binary",
		Checksum:    "bad_checksum_deadbeef",
	}
	_, err = Download(ctx, relMismatch)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}

	// 4. HTTP error
	relErr := &Release{
		DownloadURL: ts.URL + "/error",
	}
	_, err = Download(ctx, relErr)
	if err == nil {
		t.Fatal("expected HTTP error")
	}

	// 5. Bad request URL
	relBad := &Release{
		DownloadURL: "http://invalid-url-that-does-not-exist:12345/fail",
	}
	_, err = Download(ctx, relBad)
	if err == nil {
		t.Fatal("expected network error")
	}

	// 6. osExecutable failure
	origExe := osExecutable
	defer func() { osExecutable = origExe }()
	osExecutable = func() (string, error) {
		return "", errors.New("cannot resolve executable")
	}
	_, err = Download(ctx, rel)
	if err == nil {
		t.Fatal("expected error on osExecutable failure in Download")
	}
	osExecutable = origExe

	// 7. createTemp failure
	origTemp := createTemp
	defer func() { createTemp = origTemp }()
	createTemp = func(dir, pattern string) (*os.File, error) {
		return nil, errors.New("cannot create temp file")
	}
	_, err = Download(ctx, rel)
	if err == nil {
		t.Fatal("expected error on createTemp failure in Download")
	}
	createTemp = origTemp

	// 8. chmod failure
	origChmod := chmod
	defer func() { chmod = origChmod }()
	chmod = func(name string, mode os.FileMode) error {
		return errors.New("chmod error")
	}
	_, err = Download(ctx, rel)
	if err == nil {
		t.Fatal("expected error on chmod failure in Download")
	}
	chmod = origChmod

	// 9. bad request URL
	_, err = Download(ctx, &Release{DownloadURL: "://invalid-url"})
	if err == nil {
		t.Fatal("expected error on bad URL in Download")
	}

	// 10. closeFile failure
	origClose := closeFile
	defer func() { closeFile = origClose }()
	closeFile = func(f *os.File) error {
		_ = f.Close()
		return errors.New("flush failed")
	}
	_, err = Download(ctx, rel)
	if err == nil {
		t.Fatal("expected error on closeFile failure in Download")
	}
	closeFile = origClose

	// 11. io.Copy failure via truncated response
	tsAbort := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer tsAbort.Close()

	_, err = Download(ctx, &Release{DownloadURL: tsAbort.URL})
	if err == nil {
		t.Fatal("expected io.Copy error on aborted download")
	}
}

func TestRelease_FetchLatestRelease_Scenarios(t *testing.T) {
	assetName := binaryAssetName()
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			gr := map[string]any{
				"tag_name": "v1.2.3",
				"html_url": "https://github.com/qbc-qubitscoin/qubitscoin/releases/v1.2.3",
				"assets": []map[string]any{
					{
						"name":                 assetName,
						"browser_download_url": ts.URL + "/binary",
					},
					{
						"name":                 assetName + ".sha3sum",
						"browser_download_url": ts.URL + "/sha",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(gr)
		case "/sha":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("deadbeef1234\n"))
		case "/no-assets":
			gr := map[string]any{
				"tag_name": "v2.0.0",
				"assets":   []map[string]any{},
			}
			_ = json.NewEncoder(w).Encode(gr)
		case "/bad-tag":
			gr := map[string]any{
				"tag_name": "invalid_tag",
				"assets":   []map[string]any{},
			}
			_ = json.NewEncoder(w).Encode(gr)
		case "/bad-json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	ctx := context.Background()

	// 1. Success
	rel, err := FetchLatestRelease(ctx, ts.URL+"/latest")
	if err != nil {
		t.Fatalf("FetchLatestRelease failed: %v", err)
	}
	if rel.TagName != "v1.2.3" {
		t.Fatalf("expected v1.2.3, got %s", rel.TagName)
	}

	// 2. Missing asset
	_, err = FetchLatestRelease(ctx, ts.URL+"/no-assets")
	if err == nil {
		t.Fatal("expected error for missing asset")
	}

	// 3. Bad tag
	_, err = FetchLatestRelease(ctx, ts.URL+"/bad-tag")
	if err == nil {
		t.Fatal("expected error for bad tag")
	}

	// 4. Bad JSON
	_, err = FetchLatestRelease(ctx, ts.URL+"/bad-json")
	if err == nil {
		t.Fatal("expected error for bad json")
	}

	// 5. Status 500
	_, err = FetchLatestRelease(ctx, ts.URL+"/500")
	if err == nil {
		t.Fatal("expected error for status 500")
	}

	// 6. Network error
	_, err = FetchLatestRelease(ctx, "http://127.0.0.1:0/fail")
	if err == nil {
		t.Fatal("expected network error")
	}

	// 7. Empty URL branch (calls DefaultReleaseURL)
	_, _ = FetchLatestRelease(ctx, "")

	// 8. fetchText helper directly
	client := &http.Client{}
	_, err = fetchText(ctx, client, "http://127.0.0.1:0/fail")
	if err == nil {
		t.Fatal("expected error from fetchText")
	}

	// 9. fetchText bad request
	_, _ = fetchText(ctx, client, "://invalid-url")

	// 10. FetchLatestRelease bad URL
	_, err = FetchLatestRelease(ctx, "://invalid-url")
	if err == nil {
		t.Fatal("expected error on invalid URL in FetchLatestRelease")
	}

	// 11. fetchText read failure via truncated response
	tsAbort := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer tsAbort.Close()
	_, _ = fetchText(ctx, client, tsAbort.URL)
}

func TestApplier_ApplyAndClean_Hooks(t *testing.T) {
	origExe := osExecutable
	origSym := evalSymlinks
	origReExec := reExecFunc
	origExit := osExit
	defer func() {
		osExecutable = origExe
		evalSymlinks = origSym
		reExecFunc = origReExec
		osExit = origExit
	}()

	// 1. cleanOldBinary
	tmpDir, err := os.MkdirTemp("", "clean-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fakeExe := filepath.Join(tmpDir, "node.exe")
	_ = os.WriteFile(fakeExe, []byte("exe"), 0o755)
	fakeOld := fakeExe + ".old"
	_ = os.WriteFile(fakeOld, []byte("old"), 0o755)

	osExecutable = func() (string, error) {
		return fakeExe, nil
	}
	cleanOldBinary()
	if _, err := os.Stat(fakeOld); err == nil {
		t.Fatal("cleanOldBinary should have removed fakeOld")
	}

	// cleanOldBinary when osExecutable fails
	osExecutable = func() (string, error) {
		return "", errors.New("cannot resolve exe")
	}
	cleanOldBinary()

	// 2. Apply: osExecutable error
	err = Apply("new_binary")
	if err == nil {
		t.Fatal("expected error on osExecutable fail")
	}

	// 3. Apply: evalSymlinks error
	osExecutable = func() (string, error) {
		return fakeExe, nil
	}
	evalSymlinks = func(path string) (string, error) {
		return "", errors.New("symlink failure")
	}
	err = Apply("new_binary")
	if err == nil {
		t.Fatal("expected error on evalSymlinks fail")
	}

	// 4. Apply: rename step 1 error (fakeExe does not exist)
	evalSymlinks = func(path string) (string, error) {
		return filepath.Join(tmpDir, "missing_exe"), nil
	}
	err = Apply("new_binary")
	if err == nil {
		t.Fatal("expected error on rename missing exe")
	}

	// 5. Apply: rename step 2 error (newBinary does not exist, rollback original)
	evalSymlinks = func(path string) (string, error) {
		return fakeExe, nil
	}
	err = Apply("missing_new_binary")
	if err == nil {
		t.Fatal("expected error on rename missing new binary")
	}

	// 6. Apply: reExec failure and rollback
	fakeNew := filepath.Join(tmpDir, "new_node.exe")
	_ = os.WriteFile(fakeNew, []byte("new"), 0o755)

	reExecFunc = func(binPath string, args []string) error {
		return errors.New("re-exec simulated failure")
	}
	err = Apply(fakeNew)
	if err == nil {
		t.Fatal("expected error on reExec failure")
	}

	// 7. Apply: Success path (calls osExit(0))
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
	}
	reExecFunc = func(binPath string, args []string) error {
		return nil
	}
	// Re-create fakeExe and fakeNew
	_ = os.WriteFile(fakeExe, []byte("exe"), 0o755)
	_ = os.WriteFile(fakeNew, []byte("new"), 0o755)
	err = Apply(fakeNew)
	if err != nil || !exitCalled {
		t.Fatalf("expected success on Apply, got err: %v, exitCalled: %v", err, exitCalled)
	}

	// 8. Test reExec directly
	err = reExec("non_existent_binary_xyz_12345", []string{"cmd"})
	if err == nil {
		t.Fatal("expected error from reExec with non-existent binary")
	}

	// 9. Test reExec success with current executable
	exe, err := os.Executable()
	if err == nil {
		_ = reExec(exe, []string{exe, "-test.run=^$"})
	}
}
