---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 02
subsystem: infra
tags: [git, worktree, testing, ecosystem-neutrality, cobra]

# Dependency graph
requires: []
provides:
  - worktree-merge-back CLI command drives its test gate from resolveTestCommand() instead of a hard-coded `go test ./...` literal
  - D-03 refusal path: an undeterminable test command blocks the merge, creates a blocker, and returns before Step 5's destructive cleanup
  - Five non-Go / refusal-path fixture tests covering the ecosystem-neutral merge gate
affects: [187-04-lean-build-path-merge-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "resolveTestCommand() wiring: strings.TrimSpace + empty-string check before building exec.CommandContext, following verification_command_section.go's shape rather than checkTestsPass's pass-by-default shape"

key-files:
  created:
    - cmd/worktree_merge_gate_test.go
  modified:
    - cmd/worktree.go

key-decisions:
  - "Empty-command refusal branch placed before the BuildTimeout context is created, so no context is leaked, and it returns before Step 5's git worktree remove/branch -d cleanup — this is the load-bearing D-03 guarantee."
  - "Node-repo fixture test uses CLAUDE.md's 'npm test' literal plus a package.json with `\"test\": \"exit 0\"` rather than a synthetic shell command, because extractTestCommand() only recognizes the literal substrings 'go test', 'npm test', and 'cargo test' — an arbitrary command like `true` is invisible to the resolver."

patterns-established:
  - "Fail-then-pass discipline for a refusal-path test: temporarily disabled the empty-command branch (git diff not committed, backed up and restored), confirmed TestWorktreeMergeBackRefusesWhenTestCommandUnknown fails (panics on an empty testFields slice) without the guard, then restored the guard and re-verified all 11 merge-back tests pass."

requirements-completed: []

# Metrics
duration: 25min
completed: 2026-08-18
---

# Phase 187 Plan 02: Ecosystem-Neutral Merge Gate Summary

**worktree-merge-back now runs the project's own resolved test command (via resolveTestCommand()) instead of a hard-coded `go test ./...`, and refuses to merge — without touching the worktree or branch — when no test command can be determined.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-18T15:26:00Z (approx, worktree branch reset)
- **Completed:** 2026-08-18T15:51:33Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments
- `worktreeMergeBackCmd` in `cmd/worktree.go` now calls `resolveTestCommand()` and executes the resolved command via `strings.Fields` + `exec.CommandContext`, instead of the literal `"go", "test", "./..."` argv.
- Added the D-03 refusal branch: an empty resolver result creates a blocker (`Merge blocked: cannot determine how to test this project for <branch>`), reports a plain-language explanation via `outputError`, and returns before Step 5's auto-cleanup — so the branch and worktree survive.
- Updated the command's `Long` help text to describe "the project's test command, resolved from CLAUDE.md or the project's language" instead of literally naming `go test ./...`.
- Added `cmd/worktree_merge_gate_test.go` with five new tests: a Node-fixture merge success, an unresolvable-project refusal, a real-git preservation check (worktree dir, `git worktree list`, `git branch --list` all still present after refusal), a Go-repo regression guard, and a help-text ratchet.
- Manually demonstrated fail-then-pass on the refusal test: with the empty-command branch disabled, `TestWorktreeMergeBackRefusesWhenTestCommandUnknown` panics on an empty `testFields` slice (proving the guard is load-bearing, not just cosmetic); restored and re-verified all tests pass.

## Task Commits

Each task was committed atomically:

1. **Task 1: Drive the worktree-merge-back test gate from resolveTestCommand** - `3201e73f` (feat)
2. **Task 2: Non-Go fixture tests for the merge-back gate** - `eb1ce9f0` (test)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified
- `cmd/worktree.go` - `worktreeMergeBackCmd.RunE` now resolves the test command via `resolveTestCommand()`, refuses-and-preserves on empty result, and the `Long` help text no longer says `go test`.
- `cmd/worktree_merge_gate_test.go` - Five new tests covering the Node-repo success path, the unresolvable-project refusal, real-git preservation proof, the Go-repo regression guard, and the help-text ratchet.

## Decisions Made
- Kept the empty-command branch's `return nil` placed strictly before the `testCtx`/`testCancel` context construction, matching the plan's explicit "no context leaked" requirement.
- Did not consolidate this fix with the duplicate implementation in `cmd/codex_build_worktree.go` (`mergePhaseWorktrees`) — that is explicitly out of scope for this plan and owned by `187-04-PLAN.md`.
- Chose `CLAUDE.md` + `package.json` together for the Node fixture (rather than a bare `package.json` alone) so the test's assertion about "no mention of go" is exercised via the actual `npm test` invocation, keeping the fixture representative of a real non-Go project rather than only exercising the CLAUDE.md tier in isolation.

## Deviations from Plan

None - plan executed exactly as written. The plan's `<threat_model>` mitigations (T-187-06, T-187-07) are both directly asserted by the new tests (`TestWorktreeMergeBackRefusesWhenTestCommandUnknown` and `TestWorktreeMergeBackRefusalPreservesTheWorktree` respectively); T-187-05 and T-187-08 were accepted disposition and required no code change.

## Issues Encountered
- Initial Node-fixture test used a synthetic `` `true` `` command in CLAUDE.md, which `extractTestCommand()` does not recognize (it only scans for the literal substrings "go test", "npm test", "cargo test"). Switched to writing "npm test" in CLAUDE.md plus a `package.json` with `"test": "exit 0"`, confirmed `npm` is present in this environment, and the test passed. This is the one real place this plan's execution depended on an external tool (`npm`) being installed in the CI/dev environment; if that ever changes, the fixture would need adjusting to use a `CLAUDE.md`-only path with a directly-invokable binary instead.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `cmd/worktree.go`'s merge-back CLI is now ecosystem-neutral and crash-safe on the "unknown test command" boundary. `cmd/codex_build_worktree.go`'s `mergePhaseWorktrees` (the build-path duplicate, hard-coded at line 1124) is unchanged by this plan and is explicitly assigned to `187-04-PLAN.md`.
- No blockers for downstream plans. The two merge-gate implementations remain intentionally un-consolidated per CONTEXT.md's `<deferred>` section (candidate for Phase 190).

---
*Phase: 187-crash-safe-worktrees-ecosystem-neutrality*
*Completed: 2026-08-18*
