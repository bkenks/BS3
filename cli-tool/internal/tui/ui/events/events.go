package events

import (
	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

type Event interface{ isEvent() }

type SetState struct{ State shared.SessionState }

func (SetState) isEvent() {}

type SecretsRefreshed struct{ Secrets []apiclient.SecretMeta }

func (SecretsRefreshed) isEvent() {}

type SecretFetched struct{ Value string }

func (SecretFetched) isEvent() {}

type SecretStored struct{}

func (SecretStored) isEvent() {}

type SecretDeleted struct{}

func (SecretDeleted) isEvent() {}

type TokensRefreshed struct{ Tokens []apiclient.TokenMeta }

func (TokensRefreshed) isEvent() {}

type TokenGenerated struct {
	Name      string
	Token     string
	ExpiresIn int64
}

func (TokenGenerated) isEvent() {}

type TokenDeleted struct{}

func (TokenDeleted) isEvent() {}

type UsersRefreshed struct{ Users []apiclient.UserMeta }

func (UsersRefreshed) isEvent() {}

type UserAdded struct{}

func (UserAdded) isEvent() {}

type UserDeleted struct{}

func (UserDeleted) isEvent() {}

type APIError struct{ Err error }

func (APIError) isEvent() {}

type APITokenSaved struct{ Token string }

func (APITokenSaved) isEvent() {}

type ServerURLSaved struct{ URL string }

func (ServerURLSaved) isEvent() {}

func CmdSetState(state shared.SessionState) tea.Cmd {
	return func() tea.Msg { return SetState{State: state} }
}
