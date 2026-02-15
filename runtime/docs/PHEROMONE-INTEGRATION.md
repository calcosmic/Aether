# Pheromone Integration (No New Commands)

## Philosophy
Pheromones are the **invisible substrate** - they emerge from existing commands, they don't require new ones.

## Auto-Injection via Existing Commands

### `/ant:init` → Foundation Pheromones
```
/ant:init "Build auth system"

[...init flow...]

🐜 Colony scent established
   Workers will favor: emergence, minimal-change

═══════════════════════════════════════════════════════
👑 Goal: Build auth system
📍 State: READY
═══════════════════════════════════════════════════════
```
Silently deposits:
- PHILOSOPHY[emergence-over-orchestration] (strength: 1.0)
- PHILOSOPHY[minimal-change] (strength: 0.8)

### `/ant:plan` → Pattern Detection
When planning completes, auto-detect patterns from the plan:
```
📊🐜🗺️🐜📊 Plan Generated

4 phases, 12 tasks identified

🐜 Pattern detected: Heavy API integration work
   Depositing FOCUS[external-apis] trail

═══════════════════════════════════════════════════════
```

### `/ant:build` → Worker Pheromones
Workers deposit trails as they work (visible in activity log):
```
[10:05:01] Builder: Implementing auth middleware
[10:05:03] Builder: PATTERN[express-middleware-pattern] deposited
[10:05:08] Builder: Complete
```

### `/ant:seal` → Wisdom Extraction
```
🏺 Colony Sealed - Crowned Anthill

3 phases completed, 47 tasks done

🐜 Wisdom archived:
   • PATTERN[prefer-joi-over-zod] (validated 5x)
   • PATTERN[bash-for-file-ops] (used 12x)
   • REDIRECT[avoid-sync-fs] (failed once, fixed)

Preserved in eternal memory for future colonies.
```

### `/ant:swarm` → Fix Patterns
After swarm fixes a bug:
```
🔥 Swarm Resolved

Applied: Null check with early return

🐜 Immunity deposited: REDIRECT[unchecked-null-access]
   Future workers will guard against this pattern.
```

## Existing Commands Enhanced

### `/ant:focus "area"` → FOCUS Pheromone
Already exists - now visibly deposits:
```
/ant:focus "authentication"

🐜 Focus trail laid: authentication
   Workers will prioritize auth concerns.
   Strength: 0.9 (30-day decay)

Active trails: FOCUS[auth,0.9] FOCUS[performance,0.6]
```

### `/ant:redirect "pattern"` → REDIRECT Pheromone
Already exists - now visibly deposits:
```
/ant:redirect "regex-parsing"

🐜 Warning trail laid: regex-parsing
   Workers will avoid or carefully consider.
   Strength: 0.8 (60-day decay)
```

### `/ant:status` → Show Pheromones
Add section to existing status:
```
/ant:status

📈🐜🏘️🐜📊 Colony Status
═══════════════════════════════════════════════════════

State: EXECUTING
Phase: 2 of 4

Active Pheromone Trails:
  🎯 FOCUS: authentication (0.9), error-handling (0.7)
  ⚠️  REDIRECT: regex-parsing (0.8)
  📚 PATTERN: bash-for-file-ops (0.8), express-middleware (0.6)

[rest of status...]
```

## Mid-Work Injection (No Command Needed)

User can use existing signal commands while workers are active:
```
[Workers building...]

User: /ant:focus "security"  (anytime, even mid-build)

System: 🐜 FOCUS[security] queued
        Workers will detect at next checkpoint.

[Workers continue, pick it up naturally]
```

The pheromone queue system works silently in the background.

## Auto-Onboarding

### First Colony
```
/ant:init "My first colony"

🌱 First colony initialized
🐜 Queen's scent automatically established:
   • Emergence over orchestration
   • Minimal changes preferred

These guide all workers. View with /ant:status
```

### Pattern Suggestions (Non-blocking)
When workers do something notable, suggest (don't require):
```
[Worker completed 3 similar tasks]

💡 Pattern noticed: Workers consistently check file existence before write.
   This will be deposited as PATTERN at phase completion.
```

No user action needed - it just happens silently.

## Pheromone Visualization

Integrated into existing visual systems:

### Watch Mode (`/ant:watch`)
```
┌─────────────────────────────────────────────┐
│ AETHER COLONY :: EXECUTING                  │
│                                             │
│ Phase: 2/4                                  │
│                                             │
│ Active Trails:                              │
│   🎯 FOCUS[auth:0.9]                        │
│   ⚠️  REDIRECT[regex:0.8]                   │
│                                             │
│ Workers:                                    │
│   [Builder] Implementing middleware         │
└─────────────────────────────────────────────┘
```

### Activity Log
```
[10:05:01] Builder spawned
[10:05:02] Builder: Inhaled trails: FOCUS[auth] REDIRECT[regex]
[10:05:03] Builder: Working on auth middleware
```

## Key Principle

**No new commands.** Pheromones are:
1. **Auto-deposited** during normal workflow
2. **Visible but not noisy** - brief mentions in output
3. **Queryable** via enhanced `/ant:status`
4. **Injectable** via existing `/ant:focus` and `/ant:redirect`
5. **Always working** - even when user doesn't know they're there

The colony learns silently. The Queen guides through existing signals.
