---
name: planner
description: "Manage project roadmaps and tasks in ROADMAP.md, TODO.md, or project specific Markdown trackers. Use this skill when asked to plan a project, break down a feature, or track task statuses."
metadata: { "emoji": "📅", "category": "core" }
---

# Planner

When managing complex tasks, starting a new project, or following an extensive checklist, use a Markdown file (e.g., `ROADMAP.md` or `TODO.md`) in the workspace to keep track of tasks systematically.

## Status Indicators

Use literal checkbox lists to track task status. This allows the user and you to instantly see work progression:

- `[ ]` Not started
- `[/]` In progress
- `[x]` Completed
- `[-]` Skipped or cancelled

_Indent sub-tasks under a parent task to establish hierarchy._

## Structure

Divide tasks into logical phases:

1. **Phase 1: Planning and Architecture** (Requirements, schema design, package selection)
2. **Phase 2: Implementation** (Core logic, UI components, integrations)
3. **Phase 3: Verification and Deployment** (Testing, CI/CD, documentation)

## Best Practices

1. **Break it down**: Keep individual items small and actionable (should take no more than a few tool calls to complete).
2. **Keep it updated**: Update the roadmap natively using file-editing tools as you progress, rather than only at the end.
3. **Be specific**: Mention specific file names and function names in the tasks so context isn't lost.
