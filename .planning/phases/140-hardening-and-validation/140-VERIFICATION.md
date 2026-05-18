---
phase: 140-hardening-and-validation
verified: 2026-05-18T18:30:00Z
status: gaps_found
score: 7/10 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Temp files and completion artifacts are cleaned up after normal completion and after graceful shutdown"
    status: failed
    reason: "Roadmap success criterion 4 -- no test or implementation verifies temp file cleanup. No plan in Phase 140 addresses this. No later phase exists to defer to."
    artifacts: []
    missing:
      - "Integration test or assertion verifying temp files are cleaned up after build/plan/continue completion"
      - "Test or assertion verifying temp files are cleaned up after graceful shutdown"
  - truth: "An end-to-end test runs plan through build through continue with real workers (not --simulate)"
    status: failed
    reason: "Roadmap success criterion 1 says 'with real workers'. The plan-through-build-through-continue test (line 1713) uses --simulate and mocked dispatches. All host-integration tests use --simulate. This is the established test pattern (--simulate is required for test execution), but the roadmap criterion explicitly says 'with real workers'."
    artifacts:
      - path: ".aether/ts-host/test/host-integration.test.ts"
        issue: "All integration tests use --simulate flag and __setDispatchWorkers mock; no production-worker E2E test exists"
    missing:
      - "E2E test exercising the full pipeline without --simulate (or documentation explaining why --simulate is the only safe test mode)"
  - truth: "VAL-* requirement IDs are tracked in REQUIREMENTS.md"
    status: failed
    reason: "Plans 01-03 declare requirements VAL-01 through VAL-08, but these IDs do not exist in .planning/REQUIREMENTS.md (the v1.21 requirements file). The VAL-* IDs are defined only in 140-RESEARCH.md and plan frontmatter. REQUIREMENTS.md has no awareness of Phase 140 validation requirements."
    artifacts:
      - path: ".planning/REQUIREMENTS.md"
        issue: "Does not contain VAL-01 through VAL-08 entries"
    missing:
      - "VAL-01 through VAL-08 entries in REQUIREMENTS.md with descriptions and traceability"
deferred: []
---

# Phase 140: Hardening and Validation Verification Report

**Phase Goal:** End-to-end integration tests prove the full production pipeline works, wrapper alignment is verified, and the milestone is audit-ready for ship.
**Verified:** 2026-05-18T18:30:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Every dispatched command wrapper contains ceremony invocation instructions for all 4 ceremony methods | VERIFIED | cross-platform-parity.test.ts line 179: iterates 6 dispatched commands x 4 ceremony commands x 2 platforms, all pass |
| 2 | Read-only command wrappers have ceremony steps from parity matrix present in wrapper content | VERIFIED | cross-platform-parity.test.ts line 219: verifies 11 read-only commands lack dispatched ceremony patterns |
| 3 | Claude and OpenCode wrappers for the same command have matching ceremony invocation patterns | VERIFIED | cross-platform-parity.test.ts line 197: extracts ceremony sets from both wrappers and asserts deepEqual |
| 4 | Parity matrix ceremony_steps align with wrapper content for all 18 commands | VERIFIED | classic-command-parity.test.ts line 178: 3 tests verify lifecycle completeness, spawn-plan/closeout invariant, and non-empty ceremony_steps for all 18 commands |
| 5 | Build pipeline exercises hive wisdom injection, worker spawning, and confidence iteration together in a single test | VERIFIED | host-integration.test.ts line 1574: single build invocation calls hive-read, initializes SpawnOrchestrator (budget=10), runs 2 iterations with feedback injection |
| 6 | Plan-through-build-through-continue pipeline runs without errors when mocked end-to-end | VERIFIED | host-integration.test.ts line 1713: runs all 3 dispatched runner commands sequentially, verifies plan-finalize, build-finalize, continue-finalize each called |
| 7 | Spawn children from iteration re-dispatch are budget-tracked and logged | VERIFIED | host-integration.test.ts line 1825: verifies totalBudget=5, consumedBudget=3 across 2 iterations |
| 8 | Hive wisdom appears in worker dispatches across iterations | VERIFIED | host-integration.test.ts line 1574: asserts hive-read called, stderr contains "Injecting hive wisdom" |
| 9 | Every v1.21 requirement (HOST, SPAWN, ITER, HIVE, SKILL) has at least one test file that verifies it | VERIFIED | milestone-audit.test.ts: coverage map has 31 entries, all verified; test passes |
| 10 | Full TS host test suite passes with zero failures | VERIFIED | 464 tests, 92 suites, 0 failures (31s runtime) |
| 11 | Full Go test suite passes with zero failures | VERIFIED | 18 packages ok, 0 failures |
| 12 | Temp files and completion artifacts are cleaned up after normal completion and after graceful shutdown | FAILED | No test or implementation verifies this. Roadmap SC 4 is unaddressed. |
| 13 | E2E test runs plan through build through continue with real workers (not --simulate) | FAILED | All integration tests use --simulate and mocked dispatches. The --simulate flag is required for test execution per host-integration.test.ts line 94. |
| 14 | VAL-* requirement IDs tracked in REQUIREMENTS.md | FAILED | VAL-01 through VAL-08 declared in plans but absent from .planning/REQUIREMENTS.md |

**Score:** 11/14 truths verified (7/10 must-haves when grouped by plan-level claims)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.aether/ts-host/test/cross-platform-parity.test.ts` | Wrapper ceremony alignment tests (D-01, D-02, D-03) | VERIFIED | 237 lines, 8 tests (5 existing + 3 new), all pass |
| `.aether/ts-host/test/classic-command-parity.test.ts` | Ceremony marker presence tests from parity matrix | VERIFIED | 267 lines, 8 tests (5 existing + 3 new), all pass |
| `.aether/ts-host/test/host-integration.test.ts` | Cross-phase integration tests combining hive + spawn + iteration | VERIFIED | 1943 lines, 38 tests (35 existing + 3 new), all pass |
| `.aether/ts-host/test/milestone-audit.test.ts` | Milestone audit test verifying all 31 requirements have test coverage | VERIFIED | 183 lines, 4 tests, all pass |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| cross-platform-parity.test.ts | .claude/commands/ant/*.md | readFileSync reads wrapper content | WIRED | `readWrapper(".claude", command)` at line 181 reads actual wrapper files |
| cross-platform-parity.test.ts | .opencode/commands/ant/*.md | readFileSync reads wrapper content | WIRED | `readWrapper(".opencode", command)` at line 182 reads actual wrapper files |
| classic-command-parity.test.ts | .aether/commands/classic-command-parity.json | loadMatrix reads parity data | WIRED | `loadMatrix()` at line 46 reads JSON via readFileSync |
| host-integration.test.ts | .aether/ts-host/src/host.ts | runDispatchedBuildCommand, runDispatchedPlanCommand, runDispatchedContinueCommand | WIRED | Imports at file top; functions called in all 3 cross-phase tests |
| host-integration.test.ts | .aether/ts-host/src/spawn-orchestrator.ts | SpawnOrchestrator via dispatch opts | WIRED | `capturedDispatchOpts[0].spawnOrchestrator` verified at line 1681 |
| milestone-audit.test.ts | .planning/REQUIREMENTS.md | REQUIREMENTS_ID_PATTERN matching | WIRED | Reads REQUIREMENTS.md at line 133, extracts IDs, cross-references with coverage map |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| cross-platform-parity.test.ts | Wrapper file content | readFileSync of .claude/.opencode wrapper .md files | VERIFIED | Reads actual committed wrapper files -- real data |
| classic-command-parity.test.ts | ClassicCommandRecord.ceremony_steps | classic-command-parity.json | VERIFIED | Reads actual committed parity matrix -- real data |
| host-integration.test.ts | Mock goCalls, capturedDispatchOpts | Mock handlers returning structured data | N/A | Test uses mocks -- expected for integration tests |
| milestone-audit.test.ts | REQUIREMENT_COVERAGE_MAP | Hardcoded constant (31 entries) | VERIFIED | Static map verified bidirectionally against REQUIREMENTS.md content |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Milestone audit tests pass | `npx tsx --test test/milestone-audit.test.ts` | 4/4 pass, 0 fail | PASS |
| Cross-platform parity tests pass | `npx tsx --test test/cross-platform-parity.test.ts` | 8/8 pass, 0 fail | PASS |
| Classic command parity tests pass | `npx tsx --test test/classic-command-parity.test.ts` | 8/8 pass, 0 fail | PASS |
| Host integration tests pass | `npx tsx --test test/host-integration.test.ts` | 38/38 pass, 0 fail | PASS |
| Full TS suite passes | `npx tsx --test test/*.test.ts` | 464/464 pass, 0 fail | PASS |
| Full Go suite passes | `go test ./...` | 18/18 packages ok, 0 fail | PASS |

### Probe Execution

No probes declared for this phase. Phase is testing-only (no migration or CLI tooling changes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| VAL-01 | 140-01 | Ceremony steps match between wrapper and host for dispatched commands | SATISFIED | classic-command-parity.test.ts ceremony marker presence tests |
| VAL-02 | 140-01 | All commands have ceremony_steps in parity matrix | SATISFIED | classic-command-parity.test.ts "all 18 commands have non-empty ceremony steps" |
| VAL-03 | 140-02 | Build pipeline exercises hive + spawn + iteration together | SATISFIED | host-integration.test.ts "build pipeline exercises hive, spawn, and iteration together" |
| VAL-04 | 140-01 | Wrapper markdown contains ceremony invocation commands | SATISFIED | cross-platform-parity.test.ts "dispatched command wrappers contain all 4 ceremony invocations" |
| VAL-05 | 140-01 | Claude and OpenCode wrappers have matching ceremony sections | SATISFIED | cross-platform-parity.test.ts "Claude and OpenCode wrappers have matching ceremony invocation patterns" |
| VAL-06 | 140-03 | Milestone audit: all 31 requirements have passing tests | SATISFIED | milestone-audit.test.ts "all 31 v1.21 requirements have test coverage" |
| VAL-07 | 140-03 | Full TS regression suite passes | SATISFIED | 464 tests, 0 failures |
| VAL-08 | 140-03 | Full Go regression suite passes | SATISFIED | 18 packages, 0 failures |

**ORPHANED:** VAL-01 through VAL-08 are declared in plan frontmatter but do NOT appear in `.planning/REQUIREMENTS.md`. The canonical requirements file has no entries for Phase 140 validation requirements.

### Anti-Patterns Found

No anti-patterns detected. All 4 test files are clean: no TODO, FIXME, XXX, TBD, PLACEHOLDER, or HACK markers. No empty implementations. No stub returns.

### Human Verification Required

None. All tests are automated and all pass. The gaps identified are structural (missing test coverage for cleanup, missing REQUIREMENTS.md entries) that can be verified programmatically.

### Gaps Summary

3 gaps found blocking full goal achievement:

1. **Temp file cleanup untested (BLOCKER)** -- Roadmap success criterion 4 ("Temp files and completion artifacts are cleaned up after normal completion and after graceful shutdown") has no test coverage and was not addressed by any of the 3 plans. This is the final phase of the milestone -- no later phase to defer to.

2. **No production-worker E2E test (BLOCKER)** -- Roadmap success criterion 1 says the E2E test should run "with real workers." All integration tests use --simulate and mocked dispatches. While this is the established test pattern (production execution without --simulate is explicitly rejected per test line 94), the roadmap criterion is not satisfied as written. Either an E2E test without --simulate is needed, or the roadmap criterion should be updated to reflect the actual testing architecture.

3. **VAL-* requirements orphaned from REQUIREMENTS.md (WARNING)** -- 8 requirement IDs (VAL-01 through VAL-08) are declared in plan frontmatter and completed in summaries but never added to the canonical `.planning/REQUIREMENTS.md`. This breaks traceability.

The core validation work (ceremony alignment, cross-phase integration, milestone audit, full regression) is substantive and passing. The 464 TS tests and 18 Go packages all pass with zero failures. The gaps are in completeness relative to the roadmap contract.

---

_Verified: 2026-05-18T18:30:00Z_
_Verifier: Claude (gsd-verifier)_
