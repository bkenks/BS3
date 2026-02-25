package users

type Item struct {
	Username  string
	CreatedAt string
}

func (i Item) Title() string       { return i.Username }
func (i Item) Description() string { return "created: " + i.CreatedAt }
func (i Item) FilterValue() string { return i.Username }
