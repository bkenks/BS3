package menu

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/enveditor"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

// choices are the valid auth methods in display order.
var choices = []string{"token", "basic"}
var choiceLabels = []string{"Bearer Token", "Basic Auth"}

type SetAuthMethodDialog struct {
	selected  int // index into choices
	statusMsg string
	saved     bool
}

func NewSetAuthMethodDialog() *SetAuthMethodDialog {
	return &SetAuthMethodDialog{}
}

func (m *SetAuthMethodDialog) Init() tea.Cmd { return nil }

func (m *SetAuthMethodDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *SetAuthMethodDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
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
	case "left", "h", "shift+tab":
		if m.selected > 0 {
			m.selected--
		}
	case "right", "l", "tab":
		if m.selected < len(choices)-1 {
			m.selected++
		}
	case "enter":
		method := choices[m.selected]
		if err := enveditor.SetEnvValue(constants.BS3EnvPath, constants.ENV_VAR_BS3_AUTH_METHOD, method); err != nil {
			m.statusMsg = fmt.Sprintf("error saving: %v", err)
			return m, nil
		}
		_ = os.Setenv(constants.ENV_VAR_BS3_AUTH_METHOD, method)
		m.saved = true
		return m, func() tea.Msg { return events.AuthMethodSaved{Method: method} }
	}
	return m, nil
}

func (m *SetAuthMethodDialog) View() string {
	var body string
	if m.saved {
		body = fmt.Sprintf("Auth method set to %q.\n\nesc · back", choices[m.selected])
	} else {
		body = "Choose how the CLI authenticates.\n\n"

		btns := make([]string, len(choices))
		for i, label := range choiceLabels {
			if i == m.selected {
				btns[i] = shared.SelectedButton.Render(label)
			} else {
				btns[i] = shared.UnselectedButton.Render(label)
			}
		}
		body += lipgloss.JoinHorizontal(lipgloss.Top, btns...)

		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\n← → · toggle   enter · save   esc · cancel"
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Set Auth Method"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
