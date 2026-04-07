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

// fuzzerRunMsg carries fuzzing results.
type fuzzerRunMsg struct {
	results []core.FuzzResult
	err     error
}

// FuzzerModel is the Fuzzer view.
type FuzzerModel struct {
	urlInput         textinput.Model
	customWordlist   textinput.Model
	concurrencyInput textinput.Model
	results          []core.FuzzResult
	viewport         viewport.Model
	isLoading        bool
	spinner          spinner.Model
	activeField      int // 0=url, 1=wordlist, 2=concurrency
	width            int
	height           int
	t                theme.Theme
	err              error
}

// NewFuzzer creates a new FuzzerModel.
func NewFuzzer() FuzzerModel {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://api.example.com"
	urlIn.CharLimit = 2048
	urlIn.Focus()

	wordlistIn := textinput.New()
	wordlistIn.Placeholder = "custom/path,another/path (leave blank for built-in)"
	wordlistIn.CharLimit = 4096

	concurrencyIn := textinput.New()
	concurrencyIn.Placeholder = "10"
	concurrencyIn.SetValue("10")
	concurrencyIn.CharLimit = 4

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return FuzzerModel{
		urlInput:         urlIn,
		customWordlist:   wordlistIn,
		concurrencyInput: concurrencyIn,
		spinner:          sp,
		viewport:         vp,
		activeField:      0,
	}
}

// SetTheme updates the theme.
func (m *FuzzerModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *FuzzerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.urlInput.Width = contentWidth - 4
	m.customWordlist.Width = contentWidth - 4
	m.concurrencyInput.Width = 10
	vpHeight := h - 26
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = contentWidth - 4
	m.viewport.Height = vpHeight
}

func (m FuzzerModel) runFuzz() tea.Cmd {
	return func() tea.Msg {
		baseURL := m.urlInput.Value()
		if baseURL == "" {
			return fuzzerRunMsg{err: fmt.Errorf("URL is required")}
		}

		// Parse custom paths
		var customPaths []string
		if raw := m.customWordlist.Value(); raw != "" {
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					customPaths = append(customPaths, p)
				}
			}
		}

		concurrency := 10
		if v, err := strconv.Atoi(m.concurrencyInput.Value()); err == nil && v > 0 {
			concurrency = v
		}

		results := core.RunFuzz(baseURL, customPaths, concurrency)
		return fuzzerRunMsg{results: results}
	}
}

// Init implements tea.Model.
func (m FuzzerModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m FuzzerModel) Update(msg tea.Msg) (FuzzerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case fuzzerRunMsg:
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
			m.activeField = (m.activeField + 1) % 3
			m.urlInput.Blur()
			m.customWordlist.Blur()
			m.concurrencyInput.Blur()
			switch m.activeField {
			case 0:
				m.urlInput.Focus()
			case 1:
				m.customWordlist.Focus()
			case 2:
				m.concurrencyInput.Focus()
			}
		case "shift+tab":
			m.activeField = (m.activeField + 2) % 3
			m.urlInput.Blur()
			m.customWordlist.Blur()
			m.concurrencyInput.Blur()
			switch m.activeField {
			case 0:
				m.urlInput.Focus()
			case 1:
				m.customWordlist.Focus()
			case 2:
				m.concurrencyInput.Focus()
			}
		case "ctrl+r", "enter":
			if !m.isLoading {
				if m.urlInput.Value() != "" {
					m.isLoading = true
					m.results = nil
					m.err = nil
					cmds = append(cmds, m.runFuzz(), m.spinner.Tick)
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
			switch m.activeField {
			case 0:
				m.urlInput, cmd = m.urlInput.Update(msg)
			case 1:
				m.customWordlist, cmd = m.customWordlist.Update(msg)
			case 2:
				m.concurrencyInput, cmd = m.concurrencyInput.Update(msg)
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m FuzzerModel) buildResultsContent() string {
	if len(m.results) == 0 {
		return "No results"
	}
	var lines []string
	found := 0
	for _, r := range m.results {
		if r.Found {
			found++
		}
	}
	lines = append(lines, fmt.Sprintf("Probed: %d paths, Found: %d", len(m.results), found))
	lines = append(lines, strings.Repeat("─", 60))

	for _, r := range m.results {
		if !r.Found {
			continue
		}
		line := fmt.Sprintf("HTTP %-3d  %-8d bytes  %-10s  %s",
			r.StatusCode,
			r.Size,
			r.Duration.Round(1000000).String(),
			r.Path,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// View renders the fuzzer view.
func (m FuzzerModel) View(t theme.Theme) string {
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
	urlBox := inactiveStyle
	if m.activeField == 0 {
		urlBox = activeStyle
	}
	sections = append(sections, labelStyle.Render("Base URL"))
	sections = append(sections, urlBox.Render(m.urlInput.View()))
	sections = append(sections, "")

	// Custom wordlist
	wlBox := inactiveStyle
	if m.activeField == 1 {
		wlBox = activeStyle
	}
	sections = append(sections, labelStyle.Render("Custom Paths")+" "+mutedStyle.Render("(comma-separated, leave blank for built-in)"))
	sections = append(sections, wlBox.Render(m.customWordlist.View()))
	sections = append(sections, "")

	// Concurrency
	concBox := inactiveStyle.Width(20)
	if m.activeField == 2 {
		concBox = activeStyle.Width(20)
	}
	sections = append(sections, labelStyle.Render("Concurrency"))
	sections = append(sections, concBox.Render(m.concurrencyInput.View()))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Fuzzing paths... (this may take a while)"))
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
		found := 0
		for _, r := range m.results {
			if r.Found {
				found++
			}
		}
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
		sections = append(sections, titleStyle.Render(fmt.Sprintf("Paths Found: %d / %d probed", found, len(m.results))))
		sections = append(sections, renderDivider(contentWidth-4, t))

		// Header
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Bold(true)
		sections = append(sections, headerStyle.Render(fmt.Sprintf("  %-6s  %-8s  %-10s  Path", "HTTP", "Size", "Duration")))

		for _, r := range m.results {
			if !r.Found {
				continue
			}
			codeColor := statusColor(r.StatusCode, t)
			codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(codeColor)).Bold(true)
			line := fmt.Sprintf("  %-8d  %-10s  %s",
				r.Size,
				r.Duration.Round(1000000).String(),
				r.Path,
			)
			sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
				codeStyle.Render(fmt.Sprintf("  %-6d", r.StatusCode)),
				lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground)).Render(line),
			))
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
