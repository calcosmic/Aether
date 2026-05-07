# recover -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/recover.go, cmd/recover_scanner.go, cmd/recover_repair.go, cmd/recovery_classify.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --apply | bool | no | false | Apply fixes for detected issues (default is scan-only) |
| --force | bool | no | false | Allow destructive repairs |
| --json | bool | no | false | Output structured JSON |

### Arguments
None

### Environment
None

## Outputs

### Stdout
- Default: Visual recovery report via `fmt.Fprint` listing detected issues and fixes applied.
- With --json: Structured JSON output with scan results.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/COLONY_STATE.json | update | When --apply repairs state issues |
| .aether/data/pheromones.json | update | When --apply clears stale signals |
| .aether/data/activity.log | append | Recovery activity logged |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success (issues found or not) |
| 1 | Colony not initialized |

## State Mutations

### Colony State Transitions
May repair stuck state: e.g., clear build-in-progress flags, reset from interrupted states, resolve stuck phase status. Only mutates when --apply is used.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/COLONY_STATE.json | update | Repair stuck state (only with --apply) |
| .aether/data/pheromones.json | update | Clear stale signals (only with --apply) |
| .aether/data/activity.log | append | Recovery scan results |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists) -- but handles "no colony" gracefully
- Colony is in a stuck or broken state (the reason to run recover)
- Store must be initialized
