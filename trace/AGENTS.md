# trace — Correlation Tracing

## Purpose
Full lifecycle tracing: correlation IDs link updates → handler invocations → API calls → responses. Used by `tgdev trace` and as middleware in `tgdev listen`.

## Files

| File | Description |
|------|-------------|
| `tracer.go` | `Tracer`: atomic `counter` for correlation IDs, `NextID()` returns monotonically increasing uint64. `Middleware()` returns `telegram.Middleware` that wraps invocations with `[ID] >> method` / `[ID] << method [duration]` logging. `resolveMethodName()` duplicates logic from `invoke/invoke.go`. |
| `listener.go` | `UpdateListener`: handles `tg.UpdateClass` and `tg.UpdatesClass`, logs with correlation IDs. `StartUpdateLogger()` runs goroutine with 30s heartbeat. Color definitions: `colorUpdate`, `colorTiming`. |

## Dependencies

**Imports:**
- `github.com/gotd/td/bin` — `bin.Encoder`, `bin.Decoder`
- `github.com/gotd/td/tdp` — `tdp.Object`, `tdp.Format()`
- `github.com/gotd/td/tg` — `tg.UpdateClass`, `tg.UpdatesClass`
- `github.com/gotd/td/telegram` — `telegram.Middleware`
- `github.com/pageton/gotg-cli/invoke` — `NewRecordingDecoder()`, `Color*`, `FormatDuration()`, `GlobalRegistry()`, `resolveMethodName()` (duplicated)

**Imported by:**
- `cmd/tgdev` — `NewTracer()`, `Tracer`, `UpdateListener`

## Conventions

- Correlation IDs are uint64, start at 1, increment atomically
- Log format: `[ID] >> method`, `[ID] << method [duration]`, `[ID] !! method [duration] error: ...`
- Updates logged as `[ID] UPDATE <type>` or `[ID] UPDATES <type>`
- Heartbeat every 30s: `[ID] HEARTBEAT client alive`

## Gotchas

- `resolveMethodName()` is duplicated in `invoke/invoke.go` and `trace/tracer.go` — should be consolidated
- `UpdateListener.HandleRawUpdate()` only logs `tg.UpdatesClass`, not individual updates inside
- `Tracer.Printf()` reimplements `invoke.DebugInvoker.printf()` — shared color handling possible
- Middleware creates new `invoke.RecordingDecoder()` per invocation to capture response bytes
- `listener.go` doesn't use `Tracer.NextID()` for updates — they get separate ID sequence from RPC calls
