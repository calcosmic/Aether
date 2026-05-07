# init -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/init_cmd.go, cmd/init_research.go, cmd/init_ceremony.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --scope | string | no | project | Colony scope: project or meta |
| --charter-json | string | no | "" | Approved charter data as JSON string |

### Arguments
`goal` (required) -- positional arg, exactly 1. The colony goal string.

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains state, goal, scope, version, phase, session, data_dir, shelf_backlog, shelf_backlog_count.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | create | Always on success |
| .aether/data/session.json | create | Always on success |
| .aether/data/activity.log | create/append | First COLONY_INITIALIZED entry |
| .aether/data/handoffs/worker-handoffs.json | create | Handoff document written |
| .aether/data/backups/COLONY_STATE.pre-init.\*.bak | create | When overwriting a sealed colony |
| .aether/dreams/ | create | Directory ensured |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Goal empty, invalid scope, colony already initialized, or seal in progress |

## State Mutations

### Colony State Transitions
None -> READY: Creates new colony state with State=READY, CurrentPhase=0, empty plan, empty instincts/signals/decisions.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | create | Full new ColonyState v3.0 with goal, scope, session ID, run ID, initialized timestamp |
| .aether/data/session.json | create | SessionFile with session ID, goal, suggested next="aether plan" |
| .aether/data/activity.log | append | COLONY_INITIALIZED entry with goal and session ID |

## Preconditions

- .aether/data/ directory must exist or be creatable (store must be initialized)
- No active colony with non-empty goal (idempotent: reports existing state if colony already initialized)
- If a sealed colony exists (Crowned Anthill), must not have uncommitted state changes (sealInProgress check)
