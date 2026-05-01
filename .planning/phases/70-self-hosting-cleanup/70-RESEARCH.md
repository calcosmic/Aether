# Phase 70: Self-Hosting Cleanup - Research

**Researched:** 2026-04-28
**Domain:** Repository hygiene, git artifact cleanup, self-hosting leak prevention
**Confidence:** HIGH

## Summary

Phase 70 removes all artifacts that exist because Aether was used to develop itself. These are files and directories that were created by running Aether colonies inside the Aether repo during development -- things like chamber archives, runtime state files, agent mirrors, and worktree orphans. They are not part of the shipped product and never should have been committed to git.

The core operation is straightforward: `git rm` the tracked files and update `.aether/.gitignore` to prevent future leaks. The research uncovered 20 additional self-hosting leak files beyond the 276 specified in CONTEXT.md, bringing the total to 296 files. These extra files (dreams/, data/, midden/, rules/, settings/, registry.json, version.json, QUEEN.md) were committed before gitignore rules were established and remain tracked despite their parent directories now being ignored. Cleaning them now is essential for a truly clean canonical repo.

**Primary recommendation:** Execute a single-commit cleanup removing all 296 tracked self-hosting artifacts, updating `.aether/.gitignore` with comprehensive coverage, and verifying integrity post-cleanup. No Go code references any of the artifacts being removed.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Include orphaned worktree files (7 files in `.claude/worktrees/agent-a9135902/`) in addition to requirements-specified artifacts. These are stale Phase 44 artifacts whose canonical copies already exist in `.planning/phases/`.
- **D-02:** Total cleanup scope: 26 stale agent files + 241 chamber files + 2 runtime state files + 7 worktree artifacts = 276 files removed from git tracking.
- **D-03:** Perform a quick sample check (spot-check 3-5 chambers) before deletion to confirm no irreplaceable data exists. All chambers are from March 2026; active colony data lives in `.aether/data/COLONY_STATE.json` (already gitignored).
- **D-04:** If sample check finds nothing irreplaceable, proceed with `git rm -r` for all 241 chamber files. If something is found, flag it before proceeding.
- **D-07:** Single commit for all cleanup (artifact removal + gitignore update). Easy to review and revert.
- **D-08:** After cleanup, verify `agents-claude/` remains byte-identical to `.claude/agents/ant/` (already confirmed pre-cleanup: all 26 files match MD5 hashes).
- **D-09:** Run `go test ./...` after cleanup to confirm nothing breaks.

### Claude's Discretion
- Exact gitignore entries beyond chambers/ and agents/
- How to structure the sample check (which chambers, what to look for)
- Whether to also clean up local untracked chamber directories (chamber-alpha, chamber-beta) that aren't in git

### Deferred Ideas (OUT OF SCOPE)
None.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CLEAN-01 | Stale `.aether/agents/` directory removed (26 files that duplicate `agents-claude/`) | Verified: 26 tracked files in `.aether/agents/`, no Go code references this path, `.aether/agents-claude/` is byte-identical to `.claude/agents/ant/` |
| CLEAN-02 | Tracked chamber files removed from git (241 files in `.aether/chambers/`) | Verified: 241 tracked files across 21 chamber directories, all from Feb-Apr 2026 colonies |
| CLEAN-03 | Runtime state files removed from git tracking (CONTEXT.md, CROWNED-ANTHILL.md) | Verified: both files tracked at `.aether/CONTEXT.md` and `.aether/CROWNED-ANTHILL.md` |
| CLEAN-04 | Chambers directory added to `.aether/.gitignore` to prevent future self-hosting leaks | Verified: current gitignore lacks chambers/ and agents/ entries |
| CLEAN-05 | Verify `agents-claude/` is byte-identical to `.claude/agents/ant/` after cleanup | Pre-verified: `diff -r` produces no output. Both dirs have 26 files with identical contents |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Artifact identification | Git / VCS | -- | Tracked files are a git-level concern; `git ls-files` is the source of truth |
| File removal | Git / VCS | -- | `git rm` is the correct tool for removing tracked files without deleting local copies |
| Gitignore management | Repo config | -- | `.aether/.gitignore` and `.gitignore` govern future tracking behavior |
| Integrity verification | Build / Test | Git / VCS | `go test ./...` and `diff -r` confirm nothing breaks after removal |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| git | system | File removal, gitignore management | Only correct tool for removing tracked files from VCS |
| diff | system | Byte-identity verification | Built-in, reliable file comparison |
| go test | system (Go toolchain) | Post-cleanup regression check | Ensures no Go code depended on removed artifacts |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| md5sum / shasum | system | Optional hash verification for agent mirrors | If `diff -r` is insufficient for confidence |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `git rm -r` | Manual `git rm` per file | `git rm -r` is faster and equivalent; no tradeoff for batch removal |
| Single commit (D-07) | Multiple commits per artifact type | Single commit is easier to review and revert; per-type commits add noise |

**Installation:** No external dependencies required. All tools are system-level.

**Version verification:** N/A -- all tools are system utilities (git, diff, go test).

## Architecture Patterns

### System Architecture Diagram

```
Pre-cleanup state:
  Git index contains 296 self-hosting artifacts
       |
       v
  [1. Sample check] --> chambers spot-checked for irreplaceable data
       |
       v
  [2. git rm] --> removes all 296 tracked files from index
       |            (local disk copies preserved for data/, dreams/, etc.)
       v
  [3. Update .aether/.gitignore] --> adds agents/, chambers/, and other leak vectors
       |
       v
  [4. Verify] --> diff -r agents-claude/ vs .claude/agents/ant/
       |          go test ./...
       v
  [5. Single commit] --> all changes in one atomic commit
```

### Recommended Project Structure

No structural changes to project layout. This phase only removes files and updates gitignore.

### Pattern 1: git rm for tracked artifacts in gitignored directories
**What:** Files that were committed before a `.gitignore` rule was added remain tracked even after the rule is in place. `git rm --cached` removes them from the index while preserving local copies.
**When to use:** When gitignore was added after files were already committed.
**Example:**
```bash
# Files in .aether/data/ are gitignored but 1 file (COLONY_STATE.json) is still tracked
git ls-files .aether/data/
# .aether/data/COLONY_STATE.json

# Remove from tracking (local copy preserved)
git rm --cached .aether/data/COLONY_STATE.json
```

### Pattern 2: Comprehensive gitignore for self-hosting directories
**What:** Add all directories that should never be tracked to `.aether/.gitignore`, covering both currently tracked and future artifacts.
**When to use:** Preventing future self-hosting leaks.
**Example:**
```gitignore
# Current .aether/.gitignore
data/
dreams/
checkpoints/
locks/

# After update (additions marked with +)
data/
dreams/
checkpoints/
locks/
+agents/
+chambers/
+midden/
+rules/
+settings/
+archive/
+backups/
+oracle/
+temp/
```

### Anti-Patterns to Avoid
- **Using `rm -rf` instead of `git rm`:** `rm -rf` deletes local files permanently. `git rm` only removes from git tracking (with `--cached`) or stages deletion for commit. Always use `git rm`.
- **Forgetting `--cached` for gitignored directories:** If a file is in a gitignored directory but still tracked, use `git rm --cached` to remove from index without deleting the local copy. For files that should be fully deleted (chambers, stale agents), `git rm` without `--cached` is correct.
- **Partial gitignore updates:** Adding only `chambers/` and `agents/` leaves other leak vectors open. Add all self-hosting directories at once.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File removal from git | Custom script to edit git index | `git rm -r` | Handles index update, staging, and file deletion atomically |
| Gitignore management | Manual line-by-line editing | Direct edit of `.aether/.gitignore` | Simple file edit; no tool needed |
| File comparison | Custom hash comparison | `diff -r` | Recursive diff is faster and more reliable for byte-identity checks |

**Key insight:** This phase is pure git operations. No code needs to be written, no scripts need to be built. The "standard stack" is just `git rm`, `diff`, and `go test`.

## Runtime State Inventory

> This phase IS a cleanup/migration phase, so this section is required.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `.aether/data/COLONY_STATE.json` (1 tracked file) | `git rm --cached` (preserve local copy; active colony state) |
| Stored data | `.aether/dreams/` (11 tracked files: .gitkeep + 10 dream journals) | `git rm --cached` (preserve local copies; session notes) |
| Stored data | `.aether/midden/` (3 tracked files: approach-changes.md, build-failures.md, test-failures.md) | `git rm --cached` (preserve local copies; failure tracking) |
| Live service config | `.aether/settings/claude/settings.json` (1 tracked file -- Claude hooks config) | `git rm --cached` (preserve local copy; generated by `aether install`) |
| Live service config | `.aether/rules/aether-colony.md` (1 tracked file -- slightly stale copy of `.claude/rules/`) | `git rm --cached` (local copy is stale; canonical is at `.claude/rules/aether-colony.md`) |
| Build artifacts | `.aether/registry.json` (1 tracked file) | `git rm --cached` (preserve local copy; runtime registry) |
| Build artifacts | `.aether/version.json` (1 tracked file) | `git rm --cached` (preserve local copy; runtime version) |
| Build artifacts | `.aether/QUEEN.md` (1 tracked file) | `git rm --cached` (preserve local copy; generated wisdom file) |
| Stored data | `.aether/chambers/` (241 tracked files in 21 directories) | `git rm -r` (full deletion; stale colony archives from Feb-Apr 2026) |
| Stored data | `.aether/agents/` (26 tracked files) | `git rm -r` (full deletion; duplicates of agents-claude/) |
| Stored data | `.aether/CONTEXT.md` + `.aether/CROWNED-ANTHILL.md` (2 tracked files) | `git rm` (full deletion; stale runtime state from self-hosting colony) |
| OS-registered state | None found | -- |
| Secrets/env vars | None found | -- |

**Nothing found in category OS-registered state:** No OS-level registrations embed Aether self-hosting strings.

**Nothing found in category Secrets/env vars:** No secret keys or environment variable names reference the artifacts being removed.

## Common Pitfalls

### Pitfall 1: Forgetting --cached for gitignored directories
**What goes wrong:** Running `git rm` (without `--cached`) on files in `.aether/data/`, `.aether/dreams/`, etc. will delete the local copies. These directories contain active runtime state (current COLONY_STATE.json, dream journals) that the developer still needs locally.
**Why it happens:** The instinct is to use the same command for all files, but the treatment differs: artifacts that should be fully deleted (chambers, stale agents, runtime state files) vs. artifacts that should be untracked but kept locally (data/, dreams/, midden/, settings/, rules/, QUEEN.md, registry.json, version.json).
**How to avoid:** Split the cleanup into two groups:
1. **Full delete** (local + git): `.aether/agents/`, `.aether/chambers/`, `.aether/CONTEXT.md`, `.aether/CROWNED-ANTHILL.md`, `.claude/worktrees/`
2. **Untrack only** (`--cached`): `.aether/data/`, `.aether/dreams/`, `.aether/midden/`, `.aether/rules/`, `.aether/settings/`, `.aether/registry.json`, `.aether/version.json`, `.aether/QUEEN.md`
**Warning signs:** If `git status` shows deleted files in `data/` or `dreams/` that the developer needs locally, `--cached` was forgotten.

### Pitfall 2: Partial gitignore leaves future leak vectors
**What goes wrong:** Adding only `chambers/` and `agents/` to `.aether/.gitignore` leaves `midden/`, `rules/`, `settings/`, and other directories unprotected. If someone runs a colony in this repo again, those directories could get new files that accidentally get committed.
**Why it happens:** The requirements only mention chambers/ and agents/ explicitly. The additional directories (midden/, rules/, settings/) were discovered during research as tracked files in gitignored-or-should-be-gitignored directories.
**How to avoid:** Add all self-hosting directories to `.aether/.gitignore` in one pass. See the recommended gitignore in Pattern 2 above.
**Warning signs:** After cleanup, running `git ls-files .aether/ | grep -v commands/ | grep -v skills/ | grep -v agents-codex/ | grep -v agents-claude/ | grep -v templates/ | grep -v docs/ | grep -v utils/ | grep -v exchange/ | grep -v schemas/ | grep -v codex/ | grep -v ts/ | grep -v .gitignore | grep -v .npmignore | grep -v workers.md` should return zero results.

### Pitfall 3: Breaking agent mirror sync expectations
**What goes wrong:** Removing `.aether/agents/` could confuse someone who expects the old directory to exist. CLAUDE.md documents `agents-claude/` and `agents-codex/` as the packaging mirrors, but old references or muscle memory might expect `.aether/agents/`.
**Why it happens:** The directory existed historically and was used before the mirror system was formalized.
**How to avoid:** The directory is not referenced anywhere in Go code, YAML commands, or CLAUDE.md. No breakage risk. Adding `agents/` to `.aether/.gitignore` prevents it from being recreated.
**Warning signs:** None -- verified by grep audit.

### Pitfall 4: Worktree files already gitignored but still tracked
**What goes wrong:** `.claude/worktrees/` is in `.gitignore` (line 111 of top-level `.gitignore`), but 7 files in `.claude/worktrees/agent-a9135902/` were committed before the gitignore rule was added. A naive `git rm -r .claude/worktrees/` would fail because the directory is gitignored and git won't match it with glob expansion.
**Why it happens:** Gitignore only prevents new files from being tracked. Files already in the index remain tracked.
**How to avoid:** Use `git rm -r --cached .claude/worktrees/agent-a9135902/` with the explicit path, or use `git ls-files .claude/worktrees/ | xargs git rm` to operate on the tracked file list directly.
**Warning signs:** `git rm -r .claude/worktrees/` returning "did not match any files" even though `git ls-files .claude/worktrees/` shows files.

## Code Examples

Verified patterns from investigation:

### Group 1: Full deletion (stale artifacts with no local value)
```bash
# Stale agent mirror (duplicates of agents-claude/)
git rm -r .aether/agents/

# Chamber archives (stale colony data from Feb-Apr 2026)
git rm -r .aether/chambers/

# Runtime state files from self-hosting colony
git rm .aether/CONTEXT.md .aether/CROWNED-ANTHILL.md

# Orphaned worktree files (Phase 44 artifacts, canonical copies exist)
git ls-files .claude/worktrees/ | xargs git rm
```

### Group 2: Untrack only (active local state, keep on disk)
```bash
# Active colony state (preserve local copy)
git rm --cached .aether/data/COLONY_STATE.json

# Dream journals (preserve local copies)
git ls-files .aether/dreams/ | xargs git rm --cached

# Midden failure logs (preserve local copies)
git ls-files .aether/midden/ | xargs git rm --cached

# Settings and rules (generated by aether install, preserve local)
git rm --cached .aether/settings/claude/settings.json
git rm --cached .aether/rules/aether-colony.md

# Registry, version, QUEEN.md (runtime files, preserve local)
git rm --cached .aether/registry.json .aether/version.json .aether/QUEEN.md
```

### Verification
```bash
# Verify agent mirrors are still byte-identical
diff -r .aether/agents-claude/ .claude/agents/ant/
# Expected: no output

# Verify no unexpected .aether/ files remain tracked
git ls-files .aether/ | grep -v 'commands/' | grep -v 'skills/' | grep -v 'skills-codex/' | grep -v 'agents-codex/' | grep -v 'agents-claude/' | grep -v 'templates/' | grep -v 'docs/' | grep -v 'utils/' | grep -v 'exchange/' | grep -v 'schemas/' | grep -v 'codex/' | grep -v 'ts/' | grep -v '.gitignore' | grep -v '.npmignore' | grep -v 'workers.md'
# Expected: no output

# Run tests to confirm nothing breaks
go test ./...
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `git rm` without `--cached` for all artifacts | Split: full delete for stale, `--cached` for active local state | This phase | Prevents accidental deletion of active COLONY_STATE.json and dream journals |
| Partial gitignore (only data/, dreams/, checkpoints/, locks/) | Comprehensive gitignore covering all self-hosting dirs | This phase | Prevents future self-hosting leaks from any colony run in this repo |

**Deprecated/outdated:**
- `.aether/agents/` directory: Superseded by `.aether/agents-claude/` (packaging mirror) and `.aether/agents-codex/` (Codex mirror). No code references the old path.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | All 241 chamber files contain only runtime state (no unique source code) | Runtime State Inventory | Low risk -- spot-checked 3 chambers from different dates, all contain only COLONY_STATE.json, session files, dreams, pheromones. Code changes from these colonies are already in git history. |
| A2 | `.aether/settings/claude/settings.json` is generated by `aether install` and can be safely untracked | Runtime State Inventory | Low risk -- confirmed by `install_cmd.go` source which copies settings to `systemDir/settings/claude/`. The file is a generated artifact. |
| A3 | `.aether/rules/aether-colony.md` is a stale installed copy, not a source file | Runtime State Inventory | Low risk -- `diff` shows it differs from canonical `.claude/rules/aether-colony.md` (line 50 differs: "Live tmux monitoring" vs "Colony watch dashboard / compatibility view"). The canonical source is `.claude/rules/aether-colony.md`. |
| A4 | `.aether/QUEEN.md` is a generated/self-hosting artifact, not a template | Runtime State Inventory | Medium risk -- `.aether/templates/QUEEN.md.template` exists, suggesting QUEEN.md is generated from the template. If the tracked QUEEN.md contains wisdom that should be preserved, it should be backed up before `git rm --cached`. The npmignore explicitly excludes it with comment "Generated files (created from templates, never ship pre-populated)". |

## Open Questions (RESOLVED)

1. **Should QUEEN.md content be preserved before untracking?**
   - What we know: `.aether/QUEEN.md` is tracked in git, contains colony wisdom (last evolved 2026-03-24). The `.npmignore` marks it as "Generated files (created from templates, never ship pre-populated)".
   - What's unclear: Whether the wisdom content in the tracked QUEEN.md has value that should be preserved somewhere before untracking. The `--cached` flag preserves the local copy, so no data is lost -- but the git history version will no longer be the source of truth.
   - Recommendation: Use `git rm --cached` (preserves local copy). No special backup needed since local file is untouched.

2. **Should local untracked chamber directories be cleaned?**
   - What we know: 11 chamber directories exist on disk but only 21 are tracked. The 11 untracked ones include `chamber-alpha`, `chamber-beta`, `phase-chamber`, `test-chamber`, and `verify-test`.
   - What's unclear: Whether these local-only directories should also be deleted from disk.
   - Recommendation: Leave them on disk. They are already gitignored (once chambers/ is added to gitignore) and contain no tracked data. Deleting them is unnecessary scope expansion.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this phase uses only git, diff, and go test, all of which are available on any development machine).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- Go stdlib |
| Quick run command | `go test ./...` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLEAN-01 | `.aether/agents/` no longer tracked | manual/smoke | `git ls-files .aether/agents/` (expect empty) | N/A -- git state check |
| CLEAN-02 | No chamber files tracked | manual/smoke | `git ls-files .aether/chambers/` (expect empty) | N/A -- git state check |
| CLEAN-03 | Runtime state files untracked | manual/smoke | `git ls-files .aether/CONTEXT.md .aether/CROWNED-ANTHILL.md` (expect empty) | N/A -- git state check |
| CLEAN-04 | Gitignore covers chambers/ | manual/smoke | `git check-ignore .aether/chambers/test` (expect matched) | N/A -- gitignore check |
| CLEAN-05 | Agent mirrors byte-identical | manual/smoke | `diff -r .aether/agents-claude/ .claude/agents/ant/` (expect no output) | N/A -- diff check |

### Sampling Rate
- **Per task commit:** `go test ./...` (regression check)
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full verification script (all 5 checks above + `go test ./...`)

### Wave 0 Gaps
None -- this phase requires no new test files. All verification is smoke-level git state checks and a regression test run.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | no | -- |
| V6 Cryptography | no | -- |

Security is not applicable to this phase. No authentication, session management, access control, input validation, or cryptographic operations are involved. This is purely a git hygiene operation.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: git ls-files] -- All file counts verified by running `git ls-files` against the actual repository (2026-04-28)
- [VERIFIED: diff -r] -- Agent mirror byte-identity confirmed by recursive diff of `.aether/agents-claude/` vs `.claude/agents/ant/` (2026-04-28)
- [VERIFIED: git check-ignore] -- Gitignore coverage gaps identified by checking each directory against both `.aether/.gitignore` and top-level `.gitignore` (2026-04-28)
- [VERIFIED: grep audit] -- No Go code references `.aether/agents/` path confirmed by recursive grep of `cmd/` and `pkg/` (2026-04-28)
- [VERIFIED: CLAUDE.md] -- Architecture documentation confirms `agents-claude/` and `agents-codex/` as canonical mirrors, not `.aether/agents/` (2026-04-28)
- [CITED: .aether/.npmignore] -- QUEEN.md marked as "Generated files (created from templates, never ship pre-populated)"
- [CITED: .aether/.gitignore] -- Current gitignore covers data/, dreams/, checkpoints/, locks/ only

### Secondary (MEDIUM confidence)
- [CITED: 70-CONTEXT.md] -- File count summaries and artifact descriptions from discuss phase
- [CITED: 34-CONTEXT.md] -- Prior cleanup phase established `git rm` patterns for artifact removal

### Tertiary (LOW confidence)
- None -- all findings verified against actual repository state.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - git, diff, go test are system utilities with no version uncertainty
- Architecture: HIGH - pure git operations with no code changes; verified by direct file system inspection
- Pitfalls: HIGH - all pitfalls identified from direct testing of git behavior (e.g., `--cached` requirement, gitignored-but-tracked files)

**Research date:** 2026-04-28
**Valid until:** 90 days (git behavior and repository structure are stable)
