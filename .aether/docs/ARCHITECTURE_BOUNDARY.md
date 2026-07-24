# Architecture Boundary: Go, Hosts, And Editable Assets

> **Candidate version:** v1.0.42 (unreleased)
> **Last updated:** 2026-07-22
> **Status:** Current production boundary

## The Hard Rule

**Go owns lifecycle truth and subprocess-provider execution. A host may coordinate
one run, but it may not become a second state engine or launch the same work twice.**

For beginners: Aether can ask either its own engine or the current AI tool to run
workers. It must choose one for that run. The engine still checks the result and
is the only component allowed to mark work complete.

## Ownership

| Layer | Owns | Must not own |
| --- | --- | --- |
| Go core | Canonical state, locks, migrations, manifests, evidence policy, transition validation, recovery, finalizers, install/update/release integrity | Platform-specific presentation or editable role prose |
| Go adapter boundary | Provider selection, hard pins, availability/auth preflight, agent resolution, prompt assembly, subprocess launch, timeout/cancellation, typed terminal-result parsing, diagnostic redaction | Lifecycle advancement without finalizer evidence |
| `.aether/ts-host` | Manifest iteration, wave ordering/concurrency, retry coordination, ceremony calls, completion-packet transport | Direct provider CLI launch, provider fallback, prompt truth, direct `.aether/data` writes |
| Platform-native wrappers | Interviews, native visible worker panels, summaries, and completion-packet submission for an explicitly selected wrapper-owned run | Running the Go subprocess path for the same manifest, direct state edits, claiming unsupported guarantees |
| Editable assets | Agent instructions, skills, command metadata, playbooks, and declared policies | Bypassing compiled safety checks or authorizing a state transition by prose |

## Launch Modes

Exactly one launch mode is active for a manifest:

| Entry path | Launch owner | State/finalization owner |
| --- | --- | --- |
| Direct `aether build`, `plan`, `colonize`, or other Go lifecycle | Go runtime adapter | Go |
| Non-dry `aether host ...` | TS host coordinates; hidden Go `internal-worker-adapter` selects and launches each provider subprocess | Go |
| Claude/OpenCode/Codex wrapper using a dry-run manifest and native Task/subagent panels | The selected host platform | Go |
| Explicit simulation in tests | Named deterministic fake only | Isolated fixture Go state |

`AETHER_WORKER_PLATFORM` is a hard pin. A valid but unavailable pin fails with a
structured diagnostic. It never falls through to another installed provider.

## Provider Adapter Contract

The current compiled boundary returns:

- schema version and `execution_owner: "go-adapter"`;
- selected platform and its capability contract;
- the canonical caste `permission_profile` and the selected host's enforcement decision;
- sanitized availability/provider diagnostics;
- the immutable build `execution_binding` and unique provider-run ID when the
  request belongs to a build attempt;
- one terminal worker result bound to worker name, caste, and task ID;
- structured claims, artifacts, handoff, child-spawn claims, tool count, and blockers.

A terminal result is rejected when it is missing, malformed, nonterminal, or
belongs to a different worker/task. Request files must be regular non-symlink
files in an `aether-worker-request-*` system-temporary directory. Production
requests cannot override provider configuration or enable a fake adapter.

For build requests, the binding includes the run, attempt, manifest digest,
workspace/branch fingerprint, and execution owner. The adapter journals dispatch
before launch, persists the provider PID, and hashes terminal results before
responding. A stale binding fails closed. Parallel PID registry updates use the
same cross-process storage lock as other canonical state.

## Permission Boundary

Every generated build and planning dispatch carries a versioned permission
profile. Go recalculates the canonical profile from the caste and rejects stale
or broadened requests before provider launch.

The release-usable profiles are intentionally small:

| Profile | Current castes | Enforced boundary |
| --- | --- | --- |
| `repository_read_only` | Scout, Includer | Repository writes denied by Codex read-only sandbox, Claude plan mode, or OpenCode's restricted router plus no-edit/no-bash subagent |
| `workspace_write` | All other current castes | Writes confined to the project workspace by Codex sandbox, Claude fail-closed native sandbox, or OpenCode external-directory denial |

`scoped_write` and `test_write` are recognized contract values but production
adapters reject them. Probe's test-only rule, survey-only paths, documentation
scope, and review-ledger-only rules remain visible `behavioral_restrictions`
inside an enforced workspace boundary. They are not advertised as path-level
security guarantees.

Claude no longer launches workers with `bypassPermissions`. Workspace writers
use `acceptEdits` with sandboxing enabled, `failIfUnavailable: true`, and the
unsandboxed-command escape hatch disabled. OpenCode uses the infrastructure-only
`aether-worker-router`; Aether rejects a configured alternate primary agent and
attests the shipped target-agent permissions before launch. Codex selects
`read-only` or `workspace-write` per dispatch and no longer grants `CODEX_HOME`
as an additional writable directory.

## Quarantined Paths

- `.aether/ts-host/src/platform-dispatcher.ts` and `prompt-assembler.ts` are
  legacy regression-test fixtures. Production host sources are statically tested
  not to import the TypeScript provider launcher.
- `control-ts` is private and retired. Its adapters report unavailable, dispatch
  throws, and phase execution fails without emitting completion. Any direct
  legacy state/event writes are quarantined below `control-ts/.retired-state` or
  an explicit test temporary path.
- Historical migration documents can explain prior intent, but they are not
  runtime authority.

## Editable Asset Rule

Editable assets remain important, but "editable" does not mean authoritative for
safety. A human may change prompts, skills, routing hints, and presentation. Go
must schema-validate any policy that affects permissions, evidence requirements,
or transitions. Compiled fallbacks are allowed for recovery and safe failure, but
must be versioned and test-visible.

## Required Invariants

1. One launch owner per manifest/run.
2. One canonical state writer: Go.
3. Explicit provider pins never fall back.
4. Simulation requires an explicit test-only switch and is visibly labelled.
5. Provider errors and raw output are redacted before entering user-visible or
   durable summaries.
6. Worker completion cannot advance lifecycle state without Go finalization and
   evidence checks.
7. Unsupported platform capabilities are reported as limited/unavailable, not
   silently described as parity.
8. Retired/legacy control planes fail closed and cannot touch live colony state.
9. Every worker has a canonical typed permission profile; stale or broadened
   requests fail before launch.
10. Behavioral caste restrictions are never described as host-enforced scopes.
11. Every build worker result matches the current run, attempt, manifest,
    workspace/branch, and execution owner.
12. Force redispatch cancels the exact superseded run and stops if cancellation
    cannot be proven.

## Verification

The boundary is exercised by:

- `cmd/internal_worker_adapter_test.go`;
- `cmd/blackbox_harness_test.go` compiled adapter tests;
- `pkg/codex/platform_dispatch_test.go` hard-pin and provider tests;
- `pkg/codex/permission_profile_test.go` profile matrix, sandbox selection, and
  shipped OpenCode permission-attestation tests;
- `cmd/permission_profile_integration_test.go` manifest projection tests;
- `cmd/build_execution_binding_test.go` attempt, replay, terminal, and stale-run
  tests;
- `pkg/codex/execution_binding_test.go` checkout/branch identity tests;
- `pkg/codex/process_tracker_test.go` exact-run cancellation and concurrent PID
  registry tests;
- `.aether/ts-host/test/worker-dispatch.test.ts` delegation/static-import tests;
- `.aether/ts-host/test/go-bridge.test.ts` asynchronous bridge/redaction tests;
- `control-ts` fail-closed and live-state-isolation tests.

Durable build identity, exact cancellation, wrapper result replay, and native
terminal journaling are implemented. This is not universal provider-stream
reattachment: a native parent crash before aggregate commit still requires a
force retry, and a dead adapter cannot recover an orphaned provider's stdout.
Plan/continue do not yet have build-equivalent per-worker journals. Narrow path
profiles remain deferred until they can be enforced rather than inferred from
prompt compliance.

## Cross-References

- Audit ADR: `docs/audits/aether-framework-polish/05-core-architecture-decision.md`
- Implementation evidence: `docs/audits/aether-framework-polish/13-orchestration-owner-implementation.md`
- Build-run evidence: `docs/audits/aether-framework-polish/15-durable-build-run-identity.md`
- Release gate: `.aether/docs/release-readiness-handoff.md`
- Platform capability schema: `pkg/codex/platform_contract.go`
