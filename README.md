# Grok Proxy Plus — Terminal Edition

<p align="center">
  <strong>One-command, terminal-only OpenAI-compatible proxy for Grok</strong><br/>
  Multi-account · streaming · thinking · local <code>/v1</code> API · no GUI required
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#openai-compatible-proxy">OpenAI proxy</a> ·
  <a href="#build-from-source">Build</a> ·
  <a href="#disclaimer">Disclaimer</a> ·
  <a href="#license">License</a>
</p>

---

## What is this?

This is a **terminal-only fork** of [Maicon501a/grok-proxy-plus](https://github.com/Maicon501a/grok-proxy-plus) that runs the entire proxy + chat from a single command, with **no Wails, no GUI, no browser window** — perfect for headless servers, SSH sessions, dev containers, and CI.

It keeps everything from the desktop version:

1. **OAuth device-code login** with xAI / Grok (no Grok CLI required)
2. **Local OpenAI-compatible API** at `http://127.0.0.1:8787/v1`
3. **Streaming + thinking** chat (terminal REPL or one-shot)
4. **Multi-account** support (login with several xAI accounts, switch on the fly)
5. **Anthropic Messages API** at `/v1/messages` (Claude Code compatible)
6. **Token & cost tracking** persisted under AppData

Use it with **Cursor, Open Code, Continue, Open WebUI, Claude Code**, or any client that speaks OpenAI Chat Completions / Responses, or Anthropic Messages.

> **Not affiliated with xAI.** Unofficial community project. Use at your own risk. See [DISCLAIMER.md](./DISCLAIMER.md) and [LICENSE](./LICENSE).

---

## Install

### One-command install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/deivid22srk/grok-proxy-cli/main/install.sh | bash
```

The installer:

1. Detects your OS/arch (linux, darwin, windows × amd64, arm64)
2. Downloads a prebuilt binary from the latest GitHub release if one exists
3. Otherwise, bootstraps Go and builds from source
4. Installs the binary as `grok-proxy-cli` to `~/.local/bin` (or `/usr/local/bin` if root)
5. Prints next-step instructions

### Manual binary

Grab a prebuilt binary from the [Releases](../../releases) page, drop it on your `PATH`, and `chmod +x` it (unix).

### From source

```bash
git clone https://github.com/deivid22srk/grok-proxy-cli.git
cd grok-proxy-cli
make cli        # outputs build/bin/grok-proxy-cli
# or:
go build -o grok-proxy-cli ./cmd/grok-proxy-cli
```

Requirements: Go 1.23+. **No Wails, no GTK, no Node, no GUI dependencies.**

---

## Quick start

```bash
# 1) install (skip if you already did)
curl -fsSL https://raw.githubusercontent.com/deivid22srk/grok-proxy-cli/main/install.sh | bash

# 2) sign in to xAI (opens a device-code URL + code in your terminal)
grok-proxy-cli login

# 3) confirm you have an account
grok-proxy-cli accounts

# 4) start the local OpenAI-compatible proxy
grok-proxy-cli serve
```

You'll see something like:

```
grok-proxy-plus listening on http://127.0.0.1:8787/v1
endpoints:
  GET  /v1/models
  POST /v1/chat/completions
  POST /v1/responses
  POST /v1/messages
press Ctrl+C to stop
active account: you@example.com
```

Point any OpenAI-compatible client at `http://127.0.0.1:8787/v1` and you're done.

---

## Commands

```
grok-proxy-cli                 start the local OpenAI proxy (default = serve)
grok-proxy-cli serve           same as above; flags: --listen, --api-key, --no-proxy
grok-proxy-cli login           sign in with xAI device-code OAuth
grok-proxy-cli accounts        list accounts
grok-proxy-cli use <id>        set active account (id prefix supported)
grok-proxy-cli logout <id>     remove an account (id prefix supported)
grok-proxy-cli models          list models available to the active account
grok-proxy-cli chat            interactive streaming chat REPL
grok-proxy-cli ask "<prompt>"  one-shot prompt; flags: --effort, --model, --no-think
```

Global flag (any command):

```
--data-dir <path>   override AppData directory
```

### Interactive REPL

```bash
grok-proxy-cli chat
```

```
grok-proxy-cli chat — type :q to quit, :clear to reset history

> what is the capital of Brazil?
Brasília…
[usage] prompt=… completion=… reasoning=… total=…

> :q
```

Reasoning (thinking) is shown dimmed; content is shown in normal colour. Usage stats are printed at the end of each turn.

### One-shot prompt

```bash
grok-proxy-cli ask "explain CAP theorem in one paragraph" --effort high
```

### Switch account

```bash
grok-proxy-cli accounts
# ID                       LABEL                            EMAIL                ACTIVE
# 01J9F8E2HK…              you@example.com                  you@example.com      *

grok-proxy-cli use 01J9F8E2
```

---

## OpenAI-compatible proxy

After `grok-proxy-cli serve` starts, a local server listens on:

```text
http://127.0.0.1:8787/v1
```

(If `8787` is busy, the app tries **`8788`**.)

| Setting | Value |
|---------|--------|
| **Base URL** | `http://127.0.0.1:8787/v1` |
| **API key** | any string (or the optional key set with `--api-key`) |
| **Model** | `grok-4.5` or `grok-4.5-responses` |

### Example — environment

```bash
export OPENAI_BASE_URL=http://127.0.0.1:8787/v1
export OPENAI_API_KEY=grok-desktop
export OPENAI_MODEL=grok-4.5
```

### Example — cURL

```bash
curl http://127.0.0.1:8787/v1/chat/completions \
  -H "Authorization: Bearer grok-desktop" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4.5",
    "stream": true,
    "reasoning_effort": "high",
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

### Example — Claude Code / Anthropic Messages API

```bash
curl http://127.0.0.1:8787/v1/messages \
  -H "x-api-key: grok-desktop" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4.5",
    "max_tokens": 1024,
    "stream": true,
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

### Example — Open Code / openai-compatible provider

```json
{
  "provider": {
    "grok-desktop": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Grok Proxy Plus",
      "options": {
        "baseURL": "http://127.0.0.1:8787/v1",
        "apiKey": "grok-desktop"
      },
      "models": {
        "grok-4.5": { "name": "Grok 4.5" },
        "grok-4.5-responses": { "name": "Grok 4.5 (Responses)" }
      }
    }
  }
}
```

### API modes

| Mode | Endpoint | Notes |
|------|----------|--------|
| **chat** | `/v1/chat/completions` | Classic OpenAI chat + `reasoning_content` stream |
| **responses** | `/v1/responses` | Multi-turn + native `web_search` / `x_search` (tools sanitized for OpenCode) |
| **messages** | `/v1/messages` | Anthropic Messages API (stream + tools) |
| ~~completions~~ | `/v1/completions` | **Not supported** (legacy) |

---

## Multi-account

- `grok-proxy-cli login` → device-code login (xAI)
- Each account is stored separately under AppData
- `grok-proxy-cli use <id-prefix>` switches the active account
- The **active** account is used both by `chat`/`ask` and by the proxy

Data directory (never committed to git):

| OS | Path |
|----|------|
| Windows | `%LOCALAPPDATA%\GrokDesktop\` |
| macOS | `~/Library/Application Support/GrokDesktop/` |
| Linux | `~/.local/share/GrokDesktop/` |

```text
GrokDesktop/
├── settings.json
├── usage.json
├── history.json
├── accounts/<id>.json
└── logs/
```

---

## Build from source

```bash
# terminal CLI (no GUI dependencies)
make cli              # outputs build/bin/grok-proxy-cli
# or directly:
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o grok-proxy-cli ./cmd/grok-proxy-cli

# install to $GOBIN or ~/.local/bin
make install
```

The terminal CLI is **pure Go** — no cgo, no GTK, no WebKit, no Node, no Wails. It cross-compiles cleanly to every supported platform:

```bash
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o grok-proxy-cli-linux-amd64   ./cmd/grok-proxy-cli
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -o grok-proxy-cli-linux-arm64   ./cmd/grok-proxy-cli
GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -o grok-proxy-cli-darwin-amd64  ./cmd/grok-proxy-cli
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -o grok-proxy-cli-darwin-arm64  ./cmd/grok-proxy-cli
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o grok-proxy-cli-windows-amd64.exe ./cmd/grok-proxy-cli
```

### Desktop (Wails) build — still supported

The original Wails desktop app is preserved. To build it:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails build
```

See [the upstream README](https://github.com/Maicon501a/grok-proxy-plus#build-from-source) for desktop build requirements (GTK, WebKit, Node 20+, …).

### Self-test (no GUI)

```bash
go run ./cmd/selftest
```

Checks AppData layout, models list, and a live chat (requires a logged-in account on that machine).

---

## Releases

GitHub Actions builds binaries for **linux / darwin / windows × amd64 / arm64** automatically on every tag push:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Assets published to the GitHub Release:

- `grok-proxy-cli-linux-amd64`
- `grok-proxy-cli-linux-arm64`
- `grok-proxy-cli-darwin-amd64`
- `grok-proxy-cli-darwin-arm64`
- `grok-proxy-cli-windows-amd64.exe`

(The original desktop Wails builds still happen via `.github/workflows/ci.yml` and `release.yml`.)

---

## Project layout

```text
.
├── cmd/
│   ├── grok-proxy-cli/         # terminal CLI (NEW — no GUI deps)
│   └── selftest/               # integration smoke test
├── internal/
│   ├── app/                    # headless core shared by CLI + desktop (NEW)
│   ├── oauth/                  # device login + refresh
│   ├── store/                  # multi-account AppData
│   ├── upstream/               # cli-chat-proxy client (stream)
│   ├── proxyhttp/              # local OpenAI/Anthropic HTTP server
│   ├── pricing/                # token cost estimates
│   ├── skills/                 # local skill catalog (markdown)
│   └── mcpconfig/              # MCP server config
├── install.sh                  # one-command installer (NEW)
├── Makefile                    # build helpers (NEW)
├── .github/workflows/
│   ├── ci.yml                  # desktop Wails CI (preserved)
│   ├── release.yml             # desktop Wails release (preserved)
│   └── cli.yml                 # terminal CLI build + release (NEW)
├── main.go / app.go            # original Wails desktop app (preserved)
├── frontend/                   # desktop UI (preserved)
├── LICENSE
├── DISCLAIMER.md
└── README.md
```

---

## Security notes

- **Tokens never go into the git repo** — only AppData on your machine
- OAuth `client_id` in source is the **public** xAI CLI client (PKCE, no client secret)
- Do not commit `accounts/`, `*.env`, or release binaries from a dirty local machine
- Treat the local proxy as **localhost-only** unless you know what you are doing

---

## Disclaimer

**Use at your own risk.** Authors are **not responsible** for bans, billing, data loss, ToS violations, or any damages.
This is **not** an official xAI product. Full text: [DISCLAIMER.md](./DISCLAIMER.md).

---

## Acknowledgements

Forked from [Maicon501a/grok-proxy-plus](https://github.com/Maicon501a/grok-proxy-plus) — thank you for the original desktop app, OAuth flow, and proxy server. This terminal edition reuses all of that infrastructure through a new `internal/app` package and adds a small `cmd/grok-proxy-cli` entry point.

---

## License

**MIT (Non-Commercial)** — free for personal / non-commercial use.
**No commercial use** without written permission.
Full terms: [LICENSE](./LICENSE).
