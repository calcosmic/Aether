# Domain Pitfalls: Multi-Agent CLI Orchestration Systems

**Domain:** AI agent orchestration, CLI-based multi-agent systems
**Researched:** 2026-02-14
**Confidence:** HIGH (based on Aether's real-world issues, codebase analysis, and v1.1 bug fix requirements)

## Overview

Multi-agent CLI orchestration systems like Aether face unique challenges that single-agent systems do not. This document catalogs common pitfalls based on Aether's real-world bugs (race conditions, data loss, update failures) and specific pitfalls to avoid when fixing v1.1 bugs.

---

## Critical Pitfalls for v1.1 Bug Fixes

### 1. Git Stash Captures User Data (CRITICAL)

**What goes wrong:**
The build checkpoint uses `git stash push` to create rollback points. If the path specification is incorrect or if `--include-untracked` is accidentally added, user files outside Aether-managed directories get stashed. When the stash is later popped during rollback, user work is unexpectedly restored, potentially overwriting changes made after the checkpoint.

**Why it happens:**
- Git stash with path arguments is sensitive to working directory state
- The `--include-untracked` flag (if added later) stashes ALL untracked files
- Path globs in git stash can behave unexpectedly with nested directories
- Stash operations don't fail atomically — partial stashes leave repo in inconsistent state

**Prevention:**
1. **Explicit allowlist approach:** Only stash specific directories, never use broad patterns
2. **Verify before stash:** Check `git status --porcelain` output to confirm only expected files are dirty
3. **Never use `--include-untracked`:** This is the primary cause of user data being stashed
4. **Test stash scope:** After stashing, verify with `git stash show -p` that only intended files were captured
5. **Use atomic file operations:** Write checkpoint metadata only after successful stash

**Detection:**
- Stash size is unexpectedly large (check with `git stash list --stat`)
- User reports files "reappearing" after rollback
- Stash contains files outside `.aether/`, `.claude/commands/`, `.opencode/`, `runtime/`, `bin/`
- Rollback operation reports merge conflicts on user files

**Phase mapping:**
- v1.1 Bug Fixes — Checkpoint Data Loss Fix

---

### 2. Phase Advancement Loops (CRITICAL)

**What goes wrong:**
The `/ant:continue` command can enter an infinite loop where it repeatedly advances the phase counter without actually completing work. This happens when state transitions don't properly validate that the previous phase's success criteria were met.

**Why it happens:**
- State machine lacks guard conditions between transitions
- `current_phase` is incremented without verifying phase completion
- Multiple concurrent `/ant:continue` calls race and each increment the counter
- Event log shows phase transitions but no actual work completion evidence

**Prevention:**
1. **Iron Law enforcement:** No phase advancement without fresh verification evidence (already documented in continue.md)
2. **Idempotent transitions:** Check `state != "COMPLETED"` before allowing advancement
3. **Verification gate:** Must pass all 6 verification phases (build, types, lint, test, security, diff) before advancing
4. **Lock during transition:** Acquire state lock before reading/writing phase counter
5. **Audit trail:** Log not just phase change but the evidence that justified it

**Detection:**
- `COLONY_STATE.json` shows phase number increased but phase status still "in_progress"
- Events log shows multiple "phase_completed" events for same phase
- Phase counter jumps by more than 1 between state reads
- No corresponding git commits or file changes for claimed phase completion

**Phase mapping:**
- v1.1 Bug Fixes — Phase Advancement Loop Fix

---

### 3. Update Command Stashes Without Recovery Path (HIGH)

**What goes wrong:**
The `aether update --force` command stashes dirty files to proceed with update, but if the update fails or is interrupted, the stash remains. Users may not know to run `git stash pop`, or worse, may run it at the wrong time and corrupt their working directory.

**Why it happens:**
- `updateRepo()` in cli.js creates stash but doesn't register cleanup handler
- No verification that stash was successfully popped after update
- Users aren't warned that their work is stashed
- Stash message "aether-update-backup" is generic and may be confused with other stashes

**Prevention:**
1. **Always pop on success:** If update succeeds, immediately pop the stash
2. **Warn on failure:** If update fails, prominently display recovery command
3. **Unique stash names:** Include timestamp in stash message for identification
4. **Stash tracking:** Record stash ref in state file for recovery even if CLI crashes
5. **Pre-update warning:** Show what will be stashed before proceeding with --force

**Detection:**
- `git stash list` shows old "aether-update-backup" entries
- User reports files "disappearing" after update
- Update completes but working directory is unexpectedly clean
- Multiple stashes with same message accumulate

**Phase mapping:**
- v1.1 Bug Fixes — Update Command Repair

---

### 4. run_in_background Causes Misleading Output Timing (HIGH)

**What goes wrong:**
When commands are run in background (e.g., spawn operations), their output may interleave with foreground output or appear after subsequent operations complete. This makes it appear that later operations finished before earlier ones, confusing users about execution order.

**Why it happens:**
- Background processes write to stdout/stderr without synchronization
- No buffering or sequencing of output from parallel operations
- Output from background tasks appears after prompt returns
- Race conditions between process completion and output flushing

**Prevention:**
1. **Capture then emit:** Buffer all background output, emit only when task completes
2. **Use TaskOutput with block:** Wait for completion before continuing (already in build.md)
3. **Structured logging:** Write to activity.log instead of stdout for background tasks
4. **Output sequencing:** Tag output with sequence numbers and reorder before display
5. **Avoid background for user-facing ops:** Only use background for true fire-and-forget tasks

**Detection:**
- Output appears after command prompt returns
- Timestamps in logs show out-of-order execution
- User confusion about which task produced which output
- Spawn completion logged before spawn start

**Phase mapping:**
- v1.1 Bug Fixes — Misleading Output Fix

---

### 5. Missing Unit Tests for Core Sync Functions (HIGH)

**What goes wrong:**
The sync functions in `cli.js` (`syncDirWithCleanup`, `syncSystemFilesWithCleanup`) lack comprehensive unit tests. Changes to these functions risk breaking the update mechanism without detection until users report issues.

**Why it happens:**
- Sync functions depend on filesystem state, making them "hard" to test
- Tests require setup/teardown of directory structures
- Hash comparison logic has edge cases (empty files, permission issues)
- Cleanup logic may remove files that should be preserved

**Prevention:**
1. **Test each function in isolation:** Mock filesystem or use temp directories
2. **Property-based testing:** Verify idempotency — running sync twice should be no-op
3. **Edge case coverage:** Empty directories, missing files, permission errors, symlinks
4. **Hash verification tests:** Ensure hash comparison correctly identifies changed files
5. **Cleanup safety tests:** Verify only orphaned files are removed, never preserve-listed files

**Detection:**
- Updates delete user files or fail to update modified system files
- Hash comparison always returns "different" (performance issue)
- Orphaned files accumulate in `.aether/` directories
- Tests pass but real-world updates behave incorrectly

**Phase mapping:**
- v1.1 Bug Fixes — Add Unit Tests

---

### 6. Checkpoint Rollback Loses Work Done After Checkpoint (MEDIUM)

**What goes wrong:**
When rolling back to a checkpoint, the rollback operation (particularly `git reset --hard` for commit-type checkpoints) destroys not just the failed phase's work but also any user commits or changes made after the checkpoint was created.

**Why it happens:**
- `git reset --hard` moves HEAD and discards all changes after target commit
- User may have committed work between checkpoint and rollback
- Stash pop can cause merge conflicts that lose changes
- No distinction between "Aether-managed changes" and "user changes"

**Prevention:**
1. **Prefer stash over reset:** Stash pop is reversible; reset --hard is destructive
2. **Check for user commits:** Before reset, warn if user has made commits after checkpoint
3. **Backup before rollback:** Create a backup branch before any destructive operation
4. **Granular checkpoints:** Create checkpoints more frequently to minimize rollback scope
5. **Interactive rollback:** Show what will be lost and require confirmation

**Detection:**
- User reports commits "disappearing" after rollback
- Git reflog shows unexpected reset operations
- Working directory is clean but user expected changes
- Rollback reports "HEAD is now at..." with unexpected commit hash

**Phase mapping:**
- v1.1 Bug Fixes — Checkpoint Data Loss Fix

---

## General Critical Pitfalls

### 7. Race Conditions in Shared State

**What goes wrong:** Multiple workers or concurrent commands access COLONY_STATE.json simultaneously, causing corruption or lost updates.

**Why it happens:**
- No file locking before read-modify-write operations
- State files are JSON that require full rewrite (not incremental updates)
- Commands spawn parallel workers that all read state before any writes

**Consequences:**
- Partial state writes (truncated JSON)
- Duplicate keys or lost fields
- Collisions when two workers update flags/blockers simultaneously

**Prevention:**
- Implement file locking using `flock` or equivalent before ANY state read/write
- Use atomic writes: write to temp file, then `mv` to target
- Consider read-copy-update (RCU) patterns: read entire state, modify in memory, write atomically

**Detection:**
- JSON parse failures in state files
- Missing fields that should have been added
- Duplicate "status" keys or other duplicated fields

**Phase mapping:**
- Phase 2-3 (Worker spawning): Most likely during parallel worker execution
- Phase 4 (State management): When implementing state persistence

---

### 8. Data Loss from Overly Broad Checkpoints

**What goes wrong:** Build checkpoint system uses `git stash` on ALL dirty files, including user work unrelated to the build.

**Why it happens:**
- Checkpoint logic stashes everything with `git stash --include-untracked` or similar
- No allowlist of what should be checkpointed vs. left alone
- User's TO-DOs, notes, and drafts get stashed alongside system files

**Consequences:**
- Near-total data loss (1,145+ lines of user work stashed and potentially lost)
- Loss of intellectual work: specs, plans, ideas in progress
- Only recovered via manual `git stash list` and `git stash pop`

**Prevention:**
- NEVER use `git stash` for checkpoints. Use explicit file copies or targeted commits instead.
- Define allowlists: What files CAN be modified (system files only)
- Define blocklists: What files MUST NEVER be touched (user data)
  - Blocklist: `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`, `TO-DOs.md`, project files
  - Allowlist: `.aether/aether-utils.sh`, `.aether/docs/`, `.claude/commands/ant/`, `runtime/`

**Phase mapping:**
- Phase 2 (Checkpoint system): This is where the bug was introduced
- Phase 5+ (Updates): Any system that touches user files is dangerous

---

### 9. Update System Without Version Awareness

**What goes wrong:** Per-repo update mechanism lacks version checking, causing silent failures or unexpected behavior.

**Why it happens:**
- No version tracking in local `.aether/` copies
- Sync functions copy all files without checking if update is needed
- No way to notify users when system is outdated

**Consequences:**
- Users run outdated colony systems without knowing
- Bug fixes don't propagate to existing installations
- "Update" command has no way to determine if update is needed

**Prevention:**
- Store version in each repo: `.aether/data/aether-version.json` or in COLONY_STATE.json
- Compare versions before sync: only copy if source version > local version
- Add non-blocking version check at start of commands (`/ant:status`, `/ant:build`)
- Implement semantic versioning comparison

**Phase mapping:**
- Phase 5+ (Distribution): When implementing per-repo update mechanism

---

### 10. Background Task Results vs. Visual Ordering

**What goes wrong:** Build summary appears before agent notification banners, making output appear premature.

**Why it happens:**
- `run_in_background: true` spawns workers that complete in TaskOutput
- Claude Code fires `task-notification` banners asynchronously AFTER the summary
- Data is correct (TaskOutput blocks until complete), but visual ordering misleads

**Consequences:**
- User distrust: "How can it say complete before the agents finished?"
- Confusion about whether verification actually ran
- Undermines confidence in the orchestration system

**Prevention:**
- Avoid `run_in_background: true` for critical verification steps
- Use foreground Task calls (they still run in parallel without the flag)
- If background is needed, add explicit "Waiting for notifications..." delay

**Phase mapping:**
- Phase 2-3 (Build verification): When implementing parallel worker execution

---

## Moderate Pitfalls

### 11. State Isolation Between System and User Data

**What goes wrong:** No clear boundary between "system state" (tools, commands) and "user state" (work in progress, goals, flags).

**Why it happens:**
- All files in `.aether/` treated as system files
- Updates touch files that should be user-controlled
- Checkpoint/archive operations don't distinguish data types

**Consequences:**
- Update system overwrites user's flags, constraints, or colony config
- Hard to migrate colonies between machines (what to copy?)
- Difficult to reset system without losing user work

**Prevention:**
- Explicit directories: `.aether/system/` vs `.aether/data/`
- Define clear ownership: system owns commands, user owns data
- Implement "safe reset" that preserves user data

**Phase mapping:**
- Phase 1-2 (Colony initialization): Design this upfront
- Phase 5 (Distribution): Critical for update mechanism

---

### 12. Command Duplication Without Sync Mechanism

**What goes wrong:** Commands exist in both `.claude/commands/ant/` and `.opencode/commands/ant/` but drift out of sync.

**Why it happens:**
- 25+ commands manually copied between Claude Code and OpenCode variants
- Each edit requires two files with no verification
- Changes to one platform don't automatically propagate

**Consequences:**
- Features work in one platform but not the other
- Bug fixes in one mirror not applied to other
- Maintenance burden increases linearly with command count

**Prevention:**
- Single source of truth: YAML definitions in `src/commands/`
- Generator script: `./bin/generate-commands.sh sync`
- CI check: verify generated output matches committed files

**Phase mapping:**
- Phase 1 (Setup): Implement sync mechanism before commands proliferate

---

### 13. Async State Updates After Context Clear

**What goes wrong:** After `/clear`, colony state is persisted but subsequent commands don't automatically restore context.

**Why it happens:**
- State is written to disk but not reloaded on command start
- Each command starts with empty context from Claude's perspective
- User must manually reference what they were working on

**Prevention:**
- Every `/ant:*` command should start by loading COLONY_STATE.json
- Surface relevant items: "Continuing from Phase 2", "3 blockers active"
- Build context summary from state files at command start

**Phase mapping:**
- Phase 1-2 (Command design): Add context loading to all commands

---

### 14. Magic String Allowlists for File Operations

**What goes wrong:** System files list (`SYSTEM_FILES` array) is hardcoded. New files require code changes.

**Why it happens:**
- Array of filenames directly in `bin/cli.js` or `aether-utils.sh`
- No external configuration file
- Test coverage for this behavior is missing

**Consequences:**
- Adding new commands requires code change, not config change
- Risk of forgetting to add new files to sync operations
- Brittle: one missed file breaks the system

**Prevention:**
- Move to manifest file: `.aether/manifest.json` listing all system files
- Generate manifest from directory structure at build time
- Add tests that verify manifest coverage

**Phase mapping:**
- Phase 3 (CLI implementation): Use config, not code, for file lists

---

## Minor Pitfalls

### 15. No Lockfile for Dependencies

**What goes wrong:** No `package-lock.json` means `npm install` could pull different versions over time.

**Consequences:**
- Non-deterministic builds
- Breaking changes slip in silently
- Hard to reproduce issues

**Prevention:**
- Commit `package-lock.json` to repository

---

### 16. Hash Computation on Every Sync

**What goes wrong:** `syncDirWithCleanup` computes SHA256 for every file on every sync, even unchanged files.

**Consequences:**
- Slow for large repositories
- Wasted CPU cycles
- User frustration with slow commands

**Prevention:**
- Cache hash results by file path + mtime
- Skip hash if mtime hasn't changed since last sync

---

### 17. Event Timestamp Ordering

**What goes wrong:** Events appended to activity log can appear out of chronological order.

**Why it happens:**
- Events from previous sessions appended incorrectly
- No validation that timestamps are monotonically increasing

**Prevention:**
- Validate timestamp on append: reject if earlier than last event
- Sort events by timestamp before reading

---

### 18. No Input Validation on File Paths

**What goes wrong:** File path operations use `process.cwd()` directly without sanitization.

**Consequences:**
- Path traversal attacks possible
- Unexpected file access

**Prevention:**
- Validate paths stay within expected boundaries
- Use path.resolve and check prefix

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip verification for "simple" phases | Faster development | Broken phases marked complete | Never — verification is mandatory |
| Use `git stash` without verification | Simpler checkpoint code | User data loss, recovery confusion | Only with explicit scope checking |
| Degrade file locking gracefully | Works on all systems | State corruption under load | Never for state writes |
| Mock less in sync tests | Easier test writing | Miss real-world edge cases | Only for non-critical paths |
| Background output without buffering | Simpler spawn code | Misleading user experience | Never for user-facing operations |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Stashing files outside Aether directories | User data exposure, privacy risk | Strict allowlist, verify before stash |
| `git reset --hard` without confirmation | Destructive data loss | Interactive confirmation, backup branch |
| Writing state without lock | State corruption, information leak | Mandatory locking for sensitive writes |
| Including user paths in error messages | Path disclosure | Sanitize paths in error output |

## "Looks Done But Isn't" Checklist

- [ ] **Checkpoint system:** Verify stash scope with `git stash show -p` — only Aether files should appear
- [ ] **Phase advancement:** Check that verification loop actually ran — don't trust state flags alone
- [ ] **Update command:** Confirm stash is popped after successful update — check `git stash list`
- [ ] **Sync tests:** Run tests with real filesystem, not just mocks — verify actual file operations
- [ ] **State locking:** Verify lock files are cleaned up after crashes — check `.aether/locks/`
- [ ] **Output ordering:** Review logs from parallel operations — timestamps should make sense

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Git stash captured user data | MEDIUM | `git stash show -p` to inspect; `git stash pop` to restore; manually separate Aether vs user files |
| Phase loop detected | LOW | Reset `current_phase` to last known good value; clear `state` to "READY"; re-run verification |
| Update stash not popped | LOW | Run `git stash list` to find ref; `git stash pop <ref>` to restore; verify working directory |
| State file corrupted | MEDIUM | Restore from git history: `git checkout HEAD -- .aether/data/COLONY_STATE.json`; re-initialize if needed |
| Checkpoint rollback too destructive | HIGH | Use `git reflog` to find lost commits; `git reset --hard <commit>` to recover; may need manual merge |

## Phase-Specific Warning Matrix

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Init | State isolation confusion | Define system vs user data upfront |
| Phase 2: Planning | Command duplication drift | Use YAML source + generator |
| Phase 3: Build | Race conditions in state | Implement file locking + atomic writes |
| Phase 4: Verification | Background task ordering | Use foreground Task calls |
| Phase 5: Distribution | Version awareness missing | Add version tracking + checks |
| Phase 6+: Updates | Overly broad file operations | Use allowlists, never git stash |
| v1.1: Checkpoint Fix | Stash scope too broad | Verify allowlist before stash |
| v1.1: Phase Loop Fix | Missing verification gate | Iron Law enforcement |
| v1.1: Update Repair | Stash not recovered | Always pop on success, warn on failure |
| v1.1: Output Fix | Background task timing | Capture then emit pattern |
| v1.1: Tests | Insufficient sync coverage | Property-based idempotency tests |

## Pitfall-to-v1.1-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|------------|
| Git stash captures user data | v1.1 Checkpoint Fix | Stash contains only allowlisted paths; `git stash show` verification |
| Phase advancement loops | v1.1 Phase Loop Fix | Verification loop runs before any phase change; state transition audit log |
| Update stash not recovered | v1.1 Update Repair | Stash list empty after successful update; warning shown on failure |
| Misleading output timing | v1.1 Output Fix | Background task output captured before prompt return; ordered log entries |
| Missing sync tests | v1.1 Test Addition | Unit tests cover hash comparison, cleanup, edge cases; CI passes |
| Destructive rollback | v1.1 Checkpoint Fix | Prefer stash pop; warn before reset; backup branch created |

## Sources

- Aether TO-DOs.md - Real bugs encountered (data loss from stash, output ordering, version awareness)
- Aether CONCERNS.md - Technical debt and security considerations
- Aether progress.md - Race condition fixes, idempotency issues
- Aether ARCHITECTURE.md - State management patterns
- Codebase analysis: `/Users/callumcowie/repos/Aether/bin/cli.js` — update and checkpoint logic
- Codebase analysis: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` — autofix-checkpoint and autofix-rollback
- Codebase analysis: `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` — checkpoint creation procedure
- Codebase analysis: `/Users/callumcowie/repos/Aether/.claude/commands/ant/continue.md` — phase advancement logic
- Known issues from Oracle research: `/Users/callumcowie/repos/Aether/.aether/oracle/progress.md`

**Confidence Assessment:**

| Area | Level | Reason |
|------|-------|--------|
| Race conditions | HIGH | Based on Aether's actual bug history |
| Data loss bugs | HIGH | Documented in TO-DOs, nearly lost user work |
| Update issues | HIGH | Current P0 work item in TO-DOs |
| v1.1 specific pitfalls | HIGH | Based on direct codebase analysis |
| Minor pitfalls | MEDIUM | Common patterns in CLI tools |

---

*This research informs roadmap planning by flagging which phases need deeper investigation of concurrency, state management, and distribution concerns.*
*Updated for v1.1 bug fix milestone: 2026-02-14*
