# Phase 73: Rich Init Research - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Phase Boundary

The init ceremony produces deep codebase analysis: tech stack with parsed dependency lists, directory structure classification with detection signals, governance file parsing that extracts actual rules/settings, and expanded pheromone suggestion patterns (~25). All analysis runs during `aether init-research` and the results feed into the colony context summary that the init ceremony outputs.

Requirements: INIT-03, INIT-04, INIT-05, INIT-06, INIT-07.

**What this phase delivers:**
- Tech stack analysis that parses actual dependency files (package.json, go.mod, Cargo.toml, etc.) and reports all dependencies including dev/indirect
- Directory structure classification (monorepo, microservices, standard app, library) with detection signals explaining why
- Governance config file parsing that extracts actual rules/settings from all 5 categories (linters, formatters, test frameworks, CI configs, build tools)
- Expanded pheromone suggestion patterns from 10 to ~25, built-in only (no user extensibility)
- Formatted colony context summary incorporating all research output

**What this phase does NOT deliver:**
- Suggest-analyze during builds (Phase 74)
- Bayesian confidence scoring (Phase 75)
- Changes to the charter ceremony flow (Phase 72 owns the approval UX)

</domain>

<decisions>
## Implementation Decisions

### Tech Stack Depth
- **D-01:** Parse actual dependency files to extract package names, version ranges, and dependency counts — not just file-presence detection. Support: package.json (deps + devDeps), go.mod (requires), Cargo.toml (deps), pyproject.toml/requirements.txt, Gemfile, pom.xml, mix.exs, composer.json.
- **D-02:** Include ALL dependencies in output (production + dev + indirect). Full list, not summarized.

### Directory Structure Patterns
- **D-03:** Classify directory structure using heuristic pattern matching: monorepo (packages/, apps/, workspaces, pnpm-workspace.yaml), microservices (service-per-dir, multiple Dockerfiles), standard app (src/, lib/, cmd/), library (no src/, exports in root), and "unknown" fallback.
- **D-04:** Output includes both the classification type AND detection signals (which files/dirs triggered the classification). Example: "monorepo — detected: packages/, pnpm-workspace.yaml, 4 workspace members".

### Governance File Details
- **D-05:** Parse governance config files to extract actual rules/settings — not just report tool names. All 5 categories: linters (extract rules/extends), formatters (extract options), test frameworks (extract config), CI configs (extract pipeline steps), build tools (extract targets/scripts).
- **D-06:** All categories are parsed at the same depth — no category stays at detection-only level.

### Pheromone Suggestions Expansion
- **D-07:** Expand from 10 to ~25 deterministic pheromone suggestion patterns. Add patterns for: monorepo workspace consistency, API patterns (OpenAPI/swagger), database presence (migrations, schema files), security patterns (CSP headers, CORS config), container patterns (Docker compose, multi-stage builds), documentation patterns (API docs, changelog), dependency health (outdated lockfiles, known vulnerability indicators).
- **D-08:** Implementation approach is Claude's discretion (hard-coded Go functions vs data-driven registry — pick whichever fits the established patterns in init_research.go).
- **D-09:** Built-in patterns only — no user extensibility. Users who want custom patterns can use pheromone commands directly.

### Claude's Discretion
- Implementation approach for pheromone pattern registry (D-08)
- Exact dependency parsing depth for each supported file format
- How governance parsing handles malformed or unusual config files
- Exact pheromone suggestion patterns to add beyond the examples listed
- How the colony context summary formats all the new research data

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — INIT-03 through INIT-07 define rich init-research requirements

### Roadmap
- `.planning/ROADMAP.md` — Phase 73 goal, success criteria, dependency on Phase 72

### Existing Init Research Code (authoritative)
- `cmd/init_research.go` — Current implementation: project detection (12 languages), governance detection (21 config files), git history analysis, 10 pheromone suggestion patterns, charter generation (7 sections), basic complexity metrics
- `cmd/init_research_test.go` — Existing tests for init-research

### Colony State Schema
- `pkg/colony/` — ColonyState struct definition, Charter struct with 7 fields

### Platform Wrappers
- `.claude/commands/ant/init.md` — Claude Code init wrapper (calls init-research, presents charter)
- `.opencode/commands/ant/init.md` — OpenCode init wrapper
- `.codex/CODEX.md` — Codex commands and rules

### Prior Phase Context
- `.planning/phases/72-smart-init-charter/72-CONTEXT.md` — Phase 72 decisions: charter persistence (D-01/D-02), phase boundary with 73 (D-04), dual mode (D-08/D-09), zero-new-deps principle

### Architecture
- `CLAUDE.md` — Platform policy, wrapper-runtime contract, zero-new-deps principle
- `.aether/docs/wrapper-runtime-ux-contract.md` — Full contract for wrapper-runtime delegation

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/init_research.go` — Full implementation with `projectDetectors` (12 language types), `governanceDetectors` (21 config files across 5 categories), `generatePheromoneSuggestions()` (10 patterns), `detectGovernance()`, `analyzeGitHistory()`, `generateCharter()`, `complexityMetrics`, `extendedSkipDirs`
- `pkg/colony/colony.go` — ColonyState struct, Charter struct with Intent/Vision/Governance/Goals/TechStack/KeyRisks/Constraints
- Go stdlib `encoding/json` — for parsing package.json, Cargo.toml (JSON), composer.json
- `gopkg.in/yaml.v3` — already a dependency, for parsing YAML config files
- `github.com/BurntSushi/toml` — already a dependency, for parsing TOML files (Cargo.toml, pyproject.toml)

### Established Patterns
- `hasFile()` and `fileContains()` utility functions for file existence/content checks
- `outputOK()` for JSON output + visual rendering
- Struct-based data types with JSON tags for all scan results
- `extendedSkipDirs` for directory walk filtering
- Zero-new-deps principle — all parsing uses existing Go stdlib + already-imported packages

### Integration Points
- `aether init-research --goal "..."` output is consumed by wrapper init commands
- Charter fields (TechStack, KeyRisks, Constraints) should be enriched from deeper research
- Colony context summary (INIT-07) needs to incorporate all new research sections
- Phase 74 (suggest-analyze) will build on the pattern detection infrastructure

</code_context>

<specifics>
## Specific Ideas

- User wants all dependencies reported, not just production ones — full visibility into what the project uses
- Directory classification should explain WHY it classified the project that way (detection signals)
- Governance parsing should cover all 5 categories at the same depth — no half-measures
- Pheromone patterns should be built-in only — no user config to manage
- Phase 74 will add runtime suggest-analyze, so Phase 73 should build a solid pattern infrastructure

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 73-rich-init-research*
*Context gathered: 2026-04-28*
