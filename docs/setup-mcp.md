# tgdev MCP Server Setup Guide

Complete guide to setting up `tgdev` as an MCP (Model Context Protocol) server for LLM-driven Telegram interaction.

## What is MCP?

The Model Context Protocol (MCP) lets LLMs like Claude, GPT, and others call external tools. tgdev exposes the entire Telegram API as MCP tools, allowing LLMs to:

- List available TL methods (`tgdev_list_methods`)
- Describe method parameters (`tgdev_describe_method`)
- Invoke any TL method (`tgdev_invoke`)
- Get account info (`tgdev_get_me`)
- Check listener status (`tgdev_listener_status`)

## Prerequisites

1. **Install tgdev:**
   ```bash
   go install github.com/pageton/gotg-cli/cmd/tgdev@latest
   # Or build from source:
   go build -o tgdev ./cmd/tgdev/
   ```

2. **Get Telegram API credentials:**
   - Visit [my.telegram.org](https://my.telegram.org)
   - Go to "API Development Tools"
   - Create an application to get `api_id` and `api_hash`

3. **Choose an auth method:**
   - Bot token (fastest for bots): `@BotFather` → `/newbot`
   - User session string: `tgdev --phone <number>` → `tgdev export-session --db ~/tgdev.db`
   - SQLite database: `tgdev listen --session <string> --db ~/tgdev.db`

## Setup Methods

### Method 1: stdio Mode (Recommended for local LLMs)

stdio mode is best for Claude Code, Cursor, and local MCP clients that communicate via stdin/stdout.

#### Step 1: Create config file (optional, recommended)

Create `~/.tgdev.json` to avoid passing credentials as CLI args:

```json
{
  "api_id": 12345,
  "api_hash": "your_api_hash",
  "bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
  "database_path": "~/tgdev.db"
}
```

Restrict permissions: `chmod 0600 ~/.tgdev.json`

#### Step 2: Configure your MCP client

**Claude Code (`.claude/mcp.json`):**
```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp"],
      "env": {
        "TGDEV_API_ID": "12345",
        "TGDEV_API_HASH": "your_api_hash",
        "TGDEV_BOT_TOKEN": "your_bot_token"
      }
    }
  }
}
```

**Cursor (`~/.cursor/mcp.json`):**
```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp"]
    }
  }
}
```

**Generic MCP client:**
```json
{
  "mcpServers": {
    "tgdev": {
      "command": "/path/to/tgdev",
      "args": ["mcp", "--api-id", "12345", "--api-hash", "hash", "--bot-token", "token"]
    }
  }
}
```

#### Step 3: Start tgdev MCP server

With config file:
```bash
tgdev mcp
```

With CLI flags:
```bash
tgdev mcp --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

With environment variables:
```bash
export TGDEV_API_ID=12345
export TGDEV_API_HASH=YOUR_HASH
export TGDEV_BOT_TOKEN=YOUR_TOKEN
tgdev mcp
```

### Method 2: HTTP Mode (Remote/Multi-client)

HTTP mode uses stateless Streamable HTTP, suitable for remote setups or multiple clients.

#### Start HTTP server:

```bash
tgdev mcp --http :8080 --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

The server listens on `http://localhost:8080` (or your chosen port).

#### Connect from MCP clients:

Some MCP clients support HTTP transports. Check your client's documentation.

### Method 3: With Persistent Listener (Fastest)

When `tgdev listen` or `tgdev trace` is running, MCP invocations route through the listener's IPC socket — no reconnection overhead.

#### Terminal 1: Start listener
```bash
tgdev listen --db ~/tgdev.db --api-id 12345 --api-hash YOUR_HASH --bot-token YOUR_TOKEN
```

#### Terminal 2: Start MCP server
```bash
tgdev mcp
```

The MCP server automatically detects the listener and uses its IPC socket.

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `tgdev_list_methods` | List TL methods with optional prefix filtering (`prefix`: "messages.", "users.get") and cursor pagination (`cursor`, `limit`) |
| `tgdev_describe_method` | Show fields, types, and constructor hints for a method (`method`: "messages.sendMessage") |
| `tgdev_invoke` | Invoke any TL method with JSON params (`method`, `params`, `format`: "text" or "json") |
| `tgdev_get_me` | Get current account info (`users.getFullUser` for inputUserSelf) |
| `tgdev_listener_status` | Check if a listener is reachable on the IPC socket |
| `tgdev_config_info` | Show non-secret MCP configuration (socket path, format, version) |

## Example Usage (from LLM)

Once configured, your LLM can use tgdev tools:

**List methods:**
```
<invoke name="tgdev_list_methods">
  <parameter name="prefix">messages.send</parameter>
</invoke>
```

**Describe a method:**
```
<invoke name="tgdev_describe_method">
  <parameter name="method">messages.sendMessage</parameter>
</invoke>
```

**Invoke a method:**
```
<invoke name="tgdev_invoke">
  <parameter name="method">messages.sendMessage</parameter>
  <parameter name="params">{"Peer": {"_": "inputPeerUser", "UserID": 123, "AccessHash": "abc"}, "Message": "Hello!"}</parameter>
</invoke>
```

## JSON Parameter Format

Interface fields (like `InputPeer`, `InputUser`) use a `"_"` constructor key:

```json
{
  "Peer": {
    "_": "inputPeerUser",
    "UserID": 123456,
    "AccessHash": "abc123..."
  },
  "Message": "Hello from LLM!"
}
```

To discover constructor names:
```
tgdev_describe_method with method="messages.sendMessage"
→ Shows "Peer" field with constructor hint: use {"_": "inputPeerUser"}
```

## Auth Methods for MCP

| Method | MCP Config | Notes |
|--------|-------------|-------|
| Bot token | `"args": ["mcp", "--bot-token", "TOKEN"]` | Fastest for bots |
| Environment vars | `"env": {"TGDEV_BOT_TOKEN": "TOKEN"}` | Avoids exposing in process args |
| Config file | `"args": ["mcp"]` + `~/.tgdev.json` | Cleanest setup |
| Session string | `"args": ["mcp", "--session", "STRING"]` | User accounts |
| SQLite DB | `"args": ["mcp", "--db", "~/tgdev.db"]` | Persistent sessions |

## Troubleshooting

### MCP server won't start
- Check API credentials: `tgdev getme --api-id ... --api-hash ... --bot-token ...`
- Verify tgdev is in PATH: `which tgdev`
- Check config file permissions: `ls -la ~/.tgdev.json` (should be 0600)

### LLM can't call tools
- Verify MCP client config: check `.claude/mcp.json` or equivalent
- Check MCP server is running: `tgdev_listener_status` tool
- Review server logs (stderr) for connection errors

### Invocation errors
- Method not found: use `tgdev_list_methods` to discover valid method names
- Parameter errors: use `tgdev_describe_method` to see required fields
- Auth errors: ensure bot token/session is valid with `tgdev getme`

### Listener not detected
- Start `tgdev listen` before `tgdev mcp`
- Check socket path: `ls -la $XDG_RUNTIME_DIR/tgdev.sock`
- Default socket: `~/.tgdev/tgdev.sock`

## Security Notes

- **Never commit credentials** to version control
- **Use environment variables** (`TGDEV_*`) instead of CLI args to avoid exposing secrets in `ps aux`
- **Restrict config file** to owner-only: `chmod 0600 ~/.tgdev.json`
- **IPC socket** is mode 0600 (owner-only access)
- **`--debug` flag** logs full API payloads to stderr — contains session tokens, use with caution
- **Bot tokens = full account access** — treat them like passwords

## Advanced: Custom Socket Path

```bash
tgdev mcp --socket /custom/path/tgdev.sock --api-id 12345 --api-hash HASH --bot-token TOKEN
```

Update your config file:
```json
{
  "socket_path": "/custom/path/tgdev.sock"
}
```

## Reference

- [MCP Specification](https://modelcontextprotocol.io)
- [tgdev README](../README.md)
- [gotd/td Library](https://github.com/gotd/td)
- [Telegram API Documentation](https://core.telegram.org/api)
