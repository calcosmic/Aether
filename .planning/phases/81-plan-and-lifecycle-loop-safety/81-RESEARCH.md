# Phase 81: Plan and Lifecycle Loop Safety - Research

**Researched:** 2026-04-30
**Domain:** Graph cycle detection, command error recovery UX
**Confidence:** HIGH

## Summary

This phase delivers two features: (1) cycle detection on task dependency graphs during plan generation, and (2) a recovery menu system for lifecycle commands (seal, entomb, status, resume) that never suggests re-running the command that just failed.

Both features are self-contained additions to the existing Go runtime. Cycle detection plugs into the plan flow after `buildWorkerPlanPhases()` returns but before the plan is committed to `COLONY_STATE.json`. The recovery engine is a new shared module consumed by lifecycle command error paths, extending the existing `friendlyErrorForPattern` system with per-command recovery mappings that exclude the failed command.

**Primary recommendation:** Implement a `detectCycles()` function using three-color DFS in `pkg/colony/` (pure data layer, no I/O), and a `recovery_engine.go` in `cmd/` that wraps the existing `outputError` path with command-aware next-step suggestions.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Cycle detection runs at task level within the generated plan, not at phase level. Aether tracks `depends_on` on tasks, not phases -- that's where cycles can actually occur.
- **D-02:** Detection uses a one-time cycle check (DFS with visited set) on the plan's task dependency graph after the plan is generated. No persistent graph in colony state -- runs once per plan, rejects if cycle found.
- **D-03:** When a cycle is detected, the plan is rejected with a clear error identifying which tasks form the cycle. The AI is asked to regenerate the plan without the circular dependency.
- **D-04:** The cycle check runs as a validation step in the plan command flow, after the AI generates the plan structure but before it's committed to the build packet.
- **D-05:** When a lifecycle command (seal, entomb, status, resume) encounters an error, it displays an interactive recovery menu with numbered options the user can select from. No bare error without guidance.
- **D-06:** Recovery suggestions are generated dynamically -- a recovery engine analyzes the error type and context to produce relevant next-step suggestions. Each suggestion MUST be a different command than the one that failed.
- **D-07:** The recovery engine uses error classification (file not found, permission denied, state corruption, missing prerequisite, etc.) to select relevant recovery actions. The mapping is defined per-command with fallback to generic suggestions.
- **D-08:** The recovery menu is rendered after the error message, with clear numbered options. User selects a number to proceed. This replaces the current behavior where lifecycle commands just print an error and exit.

### Claude's Discretion
- Exact error classification categories and their recovery mappings
- DFS vs other cycle detection algorithm (DFS is standard and sufficient)
- How the cycle rejection prompt is phrased to the AI

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LOOP-04 | Plan circular dependency prevention -- detect and reject cycles in task `depends_on` graph | Three-color DFS on `colony.Task.DependsOn`, validation gate in plan flow |
| LOOP-05 | Lifecycle command retry safety -- error recovery suggestions must differ from the failed command | Recovery engine with per-command error classification and exclusion filter |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cycle detection algorithm | API / Backend (Go runtime) | -- | Pure graph algorithm on in-memory data structures; runs in the plan command flow |
| Plan validation gate | API / Backend (Go runtime) | -- | Rejects invalid plans before persisting to COLONY_STATE.json |
| Error classification | API / Backend (Go runtime) | -- | Pattern-matching on error messages, no external dependency |
| Recovery menu rendering | API / Backend (Go runtime) | -- | Terminal output via existing visual rendering system |
| Interactive selection | API / Backend (Go runtime) | -- | stdin prompt for numbered option selection |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.23+ (project Go version) | DFS algorithm, strings, fmt | No external graph library needed -- three-color DFS is ~40 lines |
| `pkg/colony` | current | Phase/Task types with DependsOn field | Cycle detection operates on existing data types |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cmd/codex_visuals.go` renderers | current | Banner, NextUp, visual output | Reuse `renderNextUp` and `renderBanner` for recovery menu |
| `cmd/helpers.go` outputError | current | Error output envelope | Recovery menu replaces/enhances this path for lifecycle commands |
| `cmd/ux_friendly_errors.go` | current | Error pattern matching | Extend with per-command recovery mappings |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Three-color DFS | Kahn's algorithm (topological sort) | Kahn's is slightly simpler but doesn't naturally report cycle paths; DFS with recursion stack produces the exact cycle chain for error messages |
| Per-command error map | Generic error suggestions | Generic suggestions can't exclude the failed command; per-command mapping is needed for LOOP-05 correctness |

**Installation:** No new dependencies needed. This is pure Go using existing project types and rendering.

**Version verification:** No external packages to verify.

## Architecture Patterns

### System Architecture Diagram

```
                    PLAN GENERATION FLOW
                    =====================

    Scout/RouteSetter workers
           |
           v
    buildWorkerPlanPhases()
           |
           v
    [colony.Phase[] with Tasks having DependsOn]
           |
           v
  +---- NEW: detectCycles(phases) ----+
  |                                    |
  |  Cycle found?                     |
  |    YES --> outputError(cycle)     |
  |    NO  --> continue               |
  +------------------------------------+
           |
           v
    Save COLONY_STATE.json
           |
           v
    Return plan result


              LIFECYCLE COMMAND ERROR FLOW
              ============================

    Command execution
           |
           v
    Error encountered?
           |
     NO --> Normal output
     YES  |
           v
  +---- NEW: recoveryEngine ----+
  |                              |
  |  1. Classify error type     |
  |  2. Look up command context |
  |  3. Filter out failed cmd   |
  |  4. Build recovery options  |
  |  5. Render numbered menu    |
  |  6. Prompt user selection   |
  +------------------------------+
           |
           v
    User selects option
           |
           v
    Print selected command
```

### Recommended Project Structure
```
pkg/colony/
└── cycle.go              # NEW: detectCycles(), CycleError type

cmd/
├── recovery_engine.go    # NEW: RecoveryEngine, error classification, menu rendering
├── codex_plan.go         # MODIFY: add cycle validation gate (~line 350)
├── codex_workflow_cmds.go # MODIFY: seal command error path uses recovery engine
├── entomb_cmd.go         # MODIFY: entomb command error path uses recovery engine
├── status.go             # MODIFY: status command error path uses recovery engine
├── session_flow_cmds.go  # MODIFY: resume-colony error path uses recovery engine
└── ux_friendly_errors.go # MODIFY: extend with lifecycle error patterns (optional)
```

### Pattern 1: Three-Color DFS Cycle Detection
**What:** Classic graph cycle detection using WHITE/GRAY/BLACK node coloring. A back-edge to a GRAY node means a cycle.
**When to use:** Any time you need to detect cycles in a directed graph and report the exact cycle path.
**Why this project:** The `DependsOn` field on tasks forms a directed graph. Task IDs (e.g., "1.2", "3.1") serve as node identifiers.

```go
// Source: Standard algorithm, verified against project data structures [VERIFIED: pkg/colony/colony.go lines 322-341]
func detectCycles(phases []colony.Phase) error {
    const (
        white = 0 // unvisited
        gray  = 1 // in current DFS path
        black = 2 // fully explored
    )

    // Build adjacency list: taskID -> depends_on taskIDs
    adj := make(map[string][]string)
    for _, phase := range phases {
        for _, task := range phase.Tasks {
            if task.ID == nil {
                continue
            }
            for _, dep := range task.DependsOn {
                adj[*task.ID] = append(adj[*task.ID], dep)
            }
        }
    }

    color := make(map[string]int)
    var path []string

    var dfs func(node string) error
    dfs = func(node string) error {
        color[node] = gray
        path = append(path, node)

        for _, neighbor := range adj[node] {
            if color[neighbor] == gray {
                // Found cycle -- extract it
                cycleStart := -1
                for i, n := range path {
                    if n == neighbor {
                        cycleStart = i
                        break
                    }
                }
                cycle := append(path[cycleStart:], neighbor)
                return &CycleError{Tasks: cycle}
            }
            if color[neighbor] == white {
                if err := dfs(neighbor); err != nil {
                    return err
                }
            }
        }

        path = path[:len(path)-1]
        color[node] = black
        return nil
    }

    for _, phase := range phases {
        for _, task := range phase.Tasks {
            if task.ID == nil {
                continue
            }
            if color[*task.ID] == white {
                if err := dfs(*task.ID); err != nil {
                    return err
                }
            }
        }
    }
    return nil
}
```

### Pattern 2: Recovery Engine with Command Exclusion
**What:** A function that takes the failed command name, error message, and optional colony state, and returns a list of recovery suggestions -- none of which are the failed command.
**When to use:** In lifecycle command error paths (seal, entomb, status, resume).
**Why this project:** Extends the existing `friendlyErrorForPattern` system with per-command awareness.

```go
// Source: Designed from existing patterns in cmd/codex_continue.go (lines 1704-1735) [VERIFIED: cmd/codex_continue.go]
type RecoveryOption struct {
    Label    string // e.g., "Check colony health"
    Command  string // e.g., "aether status"
    Rationale string // e.g., "Diagnostics may reveal the root cause"
}

func recoveryOptionsForCommand(failedCmd string, errMsg string) []RecoveryOption {
    classified := classifyError(errMsg)
    candidates := recoveryCandidates(failedCmd, classified)
    // Filter: exclude the failed command from all suggestions
    var result []RecoveryOption
    for _, c := range candidates {
        if normalizeCmd(c.Command) == normalizeCmd(failedCmd) {
            continue // LOOP-05: never suggest the same command
        }
        result = append(result, c)
    }
    if len(result) == 0 {
        result = fallbackRecoveryOptions(failedCmd)
    }
    return result
}
```

### Anti-Patterns to Avoid
- **Topological sort without cycle reporting:** Kahn's algorithm can detect cycles but doesn't naturally produce the cycle path. Users need to know WHICH tasks form the cycle to fix it.
- **Global error recovery map without command context:** A generic "next steps" list can accidentally suggest the same command that failed (violates LOOP-05). Always filter by failed command.
- **Cycle check before plan generation:** The cycle check must run AFTER the plan is generated (D-04), not during task creation, because the AI planner needs the full plan to understand the cycle.
- **Recovery menu with only one option:** If the engine can only produce one suggestion, it should still present it as a menu item. Never fall back to bare error output.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Graph cycle detection | Custom BFS with bookkeeping | Three-color DFS | Standard algorithm, naturally produces cycle path for error messages |
| Error pattern matching | Custom regex chain | Extend existing `friendlyErrorForPattern` | Already has case-insensitive substring matching with explanation+next steps |
| Visual rendering for menu | Custom ANSI formatting | `renderBanner`, `renderNextUp` from `codex_visuals.go` | Consistent with existing UX, already handles platform detection |

**Key insight:** The project already has error pattern matching (`ux_friendly_errors.go`), visual rendering (`codex_visuals.go`), and recovery command logic (`codex_continue.go` lines 1704-1813). The new code extends these patterns, not replaces them.

## Common Pitfalls

### Pitfall 1: DependsOn references across phases
**What goes wrong:** Task "2.3" depends on task "1.5" (cross-phase dependency). The DFS treats these as valid edges, but the cycle detection scope may be wrong.
**Why it happens:** Aether's task IDs encode phase number (e.g., "2.3" = phase 2, task 3). Cross-phase dependencies are valid and should be checked for cycles just like same-phase dependencies.
**How to avoid:** Build the adjacency list from ALL phases' tasks, not per-phase. The DFS must operate on the full task graph.
**Warning signs:** Cycle detection passes but only checks within-phase dependencies, missing A->B->C->A chains that span phases.

### Pitfall 2: Nil task IDs in dependency references
**What goes wrong:** A task has `DependsOn: ["3.1"]` but task "3.1" doesn't exist (AI hallucination). The DFS would treat it as a dangling edge.
**Why it happens:** The AI planner can generate dependency references that don't match any actual task ID.
**How to avoid:** Before cycle detection, validate that all `DependsOn` values reference existing task IDs. Report missing references as a separate error (not a cycle).
**Warning signs:** Cycle detection runs without errors but the plan has tasks referencing non-existent dependencies.

### Pitfall 3: Recovery menu blocking CI/non-interactive environments
**What goes wrong:** The recovery menu tries to read from stdin, but in a CI pipeline or non-interactive terminal, it hangs.
**Why it happens:** The current codebase already uses `shouldRenderVisualOutput()` to gate visual rendering. The recovery menu needs the same guard.
**How to avoid:** Only render interactive recovery menu when `shouldRenderVisualOutput(stdout)` is true. In JSON output mode, include recovery suggestions in the error envelope's `details` field.
**Warning signs:** Tests hang when lifecycle commands error out.

### Pitfall 4: Recovery suggestions that are technically different but effectively the same
**What goes wrong:** Seal fails, and the recovery menu suggests `aether seal --force`. The command string differs but the user still runs seal.
**Why it happens:** Simple string comparison misses flag variations of the same base command.
**How to avoid:** Normalize commands before comparison -- strip flags, compare base command names (e.g., "aether seal" == "aether seal --force").
**Warning signs:** User clicks a recovery option and immediately hits the same error.

## Code Examples

### Integration point: Plan cycle validation (codex_plan.go ~line 350)

```go
// Source: Derived from plan flow in cmd/codex_plan.go [VERIFIED: cmd/codex_plan.go lines 300-368]
// After phases are built but before saving to state:

// Validate task dependency graph for cycles
if err := detectCycles(phases); err != nil {
    var cycleErr *colony.CycleError
    if errors.As(err, &cycleErr) {
        return nil, fmt.Errorf("plan contains circular dependency: %s. Remove the cycle and regenerate the plan", cycleErr)
    }
    return nil, fmt.Errorf("plan dependency validation failed: %w", err)
}
```

### Integration point: Recovery menu in lifecycle command error path

```go
// Source: Derived from error handling in cmd/codex_workflow_cmds.go seal command [VERIFIED: cmd/codex_workflow_cmds.go lines 306-326]
// Replace bare outputError calls in lifecycle commands:

if len(state.Plan.Phases) == 0 {
    // OLD: outputError(1, "No project plan. Run `aether plan` first.", nil)
    // NEW:
    renderRecoveryMenu("seal", "No project plan. Run `aether plan` first.", nil)
    return nil
}
```

### Existing recovery command pattern from Phase 80

```go
// Source: cmd/codex_continue.go lines 1712-1735 [VERIFIED: cmd/codex_continue.go]
func continueNextCommandForBlocked(assessment codexContinueAssessment, blockers []string, options codexContinueOptions, phaseID int) string {
    lastOptions := loadLastContinueOptions(phaseID)

    if continueBlockersContainVerificationTimeout(blockers) {
        cmd := buildContinueVerificationTimeoutRecoveryCommand(options, lastOptions)
        if continueOptionsMatchCurrent(options, lastOptions) {
            return buildForceRedispatchCommand(phaseID)
        }
        return cmd
    }
    // ... pattern: always provide a DIFFERENT command than continue
    next := strings.TrimSpace(continueNextCommandForAssessment(assessment))
    if next == "aether continue" && len(blockers) > 0 {
        return "" // Don't suggest looping back to continue with blockers
    }
    return next
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bare error messages with generic hint | Per-command recovery menu with exclusion filter | Phase 81 | Lifecycle commands never suggest re-running themselves |
| No plan validation | Post-generation cycle detection with DFS | Phase 81 | Plans with circular dependencies are rejected before persisting |

**Deprecated/outdated:**
- None -- this is new functionality being added.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Task IDs are unique across all phases (format "X.Y") | Pattern 1 | Low risk -- `buildWorkerPlanPhases` generates them as `fmt.Sprintf("%d.%d", i+1, j+1)` [VERIFIED: codex_plan.go line 1083] |
| A2 | `DependsOn` values reference task IDs in the same format | Pattern 1 | Medium risk -- AI-generated plans may use different formats; validation step needed |
| A3 | Lifecycle commands use `outputError` for all error paths | Pattern 2 | Low risk -- verified seal, entomb, status, resume all use `outputError` or `outputErrorMessage` [VERIFIED: cmd/codex_workflow_cmds.go, cmd/entomb_cmd.go, cmd/status.go, cmd/session_flow_cmds.go] |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **Should cycle detection also validate that DependsOn references exist?**
   - What we know: A task can reference a non-existent task ID in DependsOn (AI hallucination).
   - What's unclear: Whether LOOP-04 requires existence validation or only cycle detection.
   - Recommendation: Include existence validation in the same function. Report missing references as a separate error type. This prevents confusing cycle detection results when a dependency points to a non-existent task.

2. **Should the recovery menu actually execute the selected command?**
   - What we know: D-08 says "User selects a number to proceed."
   - What's unclear: Does "proceed" mean the CLI runs the command automatically, or just prints it for the user to copy?
   - Recommendation: Print the command for the user to copy. Auto-execution would change the command's control flow significantly and could have unexpected side effects. The Go CLI is a subprocess of the LLM agent, not a REPL.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- this phase is pure Go code using existing project infrastructure)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify (assert/require) |
| Config file | None -- standard Go test conventions |
| Quick run command | `go test ./cmd/ -run 'TestCycle|TestRecovery' -count=1` |
| Full suite command | `go test ./... -count=1 -timeout 120s` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LOOP-04 | No cycle in valid plan | unit | `go test ./pkg/colony/ -run 'TestDetectCycles_NoCycle' -count=1` | No -- Wave 0 |
| LOOP-04 | Cycle in A->B->A detected | unit | `go test ./pkg/colony/ -run 'TestDetectCycles_SimpleCycle' -count=1` | No -- Wave 0 |
| LOOP-04 | Cycle in A->B->C->A detected | unit | `go test ./pkg/colony/ -run 'TestDetectCycles_LongCycle' -count=1` | No -- Wave 0 |
| LOOP-04 | Cross-phase cycle detected | unit | `go test ./pkg/colony/ -run 'TestDetectCycles_CrossPhase' -count=1` | No -- Wave 0 |
| LOOP-04 | Missing dependency reference reported | unit | `go test ./pkg/colony/ -run 'TestDetectCycles_MissingDep' -count=1` | No -- Wave 0 |
| LOOP-04 | Plan rejected when cycle found | unit | `go test ./cmd/ -run 'TestPlanRejectsCyclicDependency' -count=1` | No -- Wave 0 |
| LOOP-05 | Recovery excludes failed command | unit | `go test ./cmd/ -run 'TestRecoveryExcludesFailedCommand' -count=1` | No -- Wave 0 |
| LOOP-05 | Recovery provides different command for each lifecycle cmd | unit | `go test ./cmd/ -run 'TestRecoveryForLifecycle' -count=1` | No -- Wave 0 |
| LOOP-05 | Recovery menu renders with numbered options | unit | `go test ./cmd/ -run 'TestRecoveryMenuRender' -count=1` | No -- Wave 0 |
| LOOP-05 | Recovery handles command flag variants | unit | `go test ./cmd/ -run 'TestRecoveryNormalizesCommand' -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/colony/ -run 'TestCycle' -count=1 && go test ./cmd/ -run 'TestRecovery' -count=1`
- **Per wave merge:** `go test ./... -count=1 -timeout 120s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/colony/cycle_test.go` -- cycle detection unit tests
- [ ] `cmd/recovery_engine_test.go` -- recovery engine unit tests
- [ ] `pkg/colony/cycle.go` -- CycleError type and detectCycles function

Framework install: not needed (stdlib + existing testify dependency)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | yes | Validate DependsOn values are well-formed task IDs before cycle detection |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious DependsOn values (command injection) | Tampering | Task IDs are validated as "X.Y" format before use; never interpolated into shell commands |
| stdin reading in recovery menu | Tampering | Only read a single integer; no free-form input processed |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: pkg/colony/colony.go lines 322-341] -- Phase and Task struct definitions with DependsOn field
- [VERIFIED: cmd/codex_plan.go lines 300-368] -- Plan flow where cycle validation gate inserts
- [VERIFIED: cmd/codex_plan.go line 1083] -- Task ID format generation
- [VERIFIED: cmd/codex_continue.go lines 1704-1813] -- Existing recovery command patterns from Phase 80
- [VERIFIED: cmd/ux_friendly_errors.go] -- Existing error pattern matching system
- [VERIFIED: cmd/codex_workflow_cmds.go lines 301-326] -- Seal command error handling
- [VERIFIED: cmd/entomb_cmd.go lines 20-52] -- Entomb command error handling
- [VERIFIED: cmd/status.go lines 18-37] -- Status command error handling
- [VERIFIED: cmd/session_flow_cmds.go lines 163-167] -- Resume-colony command definition
- [VERIFIED: cmd/codex_visuals.go lines 304-322] -- renderNextUp helper for recovery suggestions

### Secondary (MEDIUM confidence)
- [VERIFIED: Go test suite] -- 2361+ passing tests, all plan-related tests pass

### Tertiary (LOW confidence)
- None -- all findings verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies, all existing project code
- Architecture: HIGH - Integration points verified in source code, plan flow and error paths confirmed
- Pitfalls: HIGH - Cross-phase dependencies, nil task IDs, non-interactive environments all verified as real risks from codebase inspection

**Research date:** 2026-04-30
**Valid until:** 60 days (stable domain -- graph algorithms and CLI error handling don't change rapidly)
