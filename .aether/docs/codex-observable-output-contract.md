# Codex Observable Output Contract

Updated: 2026-05-12

This contract defines the visible ant-process output Codex must surface for the
core Aether lifecycle commands. It builds on
`.aether/docs/codex-ant-workflow-gap-map.md` and intentionally does not change
runtime behavior.

For dummies: this is the checklist for what the user should see and what files
the runtime should produce while Codex runs the colony lifecycle. It also marks
the places where current safety gaps mean the contract cannot yet be fully
trusted.

## Scope

Applies to Codex wrapper-equivalent orchestration for:

- `aether init`
- `aether discuss`
- `aether plan`
- `aether build`
- `aether continue`
- `aether seal`

Raw, exact, or no-orchestration user requests bypass the Codex orchestration
layer and should run the user's literal command.

## Global Output Rules

- JSON envelopes are the machine authority. Codex must parse JSON results and
  must not parse visual output as state.
- Visual output is user-facing only. Spawn plans, wave banners, worker-complete
  ceremonies, closeouts, and runtime status displays must come from the `aether`
  CLI visual renderer.
- Codex must inspect `orchestrator_boundary_guidance` before rendering spawn
  ceremonies, spawning workers, or preparing finalizer packets. If guidance is
  active or routes to `aether discuss`, Codex stops, shows the summary, routes
  to `aether discuss`, and requests a fresh plan-only manifest after resolution.
- Temporary manifest, worker result, and completion files must stay outside
  `.aether/data`.
- Runtime finalizers are the only authority for wrapper-orchestrated
  `.aether/data` state mutation. Codex must not hand-edit state files.
- Worker descriptions must preserve runtime identity exactly:
  `{caste emoji} {Caste} {name}: {task}`.
- `aether spawn-log` must be called before each host-spawned worker and
  `aether spawn-complete` after each terminal worker result when the manifest
  path uses host workers.
- Worker `brief` and `skill_section` values are runtime-provided dispatch
  content. Codex passes them through; it does not execute shell from those
  strings.
- Worker claim paths must be clean repo-relative paths: no absolute paths,
  empty paths, `../` traversal, or `.aether/data` paths unless a command
  explicitly allows that state artifact.
- Finalizer completion packets must be fresh for the current colony state. The
  runtime should reject stale packets using a state fingerprint, session id,
  generated-at bound, plan hash, or equivalent guard before writing canonical
  state.

## Command Contract

| Command | Codex path | Required visible output | Required runtime artifacts | Spawn/finalizer surface | Blocked by known gaps |
|---|---|---|---|---|---|
| `init` | Full orchestration before state creation. Codex runs init research, asks needed scoping questions, asks for Colony Mode vs Orchestrator Mode, synthesizes a refined goal and charter JSON, then runs the visual runtime init command. | Runtime visual init result, refined charter summary, approved strategic pheromone suggestions, and the next command (`aether discuss` or `aether plan`). | Runtime-created `.aether/data/COLONY_STATE.json`, `session.json`, `activity.log`, and worker handoff scaffold. | No host worker spawn. Runtime owns state creation. | None from Anvil's P0 gap map. |
| `discuss` | Semi-intelligent clarification. Codex may run `discuss-analyze`, presents real runtime questions, and resolves answers through runtime `aether discuss --resolve`. | Runtime visual discuss result, surfaced clarification questions, settled status, and route back to `aether plan` when settled. | Runtime-updated `.aether/data/pending-decisions.json`; runtime may update `pheromones.json` for hard-constraint resolutions. | No host worker spawn. Runtime owns decision and signal persistence. | None from Anvil's P0 gap map. |
| `plan` | Full host orchestration from `aether plan --plan-only`. Codex saves the JSON envelope to temp storage, checks boundary guidance, renders spawn and wave ceremonies, spawns Scout and Route-Setter, writes a completion packet, calls `plan-finalize`, then renders closeout. | Status context, spawn-plan ceremony, wave-start banners, visible live planning workers, worker-complete ceremonies, final closeout, phase count/confidence summary, and next build command. | Temp manifest and completion files outside `.aether/data`; finalizer-owned planning artifacts and state updates. | Host workers from `plan_manifest` or `planning_manifest`; runtime finalizer `aether plan-finalize --completion-file <file>`. | `plan --plan-only` no longer persists `VerificationDepth` before `plan-finalize`, and `plan-finalize` rejects stale manifests by `generated_at`. Remaining P1: define the existing-plan no-op manifest/finalizer contract. |
| `build` | Full host orchestration from `aether build <phase> --plan-only`. Codex saves the JSON envelope, checks boundary guidance, renders spawn/wave ceremonies, calls spawn-log/spawn-complete around each worker, writes completion JSON, calls `build-finalize`, then renders closeout. | Status plus active signals, spawn-plan ceremony, wave-start banners, visible live workers, worker-complete ceremonies, final closeout, actual workers/tasks summary, and route to `aether continue`. | Temp manifest, worker results, and completion file outside `.aether/data`; finalizer-owned build state, dispatch records, claims, and handoffs. | Host workers from `dispatch_manifest.execution_plan`; runtime finalizer `aether build-finalize <phase> --completion-file <file>`. | `build --plan-only` no longer persists prior-phase reconciliation before `build-finalize`; unsafe worker claim paths are rejected by the finalizer. |
| `continue` | Default path is runtime-owned verification: `aether continue --skip-watchers --verification-depth standard`. Codex uses heavy host review only when the user requests heavy verification or runtime asks for wrapper-spawned reviewers. | Default: runtime visual verification, gates, housekeeping, advance/block decision, and next command. Heavy: spawn-plan, wave-start, live reviewers, worker-complete, closeout, and finalizer result. | Default: runtime-owned verification and state advancement/blocking artifacts. Heavy: temp manifest/completion outside `.aether/data`; finalizer-owned review results and advancement. | Default has no host spawn or `continue-finalize`. Heavy uses `continue_manifest` plus `aether continue-finalize --completion-file <file>`. | Heavy `continue --plan-only` exposes queen decisions in the manifest but does not persist `queen-state-N.json`; `continue-finalize` owns queen-state, review, gate, and advancement writes. |
| `seal` | Semi-intelligent final review from `aether seal --plan-only`. Codex stops on runtime blockers, checks boundary guidance, renders final-review ceremonies, spawns reviewers, preserves structured review fields, calls `seal-finalize`, then renders closeout. | Status readiness, blocker/recovery output if present, spawn-plan ceremony, wave-start banners, visible final-review workers, worker-complete ceremonies, seal closeout, runtime seal result, and Porter readiness output without running delivery actions. | Temp manifest/completion outside `.aether/data`; finalizer-owned final review, `.aether/CROWNED-ANTHILL.md`, seal state, review ledgers, and hive promotion output. | Host final-review workers from `seal_manifest`; runtime finalizer `aether seal-finalize --completion-file <file>`. | No seal-specific P0 in Anvil's map, but the same temp-file and path hygiene constraints still apply. |

## Blocked Contract Surface

The following observable guarantees are deliberately not marked complete until
the P0 gaps in the gap map are fixed or explicitly accepted by the Queen:

1. Existing-plan `plan --plan-only` behavior is not yet aligned with Codex
   guidance that expects a manifest or explicit no-op finalizer surface.
2. Orchestrator boundary questions can still write `pending-decisions.json`
   during plan-only paths, so boundary-question creation is part of the
   mutation surface until moved or explicitly documented.
3. The stale `cmd/contracts/plan.md`, `cmd/contracts/build.md`, and
   `cmd/contracts/continue.md` files should not be updated to promise the
   stronger contract until runtime behavior and tests match it.
4. Plan, build, and heavy `continue` spawn tracking is aligned across YAML,
   command-guide, and the Codex skill; keep the phase-6 command-guide tests as
   the release guard for this surface.

Until then, Probe and Watcher should treat this document as the intended
observable contract plus explicit blocked surface, not as proof that the runtime
already satisfies every safety invariant.

## Verification Targets

Future implementation slices should add tests before changing runtime behavior:

- Plan/build/continue plan-only commands do not mutate `.aether/data` state
  before their finalizers.
- `plan-finalize` rejects stale completion packets before writing plan state.
- Build-finalize keeps rejecting absolute, traversal, missing, ambiguous,
  symlink, and `.aether/data` claim paths before writing claims or handoffs.
- Existing-plan `plan --plan-only` has a tested manifest/no-op contract.
- Command-guide, `.aether/commands/*.yaml`, Codex lifecycle skills, and visual
  ceremony output stay aligned for the six lifecycle commands above.
- `TestCodexLifecycleGuidesRequireVisibleWorkerActivity`,
  `TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity`, and
  `TestCodexLifecycleSkillMirrorsWorkerActivityContract` remain green.
