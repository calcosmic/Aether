---
phase: 89-gate-self-healing-smart-planning
reviewed: 2026-05-01T20:30:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - cmd/fixer_dispatch.go
  - cmd/codex_visuals.go
  - cmd/unblock_cmd.go
  - cmd/gate.go
  - cmd/oracle_loop.go
  - cmd/compatibility_cmds.go
  - cmd/init_ceremony.go
  - cmd/status.go
  - pkg/codex/worker.go
findings:
  critical: 2
  warning: 7
  info: 4
  total: 13
status: issues_found
---

# Phase 89: Code Review Report

**Reviewed:** 2026-05-01T20:30:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed 9 files spanning the gate self-healing system, fixer dispatch, oracle research loop, worker invocation, colony init ceremony, status dashboard, and Codex compatibility layer. The codebase is large and well-structured with thorough error handling, but contains two critical findings: a command injection vulnerability in the gate check system where test commands are extracted from markdown files and executed without sanitization, and a potential nil pointer dereference in the status dashboard. Several warnings cover race conditions, data loss risks, and inconsistent state handling.

## Critical Issues

### CR-01: Command injection via unsanitized test command extraction from markdown files

**File:** `/Users/callumcowie/repos/Aether/cmd/gate.go:162-183`
**Issue:** `checkTestsPass()` calls `resolveTestCommand()` which reads CLAUDE.md and CODEBASE.md, then `extractTestCommand()` parses arbitrary markdown content looking for test commands. The extracted string is passed directly to `exec.Command(parts[0], parts[1:]...)` on line 183. A malicious or compromised markdown file could contain a crafted line like `go test; rm -rf /` or backtick/shell-operator payloads that would be executed as a subprocess. The `extractTestCommand` function (lines 258-283) performs no sanitization -- it extracts raw text after finding a keyword match and only trims at `#` or `//` comment markers, which does not prevent shell metacharacters.

**Fix:**
```go
func checkTestsPass() gateCheck {
    testCmd := resolveTestCommand()
    if testCmd == "" {
        return gateCheck{
            Name:   "tests_pass",
            Passed: true,
            Detail: "no test command found, skipping",
        }
    }

    parts := strings.Fields(testCmd)
    if len(parts) == 0 {
        return gateCheck{
            Name:   "tests_pass",
            Passed: true,
            Detail: "empty test command, skipping",
        }
    }

    // Validate: only allow known-safe base commands
    allowed := map[string]bool{
        "go": true, "npm": true, "npx": true, "cargo": true,
        "mvn": true, "pytest": true, "jest": true, "vitest": true,
    }
    if !allowed[parts[0]] {
        return gateCheck{
            Name:   "tests_pass",
            Passed: false,
            Detail: fmt.Sprintf("unrecognized test command base %q, refusing to execute", parts[0]),
        }
    }

    // Reject shell metacharacters in arguments
    for _, arg := range parts[1:] {
        if strings.ContainsAny(arg, "|&;`$(){}[]<>!") {
            return gateCheck{
                Name:   "tests_pass",
                Passed: false,
                Detail: "test command contains shell metacharacters, refusing to execute",
            }
        }
    }

    cmd := exec.Command(parts[0], parts[1:]...)
    cmd.Dir = storage.ResolveAetherRoot(context.Background())
    output, err := cmd.CombinedOutput()
    // ... rest unchanged
}
```

### CR-02: Nil pointer dereference when colony Goal is nil

**File:** `/Users/callumcowie/repos/Aether/cmd/status.go:404`
**Issue:** `renderDashboard()` dereferences `state.Goal` without a nil check on line 404: `goal := *state.Goal`. The `ColonyState.Goal` field is `*string` (a pointer), and while `createCeremonyColony` always sets it, older colonies, entombed states, or manually-edited COLONY_STATE.json files could have `Goal: null`. The `loadActiveColonyState()` function (called on line 25) does not guarantee Goal is non-nil. This would crash the status command with a nil pointer dereference.

**Fix:**
```go
// Line 404 in status.go
goal := ""
if state.Goal != nil {
    goal = *state.Goal
}
if len(goal) > 60 {
    goal = goal[:57] + "..."
}
```

## Warnings

### WR-01: Race condition on package-level `stdout` variable in init ceremony

**File:** `/Users/callumcowie/repos/Aether/cmd/init_ceremony.go:200-205`
**Issue:** `runCeremonyResearch()` temporarily replaces the global `stdout` variable to capture output from `initResearchCmd`, then restores it in a deferred function. If two ceremonies run concurrently (e.g., in tests or parallel invocations), this shared mutable global will cause data races and corrupted output. The same pattern exists with `origStdout := stdout` / `defer func() { stdout = origStdout }()`.

**Fix:** Either pass an explicit output writer through the function chain instead of mutating a package-level variable, or document that this function is not safe for concurrent use and add a mutex guard.

### WR-02: `globalCircuitBreaker` is never initialized outside tests

**File:** `/Users/callumcowie/repos/Aether/cmd/fixer_dispatch.go:19`
**Issue:** `globalCircuitBreaker` is declared as `var globalCircuitBreaker *CircuitBreaker` but is only assigned in test files (fixer_dispatch_test.go, unblock_cmd_test.go). In production, `isFixerDispatchBlocked()` (line 104) and `recordFixerFailure()` (line 257) both check `if cb == nil { return false/" }` and silently skip all circuit breaker logic. This means the circuit breaker feature is completely inert in production -- the unblock command's D-04 circuit breaker protection never actually fires. Users who rely on `DefaultMaxUnblockAttempts = 1` as their protection are actually only protected by the attempt cap, not the circuit breaker.

**Fix:** Initialize `globalCircuitBreaker` at package init time or in the command's `RunE` function, e.g.:
```go
func init() {
    globalCircuitBreaker = NewCircuitBreaker(3)
}
```

### WR-03: `gateResultsWritePhase` loses the wrapper format on re-write

**File:** `/Users/callumcowie/repos/Aether/cmd/gate.go:605-608`
**Issue:** `gateResultsWritePhase()` writes gate results as a plain JSON array (`[]GateCheckResult`). But `readGateResultsPhase()` (line 613) and `readGateResultsPhase()` in `fixer_dispatch.go` (line 43) both support the newer `gateResultsFile` wrapper format that includes `unblock_attempts`. When `resolveFixedGates()` in `fixer_dispatch.go` (line 240) writes back the updated results using `store.SaveJSON(rel, fileData)` where `fileData` is a `*gateResultsFile`, this correctly preserves the wrapper. However, `gateResultsWritePhase()` on line 607 writes a raw `[]GateCheckResult`, which means any gate results written through this path will lose the `unblock_attempts` field if they were previously in wrapper format. The two write paths are inconsistent.

**Fix:** Update `gateResultsWritePhase` to use the wrapper format, or at minimum, detect existing format and preserve it:
```go
func gateResultsWritePhase(phaseNum int, entries []GateCheckResult) error {
    rel := fmt.Sprintf("gate-results-%d.json", phaseNum)
    // Check if existing file uses wrapper format
    existing, err := readGateResultsPhase(phaseNum)
    if err == nil {
        wrapped := &gateResultsFile{
            Attempts: readUnblockAttempts(phaseNum),
            Results:  entries,
        }
        return store.SaveJSON(rel, wrapped)
    }
    return store.SaveJSON(rel, entries)
}
```

### WR-04: Oracle loop silently drops state update on iteration count mismatch

**File:** `/Users/callumcowie/repos/Aether/cmd/oracle_loop.go:570`
**Issue:** At line 570, after processing an iteration, the code sets `state.Iteration = iterationsRun`. But `iterationsRun` is a local counter that starts at 0 and is incremented at line 461 (`iterationsRun++`), while `state.Iteration` was already incremented at line 432 (`state.Iteration++`). These are tracking the same thing but `iterationsRun` is one step behind because `state.Iteration` was already written to disk at line 432. When the loop ends via max iterations (line 612), `state.Iteration` gets overwritten with `iterationsRun`, potentially under-reporting the final iteration count by 1.

**Fix:** Remove the reassignment on line 570 or ensure consistency:
```go
// Line 570: Remove this line -- state.Iteration was already set at line 432
// state.Iteration = iterationsRun  // DELETE THIS LINE
```

### WR-05: `runCompatibilityAutopilot` has no max iteration guard against infinite loops

**File:** `/Users/callumcowie/repos/Aether/cmd/compatibility_cmds.go:215-313`
**Issue:** The `runCompatibilityAutopilot` function has a `for {}` loop (line 215) that only terminates when colony state reaches COMPLETED, max phases, replan due, or blocked. If `runCodexBuildWithOptions` and `runCodexContinue` succeed but neither advances the phase nor blocks (e.g., a bug where the colony stays in StateBUILT or StateEXECUTING indefinitely without advancing), this loop will run forever. The `MaxPhases` guard only triggers on the READY->build path, not on the EXECUTING/BUILT->continue path (where `phasesCompleted` is incremented).

**Fix:** Add a hard upper bound on total loop iterations:
```go
const maxAutopilotIterations = 100
loopIterations := 0
for {
    loopIterations++
    if loopIterations > maxAutopilotIterations {
        return nil, fmt.Errorf("autopilot exceeded %d total iterations without completion", maxAutopilotIterations)
    }
    // ... existing loop body
}
```

### WR-06: `formatSkipSummary` always reports failures even when all gates passed

**File:** `/Users/callumcowie/repos/Aether/cmd/gate.go:643-657`
**Issue:** `formatSkipSummary()` classifies any gate that is not `Passed: true` as "failed" (line 653: `failed++`). But `GateResultEntry` uses a `Passed` boolean, and there is no "skipped" state at this level. The `shouldSkipGate` function (line 557) operates on `GateCheckResult` which has a `Status` string with "passed", "failed", "skipped" states. These are two different types. If a `GateResultEntry` has `Passed: false` but the underlying check was actually skipped (not failed), the summary will incorrectly report "re-checking N failures" when no actual failures exist.

**Fix:** Either add a `Skipped` field to `GateResultEntry`, or rename the summary to be more accurate:
```go
func formatSkipSummary(priorResults []colony.GateResultEntry) string {
    // ...
    for _, r := range priorResults {
        if r.Passed {
            passed++
        } else {
            // Cannot distinguish failed from skipped with current schema
            notPassed++
        }
    }
    return fmt.Sprintf("Skipping %d passed gates -- re-checking %d others", passed, notPassed)
}
```

### WR-07: `quoteCommandArg` replaces backtick with single quote, creating invalid shell syntax

**File:** `/Users/callumcowie/repos/Aether/cmd/status.go:966-970`
**Issue:** The `quoteCommandArg` function creates a double-quoted shell string, replacing `\`, `"`, and backtick characters. However, the replacement `"` -> `'` inside double quotes does not escape the single quote -- it inserts a literal single quote character which is valid inside double quotes. But the intent appears to be sanitizing for shell injection. The actual issue is that this function produces output intended for display (not shell execution), so the escaping is misleading. If any downstream consumer were to pass this to a shell, the `"\n"` -> `" "` replacement on line 968 would allow newline injection inside the quoted string.

**Fix:** If this is display-only, rename to make intent clear. If it needs to be shell-safe, properly escape the content:
```go
func quoteCommandArg(text string) string {
    text = compactActionText(text, 140)
    // Use shell-safe escaping for double-quoted strings
    replacer := strings.NewReplacer(
        `\`, `\\`,
        `"`, `\"`,
        "$", `\$`,
        "`", "\\`",
        "\n", " ",
    )
    return `"` + replacer.Replace(text) + `"`
}
```

## Info

### IN-01: Unused import and variable shadowing in gate.go

**File:** `/Users/callumcowie/repos/Aether/cmd/gate.go:5`
**Issue:** The imports include `"encoding/json"`, `"os"`, `"path/filepath"`, `"sort"`, `"strings"`, and `"time"`. Not all of these are used in every function, but since they are used across the file this is acceptable. However, `strconv` (line 8) is imported and only used in the `shouldSkipGateCmd` handler (line 727). This is fine -- just noting for completeness.

### IN-02: Duplicate confidence parsing logic in oracle_loop.go

**File:** `/Users/callumcowie/repos/Aether/cmd/oracle_loop.go:52-63` and `347-351`
**Issue:** The `validateOracleConfidenceTarget` function (lines 52-63) parses a confidence string into an integer and validates it. Then in `startOracleCompatibility` (lines 347-351), the same parsing is done again manually (`v = v*10 + int(ch-'0')`). This is duplicated logic that could diverge.

**Fix:** Refactor to reuse `validateOracleConfidenceTarget` for parsing, or extract a shared `parseConfidenceTarget(value string) (int, error)` function.

### IN-03: `firstActionableFlag` panics on empty flags slice

**File:** `/Users/callumcowie/repos/Aether/cmd/status.go:258-270`
**Issue:** `firstActionableFlag()` is called only when `len(active) > 0` (line 238), so the `return flags[0]` on line 269 will never panic in current usage. However, the function has no guard against an empty slice, making it fragile if called from elsewhere in the future.

**Fix:** Add a defensive check or document the precondition:
```go
// firstActionableFlag returns the highest-priority actionable flag.
// Precondition: len(flags) > 0.
func firstActionableFlag(flags []colony.FlagEntry) colony.FlagEntry {
```

### IN-04: `codex_visuals.go` is 2900+ lines

**File:** `/Users/callumcowie/repos/Aether/cmd/codex_visuals.go:1-2901`
**Issue:** At 2900+ lines, this file is extremely large and contains rendering logic for every command in the system. This makes navigation and maintenance difficult. Consider splitting into focused files (e.g., `visual_init.go`, `visual_build.go`, `visual_continue.go`, `visual_status.go`, etc.).

**Fix:** Split into domain-grouped visual rendering files based on the command they serve.

---

_Reviewed: 2026-05-01T20:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
