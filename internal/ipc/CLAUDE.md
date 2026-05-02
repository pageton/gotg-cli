# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## internal/ipc — Unix Domain Socket IPC

Inter-process communication over Unix domain sockets. Allows `tgdev listen` to serve invocation requests without reconnecting to Telegram. Self-contained — no internal dependencies, only stdlib (`net`, `encoding/json`).

### Protocol

JSON `InvokeRequest` → Unix socket → JSON `InvokeResponse`. Handler function type: `func(req InvokeRequest) InvokeResponse`.

### Limits

| Parameter                  | Value                     |
| -------------------------- | ------------------------- |
| Max request size           | 1 MiB (`maxRequestSize`)  |
| Max concurrent connections | 64 (`maxConcurrentConns`) |
| Connection deadline        | 30s                       |
| Invoke timeout             | 60s                       |
| Client dial timeout        | 5s                        |

### Conventions

- Socket file mode: 0600 (set after `net.Listen()`)
- Socket directory mode: 0700 (`os.MkdirAll()`)
- Server removes existing socket file before `net.Listen()` for clean startup
- `Close()` uses `sync.Once` to prevent double-close

### Gotchas

- `writerOnly` wrapper prevents `json.Encoder` from using `Conn.Read()` — don't remove it
- `acceptLoop()` checks `s.done` channel for graceful shutdown
- `Ping()` only checks if the socket file exists and a connection can be established
