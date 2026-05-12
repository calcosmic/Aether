# Phase 107: Classic Baseline Identification - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Identify the best Classic Aether version (v5.3.0, v5.3.3, or v5.4.0) as a behavior comparison anchor for the hybrid runtime milestone. Produce a documented comparison with evidence, write a smoke-test script verifying the lifecycle runs, and create baseline documentation with selection rationale and behavior checklist.

</domain>

<decisions>
## Implementation Decisions

### Version Comparison
- **D-01:** Full behavioral checklist comparing all 3 candidate versions (v5.3.0, v5.3.3, v5.4.0) against behavior criteria
- **D-02:** Checklist covers all 16 Classic modules with: what each does, expected behavior, which version has it, and 4-category classification (Restore in TS / Keep in Go / Obsolete / Reject as unsafe)
- **D-03:** The comparison should show what changed across versions and explain why the selected version is the bridge between Node era and Go era

### Smoke Test
- **D-04:** Full lifecycle verification — not just exit codes. Test checks: exit codes = 0 for plan/build/continue, output contains ceremony stage markers and caste labels, COLONY_STATE.json changes between commands
- **D-05:** Smoke test is a standalone Bash script (scripts/smoke-test-classic.sh), not a Go test. Simpler, runs in CI without compilation, independent of current Go runtime

### Claude's Discretion
- Smoke test implementation format (chose Bash script for simplicity and CI portability)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Boundary Contract (from Phase 106)
- `.aether/references/contracts/runtime-boundary-contract.md` — Contains the 16-module Classic classification table that this phase must verify and expand into a full behavioral checklist

### Classic Version Source Code
- `v5.4.0:bin/lib/` — 16 modules: spawn-logger, state-guard, caste-colors, event-types, file-lock, state-sync, banner, colors, logger, init, interactive-setup, nestmate-loader, binary-downloader, update-transaction, version-gate, errors
- `v5.3.0:bin/lib/` — 14 modules (no binary-downloader, no version-gate)
- `v5.3.3:bin/lib/` — 14 modules (same as v5.3.0)

### Research
- `.planning/phases/106-boundary-contract/RESEARCH.md` — Architectural responsibility map and Classic module analysis

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 106 contract's Classic Classification table: already has the 4-category classification for all 16 modules. This phase expands each entry into a full behavioral checklist with expected outputs.
- Git tags v5.3.0, v5.3.3, v5.4.0 all exist in the repo. Can be checked out directly for testing.

### Established Patterns
- v5.4.0 added binary-downloader.js and version-gate.js — these are the Go-delegation bridge modules that make v5.4.0 the natural anchor
- All three versions share identical directory structure (bin/cli.js, bin/lib/, bin/generate-commands.js, bin/npx-entry.js)
- Classic versions use Node.js (CommonJS require), not the current Go binary

### Integration Points
- The behavioral checklist feeds directly into Phase 109 (TypeScript Orchestration Host) — the "Restore in TS" modules define what the TS host must reimplement
- The smoke test validates that Classic works as expected, establishing a baseline for Phase 108 golden tests

</code_context>

<specifics>
## Specific Ideas

- The comparison document should live alongside the Phase 106 contract in `.aether/references/` or in the phase directory
- v5.4.0 is the expected winner: it has all 16 modules including the Go-delegation bridge (binary-downloader, version-gate), and Phase 106's contract already references it
- The smoke test must handle the fact that Classic versions expect Node.js and may not work with the current Go-based setup — the test environment needs to simulate the Classic Node.js context

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 107-Classic Baseline Identification*
*Context gathered: 2026-05-12*
