package rpc

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/metrics"
)

const maxRequestBody = 4 << 20 // 4 MiB

// Server is the JSON-RPC 2.0 HTTP server.
type Server struct {
	api  *API
	http *http.Server
}

// NewServer creates a JSON-RPC HTTP server that listens on addr.
// readTimeout and writeTimeout default to 30 s if zero.
func NewServer(addr string, api *API, readTimeout, writeTimeout time.Duration) *Server {
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}

	mux := http.NewServeMux()
	s := &Server{api: api}
	mux.HandleFunc("/", s.handleRPC)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.http = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	return s
}

// ServeHTTP implements http.Handler so the Server can be used directly in tests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.http.Handler.ServeHTTP(w, r)
}

// Start begins serving requests in a background goroutine.
// It shuts down gracefully when ctx is canceled.
func (s *Server) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutCtx)
	}()
	go func() {
		log.Printf("[rpc] JSON-RPC server listening on http://%s", s.http.Addr)
		_ = s.http.ListenAndServe()
	}()
}

// handleRPC handles all inbound JSON-RPC requests (both single and batch).
func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method isn't allowed", http.StatusMethodNotAllowed)
		return
	}

	// CORS — allow all origins for now (tighten in production with middleware).
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil {
		writeJSON(w, errResponse(nil, CodeParseError, "read body: "+err.Error()))
		return
	}

	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		writeJSON(w, errResponse(nil, CodeInvalidRequest, "empty request body"))
		return
	}

	// Detect batch vs. single request.
	if trimmed[0] == '[' {
		s.handleBatch(w, body)
		return
	}
	s.handleSingle(w, body)
}

func (s *Server) handleSingle(w http.ResponseWriter, body []byte) {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, errResponse(nil, CodeParseError, err.Error()))
		return
	}
	if req.JSONRPC != "2.0" {
		writeJSON(w, errResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\""))
		return
	}
	metrics.RPCRequests.WithLabelValues(req.Method).Inc()
	resp := s.api.Dispatch(&req)
	if resp.Error != nil {
		metrics.RPCErrors.WithLabelValues(req.Method).Inc()
	}
	writeJSON(w, resp)
}

func (s *Server) handleBatch(w http.ResponseWriter, body []byte) {
	var reqs []json.RawMessage
	if err := json.Unmarshal(body, &reqs); err != nil {
		writeJSON(w, errResponse(nil, CodeParseError, err.Error()))
		return
	}
	if len(reqs) == 0 {
		writeJSON(w, errResponse(nil, CodeInvalidRequest, "empty batch"))
		return
	}

	responses := make([]*Response, 0, len(reqs))
	for _, raw := range reqs {
		var req Request
		if err := json.Unmarshal(raw, &req); err != nil {
			responses = append(responses, errResponse(nil, CodeParseError, err.Error()))
			continue
		}
		metrics.RPCRequests.WithLabelValues(req.Method).Inc()
		resp := s.api.Dispatch(&req)
		if resp.Error != nil {
			metrics.RPCErrors.WithLabelValues(req.Method).Inc()
		}
		responses = append(responses, resp)
	}
	writeJSON(w, responses)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}
