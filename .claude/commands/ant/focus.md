---
name: ant:focus
description: Emit focus pheromone - guide colony attention to specific area
---

<objective>
Emit a focus pheromone (medium-strength attract signal) to guide the colony's attention toward a specific area, topic, or approach.
</objective>

<process>
You are the **Queen Ant Colony** emitting a focus pheromone to guide the colony.

## Step 1: Validate Input

Check if focus area argument is provided:
```bash
if [ -z "$1" ]; then
  echo "Usage: /ant:focus \"<area>\""
  echo ""
  echo "Examples:"
  echo "  /ant:focus \"WebSocket security\""
  echo "  /ant:focus \"database optimization\""
  echo "  /ant:focus \"user authentication\""
  exit 1
fi

focus_area="$1"
```

## Step 2: Emit FOCUS Pheromone

Create the FOCUS pheromone signal:
```bash
# Source atomic-write utility
source .aether/utils/atomic-write.sh

timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="focus_$(date +%s)"

jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg focus "$focus_area" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "FOCUS",
     "strength": 0.7,
     "created_at": $timestamp,
     "decay_rate": 3600,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $focus
     }
   }]
   ' .aether/data/pheromones.json > /tmp/pheromones.tmp

# Atomic write
atomic_write_from_file .aether/data/pheromones.json /tmp/pheromones.tmp
```

## Step 3: Present Results

Show the Queen (user) the focus pheromone emission:

```
╔══════════════════════════════════════════════════════════════╗
║  FOCUS Pheromone Emitted                                     ║
╠══════════════════════════════════════════════════════════════╣
║  Area: "{focus_area}"                                        ║
║  Type: FOCUS (attract signal)                                ║
║  Strength: 70%                                               ║
║  Half-Life: 1 hour                                          ║
║                                                               ║
║  Colony Response:                                            ║
║  Builder will prioritize {focus_area}                        ║
║  Route-setter will include in planning                       ║
║  Scout will research {focus_area} first                      ║
╚══════════════════════════════════════════════════════════════╝

COLONY RESPONDING

Next Steps:
  /ant:status   - View colony response to focus
  /ant:plan     - Show how focus influences planning
  /ant:redirect - Warn colony away from approaches (if needed)
```

</process>

<context>
# AETHER PHEROMONE SIGNAL SYSTEM - Claude Native Implementation

## Signal Decay Formula
```
Strength(t) = InitialStrength × e^(-t/HalfLife)
```

Where:
- InitialStrength: Signal strength at creation (0.0 to 1.0)
- t: Time elapsed since signal creation
- HalfLife: Time for signal to lose 50% strength

Example calculation for FOCUS (half-life = 1 hour):
- t=0: Strength = 0.7 × 1.0 = 0.7 (100%)
- t=30m: Strength = 0.7 × 0.5^0.5 = 0.49 (70%)
- t=1h: Strength = 0.7 × 0.5 = 0.35 (50%)
- t=2h: Strength = 0.7 × 0.25 = 0.175 (25%)
- t=4h: Strength = 0.7 × 0.0625 = 0.044 (~6%, expires)

## Signal Types and Properties

### INIT Signal
- **Purpose**: Set colony intention, trigger planning
- **Default Strength**: 1.0 (maximum)
- **Half-Life**: Persists (no decay until phase complete)
- **Effect**: Strong attract, mobilizes colony

### FOCUS Signal
- **Purpose**: Guide colony attention to specific area
- **Default Strength**: 0.7
- **Half-Life**: 1 hour (3600 seconds)
- **Effect**: Medium attract, guides prioritization
- **Caste Responses**:
  - Colonizer (sensitivity 0.8): Colonizes focused area first
  - Route-setter (sensitivity 0.9): Incorporates into priorities
  - Builder (sensitivity 1.0): Highly responsive, prioritizes focused work
  - Watcher (sensitivity 0.9): Intensifies testing
  - Scout (sensitivity 0.7): Researches focused topic first
  - Architect (sensitivity 0.8): Extracts patterns from focused area

### REDIRECT Signal
- **Purpose**: Warn colony away from approach/pattern
- **Default Strength**: 0.9
- **Half-Life**: 24 hours
- **Effect**: Strong repel, prevents bad patterns

### FEEDBACK Signal
- **Purpose**: Adjust colony behavior based on Queen's feedback
- **Default Strength**: 0.5-0.7 (variable based on category)
- **Half-Life**: 6 hours
- **Effect**: Variable, adjusts behavior

## Effective Strength Calculation

```
EffectiveStrength = SignalStrength(t) × CasteSensitivity
```

Example: FOCUS signal (strength 0.5 after decay)
- Colonizer: 0.5 × 0.8 = 0.40 (moderate response)
- Builder: 0.5 × 1.0 = 0.50 (strong response)
- Architect: 0.5 × 0.8 = 0.40 (moderate response)

Response threshold: 0.1 (below threshold, no response)
</context>

<reference>
# Focus Examples

## Correct Usage
```
/ant:focus "WebSocket security"
/ant:focus "database query optimization"
/ant:focus "user authentication flow"
/ant:focus "error handling"
```

## What Happens

1. **Immediate Effect**: Colony adjusts current work to prioritize focus area
2. **Lasting Effect**: Pheromone decays over 1 hour half-life
3. **Builder Response**: Highly responsive (sensitivity 1.0), prioritizes focused tasks

## Focus vs Other Pheromones

| Pheromone | Strength | Duration | Effect |
|-----------|----------|----------|--------|
| INIT | 100% | Until phase complete | Triggers planning |
| FOCUS | 70% | 1hr half-life | Guides attention |
| REDIRECT | 90% | 24hr half-life | Warns away |
| FEEDBACK | 50% | 6hr half-life | Adjusts behavior |
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
