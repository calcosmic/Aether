# Aether Repair & Stabilization

## What This Is

A self-managing development assistant using ant colony metaphor that prevents context rot. Users install it, it guides them through work with clear commands, tells them when to clear context, and maintains state across sessions. The colony learns from each phase and improves over time.

**Current State:** The system has extensive, well-designed documentation and architecture, but the implementation is broken. Commands fail, files are in wrong places, visual display doesn't work, and context collapses frequently.

## Core Value

If everything else fell away, Aether's essential value is:
- **Context preservation** — prevents context rot across Claude Code sessions
- **Clear workflow guidance** — tells users what command to run next
- **Self-improving** — learns from each phase via pheromones/instincts

## Requirements

### Validated

(None yet — repair to validate)

### Active

- [ ] **Command Infrastructure** — Core commands work reliably
  - [ ] /ant:lay-eggs — Start new colony (preserves pheromones)
  - [ ] /ant:init — Initialize after lay-eggs
  - [ ] /ant:colonize — Analyze existing codebase
  - [ ] /ant:plan — Generate project plan
  - [ ] /ant:build — Execute phase
  - [ ] /ant:continue — Verify, learn, advance

- [ ] **Visual Experience** — Nice ant-themed display
  - [ ] Swarm display works (ants working, progress bars)
  - [ ] Emoji caste identity visible
  - [ ] Colors for different castes
  - [ ] Not "loads of bash text going down the screen"

- [ ] **Context Rot Prevention** — Core promise
  - [ ] Session state persists across /clear
  - [ ] Clear "next command" guidance at phase boundaries
  - [ ] Context document tells next session what was happening

- [ ] **State Integrity** — Files in right places
  - [ ] COLONY_STATE.json updates correctly
  - [ ] No file path hallucinations
  - [ ] Commands find the right files

- [ ] **XML Integration** — Deep data structure
  - [ ] Pheromones stored/retrieved via XML
  - [ ] Wisdom exchange uses XML
  - [ ] Registry uses XML

- [ ] **Pheromone System** — Self-learning signals
  - [ ] FOCUS/REDIRECT/FEEDBACK work
  - [ ] Auto-injection of learned patterns
  - [ ] Instincts applied to new work

- [ ] **Colony Lifecycle** — Seal/Entomb/Chambers
  - [ ] /ant:seal — Crowned Anthill milestone
  - [ ] /ant:entomb — Archive colony to chambers
  - [ ] /ant:tunnels — Browse archived colonies

- [ ] **Advanced Workers** — Specialized agents
  - [ ] /ant:oracle — Deep research (RALF loop)
  - [ ] /ant:chaos — Resilience testing
  - [ ] /ant:archaeology — Git history analysis
  - [ ] /ant:dream — Philosophical wanderer
  - [ ] /ant:interpret — Validate dreams against reality

### Out of Scope

(None — all features stay in scope, we repair what's broken)

## Context

**What's implemented (extensive):**
- 21 worker castes with detailed disciplines (TDD, verification, debugging, learning)
- Pheromone signaling system (FOCUS, REDIRECT, FEEDBACK)
- Session freshness detection
- Spawn tracking and swarm visualization
- Comprehensive workers.md (769+ lines)
- Detailed command definitions for each ant role
- XML deep integration (pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh)
- Cross-colony communication via pheromones
- Oracle deep research system
- Chaos resilience testing
- Archaeologist git history analysis
- Seal/Entomb/Chambers lifecycle management
- Self-learning via QUEEN.md and instincts
- Visual display system (swarm-display)

**What broke:**
- Visual display doesn't work — just bash text, not rotating ants
- Continuous errors (401s, agent problems)
- Agents spawn forever without stopping
- Hallucinations — looks for wrong files, makes things up
- Files created in wrong repos
- Commands don't connect properly
- Context still collapses

**Why it broke:**
- Too many changes made too quickly without proper testing
- Implementation didn't match the design
- Pieces not connected — XML exists but not wired in, visual exists but doesn't display, etc.
- Context collapsed during development, losing continuity

## Constraints

- **Must work in Claude Code** — primary platform
- **Visual simplicity** — no loads of terminal text
- **Reliability first** — working > feature-rich
- **Self-contained** — minimal external dependencies

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Repair first, features later | Don't add features to broken foundation | Pending |
| Visual mode optional | Not all users want pretty display | Pending |
| Single model for all workers | Platform limitation (Task tool) | ✓ Confirmed |

---

*Last updated: 2026-02-17 after initial repair assessment*
