# gotg-cli (tgdev) — Project Context

## Role
Telegram MTProto debug and invoke CLI. Call any TL method, trace API calls with correlation IDs, and expose Telegram as an MCP server for LLM-driven interaction.

## Architecture

```
.
├── cmd/tgdev/              CLI entry point and command implementations
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
│   ├── SKILL.md          Agent instructions and workflow
│   ├── evals/            Evaluation suite
│   ├── scripts/           Helper scripts (setup_test_group.sh)
│   └── references/        TL method references, error codes, interaction patterns
├── go.mod                 Module: github.com/pageton/gotg-cli (Go 1.25.0)
├── go.sum
└── README.md              User-facing documentation
```

## Key Files

| File | Description |
|------|-------------|
| `cmd/tgdev/main.go` | Entry point: command dispatch (`invoke`, `listen`, `trace`, `mcp`, `methods`, `getme`, `export-session`), flag parsing, gotg client creation with session handling |
| `invoke/registry.go` | `GlobalRegistry()`: singleton mapping TL schema names ↔ type IDs ↔ Go reflect.Types ↔ constructors. Built from `tg.TypesMap()` and `tg.TypesConstructorMap()` |
| `invoke/invoke.go` | `Invoke()`: resolves method name → request struct, unmarshals JSON params, calls `client.Invoker().Invoke()`, decodes response |
| `invoke/invoker.go` | `DebugInvoker`: telegram.Middleware that logs requests/responses with timing, method name resolution, color output |
| `invoke/unmarshal.go` | `unmarshalRequest()`: case-insensitive JSON → gotd struct; handles interface fields via `"_"` constructor key |
| `invoke/format.go` | `InvokeJSON()`: alternative entry point returning indented JSON instead of tdp.Format text |
| `internal/config/config.go` | `Config` struct: API credentials, session string, bot token, phone, DB path, socket path. `FromFlags()`: CLI > env (`TGDEV_*`) > config file. `Validate()` checks required fields |
| `internal/ipc/ipc.go` | Unix socket IPC: `Server` (accepts JSON InvokeRequest, dispatches to Handler), `Client` (connects, sends request, reads response). Socket permissions: 0600 |
| `internal/mcpserver/server.go` | MCP server tools: `tgdev_list_methods` (paginated), `tgdev_describe_method`, `tgdev_invoke`, `tgdev_get_me`, `tgdev_listener_status`, `tgdev_config_info` |
| `trace/tracer.go` | `Tracer`: atomic correlation ID counter, `Middleware()` wraps invocations with `[ID] >> method` / `[ID] << method [duration]` logging |
| `trace/listener.go` | `UpdateListener`: logs incoming updates with correlation IDs, heartbeat every 30s |

## Module Map

```
cmd/tgdev ──→ invoke (Invoke, GlobalRegistry, DebugInvoker)
         ──→ internal/config (Config, FromFlags, Validate)
         ──→ internal/ipc (NewServer, NewClient, Ping)
         ──→ internal/mcpserver (New, Options)
         ──→ trace (NewTracer, Tracer, UpdateListener)
         ──→ github.com/pageton/gotg (gotg.Client, session, functions)
         └──→ github.com/gotd/td (telegram, tg, tdp, bin)

invoke ──→ github.com/gotd/td (tg, tdp, bin)
       └──→ stdlib (encoding/json, reflect)

internal/ipc ──→ stdlib (net, encoding/json)
             └──→ no internal deps

internal/mcpserver ──→ invoke (GlobalRegistry, Invoke, FormatHexID)
                ──→ internal/ipc (Ping)
                └──→ github.com/modelcontextprotocol/go-sdk (mcp)

trace ──→ invoke (NewRecordingDecoder, Color*, FormatDuration, resolveMethodName)
      ──→ github.com/gotd/td (tg, tdp, bin, telegram)
      └──→ github.com/pageton/gotg-cli/invoke
```

## Data Flow

1. **Standalone invoke**: `tgdev invoke <method> <json>` → `cmd/tgdev` → `invokeMethod()` → `createClient()` (gotg + session) → `invoke.Invoke()` → `client.Invoker().Invoke()` → Telegram API
2. **IPC fast path**: `tgdev listen` starts `internal/ipc.Server` on Unix socket. Subsequent `tgdev invoke` calls → `ipc.Client` → JSON request over socket → `ipc.Server` → `invoke.Invoke()` → Telegram API → JSON response
3. **MCP server**: `tgdev mcp` → `internal/mcpserver.New()` → registers tools → tools call `invokeMethod()` → IPC socket or standalone client
4. **Tracing**: `tgdev trace` → `trace.Tracer.Middleware()` wraps every invocation with `[correlationID] >> method / << method [duration]`. `trace.UpdateListener` logs incoming updates with IDs

## Dependencies

### External (go.mod)
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gotd/td` | v0.143.0 | MTProto client library (tg, tdp, bin, telegram) |
| `github.com/pageton/gotg` | v0.0.0 (local replace) | Telegram framework: session, storage, functions |
| `github.com/modelcontextprotocol/go-sdk` | v1.5.0 | MCP server SDK |
| `github.com/fatih/color` | v1.19.0 | ANSI color output for terminal logging |
| `modernc.org/sqlite` | v1.38.2 | Pure-Go SQLite driver for session storage |
| `golang.org/x/oauth2` | v0.35.0 | OAuth2 (indirect, from gotg) |

### Internal
- `invoke` package is the core: used by `mcpserver`, `trace`, `cmd/tgdev`
- `internal/config` is self-contained, used only by `cmd/tgdev`
- `internal/ipc` is self-contained protocol, used by `mcpserver` and `cmd/tgdev`
- No circular dependencies

## Conventions

### Coding Style
- Standard Go formatting (`gofmt`)
- Package-level docs for all exported types and functions
- Error wrapping with `fmt.Errorf("context: %w", err)` (Go 1.25 style)
- Reflection used in `invoke/registry.go` and `invoke/unmarshal.go` for TL schema ↔ Go type mapping

### Command Naming
- CLI: `tgdev <command> [flags]` (lowercase, no hyphens)
- MCP tools: `tgdev_<command>` (lowercase with underscores)
- Internal functions: `camelCase` or `PascalCase` per Go visibility rules

### Commit Style
- Semantic prefixes: `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`
- Imperative mood, <= 72 chars

### Branches
- No enforced convention (inferred from single-main development)

## Build & Test

### Build
```bash
# Basic build
go build -o tgdev ./cmd/tgdev/

# With version info
go build -ldflags "-X main.version=$(git describe --tags --always) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o tgdev ./cmd/tgdev/
```

### Test
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./invoke/...
go test ./internal/...

# With verbose output
go test -v ./...
```

### Lint
```bash
# Standard Go vet
go vet ./...
```

## Gotchas

### Session Formats
- `detectSessionFormat()` in `cmd/tgdev/main.go` auto-detects: gotg (native JSON), Pyrogram (URL-safe base64, 271 bytes), Telethon (prefix '1' + URL-safe base64, 263/275 bytes), Gramjs (prefix '1' + standard base64)
- SQLite (`--db`) takes priority over session string (`--session`)
- Session strings are sensitive: never log, commit, or expose in process args (use `TGDEV_*` env vars)

### IPC Socket Permissions
- Socket file is `chmod 0600` after `net.Listen()`
- Default path: `$XDG_RUNTIME_DIR/tgdev.sock` or `/run/user/<uid>/tgdev.sock`
- Config file `~/.tgdev.json` is also restricted to `0600`

### Invoke JSON Parameter Format
- Interface fields (like `InputPeer`, `InputUser`) require `"_"` constructor key: `{"_": "inputPeerUser", "UserID": 123, "AccessHash": "..."}`
- Case-insensitive field matching: JSON key `message` matches Go field `Message`
- `nil` params allowed for methods with no arguments

### Vector Response Handling
- `invoke/decodeResponse()` handles bare vectors (`bin.TypeVector`) with safety limit: 100,000 elements max
- Partial results returned even if some vector elements fail to decode

### MCP Server Modes
- stdio mode (default): for Claude Code, Cursor, local MCP clients
- HTTP mode (`--http :8080`): stateless Streamable HTTP for remote/multi-client
- When `tgdev listen` is running, MCP tools route through IPC socket (zero-reconnect overhead)

## Security Considerations

### Auth Credentials
- API credentials: `api_id` (int), `api_hash` (string) — required for all operations
- Auth methods (priority order): `--db` (SQLite) > `--session` (string) > `--bot-token` > `--phone`
- Bot tokens and session strings are equivalent to passwords — full account access

### Secret Handling
- Config file `~/.tgdev.json` auto-restricted to `0600` on load
- SQLite database `chmod 0600` after opening
- IPC socket `chmod 0600` after creation
- `--debug` flag logs FULL request/response payloads to stderr — contains session tokens, use with caution
- CLI args visible in `ps aux` — prefer `TGDEV_*` environment variables for secrets

### Trust Boundaries
- IPC socket accepts arbitrary `InvokeRequest` JSON — only local clients (Unix socket) should have access
- MCP server tools invoke arbitrary TL methods — auth is via tgdev's own auth flags, not MCP client auth
- `tg-dev-agent` skill requires a **user account** session (not bot) for group creation, member management, admin promotion

### Input Validation
- `invoke/registry.go`: method name validated against `GlobalRegistry()` before invocation
- `internal/ipc/ipc.go`: max request size 1 MiB, max 64 concurrent connections, 30s connection deadline, 60s invoke timeout
- `invoke/decodeResponse()`: vector element count capped at 100,000 to prevent OOM
