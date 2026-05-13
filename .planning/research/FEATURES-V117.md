# Feature Analysis: v1.17 Classic Restoration

**Researched:** 2026-05-13
**Sources:** Classic v5.4 comparison (4 research agents), stack research, architecture research, pitfalls research

---

## Feature Categories

### Category 1: TS Host Control Plane (Table Stakes)

These are infrastructure features required before anything else works.

| Feature | Status in v1.16 | What v1.17 Needs | Complexity |
|---------|----------------|------------------|------------|
| Worker dispatch | Simulated (100ms delay) | Real platform dispatch via Agent tool | Medium |
| Parallel waves | Sequential within waves | Concurrent worker execution per wave | Low |
| Error recovery | None | Retry logic, timeout, graceful fallback | Medium |
| Event bridge | Not built | Consume Go ceremony events from JSONL stream | Medium |
| Go → TS event streaming | Go writes JSONL, nobody reads | TS host tails JSONL, renders in real time | Low |

**Key insight from architecture research:** Three event-delivery mechanisms already exist in Go:
- Subprocess narrator pipe (`cmd/narrator_launcher.go`)
- JSONL tail + poll (`event-bus-subscribe --stream`)
- WebSocket/SSE server (`aether serve`)

Recommended: JSONL tail because it requires no background server, respects the boundary contract, and works within the existing wrapper→Go→TS call chain.

---

### Category 2: Ceremony Restoration (Core Differentiator)

These are the features that make Aether feel alive. They are the primary differentiator against generic build tools.

| Feature | v5.4 State | Current State | Restoration Path |
|---------|-----------|---------------|------------------|
| ASCII banners | Every command had hand-crafted banners | Minimal, Go-rendered | Restore to wrapper markdown with shared YAML config for caste identity |
| Crowned Anthill seal art | Full ASCII anthill drawing | Gone | Template file `.aether/templates/ceremony/seal-art.md` |
| Worker spawn notifications | Formatted: `━━━ 🏺🐜 A R C H A E O L O G I S T ━━━` | Generic spawn plan | Event-driven: Go emits `ceremony.build.spawn`, TS renders with template |
| Build summary block | Worker counts, tool usage, elapsed time | Partial in Go | Go provides data, wrapper applies template frame |
| Seal ceremony | Multi-step: Sage → Chronicler → wisdom review → art | Delegated to Go finalizer | Playbook that wrapper loads, with Go providing data |
| Survey loading display | `━━━ 🗺️🐜 S U R V E Y   L O A D E D ━━━` | Not present | Wrapper loads from template |
| Worker completion lines | `{emoji} {name}: {task} complete ✓` with trends | Minimal | Template applied by wrapper around Go data |
| Progress bars | Per-phase `[████████░░░░░░░░░░░░] N/M` | In Go `codex_visuals.go` | TS renders from event data |

**How others do it:**
- **Bazel:** Structured event protocol (BEP) streamed as protobuf, consumed by UIs. Clean separation of data from rendering.
- **Turborepo:** JSON task outputs with progress bars. Simple, not animated.
- **GitHub CLI (`gh run watch`):** Live terminal dashboard with spinners, job status, and logs. Uses `charmbracelet/huh` or similar.

**What to avoid:**
- Re-creating the "Go renders everything" trap (current state). Ceremonies must be editable.
- Hard-coding ceremony text in wrappers without shared config (creates Codex drift — Codex has no wrapper markdown).
- Heavy terminal UI frameworks (Ink adds 25+ deps, Blessed is unmaintained).

---

### Category 3: Swarm Display (Differentiator)

The live terminal dashboard was a distinctive v5.4 feature.

| Feature | v5.4 State | What v1.17 Needs |
|---------|-----------|------------------|
| Animated spinners | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` cycling per second | `ora` library in TS host |
| Per-ant progress bars | `[██████████░░░░░] 67%` with excavation phrases | `cli-progress` or custom ANSI bars |
| Tool usage counters | `📖5 🔍3 ✏️2 ⚡1` per worker | Track from Go worker events |
| Elapsed time | `(Xm Ys)` per ant | `Date.now()` diff in TS host |
| Token consumption | `🍯{tokens}` trophallaxis indicator | Event payload field |
| Chamber activity map | `🔥🔥🔥 (5 ants)` per zone | Derive from active worker list |
| Live refresh | `fswatch`/`inotifywait` 2-second polling | `chokidar` watching JSONL |
| Cycling status phrases | `excavating...`, `building...` | Config array in YAML, cycled by elapsed time |

**How others do it:**
- **k9s:** Full TUI with lists, detail views, live updates via Kubernetes watch API. Overkill for Aether.
- **Lazygit:** Terminal UI with panels, tabs, real-time git status. Uses `gocui` (Go), not relevant.
- **Docker Compose `up`:** Parallel service startup with color-coded logs and status. Closest analog, but simpler.

**What to avoid:**
- Full TUI framework (too heavy, too many deps).
- Running a persistent background process (complicates lifecycle).
- Trying to animate within wrapper markdown (impossible — wrappers are not long-running).

---

### Category 4: Queen Orchestration Intelligence (Core Differentiator)

These are the workflow patterns that made the Queen smart.

| Feature | v5.4 State | Current State | Restoration Path |
|---------|-----------|---------------|------------------|
| Workflow pattern selection | Queen selected from 6 patterns based on phase name | Generic dispatch regardless of phase type | TS host reads phase name, selects pattern, generates manifest accordingly |
| Builder-Probe Lock | Builders return `code_written`, only Probe upgrades to `completed` | `reconcileCompletedBuildTasks` marks all done | TS host enforces: after builder waves, spawn Probe for `code_written` tasks, Probe upgrades status |
| Tiered escalation | Worker retry → parent → Queen → user | Circuit breaker (different model) | TS host implements retry→reassignment→escalation chain, circuit breaker as hard stop |
| Intra-build midden checks | Auto-REDIRECT on recurring errors | Midden collected but not acted on mid-build | TS host checks midden threshold between waves, emits REDIRECT |
| Phase mode awareness | Discovery/Prototype/Production/Maintenance verification strictness | Verification depth control (light/standard/heavy) | TS host reads phase mode from manifest, maps to verification depth |
| Ambassador conditional spawn | Integration keywords trigger Ambassador pre-wave | `phaseNeedsAmbassador()` in Go dispatch | Keep Go detection, TS host spawns Ambassador with integration design prompt |

**How others do it:**
- **LangGraph:** Explicit workflow graphs with conditional edges. Good for deterministic flows, but Aether's pattern selection is LLM-driven.
- **OpenAI Agents JS:** Handoff system where one agent can transfer to another. Closest analog for tiered escalation.
- **Temporal:** Durable execution with retry policies, timeouts, and saga patterns. Relevant only if we need crash-proof long-running orchestration (not justified yet).

**What to avoid:**
- Full workflow engine adoption (too heavy, too much framework).
- Re-implementing Go's dispatch manifest generation in TS (manifest stays Go-owned).
- Moving pattern selection into Go (would make it compiled, not editable).

---

### Category 5: Oracle Behavioral Richness (Enhancement)

Restoring the RALF loop's intelligence.

| Feature | v5.4 State | Current State | Restoration Path |
|---------|-----------|---------------|------------------|
| Phase-aware prompts | Survey/investigate/synthesize/verify each got distinct directives | Flat agent prompt | Go controller injects phase directive into task brief |
| Diminishing returns | 3-iteration novelty delta plateau → force phase advance | Simpler confidence check | Go tracks novelty delta, forces advance |
| Template synthesis | Tech-eval, architecture-review, bug-investigation each got different sections | Generic synthesis | Go finalizer with template-aware output |
| Signal injection | Pheromones steered Oracle mid-research | Unclear connection | Go reads active pheromones, includes in task brief |
| Strategy modifiers | Breadth-first/depth-first/adaptive injected into prompt | Strategy field exists but unused | Go controller injects strategy note |

**How others do it:**
- **Pydantic AI:** Structured output with Zod schemas, multi-turn loops with retry. Good for Oracle response validation.
- **DSPy:** Prompt optimization and module compilation. Overkill — we want editable prompts, not compiled ones.
- **CrewAI:** Research crews with role-based agents. Aether's Oracle is single-agent with phase transitions, not multi-agent.

**What to avoid:**
- Moving Oracle prompts into compiled Go strings (defeats editability).
- Adopting a heavy research framework (DSPy, CrewAI) for a single command.

---

## Anti-Features (Do NOT Build)

| Feature | Why Not |
|---------|---------|
| Full TUI framework (Ink, Blessed) | Too heavy (25+ deps), unmaintained, React-for-CLI is overkill |
| WebSocket server for events | Requires background process; JSONL tail is simpler |
| Durable execution engine (Temporal) | Only justified if Oracle becomes truly long-running and resumable |
| Compiled ceremony rendering in Go | This is the problem we're solving |
| Multi-agent Oracle crew | Oracle is single-agent with phase transitions; crews add complexity |
| Real-time web dashboard | Out of scope per PROJECT.md |
| Cross-colony ledger sharing | Out of scope per PROJECT.md |

---

## Complexity Summary

| Area | Complexity | Risk | Confidence |
|------|-----------|------|------------|
| Event bridge (Go → TS) | Low | Low | High |
| Ceremony narrator (TS) | Medium | Low | High |
| Worker dispatch (real platform) | Medium | Medium | High |
| Swarm display (animated dashboard) | Medium | Medium | Medium |
| Workflow pattern selection | Low | Low | High |
| Builder-Probe Lock | Low | Low | High |
| Oracle phase-aware prompts | Low | Low | High |
| Tiered escalation | Medium | Medium | High |
| Parallel wave dispatch | Low | Low | High |
| Golden parity tests | Medium | Low | High |

---

## Build Order

1. **Foundation:** Event bridge + ceremony narrator + shared YAML config
2. **Dispatch:** Real worker dispatch + parallel waves + error recovery
3. **Orchestration:** Workflow patterns + Builder-Probe Lock + escalation
4. **Display:** Swarm dashboard + live refresh
5. **Oracle:** Phase-aware prompts + diminishing returns + template synthesis
6. **Integration:** Wire into lifecycle.ts + build.md wrapper
7. **Verification:** Golden parity tests against v5.4 baseline
