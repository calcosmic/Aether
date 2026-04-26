# Phase 54: Colony-Prime Prior-Reviews Section - Research

**Researched:** 2026-04-26
**Domain:** Go CLI (Aether colony-prime context assembly, review ledger integration)
**Confidence:** HIGH

## Summary

This phase adds a `prior_reviews` section to colony-prime's context assembly. The section reads open findings from the 7 domain review ledgers (security, quality, performance, resilience, testing, history, bugs), formats them into a compact status board, and injects them into worker context at priority 8. A file-based cache avoids redundant 7-file reads on every colony-prime call.

The implementation is purely additive -- it extends the existing `buildColonyPrimeOutput()` function in `cmd/colony_prime_context.go` with a new section that follows the exact same pattern as the 10+ sections already there. The review ledger types and summary command from Phase 53 provide all the data primitives needed. The cache strategy reuses the existing `SessionCache` from `pkg/cache/` with mtime-based staleness detection.

No new dependencies, no new packages, no new CLI commands. The work is confined to `cmd/colony_prime_context.go` (section assembly), `cmd/context_weighting.go` (relevance score + protected policy), and a new test file.

**Primary recommendation:** Implement the section assembly in `buildColonyPrimeOutput()`, use `store.LoadJSON` with a custom mtime check for cache invalidation (simpler than `SessionCache` for this single-purpose cache), and add `"prior_reviews"` to the `sectionRelevanceScore` and `protectedSectionPolicy` switch statements.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Each domain gets one line showing: domain name, open count, and top-severity finding with file/location (e.g., `Security (3 open): HIGH -- auth.go:45 bcrypt weakness`)
- **D-02:** Only open (not resolved) findings are included -- resolved findings don't help workers
- **D-03:** Show at most 2 findings per domain before truncating with `+N more` to stay within budget
- **D-04:** Cache file at `.aether/data/reviews/_summary_cache.json` -- stores the assembled prior-reviews text plus per-domain open counts
- **D-05:** Refresh on every colony-prime call (colony-prime runs during build/continue, not hot-path -- acceptable cost)
- **D-06:** Cache avoids 7 individual `LoadJSON` calls by reading the cache first; falls back to full read if cache is missing or stale (ledger file mtime newer than cache mtime)
- **D-07:** Domains with HIGH-severity open findings appear first, then MEDIUM, then LOW -- severity-first ordering within the section
- **D-08:** When all 7 domains don't fit in the char budget, lower-severity domains truncate to counts-only (e.g., `History (2 open)`), then get dropped entirely
- **D-09:** Use the existing deterministic `domainOrder` array as tiebreaker when severity is equal
- **D-10:** Section name `prior_reviews`, title `Prior Reviews`, priority 8 -- placed between user_preferences (7) and pheromones (9) per PRIME-01
- **D-11:** Section omitted entirely when no domain has open findings (no empty placeholder)
- **D-12:** Freshness score based on most recent ledger write timestamp; confirmation score = 1.0 (findings are factual, not predictive)

### Claude's Discretion
- Exact cache file format (flat JSON vs nested)
- String builder formatting details
- Error handling for corrupt cache files
- Whether to include agent name in finding summaries

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PRIME-01 | Colony-prime assembles a `prior-reviews` section at priority 8 (between user_preferences at 7 and pheromones at 9) | Section assembly pattern fully documented in colony_prime_context.go; priority 8 slot identified (shared with clarified_intent) |
| PRIME-02 | Prior-reviews section shows open findings per domain with severity and file/location summary | ReviewLedgerEntry has all needed fields (Severity, File, Line, Description); filter by Status=="open" is straightforward |
| PRIME-03 | Prior-reviews section is capped at 800 chars (normal) / 400 chars (compact) | Budget system in RankContextCandidates handles char budgets; section content can be pre-trimmed before ranking |
| PRIME-04 | Prior-reviews section gracefully degrades when no review ledgers exist (omitted entirely) | Pattern: existing sections (hive_wisdom, instincts) check `len > 0` before appending; same approach works here |
| PRIME-05 | Section reads from cached summary file for performance (not 7 direct ledger reads) | Cache file at `.aether/data/reviews/_summary_cache.json`; mtime comparison with ledger files for staleness |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Review ledger data read | Go runtime (pkg/storage) | -- | Ledger files are local JSON; colony-prime reads them during context assembly |
| Cache file read/write | Go runtime (pkg/storage) | -- | Cache is a local JSON file under `.aether/data/reviews/` |
| Section assembly | Go runtime (cmd/colony_prime_context.go) | -- | All colony-prime sections are assembled in buildColonyPrimeOutput() |
| Budget trimming | Go runtime (pkg/colony/context_ranking.go) | -- | RankContextCandidates handles over-budget trimming automatically |
| Severity sorting/formatting | Go runtime (cmd/) | -- | Pure string manipulation within the section builder |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | (project go.mod) | strings, fmt, os, time, encoding/json | Project uses zero external dependencies for core runtime |
| github.com/calcosmic/Aether/pkg/storage | (local) | LoadJSON, SaveJSON, BasePath, AtomicWrite | Established pattern for all JSON file I/O |
| github.com/calcosmic/Aether/pkg/colony | (local) | ReviewLedgerFile, ReviewLedgerEntry, ReviewSeverity, ContextCandidate | Types from Phase 53 |
| github.com/calcosmic/Aether/pkg/cache | (local) | SessionCache (reference only, see recommendation) | Mtime-based caching pattern |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom cache at `_summary_cache.json` | SessionCache.Load on each ledger file | SessionCache stores per-file parsed data; the prior-reviews section needs an assembled summary string, not 7 separate parsed objects. A dedicated cache file is simpler and avoids coupling to SessionCache internals. |
| `domainOrder` from `cmd/review_ledger.go` | `ValidReviewDomains` from `pkg/colony/review_ledger.go` | `ValidReviewDomains` is a map (no order). `domainOrder` in cmd/ is a slice with deterministic order. Either duplicate the array in the new code or import from cmd (which pkg/colony cannot do). Recommendation: define the order array in `pkg/colony/review_ledger.go` and use it from both cmd files. |

**Installation:**
None -- this phase uses only existing packages.

## Architecture Patterns

### System Architecture Diagram

```
colony-prime call
       |
       v
buildColonyPrimeOutput()
       |
       +--[existing sections: state, instincts, decisions, ...]
       |
       +-- NEW: buildPriorReviewsSection()
       |         |
       |         +-- Read _summary_cache.json
       |         |     |
       |         |     +-- Cache hit AND fresh? --> return cached text
       |         |     |
       |         |     +-- Cache miss OR stale? --> fall through
       |         |
       |         +-- For each domain in domainOrder:
       |         |     Read ledger.json (or from cache)
       |         |     Filter entries: Status == "open"
       |         |     Extract top-severity open findings
       |         |
       |         +-- Sort domains by max-severity (HIGH first)
       |         |     Tiebreak: domainOrder position
       |         |
       |         +-- Format: "Domain (N open): SEV -- file:line desc"
       |         |     Max 2 findings per domain, then "+N more"
       |         |     Budget: 800 (normal) / 400 (compact)
       |         |
       |         +-- Write _summary_cache.json with assembled text
       |         |
       |         +-- Return section (or empty if no open findings)
       |
       +-- Append to sections[] slice
       |
       +-- RankContextCandidates() handles budget trimming
       |
       v
  Worker context output
```

### Recommended Project Structure

No new files needed except tests. All implementation goes in existing files:

```
cmd/
  colony_prime_context.go    -- buildPriorReviewsSection() function + section insertion in buildColonyPrimeOutput()
  context_weighting.go       -- Add "prior_reviews" to sectionRelevanceScore() switch
  colony_prime_context_test.go -- Tests for the new section
pkg/colony/
  review_ledger.go           -- Add DomainOrder []string (if not duplicating from cmd/)
```

### Pattern 1: Colony-Prime Section Assembly

**What:** Each section builds a string with `strings.Builder`, sets scores, and appends to the `sections` slice.

**When to use:** Every new colony-prime section follows this pattern.

**Example (from existing code, pheromones section):**
```go
// Source: cmd/colony_prime_context.go lines 196-224
var phSB strings.Builder
phSB.WriteString("## Pheromone Signals\n\n")
phSB.WriteString(colonyLifecycleSignalContext(state))
phSB.WriteString("\n\n")
for _, sig := range activeSignals {
    text := extractText(sig.Content)
    if text == "" {
        continue
    }
    phSB.WriteString(fmt.Sprintf("- [%s] %s\n", sig.Type, text))
}
if strings.TrimSpace(phSB.String()) != "" {
    // ... append section with scores ...
    sections = append(sections, colonyPrimeSection{
        name:              "pheromones",
        title:             "Pheromone Signals",
        source:            filepath.Join(store.BasePath(), "pheromones.json"),
        content:           phSB.String(),
        priority:          9,
        freshnessScore:    latestFreshnessScore(now, 0.85, signalTimestamps...),
        confirmationScore: confidenceScoreFromSignals(activeSignals, now),
        relevanceScore:    sectionRelevanceScore("pheromones"),
        protected:         signalsProtected,
        preserveReason:    signalsPreserveReason,
    })
}
```

### Pattern 2: Graceful Degradation (Omit When Empty)

**What:** Sections that have no data are simply not appended to the `sections` slice. The ranking system never sees them.

**When to use:** For any section that may have no data (hive_wisdom, instincts, user_preferences, etc.)

**Example:**
```go
// Source: cmd/colony_prime_context.go lines 330-346
hiveEntries := readHiveWisdomEntries(hubDir, 5, &fallbacks)
hiveLines := buildHiveWisdomLines(hiveEntries)
if len(hiveLines) > 0 {
    // ... only append when there's data ...
    sections = append(sections, colonyPrimeSection{...})
}
// No else branch -- section simply doesn't appear
```

### Pattern 3: Cache with Mtime Validation

**What:** The `SessionCache` in `pkg/cache/` validates freshness by comparing the source file's mtime against the cached mtime. If they differ, the cache entry is stale.

**When to use:** For the prior-reviews cache, we need to check 7 ledger files against one cache file. The approach is:
1. Read cache file mtime
2. For each of the 7 domain ledger paths, check if its mtime > cache mtime
3. If any is newer, cache is stale -- rebuild
4. Otherwise, return cached text

**Why not SessionCache directly:** SessionCache caches individual file parse results keyed by filename. The prior-reviews section needs a single assembled summary string that aggregates data from 7 files. A dedicated cache file is cleaner.

### Anti-Patterns to Avoid
- **Don't add a new CLI command:** The section is assembled internally by `buildColonyPrimeOutput()`, not exposed as a standalone command. The existing `review-ledger-summary` command already serves the "show me all domains" use case.
- **Don't put cache logic in pkg/colony/:** The cache is a cmd-level concern (it uses `store.BasePath()` and file mtime checks). Keep it in `cmd/`.
- **Don't make the section protected:** Prior reviews are informative, not blocking. Workers don't need them to function. Only blockers, pheromones, state, user preferences, and clarified intent are protected.
- **Don't duplicate domainOrder:** The array exists in `cmd/review_ledger.go`. Either import it or move it to `pkg/colony/review_ledger.go` so both cmd files can use it without circular imports.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON file I/O with locking | Raw os.ReadFile/WriteFile | `store.LoadJSON()` / `store.SaveJSON()` | Already uses AtomicWrite with file locking (follow pheromone pattern) |
| mtime comparison | Manual os.Stat + time comparison | `os.Stat()` from stdlib (this is trivial enough to do inline) | Mtime check is 3 lines of code, not worth abstracting |
| Budget trimming | Custom char-counting with manual truncation | `colony.RankContextCandidates()` | Already handles over-budget trimming with score-based prioritization |
| Severity comparison | String comparison on severity names | `pkg/colony.ReviewSeverity` constants | HIGH > MEDIUM > LOW ordering defined in the type constants |

**Key insight:** This phase is 90% assembly work using existing primitives. The review ledger types, storage layer, cache infrastructure, and ranking system are all battle-tested from Phases 52-53. The new code is essentially a formatting function.

## Common Pitfalls

### Pitfall 1: Priority 8 Conflict with clarified_intent
**What goes wrong:** Both `clarified_intent` and `prior_reviews` have priority 8. If the ranking system uses priority as a primary sort key, sections could appear in arbitrary order.
**Why it happens:** The `sortRankedContextCandidates` function sorts by `Score.Total` first, then uses `PriorityHint` as a tiebreaker (line 192-194 of context_ranking.go). Since priority is only a tiebreaker, two sections with the same priority will sort by their computed score.
**How to avoid:** This is fine. The ranking system differentiates by score (trust * 0.20 + freshness * 0.20 + confirmation * 0.20 + relevance * 0.40). Prior reviews will have different freshness/confirmation/relevance scores than clarified intent, so they will sort differently.
**Warning signs:** If tests show both sections appearing in random order, check that relevance scores are distinct.

### Pitfall 2: Cache Staleness with Ledger Writes
**What goes wrong:** A review-ledger-write happens during a build, then colony-prime reads a stale cache and misses the new findings.
**Why it happens:** Cache mtime is checked against ledger file mtimes. If the cache was written before the ledger, the cache is stale and will be rebuilt. But if colony-prime runs within the same process invocation (e.g., build-wave calls colony-prime multiple times), the in-memory `SessionCache` may mask the staleness.
**How to avoid:** The prior-reviews cache at `_summary_cache.json` is a separate file from SessionCache. Each colony-prime call should check ledger file mtimes against the cache file mtime using `os.Stat()`, which always reads from disk. This bypasses any SessionCache staleness.
**Warning signs:** Workers report stale review counts after a review-ledger-write in the same build.

### Pitfall 3: Description Truncation Breaking Readability
**What goes wrong:** Finding descriptions are long (50-200 chars) and the 800/400 char budget fills up after 2-3 domains.
**Why it happens:** D-01 says each domain gets "top-severity finding with file/location" but doesn't specify description truncation length.
**How to avoid:** Truncate descriptions to ~60 chars with `...` suffix. The format `HIGH -- file.go:42 short description here...` keeps each finding under ~80 chars. With 2 findings per domain and 7 domains, that's ~1120 chars -- which exceeds 800, so D-08's truncation-to-counts-only will naturally activate for lower-severity domains.
**Warning signs:** Section consistently exceeds 800 chars in normal mode.

### Pitfall 4: Forgetting to Add Section to sectionRelevanceScore
**What goes wrong:** New section gets default relevance score of 0.25 (from the `default` case), making it rank lower than expected.
**Why it happens:** The `sectionRelevanceScore()` switch in `cmd/context_weighting.go` has no case for `"prior_reviews"`.
**How to avoid:** Add `"prior_reviews"` to the switch statement with an appropriate score (recommend 0.70 -- review findings are actionable and relevant to current work).
**Warning signs:** Prior reviews section gets trimmed before less relevant sections.

### Pitfall 5: File Locking on Cache Write
**What goes wrong:** Two concurrent colony-prime calls write to `_summary_cache.json` simultaneously, corrupting the file.
**Why it happens:** If colony-prime is called from parallel workers, they could race on the cache file.
**How to avoid:** Use `store.SaveJSON()` which uses `AtomicWrite` with file locking. Or simply don't write the cache -- since colony-prime refreshes on every call per D-05, the cache is an optimization for the 7-file read, not for correctness. A corrupted cache just means a rebuild next time.
**Warning signs:** JSON parse errors when reading `_summary_cache.json`.

## Code Examples

### Cache Structure (Claude's Discretion)

```go
// Recommended: flat JSON cache file
type priorReviewsCache struct {
    Text          string            `json:"text"`           // The assembled section content
    DomainCounts  map[string]int    `json:"domain_counts"`  // per-domain open count
    TotalOpen     int               `json:"total_open"`     // total open findings across all domains
    CacheWriteAt  string            `json:"cache_write_at"` // RFC3339 timestamp
}
```

### Section Assembly (Core Logic)

```go
// Source: derived from cmd/colony_prime_context.go section pattern
func buildPriorReviewsSection(store *storage.Store, compact bool) (colonyPrimeSection, bool) {
    // 1. Check cache
    cachePath := "reviews/_summary_cache.json"
    var cache priorReviewsCache
    cacheFresh := true

    if err := store.LoadJSON(cachePath, &cache); err == nil {
        // Check if any ledger file is newer than cache
        cacheStat, _ := os.Stat(filepath.Join(store.BasePath(), cachePath))
        if cacheStat != nil {
            for _, domain := range domainOrder {
                ledgerPath := filepath.Join(store.BasePath(), "reviews", domain, "ledger.json")
                ledgerStat, err := os.Stat(ledgerPath)
                if err == nil && ledgerStat.ModTime().After(cacheStat.ModTime()) {
                    cacheFresh = false
                    break
                }
            }
        }
    } else {
        cacheFresh = false
    }

    if cacheFresh && cache.Text != "" {
        // Return cached section (reconstruct colonyPrimeSection)
        // ...
    }

    // 2. Read all 7 ledgers, collect open findings per domain
    type domainFindings struct {
        domain   string
        open     []colony.ReviewLedgerEntry
        maxSev   colony.ReviewSeverity
    }
    var domains []domainFindings
    for _, d := range domainOrder {
        var lf colony.ReviewLedgerFile
        if err := store.LoadJSON(fmt.Sprintf("reviews/%s/ledger.json", d), &lf); err != nil {
            continue // no ledger for this domain
        }
        var openEntries []colony.ReviewLedgerEntry
        var maxSev colony.ReviewSeverity
        for _, e := range lf.Entries {
            if e.Status == "open" {
                openEntries = append(openEntries, e)
                if severityRank(e.Severity) > severityRank(maxSev) {
                    maxSev = e.Severity
                }
            }
        }
        if len(openEntries) > 0 {
            domains = append(domains, domainFindings{domain: d, open: openEntries, maxSev: maxSev})
        }
    }

    if len(domains) == 0 {
        return colonyPrimeSection{}, false // D-11: omit entirely
    }

    // 3. Sort domains by max-severity (HIGH first), tiebreak by domainOrder
    sort.SliceStable(domains, func(i, j int) bool {
        if domains[i].maxSev != domains[j].maxSev {
            return severityRank(domains[i].maxSev) > severityRank(domains[j].maxSev)
        }
        return domainPosition(domains[i].domain) < domainPosition(domains[j].domain)
    })

    // 4. Format and budget-cap
    budget := 800
    if compact {
        budget = 400
    }
    // ... format with D-01, D-03, D-08 rules ...

    // 5. Write cache
    store.SaveJSON(cachePath, cache)

    // 6. Return section
    // ...
}
```

### Severity Rank Helper

```go
func severityRank(s colony.ReviewSeverity) int {
    switch s {
    case colony.ReviewSeverityHigh:
        return 4
    case colony.ReviewSeverityMedium:
        return 3
    case colony.ReviewSeverityLow:
        return 2
    case colony.ReviewSeverityInfo:
        return 1
    default:
        return 0
    }
}
```

### Formatting Example (D-01)

```
## Prior Reviews

- Security (3 open): HIGH -- auth.go:45 bcrypt weakness, MEDIUM -- crypto.go:12 weak hash
- Quality (5 open): HIGH -- handler.go:89 missing error check +3 more
- Performance (2 open): MEDIUM -- db.go:201 N+1 query
- Testing (1 open): LOW -- utils_test.go:15 skipped assertion
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No review context for workers | Prior-reviews section in colony-prime | Phase 54 | Workers can see open findings from prior phases |

**No deprecated patterns in scope.** This phase adds new functionality to an existing, well-tested system.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `domainOrder` can be shared between `cmd/review_ledger.go` and the new code without circular imports | Standard Stack | Low -- can duplicate the array if import isn't possible |
| A2 | `store.SaveJSON` uses `AtomicWrite` with file locking, making cache writes safe for parallel colony-prime calls | Common Pitfalls | Medium -- need to verify AtomicWrite implementation; fallback: don't cache-write in parallel, just read |
| A3 | The `clarified_intent` section at priority 8 will not conflict with `prior_reviews` at priority 8 | Common Pitfalls | Low -- ranking uses score as primary sort, priority as tiebreaker |
| A4 | Cache refresh on every colony-prime call (D-05) is acceptable performance | Architecture Patterns | Low -- colony-prime runs once per build/continue, not in a hot loop |
| A5 | No new test helper functions needed -- `setupReviewLedgerTest` from `review_ledger_test.go` is reusable | Validation Architecture | Medium -- the test helper sets up `store` global; colony-prime tests need a full store with COLONY_STATE.json too |

## Open Questions (RESOLVED)

1. **Should `domainOrder` be moved to `pkg/colony/review_ledger.go`?** RESOLVED: Yes. Plan Task 1 moves `DomainOrder` to `pkg/colony/review_ledger.go` and updates `cmd/review_ledger.go` to use `colony.DomainOrder`.

2. **Should the cache store per-domain open counts for the LogLine?** RESOLVED: Yes. Plan Task 2 adds `ReviewCount int` to `colonyPrimeOutput` and updates the LogLine format.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified -- this phase is purely Go code changes within the existing repo).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify (for assertions) |
| Config file | none (Go test conventions) |
| Quick run command | `go test ./cmd/ -run TestPriorReviews -v -count=1` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PRIME-01 | Section assembled at priority 8 | unit | `go test ./cmd/ -run TestPriorReviewsSection_Priority -v` | No -- Wave 0 |
| PRIME-02 | Open findings shown with severity and file/location | unit | `go test ./cmd/ -run TestPriorReviewsSection_Formatting -v` | No -- Wave 0 |
| PRIME-03 | 800/400 char budget cap respected | unit | `go test ./cmd/ -run TestPriorReviewsSection_BudgetCap -v` | No -- Wave 0 |
| PRIME-04 | Section omitted when no ledgers exist | unit | `go test ./cmd/ -run TestPriorReviewsSection_OmittedWhenEmpty -v` | No -- Wave 0 |
| PRIME-05 | Cache hit avoids 7 file reads | unit | `go test ./cmd/ -run TestPriorReviewsCache_Hit -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestPriorReviews -v -count=1`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/colony_prime_prior_reviews_test.go` -- new test file for prior-reviews section tests
- [ ] Framework install: none needed (stdlib)

*(No shared fixtures gaps -- existing `setupReviewLedgerTest` helper in `review_ledger_test.go` can be adapted)*

## Security Domain

> Not applicable for this phase. The prior-reviews section reads local JSON files and formats strings. No user input, no network calls, no authentication, no secrets handling. The cache file is local-only under `.aether/data/` which is gitignored.

## Sources

### Primary (HIGH confidence)
- `cmd/colony_prime_context.go` -- Section assembly pattern, buildColonyPrimeOutput(), colonyPrimeSection struct, all existing sections with their priority/score values
- `pkg/colony/review_ledger.go` -- ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary, ComputeSummary, ReviewSeverity constants
- `cmd/review_ledger.go` -- domainOrder array, validDomains map, review-ledger-summary command pattern
- `pkg/colony/context_ranking.go` -- RankContextCandidates, ContextCandidate, scoring weights
- `cmd/context_weighting.go` -- sectionRelevanceScore(), protectedSectionPolicy()
- `pkg/cache/session_cache.go` -- SessionCache mtime-based staleness pattern (reference)
- `pkg/storage/storage.go` -- SaveJSON uses AtomicWrite with file locking

### Secondary (MEDIUM confidence)
- `cmd/review_ledger_test.go` -- Test patterns for ledger operations, setupReviewLedgerTest helper
- `.planning/REQUIREMENTS.md` -- PRIME-01 through PRIME-05 requirement text
- `.planning/phases/54-colony-prime-prior-reviews-section/54-CONTEXT.md` -- Locked decisions D-01 through D-12

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All packages are local, no new dependencies needed. Verified by reading source files.
- Architecture: HIGH - Section assembly pattern is copy-pasteable from 10+ existing sections. Verified by reading colony_prime_context.go.
- Pitfalls: HIGH - All pitfalls identified from reading the actual source code and understanding the ranking/sorting logic.

**Research date:** 2026-04-26
**Valid until:** 30 days (stable internal API, no external dependencies)
