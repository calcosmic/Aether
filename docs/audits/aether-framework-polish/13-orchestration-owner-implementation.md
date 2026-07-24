# Production Orchestration Owner Implementation

Date: 2026-07-22

Status: implemented in the unreleased working tree; full release-candidate proof
still required.

## Result

Aether now has one subprocess-provider launch boundary: Go. The TypeScript host
still coordinates manifests and concurrent waves, but it no longer detects,
selects, preflights, launches, or parses Claude, OpenCode, or Codex CLI processes.

For beginners: TypeScript can decide which jobs run together, but it hands every
actual AI-program launch to the Go engine. There is no second hidden launcher that
can choose a different model or report different results.

## Prior Failure Shape

Production behavior was split across:

- direct Go dispatchers in `pkg/codex`;
- `.aether/ts-host/src/platform-dispatcher.ts`, which independently selected and
  launched the same provider CLIs;
- TypeScript prompt/result parsing separate from Go;
- `control-ts` adapters that returned `available: true` and completed empty work;
- platform-native wrapper panels that were not clearly distinguished from the
  subprocess-host path.

This allowed provider fallback, capability/reporting drift, duplicate launch
ownership, and false completion from a reachable experimental control plane.

## Implemented Boundary

### Go Adapter

`cmd/internal_worker_adapter.go` adds a hidden internal command with two modes:

- `--preflight` selects the provider and returns its typed capability/availability
  contract without dispatching work;
- `--request-file` loads one bounded typed worker request, resolves the
  platform-specific agent, assembles the prompt, invokes `pkg/codex`, validates
  worker identity and terminal status, redacts diagnostics, and returns structured
  claims.

Production fake selection is rejected. Simulation requires explicit `--simulate`
and is test-only. Requests cannot supply provider configuration overrides. Request
files must be regular non-symlink files under an
`aether-worker-request-*` system-temporary directory.

### Hard Provider Pins

`pkg/codex/platform_dispatch.go` now treats a valid explicit
`AETHER_WORKER_PLATFORM` value as an absolute selection. If that provider is
unavailable, selection fails. It does not try the next installed provider.

### TypeScript Host

`.aether/ts-host/src/worker-dispatch.ts` writes an ephemeral typed request and
uses the asynchronous Go bridge. It verifies `execution_owner: "go-adapter"`,
schema version, platform, identity, and status before accepting a result.

`.aether/ts-host/src/host.ts` delegates production preflight to Go. The host keeps
wave ordering, concurrency, retries, ceremony calls, and completion transport.
It does not own provider policy or subprocess launch.

`platform-dispatcher.ts` and `prompt-assembler.ts` remain only as deprecated
regression fixtures. A static test blocks imports from production host sources.

### Retired Control Plane

`control-ts` is private and reports status `retired`. Every adapter is unavailable,
dispatch throws, and phase execution emits failure without completion. Its public
index no longer exports orchestration/state APIs. Direct legacy state/event paths
default below `control-ts/.retired-state`; tests inject unique temporary paths.

During this checkpoint, the old test path was confirmed to have removed the live
`.aether/data/COLONY_STATE.json`. The exact cached payload was restored from
`.cache_COLONY_STATE.json`, then the isolation repair was verified by hashing the
live state before and after all 133 `control-ts` tests. The hashes matched.

## Native Wrapper Boundary

Claude/OpenCode/Codex wrappers may deliberately request a dry-run manifest and
launch visible native Task/subagent panels. In that mode the platform wrapper is
the launch adapter for that run; Go remains manifest/state/finalizer authority.

This is not duplicate production ownership if the run selects one mode:

1. Go subprocess adapter; or
2. platform-native panels from a dry-run manifest.

Command guidance forbids using both for the same manifest. The supported TS-host
and native Go build paths now persist the selected execution owner and immutable
run binding. Advanced platform-native panels outside that adapter path remain
guidance-enforced and must not claim equivalent recovery.

## Evidence

| Invariant | Evidence | Test |
| --- | --- | --- |
| Go owns TS-host subprocess launch | `worker-dispatch.ts`, `go-bridge.ts`, hidden adapter | TS delegation and static-import tests |
| Explicit pin cannot fall back | `platform_dispatch.go` | unit plus compiled unavailable-pin case |
| Malformed/mismatched result cannot complete | hidden adapter result validation | Go unit plus compiled fake-provider case |
| Async waves remain concurrent | `callGoJSONAsync` | TS wave/bridge tests |
| Provider output is redacted | Go/TS diagnostic sanitizers | Go and TS redaction tests |
| Stub control plane cannot succeed | retired adapters and `runPhase` | 133 `control-ts` tests |
| Retired tests cannot touch live state | environment-injected state path | before/after SHA-256 equality |

Focused verification completed at this checkpoint:

```text
go test ./cmd -run 'TestInternalWorkerAdapter|TestCLIInternalWorkerAdapterOwns' -count=1
go test ./pkg/codex -run 'TestSelectPlatformInvoker' -count=1
npm --prefix .aether/ts-host run typecheck
npx tsx --test test/worker-dispatch.test.ts test/go-bridge.test.ts
npm --prefix control-ts run typecheck
npm --prefix control-ts test
```

All passed.

Full verification after the boundary change:

```text
go test ./... -count=1                                      PASS (cmd 251.644s)
go test ./... -race -count=1                                PASS (cmd 333.295s)
go vet ./...                                                PASS
go build ./cmd/aether                                       PASS
GOOS=windows GOARCH=amd64 go build ./cmd/aether             PASS
npm --prefix .aether/ts-host test                           PASS (514 tests)
npm --prefix .aether/ts-host run typecheck/build            PASS
npm --prefix control-ts test                                PASS (133 tests)
npm --prefix npm test                                       PASS (11 tests)
npm --prefix npm pack --dry-run                             PASS
goreleaser check                                            PASS
goreleaser release --snapshot --clean                       PASS (6 archives)
staged GoReleaser archive installed through packed npm      PASS
snapshot binary `version`                                   PASS (1.0.42)
git diff --check                                            PASS
```

No public release, tag, publish, push, or commit was performed.

## Remaining Limitations

- Coarse repository-read-only/workspace-write permission profiles are enforced,
  but narrow test/ledger/survey/documentation paths remain behavioral only.
- Supported build adapter responses now bind worker/task identity plus run,
  attempt, manifest, workspace/branch, owner, and provider invocation. Universal
  provider-stream reattachment is still not implemented.
- Advanced native-panel ownership outside the supported adapter path remains
  guidance-enforced.
- Live authenticated Claude/OpenCode/Codex release-candidate smokes remain manual.
- Legacy TypeScript launcher files still exist for regression tests and should be
  deleted after any useful fixture coverage is moved to Go adapter tests.

## Next Implementation Checkpoint

See `14-permission-profile-implementation.md` for the completed permission slice
and `15-durable-build-run-identity.md` for the completed run-binding slice. Next,
prove provider-backed research-to-plan revision without adding another plan store.
