# Phase 109: TypeScript Orchestration Host Prototype - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a minimal TypeScript orchestration host prototype that drives `plan -> build 1 -> continue` through Go manifests and finalizers. The host calls Go `--plan-only` commands for JSON manifests, dispatches platform workers from those manifests, records spawn lifecycle events, and calls Go finalizers to commit state — never writing `.aether/data/` directly. This is an internal prototype to validate the hybrid runtime boundary; it does not ship to users.

</domain>

<decisions>
## Implementation Decisions

### Host Runtime Model
- **D-01:** TS host runs as a Node script invoked by Go subprocess (`node .aether/ts-host/dist/host.js`). Go passes the manifest file path as a CLI argument. The host orchestrates and calls Go CLI commands back.
- **D-02:** TS host lives in `.aether/ts-host/` as a separate package from the ceremony narrator (`.aether/ts/`), per Phase 106 decision D-08. Own package.json, tsconfig.json, and src/ directory.
- **D-03:** TS host imports ceremony rendering functions from `@aether/ceremony-narrator` (in `.aether/ts/`) when it needs to render output. The narrator stays untouched.
- **D-04:** This is an internal prototype only — not shipped, not installed via `aether update`. Lives in `.aether/ts-host/` as source code. Future phases decide whether to distribute it.

### Worker Dispatch Mechanism
- **D-05:** TS host spawns platform workers (Claude/OpenCode) as subprocess exec calls per the Go manifest. Each spawn gets its worker description (task, caste, prompt) from manifest fields.
- **D-06:** TS host records real spawn-log/spawn-complete events via Go CLI subcommands (`aether spawn-log-write`, `aether spawn-complete-write`). This satisfies HOST-06 and restores Classic spawn-logger behavior.

### Integration Seam Shape
- **D-07:** Go writes manifest JSON to a file, passes the file path to TS host via CLI arg. TS host reads the manifest, orchestrates workers, then calls Go finalizer CLI commands passing data via file paths or CLI args. File-based exchange — simple, debuggable, works across processes.
- **D-08:** TS host consumes Go output exclusively in `AETHER_OUTPUT_MODE=json`. Never parses visual/ANSI output. This directly satisfies anti-pattern #2 in the boundary contract.

### Scope Boundary for Prototype
- **D-09:** Success threshold: prove the full `plan -> build 1 -> continue` lifecycle works end-to-end for one phase. The Go→TS manifest→worker→finalizer loop must complete successfully.
- **D-10:** Verification approach is Claude's discretion — either reuse Phase 108 golden tests against TS host output, write new integration tests, or a combination. The planner should choose what proves the boundary most effectively.

### Claude's Discretion
- Worker dispatch implementation details (exact spawn mechanism, error handling, timeout behavior)
- Test strategy for verifying prototype (golden test reuse vs new integration tests)
- How the TS host discovers the Go binary path
- Error handling patterns for Go subprocess failures
- Whether the TS host needs its own test framework or reuses Go test infrastructure

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Boundary Contract (from Phase 106)
- `.aether/references/contracts/runtime-boundary-contract.md` — Defines Go/TS/Assets/Bash ownership. TS host MUST follow anti-patterns: no direct state writes, no visual output parsing, no wrapper-owned recovery menus.

### Go Runtime Authority
- `cmd/codex_build.go` — Build manifest generation (`--plan-only` mode produces JSON without state mutation)
- `cmd/codex_build_finalize.go` — Build finalizer with provenance validation (TS host calls this to commit state)
- `cmd/codex_plan_finalize.go` — Plan finalizer (TS host calls this to commit plan)
- `cmd/codex_continue_finalize.go` — Continue finalizer with verification gates (TS host calls this to advance state)
- `cmd/codex_dispatch_contract.go` — Dispatch contract structures (the JSON manifest schema the TS host consumes)
- `cmd/codex_visuals.go` — Visual rendering (Go owns this, TS host does NOT parse it)
- `cmd/command_guide.go` — Command guide metadata

### TypeScript Surface
- `.aether/ts/package.json` — Existing ceremony narrator package (`@aether/ceremony-narrator`). TS host imports from this for rendering.
- `.aether/ts/narrator.ts` — Ceremony narrator entry point
- `.aether/ts/tsconfig.json` — TypeScript config (TS host should follow similar config)

### Golden Test Behavioral Contract (from Phase 108)
- `cmd/golden_workflow_test.go` — Golden tests defining the behavioral contract the TS host must eventually match
- `cmd/testdata/golden_plan.txt` — Plan ceremony output baseline
- `cmd/testdata/golden_build.txt` — Build ceremony output baseline
- `cmd/testdata/golden_continue.txt` — Continue ceremony output baseline

### Spawn Lifecycle (Go CLI)
- `cmd/codex_spawn_log.go` — Spawn-log write subcommand (TS host calls this before each worker)
- `cmd/codex_spawn_complete.go` — Spawn-complete write subcommand (TS host calls this after each worker)

### Architecture and Research
- `.aether/docs/hybrid-runtime-strategy-research.md` — Core research converging on the hybrid architecture decision
- `.aether/oracle/synthesis.md` — Oracle research synthesis with 9 answered questions and 106 sources
- `.aether/docs/wrapper-runtime-ux-contract.md` — Defines what Go owns vs what wrappers may add
- `.aether/docs/source-of-truth-map.md` — Maps what owns what in the current runtime

### Classic Baseline Reference
- `v5.4.0` git tag — Classic Node/Bash era with orchestration in `bin/` (spawn-logger.js, state-guard.js, caste-colors.js, event-types.js)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Go `--plan-only` flag:** `aether build 1 --plan-only` already produces a JSON manifest without mutating state. The TS host consumes this output.
- **Go finalizer CLI commands:** `aether build-finalize`, `aether plan-finalize`, `aether continue-finalize` already accept manifest data and commit state atomically. TS host calls these.
- **Go spawn-log CLI:** `aether spawn-log-write` and `aether spawn-complete-write` already record spawn lifecycle events. TS host calls these for HOST-06.
- **Ceremony narrator:** `@aether/ceremony-narrator` in `.aether/ts/` already has TypeScript rendering functions the TS host can import.
- **Golden test infrastructure:** `cmd/golden_workflow_test.go` provides the behavioral baseline the prototype is measured against.

### Established Patterns
- **Plan-only → finalizer pattern:** Go generates manifests without mutating state. TS host consumes the manifest. Go finalizer validates provenance and commits. This is the core integration seam.
- **JSON output mode:** `AETHER_OUTPUT_MODE=json` already produces structured JSON output for all commands. TS host uses this exclusively.
- **File-based data exchange:** `.aether/data/` already contains manifest files, build packets, and spawn logs. The TS host reads manifests from here and writes nothing to this directory.
- **Companion file distribution:** Files in `.aether/` are published to the hub via `aether publish`. The TS host source lives in `.aether/ts-host/` following this convention.

### Integration Points
- TS host reads Go manifest from file path (Go writes to `.aether/data/build/phase-N/manifest.json`)
- TS host calls Go CLI commands for spawn-log, spawn-complete, and finalizers
- TS host spawns platform CLI tools (claude, opencode) as subprocesses
- Golden tests verify the lifecycle produces correct state transitions regardless of which driver (Go direct or TS host) orchestrates

</code_context>

<specifics>
## Specific Ideas

- The prototype should demonstrate one complete `plan -> build 1 -> continue` cycle driven by the TS host
- Each worker spawn should have a real spawn-log entry before and spawn-complete entry after
- If the TS host cannot complete the full lifecycle, it MUST document the exact blocker with a reproducible test (per HOST-07)
- The boundary contract anti-patterns must hold: verify no `.aether/data/` writes from TS code, verify no visual output parsing
- Classic v5.4.0 `bin/spawn-logger.js` is the reference implementation for spawn tracking

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 109-TypeScript Orchestration Host Prototype*
*Context gathered: 2026-05-12*
