# Periodic Tasks (Heartbeat)

As an AI Engineer, these tasks run periodically in the background. If you discover actionable insights, flag them appropriately or fix minor issues autonomously based on your confidence level.

## Quick Tasks (Immediate processing)

- **Review pending TODOs/FIXMEs**: Scan the active active branch for newly added `TODO` or `FIXME` comments in the codebase.
- **Check Test Status**: Verify if unit tests (e.g., `make test` or `go test ./...`) pass in the most recently modified modules.

## Long Tasks (Use spawn for async / deep analysis)

- **Codebase Dependency Scan**: Check `go.mod` (or equivalent package files) for outdated or vulnerable dependencies using standard CLI tools (if available).
- **Architectural Debt Review**: Search the current project for heavily duplicated code patterns or bloated functions that surpass 100 lines, suggesting refactor opportunities.
- **Resource Monitoring**: Track system metrics or application telemetry to identify memory leaks or performance bottlenecks in the currently running services.

# Periodic Tasks

- Check my emails for urgent updates and send me a brief summary via Telegram.
- Draft a daily status check-in template for the 15-member team.
- Keep a running countdown of the days left for the 2-month project deadline and alert me if any high-priority issues are stalled.
