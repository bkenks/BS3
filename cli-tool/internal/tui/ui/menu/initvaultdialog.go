package menu

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/constants"
	"github.com/bkenks/bs3-cli/internal/enveditor"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

// initVaultDoneMsg signals a successful /initvault call. persistErr is set if
// writing the new connection settings to the config file failed.
type initVaultDoneMsg struct{ persistErr string }

// Field indices into inputs.
const (
	ivURL = iota
	ivToken
	ivUsername
	ivPassword
	ivPasswordConfirm
	ivPassphrase
	ivPassphraseConfirm
)

type InitVaultDialog struct {
	inputs     []textinput.Model
	focusIdx   int
	statusMsg  string
	done       bool
	persistErr string
}

func NewInitVaultDialog(client apiclient.Client) *InitVaultDialog {
	url := textinput.New()
	url.Placeholder = "http://localhost:8080"
	url.CharLimit = 512
	url.SetValue(client.BaseURL) // prefill if already configured

	token := textinput.New()
	token.Placeholder = "bootstrap API token"
	token.CharLimit = 512
	// Deliberately not prefilled: initializing a vault uses the one-time
	// bootstrap token, which differs from any BS3_API_TOKEN already configured.

	username := textinput.New()
	username.Placeholder = "admin username"
	username.CharLimit = 128

	password := textinput.New()
	password.Placeholder = "admin password (min 8 characters)"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256

	passwordConfirm := textinput.New()
	passwordConfirm.Placeholder = "re-enter password"
	passwordConfirm.EchoMode = textinput.EchoPassword
	passwordConfirm.EchoCharacter = '•'
	passwordConfirm.CharLimit = 256

	passphrase := textinput.New()
	passphrase.Placeholder = "master passphrase (min 12 characters)"
	passphrase.EchoMode = textinput.EchoPassword
	passphrase.EchoCharacter = '•'
	passphrase.CharLimit = 512

	passphraseConfirm := textinput.New()
	passphraseConfirm.Placeholder = "re-enter master passphrase"
	passphraseConfirm.EchoMode = textinput.EchoPassword
	passphraseConfirm.EchoCharacter = '•'
	passphraseConfirm.CharLimit = 512

	return &InitVaultDialog{
		inputs: []textinput.Model{
			url, token, username,
			password, passwordConfirm,
			passphrase, passphraseConfirm,
		},
	}
}

func (m *InitVaultDialog) Init() tea.Cmd { return m.inputs[ivURL].Focus() }

func (m *InitVaultDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *InitVaultDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(initVaultDoneMsg); ok {
		m.done = true
		m.persistErr = done.persistErr
		// Apply the just-entered connection settings to the running session
		// so the rest of the TUI is immediately usable without reconfiguring.
		url := m.inputs[ivURL].Value()
		user := m.inputs[ivUsername].Value()
		pass := m.inputs[ivPassword].Value()
		return m, tea.Batch(
			func() tea.Msg { return events.ServerURLSaved{URL: url} },
			func() tea.Msg { return events.UsernameSaved{Username: user} },
			func() tea.Msg { return events.PasswordSaved{Password: pass} },
			func() tea.Msg { return events.AuthMethodSaved{Method: "basic"} },
		)
	}

	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		var cmds []tea.Cmd
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	if m.done {
		if keyMsg.String() == "esc" {
			return m, events.CmdSetState(shared.StateMainMenu)
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, events.CmdSetState(shared.StateMainMenu)
	case "tab":
		m.focusIdx = (m.focusIdx + 1) % len(m.inputs)
		return m, m.cycleFocus()
	case "shift+tab":
		m.focusIdx = (m.focusIdx - 1 + len(m.inputs)) % len(m.inputs)
		return m, m.cycleFocus()
	case "enter":
		if m.focusIdx < len(m.inputs)-1 {
			m.focusIdx++
			return m, m.cycleFocus()
		}
		serverURL := m.inputs[ivURL].Value()
		token := m.inputs[ivToken].Value()
		username := m.inputs[ivUsername].Value()
		password := m.inputs[ivPassword].Value()
		passphrase := m.inputs[ivPassphrase].Value()
		if serverURL == "" || token == "" || username == "" || password == "" || passphrase == "" ||
			m.inputs[ivPasswordConfirm].Value() == "" || m.inputs[ivPassphraseConfirm].Value() == "" {
			m.statusMsg = "all fields are required"
			return m, nil
		}
		if password != m.inputs[ivPasswordConfirm].Value() {
			m.statusMsg = "passwords do not match"
			return m, nil
		}
		if passphrase != m.inputs[ivPassphraseConfirm].Value() {
			m.statusMsg = "master passphrases do not match"
			return m, nil
		}
		m.statusMsg = "initializing…"
		return m, m.initCmd(serverURL, token, username, password, passphrase)
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
}

func (m *InitVaultDialog) cycleFocus() tea.Cmd {
	var blinkCmd tea.Cmd
	for i := range m.inputs {
		if i == m.focusIdx {
			blinkCmd = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return blinkCmd
}

// initCmd calls /initvault with a client built from the entered server URL and
// bootstrap token, then persists the new connection settings to the config file.
func (m *InitVaultDialog) initCmd(serverURL, token, username, password, passphrase string) tea.Cmd {
	return func() tea.Msg {
		c := apiclient.NewClient(serverURL, token) // bootstrap token → Bearer auth
		if err := c.InitializeVault(username, password, passphrase); err != nil {
			return events.APIError{Err: err}
		}

		// Persist connection settings so future sessions are configured.
		var persistErr string
		settings := [][2]string{
			{constants.ENV_VAR_BS3_URL, c.BaseURL},
			{constants.ENV_VAR_BS3_USERNAME, username},
			{constants.ENV_VAR_BS3_PASSWORD, password},
			{constants.ENV_VAR_BS3_AUTH_METHOD, "basic"},
		}
		for _, s := range settings {
			if err := enveditor.SetEnvValue(constants.BS3EnvPath, s[0], s[1]); err != nil {
				persistErr = err.Error()
				break
			}
		}
		return initVaultDoneMsg{persistErr: persistErr}
	}
}

func (m *InitVaultDialog) View() tea.View {
	var content string
	if m.done {
		lines := []string{
			"Vault initialized successfully.",
			"",
			"The CLI is now configured and saved:",
			"  • Server URL    " + m.inputs[ivURL].Value(),
			"  • Username      " + m.inputs[ivUsername].Value(),
			"  • Auth method   basic",
			"",
			"The bootstrap API token was single-use — it is now",
			"inactive. Future commands authenticate with the admin",
			"username and password above.",
		}
		if m.persistErr != "" {
			lines = append(lines, "", "⚠ could not save config to disk: "+m.persistErr)
		}
		lines = append(lines, "", "esc · back")
		content = strings.Join(lines, "\n")
	} else {
		body := fmt.Sprintf(
			"Server URL\n%s\n\nBootstrap API Token\n%s\n\nAdmin Username\n%s\n\n"+
				"Admin Password\n%s\n\nConfirm Password\n%s\n\n"+
				"Master Passphrase\n%s\n\nConfirm Master Passphrase\n%s",
			m.inputs[ivURL].View(), m.inputs[ivToken].View(), m.inputs[ivUsername].View(),
			m.inputs[ivPassword].View(), m.inputs[ivPasswordConfirm].View(),
			m.inputs[ivPassphrase].View(), m.inputs[ivPassphraseConfirm].View(),
		)
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · next/submit   tab · cycle   esc · cancel"
		content = body
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Initialize Vault"), content),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
