# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is BS3?

BS3 is a self-hosted secrets vault for homelab environments — a lightweight alternative to HashiCorp Vault. It uses envelope encryption (AES-256-GCM + Argon2id) backed by SQLite, and ships as both a REST API server and a companion CLI with a TUI.

## Repository Structure

Three separate Go modules — build and test each independently:

- **`server/`** — The vault HTTP API server
- **`cli-tool/`** — CLI and TUI for interacting with the server
- **`logger/`** — Shared logging module (imported by the other two)

## Commands

All commands run from within the respective module directory.

### Server (`cd server/`)

```bash
go run ./cmd/              # Run dev server (or use ./bs3server.sh)
go build -o bs3-server ./cmd  # Build binary
./builddocker.sh           # build Docker image (run from any directory)
go test ./...              # Run all tests
go test ./internal/cryptoutil/... -run TestFunctionName  # Run specific test
go vet ./...               # Lint
```

Server flags: `--verbose` (debug logging). Configure port via `VAULT_API_PORT` env var (default: 8080).

### CLI (`cd cli-tool/`)

```bash
go run .                   # Run CLI
./build.sh                 # Cross-compile for linux/amd64 and linux/arm64
go test ./...
go vet ./...
```

### Logger (`cd logger/`)

```bash
./test.sh                  # Run logger tests
```

## Architecture

### Server Internals

**Vault state machine:** `Uninitialized → Locked → Unlocked`
- Uninitialized: no DB exists; a one-time bootstrap Bearer token is printed to stdout
- Locked: DB exists but master key not in memory; must `POST /openvault` with passphrase
- Unlocked: master key held in RAM (never persisted); secrets are accessible

**Key files:**
- `server/cmd/main.go` — startup, flag parsing, HTTP server setup, graceful shutdown
- `server/internal/api/api.go` — all HTTP handlers + `authMiddleware`
- `server/internal/vault/vault.go` — Vault struct, SQLite ops, state management
- `server/internal/cryptoutil/cryptoutil.go` — Argon2id key derivation, AES-256-GCM encryption, HMAC token signing
- `server/internal/constants/constants.go` — hardcoded data paths and env var names

**Encryption model (envelope encryption):**
1. Each secret encrypted with a unique DEK (AES-256-GCM)
2. DEK encrypted with master key (AES-256-GCM), stored alongside ciphertext
3. Master key = Argon2id(passphrase + salt) — derived at runtime, never written to disk

**Auth:** `authMiddleware` accepts Bearer tokens (HMAC-SHA256 signed with master key, optional TTL) or HTTP Basic Auth (Argon2-hashed passwords in `users` table). A background goroutine cleans expired tokens every 24 hours.

### CLI Internals

- `cli-tool/internal/cli/cli.go` — command router
- `cli-tool/internal/apiclient/` — HTTP client wrapper
- `cli-tool/internal/tui/` — Charmbracelet/Bubbletea TUI
- `cli-tool/internal/injector/` — env var injection into processes
- `cli-tool/internal/enveditor/` — `.env` file manipulation

### Logger Module

Shared across server and CLI. Uses Charmbracelet (lipgloss) for styled terminal output and includes the ASCII art BS3 logo.

## Data Storage (Docker)

Server expects a `/data` volume mount:
- `/data/vault.db` — SQLite database (encrypted secrets, users, tokens)
- `/data/vault_salt` — Argon2id salt (not a secret, but must persist across restarts)

The `server/internal/constants/constants.go` data path is dev-specific and differs from the Docker path — check before modifying.

## Security Notes

- TLS is not built in — put BS3 behind a reverse proxy (Caddy, Nginx) for HTTPS.
- Master key is RAM-only; vault must be re-opened after every server restart.
- Each secret has an independent DEK — a leaked DEK only exposes one secret.
