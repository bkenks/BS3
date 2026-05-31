package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	l "github.com/bkenks/bs3-logger"
	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/constants"
	"gopkg.in/yaml.v3"
)

// sidecarConfig maps an output env-filename to the list of secret refs
// (folder.name) that should be written into it. Example YAML:
//
//	app.env:
//	  - myapp.MYAPP_DB_URL
//	  - myapp.MYAPP_API_KEY
//	db.env:
//	  - db.POSTGRES_PASSWORD
//
// Each entry becomes one KEY=VALUE env file in the output directory; consumers
// subpath-mount just the file they need from the shared volume.
type sidecarConfig map[string][]string

// runSidecar is the entrypoint for `bs3 sidecar`. It reads the YAML config
// (default /config/sidecar.yml), fetches every referenced secret once, and
// writes one env file per config entry into the output dir (default /out, a
// shared tmpfs volume). It is init-once: it writes everything and exits 0 so a
// consumer can gate on `depends_on: { condition: service_completed_successfully }`.
func runSidecar(args []string) {
	// Config path: positional arg > BS3_SIDECAR_CONFIG > default.
	configPath := os.Getenv(constants.ENV_VAR_BS3_SIDECAR_CONFIG)
	if configPath == "" {
		configPath = constants.SidecarConfigDefault
	}
	if len(args) > 1 && args[1] != "" {
		configPath = args[1]
	}

	// Output dir: BS3_OUT_DIR > default.
	outDir := os.Getenv(constants.ENV_VAR_BS3_OUT_DIR)
	if outDir == "" {
		outDir = constants.SidecarOutDirDefault
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		l.LogError(l.Logger.Error, "could not read sidecar config", "path", configPath, "err", err)
		os.Exit(1)
	}

	var cfg sidecarConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		l.LogError(l.Logger.Error, "could not parse sidecar config", "path", configPath, "err", err)
		os.Exit(1)
	}
	if len(cfg) == 0 {
		l.LogAddInfo(l.Logger.Fatal, "sidecar config has no entries", "path", configPath)
		os.Exit(1)
	}

	if err := os.MkdirAll(outDir, 0700); err != nil {
		l.LogError(l.Logger.Error, "could not create output dir", "dir", outDir, "err", err)
		os.Exit(1)
	}

	client := configureAPIClient()

	// Deterministic order so logs read consistently across runs.
	filenames := make([]string, 0, len(cfg))
	for name := range cfg {
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		refs := cfg[filename]
		if len(refs) == 0 {
			l.LogAddInfo(l.Logger.Warn, "sidecar entry has no secrets, skipping", "file", filename)
			continue
		}
		writeEnvFile(client, outDir, filename, refs)
	}

	fmt.Printf("bs3 sidecar: wrote %d env file(s) to %s\n", len(filenames), outDir)
}

// writeEnvFile fetches each ref and writes a KEY=VALUE env file at
// <outDir>/<filename>. The key is the secret's name upper-cased (matching the
// `envject`/`writeenv` convention); the file is mode 0600. The file is written
// atomically (temp + rename) so a consumer never reads a half-written file.
func writeEnvFile(client *apiclient.Client, outDir, filename string, refs []string) {
	var sb strings.Builder
	for _, ref := range refs {
		folder, name := parseSecretRef(ref)
		sec, err := client.GetSecret(name, folder)
		if err != nil {
			l.LogError(l.Logger.Error, "error fetching secret",
				"file", filename, "name", name, "folder", folder, "err", err)
			os.Exit(1)
		}
		sb.WriteString(fmt.Sprintf("%s=%s\n", strings.ToUpper(sec["name"]), sec["secret"]))
	}

	dest := filepath.Join(outDir, filename)
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0600); err != nil {
		l.LogError(l.Logger.Error, "error writing env file", "path", tmp, "err", err)
		os.Exit(1)
	}
	if err := os.Rename(tmp, dest); err != nil {
		l.LogError(l.Logger.Error, "error finalizing env file", "path", dest, "err", err)
		os.Exit(1)
	}
	fmt.Printf("  %s  (%d secret(s))\n", dest, len(refs))
}
