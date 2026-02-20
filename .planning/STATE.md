# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-20)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v2.0 Worker Emergence — Phase 28 (Orchestration Layer + Surveyor Variants)

## Current Position

Phase: 28 of 31 (Orchestration Layer + Surveyor Variants)
Plan: 2 of 4 in current phase (28-01, 28-02 complete)
Status: In progress
Last activity: 2026-02-20 — 28-02 complete: Scout (WebSearch/WebFetch, read-only) and Route-Setter (Task tool, planning discipline) agents created

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
- 28-02: Scout gets WebSearch/WebFetch but no Bash — read-only posture enforced via explicit tools field
- 28-02: Route-Setter Task tool documented with graceful degradation for subagent context where Task may be ineffective

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
Stopped at: Completed 28-02-PLAN.md (Scout and Route-Setter agents created)
Next step: Continue Phase 28 — proceed to 28-03
