---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 06
subsystem: cli-visuals
tags: [go, continue, verification, gates, evidence, D-11, SHOW-02]

# Dependency graph
requires:
  - phase: 198-01
    provides: closeoutDirectVisual / closeoutContinueRenderInputs bridge, the dual-type rendering precedent (renderContinueWorkerFlowValue), byte-equality parity test fixtures
  - phase: 198-05
    provides: workerMeasurementFigures / the measured-vs-not-reported figure pair pattern, verificationStepDisplayName and codexVerificationStep.Duration (used, not re-added, by this plan)
provides:
  - "renderContinueVerificationDetail / renderContinueGateDetail (cmd/codex_visuals.go) — named per-check and per-gate lines beneath the existing tally, with fix hints and an honest per-category '(+N more)' cap"
  - "renderCriterionEvidenceLines — D-11's requirement-plus-proof shape ('✓ Login works — proved by: ...'), never a bare tick for an unproven/blocked/awaiting-owner requirement"
  - "renderSpecialistFindingBlocks — every reviewer's findings/recommendations/weak-spots/edge-cases/blockers as their own headed block, visible without reading each worker's nested detail"
  - "gateCheckDisplayNames / continueGateCheckNames — the runtime's own gate-name translation table, doubling as the source TestRestoredDetailIsPlainEnglish derives its internal-name check from"
  - "renderPlanVisual's confidence line now actually renders on both the direct and chat paths (WINDOWS.md entry 5 fixed)"
affects: [any-future-phase-touching-cmd/codex_visuals.go-continue-rendering, 198-07-onward]

# Actuals (#2632)
actuals:
  tokens: 9888
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dual-type detail rendering: renderContinueVerificationDetail/renderContinueGateDetail/renderCriterionEvidenceLines/renderSpecialistFindingBlocks all type-switch on the raw result-map value (the in-process typed struct vs. the JSON-round-tripped map/slice), reducing each shape to a small internal 'view' struct before one shared formatter renders it — follows renderContinueWorkerFlowValue's existing precedent (198-PATTERNS.md) so the typed and round-tripped shapes can never disagree."
    - "Display state derived from data, not trusted from a caller-set flag: criterionEvidenceDisplayState computes satisfied/unproven/blocked/awaiting_owner from a criterion's own Passed/Evidence/BlockingIssues/State fields, and both the mark and the wording come from that single function — a rendering change can never make an unproven requirement look satisfied."
    - "Runtime-table-derived test data: continueGateCheckNames is derived from gateCheckDisplayNames (the actual translation map used by the renderer) rather than a second literal typed into the test file, so TestRestoredDetailIsPlainEnglish cannot go stale when a gate name is added."

key-files:
  created:
    - cmd/continue_detail_render_test.go
  modified:
    - cmd/codex_visuals.go
    - cmd/testdata/golden_continue.txt
    - cmd/testdata/golden_plan.txt

key-decisions:
  - "renderCriterionEvidenceLines sources its data from result[\"verification\"].Criteria (codexCriterionVerification: Criterion/Evidence/Summary/State/Passed/BlockingIssues) rather than result[\"task_evidence\"] (assessment.Tasks, a different and simpler codexContinueTaskAssessment shape with no Evidence field at all). The plan's D-11 example ('✓ Login works — proved by: 3 tests passed, auth.go present') and behavior spec only match codexCriterionVerification's fields; task_evidence in the result map is a misnomer for the task-assessment list, not the criterion-evidence list."
  - "TestRestoredDetailIsPlainEnglish's internal-name scan is scoped to the six snake_case gate keys (manifest_present, verification_steps_passed, etc.), not the verification check keys (build/types/lint/tests). The gate keys are unambiguous code identifiers; the check keys are also ordinary English words that legitimately appear inside plain-English gate prose ('the build's own plan file is on disk'), so scanning for them would produce false positives rather than catching a real leak."
  - "gateCheckDisplayName's translation table lives in cmd/codex_visuals.go (this plan's owned file) as a map, not a switch, specifically so TestRestoredDetailIsPlainEnglish's internal-name list (continueGateCheckNames) can be derived from it rather than hand-typed into the test — satisfying the plan's 'derives from a runtime table, not a literal' requirement with an actual single source of truth."
  - "verificationStepDisplayName (cmd/codex_continue.go) was left untouched: it is outside this plan's declared files_modified (cmd/codex_visuals.go, cmd/continue_detail_render_test.go only). The new formatVerificationStepResultLine formatter lives entirely in codex_visuals.go and duplicates emitVerificationStepFinish's formatting logic rather than refactoring codex_continue.go to share it — a stylistic DRY improvement was not worth an out-of-scope file edit."
  - "Fixed the pre-existing plan-confidence rendering bug recorded in WINDOWS.md entry 5 (found by 198-04): renderPlanVisual's confidence branch only type-asserted to map[string]interface{}, never the codexPlanConfidence struct both plan-finalize construction paths actually store. Fixed with a dual-type switch mirroring the existing planning_loop struct/map switch immediately below it, since this plan owns cmd/codex_visuals.go this wave. Marked fixed in WINDOWS.md."

requirements-completed: [SHOW-02]

coverage:
  - id: D1
    description: "The closing summary names each check that ran and says whether it passed, instead of only a count."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestVerificationDetailNamesEveryCheck"
        status: pass
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestNestedDetailCapReportsWhatItOmitted"
        status: pass
    human_judgment: false
  - id: D2
    description: "The closing summary names each gate that was applied and says whether it passed, with its fix hint when it did not."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestGateDetailNamesEveryGateAndItsFixHint"
        status: pass
    human_judgment: false
  - id: D3
    description: "Each requirement the phase claimed is shown with what proved it, never as a bare tick."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestEvidenceLineNeverAppearsWithoutItsProof"
        status: pass
    human_judgment: false
  - id: D4
    description: "Specialist helpers' findings are shown as their own block rather than folded away."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestSpecialistFindingsGetTheirOwnBlock"
        status: pass
    human_judgment: false
  - id: D5
    description: "The restored detail reaches the chat path (ceremony closeout) byte-identically to the direct path, and reads in plain English."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestChatPathShowsChecksGatesAndEvidence"
        status: pass
      - kind: unit
        ref: "cmd/continue_detail_render_test.go#TestRestoredDetailIsPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath"
        status: pass
    human_judgment: false
  - id: D6
    description: "No bordered or boxed table was introduced by this plan's new rendering."
    verification:
      - kind: unit
        ref: "cmd/display_house_style_test.go#TestHumanDisplaysUseHeadedSectionsNotMachineTables"
        status: pass
    human_judgment: false
  - id: D7
    description: "Plan confidence (Confidence: N% overall) now renders on both the direct and chat paths — a pre-existing bug (WINDOWS.md entry 5) fixed as part of this plan's ownership of cmd/codex_visuals.go."
    verification:
      - kind: unit
        ref: "cmd/golden_workflow_test.go#TestGoldenPlanVisualOutput"
        status: pass
    human_judgment: false

duration: ~65min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 6: Put the Checks, Gates, and Evidence on Screen Summary

**The continue closing summary now names every check and gate it ran (not just a count), shows what actually proved each requirement, and surfaces specialist reviewers' findings as their own block — on the direct terminal path and the chat path identically, plus a fixed pre-existing bug where plan confidence never rendered at all.**

## Performance

- **Duration:** ~65 min
- **Tasks:** 3 completed
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `renderContinueVerificationDetail` and `renderContinueGateDetail` (`cmd/codex_visuals.go`) name every verification check ("Build ✓ (1.2s)", "Tests ✗ (4.3s) — 2 of 12 tests failed") and every gate ("✓ the build's own plan file is on disk") beneath the existing tally lines, with a failed check's command or a failing gate's fix hint on the nested `└──` detail line, capped per category at 8 with an honest `(+N more)` — the arithmetic remainder computed independently, never a constant.
- `renderCriterionEvidenceLines` renders D-11's requirement-plus-proof shape from the verification report's `Criteria` list: `✓ Login works — proved by: 3 tests passed, auth.go present`. A requirement's satisfied mark and its displayed state are both derived from the same `criterionEvidenceDisplayState` function, so a rendering change can never make an unproven, blocked, or awaiting-owner-confirmation requirement look satisfied.
- `renderSpecialistFindingBlocks` pulls every review-stage worker's findings, recommendations, weak spots, edge cases, and blockers into a `🔍 Specialist Findings` block, visible without reading each worker's per-line nested detail. A worker with nothing to report is silently excluded.
- All four new render functions dual-type switch between the in-process typed struct and the JSON-round-tripped map/slice shape (the `renderContinueWorkerFlowValue` precedent from 198-01), wired into both `renderContinueVisual` and `renderContinueBlockedVisual` so the direct terminal and chat-path closeout render byte-identically by construction (`TestWrapperPathRendersSameCeremonyAsDirectPath`, `TestChatPathShowsChecksGatesAndEvidence`).
- `TestRestoredDetailIsPlainEnglish` derives its internal-name check from `continueGateCheckNames`, itself derived from `gateCheckDisplayNames` (the actual translation table `renderContinueGateDetail` reads), so a newly added gate name can't silently skip the plain-English check.
- Fixed a genuine, previously-invisible bug recorded in `.planning/WINDOWS.md` entry 5 (found by 198-04): `renderPlanVisual`'s confidence line never rendered on any path because it only type-asserted `result["confidence"]` to `map[string]interface{}`, never the `codexPlanConfidence` struct both plan-finalize construction paths actually store. Fixed with a dual-type switch mirroring the existing `planning_loop` struct/map switch immediately below it (in scope since this plan owns `cmd/codex_visuals.go` this wave). WINDOWS.md entry 5 marked fixed.

## Task Commits

Each task was committed atomically:

1. **Task 1: Name every check and every gate, with its outcome** - `832d82b9` (feat)
2. **Task 2: Every requirement shows what proved it** - `1cc83123` (feat)
3. **Task 3: Prove the new detail survives to the chat path and reads as plain English** - `235e186e` (test)

## Files Created/Modified

- `cmd/codex_visuals.go` - `continueDetailCap`, `verificationStepDetailView`, `formatVerificationStepResultLine`, `renderContinueVerificationDetail`, `verificationStepDetailViewsFromTyped/Map`, `renderVerificationStepDetailLines`; `gateCheckDisplayNames`, `continueGateCheckNames`, `gateCheckDisplayName`, `gateCheckDetailView`, `renderContinueGateDetail`, `renderGateCheckDetailLines`; `criterionEvidenceView`, `criterionEvidenceDisplayState`, `renderCriterionEvidenceLines`, `criterionEvidenceViewsFromTyped/Map`, `renderCriterionEvidenceViewLines`; `specialistFindingView`, `specialistFindingViewIsEmpty`, `renderSpecialistFindingBlocks`, `writeSpecialistFindingCategory`, `specialistFindingViewsFromTyped/Map`; wired all four new render calls into `renderContinueVisual` and `renderContinueBlockedVisual`; fixed `renderPlanVisual`'s confidence dual-type switch.
- `cmd/continue_detail_render_test.go` (new) - all seven named tests: `TestVerificationDetailNamesEveryCheck`, `TestGateDetailNamesEveryGateAndItsFixHint`, `TestNestedDetailCapReportsWhatItOmitted`, `TestEvidenceLineNeverAppearsWithoutItsProof`, `TestSpecialistFindingsGetTheirOwnBlock`, `TestChatPathShowsChecksGatesAndEvidence`, `TestRestoredDetailIsPlainEnglish`.
- `cmd/testdata/golden_continue.txt` - regenerated via `-update-golden` for the new verification/gate detail lines (no evidence/specialist-finding lines appear in this fixture's data, so those sections stayed empty as expected).
- `cmd/testdata/golden_plan.txt` - regenerated via `-update-golden` for the now-correctly-rendering `Confidence: 76% overall` line.

## Decisions Made

See `key-decisions` in frontmatter: `renderCriterionEvidenceLines` sources `verification.Criteria`, not `task_evidence`; the plain-English internal-name scan is scoped to gate keys only; `gateCheckDisplayName`'s table is a map (not a switch) specifically so the test can derive from it; `verificationStepDisplayName` in `cmd/codex_continue.go` was left untouched (out of declared file scope); and the plan-confidence bug fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `renderPlanVisual`'s confidence line never rendering on any path**
- **Found during:** Task 1, per the orchestrator's note pointing at WINDOWS.md entry 5 (found by 198-04)
- **Issue:** `renderPlanVisual`'s confidence branch (`cmd/codex_visuals.go`) type-asserted `result["confidence"]` only to `map[string]interface{}`. Both `runCodexPlanWithOptions` and `runCodexPlanFinalize` always store `confidence` as a `codexPlanConfidence` struct, so the assertion never matched on either the direct or chat path — the `Confidence: N% overall` line simply never appeared, anywhere.
- **Fix:** Added a dual-type switch (`codexPlanConfidence` struct case first, `map[string]interface{}` fallback), mirroring the existing `planning_loop` struct/map switch immediately below it in the same function.
- **Files modified:** `cmd/codex_visuals.go`, `cmd/testdata/golden_plan.txt` (regenerated)
- **Verification:** `TestGoldenPlanVisualOutput` regenerated and passing without `-update-golden`; full `go test ./cmd/...` green.
- **Committed in:** `832d82b9` (Task 1 commit)
- **Windows ledger:** `.planning/WINDOWS.md` entry 5 marked `fixed`.

**2. [Rule 3 - Blocking] Golden fixtures needed regeneration for the new detail lines and the confidence fix**
- **Found during:** Task 1, after wiring the new render calls into `renderContinueVisual`/`renderContinueBlockedVisual` and fixing the confidence bug
- **Issue:** `TestGoldenContinueVisualOutput` and `TestGoldenPlanVisualOutput` diff byte-for-byte against checked-in golden files; both genuinely changed output on purpose (new verification/gate detail lines; the now-correctly-rendering confidence line).
- **Fix:** Regenerated both golden fixtures with `-update-golden`, inspected each diff to confirm it contained only the expected new lines.
- **Files modified:** `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt`
- **Verification:** Both golden tests pass without `-update-golden`; full `go test ./cmd/...` green (229s, all pass).
- **Committed in:** `832d82b9` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 pre-existing bug fixed per this plan's explicit instruction, 1 expected golden-fixture regeneration). **Impact:** Both were necessary for correctness and for the existing golden-fixture test suite to pass; no scope creep beyond `cmd/codex_visuals.go` and `cmd/continue_detail_render_test.go`.

## Issues Encountered

None beyond the deviations documented above. One implementation note: this executor initially refactored `emitVerificationStepFinish` in `cmd/codex_continue.go` to share `formatVerificationStepResultLine`, then reverted that edit on noticing `cmd/codex_continue.go` is outside this plan's declared `files_modified` (only `cmd/codex_visuals.go` and `cmd/continue_detail_render_test.go`). The formatter now lives solely in `codex_visuals.go`, duplicating (not sharing) `emitVerificationStepFinish`'s formatting logic — a deliberate, minor tradeoff to stay within declared file scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `workerMeasurementFigures` (198-05), `renderContinueVerificationDetail`/`renderContinueGateDetail`/`renderCriterionEvidenceLines`/`renderSpecialistFindingBlocks` (this plan) together restore the bulk of SHOW-02's continue-summary detail.
- `gateCheckDisplayNames` is the single place to add a plain-English translation if a new gate name is ever introduced; `continueGateCheckNames` (test-facing) picks it up automatically.
- Known residual gap (out of this plan's scope, not blocking): the external/wrapper continue lane's own gate/verification detail parity was not specifically re-verified in this plan beyond the existing `TestWrapperPathRendersSameCeremonyAsDirectPath` fixtures, which only exercise the in-process lane's typed structs and their JSON round-trip — not a live external-wrapper submission. No evidence found that this differs from prior plans' coverage.
- No blockers for subsequent plans in phase 198.

## Self-Check: PASSED

- Verified `cmd/codex_visuals.go`, `cmd/continue_detail_render_test.go`, `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt` all exist on disk with the expected content.
- Verified commits `832d82b9`, `1cc83123`, `235e186e` exist in `git log`.
- Re-ran the plan's full `<verification>` command: `go build ./...` and `go vet ./cmd` exit 0; `go test ./cmd -run 'TestVerificationDetailNamesEveryCheck|TestGateDetailNamesEveryGateAndItsFixHint|TestNestedDetailCapReportsWhatItOmitted|TestEvidenceLineNeverAppearsWithoutItsProof|TestSpecialistFindingsGetTheirOwnBlock|TestChatPathShowsChecksGatesAndEvidence|TestRestoredDetailIsPlainEnglish|TestWrapperPathRendersSameCeremonyAsDirectPath|TestHumanDisplaysUseHeadedSectionsNotMachineTables' -count=1` — ok.
- Confirmed `TestHumanDisplaysUseHeadedSectionsNotMachineTables` exits 0 with no new allowlist entry.
- Ran the full `go test ./cmd/...` suite twice (once mid-plan after Task 1+2, once at the end after all three commits): both green, ~230s each, zero failures.
- Confirmed `.planning/WINDOWS.md` entry 5 status is `fixed`.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*
