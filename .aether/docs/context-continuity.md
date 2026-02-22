# Context Continuity Plan

Updated: 2026-02-22

## Goal

Keep colony context stable across sessions and long conversations without high token cost.

## Design Principles

1. Retrieval-first, not full-history injection.
2. Hard token budgets on injected context.
3. Derived context files are caches, never source-of-truth.
4. Promotion and pheromone emission are deterministic and auditable.

## Definitive Implementation Plan

### Phase 1: Low-Token Context Backbone (implemented)

1. Add `context-capsule` runtime command in `.aether/aether-utils.sh`.
2. Generate compact capsule from:
   - `COLONY_STATE.json` (goal, phase, decisions, next action)
   - `pheromones.json` (priority-sorted active signals)
   - `flags.json` (open blockers/issues)
   - `rolling-summary.log` (latest narrative entries)
3. Enforce compact limits (`max-signals`, `max-decisions`, `max-risks`, `max-words`).
4. Add `rolling-summary` runtime command:
   - `add <event> <summary> [source]`
   - `read [--json]`
   - keep last 15 entries only.
5. Extend `memory-capture` to:
   - support `resolution` events,
   - append each captured event to rolling summary.
6. Extend `pheromone-prime`:
   - `--compact`,
   - `--max-signals`,
   - `--max-instincts`,
   - include `POSITION` signals in output.
7. Extend `colony-prime --compact`:
   - use compact pheromone priming,
   - append `context-capsule` block to prompt payload.

### Phase 2: Injection Wiring (implemented for critical flows)

1. Build flow:
   - `build-context.md` now calls `colony-prime --compact`.
   - Worker prompt context now includes capsule + top signals via `prompt_section`.
2. Planning flow:
   - `/ant:plan` (Claude/OpenCode) now loads `context-capsule --compact --json`.
   - Scout + Route-Setter prompts include `context_capsule_prompt`.
3. Continue flow:
   - `continue-advance.md` now records recurring pattern resolution candidates through `memory-capture "resolution"`.

### Phase 3: Promotion Intelligence (next)

1. Add explicit recurrence-based promotion reasons in `learning-promote-auto` output.
2. Add deterministic “failure signature -> resolution signature” matching.
3. Promote matched recurring fixes to QUEEN wisdom faster than generic patterns.

### Phase 4: Session-Wide Coverage (next)

1. Inject `context-capsule` into remaining long-running orchestration commands:
   - `/ant:continue`,
   - `/ant:resume`,
   - `/ant:swarm`,
   - `/ant:oracle`.
2. Add per-command context budget checks and fallback degradation order.

## Current Runtime Surfaces

- `context-capsule`
- `rolling-summary`
- `pheromone-prime --compact`
- `colony-prime --compact`
- `memory-capture` (`resolution` + rolling summary append)

## Verification

Unit coverage added in `tests/unit/context-continuity.test.js`:

1. Capsule generation and next-action extraction.
2. Compact priming signal cap behavior.
3. Rolling summary retention cap (15 entries).
4. Memory capture -> rolling summary integration.
