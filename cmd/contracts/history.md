# history -- Lifecycle Contract

**Last verified:** 2026-05-17
**Source files:** cmd/history.go, cmd/codex_visuals.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --limit | int | no | 20 | Maximum number of events to show |
| --filter | string | no | "" | Filter events by type or text |
| --json | bool | no | false | Output as JSON |

### Arguments
None

## Outputs

### Stdout
- Default: visual history dashboard via `renderHistoryVisual`.
- With `--json`: JSON envelope via `outputOK` containing parsed event entries.

### Ceremony Class
`history` is a dashboard. It reports stored Go-owned event facts only. Renderers
must not imply that showing history spawned, verified, or completed workers.

### Files Created/Modified
None.

## State Mutations

### Colony State Transitions
None.

### Data Artifacts Modified
None.

## Preconditions

- Store must be initialized.
- Missing colony history is rendered as an empty dashboard, not as worker
  activity.
