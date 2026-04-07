package core

import (
	"fmt"
	"regexp"
	"strings"
)

// leakPattern defines a regex pattern for detecting a specific type of data leak.
type leakPattern struct {
	name     string
	pattern  *regexp.Regexp
	severity Severity
}

var leakPatterns = []leakPattern{
	{
		name:     "Email Address",
		pattern:  regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		severity: SeverityMedium,
	},
	{
		name:     "JWT Token",
		pattern:  regexp.MustCompile(`eyJ[a-zA-Z0-9_\-]+\.eyJ[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+`),
		severity: SeverityCritical,
	},
	{
		name:     "AWS Access Key",
		pattern:  regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		severity: SeverityCritical,
	},
	{
		name:     "AWS Secret Key",
		pattern:  regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret)\s*[=:]\s*['\"]?([A-Za-z0-9/+]{40})['\"]?`),
		severity: SeverityCritical,
	},
	{
		name:     "Credit Card Number",
		pattern:  regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`),
		severity: SeverityCritical,
	},
	{
		name:     "SSN",
		pattern:  regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		severity: SeverityCritical,
	},
	{
		name:     "Private Key",
		pattern:  regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		severity: SeverityCritical,
	},
	{
		name:     "Password in JSON",
		pattern:  regexp.MustCompile(`(?i)"(?:password|passwd|pwd|secret|token|api_key|apikey)"\s*:\s*"([^"]{4,})"`),
		severity: SeverityHigh,
	},
	{
		name:     "API Key Pattern",
		pattern:  regexp.MustCompile(`(?i)(?:api[_\-]?key|access[_\-]?token|auth[_\-]?token)\s*[=:]\s*['\"]?([a-zA-Z0-9\-_]{20,})['\"]?`),
		severity: SeverityHigh,
	},
	{
		name:     "Internal IP Address",
		pattern:  regexp.MustCompile(`\b(?:10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(?:1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`),
		severity: SeverityMedium,
	},
	{
		name:     "GitHub Token",
		pattern:  regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9_]{82}`),
		severity: SeverityCritical,
	},
	{
		name:     "Slack Token",
		pattern:  regexp.MustCompile(`xox[baprs]-[a-zA-Z0-9\-]{10,}`),
		severity: SeverityHigh,
	},
	{
		name:     "Google API Key",
		pattern:  regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		severity: SeverityCritical,
	},
	{
		name:     "Bearer Token",
		pattern:  regexp.MustCompile(`(?i)(?:bearer|token)\s+([a-zA-Z0-9\-_.~+/]{20,})`),
		severity: SeverityHigh,
	},
	{
		name:     "Database Connection String",
		pattern:  regexp.MustCompile(`(?i)(?:mongodb|mysql|postgres|postgresql|redis|mssql):\/\/[^\s"']+`),
		severity: SeverityCritical,
	},
}

// ScanForLeaks scans the given text for data leaks and returns all findings.
func ScanForLeaks(text string) []DataLeak {
	var leaks []DataLeak
	lines := strings.Split(text, "\n")

	for _, pattern := range leakPatterns {
		matches := pattern.pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			matchStr := text[match[0]:match[1]]

			// Find the line number
			lineNum := 1
			pos := 0
			for i, line := range lines {
				if pos+len(line) >= match[0] {
					lineNum = i + 1
					break
				}
				pos += len(line) + 1
			}

			// Get context (surrounding text)
			contextStart := match[0] - 30
			if contextStart < 0 {
				contextStart = 0
			}
			contextEnd := match[1] + 30
			if contextEnd > len(text) {
				contextEnd = len(text)
			}
			context := text[contextStart:contextEnd]
			context = strings.ReplaceAll(context, "\n", " ")

			// Truncate match for display
			displayMatch := matchStr
			if len(displayMatch) > 60 {
				displayMatch = displayMatch[:57] + "..."
			}

			leaks = append(leaks, DataLeak{
				Type:     pattern.name,
				Severity: pattern.severity,
				Match:    displayMatch,
				Context:  fmt.Sprintf("...%s...", context),
				Line:     lineNum,
			})
		}
	}

	return leaks
}

// GroupLeaksBySeverity groups data leaks by their severity level.
func GroupLeaksBySeverity(leaks []DataLeak) map[Severity][]DataLeak {
	groups := make(map[Severity][]DataLeak)
	for _, leak := range leaks {
		groups[leak.Severity] = append(groups[leak.Severity], leak)
	}
	return groups
}

// LeakSummary returns a human-readable summary of the leaks found.
func LeakSummary(leaks []DataLeak) string {
	if len(leaks) == 0 {
		return "No leaks detected"
	}

	counts := make(map[string]int)
	for _, leak := range leaks {
		counts[leak.Severity.String()]++
	}

	parts := []string{}
	if n := counts["CRITICAL"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", n))
	}
	if n := counts["HIGH"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d high", n))
	}
	if n := counts["MEDIUM"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", n))
	}
	if n := counts["LOW"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d low", n))
	}

	return fmt.Sprintf("%d leaks found: %s", len(leaks), strings.Join(parts, ", "))
}
