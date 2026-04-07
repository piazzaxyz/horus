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

// portScanRunMsg carries port scan results.
type portScanRunMsg struct {
	results []core.PortResult
	host    string
	err     error
}

// PortScanModel is the Port Scanner view.
type PortScanModel struct {
	hostInput      textinput.Model
	startPortInput textinput.Model
	endPortInput   textinput.Model
	results        []core.PortResult
	scannedHost    string
	viewport       viewport.Model
	isLoading      bool
	spinner        spinner.Model
	activeField    int // 0=host, 1=startPort, 2=endPort
	width          int
	height         int
	t              theme.Theme
	err            error
}

// NewPortScan creates a new PortScanModel.
func NewPortScan() PortScanModel {
	hostIn := textinput.New()
	hostIn.Placeholder = "192.168.1.1 or example.com"
	hostIn.CharLimit = 256
	hostIn.Focus()

	startIn := textinput.New()
	startIn.Placeholder = "1"
	startIn.SetValue("1")
	startIn.CharLimit = 6

	endIn := textinput.New()
	endIn.Placeholder = "1024"
	endIn.SetValue("1024")
	endIn.CharLimit = 6

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 10)

	return PortScanModel{
		hostInput:      hostIn,
		startPortInput: startIn,
		endPortInput:   endIn,
		spinner:        sp,
		viewport:       vp,
		activeField:    0,
	}
}

// SetTheme updates the theme.
func (m *PortScanModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
}

// SetSize updates dimensions.
func (m *PortScanModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.hostInput.Width = contentWidth - 4
	m.startPortInput.Width = 10
	m.endPortInput.Width = 10
	vpHeight := h - 20
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = contentWidth - 4
	m.viewport.Height = vpHeight
}

func (m PortScanModel) runScan() tea.Cmd {
	return func() tea.Msg {
		host := m.hostInput.Value()
		if host == "" {
			return portScanRunMsg{err: fmt.Errorf("host is required")}
		}

		startPort := 1
		if v, err := strconv.Atoi(m.startPortInput.Value()); err == nil && v > 0 {
			startPort = v
		}

		endPort := 1024
		if v, err := strconv.Atoi(m.endPortInput.Value()); err == nil && v > 0 {
			endPort = v
		}

		if endPort < startPort {
			endPort = startPort
		}

		results := core.ScanPorts(host, startPort, endPort, 200)
		return portScanRunMsg{results: results, host: host}
	}
}

// Init implements tea.Model.
func (m PortScanModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m PortScanModel) Update(msg tea.Msg) (PortScanModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case portScanRunMsg:
		m.isLoading = false
		m.results = msg.results
		m.scannedHost = msg.host
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
			m.hostInput.Blur()
			m.startPortInput.Blur()
			m.endPortInput.Blur()
			switch m.activeField {
			case 0:
				m.hostInput.Focus()
			case 1:
				m.startPortInput.Focus()
			case 2:
				m.endPortInput.Focus()
			}
		case "shift+tab":
			m.activeField = (m.activeField + 2) % 3
			m.hostInput.Blur()
			m.startPortInput.Blur()
			m.endPortInput.Blur()
			switch m.activeField {
			case 0:
				m.hostInput.Focus()
			case 1:
				m.startPortInput.Focus()
			case 2:
				m.endPortInput.Focus()
			}
		case "ctrl+r", "enter":
			if !m.isLoading {
				if m.hostInput.Value() != "" {
					m.isLoading = true
					m.results = nil
					m.err = nil
					cmds = append(cmds, m.runScan(), m.spinner.Tick)
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
				m.hostInput, cmd = m.hostInput.Update(msg)
			case 1:
				m.startPortInput, cmd = m.startPortInput.Update(msg)
			case 2:
				m.endPortInput, cmd = m.endPortInput.Update(msg)
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m PortScanModel) buildResultsContent() string {
	if len(m.results) == 0 {
		return fmt.Sprintf("No open ports found on %s", m.scannedHost)
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Open ports on %s: %d found", m.scannedHost, len(m.results)))
	lines = append(lines, strings.Repeat("─", 60))
	lines = append(lines, fmt.Sprintf("%-6s  %-20s  %-10s  Banner", "Port", "Service", "Duration"))
	for _, r := range m.results {
		banner := truncate(r.Banner, 40)
		lines = append(lines, fmt.Sprintf("%-6d  %-20s  %-10s  %s",
			r.Port,
			r.Service,
			r.Duration.Round(1000000).String(),
			banner,
		))
	}
	return strings.Join(lines, "\n")
}

// View renders the port scan view.
func (m PortScanModel) View(t theme.Theme) string {
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

	// Host input
	hostBox := inactiveStyle
	if m.activeField == 0 {
		hostBox = activeStyle
	}
	sections = append(sections, labelStyle.Render("Host / IP"))
	sections = append(sections, hostBox.Render(m.hostInput.View()))
	sections = append(sections, "")

	// Port range inputs - side by side
	startBox := inactiveStyle.Width(20)
	endBox := inactiveStyle.Width(20)
	if m.activeField == 1 {
		startBox = activeStyle.Width(20)
	}
	if m.activeField == 2 {
		endBox = activeStyle.Width(20)
	}

	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Start Port\n")+startBox.Render(m.startPortInput.View()),
		"  ",
		labelStyle.Render("End Port\n")+endBox.Render(m.endPortInput.View()),
	))
	sections = append(sections, "")

	if m.isLoading {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Scanning ports..."))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	sections = append(sections, mutedStyle.Render("Press r or Enter to scan"))
	sections = append(sections, "")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
		sections = append(sections, errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
		return outerStyle.Render(strings.Join(sections, "\n"))
	}

	if m.scannedHost != "" {
		if len(m.results) == 0 {
			safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
			sections = append(sections, safeStyle.Render(fmt.Sprintf("No open ports found on %s", m.scannedHost)))
		} else {
			titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
			sections = append(sections, titleStyle.Render(fmt.Sprintf("Open Ports on %s: %d", m.scannedHost, len(m.results))))
			sections = append(sections, renderDivider(contentWidth-4, t))

			headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted)).Bold(true)
			sections = append(sections, headerStyle.Render(fmt.Sprintf("  %-6s  %-20s  %-10s  Banner", "Port", "Service", "Duration")))

			for _, r := range m.results {
				portStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info)).Bold(true)
				svcStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent))
				banner := truncate(r.Banner, 40)
				line := fmt.Sprintf("  %-20s  %-10s  %s",
					r.Service,
					r.Duration.Round(1000000).String(),
					banner,
				)
				sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
					portStyle.Render(fmt.Sprintf("  %-6d", r.Port)),
					svcStyle.Render(line),
				))
			}
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
