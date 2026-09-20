---
phase: 200-iterative-planning
plan: 22
subsystem: planning
tags: [planning, specification, contracts, codex, wrappers, candidates, replay]

requires:
  - phase: 200-20
    provides: restored Specification and material-decision public journey
  - phase: 200-21
    provides: synchronized Claude/OpenCode staged planning projections
  - phase: 200-24
    provides: restored Init-to-Discuss front-door handoff
  - phase: 200-25
    provides: complete runtime-native Specification command surface
provides:
  - Renderer-neutral Specification, staged planning, candidate, replay, and exact-acceptance contracts
  - Codex command-guide and build-cycle instructions for one Go-authorized planning stage at a time
  - Focused contract, lifecycle-wrapper, guide, severity, and source-parity regression coverage
affects: [200-23, planning, specification, command-guide, codex, wrapper-hosts]

tech-stack:
  added: []
  patterns: [runtime-owned authority, receipt-bound stage dispatch, exact replay, inactive candidate review, explicit exact acceptance]

key-files:
  created:
    - cmd/contracts/spec.md
    - cmd/planning_contract_docs_200_test.go
  modified:
    - cmd/contracts/plan.md
    - .aether/docs/wrapper-host-contract.md
    - cmd/lifecycle_wrapper_contract_test.go
    - cmd/command_guide.go
    - cmd/command_guide_test.go
    - .aether/skills/colony/aether-colony-build-cycle/SKILL.md
    - cmd/command_call_audit_test.go
    - cmd/visual_wrapper_contract_test.go
    - cmd/contract_validate_test.go

key-decisions:
  - "Codex planning guidance dispatches only the single Scout or Route-Setter stage currently authorized by Go; it never predicts or combines stages."
  - "Specification approval, planning stop, and exact candidate acceptance are separate owner boundaries, and only an acceptance receipt exposes build or Autopilot."
  - "The Specification is runtime-native in Codex (`aether spec`), while Claude/OpenCode retain `/ant-spec` projections over the same Go authority."
  - "The canonical Plan YAML owns the singular wrapper-host contract pointer; managed projections need not duplicate it."

patterns-established:
  - "Staged planning: approved Specification -> explicit preset -> one stage manifest/result -> Go receipt -> runtime-selected boundary."
  - "Safe recovery: exact replay retains the existing artifact; divergent replay leaves state unchanged and returns the exact recovery action."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

duration: 24 min
completed: 2026-09-08
---

# Phase 200 Plan 22: Public Planning Contracts and Codex Guidance Summary

**Specification, staged planning, candidate review, and exact acceptance now share one public authority model across runtime contracts, wrapper hosts, and Codex orchestration.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-09-08T01:40:02Z
- **Completed:** 2026-09-08T02:04:12Z
- **Tasks:** 2
- **Files created/modified:** 11

## Accomplishments

- Published complete renderer-neutral contracts for the nine-part Specification body, inspect/revise/approve/projection repair, four planning presets, Scout and Route-Setter stage manifests/results, material decision branches, causal iteration cards, candidate review, Queen recommendation evidence, and receipt-bound acceptance.
- Defined exact IDs, hashes, receipts, refusal classes, idempotent replay, divergent replay refusal, recovery commands, and state ownership in plain language for runtime callers and wrapper authors.
- Migrated the wrapper-host contract and lifecycle tests away from retired depth cards, phase-research approval, whole-chain completion packets, and wrapper-authored state transitions while preserving flat mirrors, structured blocks, read-only boundaries, and forbidden-write checks.
- Rewrote Codex Plan guidance and the canonical build-cycle skill around approved-Specification preflight, unbiased preset selection, one authorized stage at a time, complete decision/iteration cards, inactive candidate review, and exact owner acceptance before `aether build 1` or `aether run`.
- Added runtime-native `spec` command-guide coverage and registered its enrichment severity so the command catalogue and command-call audit recognize the restored lifecycle surface.
- Preserved generated-source parity across all 126 managed wrappers and aligned legacy visual-wrapper and contract-inventory gates with the Phase 200 authority boundary.

## Task Commits

Each planned task was committed atomically after its focused verification passed; compatibility repairs discovered by broader verification were committed separately.

1. **Task 1: Publish exact plan, spec, and wrapper-host contracts** — `b41328ea` (docs)
2. **Task 2: Update command-guide and build-cycle orchestration guidance** — `176fbc93` (docs)
3. **Compatibility repair: exact-acceptance wrapper closeout gate** — `070800e0` (test)
4. **Compatibility repair: shared lifecycle contract inventory and headings** — `e148cbb4` (docs)

## Files Created/Modified

- `cmd/contracts/spec.md` — Public Specification command, replay, refusal, projection, and authority contract.
- `cmd/contracts/plan.md` — Public preset, stage, decision, iteration, candidate, recommendation, replay, and exact-acceptance contract.
- `.aether/docs/wrapper-host-contract.md` — Allowed presentation/interaction duties and forbidden host authority or state writes.
- `cmd/planning_contract_docs_200_test.go` — Exact Phase 200 cross-document contract ratchet.
- `cmd/lifecycle_wrapper_contract_test.go` — Stage-shaped host keys and preserved wrapper safety invariants.
- `cmd/command_guide.go` — Runtime-native Specification entry plus staged Plan orchestration instructions.
- `cmd/command_guide_test.go` — Preset, stage, card, candidate, replay, acceptance, and retired-guidance coverage.
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — Canonical direct-Codex Plan flow.
- `cmd/command_call_audit_test.go` — Restored `spec` severity registration.
- `cmd/visual_wrapper_contract_test.go` — Acceptance-receipt Plan closeout gate while retaining other workflow ceremonies.
- `cmd/contract_validate_test.go` — Specification lifecycle contract inventory coverage.

## Decisions Made

- Worker authority always comes from `result.plan_manifest.stage_manifest`; a Scout receipt is required before Route-Setter authority, and a completed iteration card is required before any later Scout.
- Route-Setter proposes five readiness assessments, but Go validates evidence and derives overall readiness, weakest gap, semantic delta, materiality, stop policy, receipts, and next actions.
- Routine read-only phase research is automatic inside the selected preset. There is no owner-facing depth ceremony or separate phase-research approval path.
- A stopped candidate remains `NOT ACTIVE`; Codex executes the complete runtime-issued `acceptance_command` verbatim only after explicit owner confirmation and exposes build/run only after `acceptance_receipt` succeeds.
- Exact retries are idempotent. Stale or divergent requests never substitute current IDs into old packets and never mutate the active plan.

## Verification

- `rtk go test ./cmd -run 'TestPlanningContractDocuments200|TestLifecycle.*Wrapper|TestCommandGuide.*Plan|Test.*BuildCycleSkill' -count=1` — 70 tests passed.
- Exact Task 1 command — 182 tests passed.
- `rtk go test ./cmd -run 'TestCommandGuide.*Plan|Test.*BuildCycleSkill' -count=1` — 3 tests passed.
- Command-guide, severity, lifecycle-inventory, and contract-structure checks — 6 tests passed.
- `rtk go run ./cmd/aether source-check` — passed; 16 canonical surfaces, 5 retired mirrors, and 126 generated wrappers checked.
- `git diff --check` — passed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Registered the restored Specification command in shared Codex audits**

- **Found during:** Task 2, from the Plan 21 named-gate handoff.
- **Issue:** The existing public `spec` command was absent from the command-guide catalogue and command-call severity registry, leaving the restored lifecycle surface unclassified.
- **Fix:** Added a runtime-native Codex guide with exact authority/drift boundaries, dedicated guide/YAML coverage, and an explicit enrichment-severity registration.
- **Files modified:** `cmd/command_guide.go`, `cmd/command_guide_test.go`, `cmd/command_call_audit_test.go`.
- **Commit:** `176fbc93`

**2. [Rule 1 - Bug] Replaced the stale immediate Plan closeout assertion with exact acceptance**

- **Found during:** Overall Plan verification after Task 2.
- **Issue:** Two legacy visual-wrapper tests required a Plan closeout ceremony immediately after stage finalization, contradicting the Phase 200 inactive-candidate and exact-acceptance boundary.
- **Fix:** Kept visual closeout requirements for every other workflow, while requiring Plan finalization -> candidate review -> returned acceptance command -> acceptance receipt in both Claude and OpenCode wrappers.
- **Files modified:** `cmd/visual_wrapper_contract_test.go`.
- **Commit:** `070800e0`

**3. [Rule 3 - Blocking Issue] Registered the new contract in the shared lifecycle document gate**

- **Found during:** Extra full command-package verification.
- **Issue:** The legacy inventory rejected `spec.md` as a nineteenth contract and the rewritten Plan contract did not retain the repository's shared title/date/input/output/state headings.
- **Fix:** Added Specification to the exact contract inventory and made both new Phase 200 contracts conform to the shared structural contract without weakening their detailed authority model.
- **Files modified:** `cmd/contract_validate_test.go`, `cmd/contracts/plan.md`, `cmd/contracts/spec.md`.
- **Commit:** `e148cbb4`

## Issues Encountered

- An extra `go test ./cmd -count=1` sweep reported 4,296 passing, 30 failing, and 5 skipped tests on the in-progress Phase 200 branch. The Plan 22-owned contract inventory/structure and visual-wrapper failures were fixed and their focused gates now pass. Remaining reported failures are pre-existing runtime/preset, visual/golden, lifecycle-card, schema, Phase 199 vocabulary, next-action, and orchestrator-boundary migrations; they are recorded in `deferred-items.md` for Plan 200-23 and their existing owners.
- The repository commit hook printed its existing package-validation warning but accepted every atomic commit; no task file was left staged or untracked.
- `state.update-progress` found no legacy body-level `Progress:` field, but `state.advance-plan` updated the authoritative frontmatter to 58 completed plans and moved Phase 200 to Plan 23. `requirements.mark-complete` did not parse the milestone's bold-ID checklist syntax; all eight requirement rows were already checked complete, so no manual mutation was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-23 can assemble the executable end-to-end corpus against one documented Specification -> staged planning -> candidate -> exact acceptance journey.
- Runtime callers and all three platform surfaces now use the same state ownership, replay, refusal, and recovery language.
- No Plan 200-22 blocker remains; the unrelated full-suite migrations are explicitly deferred rather than hidden.

## Self-Check: PASSED

- All 11 implementation/documentation files and this summary exist.
- Task and repair commits `b41328ea`, `176fbc93`, `070800e0`, and `e148cbb4` are present in repository history.
