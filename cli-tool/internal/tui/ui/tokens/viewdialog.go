package tokens

import (
	"fmt"
	"time"

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
		return m, events.CmdSetState(shared.StateTokensList)
	}
	return m, nil
}

func (m *ViewDialog) View() string {
	exp := "never"
	if m.item.ExpiresAt != nil {
		exp = time.Unix(*m.item.ExpiresAt, 0).Format("2006-01-02 15:04:05")
	}
	body := fmt.Sprintf("Name:    %s\nCreated: %s\nExpires: %s\n\nesc · close",
		m.item.Name, m.item.CreatedAt, exp)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Token Details"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
