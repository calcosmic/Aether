---
phase: 199-front-door-and-classic-contract
plan: "15"
subsystem: lifecycle-closure
tags: [go, seal, transactions, recovery, receipts, privacy, hive, retained-state]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Digest-backed multi-root lifecycle transactions, rollback, and exactly-once replay
  - phase: 199-front-door-and-classic-contract
    plan: "08"
    provides: Read-only lifecycle and territory evidence loading
  - phase: 199-front-door-and-classic-contract
    plan: "12"
    provides: Evidence-qualified signal lifecycle and privacy boundaries
  - phase: 199-front-door-and-classic-contract
    plan: "13"
    provides: Transactional pause/resume provenance and owner-checkpoint handling
provides:
  - Typed verified-completion and owner-forced-incomplete seal preflights and outcomes
  - One crash-recoverable transaction for retained state, Crowned record, session, registry, findings, learnings, signals, checkpoints, rollback, events, and receipts
  - Memory provenance plus privacy-filtered, policy-gated, replay-safe Hive promotion receipts
  - Distinct verified and forced renderers with status primary and entomb optional
affects: [plan-199-16, plan-199-23, plan-199-27, plan-199-29, plan-199-33, entomb, status, classic-contract]

tech-stack:
  added: []
  patterns: [pure preflight before effects, typed outcome renderer, multi-root lifecycle transaction, post-commit receipt idempotency]

key-files:
  created:
    - cmd/seal_outcome.go
    - cmd/seal_outcome_199_test.go
    - cmd/seal_render.go
    - cmd/seal_transaction_199_test.go
  modified:
    - cmd/seal_confirmation.go
    - cmd/seal_final_review.go
    - cmd/codex_workflow_cmds.go

key-decisions:
  - "Use one typed SealPreflight and durable SealOutcome as completion truth; only a direct owner may request forced_incomplete with both --force and a nonblank reason."
  - "Commit every authoritative seal artifact through the shared lifecycle transaction while retaining active COLONY_STATE.json for review; status remains primary and entomb optional."
  - "Treat issue-severity findings as retained residual risks, while blockers and unresolved owner checkpoints prevent normal verified closure."
  - "Permit Hive promotion only after a verified transaction under promote/unset policy, after sanitization and sensitivity filtering, with deterministic replay-safe promotion receipts."

patterns-established:
  - "Seal boundary: load immutable facts, decide a typed preflight, obtain exact owner confirmation, then begin all durable effects."
  - "Closure truth: visual and JSON render only from the persisted disposition, never from a loose force flag."
  - "Post-commit external effect: bind each eligible Hive promotion to a deterministic transaction/receipt reference and reuse its receipt on replay."

requirements-completed: [CEC-08, LIFE-05]

duration: 65min
completed: 2026-09-04
---

# Phase 199 Plan 15: Honest Transactional Seal Summary

**Seal now records either verified completion or an unmistakable owner-forced incomplete closure through one crash-recoverable transaction while keeping the sealed colony active for review.**

## Performance

- **Duration:** 65 minutes
- **Started:** 2026-09-04T11:23:34Z
- **Completed:** 2026-09-04T12:28:24Z
- **Tasks:** 2/2
- **Files changed:** 7 production/test files

## Accomplishments

- Added a pure evidence-backed preflight that enumerates completed and unfinished work, gate status, required evidence, owner checkpoints, residual risk, preserved contents, rollback evidence, and exact next actions before any mutation.
- Restricted forced-incomplete closure to a direct owner invocation with `--force --reason`, persisted the reason and incomplete truth across retained closure artifacts, and removed every normal-success discriminator from its renderer.
- Replaced sequential seal writes with a 15-target repository/data/hub transaction whose injected-fault retries converge to one state, outcome, receipt, registry transition, and event stream.
- Retained active sealed state and recorded project/colony memory source, store, repository identity, scope, digest, evidence/decision links, sanitization, and closure transaction provenance.
- Classified FOCUS/REDIRECT retention and cross-project wisdom eligibility explicitly; prohibited policies and forced closure never promote, while verified promotion uses sanitized content and replay-safe linked receipts.

## Task Commits

Each task was committed atomically:

1. **Task 1: Separate verified and forced-incomplete seal preflights**
   - `a1fb07aa` — test(199-15): define honest seal preflight contract (RED)
   - `bdba7d19` — feat(199-15): separate seal completion outcomes (GREEN)
2. **Task 2: Commit and render the selected seal outcome atomically**
   - `5e39c1fb` — test(199-15): define atomic seal transaction contract (RED)
   - `637c5472` — feat(199-15): commit retained seal closure atomically (GREEN)
   - `2d051c03` — fix(199-15): retain non-blocking seal risks

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/seal_outcome.go` — typed preflight, caller authority, verified/forced outcome partitioning, exact confirmation copy, unresolved evidence, and retained-content contract.
- `cmd/seal_outcome_199_test.go` — Task 1 evidence, refusal, authority, confirmation, and residual-risk tests.
- `cmd/seal_render.go` — disposition-only verified and forced result renderers plus provenance/receipt structures.
- `cmd/seal_transaction_199_test.go` — 15-target crash matrix, replay, durable forced truth, retained state, memory, signals, privacy, Hive policy, and next-action proof.
- `cmd/seal_confirmation.go` — typed preflight card and zero-effect owner-confirmation boundary.
- `cmd/seal_final_review.go` — finalizer force rejection and transaction-bound final-review evidence.
- `cmd/codex_workflow_cmds.go` — retained seal transaction assembly, multi-root commit/recovery, event/registry/session records, memory/privacy decisions, and post-commit promotion receipts.

## Decisions Made

- Completion truth is decided once from immutable `LifecycleFacts` and carried by typed disposition through confirmation, transaction, receipt, state, session, summary, and renderer.
- Force is not a more permissive verified seal. It is an owner-authorized `forced_incomplete` record with reason, residuals, rollback evidence, and non-success presentation.
- Seal closes the record but does not clear it. The active state remains inspectable; `aether status` is primary and `aether entomb` is explicitly optional.
- Non-blocking issues remain visible as residual risks; only incomplete work, failed/skipped gates, missing or conflicting evidence, blockers, and unresolved owner checkpoints prevent verified closure.
- Hive writes happen only after the verified transaction commits and reuse deterministic evidence/receipt links on retry so replay cannot reinforce the same wisdom twice.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Prevented duplicate Hive reinforcement during seal replay**
- **Found during:** Task 2 (transaction replay review)
- **Issue:** A retry after the main seal commit could call the existing Hive promotion writer again before recognizing the prior post-commit effect.
- **Fix:** Added deterministic per-wisdom promotion references, existing-Hive evidence lookup, durable receipt reuse, and returned evidence linkage.
- **Files modified:** `cmd/codex_workflow_cmds.go`, `cmd/seal_transaction_199_test.go`
- **Verification:** `TestSealTransaction199ReplayExactlyOnce` proves one Hive entry/evidence row and byte-equivalent promotion receipts across replay.
- **Committed in:** `637c5472`

**2. [Rule 1 - Bug] Kept issue-severity findings as residual risk instead of unfinished work**
- **Found during:** Post-Task 2 broader seal-suite triage
- **Issue:** The initial preflight treated every unresolved flag as a closure blocker, incorrectly upgrading explicitly non-blocking issue records.
- **Fix:** Partitioned issue records into retained residual risk while blockers and unresolved owner checkpoints remain fail-closed; rendered stable IDs for each unresolved item.
- **Files modified:** `cmd/seal_outcome.go`, `cmd/seal_confirmation.go`, `cmd/seal_outcome_199_test.go`
- **Verification:** The exact Task 1 suite now proves an open issue remains visible without changing a verified result to incomplete.
- **Committed in:** `2d051c03`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs)
**Impact on plan:** Both fixes enforce the requested truth and exactly-once contracts without expanding the public surface or changing another plan's files.

## Verification

- PASS — `go test ./cmd -run '^TestSealOutcome199(VerifiedPreflight|IncompleteRefusal|ForceAuthority|ConfirmCopy)$' -count=1`.
- PASS — `go test ./cmd -run '^TestSealTransaction199(Atomic|ReplayExactlyOnce|ForcedTruth|RetainsActiveState|MemoryProvenance|SignalRetention|WisdomPrivacy|HivePolicy|NextActions)$' -count=1`.
- PASS — the combined focused suite under `go test -race`, including all 15 crash boundaries and both forced visual modes.
- PASS — `git diff --check` for every plan-owned implementation/test commit.
- EXPECTED STAGED MIGRATION — the broader pre-199 `TestSeal*` selection retains 26 top-level expectation failures documented in `deferred-items.md`; those tests require behavior D-16/D-17 explicitly replaces and are owned by later wrapper/corpus/full-gate plans.

## TDD Gate Compliance

- Task 1 RED `a1fb07aa` failed because the typed preflight/outcome API did not exist; GREEN `bdba7d19` made all four exact test groups pass.
- Task 2 RED `5e39c1fb` failed because the atomic seal transaction and disposition renderer did not exist; GREEN `637c5472` made all nine exact test groups pass.
- Correctness follow-up `2d051c03` preserves the GREEN gates while pinning non-blocking residual-risk behavior.

## Known Stubs

None. The plan-owned implementation contains no TODO/FIXME marker, placeholder/coming-soon path, or empty mock data feeding the closure result.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: multi-root-file-transaction | `cmd/codex_workflow_cmds.go` | Seal now writes repository, lifecycle-data, and hub-registry trust boundaries together; typed root allowlists, containment, staged pre-images, immutable intent, digest verification, rollback, and receipts constrain the surface. |
| threat_flag: cross-project-wisdom | `cmd/codex_workflow_cmds.go` | Verified seal may promote eligible memory across projects only after sensitivity filtering and `SanitizeSignalContent`, under resolved promote/unset policy, with transaction-linked replay-safe receipts. |

## Issues Encountered

- The old seal tests encode the prior multi-stage ceremony, including writes before the owner answers and review archiving during seal. Restoring those expectations would violate this plan's no-effect-before-confirmation and retained-state boundary, so the exact mismatch set is carried in `deferred-items.md` for Plans 199-23/27/29.
- The installed progress handler could count 17 of 34 summaries but could not match this repository's frontmatter-only progress shape and wrote `0%`; the percentage was corrected to the exact plan-count value, `50%`. The requirements handler likewise could not parse the bold-description checkbox format, but `CEC-08` and `LIFE-05` were already checked complete, so `REQUIREMENTS.md` needed no edit.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes were left untouched and unstaged throughout execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-16 can build digest-verified entomb/archive-and-clear on the retained `SealOutcome`, transaction, receipt, and active-state record.
- Plan 199-23 can align Claude/OpenCode wrappers and the existing Codex build-cycle guide without creating force authority outside the runtime.
- Plans 199-27 and 199-29 can migrate executable closure cases and require the final repository-wide normal/race gates against the settled D-16/D-17 contract.
- No Plan 199-15 blocker remains.

## Self-Check: PASSED

- All seven declared implementation/test files and this summary exist.
- RED/GREEN commits `a1fb07aa`, `bdba7d19`, `5e39c1fb`, and `637c5472`, plus correctness fix `2d051c03`, resolve in Git in the documented order.
- Exact normal and race-enabled Plan 199-15 tests pass, summary/deferred evidence is whitespace-clean, and protected pre-existing paths remain unstaged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
