package menu

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type menuEntry struct {
	title string
	desc  string
	state shared.SessionState
}

var menuEntries = []menuEntry{
	{"Initialize Vault", "Set up the vault with an admin user and master passphrase", shared.StateInitVault},
	{"Open Vault", "Authenticate with username, password, and master passphrase", shared.StateOpenVault},
	{"Set API Token", "Configure the bearer token for API requests", shared.StateSetAPIToken},
	{"Set Server URL", "Configure the BS3 server URL", shared.StateSetServerURL},
	{"Set Username", "Set the username for vault authentication", shared.StateSetUsername},
	{"Set Password", "Set the password for vault authentication", shared.StateSetPassword},
	{"View Secrets", "Browse and manage stored secrets", shared.StateSecretsList},
	{"View Tokens", "Browse and manage API tokens", shared.StateTokensList},
	{"View Users", "Browse and manage users", shared.StateUsersList},
}

type Model struct {
	List list.Model
}

func New() *Model {
	w, h := shared.SizeBuffer()
	items := make([]list.Item, len(menuEntries))
	for i, e := range menuEntries {
		items[i] = Item{title: e.title, description: e.desc}
	}
	l := list.New(items, list.NewDefaultDelegate(), w, h)
	l.Title = "BS3"
	l.SetFilteringEnabled(false)
	return &Model{List: l}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) selectedState() (shared.SessionState, bool) {
	idx := m.List.Index()
	if idx < 0 || idx >= len(menuEntries) {
		return 0, false
	}
	return menuEntries[idx].state, true
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := shared.SizeBuffer()
		m.List.SetSize(w, h)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			if state, ok := m.selectedState(); ok {
				return m, events.CmdSetState(state)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m *Model) View() string { return m.List.View() }
