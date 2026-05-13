# Domain Pitfalls: v1.17 Classic Restoration (Hybrid Go+TS)

**Domain:** Aether colony framework — restoring v5.4 Classic ceremony, swarm, workflow patterns, and Oracle richness to a hybrid Go runtime + TypeScript orchestration host architecture.
**Researched:** 2026-05-13
**Sources:** Runtime boundary contract, migration map, ceremony revival handoff, wrapper-runtime UX contract, state contract design, codex gap map, observable output contract, phase 5 lifecycle parity doc, known issues, PROJECT.md, MILESTONES.md, structural learning stack, pheromone propagation design, midden collection design, command playbooks (build-wave, build-verify, continue-verify), caste system, QUEEN system, error codes, context continuity plan, source-of-truth map, hybrid runtime strategy research.

---

## Critical Pitfalls

Mistakes that cause rewrites, state corruption, or major behavioral divergence between platforms.

### Pitfall 1: TS Host Writes State Directly (The Frankenstein Regression)
**What goes wrong:** The TypeScript orchestration host writes to `.aether/data/COLONY_STATE.json`, `pheromones.json`, `session.json`, or other branch-local state files directly, bypassing Go finalizers. This recreates the exact state corruption bug from the shell-script era: two writers with different locking models, divergent JSON schemas, and no provenance validation.
**Why it happens:** TS host developers see a JSON file, know its schema, and think "I'll just append a field." The boundary contract says Go owns state, but TS code has `fs.writeFileSync` available and no runtime enforcement.
**Consequences:**
- Frankenstein state: Go reads a file TS partially wrote, sees fields it didn't create, crashes or misinterprets.
- File lock races: Go uses `pkg/storage` locking; TS does not. Concurrent writes corrupt JSON.
- Provenance bypass: Finalizers validate manifest identity before writes. Direct TS writes skip this, allowing stale manifests to poison state.
- Regression of v1.0-era shell state corruption (documented in MEMORY.md as "state corruption bug — LLM reconstructs full JSON causing Frankenstein state; fix with state-mutate").
**Prevention:**
1. **Static enforcement:** Add an ESLint/custom lint rule in `.aether/ts-host/` that bans imports of `fs` modules for paths matching `.aether/data/**`.
2. **Runtime enforcement:** The `callGoJSON` bridge must reject any TS-side path that resolves inside `.aether/data/`. Add `assertNoDirectDataWrites` to every TS host test.
3. **Contract test:** A Go test that greps the compiled TS host bundle for `.aether/data` string literals and fails if any exist outside of Go CLI argument construction.
4. **Manifest-only pattern:** TS host may only pass `--completion-file` paths to Go finalizers. The completion file itself must live outside `.aether/data/` (e.g., in `os.tmpdir()`).
**Detection:**
- `git grep "writeFileSync.*\.aether/data" .aether/ts-host/`
- Go test `TestNoTSDirectStateWrites` (boundary enforcement test from v1.16)
- Runtime: Go finalizer rejects completion files inside `.aether/data/` with `E_VALIDATION_FAILED`
**Which phase addresses it:** Phase A-1 (Go Oracle iteration commands) and Phase C-3 (end-to-end parity verification). Must be enforced before any TS host feature ships.

---

### Pitfall 2: Duplicate Orchestration Logic in Go and TS
**What goes wrong:** Both Go and TS host contain logic for "which workers run in which wave," "when to advance phase," or "how to handle blockers." The two implementations drift. A bug fix in Go is not reflected in TS. Users see different behavior depending on whether they run `aether build` (Go path) or `/ant-build` (TS host path).
**Why it happens:**
- TS host needs to know wave grouping to dispatch workers sequentially.
- Go needs to know wave grouping to generate the manifest.
- Instead of Go being the sole authority, TS re-derives or hard-codes wave logic.
- Classic v5.4 had this problem in reverse: Bash wrappers owned orchestration, Go runtime was a passive executor.
**Consequences:**
- Platform divergence: Claude Code (TS host) and Codex (Go native) produce different worker mixes.
- Silent regressions: A Go fix to wave allocation is ignored by TS host; users on Claude see the old buggy behavior.
- Maintenance burden: Every orchestration change must be made in two places.
- Violation of boundary contract rule #2: "TS host calls Go plan-only for manifests."
**Prevention:**
1. **Single source of truth:** Go generates the full `execution_plan` and `execution_wave` in the manifest. TS host dispatches workers in the exact order and grouping specified by the manifest. TS host must not re-group, re-order, or filter workers.
2. **No TS-side wave logic:** The `dispatchWorkers()` function in TS must iterate the manifest's `dispatches` array in order, spawning each worker when its `execution_wave` number increments. Wave boundaries are manifest metadata, not TS logic.
3. **Test parity:** Golden snapshot tests compare the sequence of `spawn-log` calls between Go-only `aether build --synthetic` and TS-host `runLifecycle()`. Any divergence fails CI.
4. **Explicit anti-pattern in code reviews:** Any PR adding wave logic to TS host must be rejected with reference to this pitfall.
**Detection:**
- Diff between `dispatchWorkers()` in TS and `executeCodexBuildDispatches` in Go: they should not share logic, only data (the manifest).
- Golden test `TestBuildWaveSequenceParity` fails if spawn order diverges.
**Which phase addresses it:** Phase C-1 (colonize TS host integration) and Phase C-3 (parity verification). The build wave pattern from v1.16 is the reference.

---

### Pitfall 3: Ceremony Drift Across Platforms (Claude / OpenCode / Codex)
**What goes wrong:** The restored ceremony (banners, spawn notifications, seal rituals) is implemented differently in Claude Code wrappers, OpenCode wrappers, and Codex runtime. One platform gets rich art; another gets plain text. Users perceive Aether as inconsistent or broken on their platform.
**Why it happens:**
- Classic v5.4 ceremony was 100% wrapper-driven (Bash/Node). Each platform wrapper had its own copy of banner text, emoji maps, and stage separators.
- The v1.16 boundary contract moved visual rendering to Go (`cmd/codex_visuals.go`) to prevent drift.
- Restoring "wrapper-owned ceremony" risks reintroducing the old drift problem unless the ceremony config is shared.
**Consequences:**
- User confusion: `/ant-build` looks different in Claude Code vs OpenCode vs Codex.
- Maintenance nightmare: Changing one banner requires edits in 3+ wrapper files plus Go visuals.
- Codex gets left behind: Codex has no markdown wrappers, so any ceremony not in Go runtime is invisible to Codex users.
- Violation of platform policy: "Primary platforms: Claude Code and OpenCode. Secondary: Codex. Keep Claude/OpenCode aligned first."
**Prevention:**
1. **Shared ceremony config in YAML:** `ceremony.yaml` defines caste emoji, color ANSI codes, label names, banner templates, and stage separator strings. Go runtime, TS host, and wrapper generators all consume this file.
2. **Go runtime emits structured events:** `ceremony.build.spawn`, `ceremony.build.wave.start`, etc. carry payload data (caste, name, task, wave number). Wrappers render from events, not from hard-coded text.
3. **Codex fallback:** Go `cmd/codex_visuals.go` renders the canonical visual output from the same YAML config. If TS host or wrappers fail, Codex still shows correct ceremony.
4. **Parity tests:** `TestCeremonyParityAcrossPlatforms` compares visual output strings for the same event across Claude, OpenCode, and Codex. Differences fail CI.
5. **No wrapper-owned banners:** Wrappers may add platform-specific framing (Queen persona narration), but the core ceremony elements (spawn lines, wave headers, completion marks) must come from runtime events or shared YAML.
**Detection:**
- Visual diff between `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` spawn ceremony sections.
- Codex output missing caste emoji or stage separators.
- `TestPlatformDocHygiene` fails if wrapper markdown contains hard-coded ANSI sequences.
**Which phase addresses it:** Phase B-1 (TS host swarm display) and Phase C-2 (seal TS host integration). Ceremony config YAML should be introduced in Phase B-1.

---

### Pitfall 4: Rewriting Instead of Restoring (Scope Creep)
**What goes wrong:** The team decides "while we're restoring Classic ceremony, let's also redesign the event bus, rewrite the caste system, and add a new plugin architecture." The milestone never ships. Classic features remain missing.
**Why it happens:**
- The v5.4 Classic codebase is available as a reference. It is tempting to "improve" it while porting.
- The hybrid architecture is new. Engineers want to "do it right" instead of "do it like Classic."
- The migration map (D-04) explicitly says "Migration only, no new features," but this is hard to enforce without discipline.
**Consequences:**
- Milestone v1.17 misses its ship date.
- Users continue to lack ceremony, swarm display, and Oracle richness.
- The Go runtime accumulates technical debt from half-finished redesigns.
- Loss of the Classic behavior baseline: if Classic is rewritten, there is no stable reference to prove parity against.
**Prevention:**
1. **Classic baseline is read-only:** v5.4.0 is identified, smoke-tested, and used as a behavior baseline. No changes to the baseline.
2. **Golden tests lock behavior:** Snapshot/golden tests capture Classic output. Any deviation from the golden file must be explicitly approved as an intentional change, not a "better" implementation.
3. **Feature freeze checklist:** Every phase plan must include: "Does this change introduce new behavior not present in v5.4? If yes, defer to v1.18."
4. **Migration map compliance:** Each requirement (ORA-01..ORA-08, SWA-01..SWA-05, PAR-01..PAR-07) maps to a Classic behavior. If a requirement cannot be traced to Classic, it is out of scope.
5. **Queen review gate:** Before any phase begins, the Queen (or human reviewer) checks the phase plan against the Classic baseline checklist from v1.16.
**Detection:**
- Phase plans mentioning "redesign," "refactor," "new architecture," or "plugin system."
- Golden snapshot tests failing with large diffs that are not bug fixes.
- Increasing line counts in Go or TS without corresponding Classic behavior being restored.
**Which phase addresses it:** All phases. The v1.16 Classic baseline checklist (16-module behavioral checklist) is the guardrail.

---

### Pitfall 5: Race Conditions in Hybrid Event Streaming
**What goes wrong:** Go emits ceremony events to a file or pipe. TS host reads them. Events are lost, duplicated, or read out of order. The swarm display shows stale workers, missing completions, or flickering incorrect state.
**Why it happens:**
- Go writes events asynchronously during worker dispatch.
- TS host polls or streams the event source.
- No atomicity guarantee between event write and state finalization.
- Crash of either process leaves the event stream in an inconsistent state.
**Consequences:**
- Swarm display shows "active" workers that finished minutes ago.
- Ceremony misses spawn-complete events, making workers appear stuck.
- Duplicate events cause the narrator to render the same worker twice.
- On crash recovery, the event stream cannot be replayed accurately.
**Prevention:**
1. **Event bus with TTL and replay:** Use the existing `pkg/events.Bus` (JSONL append-log with `event-publish` / `event-subscribe`). It already has file locking, TTL cleanup, and replay by timestamp.
2. **TS host subscribes, not polls:** `event-bus-subscribe --stream --filter ceremony.*` provides an NDJSON stream. TS host reads this stream sequentially. No polling means no race between read and write.
3. **Idempotent event handling:** TS host event consumer must handle duplicate event IDs gracefully (same `spawn_id` + `status` = no-op).
4. **Crash recovery:** On TS host restart, replay events from the last known timestamp using `event-replay`. Rebuild the in-memory activity frame from the log.
5. **No in-memory-only state:** The TS host's activity frame is a cache. The event bus JSONL is the source of truth.
**Detection:**
- Swarm display shows worker in "active" section after `spawn-complete` was emitted.
- `event-bus.jsonl` contains gaps in sequence numbers.
- Integration test `TestEventStreamNoLossUnderLoad` fails.
**Which phase addresses it:** Phase B-1 (TS host swarm display) and Phase B-2 (swarm integration tests). The event bus already exists; TS host must use it correctly.

---

### Pitfall 6: Animated Terminal Dashboard Breaks in Non-TTY Environments
**What goes wrong:** The restored swarm display uses ANSI clear sequences, cursor positioning, or live redraw to create an animated dashboard. In CI, non-interactive shells, or platform wrappers that capture output as markdown, the display produces garbled text, invisible output, or broken markdown.
**Why it happens:**
- Classic v5.4 swarm was a 277-line animated Bash dashboard using `tput`, `clear`, and cursor movement.
- Modern platforms (Claude Code, OpenCode) render assistant output as markdown, not raw terminal streams.
- Codex runs in a non-TTY environment where ANSI sequences are meaningless.
**Consequences:**
- CI logs filled with ANSI escape sequences.
- Claude Code output shows raw `[CEREMONY]` lines mixed with JSON.
- Users in non-interactive mode see blank or broken output.
- Violation of known issue: "Visual output depends on terminal mode."
**Prevention:**
1. **Three output modes:**
   - `AETHER_OUTPUT_MODE=json`: Machine-only, no ceremony text in stdout.
   - `AETHER_OUTPUT_MODE=visual`: TTY-only, full ANSI animation allowed.
   - `AETHER_OUTPUT_MODE=markdown` (or default non-TTY): Plain text ceremony lines, no ANSI, suitable for markdown rendering.
2. **Go runtime detects TTY:** `cmd/codex_visuals.go` already handles terminal capability detection. TS host must respect this and downgrade to plain text when not a TTY.
3. **No ANSI in markdown wrappers:** Wrappers must never emit raw ANSI sequences. They render event payloads as markdown (e.g., `**Builder Hammer-42:** Task complete`).
4. **Debounced redraw:** If animation is used, redraw only when data changes. Skip redraw if output is piped.
**Detection:**
- `AETHER_OUTPUT_MODE=json aether build 1 --plan-only` contains `[CEREMONY]` or ANSI in stdout.
- Claude Code output shows raw escape sequences.
- `TestNarratorLauncherAutoSkipsJSONMode` fails.
**Which phase addresses it:** Phase B-1 (TS host swarm display). The narrator foundation from v1.6 already has TTY detection and JSON protection; TS host must inherit these rules.

---

### Pitfall 7: Builder-Probe Lock Bypassed by TS Host
**What goes wrong:** The TS host marks tasks as `completed` without waiting for Probe verification. Builders self-certify, violating the Builder-Probe Lock that was a core Classic safety invariant.
**Why it happens:**
- TS host orchestrates worker dispatch and processes worker results.
- A builder returns `code_written`. The TS host, eager to advance, treats this as "done" and writes a completion file with `status: "completed"`.
- The Go finalizer trusts the completion file and advances state.
- Probe never runs because the TS host skipped it.
**Consequences:**
- Untested code is marked complete.
- Colony advances on false claims.
- Verification loop is bypassed.
- Regression of v1.2-era "worker dispatch honesty" fixes.
**Prevention:**
1. **Manifest-driven verification:** The Go build manifest includes `execution_plan` with Probe as a post-wave specialist. TS host must dispatch Probe from the manifest, not skip it.
2. **Status translation layer:** TS host must not translate `code_written` to `completed`. It passes `code_written` through to the completion file. Only Go finalizer (or Probe result) may upgrade to `completed`.
3. **Completion file validation:** Go `build-finalize` rejects completion files where any dispatch has `status: "completed"` without a matching Probe `status: "passed"` in the same completion file.
4. **Explicit test:** `TestBuildFinalizeRejectsSelfCertifiedCompletion` fails if Probe is missing.
**Detection:**
- Build completion file contains `status: "completed"` for builder dispatches with no Probe dispatch.
- `TestBuildFinalizeRecordsExternalTaskResultsForContinue` fails.
**Which phase addresses it:** Phase C-1 (colonize TS host integration) and Phase C-3 (parity verification). The build wave playbook already defines the Builder-Probe Lock; TS host must obey it.

---

### Pitfall 8: Oracle RALF Loop Loses State Between Iterations
**What goes wrong:** The TS host drives the Oracle RALF iteration loop. Between iterations, the TS host process crashes or is restarted. The Oracle loses its iteration count, confidence score, and accumulated findings. The next iteration starts from scratch.
**Why it happens:**
- TS host maintains loop state in memory (current iteration, confidence, stop conditions).
- Go owns the Oracle workspace files (`oracle-state.json`, `oracle-plan.json`), but TS host does not re-read them at loop start.
- The boundary contract (D-05) says "TS host orchestrates timing and worker dispatch; Go handles all question selection, confidence calculation, and state writes." If TS host also caches state, it becomes a second source of truth.
**Consequences:**
- Oracle runs indefinitely: iteration 1 completes, TS host restarts, iteration 1 runs again.
- Confidence never reaches target because findings are lost.
- User sees "Oracle is thinking..." for hours with no progress.
- Violation of Oracle loop fix from v1.10: "Oracle loop fix with research formulation, depth selection, and state persistence."
**Prevention:**
1. **Stateless TS host loop:** `runOracleLifecycle()` in TS must not cache Oracle state. At each iteration, it calls `oracle-iterate --plan-only`, which returns the full current state (iteration count, confidence, next question). TS host uses this as the sole input for dispatch decisions.
2. **Go finalizer is the only writer:** `oracle-iterate-finalize` commits iteration results. TS host never writes to `.aether/data/oracle/`.
3. **Stop conditions in manifest:** The `--plan-only` manifest includes `confidence_target`, `max_iterations`, and `current_confidence`. TS host evaluates stop conditions from the manifest, not from local variables.
4. **Resume from Go state:** If TS host restarts, it calls `oracle-iterate --plan-only` to discover where the loop is. No resume file in TS.
**Detection:**
- Oracle runs more iterations than `max_iterations`.
- `oracle-state.json` iteration count lags behind actual iterations.
- Integration test `TestOracleStatelessLoopResumesFromGo` fails.
**Which phase addresses it:** Phase A-2 (TS host Oracle lifecycle) and Phase A-3 (Oracle integration tests).

---

### Pitfall 9: Tiered Escalation Chain Creates Infinite Loops
**What goes wrong:** The restored tiered escalation chain (worker retry -> reassignment -> Queen -> user) loops forever. A failing worker is retried, reassigned, retried again, reassigned again — never reaching the user escalation step.
**Why it happens:**
- TS host implements retry logic with a counter.
- The counter is reset when the worker is reassigned to a different caste.
- The new caste also fails, triggering retry again.
- No global "this task has been escalated" flag exists.
- Classic v5.4 had this problem: Bash retry logic did not track escalation state across castes.
**Consequences:**
- Build hangs: workers spawn, fail, respawn, fail, respawn...
- Token waste: each retry consumes API tokens.
- User is never asked: the colony appears stuck.
- Violation of v1.12 loop safety: "6 LOOP requirements covering watcher auto-skip, recovery redirect, circuit breaker, cycle detection."
**Prevention:**
1. **Escalation state in completion file:** Each dispatch in the completion file carries `escalation_level: 0|1|2|3` (0=initial, 1=retried, 2=reassigned, 3=escalated_to_user). TS host increments this level, not resets it.
2. **Go finalizer enforces ceiling:** `build-finalize` rejects completion files where any dispatch has `escalation_level > 3` without `status: "blocked"` or `"escalated"`.
3. **Circuit breaker:** After 2 retries at any level, TS host must either reassign (level 2) or escalate (level 3). No third retry at the same level.
4. **Midden threshold integration:** If a task fails 3 times total across all levels, auto-emit REDIRECT and mark blocked (existing MID-03 behavior).
**Detection:**
- Spawn log shows the same task spawned 5+ times.
- `TestEscalationChainTerminates` fails.
- Midden shows 3+ entries for the same task.
**Which phase addresses it:** Phase C-1 (colonize TS host integration) and Phase C-3 (parity verification). Escalation logic is part of the build wave playbook (Step 5.2 partial wave failure handling).

---

### Pitfall 10: Worktree Merge-Back Orphans Code
**What goes wrong:** TS host dispatches workers in worktree mode. Workers complete, but the TS host never calls `worktree-merge-back`. The worktree branch exists with valuable code, but it is never merged into main. The next wave builds on stale main, overwriting or conflicting with the orphaned work.
**Why it happens:**
- TS host implements wave dispatch but forgets the merge-back step between waves.
- The merge-back step is in the build playbook (Step 5.2.5) but is not part of the Go manifest. TS host must know to do it.
- Worktree mode is optional (`parallel_mode: worktree`). TS host may not test it.
**Consequences:**
- Wave 2 workers overwrite Wave 1 changes because they work on main, not the merged state.
- Git conflicts when worktrees are eventually cleaned up.
- Lost work: orphaned branches are garbage collected.
- Regression of MEMORY.md issue: "Worktree merge-back gap — Agents spawn worktrees but never auto-merge back to main, causing orphaned branches with valuable code."
**Prevention:**
1. **Manifest includes merge-back directive:** The build manifest includes `requires_merge_back: true` when `parallel_mode` is `worktree` and the wave is not the last. TS host checks this flag.
2. **TS host calls Go command:** `aether worktree-merge-back --branch {branch}` is called by TS host after processing wave results, before dispatching the next wave.
3. **Non-blocking but mandatory:** Merge-back failures create blockers but do not halt the build. However, skipping merge-back is not allowed.
4. **Test coverage:** Integration test `TestWorktreeMergeBackBetweenWaves` verifies that Wave 2 sees Wave 1 changes.
**Detection:**
- `.aether/data/worktree/` contains stale branches after build completes.
- Wave 2 workers do not see files created by Wave 1.
- `TestWorktreeMergeBackBetweenWaves` fails.
**Which phase addresses it:** Phase C-1 (colonize TS host integration) and Phase C-3 (parity verification). The build wave playbook already defines merge-back; TS host must implement it.

---

## Moderate Pitfalls

### Pitfall 11: Stale Manifest Reuse After Boundary Discussion
**What goes wrong:** The TS host requests a plan-only manifest, encounters `orchestrator_boundary_guidance` routing to `aether discuss`, resolves the discussion, then reuses the old manifest instead of requesting a fresh one. The stale manifest has outdated boundary questions or incorrect worker assignments.
**Why it happens:**
- TS host caches the manifest in a variable.
- After discuss, the developer forgets to clear the cache.
- The boundary contract explicitly says "request a fresh manifest after resolution."
**Prevention:**
1. **Manifest lifetime:** TS host must treat the manifest as valid only for the current turn. After any `aether discuss` call, the previous manifest is invalidated.
2. **Explicit fresh request:** The TS host code must have a comment: `// After discuss resolution, ALWAYS request a fresh manifest. Do not reuse.`
3. **Go finalizer rejects stale manifests:** Finalizers validate `generated_at` timestamp against a freshness window (e.g., 5 minutes). Stale manifests are rejected.
**Detection:**
- `TestFinalizeRejectsStaleManifest` fails.
- Wrapper test `TestWrapperRequestsFreshManifestAfterDiscuss` fails.
**Which phase addresses it:** Phase C-2 (seal TS host integration) and Phase C-3 (parity verification). The boundary guidance contract is already defined in Phase 5.

---

### Pitfall 12: Pheromone Injection Loses User Signals on Branch Switch
**What goes wrong:** User emits a REDIRECT signal. TS host runs on a feature branch. The signal is written to branch-local `pheromones.json`. When the user switches to main, the signal is gone. The constraint is violated on main.
**Why it happens:**
- Pheromones are branch-local by design (state contract Rule 4).
- User REDIRECTs should be colony-wide, but the current system scopes them to the branch.
- The pheromone propagation design doc defines injection/merge-back protocols, but they are not fully implemented.
**Prevention:**
1. **User signals go to hub:** `pheromone-write` with `source: "user"` should write to hub-global pheromone storage (or QUEEN.md) in addition to branch-local. This is a Go runtime change, not TS host.
2. **TS host reads hub signals:** Before dispatching workers, TS host calls `aether pheromone-read --hub` to get user signals and merges them with branch-local signals.
3. **Defer if complex:** If hub-global pheromones require significant Go changes, defer to v1.18. Document the limitation: "User signals are branch-local in v1.17."
**Detection:**
- User REDIRECT on feature branch is not visible on main.
- `TestPheromoneHubPropagation` (if exists) fails.
**Which phase addresses it:** Phase C-3 (parity verification) or deferred to v1.18. Not a blocker for v1.17 if documented.

---

### Pitfall 13: Codex Skill Drift from Wrapper Changes
**What goes wrong:** Claude/OpenCode wrappers are updated to use TS host orchestration, but the Codex lifecycle skill (`.aether/skills/colony/aether-colony-build-cycle/SKILL.md`) is not updated. Codex users see outdated guidance that contradicts the new runtime behavior.
**Why it happens:**
- Codex does not use wrapper markdown. It uses TOML agents and lifecycle skills.
- Changes to wrapper flow are not automatically reflected in Codex skills.
- The source-of-truth map lists 5 drift-sensitive artifacts; keeping them aligned is manual.
**Prevention:**
1. **Update all 5 artifacts together:** Any change to plan/build/continue orchestration must update:
   - `cmd/command_guide.go`
   - `.aether/commands/{plan,build,continue}.yaml`
   - `.claude/commands/ant/{plan,build,continue}.md`
   - `.opencode/commands/ant/{plan,build,continue}.md`
   - `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`
2. **Drift-guard tests:** `TestCodexLifecycleGuidesRequireVisibleWorkerActivity`, `TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity`, and `TestCodexLifecycleSkillMirrorsWorkerActivityContract` must remain green.
3. **Codex skill references command-guide:** The skill should delegate to `aether command-guide` for the latest orchestration contract, rather than hard-coding steps.
**Detection:**
- `TestCodexLifecycleSkillMirrorsWorkerActivityContract` fails.
- Codex output shows old command sequences (e.g., `aether build --synthetic` instead of `build-finalize`).
**Which phase addresses it:** All phases. The drift-guard tests from v1.6 are the enforcement mechanism.

---

### Pitfall 14: TS Host Placeholder Code Ships as Real
**What goes wrong:** The v1.16 TS host prototype contains simulated workers, synthetic completion files, and placeholder phase plans. If this code is not replaced before v1.17 ships, users get fake worker activity and no real work done.
**Why it happens:**
- The `runLifecycle()` function in `.aether/ts-host/src/lifecycle.ts` has:
  - `planningResults` marked as `completed` with synthetic summaries.
  - `dispatchWorkers` with `simulatedFileClaims`.
  - `continueResults` marked as `completed` without real reviewers.
- Prototype code is useful for integration tests but must not be the production path.
**Prevention:**
1. **Explicit PROTOTYPE markers:** All simulated code must be guarded by `if (process.env.AETHER_SIMULATION_MODE)` or similar. Default path must spawn real platform workers.
2. **Integration test vs production path:** The prototype lifecycle test uses simulation mode. The production path is tested with real (or mock) platform worker spawns.
3. **Pre-ship audit:** Before tagging v1.17, grep for `simulated`, `placeholder`, `synthetic`, `stub` in `.aether/ts-host/src/` and verify none are in the default code path.
**Detection:**
- `grep -n "simulated\|placeholder\|synthetic\|stub" .aether/ts-host/src/*.ts`
- Build produces no real file changes despite "success."
**Which phase addresses it:** Phase A-2 (TS host Oracle lifecycle) and Phase C-1 (colonize TS host integration). By Phase C-3, all simulation code must be removed or gated.

---

## Minor Pitfalls

### Pitfall 15: Golden Snapshot Tests Become Brittle
**What goes wrong:** Golden snapshot tests capture exact output strings. As the system evolves, minor formatting changes (whitespace, timestamp format, emoji) cause tests to fail, creating noise and encouraging developers to update snapshots without reviewing changes.
**Why it happens:**
- Snapshot tests are easy to write but hard to maintain.
- Classic v5.4 output format is the baseline, but the hybrid architecture may legitimately differ in non-behavioral ways (e.g., JSON field ordering).
**Prevention:**
1. **Structured golden tests:** Compare parsed JSON structures, not raw strings. For visual output, compare tokenized elements (spawn lines, wave headers) rather than exact ANSI sequences.
2. **Human review required:** Snapshot updates must be reviewed in PR. A bot or CI job should flag snapshot changes for human approval.
3. **Separate behavioral and cosmetic tests:** Behavioral tests verify "Probe runs after builders." Cosmetic tests verify "output contains caste emoji." Cosmetic tests are allowed to change; behavioral tests are not.
**Detection:**
- Snapshot tests fail on every unrelated change.
- Developers run `UPDATE_SNAPSHOTS=1` habitually.
**Which phase addresses it:** Phase C-3 (end-to-end parity verification). Golden tests are introduced in v1.16; v1.17 must keep them maintainable.

---

### Pitfall 16: Missing Node Falls Back Ungracefully
**What goes wrong:** The TS host requires Node.js. On a machine without Node, Aether commands fail with obscure errors instead of falling back to Go-only behavior.
**Why it happens:**
- TS host is spawned as a subprocess. If `node` is not found, the spawn fails.
- The v1.6 narrator had this problem and solved it with non-fatal fallback. TS host must do the same.
**Prevention:**
1. **Node optional:** `AETHER_NARRATOR=off` disables TS host features. Go runtime continues without TS host.
2. **Clear error message:** If TS host is required but Node is missing, print: "TypeScript orchestration host requires Node.js >= 18. Install Node or set AETHER_NARRATOR=off for Go-only mode."
3. **Graceful degradation:** If TS host fails to start, log the error and continue with Go-only path. Do not crash the command.
**Detection:**
- `TestNarratorLauncherMissingRuntimeDoesNotFail` (existing test pattern).
- Run Aether in a container without Node.
**Which phase addresses it:** Phase B-1 (TS host swarm display). The narrator launcher tests from v1.6 are the model.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| A-1: Go Oracle iteration commands | Oracle state mutation in `--plan-only` | Test `TestOraclePlanOnlyDoesNotMutateState` |
| A-2: TS host Oracle lifecycle | TS host caches Oracle state between iterations | Stateless loop: re-read from Go each iteration |
| A-3: Oracle integration tests | Simulation mode ships as default | Gate all simulated code behind env var |
| B-1: TS host swarm display | ANSI pollution in JSON mode | `AETHER_OUTPUT_MODE=json` strictly no ceremony text |
| B-2: Swarm integration tests | Polling causes race conditions | Use event bus subscription, not polling |
| C-1: Colonize TS host integration | Surveyor dispatch order diverges from manifest | Dispatch strictly from manifest `dispatches` array |
| C-2: Seal TS host integration | Seal finalizer called when blockers exist | TS host stops on blockers, does not call finalize |
| C-3: End-to-end parity | Golden tests become brittle | Structured comparison, not string snapshots |

---

## Anti-Patterns Specific to This Hybrid Model

| Anti-Pattern | What It Looks Like | Why It's Wrong | What To Do Instead |
|-------------|-------------------|----------------|-------------------|
| **TS Host State Authority** | TS host decides "advance phase" by writing `COLONY_STATE.json` | Bypasses Go finalizers, provenance, locking | TS host calls `aether continue-finalize --completion-file` |
| **Wrapper-Owned Recovery** | Wrapper markdown shows "Choose A/B/C" menu for failed gates | Runtime owns gating; wrappers must not invent recovery | Wrapper renders runtime's recovery template from `gate-recovery-template` |
| **Visual Parsing** | TS host scrapes `aether status` ANSI output to get current phase | Fragile, breaks on visual changes, non-TTY unsafe | Use `AETHER_OUTPUT_MODE=json aether status` |
| **Ceremony Hard-Coding** | Wrapper markdown contains `━━━ S P A W N   P L A N ━━━` | Drifts across platforms, Codex misses it | Ceremony comes from Go event payload or shared YAML |
| **Simulated Worker Dispatch** | `dispatchWorkers()` returns synthetic results without spawning | Ships fake behavior | Gate simulation behind env var; default path spawns real workers |
| **Go-Owned Orchestration** | Go `cmd/codex_build.go` spawns platform workers via `exec.Command` | Go cannot spawn Claude/OpenCode Task-tool agents | Go generates manifest; TS host/wrappers spawn platform workers |
| **TS-Owned Rendering** | TS host contains ANSI color codes and banner text | Duplicates Go visuals, drifts from YAML config | TS host delegates to Go `visuals-dump --json` or shared YAML |
| **Ignoring Boundary Guidance** | TS host spawns workers despite `orchestrator_boundary_guidance.active: true` | Violates Phase 5 contract, causes stale manifest issues | Stop, route to discuss, request fresh manifest |

---

## Sources

- `.aether/references/contracts/runtime-boundary-contract.md` — Anti-patterns #1-3, rules, failure signals
- `.aether/docs/migration-map.md` — Milestone ordering, boundary compliance, risk assessment
- `.aether/docs/ceremony-revival-v1.6-handoff.md` — Narrator launcher, build plan-only/finalize, wrapper restoration, ceremony gaps
- `.aether/docs/wrapper-runtime-ux-contract.md` — Wrapper anti-patterns, runtime surface, orchestrator boundary guidance
- `.aether/docs/codex-ant-workflow-gap-map.md` — P0/P1 gaps, stale manifest, plan-only mutation
- `.aether/docs/codex-observable-output-contract.md` — Blocked contract surface, temp file hygiene
- `.aether/docs/phase5-lifecycle-integration-parity.md` — Boundary guidance object, finalizer validation, wrapper parity
- `.aether/docs/state-contract-design.md` — Branch-local vs hub-global state, read rules, merge behavior
- `.aether/docs/known-issues.md` — Interrupted workers, visual output terminal dependency
- `.aether/docs/command-playbooks/build-wave.md` — Builder-Probe Lock, wave logic, worktree merge-back, escalation
- `.aether/docs/command-playbooks/build-verify.md` — Watcher spawn, verification loop
- `.aether/docs/command-playbooks/continue-verify.md` — Continue verification, gate recovery, claim verification
- `.aether/docs/pheromone-propagation-design.md` — Cross-branch signal propagation, injection/merge-back
- `.aether/docs/midden-collection-design.md` — Post-merge collection, cross-PR analysis
- `.aether/docs/structural-learning-stack.md` — Trust scoring, event bus, curation ants
- `CLAUDE.md` (project) — Platform policy, UX architecture, wrapper-runtime contract
- `MEMORY.md` — State corruption bug, worktree merge-back gap, worktree stale base
- `.aether/ts-host/src/lifecycle.ts` — TS host prototype, simulation code, boundary contract
