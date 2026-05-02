# internal/ipc — Unix Socket IPC

## Purpose
Inter-process communication over Unix domain sockets. Allows `tgdev listen` to serve invocation requests from `tgdev invoke` (or MCP server) without reconnecting to Telegram.

## Files

| File | Description |
|------|-------------|
| `ipc.go` | `InvokeRequest`/`InvokeResponse` JSON protocol. `Server`: listens on Unix socket, accepts JSON requests, dispatches to `Handler` func, limits to 64 concurrent connections, 1 MiB max request size, 30s connection deadline, 60s invoke timeout. `Client`: connects to socket, sends `InvokeRequest`, reads `InvokeResponse`. `Ping()` checks if server is listening. Socket permissions restricted to 0600. |

## Dependencies

**Imports:**
- `net` — `net.Listen("unix", ...)`, `net.DialTimeout("unix", ...)`
- `encoding/json` — request/response marshaling
- `os` — socket file creation, `os.Remove()` for cleanup

**Imported by:**
- `cmd/tgdev` — `NewServer()`, `NewClient()`, `Ping()`
- `internal/mcpserver` — `Ping()`

## Conventions

- Socket path: `$XDG_RUNTIME_DIR/tgdev.sock` or `~/.tgdev/tgdev.sock`
- Socket file mode: 0600 (set after `net.Listen()`)
- Directory created with `os.MkdirAll()` at 0700
- `maxConcurrentConns = 64`, `maxRequestSize = 1 << 20` (1 MiB)
- Connection deadline: 30s, invoke timeout: 60s

## Gotchas

- Server removes existing socket file before `net.Listen()` — ensures clean startup
- `writerOnly` wrapper prevents `json.NewEncoder()` from using `Conn.Read()`
- `acceptLoop()` checks `s.done` channel for graceful shutdown
- `Close()` uses `sync.Once` to prevent double-close
- Client dial timeout: 5s, respects context deadline if set
