package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultReleaseURL is the GitHub Releases API endpoint.
	// Override via Config.ReleaseURL.
	DefaultReleaseURL = "https://api.github.com/repos/qubitscoin/node/releases/latest"

	httpTimeout = 30 * time.Second
)

// Release describes a published node binary.
type Release struct {
	Version     Version
	TagName     string
	DownloadURL string // binary download URL
	Checksum    string // expected SHA-3-256 hex of the binary
	ReleaseURL  string // HTML release page
}

// githubRelease mirrors the GitHub Releases API JSON schema.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"` // release notes; contains checksum lines
}

// FetchLatestRelease queries the GitHub Releases API and returns the latest release.
// It picks the asset matching the current OS/arch (e.g. "node-linux-amd64",
// "node-windows-amd64.exe").  A companion "<name>.sha3sum" asset supplies the
// expected SHA-3-256 checksum.
func FetchLatestRelease(ctx context.Context, apiURL string) (*Release, error) {
	if apiURL == "" {
		apiURL = DefaultReleaseURL
	}

	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "QubitsCoin-node/"+Current().String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var gr githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}

	ver, err := ParseVersion(gr.TagName)
	if err != nil {
		return nil, fmt.Errorf("parse release tag %q: %w", gr.TagName, err)
	}

	// Select the asset that matches our OS + arch.
	wantName := binaryAssetName()
	checksumName := wantName + ".sha3sum"

	var downloadURL, checksum string
	for _, a := range gr.Assets {
		if a.Name == wantName {
			downloadURL = a.BrowserDownloadURL
		}
		if a.Name == checksumName {
			// Fetch the checksum asset inline (it's tiny).
			cs, err := fetchText(ctx, client, a.BrowserDownloadURL)
			if err == nil {
				checksum = strings.TrimSpace(cs)
			}
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("no asset %q found in release %s", wantName, gr.TagName)
	}

	return &Release{
		Version:     ver,
		TagName:     gr.TagName,
		DownloadURL: downloadURL,
		Checksum:    checksum,
		ReleaseURL:  gr.HTMLURL,
	}, nil
}

// binaryAssetName returns the expected asset filename for this OS/arch.
// e.g. "node-linux-amd64", "node-windows-amd64.exe", "node-darwin-arm64"
func binaryAssetName() string {
	name := fmt.Sprintf("node-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// fetchText GETs a URL and returns the body as a string.
func fetchText(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
