## 1. Current Git Touchpoints in Aether

### Complete Inventory

| # | File | Line(s) | Git Operation | Type | Purpose | Notes |
|---|------|---------|---------------|------|---------|-------|
| 1 | `.claude/commands/ant/build.md` | 82 | `git rev-parse --git-dir 2>/dev/null` | R | Detect if cwd is a git repo | Gate for checkpoint; gracefully skips if not a repo |
| 2 | `.claude/commands/ant/build.md` | 85 | `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"` | **W** | **Pre-phase safety checkpoint** | Stages ALL files unconditionally; `--allow-empty` ensures commit even with no changes; tagged with phase number |
| 3 | `.claude/commands/ant/continue.md` | 43 | _(reference)_ rollback to git checkpoint | **W** | Offer rollback on stale EXECUTING state | Not a direct git command; refers to using the checkpoint created by build.md; mechanism unspecified |
| 4 | `.claude/commands/ant/continue.md` | 119 | `git diff --stat` | R | Review changed files after phase execution | Part of Phase 6 "Diff Review"; used to detect unintended modifications |
| 5 | `.claude/commands/ant/dream.md` | 42 | `git log --oneline -30` | R | Context gathering: see recent project evolution | Part of "Codebase awareness" during dreamer awakening |
| 6 | `.claude/commands/ant/dream.md` | 43 | `git diff --stat HEAD~10..HEAD 2>/dev/null` | R | Context gathering: see which areas are actively changing | Suppresses errors if fewer than 10 commits exist |
| 7 | `.claude/commands/ant/dream.md` | 83 | _(reference)_ "A file you noticed in the git log" | R | Dreamer wandering direction hint | Not a command; suggests using git log output to pick exploration targets |
| 8 | `.claude/commands/ant/swarm.md` | 59 | `bash ~/.aether/aether-utils.sh autofix-checkpoint` | **W** | Create stash-based checkpoint before swarm investigation | Delegates to aether-utils.sh; creates git stash or records HEAD hash |
| 9 | `.claude/commands/ant/swarm.md` | 84 | `git log --oneline -20 2>/dev/null` | R | Context: scan recent commits before deploying scouts | Part of Step 4 context reading |
| 10 | `.claude/commands/ant/swarm.md` | 100 | `git log --oneline -30` | R | Scout instruction: Git Archaeologist sees recent commits | Inside spawned Scout 1 task prompt |
| 11 | `.claude/commands/ant/swarm.md` | 101 | `git log -p --since="1 week ago" -- {relevant files}` | R | Scout instruction: see recent changes to specific files | Patch-level detail for bug investigation |
| 12 | `.claude/commands/ant/swarm.md` | 102 | `git blame {suspected file}` | R | Scout instruction: trace line-level authorship | Conditional; only if a specific file is suspected |
| 13 | `.claude/commands/ant/swarm.md` | 308 | `bash ~/.aether/aether-utils.sh autofix-rollback "{checkpoint_type}" "{checkpoint_ref}"` | **W** | Rollback failed autofix to pre-swarm state | Delegates to aether-utils.sh; pops stash or does `git reset --hard` |
| 14 | `.aether/aether-utils.sh` | 790 | `git rev-parse --git-dir` | R | Detect git repo (autofix-checkpoint) | Gate for all checkpoint logic |
| 15 | `.aether/aether-utils.sh` | 792 | `git status --porcelain` | R | Check for uncommitted changes (autofix-checkpoint) | Determines if stash is needed |
| 16 | `.aether/aether-utils.sh` | 794 | `git stash push -m "$stash_name"` | **W** | Create named stash as checkpoint | Name format: `aether-autofix-{epoch}` |
| 17 | `.aether/aether-utils.sh` | 798, 803 | `git rev-parse HEAD` | R | Record current HEAD as fallback ref | Used when stash fails or working dir is clean |
| 18 | `.aether/aether-utils.sh` | 821 | `git stash list` | R | Find stash by name for rollback | Grep + cut to extract stash ref from list |
| 19 | `.aether/aether-utils.sh` | 823 | `git stash pop "$stash_ref"` | **W** | Restore stashed changes on rollback | Removes stash after applying |
| 20 | `.aether/aether-utils.sh` | 835 | `git reset --hard "$ref"` | **W** | Hard reset to checkpoint commit on rollback | Nuclear option; discards all working changes |

### Mirror Files (.opencode/)

All `.opencode/commands/ant/*.md` files are exact mirrors of their `.claude/commands/ant/*.md` counterparts. The following files contain identical git operations at identical line numbers:

- `.opencode/commands/ant/build.md` -- lines 82, 85 (same as #1, #2 above)
- `.opencode/commands/ant/continue.md` -- lines 43, 119 (same as #3, #4 above)
- `.opencode/commands/ant/dream.md` -- lines 42, 43, 83 (same as #5, #6, #7 above)
- `.opencode/commands/ant/swarm.md` -- lines 59, 84, 100, 101, 102, 308 (same as #8-#13 above)

### Files with NO Git Operations

The following command files were checked and contain zero git operations:

- `.claude/commands/ant/phase.md`
- `.claude/commands/ant/plan.md`
- `.claude/commands/ant/status.md`
- `.claude/commands/ant/init.md`
- `.claude/commands/ant/help.md`
- `.claude/commands/ant/council.md`
- `.claude/commands/ant/watch.md`
- `.claude/commands/ant/colonize.md`
- `.claude/commands/ant/focus.md`
- `.claude/commands/ant/feedback.md`
- `.claude/commands/ant/redirect.md`
- `.claude/commands/ant/flag.md`
- `.claude/commands/ant/flags.md`
- `.claude/commands/ant/organize.md`
- `.claude/commands/ant/pause-colony.md`
- `.claude/commands/ant/resume-colony.md`
- `.claude/commands/ant/migrate-state.md`
- `runtime/aether-utils.sh` (no git operations; only `.aether/aether-utils.sh` has them)

### Summary of Findings

**By category:**

| Category | Count | Operations |
|----------|-------|------------|
| Safety checkpoints (write) | 3 | build.md `git add -A && git commit`, swarm.md `autofix-checkpoint`, swarm.md `autofix-rollback` |
| Safety rollback (write) | 2 | aether-utils.sh `git stash pop`, `git reset --hard` |
| Safety checkpoint setup (write) | 1 | aether-utils.sh `git stash push` |
| Progress verification (read) | 1 | continue.md `git diff --stat` |
| Context gathering (read) | 5 | dream.md (2x), swarm.md main (1x), swarm.md scout (3x git log/blame variants) |
| Repo detection (read) | 2 | build.md and aether-utils.sh `git rev-parse --git-dir` |
| State detection (read) | 2 | aether-utils.sh `git status --porcelain`, `git rev-parse HEAD` |

**Key observations:**

1. **Two distinct checkpoint strategies exist:**
   - **build.md** uses `git add -A && git commit` -- a full commit-based checkpoint. This stages and commits everything unconditionally, creating permanent history entries.
   - **swarm.md / aether-utils.sh** uses `git stash push` -- a temporary stash-based checkpoint. This is lighter weight and doesn't pollute commit history, but stashes can be lost.

2. **No staging granularity exists.** The only `git add` in the entire system is `git add -A` (stage everything). There is no selective staging, no `.gitignore` awareness beyond what git itself provides, and no consideration of what files should vs. should not be committed.

3. **The `--allow-empty` flag on checkpoint commits** means a commit is always created even if nothing changed, which creates noise in git history.

4. **The rollback mechanism has a dangerous path:** `git reset --hard` in autofix-rollback discards all working changes. If the checkpoint ref is wrong or stale, this could lose work.

5. **No git push anywhere.** All git operations are local-only. There is no remote interaction (push, pull, fetch) in any Aether command.

6. **continue.md references "rollback to git checkpoint" on line 43** but does not specify the mechanism. It appears to rely on the user or another command to perform the actual rollback.

7. **Read-only operations outnumber writes ~2:1.** Most git usage is for context gathering (log, diff, blame) rather than state mutation.

8. **No branch management.** No command creates branches, switches branches, or considers branch strategy. All operations happen on whatever the current branch is.
