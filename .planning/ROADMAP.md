# Roadmap: Aether Repair & Stabilization

**Created:** 2026-02-17
**Core Value:** Context preservation, clear workflow guidance, self-improving colony

## Phase Structure

This is a REPAIR project — phases are organized to fix foundations first, then features.

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Diagnostic | Understand what's actually broken vs working | All | Run each major command, document failures |
| 2 | Core Infrastructure | Fix command foundations | CMD-01 to CMD-08, ERR-01 to ERR-03 | Basic commands work without errors |
| 3 | Visual Experience | Make display work nicely | VIS-01 to VIS-04 | Ants visible, no bash text flood |
| 4 | Context Persistence | Complete    | 2026-02-17 | Survives /clear |
| 5 | Pheromone System | Fix self-learning | PHER-01 to PHER-05 | Signals work, instincts apply |
| 6 | Colony Lifecycle | Complete    | 2026-02-18 | Archive and browse works |
| 7 | Advanced Workers | Fix oracle/chaos/archaeology | ADV-01 to ADV-05 | Specialized agents work |
| 8 | XML Integration | Wire XML into system | XML-01 to XML-03 | XML used throughout |
| 9 | Polish & Verify | Test end-to-end | All | Complete repair validated |

**Coverage:**
- v1 requirements: 44 total
- Mapped to phases: 44
- Unmapped: 0

---

## Phase Details

### Phase 1: Diagnostic ✅ COMPLETE
**Goal:** Understand what's actually broken vs working

**Status:** Complete (2026-02-17) — 120 tests run, 66% pass rate, 9 critical failures identified

**Requirements:** All (initial assessment)

**Plans:** 3 plans

Plans:
- [x] 01-diagnostic-01-PLAN.md — Test Layer 1 aether-utils.sh subcommands (70+ commands)
- [x] 01-diagnostic-02-PLAN.md — Test Layer 2 slash commands and Layer 3 CLI wrapper
- [x] 01-diagnostic-03-PLAN.md — Test advanced systems and create executive summary

**Success Criteria:**
1. Run `/ant:status` — works or documents error
2. Run `/ant:init` with test goal — works or documents error
3. Run `/ant:help` — displays help or errors
4. Check `.aether/aether-utils.sh` — source exists, runs
5. Check command definitions — files exist in correct places
6. Document what's working vs broken

**Output:**
- `.planning/phases/01-diagnostic/01-diagnostic-report.md` — Complete diagnostic with executive summary

---

### Phase 2: Core Infrastructure
**Goal:** Fix command foundations

**Requirements:**
- CMD-01 through CMD-08
- ERR-01 through ERR-03

**Plans:** 5 plans

Plans:
- [ ] 02-01-PLAN.md — Fix session-is-stale and session-summary JSON output
- [ ] 02-02-PLAN.md — Fix session-clear and context-update argument parsing
- [ ] 02-03-PLAN.md — Implement pheromone-read subcommand
- [ ] 02-04-PLAN.md — Fix spawn-can-spawn-swarm syntax error
- [ ] 02-05-PLAN.md — Add aether status CLI and resume.md frontmatter

**Success Criteria:**
1. `/ant:help` shows all commands
2. `/ant:init "test"` creates valid state
3. `/ant:status` shows dashboard
4. No 401 errors during commands
5. No infinite spawn loops
6. Clear error messages

**Approach:**
- Fix file paths in command definitions
- Fix aether-utils.sh functions
- Add proper error handling
- Test each command

---

### Phase 3: Visual Experience
**Goal:** Make display work nicely

**Requirements:**
- VIS-01 through VIS-04

**Plans:** 2 plans

Plans:
- [ ] 03-01-PLAN.md — Add swarm-display-text function to aether-utils.sh
- [ ] 03-02-PLAN.md — Wire swarm-display-text into /ant:swarm and /ant:colonize

**Success Criteria:**
1. `/ant:init` shows ant emoji output
2. Build phases show ants working
3. Progress indication visible
4. No bash text flood

**Approach:**
- Fix swarm-display functions
- Ensure emoji rendering
- Check terminal compatibility
- Test visual output

---

### Phase 4: Context Persistence
**Goal:** Fix session state

**Requirements:**
- CTX-01 through CTX-03
- STA-01 through STA-03

**Plans:** 2/2 plans complete

Plans:
- [ ] 04-01-PLAN.md — Add drift detection infrastructure and harden session-update consistency
- [ ] 04-02-PLAN.md — Rewrite /ant:resume with rich dashboard, drift detection, and workflow guidance

**Success Criteria:**
1. After `/clear`, state is restored
2. COLONY_STATE.json persists correctly
3. Next command guidance shown
4. No file path hallucinations

**Approach:**
- Add baseline_commit to session tracking for drift detection
- Verify session-update calls across all key commands
- Rewrite /ant:resume with rich dashboard and adaptive next-step guidance
- Add new-conversation detection to CLAUDE.md

---

### Phase 5: Pheromone System
**Goal:** Fix self-learning

**Requirements:**
- PHER-01 through PHER-05

**Plans:** 3 plans

Plans:
- [ ] 05-01-PLAN.md — Unify signal storage: pheromone-write subcommand and update emitter commands
- [ ] 05-02-PLAN.md — Inject signals and instincts into builder/watcher prompts
- [ ] 05-03-PLAN.md — Auto-emission on phase advance, midden archival, eternal memory setup

**Success Criteria:**
1. FOCUS/REDIRECT/FEEDBACK work
2. Instincts apply to new work
3. Learnings extracted after phases
4. Memory persists

**Approach:**
- Fix pheromone read/write
- Wire instincts into builder prompts
- Fix learning extraction in continue
- Test signal application

---

### Phase 6: Colony Lifecycle COMPLETE
**Goal:** Fix seal/entomb/chambers

**Status:** Complete (2026-02-18) — 3/3 plans, seal ceremony-only, entomb seal-first with full archive, tunnels timeline

**Requirements:**
- LIF-01 through LIF-03

**Plans:** 3/3 plans complete

Plans:
- [x] 06-01-PLAN.md — Rewrite /ant:seal as ceremony-only (maturity gate, confirmation, ASCII art, CROWNED-ANTHILL.md)
- [x] 06-02-PLAN.md — Rewrite /ant:entomb with seal-first enforcement, full archive, date-first naming
- [x] 06-03-PLAN.md — Rewrite /ant:tunnels with timeline view, seal document display, comparison

**Success Criteria:**
1. `/ant:seal` creates milestone
2. `/ant:entomb` archives to chambers
3. `/ant:tunnels` shows archives

**Approach:**
- Fix seal command
- Fix entomb to chamber-archive
- Fix tunnels browsing
- Test full lifecycle

---

### Phase 7: Advanced Workers COMPLETE
**Goal:** Fix specialized agents

**Status:** Complete (2026-02-18) — 3/3 plans, oracle verified, chaos/archaeology sync fixed, interpret OpenCode adapted

**Requirements:**
- ADV-01 through ADV-05

**Plans:** 3/3 plans complete

Plans:
- [x] 07-01-PLAN.md — Verify and fix Oracle command and oracle.sh across all 3 locations
- [x] 07-02-PLAN.md — Fix Chaos and Archaeology sync drift, verify function references
- [x] 07-03-PLAN.md — Verify Dream and fix Interpret OpenCode adaptation

**Success Criteria:**
1. `/ant:oracle` runs research
2. `/ant:chaos` runs resilience tests
3. `/ant:archaeology` analyzes git
4. `/ant:dream` writes wisdom
5. `/ant:interpret` validates dreams

**Approach:**
- Fix sync drift between copies (chaos/archaeology Claude Code uses wrong display function)
- Add missing OpenCode adaptation for interpret
- Verify all aether-utils.sh function references are correct
- Test each command against the Aether codebase itself

---

### Phase 8: XML Integration
**Goal:** Wire XML into system

**Requirements:**
- XML-01 through XML-03

**Success Criteria:**
1. Pheromones use XML format
2. Wisdom exchange uses XML
3. Registry uses XML

**Approach:**
- Integrate existing XML utils
- Convert pheromone storage
- Wire wisdom XML
- Test XML round-trip

---

### Phase 9: Polish & Verify
**Goal:** Complete repair validated

**Requirements:** All

**Success Criteria:**
1. Full workflow test: init -> plan -> build -> continue -> seal
2. Visual display works end-to-end
3. Context survives /clear
4. No errors in normal operation
5. All 40 requirements verified

**Approach:**
- End-to-end testing
- User acceptance testing
- Fix any remaining issues
- Document what works

---

## Notes

- This roadmap assumes each phase reveals additional issues
- Testing happens at each phase
- Phases may need splitting if too large
- Some phases depend on prior phases completing

---

*Roadmap created: 2026-02-17*
*Plans added: 2026-02-17*
