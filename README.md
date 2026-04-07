# 𓂀 Horus

> A terminal-based QA & security auditing toolkit — HTTP analysis, data leak detection, injection testing, fuzzing, port scanning, JWT analysis, CORS testing and more, all in one fast TUI.

<img width="1914" height="989" alt="image" src="https://github.com/user-attachments/assets/8f97a7e8-7fd3-45f1-975c-f97322aa42c9" />

---

## Features

### QA / Testing

| View | Key | Description |
|------|-----|-------------|
| **Dashboard** | `1` | Real-time stats, global activity log, quick overview |
| **HTTP Analyzer** | `2` | Send requests, measure response time, inspect headers and body |
| **Task Runner** | `3` | Define and run HTTP test suites with pass/fail results |
| **Leak Scanner** | `4` | Detect PII, credentials, tokens and secrets in HTTP responses |
| **Throttle Detector** | `5` | Identify rate limiting patterns, parse Retry-After headers |
| **Security Scanner** | `6` | Audit 12+ security headers, CORS, and TLS configuration |

### Cyber / Pentest

| View | Key | Description |
|------|-----|-------------|
| **Injection Tester** | `7` | SQLi, XSS, SSTI, Path Traversal, Command Injection — automated payloads with response analysis |
| **Fuzzer** | `8` | 80+ built-in paths (`.env`, `.git/config`, `swagger`, `wp-admin`...) + custom wordlist, concurrent |
| **Port Scanner** | `9` | TCP scan with banner grabbing, 50+ services mapped (MySQL, Redis, MongoDB, SSH...) |
| **JWT Analyzer** | `0` | Decode without external libs, detect `alg:none`, brute force weak secrets, check expiry |
| **CORS Tester** | `-` | 5 origin scenarios (arbitrary, null, subdomain wildcard, prefix match), misconfiguration detection |
| **Auth / IDOR** | `=` | Sequential ID probing, rate limit bypass with 10 header spoofing techniques |

---

## Leak Detection Patterns

JWT tokens · AWS keys · GitHub tokens · Slack tokens · Google API keys · Credit cards · SSNs · Private keys (PEM) · Passwords in JSON · Bearer tokens · DB connection strings · Emails · Internal IPs · ENV variables (`DATABASE_URL`, `SECRET_KEY`...) · Stack traces (Java/Python/PHP) · Debug mode active

## Security Header Checks

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

Cycle themes with `Ctrl+T` or pick one visually in the **Theme Picker** (`T`).

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
| `1` – `6` | QA / Testing views |
| `7` – `9`, `0`, `-`, `=` | Cyber / Pentest views |
| `T` | Theme Picker |
| `j` / `k` | Navigate lists |
| `h` / `l` | Previous / next (tutorial, sidebar) |
| `Tab` / `Shift+Tab` | Focus next / previous input |
| `[` / `]` | Cycle injection type or mode |
| `Ctrl+R` / `Enter` | Run action in current view |
| `Ctrl+T` | Cycle theme |
| `?` | Toggle help overlay |
| `g` / `G` | Jump to top / bottom |
| `Esc` | Cancel / back |
| `q` | Quit |

---

## No configuration needed

Horus is a black-box tool — just point it at any URL or IP and run. No `.env`, no database, no external services required. Works completely standalone against local or remote targets.

```
Horus TUI  ──HTTP──▶  localhost:3000  /  staging.app.com  /  prod
```

If the target requires authentication, add the token directly in the **Headers** field.

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
    ├── theme/            # 7 themes with 17 color slots each
    ├── core/
    │   ├── types.go      # Shared types and page constants
    │   ├── http.go       # HTTP client with timing
    │   ├── leak.go       # Regex-based leak detection (25+ patterns)
    │   ├── injection.go  # Injection payloads and response analysis
    │   ├── fuzzer.go     # Path fuzzer with built-in wordlist
    │   ├── portscan.go   # TCP port scanner with banner grabbing
    │   ├── jwt.go        # JWT decoder and vulnerability checks
    │   ├── cors.go       # CORS misconfiguration testing
    │   ├── auth.go       # IDOR probing and rate limit bypass
    │   ├── throttle.go   # Rate limit detection
    │   └── security.go   # Security header auditing
    └── ui/
        ├── app.go        # Root bubbletea model
        ├── layout.go     # Header, sidebar, footer, help overlay
        ├── dashboard.go
        ├── analyzer.go
        ├── tasks.go
        ├── leaks.go
        ├── throttle.go
        ├── security.go
        ├── injection.go
        ├── fuzzer.go
        ├── portscan.go
        ├── jwt.go
        ├── cors.go
        ├── auth.go
        ├── themes.go
        └── tutorial.go
```

---

## License

MIT — see [LICENSE](LICENSE)
