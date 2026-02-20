# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-20)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v2.0 Worker Emergence — Phase 27 (Distribution Infrastructure + First Core Agents)

## Current Position

Phase: 27 of 31 (Distribution Infrastructure + First Core Agents)
Plan: 4 of 4 in current phase (27-01, 27-02, 27-03, 27-04 complete)
Status: Phase 27 complete
Last activity: 2026-02-20 — 27-04 complete: distribution chain verified end-to-end (npm pack, hub population, idempotency, 415 tests passing)

Progress: [████████░░] 77% (26/31 phases complete through v1.4)

## Performance Metrics

**Cumulative:**
- Total plans completed: 72 (v1.0: 27, v1.1: 13, v1.2: 18, v1.3: 12, v1.4: 2)
- Total requirements validated: 109 (v1.0: 46, v1.1: 14, v1.2: 24, v1.3: 24, v1.4: 1 partial)
- Total tests: 446 passing (415 AVA + 31 bash), 0 failures

## Accumulated Context

### Decisions
- 27-01: init.js hub paths fixed — HUB_COMMANDS_CLAUDE/OPENCODE/AGENTS now use HUB_SYSTEM not HUB_DIR (v4.0 structure)
- v2.0 is a major version bump — first time Claude Code gets real agents
- Agents go in `.claude/agents/ant/` subdirectory (user decision — keeps GSD agents separate)
- Hub path: `~/.aether/system/agents-claude/` (separate from OpenCode `agents/` to prevent cross-contamination)
- Agent format: YAML frontmatter + XML body, matching GSD pattern in `.claude/agents/`
- Distribution must be proven in Phase 27 — agents that only work in source repo are not shipped
- PWR-01 through PWR-08 (agent power standards) established in Phase 27 as the conversion checklist template
- v1.4 phases 27-30 absorbed into v2.0 as Phase 31 cleanup
- Description must be quoted in YAML frontmatter (contains colons — unquoted breaks YAML parse)
- Escalation section replaces Spawning Sub-Workers — Claude Code subagents cannot spawn other subagents
- 8 XML sections (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries) define the conversion template for all remaining agents
- 27-03: Watcher has read-only tools (Read, Bash, Grep, Glob — no Write/Edit); explicit tools field enforces this; spawns field removed from return format
- 27-04: Distribution pipeline proven — npm pack includes ant agents only (not GSD agents), hub populated at ~/.aether/system/agents-claude/, second install idempotent

### Key Findings from Research
- Subagents cannot spawn other subagents — strip all spawn calls from every converted agent
- Tool inheritance over-permissions agents — explicit tools field required on every agent, no exceptions
- YAML malformation silently drops agents — run `/agents` after every file creation to confirm load
- Vague descriptions kill auto-routing — write as routing triggers, not role labels
- Surveyor XML format is the most mature agent format in Aether — port directly

### Blockers / Concerns
- GSD agent distribution concern resolved: `.claude/agents/ant/` subdirectory keeps Aether agents separate
- Model field behavior in frontmatter unverified — test during Phase 27 implementation

## Session Continuity

Last session: 2026-02-20
Stopped at: Completed 27-04-PLAN.md (distribution chain verified end-to-end)
Next step: Phase 27 complete — proceed to next phase
