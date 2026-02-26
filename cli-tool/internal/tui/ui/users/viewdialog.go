package users

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type ViewDialog struct {
	item Item
}

func NewViewDialog(item Item) *ViewDialog {
	return &ViewDialog{item: item}
}

func (m *ViewDialog) Init() tea.Cmd { return nil }

func (m *ViewDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return m, events.CmdSetState(shared.StateUsersList)
	}
	return m, nil
}

func (m *ViewDialog) View() string {
	body := fmt.Sprintf("Username: %s\nCreated:  %s\n\nesc · close", m.item.Username, m.item.CreatedAt)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("User Details"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
