# resume -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/session_flow_cmds.go, cmd/session_cmds.go

## Inputs

### Flags
None (resume-colony has no flags; reads session.json automatically)

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Displays colony state, session context, phase progress, and suggested next command.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | update | Paused flag cleared, session restored |
| .aether/data/session.json | update | Session refresh with current state |
| .aether/data/handoffs/worker-handoffs.json | update | Resume handoff written |
| .aether/data/activity.log | append | Resume activity entry |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | No colony initialized or session recovery failure |

## State Mutations

### Colony State Transitions
May clear Paused flag: If colony was paused, marks Paused=false and restores active session.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Paused=false, session restored |
| .aether/data/session.json | update | Session refreshed |
| .aether/data/handoffs/worker-handoffs.json | update | Resume handoff |
| .aether/data/activity.log | append | Resume activity |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- session.json should exist from prior session
- Store must be initialized
