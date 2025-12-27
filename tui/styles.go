package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors - a nice warm palette for gristle
var (
	ColorPrimary   = lipgloss.Color("#E67E22") // Orange - like grilled meat
	ColorSecondary = lipgloss.Color("#8E44AD") // Purple accent
	ColorMuted     = lipgloss.Color("#7F8C8D") // Gray for subtle text
	ColorSuccess   = lipgloss.Color("#27AE60") // Green
	ColorDanger    = lipgloss.Color("#E74C3C") // Red
	ColorBg        = lipgloss.Color("#1A1A2E") // Dark background
	ColorFg        = lipgloss.Color("#ECF0F1") // Light foreground
)

// Styles
var (
	// App frame
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header/title bar
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Breadcrumb navigation
	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginBottom(1)

	BreadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(ColorFg).
				Bold(true)

	BreadcrumbSeparator = lipgloss.NewStyle().
				Foreground(ColorMuted).
				SetString(" > ")

	// List items
	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	// Cursor
	CursorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			SetString("> ")

	// Footer/help
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// Status messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// Spinner/loading
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Document info box
	DocInfoStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			MarginTop(1)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorMuted)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Badge styles (for counts, status)
	BadgeStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			SetString(" (%s)")

	PinnedBadge = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			SetString(" [pinned]")
)

// Helper to create a styled list item
func RenderListItem(text string, selected bool, count int) string {
	var item string
	if selected {
		item = CursorStyle.Render() + SelectedItemStyle.Render(text)
	} else {
		item = "  " + ItemStyle.Render(text)
	}

	if count >= 0 {
		item += lipgloss.NewStyle().Foreground(ColorMuted).Render(" (" + string(rune('0'+count%10)) + ")")
	}

	return item
}

// Render breadcrumb path
func RenderBreadcrumb(parts []string) string {
	if len(parts) == 0 {
		return BreadcrumbActiveStyle.Render("gristle")
	}

	result := BreadcrumbStyle.Render("gristle")
	for i, part := range parts {
		result += BreadcrumbSeparator.Render()
		if i == len(parts)-1 {
			result += BreadcrumbActiveStyle.Render(part)
		} else {
			result += BreadcrumbStyle.Render(part)
		}
	}
	return result
}
