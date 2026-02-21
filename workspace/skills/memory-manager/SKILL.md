---
name: memory-manager
description: "Manage the user's long-term memory in memory/MEMORY.md. Use this skill whenever the user asks you to remember a fact, preference, or architectural decision, or when you learn important persistent facts."
metadata: { "emoji": "🧠", "category": "core" }
---

# Memory Manager

Your workspace has a `memory/MEMORY.md` file. This file represents your long-term memory.

## When to Update Memory

1. When the user explicitly asks you to "remember X" or "save this for later".
2. When you discover a persistent user preference (e.g., "I prefer dark mode", "always use Python 3.10").
3. When you make a major architectural decision for a project.
4. When you learn critical system/environment details (e.g., specific file paths, database schemas).

## How to Manage Memory

Read `memory/MEMORY.md` first, then append or modify using the file-editing tools. Group facts logically under standard headers:

1. **User Preferences**: How the user likes to work (e.g., preferred languages, styles).
2. **System Environment**: Details about their OS, shell, and installed tools.
3. **Project Contexts**: Specific details about projects they're working on, including important file paths or architecture choices.
4. **Agent Instructions**: Custom rules or behavioral guidelines they've given you.

Keep the memory concise. Use bullet points and clear, definitive statements. Obsolete facts should be removed or updated rather than just appending new facts.
