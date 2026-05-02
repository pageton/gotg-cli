# Common TL Methods Quick Reference

## Authentication & Identity

| Method                     | Purpose             | Key params                   |
| -------------------------- | ------------------- | ---------------------------- |
| `users.getFullUser`        | Get user profile    | `ID: {"_": "inputUserSelf"}` |
| `contacts.resolveUsername` | Resolve @username   | `Username: "bot_name"`       |
| `account.updateProfile`    | Update profile info | `FirstName`, `About`, etc.   |

## Messaging

| Method                     | Purpose          | Key params                    |
| -------------------------- | ---------------- | ----------------------------- |
| `messages.sendMessage`     | Send text        | `Peer`, `Message`, `RandomID` |
| `messages.sendMedia`       | Send media       | `Peer`, `Media`, `RandomID`   |
| `messages.forwardMessages` | Forward msgs     | `FromPeer`, `ToPeer`, `ID`    |
| `messages.editMessage`     | Edit message     | `Peer`, `ID`, `Message`       |
| `messages.deleteMessages`  | Delete messages  | `Peer`, `ID`                  |
| `messages.getHistory`      | Get chat history | `Peer`, `Limit`, `OffsetID`   |
| `messages.getDialogs`      | List chats       | `Limit`                       |
| `messages.searchGlobal`    | Search all chats | `Q`, `Limit`                  |

## Bot Interactions

| Method                          | Purpose             | Key params              |
| ------------------------------- | ------------------- | ----------------------- |
| `messages.getBotCallbackAnswer` | Click inline button | `Peer`, `MsgID`, `Data` |
| `messages.getInlineBotResults`  | Query inline mode   | `Bot`, `Peer`, `Query`  |
| `messages.sendInlineBotResult`  | Send inline result  | `Peer`, `QueryID`, `ID` |

## Group/Channel Management

| Method                     | Purpose               | Key params                         |
| -------------------------- | --------------------- | ---------------------------------- |
| `messages.createChat`      | Create basic group    | `Users`, `Title`                   |
| `messages.migrateChat`     | Upgrade to supergroup | `ChatID`                           |
| `channels.createChannel`   | Create channel/sg     | `Title`, `About`, `Broadcast`      |
| `channels.getFullChannel`  | Get channel info      | `Channel`                          |
| `channels.inviteToChannel` | Add member            | `Channel`, `Users`                 |
| `channels.editAdmin`       | Promote/demote admin  | `Channel`, `UserID`, `AdminRights` |
| `channels.getParticipants` | List members          | `Channel`, `Filter`, `Limit`       |
| `channels.joinChannel`     | Join channel          | `Channel`                          |
| `channels.leaveChannel`    | Leave channel         | `Channel`                          |

## Chat Operations

| Method                    | Purpose           | Key params         |
| ------------------------- | ----------------- | ------------------ |
| `messages.getFullChat`    | Get chat details  | `ChatID`           |
| `messages.addChatUser`    | Add user to group | `ChatID`, `UserID` |
| `messages.deleteChatUser` | Remove user       | `ChatID`, `UserID` |
| `messages.editChatTitle`  | Rename group      | `ChatID`, `Title`  |

## Constructor hints for common interface fields

### InputPeer variants

```json
{"_": "inputPeerSelf"}                    // Own account
{"_": "inputPeerUser", "UserID": N, "AccessHash": "..."}  // DM with user
{"_": "inputPeerChat", "ChatID": N}       // Basic group
{"_": "inputPeerChannel", "ChannelID": N, "AccessHash": "..."}  // Supergroup
```

### InputUser variants

```json
{"_": "inputUserSelf"}                    // Own account
{"_": "inputUser", "UserID": N, "AccessHash": "..."}      // Specific user
```

### InputChannel

```json
{"_": "inputChannel", "ChannelID": N, "AccessHash": "..."}
{"_": "inputChannelEmpty"}
```

### ChatAdminRights (for promotion)

```json
{
  "_": "chatAdminRights",
  "ChangeInfo": true,
  "PostMessages": true,
  "EditMessages": true,
  "DeleteMessages": true,
  "BanUsers": true,
  "InviteUsers": true,
  "PinMessages": true,
  "ManageTopics": true,
  "AddAdmins": false
}
```

Set `AddAdmins: false` unless the bot needs to promote others. Grant minimum necessary rights.

## Method lookup resources

When you need a method not listed here, or need exact parameter types:

1. **`tgdev describe <method>`** — shows gotd Go struct with field names and types
2. **corefork.telegram.org/methods** — canonical TL schema listing all methods
3. **corefork.telegram.org/method/<namespace>.<method>** — specific method page with parameter table, types, error codes, and return type (e.g., `corefork.telegram.org/method/messages.sendMedia`)
4. **corefork.telegram.org/schema** — raw TL schema with all type/constructor definitions (for when you need field details beyond what `tgdev describe` shows)

gotd field names match the TL schema exactly (PascalCase), so a field called `access_hash` in TL becomes `AccessHash` in tgdev JSON.

## Useful discovery commands

```bash
# List all methods matching a prefix
tgdev methods messages.send

# Describe a method's full schema
tgdev describe channels.editAdmin

# Via MCP: list methods
tgdev_list_methods with prefix "messages"

# Via MCP: describe method
tgdev_describe_method with method "channels.editAdmin"
```
