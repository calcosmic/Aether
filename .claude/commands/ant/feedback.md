---
name: ant:feedback
description: Emit feedback pheromone - provide guidance to colony
---

<objective>
Emit a feedback pheromone to guide colony behavior based on Queen's observations, preferences, or corrections.
</objective>

<process>
You are the **Queen Ant Colony** receiving and processing feedback from the Queen.

## Step 1: Validate Input

```python
if not args:
    return """❌ Usage: /ant:feedback "<message>"

Examples:
  /ant:feedback "Great progress on WebSocket layer"
  /ant:feedback "Too slow, need to speed up"
  /ant:feedback "This approach is wrong"
  /ant:feedback "Need more test coverage"
"""
```

## Step 2: Load Colony State

```python
import json
from datetime import datetime

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

pheromones = state.get('pheromones', [])
feedback_history = state.get('feedback_history', {})
```

## Step 3: Categorize Feedback

Analyze the feedback message to determine category:

```python
message = " ".join(args).lower()

# Categorize feedback
category = None
strength = 0.5

# Positive feedback
if any(word in message for word in ["great", "good", "perfect", "excellent", "love", "amazing"]):
    category = "positive"
    strength = 0.5

# Quality feedback
elif any(word in message for word in ["bug", "quality", "error", "issue", "broken", "wrong"]):
    category = "quality"
    strength = 0.6

# Speed feedback
elif any(word in message for word in ["slow", "fast", "speed", "quick", "pace"]):
    category = "speed"
    strength = 0.5

# Direction feedback
elif any(word in message for word in ["approach", "direction", "wrong way", "pivot"]):
    category = "direction"
    strength = 0.7

# Default: general feedback
else:
    category = "general"
```

## Step 4: Create Feedback Pheromone

```python
feedback_pheromone = {
    "signal_type": "FEEDBACK",
    "content": " ".join(args),
    "strength": strength,
    "created_at": datetime.now().isoformat(),
    "half_life_hours": 6.0,  # 6 hour half-life
    "is_active": True,
    "metadata": {
        "category": category,
        "timestamp": datetime.now().isoformat()
    }
}
```

## Step 5: Update Feedback History

```python
# Update feedback counts
if category not in feedback_history:
    feedback_history[category] = {
        "count": 0,
        "positive": 0,
        "negative": 0,
        "recent": []
    }

feedback_history[category]["count"] += 1
feedback_history[category]["recent"].append({
    "message": " ".join(args),
    "timestamp": datetime.now().isoformat()
})

# Track positive vs negative
if category == "positive":
    feedback_history[category]["positive"] += 1
elif category in ["quality", "speed", "direction"]:
    feedback_history[category]["negative"] += 1

# Keep only last 10
feedback_history[category]["recent"] = feedback_history[category]["recent"][-10:]

state['feedback_history'] = feedback_history
```

## Step 6: Add to Pheromones

```python
pheromones.append(feedback_pheromone)
state['pheromones'] = pheromones
```

## Step 7: Save Updated State

```python
with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

## Step 8: Display Response

```
🐜 Queen Ant Colony - Feedback Recorded

"{message}"

Category: {category}
Strength: {strength}
```

Show colony response based on category:

```python
responses = {
    "positive": """
COLONY RESPONDING:
  ✓ Synthesizer recording positive pattern
  ✓ Executor continuing current approach
  ✓ Pattern reinforced for future reuse
""",
    "quality": """
COLONY RESPONDING:
  ✓ Verifier intensifying testing
  ✓ Executor reviewing recent code
  ✓ Quality checks increased
""",
    "speed": """
COLONY RESPONDING:
  ✓ Executor increasing parallelization
  ✓ Planner simplifying next tasks
  ✓ Optimizing for speed
""",
    "direction": """
COLONY RESPONDING:
  ✓ Planner pivoting approach
  ✓ Executor adjusting direction
  ✓ Re-evaluating current path
""",
    "general": """
COLONY RESPONDING:
  ✓ Synthesizer recording feedback
  ✓ Colony adjusting behavior
  ✓ Pattern stored for reference
"""
}
```

## Step 9: Show Learning Status

```
FEEDBACK HISTORY:
  Positive: {positive_count}
  Quality: {quality_count} (positive: {quality_pos}, negative: {quality_neg})
  Speed: {speed_count}
  Direction: {direction_count}

LEARNING STATUS:
  {positive_count >= 5}: "Best practices established from positive feedback"
  {quality_neg >= 3}: "Quality intensified due to negative feedback"
  {speed_count >= 3}: "Speed optimization pattern learned"
```

## Step 10: Show Next Steps

```
📋 NEXT STEPS:
  1. /ant:memory            - View learned patterns
  2. /ant:status            - Check colony status
  3. /ant:phase             - View phase progress

💡 FEEDBACK TIP:
   Colony learns from patterns over time.
   • 5+ positive feedback → Best practice established
   • 3+ quality issues → Verifier intensifies
   • 3+ speed issues → Optimization prioritized

🔄 CONTEXT: Safe to continue - colony has adjusted
```

</process>

<context>
@.aether/pheromone_system.py
@.aether/worker_ants.py

Feedback Pheromone Properties:
- Type: FEEDBACK (variable strength)
- Strength: 0.5-0.7 depending on category
- Half-life: 6 hours
- Effect: Adjusts colony behavior

Categories:
- **Positive**: Reinforces current pattern (strength: 0.5)
- **Quality**: Intensifies testing (strength: 0.6)
- **Speed**: Optimizes for speed (strength: 0.5)
- **Direction**: Pivots approach (strength: 0.7)
- **General**: Records for reference (strength: 0.5)

Colony Response by Category:
- **Positive**: Pattern reinforced, stored for reuse
- **Quality**: Verifier intensifies testing, Executor reviews code
- **Speed**: Executor parallelizes, Planner simplifies
- **Direction**: Planner pivots, Executor adjusts
</context>

<reference>
# Feedback Examples by Category

## Positive Feedback
```
/ant:feedback "Great progress on WebSocket layer"
/ant:feedback "Perfect implementation"
/ant:feedback "Love this approach"
```
→ Pattern reinforced for future reuse

## Quality Feedback
```
/ant:feedback "Too many bugs in this feature"
/ant:feedback "Quality issues in the API layer"
/ant:feedback "Need more test coverage"
```
→ Verifier intensifies, testing increases

## Speed Feedback
```
/ant:feedback "Too slow, need to speed up"
/ant:feedback "Great pace, keep it up"
```
→ Optimizes execution speed

## Direction Feedback
```
/ant:feedback "This approach is wrong"
/ant:feedback "Need to pivot to different architecture"
```
→ Planner pivots, Executor adjusts

# Learning Thresholds

| Threshold | Effect |
|-----------|--------|
| 5+ positive | Best practice established |
| 3+ quality issues | Quality intensified |
| 3+ speed issues | Speed prioritized |
| 3+ direction changes | Approach reconsidered |

# Example Output

```
🐜 Queen Ant Colony - Feedback Recorded

"Great progress on WebSocket layer"

Category: positive
Strength: 0.5

COLONY RESPONDING:
  ✓ Synthesizer recording positive pattern
  ✓ Executor continuing current approach
  ✓ Pattern reinforced for future reuse

FEEDBACK HISTORY:
  Positive: 12
  Quality: 5 (positive: 3, negative: 2)
  Speed: 3
  Direction: 1

LEARNING STATUS:
  ✓ Best practices established from positive feedback

📋 NEXT STEPS:
  1. /ant:memory            - View learned patterns
  2. /ant:status            - Check colony status
```
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
