# Requirements: Aether v1.15

**Defined:** 2026-05-07
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.15 Requirements

### Lifecycle Coherence (LIFE)

- [ ] **LIFE-01**: Every major lifecycle command (init, discuss, colonize, plan, build, continue, seal, entomb, publish, update) has a documented contract specifying inputs, outputs, state mutations, and exit conditions
- [ ] **LIFE-02**: A command catalog scan verifies all 317 Cobra commands produce structured output (help text, error codes, state artifacts where applicable)
- [ ] **LIFE-03**: No command produces dead-end artifacts that are never consumed by later commands or user-facing output

### Worker Economy (WORK)

- [x] **WORK-01**: Every spawned worker caste has a documented purpose, expected durable output, and downstream consumer
- [x] **WORK-02**: No worker type is spawned that only reads and returns chat without persisting findings, state, or artifacts
- [x] **WORK-03**: Build/continue/seal/colonize/plan wave shapes are documented and each spawn is justified

### Platform Parity (PLAT)

- [ ] **PLAT-01**: Go runtime behavior, YAML definitions, Claude wrappers, OpenCode wrappers, and Codex command-guide output agree on command names, flags, and behavior descriptions
- [ ] **PLAT-02**: Existing parity tests (source-check, command_parity_test, command_source_hygiene_test) are extended to close 3 known gaps (command-guide alignment, wrapper contract fields, Codex coverage)
- [ ] **PLAT-03**: No platform wrapper describes behavior the runtime does not support

### Visual Ceremony (VIZ)

- [x] **VIZ-01**: Caste colors, stage markers, live worker stacking, and closeout banners reflect real runtime state
- [x] **VIZ-02**: No decorative output exists that hides missing behavior or pretends a state transition happened when it didn't

### Data Wiring (DATA)

- [ ] **DATA-01**: Every artifact in .aether/data/ (COLONY_STATE, pheromones, midden, instincts, session, handoffs, review ledgers) is traced to a downstream consumer or explicitly documented as write-only-for-async
- [ ] **DATA-02**: QUEEN.md, Hive Brain, and graph/survey artifacts are wired into colony-prime context injection or explicitly pruned
- [ ] **DATA-03**: Review ledgers accumulate across phases and survive session resets

### Release Integrity (REL)

- [ ] **REL-01**: Version bumping, binary publishing, hub sync, npm metadata, install/update behavior, and stale-file cleanup operate as one verified coherent system
- [ ] **REL-02**: Published user experience matches the source checkout (no stale hub files, no version mismatches)

### Test Contracts (TEST)

- [ ] **TEST-01**: Structural snapshot tests freeze verified command contracts so future drift fails loudly
- [ ] **TEST-02**: Regression test suite covers command contracts, wrapper parity, output modes, worker guardrails, data flow, and publish/update behavior
- [ ] **TEST-03**: `go test ./...`, `go vet ./...`, and source/wrapper checks pass consistently

## Out of Scope

| Feature | Reason |
|---------|--------|
| New user-facing features | This is a hardening audit, not feature development |
| Performance optimization | Not the goal; audit is about correctness and coherence |
| Cross-colony ledger sharing | Findings contain code-specific paths that go stale across repos |
| Auto-block on critical findings | Conflicts with existing continue-review blocking |
| Web UI for audit results | CLI-only for now |
| Full security audit | Security-specific findings are in scope, but this is not a dedicated security audit |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| LIFE-01 | Phase 100 | Pending |
| LIFE-02 | Phase 100 | Pending |
| LIFE-03 | Phase 103 | Pending |
| WORK-01 | Phase 102 | Complete |
| WORK-02 | Phase 102 | Complete |
| WORK-03 | Phase 102 | Complete |
| PLAT-01 | Phase 101 | Pending |
| PLAT-02 | Phase 101 | Pending |
| PLAT-03 | Phase 101 | Pending |
| VIZ-01 | Phase 102 | Complete |
| VIZ-02 | Phase 102 | Complete |
| DATA-01 | Phase 103 | Pending |
| DATA-02 | Phase 103 | Pending |
| DATA-03 | Phase 104 | Pending |
| REL-01 | Phase 104 | Pending |
| REL-02 | Phase 104 | Pending |
| TEST-01 | Phase 104 | Pending |
| TEST-02 | Phase 104 | Pending |
| TEST-03 | Phase 105 | Pending |

**Coverage:**
- v1.15 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0

---
*Requirements defined: 2026-05-07*
*Last updated: 2026-05-07 after roadmap creation*
