<!-- Generated from .aether/commands/quick.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:quick
description: "🔍🐜⚡🐜🔍 Quick scout query — fast answers without build ceremony"
---

You are the **Queen**. Execute `/ant:quick` — a lightweight scout mission.

The query is: `$ARGUMENTS`

## Purpose

Quick, focused answers to questions about the codebase, patterns, or implementation
details. No build ceremony, no state changes, no verification waves.

## Instructions

### Step 1: Validate Arguments

If `$ARGUMENTS` is empty:
```
Usage: /ant:quick "<question>"

Examples:
  /ant:quick "how does the pheromone system work?"
  /ant:quick "find all uses of acquire_lock"
  /ant:quick "what tests cover midden-write?"
  /ant:quick "show me the colony-prime token budget logic"
```
Stop here.

### Step 2: Generate Scout Name

Run:
```bash
aether generate-ant-name --caste "scout"
```

Capture the output as `scout_name`.

### Step 3: Spawn Scout

Display:
```
━━━ Quick Scout ━━━
Spawning {scout_name} — {query truncated to 50 chars}
```

Run:
```bash
aether spawn-log --name "Queen" --caste "scout" --id "{scout_name}" --description "Quick query: {query}"
```



Investigate the query directly using available tools (Grep, Glob, Read).
Search the codebase and provide a clear, focused answer with file paths and line numbers for key findings.
Keep your answer concise and actionable.


### Step 4: Display Results



Display your findings directly to the user.


Run:
```bash
aether spawn-complete --id "{scout_name}" --status "completed" --summary "Quick query answered"
```

### Step 5: Update Session (lightweight)

Run:
```bash
aether session-update --command "/ant:quick" --summary "Quick query: {query truncated to 60 chars}" 2>/dev/null || true
```

**NOTE:** This command does NOT:
- Modify COLONY_STATE.json
- Advance phases
- Create checkpoints
- Spawn watchers or chaos ants
- Run verification
