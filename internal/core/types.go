package core

import "time"

// Page represents the current view/page in the TUI.
type Page int

const (
	PageDashboard Page = iota
	PageAnalyzer
	PageTasks
	PageLeaks
	PageThrottle
	PageSecurity
	PageInjection // NEW
	PageFuzzer    // NEW
	PagePortScan  // NEW
	PageJWT       // NEW
	PageCORS      // NEW
	PageAuth      // NEW
	PageThemes
	PageTutorial
)

// PageNames maps page constants to display names.
var PageNames = map[Page]string{
	PageDashboard: "Dashboard",
	PageAnalyzer:  "HTTP Analyzer",
	PageTasks:     "Task Runner",
	PageLeaks:     "Leak Scanner",
	PageThrottle:  "Throttle Detector",
	PageSecurity:  "Security Scanner",
	PageInjection: "Injection Tester",
	PageFuzzer:    "Fuzzer",
	PagePortScan:  "Port Scanner",
	PageJWT:       "JWT Analyzer",
	PageCORS:      "CORS Tester",
	PageAuth:      "Auth / IDOR",
	PageThemes:    "Theme Picker",
	PageTutorial:  "Tutorial",
}

// PageIcons maps page constants to sidebar icons.
var PageIcons = map[Page]string{
	PageDashboard: "⬛",
	PageAnalyzer:  "⬛",
	PageTasks:     "⬛",
	PageLeaks:     "⬛",
	PageThrottle:  "⬛",
	PageSecurity:  "⬛",
	PageInjection: "⬛",
	PageFuzzer:    "⬛",
	PagePortScan:  "⬛",
	PageJWT:       "⬛",
	PageCORS:      "⬛",
	PageAuth:      "⬛",
	PageThemes:    "⬛",
	PageTutorial:  "⬛",
}

// Severity levels for security/leak findings.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Request represents an HTTP request to be executed.
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// Response represents an HTTP response with timing info.
type Response struct {
	StatusCode int
	Status     string
	Headers    map[string][]string
	Body       string
	Duration   time.Duration
	TLSVersion string
	Error      error
}

// Task represents a named HTTP task for the Task Runner.
type Task struct {
	Name           string
	URL            string
	Method         string
	Headers        map[string]string
	Body           string
	ExpectedStatus int
}

// TaskResult holds the result of executing a Task.
type TaskResult struct {
	Task     Task
	Response *Response
	Passed   bool
	Error    error
}

// DataLeak represents a detected piece of sensitive data in a response.
type DataLeak struct {
	Type     string
	Severity Severity
	Match    string
	Context  string
	Line     int
}

// ThrottleResult holds the result of a single request in throttle detection.
type ThrottleResult struct {
	RequestNum int
	StatusCode int
	Duration   time.Duration
	Throttled  bool
	RetryAfter string
	Error      error
}

// SecurityIssue represents a security finding from header/TLS analysis.
type SecurityIssue struct {
	Category    string
	Header      string
	Description string
	Severity    Severity
	Present     bool
	Value       string
	Recommended string
}

// LogEntry represents an entry in the global activity log.
type LogEntry struct {
	Timestamp time.Time
	Level     string // INFO, WARN, ERROR, SUCCESS
	Message   string
}

// NewLogEntry creates a new log entry with current timestamp.
func NewLogEntry(level, message string) LogEntry {
	return LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}
}

// InjectionType categorizes injection attack types.
type InjectionType int

const (
	InjectionSQLi InjectionType = iota
	InjectionXSS
	InjectionSSTI
	InjectionPathTraversal
	InjectionCmdInjection
)

func (t InjectionType) String() string {
	switch t {
	case InjectionSQLi:
		return "SQLi"
	case InjectionXSS:
		return "XSS"
	case InjectionSSTI:
		return "SSTI"
	case InjectionPathTraversal:
		return "PathTraversal"
	case InjectionCmdInjection:
		return "CmdInjection"
	default:
		return "Unknown"
	}
}

// InjectionResult holds the result of a single injection test.
type InjectionResult struct {
	Payload    string
	Type       InjectionType
	Parameter  string
	StatusCode int
	Duration   time.Duration
	Vulnerable bool
	Evidence   string
	Confidence string // "LOW", "MEDIUM", "HIGH"
}

// FuzzResult holds the result of a single fuzzing attempt.
type FuzzResult struct {
	Path       string
	StatusCode int
	Size       int
	Duration   time.Duration
	Found      bool
}

// PortResult holds the result of scanning a single port.
type PortResult struct {
	Port     int
	Open     bool
	Service  string
	Banner   string
	Duration time.Duration
}

// JWTAnalysis holds the decoded JWT and vulnerability findings.
type JWTAnalysis struct {
	Raw             string
	Header          map[string]interface{}
	Payload         map[string]interface{}
	Algorithm       string
	IsExpired       bool
	ExpiresAt       *time.Time
	IssuedAt        *time.Time
	Vulnerabilities []string
	Valid            bool
}

// CORSResult holds the result of a CORS misconfiguration test.
type CORSResult struct {
	TestedOrigin     string
	AllowedOrigin    string
	AllowCredentials bool
	AllowMethods     string
	AllowHeaders     string
	Vulnerable       bool
	VulnType         string
}

// IDORResult holds the result of a single IDOR probe.
type IDORResult struct {
	ID         string
	StatusCode int
	Size       int
	Duration   time.Duration
	Accessible bool
	Baseline   int // baseline response size
}
