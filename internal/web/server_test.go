package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/web"
)

func TestHandler_RootReturnsIndex(t *testing.T) {
	h := web.Handler()
	if h == nil {
		t.Fatal("web.Handler returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /: want 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type: want text/html, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "QubitsCoin") {
		t.Errorf("body missing 'QubitsCoin', got: %s", body[:min(len(body), 100)])
	}
}

func TestHandler_ExplicitIndexHtml(t *testing.T) {
	h := web.Handler()

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// Go's http.FileServer redirects /index.html to ./ (StatusMovedPermanently)
	if rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusOK {
		t.Errorf("GET /index.html: want 301 or 200, got %d", rec.Code)
	}
}

func TestHandler_HeadMethodSupported(t *testing.T) {
	h := web.Handler()

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HEAD /: want 200, got %d", rec.Code)
	}
}

func TestHandler_PostMethodNotAllowed(t *testing.T) {
	h := web.Handler()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"test":1}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / on web handler: want 405, got %d", rec.Code)
	}
}

func TestHandler_NotFound(t *testing.T) {
	h := web.Handler()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-file-12345.js", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /nonexistent: want 404, got %d", rec.Code)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
