---
name: ant:redirect
description: Emit redirect pheromone - warn colony away from approach or pattern
---

<objective>
Emit a redirect pheromone (strong repel signal) to warn the colony away from a specific approach, pattern, or technology.
</objective>

<process>
You are the **Queen Ant Colony** emitting a redirect pheromone to warn the colony.

## Step 1: Validate Input

```python
if not args:
    return """❌ Usage: /ant:redirect "<pattern>"

Examples:
  /ant:redirect "Don't use string concatenation for SQL"
  /ant:redirect "Avoid callbacks, use async/await"
  /ant:redirect "Don't use MongoDB for this"
"""
```

## Step 2: Load Colony State

```python
import json
from datetime import datetime

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

pheromones = state.get('pheromones', [])
error_ledger = state.get('error_ledger', {})
```

## Step 3: Create Redirect Pheromone

```python
pattern = " ".join(args)

redirect_pheromone = {
    "signal_type": "REDIRECT",
    "content": pattern,
    "strength": 0.7,  # Strong repel
    "created_at": datetime.now().isoformat(),
    "half_life_hours": 24.0,  # 24 hour half-life
    "is_active": True,
    "metadata": {
        "occurrences": 1,
        "first_occurrence": datetime.now().isoformat(),
        "pattern": pattern.lower()
    }
}
```

## Step 4: Check for Existing Redirect on Same Pattern

```python
pattern_lower = redirect_pheromone['content'].lower()

existing = None
for p in pheromones:
    if (p['signal_type'] == 'REDIRECT' and
        p.get('is_active', True) and
        pattern_lower in p['content'].lower()):
        existing = p
        break

if existing:
    # Increment occurrences
    occurrences = existing['metadata'].get('occurrences', 1) + 1
    existing['metadata']['occurrences'] = occurrences
    existing['strength'] = min(1.0, existing['strength'] + 0.1)
    existing['created_at'] = datetime.now().isoformat()

    # After 3 occurrences, create constraint
    if occurrences >= 3:
        # Add to error ledger as FLAGGED_ISSUE
        error_ledger[pattern_lower] = {
            "category": "constraint",
            "pattern": pattern,
            "occurrences": occurrences,
            "created_at": existing['metadata']['first_occurrence'],
            "became_constraint_at": datetime.now().isoformat(),
            "severity": "high",
            "prevention": "Avoid this pattern - flagged by Queen after 3 redirects"
        }

        state['error_ledger'] = error_ledger
else:
    # Add new redirect pheromone
    pheromones.append(redirect_pheromone)
    occurrences = 1
```

## Step 5: Save Updated State

```python
state['pheromones'] = pheromones

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

## Step 6: Display Response

```
🐜 Queen Ant Colony - Redirect Pheromone Emitted

"{pattern}"

Signal: REDIRECT (strength: 70%)
Duration: 24 hours

COLONY RESPONDING:
  ✓ Executor avoiding {pattern}
  ✓ Planner avoiding in future plans
  ✓ Verifier validating against {pattern}
```

Show occurrence status:
```python
if occurrences == 1:
    occurrence_msg = "OCCURRENCES: 1/3 (will become constraint after 3)"
elif occurrences == 2:
    occurrence_msg = "OCCURRENCES: 2/3 (one more becomes constraint)"
elif occurrences >= 3:
    occurrence_msg = f"⚠️  CONSTRAINT CREATED: '{pattern}' now enforced"
```

## Step 7: Show Learning Progress

```
LEARNING PROGRESS:
  Occurrence 1: Logged in ERROR_LEDGER
  Occurrence 2: Pattern detected
  Occurrence 3: ⚠️  FLAGGED_ISSUE created, constraint enforced

After 3 occurrences, the pattern becomes a permanent constraint
that validates BEFORE execution.
```

## Step 8: Show Next Steps

```
📋 NEXT STEPS:
  1. /ant:status            - Check colony response
  2. /ant:memory            - View learned patterns/constraints
  3. /ant:focus <area>      - Guide toward alternative approach

💡 REDIRECT TIP:
   Use redirect to warn the colony away from patterns you
   don't want. After 3 redirects on the same pattern, it
   becomes a permanent constraint.

🔄 CONTEXT: Safe to continue - colony is avoiding pattern
```

</process>

<context>
@.aether/pheromone_system.py
@.aether/error_prevention.py
@.aether/worker_ants.py

Redirect Pheromone Properties:
- Type: REDIRECT (strong repel)
- Default strength: 0.7 (70%)
- Half-life: 24 hours
- Effect: Warns colony away from pattern

Learning Progression:
1. **Occurrence 1**: Logged in ERROR_LEDGER
2. **Occurrence 2**: Pattern detected, warning issued
3. **Occurrence 3+**: FLAGGED_ISSUE created, constraint enforced

Constraint Behavior:
- Added to error_ledger with category "constraint"
- Validated BEFORE execution
- Executor must avoid or get explicit warning
- Planner excludes from future plans
- Verifier checks against constraint
</context>

<reference>
# Redirect Examples

## Common Redirects

```bash
# Security
/ant:redirect "Don't use string concatenation for SQL"
/ant:redirect "Avoid hardcoded credentials"
/ant:redirect "Don't store passwords in plain text"

# Architecture
/ant:redirect "Avoid callbacks, use async/await"
/ant:redirect "Don't use MongoDB for this project"
/ant:redirect "Avoid monolithic structure"

# Code Quality
/ant:redirect "Don't use var, use const/let"
/ant:redirect "Avoid any types"
/ant:redirect "Don't skip tests"
```

## Constraint Creation Example

```
# User issues 3rd redirect
/ant:redirect "Don't use string concatenation for SQL"

# Output:
🐜 Queen Ant Colony - Redirect Pheromone Emitted

"Don't use string concatenation for SQL"

Signal: REDIRECT (strength: 70%)

COLONY RESPONDING:
  ✓ Executor avoiding string concatenation
  ✓ Planner using parameterized queries
  ✓ Verifier validating SQL before execution

⚠️  CONSTRAINT CREATED: 'string concatenation for SQL' now enforced

LEARNING PROGRESS:
  Occurrence 1: ✓ Logged in ERROR_LEDGER
  Occurrence 2: ✓ Pattern detected
  Occurrence 3: ✓ FLAGGED_ISSUE created, constraint enforced

Future SQL operations will be validated against this constraint.
```

## Worker Ant Response

| Caste | Response to Redirect |
|-------|---------------------|
| Mapper | Avoids mapping redirected patterns |
| Planner | Excludes from plans |
| Executor | Avoids implementation |
| Verifier | Validates against constraint |
| Researcher | Seeks alternatives |
| Synthesizer | Extracts avoidance pattern |
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
