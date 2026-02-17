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

### Out of Scope

- **Advanced features** (defer until foundation solid):
  - XML deep integration
  - Cross-colony communication
  - Oracle deep research
  - Chambers/Seal/Entomb lifecycle
  - Pheromone auto-injection
  - Self-learning queen markdown

- **Nice-to-have** (defer):
  - Visual progress bars (unless easy)
  - Multiple color schemes
  - Advanced analytics

## Context

**What was built (extensive):**
- 21 worker castes with detailed disciplines (TDD, verification, debugging, etc.)
- Pheromone signaling system (FOCUS, REDIRECT, FEEDBACK)
- Session freshness detection
- Spawn tracking and swarm visualization
- Comprehensive workers.md (769+ lines)
- Detailed command definitions for each ant role

**What broke:**
- Commands produce wrong results
- Continuous errors (401s, etc.)
- Agents spawn forever without stopping
- Visual display doesn't work — just bash text
- Hallucinations — looks for wrong files
- Files created in wrong repos

**Why it broke:**
- Implementation didn't follow the design
- Too many changes made too quickly
- No proper testing of changes
- Context collapsed during development

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
