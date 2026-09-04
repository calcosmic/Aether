---
phase: 199-front-door-and-classic-contract
plan: "12"
subsystem: lifecycle-agency
tags: [go, lifecycle-evidence, pheromones, swarm, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "03"
    provides: Versioned lifecycle receipt and evidence contracts
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Shared lifecycle facts, projection, and catalog-backed Next Up policy
  - phase: 199-front-door-and-classic-contract
    plan: "08"
    provides: Read-only territory and lifecycle evidence loading
  - phase: 199-front-door-and-classic-contract
    plan: "11"
    provides: Bounded Autopilot contract and honest typed-event limitation
provides:
  - Evidence-qualified signal receipts for focus, feedback, and redirect
  - Honest delivery, acknowledgement, measured-effect, and work-effect projections
  - Read-only Swarm localization over exact active task and dependency evidence
  - Runtime and wrapper authority ratchets that forbid fabricated live influence
affects: [phase-202-swarm, phase-203-pheromones, lifecycle-closeouts, classic-contract-corpus]

tech-stack:
  added: []
  patterns: [persist-then-project receipt, evidence-gated claims, exact task-graph localization, runtime-owned mutation]

key-files:
  created:
    - cmd/agency_contract.go
    - cmd/agency_receipt_199_test.go
    - cmd/swarm_scope_199_test.go
    - cmd/agency_wrapper_contract_199_test.go
  modified:
    - cmd/codex_workflow_cmds.go
    - cmd/swarm_cmd.go

key-decisions:
  - "A signal can claim live delivery or acknowledgement only from linked named lifecycle evidence, and can claim measured effect only from linked changed-decision evidence."
  - "Swarm localizes only from an exact recorded active task ID and treats both its dependencies and dependent tasks as the affected path; only other active tasks may continue independently."
  - "Go remains the only signal mutator and evidence authority; Claude, OpenCode, and canonical wrappers may invoke the runtime but cannot manufacture acknowledgement or causal-effect claims."

patterns-established:
  - "Persist then project: reuse the existing sanitized, deduplicating signal write once, then derive JSON and visual output from one typed result."
  - "Evidence before causality: absent Phase 202/203 evidence remains next-safe-boundary or unsupported, not inferred live behavior."
  - "Graph-scoped intervention: exact durable task identities and declared dependencies determine the affected path; problem prose never does."

requirements-completed: [CEC-04, LIFE-03]

duration: 16min
completed: 2026-09-04
---

# Phase 199 Plan 12: Evidence-Qualified Agency and Scoped Swarm Summary

**Focus, feedback, and redirect now return durable evidence-qualified receipts, while Swarm identifies an exact affected dependency path without pretending Phase 202/203 live machinery already exists.**

## Performance

- **Duration:** 16 minutes (original implementation commit window)
- **Started:** 2026-09-04T00:25:09Z
- **Implementation completed:** 2026-09-04T00:40:44Z
- **Recovery audit verified:** 2026-09-04T09:04:42Z
- **Tasks:** 2/2
- **Files modified:** 6 production and test files

## Accomplishments

- Added one typed signal result that preserves the existing sanitized/deduplicated write and distinguishes durable delivery, acknowledgement, measured effect, and active-work effect using linked evidence only.
- Wired `focus`, `feedback`, and `redirect` through one persisted signal write and the same JSON/visual receipt, including stable identity on reinforcement and byte-identical unrelated job state.
- Added a read-only Swarm preflight that localizes exact active task IDs across upstream and downstream dependency paths, excludes only that affected path from independent work, and names the Phase 202 limitation.
- Ratcheted canonical, Claude, and OpenCode wrappers so the Go runtime remains the sole mutator and no host surface invents live delivery, acknowledgement, or measured effect.

## Task Commits

Both planned tasks followed RED then GREEN, with focused correctness follow-ups:

1. **Task 1: Define honest signal and Swarm receipt projections**
   - `4330b4d4` — test(199-12): add failing agency receipt contract tests (RED)
   - `5140c0a8` — feat(199-12): define honest agency receipt contracts (GREEN)
2. **Task 2: Wire steering and Swarm to the typed contract**
   - `72b0821d` — test(199-12): add failing agency command wiring tests (RED)
   - `cb25345a` — feat(199-12): wire steering and scoped Swarm receipts (GREEN)
   - `fb3113e2` — fix(199-12): route agency next steps through catalog
   - `6e05aac6` — test(199-12): expose dependent-job localization gap (focused RED regression)
   - `9361beea` — fix(199-12): preserve the affected dependency path (GREEN)

## Files Created/Modified

- `cmd/agency_contract.go` — pure typed signal-receipt and Swarm-intervention builders plus evidence-preserving visual renderers.
- `cmd/agency_receipt_199_test.go` — delivery, acknowledgement, measured-effect, and work-effect evidence gates.
- `cmd/codex_workflow_cmds.go` — single sanitized signal-write path feeding the shared agency result.
- `cmd/swarm_cmd.go` — read-only intervention preflight attached before issuance, dispatch, escalation, and finalization paths.
- `cmd/swarm_scope_199_test.go` — exact active-job localization, dependency-path, independent-work, limitation, and verified-result proofs.
- `cmd/agency_wrapper_contract_199_test.go` — command wiring, reinforcement identity, state preservation, and wrapper-authority ratchets.

## Decisions Made

- Durable storage alone proves that the signal exists, not that a worker received or acted on it. Without linked evidence, acknowledgement remains `Not yet acknowledged` and measured effect remains `No measured effect yet`.
- A confirmed lifecycle state can support `next_safe_boundary`; otherwise the receipt says `unsupported`. Live delivery remains unavailable until the later causal transport supplies its own evidence.
- A REDIRECT pauses only an exact active conflicting job backed by conflict evidence. Focus, feedback, and non-conflicting redirect leave independent work unchanged.
- Swarm's current capability is explicitly `read_only_localization_preflight`; verified result integration still requires checkpoint evidence and the Phase 202 typed checkpoint/pause/resume machinery.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Source hygiene] Routed agency next steps through the shared command catalog**

- **Found during:** Task 2 verification
- **Issue:** The first receipt renderer introduced hand-typed command advice, which could drift from the shared shrink-only Next Up catalog.
- **Fix:** Resolved the pheromone and status actions through catalog candidates while keeping safe plain-language fallbacks.
- **Files modified:** `cmd/agency_contract.go`
- **Verification:** `TestCommandSourceHygiene` passes.
- **Committed in:** `fb3113e2`

**2. [Rule 1 - Localization correctness] Kept dependent jobs inside the affected Swarm path**

- **Found during:** Task 2 focused dependency-path verification
- **Issue:** The first localization pass included upstream prerequisites but could classify an active downstream dependent as independent work.
- **Fix:** Added a failing dependent-job case, traversed both dependency directions, and excluded the complete affected path from the independent active-job set.
- **Files modified:** `cmd/swarm_scope_199_test.go`, `cmd/agency_contract.go`
- **Verification:** `TestSwarmScope199` passes.
- **Committed in:** `6e05aac6`, `9361beea`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 correctness/source-hygiene fixes)
**Impact on plan:** Both fixes enforce the plan's existing authority and localization contract; no new product scope or architecture was added.

## Verification

- PASS — `go test ./cmd -run '^TestAgencyReceipt199(DeliveryEvidence|Acknowledgement|MeasuredEffect|WorkEffect)$' -count=1` (10 tests)
- PASS — `go test ./cmd -run '^(TestSwarmScope199|TestAgencyWrapperAuthority199|TestAgencyReceipt199CommandWiring)$' -count=1` (3 tests)
- PASS — `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1` (7 tests)
- PASS — `git diff --check 4330b4d4^..9361beea`

## TDD Gate Compliance

- Task 1: RED `4330b4d4` precedes GREEN `5140c0a8`; `cmd/agency_contract.go` did not exist at the RED commit's parent.
- Task 2: RED `72b0821d` precedes GREEN `cb25345a`; focused regression RED `6e05aac6` precedes correction `9361beea`.

## Known Stubs

None. The plan-owned files contain no TODO/FIXME, placeholder text, mock-only output path, or empty UI data source that prevents the agency or Swarm contract from working. The explicit Phase 202/203 limitations are truthful product boundaries, not stubs presented as completed behavior.

## Issues Encountered

- The implementation and tests were committed successfully, but execution ended before the summary and GSD progress receipt were written. Recovery required documentation and bookkeeping only; no implementation was re-executed or changed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 13 can proceed from an evidence-qualified agency contract and exact affected-path localization.
- Phase 202 can later supply typed Swarm checkpoint/pause/resume events through the already explicit contract fields.
- Phase 203 can later supply live delivery, acknowledgement, and changed-decision evidence without redefining the Phase 199 receipt vocabulary.
- The protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remain untouched and uncommitted.

## Self-Check: PASSED

- All six Plan 12 production and test artifacts exist.
- All seven TDD, implementation, regression, and correction commits resolve in Git in the documented order.
- GSD discovers `199-12-SUMMARY.md`, and the summary passes whitespace validation.
- All focused plan tests and the supporting command-source hygiene test pass.
- Protected pre-existing paths remain unstaged and uncommitted.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
