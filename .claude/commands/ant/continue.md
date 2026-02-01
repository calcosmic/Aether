---
name: ant:continue
description: Queen approves phase completion and clears check-in for colony to proceed
---

<objective>
Approve the phase check-in and clear the CHECKIN pheromone, allowing the colony to proceed to the next phase.
</objective>

<process>
You are the **Queen Ant Colony** approving a phase check-in.

## Step 1: Check for Active Check-In

Verify there's an active check-in awaiting review:

```bash
COLONY_STATE=".aether/data/COLONY_STATE.json"
PHEROMONES_FILE=".aether/data/pheromones.json"

# Check if there's an active check-in
checkin_status=$(jq -r '.colony_status.queen_checkin.status // "none"' "$COLONY_STATE")

if [ "$checkin_status" != "awaiting_review" ]; then
    echo "No active check-in. Colony is not paused."
    echo ""
    echo "Current state: $(jq -r '.colony_status.state' "$COLONY_STATE")"
    exit 0
fi
```

## Step 2: Get Check-In Phase

```bash
# Get the phase being checked in
checkin_phase=$(jq -r '.colony_status.queen_checkin.phase' "$COLONY_STATE")

echo "Queen Decision: CONTINUE"
echo ""
echo "Approving phase $checkin_phase completion. Colony will proceed."
echo ""
```

## Step 3: Clear CHECKIN Pheromone

```bash
# Source atomic-write utility
source .aether/utils/atomic-write.sh

# Clear CHECKIN pheromone
jq '(.active_pheromones |= map(select(.type != "CHECKIN")))' "$PHEROMONES_FILE" > /tmp/pheromones.tmp
atomic_write_from_file "$PHEROMONES_FILE" /tmp/pheromones.tmp
rm -f /tmp/pheromones.tmp
```

## Step 4: Update Check-In Status

```bash
# Update queen_checkin status
jq --arg decision "continue" \
   --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
   '.colony_status.queen_checkin = {
     "phase": .colony_status.queen_checkin.phase,
     "status": "approved",
     "timestamp": $timestamp,
     "queen_decision": $decision
   }' "$COLONY_STATE" > /tmp/state.tmp

atomic_write_from_file "$COLONY_STATE" /tmp/state.tmp
rm -f /tmp/state.tmp
```

## Step 5: Transition to COMPLETED State

```bash
# Transition to COMPLETED state (phase boundary passed)
# Note: In full implementation, Worker Ants will transition to next phase's INIT
# For now, transition to COMPLETED to indicate check-in resolved
jq '.colony_status.state = "COMPLETED"' "$COLONY_STATE" > /tmp/state.tmp
atomic_write_from_file "$COLONY_STATE" /tmp/state.tmp
rm -f /tmp/state.tmp
```

## Step 6: Present Results

```
COLONY CHECK-IN APPROVED

Check-in cleared. Colony may proceed to next phase.

Phase Summary:
  Phase: {checkin_phase}
  Decision: continue
  Timestamp: {timestamp}

Next Steps:
  - /ant:phase {checkin_phase}  - Review phase summary
  - /ant:init {next_phase}      - Initialize next phase
  - /ant:status                - View colony status

QUEEN DECISION RECORDED
```

</process>

<context>
# AETHER PHASE BOUNDARY SYSTEM - Queen Check-In

## Phase Boundary Flow

```
EXECUTING State
    ↓ (all tasks complete)
check_phase_boundary() detects boundary
    ↓
emit_checkin_pheromone()
    ↓
Transition to VERIFYING State
    ↓
await_queen_decision() sets queen_checkin.status = "awaiting_review"
    ↓
Colony PAUSED (awaiting Queen decision)
    ↓
Queen reviews phase with /ant:phase {phase}
    ↓
Queen decides: /ant:continue OR /ant:adjust OR /ant:execute {phase}
    ↓
Check-in cleared, colony proceeds
```

## Queen Options at Check-In

| Option | Command | Effect |
|--------|---------|--------|
| **Continue** | `/ant:continue` | Approves phase, clears CHECKIN, proceeds to next phase |
| **Adjust** | `/ant:adjust` | Modifies pheromones before continuing (check-in remains active) |
| **Retry** | `/ant:execute {phase}` | Re-executes phase with different approach |

## CHECKIN Pheromone

- **Type**: CHECKIN
- **Strength**: 1.0 (maximum priority)
- **Decay Rate**: null (persists until Queen decision)
- **Purpose**: Signals Queen that phase is complete and needs review
- **Cleared By**: `/ant:continue` command

## Why No Separate `/ant:retry`

The existing `/ant:execute {phase}` command already provides retry functionality. Queen can re-execute any phase with `/ant:execute {phase_number}`.

## Colony State Transitions

During check-in process:
1. **EXECUTING → VERIFYING**: Phase complete, check-in triggered
2. **VERIFYING (awaiting_review)**: Colony paused, Queen reviewing
3. **VERIFYING → COMPLETED**: Queen approved via `/ant:continue`
4. **COMPLETED → IDLE/INIT**: Ready for next phase

## queen_checkin Schema

```json
{
  "colony_status": {
    "queen_checkin": {
      "phase": "5",
      "status": "awaiting_review | approved | adjusted",
      "timestamp": "2026-02-01T15:00:00Z",
      "queen_decision": "continue | adjust | retry"
    }
  }
}
```
</context>

<reference>
# Continue Command Examples

## Normal Flow: Phase Complete and Approved

```
State: VERIFYING
Check-in: awaiting_review

/ant:continue
    ↓
Clear CHECKIN pheromone
Update queen_checkin.status = "approved"
Transition state: VERIFYING → COMPLETED
    ↓
State: COMPLETED
Check-in: approved
Colony ready for next phase
```

## No Active Check-In

```
State: EXECUTING
Check-in: none

/ant:continue
    ↓
"No active check-in. Colony is not paused."
Current state: EXECUTING
```

## After Multiple Adjustments

```
State: VERIFYING
Check-in: awaiting_review
Pheromones: FOCUS (added via /ant:adjust)

/ant:continue
    ↓
Clear CHECKIN pheromone
Keep FOCUS pheromone (it persists independently)
Update queen_checkin.status = "approved"
Transition state: VERIFYING → COMPLETED
```
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
