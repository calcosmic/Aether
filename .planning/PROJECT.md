# Aether: Wire the Colony Together

## What This Is

Aether is a self-managing development assistant that uses ant colony metaphor to orchestrate AI workers across coding sessions. It has 36 commands, 22 agents, and ~10,000 lines of shell infrastructure. The colony is now fully self-improving: learnings, decisions, error patterns, and wisdom are automatically captured during `/ant:continue` and injected into builders during `/ant:build` via colony-prime. The learning lifecycle is complete from observation to QUEEN.md wisdom.

## Core Value

Workers automatically receive all relevant context -- learnings, decisions, error patterns, wisdom -- without manual intervention. The colony improves itself.

## Requirements

### Validated

- Colony command infrastructure (36 commands, all functional)
- Pheromone signal system (FOCUS/REDIRECT/FEEDBACK emit and display)
- colony-prime injection (pheromones reach builders via prompt_section)
- Midden failure tracking (recent failures shown to builders)
- Graveyard file cautions (unstable files flagged to builders)
- Survey territory intelligence (codebase patterns fed to builders)
- State persistence across sessions (COLONY_STATE.json, CONTEXT.md)
- Memory-capture pipeline (learning-observe, observation counting)
- Instinct infrastructure (instinct-create, instinct-read exist)
- QUEEN.md infrastructure (queen-init, queen-read, queen-promote exist)
- Suggest-analyze/approve pipeline (pheromone suggestions exist)
- Phase learnings auto-inject into future builder prompts -- v1.0
- Key decisions auto-convert to FEEDBACK pheromones -- v1.0
- Recurring error patterns auto-emit REDIRECT pheromones -- v1.0
- Learning observations auto-promote to QUEEN.md when thresholds met -- v1.0
- Escalated flags inject as warnings into next phase builders -- v1.0
- colony-prime reads CONTEXT.md decisions for builder injection -- v1.0
- instinct-create called during continue flow with confidence >= 0.7 -- v1.0
- instinct-read results included in colony-prime output (domain-grouped) -- v1.0
- queen-promote called during seal and continue flows -- v1.0
- Success criteria patterns create instincts on recurrence -- v1.0

### Active

(No active requirements -- next milestone TBD)

### Out of Scope

- Cross-colony wisdom sharing -- solve single-colony learning first
- Model routing verification -- separate concern
- XML migration -- do gradually as files are touched

## Context

Aether v1.1.11 with v1.0 colony wiring shipped. 535+ tests passing (490 existing + 45 new integration tests). The colony-prime function in aether-utils.sh now assembles 6 context sections for builders: QUEEN WISDOM, CONTEXT CAPSULE, PHASE LEARNINGS, KEY DECISIONS, BLOCKER WARNINGS, and ACTIVE SIGNALS + INSTINCTS.

The continue-advance playbook creates instincts (3 sources) and auto-emits pheromones (decisions, errors, success). The continue-finalize playbook runs batch wisdom auto-promotion. The seal command runs batch auto-promotion before interactive review.

## Constraints

- **No new commands** -- only modify existing command playbooks and utility functions
- **No new state files** -- use existing JSON structures (COLONY_STATE, pheromones, learning-observations)
- **Backward compatible** -- existing colonies must not break
- **Must work in Claude Code** -- all output via unicode/emoji, no ANSI
- **Bash 3.2 compatible** -- macOS ships bash 3.2
- **Test coverage** -- new behavior needs tests in existing test framework

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Connect, don't add | System has all pieces, just disconnected | Good -- 0 new commands, 0 new state files |
| Modify playbooks, not commands | Commands are orchestrators, playbooks define behavior | Good -- all changes in playbooks + colony-prime |
| colony-prime is the integration point | Single function that assembles all context for workers | Good -- hub-and-spoke architecture works cleanly |
| Auto-promotion with thresholds | queen-promote already has threshold logic, just call it | Good -- learning-promote-auto with grep guard |
| Confidence floor 0.7 for instincts | Only validated patterns become instincts | Good -- prevents noise |
| auto: source prefix namespace | Distinguishes auto-emitted from manual pheromones | Good -- auto:decision, auto:error, auto:success |
| Prompt assembly order | QUEEN WISDOM first (highest priority), signals last (most volatile) | Good -- natural information hierarchy |
| Batch sweep + grep guard | Safe to run auto-promotion multiple times | Good -- idempotent, no double-promotion |

---
*Last updated: 2026-03-07 after v1.0 milestone*
