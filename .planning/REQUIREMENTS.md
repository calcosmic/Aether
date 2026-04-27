# Requirements: Aether v1.10 Colony Polish

**Defined:** 2026-04-26
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.10 Requirements

### Smart Review Depth

- [ ] **DEPTH-01**: `resolveReviewDepth()` helper function determines light vs heavy review based on phase position, keyword detection, and `--light` flag
- [ ] **DEPTH-02**: `--light` flag on build and continue commands skips heavy agents (Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos) on intermediate phases
- [ ] **DEPTH-03**: Final phase always gets heavy review regardless of `--light` flag (non-negotiable)
- [ ] **DEPTH-04**: Phases with security/release keywords in name auto-detect as heavy (security, auth, crypto, secrets, permissions, compliance, audit, release, deploy, production, ship, launch)
- [ ] **DEPTH-05**: Continue playbooks skip heavy agents when depth is light — Weaver, Gatekeeper, Auditor, Medic, Probe each check review_depth
- [ ] **DEPTH-06**: Review depth displayed to user in wrapper output (e.g., "Review depth: light (Phase N of M — final phase will get full review)")

### Gate Failure Recovery

- [x] **GATE-01**: Verification gate failures show clear, actionable recovery instructions instead of just "FAILED" banner
- [x] **GATE-02**: Watcher Veto does not auto-stash work without explicit user confirmation
- [x] **GATE-03**: Re-running `/ant-continue` only re-checks previously failed gates, not all gates from scratch (incremental gate checking)

### Porter Ant

- [ ] **PORT-01**: Porter caste registered in all visual maps (emoji "📦", color "95", label "Porter", name prefixes) as 26th caste
- [ ] **PORT-02**: Porter agent definition exists across all 4 surfaces: `.claude/agents/ant/`, `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/`
- [ ] **PORT-03**: Porter is wired into seal lifecycle — after seal completes, Porter prompts user interactively with publish/push/deploy options (not a standalone command)
- [ ] **PORT-04**: `/ant-porter` slash command registered in YAML source and all platform wrappers
- [ ] **PORT-05**: Porter check subcommand reports pipeline alignment and readiness (pipeline docs found, goal alignment, blockers)

### Lifecycle Ceremony

- [x] **CERE-01**: seal blocks on active blockers (flags with blocker severity), warns on issues with `--force` override
- [x] **CERE-02**: seal promotes instincts with confidence >= 0.8 to Hive Brain via hive-promote (non-blocking — log failures but don't stop)
- [x] **CERE-03**: seal expires all FOCUS pheromones (phase-scoped) and preserves REDIRECT pheromones (hard constraints carry over)
- [x] **CERE-04**: seal enriches CROWNED-ANTHILL.md with learnings count, promoted instincts count, expired signals, flags resolved
- [x] **CERE-05**: init-research provides deeper codebase analysis: reads README.md, scans directory structure beyond top level, detects test frameworks, checks CI configs, reads key source files
- [ ] **CERE-06**: status dashboard shows runtime version line (e.g., "Runtime: v1.0.24") and one-line signal summary
- [ ] **CERE-07**: entomb extracts near-miss wisdom (confidence < 0.8 but >= 0.5), cleans temp files (spawn-tree, manifests, review artifacts), updates registry to inactive with final stats
- [ ] **CERE-08**: resume checks for stale FOCUS pheromones referencing completed phases and suggests review
- [ ] **CERE-09**: discuss/council analyzes codebase first, then asks comprehensive multiple-choice questions covering all angles (features, priorities, scope, trade-offs, architecture) — like GSD questioning pattern with 2-4 options per question and freeform allowed
- [ ] **CERE-10**: chaos auto-flags HIGH severity findings (suggests `aether flag "<finding>"`) and suggests REDIRECT for recurring midden patterns
- [ ] **CERE-11**: oracle suggests persisting high-value research findings as pheromone signals or hive wisdom entries
- [ ] **CERE-12**: patrol detects stale pheromones, verifies data file integrity (COLONY_STATE.json, pheromones.json, session.json are valid JSON), checks for interrupted builds (uncommitted manifests/spawn trees)

### Oracle Loop Fix

- [x] **ORCL-01**: Fix OpenCode worker callback URL — binary must not conflate LiteLLM proxy URL with agent messaging URL (separate config or auto-detect ACP server)
- [x] **ORCL-02**: Oracle has research brief formulation step that constructs a detailed research context before starting the iterative loop (like init-research formulates context for init)
- [x] **ORCL-03**: Oracle offers depth selection: quick (2 iterations, 60% confidence), balanced (4 iterations, 85%), deep (6 iterations, 95%), exhaustive (10 iterations, 99%) — user picks before research starts, defaults to balanced
- [x] **ORCL-04**: Oracle research state managed properly: configuration, research gaps, synthesis, and progress tracking persist across iterations

### Idea Shelving

- [ ] **SHELF-01**: Persistent shelf file (`.aether/data/shelf.json`) stores deferred ideas with trigger conditions and metadata
- [ ] **SHELF-02**: `/ant-seal` automatically shelves promising but unimplemented ideas from colony context (instincts < 0.8 confidence, unaddressed pheromones, user-mentioned ideas)
- [ ] **SHELF-03**: `/ant-init` surfaces relevant shelved ideas and lets user promote them to the new colony or defer again
- [ ] **SHELF-04**: Recurring REDIRECT pheromones (same content hash appearing across 2+ phases) get auto-shelved as permanent guidance
- [ ] **SHELF-05**: Shelved ideas survive colony entomb — archived to chambers, not lost

### QUEEN.md Fix

- [x] **QUEE-01**: Remove test junk data from `~/.aether/hive/wisdom.json` (domain: "test", text: "<repo> wisdom") and clean ~270 duplicate `<repo> wisdom` lines from `~/.aether/QUEEN.md`
- [x] **QUEE-02**: `appendEntriesToQueenSection()` in cmd/queen.go has dedup — checks if each entry already exists as a line in the target section before appending
- [x] **QUEE-03**: `queen-seed-from-hive` filters entries already present in QUEEN.md and reports count of new vs skipped entries
- [x] **QUEE-04**: colony-prime injects global QUEEN.md wisdom (`~/.aether/QUEEN.md`) alongside local wisdom — global wisdom currently not reaching workers
- [x] **QUEE-05**: colony-prime injects Philosophies and Anti-Patterns sections from QUEEN.md into worker context — currently only Wisdom and Patterns are extracted
- [x] **QUEE-06**: `queen-promote-instinct` writes to global `~/.aether/QUEEN.md` (not just local repo QUEEN.md) so promoted instincts reach all colonies
- [x] **QUEE-07**: High-confidence instincts (>= 0.8 confidence) auto-promoted to QUEEN.md at `/ant-seal` via queen-promote — currently requires explicit manual command

## v1.11 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Cross-Colony Review Sharing

- **CROSS-01**: Review findings shared across colonies via Hive Brain (generalized patterns only)
- **CROSS-02**: Auto-promotion of high-confidence finding patterns to Hive Brain instincts

### Advanced Automation

- **AUTO-01**: Auto-block on critical findings during continue flow
- **AUTO-02**: Automatic finding-to-pheromone promotion
- **AUTO-03**: Bulk resolve by domain or phase in `review-ledger-resolve`

## Out of Scope

| Feature | Reason |
|---------|--------|
| Cross-colony ledger sharing | Findings contain code-specific file paths and line numbers that go stale across repos |
| Ledger web UI | CLI-only for now; web dashboard is a future consideration |
| Porter as standalone publish tool | Porter is wired into lifecycle, not a separate CI/CD tool |
| Pheromone markets and reputation exchange | Not the next best move |
| Federation / inter-colony coordination | Not the next best move |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DEPTH-01 | Phase 58 | Pending |
| DEPTH-02 | Phase 58 | Pending |
| DEPTH-03 | Phase 58 | Pending |
| DEPTH-04 | Phase 58 | Pending |
| DEPTH-05 | Phase 58 | Pending |
| DEPTH-06 | Phase 58 | Pending |
| GATE-01 | Phase 59 | Complete |
| GATE-02 | Phase 59 | Complete |
| GATE-03 | Phase 59 | Complete |
| PORT-01 | Phase 61 | Pending |
| PORT-02 | Phase 61 | Pending |
| PORT-03 | Phase 61 | Pending |
| PORT-04 | Phase 61 | Pending |
| PORT-05 | Phase 61 | Pending |
| CERE-01 | Phase 62 | Complete |
| CERE-02 | Phase 62 | Complete |
| CERE-03 | Phase 62 | Complete |
| CERE-04 | Phase 62 | Complete |
| CERE-05 | Phase 62 | Complete |
| CERE-06 | Phase 63 | Pending |
| CERE-07 | Phase 63 | Pending |
| CERE-08 | Phase 63 | Pending |
| CERE-09 | Phase 64 | Pending |
| CERE-10 | Phase 64 | Pending |
| CERE-11 | Phase 64 | Pending |
| CERE-12 | Phase 64 | Pending |
| ORCL-01 | Phase 60 | Complete |
| ORCL-02 | Phase 60 | Complete |
| ORCL-03 | Phase 60 | Complete |
| ORCL-04 | Phase 60 | Complete |
| SHELF-01 | Phase 65 | Pending |
| SHELF-02 | Phase 65 | Pending |
| SHELF-03 | Phase 65 | Pending |
| SHELF-04 | Phase 65 | Pending |
| SHELF-05 | Phase 65 | Pending |
| QUEE-01 | Phase 57 | Complete |
| QUEE-02 | Phase 57 | Complete |
| QUEE-03 | Phase 57 | Complete |
| QUEE-04 | Phase 57 | Complete |
| QUEE-05 | Phase 57 | Complete |
| QUEE-06 | Phase 57 | Complete |
| QUEE-07 | Phase 57 | Complete |

**Coverage:**
- v1.10 requirements: 37 total
- Mapped to phases: 37
- Unmapped: 0

---
*Requirements defined: 2026-04-26*
*Last updated: 2026-04-26 after roadmap creation*
