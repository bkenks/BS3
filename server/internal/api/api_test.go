package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/vault"
)

// newTestServer spins up the full API handler stack backed by a freshly
// initialized vault in a temp directory. The constants package stores its
// filesystem paths in reassignable package-level vars, so the test redirects
// them at a unique temp dir and restores them on cleanup.
//
// It returns the running httptest.Server and a helper that issues an
// authenticated request (Basic Auth as the admin user created at init).
func newTestServer(t *testing.T) (*httptest.Server, func(method, path, body string) *http.Response) {
	t.Helper()

	dir := t.TempDir()
	origData, origVault := constants.DataPath, constants.VaultPath
	origSalt, origMKey := constants.SaltPath, constants.MasterKeyHashPath
	constants.DataPath = dir
	constants.VaultPath = filepath.Join(dir, constants.DBFilename)
	constants.SaltPath = filepath.Join(dir, constants.SaltFilename)
	constants.MasterKeyHashPath = filepath.Join(dir, constants.MasterKeyHashFilename)
	t.Cleanup(func() {
		constants.DataPath, constants.VaultPath = origData, origVault
		constants.SaltPath, constants.MasterKeyHashPath = origSalt, origMKey
	})

	var v vault.Vault
	if err := v.InitializeVault("admin", "password123", "master-passphrase-1234"); err != nil {
		t.Fatalf("InitializeVault failed: %v", err)
	}
	t.Cleanup(func() { v.Close() })

	srv := &Server{Vault: &v}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	do := func(method, path, body string) *http.Response {
		t.Helper()
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, ts.URL+path, rdr)
		if err != nil {
			t.Fatalf("building request: %v", err)
		}
		req.SetBasicAuth("admin", "password123")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %s %s failed: %v", method, path, err)
		}
		return resp
	}
	return ts, do
}

func TestFolderEndpointsEndToEnd(t *testing.T) {
	_, do := newTestServer(t)

	// Store a secret with a folder.
	if resp := do("POST", "/store", `{"name":"k1","secret":"v1","folder":"prod"}`); resp.StatusCode != http.StatusCreated {
		t.Fatalf("/store with folder: status = %d, want 201", resp.StatusCode)
	}
	// Store a secret with no folder (root).
	if resp := do("POST", "/store", `{"name":"k2","secret":"v2"}`); resp.StatusCode != http.StatusCreated {
		t.Fatalf("/store without folder: status = %d, want 201", resp.StatusCode)
	}

	// Storing the same (name, folder) again is a 409 conflict.
	if resp := do("POST", "/store", `{"name":"k1","secret":"dup","folder":"prod"}`); resp.StatusCode != http.StatusConflict {
		t.Errorf("/store duplicate (name,folder): status = %d, want 409", resp.StatusCode)
	}

	// The same name in a different folder is allowed.
	if resp := do("POST", "/store", `{"name":"k1","secret":"v1-staging","folder":"staging"}`); resp.StatusCode != http.StatusCreated {
		t.Fatalf("/store same name different folder: status = %d, want 201", resp.StatusCode)
	}

	// /listsecrets must report each secret's folder.
	resp := do("GET", "/listsecrets", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/listsecrets: status = %d, want 200", resp.StatusCode)
	}
	var secrets []struct{ Name, Folder string }
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		t.Fatalf("decoding /listsecrets: %v", err)
	}
	resp.Body.Close()
	if len(secrets) != 3 {
		t.Errorf("/listsecrets count = %d, want 3", len(secrets))
	}

	// /listfolders must group and count.
	resp = do("GET", "/listfolders", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/listfolders: status = %d, want 200", resp.StatusCode)
	}
	var groups []struct {
		Folder      string `json:"folder"`
		SecretCount int    `json:"secret_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decoding /listfolders: %v", err)
	}
	resp.Body.Close()
	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Folder] = g.SecretCount
	}
	if counts["prod"] != 1 || counts["staging"] != 1 || counts[""] != 1 {
		t.Errorf("/listfolders counts = %v, want prod:1 staging:1 root:1", counts)
	}

	// /createfolder creates an empty, persistent folder.
	if resp := do("POST", "/createfolder", `{"folder":"empties"}`); resp.StatusCode != http.StatusCreated {
		t.Errorf("/createfolder: status = %d, want 201", resp.StatusCode)
	}
	// A second identical /createfolder is a 409 conflict.
	if resp := do("POST", "/createfolder", `{"folder":"empties"}`); resp.StatusCode != http.StatusConflict {
		t.Errorf("/createfolder duplicate: status = %d, want 409", resp.StatusCode)
	}
	// An empty folder name is a 400.
	if resp := do("POST", "/createfolder", `{"folder":"   "}`); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("/createfolder empty name: status = %d, want 400", resp.StatusCode)
	}

	// /listfolders must now include the empty folder with secret_count 0.
	resp = do("GET", "/listfolders", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/listfolders after createfolder: status = %d, want 200", resp.StatusCode)
	}
	groups = nil
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decoding /listfolders: %v", err)
	}
	resp.Body.Close()
	counts = map[string]int{}
	seen := map[string]bool{}
	for _, g := range groups {
		counts[g.Folder] = g.SecretCount
		seen[g.Folder] = true
	}
	if !seen["empties"] {
		t.Errorf("/listfolders missing empty folder %q: %v", "empties", groups)
	}
	if counts["empties"] != 0 {
		t.Errorf("/listfolders empties count = %d, want 0", counts["empties"])
	}

	// /editsecret updates an existing secret's value.
	if resp := do("POST", "/editsecret", `{"name":"k1","folder":"prod","secret":"v1-edited"}`); resp.StatusCode != http.StatusOK {
		t.Errorf("/editsecret: status = %d, want 200", resp.StatusCode)
	}
	// /editsecret on a missing secret is a 404.
	if resp := do("POST", "/editsecret", `{"name":"ghost","folder":"prod","secret":"x"}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("/editsecret missing secret: status = %d, want 404", resp.StatusCode)
	}

	// /get with the folder query param returns the folder-scoped value.
	resp = do("GET", "/get?name=k1&folder=prod", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/get k1 prod: status = %d, want 200", resp.StatusCode)
	}
	var got struct{ Name, Folder, Secret string }
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decoding /get: %v", err)
	}
	resp.Body.Close()
	if got.Folder != "prod" {
		t.Errorf("/get folder = %q, want %q", got.Folder, "prod")
	}
	if got.Secret != "v1-edited" {
		t.Errorf("/get secret = %q, want %q (edited)", got.Secret, "v1-edited")
	}

	// /get for a name in a folder it does not occupy is a 404.
	if resp := do("GET", "/get?name=k1&folder=nonexistent", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("/get wrong folder: status = %d, want 404", resp.StatusCode)
	}

	// The staging copy of k1 is independent and unedited.
	resp = do("GET", "/get?name=k1&folder=staging", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/get k1 staging: status = %d, want 200", resp.StatusCode)
	}
	got = struct{ Name, Folder, Secret string }{}
	json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got.Secret != "v1-staging" {
		t.Errorf("/get k1 staging secret = %q, want %q", got.Secret, "v1-staging")
	}

	// /movesecret relocates k2 from root into a folder.
	if resp := do("POST", "/movesecret", `{"name":"k2","from_folder":"","to_folder":"archive"}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("/movesecret: status = %d, want 200", resp.StatusCode)
	}
	resp = do("GET", "/listsecrets", "")
	secrets = nil
	json.NewDecoder(resp.Body).Decode(&secrets)
	resp.Body.Close()
	for _, s := range secrets {
		if s.Name == "k2" && s.Folder != "archive" {
			t.Errorf("k2 folder after move = %q, want %q", s.Folder, "archive")
		}
	}

	// Moving a missing secret is a 404.
	if resp := do("POST", "/movesecret", `{"name":"ghost","from_folder":"","to_folder":"x"}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("/movesecret missing secret: status = %d, want 404", resp.StatusCode)
	}

	// Moving k1 from prod onto the occupied (k1, staging) is a 409 conflict.
	if resp := do("POST", "/movesecret", `{"name":"k1","from_folder":"prod","to_folder":"staging"}`); resp.StatusCode != http.StatusConflict {
		t.Errorf("/movesecret onto occupied folder: status = %d, want 409", resp.StatusCode)
	}

	// /delete with the folder query param removes only the folder-scoped copy.
	if resp := do("DELETE", "/delete?name=k1&folder=staging", ""); resp.StatusCode != http.StatusNoContent {
		t.Errorf("/delete k1 staging: status = %d, want 204", resp.StatusCode)
	}
	// The prod copy of k1 survives.
	if resp := do("GET", "/get?name=k1&folder=prod", ""); resp.StatusCode != http.StatusOK {
		t.Errorf("/get k1 prod after deleting staging: status = %d, want 200", resp.StatusCode)
	}
	// Deleting a missing secret is a 404.
	if resp := do("DELETE", "/delete?name=k1&folder=staging", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("/delete missing secret: status = %d, want 404", resp.StatusCode)
	}

	// A folder with control characters is rejected.
	if resp := do("POST", "/store", `{"name":"k3","secret":"v3","folder":"bad\tfolder"}`); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("/store with control-char folder: status = %d, want 400", resp.StatusCode)
	}
}
