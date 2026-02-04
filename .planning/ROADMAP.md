# Milestone v4.4: Colony Hardening & Real-World Readiness

**Status:** In Progress
**Phases:** 27-32
**Total Plans:** TBD

## Overview

v4.4 addresses all 23 actionable findings from the first real-world field test (filmstrip packaging, 2026-02-04). The milestone fixes critical bugs (pheromone decay growing instead of decaying, activity log overwriting), reduces UX friction (auto-continue, context clear prompting), improves colony intelligence (adaptive complexity, calibrated watcher scoring, multi-ant colonization), adds automation capabilities (auto-reviewer, debugger, tech debt reporting), evolves the architecture (two-tier learning, spawn tree engine), and polishes safety-sensitive features (archivist ant, pheromone documentation).

Ordering is driven by three constraints: broken foundations invalidate features built on top; UX friction causes abandonment faster than missing features; calibrated quality signals are prerequisites for meaningful automation.

## Phases

### Phase 27: Bug Fixes & Safety Foundation

**Goal**: Colony operates on correct data -- pheromone signals decay properly, activity history persists across phases, errors are traceable to their source phase, decisions are recorded during execution, and tasks touching the same file cannot conflict
**Depends on**: Phase 26 (v4.3 complete)
**Requirements**: BUG-01, BUG-02, BUG-03, BUG-04, INT-02
**Plans:** 2 plans

Plans:
- [x] 27-01-PLAN.md — Fix pheromone decay guards, activity log append, error-add phase param in aether-utils.sh
- [x] 27-02-PLAN.md — Wire phase to error-add calls, add decision logging, add conflict prevention rule in build.md

**Success Criteria:**
1. A FOCUS pheromone emitted 30 minutes ago shows lower effective strength than when it was emitted -- decay math produces monotonically decreasing values
2. After running 3 phases of a colony build, the activity log contains entries from all 3 phases with correct phase numbers (not just the latest phase)
3. When a worker encounters an error, the entry in errors.json includes a "phase" field identifying which phase it occurred in
4. After a build phase executes, memory.json decisions array contains entries from that phase (not only Phase 0 planning decisions)
5. When two tasks in the same phase touch the same file, the Phase Lead assigns them to the same worker in the plan

---

### Phase 28: UX & Friction Reduction

**Goal**: Users can run multi-phase colony builds without losing state on context clear and without manually approving every phase boundary
**Depends on**: Phase 27
**Requirements**: UX-01, UX-02, FLOW-01

Plans:
- [ ] TBD

**Success Criteria:**
1. After any command that completes meaningful work (/ant:build, /ant:continue, /ant:colonize), the output ends with a "safe to /clear" message confirming state has been persisted
2. Running `/ant:continue --all` executes remaining phases sequentially without requiring user approval at each phase boundary
3. After `/ant:colonize` completes, the output suggests specific pheromone injections (FOCUS, REDIRECT) before proceeding to `/ant:plan`

---

### Phase 29: Colony Intelligence & Quality Signals

**Goal**: Colony produces calibrated quality assessments, adapts its overhead to project size, and leverages multiple perspectives during colonization
**Depends on**: Phase 28
**Requirements**: INT-01, INT-03, INT-04, INT-05, INT-07, ARCH-03

Plans:
- [ ] TBD

**Success Criteria:**
1. Watcher ants produce scores that vary meaningfully across code of different quality -- a clean implementation and a messy one do not both receive 8/10
2. `/ant:colonize` spawns multiple colonizer ants that independently review the codebase and their findings are synthesized into a unified assessment
3. Phase Lead assigns independent tasks to parallel waves rather than running everything sequentially -- observable in the plan output as multiple tasks per wave
4. For phases below a complexity threshold, the Phase Lead auto-approves the plan without requiring user confirmation
5. The colony mode (LIGHTWEIGHT/STANDARD/FULL) is set during colonization based on project complexity indicators and stored in COLONY_STATE.json

---

### Phase 30: Automation & New Capabilities

**Goal**: Colony automates post-build quality gates, surfaces actionable recommendations, and provides visual feedback during execution
**Depends on**: Phase 29 (calibrated scoring prerequisite for meaningful auto-review)
**Requirements**: AUTO-01, AUTO-02, AUTO-03, AUTO-04, AUTO-05, INT-06

Plans:
- [ ] TBD

**Success Criteria:**
1. After builder waves complete, a reviewer ant auto-spawns in advisory mode -- findings are displayed to user but do not block progress, only CRITICAL severity triggers a rebuild (max 2 iterations)
2. When a worker's tests fail, a debugger ant auto-spawns to diagnose the failure
3. After a build completes, the output includes pheromone recommendations (e.g., "Recommended: /ant:focus ..." ) based on build outcomes
4. Build output includes ANSI-colored progress indicators with caste-specific colors (cyan=colonizer, green=builder, magenta=watcher)
5. At project completion, a tech debt report is generated aggregating persistent cross-phase issues from the activity log and errors.json

---

### Phase 31: Architecture Evolution

**Goal**: Colony supports hierarchical task delegation and accumulates cross-project knowledge that persists beyond individual projects
**Depends on**: Phase 30 (automation validates quality signal pipeline that spawn tree depends on)
**Requirements**: ARCH-01, ARCH-02

Plans:
- [ ] TBD

**Success Criteria:**
1. memory.json stores project-local learnings and ~/.aether/learnings.json stores promoted global learnings -- the two tiers are distinct files with independent lifecycles
2. User can manually promote a project learning to the global tier, and global learnings are injected as FEEDBACK pheromones when initializing new projects
3. Workers can signal sub-spawn needs in their output, and the Queen fulfills those requests -- observable as a depth-2 delegation chain in COLONY_STATE.json spawn_tree
4. Spawn tree depth is capped at 2 with enforcement -- a depth-2 worker cannot request further sub-spawns

---

### Phase 32: Polish & Safety Rails

**Goal**: Colony maintains codebase hygiene through safe reporting and users understand when and why to use each pheromone signal
**Depends on**: Phase 31
**Requirements**: FLOW-02, FLOW-03

Plans:
- [ ] TBD

**Success Criteria:**
1. An organizer/archivist ant can be spawned that reports stale files, dead code, and orphaned configs -- output is report-only with no deletions or modifications
2. Pheromone documentation exists explaining when and why to use FOCUS, REDIRECT, and FEEDBACK signals with practical scenarios drawn from real colony usage

---

## Progress

| Phase | Name | Plans | Status |
|-------|------|-------|--------|
| 27 | Bug Fixes & Safety Foundation | 2 plans | ✓ Complete |
| 28 | UX & Friction Reduction | TBD | Pending |
| 29 | Colony Intelligence & Quality Signals | TBD | Pending |
| 30 | Automation & New Capabilities | TBD | Pending |
| 31 | Architecture Evolution | TBD | Pending |
| 32 | Polish & Safety Rails | TBD | Pending |

## Coverage

All 24 v1 requirements mapped:

| Requirement | Phase | Description |
|-------------|-------|-------------|
| BUG-01 | 27 | Pheromone decay math fix |
| BUG-02 | 27 | Activity log append across phases |
| BUG-03 | 27 | Error phase attribution |
| BUG-04 | 27 | Decision logging during execution |
| INT-02 | 27 | Same-file task assignment |
| UX-01 | 28 | Safe-to-clear prompting |
| UX-02 | 28 | Auto-continue mode |
| FLOW-01 | 28 | Pheromone-first flow |
| INT-01 | 29 | Multi-ant colonization |
| INT-03 | 29 | Aggressive wave parallelism |
| INT-04 | 29 | Phase Lead auto-approval |
| INT-05 | 29 | Watcher scoring rubric |
| INT-07 | 29 | Colony overhead adaptation |
| ARCH-03 | 29 | Adaptive complexity mode |
| AUTO-01 | 30 | Auto-spawned reviewer |
| AUTO-02 | 30 | Auto-spawned debugger |
| AUTO-03 | 30 | Pheromone recommendations |
| AUTO-04 | 30 | Animated build indicators |
| AUTO-05 | 30 | Colonizer visual output |
| INT-06 | 30 | Tech debt report |
| ARCH-01 | 31 | Two-tier learning system |
| ARCH-02 | 31 | Spawn tree engine |
| FLOW-02 | 32 | Organizer/archivist ant |
| FLOW-03 | 32 | Pheromone user documentation |

**Mapped: 24/24** -- no orphans, no duplicates.

## Research Flags

Phases likely needing deeper research during planning:
- **Phase 29**: Watcher scoring rubric design requires testing against varied code quality. Multi-colonizer synthesis pattern is novel.
- **Phase 31**: Spawn tree engine requires a 30-minute platform validation test (Task tool availability for subagents) before implementation. Two-tier learning promotion heuristics need empirical calibration.

Phases with standard patterns (skip research):
- **Phase 27**: All fixes have verified root causes and known solutions.
- **Phase 28**: Prompt text changes and UX flow reordering only.
- **Phase 30**: Auto-reviewer follows documented patterns. ANSI progress bars are solved.
- **Phase 32**: Archivist is report-only. Documentation is documentation.

## Critical Pitfalls Encoded

- **CP-1**: Recursive spawning may be blocked by Claude Code platform -- validate in Phase 31 before designing spawn tree
- **CP-3**: Auto-reviewer (Phase 30) depends on calibrated watcher scoring (Phase 29) -- ordering enforced
- **CP-4**: Same-file conflicts (Phase 27) fixed before increasing parallelism (Phase 29)
- **CP-5**: Global learning (Phase 31) starts manual-only, capped at 50 entries

---

_Roadmap created: 2026-02-04_
_Last updated: 2026-02-04 — Phase 27 complete_
