# build -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/codex_build.go, cmd/codex_build_finalize.go, cmd/codex_build_worktree.go, cmd/build_flow_cmds.go, cmd/codex_workflow_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --task | stringArray | no | [] | Redispatch only specified task IDs (repeatable or comma-separated) |
| --force | bool | no | false | Force redispatch of active phase after interrupted build |
| --plan-only | bool | no | false | Print build dispatch manifest without mutating state or spawning workers |
| --synthetic | bool | no | false | Skip real worker dispatch, use local synthesis only |
| --worker-timeout | duration | no | 0 | Override per-worker timeout (e.g. 15m) |
| --light | bool | no | false | Force light review (skip heavy agents on intermediate phases) |
| --heavy | bool | no | false | Force heavy review (full quality gauntlet on any phase) |
| --verification-depth | string | no | "" | Verification depth: light, standard, or heavy |
| --circuit-breaker-threshold | int | no | 3 | Consecutive failures before circuit breaker trips |
| --no-suggest | bool | no | false | Skip pheromone suggestion analysis during build |
| --verbose | bool | no | false | Show full worker output (default: filtered summary) |

### Arguments
`phase` (required) -- positional arg, exactly 1. The phase number to build (1-indexed).

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains dispatch manifest with workflow, dispatch_mode, dispatches, review_depth, phase, stats, worker spawn records.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | update | Build results, worker dispatch records |
| .aether/data/pheromones.json | update | Pheromone suggestion signals (unless --no-suggest) |
| .aether/data/handoffs/worker-handoffs.json | update | Worker handoff records |
| .aether/data/activity.log | append | Build command activity |
| .aether/data/spawn/ | create/update | Worker spawn tracking data |
| .aether/worktrees/ | create | When parallel_mode=worktree |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, invalid phase, no plan, or dispatch failure |

## State Mutations

### Colony State Transitions
READY -> BUILDING: Colony enters build state for the specified phase.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Build state, dispatch records, worker spawn tracking |
| .aether/data/pheromones.json | update | Auto-generated suggestion signals |
| .aether/data/handoffs/worker-handoffs.json | update | Worker handoff documents |
| .aether/data/spawn/ | create/update | Spawn tracking entries |
| .aether/data/activity.log | append | Build command activity |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Plan must exist with phases defined
- Specified phase number must be valid (>= 1, within plan range)
- Store must be initialized
