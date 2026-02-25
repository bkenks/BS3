package constants

var (
	// DataPath              = "/data"
	DataPath              = "/Users/briankenkel/Development/bs3data"
	DBFilename            = "vault.db"
	SaltFilename          = "vault_salt"
	MasterKeyHashFilename = "mkey_hash"
	VaultPath             = DataPath + "/" + DBFilename
	SaltPath              = DataPath + "/" + SaltFilename
	MasterKeyHashPath     = DataPath + "/" + MasterKeyHashFilename
	ENV_VAR_API_PORT      = "VAULT_API_PORT"
)
