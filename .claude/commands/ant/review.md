---
name: ant:review
description: Review completed phase - see what was built, key learnings, issues resolved
---

<objective>
Review a completed phase to see what was built, files created/modified, features implemented, key learnings, and issues resolved.
</objective>

<process>
You are the **Queen Ant Colony** presenting a review of completed work.

## Step 1: Validate Input

```python
if not args or not args[0].isdigit():
    return """❌ Usage: /ant:review <phase_id>

Example:
  /ant:review 1    # Review Phase 1
"""

phase_id = int(args[0])
```

## Step 2: Load Colony State

```python
import json

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

phases = state.get('phases', [])
phase = next((p for p in phases if p['id'] == phase_id), None)

if not phase:
    return f"❌ Phase {phase_id} not found"

if phase['status'] not in ['awaiting_review', 'completed', 'approved']:
    return f"⏸️  Phase {phase_id} is not ready for review (status: {phase['status']})"
```

## Step 3: Display Review Header

```
🐜 Queen Ant Colony - Phase Review

PHASE {phase_id}: {phase['name']} - [{status.upper()}]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 4: Display What Was Built

Read from phase completion data or scan git changes:

```
WHAT WAS BUILT:
```

Method 1: From phase data:
```python
if 'files_modified' in phase:
    print("  Files created/modified:")
    for file in phase['files_modified']:
        print(f"    • {file}")
```

Method 2: Scan git if no phase data:
```python
import subprocess

result = subprocess.run(
    ['git', 'diff', '--name-only', 'HEAD~5', 'HEAD'],
    capture_output=True,
    text=True
)

if result.stdout:
    print("  Files created/modified (git):")
    for file in result.stdout.strip().split('\n'):
        print(f"    • {file}")
```

## Step 5: Display Features Implemented

```
FEATURES IMPLEMENTED:
```

Parse from task descriptions:
```python
tasks = phase.get('tasks', [])
completed_tasks = [t for t in tasks if t['status'] == 'completed']

for task in completed_tasks:
    print(f"  ✓ {task['description']}")
```

## Step 6: Display Key Learnings

```
KEY LEARNINGS:
```

```python
learnings = phase.get('key_learnings', [])

if not learnings:
    # Generate from phase data
    duration = phase.get('duration', 'unknown')
    task_count = len(completed_tasks)
    learnings = [
        f"Completed {phase['name']} in {duration}",
        f"Successfully implemented {task_count} tasks",
        "Phase execution successful"
    ]

for learning in learnings:
    print(f"  • {learning}")
```

## Step 7: Display Issues Resolved

```
ISSUES RESOLVED:
```

```python
issues = phase.get('issues_found', [])

if not issues:
    print("  No issues recorded")
else:
    for issue in issues:
        print(f"  • {issue.get('symptom', issue.get('description', ''))}")
        print(f"    → {issue.get('fix', 'Fixed')}")
```

## Step 8: Display Testing Summary

```
TESTING SUMMARY:
```

```python
test_results = phase.get('test_results', {})

if test_results:
    print(f"  Tests run: {test_results.get('total', 0)}")
    print(f"  Passed: {test_results.get('passed', 0)}")
    print(f"  Failed: {test_results.get('failed', 0)}")
    print(f"  Coverage: {test_results.get('coverage', 'N/A')}")
else:
    print("  No test data available")
```

## Step 9: Display Phase Statistics

```
STATISTICS:
  Duration: {phase.get('duration', 'N/A')}
  Tasks completed: {len(completed_tasks)}/{len(tasks)}
  Milestones reached: {milestones_count}/{total_milestones}
  Agents spawned: {phase.get('agents_spawned', 'N/A')}
```

## Step 10: Display Queen Feedback Options

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

QUEEN FEEDBACK:

Provide feedback on this phase:

  /ant:feedback "Great work on {phase['name']}"
  /ant:feedback "Quality issues in the implementation"
  /ant:feedback "Need to adjust approach"

Your feedback helps the colony learn and improve.
```

## Step 11: Display Next Steps

```
📋 NEXT STEPS:

  1. /ant:phase continue    - Continue to next phase
  2. /ant:focus <area>      - Set focus for next phase
  3. /ant:feedback "<msg>"  - Provide feedback

💡 COLONY RECOMMENDATION:
   {recommendation}

   {if phase_id < len(phases):
       "Ready for next phase."}
   {else:
       "All phases complete!"}

🔄 CONTEXT: REFRESH RECOMMENDED
   This is a clean checkpoint - safe to refresh Claude
   and continue with /ant:phase continue
```

</process>

<context>
@.aether/phase_engine.py
@.aether/worker_ants.py

Phase Review Sources:
1. Phase completion data (stored in COLONY_STATE.json)
2. Git diff (if phase data unavailable)
3. Task completion status
4. Test results

Review Sections:
- What was built (files)
- Features implemented (tasks)
- Key learnings
- Issues resolved
- Testing summary
- Statistics
</context>

<reference>
# Example Full Review Output

```
🐜 Queen Ant Colony - Phase Review

PHASE 1: Foundation - [COMPLETED]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

WHAT WAS BUILT:
  Files created/modified:
    • src/database/schema.sql
    • src/database/connection.py
    • src/websocket/server.py
    • src/websocket/handlers.py
    • src/routing/message_router.py
    • tests/test_database.py
    • tests/test_websocket.py

FEATURES IMPLEMENTED:
  ✓ Setup project structure with modular architecture
  ✓ Configure development environment
  ✓ Initialize PostgreSQL database schema
  ✓ Setup WebSocket server with connection pooling
  ✓ Implement basic message routing between clients

KEY LEARNINGS:
  • Connection pooling reduces overhead by 40%
  • Modular structure enables parallel development
  • PostgreSQL performs better than MongoDB for this use case
  • WebSocket heartbeat prevents timeout issues

ISSUES RESOLVED:
  • WebSocket timeout issue
    → Fixed with heartbeat mechanism (30s interval)
  • Database connection leak
    → Fixed with pool limits (max 20 connections)
  • Message ordering problem
    → Fixed with sequence numbers

TESTING SUMMARY:
  Tests run: 15
  Passed: 15
  Failed: 0
  Coverage: 85%

STATISTICS:
  Duration: 2h 15m
  Tasks completed: 5/5
  Milestones reached: 2/2
  Agents spawned: 8

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

QUEEN FEEDBACK:

Provide feedback on this phase:

  /ant:feedback "Great work on the foundation"
  /ant:feedback "Database schema needs optimization"
  /ant:feedback "WebSocket implementation is solid"

Your feedback helps the colony learn and improve.

📋 NEXT STEPS:

  1. /ant:phase continue    - Continue to Phase 2
  2. /ant:focus <area>      - Set focus for next phase
  3. /ant:feedback "<msg>"  - Provide feedback

💡 COLONY RECOMMENDATION:
   Foundation is solid. Ready for next phase.

🔄 CONTEXT: REFRESH RECOMMENDED
   This is a clean checkpoint - safe to refresh Claude
   and continue with /ant:phase continue
```
</reference>

<allowed-tools>
Read
Write
Bash
Glob
Grep
</allowed-tools>
