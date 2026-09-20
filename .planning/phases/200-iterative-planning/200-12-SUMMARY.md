---
phase: 200-iterative-planning
plan: 12
subsystem: lifecycle-guidance
tags: [go, lifecycle-facts, specification, iterative-planning, candidate-acceptance, next-action, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Typed specification, planning-candidate, acceptance-receipt, and plan-revision authority contracts
  - phase: 200-iterative-planning
    plan: 08
    provides: Immutable specification lineage, exact approval, and affected-scope state
  - phase: 200-iterative-planning
    plan: 09
    provides: Validated planning-stage cursor and replay-safe stage artifacts
  - phase: 200-iterative-planning
    plan: 11
    provides: Evidence-first Discuss decisions and settled-intent draft specification handoff
provides:
  - One-read charter, discussion, specification, planning-stage, candidate, accepted-plan, legacy, and affected-scope lifecycle facts
  - One pure authority-precedence policy shared by lifecycle terminal and JSON projections
  - Candidate review plus exact named-candidate acceptance guidance without premature build or Autopilot routes
  - Coequal guided-build and Autopilot choices for explicitly accepted and migrated legacy plans
affects: [200-13-plan-entry, 200-16-candidate-acceptance, 200-17-living-plan, 200-18-build-gate, 200-19-renderers, 200-24-init-journey]

# Tech tracking
tech-stack:
  added: []
  patterns: [single immutable lifecycle snapshot, closed authority binding status, pure precedence table, enumerable command candidates]

key-files:
  created: []
  modified:
    - cmd/lifecycle_facts.go
    - cmd/lifecycle_facts_test.go
    - cmd/lifecycle_projection.go
    - cmd/lifecycle_projection_test.go
    - cmd/next_action.go
    - cmd/next_action_test.go

key-decisions:
  - "Keep specification approval, plan-candidate readiness, and accepted-plan execution authority as independent facts; no state implies another."
  - "Treat a pending candidate as the next owner boundary without revoking the prior accepted revision, and suppress build/run guidance until candidate review is resolved."
  - "Resume every persisted planning stage through the canonical aether plan entry point; review with aether plan --candidate and name exact acceptance with --accept-candidate plus the content-addressed candidate ID."
  - "Preserve legacy_unbound build eligibility, while malformed specification or planning evidence fails closed through the canonical aether resume recovery door."

patterns-established:
  - "Authority snapshot pattern: load ColonyState once, derive all state-backed authority facts, validate optional planning artifacts read-only, then copy the same facts into every projection."
  - "Next-action precedence: recovery → discussion → specification → planning stage/owner decision → candidate → affected-plan reconciliation → accepted or legacy execution."
  - "Execution-choice parity: guided build and Autopilot remain equal-rank sibling actions with no recommendation marker."

requirements-completed: [CEC-03, PLAN-05, PLAN-06]

# Metrics
duration: 25 min
completed: 2026-09-07
---

# Phase 200 Plan 12: Lifecycle Authority Facts and Next-Action Policy Summary

**A single read-only snapshot now distinguishes intent, specification approval, planning progress, pending candidates, accepted revisions, affected scope, and legacy plans, then chooses one honest shared next action.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-09-07T18:05:30Z
- **Completed:** 2026-09-07T18:30:07Z
- **Tasks:** 2
- **Files modified:** 6 plan-owned code/test files

## Accomplishments

- Added explicit lifecycle facts for accepted charter/episode, scoped unresolved discussion count, current specification revision/hash/status/approval, planning run/stage/preset/pass, pending candidate/status/stop reason, active revision, acceptance binding, legacy classification, and sorted affected semantic IDs.
- Loaded `COLONY_STATE.json` through one injectable read, decoded the shared pending-decision file once for both discussion and blocker views, and validated the exact planning-stage artifact without repair, lock creation, timestamp changes, or any other mutation.
- Copied the new authority facts directly into terminal and JSON lifecycle projections, preserving existing stable fields and section order while adding sibling domains.
- Added a pure precedence table that keeps recovery first, then routes material intent to Discuss, drafts to Specification, approved contracts to planning, active stages and owner boundaries back to planning, candidates to review/exact acceptance, and affected accepted scope to reconciliation.
- Kept pending candidates non-buildable in guidance, while explicitly accepted and `legacy_unbound` plans continue to expose `aether build <phase>` and `aether run` as equal primary choices.
- Extended the enumerable command set with `aether spec`, candidate review, and content-addressed candidate acceptance so availability, runtime spelling, and purity guards remain centralized.

## Task Commits

Each TDD task was committed atomically with separate RED and GREEN commits:

1. **Task 1: Add specification and planning authority to lifecycle facts** — `d8fb458f` (test/RED), `73d5879d` (feat/GREEN)
2. **Task 2: Route every state to one honest next action** — `21be67f7` (test/RED), `1cb01864` (feat/GREEN)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/lifecycle_facts.go` — New intent/specification/planning fact domains, one-read loader seam, read-only pending/stage artifact collection, explicit acceptance/legacy/affected classification, and provenance.
- `cmd/lifecycle_facts_test.go` — Draft, approved, candidate-ready, accepted, affected, and legacy fixtures plus the exact one-state-load proof.
- `cmd/lifecycle_projection.go` — Sibling authority fields in the shared result, direct snapshot copying, unusable-authority recovery gate, and delegation to the pure authority table.
- `cmd/lifecycle_projection_test.go` — Terminal/JSON fact parity and no-recomputation coverage.
- `cmd/next_action.go` — Central command candidates and the pure recovery-to-execution authority precedence table.
- `cmd/next_action_test.go` — Ambiguous combined-state table proving every new branch, candidate non-buildability, exact acceptance guidance, and accepted/legacy execution parity.

## Decisions Made

- Specification approval is a contract fact only. It never makes a plan active, and a planning stop remains a non-active candidate until exact acceptance.
- A newly pending candidate becomes the primary owner-facing next step even when an older accepted revision remains durably valid; this avoids advertising execution while a replacement is awaiting authority.
- Planning-stage recovery uses the public `aether plan` entry point because the persisted run/stage cursor selects the exact frontier internally. Candidate inspection uses `aether plan --candidate`; exact acceptance names the content-addressed candidate through `aether plan --accept-candidate <id>` and leaves full binding validation to the acceptance operation implemented by Plan 16.
- `legacy_unbound` remains an explicit compatibility authority and retains the existing build/run route without fabricating specification approval, candidates, or receipts.
- Malformed or unavailable specification/planning evidence does not fall through to work. It routes to `aether resume`, matching the existing single recovery door.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The exact aggregate verification command completed 81 tests successfully and failed only pre-existing `TestNextActionNeverHardcoded`. Its three findings are in `cmd/spec_cmd.go` (`aether plan`, exact specification approval, and projection repair guidance) introduced by earlier Plan 200-10/11 work. That file is outside Plan 200-12 ownership, so it was preserved. The aggregate suite passes 81/81 with only that known ratchet excluded; both exact task gates, read-only/purity/availability checks, and `go vet ./cmd` are clean.
- Pre-existing changes in `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`d8fb458f`) failed only for the absent authority fact types, fields, binding statuses, and injectable one-state reader. Task 1 GREEN (`73d5879d`) then passed all 26 required fact/projection tests and the broader 36-test lifecycle-facts/projection set.
- Task 2 RED (`21be67f7`) left seven expected precedence branches failing while the already-correct recovery, approved-spec-to-plan, accepted-plan, and legacy routes remained green. Task 2 GREEN (`1cb01864`) made all 11 required authority tests pass.
- Both RED commits precede their corresponding GREEN commits in repository history.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-13 can consume the shared approved-spec and planning-stage facts to start or resume the exact Scout frontier after preset selection.
- Plan 200-16 can implement the candidate review and exact acceptance operation behind the canonical routes already projected here.
- Plans 200-17 and 200-18 can reuse `LifecyclePlanBindingAffected`, `AcceptedPlan`, and `LegacyUnbound` rather than inventing separate build/reconciliation policy.
- Plan 200-19 can render both human and machine output from the sibling authority fields without reading state again.
- No Plan 200-12 implementation blocker remains. The unrelated specification-command hardcode baseline remains an earlier-plan sequencing item.

## Self-Check: PASSED

- All six plan-owned code/test files and this summary exist; all four task commits are present in repository history in RED→GREEN order.
- The exact task gates pass 26/26 and 11/11; the authority aggregate passes 81/81 when the single known out-of-scope ratchet test is excluded; read-only, purity, and command-availability checks pass 3/3; `go vet ./cmd` is clean.
- One-state-load and terminal/JSON parity assertions prove that every lifecycle surface consumes the same immutable authority facts without another state read or write.
- Stub scanning found no TODO, FIXME, placeholder, coming-soon, or UI-flowing hardcoded empty implementation in the files created or modified by this plan.
- No network endpoint, authentication path, schema mutation, dependency, or new trust boundary was introduced; the planned local artifact reads reuse existing path validation and stable-file protections.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
