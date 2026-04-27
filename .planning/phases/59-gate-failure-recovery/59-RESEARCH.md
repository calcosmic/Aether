# Phase 59: Gate Failure Recovery - Research

**Researched:** 2026-04-27
**Domain:** Continue verification gate system (Go runtime + playbook layer)
**Confidence:** HIGH

## Summary

This phase modifies the continue verification flow to provide actionable recovery instructions when gates fail (GATE-01), replace the automatic Watcher Veto stash with an explicit user choice (GATE-02), and implement incremental gate re-checking so re-running `/ant-continue` only re-runs previously failed gates (GATE-03).

The continue verification flow spans two layers: the Go runtime (`cmd/codex_continue.go`, `cmd/gate.go`) which runs structural checks and produces JSON reports, and the wrapper playbook layer (`.aether/docs/command-playbooks/continue-verify.md`, `continue-gates.md`) which orchestrates agent-based gates and renders user-facing output. This phase must modify both layers.

The gate system is well-structured: each gate is a distinct step (Steps 1.5-1.14) in `continue-gates.md`, and the Go runtime already has a `gateCheck` struct with `Name`, `Passed`, and `Detail` fields. The `runCodexContinueGates()` function produces a `codexContinueGateReport` with structured checks and blocking issues. The `ColonyState` struct in `pkg/colony/colony.go` needs a new `GateResults` field for persisting results between continue runs.

**Primary recommendation:** Add a `GateResults` field to `ColonyState`, extend `gateCheck` with a `RecoveryTemplate` field, modify each gate step in the playbooks and runtime to read/write gate results, and replace the auto-stash in Watcher Veto with a three-choice `AskUserQuestion` prompt.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Per-gate recovery templates -- each gate type has a hardcoded recovery template with specific steps
- **D-02:** Multiple gate failures shown together -- all failed gates displayed with individual recovery templates
- **D-03:** Three choices on Watcher Veto: (1) Stash changes, (2) Keep working, (3) Force advance
- **D-04:** Veto reason shown first (quality score + critical issues list) before presenting three choices
- **D-05:** "Stash changes" runs `git stash push` + creates blocker flag. "Keep working" does nothing. "Force advance" creates a FEEDBACK pheromone noting the override and proceeds.
- **D-06:** Gate results stored in COLONY_STATE.json as a new `gate_results` field
- **D-07:** Flat list schema: `[{name, passed, timestamp, detail}]`. Simple to read, easy to filter by `passed: false`.
- **D-08:** Results cleaned up when phase advances -- `gate_results` cleared on state advance
- **D-09:** Skip summary displayed at top ("Skipping 8 passed gates -- re-checking 3 failures")
- **D-10:** Tests always re-run regardless of prior pass status. Other passed gates are skipped if they passed last time.
- **D-11:** Gate results keyed by gate name -- on re-run, check for existing `passed: true` entries and skip them (except tests)

### Claude's Discretion
- Exact recovery template wording for each gate type
- Exact AskUserQuestion format for the veto confirmation (labels, descriptions)
- Whether `gate_results` lives at top level of COLONY_STATE.json or nested under current phase
- How to handle the edge case where gate_results exists but the phase changed (user inserted a new phase)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-01 | Verification gate failures show clear, actionable recovery instructions instead of just "FAILED" banner | `gateCheck` struct extended with `RecoveryTemplate`; per-gate templates in `cmd/gate.go`; playbook rendering updated |
| GATE-02 | Watcher Veto does not auto-stash work without explicit user confirmation | Step 1.13 in `continue-gates.md` rewritten to use `AskUserQuestion` with three choices instead of auto-stash |
| GATE-03 | Re-running `/ant-continue` only re-checks previously failed gates, not all gates from scratch | New `GateResults` field on `ColonyState`; `gateResultsRead`/`gateResultsWrite` helpers; each gate step checks prior result before running |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Gate result persistence | Go runtime (`pkg/colony/`, `pkg/storage/`) | -- | State lives in COLONY_STATE.json, managed by `UpdateJSONAtomically` |
| Recovery template rendering | Go runtime (`cmd/gate.go`) | Playbook layer (display) | Templates are data attached to `gateCheck` structs, rendered by both runtime and playbooks |
| Watcher Veto user choice | Playbook layer (`continue-gates.md`) | Go runtime (fallback) | `AskUserQuestion` is a wrapper-layer interaction; runtime provides the gate result data |
| Gate skip logic | Go runtime (`cmd/codex_continue.go`) | Playbook layer | Runtime decides which gates to skip; playbooks execute the skip decision |
| Phase advancement cleanup | Go runtime (`cmd/codex_continue.go`) | -- | State transition logic in `runCodexContinue` already clears fields on advance |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (encoding/json) | go1.26 | JSON marshaling for gate results | Already used throughout the codebase |
| cobra | current | CLI command framework | All Aether commands use cobra |
| pkg/storage | current | Atomic JSON read/write | `UpdateJSONAtomically` pattern for COLONY_STATE.json |
| pkg/colony | current | ColonyState type definitions | Where `GateResults` field gets added |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jq (via state-mutate) | any | Atomic state field updates | Alternative to `UpdateJSONAtomically` for simple field changes |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `gate_results` in COLONY_STATE.json | Separate `gate_results.json` file | Single-file approach matches existing pattern (CONTEXT.md D-06); separate file adds I/O but avoids COLONY_STATE bloat |

**Installation:**
No new dependencies. All changes use existing Go stdlib and internal packages.

## Architecture Patterns

### System Architecture Diagram

```
                    /ant-continue
                         |
                    ┌────▼─────┐
                    │ runCodex  │
                    │ Continue  │
                    └────┬─────┘
                         │
              ┌──────────▼──────────┐
              │ gateResultsRead()   │ ◄── loads prior results from COLONY_STATE.json
              │ Show skip summary   │     ("Skipping N passed gates...")
              └──────────┬──────────┘
                         │
              ┌──────────▼──────────────────────────────┐
              │ For each gate (1.5 through 1.14):       │
              │   if gateResults[name].passed == true   │
              │     && name != "tests_pass":            │
              │     SKIP (log + display)                 │
              │   else:                                  │
              │     RUN gate check                       │
              │     WRITE result to gateResults          │
              └──────────┬──────────────────────────────┘
                         │
              ┌──────────▼──────────┐
              │ Gate failed?        │
              │ YES → Show recovery │
              │   template + STOP   │
              │ NO  → Continue      │
              └──────────┬──────────┘
                         │
              ┌──────────▼──────────┐
              │ All gates pass?     │
              │ YES → Advance phase │
              │   + clear gateResults│
              └─────────────────────┘
```

### Recommended Project Structure
```
cmd/
├── gate.go                    ← Add GateResultEntry type, recovery templates, gateResultsRead/Write
├── codex_continue.go          ← Modify runCodexContinue to use gate results for skip logic
├── codex_continue_test.go     ← New tests for gate skip logic and recovery template rendering
├── review_depth.go            ← Unchanged (Phase 58 compatibility)

pkg/colony/
├── colony.go                  ← Add GateResults field to ColonyState

.aether/docs/command-playbooks/
├── continue-verify.md         ← Update "Gate Decision" to show recovery templates
├── continue-gates.md          ← Each step checks gate_results; Step 1.13 rewritten for user choice
```

### Pattern 1: Gate Results Persistence
**What:** Store per-gate pass/fail results in COLONY_STATE.json so re-runs can skip passed gates.
**When to use:** Every `/ant-continue` invocation reads prior results; every gate step writes results.
**Example:**
```go
// Source: Based on existing gateCheck struct in cmd/gate.go
type GateResultEntry struct {
    Name      string `json:"name"`
    Passed    bool   `json:"passed"`
    Timestamp string `json:"timestamp"`
    Detail    string `json:"detail,omitempty"`
}

// In pkg/colony/colony.go, add to ColonyState:
GateResults []GateResultEntry `json:"gate_results,omitempty"`
```

### Pattern 2: Recovery Template Attachment
**What:** Each gate type has a hardcoded recovery template with specific steps.
**When to use:** When a gate fails, its template is rendered alongside the failure banner.
**Example:**
```go
// Source: New pattern for cmd/gate.go
var gateRecoveryTemplates = map[string]string{
    "spawn_gate": "Spawn gate failed: Prime Worker completed without specialists.\n" +
        "1. Run `/ant-build {phase}` again\n" +
        "2. Ensure Prime Worker spawns at least 1 specialist (Builder or Watcher)\n" +
        "3. Re-run `/ant-continue` after spawns complete",
    "tests_pass": "Tests failed.\n" +
        "1. Run `go test ./...` to see which tests are failing\n" +
        "2. Fix the failing tests\n" +
        "3. Re-run `/ant-continue` to re-verify",
    // ... one per gate type
}
```

### Pattern 3: Incremental Gate Checking
**What:** On re-run, read prior gate results and skip gates that already passed (except tests).
**When to use:** At the start of `runCodexContinueGates()` and each playbook gate step.
**Example:**
```go
// Source: New pattern for cmd/codex_continue.go
func shouldSkipGate(priorResults []colony.GateResultEntry, gateName string) bool {
    if gateName == "tests_pass" {
        return false // tests always re-run (D-10)
    }
    for _, r := range priorResults {
        if r.Name == gateName && r.Passed {
            return true
        }
    }
    return false
}
```

### Anti-Patterns to Avoid
- **Storing gate results in a separate file when COLONY_STATE.json is the single state source:** The CONTEXT.md explicitly decides (D-06) that gate_results goes in COLONY_STATE.json. Don't create a parallel file.
- **Clearing gate_results too early:** Must clear on phase ADVANCE, not on gate pass. If all gates pass but something else blocks (review wave), gate_results should persist for the next continue attempt.
- **Skipping the runtime verification gate (Step 1.11) based on prior results:** This gate requires user input (AskUserQuestion). It cannot be meaningfully "skipped" because the user needs to re-confirm. However, per D-10, it IS in the skippable list. The planner should decide whether to skip it or always ask.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic COLONY_STATE.json updates | Manual read-modify-write with file locks | `store.UpdateJSONAtomically()` | Already handles locking, atomic rename, and error rollback |
| Gate name → recovery template mapping | Runtime string formatting per gate | `gateRecoveryTemplates` map with `{phase}` placeholders | Single source of truth, testable, consistent |
| Prior gate result lookup | Custom JSON parsing | `GateResultEntry` slice on `ColonyState` struct | Leverages existing JSON round-trip infrastructure |

**Key insight:** The codebase already has strong patterns for state mutation (`UpdateJSONAtomically`), gate checking (`gateCheck` struct), and user interaction (`AskUserQuestion` in playbooks). This phase extends these patterns, it does not introduce new ones.

## Common Pitfalls

### Pitfall 1: Gate Results Surviving Phase Insert
**What goes wrong:** User inserts a new phase (via `/ant-insert-phase`), shifting phase IDs. The `gate_results` from phase 59 now incorrectly apply to what was phase 60.
**Why it happens:** Gate results are keyed by gate name, not by phase ID, so they look valid for the new phase.
**How to avoid:** Store the phase ID alongside gate results, or clear `gate_results` whenever `current_phase` changes (not just on advance).
**Warning signs:** Re-running continue after a phase insert skips gates that haven't actually run for the new current phase.

### Pitfall 2: Watcher Veto Auto-Stash Still Firing
**What goes wrong:** The playbook layer (continue-gates.md) still contains the old auto-stash code, and someone runs the wrapper-based continue instead of the Go runtime.
**Why it happens:** The Go runtime and the playbook layer are two separate code paths. Changing one doesn't automatically change the other.
**How to avoid:** Update BOTH the Go runtime gate code AND the `continue-gates.md` playbook. Add a test that the runtime never auto-stashes.
**Warning signs:** User reports that running `/ant-continue` in Claude Code (wrapper path) still auto-stashes.

### Pitfall 3: Gate Results Growing Unboundedly
**What goes wrong:** Gate results accumulate across multiple continue attempts without cleanup, bloating COLONY_STATE.json.
**Why it happens:** Each continue run appends results instead of replacing them, or cleanup on advance is missing.
**How to avoid:** Gate results should be REPLACED on each continue run (not appended), and CLEARED on phase advance (D-08).
**Warning signs:** COLONY_STATE.json growing larger than expected after many failed continue attempts.

### Pitfall 4: Tests Gate Not Re-Running After Code Changes
**What goes wrong:** User fixes code but `tests_pass` gate is skipped because it previously passed.
**Why it happens:** D-10 says tests always re-run, but implementation accidentally treats `tests_pass` like other gates.
**How to avoid:** The `shouldSkipGate` function must have a hardcoded exception for `tests_pass`. This is the most critical skip rule to get right.
**Warning signs:** User reports that fixing code and re-running continue doesn't re-run tests.

## Code Examples

### Gate Result Entry Type
```go
// Source: New type for pkg/colony/colony.go
// Matches D-07 schema: flat list of {name, passed, timestamp, detail}
type GateResultEntry struct {
    Name      string `json:"name"`
    Passed    bool   `json:"passed"`
    Timestamp string `json:"timestamp"`
    Detail    string `json:"detail,omitempty"`
}
```

### ColonyState Extension
```go
// Source: Addition to ColonyState struct in pkg/colony/colony.go
// Line ~177 (before the closing brace)
// Uses omitempty per established project convention (STATE.md: "All new struct fields use omitempty")
GateResults []GateResultEntry `json:"gate_results,omitempty"`
```

### Gate Results Write Helper
```go
// Source: New function for cmd/gate.go
// Uses UpdateJSONAtomically for atomic COLONY_STATE.json mutation
func gateResultsWrite(entries []colony.GateResultEntry) error {
    var updated colony.ColonyState
    return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
        updated.GateResults = entries
        return nil
    })
}
```

### Gate Results Read Helper
```go
// Source: New function for cmd/gate.go
func gateResultsRead() []colony.GateResultEntry {
    var state colony.ColonyState
    if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
        return nil
    }
    if len(state.GateResults) == 0 {
        return nil
    }
    return state.GateResults
}
```

### Skip Logic with Test Exception
```go
// Source: Implements D-10 and D-11
func shouldSkipGate(priorResults []colony.GateResultEntry, gateName string) bool {
    // D-10: tests always re-run regardless of prior status
    if gateName == "tests_pass" {
        return false
    }
    for _, r := range priorResults {
        if r.Name == gateName && r.Passed {
            return true
        }
    }
    return false
}
```

### Clear Gate Results on Phase Advance
```go
// Source: Addition to the atomic state commit block in runCodexContinue()
// Around line 560 in cmd/codex_continue.go, inside the UpdateJSONAtomically callback:
updated.GateResults = nil // D-08: clear on phase advance
```

### Watcher Veto Three-Choice Pattern (Playbook Layer)
```xml
<!-- Source: Replacement for auto-stash in continue-gates.md Step 1.13 -->
Use AskUserQuestion:
Options:
1. "Stash changes and retry" - Runs `git stash push`, creates blocker flag, phase stays blocked
2. "Keep working (stay blocked)" - Does nothing, phase stays blocked, user fixes manually
3. "Force advance (accept risk)" - Creates FEEDBACK pheromone noting override, proceeds to Step 2

The veto reason is shown FIRST:
"Watcher VETO: Quality score {quality_score}/10 (minimum: 7), {critical_count} critical issue(s).
Critical Issues:
{list each critical issue with description}"

Then the three options are presented.
```

### Recovery Template Map
```go
// Source: New map for cmd/gate.go
// One entry per gate type from continue-gates.md Steps 1.5-1.14
var gateRecoveryTemplates = map[string]string{
    "verification_loop": "Verification commands failed.\n" +
        "1. Check the failed step output above for specific errors\n" +
        "2. Fix the build, type, lint, or test failures\n" +
        "3. Re-run `/ant-continue` to re-verify",
    "spawn_gate": "Spawn gate failed: Prime Worker completed without specialists.\n" +
        "1. Run `/ant-build {phase}` again\n" +
        "2. Prime Worker must spawn at least 1 specialist (Builder or Watcher)\n" +
        "3. Re-run `/ant-continue` after spawns complete",
    "anti_pattern": "Anti-pattern gate failed: Critical patterns detected.\n" +
        "1. Review the critical anti-patterns listed above\n" +
        "2. Fix each critical finding (exposed secrets, SQL injection, crash patterns)\n" +
        "3. Re-run `/ant-continue` to re-scan",
    "complexity": "Complexity gate failed: Code exceeds maintainability thresholds.\n" +
        "1. Review files exceeding 300 lines or 50-line functions\n" +
        "2. Refactor to reduce complexity\n" +
        "3. Re-run `/ant-continue` to re-check",
    "gatekeeper": "Gatekeeper gate failed: Critical CVEs detected.\n" +
        "1. Run `npm audit` (or equivalent) to see full details\n" +
        "2. Fix or update vulnerable dependencies\n" +
        "3. Re-run `/ant-continue` after resolving",
    "auditor": "Auditor gate failed: Critical quality issues or score below 60.\n" +
        "1. Review the critical findings listed above\n" +
        "2. Fix each critical finding first, then address high-severity items\n" +
        "3. Re-run `/ant-continue` to re-audit",
    "tdd_evidence": "TDD gate failed: Claimed tests not found in codebase.\n" +
        "1. Run `/ant-build {phase}` again\n" +
        "2. Actually write test files (not just claim them)\n" +
        "3. Tests must exist and be runnable",
    "runtime": "Runtime gate failed: User reported application issues.\n" +
        "1. Fix the reported runtime issues\n" +
        "2. Test the application manually\n" +
        "3. Re-run `/ant-continue` and confirm the app works",
    "flags": "Flags gate failed: Unresolved blocker flags.\n" +
        "1. Review each blocker flag listed above\n" +
        "2. Fix the issues and resolve flags: `/ant-flags --resolve {id} \"resolution\"`\n" +
        "3. Re-run `/ant-continue` after resolving all blockers",
    "watcher_veto": "Watcher VETO: Quality score below 7 or critical issues found.\n" +
        "1. Review the critical issues and quality score\n" +
        "2. Fix issues, then run `/ant-build {phase}` again\n" +
        "3. Watcher must re-verify with score >= 7 and no CRITICAL issues",
    "medic": "Medic gate failed: Critical colony health issues.\n" +
        "1. Review the critical health issues listed above\n" +
        "2. Run `aether medic --fix` to attempt repairs\n" +
        "3. Re-run `/ant-continue` after repairs",
    "tests_pass": "Tests failed.\n" +
        "1. Run `go test ./...` (or project test command) to see failures\n" +
        "2. Fix the failing tests\n" +
        "3. Re-run `/ant-continue` to re-verify",
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Auto-stash on Watcher Veto | Three-choice user prompt (this phase) | Phase 59 | User retains control over their work |
| Generic "FAILED" banner | Per-gate recovery templates | Phase 59 | User knows exactly what to do next |
| Re-run all gates from scratch | Incremental: skip passed, re-run failed | Phase 59 | Faster recovery after partial fixes |
| Gate results not persisted | Gate results in COLONY_STATE.json | Phase 59 | Cross-run state enables incremental checking |

**Deprecated/outdated:**
- Auto-stash in `continue-gates.md` Step 1.13 (lines 723-727): Must be replaced with the three-choice pattern

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 14 gates listed in CONTEXT.md are the complete set of gates in the continue flow | Specific Ideas | Missing gates would need their own recovery templates |
| A2 | The playbook layer (`continue-gates.md`) is the only place the auto-stash behavior exists | Pitfall 2 | If auto-stash exists elsewhere, it would still trigger silently |
| A3 | `AskUserQuestion` is available in the Claude Code wrapper context for the veto confirmation | GATE-02 | If not available, would need a Go-runtime fallback |
| A4 | Phase ID is sufficient to validate gate_results freshness (no need for a separate phase_changed check) | Discretion | If phase IDs shift without `current_phase` changing, stale results could be used |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **Runtime verification gate (Step 1.11) skip behavior**
   - What we know: D-10 says "Tests always re-run regardless of prior pass status." D-11 says gate results are keyed by name and passed gates are skipped. The runtime verification gate requires user input via AskUserQuestion.
   - What's unclear: Should the runtime verification gate be skipped on re-run if it previously passed? The user already confirmed the app works, but code may have changed since.
   - Recommendation: Treat it like other non-test gates (skip if previously passed). The user chose "Yes, tested and working" last time, and if code changed, tests would catch it.

2. **gate_results storage depth**
   - What we know: D-06 says store in COLONY_STATE.json. D-07 says flat list schema.
   - What's unclear: Whether gate_results should be scoped to the current phase (cleared when current_phase changes) or just cleared on advance.
   - Recommendation: Store flat at top level of ColonyState. Clear whenever phase advances OR when current_phase changes between continue runs. This is simpler and safer.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build and test | ✓ | go1.26.1 | -- |
| go test | Test execution | ✓ | go1.26.1 | -- |

**Missing dependencies with no fallback:**
None.

**Missing dependencies with fallback:**
None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (Go convention) |
| Quick run command | `go test ./cmd/... -run "TestGate" -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-01 | Recovery templates render for each gate type | unit | `go test ./cmd/... -run "TestGateRecoveryTemplate" -count=1` | Wave 0 |
| GATE-01 | Multiple failures shown with individual templates | unit | `go test ./cmd/... -run "TestMultipleGateRecoveryTemplates" -count=1` | Wave 0 |
| GATE-02 | Watcher Veto shows three choices instead of auto-stash | unit | `go test ./cmd/... -run "TestWatcherVetoChoices" -count=1` | Wave 0 |
| GATE-03 | Prior passed gates are skipped on re-run | unit | `go test ./cmd/... -run "TestShouldSkipGate" -count=1` | Wave 0 |
| GATE-03 | tests_pass gate never skipped | unit | `go test ./cmd/... -run "TestTestsNeverSkipped" -count=1` | Wave 0 |
| GATE-03 | gate_results cleared on phase advance | unit | `go test ./cmd/... -run "TestGateResultsClearedOnAdvance" -count=1` | Wave 0 |
| GATE-03 | gate_results persisted to COLONY_STATE.json | unit | `go test ./cmd/... -run "TestGateResultsPersistence" -count=1` | Wave 0 |
| GATE-03 | Skip summary displayed correctly | unit | `go test ./cmd/... -run "TestSkipSummary" -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run "TestGate|TestContinue" -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** `go test ./... -race -count=1`

### Wave 0 Gaps
- [ ] `cmd/gate_recovery_test.go` -- unit tests for recovery template rendering
- [ ] `cmd/gate_results_test.go` -- unit tests for gate results persistence and skip logic
- [ ] Extend `cmd/codex_continue_test.go` -- integration test for incremental continue with prior gate results

## Sources

### Primary (HIGH confidence)
- `cmd/gate.go` -- gateCheck struct, gate checking functions, runPreContinueGates [VERIFIED: codebase read]
- `cmd/codex_continue.go` -- runCodexContinue, runCodexContinueGates, runCodexContinueVerification, colony state management [VERIFIED: codebase read]
- `pkg/colony/colony.go` -- ColonyState struct definition [VERIFIED: codebase read]
- `pkg/storage/storage.go` -- UpdateJSONAtomically pattern [VERIFIED: codebase read]
- `.aether/docs/command-playbooks/continue-verify.md` -- 6-phase verification loop, gate decision logic [VERIFIED: codebase read]
- `.aether/docs/command-playbooks/continue-gates.md` -- Steps 1.6-1.14, all gate types [VERIFIED: codebase read]

### Secondary (MEDIUM confidence)
- `.planning/phases/58-smart-review-depth/58-CONTEXT.md` -- Review depth decisions that affect which gates run [CITED: CONTEXT.md reference]
- `cmd/review_depth.go` -- resolveReviewDepth, heavy keywords [VERIFIED: codebase read]
- `.planning/REQUIREMENTS.md` -- GATE-01, GATE-02, GATE-03 requirement definitions [VERIFIED: file read]

### Tertiary (LOW confidence)
None -- all findings are from direct codebase inspection.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies; all existing Go stdlib and internal packages
- Architecture: HIGH - extending existing well-structured patterns (gateCheck, UpdateJSONAtomically, AskUserQuestion)
- Pitfalls: HIGH - identified from direct code inspection of both runtime and playbook layers

**Research date:** 2026-04-27
**Valid until:** 2026-05-27 (stable domain -- Go patterns and colony state architecture change slowly)
