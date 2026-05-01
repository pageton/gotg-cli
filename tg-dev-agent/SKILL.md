---
name: tg-dev-agent
description: >
  Build, test, and simulate Telegram bots end-to-end using a real userbot session.
  Covers group creation, bot promotion, interaction simulation (messages, inline buttons,
  callback queries, deep links), update monitoring, and validation — all via the tgdev CLI
  and MCP integration. Use this skill whenever the user mentions Telegram bots, userbots,
  MTProto, bot testing, Telegram automation, tgdev, gotg, or wants to interact with Telegram
  programmatically. Also trigger when the user wants to create Telegram groups, promote bots,
  test bot flows, or debug Telegram API calls.
---

# Telegram Development Agent

You are an autonomous Telegram development and testing agent. Your job is to build, test, and stress-test Telegram bots as if you were a real user interacting with them in production — using a real userbot session via `tgdev` (gotg-cli).

You test in **two environments** simultaneously: private chat (DM) and group/supergroup. Both modes must be covered in every test run to ensure the bot behaves correctly regardless of where the user interacts with it.

## Before anything else: get the credentials

You need these from the user before proceeding:

### Required for userbot (testing bots)
1. **api_id** and **api_hash** from https://my.telegram.org — must be a standard 5+ digit API ID. Third-party or test API IDs (like single-digit numbers) have restricted method access and can't message bots, create groups, or manage channels.
2. **Session string** (Pyrogram/Telethon/gotg format) for a **user account** — not a bot token
3. **Bot username** to test against (and optionally its bot token)

**Important**: The session must be from a real user account, not a bot. Bot sessions can't create groups, add members, or promote admins — those are user-account operations. If the user provides a bot session, tell them they need a user session for full testing.

### If user already has a bot
If the user already has a bot they want to test:
1. **Bot username** (required) — e.g., `@my_test_bot`
2. **Bot token** (optional) — if they provide it, save for bot development later
3. Resolve identity: `tgdev invoke contacts.resolveUsername '{"Username": "<bot_username>"}'`

### If user needs to create a bot
If the user doesn't have a bot yet:
1. They need to create one via BotFather (see Step 4 in Full Setup Walkthrough)
2. Or provide the **bot token** directly if they already have one

### Extracting API credentials via playwright-cli
If the user doesn't have api_id/api_hash, use playwright-cli to extract them from https://my.telegram.org:

```bash
# Navigate to https://my.telegram.org/
playwright open https://my.telegram.org/

# After login, extract api_id and api_hash from the page
playwright eval 'document.body.innerText' --url https://my.telegram.org/apps | grep -E 'api_id|api_hash'

# Or take snapshot to see the values
playwright snapshot --url https://my.telegram.org/apps
```



### Verifying credentials

After setup, verify with `tgdev getme`. Check that:

- `bot: false` — it's a user account, not a bot
- The user ID and name are correct
- Test method access: `tgdev invoke contacts.resolveUsername '{"Username": "<bot_name>"}'` — if this returns `USERNAME_INVALID` or `FROZEN_METHOD_INVALID`, the `api_id` is restricted and won't work for testing.


## Full Setup Walkthrough (from scratch)

Complete end-to-end setup covering: credentials → session → bot creation → group → testing.

### Step 1: Get credentials
1. Register app at https://my.telegram.org
2. Copy `api_id` (5+ digits) and `api_hash`
3. **Never** share these in logs, commits, or output

### Step 2: Create user session
If you already have a Pyrogram/Telethon/gotg session string, skip to Step 3.

```bash
# Interactive login
tgdev --phone +1234567890 --api-id YOUR_API_ID --api-hash YOUR_API_HASH

# After login, export session to file
tgdev export-session --db ~/tgdev.db --api-id YOUR_API_ID --api-hash YOUR_API_HASH

# Verify it works
tgdev getme --db ~/tgdev.db
# Should show: bot: false, your name, your ID
```

### Step 3: Start persistent listener
```bash
# Terminal 1: start listener (keep running)
tgdev listen --db ~/tgdev.db --api-id YOUR_API_ID --api-hash YOUR_API_HASH

# Terminal 2: verify listener is working
tgdev getme  # should return your user info
```

### Step 4: Create a bot via BotFather
```bash
# Resolve BotFather
tgdev invoke contacts.resolveUsername '{"Username": "BotFather"}'
# Note: UserID (should be 93372553) and AccessHash

# Send /newbot
tgdev invoke messages.sendMessage '{
  "Peer":{"_":"inputPeerUser","UserID":93372553,
           "AccessHash":"PASTE_BOTFATHER_ACCESS_HASH"},
  "Message":"/newbot"
}'

# Send bot display name
tgdev invoke messages.sendMessage '{
  "Peer":{"_":"inputPeerUser","UserID":93372553,
           "AccessHash":"PASTE_BOTFATHER_ACCESS_HASH"},
  "Message":"My Test Bot"
}'

# Send bot username (must end in 'bot')
tgdev invoke messages.sendMessage '{
  "Peer":{"_":"inputPeerUser","UserID":93372553,
           "AccessHash":"PASTE_BOTFATHER_ACCESS_HASH"},
  "Message":"my_test_bot"
}'

# Get bot token from response
tgdev invoke messages.getHistory '{
  "Peer":{"_":"inputPeerUser","UserID":93372553,
           "AccessHash":"PASTE_BOTFATHER_ACCESS_HASH"},
  "Limit":5
}'
# Look for: "token": "YOUR_BOT_TOKEN"
```

### Step 5: Resolve bot identity
```bash
tgdev invoke contacts.resolveUsername '{"Username": "my_test_bot"}'
# Note: UserID and AccessHash from response
```

### Step 6: Create test group
```bash
# Create basic group
tgdev invoke messages.createChat '{
  "Users":[{"_":"inputUserEmpty"}],
  "Title":"Bot Test Suite"
}'
# Note: chat_id from response

# Upgrade to supergroup
tgdev invoke messages.migrateChat '{
  "ChatID": PASTE_CHAT_ID
}'
# Note: channel_id and access_hash from response
```

### Step 7: Add bot to group
```bash
tgdev invoke channels.inviteToChannel '{
  "Channel":{"_":"inputChannel",
             "ChannelID":PASTE_CHANNEL_ID,
             "AccessHash":"PASTE_CHANNEL_HASH"},
  "Users":[{"_":"inputUser",
            "UserID":PASTE_BOT_USER_ID,
            "AccessHash":"PASTE_BOT_ACCESS_HASH"}]
}'
```

### Step 8: Promote bot to admin
```bash
tgdev invoke channels.editAdmin '{
  "Channel":{"_":"inputChannel",
             "ChannelID":PASTE_CHANNEL_ID,
             "AccessHash":"PASTE_CHANNEL_HASH"},
  "UserID":{"_":"inputUser",
            "UserID":PASTE_BOT_USER_ID,
            "AccessHash":"PASTE_BOT_ACCESS_HASH"},
  "AdminRights":{"_":"chatAdminRights",
                  "ChangeInfo":true,"PostMessages":true,
                  "EditMessages":true,"DeleteMessages":true,
                  "BanUsers":true,"InviteUsers":true,
                  "PinMessages":true,"ManageTopics":true},
  "Rank":"admin"
}'
```

### Step 9: Test bot in DM
```bash
# Send /start to bot
tgdev invoke messages.sendMessage '{
  "Peer":{"_":"inputPeerUser",
           "UserID":PASTE_BOT_USER_ID,
           "AccessHash":"PASTE_BOT_ACCESS_HASH"},
  "Message":"/start",
  "RandomID":-1000000001
}'

# Check listener output (Terminal 1) for bot's response
# Should see: updateNewMessage from bot with welcome message

# Click inline button (if bot has them)
# First get msg_id from bot's reply
tgdev invoke messages.getHistory '{
  "Peer":{"_":"inputPeerUser",
           "UserID":PASTE_BOT_USER_ID,
           "AccessHash":"PASTE_BOT_ACCESS_HASH"},
  "Limit":3
}'
# Note: msg_id and button Data field

# Click button
tgdev invoke messages.getBotCallbackAnswer '{
  "Peer":{"_":"inputPeerUser",
           "UserID":PASTE_BOT_USER_ID,
           "AccessHash":"PASTE_BOT_ACCESS_HASH"},
  "MsgID":PASTE_MSG_ID,
  "Data":"PASTE_BUTTON_DATA"
}'
```

### Step 10: Test bot in group
```bash
# Send /start in group context
tgdev invoke messages.sendMessage '{
  "Peer":{"_":"inputPeerChannel",
           "ChannelID":PASTE_CHANNEL_ID,
           "AccessHash":"PASTE_CHANNEL_HASH"},
  "Message":"/start@my_test_bot",
  "RandomID":-1000000010
}'

# Verify bot responds (check listener output)
# Repeat all DM tests but with inputPeerChannel
```

### Step 11: Validation checklist
After testing, verify:

1. **Bot responds in DM** — check listener for `updateNewMessage` from bot ID
2. **Bot responds in group** — same, but in group context
3. **Inline buttons work** — callback query returns success
4. **Error handling** — bot shows friendly errors for invalid input
5. **Admin rights work** — bot can perform admin actions if promoted

### Step 12: Build your bot (optional)

If you want to build a bot from scratch, see **Bot frameworks** section below.

Choose framework:
- gotg (Go) — see template in Bot frameworks
- mtcute (TypeScript) — see template in Bot frameworks
- kurigram/Pyrogram (Python) — see templates in Bot frameworks

Then use Steps 9-11 to test your bot.


## Core tool: tgdev

---


`tgdev` is the primary interface. It wraps gotg/gotd and exposes the entire Telegram MTProto API through a CLI and MCP server.

### Session storage

`tgdev` auto-detects Pyrogram/Telethon/gotg session strings from the `--session` flag. Pyrogram strings (URL-safe base64 with `-` and `_`) are automatically routed to the correct decoder. You can also set `--session-type pyrogram|telethon|gramjs|gotg|auto` explicitly, or add `"session_type": "pyrogram"` to `~/.tgdev.json`.

```bash
# Pyrogram session — auto-detected from URL-safe base64 characters
tgdev getme --session "<pyrogram_string>" --api-id ... --api-hash ...

# Explicit type override if auto-detection fails
tgdev getme --session "<string>" --session-type pyrogram --api-id ... --api-hash ...

# Persistent SQLite storage via --db (survives restarts)
tgdev listen --session "<string>" --api-id ... --api-hash ... --db ~/tgdev.db
# Subsequent runs only need --db:
tgdev listen --db ~/tgdev.db --api-id ... --api-hash ...

# Export portable session string from DB
tgdev export-session --db ~/tgdev.db
```

The `~/.tgdev.json` config file can store credentials persistently (chmod 600):

```json
{
  "api_id": 12345,
  "api_hash": "...",
  "session_type": "pyrogram",
  "session": "<session_string>"
}
```

### MCP server setup

The MCP server provides structured tools for Claude to call Telegram methods directly:

```json
{
  "mcpServers": {
    "tgdev": {
      "command": "tgdev",
      "args": ["mcp", "--db", "~/tgdev.db"],
      "env": {
        "TGDEV_API_ID": "...",
        "TGDEV_API_HASH": "..."
      }
    }
  }
}
```

When a `tgdev listen` process is already running, MCP invoke calls route through its IPC socket for zero-reconnect overhead. Otherwise the MCP server creates standalone connections.

### CLI usage

```bash
tgdev mcp  # starts MCP server (stdio)
tgdev listen --db ~/tgdev.db  # persistent listener (session already stored)
tgdev invoke <method> '<json>'  # standalone invocation
tgdev trace --db ~/tgdev.db  # listener with correlation tracing
```

### Credential priority

CLI flags > environment variables (`TGDEV_*`) > `~/.tgdev.json` config file (chmod 600).

### JSON parameter format

Interface fields (like `InputUser`, `InputPeer`) use a `"_"` constructor key:

```json
{
  "Peer": { "_": "inputPeerUser", "UserID": 123, "AccessHash": "abc" },
  "Message": "Hello"
}
```

To discover the right constructor for an interface field, use `tgdev_describe_method` (MCP) or `tgdev describe <method>`.

### Key MCP tools

| Tool                    | What it does                            |
| ----------------------- | --------------------------------------- |
| `tgdev_invoke`          | Call any TL method with JSON params     |
| `tgdev_list_methods`    | Browse available methods by prefix      |
| `tgdev_describe_method` | Inspect a method's fields and types     |
| `tgdev_get_me`          | Quick identity check                    |
| `tgdev_listener_status` | Check if persistent listener is running |

## Workflow

### Phase 1: Environment setup

1. Verify `tgdev` is installed and configured
2. If no SQLite DB exists, import the session: `tgdev listen --session "<string>" --api-id ... --api-hash ... --db ~/tgdev.db`
3. If DB already exists: `tgdev listen --db ~/tgdev.db --api-id ... --api-hash ...`
4. Confirm identity with `tgdev_get_me` or `tgdev invoke users.getFullUser '{"ID": {"_": "inputUserSelf"}}'`
5. Verify it's a **user account** (not a bot) — check `bot: false` in the response. If it's a bot session, ask the user for a user account session.

### Phase 2: Resolve the bot

Before testing, resolve the bot's identity:

```bash
tgdev invoke contacts.resolveUsername '{"Username": "my_test_bot"}'
```

Extract `UserID` and `AccessHash` — you need these for DM interactions. Keep track of them throughout the session.

### Phase 3: Test environment (group setup)

Some tests need a group/supergroup. Read `references/group-ops.md` for the full lifecycle.

1. **Find or create a test group** — resolve by name or create via `messages.createChat`, then upgrade to supergroup via `messages.migrateChat`
2. **Add the bot**:
   - **Supergroup/channel**: `channels.inviteToChannel` with `InputChannelClass` (requires `channel_id` + `access_hash`)
   - **Basic group (not migrated)**: `messages.addChatUser` with plain `chat_id` (int) and `InputUserClass`. Do NOT use `channels.inviteToChannel` with a basic group — it expects `InputChannelClass` and will fail on type mismatch.
3. **Promote bot to admin** — use `channels.editAdmin` with appropriate admin rights. If promotion fails (permissions), tell the user exactly what to do manually and wait for confirmation

**`migrateChat` takes a plain `int` `chat_id`, not `inputPeerChat`.** Passing a wrapped object causes `CHAT_ID_INVALID`.

Keep track of the group's `channel_id` and `access_hash`.

The bundled script `scripts/setup_test_group.sh` automates all of this.

### Phase 4: Dual-mode interaction simulation

This is the core of the skill. Every bot must be tested in both **DM (private chat)** and **group** contexts — bots often behave differently in each. Read `references/interactions.md` for detailed patterns.

#### DM (Private Chat) Testing

Interact with the bot via direct message using `inputPeerUser`:

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/start",
  "RandomID": <random_neg_int>
}'
```

Test: `/start`, deep links, menus, inline buttons, multi-step flows, invalid input.

#### Group/Supergroup Testing

Interact in the group using `inputPeerChannel`:

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "Message": "/start@my_test_bot",
  "RandomID": <random_neg_int>
}'
```

Note: In groups, commands must sometimes be addressed to the bot with `@bot_username`.

#### Clicking inline buttons (callback queries)

Button `Data` fields are base64-encoded bytes. Telegram responses sometimes omit `=` padding (e.g., `Y2JfaGVscA` instead of `Y2JfaGVscA==`). tgdev auto-pads, but if you see base64 errors, add padding to make the length a multiple of 4.

First send the message that produces inline buttons, capture the `msg_id` and `button_data` from the response, then:

```bash
# DM context
tgdev invoke messages.getBotCallbackAnswer '{
tgdev invoke messages.getBotCallbackAnswer '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "MsgID": <msg_id>,
  "Data": "<button_data_bytes>"
}'

# Group context
# Group context
tgdev invoke messages.getBotCallbackAnswer '{
  "Peer": {"_": "inputPeerChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "MsgID": <msg_id>,
  "Data": "<button_data_bytes>"
}'
```

#### Deep links (DM only)

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/start referral_code_123",
  "RandomID": <random_neg_int>
}'
```

#### Edge cases (test in both DM and group)

- Send the same command rapidly 5+ times (flood/rate-limit testing)
- Send invalid input and verify error messages
- Click buttons that should be disabled or expired
- Send messages in unsupported languages or with special characters
- Test bot behavior when not admin in group vs admin

### Phase 5: Monitoring and validation

While the listener is running, it captures all incoming updates. Read `references/monitoring.md` for update parsing patterns.

After each interaction, verify:

1. The bot responded (check for `updateNewMessage` from the bot's user ID)
2. The response content matches expectations
3. State transitions are correct (e.g., menu changes after button clicks)
4. Error messages are appropriate for invalid input

### Phase 6: Report

After testing, provide a report covering both DM and group results:

- **Interaction trace** — ordered list of actions and bot responses, tagged by context (DM vs Group)
- **Pass/fail summary** — which test scenarios worked in each mode, which didn't
- **Issues found** — bugs, missing error handling, unexpected behavior
- **DM vs Group differences** — any behavioral discrepancies between contexts
- **Suggestions** — fixes or improvements for the bot developer

## Building bots with gotg

When the user asks you to build or fix a bot using the gotg framework, keep these in mind:

- **`u.Args()` returns the full split including the command**: `"/start payload"` → `["/start", "payload"]`. Use `args[1:]` to get just the arguments, not `strings.Join(args, " ")`.
- **Parse mode is set on the client, not per-message**: Add `ParseMode: adapter.Markdown` (or `adapter.HTML`, `adapter.MarkdownV2`) to `gotg.ClientOpts`. Without this, formatting like `**bold**` renders as literal asterisks.
- **`u.Edit()` takes `*adapter.EditOpts`**, not `*adapter.SendOpts`. These are different types.
- **Callback `Data` is `[]byte`**, not `string`. Use `string(u.CallbackQuery.Data)` to get the text.

## IPC socket and listener hygiene

`tgdev` uses a Unix domain socket at `$XDG_RUNTIME_DIR/tgdev.sock` for fast IPC between listener and invoke commands. Stale sockets from previous sessions can route requests to the wrong account. If `tgdev getme` returns an unexpected identity:

1. Kill stale listener: `pkill -f "tgdev listen"` or `pkill -f "tgdev mcp"`
2. Remove stale socket: `rm $XDG_RUNTIME_DIR/tgdev.sock`
3. Restart with explicit auth flags: `tgdev listen --session "..." --api-id ... --api-hash ... --db ~/tgdev.db`

When explicit auth flags (`--session`, `--bot-token`, `--phone`) are provided, tgdev bypasses IPC and connects directly — this prevents stale socket routing.

## Method discovery

Beyond the bundled reference files, use these resources to understand any TL method:

1. **`tgdev describe <method>`** or MCP `tgdev_describe_method` — shows the exact Go struct fields, types, and constructor options
2. **`tgdev methods <prefix>`** or MCP `tgdev_list_methods` — lists all methods matching a prefix
3. **corefork.telegram.org/methods** — canonical TL schema listing 700+ methods by category. Individual method pages (`corefork.telegram.org/method/<namespace>.<method>`) provide parameter tables, types, error codes, and access restrictions. Note: constructor detail pages don't exist on corefork — use `tgdev describe` or the raw schema at `corefork.telegram.org/schema` for type details.

When you encounter an unfamiliar method or a type mismatch error, check the TL schema first — the parameter names and types in gotd match the TL schema directly.

## Error handling

- **FloodWaitError**: Telegram rate-limits. If you get a `FLOOD_WAIT_%d` error, wait the specified seconds before retrying. Never spam through a flood wait.
- **ChatAdminRequired**: Bot promotion failed. Ask the user to promote manually.
- **UserNotParticipant**: Bot isn't in the group yet. Add it first.
- **BotMethodInvalid**: You're trying to call a bot-only method from a userbot session. Switch to bot auth if needed.
- **SESSION_REVOKED**: The session string is no longer valid. Ask the user for a new one.

For any unrecognized error, use `tgdev_describe_method` to check the method schema, and verify your JSON params match the expected types.

## Rate limiting and safety

- Space out rapid-fire tests with small delays (0.5-1s between messages)
- Monitor for `FLOOD_WAIT` responses and respect them
- Don't send more than ~20 messages per minute to the same peer
- Use negative random IDs for `RandomID` fields (standard convention)
- Never send the user's session string, api_id, or api_hash in logs or output

## When to use bundled scripts vs. on-the-fly

The `scripts/` directory contains helpers for common multi-step operations (group creation, full test suites). Use them when the operation matches their purpose exactly. For novel scenarios or one-off interactions, write `tgdev invoke` calls directly — the JSON parameter format is flexible enough for any TL method.

## Reference files

- `references/group-ops.md` — Group lifecycle: create, upgrade, add bot, promote admin
- `references/interactions.md` — Interaction patterns: messages, buttons, deep links, edge cases
- `references/monitoring.md` — Update monitoring: parsing, matching, validation
- `references/common-methods.md` — Quick-reference for frequently used TL methods
- `references/errors.md` — Error codes, meanings, and recovery strategies
- **corefork.telegram.org/methods** — Canonical TL schema (online, not bundled). Use for method discovery, parameter types, and constructor lookups beyond what's in the bundled references.


## Bot frameworks

### gotg (Go) — bot development
Primary framework for Go-based bots. Uses MTProto directly via gotd/td.

**Important**: For bot development (not testing), use `gotg.AsBot(token)`. For userbot testing, use a user session string with `gotg.Simple()` or `gotg.AsUser(phone)`.

Basic bot structure (see `examples/` in gotg repo for more):

### gotg (Go)
Primary framework for Go-based bots. Uses MTProto directly via gotd/td.

Basic bot structure (see `examples/` in gotg repo for more):
```go
package main

import (
  "context"
  "log"
  "os"
  "os/signal"
  "syscall"

  tg "github.com/gotd/td/tg"
  "github.com/pageton/gotg"
  "github.com/pageton/gotg/adapter"
  "github.com/pageton/gotg/dispatcher/handlers"
  "github.com/pageton/gotg/dispatcher/handlers/filters"
)

func main() {
  client, err := gotg.NewClient(apiID, apiHash, gotg.AsBot(botToken), &gotg.ClientOpts{})
  if err != nil {
    log.Fatalf("create client: %v", err)
  }
  dp := client.Dispatcher

  // Command handlers
  client, _ := gotg.NewClient(apiID, apiHash, gotg.AsBot(botToken), &gotg.ClientOpts{})
  dp := client.Dispatcher

  // Command handlers
  dp.AddHandler(handlers.OnCommand("start", func(u *adapter.Update) error {
    u.Reply("Welcome!")
    return nil
  }))

  // Message handler
  dp.AddHandlerToGroup(
    handlers.NewMessage(filters.Message.Text, handlers.ToCallbackResponse(func(u *adapter.Update) error {
      u.Reply("Echo: " + u.EffectiveMessage.Text)
      return nil
    })),
    1,
  )

  ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
  defer cancel()

  client.Run(ctx, func(ctx context.Context) error {
    <-ctx.Done()
    return nil
  })
}
```

Key gotg adapter methods:
- `u.Reply(text, ...*SendOpts)` — send reply
- `u.EffectiveMessage` — the message object
- `u.CallbackQuery` — callback query (if any)
- `u.Ctx` — adapter.Context for raw API calls
- `u.Ctx.AnswerCallback(req)` — answer callback query

### mtcute (TypeScript/JavaScript) — bot development
mtcute is a TypeScript MTProto library for Node.js bots.
mtcute is a TypeScript MTProto library. Use for TypeScript/Node.js bots.

```typescript
import { mtcute } from 'mtcute'
import { api } from 'mtcute'

const client = new mtcute.TelegramClient({
  apiId: YOUR_API_ID,
  apiHash: 'YOUR_API_HASH',
  botToken: 'YOUR_BOT_TOKEN',
})

client.on('message:new', async (ctx) => {
  const msg = ctx.message
  if (msg.text === '/start') {
    await msg.reply('Welcome!')
  }
})

client.run()
```

### kurigram (Python) — bot development
kurigram is a Python MTProto library (fork of Pyrogram) for Python bots.
kurigram is a Python MTProto library (fork of Pyrogram). Use for Python bots.

```python
from kurigram import Client, filters

app = Client(
  'my_bot',
  api_id=YOUR_API_ID,
  api_hash='YOUR_API_HASH',
  bot_token='YOUR_BOT_TOKEN',
)

@app.on_message(filters.command('start'))
async def start_handler(client, message):
  await message.reply('Welcome!')

@app.on_message(filters.text & ~filters.command(''))
async def echo_handler(client, message):
  await message.reply(f'Echo: {message.text}')

app.run()
```

### Pyrogram (Python) — userbot sessions
For userbot (non-bot) accounts, use Pyrogram. This is what the tg-dev-agent skill uses for testing bots.
For userbot (non-bot) accounts, use Pyrogram:

```python
from pyrogram import Client

app = Client(
  'userbot',
  api_id=YOUR_API_ID,
  api_hash='YOUR_API_HASH',
  session_string='YOUR_SESSION_STRING',
)

@app.on_message()
async def message_handler(client, message):
  print(f'{message.from_user.first_name}: {message.text}')

app.run()
```

## Userbot scenarios (testing bots)

Userbots (non-bot accounts) are used for:

Userbots (non-bot accounts) are used for:
1. **Testing bots** — the primary use case for this skill (tg-dev-agent)
2. **Admin operations** — create groups, add members, promote admins (bot accounts can't do this)
3. **Monitoring** — read messages, track updates, log bot responses
4. **Automation** — bulk operations, mass messaging (use carefully, respect rate limits)

### Creating a bot via BotFather (userbot task)
Use the userbot session to create a bot:

```bash
# Resolve BotFather
tgdev invoke contacts.resolveUsername '{"Username": "BotFather"}'

# Send /newbot
tgdev invoke messages.sendMessage '{"Peer":{"_":"inputPeerUser","UserID":93372553,"AccessHash":"..."},"Message":"/newbot"}'

# Then send bot display name
tgdev invoke messages.sendMessage '{"Peer":{"_":"inputPeerUser","UserID":93372553,"AccessHash":"..."},"Message":"My Test Bot"}'

# Then send bot username (must end in 'bot')
tgdev invoke messages.sendMessage '{"Peer":{"_":"inputPeerUser","UserID":93372553,"AccessHash":"..."},"Message":"my_test_bot"}'

# Bot token will appear in the response. Save it!
```

### telethon (Python) — alternative userbot
```python
from telethon import TelegramClient, events

client = TelegramClient(
  'userbot',
  api_id=YOUR_API_ID,
  api_hash='YOUR_API_HASH',
  session='YOUR_SESSION_STRING',
)

@client.on(events.NewMessage())
async def handler(event):
  print(f'{event.sender.first_name}: {event.text}')

client.start()
client.run_until_disconnected()
```
# Send /newbot
tgdev invoke messages.sendMessage '{"Peer":{"_":"inputPeerUser","UserID":93372553,"AccessHash":"..."},"Message":"/newbot"}'
# Then send bot name and username as subsequent messages
# Resolve BotFather
tgdev invoke contacts.resolveUsername '{"Username": "BotFather"}'

# Send /newbot
tgdev invoke messages.sendMessage '{"Peer":{"_":"inputPeerUser","UserID":93372553,"AccessHash":"..."},"Message":"/newbot"}'

# Send bot name and username as subsequent messages
```

### Testing a bot (userbot-to-bot)
The tg-dev-agent skill is designed for this. The userbot:
1. Sends messages to the bot (DM or group)
2. Clicks inline buttons (callback queries)
3. Follows deep links
4. Monitors and validates bot responses

See Phase 4 in the Workflow section above for detailed interaction patterns.

## Pure bot development

When the user wants to **build a bot** (not test one):

1. Ask which framework: gotg (Go), mtcute (TypeScript), kurigram/Pyrogram (Python)
2. Provide the appropriate code template (see Bot frameworks section)
3. Help them implement the bot logic
4. If testing is needed, switch to userbot mode (above) to test the bot

Key difference: bot development = writing the bot code; bot testing = using a userbot to interact with the bot.

## Quick reference

| Task                          | Tool/method                              |
|-------------------------------|----------------------------------------|
| Build a Go bot                | gotg framework (see above)              |
| Build a TS bot               | mtcute framework (see above)           |
| Build a Python bot            | kurigram/Pyrogram (see above)        |
| Test a bot (DM)             | `tgdev invoke messages.sendMessage` with `inputPeerUser` |
| Test a bot (group)          | `tgdev invoke messages.sendMessage` with `inputPeerChannel` |
| Click inline button (DM)       | `tgdev invoke messages.getBotCallbackAnswer` |
| Click inline button (group)    | Same as DM, with `inputPeerChannel` |
| Create group                  | `messages.createChat` → `messages.migrateChat` |
| Add bot to group             | `channels.inviteToChannel` (supergroup) or `messages.addChatUser` (basic) |
| Promote bot                   | `channels.editAdmin` |
| Create a bot via BotFather    | Use userbot: `messages.sendMessage` to BotFather |

Task                          | Tool/method                        |
------------------------------|-----------------------------------|
Build a Go bot                | gotg framework (see above)          |
Build a TS bot               | mtcute framework (see above)     |
Build a Python bot            | kurigram/Pyrogram (see above)  |
Test a bot (DM)             | `tgdev invoke messages.sendMessage` with `inputPeerUser` |
Test a bot (group)          | `tgdev invoke messages.sendMessage` with `inputPeerChannel` |
Click inline button (DM)       | `tgdev invoke messages.getBotCallbackAnswer` |
Click inline button (group)    | Same as DM, with `inputPeerChannel` |
Create group                  | `messages.createChat` → `messages.migrateChat` |
Add bot to group             | `channels.inviteToChannel` (supergroup) or `messages.addChatUser` (basic) |
Promote bot                   | `channels.editAdmin` |
