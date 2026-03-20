# Phase 22: Agent Boilerplate Cleanup - Context

**Gathered:** 2026-02-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Strip redundant sections from all 25 agent definition files to make them leaner and more focused. This is a content cleanup — no agent merging, no new capabilities, no scope changes. The goal is removing noise so each agent's unique instructions stand out clearly.

</domain>

<decisions>
## Implementation Decisions

### Cleanup Aggressiveness
- Safety-first approach — user explicitly concerned about breaking things
- Batch processing: clean 5-8 agents at a time, verify each batch before continuing
- Target: "focused but complete" — each agent reads like a full job description, just without the parts shared across all agents
- When in doubt, keep it — only strip what's clearly redundant

### What Counts as Redundant
- Generic AI prompting tips ("be thorough", "think step by step"): keep the ones that genuinely help performance, strip pure filler
- Tool availability lists: Claude decides per agent whether these add value
- Project description sections: Claude decides per agent whether project context is needed
- Colony rules (pheromones, castes, milestones): Claude decides per agent whether these are needed

### Agent Uniformity
- Structure/template: Claude decides based on best practices (shared core + flexible extras expected)
- Similar agents (multiple scouts/builders): Claude decides whether to share base definitions or keep independent
- Agent naming: YES, fix names that don't match what the agent actually does
- Agent merging: OUT OF SCOPE — strictly boilerplate cleanup, no boundary changes

### Edge Case Handling
- Near-duplicate sections (90% same, 10% different): Claude decides per case based on importance of differences
- Outdated references: DO NOT fix — only strip boilerplate. Outdated content is a separate task
- Verification: test each batch after cleanup to confirm agents still spawn and respond correctly

### Claude's Discretion
- Whether to leave pointers ("see workers.md") or clean-remove stripped sections — decide based on what works best per section
- Colony rules per agent — some agents need them, some don't
- Tool lists per agent — remove if purely noise, keep if they guide agent behavior
- Project descriptions per agent — remove or condense based on whether agent needs context
- Shared vs independent structure for similar agents
- Which generic prompting tips actually improve agent performance

</decisions>

<specifics>
## Specific Ideas

- User wants agents to feel like "focused but complete job descriptions" — not bare skeletons, but no copy-paste noise
- The analogy: like a recipe book where "wash your hands, preheat the oven" is removed because every cook already knows that — leaving just the actual recipe
- Batch approach (5-8 agents per batch) with verification between batches gives the user confidence nothing broke

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. Agent merging was explicitly ruled out of scope.

</deferred>

---

*Phase: 22-agent-boilerplate-cleanup*
*Context gathered: 2026-02-19*
