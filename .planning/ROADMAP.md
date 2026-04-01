# Roadmap: Aether

## Milestones

- **v1.3 Maintenance & Pheromone Integration** -- Phases 1-8 (shipped 2026-03-19)
- **v2.1 Production Hardening** -- Phases 9-16 (shipped 2026-03-24)
- **v2.2 Living Wisdom** -- Phases 17-20 (shipped 2026-03-25)
- **v2.3 Per-Caste Model Routing** -- Phases 21-24 (shipped 2026-03-27)
- **v2.4 Living Wisdom** -- Phases 25-28 (shipped 2026-03-27)
- **v2.5 Smart Init** -- Phases 29-32 (shipped 2026-03-27)
- **v2.6 Bugfix & Hardening** -- Phases 33-38 (shipped 2026-03-30)
- **v2.7 PR Workflow + Stability** -- Phases 39-44 (shipped 2026-03-31)
- **v5.4 Shell-to-Go Rewrite** -- Phases 45-51 (in progress)

## Phases

<details>
<summary>v1.3 Maintenance & Pheromone Integration (Phases 1-8) -- SHIPPED 2026-03-19</summary>

- [x] Phase 1: Data Purge (2/2 plans) -- completed 2026-03-19
- [x] Phase 2: Command Audit & Data Tooling (2/2 plans) -- completed 2026-03-19
- [x] Phase 3: Pheromone Signal Plumbing (3/3 plans) -- completed 2026-03-19
- [x] Phase 4: Pheromone Worker Integration (2/2 plans) -- completed 2026-03-19
- [x] Phase 5: Learning Pipeline Validation (2/2 plans) -- completed 2026-03-19
- [x] Phase 6: XML Exchange Activation (2/2 plans) -- completed 2026-03-19
- [x] Phase 7: Fresh Install Hardening (2/2 plans) -- completed 2026-03-19
- [x] Phase 8: Documentation Update (2/2 plans) -- completed 2026-03-19

</details>

<details>
<summary>v2.1 Production Hardening (Phases 9-16) -- SHIPPED 2026-03-24</summary>

- [x] Phase 9: Quick Wins (2/2 plans) -- completed 2026-03-24
- [x] Phase 10: Error Triage (3/3 plans) -- completed 2026-03-24
- [x] Phase 11: Dead Code Deprecation (2/2 plans) -- completed 2026-03-24
- [x] Phase 12: State API & Verification (3/3 plans) -- completed 2026-03-24
- [x] Phase 13: Monolith Modularization (9/9 plans) -- completed 2026-03-24
- [x] Phase 14: Planning Depth (2/2 plans) -- completed 2026-03-24
- [x] Phase 15: Documentation Accuracy (3/3 plans) -- completed 2026-03-24
- [x] Phase 16: Ship (2/2 plans) -- completed 2026-03-24

</details>

<details>
<summary>v2.2 Living Wisdom (Phases 17-20) -- SHIPPED 2026-03-25</summary>

- [x] Phase 17: Local Wisdom Accumulation -- completed 2026-03-24
- [x] Phase 18: Local Wisdom Injection -- completed 2026-03-25
- [x] Phase 19: Cross-Colony Hive -- completed 2026-03-25
- [x] Phase 20: Hub Wisdom Layer -- completed 2026-03-25

</details>

<details>
<summary>v2.3 Per-Caste Model Routing (Phases 21-24) -- SHIPPED 2026-03-27</summary>

- [x] Phase 21: Test Infrastructure Refactor (3 plans) -- completed 2026-03-27
- [x] Phase 22: Config Foundation & Core Routing (3 plans) -- completed 2026-03-27
- [x] Phase 23: Tooling & Overrides (2 plans) -- completed 2026-03-27
- [x] Phase 24: Safety & Verification (2 plans) -- completed 2026-03-27

</details>

<details>
<summary>v2.4 Living Wisdom (Phases 25-28) -- SHIPPED 2026-03-27</summary>

- [x] Phase 25: Agent Definitions (Oracle + Architect) -- completed 2026-03-27
- [x] Phase 26: Wisdom Pipeline Wiring -- completed 2026-03-27
- [x] Phase 27: Deterministic Fallback + Dedup -- completed 2026-03-27
- [x] Phase 28: Integration Validation -- completed 2026-03-27

</details>

<details>
<summary>v2.5 Smart Init (Phases 29-32) -- SHIPPED 2026-03-27</summary>

- [x] Phase 29: Repo Scanning Module (3/3 plans) -- completed 2026-03-27
- [x] Phase 30: Charter Management (2/2 plans) -- completed 2026-03-27
- [x] Phase 31: Init.md Smart Init Rewrite (2/2 plans) -- completed 2026-03-27
- [x] Phase 32: Intelligence Enhancements (3/3 plans) -- completed 2026-03-27

</details>

<details>
<summary>v2.6 Bugfix & Hardening (Phases 33-38) -- SHIPPED 2026-03-30</summary>

- [x] **Phase 33: Input Escaping & Atomic Write Safety** - Fix ant_name injection in grep/JSON, jq-escape all dynamic values, release locks on validation failure
- [x] **Phase 34: Cross-Colony Isolation** - Eliminate information bleed between colonies via proper name extraction, lock scoping, and file namespacing
- [x] **Phase 35: Colony Depth & Model Routing** - Depth selector gates Oracle/Scout spawns; model routing either wired end-to-end or dead code removed
- [x] **Phase 36: YAML Command Generator** - Single YAML source produces both Claude and OpenCode command markdown
- [x] **Phase 37: XML Core Integration** - XML export/import wired into seal, entomb, and init lifecycle commands
- [x] **Phase 38: Cleanup & Maintenance** - Deprecate old npm versions, generate error code docs, remove dead awk code

</details>

<details>
<summary>v2.7 PR Workflow + Stability (Phases 39-44) -- SHIPPED 2026-03-31</summary>

- [x] **Phase 39: State Safety** -- STATE-01, STATE-02
- [x] **Phase 40: Pheromone Propagation** -- PHERO-01, PHERO-02, PHERO-03
- [x] **Phase 41: Midden Collection** -- MIDD-01, MIDD-02, MIDD-03
- [x] **Phase 42: CI Context Assembly** -- CI-01, CI-02, CI-03
- [x] **Phase 42.1: Release Hygiene** -- REL-01, REL-02
- [x] **Phase 43: Clash Detection Integration** -- CLASH-01, CLASH-02, CLASH-03
- [x] **Phase 44: Release Hygiene & Ship** -- REL-01, REL-02, REL-03, TEST-01, TEST-02

</details>

### v5.4 Shell-to-Go Rewrite (In Progress)

**Milestone Goal:** Replace all shell scripts with a native Go binary, eliminating bash/jq/curl dependencies while preserving exact behavioral parity with the existing system.

- [ ] **Phase 45: Core Storage** - STOR-01, STOR-02, STOR-03
- [ ] **Phase 46: Event Bus** - EVT-01, EVT-02, EVT-03
- [ ] **Phase 47: Memory Pipeline** - MEM-01 through MEM-05
- [ ] **Phase 48: Graph Layer** - GRAPH-01 through GRAPH-04
- [ ] **Phase 49: Agent System + LLM** - AGENT-01 through AGENT-04, LLM-01 through LLM-04
- [ ] **Phase 50: CLI Commands** - CLI-01 through CLI-04
- [ ] **Phase 51: XML Exchange + Distribution + Testing** - XML-01 through XML-04, DIST-01 through DIST-03, TEST-01 through TEST-03

## Phase Details

<details>
<summary>Phase details for v1.3 through v2.7 (Phases 1-44)</summary>

### Phase 33: Input Escaping & Atomic Write Safety
**Goal**: Dynamic values flowing through grep patterns, JSON construction, and atomic writes cannot corrupt data or break commands
**Depends on**: Nothing (first phase -- fixes data-corrupting bugs)
**Requirements**: SAFE-01, SAFE-03, SAFE-04
**Success Criteria** (what must be TRUE):
  1. Running `grep -F` on ant_name values containing regex metacharacters (e.g., `worker.builder+1`) returns correct matches without errors across spawn.sh, swarm.sh, spawn-tree.sh, aether-utils.sh
  2. JSON output from all 14 identified locations in utils/ produces valid JSON even when dynamic values contain quotes, newlines, or backslashes
  3. When atomic_write encounters a JSON validation failure, the lock file is released and no stale locks remain
  4. All 616+ existing tests still pass after escaping changes
**Plans**: 4 plans
- [ ] 33-01-PLAN.md -- Fix grep pattern injection: add -F flag to all ant_name greps + escape ant_name in JSON output
- [ ] 33-02-PLAN.md -- Fix json_ok string interpolation across all utils/ modules (session, queen, learning, pheromone, etc.)
- [ ] 33-03-PLAN.md -- Audit lock release on all acquire_lock callers + harden atomic_write + stale lock auto-expiry
- [ ] 33-04-PLAN.md -- Dedicated data-safety.test.js + Data Safety section in /ant:status

### Phase 34: Cross-Colony Isolation
**Goal**: Two colonies running on the same machine cannot read or corrupt each other's state
**Depends on**: Phase 33 (escaping fixes prevent masking isolation bugs)
**Requirements**: SAFE-02
**Success Criteria** (what must be TRUE):
  1. Colony name extraction uses `_colony_name()` from queen.sh instead of fragile session_id splitting -- verified at learning.sh:378, learning.sh:930, aether-utils.sh:3655
  2. `LOCK_DIR` in hive.sh is passed as a function parameter, never mutated as a global variable
  3. Shared data files (pheromones.json, learning-observations.json, session.json, run-state.json) include colony namespace so two colonies writing concurrently do not overwrite each other
  4. Existing single-colony workflows still work identically (no regression)
**Plans:** 5/5 plans complete

Plans:
- [x] 34-01-PLAN.md -- Replace all 13 session_id splitting locations with colony-name subcommand (3 shell + 9 playbook + 1 OpenCode)
- [x] 34-02-PLAN.md -- Add acquire_lock_at/release_lock_at to file-lock.sh and refactor hive.sh to eliminate LOCK_DIR mutation
- [x] 34-03-PLAN.md -- Add COLONY_DATA_DIR resolution + auto-migration infrastructure, update aether-utils.sh file references
- [x] 34-04-PLAN.md -- Update all 15 utils/ modules to use COLONY_DATA_DIR for per-colony file references
- [x] 34-05-PLAN.md -- Integration tests for colony isolation (COLONY_DATA_DIR, migration, lock tagging, backwards compat)

### Phase 35: Colony Depth & Model Routing
**Goal**: Colony operators can control how deeply the system investigates (gating expensive agent spawns) and model routing is either functional end-to-end or honestly removed
**Depends on**: Phase 34 (colony state changes in 34 affect COLONY_STATE.json which depth selector also modifies)
**Requirements**: INFRA-01, INFRA-02
**Success Criteria** (what must be TRUE):
  1. `colony_depth` field exists in COLONY_STATE.json with values light/standard/deep/full, defaulting to standard
  2. Oracle spawns in build-wave.md are gated by colony depth (only spawn at deep/full), Scout spawns respect depth setting
  3. Model routing either passes the resolved model slot to the actual agent spawn call, or all model routing code (model-profiles.yaml, caste table, model-slot subcommand) is removed with a documented decision -- no dead code left in between
  4. `/ant:status` or colony dashboard displays the active depth setting
**Plans**: TBD

### Phase 36: YAML Command Generator
**Goal**: A single set of YAML source files produces both Claude Code and OpenCode command markdown, eliminating manual duplication of 44 commands
**Depends on**: Phase 33 (no hard dependency, but safety fixes first)
**Requirements**: INFRA-03
**Success Criteria** (what must be TRUE):
  1. YAML source files exist for each command, containing the canonical command spec
  2. Running the generator script produces .claude/commands/ant/*.md and .opencode/commands/ant/*.md from YAML sources
  3. Generated output matches (or improves upon) the current hand-written command files -- no loss of functionality
  4. `npm run lint:sync` validates that generated files are up-to-date with YAML sources
**Plans**: 4 plans

Plans:
- [x] 36-01-PLAN.md -- Generator engine (bin/generate-commands.js) + unit tests
- [ ] 36-02-PLAN.md -- Convert 22 simpler commands to YAML source format
- [x] 36-03-PLAN.md -- Convert 22 complex commands to YAML (including build.md, continue.md)
- [ ] 36-04-PLAN.md -- Update sync tooling (generate-commands.sh) + npm scripts + full validation

### Phase 37: XML Core Integration
**Goal**: XML export/import is wired into colony lifecycle commands so cross-colony data transfer happens automatically at key moments
**Depends on**: Phase 34 (colony isolation must be solid before auto-exporting colony data)
**Requirements**: INFRA-04
**Success Criteria** (what must be TRUE):
  1. `/ant:seal` automatically exports pheromone signals and wisdom to XML as part of the seal process
  2. `/ant:entomb` archives XML exchange files alongside the colony chamber
  3. `/ant:init` can import XML files from a previous colony to seed a new one (opt-in, not automatic)
  4. XML files in .aether/exchange/ are included in `validate-package.sh` distribution checks
**Plans**: TBD

### Phase 38: Cleanup & Maintenance
**Goal**: Registry housekeeping, developer documentation, and dead code removal -- small items that don't warrant their own phase
**Depends on**: Phase 33 (MAINT-02 depends on error-handler.sh being stable after escaping fixes; MAINT-03 depends on no new awk usage introduced by earlier phases)
**Requirements**: MAINT-01, MAINT-02, MAINT-03
**Success Criteria** (what must be TRUE):
  1. Old 2.x npm versions are marked deprecated on the registry with a message pointing to current version
  2. An error code reference document exists in .aether/docs/ listing all error codes from error-handler.sh with descriptions, and is included in npm distribution
  3. The unused `models[]` awk array is removed from spawn-tree.sh with no test regressions
**Plans**: 2 plans

Plans:
- [x] 38-01-PLAN.md -- Remove dead models[] awk array from spawn-tree.sh + audit error-codes.md completeness
- [ ] 38-02-PLAN.md -- Deprecate old npm versions + fix dist-tag + align package.json version

### Phase 39: State Safety
**Goal**: All COLONY_STATE.json writes use atomic mutations and test suite passes on clean/empty state
**Depends on**: Nothing (foundation for all subsequent phases)
**Requirements**: STATE-01, STATE-02
**Success Criteria** (what must be TRUE):
  1. `grep -rn 'jq "\(.*\)" ' .aether/ --include='*.sh' | grep COLONY_STATE` returns zero results -- no raw jq writes to state file
  2. Every COLONY_STATE.json write path goes through `state-mutate` with atomic file locking
  3. `npm test` passes with zero failures when COLONY_STATE.json contains minimal valid state
  4. State validation tests handle missing optional fields gracefully (no hard failure on empty colony)
**Existing work**: state-mutate subcommand exists in state-api.sh; 4 pheromone subcommands already migrated; design doc at `.aether/docs/state-contract-design.md`
**Plans:** 2 plans

Plans:
- [ ] 39-01-PLAN.md -- Stash protection: add pathspec exclusion to 3 stash entry points (swarm.sh, build-prep.md, build-full.md)
- [ ] 39-02-PLAN.md -- State migration + test fixes: migrate queen.sh to _state_mutate, reset COLONY_STATE.json, fix 11 failing tests

### Phase 40: Pheromone Propagation
**Goal**: Pheromone signals flow across git branches -- signals from main reach worktrees, and branch-specific signals merge back after PR
**Depends on**: Phase 39 (state safety must be solid before adding cross-branch writes)
**Requirements**: PHERO-01, PHERO-02, PHERO-03
**Success Criteria** (what must be TRUE):
  1. Creating a worktree branch via `_worktree_create` automatically copies active main-branch pheromones into the branch
  2. `pheromone-snapshot-inject` produces a valid snapshot JSON that can be read by the branch's pheromone system
  3. `pheromone-merge-back` merges user-created branch signals into main without duplicating existing signals
  4. Merge conflict resolution follows priority: REDIRECT > FOCUS > FEEDBACK, with strength-based dedup

### Phase 41: Midden Collection
**Goal**: Failure records from merged branches are collected into main's midden with idempotency and cross-PR pattern detection
**Depends on**: Phase 40 (pheromone propagation pattern informs midden flow)
**Requirements**: MIDD-01, MIDD-02, MIDD-03
**Success Criteria** (what must be TRUE):
  1. `midden-collect --branch <branch> --merge-sha <sha>` ingests failure records from the branch into main's midden
  2. Running midden-collect twice with the same merge SHA produces no duplicates (idempotent)
  3. `midden-handle-revert --sha <sha>` tags affected entries rather than deleting them
  4. `midden-cross-pr-analysis` returns failure patterns detected across 2+ PRs

### Phase 42: CI Context Assembly
**Goal**: CI agents get machine-readable colony context via `pr-context` subcommand, replacing interactive colony-prime for automated workflows
**Depends on**: Phase 41 (needs midden data for complete context)
**Requirements**: CI-01, CI-02, CI-03
**Success Criteria** (what must be TRUE):
  1. `aether pr-context` outputs valid JSON with sections: colony_state, pheromones, phase_context, blockers, hive_wisdom
  2. When a source file is missing or corrupt, pr-context returns partial data with the missing section marked as `null` -- never hard-fails
  3. Normal mode output stays under 6,000 characters; compact mode under 3,000 characters
  4. Token budget trimming follows the same priority order as colony-prime (rolling summary first, blockers never)
**Plans:** 2 plans

Plans:
- [ ] 42-01-PLAN.md -- Extract _budget_enforce(), implement pr-context with all sections, cache, midden, tests
- [ ] 42-02-PLAN.md -- Wire pr-context into /ant:continue and /ant:run playbooks

### Phase 42.1: Release hygiene (INSERTED)
**Goal:** Fix version drift in CLAUDE.md, expand validate-package.sh coverage to all packaged utils, regenerate stale YAML-generated commands, and correct inaccurate documentation counts.
**Requirements**: REL-01, REL-02
**Depends on:** Phase 42
**Plans:** 2/2 plans complete

Plans:
- [x] 42.1-01-PLAN.md -- Expand validate-package.sh REQUIRED_FILES to all 35 utils + regenerate stale commands
- [ ] 42.1-02-PLAN.md -- Fix CLAUDE.md version drift and stale documentation counts

### Phase 43: Clash Detection Integration
**Goal**: Task-as-PR workflow prevents file conflicts between parallel worktrees via hooks and automatic context setup
**Depends on**: Phase 40 (worktree creation needs pheromone injection)
**Requirements**: CLASH-01, CLASH-02, CLASH-03
**Success Criteria** (what must be TRUE):
  1. Editing a file that is modified in another active worktree triggers a PreToolUse hook that blocks the edit with a clear message
  2. `_worktree_create` automatically copies colony context (COLONY_STATE.json, pheromones.json) and runs pheromone-snapshot-inject
  3. `.gitattributes` merge driver resolves package-lock.json conflicts by keeping "ours"
  4. `.aether/data/` files are on the allowlist -- never trigger clash detection (branch-local state)
**Plans:** 2/2 plans complete

Plans:
- [x] 43-01-PLAN.md -- Wire clash-detect.sh and worktree.sh into aether-utils.sh dispatcher
- [ ] 43-02-PLAN.md -- Wire clash detection and merge driver setup into /ant:init

### Phase 44: Release Hygiene & Ship
**Goal**: Published package is clean of dev artifacts, all tests pass, and v2.7.0 ships to npm
**Depends on**: Phases 39-43 (all features must be complete)
**Requirements**: REL-01, REL-02, REL-03, TEST-01, TEST-02
**Success Criteria** (what must be TRUE):
  1. `npm pack --dry-run` output contains no test data, worktree references, colony state, or dev artifacts
  2. `bin/validate-package.sh` passes with zero warnings
  3. `npm test` shows 620+ passing tests with zero failures
  4. `npm install -g . && aether --version` succeeds on a clean machine
  5. CLAUDE.md updated with v2.7 changes, version bumped to v2.7.0

</details>

### Phase 45: Core Storage
**Goal**: Go reads and writes all existing colony data files with identical behavior to the shell implementation -- no data loss, no partial writes, no format drift
**Depends on**: Nothing (foundation -- every subsequent phase reads/writes through this layer)
**Requirements**: STOR-01, STOR-02, STOR-03
**Success Criteria** (what must be TRUE):
  1. Go reads every existing JSON file (COLONY_STATE, pheromones, learnings, instincts, flags, constraints, midden) and produces structurally identical output when re-serialized -- round-trip tests prove byte-level parity
  2. Killing the process mid-write never leaves a partial or corrupt JSON file on disk -- temp+rename atomic writes match the shell `atomic_write` contract
  3. JSONL append and read operations produce the same line format as shell -- blank lines are skipped, malformed lines are logged and skipped
  4. A Go test suite of 20+ cases covers normal reads, missing files, corrupt JSON, concurrent writes, and large files
**Plans:** 2 plans

Plans:
- [ ] 45-01-PLAN.md -- Define typed structs for all 6 data files (pheromones, learning-observations, midden, flags, constraints, session), fix ColonyState gaps, golden file parity tests (STOR-01)
- [ ] 45-02-PLAN.md -- Storage infrastructure: backup rotation, AETHER_ROOT path resolution, JSONL malformed line handling (STOR-02, STOR-03)

### Phase 46: Event Bus
**Goal**: Typed events flow through the system via Go channels with crash-recoverable persistence -- replacing the shell's file-based pub/sub
**Depends on**: Phase 45 (event persistence uses JSONL storage layer)
**Requirements**: EVT-01, EVT-02, EVT-03
**Success Criteria** (what must be TRUE):
  1. Publishers emit typed events to named channels and all active subscribers receive them in order with sub-microsecond latency
  2. After a simulated crash (process kill), the bus replays persisted events from JSONL so no events are lost
  3. Events with expired TTLs are pruned on load and on schedule -- behavior matches the shell `event-bus-publish` TTL pruning
  4. Channel subscribe/unsubscribe works without blocking publishers -- goroutines clean up on unsubscribe
**Plans**: TBD

### Phase 47: Memory Pipeline
**Goal**: Observations flow through trust scoring, auto-promote to instincts, and high-confidence instincts promote to QUEEN.md -- identical outcomes to the shell wisdom pipeline
**Depends on**: Phase 46 (observations and promotions flow as events)
**Requirements**: MEM-01, MEM-02, MEM-03, MEM-04, MEM-05
**Success Criteria** (what must be TRUE):
  1. Trust scores calculated by Go produce the same numeric result as the shell ADR-002 algorithm (40/35/25 weighted, 60-day half-life, 7 tiers) for identical inputs
  2. An observation is auto-promoted to an instinct when trust reaches 0.50 or 3+ similar patterns are captured -- matches shell `learning-promote-auto` thresholds
  3. Promoted instincts include full provenance (source observation IDs, timestamp, confidence) and graph edges link source to instinct
  4. QUEEN.md sections written by Go match the existing 4-section template format -- no structural changes to the file
  5. Phase-end consolidation runs trust decay, archives entries below threshold, and checks promotion eligibility -- output matches shell consolidation
**Plans**: TBD

### Phase 48: Graph Layer
**Goal**: A directed graph tracks relationships between learnings, instincts, phases, and colonies -- queryable via BFS and cycle detection
**Depends on**: Phase 47 (graph edges are created during instinct promotion)
**Requirements**: GRAPH-01, GRAPH-02, GRAPH-03, GRAPH-04
**Success Criteria** (what must be TRUE):
  1. Nodes of all 5 types (learning, instinct, queen, phase, colony) and all 16 edge types can be added, queried, and removed from the in-memory directed graph
  2. 1-hop and 2-hop neighbor queries return the same set of connected nodes as the jq graph layer for identical input data
  3. Shortest path (BFS) finds the minimum-hop route between two nodes; cycle detection identifies all cycles in the graph
  4. Graph serializes to JSON and deserializes back without data loss -- round-trip parity verified against shell-produced graph JSON
**Plans**: TBD

### Phase 49: Agent System + LLM
**Goal**: Go agents run in goroutine pools with Anthropic LLM calls, replacing shell subprocess spawning and enabling tool-use loops
**Depends on**: Phase 46 (agents subscribe to events), Phase 47 (curation ants handle memory events)
**Requirements**: AGENT-01, AGENT-02, AGENT-03, AGENT-04, LLM-01, LLM-02, LLM-03, LLM-04
**Success Criteria** (what must be TRUE):
  1. All agents implement a common interface (Name, Caste, Triggers, Execute) -- adding a new agent requires only implementing the interface and registering it
  2. Worker pool runs multiple agents concurrently with bounded goroutines -- pool limit prevents resource exhaustion under load
  3. Spawn tracking records all running agents in a tree structure matching the shell spawn-tree.txt format
  4. The 8 curation ants subscribe to typed memory events and execute their handlers when events arrive -- matches shell ant orchestrator behavior
  5. Anthropic SDK sends a message and receives a valid response with correct role, content, and stop reason
  6. Streaming responses accumulate SSE chunks into a complete message -- caller receives the same result as a non-streaming call
  7. Tool use loop detects tool call blocks, executes the requested tool, and returns the result -- completes when the model returns a text-only response
  8. Agent YAML frontmatter (model, tools, triggers) parses into Go structs that configure agent behavior
**Plans**: TBD

### Phase 50: CLI Commands
**Goal**: All 37 colony commands are accessible via a Go binary with Cobra, producing output identical to the shell commands
**Depends on**: Phase 45 (all commands read/write via storage layer), Phase 47 (memory commands), Phase 49 (agent commands)
**Requirements**: CLI-01, CLI-02, CLI-03, CLI-04
**Success Criteria** (what must be TRUE):
  1. Running `aether init`, `aether build`, `aether status`, and all 34 other subcommands from the Go binary produces the same exit code and side effects as the shell equivalents
  2. Shell completion scripts for bash, zsh, and fish generate from the Cobra command tree -- tab completion works identically to shell command completion
  3. `aether status` output matches the shell `/ant:status` dashboard character-for-character in structure and data
  4. All read-only commands (status, phase, flags, history, pheromones, memory-details) produce byte-identical output to their shell counterparts for the same input data
**Plans**: TBD
**UI hint**: yes

### Phase 51: XML Exchange + Distribution + Testing
**Goal**: XML export/import, cross-platform binary distribution, and full test parity complete the Go rewrite -- the binary replaces npm
**Depends on**: Phase 50 (CLI commands), Phase 48 (graph layer used by XML exchange)
**Requirements**: XML-01, XML-02, XML-03, XML-04, DIST-01, DIST-02, DIST-03, TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. Pheromone export/import produces valid XML that validates against existing XSD schemas and round-trips without data loss
  2. Wisdom export/import respects confidence threshold filtering -- only entries above the threshold appear in XML output
  3. Registry export/import includes lineage tracking up to depth 10 -- matches shell behavior
  4. `go install github.com/aether-colony/aether@latest` produces a working binary on a clean machine
  5. Cross-compiled binaries for linux/darwin/windows on amd64/arm64 all pass their test suites
  6. `aether --version` reports a version string matching the npm package version
  7. All existing shell test cases have Go equivalents that pass -- parity verified against shell-produced output fixtures
  8. `go test -race ./...` passes with zero race conditions detected
  9. Golden file tests compare Go JSON output against shell-produced fixtures -- byte-identical or documented difference
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> ... -> 44 -> 45 -> 46 -> 47 -> 48 -> 49 -> 50 -> 51

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Data Purge | v1.3 | 2/2 | Complete | 2026-03-19 |
| 2. Command Audit & Data Tooling | v1.3 | 2/2 | Complete | 2026-03-19 |
| 3. Pheromone Signal Plumbing | v1.3 | 3/3 | Complete | 2026-03-19 |
| 4. Pheromone Worker Integration | v1.3 | 2/2 | Complete | 2026-03-19 |
| 5. Learning Pipeline Validation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 6. XML Exchange Activation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 7. Fresh Install Hardening | v1.3 | 2/2 | Complete | 2026-03-19 |
| 8. Documentation Update | v1.3 | 2/2 | Complete | 2026-03-19 |
| 9. Quick Wins | v2.1 | 2/2 | Complete | 2026-03-24 |
| 10. Error Triage | v2.1 | 3/3 | Complete | 2026-03-24 |
| 11. Dead Code Deprecation | v2.1 | 2/2 | Complete | 2026-03-24 |
| 12. State API & Verification | v2.1 | 3/3 | Complete | 2026-03-24 |
| 13. Monolith Modularization | v2.1 | 9/9 | Complete | 2026-03-24 |
| 14. Planning Depth | v2.1 | 2/2 | Complete | 2026-03-24 |
| 15. Documentation Accuracy | v2.1 | 3/3 | Complete | 2026-03-24 |
| 16. Ship | v2.1 | 2/2 | Complete | 2026-03-24 |
| 17. Local Wisdom Accumulation | v2.2 | - | Complete | 2026-03-24 |
| 18. Local Wisdom Injection | v2.2 | - | Complete | 2026-03-25 |
| 19. Cross-Colony Hive | v2.2 | - | Complete | 2026-03-25 |
| 20. Hub Wisdom Layer | v2.2 | - | Complete | 2026-03-25 |
| 21. Test Infrastructure Refactor | v2.3 | - | Complete | 2026-03-27 |
| 22. Config Foundation & Core Routing | v2.3 | - | Complete | 2026-03-27 |
| 23. Tooling & Overrides | v2.3 | - | Complete | 2026-03-27 |
| 24. Safety & Verification | v2.3 | 2/2 | Complete | 2026-03-27 |
| 25. Agent Definitions | v2.4 | - | Complete | 2026-03-27 |
| 26. Wisdom Pipeline Wiring | v2.4 | - | Complete | 2026-03-27 |
| 27. Deterministic Fallback + Dedup | v2.4 | - | Complete | 2026-03-27 |
| 28. Integration Validation | v2.4 | - | Complete | 2026-03-27 |
| 29. Repo Scanning Module | v2.5 | 3/3 | Complete | 2026-03-27 |
| 30. Charter Management | v2.5 | 2/2 | Complete | 2026-03-27 |
| 31. Init Smart Init Rewrite | v2.5 | 2/2 | Complete | 2026-03-27 |
| 32. Intelligence Enhancements | v2.5 | 3/3 | Complete | 2026-03-27 |
| 33. Input Escaping | v2.6 | - | Complete | 2026-03-29 |
| 34. Cross-Colony Isolation | v2.6 | - | Complete | 2026-03-29 |
| 35. Colony Depth & Model Routing | v2.6 | - | Complete | 2026-03-29 |
| 36. YAML Command Generator | v2.6 | - | Complete | 2026-03-29 |
| 37. XML Core Integration | v2.6 | - | Complete | 2026-03-29 |
| 38. Cleanup & Maintenance | v2.6 | - | Complete | 2026-03-29 |
| 39. State Safety | v2.7 | - | Complete | 2026-03-31 |
| 40. Pheromone Propagation | v2.7 | - | Complete | 2026-03-31 |
| 41. Midden Collection | v2.7 | - | Complete | 2026-03-31 |
| 42. CI Context Assembly | v2.7 | - | Complete | 2026-03-31 |
| 42.1. Release Hygiene | v2.7 | - | Complete | 2026-03-31 |
| 43. Clash Detection | v2.7 | - | Complete | 2026-03-31 |
| 44. Release Hygiene & Ship | v2.7 | - | Complete | 2026-03-31 |
| 45. Core Storage | v5.4 | 0/2 | Not started | - |
| 46. Event Bus | v5.4 | 0/TBD | Not started | - |
| 47. Memory Pipeline | v5.4 | 0/TBD | Not started | - |
| 48. Graph Layer | v5.4 | 0/TBD | Not started | - |
| 49. Agent System + LLM | v5.4 | 0/TBD | Not started | - |
| 50. CLI Commands | v5.4 | 0/TBD | Not started | - |
| 51. XML Exchange + Dist + Testing | v5.4 | 0/TBD | Not started | - |
