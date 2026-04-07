# Roadmap: Aether v1.0 Final Sweep

> Derived from: REQUIREMENTS.md (46 requirements), ARCHITECTURE.md, PITFALLS.md
> Created: 2026-04-07
> Phases: 8

---

## Phase 1: State Protection

**Goal:** Make every colony state mutation traceable, recoverable, and safe from corruption.

**Requirements:** STATE-01, STATE-02, STATE-03, STATE-04, STATE-05, STATE-06, STATE-07

**Plans:** 3 plans

Plans:
- [x] 01-01-PLAN.md -- Storage-layer audit infrastructure (AuditLogger, CorruptionDetector, BoundaryGuard, AutoCheckpoint)
- [x] 01-02-PLAN.md -- Command instrumentation (wire mutations through WriteBoundary, state-write --force)
- [x] 01-03-PLAN.md -- state-history command (compact/diff/tail/json output modes)

**Success Criteria:**
1. Running `aether state-mutate` appends an entry to the audit log with before/after diffs, timestamp, source, and checksum
2. Running `aether state-history` displays the full mutation history in a readable format
3. Submitting a known-bad jq expression (the bracket notation bug) produces a clear rejection error instead of corrupting state
4. Running a destructive operation (e.g., phase advance) automatically creates a checkpoint snapshot before proceeding
5. BoundaryGuard rejects unauthorized writes to sensitive paths (`.aether/data/`, `.aether/dreams/`) during colony operations

**Dependencies:** None (this is the foundation phase)

**Risk Notes:** Multiple write paths (`state-mutate`, `state-write`, `update-progress`, `phase-insert`) bypass the state machine -- every path must be routed through the audit log. The FileLocker only serializes individual writes, not read-modify-write sequences, so write boundaries must group related mutations atomically. Audit overhead on high-frequency operations (context-update called multiple times per build) must be mitigated by batching within write boundaries.

---

## Phase 2: System Integrity

**Goal:** Eliminate data-loss risks from hygiene commands and ensure all Go commands run cleanly on a fresh install.

**Requirements:** INTG-01, INTG-02, INTG-03, INTG-04, INTG-05, INTG-06

**Plans:** 3 plans

Plans:
- [x] 02-01-PLAN.md -- Fix isTestArtifact false-positive, add confirmation gates, standardize error formatting
- [x] 02-02-PLAN.md -- Remove deprecated code, create smoke test suite, run full regression
- [ ] 02-03-PLAN.md -- Gap closure: re-apply orphaned 02-01 changes (isTestArtifact source guard, --confirm safety gates, error convention)

**Success Criteria:**
1. A user pheromone containing the word "test" (e.g., "test the authentication flow") is never flagged as a test artifact by `isTestArtifact`
2. Running `aether backup-prune-global` or `aether temp-clean` without `--confirm` produces a dry-run preview and deletes nothing
3. Running `go test ./...` on a clean checkout passes all 524+ tests with zero failures
4. Every `aether` subcommand runs without panics or silent errors when executed against a freshly initialized colony
5. All error messages across Go modules follow a consistent format (prefix, description, remediation hint)

**Dependencies:** Phase 1 (hygiene audit log uses append-only infrastructure from state protection)

**Risk Notes:** `isTestArtifact` uses fragile substring matching -- fixing it requires adding a `source` field check so user-created signals are never flagged regardless of content. The `state-write` command explicitly documents itself as "bypasses transition validation" and must be guarded behind a `--force` flag to prevent accidental misuse. Agents can invoke destructive hygiene commands autonomously because slash commands run as separate processes outside agent boundary rules -- a TTY check is needed.

---

## Phase 3: Build Depth Controls

**Goal:** Let users control how thoroughly the colony builds by selecting light, standard, or deep depth.

**Requirements:** DEPTH-01, DEPTH-02, DEPTH-03, DEPTH-04, DEPTH-05, DEPTH-06

**Success Criteria:**
1. Running `/ant:init "goal" --depth light` sets the colony depth and it persists across `/ant:status` calls
2. Running `/ant:build --depth light` produces a build with 1 builder, no archaeologist scan, no measurer, and no ambassador
3. Running `/ant:build --depth deep` produces a build that includes measurer, ambassador, extended chaos iterations, and security quality gates
4. The colony-prime context budget adjusts based on depth (light = smaller budget, deep = larger budget)
5. Running `/ant:build` without `--depth` uses the colony's persisted depth setting

**Dependencies:** Phase 1 (depth changes are audited), Phase 2 (commands run cleanly)

**Risk Notes:** `ColonyDepth` is a string field with no struct validation -- `state-mutate` can set it to any value and the system silently falls back to defaults. Depth currently controls spawn count but not prompt budget, so the token budget scaling must be wired in. Per-phase depth overrides are not in v1 scope but the architecture should not preclude them.

---

## Phase 4: Planning Granularity Controls

**Goal:** Let users control how many phases the plan generates by selecting a granularity range.

**Requirements:** PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05

**Plans:** 3 plans

Plans:
- [x] 04-01-PLAN.md -- PlanGranularity enum, GranularityRange, CLI commands, status display, state-mutate validation
- [x] 04-02-PLAN.md -- Plan command --granularity flag, route-setter dynamic bounds, out-of-range validation
- [x] 04-03-PLAN.md -- Autopilot granularity awareness in Step 0

**Success Criteria:**
1. Running `/ant:plan --granularity sprint` produces a plan with 1-3 phases
2. Running `/ant:plan --granularity quarter` produces a plan with 8-12 phases
3. If the route-setter generates a plan outside the selected range, the user sees a warning and is prompted to approve or adjust
4. The selected granularity persists in COLONY_STATE.json and appears in `/ant:status` output
5. Running `/ant:run` after setting granularity respects the phase count across the entire autopilot loop

**Dependencies:** Phase 1 (planning history is append-only), Phase 3 (granularity interacts with depth for context budgeting)

**Risk Notes:** Planning granularity is a hint to the LLM-based route-setter, not a hard constraint -- the system must validate output and warn on violations rather than silently accepting out-of-range plans. The autopilot must read the persisted granularity from COLONY_STATE.json (not its own separate state file) to avoid the dual-source-of-truth problem identified in the orchestration pitfall analysis.

---

## Phase 5: Orchestration Layer

**Goal:** Provide a centralized Go-based coordinator that decomposes phases into tasks and assigns specialist agents.

**Requirements:** ORCH-01, ORCH-02, ORCH-03, ORCH-04, ORCH-05, ORCH-06, ORCH-07

**Success Criteria:**
1. Running `aether orchestrator-decompose --phase 1` produces a list of tasks with assigned castes (builder, watcher, scout, etc.)
2. Running `aether orchestrator-assign --phase 1` matches each task to a specific specialist agent based on task type, pheromones, and skills
3. During a build, no agent performs work outside its assigned task scope (enforced by the orchestrator's isolation model)
4. After each phase, all agent outputs are collected and validated against success criteria before marking the phase complete
5. Running `aether orchestrator-status` shows full visibility of the current phase's task assignments, progress, and agent states

**Dependencies:** Phase 1 (write boundaries for orchestrated operations), Phase 3 (depth config for spawn budgets), Phase 4 (planning config for phase decomposition)

**Risk Notes:** This is the most complex new component -- it touches autopilot, context assembly, spawn tree, pheromone system, agent pool, and colony state. The current autopilot has its own separate `state.json` that diverges from COLONY_STATE.json; the orchestrator must unify these into a single source of truth. Build incrementally: decomposition first, then assignment, then adaptive replan. The curation orchestrator in `pkg/agent/curation/orchestrator.go` is the proven template.

---

## Phase 6: Branching & Worktree Discipline

**Goal:** Enforce branch naming conventions, track worktree lifecycles, and prevent orphaned branches.

**Requirements:** BRAN-01, BRAN-02, BRAN-03, BRAN-04, BRAN-05, BRAN-06

**Success Criteria:**
1. Running `aether worktree-allocate --agent builder-1 --phase 2` creates a branch with the enforced naming convention (e.g., `phase-2/builder-1`) and rejects arbitrary names
2. Running `aether worktree-list` shows all tracked worktrees with their status (allocated, in-progress, merged, orphaned)
3. Running `aether worktree-orphan-scan` detects worktrees with no recent activity and flags them for cleanup
4. Running `aether worktree-merge-back` refuses to merge if tests fail or clash-check detects conflicts
5. After a successful merge, the worktree branch is removed and the worktree directory is cleaned up automatically

**Dependencies:** Phase 1 (immutable merge logs, audit trail), Phase 5 (orchestrator manages worktree allocation)

**Risk Notes:** The current `worktree-create` accepts arbitrary branch names with no validation. No branch lifecycle tracking exists -- worktrees are created but never registered, so crashed sessions leave orphaned worktrees on disk. The clash detection merge driver only fires at merge time, not during rebase or cherry-pick. Pre-commit hooks for go vet and go fmt should be set up as part of the merge protocol.

---

## Phase 7: Repository Hygiene

**Goal:** Clean up the repository so a new contributor can understand its structure in under 5 minutes.

**Requirements:** HYGN-01, HYGN-02, HYGN-03, HYGN-04, HYGN-05

**Success Criteria:**
1. Running `ls` on the repo root shows a clean directory structure with no legacy clutter or ambiguous old/new system files
2. Searching the entire repository for shell script references (`.sh`, `bash`, `aether-utils.sh`) returns zero results in active code paths
3. Running `grep -r "func " cmd/ pkg/` shows no commented-out functions or unreachable code paths
4. A developer unfamiliar with the project can navigate from CLAUDE.md to any source file within 5 minutes without confusion
5. All documentation files (README, CLAUDE.md, agent definitions, command specs) consistently reference Go commands, never shell scripts

**Dependencies:** Phase 2 (error handling is consistent), Phase 6 (worktree system is in place, deprecated worktree-merge.go can be removed)

**Risk Notes:** 41 dead shell scripts in `.aether/utils/` have DEPRECATED headers but remain on disk -- they must be removed or clearly quarantined. The parity model between Claude Code and OpenCode means cleanup must be done in both `.claude/` and `.opencode/` directories simultaneously. Documentation drift is a risk -- files may reference shell commands that were removed in the v5.0 migration but never updated in the text.

---

## Phase 8: Final Validation

**Goal:** Confirm the entire system works end-to-end with no regressions before declaring v1.0 complete.

**Requirements:** VALD-01, VALD-02, VALD-03, VALD-04, VALD-05

**Success Criteria:**
1. Running `go test ./... -race` passes with zero failures and no race conditions detected
2. A simulated full workflow (`aether init` -> colonize -> plan -> build -> continue -> seal) completes without errors on a fresh colony
3. Stress-testing concurrent state mutations (multiple `state-mutate` calls in parallel) produces no corruption and all mutations are reflected in the audit log
4. Running three sequential phases with specialist handoffs (builder -> watcher -> chaos) produces correct orchestration status at each step
5. Running `aether known-issues` (or reading `known-issues.md`) shows any remaining gaps with severity ratings and no critical issues

**Dependencies:** All previous phases (this is the final validation pass)

**Risk Notes:** The simulated workflow must test both light and deep depth modes, both sprint and quarter granularity, and the full worktree lifecycle (allocate -> work -> merge-back -> cleanup). Large colony states (many phases, many instincts) should be tested to verify the token budget trimming works correctly. Rapid phase cycling (advancing through phases quickly) should not cause state divergence or audit log corruption.

---

## Coverage Summary

| Phase | Requirements | Count |
|-------|-------------|-------|
| Phase 1: State Protection | STATE-01 through STATE-07 | 7 |
| Phase 2: System Integrity | INTG-01 through INTG-06 | 6 |
| Phase 3: Build Depth Controls | DEPTH-01 through DEPTH-06 | 6 |
| Phase 4: Planning Granularity Controls | PLAN-01 through PLAN-05 | 5 |
| Phase 5: Orchestration Layer | ORCH-01 through ORCH-07 | 7 |
| Phase 6: Branching & Worktree Discipline | BRAN-01 through BRAN-06 | 6 |
| Phase 7: Repository Hygiene | HYGN-01 through HYGN-05 | 5 |
| Phase 8: Final Validation | VALD-01 through VALD-05 | 5 |
| **Total** | | **47** |

> Note: REQUIREMENTS.md lists 46 v1 requirements. The traceability table in REQUIREMENTS.md was pre-populated with phase assignments during requirements definition. The roadmap above aligns with those assignments. A minor reconciliation note: REQUIREMENTS.md lists 7 STATE + 6 DEPTH + 5 PLAN + 7 ORCH + 6 BRAN + 6 INTG + 5 HYGN + 5 VALD = 47 entries, but the document header states "46 total". This discrepancy (one extra) should be verified during Phase 1 planning.

**Every requirement is mapped to exactly one phase. Zero requirements are unmapped.**

---
*Roadmap created: 2026-04-07*
*Plans added: 2026-04-07*
