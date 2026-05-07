# discuss -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/discuss.go, cmd/discuss_analyze.go, cmd/pending_decision.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --max-questions | int | no | 3 | Maximum number of clarification questions to surface |
| --dry-run | bool | no | false | Analyze and preview questions without writing pending decisions |
| --resolve | string | no | "" | Clarification decision ID to resolve |
| --answer | string | no | "" | Resolution text for --resolve |

### Arguments
None

### Environment
None

## Outputs

### Stdout
JSON envelope via `outputWorkflow` (visual + structured). Two modes:
- **Surface mode** (default): returns goal, question_count, created_count, existing_count, dry_run, questions, resolved, pending_count, signal_count, discussion_status.
- **Resolve mode** (--resolve): returns resolved, id, answer, redirect_emitted, redirect_text, remaining.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/data/pending-decisions.json | update | When new clarification questions are materialized (not in dry-run) |
| .aether/data/pending-decisions.json | update | When --resolve marks a clarification as resolved |
| .aether/data/pheromones.json | update | When resolved clarification is a hard constraint, emits REDIRECT signal |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, goal empty, or resolve ID not found |

## State Mutations

### Colony State Transitions
None -- does not modify COLONY_STATE.json directly.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/data/pending-decisions.json | update | Appends clarification decisions or marks existing ones resolved |
| .aether/data/pheromones.json | update | May emit REDIRECT pheromone for hard constraint resolutions |

## Preconditions

- Colony must be initialized (COLONY_STATE.json exists with non-empty goal)
- Store must be initialized
