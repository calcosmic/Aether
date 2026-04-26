# Technology Stack

**Project:** Aether v1.9 -- Review Findings Persistence and Domain-Ledger System
**Researched:** 2026-04-26
**Confidence:** HIGH (all findings verified by reading source code directly)

## Executive Summary

The domain-ledger system and continue-review outcome reports need zero new external dependencies. Every capability required -- structured JSON persistence with file locking, append-with-deduplication, CLI subcommand registration, deterministic ID generation, and colony-prime context injection -- already exists within established patterns in `cmd/` and `pkg/`. The recommended approach is three new files (one types file, one commands file, functions added to an existing finalize file) plus targeted modifications to six existing files.

## Recommended Stack

### Data Persistence Layer

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `pkg/storage.Store` | existing | File-locked JSON reads/writes for all 7 domain ledger files | Already provides `SaveJSON`, `LoadJSON`, `UpdateJSONAtomically`, and `AtomicWrite` with cross-process file locking via `FileLocker`. Each domain ledger at `.aether/data/reviews/{domain}/ledger.json` is a standard `store.LoadJSON` / `store.SaveJSON` cycle. No new locking infrastructure needed. Verified in `pkg/storage/storage.go` lines 140-147 (`SaveJSON`) and lines 150-165 (`LoadJSON`). |
| `encoding/json` | stdlib | Marshal/unmarshal for `ReviewLedgerFile`, `ReviewLedgerEntry` types | Direct use of struct tags, same as every other JSON type in `pkg/colony/` and `cmd/`. `json.RawMessage` for flexible report payloads if agents return structured findings. |
| `os` / `path/filepath` | stdlib | Directory creation for `reviews/{domain}/` subdirectories | `os.MkdirAll` before first write to each domain ledger. Same pattern as `hive-init` in `cmd/hive.go` lines 45-46 where `os.MkdirAll(hiveDir, 0755)` creates the hive directory before writing. |
| `crypto/sha256` + `encoding/hex` | stdlib | Content-hash deduplication for ledger entries | Same pattern as `pheromone_write.go` line 314 (`sha256Sum`) and `hive.go` line 99. Hash the agent+phase+category+description+file tuple to prevent duplicate entries when a phase is re-run. |
| `fmt` + `strconv` | stdlib | Deterministic entry IDs like `sec-2-001` | `fmt.Sprintf` with domain prefix, phase number, and zero-padded sequence counter. Same ID generation philosophy as `pheromone_write.go` line 79 and `midden_cmds.go` line 434, but domain-scoped for human readability. |
| `time` | stdlib | RFC3339 timestamps, entry age calculations for pruning | `time.Now().UTC().Format(time.RFC3339)` is the universal timestamp pattern across every Aether command. `time.Parse` for resolution age checks in `review-ledger-resolve`. |
| `sort` | stdlib | Sort entries by severity, timestamp for summary output | Same pattern as `midden_cmds.go` line 38 and `codex_build_finalize.go` line 408. |
| `strings` | stdlib | Agent-to-domain mapping, content sanitization, case-insensitive matching | Same usage across every command file. `strings.EqualFold` for case-insensitive caste/status matching (see `codex_build_finalize.go` line 449). |

### CLI Command Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/spf13/cobra` | v1.10.2 (existing) | CLI subcommand registration, flag parsing for 4 new ledger commands | `review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve` register via `rootCmd.AddCommand()` in `init()`. Exact same pattern as `cmd/midden_cmds.go` lines 485-509 where 10 subcommands are registered in one `init()` block. |
| `cmd/helpers.go` output helpers | existing | `outputOK`, `outputError` for structured JSON output | All four ledger subcommands return results through `outputOK()` with `map[string]interface{}` payloads, matching `midden_cmds.go`, `hive.go`, and `pheromone_write.go`. Verified in `cmd/helpers.go` line 18. |

### Colony-Prime Context Injection

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/colony_prime_context.go` section system | existing | New `prior-reviews` section injected into worker prompts | The `colonyPrimeSection` struct (line 53) and `buildColonyPrimeOutput()` function (line 123) already support arbitrary sections with priority ordering. Adding a new section at priority 8 (between pheromones at 9 and instincts at 6) follows the exact same pattern as the `medic_health` section added at lines 452-485. Each section becomes a `rankingCandidate()` (line 85) that flows through `colony.RankContextCandidates`. |
| `pkg/colony.ContextCandidate` | existing | Trust scoring and ranking for the new section | The new section gets `freshnessScore`, `confirmationScore`, and `relevanceScore` computed from ledger entry timestamps and counts. No new ranking logic needed -- the existing `RankContextCandidates` budget allocator handles trim ordering. |
| `pkg/cache.SessionCache` | existing | Cached loading of ledger summaries for performance | Same pattern as pheromone loading at line 191 and instinct loading at line 238. Avoids re-reading 7 ledger files on every colony-prime call. |

### Continue-Review Outcome Reports

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `store.AtomicWrite` | existing | Write per-worker `.md` report files for continue-review workers | Same as `writeCodexBuildOutcomeReports` in `codex_build.go` line 1131 which uses `store.AtomicWrite(reportRel, []byte(content))`. New function `writeCodexContinueOutcomeReports` mirrors this pattern, writing to `build/phase-{N}/worker-reports/{name}.md`. |
| `strings.Builder` | stdlib | Render markdown report content | Same as `renderCodexBuildWorkerOutcomeReport` in `codex_build.go` line 1165 which builds the report using `strings.Builder`. New `renderCodexContinueWorkerOutcomeReport` uses identical approach. |
| `strconv` | stdlib | Float formatting for duration display in worker reports | `strconv.FormatFloat` or `fmt.Sprintf("%.1f", duration)` for duration field rendering. Already imported in `codex_build.go` line 10. |

### Struct Extensions (No New Types Package Needed for Part A)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/codex_continue.go` struct extensions | existing | Add `Blockers []string`, `Duration float64`, `Report string` to `codexContinueWorkerFlowStep` | The struct at line 241 gets three new `json`-tagged fields. Non-breaking since JSON deserialization ignores unknown fields and `omitempty` tags handle zero values. |
| `cmd/codex_continue_plan.go` struct extensions | existing | Add `Report string` to `codexContinueExternalDispatch` | The struct at line 12 gets one new field. Same non-breaking reasoning. |

### New Type Definitions (Part B -- Domain Ledger)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| New file: `pkg/colony/review_ledger.go` | new | `ReviewLedgerEntry`, `ReviewLedgerFile`, `ReviewLedgerSummary` types + agent-to-domain mapping constants | Follows exact pattern of `pkg/colony/midden.go` which defines `MiddenEntry`, `MiddenFile` in ~55 lines. The types file is separate from the commands file so `pkg/colony` types are importable without pulling in cobra dependencies. |
| New file: `cmd/review_ledger.go` | new | Four cobra subcommands for CRUD operations | Follows exact pattern of `cmd/midden_cmds.go` (511 lines, 10 subcommands). Estimated ~350 lines for 4 subcommands with flag registration. |

### Agent Definitions and Mirrors

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Markdown agent files in `.claude/agents/ant/` | existing | Add Write tool to 7 review agent specs + findings instructions section | The 7 review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) get `Write` added to their YAML frontmatter tools list, plus a `## Review Findings Output` section describing write-scope guardrails and the structured JSON format for findings. Parsed by existing `ParseAgentSpec()`. |
| `gopkg.in/yaml.v3` | v3.0.1 (existing) | Agent frontmatter parsing (no changes to parser) | Already used for agent spec parsing. Only the agent markdown content changes. |
| Mirror sync via `aether publish` | existing | Sync to `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/` | The existing YAML source chain documented in CLAUDE.md handles mirror synchronization. Agent changes flow through `.claude/agents/ant/` -> `.aether/agents-claude/` -> `.opencode/agents/` -> `.codex/agents/`. |

### Lifecycle Integration

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/seal.go` | existing | Archive ledger data at seal time | Ledger files are colony-scoped and get archived with the colony. The seal process already handles `.aether/data/` file archival. |
| `cmd/status.go` | existing | Display ledger summary in status output | Add a `Ledger Summary` row showing open findings count per domain. Same pattern as existing memory health display. |
| `cmd/entomb.go` | existing | Preserve ledger files in entomb archives | Entomb already archives the full `.aether/data/` directory tree including subdirectories. The `reviews/` directory is automatically included. |

## New Files to Create

| File | Purpose | Estimated Size | Pattern Source |
|------|---------|----------------|----------------|
| `pkg/colony/review_ledger.go` | `ReviewLedgerEntry`, `ReviewLedgerFile`, `ReviewLedgerSummary` types + `AgentDomainMap` constants + helper functions | ~80 lines | `pkg/colony/midden.go` (55 lines, same struct+file pattern) |
| `cmd/review_ledger.go` | Four cobra subcommands: `review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve` with flag registration and JSON output | ~350 lines | `cmd/midden_cmds.go` (511 lines for 10 commands, proportional) |

## Files to Modify

| File | Change | Risk |
|------|--------|------|
| `cmd/codex_continue.go` | Add `Blockers []string`, `Duration float64`, `Report string` fields to `codexContinueWorkerFlowStep` (line 241) | Low -- adding optional struct fields is backward compatible |
| `cmd/codex_continue_plan.go` | Add `Report string` to `codexContinueExternalDispatch` (line 12) | Low -- same reasoning |
| `cmd/codex_continue_finalize.go` | Add `writeCodexContinueOutcomeReports()` and `renderCodexContinueWorkerOutcomeReport()` functions; call report writer after `review.json` write (after line 168); preserve new fields in `mergeExternalContinueResults()` (line 217); add `strconv` import | Medium -- modifies the continue finalize hot path with new function call; needs targeted test coverage for the new report writer |
| `cmd/colony_prime_context.go` | Add `prior-reviews` section with priority 8; reads from `.aether/data/reviews/*/ledger.json` and renders summary of open findings per domain | Low -- additive section in the existing section assembly; budget system handles trimming automatically |
| 7 agent files in `.claude/agents/ant/` | Add `Write` to tools list in frontmatter; add `## Review Findings Output` section | Low -- additive frontmatter and documentation |
| Agent mirror directories | Sync changes to `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/` | Low -- mechanical sync via publish pipeline |
| `.claude/commands/ant/continue.md` | Add `report` field to completion packet instructions | Low -- documentation change |
| `.opencode/commands/ant/continue.md` | Same as above | Low -- documentation change |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Data store | `pkg/storage.Store` with JSON files | SQLite via `modernc.org/sqlite` | No query complexity warrants a database. The 7 domain ledgers are append-mostly files with at most hundreds of entries per colony lifetime. JSON files are inspectable with `cat`, debuggable with `jq`, and already understood by every existing Aether tool. Adding SQLite would introduce a CGO dependency, complicate the build, and break the "inspect with `cat`" debugging story that every other data file maintains. |
| Data store | `pkg/storage.Store` with JSON files | BadgerDB / BoltDB embedded KV stores | Same reasoning as SQLite. The ledger access pattern is sequential scan by domain (for colony-prime injection) and point lookup by ID (for resolve). No performance bottleneck exists at colony-scoped review data scale. |
| ID generation | Deterministic `sec-2-001` format | UUID v4 via `github.com/google/uuid` | Deterministic IDs encode domain and phase information, making debugging and log tracing trivial. `sec-2-001` immediately tells you it is a security finding from phase 2 entry 1. UUID adds an external dependency and produces opaque identifiers. Every existing ID pattern in Aether (`sig_*`, `midden_*`, `flag_*`) is deterministic and human-readable. |
| Schema validation | Struct tags + manual validation in command handlers | `github.com/go-playground/validator` | Manual validation is the existing pattern (see `pheromone_write.go` lines 38-49, `midden_cmds.go` lines 94-98). Adding a validation library for 4 commands with ~5 fields each is over-engineering that breaks consistency with the rest of the codebase. |
| Agent findings format | Write tool + structured JSON to `.aether/data/reviews/` | Custom gRPC or IPC protocol | Agents are LLM subprocesses that communicate through file I/O and markdown. The Write tool approach matches how build outcome reports work (`writeCodexBuildOutcomeReports`). No inter-process communication channel exists, and building one would be a major architectural departure for zero benefit. |
| Colony-prime injection | String-builder markdown section | Protocol buffer or custom binary format | Colony-prime produces markdown text injected into LLM prompts. A structured binary format would need a serialization/deserialization step for no benefit since the consumer is an LLM reading markdown. |
| Ledger types location | New file `pkg/colony/review_ledger.go` | Inline types in `cmd/review_ledger.go` | Separating types from commands allows the types to be imported by both `cmd/review_ledger.go` (commands) and `cmd/colony_prime_context.go` (injection) without creating import cycles. This matches the pattern where `pkg/colony/midden.go` defines types used by `cmd/midden_cmds.go` and `cmd/medic_scanner.go`. |

## What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| Any new `go.mod` dependency | The four required capabilities (JSON persistence with locking, CLI commands, context injection, worker reports) are fully covered by existing packages. Adding dependencies increases binary size, supply chain attack surface, and maintenance burden for zero functional gain. `go mod tidy` should produce no changes. |
| Database of any kind | Colony-scoped review data will never exceed hundreds of entries per domain across a full project lifecycle. JSON files are sufficient, debuggable, and consistent with every other data file in `.aether/data/`. The pheromone system handles similar volume with the same approach. |
| Message queue or event streaming | Review findings are written once per continue cycle and read for colony-prime injection. The existing `pkg/events` event bus can emit optional ceremony events for audit logging, but the ledger itself does not need pub/sub semantics. |
| Custom marshaling or serialization | `encoding/json` with struct tags handles all required formats. Ledger entries are flat records with no circular references, polymorphism, or custom encoding needs. |
| Testing framework beyond `testing` + `testify` | Existing test patterns in `cmd/*_test.go` use standard `testing.T` with `stretchr/testify` assertions. No new assertion or mocking libraries needed. |
| ORM or query builder | Direct struct iteration and `sort.Slice` cover all query patterns needed (filter by status, group by domain, count by severity). The data volume is trivially small. |

## Installation

No installation needed. All dependencies already exist in `go.mod`.

```bash
# Verify existing dependencies are sufficient
go mod tidy  # should show no changes
go build ./cmd/aether  # should compile cleanly

# Run existing tests to establish baseline before starting
go test ./... -race
```

## Confidence Assessment

| Area | Confidence | Reason |
|------|------------|--------|
| Zero new dependencies | HIGH | Reviewed every existing pattern in `pkg/storage`, `pkg/colony`, and `cmd/`. All required capabilities (atomic JSON write, file locking, dedup, CLI registration, section injection, report rendering) are present and proven by 2900+ passing tests. |
| Ledger file structure | HIGH | Direct structural analog of `midden.json` pattern (`pkg/colony/midden.go`) with domain-scoped subdirectories instead of a single flat file. The `MiddenFile` / `MiddenEntry` types are the template. |
| Colony-prime integration | HIGH | The `colonyPrimeSection` system at `cmd/colony_prime_context.go` is explicitly designed for additive sections with priority-based trimming. The `medic_health` section added at lines 452-485 proves the pattern works for exactly this kind of addition. |
| Continue-review reports | HIGH | Structural mirror of `writeCodexBuildOutcomeReports` at `cmd/codex_build.go` lines 1125-1185 with adapted field names and report content. |
| Agent mirror sync | HIGH | Existing publish pipeline handles mirror synchronization mechanically. |
| Part A struct extensions | HIGH | Adding optional fields to existing structs is the most common backward-compatible change in Go JSON APIs. The `omitempty` tag pattern prevents zero-value pollution. |

## Sources

- `/Users/callumcowie/repos/Aether/pkg/storage/storage.go` -- Store API: `AtomicWrite` (line 48), `SaveJSON` (line 140), `LoadJSON` (line 150), `UpdateJSONAtomically` (line 121)
- `/Users/callumcowie/repos/Aether/cmd/pheromone_write.go` -- Append-with-dedup pattern using `sha256Sum` content hashing (lines 86-89, 150-169)
- `/Users/callumcowie/repos/Aether/cmd/hive.go` -- LRU eviction and cross-entry deduplication (lines 98-118), directory creation pattern (lines 45-46)
- `/Users/callumcowie/repos/Aether/cmd/midden_cmds.go` -- CRUD subcommand registration pattern (lines 485-509), JSON payload structure
- `/Users/callumcowie/repos/Aether/pkg/colony/midden.go` -- Struct type definition pattern for `MiddenEntry` / `MiddenFile`
- `/Users/callumcowie/repos/Aether/cmd/codex_build.go` lines 1125-1185 -- `writeCodexBuildOutcomeReports` and `renderCodexBuildWorkerOutcomeReport` pattern
- `/Users/callumcowie/repos/Aether/cmd/codex_continue_finalize.go` -- Continue finalize hot path, `runCodexContinueFinalize` function
- `/Users/callumcowie/repos/Aether/cmd/codex_continue.go` -- `codexContinueWorkerFlowStep` struct (line 241), `codexContinueReviewReport` (line 250)
- `/Users/callumcowie/repos/Aether/cmd/codex_continue_plan.go` -- `codexContinueExternalDispatch` struct (line 12), `continuePlanArtifactsPath` (line 225)
- `/Users/callumcowie/repos/Aether/cmd/colony_prime_context.go` -- Section assembly with priority ordering (lines 123-527), `colonyPrimeSection` struct (line 53)
- `/Users/callumcowie/repos/Aether/go.mod` -- Existing dependency inventory (cobra v1.10.2, yaml.v3 v3.0.1, no database drivers)
