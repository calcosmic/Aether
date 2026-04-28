---
name: session-handoff
description: Use when pausing, resuming, or transferring context across sessions so work can continue safely
type: colony
domains: [session-management, context-preservation]
agent_roles: [keeper, chronicler, queen]
workflow_triggers: [resume, pause]
task_keywords: [handoff, resume, pause, session, continue from]
priority: normal
version: "1.0"
---

# Session Handoff

## Purpose

Create a complete context handoff for pause/resume across sessions. Ensures no work is lost when a session ends and the next agent can pick up exactly where the previous one left off.

## When to Use

- Ending a session with in-progress work
- Pausing mid-phase before completion
- Starting a new session and prior context exists
- Switching between workstreams or milestones
- User explicitly says "I'm done for now", "pause", "resume", or "continue from where we left off"

## Instructions

### Pause (Create Handoff)

1. Gather current work state:
   - **Active phase**: Phase number, name, and current step within the plan
   - **Completed steps**: List of PLAN.md steps marked done in this session
   - **In-progress step**: The step currently being executed, with partial state
   - **Decisions made**: Any architectural or implementation decisions from this session
   - **Blockers**: Anything preventing forward progress
   - **Next actions**: Ordered list of what to do next
   - **Files modified**: List of files changed in this session with brief descriptions
   - **Uncommitted state**: Whether changes are committed or staged

2. Write `.continue-here.md` at the project root:
   ```markdown
   # Continue Here
   > Session paused: {ISO8601 timestamp}

   ## Current State
   - Phase: {number} -- {name}
   - Step: {N}/{total} -- {step title}
   - Status: {in-progress|blocked|paused-between-steps}

   ## What Was Done
   - [x] Step N-2: {description}
   - [x] Step N-1: {description}
   - [~] Step N: {description} ({% complete})

   ## Next Actions
   1. {First thing to do on resume}
   2. {Second thing}

   ## Decisions This Session
   - {decision}: {rationale}

   ## Blockers / Risks
   - {blocker if any}

   ## Modified Files
   - `path/to/file` -- {what changed}
   ```

3. Optionally commit the handoff file if git is active

### Resume (Read Handoff)

1. Check for `.continue-here.md` at project root
2. If found:
   - Read and present the context summary to the user
   - Load referenced files and phase plan into context
   - Confirm with user before proceeding: "Found a session handoff for Phase N, step M. Resume from here?"
   - If user confirms, proceed with the next actions listed
   - If user declines, archive the handoff to `.aether/data/handoffs/{timestamp}.md`
3. If not found, proceed with normal session startup (check `aether status`)

### Archive

- After successful resume, move `.continue-here.md` to `.aether/data/handoffs/{timestamp}.md`
- Keep the last 10 handoffs for historical reference

## Key Patterns

- **Always overwrite**: `.continue-here.md` always reflects the latest state; archiving preserves history
- **Include enough context**: A fresh agent with no prior session knowledge should be able to resume work from the handoff alone
- **Reference, don't copy**: Link to PLAN.md steps rather than duplicating full content
- **Git awareness**: Note uncommitted changes so the resuming agent knows the state of the working tree
- **Timestamp tracking**: All handoffs are timestamped for chronological ordering

## Output Format

**Pause:**
```
Session handoff written to .continue-here.md
  Phase 3 -- Dashboard UI | Step 5/8 (62%)
  Next: Implement chart component with real-time data binding
  3 files modified, 2 uncommitted
```

**Resume:**
```
Found session handoff -- Phase 3, Step 5/8
  Last active: 2026-04-22T14:30:00Z
  Completed: Data models, API endpoints, Layout skeleton, Auth guards
  In progress: Chart component (30%)
  Resuming from: "Implement chart component with real-time data binding"
```

## Examples

```
# End of session
> session-handoff pause

# Start of new session
> session-handoff resume

# Check if handoff exists
> session-handoff status
```
