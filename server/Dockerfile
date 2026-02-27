# ── Stage 1: Build ───────────────────────────────────────────────────────────
# Build context must be the repo root so the logger module (referenced via
# replace directive in server/go.mod) is available at ../logger relative to
# the server directory.
#
# Build with:  docker build -f server/Dockerfile -t bs3-server .
FROM golang:1.26.0-alpine AS builder

WORKDIR /build

# Copy the logger module first — server/go.mod has:
#   replace github.com/bkenks/bs3-logger => ../logger
# so Go expects to find it one level above the server working directory.
COPY logger/ ./logger/

# Copy server dependency manifests and download modules before copying source.
# Docker caches each layer, so this layer only re-runs when go.mod/go.sum
# change, not on every source edit — keeping rebuilds fast.
COPY server/go.mod server/go.sum ./server/
WORKDIR /build/server
RUN apk add --no-cache git && go mod download

# Copy the rest of the server source and compile. The output binary is placed
# at /build/server/bs3-server inside the builder container.
COPY server/ .
RUN go build -o bs3-server ./cmd

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
# Start fresh from a minimal Alpine image. None of the Go toolchain, source
# code, or module cache from the builder stage is carried over — only the
# compiled binary is copied in. This keeps the final image small (~10 MB).
FROM alpine:3.23.3

# Create the data directory expected by the vault at runtime. The DB and salt
# file paths are hardcoded in internal/constants/constants.go as:
#   /data/vault.db
#   /data/vault_salt
# A volume or bind-mount should be attached here to persist vault data.
RUN mkdir -p /data

WORKDIR /app

# Pull only the compiled binary from the builder stage.
COPY --from=builder /build/server/bs3-server .

# The server listens on 8080 by default. Override with the VAULT_API_PORT
# environment variable at runtime (e.g. docker run -e VAULT_API_PORT=9090).
EXPOSE 8080

ENTRYPOINT ["./bs3-server"]
