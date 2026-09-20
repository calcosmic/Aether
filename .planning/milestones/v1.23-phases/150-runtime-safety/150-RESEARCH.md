---
phase: 150-runtime-safety
dependencies: [149-queen-execution-policy]
requirements: [RUNTIME-01, RUNTIME-02, RUNTIME-03, RUNTIME-04, RUNTIME-05]
---

# Phase 150 Research: Runtime Safety

## Domain Context

This phase addresses the most painful reliability gap in Aether: silent data loss when builds are interrupted. The system already has worker dispatch, spawn tracking, worktree isolation, and recovery scanning -- but the seams between these systems leak completed work.

## Key Sources

- `.planning/research/PITFALLS-RELIABILITY.md` -- documents 3 interrupt-related failure modes
- `cmd/codex_build.go` -- build dispatch and worktree allocation
- `cmd/codex_build_finalize.go` -- build completion and packet creation
- `cmd/codex_continue.go` -- continue flow including worktree merge-back
- `pkg/codex/worker.go` -- worker invocation and error handling
- `cmd/recover.go` / `cmd/recovery_classify.go` -- existing recovery scanner
- `cmd/status.go` -- status command with spawn tracking display

## Known Failure Modes

### Pitfall 3: Worker Artifact Loss on Interrupt
A build worker completes its task and produces files, but the build is interrupted before the build packet is finalized. The completed code changes exist on disk but are not recorded in colony state.

**Current gap:** `cmd/finalizer_completion_contract.go` validates completion file path but has no fallback for interrupted workers.

**Fix direction:** Add `build-reconcile` command that scans filesystem for unrecorded worker changes.

### Pitfall 5: Worktree Branch Orphaning
Build waves spawn parallel agents into git worktrees. Each agent commits to its own branch. When a build is interrupted or build-complete does not run merge-back, those branches accumulate with valuable code never merged.

**Current gap:** Merge-back is in `continue-advance` playbook only. It is also NON-BLOCKING.

**Fix direction:** Move merge-back to `build-complete` step. Make it blocking. Add pre-build orphan detection.

### Pitfall 7: Provider Error Misreporting
When a worker process fails due to provider API errors (auth, rate limit), the error message says "parse worker output: no JSON found" instead of the real cause.

**Current gap:** Error handling in worker invocation parses output before classifying the failure cause.

**Fix direction:** Classify provider errors (exit codes, stderr patterns) before attempting JSON parse.

## Decision Log

- **Decision 1:** Build-reconcile will be a Go CLI command, not a playbook step -- it needs filesystem access and git operations that are awkward in markdown.
- **Decision 2:** Worktree merge-back should be extracted into a reusable function called from both build-finalize and continue-advance.
- **Decision 3:** Provider error classification should happen in `pkg/codex/worker.go` before claims parsing.
- **Decision 4:** Status reconciliation report should reuse existing spawn tree + git diff checks, not create new tracking files.
