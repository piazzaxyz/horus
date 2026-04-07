package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/theme"
)

// TutorialStep represents one step in the tutorial.
type TutorialStep struct {
	Title   string
	Content string
	Tip     string
}

// TutorialModel is the Tutorial view.
type TutorialModel struct {
	step       int
	totalSteps int
	steps      []TutorialStep
	width      int
	height     int
}

// NewTutorial creates a new TutorialModel.
func NewTutorial() TutorialModel {
	steps := []TutorialStep{
		{
			Title: "Welcome to QAITOR",
			Content: `QAITOR is a terminal-based QA and CyberSecurity testing tool built
for engineers who want fast, keyboard-driven HTTP analysis.

QAITOR stands for QA + Auditor — it combines HTTP testing,
data leak detection, throttle analysis, and security scanning
into one cohesive TUI application.

Navigate between views using number keys 1-8, or use the sidebar.
All actions use Vim-style keybindings for speed and efficiency.`,
			Tip: "Press l or → (or n) to go to the next step.",
		},
		{
			Title: "Navigation Basics",
			Content: `QAITOR uses Vim-style navigation throughout:

  1-8        Switch to a specific view
  j / ↓      Move down in lists and tables
  k / ↑      Move up in lists and tables
  h / ←      Go to previous tutorial step
  l / →      Go to next tutorial step
  g          Jump to top of a list
  G          Jump to bottom of a list
  Tab        Move focus to the next input field
  Shift+Tab  Move focus to the previous input field
  Esc        Cancel or go back
  q          Quit QAITOR
  ?          Toggle the help overlay
  t          Cycle to the next color theme`,
			Tip: "The ? key shows a full keybinding reference at any time.",
		},
		{
			Title: "HTTP Analyzer (View 2)",
			Content: `The HTTP Analyzer lets you compose and send HTTP requests
and inspect the full response.

  1. Press 2 to switch to the HTTP Analyzer view
  2. Tab to navigate between fields:
       URL → Method → Headers → Body
  3. Enter the target URL (e.g. https://httpbin.org/get)
  4. Set the HTTP method (GET, POST, PUT, DELETE, PATCH)
  5. Optionally add headers (one per line: Key: Value)
  6. Optionally add a request body (JSON, XML, etc.)
  7. Press r or Enter while URL is focused to send

The response panel shows:
  - HTTP status code and text
  - Response time in milliseconds
  - TLS version (for HTTPS)
  - Response headers
  - Full response body
  - Automatic leak detection summary`,
			Tip: "Use https://httpbin.org for safe testing — it echoes your request back.",
		},
		{
			Title: "Task Runner (View 3)",
			Content: `The Task Runner lets you define multiple HTTP tasks and
run them all in sequence, like a lightweight test suite.

  1. Press 3 to open the Task Runner
  2. Sample tasks are pre-loaded to get you started
  3. Press a to add a new task (name, URL, method, expected status)
  4. Press d to delete the selected task
  5. Press j/k to select a task
  6. Press r to run ALL tasks

Results are displayed in a table:
  PASS  — Response status matched the expected status
  FAIL  — Status mismatch or network error

Use this for smoke testing APIs, regression checks,
or validating that endpoints return expected codes.`,
			Tip: "Expected status 200 is pre-filled. Change it to 404, 401, etc. as needed.",
		},
		{
			Title: "Leak Scanner (View 4)",
			Content: `The Leak Scanner fetches an HTTP response and analyzes
the body for sensitive data that should not be exposed.

Detected leak types:
  CRITICAL  JWT tokens, AWS keys, private keys, credit cards, SSNs,
             database connection strings, GitHub tokens
  HIGH      Passwords in JSON, API keys, Bearer tokens, Slack tokens
  MEDIUM    Email addresses, internal IP addresses
  LOW       Any other potential PII patterns

How to use:
  1. Press 4 to open the Leak Scanner
  2. Enter the target URL
  3. Press r to fetch and scan
  4. Results are grouped by severity

You can also switch to Raw Text Mode (Ctrl+R) to paste
response body text directly without fetching a URL.`,
			Tip: "Always get authorization before scanning systems you don't own.",
		},
		{
			Title: "Throttle Detector (View 5)",
			Content: `The Throttle Detector sends N requests to an endpoint
and analyzes rate limiting behavior.

Configuration:
  URL           The endpoint to test
  Request Count Number of requests to send (max 200)
  Interval (ms) Milliseconds between requests

Analysis output:
  - Whether rate limiting was detected (429/503 responses)
  - Throttle rate (% of throttled requests)
  - Min/Max/Avg response times
  - Retry-After header value
  - Pattern description

The results table shows each request with:
  Status, Duration, Throttled flag, Retry-After value`,
			Tip: "Start with a small count (5-10) to test, then increase for thorough analysis.",
		},
		{
			Title: "Security Scanner (View 6)",
			Content: `The Security Scanner checks an HTTP endpoint for
common security header misconfigurations.

Checks performed:
  X-Content-Type-Options    Prevents MIME sniffing
  X-Frame-Options           Clickjacking protection
  Content-Security-Policy   Resource loading control
  Strict-Transport-Security HTTPS enforcement (HSTS)
  X-XSS-Protection          XSS filter (legacy)
  Referrer-Policy           Referrer information control
  Permissions-Policy        Browser feature permissions
  Cache-Control             Caching of sensitive data
  CORS headers              Cross-origin request policy
  TLS version               Encryption protocol version
  Server / X-Powered-By     Technology disclosure

Results are color-coded by severity:
  RED    Critical / missing high-severity headers
  ORANGE High severity warnings
  YELLOW Medium severity issues
  GREEN  Header present and correctly configured`,
			Tip: "HSTS + CSP + X-Frame-Options are the most critical headers to have.",
		},
		{
			Title: "Theme Picker (View 7)",
			Content: `QAITOR ships with 7 carefully selected color themes:

  Tokyo Night      — Cool blues and purples
  Catppuccin Mocha — Warm pastel tones
  Dracula          — Classic dark purple theme
  Nord             — Arctic, icy blue palette
  Gruvbox Dark     — Warm retro earthy tones
  One Dark         — Atom editor dark theme
  Everforest       — Natural forest greens

To switch themes:
  - Press 7 to open the Theme Picker
  - Use j/k to preview themes
  - Press Enter or r to activate the selected theme
  - Press t from any view to cycle to the next theme

Each theme preview shows the full color palette including
primary colors and severity indicators.`,
			Tip: "The 't' key cycles themes from any view without opening the picker.",
		},
	}

	return TutorialModel{
		steps:      steps,
		totalSteps: len(steps),
	}
}

// SetSize updates dimensions.
func (m *TutorialModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Init implements tea.Model.
func (m TutorialModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m TutorialModel) Update(msg tea.Msg) (TutorialModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l", "right", "n":
			if m.step < m.totalSteps-1 {
				m.step++
			}
		case "h", "left", "p":
			if m.step > 0 {
				m.step--
			}
		case "g":
			m.step = 0
		case "G":
			m.step = m.totalSteps - 1
		}
	}
	return m, nil
}

// View renders the tutorial.
func (m TutorialModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	step := m.steps[m.step]

	var sections []string

	// Progress bar
	progressFilled := m.step + 1
	progressTotal := m.totalSteps
	barWidth := contentWidth - 20
	if barWidth < 10 {
		barWidth = 10
	}

	filledWidth := barWidth * progressFilled / progressTotal
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border))
	progressBar := progressStyle.Render(strings.Repeat("█", filledWidth)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-filledWidth))

	progressLine := fmt.Sprintf("Step %d of %d  %s", m.step+1, m.totalSteps, progressBar)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
	sections = append(sections, mutedStyle.Render(progressLine))
	sections = append(sections, "")

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color(t.Primary)).
		Width(contentWidth - 6)
	sections = append(sections, titleStyle.Render(step.Title))
	sections = append(sections, "")

	// Content box
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Foreground(lipgloss.Color(t.Foreground)).
		Background(lipgloss.Color(t.Highlight)).
		Padding(1, 2).
		Width(contentWidth - 4)
	sections = append(sections, contentStyle.Render(step.Content))
	sections = append(sections, "")

	// Tip box
	if step.Tip != "" {
		tipLabelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Background)).
			Background(lipgloss.Color(t.Accent)).
			Bold(true).
			Padding(0, 1)
		tipStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Accent)).
			Foreground(lipgloss.Color(t.Accent)).
			Padding(0, 2).
			Width(contentWidth - 4)
		tipContent := tipLabelStyle.Render(" TIP ") + "  " + step.Tip
		sections = append(sections, tipStyle.Render(tipContent))
		sections = append(sections, "")
	}

	// Navigation hint
	navParts := []string{}
	if m.step > 0 {
		navParts = append(navParts, mutedStyle.Render("h/← prev"))
	}
	navParts = append(navParts, mutedStyle.Render(fmt.Sprintf("[%d/%d]", m.step+1, m.totalSteps)))
	if m.step < m.totalSteps-1 {
		navParts = append(navParts, mutedStyle.Render("l/→ next"))
	} else {
		doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
		navParts = append(navParts, doneStyle.Render("Tutorial Complete! Press 1 to go to Dashboard."))
	}

	sections = append(sections, strings.Join(navParts, "   "))

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}
