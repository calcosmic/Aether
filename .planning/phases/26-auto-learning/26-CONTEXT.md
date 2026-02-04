# Phase 26: Auto-Learning - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Automatically capture phase learnings at the end of build execution (build.md Step 7) so no manual `/ant:continue` is needed for learning extraction. Make continue.md smart enough to detect and skip duplicate extraction. Does NOT change what learnings are or how memory.json is structured — only automates when/how they're captured.

</domain>

<decisions>
## Implementation Decisions

### Learning synthesis
- Capture both errors AND successes — not just failures
- Attribute each learning to the worker/caste that produced it (e.g., "Forager: API retry logic prevents timeouts")
- Always extract learnings, even for clean phases where everything succeeded — "X approach worked well" is still valuable
- Whether to synthesize insights from raw data vs collect/format is Claude's discretion

### Feedback pheromone content
- Balanced summary — equal weight to what worked and what failed
- Always validate via pheromone-validate before writing, no exceptions
- Include the actual learnings in the pheromone body so colony can read them without checking memory.json
- Detail level (brief vs per-worker) is Claude's discretion based on what happened in the phase

### Duplicate detection
- Use an explicit flag mechanism — build.md writes a "learnings extracted" flag after extraction
- continue.md checks this flag; if set, prints "Learnings already captured in build step — skipping" and moves on
- Flag is cleared at the start of each new phase (clean state)
- Whether to support a force/override mechanism for manual re-extraction is Claude's discretion

### Memory limits & retention
- Core intent: the system must continuously get smarter and never lose proven knowledge
- Claude designs the retention strategy — options include tiered maturity (permanent vs rotating), frequency-weighted eviction, or merging similar learnings
- When compression/eviction happens, it must be visible to the user — print what was merged or evicted so nothing valuable is silently lost
- The 20-learning cap is an existing constraint; Claude determines the best way to work within or evolve it to ensure the system keeps learning indefinitely

### Claude's Discretion
- Whether to synthesize insights from raw data or collect/format existing entries
- Detail level of FEEDBACK pheromone (brief vs per-worker breakdown)
- Whether to support force re-extraction in continue.md
- Exact retention/eviction strategy for memory limits (tiers, weighting, merging)
- Whether the 20-learning cap should remain fixed or be adjusted

</decisions>

<specifics>
## Specific Ideas

- "The system should forever evolve and become more sophisticated" — the learning system must never regress or forget hard-won lessons
- Learnings that keep getting reinforced across phases should be treated as more valuable than one-off observations
- The auto-extraction should match the same logic currently in continue.md Step 4, just triggered automatically at end of build

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 26-auto-learning*
*Context gathered: 2026-02-04*
