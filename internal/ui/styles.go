package ui

import "charm.land/lipgloss/v2"

type Styles struct {
	Title            lipgloss.Style
	Section          lipgloss.Style
	Method           lipgloss.Style
	Muted            lipgloss.Style
	SmallScreen      lipgloss.Style
	Tabs             lipgloss.Style
	ActiveTab        lipgloss.Style
	InactiveTab      lipgloss.Style
	ListItem         lipgloss.Style
	SelectedListItem lipgloss.Style
	Help             lipgloss.Style
	panel            lipgloss.Style
	activePanel      lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")),
		Section: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("116")),
		Method: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("114")),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		SmallScreen: lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.Color("230")),
		Tabs: lipgloss.NewStyle().
			Padding(0, 1),
		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("31")).
			Padding(0, 1),
		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Background(lipgloss.Color("238")).
			Padding(0, 1),
		ListItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		SelectedListItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236")),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("235")).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1),
		activePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("31")).
			Padding(0, 1),
	}
}

func (s Styles) Panel(active bool) lipgloss.Style {
	if active {
		return s.activePanel
	}
	return s.panel
}

func Clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
