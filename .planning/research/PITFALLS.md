# Domain Pitfalls: Multi-Agent CLI Orchestration Systems

**Domain:** AI agent orchestration, CLI-based multi-agent systems
**Researched:** 2026-02-13
**Confidence:** MEDIUM-HIGH (based on Aether's real-world issues and general orchestration patterns)

## Overview

Multi-agent CLI orchestration systems like Aether face unique challenges that single-agent systems do not. This document catalogs common pitfalls based on Aether's real-world bugs (race conditions, data loss, update failures) and patterns observed in similar systems.

---

## Critical Pitfalls

### 1. Race Conditions in Shared State

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

### 2. Data Loss from Overly Broad Checkpoints

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

### 3. Update System Without Version Awareness

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

### 4. Background Task Results vs. Visual Ordering

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

### 5. State Isolation Between System and User Data

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

### 6. Command Duplication Without Sync Mechanism

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

### 7. Async State Updates After Context Clear

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

### 8. Magic String Allowlists for File Operations

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

### 9. No Lockfile for Dependencies

**What goes wrong:** No `package-lock.json` means `npm install` could pull different versions over time.

**Consequences:**
- Non-deterministic builds
- Breaking changes slip in silently
- Hard to reproduce issues

**Prevention:**
- Commit `package-lock.json` to repository

---

### 10. Hash Computation on Every Sync

**What goes wrong:** `syncDirWithCleanup` computes SHA256 for every file on every sync, even unchanged files.

**Consequences:**
- Slow for large repositories
- Wasted CPU cycles
- User frustration with slow commands

**Prevention:**
- Cache hash results by file path + mtime
- Skip hash if mtime hasn't changed since last sync

---

### 11. Event Timestamp Ordering

**What goes wrong:** Events appended to activity log can appear out of chronological order.

**Why it happens:**
- Events from previous sessions appended incorrectly
- No validation that timestamps are monotonically increasing

**Prevention:**
- Validate timestamp on append: reject if earlier than last event
- Sort events by timestamp before reading

---

### 12. No Input Validation on File Paths

**What goes wrong:** File path operations use `process.cwd()` directly without sanitization.

**Consequences:**
- Path traversal attacks possible
- Unexpected file access

**Prevention:**
- Validate paths stay within expected boundaries
- Use path.resolve and check prefix

---

## Phase-Specific Warning Matrix

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Init | State isolation confusion | Define system vs user data upfront |
| Phase 2: Planning | Command duplication drift | Use YAML source + generator |
| Phase 3: Build | Race conditions in state | Implement file locking + atomic writes |
| Phase 4: Verification | Background task ordering | Use foreground Task calls |
| Phase 5: Distribution | Version awareness missing | Add version tracking + checks |
| Phase 6+: Updates | Overly broad file operations | Use allowlists, never git stash |

---

## Sources

- Aether TO-DOs.md - Real bugs encountered (data loss from stash, output ordering, version awareness)
- Aether CONCERNS.md - Technical debt and security considerations
- Aether progress.md - Race condition fixes, idempotency issues
- Aether ARCHITECTURE.md - State management patterns

**Confidence Assessment:**

| Area | Level | Reason |
|------|-------|--------|
| Race conditions | HIGH | Based on Aether's actual bug history |
| Data loss bugs | HIGH | Documented in TO-DOs, nearly lost user work |
| Update issues | HIGH | Current P0 work item in TO-DOs |
| Minor pitfalls | MEDIUM | Common patterns in CLI tools |

---

*This research informs roadmap planning by flagging which phases need deeper investigation of concurrency, state management, and distribution concerns.*
