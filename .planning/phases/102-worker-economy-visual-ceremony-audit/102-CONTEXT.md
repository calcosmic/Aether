# Phase 102: Worker Economy & Visual Ceremony Audit - Context

**Gathered:** 2026-05-07
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase audits two interconnected systems: (1) the worker economy — verifying that every spawned worker caste has a documented purpose, produces durable output, and has at least one downstream consumer, and (2) the visual ceremony — verifying that caste identity, stage markers, progress bars, closeout banners, and the Aether wordmark all reflect real runtime state rather than decoration that misleads.

This is a read-only audit phase. It produces documentation (a combined report), wave shape tables for the 5 orchestration commands, and automated tests that freeze findings. It does NOT modify runtime behavior, add/remove workers, or change visual output. Phase 105 acts on the findings.

</domain>

<decisions>
## Implementation Decisions

### Worker Audit Scope
- **D-01:** Audit all 27 defined worker castes — not just actively dispatched ones. Castes that are defined but never spawned are flagged as "unused" findings rather than ignored.
- **D-02:** For each caste, verify three things: (1) documented purpose statement, (2) expected durable output (file, artifact, or state mutation), (3) at least one identified downstream consumer. Simple table format per caste.
- **D-03:** Castes that only return chat output without persisting findings, state, or artifacts are flagged as findings (WORK-02 violations). No fix suggestions in the report — Phase 105 handles remediation.

### Visual Ceremony Boundary
- **D-04:** All 5 visual element categories are in scope: caste identity (emoji + ANSI color + deterministic name), stage markers (section dividers), progress bars, closeout banners, and the Aether ASCII wordmark.
- **D-05:** Pure decorative output (like the Aether ASCII wordmark) is acceptable. Only flag elements that claim a state transition but lack a backing runtime change. The line is: "does this visual element imply something happened in the runtime?" — if yes, it must be verifiable.

### Wave Shape Documentation
- **D-06:** Per-command wave shape tables for the 5 orchestration commands: build, continue, seal, colonize, plan. Each table shows which castes spawn, why, and what they produce.
- **D-07:** Not a unified cross-reference matrix — per-command tables are easier to reference during actual builds. A summary section can note shared patterns.

### Audit Output Format
- **D-08:** Single combined report (WORKER-ECONOMY.md) covering worker economy findings, visual ceremony findings, and wave shape tables. Same severity-classified pattern as Phase 101's KNOWN-GAPS.md.
- **D-09:** Automated spawn coverage test that verifies every caste spawned in dispatch code has a documented purpose/output entry in the report. Catches undocumented spawns. Follows Phase 101's parity test pattern.

### Claude's Discretion
- Exact test file structure and naming
- How to extract caste spawn sites from dispatch code (grep patterns vs AST)
- Whether to include spawn frequency data alongside purpose/output/consumer
- Visual ceremony verification method (static analysis of output call sites vs runtime tracing)
- Report section ordering and formatting details

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 100 Artifacts (foundation)
- `cmd/audit_catalog.go` — audit-catalog command producing Go runtime truth
- `cmd/contracts/*.md` — 16 lifecycle contract documents with input/output/state patterns
- `cmd/testdata/command_catalog.json` — Frozen golden catalog

### Phase 101 Artifacts (pattern reference)
- `cmd/parity_test.go` — 4-function parity test with golden file pattern
- `.planning/phases/101-platform-parity-verification/KNOWN-GAPS.md` — Severity-classified gap report (format reference)
- `.planning/phases/101-platform-parity-verification/101-CONTEXT.md` — D-01 through D-08 decisions

### Worker & Visual System (the audit targets)
- `cmd/codex_visuals.go` — Caste identity maps (emoji, color, label), visual output functions
- `cmd/codex_build.go` — Build dispatch and worker spawning
- `cmd/codex_continue.go` — Continue dispatch and worker spawning
- `cmd/codex_plan.go` — Plan dispatch and worker spawning
- `cmd/ceremony_emitter.go` — Ceremony emission for state transitions
- `cmd/dispatch_runtime.go` — Core dispatch runtime
- `cmd/queen_wave_lifecycle.go` — Queen-managed wave lifecycle
- `cmd/spawn_track.go` — Worker spawn tracking

### Agent Definitions (worker purpose reference)
- `.claude/agents/ant/*.md` — 27 Claude Code agent definitions with role descriptions
- `.codex/agents/*.toml` — 27 Codex agent definitions
- `.aether/workers.md` — Worker definitions, spawn protocol

### Requirements
- `.planning/REQUIREMENTS.md` — WORK-01, WORK-02, WORK-03, VIZ-01, VIZ-02 definitions
- `.planning/ROADMAP.md` — Phase 102 goal and success criteria

### Project Context
- `CLAUDE.md` — The 27 Agents table (tier, agent, role)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/codex_visuals.go` contains `casteEmojiMap` (27 entries), `casteColorMap`, `casteLabelMap` — the authoritative caste registry. Any caste in these maps but never spawned is a finding.
- Phase 101's `cmd/parity_test.go` establishes the golden file + report pattern. Phase 102 follows the same structure.
- `cmd/contracts/*.md` from Phase 100 shows the per-command contract format that wave shape tables can reference.
- `.claude/agents/ant/*.md` files contain role descriptions for each worker — these are the documented "purpose" statements to verify against.

### Established Patterns
- Dispatch code in `cmd/codex_*.go` follows a consistent pattern: subcommand → context assembly → worker spawn → result collection. Grep for `subagent_type=` or caste names to find all spawn sites.
- Visual ceremony functions in `cmd/codex_visuals.go` and `cmd/ceremony_emitter.go` follow a call-site pattern where each visual output is triggered by a state transition.
- KNOWN-GAPS.md from Phase 101 uses Critical/Warning/Info severity tiers — same pattern here.

### Integration Points
- Spawn coverage test reads dispatch code (grep for spawn sites) and cross-references against the report's caste table
- Visual ceremony audit reads output function call sites and verifies they map to real state transitions
- Wave shape tables reference Phase 100 lifecycle contracts for input/output patterns

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 102-Worker Economy & Visual Ceremony Audit*
*Context gathered: 2026-05-07*
