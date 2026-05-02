# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## trace — Correlation ID Tracing

Full lifecycle tracing: atomic correlation IDs link updates → handler invocations → API calls → responses. Used by `tgdev trace` and as middleware in `tgdev listen`.

### Key Types

| Type             | File          | Role                                                                                                                  |
| ---------------- | ------------- | --------------------------------------------------------------------------------------------------------------------- |
| `Tracer`         | `tracer.go`   | Atomic `counter` (uint64, starts at 1). `Middleware()` wraps invocations with `[ID] >> method / << method [duration]` |
| `UpdateListener` | `listener.go` | Handles `tg.UpdateClass` and `tg.UpdatesClass`. Logs with correlation IDs, 30s heartbeat                              |

### Log Format

- Request: `[ID] >> method`
- Response: `[ID] << method [duration]`
- Error: `[ID] !! method [duration] error: ...`
- Update: `[ID] UPDATE <type>` or `[ID] UPDATES <type>`
- Heartbeat: `[ID] HEARTBEAT client alive` (every 30s)

### Gotchas

- `resolveMethodName()` is duplicated from `invoke/invoke.go` — consolidate if modifying either copy
- `UpdateListener` uses its own ID sequence separate from RPC call IDs
- `Tracer.Printf()` reimplements `invoke.DebugInvoker.printf()` — shared color handling possible
- Middleware creates new `invoke.RecordingDecoder()` per invocation to capture response bytes
