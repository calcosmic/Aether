---
phase: 200-iterative-planning
plan: 03
subsystem: planning-evidence
tags: [go, sha256, provenance, freshness, path-containment]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Typed planning evidence references, source kinds, five confidence dimensions, and scope identifiers
  - phase: 200-iterative-planning
    plan: 02
    provides: Validated current planning lineage and authority-neutral legacy evidence indexing
provides:
  - Deterministic content-addressed evidence records for all eight allowed planning source kinds
  - Validate-before-read repository containment with typed refusal outcomes and redacted projections
  - Exact frontier freshness across goal, session, specification, base plan, source hash, and cited-card state
  - Per-dimension applicability policies and exact owner-answer equivalence checks
affects: [planning-loop, scout-research, route-setter-scoring, owner-decisions, planning-timeline]

# Tech tracking
tech-stack:
  added: []
  patterns: [content-addressed evidence identity, validate-before-read containment, explicit frontier freshness, kind-by-dimension applicability]

key-files:
  created:
    - cmd/planning_evidence.go
    - cmd/planning_evidence_test.go
  modified: []

key-decisions:
  - "Evidence identity includes normalized content, origin, path, scope, revision, and claimed applicability while excluding observation time, so exact rediscovery cannot manufacture freshness."
  - "Freshness requires exact current scope and source hash, rejects prior-card reuse, and requires Hive evidence to carry an explicit current source state."
  - "Owner-answer evidence is reusable only when goal, session, specification, base plan, meaning, behavior, impact, risk, and acceptance hashes all match exactly."

patterns-established:
  - "Evidence before confidence: only a fresh and explicitly applicable reference can support a planning dimension."
  - "Safe evidence projection: persisted records retain a redacted summary and contained locator, never the raw source body."
  - "Fail-closed source state: superseded evidence and revoked, quarantined, or dormant Hive evidence cannot raise planning confidence."

requirements-completed: [CEC-03, PLAN-02, PLAN-06]

# Metrics
duration: 24 min
completed: 2026-09-07
---

# Phase 200 Plan 03: Planning Evidence Catalogue Summary

**Planning evidence is now content-addressed, path-contained, safely summarized, and admitted into confidence scoring only when its source, scope, frontier, and claimed dimension are all still exact.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-09-07T12:25:51Z
- **Completed:** 2026-09-07T12:50:01Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added normalization and stable SHA-256 identities for specification, survey, charter, decision, context, research, Hive, and prior-outcome evidence.
- Added catalogue collection that preflights every path before any read, rejects traversal, absolute paths, symlinks, non-regular files, and roots outside the approved repository boundary, then deduplicates and sorts records deterministically.
- Persisted only safe locators and privacy-scanned summaries, with reload validation that detects altered references, locators, summaries, applicability, or content digests.
- Added exact freshness and applicability policies across the current planning frontier, including prior-card replay refusal, explicit inactive Hive-state refusal, conservative kind-by-dimension rules, and full owner-answer equivalence.

## Task Commits

Each TDD task was committed through a failing-test commit followed by its implementation:

1. **Task 1: Build the normalized planning evidence catalogue** — `6146ca20` (test/RED), `d699e04f` (feat/GREEN)
2. **Task 2: Enforce exact freshness, applicability, and owner-answer equivalence** — `266ed144` (test/RED), `db3c3127` (feat/GREEN), `1efc2e22` (projection-integrity fix)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/planning_evidence.go` — Pure evidence normalization, collection, containment, safe projection, freshness, applicability, and owner-answer equivalence policies.
- `cmd/planning_evidence_test.go` — All-kind coverage plus identity, duplicate, traversal, typed-refusal, redaction, tamper, frontier, Hive-state, equivalence, and dimension-matrix regressions.

## Decisions Made

- Observation time is recorded for provenance but excluded from evidence identity, preventing an unchanged source from becoming a new fact merely because it was seen again.
- Path safety is atomic: all candidate locators are resolved and validated before the first source read, so one escaped source cannot leak partial catalogue results.
- Applicability is the intersection of a source kind's allowed dimensions and the exact dimensions claimed by the evidence; broad provenance alone never raises an unrelated score.
- A decision is current owner guidance only when every D-08 equivalence input matches the active planning frontier exactly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Verified persisted evidence projections against their signed reference**

- **Found during:** Plan-level self-review after Task 2
- **Issue:** A caller could reload a record whose content-addressed reference was intact but whose persisted safe summary or locator had been altered, weakening provenance integrity at the catalogue boundary.
- **Fix:** Recompute and verify the safe-summary digest, require locator/ref agreement and summary safety, and normalize owner-equivalence hash inputs before comparison.
- **Files modified:** `cmd/planning_evidence.go`, `cmd/planning_evidence_test.go`
- **Verification:** Four tampered-projection regressions fail before the fix and pass after it; the complete 38-test evidence suite and race run pass.
- **Committed in:** `1efc2e22`

---

**Total deviations:** 1 auto-fixed missing critical provenance guarantee
**Impact on plan:** The fix closes an integrity gap within the planned catalogue boundary without expanding user-facing scope.

## Issues Encountered

- The plan's historical read-first filenames `cmd/colony_prime.go` and `cmd/context_builder.go` have since been split into `cmd/colony_prime_context.go` and `cmd/context.go`; those current equivalents supplied the intended integration context.
- An additional full `cmd` diagnostic exceeded RTK's 1 MiB captured-log limit before its final status could be retained. It was not a plan gate; the authoritative evidence suite, its race run, `go vet ./cmd`, and whitespace checks all pass.
- The installed `requirements.mark-complete` handler does not recognize this milestone's legacy `**ID — title:**` checkbox shape, so the exact `CEC-03` and `PLAN-02` boxes were checked directly; `PLAN-06` was already complete. `state.update-progress` likewise found no Markdown-body progress field, while the YAML plan totals remained correct and the roadmap advanced to 3/25 normally.

## TDD Gate Compliance

- Task 1 RED (`6146ca20`) preceded GREEN (`d699e04f`).
- Task 2 RED (`266ed144`) preceded GREEN (`db3c3127`).
- `go test ./cmd -run TestPlanningEvidence -count=1` passes 38 tests.
- `go test ./cmd -run TestPlanningEvidence -race -count=1` passes 38 tests under the race detector.
- `go vet ./cmd` and `git diff --check` pass.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: bounded_file_read | `cmd/planning_evidence.go` | The catalogue introduces repository file reads at an evidence trust boundary; every locator is prevalidated as a contained regular non-symlink before any content is read. |

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 200-04 can consume stable evidence IDs and pure freshness/applicability decisions without reimplementing source policy.
- Later planner and timeline work can cite safe summaries and locators while keeping raw source bodies outside durable planning state.
- No Phase 200 blocker remains.

## Self-Check: PASSED

- Both created Go files and the plan summary exist.
- All five test, implementation, and fix commits are present in order.
- The 38-test scoped suite, its race run, `go vet ./cmd`, and whitespace checks pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
