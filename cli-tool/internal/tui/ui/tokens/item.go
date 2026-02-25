package tokens

import (
	"fmt"
	"time"
)

type Item struct {
	Name      string
	CreatedAt string
	ExpiresAt *int64
}

func (i Item) Title() string { return i.Name }

func (i Item) Description() string {
	exp := "no expiry"
	if i.ExpiresAt != nil {
		exp = fmt.Sprintf("expires: %s", time.Unix(*i.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
	}
	return fmt.Sprintf("created: %s | %s", i.CreatedAt, exp)
}

func (i Item) FilterValue() string { return i.Name }
