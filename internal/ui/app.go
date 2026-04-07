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

// App is the root Bubbletea model for HORUS.
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
	injection InjectionModel
	fuzzer    FuzzerModel
	portScan  PortScanModel
	jwt       JWTModel
	cors      CORSModel
	auth      AuthModel
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

	injection := NewInjection()
	injection.SetTheme(t)

	fuzzer := NewFuzzer()
	fuzzer.SetTheme(t)

	portScan := NewPortScan()
	portScan.SetTheme(t)

	jwt := NewJWT()
	jwt.SetTheme(t)

	cors := NewCORS()
	cors.SetTheme(t)

	auth := NewAuth()
	auth.SetTheme(t)

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
		injection:   injection,
		fuzzer:      fuzzer,
		portScan:    portScan,
		jwt:         jwt,
		cors:        cors,
		auth:        auth,
		themes:      themes,
		tutorial:    NewTutorial(),
	}

	app.addLog("INFO", "HORUS started. Press ? for help.")
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

		// Always-global shortcuts (safe ctrl combos + quit)
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "ctrl+t":
			a.cycleTheme()
			return a, nil
		}

		// Navigation shortcuts — only fire when no input field is focused
		if !a.isTyping() {
			switch msg.String() {
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
				a.currentPage = core.PageInjection
				return a, nil
			case "8":
				a.currentPage = core.PageFuzzer
				return a, nil
			case "9":
				a.currentPage = core.PagePortScan
				return a, nil
			case "0":
				a.currentPage = core.PageJWT
				return a, nil
			case "-":
				a.currentPage = core.PageCORS
				return a, nil
			case "=":
				a.currentPage = core.PageAuth
				return a, nil
			case "T":
				a.currentPage = core.PageThemes
				return a, nil
			}
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

	case injectionRunMsg:
		if msg.err == nil {
			vulns := 0
			for _, r := range msg.results {
				if r.Vulnerable {
					vulns++
				}
			}
			a.totalRequests += len(msg.results)
			a.totalIssues += vulns
			a.addLog("INFO", fmt.Sprintf("Injection test: %d payloads, %d vulnerable", len(msg.results), vulns))
		} else {
			a.addLog("ERROR", fmt.Sprintf("Injection test failed: %v", msg.err))
		}
		var newInjection InjectionModel
		var cmd tea.Cmd
		newInjection, cmd = a.injection.Update(msg)
		a.injection = newInjection
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case fuzzerRunMsg:
		if msg.err == nil {
			found := 0
			for _, r := range msg.results {
				if r.Found {
					found++
				}
			}
			a.totalRequests += len(msg.results)
			a.addLog("INFO", fmt.Sprintf("Fuzzer: %d paths probed, %d found", len(msg.results), found))
		} else {
			a.addLog("ERROR", fmt.Sprintf("Fuzzer failed: %v", msg.err))
		}
		var newFuzzer FuzzerModel
		var cmd tea.Cmd
		newFuzzer, cmd = a.fuzzer.Update(msg)
		a.fuzzer = newFuzzer
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case portScanRunMsg:
		if msg.err == nil {
			a.totalRequests += len(msg.results)
			a.addLog("INFO", fmt.Sprintf("Port scan %s: %d open ports", msg.host, len(msg.results)))
		} else {
			a.addLog("ERROR", fmt.Sprintf("Port scan failed: %v", msg.err))
		}
		var newPortScan PortScanModel
		var cmd tea.Cmd
		newPortScan, cmd = a.portScan.Update(msg)
		a.portScan = newPortScan
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case corsRunMsg:
		if msg.err == nil {
			vulns := 0
			for _, r := range msg.results {
				if r.Vulnerable {
					vulns++
				}
			}
			a.totalRequests += len(msg.results)
			a.totalIssues += vulns
			a.addLog("INFO", fmt.Sprintf("CORS test: %d tests, %d vulnerable", len(msg.results), vulns))
		} else {
			a.addLog("ERROR", fmt.Sprintf("CORS test failed: %v", msg.err))
		}
		var newCORS CORSModel
		var cmd tea.Cmd
		newCORS, cmd = a.cors.Update(msg)
		a.cors = newCORS
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case idorRunMsg:
		if msg.err == nil {
			accessible := 0
			for _, r := range msg.results {
				if r.Accessible {
					accessible++
				}
			}
			a.totalRequests += len(msg.results)
			a.totalIssues += accessible
			a.addLog("INFO", fmt.Sprintf("IDOR probe: %d IDs, %d accessible", len(msg.results), accessible))
		} else {
			a.addLog("ERROR", fmt.Sprintf("IDOR probe failed: %v", msg.err))
		}
		var newAuth AuthModel
		var cmd tea.Cmd
		newAuth, cmd = a.auth.Update(msg)
		a.auth = newAuth
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case bypassRunMsg:
		if msg.err == nil {
			a.totalRequests++
			if len(msg.results) > 0 {
				a.totalIssues++
			}
			a.addLog("INFO", fmt.Sprintf("Rate limit bypass: %d techniques found", len(msg.results)))
		} else {
			a.addLog("ERROR", fmt.Sprintf("Bypass test failed: %v", msg.err))
		}
		var newAuth AuthModel
		var cmd tea.Cmd
		newAuth, cmd = a.auth.Update(msg)
		a.auth = newAuth
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case themeSelectMsg:
		a.themeName = msg.name
		a.t = theme.Get(msg.name)
		a.applyTheme()
		a.addLog("INFO", fmt.Sprintf("Theme changed to: %s", msg.name))
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
	case core.PageInjection:
		newInjection, cmd := a.injection.Update(msg)
		a.injection = newInjection
		return cmd
	case core.PageFuzzer:
		newFuzzer, cmd := a.fuzzer.Update(msg)
		a.fuzzer = newFuzzer
		return cmd
	case core.PagePortScan:
		newPortScan, cmd := a.portScan.Update(msg)
		a.portScan = newPortScan
		return cmd
	case core.PageJWT:
		newJWT, cmd := a.jwt.Update(msg)
		a.jwt = newJWT
		return cmd
	case core.PageCORS:
		newCORS, cmd := a.cors.Update(msg)
		a.cors = newCORS
		return cmd
	case core.PageAuth:
		newAuth, cmd := a.auth.Update(msg)
		a.auth = newAuth
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
	case core.PageInjection:
		newInjection, cmd := a.injection.Update(msg)
		a.injection = newInjection
		return cmd
	case core.PageFuzzer:
		newFuzzer, cmd := a.fuzzer.Update(msg)
		a.fuzzer = newFuzzer
		return cmd
	case core.PagePortScan:
		newPortScan, cmd := a.portScan.Update(msg)
		a.portScan = newPortScan
		return cmd
	case core.PageJWT:
		newJWT, cmd := a.jwt.Update(msg)
		a.jwt = newJWT
		return cmd
	case core.PageCORS:
		newCORS, cmd := a.cors.Update(msg)
		a.cors = newCORS
		return cmd
	case core.PageAuth:
		newAuth, cmd := a.auth.Update(msg)
		a.auth = newAuth
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
	case core.PageInjection:
		content = a.injection.View(a.t)
	case core.PageFuzzer:
		content = a.fuzzer.View(a.t)
	case core.PagePortScan:
		content = a.portScan.View(a.t)
	case core.PageJWT:
		content = a.jwt.View(a.t)
	case core.PageCORS:
		content = a.cors.View(a.t)
	case core.PageAuth:
		content = a.auth.View(a.t)
	case core.PageThemes:
		content = a.themes.View(a.t)
	case core.PageTutorial:
		content = a.tutorial.View(a.t)
	default:
		content = "Unknown page"
	}

	// Clamp content height to prevent overflow and double-footer
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		content = strings.Join(contentLines[:contentHeight], "\n")
	}

	// Main area: sidebar + content
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	// Wrap with background
	bgStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(a.t.Background)).
		Foreground(lipgloss.Color(a.t.Foreground))

	return bgStyle.Render(strings.Join([]string{header, mainArea, footer}, "\n"))
}

// isTyping returns true when the current view has a focused text input,
// so global navigation shortcuts should not fire.
func (a *App) isTyping() bool {
	switch a.currentPage {
	case core.PageAnalyzer:
		return a.analyzer.IsTyping()
	case core.PageTasks:
		return a.tasks.IsTyping()
	case core.PageLeaks:
		return a.leaks.IsTyping()
	case core.PageThrottle:
		return a.throttle.IsTyping()
	case core.PageSecurity:
		return a.security.IsTyping()
	case core.PageInjection:
		return a.injection.IsTyping()
	case core.PageFuzzer:
		return a.fuzzer.IsTyping()
	case core.PagePortScan:
		return a.portScan.IsTyping()
	case core.PageJWT:
		return a.jwt.IsTyping()
	case core.PageCORS:
		return a.cors.IsTyping()
	case core.PageAuth:
		return a.auth.IsTyping()
	}
	return false
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
	a.injection.SetTheme(a.t)
	a.fuzzer.SetTheme(a.t)
	a.portScan.SetTheme(a.t)
	a.jwt.SetTheme(a.t)
	a.cors.SetTheme(a.t)
	a.auth.SetTheme(a.t)
	a.themes.SetTheme(a.t)
}

func (a *App) resizeAll() {
	a.analyzer.SetSize(a.width, a.height)
	a.tasks.SetSize(a.width, a.height)
	a.leaks.SetSize(a.width, a.height)
	a.throttle.SetSize(a.width, a.height)
	a.security.SetSize(a.width, a.height)
	a.injection.SetSize(a.width, a.height)
	a.fuzzer.SetSize(a.width, a.height)
	a.portScan.SetSize(a.width, a.height)
	a.jwt.SetSize(a.width, a.height)
	a.cors.SetSize(a.width, a.height)
	a.auth.SetSize(a.width, a.height)
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
