# invoke — TL Method Registry & Invocation

## Purpose
Bidirectional lookup between TL schema names and Go types. Generic TL method invocation via reflection over gotd's `tg.TypesMap()` and `tg.TypesConstructorMap()`.

## Files

| File | Description |
|------|-------------|
| `registry.go` | `Registry` struct: maps `idToName`, `nameToID`, `nameToType`, `nameToConstructor`, `ctors`. `GlobalRegistry()` returns singleton. Methods: `NameByID()`, `IDByName()`, `TypeByName()`, `ConstructorByName()`, `NewRequest()`, `Methods()`. |
| `invoke.go` | `Invoke()`: resolves method → `NewRequest()`, unmarshals JSON params, calls `client.Invoker().Invoke()`, decodes response via `decodeResponse()`. `captureDecoder` captures raw bytes. `decodeResponse()` handles bare vectors and constructor lookup. `genericVector` for decoded vector elements. |
| `invoker.go` | `DebugInvoker`: `telegram.Middleware` that logs requests/responses with timing. `resolveMethodName()` tries `TypeName()` then `TypeID()` + registry. `FormatDuration()` renders human-readable durations. Color variables exported for cross-package use. |
| `unmarshal.go` | `unmarshalRequest()`: case-insensitive JSON → gotd struct. `setField()` handles interface fields via `"_"` constructor key. `setInterfaceField()` looks up constructor in registry, creates concrete type. `setInterfaceSliceField()` handles `[]InterfaceClass`. |
| `format.go` | `InvokeJSON()`: alternative to `Invoke()` returning indented JSON. `jsonMarshalResult()` serializes responses. `jsonMarshalVector()` handles generic vectors. |
| `pretty.go` | Color definitions: `ColorMethod`, `ColorBody`, `ColorResponse`, `ColorResponseBody`, `ColorError`, `ColorTiming`. `FormatHexID()` for hex type IDs. `DefaultConfig()` returns sensible defaults. |

## Dependencies

**Imports:**
- `github.com/gotd/td/bin` — `bin.Encoder`, `bin.Decoder`, `bin.Buffer`, `bin.TypeVector`
- `github.com/gotd/td/tdp` — `tdp.Object`, `tdp.Format()`
- `github.com/gotd/td/tg` — `tg.TypesMap()`, `tg.TypesConstructorMap()`, request/response types

**Imported by:**
- `cmd/tgdev` — `GlobalRegistry()`, `Invoke()`, `InvokeJSON()`, `NewMiddleware()`
- `internal/mcpserver` — `GlobalRegistry()`, `FormatHexID()`
- `trace` — `NewRecordingDecoder()`, `Color*`, `FormatDuration()`, `GlobalRegistry()`

## Conventions

- Registry is a singleton (`GlobalRegistry()`) with `sync.Once` initialization
- Method names are "bare" (without `#hash` suffix): `messages.sendMessage` not `messages.sendMessage#545cd15a`
- JSON field matching is case-insensitive: `{"message": ...}` matches Go field `Message`
- Interface fields in JSON require `"_"` key: `{"_": "inputPeerUser", "UserID": 123}`
- Response decoding tries `tdp.Format()` first, falls back to `fmt.Sprintf()`

## Gotchas

- `decodeResponse()` safety limit: vectors capped at 100,000 elements to prevent OOM
- Partial vector results returned even if some elements fail to decode
- `resolveMethodName()` in both `invoke.go` and `trace/tracer.go` — duplicated logic
- `genericVector.String()` uses `tdp.Format()` for known types, `fmt.Sprintf()` for others
- `captureDecoder.Decode()` copies buffer before passing to real decoder
