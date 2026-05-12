# Phase 108: Golden Workflow Tests - Research

**Researched:** 2026-05-12
**Domain:** Go golden/snapshot testing, ANSI ceremony capture, lifecycle state verification
**Confidence:** HIGH

## Summary

This phase creates golden/snapshot tests that capture the full `plan -> build 1 -> continue` lifecycle output, verify ceremony structure, worker activity, and state mutation timing. The codebase already has a well-established golden test pattern using a `-update-golden` flag and `cmd/testdata/*.json` files, plus mature test helpers (`setupBuildFlowTest`, `createTestColonyState`, `seedContinueBuildPacket`) that handle colony state setup for lifecycle tests. The Go runtime produces ANSI-decorated visual output; the existing `stripANSIEscapeCodes` function in `pkg/codex/platform_dispatch.go` can be reused for ANSI stripping before snapshotting.

The primary challenge is not technical feasibility (all building blocks exist) but orchestrating the three-step lifecycle in a single test while keeping golden files human-readable and diffs meaningful. The test needs to run `plan` (creates phases from empty plan), `build 1` (dispatches workers, writes build packet), and `continue` (reads build packet, advances phase) in sequence, capturing each command's visual output.

**Primary recommendation:** Create a single test file `cmd/golden_workflow_test.go` with three golden files in `cmd/testdata/` (one per lifecycle command), following the existing `var updateGolden = flag.Bool("update-golden", false, ...)` pattern used by `audit_catalog_test.go`, `parity_test.go`, `regression_test.go`, and `worker_economy_test.go`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use full visual output snapshot (ANSI-stripped) as golden text files. The test captures the complete ceremony output from `plan -> build 1 -> continue` and compares against a golden baseline.
- **D-02:** Strip ANSI escape codes before snapshotting. Golden files contain clean, readable text. Tests don't break on color tweaks, only on structural ceremony changes.
- **D-03:** Use the standard Go golden test `-update` flag pattern to regenerate golden files when ceremony output intentionally changes. CI fails if golden is stale.

### Claude's Discretion
- Test implementation format (Go golden test files following existing `setupBuildFlowTest` patterns)
- State mutation assertion approach (how to verify COLONY_STATE.json writes only happen after finalizers)
- CI integration (alongside existing `go test ./...` or separate target)
- Golden file location (alongside test files or in a dedicated testdata/ directory)
- Whether to also snapshot JSON output from `--plan-only` mode alongside visual output

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | Golden/snapshot test exists for `plan -> build 1 -> continue` lifecycle | Existing test helpers (`setupBuildFlowTest`, `createTestColonyState`, `seedContinueBuildPacket`) support all three lifecycle steps; golden test pattern established in `parity_test.go` |
| TEST-02 | Test captures visible ceremony output (stage separators, caste labels, worker banners) | `AETHER_OUTPUT_MODE=visual` env var forces visual rendering; all ceremony elements (stage markers, caste identity, spawn plan) verified individually in `codex_visuals_test.go` |
| TEST-03 | Test captures worker activity (spawn-log entries, dispatch manifests, worker descriptions) | `spawn-log` command records to spawn-tree.txt; build manifest includes dispatches; ceremony output includes spawn plan and wave start |
| TEST-04 | Test captures state side effects (COLONY_STATE.json mutations only after finalizers, no pre-finalize state writes) | Finalizer code in `codex_build_finalize.go`, `codex_plan_finalize.go`, `codex_continue_finalize.go` handles all state writes; boundary contract (Phase 106) documents Go ownership |
| TEST-05 | Test runs in CI and fails if ceremony, worker activity, or state behavior regresses | GitHub Actions runs `go test ./...` on every PR; golden test will fail on any visual or structural change |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Golden file comparison | Test (Go testing) | -- | Pure test concern, no runtime involvement |
| ANSI stripping | Test helper | -- | Reuse `stripANSIEscapeCodes` from `pkg/codex/platform_dispatch.go` |
| Lifecycle command execution | Go runtime (cmd/) | -- | Tests invoke `rootCmd.Execute()` directly, same as all existing tests |
| Colony state setup | Test helper | -- | `setupBuildFlowTest` + `createTestColonyState` handle this |
| Build packet seeding | Test helper | -- | `seedContinueBuildPacket` handles this |
| State mutation verification | Go runtime (store) | -- | Read COLONY_STATE.json between commands to assert transitions |
| Golden file storage | Filesystem (testdata/) | -- | Established pattern: `cmd/testdata/*.json` and `cmd/testdata/*.txt` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go testing | 1.26.1 | Test framework, `flag.Bool`, `t.TempDir()` | Already used by 2900+ tests in repo |
| Go os | stdlib | File I/O for golden read/write | All existing golden tests use `os.ReadFile`/`os.WriteFile` |
| Go strings | stdlib | ANSI stripping, comparison | `strings.Contains`, `strings.Index` used throughout |
| Go encoding/json | stdlib | JSON marshaling for state snapshots | Colony state is JSON |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/codex/platform_dispatch.go | local | `stripANSIEscapeCodes()` function | Strip ANSI from visual output before snapshotting |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `cmd/testdata/*.txt` golden files | `testdata/TestName.golden` (Go stdlib pattern) | Go stdlib `testing` has no built-in golden support; the repo already uses `flag.Bool("update-golden")` pattern -- stay consistent |
| Text golden files | JSON golden files with structured comparison | TEXT is human-readable for code review diffs; JSON would be harder to diff visually |
| Single combined golden file | Three separate golden files (one per command) | Separate files make it clear which command regressed; combined file is harder to triage |

**Installation:**
No external dependencies needed. Everything uses the Go standard library and existing codebase packages.

**Version verification:** Go 1.26.1 installed and verified on this machine. [VERIFIED: `go version`]

## Architecture Patterns

### System Architecture Diagram

```
Test Execution Flow
===================

[1. Test Setup]
    |
    v
[setupBuildFlowTest] --> tmpDir, dataDir, AETHER_ROOT set, store initialized
    |
    v
[createTestColonyState] --> COLONY_STATE.json (state=READY, empty plan)
    |
    v
[2. aether plan] --> visual output captured in stdout buffer
    |                    |
    |                    v
    |                [stripANSIEscapeCodes] --> clean text
    |                    |
    |                    v
    |                [compare with golden_plan.txt]
    |                    |
    v
[3. aether build 1] --> visual output captured in stdout buffer
    |                     |
    |                     v
    |                 [stripANSIEscapeCodes] --> clean text
    |                     |
    |                     v
    |                 [compare with golden_build.txt]
    |                     |
    v
[4. seedContinueBuildPacket] --> build/phase-1/ directory with dispatch data
    |
    v
[5. aether continue] --> visual output captured in stdout buffer
    |                      |
    |                      v
    |                  [stripANSIEscapeCodes] --> clean text
    |                      |
    |                      v
    |                  [compare with golden_continue.txt]
    |                      |
    v
[6. State Assertions] --> COLONY_STATE.json verified:
    - state transitioned from READY -> BUILT -> COMPLETED (or READY for multi-phase)
    - phase 1 status is "completed"
    - current_phase advanced to 2 (or colony marked complete)
```

### Recommended Project Structure
```
cmd/
  golden_workflow_test.go         # New test file with 3 lifecycle golden tests
  testdata/
    golden_plan.txt               # Plan visual output baseline
    golden_build.txt              # Build visual output baseline
    golden_continue.txt           # Continue visual output baseline
```

### Pattern 1: Golden Test with -update-golden Flag

**What:** The established pattern in this codebase for golden/snapshot testing. A package-level `var updateGolden = flag.Bool("update-golden", false, ...)` flag controls whether golden files are written or compared.

**When to use:** Any test that needs a snapshot baseline that can be refreshed intentionally.

**Example:**
```go
// Source: cmd/audit_catalog_test.go (established pattern)
var updateGolden = flag.Bool("update-golden", false, "update golden files")

func TestAuditCatalogGolden(t *testing.T) {
    catalog := buildAuditCatalog(rootCmd)
    data, err := json.MarshalIndent(catalog, "", "  ")
    // ...
    goldenPath := "testdata/audit_catalog.json"

    if *updateGolden {
        if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
            t.Fatalf("write golden file: %v", err)
        }
        t.Logf("golden file updated: %s", goldenPath)
        return
    }

    golden, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("read golden file: %v (run with -update-golden to create)", err)
    }
    // Compare with trailing newline normalization
    got := string(data) + "\n"
    want := string(golden)
    if got != want {
        t.Errorf("catalog golden mismatch; run with -update-golden to refresh")
    }
}
```

[VERIFIED: cmd/audit_catalog_test.go, cmd/parity_test.go, cmd/regression_test.go, cmd/worker_economy_test.go]

### Pattern 2: Lifecycle Test Setup with Visual Mode

**What:** The established pattern for running lifecycle commands with visual output capture.

**When to use:** Any test that exercises plan, build, or continue commands with visual rendering.

**Example:**
```go
// Source: cmd/codex_visuals_test.go (established pattern)
func TestBuildVisualOutputShowsSpawnPlan(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    dataDir := setupBuildFlowTest(t)
    t.Setenv("AETHER_OUTPUT_MODE", "visual")

    goal := "Improve command visuals"
    taskOneID := "task-1"
    taskTwoID := "task-2"
    createTestColonyState(t, dataDir, colony.ColonyState{
        Version: "3.0",
        Goal:    &goal,
        State:   colony.StateREADY,
        Plan: colony.Plan{
            Phases: []colony.Phase{
                {
                    ID: 1, Name: "Visual pass", Status: colony.PhaseReady,
                    Tasks: []colony.Task{
                        {ID: &taskOneID, Goal: "Implement lifecycle renderer", Status: colony.TaskPending},
                        {ID: &taskTwoID, Goal: "Document the new output style", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
                    },
                },
            },
        },
    })

    rootCmd.SetArgs([]string{"build", "1"})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("build returned error: %v", err)
    }
    output := stdout.(*bytes.Buffer).String()
    // Assert on output...
}
```

[VERIFIED: cmd/codex_visuals_test.go]

### Pattern 3: Continue Test with Build Packet Seeding

**What:** The continue command requires a pre-existing build packet (dispatch data from a previous build). The `seedContinueBuildPacket` helper creates this data.

**When to use:** Any test that runs the continue command after a build.

**Example:**
```go
// Source: cmd/codex_visuals_test.go (established pattern)
func TestContinueVisualOutputShowsVerificationArtifactsAndSpawnTree(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    dataDir := setupBuildFlowTest(t)
    root := filepath.Dir(filepath.Dir(dataDir))
    withTestWorkspace(t, root)
    withWorkingDir(t, root)
    t.Setenv("AETHER_OUTPUT_MODE", "visual")

    goal := "Surface continue contracts"
    now := mustParseRFC3339(t, "2026-04-20T11:00:00Z")
    taskID := "1.1"
    nextTaskID := "2.1"
    createTestColonyState(t, dataDir, colony.ColonyState{
        Version: "3.0", Goal: &goal, State: colony.StateBUILT,
        CurrentPhase: 1, BuildStartedAt: &now,
        Plan: colony.Plan{
            Phases: []colony.Phase{
                {
                    ID: 1, Name: "Verify contracts", Status: colony.PhaseInProgress,
                    Tasks: []colony.Task{{ID: &taskID, Goal: "Verify the build packet", Status: colony.TaskInProgress}},
                },
                {
                    ID: 2, Name: "Next phase", Status: colony.PhasePending,
                    Tasks: []colony.Task{{ID: &nextTaskID, Goal: "Keep going", Status: colony.TaskPending}},
                },
            },
        },
    })

    dispatches := []codexBuildDispatch{
        {Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Verify the build packet", Status: "spawned", TaskID: taskID},
        {Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification before advancement", Status: "spawned"},
    }
    seedContinueBuildPacket(t, dataDir, 1, "Verify contracts", goal, dispatches)

    rootCmd.SetArgs([]string{"continue"})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("continue returned error: %v", err)
    }
    // Assert on output...
}
```

[VERIFIED: cmd/codex_visuals_test.go]

### Pattern 4: ANSI Stripping

**What:** The `stripANSIEscapeCodes` function in `pkg/codex/platform_dispatch.go` removes ANSI escape sequences from strings.

**When to use:** Before comparing visual output against golden files.

**Example:**
```go
// Source: pkg/codex/platform_dispatch.go (verified existing function)
func stripANSIEscapeCodes(value string) string {
    var b strings.Builder
    inEscape := false
    for i := 0; i < len(value); i++ {
        if value[i] == '\x1b' {
            inEscape = true
            continue
        }
        if inEscape {
            if value[i] == 'm' {
                inEscape = false
            }
            continue
        }
        b.WriteByte(value[i])
    }
    return b.String()
}
```

[VERIFIED: pkg/codex/platform_dispatch.go lines 1082-1099]

**Note:** This function is not exported (lowercase). The golden test file will either need to be in `cmd/` package (where it can access internal helpers via a wrapper) or a thin exported wrapper must be added. Since all existing test files that need this are in `cmd/`, keeping the golden test there is natural. However, `stripANSIEscapeCodes` is in `pkg/codex/` -- so the test needs an exported wrapper or an inline reimplementation. An inline reimplementation is simplest and avoids cross-package coupling.

### Anti-Patterns to Avoid
- **Don't test JSON output in golden tests:** JSON golden tests already exist (`parity_snapshot.json`, `regression_snapshot.json`). This phase is specifically about visual ceremony output. Don't mix concerns.
- **Don't parse visual output as authority:** The boundary contract explicitly forbids this. The golden test captures visual output for regression detection, not for extracting state.
- **Don't write to `.aether/data/` in tests outside of finalizers:** The boundary contract (Phase 106) mandates Go owns all state mutation. Tests should verify this invariant, not violate it.
- **Don't use `AETHER_FORCE_COLOR=1` in golden tests:** This would embed ANSI codes that the golden comparison then needs to strip. Instead, rely on visual mode output and strip whatever ANSI appears naturally.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ANSI escape code stripping | Custom regex | `stripANSIEscapeCodes` from `pkg/codex/platform_dispatch.go` (copy or wrap) | Already handles the full ANSI escape sequence format including multi-byte sequences |
| Golden file update mechanism | Custom flag parsing | `flag.Bool("update-golden", false, ...)` pattern from existing tests | Established pattern with `go test -update-golden` convention |
| Colony state setup | Manual JSON file writing | `setupBuildFlowTest` + `createTestColonyState` | Creates temp dir, sets env vars, initializes store, captures stdout |
| Build packet seeding | Manual directory creation | `seedContinueBuildPacket` | Creates correct build packet structure for continue command |
| Test isolation | Manual env cleanup | `saveGlobals(t)` + `resetRootCmd(t)` + `t.Cleanup` | Handles global state restoration between tests |

**Key insight:** All the infrastructure for this phase already exists. The work is assembly, not invention.

## Common Pitfalls

### Pitfall 1: ANSI Codes in Golden Files
**What goes wrong:** Golden files contain `\x1b[33m` sequences that make diffs unreadable and break on color palette changes.
**Why it happens:** Visual mode emits ANSI by default when `AETHER_FORCE_COLOR=1` or when output is a TTY.
**How to avoid:** Strip ANSI codes before writing to golden files AND before comparison. Never set `AETHER_FORCE_COLOR=1` in the test.
**Warning signs:** Golden file contains `\x1b[` sequences visible in `git diff`.

### Pitfall 2: Non-Deterministic Output
**What goes wrong:** Golden comparison fails because timestamps, durations, or worker names change between runs.
**Why it happens:** Visual output includes `BuildStartedAt`, worker durations, and deterministic names that depend on caste+task hashing (which is stable, but could change if task descriptions change).
**How to avoid:** Use fixed task descriptions and goal strings in the test colony state. The deterministic name function (`deterministicAntName`) is hash-based and stable for identical inputs. Verify that timestamps are either excluded from golden comparison or fixed.
**Warning signs:** Golden test fails intermittently with only timestamp or duration differences.

### Pitfall 3: stdout Buffer Not Reset Between Commands
**What goes wrong:** The second command's output includes the first command's output.
**Why it happens:** The test reuses `stdout` buffer across commands without resetting.
**How to avoid:** Assign a fresh `&bytes.Buffer{}` to `stdout` before each `rootCmd.Execute()` call.
**Warning signs:** Golden file is twice as long as expected, or contains output from multiple commands.

### Pitfall 4: Continue Requires BUILT State + Build Packet
**What goes wrong:** `aether continue` fails because colony state is not `StateBUILT` or build packet is missing.
**Why it happens:** Continue expects the colony to have been through a build cycle. The state must be `StateBUILT` with `BuildStartedAt` set, and the build packet directory must exist.
**How to avoid:** After running `build 1`, reload colony state and verify it transitioned to `StateBUILT`. Then call `seedContinueBuildPacket` if the build command's synthetic path doesn't create a full build packet.
**Warning signs:** Continue returns error about missing build packet or invalid colony state.

### Pitfall 5: Golden File in Wrong Directory
**What goes wrong:** `go test` can't find the golden file because the path is relative to the wrong directory.
**Why it happens:** Go test files use relative paths from the package directory. If the test file is in `cmd/`, the golden path `testdata/golden.txt` resolves to `cmd/testdata/golden.txt`.
**How to avoid:** Place golden files in `cmd/testdata/` alongside existing golden files. Use relative paths from the test file location.
**Warning signs:** `read golden file: no such file or directory` in test output.

### Pitfall 6: Plan Command Needs Workspace Context
**What goes wrong:** `aether plan` produces different output (or errors) depending on whether a workspace (go.mod, etc.) exists.
**Why it happens:** Plan uses `skillWorkspaceRoot()` which may require a Go module context.
**How to avoid:** Use `withTestWorkspace(t, root)` helper to set up a minimal workspace, consistent with existing visual tests (`TestPlanVisualOutputShowsDispatchContractDetails`).
**Warning signs:** Plan output differs between local runs and CI.

## Code Examples

Verified patterns from official sources:

### Complete Golden Test Structure
```go
// cmd/golden_workflow_test.go
package cmd

import (
    "bytes"
    "flag"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/calcosmic/Aether/pkg/colony"
)

var updateGolden = flag.Bool("update-golden", false, "update golden workflow files")

// stripANSI removes ANSI escape codes from visual output.
func stripANSI(s string) string {
    var b strings.Builder
    inEscape := false
    for i := 0; i < len(s); i++ {
        if s[i] == '\x1b' {
            inEscape = true
            continue
        }
        if inEscape {
            if s[i] == 'm' {
                inEscape = false
            }
            continue
        }
        b.WriteByte(s[i])
    }
    return b.String()
}

func compareGolden(t *testing.T, goldenPath, got string) {
    t.Helper()
    clean := stripANSI(got)

    if *updateGolden {
        if err := os.WriteFile(goldenPath, []byte(clean), 0644); err != nil {
            t.Fatalf("write golden: %v", err)
        }
        t.Logf("golden updated: %s", goldenPath)
        return
    }

    want, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("read golden: %v (run with -update-golden to create)", err)
    }
    if clean != string(want) {
        t.Errorf("golden mismatch; run with -update-golden to refresh")
        // Show diff context...
    }
}

func TestGoldenPlanVisualOutput(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    dataDir := setupBuildFlowTest(t)
    root := filepath.Dir(filepath.Dir(dataDir))
    withTestWorkspace(t, root)
    withWorkingDir(t, root)
    t.Setenv("AETHER_OUTPUT_MODE", "visual")

    goal := "Golden workflow test colony"
    createTestColonyState(t, dataDir, colony.ColonyState{
        Version: "3.0", Goal: &goal, State: colony.StateREADY,
        Plan: colony.Plan{Phases: []colony.Phase{}},
    })

    stdout = &bytes.Buffer{}
    rootCmd.SetArgs([]string{"plan"})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("plan failed: %v", err)
    }

    compareGolden(t, filepath.Join("testdata", "golden_plan.txt"), stdout.(*bytes.Buffer).String())
}
```

### Ceremony Elements to Assert in Golden Files
```go
// Key ceremony elements that MUST appear in each golden file:

// Plan golden:
//   "P L A N", "P L A N   D I S P A T C H", "Planning Wave", "aether build 1"
//   Stage markers: phase list with numbered phases

// Build golden:
//   "B U I L D   D I S P A T C H   1", "S P A W N   P L A N",
//   "Builder", "Watcher", caste identity labels
//   Stage markers in order: "── Context ──", "── Tasks ──", "── Dispatch ──",
//     "── Verification [", "── Housekeeping ──", "── Colony Complete ──"
//   "A R T I F A C T S" section with file paths
//   "It's safe to clear your context now."

// Continue golden:
//   "── Verification ──", "Phase 1 verified and completed"
//   "Continue Worker Flow", watcher completion entries
//   "── Housekeeping ──", "A R T I F A C T S"
//   "── Next Phase ──" (or "── Colony Complete ──" for final phase)
```

[VERIFIED: cmd/codex_visuals_test.go -- all listed elements tested individually]

### State Mutation Verification Pattern
```go
// After each command, verify COLONY_STATE.json mutations:
func assertColonyState(t *testing.T, dataDir string, checks func(colony.ColonyState)) {
    t.Helper()
    var state colony.ColonyState
    if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
        t.Fatalf("reload state: %v", err)
    }
    checks(state)
}

// Usage:
// After plan:
assertColonyState(t, dataDir, func(s colony.ColonyState) {
    if s.State != colony.StateREADY {
        t.Errorf("after plan: expected state READY, got %s", s.State)
    }
    if len(s.Plan.Phases) == 0 {
        t.Error("after plan: expected phases to be generated")
    }
})

// After build:
assertColonyState(t, dataDir, func(s colony.ColonyState) {
    if s.State != colony.StateBUILT {
        t.Errorf("after build: expected state BUILT, got %s", s.State)
    }
    if s.CurrentPhase != 1 {
        t.Errorf("after build: expected current_phase=1, got %d", s.CurrentPhase)
    }
})

// After continue:
assertColonyState(t, dataDir, func(s colony.ColonyState) {
    if s.Plan.Phases[0].Status != colony.PhaseCompleted {
        t.Errorf("after continue: expected phase 1 completed, got %s", s.Plan.Phases[0].Status)
    }
})
```

[VERIFIED: cmd/build_flow_cmds_test.go, cmd/codex_visuals_test.go]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No golden tests for lifecycle | Golden tests with `-update-golden` flag | Phase 108 (this phase) | New behavioral regression safety net |
| Individual `strings.Contains` checks per ceremony element | Full snapshot comparison | Phase 108 (this phase) | Catches structural changes individual checks miss |
| JSON-only golden tests (parity, regression snapshots) | Text golden tests for visual output | Phase 108 (this phase) | Human-readable diffs in code review |

**Deprecated/outdated:**
- None in this domain. The existing golden test pattern is current and consistent.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `stripANSIEscapeCodes` handles all ANSI sequences in visual output | Pattern 4 | If visual rendering adds new ANSI sequences not handled, golden files may contain residual escape codes |
| A2 | `seedContinueBuildPacket` creates sufficient build packet for continue to succeed | Pattern 3 | If continue requires additional fields not seeded, the test will fail at the continue step |
| A3 | Deterministic worker names (`deterministicAntName`) are stable for identical inputs | Pitfall 2 | If the hash function changes, golden files need regeneration |
| A4 | Plan command produces output without requiring real worker dispatches (synthetic/simulated mode) | Pattern 2 | If plan requires external platform connections, the test may fail in CI |
| A5 | The `cmd/testdata/` directory is committed to git | Pitfall 5 | If testdata is gitignored, golden files won't be available in CI |

## Open Questions (RESOLVED)

1. **RESOLVED: Should the golden test also capture JSON `--plan-only` output?**
   - Decision: Start with visual-only golden. JSON manifest contracts are already tested by `codex_build_test.go` and `codex_continue_test.go`. Revisit if Phase 109 (TS host) needs JSON golden baselines.

2. **RESOLVED: How to handle the `withTestWorkspace` requirement for plan?**
   - Decision: Use `withTestWorkspace(t, root)` + `withWorkingDir(t, root)` as established in `TestPlanVisualOutputShowsDispatchContractDetails`. This creates a minimal `go.mod` + `cmd/main.go` workspace.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified -- all tooling is Go stdlib and existing codebase packages)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing 1.26.1 (stdlib) |
| Config file | none -- uses Go defaults |
| Quick run command | `go test ./cmd/ -run "TestGolden" -count=1` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | Full lifecycle golden snapshot | golden (snapshot) | `go test ./cmd/ -run "TestGoldenPlanVisualOutput|TestGoldenBuildVisualOutput|TestGoldenContinueVisualOutput" -count=1` | No -- Wave 0 |
| TEST-02 | Ceremony output captured (stage markers, caste labels) | golden (snapshot) | Same as TEST-01 -- golden file content includes ceremony | No -- Wave 0 |
| TEST-03 | Worker activity captured (spawn-log, dispatches) | golden (snapshot) | Same as TEST-01 -- golden file content includes worker activity | No -- Wave 0 |
| TEST-04 | State mutations only after finalizers | unit | `go test ./cmd/ -run "TestGoldenStateMutations" -count=1` | No -- Wave 0 |
| TEST-05 | Test runs in CI | integration | `go test ./... -race` (includes golden tests) | N/A -- CI config |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestGolden" -count=1`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/golden_workflow_test.go` -- the main test file with golden lifecycle tests
- [ ] `cmd/testdata/golden_plan.txt` -- plan visual output baseline (created by first `-update-golden` run)
- [ ] `cmd/testdata/golden_build.txt` -- build visual output baseline (created by first `-update-golden` run)
- [ ] `cmd/testdata/golden_continue.txt` -- continue visual output baseline (created by first `-update-golden` run)

**Framework install:** None needed -- Go stdlib testing is always available.

## Security Domain

> Security is not a primary concern for this phase (test code only, no user-facing changes).
> The boundary contract verification (TEST-04) is a correctness check, not a security control.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | no | N/A -- test code, no user input |
| V6 Cryptography | no | N/A -- no cryptographic operations |

## Sources

### Primary (HIGH confidence)
- [cmd/audit_catalog_test.go](cmd/audit_catalog_test.go) -- established `-update-golden` flag pattern, golden file read/write
- [cmd/parity_test.go](cmd/parity_test.go) -- `TestPlatformParityGolden` with JSON golden comparison
- [cmd/regression_test.go](cmd/regression_test.go) -- `TestRegressionSnapshot` with multi-dimension golden comparison
- [cmd/build_flow_cmds_test.go](cmd/build_flow_cmds_test.go) -- `setupBuildFlowTest`, `createTestColonyState` helpers
- [cmd/codex_visuals_test.go](cmd/codex_visuals_test.go) -- visual output test patterns with `AETHER_OUTPUT_MODE=visual`, ceremony element assertions
- [cmd/codex_continue_test.go](cmd/codex_continue_test.go) -- `seedContinueBuildPacket` helper
- [pkg/codex/platform_dispatch.go](pkg/codex/platform_dispatch.go) -- `stripANSIEscapeCodes` function
- [.aether/references/contracts/runtime-boundary-contract.md](.aether/references/contracts/runtime-boundary-contract.md) -- Go state mutation ownership, anti-patterns
- [cmd/codex_build_finalize.go](cmd/codex_build_finalize.go) -- build finalizer with provenance validation
- [cmd/codex_continue_finalize.go](cmd/codex_continue_finalize.go) -- continue finalizer with gates
- [cmd/codex_plan_finalize.go](cmd/codex_plan_finalize.go) -- plan finalizer
- [cmd/ceremony_cmd.go](cmd/ceremony_cmd.go) -- ceremony structures (ceremonyDispatch, ceremonyExecutionPlan)

### Secondary (MEDIUM confidence)
- [.planning/codebase/TESTING.md](.planning/codebase/TESTING.md) -- test framework overview, patterns, quality gates

### Tertiary (LOW confidence)
- None -- all claims verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all Go stdlib, no external dependencies
- Architecture: HIGH -- all patterns verified against existing codebase tests
- Pitfalls: HIGH -- pitfalls derived from reading existing test code and understanding ANSI/visual output behavior

**Research date:** 2026-05-12
**Valid until:** 60 days (stable domain -- Go testing patterns and ceremony output structure evolve slowly)
