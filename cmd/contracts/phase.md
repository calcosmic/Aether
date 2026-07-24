# phase -- Lifecycle Contract

**Last verified:** 2026-05-17
**Source files:** cmd/phase.go, cmd/codex_visuals.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --number | int | no | 0 | Phase number to display; default resolves the current/recovery phase |
| --json | bool | no | false | Output as JSON |

### Arguments
None

## Outputs

### Stdout
- Default: visual phase dashboard via `renderPhaseVisual`.
- With `--json`: JSON envelope via `outputOK` containing phase metadata, task
  list, and progress counters.

### Ceremony Class
`phase` is a dashboard. It shows the runtime-owned phase plan and task status.
It may recommend `aether build <phase>` or `aether continue`, but it must not
claim a worker wave happened by displaying the phase.

### Files Created/Modified
None.

## State Mutations

### Colony State Transitions
None.

### Data Artifacts Modified
None.

## Preconditions

- Colony must be initialized.
- Requested phase number must exist in the current plan.
