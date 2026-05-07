# continue -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/codex_continue.go, cmd/codex_continue_finalize.go, cmd/codex_continue_plan.go, cmd/codex_workflow_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --reconcile-task | stringArray | no | [] | Mark task IDs as manually reconciled before continue gating |
| --plan-only | bool | no | false | Print continue manifest without mutating state or spawning review workers |
| --light | bool | no | false | Force light review (skip heavy review agents) |
| --heavy | bool | no | false | Force heavy review (full review gauntlet) |
| --verification-depth | string | no | "" | Verification depth: light, standard, or heavy |
| --worker-timeout | duration | no | 0 | Override per-worker timeout for continue dispatches |
| --verification-timeout | duration | no | 0 | Override deterministic verification timeout |
| --skip-watchers | bool | no | false | Skip watcher agent spawn, rely on verification commands only |
| --synthetic | bool | no | false | Mark as synthetic (skip real agents, use provided results) |
| --no-learn | bool | no | false | Disable learning capture for this run |

### Arguments
None

### Environment
| Variable | Description |
|----------|-------------|
| AETHER_CONTINUE_VERIFICATION_TIMEOUT | Default verification timeout |

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Contains verification results, review reports, phase advancement status, learning capture results, instincts recorded.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | update | Phase advancement, instinct capture, learning storage |
| .aether/data/pheromones.json | update | Decision-derived pheromone signals |
| .aether/data/reviews/ | create/update | Review ledger entries from watcher/probe/auditor |
| .aether/data/handoffs/worker-handoffs.json | update | Continue handoff records |
| .aether/data/activity.log | append | Continue command activity |
| .planning/STATE.md | update | Plan progress tracking |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, no build to verify, verification failure, or review failure |

## State Mutations

### Colony State Transitions
BUILDING -> READY: After successful verification and review, colony state returns to READY and CurrentPhase is advanced.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Phase advanced, instincts/learnings recorded, state transition |
| .aether/data/pheromones.json | update | Auto-emitted decision pheromones |
| .aether/data/reviews/ | create/update | Review reports from quality gates |
| .aether/data/handoffs/worker-handoffs.json | update | Continue handoff document |
| .aether/data/activity.log | append | Continue activity entries |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- A build must have completed for the current phase
- Workers must have been dispatched and returned results
- Store must be initialized
