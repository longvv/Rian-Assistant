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
