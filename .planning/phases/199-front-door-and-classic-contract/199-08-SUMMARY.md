---
phase: 199-front-door-and-classic-contract
plan: "08"
subsystem: territory-lifecycle
tags: [go, territory, freshness, transactions, manifests, lifecycle]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Read-only lifecycle facts and one typed lifecycle projection
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Digest-backed lifecycle transactions with rollback and exactly-once replay
provides:
  - Typed missing, fresh, stale, and unavailable territory evidence with stable reason codes
  - Automatic pre-plan survey refresh using the real surveyor manifest
  - Transactional publication of complete survey snapshots bound to immutable planning evidence
  - Wrapper authority tests keeping freshness, validation, publication, and plan mutation in Go
affects: [init, plan, colonize, status, lifecycle-projection, wrapper-parity]

tech-stack:
  added: []
  patterns: [typed freshness evidence, immutable snapshot binding, candidate-directory publication, transaction-backed survey refresh]

key-files:
  created:
    - cmd/survey_staleness_199_test.go
    - cmd/territory_lifecycle_199_test.go
    - cmd/territory_wrapper_contract_199_test.go
  modified:
    - cmd/survey_staleness.go
    - cmd/codex_colonize.go
    - cmd/codex_colonize_finalize.go
    - cmd/codex_plan.go
    - cmd/codex_plan_finalize.go
    - cmd/codex_workflow_cmds.go
    - cmd/blackbox_harness_test.go
    - cmd/codex_plan_test.go
    - cmd/codex_visuals_test.go

key-decisions:
  - "A fresh territory snapshot authenticates repository identity, canonical root, source revision, and every consumed artifact digest, including anchors.json."
  - "Automatic refresh writes worker output to a canonical candidate directory and publishes only a complete, validated snapshot through lifecycle_transaction."
  - "The plan manifest carries immutable territory evidence and finalization revalidates that exact evidence against the current repository before accepting a plan."

patterns-established:
  - "Classify without mutation: territory freshness reads evidence once and never refreshes timestamps or repairs files."
  - "Candidate then commit: worker-authored survey files remain outside the live snapshot until identity, path, baseline, transaction, completeness, and spawn evidence all validate."
  - "Runtime authority: wrappers dispatch typed manifests and finalizers but never infer freshness or write lifecycle state."

requirements-completed: [CEC-01, CEC-02, LIFE-02]

duration: 58min
completed: 2026-09-03
---

# Phase 199 Plan 08: Automatic Territory Lifecycle Summary

**Planning now consumes one authenticated territory snapshot, refreshes missing or stale surveys automatically, and atomically preserves the previous verified snapshot whenever refresh validation or publication fails.**

## Performance

- **Duration:** 58 minutes
- **Started:** 2026-09-03T20:24:23Z
- **Completed:** 2026-09-03T21:22:49Z
- **Tasks:** 3/3
- **Files modified:** 12 production/test files

## Accomplishments

- Replaced boolean survey staleness with a zero-write `SurveyFreshnessResult` that distinguishes missing, fresh, stale, and unavailable evidence using stable reason codes, age, schema, repository, revision, paths, and verified digests.
- Made plan preflight reuse fresh evidence, dispatch the complete real surveyor manifest for missing or stale evidence, and stop with typed recovery evidence when freshness is unavailable.
- Published all survey documents, compatibility data, anchors, and snapshot metadata as one lifecycle transaction after validating worker identity, declared paths, terminal spawn evidence, baseline digest, transaction ID, and worker-authored content.
- Bound planning manifests and finalization to the exact immutable territory snapshot so absent, changed, partial, tampered, or replayed evidence cannot accept a plan.
- Ratcheted canonical YAML and Claude/OpenCode wrapper surfaces so lifecycle decisions and state writes remain Go-owned.

## Task Commits

Each task was committed atomically; the two TDD tasks retain separate RED and GREEN gates:

1. **Task 1: Replace boolean staleness with typed freshness evidence** — `2937fe0a` (RED), `7ba676b6` (GREEN)
2. **Task 2: Refresh and publish territory automatically before planning** — `55e5787d` (RED), `adbb82f3` (GREEN), `e1891553` (fixture isolation)
3. **Task 3: Ratchet wrapper authority around automatic territory** — `92250da2` (contract test)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/survey_staleness.go` — typed freshness result, snapshot schema, digest verification, reason codes, and compatibility rendering.
- `cmd/survey_staleness_199_test.go` — four-state classification, stable reasons, digest completeness, and zero-write proof.
- `cmd/codex_workflow_cmds.go` — shared plan preflight that either attaches fresh evidence, requests automatic survey work, or returns an unavailable blocker.
- `cmd/codex_colonize.go` — complete internal survey manifest and collision-safe worker identities.
- `cmd/codex_colonize_finalize.go` — candidate validation and transaction-backed publication of the verified territory snapshot.
- `cmd/codex_plan.go` — territory evidence and refresh work carried in plan options and manifests.
- `cmd/codex_plan_finalize.go` — exact snapshot revalidation before plan acceptance.
- `cmd/territory_lifecycle_199_test.go` — fresh pass-through, automatic refresh, unavailable stop, atomic publication, tamper/replay, and plan-evidence coverage.
- `cmd/territory_wrapper_contract_199_test.go` — canonical and generated wrapper authority/parity guardrails.
- `cmd/blackbox_harness_test.go` — authenticated territory fixtures for compiled plan journeys.
- `cmd/codex_plan_test.go` — force-plan fixtures isolated to their temporary repositories.
- `cmd/codex_visuals_test.go` — plan visual fixtures isolated to their temporary repositories.

## Decisions Made

- Included `anchors.json` in the authenticated snapshot because planning consumes it; declaring a plan-ready snapshot without that digest would leave a mutable input outside the trust boundary.
- Kept manual `aether colonize` available as expert plumbing while the ordinary plan journey makes refresh an internal typed lifecycle step.
- Required all four surveyor castes and all seven Markdown outputs before publication; partial worker completion remains candidate data and cannot replace live territory.
- Preserved legacy non-git constructor fixtures by enforcing immutable source-revision evidence at the real repository preflight/finalization boundary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Authenticated a consumed planning artifact**

- **Found during:** Task 1 (typed freshness evidence)
- **Issue:** The legacy required-artifact list omitted `anchors.json`, even though planning consumes it.
- **Fix:** Added anchors to the snapshot digest set so fresh evidence covers every territory input used by planning.
- **Files modified:** `cmd/survey_staleness.go`, `cmd/survey_staleness_199_test.go`
- **Verification:** `TestSurveyFreshness199` proves complete digest coverage and rejects changed inputs.
- **Committed in:** `7ba676b6`

**2. [Rule 2 - Missing Critical] Extended the plan manifest at its owning definition**

- **Found during:** Task 2 (automatic refresh)
- **Issue:** The plan file list omitted `cmd/codex_plan.go`, where the manifest types and constructors that must carry immutable territory evidence are defined.
- **Fix:** Added typed territory evidence and refresh-manifest fields at the authoritative manifest boundary.
- **Files modified:** `cmd/codex_plan.go`
- **Verification:** `TestTerritoryLifecyclePlanRequiresSnapshot` rejects absent or mismatched evidence.
- **Committed in:** `adbb82f3`

**3. [Rule 1 - Bug] Prevented deterministic surveyor display-name collisions**

- **Found during:** Task 2 verification
- **Issue:** Compact deterministic display names could collide across survey workers and make valid results appear duplicated.
- **Fix:** Made manifest names unique and used TaskID as the authoritative merge identity.
- **Files modified:** `cmd/codex_colonize.go`, `cmd/codex_colonize_finalize.go`
- **Verification:** Automatic-refresh and transactional-publication tests pass repeatedly with complete worker result validation.
- **Committed in:** `adbb82f3`

**4. [Rule 1 - Bug] Canonicalized candidate paths across platform symlinks**

- **Found during:** Task 2 verification on macOS
- **Issue:** `/var` and `/private/var` aliases caused a valid temporary candidate directory to fail containment checks.
- **Fix:** Resolved the candidate root before applying boundary-safe containment and regular-file validation.
- **Files modified:** `cmd/codex_colonize_finalize.go`
- **Verification:** Transactional publication tests pass from macOS temporary directories without weakening containment.
- **Committed in:** `adbb82f3`

**5. [Rule 3 - Blocking] Isolated existing plan test fixtures to their temporary repositories**

- **Found during:** Overall command-package verification
- **Issue:** Five tests created temporary state but still executed plan from the source checkout, so the new repository-bound territory gate correctly inspected the wrong repository.
- **Fix:** Ran those fixtures inside their declared temporary repository and seeded authenticated territory evidence for compiled black-box plan journeys.
- **Files modified:** `cmd/blackbox_harness_test.go`, `cmd/codex_plan_test.go`, `cmd/codex_visuals_test.go`
- **Verification:** The focused force-plan, compiled journey, and visual-output cases pass in the final acceptance suite.
- **Committed in:** `adbb82f3`, `e1891553`

---

**Total deviations:** 5 auto-fixed (2 missing critical functionality, 2 bugs, 1 blocking fixture issue).
**Impact on plan:** Each change was required to keep the snapshot trust boundary complete or to make existing fixtures exercise their intended repository; no public command or Phase 200 planning behavior was added.

## Known Stubs

None. Stub-pattern matches are deliberate rejection fixtures or empty/nil control-flow cases; no production territory path renders placeholder or mock data.

## Verification

- Passed the complete Plan 199-08 acceptance suite, including typed freshness, automatic lifecycle refresh, atomic publication, plan snapshot binding, wrapper authority, source hygiene, compiled journeys, force recovery, E2E recovery, and plan visuals (`go test ./cmd -run '^(...)$' -count=1`, 37.587s).
- Focused RED tests failed before implementation and all focused GREEN tests pass after implementation.
- The broader command-package sweep progresses through the plan-owned tests and stops only on a known deferred Phase 199 migration expectation: `TestWorkflowSuggestionsForInterruptedExecutingSuggestsRestartBuild` expects legacy `aether build 2`, while the shared lifecycle projection now correctly returns `aether resume`.

## Issues Encountered

- The broad command suite still contains pre-199 Next Up snapshots owned by later Phase 199 renderer/recovery plans. This exact class is already recorded in `deferred-items.md`; production behavior was not weakened to satisfy the obsolete expectation.
- No authentication gates or external service setup were required.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Init/plan and status can now share one authoritative typed territory projection (CAP-049 and CAP-059) without host-side freshness inference.
- Later lifecycle renderer plans can update the remaining legacy Next Up snapshots; Plan 199-29 retains ownership of the final full/race repository gates.

## Self-Check: PASSED

- All three created contract-test files and this summary exist.
- All six task and corrective commits resolve to committed Git objects.
- Summary whitespace validation passes and no protected pre-existing path is staged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
