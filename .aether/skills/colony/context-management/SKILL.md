---
name: context-management
description: Use when performing state-changing operations, recovering sessions, creating handoffs, or managing context freshness
type: colony
domains: [context, session, recovery, state]
agent_roles: [builder, watcher, scout, architect]
priority: normal
version: "1.0"
---

# Context Management

## Purpose

Colony context must stay fresh and recoverable. When sessions break, context is the bridge that lets the next session pick up without losing progress. This skill teaches you how to maintain that bridge.

## After State-Changing Operations

Any operation that modifies colony state must update context. State-changing operations include:

- Phase advances (current_phase changes)
- Task completion (task status changes)
- Instinct creation or promotion
- Pheromone signal creation, reinforcement, or expiration
- Blocker flags raised or resolved
- Plan regeneration

After each of these, call `context-update` to write the current state summary to CONTEXT.md. This ensures the context file always reflects reality.

## File Freshness Verification

Before reading state files (COLONY_STATE.json, pheromones.json, session.json), verify freshness using `session-verify-fresh`:

- If the file is stale (older than the current session), treat data with caution.
- For protected commands (init, seal, entomb), never auto-clear stale files -- prompt the user.
- For other commands, auto-clear stale session files and log that you did so.

Freshness matters because state files from a previous session may contain outdated phase pointers, resolved blockers, or expired signals that would mislead the current session.

## HANDOFF.md Requirements

When creating a handoff (via `/ant:pause-colony` or session end), HANDOFF.md must include all five sections:

1. **Goal** -- The active colony goal, exactly as stated in COLONY_STATE.json.
2. **Phase** -- Current phase number and name, plus completion percentage.
3. **Signals** -- All active pheromone signals with their types and strengths.
4. **Blockers** -- Any active blocker flags, including who raised them and why.
5. **Next Step** -- The exact command to run next and what it will do.

Missing any of these sections makes recovery harder. If a section has no content (e.g., no blockers), include it with "None" rather than omitting it.

## Session Recovery Fallback Chain

When resuming, follow this fallback chain:

1. **HANDOFF.md** (preferred) -- Richest context, includes signals and blockers.
2. **CONTEXT.md + COLONY_STATE.json** (fallback) -- Reconstruct from state files if no handoff exists.
3. **COLONY_STATE.json alone** (last resort) -- Minimal recovery, may miss recent changes.

Always tell the user which recovery source you used: "Restored from HANDOFF.md" or "No handoff found, recovered from state files."

## Context at Natural Break Points

At natural breaks (phase completion, verification pass, seal), prompt the user to consider clearing context if the conversation has been long:

```
This session has been running for a while. Consider running /ant:pause-colony
to save state, then start fresh with /ant:resume-colony.
```

This helps prevent context window degradation in long sessions.

## CONTEXT.md Content

CONTEXT.md should contain a concise summary (not a full dump) of:
- Active goal and current phase
- Recent completions (last 2-3 tasks)
- Active signals (type and content only, not full JSON)
- Known blockers
- Recommended next action
