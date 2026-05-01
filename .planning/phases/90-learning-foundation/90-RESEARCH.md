# Phase 90: Learning Foundation - Research

**Researched:** 2026-05-01
**Domain:** Go colony learning system -- evidence-gated memory triggers, unified memory store API, repo isolation, privacy classification
**Confidence:** HIGH

## Summary

Phase 90 introduces a unified learning API (`pkg/learn/`) that wraps the existing `pkg/memory` wisdom pipeline and provides a `LearnStore` interface with two implementations: `ColonyStore` (repo-local JSON in `.aether/data/learn/`) and `HiveStore` (wraps existing `cmd/hive.go`). Learning triggers only fire after `/ant-build` + `/ant-continue` verifies all gates pass AND all workers succeeded (strictest interpretation of LRN-01). Every durable learning entry carries full structured evidence: run ID, phase, workers, files, gates, confidence, timestamp, scope.

The existing codebase already has most building blocks: `pkg/memory` (trust scoring, observation capture, instinct promotion, consolidation), `cmd/provenance.go` (SAFE-03/04 validation), `cmd/security_cmds.go` (`privacyScan()` function), `cmd/hive.go` (wisdom store/abstract/promote), and `pkg/colony/context_ranking.go` (budget-aware ranking). Phase 90 wraps these into a unified API, adds evidence-gated triggers, automatic classification, and export/import. The key integration work is: (1) creating the `pkg/learn/` package with the `LearnStore` interface, (2) inserting learning capture after continue-finalize gate passes, (3) adding a "learned memory" section to colony-prime context assembly, (4) migrating 7 existing `pkg/memory` call sites in `cmd/` to use `pkg/learn/` instead.

No new external dependencies are needed. All Go standard library. Phase 91 (SQLite + FTS5 + pheromone skills) will swap `ColonyStore` implementation without touching `HiveStore` or the `LearnStore` interface.

**Primary recommendation:** Create `pkg/learn/` as a thin orchestration layer that delegates to `pkg/memory` internally, exposes a clean CRUD interface, and gates all writes through evidence verification + privacy classification.

## User Constraints

### Locked Decisions
- **D-01:** Learning fires only after /ant-continue verifies that gates pass AND provenance is valid (SAFE-03/04). Build-complete alone is not sufficient.
- **D-02:** All workers must succeed for durable memory. If ANY worker failed, blocked, or errored, the entire phase run is treated as failed for learning purposes.
- **D-03:** Only /ant-build + /ant-continue produces durable learning. Oracle findings, chaos results, archaeology insights, dreams -- all transient, never promoted to durable memory.
- **D-04:** Existing continue provenance check (SAFE-03/04) plus gate pass status IS the verification for learning eligibility. No additional learning-specific verification layer needed.
- **D-05:** New pkg/learn/ package with a unified LearnStore interface. Existing pkg/memory becomes an internal implementation detail.
- **D-06:** Repo-local colony memory lives in .aether/data/learn/ subdirectory. Physically separate from colony state files.
- **D-07:** MemoryStore interface with two implementations: ColonyStore (JSON in .aether/data/learn/) and HiveStore (wraps existing cmd/hive.go). Promotion moves entries between stores explicitly.
- **D-08:** Existing pkg/memory call sites in cmd/ need updating to go through pkg/learn/ instead of calling pkg/memory directly.
- **D-09:** Full structured evidence for every learning entry: source run ID, phase number, all worker names + castes, all files modified, gate results summary, confidence score, timestamp, and scope.
- **D-10:** Automatic classification at creation time. No user involvement for initial classification.
- **D-11:** Classification rules extend existing privacy scanner (cmd/security_cmds.go): scanner blocks -> blocked; scanner redacts -> repo-local; scanner passes clean + generic patterns -> hive-shareable; ambiguous/partial -> needs-user-approval.
- **D-12:** Export/import uses JSON manifest + redacted entries + redaction report. `aether learn export` and `aether learn import` subcommands.
- **D-13:** Learned memory feeds into existing context_ranking.go as ContextCandidate entries. Shares the same token budget.
- **D-14:** Workers receive learned context via colony-prime context assembly only (frozen snapshot). Workers do not call pkg/learn/ directly.
- **D-15:** Snapshot is frozen at colony-prime assembly but refreshed between build waves.
- **D-16:** Learning default is enabled. Can be disabled by config (global default in .planning/config.json) and by per-command flag (--no-learn).

### Claude's Discretion
- Hive relationship design: chose MemoryStore interface with ColonyStore + HiveStore implementations over unified scope parameter or separate APIs (D-07)
- Classification rules: chose extend existing privacy scanner over path/pattern heuristics or two-pass approach (D-11)
- Context injection: chose feed into existing context_ranking.go over separate budget or full replacement (D-13)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Learning trigger (evidence gate) | API / Backend (Go runtime) | -- | Only the continue-finalize path has provenance + gate results to make eligibility decisions |
| Memory store CRUD | API / Backend (Go runtime) | -- | pkg/learn/ is a Go package; all storage operations happen in-process |
| Privacy classification | API / Backend (Go runtime) | -- | Extends existing privacyScan() in cmd/security_cmds.go |
| Context injection | API / Backend (Go runtime) | -- | colony-prime context assembly is Go code in cmd/colony_prime_context.go |
| Hive promotion | API / Backend (Go runtime) | CDN / Static (hub ~/.aether/) | HiveStore writes to ~/.aether/hive/wisdom.json (hub-level file) |
| Export/import | API / Backend (Go runtime) | -- | CLI subcommands generate JSON files for user transfer |
| Config disable flag | API / Backend (Go runtime) | -- | Reads .planning/config.json and --no-learn flag |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (encoding/json, crypto/sha256, regexp, os, path/filepath, time, math) | 1.22+ | JSON serialization, hashing, regex, file ops | Already used throughout; no new deps |
| pkg/memory (existing) | internal | Trust scoring, observation capture, instinct promotion, consolidation | Wrapped by pkg/learn/ -- not replaced |
| pkg/storage (existing) | internal | Atomic file write/read with file locking | ColonyStore uses this for .aether/data/learn/ persistence |
| pkg/events (existing) | internal | Event bus pub/sub with JSONL persistence | Learning lifecycle events |
| pkg/colony (existing) | internal | ContextCandidate, ContextRanking, Observation, InstinctEntry types | Learning entries become ContextCandidates |
| cmd/security_cmds.go privacyScan() | existing | Secret detection + path redaction | Classification layer extends this |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| cmd/hive.go promoteToHive() | existing | Hive wisdom store/abstract/promote | HiveStore implementation delegates to this |
| cmd/provenance.go | existing | Build/continue provenance validation | Learning eligibility check references SAFE-03/04 results |
| cobra | existing | CLI subcommand framework | `aether learn export`, `aether learn import` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| ColonyStore as JSON files in .aether/data/learn/ | Direct SQLite from Phase 90 | Phase 91 will swap to SQLite; JSON keeps Phase 90 simple and decoupled |
| Extending privacyScan() for classification | Separate classification service | privacyScan() already has the block/redact logic; extending is less code |
| LearnStore as a Go interface | Concrete struct with methods | Interface allows Phase 91 to swap ColonyStore to SQLite without touching HiveStore |

**Installation:**
No new packages needed. All existing codebase infrastructure.

## Architecture Patterns

### System Architecture Diagram

```
Build + Continue Path
        |
        v
[continue-finalize]
        |
        | validateBuildProvenance() -- SAFE-01/02
        | traceContinueProvenance() -- SAFE-03/04
        | all gates pass?
        |
        |--- NO ---> [log transient only, skip learning]
        |
        |--- YES --> [Learning Eligibility Check]
                           |
                           | all workers succeeded? (D-02)
                           | --no-learn flag absent? (D-16)
                           | learning.enabled != false? (D-16)
                           |
                           |--- NO ---> [log transient only]
                           |
                           |--- YES --> [pkg/learn/ ColonyStore.Add()]
                                              |
                                              | 1. Collect evidence (run ID, phase, workers, files, gates)
                                              | 2. Compute confidence (trust scoring via pkg/memory)
                                              | 3. Run privacyScan() (PRIV-01/02)
                                              |    |
                                              |    |--- BLOCKED --> [mark as blocked, log, skip]
                                              |    |
                                              |    |--- REDACTED --> [store with redacted content, classify repo-local]
                                              |    |
                                              |    |--- CLEAN --> [classify: repo-local or hive-shareable]
                                              |
                                              | 4. Write to .aether/data/learn/entries.json
                                              | 5. Publish event to event bus


Colony-Prime Context Assembly (buildColonyPrimeOutput)
        |
        | Read .aether/data/learn/entries.json
        | Convert to ContextCandidate entries
        | Feed into RankContextCandidates() with existing sections
        |
        v
[Worker Prompts] <-- frozen snapshot (D-14, D-15)


Hive Promotion Path (seal or manual)
        |
        | Read entries classified as hive-shareable
        | Run privacy scan again (safety net)
        | Abstract repo-specific content
        | Require user approval (LRN-03)
        |
        v
[HiveStore.Add()] --> cmd/hive.go promoteToHive() --> ~/.aether/hive/wisdom.json


Export/Import Path
        |
        | `aether learn export` --> JSON manifest + redacted entries + redaction report
        | `aether learn import` --> preview --> user approval --> apply
        |
        v
[User transfers learning pack between repos]
```

### Recommended Project Structure
```
pkg/learn/
    learn.go              # LearnStore interface, Entry/Evidence/Classification types
    colony_store.go       # ColonyStore implementation (JSON in .aether/data/learn/)
    hive_store.go         # HiveStore implementation (wraps cmd/hive.go promoteToHive)
    classify.go           # Classification logic (extends privacyScan)
    evidence.go           # Evidence collection from provenance + gate results
    trigger.go            # Learning eligibility check (evidence-gated)
    export.go             # Export/import manifest generation + apply

.aether/data/learn/
    entries.json          # Repo-local learning entries (ColonyStore)
    config.json           # Per-repo learning config (budget, classification overrides)
```

### Pattern 1: LearnStore Interface (D-07)
**What:** A Go interface that defines the memory store contract. ColonyStore and HiveStore implement it.
**When to use:** All learning operations go through this interface. Allows Phase 91 to swap ColonyStore to SQLite.
**Example:**
```go
// Source: designed from existing cmd/hive.go and pkg/storage/store.go patterns
package learn

type Classification string

const (
    ClassBlocked        Classification = "blocked"
    ClassRepoLocal      Classification = "repo-local"
    ClassHiveShareable  Classification = "hive-shareable"
    ClassNeedsApproval  Classification = "needs-user-approval"
)

type Evidence struct {
    RunID       string   `json:"run_id"`
    Phase       int      `json:"phase"`
    Workers     []WorkerEvidence `json:"workers"`
    FilesTouched []string `json:"files_touched,omitempty"`
    GatesPassed int      `json:"gates_passed"`
    GatesTotal  int      `json:"gates_total"`
    Confidence  float64  `json:"confidence"`
    Timestamp   string   `json:"timestamp"`
    Scope       string   `json:"scope"`
}

type WorkerEvidence struct {
    Name   string `json:"name"`
    Caste  string `json:"caste"`
    Status string `json:"status"`
}

type Entry struct {
    ID            string        `json:"id"`
    Content       string        `json:"content"`
    Evidence      Evidence      `json:"evidence"`
    Classification Classification `json:"classification"`
    CreatedAt     string        `json:"created_at"`
    Phase         int           `json:"phase"`
    Caste         string        `json:"caste,omitempty"`
    FilePath      string        `json:"file_path,omitempty"`
    Confidence    float64       `json:"confidence"`
    Redacted      bool          `json:"redacted,omitempty"`
}

type LearnStore interface {
    Add(entry Entry) error
    Get(id string) (*Entry, error)
    List(filter EntryFilter) ([]Entry, error)
    Replace(id string, entry Entry) error
    Remove(id string) error
    Compact(budget int) error
}
```

### Pattern 2: Evidence-Gated Learning Trigger (D-01, D-02, D-04)
**What:** Learning eligibility is determined by checking continue provenance results and gate pass status. No separate "learning verification" layer.
**When to use:** Called from continue-finalize after gates pass and provenance is valid.
**Example:**
```go
// Source: extends existing cmd/provenance.go pattern
func IsLearningEligible(
    allWorkersSucceeded bool,     // D-02: strictest interpretation
    provenanceValid bool,          // D-04: SAFE-03/04 already checked
    gatesPassed bool,              // D-01: all gates passed
    learningEnabled bool,          // D-16: config + flag check
) bool {
    return allWorkersSucceeded && provenanceValid && gatesPassed && learningEnabled
}
```

### Pattern 3: Classification via Extended Privacy Scan (D-10, D-11)
**What:** Extend existing `privacyScan()` function with a classification output. Scanner blocks -> blocked. Scanner redacts (found paths) -> repo-local. Scanner passes clean + generic patterns -> hive-shareable. Ambiguous -> needs-user-approval.
**When to use:** Every learning write goes through classification before storage.
**Example:**
```go
// Source: extends cmd/security_cmds.go privacyScan()
func ClassifyEntry(content string, scanResult PrivacyScanResult, isGeneric bool) Classification {
    if scanResult.Blocked {
        return ClassBlocked
    }
    if scanResult.Clean != content {
        // Content was redacted (paths removed)
        return ClassRepoLocal
    }
    if isGeneric {
        return ClassHiveShareable
    }
    // Content passed clean but may contain repo-specific patterns
    return ClassNeedsApproval
}
```

### Pattern 4: Context Injection via Existing Ranking System (D-13, D-14)
**What:** Learning entries become `ContextCandidate` structs fed into `RankContextCandidates()` alongside existing sections (instincts, pheromones, hive wisdom, etc.).
**When to use:** Colony-prime context assembly in `buildColonyPrimeOutput()`.
**Example:**
```go
// Source: maps to existing pkg/colony/context_ranking.go
candidate := colony.ContextCandidate{
    Name:           "learned_memory",
    Title:          "Learned Memory",
    Source:         learnEntriesPath,
    Content:        formattedLearnings,
    BudgetMetric:   "chars",
    PriorityHint:   5,  // D-13: phase maps to PriorityHint
    FreshnessScore: computeFreshness(entry),    // D-13: recency
    ConfirmationScore: entry.Confidence,         // D-13: confidence
    RelevanceScore: computeRelevance(entry),     // D-13: caste + file path
}
```

### Anti-Patterns to Avoid
- **Calling pkg/memory directly from cmd/ after migration:** D-08 requires all cmd/ call sites go through pkg/learn/. If cmd/ code still imports pkg/memory, the migration is incomplete.
- **Creating learning entries without evidence:** Every Entry must have a populated Evidence struct. Empty evidence means the entry should be transient only (D-09).
- **Learning from non-continue paths:** Oracle, chaos, archaeology, dreams must never produce durable learning entries (D-03).
- **Classification without privacy scan:** All classification must go through the extended privacy scanner first. Hand-rolled classification logic will miss secret patterns.
- **Injecting learned context directly into workers:** Workers receive context via colony-prime assembly only (D-14). Workers must never call pkg/learn/ directly.
- **Storing learning data in COLONY_STATE.json:** Learning data lives in .aether/data/learn/ (D-06), separate from colony state files.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Trust scoring | Custom confidence algorithm | pkg/memory/trust.go Calculate() | 40/35/25 weighted, 7 tiers, 60-day half-life -- already battle-tested |
| Observation dedup | Content hash comparison | pkg/memory/observe.go CaptureWithTrust() | SHA-256 content hash dedup with observation count tracking |
| Instinct promotion | Custom promotion logic | pkg/memory/promote.go Promote() | Dedup, capacity cap (50), graph edge writing, event publishing |
| Memory consolidation | Custom decay/archive | pkg/memory/consolidate.go Run() | Multi-step pipeline: decay, archive, promotion candidates, queen-eligible |
| Privacy/secret scanning | Custom regex patterns | cmd/security_cmds.go privacyScan() | 10+ compiled regex patterns for API keys, private keys, passwords, tokens, env files |
| Atomic file writes | Manual tmp+rename | pkg/storage.Store SaveJSON() | File locking, atomic rename, JSON validation |
| Context ranking | Custom scoring | pkg/colony/context_ranking.go RankContextCandidates() | Budget-aware trimming with trust/freshness/confirmation/relevance scoring |
| Event bus | Custom pub/sub | pkg/events.Bus | JSONL persistence, TTL, topic matching, replay |
| Hive wisdom storage | Custom JSON file management | cmd/hive.go hiveStoreCmd/promoteToHive | Dedup, LRU eviction (200 cap), access tracking, confidence boosting |

**Key insight:** Phase 90 is primarily an orchestration layer. The individual capabilities (trust scoring, observation, promotion, privacy, ranking, events) are all already implemented and tested. The value is in wiring them together with evidence gating and a clean API surface.

## Common Pitfalls

### Pitfall 1: Learning Trigger Fires Too Early
**What goes wrong:** Learning captures happen at build-complete instead of continue-finalize, meaning gates may not have passed yet.
**Why it happens:** Build-complete feels like the "done" moment, but gates are checked during continue.
**How to avoid:** Insert learning trigger ONLY in continue-finalize, after gates pass AND provenance validates. Build-complete must not trigger learning (D-01, D-03).
**Warning signs:** Learning entries appear for builds that later fail gates. Evidence shows empty gate results.

### Pitfall 2: Partial Worker Success Creates False Learning
**What goes wrong:** 4 of 5 workers succeed and learning fires, but the failed worker's task was critical.
**Why it happens:** Checking "at least one worker succeeded" is insufficient.
**How to avoid:** D-02 requires ALL workers to succeed. Check every worker result's Status field is "completed" with FilesModified > 0.
**Warning signs:** Learning entries reference failed tasks or missing files. Instincts promoted from partially-successful runs.

### Pitfall 3: Migration Breaks Existing Tests
**What goes wrong:** Changing cmd/ call sites from pkg/memory to pkg/learn/ breaks existing tests that mock pkg/memory directly.
**Why it happens:** Tests have hardcoded imports and mock setups for pkg/memory types.
**How to avoid:** Phase 90 should introduce pkg/learn/ as a thin wrapper initially. Existing cmd/ imports can be updated gradually. The wrapper should expose the same types (or compatible aliases) so test mocks still work during migration.
**Warning signs:** `go test ./cmd/...` fails after migration. Tests reference `memory.ObservationService` directly.

### Pitfall 4: Privacy Classification Ambiguity
**What goes wrong:** Entries with no secrets and no paths get classified as "needs-user-approval" too aggressively, creating approval fatigue.
**Why it happens:** The "ambiguous/partial" catch-all in D-11 is too broad.
**How to avoid:** Default to "repo-local" for clean entries that contain repo-specific patterns (file paths, project names). Only escalate to "needs-user-approval" when content is borderline -- contains partial patterns that might be sensitive in some repos.
**Warning signs:** Most entries need user approval. Users start blindly approving everything.

### Pitfall 5: Context Budget Exhaustion
**What goes wrong:** Learned memory section consumes too much of the 8000-char budget, pushing out more critical sections (pheromones, blockers).
**Why it happens:** Learning entries accumulate over many phases and are not compacted.
**How to avoid:** ColonyStore.Compact() must be called before context assembly. Prioritize recent (last 5 phases) and high-confidence entries. Low-confidence or old entries get trimmed first.
**Warning signs:** Colony-prime log shows large "Trimmed" list. Workers missing critical context they previously had.

### Pitfall 6: Export Contains Sensitive Data
**What goes wrong:** Learning packs exported for sharing contain redacted-but-still-identifiable information.
**Why it happens:** Redaction replaces paths with [REDACTED_PATH] but the surrounding context may still identify the repo or user.
**How to avoid:** Export must run a second privacy pass specifically for export context. The redaction report must list exactly what was redacted so the recipient can verify.
**Warning signs:** Export file contains [REDACTED_PATH] placeholders that are trivially reversible with project knowledge.

## Code Examples

Verified patterns from existing codebase:

### Existing Privacy Scan (Extended for Classification)
```go
// Source: cmd/security_cmds.go (verified in codebase)
func privacyScan(content string) PrivacyScanResult {
    var findings []string
    for _, sp := range secretPatterns {
        if sp.pattern.MatchString(content) {
            findings = append(findings, fmt.Sprintf("secret pattern matched: %s", sp.name))
        }
    }
    if len(findings) > 0 {
        return PrivacyScanResult{Blocked: true, Findings: findings}
    }
    clean := homePathPattern.ReplaceAllString(content, "[REDACTED_PATH]")
    return PrivacyScanResult{Blocked: false, Clean: clean}
}
```

### Existing Context Ranking Integration Point
```go
// Source: cmd/colony_prime_context.go (verified in codebase)
// This is where learned memory section will be added:
// After hive wisdom section (line ~582), before global QUEEN.md section (line ~584)

// Pattern: create colonyPrimeSection, add to sections slice
sections = append(sections, colonyPrimeSection{
    name:              "learned_memory",
    title:             "Learned Memory",
    source:            learnEntriesPath,
    content:           learnContent,
    priority:          5,
    freshnessScore:    latestLearningFreshness(now, entries),
    confirmationScore: learningConfidenceScore(entries),
    relevanceScore:    sectionRelevanceScore("learned_memory"),
})
```

### Existing Trust Scoring (Used for Confidence)
```go
// Source: pkg/memory/trust.go (verified in codebase)
func Calculate(input TrustInput) TrustResult {
    sourceScore := sourceWeights[input.SourceType]
    evidenceScore := evidenceWeights[input.Evidence]
    activityScore := math.Pow(0.5, float64(input.DaysSince)/60.0)
    rawScore := 0.4*sourceScore + 0.35*evidenceScore + 0.25*activityScore
    score := math.Max(0.2, rawScore)
    // ...
}
```

### Existing Hive Promotion (HiveStore Wraps This)
```go
// Source: cmd/hive.go (verified in codebase)
func promoteToHive(text, domain, sourceRepo string, confidence float64) error {
    // Abstract repo-specific text
    abstracted := text
    if sourceRepo != "" {
        abstracted = strings.ReplaceAll(abstracted, sourceRepo, "<repo>")
    }
    for _, prefix := range []string{"src/", "lib/", "pkg/", "cmd/", "internal/"} {
        abstracted = strings.ReplaceAll(abstracted, prefix, "")
    }
    // Store with dedup and LRU eviction
    // ...
}
```

### Existing Provenance Validation (Learning Eligibility Check)
```go
// Source: cmd/provenance.go (verified in codebase)
// These functions already validate build/continue provenance.
// Learning trigger checks their results (D-04):
func validateBuildProvenance(results []codexExternalBuildWorkerResult) error { ... }
func traceContinueProvenance(dispatches []codexBuildDispatch) error { ... }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Learning from any observation | Evidence-gated learning only from verified build+continue | Phase 90 (this phase) | Prevents false learning confidence from unverified work |
| Direct pkg/memory imports in cmd/ | pkg/learn/ unified API wrapping pkg/memory | Phase 90 (this phase) | Cleaner separation, enables Phase 91 SQLite swap |
| Learning data mixed in colony state | Separate .aether/data/learn/ directory | Phase 90 (this phase) | Clean for deletion and export |
| No classification | Automatic 4-way classification at creation time | Phase 90 (this phase) | Privacy-aware memory with clear sharing boundaries |
| No export/import | JSON manifest with redaction report | Phase 90 (this phase) | Portable learning packs between repos |

**Deprecated/outdated:**
- Direct `memory.NewObservationService()` calls in cmd/learning.go: will be replaced by pkg/learn/ calls
- Direct `memory.NewPromoteService()` calls in cmd/learning_cmds.go: will be replaced by pkg/learn/ calls
- Direct `memory.NewPipeline()` calls in cmd/graph_consolidation_cmds.go: will be replaced by pkg/learn/ calls

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Existing provenance check (SAFE-03/04) is sufficient for learning eligibility -- no additional verification needed (D-04) | Architecture Patterns | May need a learning-specific verification layer if provenance gaps are found in worktree mode |
| A2 | "Generic" pattern detection for hive-shareable classification can be done with simple heuristics (no repo paths, no file extensions) | Pattern 3 | May need more sophisticated NLP-based generic detection if too many entries end up as "needs-user-approval" |
| A3 | Existing 8000-char colony-prime budget can accommodate learned memory without starving other sections | Pattern 4 | May need budget increase or separate learned-memory budget if accumulated entries exceed ~1500 chars |
| A4 | PrivacyScanResult.Clean comparison (scanResult.Clean != content) reliably detects when path redaction occurred | Pattern 3 | If content has no paths and privacyScan returns it unchanged, comparison correctly identifies "no redaction needed" |
| A5 | The existing pkg/memory tests will continue to pass when pkg/learn/ wraps them, since pkg/learn/ calls pkg/memory internally | Pitfall 3 | If pkg/learn/ introduces new error paths or changes call signatures, existing tests may break |

## Open Questions (RESOLVED)

1. **What constitutes a "generic" pattern for hive-shareable classification?**
   - What we know: D-11 says "scanner passes clean + generic patterns -> hive-shareable"
   - What's unclear: The definition of "generic." Should it mean "no file paths, no project names, no repo-specific identifiers"?
   - Recommendation: Start with a simple heuristic -- if content contains no file paths (no `/` or `.` extensions), no project names (from COLONY_STATE.json goal), and no worker names, classify as hive-shareable. Refine based on real-world classification accuracy.

2. **Should ColonyStore.Compact() merge similar entries or just trim?**
   - What we know: HIVE-02 requires "compact" operation with configurable budgets
   - What's unclear: Whether compact means merge similar entries (dedup + combine evidence) or simply trim to budget
   - Recommendation: Phase 90 compact should trim to budget (remove lowest-confidence entries first). Merge/dedup can be added in Phase 91 when SQLite enables better similarity queries.

3. **How does the --no-learn flag interact with the existing --compact flag in colony-prime?**
   - What we know: D-16 says learning can be disabled by per-command flag (--no-learn)
   - What's unclear: Whether --no-learn should also suppress learned context injection into colony-prime (making it purely a write disable) or also suppress reads
   - Recommendation: --no-learn suppresses writes only (capture, classification, storage). Context injection from existing learned memory still works. This keeps the flag simple and prevents loss of accumulated context.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HIVE-01 | Aether-native learning concepts mapped from Hermes, with MIT notice if code is referenced | No Hermes code dependency exists. This is documentation/acknowledgment only. All learning concepts are Aether-native Go. |
| HIVE-02 | Colony memory store supports add, replace, remove, compact with configurable character/token budgets | LearnStore interface provides Add/Get/List/Replace/Remove/Compact. ColonyStore uses JSON in .aether/data/learn/. Budget configurable via config. |
| HIVE-03 | Colony memory injected into init/oracle/worker prompts as frozen snapshot; failed/empty builds never create durable memory | Context injection via colony-prime context_ranking.go. Frozen snapshot per D-14/15. Evidence-gated triggers prevent failed build learning (D-01/02/03). |
| LRN-01 | Post-run learning triggered only after verified successful outcomes; failed/empty/phantom runs logged as transient only | Trigger in continue-finalize after SAFE-03/04 + gates pass + all workers succeed (D-01/02/04). |
| LRN-02 | Every durable learning entry includes evidence: source run ID, worker, files touched, tests/gates passed, confidence, scope | Entry.Evidence struct with RunID, Phase, Workers, FilesTouched, GatesPassed/Total, Confidence, Timestamp, Scope (D-09). |
| LRN-03 | Promotion from repo to hive requires privacy scan and explicit user approval | HiveStore.Add() runs privacy scan. User approval required before promotion (D-10, D-11, D-12). |
| LRN-04 | Learned context injected into init/oracle/worker prompts ranked by phase, caste, file path, recency, confidence | ContextCandidate mapping: phase -> PriorityHint, caste -> relevance, file path -> relevance, recency -> freshness, confidence -> ConfirmationScore (D-13). |
| LRN-05 | Repo isolation -- two repos do not see each other's repo-local memory; hive entries are generic and redacted | ColonyStore scoped to .aether/data/learn/ (per-repo). HiveStore entries abstracted via promoteToHive() which removes repo paths. |
| LRN-06 | Export/import of repo learning packs with manifest, redaction report, and preview-before-apply | aether learn export/import subcommands. JSON manifest + redacted entries + redaction report. Preview before apply (D-12). |
| PRIV-03 | Learning entries classified as repo-local, hive-shareable, blocked, or needs-user-approval | Four-way Classification enum. Automatic classification at creation time via extended privacyScan() (D-10, D-11). |
| PRIV-04 | Trajectory records stored locally with strict redaction; export requires approval and redaction report | Learning entries stored in .aether/data/learn/ (repo-local). Export includes redaction report. User must approve export (D-12). |
| PRIV-05 | Learning writes can be disabled by config and by per-command flag | .planning/config.json `learning.enabled` (global default). --no-learn flag on build/continue commands (D-16). |

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified)

Phase 90 uses only Go standard library and existing packages within the repo. No new tools, CLIs, runtimes, databases, or external services are needed.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing`) |
| Config file | none -- standard `go test` |
| Quick run command | `go test ./pkg/learn/... -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LRN-01 | Learning does NOT fire when any worker failed | unit | `go test ./pkg/learn/... -run TestLearningTrigger -v` | No -- Wave 0 |
| LRN-01 | Learning does NOT fire when gates failed | unit | `go test ./pkg/learn/... -run TestLearningTrigger -v` | No -- Wave 0 |
| LRN-01 | Learning fires only after continue with all-green | unit | `go test ./pkg/learn/... -run TestLearningTrigger -v` | No -- Wave 0 |
| LRN-02 | Every entry has complete Evidence struct | unit | `go test ./pkg/learn/... -run TestEntryEvidence -v` | No -- Wave 0 |
| LRN-03 | Hive promotion blocked without privacy scan | unit | `go test ./pkg/learn/... -run TestHivePromotion -v` | No -- Wave 0 |
| LRN-04 | Context ranking uses phase/caste/file/recency/confidence | unit | `go test ./pkg/learn/... -run TestContextInjection -v` | No -- Wave 0 |
| LRN-05 | Two ColonyStore instances do not share entries | unit | `go test ./pkg/learn/... -run TestRepoIsolation -v` | No -- Wave 0 |
| LRN-06 | Export produces manifest + redaction report | unit | `go test ./pkg/learn/... -run TestExportImport -v` | No -- Wave 0 |
| HIVE-02 | ColonyStore Add/Replace/Remove/Compact | unit | `go test ./pkg/learn/... -run TestColonyStore -v` | No -- Wave 0 |
| HIVE-03 | Failed builds never create durable entries | unit | `go test ./pkg/learn/... -run TestNoLearningOnFailure -v` | No -- Wave 0 |
| PRIV-03 | Classification produces correct 4-way result | unit | `go test ./pkg/learn/... -run TestClassification -v` | No -- Wave 0 |
| PRIV-05 | --no-learn flag suppresses writes | unit | `go test ./pkg/learn/... -run TestLearningDisabled -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/learn/... -v -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/learn/learn.go` -- LearnStore interface, Entry/Evidence/Classification types
- [ ] `pkg/learn/colony_store_test.go` -- ColonyStore CRUD tests
- [ ] `pkg/learn/classify_test.go` -- Classification logic tests
- [ ] `pkg/learn/trigger_test.go` -- Evidence-gated trigger tests
- [ ] `pkg/learn/export_test.go` -- Export/import tests
- [ ] Framework install: none needed -- Go stdlib testing

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | yes | privacyScan() blocks secrets before storage; sanitize.go patterns for prompt injection in learning content |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go Learning System

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secret leakage via learning entries | Information Disclosure | privacyScan() blocks writes containing API keys, private keys, passwords, tokens, env files (PRIV-01/02) |
| Path traversal in learning export files | Tampering | Export writes to user-specified path; import validates manifest schema before applying |
| Cross-repo memory leakage | Information Disclosure | ColonyStore scoped to .aether/data/learn/ (per-repo); HiveStore abstracts repo-specific content via promoteToHive() (LRN-05) |
| Prompt injection via learning content | Tampering | SanitizeSignalContent patterns in sanitize.go; learned content is read-only in worker prompts (D-14) |
| Learning denial-of-service (budget exhaustion) | Denial of Service | Compact() trims to configurable budget; lowest-confidence entries removed first |

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis (2026-05-01): pkg/memory/ (6 files), pkg/colony/context_ranking.go, pkg/storage/storage.go, cmd/provenance.go, cmd/security_cmds.go, cmd/hive.go, cmd/learning.go, cmd/learning_cmds.go, cmd/colony_prime_context.go, cmd/codex_continue_finalize.go
- 90-CONTEXT.md -- User decisions D-01 through D-16
- 88-CONTEXT.md -- Phase 88 provenance and privacy decisions (D-08-D-11)
- REQUIREMENTS.md -- HIVE-01/02/03, LRN-01/02/03/04/05/06, PRIV-03/04/05

### Secondary (MEDIUM confidence)
- .planning/research/SUMMARY.md -- v1.13 research synthesis (pitfall 5: false learning confidence)

### Tertiary (LOW confidence)
- None -- all findings verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all existing packages verified in codebase, no new dependencies
- Architecture: HIGH - all integration points identified with specific file paths and line numbers
- Pitfalls: HIGH - derived from direct codebase analysis and Phase 88/89 experience

**Research date:** 2026-05-01
**Valid until:** 30 days (stable -- no external dependencies, all internal patterns)
