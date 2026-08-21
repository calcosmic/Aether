# Phase 152: Boundary & Parity - Context

**Gathered:** 2026-05-22
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers three documents that define the hybrid architecture salvage:
1. **Architecture boundary** — what stays in Go, what moves to TypeScript, what moves to Markdown/YAML/JSON
2. **Classic parity checklist** — golden reference of Classic v5.4.0 behaviours vs current Go, with verification methods
3. **Behaviour extraction audit** — classification of every Go file containing agent/prompt/phase/skill/memory/ritual/ceremony logic

The hard rule: "Compiled code may execute behaviour, but editable assets must define behaviour."

</domain>

<decisions>
## Implementation Decisions

### Visual/ceremony boundary
- **D-01:** Move hardcoded visuals (caste emojis, ANSI colours, banner templates, stage separators) from compiled Go to editable config files.
- **D-02:** One shared visual config file across all platforms (Claude, OpenCode, Codex) — caste identity is runtime truth, not platform quirk.
- **D-03:** Format: Markdown with YAML frontmatter (matches existing `.aether/agents/` and `.claude/agents/` patterns).
- **D-04:** Location: `.aether/visuals.md` (or equivalent under `.aether/config/`).

### Classic parity scope
- **D-05:** Full inventory of all Classic v5.4.0 behaviours, not just gaps where Go currently differs.
- **D-06:** When Classic and Go disagree on a behaviour, user decides per item — no blanket "Classic wins" or "Go wins" rule.
- **D-07:** Verification method: automated golden/snapshot tests for the 9 flagship workflows + manual checklist steps for edge cases and ceremony.

### Extraction audit granularity
- **D-08:** File-level classification for most Go files (practical, readable).
- **D-09:** Agent-assisted audit: automated first pass (keyword-based heuristics) + agent review for boundary and mixed files.
- **D-10:** Mixed files (containing both spine logic and behaviour strings) annotated as `MIXED` with function-level notes on what stays vs moves.

### Doc location and format
- **D-11:** ARCHITECTURE_BOUNDARY.md and PARITY_CLASSIC_VS_GO.md live in `.aether/docs/` — distributed system reference docs.
- **D-12:** BEHAVIOUR_EXTRACTION_AUDIT.md lives in `.planning/phases/152-boundary-parity/` — phase artifact, not user-facing.
- **D-13:** Format: plain reference doc markdown with clear sections (not ADR template — too rigid for a checklist and an audit table).

### Claude's Discretion
- Visual config format chosen as Markdown+YAML frontmatter (user deferred).
- Verification method chosen as both automated + manual (user deferred).
- Mixed files handling chosen as annotate-as-MIXED (user deferred).
- Doc format chosen as reference doc markdown (user deferred).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project direction
- `.planning/PROJECT.md` — v1.24 milestone goal, key decisions, and explicit deferrals
- `.planning/ROADMAP.md` §Phase 152 — Success criteria and requirements (BOUNDARY-01 through BOUNDARY-04)
- `.planning/STATE.md` — Current milestone state and accumulated decisions

### Classic baseline
- `.planning/milestones/v1.18-ROADMAP.md` — Classic v5.4.0 baseline established; golden workflow snapshot tests

### Existing architecture
- `.planning/codebase/ARCHITECTURE.md` — Current layer separation (Go runtime / wrappers / companion)
- `.planning/codebase/STRUCTURE.md` — Directory structure, key files, platform file locations
- `CLAUDE.md` §UX Architecture — Wrapper-runtime contract, caste identity system, stage markers

### Extraction targets
- `cmd/codex_visuals.go` — Primary example of hardcoded visual behaviour to extract
- `cmd/colony_prime_context.go` — Context assembly logic with hardcoded prompt sections
- `.aether/workers.md` — Existing editable asset pattern (source of truth for worker specs)
- `.aether/commands/*.yaml` — Existing YAML-based command definition pattern

### External reference
- `https://github.com/ruvnet/ruflo` — External hybrid engine + TS control plane architecture reference (not a dependency)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.aether/workers.md` — Existing editable asset pattern; shows how behaviour already lives outside Go for worker definitions
- `.aether/commands/*.yaml` — YAML source definitions for slash commands; proves the YAML→Go loading path already exists
- `.aether/skills/` — Skills already loaded from files at runtime; same pattern can extend to visuals and prompts

### Established Patterns
- Go runtime loads config from `.aether/` at runtime; `aether update` distributes hub changes downstream
- Platform wrappers add framing on top of runtime output — they don't replace it
- Codex is runtime-native; any visual config must be readable by Go directly (no wrapper-only rendering)

### Integration Points
- `aether update` will need to distribute new visual config alongside existing skills/commands
- The audit table will feed directly into Phase 155 (Go Boundary Refactor) — planners must know which files to touch
- Phase 153 (TS Scaffold) needs the architecture boundary doc to know which schemas to define

</code_context>

<specifics>
## Specific Ideas

- Visual config should use the same Markdown+YAML frontmatter pattern as agent definitions (`.claude/agents/ant/*.md`) so users can edit caste colours and emojis without learning a new format.
- The parity checklist should be structured as a table: Classic behaviour | Go equivalent | Status (MATCH/GAP/DEGRADED) | Verification method | Decision (Classic/Go/Hybrid).
- The extraction audit should be machine-readable (CSV or structured Markdown table) so Phase 155 planners can query it.
- Ruflo (Rust engine + TS control plane + memory + plugin system) is an external reference for hybrid architecture patterns — useful for the architecture boundary doc but not a dependency.

</specifics>

<deferred>
## Deferred Ideas

- Federation / inter-colony coordination — already explicitly deferred in PROJECT.md
- Web UI for event stream — deferred in STATE.md (terminal + NDJSON sufficient for now)
- Vector backend for memory — deferred in STATE.md (file-backed memory sufficient)
- Cross-colony wisdom sharing beyond Hive Brain — deferred in PROJECT.md

</deferred>

---

*Phase: 152-Boundary & Parity*
*Context gathered: 2026-05-22*
