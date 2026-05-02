# internal/mcpserver — MCP Server

## Purpose
Exposes Telegram as MCP (Model Context Protocol) tools for LLM-driven interaction. Implements `tgdev_list_methods`, `tgdev_describe_method`, `tgdev_invoke`, `tgdev_get_me`, `tgdev_listener_status`, `tgdev_config_info`.

## Files

| File | Description |
|------|-------------|
| `server.go` | `New()` creates MCP server with registered tools. `ListMethods()`: paginated method listing with cursor. `DescribeMethod()`: reflects on request type, returns fields with JSON keys and constructor hints. `Invoke()`: calls `opts.Invoke()` (which tries IPC then standalone). `ListenerStatus()`: checks `ipc.Ping()`. `ConfigInfo()`: returns non-secret config. Input/output structs with `jsonschema` tags for MCP schema generation. |

## Dependencies

**Imports:**
- `github.com/modelcontextprotocol/go-sdk/mcp` — MCP server, tool registration, annotations
- `github.com/pageton/gotg-cli/invoke` — `GlobalRegistry()`, `FormatHexID()`
- `github.com/pageton/gotg-cli/internal/ipc` — `Ping()`

**Imported by:**
- `cmd/tgdev` — `New()`, `Options`

## Conventions

- Tool naming: `tgdev_<action>` (lowercase with underscores)
- `InvokeFunc` type: `func(ctx, method, params, format) (string, error)`
- Pagination: cursor-based with `strconv.Itoa()` encoding, default limit 100, max 500
- `normalizeFormat()`: defaults to `"text"`, accepts `"json"` (case-insensitive)
- Tool annotations: `ReadOnlyHint`, `OpenWorldHint`, `IdempotentHint`, `DestructiveHint`

## Gotchas

- `Invoke()` returns actionable errors: suggests starting `tgdev listen` or checking auth flags if IPC ping fails
- `DescribeMethod()` only works on struct types (not pointers, not interfaces) — unwraps `reflect.Pointer` first
- `InvokeOutput.ResultJSON` only populated when `format == "json"` and response is valid JSON
- MCP server `Invoke` tool has `OpenWorldHint: true` (LLM can call any method)
- HTTP mode (`--http`): stateless `StreamableHTTPHandler`, JSON responses only
