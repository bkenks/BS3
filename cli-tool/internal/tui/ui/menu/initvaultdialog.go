package menu

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

// initVaultDoneMsg signals a successful /initvault call.
type initVaultDoneMsg struct{}

type InitVaultDialog struct {
	inputs    []textinput.Model
	focusIdx  int
	statusMsg string
	done      bool
	client    apiclient.Client
}

func NewInitVaultDialog(client apiclient.Client) *InitVaultDialog {
	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 128

	password := textinput.New()
	password.Placeholder = "password (min 8 characters)"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256

	passphrase := textinput.New()
	passphrase.Placeholder = "master passphrase (min 12 characters)"
	passphrase.EchoMode = textinput.EchoPassword
	passphrase.EchoCharacter = '•'
	passphrase.CharLimit = 512

	return &InitVaultDialog{
		inputs: []textinput.Model{username, password, passphrase},
		client: client,
	}
}

func (m *InitVaultDialog) Init() tea.Cmd { return m.inputs[0].Focus() }

func (m *InitVaultDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *InitVaultDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(initVaultDoneMsg); ok {
		m.done = true
		return m, nil
	}

	keyMsg, isKey := msg.(tea.KeyMsg)
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
	case "enter":
		if m.focusIdx < len(m.inputs)-1 {
			m.focusIdx++
			return m, m.cycleFocus()
		}
		username := m.inputs[0].Value()
		password := m.inputs[1].Value()
		passphrase := m.inputs[2].Value()
		if username == "" || password == "" || passphrase == "" {
			m.statusMsg = "all fields are required"
			return m, nil
		}
		return m, m.initCmd(username, password, passphrase)
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

func (m *InitVaultDialog) initCmd(username, password, passphrase string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.InitializeVault(username, password, passphrase); err != nil {
			return events.APIError{Err: err}
		}
		return initVaultDoneMsg{}
	}
}

func (m *InitVaultDialog) View() string {
	var content string
	if m.done {
		content = "Vault initialized successfully.\n\nesc · back"
	} else {
		body := fmt.Sprintf(
			"Username\n%s\n\nPassword\n%s\n\nMaster Passphrase\n%s",
			m.inputs[0].View(), m.inputs[1].View(), m.inputs[2].View(),
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
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
