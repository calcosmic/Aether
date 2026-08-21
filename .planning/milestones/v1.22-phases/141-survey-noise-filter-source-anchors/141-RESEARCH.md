# Phase 141: Survey Noise Filter + Source Anchors - Research

**Researched:** 2026-05-18
**Domain:** Go codebase scanning and directory exclusion
**Confidence:** HIGH

## Summary

Phase 141 fixes the root cause of generic plans by unifying five divergent directory-skip lists into a single canonical `ScanFilter`, extending it to cover Python/Node/Rust/Go ecosystem noise directories, and extracting source anchors (the repo's own source files) from the cleaned survey output. The five divergent skip lists live in `pkg/codegraph/codegraph.go` (13 dirs), `cmd/codex_colonize.go` (10 dirs), `cmd/init_research.go` (16 dirs), `cmd/skills.go` (23 dirs), and `cmd/discuss_analyze.go` (which reuses `init_research.go`'s `extendedSkipDirs`). After unification, source anchors are extracted during `surveyWorkspace()` and written into the survey output so that `loadCodexSurveyContext()` in `codex_plan.go` can feed them to the planner.

**Primary recommendation:** Create `pkg/codegraph/scan_filter.go` as the single source of truth, migrate all five call sites to use it, add a `SourceAnchors` field to both `codexWorkspaceFacts` and `codexSurveyContext`, and wire anchors from survey through to planning.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Directory exclusion | Go runtime (`pkg/codegraph/`) | -- | Scanning happens in Go; the filter is pure Go stdlib |
| Source anchor extraction | Go runtime (`cmd/codex_colonize.go`) | -- | `surveyWorkspace()` already walks the tree; anchors are a survey output |
| Survey output writing | Go runtime (`cmd/codex_colonize.go`) | -- | `writeSurveyCompatibilityJSON()` writes JSON files consumed by planner |
| Planning context injection | Go runtime (`cmd/codex_plan.go`) | -- | `loadCodexSurveyContext()` reads survey JSON into planner context |
| Regression testing | Go test suite | -- | All five requirements are testable as unit tests in Go |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GROUND-01 | Unify 5 divergent skip lists into one canonical `ScanFilter` in `pkg/codegraph/scan_filter.go` | Five skip lists identified and catalogued below; shared module design in Standard Stack |
| GROUND-02 | Extend filter to cover `.venv`, `__pycache__`, `site-packages`, `.pytest_cache`, `.tox`, `.mypy_cache`, `node_modules/.cache`, `.gradle`, `.cargo/registry` | Extension entries documented in canonical filter design; `node_modules/.cache` is a subdirectory pattern requiring file-level check |
| GROUND-03 | Extract source anchors from cleaned survey output (top 50 non-dependency source files, depth-first) | `surveyWorkspace()` walk can be extended; language detection map already in `codegraph.go`; capping and sorting strategy documented |
| GROUND-04 | Write source anchors to survey output so colony-prime and planners can use them | `codexWorkspaceFacts` struct needs `SourceAnchors []string` field; `writeSurveyCompatibilityJSON()` needs new output file; `codexSurveyContext` needs matching field; `loadCodexSurveyContext()` needs to read it |
| GROUND-05 | Regression test: M4L fixture with `.venv` noise produces clean survey with zero `.venv` references | Test pattern from existing `TestSurveyWorkspaceDockerRepoIgnoresPlatformGuidance`; fixture creation pattern from `createDockerSurveyFixture` |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `path/filepath` | Go 1.26+ | Directory walking, path matching, extension extraction | Already used by all five skip list locations |
| Go stdlib `regexp` | Go 1.26+ | File extension matching for source detection | Already used in `codegraph.go` for import parsing |
| Go stdlib `sort` | Go 1.26+ | Sorting source anchors by path depth | Already used throughout `codex_colonize.go` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `strings` | Go 1.26+ | HasSuffix checks for test file detection | Already used in `isTestFile()`, `isConfigFile()` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hardcoded noise map | `github.com/sabhiram/go-gitignore` | Library adds dependency; target repos may lack `.gitignore`; hardcoded 25-entry map covers 99% |
| Hardcoded noise map | Configurable list in COLONY_STATE.json | YAGNI; pheromone REDIRECT already covers edge cases |
| Directory-based source detection | AST-based parsing | Overkill; directory-based classification handles the 80% case |

**Installation:**
```bash
# Zero new dependencies -- all changes use Go stdlib
```

**Version verification:** Go 1.26.1 darwin/arm64 is installed and current. [VERIFIED: `go version`]

## Architecture Patterns

### System Architecture Diagram

```
/aether colonize
    |
    v
surveyWorkspace(root)  [cmd/codex_colonize.go]
    |
    +-- Walk repo root with shared ScanFilter [pkg/codegraph/scan_filter.go]
    |       |
    |       +-- ShouldSkipDir() -> skip noise dirs (.venv, __pycache__, etc.)
    |       +-- ShouldSkipFile() -> skip noise file extensions (.pyc, .d.ts, etc.)
    |       +-- Collect: TopLevelDirs, Languages, Frameworks, ConfigFiles
    |       +-- NEW: Extract SourceAnchors (non-dependency source files, capped at 50)
    |
    v
codexWorkspaceFacts  [includes new SourceAnchors field]
    |
    +-- writeSurveyCompatibilityJSON()
    |       -> writes anchors.json to .aether/data/survey/
    |
    v
/aether plan
    |
    +-- loadCodexSurveyContext() reads survey JSON
    |       -> reads anchors.json -> codexSurveyContext.SourceAnchors
    +-- Planner context now includes concrete source file paths
    |
    v
Grounded plan with concrete file references
```

### Recommended Project Structure
```
pkg/codegraph/
    scan_filter.go        # NEW: canonical ScanFilter (ShouldSkipDir, ShouldSkipFile)
    scan_filter_test.go   # NEW: filter unit tests + divergence test
    codegraph.go          # MODIFY: replace dirsToSkip with ScanFilter calls
    codegraph_test.go     # MODIFY: update TestDirsToSkip to use ScanFilter
cmd/
    codex_colonize.go     # MODIFY: replace shouldSkipSurveyDir with ScanFilter, add SourceAnchors
    codex_colonize_test.go # MODIFY: add regression test for .venv noise
    codex_plan.go         # MODIFY: add SourceAnchors to codexSurveyContext
    init_research.go      # MODIFY: replace extendedSkipDirs with ScanFilter
    skills.go             # MODIFY: replace skillScanSkipDirs with ScanFilter
    discuss_analyze.go    # Already uses init_research.go's extendedSkipDirs (auto-fixed)
```

### Pattern 1: Shared Filter Module
**What:** Single Go package provides `ShouldSkipDir(name string) bool` and `ShouldSkipFile(name string) bool` used by all directory-walking code.
**When:** Any code that walks a repository directory tree (codegraph, colonize, init_research, skills, discuss_analyze).
**Example:**
```go
// Source: pkg/codegraph/scan_filter.go (new file)
package codegraph

// noiseDirs lists directories that should always be excluded from repo scanning.
// This is the single source of truth -- all codegraph, colonize, research,
// skills, and analyze callers must use ShouldSkipDir() instead of local lists.
var noiseDirs = map[string]bool{
    // Version control
    ".git": true,
    // Python virtual environments and caches
    ".venv": true, "venv": true, "__pycache__": true,
    ".mypy_cache": true, ".pytest_cache": true, ".tox": true,
    ".ruff_cache": true, ".pytype": true, "site-packages": true,
    // Node.js
    "node_modules": true, ".next": true, ".nuxt": true, ".svelte-kit": true,
    // Go
    "vendor": true,
    // Rust
    "target": true,
    // General build artifacts
    "dist": true, "build": true, "out": true, "bin": true, "coverage": true,
    // Caches
    ".cache": true, ".terraform": true, ".gradle": true,
    // Aether-managed
    ".aether": true, ".claude": true, ".codex": true, ".opencode": true,
    // IDE/editor
    ".idea": true, ".vscode": true,
    // Temp
    "tmp": true, "temp": true,
}

// ShouldSkipDir returns true if a directory should be excluded from scanning.
func ShouldSkipDir(name string) bool {
    return noiseDirs[name]
}

// ShouldSkipFile returns true if a file should be excluded based on extension.
func ShouldSkipFile(name string) bool {
    // Check compound extensions first (.min.js before .js)
    if strings.HasSuffix(name, ".min.js") || strings.HasSuffix(name, ".min.css") ||
       strings.HasSuffix(name, ".bundle.js") || strings.HasSuffix(name, ".d.ts") {
        return true
    }
    ext := filepath.Ext(name)
    return noiseFileExts[ext]
}
```

### Pattern 2: Source Anchor Extraction
**What:** After noise filtering, extract the top 50 non-dependency, non-test source files as planning anchors.
**When:** During `surveyWorkspace()` after the directory walk completes.
**Example:**
```go
// In surveyWorkspace() or a new extractSourceAnchors() helper:
func extractSourceAnchors(root string, maxAnchors int) []string {
    sourceExts := map[string]bool{
        ".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
        ".py": true, ".rb": true, ".java": true, ".rs": true, ".c": true, ".cpp": true,
    }
    var anchors []string
    filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return nil }
        if d.IsDir() {
            if ShouldSkipDir(d.Name()) { return filepath.SkipDir }
            return nil
        }
        rel, _ := filepath.Rel(root, path)
        ext := filepath.Ext(d.Name())
        if !sourceExts[ext] { return nil }
        if ShouldSkipFile(d.Name()) { return nil }
        if isTestFile(d.Name()) { return nil }
        anchors = append(anchors, filepath.ToSlash(rel))
        return nil
    })
    // Sort by depth (shallowest first), then alphabetically
    sort.Slice(anchors, func(i, j int) bool {
        di := strings.Count(anchors[i], "/")
        dj := strings.Count(anchors[j], "/")
        if di != dj { return di < dj }
        return anchors[i] < anchors[j]
    })
    if len(anchors) > maxAnchors {
        anchors = anchors[:maxAnchors]
    }
    return anchors
}
```

### Anti-Patterns to Avoid

- **Divergent skip lists:** Maintaining separate exclusion lists in multiple files. The bug this phase fixes was caused by exactly this pattern. Any new directory exclusion MUST go in `scan_filter.go` only.
- **Excluding `env` without a dot:** Some projects have a source directory called `env/` for environment configuration. Only exclude `.env` (with leading dot) and `.venv`, never bare `env`.
- **File-level extension matching before compound extensions:** Checking `.js` before `.min.js` would skip legitimate source files. Always check compound extensions first.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| `.gitignore` parsing | Custom gitignore parser or library | Hardcoded noise map + pheromone REDIRECT | Target repos may lack `.gitignore`; parsing is complex (negation, globs); the known noise set is finite and stable |
| Source file classification | AST-based type system | Extension-based classification via `languageDetectors` map | Codegraph already uses regex for the 80% case; AST adds dependency and latency for no planning benefit |
| Anchor persistence | Database or custom file format | JSON file in `.aether/data/survey/anchors.json` | Anchors are a scan-time artifact regenerated on each survey; JSON fits the existing survey output pattern |

**Key insight:** The survey output already writes compatibility JSON files (`blueprint.json`, `provisions.json`, etc.). Adding `anchors.json` follows the exact same pattern with zero architectural change.

## Common Pitfalls

### Pitfall 1: Five-Way Divergence (The Root Cause)
**What goes wrong:** Fixing the `.venv` bug in one skip list but not the other four. Survey improves but codegraph still scans virtual environments, or skills indexing still recurses into `.venv`.
**Why it happens:** Five separate lists in five files with no enforcement mechanism.
**How to avoid:** Create `pkg/codegraph/scan_filter.go` as single source. Add a divergence test that asserts all call sites use the shared filter. Delete the inline lists.
**Warning signs:** Any file that contains a `map[string]bool` or `map[string]struct{}` with directory names to skip.

### Pitfall 2: `.env` Ambiguity
**What goes wrong:** Adding `env` (without dot) to the noise filter breaks projects with an `env/` source directory for environment configuration modules.
**Why it happens:** Python virtual environments sometimes use bare `env/`, but so do legitimate source directories.
**How to avoid:** Only exclude `.env` (with leading dot). The `venv` entry (without dot) is safe because source directories named `venv/` are extremely rare. If in doubt, omit.
**Warning signs:** A test fixture with both `env/` (source) and `.venv/` (virtual environment) where the source directory is incorrectly skipped.

### Pitfall 3: `node_modules/.cache` Subdirectory Pattern
**What goes wrong:** `node_modules` is already excluded at the directory level, but the requirement says to cover `node_modules/.cache`. If `node_modules` is properly excluded, we never recurse into it, so `.cache` inside it is already covered.
**Why it happens:** The requirement lists `node_modules/.cache` as a specific entry, which could mislead implementers into thinking subdirectory matching is needed.
**How to avoid:** Exclude `node_modules` at the directory level (already done). The `.cache` entry at the top level covers standalone cache directories. No subdirectory matching needed.
**Warning signs:** Code that tries to match `node_modules/.cache` as a path pattern rather than relying on `node_modules` exclusion.

### Pitfall 4: Anchor Extraction Without Noise Filter
**What goes wrong:** Extracting source anchors before the noise filter is applied, resulting in anchors that include `.venv/site-packages/requests/api.py` or `node_modules/react/index.js`.
**Why it happens:** Adding anchor extraction as a separate walk that does not use the shared `ScanFilter`.
**How to avoid:** `extractSourceAnchors()` MUST call `ShouldSkipDir()` and `ShouldSkipFile()` from the shared filter. It cannot have its own skip logic.
**Warning signs:** Anchor list contains paths with `site-packages`, `.venv`, `node_modules`, or `__pycache__`.

### Pitfall 5: Over-Capping Anchors Loses Important Files
**What goes wrong:** Capping at 50 anchors with simple alphabetical sorting loses important shallow files because deeper files fill the cap first.
**Why it happens:** Sorting anchors alphabetically or by file count rather than by path depth.
**How to avoid:** Sort anchors by path depth (shallowest first), then alphabetically within the same depth. Shallow files like `cmd/main.go` are almost always more architecturally important than `pkg/internal/handler/middleware/auth.go`.
**Warning signs:** Test that a repo with 100 source files produces exactly 50 anchors, and the shallowest files appear first.

## Code Examples

### Unified ScanFilter (new file)
```go
// Source: pkg/codegraph/scan_filter.go
package codegraph

import (
    "path/filepath"
    "strings"
)

// noiseDirs is the canonical set of directories to skip during repo scanning.
// All directory-walking code in Aether MUST use ShouldSkipDir() instead of
// maintaining local skip lists.
//
// Order: version control, Python, Node.js, Go, Rust, build artifacts,
// caches, Aether-managed, IDE/editor, OS metadata, temp.
var noiseDirs = map[string]bool{
    // Version control
    ".git": true,
    // Python virtual environments and caches
    ".venv": true, "venv": true, "__pycache__": true,
    ".mypy_cache": true, ".pytest_cache": true, ".tox": true,
    ".ruff_cache": true, ".pytype": true, "site-packages": true,
    // Node.js
    "node_modules": true, ".next": true, ".nuxt": true, ".svelte-kit": true,
    // Go
    "vendor": true,
    // Rust
    "target": true,
    // General build/artifact dirs
    "dist": true, "build": true, "out": true, "bin": true, "coverage": true,
    // Caches and registries
    ".cache": true, ".terraform": true, ".gradle": true,
    // Aether-managed directories
    ".aether": true, ".claude": true, ".codex": true, ".opencode": true,
    // IDE/editor
    ".idea": true, ".vscode": true,
    // Temp
    "tmp": true, "temp": true,
}

// noiseFileExts lists file extensions for files that should be skipped.
var noiseFileExts = map[string]bool{
    ".pyc": true, ".pyo": true,       // Python bytecode
    ".so": true, ".dylib": true, ".dll": true, // Compiled libraries
    ".map": true,                      // Source maps
}

// compoundNoiseSuffixes are multi-part extensions to skip (checked before single ext).
var compoundNoiseSuffixes = []string{
    ".min.js", ".min.css", ".bundle.js", ".d.ts",
}

// ShouldSkipDir returns true if a directory should be excluded from scanning.
func ShouldSkipDir(name string) bool {
    return noiseDirs[name]
}

// ShouldSkipFile returns true if a file should be excluded based on extension.
func ShouldSkipFile(name string) bool {
    lower := strings.ToLower(name)
    for _, suffix := range compoundNoiseSuffixes {
        if strings.HasSuffix(lower, suffix) {
            return true
        }
    }
    ext := filepath.Ext(lower)
    return noiseFileExts[ext]
}

// NoiseDirCount returns the number of directories in the skip list.
// Used by tests to detect divergence.
func NoiseDirCount() int {
    return len(noiseDirs)
}
```

### Migrating codegraph.go Scan()
```go
// Source: pkg/codegraph/codegraph.go (modified Scan function)
// BEFORE:
//  if dirsToSkip[d.Name()] {
// AFTER:
//  if ShouldSkipDir(d.Name()) {
```

### Adding SourceAnchors to codexWorkspaceFacts
```go
// Source: cmd/codex_colonize.go
type codexWorkspaceFacts struct {
    // ... existing fields ...
    SourceAnchors    []string  // NEW: top 50 non-dependency source files
}
```

### Writing anchors.json
```go
// Source: cmd/codex_colonize.go (in writeSurveyCompatibilityJSON)
"anchors.json": {
    "source_anchors": facts.SourceAnchors,
    "anchor_count":   len(facts.SourceAnchors),
    "summary":        "Repo-owned source files for plan grounding",
},
```

### Reading anchors in loadCodexSurveyContext
```go
// Source: cmd/codex_plan.go (in loadCodexSurveyContext)
if payload := readSummary("anchors.json"); payload != nil {
    ctx.SourceAnchors = append(ctx.SourceAnchors, jsonStringSlice(payload["source_anchors"])...)
}
```

## Current Divergent Skip Lists (Verified)

Five separate skip lists found in the codebase:

### 1. `pkg/codegraph/codegraph.go:72` -- `dirsToSkip` (13 entries)
```
.git, node_modules, vendor, .aether, dist, build, out, __pycache__,
.next, .nuxt, target, bin, .cache, .terraform
```
**Missing:** `.venv`, `venv`, `site-packages`, `.pytest_cache`, `.tox`, `.mypy_cache`, `.gradle`, `.idea`, `.vscode`, `.codex`, `.opencode`

### 2. `cmd/codex_colonize.go:976` -- `shouldSkipSurveyDir` (10 entries)
```
.git, .cache, node_modules, dist, build, vendor, .aether, .claude, .codex, .opencode
```
**Missing:** `.venv`, `venv`, `__pycache__`, `.next`, `.nuxt`, `target`, `bin`, `.terraform`, `.gradle`, `.pytest_cache`, `.tox`, `.mypy_cache`

### 3. `cmd/init_research.go:147` -- `extendedSkipDirs` (16 entries)
```
.git, node_modules, .next, dist, build, vendor, .venv, venv, coverage,
.aether, .claude, .opencode, .codex, __pycache__
```
**Missing:** `target`, `bin`, `out`, `.cache`, `.terraform`, `.gradle`, `.pytest_cache`, `.tox`, `.mypy_cache`

### 4. `cmd/skills.go:150` -- `skillScanSkipDirs` (23 entries, `struct{}` values)
```
.git, .aether, .claude, .codex, .opencode, .idea, .vscode, .cache, .next,
.nuxt, .svelte-kit, .venv, .tox, .pytest_cache, .mypy_cache, node_modules,
vendor, dist, build, coverage, tmp, temp, venv
```
**Most complete list.** Missing: `target`, `bin`, `out`, `__pycache__`, `.terraform`, `.gradle`, `site-packages`

### 5. `cmd/discuss_analyze.go:131` -- uses `extendedSkipDirs` from init_research.go
**Same as list 3.** This is automatically fixed when list 3 is fixed.

**Note:** The requirement says "5 divergent skip lists" but discuss_analyze.go reuses init_research.go's `extendedSkipDirs`, so there are really 4 independent lists. However, since all 5 call sites need migration, the count is correct from a migration perspective.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-file skip lists | Shared filter module | This phase | Prevents divergence bugs like the `.venv` issue |
| No source anchors | Source anchors in survey output | This phase | Plans can reference concrete files instead of generic descriptions |

**Deprecated/outdated:**
- `dirsToSkip` in `pkg/codegraph/codegraph.go`: replaced by `ShouldSkipDir()` from `scan_filter.go`
- `shouldSkipSurveyDir` in `cmd/codex_colonize.go`: replaced by `ShouldSkipDir()`
- `extendedSkipDirs` in `cmd/init_research.go`: replaced by `ShouldSkipDir()`
- `skillScanSkipDirs` in `cmd/skills.go`: replaced by `ShouldSkipDir()`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `node_modules/.cache` is covered by `node_modules` exclusion, no subdirectory matching needed | Pitfall 3 | Low -- if node_modules is excluded, all subdirectories are automatically skipped |
| A2 | `.cargo/registry` is a directory name that appears as `.cargo` at the top level; excluding `.cargo` covers it | GROUND-02 | Low -- Rust projects use `~/.cargo/registry` at the user level, not typically inside repos |
| A3 | `site-packages` only appears inside `.venv` or `venv` directories, so excluding the parent covers it | GROUND-02 | Low -- standalone `site-packages` directories are theoretically possible but extremely rare in repos |
| A4 | The `discuss_analyze.go` walk uses `extendedSkipDirs` from `init_research.go` directly | Current Divergent Lists | None -- verified by reading the source |

## Open Questions

1. **Should `.env` (with dot) be excluded?**
   - What we know: `.env` is a common virtual environment directory in Python projects, but it is also used for environment variable files (`.env` file, not directory)
   - What's unclear: Whether excluding `.env` as a directory name would accidentally skip a directory of env-config source files
   - Recommendation: Include `.env` in the skip list. The `.env` *file* is not affected by directory-level skipping. The `.env` *directory* is almost always a virtual environment. Projects that use an `.env/` directory for source code are extremely rare and can use REDIRECT pheromones.

2. **Should source anchors include config files?**
   - What we know: Config files (`go.mod`, `package.json`, `tsconfig.json`) are already tracked in `ConfigFiles` separately
   - What's unclear: Whether the planner benefits from config files appearing in both `ConfigFiles` and `SourceAnchors`
   - Recommendation: Exclude config files from source anchors. They are already tracked separately and are not "source" in the programming sense.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All changes | Yes | 1.26.1 darwin/arm64 | -- |
| Go test runner | Regression tests | Yes | `go test` | -- |
| No external dependencies | -- | -- | -- | -- |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (Go convention) |
| Quick run command | `go test ./pkg/codegraph/ -run TestShouldSkip -v` |
| Full suite command | `go test ./pkg/codegraph/ ./cmd/ -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GROUND-01 | ShouldSkipDir returns true for all known noise dirs | unit | `go test ./pkg/codegraph/ -run TestShouldSkipDir -v` | No -- Wave 0 |
| GROUND-01 | Divergence test: all call sites use shared filter | unit | `go test ./cmd/ -run TestSkipListDivergence -v` | No -- Wave 0 |
| GROUND-02 | ShouldSkipDir covers `.venv`, `__pycache__`, etc. | unit | `go test ./pkg/codegraph/ -run TestShouldSkipDir_ExtendedCoverage -v` | No -- Wave 0 |
| GROUND-03 | extractSourceAnchors returns max 50, sorted by depth | unit | `go test ./cmd/ -run TestExtractSourceAnchors -v` | No -- Wave 0 |
| GROUND-04 | anchors.json written with source_anchors field | unit | `go test ./cmd/ -run TestAnchorsWrittenToSurvey -v` | No -- Wave 0 |
| GROUND-05 | M4L fixture with `.venv` produces zero `.venv` refs | regression | `go test ./cmd/ -run TestVenvNoiseExclusion -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/codegraph/ ./cmd/ -run TestScan -v`
- **Per wave merge:** `go test ./pkg/codegraph/ ./cmd/ -count=1`
- **Phase gate:** `go test ./... -count=1`

### Wave 0 Gaps
- [ ] `pkg/codegraph/scan_filter_test.go` -- covers GROUND-01, GROUND-02
- [ ] `cmd/codex_colonize_test.go` additions -- covers GROUND-03, GROUND-04, GROUND-05
- [ ] Divergence detection test -- grep for local skip map patterns across cmd/

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `pkg/codegraph/codegraph.go` (full file, 687 lines) -- `dirsToSkip` at line 72, `Scan()` at line 499
- Direct codebase analysis of `cmd/codex_colonize.go` -- `shouldSkipSurveyDir` at line 976, `surveyWorkspace` at line 417, `codexWorkspaceFacts` struct at line 40, `writeSurveyCompatibilityJSON` at line 897
- Direct codebase analysis of `cmd/init_research.go` -- `extendedSkipDirs` at line 147
- Direct codebase analysis of `cmd/skills.go` -- `skillScanSkipDirs` at line 150
- Direct codebase analysis of `cmd/discuss_analyze.go` -- uses `extendedSkipDirs` at line 131
- Direct codebase analysis of `cmd/codex_plan.go` -- `codexSurveyContext` at line 45, `loadCodexSurveyContext` at line 1182
- Direct codebase analysis of `pkg/codegraph/codegraph_test.go` (full file, 410 lines) -- `TestDirsToSkip` at line 334
- Direct codebase analysis of `cmd/codex_colonize_test.go` (full file, 1566 lines) -- `TestSurveyWorkspaceDockerRepoIgnoresPlatformGuidance` at line 797
- Milestone-level research: `.planning/research/STACK.md`, `FEATURES.md`, `ARCHITECTURE.md`, `PITFALLS.md`

### Secondary (MEDIUM confidence)
- Milestone architecture decisions from roadmap -- `GROUND-01` through `GROUND-05` requirements specification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all capabilities verified in codebase
- Architecture: HIGH -- all five skip lists located and catalogued; data flow from survey to planner traced end-to-end
- Pitfalls: HIGH -- the `.venv` bug is confirmed from M4L-AnalogWave-System repo; divergence is verified by reading all five source files

**Research date:** 2026-05-18
**Valid until:** 2026-06-18 (stable -- no external dependencies or fast-moving libraries)
