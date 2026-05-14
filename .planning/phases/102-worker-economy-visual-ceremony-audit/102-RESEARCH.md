# Phase 102: Worker Economy & Visual Ceremony Audit - Research

**Researched:** 2026-05-07
**Domain:** Worker dispatch economy and visual ceremony audit (Go runtime)
**Confidence:** HIGH

## Summary

This phase audits two interconnected systems in the Aether Go runtime: (1) the worker economy, which ensures every spawned caste has documented purpose, durable output, and downstream consumers, and (2) the visual ceremony, which ensures every rendered visual element traces to real runtime state rather than decorative fiction.

The authoritative caste registry lives in three maps in `cmd/codex_visuals.go`: `casteEmojiMap`, `casteColorMap`, and `casteLabelMap` (27 entries each). Worker dispatch happens through five orchestration commands: build, continue, seal, colonize, and plan. Each has a distinct wave shape documented below. Visual output is rendered through `renderStageMarker()`, `renderBanner()`, `renderProgressSummary()`, `renderAetherWordmark()`, and `renderSpawnPlanForDispatches()` -- all in `cmd/codex_visuals.go`. Ceremony lifecycle events flow through `cmd/ceremony_emitter.go` and are backed by the `pkg/events` event bus.

**Primary recommendation:** Follow Phase 101's KNOWN-GAPS.md pattern. Produce a combined WORKER-ECONOMY.md report with severity-classified findings across both dimensions, plus a spawn coverage test that cross-references dispatch sites against documented purpose/output entries.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Audit all 27 defined worker castes -- not just actively dispatched ones. Castes defined but never spawned are flagged as "unused" findings.
- **D-02:** For each caste, verify: (1) documented purpose, (2) expected durable output, (3) downstream consumer. Table format.
- **D-03:** Castes returning only chat without persisting are flagged as WORK-02 violations. No fix suggestions in report.
- **D-04:** All 5 visual element categories in scope: caste identity, stage markers, progress bars, closeout banners, Aether ASCII wordmark.
- **D-05:** Pure decorative output (like Aether wordmark) is acceptable. Only flag elements that claim a state transition without backing runtime change.
- **D-06:** Per-command wave shape tables for build, continue, seal, colonize, plan. Each shows castes, why, and what they produce.
- **D-07:** Per-command tables, not unified cross-reference matrix.
- **D-08:** Single combined report (WORKER-ECONOMY.md). Same severity pattern as Phase 101's KNOWN-GAPS.md.
- **D-09:** Automated spawn coverage test verifying every dispatched caste has documented purpose/output entry.

### Claude's Discretion
- Exact test file structure and naming
- How to extract caste spawn sites from dispatch code (grep patterns vs AST)
- Whether to include spawn frequency data alongside purpose/output/consumer
- Visual ceremony verification method (static analysis of output call sites vs runtime tracing)
- Report section ordering and formatting details

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WORK-01 | Every spawned worker caste has documented purpose, durable output, and downstream consumer | Caste registry (27 entries), dispatch sites in cmd/codex_*.go, agent definitions in .claude/agents/ant/*.md, .aether/workers.md |
| WORK-02 | No worker type spawned that only reads and returns chat without persisting | Agent definitions show tool access (Write tool = persists), continue review specs show ledger persistence instructions |
| WORK-03 | Build/continue/seal/colonize/plan wave shapes documented and each spawn justified | Wave shape tables below from dispatch code analysis |
| VIZ-01 | Caste colors, stage markers, live worker stacking, and closeout banners reflect real runtime state | Visual rendering chain: codex_visuals.go renders from dispatch structs, ceremony_emitter.go emits from lifecycle events |
| VIZ-02 | No decorative output hiding missing behavior or pretending a state transition | D-05 distinguishes pure decoration from misleading state claims; renderBuildVisual reads real colony state fields |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Worker dispatch planning | Go runtime (cmd/) | -- | `plannedBuildDispatches()` in codex_build.go owns spawn logic |
| Worker execution | Platform wrapper | Go runtime | Wrappers use Task tool; runtime provides manifests via --plan-only |
| Visual rendering | Go runtime (cmd/) | -- | codex_visuals.go is authoritative renderer |
| Ceremony lifecycle events | Go runtime (ceremony_emitter.go) | -- | Event bus publishes state transitions |
| Agent purpose documentation | .claude/agents/ant/*.md | .aether/workers.md | Agent frontmatter descriptions are purpose statements |
| Durable output persistence | Agent definitions (Write tool) | Runtime (review-ledger-write) | Agents with Write tool persist; runtime provides ledger CLI |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (testing) | Go 1.24 | Test framework | Project standard, all 2900+ tests use it |
| pkg/codex | local | WorkerDispatch and DispatchResult types | Authoritative dispatch types in pkg layer |
| pkg/colony | local | ColonyState, Phase, Task, VerificationDepth types | Authoritative state types |
| pkg/agent | local | SpawnTree, SpawnEntry types | Authoritative spawn tracking |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| regexp | Go stdlib | Pattern matching for spawn site extraction | Grepping dispatch code for caste string literals |
| encoding/json | Go stdlib | Golden file comparison | Spawn coverage test against documented caste table |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Grep-based spawn extraction | Go AST parsing | AST more precise but grep matches existing test patterns and is simpler for string literal extraction |

**Installation:**
No new dependencies needed -- this phase uses only existing Go stdlib and project packages.

## Architecture Patterns

### System Architecture Diagram

```
.aether/workers.md (purpose statements)
.claude/agents/ant/*.md (agent definitions)
.codex/agents/*.toml (Codex agents)
        |
        v
cmd/codex_visuals.go (casteEmojiMap/ColorMap/LabelMap)  <--- AUTHORITATIVE REGISTRY (27 castes)
        |
        v
cmd/codex_build.go --plannedBuildDispatches()--> Build wave shape
cmd/codex_continue.go --codexContinueReviewSpecs--> Continue review wave
cmd/codex_plan.go --plannedPlanningWorkers()--> Plan wave shape
cmd/codex_colonize.go --surveyorSpecs--> Colonize wave shape
cmd/seal_final_review.go --sealFinalReviewRequiredCastes--> Seal wave shape
cmd/oracle_loop.go --> Oracle dispatch shape
cmd/swarm_cmd.go --> Swarm dispatch shape
        |
        v
cmd/codex_visuals.go --render*Visual()--> Visual output (banners, markers, progress)
cmd/ceremony_emitter.go --emit*()--> Ceremony events (lifecycle transitions)
        |
        v
WORKER-ECONOMY.md (audit report)
cmd/*_test.go (spawn coverage test)
```

### Recommended Project Structure
```
.planning/phases/102-worker-economy-visual-ceremony-audit/
├── 102-RESEARCH.md          # This file
├── 102-CONTEXT.md           # Locked decisions (exists)
├── WORKER-ECONOMY.md        # Combined audit report (output)
cmd/
├── worker_economy_test.go   # Spawn coverage test (output)
```

### Pattern 1: Caste Dispatch Audit
**What:** Each caste has three properties to verify: purpose, durable output, downstream consumer.
**When to use:** For every caste in `casteEmojiMap`.
**Example:**
```go
// Verified: builder caste is spawned in codex_build.go line 741
// Purpose: "Implement code, execute commands, manipulate files" (.claude/agents/ant/aether-builder.md)
// Durable output: Files created/modified, tests written (ClaimsSummary)
// Downstream consumer: build-finalize collects claims, continue verifies
```

### Pattern 2: Visual Truth Verification
**What:** Every visual output function reads from real state, not hard-coded strings.
**When to use:** For each `render*Visual()` function.
**Example:**
```go
// renderBuildVisual reads real colony state:
// - phase.ID, phase.Name, phase.Description from colony.Phase
// - len(state.Plan.Phases) for progress bar
// - dispatches from plannedBuildDispatches() for spawn plan
// All trace to actual runtime data, not decoration
```

### Anti-Patterns to Avoid
- **Mixing audit with remediation:** This phase documents findings only. Phase 105 acts on them.
- **Testing visual rendering aesthetics:** Test that visual functions exist and read state, not that they produce specific pixel output.
- **Flagging pure decoration:** The Aether wordmark (ASCII art) is decorative. D-05 explicitly allows this. Only flag visuals that imply state transitions.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Caste enumeration | Custom caste list | `casteEmojiMap` in codex_visuals.go | Already the authoritative registry with 27 entries |
| Dispatch site detection | AST parser for Go code | Grep for `Caste: "castename"` string literals | Dispatch code uses consistent literal pattern; grep matches existing test approach |
| Wave shape extraction | Runtime tracing | Static analysis of `plannedBuildDispatches()` and friends | Dispatch planning is deterministic; static analysis is sufficient |
| Severity classification | Custom severity system | Phase 101's Critical/Warning/Info pattern | Consistency with existing audit reports |

**Key insight:** The dispatch code follows a consistent pattern where caste names appear as string literals in struct initialization. This makes grep-based extraction reliable and avoids the complexity of Go AST parsing.

## Runtime State Inventory

> This is an audit phase (read-only). No state mutations, no renames, no migrations.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Colony state artifacts in .aether/data/ | Read-only access for audit verification |
| Live service config | None -- no external services involved | -- |
| OS-registered state | None | -- |
| Secrets/env vars | None | -- |
| Build artifacts | None -- documentation and test output only | -- |

**Nothing found requiring data migration or code edit in runtime state.** This phase is strictly an audit producing documentation and tests.

## Common Pitfalls

### Pitfall 1: Confusing "defined but unused" with "defined and dispatched"
**What goes wrong:** Some castes appear in `casteEmojiMap` but are never dispatched in cmd/ code. These are valid findings (unused castes) but not violations.
**Why it happens:** The 27-caste map was built as a comprehensive registry; not all castes are actively dispatched by the runtime.
**How to avoid:** Separate findings into "never dispatched" (Info) vs "dispatched without documented output" (Warning).
**Warning signs:** Finding castes like "guardian", "includer", "chronicler", "sage", "dreamer" in the map but not in dispatch code.

### Pitfall 2: Missing dispatch sites outside build/continue
**What goes wrong:** Only auditing `codex_build.go` and `codex_continue.go`, missing dispatches in `oracle_loop.go`, `swarm_cmd.go`, and `council.go`.
**Why it happens:** Build and continue are the primary paths, but oracle and swarm also spawn workers.
**How to avoid:** Search all `cmd/*.go` files for `Caste:` string patterns, not just build/continue files.
**Warning signs:** An agent definition exists but no dispatch site is found -- check oracle, swarm, and council.

### Pitfall 3: Flagging ceremony-only visual output as misleading
**What goes wrong:** The Aether ASCII wordmark or spawn plan header being flagged as "not reflecting real state."
**Why it happens:** D-05 draws the line at state transitions, not all visual output.
**How to avoid:** Apply the test: "does this visual element imply something happened in the runtime?" If no (pure decoration), it passes. If yes, verify the backing state change exists.
**Warning signs:** Marking `renderAetherWordmark()` or `renderBanner()` titles as findings.

### Pitfall 4: Treating agent documentation as runtime truth
**What goes wrong:** Agent .md files describe ideal behavior, but the actual dispatch code may not enforce all described outputs.
**Why it happens:** Agent definitions are documentation for LLM workers; the runtime doesn't programmatically verify agent compliance.
**How to avoid:** Cross-reference dispatch code (what the runtime actually asks for) with agent definitions (what the agent promises to produce).
**Warning signs:** An agent definition says "returns structured findings" but the dispatch code never passes a findings output path.

## Code Examples

### Extracting caste spawn sites from dispatch code
```go
// Pattern: search cmd/ for Caste: "castename" in struct literals
// Each match represents a dispatch site where a worker of that caste is spawned
// Example matches found during research:
//   cmd/codex_build.go:741       Caste: "builder"
//   cmd/codex_build.go:755       Caste: "watcher"
//   cmd/codex_build.go:767       Caste: "chaos"
//   cmd/codex_continue.go:1029   Caste: "gatekeeper"
//   cmd/codex_continue.go:1034   Caste: "auditor"
//   cmd/codex_continue.go:1039   Caste: "probe"
//   cmd/codex_plan.go:745        Caste: "scout"
//   cmd/codex_plan.go:756        Caste: "route_setter"
//   cmd/codex_colonize.go:579    Caste: "surveyor-provisions"
//   cmd/oracle_loop.go:1348      Caste: "oracle"
//   cmd/swarm_cmd.go:828         Caste: "tracker"
//   cmd/swarm_cmd.go:846         Caste: "archaeologist"
```

### Verifying visual output reads real state
```go
// renderBuildVisual (codex_visuals.go:1210) reads from real state:
// - state.ColonyDepth for dispatch planning
// - phase.ID, phase.Name, phase.Description from colony.Phase
// - len(state.Plan.Phases) for progress calculation
// - plannedBuildDispatches(phase, depth) for spawn plan
// - resolveVerificationDepth(phase, totalPhases, ...) for review depth
// All values come from the runtime, not hard-coded
```

### Authoritative caste registry
```go
// cmd/codex_visuals.go lines 28-113
// casteEmojiMap: 27 entries (queen, builder, watcher, scout, colonizer, surveyor,
//   architect, chaos, archaeologist, oracle, route_setter, ambassador, auditor,
//   chronicler, gatekeeper, porter, guardian, includer, keeper, measurer, probe,
//   tracker, weaver, dreamer, medic, fixer)
// casteColorMap: 26 entries (same minus one -- porter added later)
// casteLabelMap: 27 entries (same as emoji map)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-caste model routing | Agent frontmatter model slots | v1.11 | Model selection is now native to platform, not injected |
| Prime Worker as separate caste | Merged into Builder | v1.0+ | Prime is deprecated; Builder handles orchestration |
| Surveyor as single caste | 4 surveyor subtypes (nest, disciplines, pathogens, provisions) | v1.3 | Surveyors are dispatched as 4 distinct workers in colonize |
| Hard-coded wave count | Dynamic wave allocation from task dependencies | v1.1 | Waves are computed from task.depends_on at dispatch time |

**Deprecated/outdated:**
- **Prime Worker caste:** Marked as DEPRECATED in .aether/workers.md. Builder handles the Prime Worker role now. Prime Worker's model (glm-5) is no longer a distinct routing target.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "guardian", "includer", "chronicler", "sage", "dreamer" castes are never dispatched in any cmd/*.go file | Worker Economy | Some may be dispatched in paths not yet discovered |
| A2 | Review agents (gatekeeper, auditor, probe) persist findings via review-ledger-write CLI in their task prompts | Worker Economy | If the agent ignores the CLI instruction, findings are lost |
| A3 | The 5 orchestration commands (build, continue, seal, colonize, plan) are the complete set of worker-spawning commands | Wave Shapes | Other commands like swarm, oracle may also dispatch workers |

**Verification status:** A1 verified by grep across all cmd/*.go files. A2 verified by reading codex_continue.go:1029-1041 (task prompts include review-ledger-write instructions). A3 partially verified -- swarm and oracle also dispatch workers but are not part of the 5 core commands per D-06.

## Open Questions

1. **Should swarm and oracle dispatch shapes be documented alongside the 5 core commands?**
   - What we know: D-06 specifies "build, continue, seal, colonize, plan" only. But swarm and oracle also dispatch workers with caste assignments.
   - What's unclear: Whether the planner should include them as supplementary tables or exclude them entirely.
   - Recommendation: Include them as supplementary tables since WORK-01 requires documenting ALL spawned castes.

2. **Are there dispatch paths triggered by wrapper markdown that the Go runtime does not see?**
   - What we know: Wrappers can spawn agents directly via Task tool using dispatch manifests.
   - What's unclear: Whether any wrapper spawns castes not covered by Go runtime dispatch code.
   - Recommendation: Check wrapper markdown for additional caste references during implementation.

3. **Should the spawn coverage test check casteEmojiMap alignment with agent definition count?**
   - What we know: casteEmojiMap has 27 entries. Agent definitions: 27 Claude .md, 27 OpenCode .md, 27 Codex .toml.
   - What's unclear: Whether all 27 agent definitions map to the same 27 casteEmojiMap entries.
   - Recommendation: Yes, include this alignment check as a parity verification in the test.

## Worker Caste Dispatch Inventory

> This is the core audit data derived from source code analysis.

### Actively Dispatched Castes (found in cmd/ dispatch code)

| Caste | Dispatch Location | Purpose (from agent definition) | Durable Output | Downstream Consumer |
|-------|-------------------|--------------------------------|----------------|---------------------|
| builder | codex_build.go:741 | Implement code, TDD-first | Files created/modified, tests written (ClaimsSummary) | build-finalize collects claims; continue verifies |
| watcher | codex_build.go:755, codex_continue.go:1348+ | Validate implementation, run tests | Verification report, quality gate assessment | Continue advancement decision |
| chaos | codex_build.go:767+ | Adversarial testing, edge cases | Resilience findings to domain review ledger | Continue gate evaluation |
| probe | codex_build.go:749, codex_continue.go:1039 | Test coverage analysis | Test files written (via Write tool) | Continue coverage gate |
| gatekeeper | codex_continue.go:1029 | Supply chain security audit | Security findings to review ledger (security domain) | Continue gate blocking |
| auditor | codex_continue.go:1034 | Code quality and compliance audit | Quality findings to review ledger (quality domain) | Continue gate blocking |
| measurer | codex_build.go:761 | Performance profiling | Performance findings to review ledger (performance domain) | Continue gate evaluation |
| scout | codex_plan.go:745 | Research and information gathering | SCOUT.md report file | Route-setter consumes findings |
| route_setter | codex_plan.go:756 | Phase planning and task decomposition | ROUTE-SETTER.md, colony plan JSON | Build command consumes plan |
| surveyor-nest | codex_colonize.go:590 | Map architecture and chamber layout | BLUEPRINT.md, CHAMBERS.md | Colonize finalizer aggregates |
| surveyor-disciplines | codex_colonize.go:601 | Map disciplines and sentinel protocols | DISCIPLINES.md, SENTINEL-PROTOCOLS.md | Colonize finalizer aggregates |
| surveyor-pathogens | codex_colonize.go:612 | Identify pathogens and fragile boundaries | PATHOGENS.md | Colonize finalizer aggregates |
| surveyor-provisions | codex_colonize.go:579 | Map provisions and external trails | PROVISIONS.md, TRAILS.md | Colonize finalizer aggregates |
| oracle | codex_build.go:703, oracle_loop.go:1348 | Deep research | oracle-{phase}.md research file | Builder/Architect consume |
| architect | codex_build.go:704 | Architecture design | architect-{phase}.md design file | Builder consumes |
| archaeologist | codex_build.go:698, swarm_cmd.go:846 | Git history analysis | Findings to review ledger (history domain) | Continue gate evaluation |
| ambassador | codex_build.go:709 | External integration design | Integration design findings | Builder consumes |
| tracker | swarm_cmd.go:828 | Bug investigation and root cause | Findings to bugs domain review ledger | Swarm summary, Builder for fix |

### Defined But Never Dispatched in cmd/ (potential findings)

| Caste | In casteEmojiMap | Agent Definition | Notes |
|-------|-----------------|-----------------|-------|
| queen | Yes (27 entries) | aether-queen.md | Used for display/routing, not dispatched as a worker |
| chronicler | Yes | aether-chronicler.md | Documentation specialist, has Write tool but never dispatched |
| sage | Yes | aether-sage.md | Analysis specialist, referenced in council.go but not dispatched |
| keeper | Yes | aether-keeper.md | Knowledge preservation, has Write tool but never dispatched |
| weaver | Yes | aether-weaver.md | Refactoring specialist, has Write/Edit/Bash but never dispatched |
| includer | Yes | aether-includer.md | Accessibility specialist, never dispatched |
| guardian | Yes | aether-guardian.md | Safety specialist, never dispatched |
| dreamer | Yes | No dedicated .md file | Only in caste maps, no agent definition found |
| medic | Yes | aether-medic.md | Colony health, never dispatched as a worker (runs as CLI self-check) |
| fixer | Yes | aether-fixer.md | Autonomous repair, never dispatched in production code (only in e2e test) |
| porter | Yes | aether-porter.md | Post-seal delivery, dispatched via seal workflow not via caste dispatch |

**Note:** "Never dispatched" means no `Caste: "castename"` struct literal in cmd/*.go dispatch planning code. Some castes may be invoked through other paths (wrapper markdown, council, swarm).

### Wave Shape Tables

#### Build Wave Shape (`cmd/codex_build.go`)

| Stage | Wave | Caste | Condition | Output |
|-------|------|-------|-----------|--------|
| prep | 1 | archaeologist | `depth == "deep" or "full"` | Git history findings to review ledger |
| research | 2 | oracle | `depth == "deep" or "full"` and no selected tasks | oracle-{phase}.md research file |
| design | 3 | architect | `depth == "deep" or "full"` and no selected tasks | architect-{phase}.md design file |
| integration | 4 | ambassador | `phaseNeedsAmbassador()` | Integration design |
| wave | 1-N | builder (or scout per task keywords) | Always (at least 1) | Files, tests, claims |
| wave | N+1 | builder | Default fallback if no tasks | Phase objective |
| probe | lastTaskWave+1 | probe | Always (if no selected tasks) | Test files |
| verification | lastTaskWave+2 | watcher | Always | Verification report |
| measurement | lastTaskWave+3 | measurer | `depth == "deep" or "full"` and `reviewDepth == heavy` | Performance findings |
| resilience | lastTaskWave+4 | chaos | `depth == "full"` and `reviewDepth == heavy`, or light-mode deterministic sampling | Resilience findings |

#### Continue Wave Shape (`cmd/codex_continue.go`)

| Stage | Caste | Condition | Output |
|-------|-------|-----------|--------|
| verification | watcher | Always (runs watcher verification) | Verification report |
| review | gatekeeper | reviewDepth >= standard | Security findings to review ledger |
| review | auditor | reviewDepth >= standard | Quality findings to review ledger |
| review | probe | reviewDepth >= standard | Coverage findings |

#### Plan Wave Shape (`cmd/codex_plan.go`)

| Stage | Wave | Caste | Output |
|-------|------|-------|--------|
| scouting | 1 | scout | SCOUT.md |
| routing | 2 | route_setter | ROUTE-SETTER.md, colony plan |

#### Colonize Wave Shape (`cmd/codex_colonize.go`)

| Stage | Wave | Caste | Output |
|-------|------|-------|--------|
| survey | 1 | surveyor-provisions | PROVISIONS.md, TRAILS.md |
| survey | 1 | surveyor-nest | BLUEPRINT.md, CHAMBERS.md |
| survey | 1 | surveyor-disciplines | DISCIPLINES.md, SENTINEL-PROTOCOLS.md |
| survey | 1 | surveyor-pathogens | PATHOGENS.md |

Note: All 4 surveyors dispatch in wave 1 (parallel).

#### Seal Wave Shape (`cmd/seal_final_review.go`)

| Stage | Caste | Condition | Output |
|-------|-------|-----------|--------|
| final-review | gatekeeper | Always (sealFinalReviewRequiredCastes) | Security findings to review ledger |
| final-review | auditor | Always | Quality findings to review ledger |
| final-review | probe | Always | Coverage findings |

Note: `sealFinalReviewRequiredCastes = []string{"gatekeeper", "auditor", "probe"}`.

### Visual Element State Traceability

| Visual Element | Rendering Function | State Source | Traces to Runtime? |
|---------------|-------------------|--------------|-------------------|
| Caste identity (emoji + color + label) | `casteIdentity()` via `casteEmoji()`, `casteANSIColor()`, `casteLabel()` | `casteEmojiMap`, `casteColorMap`, `casteLabelMap` (27 entries) | Yes -- static maps, deterministic per caste |
| Stage markers (e.g., "Context", "Tasks") | `renderStageMarker()` | Hard-coded stage names matching playbook sections | Yes -- markers correspond to real build/continue phases |
| Progress bar | `renderProgressSummary()` via `generateProgressBar()` | `phase.ID` and `len(state.Plan.Phases)` | Yes -- reads actual colony state |
| Build banner | `renderBanner()` | Phase ID and name from `colony.Phase` | Yes -- reads actual phase state |
| Aether wordmark | `renderAetherWordmark()` | Hard-coded ASCII art | No -- pure decoration (acceptable per D-05) |
| Spawn plan | `renderSpawnPlanForDispatches()` | `plannedBuildDispatches()` return values | Yes -- reads actual dispatch plan |
| Closeout banner | `renderCloseoutVisual()` | Colony state fields (goal, phases, workers) | Yes -- reads actual completion data |
| Signal visual | `renderSignalVisual()` | Pheromone write result (replaced flag) | Yes -- reflects actual signal state |
| Continue worker flow | `renderContinueWorkerFlowLine()` | Worker flow step data from continue report | Yes -- reads actual worker outcomes |
| Seal summary | `renderSealVisual()` | Colony state, summary path | Yes -- reads actual sealed state |

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this phase uses Go stdlib, existing project packages, and grep-based static analysis only)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None |
| Quick run command | `go test ./cmd/ -run TestWorkerEconomy -count=1` |
| Full suite command | `go test ./cmd/ -count=1 -race` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WORK-01 | Every dispatched caste has documented purpose | unit | `go test ./cmd/ -run TestDispatchedCastesDocumented -count=1` | Wave 0 |
| WORK-02 | No dispatched caste only reads and returns chat | unit | `go test ./cmd/ -run TestNoChatOnlyWorkers -count=1` | Wave 0 |
| WORK-03 | Wave shapes for 5 commands are documented | manual | N/A (documentation) | N/A |
| VIZ-01 | Visual functions read real state | unit | `go test ./cmd/ -run TestVisualOutputTracesToState -count=1` | Wave 0 |
| VIZ-02 | No decorative output claims state transition | unit | `go test ./cmd/ -run TestNoMisleadingDecoration -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestWorkerEconomy -count=1`
- **Per wave merge:** `go test ./cmd/ -count=1`
- **Phase gate:** Full suite green before closing

### Wave 0 Gaps
- [ ] `cmd/worker_economy_test.go` -- covers WORK-01, WORK-02, VIZ-01, VIZ-02
- [ ] No framework install needed -- Go testing already in use

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | -- |
| V3 Session Management | No | -- |
| V4 Access Control | No | -- |
| V5 Input Validation | No | -- |
| V6 Cryptography | No | -- |

This phase is a read-only audit. No user input processing, no authentication, no cryptography.

### Known Threat Patterns for Worker Economy Audit

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| None applicable | -- | -- |

## Sources

### Primary (HIGH confidence)
- cmd/codex_visuals.go -- caste registry maps, visual rendering functions [VERIFIED: source code read]
- cmd/codex_build.go -- build dispatch planning, wave shapes [VERIFIED: source code read]
- cmd/codex_continue.go -- continue review specs, watcher dispatch [VERIFIED: source code read]
- cmd/codex_plan.go -- plan dispatch planning [VERIFIED: source code read]
- cmd/codex_colonize.go -- colonize surveyor dispatch [VERIFIED: source code read]
- cmd/seal_final_review.go -- seal review dispatch [VERIFIED: source code read]
- pkg/codex/dispatch.go -- WorkerDispatch and DispatchResult types [VERIFIED: source code read]
- .claude/agents/ant/*.md -- 27 agent definitions with purpose statements [VERIFIED: file listing + sample reads]

### Secondary (MEDIUM confidence)
- .aether/workers.md -- worker definitions and spawn protocol [CITED: source code]
- .aether/docs/wrapper-runtime-ux-contract.md -- UX contract between runtime and wrappers [CITED: source code]
- .aether/docs/command-playbooks/build-wave.md -- build wave execution playbook [CITED: source code]

### Tertiary (LOW confidence)
- cmd/ceremony_emitter.go -- ceremony lifecycle events [ASSUMED: pattern matches codex_visuals.go approach]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all packages are local to this project, verified by reading source
- Architecture: HIGH -- dispatch code follows consistent patterns, visually verified
- Pitfalls: HIGH -- derived from actual codebase patterns discovered during research
- Wave shapes: HIGH -- extracted directly from dispatch planning functions
- Caste inventory: HIGH -- grep-verified across all cmd/*.go files
- Visual traceability: HIGH -- each render function traced to its state source

**Research date:** 2026-05-07
**Valid until:** 2026-06-07 (stable codebase, no fast-moving dependencies)
