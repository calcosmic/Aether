# Queen Ant Colony Architecture

**True Emergence: Workers Spawn Workers**

---

## Core Principles

1. **Queen provides intention via constraints**
2. **Workers spawn workers directly (no Queen mediation)**
3. **Structure emerges from work, not orchestration**
4. **Depth-based behavior controls spawn cascades**
5. **Visual observability via tmux**

---

## The Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        QUEEN (User)                         │
│  Provides goal, constraints (focus/avoid), observation      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Signals (not commands)
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   /ant:init  │  │ /ant:focus   │  │/ant:redirect │
│   Set goal   │  │  Add focus   │  │  Add avoid   │
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                         ▼
        ┌─────────────────────────────────────────────┐
        │            CONSTRAINTS LAYER                 │
        │  Focus areas + Avoid patterns               │
        │  Simple, declarative, no decay              │
        └────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    /ant:plan                                 │
│  Iterative Research/Planning Loop (up to 50 iterations)    │
│  Scout + Route-Setter until 95% confidence                 │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    /ant:build                                │
│  Spawns ONE Prime Worker                                    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    PRIME WORKER                              │
│  Depth 1: Coordinator (can spawn up to 4 specialists)      │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ BUILDER      │  │   WATCHER    │  │    SCOUT     │
│ Depth 2      │  │   Depth 2    │  │   Depth 2    │
│ (can spawn 2)│  │  (can spawn 2)│ │  (can spawn 2)│
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Sub-builder  │  │ Sub-watcher  │  │  Sub-scout   │
│   Depth 3    │  │   Depth 3    │  │   Depth 3    │
│  (no spawn)  │  │  (no spawn)  │  │  (no spawn)  │
└──────────────┘  └──────────────┘  └──────────────┘
```

---

## Key Change: Workers Spawn Workers

**Before (v2.0):**
```
Queen → spawns Phase Lead → Phase Lead outputs SPAWN REQUEST text
     → Queen parses text → Queen spawns workers
     → Workers output SPAWN REQUEST → Queen parses → Queen spawns

Result: "Emergence" was actually orchestration in disguise
```

**After (v3.0):**
```
Queen → spawns Prime Worker with Task tool
     → Prime Worker uses Task tool to spawn specialists
     → Specialists use Task tool to spawn sub-specialists (if surprised)
     → Depth 3 workers complete work inline

Result: True emergence where structure comes from the work
```

---

## Depth-Based Behavior

| Depth | Role | Can Spawn? | Max Spawns | Behavior |
|-------|------|------------|------------|----------|
| 1 | Prime Worker | Yes | 4 | Coordinate phase, spawn specialists |
| 2 | Specialist | Yes (if surprised) | 2 | Focused work, spawn only for unexpected complexity |
| 3 | Deep Specialist | No | 0 | Complete work inline |

**Spawn Limits:**
- Total cap per phase: 10 workers
- Spawn only when genuinely surprised (3x+ expected complexity)

---

## Iterative Planning

### Iterative Research Loop

```
/ant:plan triggers:

for iteration in 1..50:

    Scout Ant (Research):
        → Explores codebase, web, docs
        → Returns findings + remaining gaps
        → Confidence contribution

    Route-Setter Ant (Planning):
        → Drafts/refines phase breakdown
        → Rates confidence across 5 dimensions
        → Returns plan + confidence score

    if confidence >= 95%:
        break

    # Anti-stuck checks:
    if gap stuck for 3 iterations → mark needs human input
    if confidence delta < 5% for 3 iterations → pause for user
    if confidence delta < 2% after iteration 10 → offer to accept
```

### Confidence Dimensions

| Dimension | Weight | Measures |
|-----------|--------|----------|
| Knowledge | 25% | Understanding of codebase |
| Requirements | 25% | Clarity of success criteria |
| Risks | 20% | Identification of blockers |
| Dependencies | 15% | What affects what |
| Effort | 15% | Relative task complexity |

---

## Constraints System

Simple, declarative guidance replacing the complex pheromone system.

### Storage

```json
{
  "version": "1.0",
  "focus": ["area1", "area2"],
  "constraints": [
    {
      "id": "c_123",
      "type": "AVOID",
      "content": "pattern to avoid",
      "source": "user:redirect"
    }
  ]
}
```

### Commands

| Command | Effect |
|---------|--------|
| `/ant:focus "area"` | Add to focus list (max 5) |
| `/ant:redirect "pattern"` | Add AVOID constraint (max 10) |

### What Changed

| Before (Pheromones) | After (Constraints) |
|---------------------|---------------------|
| Decay over time | Persist until removed |
| Sensitivity profiles | Workers read all constraints |
| Signal strength math | Simple list lookup |
| Complex TTL logic | No expiration |

---

## Live Visibility

### tmux Watch Session

```
/ant:watch creates:

┌─────────────────────────┬───────────────────────────────┐
│ Status                  │ Activity Log                   │
│                         │                                │
│ State: EXECUTING        │ [10:05:01] START prime-worker │
│ Phase: 1/3              │ [10:05:03] SPAWN builder-1    │
│ Confidence: 95%         │ [10:05:05] CREATED src/api.ts │
│                         │ [10:05:08] COMPLETE builder-1 │
│ Active Workers:         │ [10:05:09] SPAWN watcher-1    │
│   [Prime] Coordinating  │                                │
│   [Builder] Implementing│                                │
├─────────────────────────┤                                │
│ Progress                │                                │
│ [████████████░░░░░░] 67%│                                │
└─────────────────────────┴───────────────────────────────┘
```

### Activity Log

Workers log as they work:
```bash
bash ~/.aether/aether-utils.sh activity-log "ACTION" "caste" "description"
```

Actions: CREATED, MODIFIED, RESEARCH, SPAWN, ERROR, COMPLETE

---

## Worker Castes

### Prime Worker (Depth 1 only)

Coordinator role. Analyzes phase tasks, decides what to delegate, spawns specialists, synthesizes results.

### Builder

Implements code, executes commands, manipulates files.

### Watcher

Validates, tests, ensures quality. Mandatory quality gate.

### Scout

Researches, searches docs, gathers context.

### Colonizer

Explores codebase, maps structure, detects patterns.

### Route-Setter

Creates plans, decomposes goals, analyzes dependencies.

### Architect

Synthesizes knowledge, extracts patterns, coordinates documentation.

---

## Visual Checkpoint

For phases that touch UI:

1. Prime Worker reports `ui_touched: true`
2. Queen prompts: "Visual checkpoint - verify appearance?"
3. User approves/rejects
4. Recorded in events

---

## State (v3.0)

Simplified from v2.0:

```json
{
  "version": "3.0",
  "goal": "...",
  "state": "IDLE | READY | PLANNING | EXECUTING",
  "current_phase": 0,
  "session_id": "...",
  "initialized_at": "...",
  "build_started_at": null,
  "plan": {
    "generated_at": "...",
    "confidence": 95,
    "phases": [...]
  },
  "memory": {
    "phase_learnings": [...],
    "decisions": [...]
  },
  "errors": {
    "records": [...],
    "flagged_patterns": [...]
  },
  "events": [...]
}
```

**Removed:**
- `mode`, `mode_set_at`, `mode_indicators`
- `workers` status tracking (workers are ephemeral)
- `spawn_outcomes` Bayesian tracking
- `signals` array (replaced by constraints.json)

---

## Command Reference

| Command | Purpose |
|---------|---------|
| `/ant:init "goal"` | Initialize colony with intention |
| `/ant:plan` | Iterative planning until 95% confidence |
| `/ant:build N` | Build phase N with Prime Worker |
| `/ant:continue` | Advance to next phase |
| `/ant:focus "area"` | Add focus constraint |
| `/ant:redirect "pattern"` | Add avoid constraint |
| `/ant:status` | Quick colony status |
| `/ant:watch` | Set up tmux for live viewing |

---

## What This Achieves

**Simpler:**
- 415-line build.md → 150 lines
- No wave planning, no SPAWN REQUEST parsing
- No pheromone decay math

**More Emergent:**
- Workers actually spawn workers
- Structure emerges from work
- Prime Worker self-organizes

**More Observable:**
- tmux live view
- Activity log streaming
- Confidence tracking

---

**This architecture represents true emergence: Queen sets intention, workers self-organize.**
