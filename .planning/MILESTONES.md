# Milestones

## v1.3 Maintenance & Pheromone Integration (Shipped: 2026-03-19)

**Phases completed:** 8 phases, 17 plans, 49 commits
**Timeline:** 2026-03-19 (single day)
**Changes:** 79 files, +10,860 / -1,710 lines

**Key accomplishments:**
- Purged all test artifacts from colony state files — clean baseline for real data
- Pheromone signals flow end-to-end: emit → store → inject into worker context → influence behavior
- Workers (builder, watcher, scout) have pheromone_protocol sections acting on REDIRECT/FOCUS/FEEDBACK
- Learning pipeline validated: observations auto-promote to instincts in worker prompts
- XML exchange activated with /ant:export-signals, /ant:import-signals + seal lifecycle
- Fresh install hardened with content-aware validate-package.sh and lifecycle smoke test
- All documentation updated to match verified behavior (no aspirational claims)

---

