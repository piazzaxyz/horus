package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

const (
	sidebarWidth = 22
	appVersion   = "v1.0.0"
	appName      = "QAITOR"
)

func renderHeader(width int, themeName string, t theme.Theme) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color(t.Primary)).
		Foreground(lipgloss.Color(t.Background)).
		Bold(true).
		Width(width).
		Padding(0, 1)

	left := fmt.Sprintf(" %s %s", appName, appVersion)
	right := fmt.Sprintf("Theme: %s ", themeName)
	spacer := strings.Repeat(" ", max(0, width-len(left)-len(right)-2))

	return style.Render(left + spacer + right)
}

func renderSidebar(current core.Page, height int, t theme.Theme) string {
	menuItems := []struct {
		page  core.Page
		label string
		key   string
	}{
		{core.PageDashboard, "Dashboard", "1"},
		{core.PageAnalyzer, "HTTP Analyzer", "2"},
		{core.PageTasks, "Task Runner", "3"},
		{core.PageLeaks, "Leak Scanner", "4"},
		{core.PageThrottle, "Throttle Det.", "5"},
		{core.PageSecurity, "Security Scan", "6"},
		{core.PageThemes, "Theme Picker", "7"},
		{core.PageTutorial, "Tutorial", "8"},
	}

	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(t.Primary)).
		Foreground(lipgloss.Color(t.Background)).
		Bold(true).
		Width(sidebarWidth - 2).
		Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Foreground)).
		Width(sidebarWidth - 2).
		Padding(0, 1)

	var items []string
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		Width(sidebarWidth - 2).
		Padding(0, 1)
	items = append(items, titleStyle.Render("  VIEWS"))
	items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border)).Width(sidebarWidth-2).Render(strings.Repeat("─", sidebarWidth-4)))

	for _, item := range menuItems {
		indicator := "○"
		if item.page == current {
			indicator = "●"
		}
		label := fmt.Sprintf("%s [%s] %s", indicator, item.key, item.label)

		if item.page == current {
			items = append(items, activeStyle.Render(label))
		} else {
			items = append(items, inactiveStyle.Render(label))
		}
	}

	// Fill remaining space
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Width(sidebarWidth - 2)

	content := strings.Join(items, "\n")
	// Pad to fill height
	lines := strings.Split(content, "\n")
	for len(lines) < height-4 {
		lines = append(lines, lipgloss.NewStyle().Width(sidebarWidth-2).Render(""))
	}

	return borderStyle.Render(strings.Join(lines, "\n"))
}

func renderFooter(current core.Page, width int, t theme.Theme) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color(t.Border)).
		Foreground(lipgloss.Color(t.Foreground)).
		Width(width).
		Padding(0, 1)

	pageName := core.PageNames[current]
	keys := "q:quit  ?:help  t:theme  1-8:views  r:run  Tab:focus"
	spacer := strings.Repeat(" ", max(0, width-len(pageName)-len(keys)-4))

	return style.Render(fmt.Sprintf(" %s%s%s ", pageName, spacer, keys))
}

func renderHelp(width, height int, t theme.Theme) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Background(lipgloss.Color(t.Background)).
		Foreground(lipgloss.Color(t.Foreground)).
		Padding(1, 2).
		Width(min(70, width-4))

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Accent)).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Foreground))

	help := [][]string{
		{"Navigation", ""},
		{"1-8", "Switch to view by number"},
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"h / ←", "Move left / previous"},
		{"l / →", "Move right / next"},
		{"g", "Go to top"},
		{"G", "Go to bottom"},
		{"", ""},
		{"Actions", ""},
		{"r / Enter", "Run / Execute"},
		{"Tab", "Focus next input field"},
		{"Shift+Tab", "Focus previous input field"},
		{"Esc", "Cancel / Go back"},
		{"", ""},
		{"Global", ""},
		{"q", "Quit QAITOR"},
		{"?", "Toggle this help overlay"},
		{"t", "Cycle to next theme"},
	}

	var lines []string
	lines = append(lines, titleStyle.Render("QAITOR Keyboard Reference"))
	lines = append(lines, "")

	for _, row := range help {
		if row[1] == "" {
			if row[0] != "" {
				lines = append(lines, titleStyle.Render(row[0]))
			} else {
				lines = append(lines, "")
			}
		} else {
			key := keyStyle.Render(fmt.Sprintf("%-14s", row[0]))
			desc := descStyle.Render(row[1])
			lines = append(lines, fmt.Sprintf("  %s  %s", key, desc))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Render("Press ? to close"))

	content := strings.Join(lines, "\n")

	helpBox := style.Render(content)

	// Center horizontally and vertically
	helpWidth := lipgloss.Width(helpBox)
	helpHeight := lipgloss.Height(helpBox)

	leftPad := max(0, (width-helpWidth)/2)
	topPad := max(0, (height-helpHeight)/2)

	var rows []string
	emptyRow := strings.Repeat(" ", width)
	for i := 0; i < topPad; i++ {
		rows = append(rows, emptyRow)
	}

	helpLines := strings.Split(helpBox, "\n")
	for _, line := range helpLines {
		rows = append(rows, strings.Repeat(" ", leftPad)+line)
	}

	return strings.Join(rows, "\n")
}

func renderCard(title, value, subtitle string, cardWidth int, t theme.Theme) string {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Background(lipgloss.Color(t.Highlight)).
		Padding(0, 1).
		Width(cardWidth)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Muted)).
		Bold(false)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Comment))

	content := fmt.Sprintf("%s\n%s\n%s",
		titleStyle.Render(title),
		valueStyle.Render(value),
		subtitleStyle.Render(subtitle),
	)

	return cardStyle.Render(content)
}

func renderSectionTitle(title string, t theme.Theme) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		Render(title)
}

func renderDivider(width int, t theme.Theme) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Border)).
		Render(strings.Repeat("─", width))
}

func severityColor(s core.Severity, t theme.Theme) string {
	switch s {
	case core.SeverityCritical:
		return t.Critical
	case core.SeverityHigh:
		return t.High
	case core.SeverityMedium:
		return t.Medium
	case core.SeverityLow:
		return t.Low
	default:
		return t.Muted
	}
}

func statusColor(code int, t theme.Theme) string {
	switch {
	case code >= 200 && code < 300:
		return t.Success
	case code >= 300 && code < 400:
		return t.Info
	case code >= 400 && code < 500:
		return t.Warning
	case code >= 500:
		return t.Error
	default:
		return t.Muted
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Note: min and max are builtins in Go 1.21+
