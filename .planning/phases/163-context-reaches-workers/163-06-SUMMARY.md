---
phase: 163-context-reaches-workers
plan: 06
subsystem: cli
tags: [go, cobra, context-inspection, budget-guard, worker-brief]

# Dependency graph
requires:
  - phase: 163-context-reaches-workers (plan 01)
    provides: codexBuildManifest.ContextCapsule / resolveCodexWorkerContext, the manifest-level capsule this plan displays
  - phase: 163-context-reaches-workers (plan 02)
    provides: the charter colonyPrimeSection inside the capsule
  - phase: 163-context-reaches-workers (plan 04)
    provides: surveyStalenessNotice inside the Territory Survey section
provides:
  - "aether build <n> --print-brief" checklist-by-default inspector (present/absent, size, total vs. budget)
  - "--full" escape hatch for the raw assembled prompt + composition table
  - first-ever test coverage for printWorkerBriefs (10 CLI-level behaviors across two test funcs)
  - TestAssembledContextStaysUnderBudgetCeiling, a derived-ceiling growth guard over the full assembled context
affects: [163-context-reaches-workers (phase verification/publish gate), future context-budget work]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Checklist-by-default inspector: an expected-section registry with present/absent + char-count rows, raw output opt-in behind a flag"
    - "Derived budget ceiling: sum of named budget constants + a documented judgement-call allowance, never a literal total"

key-files:
  created:
    - cmd/build_print_brief_test.go
    - cmd/context_budget_test.go
  modified:
    - cmd/build_print_brief.go
    - cmd/codex_build.go
    - cmd/codex_workflow_cmds.go
    - cmd/testdata/command_catalog.json

key-decisions:
  - "Checklist rows are located by direct heading-text search (locateChecklistSection), not by extending splitBriefSections/briefOwnedSections -- keeps the checklist decoupled from that registry's exact heading spelling"
  - "Fixed a pre-existing bug while touching briefOwnedSections: the map key read \"Codegraph Context\" but the actual heading text is \"## Codebase Graph Context\", so that section was never attributed correctly by splitBriefSections before this plan"
  - "Territory Survey renders under an \"### \" (h3) heading, one level below the \"## \" briefOwnedSections convention -- left as-is (changing it would break two existing tests that assert on the literal \"### Territory Survey\" text); the checklist detects it via direct substring search instead of relying on splitBriefSections"
  - "Budget test fixture deliberately omits survey artifacts: since Territory Survey has no top-level \"## \" boundary, its content would otherwise bleed into whichever \"## \" section precedes it, inflating that section's measured size past what it actually contains"
  - "briefTaskContentAllowanceChars = 6000 and assembledContextTaskShareFloorPercent = 5.0 are both judgement calls, picked from reasoning about the fixture's likely order of magnitude rather than reverse-engineered from the exact measured numbers, then confirmed the fixture clears them"

requirements-completed: [CONTEXT-07, CONTEXT-08, CONTEXT-09]

# Metrics
duration: ~55min
completed: 2026-07-29
---

# Phase 163 Plan 06: Checklist-by-default context inspector Summary

**`aether build --print-brief` now defaults to a present/absent context checklist against a derived budget instead of unconditionally dumping the raw prompt; `--full` remains one flag away, and a new budget-ceiling test bounds every future context addition.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- D-06: `aether build <n> --print-brief` now answers "did the charter arrive?" in ten seconds — a sectioned checklist listing Assignment & Task Content, Territory Survey, Phase Research, Codegraph Context, Pheromone Signals, Previous Worker Handoffs, Expected Output, the manifest-level Context Capsule, and Charter, each with present/ABSENT and a char count, plus the survey staleness notice surfaced as a row annotation.
- `--full` prints the raw assembled prompt and composition table on demand, unchanged from the pre-existing behavior; it is now opt-in rather than the only option, and is inert on every mutating build path.
- CONTEXT-07/CONTEXT-08: the checklist reports a measured total (manifest capsule + brief + skill section) against a budget ceiling *derived* by summing `colonyPrimeBudgetChars`, `skillInjectNormalBudgetChars`, `phaseResearchBriefBudgetChars`, `codegraphWorkerContextBudgetChars`, and a new named `briefTaskContentAllowanceChars` constant — never a hardcoded total.
- CONTEXT-09: `TestAssembledContextStaysUnderBudgetCeiling` is a growth guard, not a gate — it bounds the total and each individually budgeted section against its own declared constant, and a task-content share floor, cross-checked against the checklist's own printed numbers so the two can never silently diverge.
- First-ever test coverage for `printWorkerBriefs`: 10 CLI-level behaviors across `TestPrintBriefChecklist` and `TestPrintBriefFullFlagAndCoverage`, using real fixtures (a git repo, a codebase graph, real phase-research content) rather than stubbed resolvers.
- Fixed a latent bug in `briefOwnedSections` discovered while building the budget test: the registry's "Codegraph Context" key never matched the renderer's actual "## Codebase Graph Context" heading, so `splitBriefSections`/`TestBuildWorkerBriefIsMostlyTask`'s underlying machinery silently mis-attributed that section's content before this plan.

## Task Commits

Each task was committed atomically:

1. **Task 1: A checklist that knows what should have been there** - `4b9d1ae5` (feat) — `renderBriefChecklist`, the expected-section registry, `briefOwnedSections` rename fix, and the checklist becoming the unconditional default output for `--print-brief`.
2. **Task 2: --full for the raw prompt, and the command's first test coverage** - `05d866dd` (feat) — `--full` flag registration, `codexBuildOptions.Full`, corrected help text, and 5 more CLI-level test behaviors.
3. **Task 3: Measure the total against real budgets, and bound it** - `d1c71e49` (test) — `TestAssembledContextStaysUnderBudgetCeiling`, `briefTaskContentAllowanceChars`, `assembledContextTaskShareFloorPercent`, and the `command_catalog.json` golden regeneration for the new `--full` flag.

**Plan metadata:** (this commit)

_Note: all three tasks used `tdd="true"` — behaviors were written and run against the implementation in the same commit per task rather than as separate RED/GREEN commits, matching the plan's `<behavior>`-driven task structure rather than the strict RED→GREEN→REFACTOR gate sequence._

## Files Created/Modified

- `cmd/build_print_brief.go` - `renderBriefChecklist`, `locateChecklistSection`, `checklistRowFor`, `briefTaskContentAllowanceChars`, `assembledContextBudgetCeilingChars`, `briefOwnedSections` fix, `--full` branching, `buildPrintBriefOptions` signature update
- `cmd/build_print_brief_test.go` - first test coverage for `printWorkerBriefs`: checklist rows, absence, capsule/charter sourcing, staleness annotation, raw-body exclusion, `--full`, `--worker` scoping, inertness, unknown-worker error
- `cmd/context_budget_test.go` - `TestAssembledContextStaysUnderBudgetCeiling` and `assembledContextTaskShareFloorPercent`
- `cmd/codex_build.go` - `codexBuildOptions.Full` field
- `cmd/codex_workflow_cmds.go` - `--full` flag registration, corrected `--print-brief` help text, flag wiring into `buildPrintBriefOptions`
- `cmd/testdata/command_catalog.json` - regenerated golden fixture (adds the `full` flag entry under `build`)

## Decisions Made

- Checklist section detection uses a dedicated `locateChecklistSection` heading-text search rather than extending `splitBriefSections`/`briefOwnedSections`, so the checklist's correctness doesn't depend on that registry's exact heading spellings staying in sync (see key-decisions in frontmatter for the rest).
- Left the Territory Survey `### ` (h3) heading as-is rather than promoting it to `## ` for consistency with `briefOwnedSections`, because two existing tests (`TestBuildWorkerBriefIncludesSurveyAndResearch`, the print-brief acceptance criteria) assert on the literal three-hash heading text.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `briefOwnedSections` key mismatch for the codegraph section**
- **Found during:** Task 1, while introducing the checklist's expected-section registry and reasoning about what `splitBriefSections` would actually measure
- **Issue:** `briefOwnedSections` declared a `"Codegraph Context"` key, but `renderCodegraphContext` emits the heading `## Codebase Graph Context`. The two never matched, so `splitBriefSections` silently folded all codegraph content into whichever section preceded it instead of giving it its own measured size.
- **Fix:** Renamed the map key to `"Codebase Graph Context"` to match the actual heading text.
- **Files modified:** cmd/build_print_brief.go
- **Verification:** `TestAssembledContextStaysUnderBudgetCeiling`'s per-section budget check now correctly isolates and bounds the Codebase Graph Context section against `codegraphWorkerContextBudgetChars`; `TestBuildWorkerBriefIsMostlyTask` (pre-existing, unrelated to this fix) still passes.
- **Committed in:** `4b9d1ae5` (Task 1 commit)

**2. [Rule 3 - Blocking] Regenerated the `command_catalog.json` golden fixture**
- **Found during:** Task 3, running the full `go test ./cmd` suite before committing
- **Issue:** `TestAuditCatalogGolden` failed because the new `--full` flag on `build` changed the catalog's byte-for-byte snapshot (242858 vs 242844 bytes).
- **Fix:** Ran `go test ./cmd -run TestAuditCatalogGolden -update-golden` and reviewed the diff — the only change is a single new `"full"` entry in `build`'s flags array, in alphabetical position between `force` and `heavy`, exactly as expected.
- **Files modified:** cmd/testdata/command_catalog.json
- **Verification:** `go test ./cmd -count=1` passes; `diff` reviewed and shown to be exactly the expected one-line addition.
- **Committed in:** `d1c71e49` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug fix, 1 blocking golden regeneration)
**Impact on plan:** Both were necessary for correctness/build-passing. No scope creep.

## Issues Encountered

- The budget test initially reported an 8-char mismatch between the total computed by direct function calls and the total the CLI checklist printed. Root cause: on macOS, `t.TempDir()` returns an unresolved `/var/...` path while `os.Getwd()` after `os.Chdir()` resolves the `/private/var` symlink, and that resolution difference showed up in the rendered `- Workspace: %s` brief line. Fixed by having the test resolve root via `skillWorkspaceRoot()` (the same call the CLI path makes) instead of the raw `t.TempDir()` value — a test-environment artifact, not a production bug.

## CONTEXT-09 / D-08 Evidence Statement

Per D-08 (no staged benchmark), CONTEXT-09 is satisfied by this inspector plus the automated presence and budget tests across plans 01–06, specifically:
- `TestPrintBriefChecklist` (5 behaviors): presence/absence, capsule/charter sourcing, staleness annotation, raw-body exclusion
- `TestPrintBriefFullFlagAndCoverage` (5 behaviors): `--full` gating, `--worker` scoping in both modes, inertness, unknown-worker error
- `TestAssembledContextStaysUnderBudgetCeiling` (4 behaviors): total ceiling, per-section budgets, task-content share floor, checklist/test agreement

Measured totals for this plan's test fixture (one phase, two tasks, a real charter, a matching codegraph, real phase-research content, no survey artifacts): capsule=334 chars, brief=863 chars, skill=0 chars, **total=1197 chars** against a derived ceiling of **27700 chars** (`colonyPrimeBudgetChars` 8000 + `skillInjectNormalBudgetChars` 8000 + `phaseResearchBriefBudgetChars` 3500 + `codegraphWorkerContextBudgetChars` 2200 + `briefTaskContentAllowanceChars` 6000). This is a synthetic two-task fixture, not a representative real colony — real-world cheap-model validation (do these numbers hold up on an actual multi-phase colony with real skill matches, pheromone signals, and worker handoffs) is explicitly deferred post-phase per D-08, matching CONTEXT-09's descoped-benchmark decision.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- This was the last plan to land in phase 163-context-reaches-workers, so it owns the phase-wide publish gate noted in the plan's `<verification>` block (`aether publish`, `aether integrity`, `aether version --check`). **Not run by this executor**: publishing mutates the shared `~/.aether` hub, a host-machine resource outside this git worktree that other colonies on the same machine may depend on, and this worktree cannot confirm sibling wave plans have already merged. This is flagged for the orchestrator/user to run explicitly once all of phase 163's plans are merged to the base branch.
- All three tasks' automated verification commands pass: `go test ./cmd -run 'TestPrintBrief|TestAssembledContextStaysUnderBudgetCeiling' -count=1`, `go test ./... -count=1 -race` (entire repo, all packages green), and `go build ./cmd/aether && go vet ./...` all exit 0.
- Manual validation from VALIDATION.md ("on a real colony, the checklist answers 'did the charter arrive?' within ten seconds") could not be performed in this isolated worktree — there is no initialized colony state here — and is explicitly deferred to the user per the plan's own D-08 evidence framing.

## Self-Check: PASSED

All created/modified files confirmed present on disk; all four commit hashes
(`4b9d1ae5`, `05d866dd`, `d1c71e49`, `afa07bdb`) confirmed in `git log`.

---
*Phase: 163-context-reaches-workers*
*Completed: 2026-07-29*
