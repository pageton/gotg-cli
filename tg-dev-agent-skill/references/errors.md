# Error Handling Reference

## Common Telegram API Errors

### Rate Limiting

| Error                      | Meaning                  | Recovery                                                |
| -------------------------- | ------------------------ | ------------------------------------------------------- |
| `FLOOD_WAIT_%d`            | Must wait %d seconds     | Sleep the specified duration, then retry. Never ignore. |
| `FLOOD_TEST_PHONE_WAIT_%d` | Test-specific rate limit | Same as FLOOD_WAIT                                      |
| `SlowModeWait%X`           | Group slow mode active   | Wait the specified seconds (group setting, not global)  |

### Permissions

| Error                       | Meaning                                  | Recovery                                        |
| --------------------------- | ---------------------------------------- | ----------------------------------------------- |
| `CHAT_ADMIN_REQUIRED`       | Need admin rights for this action        | Ask user to promote the bot or userbot to admin |
| `CHAT_WRITE_FORBIDDEN`      | Can't send in this chat                  | Chat may be restricted or read-only             |
| `CHAT_SEND_MEDIA_FORBIDDEN` | Can't send media here                    | Check group permissions                         |
| `USER_NOT_PARTICIPANT`      | Not a member of the group                | Join/add to group first                         |
| `USER_PRIVACY_RESTRICTED`   | User blocked this action                 | Can't override, try alternative approach        |
| `BOT_METHOD_INVALID`        | Bot-only method called from user session | Switch to bot auth if needed                    |
| `USER_BANNED_IN_CHANNEL`    | User is banned from this channel         | Check ban status, ask user to resolve           |

### Session & Auth

| Error                   | Meaning                 | Recovery                                  |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `SESSION_REVOKED`       | Session no longer valid | Ask user for new session string           |
| `AUTH_KEY_UNREGISTERED` | Auth key not recognized | Session may be corrupted, re-authenticate |
| `AUTH_KEY_INVALID`      | Invalid auth key        | Same as above                             |
| `SESSION_EXPIRED`       | Session timed out       | Reconnect or create new session           |

### Input Errors

| Error                   | Meaning                              | Recovery                                          |
| ----------------------- | ------------------------------------ | ------------------------------------------------- |
| `PEER_ID_INVALID`       | Invalid peer ID or access hash       | Re-resolve the peer (access hashes can change)    |
| `MESSAGE_ID_INVALID`    | Message doesn't exist                | Message may have been deleted                     |
| `USERNAME_INVALID`      | Username doesn't exist               | Check spelling, bot may have been renamed         |
| `USERNAME_NOT_OCCUPIED` | Username is not taken                | Bot may have changed username                     |
| `CHANNEL_PRIVATE`       | Don't have access to this channel    | Not a member, or channel was deleted/made private |
| `CHAT_ID_INVALID`       | Chat ID doesn't match any known chat | Verify the ID, may need to resolve again          |

### Group-Specific

| Error                      | Meaning                         | Recovery                              |
| -------------------------- | ------------------------------- | ------------------------------------- |
| `CHANNELS_TOO_MUCH`        | Account is in too many channels | Leave unused channels first           |
| `USERS_TOO_MUCH`           | Too many users in group         | Telegram limit (200k for supergroups) |
| `USER_ALREADY_PARTICIPANT` | User is already in the group    | Not an error, continue                |
| `CHAT_NOT_MODIFIED`        | No changes were made            | Typically safe to ignore              |

## Recovery Strategies

### Re-resolve peers

Access hashes can change. If you get `PEER_ID_INVALID`, re-resolve:

```bash
tgdev invoke contacts.resolveUsername '{"Username": "bot_name"}'
```

### Reconnection

If the listener disconnects, restart it. The session persists — no need to re-authenticate:

```bash
# Kill old listener
kill $(pgrep -f "tgdev listen")

# Restart
tgdev listen --session "..." --api-id ... --api-hash ...
```

### Permission escalation

When the userbot lacks permissions for an operation:

1. Check if the userbot is admin in the group: `channels.getParticipant`
2. If not admin, ask the user to promote the userbot
3. If the bot can't be promoted, ask the user to promote manually

### Flood wait handling pattern

```bash
# Pseudocode for flood-safe interaction loop
send_message() {
  result=$(tgdev invoke messages.sendMessage "$params" 2>&1)
  if echo "$result" | grep -q "FLOOD_WAIT"; then
    wait_seconds=$(echo "$result" | grep -oP 'FLOOD_WAIT_\K\d+')
    echo "Flood wait: ${wait_seconds}s"
    sleep "$wait_seconds"
    send_message  # retry
  fi
}
```

## Error diagnosis checklist

When an operation fails:

1. Check if the session is valid (`tgdev invoke users.getFullUser`)
2. Check if the peer exists and is accessible
3. Check permissions (admin rights, membership)
4. Check rate limits (recent message volume)
5. Check parameter format (use `tgdev describe <method>` to verify)
6. Check for `"_"` constructor hints in nested objects
