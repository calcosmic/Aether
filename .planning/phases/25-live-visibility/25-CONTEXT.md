# Phase 25: Live Visibility - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Make worker progress visible to users during execution. Workers write structured progress to an activity log, the Queen spawns workers directly (not delegated to Phase Lead), and displays condensed results incrementally as each worker completes. The Phase Lead shifts from "spawn and manage workers" to "produce a task assignment plan" that the Queen executes.

</domain>

<decisions>
## Implementation Decisions

### Activity log format
- Granular logging — every action: file created, modified, researched, spawn events, start/end per task
- Emoji-prefixed lines using existing worker progress format (e.g. `⏳ 🔨🐜 Working on: {task}`, `✅ 🔨🐜 Completed: {task}`)
- Spawn/delegation events visible in log: `🐜 🔨🐜 → 🔍🐜 Spawning scout-ant for: {reason}`
- Log includes phase header: phase number, name, start time, planned tasks
- Errors appear inline with `❌` prefix plus an error summary block at the end of the log
- Log rotates per phase: previous log archived as `activity-{phase}.log`, current phase gets fresh `activity.log`

### Queen display behavior
- Condensed summary per worker: task outcomes (✅/❌), files changed count, any errors
- Announce each worker BEFORE spawning: `🐜 Spawning 🔨🐜 builder-ant for: {tasks}...`
- Show results after worker returns as condensed summary
- Visual progress bar across all workers: `██████████░░░░░░░░░░ 3/6 workers complete`
- No final consolidated summary — per-worker summaries are sufficient

### Phase Lead role change
- Phase Lead produces a task assignment plan, not spawned workers
- Plan format: ordered list with context (not table)
  ```
  1. 🔍🐜 scout-ant: research auth middleware patterns (tasks 1, 2)
  2. 🔨🐜 builder-ant: implement auth middleware (tasks 3, 5)
     → Needs: scout results
  3. 🔨🐜 builder-ant: implement route guards (task 4)
  ```
- **User checkpoint**: Queen displays the Phase Lead's plan and asks "Proceed with this plan?" — user must confirm before workers spawn
- If user rejects the plan: Queen asks what they'd like changed, then re-runs the Phase Lead with that feedback

### Worker spawn sequencing
- Wave-based execution: group independent tasks into waves, run each wave in parallel, show all results, then next wave
- Clear wave boundary markers: `─── Wave 2/3 ───`
- On worker failure: retry with a different approach (spawn new worker with failure context, asking for alternative approach)
- Max 2 retries per task before escalating to user
- Continue running independent tasks in remaining waves even if a task is being retried

### Claude's Discretion
- Whether to add timestamps to individual log lines (vs keeping clean emoji-only format)
- Activity log write mechanism (aether-utils.sh function vs direct file writes — pick what's most reliable for concurrent workers)
- Exact progress bar rendering style
- How to pass failure context to retry workers

</decisions>

<specifics>
## Specific Ideas

- User wants the system to be "visually stimulating" with "clearer understanding of processes taking place" — the display should make the colony feel alive and active
- Autonomy with restraint: the plan checkpoint is the key control point — user sees what's about to happen, confirms, then execution is fully autonomous
- "The job needs to get done" — failure handling should be resilient, never waste valid work, always find a way to complete tasks rather than stopping or skipping
- The Phase Lead plan display should read as a narrative the user can quickly understand, not a dense data structure

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 25-live-visibility*
*Context gathered: 2026-02-04*
