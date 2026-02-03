# Phase 16 Research: Worker Knowledge

## Current State

### Worker Spec Inventory

6 worker specs in `.aether/workers/`, all 83-104 lines:

| File | Lines | Current Sections |
|------|-------|-----------------|
| watcher-ant.md | 104 | Purpose, Pheromone Sensitivity, Workflow, Validation Checklist (4 areas), Output Format, Spawning |
| builder-ant.md | 90 | Purpose, Pheromone Sensitivity, Workflow, Output Format, Implementation Principles, Feedback Response, Spawning |
| scout-ant.md | 97 | Purpose, Pheromone Sensitivity, Workflow, Research Strategies, Output Format, Quality Standards, Spawning |
| architect-ant.md | 86 | Purpose, Pheromone Sensitivity, Workflow, Pattern Extraction, Output Format, Quality Standards, Spawning |
| colonizer-ant.md | 83 | Purpose, Pheromone Sensitivity, Workflow, Output Format, Quality Standards, Spawning |
| route-setter-ant.md | 102 | Purpose, Pheromone Sensitivity, Workflow, Output Format, Planning Heuristics, Caste Assignment Guide, Spawning |

### Common Structure (all 6 specs)

Every spec has:
1. H1 title + "You are a **{Caste} Ant**..."
2. `## Purpose` — 1-2 sentences
3. `## Pheromone Sensitivity` — table with INIT/FOCUS/REDIRECT/FEEDBACK and numeric sensitivity values (0.0-1.0)
4. `## Workflow` — numbered steps (5-6 steps)
5. Some caste-specific sections
6. `## Output Format` — text block template
7. `## You Can Spawn Other Ants` — identical boilerplate in all 6 (lines ~60-end)

### Pheromone Sensitivity Values

| Caste | INIT | FOCUS | REDIRECT | FEEDBACK |
|-------|------|-------|----------|----------|
| colonizer | 1.0 | 0.7 | 0.3 | 0.5 |
| route-setter | 1.0 | 0.5 | 0.8 | 0.7 |
| builder | 0.5 | 0.9 | 0.9 | 0.7 |
| watcher | 0.3 | 0.8 | 0.5 | 0.9 |
| scout | 0.7 | 0.9 | 0.4 | 0.5 |
| architect | 0.2 | 0.4 | 0.3 | 0.6 |

### JSON State Schemas (from Phase 15)

**pheromones.json signals:**
```json
{
  "signals": [
    {
      "type": "FOCUS|REDIRECT|FEEDBACK",
      "content": "area/pattern/message",
      "strength": 0.0-1.0,
      "emitted_at": "ISO-8601",
      "half_life_minutes": 60|1440|360
    }
  ]
}
```

**events.json:**
```json
{
  "events": [
    {"id": "evt_...", "type": "colony_initialized|phase_advanced|...", "source": "init|build|continue|...", "content": "...", "timestamp": "ISO-8601"}
  ]
}
```

**memory.json:**
```json
{
  "phase_learnings": [{"id": "learn_...", "phase": N, "phase_name": "...", "learnings": ["..."], "errors_encountered": N, "timestamp": "..."}],
  "decisions": [{"id": "dec_...", "type": "focus|redirect|feedback", "content": "...", "context": "...", "phase": N, "timestamp": "..."}],
  "patterns": []
}
```

**errors.json:**
```json
{
  "errors": [{"id": "err_...", "category": "...", "severity": "...", "description": "...", "root_cause": "...", "phase": N, "task_id": "...", "timestamp": "..."}],
  "flagged_patterns": [{"category": "...", "count": N, "first_seen": "...", "last_seen": "..."}]
}
```

## What Needs to Be Added

### Plan 16-01: Watcher Specialist Modes

watcher-ant.md currently has a flat `## Validation Checklist` with 4 bullet sections (Security, Performance, Quality, Test Coverage). This needs to become 4 full specialist modes, each with:
- **Activation trigger** — what pheromone context activates this mode
- **Focus areas** — what to look at
- **Severity rubric** — Critical/High/Medium/Low definitions
- **Detection pattern checklist** — specific patterns to detect

The existing checklist content provides a starting point but needs significant expansion.

### Plan 16-02: Pheromone Math + Combination Effects + Feedback Interpretation (all 6 specs)

Each spec needs 3 new sections added between the existing Workflow and Output Format:

1. **Pheromone Math** — worked example: `sensitivity × strength = effective_signal`. Use each caste's actual sensitivity values from the table, show a concrete FOCUS signal at 0.7 strength, calculate the effective signal, and explain the threshold (e.g., >0.5 = act, 0.3-0.5 = note, <0.3 = ignore).

2. **Combination Effects** — what happens when conflicting signals are active (e.g., FOCUS on "auth" + REDIRECT away from "JWT" = focus on auth but use a different approach). 2-3 scenarios per caste.

3. **Feedback Interpretation** — how to interpret FEEDBACK pheromone content. builder-ant.md already has a basic version; the others need similar guidance tuned to their role.

### Plan 16-03: Event Awareness + Spawning Scenarios (all 6 specs)

Each spec needs 2 additions:

1. **Event Awareness at Startup** — new section near the top of Workflow. Read events.json, filter events from last 30 minutes (or since last phase boundary), describe how each event type affects behavior. Also read memory.json for relevant decisions/learnings (satisfies MEM-04 and EVT-03).

2. **Spawning Scenario** — replace the generic "You Can Spawn Other Ants" section with an enriched version that includes a complete worked example showing a full Task tool prompt with `--- WORKER SPEC ---`, `--- ACTIVE PHEROMONES ---`, `--- TASK ---` blocks filled in with realistic content. Each caste gets a scenario appropriate to their role.

## Design Decisions

1. **Section ordering in specs**: Purpose → Pheromone Sensitivity → Pheromone Math (NEW) → Combination Effects (NEW) → Workflow → Event Awareness at Startup (NEW, first step of workflow) → Feedback Interpretation (NEW) → [Caste-specific sections] → Output Format → Spawning Scenarios (ENRICHED)

2. **Target line counts**: ~180-200 lines per spec. Watcher may be longer (~220) due to specialist modes.

3. **Pheromone math threshold**: effective_signal > 0.5 = prioritize/act, 0.3-0.5 = note/consider, < 0.3 = ignore. Consistent across all castes.

4. **Event filtering**: Workers filter events by timestamp (last 30 min or since phase start). Use `new Date(timestamp) > cutoff` logic described in prose, not code.

5. **Memory reading**: Workers read decisions array for recent Queen guidance, phase_learnings for relevant past learnings. Brief — 2-3 sentences of instruction.

6. **Spawning examples**: Each caste gets a different target caste in their example to show variety and realistic scenarios.

## Wave Structure

- **Wave 1**: 16-01 (watcher modes) and 16-02 (pheromone math to all 6) — independent, can run in parallel
- **Wave 2**: 16-03 (event awareness + spawning to all 6) — depends on 16-02 since it modifies the same 6 files

## Risks

- **Line count**: Adding specialist modes to watcher may push it past 200 lines. Mitigation: keep each mode to ~15 lines (trigger, focus, rubric, patterns as compact list).
- **Repetitive content**: Pheromone math is structurally similar across specs. Mitigation: each caste uses different numeric values and different scenario narratives.
- **Spawning example length**: A full Task tool prompt example could be 20+ lines. Mitigation: keep the example realistic but concise (~15 lines).

## RESEARCH COMPLETE
