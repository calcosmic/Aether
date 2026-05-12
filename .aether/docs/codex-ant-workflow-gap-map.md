# Codex Ant Workflow Gap Map

Updated: 2026-05-12

This is the compact source-of-truth gap map for Phase 1 contract work. It
compares the documented ant workflow for Codex with the current Go command
behavior for `plan`, `build`, and `continue`.

For dummies: the docs say Codex should ask the runtime for a worker recipe,
show the recipe, run the listed helpers, then hand the results back to the
runtime. This map lists where the runtime, docs, and safety rules still do not
line up.

## Sources Compared

- Runtime guidance: `aether command-guide plan|build|continue --platform codex`.
- Wrapper source specs: `.aether/commands/plan.yaml`,
  `.aether/commands/build.yaml`, `.aether/commands/continue.yaml`.
- Codex lifecycle skill: `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`.
- Runtime code: `cmd/codex_plan.go`, `cmd/codex_build.go`,
  `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`,
  `cmd/codex_*_finalize.go`, `cmd/codex_workflow_cmds.go`,
  `cmd/command_guide.go`.
- Existing contract docs: `cmd/contracts/plan.md`, `cmd/contracts/build.md`,
  `cmd/contracts/continue.md`, `.aether/docs/wrapper-runtime-ux-contract.md`,
  `.aether/docs/source-of-truth-map.md`.

## Guardrails

- Runtime finalizers are the only authority for wrapper-orchestrated
  `.aether/data` state mutation.
- Worker claim paths must be clean repo-relative paths. Reject absolute paths,
  `../` traversal, empty paths, and `.aether/data` paths unless a command
  explicitly allows that state artifact.
- Temporary manifest, worker, and completion files belong outside
  `.aether/data`.
- Do not add shell execution from manifest, worker, or user-provided strings.

## Command Map

| Command | Documented Codex ant workflow | Current runtime behavior | Gap |
|---|---|---|---|
| `plan` | Codex selects depth, runs status, requests `aether plan --plan-only`, saves the JSON envelope to temp storage, checks `orchestrator_boundary_guidance`, renders ceremonies, spawns Scout and Route-Setter through the host, then calls `plan-finalize`. | The host-orchestrated shape and spawn tracking are documented across `command-guide`, YAML, and the Codex skill: all three now require `spawn-log`, `spawn-complete`, visible workers, `worker-complete`, and `plan-finalize`. The default `aether plan` still performs runtime planning/fallback and writes state directly, which is the raw/runtime path. `aether plan --plan-only` no longer persists `VerificationDepth` before `plan-finalize`, and `plan-finalize` rejects stale manifests by `generated_at`. The existing-plan branch can return no host-worker manifest even though Codex guidance expects `plan_manifest` or `planning_manifest`. | **P1:** define the existing-plan plan-only manifest contract. |
| `build` | Codex requests `aether build <phase> --plan-only`, checks boundary guidance, renders spawn and wave ceremonies, spawns manifest workers through the host, records spawn log/completion, then calls `build-finalize`. | `command-guide`, YAML, and the Codex skill agree. `build-finalize` validates manifest provenance, rejects unsafe worker claim paths, and commits build state. The default `aether build` still runs the internal runtime build path for raw/direct use. `aether build --plan-only` no longer persists prior-phase task reconciliation before `build-finalize`. | No P0 gap in the phase-6 scope. |
| `continue` | Default Codex path is runtime-owned: `aether continue --skip-watchers --verification-depth standard`. Host-spawned reviewers are only for explicit heavy external review, using `continue --plan-only --verification-depth heavy`, ceremonies, reviewer results, and `continue-finalize`. | Default continue behavior is aligned: runtime verifies, gates, handles signal housekeeping, and advances or blocks. The plan-only path creates an external-review manifest and exposes queen decisions without writing `queen-state-N.json`; `continue-finalize` owns queen-state, review, gate, and advancement writes. Heavy-review spawn tracking is aligned across YAML, command-guide, and the Codex skill. | No P0 gap in the phase-6 scope. |

## Cross-Cutting Status

- `orchestrator_boundary_guidance` is consistently documented in the command
  specs, source-of-truth docs, Codex skill, and command-guide tests.
- `cmd/command_guide_test.go` enforces YAML metadata and drift-guard anchors,
  but it does not enforce behavioral invariants such as "plan-only writes no
  state" or "worker claim paths are safe".
- `cmd/contracts/*.md` are stale in two ways: they state that plan-only is
  non-mutating while runtime plan-only paths still write state, and they use
  older state labels such as `BUILDING` instead of the current runtime states
  `EXECUTING`, `BUILT`, and `COMPLETED`.
- Temporary manifest/completion file placement is documented as outside
  `.aether/data`, but the finalizer and ceremony file flags currently accept
  arbitrary readable paths.
- Build worker claim paths are now hardened: build-finalize rejects absolute
  paths, repository escape paths, missing paths, ambiguous basename claims,
  symlinks, and `.aether/data` claims before persisting claims or handoffs.
- Finalizer packets do not consistently carry or validate freshness metadata
  such as state fingerprints, session ids, generated-at checks, or plan hashes.
  This is most visible in `plan-finalize`, where stale planning output can
  overwrite a newer plan.
- Spawn tracking is aligned for `plan`, `build`, and heavy `continue` across
  YAML, command-guide, and the Codex skill. Finalizers can still backfill
  spawn-tree records, so host-side spawn tracking remains a regression-test
  surface rather than a runtime-only guarantee.
- Orchestrator boundary questions can write `pending-decisions.json` during
  plan-only paths for `plan`, `build`, and heavy `continue`; count this as part
  of the plan-only mutation boundary until either moved or explicitly
  documented.
- Existing shell execution in this area is fixed command execution or
  verification command execution resolved from project docs/config. This phase
  did not add shell execution from manifest, worker, or user strings.

## Dependency-Ordered Next Slices

1. Add failing tests for wrapper-safety invariants: stale finalizer packets must
   be rejected; plan/build/continue plan-only paths must not mutate
   `.aether/data` state; and build-finalize must reject unsafe worker claim
   paths.
2. Add stale-packet protection to `plan-finalize`, using a state fingerprint,
   generation timestamp, session id, plan hash, or equivalent runtime-owned
   freshness guard.
3. Move plan-only state mutations into finalizers or rename/document the
   specific runtime side effect. Prefer moving them so the wrapper contract
   stays simple.
4. Define and test the existing-plan plan-only contract so Codex either receives
   a manifest/finalizer packet or gets a runtime-owned no-op response that does
   not imply host workers are needed.
5. Update `cmd/contracts/plan.md`, `cmd/contracts/build.md`, and
   `cmd/contracts/continue.md` after behavior is corrected, not before.
6. Let the next observable-output slice focus on user-facing ceremony and
   dispatch contract wording without changing these safety boundaries. Preserve
   the phase-6 worker-activity tests that keep YAML, command-guide, and skill
   guidance aligned.

## Forge-55 Handoff

Forge-55 should treat this document as the phase gap map. Avoid changing the
observable-output contract until the P0 wrapper-safety gaps are either fixed or
explicitly accepted by the Queen. If Forge-55 owns command output, preserve the
current `command-guide`/YAML alignment and do not weaken the finalizer-owned
state boundary.
