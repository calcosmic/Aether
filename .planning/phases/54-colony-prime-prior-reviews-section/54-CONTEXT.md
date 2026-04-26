# Phase 54: Colony-Prime Prior-Reviews Section - Context

**Gathered:** 2026-04-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Downstream workers see open review findings from prior phases in their context. Colony-prime assembles a `prior-reviews` section from the 7 domain ledgers, capped at 800/400 chars, cached for performance, and omitted entirely when no reviews exist.

</domain>

<decisions>
## Implementation Decisions

### Finding detail level
- **D-01:** Each domain gets one line showing: domain name, open count, and top-severity finding with file/location (e.g., `Security (3 open): HIGH — auth.go:45 bcrypt weakness`)
- **D-02:** Only open (not resolved) findings are included — resolved findings don't help workers
- **D-03:** Show at most 2 findings per domain before truncating with `+N more` to stay within budget

### Cache strategy
- **D-04:** Cache file at `.aether/data/reviews/_summary_cache.json` — stores the assembled prior-reviews text plus per-domain open counts
- **D-05:** Refresh on every colony-prime call (colony-prime runs during build/continue, not hot-path — acceptable cost)
- **D-06:** Cache avoids 7 individual `LoadJSON` calls by reading the cache first; falls back to full read if cache is missing or stale (ledger file mtime newer than cache mtime)

### Domain ordering and trimming
- **D-07:** Domains with HIGH-severity open findings appear first, then MEDIUM, then LOW — severity-first ordering within the section
- **D-08:** When all 7 domains don't fit in the char budget, lower-severity domains truncate to counts-only (e.g., `History (2 open)`), then get dropped entirely
- **D-09:** Use the existing deterministic `domainOrder` array as tiebreaker when severity is equal

### Section placement
- **D-10:** Section name `prior_reviews`, title `Prior Reviews`, priority 8 — placed between user_preferences (7) and pheromones (9) per PRIME-01
- **D-11:** Section omitted entirely when no domain has open findings (no empty placeholder)
- **D-12:** Freshness score based on most recent ledger write timestamp; confirmation score = 1.0 (findings are factual, not predictive)

### Claude's Discretion
- Exact cache file format (flat JSON vs nested)
- String builder formatting details
- Error handling for corrupt cache files
- Whether to include agent name in finding summaries

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Colony-prime assembly
- `cmd/colony_prime_context.go` — Main assembly function, section pattern, budget ranking, existing sections to use as template
- `pkg/colony/` — ContextCandidate, RankContextCandidates, AssessPromptSource used by the ranking system

### Review ledger system
- `cmd/review_ledger.go` — `review-ledger-summary` command reads all 7 domains, shows the data shape available
- `pkg/colony/review_ledger.go` — ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary types with field definitions
- `cmd/review_ledger_test.go` — Test patterns for ledger operations

### Requirements
- `.planning/REQUIREMENTS.md` PRIME-01 through PRIME-05 — The 5 locked requirements for this phase

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `buildColonyPrimeOutput()` in `colony_prime_context.go`: The section assembly pattern is copy-pasteable — each section builds a string, sets priority/scores, appends to `sections` slice
- `review-ledger-summary` in `review_ledger.go`: Already iterates all 7 domains in deterministic order and reads summaries — can be adapted for the cache writer
- `domainOrder` array in `review_ledger.go`: Deterministic domain iteration, reuse directly
- `cache.SessionCache` in `pkg/cache/`: Existing cache layer used by colony-prime for session data

### Established Patterns
- Section struct: `colonyPrimeSection` with name, title, source, content, priority, freshness/confirmation/relevance scores
- Budget trimming: `colony.RankContextCandidates()` handles over-budget trimming automatically — just set priority correctly
- Graceful degradation: Existing sections (hive_wisdom, instincts) skip entirely when empty — follow same pattern
- Protected sections: `protectedSectionPolicy()` for sections that should survive budget trim

### Integration Points
- `sections` slice in `buildColonyPrimeOutput()`: New section appends here, after user_preferences and before blockers
- `colonyPrimeOutput.LogLine`: Update to include review count (e.g., "N review findings")
- `pkg/storage/`: File locking for cache writes (follow pheromone pattern)
- `domainOrder` in `review_ledger.go`: Import or duplicate the deterministic order array

</code_context>

<specifics>
## Specific Ideas

- The section should feel like a quick status board for workers: "Here's what prior reviewers flagged that's still open"
- Format example from requirements: "Security (5 open): HIGH -- bcrypt..., MEDIUM -- auth..."
- Workers should be able to scan it in under 2 seconds — dense but not overwhelming

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 54-colony-prime-prior-reviews-section*
*Context gathered: 2026-04-26*
