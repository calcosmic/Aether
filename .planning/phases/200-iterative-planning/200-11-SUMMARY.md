---
phase: 200-iterative-planning
plan: 11
subsystem: discuss-specification-handoff
tags: [go, discuss, planning-evidence, material-decisions, specification, immutable-revisions, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 07
    provides: Typed whole-goal and feature-scoped specification state with stable semantic IDs
  - phase: 200-iterative-planning
    plan: 08
    provides: Atomic draft/projection transactions, exact replay, approval protection, and deterministic projection repair
  - phase: 200-iterative-planning
    plan: 10
    provides: Public aether spec review, revision, approval, and projection-repair surface
provides:
  - One deterministic evidence-first batch containing every unresolved material Discuss decision
  - Settled-intent creation of a canonical nine-section draft specification and readable projection
  - Exact draft reuse, feature-scope preservation, and approved-lineage refusal with explicit revision guidance
affects: [200-12-lifecycle-facts, 200-13-plan-command, 200-20-discuss-wrappers, 200-23-semantic-proof, 200-24-init-journey, 200-25-spec-wrappers]

# Tech tracking
tech-stack:
  added: []
  patterns: [evidence-frontier-first interruption, shared material-decision projection, typed discuss closeout, immutable draft handoff]

key-files:
  created:
    - cmd/discuss_spec_200_test.go
  modified:
    - cmd/discuss.go
    - cmd/discuss_test.go

key-decisions:
  - "Treat existing scoped pending decisions as the only question candidates: charter, survey, context, signals, and prior answers may suppress them, but Discuss never invents a generic fallback menu."
  - "Return every currently material behavior, authority, risk, scope, and acceptance decision in one deterministic batch even when callers still pass the legacy max-questions flag."
  - "Create a fresh whole-goal draft from settled evidence, but retain any canonical current draft exactly—including feature scope—and never replace an approved revision from Discuss."
  - "Keep specification review as a separate authority step by closing settled Discuss at aether spec and leaving every new revision in draft status."

patterns-established:
  - "Material interruption pattern: normalize admissible evidence, classify a scoped pending decision with shared policy, then render one complete decision card with exact answer syntax."
  - "Settled handoff pattern: map charter, decisions, constraints, risks, acceptance, recovery, and known paths into nine typed inputs, then let the specification engine atomically commit canonical state and projection."
  - "Replay boundary: once a canonical draft exists, re-render or repair its projection without reconstructing semantic IDs; an approved current revision yields explicit successor guidance instead of mutation."

requirements-completed: [PLAN-04, PLAN-05]

# Metrics
duration: 35 min
completed: 2026-09-07
---

# Phase 200 Plan 11: Evidence-First Discuss and Draft Specification Handoff Summary

**Discuss now interrupts only for evidence-backed material intent decisions, then atomically turns settled intent into a typed draft specification that must be reviewed through `aether spec`.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-09-07T17:28:51Z
- **Completed:** 2026-09-07T18:03:09Z
- **Tasks:** 2
- **Files modified:** 3 plan-owned code/test files

## Accomplishments

- Replaced category-count-driven fallback prompts with a current-goal evidence frontier sourced from accepted charter, survey, repository context, active owner constraints, and prior resolved answers.
- Reused the shared planning-decision classifier and card projection so each question carries a stable material domain, why-now rationale, evidence references, Queen recommendation, viable consequences, affected semantic IDs, prior-answer revalidation, resume behavior, and exact resolution syntax.
- Returned all five possible material domains in one deterministic batch and retained goal/session scoping, stale-decision quarantine, hard REDIRECT emission, exact pending resume, and no-duplicate replay.
- Created a canonical draft specification when no material question remains, with separately typed outcome, included behavior, exclusion, binding decision, requirement, acceptance, negative, recovery, and public-path content.
- Used the existing lifecycle transaction to commit `COLONY_STATE.json` and `.aether/SPEC.md` together, returned revision/hash/body/counts/receipt/projection details, and rendered the full readable draft immediately in visual output.
- Made exact settled reruns retain the same revision, repaired a drifted draft projection from canonical state, preserved feature-scoped draft content, and kept approved revisions byte-for-byte unchanged with explicit scoped-successor guidance.
- Removed settled-to-plan and orchestrator-boundary redispatch shortcuts; settled Discuss now directs the owner to `aether spec`, while specification approval remains separate and is never inferred.

## Task Commits

Each TDD task was committed atomically with separate RED and GREEN commits:

1. **Task 1: Replace generic question menus with evidence-first material batches** — `3c72b284` (test/RED), `7acce631` (feat/GREEN)
2. **Task 2: Create a draft specification when discussion is settled** — `92dcc8c9` (test/RED), `5d837e09` (feat/GREEN)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/discuss.go` — Evidence-frontier construction, material decision batching, exact-answer reuse/revalidation, typed specification request synthesis, draft/replay/approved handling, and owner-readable closeout rendering.
- `cmd/discuss_test.go` — Migrated generic-question and final-boundary expectations to the evidence-only and draft-specification lifecycle.
- `cmd/discuss_spec_200_test.go` — Five-domain decision-card coverage plus fresh draft, projection, replay, approved preservation, whole-goal identity, and feature-scope preservation fixtures.

## Decisions Made

- Existing scoped pending clarifications are the material-candidate inventory. Evidence may answer or suppress them, but goal keywords and repository shape no longer manufacture preference questions.
- The legacy `--max-questions` flag remains accepted for command compatibility but cannot truncate a material decision batch; hiding a fourth material boundary would make the owner authorize an incomplete contract.
- A fresh settled goal creates a whole-goal specification. If canonical state already contains a draft, Discuss treats that exact whole-goal or feature-scoped revision as authoritative and preserves its stable IDs rather than attempting to reverse or regenerate their semantic lineage.
- An approved current specification is never replaced, repaired as a new revision, or routed directly to planning by Discuss. The closeout identifies the exact approved revision/hash and sends the owner through explicit `aether spec` successor operations.
- Dry-run remains non-mutating: it builds and renders the deterministic draft preview in memory, then asks for a non-dry Discuss run before specification review.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved orphaned legacy clarification resolution**

- **Found during:** Task 2 full Discuss regression verification
- **Issue:** Resolving the last decision in a legacy `pending-decisions.json` with no surviving colony state persisted the answer and REDIRECT, then failed while attempting to create a specification without a current goal.
- **Fix:** Kept the exact legacy resolution successful, reported that a new colony goal is required, and only performs the draft handoff when an active scoped colony state exists.
- **Files modified:** `cmd/discuss.go`
- **Verification:** `TestDiscussResolveHardConstraintEmitsRedirect` passes alongside all 31 Discuss tests and their race run.
- **Committed in:** `5d837e09`

---

**Total deviations:** 1 auto-fixed bug.
**Impact on plan:** The compatibility guard prevents a post-persistence error for orphaned legacy state without weakening the required settled handoff for every valid active colony.

## Issues Encountered

- Repository-wide `go test ./... -count=1` completed with 9,348 passes, 16 failures, and 11 skips. The failures are cross-plan sequencing assertions: the shared Phase 199 Next Up resolver and black-box journey still encode Discuss-to-Plan, while schema/catalog/parity/reachability snapshots do not yet include the Plan 200-10 `spec` surface. These are assigned to Plans 200-12 and 200-20 through 200-25 and recorded in `deferred-items.md`; changing them here would violate this plan's file boundary and create competing lifecycle policy.
- The installed requirement helper did not parse this milestone's legacy bold-ID format for `PLAN-04` and `PLAN-05`; both entries were already checked complete in `REQUIREMENTS.md`, so no requirement-file mutation was needed.
- Pre-existing changes in `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`3c72b284`) proved the prior implementation still emitted generic fallback questions, truncated material decisions, lacked complete decision-card fields, and reused changed-impact answers.
- Task 1 GREEN (`7acce631`) made the evidence/material/resume suite pass before Task 2 began.
- Task 2 RED (`92dcc8c9`) produced four expected failures for missing draft creation, projection, replay closeout, approved preservation, and feature-scope presentation.
- Task 2 GREEN (`5d837e09`) made all seven settled/draft/approved tests pass; both RED commits precede their GREEN implementations in history.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-12 can make the shared lifecycle-facts/Next Up resolver natively select Discuss, draft-spec review, approved-spec reconciliation, candidate review, and build/run states instead of requiring this temporary command-local override.
- Plan 200-20 can project the evidence-first batch and structured draft closeout into the Claude/OpenCode Discuss wrappers without moving materiality or persistence into prompts.
- Plans 200-23 through 200-25 can update the full public journey, init/spec wrappers, frozen inventories, and parity snapshots against this Go-owned contract.
- No Plan 200-11 implementation blocker remains; its exact normal and race suites are green.

## Self-Check: PASSED

- All three plan-owned code/test files and this summary exist; all four task commits are present in repository history in RED→GREEN order for both tasks.
- The exact plan suite passes all 34 Discuss/draft-specification tests, and the same 34 tests pass under the race detector.
- Task-specific verification passes all five evidence/material/resume tests and all seven settled/draft/approved tests; all 31 Discuss tests and `go vet ./cmd` are clean.
- State and projection assertions prove one canonical draft revision, matching revision/hash output, exact replay without append, feature-scope preservation, and byte-identical approved state.
- Planning state now points to Plan 12, records the four execution decisions and Phase 200 P11 metrics, while the roadmap marks 11/25 plans complete; `PLAN-04` and `PLAN-05` were already checked.
- Stub scanning found no TODO, FIXME, placeholder, coming-soon, or UI-flowing hardcoded empty implementation in the files created or modified by this plan.
- No unplanned network, authentication, schema, or arbitrary-file access surface was introduced; projection writes use the planned canonical lifecycle transaction and repository root.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
