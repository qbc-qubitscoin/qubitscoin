package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/keystore"
)

func main() {
	rpcAddr := flag.String("rpc", "http://127.0.0.1:8545", "RPC endpoint")
	ksPath := flag.String("keystore", "keystore.json", "Path to funded keystore")
	duration := flag.Duration("duration", 10*time.Second, "Duration to run load test")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	flag.Parse()

	password := os.Getenv("QBC_PASSWORD")
	if password == "" {
		log.Fatal("QBC_PASSWORD environment variable is required")
	}

	w, err := keystore.Decrypt(*ksPath, password)
	if err != nil {
		log.Fatalf("Failed to decrypt keystore: %v", err)
	}

	log.Printf("Loaded wallet: %s", crypto.AddressToHex(w.Address))
	log.Printf("Starting load test against %s for %v with %d workers", *rpcAddr, *duration, *workers)

	// Fetch initial nonce
	nonce, err := fetchNonce(*rpcAddr, crypto.AddressToHex(w.Address))
	if err != nil {
		log.Fatalf("Failed to fetch nonce: %v", err)
	}
	log.Printf("Starting nonce: %d", nonce)

	var (
		txSent     uint64
		txFailed   uint64
		totalLatMs uint64
		wg         sync.WaitGroup
	)

	// Since we are sending from one wallet, we must strictly assign nonces
	var currentNonce uint64 = nonce

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	start := time.Now()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					n := atomic.AddUint64(&currentNonce, 1) - 1
					
					// Just send to ourselves to avoid needing another address
					tx := core.NewTransfer(w.Address, w.Address, w.PublicKey, n, 1, core.InitialBaseFee)
					if err := tx.Sign(w.PrivateKey); err != nil {
						atomic.AddUint64(&txFailed, 1)
						continue
					}

					raw, _ := gobEncodeTx(tx)
					rawHex := hex.EncodeToString(raw)

					t0 := time.Now()
					_, err := rpcCall(*rpcAddr, "qbc_sendRawTransaction", []string{rawHex})
					latMs := uint64(time.Since(t0).Milliseconds())
					atomic.AddUint64(&totalLatMs, latMs)

					if err != nil {
						atomic.AddUint64(&txFailed, 1)
					} else {
						atomic.AddUint64(&txSent, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start).Seconds()
	
	sent := atomic.LoadUint64(&txSent)
	failed := atomic.LoadUint64(&txFailed)
	totLat := atomic.LoadUint64(&totalLatMs)

	avgLat := float64(0)
	if sent+failed > 0 {
		avgLat = float64(totLat) / float64(sent+failed)
	}

	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration : %.2f seconds\n", elapsed)
	fmt.Printf("Sent     : %d\n", sent)
	fmt.Printf("Failed   : %d\n", failed)
	fmt.Printf("TPS      : %.2f\n", float64(sent)/elapsed)
	fmt.Printf("Avg Lat. : %.2f ms\n", avgLat)
}

func fetchNonce(endpoint, addr string) (uint64, error) {
	result, err := rpcCall(endpoint, "qbc_getTransactionCount", []string{addr})
	if err != nil {
		return 0, err
	}
	if v, ok := result["nonce"]; ok {
		if n, ok := v.(float64); ok {
			return uint64(n), nil
		}
	}
	return 0, nil
}

type rpcReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func rpcCall(endpoint, method string, params interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(rpcReq{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}

	httpResp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp rpcResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return map[string]interface{}{"value": string(resp.Result)}, nil
	}
	return result, nil
}

func gobEncodeTx(tx *core.Transaction) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(tx); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
