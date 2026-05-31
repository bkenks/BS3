package constants

import (
	"os"
	"path/filepath"

	l "github.com/bkenks/bs3-logger"
)

var (
	UsrHomeDir              string
	UsrConfigDir            string
	UsrCacheDir             string
	BS3EnvPath              string
	DevShmDir               = "/dev/shm"
	ENV_VAR_BS3_TOKEN       = "BS3_API_TOKEN"
	ENV_VAR_BS3_URL         = "BS3_SERVER_URL"
	ENV_VAR_BS3_USERNAME    = "BS3_USERNAME"
	ENV_VAR_BS3_PASSWORD    = "BS3_PASSWORD"
	ENV_VAR_BS3_AUTH_METHOD = "BS3_AUTH_METHOD"

	// ENV_VAR_BS3_ENV_FILE overrides where the CLI looks for bs3.env. When set,
	// it takes precedence over the default ~/.config/bs3/bs3.env path. This lets
	// the sidecar image expect bs3.env at a clean, predefined location (e.g.
	// /env/bs3.env) instead of replicating the host's config path.
	ENV_VAR_BS3_ENV_FILE = "BS3_ENV_FILE"

	// Sidecar defaults (overridable via env). The sidecar reads a YAML config
	// that maps output filenames to lists of secret refs, and writes one env
	// file per entry into the output directory (a shared tmpfs volume).
	ENV_VAR_BS3_SIDECAR_CONFIG = "BS3_SIDECAR_CONFIG"
	ENV_VAR_BS3_OUT_DIR        = "BS3_OUT_DIR"
	SidecarConfigDefault       = "/config/sidecar.yml"
	SidecarOutDirDefault       = "/out"
)

func init() {
	var err error

	// ~~~ UsrHomeDir ~~~
	if UsrHomeDir, err = os.UserHomeDir(); err != nil {
		l.LogError(
			l.Logger.Error,
			"could not get user home directory |", "err", err)
	}

	// ~~~ UsrConfigDir ~~~
	if UsrConfigDir, err = os.UserConfigDir(); err != nil {
		l.LogError(
			l.Logger.Error,
			"could not get user config directory |", "err", err)
	}

	// ~~~ UsrCacheDir ~~~
	if UsrCacheDir, err = os.UserCacheDir(); err != nil {
		l.LogError(
			l.Logger.Error,
			"could not get user cache directory |", "err", err)
	}

	// ~~~ BS3EnvPath ~~~
	if UsrConfigDir != "" {
		BS3EnvPath = filepath.Join(UsrConfigDir, "bs3/bs3.env")
	}

	// An explicit BS3_ENV_FILE overrides the default location entirely.
	if override := os.Getenv(ENV_VAR_BS3_ENV_FILE); override != "" {
		BS3EnvPath = override
	}
}
