# internal/config — Configuration Management

## Purpose
CLI configuration: API credentials, session handling, output settings. Supports layered config: CLI flags > environment variables > JSON config file.

## Files

| File | Description |
|------|-------------|
| `config.go` | `Config` struct: `APIID`, `APIHash`, `Session`, `BotToken`, `Phone`, `DatabasePath`, `DatabaseName`, `SocketPath`, `Format`, `Debug`, `NoColor`. `DefaultPath()` returns `~/.tgdev.json`. `DefaultDBPath()` returns `~/tgdev/tgdev.db`. `Load()` reads JSON, fixes permissions to 0600 if overly permissive. `Save()` writes JSON with 0600. `FromFlags()` builds config from CLI flags with env var fallback. `Validate()` checks required fields. `ParseAPIID()` converts string to int. `GetSocketPath()` defaults to `$XDG_RUNTIME_DIR/tgdev.sock`. |

## Dependencies

**Imports:**
- `encoding/json` — JSON marshaling/unmarshaling
- `os` — file operations, env vars (`TGDEV_API_ID`, `TGDEV_API_HASH`, `TGDEV_BOT_TOKEN`, `TGDEV_SESSION`, `TGDEV_PHONE`)
- `path/filepath` — path joining for home dir, socket paths

**Imported by:**
- `cmd/tgdev` — `Config`, `FromFlags()`, `Validate()`, `DefaultPath()`, `DefaultSocketPath()`

## Conventions

- Config file: `~/.tgdev.json` (mode 0600, auto-corrected if wrong)
- Environment variables: `TGDEV_*` prefix (e.g., `TGDEV_API_ID`)
- CLI flags override env vars, which override config file
- `Validate()` requires: `api_id` + `api_hash` + one auth method (`--db`, `--session`, `--bot-token`, `--phone`)
- `DatabaseName` defaults to `"default"` if empty

## Gotchas

- `FromFlags()` silently ignores missing config file (uses env vars and flags only)
- `Load()` warns and fixes config file permissions if not 0600 — potential race if multiple processes check simultaneously
- `GetSocketPath()` falls back to `/run/user/<uid>/tgdev.sock` if `$XDG_RUNTIME_DIR` not set and home dir fails
- `DefaultDBPath()` creates `~/.tgdev/` directory implicitly via `sqlitedb.NewFromDSN()` elsewhere
