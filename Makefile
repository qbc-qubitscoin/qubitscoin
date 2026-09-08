# ─────────────────────────────────────────────────────────────────────────────
# QubitsCoin (QBC) — Makefile
# ─────────────────────────────────────────────────────────────────────────────

MODULE  := github.com/qbc-qubitscoin/qubitscoin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.5.0")
LDFLAGS := -s -w -X $(MODULE)/internal/upgrade.version=$(VERSION)
BINARY  := bin/qbc-node
GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)

.PHONY: all build test lint clean docker docker-push \
        testnet-up testnet-down run-node help

## ── Primary targets ──────────────────────────────────────────────────────────

all: test build   ## Run tests then build (default)

build:            ## Build the qbc-node binary for the current platform
	@echo "▶ Building $(BINARY) ($(GOOS)/$(GOARCH))…"
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
	  go build -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) ./cmd/node
	@echo "  ✓ $(BINARY)"

build-linux:      ## Cross-compile for linux/amd64
	GOOS=linux GOARCH=amd64 $(MAKE) build BINARY=bin/qbc-node-linux-amd64

build-darwin:     ## Cross-compile for darwin/arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(MAKE) build BINARY=bin/qbc-node-darwin-arm64

build-windows:    ## Cross-compile for windows/amd64
	GOOS=windows GOARCH=amd64 $(MAKE) build BINARY=bin/qbc-node-windows-amd64.exe

build-all: build-linux build-darwin build-windows  ## Build for all platforms

## ── Testing ──────────────────────────────────────────────────────────────────

test:             ## Run all unit tests
	@echo "▶ Running tests…"
	go test -count=1 -timeout=120s ./...

test-v:           ## Run tests with verbose output
	go test -v -count=1 -timeout=120s ./...

test-race:        ## Run tests with race detector
	go test -race -count=1 -timeout=180s ./...

test-cover:       ## Run tests with HTML coverage report
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -timeout=120s ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "  ✓ Coverage report: coverage/coverage.html"

bench:            ## Run benchmarks
	go test -bench=. -benchmem -timeout=60s ./...

## ── Code quality ─────────────────────────────────────────────────────────────

lint:             ## Run golangci-lint (must be installed)
	@which golangci-lint > /dev/null 2>&1 || \
	  (echo "golangci-lint not found — install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

vet:              ## Run go vet
	go vet ./...

fmt:              ## Format all Go source files
	gofmt -w .
	goimports -w . 2>/dev/null || true

tidy:             ## Tidy go.mod / go.sum
	go mod tidy

deadcode:         ## Report unreachable functions (requires golang.org/x/tools/cmd/deadcode)
	@which deadcode > /dev/null 2>&1 || go install golang.org/x/tools/cmd/deadcode@latest
	deadcode -test ./...

## ── Docker ───────────────────────────────────────────────────────────────────

docker:           ## Build the Docker image
	@echo "▶ Building Docker image qbc-node:$(VERSION)…"
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --tag qbc-node:$(VERSION) \
	  --tag qbc-node:latest \
	  .

docker-push:      ## Push Docker image to registry (set REGISTRY env var)
	@[ -n "$(REGISTRY)" ] || (echo "REGISTRY is not set" && exit 1)
	docker tag qbc-node:$(VERSION) $(REGISTRY)/qbc-node:$(VERSION)
	docker tag qbc-node:latest     $(REGISTRY)/qbc-node:latest
	docker push $(REGISTRY)/qbc-node:$(VERSION)
	docker push $(REGISTRY)/qbc-node:latest

## ── Local node ───────────────────────────────────────────────────────────────

run-node: build   ## Build and start a local node (miner, testnet config)
	@echo "▶ Starting local testnet node…"
	QBC_PASSWORD=dev $(BINARY) start \
	  --config configs/testnet.toml \
	  --miner \
	  --rpc-listen 127.0.0.1:8545 \
	  --metrics \
	  --metrics-addr 127.0.0.1:9090

testnet-up:       ## Start the Docker Compose testnet stack
	@cp -n configs/mainnet.toml data/config.toml 2>/dev/null || true
	docker compose up -d

testnet-down:     ## Stop the Docker Compose testnet stack
	docker compose down

testnet-logs:     ## Tail logs from the Docker Compose stack
	docker compose logs -f

## ── Wallet helpers ───────────────────────────────────────────────────────────

wallet-new: build ## Create a new wallet (prompts for password via env QBC_PASSWORD)
	@[ -n "$(QBC_PASSWORD)" ] || (echo "Set QBC_PASSWORD env var first" && exit 1)
	$(BINARY) wallet new --datadir ~/.qbc

wallet-show: build ## Show the address in the default keystore
	$(BINARY) wallet show --datadir ~/.qbc

## ── Release ──────────────────────────────────────────────────────────────────

release: test build-all ## Full release: test + cross-compile all platforms
	@echo "▶ Release $(VERSION) artifacts:"
	@ls -lh bin/

## ── Housekeeping ─────────────────────────────────────────────────────────────

clean:            ## Remove build artefacts
	rm -rf bin/ coverage/

help:             ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
