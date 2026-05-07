# entomb -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/entomb_cmd.go, cmd/shelf_entomb.go, cmd/chamber.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --import-signals | bool | no | false | Import pheromone signals from the chamber's colony-archive.xml |

### Arguments
None

### Environment
None

## Outputs

### Stdout
Visual output via `fmt.Fprint`. Displays archive location, colony summary, and next steps.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/chambers/{name}/ | create | Archive directory for sealed colony |
| .aether/chambers/{name}/COLONY_STATE.json | create | Archived colony state |
| .aether/chambers/{name}/colony-archive.xml | create | XML archive of colony data |
| .aether/data/COLONY_STATE.json | update | Goal cleared, state reset to IDLE |
| .aether/CROWNED-ANTHILL.md | delete | Removed after archiving |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, not sealed, CROWNED-ANTHILL.md missing |

## State Mutations

### Colony State Transitions
SEALED -> IDLE: Colony goal cleared, state reset. Active colony data archived to chambers.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/chambers/{name}/ | create | Full colony archive directory |
| .aether/data/COLONY_STATE.json | update | Goal cleared, state=IDLE |
| .aether/CROWNED-ANTHILL.md | delete | Removed after successful archive |

## Preconditions

- Colony must be sealed (milestone = "Crowned Anthill")
- .aether/CROWNED-ANTHILL.md must exist
- Store must be initialized
