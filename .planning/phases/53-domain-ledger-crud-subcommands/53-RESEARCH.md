# Phase 53: Domain-Ledger CRUD Subcommands - Research

**Researched:** 2026-04-26
**Domain:** Go runtime -- CLI subcommands for structured review finding persistence across 7 domain ledgers
**Confidence:** HIGH

## Summary

This phase creates four cobra CLI subcommands (`review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve`) that persist review agent findings across phases in 7 domain-specific JSON ledgers under `.aether/data/reviews/`. Each ledger holds an array of entries with deterministic IDs and a computed summary. The implementation follows the pheromone-write pattern: cobra command in `cmd/`, data types in `pkg/colony/`, atomic writes via `store.AtomicWrite()`/`store.SaveJSON()` from `pkg/storage/`.

The key architectural decision is that ledgers are independent JSON files per domain -- not one monolithic file and not a database. This matches the pheromone pattern (single `pheromones.json`), the midden pattern (single `midden.json`), and the instincts pattern (single `instincts.json`). Each domain gets its own file because agents write to different domains concurrently (e.g., Gatekeeper writes security while Watcher writes testing), and file-per-domain avoids lock contention.

The data types (ledger entry struct, ledger file struct, summary struct) belong in `pkg/colony/` alongside `MiddenEntry`, `PheromoneSignal`, and other data types. The cobra commands belong in a new `cmd/review_ledger.go` file. The domain prefix mapping (security->sec, quality->qlt, etc.) and agent-to-domain mapping are constants in the command file.

**Primary recommendation:** Create `pkg/colony/review_ledger.go` with data types, then `cmd/review_ledger.go` with four cobra commands following the pheromone-write/midden pattern. Use `store.SaveJSON()` for writes (it calls `AtomicWrite` internally with JSON validation). Compute summaries on every write. Use `store.LoadJSON()` for reads with the standard "file missing = empty struct" fallback pattern.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Data types (ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary) | pkg/colony (shared types) | -- | All other data types (MiddenEntry, PheromoneSignal) live in pkg/colony/ |
| CLI subcommands (write/read/summary/resolve) | cmd/ (cobra commands) | -- | All CLI commands live in cmd/ following cobra pattern |
| File I/O (atomic writes, file locking) | pkg/storage (existing) | -- | store.SaveJSON() and store.LoadJSON() already provide atomic writes with locking |
| Deterministic ID generation | cmd/ (command logic) | -- | ID format `{prefix}-{phase}-{index}` requires reading current ledger, belongs with command |
| Domain-to-prefix mapping | cmd/ (constants) | -- | Simple static map, no shared package needed |
| Agent-to-domain validation | cmd/ (command logic) | -- | LEDG-10 requires enforcement; belongs with write command |

## User Constraints (from STATE.md)

### Locked Decisions

- Review findings are colony-scoped (not cross-colony) -- code-specific paths go stale
- Domain ledger uses append pattern with computed summaries (no separate phase snapshots -- YAGNI)
- Continue-review worker reports mirror existing build worker report pattern
- All new struct fields use `omitempty` for backward compatibility with old JSON
- Zero new dependencies -- everything uses existing pkg/storage/, cobra, Go stdlib

### Claude's Discretion

(No Claude's Discretion section -- all decisions are locked)

### Deferred Ideas (OUT OF SCOPE)

- Cross-colony ledger sharing
- Auto-block on critical findings
- Auto finding-to-pheromone promotion
- Real-time ledger sync across agents
- Separate phase-level ledger snapshots
- Ledger web UI

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LEDG-01 | `review-ledger-write --domain --phase --findings <json>` creates ledger, assigns IDs, appends entries, recomputes summary | `store.SaveJSON()` for atomic writes; pheromone-write at `cmd/pheromone_write.go:143-176` shows LoadJSON+modify+SaveJSON pattern; `pkg/storage/storage.go:140-147` SaveJSON handles JSON validation automatically |
| LEDG-02 | `review-ledger-read --domain [--phase] [--status]` reads with optional filters | `store.LoadJSON()` for reads; midden-recent-failures at `cmd/midden_cmds.go:17-48` shows filter+sort+output pattern |
| LEDG-03 | `review-ledger-summary` returns one-line summary per domain | Iterate 7 known domains, LoadJSON each, compute/return summary; no file caching needed at this phase (Phase 54 adds cached summary for colony-prime performance) |
| LEDG-04 | `review-ledger-resolve --domain --id` marks entry resolved with timestamp | Same LoadJSON+modify+SaveJSON pattern as midden-acknowledge at `cmd/midden_cmds.go:86-131` |
| LEDG-05 | Seven domain directories under `.aether/data/reviews/` | `pkg/storage/storage.go:82-85` atomicWriteLocked calls `os.MkdirAll(dir, 0755)` automatically -- intermediate directories created on first write |
| LEDG-06 | Ledger entries include: id, phase, phase_name, agent, agent_name, generated_at, status, severity, file, line, category, description, suggestion | Struct definition in `pkg/colony/review_ledger.go` with all fields; severity is string ("HIGH"/"MEDIUM"/"LOW"/"INFO"); status is "open"/"resolved" |
| LEDG-07 | Deterministic IDs use format `{domain-prefix}-{phase}-{index}` (e.g., `sec-2-001`) | Domain prefix map as const; index is zero-padded 3 digits; computed by counting existing entries for same phase in domain ledger |
| LEDG-08 | Computed summary with total, open/resolved counts, by-severity breakdown | `ReviewLedgerSummary` struct recomputed on every write; stored in ledger file alongside entries |
| LEDG-09 | All writes use file-locking atomic writes via `pkg/storage/` (follow pheromone pattern, not hive pattern) | `store.SaveJSON()` at `pkg/storage/storage.go:140-147` calls `AtomicWrite()` which acquires FileLocker lock; same pattern as pheromone-write |
| LEDG-10 | Agent-to-domain mapping enforced: Gatekeeper->security, Auditor->quality/security/performance, Chaos->resilience, Watcher->testing/quality, Archaeologist->history, Measurer->performance, Tracker->bugs | Validated in `review-ledger-write` command; agent flag checked against allowed domain list; map defined as Go map[string][]string constant |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (encoding/json, fmt, strings, sort, time, strconv) | (project go.mod) | JSON marshal/unmarshal, string ops, time formatting | Existing project dependency |
| pkg/storage | (in-repo) | `store.SaveJSON()`, `store.LoadJSON()` for atomic reads/writes | Already used by pheromone-write, midden-write, and all other data commands |
| pkg/colony | (in-repo) | Data types for ledger entries, files, summaries | All colony data types live here (MiddenEntry, PheromoneSignal, etc.) |
| cobra | (project go.mod) | CLI command framework | All 80+ subcommands use cobra |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/colony/colony.Phase | (in-repo) | Phase struct for looking up phase_name from phase number | When writing entries, resolving phase name from COLONY_STATE.json plan |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| File-per-domain (7 ledger.json files) | Single ledger.json with domain field | File-per-domain avoids lock contention when agents write concurrently; matches pheromone/midden pattern of one file per concern |
| `store.SaveJSON()` for writes | `store.UpdateJSONAtomically()` | `UpdateJSONAtomically` does read-modify-write in one lock scope, which is cleaner for append operations -- USE THIS instead |
| `store.AtomicWrite()` directly | `store.SaveJSON()` | SaveJSON adds JSON validation + formatting; preferable for JSON files |
| Separate summary file per domain | Computed summary stored in ledger file | Storing summary alongside entries avoids two-file consistency issues; recomputed on every write |

**Installation:** None needed -- all dependencies already in project.

## Architecture Patterns

### System Architecture Diagram

```
LLM Agent (Gatekeeper, Auditor, Chaos, etc.)
  |
  | 1. Agent produces structured findings JSON
  |    [{severity, file, line, category, description, suggestion}, ...]
  |
  v
aether review-ledger-write --domain security --phase 2 --agent gatekeeper --findings '<json>'
  |
  | 2. Validate domain against agent-to-domain mapping (LEDG-10)
  | 3. Validate --findings is valid JSON
  | 4. Load ledger: store.LoadJSON("reviews/security/ledger.json", &ledger)
  |    -> If missing: create empty ReviewLedgerFile
  | 5. For each finding:
  |    a. Compute deterministic ID: sec-2-001, sec-2-002, ...
  |       (count existing entries for phase 2 in security ledger, increment)
  |    b. Build ReviewLedgerEntry with all fields (LEDG-06)
  |    c. Append to ledger.Entries
  | 6. Recompute summary: total, open, resolved, by-severity counts
  | 7. Atomic write: store.SaveJSON("reviews/security/ledger.json", ledger)
  |
  v
.aether/data/reviews/security/ledger.json
  {
    "entries": [...],
    "summary": { "total": 5, "open": 4, "resolved": 1, "by_severity": {...} }
  }

--- Read Path ---

aether review-ledger-read --domain security --status open
  |
  | 1. Load ledger from store
  | 2. Filter entries by --phase and --status flags
  | 3. Return filtered entries + summary
  |
  v
{"ok":true,"result":{"entries":[...],"summary":{...}}}

--- Summary Path ---

aether review-ledger-summary
  |
  | 1. For each of 7 domains:
  |    a. Try LoadJSON("reviews/{domain}/ledger.json")
  |    b. If exists: extract summary
  |    c. If missing: skip
  | 2. Return array of domain summaries
  |
  v
{"ok":true,"result":{"domains":[{"domain":"security","total":5,"open":4,...},...]}}

--- Resolve Path ---

aether review-ledger-resolve --domain security --id sec-2-001
  |
  | 1. Load ledger from store
  | 2. Find entry by ID
  | 3. Set status="resolved", resolved_at=<now>
  | 4. Recompute summary
  | 5. Atomic write back
  |
  v
{"ok":true,"result":{"resolved":true,"id":"sec-2-001"}}
```

### Recommended Project Structure

```
pkg/colony/
  review_ledger.go          # NEW: ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary types
  review_ledger_test.go     # NEW: Round-trip tests, summary computation tests

cmd/
  review_ledger.go          # NEW: Four cobra commands (write, read, summary, resolve)
  review_ledger_test.go     # NEW: Integration tests for all four commands
```

### Pattern 1: Load-Modify-Save with Atomic Writes (pheromone/midden pattern)

**What:** Load JSON file into struct, modify in memory, write back atomically.

**When to use:** Every write operation (LEDG-01 write, LEDG-04 resolve).

**Example:**
```go
// Source: cmd/pheromone_write.go:142-176 [VERIFIED: codebase grep]
var pf colony.PheromoneFile
if err := store.LoadJSON("pheromones.json", &pf); err != nil {
    pf = colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
}
if pf.Signals == nil {
    pf.Signals = []colony.PheromoneSignal{}
}
// ... modify pf.Signals ...
if err := store.SaveJSON("pheromones.json", pf); err != nil {
    outputError(2, fmt.Sprintf("failed to save pheromones: %v", err), nil)
    return nil
}
```

### Pattern 2: Cobra Command with Flag Registration (standard pattern)

**What:** Define cobra command with RunE, register flags in init(), add to rootCmd.

**When to use:** All four new subcommands.

**Example:**
```go
// Source: cmd/pheromone_write.go:19-207 and cmd/pheromone_write.go:271-286 [VERIFIED: codebase grep]
var reviewLedgerWriteCmd = &cobra.Command{
    Use:   "review-ledger-write",
    Short: "Write review findings to a domain ledger",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil {
            outputErrorMessage("no store initialized")
            return nil
        }
        // ... implementation ...
    },
}

func init() {
    reviewLedgerWriteCmd.Flags().String("domain", "", "Review domain (required)")
    reviewLedgerWriteCmd.Flags().Int("phase", 0, "Phase number (required)")
    reviewLedgerWriteCmd.Flags().String("findings", "", "JSON array of findings (required)")
    reviewLedgerWriteCmd.Flags().String("agent", "", "Agent caste that produced findings")
    reviewLedgerWriteCmd.Flags().String("agent-name", "", "Deterministic agent name")
    rootCmd.AddCommand(reviewLedgerWriteCmd)
}
```

### Pattern 3: Test Helper Setup (established pattern)

**What:** Create temp dir, init store, set env vars, capture stdout.

**When to use:** All test functions.

**Example:**
```go
// Source: cmd/write_cmds_test.go:21-36 [VERIFIED: codebase grep]
func newTestStore(t *testing.T) (*storage.Store, string) {
    t.Helper()
    origColonyDataDir := os.Getenv("COLONY_DATA_DIR")
    t.Cleanup(func() { os.Setenv("COLONY_DATA_DIR", origColonyDataDir) })
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    os.MkdirAll(dataDir, 0755)
    os.Setenv("COLONY_DATA_DIR", dataDir)
    s, err := storage.NewStore(dataDir)
    if err != nil { t.Fatalf("create store: %v", err) }
    return s, tmpDir
}
```

### Anti-Patterns to Avoid

- **Using `store.UpdateJSONAtomically()` for the write command:** This method locks the file for a read-modify-write cycle, which is theoretically cleaner, but the pheromone-write pattern uses LoadJSON+modify+SaveJSON (two separate lock acquisitions). Since agents won't write to the same domain file concurrently (each domain has its own file), the simpler two-step pattern is fine. However, if strict atomicity is needed, `UpdateJSONAtomically` is available and avoids the theoretical race window.
- **Storing summaries in a separate file:** Keeping summary in the same file as entries ensures consistency. Storing separately requires two atomic writes and risks drift.
- **Creating all 7 domain directories at init time:** Let them be created lazily on first write (MkdirAll in `atomicWriteLocked` handles this). Empty directories with no ledger.json file would confuse the read/summary commands.
- **Using a global ledger file with domain as a field:** This creates lock contention when multiple agents write findings concurrently during a build. One file per domain is the right call.
- **Enforcing agent-to-domain mapping at the type level:** The mapping is a runtime check in the command, not a type constraint. Agents should not be trusted to write only their domain -- the command validates.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic file writes | Custom temp-file + rename logic | `store.SaveJSON()` | Already provides locking, atomic rename, MkdirAll, JSON validation |
| JSON marshaling with formatting | Manual json.Marshal + indentation | `store.SaveJSON()` | Handles MarshalIndent + trailing newline + atomic write in one call |
| File locking for concurrent access | Custom lock files | `store.AtomicWrite()` (via SaveJSON) | `FileLocker` in `pkg/storage/lock.go` already handles cross-process locking |
| Output formatting | Custom print logic | `outputOK()` / `outputError()` | Standard JSON envelope format used by all commands |
| Flag validation | Custom error handling | `mustGetString()`, `mustGetInt()` | Already handle empty/missing flag errors with standard error format |

**Key insight:** The pheromone-write command at `cmd/pheromone_write.go` is a near-complete template for review-ledger-write. The main differences are: (1) multiple domain files instead of one file, (2) deterministic IDs instead of random IDs, (3) summary computation, and (4) agent-to-domain validation. The midden-acknowledge command at `cmd/midden_cmds.go:86-131` is the template for review-ledger-resolve.

## Common Pitfalls

### Pitfall 1: Deterministic ID collision across writes
**What goes wrong:** If agent writes findings in two separate calls for the same phase+domain, the second call needs to know how many entries already exist to avoid ID collisions.
**Why it happens:** The index for `sec-2-001` is computed by counting existing entries for phase 2 in the security ledger. If the count is wrong, IDs collide.
**How to avoid:** Count entries with matching phase prefix in the existing ledger entries. The format `{prefix}-{phase}-{NNN}` means we scan entries whose ID starts with `{prefix}-{phase}-` and take the max index + 1.
**Warning signs:** Duplicate IDs in ledger entries, silent data overwrite.

### Pitfall 2: --findings JSON parsing errors
**What goes wrong:** The `--findings` flag receives a JSON string from the command line. Shell quoting, escaping, and multiline JSON can cause parse errors.
**Why it happens:** Agents pass JSON as a CLI argument. Complex JSON with quotes and special characters needs careful escaping.
**How to avoid:** Parse `--findings` with `json.Unmarshal` immediately. Return a clear error with the parse failure message if invalid. Consider accepting `@file` syntax to read from a file (but this can be deferred -- not in current requirements).
**Warning signs:** "invalid character" errors from JSON parsing, empty findings arrays.

### Pitfall 3: Summary drift from entries
**What goes wrong:** The summary stored in the ledger file doesn't match the actual entries (e.g., total says 5 but there are 6 entries).
**Why it happens:** If a code path modifies entries without recomputing the summary, or if the file is manually edited.
**How to avoid:** Always recompute summary from entries on every write/resolve operation. Never allow summary to be set independently. Add a `ComputeSummary(entries []ReviewLedgerEntry) ReviewLedgerSummary` function and call it in both write and resolve paths.
**Warning signs:** Summary counts don't match actual entry counts.

### Pitfall 4: Phase number 0 ambiguity
**What goes wrong:** `--phase 0` could mean "not provided" (Go zero value for int) or an actual phase number.
**Why it happens:** Cobra's `GetInt` returns 0 for both missing and zero-value flags.
**How to avoid:** Make `--phase` required for `review-ledger-write` (use `MarkFlagRequired`). For `review-ledger-read`, use a separate boolean flag or check if the int flag was actually set via `cmd.Flags().Changed("phase")`.
**Warning signs:** Entries with phase=0 in ledgers.

### Pitfall 5: Agent-to-domain mapping bypass
**What goes wrong:** An agent or user calls `review-ledger-write --domain security --agent builder` -- a builder agent is not mapped to the security domain per LEDG-10.
**Why it happens:** The `--agent` flag is a string, and validation might be missing or incomplete.
**How to avoid:** Maintain a `map[string][]string` of agent->allowed domains. When `--agent` is provided, validate that the agent is allowed to write to the specified domain. If `--agent` is not provided (e.g., CLI manual use), skip validation.
**Warning signs:** Entries in security ledger from builder agents.

## Code Examples

### Data Types (pkg/colony/review_ledger.go)

```go
// Source: Modeled after pkg/colony/midden.go [VERIFIED: codebase pattern]

package colony

// ReviewSeverity represents the severity level of a review finding.
type ReviewSeverity string

const (
    ReviewSeverityHigh   ReviewSeverity = "HIGH"
    ReviewSeverityMedium ReviewSeverity = "MEDIUM"
    ReviewSeverityLow    ReviewSeverity = "LOW"
    ReviewSeverityInfo   ReviewSeverity = "INFO"
)

// ReviewLedgerEntry represents a single finding in a domain review ledger.
type ReviewLedgerEntry struct {
    ID           string        `json:"id"`
    Phase        int           `json:"phase"`
    PhaseName    string        `json:"phase_name,omitempty"`
    Agent        string        `json:"agent"`
    AgentName    string        `json:"agent_name,omitempty"`
    GeneratedAt  string        `json:"generated_at"`
    Status       string        `json:"status"`              // "open" or "resolved"
    Severity     ReviewSeverity `json:"severity"`
    File         string        `json:"file,omitempty"`
    Line         int           `json:"line,omitempty"`
    Category     string        `json:"category,omitempty"`
    Description  string        `json:"description"`
    Suggestion   string        `json:"suggestion,omitempty"`
    ResolvedAt   *string       `json:"resolved_at,omitempty"`
}

// ReviewLedgerSeverityCounts holds counts by severity level.
type ReviewLedgerSeverityCounts struct {
    High   int `json:"high"`
    Medium int `json:"medium"`
    Low    int `json:"low"`
    Info   int `json:"info"`
}

// ReviewLedgerSummary holds computed summary statistics for a domain ledger.
type ReviewLedgerSummary struct {
    Total      int                       `json:"total"`
    Open       int                       `json:"open"`
    Resolved   int                       `json:"resolved"`
    BySeverity ReviewLedgerSeverityCounts `json:"by_severity"`
}

// ReviewLedgerFile represents the top-level structure of a domain ledger.
type ReviewLedgerFile struct {
    Entries []ReviewLedgerEntry `json:"entries"`
    Summary ReviewLedgerSummary `json:"summary"`
}
```

### Domain Prefix Mapping (cmd/review_ledger.go)

```go
// Source: New constants following established pattern [ASSUMED]

var domainPrefixes = map[string]string{
    "security":    "sec",
    "quality":     "qlt",
    "performance": "prf",
    "resilience":  "res",
    "testing":     "tst",
    "history":     "hst",
    "bugs":        "bug",
}

var validDomains = map[string]bool{
    "security": true, "quality": true, "performance": true,
    "resilience": true, "testing": true, "history": true, "bugs": true,
}

// agentAllowedDomains maps agent caste to the domains it may write to (LEDG-10).
var agentAllowedDomains = map[string][]string{
    "gatekeeper":   {"security"},
    "auditor":      {"quality", "security", "performance"},
    "chaos":        {"resilience"},
    "watcher":      {"testing", "quality"},
    "archaeologist": {"history"},
    "measurer":     {"performance"},
    "tracker":      {"bugs"},
}
```

### Deterministic ID Computation

```go
// Source: New function following pheromone-write ID pattern [ASSUMED]

// nextEntryIndex returns the next 1-based index for entries in the given
// domain+phase combination. It scans existing entries and finds the max index.
func nextEntryIndex(entries []colony.ReviewLedgerEntry, prefix string, phase int) int {
    prefixStr := fmt.Sprintf("%s-%d-", prefix, phase)
    maxIdx := 0
    for _, e := range entries {
        if strings.HasPrefix(e.ID, prefixStr) {
            // Parse the trailing number: "sec-2-001" -> 1
            idxStr := strings.TrimPrefix(e.ID, prefixStr)
            if idx, err := strconv.Atoi(idxStr); err == nil && idx > maxIdx {
                maxIdx = idx
            }
        }
    }
    return maxIdx + 1
}

// formatEntryID produces a deterministic ID like "sec-2-001".
func formatEntryID(prefix string, phase, index int) string {
    return fmt.Sprintf("%s-%d-%03d", prefix, phase, index)
}
```

### Summary Computation

```go
// Source: New function [ASSUMED]

func computeSummary(entries []colony.ReviewLedgerEntry) colony.ReviewLedgerSummary {
    var s colony.ReviewLedgerSummary
    s.Total = len(entries)
    for _, e := range entries {
        switch e.Status {
        case "open":
            s.Open++
        case "resolved":
            s.Resolved++
        }
        switch e.Severity {
        case colony.ReviewSeverityHigh:
            s.BySeverity.High++
        case colony.ReviewSeverityMedium:
            s.BySeverity.Medium++
        case colony.ReviewSeverityLow:
            s.BySeverity.Low++
        case colony.ReviewSeverityInfo:
            s.BySeverity.Info++
        }
    }
    return s
}
```

### review-ledger-write Command Skeleton

```go
// Source: Modeled after cmd/pheromone_write.go:19-207 [VERIFIED: codebase pattern]

var reviewLedgerWriteCmd = &cobra.Command{
    Use:   "review-ledger-write",
    Short: "Write review findings to a domain ledger",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil {
            outputErrorMessage("no store initialized")
            return nil
        }

        domain := mustGetString(cmd, "domain")
        phase := mustGetInt(cmd, "phase")
        findingsJSON := mustGetString(cmd, "findings")
        agent, _ := cmd.Flags().GetString("agent")
        agentName, _ := cmd.Flags().GetString("agent-name")
        phaseName, _ := cmd.Flags().GetString("phase-name")

        if domain == "" || findingsJSON == "" {
            return nil // mustGetString already reported error
        }

        if !validDomains[domain] {
            outputError(1, fmt.Sprintf("invalid domain %q: must be one of security, quality, performance, resilience, testing, history, bugs", domain), nil)
            return nil
        }

        if phase == 0 {
            outputError(1, "flag --phase is required and must be > 0", nil)
            return nil
        }

        // Validate agent-to-domain mapping (LEDG-10)
        if agent != "" {
            allowed, ok := agentAllowedDomains[agent]
            if !ok {
                outputError(1, fmt.Sprintf("unknown agent %q", agent), nil)
                return nil
            }
            domainOK := false
            for _, d := range allowed {
                if d == domain {
                    domainOK = true
                    break
                }
            }
            if !domainOK {
                outputError(1, fmt.Sprintf("agent %q is not allowed to write to domain %q (allowed: %v)", agent, domain, allowed), nil)
                return nil
            }
        }

        // Parse findings JSON
        var findings []struct {
            Severity   string `json:"severity"`
            File       string `json:"file"`
            Line       int    `json:"line"`
            Category   string `json:"category"`
            Description string `json:"description"`
            Suggestion string `json:"suggestion"`
        }
        if err := json.Unmarshal([]byte(findingsJSON), &findings); err != nil {
            outputError(1, fmt.Sprintf("invalid --findings JSON: %v", err), nil)
            return nil
        }

        prefix := domainPrefixes[domain]
        ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)
        now := time.Now().UTC().Format(time.RFC3339)

        // Load existing ledger
        var ledger colony.ReviewLedgerFile
        if err := store.LoadJSON(ledgerPath, &ledger); err != nil {
            ledger = colony.ReviewLedgerFile{
                Entries: []colony.ReviewLedgerEntry{},
            }
        }
        if ledger.Entries == nil {
            ledger.Entries = []colony.ReviewLedgerEntry{}
        }

        // Append entries with deterministic IDs
        for _, f := range findings {
            idx := nextEntryIndex(ledger.Entries, prefix, phase)
            entry := colony.ReviewLedgerEntry{
                ID:          formatEntryID(prefix, phase, idx),
                Phase:       phase,
                PhaseName:   phaseName,
                Agent:       agent,
                AgentName:   agentName,
                GeneratedAt: now,
                Status:      "open",
                Severity:    colony.ReviewSeverity(strings.ToUpper(f.Severity)),
                File:        f.File,
                Line:        f.Line,
                Category:    f.Category,
                Description: f.Description,
                Suggestion:  f.Suggestion,
            }
            ledger.Entries = append(ledger.Entries, entry)
        }

        // Recompute summary
        ledger.Summary = computeSummary(ledger.Entries)

        // Atomic write
        if err := store.SaveJSON(ledgerPath, ledger); err != nil {
            outputError(2, fmt.Sprintf("failed to write ledger: %v", err), nil)
            return nil
        }

        outputOK(map[string]interface{}{
            "written": len(findings),
            "domain":  domain,
            "total":   len(ledger.Entries),
            "summary": ledger.Summary,
        })
        return nil
    },
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Review findings lost after `/clear` | Findings persisted in domain ledgers | Phase 53 (this phase) | Review knowledge survives session boundaries |
| No structured query of review findings | `review-ledger-read` with phase/status filters | Phase 53 | Downstream workers can query prior findings |
| No way to mark findings resolved | `review-ledger-resolve` with timestamp | Phase 53 | Findings lifecycle becomes trackable |
| No cross-domain review summary | `review-ledger-summary` for all 7 domains | Phase 53 | Colony-prime can inject review state into worker context |

**Deprecated/outdated:**
- Nothing deprecated in this phase -- all additions are new.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `store.SaveJSON()` creates intermediate directories via `MkdirAll` in `atomicWriteLocked` | Architecture Patterns | LOW -- verified at `pkg/storage/storage.go:82-85` |
| A2 | `store.LoadJSON()` returns an error for missing files (not an empty struct) | Architecture Patterns | LOW -- verified at `pkg/storage/storage.go:150-165`, it calls `os.ReadFile` which returns error for missing files |
| A3 | Phase name can be looked up from `COLONY_STATE.json` plan phases | Code Examples | LOW -- `colony.ColonyState.Plan.Phases` contains `colony.Phase` with `Name` field, verified at `pkg/colony/colony.go:279-286` |
| A4 | Domain prefix map values are: sec, qlt, prf, res, tst, hst, bug | Code Examples | LOW -- arbitrary but reasonable; the planner should confirm these prefixes |
| A5 | `--phase` will always be a positive integer (> 0) for real colony phases | Common Pitfalls | LOW -- phase numbering starts at 1 in all existing colony data |
| A6 | The `--findings` JSON schema is a flat array of finding objects (not nested) | Code Examples | MEDIUM -- the exact schema agents will produce is defined in Phase 55, but LEDG-06 specifies the fields |

## Open Questions

1. **Should `--phase-name` be required or auto-resolved from COLONY_STATE.json?**
   - What we know: LEDG-06 requires `phase_name` in each entry. The colony state has plan phases with names.
   - What's unclear: Whether the write command should auto-resolve the phase name by reading COLONY_STATE.json, or require it as an explicit flag.
   - Recommendation: Accept `--phase-name` as an optional flag. If not provided, try to resolve from COLONY_STATE.json plan. If neither source provides a name, leave it empty (omitempty). This gives agents the flexibility to pass it explicitly while also supporting auto-resolution.

2. **Should `review-ledger-summary` show domains with no ledger file?**
   - What we know: LEDG-03 says "one-line summary per domain showing total, open, and by-severity counts."
   - What's unclear: Whether domains with zero entries (no ledger file yet) should appear in the output.
   - Recommendation: Skip domains with no ledger file. The output shows only domains that have at least one entry. This avoids noise and matches the "no empty placeholder" principle from Phase 54.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies -- all changes are in-project Go code)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- standard `go test` |
| Quick run command | `go test ./cmd/ -run "TestReviewLedger" -count=1 -timeout 60s` |
| Full suite command | `go test ./cmd/ -count=1 -race -timeout 300s` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LEDG-01 | Write creates ledger with deterministic IDs and computed summary | integration | `go test ./cmd/ -run "TestReviewLedgerWrite" -count=1` | Wave 0 |
| LEDG-02 | Read with phase/status filters returns correct entries | integration | `go test ./cmd/ -run "TestReviewLedgerRead" -count=1` | Wave 0 |
| LEDG-03 | Summary returns one-line per domain with total/open/severity | integration | `go test ./cmd/ -run "TestReviewLedgerSummary" -count=1` | Wave 0 |
| LEDG-04 | Resolve marks entry as resolved with timestamp | integration | `go test ./cmd/ -run "TestReviewLedgerResolve" -count=1` | Wave 0 |
| LEDG-05 | Seven domain directories exist under reviews/ | integration | `go test ./cmd/ -run "TestReviewLedger.*DomainDirs" -count=1` | Wave 0 |
| LEDG-06 | Entry struct has all required fields | unit | `go test ./pkg/colony/ -run "TestReviewLedgerEntry" -count=1` | Wave 0 |
| LEDG-07 | Deterministic IDs match {prefix}-{phase}-{index} format | unit | `go test ./cmd/ -run "TestReviewLedger.*DeterministicID" -count=1` | Wave 0 |
| LEDG-08 | Summary recomputed on write/resolve | unit | `go test ./cmd/ -run "TestReviewLedger.*Summary" -count=1` | Wave 0 |
| LEDG-09 | Writes use atomic file locking from pkg/storage | integration | `go test ./cmd/ -run "TestReviewLedger.*Atomic" -count=1` | Wave 0 |
| LEDG-10 | Agent-to-domain mapping enforced on write | integration | `go test ./cmd/ -run "TestReviewLedger.*AgentDomain" -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestReviewLedger" -count=1 -timeout 60s && go test ./pkg/colony/ -run "TestReviewLedger" -count=1 -timeout 60s`
- **Per wave merge:** `go test ./... -count=1 -race -timeout 300s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/colony/review_ledger.go` -- data types (ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary)
- [ ] `pkg/colony/review_ledger_test.go` -- JSON round-trip, summary computation, ID format tests
- [ ] `cmd/review_ledger.go` -- four cobra commands
- [ ] `cmd/review_ledger_test.go` -- integration tests for all four commands

## Security Domain

> New data files under `.aether/data/reviews/` with agent-written content. Agent input validation required.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | yes | Agent-to-domain mapping enforcement (LEDG-10) restricts which agents can write to which domains |
| V5 Input Validation | yes | `--findings` JSON must be validated before parsing; domain/agent strings validated against allowlists |
| V6 Cryptography | no | -- |

### Known Threat Patterns for CLI + JSON data files

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious JSON in --findings | Tampering | json.Unmarshal with strict struct typing; reject unexpected fields; limit string field lengths |
| Agent writing to wrong domain | Tampering | agentAllowedDomains map validation (LEDG-10) |
| Path traversal in --domain flag | Tampering | Validate domain against `validDomains` map (only 7 known values); reject any domain not in the map |
| Large findings array causing memory issues | Denial of Service | Cap findings array size (e.g., max 50 entries per call); this is a reasonable limit for a single agent report |

## Sources

### Primary (HIGH confidence)
- `pkg/storage/storage.go` -- AtomicWrite, SaveJSON, LoadJSON, UpdateJSONAtomically, MkdirAll behavior
- `cmd/pheromone_write.go` -- Complete CRUD command pattern (write, expire, validate)
- `cmd/midden_cmds.go` -- CRUD commands with filtering, grouping, and acknowledge pattern
- `pkg/colony/midden.go` -- Data type pattern (MiddenEntry, MiddenFile with struct + omitempty)
- `cmd/helpers.go` -- outputOK, outputError, mustGetString helpers
- `cmd/write_cmds_test.go` -- newTestStore helper, parseEnvelope helper
- `pkg/colony/colony.go:279-286` -- Phase struct with ID, Name fields
- `pkg/colony/colony.go:150-178` -- ColonyState struct with Plan.Phases
- `cmd/root.go:147-195` -- Store initialization, PersistentPreRunE pattern
- `cmd/pheromone_write_test.go` -- Test setup pattern with saveGlobals, resetRootCmd, newTestStore

### Secondary (MEDIUM confidence)
- `cmd/midden_cmds.go:485-510` -- init() flag registration and rootCmd.AddCommand pattern
- `.planning/REQUIREMENTS.md` -- LEDG-01 through LEDG-10 requirement definitions
- `.planning/phases/52-continue-review-worker-outcome-reports/52-RESEARCH.md` -- Prior phase context

### Tertiary (LOW confidence)
- None -- all findings verified against codebase or requirements.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all dependencies already in project, no new packages needed
- Architecture: HIGH - all patterns verified against existing codebase (pheromone-write, midden, storage)
- Pitfalls: HIGH - identified from reading actual code paths and understanding concurrent write patterns
- Data types: HIGH - modeled after verified MiddenEntry/MiddenFile pattern

**Research date:** 2026-04-26
**Valid until:** 90 days (stable Go codebase, no external dependency changes)
