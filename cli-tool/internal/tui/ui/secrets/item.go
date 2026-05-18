package secrets

import "fmt"

// Item is one row in the secrets browser. It is either a folder entry
// (IsFolder true) or a secret entry.
type Item struct {
	IsFolder    bool
	Name        string // secret name, or folder name when IsFolder
	Folder      string // owning folder of a secret ("" = ungrouped)
	SecretCount int    // secrets contained, when IsFolder
	CreatedAt   string
	UpdatedAt   string
}

func (i Item) Title() string {
	if i.IsFolder {
		return "📁 " + i.Name + "/"
	}
	return "🔑 " + i.Name
}

func (i Item) Description() string {
	if i.IsFolder {
		unit := "secrets"
		if i.SecretCount == 1 {
			unit = "secret"
		}
		return fmt.Sprintf("%d %s", i.SecretCount, unit)
	}
	if i.UpdatedAt != "" {
		return "updated " + i.UpdatedAt
	}
	return "created " + i.CreatedAt
}

func (i Item) FilterValue() string { return i.Name }
