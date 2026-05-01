# Phase 75: Intelligence Core - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire Bayesian trust scoring into the wisdom pipeline so every observation gets a meaningful score, and add a circuit breaker that stops dispatching to workers that fail repeatedly during parallel builds.

Requirements: INTEL-04, INTEL-05.

**What this phase delivers:**
- Extended `memory-capture` command with `--source-type` and `--evidence-type` flags
- Continue ceremony playbooks updated to pass trust-relevant flags when capturing learnings
- Circuit breaker that tracks consecutive failures per worker instance and redistributes tasks to peers
- Per-wave reset so tripped workers get fresh chances each wave
- Both in-repo and worktree parallel modes protected

**What this phase does NOT deliver:**
- Changes to the trust scoring formula itself (already correct: 40/35/25 weighted, 60-day half-life)
- New source types or evidence types (existing 5+4 types are sufficient)
- Build ceremony learning capture (learning stays in continue only)
- Auto-detection of source/evidence types from colony state
- Circuit breaker reset by cooldown timer or manual intervention

</domain>

<decisions>
## Implementation Decisions

### Trust Scoring Integration
- **D-01:** Extend `memory-capture` with `--source-type` and `--evidence-type` flags. Keep existing defaults (`observation`/`anecdotal`) so unflagged callers are unaffected. Playbooks pass explicit flags for better scores.
- **D-02:** Playbook-driven source/evidence types — each ceremony step explicitly passes the appropriate flags when calling `memory-capture`. No auto-detection from colony state.
- **D-03:** Continue ceremony uses `--source-type success_pattern --evidence-type multi_phase` for learnings extracted from completed work. These reflect real, verified patterns from phase execution.

### Circuit Breaker Design
- **D-04:** Consecutive failure count triggers the breaker. A worker fails N times consecutively (configurable threshold, default 3). A single success resets the counter.
- **D-05:** When the breaker trips, pending tasks for that worker are redistributed to other workers of the same caste. No tasks are lost.
- **D-06:** Per-wave reset — the breaker resets at the start of each new build wave. A worker that tripped in wave 1 gets a fresh chance in wave 2.
- **D-07:** Per-worker instance granularity — each worker instance (e.g., Builder-Mason-67) has its own breaker. Other workers of the same caste are unaffected.
- **D-08:** Circuit breaker applies in both parallel modes (in-repo and worktree).

### Build Ceremony Learning Flow
- **D-09:** Build ceremony does NOT capture learnings — that stays in continue only. Clean separation: build is for building, continue is for reflecting.
- **D-10:** Continue ceremony captures learnings with trust scoring via the extended `memory-capture` command. No other changes to the continue flow.

### Claude's Discretion
- Exact consecutive failure threshold (default 3, adjustable via flag)
- How the circuit breaker state is stored (colony state field, in-memory only, etc.)
- Visual rendering of circuit breaker events in build output
- Whether to log a summary of tripped workers at wave end
- Test coverage approach for the circuit breaker

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — INTEL-04 (Bayesian confidence scoring), INTEL-05 (circuit breaker)

### Roadmap
- `.planning/ROADMAP.md` — Phase 75 goal, success criteria, dependency on Phase 74

### Trust Scoring Engine (already implemented)
- `pkg/memory/trust.go` — `Calculate()`, `Decay()`, `Tier()` functions with 40/35/25 formula and 60-day half-life
- `pkg/memory/trust_test.go` — Tests for trust scoring

### Observation Capture (already implemented)
- `pkg/memory/observe.go` — `CaptureWithTrust()` computes trust scores on observation capture; `Capture()` defaults to observation/anecdotal
- `pkg/memory/observe_test.go` — Tests for capture, dedup, and trust scoring
- `cmd/learning.go` — `learning-observe` command (has full flags), `memory-capture` command (missing source/evidence flags)

### Learning Commands
- `cmd/learning_cmds.go` — `learning-inject`, `learning-promote`, `learning-approve-proposals`, `learning-undo-promotions`

### Continue Ceremony Playbooks (where learnings are captured)
- `.aether/docs/command-playbooks/continue-advance.md:67-106` — Learning extraction and `memory-capture` calls
- `.aether/docs/command-playbooks/continue-full.md:1049-1073` — Learning extraction with `memory-capture`

### Build Ceremony Playbooks (parallel worker dispatch)
- `.aether/docs/command-playbooks/build-wave.md` — Worker dispatch and wave management
- `.aether/docs/command-playbooks/build-prep.md` — Build preparation including parallel mode

### Spawn Tracking
- `cmd/spawn_runs.go` — `runtimeSpawnRun`, `summarizeRunStatus()` for tracking worker status
- `pkg/agent/` — Spawn tree and agent pool

### Build Commands (Go runtime)
- `cmd/codex_build.go` — Parallel mode handling, wave dispatch, worker status
- `cmd/codex_continue.go` — Continue ceremony with learning extraction

### Structural Learning Stack Documentation
- `.aether/docs/structural-learning-stack.md` — Full pipeline documentation including trust scoring, event bus, instinct storage, curation ants

### Platform Wrappers
- `.claude/commands/ant/build.md` — Claude Code build wrapper
- `.claude/commands/ant/continue.md` — Claude Code continue wrapper
- `.opencode/commands/ant/build.md` — OpenCode build wrapper
- `.opencode/commands/ant/continue.md` — OpenCode continue wrapper

### Prior Phase Context
- `.planning/phases/74-suggest-analyze/74-CONTEXT.md` — Phase 74 decisions on suggest-analyze (companion intelligence feature)

### Architecture
- `CLAUDE.md` — Platform policy, zero-new-deps principle, wrapper-runtime contract
- `.planning/codebase/CONVENTIONS.md` — Go code style, output patterns, state management

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/memory/trust.go` — Complete trust scoring engine. `Calculate()` takes source type, evidence type, and days-since, returns score with tier. No changes needed.
- `pkg/memory/observe.go` — `CaptureWithTrust()` already computes trust scores correctly. `Capture()` is a convenience wrapper with defaults. No changes needed.
- `cmd/learning.go:181-218` — `memory-capture` command. Needs `--source-type` and `--evidence-type` flags added, then pass them to `obsService.CaptureWithTrust()` instead of `obsService.Capture()`.
- `cmd/spawn_runs.go` — `summarizeRunStatus()` already categorizes worker statuses (active, blocked, failed). Can be extended for circuit breaker state.
- `cmd/codex_build.go` — `ParallelMode` field and wave dispatch logic. Circuit breaker hooks go here.

### Established Patterns
- `outputOK()` / `outputError()` for JSON output + visual rendering
- `mustGetString()` / `mustGetInt()` for required flag values
- `cobra.Command` with `RunE` for all commands
- `store.LoadJSON()` / `store.SaveJSON()` for state persistence
- Per-wave build structure in playbooks

### Integration Points
- `memory-capture` command: add flags, change `Capture()` call to `CaptureWithTrust()`
- Continue-advance.md playbook: add `--source-type success_pattern --evidence-type multi_phase` to `memory-capture` calls
- Continue-full.md playbook: same flag additions
- Build-wave playbook: add circuit breaker checks before worker dispatch
- `cmd/codex_build.go`: add circuit breaker state tracking and task redistribution logic
- Colony state (`COLONY_STATE.json`): may need new fields for circuit breaker state per worker

</code_context>

<specifics>
## Specific Ideas

- The circuit breaker should be a simple counter in the Go runtime, not a complex state machine
- Trust scoring changes are small — two flags on one command and flag additions in playbooks
- Circuit breaker is the larger piece — it needs to integrate with the existing spawn/run tracking
- Per-wave reset aligns naturally with the existing wave-based build structure
- The breaker should be visible in build output so the user knows when a worker is being skipped

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 75-intelligence-core*
*Context gathered: 2026-04-29*
