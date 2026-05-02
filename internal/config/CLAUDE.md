# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## internal/config — Configuration Management

CLI configuration with credential layering: CLI flags > `TGDEV_*` env vars > `~/.tgdev.json` config file.

### Key Functions

| Function            | Role                                                                                              |
| ------------------- | ------------------------------------------------------------------------------------------------- |
| `FromFlags()`       | Builds config from CLI flags with env var fallback. Silently ignores missing config file.         |
| `Load()` / `Save()` | Read/write `~/.tgdev.json`. Auto-corrects permissions to 0600 if overly permissive.               |
| `Validate()`        | Requires: `api_id` + `api_hash` + one auth method (`--db`, `--session`, `--bot-token`, `--phone`) |
| `GetSocketPath()`   | Defaults to `$XDG_RUNTIME_DIR/tgdev.sock`, falls back to `~/.tgdev/tgdev.sock`                    |
| `ParseAPIID()`      | Converts string to int for `api_id`                                                               |

### Config Struct Fields

`APIID`, `APIHash`, `Session`, `BotToken`, `Phone`, `DatabasePath`, `DatabaseName`, `SocketPath`, `Format`, `Debug`, `NoColor`

### Gotchas

- `Load()` warns and fixes config file permissions — potential race if multiple processes check simultaneously
- `DefaultDBPath()` creates `~/.tgdev/` directory implicitly via `sqlitedb.NewFromDSN()` in `cmd/tgdev`
- `DatabaseName` defaults to `"default"` if empty
