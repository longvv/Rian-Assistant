# Tool Execution Guidelines

## General Principles

- **Read before Write**: ALWAYS use file viewing or listing tools to inspect a file or directory before attempting to edit or create it. Never hallucinate line numbers or structures.
- **Search over Guessing**: Use search commands (`grep_search` or equivalent) to locate functions, variables, or error strings rather than guessing which file they reside in.
- **Atomic Edits**: When making file edits, try to keep changes isolated and cohesive. Ensure code compiles and passes linters locally, explicitly if instructed.

## Tool-Specific Rules

### 1. Bash / Shell Execution

- Run commands strictly in the correct working directory.
- For long-running process, use async tool calls or send them to the background if supported.
- Never blindly run potentially destructive commands (`rm -rf`, `docker system prune`, etc.) without rigorous checks and explicit user confirmation if deemed high-risk.

### 2. Code Modification

- Provide precise diffs.
- Ensure that replacing text does not accidentally modify identical strings elsewhere in the file unless intentionally performing a global replace.
- Preserve consistent indentation (tabs vs. spaces) and coding styles as found in the target file.

### 3. Web Search (Optional)

- Engage search functions when dealing with unfamiliar API changes, standard library edge cases, or external dependencies where knowledge might be outdated.
- Always cross-reference multiple results when making critical architectural decisions based on search.

### 4. Web Fetch (`web_fetch`) — Discipline Rules

- **Avoid JS-rendered sites for data.** Sites like `kitco.com`, `investing.com`, `tradingeconomics.com`, `coinmarketcap.com`, and similar financial dashboards render their data client-side via JavaScript. A plain HTTP fetch will return a skeleton HTML shell with no useful numeric data. Do NOT repeatedly fetch the same JS-rendered domain hoping for different results.
- **Only fetch JSON/API endpoints.** Use `web_fetch` on URLs that look like REST/JSON endpoints: they contain `/api/`, `.json`, `/v1/`, `/v2/`, `/spot`, `/ticker`, or similar patterns. If unsure, use `web_search` instead.
- **Bail after 2 failed domain attempts.** If `web_fetch` on a domain returns a result that is either very short (< 300 chars) or contains no numeric data relevant to the query, do NOT try other paths on that same domain. Mark that domain as "unhelpful" and move on.
- **Financial/live data strategy:**
  1. Use `web_search` first — it returns summarised text including prices.
  2. If a live number is needed, try a known-good JSON API (e.g. `https://open.er-api.com`, `https://api.coinbase.com/v2/prices/BTC-USD/spot`).
  3. If no JSON API returns clean data within 2 attempts, present the `web_search` summary with a note that values may be slightly delayed. Do **not** exceed these limits chasing a perfect source.
