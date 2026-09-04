---
phase: 199-front-door-and-classic-contract
plan: "27"
subsystem: testing
tags: [go, contract-testing, command-parity, claude, opencode]
requires:
  - phase: 199-front-door-and-classic-contract
    provides: Finalized public lifecycle runtime and managed wrapper surfaces
provides:
  - 43 required semantic journeys expanded to 86 Claude/OpenCode runtime cases
  - Strict corpus cardinality, causal-evidence, managed-wrapper, and replay validation
  - Ordered public command inventory ratcheted across canonical and platform surfaces
affects: [199-28, 199-29, phase-199-verification]
tech-stack:
  added: []
  patterns: [versioned semantic corpus, platform-paired runtime proof, public command inventory ratchet]
key-files:
  created: [cmd/testdata/classic-contract/v1/cases.json]
  modified: [cmd/classic_contract_test.go, cmd/classic_command_parity_test.go, .aether/commands/classic-command-parity.json]
key-decisions:
  - "A semantic journey requires exactly one Claude and one OpenCode case, with platform-expanded cardinality enforced before execution."
  - "Only public names backed by canonical YAML, managed wrappers, and a Cobra command enter the static inventory; raw protocol compatibility remains outside it."
patterns-established:
  - "Causal state digest checks run before and after isolated Go-runtime executions; visual tokens remain supplementary evidence."
requirements-completed: [SYNTH-01, CEC-01, CEC-02, CEC-04, CEC-08, LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05, LIFE-06, PROOF-01]
duration: 12min
completed: 2026-09-04
---

# Phase 199 Plan 27: Executable Classic Contract Summary

**A versioned 86-case Claude/OpenCode corpus now locks every required Phase 199 journey and a separate manifest ratchets the supported public command vocabulary.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-04T23:41:42Z
- **Completed:** 2026-09-04T23:52:59Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added all 43 required semantic journey identifiers in separate platform pairs, including seven independently removable front-door/territory journeys.
- Executes the isolated Go runtime for each Claude/OpenCode variant, with managed-wrapper authority, JSON, visual-beat, absence, and digest evidence checks.
- Replaced the broad legacy command matrix with a 20-command final public inventory checked against canonical YAML, both managed platforms, and Cobra.

## Task Commits

1. **Task 1: Populate and enforce every semantic journey category** - `6be7114a`, `dfe5988e` (test, feat)
2. **Task 2: Ratchet the final public command inventory** - `a20ed000`, `2ddbcafc` (test, feat)

## Verification

- `go test ./cmd -run '^TestClassicContractCorpus(RequiredCategories|RejectsMissingFrontDoorCase|Claude|OpenCode|CausalReceipts)$' -count=1` — PASS (98 assertions/subtests)
- `go test ./cmd -run '^TestClassicCommandParity$' -count=1` — PASS
- Combined focused contract/parity test command — PASS (99 assertions/subtests)
- `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1` — PASS (7 assertions/subtests)
- `go build ./cmd/aether` — PASS

## Decisions Made

- The seven front-door/territory scenarios are explicit independent IDs, not a countable combined bucket.
- Dream and interpret remain outside the public ratchet because no corresponding Cobra command currently exists; the manifest only admits end-to-end public names.

## Deviations from Plan

None - plan executed as a test-first contract and parity ratchet.

## Issues Encountered

- A focused race invocation outlived its terminal capture and was explicitly stopped; it made no source changes. The ordinary focused and source-hygiene checks passed.

## Next Phase Readiness

- Plan 28 can use the versioned corpus and public inventory as the fixed current-surface inputs.

## Self-Check: PASSED

- Found corpus, contract test, parity manifest, and parity test on disk.
- Found task commits `6be7114a`, `dfe5988e`, `a20ed000`, and `2ddbcafc` in Git history.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
