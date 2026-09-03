---
phase: 199-front-door-and-classic-contract
plan: "02"
subsystem: planning-and-proof
tags: [classic-restoration, json-schema, go-tests, generated-surfaces]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "01"
    provides: Exact-ownership cleanup that makes focused cmd tests safe
  - artifact: .planning/phases/199-front-door-and-classic-contract/199-RESEARCH.md
    provides: Completed Classic/current mechanism comparison and selected dispositions
provides:
  - Signed ten-decision Classic mechanism synthesis with all 34 Phase 199 capability routes
  - Strict classic-contract/v1 semantic and causal case schema
  - Machine-validated mechanism registry covering every selected decision and routed capability
  - Aggregate command-source hygiene gate defined before generated-surface implementation plans
affects: [199-front-door, classic-contract-corpus, claude-wrappers, opencode-wrappers, source-check]

tech-stack:
  added: []
  patterns: [strict JSON decoding, semantic-plus-causal proof, production-sync dry run, managed-peer parity]

key-files:
  created:
    - .planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md
    - cmd/classic_contract_test.go
    - cmd/testdata/classic-contract/v1/schema.json
    - cmd/testdata/classic-contract/v1/mechanisms.json
  modified:
    - cmd/command_source_hygiene_test.go

key-decisions:
  - "Restore Classic's information grammar and ceremony through modern Go-owned truth rather than restoring prompt-owned writes or shell-era state handling."
  - "Every behavior corpus case must combine semantic assertions with at least one causal state assertion or forbidden-artifact assertion."
  - "Command-source hygiene composes production YAML discovery, exact managed headers, platform sync rules, and Claude/OpenCode peer parity while ignoring unmanaged paths."

patterns-established:
  - "Signed synthesis: every runtime change traces to one evidence-backed mechanism disposition, owner decision, capability route, and verification ID."
  - "Exact coverage diagnostics: missing or duplicate synthesis and capability identifiers are named directly by the validator."
  - "Automation-first hygiene: regenerate into disposable directories and compare managed peers without repairing or deleting source files."

requirements-completed: [SYNTH-01, PROOF-01]

duration: 18min
completed: 2026-09-03
---

# Phase 199 Plan 02: Classic Mechanism and Proof Contract Summary

**A signed Classic-to-modern mechanism map now governs Phase 199, backed by a strict semantic/causal corpus schema and an aggregate source-hygiene gate for every managed Claude/OpenCode command family.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-09-03T14:02:25Z
- **Completed:** 2026-09-03T14:20:25Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Published the signed synthesis for all ten Classic mechanisms, 34 routed Phase 199 capabilities, and 17 locked owner decisions, including explicit boundaries for Phases 200–205 and Codex-native `$ant-*` work.
- Defined a closed JSON Schema 2020-12 contract for real public behavior cases: strict command/environment input, expected semantic fields, limited text tokens, causal state/filesystem assertions, replay behavior, and fault points.
- Added a typed Go loader that rejects unknown fields, invalid groups/platforms, duplicate case IDs, and output-only behavior cases.
- Registered all ten synthesis mechanisms and all 34 capabilities with disposition, public commands, source citations, and ownership across the eight approved corpus groups.
- Added the exact `TestCommandSourceHygiene` gate, with independent fixtures for missing sources, missing peers, stale bodies, false headers, managed orphans, and preserved unmanaged custom files.

For non-specialists: this plan writes the recipe and the exam before later plans change the product. A future change cannot count as restored merely because the command exists or the screen looks right—it must produce the promised machine-readable result and the promised real state change.

## Task Commits

Each task was committed atomically; test-first tasks retain separate RED and GREEN commits:

1. **Task 1: Publish the cited Phase 199 mechanism synthesis** - `934e7b84` (docs)
2. **Task 2 RED: Add failing Classic corpus contract tests** - `ca260f30` (test)
3. **Task 2 GREEN: Define the Classic corpus v1 contract** - `7a0dc152` (feat)
4. **Task 3 RED: Add the failing aggregate source-hygiene gate** - `035e9343` (test)
5. **Task 3 GREEN: Enforce aggregate command-source hygiene** - `caf450cb` (feat)

**Plan metadata:** This summary is committed immediately after creation; planning-state progress is recorded in the following closeout commit.

## Files Created/Modified

- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md` - Signed Classic/current comparison, selected mechanisms, capability routing, owner-decision trace, verification IDs, and phase boundaries.
- `cmd/classic_contract_test.go` - Strict typed case/registry loaders, validation rules, exact coverage diagnostics, and read-only fixture-digest proofs.
- `cmd/testdata/classic-contract/v1/schema.json` - Closed versioned schema for semantic output, causal state/filesystem assertions, replay, and fault injection.
- `cmd/testdata/classic-contract/v1/mechanisms.json` - Ten-record registry covering every approved group and routed capability.
- `cmd/command_source_hygiene_test.go` - Exact aggregate gate using production source discovery, managed-header parsing, copy-based platform sync rules, and normalized peer parity.

## Decisions Made

- Classic value is preserved as a causal experience contract—identity, goal, actors/stage, evidence, unresolved truth, and one next action—not as byte-for-byte historical wrappers.
- The modern Go runtime remains the sole durable authority; Claude/OpenCode wrappers may guide and render but cannot create proof through prose.
- The two cross-cutting mechanisms with no uniquely assigned Phase 199 capability retain explicit empty `cap_ids` arrays; the other eight records cover the exact 34-row set once each.
- Managed wrapper bodies are checked through the repository's actual copy-based platform sync and cross-platform semantic parity. Custom files outside the managed source directories are ignored and preserved.

## Verification

- The synthesis audit passed with exactly 10 synthesis identifiers, 34 routed capability rows, 17 owner-decision rows, `GOAL`/`SYNTH-01` coverage, and every explicit phase/platform boundary.
- `go test ./cmd -run '^(TestClassicContractSchema|TestClassicMechanismCoverage)$' -count=1` passed 11 focused cases.
- The exact source-hygiene selector exists once and `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1` passed 7 focused cases.
- The combined final gate `go test ./cmd -run '^(TestClassicContractSchema|TestClassicMechanismCoverage|TestCommandSourceHygiene)$' -count=1` passed all 18 cases.
- Existing tests in `cmd/command_source_hygiene_test.go` also passed together with the new aggregate gate (11 cases).
- Both JSON artifacts pass `jq empty`; all five plan-owned files pass whitespace validation.

## Deviations from Plan

None - the plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None - this plan intentionally defines contract infrastructure and routing metadata; it does not claim that later Phase 199 behavior cases or product implementations already exist.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later runtime plans have a signed, evidence-backed source for every Phase 199 mechanism and capability decision.
- Later corpus cases can use the strict v1 case shape without weakening causal proof requirements.
- Every generated-surface plan can select a real aggregate hygiene test; the selector cannot silently pass by matching zero tests.
- Phase 199 Plan 03 can proceed.

## Self-Check: PASSED

- All five planned artifacts exist at their declared paths.
- All five task/TDD commits exist in Git history.
- Focused synthesis, schema, mechanism-coverage, and command-source hygiene gates pass.
- The protected `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remain uncommitted and untouched.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
