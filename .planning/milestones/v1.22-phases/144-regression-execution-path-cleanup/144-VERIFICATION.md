---
phase: 144-regression-execution-path-cleanup
verified: 2026-05-18T23:15:00Z
status: passed
score: 8/8 must-haves verified
overrides_applied: 0

human_verification: []

---

# Phase 144: Regression + Execution Path Cleanup Verification Report

**Phase Goal:** Prove everything works end-to-end. One declared execution path per workflow per platform. Clean up stragglers.
**Verified:** 2026-05-18T23:15:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | M4L fixture produces grounded plan with zero .venv references | VERIFIED | TestM4LRegression_VenvProducesGroundedPlan passes: survey output has no .venv/site-packages/__pycache__, source anchors contain no .venv paths, grounded tasks produce zero warnings, ungrounded tasks produce exactly 1 warning with correct PhaseID=2 |
| 2 | One declared execution path per workflow per platform is documented and verified | VERIFIED | TestExecutionPathAudit_OneConductorPerWorkflow passes all 5 subtests (build, plan, colonize, continue, seal). Each YAML has runtime.command with correct host invocation. Continue has both default and heavy-review paths. |
| 3 | Codex command-guide produces correct output without playbook loading | VERIFIED | TestCommandGuideBuildSmoke and TestCommandGuidePlanSmoke both pass: non-empty RunCommand, PreSteps >= 1, PostSteps >= 1 for both build and plan |
| 4 | Build ceremony output matches between Claude Code and OpenCode | VERIFIED | 8/8 cross-platform-parity TS tests pass. "Claude and OpenCode wrappers have matching ceremony invocation patterns" test confirms build ceremony alignment. |
| 5 | All 4 Go tests that failed after Phase 143 now pass | VERIFIED | TestCLIFlagAudit, TestWrapperSourcesUseTypeScriptHostManifestSpine, TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity, TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance all pass |
| 6 | build.yaml and plan.yaml contain required anchors without restoring orchestration blocks | VERIFIED | Both files have guardrail entries with all required anchor strings. grep -c "orchestration:" returns exactly 1 for each (codex_orchestration only). No "orchestration: |" block exists in either wrapper_additions. |
| 7 | CLI flag audit no longer false-positives on pending-decisions | VERIFIED | "pending-decisions": true added to skipSubcommands map in cli_flag_audit_test.go line 39. TestCLIFlagAudit passes with 130 unique subcommands found. |
| 8 | Cross-platform parity tests confirm build ceremony alignment (CLEAN-06) | VERIFIED | 8/8 tests pass including "Claude and OpenCode wrappers have matching ceremony invocation patterns" and "dispatched command wrappers contain all 4 ceremony invocations" |

**Score:** 8/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.aether/commands/build.yaml` | Anchors restored in guardrails, no orchestration block | VERIFIED | 6 guardrails including "TS host is the sole entry point", "build-finalize", "spawn-log", "spawn-complete", "visible live Task/subagent", "orchestrator_boundary_guidance" |
| `.aether/commands/plan.yaml` | Anchors restored in guardrails, no orchestration block | VERIFIED | 6 guardrails including same anchor strings plus "after_discuss_next", "aether discuss", "fresh" |
| `cmd/codex_colonize_test.go` | TestM4LRegression integration test | VERIFIED | Lines 1949-2010: 4-step pipeline test using createVenvNoiseFixture, surveyWorkspace, extractSourceAnchors, checkPlanGrounding |
| `cmd/command_guide_test.go` | Smoke tests + execution path audit | VERIFIED | TestCommandGuideBuildSmoke (L833), TestCommandGuidePlanSmoke (L849), TestExecutionPathAudit_OneConductorPerWorkflow (L878) with 5 subtests |
| `cmd/cli_flag_audit_test.go` | pending-decisions skip fix | VERIFIED | Line 39: skipSubcommands map entry with explanatory comment |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| TestM4LRegression (L1985) | checkPlanGrounding | direct function call | WIRED | Called at lines 1985 and 2003 with colony.Phase slices and anchor strings |
| TestCommandGuideSmoke tests | buildCommandGuide | direct function call | WIRED | buildCommandGuide imported from command_guide.go:69, called at lines 834 and 850 |
| TestExecutionPathAudit | YAML runtime.command | YAML parsing + strings.Contains | WIRED | Reads .aether/commands/*.yaml, parses runtime.command via yaml.Unmarshal, asserts content |

### Data-Flow Trace (Level 4)

N/A -- This phase produces tests that verify existing data flows. The tests themselves are the deliverable; they exercise existing pipelines (survey -> anchors -> grounding, command-guide generation, YAML metadata).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 4 regression Go tests pass | `go test ./cmd -run "TestCLIFlagAudit\|TestWrapperSources\|TestCodexLifecycle\|TestLifecycleWrapper" -count=1` | All PASS | PASS |
| M4L regression test passes | `go test ./cmd -run "TestM4LRegression_VenvProducesGroundedPlan" -count=1` | PASS | PASS |
| Codex smoke tests pass | `go test ./cmd -run "TestCommandGuideBuildSmoke\|TestCommandGuidePlanSmoke" -count=1` | All PASS | PASS |
| Execution path audit passes | `go test ./cmd -run "TestExecutionPathAudit_OneConductorPerWorkflow" -count=1` | 5/5 subtests PASS | PASS |
| Cross-platform parity tests pass | `npx tsx --test test/cross-platform-parity.test.ts` | 8/8 PASS | PASS |
| Full Go test suite passes | `go test ./... -count=1` | 18/18 packages PASS | PASS |
| No orchestration blocks in build.yaml | `grep "orchestration: |" .aether/commands/build.yaml` | exit code 1 (not found) | PASS |
| No orchestration blocks in plan.yaml | `grep "orchestration: |" .aether/commands/plan.yaml` | exit code 1 (not found) | PASS |
| Exactly 1 orchestration key in build.yaml | `grep -c "orchestration:" .aether/commands/build.yaml` | returns 1 | PASS |
| Exactly 1 orchestration key in plan.yaml | `grep -c "orchestration:" .aether/commands/plan.yaml` | returns 1 | PASS |

### Probe Execution

No probes declared in PLAN or SUMMARY for this phase.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CLEAN-01 | 144-02 | M4L regression test: .venv fixture produces grounded plan | SATISFIED | TestM4LRegression_VenvProducesGroundedPlan passes 4-step pipeline verification |
| CLEAN-02 | 144-02 | Execution path audit: one declared path per workflow per platform | SATISFIED | TestExecutionPathAudit_OneConductorPerWorkflow with 5 subtests |
| CLEAN-03 | 144-01 | Audit ceremony-coupled TS tests, fix broken tests | SATISFIED | All 4 failing Go tests now pass; cross-platform-parity tests pass |
| CLEAN-04 | 144-01 | YAML packaging-only verification | SATISFIED | grep confirms no orchestration: \| in wrapper_additions of build.yaml or plan.yaml |
| CLEAN-05 | 144-02 | Codex smoke test: command-guide produces correct behavior | SATISFIED | TestCommandGuideBuildSmoke and TestCommandGuidePlanSmoke pass |
| CLEAN-06 | 144-02 | Cross-platform parity: build ceremony matches | SATISFIED | 8/8 cross-platform-parity tests pass including ceremony alignment |

**Note:** REQUIREMENTS.md was deliberately removed from the repo (commit b8288edc). Requirements are defined in ROADMAP.md success criteria and PLAN frontmatter. The milestone-audit.test.ts references REQUIREMENTS.md and produces 2 failures -- these are pre-existing and unrelated to Phase 144.

### Anti-Patterns Found

No anti-patterns detected in any files modified by Phase 144. The TODO/FIXME/HACK matches in codex_colonize_test.go are test data and assertion strings (testing the pathogen identification feature), not code debt markers.

### Human Verification Required

None required. All truths are programmatically verifiable through test execution and file content inspection.

### Gaps Summary

No gaps found. All 6 requirements (CLEAN-01 through CLEAN-06) are satisfied. The phase goal -- proving end-to-end pipeline correctness and one execution path per workflow per platform -- is fully achieved through 7 new passing tests (M4L regression, 2 Codex smoke tests, execution path audit with 5 subtests) and 4 fixed regression tests.

---

_Verified: 2026-05-18T23:15:00Z_
_Verifier: Claude (gsd-verifier)_
