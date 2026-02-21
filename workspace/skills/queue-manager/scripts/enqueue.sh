#!/usr/bin/env bash

# enqueue.sh - Queues a command to be executed after a delay in a background tmux session.

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <delay_seconds> <command>"
  echo "Example: $0 600 'curl -X POST ...'"
  exit 1
fi

DELAY="$1"
COMMAND="$2"
TASK_ID="$(date +%s)-$RANDOM"
SESSION_NAME="queue-$TASK_ID"

# Get absolute path for logs
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
DATA_DIR="$(dirname "$SCRIPT_DIR")/data"
mkdir -p "$DATA_DIR"
LOG_FILE="$DATA_DIR/$TASK_ID.log"

echo "Task ID: $TASK_ID"
echo "Delay: $DELAY seconds"
echo "Command: $COMMAND"
echo "Log File: $LOG_FILE"

# Create a tmux session detached
tmux new-session -d -s "$SESSION_NAME"

# Send the sleep and execute commands to the tmux session
# We echo status updates to the log.
tmux send-keys -t "$SESSION_NAME" \
  "echo 'Task $TASK_ID started. Waiting $DELAY seconds...' > '$LOG_FILE'" Enter \
  "sleep $DELAY" Enter \
  "echo 'Executing command at \$(date)...' >> '$LOG_FILE'" Enter \
  "eval '$COMMAND' >> '$LOG_FILE' 2>&1" Enter \
  "echo 'Execution finished with exit code \$? at \$(date).' >> '$LOG_FILE'" Enter \
  "tmux kill-session -t '$SESSION_NAME'" Enter

echo "Successfully queued task $TASK_ID in background."
echo "You can monitor the countdown and output with:"
echo "cat $LOG_FILE"
