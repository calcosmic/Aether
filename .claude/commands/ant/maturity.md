---
name: ant:maturity
description: "👑🐜🏛️🐜👑 View colony maturity journey with ASCII art anthill"
---

You are the **Queen**. Display the colony's maturity journey.

## Instructions

### Step 1: Detect Current Milestone

Run using the Bash tool with description "Detecting colony milestone...": `bash .aether/aether-utils.sh milestone-detect`

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
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal: {goal (truncated to 50 chars)}
🏆 Current: {milestone} ({version})
📍 Progress: {phases_completed} of {total_phases} phases
📅 Colony Age: {N} days
```

### Step 4: Show Milestone Description

Display the current milestone with its text description (no external file needed):

- First Mound -- "A small pile of earth. The colony has broken ground."
- Open Chambers -- "Tunnels branch outward. Feature work is underway."
- Brood Stable -- "The nursery hums. Tests are consistently green."
- Ventilated Nest -- "Air flows freely. Performance and latency are acceptable."
- Sealed Chambers -- "Walls are hardened. Interfaces are frozen."
- Crowned Anthill -- "The spire rises. The colony is release-ready."

Display the matching description for the current milestone.

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

- If milestone name is unrecognized: Show "Unknown milestone" with the raw name
- If COLONY_STATE.json missing: "No colony initialized. Run /ant:init first."
- If phases_completed is 0: All milestones show as upcoming except First Mound

### Step 7: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
