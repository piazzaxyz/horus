package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// JWTModel is the JWT Analyzer view.
type JWTModel struct {
	tokenInput  textarea.Model
	analysis    *core.JWTAnalysis
	resultVP    viewport.Model
	activeField int // 0=input
	width       int
	height      int
	t           theme.Theme
	err         error
}

// NewJWT creates a new JWTModel.
func NewJWT() JWTModel {
	ta := textarea.New()
	ta.Placeholder = "Paste your JWT token here (eyJ...)"
	ta.CharLimit = 8192
	ta.SetWidth(80)
	ta.SetHeight(4)
	ta.Focus()

	vp := viewport.New(80, 10)

	return JWTModel{
		tokenInput:  ta,
		resultVP:    vp,
		activeField: 0,
	}
}

// SetTheme updates the theme.
func (m *JWTModel) SetTheme(t theme.Theme) {
	m.t = t
}

// SetSize updates dimensions.
func (m *JWTModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.tokenInput.SetWidth(contentWidth - 6)
	vpHeight := h - 20
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.resultVP.Width = contentWidth - 4
	m.resultVP.Height = vpHeight
}

// IsTyping returns true when the textarea input is focused.
func (m JWTModel) IsTyping() bool {
	return m.activeField == 0 // textarea for JWT token
}

// Init implements tea.Model.
func (m JWTModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages.
func (m JWTModel) Update(msg tea.Msg) (JWTModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r", "enter":
			// Analyze the token
			token := strings.TrimSpace(m.tokenInput.Value())
			if token != "" {
				analysis := core.AnalyzeJWT(token)
				m.analysis = &analysis
				m.err = nil
				m.resultVP.SetContent(m.buildAnalysisContent())
			}
		case "ctrl+enter":
			// Also analyze on ctrl+enter (enter is captured by textarea)
			token := strings.TrimSpace(m.tokenInput.Value())
			if token != "" {
				analysis := core.AnalyzeJWT(token)
				m.analysis = &analysis
				m.err = nil
				m.resultVP.SetContent(m.buildAnalysisContent())
			}
		case "g":
			m.resultVP.GotoTop()
		case "G":
			m.resultVP.GotoBottom()
		case "j", "down":
			if m.analysis != nil {
				m.resultVP.LineDown(1)
			} else {
				var cmd tea.Cmd
				m.tokenInput, cmd = m.tokenInput.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "k", "up":
			if m.analysis != nil {
				m.resultVP.LineUp(1)
			} else {
				var cmd tea.Cmd
				m.tokenInput, cmd = m.tokenInput.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "esc":
			// Clear analysis to go back to input
			m.analysis = nil
		default:
			var cmd tea.Cmd
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func prettyJSON(v map[string]interface{}) string {
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return "  " + string(b)
}

func (m JWTModel) buildAnalysisContent() string {
	if m.analysis == nil {
		return ""
	}
	a := m.analysis
	var lines []string

	lines = append(lines, "=== HEADER ===")
	lines = append(lines, prettyJSON(a.Header))
	lines = append(lines, "")

	lines = append(lines, "=== PAYLOAD ===")
	lines = append(lines, prettyJSON(a.Payload))
	lines = append(lines, "")

	lines = append(lines, "=== ANALYSIS ===")
	lines = append(lines, fmt.Sprintf("Algorithm: %s", a.Algorithm))

	if a.ExpiresAt != nil {
		expStr := a.ExpiresAt.String()
		if a.IsExpired {
			expStr += " [EXPIRED]"
		} else {
			expStr += " [VALID]"
		}
		lines = append(lines, fmt.Sprintf("Expires:   %s", expStr))
	}

	if a.IssuedAt != nil {
		lines = append(lines, fmt.Sprintf("Issued:    %s", a.IssuedAt.String()))
	}
	lines = append(lines, "")

	if len(a.Vulnerabilities) > 0 {
		lines = append(lines, fmt.Sprintf("=== VULNERABILITIES (%d) ===", len(a.Vulnerabilities)))
		for _, v := range a.Vulnerabilities {
			lines = append(lines, "  ! "+v)
		}
	} else {
		lines = append(lines, "=== No critical vulnerabilities found ===")
	}

	return strings.Join(lines, "\n")
}

// View renders the JWT analyzer view.
func (m JWTModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	var sections []string

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).
		Width(contentWidth - 4)

	sections = append(sections, labelStyle.Render("JWT Token"))
	sections = append(sections, inputBoxStyle.Render(m.tokenInput.View()))
	sections = append(sections, "")
	sections = append(sections, mutedStyle.Render("Press r to analyze  |  Esc to clear results  |  j/k to scroll"))
	sections = append(sections, "")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
		sections = append(sections, errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	if m.analysis != nil {
		a := m.analysis
		sections = append(sections, renderSectionTitle("JWT Analysis", t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Algorithm
		algColor := t.Success
		if strings.ToLower(a.Algorithm) == "none" {
			algColor = t.Critical
		}
		algStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(algColor)).Bold(true)
		sections = append(sections, fmt.Sprintf("Algorithm: %s", algStyle.Render(a.Algorithm)))

		// Expiry
		if a.IsExpired {
			expStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)
			sections = append(sections, expStyle.Render("Status:    EXPIRED"))
		} else if a.ExpiresAt != nil {
			okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
			sections = append(sections, okStyle.Render(fmt.Sprintf("Status:    Valid (expires %s)", a.ExpiresAt.String())))
		} else {
			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
			sections = append(sections, warnStyle.Render("Status:    No expiry (never expires)"))
		}
		sections = append(sections, "")

		// Vulnerabilities
		if len(a.Vulnerabilities) > 0 {
			vulnTitle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Critical)).Bold(true)
			sections = append(sections, vulnTitle.Render(fmt.Sprintf("Vulnerabilities: %d found", len(a.Vulnerabilities))))
			for _, v := range a.Vulnerabilities {
				vulnColor := t.Warning
				if strings.HasPrefix(v, "CRITICAL") {
					vulnColor = t.Critical
				} else if strings.HasPrefix(v, "INFO") {
					vulnColor = t.Info
				}
				vStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(vulnColor))
				sections = append(sections, vStyle.Render("  ! "+v))
			}
		} else {
			okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
			sections = append(sections, okStyle.Render("No critical vulnerabilities detected"))
		}
		sections = append(sections, "")

		// Decoded header and payload in viewport
		sections = append(sections, renderSectionTitle("Decoded Token", t))
		sections = append(sections, renderDivider(contentWidth-4, t))

		headerJSON := prettyJSON(a.Header)
		payloadJSON := prettyJSON(a.Payload)

		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info)).Bold(true)
		fgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground))

		sections = append(sections, infoStyle.Render("Header:"))
		sections = append(sections, fgStyle.Render(headerJSON))
		sections = append(sections, "")
		sections = append(sections, infoStyle.Render("Payload:"))
		sections = append(sections, fgStyle.Render(payloadJSON))
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
