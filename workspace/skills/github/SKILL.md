---
name: github
description: "Interact with GitHub using the `gh` CLI. Use `gh repo`, `gh issue`, `gh pr`, `gh run`, and `gh api` for code search, PRs, issues, CI runs, and reviews."
metadata:
  {
    "nanobot":
      {
        "emoji": "🐙",
        "requires": { "bins": ["gh"] },
        "install":
          [
            {
              "id": "brew",
              "kind": "brew",
              "formula": "gh",
              "bins": ["gh"],
              "label": "Install GitHub CLI (brew)",
            },
            {
              "id": "apt",
              "kind": "apt",
              "package": "gh",
              "bins": ["gh"],
              "label": "Install GitHub CLI (apt)",
            },
          ],
      },
  }
---

# GitHub Skill

Use the `gh` CLI to interact with GitHub. Always specify `--repo owner/repo` when not in a git directory, or use URLs directly.

**Rate Limits:** If you encounter an API rate limit or secondary rate limit (e.g., `HTTP 403/429` with `retry-after`), **DO NOT FAIL**. Use the `queue-manager` skill (`scripts/enqueue.sh`) to queue your command and notify the user it was queued.

**Missing CLI (`gh: not found`):** If the `gh` CLI isn't installed in the environment, fallback to using the `web_fetch` or `exec` tool targeting the GitHub REST API (e.g., `curl -s "https://api.github.com/repos/owner/repo/issues?state=open&per_page=5"`). **Do NOT fetch generic URLs that return HTML payloads. Always specify JSON API endpoints (`api.github.com`).**

## Exploring Repositories

View repository details:

```bash
gh repo view owner/repo
```

Search code across a repository:

```bash
gh search code "query" --repo owner/repo --limit 10
```

## Issues

List open issues:

```bash
gh issue list --repo owner/repo --limit 10 --state open
```

Create an issue:

```bash
gh issue create --repo owner/repo --title "Bug: Description" --body "Steps to reproduce..."
```

## Pull Requests

List open PRs:

```bash
gh pr list --repo owner/repo --limit 5
```

Create a PR:

```bash
gh pr create --repo owner/repo --title "Feature: Something" --body "Description of changes"
```

Review a PR (checkout locally):

```bash
gh pr checkout 55
```

Approve a PR:

```bash
gh pr review 55 --repo owner/repo --approve -b "LGTM!"
```

## GitHub Actions & CI

Check CI status on a PR:

```bash
gh pr checks 55 --repo owner/repo
```

List recent workflow runs:

```bash
gh run list --repo owner/repo --limit 10
```

View a run and see which steps failed:

```bash
gh run view <run-id> --repo owner/repo --log-failed
```

## API for Advanced Queries

The `gh api` command is useful for accessing data not available through other subcommands.

Get PR with specific fields:

```bash
gh api repos/owner/repo/pulls/55 --jq '.title, .state, .user.login'
```

## JSON Output

Most commands support `--json` for structured output. You can use `--jq` to filter:

```bash
gh issue list --repo owner/repo --json number,title --jq '.[] | "\(.number): \(.title)"'
```
