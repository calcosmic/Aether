# Phase 57: QUEEN.md Pipeline Fix - Context

**Gathered:** 2026-04-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix the QUEEN.md wisdom pipeline so that the global QUEEN.md (~/.aether/QUEEN.md) works like CLAUDE.md — persistent instructions that shape every worker conversation. No duplicate entries in QUEEN.md or hive/wisdom.json. High-confidence instincts promote automatically at seal. Global QUEEN.md wisdom reaches all workers via colony-prime.

</domain>

<decisions>
## Implementation Decisions

### Dedup Strategy
- **D-01:** `appendEntriesToQueenSection` uses normalized matching — strip date patterns, timestamps, and normalize whitespace before comparing. This catches semantic duplicates where the same wisdom gets promoted multiple times with different dates attached (the root cause of ~270 duplicate lines)
- **D-02:** `queen-seed-from-hive` filters entries already present in QUEEN.md using the same normalized matching, and reports count of new vs skipped entries

### Global QUEEN.md Injection
- **D-03:** Colony-prime reads the entire global QUEEN.md (~/.aether/QUEEN.md) as a single block — same model as CLAUDE.md. No section-by-section parsing. Workers see the full content.
- **D-04:** Local repo QUEEN.md wisdom continues to be read separately via existing `readQUEENMd` function (repo-specific wisdom)
- **D-05:** This supersedes the earlier "separate GLOBAL QUEEN WISDOM section" approach — the file is read as-is, not injected as a parsed section

### Seal Auto-Promotion
- **D-06:** `/ant-seal` wrapper markdown includes instructions to loop through instincts with confidence >= 0.8 and call `queen-promote-instinct` for each one. Not in the Go runtime.
- **D-07:** `queen-promote-instinct` must write to global `~/.aether/QUEEN.md` (using `hubStore()`), not just the local colony store. Currently uses local store only.

### Data Cleanup
- **D-08:** Remove test junk data from `~/.aether/hive/wisdom.json` (domain: "test", text: "<repo> wisdom")
- **D-09:** Clean ~270 duplicate `<repo> wisdom` lines from `~/.aether/QUEEN.md` using the new normalized dedup (run appendEntriesToQueenSection with dedup enabled, or a one-time cleanup command)

### Claude's Discretion
- Exact normalized matching regex (what date/timestamp formats to strip)
- Priority ordering of the global QUEEN.md block in colony-prime's trim order
- Whether to keep or remove the existing `readQUEENMd` function for global QUEEN.md (local QUEEN.md still uses it)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### QUEEN.md Pipeline Code
- `cmd/queen.go` — Core QUEEN.md operations: `appendEntriesToQueenSection` (line 512), `queen-promote-instinct` (line 226), `queen-seed-from-hive` (line ~310)
- `cmd/colony_prime_context.go` — Colony-prime prompt assembly, global+local QUEEN.md reading (lines 558-609)
- `cmd/context.go` — `readQUEENMd` function (line 1468), only reads Wisdom + Patterns sections currently
- `pkg/memory/queen.go` — `QueenService`, `WriteEntry` with existing dedup logic
- `pkg/memory/promote.go` — `PromoteService`, instinct promotion pipeline

### Seal Lifecycle
- `.claude/commands/ant/seal.md` — Seal wrapper markdown (where auto-promotion instructions go)
- `cmd/codex_workflow_cmds.go` — Go runtime seal command (if needed for reference)

### Requirements
- `.planning/REQUIREMENTS.md` — QUEE-01 through QUEE-07 requirements with acceptance criteria
- `.planning/ROADMAP.md` — Phase 57 success criteria (lines 127-137)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/memory/queen.go` `QueenService.WriteEntry` — already has content-level dedup logic (exact match check before writing). May be reusable for the normalized dedup.
- `pkg/storage/` — File locking and JSON store already handles hub vs local stores
- `hubStore()` helper in `cmd/queen.go` — already resolves to `~/.aether/` for global operations
- `readUserPreferences` in colony_prime_context.go — already reads from global QUEEN.md for preferences

### Established Patterns
- Colony-prime assembles context as a list of `colonyPrimeSection` structs, each with priority, freshness, and relevance scores
- Sections are trimmed by priority when over budget — new sections need appropriate priority values
- Wrapper markdown handles ceremony (seal, continue), Go runtime handles state mutations

### Integration Points
- `appendEntriesToQueenSection` is called by `queen-promote`, `queen-seed-from-hive`, and `queen-promote-instinct` — dedup fix benefits all three
- Colony-prime `buildColonyPrimeOutput` assembles all sections — new global QUEEN.md block gets added here
- Seal wrapper needs auto-promotion loop — the wrapper already calls hive-promote, this follows the same pattern

</code_context>

<specifics>
## Specific Ideas

- Global QUEEN.md should work like CLAUDE.md — persistent instructions shaping every worker conversation, not a write-only dump
- Current global QUEEN.md is "dead text" for wisdom — only user preferences are read, the Wisdom/Patterns/Philosophies/Anti-Patterns sections are never extracted
- The ~270 duplicate lines were caused by `appendEntriesToQueenSection` having zero dedup — the same learning got appended repeatedly with different timestamps

</specifics>

<deferred>
## Deferred Ideas

- `/ant-lay-eggs` should ask users about their project and pre-fill QUEEN.md with relevant context — this is a new capability for a future phase, not a pipeline fix

</deferred>

---

*Phase: 57-queen-md-pipeline-fix*
*Context gathered: 2026-04-26*
