package vault

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	l "github.com/bkenks/bs3-logger"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/cryptoutil"
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
	v.state = state
}

func (v *Vault) SetMasterKey(masterKey []byte) {
	v.masterKey = masterKey
}

func (v *Vault) GetMasterKey() []byte {
	return v.masterKey
}

func (v *Vault) GetDB() *sql.DB {
	return v.db
}

// --- END "Getters/Setters" ---

// ~~~ IsInitialized ~~~
// checks state to determine if database is initialized
func (v *Vault) IsInitialized() bool {
	isInitialized := (v.state != Uninitialized)
	return isInitialized
}

func (v *Vault) IsUnlocked() bool {
	return v.state == Unlocked
}

// ~~~ ConnectDB ~~~
// connects the database to the vault struct
func (v *Vault) ConnectDB() error {
	// Connect to DB
	db, err := sql.Open("sqlite", constants.VaultPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database | %w", err)
	}

	v.db = db
	return nil
}

// ~~~ CheckVaultState ~~~
// checks if a vault exists on startup, if so sets state to locked and connects to db
func (v *Vault) CheckVaultState() error {
	var err error

	_, err = os.Stat(constants.VaultPath)
	vaultExists := (err == nil)

	if vaultExists {
		if v.salt, err = cryptoutil.LoadSalt(); err != nil {
			return fmt.Errorf("db exists, but salt missing | %w", err)
		}
		v.ConnectDB()
		v.state = Locked
	} else {
		v.state = Uninitialized
	}

	return nil
}

// ~~~ StoreMasterKeyHash ~~~
// stores the hashed master key in the database for verification when opening/unsealing
func (v *Vault) StoreMasterKeyHash() error {
	if v.db == nil || v.masterKey == nil {
		return fmt.Errorf("vault not initialized")
	}

	hash := cryptoutil.HashMasterKey(v.masterKey)

	// Create a table for the master key hash if it doesn't exist
	_, err := v.db.Exec(`
		CREATE TABLE IF NOT EXISTS vault_meta (
			id INTEGER PRIMARY KEY,
			master_hash BLOB NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create vault_meta table | %w", err)
	}

	// Insert or replace the master key hash (only one row)
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

// ~~~ Close ~~~
// safely closes the connected to the database
func (v *Vault) Close() error {
	if v.db != nil {
		v.db.Close()
	}

	return nil
}

// =====================================================
// END "Vault Helpers"
// =====================================================

// ~~~ InitializeVault ~~~
// creates new database, masterkey, username, and password
func (v *Vault) InitializeVault(username, password, masterPassphrase string) error {

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

	v.ConnectDB()

	// Ping DB to verify connection
	if err := v.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database | %w", err)
	}

	// Create secrets table
	_, err = v.db.Exec(`
		CREATE TABLE IF NOT EXISTS secrets (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL,
            encrypted_dek BLOB NOT NULL,
            encrypted_data BLOB NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

	// Store DB and master key in memory
	v.masterKey = masterKey
	v.state = Unlocked

	v.StoreMasterKeyHash()

	return nil
}

// ~~~ OpenVault ~~~
// connects to the DB and returns a Vault instance with master key
func (v *Vault) OpenVault(masterPassphrase string) error {
	var salt []byte
	var masterKey []byte
	var err error

	if salt, err = cryptoutil.LoadSalt(); err != nil {
		return err
	}

	if masterKey, err = cryptoutil.DeriveMasterKey(masterPassphrase, salt); err != nil {
		return err
	}

	masterKeyMatches, err := v.VerifyMasterKey(masterKey)
	if err != nil {
		return fmt.Errorf("could not verify master key | %v", err)
	}
	if masterKeyMatches == false {
		return fmt.Errorf("master key does not match")
	}

	v.ConnectDB()

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
	if v.db == nil {
		return false, fmt.Errorf("vault not initialized or does not exist")
	}

	// Query Database for password hash given the username
	var storedHash string
	err := v.db.QueryRow(`
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

	if passwordsMatch == false {
		return false, fmt.Errorf("invalid credentials")
	}

	return true, nil
}

// ~~~ VerifyMasterKey ~~~
// compares an input master key with the stored hash
func (v *Vault) VerifyMasterKey(inputKey []byte) (bool, error) {
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

// =====================================================
// END "Verification"
// =====================================================

// =====================================================
// HTTP Request Runners
// =====================================================
// functions that run when with http requests

// ~~~ AddUser ~~~
// hashes password and inserts a new user into the database
func (v *Vault) AddUser(username, password string) error {
	passwordSalt, err := cryptoutil.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate password salt | %w", err)
	}
	passwordHash, err := cryptoutil.HashPassword(password, passwordSalt)
	if err != nil {
		return fmt.Errorf("failed to hash password | %w", err)
	}

	_, err = v.db.Exec(`
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
	var count int
	if err := v.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("failed to count users | %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last user")
	}

	res, err := v.db.Exec(`DELETE FROM users WHERE username = ?`, username)
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
// encrypts secret and stores in database
func (v *Vault) StoreSecret(name string, plaintext []byte) error {
	env, err := cryptoutil.ProtectSecret(v.masterKey, plaintext)
	if err != nil {
		return err
	}

	_, err = v.db.Exec(`
        INSERT INTO secrets (name, encrypted_dek, encrypted_data)
        VALUES (?, ?, ?)
        ON CONFLICT(name) DO UPDATE SET
            encrypted_dek = excluded.encrypted_dek,
            encrypted_data = excluded.encrypted_data,
            updated_at = ?
    `, name, env.EncryptedDEK, env.EncryptedData, time.Now())

	return err
}

// ~~~ GetSecret ~~~
// retreives secret from database and decrypts
func (v *Vault) GetSecret(name string) ([]byte, error) {
	row := v.db.QueryRow(`
		SELECT encrypted_dek, encrypted_data
		FROM secrets
		WHERE name = ?
	`, name)

	var encryptedDEK, encryptedData []byte
	err := row.Scan(&encryptedDEK, &encryptedData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return nil, err
	}

	env := &cryptoutil.Envelope{
		EncryptedDEK:  encryptedDEK,
		EncryptedData: encryptedData,
	}

	plaintext, err := cryptoutil.UnprotectSecret(v.masterKey, env)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret %q | %w", name, err)
	}

	return plaintext, nil
}

// ~~~ DeleteSecret ~~~
// deletes secret from database
func (v *Vault) DeleteSecret(name string) error {
	_, err := v.db.Exec(`DELETE FROM secrets WHERE name = ?`, name)
	return err
}

// ~~~ SecretInfo ~~~
// represents one secret in the database with additional useful information
// does not contain secret or key
type SecretInfo struct {
	Name      string
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
	rows, err := v.db.Query(`
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
	_, err := v.db.Exec(`
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
	rows, err := v.db.Query(`
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
		matches, err := cryptoutil.VerifyToken(v.masterKey, rawToken, hash)
		if err == nil && matches {
			return true, nil
		}
	}
	return false, nil
}

// ~~~ DeleteToken ~~~
// removes a named API token from the database
func (v *Vault) DeleteToken(name string) error {
	res, err := v.db.Exec(`DELETE FROM api_tokens WHERE name = ?`, name)
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
	rows, err := v.db.Query(`
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
	rows, err := v.db.Query(`
		SELECT name, created_at, updated_at
		FROM secrets
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive secrets | %w", err)
	}
	defer rows.Close()

	var secrets []SecretInfo
	for rows.Next() {
		var s SecretInfo
		if err := rows.Scan(&s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}

	return secrets, nil
}

// =====================================================
// Token Cleanup
// =====================================================

// ~~~ PurgeExpiredTokens ~~~
// deletes all tokens whose expires_at is in the past
func (v *Vault) PurgeExpiredTokens() (int64, error) {
	res, err := v.db.Exec(`
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
