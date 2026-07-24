# Architecture Research: Queen Execution Policy and Worker Spawning

**Project:** Aether v1.23 Daily Driver Reliability
**Domain:** CLI colony framework -- Queen orchestration, execution policy, and worker spawning patterns
**Researched:** 2026-05-20
**Confidence:** HIGH (primary source: Go source code, command YAML specs, CLAUDE.md, and playbook files)

---

## Executive Summary

The Queen orchestration system has **two distinct execution policy layers** that have become partially decoupled:

1. **The CLAUDE.md policy** (fast / standard / final-review) -- a high-level description of how the Queen should autonomously choose execution depth. This is the design intent documented in CLAUDE.md under "Queen-Owned Orchestration."

2. **The Go runtime implementation** (`cmd/caste_relevance.go`, `cmd/codex_continue.go`) -- uses a **relevance-score + threshold + flow-type** system driven by `VerificationDepth` (light / standard / heavy). This is what actually runs.

The CLAUDE.md "fast / standard / final-review" naming does **not** map 1:1 to any Go code. The Go runtime uses `VerificationDepth` (light / standard / heavy). The CLAUDE.md description is aspirational guidance for wrapper behavior, not an implemented execution mode system. This gap is a core reliability issue: the documented policy and the runtime policy disagree on terminology, scope, and what triggers each mode.

Worker spawning falls into two categories with different waste profiles:

- **Deterministic commands** (status, phase, history, focus, redirect, feedback, pheromones, resume, watch) -- these are Go runtime passthroughs that never spawn agents. Zero waste here.
- **Lifecycle commands** (build, continue, seal, plan, colonize, oracle, swarm) -- these use the caste relevance scoring system. Waste occurs when the scoring system spawns agents for phases that don't need them.

---

## The 4 Execution Modes (CLAUDE.md Design Intent)

From CLAUDE.md "Queen-Owned Orchestration" section:

| Mode | When Used | Verification | Watcher | Specialist Review |
|------|-----------|-------------|---------|-------------------|
| **fast** | Low-risk work | Light | Skipped | None |
| **standard** | Moderate-risk / refactor | Standard | Skipped | Focused Probe only |
| **final-review** | Final, release, security, core runtime | Heavy | Enabled | Watcher + specialist review |

**Key principle:** "The Queen chooses execution and review depth autonomously by default. Users should not need to remember `--skip-watchers`, `--verification-depth`, or timeout flag combinations."

**Status:** This is **design intent only**. The Go runtime does not implement a `fast` / `standard` / `final-review` enum. The wrapper is supposed to translate between these concepts and the Go runtime's `VerificationDepth`, but the mapping is implicit and inconsistent.

---

## The Go Runtime Implementation (What Actually Runs)

### VerificationDepth (the real control knob)

Defined in `pkg/colony/colony.go`:

| Value | Aliases | Effect |
|-------|---------|--------|
| `light` | minimal, coarse | Minimal review; watcher always required but no specialists |
| `standard` | (default) | Standard review; watcher + probe always required |
| `heavy` | full, thorough | Full gauntlet; watcher + gatekeeper + auditor + probe always required |

Normalization function (`NormalizeVerificationDepth`): maps user input and aliases to canonical values. Empty/unknown input defaults to `standard`.

### The Caste Relevance Scoring System

The actual spawning decision lives in `cmd/caste_relevance.go`. It works as follows:

```
queenOrchestrate(phase, flowType, state)
  -> applyQueenSpawnBudget(
       queenCandidateDispatches(phase, flowType, state),
       phase, flowType, state
     )
```

**Step 1: Score each caste** (`casteRelevanceScore`):
- Each caste has a `BaseScore` (15-20) and `Keywords`
- Phase name and task descriptions are matched against keywords
- Matched keywords add to the base score
- Phase mode conditions (e.g., `mode==discovery`, `mode==production`) add +20

**Step 2: Always-required check** (`isAlwaysRequired`):
Certain castes bypass the scoring threshold for certain flow types:

| Flow | Always-Required Castes |
|------|----------------------|
| build | Determined by `queenBuildSafetyRequiredCaste` (builder always for non-discovery) |
| continue (light) | watcher only |
| continue (standard) | watcher + probe |
| continue (heavy) | watcher + gatekeeper + auditor + probe |
| plan | scout + route_setter |
| colonize | surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens |
| swarm | tracker, scout, archaeologist, builder, watcher (+ gatekeeper for high-risk) |
| seal (light) | none |
| seal (standard) | auditor + probe |
| seal (heavy) | gatekeeper + auditor + probe |

**Step 3: Flow threshold check** (`spawnThreshold`):

| Flow | Threshold (light/standard) | Threshold (heavy) |
|------|---------------------------|-------------------|
| build | 30 | 30 |
| continue | 30 | 25 |
| plan | 40 | 40 |
| colonize | 35 | 35 |
| swarm | 35 | 35 |
| seal | 50 | 50 |

A caste with score >= threshold gets dispatched (unless suppressed).

**Step 4: Suppression** (`isCasteSuppressed`):
- Discovery-phase builds suppress: builder, weaver, fixer, porter
- Seal with light depth suppresses: gatekeeper, auditor, probe
- Continue and seal suppress: builder, weaver, tracker, archaeologist, ambassador

**Step 5: Flow allowlist** (`casteAllowedForFlow`):
Each flow type has a hardcoded allowlist of castes that can participate. Castes not on the list are never dispatched regardless of score.

### The Registry

All 16 castes in the registry with their keywords and base scores:

| Caste | Keywords | Base Score | Build | Continue | Plan | Seal |
|-------|----------|-----------|-------|----------|------|------|
| builder | implement, build, create, add, write, fix, code, deploy | 20 | yes | no | no | no |
| watcher | verify, test, validate, check, review, quality | 20 | yes | always | no | no |
| scout | research, investigate, survey, analyze, document, readme, spec, explore | 20 | yes | no | always | no |
| route_setter | plan, route, decompose, structure, organize | 15 | yes | no | always | no |
| architect | design, schema, architecture, interface, boundary, structure, evaluate | 20 | yes | no | yes | no |
| oracle | research, spike, investigate, evaluate, unknown, deep dive, survey (+ discovery mode) | 20 | yes | no | yes | no |
| chaos | resilience, failure, robustness, crash, error handling, stress test | 15 | yes | yes | no | no |
| archaeologist | legacy, migration, modernize, rewrite, history, refactor old | 20 | yes | no | no | no |
| gatekeeper | auth, crypto, security, token, secrets, permissions, compliance, audit | 20 | yes | yes | yes | yes |
| auditor | compliance, audit, production, quality gate, standards (+ production mode) | 20 | yes | yes | no | yes |
| probe | test coverage, edge case, validation, verify, missing tests, coverage gap | 15 | yes | yes | no | yes |
| measurer | performance, optimize, latency, scale, benchmark, memory, cpu | 20 | yes | yes | no | yes |
| ambassador | api, sdk, oauth, external service, integration, webhook, third-party, stripe, etc. | 20 | yes | no | no | no |
| tracker | bug, fix, regression, investigate failure, root cause, issue | 20 | yes | no | no | no |
| weaver | refactor, cleanup, modernize, extract, simplify, restructure | 20 | yes | no | no | no |

---

## The CLAUDE.md-to-Runtime Mapping Gap

The CLAUDE.md describes three execution modes (fast, standard, final-review). The Go runtime uses `VerificationDepth` (light, standard, heavy). The mapping is:

| CLAUDE.md Mode | Likely Go Equivalent | Gap |
|---------------|---------------------|-----|
| fast | `VerificationDepth=light` | CLAUDE.md says "watcher subprocess skipped" but Go code says watcher is **always required** even at light depth |
| standard | `VerificationDepth=standard` | CLAUDE.md says "watcher subprocess skipped" but Go code says watcher is **always required** at standard depth too |
| final-review | `VerificationDepth=heavy` | Closest match. CLAUDE.md says "Watcher and specialist review enabled" -- Go code agrees (watcher + gatekeeper + auditor + probe) |

**Critical finding:** The CLAUDE.md description of fast and standard modes as "watcher subprocess skipped" directly contradicts the Go implementation where watcher is always required for continue flow. This means:

1. The documented intent (skip watcher for fast/standard) was never implemented, OR
2. The documentation was written aspirationally and the Go code took a different path

Either way, the CLAUDE.md "Queen-Owned Orchestration" section is misleading about what actually happens.

---

## How Spawning Works: Deterministic vs Non-Deterministic Commands

### Deterministic Commands (No Agent Spawning)

These commands are Go runtime passthroughs. They read state, run deterministic logic, and return output. Zero agent spawning waste.

| Category | Commands | Mechanism |
|----------|----------|-----------|
| Display | status, phase, history, pheromones, watch | Direct Go CLI call |
| Signal | focus, redirect, feedback | Direct Go CLI call (`aether focus`, `aether redirect`, `aether feedback`) |
| Session | resume, pause-colony, resume-colony | Direct Go CLI call |
| Lifecycle display | maturity, flags, memory-details | Direct Go CLI call |
| Admin | update, publish, version, integrity | Direct Go CLI call |

All of these use `category: "literal"` in `classic-command-parity.json` meaning "keep as direct runtime passthrough."

### Non-Deterministic Commands (Agent Spawning via Caste Relevance)

These use the scoring system above. The question is whether the scoring system produces unnecessary spawns.

| Command | Category | Spawning Mechanism | Waste Risk |
|---------|----------|-------------------|------------|
| init | full-orchestration | Wrapper-driven, Go creates state | Low (one-time setup) |
| discuss | semi-intelligent | Go-owned, wrapper presents | Low (interactive Q&A) |
| colonize | full-orchestration | Manifest-based worker waves | Medium (surveyors always required) |
| plan | full-orchestration | Manifest-based worker waves | Medium (scout + route_setter always) |
| build | full-orchestration | Manifest-based worker waves | **HIGH** (see below) |
| continue | semi-intelligent | Go-owned default; TS-host manifest for heavy | **HIGH** (see below) |
| seal | semi-intelligent | Manifest-based final-review workers | Medium (depth-gated) |
| oracle | full-orchestration | RALF loop with iterations | Low (user-initiated deep research) |
| swarm | full-orchestration | Manifest-based investigation/fix waves | Low (user-initiated bug fix) |

---

## Where Unnecessary Spawning Happens

### Build Flow Spawning Waste

**Problem 1: Oracle and Architect always considered for deep/full depth**

In `build-wave.md` (Step 5.0.1 and 5.0.2), Oracle and Architect are spawned at `colony_depth` "deep" or "full". But the Go caste relevance system (`casteAllowedForFlow`) does not even allow Oracle or Architect in the build flow -- they are only allowed in plan flow.

This means the **playbook instructs the wrapper to spawn Oracle and Architect**, but the **Go runtime caste relevance system would not include them**. There is a conflict between the playbook instructions and the Go scoring system.

**Problem 2: Chaos always mentioned in spawn plan**

`build-wave.md` Step 5 says: "Resilience testing -> Chaos (ALWAYS spawn one after Watcher)." But then Step 5.6 says: "DEPTH CHECK: Skip if colony depth is not 'full'." So Chaos is "always" spawned but only at full depth. The spawn plan announcement (Step 5) always lists Chaos in the verification section regardless of depth, which is misleading.

**Problem 3: Builder-Probe Lock spawns Probe for every code_written task**

Step 5.3.5 spawns a separate Probe agent for every builder that returned `status: "code_written"`. If 4 builders return code_written, that is 4 additional Probe spawns. This is the correct behavior for thorough verification, but it can be wasteful for trivial changes (e.g., a documentation phase where all tasks produce code_written status).

**Problem 4: Measurer spawns on keyword match**

The Measurer spawns whenever the phase name contains performance keywords. This is a simple string match that could false-positive on phases like "Refactor for better performance documentation" -- triggering a full measurement agent for a docs-only phase.

**Problem 5: Ambassador spawns on keyword match**

Similar to Measurer, Ambassador spawns when task descriptions contain any of 18 integration-related keywords (API, SDK, OAuth, aws, etc.). A phase titled "Add API documentation" would trigger Ambassador even though no actual integration work is needed.

### Continue Flow Spawning Waste

**Problem 6: Auditor always spawns for standard depth continue**

Per `isAlwaysRequired` in `cmd/caste_relevance.go`, at standard depth the continue flow requires watcher + probe. But the **playbook** (`continue-gates.md` Step 1.9) marks Auditor as "MANDATORY" -- it always spawns regardless of depth. This contradicts the Go code where Auditor is only always-required at heavy depth.

The playbook says: "Code quality audit -- runs on every /ant-continue for consistent coverage."

The Go code says: at standard depth, only watcher + probe are always required. Auditor needs keyword match + score >= 25.

**Problem 7: Gatekeeper spawns for every package.json project**

`continue-gates.md` Step 1.8 spawns Gatekeeper whenever `package.json` exists. This is a file-existence check, not a relevance check. Every Node.js project gets a full supply chain security audit on every `/ant-continue`, regardless of phase content.

**Problem 8: Multiple sequential spawns that could be parallel**

The continue gates run sequentially: spawn gate -> anti-pattern gate -> complexity gate -> gatekeeper -> auditor -> TDD gate -> runtime gate -> flags gate -> watcher veto -> medic. Many of these are independent checks that run as separate agent spawns (gatekeeper, auditor, complexity/weaver, medic). They could run in parallel but don't.

### Seal Flow Spawning Waste

**Problem 9: Seal spawns heavy review workers by default**

The seal manifest fetch (`aether host seal`) produces a full manifest with final-review dispatches. But the YAML spec says `category: "semi-intelligent"` and the Go `sealReviewDepthForColony` function selects depth based on colony size. Small colonies (1-3 phases) get light depth (no review workers), but medium colonies (4-10 phases) get standard (auditor + probe), and large colonies (10+ phases) get heavy (gatekeeper + auditor + probe).

The waste risk is that medium colonies always get auditor + probe even for simple seal operations.

---

## Minimal-Path Principle

The minimal execution path for each command should be:

### Build: Minimal Path

```
1. Load state (Go: `aether load-state`)
2. Validate phase
3. Update state to EXECUTING (Go: `aether state-mutate`)
4. Git checkpoint
5. Analyze tasks -> group into waves
6. [depth-dependent] Oracle research (deep/full only)
7. [depth-dependent] Architect design (deep/full only)
8. Spawn builders (parallel, per wave)
9. Spawn watcher (always, 1 instance)
10. [depth-dependent] Chaos (full only)
11. Builder-Probe Lock (1 probe per code_written task)
12. Synthesis + finalize
```

**Minimum spawns for a 2-task discovery phase at light depth:** 2 builders + 1 watcher + 1-2 probes = 5-6 agents.

**Actual spawns for the same phase at standard depth with the current playbook:** 2 builders + 1 watcher + 1 chaos + 2 probes + potentially ambassador/measurer = 7-9 agents.

### Continue: Minimal Path

```
1. Load state
2. Run verification loop (build, types, lint, tests) -- NO agents
3. [depth-dependent] Probe coverage agent (production mode only)
4. Check gate results from verification
5. [standard/heavy] Spawn specialist reviewers (1-3 agents)
6. Extract learnings
7. Advance state
```

**Minimum spawns for standard continue:** 1 probe (if production mode) + watcher if not skipped = 0-2 agents.

**Actual spawns with current playbook gates:** auditor + gatekeeper (if package.json) + probe + watcher + potential weaver/medic = 3-6 agents.

---

## Command Classification for Spawning

| Command | Should Spawn Agents? | Current Behavior | Verdict |
|---------|---------------------|-----------------|---------|
| init | No (wrapper-driven) | Correct | OK |
| discuss | No (Go-owned Q&A) | Correct | OK |
| focus/redirect/feedback | No (Go CLI passthrough) | Correct | OK |
| pheromones | No (Go CLI passthrough) | Correct | OK |
| status/phase/history | No (Go CLI passthrough) | Correct | OK |
| resume | No (Go CLI passthrough) | Correct | OK |
| watch | No (Go CLI passthrough) | Correct | OK |
| maturity/flags/memory-details | No (Go CLI passthrough) | Correct | OK |
| version/integrity/update/publish | No (Go CLI passthrough) | Correct | OK |
| colonize | Yes (surveyors) | Appropriate | OK |
| plan | Yes (scout + route_setter) | Appropriate | OK |
| oracle | Yes (RALF loop) | Appropriate | OK |
| swarm | Yes (investigation waves) | Appropriate | OK |
| build | Yes, but reduce | Over-spawns at standard depth | **NEEDS FIX** |
| continue | Yes, but reduce | Over-spawns (playbook vs Go mismatch) | **NEEDS FIX** |
| seal | Yes, depth-gated | Mostly OK, medium colonies over-reviewed | MINOR |

---

## The Queen Decision Layer (Gate Resolution)

The `queenDecide` function in `cmd/queen_decision.go` handles gate failure resolution during continue. This is separate from spawning -- it decides what to do when a gate fails.

### Gate Classification Tiers

| Tier | Response | Auto-Resolve? |
|------|----------|---------------|
| hard_block | Always escalate | Never |
| soft_block | Auto-resolve if budget > 0 | Yes, with budget |
| advisory | Log and continue | N/A |
| unclassified | Treated as advisory | N/A |

### Budget and Circuit Breaker

- **Budget:** Configurable per-phase auto-resolve attempts. Each soft_block auto-resolve consumes one unit.
- **Circuit breaker:** If the same worker triggers 2+ soft_blocks, escalation replaces auto-resolve regardless of budget.
- **State persistence:** Queen decisions stored in `.aether/data/queen-state-{phase}.json`.

### Recommendations

| Recommendation | Meaning |
|---------------|---------|
| pass | All gates passed or advisory only |
| auto-resolve | Soft block with budget remaining |
| dispatch-fixer | Targeted fix possible |
| escalate | Hard block, budget exhausted, or breaker tripped |

This system is sound and well-implemented. The issue is not in the decision logic but in how many gates get created (due to over-spawning agents that each produce gate results).

---

## Recommendations for v1.23

### R1: Align CLAUDE.md with Go Runtime

Either:
- (A) Update CLAUDE.md to reflect the actual Go behavior (watcher always required at all depths), OR
- (B) Implement the CLAUDE.md intent in Go code (actually skip watcher at light/standard continue depth)

Recommendation: (A) is safer for v1.23. The watcher-always-required invariant is well-tested and removing it risks losing independent verification.

### R2: Fix Playbook-Go Mismatches

The playbooks (`build-wave.md`, `continue-gates.md`) have instructions that contradict the Go caste relevance system:
- Playbook says Auditor is mandatory for all continues; Go says only at heavy depth
- Playbook spawns Oracle/Architect at deep/full depth; Go doesn't allow them in build flow
- Playbook lists Chaos in spawn plan regardless of depth; Go only dispatches at full depth

Fix: Make playbooks match Go behavior, or make Go match playbook intent. Since Go is authoritative, update playbooks.

### R3: Reduce Default Continue Spawning

At standard depth, the continue flow should spawn at most: watcher + probe (2 agents). The current playbook gates spawn auditor + gatekeeper + probe + watcher (4 agents) plus potential weaver and medic.

Fix: Gate specialist spawns (auditor, gatekeeper, weaver, medic) behind heavy depth or explicit user request.

### R4: Gate Ambassador and Measurer on Phase Mode

Ambassador should only spawn for phases with `mode == "production"` (not keyword match on phase name). Measurer should only spawn for phases where the phase mode is production AND the phase name contains performance keywords.

Fix: Add mode checks to Ambassador and Measurer spawning conditions.

### R5: Collapse Sequential Continue Gates

The continue gates run 8+ sequential gate checks, several of which spawn agents. Gatekeeper, Auditor, and Probe could run in parallel since they are independent checks.

Fix: Parallelize independent gate spawns where possible (post-verification, pre-advance).

---

## Sources

- HIGH: `cmd/caste_relevance.go` -- caste relevance registry, scoring, threshold, always-required, suppression (primary Go source)
- HIGH: `cmd/codex_continue.go` -- continue dispatches, review specs, always-required for continue flow
- HIGH: `cmd/codex_continue_plan.go` -- continue plan-only manifest generation, queenDecide integration
- HIGH: `cmd/queen_decision.go` -- gate classification, recommendations, circuit breaker, budget
- HIGH: `pkg/colony/colony.go` -- VerificationDepth type, normalization, aliases
- HIGH: CLAUDE.md "Queen-Owned Orchestration" section -- design intent for fast/standard/final-review
- HIGH: `.aether/commands/build.yaml` -- build command category and ownership split
- HIGH: `.aether/commands/continue.yaml` -- continue command category, verification depth, heavy review path
- HIGH: `.aether/commands/seal.yaml` -- seal command category and manifest flow
- HIGH: `.aether/references/contracts/queen-execution-policy-contract.md` -- classification tiers, auto-resolve, circuit breaker
- HIGH: `.aether/references/examples/queen-decision-example.md` -- concrete queen-state example
- HIGH: `.aether/commands/classic-command-parity.json` -- command categories (literal/semi-intelligent/full-orchestration)
- MEDIUM: `.aether/docs/command-playbooks/build-wave.md` -- build spawning instructions (playbook layer)
- MEDIUM: `.aether/docs/command-playbooks/continue-gates.md` -- continue gate spawning (playbook layer)
- MEDIUM: `.aether/docs/command-playbooks/build-verify.md` -- build verification spawning (playbook layer)
- MEDIUM: `.aether/docs/command-playbooks/continue-verify.md` -- continue verification loop (playbook layer)

---

*Architecture research for: Queen execution policy and worker spawning patterns*
*Researched: 2026-05-20*
