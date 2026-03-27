# Phase 23: Agent Resilience - Context

**Gathered:** 2026-02-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Add failure modes, success criteria, and read-only declarations to all agent definitions (both OpenCode agents and Claude Code slash commands). These are advisory sections written into markdown files that LLM agents read and follow — no enforcement layer, no runtime checks.

</domain>

<decisions>
## Implementation Decisions

### Failure behavior
- Agents should try to fix problems autonomously first (2 attempts max)
- If recovery fails, escalate by presenting 2-3 concrete options with trade-offs — let the user choose
- Tiered by severity: minor issues (missing file, command fail) → retry silently; major issues (state corruption, data loss risk) → stop immediately and escalate
- No silent failures — if an agent gives up, it explains what happened

### Success signals
- Agents report what they produced/changed AND confirm they verified their own work ("Created 3 files, ran validation, all checks pass")
- Success criteria are agent-specific — each agent defines what "done right" means for its role (builder checks code works, watcher checks tests pass, scout checks sources found)
- High-stakes agents (builder, queen, watcher) get peer review from another agent; lower-risk agents self-verify only
- If self-check fails, agent retries automatically (within 2-attempt limit) before escalating

### Safety boundaries
- Read-only declarations are advisory — written into agent definitions as rules the LLM reads and respects
- Tiered boundary approach (Claude's discretion on specifics):
  - Globally protected paths (colony state, user data, dreams, checkpoints)
  - Per-agent boundaries based on what each agent's role actually needs to touch

### Prioritization
- Tier agents by risk level based on what they can modify (Claude classifies)
- High-risk agents (those that modify files, state, git) → detailed failure modes, strict boundaries, peer review
- Low-risk agents (read-only roles like chronicler, sage) → lighter treatment
- Format: XML tags (`<failure_modes>`, `<success_criteria>`, `<read_only>`) — consistent with Aether's XML convention for LLM-readable structure

### Scope
- Both OpenCode agents (`.opencode/agents/`) AND Claude Code slash commands (`.claude/commands/ant/`)
- Matches existing post-Phase-22 cleanup format with new XML-tagged sections added

### Claude's Discretion
- Exact risk tier classification for all 25 agents
- Which specific paths go in each agent's read-only list
- How safety violations are handled (severity-based response)
- Which agents need peer review vs self-verify
- Protected path recommendations beyond current set (colony state, user data, .env)

</decisions>

<specifics>
## Specific Ideas

- XML tags for new sections: `<failure_modes>`, `<success_criteria>`, `<read_only>`
- Self-check failures follow the same 2-attempt retry rule as regular failures
- Escalation format: present options, not just error messages

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 23-agent-resilience*
*Context gathered: 2026-02-19*
