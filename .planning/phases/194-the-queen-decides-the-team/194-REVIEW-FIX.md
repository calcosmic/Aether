---
phase: 194-the-queen-decides-the-team
fixed_at: 2026-08-26T21:02:00Z
review_path: .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md
iteration: 5
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 194: Code Review Fix Report

**Fixed at:** 2026-08-26T21:02:00Z
**Source review:** `.planning/phases/194-the-queen-decides-the-team/194-REVIEW.md`
**Iteration:** 5

**Summary:**

- Findings in scope: 7 (5 Critical, 2 Warning)
- Fixed: 7
- Skipped: 0

In plain English: the owner-only reviewer waiver is now enforced at the final
dispatch boundary and cannot be forged by a worker; explicit Queen team flags
reach normal build/continue execution; trimmed workers keep their individual
reasons; untracked sensitive files cannot bypass reviewer selection; and the
reference documentation/tests now describe Codex's shipped behavior.

## Fixed Issues

### CR-01: A genuine owner waiver does not remove a plan-wording reviewer from the dispatched continue team

**Status:** Fixed — requires human verification (dispatch-state logic)

**Files modified:** `cmd/codex_continue.go`, `cmd/forced_reviewer_waiver_test.go`

**Commit:** `e68b152d`

**Applied fix:** Reconciled the final continue dispatch list against authentic
waivers. The waived phase-wording reviewer is removed while another live signal,
an explicitly proposed reviewer, or a heavy-depth requirement still preserves
that caste.

**Verification:** New end-to-end regression
`TestWaiverControlsBothFinalContinueDispatchLists` failed before the change and
passes for both in-process and external continue lanes afterward. The focused
waiver/continue set passed 14 tests.

### CR-02: Normal build and continue silently discard Queen team flags

**Status:** Fixed — requires human verification (CLI option wiring/state logic)

**Files modified:** `cmd/codex_workflow_cmds.go`, `cmd/codex_build.go`,
`cmd/codex_continue.go`, `cmd/codex_build_test.go`, `cmd/codex_continue_test.go`

**Commit:** `79af962e`

**Applied fix:** Forwarded `--castes`, `--caste-reason`, and every
`--caste-why` through normal build and continue execution. Normal build now
persists the resulting caste decision, and continue's option snapshot/comparison
includes normalized team values so a changed proposal is not mistaken for a
repeat.

**Verification:** `TestBuildCLINormalPathForwardsQueenTeamFlags` and
`TestContinueCLINormalPathForwardsQueenTeamFlags` failed before the fix and pass
afterward; the focused CLI option/state set passed 7 tests.

### CR-03: Trim optional workers silently refuses every retained worker

**Status:** Fixed — requires human verification (wrapper/runtime judgment flow)

**Files modified:** `.claude/commands/ant-build.md`,
`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`,
`cmd/queen_judgement_test.go`

**Commit:** `51a1b466`

**Applied fix:** All three wrapper recipes now replay one
`--caste-why "<caste>=<reason>"` for every retained/current optional caste when
trimming or declining a reviewer. Team-level rationale remains separate.

**Verification:** New trace test
`TestWrapperTrimReplaysReasonsForRetainedOptionalWorkers` failed six assertions
before the fix and passed afterward; 5 related judgment tests passed, and the
three wrapper copies compare byte-for-byte identical.

### CR-04: A worker can reopen the decline window and manufacture an owner-only waiver

**Status:** Fixed — requires human verification (security/attempt-state logic)

**Files modified:** `cmd/pending_decision.go`,
`cmd/forced_reviewer_waiver.go`, `cmd/ceremony_team_checkin.go`,
`cmd/handoff_decisions_cmd.go`, `cmd/codex_build.go`,
`cmd/forced_reviewer_waiver_test.go`, `cmd/testdata/command_catalog.json`

**Commits:** `a4fffdab` (runtime and adversarial tests), `6c7639a3`
(deterministic command-catalog refresh)

**Applied fix:** Bound waiver rows to the active build attempt and a random
owner capability whose SHA-256 digest alone is persisted. Resolution now checks
phase, exact question, authentic source, attempt, capability (constant-time),
and the pre-dispatch window atomically. Plan-only re-entry cannot reopen a live
attempt's decision window; terminal retries remain available.

**Verification:** Adversarial regressions
`TestWorkerCannotForceReplanToReopenReviewerDecline` and
`TestForcedReviewerWaiverRejectsLookalikeDecisionSource` failed before the fix
and pass afterward. The focused security suite passed 23 tests; the broader
plan/build-attempt/decision/waiver/check-in set passed 34. The refreshed catalog
golden passed and contains only the new `waiver-capability` flag.

### CR-05: Changed-file safety misses brand-new untracked sensitive files

**Status:** Fixed — requires human verification (reviewer-selection logic)

**Files modified:** `cmd/queen_risk_signals.go`,
`cmd/queen_forced_reviewer_test.go`

**Commit:** `0e0561ef`

**Applied fix:** The safety-only changed-file union now includes
`git ls-files --others --exclude-standard`. The scan remains deliberately local
to forced-reviewer selection, so it does not widen build finalization or commit
selection.

**Verification:** The strengthened untracked migration regression failed before
the fix and passed afterward. Five related reviewer tests passed 11 subtests.

### WR-01: Caste reference documents obsolete continue floors

**Status:** Fixed

**Files modified:** `.aether/docs/command-playbooks/caste-relevance-reference.md`,
`cmd/caste_relevance_doc_test.go`

**Commit:** `d111d633`

**Applied fix:** Documented no unconditional light/standard caste and the heavy
Gatekeeper/Auditor floor with Probe only for testable code; Watcher is explicitly
not a continue floor. The documentation test now compares exact code-derived
caste sets and rows for each depth, including documentation-only heavy work.

**Verification:** `TestCasteRelevanceDoc_ContinueFloorsMatchPolicy` failed
against the old table and passed after the update; both caste-reference tests
passed.

### WR-02: Codex visual tests assert retired slash-command guidance

**Status:** Fixed

**Files modified:** `cmd/codex_visuals_test.go`

**Commit:** `341cd4cb`

**Applied fix:** Replaced every stale `/ant-*` expectation in the Codex visual
tests with native `aether ...` lifecycle commands. Two negative assertions were
also updated so they continue to reject an incorrect native build/status hint
instead of passing vacuously.

**Verification:** The 11 scoped tests failed 11/11 before the update and passed
11/11 afterward; the expanded set including the two strengthened negative
assertions passed 13/13. No `/ant-*` expectation remains in the file.

## Verification Summary

- Finding-specific fail-before/pass-after regressions: all 7 findings covered.
- `go vet ./...`: pass.
- `go build ./cmd/aether`: pass.
- `go test ./... -count=1 -timeout 900s -p 1`: 7,183 tests passed across all
  20 packages; no exclusions.
- An earlier default-timeout full run hit the `cmd` package's 10-minute default
  and reproduced the already documented unrelated
  `TestAvailabilityProbeRetriesOnlyTimeouts` load-sensitive failure. That test
  passed 3/3 isolated reruns (12 subtests). The final full run used the standard
  900-second timeout documented in `.planning/codebase/TESTING.md`; `-p 1`
  removed cross-package load contention without omitting any package or test.

## Skipped Issues

None.

---

_Fixed: 2026-08-26T21:02:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 5_
