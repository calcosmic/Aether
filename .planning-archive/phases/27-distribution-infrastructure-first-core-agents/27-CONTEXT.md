# Phase 27: Distribution Infrastructure + First Core Agents - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Prove the end-to-end agent distribution chain works: packaging Claude Code agent files, syncing through the hub, delivering to target repos via `aether update`. Builder and Watcher are the first two agents shipped through this proven chain. All 8 PWR standards must pass for both agents.

</domain>

<decisions>
## Implementation Decisions

### Agent file format
- Files named `aether-{role}.md` (e.g. `aether-builder.md`, `aether-watcher.md`)
- Files live in `.claude/agents/ant/` in both source and target repos
- Descriptions written as routing triggers — specific trigger cases that tell the Task tool WHEN to use the agent, not generic role labels
- Full XML body ported from OpenCode agent definitions — all instructions, failure modes, success criteria carried over
- All 8 PWR standards (PWR-01 through PWR-08) required for every agent, no exceptions

### Distribution pipeline
- Target repos receive agents at `.claude/agents/ant/` via `aether update`
- Hub path and pipeline approach are Claude's discretion — pick what fits existing architecture best
- GSD agent isolation is Claude's discretion — determine if directory structure alone is sufficient

### Conversion approach
- Builder and Watcher are template/exemplar conversions — future phases copy their structure exactly
- Spawn calls handled at Claude's discretion — determine best replacement pattern per agent
- Every converted agent must verify loading in Claude Code (appears in `/agents` output) — catches silent YAML issues
- All 8 PWR standards must pass for every converted agent

### Cleanup on removal
- Auto-delete: if agent file exists in target but not in hub, remove it during `aether update`
- Show changes: `aether update` lists added, updated, and removed agent files
- Overwrite/conflict behavior is Claude's discretion — match existing system file handling
- Idempotency approach is Claude's discretion — balance accuracy and speed

### Claude's Discretion
- Model field in frontmatter — decide based on what's proven to work in Claude Code
- Hub path structure — fit the existing hub layout
- Pipeline integration — same vs separate path from .aether/ system files
- GSD agent isolation mechanism
- Spawn call replacement pattern
- Conflict handling on local modifications
- Idempotency check method (content vs existence)

</decisions>

<specifics>
## Specific Ideas

- Builder and Watcher should be exemplary — they set the template that all 20 remaining agents follow
- Research found YAML malformation silently drops agents — must verify each agent loads via `/agents`
- Research found tool inheritance over-permissions agents — explicit tools field required on every agent
- Research found subagents cannot spawn other subagents — all spawn calls must be addressed

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 27-distribution-infrastructure-first-core-agents*
*Context gathered: 2026-02-20*
