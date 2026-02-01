# Phase 5: Phase Boundaries - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

## Phase Boundary

Colony operates through explicit state machine with phase boundaries, checkpoints, and recovery capability. This phase delivers the structural backbone — the framework that manages how phases transition, when Queen check-ins happen, and how the colony can recover from failures. Within phases, pure emergence occurs; at boundaries, structure provides Queen check-ins.

## Implementation Decisions

### State machine structure
- 7 states: IDLE, INIT, PLANNING, EXECUTING, VERIFYING, COMPLETED, FAILED
- INIT is distinct from IDLE (one means "ready", the other "setting up")
- VERIFYING is separate because that's when the Watcher caste works
- Full lifecycle coverage gives clear visibility into what the colony is doing

### State transitions
- Pheromone-based triggers only (INIT, FOCUS, REDIRECT, FEEDBACK signals drive state changes)
- INIT pheromone → IDLE→INIT
- Phase complete event → EXECUTING→VERIFYING
- Emergency override function available for crash recovery (breaks from pure pheromone pattern when necessary)
- Whitelist validation: define allowed transitions in schema (e.g., IDLE→INIT, PLANNING→EXECUTING), reject everything else

### Invalid transition handling
- Invalid transitions cause colony to enter ERROR state
- Colony stops work in ERROR state (halt, don't continue cascading)
- Queen approval required to recover from ERROR state (aligns with structure-at-boundaries philosophy)
- ERROR logged with full context for debugging

### State history
- Rolling window of last 10 transitions in COLONY_STATE.json
- Enough trail to debug recent issues without unbounded growth
- Tracks: from_state, to_state, timestamp, trigger_pheromone

### Checkpoint strategy
- **Contents:** Complete colony state (COLONY_STATE.json, pheromones.json, memory.json)
- **Timing:** Pre AND post every state transition (2 checkpoints per transition)
- **Retention:** Rolling window of last 10 checkpoints
- **Location:** `.aether/checkpoints/` directory
- **Naming:** `checkpoint-{state_from}-to-{state_to}-YYYYMMDD-HHMMSS.json` format
- **Compression:** No compression — fast recovery, disk space acceptable
- **Failure mode:** Log warning but continue (some safety better than none)
- **Recovery:** Auto-rollback on ERROR state + manual recovery command available

### Queen check-in flow
- **Trigger:** Auto-pause at phase boundary (colony stops when phase completes)
- **Display:** Comprehensive summary to terminal (phase results, metrics, issues, pending decisions, memory summary)
- **Continue:** Explicit approval required — Queen runs command to proceed
- **Intervention:** Both pheromones AND direct plan modification available during check-in
- **Format:** Terminal output (no separate check-in file)
- **Timing:** Checkpoint happens first, then check-in summary displayed
- **Blocked state:** Colony enters WAITING state during check-in, blocks all work
- **Override:** `/ant:execute --auto` flag to skip check-in for trusted phases

### Next phase adaptation
- **Memory source:** All three memory layers (Working, Short-term, Long-term)
- **Method:** Explicit query by Route-setter Ant when planning next phase
- **Filtering:** All patterns available (let planner decide relevance)
- **Confidence:** Use confidence scores for ranking (high-confidence patterns prioritized, 3+ occurrences, score > 0.7)
- **Meta:** Plan documents memory usage in memory context section
- **Failure:** Use cached patterns from previous successful phases if memory access fails

### Claude's Discretion
- Exact ERROR state recovery mechanics
- Checkpoint file implementation details (bash/jq patterns)
- State transition function implementation
- Check-in summary format and content organization
- Memory query interface design

## Specific Ideas

- Check-in summary should feel like a project status report — clear metrics, issues surfaced, next steps visible
- ERROR state should feel like a safety hatch — colony stops, Queen assesses, explicit recovery
- Checkpoints are "undo points" — colony can always roll back to last known good state
- Pre+post checkpointing means "save before, save after" — maximum safety belt

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 05-phase-boundaries*
*Context gathered: 2026-02-01*
