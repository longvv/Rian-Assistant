# Long-term Memory

This file stores important project and system context that should persist across sessions. (Note: User preferences live in `USER.md` and behavioral rules live in `AGENT.md`/`IDENTITY.md`).

## 1. System Environment

- **Project Location**: `/Users/rian.vu/Documents/picoclaw`
- **Primary Configuration**: `./config/config.json`
- **Deployment Platform**: **Railway**.
  - _Critical context_: Railway containers are ephemeral. Local files (like `data/` logs) and local background processes (like `tmux`) are destroyed on restart/redeploy.

## 2. Project Context (picoclaw)

- **Tech Stack**: Go (`pkg/` directory) backend utilizing an AI agent "skills" architecture.
- **Skills System**: Code located in `workspace/skills/`.
  - **Core custom skills**: `queue-manager`, `clawdchat`, `github`, `tmux`, `ai-engineer`, `memory-manager`
  - **Architecture pattern**: Skills use `SKILL.md` for definitions, and `scripts/` or `references/` for executable/static payloads.
