package constants

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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
			tea.KeyEnter.String(),
			tea.KeySpace.String(),
		),
		key.WithHelp(
			tea.KeyEnter.String()+"/"+tea.KeySpace.String(),
			unsetText,
		),
	),
	Exit: key.NewBinding(
		key.WithKeys(tea.KeyEsc.String()),
		key.WithHelp(
			tea.KeyEsc.String(),
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
	NewSecret key.Binding
	Delete    key.Binding
}

var SecretListKeyMap = secretListKeyMap{
	Select: key.NewBinding(
		key.WithKeys(tea.KeyTab.String()),            // actual keybindings
		key.WithHelp(tea.KeyTab.String(), unsetText), // corresponding help text
	),
	NewSecret: key.NewBinding(
		key.WithKeys(tea.KeyCtrlN.String()),
		key.WithHelp(tea.KeyCtrlN.String(), unsetText),
	),
	Delete: key.NewBinding(
		key.WithKeys(tea.KeyCtrlBackslash.String()),
		key.WithHelp(tea.KeyCtrlBackslash.String(), unsetText),
	),
}

func (k secretListKeyMap) HelpBinds(helpType HelpType) func() []key.Binding {
	bindsWithHelp := []key.Binding{
		SetOnHelpType(
			helpType,                // Short or Full Help
			SecretListKeyMap.Select, // key.Binding
			"view",                  // Short Help
			"view secret",           // Full Help
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.NewSecret,
			"new",
			"new secret",
		),
		SetOnHelpType(
			helpType,
			SecretListKeyMap.Delete,
			"delete",
			"delete secret",
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
		key.WithKeys(tea.KeyTab.String()),
		key.WithHelp(tea.KeyTab.String(), unsetText),
	),
	NewToken: key.NewBinding(
		key.WithKeys(tea.KeyCtrlN.String()),
		key.WithHelp(tea.KeyCtrlN.String(), unsetText),
	),
	DeleteToken: key.NewBinding(
		key.WithKeys(tea.KeyCtrlBackslash.String()),
		key.WithHelp(tea.KeyCtrlBackslash.String(), unsetText),
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
		key.WithKeys(tea.KeyTab.String()),
		key.WithHelp(tea.KeyTab.String(), unsetText),
	),
	AddUser: key.NewBinding(
		key.WithKeys(tea.KeyCtrlN.String()),
		key.WithHelp(tea.KeyCtrlN.String(), unsetText),
	),
	DeleteUser: key.NewBinding(
		key.WithKeys(tea.KeyCtrlBackslash.String()),
		key.WithHelp(tea.KeyCtrlBackslash.String(), unsetText),
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
		key.WithKeys(tea.KeyCtrlT.String()),
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
		key.WithKeys(tea.KeyCtrlP.String()),
		key.WithHelp("ctrl+p", unsetText),
	),
	Exit: key.NewBinding(
		key.WithKeys(tea.KeyEsc.String()),
		key.WithHelp(tea.KeyEsc.String(), unsetText),
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
