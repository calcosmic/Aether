# Phase 42: CI Context Assembly - Context

**Gathered:** 2026-03-31 (discuss mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

CI agents get machine-readable colony context via `pr-context` subcommand, replacing interactive colony-prime for automated workflows. The design doc at `.aether/docs/ci-context-assembly-design.md` defines the complete specification — implementation follows it closely with three additions from discussion.

</domain>

<decisions>
## Implementation Decisions

### Midden Data Inclusion
- **D-01:** pr-context output includes a `midden` section with recent failure entries and cross-PR pattern data. This fills a gap in the original design doc (written before Phase 41 implemented midden collection). CI agents can warn about recurring issues.
- **D-02:** Midden section is classified as VOLATILE (always read fresh from `.aether/data/midden/midden.json`). Not cacheable — failures are added every build.
- **D-03:** Midden entries in pr-context output are bounded (top 10 most recent, or all entries from last 7 days — whichever is smaller) to stay within token budget.

### Cache System
- **D-04:** Implement the full TTL-based cache system as specified in the design doc (Section 4). Cache QUEEN.md (global/local), hive wisdom, and eternal memory with mtime-based invalidation. Cache storage at `.aether/data/pr-context-cache.json` (gitignored, branch-local).
- **D-05:** Cache writes use `acquire_lock` from file-lock.sh (same pattern as other data files). Cache reads are lock-free (read-only).
- **D-06:** TTL values from design doc: QUEEN.md 1 hour, hive/eternal 2 hours. Evict stale entries on each pr-context call.

### Budget Enforcement Refactor
- **D-07:** Extract `_budget_enforce()` shared function from colony-prime (pheromone.sh lines 1388-1492). Both colony-prime and pr-context call it with different `max_chars` values (8K/4K vs 6K/3K).
- **D-08:** Trim order stays identical to colony-prime (rolling-summary first, blockers never). This is verified by existing tests — any change breaks the contract.
- **D-09:** The refactor must not change colony-prime's output. Existing colony-prime tests must pass unchanged after extraction.

### Error Policy
- **D-10:** pr-context NEVER hard-fails. Every source has a fallback chain as specified in design doc Section 5. Missing QUEEN.md returns empty wisdom (not exit 1 like colony-prime).
- **D-11:** All fallbacks are logged in the `fallbacks_used` output array and `warnings` array.

### Output Schema
- **D-12:** Output matches design doc Section 3.2 schema with one addition: `midden` section alongside existing `signals`, `colony_state`, `blockers`, etc.
- **D-13:** Structured signal arrays (redirects, focus, feedback) as typed JSON — not just formatted text. This is a key improvement over colony-prime.

### Integration Points
- **D-14:** Wire pr-context into `/ant:continue` (post-verify review context) and `/ant:run` (before review cycles). CI pipeline integration is out of scope (Phase 43/44 handles that).
- **D-15:** Dispatch entry: `pr-context) _pr_context "$@" ;;` in aether-utils.sh case statement.

### Claude's Discretion
- Exact midden entry format in JSON output (count + recent items, or structured categories)
- Cache eviction granularity (per-entry vs full-cache clear)
- Whether to add `--section` flag to pr-context for requesting specific sections only
- Exact placement of _budget_enforce() extraction (inline in pheromone.sh vs separate utils/ module)

</decisions>

<specifics>
## Specific Ideas

- The design doc is the primary spec — follow it closely. The three decisions above (midden, cache, refactor) are additions/modifications to the design doc, not replacements.
- Midden data should include cross-PR analysis results (systemic/critical classifications) since Phase 41 already computes them.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Primary Design
- `.aether/docs/ci-context-assembly-design.md` — Complete pr-context specification: output schema, fallback chains, cache layer, token budgets, edge cases, integration points. This is the authoritative design doc.

### Existing Implementation (Reference)
- `.aether/utils/pheromone.sh` — colony-prime function (lines 737-1553), budget trimming (lines 1388-1492), _extract_wisdom(), _filter_wisdom_entries()
- `.aether/aether-utils.sh` — context-capsule subcommand (lines 4172-4368), dispatch case statement

### Supporting Design Docs
- `.aether/docs/state-contract-design.md` — Branch-local vs hub-global state, worktree isolation rules
- `.aether/docs/pheromone-propagation-design.md` — Pheromone snapshot-inject protocol (Phase 40)

### Midden System (New in Phase 41)
- `.aether/utils/midden.sh` — midden-collect, midden-cross-pr-analysis functions (Phase 41 additions)
- `.planning/phases/41-midden-collection/41-01-SUMMARY.md` — What Phase 41 built (4 subcommands, 13 tests)

### Infrastructure
- `.aether/utils/file-lock.sh` — acquire_lock/release_lock for cache file locking
- `.aether/utils/state-api.sh` — _state_mutate for COLONY_STATE.json reads

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `colony-prime` in pheromone.sh — complete reference implementation for context assembly. pr-context follows the same pattern but outputs structured JSON instead of prompt_section string.
- `_extract_wisdom()` and `_filter_wisdom_entries()` — existing QUEEN.md parsing functions, reusable directly
- `context-capsule` subcommand — existing bounded snapshot of colony state, callable via `bash aether-utils.sh context-capsule`
- `hive-read` subcommand — existing cross-colony wisdom retrieval with domain scoping
- `pheromone-prime` subcommand — existing pheromone signal assembly
- `acquire_lock`/`release_lock` — existing file locking for cache writes

### Established Patterns
- Subcommand pattern: function in utils/*.sh + dispatch case in aether-utils.sh + help JSON entry
- json_ok() for structured output
- Budget trimming with priority-ordered section removal
- Fallback chains with warning logging (colony-prime does this for pheromones, hive)
- Token budget enforcement (8K/4K in colony-prime, 6K/3K in pr-context)

### Integration Points
- pheromone.sh: pr-context function lives here, near colony-prime
- aether-utils.sh: dispatch entry + help JSON
- continue-advance.md playbook: integration point for review context generation
- build-verify.md playbook: integration point for /ant:run review cycles

</code_context>

<deferred>
## Deferred Ideas

- CI pipeline workflow files (GitHub Actions) — not in scope, Phase 44 handles release/CI integration
- `--section` flag for requesting specific pr-context sections only — useful but not needed for v2.7
- pr-context for OpenCode agents — deferred to future milestone

</deferred>

---
*Phase: 42-ci-context-assembly*
*Context gathered: 2026-03-31*
