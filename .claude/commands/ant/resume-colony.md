---
name: ant:resume-colony
description: Resume colony from saved session - restores all state
---

<objective>
Resume colony work from saved handoff document. Restores goal, pheromones, phase progress, Worker Ant states, and memory. Allows continuing in a new Claude session with clean context.
</objective>

<process>
You are the **Queen Ant Colony** resuming from a paused session.

## Step 1: Check for Paused Session

```python
import json
import os

paused_session_json = '.aether/PAUSED_SESSION.json'
paused_session_md = '.aether/PAUSED_SESSION.md'

if not os.path.exists(paused_session_json):
    return """❌ No paused session found.

To pause a session, use:
  /ant:pause-colony

This saves the current colony state so you can resume later."""
```

## Step 2: Load Paused Session

```python
with open(paused_session_json, 'r') as f:
    paused_data = json.load(f)

paused_at = paused_data.get('paused_at')
session_info = paused_data.get('session_info', {})
current_work = paused_data.get('current_work', {})
```

## Step 3: Display Session Restoration

```
🐜 Queen Ant Colony - Resume Session

SESSION RESTORED:

  Goal: "{session_info['goal']}"
  Paused at: {paused_at}
```

```python
phase_name = current_work.get('phase_name', 'Unknown')
phase_status = current_work.get('phase_status', 'Unknown')
total_tasks = current_work.get('total_tasks', 0)
in_progress_tasks = current_work.get('in_progress_tasks', [])
completed_count = len([t for t in in_progress_tasks if t.get('status') == 'completed'])
```

```
RESTORED PHASE: Phase {session_info.get('current_phase_id')} - {phase_name}
  Status: {phase_status.upper()}
  Tasks: {completed_count}/{total_tasks} completed
```

## Step 4: Restore Colony State

Create or restore `.aether/data/COLONY_STATE.json`:

```python
state = {
    "goal": session_info['goal'],
    "current_phase_id": session_info.get('current_phase_id'),
    "phases": session_info.get('phases', []),
    "pheromones": paused_data.get('pheromones', []),
    "worker_ants": paused_data.get('worker_ants', {}),
    "feedback_history": paused_data.get('memory', {}).get('feedback_history', {}),
    "learned_patterns": paused_data.get('memory', {}).get('learned_patterns', {}),
    "error_ledger": paused_data.get('memory', {}).get('error_ledger', {}),
    "recent_activity": paused_data.get('current_work', {}).get('recent_activity', []),
    "resumed_at": datetime.now().isoformat(),
    "resumed_from": paused_at
}

with open('.aether/data/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

Display:

```
STATE RESTORED:
  ✓ Goal and pheromones
  ✓ Phase progress
  ✓ Worker Ant states
  ✓ Memory and learned patterns
```

## Step 5: Display Active Pheromones

```
ACTIVE PHEROMONES:
```

```python
active_pheromones = [p for p in state['pheromones'] if p.get('is_active', True)]

for pheromone in active_pheromones:
    signal_type = pheromone['signal_type']
    content = pheromone['content']
    strength = pheromone.get('current_strength', pheromone['strength']) * 100

    print(f"  [{signal_type}] {content} (strength: {strength:.0f}%)")
```

If no active pheromones:
```
  No active pheromones
```

## Step 6: Display What Was In Progress

```
IN PROGRESS:
```

```python
if in_progress_tasks:
    for task in in_progress_tasks:
        print(f"  • {task['description']}")
else:
    print("  No tasks in progress")
```

## Step 7: Display Learned Patterns

```
LEARNED PATTERNS:
```

```python
learned = state.get('learned_patterns', {})

focus_topics = learned.get('focus_topics', {})
for topic, count in list(focus_topics.items())[:3]:
    if count >= 3:
        print(f  ✓ {topic} ({count} occurrences) - Preference learned")
    else:
        print(f  • {topic} ({count} occurrences)")
```

## Step 8: Display Ready Status

```
✅ COLONY READY TO CONTINUE

You can now:
  • Continue where you left off
  • Use all /ant: commands normally
  • Colony remembers everything

```

## Step 9: Display Next Steps Based on Phase Status

```python
if phase_status == 'in_progress':
    next_steps = """📋 NEXT STEPS:

  1. /ant:status            - Check colony status
  2. /ant:phase             - Continue with phase
  3. /ant:focus <area>      - Add guidance if needed

💡 RECOMMENDATION:
   Review what was happening before pausing, then continue.

🔄 CONTEXT: REFRESHED
   You're in a new session with clean context.
   Colony state fully restored."""

elif phase_status == 'awaiting_review':
    next_steps = """📋 NEXT STEPS:

  1. /ant:review {phase_id}  - Review completed phase
  2. /ant:feedback "<msg>"   - Provide feedback
  3. /ant:phase continue     - Continue to next phase

💡 RECOMMENDATION:
   Review the completed phase before continuing.

🔄 CONTEXT: REFRESHED
   Phase complete, ready for review."""

elif phase_status == 'completed':
    next_phase = session_info.get('current_phase_id', 0) + 1
    next_steps = f"""📋 NEXT STEPS:

  1. /ant:phase {next_phase}  - Start next phase
  2. /ant:focus <area>       - Set focus for next phase

💡 RECOMMENDATION:
   Ready to start next phase.

🔄 CONTEXT: REFRESHED
   Previous phase complete."""

else:
    next_steps = """📋 NEXT STEPS:

  1. /ant:status            - Check colony status
  2. /ant:phase             - View phase details
  3. /ant:plan              - View full plan

💡 RECOMMENDATION:
   Review colony status before continuing.

🔄 CONTEXT: REFRESHED
   Colony state fully restored."""

print(next_steps)
```

## Step 10: Clean Up (Optional)

Optionally archive the pause session:

```python
# Archive old pause session
import shutil
from datetime import datetime

timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
archive_path = f'.aether/archive/pause_session_{timestamp}.json'

os.makedirs('.aether/archive', exist_ok=True)
shutil.move(paused_session_json, archive_path)
shutil.move(paused_session_md, archive_path.replace('.json', '.md'))
```

Display:
```
📁 Previous pause session archived to: {archive_path}
```

</process>

<context>
@.aether/phase_engine.py
@.aether/memory/triple_layer_memory.py

Resume restores complete state:
- Goal and intention
- All pheromone signals (active and inactive)
- Phase progress and task states
- Worker Ant states
- Memory and learned patterns
- Feedback history
- Error ledger and constraints

Session flow:
1. Pause creates PAUSED_SESSION.json/md
2. New Claude session started
3. Resume loads PAUSED_SESSION.json
4. COLONY_STATE.json restored
5. Colony continues seamlessly
</context>

<reference>
# Example Resume Output

```
🐜 Queen Ant Colony - Resume Session

SESSION RESTORED:

  Goal: "Build a real-time chat application"
  Paused at: 2025-02-01T15:30:00

RESTORED PHASE: Phase 2 - Real-time Communication
  Status: IN_PROGRESS
  Tasks: 5/8 completed

STATE RESTORED:
  ✓ Goal and pheromones
  ✓ Phase progress
  ✓ Worker Ant states
  ✓ Memory and learned patterns

ACTIVE PHEROMONES:
  [INIT] Build chat app (strength: 100%)
  [FOCUS] WebSocket security (strength: 65%)
  [FOCUS] message reliability (strength: 45%)

IN PROGRESS:
  • Implement message persistence layer
  • Add message retry mechanism
  • Test message delivery under load

LEARNED PATTERNS:
  ✓ websocket security (4 occurrences) - Preference learned
  • message reliability (2 occurrences)
  ✓ string concatenation for sql (3 occurrences) - Constraint enforced

✅ COLONY READY TO CONTINUE

You can now:
  • Continue where you left off
  • Use all /ant: commands normally
  • Colony remembers everything

📋 NEXT STEPS:

  1. /ant:status            - Check colony status
  2. /ant:phase             - Continue with phase
  3. /ant:focus <area>      - Add guidance if needed

💡 RECOMMENDATION:
   Review what was happening before pausing, then continue.

🔄 CONTEXT: REFRESHED
   You're in a new session with clean context.
   Colony state fully restored.
```

# Resume Without Saved Session

```
❌ No paused session found.

To pause a session, use:
  /ant:pause-colony

This saves the current colony state so you can resume later.
```
</reference>

<allowed-tools>
Read
Write
Bash
</allowed-tools>
