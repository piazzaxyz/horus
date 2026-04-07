package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
)

// SecurityCheckResult holds the full result of a security scan.
type SecurityCheckResult struct {
	Issues      []SecurityIssue
	TLSInfo     string
	CORSInfo    string
	HeadersRaw  map[string][]string
	StatusCode  int
	URL         string
}

type headerCheck struct {
	header      string
	category    string
	description string
	severity    Severity
	recommended string
}

var securityHeaders = []headerCheck{
	{
		header:      "X-Content-Type-Options",
		category:    "Content Type",
		description: "Prevents MIME-type sniffing attacks",
		severity:    SeverityMedium,
		recommended: "nosniff",
	},
	{
		header:      "X-Frame-Options",
		category:    "Clickjacking",
		description: "Protects against clickjacking attacks",
		severity:    SeverityHigh,
		recommended: "DENY or SAMEORIGIN",
	},
	{
		header:      "Content-Security-Policy",
		category:    "CSP",
		description: "Controls which resources the browser can load",
		severity:    SeverityHigh,
		recommended: "default-src 'self'; ...",
	},
	{
		header:      "Strict-Transport-Security",
		category:    "HSTS",
		description: "Forces browsers to use HTTPS",
		severity:    SeverityHigh,
		recommended: "max-age=31536000; includeSubDomains",
	},
	{
		header:      "X-XSS-Protection",
		category:    "XSS",
		description: "Enables XSS filtering in browsers",
		severity:    SeverityMedium,
		recommended: "1; mode=block",
	},
	{
		header:      "Referrer-Policy",
		category:    "Privacy",
		description: "Controls how much referrer info is included with requests",
		severity:    SeverityLow,
		recommended: "strict-origin-when-cross-origin",
	},
	{
		header:      "Permissions-Policy",
		category:    "Permissions",
		description: "Controls which browser features can be used",
		severity:    SeverityLow,
		recommended: "geolocation=(), microphone=(), camera=()",
	},
	{
		header:      "Cache-Control",
		category:    "Caching",
		description: "Prevents sensitive data from being cached",
		severity:    SeverityMedium,
		recommended: "no-store, no-cache",
	},
	{
		header:      "X-Permitted-Cross-Domain-Policies",
		category:    "Cross-Domain",
		description: "Controls cross-domain data loading",
		severity:    SeverityLow,
		recommended: "none",
	},
	{
		header:      "Cross-Origin-Embedder-Policy",
		category:    "COEP",
		description: "Controls cross-origin resource embedding",
		severity:    SeverityMedium,
		recommended: "require-corp",
	},
	{
		header:      "Cross-Origin-Opener-Policy",
		category:    "COOP",
		description: "Isolates the browsing context",
		severity:    SeverityMedium,
		recommended: "same-origin",
	},
	{
		header:      "Cross-Origin-Resource-Policy",
		category:    "CORP",
		description: "Controls which origins can load resources",
		severity:    SeverityMedium,
		recommended: "same-origin",
	},
}

// RunSecurityScan performs a full security scan on the given URL.
func RunSecurityScan(url string) *SecurityCheckResult {
	client := NewHTTPClient()
	resp := client.Execute(Request{
		Method: "GET",
		URL:    url,
	})

	result := &SecurityCheckResult{
		URL: url,
	}

	if resp.Error != nil {
		result.Issues = append(result.Issues, SecurityIssue{
			Category:    "Connection",
			Description: fmt.Sprintf("Failed to connect: %v", resp.Error),
			Severity:    SeverityCritical,
			Present:     false,
		})
		return result
	}

	result.StatusCode = resp.StatusCode
	result.HeadersRaw = resp.Headers

	// Check security headers
	for _, check := range securityHeaders {
		issue := SecurityIssue{
			Category:    check.category,
			Header:      check.header,
			Description: check.description,
			Severity:    check.severity,
			Recommended: check.recommended,
		}

		if vals, ok := resp.Headers[check.header]; ok && len(vals) > 0 {
			issue.Present = true
			issue.Value = strings.Join(vals, ", ")
			// Analyze the value for known misconfigurations
			issue = analyzeHeaderValue(issue)
		} else {
			issue.Present = false
			issue.Description = fmt.Sprintf("Missing: %s - %s", check.header, check.description)
		}

		result.Issues = append(result.Issues, issue)
	}

	// CORS analysis
	result.CORSInfo = analyzeCORS(resp.Headers)
	if corsIssues := checkCORSIssues(resp.Headers); len(corsIssues) > 0 {
		result.Issues = append(result.Issues, corsIssues...)
	}

	// TLS analysis
	result.TLSInfo = analyzeTLS(url, resp.TLSVersion)

	// Check for server information disclosure
	if server, ok := resp.Headers["Server"]; ok && len(server) > 0 {
		result.Issues = append(result.Issues, SecurityIssue{
			Category:    "Information Disclosure",
			Header:      "Server",
			Description: fmt.Sprintf("Server header discloses: %s", strings.Join(server, ", ")),
			Severity:    SeverityLow,
			Present:     true,
			Value:       strings.Join(server, ", "),
			Recommended: "Remove or obfuscate Server header",
		})
	}

	// X-Powered-By
	if powered, ok := resp.Headers["X-Powered-By"]; ok && len(powered) > 0 {
		result.Issues = append(result.Issues, SecurityIssue{
			Category:    "Information Disclosure",
			Header:      "X-Powered-By",
			Description: fmt.Sprintf("X-Powered-By discloses technology: %s", strings.Join(powered, ", ")),
			Severity:    SeverityLow,
			Present:     true,
			Value:       strings.Join(powered, ", "),
			Recommended: "Remove X-Powered-By header",
		})
	}

	return result
}

func analyzeHeaderValue(issue SecurityIssue) SecurityIssue {
	val := strings.ToLower(issue.Value)

	switch issue.Header {
	case "Strict-Transport-Security":
		if !strings.Contains(val, "max-age") {
			issue.Severity = SeverityHigh
			issue.Description = "HSTS missing max-age directive"
		} else if strings.Contains(val, "max-age=0") {
			issue.Severity = SeverityHigh
			issue.Description = "HSTS max-age=0 effectively disables HSTS"
		}
	case "Content-Security-Policy":
		if strings.Contains(val, "unsafe-inline") {
			issue.Severity = SeverityHigh
			issue.Description = "CSP allows 'unsafe-inline' which weakens protection"
		} else if strings.Contains(val, "unsafe-eval") {
			issue.Severity = SeverityMedium
			issue.Description = "CSP allows 'unsafe-eval' which weakens protection"
		} else if strings.Contains(val, "*") {
			issue.Severity = SeverityMedium
			issue.Description = "CSP contains wildcard (*) which weakens protection"
		}
	case "X-Frame-Options":
		if val == "allow-from" || val == "allowall" {
			issue.Severity = SeverityHigh
			issue.Description = "X-Frame-Options uses deprecated ALLOW-FROM value"
		}
	}

	return issue
}

func analyzeCORS(headers map[string][]string) string {
	var parts []string

	if vals, ok := headers["Access-Control-Allow-Origin"]; ok {
		parts = append(parts, fmt.Sprintf("Allow-Origin: %s", strings.Join(vals, ", ")))
	}
	if vals, ok := headers["Access-Control-Allow-Methods"]; ok {
		parts = append(parts, fmt.Sprintf("Allow-Methods: %s", strings.Join(vals, ", ")))
	}
	if vals, ok := headers["Access-Control-Allow-Headers"]; ok {
		parts = append(parts, fmt.Sprintf("Allow-Headers: %s", strings.Join(vals, ", ")))
	}
	if vals, ok := headers["Access-Control-Allow-Credentials"]; ok {
		parts = append(parts, fmt.Sprintf("Allow-Credentials: %s", strings.Join(vals, ", ")))
	}
	if vals, ok := headers["Access-Control-Max-Age"]; ok {
		parts = append(parts, fmt.Sprintf("Max-Age: %s", strings.Join(vals, ", ")))
	}

	if len(parts) == 0 {
		return "No CORS headers present"
	}
	return strings.Join(parts, " | ")
}

func checkCORSIssues(headers map[string][]string) []SecurityIssue {
	var issues []SecurityIssue

	if vals, ok := headers["Access-Control-Allow-Origin"]; ok {
		origin := strings.Join(vals, ", ")
		if origin == "*" {
			issues = append(issues, SecurityIssue{
				Category:    "CORS",
				Header:      "Access-Control-Allow-Origin",
				Description: "CORS wildcard (*) allows any origin to access resources",
				Severity:    SeverityHigh,
				Present:     true,
				Value:       origin,
				Recommended: "Specify explicit allowed origins",
			})
		}

		// Check for CORS + Credentials combination
		if creds, ok2 := headers["Access-Control-Allow-Credentials"]; ok2 {
			if origin == "*" && strings.ToLower(strings.Join(creds, "")) == "true" {
				issues = append(issues, SecurityIssue{
					Category:    "CORS",
					Header:      "Access-Control-Allow-Credentials",
					Description: "Wildcard CORS with credentials=true is a critical misconfiguration",
					Severity:    SeverityCritical,
					Present:     true,
					Value:       "true",
					Recommended: "Never use wildcard origin with credentials=true",
				})
			}
		}
	}

	return issues
}

func analyzeTLS(url, tlsVersion string) string {
	if !strings.HasPrefix(url, "https://") {
		return "Not using HTTPS - all data transmitted in plaintext!"
	}

	if tlsVersion == "" {
		// Try to get TLS info directly
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // nolint:gosec
			},
		}
		client := &http.Client{Transport: transport}
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.TLS != nil {
				tlsVersion = tlsVersionString(resp.TLS.Version)
			}
		}
	}

	if tlsVersion == "" {
		return "HTTPS (TLS version unknown)"
	}

	risk := ""
	switch tlsVersion {
	case "TLS 1.0", "TLS 1.1":
		risk = " - DEPRECATED, upgrade to TLS 1.2+"
	case "TLS 1.2":
		risk = " - Acceptable"
	case "TLS 1.3":
		risk = " - Excellent"
	}

	return fmt.Sprintf("%s%s", tlsVersion, risk)
}

// CountIssuesBySeverity returns counts of issues by severity.
func CountIssuesBySeverity(issues []SecurityIssue) map[Severity]int {
	counts := make(map[Severity]int)
	for _, issue := range issues {
		if !issue.Present || issue.Severity == SeverityHigh || issue.Severity == SeverityCritical {
			counts[issue.Severity]++
		}
	}
	return counts
}
