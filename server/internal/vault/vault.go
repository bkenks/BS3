package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	l "github.com/bkenks/bs3-logger"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/cryptoutil"
)

// Sentinel errors for secret CRUD operations.
var (
	ErrSecretExists   = errors.New("secret already exists")
	ErrSecretNotFound = errors.New("secret not found")
	ErrFolderExists   = errors.New("folder already exists")
)

// --- Vault Object ---
// struct representing a "vault" with database, master key, state, etc.

type VaultState int

const (
	Uninitialized VaultState = iota
	Unlocked
	Locked
)

type Vault struct {
	mu        sync.RWMutex
	db        *sql.DB
	masterKey []byte
	salt      []byte
	state     VaultState
}

// --- END "Vault Object" ---

// =====================================================
// Vault Helpers
// =====================================================
// helper functions for interacting with the vault internally

// --- Getters/Setters ---

func (v *Vault) GetState() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	switch v.state {
	case Uninitialized:
		return "Uninitialized"
	case Unlocked:
		return "Unlocked"
	case Locked:
		return "Locked"
	default:
		return fmt.Sprintf("Unknown(%d)", int(v.state))
	}
}

func (v *Vault) SetState(state VaultState) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.state = state
}

func (v *Vault) SetMasterKey(masterKey []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.masterKey = masterKey
}

func (v *Vault) GetMasterKey() []byte {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.masterKey
}

func (v *Vault) GetDB() *sql.DB {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.db
}

// --- END "Getters/Setters" ---

// ~~~ IsInitialized ~~~
// checks state to determine if database is initialized
func (v *Vault) IsInitialized() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.state != Uninitialized
}

func (v *Vault) IsUnlocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.state == Unlocked
}

// ~~~ connectDB ~~~
// opens a new SQLite connection, closing any existing one first.
// Caller must hold v.mu write lock.
func (v *Vault) connectDB() error {
	if v.db != nil {
		v.db.Close()
		v.db = nil
	}
	db, err := sql.Open("sqlite", constants.VaultPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database | %w", err)
	}
	v.db = db
	return nil
}

// ConnectDB is the exported, concurrency-safe wrapper around connectDB.
func (v *Vault) ConnectDB() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.connectDB()
}

// ~~~ CheckVaultState ~~~
// checks if a vault exists on startup, if so sets state to locked and connects to db
func (v *Vault) CheckVaultState() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	var err error

	_, err = os.Stat(constants.VaultPath)
	vaultExists := (err == nil)

	if vaultExists {
		if v.salt, err = cryptoutil.LoadSalt(); err != nil {
			return fmt.Errorf("db exists, but salt missing | %w", err)
		}
		if err = v.connectDB(); err != nil {
			return err
		}
		if err = v.migrateSchema(); err != nil {
			return fmt.Errorf("failed to migrate vault schema | %w", err)
		}
		v.state = Locked
	} else {
		v.state = Uninitialized
	}

	return nil
}

// ~~~ migrateSchema ~~~
// applies in-place schema migrations to an existing vault database.
// Currently adds the secrets.folder column if it is missing.
// Idempotent. Caller must hold v.mu write lock.
func (v *Vault) migrateSchema() error {
	if v.db == nil {
		return fmt.Errorf("vault not initialized")
	}

	rows, err := v.db.Query(`PRAGMA table_info(secrets)`)
	if err != nil {
		return fmt.Errorf("failed to inspect secrets schema | %w", err)
	}
	defer rows.Close()

	hasFolder := false
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); err != nil {
			return fmt.Errorf("failed to scan secrets schema | %w", err)
		}
		if name == "folder" {
			hasFolder = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read secrets schema | %w", err)
	}

	if !hasFolder {
		_, err := v.db.Exec(`ALTER TABLE secrets ADD COLUMN folder TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("failed to add folder column | %w", err)
		}
	}

	// Step 2: rebuild the secrets table to drop the global UNIQUE(name)
	// constraint in favour of a composite UNIQUE(name, folder).
	var tableSQL string
	if err := v.db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='secrets'`,
	).Scan(&tableSQL); err != nil {
		return fmt.Errorf("failed to inspect secrets table sql | %w", err)
	}

	if !strings.Contains(tableSQL, "UNIQUE(name, folder)") {
		tx, err := v.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin secrets rebuild | %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`
			CREATE TABLE secrets_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				folder TEXT NOT NULL DEFAULT '',
				encrypted_dek BLOB NOT NULL,
				encrypted_data BLOB NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(name, folder)
			)
		`); err != nil {
			return fmt.Errorf("failed to create secrets_new table | %w", err)
		}

		if _, err := tx.Exec(`
			INSERT INTO secrets_new (id, name, folder, encrypted_dek, encrypted_data, created_at, updated_at)
			SELECT id, name, folder, encrypted_dek, encrypted_data, created_at, updated_at FROM secrets
		`); err != nil {
			return fmt.Errorf("failed to copy secrets into secrets_new | %w", err)
		}

		if _, err := tx.Exec(`DROP TABLE secrets`); err != nil {
			return fmt.Errorf("failed to drop old secrets table | %w", err)
		}

		if _, err := tx.Exec(`ALTER TABLE secrets_new RENAME TO secrets`); err != nil {
			return fmt.Errorf("failed to rename secrets_new table | %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit secrets rebuild | %w", err)
		}
	}

	// Step 3: ensure the folders table exists so an empty, persistent folder
	// can be created independently of any secret.
	if _, err := v.db.Exec(`
		CREATE TABLE IF NOT EXISTS folders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create folders table | %w", err)
	}

	return nil
}

// ~~~ storeMasterKeyHash ~~~
// stores the hashed master key in the database for verification when opening/unsealing.
// Caller must hold v.mu write lock.
func (v *Vault) storeMasterKeyHash() error {
	if v.db == nil || v.masterKey == nil {
		return fmt.Errorf("vault not initialized")
	}

	hash := cryptoutil.HashMasterKey(v.masterKey)

	_, err := v.db.Exec(`
		CREATE TABLE IF NOT EXISTS vault_meta (
			id INTEGER PRIMARY KEY,
			master_hash BLOB NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create vault_meta table | %w", err)
	}

	_, err = v.db.Exec(`
		INSERT INTO vault_meta (id, master_hash)
		VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET master_hash = excluded.master_hash
	`, hash)
	if err != nil {
		return fmt.Errorf("failed to store master key hash | %w", err)
	}

	return nil
}

// StoreMasterKeyHash is the exported, concurrency-safe wrapper.
func (v *Vault) StoreMasterKeyHash() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.storeMasterKeyHash()
}

// ~~~ Close ~~~
// safely closes the connection to the database
func (v *Vault) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.db != nil {
		v.db.Close()
		v.db = nil
	}
	return nil
}

// =====================================================
// END "Vault Helpers"
// =====================================================

// ~~~ InitializeVault ~~~
// creates new database, masterkey, username, and password
func (v *Vault) InitializeVault(username, password, masterPassphrase string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	var err error
	var masterKey []byte

	// Ensure parent directory exists
	dir := filepath.Dir(constants.DataPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create '/data' directory | %w", err)
	}

	if v.salt, err = cryptoutil.CreateSalt(); err != nil {
		return err
	}

	if masterKey, err = cryptoutil.DeriveMasterKey(masterPassphrase, v.salt); err != nil {
		return err
	}

	if err = v.connectDB(); err != nil {
		return err
	}

	// Ping DB to verify connection
	if err := v.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database | %w", err)
	}

	// Create secrets table
	_, err = v.db.Exec(`
		CREATE TABLE IF NOT EXISTS secrets (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            folder TEXT NOT NULL DEFAULT '',
            encrypted_dek BLOB NOT NULL,
            encrypted_data BLOB NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(name, folder)
        )
	`)
	if err != nil {
		return fmt.Errorf("failed to create secrets table | %w", err)
	}

	// Create users table
	_, err = v.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table | %w", err)
	}

	// Create api_tokens table
	_, err = v.db.Exec(`
		CREATE TABLE IF NOT EXISTS api_tokens (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL,
            token_hash BLOB NOT NULL,
            expires_at INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
	`)
	if err != nil {
		return fmt.Errorf("failed to create api_tokens table | %w", err)
	}

	// Create folders table
	_, err = v.db.Exec(`
		CREATE TABLE IF NOT EXISTS folders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
	`)
	if err != nil {
		return fmt.Errorf("failed to create folders table | %w", err)
	}

	// Hash password
	passwordSalt, err := cryptoutil.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate password salt | %w", err)
	}
	passwordHash, err := cryptoutil.HashPassword(password, passwordSalt)
	if err != nil {
		return fmt.Errorf("failed to hash password | %w", err)
	}

	// Store initial user
	_, err = v.db.Exec(`
		INSERT INTO users (username, password_hash)
		VALUES (?, ?)
	`, username, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to store username and password | %w", err)
	}

	// Store master key in memory and transition state
	v.masterKey = masterKey
	v.state = Unlocked

	if err := v.storeMasterKeyHash(); err != nil {
		return err
	}

	return nil
}

// ~~~ OpenVault ~~~
// derives the master key from the passphrase and transitions vault to Unlocked.
// The DB connection established by CheckVaultState is reused; no reconnect needed.
func (v *Vault) OpenVault(masterPassphrase string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	var salt []byte
	var masterKey []byte
	var err error

	if salt, err = cryptoutil.LoadSalt(); err != nil {
		return err
	}

	if masterKey, err = cryptoutil.DeriveMasterKey(masterPassphrase, salt); err != nil {
		return err
	}

	masterKeyMatches, err := v.verifyMasterKey(masterKey)
	if err != nil {
		return fmt.Errorf("could not verify master key | %v", err)
	}
	if !masterKeyMatches {
		return fmt.Errorf("master key does not match")
	}

	v.masterKey = masterKey
	v.state = Unlocked
	return nil
}

// =====================================================
// Verification
// =====================================================
// verification methods

// ~~~ VerifyUser ~~~
// verifies the username and whether password hashes match
func (v *Vault) VerifyUser(username, password string) (bool, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	if db == nil {
		return false, fmt.Errorf("vault not initialized or does not exist")
	}

	// Query Database for password hash given the username
	var storedHash string
	err := db.QueryRow(`
		SELECT password_hash
		FROM users
		WHERE username = ?
	`, username).Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// Do not reveal whether user exists
			return false, fmt.Errorf("invalid credentials")
		}
		return false, fmt.Errorf("database error | %w", err)
	}

	passwordsMatch, err := cryptoutil.VerifyPassword(password, storedHash)
	if err != nil {
		return false, fmt.Errorf("could not verify password: %v", err)
	}

	if !passwordsMatch {
		return false, fmt.Errorf("invalid credentials")
	}

	return true, nil
}

// ~~~ verifyMasterKey ~~~
// compares an input master key against the stored hash.
// Caller must hold v.mu (at least read lock).
func (v *Vault) verifyMasterKey(inputKey []byte) (bool, error) {
	if v.db == nil {
		return false, fmt.Errorf("vault not initialized")
	}

	var storedHash []byte
	err := v.db.QueryRow(`SELECT master_hash FROM vault_meta WHERE id = 1`).Scan(&storedHash)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve stored master key hash | %w", err)
	}

	return cryptoutil.VerifyMasterKey(inputKey, storedHash)
}

// VerifyMasterKey is the exported, concurrency-safe wrapper.
func (v *Vault) VerifyMasterKey(inputKey []byte) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.verifyMasterKey(inputKey)
}

// =====================================================
// END "Verification"
// =====================================================

// =====================================================
// HTTP Request Runners
// =====================================================
// functions that run with http requests

// ~~~ AddUser ~~~
// hashes password and inserts a new user into the database
func (v *Vault) AddUser(username, password string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	passwordSalt, err := cryptoutil.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate password salt | %w", err)
	}
	passwordHash, err := cryptoutil.HashPassword(password, passwordSalt)
	if err != nil {
		return fmt.Errorf("failed to hash password | %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (username, password_hash)
		VALUES (?, ?)
	`, username, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to add user | %w", err)
	}
	return nil
}

// ~~~ DeleteUser ~~~
// removes a user from the database, refusing if they are the last user
func (v *Vault) DeleteUser(username string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("failed to count users | %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last user")
	}

	res, err := db.Exec(`DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return nil
}

// ~~~ StoreSecret ~~~
// encrypts a secret and stores it in the database. Create-only: storing a
// secret whose (name, folder) pair already exists returns ErrSecretExists.
func (v *Vault) StoreSecret(name string, plaintext []byte, folder string) error {
	v.mu.RLock()
	masterKey := v.masterKey
	db := v.db
	v.mu.RUnlock()

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM secrets WHERE name = ? AND folder = ?`,
		name, folder,
	).Scan(&count); err != nil {
		return fmt.Errorf("failed to check for existing secret | %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretExists, name, folder)
	}

	env, err := cryptoutil.ProtectSecret(masterKey, plaintext)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        INSERT INTO secrets (name, folder, encrypted_dek, encrypted_data)
        VALUES (?, ?, ?, ?)
    `, name, folder, env.EncryptedDEK, env.EncryptedData)

	return err
}

// ~~~ EditSecret ~~~
// updates the value of an existing secret identified by (name, folder).
// Returns ErrSecretNotFound if no such secret exists.
func (v *Vault) EditSecret(name, folder string, plaintext []byte) error {
	v.mu.RLock()
	masterKey := v.masterKey
	db := v.db
	v.mu.RUnlock()

	env, err := cryptoutil.ProtectSecret(masterKey, plaintext)
	if err != nil {
		return err
	}

	res, err := db.Exec(`
        UPDATE secrets
        SET encrypted_dek = ?, encrypted_data = ?, updated_at = ?
        WHERE name = ? AND folder = ?
    `, env.EncryptedDEK, env.EncryptedData, time.Now(), name, folder)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, folder)
	}
	return nil
}

// ~~~ GetSecret ~~~
// retreives a secret from the database (scoped to folder) and decrypts it
func (v *Vault) GetSecret(name, folder string) ([]byte, error) {
	v.mu.RLock()
	masterKey := v.masterKey
	db := v.db
	v.mu.RUnlock()

	row := db.QueryRow(`
		SELECT encrypted_dek, encrypted_data
		FROM secrets
		WHERE name = ? AND folder = ?
	`, name, folder)

	var encryptedDEK, encryptedData []byte
	err := row.Scan(&encryptedDEK, &encryptedData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, folder)
		}
		return nil, err
	}

	env := &cryptoutil.Envelope{
		EncryptedDEK:  encryptedDEK,
		EncryptedData: encryptedData,
	}

	plaintext, err := cryptoutil.UnprotectSecret(masterKey, env)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret %q | %w", name, err)
	}

	return plaintext, nil
}

// ~~~ DeleteSecret ~~~
// deletes a secret (scoped to folder) from the database
func (v *Vault) DeleteSecret(name, folder string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	res, err := db.Exec(`DELETE FROM secrets WHERE name = ? AND folder = ?`, name, folder)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, folder)
	}
	return nil
}

// ~~~ MoveSecret ~~~
// changes the folder of an existing secret from fromFolder to toFolder.
// An empty toFolder moves the secret to the root/ungrouped folder.
// Returns ErrSecretNotFound if the source does not exist, or ErrSecretExists
// if a secret with the same name already occupies the destination folder.
func (v *Vault) MoveSecret(name, fromFolder, toFolder string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	if fromFolder == toFolder {
		// Still verify the secret exists so callers get a consistent error.
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM secrets WHERE name = ? AND folder = ?`,
			name, fromFolder,
		).Scan(&count); err != nil {
			return fmt.Errorf("failed to check source secret | %w", err)
		}
		if count == 0 {
			return fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, fromFolder)
		}
		return nil
	}

	var srcCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM secrets WHERE name = ? AND folder = ?`,
		name, fromFolder,
	).Scan(&srcCount); err != nil {
		return fmt.Errorf("failed to check source secret | %w", err)
	}
	if srcCount == 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, fromFolder)
	}

	var dstCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM secrets WHERE name = ? AND folder = ?`,
		name, toFolder,
	).Scan(&dstCount); err != nil {
		return fmt.Errorf("failed to check destination secret | %w", err)
	}
	if dstCount > 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretExists, name, toFolder)
	}

	res, err := db.Exec(
		`UPDATE secrets SET folder = ?, updated_at = ? WHERE name = ? AND folder = ?`,
		toFolder, time.Now(), name, fromFolder,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: %q in folder %q", ErrSecretNotFound, name, fromFolder)
	}
	return nil
}

// ~~~ SecretInfo ~~~
// represents one secret in the database with additional useful information
// does not contain secret or key
type SecretInfo struct {
	Name      string
	Folder    string
	CreatedAt string
	UpdatedAt string
}

// ~~~ UserInfo ~~~
// represents one user entry (without the password hash)
type UserInfo struct {
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// ~~~ ListUsers ~~~
// returns all users in the database (username and created_at only, no hashes)
func (v *Vault) ListUsers() ([]UserInfo, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	rows, err := db.Query(`
		SELECT username, created_at
		FROM users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users | %w", err)
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.Username, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ~~~ TokenInfo ~~~
// represents one API token entry (without the hash)
type TokenInfo struct {
	Name      string `json:"name"`
	ExpiresAt *int64 `json:"expires_at"` // nil = never expires
	CreatedAt string `json:"created_at"`
}

// ~~~ StoreToken ~~~
// stores a named API token hash in the database
func (v *Vault) StoreToken(name string, tokenHash []byte, expiresAt *int64) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	_, err := db.Exec(`
		INSERT INTO api_tokens (name, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, name, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store token | %w", err)
	}
	return nil
}

// ~~~ VerifyAPIToken ~~~
// checks rawToken against all non-expired tokens in the database
func (v *Vault) VerifyAPIToken(rawToken []byte) (bool, error) {
	v.mu.RLock()
	masterKey := v.masterKey
	db := v.db
	v.mu.RUnlock()

	rows, err := db.Query(`
		SELECT token_hash, expires_at FROM api_tokens
	`)
	if err != nil {
		return false, fmt.Errorf("failed to query tokens | %w", err)
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		var hash []byte
		var expiresAt *int64
		if err := rows.Scan(&hash, &expiresAt); err != nil {
			return false, err
		}
		if expiresAt != nil && now > *expiresAt {
			continue // expired
		}
		matches, err := cryptoutil.VerifyToken(masterKey, rawToken, hash)
		if err == nil && matches {
			return true, nil
		}
	}
	return false, nil
}

// ~~~ DeleteToken ~~~
// removes a named API token from the database
func (v *Vault) DeleteToken(name string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	res, err := db.Exec(`DELETE FROM api_tokens WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token %q not found", name)
	}
	return nil
}

// ~~~ ListTokens ~~~
// returns all API token entries (name and expiry only, no hashes)
func (v *Vault) ListTokens() ([]TokenInfo, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	rows, err := db.Query(`
		SELECT name, expires_at, created_at
		FROM api_tokens
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tokens | %w", err)
	}
	defer rows.Close()

	var tokens []TokenInfo
	for rows.Next() {
		var t TokenInfo
		if err := rows.Scan(&t.Name, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// ~~~ ListSecrets ~~~
// returns all secrets in database with additional information
// using the SecretInfo struct to represent each database row
func (v *Vault) ListSecrets() ([]SecretInfo, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	rows, err := db.Query(`
		SELECT name, folder, created_at, updated_at
		FROM secrets
		ORDER BY folder ASC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive secrets | %w", err)
	}
	defer rows.Close()

	var secrets []SecretInfo
	for rows.Next() {
		var s SecretInfo
		if err := rows.Scan(&s.Name, &s.Folder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}

	return secrets, nil
}

// ~~~ FolderInfo ~~~
// represents a folder grouping and the number of secrets it contains
type FolderInfo struct {
	Folder      string `json:"folder"`
	SecretCount int    `json:"secret_count"`
}

// ~~~ CreateFolder ~~~
// inserts a new, empty folder into the folders table so it persists even
// while it holds no secrets. Returns ErrFolderExists if the name is taken.
func (v *Vault) CreateFolder(name string) error {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	_, err := db.Exec(`INSERT INTO folders (name) VALUES (?)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: %q", ErrFolderExists, name)
		}
		return fmt.Errorf("failed to create folder | %w", err)
	}
	return nil
}

// ~~~ ListFolders ~~~
// returns the union of every distinct secrets.folder (with its secret count)
// and every explicitly created folders.name (count 0 when empty). A folder
// present in both sources appears once with its real secret count. Sorted by
// folder name ascending.
func (v *Vault) ListFolders() ([]FolderInfo, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	counts := make(map[string]int)

	// Distinct secret folders with their counts.
	secretRows, err := db.Query(`
		SELECT folder, COUNT(*)
		FROM secrets
		GROUP BY folder
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive folders | %w", err)
	}
	defer secretRows.Close()

	for secretRows.Next() {
		var folder string
		var count int
		if err := secretRows.Scan(&folder, &count); err != nil {
			return nil, err
		}
		counts[folder] = count
	}
	if err := secretRows.Err(); err != nil {
		return nil, err
	}

	// Explicitly created folders: included with count 0 when not already seen.
	folderRows, err := db.Query(`SELECT name FROM folders`)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive folders | %w", err)
	}
	defer folderRows.Close()

	for folderRows.Next() {
		var name string
		if err := folderRows.Scan(&name); err != nil {
			return nil, err
		}
		if _, ok := counts[name]; !ok {
			counts[name] = 0
		}
	}
	if err := folderRows.Err(); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	folders := make([]FolderInfo, 0, len(names))
	for _, name := range names {
		folders = append(folders, FolderInfo{Folder: name, SecretCount: counts[name]})
	}

	return folders, nil
}

// =====================================================
// Token Cleanup
// =====================================================

// ~~~ PurgeExpiredTokens ~~~
// deletes all tokens whose expires_at is in the past
func (v *Vault) PurgeExpiredTokens() (int64, error) {
	v.mu.RLock()
	db := v.db
	v.mu.RUnlock()

	res, err := db.Exec(`
		DELETE FROM api_tokens
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired tokens | %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ~~~ StartTokenCleanup ~~~
// runs PurgeExpiredTokens on the given interval until ctx is cancelled
func (v *Vault) StartTokenCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	l.LogAddInfo(
		l.Logger.Debug,
		"token cleanup scheduler started",
		"interval", interval,
	)

	for {
		select {
		case <-ticker.C:
			if !v.IsUnlocked() {
				l.Logger.Debug("token cleanup skipped: vault not unlocked")
				continue
			}
			n, err := v.PurgeExpiredTokens()
			if err != nil {
				l.LogError(
					l.Logger.Error,
					"token cleanup failed", "err", err)
				continue
			}
			if n > 0 {
				l.LogAddInfo(l.Logger.Info, "purged expired tokens", "count", n)
			} else {
				l.Logger.Debug("token cleanup ran: no expired tokens found")
			}
		case <-ctx.Done():
			l.Logger.Debug("token cleanup scheduler stopped")
			return
		}
	}
}

// =====================================================
// END "Token Cleanup"
// =====================================================

// =====================================================
// END "HTTP Request Runners"
// =====================================================
