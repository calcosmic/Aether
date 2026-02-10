---
name: ant
description: "🐜🐜👑🏰🐜🐜🐜 Queen Ant Colony - phased autonomy where user provides intention via pheromones"
---

You are the **Queen Ant Colony**. Display the system overview and available commands.

## Instructions

Output the following:

```
👑 AETHER QUEEN ANT COLONY

  A multi-agent system built on ant colony intelligence.
  Workers self-organize via pheromone signals. You guide with intention.

GETTING STARTED

  /ant:init "<goal>"     Set colony intention and initialize
  /ant:colonize          Analyze existing codebase (optional)
  /ant:plan              Generate project plan
  /ant:build <phase>     Execute a phase

PHEROMONE COMMANDS

  /ant:focus "<area>"    Guide colony attention (strength 0.7, 1hr half-life)
  /ant:redirect "<pat>"  Warn away from pattern (strength 0.9, 24hr half-life)
  /ant:feedback "<msg>"  Adjust behavior (strength 0.5, 6hr half-life)

STATUS COMMANDS

  /ant:status            Colony status, workers, pheromones, progress
  /ant:phase [N|list]    View phase details or list all phases
  /ant:continue          Approve phase and advance to next

SESSION COMMANDS

  /ant:pause-colony      Save state for session break
  /ant:resume-colony     Restore from pause

TYPICAL WORKFLOW

  1. /ant:init "Build a REST API with auth"
  2. /ant:colonize                           (if existing code)
  3. /ant:plan                               (generates phases)
  4. /ant:focus "security"                   (optional guidance)
  5. /ant:build 1                            (execute phase 1)
  6. /ant:continue                           (advance to phase 2)
  7. /ant:build 2                            (repeat)

WORKER CASTES

  🗺️🐜 colonizer    — explores codebase, maps structure
  📋🐜 route-setter — plans phases, breaks down goals
  🔨🐜 builder      — implements code, runs commands
  👁️🐜 watcher      — validates, tests, quality checks
  🔍🐜 scout        — researches, gathers information
  🏛️🐜 architect    — synthesizes knowledge, extracts patterns

HOW IT WORKS

  The Aether Colony is a multi-agent system inspired by ant colony intelligence.

  Colony Lifecycle:
    1. INIT: Queen sets intention (goal). Colony mobilizes. State: IDLE -> READY.
    2. PLAN: Route-setter decomposes goal into phases. State: READY -> PLANNING -> READY.
    3. BUILD: Workers execute phases. Spawn sub-workers as needed. State: READY -> EXECUTING.
    4. CONTINUE: Queen approves phase, extracts learnings. Advances to next phase.
    5. Repeat BUILD/CONTINUE until all phases complete.

  Pheromone System:
    Signals have TTL expiration (expires_at timestamp). Workers sense signals
    and adjust behavior. FOCUS attracts, REDIRECT repels, FEEDBACK calibrates.

  Autonomy Model:
    Workers spawn sub-workers autonomously (max depth 3, max 5 active).
    Spawn outcomes tracked per caste (success/fail counts).
    Phase boundaries are control points -- emergence happens within phases.

  State File (.aether/data/):
    COLONY_STATE.json  Unified colony state containing:
      - goal, state, current_phase, workers, spawn_outcomes
      - plan.phases (phase breakdown and task tracking)
      - signals (pheromone signals with TTL expiration)
      - memory (phase learnings, decisions, patterns)
      - errors (records, flagged patterns)
      - events (colony event log)
```
