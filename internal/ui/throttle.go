package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// throttleRunMsg carries throttle test results.
type throttleRunMsg struct {
	analysis core.ThrottleAnalysis
}

// ThrottleModel is the Throttle Detector view.
type ThrottleModel struct {
	urlInput      textinput.Model
	countInput    textinput.Model
	intervalInput textinput.Model
	activeField   int
	typing        bool
	analysis      *core.ThrottleAnalysis
	running       bool
	spinner       spinner.Model
	progress      progress.Model
	width         int
	height        int
	t             theme.Theme
	scrollOffset  int
}

// NewThrottle creates a new ThrottleModel.
func NewThrottle() ThrottleModel {
	url := textinput.New()
	url.Placeholder = "https://api.example.com/endpoint"
	url.CharLimit = 2048

	count := textinput.New()
	count.Placeholder = "20"
	count.SetValue("20")
	count.CharLimit = 5

	interval := textinput.New()
	interval.Placeholder = "100"
	interval.SetValue("100")
	interval.CharLimit = 8

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	prog := progress.New(progress.WithScaledGradient("#7aa2f7", "#9ece6a"))

	return ThrottleModel{
		urlInput:      url,
		countInput:    count,
		intervalInput: interval,
		spinner:       sp,
		progress:      prog,
	}
}

// SetTheme updates the theme.
func (m *ThrottleModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
	m.progress = progress.New(progress.WithScaledGradient(t.Primary, t.Success))
}

// SetSize updates dimensions.
func (m *ThrottleModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.countInput.Width = 8
	m.intervalInput.Width = 10
	m.progress.Width = contentWidth - 4
}

// IsTyping returns true when in input editing mode.
func (m ThrottleModel) IsTyping() bool {
	return m.typing
}

// runThrottle executes the throttle test.
func (m ThrottleModel) runThrottle() tea.Cmd {
	return func() tea.Msg {
		count, _ := strconv.Atoi(m.countInput.Value())
		if count <= 0 {
			count = 10
		}
		if count > 200 {
			count = 200
		}
		interval, _ := strconv.Atoi(m.intervalInput.Value())
		if interval < 0 {
			interval = 0
		}

		cfg := core.ThrottleConfig{
			URL:        m.urlInput.Value(),
			Count:      count,
			IntervalMs: interval,
			Method:     "GET",
		}
		analysis := core.RunThrottleTest(cfg)
		return throttleRunMsg{analysis: analysis}
	}
}

// Init implements tea.Model.
func (m ThrottleModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m ThrottleModel) Update(msg tea.Msg) (ThrottleModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case throttleRunMsg:
		m.running = false
		m.analysis = &msg.analysis

	case spinner.TickMsg:
		if m.running {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if !m.typing {
				m.typing = true
				m.activeField = 0
			} else {
				m.activeField = (m.activeField + 1) % 3
			}
			m.focusField()
		case "shift+tab":
			if m.typing {
				m.activeField = (m.activeField + 2) % 3
				m.focusField()
			}
		case "ctrl+s":
			m.typing = false
			m.urlInput.Blur()
			m.countInput.Blur()
			m.intervalInput.Blur()
		case "ctrl+r":
			if !m.running {
				url := m.urlInput.Value()
				if url != "" {
					m.running = true
					m.analysis = nil
					cmds = append(cmds, m.runThrottle(), m.spinner.Tick)
				}
			}
		case "enter":
			if m.typing && !m.running {
				url := m.urlInput.Value()
				if url != "" {
					m.running = true
					m.analysis = nil
					cmds = append(cmds, m.runThrottle(), m.spinner.Tick)
				}
			}
		case "j", "down":
			if m.analysis != nil {
				m.scrollOffset = min(m.scrollOffset+1, max(0, len(m.analysis.Results)-10))
			}
		case "k", "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "g":
			m.scrollOffset = 0
		case "G":
			if m.analysis != nil {
				m.scrollOffset = max(0, len(m.analysis.Results)-10)
			}
		}

		// Update focused input — only in typing mode
		if m.typing {
			switch m.activeField {
			case 0:
				var cmd tea.Cmd
				m.urlInput, cmd = m.urlInput.Update(msg)
				cmds = append(cmds, cmd)
			case 1:
				var cmd tea.Cmd
				m.countInput, cmd = m.countInput.Update(msg)
				cmds = append(cmds, cmd)
			case 2:
				var cmd tea.Cmd
				m.intervalInput, cmd = m.intervalInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ThrottleModel) focusField() {
	m.urlInput.Blur()
	m.countInput.Blur()
	m.intervalInput.Blur()
	switch m.activeField {
	case 0:
		m.urlInput.Focus()
	case 1:
		m.countInput.Focus()
	case 2:
		m.intervalInput.Focus()
	}
}

// View renders the throttle view.
func (m ThrottleModel) View(t theme.Theme) string {
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
		Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(0, 1)

	// URL input
	urlLabel := labelStyle.Render("Target URL")
	urlStyle := inactiveStyle.Width(contentWidth - 4)
	if m.activeField == 0 {
		urlStyle = activeStyle.Width(contentWidth - 4)
	}
	sections = append(sections, urlLabel)
	sections = append(sections, urlStyle.Render(m.urlInput.View()))
	sections = append(sections, "")

	// Count + Interval row
	countLabel := labelStyle.Render("Request Count")
	countStyle := inactiveStyle.Width(14)
	if m.activeField == 1 {
		countStyle = activeStyle.Width(14)
	}

	intervalLabel := labelStyle.Render("Interval (ms)")
	intervalStyle := inactiveStyle.Width(14)
	if m.activeField == 2 {
		intervalStyle = activeStyle.Width(14)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, countLabel, countStyle.Render(m.countInput.View())),
		"   ",
		lipgloss.JoinVertical(lipgloss.Left, intervalLabel, intervalStyle.Render(m.intervalInput.View())),
	)
	sections = append(sections, row)
	sections = append(sections, "")
	sections = append(sections, mutedStyle.Render("Tab: switch field  |  r/Enter: run test"))
	sections = append(sections, "")

	// Loading
	if m.running {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Running throttle test..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	// Results
	if m.analysis != nil {
		a := m.analysis
		sections = append(sections, renderSectionTitle("Throttle Analysis", t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Summary cards
		isThrottledStr := "NO"
		isThrottledColor := t.Success
		if a.IsThrottled {
			isThrottledStr = "YES"
			isThrottledColor = t.Error
		}

		summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(isThrottledColor)).Bold(true)
		sections = append(sections, fmt.Sprintf("  Rate Limited: %s  |  Throttled: %d/%d (%.1f%%)",
			summaryStyle.Render(isThrottledStr),
			a.Throttled, a.TotalRequests, a.ThrottleRate,
		))
		sections = append(sections, fmt.Sprintf("  Avg: %s  |  Min: %s  |  Max: %s",
			a.AvgDuration.Round(1e6).String(),
			a.MinDuration.Round(1e6).String(),
			a.MaxDuration.Round(1e6).String(),
		))
		if a.RetryAfter != "" {
			sections = append(sections, fmt.Sprintf("  Retry-After: %s", core.ParseRetryAfter(a.RetryAfter)))
		}

		patternStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))
		sections = append(sections, fmt.Sprintf("  Pattern: %s", patternStyle.Render(a.Pattern)))
		sections = append(sections, "")

		// Results table
		sections = append(sections, renderSectionTitle("Request Log", t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		headerCols := []string{"#", "Status", "Duration", "Throttled", "Retry-After"}
		colW := []int{5, 8, 14, 12, 20}
		sections = append(sections, tableRow(headerCols, colW, t.Muted, t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Scrollable window
		startIdx := m.scrollOffset
		endIdx := min(startIdx+15, len(a.Results))

		for _, r := range a.Results[startIdx:endIdx] {
			statusStr := fmt.Sprintf("%d", r.StatusCode)
			if r.Error != nil {
				statusStr = "ERR"
			}
			throttledStr := "  -"
			color := t.Foreground
			if r.Throttled {
				throttledStr = "  THROTTLED"
				color = t.Error
			}
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				color = t.Success
			} else if r.StatusCode == 429 || r.StatusCode == 503 {
				color = t.Error
			} else if r.StatusCode >= 400 {
				color = t.Warning
			}

			cols := []string{
				fmt.Sprintf(" %d", r.RequestNum),
				statusStr,
				r.Duration.Round(1e6).String(),
				throttledStr,
				r.RetryAfter,
			}
			sections = append(sections, tableRow(cols, colW, color, t))
		}

		if len(a.Results) > 15 {
			sections = append(sections, mutedStyle.Render(fmt.Sprintf(
				"  Showing %d-%d of %d  (j/k to scroll)",
				startIdx+1, endIdx, len(a.Results),
			)))
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}

// GetAnalysis returns the throttle analysis for stats.
func (m ThrottleModel) GetAnalysis() *core.ThrottleAnalysis {
	return m.analysis
}
