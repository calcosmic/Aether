# Durable Build Run Identity

Date: 2026-07-22

Status: implemented and locally verified through the complete source, race,
adapter, package, and staged-release matrix. No public release was performed.

## Verdict

Every current build worker is now tied to one immutable attempt, manifest,
workspace, execution owner, and provider invocation. A stale result cannot be
accepted as current work, parallel provider processes no longer overwrite one
another's registry entries, and `--force` cancels only the exact run it is
superseding.

For beginners: every build now gets a serial number. Each worker result must carry
that serial number, and Aether refuses results from an older build or another
branch.

## Durable Contract

`pkg/codex/execution_binding.go` defines the versioned binding:

| Field | Purpose |
| --- | --- |
| `run_id` | Random identity for one build execution |
| `attempt_id` | Durable build-attempt journal identity |
| `manifest_sha256` | Hash of the exact Go-authored dispatch manifest |
| `workspace_fingerprint` | Canonical checkout, Git common directory, and branch identity |
| `execution_owner` | Runtime or host surface authorized to coordinate the attempt |

The workspace fingerprint intentionally excludes file contents and `HEAD`.
Workers must be able to edit and commit during a run. Switching branches or
checkouts invalidates the binding; ordinary worker edits do not.

## State And Process Flow

```text
Go creates build attempt + run_id + workspace fingerprint
  -> Go writes immutable manifest and its digest
  -> host/native runtime sends exact execution_binding per worker
  -> Go allocates a unique provider_run_id
  -> worker journal records dispatch before provider launch
  -> provider PID is registered under the exact run
  -> terminal result is validated and hashed before host response/wave completion
  -> all wrapper workers terminal: durable completion packet is staged
  -> finalizer commits or resume points at that exact packet
```

The attempt journal is under
`.aether/data/build/phase-N/attempts/attempt-*.json`. Provider processes are
registered in `.aether/data/worker-processes.json`. Registry read-modify-write
uses `pkg/storage` cross-process locking, so concurrent adapter processes cannot
lose each other's PIDs.

## Failure Semantics

- A repeated wrapper request returns its already hashed terminal result instead
  of launching the same worker again.
- A different run, attempt, manifest digest, checkout, branch, or owner fails
  before provider launch or terminal-result acceptance.
- `aether build N --force` asks the process tracker to terminate only PIDs whose
  persisted binding has the superseded `run_id`. Cancellation failure blocks the
  new attempt.
- `status` and `resume` expose run identity, manifest digest, worker status counts,
  active provider count, and the exact next recovery command.
- Wrapper-hosted builds stage a completion packet automatically after every
  authorized worker has a terminal result. Resume can finalize that packet
  without rerunning workers.
- Native builds record terminal workers after in-repo observation or worktree
  reconciliation and before the wave returns.

## Evidence

| Invariant | Evidence |
| --- | --- |
| File edits do not change workspace identity; branch changes do | `pkg/codex/execution_binding_test.go` |
| Exact-run cancellation leaves other runs alive | `pkg/codex/process_tracker_test.go` |
| 32 concurrent registry upserts preserve every PID | process-tracker test under `-race` |
| Plan-only manifest and journal carry the same binding | `cmd/build_execution_binding_test.go` |
| Repeated adapter request does not redispatch | cached-terminal adapter test |
| Last wrapper result stages exact completion | terminal completion/resume test |
| Stale run and changed workspace fail closed | stale identity and workspace tests |
| Native build journals every terminal worker | native dispatch journal test |
| TypeScript forwards the exact binding | `.aether/ts-host/test/worker-dispatch.test.ts` |
| TypeScript rejects a mismatched echo | different-build-run test |
| Interrupted compiled build can force-resume safely | `TestCLIInterruptedBuildResumesThroughForceRedispatch` |

## Verification

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | Pass |
| `go test ./... -race -count=1` | Pass; `cmd` 322.688s, `pkg/codex` 16.690s |
| Focused workflow golden tests under `-race` | Pass |
| `go vet ./...` | Pass |
| Native build and Windows amd64 cross-build | Pass |
| Compiled binary version smoke | Pass; `1.0.42` |
| TS host typecheck/build/tests | Pass; 516 tests |
| Retired control typecheck/tests | Pass; 133 tests and protected state unchanged |
| npm bootstrap tests/package dry-run | Pass; 11 tests and four intended files |
| `goreleaser check` | Pass |
| GoReleaser snapshot | Pass; six configured targets and checksums |
| Packed npm install from staged GoReleaser archive | Pass |
| Snapshot binary version smoke | Pass; `1.0.42` |

The first full race run exposed concurrent progress callbacks writing through an
unsynchronized observer. The provider boundary now serializes callback delivery,
and the focused plus full race suites pass. A subsequent race run exposed only
elapsed-time variability in workflow golden output; the golden comparison now
normalizes observed and stored durations without changing runtime rendering.

## Remaining Limits

- This is durable identity, result replay, exact cancellation, and recovery
  diagnostics. It is not general stream reattachment to an orphaned provider.
- If a native direct-build parent dies after all workers finish but before the
  aggregate lifecycle commit, Aether preserves the worker evidence but currently
  requires a force retry. It does not yet reconstruct the direct build commit.
- If the Go adapter itself dies while its provider remains alive, Aether can
  detect and cancel the provider but cannot recover the lost stdout stream.
- Plan and continue dispatches use the Go adapter and permission contract but do
  not yet have build-equivalent attempt/worker journals.
- The fingerprint protects checkout and branch identity, not repository content
  freshness. The immutable manifest, plan revision, artifact evidence, and
  finalizer freshness checks cover the other acceptance boundaries.

These limits must remain visible in release claims. “Crash-safe build recovery”
is defensible for the wrapper completion path. “Universal worker reattachment”
is not.

## Next Checkpoint

Prove the existing iterative planning/research loop through real provider-backed
Scout, Oracle, and Route-Setter execution. Add typed assumption and decision
impact links to the existing plan revision model without creating another state
store. Native direct-build commit reconstruction should be handled only if that
journey proves it is required for the supported secondary Codex path.
