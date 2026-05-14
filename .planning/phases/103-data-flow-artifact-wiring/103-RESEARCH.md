# Phase 103: Data Flow & Artifact Wiring - Research

**Researched:** 2026-05-07
**Domain:** Go runtime data artifact tracing, colony-prime context injection wiring
**Confidence:** HIGH

## Summary

Phase 103 audits every data artifact in the Aether runtime, tracing each from its writer Go function to its reader/consumer. The core source of truth is `cmd/colony_prime_context.go`, which assembles 17+ named prompt sections from multiple data sources into the worker context that gets injected into every dispatched worker. The storage layer (`pkg/storage/storage.go`) provides `SaveJSON`, `LoadJSON`, `AppendJSONL`, `ReadJSONL`, `AtomicWrite`, and `ReadFile` -- all artifact I/O routes through these six methods.

The audit surface has two primary consumers: (1) colony-prime (`buildColonyPrimeOutput()`), which is the main context injection path for dispatched workers, and (2) the context capsule (`buildContextCapsuleOutput()`), which is the legacy/compact fallback path used by `resolveCodexWorkerContext()`. A third consumer path is user-facing CLI commands (status, resume, midden-review, etc.) that read artifacts for display without injecting into worker prompts.

Key discovery: graph artifacts (`instinct-graph.json`, `codebase-graph.json`) and survey artifacts (`survey/*.json`) are NOT wired into colony-prime context injection. They have specialized consumers (codegraph_context.go for build briefs, codex_plan.go for planning workers) but are absent from the 17 colony-prime sections. This is a gap the audit should flag.

**Primary recommendation:** Extract the full artifact-to-consumer map by grepping for `SaveJSON`/`AppendJSONL`/`AtomicWrite` calls (writers) and `LoadJSON`/`ReadJSONL`/`ReadFile` calls (readers), then cross-reference against the 17 colony-prime section names plus the context capsule section names plus user-facing CLI readers.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Audit EVERYTHING -- all files in `.aether/data/` (not just the named ones from DATA-01) plus all hub-level artifacts (`~/.aether/QUEEN.md`, `hive/wisdom.json`, `eternal/`, `registry/`). The ROADMAP names the core files, but a complete scan catches edge cases and newer artifacts that naming would miss.

- **D-02:** For each artifact, trace at command + prompt section level: (1) the Go function/subcommand that writes the file, (2) the Go function/subcommand that reads the file, (3) whether colony-prime injects the data into worker prompts (and which prompt section name). This is the sweet spot -- detailed enough to find gaps, practical enough to maintain.

- **D-03:** Verify actual wiring -- check whether graph artifacts (pkg/graph/) and survey results (.aether/data/survey/) are actually wired into colony-prime context injection. If they're not wired, document the gap as a finding. Don't just document current state; verify the wiring works.

- **D-04:** Report follows KNOWN-GAPS.md severity pattern from Phase 101 (Critical/Warning/Info tiers). Single combined report file (DATA-FLOW.md) covering all artifacts.

- **D-05:** Automated tests freeze findings -- following the Phase 102 pattern (golden snapshot + report verification tests). Tests verify the report's claims are accurate.

- **D-06:** No fix suggestions in findings. Phase 105 handles all remediation.

### Claude's Discretion
- Exact report file name and structure
- How to extract writer/reader function names from Go source (grep patterns vs AST)
- How to verify colony-prime wiring (read colony_prime_context.go section names vs runtime test)
- Whether to include artifact size or age data alongside writer/reader info
- Test file structure and naming
- How to handle artifacts that are read by user-facing CLI commands (status, resume) vs internal-only consumption

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LIFE-03 | No command produces dead-end artifacts that are never consumed by later commands or user-facing output | Writer-to-reader tracing for every `.aether/data/` file; dead-end detection flags artifacts with writers but no readers (or only test readers) |
| DATA-01 | Every artifact in .aether/data/ traced to a downstream consumer or explicitly documented as write-only-for-async | Complete artifact inventory with writer function, reader function, colony-prime section, and user-facing CLI consumers |
| DATA-02 | QUEEN.md, Hive Brain, and graph/survey artifacts are wired into colony-prime context injection or explicitly pruned | Colony-prime section map shows QUEEN.md (3 sections), Hive (1 section), graph (0 sections), survey (0 sections); gap documented |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Artifact persistence | Go runtime (pkg/storage/) | -- | All writes go through Store.SaveJSON/AtomicWrite |
| Context injection | Go runtime (cmd/colony_prime_context.go) | -- | Colony-prime assembles sections from artifacts |
| Artifact creation | Go runtime (cmd/*.go subcommands) | -- | Each lifecycle command writes specific artifacts |
| Audit report generation | Go runtime (Go test) | -- | Phase 102 pattern: Go test extracts truth, writes report |
| Hub-level artifact management | Go runtime (cmd/hive.go, cmd/queen.go) | -- | Hub artifacts live in ~/.aether/ |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (encoding/json) | 1.26.1 | JSON marshal/unmarshal for artifact files | Already in use throughout |
| github.com/spf13/cobra | existing | CLI command registration | All commands use Cobra |
| pkg/storage | existing | Atomic JSON file operations | All artifact I/O routes here |
| pkg/colony | existing | Colony data types (ColonyState, PheromoneFile, etc.) | Type definitions for all artifacts |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/graph | existing | Graph persistence (instinct-graph.json) | Tracing graph wiring |
| pkg/learn | existing | Learning pipeline (entries.json) | Tracing learned memory wiring |
| pkg/events | existing | Event bus (event-bus.jsonl) | Tracing event lifecycle |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Grep-based writer/reader extraction | Go AST parsing | AST is more precise but grep patterns are sufficient for this audit and match the Phase 100/102 approaches |

**Installation:**
No new packages needed. This phase uses existing Go stdlib and project packages only.

## Architecture Patterns

### System Architecture Diagram

```
Data Flow: Writer -> File -> Reader/Consumer

[Go Subcommands]          [.aether/data/ Files]        [Consumers]
                             |
init_cmd.go ------Write---> | COLONY_STATE.json ----Read---> colony-prime (state section)
                             |                        ----Read---> context-capsule (state section)
                             |                        ----Read---> status, resume, entomb
                             |
codex_build.go ---Write---> | pheromones.json  ------Read---> colony-prime (pheromones section)
pheromone_write.go          |                        ----Read---> context-capsule (signals section)
                             |                        ----Read---> suggest-analyze, entomb
                             |
codex_continue.go -Write--->| instincts.json    ------Read---> colony-prime (instincts section)
instinct_runtime.go         |                        ----Read---> queen-promote-instinct
                             |                        ----Read---> memory-health
                             |
codex_build.go ---Write---> | handoffs/worker-handoffs.json
codex_dispatch_contract.go  |                  ------Read---> colony-prime (worker_handoffs section)
                             |
codex_colonize.go -Write--->| survey/*.json     ------Read---> codex_plan.go (loadCodexSurveyContext)
                             |                        ----Read---> discuss.go, assumptions.go
                             |                        ----X-----> colony-prime (NOT wired)
                             |
codegraph.go ------Write--->| codebase-graph.json ----Read---> codegraph_context.go (build briefs)
                             |                        ----X-----> colony-prime (NOT wired)
                             |
graph_consolidation_cmds.go | instinct-graph.json ----Read---> graph_consolidation_cmds.go only
                             |                        ----X-----> colony-prime (NOT wired)
                             |
seal_final_review.go -Write>| reviews/{domain}/ledger.json
review_ledger.go            |                  ------Read---> colony-prime (prior_reviews section)
                             |                        ----Read---> status, continue review
                             |
[Hub Artifacts]
queen.go ----------Write--->| ~/.aether/QUEEN.md ------Read---> colony-prime (global_queen_md, user_preferences)
                             |                        ----Read---> context-capsule (queen_global, user_preferences)
                             |
hive.go -----------Write--->| ~/.aether/hive/wisdom.json -Read-> colony-prime (hive_wisdom section)
                             |                        ----Read---> context-capsule (hive_wisdom)
                             |
registry.go --------Write-->| ~/.aether/registry/registry.json -Read-> readRegistryDomainsForRepo
                             |                        ----Read---> hive domain filtering
```

### Recommended Project Structure
```
cmd/
├── data_flow_audit.go           # New: Go test helper that extracts artifact wiring
├── data_flow_audit_test.go      # New: Tests that verify DATA-FLOW.md accuracy
├── testdata/
│   └── data_flow_snapshot.json  # New: Golden snapshot of artifact wiring
.aether/data/                     # Audited artifacts (read-only)
.planning/phases/103-*/
└── DATA-FLOW.md                  # New: The audit report
```

### Pattern 1: Colony-Prime Section Registration
**What:** Each data source is registered as a `colonyPrimeSection` with a unique `name`, `title`, `source` file path, `priority`, and scoring fields.
**When to use:** When identifying which artifacts reach workers.
**Example:**
```go
// Source: cmd/colony_prime_context.go:388-399
sections = append(sections, colonyPrimeSection{
    name:              "state",
    title:             "Colony State",
    source:            statePath,   // .aether/data/COLONY_STATE.json
    content:           stateSection.String(),
    priority:          5,
    freshnessScore:    1.0,
    confirmationScore: 1.0,
    relevanceScore:    sectionRelevanceScore("state"),
})
```

### Pattern 2: Golden Snapshot + Report Verification (from Phase 102)
**What:** A Go struct captures the audit findings, serialized to `testdata/*.json`, and tests verify the report file matches the golden snapshot.
**When to use:** For the automated test that freezes DATA-FLOW.md findings.
**Example:**
```go
// Source: cmd/worker_economy_test.go (Phase 102 pattern)
type DataFlowSnapshot struct {
    Artifacts []ArtifactEntry `json:"artifacts"`
}
type ArtifactEntry struct {
    Name            string   `json:"name"`
    FilePath        string   `json:"file_path"`
    Writers         []string `json:"writers"`
    Readers         []string `json:"readers"`
    ColonyPrimeName string   `json:"colony_prime_name,omitempty"`
    CapsuleName     string   `json:"capsule_name,omitempty"`
    UserFacingReads []string `json:"user_facing_reads,omitempty"`
    DeadEnd         bool     `json:"dead_end"`
}
```

### Anti-Patterns to Avoid
- **Manual report writing without Go extraction:** The report must be generated from Go source analysis, not hand-written. Phase 102 proved that hand-written reports drift from code truth.
- **Testing the report exists without testing its content:** Tests must parse DATA-FLOW.md and verify each artifact entry has valid writer/reader function names that actually exist in the codebase.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Colony-prime section list | Manual enumeration | Read colony_prime_context.go section `name` fields | 17 sections already defined; manual lists drift |
| Writer/reader extraction | Custom Go parser | Grep for SaveJSON/LoadJSON/AppendJSONL patterns | Storage layer is the single chokepoint |
| Severity classification | New severity system | Phase 101 KNOWN-GAPS.md pattern | Established convention across v1.15 |

**Key insight:** All artifact I/O routes through `pkg/storage/Store` methods. This makes `SaveJSON`/`LoadJSON`/`AppendJSONL`/`ReadJSONL`/`AtomicWrite`/`ReadFile` the authoritative chokepoints for finding every writer and reader.

## Colony-Prime Section Map (Complete)

This is the authoritative list of all 17 colony-prime sections extracted from `buildColonyPrimeOutput()` in `cmd/colony_prime_context.go`:

| # | Section Name | Title | Source File | Priority | Protected |
|---|-------------|-------|-------------|----------|-----------|
| 1 | `state` | Colony State | COLONY_STATE.json | 5 | configurable |
| 2 | `review_depth` | Review Depth | COLONY_STATE.json | 6 | no |
| 3 | `pheromones` | Pheromone Signals | pheromones.json | 9 | configurable |
| 4 | `instincts` | Active Instincts | instincts.json (or COLONY_STATE.json fallback) | 6 | no |
| 5 | `decisions` | Key Decisions | COLONY_STATE.json | 3 | no |
| 6 | `learnings` | Phase Learnings | COLONY_STATE.json | 2 | no |
| 7 | `worker_handoffs` | Previous Worker Handoffs | handoffs/worker-handoffs.json | 4 | no |
| 8 | `hive_wisdom` | Hive Wisdom | ~/.aether/hive/wisdom.json | 4 | no |
| 9 | `learned_memory` | Learned Memory | entries.json (via learn.ColonyStore) | 5 | no |
| 10 | `global_queen_md` | Global Queen Wisdom | ~/.aether/QUEEN.md | 5 | configurable |
| 11 | `user_preferences` | User Preferences | ~/.aether/QUEEN.md + repo QUEEN.md | 7 | configurable |
| 12 | `prior_reviews` | Prior Reviews | reviews/{domain}/ledger.json | 8 | no |
| 13 | `local_queen_wisdom` | Local Queen Wisdom | repo/.aether/QUEEN.md | 5 | no |
| 14 | `clarified_intent` | Clarified Intent | pending-decisions.json | 8 | configurable |
| 15 | `blockers` | Active Blockers | pending-decisions.json (or flags.json) | 10 | configurable |
| 16 | `medic_health` | Colony Health Issues | medic-last-scan.json | 9 | configurable |

### Context Capsule Sections (separate consumer)

The context capsule (`buildContextCapsuleOutput()` in `cmd/context.go`) has its own sections, some overlapping with colony-prime:

| Section Name | Source File |
|-------------|-------------|
| `state` | COLONY_STATE.json |
| `signals` | pheromones.json |
| `decisions` | COLONY_STATE.json |
| `risks` | flags.json or pending-decisions.json |
| `recent_narrative` | rolling-summary.log |

## Complete Artifact Inventory

### Core .aether/data/ Artifacts

| Artifact File | Writer Functions | Reader Functions | Colony-Prime Section | Context Capsule | User-Facing CLI | Dead End? |
|---------------|-----------------|------------------|---------------------|-----------------|-----------------|-----------|
| COLONY_STATE.json | init_cmd.go, codex_plan.go, codex_build.go, codex_continue.go, codex_continue_finalize.go, state_cmds.go, entomb_cmd.go, session_flow_cmds.go, state_repair.go, phase_skip.go | colony_prime_context.go, context.go, status.go, codex_plan.go, codex_build.go, codex_continue.go, seal_final_review.go, entomb_cmd.go, recovery_snapshot.go, state_load.go, graph_consolidation_cmds.go, queen.go, init_cmd.go, discuss.go | state, review_depth, decisions, learnings | state | status, resume, plan | NO |
| pheromones.json | pheromone_write.go, pheromone_sync.go, suggest_approve.go, codex_build.go, codex_continue.go, seal_ceremony, entomb_cmd.go | colony_prime_context.go, context.go, suggest_analyze.go, build_flow_cmds.go, entomb_cmd.go, pheromones_read.go | pheromones | signals | pheromones display | NO |
| instincts.json | instinct_runtime.go, internal_cmds.go, codex_continue.go, codex_continue_finalize.go | colony_prime_context.go, queen.go, memory_health.go | instincts | -- | memory-details | NO |
| pending-decisions.json | discuss.go, flag_cmds.go, pending_decision.go, assumptions.go | colony_prime_context.go, context.go | clarified_intent, blockers | risks | flags list | NO |
| flags.json | flag_cmds.go, init_ceremony.go | colony_prime_context.go, context.go, shelf_seal.go | blockers (fallback) | risks (fallback) | flags list | NO (legacy fallback) |
| session.json | init_cmd.go, session_cmds.go, hook_cmds.go, recovery_snapshot.go | build_flow_cmds.go, recovery_snapshot.go, hook_cmds.go, session_cmds.go | -- | -- | session display | NO |
| handoffs/worker-handoffs.json | codex_dispatch_contract.go | colony_prime_context.go | worker_handoffs | -- | -- | NO |
| entries.json | learn.ColonyStore via codex_continue_finalize.go | colony_prime_context.go via learn.NewColonyStore | learned_memory | -- | -- | NO |
| midden.json (or midden/midden.json) | codex_build.go, codex_continue.go, entomb_cmd.go | midden_cmds.go, entomb_cmd.go, colony_prime_audit_test.go | -- | -- | midden-review, midden-recent-failures | Partial -- context capsule has midden section, but colony-prime does NOT inject midden. Audit test verifies it goes through context capsule path. |
| event-bus.jsonl | events package (ceremony_emitter.go, etc.) | colony_prime_test.go, ceremony_emitter_test.go, medic_scanner.go | -- | -- | -- (TTL cleanup by janitor) | Potential -- primarily consumed by tests and medic scanner; no colony-prime injection |
| behavior-observations.jsonl | profile.go (behavior-observe) | profile.go (profile analysis) | -- | -- | profile command | NO (consumed by profile pipeline) |
| profile.json | profile.go (generate) | profile.go (promote to QUEEN.md) | -- | -- | profile display | NO (promotes to QUEEN.md) |
| rolling-summary.log | context.go (context-update) | context.go (extractRollingSummary) | -- | recent_narrative | -- | NO |
| constraints.json | internal_cmds.go | medic_scanner.go, medic_repair.go, session_cmds.go | -- | -- | -- | LIKELY DEAD END -- Go code reads it only for medic scanning (flags it as ghost file). No colony-prime injection. Legacy pheromone predecessor. |
| assumptions.json | assumptions.go | -- (file-based, not programmatic reader) | -- | -- | assumption-list | Partial -- file is created by assumptions-analyze and consumed by user via assumption-list CLI. Not injected into workers. |
| medic-last-scan.json | medic_auto_spawn.go | colony_prime_context.go, medic_auto_spawn.go | medic_health | -- | -- | NO |
| colony.db | learn.NewSQLiteColonyStore | hive_search.go, skill_curator.go, skill_lifecycle.go | -- | -- | skill/hive commands | NO (SQLite DB for learning search) |

### Survey Artifacts (.aether/data/survey/)

| Artifact File | Writer | Reader | Colony-Prime? | Notes |
|---------------|--------|--------|---------------|-------|
| survey/blueprint.json | codex_colonize.go | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NO | Consumed by plan/discuss/assumptions commands, not colony-prime |
| survey/chambers.json | codex_colonize.go | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NO | Same pattern |
| survey/disciplines.json | codex_colonize.go | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NO | Same pattern |
| survey/provisions.json | codex_colonize.go | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NO | Same pattern |
| survey/pathogens.json | codex_colonize.go | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NO | Same pattern |

**D-03 Finding:** Survey artifacts are NOT wired into colony-prime. They are consumed by `loadCodexSurveyContext()` which is called by `codex_plan.go`, `discuss.go`, and `assumptions.go`. This is a specialized consumer path (planning workers) rather than the general worker injection path. The audit should classify this as "wired to specialized consumer, not colony-prime."

### Graph Artifacts

| Artifact File | Writer | Reader | Colony-Prime? | Notes |
|---------------|--------|--------|---------------|-------|
| codebase-graph.json | codegraph.go (graph-build), codex_colonize.go | codegraph_context.go (renderCodegraphContextForText), codegraph.go (graph-related) | NO | Injected into build worker briefs via codegraph_context.go, not colony-prime |
| instinct-graph.json | graph_consolidation_cmds.go | graph_consolidation_cmds.go (only) | NO | Read and written only by graph consolidation commands. Medic scanner checks it exists. |

**D-03 Finding:** Graph artifacts are NOT wired into colony-prime. `codebase-graph.json` is consumed by `codegraph_context.go` which adds a "Codebase Graph Context" section to build worker briefs -- a parallel injection path to colony-prime, but not through the colony-prime assembly. `instinct-graph.json` is consumed only by its own commands (graph-consolidation-merge, etc.) with no injection path.

### Review Artifacts (.aether/data/reviews/)

| Artifact File | Writer | Reader | Colony-Prime? | Notes |
|---------------|--------|--------|---------------|-------|
| reviews/{domain}/ledger.json | review_ledger.go (review-ledger-write), seal_final_review.go, codex_workflow_cmds.go | colony_prime_context.go (buildPriorReviewsSection), status.go, codex_workflow_cmds.go, entomb_cmd.go | YES (prior_reviews) | Accumulates across phases; survives session resets |
| reviews/_summary_cache.json | colony_prime_context.go (buildPriorReviewsSection) | colony_prime_context.go (buildPriorReviewsSection) | YES (internal cache) | Cache for prior reviews section; auto-invalidated when ledger changes |

### Hub-Level Artifacts

| Artifact File | Writer | Reader | Colony-Prime? | Notes |
|---------------|--------|--------|---------------|-------|
| ~/.aether/QUEEN.md | queen.go (queen-promote), profile.go | colony_prime_context.go (global_queen_md, user_preferences), context.go (queen_global, user_preferences) | YES (2 sections) | Cross-colony wisdom and user preferences |
| repo/.aether/QUEEN.md | queen.go (local writes) | colony_prime_context.go (local_queen_wisdom, user_preferences) | YES (2 sections) | Repo-specific wisdom |
| ~/.aether/hive/wisdom.json | hive.go (hive-store, hive-promote) | colony_prime_context.go (hive_wisdom), context.go (hive_wisdom) | YES | Domain-scoped cross-colony wisdom |
| ~/.aether/registry/registry.json | registry.go, install_cmd.go, entomb_cmd.go | context_weighting.go (readRegistryDomainsForRepo), registry.go, exchange.go, entomb_cmd.go | Indirect | Used to filter hive wisdom by domain -- not injected directly but controls what gets injected |
| ~/.aether/eternal/memory.json | internal_cmds.go (eternal-store) | context_weighting.go (readHiveWisdom fallback) | YES (fallback) | Legacy fallback when hive has no matching entries |

## Common Pitfalls

### Pitfall 1: Confusing "consumed by test code" with "consumed by production"
**What goes wrong:** An artifact appears to have readers, but all readers are in `*_test.go` files.
**Why it happens:** Test files often create and read artifacts to set up fixtures.
**How to avoid:** When counting readers, distinguish `_test.go` readers from production readers. Only production readers count as real consumers.
**Warning signs:** An artifact's only `LoadJSON` references appear in test files.

### Pitfall 2: Missing the context capsule path
**What goes wrong:** Focusing only on colony-prime and missing that `resolveCodexWorkerContext()` falls back to `buildContextCapsuleOutput()` when colony-prime produces insufficient context.
**Why it happens:** Two parallel context assembly functions exist.
**How to avoid:** Audit both `buildColonyPrimeOutput()` and `buildContextCapsuleOutput()` as separate consumer paths.

### Pitfall 3: Over-reporting dead ends for async artifacts
**What goes wrong:** Flagging `event-bus.jsonl` or `behavior-observations.jsonl` as dead ends when they serve async pipelines (TTL cleanup, profile promotion).
**Why it happens:** The immediate consumer isn't a worker prompt section.
**How to avoid:** Classify artifacts as: (a) colony-prime injected, (b) context capsule injected, (c) user-facing CLI consumed, (d) async pipeline consumed, (e) dead end. Only (e) is a true finding.

### Pitfall 4: Missing the constraints.json ghost file
**What goes wrong:** `constraints.json` exists and has writers, but the Go runtime ignores its content. The medic scanner explicitly flags it as a "ghost file."
**Why it happens:** Legacy from the pheromone predecessor system.
**How to avoid:** Include medic scanner findings in the audit. Constraints.json is a known ghost file already flagged by medic.

### Pitfall 5: Survey/Graph wiring gap misclassification
**What goes wrong:** Marking survey and graph artifacts as "unwired" when they have specialized injection paths outside colony-prime.
**Why it happens:** Colony-prime is the primary injection path, but not the only one.
**How to avoid:** Distinguish "not wired at all" from "wired to specialized consumer, not colony-prime." The DATA-02 requirement asks specifically about colony-prime wiring, so the gap is real, but the severity should be Info rather than Warning since the data does reach workers through alternative paths.

## Code Examples

### Extracting colony-prime section names
```go
// Source: cmd/colony_prime_context.go (pattern repeated 16 times)
// Each section has a unique "name" field identifying the prompt section.
// The audit can extract these by finding all colonyPrimeSection literals
// with a "name:" field.

// Grep pattern to find all section names:
// grep -n 'name:\s*"' cmd/colony_prime_context.go | grep -v 'colonyPrimeSection'
```

### Writer extraction pattern
```go
// Source: pkg/storage/storage.go
// All writers use one of these patterns:
//   store.SaveJSON("filename.json", data)
//   store.AtomicWrite("filename.log", data)
//   store.AppendJSONL("filename.jsonl", entry)
//   store.UpdateFile("filename.json", mutate)
//   store.UpdateJSONAtomically("filename.json", &data, mutate)
//   os.WriteFile(fullPath, data, 0644)  // some legacy paths

// Grep pattern for finding writers:
// grep -rn 'SaveJSON\|AtomicWrite\|AppendJSONL\|UpdateFile\|UpdateJSONAtomically' cmd/*.go | grep -v _test.go
```

### Reader extraction pattern
```go
// Source: pkg/storage/storage.go
// All readers use one of these patterns:
//   store.LoadJSON("filename.json", &dest)
//   store.ReadJSONL("filename.jsonl")
//   store.ReadFile("filename.log")
//   store.LoadRawJSON("filename.json")
//   os.ReadFile(fullPath)

// Grep pattern for finding readers:
// grep -rn 'LoadJSON\|ReadJSONL\|ReadFile\|LoadRawJSON' cmd/*.go | grep -v _test.go
```

### Review ledger persistence verification
```go
// Source: cmd/review_ledger.go:113
// Writers: review-ledger-write, seal_final_review.go, codex_workflow_cmds.go
// Path pattern: reviews/{domain}/ledger.json
// Domains: security, quality, performance, resilience, testing, history, bugs

// Reader: colony_prime_context.go buildPriorReviewsSection()
// Reads all 7 domain ledgers, filters open findings, assembles section.
// Cache: reviews/_summary_cache.json (auto-invalidated on ledger changes)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| constraints.json as pheromone predecessor | pheromones.json | v1.0+ | constraints.json is now a ghost file (medic flags it) |
| instincts inside COLONY_STATE.json only | standalone instincts.json | v1.13+ | Dual path with fallback; colony-prime checks instincts.json first |
| flags.json for blockers | pending-decisions.json with flags.json fallback | v1.12+ | Dual format; colony-prime tries pending-decisions.json first |
| context-capsule as primary worker context | colony-prime as primary, capsule as fallback | v1.14+ | resolveCodexWorkerContext() tries colony-prime first, falls back to capsule |

**Deprecated/outdated:**
- `constraints.json`: Ghost file flagged by medic scanner. Written by `internal_cmds.go` but Go runtime ignores content.
- `flags.json`: Legacy format for blockers. Superseded by `pending-decisions.json` but still supported as fallback.
- `eternal/memory.json`: Legacy memory system. Hive Brain is the current system; eternal is only a fallback.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Survey artifacts are NOT wired into colony-prime but ARE consumed by planning workers through loadCodexSurveyContext() | Graph & Survey Wiring | Finding severity would change |
| A2 | codebase-graph.json is injected into build worker briefs via codegraph_context.go, not through colony-prime | Graph & Survey Wiring | Finding severity would change |
| A3 | instinct-graph.json has no consumer other than its own consolidation commands | Graph & Survey Wiring | If wrong, there's a hidden reader path |
| A4 | constraints.json is a ghost file (medic-flagged) with no meaningful production reader | Core Artifacts | If wrong, a consumer exists that was missed |
| A5 | event-bus.jsonl is primarily consumed by tests and medic scanner; no colony-prime injection | Core Artifacts | If wrong, there's a production consumer path |

**All claims in this research were verified by reading source code -- no user confirmation needed for verified claims. Assumptions A1-A5 are based on code grep results and should be validated by the planner's test approach.**

## Open Questions

1. **Should the audit include spawn-tree.txt and runtime-spawn-runs.jsonl?**
   - What we know: These are mentioned in plan test cleanup code but are transient artifacts
   - What's unclear: Whether they persist between sessions or are regenerated each run
   - Recommendation: Include in the inventory but classify as transient/regenerated (not durable state)

2. **Should the audit include the colony.db SQLite database?**
   - What we know: Used by learn.NewSQLiteColonyStore for skill lifecycle, hive search, and curator
   - What's unclear: Whether its data overlaps with entries.json or is a separate storage path
   - Recommendation: Include in inventory as a non-JSON artifact with specialized readers

3. **How to handle the pr-context command which reads 10+ artifacts?**
   - What we know: `cmd/context.go` has a `prContextCmd` that assembles comprehensive context from many sources
   - What's unclear: Whether this counts as a "consumer" in the DATA-01 sense
   - Recommendation: Include as a user-facing CLI consumer, but note it's not a worker injection path

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Test execution | yes | 1.26.1 | -- |
| go test | Automated test verification | yes | -- | -- |
| go vet | Code analysis | yes | -- | -- |

**Missing dependencies with no fallback:** None

**Missing dependencies with fallback:** None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- see Wave 0 |
| Quick run command | `go test ./cmd/ -run TestDataFlow -count=1` |
| Full suite command | `go test ./cmd/ -count=1 -timeout 120s` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LIFE-03 | Dead-end detection: artifacts with writers but no production readers | unit | `go test ./cmd/ -run TestDataFlowDeadEnds -count=1` | Wave 0 |
| DATA-01 | Complete artifact inventory with writer/reader/section mapping | unit | `go test ./cmd/ -run TestDataFlowSnapshot -count=1` | Wave 0 |
| DATA-02 | Graph/survey wiring verification against colony-prime sections | unit | `go test ./cmd/ -run TestDataFlowWiring -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestDataFlow -count=1`
- **Per wave merge:** `go test ./... -count=1 -timeout 300s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/data_flow_audit_test.go` -- covers LIFE-03, DATA-01, DATA-02
- [ ] `cmd/testdata/data_flow_snapshot.json` -- golden snapshot
- [ ] Report file DATA-FLOW.md -- created during Wave 1

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Not applicable -- audit-only phase |
| V3 Session Management | no | Not applicable |
| V4 Access Control | no | Not applicable |
| V5 Input Validation | no | Not applicable -- read-only audit |
| V6 Cryptography | no | Not applicable |

### Known Threat Patterns for Data Flow Audit

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| None applicable | -- | Read-only audit phase, no code changes |

This is a read-only audit phase with no code modifications. Security findings from the audit itself (e.g., dead-end artifacts that might contain sensitive data) are reported as findings for Phase 105 remediation.

## Sources

### Primary (HIGH confidence)
- `cmd/colony_prime_context.go` -- Full source read, all 16 section names and source files extracted
- `cmd/context.go` -- Context capsule sections and buildContextCapsuleOutput() analysis
- `pkg/storage/storage.go` -- Storage API surface (SaveJSON, LoadJSON, AppendJSONL, ReadJSONL, AtomicWrite, ReadFile)
- `cmd/codex_plan.go` -- Survey context loading (loadCodexSurveyContext)
- `cmd/codegraph_context.go` -- Graph wiring to build worker briefs
- `cmd/graph_consolidation_cmds.go` -- Instinct graph readers/writers
- `cmd/review_ledger.go` -- Review ledger CRUD and domain structure
- `cmd/worker_economy_test.go` -- Phase 102 golden snapshot + report verification pattern
- `cmd/colony_prime_audit_test.go` -- AAC-005 audit pattern for required sections

### Secondary (MEDIUM confidence)
- Grep-based extraction of SaveJSON/LoadJSON patterns across cmd/*.go -- verified by reading source of key files
- Artifact file paths cross-referenced with CONTEXT.md canonical references

### Tertiary (LOW confidence)
- None -- all findings verified against source code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new packages, existing Go stdlib and project packages only
- Architecture: HIGH -- colony-prime section map extracted directly from source code
- Pitfalls: HIGH -- based on Phase 102 experience and direct source code analysis

**Research date:** 2026-05-07
**Valid until:** 2026-06-07 (30 days -- stable codebase during v1.15 audit milestone)
