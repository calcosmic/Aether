# Project Research Summary

**Project:** Aether v1.9 -- Review Findings Persistence and Domain-Ledger System
**Domain:** Internal feature integration (extending an existing Go CLI framework with file-based state management)
**Researched:** 2026-04-26
**Confidence:** HIGH

## Executive Summary

Aether v1.9 adds a review findings persistence layer to an established Go runtime with 2900+ passing tests. The system needs two things: per-worker outcome reports for continue-review workers (mirroring what build workers already have), and a domain-ledger system where seven review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) persist structured findings that survive `/clear` and accumulate across phases. Colony-prime then injects open findings into downstream worker prompts, closing the loop so Phase N+1 builders see what Phase N reviewers found.

The recommended approach requires zero new external dependencies. Every capability -- JSON persistence with file locking, CLI subcommand registration, deterministic ID generation, colony-prime context injection -- already exists as an established pattern in `cmd/` and `pkg/`. The work is two new files (one types file at ~80 lines, one commands file at ~350 lines) plus targeted modifications to six existing files and 28 agent definition mirrors. The closest analog is the midden system (`pkg/colony/midden.go` + `cmd/midden_cmds.go`), which solves the same problem (structured CRUD over JSON files with atomic writes) and provides a proven template.

The key risks are token budget blowout from the new colony-prime section (mitigated by priority 8 ranking and a hard character cap), agent write-scope escape (mitigated by write-scope guardrails restricting writes to `.aether/data/reviews/` only), and a race condition between concurrent review writes and colony-prime reads (mitigated by using `UpdateJSONAtomically` for writes and a cached summary file for reads). All three risks have straightforward prevention strategies documented in the pitfalls research.

## Key Findings

### Recommended Stack

No new dependencies. The entire v1.9 milestone is built from existing Go standard library packages and the established `pkg/storage.Store`, `github.com/spf13/cobra`, and `pkg/colony` type system already in `go.mod`.

**Core technologies:**
- `pkg/storage.Store` (existing): Atomic JSON read/write with per-file locking -- the foundation for all 7 domain ledger files
- `github.com/spf13/cobra` v1.10.2 (existing): CLI subcommand registration for 4 new ledger commands, following the `midden_cmds.go` pattern
- `cmd/colony_prime_context.go` section system (existing): Additive context sections with priority-based budget trimming, proven by the `medic_health` section
- `crypto/sha256` + `encoding/json` (stdlib): Content-hash deduplication and structured types for ledger entries
- `store.AtomicWrite` (existing): Per-worker `.md` report files, mirroring `writeCodexBuildOutcomeReports()`

### Expected Features

**Must have (table stakes) -- P1:**
- Continue-review worker outcome reports -- review workers currently produce rich findings with no file output; this is an asymmetric gap with build workers
- `Report` field on `codexContinueExternalDispatch` and `codexContinueWorkerFlowStep` -- the structural enabler for report data flow
- `review-ledger-write` / `read` / `summary` subcommands -- programmatic access to ledger data
- Colony-prime `prior-reviews` section at priority 8 -- the primary user-facing value; persistence without injection is pointless
- Agent Write tool for 7 review agents -- agents need to persist findings through the CLI
- Wrapper completion packet `report` field -- documents the data contract between wrappers and runtime
- Agent mirror sync across all 4 surfaces (Claude, OpenCode, Codex, packaging)

**Should have -- P2:**
- `review-ledger-resolve` subcommand -- marking entries as resolved (valuable but not needed for the core loop)
- Status command review findings display -- open finding counts in `aether status`
- Seal/entomb ledger lifecycle -- archiving and cleanup at colony end

**Defer (v2+):**
- Cross-colony ledger sharing via Hive Brain -- findings contain code-specific paths that go stale across repos
- Auto-block on critical findings -- would create conflicting signals with the existing continue-review blocking
- Automatic finding-to-pheromone promotion -- the mapping between "finding" and "action" requires judgment

### Architecture Approach

Review persistence extends four established subsystems without introducing new ones: continue-flow report writing, CLI subcommand registration, colony-prime context injection, and agent tool definitions. The data flow is: review agents call `aether review-ledger-write` per finding; the runtime validates, assigns deterministic IDs (`sec-2-001`), and writes atomically to `reviews/{domain}/ledger.json`; colony-prime reads open findings and injects a compact markdown section into worker prompts at priority 8.

**Major components:**
1. `cmd/review_ledger.go` (NEW): Four cobra subcommands for CRUD operations on domain-ledger entries, plus `pkg/colony/review_ledger.go` for shared types
2. `cmd/codex_continue_finalize.go` (MODIFIED): Two new functions (`writeCodexContinueOutcomeReports`, `renderCodexContinueWorkerOutcomeReport`) plus struct field preservation in `mergeExternalContinueResults`
3. `cmd/colony_prime_context.go` (MODIFIED): New `buildPriorReviewsSection()` function producing a priority-8 section from open ledger entries
4. Agent definitions across 4 surfaces (MODIFIED): 7 agents gain Write tool, findings instructions, and write-scope guardrails

### Critical Pitfalls

1. **Token budget blowout from prior-review injection** -- Cap the section at 800 chars (normal) / 400 (compact), summarize at domain level not individual findings, set priority to 8 so reviews trim before pheromones and blockers
2. **Agent write-scope escape** -- Review agents are currently read-only by design; adding Write requires hook enforcement restricting writes to `.aether/data/reviews/` only, shipped in the same phase as the tool addition
3. **JSON backward compatibility break in merge function** -- `mergeExternalContinueResults` must explicitly copy every new field from dispatch to flow step; add a round-trip test that verifies all fields survive the merge
4. **File locking contention between writes and reads** -- Use `UpdateJSONAtomically` for writes, cache a `summary.json` for colony-prime reads (one file instead of seven), never hold locks on multiple files simultaneously
5. **Stale review data accumulation** -- Seal must archive `reviews/` to chambers; `/ant-init` must clear existing reviews for the new colony; this cannot be deferred

## Implications for Roadmap

Based on research, suggested phase structure (5 phases):

### Phase 1: Continue-Review Worker Outcome Reports
**Rationale:** Simplest change -- extends an existing flow with a pattern that already exists for builds. No new CLI commands, no new data formats. Just struct fields and two functions. Must come first because later phases depend on the `Report` field existing in the continue pipeline.
**Delivers:** Per-worker `.md` files at `build/phase-N/worker-reports/{name}.md` for review workers, matching the build report UX
**Addresses:** Continue-review outcome reports, `Report` field on structs, wrapper completion packet
**Avoids:** JSON backward compatibility break (Pitfall 3) -- round-trip test validates field survival in merge

### Phase 2: Domain-Ledger CRUD Subcommands
**Rationale:** Independent infrastructure that downstream phases (colony-prime injection, agent writes) depend on. Getting CRUD right and tested first means consumers have a stable API.
**Delivers:** `review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve` subcommands with deterministic IDs, domain validation, and atomic writes
**Uses:** `pkg/storage.Store` for atomic JSON, `crypto/sha256` for dedup, `pkg/colony/review_ledger.go` for types
**Implements:** Domain-ledger storage layer
**Avoids:** File locking contention (Pitfall 4) -- uses `UpdateJSONAtomically` and sequential domain processing

### Phase 3: Colony-Prime Prior-Reviews Section
**Rationale:** Depends on Phase 2 (must be able to read ledger files). This is the primary user-facing value -- the moment when persisted findings actually inform downstream workers.
**Delivers:** `buildPriorReviewsSection()` at priority 8, reading open findings from all domain ledgers, compact markdown injection into worker context
**Avoids:** Token budget blowout (Pitfall 1) -- character cap, domain-level summary, priority-based trimming

### Phase 4: Agent Definition Updates
**Rationale:** Depends on Phase 2 (agents need `review-ledger-write` to exist as a target). Highest-touch change at 28 file edits (7 agents x 4 surfaces), but low-risk mechanically.
**Delivers:** Write tool added to 7 review agents, findings instructions, write-scope guardrails, mirror sync verified
**Avoids:** Agent write-scope escape (Pitfall 2) -- guardrails and hook enforcement ship in this phase; mirror sync drift (Pitfall 6) -- parity verified as acceptance criteria

### Phase 5: Lifecycle Integration
**Rationale:** Touches existing lifecycle commands (seal, entomb, status) that already work. Must come last because it handles cleanup of data created by earlier phases.
**Delivers:** Seal archives `reviews/` to chambers, `/ant-init` clears stale reviews, `aether status` shows open finding counts
**Avoids:** Stale review data accumulation (Pitfall 5) -- archive and cleanup prevent cross-colony contamination

### Phase Ordering Rationale

- Phase 1 is first because it adds struct fields that Phase 2-5 all depend on indirectly (the `Report` field on continue dispatch structs)
- Phase 2 is second because Phases 3 and 4 both need ledger subcommands to exist before they can consume or target them
- Phase 3 and Phase 4 are independent of each other (colony-prime reads ledgers; agents write to ledgers), but both need Phase 2 complete. Phase 3 is ordered first because it delivers user value sooner
- Phase 5 is last because it handles lifecycle cleanup of data produced by Phases 1-4
- The grouping separates: struct changes (Phase 1), new infrastructure (Phase 2), read-side integration (Phase 3), write-side integration (Phase 4), lifecycle cleanup (Phase 5)

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** Colony-prime budget tuning -- the 800-char cap and priority 8 placement need empirical validation with real finding volumes; consider a budget test as part of acceptance criteria
- **Phase 4:** Codex TOML agent translation -- the Codex mirror is a different format; verify the TOML `tools` field supports Write the same way markdown frontmatter does

Phases with standard patterns (skip research-phase):
- **Phase 1:** Exact structural mirror of `writeCodexBuildOutcomeReports` -- well-documented, line numbers verified
- **Phase 2:** Exact structural mirror of `midden_cmds.go` -- CRUD over JSON with cobra, proven by 10 existing subcommands
- **Phase 5:** Minor additions to existing lifecycle commands -- straightforward archival and display

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All patterns verified by reading source code directly. Zero new dependencies confirmed via `go.mod` review. Every capability (atomic write, file locking, CLI registration, section injection) has a proven template in the existing codebase. |
| Features | HIGH | Feature list derived from PROJECT.md v1.9 specification. Dependency chain between features is clear and verified against actual code locations (struct fields, function signatures, file paths). Anti-features well-documented with clear reasoning. |
| Architecture | HIGH | Data flow mapped end-to-end. Integration points verified at specific line numbers. Component responsibility matrix complete. Anti-patterns identified with correct alternatives. |
| Pitfalls | HIGH | All 7 pitfalls derived from direct codebase analysis with specific file/line references. Prevention strategies are concrete (character caps, hook enforcement, merge function updates). Recovery costs estimated. Pitfall-to-phase mapping provided. |

**Overall confidence:** HIGH

### Gaps to Address

- **Finalize-vs-agent write ownership:** The architecture research recommends making finalize own persistence (agents return findings in completion payload, finalize writes to ledger). The pitfalls research notes that agents could write directly via CLI invocation. This decision must be settled before Phase 4 planning. Recommendation: finalize owns writes (single source of truth, no race conditions).
- **Codex TOML tool field format:** The architecture research assumes the Codex TOML `tools` field accepts `Write` in a comma-separated list matching markdown frontmatter. This needs verification against `pkg/llm/config.go` Codex parsing during Phase 4.
- **Summary caching strategy:** The pitfalls research recommends a cached `summary.json` for colony-prime reads. The architecture research does not include this file in the data model. This should be added to Phase 2 (ledger subcommands) as an optimization, or deferred to Phase 5 if performance proves acceptable with 7 direct reads.
- **Hook system extension:** Write-scope enforcement for review agents requires either extending `protectedHookWriteReason` or adding a new scope-check function. The exact implementation path needs a brief investigation during Phase 4.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `cmd/codex_build.go:1125-1219` -- build outcome report pattern (template for continue reports)
- Direct codebase analysis of `cmd/midden_cmds.go` -- CRUD subcommand registration pattern (template for ledger commands)
- Direct codebase analysis of `cmd/colony_prime_context.go:123-527` -- section assembly, priorities, budget system
- Direct codebase analysis of `pkg/storage/storage.go` -- `AtomicWrite`, `SaveJSON`, `LoadJSON`, file locking
- Direct codebase analysis of `cmd/codex_continue_finalize.go` -- continue finalize flow, `mergeExternalContinueResults`
- Direct codebase analysis of `cmd/hive.go` -- LRU eviction, directory creation, hub-scoped storage
- Direct codebase analysis of `.claude/agents/ant/aether-gatekeeper.md` -- agent definition structure, read-only constraints
- `.planning/PROJECT.md` -- v1.9 milestone specification and feature requirements

### Secondary (MEDIUM confidence)
- `pkg/colony/context_ranking.go` -- ranking algorithm details (priority and budget trimming behavior)
- `cmd/hook_cmds.go` -- write protection and path normalization (for write-scope enforcement)
- `pkg/colony/sanitize.go` -- content sanitization patterns (for finding description safety)

### Tertiary (LOW confidence)
- Codex TOML agent format specifics -- assumed to support comma-separated `tools` field matching markdown; needs verification
- Colony-prime budget behavior with large prior-reviews sections -- theoretical analysis; needs empirical testing at Phase 3

---
*Research completed: 2026-04-26*
*Ready for roadmap: yes*
