# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## invoke — TL Method Registry & Invocation

Core package: bidirectional lookup between TL schema names and Go types, generic TL method invocation via reflection over gotd's `tg.TypesMap()` and `tg.TypesConstructorMap()`.

### Key Types

| Type                 | File           | Role                                                                                                                 |
| -------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------- |
| `Registry`           | `registry.go`  | Singleton (`GlobalRegistry()`, `sync.Once`). Maps `idToName`, `nameToID`, `nameToType`, `nameToConstructor`, `ctors` |
| `Invoke()`           | `invoke.go`    | Resolves method name → request struct, unmarshals JSON, calls `client.Invoker().Invoke()`, decodes response          |
| `InvokeJSON()`       | `format.go`    | Like `Invoke()` but returns indented JSON instead of `tdp.Format` text                                               |
| `DebugInvoker`       | `invoker.go`   | `telegram.Middleware` for logging requests/responses with timing and method name resolution                          |
| `unmarshalRequest()` | `unmarshal.go` | Case-insensitive JSON → gotd struct with interface constructor resolution                                            |

### Conventions

- Method names are "bare" — no `#hash` suffix
- JSON field matching is case-insensitive
- Interface fields require `"_"` constructor key: `{"_": "inputPeerUser", "UserID": 123}`
- Response decoding tries `tdp.Format()` first, falls back to `fmt.Sprintf()`
- Registry is a singleton — always use `GlobalRegistry()`, never construct manually

### Gotchas

- `decodeResponse()` caps vectors at 100,000 elements to prevent OOM
- Partial vector results returned even if some elements fail to decode
- `resolveMethodName()` is duplicated in `trace/tracer.go` — consolidate if modifying
- `captureDecoder.Decode()` copies buffer before passing to real decoder
