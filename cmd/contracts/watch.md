# watch -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/compatibility_cmds.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --once | bool | no | false | Render a single watch snapshot even in visual TTY mode |
| --interval | duration | no | 2s | Refresh interval for live watch output |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Shows worker activity, build status, and colony health snapshot. In TTY mode, renders live refreshing display.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/watch-snapshot.json | create | Watch snapshot artifact |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | No store initialized |

## State Mutations

### Colony State Transitions
None -- read-only monitoring.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/watch-snapshot.json | create | Watch snapshot data |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Store must be initialized
