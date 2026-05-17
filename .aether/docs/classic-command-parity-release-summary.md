# Classic Command Parity Release Summary

Updated: 2026-05-16

Use this summary when preparing seal/final review notes for the Classic Command
Parity Matrix and TypeScript Host Command Spine milestone.

## User-Facing Summary

Aether has restored the important classic command ceremony without making
TypeScript a second runtime. The TypeScript host now acts as the command spine
for host-backed workflows: direct host commands parse flags and ask Go for
manifests; the wrapper/lifecycle orchestration built on those manifests invokes
Go-owned ceremony and hands worker results back to Go finalizers. Go remains
responsible for state, verification, gates, provider/auth diagnostics, and next
actions.

## What Is Back

- Plan/build worker orchestration goes through `aether host plan` and
  `aether host build`.
- Heavy external continue review goes through `aether host continue`.
- Classic visible ceremony is preserved through Go `aether ceremony ...`
  commands.
- Read-only dashboards (`status`, `watch`, `history`, `phase`, and
  `swarm --watch`) are kept factual instead of being dressed up as worker
  dispatch.
- Delivery and update surfaces (`update`, `publish`, `install`, `lay-eggs`,
  `porter`, `source-check`, `run`, `continue`, and `bump-version`) use progress
  ceremony: they show what the runtime is doing without implying a worker wave.
- Internal helper commands and finalizers stay quiet, so wrapper plumbing does
  not look like user-facing colony work.
- Claude/OpenCode wrappers, Codex command-guide output, shipped Codex skills,
  and docs now describe the same host-assisted contract.
- The parity matrix records classic command coverage, state ownership,
  finalizer requirements, wrapper coverage, and known host gaps.

## Honest Limits

- Default `aether continue` remains Go-owned and should not become TS-host
  plan-only orchestration.
- `colonize` and `seal` are documented as future TS host orchestration targets,
  not implemented `aether host` commands.
- The parity matrix is a tested contract artifact, not runtime authority.

## Release Gate

Do not claim release readiness for this milestone unless these pass:

```bash
go test ./cmd -run 'TestCodexHostBackedGuidesUseTypeScriptHostSpine|TestWrapperSourcesUseTypeScriptHostManifestSpine|TestClassicCommandParityMatrix' -count=1
npm --prefix .aether/ts-host run typecheck
npm --prefix .aether/ts-host test
go test ./...
```
