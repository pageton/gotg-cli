# Interaction Patterns Reference

## Table of Contents

1. [Sending messages](#sending-messages)
2. [Triggering commands](#triggering-commands)
3. [Clicking inline buttons](#clicking-inline-buttons)
4. [Deep links and start parameters](#deep-links-and-start-parameters)
5. [Inline mode queries](#inline-mode-queries)
6. [Edge case simulation](#edge-case-simulation)
7. [Multi-step flows](#multi-step-flows)

---

## Sending messages

All messages need a `RandomID` — use a negative random integer (standard convention to avoid collisions).

### Direct message to bot (private chat)

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "Hello bot",
  "RandomID": -1000000001
}'
```

### Message in group (where bot is present)

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "Message": "Hello everyone",
  "RandomID": -1000000002
}'
```

### Message with formatting

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "bold **text**",
  "ParseMode": {"_": "textParseModeMarkdown"},
  "RandomID": -1000000003
}'
```

### Reply to a specific message

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "Replying to you",
  "ReplyTo": {"_": "inputReplyToMessage", "ReplyToMsgID": <msg_id>},
  "RandomID": -1000000004
}'
```

## Triggering commands

Bot commands are just messages starting with `/`:

```bash
# Basic command
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/start",
  "RandomID": -1000000010
}'

# Command with args
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/settings language",
  "RandomID": -1000000011
}'
```

## Clicking inline buttons

This is the trickiest interaction. The flow is:

1. Send a message that triggers the bot to respond with inline buttons
2. Capture the bot's response message — extract `msg_id` and the `ReplyMarkup` (inline keyboard)
3. Build the callback query using the button's `data` field

### Step 1: Trigger button response

Send any message that causes the bot to display inline buttons. Monitor the listener output for the bot's response.

### Step 2: Extract button data

Read the bot's response message. Inline keyboard buttons have a `data` field (base64-encoded bytes):

```json
{
  "ReplyMarkup": {
    "_": "replyInlineMarkup",
    "Rows": [
      {
        "Buttons": [
          {
            "_": "keyboardButtonCallback",
            "Text": "Click Me",
            "Data": "<base64_data>"
          }
        ]
      }
    ]
  }
}
```

### Step 3: Click the button

tgdev invoke messages.getBotCallbackAnswer '{
tgdev invoke messages.getBotCallbackAnswer '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "MsgID": <msg_id>,
  "Data": "<base64_data_from_button>"
}'
```

The response contains the callback answer (alert text, message, etc.).

**Note on base64 padding**: The `Data` field must be standard base64-encoded. Button data from Telegram responses may lack `=` padding (e.g., `Y2JfaGVscA` instead of `Y2JfaGVscA==`). tgdev auto-pads base64 input, but if you get a base64 error, add `=` padding to make the length a multiple of 4.

### Clicking URL buttons

URL buttons don't need callback queries — they're just links. But you can verify the button exists and its URL is valid:

```json
{ "_": "keyboardButtonUrl", "Text": "Visit Site", "URL": "https://example.com" }
```

### Clicking in group context

When the bot's message with buttons is in a group:

```bash
tgdev invoke messages.getBotCallbackAnswer '{
  "Peer": {"_": "inputPeerChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "MsgID": <msg_id>,
  "Data": "<base64_data>"
}'
```

## Deep links and start parameters

Deep links use the `/start` command with a payload:

```bash
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/start referral_abc123",
  "RandomID": -1000000020
}'
```

The bot receives the `referral_abc123` part as the start parameter. Test with:

- Valid codes
- Invalid/expired codes
- Empty codes (`/start `)
- Very long strings
- Special characters and unicode
- URL-encoded payloads

## Inline mode queries

### Query inline results

```bash
tgdev invoke messages.getInlineBotResults '{
  "Bot": {"_": "inputUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Peer": {"_": "inputPeerSelf"},
  "Query": "search term",
  "Offset": ""
}'
```

### Send an inline result

```bash
tgdev invoke messages.sendInlineBotResult '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "RandomID": -1000000030,
  "QueryID": <query_id_from_results>,
  "ID": "<result_id_from_results>"
}'
```

## Edge case simulation

### Rapid fire (flood testing)

Send 5-10 messages quickly. Watch for FLOOD_WAIT responses:

```bash
for i in $(seq 1 5); do
  tgdev invoke messages.sendMessage "{
    \"Peer\": {\"_\": \"inputPeerUser\", \"UserID\": <bot_id>, \"AccessHash\": \"<hash>\"},
    \"Message\": \"rapid test $i\",
    \"RandomID\": -100000100$i
  }"
  sleep 0.3
done
```

### Invalid input

```bash
# Nonsensical command
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "/nonexistent_command_with_weird_args !!!@@@",
  "RandomID": -1000000040
}'

# Empty message (some bots handle this)
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "",
  "RandomID": -1000000041
}'
```

### Expired/stale interactions

Click a button from an old message (e.g., a navigation button from a previous session). Bots should handle stale callback data gracefully.

### Large payloads

```bash
# Very long message
tgdev invoke messages.sendMessage '{
  "Peer": {"_": "inputPeerUser", "UserID": <bot_id>, "AccessHash": "<hash>"},
  "Message": "<4000+ character string>",
  "RandomID": -1000000050
}'
```

Telegram's message limit is 4096 characters. Messages over this will fail at the API level.

## Multi-step flows

For bots with multi-step wizards (e.g., settings > language > select):

1. Send `/settings` — capture response with buttons
2. Click "Language" button — capture new response
3. Click specific language — capture confirmation
4. Verify state persisted by restarting flow

Track state at each step by examining the bot's response content and keyboard structure. Compare against expected flow.
