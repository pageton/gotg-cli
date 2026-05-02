# Group Operations Reference

## Table of Contents

1. [Resolve existing group](#resolve-existing-group)
2. [Create a basic group](#create-a-basic-group)
3. [Upgrade to supergroup](#upgrade-to-supergroup)
4. [Add bot to group](#add-bot-to-group)
5. [Promote bot to admin](#promote-bot-to-admin)
6. [Get group info](#get-group-info)
7. [Full setup script flow](#full-setup-script-flow)

---

## Resolve existing group

Check if a group exists by searching dialogs or resolving by title.

```bash
# List dialogs to find the group
tgdev invoke messages.getDialogs '{"Limit": 100}'

# Search for a specific group title
tgdev invoke messages.searchGlobal '{"Q": "test-group-name", "Limit": 10}'
```

Match the `title` field in the response against the expected group name. Extract `channel_id` (for supergroups) or `chat_id` (for basic groups) and `access_hash`.

## Create a basic group

```bash
tgdev invoke messages.createChat '{
  "Users": [{"_": "inputUserEmpty"}],
  "Title": "Bot Test Suite"
}'
```

Response contains `updates` with the new `Chat` object. Extract `chat_id` from the result.

## Upgrade to supergroup

Basic groups have limitations (no admin rights granular enough for bots). Upgrade immediately:

```bash
tgdev invoke messages.migrateChat '{
  "ChatID": <chat_id>
}'
```

Response contains `updates` with the new `Channel` object. Extract `channel_id` and `access_hash` — you'll use these for all subsequent channel operations.

**Important:** After migration, the `chat_id` is no longer valid. All future operations use the `channel_id` as `InputChannel`.

**Note:** `migrateChat` takes a plain `int` `chat_id`, not an `inputPeerChat` wrapper. Passing `{"_": "inputPeerChat", ...}` will fail with `CHAT_ID_INVALID`.

## Add bot to group

First resolve the bot's username to get its `UserID` and `AccessHash`:

```bash
tgdev invoke contacts.resolveUsername '{
  "Username": "my_test_bot"
}'
```

### For supergroups/channels

Use `channels.inviteToChannel` — requires `InputChannelClass` (channel_id + access_hash):

```bash
tgdev invoke channels.inviteToChannel '{
  "Channel": {"_": "inputChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "Users": [{"_": "inputUser", "UserID": <bot_user_id>, "AccessHash": "<bot_access_hash>"}]
}'
```

### For basic groups (not yet migrated)

Use `messages.addChatUser` — takes plain `chat_id` (int) and `InputUserClass`:

```bash
tgdev invoke messages.addChatUser '{
  "ChatID": <chat_id>,
  "UserID": {"_": "inputUser", "UserID": <bot_user_id>, "AccessHash": "<bot_access_hash>"},
  "FwdLimit": 0
}'
```

**Do NOT use `channels.inviteToChannel` with `inputPeerChat`** — it expects `InputChannelClass` and will panic on type mismatch. Basic groups use `messages.addChatUser`.

## Promote bot to admin

Admin rights use a bitmask. Common rights for testing bots:

```bash
tgdev invoke channels.editAdmin '{
  "Channel": {"_": "inputChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "UserID": {"_": "inputUser", "UserID": <bot_user_id>, "AccessHash": "<bot_access_hash>"},
  "AdminRights": {
    "_": "chatAdminRights",
    "ChangeInfo": true,
    "PostMessages": true,
    "EditMessages": true,
    "DeleteMessages": true,
    "BanUsers": true,
    "InviteUsers": true,
    "PinMessages": true,
    "ManageTopics": true
  },
  "Rank": "admin"
}'
```

If this returns `CHAT_ADMIN_REQUIRED` or similar, the userbot account doesn't have sufficient permissions. Tell the user:

> "I couldn't promote the bot to admin. Please open the group in Telegram, go to the bot's profile, tap 'Promote to Admin', and enable at least: Post Messages, Edit Messages, Delete Messages, Pin Messages. Then let me know and I'll continue."

## Get group info

```bash
# Full channel info
tgdev invoke channels.getFullChannel '{
  "Channel": {"_": "inputChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>}
}'

# List participants (to verify bot is present)
tgdev invoke channels.getParticipants '{
  "Channel": {"_": "inputChannel", "ChannelID": <channel_id>, "AccessHash": "<hash>"},
  "Filter": {"_": "channelParticipantsRecent"},
  "Limit": 100
}'
```

## Full setup script flow

The bundled script `scripts/setup_test_group.sh` automates this entire flow:

1. Check if group exists via `messages.searchGlobal`
2. If not, create via `messages.createChat`
3. Upgrade to supergroup via `messages.migrateChat`
4. Resolve bot username via `contacts.resolveUsername`
5. Add bot via `channels.inviteToChannel`
6. Promote bot via `channels.editAdmin`
7. Report final group info

Usage:

```bash
./scripts/setup_test_group.sh --group "Bot Test Suite" --bot "my_test_bot"
```

The script outputs the `channel_id` and `access_hash` for use in subsequent operations.
