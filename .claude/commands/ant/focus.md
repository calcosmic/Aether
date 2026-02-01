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

Extract the focus area from arguments:
```python
if not args:
    return """❌ Usage: /ant:focus "<area>"

Examples:
  /ant:focus "WebSocket security"
  /ant:focus "database optimization"
  /ant:focus "user authentication"
"""
```

## Step 2: Load Colony State

```python
import json
from datetime import datetime

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

pheromones = state.get('pheromones', [])
```

## Step 3: Create Focus Pheromone

```python
focus_pheromone = {
    "signal_type": "FOCUS",
    "content": " ".join(args),  # The focus area
    "strength": 0.7,  # Medium strength
    "created_at": datetime.now().isoformat(),
    "half_life_hours": 1.0,  # 1 hour half-life
    "is_active": True,
    "metadata": {
        "occurrences": 1,  # Track occurrences for learning
        "first_occurrence": datetime.now().isoformat()
    }
}
```

## Step 4: Check for Existing Focus on Same Topic

```python
focus_content = focus_pheromone['content'].lower()

# Check if there's already a focus on this topic
existing = None
for p in pheromones:
    if (p['signal_type'] == 'FOCUS' and
        p.get('is_active', True) and
        focus_content in p['content'].lower()):
        existing = p
        break

if existing:
    # Increment occurrences
    existing['metadata']['occurrences'] = existing['metadata'].get('occurrences', 1) + 1
    existing['strength'] = min(1.0, existing['strength'] + 0.1)  # Boost strength
    existing['created_at'] = datetime.now().isoformat()  # Refresh timestamp
else:
    # Add new focus pheromone
    pheromones.append(focus_pheromone)
```

## Step 5: Save Updated State

```python
state['pheromones'] = pheromones

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

## Step 6: Display Response

```
🐜 Queen Ant Colony - Focus Pheromone Emitted

"{focus_area}"

Signal: FOCUS (strength: 70%)
Duration: 1 hour half-life

COLONY RESPONDING:
  ✓ Executor prioritizing {focus_area}
  ✓ Planner considering {focus_area} in next plan
  ✓ Verifier focusing tests on {focus_area}
  ✓ Researcher prioritizing {focus_area} topics
```

If existing focus was found:
```
OCCURRENCES: {count}
  {count < 3}: Pattern emerging
  {count >= 3}: Preference learned
```

## Step 7: Show Next Steps

```
📋 NEXT STEPS:
  1. /ant:status            - Check colony response
  2. /ant:phase             - View phase with new focus
  3. /ant:memory            - View learned patterns

💡 FOCUS TIP:
   Repeated focuses on the same topic teach the colony
   your preferences. After 3+ occurrences, it becomes a
   learned preference.

🔄 CONTEXT: Safe to continue - colony is adjusting focus
```

</process>

<context>
@.aether/pheromone_system.py
@.aether/worker_ants.py

Focus Pheromone Properties:
- Type: FOCUS (medium attract)
- Default strength: 0.7 (70%)
- Half-life: 1 hour
- Effect: Guides colony attention without forcing

Worker Ant Response:
- Executor: Prioritizes focused area in task selection
- Planner: Incorporates focus into next phase plan
- Verifier: Intensifies testing in focused area
- Researcher: Prioritizes research in focused area

Learning:
- 3+ occurrences on same topic → Preference learned
- Stored in memory as learned pattern
- Influences future autonomous decisions
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
3. **Learning Effect**: After 3+ occurrences, becomes learned preference

## Colony Response by Caste

| Caste | Response to Focus |
|-------|------------------|
| Mapper | Explores focused area first |
| Planner | Plans tasks in focused area early |
| Executor | Prioritizes focused tasks |
| Verifier | Intensifies testing in focused area |
| Researcher | Researches focused topics first |
| Synthesizer | Extracts patterns from focused area |

## Focus vs Other Pheromones

| Pheromone | Strength | Duration | Effect |
|-----------|----------|----------|--------|
| INIT | 100% | Until phase complete | Triggers planning |
| FOCUS | 70% | 1hr half-life | Guides attention |
| REDIRECT | 70% | 24hr half-life | Warns away |
| FEEDBACK | 50% | 6hr half-life | Adjusts behavior |
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
