package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// securityRunMsg carries security scan results.
type securityRunMsg struct {
	result *core.SecurityCheckResult
}

// SecurityModel is the Security Scanner view.
type SecurityModel struct {
	urlInput    textinput.Model
	result      *core.SecurityCheckResult
	isLoading   bool
	spinner     spinner.Model
	width       int
	height      int
	t           theme.Theme
	scrollOffset int
}

// NewSecurity creates a new SecurityModel.
func NewSecurity() SecurityModel {
	url := textinput.New()
	url.Placeholder = "https://example.com"
	url.CharLimit = 2048
	url.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return SecurityModel{
		urlInput: url,
		spinner:  sp,
	}
}

// SetTheme updates the theme.
func (m *SecurityModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *SecurityModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
}

// runSecurity executes the security scan.
func (m SecurityModel) runSecurity() tea.Cmd {
	return func() tea.Msg {
		result := core.RunSecurityScan(m.urlInput.Value())
		return securityRunMsg{result: result}
	}
}

// Init implements tea.Model.
func (m SecurityModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m SecurityModel) Update(msg tea.Msg) (SecurityModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case securityRunMsg:
		m.isLoading = false
		m.result = msg.result
		m.scrollOffset = 0

	case spinner.TickMsg:
		if m.isLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r", "enter":
			url := m.urlInput.Value()
			if url != "" && !m.isLoading {
				m.isLoading = true
				m.result = nil
				cmds = append(cmds, m.runSecurity(), m.spinner.Tick)
			}
		case "j", "down":
			if m.result != nil {
				m.scrollOffset = min(m.scrollOffset+1, max(0, len(m.result.Issues)-5))
			}
		case "k", "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "g":
			m.scrollOffset = 0
		case "G":
			if m.result != nil {
				m.scrollOffset = max(0, len(m.result.Issues)-5)
			}
		default:
			var cmd tea.Cmd
			m.urlInput, cmd = m.urlInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the security view.
func (m SecurityModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	var sections []string
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))

	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).Width(contentWidth - 4)

	// URL input
	sections = append(sections, labelStyle.Render("Target URL"))
	sections = append(sections, activeStyle.Render(m.urlInput.View()))
	sections = append(sections, "")
	sections = append(sections, mutedStyle.Render("Press r or Enter to scan"))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Running security scan..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	if m.result == nil {
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	r := m.result

	// TLS info
	tlsColor := t.Success
	if strings.Contains(r.TLSInfo, "plaintext") || strings.Contains(r.TLSInfo, "DEPRECATED") {
		tlsColor = t.Error
	} else if strings.Contains(r.TLSInfo, "unknown") {
		tlsColor = t.Warning
	}
	tlsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tlsColor))
	sections = append(sections, fmt.Sprintf("  TLS: %s", tlsStyle.Render(r.TLSInfo)))

	// CORS info
	corsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))
	sections = append(sections, fmt.Sprintf("  CORS: %s", corsStyle.Render(r.CORSInfo)))
	sections = append(sections, "")

	// Summary by severity
	counts := core.CountIssuesBySeverity(r.Issues)
	summaryParts := []string{}
	for _, sev := range []core.Severity{core.SeverityCritical, core.SeverityHigh, core.SeverityMedium, core.SeverityLow} {
		if n := counts[sev]; n > 0 {
			color := severityColor(sev, t)
			s := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
			summaryParts = append(summaryParts, s.Render(fmt.Sprintf("%d %s", n, sev.String())))
		}
	}
	if len(summaryParts) > 0 {
		sections = append(sections, "  "+strings.Join(summaryParts, "  "))
	} else {
		goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
		sections = append(sections, "  "+goodStyle.Render("✓ All security headers present and well-configured"))
	}
	sections = append(sections, "")

	// Issues table header
	sections = append(sections, renderSectionTitle("Security Headers", t))
	sections = append(sections, renderDivider(contentWidth-4, t))

	// Column layout
	col1 := contentWidth / 4
	col2 := col1
	col3 := contentWidth - col1 - col2 - 10

	headerRow := fmt.Sprintf("  %-*s %-*s %-*s %s",
		col1, "HEADER", col2, "STATUS", col3, "VALUE/DESCRIPTION", "SEV")
	sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Render(headerRow))
	sections = append(sections, renderDivider(contentWidth-4, t))

	// Scrollable issues list
	startIdx := m.scrollOffset
	endIdx := min(startIdx+20, len(r.Issues))

	for _, issue := range r.Issues[startIdx:endIdx] {
		// Status indicator
		statusStr := "✓ PRESENT"
		statusColor := t.Success
		if !issue.Present {
			statusStr = "✗ MISSING"
			statusColor = t.Error
		} else if issue.Severity >= core.SeverityHigh {
			statusStr = "⚠ WARNING"
			statusColor = t.Warning
		}

		sevColor := severityColor(issue.Severity, t)
		sevStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sevColor)).Bold(true)
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		headerName := truncate(issue.Header, col1-1)
		if headerName == "" {
			headerName = truncate(issue.Category, col1-1)
		}
		statusDisplay := truncate(statusStr, col2-1)
		descDisplay := issue.Description
		if issue.Present && issue.Value != "" {
			descDisplay = truncate(issue.Value, col3-1)
		} else {
			descDisplay = truncate(issue.Description, col3-1)
		}

		row := fmt.Sprintf("  %-*s %s %s %s",
			col1, headerName,
			statusStyle.Render(fmt.Sprintf("%-*s", col2, statusDisplay)),
			lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground)).Render(fmt.Sprintf("%-*s", col3, descDisplay)),
			sevStyle.Render(issue.Severity.String()),
		)
		sections = append(sections, row)
	}

	if len(r.Issues) > 20 {
		sections = append(sections, "")
		sections = append(sections, mutedStyle.Render(fmt.Sprintf(
			"  Showing %d-%d of %d  (j/k to scroll)",
			startIdx+1, endIdx, len(r.Issues),
		)))
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}

// GetIssues returns security issues for stats.
func (m SecurityModel) GetIssues() []core.SecurityIssue {
	if m.result == nil {
		return nil
	}
	return m.result.Issues
}
