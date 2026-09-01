---
phase: 194-the-queen-decides-the-team
plan: 01
subsystem: orchestration
tags: [queen, spawn-budget, continue, build-manifest, gatekeeper, auditor, risk-signals]

# Dependency graph
requires:
  - phase: 193-free-checks-are-the-floor
    provides: the single deterministic verification pass (runDeterministicFloor) and the "verify once" continue boundary this plan attaches a forced reviewer to
provides:
  - "queenRiskSignalTable: the exact five named-risk signals (credentials/auth, payments, release sign-off, data deletion, database migration) and the only place a reviewer can be forced from"
  - "codexBuildManifest.ForcedReviewers: the build-time record continue reads instead of re-deriving independently"
  - "queenForcedContinueReviewers / unionForcedContinueReviewers: the continue-side union that survives budget trims"
affects: [194-02-shrink-the-build-floor, 194-03-a-reason-for-every-helper, 194-08-close-windows-1, 196-cost-line, 197-closing-card]

# Actuals (#2632)
actuals:
  tokens: 9731
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Word-boundary phrase matching with 'longest match wins' so a specific phrase (\"password reset\") is quoted instead of a shorter one it contains (\"password\")"
    - "Forced-reviewer Rationale is tagged with a stable 'this touches ...' prefix so tests and the check-in card can distinguish a signal-forced dispatch from a keyword/mode-scored one sharing the same caste name"
    - "One derivation (build), one boundary (continue) for a cross-process record — continue always reads the build's record first and only re-derives as a fallback"

key-files:
  created:
    - cmd/queen_risk_signals.go
    - cmd/queen_forced_reviewer_test.go
  modified:
    - cmd/queen_spawn_budget.go
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/floor_unskippable_test.go
    - cmd/queen_judgement_test.go
    - .aether/schemas/completion-packet.schema.json

key-decisions:
  - "The longest matching phrase per signal wins (not the first), so 'password reset' is quoted back instead of the shorter 'password' it contains — the plan's own acceptance criterion required the exact phrase quoted."
  - "A forced reviewer's Rationale always begins with the fixed prefix 'this touches ...' so tests (and later the check-in card) can tell a signal-forced dispatch apart from a caste the pre-existing keyword/mode scoring engine independently selected under the same name — without this, 'auditor is present' cannot distinguish 'forced by a named signal' from 'the untouched legacy production-mode floor also drew it'."
  - "Reused the existing joinWithAnd helper (cmd/codex_project_docs.go) for the reason sentence's list grammar instead of writing a second copy."

patterns-established:
  - "queen_risk_signals.go is the single file owning the named-risk vocabulary, its matcher, its collapse rule, and its cross-boundary record conversion — future signal-table changes (194-06's PathPatterns) have one file to edit."

requirements-completed: [TEAM-02]

coverage:
  - id: D1
    description: "A phase whose wording names one of the five signals forces exactly the mapped reviewer at the continue step, with a reason naming the signal and quoting the matched phrase."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestReviewerForcedOnlyByNamedRisk"
        status: pass
    human_judgment: false
  - id: D2
    description: "A CSV-export phase in production mode forces no signal-based reviewer; a password-reset phase forces the security reviewer with a stated reason."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestReviewerForcedOnlyByNamedRisk"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two signals mapping to the same caste (a login and a refund) collapse into ONE dispatch naming both signals, never two dispatches of the same caste."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestTwoSignalsOneCasteCollapseToOneDispatch"
        status: pass
    human_judgment: false
  - id: D4
    description: "An empty phase (no name, description or tasks) forces no reviewer."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestEmptyPhaseForcesNoReviewer"
        status: pass
    human_judgment: false
  - id: D5
    description: "Signal matching is case-insensitive and word-boundary anchored: 'tokenizer' and 'token bucket' never match the credentials signal; 'api key' does."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestSignalMatchingIsWordBounded"
        status: pass
    human_judgment: false
  - id: D6
    description: "The forced set is derived once at build, recorded on the build manifest (forced_reviewers), and continue reads that same record rather than re-deriving independently — the build never dispatches it itself."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestForcedReviewerCrossesTheBuildContinueBoundary"
        status: pass
      - kind: manual_procedural
        ref: "aether build 1 --plan-only against a credentials-wording fixture phase; dispatch_manifest.forced_reviewers contains one gatekeeper entry"
        status: pass
    human_judgment: false

duration: 75min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 1: One Named-Risk Table, Recorded Once, Dispatched Once Summary

**Five named risk signals (credentials/auth, payments, release sign-off, data deletion, database migration) are the ONLY place a reviewer can now be forced from — derived once at build, recorded on the manifest with a plain-English reason quoting the matched phrase, and read (not re-derived) by continue.**

## Performance

- **Duration:** ~75 min
- **Tasks:** 2 completed
- **Files modified:** 7 (2 new: `cmd/queen_risk_signals.go`, `cmd/queen_forced_reviewer_test.go`)

## Accomplishments

- `cmd/queen_risk_signals.go` — the entire named-risk vocabulary in one table (`queenRiskSignalTable`, exactly 5 entries), a word-boundary phrase matcher, the caste-collapse rule (D-04), the reason-sentence builder, and the build/continue record conversion.
- `codexBuildManifest.ForcedReviewers` — a new JSON field (`forced_reviewers`) carrying the build's one-time derivation, verified end-to-end via `aether build 1 --plan-only` on a credentials-wording fixture.
- `queenContinueDispatchesWithJudgement` / `queenContinueReviewSpecsWithJudgement` now accept the build's recorded forced-reviewer set and union it into the dispatch list AFTER any proposal/budget handling, so neither a Queen proposal nor a budget trim can drop a forced reviewer.
- A worker sent because of a forced signal now sees why in its own brief ("Why you were sent: this touches logins and passwords (the plan mentions \"password reset\")").
- Five new tests pin the table, the collapse rule, the empty-input case, and the word-boundary encoding — all asserted on the real continue dispatch list, never an intermediate decision record.

## Task Commits

1. **Task 1: One named risk, end to end — signal detected, recorded at build, dispatched once at continue** - `cb22c01d` (feat)
2. **Task 2: The full five-signal table under test, and the continue-side tests the new reviewer falsifies** - `799bb438` (test)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/queen_risk_signals.go` — the named-risk vocabulary, matcher, collapse rule, reason sentence, and build↔continue record conversion.
- `cmd/queen_forced_reviewer_test.go` — `TestRiskSignalTableHasExactlyFiveEntries`, `TestForcedReviewerCrossesTheBuildContinueBoundary`, `TestReviewerForcedOnlyByNamedRisk`, `TestTwoSignalsOneCasteCollapseToOneDispatch`, `TestEmptyPhaseForcesNoReviewer`, `TestSignalMatchingIsWordBounded`.
- `cmd/codex_build.go` — `codexForcedReviewerRecord` type, `codexBuildManifest.ForcedReviewers` field, populated from `queenForcedReviewersForPhase(phase)` beside `manifest.CasteDecision`.
- `cmd/codex_continue.go` — `codexContinueReviewSpec.Rationale` field; `queenContinueDispatchesWithJudgement` / `queenContinueReviewSpecsWithJudgement` gained a `forced []codexForcedReviewerRecord` parameter and now union forced reviewers after budget trim; `renderCodexContinueReviewBrief` states the forced reason to the worker.
- `cmd/codex_continue_plan.go` — the wrapper/external continue lane's call site updated to pass `manifest.Data.ForcedReviewers` (see deviation below).
- `cmd/queen_spawn_budget.go` — `queenRequiredCastesForBudget`'s continue branch now folds in forced-reviewer castes so `RequiredCastes` stays an honest answer for any caller reading the budget struct directly.
- `cmd/floor_unskippable_test.go`, `cmd/queen_judgement_test.go` — call sites updated for the new `forced` parameter (compilation only; behavior unchanged for these fixtures, none of which carry signal wording).
- `.aether/schemas/completion-packet.schema.json` — regenerated to include the new manifest field and record type (see deviation below).

## Decisions Made

- **Longest matching phrase wins per signal**, not first-in-list. Task 1's acceptance criterion required the forced reviewer's reason to quote "password reset" specifically; the phrase list also contains the shorter "password", and picking the first match in authoring order quoted that instead. Discovered and fixed during Task 1's own verification loop, before the task commit.
- **Rationale carries a fixed "this touches ..." marker.** The pre-existing keyword/mode scoring engine (untouched by this plan) already dispatches `auditor` on any production-mode phase and `gatekeeper` on phases matching its own older security-keyword list — both castes this plan's signal table can also force. Without a way to tell the two mechanisms apart on the same caste name, `TestReviewerForcedOnlyByNamedRisk`'s "forces nothing" rows (CSV export, a production phase with no signal wording) could not be written correctly, since the OLD floor still legitimately dispatches `auditor` for those very fixtures. The marker is what makes "forced by this plan's mechanism" a checkable fact independent of what else dispatches the same caste. Confirmed empirically during Task 2 (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `cmd/codex_continue_plan.go` needed updating, though not listed in the plan's `files_modified`**
- **Found during:** Task 1
- **Issue:** `plannedExternalContinueDispatches` (the wrapper/external continue lane) calls `queenContinueReviewSpecsWithJudgement` directly; adding the new `forced` parameter to that function's signature broke compilation of this call site.
- **Fix:** Updated the one call site to pass `manifest.Data.ForcedReviewers` (the manifest was already an argument to the enclosing function — no new plumbing needed).
- **Files modified:** `cmd/codex_continue_plan.go`
- **Verification:** `go build ./cmd/aether` passes.
- **Committed in:** `cb22c01d` (Task 1 commit)

**2. [Rule 3 - Blocking] Regenerated the completion-packet JSON schema**
- **Found during:** Task 2's `go test ./cmd -count=1` reconciliation pass
- **Issue:** `TestCompletionPacketSchemaMatchesStructs` failed — the new `codexBuildManifest.ForcedReviewers` field and `codexForcedReviewerRecord` type drifted `.aether/schemas/completion-packet.schema.json` out of sync with the Go structs.
- **Fix:** Ran `aether contract-schema --write` as the test's own failure message instructed.
- **Files modified:** `.aether/schemas/completion-packet.schema.json`
- **Verification:** `TestCompletionPacketSchemaMatchesStructs` and `TestContractDocExampleValidatesAgainstSchema` both pass; full `go test ./cmd -count=1` green.
- **Committed in:** `799bb438` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking — both required for the build/test suite to stay green, no scope creep).
**Impact on plan:** Neither deviation touched the plan's stated territory (`queenBuildSafetyRequiredCastes`, `queenPhaseHasSecuritySignal`, `queenBuildSafetyReviewRequired` remain untouched, as required).

## Issues Encountered

- `TestSwarmCompatibilityWatchReportsActiveWorkers` failed once during a full `go test ./cmd -count=1` run under load (`active_count = 0, want 1`), then passed cleanly in isolation (3/3) and on a subsequent full-suite rerun (0 failures). This is a pre-existing timing-sensitive test in an unrelated file (`compatibility_cmds_test.go`) — out of this plan's scope per the deviation rules' scope boundary. Not fixed; not touched.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `queen_risk_signals.go`'s five artifacts (`riskSignal`, `queenRiskSignalTable`, `queenForcedReviewersForPhase`, `queenForcedContinueReviewers`, `codexForcedReviewerRecord`) are in place for 194-02 (which shrinks the OLD build-side floor these coexist with today) and 194-06 (which fills `PathPatterns` and wires a changed-files detector through the already-accepted `changedFiles` parameter on `queenForcedContinueReviewers`).
- The "Known interim state" the plan called out is confirmed present and unavoidable at this point: a security-signal phase still draws both the old implicit build-side reviewer (via the untouched `queenBuildSafetyRequiredCastes`) and the new forced continue-side reviewer. `.planning/WINDOWS.md` #1 stays open until 194-02 and 194-08 close it.
- No blockers for the next plan in the wave sequence.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*
