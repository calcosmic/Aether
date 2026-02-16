# Phase 5: State & Context Restoration - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Ensure reliable cross-session memory and context for the Aether colony system. Colony state loads on every command invocation, context restoration works after session pause/resume, and the spawn tree persists correctly across sessions. This is about maintaining colony continuity — like pheromone trails that persist even when individual ants are not present.

</domain>

<decisions>
## Implementation Decisions

### Context restoration trigger
- **Automatic on every command invocation** — no explicit resume command needed
- Every `ant:*` command or CLI invocation first loads and validates colony state
- If paused state exists (from `/ant:pause-colony`), display brief summary before executing command
- Resume is implicit — the colony never "forgets," it just continues

### Spawn tree persistence
- **Full serialization to COLONY_STATE.json** — complete spawn history preserved
- Spawn tree includes: active spawns, completed spawns, and spawn relationships (parent/child)
- Each spawn entry: id, caste, status (pending/active/completed/failed), parent_id, children_ids, created_at, completed_at
- On load, reconstruct the in-memory spawn tree from JSON structure
- Completed spawns are archived but retained for archaeology/history purposes

### State validation failure handling
- **Graceful degradation with clear user feedback**
- Validation on every load: duplicate keys, timestamp ordering, required fields, JSON schema
- If validation fails:
  1. Log detailed error to activity.log
  2. Display clear error message to user with specific issues found
  3. Offer: attempt auto-repair, start fresh (backup corrupted state), or exit for manual fix
  4. Never silently start fresh — always inform the user of state issues
- Backup corrupted state to `.aether/archive/COLONY_STATE.backup.{timestamp}.json` before any repair

### Pause/resume UX
- **Pheromone trail metaphor** — pause leaves markers, resume follows them
- On pause (`/ant:pause-colony`):
  - Create handoff document at `.aether/state/handoff.md` with: last command, active spawns, blockers, next steps
  - Display summary of what the colony was doing
  - State is saved to COLONY_STATE.json with paused flag
- On resume (implicit on next command):
  - If handoff.md exists, display brief summary: "Colony was working on X, Y active spawns"
  - Remove handoff.md after successful load (cleanup)
  - Continue as if no interruption occurred

### Event timestamp ordering
- **Strict chronological order enforced** — events appended in real-time, never retroactively modified
- On load: verify event timestamps are monotonically increasing
- If out-of-order events detected (should not happen with atomic writes): log warning, attempt to sort, flag for review

### State loading integration
- **Every command starts with state load** — no exceptions
- Load sequence: acquire lock → read COLONY_STATE.json → validate → reconstruct spawn tree → release lock
- Lock prevents concurrent modifications during read
- State available to all commands via colony context object

</decisions>

<specifics>
## Specific Ideas

- "Like ants following pheromone trails — the colony continues even when individual ants change shifts"
- Pause/resume should feel seamless — user returns and immediately knows where things stand
- Handoff document is temporary, like a scout ant's trail that evaporates once the message is delivered
- State validation errors are treated seriously — the colony doesn't pretend everything is fine when it's not

</specifics>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-state-context-restoration*
*Context gathered: 2026-02-14*
