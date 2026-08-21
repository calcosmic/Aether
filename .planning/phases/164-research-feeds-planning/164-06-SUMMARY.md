---
phase: 164-research-feeds-planning
plan: 06
subsystem: ts-host research confidence loop
tags: [confidence-loop, research-loop, plan-command, D-09, D-10, D-12, RESEARCH-07, RESEARCH-08]

# Dependency graph
requires:
  - phase: 164-research-feeds-planning plan 03
    provides: "researchLoopPreset/researchLoopOptions (depth->target/iteration binding) and ResearchConfidenceEvaluator (evidence-based scorer)"
provides:
  - "runResearchConfidenceLoop — the research-path ConfidenceLoop construction site host.ts was missing"
  - "runDispatchedPlanCommand now partitions phase_research dispatches through their own iteration loop before the rest of the wave"
  - "research_iterations summary surfaced in the plan command's stdout JSON"
affects: [164-08 (Oracle escalation on stalled/below-target phases), plan command CLI behaviour]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-phase ConfidenceLoop instantiation (Map<phaseId, ConfidenceLoop>) rather than a batch average, so each research Scout's progress is tracked independently"
    - "renderIterationCeremony extended with an optional prefix parameter instead of a third ceremony-renderer copy"

key-files:
  created:
    - .aether/ts-host/test/research-confidence-loop.test.ts
  modified:
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/dist/host.js
    - .aether/ts-host/dist/host.d.ts
    - .aether/ts-host/dist/research-confidence.js
    - .aether/ts-host/dist/research-confidence.d.ts

key-decisions:
  - "runResearchConfidenceLoop is declared with the `export` keyword directly at its definition (not re-exported through the line-128 test-only list) because the plan's own acceptance criteria greps for the literal string \"export async function runResearchConfidenceLoop\" — TypeScript errors (TS2323) on declaring a name export and then re-exporting it via `export { ... }` in the same module, so the two instructions in the plan text were reconciled in favor of the automated, testable check."
  - "Research loop's per-instance totalBudget defaults to 20 (RESEARCH_LOOP_DEFAULT_BUDGET), mirroring the build path's `?? 20` fallback, since PlanManifest carries no queen_execution_policy.spawn_budget field the way BuildManifest does. The depth preset's maxIterations (max 12, exhaustive) stays well under 20, so the iteration cap is the binding constraint in practice and the budget is the secondary safety net T-164-16 calls for."
  - "--target/--max-iterations overrides are clamped to the documented 70-99 / 2-12 ranges before being written into ConfidenceLoopOptions (T-164-17), scoped only to the new research path — the build path's existing unclamped parseInt was left untouched as out-of-scope for this plan."

patterns-established:
  - "Ceremony renderer extension over duplication: renderIterationCeremony(result, prefix?) — omitting prefix reproduces the build path's byte-identical line; passing `${scoutName} phase ${id}` labels each research iteration."

requirements-completed: [RESEARCH-07, RESEARCH-08]

# Metrics
duration: ~45min
completed: 2026-08-02
---

# Phase 164 Plan 06: Research Confidence Loop Wiring Summary

One-liner: `runDispatchedPlanCommand` now constructs one `ConfidenceLoop` per approved research phase via a new `runResearchConfidenceLoop`, iterating each research Scout to its depth-bound target (RESEARCH-08: fast 80%/4, balanced 90%/6, deep 95%/8, exhaustive 99%/12) with a ceremony line every iteration and an early-accept prompt on stall or near-target — closing the gap that left RESEARCH-07 unmet.

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-02T11:48:49Z
- **Tasks:** 2/2 completed
- **Files modified:** 6 (1 test file created, 5 modified — 1 source, 4 embedded dist artifacts)

## Accomplishments

- `runResearchConfidenceLoop` gives the plan path a real confidence loop: one `ConfidenceLoop` instance per research phase (keyed by phase ID, never a batch average), each fed `researchLoopOptions(depth, budget)` from Plan 03 and scored every round by `ResearchConfidenceEvaluator` against the real `phase-N-research.md` artifact on disk.
- `runDispatchedPlanCommand` restructured to partition dispatches on `stage === "phase_research"`, await the research loop first, then dispatch the remaining wave — preserving the existing ordering contract (all research completes before the Route-Setter) while giving research its own iteration budget.
- `renderIterationCeremony` extended with an optional `prefix` parameter (`${scoutName} phase ${id}`) instead of adding a third copy of the ceremony renderer; the build path's call site (no prefix) produces a byte-identical line to before.
- Early-accept prompting (D-10) implemented per-phase with a one-shot flag: fires once when a phase's confidence is within 5 points of target with iterations remaining, or once when it stalls below target via `diminishing_returns` — never both, never repeated.
- `--target`/`--max-iterations` overrides on the research path are clamped to their documented ranges (70-99, 2-12) before entering `ConfidenceLoopOptions`, per threat T-164-17.
- 7 new tests drive `runResearchConfidenceLoop` directly against real evaluator scoring and fixture markdown files, including an invariant test that fails the suite if the research construction site is ever deleted (the exact regression 164-RESEARCH.md flags as RESEARCH-07's unmet half).

## Task Commits

1. **Task 1: runResearchConfidenceLoop — one ConfidenceLoop per approved phase** - `dce24e9d` (feat)
2. **Task 2: Loop tests and dist rebuild** - `515b5507` (test)

_No architectural deviations; both tasks landed as specified with the reconciliation noted in Key Decisions above._

## Files Created/Modified

- `.aether/ts-host/src/host.ts` - Added `runResearchConfidenceLoop`, `ResearchLoopSummary`/`ResearchLoopPhaseSummary` types, phase-ID/markdown-reading/gap-derivation helpers, clamp helpers, and extended `renderIterationCeremony` with an optional prefix. Restructured `runDispatchedPlanCommand` to partition and route research dispatches through the new loop before the rest of the wave, merging results and adding `research_iterations` to the stdout JSON.
- `.aether/ts-host/test/research-confidence-loop.test.ts` - New: 6 behavioural tests (zero dispatches, 6-round never-improving cap, target-met-vs-continuing sibling, `--accept` short-circuit, per-iteration ceremony content, single early-accept prompt) plus 1 construction-site invariant test.
- `.aether/ts-host/dist/host.js`, `dist/host.d.ts`, `dist/research-confidence.js`, `dist/research-confidence.d.ts` - Rebuilt `go:embed` bundle; `research-confidence.js`/`.d.ts` are newly emitted here because this plan is the first to actually import that module from `host.ts`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Spec conflict resolved in favor of the automated check] `runResearchConfidenceLoop` export mechanism**
- **Found during:** Task 1 implementation
- **Issue:** The plan's `<action>` text asked to both declare `export async function runResearchConfidenceLoop(...)` at its definition AND add it to the existing test-only re-export list at host.ts:128 (`export { runDispatchedBuildCommand, ... }`). TypeScript raises TS2323 ("Cannot redeclare exported variable") when a name is exported both at its declaration and via a separate `export { name }` statement in the same module — confirmed with a scratch `tsc --strict` repro before implementing.
- **Fix:** Declared it with `export async function runResearchConfidenceLoop` directly (satisfying the acceptance criterion's literal grep for that exact string) and left the line-128 list untouched, since the function is already publicly exported at its declaration.
- **Files modified:** `.aether/ts-host/src/host.ts`
- **Commit:** `dce24e9d`

No other deviations — both tasks otherwise executed as written, including the threat-model mitigations (T-164-16 iteration/budget caps, T-164-17 flag clamping, T-164-18 ceremony content scope).

## Verification

- `npm --prefix .aether/ts-host run typecheck` — exits 0
- `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence-loop.test.ts` — 7/7 tests pass
- `npm --prefix .aether/ts-host run test:all` — 542/542 tests pass (no regressions)
- `npm --prefix .aether/ts-host run build` — exits 0, `dist/` regenerated and committed
- `go build ./cmd/aether` — exits 0 (the `//go:embed` bundle resolves)
- `git diff --stat .aether/ts-host/src/confidence-loop.ts` — empty (untouched, per RESEARCH-07)
- `grep -c 'new ConfidenceLoop(' .aether/ts-host/src/host.ts` — 2 (build path + research path)
- `grep -c 'function renderIterationCeremony' .aether/ts-host/src/host.ts` — 1 (no third copy)

## Threat Flags

None. All three threats in this plan's own `<threat_model>` were implemented as specified:
- T-164-16 (unbounded research iteration): each phase's loop is constructed with the depth preset's `maxIterations` and a shared `RESEARCH_LOOP_DEFAULT_BUDGET` (20); `ConfidenceLoop.checkStopConditions` enforces both before any other condition, unmodified from Plan 03/pre-existing code.
- T-164-17 (flag override tampering): `--target`/`--max-iterations` are parsed then clamped to the documented 70-99 / 2-12 ranges before entering `ConfidenceLoopOptions`.
- T-164-18 (ceremony information disclosure): ceremony and early-accept lines carry only phase number, Scout name, confidence percentage, delta, and budget — never research body text, matching the existing build-path contract.

## Self-Check: PASSED

- FOUND: .aether/ts-host/test/research-confidence-loop.test.ts (created, 7/7 tests pass)
- FOUND: .aether/ts-host/src/host.ts contains `export async function runResearchConfidenceLoop`
- FOUND commit dce24e9d: feat(164-06): construct a ConfidenceLoop per research phase in plan path
- FOUND commit 515b5507: test(164-06): pin research loop iteration count, ceremony, and construction site
