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

## v2.1 Production Hardening (Shipped: 2026-03-24)

**Phases completed:** 8 phases (9-16), 39 plans, ~3 hours execution
**Timeline:** 2026-03-24 (single day)
**Last phase number:** 16

**Key accomplishments:**
- Error handling hardened: 438 suppression patterns classified, dangerous ones fixed, intentional ones documented
- Dead code deprecated: 18 unused subcommands marked with stderr warnings
- State API centralized: single facade for all COLONY_STATE.json access + verify-claims lie detector
- Monolith modularized: aether-utils.sh from 11,663 → 5,262 lines, 9 domain modules extracted
- Planning depth added: per-phase research scout + 16K research context in builder/watcher prompts
- Documentation accuracy: all docs match verified behavior, comprehensive v2.1 changelog
- Package validated and v2.1.0 installed to local hub

---

## v2.2 Living Wisdom (Shipped: 2026-03-25)

**Phases completed:** 4 phases (17-20), 5 plans
**Timeline:** 2026-03-24 to 2026-03-25

**Key accomplishments:**
- QUEEN.md template with 4 structured sections (User Preferences, Codebase Patterns, Build Learnings, Instincts)
- Continue playbooks auto-write build learnings and promote instincts to QUEEN.md
- Post-extraction wisdom filtering via _filter_wisdom_entries() prevents noise
- queen-seed-from-hive subcommand for seeding new colonies from cross-colony wisdom
- Split QUEEN WISDOM prompt sections: Global vs Colony-Specific with distinct headers

---

## v2.3 Per-Caste Model Routing (Shipped: 2026-03-27)

**Phases completed:** 4 phases (21-24), 10 plans
**Timeline:** 2026-03-27 (single day)

**Key accomplishments:**
- Test infrastructure refactored: centralized mock profile helper, zero hardcoded model strings
- Per-caste model routing: 8 opus castes configured with model frontmatter in agent definitions
- getModelSlotForCaste() + model-slot CLI with get/list/validate verbs
- Auto-resolved model slots in spawn-tree entries (opus/sonnet/inherit)
- GLM-5 loop risk warnings added to opus-caste agents
- Compact 3-column verify-castes table for both Claude and OpenCode

---

## v2.4 Living Wisdom (Shipped: 2026-03-27)

**Phases completed:** 4 phases (25-28), 8 plans
**Timeline:** 2026-03-27 (single day)

**Key accomplishments:**
- Oracle and Architect agent definitions created (model: opus, 24-agent roster)
- Oracle + Architect spawn steps wired into build-wave playbook
- Wisdom pipeline wiring: hive-promote in continue-advance, consolidated summary in continue-finalize
- Fuzzy dedup for instinct-create prevents duplicate wisdom entries
- Deterministic fallback: git-diff-based learning extraction when LLM extraction fails
- Wisdom Pipeline documented end-to-end in CLAUDE.md (7 stages with subcommands and thresholds)

---

## v2.5 Smart Init (Shipped: 2026-03-27)

**Phases completed:** 4 phases (29-32), 10 plans, 27 commits
**Timeline:** 2026-03-27 (single day)
**Changes:** 27 files, +6,727 / -538 lines
**New tests:** 50 (616+ total passing)

**Key accomplishments:**
- Repo scanning module (scan.sh): 10th domain module with tech stack detection, directory analysis, git history, survey staleness, and complexity estimation under 2 seconds
- Charter management: colony-name derivation and charter-write function populating existing QUEEN.md v2 sections with re-init safety
- Smart init rewrite: `/ant:init` now runs scan-assemble-approve-create flow with structured approval prompt from research data
- Intelligence enrichment: prior colony context from chambers, 10 deterministic pheromone suggestion patterns, governance inference from config files
- 50 new integration tests validating the full smart init pipeline (12/12 requirements satisfied)

---

