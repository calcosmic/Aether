# Aether Colony System — Project State

**Project:** Aether Colony System
**Milestone:** v3.1 Open Chambers
**Core Value:** Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

---

## Current Position

| Field | Value |
|-------|-------|
| **Phase** | 11 — Foraging Specialization |
| **Plan** | 04 — CLI Override Integration |
| **Status** | Complete |
| **Last Action** | Completed Plan 04 - CLI --model flag support with task-based routing integration |

### Progress Bar

```
v3.0.0:  [██████████] 100% COMPLETE (14 plans, 25 requirements)
v3.1:    [████████░░░] 80% IN PROGRESS (4/4 plans complete, 13/27 requirements)
```

### Phase Status

| Phase | Name | Status | Requirements | Complete |
|-------|------|--------|--------------|----------|
| 9 | Caste Model Assignment | Complete | 8 | 100% (5/5 plans) |
| 10 | Entombment & Egg Laying | Complete | 5 | 100% (4/4 plans) |
| 11 | Foraging Specialization | Complete | 3 | 100% (3/3 plans) |
| 12 | Colony Visualization | Blocked | 11 | 0% |

---

## Project Reference

### Quick Links
- **Project:** `.planning/PROJECT.md`
- **Requirements:** `.planning/REQUIREMENTS.md`
- **Roadmap:** `.planning/ROADMAP.md`
- **Research:** `.planning/research/SUMMARY.md`

### Key Constraints
- **Tech Stack:** Node.js >= 16, Bash, jq — Minimal external dependencies
- **Distribution:** npm package (aether-colony)
- **Platform:** macOS/Linux, Claude Code and OpenCode support
- **State:** Repo-local only (no cloud dependencies)

### v3.0.0 Foundation (Completed)
- CLI installation and update system
- Core colony commands (init, build, continue, plan, phase)
- Worker caste system (Builder, Watcher, Scout, Chaos, Oracle)
- State management with Iron Law enforcement
- File locking infrastructure with PID-based stale detection
- Atomic write operations
- Safe checkpoint system with explicit allowlist
- 209+ tests (AVA unit + integration + E2E)

---

## Current Focus

### Phase 10: Entombment & Egg Laying

**Goal:** Users can archive completed colonies (entomb), start fresh colonies (lay eggs), browse history (explore tunnels), and see automatic milestone detection.

**Key Requirements:**
- LIFE-01: Entomb completed colony to chambers
- LIFE-02: Lay eggs (start fresh colony)
- LIFE-03: Explore tunnels (browse archived colonies)
- LIFE-04: Milestone auto-detection
- LIFE-05: Pheromone preservation (learnings carry forward)

**Success Criteria (What Must Be True):**
1. User runs `/ant:entomb` and colony is archived to `.aether/chambers/`
2. User runs `/ant:lay-eggs "new goal"` and fresh colony starts with preserved learnings
3. User runs `/ant:tunnels` and sees archived colony history
4. Milestone updates automatically based on progress
5. Learnings and decisions carry forward between colonies

**Blockers:** None

---

## Accumulated Context

### Decisions Made

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02-14 | 4 phases for v3.1 | Natural grouping: model routing (9), lifecycle (10), advanced routing (11), visualization (12) |
| 2026-02-14 | Phase 9 before 11 | Must verify basic routing works before building task-based routing on top |
| 2026-02-14 | Colors + emojis together | PROJECT.md explicitly requires both, not replacing each other |
| 2026-02-14 | Use proxyquire for mocking | Enables isolated unit testing of modules with fs/yaml dependencies |
| 2026-02-14 | Test both mock and real YAML | Unit tests use mocks for speed, integration test validates real config |
| 2026-02-14 | Use user_overrides section in model-profiles.yaml | Keeps all model configuration in one file, clear separation from defaults |
| 2026-02-14 | Show (override) indicator in list output | Users need to see which models are overridden vs defaults |
| 2026-02-14 | Include caste emojis in CLI output | Matches ant colony metaphor and improves scannability |
| 2026-02-14 | Use native fetch with AbortController | Node 18+ support, no external dependencies needed for proxy health |
| 2026-02-14 | Show ? when proxy is down during --verify | Distinguishes between "model not available" and "can't check" |
| 2026-02-14 | Dream timestamps extracted from filename | Consistent naming enables easy sorting and display |
| 2026-02-14 | Nestmate detection uses .aether/ directory heuristic | Simple and reliable way to identify Aether projects |
| 2026-02-14 | Cross-project TO-DOs limited to 5 items | Prevents overwhelming output |
| 2026-02-14 | Spawn tree format includes model as 6th field | Complete audit trail of which models are used per spawn |
| 2026-02-14 | Model parameter defaults to 'default' for backward compatibility | Existing spawn-log calls continue to work |
| 2026-02-14 | Use jq -Rs '.[:-1]' to strip trailing newlines | jq -Rs adds trailing newline which pollutes JSON output |
| 2026-02-14 | Entomb uses coffin emoji (⚰️) not urn (🏺) | Avoids visual conflict with seal command |
| 2026-02-14 | Compute version from phase counts | Automatic semantic versioning based on progress |
| 2026-02-14 | Preserve all learnings/decisions, filter instincts by confidence | Learnings validated; instincts need threshold |
| 2026-02-14 | Use chamber-list utility for tunnels command | Reuses existing JSON-returning utility for consistency |
| 2026-02-14 | Truncate goal at 50 chars in tunnels list view | Keeps display compact while showing enough context |
| 2026-02-14 | Detail view pattern with /command <name> | Consistent UX for single-item detail views |
| 2026-02-14 | Telemetry errors are silent | Spawn logging continues even if telemetry fails (graceful degradation) |
| 2026-02-14 | Routing decisions rotate at 1000 entries | Prevents unbounded file growth in telemetry.json |
| 2026-02-14 | Default telemetry command shows summary | Matches user expectations from tools like git status |
| 2026-02-14 | Color thresholds: green >=90%, yellow >=70%, red <70% | Clear visual indication of model performance |
| 2026-02-14 | Task routing default_model acts as catch-all | When no keywords match but default_model exists, source is 'task-routing' not 'caste-default' |
| 2026-02-14 | First-match wins in complexity_indicators | Iteration order determines priority; keywords in earlier categories take precedence |
| 2026-02-14 | Atomic writes for telemetry | Temp file + rename pattern prevents data corruption during concurrent writes |
| 2026-02-14 | CLI --model flag takes highest precedence | User intent for one-time override must be respected over all other routing |
| 2026-02-14 | Use Node.js library via bash heredoc for model selection | Reuses existing tested logic, avoids duplication between bash and JS |

### Open Questions

| Question | Blocking | Next Step |
|----------|----------|-----------|
| How does Task tool inherit environment variables? | Phase 11 | Empirical testing during Phase 9 |
| What keywords reliably indicate task complexity? | Phase 11 | Validate during implementation |
| Current proxy returns 401 — config issue? | Phase 9 | Investigate before MOD-03 |

### Known Issues (from Research)

1. **Model routing may not actually execute** — Configuration exists in YAML but execution path unverified
2. **Proxy authentication failures** — LiteLLM returns 401, needs investigation
3. **Environment variable inheritance** — Undocumented behavior in Task tool

---

## Session Continuity

### Last Session
- **Date:** 2026-02-14
- **Action:** Completed Plan 04 in Phase 11 - CLI Override Integration
- **Outcome:** Added --model flag support to /ant:build command; model-profile select/validate commands; 13 integration tests passing

### Next Actions
1. Phase 11 is now 100% complete (3/3 plans done)
2. Move to Phase 12 - Colony Visualization

### Handoff Notes
- Phase 11 Plan 04 complete - CLI --model flag support ready
- Users can run `/ant:build 1 --model glm-5` to override model for a build
- model-profile select command handles full precedence chain: CLI > user > task-routing > caste-default
- model-profile validate command validates model names against known models
- Next: Phase 12 (Colony Visualization)

---

## Performance Metrics

| Metric | v3.0.0 | v3.1 Target |
|--------|--------|-------------|
| Test Coverage | 286 tests (255 + 31 new) | Maintain + add routing tests |
| Commands | 5 core | +3 lifecycle +2 routing +1 viz |
| State Files | COLONY_STATE.json | + chambers/, manifests |
| Visualization | Basic status | Real-time immersive |

---

*State file: `.planning/STATE.md`*
*Updated: 2026-02-14*
*Next update: After Phase 12 start*
