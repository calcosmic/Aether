# Phase 25: Medic Ant Core - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a Medic ant worker that diagnoses colony health across ALL Aether-related files and repairs issues without losing colony history or work. The Medic can be spawned manually (`/ant:medic`, `aether medic`) or automatically alongside other agents when the colony detects health issues.

**Scope:** Diagnosis + basic repair in Phase 25. Advanced repair and ceremony integrity in subsequent phases.
</domain>

<decisions>
## Implementation Decisions

### Medic Role and Dispatch
- **D-01:** Medic is a first-class caste/worker, not just a CLI command. It gets dispatched alongside other agents when colony health issues are detected during builds, continues, or status checks.
- **D-02:** Manual invocation: `/ant:medic` or `aether medic`. Auto-spawn: triggered by stale session, corrupted state, critical blocker, or build failure.
- **D-03:** Medic runs in its own worker context with the Medic skill loaded. It reads colony data but does not mutate state unless `--fix` is passed.

### Scan Scope (ALL Aether Files)
- **D-04:** Scan ALL `.aether/` files: data (COLONY_STATE, session, pheromones, constraints, trace), commands (YAML sources), agents, skills, templates
- **D-05:** Scan ALL wrapper files: `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.codex/agents/*.toml`
- **D-06:** Scan ALL runtime artifacts: `.aether/data/` (all JSON/JSONL files), `.aether/checkpoints/`, `.aether/locks/`, hive brain, eternal memory
- **D-07:** Validate cross-references between files (e.g., agents referenced in YAML exist, skills referenced are present)

### Strict Validation Rules
- **D-08:** JSON/JSONL must parse without errors. Corrupted files are CRITICAL.
- **D-09:** Required fields must be present per schema. Missing fields are CRITICAL.
- **D-10:** Version compatibility: detect old Aether file formats and flag for migration. Old versions are WARNING unless they break parsing.
- **D-11:** Colony state must be internally consistent: current phase matches plan, state transitions are valid, run_id is present if colony is active
- **D-12:** Wrapper/runtime parity: command counts match, agent counts match, skill counts match between surfaces
- **D-13:** Orphaned entries: worktree entries without matching git refs, pheromones with invalid types, agents with missing files

### Version Migration Handling
- **D-14:** Detect old Aether versions by checking `version` field in COLONY_STATE.json and `.aether/version.json`
- **D-15:** If old version detected, report migration path. Do NOT auto-migrate unless `--fix` + `--migrate` passed.
- **D-16:** Document known version incompatibilities in Medic skill file

### Repair Philosophy
- **D-17:** Preserve work and history where possible. Never delete colony state without backup.
- **D-18:** Before any repair, snapshot current state to `.aether/backups/medic-{timestamp}/`
- **D-19:** Repairs are logged to trace.jsonl with before/after state
- **D-20:** `--fix` flag required for mutations. Without it, Medic is read-only.
- **D-21:** `--fix` repairs in priority order: critical first (corrupted JSON), then warnings (stale entries), then info (missing optional files)
- **D-22:** If a repair would lose history (e.g., truncating corrupted JSON), warn user explicitly and require `--force` in addition to `--fix`

### Output Format
- **D-23:** Default: human-readable visual report with severity-colored output (like `status` command)
- **D-24:** `--json` flag: structured output for programmatic use (CI/CD, other agents)
- **D-25:** Report sections: Summary (counts), Critical Issues (must fix), Warnings (should fix), Info (observational), Repair Log (what was fixed)
- **D-26:** Exit codes: 0 = healthy, 1 = warnings only, 2 = critical issues found

### Integration with Colony System
- **D-27:** Medic findings are published to event bus as `medic.scan` events
- **D-28:** If critical issues found during auto-spawn, block colony advancement until addressed (or user overrides with `--ignore-health`)
- **D-29:** Medic worker gets its own caste identity: emoji 🩹, color, deterministic name

### Claude's Discretion
- How to prioritize repairs when multiple critical issues exist (planner decides order)
- Specific backup retention policy (how many medic backups to keep)
- Whether to integrate with `/ant:patrol` for continuous health monitoring
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Colony Data Schema
- `.aether/data/COLONY_STATE.json` — Colony state structure
- `.aether/data/session.json` — Session file format
- `.aether/data/pheromones.json` — Pheromone signal schema
- `.aether/data/constraints.json` — Focus areas and constraints
- `.aether/version.json` — Aether version tracking

### Runtime Commands
- `cmd/status.go` — Status command patterns (reuse dashboard rendering)
- `cmd/patrol.go` — Patrol command (potential integration point)
- `cmd/trace_cmds.go` — Trace commands (Medic writes to trace)

### Visual System
- `cmd/codex_visuals.go` — renderBanner, caste identity, stage markers
- `cmd/codex_visuals_test.go` — Visual output tests

### Wrapper Surfaces
- `.aether/commands/*.yaml` — YAML source definitions
- `.claude/commands/ant/*.md` — Claude wrappers
- `.opencode/commands/ant/*.md` — OpenCode wrappers
- `.codex/agents/*.toml` — Codex agents

### Architecture
- `CLAUDE.md` — Project instructions, version info
- `RUNTIME UPDATE ARCHITECTURE.md` — Distribution flow
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/status.go` — Dashboard rendering pattern, colony state loading
- `cmd/patrol.go` — System health check patterns
- `cmd/trace_cmds.go` — Trace file reading/writing
- `pkg/storage/storage.go` — JSONL append, file locking
- `pkg/colony/colony.go` — ColonyState struct, state machine

### Established Patterns
- Cobra CLI commands with visual output mode
- Event bus publishing for colony events
- `loadActiveColonyState()` pattern for state loading
- `renderBanner()` + visual divider for output formatting

### Integration Points
- Add `medicCmd` to `cmd/root.go` alongside other commands
- Medic worker integrates with agent pool dispatch system
- Health checks can be called from build/continue gates
</code_context>

<specifics>
## Specific Ideas

- Medic should be able to detect when a repo was last updated (via git HEAD + session timestamp) and warn if it's "stale"
- Old version detection: compare `.aether/version.json` against current release
- The "comprehensive review" colony that ran earlier (session_id: comprehensive_1776755691) is an example of the kind of audit the Medic should automate
- Medic backups should go in `.aether/backups/medic-{timestamp}/` with full colony data snapshot
</specifics>

<deferred>
## Deferred Ideas

- **Ceremony integrity checks** — verifying wrapper/runtime parity in detail (Phase 28)
- **Trace remote diagnostics** — analyzing exported trace from other repos (Phase 29)
- **Auto-spawn integration** — fully automatic Medic dispatch on health issues (Phase 30)
- **Continuous health monitoring** — integrating Medic into `/ant:patrol` or periodic checks

### Reviewed Todos (not folded)
- None
</deferred>

---

*Phase: 25-medic-ant-core*
*Context gathered: 2026-04-21*
