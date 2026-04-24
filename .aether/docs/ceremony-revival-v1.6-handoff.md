# Ceremony Revival v1.6 Handoff

Last updated: 2026-04-24T04:25:35Z

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

- This is only the planning half of the bridge. The next runtime surface should
  record/finalize externally spawned wrapper workers after `spawn-log` and
  `spawn-complete` have captured their results. Do not make wrappers call a fake
  `aether build --synthetic` after real Task work; that would overwrite the
  evidence trail with simulated dispatch.

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

After narrator display is reliable, restore Claude/OpenCode wrapper behavior.

Build wrapper work:

- `.claude/commands/ant/build.md` becomes an orchestrator again.
- `.opencode/commands/ant/build.md` gets structural parity.
- The wrapper calls Go for a manifest/plan-only build surface, then spawns real
  caste agents with the platform agent tool.
- Pre-wave specialists: Archaeologist, Oracle, Architect, Ambassador.
- Builder waves run with full context injection.
- Post-wave specialists: Watcher, Probe, Measurer, Chaos, Gatekeeper/Auditor as
  needed.
- Wrapper calls `aether spawn-log` before Task calls and `aether spawn-complete`
  after returns.

Continue wrapper work:

- Verification gates must run before phase advancement.
- Watcher verifies claims.
- Gatekeeper, Auditor, Probe, and Measurer gates fire where appropriate.
- Failures go to midden/graves, not silent state flips.

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

Next, start the real agent-spawning bridge for Claude/OpenCode wrappers. The
preconditions are now satisfied:

1. hub fallback smoke passed in a temporary HOME;
2. TTY live redraw is consciously deferred;
3. multi-wave TS fixture passes;
4. focused Go and TS gates pass.
