# Tool Execution Guidelines

## General Principles

- **Read before Write**: ALWAYS inspect a file or directory with a viewing/listing tool before editing or creating it. Never hallucinate line numbers or structures.
- **Search over Guessing**: Use search commands (`grep_search` or equivalent) to locate functions, variables, or error strings rather than guessing which file they reside in.
- **Atomic Edits**: Keep changes isolated and cohesive. Ensure code compiles and passes linters before declaring work done.
- **Batch for Parallelism**: When multiple independent lookups are needed (file reads, web searches), issue them all in a single response turn so they execute concurrently. Never chain serial tool calls when parallel execution is possible.

---

## Tool-Specific Rules

### 1. Bash / Shell Execution

- Run commands in the correct working directory.
- For long-running processes, use async execution or background mode.
- Never run destructive commands (`rm -rf`, `docker system prune`, etc.) without explicit user confirmation.

### 1a. Shell Script Discipline (CRITICAL)

> **The runtime environment only has `sh` (busybox/Alpine). `bash` is NOT installed.**

- **Always use `#!/bin/sh`** as the shebang. Never `#!/bin/bash`.
- **Never use bash-specific syntax** — forbidden constructs include:
  - Here-strings: `cmd <<< "$var"` → use `echo "$var" | cmd` instead
  - Arrays: `arr=(a b c)` → use positional params or a newline-delimited string
  - `[[ ... ]]` conditions → use `[ ... ]` (POSIX test)
  - Process substitution: `<(cmd)` → use a temp file or pipe
- **Validate before running**: always run `sh -n <script>` after writing and confirm exit code is 0.
- **Verify writes**: after every `write_file`, immediately `read_file` to confirm the content is correct.
- **Don't loop on edit_file failures**: if `edit_file` fails with "old_text not found" twice, use `write_file` to rewrite the entire file.
- **RSS/XML feeds**: use `grep -o '<title>[^<]*</title>' | sed ...` to extract titles. Do NOT use JSON-style grep on XML — it produces no output.

### 2. Code Modification (Go)

- Provide precise diffs. Preserve consistent indentation and coding style.
- After any edit: `go build ./pkg/... ./cmd/...` then `go test ./pkg/... -timeout 60s`.
- **Concurrency rules**: never hold a mutex (`RLock`/`Lock`) across blocking I/O or channel sends — this causes deadlocks under back-pressure. Release the lock, then do the I/O.
- **Hot path discipline**: hoist stable computations (tool definitions, rendered prompts) out of hot loops. Prefer `strings.Builder` over repeated string concatenation.

### 3. Web Search

- Use search when dealing with unfamiliar API changes, library edge cases, or external dependencies.
- Cross-reference multiple results for critical architectural decisions.
- **SearXNG priority**: If `tools.web.searxng.enabled=true` in config, web_search uses SearXNG (Google + Bing + DuckDuckGo + Wikipedia multi-engine).

### 4. Web Fetch (`web_fetch`) — Discipline Rules

- **Jina AI Reader (default ON)**: `web_fetch` first tries `https://r.jina.ai/{url}` which returns clean markdown using Mozilla Readability. Preferred extractor — no JS issues.
- **Avoid JS-rendered sites for data.** Sites like `kitco.com`, `investing.com`, `tradingeconomics.com`, `coinmarketcap.com` return skeleton HTML with no useful data.
- **Only fetch JSON/API endpoints.** URLs with `/api/`, `.json`, `/v1/`, `/v2/`, `/spot`, `/ticker`, etc.
- **Bail after 2 failed domain attempts.** If a domain returns <300 chars or no relevant data, move on.
- **Results are cached** (10 min TTL): no need to re-fetch the same URL.
- **Check SOURCES.md first** for known-good free JSON APIs before scraping any website.
- **Financial/live data strategy:**
  1. Check `workspace/SOURCES.md` for a known-good API endpoint.
  2. Use `web_search` first — it returns summarised text including prices.
  3. If a live number is needed, try a known-good JSON API (e.g. `open.er-api.com`, `api.coinbase.com`).
  4. If no clean JSON returns within 2 attempts, present the `web_search` summary. Do **not** exceed these limits.

### 5. Calculator (`calculator`) — Use for All Math

- **Always use `calculator` for arithmetic** rather than mental math or shell commands.
- Supports: `+`, `-`, `*`, `/`, `%`, `^`, `()`, `sqrt`, `abs`, `sin`, `cos`, `tan`, `log`, `log2`, `log10`, `exp`, `floor`, `ceil`, `round`, `pi`, `e`, `phi`.
- Examples: `calculator("sqrt(144) + pi")`, `calculator("(1234 - 987) / 987 * 100")`
- Zero overhead — no shell required, no side effects.
