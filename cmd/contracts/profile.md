# profile -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/profile.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --dimension | string | no | "" | Behavioral dimension to observe |
| --signal | string | no | "" | Observed signal text |
| --strength | float64 | no | 1.0 | Observation strength between 0.0 and 1.0 |
| --evidence | string | no | "" | Concrete evidence for the observation |
| --command | string | no | "" | Optional command that produced the observation |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputOK`. Subcommands produce:
- `behavior-observe`: Records observation, returns confirmation.
- `profile-read`: Returns current behavioral profile data.
- `profile-update`: Promotes [profiled] directives to QUEEN.md.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/behavior-observations.jsonl | append | New behavioral observations recorded |
| ~/.aether/QUEEN.md | update | When profile-update promotes [profiled] directives |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized or invalid input |

## State Mutations

### Colony State Transitions
None -- does not modify COLONY_STATE.json directly.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/behavior-observations.jsonl | append | New observation entries |
| ~/.aether/QUEEN.md | update | [profiled] directives added to User Preferences section |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists) for observation capture
- Store must be initialized
- For profile-update/promotion: behavioral observations must exist to promote
