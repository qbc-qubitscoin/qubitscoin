# ─────────────────────────────────────────────────────────────────────────────
# Stage 1 — Build
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

# Version injected via --build-arg VERSION=vX.Y.Z (or Makefile sets it).
ARG VERSION=v0.5.0

# Install build dependencies (C compiler not needed — pure Go).
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-s -w -X github.com/qbc-qubitscoin/qubitscoin/internal/upgrade.version=${VERSION}" \
      -trimpath \
      -o /out/qbc-node \
      ./cmd/node

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2 — Runtime (scratch + CA certs for TLS to GitHub Releases API)
# ─────────────────────────────────────────────────────────────────────────────
FROM scratch

# Bring in CA certificates for HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Bring in the node binary.
COPY --from=builder /out/qbc-node /qbc-node

# Default config directory.
VOLUME ["/data"]

# P2P port (TCP).
EXPOSE 8765/tcp

# JSON-RPC port.
EXPOSE 8545/tcp

# Prometheus metrics port.
EXPOSE 9090/tcp

ENTRYPOINT ["/qbc-node"]
CMD ["start", "--config", "/data/config.toml", "--datadir", "/data"]
