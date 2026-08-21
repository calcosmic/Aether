# Accepted Tech Debt

**Phase:** 105 (Findings Remediation & Final Validation)
**Date:** 2026-05-08

## Codex TOML Coverage (Phase 101)

**Finding:** 33 of 60 commands have no Codex TOML agent.
**Decision:** Accepted.
**Rationale:** TOML agents represent worker castes (builder, watcher, scout, etc.), not command wrappers. All 60 commands are covered by `commandGuideCatalog()`. Adding 33 wrapper TOML files would duplicate the command-guide surface without adding value.
**File:** `.codex/agents/*.toml` (27 worker caste agents)

## Output Mode Classification (Phase 100)

**Finding:** `classifyOutputMode` uses a conservative heuristic (only `--json` flag = "json+visual").
**Decision:** Accepted.
**Rationale:** Static analysis of RunE function bodies is limited. The classification is deliberately conservative rather than over-claiming. This is documented in the Phase 100 verification report.
**File:** `cmd/audit_catalog.go`

## Colony-Prime Section Count Ranges (Phase 104)

**Finding:** Regression snapshot uses acceptable ranges (15-17 and 4-6) rather than exact matches for colony-prime and capsule sections.
**Decision:** Accepted.
**Rationale:** Minor legitimate additions should not require golden file updates. Ranges provide flexibility while still catching major drift.
**File:** `cmd/regression_test.go`

## Runtime Command Orphans (Phase 105)

**Finding:** The runtime catalog contains 300+ commands; only 60 have YAML wrappers.
**Decision:** Accepted.
**Rationale:** Most runtime commands are internal subcommands, system utilities, or subsystem commands intentionally not surfaced as user-facing slash commands. The reverse parity check (TestNoOrphanRuntimeCommands) is bounded to aliased commands only.
**File:** `cmd/parity_test.go`

## Flag Parity Coverage (Phase 105)

**Finding:** Most YAML command definitions do not include a `flags` array.
**Decision:** Accepted.
**Rationale:** Runtime is the source of truth for flags. YAML files define flags only when wrapper orchestration needs to reference them explicitly. TestFlagParityAcrossSurfaces skips commands with no YAML flag data.
**File:** `cmd/parity_test.go`
