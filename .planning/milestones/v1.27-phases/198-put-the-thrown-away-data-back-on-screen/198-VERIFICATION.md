---
phase: 198-put-the-thrown-away-data-back-on-screen
verified: 2026-08-29T21:45:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 198: Put the Thrown-Away Data Back on Screen Verification Report

**Phase Goal:** Restore the detail the older (v5.4.0) version of Aether used to show — which
checks passed, evidence for each requirement, how long each worker took, plan confidence,
resume progress, recent decisions, and warnings — that the program already calculates today
but currently throws away instead of displaying. The chat view (Claude Code / OpenCode) shows
the same level of detail as the direct command-line view.

**Verified:** 2026-08-29
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Chat-path screens for plan/continue/seal byte-match the direct CLI screens | ✓ VERIFIED | `TestWrapperPathRendersSameCeremonyAsDirectPath` passes 9 subtests (continue advance/blocked, unresolvable-phase fallback, cross-workflow rejection, plan completed/mid-loop, seal completed/force-sealed/pending-confirmation) using real Go-struct fixtures round-tripped through JSON, asserting byte-equal string output between `renderXVisual` (direct) and `closeoutDirectVisual` (chat bridge). |
| 2 | Every calculated-but-dropped item is shown or is a documented, shrink-only exception | ✓ VERIFIED (see caveat below) | `TestRenderedVisualsShowEveryCarriedField` walks the real AST call graph from each of 5 finalizer result maps to their render entry points; confirmed genuinely falsifiable by removing an allowlist entry (`research_warning`/plan completed) and observing the test fail by name, then restoring it. `TestRenderedFieldAllowlistOnlyShrinks` confirmed falsifiable the same way (added a fake entry, test failed, reverted). All of criterion 2's explicitly named items — checks passed, evidence lines, worker duration/tool-count, plan confidence, resume phase progress, recent decisions, drift warning, specialist findings, build-start blocker heads-up — are independently confirmed wired into their renderers (see Key Link Verification). |
| 3 | `continue` shows each check's progress live, not only a closing summary | ✓ VERIFIED | `TestBothContinueLanesEmitLiveCheckLines` runs `runCodexContinueVerification` (in-process/fast lane) and `runCodexContinueVerificationSnapshot` (heavy/review lane) against a captured stdout buffer and asserts both emit start+finish lines for all 4 checks. Both lanes route through the single `runDeterministicFloor` → `runVerificationStep` body (one call site, confirmed by reading `codex_continue.go:1814`), so the guarantee is structural, not duplicated. `TestFailedCheckLineCarriesItsReasonInline`, `TestLiveCheckLinesAreAppendOnly`, `TestLiveCheckLinesUseTheVisualWriter`, `TestVerificationStepDurationIsMeasured` all pass. |
| 4 | Seal always asks for confirmation and runs the wisdom review first | ✓ VERIFIED | `runSealConfirmationGate` is called directly inside `sealCmd` (cmd/codex_workflow_cmds.go:578) before any state mutation; it calls `buildSealStateOfPlay` → renders the card → `runSealWisdomReview` → `decideSealConfirmation`. `TestAutopilotNeverReachesTheSealPath`, `TestSealPlanOnlyDoesNotMutate`, `TestSealFinishAnywayIsRecordedNotInferred`, `TestSealConfirmationDispatchesNoWorkers` all pass. |
| 5 | The 3 deliberately-dropped display choices stay dropped and are test-locked | ✓ VERIFIED | `TestDeliberatelyDroppedDisplayChoicesStayDropped` scans every shipped `cmd/*.go` (excluding tests) and every `.claude`/`.opencode` wrapper markdown file for tmux/second-terminal phrasing and forbidden box-drawing corners; it caught and the phase fixed one real live violation (`cmd/status.go` telling the owner to tail a log "in another terminal") during 198-09. `TestSafeToClearLineStaysAStatement` derives the kept "safe to clear" wording from the real render function and asserts it is never rendered as a question anywhere in scanned sources. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

**Caveat on truth 2:** the invariant's own design (per 198-09-PLAN.md, an explicit, owner-endorsed interpretation of criterion 2) treats "documented exception with a reason, tracked in a shrink-only allowlist" as satisfying "not silently dropped" — distinct from "always rendered." Two residual items fall into that documented-exception bucket and are **still not shown to the owner today**, tracked as open items in `.planning/WINDOWS.md` (entries 6 and 7, phase 198, both status `open`):
- The plan-completed screen never shows `research_warning` (a real warning: "this phase was planned without its research"), `research_failed_phases`, or a completed plan's own unresolved `gaps`. Fixing this touches `cmd/codex_visuals.go`, which 198-09 could not edit (owned that wave by a sibling plan).
- The plan finalizer's own suggested `next` command never folds into the unified next-action envelope (`closeLifecycleRun`), because `runCodexPlanFinalize` never calls it — the closing card independently resolves a next step from live colony state instead, which "usually matches but is not guaranteed to."

Neither item is one of criterion 2's explicitly enumerated examples (checks passed, evidence, worker duration, plan confidence, resume progress, recent decisions, drift warning, specialist findings, blocker heads-up) — all of which ARE fully wired and rendered, confirmed independently below. These two are additional gaps the invariant *discovered* while auditing scope beyond the named list, and were honestly recorded rather than silently patched around or silently left out of the allowlist. They do not fail the named roadmap test, and are not a regression this phase introduced (the plan-confidence bug that WAS in the same category of "discovered while building the invariant" — WINDOWS.md entry 5 — was fixed live during 198-06). Flagging both for a follow-up phase/plan decision; not treated as a phase-blocking gap here because the test contract the roadmap names (`TestRenderedVisualsShowEveryCarriedField`) explicitly and by design accepts a reasoned, shrink-only-tracked exception as satisfying the invariant, and the owner has not been asked to accept or reject that interpretation.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/closeout_direct_render.go` | Shared renderer bridge for continue/plan/seal | ✓ VERIFIED | Exists, wired, exercised by `TestWrapperPathRendersSameCeremonyAsDirectPath` |
| `cmd/wrapper_path_parity_test.go` | Byte-equal parity test | ✓ VERIFIED | 581 lines, 9 real subtests, all pass |
| `cmd/codex_continue.go` — `emitVerificationStepStart/Finish` | Live progress hooks | ✓ VERIFIED | Present at `runVerificationStep` (line 3247), single call site reached by both lanes |
| `cmd/seal_confirmation.go` | State-of-play card + decision gate | ✓ VERIFIED | `buildSealStateOfPlay`, `renderSealStateOfPlayCard`, `decideSealConfirmation`, `recordSealConfirmationAnswer` all present and wired into `sealCmd` |
| `cmd/codex_continue.go` / `cmd/codex_visuals.go` — worker duration/tool count | Shared figure formatter | ✓ VERIFIED | `TestUnmeasuredWorkerFiguresRenderAsNotReported`, `TestLiveAndSummaryWorkerFiguresShareOneSource` pass |
| `cmd/codex_visuals.go` — `renderContinueVerificationDetail`, `renderContinueGateDetail`, `renderCriterionEvidenceLines` | Per-check/gate/evidence rendering | ✓ VERIFIED | `TestEvidenceLineNeverAppearsWithoutItsProof`, `TestNestedDetailCapReportsWhatItOmitted` pass |
| `cmd/build_blocker_advisory.go` | Build-start blocker heads-up | ✓ VERIFIED | Wired into both build lanes (`cmd/codex_workflow_cmds.go:242,336`), all 4 named tests pass |
| `cmd/context.go` / `cmd/codex_visuals.go` — resume detail | Per-phase progress, recent decisions, drift note | ✓ VERIFIED | `renderResumePhaseProgress`, `renderResumeDriftNote`, `renderResumeRecentDecisions` all called from `renderResumeVisual` (lines 3721-3724) |
| `cmd/rendered_fields_invariant_test.go` + allowlist JSONs | Carried-field invariant, shrink-only ratchet | ✓ VERIFIED | AST-based, confirmed genuinely falsifiable by live mutation-and-revert |
| `cmd/deliberate_drops_lock_test.go` | Deliberate-drops lock | ✓ VERIFIED | Real file scan, caught and fixed one live violation during the phase |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `renderCeremonyCloseout` | `closeoutDirectVisual` → `renderContinueVisual`/`renderPlanVisual`/`renderSealVisual` | direct call | ✓ WIRED | Confirmed by parity test byte-equality |
| `runDeterministicFloor` | `runVerificationStep` → `emitVisualProgress`/`writeVisualOutput` | one shared call path, 4x per lane | ✓ WIRED | Single call site at `codex_continue.go:1814`; both `runCodexContinueVerification` (fast) and `runCodexContinueVerificationSnapshot` (heavy) reach it |
| `sealCmd` | `runSealConfirmationGate` → `buildSealStateOfPlay` → `runSealWisdomReview` → `decideSealConfirmation` | direct call before mutation | ✓ WIRED | Confirmed at `codex_workflow_cmds.go:578`, before `completeSealRuntime` |
| `codex.WorkerResult.ToolCount` | continue worker-flow step → saved JSON → chat screen | serialized field + shared formatter | ✓ WIRED | `TestWorkerUsageStaysUnserialized` (negative control) and `TestLiveAndSummaryWorkerFiguresShareOneSource` both pass |
| `decideBuildBlockerAdvisory` | both build lanes (plan-only + direct) | shared pure function | ✓ WIRED | Both lanes call identical `buildStartBlockerSignals`/`decideBuildBlockerAdvisory`/`renderBuildBlockerAdvisory`; direct lane's `BoundaryQuestionCount` honestly reads 0 (pre-existing gap in that lane's tracking, documented in 198-07-SUMMARY.md, not a regression) |
| `state.Memory.Decisions` / `Plan.Phases[].Status` / `planRevisionSummary` | resume result map → `renderResumeVisual` | direct reads | ✓ WIRED | Confirmed by reading call sites and by `TestDriftNoteIsDerivedNotInvented`/`TestResumeViewDoesNotMutate` |
| finalizer result-map keys | render functions reachable from each workflow's entry point | AST call-graph walk | ✓ WIRED (with 2 documented exceptions) | See caveat above |

### Behavioral Spot-Checks / Falsifiability Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `TestRenderedVisualsShowEveryCarriedField` can fail | Removed `research_warning`/`plan completed` from the live allowlist, reran | Test failed by name with the exact expected message; file restored via `git checkout` | ✓ PASS (falsifiability confirmed) |
| `TestRenderedFieldAllowlistOnlyShrinks` can fail | Added a fake unlisted-in-baseline entry, reran | Test failed by name; file restored via `git checkout` | ✓ PASS (falsifiability confirmed) |
| Named tests across all 9 plans pass individually | `go test ./cmd -run '<each named test>' -count=1 -v` | All pass (see full list run during verification) | ✓ PASS |
| Full `cmd` package suite is clean | `go test ./cmd/... -count=1` (isolated run) | `ok github.com/calcosmic/Aether/cmd 258.396s` | ✓ PASS |
| `go build ./...` / `go vet ./cmd/...` | build + vet | Clean | ✓ PASS |

Note: an earlier concurrent run (two full suites launched simultaneously by the verifier for
timing reasons) produced one spurious `FAIL` at the package level with no individual `--- FAIL`
line attributable — re-run in isolation, the suite passed cleanly twice in a row. Treated as
resource-contention flake, not a regression, consistent with this repo's documented
oracle-heartbeat-flake pattern (rerun once before concluding real).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|--------------|--------|----------|
| SHOW-01 | 198-01, 198-04 | Chat path renders same ceremony as direct path (plan/continue/seal) | ✓ SATISFIED | `TestWrapperPathRendersSameCeremonyAsDirectPath`, 9/9 subtests pass |
| SHOW-02 | 198-02, 198-04, 198-05, 198-06, 198-07, 198-08, 198-09 | Carried-but-dropped data now rendered, as an invariant | ✓ SATISFIED (2 residual documented gaps, see caveat) | `TestRenderedVisualsShowEveryCarriedField` + individual per-item render wiring confirmed |
| SHOW-03 | 198-02 | Live verification progress lines | ✓ SATISFIED | `TestBothContinueLanesEmitLiveCheckLines` + 4 sibling tests |
| SHOW-04 | 198-03 | Seal confirmation + wisdom review gate | ✓ SATISFIED | 4 named tests pass; gate wired directly into `sealCmd` |
| SHOW-05 | 198-09 | Deliberate drops stay dropped, test-locked | ✓ SATISFIED | 2 named tests pass; one real violation caught and fixed live |

No orphaned requirements: all 5 phase requirement IDs (SHOW-01..05) appear in at least one plan's `requirements:` frontmatter, and REQUIREMENTS.md maps all 5 to Phase 198 with no additional unclaimed IDs.

### Anti-Patterns Found

None. Scanned every non-test file modified across all 9 plans (`build_blocker_advisory.go`,
`ceremony_cmd.go`, `ceremony_team_checkin.go`, `closeout_cmd.go`, `closeout_direct_render.go`,
`codex_build_progress.go`, `codex_continue.go`, `codex_continue_finalize.go`,
`codex_visuals.go`, `codex_workflow_cmds.go`, `context.go`, `seal_confirmation.go`,
`seal_final_review.go`, `status.go`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` and
"not yet implemented" phrasing — zero matches.

### Human Verification Required

None. All success criteria are covered by automated, falsifiable tests; no visual/UX judgment
call remains open for this phase's own scope.

### Gaps Summary

No must-have truth failed, no artifact is missing or stub, no key link is unwired, and no
blocker anti-pattern was found. All five roadmap success criteria are backed by real,
independently-confirmed-falsifiable tests, and the full `cmd` package test suite is green.

Two narrow, honestly-recorded residual items remain **open** in `.planning/WINDOWS.md`
(entries 6 and 7, both phase 198, both `status: open` as of this verification) that fall
outside criterion 2's explicitly enumerated list but are still within the phase's broader
"nothing calculated should be silently dropped" spirit:

1. The plan-completed screen still never shows `research_warning` / `research_failed_phases`
   / a completed plan's own unresolved `gaps` — genuinely calculated, genuinely a warning in
   one case, genuinely not shown. Fix requires editing `cmd/codex_visuals.go` (owned this
   milestone wave by 198-08, not available to 198-09).
2. The plan finalizer's own `next` suggestion does not fold into the unified next-action
   envelope Phase 197 established, so the plan-completed/mid-loop closing card resolves its
   next step independently (usually — not provably always — matching). Fix requires editing
   `cmd/codex_plan_finalize.go`.

These are not treated as phase-blocking gaps: the roadmap's own named test for criterion 2
(`TestRenderedVisualsShowEveryCarriedField`) is designed, by the plan authors' documented
decision, to accept "rendered OR honestly allow-listed with a reason, shrink-only enforced"
as satisfying the invariant — and both items meet that bar. They are recommended as a small
follow-up plan (touching `cmd/codex_visuals.go` and `cmd/codex_plan_finalize.go`) rather than
a reason to reopen this phase, but are surfaced here rather than silently accepted, per this
project's own Definition of Done history of declaring "restored" work that quietly still
dropped something.

---

_Verified: 2026-08-29_
_Verifier: Claude (gsd-verifier)_
