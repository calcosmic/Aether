# Project Research Summary

**Project:** Aether v1.15 -- Framework Coherence, Efficiency, and Ship Readiness
**Domain:** Internal audit and hardening of a Go/Cobra multi-agent CLI framework (80+ subcommands, 3 platform surfaces, 7 data subsystems)
**Researched:** 2026-05-07
**Confidence:** HIGH

## Executive Summary

Aether v1.15 is a systematic coherence audit of an 80+ subcommand Go CLI framework with three platform surfaces (Claude Code, OpenCode, Codex runtime-native), 60 YAML source-of-truth command definitions, and 7 data subsystems spanning the init-to-seal lifecycle. The audit must verify that every command has a contract, every spawned worker has justified purpose, every data artifact is consumed or pruned, and every platform surface agrees on what exists. This is not a feature milestone; it is an integrity milestone that locks in the accumulated work of v1.0 through v1.14 before the system grows further.

The recommended approach is a layered scanner pattern. Layer 1 inventories what exists (317 Cobra commands, 60 YAML definitions, 12 lifecycle wrappers, 27 agent definitions). Layer 2 cross-references surfaces for parity violations (YAML vs wrappers vs command-guide vs runtime). Layer 3 traces data flow through the full lifecycle to find dead-end artifacts. Layer 4 writes regression tests that freeze the verified contracts so future drift fails CI. Aether already has partial audit infrastructure (`source-check`, `command_parity_test.go`, `visual_wrapper_contract_test.go`) that should be extended rather than replaced.

The key risks are (1) the audit surface is large -- 317 commands across ~100 source files means manual verification is infeasible, requiring automated cataloging first; (2) the existing parity tests cover Claude/OpenCode body agreement but leave YAML-to-command-guide and YAML-to-wrapper-contract gaps unverified; and (3) regression tests must be structurally sound (checking properties, not string snapshots) to avoid alert fatigue from cosmetic changes.

## Key Findings

### Recommended Stack

The milestone requires zero new dependencies. The audit uses Go's testing package, Cobra's command tree introspection, and existing test infrastructure in `cmd/`. All findings come from direct source code analysis.

**Core technologies:**
- **Go 1.24 + Cobra v1.10.2:** Command registration and tree walking via `rootCmd.Commands()` -- provides the complete command surface for cataloging
- **Existing test infrastructure:** `source-check`, `command_parity_test.go`, `command_source_hygiene_test.go`, `visual_wrapper_contract_test.go` -- extend, do not replace
- **YAML source-of-truth chain:** 60 `.aether/commands/*.yaml` files define the canonical command set; wrappers and command-guide are derived surfaces
- **pkg/storage.Store:** All state mutations use `UpdateJSONAtomically`; the audit verifies store initialization for every data-touching command

### Expected Features

**Must have (table stakes for audit integrity):**
- **Command catalog scanner** -- walks all 317 registered Cobra commands, produces structured JSON catalog with metadata (flags, output mode, store requirement, YAML source match)
- **Three-surface parity verification** -- YAML, Claude wrappers, OpenCode wrappers, and Codex command-guide all agree on what commands exist and their contracts
- **Data flow lifecycle trace** -- every artifact in `.aether/data/` has at least one writer and one reader; dead-end (write-only) artifacts are flagged
- **Regression test suite** -- snapshot-based tests that catch future command removal, parity drift, and data flow disconnection

**Should have (efficiency and polish):**
- **Worker economy audit** -- every agent definition has a caste assignment, a clear output, and is referenced by at least one command
- **Gate system integrity check** -- all 11 gates have explicit classification (hard_block/soft_block/advisory) with no unclassified gates
- **Visual ceremony audit** -- all 12 lifecycle wrappers (6 workflows x 2 platforms) call all four ceremony surfaces in correct order
- **Command contract completeness** -- every command has non-empty `Short`, registered flags match `RunE` usage, output mode is explicit

**Defer (post-audit):**
- Performance benchmarking of the scanner itself
- Automated fix generation (audit finds, humans fix)
- Cross-milestone drift monitoring (future CI integration)

### Architecture Approach

The audit follows a catalog-first scanner pattern: build a complete inventory, then cross-reference, then trace flows, then freeze verified state. The critical constraint is that the audit must not change runtime behavior -- it reads and cross-references only. Fixes are separate phases.

**Major components:**
1. **CommandCatalogScanner** -- walks the Cobra command tree, produces `map[string]CommandCatalogEntry` with name, flags, output mode, store requirement, YAML source, guide presence, data reads/writes
2. **YAMLGuideAlignmentScanner** -- reads all YAML `codex_orchestration` entries, reads `commandGuideCatalog()`, reports mismatches (currently untested gap)
3. **LifecycleDataFlowScanner** -- traces write/read relationships for all `.aether/data/` artifacts across the init->colonize->plan->build->continue->seal lifecycle
4. **ThreeSurfaceParityScanner** -- extends existing Claude/OpenCode parity to include Codex command-guide coverage
5. **Regression test suite** -- structural snapshot tests (command count >= N, parity exact match, data flow edges present)

### Critical Pitfalls

1. **Audit that modifies behavior** -- The audit must be read-only. Findings and fixes are separate phases with separate tests. Mixing them destroys reproducibility.

2. **Brittle snapshot tests** -- Snapshot entire command bodies or YAML strings and any formatting change breaks CI. Instead, snapshot structural properties (command count, flag count, ceremony presence). Use `>=` for counts, exact match for parity.

3. **Testing wrappers instead of contracts** -- Asserting exact text in generated markdown wrappers is fragile. Assert structural contracts: generated-from header exists, ceremony calls present, finalizer/closeout ordering correct.

4. **Missing store initialization checks** -- Commands that access data without a store will panic. Verify every data-touching command either checks store or is listed in `skipStoreInit()`.

5. **Existing parity test gaps** -- `command_parity_test.go` covers Claude/OpenCode body agreement but leaves three gaps: (a) YAML `codex_orchestration` vs `commandGuideCatalog()` alignment, (b) YAML `wrapper_contract` fields vs actual runtime subcommands, (c) YAML `guardrails` presence. These gaps are where drift silently accumulates.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Command Inventory and Catalog
**Rationale:** Every subsequent phase needs to know what commands exist. This is the foundation for all cross-referencing.
**Delivers:** Structured JSON catalog of all 317 registered Cobra commands with metadata; `audit-catalog` runtime command.
**Addresses:** Lifecycle coherence -- every command has a contract and durable output.
**Avoids:** Audit-that-modifies-behavior pitfall; catalog is read-only.

### Phase 2: YAML-to-Wrapper Parity Verification
**Rationale:** With the catalog complete, cross-reference the 60 YAML source definitions against generated wrappers and command-guide. Extends existing parity tests to cover known gaps.
**Delivers:** Complete three-surface parity report (YAML, Claude, OpenCode, Codex); regression tests that freeze alignment.
**Addresses:** Platform parity -- Go runtime, YAML, Claude, OpenCode, Codex all agree.
**Avoids:** Testing-wrappers-instead-of-contracts pitfall; check structural properties, not exact text.
**Uses:** Command catalog from Phase 1; extends `source-check`, `command_parity_test.go`.

### Phase 3: Data Flow Lifecycle Trace
**Rationale:** With command catalog available, trace which commands read and write which data artifacts. Identify dead-end artifacts that are written but never consumed.
**Delivers:** Data flow map showing producer-consumer relationships for all `.aether/data/` artifacts; dead-end report.
**Addresses:** Data wiring -- every artifact consumed or pruned.
**Avoids:** Missing store initialization pitfall; every data-touching command must have store or be in skipStoreInit.
**Uses:** Command catalog from Phase 1; source analysis of cmd/*.go.

### Phase 4: Worker Economy and Ceremony Audit
**Rationale:** Verify that the 27 agent definitions across 3 platforms have caste assignments, clear outputs, and are referenced by commands. Verify ceremony surface integrity.
**Delivers:** Worker economy report; ceremony integrity report; gate classification completeness check.
**Addresses:** Worker economy -- every spawned worker has justified purpose; visual ceremony backed by real state.
**Avoids:** Brittle snapshot tests; check structural properties (caste assigned, output defined, referenced).
**Uses:** Command catalog and data flow map from prior phases.

### Phase 5: Regression Test Suite and Release Integrity
**Rationale:** With all findings verified, write regression tests that freeze the contracts. These become the CI gate for future development.
**Delivers:** New test files in `cmd/` covering command catalog snapshot, data flow edges, parity state, and gate classification.
**Addresses:** Test contracts -- regression gates against drift; release integrity -- one coherent system.
**Avoids:** Brittle snapshot tests by checking structural properties, not string content.
**Uses:** Verified findings from all prior phases.

### Phase 6: Findings Remediation
**Rationale:** The audit phases produce findings but do not fix them. This phase acts on the findings: fixing parity gaps, pruning dead-end artifacts, completing missing contracts.
**Delivers:** All audit findings resolved; regression tests pass clean.
**Addresses:** Release integrity -- one coherent system that ships.
**Avoids:** All pitfalls apply; fixes are separate from findings with separate verification.

### Phase Ordering Rationale

- **Catalog first:** You cannot verify coherence without knowing what exists. Phase 1 is the prerequisite for everything else.
- **Parallelism after catalog:** Phases 2 and 3 both depend on the catalog but are independent of each other; they can run in parallel.
- **Worker audit after data flow:** Phase 4 needs both the command catalog and the data flow map to verify that agents are referenced by commands that actually dispatch them.
- **Regression last:** Phase 5 needs all findings verified before freezing them as test assertions. Freezing premature findings wastes effort.
- **Remediation after audit:** Phase 6 is the only phase that modifies behavior. Keeping it separate preserves audit integrity.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Data Flow):** Tracing data reads/writes across ~100 source files requires careful source analysis. The exact set of data artifacts and their lifecycle transitions need verification during execution.
- **Phase 4 (Worker Economy):** Agent-to-command reference tracking requires understanding the dispatch contract system. The exact mapping between agent names, caste assignments, and dispatch invocations needs runtime verification.
- **Phase 6 (Remediation):** The scope of fixes is unknown until the audit completes. This phase may need sub-phasing based on finding severity.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Catalog):** Walking the Cobra command tree is well-documented and Aether already does it in tests.
- **Phase 2 (Parity):** Extends existing parity tests with known, documented patterns.
- **Phase 5 (Regression):** Writing structural snapshot tests is a standard Go testing pattern.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Zero new dependencies. All components are existing Go/Cobra infrastructure. Verified against source code and go.mod. |
| Features | HIGH | Features are audit operations (catalog, cross-reference, trace, snapshot). Each maps to a specific, existing integration point in the codebase. |
| Architecture | HIGH | Scanner pattern is well-established. Aether already has partial audit infrastructure. Direct source analysis confirms the gap map. |
| Pitfalls | HIGH | Pitfalls come from direct codebase analysis and prior milestone experience. The "audit must not modify behavior" constraint is learned from v1.11/v1.12 regressions. |

**Overall confidence:** HIGH

### Gaps to Address

- **Exact command count verification:** The research references 317 commands but this needs runtime verification during Phase 1. The count may have drifted since research.
- **Command-guide catalog coverage:** No existing test verifies that `commandGuideCatalog()` covers all YAML commands. The gap size is unknown until Phase 2 runs.
- **Dead-end artifact count:** The number of write-only artifacts in `.aether/data/` is unknown. Phase 3 will reveal the scope. Prior milestones added artifacts without always wiring consumers.
- **Agent dispatch mapping completeness:** The mapping from agent definitions to actual dispatch invocations across all three platforms needs runtime verification in Phase 4.
- **Regression test baseline:** The structural properties to snapshot (minimum command count, required flags per command, data flow edges) need to be determined during Phases 1-4 before Phase 5 can freeze them.

## Sources

### Primary (HIGH confidence)
- Direct source analysis: `cmd/root.go`, `cmd/codex_workflow_cmds.go`, `cmd/command_guide.go`, `cmd/source_check.go`, `cmd/ceremony_cmd.go`, `cmd/gate.go`, `cmd/fixer_dispatch.go`, `cmd/circuit_breaker.go`, `cmd/codex_dispatch_contract.go`
- Test infrastructure: `cmd/command_parity_test.go`, `cmd/command_source_hygiene_test.go`, `cmd/visual_wrapper_contract_test.go`, `cmd/codex_e2e_test.go`
- Data structures: `pkg/colony/colony.go`, `pkg/storage/storage.go`
- YAML definitions: `.aether/commands/*.yaml` (60 files)
- Go module: `go.mod` (Go 1.24, Cobra v1.10.2, modernc.org/sqlite v1.50.0)

### Secondary (MEDIUM confidence)
- Prior milestone audit reports: `.planning/v1.10-MILESTONE-AUDIT.md` through `v1.14-MILESTONE-AUDIT.md` -- patterns for what drifts between milestones
- Project context: `.planning/PROJECT.md` -- milestone history, key decisions, architecture patterns
- Feature research references: Erlang/OTP Supervisor, Google ADK, LangGraph multi-agent patterns -- informed the feature categories from FEATURES.md

### Tertiary (LOW confidence)
- Pitfall research from v1.13 (SQLite integration, process lifecycle) -- these are from an older milestone scope but contain still-relevant patterns for data integrity and cross-platform behavior

---
*Research completed: 2026-05-07*
*Ready for roadmap: yes*
