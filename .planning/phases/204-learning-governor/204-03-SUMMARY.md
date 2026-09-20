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
  - "cmd/memory_schema.go: cmd-local aliases of the same contract, plus the field-level writer census (memoryStoreFieldWriters/Exceptions/ExceptionFloor) and the owner's Task 4 field-disposition decision (memoryStoreFieldRetired)"
  - "colony.InstinctApplicationEntry: a typed application-history entry with a real, credit-ledger-derived outcome, replacing an untyped map that always recorded success:true"
  - "The field-level census: 86 fields across 6 live memory stores, 82 with a confirmed production writer, 1 retired with the owner's recorded agreement (instinct.related_instincts), 3 kept and recorded knowingly empty (midden.acknowledge_reason, learn.parent_id, pheromone.scope)"
affects: [204-04, 204-05, 204-06, 204-07, 204-08, 204-09, 204-10, 204-11]

actuals:
  tokens: 26937
  tasks: 4
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Cross-package shared contract placed in pkg/colony (a leaf-ish package with no reverse dependency risk) rather than cmd, with cmd-local type aliases (type X = colony.X) satisfying a plan's cmd-file-literal acceptance criteria without creating a second, competing declaration or an import cycle"
    - "Custom UnmarshalJSON on a single typed slice element accepting two on-disk shapes (old untyped map, new typed struct) so one slice can hold both, side by side, for as long as a real colony's file does -- the compatibility path is a plain field (LegacySuccess), never a second type"
    - "Field-level writer census via reflect.Type.Field() enumeration over live record types, namespaced '<store>.<field>' to avoid cross-store name collisions, mirroring TestEveryMemoryPackPartHasALiveWriter's whole-section census pattern at field granularity"
    - "A field-disposition decision that is irreversible (retirement) is recorded in its own map (memoryStoreFieldRetired), separate from the knowingly-empty exception list (memoryStoreFieldExceptions) -- each entry carries both a reason and the owner's recorded agreement date, and an entry missing either is refused by name, so a retirement can never happen by omission or silently collapse into an ordinary exception"

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
  - "Task 4 (owner's field-disposition decision, 2026-09-14): the owner chose 'mixture' -- exactly the per-field recommendation table carried in this plan's own halted checkpoint, unchanged. instinct.related_instincts is retired (moved out of memoryStoreFieldExceptions into a new memoryStoreFieldRetired map, each entry carrying a reason and the owner's recorded agreement date); midden.acknowledge_reason, learn.parent_id and pheromone.scope stay in memoryStoreFieldExceptions, now recorded as the owner's knowingly-empty decision rather than an unexplained gap."
  - "Retiring instinct.related_instincts touches no stored record: the struct field stays declared (now with omitempty) purely so a pre-retirement record's own \"related_instincts\": [] still round-trips on read; only the two production writers (cmd/instinct.go, pkg/memory/promote.go) stopped setting it on a newly-created record. memoryStoreFieldExceptionFloor dropped from 4 to 3 in the same change the field left the exception map, keeping TestMemoryStoreFieldExceptionsOnlyShrink honest rather than silently widened."
  - "A retirement is validated separately from an exception, by its own check (validateMemoryStoreFieldRetirements) requiring BOTH a non-empty reason AND a non-empty owner-agreement date -- proven non-vacuous by TestRetiredFieldWithoutOwnerAgreementIsRefused, which drives the check against a synthetic map (never the real memoryStoreFieldRetired) so the negative path can fail independently of whatever the real map currently contains."

patterns-established:
  - "A field-level writer census (reflect-based, namespaced by store) as a permanent structural ratchet over every live memory-store record type, extending the existing whole-section census (TestEveryMemoryPackPartHasALiveWriter) to field granularity."
  - "An irreversible field-disposition decision (retirement) is recorded in a map structurally distinct from a reversible one (a knowingly-empty exception), with its own mandatory owner-agreement field and its own refusal check -- the shape any later Phase 204 plan facing a similar one-way memory-schema decision should reuse."

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
    description: "The owner decides, field by field, what happens to the four census fields with no production writer -- connect a writer, record them as knowingly empty, or retire them. Nothing is deleted under any option; this was the phase's only one-way decision."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestEveryMemoryStoreFieldHasALiveWriter"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestRetiredFieldWithoutOwnerAgreementIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/memory_schema_test.go#TestMemoryStoreFieldExceptionsOnlyShrink"
        status: pass
    human_judgment: false
    rationale: "The owner replied 'mixture' with no field-specific overrides, exactly matching the recommendation table this plan's own halted checkpoint carried. Applied: instinct.related_instincts retired into a new, structurally distinct memoryStoreFieldRetired map (reason + owner-agreement date, both mandatory); midden.acknowledge_reason, learn.parent_id and pheromone.scope stay in memoryStoreFieldExceptions, their reasons now recording the owner's knowingly-empty decision. The retirement's own refusal path (an entry missing a reason or an agreement date) is independently proven able to fail by TestRetiredFieldWithoutOwnerAgreementIsRefused, run against a synthetic map."

duration: 150min (Tasks 1-3) + ~25min (Task 4 continuation)
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 03: Memory Schema and Provenance Summary

**Declared one shared schema version and provenance contract across every live memory store, retyped the instinct application history so its outcome comes from the real credit ledger instead of an unconditional success, censused all 86 fields across 6 memory stores against a named production writer, and applied the owner's field-by-field decision on the 4 fields with no writer -- 1 retired (instinct.related_instincts), 3 kept and recorded knowingly empty.**

## Performance

- **Duration:** ~150 min (Tasks 1-3) + ~25 min (Task 4 continuation, applying the owner's decision)
- **Tasks:** 4/4 completed
- **Files modified:** 22 total across the plan (3 created, 19 modified: the 18 from Tasks 1-3 plus `cmd/instinct.go` touched again in Task 4)
- **Commits:** 4 task commits (test/feat/test/feat, one per task) plus this docs commit

## Accomplishments

- `pkg/colony/memory_schema.go` declares the ONE schema-version and provenance contract (`CurrentMemorySchemaVersion`, `LegacyMemorySchemaVersion`, `MemoryProvenanceKind`'s closed 5-member vocabulary, `MemoryRecordLineage`) every live memory store now stamps against -- placed in `pkg/colony` rather than `cmd` specifically to avoid an import cycle (`pkg/memory` and `pkg/learn` both import `pkg/colony` already but cannot import `cmd`). `cmd/memory_schema.go` declares cmd-local `type X = colony.X` aliases satisfying the plan's own cmd-file acceptance criteria without a second, competing declaration.
- Stamped the three write chokepoints: `appendMiddenEntry` (midden, provenance `runtime`), `PromoteService.Promote` (instincts, provenance `learning`, both the new-entry and dedup-reinforcement branches), `ColonyStore.Add` (learning entries, provenance `runtime`). A legacy record (raw pre-change bytes -- the runtime can no longer produce this shape) reads its absent version as legacy and its absent lineage as `MemoryProvenanceUnknown`, proven directly against real seeded bytes.
- `colony.InstinctApplicationEntry` replaces `ApplicationHistory`'s untyped `[]interface{}` with a typed slice carrying a real `Outcome` (drawn from the credit ledger's own `helpful/neutral/harmful/pending` vocabulary) and the `CreditRecordID` that justified it -- `recordInstinctApplicationsForPhase` now looks this up via `recruitmentCreditForContribution` instead of recording `success: true` unconditionally. `cmd/consolidation_lifecycle.go` was reordered (credit recorded before applications) so the lookup can actually find a same-pass credit record; the reordering was proven safe and regression-free. The negative branch was independently verified: `TestApplicationOutcomeComesFromCreditNotFromAdvancement` was run against a temporarily reverted fixed-outcome implementation and confirmed to FAIL, then the real implementation was restored and reconfirmed passing.
- The type change rippled into every other production and test call site that constructs `ApplicationHistory` (`cmd/instinct.go`, `cmd/internal_cmds.go`'s `instinct-apply` CLI, and 6 test fixture files across `cmd`/`pkg`) -- all converted to the typed shape with no behavior change to any existing test's intent.
- The field-level writer census (`cmd/memory_schema.go`'s `memoryStoreFieldWriters`/`memoryStoreFieldExceptions`/`memoryStoreFieldExceptionFloor`, `cmd/memory_schema_test.go`'s `TestEveryMemoryStoreFieldHasALiveWriter`/`TestMemoryStoreFieldExceptionsOnlyShrink`) reflects over all six live memory-store record types (instinct, midden, learn, pheromone, credit, handoff) and cross-references all 86 discovered fields against a real, session-confirmed production writer. 82 fields have one; 4 did not and were seeded into the shrink-only exception list pending Task 4's owner decision, each with a written reason. `TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList` proves discovery is genuinely reflection-based (a synthetic type neither map has seen is still correctly reported) rather than a disguised hardcoded list.
- **Task 4 (this continuation):** applied the owner's `mixture` decision to the census's 4 writerless fields. Added `memoryStoreFieldRetired`, a map structurally distinct from `memoryStoreFieldExceptions` -- each entry requires BOTH a written reason AND the owner's recorded agreement date, and an entry missing either is refused by name (`TestRetiredFieldWithoutOwnerAgreementIsRefused`, driven against a synthetic map so the negative path can genuinely fail). Moved `instinct.related_instincts` into that new map with `ownerAgreedOn: "2026-09-14"`. Stopped the field's two production writers (`cmd/instinct.go`'s `instinct-observe`, `pkg/memory/promote.go`'s `PromoteService.Promote`) from setting it on a newly-created record, and added `omitempty` to `colony.InstinctEntry.RelatedInstincts`'s JSON tag so it no longer serializes on a new write -- the struct field itself stays declared, unchanged, so a pre-retirement record's own `"related_instincts": []` still reads back correctly. No stored record was touched. `memoryStoreFieldExceptionFloor` dropped from 4 to 3 in the same change, keeping `TestMemoryStoreFieldExceptionsOnlyShrink` honest. The remaining three fields (`midden.acknowledge_reason`, `learn.parent_id`, `pheromone.scope`) stayed in `memoryStoreFieldExceptions`, their reasons rewritten to record the owner's 2026-09-14 knowingly-empty decision rather than an unexplained gap.

## Task Commits

Each task was committed atomically:

1. **Task 1: Declare one schema version and one provenance shape across the live memory stores** - `abeb5513` (test)
2. **Task 2: Turn the untyped application history into a typed one with a real outcome** - `711d2b15` (feat)
3. **Task 3: Census every field on every memory store against a named writer** - `1d253dc6` (test)
4. **Task 4: Apply the owner's field-disposition decision** - `dcebe1b0` (feat)

## Files Created/Modified

- `pkg/colony/memory_schema.go` - The shared schema-version/provenance contract (new)
- `cmd/memory_schema.go` - cmd-local aliases + the field-level writer census + `memoryStoreFieldRetired` (Task 4) (new)
- `cmd/memory_schema_test.go` - All Task 1-4 tests (new)
- `pkg/colony/instincts.go` - `InstinctApplicationEntry` (typed, dual-shape `UnmarshalJSON`), `InstinctEntry.SchemaVersion`/`.Lineage`, `ApplicationHistory` retyped, `RelatedInstincts` retired (Task 4: doc comment + `omitempty`)
- `pkg/colony/midden.go` - `MiddenEntry.SchemaVersion`/`.Lineage`
- `pkg/learn/learn.go` - `Entry.SchemaVersion`/`.Lineage`, `SortEntriesByRecency`
- `pkg/learn/colony_store.go` - `Add` stamps schema version + runtime-provenance lineage
- `pkg/memory/promote.go` - `Promote` stamps schema version + learning-provenance lineage (both branches); Task 4: no longer sets `RelatedInstincts` on a new record
- `pkg/memory/instinct_stats.go` - `SummarizeInstinctApplications` reads both history shapes
- `cmd/midden_shared.go` - `appendMiddenEntry` stamps schema version + runtime-provenance lineage
- `cmd/instinct_application.go` - `recordInstinctApplicationsForPhase` derives outcome from the credit ledger; `instinctAlreadyAppliedForPhase` reads the typed field
- `cmd/instinct.go`, `cmd/internal_cmds.go` - Ripple fix: `ApplicationHistory` construction updated to the typed shape; Task 4: `cmd/instinct.go`'s `instinct-observe` no longer sets `RelatedInstincts` on a new record
- `cmd/consolidation_lifecycle.go` - Reordered credit-before-applications
- `cmd/instinct_application_test.go`, `cmd/instinct_runtime_test.go`, `cmd/consolidation_promotion_target_test.go`, `cmd/memory_details_render_test.go`, `pkg/memory/consolidate_test.go`, `pkg/colony/instincts_test.go`, `pkg/agent/curation/orchestrator_test.go` - Ripple fix: fixtures converted to the typed `ApplicationHistory` shape

## Decisions Made

See `key-decisions` in frontmatter above. The owner's Task 4 reply was `mixture`, with no field-specific overrides -- applying exactly the per-field recommendation table this plan's own halted checkpoint carried:

| Field | Owner's decision (2026-09-14) |
|---|---|
| `instinct.related_instincts` | **Retired.** Moved to `memoryStoreFieldRetired`; no production writer sets it on a new record; no stored record touched. |
| `midden.acknowledge_reason` | **Kept, recorded knowingly empty** in `memoryStoreFieldExceptions`. |
| `learn.parent_id` | **Kept, recorded knowingly empty** in `memoryStoreFieldExceptions`. |
| `pheromone.scope` | **Kept, recorded knowingly empty** in `memoryStoreFieldExceptions`. |

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

**Total deviations:** 3 auto-fixed (1 Rule 3 blocking, 2 Rule 1 bugs -- all direct, unavoidable consequences of the plan's own mandated type change and file list under-specification). Task 4 introduced no new deviations: the owner's `mixture` reply required no interpretation beyond applying the recommendation table already carried in the checkpoint. **Impact on plan:** None on any machine-checkable gate -- every `<acceptance_criteria>` line and every task's `<verify>` command for Tasks 1-4 pass; the full pre-existing regression sweep (instinct-application, consolidation, colony-prime, autopilot-lessons, pheromone-outcome, application-evidence, field-writer-census, and the entirety of `pkg/colony`/`pkg/learn`/`pkg/memory`/`pkg/agent/curation`) passes with zero regressions.

## TDD Gate Compliance

Tasks 1-3 all carry `tdd="true"`, but each landed as a single combined commit (test file and production code together) rather than a strict RED-then-GREEN pair. The interdependent, cross-package nature of the schema/type changes (a struct field retype in `pkg/colony` ripples through `pkg/memory`, `pkg/learn`, and a dozen `cmd` call sites before anything compiles again) made an isolated, meaningfully-failing RED commit impractical without repeatedly breaking compilation across dependent packages mid-sequence. Every named test was, however, independently verified to be able to fail: `TestApplicationOutcomeComesFromCreditNotFromAdvancement` was run against a temporarily reverted fixed-outcome implementation and confirmed to FAIL before the real implementation was restored (documented in Deviation 2/Task 2 above); the other tests' non-vacuousness follows directly from asserting on real legacy bytes, real writer output, and (for the census) a synthetic fixture the writer map has never seen. Task 4 is a `checkpoint:decision` task, not a `tdd="true"` task -- its own new test, `TestRetiredFieldWithoutOwnerAgreementIsRefused`, was written and confirmed passing alongside the implementation in the same commit, and its non-vacuousness is direct: it is driven against a synthetic map with two deliberately-broken entries and one correct one, and asserts each is judged correctly.

## Issues Encountered

None beyond the deviations documented above. `go build ./...` and `go vet ./cmd/... ./pkg/...` are clean throughout. Full regression sweep across every touched package passes.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

All four tasks are complete and independently verified. `LEARN-01` is shared with other plans in this phase (204-04, 204-06, 204-11) not yet complete -- `gsd_run query requirements.ready-ids` confirmed it is still `blocked` as of this continuation, so it correctly stays unmarked in `REQUIREMENTS.md`; a later plan's completion will trip the shared-ID gate once every declaring plan is done.

No blockers. `recordInstinctApplicationsForPhase`'s SchemaVersion/Lineage stamping, the typed `ApplicationHistory` shape, and the retired/exception-list distinction for a writerless field are all now the pattern any later Phase 204 plan reading or writing instinct/midden/learn records -- or facing its own writerless-field decision -- should follow. `pheromones.json`'s own pointer-backed convention remains the model, now shared across three more stores.

## Self-Check: PASSED

- All 3 created files and 19 modified files (across the whole plan) confirmed present via `git status`/`git log`.
- All four task commits (`abeb5513`, `711d2b15`, `1d253dc6`, `dcebe1b0`) confirmed present via `git log --oneline`.
- All acceptance-criteria grep checks re-run clean: `cmd/memory_schema.go` contains `memoryStoreSchemaVersion`, `memoryStoreLegacySchemaVersion`, `memoryProvenanceKind`, `memoryRecordLineage`, `func memoryStoreSchemaReadable(`, `memoryStoreFieldWriters`, `memoryStoreFieldExceptions`, `memoryStoreFieldExceptionFloor`, `memoryStoreFieldRetired`; `pkg/colony/instincts.go` contains `type InstinctApplicationEntry struct`; `cmd/instinct_application.go` contains no `map[string]interface{}` append into `ApplicationHistory`, and references both `recruitmentCreditForContribution` and `phaseApplicationDecisionID`.
- Full combined named `<verify>` test set for Tasks 1-3 plus Task 4's own tests (18 test functions, several with subtests) passes together: `go test ./cmd ./pkg/colony ./pkg/learn ./pkg/memory -run '^(TestLegacyRecordsReadAsLegacy|TestNewRecordsCarryVersionAndLineage|TestFutureSchemaVersionIsRefusedNotCoerced|TestMemoryProvenanceVocabularyIsClosed|TestMemoryStoreOrderingIsTotalAndStable|TestTypedAndUntypedApplicationHistoryAgree|TestApplicationOutcomeComesFromCreditNotFromAdvancement|TestMixedShapeHistoryReadsCorrectly|TestEveryMemoryStoreFieldHasALiveWriter|TestMemoryStoreFieldExceptionsOnlyShrink|TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList|TestQueenPromotionNeverHappensWithoutRecordedUse|TestRetiredFieldWithoutOwnerAgreementIsRefused)$' -count=1 -timeout 90m` -> `ok`.
- Broader regression sweep passes: `go test ./cmd -run '^(TestColonyPrime|TestAutopilot|TestLearning|TestMemoryPack|TestConfirmed|TestCapsule|TestInstinct|TestConsolidation|TestMidden|TestPheromone|TestHive|TestRecruitmentCredit|TestApplicationEvidence|TestPhaseApplicationCredit)'` and `go test ./pkg/colony ./pkg/learn ./pkg/memory ./pkg/agent/curation` both `ok`.
- Targeted spot-checks against every raw-JSON fixture referencing `related_instincts` (`TestInstinctReadFromStandaloneStore`, `TestCurationArchivistThresholdArchivesTypedInstincts`, `TestMemoryMetrics*`) still `ok` -- these fixtures write JSON directly to disk and never depend on the two production writers that stopped setting the retired field.
- `go build ./...` and `go vet ./cmd/... ./pkg/...` both clean; `gofmt -l` clean on every file this continuation touched.
- Requirement `LEARN-01` confirmed still `blocked` (shared with 204-04/204-06/204-11, not all complete) via `gsd_run query requirements.ready-ids .planning/phases/204-learning-governor/204-03-PLAN.md LEARN-01` -- correctly NOT marked complete in `REQUIREMENTS.md`, per the shared-ID gate.

---

## Task 4 Checkpoint: Field Census Output and Owner Decision (resolved)

*(Carried verbatim from the halted checkpoint for the record, plus the owner's reply and how it was applied.)*

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

### The 4 fields with no production writer, and what happened to each

| Field | What it's for | Recommendation | Owner's decision (2026-09-14) | Applied as |
|---|---|---|---|---|
| `instinct.related_instincts` | A future link between related instincts. | **Retire.** The only code that would ever read this (`pkg/graph`) is doubly orphaned, and 204-CLASSIC-SYNTHESIS.md's own ruling (e) explicitly forbids any plan in this phase from citing `pkg/graph`'s existence as justification for new work -- so nobody in this phase can name a use, by the phase's own prior ruling. | **mixture (recommendation accepted)** | Moved to `memoryStoreFieldRetired["instinct.related_instincts"]` with `ownerAgreedOn: "2026-09-14"`; both production writers stopped setting it on a new record; `omitempty` added to the struct field; no stored record touched. |
| `midden.acknowledge_reason` | A written reason when a reviewer acknowledges a logged failure (pairs with the two fields next to it, which already work: who acknowledged it and when). | **Keep, record as empty for now.** Cheap and plausibly useful -- the acknowledge command would only need one new flag -- but nothing in this phase's own planned work needs it yet. | **mixture (recommendation accepted)** | Stays in `memoryStoreFieldExceptions`, reason rewritten to record the owner's 2026-09-14 knowingly-empty decision. |
| `learn.parent_id` | A future link from one learned lesson back to the hypothesis it came from. | **Keep, record as empty for now.** Plausibly useful, but nothing in this phase's own planned work needs it yet. | **mixture (recommendation accepted)** | Stays in `memoryStoreFieldExceptions`, reason rewritten to record the owner's 2026-09-14 knowingly-empty decision. |
| `pheromone.scope` | A future distinction between a project-wide note and a personal one. | **Keep, record as empty for now.** Plausibly useful, but nothing in this phase's own planned work needs it yet. | **mixture (recommendation accepted)** | Stays in `memoryStoreFieldExceptions`, reason rewritten to record the owner's 2026-09-14 knowingly-empty decision. |

**Owner's answer:** `mixture` -- the recommendation table above, exactly, with no field-specific overrides.
