---
name: ant:feedback
description: Emit FEEDBACK pheromone - provide guidance to colony based on observations
---

<objective>
Emit a FEEDBACK pheromone to guide colony behavior based on Queen's observations, preferences, or corrections.

The FEEDBACK pheromone has a 6-hour half-life and adjusts colony behavior through caste-sensitive responses.
</objective>

<process>
You are the **Queen Ant Colony** receiving feedback from the Queen.

## Step 1: Validate Input

```bash
if [ -z "$1" ]; then
  echo "❌ Usage: /ant:feedback \"<message>\""
  echo ""
  echo "Examples:"
  echo "  /ant:feedback \"Great progress on API layer\""
  echo "  /ant:feedback \"Need more test coverage\""
  echo "  /ant:feedback \"Too slow, speed up\""
  echo "  /ant:feedback \"This approach is wrong\""
  exit 1
fi
```

## Step 2: Load State

```bash
PHEROMONES=".aether/data/pheromones.json"
```

## Step 3: Create FEEDBACK Pheromone

```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="feedback_$(date +%s)"

jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg context "$1" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "FEEDBACK",
     "strength": 0.5,
     "created_at": $timestamp,
     "decay_rate": 21600,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $context
     }
   }]
   ' "$PHEROMONES" > /tmp/pheromones.tmp
```

## Step 4: Atomic Write

```bash
# Source atomic-write utility and use atomic_write_from_file
source .aether/utils/atomic-write.sh
atomic_write_from_file "$PHEROMONES" /tmp/pheromones.tmp
```

## Step 5: Display Results

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 FEEDBACK Pheromone Emitted                                ║
╠══════════════════════════════════════════════════════════════╣
║  Message: "{feedback}"                                        ║
║  Type: FEEDBACK (guidance signal)                             ║
║  Strength: 50%                                                ║
║  Half-Life: 6 hours                                          ║
║                                                               ║
║  Colony Response:                                             ║
║  ✓ All castes will adjust based on feedback                  ║
║  ✓ Architect will record pattern for learning                 ║
║  ✓ Future decisions will consider guidance                    ║
╚══════════════════════════════════════════════════════════════╝
```

</process>

<context>
# AETHER FEEDBACK PHEROMONE SYSTEM

## FEEDBACK Pheromone Properties
- **Type**: FEEDBACK
- **Default Strength**: 0.5
- **Half-Life**: 6 hours (21600 seconds)
- **Effect**: Adjusts colony behavior based on Queen observations

## Caste Sensitivities

Each caste responds differently to FEEDBACK signals:

| Caste | Sensitivity | Response |
|-------|-------------|----------|
| Colonizer | 0.7 | Moderate - adjusts exploration patterns |
| Route-setter | 0.8 | Responds - adjusts planning approach |
| Builder | 0.9 | Strong response - modifies implementation |
| Watcher | 1.0 | Very strong - intensifies verification |
| Scout | 0.8 | Responds - adjusts information gathering |
| Architect | 1.0 | Very strong - records for learning |

## Feedback Examples

### Positive Feedback
```
/ant:feedback "Great progress on API layer"
```
- Effect: Pattern reinforced for reuse
- Architect: Records positive pattern
- Colony: Continues current approach

### Quality Feedback
```
/ant:feedback "Need more test coverage"
/ant:feedback "Quality issues in authentication"
```
- Effect: Testing intensified
- Watcher: Increases verification
- Builder: Reviews recent code

### Speed Feedback
```
/ant:feedback "Too slow, speed up"
```
- Effect: Optimizes for speed
- Builder: Increases parallelization
- Route-setter: Simplifies tasks

### Direction Feedback
```
/ant:feedback "This approach is wrong"
/ant:feedback "Need to pivot architecture"
```
- Effect: Pivots approach
- Route-setter: Replans with new direction
- Builder: Adjusts implementation

## Learning Integration

FEEDBACK pheromones are recorded in memory.json for pattern learning:
- 3+ similar feedback → preference/constraint established
- Architect caste analyzes feedback history
- Patterns influence future autonomous decisions

</context>

<reference>
# FEEDBACK Signal Schema

```json
{
  "id": "feedback_1234567890",
  "type": "FEEDBACK",
  "strength": 0.5,
  "created_at": "2025-02-01T12:00:00Z",
  "decay_rate": 21600,
  "metadata": {
    "source": "queen",
    "caste": null,
    "context": "Great progress on API layer"
  }
}
```

# Decay Calculation

After 6 hours: strength × 0.5 = 0.25
After 12 hours: strength × 0.25 = 0.125
After 18 hours: strength × 0.125 = 0.0625

Worker Ants interpret decay on-read based on time elapsed.
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
