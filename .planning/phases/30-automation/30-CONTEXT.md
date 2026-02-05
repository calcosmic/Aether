# Phase 30: Automation & New Capabilities - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony automates post-build quality gates, surfaces actionable recommendations, and provides visual feedback during execution. Delivers: auto-spawned reviewer (advisory mode), auto-spawned debugger (on test failure), pheromone recommendations from build outcomes, ANSI-colored progress indicators with caste-specific colors, and tech debt report at project completion.

Note: Aether is a standalone system. All output and state lives within Aether's own structures (COLONY_STATE.json, activity log, etc.), not in .planning/ which belongs to CDS.

</domain>

<decisions>
## Implementation Decisions

### Reviewer behavior
- Findings presented inline in build output after each wave completes — compact, no separate section
- Reviewer runs after each wave, not just at end of build — catches issues early, prevents cascading problems in later waves
- Only CRITICAL severity triggers rebuild (max 2 iterations)
- CRITICAL vs WARNING severity boundary: Claude's discretion
- Rebuild strategy (overwrite vs revert): Claude's discretion

### Debugger spawning
- Debugger attempts to diagnose AND fix — not just report
- Spawns only after worker's own retry fails — worker gets one retry attempt first, debugger triggers on second failure
- Debugger patches the worker's existing code rather than rewriting from scratch — keeps original approach intact
- If debugger's fix also fails: Claude's discretion based on task criticality

### Pheromone suggestions
- Natural language descriptive guidance, not copy-paste commands — e.g., "Consider focusing the colony on the auth module — it had repeated test failures"
- Triggered by all build outcomes — both failures (error clusters, repeated approach failures, quality variance) and successes (strong performance areas, expansion opportunities)
- Suggestions can appear between waves when urgent patterns emerge, plus final recommendations at build end
- Maximum 3 suggestions per build — forces prioritization of most impactful signals

### Visual output
- Status line per task: [caste-color] task name ... status — updated inline as tasks complete
- Full color scheme for all castes — not just the three in the roadmap. Extend to Queen (yellow), debugger (red), reviewer (blue), and any other castes
- Tech debt report format and aggregation strategy: Claude's discretion

### Claude's Discretion
- CRITICAL vs WARNING severity definition for reviewer
- Rebuild approach (overwrite vs revert-then-rebuild)
- Debugger behavior when its fix also fails (halt vs skip, based on task criticality)
- Tech debt report format (terminal, file, or both) and aggregation method
- Exact color assignments for extended caste scheme
- Tech debt report lives within Aether's state, not .planning/

</decisions>

<specifics>
## Specific Ideas

- Reviewer findings inline after each wave — not a separate report section, integrated into the build flow
- Debugger is a patch-first approach — preserve worker's intent, fix the specific failure
- Pheromone suggestions should feel like a senior engineer's observations, not automated alerts
- Worker gets one retry before debugger spawns — reduces noise from transient failures

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 30-automation*
*Context gathered: 2026-02-05*
