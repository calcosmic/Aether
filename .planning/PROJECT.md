# Aether: Wire the Colony Together

## What This Is

Aether is a self-managing development assistant that uses ant colony metaphor to orchestrate AI workers across coding sessions. It has 36 commands, 22 agents, and ~10,000 lines of shell infrastructure. The system was designed to be self-improving — learning from each project and getting smarter over time. But the learning and feedback systems are disconnected: data gets captured but never reaches the workers who need it. This project wires the existing systems together so Aether actually becomes the self-improving colony it was designed to be.

## Core Value

Workers should automatically receive all relevant context — learnings, decisions, error patterns, wisdom — without manual intervention. The colony improves itself.

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

### Active

- [ ] Phase learnings auto-inject into future builder prompts
- [ ] Key decisions auto-convert to FEEDBACK pheromones
- [ ] Recurring error patterns auto-emit REDIRECT pheromones
- [ ] Learning observations auto-promote to QUEEN.md when thresholds met
- [ ] Escalated flags inject as warnings into next phase builders
- [ ] colony-prime reads CONTEXT.md decisions for builder injection
- [ ] instinct-create actually called during continue flow
- [ ] instinct-read results included in colony-prime output
- [ ] queen-promote called during seal and continue flows
- [ ] Success criteria patterns create instincts on recurrence

### Out of Scope

- New commands — connect what exists, don't add surface area
- New agents — 22 is enough, wire them better
- UI/visual changes — this is plumbing, not paint
- Cross-colony wisdom sharing — solve single-colony learning first
- Model routing verification — separate concern
- XML migration — do gradually as files are touched

## Context

Aether v1.1.11 is the current version. 490+ tests passing. 8 milestones shipped (v1.0 through v5.0). The system has been used to build itself across those milestones, accumulating 11 learning observations (some seen 3+ times across colonies) — but QUEEN.md has zero entries because queen-promote is never called. Instincts array is empty because instinct-create is never called. Phase learnings are stored but never read back.

The architecture is sound. The utilities work. The gap is purely in the command playbooks and colony-prime not reading/writing the data that already exists.

Key files to modify:
- `.aether/docs/command-playbooks/build-context.md` — where builder context is assembled
- `.aether/docs/command-playbooks/build-wave.md` — where builders get their prompts
- `.aether/docs/command-playbooks/continue-advance.md` — where learnings/instincts should be created
- `.aether/docs/command-playbooks/continue-finalize.md` — where promotion checks should run
- `.aether/aether-utils.sh` — colony-prime and pheromone-prime functions
- `.claude/commands/ant/seal.md` — where final wisdom promotion should happen

## Constraints

- **No new commands** — only modify existing command playbooks and utility functions
- **No new state files** — use existing JSON structures (COLONY_STATE, pheromones, learning-observations)
- **Backward compatible** — existing colonies must not break
- **Must work in Claude Code** — all output via unicode/emoji, no ANSI
- **Bash 3.2 compatible** — macOS ships bash 3.2
- **Test coverage** — new behavior needs tests in existing test framework

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Connect, don't add | System has all pieces, just disconnected | -- Pending |
| Modify playbooks, not commands | Commands are orchestrators, playbooks define behavior | -- Pending |
| colony-prime is the integration point | Single function that assembles all context for workers | -- Pending |
| Auto-promotion with thresholds | queen-promote already has threshold logic, just call it | -- Pending |

---
*Last updated: 2026-03-06 after initialization*
