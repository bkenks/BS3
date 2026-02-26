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

type SetPasswordDialog struct {
	input     textinput.Model
	statusMsg string
	saved     bool
}

func NewSetPasswordDialog() *SetPasswordDialog {
	input := textinput.New()
	input.Placeholder = "password"
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	input.CharLimit = 256

	return &SetPasswordDialog{input: input}
}

func (m *SetPasswordDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *SetPasswordDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *SetPasswordDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		password := m.input.Value()
		if password == "" {
			m.statusMsg = "password is required"
			return m, nil
		}
		if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_PASSWORD, password); err != nil {
			m.statusMsg = fmt.Sprintf("error saving: %v", err)
			return m, nil
		}
		_ = os.Setenv(constants.ENV_VAR_BS3_PASSWORD, password)
		m.saved = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *SetPasswordDialog) View() string {
	var body string
	if m.saved {
		body = "Password saved.\n\nesc · back"
	} else {
		body = "Password\n" + m.input.View()
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · save   esc · cancel"
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Set Password"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
