# PicoClaw Reliability & Context Improvements

This session successfully addressed multiple critical reliability and context management flaws in PicoClaw, as prioritized in the initial codebase review.

The resulting agent is significantly more stable against Anthropic API 400 errors, handles rate limits gracefully, processes prompt injections safely, and uses far less token budget per iteration.

## 1. Context & Reliability Fixes (loop.go & session/manager.go)

### 🔴 Critical: Session Validation on Load & Expiry

Previously, a crash mid-iteration would leave a session file ending with an orphaned Assistant `tool_call`. This "dangling" tool call permanently corrupted the session because Anthropic 100% rejects prompts with un-resolved tool calls, creating an endless loop of HTTP 400 errors for that user.

**Fix Details:**

- `session/manager.go` now runs `sanitizeMessages` during boot. If it detects a dangling `tool_call`, it drops that message.
- A **7-day Session TTL** was added. The application automatically deletes sessions older than 7 days, preventing bounded disk growth on persistent Railway volumes.

### 🔴 Critical: Exponential Backoff on 429

Previously, rate limits (HTTP 429) failed immediately and threw the error to the user, disrupting conversations.

- `loop.go` now wraps `callLLM()` in a retry loop.
- It detects `rate_limit` or `429` in the error stream and automatically sleeps with exponential wait times (1s, 2s) before retrying.

### 🟡 High: Context Compression Bug Fix

`forceCompression` had a bug where it appended the compression notification to `history[0]`, assuming `history[0]` was the system prompt. However, PicoClaw injects system prompts elsewhere, meaning it corrupted the first user message instead.

- **Fix:** Stops appending the note directly to the history element.

---

## 2. Token Budget Optimizations (loop.go & config.go)

### 🟡 High: Tool Result Caps & CJK Estimates

Tool results (like `web_fetch`) could easily return 30,000+ characters, needlessly burning `$0.05` to `$0.10` per request or causing instant context overflow. Token estimation was also heavily skewed.

**Fix Details:**

- **Tool Cap:** Hard limits all tool results to 4000 characters before sending them back to the LLM context.
- **Token Heuristic:** The 2.5 char global average for token estimation was rewritten to detect ASCII vs Unicode. It now assigns ~4 chars/token for ASCII and 1.5 tokens/char for CJK blocks. This prevents premature summarizations for users communicating in Chinese or Vietnamese.
- **Targeted Expansion:** Refactored `config.go` so `os.ExpandEnv` is only used exactly on `ModelList` variables instead of the entire parsed file.

---

## 3. Security & Observability (agent/loop.go, health/metrics.go, heartbeat/service.go)

### 📈 Structured /metrics Endpoint

A new endpoint `http://0.0.0.0:18790/metrics` was added.

- Tracks `llm_calls_total`, `errors_total`, `tokens_estimated` and `tool_executions` per tool type.

### 🛡️ Prompt Injection Prevention

A pre-LLM filter was injected into `loop.go:processMessage()`.

- Drops payload matching `"ignore previous"`, `"new instructions:"`.

### ⏲️ Heartbeat Task Cooldowns

Tasks in `HEARTBEAT.md` run indefinitely. To stop them from firing off heavy tool usage endlessly while the user isn't watching, we injected `workspace/heartbeat-state.json` into the `HEARTBEAT.md` prompt context.

- The heartbeat agent now has the power to inspect timestamp cooldowns, skip tasks that shouldn't run yet, and update the file itself.
