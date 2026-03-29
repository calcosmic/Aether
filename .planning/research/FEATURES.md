# Feature Research: Colony Depth Selector (v2.6 Bugfix & Hardening)

**Domain:** Multi-agent colony orchestration -- build-time caste selection, spawn optimization, and depth/verbosity control for the Aether build pipeline
**Researched:** 2026-03-29
**Confidence:** HIGH (based on direct codebase analysis of all build playbooks, plan.md depth system, spawn system, and existing CLI flag patterns; web search partially rate-limited but CLI UX patterns are well-established domain knowledge)

---

## Context

### The Problem

Every `/ant:build` currently spawns a fixed set of castes regardless of phase complexity:

| Caste | Spawn Trigger | Model Slot | Always Spawns? |
|-------|--------------|------------|----------------|
| Builder | Task count (N) | sonnet | Yes (1-4 per wave) |
| Watcher | After all waves | sonnet | Yes (always 1) |
| Chaos | After watcher | sonnet | Yes (always 1) |
| Oracle | Before waves | **opus** | Yes (always 1) |
| Architect | After Oracle, before waves | **opus** | Yes (always 1) |
| Archaeologist | Existing files being modified | opus | Conditional |
| Ambassador | External integration keywords | sonnet | Conditional |
| Measurer | Performance keywords in phase name | sonnet | Conditional |

**The Oracle and Architect always run.** They each use the expensive opus model slot. They are "non-blocking" (failure produces a warning), but they are never *skipped*. For a simple 2-task phase like "fix a typo in the README" or "add a missing import", the user pays for 2 opus-tier spawns (Oracle research + Architect design) that produce research/design documents nobody reads.

The spawn plan header in build-wave.md line 61 says it all:
```
Total: {N} Builders + 1 Watcher + 1 Chaos + 1 Oracle + 1 Architect = {N+4} spawns
```

Minimum spawn count for any build: **5 agents** (1 builder + 1 watcher + 1 chaos + 1 oracle + 1 architect). For a trivial task, that is 3 unnecessary opus-tier spawns.

### The User Need

Users want a way to say "I know this phase is simple, skip the expensive research." Or conversely, "This is a critical phase, I want every caste involved." This is a depth/effort selector, not just a verbosity toggle.

### Existing Precedent: Planning Depth

The `/ant:plan` command already implements a depth selector with 4 named levels:

```bash
/ant:plan --fast        # target_confidence=80, max_iterations=4
/ant:plan --balanced    # target_confidence=90, max_iterations=6
/ant:plan --deep        # target_confidence=95, max_iterations=8
/ant:plan --exhaustive  # target_confidence=99, max_iterations=12
```

This is the canonical pattern. The build depth selector should follow the same naming convention for consistency.

---

## Table Stakes

Missing any of these = depth selector feels incomplete or confusing.

| # | Feature | Why Expected | Complexity | Notes |
|---|---------|--------------|------------|-------|
| T1 | **Named depth levels matching plan.md convention** | Users who know `--fast`/`--balanced`/`--deep`/`--exhaustive` from plan will expect the same names in build. Inconsistent naming creates cognitive load. | LOW | Reuse the same 4 names. Map each to a specific set of castes. |
| T2 | **Default to "standard" (current behavior minus Oracle)** | The quality gate states: "Must default to 'standard' (current behavior minus Oracle)." This means the default build drops Oracle but keeps everything else. Existing colonies must not change behavior unless the user opts in. | LOW | Define "standard" as: Builders + Watcher + Chaos + Archaeologist (conditional) + Ambassador (conditional) + Measurer (conditional). NO Oracle, NO Architect. |
| T3 | **Per-build CLI flag override** | Users must be able to override depth for a single build without changing colony defaults. This is the primary use case: "this one phase is simple." | LOW | `--depth minimal|standard|deep|full` or `--minimal`, `--standard`, `--deep`, `--full` as shorthand flags. |
| T4 | **Spawn plan header reflects active depth** | The spawn plan at build-wave.md Step 5 must show which castes are participating and which are skipped. Users need to see what depth they got. | LOW | Update the "Total: N spawns" line to show the depth level and list skipped castes. |
| T5 | **BUILD SUMMARY includes depth level** | The build summary in build-complete.md Step 7 should include the depth level used, so users can see it in logs and handoff documents. | LOW | Add `depth_level` to the synthesis JSON and display it in the summary header. |
| T6 | **No behavior change for existing colonies that don't use the flag** | Colonies that do not pass `--depth` must continue to work exactly as they do today. The default must be safe and backward-compatible. | LOW | If no `--depth` flag is passed, use "standard" (T2). This is a change from current behavior (which always spawns Oracle+Architect), but it is the requested default. |

---

## Differentiators

Features that make the depth selector feel smart rather than just a filter.

| # | Feature | Value Proposition | Complexity | Notes |
|---|---------|-------------------|------------|-------|
| D1 | **Colony-level depth preference stored in COLONY_STATE.json** | Users set their preferred depth once (`/ant:depth standard`) and all subsequent builds use it unless overridden. Avoids repeating `--depth standard` on every build. | LOW | Store `build_depth` field in COLONY_STATE.json. `/ant:depth` command to set/display. Per-build `--depth` flag overrides the colony default. |
| D2 | **Phase-type auto-suggestion** | When the user does not specify depth, the Queen analyzes the phase name and task descriptions and suggests a depth. A phase named "Fix typo in README" gets "minimal" suggested. A phase named "Design authentication system" gets "deep" suggested. | MEDIUM | Keyword matching on phase name + task descriptions. Display suggestion but do not auto-apply (user must confirm or pass the flag). |
| D3 | **Cost/time estimate per depth level** | Before building, show an estimated cost impact: "Standard depth: ~5 spawns. Full depth: ~8 spawns (+3 opus-tier)." Users can make informed tradeoffs. | LOW | Static estimates per caste (opus = expensive, sonnet = standard). Display in spawn plan header. |
| D4 | **Continue flow respects build depth** | The `/ant:continue` flow spawns Gatekeeper, Auditor, and Probe agents. These should be gated by the same depth level used during the build. "Minimal" builds should skip Gatekeeper and Probe. | MEDIUM | Pass depth level through to continue playbooks via COLONY_STATE.json or cross-stage state. |
| D5 | **Autopilot depth consistency** | `/ant:run` should use the same depth level across all phases in the autopilot loop. Changing depth mid-autopilot should pause and ask. | LOW | Read depth from colony preference on autopilot start. If per-build override was used, apply to all phases in the run. |

---

## Anti-Features

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Per-caste toggles (`--no-oracle --no-architect --no-chaos`) | Granular control over exactly which castes spawn | Explodes the API surface. Users must understand what each caste does to make intelligent decisions. If 4 depth levels are not enough, the design is wrong. The depth levels should be the right abstraction. | Named depth levels that group castes logically. If a user truly needs per-caste control, they can use the full depth and accept the cost. |
| Depth stored in QUEEN.md or user preferences | Persist across colonies | Depth is a colony-level concern, not a user preference. A user working on a simple typo-fix colony and a complex architecture colony should have different defaults. | Colony-level storage in COLONY_STATE.json (D1). |
| Auto-detect optimal depth from task complexity | "Smart" defaults that skip the user | The Queen would need to classify every phase, and misclassification is costly. If it skips Oracle on a phase that needed it, the user loses research context. False negatives are worse than false positives for depth. | Suggestion only (D2) -- display the recommendation but never auto-apply. User decides. |
| Depth affects spawn depth (recursion limit) | The word "depth" already means spawn recursion depth in workers.md (0=Queen, 1=Prime, 2=Specialist, 3=Deep Specialist) | Conflicting terminology. "Build depth" (which castes participate) is a different concept from "spawn depth" (how many levels of sub-spawning are allowed). Using the same word for both creates confusion. | Use "build depth" or "colony depth" for caste selection. Keep "spawn depth" for the existing recursion concept. The build depth levels can be called "modes" or "tiers" internally to avoid collision. |
| Depth changes mid-build | User wants to upgrade from minimal to full after seeing initial results | Mid-build depth changes require the Queen to spawn castes that should have run before waves started (Oracle before builders, Archaeologist before build). Changing depth mid-build breaks the temporal ordering of caste spawns. | Depth must be set before the build starts. If the user wants more analysis, they run `/ant:oracle` or `/ant:archaeology` manually after the build. |

---

## Feature Dependencies

```
[T1: Named depth levels matching plan.md]
    └──enables──> [T3: Per-build CLI flag]
                    └──enables──> [T4: Spawn plan header reflects depth]
                                    └──enables──> [T5: BUILD SUMMARY includes depth]

[T2: Default to "standard"]
    └──enables──> [T6: No behavior change without flag]

[D1: Colony-level depth preference]
    └──requires──> [T1: Named depth levels]
    └──requires──> [T3: Per-build CLI flag]
    └──enables──> [D5: Autopilot depth consistency]

[D2: Phase-type auto-suggestion]
    └──requires──> [T1: Named depth levels]
    └──enhances──> [T3: Per-build CLI flag]

[D4: Continue flow respects depth]
    └──requires──> [T1: Named depth levels]
    └──requires──> [D1: Colony-level preference or T3 cross-stage state]
```

### Dependency Notes

- **T1 is the foundation.** Everything else depends on defining what the depth levels mean (which castes participate at each level).
- **T2 and T3 form the core UX.** Default + override is the minimum viable depth selector.
- **D1 adds persistence.** Without D1, users must pass `--depth` on every build. With D1, they set it once.
- **D4 extends to continue.** The continue playbooks (continue-verify, continue-gates) spawn Gatekeeper, Auditor, Probe. These should respect the same depth setting. This is the largest implementation effort because it touches multiple playbook files.
- **D2 is independent polish.** Auto-suggestion is nice but not required for the depth selector to work.

---

## Depth Level Definitions

This is the core design decision. Each level maps to a specific set of castes:

### Level 0: `minimal`

**Use for:** Trivial phases (typos, config changes, single-line fixes)

| Caste | Included? | Rationale |
|-------|-----------|-----------|
| Builder | Yes | Must have someone to do the work |
| Watcher | Yes | Independent verification is non-negotiable |
| Chaos | No | Trivial phases don't need resilience testing |
| Oracle | No | No research needed for trivial changes |
| Architect | No | No design needed for trivial changes |
| Archaeologist | No | Trivial phases rarely modify existing code |
| Ambassador | No | No external integration in trivial phases |
| Measurer | No | Not performance-sensitive |

**Minimum spawns:** 1-2 (builders) + 1 (watcher) = 2-3 total
**Model cost:** sonnet-tier only (cheapest possible build)

### Level 1: `standard` (DEFAULT)

**Use for:** Most phases. Current behavior minus Oracle/Architect.

| Caste | Included? | Rationale |
|-------|-----------|-----------|
| Builder | Yes | Core implementation |
| Watcher | Yes | Mandatory independent verification |
| Chaos | Yes | Resilience testing catches edge cases |
| Oracle | No | Research is valuable but expensive; opt-in via `--deep` |
| Architect | No | Design is valuable but expensive; opt-in via `--deep` |
| Archaeologist | Conditional | Only when modifying existing files |
| Ambassador | Conditional | Only when external integration detected |
| Measurer | Conditional | Only when performance keywords detected |

**Typical spawns:** 2-4 (builders) + 1 (watcher) + 1 (chaos) + 0-1 (archaeologist) = 4-7 total
**Model cost:** sonnet-tier + conditional opus-tier (archaeologist only)

### Level 2: `deep`

**Use for:** Complex phases, architectural changes, phases where research matters.

| Caste | Included? | Rationale |
|-------|-----------|-----------|
| Builder | Yes | Core implementation |
| Watcher | Yes | Mandatory |
| Chaos | Yes | Resilience testing |
| Oracle | Yes | Research context improves builder decisions |
| Architect | Yes | Design context improves builder decisions |
| Archaeologist | Conditional | Existing file modification |
| Ambassador | Conditional | External integration |
| Measurer | Conditional | Performance sensitivity |

**Typical spawns:** 2-4 (builders) + 1 (watcher) + 1 (chaos) + 1 (oracle) + 1 (architect) + 0-1 (archaeologist) = 6-9 total
**Model cost:** 2 opus-tier (oracle, architect) + sonnet-tier

### Level 3: `full`

**Use for:** Critical phases, release candidates, phases where maximum assurance is needed. "Everything the colony can do."

| Caste | Included? | Rationale |
|-------|-----------|-----------|
| Builder | Yes | Core implementation |
| Watcher | Yes | Mandatory |
| Chaos | Yes | Resilience testing |
| Oracle | Yes | Maximum research |
| Architect | Yes | Maximum design |
| Archaeologist | Conditional | Existing file modification |
| Ambassador | Conditional | External integration |
| Measurer | Conditional | Performance sensitivity |

**Wait -- `full` and `deep` look the same?** Correct. The difference is that `full` forces all conditional castes ON:

| Caste | `deep` | `full` |
|-------|--------|--------|
| Archaeologist | Conditional (existing files) | **Always** (even for new-file-only phases, runs a lighter "context scan") |
| Ambassador | Conditional (keywords) | **Always** (runs a lighter "integration readiness check") |
| Measurer | Conditional (performance keywords) | **Always** (runs baseline measurement regardless) |

**Typical spawns:** 2-4 (builders) + 1 (watcher) + 1 (chaos) + 1 (oracle) + 1 (architect) + 1 (archaeologist) + 0-1 (ambassador) + 1 (measurer) = 8-11 total
**Model cost:** 3+ opus-tier + sonnet-tier

### Summary Table

| Level | Builders | Watcher | Chaos | Oracle | Architect | Archaeologist | Ambassador | Measurer | Min Spawns | Opus Spawns |
|-------|----------|---------|-------|--------|-----------|---------------|------------|----------|------------|-------------|
| `minimal` | Yes | Yes | - | - | - | - | - | - | 2 | 0 |
| `standard` | Yes | Yes | Yes | - | - | cond | cond | cond | 4 | 0-1 |
| `deep` | Yes | Yes | Yes | Yes | Yes | cond | cond | cond | 6 | 2-3 |
| `full` | Yes | Yes | Yes | Yes | Yes | **Yes** | **Yes** | **Yes** | 8 | 3+ |

### Relationship to Plan Depth

| Plan Depth | Build Depth Recommendation | Rationale |
|------------|---------------------------|-----------|
| `fast` | `minimal` or `standard` | Fast plan = simple project = less need for research |
| `balanced` | `standard` | Default plan = default build |
| `deep` | `standard` or `deep` | Deep plan = more complex, research may help |
| `exhaustive` | `deep` or `full` | Exhaustive plan = maximum effort justified |

This is a suggestion only (D2). The user always decides.

---

## CLI Interface Design

### Flag Syntax

```bash
# Shorthand flags (one per level)
/ant:build 1 --minimal       # level 0
/ant:build 1 --standard      # level 1 (default)
/ant:build 1 --deep          # level 2
/ant:build 1 --full          # level 3

# Explicit parameter
/ant:build 1 --depth minimal
/ant:build 1 --depth standard
/ant:build 1 --depth deep
/ant:build 1 --depth full

# No flag = colony default (stored in COLONY_STATE.json)
# If no colony default = "standard"
```

### Colony Preference Command

```bash
# Set colony-level default
/ant:depth standard

# Display current setting
/ant:depth

# Output:
# Colony build depth: standard (Builders + Watcher + Chaos)
# Override per-build with: --depth <minimal|standard|deep|full>
```

### Autopilot Integration

```bash
# Autopilot respects colony depth
/ant:run

# Override for entire autopilot run
/ant:run --depth deep
```

### Interaction with Existing Flags

```bash
# Depth is orthogonal to existing flags
/ant:build 1 --depth minimal --verbose
/ant:build 1 --depth deep --no-visual
/ant:build 1 --depth standard --model opus

# --model overrides model slots but does NOT change which castes spawn
# --depth controls WHICH castes, --model controls WHAT MODEL they use
```

---

## Mid-Colony Depth Changes

### Scenario: User changes depth between Phase 3 and Phase 4

**This is allowed and safe.** Depth is a per-build setting, not a colony invariant. Changing depth mid-colony:

1. Does NOT invalidate previous phase results
2. Does NOT require re-planning
3. Does NOT affect pheromones, instincts, or learnings
4. Simply changes which castes participate in the next build

### Scenario: User changes depth mid-autopilot

**This should pause the autopilot.** The autopilot reads depth at start and uses it consistently across all phases. If the user changes the colony depth preference while autopilot is running, the next phase will use the new depth. This is acceptable because depth changes between phases are safe (see above).

### Scenario: User passes `--depth minimal` for Phase 3 but the phase needs Oracle research

**The user takes responsibility.** If they chose `minimal` and the phase turns out to be more complex than expected, the build will complete without Oracle/Architect context. The builder will still have pheromone signals, colony-prime context, and phase research (if available from plan). The user can:

1. Rebuild the phase with `--depth deep` if results are insufficient
2. Run `/ant:oracle` manually after the build for targeted research
3. Accept the results as-is

---

## Edge Cases

### Edge Case 1: Oracle research file exists but depth is minimal

If a prior build (or plan step) created `.aether/data/research/oracle-{phase}.md` but the current build uses `--depth minimal`, the research file still exists. The builder prompt construction in build-wave.md checks for this file and injects `research_context` if present. **This is correct behavior** -- if research already exists, use it regardless of depth. Depth controls whether new research is *generated*, not whether existing research is *consumed*.

### Edge Case 2: Archaeologist already ran (existing files) but depth is minimal

The Archaeologist spawn is conditional on "existing file modification detected" (build-context.md Step 4.1). At `minimal` depth, the Archaeologist step is skipped entirely. The `archaeology_context` variable will be empty, and builders will not receive archaeology context. **This is the correct tradeoff** -- minimal means minimal. If archaeology matters, use `standard` or higher.

### Edge Case 3: Continue flow spawns Gatekeeper at minimal depth

The `/ant:continue` flow spawns Gatekeeper (security) and Auditor (quality) after verification. These are separate from build depth because they serve a different purpose: post-build quality assurance, not build-time caste selection.

**Recommendation:** Gatekeeper and Auditor should NOT be gated by build depth. They are safety nets. The depth selector controls how much *effort* goes into the build, not how much *safety* wraps it.

However, Probe (coverage analysis in continue-verify.md) should be gated:
- `minimal`: Skip Probe
- `standard`: Probe as normal (conditional on coverage < 80%)
- `deep`: Probe always runs
- `full`: Probe always runs with extended analysis

### Edge Case 4: --depth conflicts with --model override

No conflict. `--depth` controls which castes spawn. `--model` controls which model slot they use. They are orthogonal:
- `--depth minimal --model opus` = spawn minimal castes, but use opus for all of them
- `--depth full --model haiku` = spawn all castes, but use haiku for all of them (cheap but thorough)

### Edge Case 5: First build after init has no colony depth preference

When `COLONY_STATE.json` does not have a `build_depth` field (fresh colony, or colony from before v2.6), default to `standard`. This is the same as not passing `--depth` at all.

---

## Impact on Existing Behavior

### What Changes by Default

Before v2.6, every build spawns Oracle + Architect. After v2.6 with the `standard` default, these are NOT spawned unless the user explicitly requests `--deep` or `--full`.

**This is a behavioral change.** Users who relied on Oracle research and Architect design being present in every build will need to either:
1. Set their colony depth to `deep`: `/ant:depth deep`
2. Pass `--depth deep` on individual builds

**Mitigation:** The first time a user runs `/ant:build` after upgrading to v2.6, display a one-time notice:

```
Build depth: standard (new in v2.6)
Oracle and Architect are no longer spawned by default.
Use --depth deep to include them, or /ant:depth deep to set as default.
```

This notice should be shown exactly once per colony (tracked via a flag in COLONY_STATE.json).

### What Does NOT Change

- Pheromone signals still inject into worker prompts at all depth levels
- Colony-prime context still loads at all depth levels
- Skills injection still works at all depth levels
- Phase research files (from plan) still inject if they exist
- The verification loop (build-verify) still runs Watcher at all depth levels
- The continue flow (Gatekeeper, Auditor) still runs at all depth levels
- The autopilot loop still chains build-continue-advance at all depth levels

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| T1: Named depth levels (4 levels) | HIGH -- clear, consistent with plan.md | LOW -- design doc, no code | P1 |
| T2: Default to "standard" | HIGH -- right balance for most builds | LOW -- conditional spawn logic | P1 |
| T3: Per-build CLI flag | HIGH -- primary use case | LOW -- parse $ARGUMENTS, already done for --verbose etc. | P1 |
| T4: Spawn plan reflects depth | MEDIUM -- visibility into what runs | LOW -- update display text | P1 |
| T5: BUILD SUMMARY includes depth | MEDIUM -- audit trail | LOW -- add to synthesis JSON | P1 |
| T6: Backward compatibility | HIGH -- no surprises | LOW -- default handles it | P1 |
| D1: Colony-level preference | HIGH -- set once, use always | MEDIUM -- COLONY_STATE field + /ant:depth command | P1 |
| D4: Continue flow respects depth | MEDIUM -- consistency across build/continue | MEDIUM -- update continue playbooks | P2 |
| D2: Phase-type auto-suggestion | MEDIUM -- smart defaults | MEDIUM -- keyword matching + display logic | P2 |
| D3: Cost/time estimate | LOW -- nice to have | LOW -- static estimates | P3 |
| D5: Autopilot depth consistency | MEDIUM -- important for autopilot users | LOW -- read preference at start | P2 |

**Priority key:**
- P1: Must have for v2.6 -- the core depth selector
- P2: Should have -- polish and cross-command consistency
- P3: Nice to have -- future enhancement

---

## Implementation Touch Points

Files that need modification for the core depth selector (P1):

| File | Change | Why |
|------|--------|-----|
| `.claude/commands/ant/build.md` | Add `--depth` to argument parsing | CLI flag entry point |
| `.aether/docs/command-playbooks/build-prep.md` | Parse `--depth` from $ARGUMENTS, resolve to level | Set depth variable |
| `.aether/docs/command-playbooks/build-wave.md` Step 5.0.1 | Wrap Oracle spawn in depth check | Skip at minimal/standard |
| `.aether/docs/command-playbooks/build-wave.md` Step 5.0.2 | Wrap Architect spawn in depth check | Skip at minimal/standard |
| `.aether/docs/command-playbooks/build-wave.md` Step 5.1 | Wrap Chaos spawn in depth check | Skip at minimal |
| `.aether/docs/command-playbooks/build-wave.md` Step 5 | Update spawn plan header | Show depth + skipped castes |
| `.aether/docs/command-playbooks/build-complete.md` Step 7 | Add depth to BUILD SUMMARY | Audit trail |
| `.aether/data/COLONY_STATE.json` | Add `build_depth` field | Colony preference persistence |
| `.claude/commands/ant/depth.md` | New command: set/display colony depth | User-facing preference |
| `.claude/commands/ant/run.md` | Pass depth through autopilot | Consistency |
| `.opencode/commands/ant/build.md` | Mirror changes | OpenCode parity |
| `.opencode/commands/ant/depth.md` | Mirror changes | OpenCode parity |

---

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `.aether/docs/command-playbooks/build-wave.md` (746 lines, full build wave orchestration including Oracle Step 5.0.1, Architect Step 5.0.2)
- Direct codebase analysis of `.aether/docs/command-playbooks/build-prep.md` (255 lines, argument parsing precedent with --verbose, --no-visual, --no-suggest, --model)
- Direct codebase analysis of `.aether/docs/command-playbooks/build-verify.md` (399 lines, Watcher/Measurer/Chaos spawning)
- Direct codebase analysis of `.aether/docs/command-playbooks/build-complete.md` (350 lines, synthesis and BUILD SUMMARY)
- Direct codebase analysis of `.aether/docs/command-playbooks/continue-verify.md` (Probe spawning, conditional on coverage)
- Direct codebase analysis of `.aether/docs/command-playbooks/continue-gates.md` (Gatekeeper/Auditor spawning)
- Direct codebase analysis of `.claude/commands/ant/plan.md` (depth levels: --fast, --balanced, --deep, --exhaustive with target_confidence and max_iterations)
- Direct codebase analysis of `.claude/commands/ant/run.md` (autopilot argument parsing)
- Direct codebase analysis of `.aether/data/COLONY_STATE.json` (current schema, no build_depth field)
- Direct codebase analysis of `.aether/workers.md` (caste definitions, model slots, spawn depth vs build depth distinction)

### Secondary (MEDIUM confidence)
- CLI UX best practices for verbosity/depth controls (well-established domain knowledge: -v/-vv/-vvv pattern, named levels pattern, combined approach pattern)
- Plan.md depth system as precedent for naming convention (HIGH confidence -- directly inspected)
- Existing conditional caste spawning patterns (Archaeologist, Ambassador, Measurer) as implementation precedent

### Tertiary (LOW confidence)
- Multi-agent orchestration tool comparisons (CrewAI, AutoGen, LangGraph) -- web search was rate-limited; patterns based on training data
- Whether the "standard" default will satisfy most users -- needs real-world validation
- Whether the one-time migration notice is sufficient or if users will be confused by missing Oracle/Architect

---
*Feature research for: Aether v2.6 Colony Depth Selector*
*Researched: 2026-03-29*
