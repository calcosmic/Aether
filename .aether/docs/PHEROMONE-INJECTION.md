# Pheromone Injection System

## Auto-Injection at Critical Points

### Critical Moments Map

| Moment | Auto-Pheromone | Signal to User | Purpose |
|--------|---------------|----------------|---------|
| `/ant:init` completes | PHILOSOPHY[emergence-over-orchestration] | "🐜 Queen's scent laid: Emergence" | Foundation |
| First `/ant:plan` starts | PHILOSOPHY[minimal-planning-maximum-doing] | "🐜 Trail: Plan just enough, build soon" | Planning bias |
| Worker encounters error | REDIRECT[error-pattern] | "🐜 Warning pheromone deposited: {pattern}" | Learn from failure |
| Phase completes | PATTERN[what-worked] | "🐜 Success trail: {pattern} (strength: {n})" | Reinforce success |
| `/ant:seal` invoked | PHILOSOPHY[maturity-{milestone}] + PATTERN[sealed-conventions] | "🐜 Colony wisdom archived to eternal memory" | Lineage |
| `/ant:swarm` fixes bug | PATTERN[fix-strategy] + REDIRECT[what-failed] | "🐜 Swarm left trail: {solution-type}" | Bug immunity |
| User overrides worker | DECREE[override-reason] | "🐜 Decree recorded: {reason}" | Authority |

### User Signaling Format

```
═══════════════════════════════════════════════════════
  🐜 PHEROMONE DEPOSITED
═══════════════════════════════════════════════════════

  Type:     PATTERN
  Substance: prefer-bash-over-node-for-file-ops
  Strength:  0.7
  Source:    milestone:phase-3-complete
  Why:       Workers used bash 5x more than node for file ops

  This trail will guide future workers.
  To see all active trails: /ant:sniff

═══════════════════════════════════════════════════════
```

## Async Injection System

### The Challenge
User wants to inject pheromone while workers are active, without interrupting flow.

### Solution: Pheromone Queue + Checkpoint Polling

```
User: /ant:emit FOCUS "error-handling" (while workers building)

System:
1. Deposit pheromone immediately to .aether/data/pheromone-queue.json
2. Display: "🐜 Pheromone queued - workers will detect at next checkpoint"
3. Continue worker flow uninterrupted

Workers:
- Poll for queue at natural breakpoints (task completion, before spawn)
- Pick up queued pheromones without breaking context
```

### Pheromone Queue Structure

```json
{
  "queue": [
    {
      "id": "phem_123",
      "type": "FOCUS",
      "substance": "error-handling",
      "strength": 0.9,
      "deposited_at": "2026-02-15T10:30:00Z",
      "deposited_by": "user:queen",
      "status": "queued",
      "picked_up_by": null,
      "picked_up_at": null
    }
  ]
}
```

### Worker Checkpoint Protocol

```markdown
Every worker, at natural breaks:
1. Check pheromone-queue.json for unclaimed pheromones
2. If found:
   - Log: "[Worker] New scent detected: FOCUS[error-handling]"
   - Incorporate into context
   - Mark as picked_up_by: worker_id
3. Continue with adjusted context

Natural breakpoints:
- After completing a task
- Before spawning a sub-worker
- After tool use (Read/Edit/Bash)
- Every 5 minutes of continuous work
```

## Commands

### `/ant:emit <type> "<substance>" [--strength 0.0-1.0]`

Inject pheromone immediately or queue if workers active.

```
/ant:emit FOCUS "authentication-flow" --strength 0.9

🐜 PHEROMONE EMITTED
═══════════════════════════════════════════════════════

Type:      FOCUS
Substance: authentication-flow
Strength:  0.9 (high priority)
Status:    Active

Workers will detect this scent.
═══════════════════════════════════════════════════════
```

### `/ant:sniff [--type <type>] [--all]`

Display active pheromone trails.

```
/ant:sniff

🐜 ACTIVE PHEROMONE TRAILS
═══════════════════════════════════════════════════════

ETERNAL (never decay):
  PHILOSOPHY[emergence-over-orchestration]  ████████░░ 0.8
  PHILOSOPHY[minimal-change]                ████████░░ 0.8
  DECREE[no-force-push]                     █████████░ 0.9

FOCUS (decay: 30 days):
  FOCUS[authentication-flow]                █████████░ 0.9  (12 days left)
  FOCUS[performance-optimization]           ██████░░░░ 0.6  (5 days left)

REDIRECT (decay: 60 days):
  REDIRECT[regex-parsing]                   ███████░░░ 0.7  (45 days left)

PATTERN (decay: 90 days):
  PATTERN[bash-for-file-ops]                ████████░░ 0.8  (78 days left)

Queued for pickup:
  FOCUS[error-handling]                     █████████░ 0.9  (waiting)

═══════════════════════════════════════════════════════
To emit: /ant:emit <type> "<substance>" [--strength N]
```

## UX Flow: Onboarding Through Pheromones

### First Colony Experience

```
User: /ant:init "Build auth system"

System:
[Standard init flow...]

🐜 FIRST COLONY DETECTED
═══════════════════════════════════════════════════════

Your Queen's scent is being established.

Auto-deposited pheromones:
  ✓ PHILOSOPHY[emergence-over-orchestration]
  ✓ PHILOSOPHY[minimal-change]

These guide all future workers in this colony.

💡 Tip: You can adjust worker behavior mid-flight:
   /ant:emit FOCUS "security" --strength 0.9

═══════════════════════════════════════════════════════
```

### Learning by Example

When workers do something notable, system suggests pheromone:

```
[Builder Worker] Implemented 3 files using same pattern

🐜 PATTERN DETECTED
═══════════════════════════════════════════════════════

Workers consistently used: "check-file-exists-before-write"

Suggested pheromone:
  PATTERN[check-exists-before-write]

Deposit this for future colonies? [Y/n/help]

═══════════════════════════════════════════════════════
```

### Mid-Work Injection Example

```
[Workers actively building Phase 2...]

User: /ant:emit FOCUS "error-handling"

System:
🐜 PHEROMONE QUEUED
═══════════════════════════════════════════════════════

FOCUS[error-handling] queued for active workers.
Next checkpoint (within 60s), workers will adjust.

Current workers: 3 active
Queue position: 1

═══════════════════════════════════════════════════════

[30 seconds later, Prime Worker completes task]

Prime Worker: "New scent detected: FOCUS[error-handling]"
Prime Worker: "Adjusting priorities..."
Prime Worker: [Continues work with new focus]
```

## Implementation Notes

### Signal Batching
- Don't show every pheromone immediately
- Batch notifications every 30 seconds if multiple deposited
- Priority: DECREE > REDIRECT > FOCUS > PATTERN > PHILOSOPHY

### Context Window Management
- Pheromones condensed to single line in worker context:
  `[Pheromones: FOCUS(auth,0.9) REDIRECT(regex,0.7) PATTERN(bash-files,0.8)]`
- Full sniff available via tool call if worker needs details

### Persistence
- Eternal pheromones: `~/.aether/eternal/pheromones.json`
- Colony pheromones: `.aether/data/pheromones.json`
- Queue: `.aether/data/pheromone-queue.json`
- Midden (expired): `.aether/data/midden/pheromones.json`

---

*The colony learns as it works. The Queen guides without interrupting.*
