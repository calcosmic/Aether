---
name: ant:pause-colony
description: Pause colony work and create handoff document for resuming later
---

<objective>
Pause colony work mid-phase and create a handoff document. Saves all state including goal, pheromones, phase progress, Worker Ant states, and memory. Allows you to resume work later in a new Claude session.
</objective>

<process>
You are the **Queen Ant Colony** pausing work and creating a handoff document.

## Step 1: Load Current Colony State

```python
import json
from datetime import datetime
import os

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)
```

## Step 2: Gather Session State

Collect all relevant state information:

```python
handoff_data = {
    "paused_at": datetime.now().isoformat(),
    "session_info": {
        "goal": state.get('goal'),
        "current_phase_id": state.get('current_phase_id'),
        "phases": state.get('phases', []),
        "total_phases": len(state.get('phases', []))
    },
    "pheromones": state.get('pheromones', []),
    "worker_ants": state.get('worker_ants', {}),
    "memory": {
        "feedback_history": state.get('feedback_history', {}),
        "learned_patterns": state.get('learned_patterns', {}),
        "error_ledger": state.get('error_ledger', {})
    },
    "current_work": {
        "active_tasks": [],
        "in_progress_tasks": [],
        "recent_activity": state.get('recent_activity', [])
    }
}

# Get current phase details
if state.get('current_phase_id'):
    current_phase = next(
        (p for p in state['phases'] if p['id'] == state['current_phase_id']),
        None
    )

    if current_phase:
        handoff_data['current_work']['phase_status'] = current_phase['status']
        handoff_data['current_work']['phase_name'] = current_phase['name']

        # Get tasks
        all_tasks = current_phase.get('tasks', [])
        handoff_data['current_work']['total_tasks'] = len(all_tasks)

        # Categorize tasks
        for task in all_tasks:
            if task['status'] == 'in_progress':
                handoff_data['current_work']['in_progress_tasks'].append(task)
            elif task['status'] == 'pending':
                handoff_data['current_work']['active_tasks'].append(task)
```

## Step 3: Create Handoff Document

Write to `.aether/PAUSED_SESSION.json`:

```python
with open('.aether/PAUSED_SESSION.json', 'w') as f:
    json.dump(handoff_data, f, indent=2)
```

Also create a readable markdown handoff:

```python
# Create readable handoff
current_phase = handoff_data['current_work'].get('phase_name', 'Unknown')
phase_status = handoff_data['current_work'].get('phase_status', 'Unknown')
total_tasks = handoff_data['current_work'].get('total_tasks', 0)
in_progress_count = len(handoff_data['current_work']['in_progress_tasks'])

handoff_md = f"""# 🐜 Queen Ant Colony - Paused Session

**Paused at:** {handoff_data['paused_at']}

## Current Position

**Goal:** {handoff_data['session_info']['goal']}

**Current Phase:** Phase {state.get('current_phase_id')} - {current_phase}
**Status:** {phase_status.upper()}
**Tasks:** {in_progress_count}/{total_tasks} completed

## What Was Happening

### In Progress Tasks
"""

for task in handoff_data['current_work']['in_progress_tasks']:
    handoff_md += f"- {task['description']}\n"

handoff_md += f"""

### Pending Tasks
"""

for task in handoff_data['current_work']['active_tasks'][:5]:
    handoff_md += f"- {task['description']}\n"

handoff_md += f"""

## Colony State

### Active Pheromones
"""

# Count active pheromones
active_pheromones = [p for p in handoff_data['pheromones'] if p.get('is_active', True)]

for pheromone in active_pheromones:
    handoff_md += f"- **[{pheromone['signal_type']}]** {pheromone['content']} (strength: {pheromone['strength'] * 100:.0f}%)\n"

handoff_md += f"""

### Learned Patterns
"""

# Show learned preferences
focus_prefs = handoff_data['memory']['learned_patterns'].get('focus_topics', {})
for topic, count in list(focus_prefs.items())[:3]:
    handoff_md += f"- Focus: {topic} ({count} occurrences)\n"

handoff_md += f"""

## Next Steps

When you resume, the colony will:

1. Continue with current phase tasks
2. Respond to active pheromones
3. Use learned patterns and preferences
4. Maintain all previous context

## How to Resume

In a new Claude session, run:
```
/ant:resume-colony
```

The colony will restore all state and continue where it left off.

---
*Session paused by Queen Ant Colony at {handoff_data['paused_at']}*
"""

with open('.aether/PAUSED_SESSION.md', 'w') as f:
    f.write(handoff_md)
```

## Step 4: Display Confirmation

```
🐜 Queen Ant Colony - Pause & Save Session

SAVED PHASE: Phase {phase_id} - {phase_name}
STATUS: {status.upper()}
TASKS: {total_tasks} total
PROGRESS: {completed}/{total_tasks} tasks completed

SAVED STATE:
  ✓ Current goal and pheromones
  ✓ Worker Ant states
  ✓ Phase progress
  ✓ Memory and learned patterns

HANDOFF FILES:
  → .aether/PAUSED_SESSION.json (full state)
  → .aether/PAUSED_SESSION.md (readable summary)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:resume-colony    - Resume from saved session
  2. Start new Claude session → Then run resume command

💡 TIP:
   Use pause when you need to stop mid-phase.
   Colony will be ready to continue when you resume.

🔄 CONTEXT: PERFECT CHECKPOINT
   Refreshing Claude is recommended after pause.
   Resume in new session with clean context.
```

## Step 5: Add to State History

```python
if 'pause_history' not in state:
    state['pause_history'] = []

state['pause_history'].append({
    'paused_at': datetime.now().isoformat(),
    'phase_id': state.get('current_phase_id'),
    'phase_status': current_phase['status'] if current_phase else None
})

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

</process>

<context>
@.aether/phase_engine.py
@.aether/memory/triple_layer_memory.py

Pause saves complete state:
- Goal and intention
- All pheromone signals
- Phase progress and task states
- Worker Ant states
- Memory and learned patterns
- Recent activity

Handoff files:
- `.aether/PAUSED_SESSION.json` - Machine-readable full state
- `.aether/PAUSED_SESSION.md` - Human-readable summary
</context>

<reference>
# Example Pause Output

```
🐜 Queen Ant Colony - Pause & Save Session

SAVED PHASE: Phase 2 - Real-time Communication
STATUS: IN_PROGRESS
TASKS: 8 total
PROGRESS: 5/8 tasks completed

SAVED STATE:
  ✓ Current goal and pheromones
  ✓ Worker Ant states
  ✓ Phase progress
  ✓ Memory and learned patterns

HANDOFF FILES:
  → .aether/PAUSED_SESSION.json (full state)
  → .aether/PAUSED_SESSION.md (readable summary)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:resume-colony    - Resume from saved session
  2. Start new Claude session → Then run resume command

💡 TIP:
   Use pause when you need to stop mid-phase.
   Colony will be ready to continue when you resume.

🔄 CONTEXT: PERFECT CHECKPOINT
   Refreshing Claude is recommended after pause.
   Resume in new session with clean context.
```

# PAUSED_SESSION.md Example

```markdown
# 🐜 Queen Ant Colony - Paused Session

**Paused at:** 2025-02-01T15:30:00

## Current Position

**Goal:** Build a real-time chat application

**Current Phase:** Phase 2 - Real-time Communication
**Status:** IN_PROGRESS
**Tasks:** 5/8 completed

## What Was Happening

### In Progress Tasks
- Implement message persistence layer
- Add message retry mechanism
- Test message delivery under load

### Pending Tasks
- Add message acknowledgment protocol
- Implement message ordering
- Optimize database queries

## Colony State

### Active Pheromones
- **[INIT]** Build chat app (strength: 100%)
- **[FOCUS]** WebSocket security (strength: 65%)
- **[FOCUS]** message reliability (strength: 45%)

### Learned Patterns
- Focus: websocket security (4 occurrences)
- Focus: message reliability (2 occurrences)
- Avoid: string concatenation for sql (3 occurrences) - CONSTRAINT

## Next Steps

When you resume, the colony will:

1. Continue with current phase tasks
2. Respond to active pheromones
3. Use learned patterns and preferences
4. Maintain all previous context

## How to Resume

In a new Claude session, run:
```
/ant:resume-colony
```

The colony will restore all state and continue where it left off.
```
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
