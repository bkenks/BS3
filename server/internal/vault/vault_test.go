package vault

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bkenks/bs3/internal/constants"
)

// newTestVault creates a fully initialized, unlocked vault backed by a
// temporary directory. The constants package stores its filesystem paths in
// reassignable package-level vars, so each test can redirect them at a unique
// temp dir and keep itself hermetic. The original values are restored on
// cleanup.
func newTestVault(t *testing.T) *Vault {
	t.Helper()

	dir := t.TempDir()

	origData := constants.DataPath
	origVault := constants.VaultPath
	origSalt := constants.SaltPath
	origMKey := constants.MasterKeyHashPath

	constants.DataPath = dir
	constants.VaultPath = filepath.Join(dir, constants.DBFilename)
	constants.SaltPath = filepath.Join(dir, constants.SaltFilename)
	constants.MasterKeyHashPath = filepath.Join(dir, constants.MasterKeyHashFilename)

	t.Cleanup(func() {
		constants.DataPath = origData
		constants.VaultPath = origVault
		constants.SaltPath = origSalt
		constants.MasterKeyHashPath = origMKey
	})

	v := &Vault{}
	if err := v.InitializeVault("admin", "password123", "master-passphrase-1234"); err != nil {
		t.Fatalf("InitializeVault failed: %v", err)
	}
	t.Cleanup(func() { v.Close() })

	return v
}

// findSecretInFolder returns the SecretInfo for (name, folder), or fails the
// test if absent.
func findSecretInFolder(t *testing.T, secrets []SecretInfo, name, folder string) SecretInfo {
	t.Helper()
	for _, s := range secrets {
		if s.Name == name && s.Folder == folder {
			return s
		}
	}
	t.Fatalf("secret %q in folder %q not found in list", name, folder)
	return SecretInfo{}
}

func TestStoreSecretFolderRoundTrips(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("api-key", []byte("s3cr3t"), "production"); err != nil {
		t.Fatalf("StoreSecret failed: %v", err)
	}

	secrets, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	got := findSecretInFolder(t, secrets, "api-key", "production")
	if got.Folder != "production" {
		t.Errorf("folder = %q, want %q", got.Folder, "production")
	}

	// Verify the plaintext is still retrievable after storing with a folder.
	plain, err := v.GetSecret("api-key", "production")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(plain) != "s3cr3t" {
		t.Errorf("secret value = %q, want %q", string(plain), "s3cr3t")
	}
}

func TestStoreSecretEmptyFolderIsRoot(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("loose-key", []byte("v"), ""); err != nil {
		t.Fatalf("StoreSecret failed: %v", err)
	}

	secrets, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	got := findSecretInFolder(t, secrets, "loose-key", "")
	if got.Folder != "" {
		t.Errorf("folder = %q, want empty (root)", got.Folder)
	}
}

func TestStoreSecretSameNameDifferentFolders(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("db-pass", []byte("prod-value"), "production"); err != nil {
		t.Fatalf("StoreSecret(production) failed: %v", err)
	}
	if err := v.StoreSecret("db-pass", []byte("stage-value"), "staging"); err != nil {
		t.Fatalf("StoreSecret(staging) failed: %v", err)
	}

	prod, err := v.GetSecret("db-pass", "production")
	if err != nil {
		t.Fatalf("GetSecret(production) failed: %v", err)
	}
	if string(prod) != "prod-value" {
		t.Errorf("production value = %q, want %q", string(prod), "prod-value")
	}

	stage, err := v.GetSecret("db-pass", "staging")
	if err != nil {
		t.Fatalf("GetSecret(staging) failed: %v", err)
	}
	if string(stage) != "stage-value" {
		t.Errorf("staging value = %q, want %q", string(stage), "stage-value")
	}

	secrets, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("secret count = %d, want 2", len(secrets))
	}
}

func TestStoreSecretRejectsDuplicate(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("api-key", []byte("first"), "prod"); err != nil {
		t.Fatalf("initial StoreSecret failed: %v", err)
	}

	err := v.StoreSecret("api-key", []byte("second"), "prod")
	if err == nil {
		t.Fatal("StoreSecret of existing (name, folder): expected error, got nil")
	}
	if !errors.Is(err, ErrSecretExists) {
		t.Errorf("error = %v, want ErrSecretExists", err)
	}

	// The original value must be untouched.
	plain, err := v.GetSecret("api-key", "prod")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(plain) != "first" {
		t.Errorf("value after rejected overwrite = %q, want %q", string(plain), "first")
	}
}

func TestEditSecretUpdatesValue(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("api-key", []byte("old"), "prod"); err != nil {
		t.Fatalf("StoreSecret failed: %v", err)
	}

	if err := v.EditSecret("api-key", "prod", []byte("new")); err != nil {
		t.Fatalf("EditSecret failed: %v", err)
	}

	plain, err := v.GetSecret("api-key", "prod")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(plain) != "new" {
		t.Errorf("value after edit = %q, want %q", string(plain), "new")
	}
}

func TestEditSecretMissingErrors(t *testing.T) {
	v := newTestVault(t)

	err := v.EditSecret("does-not-exist", "prod", []byte("v"))
	if err == nil {
		t.Fatal("EditSecret on missing secret: expected error, got nil")
	}
	if !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("error = %v, want ErrSecretNotFound", err)
	}
}

func TestGetSecretIsFolderScoped(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("token", []byte("prod-tok"), "prod"); err != nil {
		t.Fatalf("StoreSecret failed: %v", err)
	}

	// Same name, different folder: must not be found.
	if _, err := v.GetSecret("token", "staging"); err == nil {
		t.Fatal("GetSecret in wrong folder: expected error, got nil")
	} else if !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("error = %v, want ErrSecretNotFound", err)
	}

	plain, err := v.GetSecret("token", "prod")
	if err != nil {
		t.Fatalf("GetSecret(prod) failed: %v", err)
	}
	if string(plain) != "prod-tok" {
		t.Errorf("value = %q, want %q", string(plain), "prod-tok")
	}
}

func TestDeleteSecretIsFolderScoped(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("token", []byte("v1"), "prod"); err != nil {
		t.Fatalf("StoreSecret(prod) failed: %v", err)
	}
	if err := v.StoreSecret("token", []byte("v2"), "staging"); err != nil {
		t.Fatalf("StoreSecret(staging) failed: %v", err)
	}

	// Deleting from a folder where it does not exist errors.
	if err := v.DeleteSecret("token", "nonexistent"); err == nil {
		t.Fatal("DeleteSecret in wrong folder: expected error, got nil")
	} else if !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("error = %v, want ErrSecretNotFound", err)
	}

	// Deleting the prod copy must leave the staging copy intact.
	if err := v.DeleteSecret("token", "prod"); err != nil {
		t.Fatalf("DeleteSecret(prod) failed: %v", err)
	}
	if _, err := v.GetSecret("token", "prod"); err == nil {
		t.Error("GetSecret(prod) after delete: expected error, got nil")
	}
	if _, err := v.GetSecret("token", "staging"); err != nil {
		t.Errorf("GetSecret(staging) after deleting prod: %v", err)
	}
}

func TestListFoldersCounts(t *testing.T) {
	v := newTestVault(t)

	store := func(name, folder string) {
		t.Helper()
		if err := v.StoreSecret(name, []byte("x"), folder); err != nil {
			t.Fatalf("StoreSecret(%q) failed: %v", name, err)
		}
	}
	store("a", "prod")
	store("b", "prod")
	store("c", "staging")
	store("d", "") // root

	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	counts := make(map[string]int)
	for _, f := range folders {
		counts[f.Folder] = f.SecretCount
	}

	if counts["prod"] != 2 {
		t.Errorf("prod count = %d, want 2", counts["prod"])
	}
	if counts["staging"] != 1 {
		t.Errorf("staging count = %d, want 1", counts["staging"])
	}
	if counts[""] != 1 {
		t.Errorf("root count = %d, want 1", counts[""])
	}
	if len(folders) != 3 {
		t.Errorf("folder group count = %d, want 3", len(folders))
	}
}

func TestCreateFolderShowsEmptyInListFolders(t *testing.T) {
	v := newTestVault(t)

	if err := v.CreateFolder("empty-folder"); err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	found := false
	for _, f := range folders {
		if f.Folder == "empty-folder" {
			found = true
			if f.SecretCount != 0 {
				t.Errorf("empty-folder count = %d, want 0", f.SecretCount)
			}
		}
	}
	if !found {
		t.Errorf("empty-folder not present in ListFolders: %v", folders)
	}
}

func TestCreateFolderRejectsDuplicate(t *testing.T) {
	v := newTestVault(t)

	if err := v.CreateFolder("dupe"); err != nil {
		t.Fatalf("initial CreateFolder failed: %v", err)
	}

	err := v.CreateFolder("dupe")
	if err == nil {
		t.Fatal("CreateFolder of existing name: expected error, got nil")
	}
	if !errors.Is(err, ErrFolderExists) {
		t.Errorf("error = %v, want ErrFolderExists", err)
	}
}

func TestListFoldersMergesExplicitAndSecretFolders(t *testing.T) {
	v := newTestVault(t)

	// "shared" exists both explicitly and via a secret.
	if err := v.CreateFolder("shared"); err != nil {
		t.Fatalf("CreateFolder(shared) failed: %v", err)
	}
	if err := v.StoreSecret("a", []byte("x"), "shared"); err != nil {
		t.Fatalf("StoreSecret(shared) failed: %v", err)
	}
	if err := v.StoreSecret("b", []byte("x"), "shared"); err != nil {
		t.Fatalf("StoreSecret(shared) failed: %v", err)
	}
	// "explicit-only" exists only as an explicit folder.
	if err := v.CreateFolder("explicit-only"); err != nil {
		t.Fatalf("CreateFolder(explicit-only) failed: %v", err)
	}
	// "secret-only" exists only via a secret.
	if err := v.StoreSecret("c", []byte("x"), "secret-only"); err != nil {
		t.Fatalf("StoreSecret(secret-only) failed: %v", err)
	}

	folders, err := v.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	counts := make(map[string]int)
	occurrences := make(map[string]int)
	for _, f := range folders {
		counts[f.Folder] = f.SecretCount
		occurrences[f.Folder]++
	}

	if occurrences["shared"] != 1 {
		t.Errorf("shared appears %d times, want 1", occurrences["shared"])
	}
	if counts["shared"] != 2 {
		t.Errorf("shared count = %d, want 2", counts["shared"])
	}
	if counts["explicit-only"] != 0 {
		t.Errorf("explicit-only count = %d, want 0", counts["explicit-only"])
	}
	if counts["secret-only"] != 1 {
		t.Errorf("secret-only count = %d, want 1", counts["secret-only"])
	}
}

// TestMigrateSchemaCreatesFoldersTable builds a database with a pre-folders
// schema (no folders table) and verifies migrateSchema creates it, and that a
// second run is idempotent.
func TestMigrateSchemaCreatesFoldersTable(t *testing.T) {
	dir := t.TempDir()

	origData := constants.DataPath
	origVault := constants.VaultPath
	origSalt := constants.SaltPath
	origMKey := constants.MasterKeyHashPath
	constants.DataPath = dir
	constants.VaultPath = filepath.Join(dir, constants.DBFilename)
	constants.SaltPath = filepath.Join(dir, constants.SaltFilename)
	constants.MasterKeyHashPath = filepath.Join(dir, constants.MasterKeyHashFilename)
	t.Cleanup(func() {
		constants.DataPath = origData
		constants.VaultPath = origVault
		constants.SaltPath = origSalt
		constants.MasterKeyHashPath = origMKey
	})

	// Build a DB with a secrets table already at the composite-unique schema
	// but with no folders table.
	db, err := sql.Open("sqlite", constants.VaultPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE secrets (
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
		t.Fatalf("creating secrets table failed: %v", err)
	}

	v := &Vault{db: db}
	t.Cleanup(func() { v.Close() })

	folderTableExists := func() bool {
		t.Helper()
		var name string
		err := v.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name='folders'`,
		).Scan(&name)
		if err == sql.ErrNoRows {
			return false
		}
		if err != nil {
			t.Fatalf("checking for folders table failed: %v", err)
		}
		return true
	}

	if folderTableExists() {
		t.Fatal("folders table present before migration")
	}

	v.mu.Lock()
	err = v.migrateSchema()
	v.mu.Unlock()
	if err != nil {
		t.Fatalf("migrateSchema failed: %v", err)
	}
	if !folderTableExists() {
		t.Fatal("folders table missing after migration")
	}

	// The new table must be usable.
	if err := v.CreateFolder("post-migration"); err != nil {
		t.Fatalf("CreateFolder after migration failed: %v", err)
	}

	// A second run must be idempotent and not error or drop the row.
	v.mu.Lock()
	err = v.migrateSchema()
	v.mu.Unlock()
	if err != nil {
		t.Fatalf("second migrateSchema run failed: %v", err)
	}
	if !folderTableExists() {
		t.Fatal("folders table missing after second migration")
	}
	var count int
	if err := v.db.QueryRow(`SELECT COUNT(*) FROM folders WHERE name = 'post-migration'`).Scan(&count); err != nil {
		t.Fatalf("counting folders failed: %v", err)
	}
	if count != 1 {
		t.Errorf("post-migration folder count = %d, want 1 (idempotent)", count)
	}
}

func TestMoveSecretChangesFolder(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("token", []byte("v"), "old"); err != nil {
		t.Fatalf("StoreSecret failed: %v", err)
	}

	if err := v.MoveSecret("token", "old", "new"); err != nil {
		t.Fatalf("MoveSecret failed: %v", err)
	}

	secrets, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if got := findSecretInFolder(t, secrets, "token", "new").Folder; got != "new" {
		t.Errorf("folder after move = %q, want %q", got, "new")
	}

	// Moving to an empty folder relocates the secret to root.
	if err := v.MoveSecret("token", "new", ""); err != nil {
		t.Fatalf("MoveSecret to root failed: %v", err)
	}
	secrets, err = v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if got := findSecretInFolder(t, secrets, "token", "").Folder; got != "" {
		t.Errorf("folder after move to root = %q, want empty", got)
	}

	// Moving with from == to is a no-op success.
	if err := v.MoveSecret("token", "", ""); err != nil {
		t.Errorf("MoveSecret no-op (same folder): %v", err)
	}

	// The plaintext survives the moves.
	plain, err := v.GetSecret("token", "")
	if err != nil {
		t.Fatalf("GetSecret after moves failed: %v", err)
	}
	if string(plain) != "v" {
		t.Errorf("value after moves = %q, want %q", string(plain), "v")
	}
}

func TestMoveSecretMissingErrors(t *testing.T) {
	v := newTestVault(t)

	err := v.MoveSecret("does-not-exist", "here", "there")
	if err == nil {
		t.Fatal("MoveSecret on missing secret: expected error, got nil")
	}
	if !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("error = %v, want ErrSecretNotFound", err)
	}
}

func TestMoveSecretOntoOccupiedErrors(t *testing.T) {
	v := newTestVault(t)

	if err := v.StoreSecret("token", []byte("a"), "src"); err != nil {
		t.Fatalf("StoreSecret(src) failed: %v", err)
	}
	if err := v.StoreSecret("token", []byte("b"), "dst"); err != nil {
		t.Fatalf("StoreSecret(dst) failed: %v", err)
	}

	err := v.MoveSecret("token", "src", "dst")
	if err == nil {
		t.Fatal("MoveSecret onto occupied folder: expected error, got nil")
	}
	if !errors.Is(err, ErrSecretExists) {
		t.Errorf("error = %v, want ErrSecretExists", err)
	}

	// Both originals must be unchanged.
	src, err := v.GetSecret("token", "src")
	if err != nil {
		t.Fatalf("GetSecret(src) failed: %v", err)
	}
	if string(src) != "a" {
		t.Errorf("src value = %q, want %q", string(src), "a")
	}
	dst, err := v.GetSecret("token", "dst")
	if err != nil {
		t.Fatalf("GetSecret(dst) failed: %v", err)
	}
	if string(dst) != "b" {
		t.Errorf("dst value = %q, want %q", string(dst), "b")
	}
}

// TestMigrateSchemaRebuildsCompositeUnique builds a database with the OLD
// schema (global UNIQUE(name), folder column present) and verifies that
// migrateSchema rebuilds the secrets table so the same name may exist in
// multiple folders, while preserving the pre-existing row.
func TestMigrateSchemaRebuildsCompositeUnique(t *testing.T) {
	dir := t.TempDir()

	origData := constants.DataPath
	origVault := constants.VaultPath
	origSalt := constants.SaltPath
	origMKey := constants.MasterKeyHashPath
	constants.DataPath = dir
	constants.VaultPath = filepath.Join(dir, constants.DBFilename)
	constants.SaltPath = filepath.Join(dir, constants.SaltFilename)
	constants.MasterKeyHashPath = filepath.Join(dir, constants.MasterKeyHashFilename)
	t.Cleanup(func() {
		constants.DataPath = origData
		constants.VaultPath = origVault
		constants.SaltPath = origSalt
		constants.MasterKeyHashPath = origMKey
	})

	// Build a DB with the OLD schema directly.
	db, err := sql.Open("sqlite", constants.VaultPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE secrets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			folder TEXT NOT NULL DEFAULT '',
			encrypted_dek BLOB NOT NULL,
			encrypted_data BLOB NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("creating old-schema secrets table failed: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO secrets (name, folder, encrypted_dek, encrypted_data)
		VALUES ('legacy', 'prod', X'01', X'02')
	`); err != nil {
		t.Fatalf("inserting legacy row failed: %v", err)
	}

	v := &Vault{db: db}
	t.Cleanup(func() { v.Close() })

	v.mu.Lock()
	err = v.migrateSchema()
	v.mu.Unlock()
	if err != nil {
		t.Fatalf("migrateSchema failed: %v", err)
	}

	// The composite unique constraint must now be present.
	var tableSQL string
	if err := v.db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='secrets'`,
	).Scan(&tableSQL); err != nil {
		t.Fatalf("reading rebuilt schema failed: %v", err)
	}

	// The original row must have survived the rebuild.
	var name, folder string
	if err := v.db.QueryRow(
		`SELECT name, folder FROM secrets WHERE name = 'legacy'`,
	).Scan(&name, &folder); err != nil {
		t.Fatalf("legacy row missing after migration: %v", err)
	}
	if name != "legacy" || folder != "prod" {
		t.Errorf("legacy row = (%q, %q), want (legacy, prod)", name, folder)
	}

	// The same name in a different folder must now be allowed.
	if _, err := v.db.Exec(`
		INSERT INTO secrets (name, folder, encrypted_dek, encrypted_data)
		VALUES ('legacy', 'staging', X'03', X'04')
	`); err != nil {
		t.Fatalf("inserting same name in different folder failed: %v", err)
	}

	// The same (name, folder) pair must still be rejected.
	if _, err := v.db.Exec(`
		INSERT INTO secrets (name, folder, encrypted_dek, encrypted_data)
		VALUES ('legacy', 'prod', X'05', X'06')
	`); err == nil {
		t.Error("inserting duplicate (name, folder): expected error, got nil")
	}

	// migrateSchema must be idempotent: a second run is a no-op.
	v.mu.Lock()
	err = v.migrateSchema()
	v.mu.Unlock()
	if err != nil {
		t.Fatalf("second migrateSchema run failed: %v", err)
	}
}
