# Ceremony Revival v1.6 Handoff

Last updated: 2026-04-24T04:06:48Z

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

## Phase After Launcher: Rolling Activity Display

Once the launcher is wired, improve `.aether/ts/narrator.ts` from single-line
events into a ceremony frame.

State model:

- Track active wave number.
- Track spawns by `spawn_id` or `phase/wave/caste/name/task_id`.
- Track worker status, task summary, tool count, token count, blockers, files,
  tests, and last message.
- Track wave progress from `completed` and `total`.

Rendering rules:

- Keep a plain non-TTY mode that prints stable lines.
- Use live redraw only for TTY output.
- Debounce redraws to about 4Hz if events arrive rapidly.
- Strip ANSI/control sequences from all event text.
- Truncate long user-controlled strings before rendering.
- Continue to accept Go visual metadata through `--visuals`.

Tests:

- Vitest snapshots for wave start, active workers, completion, failure, blockers,
  and no-visuals fallback.
- A fixture where one worker exits early and later events still render.
- A fixture with overlong task/blocker text proving truncation.

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

Start with the rolling activity display slice above. Do not begin wrapper
rewrites until:

1. the narrator renders a stable multi-worker frame from the persisted build
   events now emitted by Go;
2. non-TTY output remains readable and low-noise;
3. long event text is visibly truncated;
4. TS snapshot tests cover active, completed, failed, and blocked workers;
5. full Go and TS gates pass again.
