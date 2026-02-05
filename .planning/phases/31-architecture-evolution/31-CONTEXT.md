# Phase 31: Architecture Evolution - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony supports hierarchical task delegation (spawn tree engine) and accumulates cross-project knowledge (two-tier learning system) that persists beyond individual projects. Project-local learnings live in memory.json, global learnings in ~/.aether/learnings.json. Workers signal sub-spawn needs via pheromones, Queen fulfills them. Depth capped at 2.

</domain>

<decisions>
## Implementation Decisions

### Learning promotion criteria
- Promotion happens at end-of-project (milestone completion), not via standalone CLI command
- Colony presents project learnings and user picks which to promote
- 50-entry cap on global learnings (as roadmapped in CP-5)

### Claude's Discretion: Promotion UX
- Whether colony suggests top candidates or presents flat list
- Overflow strategy when cap is reached (FIFO replacement vs forced curation)

### Spawn tree mechanics
- Workers signal sub-spawn needs via SPAWN pheromone emission
- Depth-2 workers are told they cannot sub-spawn and must handle the task inline
- Delegation tree displayed visually in /ant:build output during execution
- Spawn tree recorded in COLONY_STATE.json spawn_tree

### Claude's Discretion: Spawn timing
- Whether Queen fulfills SPAWN pheromones between waves (batched) or immediately mid-wave

### Learning injection behavior
- Global learnings injected after /ant:colonize (not during /ant:init) -- colonization provides project context for relevance filtering
- Only relevant learnings injected, filtered by tech stack and domain match against colonization results
- No expiry or staleness mechanism -- learnings persist until manually removed or replaced at cap
- Injected learnings are visible to user as explicit FEEDBACK pheromones with full learning text

### Spawn tree mode awareness
- Spawn tree works the same across LIGHTWEIGHT/STANDARD/FULL modes -- always allows depth-2
- Sub-spawned workers inherit parent's pheromone context (FOCUS/REDIRECT) plus their specific sub-task

### Platform validation (CP-1)
- If Task tool is unavailable to subagents: fallback to Queen-mediated delegation (workers describe sub-tasks in output, Queen reads and spawns new workers directly)
- Validation approach left to Claude's discretion during planning

</decisions>

<specifics>
## Specific Ideas

- Queen-mediated delegation fallback should produce the same observable result as direct sub-spawning -- the user shouldn't need to know which path was taken
- Visual delegation tree in build output should show parent-child relationships and depth levels

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 31-architecture-evolution*
*Context gathered: 2026-02-05*
