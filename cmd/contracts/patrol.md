# patrol -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/patrol_check.go

## Inputs

### Flags
None

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains health check results: JSON validity, stale pheromones, interrupted builds. Overall status: healthy, warning, or error.

### Files Created/Modified
None -- read-only diagnostics.

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | No store initialized |

## State Mutations

### Colony State Transitions
None -- read-only diagnostics.

### Data Artifacts Modified
None -- read-only command.

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Store must be initialized
