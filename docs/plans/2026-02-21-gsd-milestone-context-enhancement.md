# GSD Milestone Request: Aether Colony Context Enhancement

**Project:** Aether (npm package: aether-colony)
**Repository:** /Users/callumcowie/repos/Aether
**Type:** Research → Design → Implementation
**Created:** 2026-02-21

---

## Problem Statement

Aether is a multi-agent orchestration system for Claude Code where AI workers self-organize around goals using ant colony metaphors. The system works, but has a critical weakness: **context does not persist well across sessions.**

### The Core Problem

When a new Claude Code session starts, the colony "forgets" where it was. The current state system (`COLONY_STATE.json`) is:
- Machine-readable (JSON) but sparse
- Missing project-level context (why does this colony exist?)
- Missing decision history (why were choices made?)
- Missing rich phase documentation (what happened in each phase?)
- Missing explicit session continuity (where exactly do I resume?)

**Result:** Claude improvises instead of following established patterns. Work is repeated. Decisions are forgotten.

---

## Current Architecture (Research Starting Point)

### What Exists

**File Structure:**
```
.aether/
├── aether-utils.sh        # 80+ subcommands, ~3800 lines
├── workers.md             # Worker definitions
├── CONTEXT.md             # Manual context document (EXISTS but manual)
├── templates/             # 9 templates (EXISTS)
│   ├── QUEEN.md.template
│   ├── colony-state.template.json
│   ├── constraints.template.json
│   └── handoff templates...
│
├── data/
│   ├── COLONY_STATE.json  # Current state (sparse JSON)
│   ├── pheromones.json    # Pheromone signals (EXISTS)
│   ├── constraints.json   # Focus/redirect (EXISTS)
│   ├── session.json       # Session tracking
│   ├── learnings.json     # Learning storage
│   ├── queen-wisdom.json  # Promoted patterns
│   ├── survey/            # Territory survey results
│   ├── midden/            # Waste (ant-themed, EXISTS)
│   └── backups/
│
├── chambers/              # Archived colonies
├── oracle/                # Deep research system
└── docs/                  # System documentation
```

**COLONY_STATE.json Schema:**
```json
{
  "version": "3.0",
  "goal": "...",
  "state": "READY",
  "current_phase": 0,
  "session_id": "...",
  "plan": { "phases": [] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "signals": [],
  "events": []
}
```

**Key Observation:** The `memory.decisions` array is always empty. The `plan.phases` array is sparse. There's no rich documentation of what happened or why.

---

## Desired Outcome

A context system that enables **instant session restoration** — a new Claude session can read 2-3 files and know:
1. What the colony is building (project context)
2. Where we are right now (current position)
3. What decisions were made and why (decision history)
4. Exactly what to do next (session continuity)

The system should be:
- **Rich enough** for Claude to maintain coherence
- **Simple enough** to not overwhelm
- **Backwards compatible** with existing colonies
- **Ant-themed** in naming (see reference below)

---

## Research Questions (Let GSD Decide)

1. **Should we enhance existing files or create new ones?**
   - Enhance `CONTEXT.md` to be auto-generated?
   - Create new `NEST.md` for project context?
   - Both?

2. **How should decisions be logged?**
   - Add to `COLONY_STATE.json`?
   - Create separate `decrees.md`?
   - Both with sync?

3. **How should phases be documented?**
   - Enhance `plan.phases[]` in JSON?
   - Create `TUNNELS/{N}/` folders with markdown?
   - Hybrid approach?

4. **How should pheromones be organized?**
   - Keep single `pheromones.json`?
   - Split into `TRAILS/focus.json`, `TRAILS/redirect.json`?
   - Both?

5. **How should session continuity work?**
   - Enhance `session.json`?
   - Create `.continue-here.md`?
   - Both?

6. **What templates are needed?**
   - Currently have 9 templates
   - Need more? Which ones?

---

## Biological Naming Reference

Aether uses ant colony metaphors. All naming should follow this pattern:

| Concept | Ant-Themed Name | Biological Basis |
|---------|-----------------|------------------|
| Project context | NEST.md or enhance CONTEXT.md | The colony's nest |
| Current state | ROYAL-CHAMBER/ or enhance data/ | Queen's active area |
| Phases | TUNNELS/ or enhance plan.phases | Ants dig tunnels |
| Phase plan | EXCAVATION.md or enhance existing | Excavating new tunnel |
| Phase summary | DEPOSIT.md or enhance existing | Depositing findings |
| Decisions | decrees or enhance decisions[] | Royal decrees |
| Signals | TRAILS/ or enhance pheromones.json | Pheromone trails |
| Waste/blockers | midden/ (already exists!) | Waste pile |
| Work in progress | BROOD/ or enhance existing | Larvae being nurtured |

**Existing ant-themed names to preserve:**
- `midden/` — waste pile (already in use)
- `chambers/` — archived colonies (already in use)
- `oracle/` — deep research (already in use)
- `workers.md` — worker definitions (already in use)

---

## Constraints

1. **Backwards compatibility required** — existing colonies must not break
2. **No data loss** — migration path for existing data
3. **Ant-themed naming** — no mixed metaphors
4. **Template-based** — generated files should use templates
5. **Single source of truth where possible** — avoid duplication
6. **Context reading < 5 minutes** — Queen can understand quickly

---

## Success Criteria

The implementation is successful when:

1. A new session can read context and understand the project
2. Current position is clear (which phase, which task)
3. Decisions are logged with rationale
4. Session can be resumed exactly where it left off
5. All existing colonies still work
6. All existing tests pass

---

## Reference Files (Research Starting Points)

**Read these first:**
- `.aether/data/COLONY_STATE.json` — Current state schema
- `.aether/CONTEXT.md` — Existing context document
- `.aether/aether-utils.sh` — Existing subcommands (look at state management)
- `.aether/templates/` — Existing templates
- `.claude/commands/ant/init.md` — How initialization works
- `.claude/commands/ant/continue.md` — How continuation works
- `.claude/commands/ant/seal.md` — How archiving works

**Reference for GSD patterns:**
- `.planning/PROJECT.md` — How GSD does project context
- `.planning/STATE.md` — How GSD does state tracking
- `.planning/phases/` — How GSD does phase folders

---

## What We Need From GSD

1. **Research Phase:** Investigate current Aether architecture, understand what exists, identify gaps
2. **Design Phase:** Propose implementation approach (enhance existing vs. create new)
3. **Plan Phase:** Break down into implementable phases
4. **Implementation:** Execute with backwards compatibility

**Do not prescribe exact implementation.** Let GSD research and decide based on:
- What already exists (don't duplicate)
- What needs to be created
- How to integrate safely
- Migration path for existing data

---

## Starting Command

```
/gsd:init "Enhance Aether colony context system to enable instant session restoration with rich project context, decision logging, and phase documentation. Research current architecture first, then design backwards-compatible implementation."
```

---

## Final Note

The goal is NOT to copy GSD's system, but to learn from its patterns and apply them to Aether's ant-themed architecture. The solution should feel native to Aether, not like GSD was grafted on.

What matters most:
- Can a new session understand the project quickly?
- Can we resume exactly where we left off?
- Are decisions preserved with rationale?
- Does it feel like an ant colony?
