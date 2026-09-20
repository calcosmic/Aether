# Milestone v1.19: TypeScript Host Cutover + Oracle Confidence Recovery

**Defined:** 2026-05-14
**Depends on:** v1.18 Hybrid Runtime Parity & Release Gate (shipped)

## Goal

Make the TypeScript host the primary orchestration layer for real user workflows. Cut over plan/build/continue wrappers from manual Go command chains to host-backed orchestration. Restore Oracle/RALF confidence iteration through the hybrid manifest/finalizer boundary.

## Architecture Principle

- **Go** = safety kernel (state, locks, finalizers, validation, install/update/publish)
- **TypeScript** = orchestration host (worker dispatch, ceremony, platform adapters, Oracle loops)
- **Markdown/YAML/TOML** = editable colony brain (commands, agents, skills)
- **AETHER_OUTPUT_MODE** = internal contract between Go and TS (not wrapper-level)

## Phases

### Phase 124: Host Entry Point
- Add `aether host <workflow>` CLI entry (lifecycle, plan, build, continue, oracle)
- Ensure publish/update installs and builds TS host correctly
- Add fallback messaging if Node/host deps missing

### Phase 125: Wrapper Cutover — Plan/Build/Continue
- Update Claude/OpenCode wrappers and YAML sources to call TS host
- Keep Go CLI direct commands working for fallback
- Reduce explicit AETHER_OUTPUT_MODE references in wrappers

### Phase 126: Oracle Iteration Manifests
- Add Go commands: `oracle-iterate --plan-only`, `oracle-iterate-finalize`
- Go owns Oracle state, confidence math, file writes, finalizers

### Phase 127: TS Host Oracle Lifecycle
- Add `runOracleLifecycle()` in TS
- TS dispatches Oracle workers, loops until confidence target / max iterations / stop
- No TS writes to `.aether/data/oracle`

### Phase 128: Swarm/Watch Host Bridge
- Add TS host watch/swarm display using Go JSON output
- Restore live visibility without wrapper visual parsing

### Phase 129: Release Gate
- Typecheck, TS tests, Go tests
- Downstream smoke test
- Verify all 3 platforms point to same host-backed behavior

## Non-Goals

- Interactive shell (deferred)
- Rewrite Go runtime in TypeScript
- Remove AETHER_OUTPUT_MODE (centralize it)
- Visual output parsing as authoritative
- Raw Bash orchestration

## Acceptance Criteria

1. `aether host lifecycle` drives plan -> build -> continue
2. Claude/OpenCode plan/build/continue wrappers are materially shorter and host-backed
3. Oracle runs through TS host with confidence iteration and Go-owned state
4. `npm --prefix .aether/ts-host run typecheck` passes
5. `npm --prefix .aether/ts-host test` passes
6. `go test ./...` passes
7. Downstream repo can run host-backed workflow after `aether update --force`
