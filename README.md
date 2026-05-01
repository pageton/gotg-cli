<!-- prettier-ignore -->
<div align="center">

# tgdev

**Telegram MTProto debug and invoke CLI**

Call any TL method, trace API calls with correlation IDs, and expose Telegram as an MCP server for LLM-driven interaction.

[![Go Reference](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-1.5-orange?style=flat-square)](https://modelcontextprotocol.io)
[![gotd/td](https://img.shields.io/badge/gotd-td-00ADD8?style=flat-square&logo=go)](https://github.com/gotd/td)

[Install](#install) • [Quick start](#quick-start) • [Commands](#commands) • [MCP integration](#mcp-integration) • [Configuration](#configuration)

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

<details>
<summary>Get your API credentials</summary>

1. Log in at [my.telegram.org](https://my.telegram.org)
2. Go to **API Development Tools**
3. Create an application to get your `api_id` and `api_hash`

</details>

<details>
<summary>Get current account info</summary>

```bash
tgdev getme --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

Or through a running listener:

```bash
tgdev getme
```

</details>

## Commands

| Command | Description |
|---------|-------------|
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

## MCP integration

tgdev exposes Telegram as an MCP server, letting LLMs call any TL method as a tool.

### stdio mode

For Claude Code, Cursor, and other MCP clients that communicate over stdio:

```bash
tgdev mcp --api-id 12345 --api-hash HASH --bot-token TOKEN
```

### HTTP mode

Stateless Streamable HTTP for remote or multi-client setups:

```bash
tgdev mcp --http :8080 --api-id 12345 --api-hash HASH --bot-token TOKEN
```

### Available MCP tools

| Tool | Description |
|------|-------------|
| `tgdev_list_methods` | List TL methods with prefix filtering and cursor pagination |
| `tgdev_describe_method` | Show fields, types, and constructor hints for a method |
| `tgdev_invoke` | Invoke any TL method with JSON params |
| `tgdev_get_me` | Get current account info |
| `tgdev_listener_status` | Check if a listener is reachable on the IPC socket |
| `tgdev_config_info` | Show non-secret MCP configuration |

> [!NOTE]
> When a `tgdev listen` or `tgdev trace` process is running, MCP tools route invocations through the listener's IPC socket. Otherwise, the MCP server uses its own standalone connection.

<details>
<summary>Example MCP client configuration</summary>

```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp", "--api-id", "12345", "--api-hash", "HASH", "--bot-token", "TOKEN"]
    }
  }
}
```

Or with environment variables to avoid exposing secrets in process args:

```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp"],
      "env": {
        "TGDEV_API_ID": "12345",
        "TGDEV_API_HASH": "HASH",
        "TGDEV_BOT_TOKEN": "TOKEN"
      }
    }
  }
}
```

</details>

## Configuration

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
|--------|------|-------|
| Bot token | `--bot-token` | Fastest for bots |
| Phone number | `--phone` | Interactive user login |
| Session string | `--session` | Telethon/Pyrogram/gotg format |
| SQLite database | `--db` | Persistent sessions (recommended) |

### Credential priority

CLI flags > environment variables > config file.

### Environment variables

| Variable | Maps to |
|----------|---------|
| `TGDEV_API_ID` | `--api-id` |
| `TGDEV_API_HASH` | `--api-hash` |
| `TGDEV_BOT_TOKEN` | `--bot-token` |
| `TGDEV_SESSION` | `--session` |
| `TGDEV_PHONE` | `--phone` |

> [!IMPORTANT]
> Prefer environment variables over CLI flags for secrets. CLI args are visible in `ps aux` output.

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
> `--debug` logs full API request and response payloads to stderr. Do not use in shared terminals or redirect to persistent logs — the output may contain session tokens and other sensitive data.

## Architecture

```
cmd/tgdev/          CLI entrypoint and command implementations
internal/config/    Config loading, validation, credential layering
internal/ipc/       Unix domain socket IPC server and client
internal/mcpserver/ MCP server with tool definitions
invoke/             TL method registry, JSON unmarshal, invocation, response formatting
trace/              Correlation ID tracing, update listener, RPC middleware
```

The `invoke` package provides bidirectional lookup between TL schema names and gotd Go types using reflection over the full `tg.TypesMap()` and `tg.TypesConstructorMap()` registries. The `trace` package adds correlation IDs that link updates to handler invocations to API calls to responses.

## Dependencies

- [gotd/td](https://github.com/gotd/td) — MTProto client library
- [pageton/gotg](https://github.com/pageton/gotg) — Telegram framework (local replace directive)
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) — MCP server SDK
- [fatih/color](https://github.com/fatih/color) — Colored terminal output
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure-Go SQLite driver
