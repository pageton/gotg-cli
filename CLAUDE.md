# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tgdev** — a Go CLI for interacting with the Telegram MTProto API. Call any TL method, trace API calls with correlation IDs, and expose Telegram as an MCP server for LLM-driven interaction.

Module: `github.com/pageton/gotg-cli` (Go 1.25.0)

**Critical dependency:** `github.com/pageton/gotg` is a local `replace` directive pointing to `/home/sadiq/Projects/go/gotg`. Changes to gotg affect this project.

## Build, Test, Lint

```bash
# Build (basic)
go build -o tgdev ./cmd/tgdev/

# Build with version info
go build -ldflags "-X main.version=$(git describe --tags --always) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o tgdev ./cmd/tgdev/

# Test all
go test ./...

# Test specific package
go test ./invoke/...
go test ./internal/...

# Verbose
go test -v ./...

# Lint
go vet ./...
```

Tests exist in: `invoke/`, `internal/ipc/`, `internal/mcpserver/`. No tests for `cmd/tgdev/`, `trace/`, or `internal/config/`.

## Architecture

```
cmd/tgdev/       CLI entry point — command dispatch, flag parsing, client creation (no CLI framework)
invoke/          Core: TL method registry (reflection-based), JSON unmarshal with interface resolution, invocation
internal/config  Config layering: CLI flags > TGDEV_* env vars > ~/.tgdev.json
internal/ipc     Unix domain socket IPC — avoids Telegram reconnects when listener is running
internal/mcpserver  MCP server exposing 6 tools (list_methods, describe_method, invoke, get_me, listener_status, config_info)
trace/           Correlation ID tracing linking updates → API calls → responses
```

### Data Flows

1. **Standalone invoke**: `tgdev invoke` → create gotg client → `invoke.Invoke()` → Telegram API
2. **IPC fast path**: `tgdev listen` starts IPC server → subsequent `tgdev invoke` calls go through Unix socket → reuse existing Telegram connection
3. **MCP server**: `tgdev mcp` → tool handlers → IPC socket (if listener running) or standalone client
4. **Tracing**: `tgdev trace` → middleware wraps invocations with `[correlationID] >> method / << method [duration]`

### Dependency Graph

```
cmd/tgdev → invoke, internal/config, internal/ipc, internal/mcpserver, trace, gotg
invoke → gotd/td (tg, tdp, bin)
internal/ipc → stdlib only (net, encoding/json)
internal/mcpserver → invoke, internal/ipc, MCP Go SDK
trace → invoke, gotd/td
```

No circular dependencies. `internal/ipc` is fully self-contained.

## Key Design Patterns

- **TL Registry** (`invoke/registry.go`): Singleton (`sync.Once`) mapping TL schema names ↔ type IDs ↔ Go `reflect.Type`. Built from `tg.TypesMap()` and `tg.TypesConstructorMap()`. All method resolution flows through `GlobalRegistry()`.
- **JSON unmarshalling** (`invoke/unmarshal.go`): Case-insensitive JSON → gotd struct. Interface fields (like `InputPeer`) require `"_"` constructor key: `{"_": "inputPeerUser", "UserID": 123}`.
- **IPC protocol** (`internal/ipc/ipc.go`): `InvokeRequest`/`InvokeResponse` JSON over Unix socket. Limits: 1 MiB max request, 64 concurrent connections, 30s connection deadline, 60s invoke timeout.
- **Command dispatch** (`cmd/tgdev/main.go`): Hand-rolled flag parsing and `switch` statement — no CLI framework like cobra. Command functions follow `cmd<Name>(args []string) error` pattern.

## Conventions

- CLI commands: lowercase, no hyphens (`tgdev invoke`, `tgdev listen`)
- MCP tools: `tgdev_<action>` with underscores
- Method names are "bare" — without `#hash` suffix (`messages.sendMessage`, not `messages.sendMessage#545cd15a`)
- Commit style: semantic prefixes (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`), imperative mood, ≤72 chars
- Error wrapping: `fmt.Errorf("context: %w", err)`

## Security-Sensitive Patterns

- Session strings, bot tokens, and config files are equivalent to full account access
- `--debug` flag logs **full** request/response payloads including session tokens
- CLI args are visible in `ps aux` — prefer `TGDEV_*` env vars for secrets
- Config file (`~/.tgdev.json`), SQLite DB, and IPC socket all auto-restricted to `0600`
- Vector responses capped at 100,000 elements to prevent OOM

## Gotchas

- `resolveMethodName()` is duplicated between `invoke/invoke.go` and `trace/tracer.go`
- `internal/ipc/ipc.go` uses `writerOnly` wrapper to prevent `json.Encoder` from using `Conn.Read()`
- `detectSessionFormat()` auto-detects session types: gotg (native JSON), Pyrogram, Telethon, GramJS, mtcute
- SQLite (`--db`) takes priority over session string (`--session`) for auth
- MCP HTTP mode (`--http`) is stateless — each request may create a new Telegram connection unless listener is running
