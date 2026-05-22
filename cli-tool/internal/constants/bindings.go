package constants

import (
	"charm.land/bubbles/v2/key"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Helpers

type HelpType int

const (
	Short HelpType = iota
	Full
)

var unsetText = "not set"

// type keyMap interface {
// 	HelpBinds()
// }

func SetOnHelpType(helpType HelpType, bind key.Binding, shortHelp string, fullHelp string) key.Binding {
	bindWithHelp := bind

	switch helpType {
	case Short:
		bindWithHelp.SetHelp(bind.Help().Key, shortHelp)
	case Full:
		bindWithHelp.SetHelp(bind.Help().Key, fullHelp)
	}
	return bindWithHelp
}

// End "Helpers"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
//// Default Key Map

type defaultKeyMap struct {
	Select key.Binding
	Exit   key.Binding
}

var DefaultKeyMap = defaultKeyMap{
	Select: key.NewBinding(
		key.WithKeys(
			"enter",
			"space",
		),
		key.WithHelp(
			"enter/space",
			unsetText,
		),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp(
			"esc",
			unsetText,
		),
	),
}

func (k defaultKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(
			helpType,             // Short or Full Help
			DefaultKeyMap.Select, // key.Binding
			"select",             // Short Help
			"select",             // Full Help
		),
		SetOnHelpType(
			helpType,
			DefaultKeyMap.Exit,
			"exit",
			"exit",
		),
	}

	return func() []key.Binding { return bindsWithHelp }
}

// End "Default Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Repo List Key Map

type secretListKeyMap struct {
	Select    key.Binding
	Back      key.Binding
	NewSecret key.Binding
	NewFolder key.Binding
	Delete    key.Binding
	Move      key.Binding
	Edit      key.Binding
}

var SecretListKeyMap = secretListKeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter", "tab"),     // open folder / view secret
		key.WithHelp("enter", unsetText), // corresponding help text
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", unsetText),
	),
	NewSecret: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", unsetText),
	),
	NewFolder: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", unsetText),
	),
	Delete: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("ctrl+\\", unsetText),
	),
	Move: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", unsetText),
	),
	Edit: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", unsetText),
	),
}

func (k secretListKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(
			helpType,                    // Short or Full Help
			SecretListKeyMap.Select,     // key.Binding
			"open",                      // Short Help
			"open folder / view secret", // Full Help
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.Back,
			"back",
			"back / up a folder",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.NewSecret,
			"new",
			"new secret",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.NewFolder,
			"new folder",
			"new folder",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.Delete,
			"delete",
			"delete secret",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.Move,
			"move",
			"move to folder",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.Edit,
			"edit",
			"edit secret value",
		),
	}

	return func() []key.Binding { return bindsWithHelp }
}

// End "Secret List Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Token List Key Map

type tokenListKeyMap struct {
	Select      key.Binding
	NewToken    key.Binding
	DeleteToken key.Binding
}

var TokenListKeyMap = tokenListKeyMap{
	Select: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", unsetText),
	),
	NewToken: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", unsetText),
	),
	DeleteToken: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("ctrl+\\", unsetText),
	),
}

func (k tokenListKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(helpType, TokenListKeyMap.Select, "view", "view token details"),
		SetOnHelpType(helpType, TokenListKeyMap.NewToken, "new token", "generate new token"),
		SetOnHelpType(helpType, TokenListKeyMap.DeleteToken, "delete", "delete token"),
	}
	return func() []key.Binding { return bindsWithHelp }
}

// End "Token List Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// User List Key Map

type userListKeyMap struct {
	Select     key.Binding
	AddUser    key.Binding
	DeleteUser key.Binding
}

var UserListKeyMap = userListKeyMap{
	Select: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", unsetText),
	),
	AddUser: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", unsetText),
	),
	DeleteUser: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("ctrl+\\", unsetText),
	),
}

func (k userListKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(helpType, UserListKeyMap.Select, "view", "view user details"),
		SetOnHelpType(helpType, UserListKeyMap.AddUser, "add user", "add new user"),
		SetOnHelpType(helpType, UserListKeyMap.DeleteUser, "delete", "delete user"),
	}
	return func() []key.Binding { return bindsWithHelp }
}

// End "User List Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// App Key Map

type appKeyMap struct {
	ToggleView key.Binding
}

var AppKeyMap = appKeyMap{
	ToggleView: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle view"),
	),
}

// End "App Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Confirm Key Map

type confirmKeyMap struct {
	Proceed key.Binding
	Exit    key.Binding
}

var ConfirmKeyMap = confirmKeyMap{
	Proceed: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", unsetText),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", unsetText),
	),
}

func (k confirmKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(
			helpType,                // Short or Full Help
			ConfirmKeyMap.Proceed,   // key.Binding
			"proceed",               // Short Help
			"proceed with deleting", // Full Help
		),
		SetOnHelpType(
			helpType,
			ConfirmKeyMap.Exit,
			"back",
			"back to menu",
		),
	}

	return func() []key.Binding { return bindsWithHelp }
}

// End "Confirm Key Map"
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
