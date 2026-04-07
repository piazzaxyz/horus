package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

// App is the root Bubbletea model for QAITOR.
type App struct {
	currentPage     core.Page
	width           int
	height          int
	themeName       string
	t               theme.Theme
	globalLogs      []core.LogEntry
	totalRequests   int
	totalLeaks      int
	totalIssues     int
	avgResponseTime float64
	responseTimesMs []float64
	showHelp        bool

	// Sub-view models
	dashboard DashboardModel
	analyzer  AnalyzerModel
	tasks     TasksModel
	leaks     LeaksModel
	throttle  ThrottleModel
	security  SecurityModel
	themes    ThemesModel
	tutorial  TutorialModel
}

// New creates a new App with default settings.
func New() *App {
	themeName := "Tokyo Night"
	t := theme.Get(themeName)

	analyzer := NewAnalyzer()
	analyzer.SetTheme(t)

	tasks := NewTasks()
	tasks.SetTheme(t)

	leaks := NewLeaks()
	leaks.SetTheme(t)

	throttle := NewThrottle()
	throttle.SetTheme(t)

	security := NewSecurity()
	security.SetTheme(t)

	themes := NewThemes()
	themes.SetTheme(t)

	app := &App{
		currentPage: core.PageDashboard,
		themeName:   themeName,
		t:           t,
		dashboard:   NewDashboard(),
		analyzer:    analyzer,
		tasks:       tasks,
		leaks:       leaks,
		throttle:    throttle,
		security:    security,
		themes:      themes,
		tutorial:    NewTutorial(),
	}

	app.addLog("INFO", "QAITOR started. Press ? for help, 8 for tutorial.")
	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resizeAll()
		return a, nil

	case tea.KeyMsg:
		// Handle help overlay toggle first
		if msg.String() == "?" {
			a.showHelp = !a.showHelp
			return a, nil
		}

		// Close help on any key if visible
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}

		// Global navigation
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "1":
			a.currentPage = core.PageDashboard
			return a, nil
		case "2":
			a.currentPage = core.PageAnalyzer
			return a, nil
		case "3":
			a.currentPage = core.PageTasks
			return a, nil
		case "4":
			a.currentPage = core.PageLeaks
			return a, nil
		case "5":
			a.currentPage = core.PageThrottle
			return a, nil
		case "6":
			a.currentPage = core.PageSecurity
			return a, nil
		case "7":
			a.currentPage = core.PageThemes
			return a, nil
		case "8":
			a.currentPage = core.PageTutorial
			return a, nil
		case "t":
			a.cycleTheme()
			return a, nil
		}

		// Delegate to current view
		cmd := a.delegateKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case analyzerRunMsg:
		// Update stats from analyzer result
		if msg.resp != nil && msg.resp.Error == nil {
			a.totalRequests++
			a.totalLeaks += len(msg.leaks)
			a.totalIssues += len(msg.secIssues)
			ms := float64(msg.resp.Duration.Milliseconds())
			a.responseTimesMs = append(a.responseTimesMs, ms)
			a.avgResponseTime = average(a.responseTimesMs)

			a.addLog("INFO", fmt.Sprintf("HTTP %s %s → %s (%.0fms, %d leaks)",
				"GET", msg.resp.Status[:3],
				msg.resp.Status, ms, len(msg.leaks)))

			if len(msg.leaks) > 0 {
				a.addLog("WARN", fmt.Sprintf("%s", core.LeakSummary(msg.leaks)))
			}
		} else if msg.resp != nil && msg.resp.Error != nil {
			a.addLog("ERROR", fmt.Sprintf("Request failed: %v", msg.resp.Error))
		}
		// Also delegate to analyzer
		var newAnalyzer AnalyzerModel
		var cmd tea.Cmd
		newAnalyzer, cmd = a.analyzer.Update(msg)
		a.analyzer = newAnalyzer
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tasksRunMsg:
		passed := 0
		for _, r := range msg.results {
			if r.Passed {
				passed++
			}
			if r.Response != nil && r.Response.Error == nil {
				a.totalRequests++
				ms := float64(r.Response.Duration.Milliseconds())
				a.responseTimesMs = append(a.responseTimesMs, ms)
			}
		}
		a.avgResponseTime = average(a.responseTimesMs)
		a.addLog("INFO", fmt.Sprintf("Task Run: %d/%d passed", passed, len(msg.results)))

		var newTasks TasksModel
		var cmd tea.Cmd
		newTasks, cmd = a.tasks.Update(msg)
		a.tasks = newTasks
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case leaksRunMsg:
		if msg.err == nil {
			a.totalRequests++
			a.totalLeaks += len(msg.leaks)
			a.addLog("INFO", fmt.Sprintf("Leak scan: %s", core.LeakSummary(msg.leaks)))
		} else {
			a.addLog("ERROR", fmt.Sprintf("Leak scan failed: %v", msg.err))
		}
		var newLeaks LeaksModel
		var cmd tea.Cmd
		newLeaks, cmd = a.leaks.Update(msg)
		a.leaks = newLeaks
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case throttleRunMsg:
		a.totalRequests += msg.analysis.TotalRequests
		for _, r := range msg.analysis.Results {
			ms := float64(r.Duration.Milliseconds())
			a.responseTimesMs = append(a.responseTimesMs, ms)
		}
		a.avgResponseTime = average(a.responseTimesMs)
		a.addLog("INFO", fmt.Sprintf("Throttle: %s", msg.analysis.Pattern))

		var newThrottle ThrottleModel
		var cmd tea.Cmd
		newThrottle, cmd = a.throttle.Update(msg)
		a.throttle = newThrottle
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case securityRunMsg:
		if msg.result != nil {
			issues := 0
			for _, iss := range msg.result.Issues {
				if !iss.Present || iss.Severity >= core.SeverityHigh {
					issues++
				}
			}
			a.totalIssues += issues
			a.totalRequests++
			a.addLog("INFO", fmt.Sprintf("Security scan: %d issues found for %s", issues, msg.result.URL))
		}
		var newSecurity SecurityModel
		var cmd tea.Cmd
		newSecurity, cmd = a.security.Update(msg)
		a.security = newSecurity
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case themeSelectMsg:
		a.themeName = msg.name
		a.t = theme.Get(msg.name)
		a.applyTheme()
		a.addLog("INFO", fmt.Sprintf("Theme changed to: %s", msg.name))
		// delegate to themes model
		var newThemes ThemesModel
		var cmd tea.Cmd
		newThemes, cmd = a.themes.Update(msg)
		a.themes = newThemes
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	default:
		// Delegate to all views that might have ongoing operations
		cmd := a.delegateToCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) delegateKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch a.currentPage {
	case core.PageAnalyzer:
		newAnalyzer, cmd := a.analyzer.Update(msg)
		a.analyzer = newAnalyzer
		return cmd
	case core.PageTasks:
		newTasks, cmd := a.tasks.Update(msg)
		a.tasks = newTasks
		return cmd
	case core.PageLeaks:
		newLeaks, cmd := a.leaks.Update(msg)
		a.leaks = newLeaks
		return cmd
	case core.PageThrottle:
		newThrottle, cmd := a.throttle.Update(msg)
		a.throttle = newThrottle
		return cmd
	case core.PageSecurity:
		newSecurity, cmd := a.security.Update(msg)
		a.security = newSecurity
		return cmd
	case core.PageThemes:
		newThemes, cmd := a.themes.Update(msg)
		a.themes = newThemes
		return cmd
	case core.PageTutorial:
		newTutorial, cmd := a.tutorial.Update(msg)
		a.tutorial = newTutorial
		return cmd
	}
	return nil
}

func (a *App) delegateToCurrentView(msg tea.Msg) tea.Cmd {
	switch a.currentPage {
	case core.PageAnalyzer:
		newAnalyzer, cmd := a.analyzer.Update(msg)
		a.analyzer = newAnalyzer
		return cmd
	case core.PageTasks:
		newTasks, cmd := a.tasks.Update(msg)
		a.tasks = newTasks
		return cmd
	case core.PageLeaks:
		newLeaks, cmd := a.leaks.Update(msg)
		a.leaks = newLeaks
		return cmd
	case core.PageThrottle:
		newThrottle, cmd := a.throttle.Update(msg)
		a.throttle = newThrottle
		return cmd
	case core.PageSecurity:
		newSecurity, cmd := a.security.Update(msg)
		a.security = newSecurity
		return cmd
	case core.PageThemes:
		newThemes, cmd := a.themes.Update(msg)
		a.themes = newThemes
		return cmd
	}
	return nil
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Help overlay
	if a.showHelp {
		return renderHelp(a.width, a.height, a.t)
	}

	header := renderHeader(a.width, a.themeName, a.t)
	footer := renderFooter(a.currentPage, a.width, a.t)

	contentHeight := a.height - 3 // header + footer + some padding

	sidebar := renderSidebar(a.currentPage, contentHeight, a.t)

	// Render current page content
	var content string
	switch a.currentPage {
	case core.PageDashboard:
		content = a.dashboard.View(a.width, contentHeight, a.t, a.totalRequests, a.totalLeaks, a.totalIssues, a.avgResponseTime, a.globalLogs)
	case core.PageAnalyzer:
		content = a.analyzer.View(a.t)
	case core.PageTasks:
		content = a.tasks.View(a.t)
	case core.PageLeaks:
		content = a.leaks.View(a.t)
	case core.PageThrottle:
		content = a.throttle.View(a.t)
	case core.PageSecurity:
		content = a.security.View(a.t)
	case core.PageThemes:
		content = a.themes.View(a.t)
	case core.PageTutorial:
		content = a.tutorial.View(a.t)
	default:
		content = "Unknown page"
	}

	// Main area: sidebar + content
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	// Wrap with background
	bgStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(a.t.Background)).
		Foreground(lipgloss.Color(a.t.Foreground))

	return bgStyle.Render(strings.Join([]string{header, mainArea, footer}, "\n"))
}

func (a *App) cycleTheme() {
	a.themeName = theme.Next(a.themeName)
	a.t = theme.Get(a.themeName)
	a.applyTheme()
	a.addLog("INFO", fmt.Sprintf("Theme: %s", a.themeName))
}

func (a *App) applyTheme() {
	a.analyzer.SetTheme(a.t)
	a.tasks.SetTheme(a.t)
	a.leaks.SetTheme(a.t)
	a.throttle.SetTheme(a.t)
	a.security.SetTheme(a.t)
	a.themes.SetTheme(a.t)
}

func (a *App) resizeAll() {
	a.analyzer.SetSize(a.width, a.height)
	a.tasks.SetSize(a.width, a.height)
	a.leaks.SetSize(a.width, a.height)
	a.throttle.SetSize(a.width, a.height)
	a.security.SetSize(a.width, a.height)
	a.themes.SetSize(a.width, a.height)
	a.tutorial.SetSize(a.width, a.height)
}

func (a *App) addLog(level, message string) {
	entry := core.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}
	a.globalLogs = append(a.globalLogs, entry)
	// Keep only last 100 logs
	if len(a.globalLogs) > 100 {
		a.globalLogs = a.globalLogs[len(a.globalLogs)-100:]
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
