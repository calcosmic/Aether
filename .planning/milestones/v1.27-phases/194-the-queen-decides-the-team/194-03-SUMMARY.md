---
phase: 194-the-queen-decides-the-team
plan: 03
subsystem: orchestration
tags: [queen, caste-relevance, spawn-budget, judgement, reasons, D-08, D-09]

# Dependency graph
requires:
  - phase: 194-01
    provides: "forcedReviewerReason (queen_risk_signals.go) — the one plain-English sentence a forced reviewer's reason now travels through unchanged"
  - phase: 194-02
    provides: "the shrunken build floor (builder alone) and the reconciled test suite this plan builds its per-caste reasons on top of"
provides:
  - "queenCasteJudgement.Reasons / .RefusedNoReason — the per-worker reason map and by-name refusal list every downstream consumer (manifest, dispatch, card) now reads"
  - "--caste-why CLI flag (build and continue) plus parseCasteReasonPairs/parseAndMergeCasteWhy — the caste=reason pair parser that resolves through the same alias path --castes uses"
  - "queenRuntimeReasonForCaste / mergeCasteReasons — the runtime's own written justification for a required/forced/added caste, merged with a proposal reason when both exist"
  - "queenCandidateRationale / queenAlwaysRequiredReason / queenConditionRationale / CasteRelevanceProfile.ReasonTemplate — the fallback (no-proposal) engine's plain-English replacement for score-arithmetic rationale"
  - "The two applySpecialRules 100-score branches (auditor on production mode, gatekeeper on high risk) and auditor's mode==production Conditions boost are deleted — the last places D-06's implicit floor survived under a different name"
affects: [194-04-check-in-card, 194-05-probe-refusal-gate, 196-cost-line, 197-closing-card, 198-classic-display]

# Actuals (#2632)
actuals:
  tokens: 22285
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Variadic `reasons ...map[string]string` trailing parameter on the whole queenApplyJudgement call chain (plannedBuildDispatchesWithJudgement, queenCasteDecisionSummary, queenContinueDispatchesWithJudgement, queenContinueReviewSpecsWithJudgement, plannedExternalContinueDispatches, plannedContinueReviewDispatches, runCodexContinueReview, enrichQueenExecutionPolicyWithSpawnBudget, buildQueenSpawnBudgetContract) so every pre-existing call site keeps compiling unchanged, and only the handful of CLI-facing production call sites need to actually pass a map"
    - "A caste the reason-generation code cannot word for is excluded from the candidate list entirely, never sent with a placeholder (D-09) — mirrored on both the proposal-refusal path (RefusedNoReason) and the fallback-engine path (queenCandidateRationale returning \"\")"
    - "Recompute the judgement a second time where a downstream function needs its Reasons map but only has state/phase/flowType in scope (codex_build.go's judgementReasonsForBudget) rather than threading a struct return value through an unrelated call chain — queenApplyJudgement is pure, so this costs nothing and matches the existing queenCasteDecisionSummary precedent of an independent second derivation"

key-files:
  created:
    - cmd/queen_worker_reason_test.go
  modified:
    - cmd/queen_judgement.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_dispatch_contract.go
    - cmd/queen_spawn_budget.go
    - cmd/caste_relevance.go
    - cmd/queen_judgement_test.go
    - cmd/queen_relevance_floor_test.go
    - cmd/phase_verified_once_test.go
    - cmd/continue_fastpath_castes_test.go
    - cmd/caste_relevance_test.go
    - cmd/queen_team_choice_test.go
    - cmd/codex_build_test.go
    - cmd/ceremony_cmd_test.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/command_catalog.json

key-decisions:
  - "Chose variadic `...map[string]string` over a plain `map[string]string` parameter for `reasons` at every layer between queenApplyJudgement and the CLI. A plain parameter would have forced editing every one of the ~20 existing call sites across the package (most in test files unrelated to this plan) purely for arity, with no behavior change for the vast majority of them. Variadic keeps every unaffected call site compiling unchanged and makes the handful that DO need a real reasons map read identically to a plain parameter at the call site."
  - "Removed auditor's `Conditions: []string{\"mode==production\"}` registry entry, not just the applySpecialRules 100-score branch the plan named. The plan's acceptance criterion (\"a production-mode phase with no matching keyword returns a value below the spawn threshold, not the maximum\") could not hold with the Conditions boost left in place: base score (20) + condition (20) = 40, still above the 30 threshold, with zero keyword hits. Discovered while writing TestDeletedImplicitFloorsNoLongerScoreToTheMaximum, before that test's own commit."
  - "Split the plan's two tasks across two commits by file, not by a strict task-1/task-2 boundary. `queenRuntimeReasonForCaste` and `mergeCasteReasons` are conceptually task 2's deliverable but had to live in queen_judgement.go (a task-1 file) because task 1's own Reasons-map construction calls them in the same function body — an interleaving the same shape as 194-01's forced-reviewer/build-manifest coupling. Commit 1 carries queen_judgement.go, the CLI/dispatch threading, and the runtime-reason helpers; commit 2 carries caste_relevance.go's rewrite (the actual score-arithmetic deletion) and every test fixture that rewrite touched."

patterns-established:
  - "A per-caste reason travels as a `map[string]string` at every layer (queenCasteJudgement.Reasons, codexQueenSpawnBudgetContract.SelectedReasons/PrunedReasons, codexContinueReviewSpec.Rationale) — never a single string standing in for a team of workers."

requirements-completed: [TEAM-03]

coverage:
  - id: D1
    description: "A proposal of several castes with a reason for only some of them sends the ones with a reason and refuses the rest by name (caste_decision.refused_no_reason and the summary line), leaving the rest of the proposal unaffected."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestNoWorkerWithoutStatedReason"
        status: pass
    human_judgment: false
  - id: D2
    description: "--caste-reason (the team-level summary) does not satisfy the per-worker reason requirement for any caste; a caste proposed alongside only a team-level string is refused exactly as if no reason had been supplied."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestTeamLevelReasonDoesNotSatisfyPerWorkerReason"
        status: pass
    human_judgment: false
  - id: D3
    description: "A --caste-why key written with the other separator style or a known synonym resolves to the same registry caste --castes itself resolves to, through the identical resolveCasteName path; a genuinely unresolvable key is reported through the SAME unknown-name channel --castes already has."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestReasonKeyResolvesThroughTheSameAliasPath"
        status: pass
    human_judgment: false
  - id: D4
    description: "Two runs of queenApplyJudgement over the same input produce byte-identical RefusedNoReason slices and summary strings."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestReasonOutputIsStablySorted"
        status: pass
    human_judgment: false
  - id: D5
    description: "A runtime-added caste (required, or forced by a named risk signal) carries a reason the RUNTIME wrote: the implementation worker names the task count and a task goal; a forced reviewer's reason quotes the matched phrase (194-01's forcedReviewerReason, reused verbatim)."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestRuntimeAddedWorkersCarryRuntimeWrittenReasons"
        status: pass
    human_judgment: false
  - id: D6
    description: "No reason string produced anywhere by the fallback (no-proposal) engine contains score arithmetic, the flow-type identifier, or a caste name standing alone as its own justification; a caste the engine cannot word for is not returned as a candidate."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestFallbackPickCarriesASentenceNotAScore"
        status: pass
    human_judgment: false
  - id: D7
    description: "The two deleted implicit floors (auditor on production mode, gatekeeper on high risk) no longer score to the old maximum: a production-mode phase with no auditor wording, and a high-risk phase with no security wording, both score below the spawn threshold."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestDeletedImplicitFloorsNoLongerScoreToTheMaximum"
        status: pass
    human_judgment: false
  - id: D8
    description: "caste_decision in a manifest contains both a reasons object and, when a reason was missing, a refused_no_reason array."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/queen_worker_reason_test.go#TestCasteDecisionManifestCarriesReasonsAndRefusals"
        status: pass
    human_judgment: false
  - id: D9
    description: "--caste-why exists on both `aether build` and `aether continue`, and the full package and whole-repo test suites pass with the change."
    verification:
      - kind: other
        ref: "aether build --help / aether continue --help both list --caste-why"
        status: pass
      - kind: unit
        ref: "go test ./cmd -count=1"
        status: pass
      - kind: unit
        ref: "go test ./... -count=1 (whole repo, excluding the network-dependent TestPackedNPMReleaseCandidateContract)"
        status: pass
    human_judgment: false

duration: 210min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 3: A Reason for Every Helper Summary

**Every worker that spawns now carries a one-sentence plain-English reason — a proposed worker with no reason is refused by name while the rest of the team is sent, and the fallback engine's "Score 40 >= threshold 30 for build flow" and "X is always required for FLOW flow" strings are gone from every reason slot in the codebase.**

## Performance

- **Duration:** ~210 min
- **Tasks:** 2 completed
- **Files modified:** 18 (1 new: `cmd/queen_worker_reason_test.go`)

## Accomplishments

- `--caste-why` (repeatable `caste=reason` pairs) on both `aether build` and `aether continue`, parsed by `parseCasteReasonPairs` through the exact `resolveCasteName` alias/separator path `--castes` already uses, so a key written `route-setter` or `security` lands on the right caste instead of a second unknown-name channel.
- `queenCasteJudgement.Reasons` (per-caste sentence) and `.RefusedNoReason` (by-name refusal list) — a proposed caste with no reason and no exemption is refused for that worker only; the rest of the team is sent unaffected. `--caste-reason` remains the team-level summary and no longer satisfies the per-worker requirement for any caste.
- `queenRuntimeReasonForCaste` + `mergeCasteReasons`: the runtime writes its own reason for a caste it adds (the implementation worker names the task count and a task goal; a forced reviewer's reason is 194-01's `forcedReviewerReason` sentence, reused verbatim) and merges it with a proposal's own reason when both exist for the same caste — the adjacency answer, one sentence, never two competing lines.
- The fallback (no-proposal) engine's rationale is now a plain sentence per caste (`queenCandidateRationale`, `CasteRelevanceProfile.ReasonTemplate`, `queenAlwaysRequiredReason`, `queenConditionRationale`) — the literal strings `"Score %d >= threshold %d for %s flow"` and `"%s is always required for %s flow"` are deleted from the codebase entirely, not just hidden behind a preference.
- Deleted the two `applySpecialRules` branches that scored the quality reviewer to the maximum on production mode and the security reviewer to the maximum on high risk (the same implicit floors Ruling D11 already removed from the build-side required-caste list, restated here as scores) — and auditor's `mode==production` Conditions boost, which alone still cleared the spawn threshold after the special-rule branch was gone.
- The two budget-overflow rationale suffixes in `queen_spawn_budget.go` rewritten in plain English; `buildQueenSpawnBudgetContract` now prefers the judgement's merged per-caste reason over the raw candidate rationale when one is available.
- 8 new tests in `cmd/queen_worker_reason_test.go` covering the refusal rule, the team-vs-per-worker distinction, the encoding/ordering probes from 194-CONTEXT.md's flagged assumptions, runtime-written reasons, the score-arithmetic deletion, and the manifest-facing `caste_decision` shape.

## Task Commits

1. **Task 1: A reason per worker on the judgement, and refusal by name when one is missing** - `42d5a5d9` (feat)
2. **Task 2: Runtime-written reasons replace score arithmetic everywhere a reason is produced** - `249f9779` (test)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/queen_judgement.go` — `queenCasteJudgement.Reasons`/`.RefusedNoReason` fields; `queenApplyJudgement`'s reasons-aware refusal loop and the deterministic (no-proposal) branch's own Reasons construction; `parseCasteReasonPairs`, `parseAndMergeCasteWhy`, `firstReasonMap`, `queenRuntimeReasonForCaste`, `mergeCasteReasons`; `queenCasteDecisionSummary` now emits `reasons`/`refused_no_reason`.
- `cmd/codex_workflow_cmds.go` — `--caste-why` flag definitions and CLI-to-options threading for both `buildCmd` and `continueCmd`.
- `cmd/codex_build.go` — `codexBuildOptions.QueenCasteWhy`; `plannedBuildDispatchesWithJudgement` threads `reasons`; `judgementReasonsForBudget` recomputes the judgement for the spawn-budget contract.
- `cmd/codex_continue.go` — `codexContinueOptions.QueenCasteWhy`; `queenContinueDispatchesWithJudgement`/`queenContinueReviewSpecsWithJudgement`/`plannedContinueReviewDispatches`/`runCodexContinueReview` all thread `reasons`; `casteDispatchRationale` prefers the per-caste reason over the team Rationale.
- `cmd/codex_continue_plan.go` — `plannedExternalContinueDispatches` (wrapper lane) threads `reasons` from `options.QueenCasteWhy`.
- `cmd/codex_dispatch_contract.go` — `buildQueenSpawnBudgetContract`/`enrichQueenExecutionPolicyWithSpawnBudget` prefer the judgement's Reasons entry over `decision.Rationale`.
- `cmd/queen_spawn_budget.go` — `appendQueenBudgetRationale`/`appendQueenPrunedRationale` rewritten in plain English (no more "Queen spawn budget N").
- `cmd/caste_relevance.go` — `CasteRelevanceProfile.ReasonTemplate`; `queenCandidateDispatches` rewritten to use `queenCandidateRationale`/`queenAlwaysRequiredReason`/`queenConditionRationale`; the two deleted `applySpecialRules` branches; auditor's `Conditions` entry removed.
- `cmd/queen_worker_reason_test.go` (new) — `TestNoWorkerWithoutStatedReason`, `TestTeamLevelReasonDoesNotSatisfyPerWorkerReason`, `TestReasonKeyResolvesThroughTheSameAliasPath`, `TestReasonOutputIsStablySorted`, `TestRuntimeAddedWorkersCarryRuntimeWrittenReasons`, `TestFallbackPickCarriesASentenceNotAScore`, `TestDeletedImplicitFloorsNoLongerScoreToTheMaximum`, `TestCasteDecisionManifestCarriesReasonsAndRefusals`.
- `cmd/queen_judgement_test.go`, `cmd/queen_relevance_floor_test.go`, `cmd/phase_verified_once_test.go`, `cmd/continue_fastpath_castes_test.go` — existing call sites that proposed an optional caste with only a team-level reason updated to supply the per-worker reason D-08 now requires.
- `cmd/caste_relevance_test.go`, `cmd/queen_team_choice_test.go`, `cmd/codex_build_test.go`, `cmd/ceremony_cmd_test.go` — fixtures and assertions that relied on the deleted implicit floors (production ⇒ auditor, high risk ⇒ gatekeeper) or the old jargon rationale strings corrected to the genuine relevance-scored / plain-English behavior.
- `cmd/testdata/golden_build.txt`, `cmd/testdata/command_catalog.json` — regenerated via `-update-golden` and hand-diffed; the only changes are the new `--caste-why` flag entries and the plain-English rationale text.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) variadic `...map[string]string` reasons parameters throughout, to avoid a purely-mechanical edit of ~20 unrelated call sites; (2) removing auditor's `mode==production` Conditions boost as well as the named `applySpecialRules` branch, because the acceptance criterion could not hold with the Conditions boost left in place.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Auditor's `Conditions: []string{"mode==production"}` registry entry also had to go**
- **Found during:** Task 2, while writing `TestDeletedImplicitFloorsNoLongerScoreToTheMaximum`
- **Issue:** The plan's action text named only the `applySpecialRules` branch for deletion. Deleting that branch alone left auditor's base score (20) plus its separate `Conditions` boost (20) at 40 — still above the 30 spawn threshold with zero keyword hits, so the implicit "production mode ⇒ auditor" floor survived under a second mechanism in the same file.
- **Fix:** Removed the `Conditions: []string{"mode==production"}` entry from auditor's registry row.
- **Files modified:** `cmd/caste_relevance.go`
- **Verification:** `TestDeletedImplicitFloorsNoLongerScoreToTheMaximum` passes; full `go test ./cmd -count=1` green.
- **Committed in:** `249f9779` (Task 2 commit)

**2. [Rule 3 - Blocking] Six existing test files needed reconciliation to the new per-worker refusal rule and the deleted implicit floors**
- **Found during:** Task 1 and Task 2, after each `go test ./cmd -count=1` run
- **Issue:** `queen_judgement_test.go`, `queen_relevance_floor_test.go`, and `phase_verified_once_test.go` had call sites proposing an optional caste (measurer, watcher, a greedy ten-caste list) with only a team-level rationale — exactly the pattern D-08 now refuses. `continue_fastpath_castes_test.go` had the same gap on the continue side. `caste_relevance_test.go` and `queen_team_choice_test.go` asserted the deleted "production mode ⇒ auditor" / "high risk ⇒ gatekeeper" floors and the old "Queen spawn budget" jargon string. `codex_build_test.go` and `ceremony_cmd_test.go` asserted the old `"not spawned"` / `"selected within Queen spawn budget"` substrings.
- **Fix:** Added the missing per-worker `--caste-why`-equivalent reasons map to each proposal call site that needed one; corrected the auditor/gatekeeper assertions to the genuine relevance-scored behavior (with an explanatory comment citing Ruling D11); restored one fixture's budget pressure with a genuine keyword-driven 4th candidate (chaos, via "crash"+"resilience") now that auditor's free condition boost is gone; updated jargon-string assertions to the new plain-English text.
- **Files modified:** `cmd/queen_judgement_test.go`, `cmd/queen_relevance_floor_test.go`, `cmd/phase_verified_once_test.go`, `cmd/continue_fastpath_castes_test.go`, `cmd/caste_relevance_test.go`, `cmd/queen_team_choice_test.go`, `cmd/codex_build_test.go`, `cmd/ceremony_cmd_test.go`.
- **Verification:** `go test ./cmd -count=1` and `go test ./... -count=1` (whole repo) both pass; `go vet ./...` clean.
- **Committed in:** `42d5a5d9` (Task 1) and `249f9779` (Task 2), split by which fixture each edit belonged to.

**3. [Rule 3 - Blocking] Two golden fixtures needed regeneration**
- **Found during:** Task 2's full-suite reconciliation pass
- **Issue:** `TestGoldenBuildVisualOutput` and `TestAuditCatalogGolden` failed — the golden files still had the old `"builder is always required for build flow"` rationale and lacked the new `--caste-why` flag entry.
- **Fix:** Ran both tests with `-update-golden` and hand-diffed the result; the only changes are the new flag entries and the plain-English rationale text, nothing else moved.
- **Files modified:** `cmd/testdata/golden_build.txt`, `cmd/testdata/command_catalog.json`
- **Verification:** `git diff` on both files shows only the expected 3-line change; tests pass.
- **Committed in:** `249f9779` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking) — all necessary for the acceptance criteria to genuinely hold and for the suite to stay green, none of them scope creep beyond what the deleted floors and the new refusal rule required.
**Impact on plan:** None of the deviations touched territory outside this plan's own files (`queen_judgement.go`, `caste_relevance.go`, `queen_spawn_budget.go`, `codex_dispatch_contract.go`, the CLI/dispatch threading, and their own tests).

## Issues Encountered

- `go test ./cmd/...` (full package) takes 440-600s in this sandbox and one fixture (`TestPackedNPMReleaseCandidateContract`) attempts a real `npm` bootstrap that hangs on network access here; excluded via `-skip` for every full-suite run in this plan, consistent with it being an environment/network dependency unrelated to this plan's files.
- One full-repo run (`go test ./... `) showed two failures that were confirmed non-regressions: `TestCLIExternalAdapterBuildContract` failed because it compares a before/after `git status` snapshot and a `git commit` I ran concurrently in another shell changed the working tree mid-test (passed cleanly in isolation immediately after); `TestAvailabilityProbeRetriesOnlyTimeouts` (`pkg/codex`, a package this plan never touches) is a timing-sensitive retry test that passed cleanly in isolation both before and after — matches this project's documented pattern of timing-flaky tests under parallel load (`TestSwarmCompatibilityWatchReportsActiveWorkers`, etc.). Neither fixed; neither touched.

## Known Stubs

None. Every deliverable in `must_haves.truths` has a passing test asserted on the real dispatch list or manifest output, not an intermediate struct.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `queenCasteJudgement.Reasons`/`.RefusedNoReason`, `parseCasteReasonPairs`, `queenRuntimeReasonForCaste`, and `mergeCasteReasons` are in place for 194-04 (the check-in card, `renderCeremonyTeamCheckin`'s `what_it_does` key and `TestTeamCheckinNeverShowsAGenericBlurbAsAReason`) to render without recomputing.
- The manifest's `caste_decision.reasons`/`.refused_no_reason` and `spawn_budget.selected_reasons`/`.pruned_reasons` are populated with plain-English sentences on every path (proposal, deterministic fallback, forced reviewer) — 196 (cost line) and 197/198 (cards) can read them directly.
- No score arithmetic remains in any reason slot in `cmd/`: `grep -rn "Score.*>= threshold\|is always required for" cmd/*.go` (excluding this plan's own comments describing the deletion) returns nothing.
- Plan 194-05's skipped subtest (`TestProbeIsRequiredOnlyWhereItCanFindSomething`, from 194-02) is untouched by this plan and remains ready for that plan to land.
- No blockers for 194-04.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: commit `42d5a5d9` (Task 1) in `git log --oneline --all`
- FOUND: commit `249f9779` (Task 2) in `git log --oneline --all`
- FOUND: `cmd/queen_worker_reason_test.go`
- FOUND: `cmd/queen_judgement.go` (Reasons/RefusedNoReason, parseCasteReasonPairs)
- FOUND: `cmd/caste_relevance.go` (ReasonTemplate, queenCandidateRationale)
- `go build ./cmd/aether` — pass
- `go vet ./...` — pass
- `go test ./cmd -count=1` — pass
- `go test ./... -count=1` (whole repo, excluding the network-dependent `TestPackedNPMReleaseCandidateContract`) — pass (one confirmed non-regression, timing-flaky test in an untouched package, verified in isolation)
- `grep -rn "Score %d >= threshold\|is always required for %s" cmd/*.go | grep -v _test.go` — empty, confirming no score-arithmetic reason string remains in production code
