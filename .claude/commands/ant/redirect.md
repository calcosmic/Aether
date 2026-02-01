---
name: ant:redirect
description: Emit REDIRECT pheromone - Queen warns colony away from specific approaches
---

<objective>
Emit a REDIRECT pheromone to warn the colony away from specific approaches or patterns.

The REDIRECT pheromone is a strong repel signal (strength 0.9) with a 24-hour half-life.
It creates avoidance patterns for Builder, exclusion from Route-setter planning, and
validation constraints for Watcher.
</objective>

<process>
You are the **Queen Ant Colony** receiving a redirect command from the Queen.

## Step 1: Validate Input
Check if redirect pattern argument is provided:
```bash
if [ -z "$1" ]; then
  echo "Usage: /ant:redirect \"<pattern to avoid>\""
  echo ""
  echo "Example:"
  echo "  /ant:redirect \"synchronous patterns\""
  echo "  /ant:redirect \"blocking I/O operations\""
  echo "  /ant:redirect \"global state mutations\""
  exit 1
fi
```

## Step 2: Load State
Set the pheromones file path:
```bash
PHEROMONES=".aether/data/pheromones.json"
```

## Step 3: Create REDIRECT Pheromone
Create the REDIRECT pheromone object with timestamp and append to active_pheromones:
```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="redirect_$(date +%s)"

jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg pattern "$1" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "REDIRECT",
     "strength": 0.9,
     "created_at": $timestamp,
     "decay_rate": 86400,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $pattern
     }
   }]
   ' "$PHEROMONES" > /tmp/pheromones.tmp

# Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file "$PHEROMONES" /tmp/pheromones.tmp
```

## Step 4: Present Results
Show the Queen (user) the REDIRECT signal was emitted:

```
╔══════════════════════════════════════════════════════════════╗
║  REDIRECT Pheromone Emitted                                  ║
╠══════════════════════════════════════════════════════════════╣
║  Avoid: "{pattern}"                                           ║
║  Type: REDIRECT (repel signal)                                ║
║  Strength: 90%                                                ║
║  Half-Life: 24 hours                                          ║
║                                                               ║
║  Colony Response:                                             ║
║  ✓ Builder will avoid {pattern}                               ║
║  ✓ Route-setter will exclude from planning                    ║
║  ✓ Watcher will validate against constraint                   ║
╚══════════════════════════════════════════════════════════════╝

Colony will steer away from this approach.
Signal will decay over 24 hours.

Next Steps:
  /ant:status   - View all active pheromones
  /ant:focus    - Guide colony attention (optional)
```

</process>

<context>
# AETHER ARCHITECTURE - REDIRECT Pheromone

## REDIRECT Signal Characteristics

- **Type**: Strong repel signal
- **Default Strength**: 0.9 (90%)
- **Half-Life**: 24 hours (86400 seconds)
- **Decay Formula**: Strength(t) = 0.9 × e^(-t/86400)
- **Purpose**: Warn colony away from specific approaches or patterns

## Caste Sensitivity to REDIRECT

Different castes have different sensitivity to REDIRECT signals:

| Caste | REDIRECT Sensitivity | Effective Strength |
|-------|---------------------|-------------------|
| Colonizer | 0.9 | 0.81 |
| Route-setter | 0.8 | 0.72 |
| Builder | 0.7 | 0.63 |
| Watcher | 1.0 | 0.90 |
| Scout | 0.8 | 0.72 |
| Architect | 0.9 | 0.81 |

**Effective Strength** = Signal Strength × Caste Sensitivity

## Colony Behavior

### Builder Ant (Sensitivity: 0.7)
- Will avoid implementing redirected patterns
- Seeks alternative approaches when encountering redirected context
- Lower sensitivity allows flexibility when no alternatives exist

### Route-setter Ant (Sensitivity: 0.8)
- Excludes redirected patterns from phase planning
- Avoids creating tasks that require redirected approaches
- Medium sensitivity allows strategic exceptions

### Watcher Ant (Sensitivity: 1.0)
- Validates implementation against redirect constraints
- Highest sensitivity ensures redirected patterns are caught
- Will flag violations even in edge cases

### Colonizer Ant (Sensitivity: 0.9)
- Avoids indexing redirected patterns as favorable approaches
- High sensitivity prevents colony from "rediscovering" bad patterns

### Scout Ant (Sensitivity: 0.8)
- Avoids researching redirected approaches
- Seeks alternative information sources

### Architect Ant (Sensitivity: 0.9)
- Avoids compressing redirected patterns into long-term memory
- High sensitivity prevents bad patterns from becoming institutional knowledge

## Learning Patterns

After 3+ REDIRECT signals on the same pattern:
- Pattern added to `learning_patterns.redirect_constraints`
- Colony treats as permanent constraint (even after signal decays)
- Requires explicit Queen override to remove

## Signal Combinations

REDIRECT combines with other signals:

- **INIT + REDIRECT**: Goal established with avoidance patterns
- **FOCUS + REDIRECT**: Increased attention in area, but avoiding specific patterns
- **FEEDBACK + REDIRECT**: Strong behavioral adjustment - avoid this AND do that instead

## Examples

```bash
# Warn against synchronous patterns
/ant:redirect "synchronous patterns"

# Warn against blocking I/O
/ant:redirect "blocking I/O operations"

# Warn against global state
/ant:redirect "global state mutations"
```

</context>

<reference>
# Pheromone Schema

REDIRECT pheromone objects follow this schema:

```json
{
  "id": "redirect_1738400000",
  "type": "REDIRECT",
  "strength": 0.9,
  "created_at": "2026-02-01T15:25:00Z",
  "decay_rate": 86400,
  "metadata": {
    "source": "queen",
    "caste": null,
    "context": "pattern to avoid"
  }
}
```

Key fields:
- `decay_rate`: 86400 (24 hours in seconds) - determines half-life
- `strength`: 0.9 (90%) - default REDIRECT strength
- `metadata.context`: The pattern or approach to avoid
- `metadata.caste`: null for Queen signals (Worker Ants set caste when emitting)
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
