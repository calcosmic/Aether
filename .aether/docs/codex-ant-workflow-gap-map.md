# Codex Ant Workflow Gap Map

Updated: 2026-05-18

This is the compact source-of-truth gap map for Codex lifecycle orchestration
after the TypeScript host command spine, wrapper-contract, ceremony, and release
hardening work.

For dummies: Codex should ask Go for a recipe, show the ceremony, run only the
workers in that recipe, then hand the results back to Go. Go is still the engine
that writes state and decides whether the phase really advanced.

## Sources Compared

- Runtime guidance:
  `aether command-guide plan|build|continue|colonize|seal --platform codex`.
- Wrapper source specs: `.aether/commands/plan.yaml`,
  `.aether/commands/build.yaml`, `.aether/commands/continue.yaml`,
  `.aether/commands/colonize.yaml`, and `.aether/commands/seal.yaml`.
- Codex lifecycle skill:
  `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`.
- TypeScript host source: `.aether/ts-host/src/host.ts`,
  `.aether/ts-host/src/command-registry.ts`, and
  `.aether/ts-host/src/lifecycle.ts`.
- Runtime code: `cmd/host_cmd.go`, `cmd/command_guide.go`,
  `cmd/codex_*_finalize.go`, `cmd/finalizer_completion_contract.go`, and
  `cmd/codex_workflow_cmds.go`.
- Contract docs: `cmd/contracts/plan.md`, `cmd/contracts/build.md`,
  `cmd/contracts/continue.md`, `.aether/docs/host-command-reference.md`,
  `.aether/docs/wrapper-runtime-ux-contract.md`, and
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
- Provider availability preflight is separate from post-launch provider/API/auth
  failure. Generated context must keep those failure classes distinct.
- Generated context may surface only sanitized provider, cause, and next action.
  Do not include raw provider stdout/stderr, tokens, or auth probe output.
- `.opencode/package.json` and `.opencode/package-lock.json` are ignored local
  OpenCode install artifacts when present. Tracked TS-host package files live
  under `.aether/ts-host/`.

## Command Map

| Command | Documented Codex ant workflow | Current runtime behavior | Gap |
|---|---|---|---|
| `plan` | Codex selects planning depth, requests `aether host plan`, saves the JSON envelope outside `.aether/data`, renders ceremony, runs Scout and Route-Setter through the platform host, records worker completion, then calls `plan-finalize`. | `command-guide`, YAML, the Codex skill, TS host, and Go finalizer now describe the same host-backed shape. `plan-finalize` validates manifest provenance/freshness and owns plan persistence. Raw `aether plan` remains the direct Go path. | **P1:** define the existing-plan host response as an explicit no-worker/no-finalizer contract or a manifest contract. |
| `build` | Codex requests `aether host build <phase>`, checks boundary guidance, renders spawn/wave ceremony, runs only the manifest workers, records worker completion, then calls `build-finalize`. | Aligned across command-guide, YAML, Codex skill, TS host, and Go. `build-finalize` validates manifest provenance/freshness, rejects unsafe worker claim paths, reconciles completed-vs-timeout worker results, and commits build state. Raw `aether build` remains available for direct runtime use. | No known P0 gap. |
| `continue` | Default Codex path uses Go-owned `aether continue --skip-watchers --verification-depth standard`. Heavy external review is explicit with `aether host continue --classic-ceremony`, worker ceremony, reviewer results, and `continue-finalize`. | Default continue stays Go-owned. The host-backed heavy-review path exposes queen decisions and reviewer dispatches without writing queen-state before `continue-finalize`, and the finalizer owns review, gate, and advancement writes. | No known P0 gap. |
| `colonize` | Codex requests `aether host colonize`, renders survey worker ceremony, runs the survey manifest, then calls `colonize-finalize`. | TS host delegates to `aether colonize --plan-only`; Go owns survey persistence through `colonize-finalize`. The command reference and wrapper sources now treat colonize as host-backed rather than future work. | No known P0 gap. |
| `seal` | Codex requests `aether host seal`, renders final-review ceremony, runs the final review manifest, then calls `seal-finalize`. | TS host delegates to `aether seal --plan-only`; Go owns seal review persistence, release evidence, and final state transition through `seal-finalize`. The command reference and wrapper sources now treat seal as host-backed rather than future work. | No known P0 gap. |

## Cross-Cutting Status

- Host-backed lifecycle coverage now includes `colonize`, `plan`, `build`,
  heavy `continue`, and `seal`.
- Runtime ceremony is restored for manifest spawn plans, worker wave theatre,
  worker completion lines, and closeout summaries.
- Finalizers reject completion files under `.aether/data` and reject stdin
  completion input. The documented pattern is a temp run directory such as
  `${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json`.
- Plan/build finalizers validate manifest freshness with `generated_at`; plan
  finalization also rejects stale claimed planning artifacts.
- Build worker claim validation rejects absolute paths, repository escapes,
  missing paths, ambiguous basename claims, symlinks, and `.aether/data` claims.
- Spawn tracking is aligned for host-backed lifecycle commands across YAML,
  command-guide, Codex skill, TS host, and Go tests.
- Orchestrator boundary questions remain an explicit orchestration side effect:
  questions can be created before finalization, but finalizers still own
  lifecycle state writes.
- Provider/auth wording now distinguishes preflight availability failures from
  launched-worker provider/API/auth diagnostics.

## Dependency-Ordered Next Slices

1. Define and test the existing-plan host response contract so Codex does not
   imply workers are needed when Go is simply returning an already accepted
   plan.
2. Add optional stronger finalizer provenance, such as state fingerprint,
   session id, or plan hash, on top of the current `generated_at` freshness
   checks.
3. Consider tightening completion-file validation from "outside `.aether/data`"
   to an explicit temp-run prefix once every wrapper uses the documented temp
   path.
4. Keep provider/auth launched-worker redaction tests in the release gate as
   new platforms or provider adapters are added.
5. Keep the parity matrix, command-guide, YAML wrappers, Codex skill, and
   TypeScript host command registry updated in the same change whenever a
   lifecycle command changes.

## Handoff

Treat this file as a current orientation map, not as runtime authority. Runtime
behavior is authoritative in Go finalizers and the TS host registry; this map is
here to keep the next milestone focused on remaining contract edges instead of
reopening already-hardened P0 work.
