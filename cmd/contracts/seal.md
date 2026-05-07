# seal -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/seal_final_review.go, cmd/shelf_seal.go, cmd/codex_workflow_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --force | bool | no | false | Force seal even with active blockers |
| --plan-only | bool | no | false | Print seal review manifest without mutating state or spawning workers |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains seal review results, hive promotion results, archive location.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | update | Milestone set to "Crowned Anthill", state sealed |
| .aether/CROWNED-ANTHILL.md | create | Seal summary artifact |
| .aether/data/activity.log | append | Seal activity entries |
| ~/.aether/hive/wisdom.json | update | High-confidence instincts promoted to hive |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, not all phases complete, active blockers, or hive promotion failure |

## State Mutations

### Colony State Transitions
READY -> SEALED: Colony milestone set to "Crowned Anthill". All phases must be complete.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Milestone=Crowned Anthill, sealed state |
| .aether/CROWNED-ANTHILL.md | create | Seal summary document |
| ~/.aether/hive/wisdom.json | update | Instincts with confidence >= 0.8 promoted (non-blocking) |
| .aether/data/activity.log | append | Seal activity entries |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- All plan phases must be complete (or --force used to override blockers)
- No active blockers unless --force is used
- Colony must not already be sealed
