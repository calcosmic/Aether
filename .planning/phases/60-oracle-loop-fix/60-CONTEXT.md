# Phase 60: Oracle Loop Fix - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

The Oracle deep-research agent gets a research brief formulation step, four user-selectable depth levels, a RALF loop with smart question formulation, and an OpenCode callback URL fix. Four requirements: ORCL-01 (callback fix), ORCL-02 (research brief), ORCL-03 (depth selection), ORCL-04 (RALF state management).

This phase modifies the Oracle loop in `cmd/oracle_loop.go`, the worker invoker in `pkg/codex/`, and the Oracle agent definition. It does NOT change build/continue flows or add new commands beyond Oracle.

</domain>

<decisions>
## Implementation Decisions

### Research Brief (ORCL-02)
- **D-01:** Full colony context in the research brief — project profile (type, languages, frameworks), pheromones (FOCUS/REDIRECT signals), recent learnings (from colony state), codebase structure (directory tree, key files), and colony goal. One comprehensive brief gathered before the loop starts.
- **D-02:** Brief drives question generation — replaces the current `buildOracleQuestions()` keyword-matching approach with brief-informed questions. The brief content shapes what the Oracle asks about, not just what context workers see.

### Depth Selection (ORCL-03)
- **D-03:** Four depth levels with specific iteration caps and confidence targets:
  - Quick: 2 max iterations, 60% confidence target
  - Balanced: 4 max iterations, 85% confidence target
  - Deep: 6 max iterations, 95% confidence target
  - Exhaustive: 10 max iterations, 99% confidence target
- **D-04:** Interactive prompt after brief formulation but before loop starts, PLUS CLI flags (`--depth quick|balanced|deep|exhaustive`) for automation. Defaults to Balanced if not specified.
- **D-05:** Depth selection sets both `MaxIterations` and `TargetConfidence` on the Oracle state — the loop still respects the existing completion logic (confidence reached = done).

### OpenCode Callback Fix (ORCL-01)
- **D-06:** Env var override — add a separate environment variable for the agent messaging endpoint, distinct from the LiteLLM proxy URL. The OpenCode dispatcher passes this env var to the worker subprocess so it uses the correct messaging endpoint instead of conflating it with the proxy.

### RALF Loop Formulation (ORCL-04)
- **D-07:** Smart formulation step replaces simple lowest-confidence selection. After each iteration, the controller analyzes accumulated knowledge (open gaps, contradictions, findings across all questions) to pick the most impactful next question — the one where new research would resolve the most gaps or resolve contradictions, not just the lowest confidence score.

### Claude's Discretion
- Exact env var name for the OpenCode callback override
- How to render the depth selection prompt (labels, descriptions)
- How brief-informed question generation works in detail (algorithm for generating questions from brief content)
- Exact formulation algorithm for smart question selection (scoring function for question impact)
- Whether the brief formulation step runs as a separate function or is integrated into the existing `startOracleCompatibility()` flow

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — ORCL-01, ORCL-02, ORCL-03, ORCL-04 requirements with acceptance criteria
- `.planning/ROADMAP.md` — Phase 60 success criteria (lines 174-183)

### Oracle Loop (primary code)
- `cmd/oracle_loop.go` — Full Oracle loop: state management, question selection, worker invocation, iteration tracking, progress monitoring (~1900 lines)
- `cmd/oracle_process_unix.go` — Process management for Oracle workers
- `.aether/utils/oracle/oracle.md` — Oracle agent prompt (worker instructions, response contract, confidence rubric)

### Worker Invocation (callback fix)
- `pkg/codex/worker.go` — RealInvoker, FakeInvoker, worker claims parsing, prompt assembly
- `pkg/codex/platform_dispatch.go` — OpenCodeDispatcher, ClaudeDispatcher, env var handling (`envOpenCodePath`, `envOpenCodePrimary`)
- `pkg/codex/dispatch.go` — WorkerDispatch, DispatchResult structs

### Context Assembly (brief formulation)
- `cmd/oracle_loop.go` — `detectOracleProjectProfile()`, `currentOracleFocusAreas()`, `renderOracleContextCapsule()`, `buildOracleQuestions()` (all will be modified)

### Related Patterns
- `.planning/phases/59-gate-failure-recovery/59-CONTEXT.md` — Gate results pattern (structured state in existing JSON files)
- `cmd/compatibility_cmds.go` — Oracle CLI subcommand routing (`runOracleCompatibility`)
- `cmd/codex_visuals.go` — Visual rendering for Oracle progress banners

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `detectOracleProjectProfile()` in oracle_loop.go — already detects type/languages/frameworks. Extend to also gather codebase structure, learnings, colony goal.
- `currentOracleFocusAreas()` — already reads FOCUS pheromones. Extend to also read REDIRECT signals and recent learnings.
- `renderOracleContextCapsule()` — already builds per-iteration context. The brief formulation is a pre-loop superset of this.
- `selectOracleQuestion()` — current question selection (lowest-confidence). Replace with smart formulation.
- `oracleStateFile` struct — already has MaxIterations, TargetConfidence, Strategy fields. Depth selection sets these.
- `defaultOracleMaxIterations = 8`, `defaultOracleTargetConfidence = 85` — constants to be replaced with depth-dependent values.

### Established Patterns
- Oracle state lives in `.aether/oracle/state.json` (not COLONY_STATE.json) — separate workspace with its own lifecycle
- The controller-owns-truth pattern: Go runtime selects questions and merges findings, workers just research one question at a time
- Env var config pattern in platform_dispatch.go: `AETHER_CODEX_PATH`, `AETHER_OPENCODE_PATH`, `AETHER_OPENCODE_PRIMARY_AGENT` — follow this naming convention for the callback URL var

### Integration Points
- `startOracleCompatibility()` — entry point where brief formulation and depth selection get added (before `runOracleLoop()`)
- `runOracleLoop()` — the iteration loop where smart formulation replaces `selectOracleQuestion()`
- `OpenCodeDispatcher.InvokeWithProgress()` — where the callback URL env var gets passed to the subprocess
- `cmd/compatibility_cmds.go` — CLI flag parsing for `--depth` flag on the oracle subcommand

</code_context>

<specifics>
## Specific Ideas

- The brief formulation step runs between `ensureOracleWorkspace()` and `runOracleLoop()` in `startOracleCompatibility()` — gather context, generate questions, then start the loop
- Depth prompt should show the four options with iteration counts and confidence targets so the user knows what each level means
- Smart formulation could score questions by: (1) number of open gaps that reference the question's topic, (2) whether contradictions exist that the question could resolve, (3) how many other questions would benefit from its findings, (4) raw confidence gap. Weighted combination.
- The callback URL env var should be read in `OpenCodeDispatcher.InvokeWithProgress()` and passed to the `cmd` environment
- Brief-informed questions should use the colony goal + pheromones + codebase structure to generate 5-8 specific questions instead of the current generic keyword-matched templates

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 60-oracle-loop-fix*
*Context gathered: 2026-04-27*
