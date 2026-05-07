# status -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/status.go

## Inputs

### Flags
None

### Arguments
None

### Environment
| Variable | Description |
|----------|-------------|
| AETHER_OUTPUT_MODE | Set to "json" for JSON output; default is visual dashboard |

## Outputs

### Stdout
- Default: Visual colony dashboard via `renderDashboard` showing phase progress, memory health, active signals, flags.
- With AETHER_OUTPUT_MODE=json: JSON envelope via `outputOK` with structured state data.

### Files Created/Modified
None -- read-only command.

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success (colony exists) |
| 1 | Error loading colony state |

## State Mutations

### Colony State Transitions
None -- read-only.

### Data Artifacts Modified
None -- read-only command.

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists)
- Handles "no colony" gracefully with visual prompt to initialize
