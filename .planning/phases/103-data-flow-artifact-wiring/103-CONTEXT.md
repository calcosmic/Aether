# Phase 103: Data Flow & Artifact Wiring - Context

**Gathered:** 2026-05-07
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase audits the entire data pipeline — tracing every artifact file from its writer command to its reader/consumer. The scope covers all files in `.aether/data/` (COLONY_STATE, pheromones, midden/, instincts, session, handoffs, constraints, assumptions, behavior-observations, survey/) plus hub-level artifacts (QUEEN.md, hive/wisdom.json, eternal/, registry/).

For each artifact, the audit identifies: (1) the Go function that writes it, (2) the Go function that reads it, and (3) whether colony-prime injects it into worker prompts. Graph and survey artifacts get actual wiring verification.

This is a read-only audit phase. It produces a data flow report (DATA-FLOW.md) and automated tests that freeze findings. It does NOT modify runtime behavior, add/remove artifacts, or change colony-prime injection logic. Phase 105 acts on the findings.

</domain>

<decisions>
## Implementation Decisions

### Artifact Inventory Scope
- **D-01:** Audit EVERYTHING — all files in `.aether/data/` (not just the named ones from DATA-01) plus all hub-level artifacts (`~/.aether/QUEEN.md`, `hive/wisdom.json`, `eternal/`, `registry/`). The ROADMAP names the core files, but a complete scan catches edge cases and newer artifacts that naming would miss.

### Consumer Tracing Depth
- **D-02:** For each artifact, trace at command + prompt section level: (1) the Go function/subcommand that writes the file, (2) the Go function/subcommand that reads the file, (3) whether colony-prime injects the data into worker prompts (and which prompt section name). This is the sweet spot — detailed enough to find gaps, practical enough to maintain.

### Graph & Survey Wiring
- **D-03:** Verify actual wiring — check whether graph artifacts (pkg/graph/) and survey results (.aether/data/survey/) are actually wired into colony-prime context injection. If they're not wired, document the gap as a finding. Don't just document current state; verify the wiring works.

### Report & Test Approach (from established patterns)
- **D-04:** Report follows KNOWN-GAPS.md severity pattern from Phase 101 (Critical/Warning/Info tiers). Single combined report file (DATA-FLOW.md) covering all artifacts.
- **D-05:** Automated tests freeze findings — following the Phase 102 pattern (golden snapshot + report verification tests). Tests verify the report's claims are accurate.
- **D-06:** No fix suggestions in findings. Phase 105 handles all remediation.

### Claude's Discretion
- Exact report file name and structure
- How to extract writer/reader function names from Go source (grep patterns vs AST)
- How to verify colony-prime wiring (read colony_prime_context.go section names vs runtime test)
- Whether to include artifact size or age data alongside writer/reader info
- Test file structure and naming
- How to handle artifacts that are read by user-facing CLI commands (status, resume) vs internal-only consumption

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Colony-Prime Context Injection (the primary consumer)
- `cmd/colony_prime_context.go` — Colony-prime context assembly with 16 named prompt sections (pheromones, instincts, state, handoffs, hive_wisdom, etc.)
- `cmd/colony_prime_test.go` — Tests for colony-prime context assembly
- `cmd/colony_prime_audit_test.go` — Audit tests for colony-prime output

### Data Artifact Writers (Go source)
- `cmd/codex_build.go` — Writes COLONY_STATE, pheromones, handoffs, midden
- `cmd/codex_continue.go` — Writes COLONY_STATE, learnings, instincts, midden
- `cmd/codex_continue_finalize.go` — Finalizes state, learnings, instincts
- `cmd/codex_plan.go` — Writes COLONY_STATE, pending-decisions
- `cmd/codex_colonize.go` — Writes survey/ artifacts
- `cmd/seal_final_review.go` — Writes review ledgers, promotes instincts to hive
- `cmd/pheromone_mgmt.go` — Writes pheromones.json
- `cmd/assumptions.go` — Writes assumptions.json
- `cmd/worker_handoff.go` — Writes worker handoffs

### Data Artifact Storage
- `pkg/storage/` — JSON store with file locking (the low-level read/write layer)
- `pkg/learn/` — Learning pipeline (observations → instincts)
- `pkg/graph/` — Knowledge graph persistence
- `pkg/memory/` — Memory pipeline, instincts, promotion
- `pkg/events/` — Event bus with TTL

### Hub-Level Artifacts
- `cmd/hive.go` — Hive Brain operations (hive-store, hive-read, hive-promote)
- `cmd/queen_wisdom.go` — QUEEN.md wisdom operations
- `cmd/eternal.go` — Eternal memory operations
- `cmd/registry.go` — Colony registry operations

### Phase 100 Artifacts (foundation)
- `cmd/contracts/*.md` — 16 lifecycle contract documents
- `cmd/testdata/command_catalog.json` — Frozen golden catalog

### Phase 101-102 Pattern References
- `cmd/parity_test.go` — Golden file test pattern
- `cmd/worker_economy_test.go` — Audit report verification test pattern
- `.planning/phases/101-platform-parity-verification/KNOWN-GAPS.md` — Report format reference
- `.planning/phases/102-worker-economy-visual-ceremony-audit/WORKER-ECONOMY.md` — Report format reference

### Requirements
- `.planning/REQUIREMENTS.md` — LIFE-03, DATA-01, DATA-02 definitions
- `.planning/ROADMAP.md` — Phase 103 goal and success criteria

### Project Context
- `CLAUDE.md` — Architecture overview with data directory structure

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/colony_prime_context.go` is the authoritative list of what colony-prime injects — it has 16 named prompt sections with source file paths. The audit can extract this list directly.
- Phase 100's `cmd/audit_catalog.go` demonstrates Go-runtime truth extraction — same approach works for data flow.
- Phase 102's `cmd/worker_economy_test.go` demonstrates the golden snapshot + report verification pattern.
- `pkg/storage/` provides the low-level JSON read/write — every data file goes through this layer, making it a reliable place to find all writers/readers.

### Established Patterns
- Data artifacts follow a consistent pattern: Go subcommand → `store.SaveJSON()` → file. The inverse (read) follows `store.LoadJSON()` → Go subcommand.
- Colony-prime prompt sections are named consistently in `colony_prime_context.go` — each section has a `name` field, a `source` file path, and a priority score.
- Review ledgers are accumulated across phases and stored in `.aether/data/review-*.json` files — the audit should verify these persist correctly.
- KNOWN-GAPS.md severity classification (Critical/Warning/Info) is the established report format.

### Integration Points
- The audit reads colony-prime source to extract the 16 prompt section names — these are the "consumers" for most artifacts.
- The audit reads Go source files to find `SaveJSON`/`LoadJSON` calls for each artifact file — these are the writers/readers.
- Graph/survey wiring requires checking both pkg/graph/ and colony_prime_context.go to see if graph data appears in any prompt section.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 103-Data Flow & Artifact Wiring*
*Context gathered: 2026-05-07*
