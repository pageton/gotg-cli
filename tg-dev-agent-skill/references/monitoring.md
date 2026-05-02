# Update Monitoring Reference

## Table of Contents

1. [Starting the listener](#starting-the-listener)
2. [Update types](#update-types)
3. [Matching bot responses](#matching-bot-responses)
4. [Correlation with trace mode](#correlation-with-trace-mode)
5. [Validation patterns](#validation-patterns)
6. [Programmatic monitoring](#programmatic-monitoring)

---

## Starting the listener

The persistent listener maintains a Telegram connection and logs all incoming updates:

```bash
tgdev listen --session "..." --api-id ... --api-hash ...
```

For correlation tracing (links API calls to their responses):

```bash
tgdev trace --session "..." --api-id ... --api-hash ...
```

The listener runs in the foreground. All updates are logged to stderr with their type names. Run it in a separate terminal or background it.

When using MCP, `tgdev_listener_status` checks if the listener is running. The MCP `tgdev_invoke` tool routes through the listener's IPC socket if available, avoiding reconnection overhead.

## Update types

The listener captures these update types relevant to bot testing:

| Update Type                | When it fires                         | Key fields                    |
| -------------------------- | ------------------------------------- | ----------------------------- |
| `updateNewMessage`         | New message (from bot or user)        | `message` (text, peer, date)  |
| `updateNewChannelMessage`  | New message in channel/supergroup     | `message` (text, peer, post)  |
| `updateEditMessage`        | Message edited                        | `message` (updated content)   |
| `updateEditChannelMessage` | Channel message edited                | `message` (updated content)   |
| `updateNewCallbackQuery`   | Inline button pressed                 | `query_id`, `data`, `user_id` |
| `updateNewInlineQuery`     | Inline mode activated                 | `query`, `user_id`, `offset`  |
| `updateBotCallbackQuery`   | Bot callback (when using bot session) | `query_id`, `data`            |

## Matching bot responses

After sending a message to the bot, match its response by:

### By message order

The bot's response typically arrives as the next `updateNewMessage` from the bot's user ID after your message. Capture the timestamp of your sent message and look for the first response from the bot after that timestamp.

### By reply markup

If you expect the bot to respond with inline buttons, check for `ReplyMarkup` in the response:

```json
{
  "ReplyMarkup": {
    "_": "replyInlineMarkup",
    "Rows": [...]
  }
}
```

### By text content

Match specific text patterns in the response to verify correct behavior:

```
Expected: "Welcome! Choose an option:"
Actual response: <check message.Text matches>
```

### By edit

Some bots edit their messages (e.g., loading indicators). Watch for `updateEditMessage` from the bot's user ID with the same `msg_id`.

## Correlation with trace mode

`tgdev trace` adds correlation IDs linking:

1. Your API call → assigned correlation ID `[42]`
2. Telegram's response → same ID `[42]`
3. Triggered updates → same ID or referenced

This makes it easy to trace: "I sent `/start` (call [42]), got `updatesTooLong` (response [42]), then received `updateNewMessage` (update [43]) from the bot."

Use trace mode when debugging complex interaction flows where timing matters.

## Validation patterns

### Verify bot responded within timeout

After sending a message, check the listener output for a response from the bot's user ID within a reasonable window (5-10 seconds for most bots).

### Verify response content

Parse the bot's message text against expected values:

- Exact match for fixed responses (e.g., "Welcome to Bot!")
- Pattern match for dynamic responses (e.g., "Order #12345 confirmed")
- Keyboard structure match for button layouts

### Verify state transitions

For multi-step flows, track the conversation state:

1. Send `/start` → expect main menu keyboard
2. Click "Settings" → expect settings keyboard
3. Click "Language" → expect language list
4. Select language → expect confirmation + back to settings

At each step, verify the bot's keyboard matches the expected state.

### Verify error handling

Send invalid input and verify:

- Bot responds (doesn't silently fail)
- Error message is user-friendly
- Bot state is recoverable (user can navigate back)

## Programmatic monitoring

For automated test suites, use the MCP tools or write a Go program using gotg directly:

### MCP-based monitoring

The `tgdev_invoke` MCP tool can be used to fetch recent messages after interactions:

```bash
# Get recent messages in a chat
tgdev invoke messages.getHistory '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Limit": 10
}'
```

### Go-based monitoring with gotg

For complex monitoring, write a Go program using the gotg dispatcher:

```go
client := gotg.NewClient(gotg.Simple(), session.StringSession(s))
client.Dispatcher.AddHandler(handlers.OnMessage(filters.Message.Text, func(c *adapter.Context, u *adapter.Update) error {
    if u.EffectiveUser().ID == botID {
        // Process bot response
        fmt.Printf("Bot: %s\n", u.Text())
    }
    return nil
}))
```

See the gotg examples for full dispatcher patterns.
