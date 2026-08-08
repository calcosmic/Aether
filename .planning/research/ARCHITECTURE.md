# Architecture Research — v1.26 Intelligent Orchestration

**Domain:** Multi-agent orchestration runtime (Go kernel + platform wrapper control plane)
**Researched:** 2026-08-08
**Confidence:** HIGH for internal integration points (read from source), MEDIUM for platform nested-spawn behaviour (verified against vendor docs + open issues, changes fast)

**Scope:** How four new capabilities integrate with the existing architecture. Existing structure is treated as settled and is not redesigned.

---

## Executive Summary

All four capabilities have a **pre-existing seam** in the codebase. None require a new architectural layer. Three of the four have partial machinery already committed that has never been switched on — the same pattern the project's own Definition of Done was written against.

| Capability | Pre-existing seam | Status of that seam |
|------------|-------------------|---------------------|
| Recursive delegation | `cmd/spawn.go:169` `spawn-can-spawn`, `pkg/agent/spawn_tree.go` (parent+depth already modelled), `.aether/ts-host/src/spawn-orchestrator.ts` (full policy engine, depth 2, budget) | `spawn-can-spawn` is a **stub that always returns `can_spawn: true`**. The TS orchestrator is on a path the build wrapper is explicitly forbidden to use. |
| Data-driven roster | `cmd/policy_loader.go`, `cmd/visuals_config.go`, `cmd/prompt_template_loader.go` — three working "YAML file overlays hardcoded default" loaders | Pattern proven. `colony/agents/*.yaml` and `colony/policies/model-routing.yaml` have **zero Go readers**. |
| Survey digest | `cmd/codegraph_context.go` — task-relevant slice of a large artifact, own char budget, injected into the brief | Pattern proven and live. Survey uses `resolveSurveySection()` (`cmd/helpers.go:225`) which emits **filenames only**. |
| Spend accounting | `pkg/codex/usage.go` `ParseUsage`, `pkg/codex/platform_dispatch.go:211` `AttachWorkerUsage`, `pkg/trace` `LogTokenUsage`, `trace-summary` | Usage is parsed, attached to `WorkerResult.Usage` — and **read by nothing**. `LogTokenUsage` has one caller, `pkg/agent/pool.go:193`, which is a different execution path. |

**The single hardest constraint** is not any of the four features. It is that the plan-only manifest is cryptographically bound to its build attempt (`cmd/build_attempt.go:193` `prepareBuildAttemptManifestBinding` → `codex.ExecutionBinding.ManifestSHA256`). Recursive delegation adds work *after* that hash is computed. That collision must be designed for explicitly, not discovered during implementation.

**The second hardest constraint** is the prompt budget arithmetic. The per-section budgets already sum to ~17,700 chars before the task brief, against a 24,000 global cap (`pkg/codex/prompt.go:13`). A survey digest is additive scaffolding, and `TestBuildWorkerBriefIsMostlyTask` names survey pointers explicitly as scaffolding. Adding a digest without displacing something else will fail that test — by design.

---

## Standard Architecture (current, as read from source)

### System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│  PLATFORM WRAPPER  (.claude/commands/ant/build.md, .opencode/…)      │
│  Owns: spawning, narration, pacing.  Mutates: nothing.               │
│  ┌────────────┐   ┌───────────┐   ┌────────────┐   ┌──────────────┐ │
│  │ fetch      │→  │ Queen     │→  │ spawn per  │→  │ collect      │ │
│  │ manifest   │   │ --castes  │   │ wave       │   │ completion   │ │
│  └────────────┘   └───────────┘   └────────────┘   └──────────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│  GO RUNTIME  (cmd/, pkg/)  — sole authority for state               │
│                                                                      │
│  aether build N --plan-only                                          │
│   ├─ queenOrchestrate ────────── cmd/caste_relevance.go:146          │
│   ├─ queenApplyJudgement ─────── cmd/queen_judgement.go:83           │
│   ├─ queenSpawnBudgetForPhase ── cmd/queen_spawn_budget.go:27        │
│   ├─ renderCodexBuildWorkerBrief cmd/codex_build.go:2487             │
│   │    ├─ resolveSurveySection    cmd/helpers.go:225   (filenames)   │
│   │    ├─ renderCodegraphContext  cmd/codegraph_context.go  (2200)   │
│   │    ├─ resolvePheromoneSection cmd/codex_build.go:2963            │
│   │    └─ renderWorkerHandoffSection cmd/codex_dispatch_contract.go  │
│   ├─ writeCodexBuildArtifacts ── cmd/codex_build.go:1896             │
│   │    └─ writes .aether/data/build/phase-N/worker-briefs/*.md       │
│   └─ prepareBuildAttemptManifestBinding cmd/build_attempt.go:193     │
│        └─ SHA-256 over the whole manifest → ExecutionBinding         │
│                                                                      │
│  aether build-completion-stage N   cmd/codex_build_finalize.go:158   │
│   └─ validateCompletionPacketStructure  cmd/contract_schema.go:127   │
│        (schema REFLECTED from Go structs, byte-compared to           │
│         .aether/schemas/completion-packet.schema.json)               │
│                                                                      │
│  aether build-finalize N          cmd/codex_build_finalize.go:122    │
│   ├─ validateCompletionPacketSemantics  :918                         │
│   └─ persistExternalBuildHandoffs       :1122                        │
├──────────────────────────────────────────────────────────────────────┤
│  STORES (.aether/data/, gitignored)                                  │
│  COLONY_STATE.json │ build/phase-N/{manifest,attempt,briefs}         │
│  handoffs/worker-handoffs.json │ spawn-tree.txt │ trace.jsonl        │
│  survey/*.md (~123KB, never read by a worker)                        │
├──────────────────────────────────────────────────────────────────────┤
│  COLONY ASSETS (colony/, distributed)                                │
│  agents/*.yaml (27) ── NO READER   │ policies/*.yaml (10, 4 read)    │
│  prompts/*.md (read) │ ceremony/visuals.md (read)                    │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities (unchanged by this milestone)

| Component | Responsibility | Where |
|-----------|----------------|-------|
| Manifest generator | Decides the team, composes every brief, binds the attempt | `cmd/codex_build.go` |
| Judgement layer | Reconciles a model-proposed team with floors and ceiling | `cmd/queen_judgement.go` |
| Wrapper | Spawns exactly what the manifest names; never invents | `.claude/commands/ant/build.md` |
| Dispatch executor | Runs waves for the *hosted* (non-wrapper) path only | `pkg/codex/dispatch.go` |
| Finalizer | Validates, persists, advances | `cmd/codex_build_finalize.go` |
| Schema | Reflected from structs, guards the wrapper→runtime boundary | `cmd/contract_schema.go` |

---

## Capability 1: Recursive Delegation

### The collision, stated precisely

The current model is **plan-then-execute with a frozen plan**. Three things freeze at manifest time:

1. `dispatch_manifest.dispatches` — the full worker list
2. Each `dispatch.brief` / `brief_path` — every prompt, byte-identical on disk and inline (`cmd/codex_build.go:1902-1923`)
3. `ExecutionBinding.ManifestSHA256` — a SHA-256 over the whole manifest, validated on the way back in (`cmd/build_attempt.go:220` `bindBuildAttemptManifest`, checked at `:280`)

A nested spawn is, definitionally, a worker that was not in the manifest and whose brief did not exist when the hash was taken. Any design that lets a child result reach `build-finalize` must answer the binding question. There are exactly three coherent answers:

| Option | Mechanism | Verdict |
|--------|-----------|---------|
| **A. Rebind** | Amend the manifest with child dispatches, recompute the digest, rebind the attempt mid-run | **Reject.** The digest exists so a result from a stale checkout or a superseded attempt cannot be accepted. A mutable digest is not a digest. |
| **B. Child is invisible to the manifest** | Child result is folded into the *parent's* result before submission; the packet still contains exactly the manifest's workers | **Recommended.** The binding never moves. The parent's `handoff` and `child_results` carry the evidence. |
| **C. Follow-on wave** | Runtime adjudicates a delegation request and emits a *new bound attempt* for a follow-on wave; the wrapper spawns it | **Recommended as the second channel.** Needed when the parent cannot spawn (see platform constraints). |

**Recommendation: B as the default, C as the guaranteed fallback.** They share one adjudication point and one storage shape, so this is one feature with two delivery channels, not two features.

### Who spawns the child?

**The parent worker spawns it when the platform allows; the top-level wrapper spawns it otherwise. The Go runtime never spawns on the wrapper path.**

This preserves the boundary exactly. A parent worker is itself a wrapper-spawned platform agent; when it calls the platform Agent tool it is the *platform* doing the spawning, not the Go runtime. Go's role is unchanged: it adjudicates and it persists.

The reason a second channel is needed is platform reality, not architecture:

| Platform | Nested spawn support | Confidence | Source |
|----------|---------------------|------------|--------|
| Claude Code | Supported. Re-enabled v2.1.219 (2026-07-24), default depth 3, cap 5, `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`. Concurrent cap 20, session cap 200. **But** open bug: Agent/Task tool stripped for some subagent types, making them leaf agents (anthropics/claude-code#80036, #4182). | MEDIUM | vendor docs + open issues |
| OpenCode | Supported, but blocked by default since v1.18.2 with a depth config. Frontmatter `task:` override has been broken by a permission rework (anomalyco/opencode#8114, #14308). | MEDIUM | open issues |
| Aether's own agents | 25 of 28 OpenCode agents ship `task: false`; only 2 Claude agents (`aether-queen`, `aether-route-setter`) declare `Task`. `aether-route-setter.md:153` still documents "Claude Code subagents cannot reliably spawn further subagents" — **now stale**. | HIGH | repo |

So: build the request/grant protocol first, make the follow-on wave the channel that always works, and treat parent-direct spawn as an opportunistic optimization gated on a runtime-detected capability. Do not make the feature's correctness depend on a platform behaviour that two vendors have each broken in the last quarter.

### Data flow (Option B, the default path)

```
manifest wave 1
   └─ wrapper spawns Builder Mason-67 (depth 1)
        │  worker decides it needs a Scout
        │
        ├─ aether spawn-request --parent Mason-67 --depth 1 \
        │      --caste scout --task "..." --reason "..."      ← NEW command
        │      ↓ Go adjudicates against depth cap + remaining budget
        │      ↓ returns {granted, child_name, brief, brief_path, agent_name}
        │      ↓ writes spawn-tree entry (parent, depth 2)  ← EXISTING store
        │
        ├─ parent spawns child via platform Agent tool with the runtime brief
        │      (verbatim, same rule as the top-level wrapper)
        │
        └─ parent returns ONE result containing:
              handoff:        { …, }                      ← unchanged shape
              child_results:  [ { name, caste, task, status,
                                  summary, files_created,
                                  files_modified, handoff } ]  ← NEW field
                    ↓
   build-completion-stage → schema validation (regenerated, see below)
                    ↓
   build-finalize
     ├─ claims: child files_created/modified merged into the PARENT's task claim
     ├─ handoffs: persistExternalBuildHandoffs also walks child_results
     └─ spend: child usage rows attributed to the parent's dispatch, own trace row
```

### What breaks, and where

**Wave model.** Nothing breaks on Option B — the child is inside the parent's wall-clock, so wave semantics are untouched. The real cost is **latency compounding**: `pkg/codex/dispatch.go:132` `DispatchWaveWithObserver` runs a wave to completion before the next; a parent that blocks on a child stretches its whole wave. The per-worker `Timeout` (`WorkerDispatch.Timeout`) is now a *subtree* timeout, not a worker timeout, and nothing in the current code says so. Decide and document whether the child's time comes out of the parent's budget (recommended) or gets its own.

On Option C, wave numbering does break: `codexBuildExecutionPlan.ExecutionWave` and `codexWaveExecutionPlan.Wave` (`cmd/codex_build.go:131-143`) are computed once at manifest time. A follow-on wave needs a wave index that provably does not collide — use a separate attempt with its own wave 1 rather than appending to the bound manifest's numbering.

**Budget model.** This is where the current code is actively wrong for the new feature:

- `queenSpawnBudget.MaxWorkers` (`cmd/queen_spawn_budget.go:11`) is a **caste-selection ceiling**, not a live worker counter. It is consumed once, at manifest time, by `applyQueenSpawnBudget` and `queenApplyJudgement`. There is no runtime object that knows how many workers have actually run.
- `spawn-can-spawn` (`cmd/spawn.go:169`) takes `--depth` and unconditionally returns `can_spawn: true`. It does not read the tree, the budget, or anything else. This is the exact "documented but never implemented" shape the project's Definition of Done exists to catch — fixing it is a prerequisite, not a nice-to-have.
- The TS host already solved this once: `spawn-orchestrator.ts` carries `totalBudget`/`consumedBudget`/`currentDepth` with `MAX_SPAWN_DEPTH = 2` and fail-closed rejection. That policy is correct and should be **ported to Go**, not re-invented, and not called through the TS host (which `build.md` forbids on this path).
- Required castes must not be spendable. `queenBuildSafetyRequiredCastes` bypasses the budget cap at selection time; delegation must draw from the *optional* remainder only, or a chatty Builder can spend the Watcher's slot.

**Handoff relay.** Two concrete breakages:

- `renderWorkerHandoffSection` (`cmd/codex_dispatch_contract.go:704`) takes the top 5 records sorted by freshness, after `pruneWorkerHandoffRecords(…, 100)`. Children are the *most recent* records by construction, so a build with several delegations will flood the relay and evict the parent handoffs the next phase actually needs. Children must either be excluded from the top-level relay or ranked below same-depth peers.
- `handoffProvenance` already renders caste/wave/age. It does not render **depth or parent**. A reader given a child's handoff with no parent cannot tell whether it describes the phase or a sub-task of one dispatch. Add depth/parent to `workerHandoffRecord` and render it.
- `IsEmptyWorkerHandoff` (`pkg/codex/handoff.go:26`) rejects content-free handoffs, and `build.md` states the finalizer rejects empty handoffs. A trivially-scoped child ("read this one file") will legitimately produce a near-empty handoff. Either exempt children from the emptiness rule or require the parent to absorb the child's findings into its own handoff. **Recommend the latter** — it keeps one rule and forces the parent to actually read what it delegated.

**Schema.** `cmd/contract_schema.go:61` reflects the schema from `codexExternalBuildCompletion` and byte-compares to `.aether/schemas/completion-packet.schema.json`. Adding `ChildResults` to `codexExternalBuildWorkerResult` (`cmd/codex_build_finalize.go:62-81`) regenerates the schema automatically and *will* fail `cmd/contract_schema_test.go` until the committed file is refreshed. This is the system working. Budget one plan step for it.

**Recursion self-reference.** `codexExternalBuildWorkerResult` containing `[]codexExternalBuildWorkerResult` is a recursive Go type. `invopop/jsonschema` handles this with `$ref`/`$defs`, but the byte-determinism requirement in `generateCompletionPacketSchemaBytes` makes this worth proving on a spike before committing to the shape. **Recommendation: define a distinct, flatter `codexChildWorkerResult` struct** with only the fields a child can meaningfully return. It avoids the recursion, and it makes "a child cannot itself declare children in the packet" a type-level fact rather than a validation rule.

### Depth policy

Adopt the TS host's `MAX_SPAWN_DEPTH = 2` (manifest worker = depth 1, child = depth 2, no grandchildren). Reasons: it matches `spawn-tree.txt`'s existing depth field and `spawn-log --depth`; it is below both platforms' current caps so it never depends on vendor defaults; and unbounded recursion is a live, filed defect on one of the two platforms (anomalyco/opencode#18100). Make the cap a named constant with a test, not an env var.

---

## Capability 2: Data-Driven Agent Roster

### Current fragmentation (measured, not estimated)

| Table | Count | Location | Purpose |
|-------|-------|----------|---------|
| `casteRelevanceRegistry` | **26** | `cmd/caste_relevance.go:29` | Keyword scoring, base scores, conditions |
| `casteCapabilities` | **26** | `cmd/queen_judgement.go:267` | Queen-readable `produces` / `avoid_for` |
| `casteEmojiMap` / `casteColorMap` / `casteLabelMap` | **35 each** | `cmd/codex_visuals.go:38,81,120` | Visual identity — a *superset* incl. queen, colonizer, guardian, dreamer, 8 curation ants |
| `colony/agents/*.yaml` | **27** | `colony/agents/` | id, role, prompt_file, allowed_tools, tier, model, color, description — **no Go reader** |
| Claude agents | 28 files | `.claude/agents/ant/*.md` | Frontmatter tools/model/color + body |
| OpenCode agents | 28 files | `.opencode/agents/*.md` | Same, different frontmatter dialect |
| Codex agents | 27 files | `.codex/agents/*.toml` | `developer_instructions` read by `pkg/codex/prompt.go:30` |

Three name collisions already exist and will bite a naive migration:

- `route_setter` (registry) vs `route-setter.yaml` (colony/agents) vs `route_setter` (emoji map). `resolveCasteName` (`cmd/queen_judgement.go:215`) already papers over separator style at the *proposal* boundary — the loader needs the same normalization at the *definition* boundary.
- `queen` exists in `colony/agents/` and in the visuals maps but is **not** in `casteRelevanceRegistry` (it is not dispatchable). A loader that treats `colony/agents/` as the roster will make the Queen dispatchable.
- `sage` is in the registry and capabilities but **not** in `casteEmojiMap`; the four `surveyor-*` castes collapse to a single `surveyor` key via `normalizeCasteKey` (`cmd/codex_visuals.go:3794`). The visuals maps are therefore not a caste roster and must not be unified with one.

### Recommended migration path

**Do not replace the Go slice. Overlay it.** This is the pattern the codebase already runs three times:

- `cmd/policy_loader.go:25` `loadYAMLPolicy` — read file, fall back to hardcoded default on any error
- `cmd/visuals_config.go:35` `loadVisualsConfig` — `colony/ceremony/visuals.md` frontmatter overrides `casteEmojiMap`, absent file is a no-op
- `cmd/prompt_template_loader.go:66` — `colony/prompts/colony-prime.md` with a compiled-in default

Concretely:

```
NEW  cmd/agent_roster.go
       type agentRosterEntry struct {
           ID, Role, Tier, Model, Color, Description string
           PromptFile   string
           AllowedTools []string
           // scoring fields, additive to the YAML that exists today:
           Keywords   []string
           Conditions []string
           BaseScore  int
           Produces   string
           AvoidFor   string
           Dispatchable bool   // queen.yaml sets false
       }
       func loadAgentRoster() map[string]agentRosterEntry   // cached, sync.Once
       func agentRosterOrDefault(caste string) agentRosterEntry
```

Resolution order, copying `cmd/oracle_loop.go:1585`'s pattern (which exists precisely because a bare CWD-relative path broke for every repo that was not the Aether checkout):

1. `<cwd>/colony/agents/*.yaml` — source checkout
2. `<root>/colony/agents/*.yaml` — repo root when cwd differs
3. `~/.aether/system/colony/agents/*.yaml` — hub (published by `cmd/install_cmd.go:794`'s pattern; **the agents dir is not currently in `installSyncPairs()` and must be added**, or the roster works only in this repo)
4. Compiled-in `casteRelevanceRegistry` + `casteCapabilities` — always present, always correct

### What the loader must NOT own

| Concern | Stays in Go | Why |
|---------|-------------|-----|
| `queenBuildSafetyRequiredCastes` (`cmd/queen_spawn_budget.go:139`) | Yes | A safety floor readable from a user-editable file is not a floor. The judgement layer's whole premise (`cmd/queen_judgement.go:11-27`) is that the model may propose but not lower the floor; a YAML file the model can write is a strictly worse version of the same hole. |
| `isAlwaysRequired`, `casteAllowedForFlow`, `isCasteSuppressed` (`cmd/caste_relevance.go:308,368,387`) | Yes | Flow policy, not caste identity. |
| `spawnThreshold` (`cmd/caste_relevance.go:263`) | Yes | Carries 30 lines of load-bearing measurement in its comment explaining why the number must not be retuned. Moving it to YAML deletes that argument. |
| Visual identity maps | Yes, already overlaid by `visuals.md` | Superset with different semantics; unifying them is a separate, optional job. |

### Validation

**Validation lives in the loader and in a command that fails.** Per the Definition of Done, a roster that silently degrades is worse than one that does not exist.

```
NEW  aether roster-validate          # exit non-zero on any violation
       - every dispatchable entry has a capability (produces + avoid_for)
       - every entry's prompt_file resolves
       - every entry has a platform agent file on all three surfaces
       - no name is reachable under two separator spellings
       - required-caste names referenced by Go all exist in the roster
```

The existing tests become the regression net rather than an obstacle:

| Test | Effect of migration | Action |
|------|--------------------|--------|
| `TestEveryCasteHasACapability` (`cmd/queen_judgement_test.go:190`) | Compares roster length to registry length | Repoint at the loaded roster; it becomes the file-vs-code drift detector |
| `TestRegressionSnapshot` documented_castes (`cmd/regression_test.go:76`) | Counts `casteEmojiMap` (35), not the registry | **Unaffected** if visuals stay out of scope — and they should |
| `TestOpenCodeAgentSchema` (28 files), `TestClaudeOpenCodeAgentContentParity` | Pin platform file counts | Unaffected; add roster→platform-file cross-check to `roster-validate` |

**Do not delete `casteRelevanceRegistry` in this milestone.** Ship the loader, prove the file and the slice agree via a test, and only then consider which is authoritative. The project has 18 milestones of evidence that "the old thing was removed and the new thing was never wired" is its dominant failure mode.

---

## Capability 3: Survey Digest Reaching Workers

### Current behaviour

`resolveSurveySection()` (`cmd/helpers.go:225`) reads `.aether/data/survey/`, lists `.md`/`.json` filenames as repo-relative paths, prepends a staleness notice, and returns. That is the entire mechanism. On this repo it points at ~123KB across BLUEPRINT.md (17K), CHAMBERS.md (20K), DISCIPLINES.md (20K), PATHOGENS.md (25K), SENTINEL-PROTOCOLS.md (19K), TRAILS.md (12K), PROVISIONS.md (9K). A worker must choose to open them, and a cheap model on a narrow task usually will not.

A structured extract already exists — `loadCodexSurveyContext` (`cmd/codex_plan.go:1645`) parses the `*.json` sidecars into `codexSurveyContext` (languages, frameworks, directories, entry points, dependencies, test files, issues, source anchors) — but it is used by **planning**, not by build briefs. `renderPhaseResearchSurveySection` (`cmd/phase_research.go:163`) renders it for research Scouts only.

### Recommended design

Copy `cmd/codegraph_context.go` exactly. It is the proven pattern for "large artifact → task-relevant slice → own char budget → brief":

```
NEW  cmd/survey_digest.go
       const surveyDigestBudgetChars = 2500
       func renderSurveyDigestForText(root string, textParts []string, maxChars int) string
            // 1. loadCodexSurveyContext(root)          — already exists, reuse
            // 2. score survey sections against textParts
            //      (phase name + description + task goal + declared_paths)
            // 3. render top-N as prose facts, not pointers
            // 4. truncate to maxChars
```

**Generated where:** inside `renderCodexBuildWorkerBrief` (`cmd/codex_build.go:2487`), at the existing `resolveSurveySection()` call site (`cmd/codex_build.go:2607`). This is per-dispatch by necessity — the digest is task-relevant, so it cannot be hoisted to the manifest-level `ContextCapsule` the way colony-prime was.

**Stored where:** it does not need to be stored. It is a pure function of `survey/*.json` + the task text, and it is already persisted as part of the brief file at `.aether/data/build/phase-N/worker-briefs/<name>.md` (`cmd/codex_build.go:1902`). Adding a cache is premature; `loadCodexSurveyContext` is a handful of small JSON reads plus `surveyWorkspace(root)`.

**On "an inspection command must never mutate state":** this is satisfied by construction, because nothing is written. If a digest cache is later added for performance, the rule bites in one specific place — a `aether survey-digest --task "..."` inspection command must print and exit, and the cache write must happen only on the `build --plan-only` path, which already writes brief artifacts and is not an inspection command. Lock it with a `TestSurveyDigestInspectionDoesNotMutate` in the shape of `TestConsolidationPhaseEndDryRunDoesNotMutate`.

### The budget conflict — this is the real design decision

Current per-section budgets:

| Section | Budget | Constant |
|---------|--------|----------|
| Colony-prime capsule (compact) | 4,000 | `cmd/colony_prime_context.go:22` |
| Skills | 8,000 | (skill injection, independent budget) |
| Phase research | 3,500 | `cmd/phase_research.go:202` |
| Codegraph | 2,200 | `cmd/codegraph_context.go:12` |
| **Subtotal before task brief** | **17,700** | |
| Global assembled cap | **24,000** | `pkg/codex/prompt.go:13` |

A 2,500-char digest leaves ~3,800 chars for agent instructions plus the entire task brief. Two named tests will catch this, and both should be respected rather than retuned:

- `TestBuildWorkerBriefIsMostlyTask` (`cmd/codex_build_test.go:3629`) — 40% floor, and its comment explicitly classifies **survey pointers as scaffolding**: *"Colony state, skills, pheromones and survey pointers do not meet it and stay on the scaffolding side."* A digest is more scaffolding than a pointer list, not less.
- `assembledContextTaskShareFloorPercent = 5.0` (`cmd/context_budget_test.go:24`), whose comment says outright: *"If real usage shows this threshold is wrong in either direction, that is a finding to raise, not a number to silently retune."*

**Recommendation: make the digest displace the pointer list, not join it.** Keep the total survey allocation at roughly its current size by replacing "here are 7 filenames" with "here are the 8 facts from those 7 files that concern your task, plus the 2 filenames worth opening." That converts dead pointers into live grounding at near-zero net cost, and it is the only version of this feature that does not require raising a cap that two invariant tests were written to defend.

If measurement later shows the digest earns more room, raise `defaultPromptBudgetChars` deliberately, with the measurement recorded — the same way `spawnThreshold`'s comment records why 30 stayed 30.

---

## Capability 4: Spend Accounting

### Current state

```
provider stdout
   → ParseUsage                pkg/codex/usage.go:79      (both provider shapes)
   → AttachWorkerUsage         pkg/codex/platform_dispatch.go:211
   → WorkerResult.Usage        pkg/codex/worker.go:84
   → ✗ nothing reads it
```

Meanwhile `trace.LogTokenUsage` (`pkg/trace/trace.go:132`) has exactly one caller — `pkg/agent/pool.go:193`, the in-process LLM agent pool, which is not the worker-dispatch path. `trace-summary --run-id` (`cmd/trace_cmds.go:192`) aggregates `trace.jsonl` by `run_id`. So the producer and the consumer both exist and have never been connected.

Worse for the wrapper path: `codexExternalBuildWorkerResult` (`cmd/codex_build_finalize.go:62-81`) has **no usage field at all**. On the interactive build path — the one users actually run — the token counts never leave the platform.

### Recommended design

**Persist to `trace.jsonl` keyed on the build attempt's `run_id`. Do not invent a new store.**

The join already exists: `buildAttemptRecord.RunID` (`cmd/build_attempt.go:47`) is produced by `codex.NewExecutionRunID()` and is the same value carried in `ExecutionBinding.RunID`. Every worker on a build already belongs to exactly one run id, and `trace-summary --run-id` already aggregates by it. That makes the whole feature a wiring job plus one struct field.

```
MODIFIED cmd/codex_build_finalize.go
   codexExternalBuildWorkerResult
     + Usage *codex.WorkerUsage `json:"usage,omitempty"`      ← NEW field
       (schema regenerates; refresh .aether/schemas/completion-packet.schema.json)

NEW  cmd/build_spend.go
       func recordBuildSpend(runID string, results []codexExternalBuildWorkerResult) error
         per worker → tracer.LogTokenUsage(runID, model, in, out, usd, "build-worker")
         payload MUST additionally carry: caste, worker_name, task_id, depth,
                                          parent, source ("provider"|"estimate")

MODIFIED cmd/codex_build.go:1610 (hosted path)
       already has result.Usage in hand — log it there too, same function

MODIFIED pkg/trace/trace.go
       LogTokenUsage signature is fixed at (runID, model, in, out, usd, source).
       It cannot carry caste/worker/depth. Either widen it or add
       LogWorkerSpend(runID string, entry WorkerSpendEntry).
       Recommend the latter — widening breaks pkg/agent/pool.go's only caller.

MODIFIED cmd/trace_cmds.go  summarizeTraceEntries
       + per-caste and per-worker rollup, + measured-vs-estimated split
```

### Three rules the data model already demands

1. **Never blend measured and estimated.** `WorkerUsage.Source` and `Measured()` (`pkg/codex/usage.go:38`) exist precisely so an estimate cannot be presented as a measurement. `trace-summary` must report the two totals separately, or the first cheap-model efficiency claim made from this data will be unfalsifiable — which is the exact failure `usage.go`'s own header comment was written to end.
2. **Cost must not be double-computed.** Claude reports `total_cost_usd` directly (`usageFromEvent`, `pkg/codex/usage.go:123`); `trace.CalculateCost` (`pkg/trace/cost.go:29`) computes from a hardcoded rate table that contains **no current model names** — `claude-sonnet-4`, `gpt-4`, and nothing newer, returning 0 for anything unlisted. Prefer the provider's number; fall back to the table; label which was used. Add a test that fails when a model seen in the wild is absent from the table, rather than silently costing it at zero.
3. **Child spend rolls up to the parent.** With recursive delegation, a child's tokens are part of the parent's task cost. Log the child as its own row with `parent` and `depth` set, and have `trace-summary` present both a flat total and a per-dispatch subtree total.

### Deliberately out of scope

Per-worker budget *enforcement* (halting a run on spend). Measure first. The project has no measured baseline for what a phase costs, and a cap set from a guess will either never fire or fire on correct behaviour. Ship the ledger, run three real phases, then decide.

---

## NEW vs MODIFIED — complete component list

### NEW

| Component | Path | Purpose |
|-----------|------|---------|
| Spawn adjudicator | `cmd/spawn_request.go` | `aether spawn-request` — depth + budget + caste validation, returns a runtime-composed child brief or a refusal |
| Spawn budget ledger | `pkg/codex/spawn_budget.go` | Live worker counter for a run; Go port of `spawn-orchestrator.ts` policy |
| Child result type | `cmd/codex_build_finalize.go` | `codexChildWorkerResult` — flat, non-recursive |
| Agent roster loader | `cmd/agent_roster.go` | `colony/agents/*.yaml` reader with 4-step resolution + compiled-in fallback |
| Roster validator | `cmd/roster_validate.go` | `aether roster-validate`, exits non-zero |
| Survey digest | `cmd/survey_digest.go` | Task-relevant slice of `codexSurveyContext`, own char budget |
| Spend recorder | `cmd/build_spend.go` | Worker usage → `trace.jsonl` keyed on attempt run id |
| Worker spend trace entry | `pkg/trace/spend.go` | `LogWorkerSpend` carrying caste/worker/depth/parent/source |

### MODIFIED

| Component | Path | Change |
|-----------|------|--------|
| `spawn-can-spawn` | `cmd/spawn.go:169` | **Stop returning `can_spawn: true` unconditionally.** Consult depth cap, spawn tree, and live budget |
| Spawn tree render | `pkg/agent/spawn_tree.go` | Already models parent/depth; surface depth in `spawn-tree-active` output and ceremony |
| External worker result | `cmd/codex_build_finalize.go:62` | `+ ChildResults`, `+ Usage` |
| Committed schema | `.aether/schemas/completion-packet.schema.json` | Regenerate (byte-compared by `cmd/contract_schema_test.go`) |
| Handoff persistence | `cmd/codex_build_finalize.go:1122` `persistExternalBuildHandoffs` | Walk `child_results`; stamp depth + parent |
| Handoff record | `cmd/codex_dispatch_contract.go` `workerHandoffRecord` | `+ Depth`, `+ Parent`; render in `handoffProvenance` |
| Handoff relay ranking | `cmd/codex_dispatch_contract.go:704` `renderWorkerHandoffSection` | Rank children below same-depth peers so they cannot flood the top-5 |
| Claims aggregation | `cmd/codex_build_finalize.go` | Merge child file claims into the parent's `TaskClaimsSummary` |
| Caste registry | `cmd/caste_relevance.go:29` | Reads through `agentRosterOrDefault`; slice stays as fallback |
| Caste capabilities | `cmd/queen_judgement.go:267` | Same; `queenCasteRoster()` sources from the loader |
| Hub sync pairs | `cmd/install_cmd.go:794` | Add `colony/agents` (and `colony/prompts` if not present) so the roster ships |
| Survey section | `cmd/helpers.go:225` `resolveSurveySection` | Digest displaces the pointer list; keep the staleness notice |
| Build brief | `cmd/codex_build.go:2487`, `:2607` | Call the digest renderer |
| Trace summary | `cmd/trace_cmds.go:192` | Per-caste/per-worker rollup; measured-vs-estimated split |
| Cost table | `pkg/trace/cost.go` | Prefer provider-reported cost; fail loudly on unknown model instead of returning 0 |
| Build wrapper | `.claude/commands/ant/build.md` | Delegation protocol section; relay `spawn-request` refusals in plain English |
| OpenCode wrapper | `.opencode/commands/ant/build.md` | Mirror |
| Agent definitions | `.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml` | Delegation instructions for castes granted it; correct the stale claim at `aether-route-setter.md:153` |
| Docs | `.aether/docs/wrapper-runtime-ux-contract.md` | Nested spawn is a wrapper/parent action under runtime adjudication — state it, or the next audit reads it as a boundary violation |

### UNCHANGED — and must stay so

| Component | Why |
|-----------|-----|
| `ExecutionBinding.ManifestSHA256` and `bindBuildAttemptManifest` | The digest must not become amendable |
| `queenBuildSafetyRequiredCastes` | Floors stay in Go, not in a file the model can write |
| `spawnThreshold` = 30 | Its comment records the measurement that killed retuning it |
| `pkg/codex/dispatch.go` wave semantics | Option B keeps children inside the parent's wall-clock |
| Wrapper prohibition on state mutation | Delegation adds no wrapper writes; `spawn-request` is a runtime call |

---

## Data Flow Changes

### Before

```
build --plan-only ──▶ manifest{dispatches[], briefs, capsule, binding}
                          │
   wrapper ───────────────┴──▶ spawn wave 1 ──▶ … ──▶ spawn wave N
                                    │
                          results[] (one per manifest dispatch)
                                    │
                   build-completion-stage ──▶ build-finalize
                                    │
                    claims  handoffs  state advance
                    (token usage: parsed, then discarded)
```

### After

```
build --plan-only ──▶ manifest{…, spawn_policy{max_depth, optional_budget_remaining}}
                          │
   wrapper ───────────────┴──▶ spawn wave 1
                                    │
                              worker (depth 1)
                                    │
                        ┌───────────┴──────────────┐
                        │  aether spawn-request    │  ← Go adjudicates, never spawns
                        │  → granted + child brief │
                        │  → refused + reason      │
                        └───────────┬──────────────┘
                                    │ granted
                        parent spawns child (platform Agent tool)
                        └── or ──▶ runtime queues a follow-on bound attempt
                                   the top-level wrapper spawns  (fallback channel)
                                    │
                       result{ …, child_results[], usage }
                                    │
                   build-completion-stage (schema now admits both fields)
                                    │
                            build-finalize
                       ├── claims: child files → parent's task claim
                       ├── handoffs: children stamped depth+parent, ranked below peers
                       ├── spend: trace.jsonl rows keyed on attempt run_id
                       └── state advance (unchanged)
                                    │
                   aether trace-summary --run-id <run> ──▶ per-caste, per-worker,
                                                           measured vs estimated
```

**Three new persistent facts:** child lineage in `spawn-tree.txt` (existing file, existing fields, first real use at depth 2); depth/parent on handoff records; token spend rows in `trace.jsonl`.

**No new store.** Every write lands in a file the runtime already owns and already locks.

---

## Suggested Build Order

Dependencies are real and mostly one-directional. This order lets each phase ship something a command can prove.

### Phase A — Spend accounting (no dependencies, unblocks measurement)

Ships first because every later decision — delegation budgets, digest size, roster model routing — should be argued from measured numbers rather than estimates, and because it is the smallest end-to-end slice.

1. `+ Usage` on `codexExternalBuildWorkerResult`; regenerate the committed schema
2. `pkg/trace` `LogWorkerSpend`; wire both the wrapper path (`build-finalize`) and the hosted path (`cmd/codex_build.go:1610`)
3. `trace-summary` rollups with a hard measured/estimated split
4. Fix `pkg/trace/cost.go` — prefer provider cost, fail loudly on unknown models

*Proof:* `aether trace-summary --run-id <run>` reports non-zero per-caste totals after a real build, and fails when a worker's usage is missing.

### Phase B — Agent roster loader (no dependencies; unblocks C and D)

Independent of A, sequenced second because delegation needs to look up a child caste's model, tools, and prompt from *somewhere*, and hardcoding a second lookup would be work thrown away.

1. `cmd/agent_roster.go` with 4-step resolution and compiled-in fallback
2. Extend `colony/agents/*.yaml` with the scoring and capability fields
3. Repoint `queenCasteRoster()` and `casteRelevanceScore` through the loader
4. `aether roster-validate`; add `colony/agents` to the hub sync pairs
5. Keep `casteRelevanceRegistry` as fallback; add a drift test asserting file and slice agree

*Proof:* `aether roster-validate` exits non-zero when a YAML entry loses its capability; deleting `colony/agents/` degrades to the compiled roster with a warning, not a crash.

### Phase C — Survey digest (depends on B only for caste-aware weighting; can run parallel)

1. `cmd/survey_digest.go` reusing `loadCodexSurveyContext`
2. Displace the pointer list in `resolveSurveySection`; keep the staleness notice
3. Measure against `TestBuildWorkerBriefIsMostlyTask` and `assembledContextTaskShareFloorPercent` — if either fails, shrink the digest, do not raise the floor
4. Record a before/after exhibit as Phase 162 did for memory injection

*Proof:* a named test asserts the brief contains a fact from a survey document that the phase text did not mention, and that the survey allocation did not grow.

### Phase D — Recursive delegation (depends on A for budget data, B for child agent lookup)

Largest and last. Split it:

**D1 — Adjudication, no spawning.** Replace the `spawn-can-spawn` stub with a real depth+budget check. Add `spawn-request` returning granted/refused with a composed child brief. Nothing spawns yet.
*Proof:* `spawn-can-spawn --depth 2` refuses; `--depth 1` grants; over-budget refuses with a reason.

**D2 — Result plumbing.** `codexChildWorkerResult`, schema regeneration, claims merge, handoff persistence with depth/parent, relay ranking.
*Proof:* a fixture completion packet with a child result finalizes; the child's files appear in the parent's task claims; the child's handoff does not evict a peer's from the top-5.

**D3 — Follow-on wave channel.** The guaranteed path: a granted request that the parent cannot execute becomes a new bound attempt the top-level wrapper spawns.
*Proof:* an e2e run where the parent's platform reports no Agent tool still completes the delegated work.

**D4 — Parent-direct spawn.** The opportunistic path, gated on runtime capability detection. Wrapper and agent-definition updates.
*Proof:* a real build where a Builder delegates to a Scout and the Scout's finding appears in the parent's handoff.

### Ordering rationale

- **A before D** — delegation budgets set without spend data are guesses, and the project has a documented history of unfalsifiable efficiency claims.
- **B before D** — a child dispatch needs a caste's agent file, model, and tools. Two lookup paths would be built and one discarded.
- **C is independent** — it can run alongside B or D; it touches only brief composition.
- **D1 before D2 before D3/D4** — each is separately provable, and D1 alone closes a live "documented but never implemented" defect regardless of whether the rest ships.

---

## Anti-Patterns to Avoid

### Making the manifest digest amendable

**What people do:** add child dispatches to the bound manifest and recompute `ManifestSHA256`.
**Why it's wrong:** the digest exists so a result from a stale checkout or superseded attempt cannot be accepted (`cmd/build_attempt.go:280`). A digest recomputed on demand authenticates nothing.
**Instead:** children ride inside the parent's result (Option B), or the runtime issues a *new bound attempt* (Option C).

### Routing delegation through the TypeScript host

**What people do:** reach for `spawn-orchestrator.ts`, which already implements exactly this policy.
**Why it's wrong:** `.claude/commands/ant/build.md` explicitly forbids `aether host build` on the interactive path. Reintroducing the hop reverses a decision that was made deliberately.
**Instead:** port the policy — depth cap, budget accounting, fail-closed rejection — into Go. It is roughly 150 lines and the TS file is a good specification.

### Deleting the hardcoded caste slice in the same change that adds the loader

**What people do:** move the registry to YAML and remove the Go table in one commit.
**Why it's wrong:** the project's dominant failure mode across 18 of 25 milestones is "the old thing was removed and the new thing was never wired." A missing `colony/` directory in a downstream repo would then produce an empty roster and a build with no workers.
**Instead:** overlay, prove agreement with a drift test, and consider deletion in a later milestone with the test as evidence.

### Letting delegation spend the safety castes' budget

**What people do:** decrement one shared worker counter for both required and delegated workers.
**Why it's wrong:** `queenBuildSafetyRequiredCastes` deliberately bypasses the cap so that choosing "light" removes optional specialists and never safety ones. A Builder that delegates three times could consume the Watcher's slot and produce a build nobody checked — the exact outcome `TestWatcherIsAlwaysRequiredOnBuild` exists to prevent.
**Instead:** delegation draws only from the optional remainder, computed as `MaxWorkers - len(RequiredCastes)`, floored at zero.

### Adding the survey digest on top of the pointer list

**What people do:** keep the filenames and prepend the digest.
**Why it's wrong:** the per-section budgets already sum to ~17,700 against a 24,000 cap, and `TestBuildWorkerBriefIsMostlyTask` names survey pointers as scaffolding.
**Instead:** the digest replaces the pointer list, keeping at most the two or three documents actually worth opening.

### Reporting estimated tokens as measured

**What people do:** sum `WorkerUsage.TotalTokens` for a run total.
**Why it's wrong:** `EstimateUsage` fills gaps with a crude chars/4 ratio, tagged `UsageSourceEstimate`. Blending makes a regression look like an improvement — the failure `pkg/codex/usage.go`'s header comment was written to end.
**Instead:** two totals, always, with the estimated count shown as a confidence caveat.

---

## Open Questions for Requirements

1. **Does a delegated child's time come out of the parent's timeout, or get its own?** `WorkerDispatch.Timeout` currently means "one worker." Under Option B it silently becomes "one subtree." Pick one and name it in the contract.
2. **Which castes may delegate?** All, or an allowlist? An allowlist is smaller, safer, and matches how `casteAllowedForFlow` already restricts flows. Recommend starting with Builder and Tracker only.
3. **Does an empty child handoff fail the build?** Current rule rejects empty handoffs. Recommend requiring the parent to absorb the child's findings rather than exempting children — one rule, and it forces the parent to read what it delegated.
4. **Should the roster own model routing?** `colony/policies/model-routing.yaml` still has zero Go readers and `colony/agents/*.yaml` already carries a `model:` field. These overlap. Deciding which is authoritative belongs in requirements, not implementation.
5. **Is the survey digest caste-aware?** A Gatekeeper wants SENTINEL-PROTOCOLS.md; a Builder wants CHAMBERS.md. Caste weighting is cheap once the roster loader exists (Phase B), but it is a scope decision.

---

## Sources

**Primary — repository source, read directly (HIGH confidence):**
- `cmd/queen_judgement.go`, `cmd/queen_spawn_budget.go`, `cmd/caste_relevance.go` — team selection, floors, budget
- `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, `cmd/build_attempt.go` — manifest, binding, finalize
- `cmd/contract_schema.go`, `.aether/schemas/completion-packet.schema.json` — reflected packet schema
- `cmd/spawn.go`, `pkg/agent/spawn_tree.go` — spawn adjudication stub and lineage store
- `cmd/helpers.go:225`, `cmd/codex_plan.go:1645`, `cmd/codegraph_context.go` — survey and context-slice patterns
- `pkg/codex/usage.go`, `pkg/codex/platform_dispatch.go`, `pkg/trace/{trace,cost}.go`, `cmd/trace_cmds.go` — spend
- `cmd/policy_loader.go`, `cmd/visuals_config.go`, `cmd/prompt_template_loader.go` — the three proven YAML-overlay loaders
- `.aether/ts-host/src/{types,spawn-orchestrator}.ts` — prior recursive-delegation implementation (v1.21)
- `.claude/commands/ant/build.md`, `.aether/docs/wrapper-runtime-ux-contract.md` — the boundary being respected
- `cmd/codex_build_test.go:3629`, `cmd/context_budget_test.go:24` — the budget invariants this milestone must not retune

**Secondary — platform nested-spawn behaviour (MEDIUM confidence, changes fast):**
- [Create custom subagents — Claude Code Docs](https://code.claude.com/docs/en/sub-agents)
- [Sub-Agent Task Tool Not Exposed When Launching Nested Agents · anthropics/claude-code#4182](https://github.com/anthropics/claude-code/issues/4182)
- [Subagent tool stripped in nested subagent calls · anthropics/claude-code#80036](https://github.com/anthropics/claude-code/issues/80036)
- [Task tool permission override no longer works for nested sub-agents · anomalyco/opencode#8114](https://github.com/anomalyco/opencode/issues/8114)
- [Subagents can infinitely recurse via Task tool — no max depth limit · anomalyco/opencode#18100](https://github.com/anomalyco/opencode/issues/18100)
- [Custom agents cannot access task tool despite frontmatter configuration · anomalyco/opencode#14308](https://github.com/anomalyco/opencode/issues/14308)

*Platform caveat: nested-spawn availability and depth defaults have each changed on both platforms within the last quarter. Any requirement that depends on parent-direct spawning should carry a runtime capability check rather than a version assumption.*

---
*Architecture research for: Aether v1.26 Intelligent Orchestration*
*Researched: 2026-08-08*
