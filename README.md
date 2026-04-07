# 𓂀 Horus

> A terminal-based QA & security auditing toolkit — HTTP analysis, data leak detection, throttle testing and security scanning, all in one fast TUI.

```
┌──────────────────────────────────────────────────────────────────────┐
│  𓂀 HORUS v1.0.0  [Tokyo Night]                        ● Connected    │
├────────────────────┬─────────────────────────────────────────────────┤
│                    │                                                 │
│  NAVIGATION        │   Dashboard                                     │
│                    │                                                 │
│  ● Dashboard       │   ┌──────────────┐  ┌──────────────┐            │
│  ○ HTTP Analyzer   │   │ Tasks Run    │  │ Leaks Found  │            │
│  ○ Task Runner     │   │     42       │  │      3       │            │
│  ○ Leak Scanner    │   └──────────────┘  └──────────────┘            │
│  ○ Throttle        │                                                 │
│  ○ Security        │   ┌──────────────┐  ┌──────────────┐            │
│  ────────────────  │   │  Avg. Time   │  │   Security   │            │
│  ○ Themes          │   │   145ms      │  │  Issues: 7   │            │
│  ○ Tutorial        │   └──────────────┘  └──────────────┘            │
│                    │                                                 │
│                    │   Recent Activity                               │
│                    │   ✓ GET /api/users     200  142ms               │
│                    │   ✗ POST /api/auth     401   89ms  [LEAK]       │
│                    │   ⚠ GET /api/data      200 1204ms  [SLOW]       │
├────────────────────┴──────────────────────────────────────────────────┤
│  [1-8] Navigate   [r] Run   [t] Theme   [?] Help   [q] Quit           │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Features

| View | Description |
|------|-------------|
| **Dashboard** | Real-time stats, activity log, and quick overview |
| **HTTP Analyzer** | Send requests, measure response time, inspect headers and body |
| **Task Runner** | Define and run suites of HTTP tests with pass/fail results |
| **Leak Scanner** | Detect PII, credentials, tokens, and secrets in HTTP responses |
| **Throttle Detector** | Identify rate limiting patterns, parse Retry-After headers |
| **Security Scanner** | Audit 12+ security headers, CORS, and TLS configuration |
| **Theme Picker** | Live-preview and switch between 7 themes |
| **Tutorial** | Interactive 8-step guide to get started |

### Leak Detection Patterns
JWT tokens · AWS keys · GitHub tokens · Slack tokens · Google API keys · Credit cards · SSNs · Private keys (PEM) · Passwords in JSON · Bearer tokens · DB connection strings · Emails · Internal IPs

### Security Header Checks
`Content-Security-Policy` · `Strict-Transport-Security` · `X-Frame-Options` · `X-Content-Type-Options` · `Referrer-Policy` · `Permissions-Policy` · `X-XSS-Protection` · CORS misconfiguration · TLS version analysis

---

## Themes

| Theme | Style |
|-------|-------|
| Tokyo Night | Dark blue & purple |
| Catppuccin Mocha | Soft pastel dark |
| Dracula | Purple & pink |
| Nord | Arctic blue |
| Gruvbox Dark | Warm earth tones |
| One Dark | Atom-inspired |
| Everforest | Green & earthy |

---

## Install

### Requirements
- [Go 1.22+](https://go.dev/dl/)

### Build from source

```bash
# Linux / Arch
git clone https://github.com/piazzaxyz/horus.git
cd horus
go mod tidy
go build -o horus .
./horus

# Quick install to ~/.local/bin
bash install.sh
```

```powershell
# Windows (PowerShell)
git clone https://github.com/piazzaxyz/horus.git
cd horus
go mod tidy
go build -o horus.exe .
.\horus.exe

# Quick install to ~/bin
.\install.ps1
```

### Cross-compile

```bash
# Linux binary from Windows
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/horus-linux .

# All platforms
make build-all
```

---

## Keybindings

| Key | Action |
|-----|--------|
| `1` – `8` | Switch views |
| `j` / `k` | Navigate lists |
| `h` / `l` | Sidebar / tutorial prev & next |
| `Tab` / `Shift+Tab` | Focus next / previous input |
| `r` / `Enter` | Run action in current view |
| `t` | Cycle theme |
| `?` | Toggle help overlay |
| `g` / `G` | Jump to top / bottom |
| `Esc` | Cancel / back |
| `q` | Quit |

---

## Project Structure

```
horus/
├── main.go
├── go.mod
├── Makefile
├── install.sh
├── install.ps1
└── internal/
    ├── theme/        # 7 themes with 17 color slots each
    ├── core/
    │   ├── types.go      # Shared types
    │   ├── http.go       # HTTP client with timing
    │   ├── leak.go       # Regex-based leak detection
    │   ├── throttle.go   # Rate limit detection
    │   └── security.go   # Security header auditing
    └── ui/
        ├── app.go        # Root bubbletea model
        ├── layout.go     # Header, sidebar, footer
        ├── dashboard.go
        ├── analyzer.go
        ├── tasks.go
        ├── leaks.go
        ├── throttle.go
        ├── security.go
        ├── themes.go
        └── tutorial.go
```

---

## License

MIT — see [LICENSE](LICENSE)
