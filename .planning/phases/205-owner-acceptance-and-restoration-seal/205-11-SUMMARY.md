---
phase: 205-owner-acceptance-and-restoration-seal
plan: 11
subsystem: testing
tags: [go, classic-coverage, proof-02, proof-03, proof-04]

requires:
  - phase: 205-01
    provides: Shared ledger routing, phase validators and AST proof resolution
  - phase: 205-04
    provides: Documentation accuracy and seal/entomb wrapper proofs
  - phase: 205-07
    provides: Signed Phase 200, 201 and 202 coverage
  - phase: 205-08
    provides: Signed Phase 204 coverage and bounded re-adjudication check
  - phase: 205-09
    provides: Four-dimension parity record and its integration checks
  - phase: 205-10
    provides: Three captured platform runs, honesty cards and citation checks
provides:
  - Phase 205's two signed capability records with ten individually executed proofs
  - Ledger-derived master validation of all 72 capabilities across seven phase files
  - Demonstrated rejection of missing, duplicate, misplaced, extra and invalid-evidence records
affects: [205-restoration-verification, 205-limitations-card, 205-owner-acceptance]

actuals:
  tokens: 4735
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - Derive the capability universe and required phase files from the frozen ledger
    - Compose the shared phase validator with cross-file uniqueness and routing checks
    - Mutate deep copies and temporary roots while leaving signed files unchanged

key-files:
  created:
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-CLASSIC-COVERAGE.json
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-CLASSIC-COVERAGE.md
    - cmd/classic_coverage_master_test.go
  modified: []

key-decisions:
  - "The ledger routes 72 ids to seven phase files, not six: 199, 200, 201, 202, 203, 204 and 205. Derivation takes precedence over the plan's arithmetic error."
  - "Both Phase 205 dispositions match the frozen ledger; no re-adjudication marker or allowance change was needed."
  - "The master retains Phase 199's non-GOAL records for shared validation, but only GOAL records contribute to capability counts."
  - "The master success test also invokes the existing bounded re-adjudication check; a marker cannot silently bypass the frozen dispositions."

requirements-completed: [PROOF-02, PROOF-03, PROOF-04]

coverage:
  - id: D1
    description: "Phase 205's documentation and platform-readiness records cite ten tests run individually before signing, with a rendered companion and explicit evidence limits."
    requirement: PROOF-02
    verification:
      - kind: integration
        ref: "cmd/classic_coverage_master_test.go#TestClassicCoveragePhase205Signed"
        status: pass
      - kind: other
        ref: "All ten individually executed commands in the Row Proof Verification Log below"
        status: pass
    human_judgment: false
  - id: D2
    description: "The master check validates every ledger-routed capability exactly once in its assigned phase, using shared evidence validators and whole-number counts."
    requirement: PROOF-02
    verification:
      - kind: integration
        ref: "cmd/classic_coverage_master_test.go#TestClassicCoverageMasterUnionIsCompleteAndUnique"
        status: pass
      - kind: unit
        ref: "cmd/classic_coverage_master_test.go#TestClassicCoverageMasterReportsCountsNotPercentages"
        status: pass
    human_judgment: false
  - id: D3
    description: "Missing, duplicate, misplaced, extra and invalid-evidence records fail by name; diagnostics stay ordered and stable, and coverage/parity integration executes every discovered check."
    requirement: PROOF-02
    verification:
      - kind: unit
        ref: "cmd/classic_coverage_master_test.go#TestClassicCoverageMasterFailureOutputIsOrderedAndStable"
        status: pass
      - kind: unit
        ref: "cmd/classic_coverage_master_test.go#TestClassicCoverageMasterRejectsInvalidEvidence"
        status: pass
      - kind: integration
        ref: "go test ./cmd/ -run 'TestClassicCoverage|TestClassicParity' -count=1 -timeout 90m -json"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-09-15
status: complete
---

# Phase 205 Plan 11: Complete Capability Reconciliation Summary

**Signed the final two capability records and added one combined check that verifies all 72 identities, their assigned phases and their supporting evidence across seven files.**

In plain English: the inventory now checks that every promised capability has one
documented home and real supporting checks. A matching total alone cannot hide a
duplicate, a misplaced entry or a made-up proof. This completes the assigned
code/evidence plan; the owner's later walk-through and acceptance remain separate.

## Performance

- **Recorded execution start:** 2026-09-15T21:17:42Z (after initial context loading)
- **Implementation and verification completed:** 2026-09-15T21:25:14Z
- **Duration:** approximately 8 minutes from the recorded execution clock
- **Tasks:** 2
- **Task deliverable files:** 3; this summary is the fourth committed file
- **Actuals basis:** 18,939 characters in the task-commit diff, rounded up after dividing by four; excludes this metadata summary and counts the two task commits.

## Accomplishments

- Added `205-CLASSIC-COVERAGE.json` with version `205/classic-coverage/v1` and exactly two GOAL records: CAP-023 uses the ledger's `restore-modern` disposition; CAP-054 uses `replace-better`. All ten cited proofs were run separately before signing, and the Markdown companion renders both records.
- Added the master check using `loadClassicCoverageLedgerRouting`, `loadClassicCoveragePhase`, `validateClassicCoverage` and the existing re-adjudication bounds check. The expected identities and phase files come from the ledger. The successful output is: `72 distinct capability ids; 72 capability rows; 72 expected ids; 7 phase files.`
- Demonstrated failures for one removed record, a duplicate, a record moved to the wrong phase, a new identity outside the ledger, a nonexistent proof in each of the seven phase documents, an invalid disposition, a nonpassing status, an empty home and a nonexistent artifact. The report-format guard rejects percentages, decimals and proportions. Multiple faults report their ids in ascending order, identically across repeated runs and reordered input. The targeted coverage/parity integration passed all 40 discovered top-level tests.

## Task Commits

1. **Task 1: Phase 205's two signed records** — `d924fbe8df13a254b6069f92045e83411b5d1c1e` (`feat(205-11): sign documentation and platform capability proofs`)
2. **Task 2: Master cross-phase check** — `759cc4461648a825bd45b16bab01a95105e2ce9b` (`test(205-11): verify the complete classic capability union`)

Both commits ran normally with hooks enabled. This summary is committed separately
after both task commits. Shared STATE, ROADMAP and REQUIREMENTS files were left to
the parent executor.

## Files Created

- `205-CLASSIC-COVERAGE.json` — two machine-validated records, their evidence and scope limits.
- `205-CLASSIC-COVERAGE.md` — readable table and explanation of what the evidence establishes.
- `cmd/classic_coverage_master_test.go` — Phase 205 validation, master checks, count reporting and fault-injection tests.

## Row Proof Verification Log

Every command below ran separately, with the shown result, before the corresponding
record was written. RTK only filters command output; the underlying runner is
`go test`.

| Record | Command | Observed result |
|---|---|---|
| CAP-023 | `rtk go test ./cmd/ -run '^TestSealWrapperReviewClaimMatchesTheRuntime$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-023 | `rtk go test ./cmd/ -run '^TestSealWrapperTripletStaysIdentical$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-023 | `rtk go test ./cmd/ -run '^TestSealAndEntombWrappersRelayTheCard$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-023 | `rtk go test ./cmd/ -run '^TestEntombWrapperTripletStaysIdentical$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestOpenCodeCardClaimsResolveToTheCapturedRun$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestCodexCardClaimsResolveToTheCapturedRun$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestClaudeCardClaimsResolveToTheCapturedRun$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestEveryContractedPlatformHasACardAndARun$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestPlatformCardsDoNotClaimUngovernedAbilityAsGoverned$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| CAP-054 | `rtk go test ./cmd/ -run '^TestPlatformCardsPromiseNoFutureWork$' -count=1 -timeout 90m -v` | PASS (exit 0) |

### Scope of the two records

- **CAP-023 / PROOF-03:** `cmd/seal_wrapper_accuracy_test.go` checks the finish command's review-team claim against the program's own derivation, keeps its source and platform copies aligned, and checks the instructions to relay the finish/archive output. These checks do not establish that the owner actually sees that output in a live chat; Journey 1 retains that acceptance obligation.
- **CAP-054 / PROOF-04:** `cmd/platform_honesty_card_test.go` binds the three platform cards to their captured runs and rejects unsubstantiated tracking claims and future promises. The artifact remains `pkg/codex/platform_contract.go`. Claude Code and Codex reached status; OpenCode stopped at a provider connection error. These captures do not establish worker dispatch, named-helper routing or full lifecycle parity. No new provider run or contract rewrite was needed.

## Master and Integration Verification

### Individually executed checks

| Command | Observed result |
|---|---|
| `rtk go test ./cmd/ -run '^TestClassicCoveragePhase205Signed$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `go test ./cmd/ -run '^TestClassicCoverageMasterUnionIsCompleteAndUnique$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailsOnAMissingRow$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailsOnADuplicateRow$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailsOnAMisroutedRow$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailsOnAnExtraRow$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterReportsCountsNotPercentages$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailureOutputIsOrderedAndStable$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterRejectsInvalidEvidence$' -count=1 -timeout 90m -v` | PASS (exit 0) |
| `rtk go test ./cmd/ -run '^TestClassicCoverageMasterFailsWhenPhaseFilesAreAbsent$' -count=1 -timeout 90m -v` | PASS (exit 0) |

Before adding the two coverage records, `TestClassicCoveragePhase205Signed` failed
because the Phase 205 coverage file was absent. After the files were written it
passed, and it passed again after Task 1's commit as the autonomous tracer check.
No user preference or new approval was needed for this already-authorized
code/evidence step.

The missing-record and extra-identity fixtures exercise 71 and 73 distinct ids
against the ledger's 72. The misplaced-record fixture preserves the total but
still fails. Phase 199's other record types remain present and do not inflate the
capability total.

### Combined integration

```sh
go test ./cmd/ -run 'TestClassicCoverage|TestClassicParity' -count=1 -timeout 90m -json
```

**PASS: discovered 40, executed 40, passed 40; zero failures, zero omitted tests.**
Discovery was independently collected using `go test ./cmd/ -list
'TestClassicCoverage|TestClassicParity' -timeout 90m` and compared with the JSON
run and pass events.

Local run evidence:

- `/tmp/aether-phase205-execution/plan11-integration-list.txt`
- `/tmp/aether-phase205-execution/plan11-integration.jsonl`
- `/tmp/aether-phase205-execution/plan11-integration-result.json`

Additional checks passed:

- `go vet ./cmd/...`
- `go build -o /tmp/aether-phase205-execution/plan11-aether ./cmd/aether`
- Exact JSON version, two GOAL ids, rendered table rows, and all ten proof names declared in the two required test files.
- `git diff --check`.
- `git status --porcelain .planning/phases/` returned no changes after the tests: no signed coverage file was modified.
- The master test contains no literal capability-id list or hard-coded expected universe count.

The full repository and race release gates were not run; the parent owns the
serialized release gate.

## Decisions Made

- Included all seven files derived from the ledger. A phase with no routed ids
  needs no special case and is not required to carry a coverage file.
- Reused the shared phase loader and validator for every file, including Phase
  199's complete document. The master adds cross-file ownership checks rather
  than copying the evidence-validation implementation.
- Kept the two frozen dispositions unchanged. Both records describe the actual
  evidence limits, without converting a recorded platform connection failure or
  a pending owner-observed outcome into a success claim.
- Added three further tests within the owned master-test file for an extra
  identity, invalid evidence and absent phase documents. They directly prove the
  plan's exact-universe and evidence requirements.

## Deviations from Plan

### 1. [Rule 1 - Planning arithmetic] Seven routed files, not six

- **Found during:** Task 1 context reading; enforced in Task 2.
- **Issue:** The plan and research diagram say six coverage files while requiring
  every ledger-routed phase, including Phase 199 and Phase 205. The ledger routes
  to seven files: 199 (34 records), 200 (7), 201 (8), 202 (8), 203 (6), 204 (7),
  and 205 (2), totaling 72.
- **Fix:** Derived the phase set from the ledger as the plan's implementation
  instructions require. No phase was dropped to fit the erroneous file count.
- **Files modified:** `cmd/classic_coverage_master_test.go`; this summary records
  the correction. The plan and frozen ledger were not edited.
- **Verification:** Master output reports 72 distinct ids across 7 phase files;
  all master and coverage/parity integration checks pass.
- **Commit:** `759cc446`.

**Total deviations:** one arithmetic correction; no out-of-scope file edits and
no production behavior changes.

## Issues Encountered

The initial filesystem sandbox prevented Go from writing its normal build cache,
so that first attempt did not execute a proof. The parent restored the executor's
authorized unrestricted settings; the tests then ran successfully with the normal
cache. An interrupted temporary-cache attempt was not counted as a passing run.
No repository workaround was retained.

## Remaining Limits and Next Phase Readiness

- Plan 205-11 is complete and its records and checks are ready for the parent to
  integrate into the release verification.
- The captured OpenCode connection failure and the narrow scope of all three
  status captures remain explicit.
- Owner-visible delivery of the finish/archive card and the owner's overall
  acceptance remain for the later walk-through.
- CAP-057's pre-existing numeric-threshold precision limit and the parity record's
  honest “not evidenced” entries are preserved; complete traceability does not
  erase those limits.
- Requirement ids in this summary are the plan's required metadata. Shared
  milestone tracking and final acceptance remain parent-owned.

## User Setup Required

None.

## Self-Check: PASSED

- Both task commits and all three task deliverables were verified present.
- All ten row proofs passed individually; all ten new named checks passed.
- All 40 targeted integration tests were discovered, executed and passed.
- No placeholder implementation or skipped test was introduced.
- Worktree: `/Users/callumcowie/repos/Aether/.claude/worktrees/agent-p205-11-codex-20260915`
- Branch: `worktree-agent-p205-11-codex-20260915`
