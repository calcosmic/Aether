# Requirements: Aether

**Defined:** 2026-03-24 (updated 2026-03-27)
**Core Value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.

## v2.3 Requirements — Per-Caste Model Routing

### Test Infrastructure (Prerequisite)

- [ ] **TEST-01**: All test files use a centralized mock profile helper instead of inline model name constants
- [ ] **TEST-02**: Mock profile helper reads caste-to-model mappings from model-profiles.yaml (single source of truth)
- [ ] **TEST-03**: Test suite passes without modification when model-profiles.yaml castes change

### Core Routing

- [ ] **ROUTE-01**: Reasoning castes (queen, archaeologist, route-setter, sage, tracker, auditor, gatekeeper, measurer) use the `opus` model slot via agent frontmatter
- [ ] **ROUTE-02**: Execution castes (builder, watcher, scout, chaos, probe, weaver, ambassador, nest, disciplines, pathogens, provisions) use the `sonnet` model slot via agent frontmatter
- [ ] **ROUTE-03**: Surveyor castes (nest, disciplines, pathogens, provisions) use the `sonnet` model slot via agent frontmatter (subset of ROUTE-02, kept for explicit traceability)
- [ ] **ROUTE-04**: Inherit castes (chronicler, includer, keeper) use `inherit` (unchanged from current behavior)
- [ ] **ROUTE-05**: `ANTHROPIC_DEFAULT_OPUS_MODEL` in settings.json.3model maps to `glm-5`
- [ ] **ROUTE-06**: model-profiles.yaml `worker_models` reflects the two-tier caste split (opus castes vs sonnet castes)
- [ ] **ROUTE-07**: Dual-mode operation — same Aether routing code works when user switches between Claude API and GLM proxy
- [ ] **ROUTE-08**: workers.md no longer claims per-caste routing is impossible

### Tooling & Overrides

- [ ] **TOOL-01**: `getModelSlotForCaste(profiles, caste)` function in bin/lib/model-profiles.js returns the correct slot name per caste
- [ ] **TOOL-02**: `model-slot get <caste>` subcommand in aether-utils.sh resolves a caste to its model slot
- [ ] **TOOL-03**: `validateSlot()` function validates slot names (opus, sonnet, haiku, inherit)
- [ ] **TOOL-04**: `/ant:build <phase> --model opus` CLI flag overrides the default slot for all workers in that build

### Safety & Verification

- [ ] **SAFE-01**: spawn-tree.txt output includes the model slot used for each spawned worker
- [ ] **SAFE-02**: `/ant:verify-castes` displays the model slot assignment per caste (not just the model name)
- [ ] **SAFE-03**: Reasoning caste agents include a safety note about GLM-5 loop risk in their definitions
- [ ] **SAFE-04**: Config swap workflow (Claude API <-> GLM proxy) documented in workers.md or verify-castes.md

### Future (Deferred to v2.4+)

- Per-profile caste assignments (deep/default/fast like GSD quality/balanced/budget)
- Runtime profile switching (`/ant:set-profile fast`)
- Task complexity-based auto-routing
- Model usage tracking per phase in `/ant:status`
- OpenCode agent frontmatter `model:` field parity
- Application-level GLM-5 loop detection (timeout/watchdog)

### Out of Scope

- Per-worker environment variable injection — Claude Code Task tool does not support this (archived v1 failure)
- Model name routing in Aether code — Aether routes by slot, never by actual model name
- Cost-based automatic model selection — no billing data available in proxy

## v2.2 Requirements (Completed)

### QUEEN.md Learning

- [x] **QUEEN-01**: Colony writes learnings to local QUEEN.md after builds
- [x] **QUEEN-02**: High-confidence instincts promote to local QUEEN.md wisdom section
- [x] **QUEEN-03**: Local QUEEN.md wisdom injected into builder/watcher prompts

### Hive Brain

- [x] **HIVE-01**: /ant:seal promotes high-confidence instincts to global hive
- [x] **HIVE-02**: /ant:init reads relevant hive wisdom for new colonies
- [x] **HIVE-03**: Hive wisdom is domain-scoped

### Global QUEEN.md

- [x] **HUB-01**: Global QUEEN.md accumulates cross-cutting wisdom
- [x] **HUB-02**: Colony-prime reads and injects global QUEEN.md wisdom

## v2.1 Requirements (Completed)

### Reliability
- [x] **REL-01** through **REL-09**: Error handling, state integrity, context trimming

### Quality
- [x] **QUAL-01** through **QUAL-09**: Dead code, state API, modularization, verification

### UX
- [x] **UX-01** through **UX-07**: Research depth, docs accuracy, clean install

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| TEST-01 | 21 | Pending |
| TEST-02 | 21 | Pending |
| TEST-03 | 21 | Pending |
| ROUTE-01 | 22 | Pending |
| ROUTE-02 | 22 | Pending |
| ROUTE-03 | 22 | Pending |
| ROUTE-04 | 22 | Pending |
| ROUTE-05 | 22 | Pending |
| ROUTE-06 | 22 | Pending |
| ROUTE-07 | 22 | Pending |
| ROUTE-08 | 22 | Pending |
| TOOL-01 | 23 | Pending |
| TOOL-02 | 23 | Pending |
| TOOL-03 | 23 | Pending |
| TOOL-04 | 23 | Pending |
| SAFE-01 | 24 | Pending |
| SAFE-02 | 24 | Pending |
| SAFE-03 | 24 | Pending |
| SAFE-04 | 24 | Pending |

**Coverage:**
- v2.3 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0

---
*Requirements updated: 2026-03-27*
