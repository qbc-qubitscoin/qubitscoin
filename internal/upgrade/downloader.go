package upgrade

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/sha3"
)

var (
	createTemp = os.CreateTemp
	chmod      = os.Chmod
	closeFile  = func(f *os.File) error { return f.Close() }
)

// Download fetches a release binary, streams it to a temp file in the same
// directory as the running executable, and verifies its SHA-3-256 checksum.
// Returns the path of the verified temp file.
//
// If expectedChecksum is empty, the checksum step is skipped (dev / test use).
func Download(ctx context.Context, rel *Release) (string, error) {
	// Resolve destination directory (same as the running binary so that an
	// atomic rename later stays on the same filesystem / partition).
	exePath, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	tmp, err := createTemp(exeDir, "node-upgrade-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create a temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Ensure the temp file is removed on error.
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Stream the binary.
	client := &http.Client{Timeout: 0} // streaming, no timeout
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("User-Agent", "QubitsCoin-node/"+Current().String())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	// Write + hash simultaneously.
	h := sha3.New256()
	w := io.MultiWriter(tmp, h)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return "", fmt.Errorf("write binary: %w", err)
	}
	if err := closeFile(tmp); err != nil {
		return "", fmt.Errorf("flush temp file: %w", err)
	}

	// Verify checksum if provided.
	if rel.Checksum != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != rel.Checksum {
			return "", fmt.Errorf("checksum mismatch: want %s got %s", rel.Checksum, got)
		}
	}

	// Make the temp file executable.
	if err := chmod(tmpPath, 0o755); err != nil {
		return "", fmt.Errorf("chmod: %w", err)
	}

	success = true
	return tmpPath, nil
}

// ComputeFileHash returns the SHA-3-256 hex digest of a file on disk.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha3.New256()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
