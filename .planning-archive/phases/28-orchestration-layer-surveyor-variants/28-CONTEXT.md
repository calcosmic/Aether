# Phase 28: Orchestration Layer + Surveyor Variants - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Convert 7 OpenCode agents to Claude Code subagents: Queen, Scout, Route-Setter, and all 4 Surveyors (nest, disciplines, pathogens, provisions). Each agent gets YAML frontmatter + XML body matching the format established in Phase 27 (Builder and Watcher). The existing workflow and colony philosophy must be preserved — this is a format conversion, not a redesign.

</domain>

<decisions>
## Implementation Decisions

### Queen's Coordination Model
- Queen gets the Task tool in its tools field — it CAN spawn other named agents (aether-builder, aether-scout, etc.)
- This makes Queen a true orchestrator in Claude Code, not just an advisor
- Route-Setter also gets the Task tool (needs to verify plans sometimes) — all other agents escalate instead of spawning
- Preserve the full 4-tier escalation chain from OpenCode (worker retry -> parent reassign -> Queen reassign -> user escalation)
- The existing workflow is working well — this is a faithful port, not a redesign
- Agents that aren't Queen/Route-Setter can still spawn general-purpose tasks via Task tool — preserves the "ants spawning ants" philosophy. They just can't invoke named agents.

### Surveyor Consolidation
- Keep all 4 surveyors as separate Claude Code agent files (aether-surveyor-nest, aether-surveyor-disciplines, aether-surveyor-pathogens, aether-surveyor-provisions)
- Surveyors write their output files directly to `.aether/data/survey/` (not read-only — they need Write in tools field)
- NOTE: Roadmap success criteria says "no Write or Edit" for surveyors — override this. Surveyors need Write to create their survey documents. Restrict write scope to `.aether/data/survey/` only in their boundaries section.
- Keep the existing output location: `.aether/data/survey/`

### Routing Descriptions
- Descriptions should mention specific Aether commands that spawn them (e.g., "Spawned by /ant:build and /ant:oracle")
- Queen description must be specific enough to NOT fire for simple build tasks (Builder) or simple research (Scout)
- When Queen spawns workers via Task tool, the `description` parameter should include the caste emoji (e.g., "🔨🐜 Build authentication module", "🔭🐜 Research API patterns") so the terminal display shows which ant type is working

### Scout Capabilities
- Scout gets web search tools (WebSearch, WebFetch) in addition to codebase tools — broad research capability
- Keep Scout simple — quick research and report. Oracle stays as the deep iterative research path. Clear separation.
- Read-only vs writing research files: Claude's discretion based on how research results are consumed in colony workflows

### Claude's Discretion
- Sequential vs parallel spawning in Queen — pick based on what's practical in Claude Code's Task tool constraints
- Exact routing descriptions for all 7 agents — craft for optimal auto-selection while mentioning spawn sources
- Whether surveyors emphasize standalone use or colony-spawned use in descriptions
- Scout's read-only status vs ability to write research files

</decisions>

<specifics>
## Specific Ideas

- The user wants caste emojis visible when agents spawn in the terminal — include emoji in Task tool's `description` parameter
- Follow the exact same YAML frontmatter + XML body format from Phase 27 (Builder and Watcher are the template)
- PWR-01 through PWR-08 compliance required on all 7 agents (same checklist as Phase 27)
- 8 XML sections define the template: role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 28-orchestration-layer-surveyor-variants*
*Context gathered: 2026-02-20*
