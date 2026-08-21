---
phase: 165-core-lifecycle-commands
plan: 08
subsystem: infra
tags: [wrapper-contract, go-test, markdown-yaml, ceremony]

# Dependency graph
requires:
  - phase: 165-core-lifecycle-commands
    provides: plan.md/build.md/continue.md wrapper contract established in earlier 165 plans, VERIFICATION.md and REVIEW.md naming WR-01 and WR-04
provides:
  - TestLifecycleWrapperReadOnlyBlocksAreConsistent fencing the read/write-boundary contradiction across all 4 lifecycle verbs x 3 surfaces
  - build.md and continue.md <read_only> blocks reconciled with their Guardrails lists (WR-01 closed)
  - plan.md's Clarification Gate relocated ahead of Decision Moment 2 so research approvals are never spent against a manifest about to be discarded (WR-04 closed)
affects: [165-core-lifecycle-commands, any future phase touching build.md/continue.md/plan.md wrapper prose]

# Tech tracking
tech-stack:
  added: []
  patterns: ["read/write-boundary consistency fence (extractReadOnlyBlock + no_permissive_read_phrasing + read_only_block_matches_guardrails subtests)", "ordering fence via assertSubstringsInOrder inOrder slice with a code comment recording why an order is deliberate"]

key-files:
  created: []
  modified:
    - cmd/lifecycle_wrapper_contract_test.go
    - cmd/plan_wrapper_ceremony_test.go
    - .claude/commands/ant/build.md
    - .opencode/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .claude/commands/ant/continue.md
    - .opencode/commands/ant/continue.md
    - .claude/commands/ant-continue.md
    - .claude/commands/ant/plan.md
    - .opencode/commands/ant/plan.md
    - .claude/commands/ant-plan.md
    - .aether/commands/build.yaml
    - .aether/commands/continue.yaml
    - .aether/commands/plan.yaml

key-decisions:
  - "plan.md's already-reconciled read_only wording ('never reads or writes, by hand') was standardized as the wording all lifecycle wrappers must use, rather than inventing a fourth formulation for build.md and continue.md."
  - "The read_only_block_matches_guardrails subtest is conditional on the surface containing 'Do NOT read or write', so init.md (whose guardrails live in YAML, not a markdown heading) is correctly exempted without a special case."
  - "The Clarification Gate purpose line gained one sentence explaining why it now runs before Decision Moment 2, rather than adding a new heading or a third decision moment."

requirements-completed: [CMD-01]

duration: 11min
completed: 2026-08-03
---

# Phase 165 Plan 08: Read/Write-Boundary Contradiction and Clarification-Gate Ordering Summary

**Reconciled build.md/continue.md `<read_only>` wording with their own Guardrails lists (WR-01) and moved plan.md's Clarification Gate ahead of Decision Moment 2 so research approvals are never recorded against a manifest about to be discarded (WR-04), both now fenced by Go tests.**

## Performance

- **Duration:** 11 min (13:28 - 13:39 UTC+2, commit timestamps)
- **Started:** 2026-08-03T13:28:00+02:00 (approx, first commit)
- **Completed:** 2026-08-03T13:39:15+02:00
- **Tasks:** 3
- **Files modified:** 14 (1 new test file, 13 existing wrapper/YAML/test files)

## Accomplishments

- Added `TestLifecycleWrapperReadOnlyBlocksAreConsistent` to `cmd/lifecycle_wrapper_contract_test.go`, iterating all 4 lifecycle verbs (build, continue, plan, init) across both canonical paths and the flat mirror (12 surfaces). Proven RED first for exactly the six offending build/continue surfaces, GREEN for plan/init.
- Reconciled `build.md` and `continue.md`'s `<read_only>` blocks to plan.md's already-correct wording ("never reads or writes, by hand"), propagated byte-for-byte to `.opencode` and the flat mirrors, and recorded the same rule as a new `read_only_boundary` entry in `build.yaml`/`continue.yaml`.
- Moved plan.md's `## Clarification Gate` section from after `## Decision Moment 2 — Research Batch` to between `## Planning Manifest` and `## Decision Moment 2`, so boundary guidance and unresolved clarifications are resolved before `aether plan-research-approve` mutates approval state. Updated the `inOrder` ordering assertion in `cmd/plan_wrapper_ceremony_test.go` with a code comment explaining the deliberate order (citing WR-04), and added `clarification_gate_precedence` to `plan.yaml`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the read/write-boundary consistency fence to the shared lifecycle contract test** - `ce7b41f9` (test)
2. **Task 2: Reconcile the read_only blocks in build.md and continue.md across all surfaces and their YAML sources** - `1f8a6ace` (fix)
3. **Task 3: Move plan.md's Clarification Gate ahead of Decision Moment 2** - `201a5c9f` (fix)

**Plan metadata:** committed alongside this SUMMARY (see final commit for this plan).

## Files Created/Modified

- `cmd/lifecycle_wrapper_contract_test.go` - Added `extractReadOnlyBlock`, `lifecycleWrapperSurfaces`, `relWrapperPath` helpers and `TestLifecycleWrapperReadOnlyBlocksAreConsistent` (3 subtests: exists / no permissive phrasing / matches Guardrails)
- `cmd/plan_wrapper_ceremony_test.go` - Swapped `"## Clarification Gate"` and `"## Decision Moment 2 — Research Batch"` in the `inOrder` slice, with a comment recording the WR-04 rationale
- `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.claude/commands/ant-build.md` - `<read_only>` block reconciled with Guardrails' "Do NOT read or write" bullet
- `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`, `.claude/commands/ant-continue.md` - same reconciliation
- `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, `.claude/commands/ant-plan.md` - Clarification Gate relocated ahead of Decision Moment 2
- `.aether/commands/build.yaml`, `.aether/commands/continue.yaml` - added `read_only_boundary` under `wrapper_additions`
- `.aether/commands/plan.yaml` - added `clarification_gate_precedence` under `wrapper_additions`

## Decisions Made

- Used plan.md's existing reconciled wording verbatim as the standard for build.md and continue.md rather than composing new prose, per the plan's explicit instruction to follow plan.md as the model.
- Kept the `read_only_block_matches_guardrails` subtest conditional (only applies when "Do NOT read or write" is present in the file) rather than requiring every surface to carry the exact same wording, since init.md's guardrails live in YAML rather than markdown and already carries a structurally different (but equally correct) formulation.

## Deviations from Plan

None - plan executed exactly as written. All three tasks matched their `<action>` and `<verify>` blocks; no Rule 1-4 deviations were needed.

## Issues Encountered

- The first full `go test ./cmd/...` run (started immediately after Task 2's commit, before Task 3's edits) returned FAIL because Task 3's file edits landed on disk mid-run while the test binary had already compiled from the pre-Task-3 source — a race condition from working on Task 3 while Task 2's background verification was still executing, not a real defect. Re-ran `go test ./cmd/...` from a clean state after all edits settled; it passed (`ok`, 309.9s) confirming both WR-01 and WR-04 fixes hold together with Task 1's fence.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- WR-01 and WR-04 are closed and fenced by tests; `TestLifecycleWrapperReadOnlyBlocksAreConsistent` and the reordered `inOrder` slice in `TestPlanWrapperCeremonyContract` will fail if either regresses.
- `TestSpecialistCommandSurfacesUnchanged` and `TestBuildMdOwnershipHandshake` both still pass — no collateral damage to Phase 165's CMD-04/CMD-05 deliverables.
- Full `go test ./cmd/...`, `go vet ./...`, and `go build ./cmd/aether` all pass with every plan 08 change applied.
- No blockers for the orchestrator's post-wave STATE.md/ROADMAP.md updates.

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: .planning/phases/165-core-lifecycle-commands/165-08-SUMMARY.md
- FOUND: ce7b41f9 (Task 1 commit)
- FOUND: 1f8a6ace (Task 2 commit)
- FOUND: 201a5c9f (Task 3 commit)
- FOUND: 17843eb9 (this SUMMARY.md commit)
