package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/theme"
)

// themeSelectMsg is sent when the user selects a theme.
type themeSelectMsg struct {
	name string
}

// ThemesModel is the Theme Picker view.
type ThemesModel struct {
	themes      []theme.Theme
	selectedIdx int
	width       int
	height      int
	t           theme.Theme
}

// NewThemes creates a new ThemesModel.
func NewThemes() ThemesModel {
	return ThemesModel{
		themes: theme.All(),
	}
}

// SetTheme updates the active theme display.
func (m *ThemesModel) SetTheme(t theme.Theme) {
	m.t = t
	// Keep selectedIdx in sync
	for i, th := range m.themes {
		if th.Name == t.Name {
			m.selectedIdx = i
			break
		}
	}
}

// SetSize updates dimensions.
func (m *ThemesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Init implements tea.Model.
func (m ThemesModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m ThemesModel) Update(msg tea.Msg) (ThemesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.themes)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "g":
			m.selectedIdx = 0
		case "G":
			m.selectedIdx = len(m.themes) - 1
		case "enter", "r", " ":
			return m, func() tea.Msg {
				return themeSelectMsg{name: m.themes[m.selectedIdx].Name}
			}
		}
	}
	return m, nil
}

// View renders the theme picker.
func (m ThemesModel) View(currentTheme theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	t := currentTheme
	var sections []string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
	sections = append(sections, titleStyle.Render("Theme Picker"))
	sections = append(sections, mutedStyle.Render("j/k: navigate  Enter/r: apply theme"))
	sections = append(sections, "")

	for i, th := range m.themes {
		isSelected := i == m.selectedIdx
		isActive := th.Name == currentTheme.Name

		indicator := " "
		if isActive {
			indicator = "●"
		} else if isSelected {
			indicator = ">"
		}

		// Border style for the card
		borderColor := th.Border
		if isSelected {
			borderColor = t.Primary
		}

		cardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Background(lipgloss.Color(th.Background)).
			Padding(0, 2).
			Width(contentWidth - 4)

		// Theme name
		nameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Primary)).
			Background(lipgloss.Color(th.Background)).
			Bold(true)

		// Active badge
		activeBadge := ""
		if isActive {
			activeBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Background)).
				Background(lipgloss.Color(th.Success)).
				Bold(true).
				Padding(0, 1).
				Render(" ACTIVE ")
		}

		// Color swatches
		swatchColors := []struct {
			color string
			label string
		}{
			{th.Primary, "PRI"},
			{th.Secondary, "SEC"},
			{th.Accent, "ACC"},
			{th.Success, "SUC"},
			{th.Warning, "WRN"},
			{th.Error, "ERR"},
			{th.Info, "INF"},
			{th.Muted, "MUT"},
		}

		swatches := ""
		for _, s := range swatchColors {
			swatch := lipgloss.NewStyle().
				Background(lipgloss.Color(s.color)).
				Foreground(lipgloss.Color(th.Background)).
				Padding(0, 1).
				Render(s.label)
			swatches += swatch + " "
		}

		// Severity preview
		sevColors := []struct {
			color string
			label string
		}{
			{th.Critical, "CRIT"},
			{th.High, "HIGH"},
			{th.Medium, "MED"},
			{th.Low, "LOW"},
		}
		sevSwatches := ""
		for _, s := range sevColors {
			sev := lipgloss.NewStyle().
				Foreground(lipgloss.Color(s.color)).
				Background(lipgloss.Color(th.Background)).
				Bold(true).
				Render(s.label)
			sevSwatches += sev + " "
		}

		// Sample text
		sampleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Foreground)).
			Background(lipgloss.Color(th.Background))
		commentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Comment)).
			Background(lipgloss.Color(th.Background))

		nameLine := fmt.Sprintf("%s %s  %s", indicator, nameStyle.Render(th.Name), activeBadge)
		content := strings.Join([]string{
			nameLine,
			sampleStyle.Render("  Normal text  ") + commentStyle.Render("# comment style"),
			"  " + swatches,
			"  " + sevSwatches,
		}, "\n")

		sections = append(sections, cardStyle.Render(content))
		sections = append(sections, "")
	}

	sections = append(sections, mutedStyle.Render("Tip: Press 't' from any view to cycle through themes"))

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
