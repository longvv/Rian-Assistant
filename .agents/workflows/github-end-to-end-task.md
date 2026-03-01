---
description: End-to-end workflow for handling a feature request or bug fix and submitting it via GitHub PR
---

# GitHub End-to-End Task Resolution Workflow

This workflow guides the AI agent through the complete process of receiving a task, implementing the solution, testing it, and submitting a Pull Request on GitHub. It heavily leverages the `github` and `ai-engineer` skills.

## Phase 1: Planning and Analysis

1. **Understand Request**: Read the user's request carefully. Identify the core problem or feature.
2. **Codebase Exploration**: Use file search tools to locate relevant files and understand the current architecture.
3. **Draft Plan**: Following `ai-engineer` principles, create or propose an implementation plan. Consider edge cases, performance, and architecture.
4. **Propose Plan**: If the change is significant, get user approval before writing code.

## Phase 2: Implementation

1. **Create Branch**: Create a new git branch for the task.
   ```bash
   git checkout -b feature/your-feature-name
   ```
   // turbo
2. **Write Code**: Modify the files to implement the plan. Write clean, maintainable, and well-structured code.
3. **Commit Changes**: Stage and commit the code using conventional commit messages.
   ```bash
   git add .
   git commit -m "feat: description of changes"
   ```

## Phase 3: Verification

1. **Local Testing**: Run local tests, builds, or linting (e.g., `npm test`, `make build`, etc.).
2. **Fix Issues**: If tests fail, trace the errors and fix them before proceeding.

## Phase 4: GitHub Integration

1. **Push Branch**: Push the new branch to the remote repository.
   ```bash
   git push -u origin HEAD
   ```
2. **Create PR**: Use the `github` skill to create a Pull Request.
   ```bash
   gh pr create --title "feat: Your Feature Name" --body "### Description\nDetailed description of the changes made.\n\n### Testing\nSteps taken to verify."
   ```
   // turbo
3. **Check CI/CD**: Verify that the GitHub checks pass.
   ```bash
   gh pr checks
   ```
   If checks fail, investigate the logs, fix the issues, and push new commits.

## Phase 5: Completion

1. **Notify User**: Provide the Pull Request link to the user for final review and approval.
