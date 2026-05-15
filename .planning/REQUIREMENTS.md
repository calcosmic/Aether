# Requirements: Aether v1.20

**Defined:** 2026-05-15
**Core Value:** The TS host contract must be honest — docs, CLI flags, tests, and wrapper behavior agree on who owns what.

## v1.20 Requirements: Host Contract Hardening and Wrapper Reality Check

### Host Surface Completeness (HSC)

- [ ] **HSC-01:** `aether host plan` exists, accepts `--depth`, `--planning-depth`, and executes end-to-end
- [ ] **HSC-02:** `aether host build` exists and dispatches to TS host
- [ ] **HSC-03:** `aether host continue` exists, accepts `--verification-depth`, and executes end-to-end
- [ ] **HSC-04:** `aether host oracle` exists and dispatches Oracle lifecycle
- [ ] **HSC-05:** `aether host watch` exists, accepts `--no-dashboard`, and renders status
- [ ] **HSC-06:** `aether host swarm` exists, accepts `<target>` and `--no-dashboard`
- [ ] **HSC-07:** `aether host lifecycle` exists and orchestrates full plan→build→continue

### Test Coverage (TCV)

- [ ] **TCV-01:** `aether host plan --depth balanced --planning-depth standard` is exercised in tests
- [ ] **TCV-02:** `aether host continue --verification-depth heavy` is exercised in tests
- [ ] **TCV-03:** `aether host watch --no-dashboard` is exercised in tests
- [ ] **TCV-04:** `aether host swarm <target> --no-dashboard` is exercised in tests
- [ ] **TCV-05:** Every documented `aether host` subcommand has at least one test invocation

### Wrapper Ownership (WRO)

- [ ] **WRO-01:** Decision recorded: wrappers are either "host-assisted orchestrators" or TS host fully owns dispatch/finalizers
- [ ] **WRO-02:** Wrapper code updated to match the recorded decision
- [ ] **WRO-03:** No wrapper duplicates host-owned logic (verification, gating, finalizers)
- [ ] **WRO-04:** Wrapper README or contract doc explains the ownership boundary

### Lifecycle Honesty (LHO)

- [x] **LHO-01:** No production-facing synthetic lifecycle shortcuts exist
- [x] **LHO-02:** Any remaining synthetic paths are gated behind `--simulate` flag
- [x] **LHO-03:** `--simulate` is documented and tested

### Oracle Storage (ORS)

- [ ] **ORS-01:** Oracle iteration state uses locked/atomic Go-owned storage pattern
- [ ] **ORS-02:** TS host reads Oracle state via Go CLI JSON, never writes directly
- [ ] **ORS-03:** Oracle state is recoverable after crash/interrupt

### Doc-CLI Alignment (DCA)

- [ ] **DCA-01:** Executable YAML smoke test exists that exercises every documented command
- [ ] **DCA-02:** Smoke test fails if a documented flag is rejected by the CLI
- [ ] **DCA-03:** Smoke test runs in CI or as part of release gate

## Non-Goals

| Feature | Reason |
|---------|--------|
| New workflows | v1.20 is hardening existing surface, not adding features |
| Interactive shell | Deferred to v1.21 |
| Dashboard redesign | Out of scope — only flag parity and testing |

## Architecture Reminder

```
Go                  = safety kernel and runtime authority
TypeScript/Node     = orchestration host and agent control plane
Markdown/YAML/TOML  = editable colony brain (wrappers)
Bash                = small glue and smoke tests only
```

The v1.20 boundary decision: if wrappers become "host-assisted orchestrators", they add colony framing and narration but must not mutate state, duplicate verification, or gate logic. If TS host fully owns dispatch, wrappers are thin pass-throughs that call `aether host` and render the result.

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| HSC-01 | Phase 130 | Pending |
| HSC-02 | Phase 130 | Pending |
| HSC-03 | Phase 130 | Pending |
| HSC-04 | Phase 130 | Pending |
| HSC-05 | Phase 130 | Pending |
| HSC-06 | Phase 130 | Pending |
| HSC-07 | Phase 130 | Pending |
| TCV-01 | Phase 131 | Pending |
| TCV-02 | Phase 131 | Pending |
| TCV-03 | Phase 131 | Pending |
| TCV-04 | Phase 131 | Pending |
| TCV-05 | Phase 131 | Pending |
| WRO-01 | Phase 132 | Pending |
| WRO-02 | Phase 132 | Pending |
| WRO-03 | Phase 132 | Pending |
| WRO-04 | Phase 132 | Pending |
| LHO-01 | Phase 133 | Complete |
| LHO-02 | Phase 133 | Complete |
| LHO-03 | Phase 133 | Complete |
| ORS-01 | Phase 134 | Complete |
| ORS-02 | Phase 134 | Complete |
| ORS-03 | Phase 134 | Complete |
| DCA-01 | Phase 135 | Pending |
| DCA-02 | Phase 135 | Pending |
| DCA-03 | Phase 135 | Pending |
