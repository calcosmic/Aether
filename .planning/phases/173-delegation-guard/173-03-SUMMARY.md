---
phase: 173-delegation-guard
plan: 03
subsystem: infra
tags: [go, cli, spawn-guard, fail-closed, cobra]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: phase context and decision record (D-19, D-22) from 173-CONTEXT.md
provides:
  - "spawn-can-spawn-swarm denies (can_spawn:false, non-zero exit) when COLONY_STATE.json or spawn-tree.txt cannot be read, instead of granting a full budget"
  - "spawn-can-spawn-swarm's live-spawn count reads through agent.SpawnTree.Parse() instead of a fourth hand-rolled pipe parser"
  - "documented residue: agent.SpawnTree.Parse() cannot itself distinguish an absent spawn-tree.txt from an unreadable one (swallows all read errors as nil), which plan 10's per-guard fault-axis table should account for"
affects: [173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fail-closed guard: deny with a named reason + non-zero exit on any input a guard cannot read, rather than assuming an empty/permissive default"
    - "Check file existence/type via os.Stat ahead of a library Parse() call when that Parse() cannot itself distinguish absent from unreadable"

key-files:
  created:
    - cmd/spawn_failclosed_test.go
  modified:
    - cmd/internal_cmds.go

key-decisions:
  - "spawn-can-spawn-swarm is fixed in place, not merged into spawn-can-spawn (RESEARCH.md Pitfall 5 -- they are separate, currently-divergent paths and spawn-can-spawn-swarm has no real caller today)"
  - "agent.SpawnTree.Parse() never returns a non-nil error (confirmed by reading pkg/agent/spawn_tree.go's parseFile()), so the deny-on-unreadable-tree branch is implemented via os.Stat existence/type checks ahead of Parse(), not via Parse()'s own error return"
  - "A missing spawn-tree.txt (fresh colony, no spawns yet) is explicitly NOT treated as an error -- only a directory at the path, or another non-not-found stat error, denies"

requirements-completed: [SPAWN-04]

# Metrics
duration: 30min
completed: 2026-08-12
---

# Phase 173 Plan 03: Delegation Guard - spawn-can-spawn-swarm fail-closed fix Summary

**Inverted the named D-19 defect: `spawn-can-spawn-swarm` now denies with a reason and a non-zero exit when it cannot read colony state or the spawn tree, instead of answering "yes, budget 5" on the exact inputs it failed to read.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 2 completed
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- `spawn-can-spawn-swarm`'s colony-state-load branch now denies (`can_spawn:false`, exit code 1, reason `state-unreadable`) instead of granting a full budget of 5
- `spawn-can-spawn-swarm`'s spawn-tree-read branch now denies (`can_spawn:false`, exit code 1, reason `tree-unreadable`) instead of silently counting zero spawns and reporting the budget as entirely free
- The live-spawn count now goes through `agent.SpawnTree.Parse()` and `agent.IsLiveSpawnStatus()`, replacing the fourth hand-rolled pipe parser (`splitLineFields`) this command used to run
- Three fault-injection tests exist and were red-proofed by hand: reverting the fix makes the two deny tests fail while the negative control (valid state + valid tree still allows) stays green

## Task Commits

1. **Task 1: Make spawn-can-spawn-swarm deny when it cannot see** - `a9ef09fc` (fix)
2. **Task 2: Fault-injection red-proofs for the swarm guard** - `0c9b9450` (test)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified

- `cmd/internal_cmds.go` - `spawnCanSpawnSwarmCmd`'s two fail-open branches inverted to fail-closed; live-spawn counting moved onto `agent.SpawnTree.Parse()`
- `cmd/spawn_failclosed_test.go` - `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableColonyState`, `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree`, `TestSpawnCanSpawnSwarmAllowsWhenStateAndTreeAreBothReadable`

## Decisions Made

- **`Parse()` cannot itself be made to error, so the tree-unreadable branch is implemented above it.** Reading `pkg/agent/spawn_tree.go`'s `parseFile()` shows it treats every `store.ReadFile` error (missing file, or a directory at the path, or a permission error) identically as "file doesn't exist -- return empty" and always returns a nil error; malformed pipe lines are silently skipped rather than surfaced as an error. This means no byte content exists that makes `Parse()` return a non-nil error. The plan anticipated this possibility explicitly ("confirm whether `Parse()` distinguishes... if it does not, treat the absent-file case explicitly"). The fix checks the tree path's existence and type via `os.Stat` before calling `Parse()`: a directory at the path, or any stat error other than "not found," denies; a genuinely missing file (fresh colony, no spawns recorded yet) is not an error and counts zero live spawns, so a new colony is not permanently locked out of spawning.
- **The fault-injection test for the unreadable tree uses a directory at the path**, since that is the one input that is genuinely rejected before `Parse()` is ever called (per the residue above, corrupting the file's bytes would not have made `Parse()` fail).
- **The negative-control test's fixture line is deliberately crafted** (a single space before the status field and nowhere else) so it satisfies both the new pipe-based parser and the old whitespace-token scanner being replaced. This was needed to perform the D-22 red-proof: without it, the negative control would also go red against the reverted code (because the old scanner's undercount bug, named separately as threat T-173-15, does not recognize "active" inside an unbroken pipe-joined line unless a task field happens to contain a space in exactly the wrong place) -- which would have made the red-proof ambiguous about which behavior was actually being tested.

## Deviations from Plan

None as auto-fixes — the plan's own text already anticipated the `Parse()` residue and instructed how to handle it ("if it does not [distinguish], treat the absent-file case explicitly... State which behaviour `Parse()` has in the plan summary"). That investigation and its resolution are documented above under Decisions Made rather than as a deviation, since the plan explicitly asked for this determination rather than assuming one outcome.

## Issues Encountered

**Full test suite runtime under machine contention.** With multiple parallel worktree agents building/testing concurrently on the same machine (load average ~22 on 10 cores), `go test ./... -count=1 -timeout 900s` took about 5.5 minutes wall-clock instead of its usual ~90s. It completed well inside the 900s budget with all 18 packages green; no code issue, just contention. Noted here since it stretched this plan's own duration.

## Bounded Residue (named, not implied as covered)

Per 173-CONTEXT.md's D-21, this residue is stated explicitly rather than glossed:

- **`agent.SpawnTree.Parse()` cannot distinguish "spawn-tree.txt is absent" from "spawn-tree.txt is unreadable"** — both collapse to a nil-error, empty-entries return inside `parseFile()`. This plan's fix works around it at the `cmd/internal_cmds.go` call site (existence/type check via `os.Stat` before calling `Parse()`), so the guard itself is fail-closed for the cases that can actually be constructed (a directory at the path, or another stat error). A corrupted-but-still-a-regular-file `spawn-tree.txt` (e.g. truncated mid-write, or garbled pipe fields) is NOT denied by this fix — `Parse()` will silently skip malformed lines and return whatever it can parse, which may undercount but will not deny. This matches 173-VALIDATION.md's Known Residue item 3 framing ("fail-closed is per-guard, not universal") and is a `Parse()`-level limitation, not something this plan's narrower scope (fix `spawn-can-spawn-swarm` in place) set out to change. Flagging for plan 10, whose per-guard fault-axis table should record this exact boundary.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `spawn-can-spawn-swarm`'s D-19 defect is closed and covered by named, red-proofed tests.
- `cmd/spawn_failclosed_test.go` and its three test names are deliberately NOT yet added to `.github/workflows/ci.yml`'s named wiring-gate `-run` filter, per this plan's explicit scope boundary — plan 10 registers this file in `wiringGateGuardFiles` and adds all three names in one consistent change, alongside the other guard files from plans 05, 06, 08, 09. Until then, these tests still run under the blanket `go test ./...` release-gate step, which was confirmed green in this plan's own full-suite run.
- No blockers for plan 10's cross-guard consolidation work.

## Self-Check: PASSED

- FOUND: `cmd/spawn_failclosed_test.go`
- FOUND: `.planning/phases/173-delegation-guard/173-03-SUMMARY.md`
- FOUND commit: `a9ef09fc` (Task 1)
- FOUND commit: `0c9b9450` (Task 2)
- FOUND commit: `0f6b469c` (this summary)
- `go build ./cmd/aether`, `go vet ./cmd` clean
- `go test ./cmd -run 'TestSpawnCanSpawnSwarm' -count=1 -v` — 3/3 green
- `go test ./... -count=1 -timeout 900s` — 18/18 packages green

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-12*
