# Milestones

## v1.0 Aether Repair & Stabilization (Shipped: 2026-02-18)

**Phases completed:** 9 phases, 27 plans
**Requirements:** 46/46 verified PASS (100%)
**Timeline:** 2026-02-17 to 2026-02-18

**Key accomplishments:**
1. Diagnosed and fixed all 9 critical failures across command infrastructure, visual display, and context persistence
2. Built complete pheromone signaling system (FOCUS/REDIRECT/FEEDBACK) with auto-injection into builder/watcher prompts
3. Implemented colony lifecycle management (seal ceremony, chamber archival, tunnels browser)
4. Added XML exchange format for cross-colony pheromone/wisdom/registry transfer
5. Created comprehensive e2e test suite (12 scripts, 46 automated assertions) with bash 3.2 compatibility
6. Verified full connected workflow: init -> colonize -> plan -> build -> continue -> seal -> entomb

**Known caveats:**
- XSD schema validation rejects `expires_at="phase_end"` (semantic value, not ISO 8601) — round-trip works, only strict XSD fails
- Model-per-caste routing unverified under real conditions — all workers use default model

**Archives:** `.planning/milestones/v1.0-ROADMAP.md`, `.planning/milestones/v1.0-REQUIREMENTS.md`

---


## v1.1 Colony Polish & Identity (Shipped: 2026-02-18)

**Phases completed:** 4 phases (10-13), 13 plans
**Requirements:** 14/15 satisfied (NOISE-04 partial — intentional)
**Timeline:** 2026-02-01 to 2026-02-18 (17 days)
**Commits:** 36 | **Files:** 55 | **Lines:** +3,271 / -819
**Git range:** feat(10-02) → test(13-01)

**Key accomplishments:**
1. All 34 commands now show human-readable descriptions instead of raw bash, with session-wide version check caching
2. Unified visual identity — consistent ━━━━ banners, Unicode progress bars, and state-aware "Next Up" blocks across every command
3. Real-time build progress — spawn announcements, worker completion lines with tool counts, and BUILD SUMMARY block
4. Canonical caste emoji system — single source of truth in caste-system.md, consistent in all worker output and documentation
5. Atomic update recovery — `.update-pending` sentinel detects incomplete updates and auto-recovers on next run

### Known Gaps
- **NOISE-04** (partial): 3 session-management commands (resume.md, pause-colony.md, resume-colony.md) still display session_id — intentional deviation for debugging context
- **OpenCode mirror**: 5 connections not backported (cached version check, ━━━━ banners, Next Up block in update.md) — secondary platform, non-blocking

**Archives:** `.planning/milestones/v1.1-ROADMAP.md`, `.planning/milestones/v1.1-REQUIREMENTS.md`

---


## v1.2 Hardening & Reliability (Shipped: 2026-02-19)

**Phases completed:** 6 phases (14-19), 18 plans
**Requirements:** 24/24 satisfied (100%)
**Timeline:** 2026-02-18 to 2026-02-19 (2 days)
**Commits:** 56 | **Files:** 112 | **Lines:** +2,929 / -23,163
**Tests:** 446 passing (415 AVA + 31 bash), 0 failures
**Git range:** feat(14-01) → docs(19-04)

**Key accomplishments:**
1. Fixed json_err fallback and template path resolution to unblock all hardening work (ERR-01, ARCH-01)
2. Cleaned up entire distribution chain — correct hub source directory, removed dead duplicates, deprecated old npm versions (DIST-01 through DIST-06)
3. Eliminated lock deadlocks with uniform trap-based cleanup on all exit paths, stale lock user prompts (LOCK-01 through LOCK-04)
4. Standardized all 49 bare-string json_err calls to E_* constants with contributor documentation (ERR-02, ERR-03, ERR-04)
5. Fixed startup ordering, composed EXIT trap, spawn-tree rotation, model error handling, queen-read validation (ARCH-02 through ARCH-10)
6. Closed all audit gaps with 21 new AVA tests, bringing total to 446 tests with 0 failures

**Known Tech Debt (non-blocking):**
- Flag command trap overwrites composed cleanup (mitigated by startup orphan scan)
- file-lock.sh E_LOCK_STALE uses printf 2-field format vs json_err 5-field (pre-existing, correctly documented)

**Archives:** `.planning/milestones/v1.2-ROADMAP.md`, `.planning/milestones/v1.2-REQUIREMENTS.md`

---


## v1.3 The Great Restructuring (Shipped: 2026-02-20)

**Phases completed:** 6 phases (20-25), 12 plans
**Requirements:** 24/24 satisfied (100%)
**Timeline:** 2026-02-19 to 2026-02-20

**Key accomplishments:**
1. Eliminated runtime/ staging — npm package reads directly from .aether/ (PIPE-01 through PIPE-03)
2. Extracted 5 critical templates (colony-state, constraints, crowned-anthill, handoff, worker-result) and wired all commands to use them (TMPL-01 through TMPL-06, WIRE-01 through WIRE-05)
3. Stripped boilerplate from all 25 agents — removed Aether Integration, Depth-Based Behavior, dead model refs (AGENT-01 through AGENT-04)
4. Added failure modes, success criteria, and read-only declarations to all agents and 6 high-risk commands (RESIL-01 through RESIL-03)
5. Queen rewrite with 4-tier escalation chain, 6 named workflow patterns, and 2 agent merges (Architect→Keeper, Guardian→Auditor) (COORD-01 through COORD-04)

---


## v1.4 Deep Cleanup (Partial — phase 26 only, phases 27-30 absorbed into v2.0)

**Phases completed:** 1 phase (26), 4 plans
**Requirements:** 10/10 satisfied (100%) for phase 26
**Timeline:** 2026-02-20

**Key accomplishments:**
1. Full file audit — every file in repo root and .aether/ classified as KEEP/ARCHIVE/DELETE
2. Removed dead duplicates: .aether/agents/, .aether/commands/, .aether/docs/ subdirectories
3. Cleaned docs/plans/, .planning/milestones/ archives, TO-DOS.md completed items
4. Verified cleanup safety: npm pack (180 files), npm install, npm test, lint:sync (34/34)

**Note:** Phases 27-30 (doc cleanup, bash bug fix, verify) deferred and absorbed into v2.0.

---


## v2.0 Worker Emergence (Shipped: 2026-02-20)

**Phases completed:** 12 phases, 38 plans, 0 tasks

**Key accomplishments:**
- (none recorded)

---


## v3.0 Wisdom & Pheromone Evolution (Shipped: 2026-02-21)

**Phases completed:** 4 phases (32-35), 11 plans
**Requirements:** 25/25 verified (100%)
**Timeline:** 2026-02-20 to 2026-02-21 (2 days)

**Key accomplishments:**
1. Unified colony-prime() function with two-level QUEEN.md loading (global + local)
2. Observation tracking system with content hashing and cross-colony accumulation
3. Tick-to-approve UX with threshold bars, multi-select, and undo support
4. Lifecycle integration — seal.md and entomb.md get wisdom review gates
5. All 5 wisdom categories (Philosophies, Patterns, Redirects, Stack, Decrees) with type-specific thresholds

**Known caveats:**
- OBS-01 observation wire just committed — will be tested in next colony session

**Archives:** `.planning/milestones/v3.0-ROADMAP.md`, `.planning/milestones/v3.0-REQUIREMENTS.md`

---


## v5.0 Agent Integration (Shipped: 2026-02-22)

**Phases completed:** 22 phases, 63 plans, 7 tasks

**Key accomplishments:**
- (none recorded)

---

