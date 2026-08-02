---
phase: 164-research-feeds-planning
plan: 10
subsystem: infra
tags: [typescript, go, worker-dispatch, permission-boundary, gap-closure]

# Dependency graph
requires:
  - phase: 164-research-feeds-planning (plans 01-09)
    provides: phase-domain research dispatch machinery (renderPhaseResearchBrief, plannedPhaseResearchDispatches, confidence loop)
provides:
  - Field-faithful toWorkerDispatches that copies permission_profile and brief (as task_brief) through from Go to the worker request
  - TS permission fallback for scout aligned with Go's canonical repositoryReadOnlyCastes map (workspace_write, not repository_read_only)
  - Non-mocked field-fidelity invariant test guarding the Go->worker conversion boundary
  - Go-side proof that a real plan-time phase_research dispatch resolves through internalWorkerConfig/ResolvePermissionProfile for caste scout
  - Rebuilt .aether/ts-host/dist/ carrying both fixes, proven by direct execution
  - Wrapper/command-guide docs no longer claim a host-enforced repository_read_only sandbox for Scout
affects: [165-core-lifecycle-commands, any future phase touching plan-time worker dispatch or the TS host conversion boundary]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Field-fidelity invariant test: classify every source field as mapped/renamed/intentionally-unmapped, so a newly added field fails the test until consciously classified"
    - "Cross-language invariant test: derive the expected caste set from the Go source file at test time (regex over pkg/codex/permission_profile.go), rather than hardcoding a mirrored list, so the TS test fails the moment Go's canonical map changes without updating the TS mirror"

key-files:
  created:
    - .aether/ts-host/test/dispatch-field-fidelity.test.ts
    - cmd/phase_research_permission_boundary_test.go
  modified:
    - .aether/ts-host/src/types.ts
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/src/worker-dispatch.ts
    - .aether/ts-host/dist/host.js
    - .aether/ts-host/dist/host.d.ts
    - .aether/ts-host/dist/worker-dispatch.js
    - .aether/ts-host/dist/worker-dispatch.d.ts
    - .aether/ts-host/dist/types.d.ts
    - .aether/commands/plan.yaml
    - .claude/commands/ant/plan.md
    - .claude/commands/ant-plan.md
    - .opencode/commands/ant/plan.md
    - cmd/command_guide.go

key-decisions:
  - "toWorkerDispatches now copies permission_profile through verbatim (never invents one) and promotes brief to task_brief only when task_brief was not already host-injected, preserving build/continue precedence"
  - "TS permissionProfileForCaste fallback narrowed to match Go's repositoryReadOnlyCastes exactly (only 'includer'); scout is workspace_write"
  - "Task 2's Go tests required no Go source changes -- the Go side (PermissionProfileForCaste, ResolvePermissionProfile) was already correct; the gap was the absence of a test that drove the boundary from a real dispatch instead of a hand-written profile literal"
  - "All doc references to a host-enforced Scout repository_read_only sandbox replaced with an accurate claim: permission_profile passes through verbatim, and Scout's canonical profile is workspace_write scoped behaviorally to .aether/data/phase-research"

requirements-completed: [RESEARCH-01, RESEARCH-02, RESEARCH-03, RESEARCH-04, RESEARCH-05, RESEARCH-07]

# Metrics
duration: 13min
completed: 2026-08-02
---

# Phase 164 Plan 10: Field-Fidelity Fix for toWorkerDispatches (CR-01/CR-02 Gap Closure) Summary

**Fixed `toWorkerDispatches` silently dropping `permission_profile` and `brief` on every plan-time research dispatch, which made every real Scout worker fail Go's exact-equality permission check before doing any work.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-02T17:39:05+02:00
- **Completed:** 2026-08-02T17:52:01+02:00
- **Tasks:** 3
- **Files modified:** 15 (2 created, 13 modified)

## Accomplishments

- `toWorkerDispatches` (the TS host's Go->worker conversion boundary) now passes `permission_profile` through verbatim and promotes Go's `brief` field to `task_brief` (with host-injected `task_brief` still taking precedence for build/continue paths)
- TS-side `permissionProfileForCaste` fallback narrowed to mirror Go's `repositoryReadOnlyCastes` exactly (only `includer`), so scout now resolves to `workspace_write` instead of the stale `repository_read_only`
- Added a non-mocked field-fidelity invariant test (`dispatch-field-fidelity.test.ts`) that drives the real exported `toWorkerDispatches` and fails on any future silently-dropped Go dispatch field
- Added a Go-side proof (`phase_research_permission_boundary_test.go`) that a real `plannedPhaseResearchDispatches` output, round-tripped through JSON, resolves correctly at `internalWorkerConfig`/`ResolvePermissionProfile` for caste scout — and that the stale read-only profile is still rejected
- Rebuilt `.aether/ts-host/dist/` (the artifact the runtime actually executes) and proved via direct `node` execution that both fields survive conversion in the compiled output
- Removed the stale "Scout `repository_read_only` must remain host-enforced" claim from `plan.yaml`, all three byte-parallel wrapper markdown files, and `cmd/command_guide.go`'s plan PreSteps — replaced with the claim the runtime actually makes

## Task Commits

Each task was committed atomically (Tasks 1 and 2 used TDD's RED/GREEN discipline):

1. **Task 1 RED: field-fidelity test** - `cf261613` (test)
2. **Task 1 GREEN: toWorkerDispatches + TS fallback fix** - `fea437a6` (feat)
3. **Task 2: Go boundary proof test** - `36fac006` (test)
4. **Task 3: rebuild dist + retire stale doc claim** - `d57754c3` (fix)

## Files Created/Modified

- `.aether/ts-host/test/dispatch-field-fidelity.test.ts` - Non-mocked invariant test over the real `toWorkerDispatches`; classifies every `PlanningDispatch`/`ContinueExternalDispatch` field as mapped, renamed, or intentionally unmapped
- `.aether/ts-host/src/types.ts` - Added `permission_profile?: PermissionProfile` to `PlanningDispatch` and `ContinueExternalDispatch`
- `.aether/ts-host/src/host.ts` - `toWorkerDispatches` now copies `permission_profile` through and promotes `brief` to `task_brief` with correct precedence; exported for direct test access
- `.aether/ts-host/src/worker-dispatch.ts` - `permissionProfileForCaste` read-only predicate narrowed to `includer` only; exported for direct test access
- `.aether/ts-host/dist/{host,worker-dispatch,types}.{js,d.ts}` - Rebuilt compiled artifacts carrying both fixes
- `cmd/phase_research_permission_boundary_test.go` - Proves a real plan-time research dispatch resolves at the Go permission boundary for caste scout, JSON round-trip included
- `.aether/commands/plan.yaml`, `.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md`, `.opencode/commands/ant/plan.md`, `cmd/command_guide.go` - Replaced the stale "Scout repository_read_only must be host-enforced" claim with an accurate one

## Decisions Made

- Task 2 required no Go source changes — `PermissionProfileForCaste`/`ResolvePermissionProfile` were already correct; the actual gap was the total absence of a test driving the boundary from a *real* dispatch (every prior test used a hand-written profile literal, which is exactly what let CR-01 hide). The three new Go tests pass immediately against the existing Go source and document this as intentional proof-of-correctness work, not a bug fix — confirmed by temporarily reintroducing `"scout": {}` into `repositoryReadOnlyCastes`, which reliably breaks `TestPlanResearchDispatchResolvesAtWorkerBoundary` before the file was restored (`git diff --exit-code` clean).
- `brief` -> `task_brief` promotion only fires when `task_brief` is not already present, preserving build/continue's existing host-injection precedence.
- Doc claim replacement text: "permission_profile must be passed through verbatim from the manifest, never substituted or broadened; Scout's canonical profile is workspace_write, scoped behaviorally to .aether/data/phase-research" — applied identically across all five surfaces so wrapper-parity stays intact.

## Deviations from Plan

None - plan executed exactly as written. `npm install` was required in `.aether/ts-host` before any test could run (worktree had no `node_modules/`), which is expected first-run worktree setup, not a plan deviation.

## Issues Encountered

- Initial worktree had no `node_modules/` in `.aether/ts-host`; running the RED test surfaced an unrelated `ERR_MODULE_NOT_FOUND` for `log-update` before the real (RED) failure could show. Resolved with `npm install` (63 packages), then reran to confirm the intended RED failure (missing exports for `toWorkerDispatches`/`permissionProfileForCaste`).

## TDD Gate Compliance

- Task 1 (tdd="true"): RED commit `cf261613` (test, confirmed failing — module export errors) followed by GREEN commit `fea437a6` (feat, confirmed 4/4 passing). No REFACTOR commit needed.
- Task 2 (tdd="true"): Single `test` commit `36fac006`. All three tests pass against the existing Go source with no companion source change, because the Go side of this boundary was already correct — the task's own `<files>` list only the test file, and its `<action>` block describes no Go source edits. Verified this is not a false-negative RED skip by temporarily reintroducing `"scout": {}` into `pkg/codex/permission_profile.go`'s `repositoryReadOnlyCastes`, which reliably broke `TestPlanResearchDispatchResolvesAtWorkerBoundary`, then restored the file (`git diff --exit-code` confirmed clean).
- Task 3 (type="auto", not tdd): single `fix` commit `d57754c3` covering dist rebuild and doc updates, matching its verify block.

## Known Stubs

None.

## Threat Flags

None — this plan's threat model (T-164-01 through T-164-04) was fully addressed as designed: `toWorkerDispatches` remains pass-through only (never constructs a profile), `ResolvePermissionProfile`'s exact-equality rejection was proven live by `TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary`, and the TS fallback widening is derived from the Go source at test time rather than hand-mirrored.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- RESEARCH-01, -02, -03, -04, -05, and -07 are now unblocked: a real phase_research Scout dispatch carries its correct `permission_profile` and full six-section brief end to end, both in `src/` (unit test) and in the compiled `dist/host.js` the runtime actually executes.
- The field-fidelity test (`dispatch-field-fidelity.test.ts`) will catch the next Go dispatch field silently dropped by `toWorkerDispatches`, closing the regression class that let CR-01/CR-02 hide for months.
- No blockers for Phase 165 or later phases touching plan-time worker dispatch.

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*
