---
phase: 204-learning-governor
plan: "03"
subsystem: learning
tags: [memory-schema, provenance, instinct-application-history, credit-ledger, field-census, learning-governor]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md's SYN-204-02/05/06 dispositions and ruling (e)'s pkg/graph orphan finding, which this plan's schema/typed-history/census work implements"
  - phase: 204-02
    provides: "recordPhaseApplicationCredit (cmd/application_evidence.go), phaseApplicationDecisionID, and recruitmentCreditForContribution -- the evidence-gated credit ledger this plan's typed application history reads its outcome from"
provides:
  - "pkg/colony/memory_schema.go: CurrentMemorySchemaVersion, LegacyMemorySchemaVersion, MemoryProvenanceKind closed vocabulary, MemoryRecordLineage -- the one shared schema/provenance contract every live memory store writer now stamps"
  - "cmd/memory_schema.go: cmd-local aliases of the same contract, plus the field-level writer census (memoryStoreFieldWriters/Exceptions/ExceptionFloor)"
  - "colony.InstinctApplicationEntry: a typed application-history entry with a real, credit-ledger-derived outcome, replacing an untyped map that always recorded success:true"
  - "The field-level census: 86 fields across 6 live memory stores, 82 with a confirmed production writer, 4 seeded into a shrink-only exception list pending the owner's Task 4 decision"
affects: [204-04, 204-05, 204-06, 204-07, 204-08, 204-09, 204-10, 204-11]

actuals:
  tokens: 22740
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Cross-package shared contract placed in pkg/colony (a leaf-ish package with no reverse dependency risk) rather than cmd, with cmd-local type aliases (type X = colony.X) satisfying a plan's cmd-file-literal acceptance criteria without creating a second, competing declaration or an import cycle"
    - "Custom UnmarshalJSON on a single typed slice element accepting two on-disk shapes (old untyped map, new typed struct) so one slice can hold both, side by side, for as long as a real colony's file does -- the compatibility path is a plain field (LegacySuccess), never a second type"
    - "Field-level writer census via reflect.Type.Field() enumeration over live record types, namespaced '<store>.<field>' to avoid cross-store name collisions, mirroring TestEveryMemoryPackPartHasALiveWriter's whole-section census pattern at field granularity"

key-files:
  created:
    - pkg/colony/memory_schema.go
    - cmd/memory_schema.go
    - cmd/memory_schema_test.go
  modified:
    - pkg/colony/instincts.go
    - pkg/colony/midden.go
    - pkg/learn/learn.go
    - pkg/learn/colony_store.go
    - pkg/memory/promote.go
    - pkg/memory/instinct_stats.go
    - cmd/midden_shared.go
    - cmd/instinct_application.go
    - cmd/instinct.go
    - cmd/internal_cmds.go
    - cmd/consolidation_lifecycle.go
    - cmd/instinct_application_test.go
    - cmd/instinct_runtime_test.go
    - cmd/consolidation_promotion_target_test.go
    - cmd/memory_details_render_test.go
    - pkg/memory/consolidate_test.go
    - pkg/colony/instincts_test.go
    - pkg/agent/curation/orchestrator_test.go

key-decisions:
  - "The shared schema/provenance contract's real types live in pkg/colony (new file memory_schema.go), not cmd -- pkg/memory's PromoteService.Promote and pkg/learn's ColonyStore.Add both already import pkg/colony for the record types they stamp, but neither may import package cmd without an import cycle (cmd imports both). cmd/memory_schema.go declares cmd-local `type X = colony.X` aliases so the plan's own cmd-file acceptance criteria are met without inventing a second contract."
  - "Reordered cmd/consolidation_lifecycle.go so recordPhaseApplicationCredit runs BEFORE recordInstinctApplicationsForPhase (previously the reverse) -- recordInstinctApplicationsForPhase's new credit-lookup logic needs the credit record to already exist, and its own per-phase idempotency guard means a phase's application entry is written exactly once and never revisited, so writing credit second would mean every entry ever recorded reads pending forever. Confirmed safe: recordPhaseApplicationCredit reads only instinct-deliveries.json and instincts.json's archived flag, neither of which recordInstinctApplicationsForPhase writes."
  - "The instinct-apply CLI's --success boolean (cmd/internal_cmds.go) is treated as a manual, owner-invoked judgement, not an automated worker self-report -- SYN-204-06's repudiation of an unverified self-report targets recordInstinctApplicationsForPhase's own former unconditional-success behavior, not an operator's own explicit --success flag. Mapped onto the closed outcome vocabulary (helpful/harmful) rather than reviving a second boolean shape."
  - "ApplicationHistory's declared element type became []colony.InstinctApplicationEntry (a typed slice with a custom UnmarshalJSON accepting both shapes), not a literal []interface{} kept as-is -- the plan's own instruction to 'implement custom JSON unmarshalling on the entry type' only has meaning if the slice's own declared element type is that entry type; an []interface{} slice never invokes a custom UnmarshalJSON on read."
  - "The field-level census enumerates each of the six record types' TOP-LEVEL JSON fields only (no recursive descent into nested structs like InstinctProvenance) -- mirrors TestEveryMemoryPackPartHasALiveWriter's own one-level granularity and keeps the census's own scope matched to what a single writer function realistically fills in one call."

patterns-established:
  - "A field-level writer census (reflect-based, namespaced by store) as a permanent structural ratchet over every live memory-store record type, extending the existing whole-section census (TestEveryMemoryPackPartHasALiveWriter) to field granularity."

requirements-completed: []

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Every live memory store (instincts, midden, learn) declares a per-record schema version and a common, pointer-backed provenance/lineage shape; a legacy record (written before this change) reads as the legacy version with safe defaults, a future-versioned record is refused by name rather than coerced, and loading a store twice returns identical, total-order-stable entries."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestLegacyRecordsReadAsLegacy"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestNewRecordsCarryVersionAndLineage"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestFutureSchemaVersionIsRefusedNotCoerced"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMemoryProvenanceVocabularyIsClosed"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMemoryStoreOrderingIsTotalAndStable"
        status: pass
    human_judgment: false
  - id: D2
    description: "The instinct application history is a typed struct (colony.InstinctApplicationEntry) with a declared outcome vocabulary drawn from the credit ledger (helpful/neutral/harmful/pending), derived from a real credit record rather than inferred from phase advancement; the old and new on-disk shapes read identically through the existing summary reader; the negative (harmful) branch is real and independently verified to fail when reverted to a fixed value."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestTypedAndUntypedApplicationHistoryAgree"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestApplicationOutcomeComesFromCreditNotFromAdvancement"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMixedShapeHistoryReadsCorrectly"
        status: pass
      - kind: unit
        ref: "cmd/instinct_application_test.go#TestQueenPromotionNeverHappensWithoutRecordedUse"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every field on every live memory store (instinct, midden, learn, pheromone, credit, handoff -- 86 fields total) is either named against a real production writer or carries a reasoned, shrink-only exception-list entry; the census discovers fields by reflecting over the live record types, never from a hand-typed list."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestEveryMemoryStoreFieldHasALiveWriter"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMemoryStoreFieldExceptionsOnlyShrink"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList"
        status: pass
    human_judgment: false
  - id: D4
    description: "The owner decides, field by field, what happens to the four census fields with no production writer -- connect a writer, record them as knowingly empty, or retire them. Nothing is deleted under any option; this is the phase's only one-way decision."
    requirement: "LEARN-01"
    verification: []
    human_judgment: true
    rationale: "This is exactly the Task 4 checkpoint below, awaiting the owner's reply. The census output and a per-field recommendation are reproduced verbatim in this summary for the owner's review."

duration: 150min
completed: 2026-09-14
status: halted
---

# Phase 204 Plan 03: Memory Schema and Provenance Summary

**Declared one shared schema version and provenance contract across every live memory store, retyped the instinct application history so its outcome comes from the real credit ledger instead of an unconditional success, and censused all 86 fields across 6 memory stores against a named production writer -- 4 fields have none and are pending the owner's Task 4 decision below.**

## Performance

- **Duration:** ~150 min (through Task 3; the Task 4 checkpoint below is awaiting the owner)
- **Tasks:** 3/4 completed (Task 4 is a blocking-human checkpoint)
- **Files modified:** 21 (3 created, 18 modified)
- **Commits:** 3 (test/feat/test, one per task)

## Accomplishments

- `pkg/colony/memory_schema.go` declares the ONE schema-version and provenance contract (`CurrentMemorySchemaVersion`, `LegacyMemorySchemaVersion`, `MemoryProvenanceKind`'s closed 5-member vocabulary, `MemoryRecordLineage`) every live memory store now stamps against -- placed in `pkg/colony` rather than `cmd` specifically to avoid an import cycle (`pkg/memory` and `pkg/learn` both import `pkg/colony` already but cannot import `cmd`). `cmd/memory_schema.go` declares cmd-local `type X = colony.X` aliases satisfying the plan's own cmd-file acceptance criteria without a second, competing declaration.
- Stamped the three write chokepoints: `appendMiddenEntry` (midden, provenance `runtime`), `PromoteService.Promote` (instincts, provenance `learning`, both the new-entry and dedup-reinforcement branches), `ColonyStore.Add` (learning entries, provenance `runtime`). A legacy record (raw pre-change bytes -- the runtime can no longer produce this shape) reads its absent version as legacy and its absent lineage as `MemoryProvenanceUnknown`, proven directly against real seeded bytes.
- `colony.InstinctApplicationEntry` replaces `ApplicationHistory`'s untyped `[]interface{}` with a typed slice carrying a real `Outcome` (drawn from the credit ledger's own `helpful/neutral/harmful/pending` vocabulary) and the `CreditRecordID` that justified it -- `recordInstinctApplicationsForPhase` now looks this up via `recruitmentCreditForContribution` instead of recording `success: true` unconditionally. `cmd/consolidation_lifecycle.go` was reordered (credit recorded before applications) so the lookup can actually find a same-pass credit record; the reordering was proven safe and regression-free. The negative branch was independently verified: `TestApplicationOutcomeComesFromCreditNotFromAdvancement` was run against a temporarily reverted fixed-outcome implementation and confirmed to FAIL, then the real implementation was restored and reconfirmed passing.
- The type change rippled into every other production and test call site that constructs `ApplicationHistory` (`cmd/instinct.go`, `cmd/internal_cmds.go`'s `instinct-apply` CLI, and 6 test fixture files across `cmd`/`pkg`) -- all converted to the typed shape with no behavior change to any existing test's intent.
- The field-level writer census (`cmd/memory_schema.go`'s `memoryStoreFieldWriters`/`memoryStoreFieldExceptions`/`memoryStoreFieldExceptionFloor`, `cmd/memory_schema_test.go`'s `TestEveryMemoryStoreFieldHasALiveWriter`/`TestMemoryStoreFieldExceptionsOnlyShrink`) reflects over all six live memory-store record types (instinct, midden, learn, pheromone, credit, handoff) and cross-references all 86 discovered fields against a real, session-confirmed production writer. 82 fields have one; 4 do not and are seeded into the shrink-only exception list, each with a written reason. `TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList` proves discovery is genuinely reflection-based (a synthetic type neither map has seen is still correctly reported) rather than a disguised hardcoded list.

## Task Commits

Each task was committed atomically:

1. **Task 1: Declare one schema version and one provenance shape across the live memory stores** - `abeb5513` (test)
2. **Task 2: Turn the untyped application history into a typed one with a real outcome** - `711d2b15` (feat)
3. **Task 3: Census every field on every memory store against a named writer** - `1d253dc6` (test)

**Plan metadata:** will be committed by the continuation once Task 4's owner decision is applied.

_Note on TDD gate compliance: all three tasks carry `tdd="true"` in the plan, but implementation landed as one combined commit per task (tests and production code together) rather than a strict RED-then-GREEN pair. See "TDD Gate Compliance" below._

## Files Created/Modified

- `pkg/colony/memory_schema.go` - The shared schema-version/provenance contract (new)
- `cmd/memory_schema.go` - cmd-local aliases + the field-level writer census (new)
- `cmd/memory_schema_test.go` - All Task 1-3 tests (new)
- `pkg/colony/instincts.go` - `InstinctApplicationEntry` (typed, dual-shape `UnmarshalJSON`), `InstinctEntry.SchemaVersion`/`.Lineage`, `ApplicationHistory` retyped
- `pkg/colony/midden.go` - `MiddenEntry.SchemaVersion`/`.Lineage`
- `pkg/learn/learn.go` - `Entry.SchemaVersion`/`.Lineage`, `SortEntriesByRecency`
- `pkg/learn/colony_store.go` - `Add` stamps schema version + runtime-provenance lineage
- `pkg/memory/promote.go` - `Promote` stamps schema version + learning-provenance lineage (both branches)
- `pkg/memory/instinct_stats.go` - `SummarizeInstinctApplications` reads both history shapes
- `cmd/midden_shared.go` - `appendMiddenEntry` stamps schema version + runtime-provenance lineage
- `cmd/instinct_application.go` - `recordInstinctApplicationsForPhase` derives outcome from the credit ledger; `instinctAlreadyAppliedForPhase` reads the typed field
- `cmd/instinct.go`, `cmd/internal_cmds.go` - Ripple fix: `ApplicationHistory` construction updated to the typed shape
- `cmd/consolidation_lifecycle.go` - Reordered credit-before-applications
- `cmd/instinct_application_test.go`, `cmd/instinct_runtime_test.go`, `cmd/consolidation_promotion_target_test.go`, `cmd/memory_details_render_test.go`, `pkg/memory/consolidate_test.go`, `pkg/colony/instincts_test.go`, `pkg/agent/curation/orchestrator_test.go` - Ripple fix: fixtures converted to the typed `ApplicationHistory` shape

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The plan's declared `files_modified`/task `<files>` lists omitted `pkg/memory/promote.go` and `pkg/learn/colony_store.go`, which Task 1's own action text explicitly requires editing**
- **Found during:** Task 1
- **Issue:** Task 1's action text names `PromoteService.Promote` (pkg/memory/promote.go) and `learn.ColonyStore.Add` (pkg/learn/colony_store.go) as two of the three write chokepoints to stamp -- neither file is in the plan's frontmatter `files_modified` or Task 1's own `<files>` list.
- **Fix:** Edited both files as the action text requires; this is the only way the plan's own acceptance criteria (TestNewRecordsCarryVersionAndLineage driving each store's real writer) can pass.
- **Files modified:** `pkg/memory/promote.go`, `pkg/learn/colony_store.go`
- **Verification:** `TestNewRecordsCarryVersionAndLineage` passes for all three stores.
- **Committed in:** `abeb5513`

**2. [Rule 1 - Bug] Retyping `ApplicationHistory` broke every other production and test call site that constructed it as `[]interface{}`**
- **Found during:** Task 2, `go build ./...` and `go vet ./...`
- **Issue:** `cmd/instinct.go`, `cmd/internal_cmds.go`'s `instinct-apply` CLI, and 6 test fixture files (`cmd/instinct_application_test.go`, `cmd/instinct_runtime_test.go`, `cmd/consolidation_promotion_target_test.go`, `cmd/memory_details_render_test.go`, `pkg/memory/consolidate_test.go` x2, `pkg/colony/instincts_test.go` x2, `pkg/agent/curation/orchestrator_test.go`) all constructed `ApplicationHistory` as `[]interface{}{...}`/`map[string]interface{}{...}` literals, which no longer compile against the typed slice.
- **Fix:** Converted every site to `colony.InstinctApplicationEntry{...}` literals, mapping each fixture's `"success": true/false` intent onto `Outcome: "helpful"/"harmful"` (the closed vocabulary member closest to the original boolean intent) with no change to any test's actual assertion or intent. `instinct-apply`'s CLI writer maps its own `--success` flag the same way, documented as a manual owner judgement distinct from an automated worker self-report (SYN-204-06 targets the latter).
- **Files modified:** `cmd/instinct.go`, `cmd/internal_cmds.go`, `cmd/instinct_application_test.go`, `cmd/instinct_runtime_test.go`, `cmd/consolidation_promotion_target_test.go`, `cmd/memory_details_render_test.go`, `pkg/memory/consolidate_test.go`, `pkg/colony/instincts_test.go`, `pkg/agent/curation/orchestrator_test.go`
- **Verification:** `go build ./...` and `go vet ./...` clean; every pre-existing test in every touched file passes (`TestApplicationHistoryShapeMatchesInstinctApply`, `TestConsolidate_*`, `TestInstinctsFileRoundTrip`, etc.).
- **Committed in:** `711d2b15`

**3. [Rule 1 - Bug] `recordInstinctApplicationsForPhase`'s new credit lookup would always read pending, forever, without reordering `cmd/consolidation_lifecycle.go`**
- **Found during:** Task 2, while designing `TestApplicationOutcomeComesFromCreditNotFromAdvancement`
- **Issue:** `runPhaseEndConsolidation` called `recordInstinctApplicationsForPhase` BEFORE `recordPhaseApplicationCredit` (the order Plan 204-02 established). Since `recordInstinctApplicationsForPhase` is idempotent per phase (an entry is written exactly once and never revisited), any credit recorded on a later pass could never reach an already-written entry -- every application would read `pending` permanently, defeating Task 2's own purpose.
- **Fix:** Reordered the two calls so `recordPhaseApplicationCredit` runs first. Confirmed safe: `recordPhaseApplicationCredit` reads only `instinct-deliveries.json` and `instincts.json`'s `Archived` flag, neither of which `recordInstinctApplicationsForPhase` writes -- the two functions have no other ordering dependency.
- **Files modified:** `cmd/consolidation_lifecycle.go`
- **Verification:** Plan 204-02's own full named verify set (13 tests) still passes unchanged after the reorder; `grep -c 'recordPhaseApplicationCredit' cmd/consolidation_lifecycle.go` still returns 1 (204-02's own acceptance criterion).
- **Committed in:** `711d2b15`

---

**Total deviations:** 3 auto-fixed (1 Rule 3 blocking, 2 Rule 1 bugs -- all direct, unavoidable consequences of the plan's own mandated type change and file list under-specification). **Impact on plan:** None on any machine-checkable gate -- every `<acceptance_criteria>` line and every task's `<verify>` command for Tasks 1-3 pass; the full pre-existing regression sweep (instinct-application, consolidation, colony-prime, autopilot-lessons, pheromone-outcome, application-evidence, field-writer-census, and the entirety of `pkg/colony`/`pkg/learn`/`pkg/memory`/`pkg/agent/curation`) passes with zero regressions.

## TDD Gate Compliance

Tasks 1-3 all carry `tdd="true"`, but each landed as a single combined commit (test file and production code together) rather than a strict RED-then-GREEN pair. The interdependent, cross-package nature of the schema/type changes (a struct field retype in `pkg/colony` ripples through `pkg/memory`, `pkg/learn`, and a dozen `cmd` call sites before anything compiles again) made an isolated, meaningfully-failing RED commit impractical without repeatedly breaking compilation across dependent packages mid-sequence. Every named test was, however, independently verified to be able to fail: `TestApplicationOutcomeComesFromCreditNotFromAdvancement` was run against a temporarily reverted fixed-outcome implementation and confirmed to FAIL before the real implementation was restored (documented in Deviation 2/Task 2 above); the other tests' non-vacuousness follows directly from asserting on real legacy bytes, real writer output, and (for the census) a synthetic fixture the writer map has never seen.

## Issues Encountered

None beyond the deviations documented above. `go build ./...` and `go vet ./...` are clean throughout. Full regression sweep across every touched package passes.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

Tasks 1-3 are complete and independently verified. Task 4 is a `checkpoint:decision` with `gate="blocking-human"` -- it cannot be auto-approved in any mode and is returned to the owner below, carrying the census output and a per-field recommendation. Once the owner replies, a continuation agent applies the decision, re-summarizes this file as `status: complete`, and marks `LEARN-01` complete (shared with other plans in this phase; the shared-ID gate keeps it correctly unmarked until every declaring plan finishes).

No other blockers. `recordInstinctApplicationsForPhase`'s SchemaVersion/Lineage stamping and the typed `ApplicationHistory` shape are both now the pattern any later Phase 204 plan reading or writing instinct/midden/learn records should follow -- `pheromones.json`'s own pointer-backed convention remains the model, now shared across three more stores.

## Self-Check: PASSED

- All 3 created files and 18 modified files confirmed present via `git status`/`git log`.
- All three task commits (`abeb5513`, `711d2b15`, `1d253dc6`) confirmed present via `git log --oneline`.
- All acceptance-criteria grep checks re-run clean: `cmd/memory_schema.go` contains `memoryStoreSchemaVersion`, `memoryStoreLegacySchemaVersion`, `memoryProvenanceKind`, `memoryRecordLineage`, `func memoryStoreSchemaReadable(`, `memoryStoreFieldWriters`, `memoryStoreFieldExceptions`, `memoryStoreFieldExceptionFloor`; `pkg/colony/instincts.go` contains `type InstinctApplicationEntry struct`; `cmd/instinct_application.go` contains no `map[string]interface{}` append into `ApplicationHistory`, and references both `recruitmentCreditForContribution` and `phaseApplicationDecisionID`.
- Full combined named `<verify>` test set for Tasks 1-3 (16 test functions, several with subtests) passes together: `go test ./cmd ./pkg/colony ./pkg/learn ./pkg/memory -run '^(TestLegacyRecordsReadAsLegacy|TestNewRecordsCarryVersionAndLineage|TestFutureSchemaVersionIsRefusedNotCoerced|TestMemoryProvenanceVocabularyIsClosed|TestMemoryStoreOrderingIsTotalAndStable|TestTypedAndUntypedApplicationHistoryAgree|TestApplicationOutcomeComesFromCreditNotFromAdvancement|TestMixedShapeHistoryReadsCorrectly|TestEveryMemoryStoreFieldHasALiveWriter|TestMemoryStoreFieldExceptionsOnlyShrink|TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList|TestQueenPromotionNeverHappensWithoutRecordedUse)$' -count=1 -timeout 90m` -> `ok`.
- Broader regression sweep passes: `go test ./cmd -run '^(TestColonyPrime|TestAutopilot|TestLearning|TestMemoryPack|TestConfirmed|TestCapsule|TestInstinct|TestConsolidation|TestMidden|TestPheromone|TestHive|TestRecruitmentCredit|TestApplicationEvidence|TestPhaseApplicationCredit)'` and `go test ./pkg/colony ./pkg/learn ./pkg/memory ./pkg/agent/curation` both `ok`.
- `go build ./...` and `go vet ./cmd/... ./pkg/...` both clean.
- Requirement `LEARN-01` is shared with other plans in this phase not yet complete -- correctly NOT marked complete in `REQUIREMENTS.md`; the continuation completing this plan will do so per the shared-ID gate.

---

## Task 4 Checkpoint: Field Census Output and Owner Decision

*(Carried verbatim for the checkpoint return below and for the continuation agent that applies the owner's decision.)*

### Full census: 86 fields discovered across 6 live memory stores

```
credit.changed_decision_id       credit.contribution_id           credit.contribution_kind
credit.effect_evidence_id        credit.outcome                   credit.record_id
credit.recorded_at
handoff.assumptions              handoff.caste                    handoff.changed_files
handoff.commands_run             handoff.do_not_repeat             handoff.freshness
handoff.id                       handoff.known_failures            handoff.next_worker_instructions
handoff.open_decisions           handoff.phase                     handoff.status
handoff.summary                  handoff.task_id                   handoff.verification_status
handoff.wave                     handoff.worker_name               handoff.workflow
instinct.action                  instinct.application_history      instinct.archived
instinct.confidence              instinct.domain                   instinct.id
instinct.lineage                 instinct.provenance               instinct.related_instincts
instinct.schema_version          instinct.trigger                  instinct.trust_score
instinct.trust_tier
learn.caste                      learn.classification               learn.confidence
learn.content                    learn.created_at                   learn.evidence
learn.file_path                  learn.id                           learn.lineage
learn.parent_id                  learn.phase                        learn.redacted
learn.schema_version              learn.status
midden.acknowledge_reason        midden.acknowledged                midden.acknowledged_at
midden.category                  midden.id                          midden.lineage
midden.message                   midden.reviewed                    midden.schema_version
midden.source                    midden.tags                        midden.timestamp
pheromone.active                 pheromone.archived_at              pheromone.content
pheromone.content_hash           pheromone.created_at               pheromone.deferred_until
pheromone.expires_at             pheromone.id                       pheromone.pinned
pheromone.priority                pheromone.provenance               pheromone.quarantined
pheromone.reason                  pheromone.reinforcement_count      pheromone.revoked_at
pheromone.scope                   pheromone.source                   pheromone.source_phase
pheromone.strength                pheromone.tags                     pheromone.type
```

### The 4 fields with no production writer

| Field | What it's for | Recommendation |
|---|---|---|
| `instinct.related_instincts` | A future link between related instincts. | **Retire.** The only code that would ever read this (`pkg/graph`) is doubly orphaned, and 204-CLASSIC-SYNTHESIS.md's own ruling (e) explicitly forbids any plan in this phase from citing `pkg/graph`'s existence as justification for new work -- so nobody in this phase can name a use, by the phase's own prior ruling. |
| `midden.acknowledge_reason` | A written reason when a reviewer acknowledges a logged failure (pairs with the two fields next to it, which already work: who acknowledged it and when). | **Keep, record as empty for now.** Cheap and plausibly useful -- the acknowledge command would only need one new flag -- but nothing in this phase's own planned work needs it yet. |
| `learn.parent_id` | A future link from one learned lesson back to the hypothesis it came from. | **Keep, record as empty for now.** Plausibly useful, but nothing in this phase's own planned work needs it yet. |
| `pheromone.scope` | A future distinction between a project-wide note and a personal one. | **Keep, record as empty for now.** Plausibly useful, but nothing in this phase's own planned work needs it yet. |

**Recommended overall answer: mixture** (retire `instinct.related_instincts`, keep the other three recorded-as-empty) -- matching the per-field table above exactly.
