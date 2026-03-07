# Phase 1: Instinct Pipeline - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire instinct-create into the continue-advance flow so high-confidence patterns become instincts in COLONY_STATE.json, then wire instinct-read into colony-prime so builders receive those instincts in their prompts. Requirements: LEARN-02 (instinct creation), LEARN-03 (instinct injection).

</domain>

<decisions>
## Implementation Decisions

### Instinct Creation Rules
- All three pattern sources feed instinct creation: phase learnings, error patterns (midden), and success patterns
- Claude's Discretion: confidence threshold and volume cap per phase — Claude judges based on pattern strength and signal quality
- When a new pattern matches an existing instinct, strengthen the existing instinct (bump confidence score) rather than creating a duplicate
- instinct-create must be called in continue-advance.md after learnings are extracted

### Builder Prompt Format
- Instincts grouped by domain in the prompt (testing instincts together, architecture instincts together, etc.)
- Claude's Discretion: verbosity level per instinct — Claude picks between terse one-liners and guided context based on instinct type
- Injected instincts must be visible in build output so the user can see what the colony knows (not silent)
- Claude's Discretion: prompt budget management — Claude decides what gets cut when space is limited

### Instinct Lifecycle
- Claude's Discretion: expiry/decay behavior — Claude manages instinct freshness
- A REDIRECT pheromone can override a conflicting instinct at runtime
- When instinct and pheromone conflict, highest confidence signal wins regardless of source
- Instincts are colony-scoped — they do NOT survive seal/entomb. Cross-colony persistence is QUEEN.md's job (Phase 5)

### Claude's Discretion
- Confidence threshold for instinct creation (0.5-0.9 range)
- Maximum instincts per phase (recommended 3-5 but not hard-capped)
- Instinct verbosity in builder prompts
- Prompt budget allocation between instincts, pheromones, and other context
- Whether instincts decay over time or remain at created confidence

</decisions>

<specifics>
## Specific Ideas

- Duplicate instinct detection should strengthen existing instincts rather than creating parallel entries — the system should get more confident, not more verbose
- Build output should show instincts being applied so the user feels the colony is actually learning ("3 instincts applied" plus the list)
- The pheromone override mechanism means users always have a kill switch — they can REDIRECT away from any instinct they disagree with

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-instinct-pipeline*
*Context gathered: 2026-03-06*
