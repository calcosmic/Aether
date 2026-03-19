# Roadmap: Aether v1.3 Maintenance & Pheromone Integration

## Overview

This milestone takes Aether from a system where pheromone signals are stored but ignored, state files are polluted with test data, and documentation describes aspirational behavior -- to a system where signals actually change worker behavior, state is clean and verifiable, the learning pipeline works end-to-end, and docs match reality. The dependency chain is strict: clean before integrating, integrate before documenting. Every phase builds on the last.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Data Purge** - Remove all test artifacts from colony state files to establish a clean baseline (completed 2026-03-19)
- [x] **Phase 2: Command Audit & Data Tooling** - Audit all 37 slash commands, fix broken ones, and build the data-clean utility (completed 2026-03-19)
- [x] **Phase 3: Pheromone Signal Plumbing** - Verify the injection chain end-to-end, fix signal lifecycle and decay, ensure session persistence (completed 2026-03-19)
- [x] **Phase 4: Pheromone Worker Integration** - Update agent definitions to act on signals, verify auto-emit influences builds, wire midden threshold (completed 2026-03-19)
- [x] **Phase 5: Learning Pipeline Validation** - Validate the observation-to-instinct pipeline end-to-end with real data (completed 2026-03-19)
- [x] **Phase 6: XML Exchange Activation** - Wire the existing XML exchange system into commands and lifecycle hooks (completed 2026-03-19)
- [x] **Phase 7: Fresh Install Hardening** - Smoke test the full install-to-build flow and validate pre-publish artifact rejection (completed 2026-03-19)
- [ ] **Phase 8: Documentation Update** - Update all documentation to match verified behavior

## Phase Details

### Phase 1: Data Purge
**Goal**: Colony state files contain only real, meaningful data -- no test artifacts polluting signals, wisdom, or observations
**Depends on**: Nothing (first phase)
**Requirements**: DATA-01, DATA-02, DATA-03, DATA-04, DATA-05, DATA-06
**Success Criteria** (what must be TRUE):
  1. QUEEN.md contains only the 5 canonical seed entries -- all 25+ test entries are gone
  2. pheromones.json contains zero test signals (no "test signal", "demo focus", "sanity signal", or "test area" entries)
  3. constraints.json focus array contains no test entries
  4. COLONY_STATE.json has no stale goal from a different project
  5. learning-observations.json contains zero synthetic test data entries; spawn-tree.txt and midden.json are clean
**Plans:** 2/2 plans complete

Plans:
- [x] 01-01-PLAN.md -- Purge test data from QUEEN.md, pheromones.json, constraints.json, and COLONY_STATE.json
- [x] 01-02-PLAN.md -- Purge test data from learning-observations.json, spawn-tree.txt, and midden.json

### Phase 2: Command Audit & Data Tooling
**Goal**: Every slash command is verified correct and a data-clean command exists for ongoing artifact removal
**Depends on**: Phase 1
**Requirements**: DATA-07, INST-02, INST-03
**Success Criteria** (what must be TRUE):
  1. All 37 slash commands have been audited for reference correctness and their status documented (pass/fail/warning)
  2. Every command with broken references or structural issues has been fixed
  3. Running `/ant:data-clean` shows artifacts found, prompts for confirmation, and safely removes them
**Plans:** 2/2 plans complete

Plans:
- [ ] 02-01-PLAN.md -- Audit all 37 slash commands for reference correctness and fix broken ones
- [ ] 02-02-PLAN.md -- Build the /ant:data-clean command (subcommand + slash command + help listing)

### Phase 3: Pheromone Signal Plumbing
**Goal**: Pheromone signals flow correctly from user input through colony-prime to worker spawn context, with working lifecycle management
**Depends on**: Phase 1
**Requirements**: PHER-01, PHER-02, PHER-06, PHER-07
**Success Criteria** (what must be TRUE):
  1. A signal emitted via `/ant:focus` appears in the prompt_section output of colony-prime and is present in the worker spawn context
  2. Signals expire correctly at phase_end events, time-based decay reduces signal strength over time, and expired signals are garbage collected
  3. Pheromone decay math produces correct results for known timestamps and edge cases (zero time elapsed, exactly at expiry, past expiry)
  4. Pheromones survive `/clear` and are available when the user runs `/ant:resume` in a new session
**Plans:** 2/3 plans complete

Plans:
- [ ] 03-01-PLAN.md -- Unify dual epoch conversion functions and write decay math edge case tests
- [ ] 03-02-PLAN.md -- Fix /ant:resume to read pheromones.json instead of constraints.json
- [ ] 03-03-PLAN.md -- End-to-end injection chain and signal lifecycle integration tests

### Phase 4: Pheromone Worker Integration
**Goal**: Workers actually read and respond to pheromone signals -- signals change what workers do, not just what gets stored
**Depends on**: Phase 3
**Requirements**: PHER-03, PHER-04, PHER-05
**Success Criteria** (what must be TRUE):
  1. Agent definitions for builder, watcher, and scout contain explicit instructions to acknowledge and act on injected pheromone context
  2. A signal auto-emitted during one build phase demonstrably influences worker behavior in a subsequent build phase
  3. When midden failure count exceeds threshold for a pattern, an auto-REDIRECT signal is created and workers avoid that pattern in subsequent work
**Plans:** 2/2 plans complete

Plans:
- [ ] 04-01-PLAN.md -- Add pheromone_protocol section to builder, watcher, and scout agent definitions
- [ ] 04-02-PLAN.md -- Cross-phase signal influence and midden threshold auto-REDIRECT integration tests

### Phase 5: Learning Pipeline Validation
**Goal**: The observation-to-instinct learning pipeline works end-to-end with real data, and promoted instincts actually influence worker behavior
**Depends on**: Phase 1, Phase 3
**Requirements**: LRNG-01, LRNG-02, LRNG-03
**Success Criteria** (what must be TRUE):
  1. A real observation entered via memory-capture flows through learning-observe, meets the promotion threshold, triggers learning-promote-auto, and creates an instinct
  2. Promoted instincts appear in colony-prime output and are present in worker prompt context
  3. An integration test covers the full pipeline path: memory-capture through to instinct-create, using non-synthetic data
**Plans:** 2/2 plans complete

Plans:
- [ ] 05-01-PLAN.md -- End-to-end pipeline validation tests with realistic data (LRNG-01, LRNG-03)
- [ ] 05-02-PLAN.md -- Instinct influence and colony-prime verification tests (LRNG-02)

### Phase 6: XML Exchange Activation
**Goal**: The existing XML exchange system is wired into commands so colonies can export and import pheromone signals
**Depends on**: Phase 3
**Requirements**: XML-01, XML-02, XML-03
**Success Criteria** (what must be TRUE):
  1. Slash commands exist for exporting and importing signals (e.g., `/ant:export-signals` and `/ant:import-signals` or equivalent)
  2. Exporting signals from one colony and importing them into another produces working signals in the receiving colony
  3. Sealing a colony automatically exports its pheromone signals as part of the seal lifecycle
**Plans:** 2/2 plans complete

Plans:
- [ ] 06-01-PLAN.md -- Create /ant:export-signals and /ant:import-signals slash commands with OpenCode parity
- [ ] 06-02-PLAN.md -- Wire standalone pheromone export into seal lifecycle and cross-colony integration tests

### Phase 7: Fresh Install Hardening
**Goal**: A new user can install Aether and run a full colony lifecycle without hitting errors or receiving test artifacts
**Depends on**: Phase 1, Phase 2, Phase 3, Phase 4
**Requirements**: INST-01, INST-04
**Success Criteria** (what must be TRUE):
  1. The sequence lay-eggs, init, plan, build, continue runs without errors on a clean repo with no prior Aether state
  2. Running validate-package.sh before publish rejects any package that contains test artifacts in QUEEN.md, pheromones.json, or other data files
**Plans:** 2/2 plans complete

Plans:
- [ ] 07-01-PLAN.md -- Harden validate-package.sh with content inspection and fix .npmignore to exclude QUEEN.md, temp files, and CONTEXT.md
- [ ] 07-02-PLAN.md -- Create fresh install smoke test covering the full lay-eggs through continue lifecycle

### Phase 8: Documentation Update
**Goal**: All documentation accurately describes verified, working behavior -- no aspirational claims, no references to eliminated features
**Depends on**: Phase 3, Phase 4, Phase 5, Phase 6, Phase 7
**Requirements**: DOCS-01, DOCS-02, DOCS-03, DOCS-04
**Success Criteria** (what must be TRUE):
  1. CLAUDE.md contains no references to eliminated features (like runtime/) and accurately describes the current architecture
  2. Pheromone documentation describes the injection model (colony-prime injects context into worker prompts) rather than claiming workers independently read signals
  3. known-issues.md has no stale FIXED statuses and all resolved items have been removed
  4. README and user-facing docs describe only behavior that has been verified working in this milestone
**Plans:** 2 plans

Plans:
- [ ] 08-01-PLAN.md — Update CLAUDE.md to match verified codebase state and clean known-issues.md of resolved entries
- [ ] 08-02-PLAN.md — Fix pheromone docs injection model and update README, aether-colony.md, source-of-truth-map.md

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8

Note: Phases 2 and 3 both depend only on Phase 1 and could execute in parallel. Phases 5 and 6 similarly have independent dependency chains. The linear order is the default; parallelization is at the executor's discretion.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Data Purge | 2/2 | Complete    | 2026-03-19 |
| 2. Command Audit & Data Tooling | 2/2 | Complete    | 2026-03-19 |
| 3. Pheromone Signal Plumbing | 0/3 | Complete    | 2026-03-19 |
| 4. Pheromone Worker Integration | 0/2 | Complete    | 2026-03-19 |
| 5. Learning Pipeline Validation | 0/2 | Complete    | 2026-03-19 |
| 6. XML Exchange Activation | 0/2 | Complete    | 2026-03-19 |
| 7. Fresh Install Hardening | 0/2 | Complete    | 2026-03-19 |
| 8. Documentation Update | 0/2 | Not started | - |
