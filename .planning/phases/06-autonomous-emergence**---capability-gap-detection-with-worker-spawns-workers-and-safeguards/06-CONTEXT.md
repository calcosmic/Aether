# Phase 6: Autonomous Emergence - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Worker Ants detect capability gaps and spawn specialists automatically with safeguards against infinite loops. This phase delivers the autonomous spawning mechanism — Worker-spawns-Workers without Queen approval. The colony self-organizes within phases while Queen provides intention at boundaries.

</domain>

<decisions>
## Implementation Decisions

### Capability Gap Detection

**Spawn Triggers (all three apply):**
- **Explicit domain mismatch** — Worker Ant analyzes task and sees required skills outside its caste's domain
- **Failure after attempts** — Worker Ant tries first, spawns if stuck or unclear path forward
- **Pattern recognition** — Worker Ant recognizes patterns that historically required specialists

**Spawn Timing (Hybrid approach):**
- **Known gaps** → Spawn immediately before attempting (e.g., Colonizer Ant sees database task → spawn database_specialist)
- **Ambiguous cases** → Attempt first, spawn if struggling or unclear path forward
- This balances speed (no wasted attempts) with learning (trying builds capability awareness)

**Claude's Discretion:**
- Confidence threshold for spawn decisions (how much evidence before spawning)
- Self-assessment method (capability inventory, boundary definition, or task analysis)
- Exact number of "attempts" before failure-triggered spawn

### Specialist Selection Logic

**Caste Mapping:**
- Keyword/capability mapping table with semantic analysis as fallback (hybrid approach)
- Direct lookup for known patterns (e.g., "database" → database_specialist caste)
- Semantic analysis handles novel/unfamiliar gaps

**Ambiguity Handling:**
- Best-guess single spawn with sequential fallback
- Use heuristics to pick best match, spawn only that one
- If first specialist fails, consider next-best caste

**Spawn Granularity:**
- Balance between fine and coarse based on resource budget
- Adaptive to task complexity (simple = one specialist, complex = more targeted specialists)
- Resource budget (max 10 per phase) prevents over-spawning

**Meta-Learning Integration:**
- **Yes, use confidence scores** — Historical success rates influence caste selection
- Prefer castes with higher confidence for the task type
- Phase 8 will provide deeper Bayesian confidence; Phase 6 sets up basic integration
- This creates a learning loop: successful spawns → higher confidence → more likely to spawn that caste again

### Claude's Discretion
- Exact mapping table structure (keyword → caste mappings)
- Semantic analysis algorithm for novel gaps
- Heuristics for "best match" when multiple castes qualify
- Definition of "task complexity" for adaptive granularity
- How much weight meta-learning confidence has vs. semantic analysis

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for autonomous agent spawning.

Key principle: **Worker-spawns-Workers is core Aether philosophy**. This is what makes Aether unique compared to orchestrated systems. The implementation should feel emergent, not scripted.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 06-autonomous-emergence*
*Context gathered: 2026-02-01*
