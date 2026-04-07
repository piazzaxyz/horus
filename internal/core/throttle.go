package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ThrottleConfig holds configuration for throttle detection.
type ThrottleConfig struct {
	URL         string
	Count       int
	IntervalMs  int
	Method      string
	Headers     map[string]string
}

// ThrottleAnalysis holds the analysis of a throttle test run.
type ThrottleAnalysis struct {
	Results       []ThrottleResult
	TotalRequests int
	Throttled     int
	ThrottleRate  float64
	AvgDuration   time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	RetryAfter    string
	IsThrottled   bool
	Pattern       string
}

// RunThrottleTest sends N requests to the URL and analyzes throttling behavior.
func RunThrottleTest(cfg ThrottleConfig) ThrottleAnalysis {
	client := NewHTTPClient()
	req := Request{
		Method:  cfg.Method,
		URL:     cfg.URL,
		Headers: cfg.Headers,
	}
	if req.Method == "" {
		req.Method = "GET"
	}

	results := make([]ThrottleResult, 0, cfg.Count)
	var totalDuration time.Duration
	minDuration := time.Duration(1<<63 - 1)
	maxDuration := time.Duration(0)
	throttledCount := 0
	retryAfter := ""

	for i := 1; i <= cfg.Count; i++ {
		resp := client.Execute(req)

		throttled := false
		ra := ""

		if resp.Error == nil {
			throttled = resp.StatusCode == 429 || resp.StatusCode == 503
			if throttled {
				throttledCount++
				// Parse Retry-After header
				if vals, ok := resp.Headers["Retry-After"]; ok && len(vals) > 0 {
					ra = vals[0]
					if retryAfter == "" {
						retryAfter = ra
					}
				}
				if vals, ok := resp.Headers["X-RateLimit-Reset"]; ok && len(vals) > 0 && ra == "" {
					ra = "Reset at: " + vals[0]
				}
			}
		}

		results = append(results, ThrottleResult{
			RequestNum: i,
			StatusCode: func() int {
				if resp.Error != nil {
					return 0
				}
				return resp.StatusCode
			}(),
			Duration:   resp.Duration,
			Throttled:  throttled,
			RetryAfter: ra,
			Error:      resp.Error,
		})

		if resp.Error == nil {
			totalDuration += resp.Duration
			if resp.Duration < minDuration {
				minDuration = resp.Duration
			}
			if resp.Duration > maxDuration {
				maxDuration = resp.Duration
			}
		}

		if cfg.IntervalMs > 0 && i < cfg.Count {
			time.Sleep(time.Duration(cfg.IntervalMs) * time.Millisecond)
		}
	}

	avgDuration := time.Duration(0)
	if len(results) > 0 {
		avgDuration = totalDuration / time.Duration(len(results))
	}
	if minDuration == time.Duration(1<<63-1) {
		minDuration = 0
	}

	throttleRate := 0.0
	if len(results) > 0 {
		throttleRate = float64(throttledCount) / float64(len(results)) * 100
	}

	pattern := detectThrottlePattern(results)

	return ThrottleAnalysis{
		Results:       results,
		TotalRequests: len(results),
		Throttled:     throttledCount,
		ThrottleRate:  throttleRate,
		AvgDuration:   avgDuration,
		MinDuration:   minDuration,
		MaxDuration:   maxDuration,
		RetryAfter:    retryAfter,
		IsThrottled:   throttledCount > 0,
		Pattern:       pattern,
	}
}

// detectThrottlePattern analyzes results to identify the throttle pattern.
func detectThrottlePattern(results []ThrottleResult) string {
	if len(results) == 0 {
		return "No data"
	}

	throttledCount := 0
	firstThrottle := -1
	for i, r := range results {
		if r.Throttled {
			throttledCount++
			if firstThrottle == -1 {
				firstThrottle = i + 1
			}
		}
	}

	if throttledCount == 0 {
		return "No rate limiting detected"
	}

	if firstThrottle > 0 {
		patterns := []string{
			fmt.Sprintf("Rate limited after %d requests", firstThrottle-1),
		}

		// Check if throttling is consistent
		consecutiveThrottled := 0
		maxConsecutive := 0
		for _, r := range results {
			if r.Throttled {
				consecutiveThrottled++
				if consecutiveThrottled > maxConsecutive {
					maxConsecutive = consecutiveThrottled
				}
			} else {
				consecutiveThrottled = 0
			}
		}

		if maxConsecutive > 1 {
			patterns = append(patterns, fmt.Sprintf("%d consecutive 429s", maxConsecutive))
		}

		// Parse retry-after values
		retryValues := []string{}
		for _, r := range results {
			if r.RetryAfter != "" && !contains(retryValues, r.RetryAfter) {
				retryValues = append(retryValues, r.RetryAfter)
			}
		}
		if len(retryValues) > 0 {
			patterns = append(patterns, fmt.Sprintf("Retry-After: %s", strings.Join(retryValues, ", ")))
		}

		return strings.Join(patterns, " | ")
	}

	return "Intermittent throttling"
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ParseRetryAfter parses the Retry-After header value (seconds or HTTP date).
func ParseRetryAfter(value string) string {
	if secs, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return fmt.Sprintf("%d seconds", secs)
	}
	return value
}
