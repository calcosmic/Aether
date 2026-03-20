# Phase 26: File Audit - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Audit every file in the repo and remove dead weight. Focus on `.aether/` root, `docs/`, `.planning/phases/`, `.opencode/`, `.claude/`, and the repo root. The goal is a lean, clean repo that's understandable at a glance — no debugging artifacts, no stale planning docs, no dead duplicates.

</domain>

<decisions>
## Implementation Decisions

### Deletion aggressiveness
- Aggressive approach — delete everything not actively used for the colony to run or for current development
- Old planning docs (design plans, handoff docs, implementation plans from v1.0-v1.3) should be deleted outright — git history preserves them
- `.aether/docs/` — Claude's discretion on which docs serve a clear purpose (user-facing or dev); delete the rest

### Archive vs delete policy
- Colony-related artifacts go to `.aether/archive/` before deletion — safety net beyond git history
- Truly dead files (empty dirs, debugging artifacts, dated handoffs) are deleted outright — no archive
- `.aether/archive/` keeps its existing content (old model-routing research) and receives new archived items

### Borderline file handling
- `.planning/phases/` — delete v1.0-v1.2 phase directories, keep v1.3 and v1.4 phase directories
- `TO-DOS.md` — clean it up, remove completed/obsolete items, keep only what's still relevant
- Audit covers EVERYTHING: `.aether/`, `.claude/`, `.opencode/`, `docs/`, repo root
- Repo root files audited too — if it's not serving a purpose, flag it

### Safety verification
- Run full test suite (446 tests) AND `npm pack --dry-run` after deletions
- Small batches — one commit per logical category of deletions (easier to revert)
- Spot-check 3-5 key slash commands after cleanup to verify they still work
- If deletion breaks something: Claude decides per-case whether to fix the reference or revert

### Claude's Discretion
- Which `.aether/docs/` files serve a clear purpose vs should be deleted
- Per-file decisions in `docs/` directory (audit each, delete what's clearly dead)
- Which root-level files to flag for removal
- Grouping of deletions into logical commit categories
- Per-case judgment when deletions break references

</decisions>

<specifics>
## Specific Ideas

- "Aggressive" is the operative word — lean and clean over comprehensive preservation
- Git history is trusted as the long-term record — files don't need to live in the repo to be recoverable
- Colony artifacts (dreams, ceremonies, state) remain precious and untouched
- The `.aether/archive/` folder is the one concession to safety beyond git — only for colony-related items

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 26-file-audit*
*Context gathered: 2026-02-20*
