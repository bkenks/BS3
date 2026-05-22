package menu

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/enveditor"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type SetAPITokenDialog struct {
	input     textinput.Model
	statusMsg string
	saved     bool
}

func NewSetAPITokenDialog() *SetAPITokenDialog {
	input := textinput.New()
	input.Placeholder = "API token"
	input.CharLimit = 512
	input.SetValue(os.Getenv(constants.ENV_VAR_BS3_TOKEN))

	return &SetAPITokenDialog{input: input}
}

func (m *SetAPITokenDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *SetAPITokenDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *SetAPITokenDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
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
		token := m.input.Value()
		if token == "" {
			m.statusMsg = "token is required"
			return m, nil
		}
		if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_TOKEN, token); err != nil {
			m.statusMsg = fmt.Sprintf("error saving: %v", err)
			return m, nil
		}
		m.saved = true
		return m, func() tea.Msg { return events.APITokenSaved{Token: token} }
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *SetAPITokenDialog) View() tea.View {
	var body string
	if m.saved {
		body = "API token saved.\n\nesc · back"
	} else {
		body = "Token\n" + m.input.View()
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · save   esc · cancel"
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Set API Token"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
