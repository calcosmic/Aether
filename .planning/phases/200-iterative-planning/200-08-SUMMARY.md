---
phase: 200-iterative-planning
plan: 08
subsystem: specification-authority
tags: [go, specification, approval, immutable-revisions, projection, lifecycle-transactions, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Typed specification snapshots, closed authority states, classified deltas, and exact approval-receipt contracts
  - phase: 200-iterative-planning
    plan: 02
    provides: Canonical planning-state validation, compatibility-safe loads, content-addressed IDs, and lifecycle state bindings
  - phase: 200-iterative-planning
    plan: 07
    provides: Typed successor_spec_required decisions with exact affected semantic IDs and revision evidence
provides:
  - Idempotent whole-goal and feature-scoped draft creation with typed semantic IDs and immutable successor revisions
  - Exact revision/hash/token-bound approval receipts kept separate from plan acceptance
  - Deterministic state-derived .aether/SPEC.md rendering, drift inspection, repair, and two-target transaction recovery
affects: [spec-command, discuss-closeout, iterative-plan-reconciliation, planning-candidate-acceptance, seal-gates]

# Tech tracking
tech-stack:
  added: []
  patterns: [semantic-lineage IDs, immutable full-snapshot successors, public action-token binding, one-way Markdown projection, multi-root lifecycle transactions]

key-files:
  created:
    - cmd/specification.go
    - cmd/specification_test.go
    - cmd/spec_projection.go
    - cmd/spec_projection_test.go
  modified:
    - .planning/phases/200-iterative-planning/deferred-items.md

key-decisions:
  - "Derive every body ID from typed section plus normalized semantic lineage, and exclude observation time and authority status from material revision hashes."
  - "Treat approval as a separate exact action bound to specification ID, current revision ID, content hash, approval token, and owner identity; specification approval never accepts a plan."
  - "Keep ColonyState canonical and render .aether/SPEC.md only from validated state; Markdown is never parsed back into authority."
  - "Commit canonical state before its projection within one recoverable lifecycle transaction, while allocating a fresh attempt only after a prior attempt proves it rolled back."

patterns-established:
  - "Specification lineage pattern: build a pure typed snapshot, address it by semantic content, then commit the whole aggregate through the shared lifecycle transaction."
  - "Projection pattern: validate canonical state, render every required category in fixed order, expose exact next action, and repair byte drift without touching authority records."
  - "Approval pattern: accept only the current draft's exact token/hash binding, persist a hashed token receipt, and replay the original receipt regardless of retry observation time."

requirements-completed: [PLAN-04, PLAN-05, PLAN-06]

# Metrics
duration: 49 min
completed: 2026-09-07
---

# Phase 200 Plan 08: Specification Lineage Summary

**Settled intent now becomes an immutable typed specification lineage whose exact owner approval and readable `SPEC.md` projection commit together, replay safely, and remain independent of plan acceptance.**

## Performance

- **Duration:** 49 min
- **Started:** 2026-09-07T15:20:18Z
- **Completed:** 2026-09-07T16:08:54Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added pure whole-goal and feature-scoped draft construction across outcome, included behavior, exclusions, binding decisions, stable requirements, owner-checkable acceptance, negative expectations, recovery expectations, and affected public paths.
- Added immutable add/modify/remove successors that preserve unaffected IDs and hashes, retain predecessor history, classify every category, accept typed material-decision evidence, and calculate only the immediate affected requirement/task/proof scope without rewriting the active plan.
- Added exact approval tokens and immutable receipts bound to the current revision and content hash. Wrong tokens, wrong hashes, stale IDs, superseded revisions, and divergent replays fail before mutation.
- Added deterministic `SPEC.md` rendering in the approved category order, including stable IDs, verification text, evidence summaries, every delta class, affected downstream scope, explicit `none` values, and the exact safe next command.
- Added canonical-state projection inspection and repair. Tampered or deleted Markdown is regenerated through the shared transaction engine and cannot change revision or approval authority.
- Extended draft, revision, and approval writes to commit canonical state plus the readable projection through one lifecycle transaction, with rollback consistency and interrupted-commit replay coverage.

## Task Commits

Each TDD task was committed through a failing-test commit followed by its implementation:

1. **Task 1: Create and revise immutable specification lineages** — `8499bdcf` (test/RED), `be7c980f` (feat/GREEN)
2. **Task 2: Approve exact revisions and maintain the readable projection** — `827b08c6` (test/RED), `516f8871` (feat/GREEN)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/specification.go` — Typed draft and successor reducers, semantic IDs, section deltas, affected-scope seeds, exact approval tokens/receipts, state cloning, and specification-specific lifecycle transaction/retry handling.
- `cmd/specification_test.go` — Whole-goal and feature fixtures covering creation, revision, stable identity, stale/divergent refusal, exact approval, plan non-mutation, rollback, and interrupted replay.
- `cmd/spec_projection.go` — Deterministic one-way `SPEC.md` renderer plus drift inspection, repeatable state-derived repair, explicit empty-list rendering, and safe transaction-journal selection.
- `cmd/spec_projection_test.go` — Fixed-order body, stable-ID, draft/approved next action, tamper, deletion, repair, and canonical-state immutability coverage.
- `.planning/phases/200-iterative-planning/deferred-items.md` — Records the intentional Plan 08-to-Plan 10 command-registration integration handoff exposed by the repository-wide hint audit.

## Decisions Made

- Used semantic lineage rather than list position for stable IDs. Reordering input cannot rename an owner-visible contract item, while real content changes create a new revision hash.
- Stored each successor as a complete immutable snapshot. Removed items remain in predecessor history, and unaffected items remain byte-identical in the successor.
- Kept creation time, approval status, and approval receipt outside the material revision hash. Authority can change without masquerading as a product-contract edit.
- Made the approval token public but exact: it is an action binding over the specification, revision, and content hash, not a secret or a generic confirmation. Only its SHA-256 digest is stored in the authority receipt.
- Required owner identity and a non-retroactive approval time, while allowing later exact retries to return the originally committed approval and lifecycle receipts.
- Rendered Markdown from the current validated aggregate only. Projection inspection compares digests; repair never parses or trusts edited Markdown.
- Kept affected-scope calculation deliberately immediate. Plan 200-17 owns transitive dependency closure and unreconciled-plan state, so this plan neither activates nor rewrites a plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Made a rolled-back two-target transaction safely retryable**

- **Found during:** Task 2 write-failure verification
- **Issue:** Reusing one deterministic transaction ID after a state-plus-projection write had fully rolled back could return the old no-change rollback receipt instead of performing the requested approval on retry.
- **Fix:** Added attempt selection that reuses verified or interrupted journals for exact replay but allocates a stable retry ordinal after a proven rollback. The successful retry then becomes the receipt returned by every later exact replay.
- **Files modified:** `cmd/specification.go`, `cmd/specification_test.go`
- **Verification:** `TestSpecificationApproveKeepsStateAndProjectionTogetherOnWriteFailure` proves byte-identical rollback, successful retry, and stable receipt replay; the focused normal and race suites pass.
- **Committed in:** `516f8871`

---

**Total deviations:** 1 auto-fixed bug
**Impact on plan:** The fix is required for the planned atomic-write and idempotent-retry guarantees and stays inside Plan 08's transaction helper and tests.

## Issues Encountered

- The plan referenced the former filename `cmd/lifecycle_tx.go`; the repository's actual shared implementation is `cmd/lifecycle_transaction.go`. The real file was read and reused without modification.
- Repository-wide `go test ./... -count=1` completed with 9,309 passes, 11 skips, and five failures. Four are the already-recorded protected Phase 199 vocabulary/receipt bookkeeping failures. The fifth is `TestGoSourceHintsMatchCobraContracts`, because the required projection honestly emits `aether spec --approve ...` while Plan 200-10—not this plan—owns registering `spec` and its Cobra flags.
- Repository-wide `go test ./... -race -count=1` completed with 9,306 passes, 11 skips, those same five deterministic failures, and three unrelated contention-sensitive lifecycle/reviewer child failures. All three pass immediately when rerun alone under `-race` (one lifecycle next-action test and two reviewer-artifact tests).
- The Plan 10 integration gap is recorded in `deferred-items.md`. Registering the public command here would violate the declared plan/file boundary; disguising the command would make the projection dishonest.
- The installed requirement helper did not recognize this milestone's legacy bold-ID checkbox format. `PLAN-04`, `PLAN-05`, and `PLAN-06` were already checked complete, so no manual requirement-file mutation was needed.
- The unrelated changes in `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`8499bdcf`) failed because the draft/revision API did not exist; GREEN (`be7c980f`) made all ten draft, successor, scope, delta, affected-link, and replay fixtures pass.
- Task 2 RED (`827b08c6`) failed because approval and projection APIs did not exist; GREEN (`516f8871`) made all 13 exact-authority, transaction-failure, projection-order, tamper, deletion, and repair fixtures pass.
- Every RED commit precedes its corresponding production implementation in repository history.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-10 can expose inspect, add/modify/remove, approve, and repair operations directly over these engine functions and can register the exact flags already shown by the projection.
- Plan 200-11 can hand settled discuss evidence to `createSpecificationDraft` and receive a complete draft plus `SPEC.md` in one transaction.
- Plan 200-17 can expand the immediate affected scope transitively and mark approved material successors unreconciled without changing this plan's lineage or approval semantics.
- No Plan 200-08 implementation blocker remains; the repository-wide command-hint audit will become green when its already-scheduled Plan 200-10 command registration lands.

## Self-Check: PASSED

- All four plan-owned Go files and this summary exist.
- All four TDD commits exist in the required RED→GREEN order.
- The final focused suite passes all 23 specification/projection tests; the same suite passes under `-race`, `go vet ./cmd` is clean, and whitespace checks pass.
- Exact replay, stale and divergent refusal, scoped identity preservation, add/modify/remove classification, projection rollback/recovery, tamper detection, deletion repair, and plan non-mutation each have direct regression coverage.
- Stub scanning found no TODO, FIXME, placeholder, coming-soon copy, or unwired production data source. File-access changes are limited to the plan-declared canonical state, lifecycle journals, and `.aether/SPEC.md`; no unplanned endpoint, authentication path, network boundary, or schema trust boundary was introduced.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
