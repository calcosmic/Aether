# Phase 74: Suggest-Analyze - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

During builds (Step 4.2), Aether automatically detects codebase patterns and suggests them as pheromone signals for user review. Suggestions are deduplicated against active pheromones, presented inline during the build ceremony for tick-to-approve review, and persist until resolved.

Requirements: INTEL-01, INTEL-02, INTEL-03.

**What this phase delivers:**
- A Go CLI command (`aether suggest-analyze`) that runs the existing 25 pheromone patterns plus build-specific extras against the codebase
- Deduplication against existing active pheromones (exact match by type + content hash)
- Inline approval UX during the build ceremony (Step 4.2) — user approves or dismisses each suggestion before workers spawn
- Persistent suggestion storage — unreviewed suggestions survive `/clear` and show up on every build until resolved
- Updated `suggest-approve` command to handle the full approval flow
- Wrapper updates for Claude Code and OpenCode to wire Step 4.2 into the build playbook
- Codex gets the same experience via the Go runtime (no separate markdown)

**What this phase does NOT deliver:**
- Bayesian confidence scoring (Phase 75)
- Circuit breaker (Phase 75)
- User-extensible pattern definitions (D-09 from Phase 73: built-in only)

</domain>

<decisions>
## Implementation Decisions

### Suggestion Trigger and Scope
- **D-01:** Suggest-analyze runs on the first build of each colony, then re-runs on subsequent builds only if the codebase changed significantly since last analysis. Change detection uses git diff (files changed since last analysis timestamp or commit).
- **D-02:** Uses the same 25 base patterns from `generatePheromoneSuggestions()` in `cmd/init_research.go` PLUS build-specific extras. Build-specific extras detect: TODO/FIXME density, test coverage gaps, large files, and similar build-relevant patterns.
- **D-03:** The `--no-suggest` flag (already documented in build-prep.md) skips suggest-analyze entirely.

### Approval UX
- **D-04:** Suggestions display inline during the build ceremony at Step 4.2, after context assembly and before skill detection. User reviews and approves/dismisses each suggestion before workers spawn. This briefly pauses the build flow.
- **D-05:** Unreviewed suggestions persist in colony state until explicitly approved or dismissed. They survive `/clear` and show up on every build until resolved. No auto-expiry.
- **D-06:** Approved suggestions are written as pheromone signals using the existing `pheromone-write` command (with content hash dedup). Dismissed suggestions are recorded as dismissed and never re-shown.

### Deduplication Behavior
- **D-07:** Deduplication uses exact match only — same pheromone type + same content hash. This is already implemented in `pheromone-write` (content_hash + reinforcement logic).
- **D-08:** Suggestions that match expired pheromones are still shown — the user might want to re-activate them. Only active pheromones suppress suggestions.

### Platform Differences
- **D-09:** Single Go CLI command (`aether suggest-analyze`) is the authoritative implementation. Claude Code and OpenCode wrappers call it and display the output. Codex calls it directly via the runtime. This follows the CLAUDE.md platform policy: Go runtime owns state mutations, wrappers own presentation.

### Claude's Discretion
- Exact change detection threshold (how many files changed triggers re-analysis)
- Number and content of build-specific extra patterns
- Visual rendering of the approval UI in each platform's wrapper
- Storage format for pending suggestions in colony state
- Whether to show a summary count ("3 new suggestions") or full details when re-displaying persisted unreviewed suggestions

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — INTEL-01, INTEL-02, INTEL-03 define suggest-analyze requirements

### Roadmap
- `.planning/ROADMAP.md` — Phase 74 goal, success criteria, dependency on Phase 71

### Existing Suggest-Approve Code (stub)
- `cmd/compatibility_cmds.go:113-133` — Current `suggest-approve` command (stub returning empty suggestions)

### Existing Pheromone Pattern Detection
- `cmd/init_research.go:50-54` — `pheromoneSuggestion` struct (Type, Content, Reason)
- `cmd/init_research.go:1308-1600` — `generatePheromoneSuggestions()` — 25 deterministic patterns
- `cmd/init_ceremony.go:110-130` — Init ceremony auto-approves pheromone suggestions

### Pheromone Dedup Infrastructure
- `cmd/pheromone_write_test.go` — Tests for content_hash dedup and reinforcement behavior
- `cmd/write_cmds_test.go:147-178` — Test proving duplicate content reinforces instead of appending

### Build Ceremony Playbooks
- `.aether/docs/command-playbooks/build-prep.md:118-137` — `--no-suggest` flag parsing
- `.aether/docs/command-playbooks/build-context.md:147-181` — Step 4.2 (currently DEPRECATED, needs restoration)

### Colony State Schema
- `pkg/colony/` — ColonyState struct definition

### Platform Wrappers
- `.claude/commands/ant/build.md` — Claude Code build wrapper
- `.opencode/commands/ant/build.md` — OpenCode build wrapper
- `.codex/CODEX.md` — Codex commands and rules

### Prior Phase Context
- `.planning/phases/73-rich-init-research/73-CONTEXT.md` — Phase 73 decisions: 25 built-in patterns (D-07/D-09), no user extensibility, Claude's discretion on implementation approach (D-08)

### Architecture
- `CLAUDE.md` — Platform policy, wrapper-runtime contract, zero-new-deps principle
- `.aether/docs/wrapper-runtime-ux-contract.md` — Full contract for wrapper-runtime delegation

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `generatePheromoneSuggestions()` in `cmd/init_research.go` — 25 patterns covering security, governance, documentation, containers, API, database, dependency health. Already takes governance info, directory classification, and tech stack as input. Can be extracted or called directly.
- `pheromoneSuggestion` struct — Type/Content/Reason with JSON tags. Ready to serialize.
- `pheromone-write` CLI command — Already handles content_hash dedup and reinforcement. Approved suggestions just need to pipe through this.
- `suggest-approve` CLI command — Stub exists, needs real implementation.
- `hasFile()`, `fileContains()`, `hasDir()` utility functions in `cmd/init_research.go` — Used by pattern detection.

### Established Patterns
- `outputOK()` for JSON output + visual rendering
- Struct-based data types with JSON tags for all results
- `--dry-run` flag pattern for preview operations
- `--no-suggest` flag already parsed in build-prep.md
- Zero-new-deps principle — all new code uses existing Go stdlib + cobra + pkg/storage

### Integration Points
- Step 4.2 in build-context.md playbook (currently DEPRECATED — needs restoration)
- Colony state needs new fields for pending suggestions (stored in COLONY_STATE.json)
- `aether pheromone-write` for approved suggestions
- `aether pheromone-read` for dedup checking against active signals
- Init ceremony already auto-approves pheromone suggestions — suggest-analyze during build should use the same pattern infrastructure but with user approval instead of auto-approve

</code_context>

<specifics>
## Specific Ideas

- The existing `suggest-approve` stub in `cmd/compatibility_cmds.go` needs to be replaced with real logic or a new `suggest-analyze` command should be created
- Build-specific extras should detect patterns relevant to the current build phase (e.g., if building tests, detect test coverage gaps)
- The approval UX should feel lightweight — a quick scan and tick, not a heavy decision flow
- Re-analyzing only on significant change avoids wasting time on unchanged codebases

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 74-suggest-analyze*
*Context gathered: 2026-04-29*
