package secrets

type Item struct {
	Name      string
	CreatedAt string
	UpdatedAt string
}

func (i Item) Title() string       { return i.Name }
func (i Item) Description() string { return i.CreatedAt + " | " + i.UpdatedAt }
func (i Item) FilterValue() string { return i.Name }
