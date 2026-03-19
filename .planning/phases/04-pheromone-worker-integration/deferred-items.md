# Deferred Items - Phase 04

## Pre-existing Issues

### lint:sync command count mismatch (Claude Code 38 vs OpenCode 37)
- **Discovered during:** 04-01 Task 2
- **Details:** `npm run lint:sync` fails with "Command counts don't match!" -- Claude Code has 38 commands, OpenCode has 37. This is a pre-existing issue unrelated to agent definition sync.
- **Impact:** lint:sync exits non-zero, but the agent mirror byte-identical requirement is verified independently via diff.
- **Recommendation:** Add the missing OpenCode command in a future chore task.
