# Phase 62: Lifecycle Ceremony -- Seal and Init - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 62 makes seal and init into real ceremonies. Seal blocks on active blockers, promotes wisdom selectively, cleans pheromones, and enriches the CROWNED-ANTHILL.md archive. Init performs a deep codebase scan, generates a charter with pheromone suggestions, and presents both for user approval before creating the colony.

**What this phase delivers:**
- Seal checks blocker-severity flags and hard-stops with a summary table + resolution commands
- Seal auto-promotes instincts to repo QUEEN.md (local only); global QUEEN.md and Hive Brain stay manual
- Seal expires all FOCUS pheromones while preserving REDIRECT pheromones
- CROWNED-ANTHILL.md gets enriched with a summary table + detail lists (instinct names, signal types, flags resolved)
- Init-research does a deep scan: recursive directory walk, git history, governance detection (linters, CI, test configs), pheromone suggestions from deterministic patterns, complexity metrics
- Init ceremony includes charter output and pheromone tick-to-approve — user sees a founding document and approves signals before the colony is created
- Runtime handles scanning and data; wrappers handle approval interaction

**What this phase does NOT deliver:**
- Suggest-analyze during builds (Phase 64 territory)
- Bayesian confidence parity verification (separate investigation)
- Status, entomb, resume ceremony changes (Phase 63)
- Idea shelving (Phase 65)

</domain>

<decisions>
## Implementation Decisions

### Seal Blocker UX
- **D-01:** Seal hard-stops when blocker-severity flags exist — prints a table of all blockers (title, description, age) with suggested resolution commands, then exits. No interactive prompt. User must resolve flags or use `--force` override.
- **D-02:** The blocker summary includes actionable resolution suggestions per flag (e.g., `aether flag <id> --resolve` or `--force to override`).

### Init-Research Depth
- **D-03:** Init-research performs a deep recursive directory walk (skip `.git`, `node_modules`, `vendor`), reads main entry point + top 5 largest source files, detects test frameworks from config files, checks CI configs (`.github/workflows/*.yml`), and reports architecture patterns.
- **D-04:** Init-research restores lost richness from the shell-era scan: git history analysis (commit count, contributors), colony context (prior colonies from chambers), governance detection (linters, CI, test configs → rules), pheromone suggestions from 10 deterministic patterns, and complexity metrics.
- **D-05:** Init ceremony includes full charter flow: deep scan generates charter text with Intent/Vision/Governance/Goals, pheromone suggestions run as tick-to-approve during init (not build), user approves both charter and signals before colony creation.

### Init Ceremony Architecture
- **D-06:** Go runtime (`aether init-research`) does the scanning and outputs structured data (charter + pheromone suggestions). Platform wrappers (`.claude/commands/ant/init.md`, `.opencode/commands/ant/init.md`) handle the interactive approval and tick-to-approve UX. This keeps runtime stateless and portable.

### CROWNED-ANTHILL Enrichment
- **D-07:** CROWNED-ANTHILL.md gets a summary statistics table (counts: learnings captured, instincts promoted, signals expired, flags resolved) followed by detail lists showing instinct names promoted and signal types expired. Both machine-parseable and human-readable.

### Promotion Ordering
- **D-08:** Seal auto-promotes instincts >= 0.8 confidence to repo QUEEN.md only (local). Global QUEEN.md (`~/.aether/QUEEN.md`) and Hive Brain promotion stay manual — user must explicitly run `aether queen-promote` or `aether hive-promote`. Seal logs eligible instincts: "3 instincts eligible for global promotion" as a suggestion, not an action.
- **D-09:** This means CERE-02 (hive-promote at seal) is re-scoped: seal logs suggestions for hive promotion but does not auto-execute it. The `hive-promote` subcommand exists and works — it just doesn't auto-fire at seal.

### Claude's Discretion
- Exact phrasing of blocker summary table (column order, widths)
- Number of deterministic pheromone patterns (user said "10" from old system, but planner can adjust based on what makes sense for Go)
- Charter markdown format and section structure
- Which "top 5 largest source files" to read (planner decides heuristic)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Seal ceremony
- `cmd/codex_workflow_cmds.go` -- sealCmd definition (line 241), buildSealSummary() (line 639)
- `cmd/flags.go` -- flag system with severity levels, blocker detection
- `cmd/flag_cmds.go` -- flag CRUD commands
- `cmd/hive.go` -- hive-promote subcommand
- `cmd/porter_cmd.go` -- buildPorterReadinessSummary (seal post-step)

### Init ceremony
- `cmd/init_cmd.go` -- init command, colony creation flow
- `cmd/init_research.go` -- current init-research implementation (stub, ~120 lines)
- `cmd/init_research_test.go` -- existing tests
- `.claude/commands/ant/init.md` -- Claude Code init wrapper (handles ceremony UX)
- `.opencode/commands/ant/init.md` -- OpenCode init wrapper

### Pheromone system
- `cmd/pheromone_write.go` -- pheromone CRUD, expiry logic
- `cmd/pheromone_lifecycle_test.go` -- lifecycle tests
- `cmd/pheromone_dedup_test.go` -- dedup tests

### Cross-references
- `.planning/REQUIREMENTS.md` -- CERE-01 through CERE-05 definitions
- `CLAUDE.md` -- UX Architecture section (wrapper-runtime contract)
- `.aether/docs/wrapper-runtime-ux-contract.md` -- full wrapper-runtime contract
- `.aether/docs/command-playbooks/` -- build and continue playbooks for pheromone integration patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `sealCmd` in `cmd/codex_workflow_cmds.go:241`: Clean foundation for blocker checking insertion point. Currently checks phase completion, writes CROWNED-ANTHILL.md, emits ceremony event.
- `init-research` in `cmd/init_research.go`: Working stub with project detection and framework identification. Extendable for deeper scanning.
- `flags` system in `cmd/flags.go`: Has severity levels already. Blocker detection reads from `pending-decisions.json` or `flags.json`.
- `hive-promote` in `cmd/hive.go`: Full subcommand with dedup, merge, 200-cap enforcement. Ready to be called from seal.
- `pheromone-write` in `cmd/pheromone_write.go`: Has pheromone type handling (FOCUS/REDIRECT). Expiry logic can be extended.
- `buildSealSummary()` in `cmd/codex_workflow_cmds.go:639`: Generates CROWNED-ANTHILL.md. Extendable for enrichment.

### Established Patterns
- State mutations via `store.SaveJSON()` pattern throughout cmd/
- Ceremony emission via `emitLifecycleCeremony()` for lifecycle events
- Wrapper-runtime contract: Go outputs JSON, wrappers handle interaction
- Flag severity already supports different levels (blocker vs issue vs info)

### Integration Points
- Seal flow: `sealCmd` → blocker check → promotion → pheromone cleanup → CROWNED-ANTHILL.md → Porter readiness
- Init flow: wrapper calls `init-research` → reads charter + suggestions → user approves → wrapper calls `init` with goal
- Pheromone system: `pheromone-write` with type filtering for FOCUS vs REDIRECT expiry
- Registry: colony registration in `~/.aether/registry/` for cross-colony context

</code_context>

<specifics>
## Specific Ideas

- The old shell-era init had a charter with Intent, Vision, Governance, Goals sections. User wants this founding document restored — not just a JSON file but something the user reads and approves.
- Old suggest-analyze had 10 deterministic patterns (e.g., .env files → REDIRECT "never commit secrets", no CI → FOCUS "add CI"). These should be ported to Go as part of init-research.
- User specifically wants init to feel like a ceremony again, not just a Go command that writes files.

</specifics>

<deferred>
## Deferred Ideas

- Suggest-analyze during builds (Phase 64 — CERE-10 auto-flagging covers this)
- Bayesian confidence scoring parity verification (separate investigation, not Phase 62)
- Global QUEEN.md auto-promotion at seal (user wants this manual — may revisit if promotion noise is low)

</deferred>

---

*Phase: 62-lifecycle-ceremony-seal-and-init*
*Context gathered: 2026-04-27*
