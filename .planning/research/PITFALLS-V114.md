# Domain Pitfalls: v1.14 Queen Authority

**Domain:** Adding autonomous queen coordination, auto-recovery, smart gating, and output filtering to an existing multi-agent colony framework
**Researched:** 2026-05-03
**Confidence:** HIGH (based on direct codebase analysis of `cmd/circuit_breaker.go`, `cmd/gate.go`, `cmd/queen.go`, `cmd/fixer_dispatch.go`, `cmd/colony_prime_context.go`, and `.aether/docs/command-playbooks/continue-gates.md`)

---

## Critical Pitfalls

### Pitfall 1: Queen Auto-Recovery Creating Cascading Fix-Fail Cycles

**What goes wrong:**
The queen gains the ability to re-spawn, re-assign, or skip stalled workers autonomously. The existing circuit breaker (`cmd/circuit_breaker.go`) tracks per-worker-name consecutive failures with a configurable threshold (default 3) and supports same-caste peer redistribution (`findSameCastePeer`). But wiring the queen into this system creates a new failure cascade:

1. **Queen re-spawns a failed worker with a new deterministic name.** The circuit breaker resets on `Reset()` (per-wave, per D-06) but tracks by worker name. When the queen re-spawns "Builder Mason-67" as "Builder Mason-92" (new hash from the same caste+task), the circuit breaker sees a fresh worker and allows dispatch. The underlying failure (bad task spec, impossible requirement, missing dependency) hasn't changed, so Mason-92 fails too. The queen re-spawns again as "Builder Mason-41". This continues until the queen exhausts her dispatch budget or hits an arbitrary cap -- but the cap may be too high (wasting API tokens and time) or the failure is intermittent (passing 1 in 5 times, creating unpredictable partial-success states).

2. **Re-assignment to same-caste peer creates correlated failures.** The `findSameCastePeer` function (line 102-117) finds a non-tripped worker of the same caste. But if the task itself is broken (e.g., it references a file that doesn't exist, or its constraints contradict each other), reassigning to a different Builder doesn't help. The peer fails on the same task, gets tripped, and the queen runs out of same-caste peers. Now the queen has multiple tripped workers and no one to do the task.

3. **Skip decisions hide systemic problems.** When the queen decides to skip a stalled worker (no peers available, circuit breaker tripped), she marks the task as "skipped" and continues. But if the task was a prerequisite for other tasks (e.g., the Builder was supposed to create a module that a Watcher needs to test), skipping it causes downstream tasks to fail with confusing errors ("module not found"). The queen then tries to recover those downstream failures, creating a cascade of recovery attempts that all trace back to the original skip.

4. **Auto-recovery masks prompt engineering failures.** If a worker fails because its prompt is poorly constructed (too vague, contradictory constraints, missing context), auto-recovery treats this as a transient failure and retries. But the same prompt will produce the same failure. The queen burns through retries without ever identifying that the prompt itself is the problem. Only a human looking at the worker's output can diagnose this.

**Why it happens:**
The circuit breaker was designed for the case where a worker process crashes or times out -- a failure mode that re-spawning genuinely fixes. Queen auto-recovery extends this to all failure types, including logical failures (bad task spec, impossible constraints, missing context) that are not fixable by re-spawning. The system cannot distinguish between "the worker process died" (transient, retryable) and "the worker completed but produced garbage" (systemic, not retryable) because both manifest as "task failed."

**Consequences:**
- API token costs explode: each queen auto-recovery attempt consumes a full LLM invocation
- Workers cycle through names without making progress: `aether status` shows 8 failed Builder variants
- Downstream tasks fail with confusing errors when prerequisite tasks are skipped
- The queen appears busy (dispatching, recovering, re-dispatching) while making no forward progress
- Humans lose visibility: the queen is "handling it" but the colony is stuck in a recovery loop

**Prevention:**
- Classify worker failures into retryable vs non-retryable BEFORE attempting recovery. Retryable: process crash, timeout, network error, empty output. Non-retryable: worker completed but output failed validation, worker reported "blocked" status, task has been attempted 2+ times by different workers. For non-retryable failures, immediately escalate to the human instead of re-spawning.
- Track task-level attempt count in COLONY_STATE.json (not just worker-level). If a task has been attempted by 2 different workers and failed both times, the queen must NOT re-assign it -- she must flag it for human review.
- Add a "queen recovery budget" per phase: max 3 auto-recovery attempts per phase. When the budget is exhausted, the queen pauses and surfaces a summary of what she tried and what failed.
- When the queen skips a task, check whether any other task in the current phase depends on it (via the task dependency graph). If yes, skip the dependent tasks too and report the dependency chain to the human.
- Emit telemetry for every queen recovery action: what failed, what action the queen took, and the result. This creates an audit trail for diagnosing recovery loops after the fact.

**Detection:**
- Multiple workers of the same caste with different names have all failed in the same phase
- Queen dispatch log shows 3+ recovery attempts for the same task
- Downstream tasks fail with "module not found" or "file does not exist" after a task was skipped
- API token usage spikes during a phase where the queen is "handling" failures
- The queen has been active for more than 10 minutes without completing any new tasks

**Phase assignment:** Queen auto-recovery (first phase of v1.14)

---

### Pitfall 2: Smart Gates Auto-Resolving Findings That Should Block

**What goes wrong:**
Smart gates automatically resolve non-critical findings so only genuine problems block advancement. The existing gate system (`cmd/gate.go`) has 11 gates (spawn, anti_pattern, complexity, gatekeeper, auditor, tdd_evidence, runtime, flags, watcher_veto, medic, tests_pass) with per-gate recovery templates and a `shouldSkipGate` function that skips previously-passed gates. Smart gating adds a layer of intelligence on top: the queen evaluates whether a gate failure is "real" or a false positive, and auto-resolves false positives.

1. **The queen auto-resolves a legitimate gate failure as a false positive.** The auditor gate reports "quality score 55/100" (below the 60 threshold). The queen examines the finding, decides that the low score is because the worker wrote a large file (300+ lines) during refactoring, and auto-resolves it as "expected complexity from refactoring, not a quality issue." But the low score was actually caused by missing error handling in 4 functions -- a real problem that will cause runtime panics. The queen's judgment was wrong because she evaluated the surface explanation ("large file") instead of the root cause ("missing error handling").

2. **Auto-resolution erases the audit trail.** When the queen auto-resolves a gate failure, she marks it as "passed" in gate-results. But the original failure detail is cleared (see `resolveFixedGates` in `cmd/fixer_dispatch.go` line 236-238: `Detail = ""`, `FixHint = ""`, `RecoveryOptions = nil`). If the auto-resolution was wrong, there is no record of what the original finding was. The human has no way to review the queen's gate decisions after the fact.

3. **The queen develops a bias toward auto-resolution.** Auto-resolving a gate is "cheaper" than escalating to the human -- it keeps the phase moving. Over time, the queen's gate evaluation threshold drifts downward: findings that should block get auto-resolved because the queen has learned that "most gate failures are false positives." This is a form of reward hacking: the queen is optimizing for phase velocity, not code quality.

4. **Gate severity is context-dependent, but the queen evaluates in isolation.** A "high severity" finding in the auditor gate means something different in phase 1 (foundation code that everything depends on) vs phase 8 (leaf feature code). The queen evaluates each gate failure against a static severity threshold without considering the phase context. A high-severity finding in phase 1 should almost always block, but the queen may auto-resolve it using the same threshold she uses for phase 8.

5. **Auto-resolution of the watcher_veto gate undermines the Watcher's authority.** The watcher_veto gate (Step 1.13 in continue-gates.md) is the final quality check -- the Watcher scores the build and can veto advancement. If the queen can auto-resolve a watcher veto, the Watcher's role becomes advisory rather than authoritative. This contradicts the colony's core principle ("Watcher has final say" per the continue-gates playbook).

**Why it happens:**
Smart gating requires the queen to make judgment calls about gate findings -- specifically, whether a finding is a false positive. But LLM-based judgment is unreliable for security and quality assessments. The queen doesn't have access to the full context that produced the finding (she sees the gate result JSON, not the source code that triggered it). The existing gate system was designed to be deterministic: checks either pass or fail based on measurable criteria. Smart gating introduces non-deterministic judgment on top of deterministic checks, creating a layer where mistakes are invisible.

**Consequences:**
- Legitimate quality/security issues slip through because the queen auto-resolved them
- No audit trail of what the queen auto-resolved or why
- Watcher veto loses its authority as the final quality gate
- Queen develops a bias toward auto-resolution over time
- Humans lose trust in the gate system because "it always passes anyway"

**Prevention:**
- NEVER auto-resolve security gates (gatekeeper, anti_pattern). These must always block on critical findings. The queen can only auto-resolve quality gates (auditor, complexity) and only when the finding severity is MEDIUM or lower. HIGH and CRITICAL findings always block regardless of the queen's judgment.
- NEVER auto-resolve the watcher_veto gate. The Watcher's authority is a colony invariant. The queen can escalate a watcher veto to the human with her analysis ("I believe this veto is a false positive because..."), but she cannot override it.
- When the queen auto-resolves a finding, preserve the original failure detail in a separate "queen_gate_decisions.json" file. Record: `{gate_name, original_status, original_detail, queen_decision, queen_reason, timestamp}`. This creates a reviewable audit trail.
- Add a "gate auto-resolution rate" metric to `/ant-status`. If the queen is auto-resolving more than 30% of gate failures, flag it as a potential bias drift.
- Require the queen to read the relevant source code before auto-resolving any gate. She must not resolve a finding based solely on the gate result JSON -- she must verify that the finding is genuinely a false positive by reading the code that triggered it.

**Detection:**
- Gate results show "passed" for gates that previously showed "failed" with no human intervention
- `queen_gate_decisions.json` shows a high auto-resolution rate (>30%)
- Security issues appear in production that should have been caught by gatekeeper or anti_pattern gates
- Watcher veto is auto-resolved without human review
- Code quality degrades across phases despite all gates passing

**Phase assignment:** Smart gates (second phase of v1.14, after auto-recovery is stable)

---

### Pitfall 3: Output Filtering Hiding Critical Information From the User

**What goes wrong:**
Clean output means the queen filters and summarizes worker output so the user sees "what matters, not raw worker noise." The existing visual system (`cmd/codex_visuals.go`) renders caste identities, stage markers, and ceremony output. The colony-prime context builder (`cmd/colony_prime_context.go`) already trims sections by priority. Output filtering adds a queen-driven summarization layer on top.

1. **The queen filters out a legitimate error because it looks like noise.** Worker output often contains verbose build logs, test output, and dependency resolution messages interspersed with actual errors. A real compilation error ("undefined: Foo.Bar") may appear in the middle of 200 lines of dependency download output. The queen's filter, looking for "actionable" output, may classify the compilation error as "build noise" and suppress it. The user sees "Phase 3 complete -- all tasks done" without knowing that one of the "completed" tasks has a compilation error.

2. **Summarization loses the specifics that matter for debugging.** The queen summarizes worker output into a human-readable summary: "Builder completed 3 tasks: implemented auth module, added tests, updated docs." But the specifics that matter for debugging are lost: "Builder wrote `func Login()` that always returns `true`" becomes "implemented auth module." When the user later needs to debug the auth module, they have no record of what the worker actually did.

3. **Progress filtering creates a false sense of speed.** The queen filters out "in progress" updates and only shows completed tasks. The user sees a clean sequence of completions without seeing the failed attempts, retries, and recovery actions. This creates a false impression that the colony is working smoothly when it's actually struggling. The user cannot intervene early because they don't see the struggle.

4. **Gate failure details are filtered to "keep it clean."** Gate failures contain specific file paths, line numbers, and error descriptions. The queen may summarize these as "quality gate: 2 issues found" instead of showing the full detail. The user sees a clean status but cannot diagnose the problem without running additional commands.

5. **The queen's "importance" heuristic doesn't match the user's.** The queen and the user have different definitions of what's important. The queen optimizes for "does this require human action?" The user cares about "what did my colony actually do?" A worker that spent 15 minutes refactoring a module and produced a 500-line diff is "important" to the user even if it doesn't require action. But the queen filters it as "no action needed, completed successfully."

**Why it happens:**
Output filtering requires a model of what the user cares about, which is subjective and context-dependent. The queen's filter is based on LLM judgment, which optimizes for conciseness and actionability. But the user needs both conciseness AND the ability to drill down into details. The existing system already has this problem partially (colony-prime trims context by priority), but the user can always run `aether status` or read COLONY_STATE.json directly. Queen output filtering may hide information that the user doesn't even know to look for.

**Consequences:**
- Real errors hidden behind "all tasks completed" summaries
- Users cannot debug issues because the specific error messages were filtered out
- Users develop false confidence in the colony's output quality
- Users lose situational awareness of colony struggles
- Users must run additional commands to see what the queen filtered out, defeating the purpose of filtering

**Prevention:**
- Use a two-tier output model: the queen always shows a summary line for each worker/task, but NEVER filters error messages, gate failure details, or worker-reported blockers. These are always shown verbatim.
- Add a `--verbose` flag to the queen's output that shows the full unfiltered worker output. Make this the DEFAULT for the first phase after queen authority is enabled, so users can see what the queen is filtering and provide feedback.
- Store the full unfiltered worker output in `.aether/data/queen-output-{phase}.json` so users can review it after the fact, even if it wasn't shown in the terminal.
- Never filter output that contains any of these keywords: "error", "failed", "panic", "fatal", "blocked", "veto", "critical". These always pass through to the user.
- Show recovery actions: when the queen auto-recovers a failure, display a brief line like "Queen: re-spawned Builder Mason-92 (original failed with timeout)" so the user knows recovery happened.

**Detection:**
- User reports "I didn't know X failed until I looked at the code"
- `queen-output-{phase}.json` contains error messages that were not shown in the terminal
- User runs `--verbose` and discovers information that changes their understanding of the phase
- Gate failure details are missing from the terminal output but present in gate-results files

**Phase assignment:** Clean output (third phase of v1.14, after auto-recovery and smart gates are stable)

---

### Pitfall 4: Queen Coordination Introducing a Single Point of Failure

**What goes wrong:**
The queen manages the wave lifecycle end-to-end within a phase. Currently, the build flow (`cmd/codex_build.go`) dispatches workers and the continue flow (`cmd/codex_continue.go`) runs gates and advances the phase. The queen sits between these flows, coordinating them. This creates a new single point of failure:

1. **Queen decision loop becomes a bottleneck.** The queen must make a decision before every worker dispatch, every gate evaluation, and every phase transition. If the queen's decision process is slow (LLM call with full colony context), the entire build-continue cycle slows down. Workers that could run in parallel now wait for the queen to approve each one.

2. **Queen crash corrupts the coordination state.** The queen maintains coordination state (which workers are active, which tasks are in progress, which gates have been evaluated). If the queen process crashes (OOM, SIGKILL, network error), this state is lost. The colony is left with workers that were dispatched by the queen but have no coordinator. Workers may complete, but their results are not collected because the queen that dispatched them is gone.

3. **Queen prompt context grows unboundedly.** The queen needs context about all active workers, pending tasks, gate results, and recovery history to make good decisions. As a phase progresses, this context grows. The existing colony-prime context budget (8K chars) was designed for worker context injection, not queen coordination context. If the queen's context exceeds her effective window, her decisions degrade.

4. **Queen authority conflicts with existing gate authority.** The existing gate system has clear ownership: `gate.go` owns gate evaluation, `circuit_breaker.go` owns failure tracking, `fixer_dispatch.go` owns recovery. The queen needs to interact with all of these, but if she bypasses them (e.g., dispatching a worker without going through the circuit breaker check, or marking a gate as passed without running the actual check), she creates parallel authority paths that can produce inconsistent state.

5. **The queen cannot be tested in isolation.** The queen's decisions depend on the full colony state: workers, tasks, gates, pheromones, instincts, hive wisdom. Testing the queen requires setting up the entire colony state, which makes unit tests fragile and E2E tests slow. Bugs in the queen's decision logic are hard to reproduce because they depend on specific colony state combinations.

**Why it happens:**
The queen is being added as a coordinator on top of a system that was designed to work without one. The existing flows (build dispatches workers, continue runs gates) are relatively independent. The queen creates a dependency between them: she must approve worker dispatches and gate evaluations. This centralizes authority that was previously distributed, creating a single point of failure and a bottleneck. The existing system's resilience comes from its relative simplicity -- each flow does one thing. The queen makes the system more complex by adding a decision layer on top.

**Consequences:**
- Build-continue cycle slows down because the queen must approve each step
- Queen crash leaves the colony in an inconsistent state with orphaned workers
- Queen decisions degrade as context grows beyond her effective window
- State inconsistencies when the queen bypasses existing gate/circuit breaker authority
- Queen bugs are hard to test and reproduce

**Prevention:**
- The queen should ADVISE, not COMMAND. The queen evaluates the colony state and produces recommendations ("re-spawn worker X", "auto-resolve gate Y", "skip task Z"), but the existing Go runtime functions (`circuit_breaker.Allow()`, `shouldSkipGate()`, `dispatchFixer()`) remain the actual decision makers. The queen's role is to assemble the inputs and call the existing functions, not to replace them.
- Persist queen coordination state to COLONY_STATE.json using the existing `UpdateJSONAtomically` pattern. If the queen crashes, the next invocation reads the persisted state and resumes coordination. Never hold coordination state only in memory.
- Budget the queen's context separately from colony-prime. The queen needs her own context budget (suggest 12K chars) that includes: active workers and their status, pending tasks, gate results, recovery history, and pheromone signals. Do NOT reuse the colony-prime 8K budget for queen coordination.
- Add a queen health check to `aether patrol`: verify that queen coordination state is consistent with COLONY_STATE.json (no orphaned workers, no pending tasks without active workers, no gates in an impossible state).
- Write the queen's decision logic as pure functions that take colony state as input and return recommendations as output. This makes the logic testable without setting up the full colony.

**Detection:**
- Build-continue cycle takes noticeably longer after queen authority is enabled
- COLONY_STATE.json shows workers in "dispatched" state with no coordinator
- Queen's decisions become inconsistent or contradictory as a phase progresses
- Gate results show states that are impossible given the gate evaluation logic
- E2E tests for the queen are slow and flaky

**Phase assignment:** Queen phase coordination (first phase of v1.14, foundational)

---

### Pitfall 5: Auto-Recovery and Smart Gates Creating a "Silent Failure" Mode

**What goes wrong:**
The combination of auto-recovery (pitfall 1) and smart gates (pitfall 2) creates a system where failures are handled so smoothly that the user never knows anything went wrong. This is the most dangerous pitfall because it undermines the core value proposition of Aether: "runtime truth" and "inspectable state."

1. **A worker produces subtly wrong code.** The Builder implements a feature but introduces a subtle bug (off-by-one error, race condition, wrong default value). Tests pass because the bug is in an untested edge case. The auditor gate reports "quality score 72" (above threshold). The watcher_veto gate reports "score 8/10" (above threshold). The queen sees all gates passing and marks the phase as complete. The bug is now in the codebase, invisible to the gate system and the user.

2. **The queen auto-recovers from a failure that reveals a deeper problem.** A worker fails because of a missing dependency. The queen re-spawns the worker, and the second attempt succeeds (the dependency was installed by another worker in the meantime). The queen logs "recovered: worker re-spawned after dependency failure." But the missing dependency was a symptom of a planning error: the phase plan didn't specify the correct task ordering. The queen's smooth recovery masks the planning problem, which will recur in future phases.

3. **Smart gate auto-resolution creates a false quality trend.** Over multiple phases, the queen auto-resolves more and more gate findings. The `/ant-status` quality metrics show a steady improvement because fewer gates are blocking. But the actual code quality isn't improving -- the queen is just getting more lenient. The user sees "quality score trending up" and thinks the colony is getting better, when in fact the evaluation criteria are drifting.

4. **The user loses the ability to intervene because they don't know they need to.** The current system requires human intervention at gates: the user sees a gate failure, reads the details, and decides what to do. This is "friction" (as stated in PROJECT.md: "too much manual intervention at gates"), but it's also visibility. The queen removes the friction but also removes the visibility. The user cannot intervene on a problem they don't know exists.

**Why it happens:**
Auto-recovery and smart gates are designed to reduce human intervention, which is a valid goal. But the current gate system's "friction" serves a dual purpose: it slows down advancement (good for catching problems) AND it informs the user (essential for trust). Removing friction without preserving visibility creates a system that appears to work well but may be accumulating invisible problems.

**Consequences:**
- Subtle bugs accumulate across phases without being caught
- Planning errors are masked by smooth recovery
- Quality metrics become meaningless as evaluation criteria drift
- Users lose trust when they discover hidden problems
- The colony's output quality degrades gradually, making it hard to identify when the decline started

**Prevention:**
- Always log every queen decision (auto-recovery, gate auto-resolution, task skip) to a persistent file. Make this log accessible via `/ant-status` or a new `/ant-queen-log` command. The user should be able to review every autonomous decision the queen made, even if they weren't prompted to approve it.
- Add a "queen activity summary" at the end of each phase: "Queen made 3 auto-recovery decisions and 1 gate auto-resolution this phase. Review with `/ant-queen-log`." This keeps the user informed without requiring their intervention.
- Implement a "queen transparency mode" (on by default) where the queen prints a single line for every autonomous decision: "Queen: auto-resolved auditor gate (quality score 58, threshold 60) -- reason: refactoring artifact, not quality issue." This takes 1 line of terminal output per decision and preserves full visibility.
- Periodically (every 3 phases) require the queen to present a "trust report" to the user: summary of auto-resolutions, auto-recoveries, and skipped tasks. The user reviews this and can adjust the queen's autonomy level.
- Never auto-resolve a finding that was flagged by two independent gates. If both the auditor AND the watcher flag the same issue, it must block regardless of the queen's judgment. Dual-flagged findings indicate a real problem that two independent evaluations agreed on.

**Detection:**
- Users discover bugs in code that "passed all gates"
- Queen auto-resolution rate increases across phases without a corresponding quality improvement
- `/ant-status` shows quality trending up but code review shows quality trending down
- Users express surprise when they see the queen's decision log: "I didn't know it did that"
- Planning errors recur across phases without being identified

**Phase assignment:** Cross-cutting concern -- applies to all v1.14 phases, must be addressed in the first phase

---

## Moderate Pitfalls

### Pitfall 6: Queen Authority Conflicting With Existing Fixer Caste

**What goes wrong:**
The Fixer caste (added in v1.13, defined in `.claude/agents/ant/aether-fixer.md`) already handles gate recovery with three autonomy modes: full, propose, and advise. The queen's auto-recovery duplicates some of this functionality. If both the queen and the Fixer try to recover from the same failure, they may conflict: the queen re-spawns a worker while the Fixer is already investigating the failure. The re-spawned worker may modify files that the Fixer is reading, causing the Fixer to produce an incorrect diagnosis.

**Prevention:**
- The queen should be the SOLE recovery coordinator. When the queen decides to recover from a failure, she either handles it herself (for simple re-spawns) or delegates to the Fixer (for complex gate failures). The Fixer should never be spawned independently of the queen's coordination.
- Add a "recovery lock" to COLONY_STATE.json: when the queen initiates recovery for a task/gate, she sets the lock. The Fixer checks the lock before acting. If a recovery is already in progress, the Fixer defers.

---

### Pitfall 7: Queen Phase Coordination Breaking the Wave Dispatch Model

**What goes wrong:**
The existing build flow dispatches workers in waves (see `cmd/codex_build.go` and `cmd/codex_dispatch_contract.go`). Each wave has a set of workers that run in parallel. The queen coordinating within a phase means she may want to dispatch workers between waves, re-order tasks, or split waves. But the wave dispatch model assumes that all workers in a wave are dispatched together and complete before the next wave starts. If the queen injects new workers into an active wave, the wave completion logic may not account for them.

**Prevention:**
- The queen should coordinate at the wave level, not the individual worker level. She can decide "start wave 2" or "retry wave 1 with different workers" but should not inject individual workers into an active wave.
- Add a `queen_coordinated` flag to wave state in COLONY_STATE.json. When set, wave completion checks also verify that the queen has approved the wave results before advancing.

---

### Pitfall 8: Queen Context Budget Competing With Colony-Prime Context Budget

**What goes wrong:**
The existing colony-prime context builder (`cmd/colony_prime_context.go`) has an 8K character budget with a defined trim order. The queen needs her own context for coordination decisions. If both the queen and colony-prime compete for the same token budget in the LLM prompt, the queen's coordination context may get trimmed, leading to poor decisions. Alternatively, if the queen's context is added on top of colony-prime's context, the total prompt may exceed the LLM's effective window.

**Prevention:**
- Give the queen a separate context budget (12K chars) that is independent of colony-prime's 8K budget.
- The queen's context should include: active workers, pending tasks, gate results, recovery history, pheromone signals, and a summary of hive wisdom. Do NOT include the full colony-prime context in the queen's context.
- If the queen needs to make a decision about a specific worker, she reads that worker's full context on demand (via the existing colony-prime pipeline) rather than including all worker contexts in her own context.

---

### Pitfall 9: Queen Autonomy Level Not Persisting Across Sessions

**What goes wrong:**
The queen's autonomy level (how much she can do without human approval) is a user preference. If this is stored only in memory or in the LLM conversation, it resets when the session ends (after `/clear` or a new conversation). The queen may start a new session with full autonomy when the user expected limited autonomy, or vice versa.

**Prevention:**
- Store the queen's autonomy level in COLONY_STATE.json as `queen_autonomy_level` with values: `"full"` (auto-recover + auto-resolve), `"advisory"` (recommend but don't act), `"manual"` (no autonomous actions).
- Expose a `/ant-queen-autonomy <level>` command for the user to adjust.
- The queen reads the autonomy level at the start of every coordination decision, not just at session start.

---

## Minor Pitfalls

### Pitfall 10: Queen Ceremony Output Clashing With Existing Visual System

**What goes wrong:**
The existing visual system (`cmd/codex_visuals.go`) renders caste identities, stage markers, and ceremony events. Queen coordination adds new ceremony events (recovery, auto-resolution, gate decision). If these new events use the same visual format as existing events, the terminal output becomes cluttered and hard to parse.

**Prevention:**
- Use a distinct visual prefix for queen actions (e.g., a crown emoji or a different ANSI color) so users can distinguish queen decisions from worker output.
- Batch queen ceremony output: instead of printing a line for every coordination decision, accumulate decisions and print a summary at the end of each wave or phase.

---

### Pitfall 11: Queen Decision Latency Making /ant-run Less Responsive

**What goes wrong:**
`/ant-run` is the autopilot that chains build-verify-advance. Currently, each step runs immediately after the previous one completes. With queen coordination, each step requires a queen decision (an LLM call). This adds latency to every step, making `/ant-run` noticeably slower.

**Prevention:**
- Pre-compute queen decisions where possible: the queen can evaluate the current colony state and produce a batch of decisions ("dispatch wave 1 with these workers, then run these gates, then advance if all pass") in a single LLM call, rather than making a separate call for each decision point.
- Cache queen decisions for deterministic scenarios: if the colony state hasn't changed since the last decision, reuse the cached decision.

---

### Pitfall 12: Queen Auto-Recovery Interacting With Worktree Parallel Mode

**What goes wrong:**
In worktree parallel mode, each worker gets an isolated git worktree. If the queen re-spawns a failed worker, she must decide whether to reuse the failed worker's worktree (which may contain partial work) or create a new one. Reusing the worktree may expose the new worker to the failed worker's partial state. Creating a new one wastes time and disk space.

**Prevention:**
- When re-spawning in worktree mode, always create a fresh worktree. Partial state from a failed worker is not reliable -- the worker may have been in the middle of a write when it failed.
- Clean up the failed worker's worktree before creating the new one. Use the existing worktree cleanup logic.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Queen phase coordination (foundational) | Single point of failure, context budget overflow, wave dispatch conflicts | Queen advises not commands, separate context budget, coordinate at wave level not worker level |
| Queen auto-recovery | Cascading fix-fail cycles, masking prompt failures, skipping prerequisite tasks | Failure classification, task-level attempt cap, dependency-aware skip, recovery budget |
| Smart gates | Auto-resolving legitimate findings, erasing audit trail, watcher veto override | Never auto-resolve security/watcher gates, preserve original findings in audit log, read source code before resolving |
| Clean output | Hiding real errors, losing debugging specifics, false sense of speed | Two-tier output, --verbose default for first phase, persistent full output, never filter error keywords |
| Integration with /ant-run | Decision latency, cached decisions going stale | Batch queen decisions, cache with state-change invalidation |

---

## "Looks Done But Isn't" Checklist

- [ ] **Queen recovery respects task dependencies:** Often tested with independent tasks -- test with a task graph where task B depends on task A, and task A fails. Verify the queen does NOT skip task A and proceed to task B.
- [ ] **Smart gates never auto-resolve security findings:** Often tested with quality findings -- test with gatekeeper critical CVEs and verify the queen always blocks.
- [ ] **Output filtering preserves error messages:** Often tested with clean worker output -- test with a worker that produces 200 lines of noise containing one real error. Verify the error is shown.
- [ ] **Queen survives process crash:** Often tested with clean shutdown -- test with SIGKILL during an active queen coordination cycle. Verify state is recoverable.
- [ ] **Queen autonomy level persists across sessions:** Often tested within a single session -- test with `/clear` or a new conversation. Verify the autonomy level is restored from COLONY_STATE.json.
- [ ] **Queen doesn't conflict with Fixer caste:** Often tested with queen-only recovery -- test with a gate failure that triggers both queen auto-recovery and Fixer dispatch. Verify they don't conflict.
- [ ] **Queen context stays within budget across a full phase:** Often tested with small phases (2-3 tasks) -- test with a large phase (8+ tasks, multiple waves). Verify the queen's context doesn't overflow.
- [ ] **Watcher veto remains authoritative:** Often tested with watcher score 8+ -- test with watcher score 6 and critical findings. Verify the queen cannot auto-resolve it.
- [ ] **Decision audit trail is complete:** Often tested with a single auto-recovery -- test with a full phase that has 5+ queen decisions. Verify all decisions are logged with reasons.
- [ ] **Worktree mode cleanup after re-spawn:** Often tested in in-repo mode -- test with worktree parallel mode and a failed worker. Verify the old worktree is cleaned up before the new one is created.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Cascading fix-fail cycles | LOW | Add task-level attempt cap; review queen decision log; no data migration |
| Smart gate false positive | MEDIUM | Review queen_gate_decisions.json; revert auto-resolved gates; add to manual review list |
| Output filtering hiding errors | LOW | Enable --verbose mode; review queen-output files; no data migration |
| Queen single point of failure | HIGH | Restore COLONY_STATE.json from backup; re-run queen coordination; possible state inconsistency |
| Silent failure mode | HIGH | Audit all code since queen authority was enabled; may need manual code review for each phase |

---

## Sources

- Direct codebase analysis: `cmd/circuit_breaker.go` (circuit breaker, per-worker-name tracking, same-caste peer redistribution)
- Direct codebase analysis: `cmd/gate.go` (11 gates, shouldSkipGate, gate recovery templates, gateResultsWrite/Read)
- Direct codebase analysis: `cmd/queen.go` (queen commands, QUEEN.md management, wisdom promotion)
- Direct codebase analysis: `cmd/fixer_dispatch.go` (Fixer dispatch, resolveFixedGates, circuit breaker integration, attempt cap)
- Direct codebase analysis: `cmd/colony_prime_context.go` (context budget system, 8K char limit, section trimming)
- Direct codebase analysis: `cmd/unblock_cmd.go` (gate recovery summary, Fixer dispatch trigger)
- Direct codebase analysis: `.aether/docs/command-playbooks/continue-gates.md` (11 gate definitions, watcher veto authority, gate decision logic)
- Direct codebase analysis: `.claude/agents/ant/aether-fixer.md` (Fixer agent definition, three autonomy modes, protected paths)
- Direct codebase analysis: `cmd/codex_visuals.go` (caste identity system, ceremony event rendering)
- Direct codebase analysis: `cmd/codex_dispatch_contract.go` (worker dispatch contract, wave management)
- Direct codebase analysis: `cmd/codex_build.go` (build flow, wave dispatch)
- PROJECT.md v1.14 requirements (queen auto-recovery, smart gates, clean output, phase coordination)
- CLAUDE.md (existing architecture, platform policy, colony state management)
- [Agent Response Filtering -- Upsonic AI](https://upsonic.ai/lexicon/agent-response-filtering) (output filtering risks, over-filtering concerns)
- [Establishing Trust in AI Agents -- Medium](https://medium.com/@adnanmasood/establishing-trust-in-ai-agents-i-monitoring-control-reliability-and-accuracy-f440664df5fd) (supervisor pattern, monitoring, control layers)
- [Agentic AI Security: Threats, Defenses, Evaluation -- arXiv](https://arxiv.org/html/2510.23883v1) (output filtering vs sandboxing tradeoffs)

---
*Pitfalls research for: Aether v1.14 Queen Authority*
*Researched: 2026-05-03*
