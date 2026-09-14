---
phase: 204-learning-governor
plan: "01"
subsystem: learning
tags: [classic-contract, synthesis, learning-governor, recruitment-credit, hypothesis-labeling, event-bus, hive, midden, skill-auto-creation]

requires:
  - phase: 203-biological-runtime
    provides: "The recruitment/credit-ledger vocabulary (helpful/neutral/harmful/pending), pheromone resolver, and forced-reviewer authority pattern this study extends rather than duplicates"
provides:
  - "204-CLASSIC-HISTORICAL-EVIDENCE.md: direct git-show citations for all 7 SYNTH-06 Classic learning mechanics, recovered from commit 6732b0ba"
  - "204-CLASSIC-SYNTHESIS.md: SYN-204-01 through SYN-204-12 comparative matrix, 6 named rulings (a)-(f), 13-store current-Go census re-verified against live code, 7 CAP rows independently re-adjudicated"
  - "Widened Classic contract corpus (schema.json + mechanisms.json) accepting SYN-204 mechanism identifiers and 6 new V-204-* groups, 72 total registered mechanisms"
affects: [204-02, 204-03, 204-04, 204-05, 204-06, 204-07, 204-08, 204-09, 204-10, 204-11]

actuals:
  tokens: 31000
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "AST-based 'no runtime writer' proof for an authored corpus file, mirroring TestCreditRequiresBothFacts's scan-for-writes-outside-the-one-function shape"
    - "Aggregate (not per-entry) CAP coverage assertion for mechanism rows the capability ledger never routed a CAP to, mirroring Phase 203's own documented precedent"

key-files:
  created:
    - .planning/phases/204-learning-governor/204-CLASSIC-HISTORICAL-EVIDENCE.md
    - .planning/phases/204-learning-governor/204-CLASSIC-SYNTHESIS.md
  modified:
    - cmd/testdata/classic-contract/v1/schema.json
    - cmd/testdata/classic-contract/v1/mechanisms.json
    - cmd/classic_contract_test.go

key-decisions:
  - "Commit 6732b0ba (deprecation-only, content unchanged from its parent) used as the single citation anchor for all 7 Classic mechanics, since the shell scripts were physically deleted only later across 4 separate non-linear rewrite-branch commits."
  - "Corrected a draft assumption in ruling (d): pkg/learn/difficulty.go's AutoSkillModeDefault is 'propose', not 'auto' -- the runtime is safer by default than assumed, but 'propose' mode is a silent no-op with a stale comment claiming it logs a candidate when it does not."
  - "Only 6 of the 12 SYN-204 mechanism entries carry a cap_id (matching the synthesis's own routed-capability table exactly); the other 6 (LEARN-04/06/08's mechanisms) legitimately have no CAP-ledger routing and were left with empty cap_ids arrays rather than fabricated associations."

patterns-established:
  - "Historical evidence and synthesis as two separate files (Task 1 / Task 2), so the synthesis cites rather than restates its evidence -- mirrors Phase 203's own genre but adds the explicit separation this plan's own action text required."

requirements-completed: [SYNTH-06]

coverage:
  - id: D1
    description: "Recover the 7 Classic-era learning mechanics (observation capture, instinct confidence, Queen memory promotion, Hive sharing, midden reinforcement, signal strength/decay, learning presentation) from git history with direct git-show citations, causally-real/presentation/honour-system judgements, and explicit not-recoverable records where evidence does not survive."
    requirement: SYNTH-06
    verification:
      - kind: other
        ref: "grep-based acceptance criteria in 204-01-PLAN.md Task 1 <verify> block (all 6 checks + git status scope check)"
        status: pass
    human_judgment: true
    rationale: "The structural presence of citations and judgement-vocabulary is machine-checked and passed, but whether the historical claims are ACCURATE (correctly attributing behavior to the cited shell functions) requires a human familiar with this project's git history to spot-check, not merely grep for keywords."
  - id: D2
    description: "Publish the cited Phase 204 mechanism synthesis: SYN-204-01 through SYN-204-12 comparative matrix, 13-store current-Go census re-verified live this session, 6 named rulings (a)-(f) with corrected evidence, 7 CAP rows independently re-adjudicated, research-to-plan linkage, unresolved edge probes, and an owner-facing plain-English summary."
    requirement: SYNTH-06
    verification:
      - kind: other
        ref: "grep-based acceptance criteria in 204-01-PLAN.md Task 2 <verify> block (all identifier/citation/section checks)"
        status: pass
    human_judgment: true
    rationale: "Every grep-checkable acceptance criterion passed, but the SUBSTANTIVE correctness of each disposition, ruling, and CAP re-adjudication is a judgment call about architecture and evidence-sufficiency that this project's own Definition of Done reserves for a human review pass, not an automated check."
  - id: D3
    description: "Widen the versioned Classic contract corpus (schema.json pattern + group enum, mechanisms.json's 12 new SYN-204 entries) to accept Phase 204 mechanism identifiers and reject unknown/case-variant ones by name, proven via TDD (RED then GREEN) with two new Go tests."
    requirement: SYNTH-06
    verification:
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase204MechanismRegistry"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractRegistryHasNoRuntimeWriter"
        status: pass
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
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase202MechanismRegistry"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase203MechanismRegistry"
        status: pass
    human_judgment: false

duration: 90min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 01: Learning Governor Synthesis Summary

**Recovered 7 Classic-era learning mechanics from git history, published the SYN-204-01..12 mechanism synthesis with 6 named rulings re-adjudicating 7 stale/imprecise CAP dispositions against live code, and widened the versioned Classic contract corpus (72 total registered mechanisms) via a genuine RED-then-GREEN TDD cycle.**

## Performance

- **Duration:** ~90 min
- **Tasks:** 3/3 completed
- **Files modified:** 5 (2 created, 3 modified)
- **Commits:** 5 (2 docs, 1 test/RED, 1 feat/GREEN, 1 docs for this summary)

## Accomplishments

- `204-CLASSIC-HISTORICAL-EVIDENCE.md` recovers all 7 SYNTH-06 mechanics (observation capture, instinct confidence, Queen memory promotion, Hive sharing, midden reinforcement, signal strength/decay, learning presentation) with direct `git show 6732b0ba:...` citations, explicitly identifying that the "three failures produce one REDIRECT" recurrence-reinforcement behavior has **no historical evidence recoverable** and is genuinely new to the Go rewrite.
- `204-CLASSIC-SYNTHESIS.md` re-verifies all thirteen census stores against live code at this session's HEAD (not inherited from RESEARCH.md unread), records six named rulings (a)-(f) with corrections — most notably ruling (d), which corrects a draft assumption about `pkg/learn/difficulty.go`'s auto-skill-creation default (it is `"propose"`, not `"auto"`, but `"propose"` mode is a silent no-op with a stale claiming-to-log comment) — and independently re-adjudicates CAP-001, CAP-025, CAP-043, CAP-055, CAP-057, CAP-067, and CAP-070.
- The Classic contract corpus now accepts `SYN-204-01` through `SYN-204-12` and six new `V-204-*` groups; the registry carries 72 total mechanism entries (10+12+14+12+12+12), proven via a genuine RED-then-GREEN TDD cycle for the two new Go tests.

## Task Commits

Each task was committed atomically:

1. **Task 1: Recover the Classic-era learning mechanics from this repository's own history** - `8beb8ebc` (docs)
2. **Task 2: Publish the cited Phase 204 learning-governor mechanism synthesis** - `712eebcd` (docs)
3. **Task 3: Register SYN-204 mechanisms in the versioned Classic corpus** - `f1f160dd` (test, RED) then `c57c7942` (feat, GREEN)

**Plan metadata:** committed alongside this SUMMARY.

_Task 3 is TDD (`tdd="true"`): the RED commit (`f1f160dd`) adds the two new tests against the unmodified 60-entry registry and was verified to fail (`unexpected synthesis_sources`) before being committed; the GREEN commit (`c57c7942`) restores the widened `schema.json`/`mechanisms.json` and was verified to pass, along with every pre-existing Classic contract test in the file (`go test ./cmd -run '^TestClassicContract' -count=1 -timeout 90m` — 87s, all pass)._

## Files Created/Modified

- `.planning/phases/204-learning-governor/204-CLASSIC-HISTORICAL-EVIDENCE.md` - Direct `git show` citations for the 7 Classic learning mechanics
- `.planning/phases/204-learning-governor/204-CLASSIC-SYNTHESIS.md` - SYN-204-01..12 synthesis, current-Go census, 6 rulings, CAP re-adjudication, linkage, owner summary
- `cmd/testdata/classic-contract/v1/schema.json` - Widened `synthesis_decision`/mechanism `id` patterns and `group` enum for SYN-204/V-204-*
- `cmd/testdata/classic-contract/v1/mechanisms.json` - 12 new SYN-204 mechanism entries, `synthesis_sources` extended
- `cmd/classic_contract_test.go` - New `classicContractPhase204*` vocabulary vars, `validateClassicMechanismRegistry` CAP/command branches for SYN-204, `TestClassicMechanismCoverage`'s synthesis-sources assertion extended, and two new tests (`TestClassicContractPhase204MechanismRegistry`, `TestClassicContractRegistryHasNoRuntimeWriter`)

## Decisions Made

1. **Migration-boundary citation anchor.** Commit `6732b0ba` ("Deprecate dead shell scripts") preserves the FULL, unmodified shell-script content (only prepending a 7-line deprecation banner) — confirmed by direct diff. The scripts were physically deleted only later, across 4 non-linear rewrite-branch commits (one dated 2 days *before* the deprecation commit, evidence of parallel rewrite branches later merged). `6732b0ba` is therefore the single correct, plan-instructed citation anchor for every mechanic.
2. **CAP-057's Classic ancestor is more precise than the ledger's own wording implied.** `_pheromone_expire`'s eternal-memory promotion gate was never "any expiry" — it required decayed `effective_strength > 80%`. This precision is carried into ruling (c) and the current-Go gate must be checked against it, not a strawman.
3. **Ruling (d) corrects a draft assumption, not a bug.** `AutoSkillModeDefault = "propose"` (not `"auto"`) — the runtime is safer by default than the plan's own draft text assumed. But `"propose"` mode's own comment ("only logged as candidate") is stale and false: the function returns `nil` with zero logging or `PendingSuggestion` write. Both facts are recorded and the closure (route the candidate into the existing tick-to-approve queue) is assigned to plan `204-09`.
4. **CAP-id distribution per SYN-204 mechanism follows the synthesis's own evidence, not the task's descriptive prose literally.** See Deviations below.

## Deviations from Plan

### Judgment calls (not Rule 1-3 auto-fixes — documented per CLAUDE.md's rigor bar)

**1. [Judgment call] Per-entry CAP-id requirement in Task 3's action prose not applied literally**
- **Found during:** Task 3, while drafting the 12 SYN-204 mechanism entries
- **Issue:** Task 3's `<action>` prose asks the new test to "assert every entry names at least one capability identifier." The capability ledger routes only 7 CAP rows to Phase 204, and the synthesis's own routed-capability table (section 4) maps each of those 7 to specific SYN-204 rows — 6 of the 12 rows (LEARN-04/06/08's mechanisms: SYN-204-02, -06, -07, -09, -11, -12) have no evidence-backed CAP routing at all, exactly mirroring Phase 203's own precedent (SYN-203-01 through 09 all carry empty `cap_ids`). Forcing a CAP onto those 6 rows would mean inventing an association the study's own evidence does not support — a direct violation of this phase's own stated prohibition ("never record a disposition for a routed capability row without citing both historical and current evidence") and CLAUDE.md's Definition of Done ("derive fixture values the way the runtime derives them; do not type a plausible-looking literal").
- **Fix:** Left `cap_ids: []` on the 6 mechanisms with no evidence-backed routing; wrote `TestClassicContractPhase204MechanismRegistry` to assert (a) every entry has ≥1 source citation (true, and schema-required), and (b) the AGGREGATE union of CAP ids across all 12 entries covers the 7 routed rows — matching both the plan's own machine-checkable `<verify>` block (which only checks the aggregate union, not per-entry presence) and `<acceptance_criteria>` (neither of which requires per-entry CAP presence). Documented this choice inline in the test's own doc comment, mirroring `TestClassicContractPhase203MechanismRegistry`'s own precedent comment.
- **Files modified:** `cmd/classic_contract_test.go`, `cmd/testdata/classic-contract/v1/mechanisms.json`
- **Verification:** `go test ./cmd -run '^TestClassicContractPhase204MechanismRegistry$' -v` passes; the plan's own `<verify>` python check (aggregate CAP subset check) passes.
- **Committed in:** `c57c7942`

---

**Total deviations:** 1 judgment call (evidence-integrity preservation, no code bug). **Impact on plan:** None on any machine-checkable gate — every `<acceptance_criteria>` line and the full `<verify>` command for all three tasks pass. The only divergence is from one clause of descriptive prose in Task 3's `<action>` block, resolved in favor of the plan's own stated evidence-integrity prohibition and its own machine-checkable `<verify>`/`<acceptance_criteria>` (both of which this resolution satisfies exactly).

## Issues Encountered

None beyond the judgment call documented above. All acceptance criteria and verify commands for all three tasks pass; `go build ./...` and `go vet ./cmd/... ./pkg/...` are clean; the full `TestClassicContract*` test group (87s) passes with no regressions.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

This plan is the SYNTH-07 planning gate for the whole Phase 204 learning-governor scope. `204-CLASSIC-SYNTHESIS.md` now gives plans `204-02` through `204-11` a cited disposition to point at for every LEARN-01 through LEARN-08 requirement (section 6 linkage table), and the widened Classic contract corpus is ready to accept executable case registrations once those plans build the actual behaviors. Plan `204-02` (the phase's tracer, per this plan's own objective) is next: it wires `recordRecruitmentCredit` into a real production caller (ruling (a)) and applies the status-aware render filter (ruling (b)) as one thin, end-to-end slice.

No blockers. Six unresolved edge probes are explicitly flagged in the synthesis's section 8 (LEARN-01, LEARN-03, LEARN-04, LEARN-05, LEARN-06, LEARN-07) as surfaced planner assumptions, not resolved criteria — later plans should re-confirm each against their own implementation evidence before treating them as settled.

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
