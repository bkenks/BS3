package users

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type Model struct {
	List   list.Model
	client *apiclient.Client
}

func New(client *apiclient.Client) *Model {
	w, h := shared.SizeBuffer()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), w, h)
	l.Title = "BS3 Users"
	l.AdditionalShortHelpKeys = constants.UserListKeyMap.HelpBinds(constants.Short)
	l.AdditionalFullHelpKeys = constants.UserListKeyMap.HelpBinds(constants.Full)
	return &Model{List: l, client: client}
}

func (m *Model) Init() tea.Cmd { return m.RefreshCmd() }

func (m *Model) RefreshCmd() tea.Cmd {
	return func() tea.Msg {
		usrs, err := m.client.ListUsers()
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.UsersRefreshed{Users: usrs}
	}
}

func (m *Model) SetItems(usrs []apiclient.UserMeta) {
	items := make([]list.Item, len(usrs))
	for i, u := range usrs {
		items[i] = Item{Username: u.Username, CreatedAt: u.CreatedAt}
	}
	m.List.SetItems(items)
}

func (m *Model) SelectedItem() (Item, bool) {
	item, ok := m.List.SelectedItem().(Item)
	return item, ok
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := shared.SizeBuffer()
		m.List.SetSize(w, h)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, constants.UserListKeyMap.Select):
			if _, ok := m.SelectedItem(); ok {
				return m, events.CmdSetState(shared.StateViewUser)
			}
		case key.Matches(msg, constants.UserListKeyMap.AddUser):
			return m, events.CmdSetState(shared.StateAddUser)
		case key.Matches(msg, constants.UserListKeyMap.DeleteUser):
			if _, ok := m.SelectedItem(); ok {
				return m, events.CmdSetState(shared.StateDeleteUser)
			}
		}
	}
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m *Model) View() string { return m.List.View() }
