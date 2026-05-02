#!/usr/bin/env bash
# setup_test_group.sh — Create or find a test supergroup, add bot, promote to admin.
#
# Usage:
#   ./setup_test_group.sh --group "Bot Test Suite" --bot "my_test_bot"
#
# Requires tgdev in PATH and credentials configured (env or ~/.tgdev.json).
# Outputs channel_id and access_hash for subsequent operations.

set -euo pipefail

GROUP_NAME=""
BOT_USERNAME=""
SESSION="${TGDEV_SESSION:-}"
API_ID="${TGDEV_API_ID:-}"
API_HASH="${TGDEV_API_HASH:-}"

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --group)  GROUP_NAME="$2"; shift 2 ;;
    --bot)    BOT_USERNAME="$2"; shift 2 ;;
    --session) SESSION="$2"; shift 2 ;;
    --api-id) API_ID="$2"; shift 2 ;;
    --api-hash) API_HASH="$2"; shift 2 ;;
    *) echo "Unknown flag: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$GROUP_NAME" || -z "$BOT_USERNAME" ]]; then
  echo "Usage: $0 --group <name> --bot <username> [--session <str>] [--api-id <id>] [--api-hash <hash>]" >&2
  exit 1
fi

# Build tgdev invoke prefix
TGDEV="tgdev"
if [[ -n "$SESSION" ]]; then TGDEV="$TGDEV --session $SESSION"; fi
if [[ -n "$API_ID" ]]; then TGDEV="$TGDEV --api-id $API_ID"; fi
if [[ -n "$API_HASH" ]]; then TGDEV="$TGDEV --api-hash $API_HASH"; fi

invoke() {
  $TGDEV invoke "$1" "$2" 2>/dev/null
}

echo "=== Telegram Test Group Setup ==="
echo "Group: $GROUP_NAME"
echo "Bot: @$BOT_USERNAME"
echo

# Step 1: Resolve bot user
echo "[1/5] Resolving bot @$BOT_USERNAME..."
BOT_INFO=$(invoke contacts.resolveUsername "{\"Username\": \"$BOT_USERNAME\"}")
if [[ -z "$BOT_INFO" ]]; then
  echo "ERROR: Could not resolve bot @$BOT_USERNAME. Check the username." >&2
  exit 1
fi

BOT_ID=$(echo "$BOT_INFO" | jq -r '.User.ID // .users[0].id // .User.id // empty' 2>/dev/null || true)
BOT_HASH=$(echo "$BOT_INFO" | jq -r '.User.AccessHash // .users[0].access_hash // .User.access_hash // empty' 2>/dev/null || true)

if [[ -z "$BOT_ID" || -z "$BOT_HASH" ]]; then
  echo "ERROR: Could not extract bot ID/access_hash from response." >&2
  echo "Raw response: $BOT_INFO" >&2
  exit 1
fi
echo "  Bot resolved: ID=$BOT_ID"

# Step 2: Search for existing group
echo "[2/5] Searching for existing group '$GROUP_NAME'..."
SEARCH=$(invoke messages.searchGlobal "{\"Q\": \"$GROUP_NAME\", \"Limit\": 10}")
EXISTING=$(echo "$SEARCH" | jq -r ".chats[] | select(.title == \"$GROUP_NAME\") | .id" 2>/dev/null | head -1 || true)

CHANNEL_ID=""
ACCESS_HASH=""

if [[ -n "$EXISTING" ]]; then
  CHANNEL_ID="$EXISTING"
  ACCESS_HASH=$(echo "$SEARCH" | jq -r ".chats[] | select(.id == $CHANNEL_ID) | .access_hash" 2>/dev/null || true)
  echo "  Found existing group: channel_id=$CHANNEL_ID"
else
  # Step 3: Create group
  echo "[3/5] Creating group '$GROUP_NAME'..."
  CREATE=$(invoke messages.createChat "{\"Users\": [{\"_\": \"inputUserEmpty\"}], \"Title\": \"$GROUP_NAME\"}")
  CHAT_ID=$(echo "$CREATE" | jq -r '.Updates.Chats[0].ID // .updates[0].message.chat_id // empty' 2>/dev/null || true)

  if [[ -z "$CHAT_ID" ]]; then
    echo "ERROR: Could not create group." >&2
    echo "Raw response: $CREATE" >&2
    exit 1
  fi
  echo "  Created basic group: chat_id=$CHAT_ID"

  # Step 4: Upgrade to supergroup
  echo "[4/5] Upgrading to supergroup..."
  MIGRATE=$(invoke messages.migrateChat "{\"ChatID\": $CHAT_ID}")
  CHANNEL_ID=$(echo "$MIGRATE" | jq -r '.Updates.Chats[0].ID // .updates[0].message.channel_id // empty' 2>/dev/null || true)
  ACCESS_HASH=$(echo "$MIGRATE" | jq -r '.Updates.Chats[0].AccessHash // .chats[0].access_hash // empty' 2>/dev/null || true)

  if [[ -z "$CHANNEL_ID" ]]; then
    echo "ERROR: Could not upgrade to supergroup." >&2
    exit 1
  fi
  echo "  Upgraded to supergroup: channel_id=$CHANNEL_ID"
fi

# Step 5: Add bot to group
echo "[5/5] Adding bot to group..."
invoke "channels.inviteToChannel" "{
  \"Channel\": {\"_\": \"inputChannel\", \"ChannelID\": $CHANNEL_ID, \"AccessHash\": \"$ACCESS_HASH\"},
  \"Users\": [{\"_\": \"inputUser\", \"UserID\": $BOT_ID, \"AccessHash\": \"$BOT_HASH\"}]
}" > /dev/null 2>&1 || echo "  (Bot may already be in group)"

# Step 6: Promote bot to admin
echo "[bonus] Promoting bot to admin..."
PROMOTE_RESULT=$(invoke "channels.editAdmin" "{
  \"Channel\": {\"_\": \"inputChannel\", \"ChannelID\": $CHANNEL_ID, \"AccessHash\": \"$ACCESS_HASH\"},
  \"UserID\": {\"_\": \"inputUser\", \"UserID\": $BOT_ID, \"AccessHash\": \"$BOT_HASH\"},
  \"AdminRights\": {
    \"_\": \"chatAdminRights\",
    \"ChangeInfo\": true,
    \"PostMessages\": true,
    \"EditMessages\": true,
    \"DeleteMessages\": true,
    \"BanUsers\": true,
    \"InviteUsers\": true,
    \"PinMessages\": true,
    \"ManageTopics\": true
  },
  \"Rank\": \"admin\"
}" 2>&1) || true

if echo "$PROMOTE_RESULT" | grep -qi "admin\|forbidden\|error"; then
  echo
  echo "  WARNING: Could not auto-promote bot."
  echo "  Please promote @${BOT_USERNAME} to admin manually in the group."
  echo "  Grant: PostMessages, EditMessages, DeleteMessages, PinMessages."
fi

echo
echo "=== Setup Complete ==="
echo "channel_id: $CHANNEL_ID"
echo "access_hash: $ACCESS_HASH"
echo "bot_id: $BOT_ID"
echo "bot_access_hash: $BOT_HASH"
echo
echo "Use these values in subsequent tgdev invoke calls."
