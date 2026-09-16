package upgrade

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestManager_Branches(t *testing.T) {
	mux := http.NewServeMux()
	var tagToReturn string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "` + tagToReturn + `", "html_url": "http://foo", "assets": [{"name": "` + binaryAssetName() + `", "browser_download_url": "http://` + r.Host + `/bin"}, {"name": "` + binaryAssetName() + `.sha3sum", "browser_download_url": "http://` + r.Host + `/sum"}]}`))
	})
	mux.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`5c70c10245e6eb0fe36bd6bb8afc39f7b90ee356edae4409393fbcfdf65c1315`)) // SHA3-256 of "bad bin"
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`bad bin`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. isNewer, AutoApply = true (already tested in TestManager_Run, but we can do it again)
	tagToReturn = "v99.0.0"
	m := NewManager(Config{ReleaseURL: ts.URL, AutoApply: true}, nil)
	m.check(ctx)

	// 2. isNewer, AutoApply = false
	tagToReturn = "v99.0.0"
	m = NewManager(Config{ReleaseURL: ts.URL, AutoApply: false}, nil)
	m.check(ctx)

	// 3. rel.Version.Equal(current)
	tagToReturn = Current().String()
	m = NewManager(Config{ReleaseURL: ts.URL, AutoApply: true}, nil)
	m.check(ctx)

	// 4. isOlder, AllowDowngrade = false
	tagToReturn = "v0.0.1"
	m = NewManager(Config{ReleaseURL: ts.URL, AutoApply: true}, nil)
	m.check(ctx)

	// 5. isOlder, AllowDowngrade = true
	tagToReturn = "v0.0.1"
	m = NewManager(Config{ReleaseURL: ts.URL, AutoApply: true, AllowDowngrade: true}, nil)
	m.check(ctx)
}

func TestManager_FetchFailure(t *testing.T) {
	ctx := context.Background()
	m := NewManager(Config{ReleaseURL: "http://127.0.0.1:0"}, nil) // connection refused
	m.check(ctx)
}

func TestManager_Run(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "v99.0.0", "html_url": "http://foo", "assets": [{"name": "` + binaryAssetName() + `", "browser_download_url": "http://` + r.Host + `/bin"}, {"name": "` + binaryAssetName() + `.sha3sum", "browser_download_url": "http://` + r.Host + `/sum"}]}`))
	})
	mux.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`5c70c10245e6eb0fe36bd6bb8afc39f7b90ee356edae4409393fbcfdf65c1315`)) // SHA3-256 of "bad bin"
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`bad bin`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cfg := Config{
		ReleaseURL:    ts.URL,
		CheckInterval: 50 * time.Millisecond,
		AutoApply:     true,
	}
	m := NewManager(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	m.Run(ctx)
}

func TestManager_ApplyRelease_HashErr(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`bad bin`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx := context.Background()
	m := NewManager(Config{}, nil)

	rel := &Release{
		Version:     Current(),
		DownloadURL: ts.URL + "/bin",
		Checksum:    "5c70c10245e6eb0fe36bd6bb8afc39f7b90ee356edae4409393fbcfdf65c1315", // matches "bad bin"
	}

	// Mock hash error
	orig := computeFileHashFunc
	computeFileHashFunc = func(path string) (string, error) {
		return "", os.ErrNotExist
	}
	m.applyRelease(ctx, rel)

	// Mock hash mismatch
	m2 := NewManager(Config{}, nil)
	computeFileHashFunc = func(path string) (string, error) {
		return "wronghash", nil
	}
	m2.applyRelease(ctx, rel)
	
	computeFileHashFunc = orig
}

func TestManager_ApplyRelease_DownloadError(t *testing.T) {
	ctx := context.Background()
	m := NewManager(Config{}, nil)
	m.applyRelease(ctx, &Release{
		Version:     Current(),
		DownloadURL: "http://127.0.0.1:0", // Invalid URL
	})
}
