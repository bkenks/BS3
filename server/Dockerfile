# ── Stage 1: Build ───────────────────────────────────────────────────────────
# Use the full Go toolchain image to compile the binary. This image is large
# (~300 MB) but is discarded after compilation — it never ends up in the final
# image.
FROM golang:1.26.0-alpine AS builder

WORKDIR /app

# Copy dependency manifests first and download modules before copying source.
# Docker caches each layer, so this layer only re-runs when go.mod/go.sum
# change, not on every source edit — keeping rebuilds fast.
COPY go.mod go.sum ./
RUN apk add --no-cache git && go mod download

# Copy the rest of the source and compile. The output binary is placed at
# /app/bs3-server inside the builder container.
COPY . .
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
COPY --from=builder /app/bs3-server .

# The server listens on 8080 by default. Override with the VAULT_API_PORT
# environment variable at runtime (e.g. docker run -e VAULT_API_PORT=9090).
EXPOSE 8080

ENTRYPOINT ["./bs3-server"]
