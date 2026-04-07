package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agromai/qaitor/internal/core"
	"github.com/agromai/qaitor/internal/theme"
)

type tasksMode int

const (
	tasksModeList tasksMode = iota
	tasksModeAdd
	tasksModeRunning
)

// tasksRunMsg is sent when all tasks have completed.
type tasksRunMsg struct {
	results []core.TaskResult
}

// TasksModel is the Task Runner view.
type TasksModel struct {
	tasks        []core.Task
	results      []core.TaskResult
	mode         tasksMode
	selectedTask int
	progress     progress.Model
	spinner      spinner.Model
	running      bool
	width        int
	height       int
	t            theme.Theme

	// Add task form inputs
	nameInput     textinput.Model
	urlInput      textinput.Model
	methodInput   textinput.Model
	statusInput   textinput.Model
	addFieldFocus int
}

// NewTasks creates a new TasksModel with some example tasks.
func NewTasks() TasksModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	prog := progress.New(progress.WithScaledGradient("#7aa2f7", "#9ece6a"))

	nameInput := textinput.New()
	nameInput.Placeholder = "Task name (e.g. Health Check)"
	nameInput.CharLimit = 64
	nameInput.Focus()

	urlInput := textinput.New()
	urlInput.Placeholder = "https://api.example.com/health"
	urlInput.CharLimit = 2048

	methodInput := textinput.New()
	methodInput.Placeholder = "GET"
	methodInput.SetValue("GET")
	methodInput.CharLimit = 10

	statusInput := textinput.New()
	statusInput.Placeholder = "200"
	statusInput.SetValue("200")
	statusInput.CharLimit = 5

	// Default sample tasks
	sampleTasks := []core.Task{
		{Name: "Health Check", URL: "https://httpbin.org/get", Method: "GET", ExpectedStatus: 200},
		{Name: "POST Echo", URL: "https://httpbin.org/post", Method: "POST", ExpectedStatus: 200},
		{Name: "Status 404", URL: "https://httpbin.org/status/404", Method: "GET", ExpectedStatus: 404},
	}

	return TasksModel{
		tasks:       sampleTasks,
		progress:    prog,
		spinner:     sp,
		nameInput:   nameInput,
		urlInput:    urlInput,
		methodInput: methodInput,
		statusInput: statusInput,
	}
}

// SetTheme updates the theme.
func (m *TasksModel) SetTheme(t theme.Theme) {
	m.t = t
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
	m.progress = progress.New(progress.WithScaledGradient(t.Primary, t.Success))
}

// SetSize updates dimensions.
func (m *TasksModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentWidth := w - sidebarWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.nameInput.Width = contentWidth / 2
	m.urlInput.Width = contentWidth - 4
	m.methodInput.Width = 10
	m.statusInput.Width = 8
}

// runAllTasks executes all tasks sequentially.
func (m TasksModel) runAllTasks() tea.Cmd {
	return func() tea.Msg {
		client := core.NewHTTPClient()
		results := make([]core.TaskResult, 0, len(m.tasks))

		for _, task := range m.tasks {
			req := core.Request{
				Method:  task.Method,
				URL:     task.URL,
				Headers: task.Headers,
				Body:    task.Body,
			}
			resp := client.Execute(req)

			passed := false
			if resp.Error == nil {
				passed = resp.StatusCode == task.ExpectedStatus
			}

			results = append(results, core.TaskResult{
				Task:     task,
				Response: resp,
				Passed:   passed,
				Error:    resp.Error,
			})
		}

		return tasksRunMsg{results: results}
	}
}

// Init implements tea.Model.
func (m TasksModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tasksRunMsg:
		m.running = false
		m.results = msg.results
		m.mode = tasksModeList

	case spinner.TickMsg:
		if m.running {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch m.mode {
		case tasksModeList:
			switch msg.String() {
			case "ctrl+r", "enter":
				if len(m.tasks) > 0 && !m.running {
					m.running = true
					m.results = nil
					m.mode = tasksModeRunning
					cmds = append(cmds, m.runAllTasks(), m.spinner.Tick)
				}
			case "a":
				m.mode = tasksModeAdd
				m.addFieldFocus = 0
				m.nameInput.SetValue("")
				m.urlInput.SetValue("")
				m.methodInput.SetValue("GET")
				m.statusInput.SetValue("200")
				m.nameInput.Focus()
				m.urlInput.Blur()
				m.methodInput.Blur()
				m.statusInput.Blur()
			case "d":
				if len(m.tasks) > 0 {
					idx := m.selectedTask
					m.tasks = append(m.tasks[:idx], m.tasks[idx+1:]...)
					if m.selectedTask >= len(m.tasks) && m.selectedTask > 0 {
						m.selectedTask--
					}
				}
			case "j", "down":
				if m.selectedTask < len(m.tasks)-1 {
					m.selectedTask++
				}
			case "k", "up":
				if m.selectedTask > 0 {
					m.selectedTask--
				}
			case "g":
				m.selectedTask = 0
			case "G":
				m.selectedTask = max(0, len(m.tasks)-1)
			}

		case tasksModeAdd:
			switch msg.String() {
			case "esc":
				m.mode = tasksModeList
			case "tab":
				m.addFieldFocus = (m.addFieldFocus + 1) % 4
				m.focusAddField()
			case "shift+tab":
				m.addFieldFocus = (m.addFieldFocus + 3) % 4
				m.focusAddField()
			case "enter":
				if m.addFieldFocus == 3 || msg.String() == "ctrl+enter" {
					// Save task
					expectedStatus := 200
					fmt.Sscanf(m.statusInput.Value(), "%d", &expectedStatus)
					m.tasks = append(m.tasks, core.Task{
						Name:           m.nameInput.Value(),
						URL:            m.urlInput.Value(),
						Method:         m.methodInput.Value(),
						ExpectedStatus: expectedStatus,
					})
					m.mode = tasksModeList
					m.selectedTask = len(m.tasks) - 1
				} else {
					m.addFieldFocus = (m.addFieldFocus + 1) % 4
					m.focusAddField()
				}
			}

			// Update focused add-field input
			switch m.addFieldFocus {
			case 0:
				var cmd tea.Cmd
				m.nameInput, cmd = m.nameInput.Update(msg)
				cmds = append(cmds, cmd)
			case 1:
				var cmd tea.Cmd
				m.urlInput, cmd = m.urlInput.Update(msg)
				cmds = append(cmds, cmd)
			case 2:
				var cmd tea.Cmd
				m.methodInput, cmd = m.methodInput.Update(msg)
				cmds = append(cmds, cmd)
			case 3:
				var cmd tea.Cmd
				m.statusInput, cmd = m.statusInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *TasksModel) focusAddField() {
	m.nameInput.Blur()
	m.urlInput.Blur()
	m.methodInput.Blur()
	m.statusInput.Blur()
	switch m.addFieldFocus {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.urlInput.Focus()
	case 2:
		m.methodInput.Focus()
	case 3:
		m.statusInput.Focus()
	}
}

// View renders the tasks view.
func (m TasksModel) View(t theme.Theme) string {
	contentWidth := m.width - sidebarWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	var sections []string
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))

	switch m.mode {
	case tasksModeAdd:
		sections = append(sections, renderSectionTitle("Add New Task", t))
		sections = append(sections, "")

		activeStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Primary)).
			Padding(0, 1).Width(contentWidth - 4)
		inactiveStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Border)).
			Padding(0, 1).Width(contentWidth - 4)

		fields := []struct {
			label string
			view  string
			idx   int
		}{
			{"Task Name", m.nameInput.View(), 0},
			{"URL", m.urlInput.View(), 1},
			{"Method", m.methodInput.View(), 2},
			{"Expected Status", m.statusInput.View(), 3},
		}
		for _, f := range fields {
			sections = append(sections, labelStyle.Render(f.label))
			style := inactiveStyle
			if m.addFieldFocus == f.idx {
				style = activeStyle
			}
			sections = append(sections, style.Render(f.view))
			sections = append(sections, "")
		}

		sections = append(sections, mutedStyle.Render("Tab: next field  |  Enter on last field: save  |  Esc: cancel"))

	case tasksModeRunning:
		sections = append(sections, renderSectionTitle("Running Tasks...", t))
		sections = append(sections, "")
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))
		sections = append(sections, runStyle.Render(m.spinner.View()+" Executing tasks, please wait..."))

	case tasksModeList:
		// Header
		titleRow := lipgloss.JoinHorizontal(lipgloss.Top,
			renderSectionTitle(fmt.Sprintf("Tasks (%d)", len(m.tasks)), t),
			"  ",
			mutedStyle.Render("a:add  d:delete  r:run all  j/k:navigate"),
		)
		sections = append(sections, titleRow)
		sections = append(sections, renderDivider(contentWidth-2, t))

		if len(m.tasks) == 0 {
			sections = append(sections, mutedStyle.Render("No tasks. Press 'a' to add a task."))
		} else {
			// Table header
			colWidths := taskColWidths(contentWidth)
			header := tableRow([]string{"#", "Name", "Method", "URL", "Expect"}, colWidths, t.Muted, t)
			sections = append(sections, header)
			sections = append(sections, renderDivider(contentWidth-2, t))

			for i, task := range m.tasks {
				selector := " "
				if i == m.selectedTask {
					selector = ">"
				}
				name := truncate(task.Name, colWidths[1]-2)
				url := truncate(task.URL, colWidths[3]-2)
				cols := []string{
					fmt.Sprintf("%s%d", selector, i+1),
					name,
					task.Method,
					url,
					fmt.Sprintf("%d", task.ExpectedStatus),
				}
				fg := t.Foreground
				if i == m.selectedTask {
					fg = t.Primary
				}
				sections = append(sections, tableRow(cols, colWidths, fg, t))
			}
		}

		// Results
		if len(m.results) > 0 {
			sections = append(sections, "")
			sections = append(sections, renderSectionTitle("Results", t))
			sections = append(sections, renderDivider(contentWidth-2, t))

			passed := 0
			for _, r := range m.results {
				if r.Passed {
					passed++
				}
			}
			summaryColor := t.Success
			if passed < len(m.results) {
				summaryColor = t.Warning
			}
			if passed == 0 {
				summaryColor = t.Error
			}
			summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(summaryColor)).Bold(true)
			sections = append(sections, summaryStyle.Render(
				fmt.Sprintf("  %d/%d passed", passed, len(m.results))))

			sections = append(sections, "")
			colWidths := taskColWidths(contentWidth)
			header := tableRow([]string{"#", "Name", "Status", "Duration", "Result"}, colWidths, t.Muted, t)
			sections = append(sections, header)
			sections = append(sections, renderDivider(contentWidth-2, t))

			for i, r := range m.results {
				statusStr := "ERR"
				if r.Response != nil && r.Response.Error == nil {
					statusStr = fmt.Sprintf("%d", r.Response.StatusCode)
				}
				durationStr := "-"
				if r.Response != nil {
					durationStr = r.Response.Duration.Round(1e6).String()
				}
				result := "FAIL"
				resultColor := t.Error
				if r.Passed {
					result = "PASS"
					resultColor = t.Success
				}

				cols := []string{
					fmt.Sprintf(" %d", i+1),
					truncate(r.Task.Name, colWidths[1]-2),
					statusStr,
					durationStr,
					result,
				}
				resultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(resultColor)).Bold(true)
				row := tableRow(cols[:4], colWidths[:4], t.Foreground, t)
				sections = append(sections, row+"  "+resultStyle.Render(result))
			}
		}
	}

	outerStyle := lipgloss.NewStyle().Width(contentWidth).Padding(1, 2)
	return outerStyle.Render(strings.Join(sections, "\n"))
}

func taskColWidths(total int) []int {
	return []int{4, total/4 + 4, 8, total / 2, 8}
}

func tableRow(cols []string, widths []int, color string, t theme.Theme) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	parts := make([]string, len(cols))
	for i, col := range cols {
		w := 10
		if i < len(widths) {
			w = widths[i]
		}
		cell := truncate(col, w)
		cell = cell + strings.Repeat(" ", max(0, w-len(cell)))
		parts[i] = style.Render(cell)
	}
	return strings.Join(parts, " ")
}

// GetResults returns task results for stats.
func (m TasksModel) GetResults() []core.TaskResult {
	return m.results
}
