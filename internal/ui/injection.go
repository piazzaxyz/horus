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

// injectionRunMsg carries injection test results.
type injectionRunMsg struct {
	results []core.InjectionResult
	err     error
}

// InjectionModel is the Injection Tester view.
type InjectionModel struct {
	urlInput    textinput.Model
	paramInput  textinput.Model
	injType     core.InjectionType
	results     []core.InjectionResult
	viewport    viewport.Model
	isLoading   bool
	spinner     spinner.Model
	activeField int // 0=url, 1=param
	typing      bool
	width       int
	height      int
	t           theme.Theme
	err         error
}

// NewInjection creates a new InjectionModel.
func NewInjection() InjectionModel {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://api.example.com/search"
	urlIn.CharLimit = 2048

	paramIn := textinput.New()
	paramIn.Placeholder = "q"
	paramIn.CharLimit = 256

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return InjectionModel{
		urlInput:    urlIn,
		paramInput:  paramIn,
		spinner:     sp,
		viewport:    vp,
		injType:     core.InjectionSQLi,
		activeField: 0,
	}
}

// SetTheme updates the theme.
func (m *InjectionModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *InjectionModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.paramInput.Width = contentWidth - 4
	vpHeight := h - 22
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = contentWidth - 4
	m.viewport.Height = vpHeight
}

// IsTyping returns true when in input editing mode.
func (m InjectionModel) IsTyping() bool {
	return m.typing
}

func (m InjectionModel) runInjection() tea.Cmd {
	return func() tea.Msg {
		url := m.urlInput.Value()
		param := m.paramInput.Value()
		if url == "" {
			return injectionRunMsg{err: fmt.Errorf("URL is required")}
		}
		if param == "" {
			param = "q"
		}
		results := core.RunInjectionTest(url, param, m.injType)
		return injectionRunMsg{results: results}
	}
}

// Init implements tea.Model.
func (m InjectionModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m InjectionModel) Update(msg tea.Msg) (InjectionModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case injectionRunMsg:
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
		case "tab":
			if !m.typing {
				m.typing = true
				m.activeField = 0
				m.urlInput.Focus()
				m.paramInput.Blur()
			} else {
				m.activeField = (m.activeField + 1) % 2
				if m.activeField == 0 {
					m.urlInput.Focus()
					m.paramInput.Blur()
				} else {
					m.urlInput.Blur()
					m.paramInput.Focus()
				}
			}
		case "shift+tab":
			if m.typing {
				m.activeField = (m.activeField + 1) % 2
				if m.activeField == 0 {
					m.urlInput.Focus()
					m.paramInput.Blur()
				} else {
					m.urlInput.Blur()
					m.paramInput.Focus()
				}
			}
		case "ctrl+s":
			m.typing = false
			m.urlInput.Blur()
			m.paramInput.Blur()
		case "[":
			// Cycle injection type backwards
			if m.injType > core.InjectionSQLi {
				m.injType--
			} else {
				m.injType = core.InjectionCmdInjection
			}
		case "]":
			// Cycle injection type forwards
			if m.injType < core.InjectionCmdInjection {
				m.injType++
			} else {
				m.injType = core.InjectionSQLi
			}
		case "ctrl+r":
			if !m.isLoading && m.urlInput.Value() != "" {
				m.isLoading = true
				m.results = nil
				m.err = nil
				cmds = append(cmds, m.runInjection(), m.spinner.Tick)
			}
		case "enter":
			if m.typing && !m.isLoading && m.urlInput.Value() != "" {
				m.isLoading = true
				m.results = nil
				m.err = nil
				cmds = append(cmds, m.runInjection(), m.spinner.Tick)
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

		if m.typing && !m.isLoading {
			var cmd tea.Cmd
			if m.activeField == 0 {
				m.urlInput, cmd = m.urlInput.Update(msg)
			} else {
				m.paramInput, cmd = m.paramInput.Update(msg)
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m InjectionModel) buildResultsContent() string {
	if len(m.results) == 0 {
		return ""
	}

	var lines []string
	vulnCount := 0
	for _, r := range m.results {
		if r.Vulnerable {
			vulnCount++
		}
	}

	lines = append(lines, fmt.Sprintf("Results: %d payloads tested, %d potentially vulnerable", len(m.results), vulnCount))
	lines = append(lines, strings.Repeat("─", 60))

	for _, r := range m.results {
		status := "SAFE  "
		if r.Vulnerable {
			status = "VULN  "
		}
		payload := truncate(r.Payload, 30)
		evidence := ""
		if r.Evidence != "" {
			evidence = " | " + truncate(r.Evidence, 40)
		}
		line := fmt.Sprintf("%-6s  %-12s  HTTP %-3d  %-8s  %s%s",
			status,
			r.Type.String(),
			r.StatusCode,
			r.Confidence,
			payload,
			evidence,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// View renders the injection view.
func (m InjectionModel) View(t theme.Theme) string {
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

	// URL input
	urlLabel := labelStyle.Render("Target URL")
	urlBox := inactiveStyle
	if m.activeField == 0 {
		urlBox = activeStyle
	}
	sections = append(sections, urlLabel)
	sections = append(sections, urlBox.Render(m.urlInput.View()))
	sections = append(sections, "")

	// Param input
	paramLabel := labelStyle.Render("Parameter to Inject")
	paramBox := inactiveStyle
	if m.activeField == 1 {
		paramBox = activeStyle
	}
	sections = append(sections, paramLabel)
	sections = append(sections, paramBox.Render(m.paramInput.View()))
	sections = append(sections, "")

	// Injection type selector
	injTypes := []core.InjectionType{
		core.InjectionSQLi,
		core.InjectionXSS,
		core.InjectionSSTI,
		core.InjectionPathTraversal,
		core.InjectionCmdInjection,
	}
	var typeParts []string
	for _, it := range injTypes {
		s := it.String()
		if it == m.injType {
			typeParts = append(typeParts, lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.Background)).
				Background(lipgloss.Color(t.Primary)).
				Bold(true).
				Padding(0, 1).
				Render(s))
		} else {
			typeParts = append(typeParts, lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.Muted)).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(t.Border)).
				Padding(0, 1).
				Render(s))
		}
	}
	sections = append(sections, labelStyle.Render("Injection Type")+" "+mutedStyle.Render("[ [ / ] to cycle ]"))
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, typeParts...))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Running injection tests..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	sections = append(sections, mutedStyle.Render("Press r or Enter to run  |  [ / ] to cycle type"))
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
		sections = append(sections, titleStyle.Render(fmt.Sprintf("Results: %d tested, %d vulnerable", len(m.results), vulnCount)))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Render results in viewport
		vpStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Border)).
			Width(contentWidth - 4)

		// Header row
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Bold(true)
		sections = append(sections, headerStyle.Render(fmt.Sprintf("%-6s  %-12s  %-8s  %-8s  %-30s  Evidence",
			"Status", "Type", "HTTP", "Conf", "Payload")))

		for _, r := range m.results {
			vulnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
			statusStr := "SAFE"
			if r.Vulnerable {
				vulnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical)).Bold(true)
				statusStr = "VULN"
			}

			evidence := truncate(r.Evidence, 35)
			payload := truncate(r.Payload, 28)
			line := fmt.Sprintf("  %-12s  HTTP %-3d  %-8s  %-28s  %s",
				r.Type.String(),
				r.StatusCode,
				r.Confidence,
				payload,
				evidence,
			)
			sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
				vulnStyle.Render(fmt.Sprintf("%-6s", statusStr)),
				lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground)).Render(line),
			))
		}

		_ = vpStyle
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
