---
name: help
description: "🐜📖 Aether Colony - command reference and system overview"
---

You are the **Queen Ant Colony**. Display the system overview and available commands.

## Instructions

Output the following:

```
👑 AETHER QUEEN ANT COLONY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  A multi-agent system built on ant colony intelligence.
  Workers self-organize via pheromone signals. You guide with intention.

SETUP & GETTING STARTED

  /ant:lay-eggs          Set up Aether in this repo (one-time, creates .aether/)
  /ant:init "<goal>"     Start a colony with a goal
  /ant:colonize          Analyze existing codebase (optional)
  /ant:plan              Generate project plan
  /ant:build <phase>     Execute a phase (spawns parallel workers)
  /ant:continue          Verify work, extract learnings, advance

PHEROMONE COMMANDS

  /ant:focus "<area>"    Guide colony attention (priority: normal, expires: phase end)
  /ant:redirect "<pat>"  Warn away from pattern (priority: high, expires: phase end)
  /ant:feedback "<msg>"  Adjust behavior (priority: low, expires: phase end)
  /ant:pheromones        View and manage active pheromone signals

STATUS & UPDATES

  /ant:status            Colony dashboard — goal, phase, instincts, flags
  /ant:update            Update system files from global hub (~/.aether/)
  /ant:phase [N|list]    View phase details or list all phases
  /ant:insert-phase      Insert a corrective phase after current phase
  /ant:flags             List active flags (blockers, issues, notes)
  /ant:flag "<title>"    Create a flag (blocker, issue, or note)
  /ant:memory-details    Show detailed colony memory — wisdom, promotions, failures
  /ant:maturity          View colony maturity journey and milestone progress

SESSION COMMANDS

  /ant:pause-colony      Save state and create handoff document
  /ant:resume-colony     Restore from pause (full state + context)
  /ant:resume            Quick session restore (memory health + next steps)
  /ant:watch             Set up tmux session for live colony visibility

COLONY LIFECYCLE

  /ant:seal             Seal colony with Crowned Anthill milestone
  /ant:entomb           Archive completed colony into chambers
  /ant:history          Browse colony event history

ADVANCED

  /ant:swarm "<bug>"     Parallel scouts investigate stubborn bugs
  /ant:oracle            Deep research agent using RALF iterative loop
  /ant:organize          Codebase hygiene report (stale files, dead code)
  /ant:council           Convene council for intent clarification
  /ant:dream             Philosophical wanderer — observes and writes wisdom
  /ant:interpret         Review dreams — validate against codebase, discuss action
  /ant:chaos             🎲 Resilience testing — adversarial probing of the codebase
  /ant:archaeology       🏺 Git history analysis — excavate patterns from commit history
  /ant:tunnels           Browse archived colonies and compare chambers

MAINTENANCE

  /ant:migrate-state     One-time state migration from v1 to v2.0 format
  /ant:verify-castes     Verify colony caste assignments and system status

TYPICAL WORKFLOW

  First time in a repo:
  0. /ant:lay-eggs                           (set up Aether in this repo)

  Starting a colony:
  1. /ant:init "Build a REST API with auth"  (start colony with a goal)
  2. /ant:colonize                           (if existing code)
  3. /ant:plan                               (generates phases)
  4. /ant:focus "security"                   (optional guidance)
  5. /ant:build 1                            (workers execute phase 1)
  6. /ant:continue                           (verify, learn, advance)
  7. /ant:build 2                            (repeat until complete)

  After /clear or session break:
  8. /ant:resume-colony                      (restore full context)
  9. /ant:status                             (see where you left off)

  After completing a colony:
  10. /ant:seal                              (mark as complete)
  11. /ant:entomb                            (archive to chambers)
  12. /ant:init "next project goal"          (start fresh colony)

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
    Signals expire after their TTL. Workers sense active signals
    and adjust behavior. FOCUS attracts, REDIRECT repels, FEEDBACK calibrates.

  Colony Memory:
    Instincts — learned patterns with confidence scores (validated through use)
    Learnings — per-phase observations (hypothesis → validated → disproven)
    Graveyards — markers on files where workers previously failed

  State Files (.aether/data/):
    COLONY_STATE.json   Goal, phases, tasks, memory, events
    activity.log        Timestamped worker activity
    spawn-tree.txt      Worker spawn hierarchy
    pheromones.json     Active FOCUS/REDIRECT/FEEDBACK signals
    constraints.json    Compatibility mirror for focus/redirect data
```

### Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
