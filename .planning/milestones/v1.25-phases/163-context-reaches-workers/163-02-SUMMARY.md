---
phase: 163-context-reaches-workers
plan: 02
subsystem: colony-context
tags: [charter, colony-prime, gate, governance, prompt-integrity, go]

# Dependency graph
requires:
  - phase: 160-fail-loudly
    provides: shared gate wiring/classification conventions and two-check producer pattern
provides:
  - Charter colonyPrimeSection (governance + constraints) reaching every worker prompt
  - charter_compliance / charter_compliance_executed gate producer and live wiring into continue
  - Named colony-prime budget constants (colonyPrimeBudgetChars/colonyPrimeCompactBudgetChars)
affects: [164-research-feeds-planning, 165-core-lifecycle-commands, 169-charter-and-standards]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "colonyPrimeSection template applied to a 15th case (charter) -- content builder + protectedSectionPolicy + relevanceScore, zero new plumbing"
    - "Two-check gate producer (findings vs executed) applied to a second gate beyond anti_pattern -- charter_compliance / charter_compliance_executed"

key-files:
  created:
    - cmd/colony_prime_charter_test.go
    - cmd/charter_gate_test.go
  modified:
    - cmd/colony_prime_context.go
    - cmd/context_weighting.go
    - cmd/gate.go
    - cmd/codex_continue.go
    - cmd/gate_test.go
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "Charter loads from the global store inside checkCharterComplianceGate rather than adding a parameter to runCodexContinueGates -- avoids touching ~15 existing call sites across cmd/codex_continue_finalize.go, cmd/codex_continue_plan.go, and multiple _test.go files outside the plan's file list, matching how checkAntiPatternGate already reads store directly"
  - "Registered charter_compliance/charter_compliance_executed in gateClassifications, gateRecoveryTemplates, and gateAutoResolveThresholds (not just alwaysRunGates) -- gate.go's own checkAntiPatternGate doc comment names this exact three-map registration as the producer-without-a-caller failure mode this project keeps repeating; an unregistered gate falls through to unclassified auto-recovery handling instead of hard-escalating on charter_compliance_executed failure"
  - "Only Linting/Testing/Formatting governance categories are mechanically gated; CI/Build categories are intentionally excluded (CI runs outside colony observation, build tools are not conduct rules) -- everything else in the charter reaches workers as prose hard rules via the colony-prime section instead"

requirements-completed: [CONTEXT-06]

# Metrics
duration: 27min
completed: 2026-07-29
---

# Phase 163 Plan 02: Charter Reaches Workers Summary

**Colony charter governance now flows to every worker as a protected, integrity-assessed colony-prime section, and a two-check compliance gate blocks continue when a declared governance tool goes unexercised.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-07-29T18:31:34+02:00
- **Completed:** 2026-07-29T18:57:59+02:00
- **Tasks:** 3 (plus 2 post-hoc fix commits found by the full test suite)
- **Files modified:** 8

## Accomplishments
- `state.Charter.Governance`/`.Constraints` now render as a `charter` colonyPrimeSection, framed as binding rules, and pass through the same `colony.AssessPromptSource` / `colony.RankContextCandidates` pipeline as every other section
- Charter is registered in `protectedSectionPolicy`, so it survives budget trimming even when other sections are cut under the compact 4000-char budget
- `checkCharterComplianceGate` enforces the narrow, mechanically-checkable slice of the charter (declared + config file present + no verification step exercised it → violation) and is wired live into `runCodexContinueGates`
- Bare `8000`/`4000` budget literals replaced with named constants (`colonyPrimeBudgetChars`, `colonyPrimeCompactBudgetChars`)

## Task Commits

Each task was committed atomically:

1. **Task 1: Charter reaches every worker as a protected colony-prime section** - `8d3f0d50` (feat)
2. **Task 2: A charter compliance gate that distinguishes violated from never-ran** - `a1468553` (feat)
3. **Task 3: Wire the charter gate into continue so it actually runs** - `f0d56704` (feat)
4. **Fix: hardcoded soft_block threshold count** - `3ac43ca0` (fix, found by full suite)
5. **Fix: regression golden snapshot gate counts** - `66241220` (fix, found by full suite)

_All three tasks were TDD: tests written first and confirmed failing (RED) before implementation (GREEN)._

## Files Created/Modified
- `cmd/colony_prime_context.go` - charter colonyPrimeSection, budget constants, `charterNoGovernanceFallback`
- `cmd/context_weighting.go` - `protectedSectionPolicy("charter")`, `sectionRelevanceScore("charter")`
- `cmd/gate.go` - `checkCharterComplianceGate`, `governanceInvocationTokens`, `governanceConfigFilesForLabel`, `parseCharterGovernanceLabels`, and registration in `alwaysRunGates`/`gateClassifications`/`gateRecoveryTemplates`/`gateAutoResolveThresholds`
- `cmd/codex_continue.go` - live call site inside `runCodexContinueGates`, immediately after the anti-pattern block
- `cmd/colony_prime_charter_test.go` (new) - 4 tests for Task 1 behaviors
- `cmd/charter_gate_test.go` (new) - 5 subtests for Task 2 behaviors + 3 wired-path subtests for Task 3 + always-run pin
- `cmd/gate_test.go` - updated `TestEffectiveGateAutoResolveThresholds_Defaults`/`_NoConfig` expected count (6→7)
- `cmd/testdata/regression_snapshot.json` - updated `gate_classifications` golden counts (total 14→16, hard_block 6→7, soft_block 6→7)

## Decisions Made
- `checkCharterComplianceGate(steps []codexVerificationStep) (gateCheck, gateCheck)` loads `COLONY_STATE.json` from the global `store` internally rather than taking the charter as a parameter of `runCodexContinueGates`. Changing that function's signature would have touched ~15 call sites in files outside this plan's scope (`cmd/codex_continue_finalize.go`, `cmd/codex_continue_plan.go`, and several `_test.go` files). This mirrors how `checkAntiPatternGate` already reads `store` directly rather than receiving colony state as an argument.
- Full registration of the new gate in `gateClassifications` (soft_block for `charter_compliance`, hard_block for `charter_compliance_executed`), `gateRecoveryTemplates`, and `gateAutoResolveThresholds`, not just `alwaysRunGates`. `checkAntiPatternGate`'s own doc comment in `cmd/gate.go` names exactly this three-map registration as the fix for "a fully specified gate name that no code anywhere produced" — the project's own recurring failure mode per CLAUDE.md's Definition of Done. Verified necessary by `TestGateClassifications_CoversAllNamedGates`/`_CoversAllAlwaysRunGates` and `TestGateRecoveryTemplates_HasAllGateNames`, which would have failed without it.
- Governance parsing only inspects the "Linting:"/"Testing:"/"Formatting:" categories of `Charter.Governance` (matching `generateCharter`'s format exactly) and ignores "CI:"/"Build:" — recorded in a code comment per the plan's instruction, so a future reader knows the boundary was decided, not forgotten.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated hardcoded soft_block threshold count assertions**
- **Found during:** Full `go test ./cmd/...` run after Task 3
- **Issue:** `TestEffectiveGateAutoResolveThresholds_Defaults` and `_NoConfig` asserted a fixed count of 6 default auto-resolve thresholds; registering `charter_compliance` as the 7th soft_block gate made the assertion fail
- **Fix:** Updated both tests' expected count from 6 to 7, with a comment explaining the new gate
- **Files modified:** cmd/gate_test.go
- **Verification:** `go test ./cmd -run TestEffectiveGateAutoResolveThresholds -count=1 -v` passes
- **Committed in:** 3ac43ca0

**2. [Rule 1 - Bug] Refreshed regression golden snapshot gate counts**
- **Found during:** Full `go test ./cmd/...` run after fix #1
- **Issue:** `TestRegressionSnapshot` pins `gate_classifications.total_gates`/`hard_block`/`soft_block` against `cmd/testdata/regression_snapshot.json`; adding `charter_compliance` (soft_block) and `charter_compliance_executed` (hard_block) made counts drift from 14/6/6 to 16/7/7
- **Fix:** Updated the three counts in the golden JSON file directly (not via `-update-golden`, to avoid picking up unrelated incidental drift from the full-suite run)
- **Files modified:** cmd/testdata/regression_snapshot.json
- **Verification:** `go test ./cmd -run TestRegressionSnapshot -count=1 -v` passes; full `go test ./cmd/...` run green afterward
- **Committed in:** 66241220

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs — stale test expectations, not implementation defects)
**Impact on plan:** Both fixes are direct, mechanical consequences of the gate registration the plan itself required (Task 2's instruction to copy `checkAntiPatternGate`'s registration shape). No scope creep; neither fix touches production behavior.

## Deliberate-Regression Proof (Task 3 requirement)

Per Task 3's instruction, the `checkCharterComplianceGate` call site inside `runCodexContinueGates` was deliberately removed, the wired test rerun to observe failure, then restored and rerun green.

**Before removal:** `TestContinueCharterComplianceGateWiredIntoPipeline` (3 subtests) — all PASS.

**Call site removed** (`cmd/codex_continue.go`, replaced the wiring block with a single comment):
```go
// DELIBERATE-REGRESSION-PROOF: call site removed to prove the wired test fails.
```

**Observed failure** (`go test ./cmd -run TestContinueCharterComplianceGateWiredIntoPipeline -count=1 -v`):
```
=== RUN   TestContinueCharterComplianceGateWiredIntoPipeline/BlocksOnUnexercisedTool
    charter_gate_test.go:256: charter_compliance check not present in gate report: [...]
--- FAIL: TestContinueCharterComplianceGateWiredIntoPipeline/BlocksOnUnexercisedTool (0.04s)
=== RUN   TestContinueCharterComplianceGateWiredIntoPipeline/ExecutedCheckAlwaysPresentEvenWhenFindingsSkipped
    charter_gate_test.go:311: charter_compliance check missing: [...]
--- FAIL: TestContinueCharterComplianceGateWiredIntoPipeline/ExecutedCheckAlwaysPresentEvenWhenFindingsSkipped (0.02s)
=== RUN   TestContinueCharterComplianceGateWiredIntoPipeline/PipelineUsesRealProducer
    charter_gate_test.go:362: runCodexContinueGates report is missing the "charter_compliance" check — the charter gate call site has been removed from the continue pipeline
--- FAIL: TestContinueCharterComplianceGateWiredIntoPipeline/PipelineUsesRealProducer (0.00s)
--- FAIL: TestContinueCharterComplianceGateWiredIntoPipeline (0.05s)
FAIL
```

**Restored and rerun green:**
```
--- PASS: TestContinueCharterComplianceGateWiredIntoPipeline (0.01s)
    --- PASS: TestContinueCharterComplianceGateWiredIntoPipeline/BlocksOnUnexercisedTool (0.01s)
    --- PASS: TestContinueCharterComplianceGateWiredIntoPipeline/ExecutedCheckAlwaysPresentEvenWhenFindingsSkipped (0.00s)
    --- PASS: TestContinueCharterComplianceGateWiredIntoPipeline/PipelineUsesRealProducer (0.01s)
PASS
```

This proves the wiring test genuinely exercises the live call site, not a stub that happens to match `Name`/`Passed`.

## TDD Gate Compliance

All three tasks had `tdd="true"`. Verified RED before GREEN for each:
- Task 1: `cmd/colony_prime_charter_test.go` — 3 of 4 new tests failed before the charter section existed (the 4th, empty-charter, passed trivially since it asserts absence); all 4 pass after implementation.
- Task 2: `cmd/charter_gate_test.go` — compile failure (`undefined: checkCharterComplianceGate`) confirmed before implementation; all subtests pass after.
- Task 3: wired-path subtests passed immediately after the Task 3 wiring commit (implementation-first for the wiring itself), and the deliberate-regression proof above independently confirms the wired tests actually depend on the call site.

## Verification

- `go build ./cmd/aether` — exit 0
- `go vet ./...` — exit 0
- `go test ./cmd -run 'TestColonyPrime|TestContinueCharterComplianceGate' -count=1` — pass
- `go test ./cmd/... -count=1` — pass (full package, 290s)
- `grep -rn 'Charter' cmd/codex_build.go` — no matches (charter never bypasses the colonyPrimeSection route)
- `grep -c 'budget := 8000\|budget = 4000' cmd/colony_prime_context.go` — 0

## Known Stubs

None. No hardcoded empty values or placeholder text were introduced; the charter section and gate both operate on real `state.Charter` data with graceful (not stubbed) empty/absent handling.

## Threat Flags

None beyond what the plan's own threat model already covers (T-163-03, T-163-07, T-163-08 — all addressed as designed: charter routes only through `colonyPrimeSection`, the compliance gate's triple-condition prevents false positives from charter drift, and `charter_compliance_executed` is in `alwaysRunGates` so non-execution can't be masked as a pass).

## Issues Encountered
- The full `go test ./cmd/...` suite took several minutes per run due to CPU contention from concurrent parallel-wave test runs (other worktree agents in this sandbox running their own `go test` invocations simultaneously). Not a defect in this plan's work — confirmed by rerunning the suite alone after each fix and getting a clean, reproducible signal.

## Next Phase Readiness
- CONTEXT-06 is now fully satisfied end-to-end: charter reaches workers (Task 1) and the mechanically-checkable part of it is verified, not just trusted (Tasks 2-3), matching D-09.
- Phase 169 (Charter & Standards) can build further charter-facing UX on top of this plumbing without re-deriving the colony-prime section or gate patterns.

---
*Phase: 163-context-reaches-workers*
*Plan: 02*
*Completed: 2026-07-29*

## Self-Check: PASSED

All 9 claimed files verified present on disk; all 5 claimed commit hashes verified present in `git log --oneline --all`.
