# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (completed 2026-04-24)
- **v1.7 Planning Pipeline Recovery** - Phases 47-48 (completed 2026-04-24)
- **v1.8 Colony Recovery** - Phases 49-51 (shipped 2026-04-25)
- **v1.9 Review Persistence** - Phases 52-56 (in progress)

## Phases

<details>
<summary>v1.0 MVP (Phases 1-6) -- SHIPPED</summary>

- Phase 1: Housekeeping and Foundation
- Phase 2: Colony Scope System
- Phase 3: Restore Build Ceremony
- Phase 4: Restore Continue Ceremony
- Phase 5: Living Watch and Status Surfaces
- Phase 6: Pheromone Visibility and Steering

</details>

<details>
<summary>v1.1 Trusted Context (Phases 7-11) -- SHIPPED</summary>

- Phase 7: Context Ledger and Skill Routing Foundation
- Phase 8: Prompt Integrity and Trust Boundaries
- Phase 9: Trust-Weighted Context Assembly
- Phase 10: Curation Spine and Structural Learning
- Phase 11: Competitive Proof Surfaces and Evaluation

</details>

<details>
<summary>v1.2 Live Dispatch Truth and Recovery (Phases 12-16) -- SHIPPED</summary>

- Phase 12: Dispatch Truth Model and Run Scoping
- Phase 13: Live Workflow Visibility Across Colonize, Plan, and Build
- Phase 14: Worker Execution Robustness and Honest Activity Tracking
- Phase 15: Verification-Led Continue and Partial Success
- Phase 16: Recovery, Reconciliation, and Runtime UX Finalization

</details>

<details>
<summary>v1.3 Visual Truth and Core Hardening (Phases 17-24) -- SHIPPED 2026-04-21</summary>

- Phase 17: Slash Command Format Audit
- Phase 18: Visual UX Restoration -- Caste Identity and Spawn Lists
- Phase 19: Visual UX Restoration -- Stage Separators and Ceremony
- Phase 20: Visual UX Restoration -- Emoji Consistency
- Phase 21: Codex CLI Visual Parity
- Phase 22: Core Path Hardening
- Phase 23: Recovery and Continuity
- Phase 24: Full Instrumentation -- Trace Logging

</details>

<details>
<summary>v1.4 Self-Healing Colony (Phases 25-30) -- COMPLETED 2026-04-21</summary>

- Phase 25: Medic Ant Core -- Health diagnosis command, colony data scanner
- Phase 26: Auto-Repair -- Fix common colony data issues with `--fix` flag
- Phase 27: Medic Skill -- Healthy state specification skill file
- Phase 28: Ceremony Integrity -- Verify wrapper/runtime parity
- Phase 29: Trace Diagnostics -- Remote debugging via trace export analysis
- Phase 30: Medic Worker Integration -- Caste integration, auto-spawn

</details>

<details>
<summary>v1.5 Runtime Truth Recovery (Phases 31-38) -- COMPLETED 2026-04-23</summary>

8 phases, 17 plans, 176 commits. P0 runtime truth fixes, continue unblock, dispatch robustness, cleanup, platform parity, v1.0.20 release, codebase hygiene, Nyquist validation. [Full archive -> milestones/v1.5-ROADMAP.md]

</details>

<details>
<summary>v1.6 Release Pipeline Integrity (Phases 39-46) -- COMPLETED 2026-04-24</summary>

8 phases (including inserted 44.1, 44.2), 10 plans. OpenCode agent frontmatter fix, stable publish hardening, dev channel isolation, stale publish detection, release integrity checks, doc alignment, E2E regression coverage, stuck-plan investigation. [Full archive -> milestones/v1.6-ROADMAP.md]

</details>

<details>
<summary>v1.7 Planning Pipeline Recovery (Phases 47-48) -- COMPLETED 2026-04-24</summary>

2 phases. Plan --force recovery, fallback artifact cleanup, scout timeout, E2E recovery test. [Full archive -> milestones/v1.7-ROADMAP.md]

</details>

<details>
<summary>v1.8 Colony Recovery (Phases 49-51) -- SHIPPED 2026-04-25</summary>

3 phases, 6 plans, 17 requirements. Stuck-state scanner (7 detectors), auto-repair pipeline (backup-first, atomic rollback), 10 E2E tests. [Full archive -> milestones/v1.8-ROADMAP.md]

</details>

### v1.9 Review Persistence (In Progress)

**Milestone Goal:** Review agent findings survive `/clear` and accumulate across phases so downstream workers can learn from prior reviews.

- [x] **Phase 52: Continue-Review Worker Outcome Reports** - Per-worker .md reports for review workers, mirroring build report pattern (completed 2026-04-26)
- [x] **Phase 53: Domain-Ledger CRUD Subcommands** - Four CLI subcommands for structured review finding persistence across 7 domains (completed 2026-04-26)
- [x] **Phase 54: Colony-Prime Prior-Reviews Section** - Open review findings injected into downstream worker context (completed 2026-04-26)
- [ ] **Phase 55: Agent Definition Updates** - Seven review agents gain Write tool, findings instructions, and guardrails
- [ ] **Phase 56: Lifecycle Integration** - Seal, entomb, status, and init updated for review ledger lifecycle

## Phase Details

### Phase 52: Continue-Review Worker Outcome Reports
**Goal**: Continue-review workers produce per-worker outcome reports on disk, closing the asymmetry with build workers
**Depends on**: Nothing (first phase of v1.9)
**Requirements**: CONT-01, CONT-02, CONT-03, CONT-04, CONT-05, CONT-06
**Success Criteria** (what must be TRUE):
  1. After `/ant-continue`, each review worker (Watcher, Gatekeeper, Auditor, Probe) has a `.md` file at `build/phase-N/worker-reports/{name}.md` containing its full findings
  2. The `codexContinueWorkerFlowStep` struct carries `Blockers`, `Duration`, and `Report` fields through the continue pipeline without data loss
  3. Claude and OpenCode wrappers pass the `report` field in their completion packet and the runtime preserves it in the merged result
  4. Old completion packets that lack `report`, `blockers`, or `duration` fields still work correctly with no errors
**Plans**: 2 plans

Plans:
- [ ] 52-01-PLAN.md -- Go runtime: struct fields, merge propagation, report writing, Codex-native path, tests
- [ ] 52-02-PLAN.md -- Wrapper docs: report field in Claude and OpenCode continue.md completion packets

### Phase 53: Domain-Ledger CRUD Subcommands
**Goal**: Structured review findings persist across phases in 7 domain-specific ledgers, queryable and resolvable via CLI
**Depends on**: Phase 52 (struct fields from continue pipeline)
**Requirements**: LEDG-01, LEDG-02, LEDG-03, LEDG-04, LEDG-05, LEDG-06, LEDG-07, LEDG-08, LEDG-09, LEDG-10
**Success Criteria** (what must be TRUE):
  1. `aether review-ledger-write --domain security --phase 2 --findings '<json>'` creates `reviews/security/ledger.json` with deterministic IDs like `sec-2-001` and a computed summary
  2. `aether review-ledger-read --domain quality --status open` returns only open quality findings, filterable by phase
  3. `aether review-ledger-summary` prints one line per domain showing total, open, and severity breakdowns
  4. `aether review-ledger-resolve --domain security --id sec-2-001` marks the entry resolved with a timestamp
  5. All 7 domain directories exist under `.aether/data/reviews/` and writes use file-locking atomic writes from `pkg/storage/`
**Plans**: 2 plans

Plans:
- [x] 53-01-PLAN.md -- Data types: ReviewLedgerEntry, ReviewLedgerFile, ReviewLedgerSummary in pkg/colony/ with unit tests (completed 2026-04-26)
- [x] 53-02-PLAN.md -- CLI commands: four cobra subcommands (write, read, summary, resolve) with integration tests (completed 2026-04-26)

### Phase 54: Colony-Prime Prior-Reviews Section
**Goal**: Downstream workers see open review findings from prior phases in their context, so review knowledge survives `/clear`
**Depends on**: Phase 53 (must be able to read domain ledgers)
**Requirements**: PRIME-01, PRIME-02, PRIME-03, PRIME-04, PRIME-05
**Success Criteria** (what must be TRUE):
  1. Colony-prime assembles a `prior-reviews` section in worker prompts at priority 8 (between user_preferences at 7 and pheromones at 9)
  2. The section shows open findings per domain with severity and file/location summary (e.g., "Security (5 open): HIGH -- bcrypt..., MEDIUM -- auth...")
  3. The section is capped at 800 chars in normal mode and 400 chars in compact mode
  4. When no review ledgers exist, the section is omitted entirely (no empty placeholder)
  5. Colony-prime reads from a cached summary file for performance rather than 7 direct ledger reads on every call
**Plans**: 1 plan

Plans:
- [x] 54-01-PLAN.md -- DomainOrder sharing, cache type, section assembly with formatting/severity-sorting/budget-caps, scoring integration, tests

### Phase 55: Agent Definition Updates
**Goal**: Seven review agents can persist findings to their domain ledgers, with write-scope guardrails preventing escape
**Depends on**: Phase 53 (agents need `review-ledger-write` to exist as a target)
**Requirements**: AGENT-01, AGENT-02, AGENT-03, AGENT-04, AGENT-05, AGENT-06, AGENT-07, AGENT-08, AGENT-09, AGENT-10
**Success Criteria** (what must be TRUE):
  1. Each of the 7 review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) has Write tool in its `tools:` frontmatter
  2. Each agent's instructions include findings write instructions targeting its designated domain(s) (e.g., Gatekeeper writes to security)
  3. Write-scope guardrails explicitly restrict agents to ONLY write to their designated review ledger files under `.aether/data/reviews/`, never source code, tests, or colony state
  4. All 7 agents are synced across all 4 surfaces: `.claude/agents/ant/`, `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/` (28 files verified in sync)
  5. Build and continue dispatch flows inject findings-path instructions into review agent task prompts so agents know where to write
**Plans**: TBD

### Phase 56: Lifecycle Integration
**Goal**: Review ledgers integrate with colony lifecycle -- archived at seal, included in entomb chambers, visible in status, and cleaned at init
**Depends on**: Phase 53 (ledgers must exist), Phase 54 (colony-prime reads ledgers)
**Requirements**: LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05
**Success Criteria** (what must be TRUE):
  1. `/ant-seal` archives the `.aether/data/reviews/` directory alongside existing survey and build archives
  2. `/ant-seal` flags any high-severity unresolved findings in the seal report before archiving
  3. `/ant-entomb` includes the reviews directory in the chamber archive
  4. `/ant-status` displays review ledger counts per domain (total and open entries)
  5. `/ant-init` clears stale reviews from any prior colony to prevent cross-colony contamination
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 52 -> 53 -> 54 -> 55 -> 56

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 52. Continue-Review Worker Outcome Reports | 2/2 | Complete | 2026-04-26 |
| 53. Domain-Ledger CRUD Subcommands | 2/2 | Complete    | 2026-04-26 |
| 54. Colony-Prime Prior-Reviews Section | 0/1 | Not started | - |
| 55. Agent Definition Updates | 0/? | Not started | - |
| 56. Lifecycle Integration | 0/? | Not started | - |
