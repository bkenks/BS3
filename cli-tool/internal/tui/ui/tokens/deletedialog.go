package tokens

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/constants"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type DeleteDialog struct {
	name      string
	statusMsg string
	client    apiclient.Client
}

func NewDeleteDialog(name string, client apiclient.Client) *DeleteDialog {
	return &DeleteDialog{name: name, client: client}
}

func (m *DeleteDialog) Init() tea.Cmd           { return nil }
func (m *DeleteDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *DeleteDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, constants.ConfirmKeyMap.Proceed):
		return m, m.deleteCmd()
	case key.Matches(keyMsg, constants.ConfirmKeyMap.Exit):
		return m, events.CmdSetState(shared.StateTokensList)
	}
	return m, nil
}

func (m *DeleteDialog) deleteCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteToken(m.name); err != nil {
			return events.APIError{Err: err}
		}
		return events.TokenDeleted{}
	}
}

func (m *DeleteDialog) View() string {
	body := fmt.Sprintf("Delete token %q?", m.name)
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nctrl+p · confirm   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Delete Token"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
