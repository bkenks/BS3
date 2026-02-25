<p align="center">
  <img src="assets/bs3_Light_BG-Hero-w_Full_Title-With_Radial.svg" alt="BS3 Logo" width="600"/>
</p>

<p align="center">
  A self-hosted secrets vault for homelabbers who want real encryption without the enterprise price tag.
</p>

<p align="center">
  <img src="assets/Dual_Demo_with_Drop.png" alt="BS3 in action" width="800"/>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go version"/>
  <img src="https://img.shields.io/badge/SQLite-backend-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite"/>
  <img src="https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker"/>
  <img src="https://img.shields.io/badge/encryption-AES--256--GCM-brightgreen?style=flat-square" alt="AES-256-GCM"/>
</p>

---

## What is BS3?

**BS3** is a lightweight, self-hosted secret management server and CLI built for homelab environments. It exposes a REST API backed by a SQLite database and uses the same **envelope encryption** strategy employed by tools like HashiCorp Vault and AWS Secrets Manager — but without the complexity or cost.

Secrets are encrypted at rest with AES-256-GCM. The master key never touches disk. Authentication supports both HTTP Basic Auth and HMAC-signed Bearer tokens with optional TTL expiration.

> **Intended for homelab use.** If you're currently shoving secrets into `.env` files, this is for you.

---

## How It Works

### Vault Lifecycle

The vault operates in three states:

```
Uninitialized  →  POST /initvault  →  Locked  →  POST /openvault  →  Unlocked
```

| State | Description |
|---|---|
| **Uninitialized** | No database exists. A one-time Bearer token is printed to stdout for bootstrapping. |
| **Locked** | Database exists but the master key is not in memory. Vault must be opened with the master passphrase before secrets can be accessed. |
| **Unlocked** | Master key is held in memory (RAM only — never written to disk). Secrets can be read and written. |

### Encryption Model

BS3 uses **envelope encryption**:

1. Each secret is encrypted with a unique, randomly generated **DEK** (Data Encryption Key) using AES-256-GCM.
2. The DEK is itself encrypted with the **master key** and stored alongside the ciphertext.
3. The master key is derived at runtime from the master passphrase + salt using **Argon2id** — it is never persisted anywhere.

```
Secret → [AES-256-GCM, DEK] → Encrypted Secret
DEK    → [AES-256-GCM, Master Key] → Encrypted DEK
Master Key ← Argon2id(passphrase + salt) — lives in RAM only
```

If someone steals the database, they get encrypted blobs and encrypted DEKs. Without the master key (which is never stored), nothing is readable.

### Authentication

All API routes are protected by `authMiddleware`, which supports two methods:

- **Bearer Token** — HMAC-SHA256 token keyed with the master key, transmitted as base64url. Supports optional TTL expiration.
- **HTTP Basic Auth** — Username + Argon2-hashed password stored in the `users` table.

---

## API Reference

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/initvault` | Initial token | Initialize vault with username, password, and master passphrase |
| `POST` | `/openvault` | Basic Auth | Unlock vault with master passphrase |
| `GET` | `/token?name=X&ttl=N` | Basic Auth | Generate a named Bearer token (TTL in seconds, optional) |
| `DELETE` | `/deletetoken?name=X` | Bearer or Basic | Delete a named token |
| `GET` | `/listtokens` | Bearer or Basic | List all tokens |
| `POST` | `/store` | Bearer or Basic | Store a named secret |
| `GET` | `/get?name=X` | Bearer or Basic | Retrieve a secret by name |
| `DELETE` | `/delete?name=X` | Bearer or Basic | Delete a secret by name |
| `GET` | `/listsecrets` | Bearer or Basic | List all secret names with timestamps |
| `POST` | `/adduser` | Bearer or Basic | Add a user |
| `DELETE` | `/deleteuser?username=X` | Bearer or Basic | Delete a user |
| `GET` | `/listusers` | Bearer or Basic | List all users |

### Example Workflow

```bash
# 1. Start the server — grab the one-time init token from stdout

# 2. Initialize the vault
curl -X POST http://localhost:8080/initvault \
  -H "Authorization: Bearer <init-token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"mypassword","master_passphrase":"mysuperpassphrase"}'

# 3. Open the vault after any restart
curl -X POST http://localhost:8080/openvault \
  -u admin:mypassword \
  -H "Content-Type: application/json" \
  -d '{"master_passphrase":"mysuperpassphrase"}'

# 4. Store a secret
curl -X POST http://localhost:8080/store \
  -u admin:mypassword \
  -H "Content-Type: application/json" \
  -d '{"name":"db_password","secret":"hunter2"}'

# 5. Retrieve a secret
curl http://localhost:8080/get?name=db_password \
  -u admin:mypassword

# 6. Generate a Bearer token (1 hour TTL)
curl "http://localhost:8080/token?name=ci_token&ttl=3600" \
  -u admin:mypassword
```

---

## Installation

### Docker Compose (Recommended)

```yaml
services:
  bs3-server:
    image: bs3-server:latest
    ports:
      - "8080:8080"
    volumes:
      - bs3-data:/data
    restart: unless-stopped

volumes:
  bs3-data:
```

The `/data` volume persists your encrypted database (`vault.db`) and salt file (`vault_salt`). Mount it to keep your vault across container restarts.

### Build from Source

```bash
git clone <repo-url>
cd BS3

# Build binary
go build -o bs3-server ./cmd

# Run
./bs3-server
./bs3-server --verbose   # enable debug logging
```

### Build Docker Image

```bash
docker build -t bs3-server .
```

The image uses a multi-stage build — only the compiled binary ends up in the final Alpine-based image (~10 MB).

---

## Configuration

| Method | Variable | Default | Description |
|---|---|---|---|
| Env var | `VAULT_API_PORT` | `8080` | Port the server listens on |
| Flag | `--verbose` | off | Enable debug-level logging |

---

## CLI Tool

BS3 ships with a companion CLI for interacting with the server. It supports both a standard command-line interface and a **TUI** (Terminal User Interface).

```bash
# Launch the TUI
bs3 tui

# Command tree
bs3
├── openvault
├── envject
├── generatetoken
├── set
│   ├── apitoken
│   ├── serverurl
│   ├── username
│   └── password
└── list
    ├── secrets
    ├── users
    └── tokens
```

---

## Development

```bash
# Run tests
go test ./...

# Run a specific test
go test ./internal/cryptoutil/... -run TestFunctionName

# Lint
go vet ./...
```

### Project Structure

| File | Purpose |
|---|---|
| `cmd/main.go` | Entry point: flag parsing, vault state check, HTTP server, graceful shutdown |
| `internal/api/api.go` | All HTTP handlers and auth middleware |
| `internal/vault/vault.go` | Vault struct, DB operations, secret CRUD |
| `internal/cryptoutil/cryptoutil.go` | Argon2id, AES-GCM, HMAC token generation |
| `internal/constants/constants.go` | File paths and env var names |

---

## Security Notes

- The master key is **never written to disk** — it lives in RAM only and is gone when the process stops.
- Each secret has its own unique DEK — a compromised secret does not expose others.
- Passwords are hashed with **Argon2id** before storage.
- Bearer tokens are **HMAC-SHA256** signed with the master key and optionally time-limited.
- TLS is not built in — put BS3 behind a reverse proxy (e.g. Caddy, Nginx) if you need HTTPS.

> If you spot a security issue, open a pull request or issue. Contributions are welcome.

---

*BS3 — Because `.env` files aren't a secrets strategy.*
