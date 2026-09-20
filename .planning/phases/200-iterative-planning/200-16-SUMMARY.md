---
phase: 200-iterative-planning
plan: 16
subsystem: planning-runtime
tags: [go, candidate-review, exact-acceptance, plan-revisions, lifecycle-transactions, owner-authority]

# Dependency graph
requires:
  - phase: 200-08
    provides: explicit-owner planning state, immutable revision lineage, candidate and acceptance contracts
  - phase: 200-15
    provides: stopped content-addressed candidates with complete evidence timelines and Queen recommendations
provides:
  - Read-only full candidate and exact iteration-card review through the public plan command
  - Exact owner acceptance commands binding candidate, specification, base revision, timeline, proposal, and action token
  - Atomic immutable PlanRevision activation with durable acceptance receipts and exact replay
  - Canonical first/current/legacy candidate base binding with completed-prefix preservation
affects: [200-17, 200-19, 200-22, iterative-planning, plan-revision-lineage, candidate-acceptance]

# Tech tracking
tech-stack:
  added: []
  patterns: [renderer-neutral candidate queries, exact frontier tokens, four-target lifecycle transaction, immutable acceptance replay]

key-files:
  created: [cmd/plan_candidate.go, cmd/plan_candidate_test.go]
  modified: [cmd/codex_workflow_cmds.go, cmd/codex_workflow_cmds_test.go, cmd/plan_revision.go, cmd/plan_revision_test.go, cmd/codex_plan_finalize.go, cmd/planning_state.go, cmd/planning_state_test.go]

key-decisions:
  - "Candidate review returns persisted evidence and the Queen recommendation unchanged; renderers do not derive either authority signal."
  - "Acceptance is one exact owner action whose deterministic token and six frontier bindings are validated before any lifecycle transaction begins."
  - "Completed lifecycle status is restored onto the newly bound candidate definition, preserving finished work while retaining the new candidate/spec/timeline authority links."
  - "Planning-stage state hashes remain stale-work guards, while candidate bases name immutable retained revision hashes; first plans use a constrained plan-unbound genesis marker."

patterns-established:
  - "Review/accept separation: read-only candidate inspection -> copied exact command -> atomic explicit-owner transition."
  - "Acceptance replay: an already accepted exact candidate returns its original revision and receipt without opening a new transaction."
  - "Candidate base translation: mutable run-header snapshot for drift detection plus immutable revision identity for lineage validation."

requirements-completed: [PLAN-03, PLAN-05, PLAN-06]

# Metrics
duration: 35m
completed: 2026-09-07
---

# Phase 200 Plan 16: Exact Candidate Review and Acceptance Summary

**Stopped planning candidates are now fully reviewable and can become active only through one exact, atomic owner acceptance that preserves immutable revision lineage and completed work.**

## Performance

- **Duration:** 35m
- **Started:** 2026-09-07T21:08:47Z
- **Completed:** 2026-09-07T21:43:32Z
- **Tasks:** 2
- **Files modified:** 9 implementation/test files

## Accomplishments

- Added `aether plan --candidate` and exact `--show-iteration N --details` operations that expose the complete proposed plan, all five scores, target versus actual confidence, stop diagnostics, causal residual gaps, semantic/authority changes, persisted Queen recommendation, and verified timeline without writing state.
- Added a copyable exact acceptance command. Missing, mixed, ambiguous, deprecated, stale, rejected, or divergent inputs fail with field-specific guidance before mutation.
- Added one lifecycle transaction that writes `COLONY_STATE.json`, the accepted candidate, the accepted stage marker, and the standalone `PlanAcceptanceReceipt`; successful activation makes the new immutable revision executable and exact retries return the original receipt.
- Preserved completed compatible phase/task lifecycle statuses while keeping the newly accepted specification, candidate, and timeline bindings, and protected every predecessor revision from mutation.
- Closed the Plan 15/16 base seam so real first, current, and legacy-derived candidates name admissible immutable bases instead of incorrectly treating a mutable plan-state snapshot as a revision hash.

## Task Commits

Each TDD task used separate RED and GREEN gates; the discovered integration correction was committed independently:

1. **Task 1 RED: Candidate review contract** - `075ed206` (test)
2. **Task 1 GREEN: Candidate review and exact input routes** - `d4b0a8ce` (feat)
3. **Task 2 RED: Exact acceptance, stale, preservation, and replay contracts** - `82e98e37` (test)
4. **Integration deviation: Admissible candidate base lineage** - `6ecba9ba` (fix)
5. **Task 2 GREEN: Atomic exact candidate activation** - `3b26535c` (feat)
6. **Post-plan gate fix: Exact deprecated-accept recovery guidance** - `7713d05` (fix)

## Files Created/Modified

- `cmd/plan_candidate.go` - Typed candidate review/detail/accept operations, artifact discovery, exact action tokens, immutable timeline verification, and public result maps.
- `cmd/plan_candidate_test.go` - Read-only review, iteration bounds, exact input, atomic activation, rejection, transaction-fault, and command-route coverage.
- `cmd/codex_workflow_cmds.go` - Public candidate review, detail, and exact acceptance flags on `aether plan`.
- `cmd/codex_workflow_cmds_test.go` - Public flag registration and deprecated bare-accept migration coverage.
- `cmd/plan_revision.go` - Exact binding validation, completed-work preservation, receipt construction, accepted-state validation, four-target lifecycle transaction, and replay.
- `cmd/plan_revision_test.go` - Completed-compatible-work and exact-replay contracts.
- `cmd/codex_plan_finalize.go` - Candidate base translation and authoritative completed-prefix inclusion at the Plan 15 producer boundary.
- `cmd/planning_state.go` - Constrained first-plan genesis validation plus current/legacy immutable base resolution.
- `cmd/planning_state_test.go` - First/current/legacy base-binding regression coverage.

## Decisions Made

- Candidate inspection selects only `pending_review` artifacts when no ID is supplied, so retained accepted history cannot make the next review ambiguous. Exact acceptance/replay still resolves a specifically named candidate.
- The public renderer returns stored recommendation, rationale, evidence IDs, and producer verbatim. It cannot synthesize a recommendation from scores or stop reason.
- A planning run header continues to bind the mutable base-state hash for stale-work detection. The stopped candidate separately binds the retained revision's immutable `PlanHash`, avoiding conflation of lifecycle status with plan identity.
- For the first plan only, `plan-unbound` is accepted as a genesis marker when the proposal is revision 1 with no parent and becomes that retained first revision. It is not a wildcard for later lineage.
- Completed work compatibility ignores lifecycle status and changing planning-authority metadata, then restores only phase/task lifecycle status and watcher count onto the newly bound proposal. Executable semantic changes remain a hard refusal.
- The acceptance receipt is independently content-addressed and binds the exact token hash, candidate/spec/base/timeline/proposal identities, actor, time, and activated revision. Replay verifies both the receipt payload and retained candidate before returning it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bound Plan 15 candidates to admissible immutable bases**

- **Found during:** Task 2 (real stopped-candidate acceptance verification)
- **Issue:** Plan 15 stored the full mutable `Plan` state hash as `BasePlanRevisionHash`, while the explicit-owner validator requires the retained base revision's immutable `PlanHash`. A first candidate also named `plan-unbound`, which the validator unconditionally required to exist as a retained revision. Therefore no real first/current candidate could satisfy Plan 16 acceptance.
- **Fix:** Kept the run header's full state hash as its stale-dispatch guard, translated the persisted candidate to an immutable current revision hash, materialized a deterministic legacy baseline when needed, constrained the first-plan genesis exception, and carried the authoritative completed prefix into replacement-only Route proposals.
- **Files modified:** `cmd/codex_plan_finalize.go`, `cmd/planning_state.go`, `cmd/planning_state_test.go`
- **Verification:** `go test ./cmd -run 'TestPlanningRouteStage|TestCandidateAcceptanceBase' -count=1` passed, followed by the combined Plan 15/16 regression run in 133.082s.
- **Committed in:** `6ecba9ba`

**2. [Rule 1 - Bug] Restored the exact candidate-ID recovery shape for deprecated `--accept`**

- **Found during:** Wave 7 post-plan regression gate
- **Issue:** Plan 16's early candidate-operation validator correctly rejected bare `--accept`, but its replacement message omitted the literal `<candidate-id>` command shape required by the established CLI recovery contract.
- **Fix:** The non-mutating migration error now tells the owner to inspect `aether plan --candidate` and run the exact `aether plan --accept-candidate <candidate-id> ...` command shown there.
- **Files modified:** `cmd/plan_candidate.go`
- **Verification:** The focused legacy-accept regression passes, and the combined Plan 15/16 suite passes 50 tests.
- **Committed in:** `7713d05`

---

**Total deviations:** 2 auto-fixed (1 Rule 3 blocking integration issue, 1 Rule 1 recovery-guidance bug)
**Impact on plan:** Both corrections are narrowly limited to the candidate producer/validator and migration-guidance seams. They make the planned acceptance journey usable without granting any new worker or automatic activation authority.

## Issues Encountered

- The initial exact-replay assertion exposed a nil-versus-empty preserved-phase slice after JSON round-trip. The activation result now uses the canonical omitted representation, so a retry returns the same revision value and byte-identical artifacts.
- The installed `requirements.mark-complete` handler does not parse this milestone's legacy bold-ID checkbox format. `PLAN-03`, `PLAN-05`, and `PLAN-06` were already checked complete, so no requirement-file edit was needed. `state.update-progress` likewise found no Markdown-body progress field; `state.advance-plan` updated the structured completed-plan count and the roadmap correctly advanced to 16/25.
- No package, authentication, or external-service gates were encountered.

## Known Stubs

None.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: authority_transition | `cmd/plan_candidate.go`, `cmd/plan_revision.go` | The public plan command now has a state-changing owner-acceptance path. Exact token/frontier validation, approved-spec checks, current-base/timeline/proposal verification, fail-before-write refusals, and one allowlisted lifecycle transaction constrain the surface. |

## User Setup Required

None - no external service configuration required.

## Verification

- `go test ./cmd -run 'TestPlanCandidate.*Review|TestPlanCandidate.*Inputs|TestPlanCandidate.*Details|TestPlanCommand.*Candidate' -count=1` - passed after Task 1.
- `go test ./cmd -run 'TestPlanCandidate.*Accept|TestPlanRevision.*Candidate|TestPlanRevision.*Replay|TestPlanRevision.*Stale' -count=1` - passed after Task 2.
- `go test ./cmd -run 'TestPlanningRouteStage|TestCandidateAcceptanceBase|TestPlanCandidate|TestPlanRevision.*Candidate|TestPlanCommand.*Candidate' -count=1` - passed in 133.082s as the combined Plan 15/16 regression run.
- `go test ./cmd -run '^TestPlanAcceptWithExistingPlanFailsLoudly$' -count=1` - passed after the Wave 7 recovery-guidance fix.
- The combined Plan 15/16 regression command above was rerun after `7713d05` and passed all 50 selected tests.

## Next Phase Readiness

- Plan 200-17 can treat the active accepted revision as the exact base for subsequent planning and successor-specification reconciliation.
- Plan 200-19 can render the typed candidate review, recommendation, exact command, acceptance receipt, and replay result without recreating authority logic.
- Plan 200-22 can document a real copy-review-accept workflow backed by stable public flags and field-specific refusal guidance.
- No Plan 16 implementation blocker remains.

## Self-Check: PASSED

- All nine implementation/test files and this summary exist.
- All six RED/GREEN/deviation/post-plan fix commits are present in repository history.
- The required Plan 16 tests and combined Plan 15/16 regression suite pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
