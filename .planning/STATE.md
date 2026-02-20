# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-20)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v2.0 Worker Emergence — Phase 29 (Specialist Agents + Agent Tests)

## Current Position

Phase: 29 of 31 (Specialist Agents + Agent Tests)
Plan: 3 of 3 in current phase (29-01, 29-02, 29-03 — 29-01 and 29-02 now complete)
Status: In progress
Last activity: 2026-02-20 — 29-01 complete: Keeper (knowledge management), Tracker (diagnose-only, no Write/Edit), and Auditor (strict read-only, no Write/Edit/Bash) specialist agents created

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
- 28-01: Queen gets Task tool unrestricted — true orchestrator; spawn_tree field removed from return (requires aether-utils.sh); flag-add replaced with structured text note
- 28-01: Caste emoji protocol established — 🔨🐜 Builder, 🔭🐜 Scout, 👁🐜 Watcher, 🗺🐜 Route-Setter/Surveyor
- 28-02: Scout gets WebSearch/WebFetch but no Bash — read-only posture enforced via explicit tools field
- 28-02: Route-Setter Task tool documented with graceful degradation for subagent context where Task may be ineffective
- 28-03: Surveyors get Write tool (not read-only) with scope restricted to .aether/data/survey/ only — locked decision from 28-CONTEXT.md overriding roadmap
- 28-03: OpenCode read_only section maps to boundaries section; consumption section embedded in execution_flow
- 29-02: Probe writes AND runs tests — untested tests are incomplete work; Bash available specifically for this purpose
- 29-02: Weaver revert protocol uses explicit git commands in failure_modes — behavior preservation enforced not merely documented
- 29-02: Changing test expectations to make tests pass is a behavior change, not a refactor — prohibited in Weaver critical_rules
- 29-01: Tracker boundary enforced at schema level — field is named suggested_fix (not fix_applied) to reinforce that Builder applies the fix
- 29-01: Auditor has no Bash — even for running linters; when Bash is needed for an audit dimension, Auditor returns blocked and routes to Builder or Tracker
- 29-01: Cross-reference escalation pattern established — specialists name the agent they route to (Tracker → Builder/Weaver, Auditor → Queen/Probe), not generic "escalate to orchestrator"

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
Stopped at: Completed 29-01-PLAN.md (Keeper, Tracker, Auditor specialist agents created)
Next step: Continue Phase 29 — proceed to 29-03 (agent quality tests AVA suite)
