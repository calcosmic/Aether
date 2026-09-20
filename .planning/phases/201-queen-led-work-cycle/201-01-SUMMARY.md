---
phase: 201-queen-led-work-cycle
plan: "01"
subsystem: planning
tags: [classic-synthesis, work-cycle, mechanism-study, classic-contract-corpus, go-testing]

# Dependency graph
requires:
  - phase: 199-front-door-and-classic-contract
    provides: the versioned Classic contract corpus (schema.json/mechanisms.json/cases.json) and its strict Go loader, extended here rather than re-invented
  - phase: 200-iterative-planning
    provides: the exact synthesis-document structure and corpus-extension pattern (SYN-200-*, V-200-* groups) this plan mirrors for Phase 201
provides:
  - "201-CLASSIC-SYNTHESIS.md: a signed, evidence-cited mechanism study covering all 14 SYN-201 work-cycle decisions, all 8 routed CAP rows, and the 3 RESEARCH.md open questions plus Assumption A2"
  - "A widened Classic contract corpus (schema.json + mechanisms.json) that accepts SYN-201 mechanism identifiers and 6 new V-201-* groups, with the Go loader extended to enforce them"
  - "A reusable, general-purpose case/registry coverage check (validateClassicCaseSynthesisCoverage) that later phases' case sets can reuse without a phase-scoped copy"
affects: [201-02, 201-03, 201-04, 201-05, 201-06, 201-07, 201-08, 201-09, 201-10, 201-11, 201-12, 201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 24019
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mechanism-study-then-synthesize gate (SYNTH-03/SYNTH-07): a signed CLASSIC-SYNTHESIS.md with cited git-show/path:line evidence must exist before any implementation plan in this phase is approved, mirroring the Phase 199/200 precedent exactly"
    - "Corpus extension pattern: each phase widens the shared schema.json pattern/group enum and appends its own SYN-<phase>-* mechanism block to mechanisms.json, extending (never replacing) the strict Go loader's allowlists"

key-files:
  created:
    - .planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md
  modified:
    - cmd/testdata/classic-contract/v1/schema.json
    - cmd/testdata/classic-contract/v1/mechanisms.json
    - cmd/classic_contract_test.go

key-decisions:
  - "SYN-201-03 resolves RESEARCH.md Open Question 1: the verification-boundary choice is a new sibling type (verificationBoundaryDecision{Choice, Reason, Source}) that references, but does not extend, queenCasteJudgement"
  - "SYN-201-04 resolves Open Question 2: the unified accept/verify/advance function operates on lane-normalized Go types; each continue lane keeps its own untrusted-input parsing responsibility, only the decision core is single-sourced"
  - "SYN-201-10 resolves Assumption A2: bounded-repair checkpointing reuses the Phase 199 /ant-pause handoff-ID idempotency pattern (colony.PauseHandoff.HandoffID) rather than inventing a distinct checkpoint primitive"
  - "SYN-201-14 resolves Open Question 3: model routing by caste is a policy layer above the existing spend ledger, not new ledger fields"
  - "Classic's build-time Watcher plus continue-time review squad was itself the double-verification pattern (OLD-03/OLD-04) — not a modern regression — so SYN-201-03 is disposed replace-better, not restore-modern"
  - "Automated bounded repair (D-09..D-12) has no Classic precedent whatsoever (OLD-09/OLD-10 show manual-only rollback, zero automatic repair); every repair-related SYN row is new work grounded in current Go's own narrow, already-tested mechanisms (checkFixAttemptRecord, autopilotRepairLedger)"

patterns-established:
  - "Every SYN-201 mechanism entry cites at least one CAP identifier drawn from the 8 rows this phase routes, at least one owner-facing public command from the Phase 199-locked vocabulary, and at least one owning V-201-* group, so the registry itself is exhaustively coverage-checked (all 6 new groups and all 8 CAP rows appear at least once across the 14 entries)"

requirements-completed: [SYNTH-03]

coverage:
  - id: D1
    description: "Signed Phase 201 work-cycle mechanism study (201-CLASSIC-SYNTHESIS.md) covering Queen judgment, coherent jobs, one verification boundary, unified accept/verify/advance, result truth, cost/time, next-action, credited/orphan files, bounded repair, autopilot, and telemetry -- with all 14 SYN-201 rows, all 8 routed CAP rows, and D-01..D-16 traceability"
    requirement: SYNTH-03
    verification:
      - kind: other
        ref: "automated acceptance-criteria grep checks (SYN-201-01..14 present, CAP-003/004/022/024/029/051/066/071 present, D-01..D-16 present, git show citations present, frontmatter phase/area correct)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Versioned Classic contract corpus (schema.json + mechanisms.json) accepts SYN-201 mechanism identifiers and the 6 new V-201-* groups, rejects an unregistered SYN identifier or an unregistered case reference by name, and leaves the 22 existing SYN-199/SYN-200 entries and their tests untouched"
    requirement: SYNTH-03
    verification:
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractSchema"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicMechanismCoverage"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase201MechanismRegistry"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase200SchemaMechanismsAndCausalCases"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase200CausalExecution"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 01: Queen-Led Work Cycle Classic Synthesis Summary

**Signed Phase 201 mechanism study (14 SYN-201 rows, 8 confirmed CAP dispositions, 3 open questions + Assumption A2 resolved) plus a widened, Go-loader-enforced Classic contract corpus that accepts the new identifiers and rejects unregistered ones by name.**

## Performance

- **Duration:** 55 min
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Produced `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md`: reconstructed 14 Classic (`OLD-*`) mechanisms from `3a5b81c2`, `v5.0.0`, and `v5.4` (confirming `v5.4` is the only anchor where `/ant-run` exists at all), audited 21 current Go (`NOW-*`) mechanisms with `path:line`/test citations, and produced 14 comparative `SYN-201-*` decisions with independently-confirmed dispositions.
- Independently confirmed or revised all 8 routed capability rows (CAP-003, 004, 022, 024, 029, 051, 066, 071) against fresh old/current evidence rather than inheriting the ledger's hypothesis verbatim — e.g. confirmed CAP-003's gap by tracing `appendMiddenEntry`'s actual callers, and confirmed CAP-029's gap by reading `runQuickScout`'s current read-only shape.
- Resolved all three RESEARCH.md open questions and Assumption A2 as explicit, cited synthesis decisions rather than leaving them silent for an implementer to guess.
- Traced all 16 locked decisions (D-01..D-16) to a SYN-201 row and a named implementation plan (201-02 through 201-14).
- Widened `cmd/testdata/classic-contract/v1/schema.json`'s `synthesis_decision`/mechanism-`id` pattern and `group` enum, and appended 14 fully-populated `SYN-201-*` entries to `mechanisms.json`, each covering all 8 routed CAP rows and all 6 new `V-201-*` groups at least once.
- Extended the strict Go loader (`cmd/classic_contract_test.go`) with Phase-201-aware allowlists and added `TestClassicContractPhase201MechanismRegistry`, proving by negative fixture that an unregistered SYN identifier and a case referencing an unregistered decision are each refused by name.

## Task Commits

1. **Task 1: Publish the cited Phase 201 work-cycle mechanism synthesis** - `f7a6247c` (docs)
2. **Task 2: Register SYN-201 mechanisms in the versioned Classic corpus** - `fd246978` (test)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md` - The signed mechanism study; SYNTH-07 planning gate for the rest of Phase 201
- `cmd/testdata/classic-contract/v1/schema.json` - Widened `synthesis_decision`/mechanism-`id` pattern and `group` enum to accept SYN-201 identifiers and 6 new groups
- `cmd/testdata/classic-contract/v1/mechanisms.json` - 14 new SYN-201 mechanism entries; `synthesis_sources` now lists all three signed phase syntheses
- `cmd/classic_contract_test.go` - Extended shared allowlists, extended the strict loader's CAP/public-command/synthesis-source checks for Phase 201, added a reusable `validateClassicCaseSynthesisCoverage` helper, added `TestClassicContractPhase201MechanismRegistry`

## Decisions Made

- The comparative synthesis (`SYN-201-03`) disposed the double-review-pass pattern `replace-better` rather than `restore-modern`, because Classic evidence (`OLD-03`/`OLD-04`) shows the double-verification `.planning/codebase/CONCERNS.md` names was *already present in Classic*, not a modern-only regression — restoring it would restore the defect.
- The bounded-repair cluster (`SYN-201-10`/`SYN-201-11`) was disposed `replace-better` rather than `restore-modern`, since Classic (`v5.4` `run.yaml`'s own words: "Hard failure ... = halt immediately, no recovery attempt") had zero automated repair to restore — this is new work generalizing current Go's already-tested narrow mechanisms (`checkFixAttemptRecord`, `autopilotRepairLedger`).
- Assumption A2 was resolved in favor of reuse: bounded-repair checkpoints will reuse the `/ant-pause`/`/ant-resume` handoff-ID idempotency pattern (`colony.PauseHandoff`) rather than inventing a second checkpoint concept.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Extended the strict Go loader's allowlists and checks, not just appended a new test function**
- **Found during:** Task 2, first test run
- **Issue:** `validateClassicMechanismRegistry` (the "existing strict loader" the plan names) hardcodes the Phase 199/200 `synthesis_sources` list, the Phase 200-only CAP-membership branch, and the Phase 200-only public-command allowlist. Appending SYN-201 entries to `mechanisms.json` without also extending these would make every mechanism-registry test in the file fail immediately — the loader would reject every new entry's CAP IDs as "unexpected Phase 200 capability" and reject the widened `synthesis_sources` array outright.
- **Fix:** Added `classicContractPhase201Groups`/`Decisions`/`Capabilities`/`PublicCommands` package vars (mirroring the exact Phase 199→200 extension pattern already in the file), extended the combined `classicContractGroups`/`classicContractDecisions` vars, added a `SYN-201-` branch to the loader's CAP-membership and public-command checks, and extended `synthesis_sources`' expected value to the three-element ordered list.
- **Files modified:** `cmd/classic_contract_test.go`
- **Verification:** `TestClassicContractSchema`, `TestClassicMechanismCoverage`, and `TestClassicContractPhase201MechanismRegistry` all pass; the full `TestClassicContract*`/`TestClassicMechanism*` suite (including the Phase 200-specific corpus/causal-execution tests) still passes unchanged.
- **Committed in:** `fd246978` (Task 2 commit)

**2. [Rule 2 - Missing Critical] Added a general-purpose case/registry coverage-check function**
- **Found during:** Task 2, writing the negative fixture for "a case referencing a decision that no mechanism entry registers"
- **Issue:** The plan's third `<behavior>` bullet requires this exact negative case to fail "by name," but no existing function in the file performs a phase-agnostic case-to-registry cross-check (the only existing cross-check, `validateClassicPhase200Corpus`, is hardcoded to the `SYN-200-` prefix and belongs to Phase 200's own executable case set, which this plan explicitly does not own).
- **Fix:** Added `validateClassicCaseSynthesisCoverage(cases, registry)`, a small, general-purpose (non-phase-scoped) function that any later phase's case set — including the Phase 201 case set plan 201-14 will add — can reuse without a phase-scoped copy.
- **Files modified:** `cmd/classic_contract_test.go`
- **Verification:** `TestClassicContractPhase201MechanismRegistry`'s three subtests (unregistered identifier refused by name, unregistered case reference refused by name, registered case validates against the widened schema and passes coverage) all pass.
- **Committed in:** `fd246978` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both auto-fixes were necessary for the corpus extension to actually function — the plan's own acceptance criteria ("existing tests still pass," "unregistered decision fails coverage validation by name") could not be satisfied without them. No scope creep: no executable Phase 201 behavior cases were added, per the plan's explicit "plan 201-14 owns the Phase 201 case set" boundary.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The SYNTH-07 planning gate is now satisfied: `201-CLASSIC-SYNTHESIS.md` exists, is signed (`status: ready_for_planning`), and every in-scope requirement, routed CAP row, and locked decision (D-01..D-16) traces to a cited SYN-201 row and a named plan (201-02 through 201-14).
- Plans 201-02 through 201-14 can now cite specific `SYN-201-*` decisions as their evidence-backed justification instead of a Classic feature name alone.
- The versioned Classic contract corpus is ready to accept Phase 201 executable cases whenever plan 201-14 builds the case set; `validateClassicCaseSynthesisCoverage` is already in place to enforce that every case names a registered decision.
- Ready for `201-02-PLAN.md` (collapsing the three continue lanes onto one authoritative accept/verify/advance function, per `SYN-201-04`).

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*
