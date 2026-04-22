# Phase 34: Cleanup - Context

**Gathered:** 2026-04-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Clean up stale worktrees, orphaned branches, and stale blocker flags accumulated from prior colony work. Preserve any valuable code before deletion. This is a housekeeping phase — no new features, just restoring the repo to a clean state.

**Scope:**
- Review 2 identified worktree commits for integration value
- Remove ~520 disposable worktree entries
- Remove orphaned branches
- Review and resolve/archive 13 unresolved blocker flags

**Out of scope:**
- New cleanup commands (worktree-cleanup already exists in clash.go)
- Cross-repo cleanup
- Automated recurring cleanup (future enhancement)

</domain>

<decisions>
## Implementation Decisions

### Preservation Strategy
- **D-01:** Before any deletion, review the 2 candidate commits identified in the worktree audit for value, merge-readiness, and safety.
- **D-02:** If a commit passes all three checks (valuable + ready + safe), integrate it into `main` during this phase via selective porting or cherry-pick — never wholesale merge.
- **D-03:** If a commit does not pass all three checks, create a `preserve/` branch pointing to its SHA, then proceed with cleanup.
- **D-04:** Everything else among the ~522 worktree entries is disposable and can be removed.

### Cleanup Safety Model
- **D-05:** Interactive confirmation required before any destructive action. The cleanup command must display a complete list of what will be deleted (worktrees, branches, files) and pause for explicit user confirmation.
- **D-06:** No `--force` flag bypasses the confirmation. The only way to proceed is for the user to confirm after reviewing the list.
- **D-07:** Back up `.aether/data/` before any modifications (reuse Phase 26 backup pattern).

### Blocker Flags
- **D-08:** Manual review for all 13 unresolved blocker flags. The cleanup command presents each flag with its age, severity, description, and source phase.
- **D-09:** For each flag, the user chooses: keep active, archive (retain history but remove from active blockers), or resolve (mark as fixed).
- **D-10:** No auto-archive by age. Every flag requires an explicit decision.

### Claude's Discretion
- Specific porting strategy for each commit (which files to cherry-pick, which to skip)
- Exact order of cleanup operations (worktrees first, then branches, then blockers)
- Output formatting for the interactive review screens

</decisions>

<canonical_refs>
## Canonical References

### Worktree Audit
- Worktree preservation audit shared 2026-04-23 — identifies 2 commits worth preserving vs ~520 disposable

### Existing Cleanup Code
- `cmd/clash.go:149-185` — existing `worktree-cleanup` command
- `cmd/init_cmd.go:110-115` — `gcOrphanedWorktrees` call and `RemoveAll(worktrees)`

### Phase 26 Auto-Repair Decisions
- `.planning/phases/26-auto-repair/26-CONTEXT.md` — backup-before-repair, read-only-by-default, trace logging

### Flag System
- `cmd/flag_cmds.go` — flag creation, severity levels, phase association

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/clash.go` — `worktreeCleanupCmd` already exists; can be extended for bulk cleanup
- `cmd/init_cmd.go` — `gcOrphanedWorktrees()` already handles stale worktree detection
- Phase 26 backup logic — `copyFile` to `.aether/backups/medic-{timestamp}/`

### Established Patterns
- Read-only scan first, then explicit repair (Medic pattern from Phase 25-26)
- `--force` flag for destructive operations
- `outputOK` / `outputError` for CLI feedback

### Integration Points
- The cleanup will touch `.aether/data/` (blockers, flags), git worktrees, and git branches
- Should integrate with existing `aether medic` health scan if possible

</code_context>

<specifics>
## Specific Ideas

### Commit 1: `claude-dispatch-ux-20260421-1`
- SHA: `98cda87164c2741df9f777127c201313acdca817`
- Date: 2026-04-21
- Files: `cmd/codex_build.go`, `cmd/codex_build_progress.go`, `cmd/codex_build_test.go`, `cmd/codex_build_worktree.go`, `cmd/codex_continue.go`, `cmd/codex_continue_test.go`, `cmd/codex_visuals.go`, `cmd/status.go`, `pkg/codex/worker.go`, `pkg/codex/worker_test.go`
- 10 files changed, 377 insertions, 40 deletions
- **Assessment needed:** Is live dispatch/progress behavior still missing from current `main`?

### Commit 2: `feature/test-audit-*` duplicate refs
- SHA: `4bbb9273379d31721f7cbdf2466a7189626f3109`
- Date: 2026-04-19
- Files: `cmd/assumptions.go`, `cmd/discuss.go`, `cmd/pheromone_sync.go`, `cmd/codex_plan.go`, `cmd/codex_visuals.go`, `pkg/colony/assumptions.go`, `pkg/codex/worker.go`, plus tests and docs
- 33 files changed, 2878 insertions, 112 deletions
- **Assessment needed:** Treat as patch set to mine selectively, not wholesale merge. Likely overlaps with things already landed differently.

### Safe Preservation Branches (if not integrating)
- `preserve/claude-dispatch-ux` → `98cda871...`
- `preserve/intent-workflows` → `4bbb9273...`

</specifics>

<deferred>
## Deferred Ideas

- **Automated recurring cleanup** — Run cleanup automatically during `/ant:init` or on a schedule. Future enhancement.
- **Cross-repo worktree cleanup** — Clean up worktrees in other repos using Aether. Out of scope for this phase.
- **Visual cleanup dashboard** — A `/ant:watch` view showing cleanup status and history. Nice-to-have, not critical.

</deferred>

---

*Phase: 34-cleanup*
*Context gathered: 2026-04-23*
