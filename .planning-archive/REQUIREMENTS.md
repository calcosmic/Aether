# REQUIREMENTS — v6.0 System Integration

> Wire existing systems together — no new features, just integration

---

## Phase 42: Fix Update Bugs

### UPDATE-01: Fix atomic writes in syncDirWithCleanup

**Status**: v1 requirement

**Description**: Currently, syncDirWithCleanup copies files directly to destination. If the copy is interrupted, the file is corrupted. Use atomic write pattern (write to temp, then rename).

**Acceptance Criteria**:
- [ ] Files are written to temp location first
- [ ] Rename happens only after write completes
- [ ] Interrupted writes leave original file intact
- [ ] No partial file corruption possible

**Source**: CODEBASE-FLOW.md Gap G-1

---

### UPDATE-02: Fix counter bug in dry-run mode

**Status**: v1 requirement

**Description**: In syncAetherToRepo, the `copied++` counter increments even in dry-run mode, causing inaccurate reporting.

**Acceptance Criteria**:
- [ ] Counter only increments when files are actually copied
- [ ] Dry-run mode reports "0 files would be copied"
- [ ] Actual copy mode reports correct count

**Source**: CODEBASE-FLOW.md Gap G-2

---

### UPDATE-03: Clean old directories from user repos

**Status**: v1 requirement

**Description**: Old directories (.aether/agents/, .aether/commands/, etc.) may still exist in user repos even though they're excluded from sync. Clean these up during update.

**Acceptance Criteria**:
- [ ] Detect stale directories in user .aether/
- [ ] Remove directories that are no longer in source
- [ ] Preserve user data directories (data/, dreams/, etc.)
- [ ] Log cleanup actions for visibility

**Source**: CODEBASE-FLOW.md Gap G-3

---

## Phase 43: Make Learning Flow

### FLOW-01: Auto-create learning-observations.json if missing

**Status**: v1 requirement

**Description**: The learning pipeline expects learning-observations.json to exist. If it's missing, operations fail. Auto-create with empty structure during /ant:init.

**Acceptance Criteria**:
- [ ] /ant:init creates learning-observations.json if not exists
- [ ] File has valid empty structure (observations: [])
- [ ] learning-check-promotion handles missing file gracefully
- [ ] No errors when first observation is recorded

**Source**: CODEBASE-FLOW.md Gap L-2, Q-4

---

### FLOW-02: Verify observations → proposals → promotions → QUEEN.md pipeline

**Status**: v1 requirement

**Description**: The learning pipeline has all the pieces but integration is incomplete. Verify the full flow works: observations accumulate, thresholds trigger proposals, approval promotes to QUEEN.md.

**Acceptance Criteria**:
- [ ] Observations recorded during builds accumulate correctly
- [ ] learning-check-promotion returns proposals meeting thresholds
- [ ] User sees tick-to-approve UI for proposals
- [ ] Approved proposals are written to QUEEN.md
- [ ] queen-promote validates thresholds before writing
- [ ] colony-prime reads promoted wisdom correctly

**Source**: CODEBASE-FLOW.md Gaps L-1, L-4

---

### FLOW-03: Test end-to-end with real learning

**Status**: v1 requirement

**Description**: Run actual colony session to verify learning flows from observation to QUEEN.md promotion.

**Acceptance Criteria**:
- [ ] Run /ant:build that records observations
- [ ] Run /ant:continue that checks thresholds
- [ ] Approve proposals in tick-to-approve UI
- [ ] Verify QUEEN.md contains promoted wisdom
- [ ] Verify next /ant:build includes wisdom in colony-prime output

**Source**: Integration testing requirement

---

## Phase 44: Suggest Pheromones

### SUGG-01: Show suggested pheromones with tick-to-approve at build start

**Status**: v1 requirement

**Description**: Before spawning workers, analyze the current situation and suggest pheromones that might help. Show tick-to-approve UI for user to accept/reject suggestions.

**Acceptance Criteria**:
- [ ] Build start triggers pheromone analysis
- [ ] Suggestions displayed with tick-to-approve UI
- [ ] User can select which suggestions to apply
- [ ] Approved suggestions written as FOCUS signals
- [ ] Rejected suggestions are dismissed
- [ ] No build delay if user dismisses all

**Source**: Feature requirement

---

### SUGG-02: Suggestions based on codebase analysis

**Status**: v1 requirement

**Description**: Suggestions should be based on actual code analysis, not random or static suggestions. Look at current state, recent changes, and patterns.

**Acceptance Criteria**:
- [ ] Analysis considers current phase and goal
- [ ] Analysis considers recent git changes
- [ ] Analysis considers existing pheromones (don't duplicate)
- [ ] Suggestions are specific to current context
- [ ] No generic "write good code" suggestions

**Source**: Feature requirement

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| UPDATE-01 | Phase 42 | Complete |
| UPDATE-02 | Phase 42 | Complete |
| UPDATE-03 | Phase 42 | Complete |
| FLOW-01 | Phase 43 | Pending |
| FLOW-02 | Phase 43 | Pending |
| FLOW-03 | Phase 43 | Pending |
| SUGG-01 | Phase 44 | Pending |
| SUGG-02 | Phase 44 | Pending |

**Total: 8 v1 requirements**

---

## Out of Scope

The following are explicitly NOT in v6.0:

- Model-per-caste routing verification
- YAML command generator
- New agent creation
- UI changes beyond tick-to-approve
- Performance optimization
- New wisdom categories

---

*Created: 2026-02-22*
*Milestone: v6.0 System Integration*
