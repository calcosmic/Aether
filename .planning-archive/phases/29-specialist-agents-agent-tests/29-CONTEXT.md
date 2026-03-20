# Phase 29: Specialist Agents + Agent Tests - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Create 5 specialist Claude Code subagents (Keeper, Tracker, Probe, Weaver, Auditor) and build a comprehensive AVA test suite that enforces quality standards on all agent files — frontmatter, tool restrictions, naming, and body content. No new agent roles, no slash command changes, no distribution pipeline work.

</domain>

<decisions>
## Implementation Decisions

### Agent depth & personality
- Full 8-section XML body for ALL 5 specialists — same depth as Phase 28 agents (Queen, Scout, Surveyors)
- Port structure from OpenCode source, rewrite content for Claude Code context — not a copy-paste, a reimagining
- Light colony flavor — functional first, colony references where natural (e.g., "escalate to the Queen" not "escalate to the orchestrator"). Matches Phase 28 tone.

### Test strictness & scope
- Tests validate ALL agent files that exist (not just the 5 new ones) — dynamic count that grows as phases ship
- TEST-05 (count=22) starts as a failing target until Phase 30 completes — that's intentional, not a bug
- Add body quality checks beyond the 5 requirements: verify XML sections present, no empty sections, minimum content length to catch lazy ports
- Claude's Discretion: test file organization (one file vs multiple), exact tool validation approach (forbidden-only vs exact match vs hybrid)

### Read-only boundaries
- Tracker: diagnose + suggest — returns root cause analysis AND a suggested fix, but doesn't apply it. Builder makes the change.
- Auditor: structured findings — returns file, line, severity, category, description, suggestion. No narrative review.
- Probe: Claude decides whether it writes + runs tests or writes only, based on the existing test workflow
- Claude's Discretion: Probe's run/write scope

### Agent specialization
- Keeper is ONE unified agent — "maintain project knowledge" encompasses architecture understanding AND wisdom management. No mode split.
- Weaver runs tests before + after refactoring. If tests break, it reverts. Behavior preservation is enforced, not just documented.
- Cross-reference escalation: specialists reference each other (Tracker → Builder for fixes, Auditor → Queen for security issues). Colony feels connected.
- Claude's Discretion: description style (generic vs colony-aware) — pick what routes best in Claude Code

</decisions>

<specifics>
## Specific Ideas

- Agent descriptions should be routing triggers, not role labels (PWR-06 from Phase 27 learnings)
- "YAML malformation silently drops agents" — run `/agents` verification pattern after each agent creation (established in Phase 27-28)
- Auditor and Tracker read-only status enforced by BOTH the tools field AND the test suite — belt and suspenders

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 29-specialist-agents-agent-tests*
*Context gathered: 2026-02-20*
