package secrets

import (
	"sort"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

// Model is the secrets browser. It works like a file explorer: the root level
// lists folders followed by ungrouped secrets; entering a folder shows just
// that folder's secrets. GoUp walks back to the root.
type Model struct {
	List   list.Model
	client *apiclient.Client

	all           []apiclient.SecretMeta // every secret, refreshed from the server
	folders       []apiclient.FolderMeta // every folder, refreshed from the server
	currentFolder string                 // folder currently open ("" at root)
	inFolder      bool                   // whether we are inside a folder
}

func New(client *apiclient.Client) *Model {
	w, h := shared.SizeBuffer()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), w, h)
	l.Title = "BS3 Vault"
	l.AdditionalShortHelpKeys = constants.SecretListKeyMap.HelpBinds(constants.Short)
	l.AdditionalFullHelpKeys = constants.SecretListKeyMap.HelpBinds(constants.Full)
	return &Model{List: l, client: client}
}

func (m *Model) Init() tea.Cmd { return m.RefreshCmd() }

func (m *Model) RefreshCmd() tea.Cmd {
	return func() tea.Msg {
		secs, err := m.client.ListSecretsMeta()
		if err != nil {
			return events.APIError{Err: err}
		}
		folders, err := m.client.ListFolders()
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.SecretsRefreshed{Secrets: secs, Folders: folders}
	}
}

// SetItems stores the full secret and folder sets and rebuilds the current
// view, so a refresh keeps the user wherever they were browsing.
func (m *Model) SetItems(secs []apiclient.SecretMeta, folders []apiclient.FolderMeta) {
	m.all = secs
	m.folders = folders
	m.rebuild()
}

// rebuild populates the list for the current navigation level.
func (m *Model) rebuild() {
	var items []list.Item

	if m.inFolder {
		m.List.Title = "BS3 Vault / " + m.currentFolder
		for _, s := range m.all {
			if s.Folder == m.currentFolder {
				items = append(items, Item{
					Name: s.Name, Folder: s.Folder,
					CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
				})
			}
		}
		m.List.SetItems(items)
		return
	}

	// Root level: folder entries first, then ungrouped secrets.
	// Folder entries come from the server's folder list so that
	// explicitly-created empty folders still appear.
	m.List.Title = "BS3 Vault"
	folders := make([]apiclient.FolderMeta, 0, len(m.folders))
	for _, f := range m.folders {
		if f.Folder == "" {
			continue
		}
		folders = append(folders, f)
	}
	sort.Slice(folders, func(i, j int) bool { return folders[i].Folder < folders[j].Folder })
	for _, f := range folders {
		items = append(items, Item{IsFolder: true, Name: f.Folder, SecretCount: f.SecretCount})
	}
	for _, s := range m.all {
		if s.Folder == "" {
			items = append(items, Item{
				Name: s.Name, Folder: "",
				CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
			})
		}
	}
	m.List.SetItems(items)
}

// InFolder reports whether the browser is currently inside a folder.
func (m *Model) InFolder() bool { return m.inFolder }

// CurrentFolder returns the folder currently open ("" at the root).
func (m *Model) CurrentFolder() string { return m.currentFolder }

func (m *Model) enterFolder(name string) {
	m.inFolder = true
	m.currentFolder = name
	m.List.ResetFilter()
	m.rebuild()
	m.List.Select(0)
}

// GoUp returns from a folder to the root view.
func (m *Model) GoUp() {
	m.inFolder = false
	m.currentFolder = ""
	m.List.ResetFilter()
	m.rebuild()
	m.List.Select(0)
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
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, constants.SecretListKeyMap.Select):
			// Enter doubles as the list's filter-accept key while typing a filter.
			if msg.String() == "enter" && m.List.FilterState() == list.Filtering {
				break
			}
			if item, ok := m.SelectedItem(); ok {
				if item.IsFolder {
					m.enterFolder(item.Name)
					return m, nil
				}
				return m, events.CmdSetState(shared.StateViewSecret)
			}
		case key.Matches(msg, constants.SecretListKeyMap.NewSecret):
			return m, events.CmdSetState(shared.StateNewSecret)
		case key.Matches(msg, constants.SecretListKeyMap.NewFolder):
			return m, events.CmdSetState(shared.StateNewFolder)
		case key.Matches(msg, constants.SecretListKeyMap.Delete):
			if item, ok := m.SelectedItem(); ok && !item.IsFolder {
				return m, events.CmdSetState(shared.StateDeleteSecret)
			}
		case key.Matches(msg, constants.SecretListKeyMap.Move):
			if item, ok := m.SelectedItem(); ok && !item.IsFolder {
				return m, events.CmdSetState(shared.StateMoveSecret)
			}
		case key.Matches(msg, constants.SecretListKeyMap.Edit):
			if item, ok := m.SelectedItem(); ok && !item.IsFolder {
				return m, events.CmdSetState(shared.StateEditSecret)
			}
		}
	}
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m *Model) View() tea.View { return tea.NewView(m.List.View()) }
