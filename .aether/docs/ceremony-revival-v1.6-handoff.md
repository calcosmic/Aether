# Ceremony Revival v1.6 Handoff

Last updated: 2026-04-24T05:20:37Z

Branch: `codex/ceremony-narrator-foundation-v16`
Remote branch: `origin/codex/ceremony-narrator-foundation-v16`
Open PR URL: `https://github.com/calcosmic/Aether/pull/new/codex/ceremony-narrator-foundation-v16`

## Purpose

This handoff preserves the exact implementation plan for finishing the v1.6
ceremony revival if the current session loses context.

The target architecture is:

- Go owns colony state, dispatch contracts, event persistence, and lifecycle
  safety.
- The bundled TypeScript narrator owns rich ceremony rendering.
- Claude Code and OpenCode wrappers own real Task-tool subagent spawning.
- Codex remains a direct CLI surface and must not claim Task-tool work happened
  inside Go.

Do not revert the Go runtime rewrite. Restore ceremony and real agent spawning
on top of the current runtime.

## Already Shipped On This Branch

These commits are pushed:

- `5b77a8dc feat: add ceremony narrator foundation`
- `c1880184 docs: add aether pipeline diagrams`
- `1fa1f95f fix: ship dependency-free ceremony narrator runtime`
- `19bd3d66 feat: let narrator consume visual metadata`
- `aa8d6d4b test: pipe event stream through narrator runtime`
- `e2882ff2 docs: add ceremony revival handoff`
- `f8d91afa feat: launch narrator for build ceremony events`
- `28b9e857 feat: render ceremony activity frames`
- `d7564ca6 test: cover multi-wave ceremony activity`
- `2dd4070c feat: add build plan-only manifest`
- `8929274e feat: finalize external build workers`
- `99a729d5 feat: restore build wrapper orchestration`
- `315d4ee8 feat: add continue plan-only manifest`
- `dfd7f6da feat: finalize external continue workers`
- `7c1d25f4 feat: restore continue wrapper orchestration`

Implemented foundation:

- `.aether/ts/` exists with strict TypeScript source and tests.
- `.aether/ts/dist/narrator.js` is committed as the dependency-free Node runtime.
- Install/update packaging syncs `.aether/ts` and excludes `node_modules`.
- CI/release/dependabot cover the narrator package and dist drift.
- Ceremony event topic and payload types live in `pkg/events/ceremony.go`.
- `aether event-bus-subscribe --stream --filter ceremony.*` streams persisted
  event-bus entries as NDJSON.
- `aether visuals-dump --json` exposes Go-owned caste emoji/color/label
  metadata.
- The narrator accepts Go visual metadata via `--visuals`.
- A Go smoke test pipes event-bus stream output through `dist/narrator.js`.
- Go can now auto-launch the narrator sidecar for build ceremony events.
- Build dispatch lifecycle events are persisted to `event-bus.jsonl` even in
  JSON mode.
- `AETHER_NARRATOR` launch gating is implemented with JSON mode protected from
  narrator text.
- The narrator now keeps an in-memory activity frame and renders worker
  sections for active, completed, blocked, and other workers.
- Temporary HOME smoke verified the source-built binary can launch the narrator
  from the installed hub fallback when the fixture repo has no local
  `.aether/ts` runtime.
- `aether build <phase> --plan-only` now prints a machine-readable
  `dispatch_manifest` without changing colony state, writing checkpoints,
  writing worker briefs, writing claims, or spawning workers.
- Plan-only dispatch entries include the intended caste, deterministic worker
  name, `agent_name`, wave/task metadata, and `planned` status so wrappers can
  spawn real Task-tool agents from JSON instead of scraping visual output.
- `aether build-finalize <phase> --completion-file <path|->` records externally
  spawned wrapper Task results as the build manifest and claims packet that
  `aether continue` already trusts.
- Claude/OpenCode build wrappers now spawn real manifest-driven Task/subagents
  and finalize through `build-finalize`.
- `aether continue --plan-only` emits a read-only `continue_manifest` for
  wrapper-spawned Watcher/Gatekeeper/Auditor/Probe review.
- `aether continue-finalize --completion-file <path|->` merges wrapper review
  results, re-runs deterministic verification, writes reports, and advances or
  blocks through runtime gates.
- Claude/OpenCode continue wrappers now spawn real manifest-driven review
  agents and finalize through `continue-finalize`.
- `aether plan --plan-only --depth <fast|balanced|deep|exhaustive>` now emits a
  read-only `plan_manifest` / `planning_manifest` for wrapper-spawned Scout and
  Route-Setter agents.

Verification already passed for the pushed foundation:

- `npm --prefix .aether/ts ci`
- `npm --prefix .aether/ts audit --package-lock-only --audit-level=low`
- `npm --prefix .aether/ts run build`
- `git diff --exit-code -- .aether/ts/dist/narrator.js`
- `npm --prefix .aether/ts run typecheck`
- `npm --prefix .aether/ts test`
- narrator smoke with `--visuals`
- focused Go tests for ceremony/event-bus/visuals
- `go test ./... -count=1 -timeout 300s`
- `go test ./... -race -count=1 -timeout 600s`
- `go vet ./...`
- `git diff --check`

## Specialist Review Synthesis

Scout mapped the build dispatch path:

- CLI entry is `cmd/codex_workflow_cmds.go`; `buildCmd` calls
  `runCodexBuild`.
- Build orchestration is `cmd/codex_build.go`; it plans dispatches, writes
  artifacts, records spawn-tree entries, emits the preview, then calls
  `executeCodexBuildDispatches`.
- Runtime execution is custom in `cmd/codex_build_worktree.go` because worktree
  allocation, sync, and claim collection live there.
- Progress helpers are centralized in `cmd/codex_build_progress.go`.
- Do not route build through the generic `pkg/codex.DispatchBatchWithObserver`
  yet; that would be a broader refactor.

Watcher identified launcher tests:

- Cover `AETHER_NARRATOR=off`, `auto`, `on`, JSON mode, missing Node, missing
  runtime, early runtime exit, child cleanup, event persistence, and command
  output not being polluted.
- Add injection seams for `exec.LookPath`, runtime path resolution, and process
  start so tests are not brittle.
- Avoid writing child stdout directly to the shared `stdout` writer because
  tests often replace it with `bytes.Buffer`.

Gatekeeper guardrails:

- Use `exec.CommandContext` with absolute `node` and absolute
  `dist/narrator.js` paths.
- Never use shell, `npm`, `npx`, package scripts, or `narrator.ts` at runtime.
- Missing Node or missing runtime must be non-fatal.
- Pipe child stdout back through Go and write using the existing visual output
  mutex path.
- Add length caps/truncation for event fields before rendering or forwarding
  large payloads.
- Keep `node_modules` out of install/update/release artifacts.

Existing non-narrator dependency advisories were noted by Gatekeeper but not
changed in this slice. They should be handled as a separate release-hardening
task, not mixed into the launcher.

## Completed Slice: Build Plan-Only Manifest

Purpose:

- Give Claude Code and OpenCode wrappers a safe machine-readable dispatch
  contract before restoring real Task-tool spawning.
- Keep Go authoritative for phase/task/wave planning while keeping wrapper
  execution outside the Go binary.
- Avoid the unsafe fallback of parsing the visual spawn plan.

Implemented behavior:

- `aether build <phase> --plan-only` validates the requested phase, task filter,
  critical pre-build gates, and build order using the same checks as a real
  build.
- The command returns JSON with `plan_only: true`, `dispatch_mode: "plan-only"`,
  top-level `dispatches`, and a structured `dispatch_manifest`.
- The manifest includes phase metadata, root, colony depth, parallel mode, wave
  execution strategy, playbooks, task plans, success criteria, selected tasks,
  and planned dispatches.
- Dispatch maps include `agent_name` values such as `aether-builder` and
  `aether-watcher` for wrapper Task-tool routing.
- The command does not call the worker invoker, does not publish ceremony
  events, does not update `COLONY_STATE.json`, does not update session context,
  and does not write build/checkpoint/claim artifacts.

Focused verification:

- `go test ./cmd -run 'TestBuildPlanOnly|TestBuildWritesDispatchArtifactsAndUpdatesState|TestBuildSupportsTaskScopedRedispatch' -count=1`

Important limitation:

- This is only the planning half of the wrapper bridge unless paired with
  `aether build-finalize`. Do not make wrappers call a fake `aether build
  --synthetic` after real Task work; that would overwrite the evidence trail
  with simulated dispatch.

## Completed Slice: Build Finalize For Wrapper Agents

Architect review confirmed the missing contract:

- `spawn-log` and `spawn-complete` are visibility only.
- `aether continue` trusts `.aether/data/build/phase-N/manifest.json` and
  `.aether/data/last-build-claims.json`.
- Therefore wrappers need one Go-owned finalization packet after real Task-tool
  execution, not fake `aether build` execution.

Implemented behavior:

- New command: `aether build-finalize <phase> --completion-file <path|->`.
- Completion JSON must include the original `dispatch_manifest` from
  `aether build --plan-only` plus terminal worker results.
- The finalizer validates phase/state/task filters, manifest identity, dispatch
  identity, terminal statuses, and critical pre-build gates.
- It writes:
  - `.aether/data/checkpoints/pre-build-phase-N.json`
  - `.aether/data/build/phase-N/manifest.json` with
    `dispatch_mode: "external-task"` and terminal dispatch statuses
  - `.aether/data/last-build-claims.json`, either from explicit claims or
    aggregated completed worker file/test claims
  - `spawn-tree.txt` entries/statuses if wrappers did not already record them
- It sets colony state to `BUILT`, keeps the phase `in_progress`, and points the
  next action at `aether continue`.

Expected wrapper flow:

1. `AETHER_OUTPUT_MODE=json aether build <phase> --plan-only`
2. Parse `result.dispatch_manifest`
3. For each dispatch wave, call `spawn-log`, spawn the matching Task/subagent,
   then call `spawn-complete`
4. Write a completion JSON file containing the original `dispatch_manifest` and
   the worker results/claims
5. `AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <file>`
6. `aether continue`

Focused verification:

- `go test ./cmd -run 'TestBuildPlanOnly|TestBuildFinalizeRecordsExternalTaskResultsForContinue' -count=1`

## Completed Slice: Build Wrapper Restoration

Claude/OpenCode build wrappers now use the bridge instead of the old
pass-through contract.

Changed files:

- `.aether/commands/build.yaml`
- `.claude/commands/ant/build.md`
- `.opencode/commands/ant/build.md`
- `.aether/docs/wrapper-runtime-ux-contract.md`
- `cmd/build_wrapper_ceremony_test.go`
- `cmd/platform_doc_hygiene_test.go`

Wrapper flow now required:

1. Ground with `AETHER_OUTPUT_MODE=visual aether status`.
2. Request the authoritative plan with
   `AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only`.
3. Parse `result.dispatch_manifest`.
4. Load `.aether/docs/command-playbooks/build-wave.md`.
5. Spawn real platform agents using `subagent_type="{agent_name}"` or the
   platform equivalent, with names/castes/waves/tasks from the manifest.
6. Call `aether spawn-log` before each worker and `aether spawn-complete` after.
7. Write a completion JSON file outside `.aether/data/`.
8. Run `AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS
   --completion-file <file>`.
9. Route to `/ant-continue`.

Important guardrails now enforced:

- Wrappers must not run `AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS`.
- Wrappers must not run `aether build --synthetic` after real agents finish.
- Wrappers must not invent castes, waves, or names.
- Wrappers must not parse visual output as truth.

Focused verification:

- `go test ./cmd -run 'TestBuildWrapperCeremonyContract|TestPlatformDocHygiene|TestClaudeOpenCodeCommandParity|TestCommandWrappersDeclareGeneratedSource' -count=1`

## Completed Slice: Continue Plan-Only Manifest

Architect review confirmed the continue bridge should mirror build:

- Keep normal `aether continue` as the direct Codex/runtime compatibility path.
- Add `AETHER_OUTPUT_MODE=json aether continue --plan-only` for wrappers.
- Add `continue-finalize` next; do not trust `spawn-log` / `spawn-complete` as
  advancement evidence.
- Finalization must re-run deterministic verification and claim checks before
  advancing. It must not trust stale plan-only output.

Implemented in this slice:

- `aether continue --plan-only` runs runtime-owned verification commands and
  claim checks, but does not mutate state, does not write reports, and does not
  spawn Go-side workers.
- It emits `result.continue_manifest` with:
  - phase/root metadata
  - verification snapshot
  - assessment snapshot
  - planned wrapper workers: Watcher, Gatekeeper, Auditor, Probe
  - worker names, castes, `agent_name`, waves, task IDs, status `planned`, and
    rendered worker briefs
  - `finalize_surface: "pending"` as the wrapper handoff marker

Focused verification:

- `go test ./cmd -run 'TestContinuePlanOnlyPrintsReviewManifestWithoutMutatingState' -count=1`

Next exact slice:

- Add `aether continue-finalize --completion-file <path|->`.
- It should re-load current state, re-run verification/claims, merge wrapper
  watcher/review results, run the existing gate/assessment path, then use the
  existing atomic advancement/report/session update logic.

## Completed Slice: Continue Finalize For Wrapper Agents

Implemented in this slice:

- New command: `aether continue-finalize --completion-file <path|->`.
- Completion JSON must include the original `continue_manifest` from
  `aether continue --plan-only` plus terminal wrapper worker results.
- The finalizer re-loads current state and rejects stale phase mismatch.
- It re-runs runtime-owned verification commands and build-claim checks at
  finalization time.
- It merges the external Watcher result into the verification report.
- It merges external Gatekeeper/Auditor/Probe results into the review report.
- It writes:
  - `.aether/data/build/phase-N/verification.json`
  - `.aether/data/build/phase-N/gates.json`
  - `.aether/data/build/phase-N/review.json`
  - `.aether/data/build/phase-N/continue.json`
- It records/updates spawn-tree entries for wrapper continue workers without
  duplicating entries when wrappers already called `spawn-log`.
- It uses the existing gate, assessment, housekeeping, context, and atomic
  advancement behavior to move to the next phase or block safely.

Expected continue wrapper flow:

1. `AETHER_OUTPUT_MODE=json aether continue --plan-only`
2. Parse `result.continue_manifest`
3. Spawn Watcher first, then Gatekeeper/Auditor/Probe from the manifest
4. Call `aether spawn-log` before each worker and `aether spawn-complete` after
5. Write a completion JSON file outside `.aether/data/`
6. `AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file <file>`
7. Route based on the finalizer's runtime result

Focused verification:

- `go test ./cmd -run 'TestContinuePlanOnly|TestContinueFinalizeRecordsExternalReviewAndAdvances' -count=1`

## Completed Slice: Continue Wrapper Restoration

Claude/OpenCode continue wrappers now use the continue bridge instead of the old
visual pass-through contract.

Changed files:

- `.aether/commands/continue.yaml`
- `.claude/commands/ant/continue.md`
- `.opencode/commands/ant/continue.md`
- `cmd/continue_wrapper_ceremony_test.go`
- `cmd/platform_doc_hygiene_test.go`
- `cmd/codex_visuals.go`

Wrapper flow now required:

1. Ground with `AETHER_OUTPUT_MODE=visual aether status`.
2. Request the authoritative plan with
   `AETHER_OUTPUT_MODE=json aether continue --plan-only $ARGUMENTS`.
3. Parse `result.continue_manifest`.
4. Spawn Watcher first from wave 1.
5. Spawn Gatekeeper, Auditor, and Probe from wave 2; these can run in parallel
   when the platform supports it.
6. Call `aether spawn-log` before each worker and `aether spawn-complete` after.
7. Write a completion JSON file outside `.aether/data/`.
8. Run `AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file
   <file>`.
9. Route from the finalizer result to `/ant-build N+1`, `/ant-continue`, or
   `/ant-seal`.

Important guardrails now enforced:

- Wrappers must not run `AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS`.
- Wrappers must not run direct non-plan-only `aether continue`.
- Wrappers must not invent castes, waves, or names.
- Wrappers must not parse visual output as truth.
- The normal direct `aether continue` path remains available for Codex/direct
  CLI compatibility.

Focused verification:

- `go test ./cmd -run 'TestContinueWrapperCeremonyContract|TestPlatformCommandDocsAvoidLegacyShellRuntime|TestContinuePlanOnly|TestContinueFinalizeRecordsExternalReviewAndAdvances' -count=1`

## Completed Slice: Plan Plan-Only Manifest

Architect review confirmed the plan bridge should mirror build/continue:

- Keep normal `aether plan` as the direct Codex/runtime compatibility path.
- Add a read-only manifest surface for Claude/OpenCode wrappers.
- Preserve one Go-owned planning-depth mapping instead of inventing a separate
  wrapper-only depth system.
- Add `plan-finalize` next; wrappers should not write `.aether/data/planning`
  as the authority path.

Implemented in this slice:

- `aether plan --plan-only --depth <fast|balanced|deep|exhaustive>` runs the
  same plan preflight checks but does not mutate state, write planning
  artifacts, update session context, record spawn-tree entries, or spawn
  Go-side planning workers.
- Depth maps to existing plan granularity:
  - fast -> sprint (1-3 phases)
  - balanced -> milestone (4-7 phases)
  - deep -> quarter (8-12 phases)
  - exhaustive -> major (13-20 phases)
- The command emits both `result.plan_manifest` and `result.planning_manifest`
  for wrapper compatibility.
- The manifest includes:
  - goal/root/generated_at
  - depth and granularity bounds
  - survey context
  - dispatch contract
  - two planned workers: Scout wave 1 and Route-Setter wave 2
  - worker names, castes, `agent_name`, waves, task IDs, status `planned`, and
    rendered worker briefs
  - `finalize_surface: "pending"` and `requires_finalizer: true`
- Existing plans without `--refresh` return `existing_plan: true` and
  `requires_finalizer: false`, preserving the direct "use current plan" path.

Expected plan wrapper flow after the next slices:

1. Ground with `AETHER_OUTPUT_MODE=visual aether status`.
2. Ask the user for planning depth: Fast, Balanced, Deep, or Exhaustive.
3. Run `AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>`.
4. Parse `result.plan_manifest` or `result.planning_manifest`.
5. Spawn Scout from wave 1.
6. Spawn Route-Setter from wave 2 with Scout findings and the exact
   `phase-plan.json` schema.
7. Call `aether spawn-log` before each worker and `aether spawn-complete` after.
8. Write a completion JSON file outside `.aether/data/`.
9. Run `AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <file>`.
10. Route from the finalizer result to `/ant-build 1` or a runtime-surfaced
    recovery command.

Plan-finalize contract for the next worker:

- New command: `aether plan-finalize --completion-file <path|->`.
- Completion JSON should include the original `plan_manifest`/`planning_manifest`
  and terminal worker results.
- Prefer embedded completion data as authority:
  - Scout result: `summary`, sourced `findings`, `gaps`, optional `scout_report`
  - Route-Setter result: `phase_plan` matching `codexWorkerPlanArtifact`
- Support claimed file fallback only after validation.
- Re-load current state and reject stale goal/state/refresh conflicts.
- Validate dispatch identities by name/caste/wave/task_id.
- Write canonical planning artifacts:
  - `.aether/data/planning/SCOUT.md`
  - `.aether/data/planning/ROUTE-SETTER.md`
  - `.aether/data/planning/phase-plan.json`
  - `.aether/data/phase-research/phase-N-research.md`
- Update spawn tree/run status for wrapper workers without duplicating
  `spawn-log` entries.
- Update `COLONY_STATE.json`: `READY`, first buildable phase, selected
  granularity, generated plan/confidence, `build_started_at: nil`, and planning
  events.
- Update session/CONTEXT/HANDOFF through existing runtime helpers.

Focused verification:

- `go test ./cmd -run 'TestPlanOnlyPrintsManifestWithoutMutatingState|TestPlanDepthMapsToGranularityBounds|TestPlanIncludesDispatchContract|TestPlanUsesSurveyAndRecordsPlanningDispatches|TestDispatchRealPlanningWorkers' -count=1`

Watcher audit notes before flipping `/ant-plan` wrappers:

- Add `plan-finalize` first. The wrappers must not move to manifest-driven
  orchestration while the finalizer surface is missing.
- Add `TestPlanWrapperCeremonyContract` before editing `.claude`/`.opencode`
  plan markdown. It should require the depth prompt, JSON `plan --plan-only
  --depth`, `result.plan_manifest` / `result.planning_manifest`, Scout wave 1,
  Route-Setter wave 2, spawn-log/spawn-complete, completion JSON outside
  `.aether/data`, and `plan-finalize`.
- Preserve direct CLI compatibility explicitly: direct `aether plan --depth
  deep` must still generate and persist a plan without `plan_only`, without
  `requires_finalizer`, and with `next: aether build N`.
- Extend plan-only non-mutation coverage whenever finalizer work touches nearby
  state paths. Current coverage checks planning dirs, phase research, spawn tree,
  session, event bus, runtime spawn-run files, CONTEXT, HANDOFF, and colony plan
  state.

Plan wrapper rollback note:

- If plan wrapper orchestration misbehaves, revert only `.aether/commands/plan.yaml`,
  `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, and their
  wrapper contract tests back to direct `AETHER_OUTPUT_MODE=visual aether plan`.
- Keep direct `aether plan` unchanged throughout; this remains the fallback
  runtime path for Codex/direct CLI users.
- Do not roll back build/continue bridges unless their own tests regress.

## Completed Slice: Go Narrator Launcher

### Files To Add Or Edit

- Added `cmd/narrator_launcher.go`
- Added `cmd/ceremony_emitter.go`
- Added `cmd/narrator_launcher_test.go`
- Added `cmd/ceremony_emitter_test.go`
- Edited `cmd/codex_build.go`
- Edited `cmd/codex_build_worktree.go`
- Edited `cmd/testing_main_test.go`

### Runtime Policy

Safe output policy:

- `AETHER_NARRATOR=off`: never launch.
- `AETHER_NARRATOR=auto` or unset: launch only when visual output is enabled.
- `AETHER_NARRATOR=on`: force launch in visual/human output, but still do not
  launch when `AETHER_OUTPUT_MODE=json`.
- `AETHER_OUTPUT_MODE=json`: no narrator stdout under any mode. JSON output must
  stay machine-parseable.

If a future release wants narrator data during JSON mode, add an explicit
stderr/file sink. Do not silently mix `[CEREMONY]` lines into JSON envelopes.

Failure policy:

- Missing Node is non-fatal.
- Missing `.aether/ts/dist/narrator.js` is non-fatal.
- Runtime start failure is non-fatal.
- Broken pipe or early narrator exit is non-fatal.
- Event publish failures are non-fatal for the build command; they should be
  test-visible but must not lose user work.

### Launcher Shape

`cmd/narrator_launcher.go` should provide:

```go
type narratorLauncher struct {
    // Owns the process, stdin pipe, stdout scanner, visual metadata temp file,
    // and cancellation.
}
```

Implemented behavior:

- Resolve `node` with an injectable `lookPath`.
- Resolve the runtime as an absolute path:
  - prefer `<repo root>/.aether/ts/dist/narrator.js`;
  - optionally fallback to `<hub>/system/ts/dist/narrator.js` if repo-local
    runtime is absent.
- Write a temporary visuals JSON envelope from `casteVisualContracts()` and pass
  it as `--visuals <path>`.
- Start `node dist/narrator.js --visuals <path>` with `exec.CommandContext`.
- Pipe event JSON lines to child stdin.
- Read child stdout in Go and call `writeVisualOutput(stdout, line+"\n")`.
- Drain child stderr without spamming command output.
- `Close()` closes stdin, waits, cancels if needed, waits for stdout drain,
  removes the temp visuals file, and is idempotent.

Do not call `event-bus-subscribe` from the parent build path. The Go process has
the events in hand; feed the sidecar directly through stdin. Keep
`event-bus-subscribe --stream` as a CLI/manual bridge and test fixture.

### Ceremony Emitter Shape

`cmd/ceremony_emitter.go` should provide a build-scoped emitter:

```go
type buildCeremonyEmitter struct {
    bus      *events.Bus
    narrator *narratorLauncher
    phaseID int
    phaseName string
}
```

Implemented behavior:

- Publish `events.CeremonyPayload` to `pkg/events.Bus` when `store` is
  available.
- Forward the exact persisted `events.Event` JSON to the narrator sidecar.
- If persistence fails, synthesize an event for the narrator only and continue.
- Protect the active emitter with a small mutex.
- Use a package-level active emitter only for the duration of `runCodexBuild`;
  it is restored with `defer`.
- Trim user-controlled event text and lists before persistence/forwarding.

### Build Event Insertion Points

Implemented phase-level events in `runCodexBuild`:

- After dispatches are planned and named: `ceremony.build.prewave`
  - include `phase`, `phase_name`, `total`, and success criteria count.
- Before worker execution starts, make the emitter active.
- After dispatch execution completes, close the launcher before final JSON/visual
  workflow output is written.

Implemented worker-level events in `cmd/codex_build_worktree.go`:

- Before each wave starts: `ceremony.build.wave.start`
  - include `phase`, `phase_name`, `wave`, `total`, message with execution
    strategy.
- On context-cancel before worker start: `ceremony.build.spawn`
  - status `timeout`, include worker identity and task id.
- On worktree allocation failure: `ceremony.build.spawn`
  - status `failed`, include blocker text.
- Immediately before invoking a worker: `ceremony.build.spawn`
  - status `starting`, include caste/name/task/task_id/spawn_id.
- In `invokeCodexWorkerWithRuntimeProgress`: `ceremony.build.tool_use`
  - status `running`, include message if present.
- After worker result: `ceremony.build.spawn`
  - status from result, include blockers, files created/modified, tests, tool
    count, and duration if available.
- After each wave finishes: `ceremony.build.wave.end`
  - include `completed`, `total`, and blocker count/message.

Scout's recommended narrow insertion points:

- `cmd/codex_build_worktree.go` wave start around the existing calls to
  `emitCodexBuildWaveProgress`.
- worker start around the existing calls to `emitCodexBuildWorkerStarted`.
- running progress inside `invokeCodexWorkerWithRuntimeProgress`.
- worker finish around the existing calls to `emitCodexBuildWorkerFinished`.

### Launcher Tests Added

- `TestNarratorLauncherOffSuppressesLaunch`
- `TestNarratorLauncherAutoSkipsJSONMode`
- `TestNarratorLauncherOnSkipsJSONMode`
- `TestNarratorLauncherAutoSkipsWhenNodeMissing`
- `TestNarratorLauncherMissingRuntimeDoesNotFail`
- `TestNarratorLauncherOnStreamsCeremonyEventsToBundledRuntime`
- `TestNarratorLauncherKeepsEventJSONLPersistence`
- `TestNarratorLauncherCloseCancelsStreamAndWaitsForRuntime`
- `TestNarratorLauncherHandlesEarlyRuntimeExit`
- `TestBuildSyntheticNarratorDoesNotPolluteJSONOutput`
- `TestNarratorLauncherUsesDistRuntimeDirectly` proves runtime launch uses
  `dist/narrator.js` and does not invoke `npm`, `npx`, `tsx`, or `narrator.ts`.
- `TestBuildCeremonyEmitterPersistsAndForwardsEvents`
- `TestBuildCeremonyEmitterTrimsUserControlledPayload`
- `TestActiveBuildCeremonyScopeRestoresPreviousEmitter`

Command-level JSON smoke should use a synthetic build fixture and assert:

- stdout is a valid `{"ok":true,"result":...}` or `{"ok":false,...}` envelope;
- stdout does not contain `[CEREMONY]`;
- event-bus JSONL still contains ceremony events if the build reached dispatch.

### Verification After Launcher Slice

Run:

```bash
npm --prefix .aether/ts ci
npm --prefix .aether/ts audit --package-lock-only --audit-level=low
npm --prefix .aether/ts run build
git diff --exit-code -- .aether/ts/dist/narrator.js
npm --prefix .aether/ts run typecheck
npm --prefix .aether/ts test
go test ./cmd -run 'TestNarrator|TestCeremony|TestEventBusSubscribe|TestEventBusStreamPipesToNarratorRuntime|TestVisualsDumpExportsCasteIdentityContract' -count=1
go test ./... -count=1 -timeout 300s
go test ./... -race -count=1 -timeout 600s
go vet ./...
git diff --check
rm -rf .aether/ts/node_modules
git status --short
```

Commit and push after the launcher is green.

Completed verification on 2026-04-24:

- `go test ./cmd -run 'TestNarratorLauncher|TestBuildCeremonyEmitter|TestActiveBuildCeremonyScope|TestBuildSyntheticNarratorDoesNotPolluteJSONOutput' -count=1`
- `go test ./cmd ./pkg/events -run 'TestNarratorLauncher|TestBuildCeremonyEmitter|TestActiveBuildCeremonyScope|TestBuildSyntheticNarratorDoesNotPolluteJSONOutput|TestCeremony|TestEventBusSubscribe|TestEventBusStreamPipesToNarratorRuntime|TestVisualsDumpExportsCasteIdentityContract' -count=1`
- `go test ./cmd -count=1`
- `go test ./... -count=1 -timeout 300s`
- `go vet ./...`
- `npm --prefix .aether/ts ci`
- `npm --prefix .aether/ts audit --package-lock-only --audit-level=low`
- `npm --prefix .aether/ts run build`
- `git diff --exit-code -- .aether/ts/dist/narrator.js`
- `npm --prefix .aether/ts run typecheck`
- `npm --prefix .aether/ts test`
- `go test ./... -race -count=1 -timeout 600s`
- `git diff --check`
- `rm -rf .aether/ts/node_modules`

## Completed Slice: Rolling Activity Display Foundation

The first rolling display slice is implemented in `.aether/ts/narrator.ts`.
The narrator preserves the compatibility event line, then renders a stateful
`COLONY ACTIVITY` frame.

State model:

- Tracks active wave number.
- Tracks spawns by `spawn_id` or `phase/wave/caste/name/task_id`.
- Tracks worker status, task summary, tool count, token count, blockers, files,
  tests, and last message.
- Tracks wave progress from `completed` and `total`.

Rendering rules:

- Plain output prints stable lines, preserving the existing event line first.
- Frame output groups workers into Active, Completed, Blocked, and Other.
- ANSI/control sequences are stripped through the existing sanitizer.
- Long frame text is truncated.
- Go visual metadata through `--visuals` drives caste labels and emoji.

Tests:

- `renders rolling activity frame with active and completed workers`
- `renders blocked workers and truncates long frame text`
- `keeps multi-wave activity history while current wave advances`

Remaining display work:

- Visual polish after seeing real build output in Claude/OpenCode/Codex.
- TTY live redraw/debounce is consciously deferred. The child narrator writes to
  a Go pipe, so true terminal redraw needs an explicit parent-controlled
  terminal contract such as a `--live` flag or Go-side redraw coordinator.

## Later Phases: Real Agent Spawning Bridge

Build and continue wrapper bridges are implemented for Claude/OpenCode. Remaining
work is to broaden the same bridge pattern across planning and later lifecycle
ceremonies.

Plan wrapper work:

- Restore depth prompt: Fast, Balanced, Deep, Exhaustive.
- Spawn Scout and Route-Setter planning loops.
- Persist confidence, assumptions, stall detection, and plan revisions.

Skill work:

- Preserve `skill-match`/`skill-inject` as Go-owned matching.
- Ensure wrapper-spawned agents receive matched skill sections.
- Emit `ceremony.skill.activate` when a worker activates a skill.

## Release And Rollback

Rollback controls:

- `AETHER_NARRATOR=off` disables the launcher immediately.
- Revert launcher commit to remove sidecar integration while keeping TS runtime.
- Revert wrapper commits independently if Task-tool orchestration regresses.
- Full milestone rollback remains a normal git revert of the v1.6 branch range.

Before release:

- Verify `aether install --package-dir "$PWD"` publishes `.aether/ts` to hub.
- Verify `aether update --force` syncs `.aether/ts` into a fixture repo.
- Verify missing Node falls back cleanly.
- Verify JSON mode is never polluted.
- Verify Claude Code, OpenCode, and Codex docs correctly describe their
  authority boundaries.

## Current Next Step

Next, commit/push the continue-wrapper checkpoint, then start plan wrapper
orchestration. The preconditions are now satisfied:

1. hub fallback smoke passed in a temporary HOME;
2. TTY live redraw is consciously deferred;
3. multi-wave TS fixture passes;
4. build and continue wrappers now have plan-only/finalizer bridges;
5. focused Go and TS gates pass.
