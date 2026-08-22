---
last_mapped_commit: 92252d01
---

<!-- refreshed: 2026-08-22 -->
# Architecture

**Analysis Date:** 2026-08-22

## System Overview

```text
┌──────────────────────────────────────────────────────────────────────┐
│                    WRAPPER LAYER (Presentation)                       │
│                                                                       │
│  .claude/commands/ant/*.md  (Claude Code)                             │
│  .opencode/commands/ant/*.md (OpenCode)                               │
│  .codex/agents/*.toml       (Codex — agent definitions only)          │
│                                                                       │
│  Role: Render user experience, spawn workers via platform tools,     │
│        manage ceremony and narration, read runtime output.            │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                    (fetch manifest)
                    (spawn workers)
                    (finalize results)
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                    GO RUNTIME (Truth Owner)                          │
│                                                                       │
│  cmd/                                                                 │
│  ├── root.go              CLI entry + store initialization            │
│  ├── codex_build.go       Build orchestration and dispatch logic      │
│  ├── codex_continue.go    Verification + gating + learning           │
│  ├── codex_continue_plan.go (Continue manifest generation)           │
│  ├── codex_continue_finalize.go (State commitment)                  │
│  ├── codex_build_finalize.go (Build result absorption)              │
│  ├── skills.go            Worker brief assembly + skill matching     │
│  ├── criterion_evidence.go (Success criteria verification)           │
│  ├── consolidation_lifecycle.go (Learning consolidation)            │
│  └── ...                  (80+ subcommands for orchestration)         │
│                                                                       │
│  pkg/                                                                 │
│  ├── agent/               Agent pool, spawn tree, task routing       │
│  ├── codex/               Runtime state machines, contracts           │
│  ├── colony/              Phase plans, tasks, state definitions       │
│  ├── memory/              Learning pipeline, instincts, promotion    │
│  ├── storage/             JSON persistence, file locking             │
│  ├── events/              Event bus with TTL                         │
│  ├── graph/               Knowledge graph (instinct relationships)   │
│  ├── exchange/            XML import/export for signals/wisdom       │
│  ├── llm/                 LLM provider routing + adapters            │
│  ├── learn/               Structural learning stack, ants            │
│  └── trace/               Audit logging                              │
│                                                                       │
│  Role: Owns all state mutations, verification, gating, learning,    │
│        phase advancement, and command orchestration.                │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                   (JSON serialization)
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                    LOCAL STATE (Gitignored)                          │
│                                                                       │
│  .aether/data/                                                        │
│  ├── COLONY_STATE.json      Phase plan + phase status + instincts   │
│  ├── session.json            Session bookkeeping                     │
│  ├── pheromones.json         Active signals (FOCUS/REDIRECT/FEEDBACK)│
│  ├── pending-decisions.json  Unanswered open decisions               │
│  ├── assumptions.json        Plan assumptions + validation           │
│  ├── behavior-observations.jsonl (Learning observations)            │
│  ├── midden/                 Failure tracking per category           │
│  ├── survey/                 Territory survey results                │
│  └── handoffs/               Worker-to-worker relay notes            │
│                                                                       │
│  .aether/data/phase-research/  (Scout writes phase research)        │
│  .aether/data/worker-debug/    (Debug artifacts from workers)       │
│                                                                       │
│  Role: Source of truth for colony state, governed by runtime only.  │
└──────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| **Runtime (Go)** | State mutations, orchestration, verification, gating, learning | `cmd/*.go`, `pkg/` |
| **Wrapper (Markdown)** | Ceremony, narration, worker spawning, user interaction | `.claude/commands/ant/`, `.opencode/commands/ant/` |
| **Agent Pool** | Parallel task dispatch, streaming, resource limits | `pkg/agent/pool.go` |
| **Skill Matcher** | Worker context assembly, skill injection | `cmd/skills.go` |
| **Build Flow** | Phase → Tasks → Dispatches → Workers → Results | `cmd/codex_build.go` |
| **Continue Flow** | Verification → Gates → Learning → Advance | `cmd/codex_continue.go` |
| **State Store** | JSON persistence, atomic updates, file locking | `pkg/storage/store.go` |
| **Learning Pipeline** | Observations → Instincts → QUEEN.md → Hive wisdom | `pkg/memory/pipeline.go` |

## Pattern Overview

**Overall:** Layered separation of concerns with Go runtime owning state and wrappers owning presentation.

**Key Characteristics:**
- **State centralization:** All mutable state lives in `.aether/data/` JSON files, governed exclusively by Go runtime commands
- **Wrapper delegation:** Markdown wrappers fetch JSON manifests, spawn workers via platform tools, and finalize results by handing JSON to the runtime
- **Contract-driven:** Communication between wrapper and runtime is mediated by YAML command definitions (`.aether/commands/*.yaml`) and documented contracts (`.aether/docs/wrapper-host-contract.md`)
- **Manifest-based dispatch:** Build and heavy-continue use plan-only to generate manifests; wrappers interpret manifests and spawn workers; finalizers absorb results

## Layers

**Wrapper Layer (Presentation):**
- Purpose: User experience, ceremony, worker spawning, narration
- Location: `.claude/commands/ant/`, `.opencode/commands/ant/`, `.codex/agents/`
- Contains: Markdown command files + Codex agent definitions
- Depends on: Go runtime (via CLI and JSON output)
- Used by: Users via CLI slash commands or platform UI

**Runtime Layer (Orchestration & Truth):**
- Purpose: Command execution, state management, orchestration, verification
- Location: `cmd/*.go`
- Contains: Cobra command handlers, main control flows, event dispatch
- Depends on: Shared packages (`pkg/`)
- Used by: Wrappers (via CLI), other commands via RPC

**Shared Packages (Infrastructure):**
- Purpose: Reusable logic for agents, state, learning, events
- Location: `pkg/agent/`, `pkg/colony/`, `pkg/memory/`, `pkg/storage/`, `pkg/codex/`, `pkg/events/`, `pkg/graph/`, `pkg/exchange/`
- Contains: Type definitions, business logic, persistence adapters
- Used by: Runtime (`cmd/`) and one another

**State Layer (Truth Store):**
- Purpose: Persistent truth for colony, phases, learnings
- Location: `.aether/data/` (gitignored)
- Contains: JSON files (COLONY_STATE.json, session.json, pheromones.json, etc.)
- Accessed: Only via Go runtime commands; wrappers never read/write directly

## Data Flow

### Primary Request Path (Build → Continue)

1. **User invokes `/ant-build N`** (`.claude/commands/ant/build.md`)
2. **Wrapper fetches manifest:** `aether build N --plan-only` → returns `codexBuildManifest` JSON (`cmd/codex_build.go:runCodexBuildPlanOnly`)
3. **Runtime assembles manifest:**
   - Loads phase from COLONY_STATE.json
   - Breaks phase into tasks
   - Composes worker briefs (phase objective, constraints, pheromones, skills, prior handoffs)
   - Selects castes via Queen judgement (`cmd/codex_build.go:castesForBuild`)
   - Generates dispatches with wave execution plan
4. **Wrapper renders spawn plan** and spawns workers via platform agents using `agent_name` from each dispatch
5. **Worker executes with injected brief**, produces results (changed files, test evidence, errors)
6. **Wrapper finalizes results:** Stages completion JSON, then calls `aether build-finalize --completion-file <path>` (`cmd/codex_build_finalize.go`)
7. **Runtime absorbs results:**
   - Verifies claims match evidence
   - Stores worker outputs and artifacts
   - Updates task status
   - Returns closeout JSON
8. **User invokes `/ant-continue`** (`.claude/commands/ant/continue.md`)
9. **Runtime verification (default path):** `aether continue --verification-depth standard` runs verification commands, gate checks, and learning all in-process (`cmd/codex_continue.go`)
   - Runs verification steps (build, type check, lint, test)
   - Checks gate criteria (security, quality, performance)
   - Captures learning observations
   - Advances phase if gates pass
   - Returns verification + gate + learning report
10. **Wrapper renders closeout**, routes to next phase or `/ant-seal`

### Heavy Continue Path (Manifest-Based Review)

1. **User invokes `/ant-continue --verification-depth heavy`** or `--classic-ceremony`
2. **Wrapper fetches manifest:** `aether host continue --dry-run --classic-ceremony $ARGS` → TS host calls `aether continue --plan-only --classic-ceremony` → returns `codexContinueManifest` JSON
3. **Runtime assembles continue manifest:**
   - Runs default verification (same as step 9 above)
   - If deep review requested, generates dispatches for Watcher + Gatekeeper + Auditor + Probe
   - Includes candidate tasks for reconciliation, artifact evidence paths
4. **Wrapper renders spawn plan** and spawns review workers via agents
5. **Reviewers consume brief verbatim**, analyze code/tests/artifacts, produce review summary
6. **Wrapper finalizes:** `aether continue-finalize --completion-file <path>` (`cmd/codex_continue_finalize.go`)
7. **Runtime commits learning and advances phase**

### State Mutation Boundaries

- **Read-only by wrapper:** Only via `aether status` (visual output), manifest commands (plan-only)
- **Mutated only by runtime:** Build-finalize, continue-finalize, advance-phase, seal
- **Never by wrapper:** COLONY_STATE.json, session.json, pheromones.json, instincts.json

## Key Abstractions

**Phase & Task:**
- Purpose: Represent a unit of work and its decomposition
- Examples: `pkg/colony/phase.go`, `cmd/codex_build.go` (task planning)
- Pattern: Immutable data structures loaded from COLONY_STATE.json, used to generate dispatches

**Dispatch:**
- Purpose: One worker assignment with full context
- Examples: `cmd/codex_build.go:codexBuildDispatch` (build), `cmd/codex_continue.go` (review)
- Pattern: Manifest includes all dispatches; wrapper interprets and spawns; finalizer absorbs results

**Manifest:**
- Purpose: Complete, immutable plan for a workflow (build or continue)
- Examples: `cmd/codex_build.go:codexBuildManifest`, `cmd/codex_continue.go:codexContinueManifest`
- Pattern: Returned by plan-only commands; wrapper uses it to spawn workers; no in-manifest mutation

**Worker Brief:**
- Purpose: Full context injected into a worker's prompt
- Includes: Phase objective, constraints, hints, success criteria, pheromones, skills, prior handoffs, decision clarifications
- Location: Written to disk by runtime at brief generation time; passed verbatim to worker (`cmd/codex_build.go:Brief` field)
- Pattern: Wrapper reads `brief_path` from dispatch and passes text to agent; agent receives it as part of prompt context

**Colony State:**
- Purpose: Complete, persistent snapshot of colony progress
- Location: `.aether/data/COLONY_STATE.json`
- Contains: Goal, plan, phase list, phase status, instincts with confidence scores, pheromones
- Mutated: Only by runtime finalizers (`build-finalize`, `continue-finalize`, phase advance)
- Pattern: Loaded fresh on every command; atomically updated via `store.UpdateJSONAtomically()`

**Instinct:**
- Purpose: Learned pattern with provenance and confidence
- Examples: `pkg/colony/instincts.go`, `pkg/memory/pipeline.go`
- Pattern: Captured during build, promoted during continue-finalize, consolidated at seal, promoted to hive if confidence >= 0.8

**Pheromone (Signal):**
- Purpose: User-directed colony steering (FOCUS/REDIRECT/FEEDBACK)
- Location: `.aether/data/pheromones.json`
- Pattern: User emits via `/ant-focus`, `/ant-redirect`, `/ant-feedback`; runtime injects into colony-prime context; workers see and respond; signals decay or expire at phase boundaries

**Hive Wisdom:**
- Purpose: Cross-colony learned patterns
- Location: `~/.aether/hive/wisdom.json` (hub-level, shared across all repos)
- Pattern: High-confidence instincts promoted at seal; domain-scoped retrieval during colony-prime context assembly

## Entry Points

**Primary User Commands:**
- `/ant-init "<goal>"` → `cmd/init_cmd.go` (initializes COLONY_STATE.json with plan)
- `/ant-plan` → `cmd/plan_cmd.go` (generates phase plan)
- `/ant-build <phase>` → `.claude/commands/ant/build.md` (wrapper) → `aether build $ARGS --plan-only` (runtime)
- `/ant-continue` → `.claude/commands/ant/continue.md` (wrapper) → `aether continue` (runtime, default) or `aether host continue --dry-run` (heavy path)
- `/ant-seal` → `cmd/seal_cmd.go` (finalizes colony, promotes learnings to hive)

**Go Runtime Commands (Invoked by Wrappers):**
- `aether build <phase> --plan-only` → `cmd/codex_build.go:runCodexBuildPlanOnly()` (generates JSON manifest)
- `aether build-finalize --completion-file <file>` → `cmd/codex_build_finalize.go` (absorbs worker results)
- `aether continue [--verification-depth <level>]` → `cmd/codex_continue.go:runCodexContinue()` (all-in-one verification + gating + learning)
- `aether host continue --dry-run --classic-ceremony` → TS host (Typescript; spawned as subprocess) calls back `aether continue --plan-only --classic-ceremony`
- `aether continue-finalize --completion-file <file>` → `cmd/codex_continue_finalize.go` (absorbs review results, commits learning)

**Administrative Commands:**
- `aether status` → Visual output (JSON with `--output json`)
- `aether phase <N>` → Phase details
- `aether colonize` → Codebase analysis (scaffolds .aether/)
- `aether seal` → Finalize colony, promote instincts to hive

## Architectural Constraints

- **Threading:** Go runtime is single-threaded per invocation (one CLI run = one command = one thread sequence). Agent pool uses goroutines for concurrent worker execution during `aether build` and `aether host build` (subprocess), but the interactive wrapper runs sequentially.
- **Global state:** Store (storage.Store) is initialized once at CLI startup; shared across command execution via package-level `var store *storage.Store` in `cmd/root.go`. No concurrent mutation; `UpdateJSONAtomically()` serializes updates via file locking.
- **Circular imports:** None enforced; packages are layered: `cmd/` imports `pkg/*`, `pkg/` does not import `cmd/`.
- **Manifest immutability:** Once a manifest is returned by a plan-only command, it is never re-fetched during that workflow. Wrapper interprets it and spawns workers; no in-flight manifest updates.
- **Worker isolation:** Each worker receives the full brief at spawn time; no in-prompt context sharing or consensus building between workers during execution.
- **Output contracts:** All JSON output is defined by strict struct types (e.g., `codexBuildManifest`, `codexContinueReport`). Wrappers parse only declared fields.

## Cross-Cutting Concerns

**Logging:** Audit trail via `pkg/trace/tracer.go`; events written to `.aether/data/events/` JSONL files with TTL cleanup.

**Validation:** Success criteria defined in phase; verified by `cmd/criterion_evidence.go` during continue gate checks. Claims (files, tests) verified against evidence.

**Authentication:** Not built into runtime; delegated to platform (user's IDE login, GitHub token for git, etc.). Runtime does not manage secrets.

**Learning & Consolidation:** Observations captured during worker handoff → promoted to instincts → consolidated at seal via 9-ant curation pipeline (`pkg/learn/curation_ants.go`) → high-confidence instincts promoted to hive.

**Signal Management:** Pheromones (FOCUS/REDIRECT/FEEDBACK) managed via `cmd/pheromone_*.go` commands; injected into colony-prime context at worker dispatch time; decay/expiry at phase boundaries.

## Anti-Patterns

### State Mutation in Wrappers

**What happens:** Wrapper directly edits COLONY_STATE.json, session.json, or pheromones.json.

**Why it's wrong:** State is the runtime's source of truth. Wrapper mutations bypass runtime invariants (atomic updates, consistency checks). Next runtime command loads stale or corrupted state.

**Do this instead:** Route all state changes through Go runtime commands. For steering, use `/ant-focus`, `/ant-redirect`, `/ant-feedback` (which call runtime subcommands). For manual state surgery, use `aether state-mutate` (developer-only, logged to audit trail).

**Test:** `cmd/command_guide.go` documents guardrails; `cmd/codex_build_test.go` asserts wrapper contract violations.

### Manifest Reuse Across Workflows

**What happens:** Wrapper fetches a manifest for build, then attempts to reuse it for a subsequent continue after user makes decisions.

**Why it's wrong:** Manifest is a point-in-time snapshot. If user invokes `/ant-focus` or `/ant-redirect` between build and continue, the manifest is stale and no longer reflects current signals. Workers see outdated context.

**Do this instead:** Fetch a fresh manifest after any workflow-altering user decision. Build.md enforces this: after discuss redirects, the guard says "request a fresh manifest after_discuss_next; never reuse a pre-discuss manifest."

**Test:** `cmd/orchestrator_boundary_clarification.go` guards orchestrator transitions; `cmd/codex_build_test.go` tests manifest freshness after discuss.

### Wrapper-Based Verification Reimplementation

**What happens:** Wrapper parses build output and tries to determine if verification passed, rather than trusting the runtime's verification report.

**Why it's wrong:** Verification logic (success criteria, gate policies, failure classification) is complex and owned by the runtime. Wrapper reimplementation drifts from reality.

**Do this instead:** Parse the JSON report from `aether continue` (or the continue-finalize result). Trust `passed`, `criteria_passed`, `blocking_issues` fields.

**Test:** `cmd/codex_continue_test.go` asserts verification gate contracts.

## Error Handling

**Strategy:** Two-tier (checked errors in Go, visible recovery in wrapper).

**Patterns:**
- **Build errors:** Worker produces output → claims checked against evidence → if claims fail, task marked failed → finalize returns error detail; wrapper routes to `/ant-continue` for retry or `/ant-redirect` for guidance
- **Verification errors:** Verification step fails → classified by error class (product, environment, timeout) → gate fails → finalize returns recovery command; wrapper follows recovery (e.g., "re-run build with X environment var set")
- **Learning errors:** Promotion failures logged but never block seal; seal completes and returns warnings
- **Orchestration errors:** Runtime returns structured error with `next_action`; wrapper follows it (e.g., "Run `/ant-discuss`")

## Verification Gates

**Goal:** Ensure build work is correct and safe before advancing.

**Gates run during `/ant-continue`:**
1. **Verification:** Build commands (compile, test, lint) re-run in the repo after workers complete. Pass/fail only; not a review.
2. **Criteria:** Success criteria defined in phase checked against artifacts produced (files, test counts, coverage %).
3. **Gatekeeper:** Security gate (secret detection, antipattern scan).
4. **Auditor:** Quality gate (code metrics, no regression).
5. **Probe:** Test coverage gate (analyzer suggests gaps).

**Watcher (default continue path only):** If a verification step fails and there's no deterministic recovery, Watcher analyzes the failure and suggests a fix. Wrapper offers it as an action (e.g., "Add an environment variable" or "Run again — the test is flaky").

---

*Architecture analysis: 2026-08-22*
