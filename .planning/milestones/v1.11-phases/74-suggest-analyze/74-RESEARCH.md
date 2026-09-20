# Phase 74: Suggest-Analyze - Research

**Researched:** 2026-04-29
**Domain:** Go CLI command implementation, pheromone system integration, build ceremony workflow
**Confidence:** HIGH

## Summary

This phase implements three requirements (INTEL-01, INTEL-02, INTEL-03) that restore the suggest-analyze feature from the shell-to-Go migration. The core work is a new `aether suggest-analyze` Go CLI command that reuses the existing 25 pattern detectors from `cmd/init_research.go:generatePheromoneSuggestions()`, adds build-specific extra patterns, deduplicates against active pheromones via content hash comparison, and persists suggestions for review. The existing `aether suggest-approve` stub in `cmd/compatibility_cmds.go` (lines 113-133) needs real implementation. The existing `aether pheromone-write` command already handles content hash dedup and reinforcement, so approved suggestions pipe through it directly. The wrapper-runtime contract requires Go runtime to own state mutations; wrappers only present output.

The main complexity is not in pattern detection (already done) but in the orchestration: change detection for re-analysis, persistent suggestion storage in colony state, deduplication against active pheromones, and wiring Step 4.2 of the build ceremony back from its current DEPRECATED state.

**Primary recommendation:** Extract `generatePheromoneSuggestions()` and its helper functions into a shared internal package or make them accessible from the new command, then build `suggest-analyze` as a standalone CLI command that calls the shared pattern logic plus build-specific extras.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Suggest-analyze runs on the first build of each colony, then re-runs on subsequent builds only if the codebase changed significantly since last analysis. Change detection uses git diff (files changed since last analysis timestamp or commit).
- **D-02:** Uses the same 25 base patterns from `generatePheromoneSuggestions()` in `cmd/init_research.go` PLUS build-specific extras. Build-specific extras detect: TODO/FIXME density, test coverage gaps, large files, and similar build-relevant patterns.
- **D-03:** The `--no-suggest` flag (already documented in build-prep.md) skips suggest-analyze entirely.
- **D-04:** Suggestions display inline during the build ceremony at Step 4.2, after context assembly and before skill detection. User reviews and approves/dismisses each suggestion before workers spawn. This briefly pauses the build flow.
- **D-05:** Unreviewed suggestions persist in colony state until explicitly approved or dismissed. They survive `/clear` and show up on every build until resolved. No auto-expiry.
- **D-06:** Approved suggestions are written as pheromone signals using the existing `pheromone-write` command (with content hash dedup). Dismissed suggestions are recorded as dismissed and never re-shown.
- **D-07:** Deduplication uses exact match only -- same pheromone type + same content hash. This is already implemented in `pheromone-write` (content_hash + reinforcement logic).
- **D-08:** Suggestions that match expired pheromones are still shown -- the user might want to re-activate them. Only active pheromones suppress suggestions.
- **D-09:** Single Go CLI command (`aether suggest-analyze`) is the authoritative implementation. Claude Code and OpenCode wrappers call it and display the output. Codex calls it directly via the runtime.

### Claude's Discretion
- Exact change detection threshold (how many files changed triggers re-analysis)
- Number and content of build-specific extra patterns
- Visual rendering of the approval UI in each platform's wrapper
- Storage format for pending suggestions in colony state
- Whether to show a summary count ("3 new suggestions") or full details when re-displaying persisted unreviewed suggestions

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INTEL-01 | Suggest-analyze runs during build (Step 4.2) -- automatic pattern detection across codebase | `generatePheromoneSuggestions()` at cmd/init_research.go:1308-1596 provides 25 patterns; helper functions `hasFile`, `hasDir`, `fileContains`, `readFileContent` at lines 264-293; `detectGovernance()` at line 157; `classifyDirectory()` at line 297; build playbook Step 4.2 at build-context.md:147-181 currently DEPRECATED, needs restoration |
| INTEL-02 | Suggest-analyze deduplicates against existing pheromone signals | Content hash dedup already implemented in `pheromone-write` at cmd/pheromone_write.go:157-181; `pheromone-read` returns active signals at cmd/pheromones_read.go:27-32; content hash computed via SHA-256 at cmd/pheromone_write.go:87-88; PheromoneSignal struct has `ContentHash *string` field at pkg/colony/pheromones.go:31 |
| INTEL-03 | Suggest-approve provides tick-to-approve UI for reviewing suggestions | `suggest-approve` stub exists at cmd/compatibility_cmds.go:113-133 (returns empty array); `pheromone-write` command accepts --type, --content, --source, --reason flags for creating signals; ColonyState struct at pkg/colony/colony.go:175-204 needs new fields for pending suggestions storage |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Pattern detection (25 base + build extras) | Go runtime (CLI) | -- | CPU-bound filesystem scanning, must run in Go per platform policy |
| Change detection (git diff threshold) | Go runtime (CLI) | -- | Shell `git diff` invocation, part of the CLI command |
| Deduplication against active pheromones | Go runtime (CLI) | -- | Reads pheromones.json, compares content hashes -- data access layer |
| Suggestion persistence in colony state | Go runtime (CLI) | -- | Writes to COLONY_STATE.json -- state mutation owned by runtime |
| Pheromone creation on approval | Go runtime (CLI) | -- | Delegates to existing `pheromone-write` command |
| Approval UX display | Wrapper (Claude/OpenCode) | Go runtime (Codex) | Wrappers own presentation; Codex uses runtime renderer |
| Build ceremony Step 4.2 orchestration | Wrapper (build playbook) | Go runtime (Codex) | Playbooks call `aether suggest-analyze` and `aether suggest-approve` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | 1.21+ | os, filepath, strings, encoding/json, crypto/sha256, exec (for git diff) | Already used throughout codebase |
| cobra | current (go.mod) | CLI command framework | All 80+ subcommands use cobra |
| pkg/storage | internal | JSON persistence (COLONY_STATE.json, pheromones.json) | Established pattern across all commands |
| pkg/colony | internal | ColonyState, PheromoneSignal, PheromoneFile types | Type definitions for state and pheromones |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| git CLI | system | Change detection via `git diff --stat` | Trigger re-analysis check |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| git diff for change detection | Go git library (go-git) | go-git adds a dependency; the project has zero-new-deps principle (REQUIREMENTS.md: "All features use existing Go stdlib + cobra + pkg/storage") |
| New suggest-analyze file | Extend compatibility_cmds.go | New file is cleaner; compatibility_cmds.go is for backward-compat stubs |

**Installation:**
No new packages needed. All code uses existing dependencies.

**Version verification:** No external packages to verify.

## Architecture Patterns

### System Architecture Diagram

```
                    Build Ceremony (Step 4.2)
                              |
                    +---------v---------+
                    |  Wrapper calls:    |
                    |  aether            |
                    |  suggest-analyze   |
                    +---------+---------+
                              |
              +---------------+----------------+
              |                                |
     +--------v--------+            +---------v---------+
     | Change Detection |            | Pattern Detection  |
     | (git diff stat)  |            | (25 base + build  |
     | Skip if < N files|            |  extras)          |
     | or first build   |            | Uses: hasFile(),  |
     +--------+---------+            | hasDir(), etc.    |
              |                      +---------+---------+
              |                                |
              +---------------+----------------+
                              |
                    +---------v---------+
                    |   Deduplication    |
                    | Load pheromones.json|
                    | Filter out active  |
                    | signals matching   |
                    | type + content_hash|
                    +---------+---------+
                              |
                    +---------v---------+
                    | Persist Pending   |
                    | suggestions to    |
                    | COLONY_STATE.json |
                    +---------+---------+
                              |
                    +---------v---------+
                    | Display to User   |
                    | (tick-to-approve) |
                    +---------+---------+
                              |
                   +----------+----------+
                   |                     |
          +--------v------+      +------v--------+
          |   Approve     |      |   Dismiss     |
          | pheromone-    |      | Record as     |
          | write --type  |      | dismissed in  |
          | --content     |      | colony state  |
          +---------------+      +---------------+
```

### Recommended Project Structure
```
cmd/
  suggest_analyze.go          # New: aether suggest-analyze command
  suggest_analyze_test.go     # New: tests for suggest-analyze
  suggest_approve.go          # New: real suggest-approve (replaces stub)
  suggest_approve_test.go     # New: tests for suggest-approve
  compatibility_cmds.go       # Existing: remove suggestApproveCmd stub

pkg/colony/
  colony.go                   # Existing: add PendingSuggestions field to ColonyState
```

### Pattern 1: CLI Command with outputOK envelope
**What:** All Aether CLI commands return results via `outputOK()` which wraps data in a JSON envelope `{"ok": true, "result": {...}}`. Errors use `outputError()`.
**When to use:** Every new CLI command.
**Example:**
```go
// Source: cmd/pheromone_write.go:211-216
outputOK(map[string]interface{}{
    "created":  true,
    "signal":   signal,
    "total":    len(pf.Signals),
    "replaced": replaced,
})
```

### Pattern 2: Load and save colony state
**What:** Commands that modify colony state use `loadActiveColonyState()` to read and `store.SaveJSON("COLONY_STATE.json", cs)` to write.
**When to use:** Any command that reads or writes COLONY_STATE.json.
**Example:**
```go
// Source: cmd/pheromone_write.go:114-117
var cs colony.ColonyState
if loadErr := store.LoadJSON("COLONY_STATE.json", &cs); loadErr == nil && cs.CurrentPhase > 0 {
    signal.SourcePhase = &cs.CurrentPhase
}
```

### Pattern 3: Test helper with newTestStore
**What:** Tests use `newTestStore(t)` to create a temp directory with a storage.Store, and `saveGlobals(t)` / `resetRootCmd(t)` to isolate global state.
**When to use:** Every test file.
**Example:**
```go
// Source: cmd/write_cmds_test.go:141-143
s, tmpDir := newTestStore(t)
defer os.RemoveAll(tmpDir)
store = s
```

### Pattern 4: Pheromone content hash dedup
**What:** Pheromone signals are deduplicated by computing SHA-256 of the raw content and comparing against active signals with the same type + content_hash. Duplicates reinforce instead of appending.
**When to use:** When checking if a suggestion already exists as an active pheromone.
**Example:**
```go
// Source: cmd/pheromone_write.go:157-181
for i := range pf.Signals {
    sig := &pf.Signals[i]
    if !sig.Active { continue }
    if sig.Type == sigType && sig.ContentHash != nil && *sig.ContentHash == contentHash {
        // Reinforce existing signal
        sig.ReinforcementCount = ...
        replaced = true
        break
    }
}
```

### Anti-Patterns to Avoid
- **Don't put suggestion patterns in wrapper markdown:** The Go runtime owns all pattern detection logic per the wrapper-runtime contract.
- **Don't create a new data file for suggestions:** Store pending suggestions in COLONY_STATE.json alongside other colony state. A separate file adds unnecessary I/O complexity.
- **Don't bypass pheromone-write for approved suggestions:** Always pipe through `pheromone-write` to get content hash computation, sanitization, and reinforcement logic for free.
- **Don't auto-approve suggestions:** The user must explicitly approve each one (D-04). Init ceremony auto-approves, but build-time suggestions require user review.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Content hash computation | Custom hash function | `sha256Sum()` from cmd/pheromone_write.go:350-353 | Already exists, tested, consistent with pheromone dedup |
| Signal sanitization | Custom validation | `colony.SanitizeSignalContent()` from pkg/colony/sanitize.go | Handles prompt injection, XML tags, length limits |
| Pheromone persistence | Direct JSON file writes | `pheromone-write` CLI command | Handles dedup, reinforcement, TTL, event emission |
| Pheromone signal ID generation | Custom ID format | `generateSignalID()` from cmd/pheromone_write.go:356-360 | Consistent format: sig_<timestamp>_<random> |
| File existence checks | os.Stat wrappers | `hasFile()`, `hasDir()`, `fileContains()` from cmd/init_research.go:264-293 | Already used by all 25 patterns |
| Governance detection | Custom file scanning | `detectGovernance()` from cmd/init_research.go:157 | Parses linters, formatters, test frameworks, CI configs, build tools |
| Directory classification | Custom heuristics | `classifyDirectory()` from cmd/init_research.go:297 | Detects monorepo, microservices, standard app, library |
| Tech stack detection | Custom file parsing | `parseTechStack()` from cmd/init_research.go (line ~1240+) | Parses package.json, go.mod, Cargo.toml, pyproject.toml, etc. |

**Key insight:** The most complex part of this phase (pattern detection) is already fully implemented. The new work is orchestration: change detection, suggestion persistence, dedup filtering, and approval flow.

## Common Pitfalls

### Pitfall 1: Importing init_research.go functions from a new file
**What goes wrong:** `generatePheromoneSuggestions()` and its helper types (`pheromoneSuggestion`, `governanceInfo`, `dirClassification`, `techStackDetail`) are defined in `cmd/init_research.go`. A new `cmd/suggest_analyze.go` file in the same package can access them directly (same package = `cmd`). No import needed.
**Why it happens:** Developers assume they need to create a separate package or import path.
**How to avoid:** Keep `cmd/suggest_analyze.go` in the `cmd` package. All the types and functions are package-private or package-level in `cmd`.
**Warning signs:** Compilation errors about unexported types if trying to import from another package.

### Pitfall 2: Running full init-research just to get pattern inputs
**What goes wrong:** Calling the full `init-research` command (which scans the entire codebase, parses all dependencies, etc.) just to get the governance info and dir classification needed for `generatePheromoneSuggestions()` is slow and wasteful.
**Why it happens:** `generatePheromoneSuggestions()` takes `governanceInfo`, `dirClassification`, and `[]techStackDetail` as inputs -- these come from the full research scan.
**How to avoid:** Call `detectGovernance(target)`, `classifyDirectory(target)`, and `parseTechStack()` directly. These are lightweight functions that only scan specific files, not the entire codebase. For build-specific extras (TODO density, large files), write targeted scanners that only look at what they need.
**Warning signs:** `suggest-analyze` taking > 2 seconds on a normal codebase.

### Pitfall 3: Blocking the build when suggest-analyze fails
**What goes wrong:** If pattern detection throws an error, the entire build stops.
**Why it happens:** The build playbook has error handling rules but the new command might not follow them.
**How to avoid:** The existing Step 4.2 playbook already says "Non-blocking: This step never stops the build." The `suggest-analyze` command should return `ok: true` with `suggestions: []` on any error, never `ok: false`.
**Warning signs:** Build ceremony stops at Step 4.2.

### Pitfall 4: Race condition on COLONY_STATE.json writes
**What goes wrong:** Multiple commands writing to COLONY_STATE.json simultaneously (e.g., suggest-analyze writing pending suggestions while another command updates phase state).
**Why it happens:** No file locking on colony state writes in the current codebase.
**How to avoid:** Use the existing pattern: load -> modify -> save in a single operation. The storage package's `SaveJSON` does atomic writes (write to temp, rename). For extra safety, the suggest-analyze command should load the latest state, append/modify suggestions, and save immediately.
**Warning signs:** Suggestions disappearing or colony state corruption after concurrent builds.

### Pitfall 5: Forgetting to handle the "no colony" case
**What goes wrong:** `suggest-analyze` is called outside of an active colony (no COLONY_STATE.json).
**Why it happens:** The build playbook runs suggest-analyze as part of the build ceremony, but someone might call it standalone.
**How to avoid:** Check for colony state early. If no colony is active, either skip silently or return an informative message. The existing `suggest-approve` stub already calls `loadActiveColonyState()` and errors if it fails.
**Warning signs:** nil pointer dereference on colony state access.

## Code Examples

Verified patterns from existing codebase:

### Reusing generatePheromoneSuggestions (in same package)
```go
// Source: cmd/init_research.go:50-54 (type definition), 1308-1310 (function signature)
// In cmd/suggest_analyze.go (same package "cmd"), direct access:

target := "." // or resolved from flags
gov := detectGovernance(target)
dirClass := classifyDirectory(target)
techStack := parseTechStack(target) // need to verify this function name

suggestions := generatePheromoneSuggestions(target, gov, dirClass, techStack)
```

### Computing content hash for dedup (same approach as pheromone-write)
```go
// Source: cmd/pheromone_write.go:87-88
h := sha256Sum(content)
contentHash := "sha256:" + h
```

### Loading active pheromones for dedup check
```go
// Source: cmd/pheromones_read.go:17-35 (pattern)
var pf colony.PheromoneFile
if err := store.LoadJSON("pheromones.json", &pf); err != nil {
    // No pheromones file -- no dedup needed
}
var activeSignals []colony.PheromoneSignal
for _, sig := range pf.Signals {
    if sig.Active {
        activeSignals = append(activeSignals, sig)
    }
}
```

### Checking if a suggestion matches an active pheromone
```go
// Source: cmd/pheromone_write.go:157-165 (dedup logic, adapted)
func isActivePheromone(activeSignals []colony.PheromoneSignal, sigType, content string) bool {
    hash := "sha256:" + sha256Sum(content)
    for _, sig := range activeSignals {
        if sig.Type == sigType && sig.ContentHash != nil && *sig.ContentHash == hash {
            return true
        }
    }
    return false
}
```

### Writing approved suggestion as pheromone
```go
// Source: cmd/pheromone_write.go:30-36 (flag pattern)
// Best approach: call pheromone-write programmatically via cobra
// or replicate the logic inline (simpler, same package)
// The key is to use the existing content_hash + dedup flow
```

### Git diff for change detection
```go
// Assumption: use os/exec to run git diff --stat
// Context: D-01 specifies "git diff (files changed since last analysis timestamp or commit)"
out, err := exec.Command("git", "diff", "--stat", lastAnalyzedCommit).Output()
// Parse output to count changed files
// If count >= threshold, re-run analysis
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell script suggest-analyze | Go CLI command (this phase) | Phase 74 | Consistent with all other commands being Go-native |
| suggest-approve stub returning empty array | Real implementation with persistence | Phase 74 | Users can actually review suggestions |
| Step 4.2 DEPRECATED in build playbook | Restored with real suggest-analyze call | Phase 74 | Build ceremony regains intelligence features |
| No suggestion persistence | COLONY_STATE.json pending suggestions | Phase 74 | Suggestions survive /clear, shown until resolved |

**Deprecated/outdated:**
- `suggestApproveCmd` stub in `cmd/compatibility_cmds.go:113-133`: Returns `{"suggestions": [], "dry_run": false}`. Must be replaced with real implementation.
- Step 4.2 in `build-context.md:147-181`: Currently marked DEPRECATED with skip logic. Must be restored to call the real `suggest-analyze` and `suggest-approve` commands.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `generatePheromoneSuggestions()` and helper functions are accessible from new files in the `cmd` package (same package, unexported types accessible) | Architecture Patterns | Low risk -- verified by Go package rules, all files in `cmd/` are package `cmd` |
| A2 | `parseTechStack()` is a standalone function callable independently of the full init-research command | Code Examples | Medium risk -- function name needs verification; may be named differently or inline in RunE |
| A3 | Git CLI is available on all platforms where Aether runs | Change Detection | Low risk -- Aether already requires git for other features |
| A4 | The existing 4 pre-existing test failures (TestContinueEmitsLifecycleCeremonyEvents, TestContinueBlocksWhenWatcherUsesFakeInvoker, TestClaudeOpenCodeCommandParity, TestLifecycleCommandDocsPreferRuntimeCLI) are unrelated to this phase | Validation Architecture | Low risk -- verified by test names being about continue flow and command parity, not suggest-analyze |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions (RESOLVED)

1. **Tech stack parsing function name**
   - What we know: `generatePheromoneSuggestions()` takes `[]techStackDetail` as input. The init-research command builds this by parsing dependency files.
   - What's unclear: Whether there's a standalone `parseTechStack(target string) []techStackDetail` function or if the parsing is inline in the `initResearchCmd.RunE`.
   - Recommendation: Extract or call the tech stack parsing logic from `init_research.go`. If it's inline, create a shared function during implementation.

2. **Exact change detection threshold**
   - What we know: D-01 says "significant change" triggers re-analysis. Claude's discretion.
   - What's unclear: What threshold is appropriate (5 files? 10? 20? percentage of codebase?).
   - Recommendation: Start with 5 changed files as the threshold. This is conservative enough to avoid unnecessary re-analysis but sensitive enough to catch meaningful changes. The threshold can be tuned based on user feedback.

3. **Build-specific extra patterns content**
   - What we know: D-02 lists TODO/FIXME density, test coverage gaps, large files as examples. Claude's discretion.
   - What's unclear: Exact patterns and their thresholds.
   - Recommendation: Implement 4-6 patterns: (1) TODO/FIXME count above threshold, (2) files > 500 lines, (3) directories without test files, (4) dependency count increase since last analysis, (5) new files added without corresponding tests. Keep each pattern simple and deterministic.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all work uses existing Go stdlib + cobra + pkg/storage + system git CLI which Aether already requires).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing`) |
| Config file | none |
| Quick run command | `go test ./cmd/... -run "TestSuggest" -count=1` |
| Full suite command | `go test ./cmd/... -count=1 -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INTEL-01 | suggest-analyze runs pattern detection | unit | `go test ./cmd/... -run "TestSuggestAnalyze" -count=1` | No -- Wave 0 |
| INTEL-01 | suggest-analyze detects build-specific patterns (TODO density, large files) | unit | `go test ./cmd/... -run "TestSuggestAnalyzeBuildPatterns" -count=1` | No -- Wave 0 |
| INTEL-01 | suggest-analyze skips on --no-suggest | unit | `go test ./cmd/... -run "TestSuggestAnalyzeSkip" -count=1` | No -- Wave 0 |
| INTEL-01 | suggest-analyze runs change detection | unit | `go test ./cmd/... -run "TestSuggestAnalyzeChangeDetection" -count=1` | No -- Wave 0 |
| INTEL-02 | dedup filters out suggestions matching active pheromones | unit | `go test ./cmd/... -run "TestSuggestDedup" -count=1` | No -- Wave 0 |
| INTEL-02 | expired pheromones don't suppress suggestions | unit | `go test ./cmd/... -run "TestSuggestDedupExpired" -count=1` | No -- Wave 0 |
| INTEL-03 | suggest-approve lists pending suggestions | unit | `go test ./cmd/... -run "TestSuggestApprove" -count=1` | No -- Wave 0 |
| INTEL-03 | approve writes pheromone via pheromone-write | unit | `go test ./cmd/... -run "TestSuggestApproveAccept" -count=1` | No -- Wave 0 |
| INTEL-03 | dismiss records suggestion as dismissed | unit | `go test ./cmd/... -run "TestSuggestApproveDismiss" -count=1` | No -- Wave 0 |
| INTEL-03 | dismissed suggestions don't reappear | unit | `go test ./cmd/... -run "TestSuggestApproveDismissPersist" -count=1` | No -- Wave 0 |
| INTEL-05 | pending suggestions survive colony state reload | unit | `go test ./cmd/... -run "TestSuggestPersist" -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run "TestSuggest" -count=1`
- **Per wave merge:** `go test ./cmd/... -count=1 -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/suggest_analyze_test.go` -- tests for INTEL-01 (pattern detection, change detection, --no-suggest skip)
- [ ] `cmd/suggest_approve_test.go` -- tests for INTEL-02 and INTEL-03 (dedup, approve, dismiss, persistence)
- [ ] Framework install: none needed -- Go stdlib testing already in use

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | yes | `colony.SanitizeSignalContent()` for all suggestion content before storage; existing prompt injection and shell injection detection |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go CLI + JSON storage

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt injection via suggestion content | Tampering | `SanitizeSignalContent()` blocks XML tags, prompt injection patterns, shell injection; enforced in `pheromone-write` |
| Path traversal in target directory | Tampering | Target resolved relative to current directory; no user-controlled path input for suggest-analyze |
| JSON injection in colony state | Tampering | `encoding/json` handles escaping; content stored as structured fields, not raw strings |

## Sources

### Primary (HIGH confidence)
- [Codebase] `cmd/init_research.go:50-54, 1308-1596` -- pheromoneSuggestion struct and generatePheromoneSuggestions() function with 25 patterns
- [Codebase] `cmd/pheromone_write.go:20-219` -- pheromone-write command with content hash dedup logic
- [Codebase] `cmd/pheromones_read.go:8-42` -- pheromone-read command showing active signal filtering
- [Codebase] `pkg/colony/pheromones.go:17-44` -- PheromoneSignal and PheromoneFile struct definitions
- [Codebase] `pkg/colony/colony.go:175-204` -- ColonyState struct definition
- [Codebase] `cmd/compatibility_cmds.go:113-133` -- current suggest-approve stub
- [Codebase] `.aether/docs/command-playbooks/build-context.md:147-181` -- Step 4.2 (currently DEPRECATED)
- [Codebase] `.aether/docs/command-playbooks/build-prep.md:118-137` -- --no-suggest flag parsing
- [Codebase] `.aether/docs/wrapper-runtime-ux-contract.md` -- Go runtime owns state mutations

### Secondary (MEDIUM confidence)
- [Codebase] `cmd/pheromone_write_test.go` -- tests proving dedup behavior
- [Codebase] `cmd/write_cmds_test.go:147-178` -- test proving duplicate content reinforces

### Tertiary (LOW confidence)
- None -- all findings are from direct codebase inspection.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all existing Go codebase, no new dependencies
- Architecture: HIGH - pattern detection already exists, integration points are well-defined
- Pitfalls: MEDIUM - identified from codebase analysis but some depend on implementation choices

**Research date:** 2026-04-29
**Valid until:** 60 days (stable codebase, no external dependency changes expected)
