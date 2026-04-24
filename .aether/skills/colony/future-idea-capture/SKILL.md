---
name: future-idea-capture
description: Use when a future idea should be captured with trigger conditions rather than forced into the active phase
type: colony
domains: [idea-management, forward-planning, knowledge-base]
agent_roles: [keeper, chronicler, queen]
workflow_triggers: [discuss, plan]
task_keywords: [future, someday, seed, later, idea]
priority: normal
version: "1.0"
---

# Future Idea Capture

## Purpose

Capture forward-looking ideas with trigger conditions that auto-surface when the right milestone arrives. Tracks seed lifecycle from planting through germination to harvest (implementation) or withering (discarded).

## When to Use

- An idea is too early to act on but too good to lose
- User says "someday we should...", "in the future...", "when we get to X, consider Y"
- A dependency or prerequisite exists before the idea can be pursued
- You want to queue up improvements for a future milestone
- Reviewing roadmap and identifying opportunities that aren't ready yet

## Instructions

### Plant a Seed

1. Capture the idea with structured metadata:
   ```json
   {
     "id": "SEED-{NNN}",
     "title": "{concise title}",
     "description": "{full idea description}",
     "planted": "ISO8601",
     "status": "planted",
     "triggers": {
       "milestone": "{milestone id or null}",
       "phase": "{phase number or null}",
       "tag": "{keyword tag}",
       "condition": "{natural language condition}"
     },
     "priority": "low|normal|high",
     "source": "{where the idea came from}",
     "related_seeds": ["SEED-{NNN}"]
   }
   ```

2. Append to `.aether/data/seeds.jsonl`
3. Confirm planting with the user, showing the seed ID and trigger conditions

### Check for Germination

1. Run automatically at milestone creation and phase planning
2. For each `planted` seed, evaluate triggers against current context:
   - `milestone`: Does the current or upcoming milestone match?
   - `phase`: Is the current phase number >= the trigger phase?
   - `tag`: Does any active work mention or relate to the tag?
   - `condition`: Natural language condition evaluated against current project state
3. Seeds with met triggers transition to `germinating` status
4. Present germinated seeds to the user for triage:
   - **Harvest now**: Promote to a todo or phase requirement
   - **Let grow**: Keep germinating, revisit next cycle
   - **Wither**: Mark as `withered` with reason

### Lifecycle States

- `planted` -> Trigger conditions not yet met, waiting
- `germinating` -> Triggers met, awaiting user decision
- `harvested` -> Promoted to actionable work (link to todo or phase)
- `withered` -> Discarded (with reason recorded)
- `dormant` -> Temporarily paused (user explicitly deferred)

### Review Seeds

1. List all seeds with optional filters:
   - `--status planted|germinating|harvested|withered|dormant`
   - `--tag <tag>`
   - `--milestone <milestone id>`
2. Display as a table with age (days since planting) and trigger status

## Key Patterns

- **Trigger specificity**: Vague triggers ("eventually") are valid; specific triggers ("milestone 3") are preferred
- **Non-blocking**: Seeds never block current work; they only surface when conditions align
- **Deduplication**: Before planting, check for similar seeds. Merge if 80%+ similar.
- **Age awareness**: Seeds older than 90 days with no trigger match should be flagged for review
- **Harvest trail**: When a seed is harvested, record what it became (todo ID, phase number, etc.)

## Output Format

```
SEED-007 planted: "Migrate to event-sourced order processing"
  Triggers: milestone >= MS-04, tag: "scalability"
  Priority: low | Age: 0 days
  Status: planted -> waiting for conditions

Germinating seeds for this milestone:
  SEED-003: "Add real-time notifications via WebSocket" (42 days old)
  -> [harvest] [let grow] [wither]
```

## Examples

```
# Plant a seed
> seed-planter plant "When we hit 10k users, migrate to read replicas" --trigger "milestone:MS-05" --tag scalability

# Check for germinating seeds
> seed-planter check

# List all seeds
> seed-planter list --status planted

# Harvest a germinated seed
> seed-planter harvest SEED-003 --as todo
```
