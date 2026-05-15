# Requirements: Aether v1.19

**Defined:** 2026-05-14
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.19 Requirements: TypeScript Host Cutover + Oracle Confidence Recovery

### Host Entry Point (HEP)

- [x] **HEP-01:** `aether host lifecycle` command exists and runs the TS host
- [x] **HEP-02:** `aether host plan` runs plan workflow through TS host
- [x] **HEP-03:** `aether host build` runs build workflow through TS host
- [x] **HEP-04:** `aether host continue` runs continue workflow through TS host
- [x] **HEP-05:** `aether host oracle` runs Oracle workflow through TS host
- [x] **HEP-06:** Publish/update installs and builds TS host correctly in downstream repos
- [x] **HEP-07:** Fallback messaging when Node/host deps are missing

### Wrapper Cutover (WCO)

- [x] **WCO-01:** Claude plan wrapper calls TS host instead of manual Go commands
- [x] **WCO-02:** Claude build wrapper calls TS host instead of manual Go commands
- [x] **WCO-03:** Claude continue wrapper calls TS host instead of manual Go commands
- [x] **WCO-04:** OpenCode plan/build/continue wrappers call TS host
- [x] **WCO-05:** YAML command sources updated to reference host path
- [x] **WCO-06:** Go CLI direct commands still work for fallback
- [x] **WCO-07:** Wrapper files are materially shorter after cutover

### Oracle Iteration Manifests (OIM)

- [x] **OIM-01:** Go `oracle-iterate --plan-only` returns iteration manifest
- [x] **OIM-02:** Go `oracle-iterate-finalize` commits Oracle state
- [x] **OIM-03:** Go owns Oracle state, confidence math, and file writes
- [x] **OIM-04:** TS host calls Oracle iterate commands, never writes to `.aether/data/oracle`

### TS Host Oracle Lifecycle (TOL)

- [x] **TOL-01:** `runOracleLifecycle()` exists in TS host
- [x] **TOL-02:** TS dispatches Oracle workers via platform dispatcher
- [x] **TOL-03:** TS loops until confidence target, max iterations, or stop condition
- [x] **TOL-04:** TS reads Oracle state from Go CLI JSON output
- [x] **TOL-05:** TS never writes directly to `.aether/data/oracle`

### Swarm/Watch Bridge (SWB)

- [x] **SWB-01:** TS host watch command uses Go JSON output for structured display
- [x] **SWB-02:** TS host swarm command uses Go JSON output for structured display
- [x] **SWB-03:** Live visibility restored without wrapper visual parsing

### Release Gate (REL)

- [x] **REL-01:** `npm run typecheck` passes
- [x] **REL-02:** `npm test` passes
- [x] **REL-03:** `go test ./...` passes
- [x] **REL-04:** Downstream smoke test passes after `aether update --force`
- [x] **REL-05:** Claude, OpenCode, and Codex surfaces behave consistently

## Non-Goals

| Feature | Reason |
|---------|--------|
| Interactive Shell | Deferred to v1.20 — host cutover first |
| Rewrite Go in TypeScript | Violates architecture boundary |
| Remove AETHER_OUTPUT_MODE | Centralize behind host, don't remove |
| Visual output parsing | Use JSON contract instead |
| Raw Bash orchestration | Deprecated |

## Architecture Reminder

```
Go                  = safety kernel and runtime authority
TypeScript/Node     = orchestration host and agent control plane
Markdown/YAML/TOML  = editable colony brain
Bash                = small glue and smoke tests only
```

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| HEP-01 | Phase 124 | Complete |
| HEP-02 | Phase 124 | Complete |
| HEP-03 | Phase 124 | Complete |
| HEP-04 | Phase 124 | Complete |
| HEP-05 | Phase 124 | Complete |
| HEP-06 | Phase 124 | Complete |
| HEP-07 | Phase 124 | Complete |
| WCO-01 | Phase 125 | Complete |
| WCO-02 | Phase 125 | Complete |
| WCO-03 | Phase 125 | Complete |
| WCO-04 | Phase 125 | Complete |
| WCO-05 | Phase 125 | Complete |
| WCO-06 | Phase 125 | Complete |
| WCO-07 | Phase 125 | Complete |
| OIM-01 | Phase 126 | Complete |
| OIM-02 | Phase 126 | Complete |
| OIM-03 | Phase 126 | Complete |
| OIM-04 | Phase 126 | Complete |
| TOL-01 | Phase 127 | Complete |
| TOL-02 | Phase 127 | Complete |
| TOL-03 | Phase 127 | Complete |
| TOL-04 | Phase 127 | Complete |
| TOL-05 | Phase 127 | Complete |
| SWB-01 | Phase 128 | Complete |
| SWB-02 | Phase 128 | Complete |
| SWB-03 | Phase 128 | Complete |
| REL-01 | Phase 129 | Complete |
| REL-02 | Phase 129 | Complete |
| REL-03 | Phase 129 | Complete |
| REL-04 | Phase 129 | Complete |
| REL-05 | Phase 129 | Complete |

## Prior Requirements

See `.planning/milestones/v1.18-REQUIREMENTS.md` for validated requirements from v1.18.
