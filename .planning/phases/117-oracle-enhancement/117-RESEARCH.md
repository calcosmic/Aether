# Phase 117: Oracle Enhancement - Research

**Researched:** 2026-05-13
**Domain:** Oracle deep-research loop (Go runtime) + TS host orchestration integration
**Confidence:** HIGH

## Summary

Phase 117 enriches the Oracle RALF (Research-Analyze-Learn-Formulate) loop with three capabilities: phase-aware prompt directives, diminishing-returns detection via novelty-delta tracking, and template-specific synthesis sections. The Oracle is implemented entirely in the Go runtime (`cmd/oracle_loop.go`, ~3,200 lines). The TS host currently has no Oracle-specific integration beyond caste config and workflow-pattern recognition. All three requirements (ORA-01, ORA-02, ORA-03) can be satisfied by extending the Go Oracle controller and its prompt-generation logic, with optional TS-host-side ceremony event emission for loop state transitions.

**Primary recommendation:** Implement all three enhancements in `cmd/oracle_loop.go` and `.aether/utils/oracle/oracle.md`. Add ceremony event emission for Oracle phase transitions so the TS host narrator can render them. Do NOT build a parallel Oracle engine in the TS host.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Oracle RALF loop execution | Go runtime (API / Backend) | — | Go owns state, iteration control, worker dispatch |
| Phase-aware prompt generation | Go runtime | — | Prompts are built in `buildOracleWorkerConfig` |
| Diminishing-returns detection | Go runtime | — | Delta tracking uses `oraclePlanFile` / `oracleStateFile` |
| Template-specific synthesis | Go runtime | — | Synthesis reports written by `writeOracleSynthesisReport` |
| Oracle loop ceremony events | Go runtime (emits) | TS host (renders) | Go emits; TS narrator already subscribes to ceremony events |
| Queen workflow pattern derivation | TS host | — | Already recognizes "Deep Research" when oracle+scout present |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.24 | Oracle loop, state files, worker dispatch | Already the runtime language |
| `pkg/codex` | repo-local | Worker invoker abstraction (Claude/OpenCode/Codex) | Existing dispatch layer |
| `pkg/events` | repo-local | Ceremony event bus | Already used by build/continue |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `js-yaml` | ^4 | TS host template/config loading | Already in ts-host dependencies |
| `chalk` | ^5 | TS host narrator rendering | Already in ts-host dependencies |
| `ora` | ^8 | Dashboard spinners | Already in ts-host dependencies |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending Go Oracle | Building Oracle in TS host | Violates boundary contract (TS host must not own state); Go already has iteration control, state files, and worker dispatch |
| Novelty delta in Go | Novelty delta in TS host | TS host never reads `.aether/oracle/` state directly; would require new Go-to-TS sync protocol |

**Installation:** No new dependencies required.

## Architecture Patterns

### System Architecture Diagram

```
User invokes /ant-oracle "topic"
        |
        v
+-----------------------------------+
|  Go CLI: aether oracle            |
|  (cmd/oracle_loop.go)             |
|  - resolve depth/scope/template   |
|  - formulate brief                |
|  - build questions                |
|  - write state.json + plan.json   |
+-----------------------------------+
        |
        v
+-----------------------------------+
|  Oracle Controller Loop           |
|  (runOracleLoop)                  |
|  - nextOraclePhase()              |
|  - selectOracleQuestionSmart()    |
|  - buildOracleWorkerConfig()      |
|  - runOracleIterationAttempt()    |
|  - applyOracleWorkerResponse()    |
|  - oracleProgressedSince()        |
|  - oracleReadyForCompletion()     |
+-----------------------------------+
        |
        v
+-----------------------------------+
|  Worker Dispatch (pkg/codex)      |
|  - Invokes aether-oracle agent    |
|  - Platform: Claude/OpenCode/Codex|
+-----------------------------------+
        |
        v
+-----------------------------------+
|  Oracle Agent                     |
|  (.claude/agents/ant/aether-oracle.md)
|  - Receives phase-aware prompt    |
|  - Writes iteration-N.json        |
|  - Returns structured JSON        |
+-----------------------------------+
        |
        v
+-----------------------------------+
|  Go Controller merges response    |
|  - Updates plan.json              |
|  - Writes synthesis.md / gaps.md  |
|  - Emits ceremony.loop.break      |
|    (on phase transition / stop)   |
+-----------------------------------+
        |
        v
+-----------------------------------+
|  TS Host Event Bridge             |
|  (optional)                       |
|  - Reads ceremony events          |
|  - Narrator renders Oracle banners|
+-----------------------------------+
```

### Recommended Project Structure (no new files needed)

All changes are localized to existing files:

```
cmd/oracle_loop.go                    ← Primary implementation target
.aether/utils/oracle/oracle.md        ← Update agent instructions for phase-aware behavior
pkg/events/ceremony.go                ← Already has ceremony.loop.break (verify if enough)
cmd/ceremony_emitter.go               ← Add emitOraclePhaseTransition if missing
.aether/ts-host/src/narrator.ts       ← Add handler for oracle ceremony topics
```

### Pattern 1: Phase-Aware Prompt Directives
**What:** The Oracle controller injects a directive string into the worker brief based on the current loop phase (`survey`, `investigate`, `synthesize`, `verify`).
**When to use:** Every Oracle iteration; the directive shapes the worker's research strategy.
**Example:**
```go
// Source: cmd/oracle_loop.go:1523
func oraclePhaseDirective(state oracleStateFile, plan oraclePlanFile) string {
    switch strings.ToLower(strings.TrimSpace(state.Phase)) {
    case "survey":
        return "Survey pass: prioritize untouched questions first and establish the evidence map."
    case "verify":
        return "Verify pass: resolve contradictions, tighten confidence scores, and sharpen the release recommendation."
    default:
        return "Investigate pass: deepen the lowest-confidence unresolved question with new source-backed findings."
    }
}
```

**Current gap:** The existing `oraclePhaseDirective` only handles `survey`, `verify`, and a generic default. It does NOT have a `synthesize` phase. The `nextOraclePhase` function transitions: `survey` -> `investigate` -> `verify`. There is no explicit `synthesize` phase in the Go code.

### Pattern 2: Diminishing Returns Detection
**What:** After each iteration, compare the new worker response against prior findings for the same question. If the new findings overlap significantly (low novelty delta), flag diminishing returns and force phase advancement.
**When to use:** In `runOracleLoop`, after `applyOracleWorkerResponse`, before deciding whether to continue.
**How it should work:**
1. Extract keywords from new findings.
2. Compare against keywords from prior findings for the same question.
3. Compute a novelty score: percentage of new keywords not seen before.
4. If novelty score < threshold (e.g., 20%) for N consecutive iterations, advance phase or stop.

**Current gap:** The Go code has `oracleProgressedSince` which checks coarse metrics (answered count, touched count, findings count, confidence). It does NOT do semantic novelty comparison.

### Pattern 3: Template-Specific Synthesis
**What:** The final `synthesis.md` report is structured according to the selected template (`tech-eval`, `architecture-review`, `bug-investigation`).
**When to use:** During the verify/synthesize phase, or when the loop completes.
**Example templates (already exist in `.aether/references/templates/`):**
- `oracle-tech-evaluation.md` — criteria matrix, tradeoff table, adopt/trial/defer/reject verdict
- `architecture-review-template.md` — current state, proposed shape, ownership boundaries, validation plan
- `bug-investigation-template.md` — symptom, reproduction, hypotheses table, root cause, fix boundary

**Current gap:** `writeOracleSynthesisReport` writes a generic markdown report. It does not use template-specific sections.

### Anti-Patterns to Avoid
- **Building a second Oracle in TS host:** The TS host must not duplicate Go runtime logic. It may render events, but it must not control the loop.
- **Storing Oracle state in TS host memory:** All state lives in `.aether/oracle/state.json` and `plan.json`.
- **Letting the worker rewrite state files:** The Go controller owns state; the worker writes only the response JSON file.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Worker dispatch | Custom subprocess spawning | `pkg/codex.WorkerInvoker` | Already handles platform detection, agent validation, timeout, heartbeat |
| State file I/O | Raw `ioutil.ReadFile` | `loadOracleStateFile` / `writeOracleStateFile` | Handles defaults, versioning, normalization |
| JSON response extraction | Regex on worker stdout | `parseOracleWorkerResponseFromText` | Brace-depth parser handles nested JSON, escaped strings |
| Ceremony event emission | Direct file writes | `emitLifecycleCeremony` / `emitLoopBreakEvent` | Already integrated with event bus and narrator |
| Template loading | Custom markdown parser | `template-loader.ts` (for TS host) or frontmatter parsing in Go | YAML frontmatter + body split already solved |

## Common Pitfalls

### Pitfall 1: Worker Rewrites Controller State
**What goes wrong:** If the worker prompt accidentally tells the worker to update `state.json` or `plan.json`, the controller and worker will race on file writes, corrupting the Oracle workspace.
**Why it happens:** The agent definition (`aether-oracle.md`) historically mentioned updating state files. The Go prompt now explicitly forbids this, but agent definitions in other platforms may still contain old language.
**How to avoid:** The `buildOracleWorkerConfig` constraints already include: "Do not read or rewrite .aether/oracle/state.json, plan.json, gaps.md, synthesis.md, or research-plan.md." Verify this constraint appears in all platform agent definitions.
**Warning signs:** `oracleProgressedSince` returns true but the iteration artifact shows no new findings — the worker may have written state directly instead of the response file.

### Pitfall 2: Diminishing Returns Threshold Too Aggressive
**What goes wrong:** If the novelty threshold is too high, the Oracle stops after 2-3 iterations, leaving questions only partially answered.
**Why it happens:** Keyword overlap is a crude proxy for semantic novelty. A worker may use different words to express the same concept, or it may find genuinely new evidence that happens to share keywords.
**How to avoid:** Use a conservative threshold (e.g., <15% novelty for 3 consecutive iterations on the same question). Also require that overall confidence is above a minimum (e.g., 50%) before allowing diminishing-returns stop.
**Warning signs:** Oracle consistently stops at "no_progress" or "max_iterations_reached" with low overall confidence.

### Pitfall 3: Template Mismatch in Synthesis
**What goes wrong:** The user requests `--template tech-eval`, but the synthesis report uses the generic structure, missing the criteria matrix and verdict.
**Why it happens:** `writeOracleSynthesisReport` does not branch on `state.Template`.
**How to avoid:** Add a template-specific branch in `writeOracleSynthesisReport` that loads the matching reference template and uses its section headers.
**Warning signs:** Synthesis report missing expected sections (e.g., no "Tradeoff Matrix" for tech-eval).

## Code Examples

### Phase-Aware Directive Injection (existing pattern)
```go
// Source: cmd/oracle_loop.go:1423-1444
brief := codex.RenderTaskBrief(codex.TaskBriefData{
    // ...
    Hints: []string{
        fmt.Sprintf("Current phase: %s", state.Phase),
        fmt.Sprintf("Iteration: %d of %d", state.Iteration, state.MaxIterations),
        // ...
        oraclePhaseDirective(state, plan),
    },
})
```

### Diminishing Returns Detection (proposed)
```go
// Proposed addition near cmd/oracle_loop.go:1022
func oracleDiminishingReturns(plan oraclePlanFile, state oracleStateFile, lastResponse oracleWorkerResponse) (bool, string) {
    // Find the question we just worked on
    var target oracleQuestion
    for _, q := range plan.Questions {
        if q.ID == state.ActiveQuestionID {
            target = q
            break
        }
    }
    if target.ID == "" {
        return false, ""
    }

    // Build keyword set from prior findings
    priorKeywords := make(map[string]bool)
    for _, f := range target.KeyFindings {
        for _, kw := range extractKeywords(f.Text) {
            priorKeywords[kw] = true
        }
    }

    // Count new keywords in the latest response
    newKeywords := 0
    totalNewKeywords := 0
    for _, f := range lastResponse.Findings {
        for _, kw := range extractKeywords(f.Text) {
            totalNewKeywords++
            if !priorKeywords[kw] {
                newKeywords++
            }
        }
    }

    if totalNewKeywords == 0 {
        return true, "no new findings"
    }

    novelty := float64(newKeywords) / float64(totalNewKeywords)
    if novelty < 0.15 {
        return true, fmt.Sprintf("novelty %.0f%% below threshold", novelty*100)
    }
    return false, ""
}
```

### Template-Specific Synthesis (proposed)
```go
// Proposed addition in writeOracleSynthesisReport
func writeOracleSynthesisReport(path string, state oracleStateFile, plan oraclePlanFile) error {
    var b strings.Builder
    switch state.Template {
    case "tech-eval":
        b.WriteString(renderTechEvalSynthesis(state, plan))
    case "architecture-review":
        b.WriteString(renderArchitectureReviewSynthesis(state, plan))
    case "bug-investigation":
        b.WriteString(renderBugInvestigationSynthesis(state, plan))
    default:
        b.WriteString(renderGenericSynthesis(state, plan))
    }
    return os.WriteFile(path, []byte(b.String()), 0644)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bash/tmux-based Oracle loop | Go-controlled loop with worker invoker | v1.0.20+ | Reliable iteration control, heartbeat, timeout |
| Worker updates state files directly | Controller owns state; worker writes response JSON only | v1.0.30+ | Eliminates race conditions, enables background mode |
| Lowest-confidence question selection | Smart multi-factor scoring (gap/contradiction/cross/confidence) | v1.0.34 | More impactful question targeting |
| Generic synthesis report | Template-specific synthesis (Phase 117) | v1.17 | Actionable, structured output per research type |

**Deprecated/outdated:**
- `--legacy` tmux loop: still available but not recommended; background Go loop is the standard.
- Worker self-updating `plan.json`: forbidden by current constraints.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The TS host does not need to implement Oracle loop logic; it only renders ceremony events emitted by Go. | Architectural Responsibility Map | If the user expects TS host to drive Oracle, the boundary contract is violated. |
| A2 | The `synthesize` phase mentioned in agent definitions (`survey -> investigate -> synthesize -> verify`) can be collapsed into `verify` in the Go code, or `nextOraclePhase` can be extended to include it. | Phase-Aware Prompt Design | If user expects a distinct synthesize phase with unique behavior, the plan must add it explicitly. |
| A3 | Keyword overlap (3+ character words) is sufficient for novelty delta tracking. | Diminishing Returns Detection | If semantic similarity is required, a more sophisticated approach (embeddings) would be needed, which is out of scope. |
| A4 | The existing ceremony event topics (`ceremony.loop.break`) are sufficient for TS host visibility into Oracle state transitions. | TS Host Integration | If the user wants per-iteration Oracle progress in the dashboard, new ceremony topics (e.g., `ceremony.oracle.iteration`) may be needed. |

## Open Questions

1. **Should the TS host dashboard display Oracle loop progress?**
   - What we know: The dashboard currently handles `ceremony.build.*`, `ceremony.plan.*`, `ceremony.continue.*`, and `ceremony.loop.break`.
   - What's unclear: Whether per-iteration Oracle events (e.g., `ceremony.oracle.iteration`) should be added so the dashboard shows "Oracle: investigating q3 (investigate phase, 67% confidence)".
   - Recommendation: Add `ceremony.oracle.iteration` and `ceremony.oracle.phase_transition` events if the user wants live Oracle progress in the dashboard. Otherwise, `ceremony.loop.break` on completion is sufficient.

2. **Should the `synthesize` phase be explicit in `nextOraclePhase`?**
   - What we know: Agent definitions mention `survey -> investigate -> synthesize -> verify`, but Go code has only `survey -> investigate -> verify`.
   - What's unclear: Whether `synthesize` is a distinct phase with unique prompt directives and attempt policies, or just a sub-mode of `verify`.
   - Recommendation: Treat `synthesize` as a sub-phase of `verify` for now (triggered when overall confidence >= target but before final verification). If user wants explicit phase, extend `nextOraclePhase`.

3. **Should diminishing returns detection emit a `ceremony.loop.break` event?**
   - What we know: `emitLoopBreakEvent` exists and is used for watcher auto-skip, circuit breaker, cycle detection, and lifecycle recovery.
   - What's unclear: Whether Oracle stopping due to diminishing returns should emit `ceremony.loop.break` with `loop_type="oracle_diminishing_returns"`.
   - Recommendation: Yes — this gives the TS host visibility into why the Oracle stopped, and it aligns with existing loop break semantics.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go runtime | Oracle loop execution | Yes | 1.24 | — |
| `aether` binary | CLI commands, worker dispatch | Yes | v1.0.34 | `go run ./cmd/aether` |
| TS host dependencies | Narrator, dashboard | Yes | Node >=20 | — |
| Platform CLIs (Claude/OpenCode/Codex) | Real worker dispatch | Partial | — | Simulation mode |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- Real platform CLIs: TS host falls back to simulation mode with a warning.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (built-in) |
| Config file | None — standard `go test` |
| Quick run command | `go test ./cmd -run TestOracle -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ORA-01 | Phase-aware directive injected into worker brief | unit | `go test ./cmd -run TestOraclePhaseDirective -v` | No — must add |
| ORA-02 | Diminishing returns detection forces phase advancement | unit | `go test ./cmd -run TestOracleDiminishingReturns -v` | No — must add |
| ORA-03 | Template-specific synthesis report generated | unit | `go test ./cmd -run TestOracleTemplateSynthesis -v` | No — must add |

### Sampling Rate
- **Per task commit:** `go test ./cmd -run TestOracle -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/oracle_loop_test.go` — add tests for phase-aware directive (ORA-01)
- [ ] `cmd/oracle_loop_test.go` — add tests for diminishing returns detection (ORA-02)
- [ ] `cmd/oracle_loop_test.go` — add tests for template-specific synthesis (ORA-03)
- [ ] `cmd/ceremony_emitter_test.go` — verify `emitOraclePhaseTransition` if added

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | No | — |
| V5 Input Validation | Yes | Worker response JSON is validated by `normalizeOracleWorkerResponse`; confidence clamped; status whitelist enforced |
| V6 Cryptography | No | — |

### Known Threat Patterns for Oracle Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious worker response JSON | Tampering | `normalizeOracleWorkerResponse` validates schema, clamps confidence, whitelists status |
| Worker writing outside response file | Elevation of Privilege | Prompt constraints + file path isolation (responses go to `.aether/oracle/responses/`) |
| Infinite Oracle loop | Denial of Service | `MaxIterations` cap (default 15, max 50) + `defaultOracleTimeout` (12 min) per attempt |

## Sources

### Primary (HIGH confidence)
- `cmd/oracle_loop.go` — Full Oracle loop implementation (3,248 lines). Verified all functions: `nextOraclePhase`, `selectOracleQuestionSmart`, `buildOracleWorkerConfig`, `oraclePhaseDirective`, `defaultOracleAttemptPolicy`, `applyOracleWorkerResponse`, `writeOracleSynthesisReport`, `oracleProgressedSince`, `oracleReadyForCompletion`, `oracleOverallConfidence`.
- `.aether/utils/oracle/oracle.md` — Oracle agent instructions. Verified phase directive mention and response contract.
- `.aether/references/playbooks/oracle-ralf-playbook.md` — RALF loop shape and template mapping.
- `.aether/references/templates/oracle-tech-evaluation.md` — Tech-eval template structure.
- `.aether/references/templates/architecture-review-template.md` — Architecture review template structure.
- `.aether/references/templates/bug-investigation-template.md` — Bug investigation template structure.
- `.aether/references/field-guides/oracle-template-selection-field-guide.md` — Template selection rules and RALF stage mapping.
- `.aether/references/contracts/oracle-output-contract.md` — Required output sections.
- `pkg/events/ceremony.go` — Ceremony event topics. Verified `CeremonyTopicLoopBreak` exists.
- `cmd/ceremony_emitter.go` — Verified `emitLoopBreakEvent` function.
- `.aether/ts-host/src/narrator.ts` — Verified narrator handles `ceremony.build.*`, `ceremony.loop.break`, etc.
- `.aether/ts-host/src/types.ts` — Verified `CeremonyPayload` includes `loop_type`, `detection_signal`, `action_taken`.

### Secondary (MEDIUM confidence)
- `.claude/agents/ant/aether-oracle.md` and `.opencode/agents/aether-oracle.md` — Agent definitions mention phase transitions and diminishing returns, but these are aspirational descriptions of what the Stop hook "manages." The actual implementation is in Go.
- `.planning/REQUIREMENTS.md` — ORA-01, ORA-02, ORA-03 requirements.
- `.planning/ROADMAP.md` — Phase 117 success criteria.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing repo-local Go packages.
- Architecture: HIGH — read the full 3,248-line Oracle loop and all related TS host files.
- Pitfalls: HIGH — derived directly from existing code behavior and agent definition text.

**Research date:** 2026-05-13
**Valid until:** 2026-06-13 (Oracle loop is stable; changes only when requirements change)
