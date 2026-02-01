---
name: ant:adjust
description: Adjust pheromones during check-in before approving phase continuation
---

<objective>
Allow the Queen to modify pheromones during a check-in before approving the phase to continue. This enables the Queen to guide the next phase based on observations from the completed phase.
</objective>

<process>
You are the **Queen Ant Colony** adjusting pheromones during check-in.

## Step 1: Validate Input and Check-In Status

```bash
COLONY_STATE=".aether/data/COLONY_STATE.json"
PHEROMONES_FILE=".aether/data/pheromones.json"

# Source atomic-write utility
source .aether/utils/atomic-write.sh

# Check if there's an active check-in
checkin_status=$(jq -r '.colony_status.queen_checkin.status // "none"' "$COLONY_STATE")

if [ "$checkin_status" != "awaiting_review" ]; then
    echo "No active check-in. Pheromone adjustment only available during check-in."
    echo ""
    echo "Current state: $(jq -r '.colony_status.state' "$COLONY_STATE")"
    echo ""
    echo "To emit pheromones outside check-in, use:"
    echo "  - /ant:focus \"area\""
    echo "  - /ant:redirect \"pattern\""
    echo "  - /ant:feedback \"message\""
    exit 1
fi
```

## Step 2: Parse Arguments

```bash
# Parse arguments
if [ $# -lt 2 ]; then
    echo "Usage: /ant:adjust [focus|redirect|feedback] \"area/pattern\" [strength]"
    echo ""
    echo "Examples:"
    echo "  /ant:adjust focus \"database schema\" 0.9"
    echo "  /ant:adjust redirect \"api endpoints\" 0.8"
    exit 1
fi

pheromone_type="$1"
area="$2"
strength="${3:-0.8}"

# Validate pheromone type
case "$pheromone_type" in
    focus|redirect|feedback)
        ;;
    *)
        echo "Invalid pheromone type: $pheromone_type"
        echo "Valid types: focus, redirect, feedback"
        exit 1
        ;;
esac
```

## Step 3: Get Check-In Phase

```bash
# Get the phase being checked in
checkin_phase=$(jq -r '.colony_status.queen_checkin.phase' "$COLONY_STATE")

echo "Queen Adjustment: $pheromone_type"
echo ""
echo "Adjusting pheromone for phase $checkin_phase check-in."
echo ""
```

## Step 4: Emit the Appropriate Pheromone

```bash
# Emit the appropriate pheromone based on type
case "$pheromone_type" in
    focus)
        # Emit FOCUS pheromone
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        pheromone_id="focus_$(date +%s)"

        jq --arg id "$pheromone_id" \
           --arg timestamp "$timestamp" \
           --arg focus "$area" \
           --argjson strength "$strength" \
           '
           .active_pheromones += [{
             "id": $id,
             "type": "FOCUS",
             "strength": $strength,
             "created_at": $timestamp,
             "decay_rate": 3600,
             "metadata": {
               "source": "queen",
               "caste": null,
               "context": $focus
             }
           }]
           ' "$PHEROMONES_FILE" > /tmp/pheromones.tmp

        atomic_write_from_file "$PHEROMONES_FILE" /tmp/pheromones.tmp
        rm -f /tmp/pheromones.tmp
        ;;
    redirect)
        # Emit REDIRECT pheromone
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        pheromone_id="redirect_$(date +%s)"

        jq --arg id "$pheromone_id" \
           --arg timestamp "$timestamp" \
           --arg redirect "$area" \
           --argjson strength "$strength" \
           '
           .active_pheromones += [{
             "id": $id,
             "type": "REDIRECT",
             "strength": $strength,
             "created_at": $timestamp,
             "decay_rate": 86400,
             "metadata": {
               "source": "queen",
               "caste": null,
               "context": $redirect
             }
           }]
           ' "$PHEROMONES_FILE" > /tmp/pheromones.tmp

        atomic_write_from_file "$PHEROMONES_FILE" /tmp/pheromones.tmp
        rm -f /tmp/pheromones.tmp
        ;;
    feedback)
        # Emit FEEDBACK pheromone
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        pheromone_id="feedback_$(date +%s)"

        jq --arg id "$pheromone_id" \
           --arg timestamp "$timestamp" \
           --arg feedback "$area" \
           --argjson strength "$strength" \
           '
           .active_pheromones += [{
             "id": $id,
             "type": "FEEDBACK",
             "strength": $strength,
             "created_at": $timestamp,
             "decay_rate": 21600,
             "metadata": {
               "source": "queen",
               "caste": null,
               "context": $feedback
             }
           }]
           ' "$PHEROMONES_FILE" > /tmp/pheromones.tmp

        atomic_write_from_file "$PHEROMONES_FILE" /tmp/pheromones.tmp
        rm -f /tmp/pheromones.tmp
        ;;
esac
```

## Step 5: Present Results

```
PHEROMONE ADJUSTED

Type: {pheromone_type}
Context: "{area}"
Strength: {strength}

Check-in remains active.

Next Steps:
  - /ant:adjust [type] "area" [strength]  - Make more adjustments
  - /ant:continue                       - Approve and continue
  - /ant:execute {checkin_phase}        - Retry this phase

QUEEN ADJUSTMENT RECORDED
```

</process>

<context>
# AETHER PHEROMONE ADJUSTMENT - Check-In Context

## When to Use /ant:adjust

Use `/ant:adjust` during a check-in (when queen_checkin.status is "awaiting_review") to:

1. **Focus next phase attention**: Emit FOCUS pheromone for areas you want the next phase to prioritize
2. **Redirect from bad patterns**: Emit REDIRECT pheromone for approaches that should be avoided
3. **Provide feedback**: Emit FEEDBACK pheromone with observations or guidance

## Check-In Adjustments vs Normal Pheromones

| Aspect | /ant:adjust | /ant:focus /ant:redirect /ant:feedback |
|--------|-------------|----------------------------------------|
| **When Available** | Only during check-in | Anytime colony is not EXECUTING |
| **Check-In Status** | Must be "awaiting_review" | Any status |
| **CHECKIN Pheromone** | Preserved (not cleared) | N/A (no check-in active) |
| **Purpose** | Guide next phase before continuing | Guide current phase work |

## Why /ant:adjust Instead of Direct Commands

Adjusting during check-in is a special context:
- Queen is reviewing phase results
- Queen wants to guide the **next** phase based on observations
- Check-in remains active until Queen uses `/ant:continue`
- Multiple adjustments can be made before continuing

## Command Flow

```
Check-in Active (awaiting_review)
    ↓
/ant:adjust focus "database optimization" 0.9
    ↓
FOCUS pheromone emitted
Check-in STILL ACTIVE (not cleared)
    ↓
/ant:adjust redirect "monolithic architecture" 0.8
    ↓
REDIRECT pheromone emitted
Check-in STILL ACTIVE
    ↓
/ant:continue
    ↓
CHECKIN pheromone cleared
New FOCUS and REDIRECT persist into next phase
```

## Pheromone Types

### FOCUS (adjust focus)
- **Purpose**: Guide colony attention to specific area
- **Strength**: 0.0 to 1.0 (default 0.8)
- **Half-Life**: 1 hour
- **Effect**: Builder prioritizes focused area

### REDIRECT (avoid pattern)
- **Purpose**: Warn colony away from approach
- **Strength**: 0.0 to 1.0 (default 0.8)
- **Half-Life**: 24 hours
- **Effect**: Colony avoids specified pattern

### FEEDBACK (provide guidance)
- **Purpose**: Adjust colony behavior based on observations
- **Strength**: 0.0 to 1.0 (default 0.8)
- **Half-Life**: 6 hours
- **Effect**: Colony adjusts behavior
</context>

<reference>
# /ant:adjust Examples

## Example 1: Focus Next Phase on Database

```
State: VERIFYING
Check-in: awaiting_review (Phase 4: Triple-Layer Memory complete)

Queen observation: "Database queries are slow in current implementation."

/ant:adjust focus "database query optimization" 0.9

↓ Result

FOCUS pheromone emitted:
  Type: FOCUS
  Context: "database query optimization"
  Strength: 0.9

Check-in still active. Next phase will prioritize database optimization.
```

## Example 2: Redirect from Bad Pattern

```
State: VERIFYING
Check-in: awaiting_review

Queen observation: "Monolithic approach caused problems."

/ant:adjust redirect "monolithic architecture" 0.8

↓ Result

REDIRECT pheromone emitted:
  Type: REDIRECT
  Context: "monolithic architecture"
  Strength: 0.8

Colony will avoid monolithic patterns in next phase.
```

## Example 3: No Check-In Active

```
State: EXECUTING
Check-in: none

/ant:adjust focus "testing" 0.9

↓ Result

"No active check-in. Pheromone adjustment only available during check-in."

Suggested alternatives:
  - /ant:focus "testing"
  - /ant:redirect "pattern"
  - /ant:feedback "message"
```

## Example 4: Multiple Adjustments

```
Check-in: awaiting_review

/ant:adjust focus "authentication" 0.9
/ant:adjust redirect "plain text passwords" 1.0
/ant:adjust feedback "Good progress on API design" 0.7

↓ Result

3 pheromones emitted, check-in still active.

/ant:continue

↓ Result

CHECKIN cleared, 3 new pheromones persist into next phase.
```
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
