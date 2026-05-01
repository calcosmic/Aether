# Architecture Research: Aether v1.11 Integration

**Domain:** Smart Init, suggest-analyze, and self-hosting cleanup for the Aether Go codebase
**Researched:** 2026-04-28
**Overall confidence:** HIGH (based on direct source code analysis of cmd/, pkg/, and wrapper markdown)

## Executive Summary

Aether v1.11 restores three lost intelligence features from the original shell-to-Go migration (April 2026) and performs a self-hosting cleanup. The integration architecture is clean because the Go runtime already has well-defined hooks at the right points -- the shell features were never re-ported, so this is additive integration, not retrofitting.

**Smart Init** integrates into the existing `aether init` command flow as a pre-init ceremony: `init-research` already runs as a standalone Cobra subcommand called by wrapper markdown. The missing piece is the charter approval flow (currently the wrapper does the approval UX, but the Go runtime has no ceremony for it). The architecture is: wrapper calls `init-research`, presents charter for approval, then calls `init`. No Go-side ceremony gap needs fixing -- the wrappers own the approval UX per the wrapper-runtime contract.

**Suggest-analyze** is the biggest integration surface. The original shell version was 618 lines of pattern detection. The Go `init-research` already has `generatePheromoneSuggestions()` with 10 patterns, but this only runs during init. The build-time suggest-analyze (called in build-prep.md Step 4.2) was never ported to Go -- it only exists as a documentation concept. The architecture requires: (1) a new `suggest-analyze` Cobra subcommand, (2) a `suggest-approve` tick-to-approve command, (3) integration into the build flow via the Codex build pipeline (`executeCodexBuildDispatches`), and (4) deduplication against existing active signals.

**Self-hosting cleanup** is the simplest -- it is deletion and documentation. The agent mirrors (`.aether/agents-claude/`, `.aether/agents-codex/`, `.aether/skills-codex/`) are already byte-identical to their sources. The cleanup is: (1) audit for orphaned companion files in `.aether/`, (2) verify agent mirror integrity as part of `aether integrity`, (3) remove any stale artifacts that exist because Aether was used to develop itself.

## Key Findings

**Stack:** Go 1.24, Cobra CLI, pkg/storage file locking -- no new dependencies needed
**Architecture:** Three independent feature streams with one shared integration surface (pheromone pipeline)
**Critical pitfall:** suggest-analyze during build must not block the build pipeline if the scan is slow or the user has `--no-suggest`

## Integration Architecture

### Component Map

```
EXISTING                                           NEW (v1.11)
========                                           ============

cmd/init_cmd.go .......................... cmd/suggest_analyze.go
cmd/init_research.go .................... cmd/suggest_analyze.go
cmd/pheromone_write.go .................. cmd/suggest_analyze.go
cmd/colony_prime_context.go ............. (no change -- reads pheromones.json)
cmd/codex_build.go ...................... cmd/codex_build.go (suggest hook)
cmd/recover_scanner.go .................. cmd/integrity_audit.go (new)
.aether/docs/command-playbooks/ ......... build-prep.md (suggest step)

WRAPPER LAYER
=============
.claude/commands/ant/init.md ............ (minor -- suggest-analyze step)
.opencode/commands/ant/init.md .......... (minor -- suggest-analyze step)
.codex/CODEX.md ......................... (new build-prep suggest step)
```

### Data Flow: Smart Init

```
User: /ant-init "Build feature X"
  |
  v
Wrapper (.claude/commands/ant/init.md):
  1. aether init-research --goal "Build feature X" --target .
     |-> Returns: charter, pheromone_suggestions, governance, git_history, etc.
  |
  v
Wrapper presents charter for approval:
  - Intent, Vision, Governance, Goals
  - Tick-to-approve pheromone suggestions
  |
  v
User approves/revises
  |
  v
Wrapper: for each approved suggestion:
  aether pheromone-write --type FOCUS --content "..." --source "init-research"
  |
  v
Wrapper: aether init "Build feature X"
  |-> Creates COLONY_STATE.json, session.json
  |-> Returns state + shelf_backlog
  |
  v
Wrapper: handles shelf backlog (promote/dismiss)
```

**Key insight:** The Go runtime already does everything needed. The wrapper markdown (init.md) already implements the full ceremony. The gap documented in PROJECT.md ("charter ceremony lost in shell-to-Go migration") is actually about the Codex lane -- Codex has no wrapper markdown, so it skips the charter approval and pheromone suggestion steps entirely.

**For Codex:** Add a charter approval step to the `aether init` command itself, controlled by a `--no-ceremony` flag. The Go runtime presents charter data in structured output and the Codex agent workflow handles approval. Alternatively, add a `aether init --charter-only` mode that returns the charter for review before creating the colony.

### Data Flow: Suggest-Analyze (Build Time)

```
User: /ant-build 1
  |
  v
Codex build pipeline (cmd/codex_build.go):
  runCodexBuildWithOptions()
    |
    +-- NEW: if !options.NoSuggest {
    |     suggestions = runSuggestAnalyze(root, phase)
    |     // Returns []pheromoneSuggestion (deduped against active signals)
    |     // Stores to .aether/data/suggest-pending.json for approve step
    |   }
    |
    +-- existing: validateCodexBuildState()
    +-- existing: plannedBuildDispatchesForSelection()
    +-- existing: executeCodexBuildDispatches()
```

**For wrapper builds (Claude/OpenCode):** The suggest-analyze step is already documented in build-prep.md header but never implemented as a Go command. The wrapper would call:

```
aether suggest-analyze --phase 1
-> Returns: suggestions (deduped)
Wrapper presents for approval
aether suggest-approve --ids "1,3,5"  // tick-to-approve
-> Writes approved signals to pheromones.json
```

**For Codex builds:** The Codex build pipeline (`executeCodexBuildDispatches`) would call `runSuggestAnalyze()` as a pre-dispatch step. Since Codex has no tick-to-approve UI, suggestions would be auto-written as low-strength FOCUS signals, or skipped entirely (Codex builds are already verbose).

### Data Flow: Self-Hosting Cleanup

```
aether integrity
  |
  +-- NEW: auditAgentMirrors()
  |     Compare .claude/agents/ant/*.md vs .aether/agents-claude/*.md
  |     Compare .codex/agents/*.toml vs .aether/agents-codex/*.toml
  |     Compare .aether/skills/ vs .aether/skills-codex/
  |     Report mismatches
  |
  +-- NEW: auditOrphanedCompanionFiles()
  |     Check for files in .aether/ that have no corresponding source
  |     Report stale artifacts
  |
  +-- existing: binary version check
  +-- existing: hub version check
  +-- existing: release pipeline chain
```

## Component Boundaries

### New: cmd/suggest_analyze.go

| Responsibility | Communicates With |
|----------------|-------------------|
| Pattern detection (reuse `generatePheromoneSuggestions` from `init_research.go`) | pheromones.json (dedup check), suggest-pending.json (storage) |
| Deduplication against active signals | `pheromone_write.go` signal format |
| Approval flow (`suggest-approve` subcommand) | pheromones.json (writes approved signals) |
| Phase-aware pattern detection | COLONY_STATE.json (reads current phase, plan) |

**Key design decision:** Extract `generatePheromoneSuggestions()` from `init_research.go` into a shared function (or move to a new file) so both `init-research` and `suggest-analyze` can use it. Currently it is a package-level function in `cmd/`, so it is already accessible -- no extraction needed.

**Additional patterns for build-time analysis** (beyond the 10 init patterns):
- Large files (>1MB) -> FOCUS about refactoring
- TODO/FIXME density -> FOCUS about tech debt
- Test file coverage gaps -> FOCUS about testing
- Debug artifacts (console.log, fmt.Println) -> REDIRECT about debug code
- Circular dependencies -> REDIRECT about architecture

### New: cmd/integrity_audit.go

| Responsibility | Communicates With |
|----------------|-------------------|
| Agent mirror byte comparison | Filesystem (3 mirror locations) |
| Orphan detection in .aether/ | Filesystem, manifest |
| Report generation | stdout (JSON/visual) |

### Modified: cmd/codex_build.go

| Change | Location | Impact |
|--------|----------|--------|
| Add suggest-analyze hook | `runCodexBuildWithOptions()` before dispatch | Non-blocking, gated by `--no-suggest` flag |
| Add `NoSuggest` to `codexBuildOptions` | Struct definition | New field |
| Store suggestions as build metadata | `last-build-claims.json` or new file | Audit trail |

### Modified: cmd/init_cmd.go

| Change | Location | Impact |
|--------|----------|--------|
| Return charter data in init output | `outputWorkflow()` call | Enables Codex to present charter |
| Return pheromone suggestions in init output | `outputWorkflow()` call | Enables Codex auto-suggest |

### Modified: .aether/docs/command-playbooks/build-prep.md

| Change | Location | Impact |
|--------|----------|--------|
| Add suggest-analyze step as Go command | After Step 3.1 | Replaces documentation-only concept with real command |

## Patterns to Follow

### Pattern 1: Suggest-Analyze as Pre-Dispatch Hook

**What:** Run pattern analysis before worker dispatch, store suggestions for approval, write approved signals.

**When:** During `aether build` (both wrapper and Codex paths).

**Why:** This follows the existing build-prep pattern where `init-research` runs before `init`. The suggest step is a parallel -- it runs before dispatch to inform workers with fresh signals.

```go
// In cmd/codex_build.go, before executeCodexBuildDispatches:
if !options.NoSuggest {
    suggestions := runBuildSuggestAnalyze(root, phase, store)
    if len(suggestions) > 0 {
        storePendingSuggestions(store, suggestions)
        // For Codex: auto-approve with low strength
        // For wrappers: return in output for tick-to-approve
    }
}
```

### Pattern 2: Deduplication Against Active Signals

**What:** Before presenting suggestions, check pheromones.json for existing signals with the same type+content hash.

**When:** Every suggest-analyze invocation.

**Why:** The pheromone system already has content-hash-based dedup in `pheromone_write.go`. Suggest-analyze must use the same dedup logic to avoid presenting duplicates.

```go
func dedupSuggestionsAgainstActive(existing []colony.PheromoneSignal, new []pheromoneSuggestion) []pheromoneSuggestion {
    activeHashes := make(map[string]bool)
    for _, sig := range existing {
        if sig.Active && sig.ContentHash != nil {
            activeHashes[*sig.ContentHash] = true
        }
    }
    var filtered []pheromoneSuggestion
    for _, s := range new {
        hash := "sha256:" + sha256Sum(s.Content)
        if !activeHashes[hash] {
            filtered = append(filtered, s)
        }
    }
    return filtered
}
```

### Pattern 3: Agent Mirror Integrity as Part of `aether integrity`

**What:** Compare source agent files against their packaging mirrors during integrity checks.

**When:** `aether integrity` invocation (also runs during `aether publish`).

**Why:** Self-hosting artifacts are a silent source of drift. A byte-level comparison catches accidental edits to mirrors.

```go
func auditAgentMirrors(baseDir string) []integrityIssue {
    // Compare .claude/agents/ant/*.md vs .aether/agents-claude/*.md
    // Compare .codex/agents/*.toml vs .aether/agents-codex/*.toml
    // Compare .aether/skills/ vs .aether/skills-codex/
    // Report any differences as warnings
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Suggest-Analyze Blocks Build

**What:** Running suggest-analyze synchronously in the build pipeline and failing if it errors.

**Why bad:** The suggest step is advisory. A slow or failed scan should not prevent the build from proceeding.

**Instead:** Run suggest-analyze with a timeout. If it fails or times out, log a warning and proceed with the build. The `--no-suggest` flag provides an explicit opt-out.

### Anti-Pattern 2: Charter Approval in Go Runtime

**What:** Adding interactive approval prompts to `aether init` in the Go binary.

**Why bad:** The wrapper-runtime contract says wrappers own presentation. Adding approval logic to the Go runtime breaks this contract and creates a CLI UX problem for non-interactive use.

**Instead:** Return charter data in structured output. Let the wrapper (or Codex agent workflow) handle the approval UX. For Codex, add a `--charter-only` flag to `init-research` that returns just the charter without creating the colony.

### Anti-Pattern 3: Deleting Self-Hosting Artifacts Without Audit

**What:** Removing files from `.aether/agents-claude/` or `.aether/agents-codex/` because they "look stale."

**Why bad:** These are packaging mirrors that `aether publish` distributes. Deleting them breaks the publish pipeline.

**Instead:** Run the byte-comparison audit first. Only flag files that differ from their source. Never delete -- report and let the user decide.

### Anti-Pattern 4: Suggest-Analyze Writes Signals Directly

**What:** Having `suggest-analyze` write pheromone signals directly to pheromones.json without user approval.

**Why bad:** Auto-writing signals bypasses the user's control over their colony's behavior. The pheromone system is the user-colony communication channel -- it should not be automated without consent.

**Instead:** `suggest-analyze` stores suggestions to a pending file (`suggest-pending.json`). A separate `suggest-approve` command writes approved signals. For Codex auto-approve, use a low strength (0.5) and source "auto:suggest" so they are distinguishable.

## Scalability Considerations

| Concern | Impact |
|---------|--------|
| Suggest-analyze scan time on large repos | Cap file walk at 10K files or 5s timeout; skip vendor/node_modules (already in `extendedSkipDirs`) |
| Pheromone dedup with many active signals | O(N*M) where N=active signals, M=suggestions; both are small (<100 each) |
| Agent mirror comparison | O(F) file reads where F=26 agents + 29 skills; negligible |
| Integrity audit in publish pipeline | Adds <100ms to `aether integrity`; acceptable |

## Build Order (Suggested Phase Sequence)

Based on dependency analysis:

### 1. Self-Hosting Cleanup (lowest risk, highest signal)

**Why first:** It is pure deletion and audit. No new logic. Proves the integrity pipeline works. Removes noise from the codebase before adding new features.

**Files:**
- `cmd/integrity_audit.go` (new -- agent mirror audit + orphan detection)
- Modify `aether integrity` to include new audits
- Remove any confirmed orphaned files

**Dependencies:** None. Standalone.

### 2. Suggest-Analyze (new command)

**Why second:** It is a new standalone command that does not modify existing behavior. Can be tested in isolation.

**Files:**
- `cmd/suggest_analyze.go` (new -- `suggest-analyze` and `suggest-approve` subcommands)
- Reuse `generatePheromoneSuggestions()` from `init_research.go`
- Add `suggest-pending.json` storage format
- Tests: `cmd/suggest_analyze_test.go`

**Dependencies:** `pheromone_write.go` (for writing approved signals), `init_research.go` (for pattern functions).

### 3. Suggest-Analyze Build Integration (wires into build)

**Why third:** Depends on suggest-analyze command existing.

**Files:**
- `cmd/codex_build.go` (add suggest hook to `runCodexBuildWithOptions`)
- `cmd/codex_build.go` (add `NoSuggest` to `codexBuildOptions`)
- `.aether/docs/command-playbooks/build-prep.md` (add suggest step)
- Wrapper markdown updates (Claude/OpenCode init and build)

**Dependencies:** `cmd/suggest_analyze.go` must exist.

### 4. Smart Init Hardening (charter ceremony for Codex)

**Why last:** The charter ceremony already works in wrappers. This phase extends it to the Codex lane and adds any missing pieces.

**Files:**
- `cmd/init_cmd.go` (return charter + suggestions in output)
- `.codex/CODEX.md` (add charter approval workflow)
- Possibly `cmd/init_research.go` (add `--charter-only` flag)

**Dependencies:** `cmd/suggest_analyze.go` (for Codex to auto-suggest after init).

## Integration Points Summary

| Integration Point | Existing File | Change Type | Risk |
|------------------|---------------|-------------|------|
| suggest-analyze command | `cmd/suggest_analyze.go` | NEW | Low |
| suggest-approve command | `cmd/suggest_analyze.go` | NEW | Low |
| Build pipeline suggest hook | `cmd/codex_build.go` | MODIFY (add hook) | Medium |
| Init output enhancement | `cmd/init_cmd.go` | MODIFY (add fields) | Low |
| Integrity audit | `cmd/integrity_audit.go` | NEW | Low |
| Build-prep playbook | `build-prep.md` | MODIFY (add step) | Low |
| Wrapper init commands | `ant/init.md` | MODIFY (minor) | Low |
| Codex init workflow | `CODEX.md` | MODIFY (add charter) | Low |

## Open Questions

1. **Codex suggest-analyze UX:** Should Codex auto-approve suggestions or skip them entirely? The `--no-suggest` flag already exists for wrappers. Codex builds have no tick-to-approve UI. Recommendation: auto-write with low strength and `auto:suggest` source, or skip entirely.

2. **Suggest-analyze scope for builds:** Should build-time suggest-analyze use the same 10 patterns as init, or a different set? Init patterns are about project setup (no .gitignore, no CI). Build patterns should be about code quality (TODOs, debug artifacts, large files). Recommendation: use a separate `generateBuildPheromoneSuggestions()` function with build-relevant patterns.

3. **Agent mirror automation:** Should `aether publish` auto-sync agent mirrors, or should it require manual confirmation? Currently the mirrors are maintained manually. Recommendation: auto-sync with a diff report, no confirmation needed (they are generated artifacts).

4. **Pending suggestions lifecycle:** How long should `suggest-pending.json` persist? Should unapproved suggestions expire? Recommendation: clear on build completion (they become stale if the codebase changed during the build).

## Sources

- Direct source code analysis: `cmd/init_research.go`, `cmd/init_cmd.go`, `cmd/codex_build.go`, `cmd/pheromone_write.go`, `cmd/colony_prime_context.go`, `cmd/discuss_analyze.go`
- Wrapper markdown: `.claude/commands/ant/init.md`, `.aether/docs/command-playbooks/build-prep.md`, `.aether/docs/command-playbooks/build-wave.md`
- Platform documentation: `.codex/CODEX.md`, `.opencode/OPENCODE.md`
- Agent definitions: `.claude/agents/ant/`, `.aether/agents-claude/`, `.codex/agents/`, `.opencode/agents/`
- Colony types: `pkg/colony/colony.go`, `pkg/storage/storage.go`
- HIGH confidence: all findings based on direct source code reading
