package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TestIDOR probes a range of IDs to test for Insecure Direct Object Reference.
// The URL should contain {id} as a placeholder for the ID value.
// If no placeholder is present, the ID is appended as a path segment.
func TestIDOR(baseURL string, idStart, idEnd int, method string, headers map[string]string) []IDORResult {
	if method == "" {
		method = "GET"
	}

	client := NewHTTPClient()

	// Determine baseline: request a presumably non-existent ID (very large number)
	baselineURL := buildIDURL(baseURL, "99999999")
	baselineReq := Request{
		Method:  method,
		URL:     baselineURL,
		Headers: headers,
	}
	baselineResp := client.Execute(baselineReq)
	baselineSize := 0
	if baselineResp.Error == nil {
		baselineSize = len(baselineResp.Body)
	}

	var results []IDORResult

	for id := idStart; id <= idEnd; id++ {
		idStr := strconv.Itoa(id)
		targetURL := buildIDURL(baseURL, idStr)

		req := Request{
			Method:  method,
			URL:     targetURL,
			Headers: headers,
		}

		start := time.Now()
		resp := client.Execute(req)
		elapsed := time.Since(start)

		result := IDORResult{
			ID:       idStr,
			Duration: elapsed,
			Baseline: baselineSize,
		}

		if resp.Error != nil {
			result.StatusCode = 0
			result.Accessible = false
		} else {
			result.StatusCode = resp.StatusCode
			result.Size = len(resp.Body)
			// Accessible if 200 and size differs from baseline (not a generic 404 page)
			result.Accessible = resp.StatusCode == 200 && result.Size != baselineSize
		}

		results = append(results, result)
	}

	return results
}

func buildIDURL(baseURL, id string) string {
	if strings.Contains(baseURL, "{id}") {
		return strings.ReplaceAll(baseURL, "{id}", id)
	}
	return strings.TrimRight(baseURL, "/") + "/" + id
}

// TestRateLimitBypass tries common HTTP header techniques to bypass rate limiting.
// Returns a list of bypass technique descriptions that returned HTTP 200 instead of 429.
func TestRateLimitBypass(targetURL string, headers map[string]string) []string {
	client := NewHTTPClient()

	// First, check the baseline response without bypass headers
	baseReq := Request{
		Method:  "GET",
		URL:     targetURL,
		Headers: headers,
	}
	baseResp := client.Execute(baseReq)
	baseStatus := 0
	if baseResp.Error == nil {
		baseStatus = baseResp.StatusCode
	}

	bypassTechniques := []struct {
		description string
		headerKey   string
		headerValue string
	}{
		{"X-Forwarded-For: 127.0.0.1", "X-Forwarded-For", "127.0.0.1"},
		{"X-Real-IP: 127.0.0.1", "X-Real-IP", "127.0.0.1"},
		{"X-Originating-IP: 127.0.0.1", "X-Originating-IP", "127.0.0.1"},
		{"X-Remote-IP: 127.0.0.1", "X-Remote-IP", "127.0.0.1"},
		{"X-Client-IP: 127.0.0.1", "X-Client-IP", "127.0.0.1"},
		{"X-Host: 127.0.0.1", "X-Host", "127.0.0.1"},
		{"Forwarded: for=127.0.0.1", "Forwarded", "for=127.0.0.1"},
		{"X-Forwarded-For: ::1", "X-Forwarded-For", "::1"},
		{"X-Real-IP: 10.0.0.1", "X-Real-IP", "10.0.0.1"},
		{"X-Forwarded-For: 0.0.0.0", "X-Forwarded-For", "0.0.0.0"},
	}

	var bypasses []string

	for _, technique := range bypassTechniques {
		// Build headers with bypass header added
		reqHeaders := make(map[string]string)
		for k, v := range headers {
			reqHeaders[k] = v
		}
		reqHeaders[technique.headerKey] = technique.headerValue

		req := Request{
			Method:  "GET",
			URL:     targetURL,
			Headers: reqHeaders,
		}

		resp := client.Execute(req)
		if resp.Error != nil {
			continue
		}

		// If baseline was 429 (rate limited) and bypass got 200, it worked
		// Also report if baseline was 200 and bypass also 200 (confirms header is accepted)
		if (baseStatus == 429 && resp.StatusCode == 200) ||
			(baseStatus != 429 && resp.StatusCode == 200) {
			bypasses = append(bypasses, fmt.Sprintf("%s → HTTP %d", technique.description, resp.StatusCode))
		}
	}

	return bypasses
}
