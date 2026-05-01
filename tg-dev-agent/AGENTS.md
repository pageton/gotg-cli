# tg-dev-agent — Telegram Development Agent Skill

## Purpose
Agent skill for building, testing, and simulating Telegram bots end-to-end using a real userbot session. Covers group creation, bot promotion, interaction simulation (messages, inline buttons, callback queries, deep links), update monitoring, and validation — all via the tgdev CLI and MCP integration.

## Files

| File | Description |
|------|-------------|
| `SKILL.md` | Agent instructions: workflow phases (environment setup, bot resolution, group setup, dual-mode testing, monitoring, reporting), error handling, rate limiting, reference file pointers |
| `evals/evals.json` | Evaluation suite configuration |
| `scripts/setup_test_group.sh` | Helper script: automates group creation, bot invitation, admin promotion |
| `references/errors.md` | Telegram error codes, meanings, and recovery strategies |
| `references/common-methods.md` | Quick-reference for frequently used TL methods |
| `references/monitoring.md` | Update monitoring: parsing, matching, validation patterns |
| `references/interactions.md` | Interaction patterns: messages, buttons, deep links, edge cases for DM and group contexts |
| `references/group-ops.md` | Group lifecycle: create, upgrade to supergroup, add bot, promote admin |

## Dependencies

**Imports (from SKILL.md references):**
- `tgdev` CLI — `invoke`, `listen`, `trace`, `mcp`, `getme`, `export-session`, `methods`
- `tgdev` MCP tools — `tgdev_invoke`, `tgdev_list_methods`, `tgdev_describe_method`, `tgdev_get_me`, `tgdev_listener_status`
- Telegram API (via tgdev) — `contacts.resolveUsername`, `messages.sendMessage`, `messages.getBotCallbackAnswer`, `channels.inviteToChannel`, `channels.editAdmin`, `messages.createChat`, `messages.migrateChat`, `users.getFullUser`

**Imported by:** Nothing (standalone skill directory).

## Conventions

- Skill trigger: mentions of "Telegram bots", "userbots", "MTProto", "bot testing", "tgdev", "gotg", or Telegram automation
- Session requirement: **user account** (not bot) — sessions from Pyrogram/Telethon/gotg format
- Dual-mode testing: every test must cover both DM (private chat) and group/supergroup contexts
- JSON interface fields: use `"_"` constructor key: `{"_": "inputPeerUser", ...}`
- Credential priority: CLI flags > `TGDEV_*` env vars > `~/.tgdev.json`
- Rate limiting: 0.5-1s delays between messages, max ~20 msgs/minute to same peer

## Gotchas

- Session must be from a **user account** (`bot: false` in `getme`) — bot sessions can't create groups, add members, or promote admins
- API ID must be a standard 5+ digit ID from my.telegram.org — third-party/test IDs (single-digit) have restricted method access
- `FLOOD_WAIT_%d` errors: must wait the specified seconds, never spam through
- In groups, commands may need `@bot_username` suffix to target the bot
- `callback_query` data: capture `msg_id` and `button_data` from the message that produced inline buttons
- Deep links only work in DM context, not groups
