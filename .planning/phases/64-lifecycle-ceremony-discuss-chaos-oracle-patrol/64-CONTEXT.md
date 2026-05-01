# Phase 64: Lifecycle Ceremony -- Discuss, Chaos, Oracle, Patrol - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 64 enriches four lifecycle commands with ceremony-level intelligence. Discuss/council gets codebase-aware analysis that generates comprehensive questions before planning. Chaos auto-flags HIGH severity findings and suggests REDIRECT for recurring failure patterns. Oracle suggests persisting high-value research as pheromone signals or hive wisdom. Patrol graduates from a memory-details alias to a real health checker with JSON validation, stale signal detection, and interrupted build detection.

**What this phase delivers:**
- New `discuss-analyze` Go subcommand that scans codebase (file tree, tech stack, architecture patterns) and outputs structured suggested questions for wrappers to present
- Council shares the same discuss-analyze output with multi-position framing
- Chaos wrapper enhanced with instructions to auto-flag HIGH findings via `aether flag-add` and check midden recurrence for REDIRECT suggestions
- Oracle wrapper enhanced with post-completion instructions to suggest persisting high-value findings as pheromones or hive entries
- New `patrol-check` Go subcommand replacing memory-details alias: JSON validity check for COLONY_STATE/pheromones/session, stale pheromone detection, interrupted build detection

**What this phase does NOT deliver:**
- Idea shelving system (Phase 65)
- Seal, init, status, entomb, resume ceremony changes (Phases 62-63, complete)
- Changes to build/continue verification flow
- Cross-colony review sharing or federation

</domain>

<decisions>
## Implementation Decisions

### Discuss/Council Codebase Awareness
- **D-01:** New `discuss-analyze` Go subcommand performs inventory-level codebase scan: file tree structure, tech stack detection, test framework identification, architecture patterns. Outputs structured suggested questions as JSON. No source file reading — inventory only.
- **D-02:** Discuss wrapper calls `aether discuss-analyze` before asking questions, uses the structured output to formulate comprehensive multiple-choice questions covering features, priorities, scope, trade-offs, and architecture. Wrapper still owns question presentation and freeform handling.
- **D-03:** Council shares the same discuss-analyze output. Council wrapper adds multi-position framing on top of the same codebase data. No separate council-specific analysis.

### Chaos Auto-Flagging
- **D-04:** Chaos auto-flagging is wrapper-driven. The chaos.md wrapper gets instructions to run `aether flag-add` when it finds HIGH severity issues. Wrapper agent determines severity and creates flags. No new Go runtime code for chaos.
- **D-05:** Midden recurrence checking is wrapper-driven. Chaos wrapper reads recent midden entries, detects same category appearing 3+ times, and suggests a REDIRECT pheromone. Wrapper instructions only — no new Go subcommand.

### Oracle Persistence Suggestions
- **D-06:** Oracle persistence suggestions are wrapper-driven. After oracle loop completes, the oracle.md wrapper reads the research output and suggests persisting specific high-value findings as pheromone signals (`aether pheromone-write`) or hive wisdom entries (`aether hive-store`). User approves each suggestion.
- **D-07:** "High-value" is judged by the wrapper agent based on confidence, applicability, and actionability of the research finding. No deterministic threshold — the wrapper decides contextually.

### Patrol Health Checks
- **D-08:** New `patrol-check` Go subcommand replaces the memory-details alias. Runs three core health checks matching CERE-12 exactly: (1) JSON validity for COLONY_STATE.json, pheromones.json, session.json, (2) stale pheromone detection (signals referencing completed phases or zero-strength signals), (3) interrupted build detection (uncommitted manifests or spawn trees).
- **D-09:** Patrol-check outputs structured results (status per check: healthy/warning/error + details). Wrapper displays the report. Codex gets runtime-native output (no interaction needed).

### Claude's Discretion
- Exact file tree scan depth and exclusion patterns for discuss-analyze
- Number and format of suggested questions from discuss-analyze
- Specific wording of chaos wrapper auto-flagging instructions
- How oracle wrapper identifies "high-value" findings in practice
- Exact output format of patrol-check structured results
- Whether patrol-check subcommand is `patrol-check` or `patrol check` (subcommand vs sub-subcommand)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Discuss/Council Code
- `cmd/discuss.go` — Existing discuss command, clarification Q&A storage, pending-decisions
- `cmd/discuss_test.go` — Existing discuss tests
- `cmd/council.go` — Council deliberation, multi-position management
- `.claude/commands/ant/discuss.md` — Claude Code discuss wrapper
- `.opencode/commands/ant/discuss.md` — OpenCode discuss wrapper
- `.claude/commands/ant/council.md` — Claude Code council wrapper

### Init-Research Pattern (reference for discuss-analyze)
- `cmd/codex_workflow_cmds.go` — `init-research` subcommand pattern for codebase scanning
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` — Phase 62 init-research decisions (D-03 through D-06)

### Chaos Code
- `.claude/commands/ant/chaos.md` — Chaos wrapper (pure prompt command, no Go runtime)
- `.opencode/commands/ant/chaos.md` — OpenCode chaos wrapper
- `cmd/flag_cmds.go` — `flag-add` subcommand for flagging findings
- `cmd/midden_cmds.go` — `midden-recent-failures` for reading midden entries

### Oracle Code
- `cmd/oracle_loop.go` — Oracle loop with depth levels, state management, archiving
- `.claude/commands/ant/oracle.md` — Oracle wrapper
- `cmd/pheromone_cmds.go` — `pheromone-write` for persisting signals
- `cmd/hive_cmds.go` — `hive-store` for persisting wisdom

### Patrol Code
- `cmd/memory_details.go` — Current patrol alias (memory-details), to be replaced
- `cmd/codex_visuals.go` — `renderPatrolVisual()` for patrol display rendering
- `cmd/compatibility_cmds_test.go` — Existing tests for JSON validation patterns

### Lifecycle Pattern References
- `.planning/phases/63-lifecycle-ceremony-status-entomb-resume/63-CONTEXT.md` — Phase 63 decisions (wrapper-runtime contract, stale pheromone detection)
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` — Phase 62 seal ceremony pattern

### Requirements
- `.planning/REQUIREMENTS.md` — CERE-09 through CERE-12 requirements

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `init-research` subcommand in codex_workflow_cmds.go: existing codebase inventory scan pattern — discuss-analyze should follow same structure
- `midden-recent-failures` in midden_cmds.go: already queries recent midden by category — wrapper can call this directly
- `pheromone-write` in pheromone_cmds.go: already handles pheromone creation with dedup — oracle suggestions can call this
- `flag-add` in flag_cmds.go: already creates flags with severity — chaos can call this directly
- `renderPatrolVisual()` in codex_visuals.go: patrol display rendering already exists — new patrol-check output feeds into it

### Established Patterns
- Wrapper-runtime contract: Go runtime outputs structured data, wrappers handle interaction (established in Phases 62-63)
- Lifecycle ceremony pattern: each command gets ceremony-level behavior through a mix of runtime enrichment and wrapper instructions
- Inventory scan pattern: file tree walk + tech detection + pattern matching, no source reading (established in Phase 62 init-research)

### Integration Points
- `.claude/commands/ant/discuss.md` and `.opencode/commands/ant/discuss.md` — wrapper needs discuss-analyze call
- `.claude/commands/ant/chaos.md` and `.opencode/commands/ant/chaos.md` — wrapper needs flag-add and midden-check instructions
- `.claude/commands/ant/oracle.md` and `.opencode/commands/ant/oracle.md` — wrapper needs post-completion persistence instructions
- `.claude/commands/ant/patrol.md` and `.opencode/commands/ant/patrol.md` — wrapper needs patrol-check call replacing memory-details
- `cmd/memory_details.go` — patrol alias needs updating to call patrol-check instead

</code_context>

<specifics>
## Specific Ideas

- Discuss-analyze should feel like GSD's questioning pattern: 2-4 options per question, freeform allowed, covering all angles
- Chaos auto-flagging should feel automatic but transparent — user sees "Flagging HIGH finding: <title>" so they know what happened
- Oracle suggestions should appear after research completes as "I found 3 findings worth preserving" with tick-to-approve
- Patrol health check results should be visual and actionable: green checkmarks for healthy, red X with remediation for issues

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 64-lifecycle-ceremony-discuss-chaos-oracle-patrol*
*Context gathered: 2026-04-27*
