# Phase 101: Platform Parity Verification - Context

**Gathered:** 2026-05-07
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase verifies that all five surfaces — Go runtime (Cobra commands), YAML definitions, Claude wrappers, OpenCode wrappers, and Codex command-guide — agree on command names. It produces a parity report classifying mismatches by severity and a golden test that freezes the parity state so CI catches future drift.

This phase does NOT fix any parity gaps. It is a read-only audit that produces a report (KNOWN-GAPS.md or similar) and a regression test. Phase 105 acts on the findings.

The audit-catalog from Phase 100 provides the Go runtime truth. This phase cross-references that against the other four surfaces.

</domain>

<decisions>
## Implementation Decisions

### Parity Gap Severity Classification
- **D-01:** Parity mismatches are classified in three tiers: Critical (wrong flag name, phantom command, or behavior that doesn't match runtime), Warning (description mismatch, lifecycle command missing from Codex), Info (formatting only, non-lifecycle Codex gap).
- **D-02:** The parity report includes counts per tier and a summary, but no fix suggestions. Researcher/planner decide how to fix gaps.
- **D-03:** A wrapper or YAML file referencing a command NOT in the Go runtime audit-catalog is flagged as a Critical gap (phantom command).

### Codex Coverage Scope
- **D-04:** The 33 commands that have YAML definitions but no Codex TOML agent are flagged as Info-level gaps. All 60 commands are checked against all five surfaces.
- **D-05:** If a lifecycle command (per the D-06 list from Phase 100) is missing from Codex, it is escalated to Warning severity instead of Info.

### Test Freeze Approach
- **D-06:** Parity tests freeze current state. Known drift is recorded in a KNOWN-GAPS.md that Phase 105 resolves. Tests pass today to keep CI green.
- **D-07:** The parity golden test freezes command names only from each surface — not flags or descriptions. This keeps the golden file maintainable while still catching the most common drift (command additions/removals).
- **D-08:** A single combined test checks all 5 surfaces at once, rather than per-surface-pair tests. Simpler, fewer test files, and the report identifies which surface drifted.

### Claude's Discretion
- Exact test file name and structure
- How to extract command names from each surface (YAML parsing, wrapper markdown parsing, Codex TOML parsing, command-guide extraction)
- Whether KNOWN-GAPS.md is a separate file or embedded in the test output
- Parity report format and file location

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 100 Artifacts (primary input)
- `cmd/audit_catalog.go` — audit-catalog command producing Go runtime truth (377 commands)
- `cmd/audit_catalog_test.go` — Golden test pattern to follow
- `cmd/testdata/command_catalog.json` — Frozen golden catalog
- `cmd/contracts/*.md` — 16 lifecycle contract documents with flag details

### Existing Parity Infrastructure (extend, don't replace)
- `cmd/source_check.go` — Existing source-check command that verifies YAML-to-wrapper parity
- `cmd/command_parity_test.go` — Existing parity test between Claude/OpenCode wrappers
- `cmd/command_source_hygiene_test.go` — Wrapper hygiene verification
- `cmd/visual_wrapper_contract_test.go` — Visual ceremony contract checks
- `cmd/command_guide.go` — commandGuideCatalog() function producing Codex command guide
- `cmd/command_guide_test.go` — Tests for command guide generation

### Surface Definitions (the five surfaces to compare)
- `.aether/commands/*.yaml` — 60 YAML source definitions (command names, flags, descriptions)
- `.claude/commands/ant/*.md` — 60 Claude Code wrapper markdown files
- `.opencode/commands/ant/*.md` — 60 OpenCode wrapper markdown files
- `.codex/agents/*.toml` — 27 Codex TOML agent definitions
- `cmd/command_guide.go` — commandGuideCatalog() producing Codex command-guide output

### Project Context
- `.planning/REQUIREMENTS.md` — PLAT-01, PLAT-02, PLAT-03 definitions
- `.planning/ROADMAP.md` — Phase 101 goal and success criteria
- `.planning/phases/100-command-inventory-lifecycle-contracts/100-CONTEXT.md` — Phase 100 decisions (D-01 through D-09)
- `.planning/phases/100-command-inventory-lifecycle-contracts/100-RESEARCH.md` — Cobra tree walking patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `buildAuditCatalog(rootCmd)` in `cmd/audit_catalog.go` — already walks Go runtime and produces []CatalogEntry with command names. The parity test can call this directly.
- `source-check` in `cmd/source_check.go` — existing parity verification between YAML and wrappers. Phase 101 extends this with the full 5-surface check.
- `commandGuideCatalog()` in `cmd/command_guide.go` — produces the Codex command-guide surface. Already walks Cobra tree.
- Golden test pattern from Phase 100 (`TestAuditCatalogGolden`) — same pattern works for parity snapshots.

### Established Patterns
- YAML frontmatter parsing in wrapper files — wrappers start with YAML frontmatter containing command metadata
- TOML agent definitions in `.codex/agents/` — each has a `name` and `command` field
- `--json` flag on CLI commands for structured output
- `testdata/` for golden files

### Integration Points
- New parity test reads from the same surfaces as `source_check.go` but compares against `buildAuditCatalog()` output
- KNOWN-GAPS.md lives in the phase directory (`.planning/phases/101-*`) — consumed by Phase 105

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 101-Platform Parity Verification*
*Context gathered: 2026-05-07*
