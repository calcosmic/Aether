# Project State

**Project:** Aether Repair & Stabilization
**Core Value:** Context preservation, clear workflow guidance, self-improving colony

## Current Status

- **State:** Phase 7 CONTEXT GATHERED
- **Phase:** 07 (Advanced Workers) — context gathered, ready for planning
- **Plan:** —
- **Total Plans in Phase:** TBD
- **Mode:** YOLO (auto-approve)

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-18)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users

**Current focus:** Phase 7: Advanced Workers — plan next

## Progress

- [x] Phase 1: Diagnostic — COMPLETE (120 tests, 66% pass, 9 critical failures identified)
- [x] Phase 2: Core Infrastructure — COMPLETE (5/5 plans)
- [x] Phase 3: Visual Experience — COMPLETE (2/2 plans)
- [x] Phase 4: Context Persistence — COMPLETE (3/3 plans)
- [x] Phase 5: Pheromone System — COMPLETE (3/3 plans)
- [x] Phase 6: Colony Lifecycle — COMPLETE (3/3 plans)
- [ ] Phase 7: Advanced Workers
- [ ] Phase 8: XML Integration
- [ ] Phase 9: Polish & Verify

## Decisions

- **02-01:** session-is-stale uses json_ok wrapper instead of raw echo for consistent JSON output
- **02-01:** session-summary preserves text output as default, adds --json flag for machine parsing
- **02-02:** Add early validation for empty ctx_action before case statement (cleaner error handling)
- **02-02:** Include all valid actions in error messages for discoverability
- **02-03:** Case-insensitive type filtering for pheromone-read (FOCUS/focus/Focus all work)
- **02-03:** Return full pheromone object with metadata, not just content
- **02-04:** Fix grep -c || echo 0 bug — use `|| current=0` instead to avoid double output
- **02-05:** aether status CLI already implemented, resume.md frontmatter already present
- **03-01:** swarm-display-text is additive alongside swarm-display-inline — both coexist, commands opt-in to text variant
- **03-01:** Local helper renamed format_tools_text to avoid bash name collision with swarm-display-inline's format_tools function
- **03-01:** jq total_active expression handles both flat and nested JSON structures for flexibility
- **03-02:** Variable casing matched existing conventions per-command ($SWARM_ID in swarm.md, $colonize_id in colonize.md)
- **04-01:** session-update refreshes baseline_commit on every call (not just init) so stored hash is always last-known HEAD
- **04-01:** Task 2 audit found all four commands already had correct session tracking calls — no changes needed
- **04-01:** validate-state added to plan.md, build.md, continue.md after COLONY_STATE.json writes; init.md already had it
- **04-02:** Dashboard ordering is straight-to-action — Next recommendation renders before Goal and phase context
- **04-02:** Blocking is early-return not guidance — BLOCKED conditions output redirect and stop, no dashboard rendered
- **04-02:** Time-agnostic restore — no 24h staleness, no age warnings, identical restore regardless of gap duration
- **04-02:** Session Recovery in CLAUDE.md for new conversations (not /clear) with mandatory explicit /ant:resume
- **05-01:** pheromone-write uses .signals | map() not |= map() to avoid pipe-to-object jq bug
- **05-01:** Rough epoch approximation in jq via string parsing (years*365d + months*30d) — sufficient for decay math
- **05-01:** Dual-write backward compat: FOCUS->constraints.json focus[], REDIRECT->constraints.json constraints[]
- **05-01:** Medium confirmation format — 3-4 lines, no banners (type, content preview, strength/expiry, active counts)
- **05-02:** Watcher prompts receive pheromone_section between file list and verification sections (same injection pattern as builders)
- **05-02:** Checkpoint polling is lightweight polling (check at natural breakpoints) not a formal queue — practical and zero-infrastructure
- **05-02:** Graceful degradation: pheromone-prime failure never blocks a build — pheromone_section defaults to empty string
- **05-03:** pheromone-expire sets active=false only — signals archived to midden, never deleted
- **05-03:** Phase_end expiry only in continue.md, never build.md — signals must survive builds
- **05-03:** Auto FEEDBACK strength 0.6 vs auto REDIRECT strength 0.7 — failures produce stronger signals
- **05-03:** Pause-aware TTL adds pause_duration to expires_at before comparison (macOS-safe epoch math)
- **05-03:** eternal-init is idempotent — safe to call on every /ant:continue invocation

## Decisions

- **06-01:** Source of truth seal.md already matched plan — only OpenCode copy needed rewriting
- **06-02:** Seal-first enforcement replaces old 3-precondition gate; belt-and-suspenders checks both milestone field and CROWNED-ANTHILL.md file
- **06-02:** Date-first naming (YYYY-MM-DD-goal) replaces old goal-timestamp format
- **06-02:** Full archive copies all data files including pheromones, dreams, CROWNED-ANTHILL.md; excludes backups/locks/midden/survey
- **06-02:** State reset now clears memory.instincts, memory.phase_learnings, memory.decisions (promoted to QUEEN.md first)
- **06-03:** Timeline uses date-first entries with milestone emoji indicators (crown/lock/circle)
- **06-03:** Detail view prioritizes CROWNED-ANTHILL.md display over manifest data; graceful fallback for older chambers

## Last Updated

2026-02-18 — Phase 6 COMPLETE (seal ceremony-only rewrite, entomb seal-first with full archive, tunnels timeline with seal doc display)
