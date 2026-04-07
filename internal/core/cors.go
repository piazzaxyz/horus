package core

import (
	"fmt"
	"net/url"
	"strings"
)

// TestCORS tests a target URL for CORS misconfigurations by sending
// requests with various crafted Origin headers.
func TestCORS(targetURL string) []CORSResult {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}

	host := parsed.Hostname()

	// Build test origins
	testOrigins := []struct {
		origin   string
		vulnType string
	}{
		{"https://evil.com", "Arbitrary Origin Reflection"},
		{"null", "Null Origin"},
		{"https://evil." + host, "Subdomain Wildcard"},
		{"https://" + host + ".evil.com", "Trusted with Extra"},
		{"https://not" + host, "Prefix Match Bypass"},
	}

	client := NewHTTPClient()
	var results []CORSResult

	for _, test := range testOrigins {
		result := testCORSOrigin(client, targetURL, test.origin, test.vulnType)
		results = append(results, result)
	}

	return results
}

func testCORSOrigin(client *HTTPClient, targetURL, origin, vulnType string) CORSResult {
	result := CORSResult{
		TestedOrigin: origin,
		VulnType:     vulnType,
	}

	// Send OPTIONS preflight first
	optReq := Request{
		Method: "OPTIONS",
		URL:    targetURL,
		Headers: map[string]string{
			"Origin":                        origin,
			"Access-Control-Request-Method": "GET",
		},
	}

	optResp := client.Execute(optReq)
	if optResp.Error == nil {
		extractCORSHeaders(optResp, &result)
	}

	// Also send GET request with Origin header
	getReq := Request{
		Method: "GET",
		URL:    targetURL,
		Headers: map[string]string{
			"Origin": origin,
		},
	}

	getResp := client.Execute(getReq)
	if getResp.Error == nil {
		extractCORSHeaders(getResp, &result)
	}

	// Determine vulnerability
	assessCORSVulnerability(&result, origin)

	return result
}

func extractCORSHeaders(resp *Response, result *CORSResult) {
	if acao, ok := resp.Headers["Access-Control-Allow-Origin"]; ok && len(acao) > 0 {
		result.AllowedOrigin = acao[0]
	}
	if acac, ok := resp.Headers["Access-Control-Allow-Credentials"]; ok && len(acac) > 0 {
		result.AllowCredentials = strings.EqualFold(acac[0], "true")
	}
	if acam, ok := resp.Headers["Access-Control-Allow-Methods"]; ok && len(acam) > 0 {
		result.AllowMethods = strings.Join(acam, ", ")
	}
	if acah, ok := resp.Headers["Access-Control-Allow-Headers"]; ok && len(acah) > 0 {
		result.AllowHeaders = strings.Join(acah, ", ")
	}
}

func assessCORSVulnerability(result *CORSResult, testedOrigin string) {
	if result.AllowedOrigin == "" {
		// No CORS headers - not vulnerable for this test
		result.Vulnerable = false
		return
	}

	// Wildcard + credentials
	if result.AllowedOrigin == "*" && result.AllowCredentials {
		result.Vulnerable = true
		result.VulnType = fmt.Sprintf("Wildcard + Credentials (invalid but server sends both): %s", result.VulnType)
		return
	}

	// Origin reflection
	if result.AllowedOrigin == testedOrigin && testedOrigin != "*" {
		result.Vulnerable = true
		if result.AllowCredentials {
			result.VulnType = fmt.Sprintf("CRITICAL - Origin Reflection + Credentials: %s", result.VulnType)
		} else {
			result.VulnType = fmt.Sprintf("HIGH - Origin Reflection (no credentials): %s", result.VulnType)
		}
		return
	}

	// Null origin
	if result.AllowedOrigin == "null" {
		result.Vulnerable = true
		result.VulnType = "MEDIUM - Null Origin Accepted"
		return
	}
}
