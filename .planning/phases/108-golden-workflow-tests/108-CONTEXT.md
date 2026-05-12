# Phase 108: Golden Workflow Tests - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Golden/snapshot tests for the `plan -> build 1 -> continue` lifecycle that capture full ceremony output, worker activity, and state side effects. Tests run in CI and fail if ceremony, worker activity, or state behavior regresses. This is the behavioral regression safety net for the hybrid runtime milestone.

</domain>

<decisions>
## Implementation Decisions

### Ceremony Snapshot Format
- **D-01:** Use full visual output snapshot (ANSI-stripped) as golden text files. The test captures the complete ceremony output from `plan -> build 1 -> continue` and compares against a golden baseline.
- **D-02:** Strip ANSI escape codes before snapshotting. Golden files contain clean, readable text. Tests don't break on color tweaks, only on structural ceremony changes.
- **D-03:** Use the standard Go golden test `-update` flag pattern to regenerate golden files when ceremony output intentionally changes. CI fails if golden is stale.

### Claude's Discretion
- Test implementation format (Go golden test files following existing `setupBuildFlowTest` patterns)
- State mutation assertion approach (how to verify COLONY_STATE.json writes only happen after finalizers)
- CI integration (alongside existing `go test ./...` or separate target)
- Golden file location (alongside test files or in a dedicated testdata/ directory)
- Whether to also snapshot JSON output from `--plan-only` mode alongside visual output

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Boundary Contract (from Phase 106)
- `.aether/references/contracts/runtime-boundary-contract.md` — Defines Go ownership of state mutation, finalizers, and verification contracts. Golden tests must verify these boundaries hold.

### Go Runtime Authority
- `cmd/codex_visuals.go` — Visual rendering functions (casteIdentity, casteLabel, casteEmoji, stage markers). Source of ceremony output structure.
- `cmd/codex_build.go` — Build command with ceremony emission and manifest generation.
- `cmd/codex_continue.go` — Continue command with ceremony emission.
- `cmd/codex_build_finalize.go` — Build finalizer with provenance validation (Go owns).
- `cmd/codex_plan_finalize.go` — Plan finalizer (Go owns).
- `cmd/codex_continue_finalize.go` — Continue finalizer (Go owns).
- `cmd/ceremony_cmd.go` — Ceremony subcommand structures and dispatch types.

### Existing Test Patterns
- `cmd/build_flow_cmds_test.go` — `setupBuildFlowTest` and `createTestColonyState` helpers (reuse these).
- `cmd/build_wrapper_ceremony_test.go` — Wrapper ceremony contract tests (pattern reference).
- `cmd/continue_wrapper_ceremony_test.go` — Continue ceremony contract tests (pattern reference).
- `cmd/codex_visuals_test.go` — Visual output tests with `strings.Contains` checks for stage markers and caste labels.

### Codebase Maps
- `.planning/codebase/TESTING.md` — Test framework, patterns, helpers, and quality gates.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `setupBuildFlowTest(t)` — Creates temp directory with store, sets AETHER_ROOT, captures stdout/stderr. Every lifecycle test should use this.
- `createTestColonyState(t, dataDir, state)` — Creates minimal COLONY_STATE.json. Used to set up phases with tasks for build/continue tests.
- `saveGlobals(t)` / `resetRootCmd(t)` — Test isolation helpers used by all command tests.
- `parseEnvelope(t, output)` — JSON output parsing helper.

### Established Patterns
- Existing tests already check for ceremony elements via `strings.Contains` (stage markers, caste labels, visual banners). Golden tests generalize this into a single snapshot comparison.
- The build visual test (`TestBuildVisualOutputHasStageMarkers`) already verifies: `── Context ──`, `── Tasks ──`, `── Dispatch ──`, `── Verification [heavy] ──`, `── Housekeeping ──`, `── Colony Complete ──`, caste labels (Builder, Watcher), and visual banners.
- Golden files follow Go convention: `testdata/TestName.golden` with `-update` flag regeneration.

### Integration Points
- Phase 106 boundary contract defines what Go owns — golden tests verify no `.aether/data/` writes before finalizers
- Phase 107 Classic baseline provides behavioral comparison context
- Phase 109 TypeScript host will consume these tests as the behavioral contract for what the TS host must produce

</code_context>

<specifics>
## Specific Ideas

- The golden test should run the full lifecycle: `aether plan` → `aether build 1` → `aether continue` in sequence, capturing each command's visual output
- Stage markers must appear in correct order (Context → Tasks → Dispatch → Verification → Housekeeping → Colony Complete)
- Caste labels must be present for dispatched workers (Builder, Watcher at minimum)
- The test must verify COLONY_STATE.json changes between commands (e.g., status transitions from ready to building to continuing)
- Golden files should be human-readable plain text (ANSI-stripped) for easy diffing in code review

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 108-Golden Workflow Tests*
*Context gathered: 2026-05-12*
