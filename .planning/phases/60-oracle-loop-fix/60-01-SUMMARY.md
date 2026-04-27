---
phase: 60-oracle-loop-fix
plan: 01
subsystem: worker-dispatch
tags: [opencode, env-var, subprocess, acp, platform-dispatch]

# Dependency graph
requires: []
provides:
  - AETHER_OPENCODE_AGENT_URL env var constant in platform_dispatch.go
  - Env var injection into OpenCode worker subprocess via cmd.Env
  - Tests proving override behavior and backward compatibility
affects: [60-02, 60-03, 60-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [env-var-override-in-subprocess]

key-files:
  created:
    - pkg/codex/platform_dispatch_test.go
  modified:
    - pkg/codex/platform_dispatch.go

key-decisions:
  - "Used append(os.Environ(), ...) pattern so existing parent env is preserved when injecting the override"
  - "Left cmd.Env nil when env var absent to maintain backward compatible behavior (Go inherits parent env when nil)"
  - "Named constant envOpenCodeAgentURL following existing AETHER_ prefix convention"

patterns-established:
  - "Env var override pattern: read from os.Getenv, conditionally append to os.Environ() slice"

requirements-completed: [ORCL-01]

# Metrics
duration: 2min
completed: 2026-04-27
---

# Phase 60 Plan 01: OpenCode Agent URL Env Var Injection Summary

**AETHER_OPENCODE_AGENT_URL env var injection into worker subprocess, separating the ACP messaging endpoint from LiteLLM proxy URLs**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-27T11:46:22Z
- **Completed:** 2026-04-27T11:48:50Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Added `envOpenCodeAgentURL = "AETHER_OPENCODE_AGENT_URL"` constant to the env var block
- Injected env var into `cmd.Env` in `invokeHostedWorker()` when set in parent process
- Backward compatible: `cmd.Env` stays nil when env var absent (Go inherits parent env)
- Two tests covering override pass-through and no-op when unset

## Task Commits

Each task was committed atomically:

1. **Task 1: Add AETHER_OPENCODE_AGENT_URL env var and inject into subprocess** - `c246e4d5` (test) -> `1dd7f2fa` (feat)

_Note: TDD task with RED/GREEN commits_

## Files Created/Modified
- `pkg/codex/platform_dispatch.go` - Added envOpenCodeAgentURL constant and env var injection in invokeHostedWorker
- `pkg/codex/platform_dispatch_test.go` - Created with two tests for env var override behavior

## Decisions Made
- Used `append(os.Environ(), envOpenCodeAgentURL+"="+agentURL)` to preserve all parent env vars while appending the override (last occurrence wins in most systems)
- Left `cmd.Env` as nil when env var is absent -- Go's `exec.Cmd` inherits parent environment when `Env` is nil, ensuring zero behavior change for existing users
- Followed existing naming convention (`AETHER_` prefix) for the new constant

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Test for "no override" case initially checked for the env var name in subprocess output, but `t.Setenv(var, "")` still makes the var present in `os.Environ()`. Fixed the assertion to check that the value is empty rather than that the var is absent.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- ORCL-01 requirement satisfied
- Phase 60-02 (research brief formulation) can proceed independently
- The env var is now available for the OpenCode dispatcher to pass the correct ACP endpoint to worker subprocesses

---
*Phase: 60-oracle-loop-fix*
*Completed: 2026-04-27*
