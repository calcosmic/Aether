# Phase 89: Gate Self-Healing & Smart Planning - Research

**Researched:** 2026-05-01
**Domain:** AI colony framework -- gate recovery automation, confidence-targeted research, init synthesis, platform fixes
**Confidence:** HIGH

## Summary

Phase 89 extends the Phase 88 recovery foundation with four capability groups: (1) a new Fixer caste agent that investigates and repairs gate failures, (2) `/ant-unblock` upgraded from info-only to dispatch-capable with circuit breaker integration and attempt caps, (3) Oracle confidence targeting with a `--confidence-target` flag and rubric output, and (4) init synthesis that scouts the repo and blocks colony launch until the user approves an approval-ready brief. Two platform fixes round out the scope: OpenCode agent hub template `name` field validation, and LLM provider `baseURL` separation from worker callback URLs.

The phase is maximally additive -- it extends existing `cmd/unblock_cmd.go`, `cmd/circuit_breaker.go`, `cmd/oracle_loop.go`, `cmd/init_ceremony.go`, and `cmd/status.go` without replacing any existing infrastructure. The Fixer caste is the only truly novel agent design; all other work follows established patterns proven across 2900+ tests.

**Primary recommendation:** Implement in four waves: Fixer agent + unblock dispatch (GATE-06/07/08, LOOP-02/03/04), then Oracle confidence targeting (CONF-01/02/03), then init synthesis (CONF-04/05), then platform fixes + status gate display (PLAT-01/02, GATE-09).

## User Constraints

### Locked Decisions

- **D-01:** Fixer is a configurable-autonomy agent with three modes: `full` (writes code autonomously), `propose` (investigates and proposes, waits for approval), `advise` (diagnostic report only). Default is `propose`.
- **D-02:** Fixer mode is settable per-phase via `/ant-unblock --fixer-mode <mode>` and per-fix when Fixer proposes changes. Config default stored in `.planning/config.json` under `workflow.fixer_default_mode`.
- **D-03:** Fixer is the 27th agent. New caste: `fixer`. Agent files in `.claude/agents/ant/aether-fixer.md`, `.opencode/agents/aether-fixer.md`, `.codex/agents/aether-fixer.toml`.
- **D-04:** `/ant-unblock` upgraded from Phase 88's info-only mode to dispatch-capable. Reads `gate-results-{phase}.json`, shows Gate Recovery Summary, offers: (1) dispatch Fixer, (2) manual fix instructions, (3) skip.
- **D-05:** Fixer dispatch blocked when circuit breaker has tripped for the current phase (LOOP-03). Clear error message.
- **D-06:** Attempt cap defaults to 1 per phase (most conservative). Configurable via `workflow.max_unblock_attempts`. After cap: human-intervention message.
- **D-07:** After Fixer resolves issues, addressed blockers are auto-resolved in `gate-results-{phase}.json` and `/ant-continue` re-evaluates. Only previously-failed gates are re-run.
- **D-08:** Oracle loop accepts `--confidence-target` flag (default 95). Does not finalize below target unless hard blocker or max-iteration cap.
- **D-09:** Oracle output includes: target score, final score, iteration count, rubric breakdown with evidence, gaps, original prompt, synthesized prompt, approval status. Structured JSON output.
- **D-10:** Init command scouts repo and synthesizes a structured markdown launch brief with sections: Goal, Scope, Risks, Tech Stack, Dependencies, Success Criteria.
- **D-11:** Colony launch is blocked until user approves, edits, or rejects the brief. On edit: re-open in editor. On reject: return to prompt.
- **D-12:** OpenCode agent hub template generates valid `name` field that survives `aether update` (PLAT-01).
- **D-13:** LLM provider `baseURL` separated from worker callback/messaging URL (PLAT-02). Missing callback fails before worker spawn with clear config error.
- **D-14:** `/ant-status` shows a Gate Status section when `gate-results-{current}.json` exists. Shows: phase, gate count, pass/fail/skip breakdown, last run timestamp, fixer attempts used.
- **D-15:** Gate failures use JSON + wrapper rendering -- Go runtime outputs structured `gateCheck` with `fix_hint` and `recovery_options`. (Carried from 88)
- **D-16:** Gate results persist in per-phase `gate-results-{N}.json` in `.aether/data/`. (Carried from 88)
- **D-17:** Always-run gates (Flags, Watcher Veto) skip the per-phase persistence check. (Carried from 88)
- **D-18:** Circuit breaker `gateRetryKey` helper in `cmd/circuit_breaker.go` -- Fixer dispatch uses this. (Carried from 88)

### Claude's Discretion

- Fixer agent naming convention and deterministic name generation (follows existing caste system in `cmd/codex_visuals.go`)
- Oracle rubric breakdown format and confidence scoring algorithm details
- Init scouting depth and which files/patterns to analyze
- Fixer verification scope (re-run all gates vs only failed ones after fix)
- Telemetry and cycle detection wiring for Fixer dispatch paths (LOOP-04)

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-06 | /ant-unblock reads gate-results.json, shows Gate Recovery Summary, offers to dispatch Fixer | Existing `unblock_cmd.go` reads gate results; extend with dispatch option and `--fixer-mode` flag |
| GATE-07 | After Fixer resolves issues, addressed blockers auto-resolved, /ant-continue re-runs | Existing `gateResultsWritePhase()` and `shouldSkipGate()` handle persistence and skip logic |
| GATE-08 | Fixer caste (27th agent) reads gate context, investigates, fixes, verifies, reports JSON | New agent definition following existing caste pattern; dispatch via Agent tool |
| GATE-09 | /ant-status shows Gate Status section when gate-results.json exists | Existing `status.go` `renderDashboard()` -- add gate section conditionally |
| LOOP-02 | /ant-unblock tracks unblock attempts per phase, refuses after configurable cap | New tracking in per-phase file or COLONY_STATE; read cap from `.planning/config.json` |
| LOOP-03 | Fixer dispatch blocked when circuit breaker tripped for current phase | Existing `circuitBreaker.Allow(key)` with `gateRetryKey(phase, gateName)` |
| LOOP-04 | All new gate/recovery paths wire through existing cycle detection and telemetry | Existing `emitLoopBreakEvent()` in `cmd/ceremony_emitter.go` |
| CONF-01 | Oracle loop accepts --confidence-target flag (default 95) | Existing `oracleStateFile.TargetConfidence` field and `defaultOracleTargetConfidence=85` |
| CONF-02 | Oracle does not finalize below target unless hard blocker or max iterations | Existing `finalizeOracleLoop()` controls completion status |
| CONF-03 | Oracle output includes target, final score, iteration count, rubric breakdown, evidence | Existing `oracleStateFile` struct has most fields; extend output rendering |
| CONF-04 | Init scouts repo and synthesizes approval-ready launch brief | Existing `init-research` already scans repo; extend with synthesis step |
| CONF-05 | Colony launch blocked until user approves, edits, or rejects brief | Existing `runInitCeremony()` has Proceed/Revise/Cancel flow -- extend |
| PLAT-01 | OpenCode agent hub template generates valid name field; aether update preserves it | `validateOpenCodeAgentFile()` already checks `fm.Name` (line 294); template generation is the gap |
| PLAT-02 | LLM provider baseURL separated from worker callback URL; missing callback fails clearly | No existing separation -- new config field + validation in worker spawn path |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Fixer agent dispatch | API / Backend (Go runtime) | Frontend Server (wrapper markdown) | Go runtime owns state mutations (gate-results, circuit breaker); wrappers present dispatch UI |
| Fixer agent execution | Browser / Client (subagent) | API / Backend (Go runtime) | Fixer runs as an LLM subagent spawned by the wrapper or runtime; runtime tracks dispatch |
| Unblock command | API / Backend (Go runtime) | Frontend Server (wrapper markdown) | Go runtime reads gate-results, checks circuit breaker, tracks attempts |
| Oracle confidence loop | API / Backend (Go runtime) | -- | Entirely Go runtime logic -- flag parsing, iteration control, finalization |
| Init synthesis | API / Backend (Go runtime) | Frontend Server (wrapper markdown) | Go runtime scouts and synthesizes; wrapper presents brief for approval |
| Platform fixes | API / Backend (Go runtime) | -- | Template generation and config validation are Go-only |
| Status gate display | API / Backend (Go runtime) | -- | Go runtime renders gate section in status dashboard |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | 1.23+ | All runtime logic | Already the project language; no new dependencies for Phase 89 |
| cobra | existing | CLI commands | Every Aether command uses cobra; new flags follow `mustGetString`/`cmd.Flags()` pattern |
| pkg/storage | existing | JSON persistence | `gateResultsWritePhase()` and `store.SaveJSON()` already proven |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/colony | existing | Colony state types | `ColonyState`, `Phase`, `GateResultEntry` types |
| pkg/events | existing | Event bus / telemetry | `emitLifecycleCeremony()`, `emitLoopBreakEvent()` for LOOP-04 |
| cmd/ceremony_emitter | existing | Loop break events | `emitLoopBreakEvent(loopType, detectionSignal, actionTaken, source)` |
| cmd/circuit_breaker | existing | Circuit breaker | `gateRetryKey()`, `Allow()`, `RecordFailure()` for LOOP-03 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Fixer as LLM subagent (via Agent tool) | Fixer as Go-coded automation | LLM subagent handles novel root-cause analysis; Go automation only handles known patterns. Decision: LLM subagent per D-01/D-03. |
| Attempt tracking in gate-results-N.json | Separate unblock-tracking-N.json | Keeping in same file reduces file sprawl; adding an `unblock_attempts` field is simpler. |
| Oracle confidence stored in oracle state only | Also persisted to colony state | Oracle state is per-research session; colony state persistence is for cross-session. Oracle state is sufficient per D-08/D-09. |

**Installation:** No new packages required. All dependencies are already in `go.mod`.

**Version verification:** No external packages to verify -- all are in-repo.

## Architecture Patterns

### System Architecture Diagram

```
User -> /ant-unblock --phase N --fixer-mode propose
    |
    v
[Go Runtime: unblock_cmd.go]
    |
    +-- Read gate-results-N.json
    +-- Check circuit breaker: circuitBreaker.Allow(gateRetryKey(N, gateName))
    |   |-- Tripped? -> Error: "Circuit breaker tripped -- manual intervention required"
    +-- Check attempt cap: unblock_attempts < max_unblock_attempts (default 1)
    |   |-- Exceeded? -> Error: "Max unblock attempts reached -- human intervention required"
    +-- Render Gate Recovery Summary (existing buildGateRecoverySummary)
    +-- Offer: (1) Dispatch Fixer, (2) Manual fix instructions, (3) Skip
    |
    +-- [Dispatch Fixer] --> emitLoopBreakEvent("fixer_dispatch", ...)
    |       |
    |       v
    |   [Wrapper: Agent tool spawn with aether-fixer]
    |       |
    |       v
    |   [Fixer Subagent]
    |       +-- Read gate failure context
    |       +-- Investigate root cause (mode-dependent)
    |       +-- full: Apply fix, verify all gates pass
    |       +-- propose: Investigate, propose fix, wait for approval
    |       +-- advise: Diagnostic report only
    |       +-- Update gate-results-N.json (auto-resolve addressed blockers)
    |       +-- Return structured JSON with fix_report
    |       |
    |       v
    |   [Go Runtime: unblock_cmd.go processes Fixer result]
    |       +-- Update unblock_attempts count
    |       +-- Record success/failure in circuit breaker
    |       +-- emitLoopBreakEvent("fixer_complete" | "fixer_failed", ...)
    |
    v
User -> /ant-continue
    |
    v
[Go Runtime: codex_continue.go runCodexContinueGates()]
    +-- Load gate-results-N.json
    +-- shouldSkipGate(): skip passed gates, re-run failed (including Fixer-resolved)
    +-- Only re-run gates that were previously failed
    +-- If all pass -> phase advance

=== Oracle Confidence Targeting ===

User -> /ant-oracle "topic" --confidence-target 95
    |
    v
[Go Runtime: oracle_loop.go runOracleLoop()]
    +-- state.TargetConfidence = 95 (from flag)
    +-- Iteration loop:
    |   +-- Select target question (selectOracleQuestionSmart)
    |   +-- Invoke oracle worker
    |   +-- Merge findings, recalculate confidence
    |   +-- Check: OverallConfidence >= TargetConfidence?
    |   |   |-- YES -> finalizeOracleLoop(status="complete")
    |   |   |-- NO + hard blocker -> finalizeOracleLoop(status="blocked")
    |   |   |-- NO + max iterations -> finalizeOracleLoop(status="max_iterations_reached")
    |   |   |-- NO -> continue iteration
    +-- Output: rubric breakdown with evidence, gaps, approval status

=== Init Synthesis ===

User -> /ant-init "Build feature X"
    |
    v
[Go Runtime: init_ceremony.go runInitCeremony()]
    +-- runCeremonyResearch() -> charter + pheromone suggestions
    +-- Synthesize launch brief (NEW):
    |   +-- Sections: Goal, Scope, Risks, Tech Stack, Dependencies, Success Criteria
    |   +-- From charter + research data
    +-- Display launch brief
    +-- Prompt: (1) Approve, (2) Edit, (3) Reject
    |   |-- Approve -> createCeremonyColony()
    |   |-- Edit -> re-open in editor, re-display, re-prompt
    |   |-- Reject -> return to goal prompt
    +-- Colony launch BLOCKED until approval

=== Platform Fixes ===

PLAT-01: OpenCode agent validation already requires `name` field (line 294).
         Gap: template generation code that creates agent files may not set `name`.
         Fix: ensure any agent file generation copies the `name` from frontmatter.

PLAT-02: Worker spawn path currently conflates LLM provider baseURL with callback URL.
         Fix: separate into two config fields; validate callback URL exists before spawn.
         Error: "Missing worker callback URL -- configure provider.callback_url before spawning workers"
```

### Recommended Project Structure

```
cmd/
  unblock_cmd.go           # EXTEND: add Fixer dispatch, attempt tracking, circuit breaker check
  unblock_cmd_test.go      # EXTEND: add tests for dispatch, cap enforcement, breaker blocking
  fixer_dispatch.go        # NEW: Fixer dispatch logic, attempt tracking, result processing
  fixer_dispatch_test.go   # NEW: tests for Fixer dispatch
  oracle_loop.go           # EXTEND: confidence-target flag, rubric output, non-finalization logic
  oracle_loop_test.go      # EXTEND: tests for confidence targeting
  init_ceremony.go         # EXTEND: launch brief synthesis, approve/edit/reject flow
  init_ceremony_test.go    # EXTEND: tests for brief display and approval flow
  status.go                # EXTEND: gate status section in dashboard
  status_test.go           # EXTEND: tests for gate status display
  codex_visuals.go         # EXTEND: add "fixer" caste to maps
  circuit_breaker.go       # no changes needed -- already has gateRetryKey + Allow/RecordFailure
  gate.go                  # no changes needed -- already has gateResultsWritePhase/ReadPhase

.aether/
  config_schema.json       # EXTEND: add workflow.fixer_default_mode, workflow.max_unblock_attempts

.claude/agents/ant/
  aether-fixer.md          # NEW: Fixer agent definition (Claude Code)
.opencode/agents/
  aether-fixer.md          # NEW: Fixer agent definition (OpenCode)
.codex/agents/
  aether-fixer.toml        # NEW: Fixer agent definition (Codex, TOML format)
.aether/agents-claude/
  aether-fixer.md          # NEW: Fixer agent mirror (packaging)
.aether/agents-codex/
  aether-fixer.toml        # NEW: Fixer agent mirror (packaging)
```

### Pattern 1: Fixer Dispatch via Circuit Breaker

**What:** Before dispatching the Fixer, check the circuit breaker for the target phase's gates. If any gate has tripped the breaker, refuse dispatch with a clear error.

**When to use:** Every Fixer dispatch path in `/ant-unblock`.

**Example:**
```go
// Source: [VERIFIED: cmd/circuit_breaker.go, cmd/codex_continue.go:2390-2404]
// Existing pattern from runCodexContinueGates:
for _, prior := range priorGateResults {
    if prior.Status == "failed" {
        key := gateRetryKey(phaseNum, prior.Name)
        if circuitBreaker != nil && !circuitBreaker.Allow(key) {
            // Block dispatch
            return fmt.Errorf("Circuit breaker tripped for Phase %d -- manual intervention required", phaseNum)
        }
    }
}
```

### Pattern 2: Attempt Tracking per Phase

**What:** Track unblock attempts per phase in a simple JSON file or by adding a field to the gate-results file. Check against `workflow.max_unblock_attempts` before allowing dispatch.

**When to use:** Every `/ant-unblock` invocation.

**Example:**
```go
// Config read pattern (from existing config.json handling):
config := loadPlanningConfig() // reads .planning/config.json
maxAttempts := config.Workflow.MaxUnblockAttempts // default 1 per D-06
currentAttempts := readUnblockAttempts(phaseNum)
if currentAttempts >= maxAttempts {
    return fmt.Errorf("Max unblock attempts (%d) reached for Phase %d. Human intervention required.", maxAttempts, phaseNum)
}
```

### Pattern 3: Oracle Confidence Targeting

**What:** The Oracle loop already has `TargetConfidence` in `oracleStateFile`. Change the default from 85 to 95 (D-08), add a `--confidence-target` CLI flag, and modify `finalizeOracleLoop` to refuse finalization below target unless a hard blocker is reported or max iterations reached.

**When to use:** Every Oracle invocation with `--confidence-target` flag.

**Example:**
```go
// Source: [VERIFIED: cmd/oracle_loop.go:574-589]
// Current completion logic:
if state.OverallConfidence >= state.TargetConfidence {
    // complete
} else if maxIterations {
    // max_iterations_reached
}
// NEW: Add rubric output to finalization:
// Include: target, final score, iteration count, rubric breakdown,
// evidence per question, gaps, original prompt, synthesized prompt, approval status
```

### Pattern 4: Agent Definition (Fixer Caste)

**What:** New agent following the existing caste system. Must add `fixer` to all three visual maps in `cmd/codex_visuals.go` and create agent definition files for all three platforms.

**When to use:** Fixer caste registration and agent file creation.

**Example:**
```go
// Source: [VERIFIED: cmd/codex_visuals.go:28-54, 56-82, 84-110]
// Add to casteEmojiMap:
"fixer": "\U0001F527", // wrench emoji

// Add to casteColorMap:
"fixer": "33", // yellow (same as builder -- repair is building)

// Add to casteLabelMap:
"fixer": "Fixer",
```

### Pattern 5: Init Synthesis Brief

**What:** Extend `runInitCeremony()` to synthesize a structured markdown brief from the charter and research data. Present it before the existing Proceed/Revise/Cancel prompt. Add an Edit option that opens the brief in the user's `$EDITOR`.

**When to use:** Colony initialization flow.

**Example:**
```go
// Source: [VERIFIED: cmd/init_ceremony.go:108-179]
// Existing flow: charter display -> pheromone auto-approve -> Proceed/Revise/Cancel
// NEW flow: charter display -> pheromone auto-approve -> LAUNCH BRIEF -> Approve/Edit/Reject
brief := synthesizeLaunchBrief(charter, researchData)
fmt.Fprint(os.Stderr, brief)
choice := promptNumberedChoice("What would you like to do?", []string{
    "Approve -- accept brief and create colony",
    "Edit -- modify the brief in your editor",
    "Reject -- return to goal prompt",
})
```

### Anti-Patterns to Avoid

- **Fixer modifying gate-results directly**: Fixer should return a structured fix report; the Go runtime updates gate-results. This prevents the Fixer from corrupting state.
- **Oracle confidence inflation without evidence**: The existing Oracle rubric already penalizes single-source claims (cap at 50%). Do not weaken this -- the confidence target must mean something.
- **Init brief bypass**: Never allow colony creation without displaying the brief. The current `runInitCeremony()` always prompts; extend, don't skip.
- **Circuit breaker bypass**: The Fixer dispatch MUST check the same circuit breaker that gate evaluation uses. A separate breaker would create inconsistency.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Circuit breaker for gate retries | Custom retry counter | `cmd/circuit_breaker.go` `CircuitBreaker` with `gateRetryKey()` | Already exists, tested, handles goroutine safety, and is wired into continue gates |
| Gate result persistence | Custom file I/O | `gateResultsWritePhase()` / `gateResultsReadPhase()` | Already tested, handles per-phase naming, JSON serialization |
| Agent spawn and dispatch | Custom agent invocation | Existing `Agent()` tool / `codex.WorkerInvoker` | Handles subagent lifecycle, context injection, result parsing |
| Colony state reads | Direct JSON parsing | `store.LoadJSON("COLONY_STATE.json", &state)` | Handles file locking, atomic writes, error wrapping |
| Event emission for telemetry | Custom logging | `emitLoopBreakEvent()` / `emitLifecycleCeremony()` | Already wired into event bus, JSONL persistence, ceremony display |
| Config loading | Custom config parser | Existing `.planning/config.json` pattern | Follows established config schema, supports defaults |
| Caste identity (name/emoji/color) | Hardcoded per-agent strings | `casteIdentity()` / `casteLabel()` / `casteEmoji()` | Deterministic hash-based naming, consistent visual rendering |

**Key insight:** Phase 89's entire scope is about wiring existing infrastructure together in new ways. The only genuinely new component is the Fixer agent definition (a markdown file), and even that follows the established caste pattern exactly.

## Common Pitfalls

### Pitfall 1: Fixer Creates Infinite Gate-Recovery Loops
**What goes wrong:** Fixer gets dispatched, fixes a gate, but the fix introduces a new failure. The user runs `/ant-unblock` again, Fixer is dispatched again, ad infinitum.
**Why it happens:** The circuit breaker tracks by `gateRetryKey(phase, gateName)`, but the Fixer may fix one gate and break another, so different keys are used each time. The attempt cap (D-06, default 1) prevents this -- but only if tracking is per-phase, not per-gate.
**How to avoid:** Track `unblock_attempts` per phase (not per gate). The default cap of 1 means the Fixer gets exactly one chance per phase. The circuit breaker provides a secondary safety net at the gate level.
**Warning signs:** User running `/ant-unblock` more than once for the same phase without manual intervention.

### Pitfall 2: Fixer Mode Escalation Without User Control
**What goes wrong:** Fixer in `propose` mode proposes a fix, user approves, but the Fixer then applies additional changes beyond the proposal.
**Why it happens:** The Fixer's agent prompt doesn't clearly scope what "approval" means.
**How to avoid:** In the Fixer agent definition, clearly state: "When in propose mode, you may ONLY apply changes that were explicitly described in your proposal. If you discover additional issues, report them but do NOT fix them."
**Warning signs:** Fixer returning more `files_modified` than proposed.

### Pitfall 3: Oracle Confidence Target Creates Unbounded Loops
**What goes wrong:** Oracle with `--confidence-target 99` never reaches the target, burning through max iterations (currently 8 default) without finalizing.
**Why it happens:** The Oracle's `selectOracleQuestionSmart` function may keep selecting different questions without converging.
**How to avoid:** The existing `max_iterations` cap (default 8, configurable via depth levels) prevents truly unbounded loops. Additionally, the `max_iterations_reached` stop reason already exists. Ensure the confidence check happens BEFORE the iteration cap check so the rubric output includes the gap analysis.
**Warning signs:** Oracle status showing `status: "max_iterations_reached"` with `overall_confidence` far below target.

### Pitfall 4: Init Brief Edit Loses Structured Format
**What goes wrong:** User edits the launch brief in their `$EDITOR` and removes required sections, then approves a malformed brief.
**Why it happens:** The edit step opens raw markdown; no validation on re-read.
**How to avoid:** After edit, re-parse the brief and verify required sections exist (Goal, Scope, Risks, Tech Stack, Dependencies, Success Criteria). If any are missing, warn but allow approval (the brief is advisory, not a contract).
**Warning signs:** Brief missing sections after edit.

### Pitfall 5: OpenCode Agent `name` Field Stripped During `aether update`
**What goes wrong:** Agent files have valid `name` in frontmatter, but the update/publish pipeline regenerates or copies files without preserving the field.
**Why it happens:** The validation function (`validateOpenCodeAgentFile`) checks `fm.Name` at line 294, but the generation/copy code path may not set it. The `name` field must be explicitly included in the frontmatter template.
**How to avoid:** Audit the publish/install pipeline for agent file generation. Ensure the `name` field is populated from the filename or frontmatter source. The validation test `TestValidateOpenCodeAgent` already tests for valid agents -- extend to test that the `name` field survives a round-trip through the publish/install pipeline.
**Warning signs:** `aether update` producing agents that fail `validateOpenCodeAgentFile`.

## Code Examples

### Reading Gate Results for Fixer Context

```go
// Source: [VERIFIED: cmd/unblock_cmd.go:40-45]
// Existing pattern for reading gate results:
results, err := gateResultsReadPhase(phaseNum)
if err != nil || len(results) == 0 {
    outputOK(fmt.Sprintf("No gate results found for phase %d.", phaseNum))
    return nil
}

// Filter to only failed gates for Fixer context:
var failedGates []GateCheckResult
for _, r := range results {
    if r.Status == "failed" {
        failedGates = append(failedGates, r)
    }
}
```

### Circuit Breaker Check Before Dispatch

```go
// Source: [VERIFIED: cmd/circuit_breaker.go:44-48, cmd/codex_continue.go:2390-2404]
// Check if any failed gate has tripped the breaker:
func isFixerDispatchBlocked(phaseNum int, results []GateCheckResult) bool {
    for _, r := range results {
        if r.Status == "failed" {
            key := gateRetryKey(phaseNum, r.Name)
            if circuitBreaker != nil && !circuitBreaker.Allow(key) {
                return true
            }
        }
    }
    return false
}
```

### Loop Break Event Emission (LOOP-04)

```go
// Source: [VERIFIED: cmd/ceremony_emitter.go:562-568]
// Wire telemetry for Fixer dispatch:
emitLoopBreakEvent("fixer_dispatch",
    fmt.Sprintf("Phase %d: %d failed gates, attempt %d/%d",
        phaseNum, len(failedGates), attemptNum, maxAttempts),
    fmt.Sprintf("Dispatching Fixer in %s mode", fixerMode),
    "aether-unblock")

// Wire telemetry for Fixer completion:
emitLoopBreakEvent("fixer_complete",
    fmt.Sprintf("Phase %d: Fixer resolved %d/%d gates", phaseNum, resolved, total),
    "Gates updated in gate-results",
    "aether-unblock")
```

### Adding Fixer Caste to Visual System

```go
// Source: [VERIFIED: cmd/codex_visuals.go:28-110]
// In casteEmojiMap (after "medic"):
"fixer": "\U0001F527", // wrench

// In casteColorMap (after "medic"):
"fixer": "33", // yellow -- same family as builder

// In casteLabelMap (after "medic"):
"fixer": "Fixer",
```

### OpenCode Agent Frontmatter Format (PLAT-01)

```yaml
# Source: [VERIFIED: cmd/platform_sync.go:212-306]
# Valid OpenCode agent frontmatter requires:
---
name: aether-fixer                    # REQUIRED (Rule 8, line 294)
description: "Gate recovery agent that investigates and fixes failed colony gates. Spawned by /ant-unblock."
mode: subagent                        # REQUIRED: primary, subagent, or all (Rule 5, line 254)
color: "#f1c40f"                      # REQUIRED: hex or theme color (Rule 7, line 286)
tools:                                # REQUIRED: map/object with true/false (Rule 6, line 263)
  write: true
  edit: true
  bash: true
  grep: true
  glob: true
  task: false
---
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Gate failures = STOP wall | Structured recovery with fix_hint + recovery_options | Phase 88 (GATE-01/02) | Users can recover without reading error logs |
| `/ant-unblock` = info only | `/ant-unblock` = dispatch-capable | Phase 89 (GATE-06) | Colony can self-heal gate failures |
| Oracle confidence = fixed 85 | Oracle confidence = user-settable target (default 95) | Phase 89 (CONF-01) | Research quality matches user's risk tolerance |
| Init = scan + approve charter | Init = scan + synthesize brief + approve/edit/reject | Phase 89 (CONF-04/05) | Better launch decisions from structured brief |
| No gate status in dashboard | Gate status section in /ant-status | Phase 89 (GATE-09) | Users see gate health at a glance |

**Deprecated/outdated:**
- None -- all existing patterns remain valid and are extended, not replaced.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `openCodeAgentFrontmatter.Name` field is correctly populated during `aether publish` for existing agents | PLAT-01 / Standard Stack | If publish pipeline strips the `name` field, PLAT-01 is about fixing generation, not just validation. Need to verify the publish code path. |
| A2 | `cmd/serve.go` is where LLM provider `baseURL` is configured for worker spawning | PLAT-02 / Architecture Patterns | If baseURL is configured elsewhere (e.g., in `pkg/codex/`), the fix location changes. |
| A3 | The wrapper (Claude Code) Agent tool can spawn the Fixer agent by name `aether-fixer` matching the filename in `.claude/agents/ant/` | Fixer Dispatch | If the spawn mechanism requires different name mapping, agent definition filename must match. |
| A4 | The existing `runInitCeremony()` flow in Go runtime is what the wrapper command calls (not a separate wrapper-only flow) | Init Synthesis | CLAUDE.md init.md says "Use the Go aether CLI as the source of truth" -- confirming Go runtime owns the ceremony. |
| A5 | `defaultOracleTargetConfidence` can be changed from 85 to 95 without breaking existing Oracle depth presets | Oracle Confidence | The depth presets in `oracleDepthLevels` set their own target confidence (e.g., "deep" = 95, "balanced" = 85). The default change only affects the bare `--confidence-target` flag default. |

## Open Questions

1. **Where does the publish pipeline generate/copy OpenCode agent files?**
   - What we know: `validateOpenCodeAgentFile` checks the `name` field. `platform_sync.go` syncs `.opencode/agents/` to hub. `install_cmd.go` copies from package to target.
   - What's unclear: Whether any code path *generates* agent files from templates (as opposed to copying them). If generation exists, the template must include `name`.
   - Recommendation: Grep for agent file generation patterns (string templates, `text/template`, file creation with frontmatter). If none found, PLAT-01 is simply ensuring all existing agent files have `name` in frontmatter (which they already do based on the test data).

2. **Where is the LLM provider `baseURL` configured for worker spawning?**
   - What we know: `cmd/serve.go` has a WebSocket/SSE server for agent streams. `pkg/codex/` has `WorkerInvoker` for spawning workers.
   - What's unclear: Whether `baseURL` is a single field or already separated from callback URLs.
   - Recommendation: Read `pkg/codex/` to find the worker invocation config. The fix adds a `callback_url` field separate from `base_url`.

3. **Should the Fixer re-run ALL gates or only the ones it was dispatched to fix?**
   - What we know: D-07 says "Only previously-failed gates are re-run; passed gates stay passed." This is about `/ant-continue` behavior, not Fixer behavior.
   - What's unclear: Whether the Fixer should verify its own fix by running the specific gate(s) it addressed, or whether verification is deferred to `/ant-continue`.
   - Recommendation: Fixer should verify its specific fix (run the relevant test command or check), but NOT run the full gate suite. Full gate re-evaluation happens in `/ant-continue` per D-07.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified -- all work is code/config changes within the existing Go project)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing`) |
| Config file | none -- tests use `t.TempDir()` and `setupBuildFlowTest()` |
| Quick run command | `go test ./cmd/ -run TestUnblock -count=1` |
| Full suite command | `go test ./cmd/ -count=1` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-06 | /ant-unblock shows recovery summary + Fixer dispatch option | unit | `go test ./cmd/ -run TestUnblock -count=1` | Yes (extend `unblock_cmd_test.go`) |
| GATE-07 | Fixer-resolved gates auto-resolved, continue re-runs | unit | `go test ./cmd/ -run TestFixerResolve -count=1` | No -- Wave 0 |
| GATE-08 | Fixer agent reads context, investigates, fixes, reports JSON | unit | `go test ./cmd/ -run TestFixerDispatch -count=1` | No -- Wave 0 |
| GATE-09 | /ant-status shows Gate Status section | unit | `go test ./cmd/ -run TestStatus.*Gate -count=1` | Yes (extend `status_test.go`) |
| LOOP-02 | Attempt cap enforced per phase | unit | `go test ./cmd/ -run TestUnblock.*Attempt -count=1` | No -- Wave 0 |
| LOOP-03 | Fixer dispatch blocked when circuit breaker tripped | unit | `go test ./cmd/ -run TestUnblock.*Circuit -count=1` | No -- Wave 0 |
| LOOP-04 | Fixer dispatch emits loop break events | unit | `go test ./cmd/ -run TestFixer.*Telemetry -count=1` | No -- Wave 0 |
| CONF-01 | Oracle accepts --confidence-target flag | unit | `go test ./cmd/ -run TestOracle.*Confidence -count=1` | Yes (extend oracle tests) |
| CONF-02 | Oracle does not finalize below target | unit | `go test ./cmd/ -run TestOracle.*Finalize -count=1` | No -- Wave 0 |
| CONF-03 | Oracle output includes rubric breakdown | unit | `go test ./cmd/ -run TestOracle.*Rubric -count=1` | No -- Wave 0 |
| CONF-04 | Init synthesizes launch brief | unit | `go test ./cmd/ -run TestInit.*Brief -count=1` | No -- Wave 0 |
| CONF-05 | Colony launch blocked until brief approved | unit | `go test ./cmd/ -run TestInit.*Approval -count=1` | No -- Wave 0 |
| PLAT-01 | OpenCode agent name field survives update | unit | `go test ./cmd/ -run TestOpenCode.*Name -count=1` | Yes (extend `opencode_agent_validate_test.go`) |
| PLAT-02 | Missing callback URL fails before spawn | unit | `go test ./cmd/ -run TestWorker.*Callback -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestUnblock -count=1 && go test ./cmd/ -run TestFixer -count=1 && go test ./cmd/ -run TestOracle -count=1`
- **Per wave merge:** `go test ./cmd/ -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `cmd/fixer_dispatch_test.go` -- covers GATE-07, GATE-08, LOOP-02, LOOP-03, LOOP-04
- [ ] `cmd/init_ceremony_test.go` -- covers CONF-04, CONF-05 (may already exist; check)
- [ ] `cmd/oracle_loop_test.go` -- CONF-02, CONF-03 tests (may already exist; check)
- [ ] Framework install: none needed -- Go stdlib only

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A -- no user auth in this phase |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes | Fixer agent prompt injection via gate result content -- sanitize fix_hint and detail before injection into Fixer context |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for Go CLI + Agent System

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Fixer agent prompt injection via malicious gate-results content | Tampering | Sanitize gate result content (fix_hint, detail) before injecting into Fixer prompt. Use existing `pkg/colony/sanitize.go` patterns. |
| Attempt cap bypass via direct file manipulation | Tampering | Attempt tracking uses atomic JSON writes via `store.UpdateJSONAtomically()`. File-level tampering is out of scope (same trust model as COLONY_STATE.json). |
| Oracle confidence gaming (inflating scores to reach target) | Spoofing | Confidence rubric already penalizes single-source claims (cap 50%). Target confidence requires multi-source evidence. Do not weaken this constraint. |
| Init brief injection via malicious repo content | Tampering | Brief synthesis reads from repo files (README, go.mod, etc.). Content is displayed to user for approval, not executed. Low risk. |

## Sources

### Primary (HIGH confidence)
- `cmd/unblock_cmd.go` -- Existing /ant-unblock implementation to extend [VERIFIED: codebase read]
- `cmd/circuit_breaker.go` -- Circuit breaker with gateRetryKey [VERIFIED: codebase read]
- `cmd/gate.go` -- Gate check/result structs, persistence, skip logic [VERIFIED: codebase read]
- `cmd/oracle_loop.go` -- Oracle loop with confidence tracking, state file, finalization [VERIFIED: codebase read]
- `cmd/init_ceremony.go` -- Init ceremony flow with research, charter, approval [VERIFIED: codebase read]
- `cmd/codex_visuals.go` -- Caste emoji/color/label maps for Fixer registration [VERIFIED: codebase read]
- `cmd/status.go` -- Status dashboard rendering [VERIFIED: codebase read]
- `cmd/platform_sync.go` -- OpenCode agent validation with `name` field check [VERIFIED: codebase read]
- `cmd/codex_continue.go:2386-2499` -- Continue gate evaluation with circuit breaker [VERIFIED: codebase read]
- `cmd/ceremony_emitter.go:562-568` -- emitLoopBreakEvent for telemetry [VERIFIED: codebase read]
- `.claude/agents/ant/aether-builder.md` -- Agent definition pattern for Fixer [VERIFIED: codebase read]
- `.opencode/agents/aether-builder.md` -- OpenCode agent format reference [VERIFIED: codebase read]
- `.codex/agents/aether-builder.toml` -- Codex agent TOML format reference [VERIFIED: codebase read]
- `89-CONTEXT.md` -- User decisions (D-01 through D-18) [VERIFIED: codebase read]
- `.planning/REQUIREMENTS.md` -- Requirement definitions and traceability [VERIFIED: codebase read]
- `.planning/research/SUMMARY.md` -- v1.13 research synthesis with architecture approach [VERIFIED: codebase read]
- `CLAUDE.md` -- Project architecture, agent roster, caste system, coding conventions [VERIFIED: codebase read]

### Secondary (MEDIUM confidence)
- `cmd/unblock_cmd_test.go` -- Existing test patterns for unblock command [VERIFIED: codebase read + tests pass]
- `cmd/opencode_agent_validate_test.go` -- OpenCode agent validation tests [VERIFIED: codebase read]

### Tertiary (LOW confidence)
- None -- all findings verified against codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies; all existing Go stdlib patterns
- Architecture: HIGH -- all integration points verified in codebase; existing code provides clear extension points
- Pitfalls: HIGH -- circuit breaker infinite loop risk identified in v1.13 research summary and mitigated by D-06 attempt cap

**Research date:** 2026-05-01
**Valid until:** 30 days (stable domain -- extends existing infrastructure)
