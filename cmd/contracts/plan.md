# plan -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/codex_plan.go, cmd/codex_plan_finalize.go, cmd/codex_workflow_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --refresh | bool | no | false | Regenerate the plan even when an existing plan is present |
| --force | bool | no | false | Alias for --refresh |
| --plan-only | bool | no | false | Print dispatch manifest without mutating state or spawning workers |
| --depth | string | no | "" | Planning depth: fast, balanced, deep, or exhaustive |
| --planning-depth | string | no | "" | Task decomposition depth: light, standard, or deep |
| --verification-depth | string | no | "" | Verification depth: light, standard, or heavy |
| --synthetic | bool | no | false | Skip real worker dispatch, use local synthesis only |
| --worker-timeout | duration | no | 0 | Override per-worker timeout for planning dispatches |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains dispatch manifest with workflow, dispatch_mode, dispatches, planning depth, review_depth, stats.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .planning/phases/ | create | Phase plan files generated |
| .planning/ROADMAP.md | create/update | Roadmap with phase plan summary |
| .planning/STATE.md | create/update | State tracking for planning progress |
| .aether/data/COLONY_STATE.json | update | Plan metadata written to colony state |
| .aether/data/activity.log | append | Plan command activity |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, no survey data, or planning failure |

## State Mutations

### Colony State Transitions
READY -> READY (no state transition, but plan is populated in colony state).

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Plan phases, metadata written |
| .planning/phases/ | create | Phase plan documents |
| .planning/ROADMAP.md | create/update | Phase progress tracking |
| .aether/data/activity.log | append | Plan command activity |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Store must be initialized
- Survey data from colonize may be expected but not strictly required
