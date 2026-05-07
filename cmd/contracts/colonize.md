# colonize -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/codex_colonize.go, cmd/codex_colonize_finalize.go, cmd/codex_workflow_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --force-resurvey | bool | no | false | Refresh survey artifacts even when existing survey is present |
| --force | bool | no | false | Alias for --force-resurvey |
| --plan-only | bool | no | false | Print dispatch manifest without mutating state or spawning workers |
| --worker-timeout | duration | no | 0 | Override per-worker timeout (e.g. 5m) |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains dispatch manifest with workflow, dispatch_mode, dispatches, detected_type, languages, frameworks, domains, entry_points, existing_survey, stats.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/survey/ | create | Survey territory reports from surveyor dispatches |
| .aether/data/COLONY_STATE.json | update | Survey results recorded in colony state |
| .aether/data/activity.log | append | Colonize activity entry |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, or worker dispatch failure |

## State Mutations

### Colony State Transitions
READY -> READY (no state transition, but survey data is populated).

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Survey metadata and results written |
| .aether/data/survey/ | create | Survey territory report files |
| .aether/data/activity.log | append | Colonize command activity |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Store must be initialized
- Working directory must be a valid project with scannable files
