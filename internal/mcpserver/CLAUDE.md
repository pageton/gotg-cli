# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## internal/mcpserver — MCP Server

Exposes Telegram as MCP tools for LLM-driven interaction. Uses the MCP Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`).

### Tools

| Tool                    | Purpose                                                                       |
| ----------------------- | ----------------------------------------------------------------------------- |
| `tgdev_list_methods`    | Paginated method listing with cursor (default 100, max 500)                   |
| `tgdev_describe_method` | Reflects on request type, returns fields with JSON keys and constructor hints |
| `tgdev_invoke`          | Calls `opts.Invoke()` — tries IPC first, falls back to standalone             |
| `tgdev_get_me`          | Returns current user info                                                     |
| `tgdev_listener_status` | Checks `ipc.Ping()` on socket                                                 |
| `tgdev_config_info`     | Returns non-secret config                                                     |

### Key Types

- `InvokeFunc`: `func(ctx, method, params, format) (string, error)` — injected dependency for invocation
- `Options`: Holds `InvokeFunc`, socket path, and format default
- Input/output structs use `jsonschema` tags for MCP schema generation

### Conventions

- Tool naming: `tgdev_<action>`
- `normalizeFormat()`: defaults to `"text"`, accepts `"json"` (case-insensitive)
- Tool annotations set `ReadOnlyHint`, `OpenWorldHint`, `IdempotentHint`, `DestructiveHint`

### Gotchas

- `Invoke()` returns actionable errors — suggests starting `tgdev listen` or checking auth if IPC fails
- `DescribeMethod()` only works on struct types — unwraps `reflect.Pointer` first
- `InvokeOutput.ResultJSON` only populated when `format == "json"` and response is valid JSON
- HTTP mode (`--http`): stateless `StreamableHTTPHandler`, JSON responses only
