# Ceremony Revival via Bundled TypeScript Narrator

Milestone: v1.6

## Architecture

Go owns state, dispatch contracts, event truth, and lifecycle safety. The bundled
TypeScript narrator owns rich terminal rendering. The LLM wrapper owns real
Task-tool subagent spawning on platforms that expose that tool.

This deliberately keeps the Go rewrite. The goal is to restore v5.4's ceremony
and real worker bridge on top of the current runtime instead of replacing it.

## Verified Assumptions

- Node is already part of the public npm bootstrap path: `npm/package.json`
  requires Node `>=18`. For direct Go-only installs, Node remains optional.
- `.aether/ts` is embedded as an explicit file list in `embedded_assets.go` so
  install/update syncs the narrator package without accidentally shipping
  `node_modules`.
- The installed narrator runtime is `dist/narrator.js`, a dependency-free Node
  artifact generated from the TypeScript source. `npm ci` is a developer/CI step,
  not an installed-runtime requirement.
- `pkg/events.Bus.Subscribe` is in-process only. Cross-process narration needs
  either parent-process piping or a file-backed stream over `event-bus.jsonl`.
- Current playbooks still contain the intended direct-spawn flow. In particular,
  `build-wave.md` and `build-full.md` explicitly instruct the Queen to spawn
  caste agents with the Task tool, log spawn state, run Probe verification, and
  persist failures to midden/graves.
- Current Go build/plan/colonize paths render dispatches and artifacts, but the
  markdown wrappers do not yet parse manifests and spawn Claude/OpenCode agents.

## Phase 1: Assumptions and Gap Audit

- Map current wrapper behavior against the v5.4 direct-spawn playbooks.
- Define the platform contract: Go command state vs wrapper Task-tool authority.
- Record the event protocol the narrator consumes.
- Preserve Codex best-effort direct CLI execution without claiming Claude/OpenCode
  Task-tool work happened inside Go.

## Phase 2: Event Protocol and Narrator Foundations

- Add `.aether/ts` with a strict TypeScript narrator package. (Initial scaffold
  exists.)
- Embed `.aether/ts` in install assets with explicit files only. (Initial
  scaffold exists.)
- Add ceremony event topic and payload types. (Initial scaffold exists.)
- Add `event-bus-subscribe --stream --filter ceremony.*` for NDJSON streaming.
  (Initial scaffold exists.)
- Add `visuals-dump --json` so TS consumes Go's caste emoji/color/label maps.
  (Initial scaffold exists; narrator can consume the envelope via `--visuals`.)
- Build a narrator stub that reads NDJSON from stdin and prints one ceremony line
  per event. (Initial scaffold exists; `dist/narrator.js` is dependency-free at
  runtime; Go auto-launch is not wired yet.)

### Phase 2 Checkpoint: 2026-04-24

Foundation commits through `aa8d6d4b` are pushed to
`codex/ceremony-narrator-foundation-v16`.

Completed:

- Dependency-free `dist/narrator.js` runtime is committed and install/update
  packaged.
- `visuals-dump --json` exposes Go-owned caste identity, and the narrator
  accepts it via `--visuals`.
- `event-bus-subscribe --stream --filter ceremony.*` can feed NDJSON into the
  narrator runtime.
- CI/release/dependabot cover the TS package and dist drift.

Completed after this checkpoint:

- Add Go auto-launch behind `AETHER_NARRATOR`.
- Emit build ceremony events from the build dispatch lifecycle.
- Prove JSON output is never polluted by narrator text.
- Prove missing Node/runtime is non-fatal.
- Prove sidecar stdout is routed through Go's visual output mutex.

Phase 2/3 foundation checkpoint:

- Manual installed-hub smoke passed in a temporary HOME and fixture repo.
- The fixture removed local `.aether/ts`, and `AETHER_NARRATOR=on aether build
  1 --synthetic` still rendered `COLONY ACTIVITY` from the hub fallback.
- Optional TTY live redraw/debounce is deferred until real wrapper output can
  guide the terminal contract.

The detailed implementation contract is tracked in
`.aether/docs/ceremony-revival-v1.6-handoff.md`.

## Phase 3: Rolling Activity Display

- Maintain an in-memory ceremony frame keyed by spawn ID. (Initial foundation
  exists.)
- Render wave progress, active workers, completed workers, tool counts, blockers,
  and token counts where available. (Initial foundation exists.)
- Strip ANSI when stdout is not a TTY unless explicitly forced.
- Debounce live redraws if event volume makes the terminal flicker. (Deferred
  pending real wrapper output.)

## Phase 4: Subagent Spawn Restoration

- Restore Claude/OpenCode build wrappers as orchestrators, not pass-throughs.
  (Implemented for build wrappers.)
- Run `aether build --plan-only <phase>` or equivalent manifest generation.
  (Implemented as `AETHER_OUTPUT_MODE=json aether build <phase> --plan-only`;
  it is read-only and emits `dispatch_manifest`.)
- For each wave, spawn real caste agents via the platform agent tool.
  (Build wrappers now instruct this from `dispatch_manifest`; live platform
  smoke remains next.)
- Restore pre-wave Archaeologist, Oracle, Architect, and Ambassador hooks.
- Restore Builder waves, Probe verification, Watcher verification, Measurer, and
  Chaos.
- Call `aether spawn-log` before each Task call and `aether spawn-complete` after
  each returned result so the narrator and spawn tree reflect real work.
- Add a runtime record/finalize surface for wrapper-spawned work before the
  wrappers advance phase state. Do not finalize by running `aether build
  --synthetic` after real wrapper agents; that would mix simulated evidence with
  real Task-tool execution.
  (Implemented as `AETHER_OUTPUT_MODE=json aether build-finalize <phase>
  --completion-file <path|->`; it writes the external-task manifest, claims,
  spawn tree statuses, and BUILT state that `aether continue` verifies.)

## Phase 5: Continue and Plan Orchestration

- Restore continue as real verification before advancement.
  (Continue plan-only manifest and continue-finalize implemented; wrapper
  rewrite is next.)
- Restore plan depth ceremony: Fast, Balanced, Deep, Exhaustive.
- Run Scout plus Route-Setter planning loops with stall detection and confidence.
- Persist validated learnings, midden failures, and blocker status honestly.

## Phase 6: Full Lifecycle Ceremony and Skills

- Add narrator ceremonies for colonize, plan, build, continue, pheromones,
  graveyard/entomb, and chamber sealing.
- Emit skill activation events from worker prompts or spawn wrappers.
- Keep QUEEN.md, Hive, midden, graveyard, and pheromone context visible where
  they affect worker behavior.

## Phase 7: Parity, Verification, and Release

- Verify Claude Code, OpenCode, and Codex surfaces do not drift.
- Verify `AETHER_NARRATOR=off` and missing Node both fall back cleanly.
- Verify `aether update --force` syncs `.aether/ts`.
- Keep rollback simple: disable narrator by env var, revert wrapper playbook
  commits independently, or revert the full v1.6 milestone.

## Out of Scope

- Reverting the Go runtime rewrite.
- Reviving the old standalone `aether-colony` runtime package.
- Redesigning the visual language beyond v5.4 parity.
- Reworking Hive Brain internals beyond restoring visible lifecycle integration.
