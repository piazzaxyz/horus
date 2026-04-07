package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// leaksRunMsg carries leak scan results.
type leaksRunMsg struct {
	leaks       []core.DataLeak
	rawResponse string
	statusCode  int
	err         error
}

// LeaksModel is the Leak Scanner view.
type LeaksModel struct {
	urlInput    textinput.Model
	rawInput    textinput.Model // for pasting raw text directly
	useRaw      bool
	results     []core.DataLeak
	rawResponse string
	responseVP  viewport.Model
	isLoading   bool
	spinner     spinner.Model
	typing      bool
	width       int
	height      int
	t           theme.Theme
	activeField int // 0=url, 1=run button area
	statusCode  int
	err         error
}

// NewLeaks creates a new LeaksModel.
func NewLeaks() LeaksModel {
	url := textinput.New()
	url.Placeholder = "https://api.example.com/data"
	url.CharLimit = 2048

	raw := textinput.New()
	raw.Placeholder = "Or paste raw text/JSON to scan here..."
	raw.CharLimit = 65536

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return LeaksModel{
		urlInput:   url,
		rawInput:   raw,
		spinner:    sp,
		responseVP: vp,
	}
}

// SetTheme updates the theme.
func (m *LeaksModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *LeaksModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.rawInput.Width = contentWidth - 4
	vpHeight := h - 28
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.responseVP.Width = contentWidth - 4
	m.responseVP.Height = vpHeight
}

// IsTyping returns true when in input editing mode.
func (m LeaksModel) IsTyping() bool {
	return m.typing
}

// runLeakScan fetches URL and scans for leaks.
func (m LeaksModel) runLeakScan() tea.Cmd {
	return func() tea.Msg {
		if m.useRaw {
			text := m.rawInput.Value()
			leaks := core.ScanForLeaks(text)
			return leaksRunMsg{leaks: leaks, rawResponse: text, statusCode: 0}
		}

		client := core.NewHTTPClient()
		req := core.Request{Method: "GET", URL: m.urlInput.Value()}
		resp := client.Execute(req)

		if resp.Error != nil {
			return leaksRunMsg{err: resp.Error}
		}

		leaks := core.ScanForLeaks(resp.Body)
		return leaksRunMsg{leaks: leaks, rawResponse: resp.Body, statusCode: resp.StatusCode}
	}
}

// Init implements tea.Model.
func (m LeaksModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m LeaksModel) Update(msg tea.Msg) (LeaksModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case leaksRunMsg:
		m.isLoading = false
		m.results = msg.leaks
		m.rawResponse = msg.rawResponse
		m.statusCode = msg.statusCode
		m.err = msg.err
		m.responseVP.SetContent(m.rawResponse)

	case spinner.TickMsg:
		if m.isLoading {
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
				m.urlInput.Focus()
				m.rawInput.Blur()
			} else {
				// Cycle between the two fields
				m.activeField = 1 - m.activeField
				if m.activeField == 0 {
					m.urlInput.Focus()
					m.rawInput.Blur()
				} else {
					m.urlInput.Blur()
					m.rawInput.Focus()
				}
			}
		case "shift+tab":
			if m.typing {
				m.activeField = 1 - m.activeField
				if m.activeField == 0 {
					m.urlInput.Focus()
					m.rawInput.Blur()
				} else {
					m.urlInput.Blur()
					m.rawInput.Focus()
				}
			}
		case "ctrl+s":
			m.typing = false
			m.urlInput.Blur()
			m.rawInput.Blur()
		case "ctrl+m":
			// Toggle between URL and raw input modes
			m.useRaw = !m.useRaw
		case "ctrl+r":
			if !m.isLoading {
				input := m.urlInput.Value()
				if m.useRaw {
					input = m.rawInput.Value()
				}
				if input != "" {
					m.isLoading = true
					cmds = append(cmds, m.runLeakScan(), m.spinner.Tick)
				}
			}
		case "enter":
			if m.typing && !m.isLoading {
				input := m.urlInput.Value()
				if m.useRaw {
					input = m.rawInput.Value()
				}
				if input != "" {
					m.isLoading = true
					cmds = append(cmds, m.runLeakScan(), m.spinner.Tick)
				}
			}
		case "g":
			m.responseVP.GotoTop()
		case "G":
			m.responseVP.GotoBottom()
		}

		if m.typing && !m.isLoading {
			if m.activeField == 0 {
				var cmd tea.Cmd
				m.urlInput, cmd = m.urlInput.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				var cmd tea.Cmd
				m.rawInput, cmd = m.rawInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the leaks view.
func (m LeaksModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	var sections []string

	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).Width(contentWidth - 4)
	inactiveStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(0, 1).Width(contentWidth - 4)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))

	// Mode toggle hint
	modeHint := "[Ctrl+R] Toggle mode: "
	if m.useRaw {
		modeHint += lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Render("Raw Text Mode")
	} else {
		modeHint += lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Render("URL Mode")
	}
	sections = append(sections, mutedStyle.Render(modeHint))
	sections = append(sections, "")

	if !m.useRaw {
		// URL input
		urlLabel := labelStyle.Render("Target URL")
		urlStyle := inactiveStyle
		if m.activeField == 0 {
			urlStyle = activeStyle
		}
		sections = append(sections, urlLabel)
		sections = append(sections, urlStyle.Render(m.urlInput.View()))
	} else {
		// Raw text input
		rawLabel := labelStyle.Render("Paste Raw Text / JSON to Scan")
		rawStyle := inactiveStyle
		if m.activeField == 1 {
			rawStyle = activeStyle
		}
		sections = append(sections, rawLabel)
		sections = append(sections, rawStyle.Render(m.rawInput.View()))
	}

	sections = append(sections, "")

	// Loading state
	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Scanning for leaks..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	// Run hint
	sections = append(sections, mutedStyle.Render("Press r or Enter to scan"))
	sections = append(sections, "")

	// Error state
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
		sections = append(sections, errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	// Results
	if len(m.results) > 0 {
		sections = append(sections, renderSectionTitle(fmt.Sprintf("Leaks Found: %d", len(m.results)), t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Group by severity
		groups := core.GroupLeaksBySeverity(m.results)
		severities := []core.Severity{core.SeverityCritical, core.SeverityHigh, core.SeverityMedium, core.SeverityLow}

		for _, sev := range severities {
			leaks, ok := groups[sev]
			if !ok || len(leaks) == 0 {
				continue
			}

			color := severityColor(sev, t)
			sevStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
			sections = append(sections, sevStyle.Render(fmt.Sprintf("  ● %s (%d)", sev.String(), len(leaks))))

			for _, leak := range leaks {
				typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))
				matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground))
				lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Comment))

				sections = append(sections, fmt.Sprintf("    %s  %s  %s",
					typeStyle.Render(fmt.Sprintf("%-28s", leak.Type)),
					matchStyle.Render(truncate(leak.Match, 40)),
					lineStyle.Render(fmt.Sprintf("line %d", leak.Line)),
				))
			}
			sections = append(sections, "")
		}

		// Response preview viewport
		if m.rawResponse != "" {
			sections = append(sections, renderSectionTitle("Response Preview", t))
			sections = append(sections, renderDivider(contentWidth-4, t))
			vpStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(t.Border)).
				Width(contentWidth - 4)
			sections = append(sections, vpStyle.Render(m.responseVP.View()))
		}
	} else if m.rawResponse != "" {
		safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
		sections = append(sections, safeStyle.Render("  ✓ No leaks detected!"))
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}

// GetLeaks returns detected leaks for stats.
func (m LeaksModel) GetLeaks() []core.DataLeak {
	return m.results
}
