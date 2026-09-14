---
phase: 204-learning-governor
plan: "05"
subsystem: learning
tags: [regression-fixtures, midden, defect-register, audit-findings, fixture-bank, json-schema, sanitizer]

requires:
  - phase: 204-01
    provides: "SYN-204-07's disposition (confirmed midden.json incidents become sanitised, deduplicated, versioned fixture entries) and the naming of pkg/colony/midden.go / pkg/colony/sanitize.go as the sources this plan reuses"
provides:
  - "A versioned JSON Schema (cmd/testdata/fixture-bank/v1/schema.json) every fixture bank must validate against, closing on failing_baseline, invariant, allowed_mutations, provenance, privacy, severity and content_digest, and requiring a retirement to always name a successor"
  - "Four confirmation-gated readers (failure log, audit findings, defect register, phase closing reports) that normalise every source into one confirmedIncident shape and refuse anything unconfirmed by name"
  - "convertConfirmedIncidentsToFixtures: sanitiser reuse, sha256 content-digest dedup, deterministic fixture IDs, honest broad-invariant labeling"
  - "writeFixtureBank as the bank file's one writer (AST-scan proven) and retireFixture (names a successor, never deletes)"
  - "A generated, reproducible 46-fixture seeded bank drawn from this repository's own 42 verified audit findings, 9 fixed defect-register entries and 6 phase-report rows (covering 9 named test fixes)"
affects: [204-06, 204-07, 204-08, 204-09, 204-10, 204-11]

actuals:
  tokens: 32851
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Repo-root-override pattern (fixtureBankRepoRootOverride) so a committed-testdata file's read/write path can be redirected to an isolated temp dir under test, mirroring how newTestStore redirects COLONY_DATA_DIR"
    - "AST-scan single-writer proof (TestFixtureBankHasOneWriter), same shape as TestClassicContractRegistryHasNoRuntimeWriter and TestCreditRequiresBothFacts"
    - "Reused the existing santhosh-tekuri/jsonschema validator and violation-formatting helpers from cmd/contract_schema.go (jsonPointerFromSegments, violationPrinter) rather than writing a second schema-validation stack"

key-files:
  created:
    - cmd/fixture_conversion.go
    - cmd/fixture_conversion_test.go
    - cmd/testdata/fixture-bank/v1/schema.json
    - cmd/testdata/fixture-bank/v1/bank.json
  modified: []

key-decisions:
  - "Incident text is truncated to the sanitiser's own 500-char budget BEFORE calling colony.SanitizeSignalContent, since real confirmed-incident prose in this repository (audit claims, defect descriptions) routinely exceeds 500 characters and the sanitiser's length check runs before its escaping step -- without pre-truncation, nearly every incident would be refused for length, not for unsafe content. This is a disclosed normalisation step, not a bypass: the sanitiser still runs, and still refuses genuinely unsafe content (11 of 60 candidates were dropped this run for backtick/XML-tag content)."
  - "Every generated fixture's invariant is mechanically restated from the incident's own confirmed description, never guessed from reading the current code. Because this program cannot itself perform deep semantic invariant extraction, every generated fixture is honestly marked '[broad-invariant]' in its title and given the full permissive allowed_mutations set (wording/formatting/file_path/symbol_name) -- the plan's own escape valve for a description too thin to yield a precise invariant, applied uniformly rather than pretending selective precision a mechanical pass cannot back up."
  - "The defect register reader parses WINDOWS.md's own fenced ```json ledger dump rather than its markdown table -- the file's own table has a stray, out-of-band duplicate row after the JSON block closes (unrelated pre-existing content), so the JSON dump is the only well-formed, unambiguous source."
  - "Audit-finding provenance identifiers are the entry's own array index (\"confirmed[N]\"), not its \"subject\" text -- 5 of the 42 verified subjects in the real audit file are not unique, so subject text cannot serve as a resolvable identifier."
  - "Phase-report provenance is parsed at markdown-table-row granularity, not per individual test name. 203-VERIFICATION.md's own Fixed/Cause/Commit table names 9 distinct test fixes across 6 rows (one row groups 4 related failures under one shared root cause; one row names two commits for one fix) -- the reader keeps the row as the atomic confirmed unit, since that is the shape the report itself confirmed."
  - "TDD compliance note: Task 1 and Task 2 (tdd=\"true\") were each committed as a single feat commit carrying both the implementation and its tests, rather than separate test(RED)/feat(GREEN) commits -- their own action text defines the schema/types and the tests together as one task, with no meaningful pre-implementation red state for declarative schema/type work. Both tasks' genuine fail-capability was proven live rather than by commit history: Task 1 by temporarily adding an undeclared severity value to schema.json and watching TestFixtureVocabulariesAreClosed fail, then reverting; Task 2 by temporarily disabling the failure-log confirmation gate and watching TestUnconfirmedIncidentProducesNoFixture fail, then reverting. Both reverts were verified back to green before committing."

patterns-established:
  - "confirmedIncident as the one normalised input shape for a multi-source, confirmation-gated conversion -- each reader applies its own gate at the point of reading and returns both what it confirmed and what it refused (by identifier and reason), so a conversion never silently narrows its input."

requirements-completed: [LEARN-04]

coverage:
  - id: D1
    description: "A versioned fixture-bank schema (cmd/testdata/fixture-bank/v1/schema.json) that a fixture cannot enter without naming its failing baseline, invariant, allowed mutations, provenance, privacy disposition and severity, and where a retirement always names a successor."
    requirement: LEARN-04
    verification:
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestFixtureBankSchemaRejectsIncompleteFixtures"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestFixtureRetirementRequiresASuccessor"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestFixtureVocabulariesAreClosed"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestEmptyFixtureBankIsValid"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestFixtureBankOrderingIsTotalAndStable"
        status: pass
    human_judgment: false
  - id: D2
    description: "Confirmed incidents from four sources (failure log, audit findings, defect register, phase closing reports) convert into deduplicated, sanitised, provenance-carrying fixtures; unconfirmed incidents produce nothing and the refusal names the missing confirmation fact."
    requirement: LEARN-04
    verification:
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestUnconfirmedIncidentProducesNoFixture"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestSameIncidentTwiceProducesOneFixture"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestTwoIncidentsWithOneDigestShareAFixture"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestUnsanitisableIncidentIsDroppedNotStored"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestRetirementNamesASuccessorAndKeepsTheFixture"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestEmptyIncidentSetWritesAnEmptyBank"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestFixtureBankHasOneWriter"
        status: pass
    human_judgment: false
  - id: D3
    description: "A generated, reproducible fixture bank seeded from this repository's own confirmed failures (46 fixtures from 42 verified audit findings, 9 fixed defect-register entries and 6 phase-report rows), every fixture provably tracing to a real, still-confirmed source entry."
    requirement: LEARN-04
    verification:
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestSeededBankValidatesAgainstItsSchema"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestSeededBankFixturesAllCiteRealProvenance"
        status: pass
      - kind: unit
        ref: "cmd/fixture_conversion_test.go#TestSeededBankIsReproducible"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 05: Fixture Conversion Summary

**A versioned regression-fixture bank, a confirmation-gated conversion from four real sources, and a generated 46-fixture seed drawn entirely from this repository's own already-confirmed failures -- so a defect this project has already agreed was real now has something standing guard against it recurring.**

## Performance

- **Duration:** ~40 min (estimate; commit span 12:23-12:33 plus prior reading/research)
- **Tasks:** 3/3 completed
- **Files created:** 4 (cmd/fixture_conversion.go, cmd/fixture_conversion_test.go, schema.json, bank.json)
- **Tests:** 15 new tests, all passing

## Accomplishments

- `cmd/testdata/fixture-bank/v1/schema.json` -- a JSON Schema (draft 2020-12), same shape as the existing Classic contract schema beside it, requiring `failing_baseline`, `invariant`, `allowed_mutations`, `provenance`, `privacy`, `severity` and `content_digest` on every fixture, and requiring `retired_by`/`successor_id` together or not at all.
- `cmd/fixture_conversion.go` -- the four declared closed vocabularies (allowed mutations, provenance kind, privacy, severity), the `regressionFixture`/`regressionFixtureBank` types, `loadFixtureBank`/`fixtureBankSortedByID`, four confirmation-gated readers (`confirmedIncidentsFromFailureLog`, `confirmedIncidentsFromAuditFindings`, `confirmedIncidentsFromDefectRegister`, `confirmedIncidentsFromPhaseReports`), `convertConfirmedIncidentsToFixtures` (sanitiser reuse, sha256 digest dedup), `writeFixtureBank` (the bank's one writer) and `retireFixture` (names a successor, never deletes).
- `cmd/testdata/fixture-bank/v1/bank.json` -- generated (not hand-authored), 46 fixtures from this repository's own 42 verified audit findings, 9 fixed defect-register entries and 6 phase-report rows; regenerates byte-identically from its sources (`TestSeededBankIsReproducible`).

## Task Commits

Each task was committed atomically:

1. **Task 1: Define the versioned fixture bank and its schema** - `5d343336` (feat)
2. **Task 2: Convert confirmed incidents into fixtures, refusing everything unconfirmed** - `071e81e6` (feat)
3. **Task 3: Seed the bank from this repository's own confirmed failures** - `1481edf9` (feat)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `cmd/fixture_conversion.go` - Schema-validation, readers, conversion, writer, retirement
- `cmd/fixture_conversion_test.go` - 15 tests across all three tasks
- `cmd/testdata/fixture-bank/v1/schema.json` - The versioned fixture-bank schema
- `cmd/testdata/fixture-bank/v1/bank.json` - The generated, seeded fixture bank

## Reader counts (Task 3's own completion notes, per acceptance criteria)

| Source | Found | Refused | Refusal reason |
|---|---|---|---|
| failure_log | 0 | 0 | No local `midden.json` exists in this worktree's `.aether/data` -- honestly zero, not padded |
| audit_finding | 42 | 242 | `verified` flag not set |
| defect_register | 9 | 28 | status other than `fixed` |
| phase_report | 6 | 0 | (all 6 rows in 203-VERIFICATION.md's Fixed table name a commit) |

- **Deduplicated:** 0 (no two incidents in this run shared a sanitised-content digest)
- **Dropped by the sanitiser:** 11 of 60 candidates (10 audit findings, 1 defect-register entry) -- all for containing backtick-quoted code (flagged as shell-injection patterns) or XML-like tags in their prose, the sanitiser working as designed
- **Final fixture count:** 46 (exceeds the 43 the requirement names; nothing invented to reach it)

## Decisions Made

See `key-decisions` in the frontmatter above for the full account. Summary:

1. Pre-truncate incident text to the sanitiser's 500-char budget before calling it, since untruncated real prose here would otherwise be refused for length on nearly every incident.
2. Every generated fixture is honestly marked `[broad-invariant]` with the full permissive mutation set, because this program cannot itself derive a precise code invariant from prose -- an explicit, uniform application of the plan's own escape valve rather than selective, unsupportable precision.
3. Defect-register parsing uses WINDOWS.md's fenced JSON ledger, not its markdown table, because the table has a stray unrelated duplicate row after the JSON block.
4. Audit-finding provenance uses array index (`confirmed[N]`), not subject text, because 5 of 42 verified subjects are not unique.
5. Phase-report provenance is row-granular (6 rows / 9 named fixes), matching the shape the report itself confirmed.

## Deviations from Plan

### Judgment calls (not Rule 1-3 auto-fixes)

**1. [Judgment call] TDD commit granularity for Tasks 1 and 2**
- **Found during:** Tasks 1 and 2 (both `tdd="true"`)
- **Issue:** Both tasks' own `<action>` text defines schema/types and tests together in one coherent description, with no clean pre-implementation RED state for declarative schema/vocabulary work (there is no meaningful "failing test" before a JSON Schema or a Go type exists to validate against).
- **Fix:** Committed each task as a single `feat` commit carrying both implementation and tests, then independently proved each task's key test can genuinely fail: Task 1 by temporarily adding an undeclared `severity` enum value to `schema.json` and watching `TestFixtureVocabulariesAreClosed` fail (then reverting to a confirmed-green state); Task 2 by temporarily disabling the failure-log confirmation gate and watching `TestUnconfirmedIncidentProducesNoFixture` fail (then reverting to a confirmed-green state).
- **Files modified:** None beyond the task's own declared files (the fail-proof edits were reverted before committing).
- **Verification:** Both live before/after checks logged above; final `go test` runs after each revert confirmed green before committing.
- **Committed in:** `5d343336`, `071e81e6`

---

**Total deviations:** 1 judgment call (TDD commit-granularity documentation, no code bug). **Impact on plan:** None on any machine-checkable gate -- every `<acceptance_criteria>` line and the full `<verify>` command for all three tasks pass.

## Issues Encountered

None beyond the judgment call above. `go build ./...` and `go vet ./cmd/... ./pkg/...` are clean; the full targeted 15-test group passes with no regressions.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

The fixture bank exists, validates, regenerates reproducibly, and is provenance-clean. Plans `204-06` through `204-11` can now reference `regressionFixture`/`regressionFixtureBank`, `loadFixtureBank`, and `retireFixture` as an established, tested foundation. `retireFixture` is implemented and tested but has no CLI-level caller yet -- any later plan wiring a retirement workflow should call it directly rather than duplicating its successor-existence check.

No blockers.

## Self-Check: PASSED

- `cmd/fixture_conversion.go` exists: FOUND
- `cmd/fixture_conversion_test.go` exists: FOUND
- `cmd/testdata/fixture-bank/v1/schema.json` exists: FOUND
- `cmd/testdata/fixture-bank/v1/bank.json` exists: FOUND
- `git log --oneline --all --grep="204-05"` returns 3 commits (5d343336, 071e81e6, 1481edf9): FOUND
- All 15 named tests re-run clean: PASSED (`go test ./cmd -run '^(TestFixtureBankSchemaRejectsIncompleteFixtures|TestFixtureRetirementRequiresASuccessor|TestFixtureVocabulariesAreClosed|TestEmptyFixtureBankIsValid|TestFixtureBankOrderingIsTotalAndStable|TestUnconfirmedIncidentProducesNoFixture|TestSameIncidentTwiceProducesOneFixture|TestTwoIncidentsWithOneDigestShareAFixture|TestUnsanitisableIncidentIsDroppedNotStored|TestRetirementNamesASuccessorAndKeepsTheFixture|TestEmptyIncidentSetWritesAnEmptyBank|TestFixtureBankHasOneWriter|TestSeededBankValidatesAgainstItsSchema|TestSeededBankFixturesAllCiteRealProvenance|TestSeededBankIsReproducible)$' -count=1 -timeout 90m`)
- `go build ./...`: PASSED
- `go vet ./cmd/... ./pkg/...`: PASSED

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
