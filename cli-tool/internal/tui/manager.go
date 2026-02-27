package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/menu"
	"github.com/bkenks/bs3/internal/tui/ui/secrets"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
	"github.com/bkenks/bs3/internal/tui/ui/tokens"
	"github.com/bkenks/bs3/internal/tui/ui/users"
)

// statusMsgSetter is implemented by dialog models that can display error messages.
type statusMsgSetter interface {
	SetStatusMsg(string)
}

type ModelManager struct {
	state  shared.SessionState
	client apiclient.Client
	active tea.Model

	// Main menu
	mainMenu *menu.Model

	// Menu dialogs
	initVaultDlg      *menu.InitVaultDialog
	openVaultDlg      *menu.OpenVaultDialog
	errDialog         *menu.ErrorDialog
	setTokenDlg       *menu.SetAPITokenDialog
	setURLDlg         *menu.SetServerURLDialog
	setUsernameDlg    *menu.SetUsernameDialog
	setPasswordDlg    *menu.SetPasswordDialog
	setAuthMethodDlg  *menu.SetAuthMethodDialog

	// List models (persistent, hold pointer to m.client)
	secretsList *secrets.Model
	tokensList  *tokens.Model
	usersList   *users.Model

	// Dialog models (recreated on each activation)
	secretView   *secrets.ViewDialog
	secretCreate *secrets.CreateDialog
	secretDelete *secrets.DeleteDialog

	tokenView      *tokens.ViewDialog
	tokenGenerate  *tokens.GenerateDialog
	tokenGenerated *tokens.GeneratedDialog
	tokenDelete    *tokens.DeleteDialog

	userView   *users.ViewDialog
	userAdd    *users.AddDialog
	userDelete *users.DeleteDialog
}

func New(client apiclient.Client) *ModelManager {
	m := &ModelManager{
		client: client,
	}
	// Persistent list models hold a pointer to m.client so they see
	// any in-session credential updates immediately.
	m.secretsList = secrets.New(&m.client)
	m.tokensList = tokens.New(&m.client)
	m.usersList = users.New(&m.client)
	m.mainMenu = menu.New()
	m.state = shared.StateMainMenu
	m.active = m.mainMenu
	return m
}

func (m *ModelManager) Init() tea.Cmd {
	return m.mainMenu.Init()
}

// ─── switchState ─────────────────────────────────────────────────────────────

func (m *ModelManager) switchState(state shared.SessionState) tea.Cmd {
	m.state = state
	switch state {
	case shared.StateMainMenu:
		m.active = m.mainMenu

	case shared.StateSecretsList:
		m.active = m.secretsList
		return m.secretsList.RefreshCmd()

	case shared.StateTokensList:
		m.active = m.tokensList
		return m.tokensList.RefreshCmd()

	case shared.StateUsersList:
		m.active = m.usersList
		return m.usersList.RefreshCmd()

	case shared.StateErrorDialog:
		m.active = m.errDialog // set by APIError handler before calling switchState

	case shared.StateInitVault:
		m.initVaultDlg = menu.NewInitVaultDialog(m.client)
		m.active = m.initVaultDlg
		return m.initVaultDlg.Init()

	case shared.StateOpenVault:
		m.openVaultDlg = menu.NewOpenVaultDialog(m.client)
		m.active = m.openVaultDlg
		return m.openVaultDlg.Init()

	case shared.StateSetAPIToken:
		m.setTokenDlg = menu.NewSetAPITokenDialog()
		m.active = m.setTokenDlg
		return m.setTokenDlg.Init()

	case shared.StateSetServerURL:
		m.setURLDlg = menu.NewSetServerURLDialog()
		m.active = m.setURLDlg
		return m.setURLDlg.Init()

	case shared.StateSetUsername:
		m.setUsernameDlg = menu.NewSetUsernameDialog()
		m.active = m.setUsernameDlg
		return m.setUsernameDlg.Init()

	case shared.StateSetPassword:
		m.setPasswordDlg = menu.NewSetPasswordDialog()
		m.active = m.setPasswordDlg
		return m.setPasswordDlg.Init()

	case shared.StateSetAuthMethod:
		m.setAuthMethodDlg = menu.NewSetAuthMethodDialog()
		m.active = m.setAuthMethodDlg
		return m.setAuthMethodDlg.Init()

	case shared.StateViewSecret:
		if item, ok := m.secretsList.SelectedItem(); ok {
			m.secretView = secrets.NewViewDialog(item.Name, m.client)
			m.active = m.secretView
			return m.secretView.Init()
		}
		m.state = shared.StateSecretsList
		m.active = m.secretsList

	case shared.StateNewSecret:
		m.secretCreate = secrets.NewCreateDialog(m.client)
		m.active = m.secretCreate
		return m.secretCreate.Init()

	case shared.StateDeleteSecret:
		if item, ok := m.secretsList.SelectedItem(); ok {
			m.secretDelete = secrets.NewDeleteDialog(item.Name, m.client)
			m.active = m.secretDelete
		} else {
			m.state = shared.StateSecretsList
			m.active = m.secretsList
		}

	case shared.StateViewToken:
		if item, ok := m.tokensList.SelectedItem(); ok {
			m.tokenView = tokens.NewViewDialog(item)
			m.active = m.tokenView
		} else {
			m.state = shared.StateTokensList
			m.active = m.tokensList
		}

	case shared.StateGenerateToken:
		m.tokenGenerate = tokens.NewGenerateDialog(m.client)
		m.active = m.tokenGenerate
		return m.tokenGenerate.Init()

	case shared.StateTokenGenerated:
		m.active = m.tokenGenerated // set by TokenGenerated event handler before calling switchState

	case shared.StateDeleteToken:
		if item, ok := m.tokensList.SelectedItem(); ok {
			m.tokenDelete = tokens.NewDeleteDialog(item.Name, m.client)
			m.active = m.tokenDelete
		} else {
			m.state = shared.StateTokensList
			m.active = m.tokensList
		}

	case shared.StateViewUser:
		if item, ok := m.usersList.SelectedItem(); ok {
			m.userView = users.NewViewDialog(item)
			m.active = m.userView
		} else {
			m.state = shared.StateUsersList
			m.active = m.usersList
		}

	case shared.StateAddUser:
		m.userAdd = users.NewAddDialog(m.client)
		m.active = m.userAdd
		return m.userAdd.Init()

	case shared.StateDeleteUser:
		if item, ok := m.usersList.SelectedItem(); ok {
			m.userDelete = users.NewDeleteDialog(item.Username, m.client)
			m.active = m.userDelete
		} else {
			m.state = shared.StateUsersList
			m.active = m.usersList
		}
	}
	return nil
}

// ─── Update ──────────────────────────────────────────────────────────────────

func (m *ModelManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Global keys handled before sub-model routing.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// ESC from a list view goes back to the main menu,
		// but only when the list's filter is not active.
		if keyMsg.String() == "esc" {
			switch m.state {
			case shared.StateSecretsList:
				if m.secretsList.List.FilterState() == list.Unfiltered {
					return m, m.switchState(shared.StateMainMenu)
				}
			case shared.StateTokensList:
				if m.tokensList.List.FilterState() == list.Unfiltered {
					return m, m.switchState(shared.StateMainMenu)
				}
			case shared.StateUsersList:
				if m.usersList.List.FilterState() == list.Unfiltered {
					return m, m.switchState(shared.StateMainMenu)
				}
			}
		}
	}

	// Update shared.WindowSize and resize all list models so inactive lists
	// are always correctly sized when the user switches to them.
	if winMsg, ok := msg.(tea.WindowSizeMsg); ok {
		shared.WindowSize = winMsg
		w, h := shared.SizeBuffer()
		m.mainMenu.List.SetSize(w, h)
		m.secretsList.List.SetSize(w, h)
		m.tokensList.List.SetSize(w, h)
		m.usersList.List.SetSize(w, h)
		return m, nil
	}

	// Handle events emitted by sub-models.
	if ev, ok := msg.(events.Event); ok {
		switch ev := ev.(type) {

		case events.SetState:
			cmds = append(cmds, m.switchState(ev.State))
			return m, tea.Batch(cmds...)

		case events.APITokenSaved:
			m.client.Token = ev.Token
			return m, nil

		case events.ServerURLSaved:
			m.client.BaseURL = strings.TrimRight(ev.URL, "/")
			return m, nil

		case events.UsernameSaved:
			m.client.Username = ev.Username
			return m, nil

		case events.PasswordSaved:
			m.client.Password = ev.Password
			return m, nil

		case events.AuthMethodSaved:
			m.client.AuthMethod = ev.Method
			return m, nil

		case events.SecretsRefreshed:
			m.secretsList.SetItems(ev.Secrets)
			return m, nil

		case events.SecretFetched:
			if m.secretView != nil {
				m.secretView.SetValue(ev.Value)
			}
			return m, nil

		case events.SecretStored:
			cmds = append(cmds,
				m.switchState(shared.StateSecretsList),
				m.secretsList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.SecretDeleted:
			cmds = append(cmds,
				m.switchState(shared.StateSecretsList),
				m.secretsList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.TokensRefreshed:
			m.tokensList.SetItems(ev.Tokens)
			return m, nil

		case events.TokenGenerated:
			m.tokenGenerated = tokens.NewGeneratedDialog(ev.Name, ev.Token, ev.ExpiresIn)
			cmds = append(cmds,
				m.switchState(shared.StateTokenGenerated),
				m.tokensList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.TokenDeleted:
			cmds = append(cmds,
				m.switchState(shared.StateTokensList),
				m.tokensList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.UsersRefreshed:
			m.usersList.SetItems(ev.Users)
			return m, nil

		case events.UserAdded:
			cmds = append(cmds,
				m.switchState(shared.StateUsersList),
				m.usersList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.UserDeleted:
			cmds = append(cmds,
				m.switchState(shared.StateUsersList),
				m.usersList.RefreshCmd(),
			)
			return m, tea.Batch(cmds...)

		case events.APIError:
			// Show a dedicated error dialog for top-level flows (vault lifecycle and
			// list entry points). Sub-form dialogs (create, delete, generate, etc.)
			// continue to display errors inline via their status message.
			switch m.state {
			case shared.StateOpenVault, shared.StateInitVault,
				shared.StateSecretsList, shared.StateTokensList, shared.StateUsersList:
				m.errDialog = menu.NewErrorDialog(ev.Err.Error())
				cmds = append(cmds, m.switchState(shared.StateErrorDialog))
				return m, tea.Batch(cmds...)
			default:
				if setter, ok := m.active.(statusMsgSetter); ok {
					setter.SetStatusMsg(ev.Err.Error())
				}
			}
			return m, nil
		}
	}

	// Forward all other messages to the active sub-model.
	var cmd tea.Cmd
	m.active, cmd = m.active.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// ─── View ────────────────────────────────────────────────────────────────────

func (m *ModelManager) View() string {
	// List views and the main menu wrap in DocStyle; dialogs render full-screen themselves.
	switch m.state {
	case shared.StateMainMenu, shared.StateSecretsList, shared.StateTokensList, shared.StateUsersList:
		return shared.DocStyle.Render(m.active.View())
	default:
		return m.active.View()
	}
}

// ─── Run ─────────────────────────────────────────────────────────────────────

func Run(baseURL, token, username, password, authMethod string) error {
	client := apiclient.NewClient(baseURL, token)
	client.Username = username
	client.Password = password
	client.AuthMethod = authMethod
	p := tea.NewProgram(New(*client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
