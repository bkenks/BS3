package shared

type SessionState int

const (
	StateMainMenu     SessionState = iota
	StateSecretsList
	StateTokensList
	StateViewSecret
	StateNewSecret
	StateDeleteSecret
	StateViewToken
	StateGenerateToken
	StateTokenGenerated
	StateDeleteToken
	StateUsersList
	StateViewUser
	StateAddUser
	StateDeleteUser
	StateInitVault
	StateOpenVault
	StateSetAPIToken
	StateSetServerURL
	StateSetUsername
	StateSetPassword
	StateSetAuthMethod
	StateErrorDialog
)
