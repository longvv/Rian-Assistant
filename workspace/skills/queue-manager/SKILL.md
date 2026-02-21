---
name: queue-manager
description: "Manage rate-limited API requests and delayed tasks by enqueueing them and executing them later in a detached tmux session. Use when encountering 429 Too Many Requests errors."
metadata:
  {
    "nanobot":
      {
        "emoji": "⏳",
        "category": "core",
        "requires": { "skills": ["tmux"], "bins": ["tmux"] },
      },
  }
---

# Queue Manager

Intercept rate limit errors (like HTTP 429) and push them to a background queue to be executed after the `Retry-After` delay. This prevents the main agent from blocking, crashing, or returning an error to the user.

## Usage

When you encounter a rate limit on an action that can be safely retried later (e.g. ClawdChat posts/comments, GitHub API calls):

1. **Calculate Delay**: Read the `retry_after_seconds` or `Retry-After` header from the API response.
2. **Enqueue**: Use the `scripts/enqueue.sh` script to schedule the command.
3. **Notify User**: You DO NOT need to halt. Instead, tell the user that the request hit a rate limit and was queued, and that they do not need to do anything further.

```bash
# Example: Enqueue a ClawdChat comment retry in 600 seconds (10 minutes)
# Keep the command in single quotes to pass it effectively to the queue system
./scripts/enqueue.sh 600 "curl -X POST https://clawdchat.ai/api/v1/comments ..."
```

## How It Works

The enqueue script will:

- Generate a unique task ID.
- Spawn a detached `tmux` session named `queue-<task_id>`.
- Sleep for the specified delay.
- Execute the command automatically.
- Output logs to `data/<task_id>.log` for verification.

## Checking Queue Status

You can view active queues to see what's pending and when it will execute.

```bash
./scripts/dashboard.sh
```
