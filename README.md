<!-- prettier-ignore -->
<div align="center">

# tgdev

**Telegram MTProto debug and invoke CLI**

Call any TL method, trace API calls with correlation IDs, and expose Telegram as an MCP server for LLM-driven interaction.

[Install](#install) • [Quick start](#quick-start) • [Commands](#commands) • [MCP Setup](#mcp-setup) • [Session Management](#session-management) • [Configuration](#configuration)

[![Go Reference](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-1.5-orange?style=flat-square)](https://modelcontextprotocol.io)
[![gotd/td](https://img.shields.io/badge/gotd-td-00ADD8?style=flat-square&logo=go)](https://github.com/gotd/td)

</div>

---

## Features

- **Invoke any TL method** — call the full Telegram API from the terminal with JSON parameters
- **Persistent listener** — long-running client with IPC server for fast repeated invocations
- **Lifecycle tracing** — correlation IDs link updates to API calls to responses
- **MCP server** — expose Telegram as tools for Claude, GPT, or any MCP-compatible LLM
- **Multiple auth methods** — bot token, phone login, session string, or SQLite database
- **Auto-completion** — shell completions for bash, zsh, and fish with full method discovery

## Install

```bash
go install github.com/pageton/gotg-cli/cmd/tgdev@latest
```

Or build from source:

```bash
go build -o tgdev ./cmd/tgdev/
```

With version info:

```bash
go build -ldflags "-X main.version=$(git describe --tags --always) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o tgdev ./cmd/tgdev/
```

Requires Go 1.25+.

## Quick start

```bash
# Terminal 1 — start a persistent listener
tgdev listen --db ~/tgdev.db --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN

# Terminal 2 — invoke through the listener (fast, reuses connection)
tgdev invoke messages.sendMessage '{"Message":"hello","Peer":{"_":"inputPeerUser","UserID":123,"AccessHash":"..."}}'
```

> [!TIP]
> When `tgdev listen` is running, all `invoke` calls go through a Unix domain socket — no reconnection overhead. Without a listener, `invoke` creates a standalone connection (slower, requires auth flags).

### Get your API credentials

1. Log in at [my.telegram.org](https://my.telegram.org)
2. Go to **API Development Tools**
3. Create an application to get your `api_id` and `api_hash`

### Get current account info

```bash
tgdev getme --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

Or through a running listener:

```bash
tgdev getme
```

## Commands

| Command | Description |
| ----------------------------- | --------------------------------------------------------- |
| `tgdev invoke <method> [json]` | Invoke a TL method via listener or standalone connection |
| `tgdev listen` | Start persistent client with update stream and IPC server |
| `tgdev trace` | Full lifecycle tracing with correlation IDs |
| `tgdev methods [prefix]` | List available TL methods |
| `tgdev getme` | Get current user/bot info (`users.getFullUser`) |
| `tgdev export-session` | Export session string from SQLite database |
| `tgdev mcp` | Start MCP server (stdio or HTTP) |
| `tgdev completion <shell>` | Generate shell completions (bash, zsh, fish) |
| `tgdev version` | Print version |
| `tgdev help` | Show usage |

### Invoke

```bash
# Through a running listener
tgdev invoke users.getFullUser '{"ID":{"_":"inputUserSelf"}}'

# Standalone (no listener needed)
tgdev invoke users.getFullUser '{"ID":{"_":"inputUserSelf"}}' --api-id 12345 --api-hash HASH --bot-token TOKEN

# JSON output
tgdev invoke --format json users.getFullUser '{"ID":{"_":"inputUserSelf"}}'
```

### Listen

Starts a persistent Telegram client that logs incoming updates and accepts invoke commands over IPC.

```bash
tgdev listen --db ~/tgdev.db --api-id 12345 --api-hash HASH --bot-token TOKEN
```

### Trace

Like `listen`, but adds correlation IDs that link updates, handler invocations, API calls, and responses together.

```bash
tgdev trace --db ~/tgdev.db --api-id 12345 --api-hash HASH --bot-token TOKEN
```

Output example:

```
[1] >> messages.sendMessage
[1]    {Message:"hello", Peer:{...}}
[1] << messages.sendMessage [12ms]
[1]    {ID:42, Date:1709123456}
[2] UPDATE updateNewMessage
```

### Methods

```bash
# List all methods
tgdev methods

# Filter by prefix
tgdev methods messages.

# JSON output
tgdev methods --format json messages.send
```

## MCP Setup

tgdev exposes Telegram as an MCP server, letting LLMs call any TL method as a tool.

### Quick Setup

**stdio mode** (Claude Code, Cursor, local MCP clients):

```bash
tgdev mcp --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

**HTTP mode** (remote/multi-client):

```bash
tgdev mcp --http :8080 --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

**With persistent listener** (fastest, reuses connection):

```bash
# Terminal 1: start listener
tgdev listen --db ~/tgdev.db --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN

# Terminal 2: start MCP server
tgdev mcp
```

> [!TIP]
> When `tgdev listen` or `tgdev trace` is running, MCP tools route through the listener's IPC socket.
> Otherwise, the MCP server creates its own standalone connection.

### Full Setup Guide

For detailed setup instructions, configuration examples, and troubleshooting:

- **[Complete MCP Setup Guide](docs/setup-mcp.md)** — stdio/HTTP modes, auth methods, client configs, security

### MCP Client Configuration

**Claude Code (`.claude/mcp.json`):**

```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp"],
      "env": {
        "TGDEV_API_ID": "12345",
        "TGDEV_API_HASH": "YOUR_HASH",
        "TGDEV_BOT_TOKEN": "YOUR_TOKEN"
      }
    }
  }
}
```

**Cursor (`~/.cursor/mcp.json`):**

```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp"]
    }
  }
}
```

> [!IMPORTANT]
> Prefer environment variables (`TGDEV_*`) over CLI flags for secrets.
> CLI args are visible in `ps aux` output.

### Available MCP Tools

| Tool | Description |
| ----------------------- | ----------------------------------------------------------- |
| `tgdev_list_methods` | List TL methods with prefix filtering and cursor pagination |
| `tgdev_describe_method` | Show fields, types, and constructor hints for a method |
| `tgdev_invoke` | Invoke any TL method with JSON params |
| `tgdev_get_me` | Get current account info |
| `tgdev_listener_status` | Check if a listener is reachable on the IPC socket |
| `tgdev_config_info` | Show non-secret MCP configuration |

### JSON Parameter Format

Interface fields (like `InputPeer`, `InputUser`) use a `"_"` constructor key:

```json
{
  "Peer": {"_": "inputPeerUser", "UserID": 123, "AccessHash": "..."},
  "Message": "Hello!"
}
```

To discover constructor names: use `tgdev_describe_method` tool.


## Session Management

tgdev supports multiple session formats. Use [telegram-tools](https://github.com/pageton/telegram-tools) (client-side web app) to:

- **Generate** session strings via Telegram auth (phone or bot token) in gotg, GramJS, Kurigram, mtcute, or Telethon formats
- **Convert** between formats — paste a session string, auto-detect its format, convert to any other
- **Analyze** session strings — decode and inspect DC, auth key, user ID, isBot flag
- **Test DC connectivity** — ping all Telegram production and test data centers

> [!NOTE]
> Session strings grant full account access — treat them like passwords.
> Prefer environment variables (`TGDEV_SESSION`) or config file over CLI flags to avoid exposing in `ps aux`.

### Quick session setup

1. Visit [telegram-tools](https://pageton.github.io/telegram-tools/) (client-side, no backend)
2. Authenticate with phone number or bot token
3. Export session in **gotg** format
4. Use with tgdev:

   ```bash
   # Via CLI flag (visible in ps aux)
   tgdev listen --session "YOUR_SESSION_STRING" --api-id 12345 --api-hash HASH

   # Via environment variable (recommended)
   export TGDEV_SESSION="YOUR_SESSION_STRING"
   tgdev listen --api-id 12345 --api-hash HASH

   # Via config file (safest)
   echo '{"session": "YOUR_SESSION_STRING", "api_id": 12345, "api_hash": "HASH"}' > ~/.tgdev.json
   chmod 0600 ~/.tgdev.json
   tgdev listen
   ```

### Session formats supported by tgdev

| Format | Source | Detection Method |
|--------|--------|-------------------|
| gotg native | `session.StringSession()` | JSON, standard base64 |
| Pyrogram | `session.PyrogramSession()` | URL-safe base64, 271 bytes decoded |
| Telethon | `session.TelethonSession()` | Prefix `1` + URL-safe base64, 263/275 bytes |
| GramJS | `session.GramjsSession()` | Prefix `1` + standard base64 |
| mtcute | `session.MtcuteSession()` | URL-safe base64, prefix byte 0x03 |

`tgdev` auto-detects session format via `detectSessionFormat()` in `cmd/tgdev/main.go`.

### Export session from SQLite

If you already have a SQLite database from `tgdev listen --db ~/tgdev.db`:

```bash
tgdev export-session --db ~/tgdev.db
```

The output is a gotg-compatible session string. Use `--session-type` to specify format if auto-detection fails.

### Session string safety

- Never commit session strings to version control
- Use `chmod 0600 ~/.tgdev.json` for config file
- IPC socket is mode 0600 (owner-only access)
- `--debug` logs full API payloads to stderr — contains session tokens


Config file: `~/.tgdev.json` (auto-restricted to `0600` permissions).

```json
{
  "api_id": 12345,
  "api_hash": "your_api_hash",
  "bot_token": "123:ABC",
  "database_path": "~/tgdev.db"
}
```

### Auth methods

| Method | Flag | Notes |
| --------------- | ------------- | --------------------------------- |
| Bot token | `--bot-token` | Fastest for bots |
| Phone number | `--phone` | Interactive user login |
| Session string | `--session` | Telethon/Pyrogram/gotg format |
| SQLite database | `--db` | Persistent sessions (recommended) |

### Credential priority

CLI flags > environment variables > config file.

### Environment variables

| Variable | Maps to |
| ----------------- | ------------- |
| `TGDEV_API_ID` | `--api-id` |
| `TGDEV_API_HASH` | `--api-hash` |
| `TGDEV_BOT_TOKEN` | `--bot-token` |
| `TGDEV_SESSION` | `--session` |
| `TGDEV_PHONE` | `--phone` |

### Global flags

```
--api-id INT        Telegram API ID
--api-hash STRING   Telegram API Hash
--session STRING    Session string
--bot-token STRING  Bot token
--phone STRING      Phone number
--db PATH           SQLite database path for persistent sessions
--db-name STRING    Session name within the database (default: "default")
--socket PATH       Unix socket path for IPC (default: $XDG_RUNTIME_DIR/tgdev.sock)
--config PATH       Config file path (default: ~/.tgdev.json)
--no-color          Disable colored output
--debug             Enable verbose debug output (logs full request/response payloads)
--format FORMAT     Output format: text (default), json
```

> [!WARNING]
> `--debug` logs full API request and response payloads to stderr. Do not use in shared terminals
> or redirect to persistent logs — the output may contain session tokens and other sensitive data.

## Architecture

```
.
├── cmd/tgdev/              CLI entrypoint and command implementations
│   ├── main.go            Command dispatch, flag parsing, client creation
│   └── version.go         Build version variables (injected via ldflags)
├── invoke/                 TL method registry, JSON unmarshal, invocation, response formatting
│   ├── registry.go        Bidirectional lookup: TL schema names ↔ Go types
│   ├── invoke.go          Generic TL method invocation via raw RPC
│   ├── invoker.go         DebugInvoker middleware: logging, timing, correlation
│   ├── unmarshal.go       JSON → gotd struct with interface constructor resolution
│   ├── format.go          InvokeJSON: JSON-serialized response path
│   └── pretty.go         Color definitions and duration formatting
├── internal/
│   ├── config/            Config loading, validation, credential layering
│   │   └── config.go     CLI flags > env vars > config file (~/.tgdev.json)
│   ├── ipc/              Unix domain socket IPC server and client
│   │   └── ipc.go        InvokeRequest/Response JSON protocol
│   └── mcpserver/        MCP server with tool definitions
│       └── server.go      tgdev_list_methods, tgdev_invoke, etc.
├── trace/                 Correlation ID tracing, update listener, RPC middleware
│   ├── tracer.go         Tracer: combines invoke logging + update tracking
│   └── listener.go       UpdateListener for gotg dispatcher
├── tg-dev-agent/          Skill: Telegram bot testing via tgdev + MCP
├── go.mod                 Module: github.com/pageton/gotg-cli (Go 1.25.0)
└── README.md              User-facing documentation
```

The `invoke` package provides bidirectional lookup between TL schema names and gotd Go types using reflection over the full `tg.TypesMap()` and `tg.TypesConstructorMap()` registries. The `trace` package adds correlation IDs that link updates to handler invocations to API calls to responses.

## Dependencies

- [gotd/td](https://github.com/gotd/td) — MTProto client library
- [pageton/gotg](https://github.com/pageton/gotg) — Telegram framework (local replace directive)
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) — MCP server SDK
- [fatih/color](https://github.com/fatih/color) — Colored terminal output
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure-Go SQLite driver
