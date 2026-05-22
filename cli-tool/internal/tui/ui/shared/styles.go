package shared

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

var (
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Colors
	DarkPink         = compat.AdaptiveColor{Light: lipgloss.Color("#EE6FF8"), Dark: lipgloss.Color("#EE6FF8")}
	DullGrey         = compat.AdaptiveColor{Light: lipgloss.Color("#C2B8C2"), Dark: lipgloss.Color("#4D4D4D")}
	Purple           = compat.AdaptiveColor{Light: lipgloss.Color("#F793FF"), Dark: lipgloss.Color("#AD58B4")}
	VerySubduedColor = compat.AdaptiveColor{Light: lipgloss.Color("#DDDADA"), Dark: lipgloss.Color("#4b4b4b")}
	SubduedColor     = compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#5C5C5C")}
	MediumGrey       = compat.AdaptiveColor{Light: lipgloss.Color("#A49FA5"), Dark: lipgloss.Color("#777777")}
	DarkPurple       = lipgloss.Color("62")
	White            = lipgloss.Color("230")

	// End "Colors"
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Terminal Window

	DocStyle = lipgloss.NewStyle().
			Margin(1, 2)

	// End "Terminal Window"
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Menu

	MenuTitle = lipgloss.NewStyle().
			Background(DarkPurple).
			Foreground(White).
			Padding(0, 1).
			Margin(1, 0, 1, 2)

	MenuHelpStyle = lipgloss.NewStyle().
			Margin(1, 0, 0, 2)

	MenuSubStyle = lipgloss.NewStyle().
			Foreground(MediumGrey).
			MarginLeft(2).
			MarginBottom(1)

	// End "Menu"
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Buttons

	ButtonStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 1)

	SelectedButton = ButtonStyle.
			Background(DarkPurple).
			Foreground(White).
			Bold(true)

	UnselectedButton = ButtonStyle.
				Background(DullGrey).
				Foreground(lipgloss.Color("250"))

	// End "Buttons"
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Dialog

	DialogStyle = lipgloss.NewStyle().
			Padding(1, 6, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SubduedColor)

	DialogTitleStyle = lipgloss.NewStyle().
				Background(DarkPurple).
				Foreground(White).
				Padding(0, 1).
				Margin(0, 0, 2)

	DialogHelpStyle = lipgloss.NewStyle()

	DialogSubtitleStyle = lipgloss.NewStyle().
				MarginBottom(1)

	DialogRepoPath = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(2).
			Foreground(DarkPink)

	// End "Dialog"
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
)
