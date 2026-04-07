package core

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// builtinWordlist contains common paths to probe during fuzzing.
var builtinWordlist = []string{
	"admin", "api", "api/v1", "api/v2", "login", "logout", "register",
	"dashboard", "config", ".env", ".git/config", "wp-admin", "wp-config.php",
	"backup", "backup.zip", "robots.txt", "sitemap.xml", "phpinfo.php",
	"info.php", "test", "dev", "staging", "uploads", "files", "static",
	"assets", "images", "js", "css", "vendor", "node_modules", ".htaccess",
	"web.config", "server-status", "swagger", "swagger-ui", "openapi.json",
	"api/docs", "actuator", "health", "metrics", "debug", "console",
	"adminer.php", "phpmyadmin", "db", "database", "sql", "dump.sql",
	"users", "auth", "token", "keys", "private", "secret", "internal",
	"hidden", "old", "bak", "tmp", "temp", "cache", "log", "logs",
	"error_log", "access_log", ".DS_Store", "Thumbs.db", "index.php~",
	"index.bak", "config.php", "settings.php", "wp-login.php",
	"administrator", "manage", "management", "portal", "panel",
	"control", "cpanel", "plesk", "webmail", "mail", "email",
	"api/admin", "api/users", "api/config", "v1", "v2", "v3",
	"graphql", "gql", "rest", "soap", "wsdl", "xmlrpc.php",
	"install.php", "setup.php", "update.php", "upgrade.php",
	"cgi-bin", "scripts", "bin", "includes", "lib", "library",
	"download", "downloads", "export", "import", "report", "reports",
	"data", "docs", "documentation", "readme", "README.md", "CHANGELOG",
	"server-info", "nginx_status", "status", "ping", "heartbeat",
}

// RunFuzz probes paths on baseURL for accessible resources.
// customPaths can provide additional paths to test; concurrency controls parallelism.
func RunFuzz(baseURL string, customPaths []string, concurrency int) []FuzzResult {
	// Normalize base URL
	baseURL = strings.TrimRight(baseURL, "/")

	// Merge wordlists
	paths := make([]string, 0, len(builtinWordlist)+len(customPaths))
	paths = append(paths, builtinWordlist...)
	for _, p := range customPaths {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}

	if concurrency <= 0 {
		concurrency = 10
	}

	type job struct {
		path string
	}

	jobs := make(chan job, len(paths))
	for _, p := range paths {
		jobs <- job{path: p}
	}
	close(jobs)

	resultsCh := make(chan FuzzResult, len(paths))
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewHTTPClient()
			for j := range jobs {
				r := fuzzPath(client, baseURL, j.path)
				resultsCh <- r
			}
		}()
	}

	wg.Wait()
	close(resultsCh)

	var results []FuzzResult
	for r := range resultsCh {
		results = append(results, r)
	}

	// Sort by status code, then path
	sort.Slice(results, func(i, j int) bool {
		if results[i].StatusCode != results[j].StatusCode {
			return results[i].StatusCode < results[j].StatusCode
		}
		return results[i].Path < results[j].Path
	})

	return results
}

func fuzzPath(client *HTTPClient, baseURL, path string) FuzzResult {
	targetURL := baseURL + "/" + path

	req := Request{
		Method: "GET",
		URL:    targetURL,
	}

	start := time.Now()
	resp := client.Execute(req)
	elapsed := time.Since(start)

	result := FuzzResult{
		Path:     path,
		Duration: elapsed,
	}

	if resp.Error != nil {
		result.StatusCode = 0
		result.Found = false
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Size = len(resp.Body)
	result.Found = resp.StatusCode != 404 && resp.StatusCode != 0

	return result
}
