---
phase: 201-queen-led-work-cycle
plan: "12"
subsystem: queen-orchestration
tags: [go, telemetry, timing, build-attempt, spend, status, WORK-08, CEC-06]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 06)
    provides: "cmd/spend_cost_line.go: renderSpendCostLine(phase)'s attempt-bound Elapsed line and renderSpendCostLineBlock's reserved extension point -- this plan's own doc comment named plan 201-12 by number as the intended extender"
  - phase: 201-queen-led-work-cycle (plan 08)
    provides: "cmd/attempt_artifacts.go: attemptBoundArtifactPath/writeAttemptBoundArtifact/readAttemptBoundArtifact, the one canonical attempt-bound artifact mechanism this plan registers a third kind (telemetry) onto; codexBuildDispatch.AttemptID, stamped before this plan's build-lane write needs it"
provides:
  - "cmd/job_telemetry.go: jobTelemetryRecord, the eight-segment (queue/preflight/model/tool_call/context/work/verification/wait) attempt-bound timing record, each segment either a measured duration naming its instrumentation source or an explicit reasoned-unmeasured state -- modeled on verificationScope's measured/source discipline; writeJobTelemetryRecord/readJobTelemetryRecord, non-fatal and skip-when-nothing-measured"
  - "cmd/codex_build.go: the direct/native build lane measures queue (job planning done to brief assembly start), context (prepareBuildWorkerBriefFiles), and work (executeCodexBuildDispatches) -- the only segments genuinely observable inside one process invocation on this lane -- and writes the record at the end of a successful build, alongside the existing per-worker spend-row write"
  - "cmd/codex_continue.go: runCodexContinueVerification measures the verification segment around the one shared runDeterministicFloor call, merging it into whatever the build lane already wrote for the same attempt (jobTelemetryMergeRecords) rather than overwriting it"
  - "cmd/spend_cost_line.go: renderJobTelemetryClosingLine extends the existing attempt-bound Elapsed line to also name the largest measured segment, or say the breakdown was not measured -- still exactly one block, one timing line"
  - "cmd/status.go: renderJobTelemetryDrillDown renders the full eight-segment breakdown for a selected build attempt inside the existing renderBuildAttemptStatus block, read-only"
  - "cmd/job_telemetry_test.go: the D-16 report-only guard (AST call-graph walk from the four real selection entry points -- resolveCasteModel, queenApplyJudgement, deriveVerificationScope, composeBuildManifestBrief -- to a telemetry read), proven both to pass against the real package and to bite against a fixture call path"
affects: [201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 26624
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Measured/source discipline reused a second layer down: jobTelemetrySegment mirrors verificationScope's own shape (cmd/verification_scope.go) -- a boolean plus a required plain-English string that means 'instrumentation point' when true and 'reason nothing was captured' when false. No segment is ever computed as total minus the others; a dedicated AST test parses cmd/job_telemetry.go itself and refuses the file to contain any subtraction operator at all."
    - "Merge-on-write across two process invocations, never a blind overwrite: build and continue write the same attempt-bound telemetry file from two separate cmd process invocations. jobTelemetryMergeRecords (cmd/codex_continue.go) reads whatever exists first and keeps any measured segment the incoming write did not itself measure, so continue's verification-only write can never silently erase build's own queue/context/work measurements. This is a selection between two independently measured facts, never an average and never a derivation."
    - "One record per attempt, not one per job: jobTelemetryRecord is written once per build attempt (attempts/<id>/telemetry.json, a new attempt-bound artifact kind alongside claims/verification). A build whose dispatches carry exactly one distinct job name uses it (jobTelemetryOneJobName); a build with zero or more than one leaves JobName empty rather than guessing which job the build-level segments belong to."
    - "Instrumentation is timestamps and a JSON write only -- no new dispatch, no new checkpoint, no new pause. Proven directly rather than assumed: TestInstrumentationAddsNoDispatchOrPause drives a real one-job build plus check and asserts the dispatch count and the attempt identifier are unchanged throughout."

key-files:
  created:
    - cmd/job_telemetry.go
    - cmd/job_telemetry_test.go
  modified:
    - cmd/attempt_artifacts.go
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/spend_cost_line.go
    - cmd/spend_cost_line_test.go
    - cmd/status.go

key-decisions:
  - "attemptArtifactKindTelemetry was registered in cmd/attempt_artifacts.go's legacyAttemptArtifactNames map (a file outside this plan's own declared Task 1 file list, cmd/job_telemetry.go/cmd/job_telemetry_test.go) because attemptBoundArtifactPath refuses any kind it does not already know about, and the plan's own Task 1 action explicitly said to write through attemptBoundArtifactPath. Telemetry has no genuine legacy colony-wide filename (it never existed before this plan); its registered 'legacy' name (job-telemetry.json) will simply never be found on a real colony, letting it reuse the identical validation path rather than a second one. A small, necessary deviation (Rule 3), not a redesign."
  - "The eight segments' genuine observability was assessed against the real architecture before writing a line of instrumentation, not assumed from the plan's prose. Worker dispatch and execution on the primary wrapper-driven build lane happen in an external process (the Claude Code Task tool) this Go binary never observes; only the direct/native lane (executeCodexBuildDispatches, used by Codex CLI's native dispatch and by every test in this codebase that drives a real build synchronously) genuinely executes workers inside this process. Queue, context, and work are measured there; preflight, model, tool-call, and wait are recorded unmeasured with a stated reason, because the platform reports total tokens per worker but no per-call duration, and this lane observes no first-response or external-wait boundary today. Verification is measured in codex_continue.go, the one place both continue lanes share (runDeterministicFloor)."
  - "loadLatestBuildAttempt's first return value is the attempt's store-relative FILE PATH, not its bare identifier -- a genuine bug caught by TestRealBuildProducesMeasuredSegments failing after the first implementation attempt (the merge write was building an attempt-bound path out of a full file path and being refused as 'not a bare identifier'). Fixed by reading the loaded record's own .ID field instead, matching cmd/spend_cost_line.go's existing attempt.ID usage one function over."
  - "Two of plan 201-06's own tests (TestElapsedTimeComesFromTheAttemptTimestamps, TestMissingTimestampRendersTheUnmeasuredSentinel) asserted the Elapsed line's exact text with no telemetry suffix, because no telemetry existed when they were written. 201-06's own renderSpendCostLineBlock doc comment explicitly reserved this exact extension point for plan 201-12 by number. Their expected strings were updated to include the new ' (timing breakdown not measured)' suffix that now always accompanies an attempt with no telemetry record -- a deliberate, plan-documented evolution of a shared line's text, not scope creep into an unrelated test."

requirements-completed: []
# WORK-08 (this plan's own declared requirement) is ALSO declared by 201-13
# and 201-14, neither of which has a SUMMARY.md yet -- per the shared-ID
# gate (#2388) it stays open until every plan declaring it has finished.
# CEC-06 is ALSO declared by 201-02/04/06/07 (all complete) and 201-15
# (not yet complete) -- it stays open for the same reason until 201-15
# finishes. Confirmed by reading every sibling 201-*-PLAN.md's own
# `requirements:` frontmatter field directly (the installed gsd-tools'
# `requirements ready-ids` subcommand is unavailable in this environment --
# an older get-shit-done package version, not the newer GSD Core the
# workflow instructions assume).

coverage:
  - id: D1
    description: "jobTelemetryRecord carries exactly eight named segments (queue, preflight, model, tool_call, context, work, verification, wait), each a measured duration naming its instrumentation source or an explicit reasoned-unmeasured state; no segment is ever derived from the total or from another segment; an all-unmeasured record writes no file; a storage failure is reported and swallowed, never fatal"
    requirement: WORK-08
    verification:
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestUnmeasuredSegmentIsNeverDerived"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestSegmentRequiresAnInstrumentationSource"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestAllUnmeasuredRecordWritesNoFile"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestTelemetryStorageFailureIsNonFatal"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestJobTelemetryJSONRoundTrip"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestJobTelemetryKeyedByAttemptAndJob"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real one-job build plus check produces a telemetry record with queue, context, work, and verification measured (each naming its source) and every unmeasured segment carrying a stated reason; instrumentation adds no worker dispatch and no owner-pause boundary, proven against the same fixture's dispatch count and attempt identity across build and continue"
    requirement: WORK-08
    verification:
      - kind: integration
        ref: "cmd/job_telemetry_test.go#TestRealBuildProducesMeasuredSegments"
        status: pass
      - kind: integration
        ref: "cmd/job_telemetry_test.go#TestInstrumentationAddsNoDispatchOrPause"
        status: pass
    human_judgment: false
  - id: D3
    description: "Each closeout renders exactly one timing line naming the total and the largest measured segment, or plainly states the breakdown was not measured; the status drill-down renders all eight segments for a selected attempt, read-only; a report-only guard proves none of the four real selection entry points (model, team, test-scope, brief) ever reaches a telemetry read, and bites against a fixture call path that does"
    requirement: WORK-08
    verification:
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestCloseoutRendersOneTimingLine"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestStatusDrillDownRendersAllEightSegments"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestTelemetryIsReportOnly"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestElapsedTimeComesFromTheAttemptTimestamps"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestMissingTimestampRendersTheUnmeasuredSentinel"
        status: pass
    human_judgment: false
  - id: D4
    description: "Timing is attempt-bound: the record's AttemptID equals the exact build attempt the evidence and cost figures already use"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestJobTelemetryKeyedByAttemptAndJob"
        status: pass
      - kind: integration
        ref: "cmd/job_telemetry_test.go#TestRealBuildProducesMeasuredSegments"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 12: Measured Job Timing, an Honest Unmeasured Sentinel Summary

**An eight-segment, attempt-bound timing record now measures queue/context/work on the direct build lane and verification on the shared continue path -- each segment naming either a real instrumentation point or a stated reason it could not be observed -- surfaced as one closing timing line, a full status drill-down, and a report-only guard that bites.**

## Performance

- **Duration:** 55 min (approx.)
- **Started:** 2026-09-10T17:03:33Z (approx.)
- **Completed:** 2026-09-10T17:21:26Z
- **Tasks:** 3
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- `jobTelemetryRecord` (`cmd/job_telemetry.go`, new) carries the eight named segments D-13 requires -- `queue`, `preflight`, `model`, `tool_call`, `context`, `work`, `verification`, `wait` -- each a `jobTelemetrySegment` that is either a measured duration naming the instrumentation point that captured it (`newMeasuredJobTelemetrySegment`, refusing an offer with no source) or an explicit reasoned admission nothing was captured (`unmeasuredJobTelemetrySegment`, refusing an offer with no reason). `RenderedDuration()` is the one safe reader: an unmeasured segment always renders the `"unmeasured"` sentinel, never Go's zero-value duration. A dedicated AST test parses the file itself and refuses any subtraction operator to exist at all -- structural proof no segment is ever derived from the total or from another segment. A record with nothing measured writes no file.
- `cmd/codex_build.go`'s direct/native build lane measures the three segments genuinely observable inside one process invocation: **queue** (job planning complete to brief assembly start), **context** (`prepareBuildWorkerBriefFiles`), and **work** (`executeCodexBuildDispatches`, the synchronous worker dispatch-and-wait loop). Everything else this record names -- preflight, model, tool-call, wait -- is recorded unmeasured with a plain-English stated reason, because the platform reports token totals but no per-call duration, and this lane observes no first-response or external-wait boundary today. The write happens once, at the end of a successful build, alongside the existing non-blocking per-worker spend-row write, bound to the same attempt identifier.
- `cmd/codex_continue.go`'s `runCodexContinueVerification` measures **verification** around the one shared `runDeterministicFloor` call both continue lanes already use, then merges it into whatever the build lane already recorded for the same attempt (`jobTelemetryMergeRecords`) rather than overwriting it -- build and continue are separate process invocations writing the same attempt-bound file, and a naive overwrite would have silently erased the build's own measurements.
- `cmd/spend_cost_line.go`'s existing attempt-bound `Elapsed:` line (plan 201-06's own reserved extension point) now also names the largest measured segment and its duration, or plainly states the breakdown was not measured -- still exactly one block, one timing line. `cmd/status.go`'s existing per-attempt status block gained a full eight-segment drill-down, read-only, naming unmeasured segments as such.
- D-16's report-only guarantee is enforced by an AST call-graph walk (`cmd/job_telemetry_test.go`), not a comment: it resolves the real four selection entry points in this codebase -- `resolveCasteModel` (model), `queenApplyJudgement` (team), `deriveVerificationScope` (test scope), `composeBuildManifestBrief` (brief) -- and proves none of their call graphs reaches `readJobTelemetryRecord`, while a temporary fixture call path proves the guard actually fires and names the offending function and file position.
- All three tasks followed RED-GREEN TDD, each a `test(...)` commit that failed to build/pass before the corresponding `feat(...)` commit made it pass -- confirmed for real by temporarily reverting each task's production files and re-running the tests before restoring them, not merely asserted.

## Task Commits

1. **Task 1: Define the eight-segment record with an honest unmeasured sentinel** - `99580dee` (test, RED) + `499c190a` (feat, GREEN)
2. **Task 2: Capture the segments at real instrumentation points** - `c8fa78d1` (test, RED) + `351a2fe9` (feat, GREEN)
3. **Task 3: Render one timing line, a drill-down, and keep it report-only** - `2fc5258f` (test, RED) + `b5b520d0` (feat, GREEN)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/job_telemetry.go` (new) - `jobTelemetryRecord`, `jobTelemetrySegment`, the measured/unmeasured constructors, `jobTelemetryCapture`, `newJobTelemetryRecord`, `writeJobTelemetryRecord`/`readJobTelemetryRecord`, `attemptArtifactKindTelemetry`
- `cmd/job_telemetry_test.go` (new) - all nine plan-declared tests plus three supporting tests (JSON round trip, attempt/job keying) and the D-16 AST guard machinery
- `cmd/attempt_artifacts.go` - registers `attemptArtifactKindTelemetry` in `legacyAttemptArtifactNames`
- `cmd/codex_build.go` - queue/context/work capture and non-fatal telemetry write in the direct build lane; `jobTelemetryOneJobName`
- `cmd/codex_continue.go` - verification capture around `runDeterministicFloor`; `recordCodexContinueVerificationTelemetry`, `jobTelemetryMergeSegment`, `jobTelemetryMergeRecords`
- `cmd/spend_cost_line.go` - `renderJobTelemetryClosingLine`; `renderSpendCostLineBlock` gained a `telemetry *jobTelemetryRecord` parameter
- `cmd/spend_cost_line_test.go` - updated two 201-06 tests' expected Elapsed-line text for the new suffix
- `cmd/status.go` - `renderJobTelemetryDrillDown`, wired into `renderBuildAttemptStatus`

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: `attemptArtifactKindTelemetry` was registered in `cmd/attempt_artifacts.go` (a small, necessary deviation outside Task 1's declared file list, required by the plan's own instruction to write through `attemptBoundArtifactPath`); genuine observability of each segment was assessed against the real architecture (only the direct/native build lane executes workers inside this process) before any instrumentation was written; `loadLatestBuildAttempt`'s first return value is a file path, not a bare ID, a bug caught and fixed by the plan's own required test; and two of plan 201-06's own tests were updated to match the exact extension its own doc comment reserved for this plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Registered a new attempt-bound artifact kind outside Task 1's declared file list**
- **Found during:** Task 1, writing `writeJobTelemetryRecord`
- **Issue:** The plan's own action text says to write the record "through `attemptBoundArtifactPath`", but that function refuses any artifact kind not already registered in `cmd/attempt_artifacts.go`'s `legacyAttemptArtifactNames` map -- a file outside Task 1's declared scope (`cmd/job_telemetry.go`, `cmd/job_telemetry_test.go`).
- **Fix:** Added `attemptArtifactKindTelemetry = "telemetry"` and a corresponding (never-realized) legacy filename entry to `legacyAttemptArtifactNames`, reusing the exact same validation path claims/verification already use rather than a second one.
- **Files modified:** `cmd/attempt_artifacts.go`
- **Verification:** `go build ./...` clean; `TestArtifactsAreAttemptBound`/`TestLegacyArtifactReadIsValidatedIdentically`/`TestArtifactPathRefusesEscape` (pre-existing, unmodified) still pass.
- **Committed in:** `499c190a` (Task 1 feat commit)

**2. [Rule 1 - Bug] `loadLatestBuildAttempt`'s first return value used as a bare attempt ID**
- **Found during:** Task 2, `TestRealBuildProducesMeasuredSegments` failing with "must be a bare identifier, not a path"
- **Issue:** `recordCodexContinueVerificationTelemetry`'s first implementation used `loadLatestBuildAttempt`'s first return value as the attempt identifier; that value is the attempt's store-relative FILE PATH, not its bare ID.
- **Fix:** Read the loaded `buildAttemptRecord`'s own `.ID` field instead, matching `cmd/spend_cost_line.go`'s existing usage one function over.
- **Files modified:** `cmd/codex_continue.go`
- **Verification:** `TestRealBuildProducesMeasuredSegments` and `TestInstrumentationAddsNoDispatchOrPause` both pass.
- **Committed in:** `351a2fe9` (Task 2 feat commit)

**3. [Rule 1 - Bug] Two pre-existing 201-06 tests asserted the old (pre-telemetry) Elapsed-line text**
- **Found during:** Task 3, running the existing spend-cost-line regression sweep after wiring `renderJobTelemetryClosingLine`
- **Issue:** `TestElapsedTimeComesFromTheAttemptTimestamps` and `TestMissingTimestampRendersTheUnmeasuredSentinel` asserted the Elapsed cell's exact text with no suffix -- correct when they were written (before telemetry existed), but 201-06's own doc comment explicitly reserved this exact extension point for this plan by number.
- **Fix:** Updated both tests' expected strings to include the new `" (timing breakdown not measured)"` suffix that now accompanies any attempt with no telemetry record, with a comment explaining why.
- **Files modified:** `cmd/spend_cost_line_test.go`
- **Verification:** Both tests pass with the updated expectations; the dash sentinel itself is unchanged (still byte-identical to the unreported-cost sentinel), only the surrounding line text grew.
- **Committed in:** `2fc5258f` (Task 3 test commit, alongside the new Task 3 tests)

---

**Total deviations:** 3 auto-fixed (1 blocking file-scope extension, 1 bug, 1 bug in a sibling plan's now-superseded test expectation). **Impact on plan:** All three were necessary to satisfy the plan's own stated design (attempt-bound writes through the existing mechanism, a genuinely correct attempt identifier, and the exact line-extension 201-06 reserved). No unrelated behavior was touched.

## Issues Encountered

- `TestBuildStartLegacyHelpersRetired200` (`cmd/build_attempt_external_test.go`) fails both inside a broad regression sweep and in isolation, naming `cmd/failure_evidence_test.go:29:1` as a "legacy build-start helper" -- a file last touched in plan 201-10 (`a8ee899f`), well before this plan, and never read or modified by this plan's own tasks. Confirmed pre-existing and unrelated: this plan's own declared `<verify>` commands (all nine tests, `go build ./...`, `go vet ./cmd`) are unaffected and all pass.
- Per the executor's verification policy (targeted runs preferred over the ~20-minute full `go test ./cmd` sweep, a documented pre-existing environment ceiling), verification for this plan ran every task-level `<verify>` command individually, the full telemetry-focused test file, the pre-existing `spend_cost_line_test.go`/`status_running_total_test.go` regression suites, `TestSpendPipelineIsReachableEndToEnd`, and the plan 201-08 `work_identity_test.go` family -- all clean except the one documented pre-existing failure above. `go build ./...` and `go vet ./cmd` are both clean.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 201-13 ("set a target from the real numbers") has a real, honest per-segment record to read from `readJobTelemetryRecord` -- no invented number for any segment this plan's own architecture could not observe.
- Plan 201-14 (also declaring `WORK-08`) inherits the same measured foundation.
- Plan 201-15 (also declaring `CEC-06`) inherits the attempt-bound timing this plan added alongside the evidence and cost figures it already carries.
- The wrapper-driven ("plan-only") build lane -- the primary, documented interactive path per CLAUDE.md -- does NOT yet write its own queue/context measurements; only the direct/native lane (`executeCodexBuildDispatches`) does, since it is the only lane where worker execution happens synchronously inside this process today. A colony built exclusively through the wrapper-driven lane will see no telemetry record at all until a later plan adds capture points there, or until continue's own verification-only write creates one. This is a scope boundary, not a bug: the plan-only lane genuinely cannot observe worker execution time from inside this process (dispatch happens externally, via the Claude Code Task tool), and Task 2's own acceptance criteria were scoped to what a real one-job build can honestly measure.
- `go build ./...` and `go vet ./cmd` are clean. Every task-level `<verify>` command from `201-12-PLAN.md` passes individually.
- Ready for `201-13-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/job_telemetry.go` — FOUND
- `cmd/job_telemetry_test.go` — FOUND
- Commit `99580dee` (Task 1 RED) — FOUND in git log
- Commit `499c190a` (Task 1 GREEN) — FOUND in git log
- Commit `c8fa78d1` (Task 2 RED) — FOUND in git log
- Commit `351a2fe9` (Task 2 GREEN) — FOUND in git log
- Commit `2fc5258f` (Task 3 RED) — FOUND in git log
- Commit `b5b520d0` (Task 3 GREEN) — FOUND in git log
- `go build ./...` — clean
- `go vet ./cmd` — clean
- All plan `<verify>` commands re-run and passing: `TestUnmeasuredSegmentIsNeverDerived`, `TestSegmentRequiresAnInstrumentationSource`, `TestAllUnmeasuredRecordWritesNoFile`, `TestTelemetryStorageFailureIsNonFatal`, `TestRealBuildProducesMeasuredSegments`, `TestInstrumentationAddsNoDispatchOrPause`, `TestCloseoutRendersOneTimingLine`, `TestStatusDrillDownRendersAllEightSegments`, `TestTelemetryIsReportOnly`
