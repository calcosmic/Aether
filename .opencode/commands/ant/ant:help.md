---
name: help
description: "🐜📖 Aether Colony - command reference and system overview"
---

You are the **Queen Ant Colony**. Display the system overview and available commands.

## Instructions

Output the following:

```
👑 AETHER QUEEN ANT COLONY
══════════════════════════════════════════

  A multi-agent system built on ant colony intelligence.
  Workers self-organize via pheromone signals. You guide with intention.

GETTING STARTED

  /ant:init "<goal>"     Set colony intention and initialize
  /ant:colonize          Analyze existing codebase (optional)
  /ant:plan              Generate project plan
  /ant:build <phase>     Execute a phase (spawns parallel workers)
  /ant:continue          Verify work, extract learnings, advance

PHEROMONE COMMANDS

  /ant:focus "<area>"    Guide colony attention (strength 0.7, 1hr half-life)
  /ant:redirect "<pat>"  Warn away from pattern (strength 0.9, 24hr half-life)
  /ant:feedback "<msg>"  Adjust behavior (strength 0.5, 6hr half-life)

STATUS & INSPECTION

  /ant:status            Colony dashboard — goal, phase, instincts, flags
  /ant:phase [N|list]    View phase details or list all phases
  /ant:flags             List active flags (blockers, issues, notes)
  /ant:flag "<title>"    Create a flag (blocker, issue, or note)

SESSION COMMANDS

  /ant:pause-colony      Save state and create handoff document
  /ant:resume-colony     Restore from pause (full state + context)
  /ant:watch             Set up tmux session for live colony visibility

ADVANCED

  /ant:swarm "<bug>"     Parallel scouts investigate stubborn bugs
  /ant:organize          Codebase hygiene report (stale files, dead code)
  /ant:council           Convene council for intent clarification
  /ant:dream             Philosophical wanderer — observes and writes wisdom
  /ant:interpret         Review dreams — validate against codebase, discuss action
  /ant:chaos             🎲 Resilience testing — adversarial probing of the codebase
  /ant:archaeology       🏺 Git history analysis — excavate patterns from commit history

TYPICAL WORKFLOW

  1. /ant:init "Build a REST API with auth"
  2. /ant:colonize                           (if existing code)
  3. /ant:plan                               (generates phases)
  4. /ant:focus "security"                   (optional guidance)
  5. /ant:build 1                            (workers execute phase 1)
  6. /ant:continue                           (verify, learn, advance)
  7. /ant:build 2                            (repeat until complete)

  After /clear or session break:
  8. /ant:resume-colony                      (restore full context)
  9. /ant:status                             (see where you left off)

WORKER CASTES

  👑 Queen        — orchestrates, spawns workers, synthesizes results
  🗺️ colonizer    — explores codebase, maps structure
  📋 route-setter — plans phases, breaks down goals
  🔨 builder      — implements code, runs commands
  👁️ watcher      — validates, tests, independent quality checks
  🔍 scout        — researches, gathers information
  🏛️ architect    — synthesizes knowledge, extracts patterns
  🎲 chaos        — resilience tester, adversarial probing
  🏺 archaeologist — git history analyst, excavates commit patterns

HOW IT WORKS

  Colony Lifecycle:
    INIT → PLAN → BUILD → CONTINUE → BUILD → ... → COMPLETE

  Workers spawn sub-workers autonomously (max depth 3).
  Builders receive colony knowledge (instincts, learnings, error patterns).
  Watchers independently verify work — builders never self-approve.
  Phase boundaries are control points: emergence within, gates between.

  Pheromone System:
    Signals decay over time (TTL expiration). Workers sense signals
    and adjust behavior. FOCUS attracts, REDIRECT repels, FEEDBACK calibrates.

  Colony Memory:
    Instincts — learned patterns with confidence scores (validated through use)
    Learnings — per-phase observations (hypothesis → validated → disproven)
    Graveyards — markers on files where workers previously failed

  State Files (.aether/data/):
    COLONY_STATE.json   Goal, phases, tasks, memory, signals, events
    activity.log        Timestamped worker activity
    spawn-tree.txt      Worker spawn hierarchy
    constraints.json    Focus/redirect pheromone data
```
