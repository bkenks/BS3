package users

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type ViewDialog struct {
	item Item
}

func NewViewDialog(item Item) *ViewDialog {
	return &ViewDialog{item: item}
}

func (m *ViewDialog) Init() tea.Cmd { return nil }

func (m *ViewDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return m, events.CmdSetState(shared.StateUsersList)
	}
	return m, nil
}

func (m *ViewDialog) View() tea.View {
	body := fmt.Sprintf("Username: %s\nCreated:  %s\n\nesc · close", m.item.Username, m.item.CreatedAt)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("User Details"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
