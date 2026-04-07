package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// idorRunMsg carries IDOR test results.
type idorRunMsg struct {
	results []core.IDORResult
	err     error
}

// bypassRunMsg carries rate limit bypass results.
type bypassRunMsg struct {
	results []string
	err     error
}

// AuthModel is the Auth / IDOR view.
type AuthModel struct {
	urlInput     textinput.Model
	startIDInput textinput.Model
	endIDInput   textinput.Model
	headersInput textinput.Model
	mode         int // 0=IDOR, 1=RateLimit
	activeField  int // 0..3

	bypassResults []string
	idorResults   []core.IDORResult

	viewport  viewport.Model
	isLoading bool
	spinner   spinner.Model
	width     int
	height    int
	t         theme.Theme
	err       error
}

// NewAuth creates a new AuthModel.
func NewAuth() AuthModel {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://api.example.com/users/{id}"
	urlIn.CharLimit = 2048
	urlIn.Focus()

	startIn := textinput.New()
	startIn.Placeholder = "1"
	startIn.SetValue("1")
	startIn.CharLimit = 10

	endIn := textinput.New()
	endIn.Placeholder = "20"
	endIn.SetValue("20")
	endIn.CharLimit = 10

	headersIn := textinput.New()
	headersIn.Placeholder = "Authorization: Bearer TOKEN"
	headersIn.CharLimit = 1024

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return AuthModel{
		urlInput:     urlIn,
		startIDInput: startIn,
		endIDInput:   endIn,
		headersInput: headersIn,
		spinner:      sp,
		viewport:     vp,
		mode:         0,
		activeField:  0,
	}
}

// SetTheme updates the theme.
func (m *AuthModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *AuthModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.headersInput.Width = contentWidth - 4
	m.startIDInput.Width = 15
	m.endIDInput.Width = 15
	vpHeight := h - 24
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = contentWidth - 4
	m.viewport.Height = vpHeight
}

func (m AuthModel) runIDOR() tea.Cmd {
	return func() tea.Msg {
		url := m.urlInput.Value()
		if url == "" {
			return idorRunMsg{err: fmt.Errorf("URL is required")}
		}

		startID := 1
		if v, err := strconv.Atoi(m.startIDInput.Value()); err == nil && v >= 0 {
			startID = v
		}

		endID := 20
		if v, err := strconv.Atoi(m.endIDInput.Value()); err == nil && v >= 0 {
			endID = v
		}

		headers := core.ParseHeaders(m.headersInput.Value())
		results := core.TestIDOR(url, startID, endID, "GET", headers)
		return idorRunMsg{results: results}
	}
}

func (m AuthModel) runBypass() tea.Cmd {
	return func() tea.Msg {
		url := m.urlInput.Value()
		if url == "" {
			return bypassRunMsg{err: fmt.Errorf("URL is required")}
		}
		headers := core.ParseHeaders(m.headersInput.Value())
		results := core.TestRateLimitBypass(url, headers)
		return bypassRunMsg{results: results}
	}
}

func (m AuthModel) maxFields() int {
	if m.mode == 0 {
		return 4 // url, startID, endID, headers
	}
	return 2 // url, headers
}

func (m AuthModel) focusField(field int) AuthModel {
	m.urlInput.Blur()
	m.startIDInput.Blur()
	m.endIDInput.Blur()
	m.headersInput.Blur()

	if m.mode == 0 {
		switch field {
		case 0:
			m.urlInput.Focus()
		case 1:
			m.startIDInput.Focus()
		case 2:
			m.endIDInput.Focus()
		case 3:
			m.headersInput.Focus()
		}
	} else {
		switch field {
		case 0:
			m.urlInput.Focus()
		case 1:
			m.headersInput.Focus()
		}
	}
	return m
}

// Init implements tea.Model.
func (m AuthModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case idorRunMsg:
		m.isLoading = false
		m.idorResults = msg.results
		m.err = msg.err
		m.viewport.SetContent(m.buildIDORContent())

	case bypassRunMsg:
		m.isLoading = false
		m.bypassResults = msg.results
		m.err = msg.err
		m.viewport.SetContent(m.buildBypassContent())

	case spinner.TickMsg:
		if m.isLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeField = (m.activeField + 1) % m.maxFields()
			m = m.focusField(m.activeField)
		case "shift+tab":
			m.activeField = (m.activeField + m.maxFields() - 1) % m.maxFields()
			m = m.focusField(m.activeField)
		case "[":
			if m.mode > 0 {
				m.mode--
				m.activeField = 0
				m = m.focusField(0)
			}
		case "]":
			if m.mode < 1 {
				m.mode++
				m.activeField = 0
				m = m.focusField(0)
			}
		case "ctrl+r", "enter":
			if !m.isLoading {
				if m.urlInput.Value() != "" {
					m.isLoading = true
					m.idorResults = nil
					m.bypassResults = nil
					m.err = nil
					if m.mode == 0 {
						cmds = append(cmds, m.runIDOR(), m.spinner.Tick)
					} else {
						cmds = append(cmds, m.runBypass(), m.spinner.Tick)
					}
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

		// Delegate key to active input
		if !m.isLoading {
			var cmd tea.Cmd
			if m.mode == 0 {
				switch m.activeField {
				case 0:
					m.urlInput, cmd = m.urlInput.Update(msg)
				case 1:
					m.startIDInput, cmd = m.startIDInput.Update(msg)
				case 2:
					m.endIDInput, cmd = m.endIDInput.Update(msg)
				case 3:
					m.headersInput, cmd = m.headersInput.Update(msg)
				}
			} else {
				switch m.activeField {
				case 0:
					m.urlInput, cmd = m.urlInput.Update(msg)
				case 1:
					m.headersInput, cmd = m.headersInput.Update(msg)
				}
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AuthModel) buildIDORContent() string {
	if len(m.idorResults) == 0 {
		return "No results"
	}
	var lines []string
	accessible := 0
	for _, r := range m.idorResults {
		if r.Accessible {
			accessible++
		}
	}
	lines = append(lines, fmt.Sprintf("IDs tested: %d, Accessible: %d", len(m.idorResults), accessible))
	lines = append(lines, strings.Repeat("─", 60))
	lines = append(lines, fmt.Sprintf("%-6s  %-6s  %-8s  %-10s  Accessible", "ID", "HTTP", "Size", "Duration"))
	for _, r := range m.idorResults {
		acc := "no"
		if r.Accessible {
			acc = "YES"
		}
		lines = append(lines, fmt.Sprintf("%-6s  %-6d  %-8d  %-10s  %s",
			r.ID, r.StatusCode, r.Size, r.Duration.Round(1000000).String(), acc))
	}
	return strings.Join(lines, "\n")
}

func (m AuthModel) buildBypassContent() string {
	if len(m.bypassResults) == 0 {
		return "No bypass techniques found (or target not rate-limiting)"
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Bypass techniques that returned HTTP 200: %d", len(m.bypassResults)))
	lines = append(lines, strings.Repeat("─", 60))
	for _, r := range m.bypassResults {
		lines = append(lines, "  + "+r)
	}
	return strings.Join(lines, "\n")
}

// View renders the auth view.
func (m AuthModel) View(t theme.Theme) string {
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

	// Mode selector
	modes := []string{"IDOR Probe", "Rate Limit Bypass"}
	var modeParts []string
	for i, name := range modes {
		if i == m.mode {
			modeParts = append(modeParts, lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.Background)).
				Background(lipgloss.Color(t.Primary)).
				Bold(true).Padding(0, 1).
				Render(name))
		} else {
			modeParts = append(modeParts, lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.Muted)).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(t.Border)).
				Padding(0, 1).
				Render(name))
		}
	}
	sections = append(sections, labelStyle.Render("Mode")+" "+mutedStyle.Render("[ [ / ] to switch ]"))
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, modeParts...))
	sections = append(sections, "")

	// URL input
	urlBox := inactiveStyle
	if m.activeField == 0 {
		urlBox = activeStyle
	}
	urlLabel := "Target URL"
	if m.mode == 0 {
		urlLabel = "Target URL (use {id} placeholder)"
	}
	sections = append(sections, labelStyle.Render(urlLabel))
	sections = append(sections, urlBox.Render(m.urlInput.View()))
	sections = append(sections, "")

	if m.mode == 0 {
		// IDOR: show start/end ID inputs
		startBox := inactiveStyle.Width(20)
		endBox := inactiveStyle.Width(20)
		if m.activeField == 1 {
			startBox = activeStyle.Width(20)
		}
		if m.activeField == 2 {
			endBox = activeStyle.Width(20)
		}
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Start ID\n")+startBox.Render(m.startIDInput.View()),
			"  ",
			labelStyle.Render("End ID\n")+endBox.Render(m.endIDInput.View()),
		))
		sections = append(sections, "")
	}

	// Headers input
	headersIdx := 1
	if m.mode == 0 {
		headersIdx = 3
	}
	headersBox := inactiveStyle
	if m.activeField == headersIdx {
		headersBox = activeStyle
	}
	sections = append(sections, labelStyle.Render("Auth Header")+" "+mutedStyle.Render("(e.g. Authorization: Bearer TOKEN)"))
	sections = append(sections, headersBox.Render(m.headersInput.View()))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		msg := "Running IDOR probe..."
		if m.mode == 1 {
			msg = "Testing rate limit bypass..."
		}
		sections = append(sections, runStyle.Render(m.spinner.View()+" "+msg))
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

	if m.mode == 0 && len(m.idorResults) > 0 {
		accessible := 0
		for _, r := range m.idorResults {
			if r.Accessible {
				accessible++
			}
		}
		titleColor := t.Success
		if accessible > 0 {
			titleColor = t.Critical
		}
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(titleColor)).Bold(true)
		sections = append(sections, titleStyle.Render(fmt.Sprintf("IDOR Results: %d IDs, %d accessible", len(m.idorResults), accessible)))
		sections = append(sections, renderDivider(contentWidth-4, t))

		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Bold(true)
		sections = append(sections, headerStyle.Render(fmt.Sprintf("  %-6s  %-6s  %-8s  %-10s", "ID", "HTTP", "Size", "Duration")))

		for _, r := range m.idorResults {
			var rowStyle lipgloss.Style
			if r.Accessible {
				rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical)).Bold(true)
			} else {
				rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
			}
			acc := ""
			if r.Accessible {
				acc = " [ACCESSIBLE]"
			}
			sections = append(sections, rowStyle.Render(fmt.Sprintf("  %-6s  %-6d  %-8d  %-10s%s",
				r.ID, r.StatusCode, r.Size, r.Duration.Round(1000000).String(), acc)))
		}
	}

	if m.mode == 1 && m.bypassResults != nil {
		if len(m.bypassResults) == 0 {
			safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
			sections = append(sections, safeStyle.Render("No rate limit bypass techniques found"))
		} else {
			titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical)).Bold(true)
			sections = append(sections, titleStyle.Render(fmt.Sprintf("Bypass Techniques Found: %d", len(m.bypassResults))))
			sections = append(sections, renderDivider(contentWidth-4, t))
			for _, r := range m.bypassResults {
				hitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
				sections = append(sections, hitStyle.Render("  + "+r))
			}
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
