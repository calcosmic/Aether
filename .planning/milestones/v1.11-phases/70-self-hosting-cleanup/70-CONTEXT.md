# Phase 70: Self-Hosting Cleanup - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Remove all stale artifacts that exist because Aether was used to develop itself, leaving a clean canonical repo. This covers agents, chambers, runtime state files, orphaned worktree artifacts, and gitignore hardening.

</domain>

<decisions>
## Implementation Decisions

### Artifact Scope
- **D-01:** Include orphaned worktree files (7 files in `.claude/worktrees/agent-a9135902/`) in addition to requirements-specified artifacts. These are stale Phase 44 artifacts whose canonical copies already exist in `.planning/phases/`.
- **D-02:** Total cleanup scope: 26 stale agent files + 241 chamber files + 2 runtime state files + 7 worktree artifacts = 276 files removed from git tracking.

### Chamber Safety
- **D-03:** Perform a quick sample check (spot-check 3-5 chambers) before deletion to confirm no irreplaceable data exists. All chambers are from March 2026; active colony data lives in `.aether/data/COLONY_STATE.json` (already gitignored).
- **D-04:** If sample check finds nothing irreplaceable, proceed with `git rm -r` for all 241 chamber files. If something is found, flag it before proceeding.

### Gitignore Coverage
- **D-05:** Expand `.aether/.gitignore` beyond just `chambers/` — also add `agents/` and any other self-hosting directories that should never be tracked. Prevents future self-hosting leaks.
- **D-06:** Current gitignore covers: `data/`, `dreams/`, `checkpoints/`, `locks/`. After update: also `chambers/`, `agents/`, and other leak vectors identified during planning.

### Commit Strategy
- **D-07:** Single commit for all cleanup (artifact removal + gitignore update). Easy to review and revert.

### Verification
- **D-08:** After cleanup, verify `agents-claude/` remains byte-identical to `.claude/agents/ant/` (already confirmed pre-cleanup: all 26 files match MD5 hashes).
- **D-09:** Run `go test ./...` after cleanup to confirm nothing breaks.

### Claude's Discretion
- Exact gitignore entries beyond chambers/ and agents/
- How to structure the sample check (which chambers, what to look for)
- Whether to also clean up local untracked chamber directories (chamber-alpha, chamber-beta) that aren't in git

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — CLEAN-01 through CLEAN-05 define the five cleanup requirements
- `.planning/ROADMAP.md` §Phase 70 — Phase goal, success criteria, and dependency chain

### Prior Cleanup
- `.planning/phases/34-cleanup/34-CONTEXT.md` — Prior cleanup phase; established `git rm` patterns for artifact removal

### Git Configuration
- `.aether/.gitignore` — Current gitignore (needs updating per CLEAN-04 and D-05)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `git ls-files` commands from scout phase identify exactly which files are tracked vs local-only
- MD5 comparison confirmed agents-claude/ and .claude/agents/ant/ are already byte-identical (CLEAN-05 pre-verified)

### Established Patterns
- Prior cleanup phases (Phase 34) used `git rm` for artifact removal — follow same pattern
- `.aether/.gitignore` uses simple directory-level ignores (e.g., `data/`)

### Integration Points
- `.aether/agents/` — 26 files that duplicate `agents-claude/` and `.claude/agents/ant/`. Safe to remove entirely.
- `.aether/chambers/` — 241 tracked files, all from March 2026 sessions. Local-only dirs (chamber-alpha, chamber-beta) not tracked.
- `.aether/CONTEXT.md` and `.aether/CROWNED-ANTHILL.md` — 2 runtime state files that should never have been tracked.
- `.claude/worktrees/agent-a9135902/` — 7 orphaned files from Phase 44 agent worktree.

### File Count Summary
| Artifact | Tracked in git | On disk |
|----------|---------------|---------|
| `.aether/agents/` | 26 | 26 |
| `.aether/chambers/` | 241 | 288 (47 local-only) |
| `.aether/CONTEXT.md` + `CROWNED-ANTHILL.md` | 2 | 2 |
| `.claude/worktrees/` | 7 | 7 |
| **Total to remove from git** | **276** | — |

</code_context>

<specifics>
## Specific Ideas

No specific requirements — standard cleanup approach. User emphasized safety verification (sample check before chamber deletion) and broad gitignore coverage to prevent future leaks.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 70-self-hosting-cleanup*
*Context gathered: 2026-04-28*
