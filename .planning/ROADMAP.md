# Roadmap: Aether

## Milestones

- ✅ **v1.0 Repair & Stabilization** — Phases 1-9 (shipped 2026-02-18)
- [ ] **v1.1 Colony Polish & Identity** — Phases 10-13

## Phases

<details>
<summary>✅ v1.0 Repair & Stabilization (Phases 1-9) — SHIPPED 2026-02-18</summary>

- [x] Phase 1: Diagnostic (3 plans) — 120 tests, 66% pass, 9 critical failures identified
- [x] Phase 2: Core Infrastructure (5 plans) — fixed command foundations
- [x] Phase 3: Visual Experience (2 plans) — swarm display, emoji castes, colors
- [x] Phase 4: Context Persistence (2 plans) — drift detection, rich resume dashboard
- [x] Phase 5: Pheromone System (3 plans) — FOCUS/REDIRECT/FEEDBACK, auto-injection, eternal memory
- [x] Phase 6: Colony Lifecycle (3 plans) — seal ceremony, entomb archival, tunnels browser
- [x] Phase 7: Advanced Workers (3 plans) — oracle, chaos, archaeology, dream, interpret synced
- [x] Phase 8: XML Integration (4 plans) — pheromone/wisdom/registry XML, seal export, entomb hard-stop
- [x] Phase 9: Polish & Verify (4 plans) — 46/46 requirements PASS, full e2e test suite

**46 requirements verified. Full details: `.planning/milestones/v1.0-ROADMAP.md`**

</details>

---

### v1.1 Colony Polish & Identity

- [ ] **Phase 10: Noise Reduction** — Human-readable bash descriptions and call consolidation across all commands
- [ ] **Phase 11: Visual Identity** — Consistent banners, dividers, Next Up blocks, progress bars, unified caste emojis
- [ ] **Phase 12: Build Progress** — Spawning indicators, worker completion lines, and tmux-only swarm display
- [ ] **Phase 13: Distribution Reliability** — Fix update version detection and add atomic version stamp

---

## Phase Details

### Phase 10: Noise Reduction
**Goal**: Every bash tool call shows a human-readable header and the total call count is reduced by 30-40%
**Depends on**: Nothing (first phase of v1.1)
**Requirements**: NOISE-01, NOISE-02, NOISE-03, NOISE-04
**Success Criteria** (what must be TRUE):
  1. Every visible bash tool call header reads as a plain English status ("Checking colony state..." not raw bash syntax) across all 34 commands
  2. Sequential non-dependent bash calls are consolidated so a typical /ant:build shows at least 30% fewer tool call headers than before
  3. Running any command twice in a session shows the version check only once — the second run skips it silently
  4. No session IDs, internal identifiers, or technical tokens appear in any user-facing command output
**Plans**: 4 plans
Plans:
- [ ] 10-01-PLAN.md — Version check caching (NOISE-03) and identifier cleanup (NOISE-04)
- [ ] 10-02-PLAN.md — Descriptions for 22 low-complexity commands (NOISE-01, NOISE-02)
- [ ] 10-03-PLAN.md — Descriptions for 6 medium-complexity commands (NOISE-01, NOISE-02)
- [ ] 10-04-PLAN.md — Descriptions for build.md and continue.md (NOISE-01, NOISE-02)

### Phase 11: Visual Identity
**Goal**: All commands share one coherent visual language — same banners, dividers, status icons, and a "Next Up" block on every completion
**Depends on**: Phase 10
**Requirements**: VIS-01, VIS-02, VIS-03, VIS-04, VIS-05
**Success Criteria** (what must be TRUE):
  1. Every command that shows worker activity displays the worker's caste emoji next to their name (e.g., "🔨 Hammer-42")
  2. Every command completion output ends with a "Next Up" block containing at least one copy-paste-ready command
  3. /ant:status shows a visual progress bar reflecting current phase and task completion percentage
  4. All commands use identical banner and divider styles — no mix of ===, ----, ━━━, or other variants
  5. Caste emoji definitions exist in exactly one shared location; all commands reference that single source
**Plans**: 4 plans
Plans:
- [ ] 11-01-PLAN.md — Create canonical caste-system.md and update references (VIS-05)
- [ ] 11-02-PLAN.md — Add progress bars to /ant:status and Next Up block (VIS-03, VIS-02)
- [ ] 11-03-PLAN.md — Standardize banners in high-visibility commands (VIS-04, VIS-02, VIS-01)
- [ ] 11-04-PLAN.md — Apply visual identity to remaining 28 commands (VIS-04, VIS-02)

### Phase 12: Build Progress
**Goal**: Users see what the colony is doing during parallel execution — no silent gaps, no black boxes
**Depends on**: Phase 11
**Requirements**: PROG-01, PROG-02, PROG-03, PROG-04
**Success Criteria** (what must be TRUE):
  1. Before any parallel worker wave begins, the user sees "Spawning N [caste] workers in parallel..." with the count and caste
  2. When each worker completes, a consistent single-line summary appears showing caste emoji, ant name, task, and tool counts
  3. Swarm display update calls only render live output inside an active tmux session; in Claude Code chat, only the final summary appears
  4. Every spawned sub-agent Task call includes the caste emoji and ant name in its description (e.g. "🔨 Builder Hammer-42: Implement login")
**Plans**: TBD

### Phase 13: Distribution Reliability
**Goal**: /ant:update gives accurate status and leaves the installation in a clean, consistent state regardless of what happened before
**Depends on**: Nothing (independent of Phases 10-12, run after Phase 12 to keep visual changes batched)
**Requirements**: DIST-01, DIST-02
**Success Criteria** (what must be TRUE):
  1. Running /ant:update when already on the current version shows "Already up to date" and performs no sync work
  2. If an update fails partway through, re-running /ant:update detects the incomplete state and re-syncs cleanly without manual intervention
**Plans**: TBD

---

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 10. Noise Reduction | 0/4 | Planned | - |
| 11. Visual Identity | 0/4 | Planned | - |
| 12. Build Progress | 0/? | Not started | - |
| 13. Distribution Reliability | 0/? | Not started | - |

---

*Roadmap created: 2026-02-17*
*v1.0 shipped: 2026-02-18*
*v1.1 roadmap added: 2026-02-18*
