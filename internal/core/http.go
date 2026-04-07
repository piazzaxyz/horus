package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultUserAgent = "QAITOR/1.0 QA-Security-Tool"
)

// HTTPClient wraps http.Client with additional configuration.
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTPClient with TLS configured to skip verification
// (for testing purposes) and a default timeout.
func NewHTTPClient() *HTTPClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // nolint:gosec - intentional for security testing tool
		},
	}
	return &HTTPClient{
		client: &http.Client{
			Timeout:   defaultTimeout,
			Transport: transport,
		},
	}
}

// Execute sends an HTTP request and returns a Response with timing information.
func (c *HTTPClient) Execute(req Request) *Response {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewBufferString(req.Body)
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequest(method, req.URL, bodyReader)
	if err != nil {
		return &Response{Error: fmt.Errorf("failed to create request: %w", err)}
	}

	// Set default headers
	httpReq.Header.Set("User-Agent", defaultUserAgent)

	// Set custom headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return &Response{
			Duration: duration,
			Error:    fmt.Errorf("request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Response{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Duration:   duration,
			Error:      fmt.Errorf("failed to read body: %w", err),
		}
	}

	// Get TLS version info
	tlsVersion := ""
	if resp.TLS != nil {
		tlsVersion = tlsVersionString(resp.TLS.Version)
	}

	// Copy headers
	headers := make(map[string][]string)
	for k, v := range resp.Header {
		headers[k] = v
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(bodyBytes),
		Duration:   duration,
		TLSVersion: tlsVersion,
	}
}

// ParseHeaders parses a raw headers string (one per line, "Key: Value") into a map.
func ParseHeaders(raw string) map[string]string {
	headers := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}

// FormatResponse formats an HTTP response for display.
func FormatResponse(resp *Response) string {
	if resp == nil {
		return "No response"
	}
	if resp.Error != nil {
		return fmt.Sprintf("Error: %v", resp.Error)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTP %s\n", resp.Status))
	sb.WriteString(fmt.Sprintf("Duration: %v\n", resp.Duration.Round(time.Millisecond)))
	if resp.TLSVersion != "" {
		sb.WriteString(fmt.Sprintf("TLS: %s\n", resp.TLSVersion))
	}
	sb.WriteString("\n--- Headers ---\n")
	for k, v := range resp.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(v, ", ")))
	}
	sb.WriteString("\n--- Body ---\n")
	body := resp.Body
	if len(body) > 8192 {
		body = body[:8192] + "\n... (truncated)"
	}
	sb.WriteString(body)
	return sb.String()
}

func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", v)
	}
}
