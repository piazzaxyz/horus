package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// analyzerRunMsg carries a completed HTTP response.
type analyzerRunMsg struct {
	resp      *core.Response
	leaks     []core.DataLeak
	secIssues []core.SecurityIssue
}

// AnalyzerModel is the HTTP Analyzer view.
type AnalyzerModel struct {
	urlInput     textinput.Model
	methodInput  textinput.Model
	headersInput textarea.Model
	bodyInput    textarea.Model
	responseVP   viewport.Model
	activeField  int
	isLoading    bool
	spinner      spinner.Model
	result       *core.Response
	leaks        []core.DataLeak
	secIssues    []core.SecurityIssue
	width        int
	height       int
	t            theme.Theme
}

// NewAnalyzer creates a new AnalyzerModel.
func NewAnalyzer() AnalyzerModel {
	url := textinput.New()
	url.Placeholder = "https://api.example.com/endpoint"
	url.Focus()
	url.CharLimit = 2048

	method := textinput.New()
	method.Placeholder = "GET"
	method.SetValue("GET")
	method.CharLimit = 10

	headers := textarea.New()
	headers.Placeholder = "Content-Type: application/json\nAuthorization: Bearer <token>"
	headers.SetHeight(4)
	headers.CharLimit = 4096

	body := textarea.New()
	body.Placeholder = `{"key": "value"}`
	body.SetHeight(4)
	body.CharLimit = 16384

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))

	vp := viewport.New(80, 10)

	return AnalyzerModel{
		urlInput:     url,
		methodInput:  method,
		headersInput: headers,
		bodyInput:    body,
		responseVP:   vp,
		spinner:      sp,
		activeField:  0,
	}
}

// SetTheme updates the theme.
func (m *AnalyzerModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *AnalyzerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.methodInput.Width = 10
	m.headersInput.SetWidth(contentWidth - 4)
	m.bodyInput.SetWidth(contentWidth - 4)

	vpHeight := h - 30
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.responseVP.Width = contentWidth - 4
	m.responseVP.Height = vpHeight
}

// runAnalyzer executes the HTTP request and returns a command.
func (m AnalyzerModel) runAnalyzer() tea.Cmd {
	return func() tea.Msg {
		client := core.NewHTTPClient()
		req := core.Request{
			Method:  m.methodInput.Value(),
			URL:     m.urlInput.Value(),
			Headers: core.ParseHeaders(m.headersInput.Value()),
			Body:    m.bodyInput.Value(),
		}
		if req.Method == "" {
			req.Method = "GET"
		}
		resp := client.Execute(req)

		var leaks []core.DataLeak
		var secIssues []core.SecurityIssue

		if resp.Error == nil {
			leaks = core.ScanForLeaks(resp.Body)
			scanResult := core.RunSecurityScan(req.URL)
			secIssues = scanResult.Issues
		}

		return analyzerRunMsg{resp: resp, leaks: leaks, secIssues: secIssues}
	}
}

// Init implements tea.Model.
func (m AnalyzerModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m AnalyzerModel) Update(msg tea.Msg) (AnalyzerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case analyzerRunMsg:
		m.isLoading = false
		m.result = msg.resp
		m.leaks = msg.leaks
		m.secIssues = msg.secIssues
		m.responseVP.SetContent(core.FormatResponse(msg.resp))
		return m, nil

	case spinner.TickMsg:
		if m.isLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeField = (m.activeField + 1) % 4
			m.focusField()
		case "shift+tab":
			m.activeField = (m.activeField + 3) % 4
			m.focusField()
		case "enter", "r":
			if m.activeField != 2 && m.activeField != 3 { // not in textarea
				url := m.urlInput.Value()
				if url != "" && !m.isLoading {
					m.isLoading = true
					cmds = append(cmds, m.runAnalyzer(), m.spinner.Tick)
				}
			}
		case "ctrl+enter":
			url := m.urlInput.Value()
			if url != "" && !m.isLoading {
				m.isLoading = true
				cmds = append(cmds, m.runAnalyzer(), m.spinner.Tick)
			}
		case "g":
			if m.activeField == 4 {
				m.responseVP.GotoTop()
			}
		case "G":
			if m.activeField == 4 {
				m.responseVP.GotoBottom()
			}
		}
	}

	// Update active field
	switch m.activeField {
	case 0:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		var cmd tea.Cmd
		m.methodInput, cmd = m.methodInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		var cmd tea.Cmd
		m.headersInput, cmd = m.headersInput.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		var cmd tea.Cmd
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	case 4:
		var cmd tea.Cmd
		m.responseVP, cmd = m.responseVP.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AnalyzerModel) focusField() {
	m.urlInput.Blur()
	m.methodInput.Blur()
	m.headersInput.Blur()
	m.bodyInput.Blur()

	switch m.activeField {
	case 0:
		m.urlInput.Focus()
	case 1:
		m.methodInput.Focus()
	case 2:
		m.headersInput.Focus()
	case 3:
		m.bodyInput.Focus()
	}
}

// View renders the analyzer.
func (m AnalyzerModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}

	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).
		Width(contentWidth - 2)

	inactiveStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(0, 1).
		Width(contentWidth - 2)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Accent)).
		Bold(true)

	var sections []string

	// URL row
	urlLabel := labelStyle.Render("[Tab] URL")
	urlBox := inactiveStyle
	if m.activeField == 0 {
		urlBox = activeStyle
		urlLabel = labelStyle.Render("[Tab] URL  ●")
	}
	sections = append(sections, urlLabel)
	sections = append(sections, urlBox.Render(m.urlInput.View()))

	// Method + Run row
	methodLabel := labelStyle.Render("Method")
	methodBox := inactiveStyle.Width(14)
	if m.activeField == 1 {
		methodBox = activeStyle.Width(14)
		methodLabel = labelStyle.Render("Method  ●")
	}

	var runPart string
	if m.isLoading {
		runPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Render(
			m.spinner.View() + " Sending request...")
	} else {
		runHint := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Render("  Press r or Enter (when URL focused) to run")
		runPart = runHint
	}

	methodRow := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, methodLabel, methodBox.Render(m.methodInput.View())),
		"  ",
		lipgloss.NewStyle().PaddingTop(1).Render(runPart),
	)
	sections = append(sections, "")
	sections = append(sections, methodRow)

	// Headers
	headersLabel := labelStyle.Render("Headers (key: value, one per line)")
	headersBox := inactiveStyle
	if m.activeField == 2 {
		headersBox = activeStyle
		headersLabel = labelStyle.Render("Headers  ●")
	}
	sections = append(sections, "")
	sections = append(sections, headersLabel)
	sections = append(sections, headersBox.Render(m.headersInput.View()))

	// Body
	bodyLabel := labelStyle.Render("Request Body")
	bodyBox := inactiveStyle
	if m.activeField == 3 {
		bodyBox = activeStyle
		bodyLabel = labelStyle.Render("Request Body  ●")
	}
	sections = append(sections, "")
	sections = append(sections, bodyLabel)
	sections = append(sections, bodyBox.Render(m.bodyInput.View()))

	// Response
	if m.result != nil || m.isLoading {
		sections = append(sections, "")
		sections = append(sections, renderSectionTitle("Response", t))
		sections = append(sections, renderDivider(contentWidth-2, t))

		if m.result != nil {
			// Status line
			statusColorHex := statusColor(m.result.StatusCode, t)
			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColorHex)).Bold(true)
			durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
			statusLine := fmt.Sprintf("%s  %s",
				statusStyle.Render(m.result.Status),
				durationStyle.Render(m.result.Duration.String()),
			)
			sections = append(sections, statusLine)

			// Leaks summary
			if len(m.leaks) > 0 {
				leakStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)
				sections = append(sections, leakStyle.Render(
					fmt.Sprintf("  ⚠ %s", core.LeakSummary(m.leaks))))
			} else if m.result.Error == nil {
				safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
				sections = append(sections, safeStyle.Render("  ✓ No leaks detected in response"))
			}

			// Response viewport
			vpStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(t.Border)).
				Width(contentWidth - 2)
			sections = append(sections, vpStyle.Render(m.responseVP.View()))
		}
	}

	outerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 2)

	return outerStyle.Render(strings.Join(sections, "\n"))
}

// GetResult returns the latest result for app-level stats tracking.
func (m AnalyzerModel) GetResult() (*core.Response, []core.DataLeak, []core.SecurityIssue) {
	return m.result, m.leaks, m.secIssues
}
