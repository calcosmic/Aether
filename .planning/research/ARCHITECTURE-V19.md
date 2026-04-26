# Architecture Research: Review Findings Persistence (v1.9)

**Domain:** Aether internal feature integration (v1.9 milestone)
**Researched:** 2026-04-26
**Confidence:** HIGH (all findings verified against source code)

## System Overview

Review persistence sits inside the existing Aether runtime, touching four established subsystems: continue-flow report writing, CLI subcommand registration, colony-prime context injection, and agent tool definitions. No new subsystems are needed; the work is extension of existing patterns.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CONTINUE FLOW                                 │
│  ┌───────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────┐    │
│  │ Review    │──>│ Review   │──>│ Worker   │──>│ Continue     │    │
│  │ Workers   │   │ report   │   │ Reports  │   │ Advance      │    │
│  │ (3 specs) │   │ .json    │   │ .md NEW  │   │ + Ledger     │    │
│  └───────────┘   └──────────┘   └──────────┘   │ write NEW    │    │
│                                                  └──────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│                     DOMAIN LEDGER (NEW)                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  cmd/review_ledger.go                                        │    │
│  │  review-ledger-write / read / summary / resolve              │    │
│  │  Storage: .aether/data/reviews/{domain}/ledger.json          │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│                     COLONY PRIME                                     │
│  ┌───────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────┐    │
│  │ State     │   │ Pheromon │   │ Prior    │   │ Instincts    │    │
│  │ (pri 5)   │   │ (pri 9)  │   │ Reviews  │   │ (pri 6)      │    │
│  └───────────┘   └──────────┘   │ (pri 8)  │   └──────────────┘    │
│                                  │ NEW      │                       │
│                                  └──────────┘                       │
├─────────────────────────────────────────────────────────────────────┤
│                     AGENT DEFINITIONS                                │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  7 agents get Write tool + findings instructions             │    │
│  │  Mirrors: .claude/, .opencode/, .codex/, .aether/agents-*   │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Integration Points (Verified Against Source)

### 1. Continue-Review Worker Outcome Reports

**Existing pattern to mirror:** `writeCodexBuildOutcomeReports()` at `cmd/codex_build.go:1125` writes per-worker `.md` files under `build/phase-N/worker-reports/{name}.md` using `store.AtomicWrite()`. The render function `renderCodexBuildWorkerOutcomeReport()` at line 1165 formats the markdown.

**Where to add:**
- New function `writeCodexContinueOutcomeReports()` in `cmd/codex_continue_finalize.go` (after `review.json` write at line 166-168)
- New function `renderCodexContinueWorkerOutcomeReport()` in the same file
- Call site: `runCodexContinueFinalize()` after the `reviewReportRel` write succeeds (line 168), and in `runCodexContinue()` after the `reviewReportRel` write (line 472)
- Output path: `build/phase-N/worker-reports/{name}.md` (same directory structure as build reports)
- Add `report` field to `codexContinueWorkerFlowStep` struct for full text capture
- Add `Report` field to `codexContinueExternalDispatch` struct for external task lane
- Preserve new fields in `mergeExternalContinueResults()` (currently drops them)

**Struct changes needed in `cmd/codex_continue.go`:**
```go
type codexContinueWorkerFlowStep struct {
    Stage   string  `json:"stage,omitempty"`
    Caste   string  `json:"caste,omitempty"`
    Name    string  `json:"name"`
    Task    string  `json:"task,omitempty"`
    Status  string  `json:"status"`
    Summary string  `json:"summary,omitempty"`
    Report  string  `json:"report,omitempty"`    // NEW: full markdown report
}
```

**Struct changes needed in `cmd/codex_continue_plan.go`:**
```go
type codexContinueExternalDispatch struct {
    // ... existing fields ...
    Report   string  `json:"report,omitempty"`    // NEW: full markdown report
}
```

**Wrapper completion packet change:** Add `report` field to each dispatch entry in the continue wrapper completion packet. Both `.claude/commands/ant/continue.md` (line 70-88) and `.opencode/commands/ant/continue.md` must document the field.

**Import needed:** Add `strconv` to `cmd/codex_continue_finalize.go` for duration formatting (mirroring `codex_build.go`).

### 2. Domain-Ledger CLI Subcommands

**Existing pattern to follow:** `cmd/hive.go` defines `hive-init`, `hive-store`, `hive-read`, `hive-abstract`, `hive-promote` using cobra commands registered via `rootCmd.AddCommand()` in `init()`. Storage uses `os.ReadFile` / `os.WriteFile` with manual JSON marshal, not `store.SaveJSON()` (because hive lives in the hub, not `.aether/data/`).

**Ledger lives in `.aether/data/`**, so use `store.SaveJSON()` and `store.LoadJSON()` instead of raw file I/O. This matches the pheromone pattern (`cmd/pheromone_write.go` lines 142-176).

**New file:** `cmd/review_ledger.go`

**Subcommands:**

| Subcommand | Flags | Purpose |
|------------|-------|---------|
| `review-ledger-write` | `--domain`, `--agent`, `--agent-name`, `--phase`, `--phase-name`, `--severity`, `--file`, `--line`, `--category`, `--description`, `--suggestion` | Write a single ledger entry |
| `review-ledger-read` | `--domain`, `--status` | Read entries with optional filters |
| `review-ledger-summary` | (none) | Aggregate summary across all domains |
| `review-ledger-resolve` | `--id` | Mark an entry as resolved |

**Ledger entry struct:**
```go
type reviewLedgerEntry struct {
    ID          string `json:"id"`
    Phase       int    `json:"phase"`
    PhaseName   string `json:"phase_name"`
    Agent       string `json:"agent"`
    AgentName   string `json:"agent_name"`
    GeneratedAt string `json:"generated_at"`
    Status      string `json:"status"`       // "open" | "resolved"
    Severity    string `json:"severity"`
    File        string `json:"file,omitempty"`
    Line        int    `json:"line,omitempty"`
    Category    string `json:"category"`
    Description string `json:"description"`
    Suggestion  string `json:"suggestion,omitempty"`
    ResolvedAt  string `json:"resolved_at,omitempty"`
}
```

**ID generation pattern:** Deterministic IDs like `sec-2-001` (prefix from domain abbreviation, phase number, sequence number). Domain abbreviations: `sec`=security, `qual`=quality, `perf`=performance, `res`=resilience, `test`=testing, `hist`=history, `bug`=bugs.

**Agent-to-domain mapping (hardcoded in review_ledger.go):**
```go
var agentDomainMap = map[string][]string{
    "gatekeeper":    {"security"},
    "auditor":       {"quality", "security"},
    "chaos":         {"resilience"},
    "watcher":       {"testing", "quality"},
    "archaeologist": {"history"},
    "measurer":      {"performance"},
    "tracker":       {"bugs"},
    "probe":         {"testing"},
}
```

**Storage path:** `reviews/{domain}/ledger.json` relative to `.aether/data/` (i.e., `store.SaveJSON("reviews/security/ledger.json", data)`). The store creates parent directories on write (verified in `pkg/storage/storage.go:88`).

### 3. Colony-Prime Prior-Reviews Section

**Existing pattern:** `buildColonyPrimeOutput()` in `cmd/colony_prime_context.go` builds sections as `colonyPrimeSection` structs with priority, content, freshness, confirmation, and relevance scores. Sections are appended in order, then ranked by the shared `colony.RankContextCandidates()` function within the budget.

**Current section priorities (from source):**

| Section | Priority | Source |
|---------|----------|--------|
| learnings | 2 | COLONY_STATE.json |
| decisions | 3 | COLONY_STATE.json |
| hive_wisdom | 4 | ~/.aether/hive/wisdom.json |
| state | 5 | COLONY_STATE.json |
| local_queen_wisdom | 5 | local QUEEN.md |
| instincts | 6 | instincts.json or COLONY_STATE.json |
| user_preferences | 7 | ~/.aether/QUEEN.md |
| clarified_intent | 8 | pending-decisions.json |
| pheromones | 9 | pheromones.json |
| blockers | 10 | pending-decisions.json |
| medic_health | 9 | medic-last-scan.json |

**Proposed:** `prior_reviews` section at priority 8 (between user_preferences at 7 and pheromones at 9, same as clarified_intent). This positions review findings as highly relevant context that informs how workers approach their tasks, above raw signal data but below active blockers.

**Implementation in `buildColonyPrimeOutput()`:**
1. After the `blockers` section assembly (line 450), add a new block that reads from all 7 domain ledgers
2. For each domain with an existing ledger file, read the open (unresolved) entries
3. Build a markdown section showing open findings per domain with severity counts
4. Cap the section at a reasonable character budget (suggest 1500 chars total, trimming older/low-severity entries first)
5. Add to sections slice with priority 8

```go
// After blockers section assembly, before final ranking:
if priorReviews := buildPriorReviewsSection(store, now); priorReviews != "" {
    sections = append(sections, colonyPrimeSection{
        name:              "prior_reviews",
        title:             "Prior Review Findings",
        source:            filepath.Join(store.BasePath(), "reviews"),
        content:           priorReviews,
        priority:          8,
        freshnessScore:    latestReviewFreshness(now, reviewEntries),
        confirmationScore: 0.90,
        relevanceScore:    sectionRelevanceScore("prior_reviews"),
    })
}
```

**Helper function:** `buildPriorReviewsSection(store, now)` reads all 7 domain ledgers, collects open entries, sorts by severity (critical first), and formats a compact summary.

### 4. Agent Definition Updates

**Current state:** Agent definitions live in 4 locations that must stay in sync:
- `.claude/agents/ant/*.md` (canonical Claude Code)
- `.aether/agents-claude/*.md` (packaging mirror for Claude)
- `.opencode/agents/*.md` (OpenCode)
- `.codex/agents/*.toml` (Codex)

**7 agents need Write tool added:**

| Agent | Current Tools | Target Tools |
|-------|---------------|--------------|
| aether-gatekeeper | Read, Grep, Glob | Read, Grep, Glob, Write |
| aether-auditor | Read, Grep, Glob | Read, Grep, Glob, Write |
| aether-chaos | Read, Bash, Grep, Glob | Read, Bash, Grep, Glob, Write |
| aether-watcher | (verify current) | + Write |
| aether-archaeologist | (verify current) | + Write |
| aether-measurer | (verify current) | + Write |
| aether-tracker | (verify current) | + Write |

**YAML frontmatter change** (parsed by `pkg/llm/config.go` `ParseAgentSpec()`):
```yaml
tools: Read, Grep, Glob, Write
```

**Body additions per agent:**
1. Add a `<review_findings>` section to each agent's markdown body describing:
   - How to format findings for the ledger (domain, severity, file, line, category, description, suggestion)
   - The `aether review-ledger-write` command to persist each finding
   - Write-scope guardrails: only write to `reviews/` directory, never to source code or colony state
2. Add a `<write_scope>` section declaring the write boundaries

**Write-scope guardrail pattern:**
```
<write_scope>
You may ONLY use the Write tool to persist review findings via:
  aether review-ledger-write --domain <domain> --agent <agent> ...

You must NOT:
- Write to source code files
- Write to .aether/data/ state files
- Write to any path outside .aether/data/reviews/
- Modify existing code or tests
</write_scope>
```

**Note on current agent boundaries:** The gatekeeper agent explicitly states "No Write. No Edit. No Bash." (line 13 of aether-gatekeeper.md). This boundary must be updated carefully -- the agent stays read-only for source code but gains Write specifically for persisting review findings through the CLI subcommand. The agent body should frame this as "write findings to the review ledger" not "write files."

## Recommended Project Structure

```
cmd/
  codex_continue.go              # MODIFY: add Report field to codexContinueWorkerFlowStep
  codex_continue_plan.go         # MODIFY: add Report field to codexContinueExternalDispatch
  codex_continue_finalize.go     # MODIFY: add writeCodexContinueOutcomeReports(), render func
  colony_prime_context.go        # MODIFY: add prior_reviews section (priority 8)
  review_ledger.go               # NEW: 4 subcommands (write, read, summary, resolve)
  review_ledger_test.go          # NEW: unit tests for ledger CRUD

.aether/agents-claude/             # MODIFY: 7 agent files (Write tool + findings sections)
.claude/agents/ant/                # MODIFY: 7 agent files (canonical, must match agents-claude)
.opencode/agents/                  # MODIFY: 7 agent files (structural parity)
.codex/agents/                     # MODIFY: 7 agent TOML files (Codex translation)
.claude/commands/ant/continue.md   # MODIFY: add report field to completion packet docs
.opencode/commands/ant/continue.md # MODIFY: add report field to completion packet docs
```

## Data Flow

### Continue Flow with Reports

```
User runs /ant-continue
    |
    v
Wrapper: aether continue --plan-only
    |
    v
Runtime: runCodexContinuePlanOnly() -> manifest with dispatches
    |
    v
Wrapper: spawns review workers (Gatekeeper, Auditor, Probe)
    |        Each worker runs aether review-ledger-write per finding
    v
Wrapper: collects results, writes completion packet with report field
    |
    v
Runtime: aether continue-finalize --completion-file <path>
    |
    v
runCodexContinueFinalize():
    1. mergeExternalContinueResults() -> workerFlow (with Report fields)
    2. Run verification, assessment, gates
    3. Write review.json (existing)
    4. writeCodexContinueOutcomeReports() -> .md per worker (NEW)
    5. If passed: advanceExternalContinue() -> state mutation
    6. Colony-prime now includes prior_reviews from ledger data
```

### Ledger Write Flow (from Agent)

```
Review Agent finds issue
    |
    v
Agent calls: aether review-ledger-write
    --domain security
    --agent gatekeeper
    --agent-name "Bastion-42"
    --phase 2
    --phase-name "Add auth"
    --severity HIGH
    --file "auth/handler.go"
    --line 47
    --category "hardcoded-secret"
    --description "API key hardcoded in handler"
    --suggestion "Move to environment variable"
    |
    v
review_ledger.go:
    1. Resolve .aether/data/reviews/security/ledger.json
    2. Load existing entries (or init empty)
    3. Compute deterministic ID: sec-2-001
    4. Append entry
    5. store.SaveJSON()
```

### Colony-Prime Injection Flow

```
buildColonyPrimeOutput()
    |
    v
buildPriorReviewsSection():
    1. For each of 7 domains:
        a. store.LoadJSON("reviews/{domain}/ledger.json")
        b. Filter open (unresolved) entries
    2. Sort by severity (critical > high > medium > low > info)
    3. Format compact markdown summary
    4. Cap at ~1500 chars (trim low-severity first)
    |
    v
Sections ranked by colony.RankContextCandidates()
    -> Included in worker prompt context
```

## Architectural Patterns

### Pattern 1: Mirror Build Report Writer

**What:** The continue outcome reports use the exact same pattern as build outcome reports: a `write*()` function iterates dispatches, renders markdown per worker, and writes via `store.AtomicWrite()`.

**When:** Any flow that spawns workers and needs per-worker persisted output.

**Trade-offs:**
- Pro: Consistent UX, workers and users can inspect detailed per-worker results
- Pro: Survives `/clear` because files persist on disk
- Con: Additional I/O per continue run (negligible -- 3 small files)

### Pattern 2: CLI-First Ledger with Store

**What:** Ledger CRUD is exposed as CLI subcommands that use `store.SaveJSON()` / `store.LoadJSON()`, following the pheromone pattern rather than the hive pattern (which uses raw `os.ReadFile` because hive lives outside `.aether/data/`).

**When:** Any structured data that lives within `.aether/data/` and needs programmatic read/write.

**Trade-offs:**
- Pro: File locking comes for free via store
- Pro: Atomic writes prevent corruption
- Pro: Agents can write via CLI invocation
- Con: Each subcommand is a separate process invocation from agents

### Pattern 3: Priority-Based Context Injection

**What:** Prior reviews join the colony-prime section list at priority 8, ranked alongside pheromones (9), blockers (10), and instincts (6). The existing ranking system decides what fits in the budget.

**When:** Any new data type that should influence worker behavior.

**Trade-offs:**
- Pro: Works within existing budget system, no special-casing
- Pro: Graceful degradation -- low-priority items get trimmed first
- Con: If many domains have many open findings, the section can be large and crowd out other context

## Anti-Patterns

### Anti-Pattern 1: Ledger Outside .aether/data/

**What people do:** Put the review ledger in a hub-level directory like `~/.aether/reviews/`
**Why it is wrong:** Review findings contain code-specific file paths and line numbers that go stale across colonies. The PROJECT.md explicitly states "Not cross-colony -- findings contain code-specific file paths that go stale."
**Do this instead:** Store in `.aether/data/reviews/` (colony-scoped, gitignored, archived at entomb).

### Anti-Pattern 2: Agents Writing Directly to JSON

**What people do:** Have agents construct JSON and write ledger files directly.
**Why it is wrong:** Breaks the wrapper-runtime contract ("wrappers may add colony framing and narration but must not mutate state"). Agents should invoke the Go runtime CLI subcommand.
**Do this instead:** Agents call `aether review-ledger-write` with flags. The runtime validates, assigns IDs, and writes atomically.

### Anti-Pattern 3: Modifying review.json Schema

**What people do:** Embed full findings in the existing `review.json` to avoid new files.
**Why it is wrong:** `review.json` is a pass/fail gate report consumed by continue-flow logic. Bloating it with per-finding detail changes its contract with every reader.
**Do this instead:** Keep `review.json` as the gate report. Per-worker reports go to `worker-reports/`. Structured findings go to the domain ledger. Three separate concerns, three separate storage locations.

### Anti-Pattern 4: Cross-Colony Review Sharing

**What people do:** Promote review findings to the hive brain for cross-colony learning.
**Why it is wrong:** Findings reference specific files (`auth/handler.go:47`) that do not exist in other repos.
**Do this instead:** Let high-value patterns from reviews enter the hive through the existing instinct pipeline (instinct -> hive-promote at seal), where the abstraction step strips repo-specific paths.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0-10 phases | All 7 domain ledgers accumulate, each stays under 50 entries. Colony-prime section is compact. |
| 10-50 phases | Entries per domain could reach 100+. Colony-prime prior-reviews section should cap display at top N per domain. Consider severity-based summary (counts only). |
| 50+ phases | Resolve old entries proactively. Seal/entomb should archive the full `reviews/` directory. Consider auto-resolving entries from completed phases. |

### Scaling Priorities

1. **First bottleneck:** Colony-prime section size if many open findings exist. Mitigate with character cap in `buildPriorReviewsSection()` (trim low-severity entries first, keep critical/high).
2. **Second bottleneck:** Ledger file I/O during every `buildColonyPrimeOutput()` call (reads 7 JSON files). Mitigate with the existing `cache.SessionCache` pattern already used in `colony_prime_context.go` (line 143-144).

## Build Order (Dependencies Considered)

The order matters because later phases depend on earlier phases being complete and tested.

### Phase 1: Continue-Review Worker Reports (Part A)

**Why first:** This is the simplest change -- it extends an existing flow with a pattern that already exists for builds. No new CLI commands, no new data formats. Just adding a field and two functions.

**Files:**
- `cmd/codex_continue.go` -- add `Report` field to `codexContinueWorkerFlowStep`
- `cmd/codex_continue_plan.go` -- add `Report` field to `codexContinueExternalDispatch`
- `cmd/codex_continue_finalize.go` -- add `writeCodexContinueOutcomeReports()`, `renderCodexContinueWorkerOutcomeReport()`, call from `runCodexContinueFinalize()` and preserve in `mergeExternalContinueResults()`
- `.claude/commands/ant/continue.md` -- document `report` field in completion packet
- `.opencode/commands/ant/continue.md` -- same

**Tests:**
- Unit test for `writeCodexContinueOutcomeReports()` verifying file creation
- Unit test for `renderCodexContinueWorkerOutcomeReport()` verifying markdown format
- Integration test for full continue-finalize flow producing worker reports

### Phase 2: Domain-Ledger CRUD (Part B core)

**Why second:** The ledger system is independent infrastructure that later phases (colony-prime, agent writes) depend on. Getting CRUD right and tested first means downstream consumers have a stable API.

**Files:**
- `cmd/review_ledger.go` -- 4 subcommands, entry struct, domain map, ID generation
- `cmd/review_ledger_test.go` -- CRUD tests, ID determinism, domain validation

**Tests:**
- Unit test for each subcommand (write, read, summary, resolve)
- Test deterministic ID generation (`sec-2-001` format)
- Test store.SaveJSON creates parent directories
- Test concurrent write safety (two writes to same domain ledger)
- Test summary aggregation across domains

### Phase 3: Colony-Prime Prior-Reviews Section (Part B context)

**Why third:** Depends on Phase 2 (must be able to read ledger files). Extends the context injection system.

**Files:**
- `cmd/colony_prime_context.go` -- add `buildPriorReviewsSection()`, integrate after blockers section

**Tests:**
- Unit test for `buildPriorReviewsSection()` with empty ledgers
- Unit test with populated ledgers across multiple domains
- Unit test verifying priority 8 ranking behavior
- Unit test verifying character budget cap

### Phase 4: Agent Definition Updates (Part B agents)

**Why fourth:** Depends on Phase 2 (agents need `review-ledger-write` to exist). Also the highest-touch change (7 agent files x 4 locations = 28 file edits).

**Files:**
- 7 files in `.claude/agents/ant/`
- 7 files in `.aether/agents-claude/`
- 7 files in `.opencode/agents/`
- 7 files in `.codex/agents/`

**Tests:**
- Verify `pkg/llm/config.go` `ParseAgentSpec()` still parses all modified agents
- Verify Write tool appears in parsed config
- Manual: verify agent behavior with review-ledger-write command

### Phase 5: Lifecycle Integration

**Why last:** Touches existing lifecycle commands that already work. Seal and entomb need to handle the new `reviews/` directory. Status should show ledger counts.

**Files:**
- `cmd/entomb_cmd.go` -- ensure `copyEntombArtifacts()` copies `reviews/` directory
- Status/seal commands -- add review counts to display if appropriate

**Tests:**
- Test entomb includes review ledger files
- Test status output includes review counts

## Component Responsibility Matrix

| Component | Responsibility | New or Modified | Files |
|-----------|----------------|-----------------|-------|
| `codexContinueWorkerFlowStep` | Carry per-worker report text | Modified | `cmd/codex_continue.go` |
| `codexContinueExternalDispatch` | Carry per-worker report from wrappers | Modified | `cmd/codex_continue_plan.go` |
| `writeCodexContinueOutcomeReports()` | Write per-worker .md files | New | `cmd/codex_continue_finalize.go` |
| `renderCodexContinueWorkerOutcomeReport()` | Format worker report markdown | New | `cmd/codex_continue_finalize.go` |
| `review-ledger-write` | Persist a single finding | New | `cmd/review_ledger.go` |
| `review-ledger-read` | Query findings with filters | New | `cmd/review_ledger.go` |
| `review-ledger-summary` | Aggregate counts across domains | New | `cmd/review_ledger.go` |
| `review-ledger-resolve` | Mark finding as resolved | New | `cmd/review_ledger.go` |
| `buildPriorReviewsSection()` | Build colony-prime section from ledgers | New | `cmd/colony_prime_context.go` |
| 7 agent definitions | Add Write tool + findings instructions | Modified | 4 locations x 7 agents |
| Continue wrappers | Document `report` field | Modified | `.claude/commands/ant/continue.md`, `.opencode/` |

## Sources

- `cmd/codex_continue.go` -- continue flow structs and review wave logic
- `cmd/codex_continue_finalize.go` -- external continue completion handling
- `cmd/codex_continue_plan.go` -- plan-only dispatch and external dispatch struct
- `cmd/codex_build.go:1125-1219` -- existing build outcome report pattern
- `cmd/colony_prime_context.go:123-527` -- colony-prime section assembly and priorities
- `cmd/hive.go` -- CLI subcommand pattern for structured data CRUD
- `cmd/pheromone_write.go` -- file-locking JSON write pattern with store
- `pkg/storage/storage.go:88-147` -- AtomicWrite and SaveJSON
- `pkg/llm/config.go:28-77` -- ParseAgentSpec for YAML frontmatter
- `.claude/agents/ant/aether-gatekeeper.md` -- agent definition format
- `.claude/commands/ant/continue.md` -- continue wrapper and completion packet format
- `.planning/PROJECT.md` -- v1.9 milestone specification

---
*Architecture research for: Aether v1.9 Review Persistence*
*Researched: 2026-04-26*
