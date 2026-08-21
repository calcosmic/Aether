---
phase: 146-command-classification
date: 2026-05-20
researcher: gsd-phase-researcher
---

# Phase 146 Research: Command Classification

## Command Inventory

**Total commands: 367** registered in `cmd/testdata/command_catalog.json`

| Classification | Count | Description |
|----------------|-------|-------------|
| public_lifecycle | 16 | Core colony lifecycle: init, plan, build, continue, seal, colonize, run, entomb |
| public_utility | 341 | Supporting commands: export, import, council, skill-*, host subcommands, etc. |
| alias | 8 | Wrapper aliases: watch, pheromone-export-xml, wisdom-export-xml, etc. |
| internal_runtime | 2 | System hooks: autofix-checkpoint, autofix-rollback |
| deprecated | 0 | None currently tagged (error-pattern-check is internal) |

**Lifecycle commands (public_lifecycle):**
- `init`, `init-ceremony`, `init-research`
- `plan`, `plan-finalize`, `plan-granularity`
- `build`, `build-finalize`
- `continue`, `continue-finalize`
- `seal`, `seal-finalize`
- `colonize`, `colonize-finalize`
- `run`, `entomb`

**Alias commands (8):**
- `watch` (alias for host watch)
- `pheromone-export-xml`, `pheromone-import-xml`
- `wisdom-export-xml`, `wisdom-import-xml`
- `registry-export-xml`, `registry-import-xml`
- `colony-archive-xml`

**Internal runtime (2):**
- `autofix-checkpoint`, `autofix-rollback`

## Classification Scheme

Five-tier scheme derived from existing command patterns:

1. **public_lifecycle** — Commands a user runs to drive colony phases (init → plan → build → continue → seal)
2. **public_utility** — Commands that support or inspect the colony (export, council, skill-*, etc.)
3. **internal_runtime** — Commands the system uses, not for manual invocation (hooks, autofix)
4. **alias** — Shorthand wrappers that delegate to another command
5. **deprecated** — Commands kept for backward compatibility but scheduled for removal

**Decision rationale:**
- Matches natural user mental model (lifecycle vs utility)
- Distinguishes human-facing from system-facing commands
- Separates aliases so they don't duplicate test coverage requirements
- Deprecated bucket prevents obsolete commands from skewing reliability metrics

## Historical Tag Sourcing

**Git tags available:** v1.10, v1.11, v1.12, v1.13, v1.14, v1.17, v1.18, v1.19, v1.20, v1.21, v5.4.0

**Historical growth:**
- v5.4.0 (Classic): 14 commands
- v1.10: 316 commands
- v1.21: 368 commands
- Current: 367 commands

**55 commands added after v1.10**, including:
- `audit-catalog` (first in v1.17)
- `host`, `lifecycle`, `oracle-iterate` (first in v1.19)
- `ceremony`, `closeout`, `spawn-plan` (first in v1.17)
- `council`, `suggest-analyze`, `versions` (first in v1.11)
- `feedback`, `redirect` (first in current/unreleased)

**Reliability matrix approach:**
For each command in current catalog, check presence in each historical tag.
Output: `Y` if command exists in that era, `-` if not.
Commands present in more tags = higher historical reliability.

## Implementation Approach for `aether audit-catalog`

**Option A: JSON catalog extension (recommended)**
- Extend existing `cmd/testdata/command_catalog.json` with `classification` and `historical_tags` fields
- `aether audit-catalog` reads this JSON and outputs:
  - Markdown table (human-readable)
  - JSON (machine-readable, for downstream automation)
- Catalog is already the source of truth for command metadata

**Option B: Go source annotation**
- Add classification constants to `cmd/` source files
- Derive catalog at build time from source annotations
- More robust but requires touching every command file

**Recommendation: Option A** — catalog JSON already exists and is maintained. Adding classification fields is low-touch. The catalog can be enriched with historical data generated once from git history.

## Files to Modify

| File | Purpose |
|------|---------|
| `cmd/testdata/command_catalog.json` | Add `classification`, `historical_tags`, `since_version` fields |
| `cmd/audit_catalog.go` | Implement `aether audit-catalog` with markdown + JSON output |
| `cmd/audit_catalog_test.go` | Tests for classification accuracy and output formats |
| `.planning/REQUIREMENTS.md` | Mark CATALOG requirements as addressed |

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Classification disagreements | Medium | Document heuristics; make overridable per-command |
| Historical tag scraping is slow | Low | Run once, cache in catalog JSON |
| Wrapper-only commands missing from catalog | Medium | Cross-check `.claude/commands/` and `.opencode/commands/` against catalog |
| Classification drifts as commands added | Medium | CI gate: new commands must have classification in catalog |

## Research Gaps

1. Wrapper markdown commands (Claude Code slash commands) are NOT in `command_catalog.json` — should they be classified separately or included?
2. The `host` command family blurs lifecycle/utility boundary (host build = lifecycle, host oracle = utility)
3. Some subcommands have no tests — classification helps identify coverage gaps but doesn't fix them
