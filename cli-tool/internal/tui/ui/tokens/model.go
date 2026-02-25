package tokens

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/constants"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type Model struct {
	List   list.Model
	client *apiclient.Client
}

func New(client *apiclient.Client) *Model {
	w, h := shared.SizeBuffer()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), w, h)
	l.Title = "BS3 API Tokens"
	l.AdditionalShortHelpKeys = constants.TokenListKeyMap.HelpBinds(constants.Short)
	l.AdditionalFullHelpKeys = constants.TokenListKeyMap.HelpBinds(constants.Full)
	return &Model{List: l, client: client}
}

func (m *Model) Init() tea.Cmd { return m.RefreshCmd() }

func (m *Model) RefreshCmd() tea.Cmd {
	return func() tea.Msg {
		tkns, err := m.client.ListTokens()
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.TokensRefreshed{Tokens: tkns}
	}
}

func (m *Model) SetItems(tkns []apiclient.TokenMeta) {
	items := make([]list.Item, len(tkns))
	for i, t := range tkns {
		items[i] = Item{Name: t.Name, CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt}
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
		case key.Matches(msg, constants.TokenListKeyMap.Select):
			if _, ok := m.SelectedItem(); ok {
				return m, events.CmdSetState(shared.StateViewToken)
			}
		case key.Matches(msg, constants.TokenListKeyMap.NewToken):
			return m, events.CmdSetState(shared.StateGenerateToken)
		case key.Matches(msg, constants.TokenListKeyMap.DeleteToken):
			if _, ok := m.SelectedItem(); ok {
				return m, events.CmdSetState(shared.StateDeleteToken)
			}
		}
	}
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m *Model) View() string { return m.List.View() }
