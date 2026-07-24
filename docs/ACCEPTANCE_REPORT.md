# Phase 159 Acceptance Report — v1.24 Hybrid Architecture Salvage

> **Date:** 2026-05-24
> **Phase:** 159 — End-to-End Acceptance
> **Milestone:** v1.24 Hybrid Architecture Salvage

---

## Summary

Phase 159 validates the hybrid system: Go runtime spine + TypeScript control plane + editable colony assets (Markdown/YAML/JSON). All 7 acceptance criteria were evaluated against the running system.

The hybrid architecture is functional. Go tests pass without regressions, TypeScript control plane loads and executes colony assets, the demo flow runs end-to-end with event emission, and audits confirm no hardcoded behaviour remains in Go source files. The Classic parity checklist documents 80% verifiable items, exceeding the 50% threshold.

---

## Test Results

| Requirement | Test Command | Status | Evidence |
|-------------|--------------|--------|----------|
| TEST-01 | `go test ./...` | **PASS** | All packages green (cmd 93.8s, others cached) |
| TEST-02 | `npm run test:schemas` | **PASS** | 25 tests passing in 4 files |
| TEST-03 | `npm run test:control` | **PASS** | 43 tests passing in 7 files |
| TEST-04 | `npm run aether:control -- --task "create a small test file and verify it"` | **PASS** | Plan completed, 3 phases executed, 24 events emitted |
| TEST-05 | `bash scripts/audit-hardcoded.sh` | **PASS** | 4/4 checks passed — no agent directives in non-test Go |
| TEST-06 | `bash scripts/audit-extraction.sh` | **PASS** | 5/5 checks passed — 27 agents, 28 prompts, 9 phases |
| TEST-07 | `.aether/docs/PARITY_CLASSIC_VS_GO.md` | **PASS** | 12/15 items verifiable (80%) |

---

## Verification Methodology

### Automated vs Manual

| Test | Type | Rationale |
|------|------|-----------|
| TEST-01 through TEST-04 | Automated | Commands produce deterministic exit codes and output |
| TEST-05 through TEST-06 | Automated (grep-based audit) | Static analysis of Go source for forbidden patterns |
| TEST-07 | Manual + documented | Checklist counting method is explicit; arithmetic is verifiable |

### How the 50% Threshold Was Calculated

The parity checklist contains 15 rows. The counting method (documented in `.aether/docs/PARITY_CLASSIC_VS_GO.md`) defines:

- **MATCH** (10 items) = verifiable — behaviour matches Classic
- **DEGRADED** (2 items) = verifiable — behaviour is observable but differs
- **GAP** (1 item) = not verifiable — behaviour is missing
- **INTENTIONALLY_CHANGED** (6 items in Known Gaps) = not verifiable — deliberately altered

Verifiable items: 10 + 2 = 12. Total checklist items: 15. Percentage: 12 / 15 = **80%**. Threshold: >= 50%. Result: **PASS**.

### What "Verifiable" Means

A checklist item is "verifiable" if the behaviour it describes can be observed and checked against an expected outcome, even if that outcome differs from Classic. A GAP is not verifiable because there is nothing to observe. An INTENTIONALLY_CHANGED item is not verifiable because it is not a parity target.

---

## Known Limitations

1. **Demo flow uses stub adapters.** The `npm run aether:control` demo runs with stub platform adapters (no real LLM calls). This is by design — the acceptance test validates the orchestration pipeline and event emission, not LLM output quality.

2. **Probe coverage analysis is a GAP.** Per the parity checklist, Probe coverage analysis (coverage percentage reporting, test gap identification) is marked GAP. The Go test suite provides coverage, but dedicated Probe caste coverage analysis is not yet implemented.

3. **7 Classic behaviours are INTENTIONALLY_CHANGED.** These were deliberately altered per the v1.24 architecture boundary:
   - Raw Bash state mutation (Go owns state now)
   - Visual output parsing as authority (never parse banners for state)
   - 100+ default agents (8 core agents maximum for v1.24)
   - Cross-colony ledger sharing (repo-specific paths go stale)
   - Real-time ledger sync across agents (YAGNI — serial writes)
   - Interactive shell (deferred post-v1.24)
   - Vector backend for memory (file-backed sufficient)

---

## Cross-References

- **Classic parity checklist:** [`.aether/docs/PARITY_CLASSIC_VS_GO.md`](.aether/docs/PARITY_CLASSIC_VS_GO.md) — golden reference for Classic v5.4.0 vs current runtime
- **Behaviour extraction audit:** [`.planning/phases/phase-152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md`](.planning/phases/phase-152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md) — classification of 162 Go symbols
- **Architecture boundary:** [`.aether/docs/ARCHITECTURE_BOUNDARY.md`](.aether/docs/ARCHITECTURE_BOUNDARY.md) — three-tier boundary and asset-type matrix

---

## Sign-off

- [x] TEST-01 — Go tests pass with no regressions
- [x] TEST-02 — Zod schemas validate against sample YAML
- [x] TEST-03 — Phase runner and orchestrator tests pass
- [x] TEST-04 — Demo flow runs end-to-end with events
- [x] TEST-05 — No prompt text hardcoded in Go (except fallback/error)
- [x] TEST-06 — No agent behaviour exists only in Go
- [x] TEST-07 — Classic parity checklist >=50% verifiable (actual: 80%)

- [x] **Phase 159 Complete**

---

*Report generated during Plan 03 execution of Phase 159.*
