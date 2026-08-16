# Re-init boundary — what a new colony clears, what it preserves

**Decision date:** 2026-08-16
**Status:** settled — locked by `TestInitPreservesWisdomClearsResidue`
(both directions) alongside `TestInitClearsPriorColonyDecisionResidue`
(RUNTIME-01, the clearing direction).

Running `aether init` over a previous colony draws one line:

**CLEARED — per-colony working residue.** Anything that describes the
*previous conversation and its workers*, because it leaks into the new
colony's worker prompts as if it were their own context (RUNTIME-01):
- `session.json` (recreated fresh)
- `pending-decisions.json` (rendered into prompts as CLARIFIED INTENT)
- `assumptions.json`
- `handoffs/` (rendered as Previous Worker Handoffs)
- `reviews/`, `worktrees/` (working debris)

**PRESERVED — cross-colony wisdom.** Anything that describes *what the system
has learned*, because learning is supposed to outlive a single colony:
- `pheromones.json` (steering signals decay on their own schedule)
- `instincts.json` (standalone instinct store)
- `learning-observations.json`
- `midden/` (the failure log exists precisely so lessons survive)
- `QUEEN.md`, hive and eternal memory (hub-level, untouched by init)
- the prior `COLONY_STATE.json`, kept as a **mandatory** backup under
  `backups/COLONY_STATE.pre-init.<timestamp>.bak` — init refuses to run if
  the backup cannot be written

SEE-11's original wording ("re-init preserves all colony state") and
RUNTIME-01 ("re-init clears leaking residue") are not in conflict — they name
the two sides of this line. Any future change that moves a file across the
line must update both tests and this document.
