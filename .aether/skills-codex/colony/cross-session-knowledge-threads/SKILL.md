---
name: cross-session-knowledge-threads
description: Use when cross-cutting investigations or decisions need a persistent thread across sessions and phases
type: colony
domains: [knowledge-management, cross-session, persistent-context]
agent_roles: [keeper, chronicler]
workflow_triggers: [resume, plan]
task_keywords: [thread, cross-session, persistent, keep track, remember]
priority: normal
version: "1.0"
---

# Cross Session Knowledge Threads

## Purpose

Persistent cross-session knowledge threads for work that spans sessions but isn't phase-specific. Provides a structured way to accumulate context, decisions, and partial work on topics that cut across the project.

## When to Use

- A topic comes up repeatedly across sessions but doesn't map to a single phase
- User says "keep track of...", "note this for later...", or "remember that..."
- Cross-cutting concerns like performance, accessibility, or security need ongoing tracking
- Research accumulates across sessions and needs a central collection point
- You want to preserve investigation results that don't yet have a home in a phase

## Instructions

### Open a Thread

1. Create a thread with a descriptive title and initial entry:
   ```json
   {
     "id": "KT-{NNN}",
     "title": "{thread title}",
     "opened": "ISO8601",
     "status": "open",
     "tags": ["{tag1}", "{tag2}"],
     "entries": [
       {
         "timestamp": "ISO8601",
         "author": "{session_id or agent}",
         "type": "note|decision|finding|question",
         "content": "{entry text}"
       }
     ]
   }
   ```

2. Write to `.aether/data/threads/KT-{NNN}.json`
3. Update `.aether/data/threads/index.json` with a summary entry for fast listing

### Append to Thread

1. Locate thread by ID (`KT-NNN`) or title match
2. Add new entry to the entries array
3. Entries can be:
   - `note`: General observation or information
   - `decision`: A choice made with rationale
   - `finding`: Research result, benchmark, or discovered fact
   - `question`: Open question awaiting resolution
4. Each entry is timestamped and attributed to the session that created it

### List Threads

1. Read `.aether/data/threads/index.json`
2. Display threads grouped by status:
   - **Open**: Active threads with entry count and last activity date
   - **Closed**: Completed threads with resolution summary
3. Filter by `--tag <tag>` or `--status open|closed`

### Close a Thread

1. Add a final entry with `type: "resolution"` summarizing the thread outcome
2. Set status to `closed` with `closed` timestamp
3. If the thread's knowledge should be preserved long-term:
   - Extract key findings into colony knowledge base
   - Optionally create a LEARNINGS.md entry
   - Tag relevant seeds if the thread uncovered future work

### Resume a Thread

1. Present thread summary: title, entry count, date range, key decisions/findings
2. Show last 3 entries for context
3. Accept new entries or questions to append

## Key Patterns

- **Append-only entries**: Never modify or delete entries; only add new ones. Corrections are new entries that reference the earlier one.
- **Thread granularity**: One thread per topic, not per session. If "database performance" comes up in 5 sessions, it's still one thread.
- **Cross-referencing**: Entries can reference phases (`phase:3`), todos (`TD-042`), seeds (`SEED-007`), and other threads (`KT-001`)
- **Auto-close on phase completion**: When a phase completes, scan open threads for any that are fully resolved by the phase work and suggest closure

## Output Format

```
KT-003: "Database query performance optimization" [open]
  Opened: 2026-04-15 | Entries: 7 | Last activity: 2026-04-22
  Tags: performance, database, scalability
  Key decisions: Use connection pooling (entry 3), Add query timeout (entry 5)
  Open questions: 1 (sharding strategy for multi-tenant)

  Last entry (2026-04-22):
  [finding] Benchmark shows 3x improvement with prepared statements
```

## Examples

```
# Open a new thread
> knowledge-threads open "API rate limiting strategy" --tag api --tag security

# Append a finding
> knowledge-threads append KT-003 --type finding "Redis-based rate limiter handles 10k rps"

# List open threads
> knowledge-threads list --status open

# Close a resolved thread
> knowledge-threads close KT-003 --resolution "Implemented token bucket with Redis backend"
```
