---
name: ant:memory
description: View Queen Ant Colony memory - learned preferences, pheromone patterns
---

<objective>
Display colony's learned patterns from pheromone signals including focus topics, avoid patterns, and feedback categories.
</objective>

<process>
You are the **Queen Ant Colony** displaying learned patterns and memory.

## Step 1: Load Colony State

```python
import json
from datetime import datetime
from collections import Counter

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

pheromones = state.get('pheromones', [])
feedback_history = state.get('feedback_history', {})
error_ledger = state.get('error_ledger', {})
learned_patterns = state.get('learned_patterns', {})
```

## Step 2: Display Header

```
🐜 Queen Ant Colony - Learned Patterns & Memory

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 3: Analyze Focus Patterns

```python
# Group focus pheromones by topic
focus_topics = []

for p in pheromones:
    if p['signal_type'] == 'FOCUS':
        content = p['content'].lower()
        occurrences = p.get('metadata', {}).get('occurrences', 1)
        focus_topics.append((content, occurrences))

# Count occurrences
topic_counts = Counter([topic for topic, _ in focus_topics])
```

Display:

```
LEARNED PREFERENCES:

FOCUS TOPICS:
```

```python
if topic_counts:
    for topic, count in topic_counts.most_common():
        if count >= 3:
            print(f"  ✓ {topic} ({count} occurrences) - PREFERENCE LEARNED")
        elif count == 2:
            print(f"  • {topic} ({count} occurrences) - Pattern emerging")
        else:
            print(f"  • {topic} ({count} occurrence)")
else:
    print("  No focus patterns yet")
```

## Step 4: Analyze Avoid Patterns (Redirects)

```python
# Group redirect pheromones
avoid_patterns = []

for p in pheromones:
    if p['signal_type'] == 'REDIRECT':
        content = p['content'].lower()
        occurrences = p.get('metadata', {}).get('occurrences', 1)
        avoid_patterns.append((content, occurrences))

pattern_counts = Counter([pattern for pattern, _ in avoid_patterns])
```

Display:

```
AVOID PATTERNS:
```

```python
if pattern_counts:
    for pattern, count in pattern_counts.most_common():
        if count >= 3:
            print(f"  ✓ {pattern} ({count} occurrences) - CONSTRAINT ENFORCED")
        elif count == 2:
            print(f"  • {pattern} ({count} occurrences) - One more becomes constraint")
        else:
            print(f"  • {pattern} ({count} occurrence)")
else:
    print("  No avoid patterns yet")
```

## Step 5: Display Constraints from Error Ledger

```python
constraints = {k: v for k, v in error_ledger.items() if v.get('category') == 'constraint'}
```

```
ACTIVE CONSTRAINTS:
```

```python
if constraints:
    for pattern, constraint in constraints.items():
        print(f"  ⚠️  {constraint.get('pattern', pattern)}")
        print(f"    Created: {constraint.get('became_constraint_at', 'N/A')}")
        print(f"    Severity: {constraint.get('severity', 'high').upper()}")
else:
    print("  No constraints yet (need 3+ redirects on same pattern)")
```

## Step 6: Display Feedback Summary

```
FEEDBACK CATEGORIES:
```

```python
if feedback_history:
    for category, data in feedback_history.items():
        total = data.get('count', 0)
        positive = data.get('positive', 0)
        negative = data.get('negative', 0)

        print(f"  {category.upper()}: {total} total")

        if positive > 0:
            print(f"    {positive} positive")
        if negative > 0:
            print(f"    {negative} negative")

        if positive >= 5:
            print(f"    → Best practice established")

        if category == 'quality' and negative >= 3:
            print(f"    → Quality intensified")

        # Show recent feedback
        recent = data.get('recent', [])
        if recent:
            print(f"    Recent: '{recent[-1]['message']}'")
else:
    print("  No feedback history yet")
```

## Step 7: Display Learning Status

```
LEARNING STATUS:
```

```python
# Calculate learning metrics
total_focus = sum(topic_counts.values())
strong_patterns = sum(1 for count in topic_counts.values() if count >= 3)
total_redirect = sum(pattern_counts.values())
constraints_count = len(constraints)
total_feedback = sum(data.get('count', 0) for data in feedback_history.values())

print(f"  Focus patterns learned: {strong_patterns}")
print(f"  Constraints enforced: {constraints_count}")
print(f"  Total feedback processed: {total_feedback}")
print(f"  Colony maturity: {'early': 'EARLY', 'developing': 'DEVELOPING', 'mature': 'MATURE'}.get(
    'mature' if strong_patterns >= 5 else 'developing' if strong_patterns >= 2 else 'early'
)}")
```

## Step 8: Display How Learning Works

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

HOW LEARNING WORKS:

Focus Pheromones:
  • 1 occurrence: Topic noted
  • 2 occurrences: Pattern emerging
  • 3+ occurrences: Preference learned

Redirect Pheromones:
  • 1 occurrence: Logged in ERROR_LEDGER
  • 2 occurrences: Pattern detected
  • 3+ occurrences: Constraint created and enforced

Feedback:
  • 5+ positive: Best practice established
  • 3+ quality issues: Verifier intensifies
  • 3+ speed issues: Optimization prioritized
```

## Step 9: Display Next Steps

```
📋 NEXT STEPS:

  1. /ant:status            - Check colony status
  2. /ant:focus <area>      - Add focus (teaches preferences)
  3. /ant:redirect <pattern> - Avoid pattern (teaches constraints)
  4. /ant:feedback "<msg>"   - Provide feedback

💡 MEMORY TIP:
   Colony learns from your signals over time.
   Repeated patterns become learned preferences that
   influence autonomous decisions.

🔄 CONTEXT: Safe to continue - memory display only
```

</process>

<context>
@.aether/pheromone_system.py
@.aether/memory/triple_layer_memory.py
@.aether/error_prevention.py

Learning Mechanisms:
- **Focus patterns**: 3+ occurrences → learned preference
- **Redirect patterns**: 3+ occurrences → constraint enforced
- **Feedback patterns**: 5+ positive → best practice

Memory Storage:
- Short-term: Recent pheromones and feedback
- Long-term: Learned patterns and constraints
- Error ledger: Constraints and flagged issues
</context>

<reference>
# `/ant:memory` - View Colony Memory

## What It Shows

Displays the colony's learned patterns and preferences from pheromone signals.

## Memory Sections

### 1. Learned Preferences

Shows what the colony has learned from Queen's pheromone patterns:

```
🧠 LEARNED PREFERENCES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

FOCUS TOPICS (What Queen prioritizes)
  WebSocket security (3 occurrences)
  message reliability (2 occurrences)
  authentication (1 occurrence)
  test coverage (1 occurrence)

AVOID PATTERNS (What Queen redirects away from)
  string concatenation for SQL (2 occurrences) → ⚠️ One more becomes constraint
  callback patterns (1 occurrence)
  MongoDB for this project (1 occurrence)

FEEDBACK CATEGORIES
  Quality: 12 positive, 3 negative
  Speed: 5 "too slow", 8 "good pace"
  Direction: 2 "wrong approach" corrections
```

### 2. Best Practices

Shows best practices learned from successful executions:

```
✅ BEST PRACTICES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SPAWNING
  [bp-1] Spawn specialists when capability gap > 30%
  [bp-2] Max subagent depth 3 prevents complexity explosion
  [bp-3] Terminating subagents after task completion frees resources

COMMUNICATION
  [bp-4] Peer-to-peer coordination reduces bottleneck
  [bp-5] Pheromone signals guide without commands
  [bp-6] Focus on areas, not specific implementations

EXECUTION
  [bp-7] Implement in priority order based on focus pheromones
  [bp-8] Test critical paths before edge cases
  [bp-9] Compress memory between phases
```

### 3. Anti-Patterns

Shows what to avoid (learned from redirects and errors):

```
❌ ANTI-PATTERNS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

APPROACHES TO AVOID
  [ap-1] String concatenation for SQL (security risk)
  [ap-2] Callback hell (use async/await)
  [ap-3] Monolithic architecture (prevents parallel development)

BEHAVIORS TO AVOID
  [ap-4] Spawning without clear purpose (resource waste)
  [ap-5] Ignoring redirect pheromones (leads to constraints)
  [ap-6] Exceeding subagent depth limit (confusion)
```

## Pheromone Learning

The colony learns from pheromone patterns:

### Focus Learning

```
After 3+ focuses on "WebSocket security":
  → Pattern: "Queen prioritizes WebSocket security"
  → Behavior: Executor always includes security in WebSocket work
  → Association: WebSocket ←→ security (strong link)
```

### Redirect Learning

```
After 1 redirect:
  → Logged in ERROR_LEDGER

After 2 redirects:
  → Pattern detected

After 3 redirects:
  → FLAGGED_ISSUE created
  → Constraint created: validate_approach_before_use
  → Blocks approach BEFORE execution
```

### Feedback Learning

```
Positive feedback ("Great work"):
  → Pattern recorded
  → Reused in similar contexts

Negative feedback ("Too many bugs"):
  → Verifier intensifies testing
  → Pattern: "Increase scrutiny when quality feedback"
```

## When Patterns Form

- **3+ focuses** on same topic → Preference learned
- **3+ redirects** on same pattern → Constraint created
- **5+ positive feedback** → Best practice established

## Research Foundation

Based on Phase 5 research:
- **Verification Feedback Loops**: Learning from feedback improves 39%
- **Explainable Verification**: Understanding why patterns work

Based on Phase 4 research:
- **Adaptive Personalization**: Systems learn user preferences
- **Anticipatory Context**: Predicting needs based on patterns

## Related Commands

```
/ant:status    # Colony status with memory stats
/ant:focus     # Add focus pheromone (teaches preferences)
/ant:redirect  # Add redirect pheromone (teaches constraints)
```
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
