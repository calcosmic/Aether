# Phase 58: Smart Review Depth - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Add auto/light/heavy review modes so intermediate phases get fast review while final phases and security-sensitive phases always get full review. The `resolveReviewDepth()` helper determines depth based on phase position, keyword detection, and flags. Both build and continue commands respect review depth. Workers know their depth and adapt accordingly.

</domain>

<decisions>
## Implementation Decisions

### Light Mode Agent Scope
- **D-01:** Light mode runs Watcher only — the 7 heavy agents (Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos) are skipped on intermediate phases
- **D-02:** Heavy mode runs all agents (Watcher + all 7 heavy agents) — this is the full review gauntlet
- **D-03:** Chaos has 30% random sampling on light phases (deterministic by phase number hash — same phase always gets Chaos or not across runs)
- **D-04:** Chaos always runs on heavy phases (final phase + security/release keyword phases)

### Final Phase Detection
- **D-05:** Final phase = last entry in the COLONY_STATE.json phases array. Simple, deterministic, auto-adjusts when phases are inserted (e.g., inserting 58.1 shifts the "final" boundary)

### Flag Design
- **D-06:** `--light` flag forces light review on any phase (except cannot override final phase — DEPTH-03)
- **D-07:** `--heavy` flag forces heavy review on any phase — gives user full control. Symmetrical with `--light`
- **D-08:** Without either flag, depth is auto-detected: heavy if final phase or security/release keywords, light otherwise

### Keyword Detection
- **D-09:** Case-insensitive substring matching on the phase name for 12 keywords: security, auth, crypto, secrets, permissions, compliance, audit, release, deploy, production, ship, launch
- **D-10:** Keyword list is hardcoded in the Go runtime (not configurable) — simple, no new config surface

### Depth Display
- **D-11:** Review depth shown in wrapper output only, not runtime dispatch output
- **D-12:** Format: "Review depth: light (Phase 3 of 7 — final phase gets full review)" or "Review depth: heavy (Phase 7 of 7 — final phase)"

### Worker Depth Awareness
- **D-13:** Workers receive their review depth in colony-prime context: "Light review — core verification only" or "Heavy review — full quality gauntlet"
- **D-14:** This context lets agents adapt their thoroughness — light mode Watcher focuses on core correctness, heavy mode Watcher does comprehensive verification

### Claude's Discretion
- Exact hash function for deterministic Chaos sampling (simple modulo is fine)
- Exact phrasing of the worker depth context injection
- How resolveReviewDepth integrates with the existing normalizedBuildDepth system
- Whether review_depth gets stored in COLONY_STATE.json or computed on-the-fly each time

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Review Depth Requirements
- `.planning/REQUIREMENTS.md` — DEPTH-01 through DEPTH-06 requirements with acceptance criteria
- `.planning/ROADMAP.md` — Phase 58 success criteria (lines 144-154)

### Build Depth System (existing)
- `cmd/codex_build.go` — `normalizedBuildDepth()` (line 859), depth-based dispatch logic (lines 580-654), Chaos spawn (line 650)
- `.aether/docs/command-playbooks/build-verify.md` — Chaos spawning in build verification
- `.aether/docs/command-playbooks/build-full.md` — Full depth build with Chaos dispatch

### Continue Review System (existing)
- `cmd/codex_continue.go` — `codexContinueReviewSpecs` (lines 793-808), `plannedContinueReviewDispatches()` (lines 892-916) — always spawns all 3 review agents today
- `.aether/docs/command-playbooks/continue-gates.md` — Continue gate verification flow
- `.aether/docs/command-playbooks/continue-full.md` — Full continue playbook with all review agents

### Colony-Prime Context (for depth awareness injection)
- `cmd/colony_prime_context.go` — `buildColonyPrimeOutput()` (lines 333-768) — where worker context gets assembled

### Wrapper Surfaces
- `.claude/commands/ant/build.md` — Build wrapper (where --light/--heavy flags and depth display get added)
- `.claude/commands/ant/continue.md` — Continue wrapper (where depth display gets added)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `normalizedBuildDepth()` in `cmd/codex_build.go` — already normalizes depth strings, can be extended for review depth
- `codexContinueReviewSpecs` in `cmd/codex_continue.go` — already defines review agent list, can be filtered by depth
- Colony-prime section system in `cmd/colony_prime_context.go` — inject review depth as a new section
- COLONY_STATE.json phases array — already tracks all phases, last element = final phase

### Established Patterns
- Depth-based dispatch already exists for build (full/deep/standard controls specialist spawning)
- Review agents are defined as specs (caste, name, task, domains) and dispatched in waves
- Wrapper markdown handles user-facing output, Go runtime handles dispatch logic
- Flags on build/continue commands follow Cobra patterns (--flag with boolean)

### Integration Points
- Build dispatch: extend `normalizedBuildDepth` or create parallel `resolveReviewDepth`
- Continue dispatch: filter `codexContinueReviewSpecs` based on computed review depth
- Colony-prime: add review_depth section to worker context
- Wrapper markdown: add depth display line after dispatch manifest generation

</code_context>

<specifics>
## Specific Ideas

- The 7 heavy agents to skip in light mode: Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos
- Currently Chaos is only spawned at "full" depth in builds — it's been dormant because the default is "standard"
- The user specifically noticed Chaos not running recently — this phase makes Chaos a regular part of the lifecycle again
- "Review depth: light (Phase 3 of 7 — final phase gets full review)" format gives users confidence that full review is coming

</specifics>

<deferred>
## Deferred Ideas

- Configurable keyword list for auto-heavy detection — hardcoding is simpler for now
- Random (non-deterministic) Chaos sampling — deterministic is better for reproducibility
- Depth-based time budgets for agents (e.g., light mode gets 30s per agent, heavy gets 60s) — could be a future enhancement

</deferred>

---

*Phase: 58-smart-review-depth*
*Context gathered: 2026-04-27*
