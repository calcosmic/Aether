---
phase: 200-iterative-planning
plan: 07
subsystem: planning-authority
tags: [go, owner-authority, decision-cards, content-addressing, resume-validation, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Typed planning evidence, immutable iteration-card identity, semantic IDs, and specification/plan revision authority contracts
provides:
  - Pure materiality classification for behavior, authority, risk-tolerance, scope, and acceptance-meaning decisions
  - Stable first-pass batching plus completed Scout/Route-Setter/card boundaries for later owner pauses
  - Evidence-first decision cards, exact scoped answer reuse, closed resolution dispositions, and tamper-evident resume tokens
affects: [discuss, specification-revision, planning-orchestrator, route-setter, planning-renderers]

# Tech tracking
tech-stack:
  added: []
  patterns: [closed owner-decision domains, complete-pass boundary receipts, exact semantic equivalence keys, content-addressed idempotent resume]

key-files:
  created:
    - cmd/planning_decision.go
    - cmd/planning_decision_test.go
  modified: []

key-decisions:
  - "Only evidence-unresolved choices in behavior, authority, risk tolerance, scope, or acceptance meaning may interrupt the owner; research depth, worker mechanics, scoring, formatting, and generic preferences remain autonomous."
  - "Pass one may batch decisions immediately after Scout, while every later pause requires matching Scout and Route-Setter receipts plus the persisted iteration-card hash for the complete pass."
  - "Prior answers are reusable only under an exact content-addressed key spanning goal, session, approved specification revision, base plan revision, normalized meaning, contract impacts, and affected semantic IDs."
  - "Only contract-equivalent answers return direct_resume; any changed contract or affected specification item returns successor_spec_required with stable IDs and normalized revision evidence."

patterns-established:
  - "Decision boundary pattern: classify all candidates first, discard autonomous/evidence-answerable choices, then canonicalize every material candidate into one stable batch at a proven pass boundary."
  - "Authority reuse pattern: compare the complete semantic key, surface a drifted prior answer as evidence only, and never silently promote it to current authorization."
  - "Resume pattern: bind batch, frontier-or-card proof, canonical answers, disposition, affected IDs, and revision evidence into one deterministic token; exact retries pass idempotently and stale/divergent values fail without state mutation."

requirements-completed: [CEC-03, PLAN-04]

# Metrics
duration: 27 min
completed: 2026-09-07
---

# Phase 200 Plan 07: Material Decision Boundary Summary

**Planning now interrupts the owner only for evidence-unresolved contract choices, presents those choices as complete evidence-first batches, and resumes only from an exact goal/session/revision-bound token.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-09-07T14:42:01Z
- **Completed:** 2026-09-07T15:08:10Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added closed decision-domain and classification contracts that suppress generic preferences, evidence-answerable questions, routine research depth, worker mechanics, scoring, and plan-formatting choices.
- Added deterministic all-at-once batching after the first Scout pass and required matching Scout, Route-Setter, and persisted iteration-card receipts before any later-pass owner interruption.
- Added evidence-first decision-card projections with the exact decision, why-now explanation, cited evidence, Queen recommendation, viable-choice consequences, affected scope, prior-answer status, revalidation reason, and resume instruction.
- Added content-addressed equivalence keys across goal, session, approved specification, base plan, normalized meaning, behavior, authority, scope, risk, acceptance, and affected semantic IDs.
- Added closed `direct_resume` and `successor_spec_required` outcomes, rejecting contract changes without stable specification anchors and preserving normalized revision evidence for the successor draft.
- Added deterministic resume tokens that bind the batch, one exact planning-frontier proof, canonical answers, resolution disposition, and successor evidence; exact retries are idempotent while stale or altered tokens return recovery guidance without mutation.

## Task Commits

Each TDD task was committed through a failing-test commit followed by its implementation:

1. **Task 1: Classify and batch only material owner decisions** — `041855df` (test/RED), `83c5d838` (feat/GREEN)
2. **Task 2: Build evidence-first cards and scoped answer reuse** — `5e1b3b3c` (test/RED), `5723edbe` (feat/GREEN), `66037a7c` (race fix), `0bb53286` (successor-anchor hardening)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/planning_decision.go` — Material-domain classification, admissible-evidence suppression, stable complete-pass batching, evidence-first card projection, exact answer-equivalence keys, resolution dispositions, revision evidence, and pure content-addressed resume issuance/validation.
- `cmd/planning_decision_test.go` — Coverage for autonomous boundaries, first/later-pass batching, every card field, all answer-key drift dimensions, direct-versus-successor resolution, stable affected IDs, exact retry, wrong-session staleness, token alteration, and frontier/card exclusivity.

## Decisions Made

- Kept materiality explicit and typed. A decision requires one of the five owner domains plus a concrete contract impact or affected semantic item; unknown domains fail closed, while four routine planning domains are explicitly autonomous.
- Used the current `PlanningEvidenceRef` validation contract before evidence can suppress or support a material prompt. Only cited evidence that is both fresh and admissible can answer a candidate autonomously.
- Allowed the first pass to pause at its special post-Scout boundary, but required later passes to bind both stage receipts into the exact persisted iteration card before yielding owner work.
- Normalized meaning and impact text for equivalence while sorting semantic IDs, evidence, decisions, choices, answers, and revision evidence before hashing. Discovery order therefore cannot alter authority.
- Treated a prior answer with any scope or meaning drift as evidence to display, never permission to reuse. The complete key hash must validate on both the current and prior record before reuse succeeds.
- Made successor-specification anchors mandatory: contract-impact drift without exact affected semantic IDs is an error rather than an unrouteable revision request.
- Kept resume validation pure. It reconstructs and compares content addresses, classifies valid-but-old boundaries as stale and tampered/same-boundary disagreements as divergent, and never advances a stage or rewrites planning state.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed shared state from parallel reuse tests**

- **Found during:** Overall race verification after Task 2
- **Issue:** Parallel equivalence-drift subtests captured and wrote the parent test's shared `err` variable, producing a test-only data race even though the policy implementation was pure.
- **Fix:** Gave every subtest its own hash error and assigned the computed digest locally.
- **Files modified:** `cmd/planning_decision_test.go`
- **Verification:** `go test -race ./cmd -run TestPlanningDecision -count=1` passes.
- **Committed in:** `66037a7c`

**2. [Rule 2 - Missing Critical Functionality] Required stable anchors for successor specifications**

- **Found during:** Final Task 2 contract audit
- **Issue:** A changed contract impact could initially produce `successor_spec_required` without identifying which stable specification items the next draft must reconcile.
- **Fix:** Rejected contract-changing answers without affected semantic IDs, canonicalized choice order, and validated successor revision-evidence records.
- **Files modified:** `cmd/planning_decision.go`, `cmd/planning_decision_test.go`
- **Verification:** The new negative fixture and the complete normal/race decision suite pass.
- **Committed in:** `0bb53286`

---

**Total deviations:** 2 auto-fixed (1 test concurrency bug, 1 missing successor-specification safety invariant)
**Impact on plan:** Both changes strengthen the requested determinism and fail-closed authority boundary. No production file outside the two plan-owned files changed.

## Issues Encountered

- Repository-wide `go test ./...` reached the same protected Phase 199 bookkeeping failures recorded before this plan: `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified` finds one unclassified `legacy_pause` and one unclassified `legacy_resume` byte in `199-UAT.md`, while `TestPhase199GateReceiptSchema` and `TestPhase199GateReceipt` reject stale execution timestamps.
- Repository-wide `go test -race ./...` reached the existing ten-minute `cmd` full-suite timeout under integration-test contention. The exact Plan 200-07 race suite passes immediately in isolation, as do all packages outside `cmd` in the broad run.
- Those Phase 199 items were already recorded in `.planning/phases/200-iterative-planning/deferred-items.md`; protected Phase 199 artifacts were left untouched.
- The installed requirements helper did not recognize this milestone's legacy `**ID — title:**` checkbox format. `CEC-03` was already complete, and `PLAN-04` was checked directly using the established narrow Phase 200 tracking fallback.
- The unrelated changes in `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`041855df`) failed on the absent classification, domain, batching, and boundary API before GREEN (`83c5d838`) made all classification and complete-pass boundary fixtures pass.
- Task 2 RED (`5e1b3b3c`) failed on the absent card, equivalence, resolution, and resume API before GREEN (`5723edbe`) made the card/reuse/resume fixtures pass.
- Race verification then exposed the test-only shared-variable defect fixed in `66037a7c`; final contract review added the stable-ID negative fixture with `0bb53286`.
- Every RED commit precedes its corresponding production implementation in repository history.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 08's specification engine can consume `successor_spec_required` together with exact affected semantic IDs and normalized revision evidence.
- Plans 11, 14, and 15 can share one materiality/card/reuse policy instead of allowing discuss, Scout, or Route-Setter to invent separate authority rules.
- Planning renderers can show the card model in its locked evidence-first order without owning classification, hashing, or resume authority.
- No Plan 200-07 blocker remains.

## Self-Check: PASSED

- Both plan-owned Go files and this summary exist.
- All six implementation/TDD/fix commits exist in the required RED→GREEN order.
- Every `TestPlanningDecision` fixture passes normally and under the race detector; `go vet ./cmd` and whitespace checks pass.
- Materiality suppression, stable first/later-pass batching, evidence-first cards, all equivalence-drift dimensions, direct/successor resolution, exact retry, stale scope, altered tokens, and successor stable-ID anchoring each have direct regression coverage.
- Stub and threat-surface scans found no placeholder implementation, incomplete UI data path, new endpoint, authentication path, file-write path, external network boundary, or unplanned schema trust boundary.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
