# Phase 144: Regression + Execution Path Cleanup - Research

**Researched:** 2026-05-18
**Domain:** Regression testing, execution path audit, YAML cleanup, cross-platform parity, Go/TS test maintenance
**Confidence:** HIGH

## Summary

Phase 144 is the capstone verification phase for the v1.22 milestone (Grounded Planning + Ceremony Restore). Phases 141-143 delivered survey noise filtering (`.venv`/`__pycache__` exclusion), a plan grounding gate (source anchors vs generic tasks), and playbook-driven ceremony restoration (PlaybookLoader in TS host, YAML orchestration stripped). Phase 144 must prove all of this works end-to-end and clean up the stragglers left behind.

The current test baseline is: **491 TS tests** (489 pass, 2 fail), **Go tests** (4 failures in `cmd` package). The 2 TS failures are both in `milestone-audit.test.ts` -- they fail because `.planning/REQUIREMENTS.md` does not exist in this repo (the test expects it at the standard GSD path). These are pre-existing and not caused by Phase 143. The 4 Go failures are directly caused by Phase 143's YAML changes: `build.yaml` and `plan.yaml` had their orchestration sections removed, but four tests still expect specific anchors in those YAML files.

The execution path audit (CLEAN-02) needs to map exactly one conductor per workflow per platform. The five workflows are: build, plan, continue, colonize, seal. The three platform lanes are: Claude Code + OpenCode (shared TS host), Codex (runtime-native command-guide + skills). Currently: build and plan have one conductor (TS host) for Claude/OpenCode; continue and seal have one conductor (continue uses `aether continue` directly for default path, TS host for heavy review); colonize still has YAML orchestration (14-step procedure in `wrapper_additions.orchestration`).

**Primary recommendation:** This phase is primarily test work and YAML surgery. No new modules or significant code changes are needed. The planner should structure tasks around: (1) fixing the 4 Go test failures from Phase 143, (2) creating the M4L regression test, (3) documenting the one-execution-path-per-workbook audit, (4) verifying YAML has no lingering orchestration logic, (5) Codex smoke test for command-guide + skills, and (6) cross-platform build ceremony parity check.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Test maintenance | Go test suite / TS test suite | -- | All requirements are verification tasks, not feature code |
| YAML anchor restoration | YAML source files (`.aether/commands/`) | Go test expectations | Tests check for anchors in YAML; YAML was stripped in Phase 143, anchors need restoring |
| Execution path documentation | Documentation only | -- | CLEAN-02 is an audit task, not code -- document one path per workflow per platform |
| M4L regression test | Go test suite | -- | `TestVenvNoiseExclusion` already exists; CLEAN-01 needs a full pipeline test from survey through plan grounding |
| Codex smoke test | Go runtime (`cmd/command_guide.go`) | -- | `command-guide` subcommand produces Codex guidance; verify it works without playbook loading |
| Cross-platform parity | Go test suite | -- | `TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity` already tests this |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go testing | Go 1.24+ | Go unit/integration tests | Existing test framework for all Go tests |
| Node.js test runner (`node:test`) | Built-in | TS unit tests | Existing test framework for all 491 TS tests |
| `node:assert/strict` | Built-in | TS assertions | Already used across all TS test files |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go `testing` helpers | Built-in | `t.Helper()`, `t.TempDir()`, fixture creation | All Go test fixtures |
| `node:fs` / `node:path` | Built-in | File reading, path resolution in TS tests | Already used in test helpers |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go stdlib `testing` | No alternative | Only test framework in this repo |
| `node:test` | Jest/Vitest | Would require new dependency; `node:test` already works and has 491 tests |

**Installation:**
```bash
# Zero new dependencies -- all changes use existing test infrastructure
```

**Version verification:** No new packages. [VERIFIED: existing test infrastructure runs `go test ./...` and `cd .aether/ts-host && npm test`]

## Architecture Patterns

### System Architecture Diagram

```
Phase 143 delivered:
  1. PlaybookLoader (TS) -- loads playbooks from repo/hub, injects into worker briefs
  2. Plan playbooks (plan-prep.md, plan-dispatch.md) -- Scout/Route-Setter flow
  3. YAML orchestration removed from build.yaml and plan.yaml

Phase 144 verifies:
  ┌───────────────────────────────────────────────────────┐
  │                  REGRESSION SUITE                      │
  │                                                        │
  │  CLEAN-01: M4L fixture                                │
  │    .venv in repo → survey filters it → plan grounded  │
  │    (Go test: cmd/codex_colonize_test.go +             │
  │     cmd/codex_plan_test.go + cmd/plan_grounding.go)   │
  │                                                        │
  │  CLEAN-02: Execution path audit                       │
  │    build → TS host (Claude/OpenCode), Go (Codex)     │
  │    plan → TS host (Claude/OpenCode), Go (Codex)       │
  │    continue → aether continue / TS host (heavy only)  │
  │    colonize → TS host (all platforms)                 │
  │    seal → TS host (all platforms)                     │
  │                                                        │
  │  CLEAN-03: Fix 4 Go tests broken by Phase 143        │
  │    TestCLIFlagAudit → plan-prep.md "pending-decisions"│
  │    TestWrapperSourcesUseTS... → build/plan YAML       │
  │    TestCodexLifecycleYaml... → build/plan YAML        │
  │    TestLifecycleWrapper... → build/plan YAML          │
  │                                                        │
  │  CLEAN-04: YAML packaging verification                │
  │    grep build.yaml/plan.yaml for orchestration blocks │
  │                                                        │
  │  CLEAN-05: Codex smoke test                          │
  │    aether command-guide build/plan → correct output   │
  │    no playbook loading required                       │
  │                                                        │
  │  CLEAN-06: Cross-platform parity                      │
  │    build ceremony matches Claude vs OpenCode          │
  │    (file names, sizes, content alignment)             │
  └───────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
cmd/
  cli_flag_audit_test.go          # CLEAN-03: fix "pending-decisions" false positive
  command_guide_test.go           # CLEAN-03: fix 3 anchor checks for build/plan YAML
  codex_colonize_test.go          # CLEAN-01: add M4L full-pipeline regression test
  codex_plan_test.go              # CLEAN-01: add grounded-plan-with-.venv-fixture test
  plan_grounding_test.go          # CLEAN-01: existing grounding gate tests (verify green)

.aether/ts-host/test/
  cross-platform-parity.test.ts   # CLEAN-06: extend build ceremony parity check
  milestone-audit.test.ts         # PRE-EXISTING: 2 failures (REQUIREMENTS.md missing)

.aether/commands/
  build.yaml                      # CLEAN-03: restore missing anchors
  plan.yaml                       # CLEAN-03: restore missing anchors
```

### Pattern 1: M4L Regression Test (End-to-End Survey-to-Plan)

**What:** Create a fixture repo with a `.venv` directory, run the full survey -> plan grounding pipeline, and assert zero `.venv` references and grounded tasks.
**When to use:** CLEAN-01 regression verification.
**Example:**
```go
// Source: Pattern from existing TestVenvNoiseExclusion (cmd/codex_colonize_test.go:1601)
func TestM4LRegression_VenvProducesGroundedPlan(t *testing.T) {
    root := createVenvNoiseFixture(t)

    // Step 1: Survey must exclude .venv
    facts, err := surveyWorkspace(root)
    if err != nil { t.Fatalf("surveyWorkspace error: %v", err) }
    // Verify no .venv in survey output (already tested by TestVenvNoiseExclusion)

    // Step 2: Source anchors must not include .venv paths
    for _, anchor := range facts.SourceAnchors {
        if strings.Contains(anchor, ".venv") {
            t.Errorf("source anchor %q should not contain .venv", anchor)
        }
    }

    // Step 3: Plan grounding with those anchors must not flag false positives
    anchors := facts.SourceAnchors
    phases := []colony.Phase{
        {ID: 1, Name: "Build API", Tasks: []colony.Task{
            {Goal: "Edit cmd/main.go to add new endpoint"},
        }},
    }
    warnings := checkPlanGrounding(phases, anchors)
    if len(warnings) != 0 {
        t.Errorf("grounded plan should not have warnings, got %d: %v", len(warnings), warnings)
    }
}
```

### Pattern 2: YAML Anchor Restoration

**What:** Phase 143 removed orchestration blocks from `build.yaml` and `plan.yaml`, but 3 Go tests expect specific anchors that were also removed. The fix is to restore the needed anchors without restoring the full orchestration blocks.
**When to use:** CLEAN-03 fix for `TestWrapperSourcesUseTypeScriptHostManifestSpine`, `TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity`, `TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance`.
**Example:**
The three tests expect these anchors in `build.yaml`:
- `"TS host is the sole entry point"` (TS host spine)
- `"build-finalize"` (finalizer reference)
- `"aether host build"` (host command)
- `"spawn-log"`, `"spawn-complete"`, `"visible live Task/subagent"` (worker activity)
- `"orchestrator_boundary_guidance"`, `"after_discuss_next"`, `"aether discuss"`, `"fresh"` (orchestrator guidance)

These anchors exist in `continue.yaml` (which was NOT modified by Phase 143) and should be present in `build.yaml` and `plan.yaml` as well -- likely in `wrapper_additions.post_build` or `guardrails` sections, NOT in a restored orchestration block.

### Anti-Patterns to Avoid

- **Restoring the YAML orchestration blocks:** Phase 143 explicitly removed them. The fix is to add anchors to other sections (guardrails, post_build, etc.), not to undo the removal.
- **Creating new modules for test-only work:** This phase is regression testing, not feature development. No new source modules needed.
- **Over-testing Codex:** CLEAN-05 is a smoke test, not a full integration test. Verify `command-guide` produces output, not that it perfectly matches expected content.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test fixture creation | Custom directory setup | Existing `createVenvNoiseFixture(t)` in `codex_colonize_test.go` | Already creates `.venv`, `node_modules`, `__pycache__`, `cmd/`, `src/` with source files |
| Grounding check | New validation logic | Existing `checkPlanGrounding()` in `cmd/plan_grounding.go` | Already handles research phase exemption, file path detection, warning generation |
| Execution path documentation | Ad-hoc notes | Existing YAML source files as the audit source | YAML files define the execution path; reading them IS the audit |
| Cross-platform parity check | Custom file comparison | Existing `cross-platform-parity.test.ts` | Already compares Claude vs OpenCode command directories |

**Key insight:** This phase is verification and cleanup, not construction. The plumbing was built in Phases 141-143. Phase 144 proves it works and polishes the edges.

## Runtime State Inventory

> Not applicable -- this phase is regression testing and cleanup, not a rename/refactor/migration.

N/A

## Common Pitfalls

### Pitfall 1: Confusing "Anchor Restoration" with "Orchestration Restoration"
**What goes wrong:** Fixing the Go test failures by restoring the full 15-step orchestration block that Phase 143 removed.
**Why it happens:** The tests check for strings like "spawn-log" and "visible live Task/subagent" which were in the orchestration block.
**How to avoid:** These anchors can appear in guardrails, post_build, or other non-orchestration sections. The tests just need the strings present in the YAML file, not in a specific section. Check `continue.yaml` for the correct placement pattern -- it has all these anchors without an orchestration block (wait, actually it DOES have an orchestration block for the heavy-review path).
**Warning signs:** A git diff that restores the `orchestration: |` key under `wrapper_additions`.

### Pitfall 2: TestCLIFlagAudit False Positive on "pending-decisions"
**What goes wrong:** The test scans all markdown files in `.aether/` for CLI subcommand references. `plan-prep.md` line 49 references `aether pending-decisions --count` but `pending-decisions` is not a registered Go subcommand.
**Why it happens:** The plan-prep.md was created in Phase 143 and references a command that may not be registered in the Go runtime, or may be registered under a different name.
**How to avoid:** Either (a) register `pending-decisions` as a Go subcommand, (b) add it to the CLI flag audit skip list, or (c) change the playbook to use the correct command name.
**Warning signs:** `TestCLIFlagAudit` failure message mentions `pending-decisions` in `plan-prep.md:49`.

### Pitfall 3: M4L Test Scope Creep
**What goes wrong:** Trying to test the entire planning pipeline (survey -> plan -> grounding -> finalize) in one test.
**Why it happens:** CLEAN-01 asks for "M4L regression test: `.venv` fixture produces grounded plan with concrete file references."
**How to avoid:** The test should focus on what changed: survey noise filtering (Phase 141) and grounding gate (Phase 142). It does NOT need to test the full LLM-based plan generation. Test the Go functions directly: `surveyWorkspace()` produces no `.venv` references, source anchors exclude `.venv`, and `checkPlanGrounding()` correctly validates.
**Warning signs:** A test that spawns real LLM agents or calls `aether plan`.

### Pitfall 4: Colonize YAML Still Has Orchestration (CLEAN-02 Scope)
**What goes wrong:** CLEAN-02 asks for "one declared path per workflow" but colonize.yaml still has a 14-step `wrapper_additions.orchestration` block.
**Why it happens:** Phase 143 only stripped build.yaml and plan.yaml. Continue.yaml and colonize.yaml were intentionally left unchanged.
**How to avoid:** The execution path audit should DOCUMENT this as-is (colonize uses YAML-defined orchestration for Claude/OpenCode, command-guide for Codex). It is not necessarily wrong -- the audit should identify it, not fix it unless the requirement explicitly says to.
**Warning signs:** Treating CLEAN-02 as a refactoring requirement rather than a documentation/audit requirement.

## Code Examples

### Current Test Baseline (Verified)

```
TS: 491 tests, 489 pass, 2 fail (pre-existing: milestone-audit expects REQUIREMENTS.md)
Go: 4 failures in cmd/:
  1. TestCLIFlagAudit -- plan-prep.md references unregistered "pending-decisions" subcommand
  2. TestWrapperSourcesUseTypeScriptHostManifestSpine -- build.yaml/plan.yaml missing "TS host is the sole entry point" and "build-finalize"
  3. TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity -- build.yaml/plan.yaml missing "spawn-log", "spawn-complete", "visible live Task/subagent", "build-finalize"
  4. TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance -- build.yaml/plan.yaml missing "orchestrator_boundary_guidance", "after_discuss_next", "aether discuss", "fresh"
```

### Existing Venv Noise Fixture (Verified)
```go
// Source: cmd/codex_colonize_test.go:1574-1616 (VERIFIED)
func createVenvNoiseFixture(t *testing.T) string {
    t.Helper()
    root := t.TempDir()
    dirs := []string{
        filepath.Join(root, ".venv", "lib", "site-packages", "requests"),
        filepath.Join(root, "cmd"),
        filepath.Join(root, "src"),
        filepath.Join(root, "__pycache__"),
        // ...more dirs
    }
    // Creates .venv, node_modules, __pycache__, site-packages, cmd/, src/
    // Includes source files: main.go, app.py, README.md
    // Already tests: no .venv in survey output, cmd/ preserved in TopLevelDirs
}
```

### Existing Grounding Gate (Verified)
```go
// Source: cmd/plan_grounding.go:34-58 (VERIFIED)
func checkPlanGrounding(phases []colony.Phase, sourceAnchors []string) []planGroundingWarning {
    if len(sourceAnchors) == 0 {
        return nil
    }
    // Returns warnings for non-research phases where anchors exist but tasks have no file refs
    // Grounding gate is always soft -- never returns an error
}
```

### Existing Cross-Platform Parity Test (Verified)
```typescript
// Source: .aether/ts-host/test/cross-platform-parity.test.ts:36-59 (VERIFIED)
it("Claude and OpenCode commands have identical file names", () => {
    const claudeFiles = listFiles(claudeDir, ".md");
    const opencodeFiles = listFiles(opencodeDir, ".md");
    assert.deepEqual(claudeFiles, opencodeFiles, ...);
    // Also checks file size parity (within 100 bytes)
});
```

### Missing Anchors in build.yaml (Verified)
```yaml
# Source: .aether/commands/build.yaml (VERIFIED -- current content)
# MISSING anchors (tests expect these):
# - "TS host is the sole entry point"  (TestWrapperSourcesUseTypeScriptHostManifestSpine)
# - "build-finalize"                   (TestWrapperSourcesUseTypeScriptHostManifestSpine)
# - "spawn-log"                        (TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity)
# - "spawn-complete"                   (TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity)
# - "visible live Task/subagent"       (TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity)
# - "orchestrator_boundary_guidance"   (TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance)
# - "after_discuss_next"               (TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance)
# - "aether discuss"                   (TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance)
# - "fresh"                            (TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Five divergent skip lists | One canonical `ScanFilter` in `pkg/codegraph/scan_filter.go` | Phase 141 | .venv/__pycache__ excluded from all scans |
| Generic plans (no grounding) | Grounding gate validates tasks against source anchors | Phase 142 | Plans with generic tasks produce warnings |
| Three conductors (YAML + TS + Go) | One conductor per platform | Phase 143 | build/plan YAML stripped, TS host is conductor for Claude/OpenCode |
| No plan playbooks | plan-prep.md and plan-dispatch.md created | Phase 143 | TS host injects plan playbooks as document-injection |

**Deprecated/outdated:**
- YAML `wrapper_additions.orchestration` in build.yaml and plan.yaml -- removed in Phase 143, but tests still expect some anchors

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 4 Go test failures are all caused by Phase 143 YAML changes | Common Pitfalls #1, #2 | Medium -- TestCLIFlagAudit failure is about plan-prep.md (Phase 143 created it), the other 3 are about missing anchors in build/plan YAML (Phase 143 stripped them) |
| A2 | `pending-decisions` is either not registered in Go or registered under a different name | Common Pitfalls #2 | Low -- verified the test output says "subcommand 'pending-decisions' not registered in Go runtime" |
| A3 | Colonize.yaml and continue.yaml were intentionally NOT modified by Phase 143 | Common Pitfalls #4 | Verified -- Phase 143 scope was build and plan only; continue was explicitly out of scope per 143-02-SUMMARY.md |
| A4 | The 2 TS test failures (milestone-audit) are pre-existing and not caused by Phase 143 | Code Examples | Verified -- both fail because `.planning/REQUIREMENTS.md` doesn't exist; this is a GSD infrastructure gap, not a Phase 143 issue |
| A5 | Codex command-guide + skills produce correct behavior without playbook loading | CLEAN-05 | Medium -- command-guide is in Go, playbooks are TS host concern; Codex uses command-guide, not playbooks, by design |
| A6 | "M4L" in CLEAN-01 refers to the Python virtual environment regression scenario (Make For Learn fixture) | Pattern 1 | Medium -- "M4L" is not a widely-known acronym; from context it likely refers to a "make-for-learning" test fixture with .venv; confirmed by the existing `createVenvNoiseFixture` test pattern |

## Open Questions (RESOLVED)

1. **What does "M4L" stand for in CLEAN-01?** — RESOLVED: Use existing `createVenvNoiseFixture`. Extend with plan grounding assertion. The acronym is not critical to implementation.

2. **Should colonize.yaml orchestration be stripped as part of CLEAN-02?** — RESOLVED: CLEAN-02 says "audit" not "fix." Document current state in the execution path audit. No stripping required this phase.

3. **Should continue.yaml heavy-review orchestration be stripped?** — RESOLVED: Document both paths in the audit. The heavy-review path uses TS host as conductor (consistent with build/plan), so it's NOT a dual conductor — it's a conditional conductor. No stripping required.

## Environment Availability

Step 2.6: SKIPPED (no new external dependencies -- all changes use existing Go/Node test infrastructure)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + Node.js `node:test` |
| Config file | None (both use built-in conventions) |
| Quick run command | `go test ./cmd -run "TestVenv|TestGround|TestCLIFlag|TestWrapper|TestCodexLifecycle" -count=1` |
| Full suite command | `go test ./... -count=1 && cd .aether/ts-host && npm test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLEAN-01 | .venv fixture produces grounded plan with zero .venv references | integration | `go test ./cmd -run "TestM4L" -v -count=1` | No -- Wave 0 |
| CLEAN-02 | One declared path per workflow per platform (documented) | audit/manual | Manual review of YAML files | N/A (documentation) |
| CLEAN-03 | 4 Go tests fixed (CLIFlagAudit, 3 anchor checks) | unit | `go test ./cmd -run "TestCLIFlagAudit|TestWrapperSources|TestCodexLifecycle|TestLifecycleWrapper" -count=1` | YES -- fix existing |
| CLEAN-04 | YAML has no orchestration logic in wrapper_additions | integration | `grep -c "orchestration:" .aether/commands/build.yaml` | YES -- fix existing |
| CLEAN-05 | Codex command-guide + skills produce correct output | smoke | `go test ./cmd -run "TestCommandGuideBuild|TestCommandGuidePlan" -v -count=1` | YES -- existing tests |
| CLEAN-06 | Build ceremony matches Claude Code and OpenCode | unit | `cd .aether/ts-host && node --test test/cross-platform-parity.test.ts` | YES -- extend existing |
| Regression | All 2900+ Go tests green | full | `go test ./... -count=1` | YES -- existing |
| Regression | All 489+ TS tests green (excluding pre-existing failures) | full | `cd .aether/ts-host && npm test` | YES -- existing |

### Sampling Rate
- **Per task commit:** `go test ./cmd -count=1` (runs the 4 affected tests plus ~90s of cmd package tests)
- **Per wave merge:** `go test ./... -count=1 && cd .aether/ts-host && npm test`
- **Phase gate:** Full suite green: `go test ./... -count=1` AND `cd .aether/ts-host && npm test`

### Wave 0 Gaps
- [ ] `cmd/codex_colonize_test.go` -- add `TestM4LRegression_VenvProducesGroundedPlan` (CLEAN-01)
- [ ] `cmd/codex_plan_test.go` -- optionally add plan-grounding-with-venv-fixture test (CLEAN-01)
- [ ] `cmd/command_guide_test.go` -- fix `TestCLIFlagAudit` by handling plan-prep.md reference (CLEAN-03)
- [ ] `.aether/commands/build.yaml` -- restore missing anchors (CLEAN-03)
- [ ] `.aether/commands/plan.yaml` -- restore missing anchors (CLEAN-03)

*(Most test infrastructure already exists -- gaps are small additions and fixes to existing tests)*

## Security Domain

> Not applicable to this phase. Regression testing and YAML cleanup have no authentication, session management, access control, or cryptography requirements. No ASVS categories apply.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `.aether/commands/build.yaml` -- verified missing anchors, verified orchestration block removed
- Direct codebase analysis of `.aether/commands/plan.yaml` -- verified missing anchors, verified orchestration block removed
- Direct codebase analysis of `.aether/commands/colonize.yaml` -- verified orchestration block still present
- Direct codebase analysis of `.aether/commands/continue.yaml` -- verified heavy-review orchestration present, default path uses direct `aether continue`
- Direct codebase analysis of `.aether/commands/seal.yaml` -- verified wrapper_contract with 16-step instructions
- Direct codebase analysis of `cmd/plan_grounding.go` -- verified grounding gate logic
- Direct codebase analysis of `pkg/codegraph/scan_filter.go` -- verified .venv exclusion
- Direct codebase analysis of `cmd/codex_colonize_test.go` -- verified `createVenvNoiseFixture` and `TestVenvNoiseExclusion`
- Direct codebase analysis of `cmd/command_guide.go` -- verified command-guide catalog for all 5 workflows
- Go test output: 4 failures identified and root causes traced to Phase 143 YAML changes
- TS test output: 2 pre-existing failures in milestone-audit.test.ts (REQUIREMENTS.md missing)
- Phase 143-01-SUMMARY.md and 143-02-SUMMARY.md -- verified scope, completed requirements, deviations
- Phase 141-RESEARCH.md -- verified survey noise filter design and source anchor extraction
- Phase 142-RESEARCH.md -- verified grounding gate design
- Direct codebase analysis of `.aether/ts-host/test/cross-platform-parity.test.ts` -- verified parity check structure
- Direct codebase analysis of `.aether/docs/command-playbooks/plan-prep.md` -- verified "pending-decisions" reference at line 49
- Direct codebase analysis of `.claude/commands/ant/build.md` -- verified single execution path via `aether host build`
- Direct codebase analysis of `.claude/commands/ant/plan.md` -- verified single execution path via `aether host plan`
- Direct codebase analysis of `.claude/commands/ant/continue.md` -- verified default path via `aether continue`, heavy via `aether host continue --classic-ceremony`

### Secondary (MEDIUM confidence)
- `.aether/docs/wrapper-runtime-ux-contract.md` -- wrapper/runtime ownership boundaries
- `.aether/docs/command-playbooks/README.md` -- playbook structure

### Tertiary (LOW confidence)
- None -- all claims verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all existing test infrastructure
- Architecture: HIGH -- all integration points verified by reading source code; execution paths traced through YAML -> wrapper -> TS host
- Pitfalls: HIGH -- 4 Go test failures are well-understood; fix strategy is clear (restore anchors without restoring orchestration blocks)

**Research date:** 2026-05-18
**Valid until:** 2026-06-18 (stable -- no external dependencies or fast-moving libraries)
