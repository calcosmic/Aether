---
phase: 201-queen-led-work-cycle
reviewed: 2026-09-10T19:54:39Z
depth: standard
files_reviewed: 50
files_reviewed_list:
  - cmd/attempt_artifacts.go
  - cmd/autopilot_goal_level_test.go
  - cmd/autopilot_policy.go
  - cmd/boundary_double_dispatch_test.go
  - cmd/build_attempt.go
  - cmd/build_print_brief.go
  - cmd/build_unverified_card_test.go
  - cmd/caste_model_routing.go
  - cmd/classic_contract_test.go
  - cmd/classic_coverage_201_test.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_test.go
  - cmd/codex_build.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue.go
  - cmd/codex_verify_advance_test.go
  - cmd/codex_verify_advance.go
  - cmd/coherent_jobs.go
  - cmd/command_truth.go
  - cmd/compatibility_cmds.go
  - cmd/deterministic_floor.go
  - cmd/failure_evidence_test.go
  - cmd/job_telemetry_test.go
  - cmd/job_telemetry.go
  - cmd/lifecycle_closeout_work_outcome_test.go
  - cmd/lifecycle_closeout.go
  - cmd/memory_feed_continue.go
  - cmd/memory_feed.go
  - cmd/result_file_precision_test.go
  - cmd/review_depth_test.go
  - cmd/spend_cost_line_test.go
  - cmd/spend_cost_line.go
  - cmd/status_running_total_test.go
  - cmd/status.go
  - cmd/testdata/classic-contract/v1/cases.json
  - cmd/testdata/classic-contract/v1/mechanisms.json
  - cmd/testdata/classic-contract/v1/schema.json
  - cmd/turnaround_levers_test.go
  - cmd/turnaround_target_test.go
  - cmd/turnaround_target.go
  - cmd/verification_boundary_test.go
  - cmd/verification_boundary.go
  - cmd/verification_scope.go
  - cmd/work_identity_test.go
  - cmd/work_repair_test.go
  - cmd/work_repair.go
  - pkg/colony/flags.go
  - pkg/colony/work_outcome_test.go
  - pkg/colony/work_outcome.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 201: Code Review Report

**Reviewed:** 2026-09-10T19:54:39Z
**Depth:** standard
**Files Reviewed:** 50
**Status:** issues_found

## Summary

Phase 201 ("Queen-Led Work Cycle") adds a durable per-attempt evidence and
timing model on top of the existing build/continue lifecycle: attempt-bound
claims/verification/telemetry artifacts, a single shared
accept/verify/advance decision body used by all three continue lanes, a
recorded verification-boundary decision (check-step vs. build-end), a
six-verdict work-outcome vocabulary with full-ceremony closeout rendering, an
eight-segment job timing record with an honesty-first measured/unmeasured
discipline, a ratified turnaround target and comparison, and a proportionate
one-worker "quick" attempt model.

I read every file in the required-reading list, cross-referenced the ~2,400
lines actually changed in this phase's largest files against `git diff` from
the phase's start commit, and traced call graphs for the newly-added
functions to confirm which ones are genuinely reachable in production versus
only reachable from tests.

The code is unusually well-documented and defensively written (explicit
refusal-by-name on malformed inputs, non-fatal/warn-only discipline for
purely-evidentiary writes, deliberate avoidance of derived/estimated
figures). I found one genuine pre-existing bug that this phase's own work
fixed and explicitly documented (a variable-shadowing bug in
`cmd/codex_continue_finalize.go` that made a soft-block gate auto-resolution
invisible to the advancement decision — already fixed, not a re-finding).
I did not find any new correctness or security defects introduced by this
phase's changes. The issues below are quality/maintainability observations.

**Verification of the two disclosed gaps in the review brief:**

- `runBoundedRepairRound` (`cmd/work_repair.go`) has no production caller —
  confirmed by grep and by the call-graph test
  `TestAutopilotSelectsTransitionsFromOneAcceptedGoal`'s own subtest, which
  explicitly documents and asserts this as a known gap. Honestly disclosed.
- `buildUnverifiedCloseoutDetails` / `buildVerifiedCloseoutDetails`
  (`cmd/codex_build.go`) have no production caller — confirmed by grep across
  the whole non-test tree. Honestly disclosed in `201-05-SUMMARY.md`,
  `201-06-SUMMARY.md`, and `201-07-SUMMARY.md`'s own "Open gap for a later
  plan" sections, and never wired in any subsequent 201-0x/201-1x plan.

**A third gap of the same shape, not named in the review brief but verified
during this review:** `queenApplyVerificationBoundary` and
`attachVerificationBoundary` (`cmd/verification_boundary.go`) also have zero
production callers — grep confirms both are referenced only from doc
comments and from test fixtures (`attemptWithVerificationBoundaryRecorded`,
`newTestVerificationBoundaryAttempt`). This means the entire D-05
build-end-reviewer mechanism (`queenBuildPostWaveDispatches`'s gate,
`plannedContinueReviewDispatches`'s gate, `continueReviewReportFromBuildEndFindings`)
currently always takes the check-step default in production: nothing ever
proposes or persists a build-end decision, so `verificationBoundaryForAttempt`
never returns `ok=true` on a real colony. This is honestly disclosed in
`201-05-SUMMARY.md`'s own "Open gap for a later plan" section ("nothing in
production yet calls `queenApplyVerificationBoundary` + `attachVerificationBoundary`
to actually PROPOSE and PERSIST a build-end decision ... build-end reviewer
dispatch and the verified (success) build closeout can never actually fire in
production") and confirmed never wired in any later plan through 201-15. I am
surfacing this explicitly because it was not named in the review's own list
of known gaps, but it checks out as the same honestly-disclosed pattern as
the other two — not a new defect, just worth the orchestrator's awareness
that three, not two, pieces of this phase's machinery are currently inert in
production.

## Warnings

### WR-01: `quickAttemptRecord` is described as an "attempt" but is never durably recorded

**File:** `cmd/command_truth.go:393-424`
**Issue:** The doc comments for `quickAttemptRecord`, `newQuickAttempt`, and
`recordDispatch` describe `/ant-quick` as running "on the same attempt ...
model the rest of the work cycle uses" and "opens one attempt for a quick
request" — language that mirrors `buildAttemptRecord`'s durable,
disk-persisted journal (`cmd/build_attempt.go`). In practice,
`quickAttemptRecord` is populated purely in memory inside `runQuickScout` and
is **never written to disk anywhere** — there is no `store.SaveJSON` /
`writeAttemptBoundArtifact` call for it, and no other file in the codebase
reads it back. Its only externally-visible traces are the `attempt_id`
returned in the JSON result map and the `attempt:<id>` tag written into a
midden entry on failure (via `recordQuickFailureToMidden`). A future
maintainer reading the doc comment or debugging a quick-request issue by
looking for `.aether/data/attempts/quick-*.json` (the pattern every other
attempt-bound artifact in this phase follows, per
`cmd/attempt_artifacts.go`) will not find one, because none is ever written.
This doesn't break anything today (the current tests only assert the midden
tag and the returned `attempt_id`, which both work), but it's a
maintainability trap: the abstraction promises durability parity with the
build-attempt model that it does not actually deliver.
**Fix:** Either persist the record (e.g. `writeAttemptBoundArtifact(attempt.ID, "quick", attempt)` after the verdict is known, mirroring the claims/verification/telemetry pattern this same phase established), or soften the doc comments to state plainly that this is an in-memory-only identity used to tag the midden entry and the result payload, not a durable journal entry:
```go
// quickAttemptRecord is an in-memory identity for one quick request -- it is
// NOT persisted to disk. Its only durable trace is the "attempt:<id>" tag on
// a midden failure entry (see recordQuickFailureToMidden) and the attempt_id
// field in the returned result. Unlike buildAttemptRecord, there is no
// attempts/quick-<id>.json journal file to read back.
```

### WR-02: `os.Stat` error stored in a variable named as if it were a boolean

**File:** `cmd/build_attempt.go:1015-1016, 1060-1062`
**Issue:**
```go
durableAbsolute := filepath.Join(store.BasePath(), filepath.FromSlash(completionRel))
_, durableAlreadyExists := os.Stat(durableAbsolute)
```
`os.Stat` returns `(os.FileInfo, error)`, so `durableAlreadyExists` actually
holds an `error` (nil when the file exists, a `*PathError` when it doesn't),
not a boolean. The name reads as "true means the file already exists," which
is the *opposite* of what the value represents once you look at how it's
used two dozen lines later:
```go
if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {...}); err != nil {
    if os.IsNotExist(durableAlreadyExists) {
        _ = os.Remove(durableAbsolute)
    }
    ...
}
```
The logic itself is correct (only clean up the just-written completion file
if it did **not** exist before this call — i.e. this call created it fresh),
but the name actively misleads a reader trying to verify that correctness.
This is exactly the kind of naming mismatch that causes a future edit (e.g.
"let's early-return if `durableAlreadyExists`") to introduce a real bug.
**Fix:** Rename to reflect what it actually holds, e.g.:
```go
_, statErr := os.Stat(durableAbsolute)
...
if os.IsNotExist(statErr) {
    _ = os.Remove(durableAbsolute)
}
```

## Info

### IN-01: `newQuickAttempt`'s duration figure is computed but the pre-flight failure paths still bypass the attempt model entirely

**File:** `cmd/command_truth.go:473-487`
**Issue:** `runQuickScout` opens `attempt := newQuickAttempt(...)` at the top
of the function, but the two pre-flight availability checks (dispatcher
unavailable, agent validation failure) both `return nil, err` before
`attempt.recordDispatch(...)` is ever called and before any midden or result
data references the attempt at all — the `attempt` value is silently
discarded on those two paths. This mirrors the pre-existing, documented
behavior (`recordQuickFailureToMidden`'s own doc comment explicitly says
"the two pre-flight availability returns ... are deliberately not recorded
here"), so it is not a new defect, but the `attempt` variable now exists on
those code paths purely to be thrown away, which is a minor readability
smell — a reader has to check the doc comment on a *different* function to
learn that this is intentional rather than an oversight.
**Fix:** Optional — a one-line comment at the two early-return sites (`return nil, fmt.Errorf("quick scout cannot start because %s", ...)` and `return nil, fmt.Errorf("scout agent unavailable: %w", err)`) noting "pre-flight failure: no attempt is recorded, matching recordQuickFailureToMidden's documented scope" would save a future reader the cross-file lookup.

---

_Reviewed: 2026-09-10T19:54:39Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
