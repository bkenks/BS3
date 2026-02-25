package constants

import (
	"os"
	"path/filepath"

	l "github.com/bkenks/bs3-logger"
)

var (
	UsrHomeDir           string
	UsrConfigDir         string
	UsrCacheDir          string
	BS3EnvPath           string
	ENV_VAR_BS3_TOKEN    = "BS3_API_TOKEN"
	ENV_VAR_BS3_URL      = "BS3_SERVER_URL"
	ENV_VAR_BS3_USERNAME = "BS3_USERNAME"
	ENV_VAR_BS3_PASSWORD = "BS3_PASSWORD"
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
}
