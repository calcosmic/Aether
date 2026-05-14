# Requirements: Aether v1.18

**Defined:** 2026-05-14
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.18 Requirements: Hybrid Runtime Parity & Release Gate

### TS Host Reliability (REL)

- [ ] **REL-01:** `npm --prefix .aether/ts-host run typecheck` passes with zero errors
- [ ] **REL-02:** Full TS test suite (`npm --prefix .aether/ts-host test`) exits cleanly without hangs
- [ ] **REL-03:** Event bridge teardown awaits full subprocess/readline cleanup
- [ ] **REL-04:** Completion file paths are unique per lifecycle run (no fixed `/tmp/aether-lifecycle/`)
- [ ] **REL-05:** Individual tests and suite mode both pass for lifecycle and golden workflow

### Platform Dispatch Correctness (DSP)

- [ ] **DSP-01:** Codex dispatch passes the worker prompt to `codex exec` (not just `--output-schema`)
- [ ] **DSP-02:** Claude argument construction is tested and verified
- [ ] **DSP-03:** OpenCode argument construction is tested and verified
- [ ] **DSP-04:** Simulation fallback is explicit and does not mask broken real dispatch
- [ ] **DSP-05:** Spawn-log/spawn-complete only records manifest workers

### Go Runtime Test Restoration (GOT)

- [ ] **GOT-01:** `go test ./cmd` passes with zero failures
- [ ] **GOT-02:** Resume dashboard signal injection failure is resolved
- [ ] **GOT-03:** Workspace/planning cleanup state is resolved (deleted files committed or archived)
- [ ] **GOT-04:** Scratch files (`seal-debug.ts`, etc.) are removed or promoted
- [ ] **GOT-05:** Go remains sole authority for `.aether/data` mutation

### Classic Parity Coverage (PAR)

- [ ] **PAR-01:** Golden tests verify build ceremony matches v5.4 baseline
- [ ] **PAR-02:** Golden tests verify continue ceremony matches v5.4 baseline
- [ ] **PAR-03:** Oracle confidence loop behavior is tested against v5.4 baseline
- [ ] **PAR-04:** Swarm/dashboard visibility is tested against v5.4 baseline
- [ ] **PAR-05:** Install/update flow is tested against v5.4 baseline
- [ ] **PAR-06:** State mutation through approved APIs is tested against v5.4 baseline
- [ ] **PAR-07:** Any Classic behavior intentionally not restored is documented

### Release Gate (REL)

- [ ] **REL-06:** Dev channel publish succeeds
- [ ] **REL-07:** Downstream smoke test passes: `aether update --force`, `aether init`, `aether plan`, `aether build 1`, `aether continue`, `aether oracle`
- [ ] **REL-08:** Exact blocker list is recorded before stable release

## Non-Goals

| Feature | Reason |
|---------|--------|
| Interactive Shell | Deferred to v1.19 — blocked until release gate passes |
| New framework adoption | Out of scope — tighten what exists |
| Runtime rewrite | Out of scope — fix what exists |
| Moving Go responsibilities to TS | Violates architecture boundary |
| Restoring raw Bash state mutation | Violates architecture boundary |
| Stable release before gate passes | Gate must be green first |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REL-01 | Phase 119 | Completed |
| REL-02 | Phase 119 | Completed |
| REL-03 | Phase 119 | Completed |
| REL-04 | Phase 119 | Completed |
| REL-05 | Phase 119 | Completed |
| DSP-01 | Phase 120 | Completed |
| DSP-02 | Phase 120 | Completed |
| DSP-03 | Phase 120 | Completed |
| DSP-04 | Phase 120 | Completed |
| DSP-05 | Phase 120 | Completed |
| GOT-01 | Phase 121 | Completed |
| GOT-02 | Phase 121 | Completed |
| GOT-03 | Phase 121 | Completed |
| GOT-04 | Phase 121 | Completed |
| GOT-05 | Phase 121 | Completed |
| PAR-01 | Phase 122 | Completed |
| PAR-02 | Phase 122 | Completed |
| PAR-03 | Phase 122 | Completed |
| PAR-04 | Phase 122 | Completed |
| PAR-05 | Phase 122 | Completed |
| PAR-06 | Phase 122 | Completed |
| PAR-07 | Phase 122 | Completed |
| REL-06 | Phase 123 | Completed |
| REL-07 | Phase 123 | Completed |
| REL-08 | Phase 123 | Completed |

---

## Prior Requirements

See `.planning/milestones/v1.17-REQUIREMENTS.md` for validated requirements from v1.17.
