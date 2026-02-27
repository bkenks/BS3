package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/enveditor"
	"github.com/bkenks/bs3/internal/injector"
	"github.com/bkenks/bs3/internal/tui"
	l "github.com/bkenks/bs3-logger"
	"github.com/joho/godotenv"
)

type format int
type secrets map[string]string

const (
	env format = iota
	plain
)

var (
	// Config
	argSet        = "set"
	argAPIToken   = "apitoken"
	argServerURL  = "serverurl"
	argUsername   = "username"
	argPassword   = "password"
	argAuthMethod = "authmethod"

	// Vault lifecycle
	argTUI       = "tui"
	argInitVault = "initvault"
	argOpenVault = "openvault"

	// Secrets
	argEnvject    = "envject"
	argGet        = "get"
	argStore      = "store"
	argDelete     = "delete"
	argListSecret = "listsecrets"
	argWriteEnv   = "writeenv"
	argRmEnv      = "rmenv"

	// Tokens
	argGenerateToken = "generatetoken"
	argDeleteToken   = "deletetoken"
	argListTokens    = "listtokens"

	// Users
	argAddUser    = "adduser"
	argDeleteUser = "deleteuser"
	argListUsers  = "listusers"
)

// ~~~ printHelp ~~~
func printHelp() {
	fmt.Print(`BS3 - Self-hosted secrets vault CLI

USAGE:
    bs3 <command> [arguments]

VAULT LIFECYCLE:
    tui                                     Launch the interactive TUI
    initvault <username> <password>         Initialize the vault with an admin user
                <master_passphrase>
    openvault <master_passphrase>           Unlock the vault (required after every restart)

SECRETS:
    get <name>                              Fetch and print a secret value
    store <name> <value>                    Store a new secret
    delete <name>                           Delete a secret
    listsecrets                             List all secrets (name, created, updated)
    envject <secret1> [secret2...] --       Fetch secrets and inject them as env vars
                <command> [args...]         into the given command
    writeenv <prefix> <secret1>            Write secrets as KEY=VALUE pairs to
                [secret2...]               /dev/shm/bs3-<prefix>.env (tmpfs, 0600)
    rmenv <prefix>                          Delete /dev/shm/bs3-<prefix>.env

TOKENS:
    generatetoken <name> [ttl_seconds]      Generate a Bearer token (0 = no expiry)
    deletetoken <name>                      Delete a token by name
    listtokens                              List all tokens

USERS:
    adduser <username> <password>           Add a new user
    deleteuser <username>                   Delete a user
    listusers                               List all users

CONFIG:
    set apitoken <value>                    Save API token to bs3.env
    set serverurl <value>                   Save server URL to bs3.env
    set username <value>                    Save username to bs3.env
    set password <value>                    Save password to bs3.env
    set authmethod <token|basic>            Set auth method to bs3.env

FLAGS:
    --help, -h                              Show this help message
`)
}

// ~~~ CLI Package Entrypoint ~~~
func Run(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	}

	mode := args[0]

	switch mode {

	// ─── TUI ─────────────────────────────────────────────────────────────────
	case argTUI:
		if err := godotenv.Load(constants.BS3EnvPath); err != nil {
			l.LogAddInfo(l.Logger.Info,
				"variables not set in bs3.env, using global env",
				"bs3.env", constants.BS3EnvPath)
		}
		baseURL := os.Getenv(constants.ENV_VAR_BS3_URL)
		token := os.Getenv(constants.ENV_VAR_BS3_TOKEN)
		username := os.Getenv(constants.ENV_VAR_BS3_USERNAME)
		password := os.Getenv(constants.ENV_VAR_BS3_PASSWORD)
		authMethod := os.Getenv(constants.ENV_VAR_BS3_AUTH_METHOD)
		if err := tui.Run(baseURL, token, username, password, authMethod); err != nil {
			l.LogError(l.Logger.Error, "tui error", "err", err)
			os.Exit(1)
		}

	// ─── Vault Lifecycle ──────────────────────────────────────────────────────
	case argInitVault:
		usage := fmt.Sprintf("bs3 %s <username> <password> <master_passphrase>", argInitVault)
		username := getArgSafe(args, 1, usage)
		password := getArgSafe(args, 2, usage)
		passphrase := getArgSafe(args, 3, usage)
		client := configureAPIClient()
		if err := client.InitializeVault(username, password, passphrase); err != nil {
			l.LogError(l.Logger.Error, "error initializing vault", "err", err)
			os.Exit(1)
		}
		fmt.Println("vault initialized successfully")

	case argOpenVault:
		passphrase := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <master_passphrase>", argOpenVault))
		openVault(passphrase)

	// ─── Secrets ──────────────────────────────────────────────────────────────
	case argEnvject:
		client := configureAPIClient()
		inject(*client, env, args)

	case argGet:
		name := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <name>", argGet))
		client := configureAPIClient()
		sec, err := client.GetSecret(name)
		if err != nil {
			l.LogError(l.Logger.Error, "error fetching secret", "name", name, "err", err)
			os.Exit(1)
		}
		fmt.Println(sec["secret"])

	case argStore:
		usage := fmt.Sprintf("bs3 %s <name> <value>", argStore)
		name := getArgSafe(args, 1, usage)
		value := getArgSafe(args, 2, usage)
		client := configureAPIClient()
		if err := client.StoreSecret(name, value); err != nil {
			l.LogError(l.Logger.Error, "error storing secret", "name", name, "err", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q stored successfully\n", name)

	case argDelete:
		name := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <name>", argDelete))
		client := configureAPIClient()
		if err := client.DeleteSecret(name); err != nil {
			l.LogError(l.Logger.Error, "error deleting secret", "name", name, "err", err)
			os.Exit(1)
		}
		fmt.Printf("secret %q deleted successfully\n", name)

	case argListSecret:
		client := configureAPIClient()
		secs, err := client.ListSecretsMeta()
		if err != nil {
			l.LogError(l.Logger.Error, "error listing secrets", "err", err)
			os.Exit(1)
		}
		if len(secs) == 0 {
			fmt.Println("no secrets found")
			return
		}
		fmt.Printf("%-30s  %-24s  %s\n", "NAME", "CREATED", "UPDATED")
		for _, s := range secs {
			fmt.Printf("%-30s  %-24s  %s\n", s.Name, s.CreatedAt, s.UpdatedAt)
		}

	case argWriteEnv:
		usage := fmt.Sprintf("bs3 %s <prefix> <secret1> [secret2...]", argWriteEnv)
		prefix := getArgSafe(args, 1, usage)
		if len(args) < 3 {
			l.LogAddInfo(l.Logger.Fatal, "incorrect usage", "usage", usage)
		}
		secretNames := args[2:]
		client := configureAPIClient()
		var sb strings.Builder
		for _, name := range secretNames {
			sec, err := client.GetSecret(name)
			if err != nil {
				l.LogError(l.Logger.Error, "error fetching secret", "name", name, "err", err)
				os.Exit(1)
			}
			sb.WriteString(fmt.Sprintf("%s=%s\n", strings.ToUpper(sec["name"]), sec["secret"]))
		}
		envPath := filepath.Join(constants.DevShmDir, fmt.Sprintf("bs3-%s.env", prefix))
		f, err := os.OpenFile(envPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			l.LogError(l.Logger.Error, "error creating env file", "path", envPath, "err", err)
			os.Exit(1)
		}
		if _, err := f.WriteString(sb.String()); err != nil {
			f.Close()
			l.LogError(l.Logger.Error, "error writing env file", "path", envPath, "err", err)
			os.Exit(1)
		}
		f.Close()
		fmt.Println(envPath)

	case argRmEnv:
		prefix := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <prefix>", argRmEnv))
		envPath := filepath.Join(constants.DevShmDir, fmt.Sprintf("bs3-%s.env", prefix))
		if err := os.Remove(envPath); err != nil {
			l.LogError(l.Logger.Error, "error removing env file", "path", envPath, "err", err)
			os.Exit(1)
		}
		fmt.Printf("removed %s\n", envPath)

	// ─── Tokens ───────────────────────────────────────────────────────────────
	case argGenerateToken:
		usage := fmt.Sprintf("bs3 %s <name> [ttl_seconds]", argGenerateToken)
		name := getArgSafe(args, 1, usage)
		var ttl int64
		if len(args) > 2 {
			parsed, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil || parsed < 0 {
				l.LogAddInfo(l.Logger.Fatal, "invalid ttl", "ttl", args[2], "err", "must be a non-negative integer")
			}
			ttl = parsed
		}
		client := configureAPIClient()
		result, err := client.GenerateToken(name, ttl)
		if err != nil {
			l.LogError(l.Logger.Error, "error generating token", "name", name, "err", err)
			os.Exit(1)
		}
		fmt.Printf("name:    %s\ntoken:   %s\n", result.Name, result.Token)
		if result.ExpiresIn == 0 {
			fmt.Println("expires: never")
		} else {
			fmt.Printf("expires: in %d seconds\n", result.ExpiresIn)
		}

	case argDeleteToken:
		name := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <name>", argDeleteToken))
		client := configureAPIClient()
		if err := client.DeleteToken(name); err != nil {
			l.LogError(l.Logger.Error, "error deleting token", "name", name, "err", err)
			os.Exit(1)
		}
		fmt.Printf("token %q deleted successfully\n", name)

	case argListTokens:
		client := configureAPIClient()
		tkns, err := client.ListTokens()
		if err != nil {
			l.LogError(l.Logger.Error, "error listing tokens", "err", err)
			os.Exit(1)
		}
		if len(tkns) == 0 {
			fmt.Println("no tokens found")
			return
		}
		fmt.Printf("%-30s  %-24s  %s\n", "NAME", "CREATED", "EXPIRES")
		for _, t := range tkns {
			expires := "never"
			if t.ExpiresAt != nil {
				expires = strconv.FormatInt(*t.ExpiresAt, 10)
			}
			fmt.Printf("%-30s  %-24s  %s\n", t.Name, t.CreatedAt, expires)
		}

	// ─── Users ────────────────────────────────────────────────────────────────
	case argAddUser:
		usage := fmt.Sprintf("bs3 %s <username> <password>", argAddUser)
		username := getArgSafe(args, 1, usage)
		password := getArgSafe(args, 2, usage)
		client := configureAPIClient()
		if err := client.AddUser(username, password); err != nil {
			l.LogError(l.Logger.Error, "error adding user", "username", username, "err", err)
			os.Exit(1)
		}
		fmt.Printf("user %q added successfully\n", username)

	case argDeleteUser:
		username := getArgSafe(args, 1, fmt.Sprintf("bs3 %s <username>", argDeleteUser))
		client := configureAPIClient()
		if err := client.DeleteUser(username); err != nil {
			l.LogError(l.Logger.Error, "error deleting user", "username", username, "err", err)
			os.Exit(1)
		}
		fmt.Printf("user %q deleted successfully\n", username)

	case argListUsers:
		client := configureAPIClient()
		usrs, err := client.ListUsers()
		if err != nil {
			l.LogError(l.Logger.Error, "error listing users", "err", err)
			os.Exit(1)
		}
		if len(usrs) == 0 {
			fmt.Println("no users found")
			return
		}
		fmt.Printf("%-30s  %s\n", "USERNAME", "CREATED")
		for _, u := range usrs {
			fmt.Printf("%-30s  %s\n", u.Username, u.CreatedAt)
		}

	// ─── Config ───────────────────────────────────────────────────────────────
	case argSet:
		argToSet := getArgSafe(args, 1,
			fmt.Sprintf("bs3 %s <%s|%s|%s|%s|%s> <value>", argSet, argAPIToken, argServerURL, argUsername, argPassword, argAuthMethod),
		)
		switch argToSet {
		case argAPIToken:
			token := getArgSafe(args, 2, fmt.Sprintf("bs3 %s %s <value>", argSet, argAPIToken))
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_TOKEN, token); err != nil {
				l.LogAddInfo(l.Logger.Fatal, "could not set environment variable",
					"var", constants.ENV_VAR_BS3_TOKEN, "err", err)
			}
		case argServerURL:
			url := getArgSafe(args, 2, fmt.Sprintf("bs3 %s %s <value>", argSet, argServerURL))
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_URL, url); err != nil {
				l.LogAddInfo(l.Logger.Fatal, "could not set environment variable",
					"var", constants.ENV_VAR_BS3_URL, "err", err)
			}
		case argUsername:
			username := getArgSafe(args, 2, fmt.Sprintf("bs3 %s %s <value>", argSet, argUsername))
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_USERNAME, username); err != nil {
				l.LogAddInfo(l.Logger.Fatal, "could not set environment variable",
					"var", constants.ENV_VAR_BS3_USERNAME, "err", err)
			}
		case argPassword:
			password := getArgSafe(args, 2, fmt.Sprintf("bs3 %s %s <value>", argSet, argPassword))
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_PASSWORD, password); err != nil {
				l.LogAddInfo(l.Logger.Fatal, "could not set environment variable",
					"var", constants.ENV_VAR_BS3_PASSWORD, "err", err)
			}
		case argAuthMethod:
			method := getArgSafe(args, 2, fmt.Sprintf("bs3 %s %s <token|basic>", argSet, argAuthMethod))
			if method != "token" && method != "basic" {
				l.LogAddInfo(l.Logger.Fatal, "invalid auth method", "value", method, "valid", "token|basic")
			}
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_AUTH_METHOD, method); err != nil {
				l.LogAddInfo(l.Logger.Fatal, "could not set environment variable",
					"var", constants.ENV_VAR_BS3_AUTH_METHOD, "err", err)
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nRun 'bs3 --help' for usage.\n", mode)
		os.Exit(1)
	}
}

// ~~~ getArgSafe ~~~
func getArgSafe(args []string, arg int, usage string) string {
	if len(args) < arg+1 {
		l.LogAddInfo(l.Logger.Fatal, "incorrect usage", "usage", usage)
		os.Exit(1)
	}
	return args[arg]
}

// ~~~ configureAPIClient ~~~
func configureAPIClient() *apiclient.Client {
	if err := godotenv.Load(constants.BS3EnvPath); err != nil {
		l.LogAddInfo(l.Logger.Info,
			"variables not set in bs3.env, using global env",
			"bs3.env", constants.BS3EnvPath)
	}

	token := os.Getenv(constants.ENV_VAR_BS3_TOKEN)
	baseURL := os.Getenv(constants.ENV_VAR_BS3_URL)
	username := os.Getenv(constants.ENV_VAR_BS3_USERNAME)
	password := os.Getenv(constants.ENV_VAR_BS3_PASSWORD)
	authMethod := os.Getenv(constants.ENV_VAR_BS3_AUTH_METHOD)

	if baseURL == "" {
		l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_URL)
		os.Exit(1)
	}

	if authMethod == "basic" {
		if username == "" {
			l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_USERNAME)
			os.Exit(1)
		}
	} else {
		if token == "" {
			l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_TOKEN)
			os.Exit(1)
		}
	}

	client := apiclient.NewClient(baseURL, token)
	client.Username = username
	client.Password = password
	client.AuthMethod = authMethod
	return client
}

// ~~~ openVault ~~~
// loads credentials from env and calls the /openvault endpoint with basic auth
func openVault(masterPassphrase string) {
	if err := godotenv.Load(constants.BS3EnvPath); err != nil {
		l.LogAddInfo(l.Logger.Info,
			"variables not set in bs3.env, using global env",
			"bs3.env", constants.BS3EnvPath)
	}

	baseURL := os.Getenv(constants.ENV_VAR_BS3_URL)
	username := os.Getenv(constants.ENV_VAR_BS3_USERNAME)
	password := os.Getenv(constants.ENV_VAR_BS3_PASSWORD)

	if baseURL == "" {
		l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_URL)
		os.Exit(1)
	}
	if username == "" {
		l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_USERNAME)
		os.Exit(1)
	}
	if password == "" {
		l.LogError(l.Logger.Error, "variable not set", "var", constants.ENV_VAR_BS3_PASSWORD)
		os.Exit(1)
	}

	client := apiclient.NewClient(baseURL, "")
	body, err := client.OpenVault(username, password, masterPassphrase)
	if err != nil {
		l.LogError(l.Logger.Error, "error opening vault", "err", err)
		os.Exit(1)
	}

	fmt.Println(string(body))
}

// ~~~ inject ~~~
// fetches secrets from the BS3 server and exec's the child command with them injected as env vars
func inject(client apiclient.Client, format format, args []string) {
	args = args[1:]

	cmdIdx := len(args)
	for i, a := range args {
		if a == "--" {
			cmdIdx = i
			break
		}
	}

	secretNames := args[:cmdIdx]

	command := []string{}
	if cmdIdx < len(args)-1 {
		command = args[cmdIdx+1:]
	}

	secretsList := make(secrets)
	for _, name := range secretNames {
		var key string

		sec, err := client.GetSecret(name)
		if err != nil {
			l.LogError(l.Logger.Error, "error fetching secret", "name", name, "err", err)
			os.Exit(1)
		}

		switch format {
		case env:
			key = strings.ToUpper(sec["name"])
		}

		secretsList[key] = sec["secret"]
	}

	if err := injector.Run(secretsList, command); err != nil {
		l.LogError(l.Logger.Error, "Error running command", "err", err)
		os.Exit(1)
	}
}
