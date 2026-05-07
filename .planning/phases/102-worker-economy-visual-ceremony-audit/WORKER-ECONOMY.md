# Worker Economy and Visual Ceremony Audit

**Phase:** 102 (Worker Economy & Visual Ceremony Audit)
**Generated:** 2026-05-07
**Status:** For Phase 105 remediation

---

## 1. Severity Summary

| Severity | Count |
|----------|-------|
| Critical | 0     |
| Warning  | 2     |
| Info     | 9     |

---

## 2. Worker Caste Inventory

### Actively Dispatched

| Caste | Status | Dispatch Location | Purpose | Durable Output | Downstream Consumer |
|-------|--------|-------------------|---------|----------------|---------------------|
| builder | dispatched | codex_build.go:741, swarm_cmd.go:859 | Implement code, execute commands, manipulate files | Files created/modified, tests written (ClaimsSummary) | build-finalize collects claims; continue verifies |
| watcher | dispatched | codex_build.go:755, codex_continue.go:1348+, codex_continue.go:1027, swarm_cmd.go:871 | Validate implementation, run tests, quality gate | Verification report, review ledger entries | Continue advancement decision; seal final review gate |
| scout | dispatched | codex_plan.go:745, swarm_cmd.go:837 | Research and information gathering | SCOUT.md report file | Route-setter consumes findings |
| route_setter | dispatched | codex_plan.go:756 | Phase planning and task decomposition | ROUTE-SETTER.md, colony plan JSON | Build command consumes plan |
| probe | dispatched | codex_build.go:749, codex_continue.go:1039 | Test coverage analysis | Test files written, coverage findings | Continue coverage gate |
| gatekeeper | dispatched | codex_continue.go:1029 | Security and supply chain audit | Security findings to review ledger (security domain) | Continue gate blocking |
| auditor | dispatched | codex_continue.go:1034 | Code quality and compliance audit | Quality findings to review ledger (quality domain) | Continue gate blocking |
| measurer | dispatched | codex_build.go:761 | Performance profiling | Performance findings to review ledger (performance domain) | Continue gate evaluation |
| chaos | dispatched | codex_build.go:767 | Adversarial testing, edge cases | Resilience findings to review ledger (resilience domain) | Continue gate evaluation |
| archaeologist | dispatched | codex_build.go:698, swarm_cmd.go:846 | Git history analysis | History findings to review ledger (history domain) | Continue gate evaluation |
| oracle | dispatched | codex_build.go:703, oracle_loop.go:1348 | Deep research (RALF loop) | oracle-{phase}.md research file | Builder/Architect consume |
| architect | dispatched | codex_build.go:704 | Architecture design | architect-{phase}.md design file | Builder consumes |
| ambassador | dispatched | codex_build.go:709 | External integration design | Integration design findings | Builder consumes |
| tracker | dispatched | swarm_cmd.go:828 | Bug investigation and root cause | Findings to bugs domain review ledger | Swarm summary; Builder for fix |
| surveyor-provisions | dispatched | codex_colonize.go:579 | Map provisions and external trails | PROVISIONS.md, TRAILS.md | Colonize finalizer aggregates |
| surveyor-nest | dispatched | codex_colonize.go:590 | Map architecture and chamber layout | BLUEPRINT.md, CHAMBERS.md | Colonize finalizer aggregates |
| surveyor-disciplines | dispatched | codex_colonize.go:601 | Map disciplines and sentinel protocols | DISCIPLINES.md, SENTINEL-PROTOCOLS.md | Colonize finalizer aggregates |
| surveyor-pathogens | dispatched | codex_colonize.go:612 | Identify pathogens and fragile boundaries | PATHOGENS.md | Colonize finalizer aggregates |

### Defined But Never Dispatched

| Caste | Status | Dispatch Location | Purpose | Durable Output | Downstream Consumer |
|-------|--------|-------------------|---------|----------------|---------------------|
| queen | defined-only | never dispatched | Colony orchestrator, decision layer | Colony state mutations (owns orchestration, not dispatched as worker) | Not applicable -- queen dispatches workers, is not dispatched |
| surveyor | defined-only | never dispatched (subtypes dispatched instead) | Base surveyor caste; subtypes (surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens) are dispatched | chat-only (base caste never spawned) | none identified |
| colonizer | defined-only | never dispatched | Codebase exploration | chat-only | none identified |
| chronicler | defined-only | never dispatched | Documentation specialist | chat-only | none identified |
| keeper | defined-only | never dispatched | Knowledge preservation | chat-only | none identified |
| weaver | defined-only | never dispatched | Refactoring specialist | chat-only | none identified |
| includer | defined-only | never dispatched | Accessibility specialist | chat-only | none identified |
| guardian | defined-only | never dispatched | Safety specialist | chat-only | none identified |
| dreamer | defined-only | never dispatched | No dedicated agent .md file found | chat-only | none identified |
| medic | defined-only | never dispatched | Colony health, runs as CLI self-check | chat-only | none identified |
| fixer | defined-only | never dispatched | Autonomous repair, only in e2e tests | chat-only | none identified |
| porter | defined-only | dispatched via seal workflow, not standard caste dispatch | Post-seal delivery | Delivery readiness output | Seal closeout section |

---

## 3. Wave Shape Tables

### Build Wave Shape (`cmd/codex_build.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| prep | 1 | archaeologist | `depth == "full"` and no selected tasks | Git history findings to review ledger | Provides historical risk context for builders |
| research | 2 | oracle | `depth == "deep" or "full"` and no selected tasks | oracle-{phase}.md research file | Provides research context for builders |
| design | 3 | architect | `depth == "deep" or "full"` and no selected tasks | architect-{phase}.md design file | Provides design boundaries for builders |
| integration | 4 | ambassador | `phaseNeedsAmbassador()` returns true | Integration design findings | Provides integration constraints for builders |
| wave | 1-N | builder (or scout per task keywords) | Always (at least 1) | Files, tests, claims (ClaimsSummary) | Core implementation; continue verifies |
| wave | N+1 | builder | Default fallback if no tasks and no selected tasks | Phase objective implementation | Core implementation |
| probe | lastTaskWave+1 | probe | Always if no selected tasks | Test files, independent verification | Continue coverage gate |
| verification | lastTaskWave+2 | watcher | Always | Verification report | Continue advancement decision |
| measurement | lastTaskWave+3 | measurer | `depth == "deep" or "full"` and `reviewDepth == heavy` | Performance findings | Continue gate evaluation |
| resilience | lastTaskWave+4 | chaos | `depth == "full"` and `reviewDepth == heavy`, or light-mode deterministic sampling | Resilience findings | Continue gate evaluation |

### Continue Wave Shape (`cmd/codex_continue.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| verification | 1 | watcher | Always (runs watcher verification) | Verification report, worker flow step | Advancement decision; gate evaluation |
| review | 1 | gatekeeper | `reviewDepth >= standard` | Security findings to review ledger | Continue gate blocking |
| review | 1 | auditor | `reviewDepth >= standard` | Quality findings to review ledger | Continue gate blocking |
| review | 1 | probe | `reviewDepth >= standard` | Coverage findings | Continue gate evaluation |

Note: Continue review workers dispatch from `codexContinueReviewSpecs` (codex_continue.go:1027-1042). At `reviewDepth == light`, only probe runs (index 2). At `standard` and above, all three run.

### Plan Wave Shape (`cmd/codex_plan.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| scouting | 1 | scout | Always | SCOUT.md | Route-setter consumes findings |
| routing | 2 | route_setter | Always (after scout wave) | ROUTE-SETTER.md, colony plan JSON | Build command consumes plan |

Note: Plan workers dispatch sequentially. Route-setter waits for scout completion (codex_plan.go:840-866).

### Colonize Wave Shape (`cmd/codex_colonize.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| survey | 1 | surveyor-provisions | Always | PROVISIONS.md, TRAILS.md | Colonize finalizer aggregates |
| survey | 1 | surveyor-nest | Always (parallel) | BLUEPRINT.md, CHAMBERS.md | Colonize finalizer aggregates |
| survey | 1 | surveyor-disciplines | Always (parallel) | DISCIPLINES.md, SENTINEL-PROTOCOLS.md | Colonize finalizer aggregates |
| survey | 1 | surveyor-pathogens | Always (parallel) | PATHOGENS.md | Colonize finalizer aggregates |

Note: All 4 surveyors dispatch in wave 1 (parallel) via `plannedSurveyors()` (codex_colonize.go:574-621).

### Seal Wave Shape (`cmd/seal_final_review.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| seal-review | 1 | gatekeeper | Always (sealFinalReviewRequiredCastes) | Security findings to review ledger | Seal gate blocking |
| seal-review | 1 | auditor | Always | Quality findings to review ledger | Seal gate blocking |
| seal-review | 1 | probe | Always | Coverage findings | Seal gate evaluation |

Note: `sealFinalReviewRequiredCastes = []string{"gatekeeper", "auditor", "probe"}` (seal_final_review.go:24). These reuse the same `codexContinueReviewSpecs` dispatch pipeline from continue (seal_final_review.go:879).

### Supplementary: Swarm Wave Shape (`cmd/swarm_cmd.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| investigation | 1 | tracker | Always | Structured swarm response JSON | Builder fix wave consumes findings |
| investigation | 1 | scout | Always (parallel) | Structured swarm response JSON | Builder fix wave consumes findings |
| investigation | 1 | archaeologist | Always (parallel) | Structured swarm response JSON | Builder fix wave consumes findings |
| fix | 2 | builder | Always | Code changes, tests | Watcher verification wave |
| verification | 3 | watcher | Always | Structured swarm response JSON | Swarm summary and next command |

### Supplementary: Oracle Loop Wave Shape (`cmd/oracle_loop.go`)

| Stage | Wave | Caste | Condition | Output | Downstream Need |
|-------|------|-------|-----------|--------|-----------------|
| research | 1 | oracle | Always (oracle loop iterations) | oracle-{phase}.md research file | Builder consumes for deep research tasks |

---

## 4. Visual Ceremony Traceability

| Visual Element | Rendering Function | State Source | Traces to Runtime? | Finding |
|---------------|-------------------|--------------|-------------------|---------|
| Caste identity (emoji + ANSI color + label) | `casteIdentity()` via `casteEmoji()`, `casteANSIColor()`, `casteLabel()` | `casteEmojiMap`, `casteColorMap`, `casteLabelMap` (26 entries each) | Yes -- static maps, deterministic per caste | |
| Build banner | `renderBanner(commandEmoji("build"), ...)` | Phase ID and name from `colony.Phase` passed as argument | Yes -- reads actual phase state | |
| Progress bar | `renderProgressSummary(current, total)` | `phase.ID` and `len(state.Plan.Phases)` passed as arguments | Yes -- reads actual colony state | |
| Stage markers (e.g., "Context", "Tasks") | `renderStageMarker(title)` | Hard-coded stage names matching playbook sections | Yes -- markers correspond to real build/continue phases | |
| Spawn plan | `renderSpawnPlanForDispatches(dispatches, ...)` | `plannedBuildDispatchesForSelection()` return values passed as argument | Yes -- reads actual dispatch plan | |
| Aether wordmark | `renderAetherWordmark()` | Hard-coded ASCII art string | No -- pure decoration | decorative-only -- acceptable per D-05 |
| Closeout banner | `renderCloseoutVisual(result)` | Colony state fields (goal, phases, workers) from result map | Yes -- reads actual completion data | |
| Signal visual | `renderSignalVisual(sigType, content, priority, replaced)` | Signal parameters passed as arguments (from pheromone write result) | Yes -- reflects actual signal state | |
| Continue worker flow | `renderContinueWorkerFlowLine(b, name, caste, status, summary)` | Worker flow step data from continue report | Yes -- reads actual worker outcomes | |
| Seal summary | `renderSealVisual(state, summaryPath)` | Colony state and summary path passed as arguments | Yes -- reads actual sealed state | |

---

## 5. Findings

### I-01: CLAUDE.md claims 27 agents but runtime defines 26 castes

The CLAUDE.md "The 27 Agents" table lists 27 entries including Sage, but the authoritative runtime registry in `cmd/codex_visuals.go` defines only 26 entries across `casteEmojiMap`, `casteColorMap`, and `casteLabelMap`. The "sage" caste has a Claude agent definition (`.claude/agents/ant/aether-sage.md`) but no entry in any of the three runtime caste maps. Sage is referenced in `cmd/council.go` and in session command maps, but is not part of the visual caste identity system.

Severity: Info because sage exists in documentation and has non-visual runtime references, but the authoritative visual identity registry has 26 entries, not 27.

### I-02: Ten castes defined but never dispatched

The following castes appear in `casteEmojiMap` but have no `Caste: "name"` struct literal in any production cmd/*.go dispatch planning code: surveyor, colonizer, chronicler, keeper, weaver, includer, guardian, dreamer, medic, fixer. These castes have agent definitions but the runtime never spawns them as workers. Note: "surveyor" base caste is defined but only its subtypes (surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens) are dispatched.

Severity: Info because these castes may be used through wrapper markdown dispatch or reserved for future use. They are documented findings, not violations.

### W-01: Eight defined-only castes produce no durable output

The castes surveyor, colonizer, chronicler, keeper, weaver, includer, guardian, and dreamer are defined in the runtime caste registry and have agent definitions, but they are never dispatched by the Go runtime. If invoked through wrapper markdown, they have no enforced output contract (no review-ledger-write instructions in their dispatch paths). This is a WORK-02 concern: workers that only read and return chat without persisting.

Severity: Warning because if any wrapper or future dispatch path invokes these castes, they would produce no persisted artifacts, violating the principle that every spawned worker must produce durable output.

### W-02: Porter dispatch path is ambiguous

The porter caste has an entry in all three runtime maps and an agent definition, but it is not dispatched through standard caste dispatch code. Instead, porter runs through the seal workflow closeout section (referenced in `renderCloseoutVisual` as "Post-Seal: Delivery Readiness"). This means porter does not appear in the standard wave shape tables for build, continue, or seal review dispatch.

Severity: Warning because the porter dispatch path is not discoverable through the standard dispatch planning functions that the other castes use, making it easy to miss during auditing.

### I-03: colonizer caste has no dispatch site

The colonizer caste appears in all three runtime maps and has an agent definition, but has no `Caste: "colonizer"` struct literal in any production cmd/*.go file. The `/ant-colonize` command dispatches "surveyor-*" subtypes, not "colonizer" caste workers.

Severity: Info because the caste name suggests it should be dispatched during colonization, but the actual dispatch uses surveyor subtypes instead.

### I-04: dreamer caste has no agent definition file

The dreamer caste appears in all three runtime maps but has no dedicated agent definition file at `.claude/agents/ant/aether-dreamer.md` or equivalent. Other castes without dispatch sites (chronicler, keeper, etc.) at least have agent definitions describing their purpose.

Severity: Info because dreamer's purpose is undefined beyond the caste label "Dreamer" and emoji.

### I-05: suggestedBuildCaste can dispatch scout through build task waves

The `suggestedBuildCaste()` function in `codex_visuals.go:3255-3270` returns "scout" for tasks containing research-related keywords (research, investigat, survey, analy, document, readme, spec). This means scout can be dispatched during build task waves, not just during plan. This is a valid dispatch site not captured in the primary build wave shape documentation.

Severity: Info because this is intentional behavior (task-keyword-based caste selection) but it means scout has more dispatch paths than the plan-only wave shape suggests.

### I-06: Surveyor base caste defined but only subtypes dispatched

The "surveyor" base caste appears in all three runtime maps (`casteEmojiMap`, `casteColorMap`, `casteLabelMap`) but is never dispatched as a worker. Instead, four surveyor subtypes are dispatched during colonize: surveyor-provisions, surveyor-nest, surveyor-disciplines, and surveyor-pathogens. These subtypes use `Caste: "surveyor-*"` in dispatch code and have dedicated agent definition files.

Severity: Info because the base caste name exists for display/identity purposes while the subtypes handle actual dispatch. The colonize wave shape table correctly shows the four subtypes.

### I-07: casteColorMap has same count as casteEmojiMap (26 entries)

The research doc claimed casteColorMap had 26 entries while casteEmojiMap had 27, suggesting porter was added later. In reality, all three maps (casteEmojiMap, casteColorMap, casteLabelMap) have 26 entries with identical key sets. Porter is present in all three maps.

Severity: Info because this corrects the research assumption about a count mismatch between maps.

### I-08: medic caste runs as CLI self-check, not worker dispatch

The medic caste is defined in the runtime maps and has an agent definition, but it is never dispatched as a worker through the standard dispatch pipeline. Instead, medic functionality runs as a CLI self-check command (`aether patrol`). This is a different execution model from the worker dispatch pattern used by other castes.

Severity: Info because medic's health-check behavior is correctly implemented through CLI commands rather than worker dispatch.

### I-09: fixer caste only dispatched in e2e tests

The fixer caste has a runtime map entry and agent definition but is only dispatched in end-to-end test code, never in production dispatch paths. The auto-recovery orchestrator in v1.14 was designed to dispatch fixer workers, but the current production code does not include this dispatch site.

Severity: Info because fixer is documented as part of the auto-recovery system but is not actively dispatched in production code paths.

---

## 6. Verified Counts

| Category | Count | Notes |
|----------|-------|-------|
| Total defined castes | 26 | From casteEmojiMap (not 27 as documented in CLAUDE.md; see I-01) |
| Actively dispatched (production) | 18 | Found via Caste: string literals in cmd/*.go non-test files |
| Defined but never dispatched | 9 | surveyor, colonizer, chronicler, keeper, weaver, includer, guardian, dreamer, medic |
| Porter (special dispatch) | 1 | Dispatched through seal closeout, not standard caste dispatch |
| Fixer (test-only dispatch) | 1 | Dispatched in e2e tests, not production paths |
| Visual elements traced | 10 | From codex_visuals.go render functions |
| Visual elements decorative | 1 | Aether ASCII wordmark only |
