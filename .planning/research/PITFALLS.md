# Domain Pitfalls: v1.21 Production Orchestration

**Domain:** Aether colony framework -- flipping TS host from simulation to real worker dispatch, adding worker-to-worker spawning, confidence-driven iteration loops, and cross-colony hive wisdom reuse to an existing Go/TypeScript hybrid architecture.
**Researched:** 2026-05-18
**Confidence:** HIGH (findings grounded in direct code inspection of Go/TS boundary, spawn budget, hive store, Oracle lifecycle, and worker dispatch modules)

---

## Critical Pitfalls

Mistakes that cause silent data loss, broken colony state, unbounded loops, or cross-colony corruption.

### Pitfall 1: Simulation Guard Silently Blocks Production Dispatch

**What goes wrong:** The lifecycle orchestrator in `lifecycle.ts` (lines 268-275) contains a hard gate: if `simulateWorkers` is not `true`, the entire lifecycle returns `success: false` with an error message saying "Lifecycle host is experimental and simulate-only." When the team flips `simulateWorkers` to `false` for production, this gate blocks everything and makes it look like the TS host is broken rather than merely outdated.

**Why it happens:** The lifecycle module was marked "experimental" in v1.16 and protected by this gate. The separate `worker-dispatch.ts` already has real dispatch code (the `dispatchRealWorker` path at line 237), but the lifecycle orchestrator never calls it because it short-circuits before reaching the build step. The gate was added to prevent premature production use, but it was never updated as the dispatch code matured.

**Consequences:**
- Production cutover appears to fail immediately. The team wastes time debugging "why doesn't it work" when the answer is "remove the simulation gate."
- If the gate is removed hastily without checking all downstream paths, the code falls through to the real dispatch path which has never been exercised end-to-end with a live platform.
- The lifecycle module's build step (line 424) hard-codes `const simulateWorkers = true` even after the entry gate is removed, meaning the gate is duplicated.

**Prevention:**
1. **Remove the gate in the first phase of v1.21.** The gate at `lifecycle.ts:268` must be the first thing deleted. Document the removal in the phase plan.
2. **Remove the hardcoded `simulateWorkers = true` at line 424.** This is a second copy of the same guard. Both must be removed.
3. **Add a feature flag, not a simulation flag.** Replace `simulateWorkers` with `--simulate` as an explicit CLI opt-in for testing. The default path should be real dispatch.
4. **Integration test with mock platform.** Before removing the gate, write a test that mocks `spawnWorker` to return realistic claims JSON, proving the full pipeline works from manifest to finalizer.
5. **Audit all `simulateWorkers` references.** Grep for the term across the TS host. Every reference must be intentional and gated, not accidental.

**Detection:**
- `grep -rn "simulateWorkers" .aether/ts-host/src/` -- any reference in the default production path is a bug.
- `grep -rn "simulate-only\|experimental" .aether/ts-host/src/lifecycle.ts` -- these comments must be removed or updated.
- Test: `test/lifecycle.test.ts` should have a test named `lifecycle dispatches real workers by default`.

**Which phase addresses it:** First phase of v1.21. This is the prerequisite for everything else.

---

### Pitfall 2: Worker Claims Parsing Fails on Real Platform Output

**What goes wrong:** The `parseWorkerClaims` function in `claims-parser.ts` expects clean JSON output from platform workers. Real platform output contains markdown wrappers, thinking traces, stderr diagnostics, and API error payloads mixed with the claims JSON. The parser fails, and the TS host reports "Failed to parse worker claims" instead of recognizing the real error.

**Why it happens:** The parser has three strategies (direct JSON, code-fence stripped, trailing JSON block), but these all assume the worker successfully produced claims JSON somewhere in the output. When the platform returns an auth failure, rate limit, or internal error, the output contains none of these formats. The known issues doc (2026-05-15) already documented this: "A real lifecycle smoke launched OpenCode-compatible workers and the provider returned an auth/API failure payload instead of Aether worker claims JSON."

**Consequences:**
- Real worker failures are misreported as "parse errors," hiding the actual problem (auth, rate limit, provider error).
- The TS host marks the worker as `failed` with an unhelpful summary, losing diagnostic information.
- Users see "parse worker output: no JSON found in output" when the real problem is their API key expired.
- The Go finalizer receives a completion file with a vague failure reason, making colony state useless for debugging.

**Prevention:**
1. **Pre-parse classification.** Before attempting to extract claims JSON, scan the output for known platform error patterns: auth failures, rate limits, timeout markers, and API error envelopes. Classify the failure before parsing.
2. **Structured error payload.** When claims parsing fails, return a `DispatchResult` with `status: "failed"` and a `summary` that includes: (a) the classification (auth/timeout/parse/unknown), (b) the sanitized diagnostic (no tokens), (c) the platform name and exit code.
3. **Worker claims schema enforcement.** The `workerClaimsSchema()` in `platform-dispatcher.ts` is passed via `--json-schema` to Claude and Codex. But OpenCode's `run --format json` does not support a schema flag. For OpenCode, the parser is the only line of defense. Add OpenCode-specific post-processing.
4. **Diagnostic artifact.** When parsing fails, write a debug artifact (to `/tmp/aether-worker-debug-{name}.txt`) containing the first 500 chars of sanitized stdout and stderr. Reference this artifact in the DispatchResult summary.
5. **Retry classification.** Not all parse failures warrant a retry. Auth failures should not be retried (they'll fail again). Rate limits should be retried with backoff. Parse failures of otherwise clean output might warrant retry. The existing `classifyFailure` in `escalation.ts` does not cover these cases.

**Detection:**
- Real dispatch test that returns non-JSON output. Currently, tests only exercise the happy path.
- `TestClaimsParserClassifiesPlatformErrors` -- a test that feeds auth failure text to `parseWorkerClaims` and verifies the error message includes the classification.
- Known issue regression: "Post-launch provider/API/auth failures can look like worker parse failures" from `known-issues.md`.

**Which phase addresses it:** First phase (production dispatch). This must work before any real worker can be dispatched.

---

### Pitfall 3: Worker-to-Worker Spawning Creates Unbound Fan-Out

**What goes wrong:** A builder worker completes its task and requests a sub-worker (e.g., a watcher to verify, or a specialist to handle a subtask). The TS host dispatches the sub-worker. That sub-worker also requests a sub-worker. The fan-out is unbounded: 1 worker becomes 2, becomes 4, becomes 8. Token costs explode, file locks contend, and the colony hangs.

**Why it happens:** The `WorkerClaims` interface in `claims-parser.ts` already has a `spawns?: string[]` field and the `workerClaimsSchema()` includes it as a required field. This means the current design already expects workers to request sub-workers. But the TS host dispatch code (`dispatchSingleWorker`) and the wave orchestrator (`dispatchWaves`) have no logic for handling the `spawns` field. If workers start returning `spawns` arrays, the TS host ignores them silently, and the sub-workers are never dispatched. If the team adds dispatch logic for `spawns` without depth limits, the fan-out problem appears.

**Consequences:**
- Token costs multiply exponentially. A 3-deep spawn tree with 2 children each creates 14 total worker invocations for what was supposed to be 1 task.
- File lock contention: Go's `pkg/storage` file locking expects sequential finalizer calls. Parallel sub-worker finalizers create lock contention or deadlocks.
- Manifest inconsistency: Go manifest defines N workers. TS host dispatches N + M sub-workers. Go finalizer receives a completion file with workers it never manifested. This violates the boundary contract rule #3: "TS host never invents workers -- dispatches only from Go manifest."
- Queen spawn budget bypass: Go's `queenSpawnBudgetForPhase` limits workers to 4-8 depending on risk level. Sub-workers spawned by workers bypass this budget entirely.

**Prevention:**
1. **Max spawn depth of 2.** The existing Classic design (CHANGELOG.md line 710) says "workers can spawn sub-workers up to depth 3." Change this to depth 2. A builder can spawn a watcher, but the watcher cannot spawn anything. This is a hard ceiling.
2. **Sub-workers go through Go manifest.** The TS host must not dispatch sub-workers directly. Instead, when a worker returns `spawns`, the TS host calls `aether build --plan-only --parent-worker {name} --spawn-requests {json}` to get a new manifest that includes the sub-workers. Go applies the spawn budget and returns a manifest (possibly pruned). TS host dispatches from the new manifest.
3. **Spawn budget inheritance.** Sub-worker manifest requests include the parent's remaining budget. Go deducts the parent's cost from the budget before allocating sub-workers.
4. **The `spawns` field in WorkerClaims is advisory only.** Workers can suggest sub-workers, but Go decides whether to allocate them. This is the same authority model as the main manifest.
5. **Claude Code subagent limitation.** The Queen agent definition explicitly states: "Claude Code subagents cannot spawn other subagents. Only Queen has access to the Task tool for spawning named agents." This means worker-to-worker spawning only works on platforms that support it (Codex with `spawn_agents_on_csv`). On Claude Code, sub-worker requests must be routed back to the TS host for dispatch.

**Detection:**
- `TestSubWorkerRespectsDepthLimit` -- a test that provides a `spawns` array at depth 2 and verifies the TS host rejects it.
- `TestSubWorkerGoesThroughManifest` -- a test that verifies sub-workers are dispatched from a Go manifest, not directly.
- Completion file validation: Go finalizer rejects completion files with workers not in the manifest.

**Which phase addresses it:** Phase 2 (worker-to-worker spawning). This is a new capability, not a fix, so it should be its own phase with dedicated tests.

---

### Pitfall 4: Confidence-Driven Iteration Loop Never Converges

**What goes wrong:** The plan/build loop iterates until confidence targets are met. But confidence is computed from partial information (test results, coverage, review findings). The loop keeps rebuilding with minor changes, each time gaining a small confidence increment but never reaching the target. The colony loops indefinitely, burning tokens.

**Why it happens:** The Oracle lifecycle already has this pattern (`runOracleLifecycle` loops until `confidence_target` is met or `max_iterations` is reached). But the plan/build confidence loop is different: it re-plans and re-builds entire phases. Each iteration costs significant tokens (full build wave dispatch). If the confidence target is 95% and each iteration only gains 2-5%, the loop runs 10-20 times.

The Oracle loop avoids this with `max_iterations` (a hard ceiling). But the plan/build loop currently has no such ceiling. The existing `checkStopConditions` in `oracle-lifecycle.ts` only checks `max_iterations` and `confidence_met`. For the broader plan/build loop, no equivalent exists.

**Consequences:**
- Token costs multiply: 20 iterations x 4-8 workers x $0.10 per worker = $8-16 per phase.
- Time: 20 iterations x 10 minutes per build = 3+ hours per phase.
- Colony state bloat: each iteration writes to the event bus, midden, and colony state. The state file grows.
- Queen spawn budget is respected per iteration but not across iterations. 20 iterations x 6 workers = 120 total worker invocations, far exceeding the intended 6.

**Prevention:**
1. **Hard iteration ceiling.** The plan/build loop must have a `max_iterations` parameter (default: 3). After 3 plan/build cycles without reaching the confidence target, the loop stops and reports "confidence target not met after N iterations."
2. **Diminishing returns detection.** Track confidence delta between iterations. If the delta is below a threshold (e.g., 5%) for two consecutive iterations, stop early. This is the same pattern as the Oracle's diminishing returns detection.
3. **Cross-iteration budget.** The Queen spawn budget must be cumulative across iterations, not per-iteration. If the budget is 6 workers per build, and the loop runs 3 times, the total budget is 18. But each subsequent iteration should receive fewer workers (iteration 1: 6, iteration 2: 4, iteration 3: 2).
4. **Explicit user consent for iteration > 1.** The first plan/build cycle runs automatically. After that, the TS host should pause and ask the user: "Confidence is at X% (target Y%). Iterate again? [y/n]"
5. **Confidence source validation.** The confidence metric must come from Go (test results, coverage analysis, review gates), not from worker self-reports. Workers claiming "I'm 90% confident" is not the same as test coverage being 90%.

**Detection:**
- `TestConfidenceLoopStopsAtMaxIterations` -- a test that mocks confidence at 60% and verifies the loop stops after `max_iterations`.
- `TestConfidenceLoopDetectsDiminishingReturns` -- a test that provides decreasing confidence deltas and verifies early termination.
- `TestConfidenceLoopBudgetCumulative` -- a test that verifies the total worker count across all iterations does not exceed the cumulative budget.

**Which phase addresses it:** Phase 3 (confidence-driven iteration). This must be implemented with the ceiling and diminishing returns detection from the start.

---

### Pitfall 5: Hive Wisdom Reuse Injects Stale or Irrelevant Advice

**What goes wrong:** Colony B starts and receives hive wisdom from Colony A. But the wisdom is stale (Colony A's instincts were formed under different conditions), irrelevant (Colony A was a Go project, Colony B is a TypeScript project), or conflicting (two colonies produced contradictory wisdom with similar confidence). Colony B's workers follow bad advice, producing incorrect code or wasting time on irrelevant patterns.

**Why it happens:** The `HiveStore` in `pkg/learn/hive_store.go` uses text matching and domain tags to retrieve wisdom. The `abstractContent` method (line 81) strips repo-specific paths, replacing them with `<repo>`. But abstraction is lossy: "Use `pkg/storage` for file locking" becomes "Use storage for file locking," which is meaningless without the Go context. The domain tag system is coarse: `["web", "api"]` matches most projects, so Colony B receives wisdom about React routing even though it is a CLI tool.

The multi-repo confidence boosting (2 repos = 0.70, 4+ = 0.95) assumes all confirming repos are relevant. If two unrelated repos both learn "prefer early returns," the confidence is boosted even though the pattern is trivially common.

**Consequences:**
- Workers receive irrelevant advice in their prompts, wasting context window tokens.
- Workers follow stale patterns that were correct for a different tech stack or architecture.
- Contradictory wisdom creates confusion: "prefer dependency injection" vs "avoid over-abstraction."
- Hive wisdom entries that were useful in Colony A become noise in Colony B. The 200-entry cap means useful Colony B wisdom gets evicted by irrelevant Colony A wisdom.

**Prevention:**
1. **Domain tag granularity.** Expand domain tags to include technology stack (e.g., `["go", "cli", "cobra"]`, not just `["api"]`). Colony B only receives wisdom tagged with at least one matching technology tag.
2. **Confidence decay based on relevance.** When Colony B reads wisdom from Colony A, apply a relevance discount. If the domain match is partial (1 of 3 tags match), reduce confidence by 20%. If no tags match, do not return the entry.
3. **Contextual validation at injection time.** Before injecting hive wisdom into a worker prompt, check whether the wisdom is compatible with the current repo's detected tech stack. The existing `skill-detect` patterns could be reused for this.
4. **Contradiction detection.** When two hive entries have similar text but opposite advice (e.g., "always use X" vs "never use X"), flag the contradiction and inject neither. This requires a similarity check (SHA-256 prefix collision or text embedding) at hive-write time.
5. **Colony B feedback loop.** When Colony B workers ignore or actively contradict hive wisdom, reduce that wisdom's confidence. The hive should learn from negative signals, not just positive ones.
6. **Separate hive namespaces.** Consider separating hive wisdom into "universal patterns" (coding best practices that apply everywhere) and "domain patterns" (tech-stack-specific advice). Only inject universal patterns by default; domain patterns require explicit opt-in.

**Detection:**
- `TestHiveWisdomFiltersByTechnologyTag` -- a test that adds wisdom tagged `["python", "django"]` and verifies it is not returned for a Go project.
- `TestHiveWisdomAppliesRelevanceDiscount` -- a test that verifies partial domain matches reduce confidence.
- `TestHiveWisdomDetectsContradictions` -- a test that stores "prefer X" and "avoid X" and verifies neither is returned.
- Integration test: Colony A learns "use goroutines for concurrency." Colony B (a Python project) starts and does NOT receive this wisdom.

**Which phase addresses it:** Phase 4 (hive wisdom reuse proof). This is the final feature and should only be attempted after the first three are stable.

---

### Pitfall 6: Go Finalizer Receives Workers It Never Manifested

**What goes wrong:** The TS host dispatches workers (either sub-workers from Pitfall 3, or re-dispatched workers from a retry), and includes them in the completion file. But the Go finalizer validates completion files against the original manifest. Workers not in the manifest are rejected. The finalizer fails, and the colony state is not updated.

**Why it happens:** The Go `build-finalize` command validates that every worker in the completion file corresponds to a dispatch in the manifest. This is by design: it prevents the TS host from inventing workers (boundary contract rule #3). But when the TS host retries a worker or dispatches a sub-worker, the new worker has a different name or caste than the original manifest dispatch. The finalizer rejects it.

**Consequences:**
- Retries fail silently: the completion file is rejected, the build is not finalized, and the colony remains stuck on the active phase.
- Sub-workers are wasted: they complete their work, but the finalizer ignores their results because they were not in the manifest.
- The user sees "Build completed" from the TS host but `aether status` still shows the old phase.

**Prevention:**
1. **Retries use the original manifest name.** When retrying a failed worker, the TS host must use the same `name`, `caste`, `task_id`, and `wave` from the original manifest dispatch. The retry result replaces the original dispatch result, not adds a new one.
2. **Sub-workers require a new manifest.** The TS host must call `aether build --plan-only --spawn-requests` (a new Go command) to get a manifest that includes sub-workers. The completion file is then validated against this new manifest.
3. **Completion file includes manifest identity.** The completion file must include the `manifest_id` (a hash of the original manifest) so the finalizer can verify the completion matches the manifest. If the TS host uses a new manifest for sub-workers, the completion file must reference that manifest.
4. **Finalizer error surface.** When the finalizer rejects a completion file, it must return a structured error (JSON with `rejected_workers: [...]` and `reason: "not in manifest"`) so the TS host can handle it programmatically.

**Detection:**
- `TestFinalizerRejectsUnknownWorkers` -- a Go test that sends a completion file with workers not in the manifest and verifies rejection.
- `TestRetryUsesOriginalManifestName` -- a TS test that verifies retry dispatch uses the same name as the original.
- `TestSubWorkerManifestFlow` -- an integration test that verifies the TS host requests a new manifest for sub-workers.

**Which phase addresses it:** Phase 1 (production dispatch) for the retry case. Phase 2 (worker-to-worker spawning) for the sub-worker case.

---

## Moderate Pitfalls

### Pitfall 7: Parallel Wave Dispatch Creates File Conflicts

**What goes wrong:** The wave orchestrator dispatches multiple workers in parallel within a wave (via `Promise.all`). Two workers modify the same file simultaneously. The second write overwrites the first worker's changes. Tests pass individually but fail in combination.

**Why it happens:** Go's manifest groups workers into waves and execution waves, but it does not guarantee non-overlapping file access. Two builders in the same wave might both modify `src/index.ts`. The `Promise.all` in `wave-orchestrator.ts` (line 154) runs them concurrently with no file-level conflict detection.

**Consequences:**
- Silent data loss: one worker's changes are overwritten by another.
- Flaky tests: tests pass when run in isolation but fail when run in parallel (wave order varies).
- The Builder-Probe Lock prevents self-certification but does not prevent file conflicts between parallel builders.

**Prevention:**
1. **File access declarations in manifest.** Go manifest should include `files_touched: string[]` per dispatch (a prediction of which files the worker will modify). TS host checks for overlaps before parallel dispatch.
2. **Fall back to sequential on conflict.** If two workers in the same wave declare overlapping files, dispatch them sequentially instead of in parallel.
3. **Worker handoff awareness.** The existing handoff system (`handoff_section` in dispatches) could include file change awareness. If Worker A's handoff says "I modified X.ts," Worker B should know before starting.
4. **Post-wave conflict detection.** After a wave completes, run `git diff --check` to detect merge markers or whitespace conflicts. If conflicts are found, re-dispatch affected workers sequentially.

**Detection:**
- Integration test with two workers modifying the same file. Verify the second worker's changes are preserved or the conflict is detected.
- `TestWaveDispatchDetectsFileConflicts` -- a test that provides overlapping file claims and verifies sequential fallback.

**Which phase addresses it:** Phase 1 (production dispatch). File conflicts are most likely in the first real build.

---

### Pitfall 8: Completion File Temp Directory Leaks

**What goes wrong:** The `writeCompletionFile` function in `go-bridge.ts` creates a unique temp directory per completion file (`mkdtempSync`). These temp directories are never cleaned up. After 100+ builds, the temp directory accumulates hundreds of stale JSON files.

**Why it happens:** The function creates a temp directory, writes the completion file, and returns the path. It never deletes the directory. Go reads the file and finalizes, but does not clean up the temp directory either. The TS host does not track temp directories for cleanup.

**Consequences:**
- Disk space leak: each completion file is small (few KB), but the temp directories accumulate.
- Security: completion files contain worker summaries, file paths, and potentially diagnostic information. Old files in `/tmp` are readable by other users on shared machines.
- Debugging confusion: stale completion files in `/tmp` can be confused with current ones.

**Prevention:**
1. **Go finalizer cleans up.** After reading the completion file, Go should delete it (and its parent temp directory). Add cleanup to `build-finalize`, `plan-finalize`, and `continue-finalize`.
2. **TS host cleans up on finalize success.** After `callGoJSON` returns from a finalizer, the TS host should delete the completion file and its temp directory.
3. **Temp directory prefix.** Use a consistent prefix (`aether-completions-`) so stale directories can be found and cleaned with a single command.
4. **Startup cleanup.** On TS host start, scan for temp directories with the `aether-` prefix that are older than 24 hours and delete them.

**Detection:**
- `ls /tmp/ | grep aether- | wc -l` -- count stale temp directories.
- `TestCompletionFileCleanup` -- a test that verifies temp directories are deleted after finalization.

**Which phase addresses it:** Phase 1 (production dispatch). This is a hygiene fix that should be done alongside the simulation gate removal.

---

### Pitfall 9: Platform-Specific CLI Argument Drift

**What goes wrong:** The `buildArgs` function in `platform-dispatcher.ts` constructs CLI arguments for each platform. These arguments are hardcoded and version-specific. When Claude Code, OpenCode, or Codex update their CLI interfaces, the arguments break silently.

**Why it happens:**
- Claude Code uses `-p --output-format json --json-schema <schema> --agent <name> --permission-mode bypassPermissions`.
- OpenCode uses `run --agent <name> --format json`.
- Codex uses `--sandbox workspace-write --ask-for-approval never exec --json --ephemeral --output-schema <schema>`.
- These are all different, and there is no automated test that verifies they still work against the current platform versions.

**Consequences:**
- Workers fail to spawn with cryptic CLI errors ("unknown flag --output-format").
- Different behavior across platforms: Claude Code gets schema enforcement, Codex gets schema enforcement, but OpenCode does not (no schema flag).
- The `--permission-mode bypassPermissions` flag for Claude Code is a security concern: it disables all permission checks for worker subprocesses.

**Prevention:**
1. **Platform version detection.** Before dispatching, detect the installed version of each platform CLI. Log it. If the version is outside the tested range, warn.
2. **Platform smoke test in CI.** Add a CI step that runs `claude -p --output-format json --help` and `opencode run --help` and `codex --help` to verify the arguments still exist.
3. **Remove `bypassPermissions`.** This flag disables all Claude Code safety checks. Workers should use the default permission mode. If a worker needs file write access, it should be granted specifically.
4. **Argument schema.** Define a JSON schema for each platform's CLI arguments. Validate arguments against the schema before dispatch. This catches typos and deprecated flags.

**Detection:**
- CI smoke test that runs each platform's help command and checks for expected flags.
- `TestBuildArgsAgainstPlatformHelp` -- a test that constructs args and verifies they are accepted by the platform's `--help` output.

**Which phase addresses it:** Phase 1 (production dispatch). Platform argument compatibility is a prerequisite for any real dispatch.

---

### Pitfall 10: Escalation Recovery Actions Are Never Executed

**What goes wrong:** The Queen orchestrator generates `RecoveryAction[]` for failed workers (retry, peer_reassign, fixer_dispatch, escalate). These actions are formatted into a human-readable summary and written to stderr. But they are never actually executed. The build completes with failures, and the recovery actions are just informational text.

**Why it happens:** The `handleWaveFailures` function in `escalation.ts` returns `RecoveryAction[]`. The orchestrator formats them with `formatRecoveryActions` and writes to stderr. But nobody reads stderr during automated builds, and the orchestrator does not act on the actions. The recovery actions are a suggestion, not a command.

**Consequences:**
- Failed workers are never retried or reassigned. The build reports partial success, and the user must manually investigate.
- The escalation chain (worker retry -> reassignment -> Queen -> user) exists in the type system but not in the execution path.
- Users see "Recovery actions (3)" in the output but nothing happens.

**Prevention:**
1. **Execute recovery actions, don't just report them.** The Queen orchestrator should have an `executeRecovery` method that acts on `RecoveryAction[]`:
   - `retry`: re-dispatch the failed worker (up to the retry limit).
   - `peer_reassign`: request a new manifest from Go with a different caste for the same task.
   - `fixer_dispatch`: request a new manifest with a Fixer caste worker.
   - `escalate`: stop the build and report to the user.
2. **Recovery within the wave, not after.** Recovery actions should be executed within the current wave, before moving to the next wave. Currently, failures are collected after all waves complete.
3. **Recovery budget.** Total recovery attempts across all actions must be bounded (e.g., 5 total). This prevents the infinite loop from Pitfall 9 of the previous PITFALLS.md.
4. **Recovery results in the completion file.** If a recovery action succeeds, the completion file includes both the original failure and the recovery success.

**Detection:**
- `TestRecoveryActionsAreExecuted` -- a test that provides a failed worker and verifies the retry recovery action is dispatched.
- `TestRecoveryBudgetIsRespected` -- a test that provides multiple failures and verifies recovery stops after the budget.

**Which phase addresses it:** Phase 1 (production dispatch). Recovery is part of the basic dispatch loop.

---

## Minor Pitfalls

### Pitfall 11: Worker Prompt Size Exceeds Platform Limits

**What goes wrong:** The `assemblePrompt` function in `prompt-assembler.ts` assembles a worker prompt from multiple sections: task, context capsule, handoff, skills, pheromones. For complex colonies, the combined prompt can exceed the platform's input token limit. The worker receives a truncated prompt and produces incorrect results.

**Prevention:** Add a prompt size check before dispatch. If the prompt exceeds a configurable limit (default: 100K characters), trim the lowest-priority sections (handoff, then skills, then pheromones) until it fits. Log the trimming.

**Which phase:** Phase 1 (production dispatch). Prompt assembly is exercised on every real dispatch.

---

### Pitfall 12: Oracle Confidence Target Set Too High by Default

**What goes wrong:** The Oracle lifecycle defaults to a confidence target that is never achievable within `max_iterations`. The loop runs to max iterations, wasting tokens, and reports "max_iterations_met" instead of "confidence_met."

**Prevention:** Set a reasonable default confidence target (e.g., 80%). Allow the user to override it. If the first iteration returns less than 50% confidence, warn that the target may not be achievable.

**Which phase:** Phase 3 (confidence-driven iteration).

---

### Pitfall 13: Hive Wisdom JSON File Corruption Under Concurrent Access

**What goes wrong:** Two colonies on the same machine both promote instincts to the hive simultaneously. Both read `wisdom.json`, both append entries, both write back. One write overwrites the other. Entries are lost.

**Why it happens:** The `HiveStore.saveWisdom` method uses `os.WriteFile` (line 76) which is not atomic. The method does not use file locking. Two concurrent writes can corrupt the JSON.

**Prevention:** Use the existing `pkg/storage` file locking (already used by colony state). Wrap `saveWisdom` in a file lock. Alternatively, use atomic write (write to temp file, rename). The hub-level file locking is mentioned in CLAUDE.md but may not be implemented for hive writes.

**Detection:** `TestHiveStoreConcurrentWrites` -- a test that writes to the hive from two goroutines simultaneously and verifies no data loss.

**Which phase:** Phase 4 (hive wisdom reuse proof). Hive writes happen during seal, which is after builds complete.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Simulation gate removal | Simulation guard blocks real dispatch (Pitfall 1) | Remove gate first, add mock-platform integration test |
| Phase 1: Real platform dispatch | Claims parsing fails on real output (Pitfall 2) | Pre-parse classification, diagnostic artifacts |
| Phase 1: Real platform dispatch | Platform CLI argument drift (Pitfall 9) | Version detection, CI smoke tests |
| Phase 1: Real platform dispatch | Recovery actions not executed (Pitfall 10) | Add executeRecovery to Queen orchestrator |
| Phase 1: Real platform dispatch | Completion file temp leaks (Pitfall 8) | Cleanup after finalization |
| Phase 2: Worker-to-worker spawning | Unbounded fan-out (Pitfall 3) | Max depth 2, Go manifest for sub-workers |
| Phase 2: Worker-to-worker spawning | Finalizer rejects unknown workers (Pitfall 6) | New manifest for sub-workers |
| Phase 3: Confidence iteration | Loop never converges (Pitfall 4) | Hard ceiling, diminishing returns detection |
| Phase 3: Confidence iteration | Default target too high (Pitfall 12) | Reasonable default (80%) |
| Phase 4: Hive wisdom reuse | Stale/irrelevant wisdom (Pitfall 5) | Tech tag filtering, relevance discount |
| Phase 4: Hive wisdom reuse | Concurrent write corruption (Pitfall 13) | File locking on hive writes |

---

## Anti-Patterns Specific to Production Orchestration

| Anti-Pattern | What It Looks Like | Why It's Wrong | What To Do Instead |
|-------------|-------------------|----------------|-------------------|
| **Shipping simulation guards** | `if (simulateWorkers !== true) return error` | Blocks production use; duplicated guards hide real code paths | Remove all simulation guards; add `--simulate` opt-in flag |
| **Ignoring spawns field** | Worker returns `spawns: ["watcher-42"]`, TS host discards it | Workers cannot request verification; breaks the sub-worker contract | Route spawns through Go manifest for budget enforcement |
| **Confidence from worker self-report** | Worker claims `confidence: 95%` and loop accepts it | Workers have no incentive to be honest; confidence is unverifiable | Confidence must come from Go (test results, coverage, gates) |
| **Hive wisdom firehose** | Inject all 200 hive entries into every worker prompt | Wastes context tokens; irrelevant advice drowns useful patterns | Filter by domain tags, limit to top 5 entries by relevance score |
| **Recovery as suggestion** | `formatRecoveryActions` writes to stderr but nothing executes | Failed workers stay failed; escalation chain is decorative | Queen orchestrator must execute recovery actions |
| **Parallel workers, same files** | Two builders in wave 1 both modify `index.ts` | Second write overwrites first; silent data loss | Detect file overlaps in manifest, fall back to sequential |
| **Hardcoded platform arguments** | `["-p", "--output-format", "json", "--agent", name]` | Breaks on platform version changes; no validation | Detect platform version, validate arguments, CI smoke tests |
| **Completion file /tmp leaks** | `mkdtempSync` creates dir, nobody cleans it up | Disk space leak, security exposure | Go finalizer or TS host cleans up after successful finalize |

---

## Sources

- `.aether/ts-host/src/lifecycle.ts` lines 268-275 (simulation guard), line 424 (hardcoded simulate=true)
- `.aether/ts-host/src/worker-dispatch.ts` (real dispatch path at line 237, simulation path at line 163)
- `.aether/ts-host/src/claims-parser.ts` (three-strategy parsing, spawns field handling)
- `.aether/ts-host/src/platform-dispatcher.ts` (platform-specific CLI args, buildArgs function)
- `.aether/ts-host/src/wave-orchestrator.ts` (parallel dispatch via Promise.all at line 154)
- `.aether/ts-host/src/queen/escalation.ts` (recovery action generation, failure classification)
- `.aether/ts-host/src/queen/orchestrator.ts` (Queen build execution, recovery action formatting)
- `.aether/ts-host/src/oracle-lifecycle.ts` (confidence loop, stop conditions, iteration state)
- `.aether/ts-host/src/go-bridge.ts` (completion file temp directory, boundary enforcement)
- `.aether/ts-host/src/boundary-reference.ts` (GO_OWNED_PATHS, boundary violation detection)
- `cmd/queen_spawn_budget.go` (spawn budget calculation, max workers per flow type)
- `pkg/learn/hive_store.go` (hive wisdom storage, abstraction, LRU eviction, concurrent write risk)
- `.aether/references/contracts/runtime-boundary-contract.md` (anti-patterns, ownership rules)
- `.aether/docs/known-issues.md` (provider availability preflight, post-launch parse failures)
- `.claude/agents/ant/aether-queen.md` line 302 (Claude Code subagent spawning limitation)
- `CLAUDE.md` (hive brain 200-entry cap, confidence boosting tiers, pheromone system)
- `CHANGELOG.md` lines 545, 674, 710 (multi-agent emergence, nested spawn, worker spec consolidation)
