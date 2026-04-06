<!-- Generated from .aether/commands/maturity.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:maturity
description: "👑🐜🏛️🐜👑 View colony maturity journey with ASCII art anthill"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen**. Display the colony's maturity journey.

## Instructions

### Step 1: Detect Current Milestone

Run:
`bash .aether/aether-utils.sh milestone-detect`

Parse JSON result to get:
- `milestone`: Current milestone name (First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill)
- `version`: Computed version string
- `phases_completed`: Number of completed phases
- `total_phases`: Total phases in plan

### Step 2: Read Colony State

Read `.aether/data/COLONY_STATE.json` to get:
- `goal`: Colony goal
- `initialized_at`: When colony was started

Calculate colony age from initialized_at to now (in days).

### Step 3: Display Maturity Journey

Display header:
```
       .-.
      (o o)  AETHER COLONY
      | O |  Maturity Journey
       `-'
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal: {goal (truncated to 50 chars)}
🏆 Current: {milestone} ({version})
📍 Progress: {phases_completed} of {total_phases} phases
📅 Colony Age: {N} days
```

### Step 4: Show ASCII Art Anthill



Read the ASCII art file for the current milestone:
- First Mound → `.aether/visualizations/anthill-stages/first-mound.txt`
- Open Chambers → `.aether/visualizations/anthill-stages/open-chambers.txt`
- Brood Stable → `.aether/visualizations/anthill-stages/brood-stable.txt`
- Ventilated Nest → `.aether/visualizations/anthill-stages/ventilated-nest.txt`
- Sealed Chambers → `.aether/visualizations/anthill-stages/sealed-chambers.txt`
- Crowned Anthill → `.aether/visualizations/anthill-stages/crowned-anthill.txt`

Display the ASCII art with current milestone highlighted (bold/bright).


### Step 5: Show Journey Progress Bar

Display progress through all milestones:

```
Journey Progress:

[█░░░░░] First Mound        (0 phases)   - Complete
[██░░░░] Open Chambers      (1-3 phases) - Complete
[███░░░] Brood Stable       (4-6 phases) - Complete
[████░░] Ventilated Nest    (7-10 phases) - Current
[█████░] Sealed Chambers    (11-14 phases)
[██████] Crowned Anthill    (15+ phases)

Next: Ventilated Nest → Sealed Chambers
      Complete {N} more phases to advance
```

Calculate which milestones are complete vs current vs upcoming based on phases_completed.

### Step 6: Show Colony Statistics

Display summary stats:
```
Colony Statistics:
  🐜 Phases Completed: {phases_completed}
  📋 Total Phases: {total_phases}
  📅 Days Active: {colony_age_days}
  🏆 Current Milestone: {milestone}
  🎯 Completion: {percent}%
```

### Edge Cases



- If milestone file doesn't exist: Show error "Milestone visualization not found"

- If COLONY_STATE.json missing: "No colony initialized. Run /ant:init first."
- If phases_completed is 0: All milestones show as upcoming except First Mound


