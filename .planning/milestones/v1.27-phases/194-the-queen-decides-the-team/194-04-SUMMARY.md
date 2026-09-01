---
phase: 194-the-queen-decides-the-team
plan: 04
subsystem: orchestration
tags: [queen, check-in-card, ceremony, forced-reviewer, plain-english, D-09, D-10]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenRiskSignalTable / codexForcedReviewerRecord (cmd/queen_risk_signals.go, cmd/codex_build.go) — the forced-reviewer record this plan renders on the card"
  - phase: 194-02
    provides: "the shrunken build floor (builder alone / nil on discovery) — what REQUIRED can legitimately mean on a build manifest"
  - phase: 194-03
    provides: "queenCasteJudgement.Reasons / caste_decision.reasons — the per-worker reason sentences the card's reason slot already reads via spawn_budget.selected_reasons"
provides:
  - "renderCeremonyTeamCheckin's reason slot carries only per-phase sentences — the roster's generic caste description never appears there, only in a separately labelled what_it_does slot"
  - "codexBuildManifest.ForcedReviewerAnnouncement — one owner-facing sentence per forced reviewer, composed once at build from ForcedReviewers"
  - "the check-in card's announcement block and forced result-map key — the owner sees which reviewer will run at the checking step and why, before any worker spawns"
  - "an explicit test (TestRequiredMeansBuilderOrNamedSignal) that REQUIRED means the implementation worker or a named signal, nothing else"
affects: [194-05-probe-refusal-gate, 194-07-waive-choice, 196-cost-line, 197-closing-card]

# Actuals (#2632)
actuals:
  tokens: 5495
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A caste reaching the card with no per-phase reason renders an explicit marker ('no reason was recorded for sending this worker') rather than silently falling back to a generic description — a bug upstream is shown, not papered over"
    - "One composition function (composeForcedReviewerAnnouncement) produces the owner-facing sentence in both the manifest field and the card's live render, so the two can never independently drift"
    - "Plain human labels for forced-reviewer castes (\"security reviewer\", \"quality reviewer\") kept separate from the internal registry identifiers (gatekeeper, auditor) — never shown to the owner"

key-files:
  created: []
  modified:
    - cmd/ceremony_team_checkin.go
    - cmd/ceremony_team_checkin_test.go
    - cmd/codex_build.go
    - .aether/schemas/completion-packet.schema.json
    - .aether/docs/retired-tests-ledger.md

key-decisions:
  - "The card composes its own announcement text from the raw forced_reviewers array (via forcedReviewerRecordsFromManifest + composeForcedReviewerAnnouncement) rather than reading the manifest's precomputed ForcedReviewerAnnouncement string directly — the same composition function backs both, so there is one derivation, not two that could drift, and the card works from any manifest that carries forced_reviewers even if a caller never set the standalone field."
  - "REQUIRED marking logic itself did not need to change: the build-side spawn budget's required_castes never includes a forced-reviewer caste (queenRequiredCastesForBudget only folds forced reviewers into the continue flow, per 194-01), and a forced reviewer is never an actual build-time dispatch (D-05 — announced, not dispatched). TestRequiredMeansBuilderOrNamedSignal is therefore a locking regression test against a property already true by construction, exactly as the plan asked for (\"add an explicit assertion instead of trusting that\")."

patterns-established:
  - "Plain-English reviewer labels (forcedReviewerPlainLabel) live beside the announcement composer in codex_build.go, not in the visuals caste-label map — that map is for internal caste identity displays (Builder, Watcher, ...), while this one exists specifically so the owner never sees a registry identifier in a forced-reviewer sentence."

requirements-completed: [TEAM-02, TEAM-03]

coverage:
  - id: D1
    description: "The card's reason slot carries only per-phase sentences; the roster's generic description of what a caste does is shown, if at all, in a separately labelled what_it_does position and never as a justification."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestTeamCheckinNeverShowsAGenericBlurbAsAReason"
        status: pass
    human_judgment: false
  - id: D2
    description: "A caste that reaches the card with no per-phase reason renders an explicit no-reason-recorded marker, never the generic description."
    requirement: TEAM-03
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestTeamCheckinNeverShowsAGenericBlurbAsAReason"
        status: pass
    human_judgment: false
  - id: D3
    description: "The check-in card names the signal every time a reviewer is forced, in plain English, quoting the matched phrase — asserted on the rendered card text, not the result map alone."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestCardNamesTheSignalForEveryForcedReviewer"
        status: pass
    human_judgment: false
  - id: D4
    description: "REQUIRED on the card means exactly one of two things: the worker writes the code (builder), or a named signal forced it and that signal is shown. Nothing else is marked REQUIRED."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestRequiredMeansBuilderOrNamedSignal"
        status: pass
    human_judgment: false
  - id: D5
    description: "The card renders no announcement block and no empty heading when no reviewer is forced."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestCardNamesTheSignalForEveryForcedReviewer (plain phase has no announcement block)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The check-in command remains an inspection: rendering the card writes nothing to colony state."
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestTeamCheckinDoesNotMutate"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 4: The Card Stops Guessing and Starts Naming the Signal Summary

**The pre-build check-in card no longer shows a caste's generic "what it does" blurb as if it were a reason for sending that worker, and it now names the exact signal and matched phrase whenever a reviewer is forced onto the checking step — both changes proven on the rendered card text itself, not an intermediate struct.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 2 completed
- **Files modified:** 5 (`cmd/ceremony_team_checkin.go`, `cmd/ceremony_team_checkin_test.go`, `cmd/codex_build.go`, `.aether/schemas/completion-packet.schema.json`, `.aether/docs/retired-tests-ledger.md`)

## Accomplishments

- `renderCeremonyTeamCheckin`'s reason slot (`result["reasons"]`) no longer falls back to `casteRosterProduces` (the roster's generic caste description) when the spawn budget carried no per-caste rationale — that fallback read as a justification for sending the worker and, per D-09, is not one. A caste reaching the card with no per-phase reason now renders an explicit `"no reason was recorded for sending this worker"` marker, surfacing the upstream bug instead of papering over it.
- The roster description still appears on the card, but only in a new, separately labelled `what_it_does` position (both in the result map and on the rendered line, `(what it does: ...)`), so the owner can see what a caste generally does without mistaking it for why it was sent today.
- `codexBuildManifest.ForcedReviewerAnnouncement` (new field, `forced_reviewer_announcement`) composes one owner-facing sentence per forced reviewer from the build's `ForcedReviewers` record, in the exact shape D-05 specifies: `"a security reviewer will check this at the verification step — this touches logins (the plan mentions \"password reset\")"`. Uses plain human labels (`security reviewer`, `quality reviewer`), never the registry identifiers (`gatekeeper`, `auditor`).
- The check-in card itself reads `forced_reviewers` straight off the manifest (`forcedReviewerRecordsFromManifest`) and renders the same composed sentences as an announcement block below the worker lines, headed so it is clear these reviewers are not part of this build's team. A manifest with no forced reviewers renders no block and no empty heading. A new `forced` key on the result map carries each forced caste's signals and reason so a wrapper can ask about it without re-deriving anything.
- `TestRequiredMeansBuilderOrNamedSignal`: an explicit, four-fixture regression test (plain phase, password-reset phase, discovery phase, two-signal phase) asserting that every caste the card marks REQUIRED is either `builder` or a caste the manifest's `forced_reviewers` names with a non-empty signal list — nothing else. Built from real `runCodexBuildPlanOnlyWithOptions` manifests (JSON round-tripped, matching the exact shape the card reads in production), not hand-typed fixtures, so it asserts on the real dispatch/manifest surface rather than an intermediate struct.
- `TestTeamCheckinNeverShowsAGenericBlurbAsAReason` replaces the retired `TestTeamCheckinFallsBackToRosterProduces` (which asserted exactly the forbidden fallback); recorded in `.aether/docs/retired-tests-ledger.md`.
- `TestTeamCheckinDoesNotMutate` re-proves the check-in command is a pure inspection: rendering the card leaves colony state byte-identical before and after.

## Task Commits

1. **Task 1: The card stops presenting a generic caste description as a reason** - `b1e1aa2f` (feat)
2. **Task 2: The card says which signal forced each reviewer, and REQUIRED stops meaning anything else** - `caf459bb` (feat)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/ceremony_team_checkin.go` — reason-slot fallback removed; `what_it_does` map added to the result and rendered line; `forcedReviewerRecordsFromManifest` (reads `forced_reviewers` back off the manifest map); announcement block render; `forced` key on the result map.
- `cmd/ceremony_team_checkin_test.go` — fixture updated off the two forbidden reason shapes onto plain-English sentences 194-03 now produces; `TestTeamCheckinNeverShowsAGenericBlurbAsAReason` (new); `TestCardNamesTheSignalForEveryForcedReviewer`, `TestRequiredMeansBuilderOrNamedSignal`, `TestTeamCheckinDoesNotMutate` (new) — all built from real plan-only build manifests via a new `manifestMapFromBuild` test helper.
- `cmd/codex_build.go` — `codexBuildManifest.ForcedReviewerAnnouncement` field; `composeForcedReviewerAnnouncement` and `forcedReviewerPlainLabel` functions; manifest population wired in beside the existing `ForcedReviewers` assignment.
- `.aether/schemas/completion-packet.schema.json` — regenerated (`aether contract-schema --write`) for the new manifest field.
- `.aether/docs/retired-tests-ledger.md` — entry for `TestTeamCheckinFallsBackToRosterProduces`, recovered-by `TestTeamCheckinNeverShowsAGenericBlurbAsAReason`.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) the card composes its own announcement text from the raw `forced_reviewers` array via the same `composeForcedReviewerAnnouncement` function the manifest field uses, rather than reading the manifest's precomputed string directly — one derivation, usable from any manifest shape; (2) the REQUIRED-marking logic itself needed no code change, since a forced reviewer is never an actual build-time dispatch — the new test is a locking regression guard on a property already true by construction.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Regenerated the completion-packet JSON schema**
- **Found during:** Task 2's `go test ./cmd -count=1` reconciliation pass
- **Issue:** `TestCompletionPacketSchemaMatchesStructs` failed — the new `codexBuildManifest.ForcedReviewerAnnouncement` field drifted `.aether/schemas/completion-packet.schema.json` out of sync with the Go structs (same pattern as plan 194-01's Task 2 deviation).
- **Fix:** Ran `aether contract-schema --write` as the test's own failure message instructed. Diff is a clean 3-line addition (the new `forced_reviewer_announcement` string field).
- **Files modified:** `.aether/schemas/completion-packet.schema.json`
- **Verification:** `TestCompletionPacketSchemaMatchesStructs` passes; full `go test ./cmd -count=1` green.
- **Committed in:** `caf459bb` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (blocking — required for the completion-packet contract test to stay green, no scope creep).
**Impact on plan:** Deviation touched only the generated schema artifact, not the plan's stated territory.

## Issues Encountered

- Both tasks landed tightly interleaved edits within `renderCeremonyTeamCheckin` (same function, adjacent lines). To keep the per-task commit protocol honest rather than bundling both tasks into one commit, Task 2's code was temporarily reverted (via targeted `git checkout -- <file>` on `cmd/codex_build.go` and manual re-edits on `cmd/ceremony_team_checkin.go`/`_test.go`, restored from an in-memory backup afterward — no `git stash`, no blanket reset), Task 1 was verified and committed in isolation, then Task 2's code was restored and committed separately. Both intermediate states were independently build- and test-verified before their commit.
- One full-repo `go test ./... -count=1` run showed `TestAvailabilityProbeRetriesOnlyTimeouts` (`pkg/codex`) failing; confirmed non-regression by three clean isolated reruns immediately after — this matches the project's documented pre-existing timing-flaky pattern for that exact test, in a package this plan never touches. Not fixed; not touched.

## Known Stubs

None. Every `must_haves.truths` deliverable has a passing test asserted on the real rendered card text or the real manifest surface, not an intermediate struct.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `codexBuildManifest.ForcedReviewerAnnouncement`, `forcedReviewerRecordsFromManifest`, and the card's `forced` result-map key are in place for plan 194-07 (the waive choice, D-03/D-14) to extend without re-deriving the forced-reviewer set.
- The card's `what_it_does` slot is available for any future surface that wants the roster's generic description without risking it being read as a justification.
- No blockers for 194-05 (the probe-refusal gate) or 194-07.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: commit `b1e1aa2f` (Task 1) in `git log --oneline --all`
- FOUND: commit `caf459bb` (Task 2) in `git log --oneline --all`
- FOUND: `cmd/ceremony_team_checkin.go`
- FOUND: `cmd/ceremony_team_checkin_test.go`
- FOUND: `cmd/codex_build.go`
- FOUND: `.aether/schemas/completion-packet.schema.json`
- FOUND: `.aether/docs/retired-tests-ledger.md`
- `go build ./cmd/aether` — pass
- `go vet ./...` — pass
- `go test ./cmd -count=1` — pass
- `go test ./... -count=1` (whole repo) — pass (one confirmed non-regression, timing-flaky test in an untouched package, verified clean in 3/3 isolated reruns)
- Acceptance criteria greps (retired-test grep count = 0, ledger citation present, `what_it_does` key present) — all verified pass
