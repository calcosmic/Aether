# Phase 81: Plan and Lifecycle Loop Safety - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Plans cannot contain circular task dependencies, and lifecycle commands always suggest a different recovery action than the command that just failed.

This phase covers LOOP-04 and LOOP-05 from REQUIREMENTS.md.

**What this phase delivers:**
- Cycle detection on plan task dependency graphs — if Task A depends on Task B and Task B depends on Task A, the plan is rejected
- Lifecycle commands (seal, entomb, status, resume) that on error show an interactive recovery menu with context-aware suggestions — never suggesting re-run of the same command

**What this phase does NOT deliver:**
- Loop detection telemetry (LOOP-06, Phase 82)
- Planning depth system (DEPTH-01, Phase 83)

</domain>

<decisions>
## Implementation Decisions

### Circular dependency detection (LOOP-04)
- **D-01:** Cycle detection runs at task level within the generated plan, not at phase level. Aether tracks `depends_on` on tasks, not phases — that's where cycles can actually occur.
- **D-02:** Detection uses a one-time cycle check (DFS with visited set) on the plan's task dependency graph after the plan is generated. No persistent graph in colony state — runs once per plan, rejects if cycle found.
- **D-03:** When a cycle is detected, the plan is rejected with a clear error identifying which tasks form the cycle. The AI is asked to regenerate the plan without the circular dependency.
- **D-04:** The cycle check runs as a validation step in the plan command flow, after the AI generates the plan structure but before it's committed to the build packet.

### Lifecycle command recovery (LOOP-05)
- **D-05:** When a lifecycle command (seal, entomb, status, resume) encounters an error, it displays an interactive recovery menu with numbered options the user can select from. No bare error without guidance.
- **D-06:** Recovery suggestions are generated dynamically — a recovery engine analyzes the error type and context to produce relevant next-step suggestions. Each suggestion MUST be a different command than the one that failed.
- **D-07:** The recovery engine uses error classification (file not found, permission denied, state corruption, missing prerequisite, etc.) to select relevant recovery actions. The mapping is defined per-command with fallback to generic suggestions.
- **D-08:** The recovery menu is rendered after the error message, with clear numbered options. User selects a number to proceed. This replaces the current behavior where lifecycle commands just print an error and exit.

### Claude's Discretion
- Exact error classification categories and their recovery mappings
- DFS vs other cycle detection algorithm (DFS is standard and sufficient)
- How the cycle rejection prompt is phrased to the AI

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Loop Prevention (Phase 80)
- `cmd/codex_continue.go` — Existing recovery command pattern: `continueNextCommandForBlocked`, `buildForceRedispatchCommand`, `loadLastContinueOptions`
- `pkg/colony/colony.go` — Phase struct with `WatcherFailureCount` field pattern

### Plan Command
- `cmd/codex_plan.go` — Plan generation, task structure with `DependsOn` field (line 97), plan template (line 1023)
- `cmd/codex_build.go` — Build manifest structure, `DependsOn` on dispatches (line 33), dependency writing (line 1429)

### Lifecycle Commands
- `cmd/session_cmds.go` — Session management, protected commands list (seal, entomb at line 66-68)
- `cmd/codex_visuals.go` — Lifecycle command emojis and banner rendering

### Requirements
- `.planning/REQUIREMENTS.md` — LOOP-04 (circular dependency prevention), LOOP-05 (lifecycle command retry safety)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `continueNextCommandForBlocked` pattern in `cmd/codex_continue.go` — already implements parameter comparison and fallback logic that can inform the lifecycle recovery engine design
- `codexContinueOptionsJSON` + `continueOptionsMatchCurrent` — parameter serialization and comparison functions from Phase 80

### Established Patterns
- Recovery commands use `buildForceRedispatchCommand` as fallback when no different parameters exist
- Error output uses `outputError(level, message, details)` pattern
- Interactive menus are not yet used in lifecycle commands — this would be new

### Integration Points
- Plan command flow: `cmd/codex_plan.go` — cycle check should run after plan structure is generated, before write to build packet
- Lifecycle commands: `cmd/session_cmds.go` — recovery menu logic should be added to the error handling paths of seal, entomb, status, resume
- The recovery engine could be a shared function in a new file (e.g., `cmd/recovery_engine.go`) used by all lifecycle commands

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for cycle detection and recovery menus.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 81-plan-and-lifecycle-loop-safety*
*Context gathered: 2026-04-30*
