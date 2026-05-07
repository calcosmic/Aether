# Architecture Research: v1.15 Framework Coherence Audit

**Domain:** Systematic audit and hardening of an existing Go/Cobra multi-agent CLI framework (80+ subcommands, 3 platform surfaces, 7 data subsystems)
**Researched:** 2026-05-07
**Overall confidence:** HIGH (direct source code analysis of cmd/, pkg/, .aether/commands/*.yaml, wrapper contracts, existing test infrastructure)

## Executive Summary

Aether has 317 registered Cobra subcommands spread across ~100 source files in cmd/, with 60 YAML source-of-truth command definitions, dual platform wrapper surfaces (Claude + OpenCode), and a Codex runtime-native lane. The audit must verify coherence across four integration surfaces: (1) Go runtime command contracts, (2) YAML-to-wrapper generation chain, (3) data flow through the full init-to-seal lifecycle, and (4) regression test coverage that catches drift.

The architecture for the audit itself is a layered scanner pattern. Layer 1 inventories what exists (command catalog, data artifact catalog, wrapper catalog). Layer 2 cross-references surfaces for parity violations. Layer 3 traces data flow through the lifecycle to find dead-end artifacts and unconsumed outputs. Layer 4 writes regression tests that freeze the verified contracts so future drift fails loudly.

The key insight is that Aether already has partial audit infrastructure: `source-check` verifies YAML-to-wrapper header parity, `command_parity_test.go` verifies Claude/OpenCode body parity, `command_source_hygiene_test.go` verifies wrapper-to-YAML source references, and `visual_wrapper_contract_test.go` verifies ceremony surface presence. The audit extends these into a complete coherence checker rather than building from scratch.

**Critical constraint:** The audit must not change runtime behavior. It reads and cross-references. Fixes are separate phases. The audit architecture produces findings; subsequent phases act on them.

## System Overview: The Audit Scanner Architecture

```
+=====================================================================+
|                     AUDIT SCANNER (new code)                         |
|                                                                      |
|  +------------------+  +------------------+  +------------------+    |
|  | Command Catalog  |  | Data Artifact    |  | Wrapper Parity   |    |
|  | Scanner          |  | Scanner          |  | Scanner          |    |
|  +--------+---------+  +--------+---------+  +--------+---------+    |
|           |                       |                       |           |
+-----------+-----------+-----------+-----------+-----------+-----------+
            |                       |                       |
            v                       v                       v
+===================+  +===================+  +===================+
| cmd/*.go          |  | .aether/data/     |  | .claude/commands/ |
| (317 subcommands) |  | (7 data subsystems|  | .opencode/        |
|                   |  |  + artifacts)     |  | (wrapper parity)  |
+===================+  +===================+  +===================+
            |                       |                       |
            v                       v                       v
+=====================================================================+
|                  AUDIT FINDINGS (structured output)                  |
|                                                                      |
|  +------------------+  +------------------+  +------------------+    |
|  | Command Contract |  | Lifecycle Data   |  | Parity Drift     |    |
|  | Violations       |  | Dead Ends        |  | Report           |    |
|  +------------------+  +------------------+  +------------------+    |
|                                                                      |
+=====================================================================+
            |
            v
+=====================================================================+
|              REGRESSION TESTS (freeze verified contracts)            |
|                                                                      |
|  TestCommandCatalogComplete      - no orphan commands                |
|  TestDataFlowLifecycleConnected  - no dead-end artifacts             |
|  TestWrapperParityAtScale        - all surfaces agree                |
|  TestCommandGuideYAMLAlignment   - guide catalog matches YAML        |
+=====================================================================+
```

## Integration Points That Need Verification

### 1. Go Runtime Command Surface (cmd/*.go)

**Scale:** 317 `rootCmd.AddCommand()` calls across ~100 files.

The audit must verify:

| What | Where | Contract |
|------|-------|----------|
| Every command has a `Use` field | `&cobra.Command{Use: ...}` | Must match `aether <name>` pattern |
| Every command has a `Short` description | `Short: "..."` | Non-empty, human-readable |
| Output mode consistency | `outputOK()`, `outputError()`, `outputWorkflow()` | Commands that produce JSON must use `outputOK`; visual commands must use `outputWorkflow` |
| Store initialization | `store *storage.Store` | Commands that read/write data must not have `nil` store; `skipStoreInit()` must list all storeless commands |
| Flag registration | `cmd.Flags().String(...)` | Flags referenced in `RunE` must be registered in `init()` |

**New component:** `CommandCatalogScanner` -- walks all registered Cobra commands via `rootCmd.Commands()` and produces a structured catalog.

### 2. YAML Source-of-Truth Chain

**Scale:** 60 YAML files in `.aether/commands/`.

```
.aether/commands/*.yaml          <- Source definitions (name, runtime, codex_orchestration, wrapper_contract, guardrails)
    |
    v  (generation)
.claude/commands/ant/*.md        <- Claude Code wrappers (must have generated-from header)
.opencode/commands/ant/*.md      <- OpenCode wrappers (must have generated-from header)
    |
    v  (delegation)
cmd/codex_workflow_cmds.go       <- Go runtime (authoritative state mutations)
cmd/command_guide.go             <- Codex orchestration guide catalog
```

**Verification points:**

| Check | Existing Coverage | Gap |
|-------|-------------------|-----|
| Every YAML has matching Claude wrapper | `source-check` | Partial -- checks header, not body contract |
| Every YAML has matching OpenCode wrapper | `source-check` | Same as above |
| Claude/OpenCode wrapper bodies are identical | `command_parity_test.go` | Covered |
| Wrapper generated-from header matches YAML | `source_source_hygiene_test.go` | Covered |
| YAML `codex_orchestration` matches `commandGuideCatalog()` | None | GAP -- no test verifies the command_guide catalog includes all YAML commands |
| YAML `wrapper_contract` fields match actual runtime subcommands | None | GAP -- no test verifies manifest/finalizer fields reference real commands |
| YAML `guardrails` are present and non-empty | None | GAP |
| Lifecycle wrappers follow finalizer-then-closeout ordering | `visual_wrapper_contract_test.go` | Covered |

**New component:** `YAMLGuideAlignmentScanner` -- reads all YAML `codex_orchestration` entries, reads `commandGuideCatalog()`, reports mismatches.

### 3. Data Flow Through Full Lifecycle (init -> seal)

The lifecycle creates, reads, and mutates data artifacts across 7 subsystems:

```
.aaether/data/
+-- COLONY_STATE.json          <- Central colony state (init creates, all mutate)
+-- pheromones.json            <- Signal system (write/decay/display/display-prime)
+-- midden/
|   +-- midden.json            <- Failure tracking (write/recent/review/acknowledge)
+-- constraints.json           <- Legacy constraints (write/read)
+-- pending-decisions.json     <- Discuss system (add/list/resolve)
+-- assumptions.json           <- Assumptions (write/read)
+-- session.json               <- Session state (init/read/update/clear/verify)
+-- behavior-observations.jsonl <- Profile observations (observe/read)
+-- survey/                    <- Territory survey (load/verify)
+-- handoffs/
|   +-- worker-handoffs.json   <- Worker handoffs (build writes, continue reads)
+-- spawn-runs/                <- Spawn tracking (log/complete)
+-- review-ledgers/            <- Review persistence (write/read/summary/resolve)
+-- gates/                     <- Gate results (write/read)
+-- flags/                     <- Colony flags (add/resolve/check)
+-- eventbus/                  <- Event bus (publish/query/replay/cleanup)
+-- instincts/                 <- Instincts (create/read/trusted/decay/archive)
+-- changelog/
|   +-- CHANGELOG.md           <- Changelog (append)
+-- chamber/                   <- Chambers (create/verify/list/compare)
+-- views/                     <- View state (init/get/set/toggle)
+-- traces/                    <- Traces (replay/export/summary/inspect)
+-- grave/                     <- Grave patterns (add/check)
+-- error-patterns/            <- Error patterns (add/flag/summary/check)
+-- graph/                     <- Knowledge graph (link/neighbors/reach/cluster)
+-- seal/
|   +-- final-review.json      <- Seal review (seal writes)
+-- codex/
|   +-- build/                 <- Build artifacts (manifest, worker files)
|   +-- continue/              <- Continue artifacts
+-- oracle/                    <- Oracle workspace
+-- hive/                      <- Hive search index (if SQLite enabled)
+-- backups/                   <- Backup rotation
```

**Data flow trace (init to seal):**

```
INIT
  |-- writes COLONY_STATE.json (goal, phase=0, state=INITIALIZED)
  |-- writes session.json (session_id, colony_goal)
  |
COLONIZE
  |-- reads COLONY_STATE.json (goal, scope)
  |-- writes survey/* files
  |-- writes COLONY_STATE.json (territory_surveyed)
  |
PLAN
  |-- reads COLONY_STATE.json (goal, scope, depth)
  |-- writes COLONY_STATE.json (plan with phases)
  |
BUILD N
  |-- reads COLONY_STATE.json (current_phase, plan, depth)
  |-- writes spawn-runs/* (spawn tracking)
  |-- writes handoffs/worker-handoffs.json
  |-- writes codex/build/* (manifest, worker files)
  |-- writes COLONY_STATE.json (StateBUILT)
  |
CONTINUE N
  |-- reads COLONY_STATE.json (current_phase, plan, gate_results)
  |-- reads handoffs/worker-handoffs.json
  |-- reads codex/build/* (verification)
  |-- writes gates/* (gate results)
  |-- writes review-ledgers/* (review findings)
  |-- runs verification (tests, lint, claims)
  |-- runs gate checks (11 gates)
  |-- writes pheromones.json (decay + signal housekeeping)
  |-- writes midden/* (failure tracking)
  |-- writes instincts/* (promoted learnings)
  |-- writes COLONY_STATE.json (phase advanced or blocked)
  |
SEAL
  |-- reads COLONY_STATE.json (all phases complete)
  |-- writes seal/final-review.json
  |-- writes review-ledgers/* (final review findings)
  |-- writes QUEEN.md (lessons, wisdom)
  |-- promotes instincts to hive (hive-promote)
  |-- writes COLONY_STATE.json (state=SEALED, milestone=CROWNED_ANTHILL)
```

**Dead-end detection:** For each artifact, the audit must answer: who writes it, who reads it, is it consumed downstream, or is it write-only decoration?

**New component:** `LifecycleDataFlowScanner` -- traces write/read relationships across all data files.

### 4. Platform Parity (3 Surfaces)

| Surface | Location | Format | Parity Check |
|---------|----------|--------|-------------|
| Claude Code | `.claude/commands/ant/*.md` | Markdown (generated) | Body must match OpenCode |
| OpenCode | `.opencode/commands/ant/*.md` | Markdown (generated) | Body must match Claude |
| Codex | `command_guide.go` + Codex skills | Go struct catalog + TOML agents | Must cover same commands |

**New component:** `ThreeSurfaceParityScanner` -- verifies that every YAML command appears in all three surfaces with consistent contracts.

### 5. Ceremony Surface Integrity

The ceremony system (`cmd/ceremony_cmd.go`) provides 4 visual surfaces:

| Ceremony | Purpose | Used By |
|----------|---------|---------|
| `spawn-plan` | Render worker spawn plan | build, plan, colonize, seal, swarm |
| `wave-start` | Render wave banner | build, plan, colonize, seal, swarm |
| `worker-complete` | Render worker completion | build, plan, colonize, seal, swarm |
| `closeout` | Render lifecycle summary | build, plan, colonize, continue, seal, swarm |

Every lifecycle wrapper (6 workflows x 2 platforms = 12 wrappers) must call all four ceremony surfaces in the correct order.

**Existing coverage:** `visual_wrapper_contract_test.go` verifies presence and ordering.

### 6. Gate System Integrity

The gate system (`cmd/gate.go`) has 11 gates with classification:

| Classification | Behavior | Auto-resolve? |
|----------------|----------|---------------|
| `hard_block` | Stops advancement | Never |
| `soft_block` | Warns, can be overridden | Only if queen approves |
| `advisory` | Informational | Always safe |

**Audit check:** Every gate must have an explicit classification. No gate should be unclassified.

### 7. Worker Economy

27 agent definitions across 3 platforms:

| Platform | Location | Format | Count |
|----------|----------|--------|-------|
| Claude | `.claude/agents/ant/*.md` | Markdown | 25 |
| OpenCode | `.opencode/agents/*.md` | Markdown | varies |
| Codex | `.codex/agents/*.toml` | TOML | 25 |

**Audit check:** Every agent must have a caste assignment, a clear write output (not read-only), and must be referenced by at least one command.

## Recommended Audit Architecture (New Components)

### Component Inventory

All new code lives in cmd/ as test files and potentially a new `audit_catalog.go` for runtime catalog generation.

| Component | Type | Purpose | Depends On |
|-----------|------|---------|------------|
| `audit-catalog` | Cobra command (new) | Produce a structured JSON catalog of all registered commands, their flags, output mode, and data file interactions | `rootCmd` tree walk |
| `CommandCatalogScanner` | Test helper | Walk Cobra tree, produce map of command name -> metadata | `audit-catalog` or direct `rootCmd` inspection |
| `YAMLGuideAlignmentTest` | Test file | Verify `commandGuideCatalog()` covers all YAML commands and vice versa | `commandGuideCatalog()`, YAML directory |
| `DataFlowLifecycleTest` | Test file | Verify every data artifact has at least one writer and one reader across the lifecycle | Source analysis of cmd/*.go |
| `ThreeSurfaceParityTest` | Test file | Extend existing parity test to include command-guide coverage | Existing `command_parity_test.go` |
| `CommandContractRegressionTest` | Test file | Freeze the verified command catalog as a snapshot; future drift fails the test | Catalog snapshot |

### Data Structures

```go
// CommandCatalogEntry describes a single registered Cobra command.
type CommandCatalogEntry struct {
    Name         string   `json:"name"`
    HasShort     bool     `json:"has_short"`
    HasRunE      bool     `json:"has_run_e"`
    Flags        []string `json:"flags"`
    OutputMode   string   `json:"output_mode"`   // "json", "visual", "workflow", "unknown"
    NeedsStore   bool     `json:"needs_store"`
    Category     string   `json:"category"`       // "literal", "full-orchestration", "semi-intelligent"
    YAMLSource   string   `json:"yaml_source"`     // matching .aether/commands/<name>.yaml
    InGuide      bool     `json:"in_guide"`        // appears in commandGuideCatalog()
    DataReads    []string `json:"data_reads"`       // data files this command reads
    DataWrites   []string `json:"data_writes"`      // data files this command writes
}

// DataFlowEdge describes a producer-consumer relationship.
type DataFlowEdge struct {
    Artifact string `json:"artifact"` // e.g., "COLONY_STATE.json", "pheromones.json"
    Writers  []string `json:"writers"`  // commands that write this file
    Readers  []string `json:"readers"`  // commands that read this file
    Orphan   bool     `json:"orphan"`   // true if write-only (no reader) or read-only (no writer)
}

// ParityViolation describes a mismatch across surfaces.
type ParityViolation struct {
    Type       string `json:"type"`        // "missing_yaml", "missing_wrapper", "body_drift", "guide_mismatch"
    Command    string `json:"command"`
    Surface    string `json:"surface"`     // "claude", "opencode", "codex_guide", "yaml"
    Expected   string `json:"expected"`
    Actual     string `json:"actual"`
}
```

## Build Order (Phase Dependencies)

The audit phases must be ordered so each builds on verified results from the previous.

```
Phase 1: Command Inventory (no dependencies)
  |-- Build CommandCatalogScanner
  |-- Walk rootCmd tree, produce full catalog
  |-- Output: cmd_audit_catalog.json
  |-- Why first: every subsequent phase needs to know what commands exist
  |
Phase 2: YAML/Wrapper Parity (depends on Phase 1 catalog)
  |-- Extend existing source-check with catalog-aware validation
  |-- Verify YAML-to-wrapper generation chain completeness
  |-- Verify command-guide catalog matches YAML
  |-- Output: parity_violations.json
  |-- Why second: needs the command catalog to cross-reference
  |
Phase 3: Data Flow Tracing (depends on Phase 1 catalog)
  |-- For each command in catalog, trace data reads/writes
  |-- Build DataFlowEdge map for all artifacts
  |-- Identify dead-end artifacts (write-only, read-only, orphan)
  |-- Output: data_flow_map.json
  |-- Why third: needs command catalog to know which commands to trace
  |
Phase 4: Lifecycle Walkthrough (depends on Phase 3 data flow)
  |-- Walk the init->colonize->plan->build->continue->seal lifecycle
  |-- Verify each transition produces artifacts that the next transition consumes
  |-- Identify lifecycle gaps (artifacts produced but never consumed downstream)
  |-- Output: lifecycle_gaps.json
  |-- Why fourth: needs data flow map to understand the full chain
  |
Phase 5: Regression Test Freezing (depends on Phases 1-4 findings)
  |-- Write regression tests that snapshot the verified catalog
  |-- Write regression tests that snapshot data flow edges
  |-- Write regression tests that snapshot parity state
  |-- Output: new test files in cmd/
  |-- Why last: needs verified findings to know what to freeze
```

**Dependency graph:**

```
Phase 1 (Inventory)
    |
    +---> Phase 2 (Parity)
    |
    +---> Phase 3 (Data Flow)
              |
              v
          Phase 4 (Lifecycle)
              |
              v
          Phase 5 (Regression)
```

Phases 2 and 3 can run in parallel. Phase 4 requires Phase 3. Phase 5 requires all prior phases.

## Patterns to Follow

### Pattern 1: Catalog-First Auditing

**What:** Build a complete inventory before checking any individual item.
**When:** Always. You cannot verify coherence without knowing what exists.
**Example:**

```go
func buildCommandCatalog(t *testing.T) map[string]CommandCatalogEntry {
    catalog := make(map[string]CommandCatalogEntry)
    for _, cmd := range rootCmd.Commands() {
        entry := CommandCatalogEntry{
            Name:     cmd.Name(),
            HasShort: cmd.Short != "",
            HasRunE:  cmd.RunE != nil,
        }
        // ... populate flags, output mode, etc.
        catalog[entry.Name] = entry
    }
    return catalog
}
```

### Pattern 2: Snapshot Regression

**What:** After verifying a surface is correct, freeze it as a test assertion so future drift fails CI.
**When:** At the end of the audit, not during.
**Example:**

```go
func TestCommandCatalogSnapshot(t *testing.T) {
    catalog := buildCommandCatalog(t)
    // Verify no commands were removed
    knownCount := 317 // snapshot from audit
    if len(catalog) < knownCount {
        t.Fatalf("command count decreased: got %d, expected at least %d", len(catalog), knownCount)
    }
}
```

### Pattern 3: Cross-Reference Verification

**What:** For every item in surface A, verify it exists in surface B.
**When:** Parity checks across YAML, wrappers, command-guide, and runtime.
**Example:**

```go
func TestYAMLHasGuideEntry(t *testing.T) {
    yamlNames := loadYAMLCommandNames(t)
    guideCatalog := commandGuideCatalog()
    literalSet := make(map[string]bool)
    for _, name := range commandGuideLiteralCommands() {
        literalSet[name] = true
    }
    for name := range yamlNames {
        _, inGuide := guideCatalog[name]
        if !inGuide && !literalSet[name] {
            t.Errorf("YAML command %q has no command-guide entry", name)
        }
    }
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Audit That Modifies Behavior

**What people do:** Write audit code that fixes issues as it finds them.
**Why it is wrong:** Makes it impossible to distinguish findings from fixes. Breaks reproducibility.
**Do this instead:** Audit produces findings. Fixes are separate phases with separate tests.

### Anti-Pattern 2: Brittle Snapshot Tests

**What people do:** Snapshot entire command bodies or YAML contents as strings.
**Why it is wrong:** Any minor formatting change breaks the test, causing alert fatigue.
**Do this instead:** Snapshot structural properties (command count, flag count, required ceremony surfaces, data flow edges). Use `>=` for counts (growth is fine), exact match for parity.

### Anti-Pattern 3: Testing Wrappers Instead of Contracts

**What people do:** Assert exact text in wrapper markdown files.
**Why it is wrong:** Wrappers are generated and will change with generator improvements.
**Do this instead:** Assert structural contracts: generated-from header exists, ceremony calls are present, finalizer/closeout ordering is correct, hand-editing guardrails are present.

### Anti-Pattern 4: Ignoring Store Initialization

**What people do:** Assume all commands that access data properly check `store != nil`.
**Why it is wrong:** A command that runs without a store will panic or silently fail.
**Do this instead:** Verify that every command with data reads/writes either checks store or is listed in `skipStoreInit()`.

## Existing Audit Infrastructure (Extend, Do Not Replace)

| Existing Tool | What It Checks | Extension Needed |
|---------------|---------------|------------------|
| `aether source-check` | YAML-to-wrapper headers, retired mirrors, canonical surfaces | Add command-guide alignment, wrapper contract field validation |
| `command_parity_test.go` | Claude/OpenCode wrapper body parity | Add Codex command-guide parity |
| `command_source_hygiene_test.go` | Wrapper generated-from headers, YAML source references | Add YAML field completeness (codex_orchestration, wrapper_contract) |
| `visual_wrapper_contract_test.go` | Ceremony surface presence in lifecycle wrappers | Add ceremony parameter validation |
| `platform_doc_hygiene_test.go` | Runtime CLI references in lifecycle docs | Extend to cover new commands |
| `emoji_audit_test.go` | Emoji map coverage for visual commands | Add to catalog completeness check |

## Integration Points Summary

| Integration Point | Left Side | Right Side | Contract | Existing Test |
|-------------------|-----------|------------|----------|--------------|
| YAML -> Claude wrapper | `.aether/commands/*.yaml` | `.claude/commands/ant/*.md` | generated-from header, body content | `source-check`, `command_source_hygiene_test` |
| YAML -> OpenCode wrapper | `.aether/commands/*.yaml` | `.opencode/commands/ant/*.md` | generated-from header, body content | Same as above |
| Claude <-> OpenCode parity | `.claude/commands/ant/*.md` | `.opencode/commands/ant/*.md` | Identical body | `command_parity_test` |
| YAML -> command-guide | `.aether/commands/*.yaml` | `commandGuideCatalog()` | Every YAML has a guide entry | GAP |
| YAML -> Go runtime | `.aether/commands/*.yaml` | `cmd/codex_workflow_cmds.go` | wrapper_contract fields match real commands | GAP |
| Lifecycle wrapper -> ceremony | `.claude/commands/ant/{build,plan,...}.md` | `ceremony_cmd.go` | All 4 ceremony surfaces called in order | `visual_wrapper_contract_test` |
| Build -> Continue data flow | `codex/build/*` artifacts | `codex_continue.go` reads | Build artifacts consumed by continue | GAP |
| Continue -> Seal data flow | `gates/*`, `review-ledgers/*` | `seal_final_review.go` | Gate results and ledgers consumed by seal | GAP |
| Seal -> Hive promotion | `instincts/*` | `hive.go` | High-confidence instincts promoted | GAP |
| Colony-prime -> Worker context | `COLONY_STATE.json`, `pheromones.json`, `QUEEN.md` | `colony_prime_context.go` | All injected into worker prompt | `colony_prime_test` |

## Sources

- Direct source analysis: `cmd/root.go`, `cmd/codex_workflow_cmds.go`, `cmd/command_guide.go`, `cmd/source_check.go`, `cmd/ceremony_cmd.go`, `cmd/gate.go`
- Test infrastructure: `cmd/command_parity_test.go`, `cmd/command_source_hygiene_test.go`, `cmd/visual_wrapper_contract_test.go`, `cmd/codex_e2e_test.go`
- Data structures: `pkg/colony/colony.go` (ColonyState), `pkg/storage/storage.go` (Store)
- Lifecycle flow: `cmd/codex_continue.go`, `cmd/seal_final_review.go`, `cmd/entomb_cmd.go`
- YAML definitions: `.aether/commands/*.yaml` (60 files)
- Project context: `.planning/PROJECT.md`

---
*Architecture research for: Aether v1.15 Framework Coherence Audit*
*Researched: 2026-05-07*
