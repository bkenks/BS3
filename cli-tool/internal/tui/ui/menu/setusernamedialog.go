package menu

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/enveditor"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type SetUsernameDialog struct {
	input     textinput.Model
	statusMsg string
	saved     bool
}

func NewSetUsernameDialog() *SetUsernameDialog {
	input := textinput.New()
	input.Placeholder = "username"
	input.CharLimit = 128
	input.SetValue(os.Getenv(constants.ENV_VAR_BS3_USERNAME))

	return &SetUsernameDialog{input: input}
}

func (m *SetUsernameDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *SetUsernameDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *SetUsernameDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if m.saved {
		if keyMsg.String() == "esc" {
			return m, events.CmdSetState(shared.StateMainMenu)
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, events.CmdSetState(shared.StateMainMenu)
	case "enter":
		username := m.input.Value()
		if username == "" {
			m.statusMsg = "username is required"
			return m, nil
		}
		if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_USERNAME, username); err != nil {
			m.statusMsg = fmt.Sprintf("error saving: %v", err)
			return m, nil
		}
		_ = os.Setenv(constants.ENV_VAR_BS3_USERNAME, username)
		m.saved = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *SetUsernameDialog) View() string {
	var body string
	if m.saved {
		body = "Username saved.\n\nesc · back"
	} else {
		body = "Username\n" + m.input.View()
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · save   esc · cancel"
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Set Username"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
