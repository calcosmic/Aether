# Phase 34: Cleanup - Research

**Researched:** 2026-04-23
**Domain:** Git worktree/branch cleanup, colony data maintenance, commit preservation
**Confidence:** HIGH

## Summary

This phase cleans up accumulated debris from prior colony work: 522 registered git worktrees (59 GB on disk), 523 branches (mostly `feature/test-audit-*`), and 18 unresolved blocker flags in `pending-decisions.json`. Two candidate commits from worktree branches need evaluation before deletion. The existing cleanup code (`gcOrphanedWorktrees`, `worktreeCleanupCmd`) handles single-branch cleanup but cannot handle bulk operations -- the plan must script the bulk cleanup using git CLI commands directly.

**Primary recommendation:** Execute cleanup in three ordered stages: (1) evaluate and preserve/integrate the 2 candidate commits, (2) bulk-remove worktrees and branches via scripted git commands with interactive confirmation, (3) interactive blocker flag review. Use Phase 26's backup pattern for `.aether/data/` before any modifications.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Before any deletion, review the 2 candidate commits for value, merge-readiness, and safety.
- **D-02:** If a commit passes all three checks (valuable + ready + safe), integrate into `main` via selective porting or cherry-pick -- never wholesale merge.
- **D-03:** If a commit does not pass all three checks, create a `preserve/` branch pointing to its SHA, then proceed with cleanup.
- **D-04:** Everything else among the ~522 worktree entries is disposable and can be removed.
- **D-05:** Interactive confirmation required before any destructive action. Show complete list of what will be deleted, pause for explicit user confirmation.
- **D-06:** No `--force` flag bypasses confirmation. The only way to proceed is explicit user approval.
- **D-07:** Back up `.aether/data/` before any modifications (reuse Phase 26 backup pattern).
- **D-08:** Manual review for all 13+ unresolved blocker flags. Present each flag with age, severity, description, and source phase.
- **D-09:** For each flag, user chooses: keep active, archive, or resolve.
- **D-10:** No auto-archive by age. Every flag requires an explicit decision.

### Claude's Discretion
- Specific porting strategy for each commit (which files to cherry-pick, which to skip)
- Exact order of cleanup operations (worktrees first, then branches, then blockers)
- Output formatting for the interactive review screens

### Deferred Ideas (OUT OF SCOPE)
- Automated recurring cleanup -- run cleanup automatically during `/ant:init` or on a schedule.
- Cross-repo worktree cleanup -- clean up worktrees in other repos using Aether.
- Visual cleanup dashboard -- a `/ant:watch` view showing cleanup status and history.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| R056 | Stale worktrees (~520 entries, ~59 GB) distort the system | Worktree inventory and cleanup strategy below |
| R057 | Stale test-audit branches (523 total branches) | Branch inventory and cleanup strategy below |
| R058 | Unresolved blocker flags (18 found) | Flag system analysis and interactive review strategy below |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Commit evaluation | Git CLI + manual review | -- | Requires human judgment for code value assessment |
| Worktree removal | Git CLI (scripted) | -- | `git worktree remove` is the only reliable path |
| Branch deletion | Git CLI (scripted) | -- | `git branch -D` after worktrees are cleared |
| Flag review | Go CLI (flag-resolve) | Manual review | Existing `flag-resolve` subcommand handles resolution |
| Data backup | File system (copyFile) | -- | Phase 26 pattern: copy to `.aether/data/backups/` |
| Preservation branches | Git CLI | -- | `git branch preserve/<name> <SHA>` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| git (CLI) | system | Worktree/branch operations | Only reliable tool for bulk git worktree operations |
| Go stdlib | 1.26.1 | Scripting, file ops, JSON | Project language, no additional dependencies needed |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| cobra | (existing) | CLI flags for any new subcommands | If extending existing cleanup commands |
| colony package | (existing) | FlagEntry, FlagsFile types | For reading/writing pending-decisions.json |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Scripted git commands | Extend `worktreeCleanupCmd` in Go | Go extension is overkill for a one-time bulk operation; scripted git is simpler and more auditable |
| New `cleanup` subcommand | Manual git commands | New subcommand adds permanent code for a one-time task; manual is fine with confirmation |

**Installation:** No new packages needed. All tools already available.

## Architecture Patterns

### System Architecture Diagram

```
.aether/data/                    Git Worktrees/                    Git Branches/
(Colony State)                   (522 entries)                     (523 branches)
     |                                |                                 |
     v                                v                                 v
[1. BACKUP] ──────────────────────────────────────────────────────────────────
     |
     v
[2. COMMIT EVALUATION]
     |
     +---> Commit 1: claude-dispatch-ux-20260421-1
     |         |
     |         +---> Passes checks? ──> Selective port to main
     |         +---> Fails checks? ──> preserve/ branch ──> continue cleanup
     |
     +---> Commit 2: 4bbb9273 (intent-workflows)
               |
               +---> Passes checks? ──> Selective port to main
               +---> Fails checks? ──> preserve/ branch ──> continue cleanup
     |
     v
[3. WORKTREE REMOVAL] (interactive confirmation)
     |
     +---> 265 prunable worktrees (git worktree remove)
     +---> 257 non-prunable worktrees (git worktree remove --force)
     +---> Nested worktrees in Aether-worktrees/
     +---> /tmp/ worktrees
     +---> .claude/worktrees/ worktrees
     |
     v
[4. BRANCH DELETION] (after all worktrees removed)
     |
     +---> 520 feature/test-audit-* branches (bulk -D)
     +---> feature/auth (no unique commits, safe delete)
     +---> phase-1/builder-1, phase-2/builder-1 (stale colony branches)
     +---> claude-dispatch-ux-20260421-1 (unless used for preserve/)
     |
     v
[5. FLAG REVIEW] (interactive per-flag)
     |
     +---> Present each of 18 unresolved flags
     +---> User decides: keep / archive / resolve
     +---> Write decisions to pending-decisions.json
     |
     v
[6. VERIFICATION]
     |
     +---> git worktree list (should show only main)
     +---> git branch --list (should show only main + preserve/*)
     +---> go test ./... (nothing broken)
     +---> du -sh cmd/.aether/worktrees/ (should be ~0)
```

### Recommended Project Structure

No new files needed. This phase operates entirely through:
- Existing Go commands in `cmd/` (for flag operations)
- Git CLI commands (for worktree/branch cleanup)
- Direct file operations on `.aether/data/` (for backups)

### Pattern 1: Read-First, Confirm-Then-Act (Medic Pattern from Phase 25-26)
**What:** Scan and report everything before making any changes. Present a summary. Require explicit confirmation.
**When to use:** Every destructive operation in this phase.
**Example:**
```
# Step 1: Scan (read-only)
git worktree list
git branch --list 'feature/test-audit-*'
# Step 2: Present summary to user
# Step 3: User confirms
# Step 4: Execute removals
```

### Pattern 2: Backup Before Mutate (Phase 26 Pattern)
**What:** Copy files to `.aether/data/backups/cleanup-{timestamp}/` before modifying.
**When to use:** Before modifying `pending-decisions.json` or any `.aether/data/` file.
**Example:**
```go
// Source: cmd/init_cmd.go:100-108, cmd/install_cmd.go:487+
backupDir := filepath.Join(dataDir, "backups", fmt.Sprintf("cleanup-%s", time.Now().Format("20060102-150405")))
os.MkdirAll(backupDir, 0755)
copyFile(srcPath, filepath.Join(backupDir, filename))
```

### Anti-Patterns to Avoid
- **Blind bulk delete without verification:** Running `git worktree remove` on all entries without first checking what exists. Always list first, verify, then confirm.
- **Deleting worktrees before evaluating commits:** The two candidate commits live on worktree branches. Evaluating them must happen before the branches are deleted.
- **Modifying pending-decisions.json without backup:** Flags represent important project context. Backup first.
- **Using `gcOrphanedWorktrees` for bulk cleanup:** It only handles entries tracked in `COLONY_STATE.json` Worktrees array, which may not cover all 522 registered git worktrees.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Worktree removal | Custom Go worktree scanner | `git worktree list --porcelain` + `git worktree remove` | Git knows its own worktree state; custom scanning misses edge cases |
| Branch deletion | Custom branch listing | `git branch --list` + `git branch -D` | Git handles reflog, packed-refs, and other internals |
| Flag resolution | Custom JSON mutation | Existing `flag-resolve` subcommand or direct `store.SaveJSON` | The flag_cmds.go pattern already handles load/save/resolve correctly |

**Key insight:** This is a one-time housekeeping operation. Resist the urge to build a reusable bulk-cleanup command. Script it, run it, verify it, move on.

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `pending-decisions.json`: 27 flags (18 unresolved) | Interactive review, backup before modification |
| Live service config | No external services affected | None |
| OS-registered state | Git worktrees registered in `.git/worktrees/` | `git worktree remove` cleans up both directory and git registration |
| Secrets/env vars | None affected | None |
| Build artifacts | 252 worktree directories in `cmd/.aether/worktrees/` (59 GB) | Remove directories via git worktree removal; `git worktree prune` for stragglers |

**Nothing found in category:** No external services, no secrets, no installed packages affected.

## Common Pitfalls

### Pitfall 1: Nested Worktrees Inside Worktrees
**What goes wrong:** The `claude-dispatch-ux-20260421-1` worktree contains 4 nested worktrees inside it. Removing the parent worktree fails if child worktrees still exist.
**Why it happens:** Worktrees can create their own worktrees recursively.
**How to avoid:** Remove worktrees bottom-up: child worktrees first, then parent. For the dispatch-ux worktree at `/Users/callumcowie/repos/Aether-worktrees/claude-dispatch-ux-20260421-1`, remove its 4 nested `feature/test-audit-*` worktrees first.
**Warning signs:** `git worktree remove` returns "not a working tree" or reports dependent worktrees.

### Pitfall 2: Prunable vs Non-Prunable Distinction
**What goes wrong:** 265 worktrees are marked "prunable" (directory gone but git reference remains), while 257 are non-prunable (directory still exists). Different removal strategies needed.
**Why it happens:** Some worktree directories were manually deleted without running `git worktree prune`.
**How to avoid:** Run `git worktree prune` first to clean up prunable entries. Then remove remaining (non-prunable) worktrees with `git worktree remove`.
**Warning signs:** `git worktree remove` fails on prunable entries (directory already gone).

### Pitfall 3: Cannot Delete Branch With Active Worktree
**What goes wrong:** `git branch -D feature/test-audit-1234` fails because a worktree is still associated with that branch.
**Why it happens:** Git prevents branch deletion while the branch is checked out in any worktree.
**How to avoid:** Always remove worktrees before deleting branches. The order matters: worktrees first, branches second.
**Warning signs:** Error "error: The branch 'X' is not fully merged" or "branch X is checked out by worktree".

### Pitfall 4: Candidate Commit Conflicts With Main
**What goes wrong:** Cherry-picking a candidate commit fails because main has evolved since the branch was created.
**Why it happens:** Both candidate commits have significant conflicts with current main (9 and 16 files changed in both). Phase 31-33 made substantial changes to the same files.
**How to avoid:** Do not attempt wholesale cherry-pick. Instead, assess each file individually. For the dispatch-ux commit, the `codex_build_progress.go` changes add richer output that may still be valuable. For the intent-workflows commit, most features (discuss.go, assumptions.go) already exist on main in a different form.
**Warning signs:** Cherry-pick conflicts in `cmd/codex_build_*.go`, `cmd/codex_continue.go`, `cmd/codex_visuals.go`.

### Pitfall 5: Incomplete Worktree Cleanup Leaves Phantom Directories
**What goes wrong:** After removing worktrees via `git worktree remove`, the parent directory `cmd/.aether/worktrees/` still exists with empty subdirectories.
**Why it happens:** Git removes the worktree content but may leave empty parent directories.
**How to avoid:** After all worktree removals, run `find cmd/.aether/worktrees/ -type d -empty -delete` and verify with `du -sh cmd/.aether/worktrees/`.
**Warning signs:** `ls cmd/.aether/worktrees/` shows empty directories after cleanup.

## Code Examples

### Bulk Worktree Prune and Remove
```bash
# Step 1: Prune stale (prunable) worktrees
git worktree prune

# Step 2: List remaining worktrees (excluding main)
git worktree list --porcelain | grep -A2 "^worktree" | grep -v "$(git rev-parse --show-toplevel)$"

# Step 3: Remove each remaining non-main worktree
# (with interactive confirmation first)
git worktree list | grep -v "main" | while read path sha branch; do
  git worktree remove "$path" --force
done

# Step 4: Final prune
git worktree prune
```

### Candidate Commit Assessment
```bash
# Check if commit's changes are already on main
git diff main...98cda871 --stat

# Check for conflict potential
git merge-tree $(git merge-base main 98cda871) main 98cda871 | grep "changed in both"

# Create preserve branch if not integrating
git branch preserve/claude-dispatch-ux 98cda871
```

### Flag Review Pattern
```go
// Source: cmd/flag_cmds.go -- existing pattern for flag resolution
var ff colony.FlagsFile
store.LoadJSON("pending-decisions.json", &ff)

// Present flag to user, get decision
for _, f := range ff.Decisions {
    if !f.Resolved {
        // Display: f.ID, f.Type, f.Description, f.CreatedAt, f.Phase
        // User decides: keep / archive / resolve
    }
}

store.SaveJSON("pending-decisions.json", ff)
```

### Backup Pattern (from Phase 26)
```go
// Source: cmd/init_cmd.go:100-108
backupDir := filepath.Join(dataDir, "backups", fmt.Sprintf("cleanup-%s", time.Now().Format("20060102-150405")))
os.MkdirAll(backupDir, 0755)
copyFile(srcPath, filepath.Join(backupDir, filename))
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `gcOrphanedWorktrees()` in init_cmd.go | Still used at init time | Original | Only handles COLONY_STATE.json-tracked worktrees, not raw git worktrees |
| `worktreeCleanupCmd` in clash.go | Single-branch cleanup | Original | Requires `--branch` flag, no bulk mode |
| `flag-auto-resolve` by age | Disabled for this phase (D-10) | Phase 34 decision | Each flag requires manual review |

**Deprecated/outdated:**
- The 522 worktree entries are from colony work that used in-repo worktree isolation extensively during v1.3-v1.4 development. The current parallel_mode for this colony is `in-repo`, which doesn't create these worktrees.

## Existing Code Analysis

### `gcOrphanedWorktrees()` (cmd/codex_build_worktree.go:732-767)
- Scans `COLONY_STATE.json` Worktrees array only
- Filters by status (Allocated, InProgress, Orphaned)
- Attempts to remove each via `removeGitWorktree()`
- Updates state file after cleanup
- **Limitation:** Does NOT scan raw git worktrees. Only knows about worktrees tracked in colony state. Most of the 522 worktrees are raw git worktrees not in COLONY_STATE.json.
- **Verdict:** Cannot be used for bulk cleanup. Scripted git commands needed.

### `worktreeCleanupCmd` (cmd/clash.go:149-185)
- Takes a single `--branch` flag
- Runs `git worktree remove <branch> --force`
- Runs `git worktree prune` after
- **Limitation:** Single branch only. No bulk mode. No summary/confirmation.
- **Verdict:** Useful for individual cleanup but needs scripting wrapper for bulk.

### `copyFile()` (cmd/install_cmd.go:487+)
- Simple file copy preserving permissions
- Already used in init_cmd.go for backups
- **Verdict:** Reuse directly for backup pattern.

### Flag System (cmd/flag_cmds.go)
- `FlagEntry` struct: ID, Type, Description, Phase, Source, CreatedAt, Resolved, ResolvedAt, Resolution, Acknowledged
- `FlagsFile` struct: Version + Decisions array
- Stored in `pending-decisions.json` (falls back to `flags.json`)
- `flag-resolve --id <id> --message <msg>` marks a flag resolved
- `flag-check-blockers` counts active blockers/issues/notes
- `flag-auto-resolve --max-days 7` auto-resolves by age (NOT used per D-10)
- **Current state:** 27 total flags, 18 unresolved (more than the 13 initially estimated -- flag count has grown)

### Candidate Commit 1: `claude-dispatch-ux-20260421-1` (98cda871)
- **Files:** 10 files, 377 insertions, 40 deletions
- **What it adds:** Richer dispatch progress output in Codex runtime
  - `emitCodexDispatchWaveProgress()` with wave execution plan details
  - `emitCodexDispatchWorkerStarted()` for worker lifecycle events
  - `emitCodexDispatchWorkerRunning()` for running state display
  - Wave strategy display (serial vs parallel, with reasons)
  - Parallel mode upgrade hint for serial worktree builds
- **Conflict status:** ALL 10 files have been modified on main since branch point. 9 files show as "changed in both".
- **Main already has:** `codex_build_progress.go` (203 lines, vs 164 in commit). Main has a simpler version of the same file.
- **Assessment:** The richer output features (worker-started/running events, wave execution plans) are likely still valuable and NOT present on main. However, every file conflicts. Selective porting would require careful diffing.
- **Recommendation:** Create `preserve/claude-dispatch-ux` branch. The features can be ported incrementally in a future phase when Codex UX is a priority.

### Candidate Commit 2: `4bbb9273` (intent-workflows)
- **Files:** 33 files, 2878 insertions, 112 deletions
- **What it adds:** discuss command, assumptions command, pheromone sync, Codex plan improvements
- **Main already has:** `cmd/discuss.go`, `cmd/assumptions.go`, `pkg/colony/assumptions.go` -- all present on main in different forms
- **Conflict status:** 16 files changed in both.
- **Assessment:** High overlap with what's already landed on main. The commit was a large patch set where most features have since been implemented separately. Mining selectively is high effort for low return.
- **Recommendation:** Create `preserve/intent-workflows` branch. Most valuable code already on main.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Most features from commit 2 (intent-workflows) are already on main in different forms | Candidate Commit 2 | If wrong, valuable code is lost. Mitigated by preserve/ branch. |
| A2 | The dispatch-ux progress features (worker-started/running, wave plans) are NOT on main | Candidate Commit 1 | If wrong, preserve/ branch is harmless. |
| A3 | The feature/auth, phase-1/builder-1, and phase-2/builder-1 branches have no unique commits worth preserving | Branch Inventory | If wrong, code is lost. feature/auth has 0 unique commits, phase branches appear empty. |
| A4 | `gcOrphanedWorktrees()` cannot handle bulk cleanup because it only reads COLONY_STATE.json Worktrees array | Existing Code Analysis | If wrong, we build unnecessary scripting. Verified by reading the code. |

## Open Questions

1. **Should the dispatch-ux commit's progress features be ported now or deferred?**
   - What we know: The features are valuable but all 10 files conflict with main. Porting requires careful manual diffing.
   - What's unclear: Whether the Codex UX improvements are needed for v1.0.20 or can wait.
   - Recommendation: Create preserve/ branch now, port in a future phase focused on Codex UX.

2. **What about the 4 nested worktrees inside the dispatch-ux worktree?**
   - What we know: They are `feature/test-audit-*` branches at `/Users/callumcowie/repos/Aether-worktrees/claude-dispatch-ux-20260421-1/cmd/.aether/worktrees/`.
   - What's unclear: Whether they contain anything beyond the dispatch-ux commit itself (they all point to same SHA `48a32c3a`).
   - Recommendation: These are disposable -- they're test-audit worktrees created during the dispatch-ux work. Remove them as part of bulk cleanup.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| git | Worktree/branch operations | YES | system | -- |
| Go 1.26+ | Build verification | YES | 1.26.1 darwin/arm64 | -- |
| `aether` CLI | Flag operations | YES | built from source | -- |

**Missing dependencies with no fallback:** None -- all tools available.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing |
| Config file | None (standard go test) |
| Quick run command | `go test ./... -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | Notes |
|--------|----------|-----------|-------------------|-------|
| R056 | Stale worktrees removed | Manual verification | `git worktree list` after cleanup | Verify count is 1 (main only) |
| R057 | Stale branches removed | Manual verification | `git branch --list` after cleanup | Verify only main + preserve/* remain |
| R058 | Blocker flags reviewed | Manual + automated | `aether flag-check-blockers` after review | Verify blocker count matches decisions |
| -- | No regressions | Automated | `go test ./... -race -count=1` | Must pass clean |
| -- | Disk space recovered | Manual verification | `du -sh cmd/.aether/worktrees/` | Should be ~0 |

### Sampling Rate
- **After each stage:** `go test ./... -count=1` (quick check, ~30s)
- **Phase gate:** `go test ./... -race -count=1` (full suite)

### Wave 0 Gaps
None -- existing test infrastructure covers the regression check. The cleanup itself is tested by verification commands, not unit tests.

## Sources

### Primary (HIGH confidence)
- `cmd/clash.go` -- Read directly, worktreeCleanupCmd implementation [VERIFIED: codebase]
- `cmd/codex_build_worktree.go` -- Read directly, gcOrphanedWorktrees implementation [VERIFIED: codebase]
- `cmd/flag_cmds.go` -- Read directly, flag system implementation [VERIFIED: codebase]
- `pkg/colony/flags.go` -- Read directly, FlagEntry/FlagsFile types [VERIFIED: codebase]
- `cmd/init_cmd.go` -- Read directly, copyFile and backup pattern [VERIFIED: codebase]

### Secondary (MEDIUM confidence)
- `git worktree list` output -- Verified live state of 522 registered worktrees [VERIFIED: live execution]
- `git branch --list` output -- Verified live state of 523 branches [VERIFIED: live execution]
- `pending-decisions.json` content -- Verified 27 flags, 18 unresolved [VERIFIED: live execution]
- `git diff` and `git merge-tree` output for conflict assessment [VERIFIED: live execution]

### Tertiary (LOW confidence)
- None -- all findings verified against live codebase state.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all tools verified on machine
- Architecture: HIGH -- existing patterns well understood, codebase read directly
- Pitfalls: HIGH -- identified from live git state and code analysis
- Candidate commits: HIGH -- conflict analysis run against live main branch

**Research date:** 2026-04-23
**Valid until:** 2026-05-23 (stable -- git state may change but patterns are constant)
