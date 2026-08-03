---
phase: 165-core-lifecycle-commands
plan: 07
subsystem: cli
tags: [go, cobra, colony-state, session, wrapper-markdown, tdd]

# Dependency graph
requires:
  - phase: 165-core-lifecycle-commands
    provides: init.md ceremony contract, wrapper-runtime host-manifest flow, lifecycle wrapper contract tests
provides:
  - promotedShelfTodos/mergeShelfTodos runtime wiring so a promoted shelf entry becomes a durable colony todo
  - shelf-promote-batch JSON result now includes a todos array
  - init.md ceremony fence for the active_todos hand-append regression class
  - init.md Approval reordered so pheromone-write is gated on aether init success
affects: [165-core-lifecycle-commands, future shelf/backlog work, init.md ceremony tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "merge-not-overwrite for session.ActiveTodos: shelf-prefixed entries preserved across session refresh via mergeShelfTodos"
    - "regression class fenced by forbidden-string test entries in TestInitWrapperCeremonyContract, proven RED before the fix"

key-files:
  created:
    - cmd/shelf_todo_wiring_test.go
  modified:
    - cmd/shelf_init.go
    - cmd/init_cmd.go
    - cmd/recovery_snapshot.go
    - cmd/session_cmds.go
    - cmd/init_wrapper_ceremony_test.go
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md
    - .claude/commands/ant-init.md
    - .aether/commands/init.yaml

key-decisions:
  - "Routed the shelf-to-todo promise through the runtime instead of deleting it — shelf-promote-batch now returns a todos array, aether init seeds session.ActiveTodos from it, and session refresh merges rather than overwrites."
  - "Fenced the regression class with forbidden-string test entries (active_todos, the append phrase, the write-target phrase) before removing them from the wrappers, so the test is provably not decorative (RED before, GREEN after)."
  - "Moved the aether pheromone-write loop in init.md's Approval stage to after the aether init call and gated it on success, closing WR-05."

patterns-established:
  - "Merge-not-overwrite for session.json fields that can be seeded before colony state exists and later recomputed from colony state."

requirements-completed: [CMD-01]

# Metrics
duration: ~25min
completed: 2026-08-03
---

# Phase 165 Plan 07: Shelf-to-Todo Runtime Wiring and init.md Ceremony Fence Summary

**Closed the Phase 165 CR-01 blocker by routing shelf-promote-batch's promised "become todos" behavior through the Go runtime instead of a hand-write instruction, and closed review WR-05 by gating init.md's pheromone writes on `aether init` success.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3 completed
- **Files modified:** 9 (1 created, 8 modified)

## Accomplishments

- `shelfEntryToTodo` has a real runtime call site for the first time — `shelf-promote-batch`'s JSON result now carries a `todos` array, and `aether init` seeds `session.ActiveTodos` from shelf entries promoted to that colony goal
- `mergeShelfTodos` stops the two existing session-refresh code paths (`recovery_snapshot.go`, `session_cmds.go`) from silently erasing shelf-seeded todos the next time a session refresh runs
- `init.md`'s `## Shelf Backlog` stage no longer instructs a hand-append to `active_todos` in the session file or colony state — the exact D-2 Frankenstein-state class this project has hit before — on all three wrapper surfaces plus the YAML source
- `init.md`'s `## Approval` stage now runs `aether init --colony-mode` before `aether pheromone-write`, so a failed or cancelled init writes nothing, matching the wrapper's own `<failure_modes>` promise
- `TestInitWrapperCeremonyContract` gained a forbidden-string fence for the old phrasing and a new ordering subtest, both proven RED against the unmodified wrappers before Task 3 turned them GREEN

## Task Commits

Each task was committed atomically (TDD tasks produced test-then-implementation commit pairs):

1. **Task 1: Wire shelfEntryToTodo into the runtime** — `dde2cde1` (test, RED) + `5708d394` (feat, GREEN)
2. **Task 2: Fence the hand-write phrasing and pheromone-ordering hazard** — `652b1b2e` (test, intentionally RED — Task 3 turns it GREEN)
3. **Task 3: Route init.md's shelf promotion through the runtime, reorder Approval, update init.yaml** — `ac49c5ad` (feat, GREEN)

_TDD tasks produced test → feat commit pairs; no refactor commit was needed._

## Files Created/Modified

- `cmd/shelf_init.go` — added `shelfTodoPrefix` constant, `promotedShelfTodos`, `mergeShelfTodos`; `shelf-promote-batch` RunE now returns a `todos` array
- `cmd/init_cmd.go` — `aether init` seeds `session.ActiveTodos` via `promotedShelfTodos(store, goal)` instead of a hard-coded empty slice
- `cmd/recovery_snapshot.go` — `syncSessionFromState` merges shelf todos into the phase-derived list instead of overwriting `session.ActiveTodos`
- `cmd/session_cmds.go` — `session-update` path applies the same merge instead of a bare assignment
- `cmd/shelf_todo_wiring_test.go` (new) — four tests proving the full chain: promote → init → session.json → session refresh
- `cmd/init_wrapper_ceremony_test.go` — forbidden slice gained four entries fencing the hand-append vocabulary; new `approval_writes_pheromones_only_after_init_succeeds` subtest asserts init precedes pheromone-write in the Approval section
- `.claude/commands/ant/init.md`, `.opencode/commands/ant/init.md`, `.claude/commands/ant-init.md` — Shelf Backlog bullet rewritten to describe the runtime `todos` array instead of a hand-append; Approval bullets reordered so `aether init` runs before `aether pheromone-write`, gated on success; Stop conditions updated to cover failure as well as cancel
- `.aether/commands/init.yaml` — two new `guardrails` bullets documenting the runtime-owned shelf-todo promise and the pheromone-write-after-init-success ordering

## Decisions Made

- Chose to route the shelf-to-todo promise through the runtime rather than simply deleting the offending sentence, because deleting alone would silently drop the "promoted items become todos" promise the wrapper makes to the user (see plan `<objective>`).
- Used a merge-not-overwrite strategy (`mergeShelfTodos`) at both session-refresh call sites rather than special-casing the post-init state, since any future session refresh (not just the one immediately after init) needed to preserve shelf todos.

## Deviations from Plan

None — plan executed exactly as written. All three tasks matched their specified action, verify, and acceptance criteria without requiring Rule 1-4 deviations.

## Issues Encountered

- Initial forbidden-string literals for Task 2 misparsed the CommonMark double-backtick code-span space-stripping rule (added a stray trailing space to `` append `[shelf: `` and a stray leading `to ` ambiguity). Corrected by testing the RED state directly against the unmodified wrapper text and adjusting the literal to `append \`[shelf:` (no trailing space), confirmed by rerunning the test.
- A code-comment restating the forbidden phrase `in the session file or colony state` initially pushed that phrase's occurrence count to 2 in the test file; reworded the comment to keep the phrase's only occurrence in the forbidden slice itself, matching the plan's acceptance criterion of exactly 1.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Truth 10 / blocker CR-01 closed: no init.md surface instructs a hand-write to protected state.
- Review WR-05 closed: pheromones are written only after `aether init` succeeds.
- `go test ./cmd/... ./pkg/...` passes in full (run twice, both clean); `go vet ./...` clean; `go build ./cmd/aether` exits 0.
- `.claude/commands/ant-init.md` remains byte-identical to `.claude/commands/ant/init.md`; the `.claude`/`.opencode` init.md files differ on exactly one sanctioned line pair (AskUserQuestion vs. Ask wording).

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*
