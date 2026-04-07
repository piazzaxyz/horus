package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// DashboardModel holds state for the dashboard view.
type DashboardModel struct{}

// NewDashboard creates a new DashboardModel.
func NewDashboard() DashboardModel {
	return DashboardModel{}
}

// View renders the dashboard view.
func (d DashboardModel) View(width, height int, t theme.Theme, totalReqs, totalLeaks, totalIssues int, avgMs float64, logs []core.LogEntry) string {
	contentWidth := width - sidebarWidth - 2

	var sections []string

	// Welcome banner
	bannerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		Padding(0, 1)
	sections = append(sections, bannerStyle.Render("QAITOR - QA & Security Testing Dashboard"))
	sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Render("  Analyze. Detect. Secure."))
	sections = append(sections, "")

	// Stats cards - 2x2 grid
	cardWidth := (contentWidth - 6) / 2
	if cardWidth < 20 {
		cardWidth = 20
	}

	card1 := renderCard(
		"TASKS RUN",
		fmt.Sprintf("%d", totalReqs),
		"Total HTTP requests executed",
		cardWidth, t,
	)
	card2 := renderCard(
		"LEAKS FOUND",
		fmt.Sprintf("%d", totalLeaks),
		"Data leaks detected",
		cardWidth, t,
	)
	card3 := renderCard(
		"AVG RESPONSE",
		fmt.Sprintf("%.0f ms", avgMs),
		"Average response time",
		cardWidth, t,
	)
	card4 := renderCard(
		"SECURITY ISSUES",
		fmt.Sprintf("%d", totalIssues),
		"Security findings",
		cardWidth, t,
	)

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, card3, "  ", card4)
	sections = append(sections, row1)
	sections = append(sections, "")
	sections = append(sections, row2)
	sections = append(sections, "")

	// Recent activity
	sections = append(sections, renderSectionTitle("Recent Activity", t))
	sections = append(sections, renderDivider(contentWidth, t))

	if len(logs) == 0 {
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
		sections = append(sections, mutedStyle.Render("  No activity yet. Press 1-8 to navigate to a view and run a scan."))
	} else {
		// Show last N logs
		showLogs := logs
		maxLogs := height - 20
		if maxLogs < 3 {
			maxLogs = 3
		}
		if len(showLogs) > maxLogs {
			showLogs = showLogs[len(showLogs)-maxLogs:]
		}

		for _, entry := range showLogs {
			levelColor := logLevelColor(entry.Level, t)
			levelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(levelColor)).Bold(true)
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Comment))
			msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground))

			timeStr := entry.Timestamp.Format(time.Kitchen)
			line := fmt.Sprintf("  %s %s %s",
				timeStyle.Render(timeStr),
				levelStyle.Render(fmt.Sprintf("[%s]", entry.Level)),
				msgStyle.Render(entry.Message),
			)
			sections = append(sections, line)
		}
	}

	// Quick start tips if no activity
	if len(logs) == 0 {
		sections = append(sections, "")
		sections = append(sections, renderSectionTitle("Quick Start", t))
		sections = append(sections, renderDivider(contentWidth, t))
		tips := []string{
			"  [2] HTTP Analyzer  - Send HTTP requests and inspect responses",
			"  [3] Task Runner    - Define and run multiple test tasks",
			"  [4] Leak Scanner   - Scan responses for PII and credentials",
			"  [5] Throttle Det.  - Detect rate limiting behavior",
			"  [6] Security Scan  - Check security headers and TLS",
			"  [7] Theme Picker   - Customize the visual theme",
			"  [8] Tutorial       - Learn how to use QAITOR",
		}
		tipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))
		for _, tip := range tips {
			sections = append(sections, tipStyle.Render(tip))
		}
	}

	outerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 2)

	content := strings.Join(sections, "\n")
	return outerStyle.Render(content)
}

func logLevelColor(level string, t theme.Theme) string {
	switch strings.ToUpper(level) {
	case "SUCCESS":
		return t.Success
	case "ERROR":
		return t.Error
	case "WARN", "WARNING":
		return t.Warning
	case "INFO":
		return t.Info
	default:
		return t.Muted
	}
}
