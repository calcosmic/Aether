# Phase 30: Niche Agents - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Create all 8 niche agents as Claude Code subagents in `.claude/agents/ant/`, completing the full 22-agent roster. Agents: Chaos, Archaeologist, Ambassador, Chronicler, Gatekeeper, Includer, Measurer, Sage. Each must pass the existing AVA test suite (TEST-01 through TEST-04) and the final count test (TEST-05 updated to 22).

</domain>

<decisions>
## Implementation Decisions

### Plan Organization
- Group agents by similarity into 2-3 plans (Claude's discretion on exact grouping)
- All plans run in wave 1 (parallel) — agents are independent, no dependencies between them
- Separate verification plan that updates TEST-05 expected count from 14 to 22 and runs the full test suite
- Verification plan runs in wave 2 (after all agent plans complete)

### Read/Write Permissions
- Chaos agent: read-only (no Write/Edit). Analyzes code for edge cases but cannot modify anything
- Ambassador: full access as specced (Read, Write, Edit, Bash, Grep, Glob) — needs Bash for SDK installs and API calls
- Chronicler: Claude's discretion on whether to include Edit alongside Write
- Sage: Claude's discretion on whether to add Write for persisting analysis reports
- All other niche agents (Archaeologist, Gatekeeper, Includer, Measurer): read-only as specced in requirements

### Agent Triggers (Routing Descriptions)
- Archaeologist: primary value is **regression prevention** — excavates git history to find patterns of what was done before, ensures we're not repeating past mistakes or undoing previous fixes
- Gatekeeper: Claude's discretion on exact scope (dependencies only vs dependencies + import graphs)
- Includer: Claude's discretion on depth — assess based on typical project needs
- Measurer: Claude's discretion — scope for general profiling across project types, not just Aether-specific
- Each description must name a specific trigger case, not a generic role label (per success criteria)

### Agent Depth & Quality
- Equal depth across all 8 agents — no agent gets less attention than others
- Same 8-section XML template as Phases 28-29 (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries)
- Fresh designs — NOT mechanical ports from OpenCode definitions. Use the PWR template and design each agent to be best-in-class
- Same high quality bar as Phases 28-29: genuinely powerful agents you'd actually want to invoke, comparable to superpowers/everything-claude-code quality

### Claude's Discretion
- Exact plan grouping (how to batch the 8 agents into 2-3 plans)
- Chronicler Edit tool inclusion
- Sage Write tool inclusion
- Gatekeeper trigger scope
- Includer and Measurer execution flow depth
- Internal execution flow details for all agents

</decisions>

<specifics>
## Specific Ideas

- Archaeologist framed as a "regression guard" — its main job is making sure we don't go backwards on things we've already fixed
- OpenCode agents should eventually be synced with Claude Code versions, but that's future work — these agents are fresh designs first
- TEST-05 currently hardcoded to 22 but intentionally fails at 14 — the verification plan updates this and confirms all 22 pass

</specifics>

<deferred>
## Deferred Ideas

- OpenCode agent sync (keeping Claude Code and OpenCode agents in parity) — listed as future requirement, not this phase
- Agent A/B testing for routing effectiveness — future requirement
- Agent metrics/telemetry — future requirement

</deferred>

---

*Phase: 30-niche-agents*
*Context gathered: 2026-02-20*
