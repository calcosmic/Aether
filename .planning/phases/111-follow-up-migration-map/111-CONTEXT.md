# Phase 111: Follow-up Migration Map - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Produce a written migration map with milestone-ready plans for three deferred capabilities: Oracle/RALF confidence iteration, swarm visibility, and broader build/continue parity. Each plan includes phases, requirements, and success criteria — ready for `/gsd-plan-phase` without additional research. This is a documentation-only phase; no new runtime code.

</domain>

<decisions>
## Implementation Decisions

### Migration Granularity
- **D-01:** Milestone-ready granularity — each deferred item gets its own milestone with phases, requirements, and success criteria. The output is detailed enough that `/gsd-plan-phase` can proceed without another discuss cycle, but not so detailed that it includes task-level breakdowns.

### Prioritization and Ordering
- **D-02:** Sequential ordering: Oracle/RALF first, then swarm visibility, then broader build/continue parity. Oracle goes first because it proves the TS host can handle complex iterative flows. Swarm goes second as a simpler dispatch use case. Parity goes last because it depends on the patterns established by the first two.
- **D-03:** Dependency chain is explicit — parity depends on Oracle and swarm patterns being proven. Do not parallelize.

### Scope Boundaries
- **D-04:** Migration only — each item migrates existing Go behavior to TS host orchestration. No new features, no improvements beyond what's needed for the TS host integration. Respect the Go/TS boundary contract from Phase 106.
- **D-05:** Oracle migration = TS host drives the RALF loop with confidence targets, using Go manifests and finalizers. The Oracle loop logic stays in Go; TS host handles lifecycle orchestration.
- **D-06:** Swarm migration = TS host renders swarm display output (tree, JSON, text formats already exist in Go). Go owns the data, TS host owns the presentation.
- **D-07:** Parity migration = all remaining flows (colonize, seal, swarm, oracle) use TS host orchestration. Plan/build/continue already work through TS host (Phase 109).

### Claude's Discretion
- Exact milestone version numbers (v1.17, v1.18, etc.)
- Phase count per milestone
- Requirement ID naming convention
- Whether to produce one combined document or separate documents per item
- How to format the map (markdown document, separate milestone files, etc.)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Boundary Contract (from Phase 106)
- `.aether/references/contracts/runtime-boundary-contract.md` — Defines Go/TS/Assets/Bash ownership. All migration plans must respect this contract.

### TS Host (from Phase 109)
- `.aether/ts-host/src/lifecycle.ts` — Current TS host lifecycle orchestrator, shows existing patterns
- `.aether/ts-host/src/go-bridge.ts` — Boundary enforcement, Go command invocation patterns
- `.aether/ts-host/src/worker-dispatch.ts` — Worker dispatch patterns for the TS host

### Go Oracle and Swarm (migration targets)
- `cmd/compatibility_cmds.go` — Oracle RALF loop implementation with confidence targets
- `cmd/swarm_display*.go` — Swarm display rendering (tree, JSON, text formats)
- `cmd/codex_visuals.go` — Visual rendering with caste identity and stage markers

### Existing Test Infrastructure
- `cmd/safety_invariant_test.go` — Safety invariants that must hold after migration
- `cmd/boundary_contract_test.go` — Boundary contract tests

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **TS host lifecycle pattern** (Phase 109): The `lifecycle.ts` already handles plan→build→continue with Go manifest/finalizer calls. Oracle and swarm can follow this same pattern.
- **Go Oracle loop** (`cmd/compatibility_cmds.go`): Full RALF loop with depth, confidence targets, scope, max iterations. Already tested. Needs TS host wrapper, not rewrite.
- **Go swarm display** (`cmd/swarm_display*.go`): Tree, JSON, text rendering already exists. TS host just needs to invoke and present.
- **Boundary contract tests** (`cmd/boundary_contract_test.go`, `cmd/safety_invariant_test.go`): These prove the Go safety invariants hold and will catch any migration that breaks the boundary.

### Established Patterns
- TS host calls Go `--plan-only` for manifests, then dispatches workers, then calls Go finalizers for state writes
- Go owns all state mutation; TS host owns orchestration and presentation
- Each flow (plan, build, continue) has its own manifest structure and finalizer
- Oracle and swarm currently work through wrapper markdown, not TS host

### Integration Points
- Oracle migration: TS host needs to drive the RALF loop, calling Go for each iteration's research step
- Swarm migration: TS host needs to render swarm display output from Go's existing rendering commands
- Parity migration: colonize and seal flows need the same TS host treatment that plan/build/continue got in Phase 109

</code_context>

<specifics>
## Specific Ideas

- The three items were explicitly deferred in STATE.md with `Deferred to follow-up map` status — this phase fulfills that deferral
- Oracle is the most complex migration because it involves iterative loops (RALF), not single-pass flows
- Swarm is simpler because it's mostly display/rendering — Go already does the heavy lifting
- Parity is the largest scope but most mechanical — it's repeating the Phase 109 pattern for remaining flows

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 111-Follow-up Migration Map*
*Context gathered: 2026-05-12*
