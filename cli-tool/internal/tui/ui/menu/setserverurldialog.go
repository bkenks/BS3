package menu

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/constants"
	"github.com/bkenks/bs3-cli/internal/enveditor"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type SetServerURLDialog struct {
	input     textinput.Model
	statusMsg string
	saved     bool
}

func NewSetServerURLDialog() *SetServerURLDialog {
	input := textinput.New()
	input.Placeholder = "https://bs3.example.com"
	input.CharLimit = 512
	input.SetValue(os.Getenv(constants.ENV_VAR_BS3_URL))

	return &SetServerURLDialog{input: input}
}

func (m *SetServerURLDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *SetServerURLDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *SetServerURLDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		url := m.input.Value()
		if url == "" {
			m.statusMsg = "URL is required"
			return m, nil
		}
		if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_URL, url); err != nil {
			m.statusMsg = fmt.Sprintf("error saving: %v", err)
			return m, nil
		}
		m.saved = true
		return m, func() tea.Msg { return events.ServerURLSaved{URL: url} }
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *SetServerURLDialog) View() string {
	var body string
	if m.saved {
		body = "Server URL saved.\n\nesc · back"
	} else {
		body = "URL\n" + m.input.View()
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · save   esc · cancel"
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Set Server URL"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
