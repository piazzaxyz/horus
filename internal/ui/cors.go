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

// corsRunMsg carries CORS test results.
type corsRunMsg struct {
	results []core.CORSResult
	err     error
}

// CORSModel is the CORS Tester view.
type CORSModel struct {
	urlInput  textinput.Model
	results   []core.CORSResult
	viewport  viewport.Model
	isLoading bool
	spinner   spinner.Model
	width     int
	height    int
	t         theme.Theme
	err       error
}

// NewCORS creates a new CORSModel.
func NewCORS() CORSModel {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://api.example.com/endpoint"
	urlIn.CharLimit = 2048
	urlIn.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return CORSModel{
		urlInput: urlIn,
		spinner:  sp,
		viewport: vp,
	}
}

// SetTheme updates the theme.
func (m *CORSModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *CORSModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	vpHeight := h - 18
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = contentWidth - 4
	m.viewport.Height = vpHeight
}

// IsTyping returns true when the URL input is focused (always, as it's the only input).
func (m CORSModel) IsTyping() bool {
	return true
}

func (m CORSModel) runCORS() tea.Cmd {
	return func() tea.Msg {
		url := m.urlInput.Value()
		if url == "" {
			return corsRunMsg{err: fmt.Errorf("URL is required")}
		}
		results := core.TestCORS(url)
		return corsRunMsg{results: results}
	}
}

// Init implements tea.Model.
func (m CORSModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m CORSModel) Update(msg tea.Msg) (CORSModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case corsRunMsg:
		m.isLoading = false
		m.results = msg.results
		m.err = msg.err
		m.viewport.SetContent(m.buildResultsContent())

	case spinner.TickMsg:
		if m.isLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r", "enter":
			if !m.isLoading {
				if m.urlInput.Value() != "" {
					m.isLoading = true
					m.results = nil
					m.err = nil
					cmds = append(cmds, m.runCORS(), m.spinner.Tick)
				}
			}
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		}

		if !m.isLoading {
			var cmd tea.Cmd
			m.urlInput, cmd = m.urlInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m CORSModel) buildResultsContent() string {
	if len(m.results) == 0 {
		return "No results"
	}
	var lines []string
	vulnCount := 0
	for _, r := range m.results {
		if r.Vulnerable {
			vulnCount++
		}
	}
	lines = append(lines, fmt.Sprintf("Tests run: %d, Vulnerable: %d", len(m.results), vulnCount))
	lines = append(lines, strings.Repeat("─", 60))
	for _, r := range m.results {
		status := "SAFE"
		if r.Vulnerable {
			status = "VULN"
		}
		line := fmt.Sprintf("%-4s  Origin: %-35s  ACAO: %-30s  Creds: %v",
			status, r.TestedOrigin, r.AllowedOrigin, r.AllowCredentials)
		if r.Vulnerable {
			line += "\n       VulnType: " + r.VulnType
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// View renders the CORS tester view.
func (m CORSModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	var sections []string

	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).Width(contentWidth - 4)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))

	sections = append(sections, labelStyle.Render("Target URL"))
	sections = append(sections, activeStyle.Render(m.urlInput.View()))
	sections = append(sections, "")
	sections = append(sections, mutedStyle.Render("Tests: arbitrary origin, null origin, subdomain wildcard, prefix match"))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Testing CORS configurations..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	sections = append(sections, mutedStyle.Render("Press r or Enter to run"))
	sections = append(sections, "")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
		sections = append(sections, errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	if len(m.results) > 0 {
		vulnCount := 0
		for _, r := range m.results {
			if r.Vulnerable {
				vulnCount++
			}
		}

		titleColor := t.Success
		if vulnCount > 0 {
			titleColor = t.Critical
		}
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(titleColor)).Bold(true)
		sections = append(sections, titleStyle.Render(fmt.Sprintf("CORS Results: %d tests, %d vulnerable", len(m.results), vulnCount)))
		sections = append(sections, renderDivider(contentWidth-4, t))

		for _, r := range m.results {
			var statusStyle lipgloss.Style
			statusStr := "SAFE"
			if r.Vulnerable {
				statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical)).Bold(true)
				statusStr = "VULN"
			} else {
				statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
			}

			originStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground))
			acacStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))

			acao := r.AllowedOrigin
			if acao == "" {
				acao = "(none)"
			}

			sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
				statusStyle.Render(fmt.Sprintf("%-4s ", statusStr)),
				originStyle.Render(fmt.Sprintf("%-35s  ", truncate(r.TestedOrigin, 35))),
				acacStyle.Render(fmt.Sprintf("ACAO: %-30s", truncate(acao, 30))),
			))

			if r.AllowCredentials {
				credStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
				sections = append(sections, credStyle.Render("      credentials: true"))
			}

			if r.Vulnerable && r.VulnType != "" {
				vtStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical))
				sections = append(sections, vtStyle.Render("      "+r.VulnType))
			}
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
