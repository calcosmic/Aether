# Codebase Concerns

**Analysis Date:** 2026-02-13

## Critical Bugs

**Build checkpoint stashes user data - nearly lost Oracle spec:**
- Issue: The build checkpoint system uses `git stash` on ALL dirty files, including user work that has nothing to do with phase. During repo-local migration (phase 5), checkpoint stashed 1,145 lines of uncommitted TO-DOS.md content (Oracle spec, 10 advanced colony ideas, multi-ant vision) and never popped it back. User nearly lost hours of work -- only recovered by manually searching git stashes.
- Files: `.claude/commands/ant/build.md` (checkpoint logic), `.aether/aether-utils.sh`
- Root cause: `git stash` is a blunt instrument. The checkpoint system doesn't distinguish between "system files I'm about to modify" and "user's unrelated work in progress."
- Fix approach: The build/update system must ONLY modify files on an explicit allowlist. System files are the tools; user data is their work. Updates should only touch tool, never work. System files (safe to modify): `.aether/*.md`, `.aether/aether-utils.sh`, `.aether/docs`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `runtime/`, `bin/cli.js`. User data (NEVER touch): `.aether/data/`, `.aether/dreams/`, `.aether/oracle`, `TO-DOS.md`, `COLONY_STATE.json`, flags, learnings, constraints.

## High Priority Issues

**run_in_background in build.md causes misleading output timing:**
- Issue: Phase summary appears before all background agent notifications are shown to the user. During `/ant:build`, workers are spawned with `run_in_background: true` and then collected via `TaskOutput` with `block: true`. The Queen synthesizes results and displays summary based on TaskOutput data (which IS real completed output). However, Claude Code's `task-notification` banners for each agent arrive asynchronously AFTER the summary is already displayed, making it look like the summary was written before agents finished.
- Files: `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`
- Impact: Users may think verification was premature
- Fix approach: Remove `run_in_background: true` from all Task calls in build.md Steps 5.1, 5.4, and 5.4.2. Multiple Task calls in a single message already run in parallel without background flag. Then remove Steps 5.2 and 5.4.1 (TaskOutput collection) since results come back from Task calls themselves.

**Stale npm 2.x versions cause public confusion:**
- Issue: npm registry has stale 2.x pre-release versions (2.0.0 through 2.4.2) that could confuse users. The `latest` dist-tag correctly points to 1.0.0, so `npm install` works fine. But 2.x versions are visible on npm page and could confuse people into thinking they're newer.
- Files: package.json (npm publishing)
- Fix approach: Run `npm deprecate aether-colony@">=2.0.0" "Pre-release versions. Install 1.0.0 for stable release."`

## Tech Debt

**Large monolithic shell script:**
- Issue: `runtime/aether-utils.sh` is 1,390 lines of shell code with many functions. This makes it difficult to maintain, test, and understand.
- Files: `runtime/aether-utils.sh`, `.aether/aether-utils.sh`
- Impact: Hard to debug, high risk of unintended side effects when modifying
- Fix approach: Consider extracting into modular scripts by function category (e.g., `utils/git.sh`, `utils/ants.sh`, `utils/phases.sh`)

**No JavaScript linting for main CLI:**
- Issue: `package.json` includes `lint:shell` for shell scripts but no linting for the main `bin/cli.js` (811 lines). Missing ESLint or similar tooling.
- Files: `bin/cli.js`, package.json
- Fix approach: Add ESLint configuration with appropriate Node.js rules

**Checkpoint state not properly isolated:**
- Issue: The system lacks clear separation between "system state" (colony configuration, tools) and "user state" (work in progress, dreams, flags). Checkpoint operations can affect both.
- Files: `.aether/aether-utils.sh` (checkpoint functions), `.claude/commands/ant/build.md`
- Fix approach: Define clear allowlists for what each operation can modify

## Known Missing Features

**No colony lifecycle management:**
- Issue: Colony model is rigid: init -> plan -> build all phases -> complete. No way to park a colony mid-work, archive what was done, and start something new.
- Files: `.claude/commands/ant/*.md`, `.aether/data/COLONY_STATE.json`
- Impact: Users cannot naturally switch between different projects or archive sessions
- Fix approach: Implement `/ant:archive` command, milestone auto-detection, colony history browsing

**No session continuity tracking:**
- Issue: After clearing context (`/clear`), colony commands don't automatically restore relevant context from persistent state.
- Files: All `.claude/commands/ant/*.md` command files
- Fix approach: Add universal context loading at start of every `/ant:*` command

## Security Considerations

**Git stash during updates could lose work:**
- Issue: The `--force` flag on update causes git stash of user files, which could result in lost work if stash isn't properly recovered.
- Files: `bin/cli.js` (lines 477-479, 684)
- Current mitigation: Warning message suggests `git stash pop`, but no automatic recovery tracking
- Recommendations: Track stash operations in a local log, verify stash pop after update, or better yet -- never stash user files at all

**No input validation on file paths:**
- Issue: File path operations in `cli.js` use user-provided paths without sanitization (e.g., `process.cwd()` used directly).
- Files: `bin/cli.js`
- Risk: Path traversal attacks if running in untrusted directories
- Recommendations: Validate paths stay within expected boundaries

## Performance Bottlenecks

**Hash computation on every sync:**
- Issue: `syncDirWithCleanup` computes SHA256 hash for every file on every sync operation, even when files haven't changed. This is slow for large repositories.
- Files: `bin/cli.js` (lines 131-139, 224-232)
- Improvement path: Cache hash results or use mtime-based comparison for quick skip

**Manifest generation reads all files:**
- Issue: `generateManifest` reads and hashes every file in the hub directory on every operation.
- Files: `bin/cli.js` (lines 186-200)
- Improvement path: Incremental manifest updates or lazy manifest generation

## Fragile Areas

**Magic string allowlist for system files:**
- Issue: `SYSTEM_FILES` array in `cli.js` (lines 78-99) is hardcoded. New system files require code changes to be included in sync operations.
- Files: `bin/cli.js` (lines 78-99)
- Safe modification: Document the pattern clearly; consider moving to a manifest file
- Test coverage: No tests for this allowlist behavior

**Git dirty file detection:**
- Issue: `getGitDirtyFiles` relies on `git status --porcelain` which can be fragile with special characters in filenames.
- Files: `bin/cli.js` (lines 328-341)
- Test coverage: Limited -- only tested implicitly through integration tests

## Test Coverage Gaps

**No unit tests for core sync functions:**
- Issue: The critical file sync and hash comparison logic in `cli.js` has no dedicated unit tests. Only integration tests exist in `tests/e2e/`.
- Files: `bin/cli.js` (functions: `syncDirWithCleanup`, `hashFileSync`, `generateManifest`)
- What's not tested: Hash collision handling, empty directory cleanup, dry-run mode edge cases
- Risk: Silent failures or data loss during sync operations
- Priority: High

**No tests for error conditions in shell scripts:**
- Issue: `aether-utils.sh` has no test coverage despite being critical infrastructure.
- Files: `runtime/aether-utils.sh`, `.aether/aether-utils.sh`
- What's not tested: All functions (pheromones, ants, phases, verification)
- Risk: Undetected bugs in core colony logic
- Priority: Medium

## Dependency Risks

**No lockfile for npm dependencies:**
- Issue: No `package-lock.json` or `yarn.lock` in the repository. `npm install` could pull different versions over time.
- Files: package.json
- Impact: Non-deterministic builds
- Migration plan: Run `npm install` to generate `package-lock.json` and commit it

---

*Concerns audit: 2026-02-13*
