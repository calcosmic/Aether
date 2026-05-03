# Phase 94: Recovery Data Model - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Worker failures have a deterministic classification system (recoverable, requires-attempt, blocking), transient failures are distinguished from systemic failures, and every recovery action is logged to a phase-scoped file. This is pure data model and classification infrastructure — no behavioral changes to existing build/continue flows, no auto-recovery orchestration (that's Phase 96). Downstream phases (95: Smart Gate Pipeline, 96: Auto-Recovery Orchestrator) consume the failure records and recovery logs this phase produces.

The phase adds: (1) a FailureRecord struct with classification, failure type (transient/systemic), original error, and timestamp, (2) deterministic classification rules mapping error patterns to failure types and classifications, (3) a phase-scoped recovery log file that records every recovery action with original error, action taken, and outcome, (4) real-time retry messages during build output.

</domain>

<decisions>
## Implementation Decisions

### Retry Boundary
- **D-01:** Three-tier failure classification: recoverable (auto-retry up to 3 times), requires-attempt (try once then tell user), blocking (immediate escalation, no retry). This aligns with RECV-01 and RECV-02 defaults.
- **D-02:** Recoverable failures retry up to 3 times with a small pause between attempts, then stop and tell the user with a clear message: "Worker X failed 3 times on [task]. Here's what went wrong." Per RECV-02 default budget.
- **D-03:** Requires-attempt failures get exactly one attempt. If it succeeds, great. If it fails, the colony stops and tells the user what happened. No silent skipping.
- **D-04:** Blocking failures never retry — the colony stops immediately and tells the user. These are fundamental problems that retrying won't fix.

### Recovery Visibility
- **D-05:** Real-time retry messages during build output. When a worker is retried, the user sees a one-line message like "Worker Builder-67 timed out on task 2 — retrying (attempt 2/3)". This builds trust — the user knows the colony is actively working, not silently stuck.
- **D-06:** One-line format per retry — concise message with worker name, failure reason, and attempt count. No full error output in the retry message (that goes in the recovery log file).
- **D-07:** Recovery log file is phase-scoped (per-phase, like gate-results-{N}.json). Contains full detail: original error, classification, action taken, retry attempts, outcome, timestamps. This is the detailed record for debugging.

### Error Classification Rules
- **D-08:** Transient failures (recoverable, auto-retry): timeout, context window overflow, temporary resource limits. These are environmental hiccups — the task itself is fine but the environment had a moment. Retrying makes sense.
- **D-09:** Systemic failures (blocking, immediate escalate): bad task spec, missing dependency, invalid file path, structural code error. These are fundamental problems with the task or setup — retrying won't help, the task itself needs to be fixed.
- **D-10:** Requires-attempt failures (try once then tell user): partial completion (worker finished some tasks but not all), garbled/unparseable worker output. These might recover on retry but might also indicate a deeper issue. One attempt, then escalate.
- **D-11:** Classification is by deterministic rules matching error patterns — not by LLM inference. Per RECV-05, the rules are code-level constants, not runtime interpretation. Unknown/uncategorized errors default to requires-attempt (safe middle ground).

### Claude's Discretion
- Exact struct field names and Go types for FailureRecord and RecoveryLogEntry
- Error pattern matching implementation (string matching, exit codes, or both)
- Recovery log file naming convention (follow gate-results-{N}.json pattern)
- How the recovery log relates to existing midden system (failure tracking)
- CLI commands for inspecting recovery logs (follow gate-classify pattern from Phase 93)
- Whether to add a recovery-log subcommand or extend an existing command
- Pause duration between retries
- How partial completion detection works (comparing task count vs completed count)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 93 (Gate Classification — parallel infrastructure)
- `.planning/phases/93-gate-classification-infrastructure/93-CONTEXT.md` — Gate classification tiers, QueenAnnotation struct, gate-classify CLI command pattern. The recovery data model should align with the classification and annotation patterns established here.

### Gate System (existing code to extend)
- `cmd/gate.go` — GateCheckResult struct, QueenAnnotation struct, gateClassificationEntry, gateClassify() function. The FailureRecord should follow similar struct patterns.
- `cmd/gate.go` § gate-results-{N}.json — Per-phase persistence pattern. Recovery log follows this same pattern.
- `cmd/gate_results.go` — Gate results validation and format patterns.

### Worker Dispatch and Failure Points
- `cmd/codex_continue.go` — Continue command that runs gates and handles failures.
- `cmd/codex_continue_finalize.go` — Finalize phase where recovery actions would be logged.
- `cmd/fixer_dispatch.go` — Fixer agent dispatch on gate failure (Phase 96 consumes the failure records this phase creates).
- `cmd/codex_worker_cleanup.go` — Stale worker cleanup. Failure records should integrate with this flow.

### Midden System (failure tracking — related but separate)
- `cmd/midden_cmds.go` — Existing failure tracking system. Recovery log is distinct (per-phase action log) but should not duplicate midden data.
- `cmd/medic_scanner.go` — Health diagnostics. Recovery log supplements but doesn't replace.

### Requirements
- `.planning/REQUIREMENTS.md` — RECV-01 (failure classification), RECV-05 (transient vs systemic), RECV-06 (phase-scoped recovery log)
- `.planning/ROADMAP.md` § Phase 94 — Success criteria and goal definition

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GateCheckResult` struct in `cmd/gate.go` — Pattern for structured result records with Name, Status, Detail, FixHint, RecoveryOptions, Timestamp, RetryCount, QueenAnnotation. FailureRecord should follow this pattern.
- `QueenAnnotation` struct — Decision trail pattern (decision, rationale, timestamp, queen_version). Recovery actions should use a similar annotation approach.
- `gate-results-{N}.json` persistence — Per-phase file pattern via `gateResultsWritePhase()` / `gateResultsReadPhase()`. Recovery log follows this exact pattern.
- `gateClassifications` map — Deterministic classification map pattern. Failure type classification follows this same approach (code-level constant, not configurable).
- `OutputWorkflow` pattern — `outputOK()` for JSON+visual, `outputError()` for errors. Any recovery-log CLI command follows this.

### Established Patterns
- Per-phase persistence: gate-results-{N}.json files with atomic JSON writes
- Classification as Go map constants: deterministic, not user-configurable
- Cobra CLI subcommands: flags for input, JSON output via outputOK()
- Atomic JSON writes via `store.UpdateJSONAtomically()` for safe concurrent access
- `omitempty` on all new struct fields for backward compatibility with old JSON

### Integration Points
- Phase 95 (Smart Gate Pipeline) will use failure classifications to decide gate auto-resolution behavior
- Phase 96 (Auto-Recovery Orchestrator) will consume FailureRecord and recovery log to drive retry/reassign/fixer dispatch
- Continue flow in `codex_continue.go` reads gate results — recovery log should be accessible from the same flow
- Midden system tracks failures — recovery log tracks recovery actions (distinct concerns, complementary data)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 94-Recovery Data Model*
*Context gathered: 2026-05-03*
