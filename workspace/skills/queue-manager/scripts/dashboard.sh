#!/usr/bin/env bash

# dashboard.sh - Lists all currently running queue tasks in tmux.

echo "=== Rate Limit Queue Dashboard ==="

# Check if there are any active queue sessions
SESSIONS=$(tmux list-sessions 2>/dev/null | grep "^queue-")

if [ -z "$SESSIONS" ]; then
  echo "No pending tasks in the queue."
  exit 0
fi

echo "$SESSIONS" | while read -r line; do
  SESSION_NAME=$(echo "$line" | cut -d: -f1)
  TASK_ID=${SESSION_NAME#queue-}
  
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
  DATA_DIR="$(dirname "$SCRIPT_DIR")/data"
  LOG_FILE="$DATA_DIR/$TASK_ID.log"
  
  echo "-----------------------------------"
  echo "Session: $SESSION_NAME"
  if [ -f "$LOG_FILE" ]; then
    echo "Last Log Status:"
    tail -n 3 "$LOG_FILE" | sed 's/^/  | /'
  else
    echo "  | (No log output yet)"
  fi
done
echo "-----------------------------------"
