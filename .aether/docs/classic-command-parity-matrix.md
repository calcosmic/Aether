# Classic Command Parity Matrix And TypeScript Host Spine

Updated: 2026-05-16

This document explains how Aether is restoring the useful ceremony and command
behavior from older releases while keeping the modern Go runtime as the
authority.

For dummies: TypeScript is the conductor for wrapper-assisted flows. Direct
host commands fetch the work list from Go; the wrapper/lifecycle layer lines up
workers and asks Go to render ceremony and finalize results. Go is still the
engine. Go changes state, verifies work, renders canonical ceremony, and decides
what happens next.

## Authority

The machine-readable matrix lives at `.aether/commands/classic-command-parity.json`.
That file is a contract and test artifact, not runtime authority.

Runtime authority remains in:

- `cmd/command_guide.go` for Codex orchestration guidance.
- Cobra command implementations in `cmd/`.
- Plan-only manifests and Go finalizers.
- `.aether/commands/*.yaml`, `.claude/commands/ant/*.md`, and
  `.opencode/commands/ant/*.md` for wrapper source parity.
- `.aether/ts-host/src/command-registry.ts` and `.aether/ts-host/src/host.ts`
  for the TypeScript host command spine.

## Current Command Matrix

| Command | Category | TypeScript host surface | State owner | Finalizer | Wrapper coverage | Status |
|---|---|---|---|---|---|---|
| `init` | full orchestration | none | Go runtime | none | YAML, Claude, OpenCode | Host not needed; wrapper/Codex intake stays rich. |
| `discuss` | semi-intelligent | none | Go runtime | none | YAML, Claude, OpenCode | Host not needed; clarification state stays Go-owned. |
| `colonize` | full orchestration | missing orchestration target | Go finalizer | `colonize-finalize` | YAML, Claude, OpenCode | Future host target; current wrappers use Go plan-only. |
| `plan` | full orchestration | orchestration manifest | Go finalizer | `plan-finalize` | YAML, Claude, OpenCode | Host-backed. |
| `build` | full orchestration | orchestration manifest | Go finalizer | `build-finalize` | YAML, Claude, OpenCode | Host-backed. |
| `continue` | semi-intelligent | heavy-review manifest | Go runtime or Go finalizer | heavy only: `continue-finalize` | YAML, Claude, OpenCode | Default is Go-owned; heavy review is host-backed. |
| `seal` | semi-intelligent | missing orchestration target | Go finalizer | `seal-finalize` | YAML, Claude, OpenCode | Future host target; current wrappers use Go plan-only. |
| `oracle` | full orchestration | lifecycle loop | Go runtime or Go finalizer | iteration finalizer when used | YAML, Claude, OpenCode | Host owns loop conduct, Go owns state. |
| `swarm` | full orchestration + dashboard | display and plan | Go finalizer for problem runs; Go runtime for `--watch` | `swarm-finalize` for problem runs only | YAML, Claude, OpenCode | Problem-bearing swarm runs use worker theatre; `swarm --watch` is a dashboard. |
| `watch` | literal/display | display | Go runtime | none | YAML, Claude, OpenCode | Host can display; Go owns data. |
| `status` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Direct runtime passthrough. |
| `history` | literal/display | none | Go runtime | none | YAML, Claude, OpenCode | Dashboard only; reports stored events. |
| `phase` | literal/display | none | Go runtime | none | YAML, Claude, OpenCode | Dashboard only; reports current phase facts. |
| `resume` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Direct runtime passthrough. |
| `focus` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Signal write stays Go-owned. |
| `redirect` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Signal write stays Go-owned. |
| `feedback` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Signal write stays Go-owned. |
| `pheromones` | literal | none | Go runtime | none | YAML, Claude, OpenCode | Direct runtime display. |

## TypeScript Host Spine

The host spine is deliberately small:

1. Parse command flags and normalize arguments.
2. Call Go in JSON mode for programmatic plan/build/continue manifests.
3. Leave visual ceremony and finalization to the wrapper/lifecycle layer that
   consumes those manifests.
4. Surface sanitized provider/auth diagnostics from Go-owned checks.
5. Preserve completion packet contracts for Go finalizers.

It must not:

- write `.aether/data` directly;
- parse visual output as state;
- duplicate Go verification, gates, finalizers, provider diagnostics, or
  ceremony templates;
- claim `aether host colonize` or `aether host seal` until those host surfaces
  are actually implemented.

## Ceremony Routing

- Worker theatre: `plan`, `build`, `colonize`, `seal`, and problem-bearing
  `swarm` runs because they prepare or consume real worker manifests.
- Guided rituals: `init`, `discuss`, and `oracle` because they guide intent,
  research, or iteration while Go owns state and canonical facts.
- Dashboards: `status`, `watch`, `history`, `phase`, `resume`, and
  `swarm --watch` because they show current runtime facts without dispatching
  new workers.
- Progress/delivery: `run`, `continue`, `update`, `publish`, `install`,
  `lay-eggs`, `porter`, `source-check`, and `bump-version` because they show
  setup, verification, release, or delivery progress without inventing workers.
- Quiet/internal: finalizers, `command-guide`, spawn log helpers, ceremony
  helper commands, shell completion, version helpers, and generated progress
  helpers because they are machine or wrapper plumbing rather than user-facing
  worker ceremonies.

## Verification Anchors

Focused parity checks:

```bash
go test ./cmd -run 'TestCodexHostBackedGuidesUseTypeScriptHostSpine|TestWrapperSourcesUseTypeScriptHostManifestSpine|TestClassicCommandParityMatrix' -count=1
npm --prefix .aether/ts-host run typecheck
npm --prefix .aether/ts-host test
```

Release checks still need the broader smoke listed in
`.aether/docs/release-readiness-handoff.md`.
