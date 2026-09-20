---
phase: 200-iterative-planning
plan: 48
subsystem: build-lifecycle
tags: [go, partial-credit, coherent-retry, idempotent-replay, receipts, tdd]

# Dependency graph
requires:
  - phase: 200-45
    provides: repository-session mutation authority for lifecycle writes
  - phase: 200-47
    provides: canonical build-attempt paths and receipt-backed completion binding
provides:
  - exactly one durable unfinished-only child attempt for every successful partial finalize
  - identical mutation-free replay projection from verified parent, child, and receipt evidence
  - fail-closed handling for recovery-start faults and missing, forged, duplicate, changed, or exhausted recovery evidence
affects: [build-finalize, coherent-jobs, partial-recovery, build-replay, final-gate]

# Tech tracking
tech-stack:
  added: []
  patterns: [plan-before-state partial finalize, receipt-bound child evidence, persisted owner recovery projection]

key-files:
  created: []
  modified:
    - cmd/build_attempt.go
    - cmd/coherent_job_retry.go
    - cmd/codex_build_finalize.go
    - cmd/build_attempt_external_test.go
    - cmd/build_finalize_partial_replay_test.go

key-decisions:
  - "Partial finalize reports success only after canonical build start has durably committed exactly one unfinished-only recovery child; a child-start fault returns an error while retaining honest credited evidence."
  - "Partial parents and children persist the exact unfinished task IDs and redispatch command; replay projects those persisted values and uses pure derivation only to detect tampering."
  - "A recovery child is executable evidence only while prepared and bound to a valid canonical start receipt; missing, forged, duplicate, or exhausted evidence fails closed without repair writes."
  - "A current partial manifest may reach the read-only replay verifier before stale-state checks, but completion digest, accepted task graph, child journal, and receipt checks still gate every result."

patterns-established:
  - "Honest partial terminal boundary: validate the recovery plan before state credit, then require the canonical child transaction before rendering success."
  - "Exactly-once replay: return owner-facing command and IDs only from verified durable parent/child evidence, with zero replay mutation."

requirements-completed: [CEC-03, PLAN-05, PLAN-06]

# Metrics
duration: 31min
completed: 2026-09-09
---

# Phase 200 Plan 48: Durable Partial Finalize and Replay Summary

**Partial build credit now succeeds only with one receipt-backed retry child for the unfinished tasks, and an identical completion replay returns the same command and IDs without writing a byte.**

## Performance

- **Duration:** 31 min
- **Started:** 2026-09-09T11:47:47Z
- **Completed:** 2026-09-09T12:18:49Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Normalized the repository-session state passed to canonical child start, so a valid four-of-six partial completion creates one prepared child for exactly tasks five and six.
- Turned child-start failure into an honest finalize error instead of a warning-success, while preserving the four tasks whose completion was already durably proven.
- Persisted exact recovery task IDs and the owner-facing retry command on both the partial parent and canonical child.
- Made replay verify one prepared child, its accepted dispatch plan, and its receipt-bound current bytes before returning the persisted recovery projection.
- Added mutation-fingerprint coverage and fail-closed cases for changed packets and missing, forged, duplicate, or exhausted recovery evidence.

## Task Commits

Each TDD gate and implementation unit was committed atomically:

1. **Task 1 RED A: Expose successful partial finalize without a recovery child** - `b6dbfb85` (test)
2. **Task 1 GREEN A: Create the canonical unfinished-only retry child** - `d881ec1b` (fix)
3. **Task 1 RED B: Inject a canonical child-start transaction fault** - `085cf924` (test)
4. **Task 1 GREEN B: Refuse success when recovery start fails** - `049ea134` (fix)
5. **Task 2 RED: Define durable identical replay and hostile evidence cases** - `0d05cfcc` (test)
6. **Task 2 GREEN: Verify and project receipt-backed recovery evidence** - `fb45684b` (fix)

## Files Created/Modified

- `cmd/build_attempt.go` - Persists exact recovery task IDs and command on partial parents and coherent children, and routes current partial manifests to read-only replay verification.
- `cmd/coherent_job_retry.go` - Starts child attempts through canonical build start and verifies unique, prepared, receipt-bound recovery evidence.
- `cmd/codex_build_finalize.go` - Validates recovery before partial credit, refuses recovery-start failure, and reconstructs replay solely from verified durable evidence.
- `cmd/build_attempt_external_test.go` - Proves four-of-six recovery linkage, latest-pointer behavior, and injected child-start failure semantics.
- `cmd/build_finalize_partial_replay_test.go` - Proves identical replay, whole-data-tree byte stability, and fail-closed hostile recovery cases.

## Decisions Made

- Kept D-16 accepted-plan authority intact: the recovery child still enters through `commitBuildStart`, which reloads authority inside the repository mutation session.
- Kept successful partial credit and the child as ordered durable transactions. If the second transaction faults, finalize returns non-success with exact same-packet repair guidance and does not erase already proven completed work.
- Stored the exact unfinished task list beside the exact command on parent and child. Recomputing the plan is a consistency check only; it cannot silently replace owner-facing durable authority during replay.
- Required exactly one prepared child whose current bytes match its canonical build-start receipt. Suspicious or spent evidence cannot cause replay to mint a replacement or report a usable recovery route.
- Allowed a current partial attempt to bypass only the preliminary stale-state comparison so its packet reaches the read-only verifier; different digests and forged evidence remain explicit refusals.

## Deviations from Plan

None - the state-baseline mismatch, warning-success path, and incomplete replay evidence were the planned regression and were repaired inside the specified files and existing transaction/session boundaries.

## Issues Encountered

- The first RED exposed that the child transaction compared normalized repository state with an unnormalized in-memory baseline, producing `build start: state baseline is stale`. Normalizing the passed baseline to the same legacy-state representation restored canonical start without adding a writer.
- The second RED confirmed the finalizer downgraded an injected child-start transaction failure to a warning and apparent success. The finalizer now returns that cause and same-packet repair guidance.
- Receipt dispatch comparison initially treated JSON-equivalent nil and empty slices as different Go values. Comparing their canonical JSON digests retained strict dispatch verification without rejecting canonically persisted evidence.
- The installed `state.update-progress` helper found no Markdown-body progress field, while `state.advance-plan` correctly updated structured progress and advanced to Plan 49. Its serialization also dropped established phase/provenance frontmatter, which was restored narrowly before commit.
- The installed requirement marker does not parse this milestone's legacy bold-ID checkbox format. `CEC-03`, `PLAN-05`, and `PLAN-06` were already checked complete, so no requirements bytes needed changing.

## TDD Gate Compliance

- **Task 1 RED A (`b6dbfb85`):** four credited tasks produced no child attempt because canonical recovery start rejected a stale state baseline, yet finalize returned success.
- **Task 1 GREEN A (`d881ec1b`):** the exact gate passed with one prepared child for only the two unfinished tasks and the parent left as the latest dispatched attempt.
- **Task 1 RED B (`085cf924`):** an injected `build-start-before-commit` fault proved recovery failure still returned a successful partial result.
- **Task 1 GREEN B (`049ea134`):** the exact gate passed with a nonzero wrapped fault, no child, and the four proven task completions retained honestly.
- **Task 2 RED (`0d05cfcc`):** the child retained a generic force command, replay guidance diverged, and hostile child states lacked the required fail-closed contract.
- **Task 2 GREEN (`fb45684b`):** identical replay returns the same persisted projection with zero writes, while changed, missing, forged, duplicate, and exhausted evidence is refused.

## Verification

- `go test ./cmd -run '^Test(GroupedJobPartialRetryIsAppendOnlyExternal|PartialRetryFailureIsNeverSuccessful200)$' -count=1` - PASS (`ok`, 5.615s).
- `go test ./cmd -run '^Test(PartialFinalizeCanBeReplayedWithTheSamePacket|PartialFinalizeReplayWritesNothing|PartialFinalizeRejectsDifferentPacket200)$' -count=1` - PASS (`ok`, 9.892s).
- `go test ./cmd -run '^Test(GroupedJobPartialRetry|PartialFinalize|PartialBuild)' -count=1` - PASS (`ok`, 30.282s).
- `go test ./cmd -run '^Test(CoherentJobRetry|BuildAttemptChildLinksParent|PartialRetryCommand|PartialRetryAttempt)' -count=1` - PASS (`ok`, 38.387s).
- `go test ./cmd -run '^TestPartialFinalize' -count=1` - PASS (`ok`, 10.280s).
- `go test ./cmd -run '^Test(BuildAttemptExternalFixturesUseCanonicalTransaction200|BuildAttemptFixtureUsesCanonicalTransaction)$' -count=1` - PASS (`ok`, 10.051s).
- `gsd_run query verify.key-links .planning/phases/200-iterative-planning/200-48-PLAN.md --raw` - PASS (`valid`).
- `git diff --check` - PASS for all Plan 48 implementation and test commits.

## Known Stubs

None.

## Threat Flags

None. The changed completion-to-credit and partial-parent-to-child boundaries are exactly T-200-48-01 and T-200-48-02 from the plan. Completed IDs remain validated against accepted task evidence, and recovery linkage is persisted and receipt-verified. No package, endpoint, credential path, schema, or new filesystem authority was introduced.

## User Setup Required

None - no dependency, credential, service, or configuration change is required.

## Next Phase Readiness

- Plan 49 can proceed with a truthful partial-finalize foundation: successful partial results always expose one executable unfinished-only recovery job.
- Replay is now stable and read-only, so later final-gate work can distinguish valid retries from forged or exhausted journal evidence.
- Plan 48 does not alter Queen-led execution, team sizing, turnaround policy, or the Phase 201 recovery UX.

## Self-Check: PASSED

- All five modified implementation/test files and this summary exist.
- TDD commits `b6dbfb85`, `d881ec1b`, `085cf924`, `049ea134`, `0d05cfcc`, and `fb45684b` exist in repository history.
- Both required key-link families resolve, all six scoped verification commands pass, and the Plan 48 commit range deletes no tracked files.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` were neither staged nor changed by Plan 48.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
