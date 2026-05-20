# Stack Research: v1.22 Grounded Planning + Ceremony Restore

**Domain:** Biomimetic AI colony framework (Go runtime + TypeScript host + YAML wrappers)
**Researched:** 2026-05-18
**Confidence:** HIGH (direct codebase analysis of all relevant source files; no new external dependencies needed)

## Recommended Stack

### Core Principle: Zero New Dependencies

This milestone is an architecture restoration, not a feature expansion. Every capability
already exists in some form in the codebase. The work is about connecting them correctly,
not introducing new libraries.

### Go Runtime Additions (cmd/, pkg/)

| Component | Location | Purpose | Why |
|-----------|----------|---------|-----|
| Survey noise filter | `pkg/codegraph/scan_filter.go` (new) | Extended directory/file exclusion for survey and codegraph scanning | `dirsToSkip` in codegraph already excludes 13 dirs; `shouldSkipSurveyDir` in colonize excludes 10; both miss `.venv`, `site-packages`, `.mypy_cache`, `.pytest_cache`, `.tox`, `eggs`, `*.egg-info`, `.eggs`, and generated protobuf/typescript `.d.ts` artifacts. A shared filter function used by both avoids divergence. |
| Source anchor extraction | `cmd/codex_colonize.go` (extend `surveyWorkspace`) | Extract repo-owned source files (non-dependency, non-generated) as planning anchors | `surveyWorkspace` already collects `TopLevelDirs`, `EntryPoints`, `TestFiles`, `KeyDependencies`; it needs a new field `SourceAnchors []string` that lists the project's own source files (not vendored). These anchors become evidence for plan grounding. |
| Plan grounding gate | `cmd/codex_plan_finalize.go` (extend) | Reject plans that reference no specific repo files when source anchors exist | `codex_plan_finalize` already validates plans; add a check: if `SourceAnchors` is non-empty in the colony context but the plan contains zero concrete file references, emit a `grounding_warning` instead of accepting the plan as-is. This is a soft gate (warning, not rejection) because some phases legitimately have no file targets (e.g., research phases). |
| Decision-to-constraint binding | `cmd/discuss.go` (extend `resolveDiscussQuestion`) | When a discuss question is resolved with `hard_constraint: true`, write it to the pending-decisions store and inject into colony-prime as a REDIRECT pheromone | Already has `discussQuestion.HardConstraint` field and `pending-decision-add` command; needs a new path in `resolveDiscussQuestion` that auto-emits a REDIRECT pheromone for hard constraints, so the plan reads them. |

### TypeScript Host Additions (.aether/ts-host/)

| Component | Location | Purpose | Why |
|-----------|----------|---------|-----|
| Playbook-aware orchestration | `.aether/ts-host/src/lifecycle.ts` (extend) | Load and execute build/plan/continue playbooks as the ceremony source of truth, instead of the current approach where lifecycle.ts bypasses playbooks and directly calls Go commands | Playbooks in `.aether/docs/command-playbooks/` contain the ceremony steps; `build_playbook_context.go` already injects playbook snippets into worker context; the TS host needs to read playbooks and use them as the execution script rather than hardcoding orchestration steps. The Go runtime already has `renderBuildPlaybookContext` -- the TS host should consume the same playbook files. |
| Go ceremony adapter integration | `.aether/ts-host/src/ceremony-adapter.ts` (existing, extend usage) | Route all visual ceremony through `GoCeremonyAdapter` which already calls `aether ceremony spawn-plan/wave-start/worker-complete/closeout` | Already implemented and tested (465 TS tests). The ceremony adapter IS the restore mechanism -- wrappers call Go ceremony commands which produce visual output. No new code needed, just ensure all TS host orchestration paths use it. |

### Shared Filter Design

The most important stack decision is the noise filter. Three locations currently have separate skip lists:

```go
// pkg/codegraph/codegraph.go:dirsToSkip (13 entries)
// Missing: .venv, venv, site-packages, .mypy_cache, .pytest_cache, .tox, eggs, .eggs, *.egg-info

// cmd/codex_colonize.go:shouldSkipSurveyDir (10 entries)
// Missing: .venv, venv, __pycache__, site-packages, .mypy_cache, .pytest_cache, .tox, .eggs

// No file-level exclusions in either (e.g., *.min.js, *.bundle.js, .d.ts)
```

**Recommendation:** Create a single `ScanFilter` in `pkg/codegraph/scan_filter.go` with both directory and file-level exclusions, used by both codegraph and colonize. This is pure Go stdlib -- no external dependency.

```go
// pkg/codegraph/scan_filter.go
package codegraph

// ScanFilter provides shared path exclusion logic for repo scanning.
// Both codegraph.Scan() and codex_colonize.surveyWorkspace() call
// ShouldSkipDir and ShouldSkipFile to keep filtering consistent.

var noiseDirs = map[string]bool{
    // Version control
    ".git": true,
    // Python virtual environments and caches
    ".venv": true, "venv": true, "env": true, ".env": true,
    "__pycache__": true, ".mypy_cache": true, ".pytest_cache": true,
    ".tox": true, ".ruff_cache": true, ".pytype": true,
    "site-packages": true,
    // Node.js
    "node_modules": true, ".next": true, ".nuxt": true,
    // Go
    "vendor": true,
    // Rust
    "target": true,
    // General build artifacts
    "dist": true, "build": true, "out": true, "bin": true,
    // Caches
    ".cache": true, ".terraform": true, ".gradle": true,
    // Aether-managed
    ".aether": true, ".claude": true, ".codex": true, ".opencode": true,
    // IDE/editor
    ".idea": true, ".vscode": true, ".vs": true,
    // OS metadata
    ".DS_Store": true, "Thumbs.db": true,
}

// noiseFileExts lists file extensions to skip during scanning.
var noiseFileExts = map[string]bool{
    ".min.js": true, ".min.css": true, ".bundle.js": true, ".map": true,
    ".pyc": true, ".pyo": true, ".so": true, ".dylib": true, ".dll": true,
    ".d.ts": true, // TypeScript declaration files (not user source)
}
```

**Why not use `go-gitignore` library:** Adding a dependency violates the project's zero-new-dependencies principle (established in v1.9 and reaffirmed in every milestone since). The `.gitignore` approach sounds flexible but introduces complexity: Aether targets repos that may have stale or missing `.gitignore` files, and parsing gitignore patterns adds a dependency for a problem solvable with a 25-entry map. The known noise directories are finite and stable across language ecosystems. If a user has an unusual exclusion need, pheromone REDIRECT covers it.

**Why not `.gitignore` parsing:** Some target repos are new/empty (the exact case where Aether's survey matters most) and may not have `.gitignore`. The hardcoded list is safer because it always works.

### Source Anchor Extraction Strategy

**Approach:** After the noise filter removes directories, what remains falls into two categories:
1. **Project source** -- files the user wrote (e.g., `src/`, `lib/`, `cmd/`, `internal/`, `app/`)
2. **Configuration and tooling** -- files that describe but don't implement (e.g., `README.md`, `Dockerfile`, `.eslintrc`)

Source anchors are project source files. The extraction logic:

```
1. Walk repo root with noise filter
2. For each file, check if it's a source file (language extension match)
3. Exclude test files from anchors (test files are already tracked separately)
4. Group by directory; directories with >50% source files are "source directories"
5. Files in source directories are anchors
6. Cap at 50 anchors (sorted by path depth -- shallowest first = most important)
```

This gives the planner concrete file paths to reference, producing grounded plans like:
```
Phase 1: Add user authentication
  - Modify src/auth/middleware.ts (session validation)
  - Create src/auth/login.ts (new endpoint)
```
Instead of generic plans like:
```
Phase 1: Add user authentication
  - Implement session management
  - Add login endpoint
```

### Plan Grounding Gate Design

**Approach:** A soft validation gate in `codex_plan_finalize.go`. After the Route-Setter produces a plan, scan each task's `goal`, `constraints`, and `hints` fields for concrete file path patterns (e.g., `src/`, `cmd/`, relative paths with extensions). If the plan contains zero concrete file references AND `SourceAnchors` is non-empty, emit a `grounding_warning` in the plan output.

**Why a soft gate, not hard rejection:** Some phases legitimately have no file targets:
- Research phases (Oracle work)
- Architecture decisions (Architect work)
- Infrastructure phases (deployment config)

A hard rejection would break these. A warning lets the user decide whether the plan needs more specificity.

**Pattern for detecting file references:**
```go
// File reference pattern: anything that looks like a path with an extension
var fileRefPattern = regexp.MustCompile(`(?:src|lib|cmd|pkg|app|internal|docs)/[^\s"')\]]+\.\w+`)

// Or direct relative path references like "./file.ts", "../file.go"
var relPathPattern = regexp.MustCompile(`\.\.?/[\w./-]+\.\w+`)
```

### Decision Binding Design

The discuss command already stores decisions as `PendingDecision` structs with a `Type` field. The binding mechanism:

1. When `resolveDiscussQuestion` is called with `HardConstraint: true`:
   - Create a `PendingDecision` with type `"hard_constraint"` (already uses this type)
   - Emit a REDIRECT pheromone via `pheromone-write` with content from the resolved answer
   - Colony-prime already injects REDIRECT signals into worker context
2. During plan generation, colony-prime context includes active REDIRECT signals
3. The Route-Setter sees constraints like: `REDIRECT: "Do not use ORM -- raw SQL only (source: discuss:pd_12345)"`
4. Plans naturally respect these constraints

**Why pheromone-based binding:** The entire colony system already routes through pheromones. Adding a parallel constraint mechanism would create confusion. The pheromone system has content deduplication (SHA-256 hash), TTL, strength decay, and priority ordering. It is the right tool for this job.

### Ceremony Restore via Playbooks

The current situation: Playbooks exist in `.aether/docs/command-playbooks/` (13 files), the Go runtime has `build_playbook_context.go` that injects playbook snippets into worker context, but the TS host lifecycle.ts orchestrates directly without reading playbooks.

**The restore approach:**

1. **TS host reads playbooks** from `.aether/docs/command-playbooks/` (same files the Go runtime uses)
2. **Playbooks become the execution script** -- lifecycle.ts follows the step sequence defined in the playbook markdown rather than hardcoded Go-command sequences
3. **Go ceremony adapter renders visuals** -- the TS host calls `GoCeremonyAdapter` for every visual element (spawn-plan, wave-start, worker-complete, closeout)
4. **YAML becomes packaging metadata only** -- `.aether/commands/*.yaml` defines the command name, flags, and runtime command mapping; ceremony content lives exclusively in playbooks

This means:
- **Build ceremony**: lifecycle.ts reads `build-full.md` (or the split playbooks), follows each step, and calls Go ceremony adapter for visuals
- **Plan ceremony**: lifecycle.ts reads the plan playbook steps, follows the Scout/Route-Setter flow, calls Go ceremony adapter for visuals
- **Continue ceremony**: lifecycle.ts reads `continue-full.md` (or the split playbooks), follows the verify-gates-advance flow

**Why playbooks, not code:** Playbooks are markdown files that the user can read and edit. This is core to Aether's philosophy -- "editable colony brain." If ceremony steps live in TypeScript code, only developers can modify them. If they live in markdown playbooks, any user can understand and customize the ceremony.

**Key existing code to reuse:**
- `renderBuildPlaybookContext()` in `cmd/build_playbook_context.go` -- already reads playbooks
- `GoCeremonyAdapter` in `.aether/ts-host/src/ceremony-adapter.ts` -- already calls Go ceremony commands
- `loadTemplate()` in `.aether/ts-host/src/template-loader.ts` -- already reads ceremony templates
- `buildPlaybookCandidates()` in `cmd/build_playbook_context.go` -- already resolves playbook paths (repo, hub, absolute)

## Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `path/filepath` | Go 1.24+ | Directory walking, path matching | All noise filtering (no external dep needed) |
| Go stdlib `regexp` | Go 1.24+ | File reference pattern detection for grounding gate | Plan validation |
| Go stdlib `encoding/json` | Go 1.24+ | Source anchor serialization, plan output | Data flow |
| Node.js `node:fs` | Built-in | Playbook file reading in TS host | Playbook consumption |
| Node.js `node:child_process` | Built-in | Go ceremony adapter subprocess calls | Already used |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `github.com/sabhiram/go-gitignore` | Adds external dependency for a problem solved by a 25-entry map. Target repos may lack `.gitignore`. Parsing gitignore patterns is complex (negation, globs, directory-only semantics). | Hardcoded `noiseDirs` map in `pkg/codegraph/scan_filter.go` |
| AST-based parsing (e.g., `go/ast`, `typescript-eslint/parser`) | Overkill for survey noise filtering. Codegraph intentionally uses regex-based import parsing for the 80% case (see codegraph.go design doc). Full AST would add parsing dependencies and slow scans. | Keep existing regex-based parsing, extend `dirsToSkip` |
| New YAML/config file for exclusion rules | Creates configuration drift. Users already have pheromone REDIRECT for custom exclusions. | Hardcoded defaults + pheromone override |
| Database (SQLite, etc.) for source anchors | Anchors are a scan-time artifact, not persistent state. They should be regenerated on each survey. | In-memory computation during `surveyWorkspace()`, stored in survey JSON |
| New TypeScript framework (e.g., Zod, Ajv) for playbook parsing | Playbooks are markdown with step headers, not structured data that needs schema validation. The Go runtime already handles playbook file resolution. | Simple string parsing of step headers (`### Step N:`) |
| LLM-based grounding validation | Grounding is a structural check (does the plan mention concrete files?), not a semantic judgment. An LLM check would add latency and cost for a deterministic problem. | Regex pattern matching for file references |

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| Shared Go filter function | `.gitignore` parsing library | Adds dependency; `.gitignore` may not exist in target repos |
| Soft grounding gate (warning) | Hard grounding gate (rejection) | Research/architecture phases legitimately have no file targets |
| Pheromone-based decision binding | Separate constraint file | Pheromone system already handles priority, TTL, dedup, injection |
| Playbook-driven TS orchestration | Code-driven TS orchestration | Playbooks are editable by users; code is only editable by developers |
| Hardcoded noise directory map | Configurable noise list in COLONY_STATE.json | YAGNI -- the 25 known noise dirs cover 99% of cases; REDIRECT covers edge cases |

## Stack Patterns by Variant

**If the target repo is a Python project:**
- Source anchors come from directories not in the noise list (e.g., `src/`, `lib/`, `app/`)
- Noise filtering is critical -- `.venv` and `site-packages` must be excluded to avoid polluting the survey with dependency code
- Test files detected by `_test.py` suffix or `tests/` directory

**If the target repo is a Go project:**
- Source anchors from `cmd/`, `pkg/`, `internal/` (standard Go layout)
- `vendor/` already excluded by both codegraph and colonize
- Test files detected by `_test.go` suffix

**If the target repo is a TypeScript/Node project:**
- Source anchors from `src/`, `lib/`, `app/`, `pages/`, `components/`
- `node_modules/` and `.next/`/`.nuxt/` already excluded
- Declaration files (`.d.ts`) excluded by new `noiseFileExts`

**If the target repo is a monorepo:**
- Source anchors from each package's `src/` or equivalent
- `node_modules/` at both root and package level already excluded by name match
- Plan grounding should produce per-package file references

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| Go 1.24 (current) | All new code uses `path/filepath`, `regexp`, `strings` | No new Go dependencies |
| Node.js (current TS host) | All new TS code uses `node:fs`, `node:child_process` | No new npm dependencies |
| Existing `pkg/codegraph` API | New `ScanFilter` is additive; `dirsToSkip` remains for backward compat | Both codegraph and colonize should migrate to shared filter |
| Existing ceremony adapter | No changes to `GoCeremonyAdapter` interface | TS host simply calls it more consistently |

## Installation

No new packages required. All additions use Go stdlib and Node.js built-ins.

```bash
# Zero new dependencies
# Go changes: new file pkg/codegraph/scan_filter.go
# TS changes: extend lifecycle.ts to read playbooks
# Playbook changes: already exist in .aether/docs/command-playbooks/
```

## Sources

- Direct codebase analysis: `cmd/survey.go`, `cmd/codex_colonize.go` (lines 417-982), `pkg/codegraph/codegraph.go` (full file), `cmd/codex_plan.go` (lines 1-150), `cmd/ceremony_cmd.go`, `cmd/ceremony_emitter.go`, `cmd/codex_plan_finalize.go`, `cmd/discuss.go`, `cmd/pending_decision.go`, `cmd/build_playbook_context.go`
- TS host analysis: `.aether/ts-host/src/go-bridge.ts`, `.aether/ts-host/src/ceremony-adapter.ts`, `.aether/ts-host/src/template-loader.ts`, `.aether/ts-host/src/lifecycle.ts`, `.aether/ts-host/src/wave-orchestrator.ts`
- Playbook analysis: `.aether/docs/command-playbooks/` (13 files)
- PROJECT.md context: `.planning/PROJECT.md` (v1.22 milestone definition)
- Noise filter patterns: GitHub community patterns for directory exclusion (HIGH confidence based on multiple open-source projects using identical exclusion lists)
- Plan grounding: Research on evidence-based planning validation from academic sources (MEDIUM confidence -- the concept is sound but Aether's implementation is novel)

---
*Stack research for: Aether v1.22 Grounded Planning + Ceremony Restore*
*Researched: 2026-05-18*
