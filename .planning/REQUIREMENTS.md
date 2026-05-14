# Requirements: Aether v1.19

**Defined:** 2026-05-14
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.19 Requirements: TypeScript Host Cutover + Oracle Confidence Recovery

### Host Entry Point (HEP)

- [ ] **HEP-01:** `aether host lifecycle` command exists and runs the TS host
- [ ] **HEP-02:** `aether host plan` runs plan workflow through TS host
- [ ] **HEP-03:** `aether host build` runs build workflow through TS host
- [ ] **HEP-04:** `aether host continue` runs continue workflow through TS host
- [ ] **HEP-05:** `aether host oracle` runs Oracle workflow through TS host
- [ ] **HEP-06:** Publish/update installs and builds TS host correctly in downstream repos
- [ ] **HEP-07:** Fallback messaging when Node/host deps are missing

### Wrapper Cutover (WCO)

- [ ] **WCO-01:** Claude plan wrapper calls TS host instead of manual Go commands
- [ ] **WCO-02:** Claude build wrapper calls TS host instead of manual Go commands
- [ ] **WCO-03:** Claude continue wrapper calls TS host instead of manual Go commands
- [ ] **WCO-04:** OpenCode plan/build/continue wrappers call TS host
- [ ] **WCO-05:** YAML command sources updated to reference host path
- [ ] **WCO-06:** Go CLI direct commands still work for fallback
- [ ] **WCO-07:** Wrapper files are materially shorter after cutover

### Oracle Iteration Manifests (OIM)

- [ ] **OIM-01:** Go `oracle-iterate --plan-only` returns iteration manifest
- [ ] **OIM-02:** Go `oracle-iterate-finalize` commits Oracle state
- [ ] **OIM-03:** Go owns Oracle state, confidence math, and file writes
- [ ] **OIM-04:** TS host calls Oracle iterate commands, never writes to `.aether/data/oracle`

### TS Host Oracle Lifecycle (TOL)

- [ ] **TOL-01:** `runOracleLifecycle()` exists in TS host
- [ ] **TOL-02:** TS dispatches Oracle workers via platform dispatcher
- [ ] **TOL-03:** TS loops until confidence target, max iterations, or stop condition
- [ ] **TOL-04:** TS reads Oracle state from Go CLI JSON output
- [ ] **TOL-05:** TS never writes directly to `.aether/data/oracle`

### Swarm/Watch Bridge (SWB)

- [ ] **SWB-01:** TS host watch command uses Go JSON output for structured display
- [ ] **SWB-02:** TS host swarm command uses Go JSON output for structured display
- [ ] **SWB-03:** Live visibility restored without wrapper visual parsing

### Release Gate (REL)

- [ ] **REL-01:** `npm run typecheck` passes
- [ ] **REL-02:** `npm test` passes
- [ ] **REL-03:** `go test ./...` passes
- [ ] **REL-04:** Downstream smoke test passes after `aether update --force`
- [ ] **REL-05:** Claude, OpenCode, and Codex surfaces behave consistently

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
| HEP-01 | Phase 124 | Pending |
| HEP-02 | Phase 124 | Pending |
| HEP-03 | Phase 124 | Pending |
| HEP-04 | Phase 124 | Pending |
| HEP-05 | Phase 124 | Pending |
| HEP-06 | Phase 124 | Pending |
| HEP-07 | Phase 124 | Pending |
| WCO-01 | Phase 125 | Pending |
| WCO-02 | Phase 125 | Pending |
| WCO-03 | Phase 125 | Pending |
| WCO-04 | Phase 125 | Pending |
| WCO-05 | Phase 125 | Pending |
| WCO-06 | Phase 125 | Pending |
| WCO-07 | Phase 125 | Pending |
| OIM-01 | Phase 126 | Pending |
| OIM-02 | Phase 126 | Pending |
| OIM-03 | Phase 126 | Pending |
| OIM-04 | Phase 126 | Pending |
| TOL-01 | Phase 127 | Pending |
| TOL-02 | Phase 127 | Pending |
| TOL-03 | Phase 127 | Pending |
| TOL-04 | Phase 127 | Pending |
| TOL-05 | Phase 127 | Pending |
| SWB-01 | Phase 128 | Pending |
| SWB-02 | Phase 128 | Pending |
| SWB-03 | Phase 128 | Pending |
| REL-01 | Phase 129 | Pending |
| REL-02 | Phase 129 | Pending |
| REL-03 | Phase 129 | Pending |
| REL-04 | Phase 129 | Pending |
| REL-05 | Phase 129 | Pending |

## Prior Requirements

See `.planning/milestones/v1.18-REQUIREMENTS.md` for validated requirements from v1.18.
