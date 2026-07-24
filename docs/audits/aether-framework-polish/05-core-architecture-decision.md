# Core Architecture Decision

## Options Scorecard

Score: 1 poor, 3 acceptable with major qualifications, 5 strongest. Scores assess the likely target after a disciplined migration, not the amount of existing code.

| Criterion | A. Current Go-centred | B. Go core + adapters | C. Go state/distribution only | D. Replace runtime | E. Split products |
| --- | ---: | ---: | ---: | ---: | ---: |
| Reliability | 2 | 5 | 2 | 2 | 3 |
| Maintainability | 2 | 4 | 3 | 3 | 3 |
| Model integration | 3 | 5 | 5 | 5 | 4 |
| Platform portability | 3 | 5 | 4 | 4 | 4 |
| Testability | 3 | 5 | 2 | 3 | 4 |
| Performance | 5 | 4 | 3 | 3 | 3 |
| Release complexity | 2 | 4 | 3 | 2 | 1 |
| Migration risk | 5 | 3 | 3 | 1 | 2 |
| Contributor accessibility | 3 | 4 | 4 | 4 | 3 |
| Debuggability | 2 | 5 | 2 | 3 | 3 |
| Preserve Aether identity | 5 | 5 | 4 | 4 | 5 |
| Iterative planning/research | 3 | 5 | 4 | 4 | 4 |
| **Total / 60** | **38** | **54** | **39** | **38** | **39** |

## Option Analysis

### Option A: Retain Current Go-Centred Architecture

This has the lowest migration risk and preserves working Go safety code. It fails because "Go-centred" currently includes direct orchestration, platform dispatch, routing, prompt assembly, ceremony, and state transition policy while TypeScript hosts and wrappers implement overlapping behavior. Keeping it does not remove the split brain.

Use from this option: retain the Go code that owns deterministic state, validation, finalization, storage, download verification, and recovery.

### Option B: Deterministic Go Core Plus Agent-Runtime Adapters

This produces the clearest safety boundary. It accepts that model processes and platform-native subagents differ, but requires each to implement the same observable contract. It allows either Go or one TypeScript host to schedule a workflow during migration without allowing that scheduler to commit state directly.

This option best matches Aether's implemented strengths and its own architecture research, while correcting the unfinished boundary.

### Option C: Reduce Go To State And Distribution

This is attractive for fast prompt/workflow evolution. It would move too much completion and recovery policy into platform-native behavior. Aether would recreate the prompt-owned state problem that motivated the Go migration, and parity would remain dependent on model compliance.

### Option D: Replace The Orchestration Runtime

TypeScript or Python would improve SDK access but would not inherently solve false completion, split state, or adapter contracts. A rewrite would discard tested Go logic, create a long migration, and likely repeat the previous language-led boundary failure.

### Option E: Split Aether Into Products

Core/Colony/Adapters is a useful module and package boundary, but separate products now would multiply versioning, installation, documentation, and support while the core is unproven. Use this as internal architecture and future packaging, not three public products in the next release.

## ADR-001: Deterministic Go Core With Capability-Negotiated Adapters

**Status:** Accepted; migration in progress

**Decision owners:** Aether maintainers

**Date:** 2026-07-21

### Context

Aether has useful deterministic Go safety machinery and real model dispatch. It also has direct Go lifecycle loops, `.aether/ts-host`, `control-ts`, Claude/OpenCode wrappers, and platform agent files that overlap. Completion, state, and presentation can disagree. The public release train is version-skewed. Current platform permissions do not enforce caste claims.

The product needs persistent, portable lifecycle truth while allowing platforms to use native subagents, tools, and streaming.

### Implementation Checkpoint: 2026-07-22

The first adapter boundary is implemented for subprocess execution. Direct Go
lifecycles already use `pkg/codex`; `.aether/ts-host` now delegates provider
selection, preflight, prompt assembly, launch, and terminal-result parsing to the
hidden Go `internal-worker-adapter` command. Explicit pins are absolute.
`control-ts` is private, unavailable, and fail-closed.

Platform-native wrapper panels remain a separate adapter mode selected by using a
dry-run manifest. They may launch native workers for that run, while Go still owns
the manifest and finalizer. This checkpoint does not yet enforce permissions,
persist a universal execution-owner journal record, or implement reattach/cancel
handles from the full target adapter contract.

### Decision

1. **Go is the only lifecycle authority.** It owns a versioned append-only journal, materialized state, locks, migrations, idempotency, evidence ledger, transition policy, recovery, packaging, and release verification.
2. **Agent runtimes are adapters.** An adapter performs preflight, declares capabilities, launches/cancels workers, streams typed events, enforces available permissions, and returns structured terminal results.
3. **One orchestration owner is active per run.** During migration this may be direct Go or `.aether/ts-host`, chosen explicitly in the run record. `control-ts` is not a production owner while its adapters are stubs.
4. **Orchestrators cannot write canonical state.** They request manifests and submit results to Go finalizers/transition APIs.
5. **Editable assets define prompts and declared policies, not lifecycle truth.** Policy files are schema-validated and evaluated by Go; prose alone cannot authorize advancement.
6. **All other state is projection.** Queen, context, handoff, session, plan/report Markdown, ceremony, dashboard, and SQLite are rebuilt from journal/evidence records.
7. **Platform parity is contract parity, not presentation parity.** Every adapter publishes supported capabilities and limitations. Unsupported guarantees cause a visible block or explicit degraded mode, never silent equivalence.

### Adapter Contract

```text
preflight(request) -> availability + identity + version + auth + limitations
capabilities() -> sandbox/read/write/network/subagents/stream/cancel/worktree
dispatch(manifest, worker, permission_profile) -> run_handle
events(run_handle, cursor) -> typed ordered events
cancel(run_handle, reason) -> acknowledged terminal intent
collect(run_handle) -> terminal result + raw transcript hash + claims + usage
health() -> adapter/runtime status
```

Required invariants:

- Explicit platform pins are absolute.
- A terminal `completed` result requires a valid result schema and adapter process success.
- Empty file claims are not implementation evidence when implementation is required.
- Cancellation/timeout cannot become completion.
- Read-only roles must use an enforced read-only profile or be reported unsupported.
- Every result is bound to run ID, manifest hash, workspace fingerprint, phase, task, and agent definition version.

### Canonical State Contract

The journal records:

- Colony/session/goal/plan revisions.
- Decisions, assumptions, research conclusions, and supersession links.
- Workspace/branch/commit/tree fingerprints.
- Transition start/pause/abort/commit.
- Worker selection rationale and permission profile.
- Process status and heartbeats.
- Claims, artifact hashes, commands, test results, model reviews, and human approvals.
- Signal creation, resolution, consumption, and outcome.
- Knowledge observation, evidence, validation, contradiction, decay, and revocation.

`COLONY_STATE.json` remains for compatibility as a generated snapshot until consumers migrate.

### Consequences

Positive:

- False completion becomes structurally harder.
- Interrupted work can resume result collection or roll back one transaction.
- Platforms can differ honestly without corrupting shared state.
- Model/provider integrations can evolve without moving storage/recovery logic.
- Black-box tests can run against a fake adapter implementing the same contract.
- Queen and ant identity remain visible without owning hidden state.

Negative:

- Significant migration work across Go, `.aether/ts-host`, wrappers, fixtures, and generated assets.
- Temporary compatibility adapters and projection rebuilds are required.
- Some current commands and castes will disappear or move behind advanced namespaces.
- Adapter capability gaps will make current "parity" look worse before it becomes accurate.
- Maintaining Go plus one host language remains a two-language contributor cost.

### Opportunity Cost

For at least one major stabilization milestone, Aether will not add platforms, castes, skills, dashboards, marketplace features, or richer ceremony. Engineering time moves from visible expansion to deleting duplicate paths, migrating state, and writing black-box release gates. The short-term product will look smaller. The benefit is that its remaining features become credible.

### Rejected Alternatives

- All-Go is rejected because platform-native orchestration and prompt behavior should not be embedded in the safety kernel.
- Prompt/wrapper-owned orchestration is rejected because lifecycle safety cannot depend on model compliance.
- A full TypeScript/Python rewrite is rejected because language replacement does not solve ownership and discards tested Go behavior.
- Public multi-product split is deferred because release integrity is already fragmented.

### Migration Sequence

1. Specify schemas and invariants; snapshot current state and release behavior.
2. Add journal/evidence APIs behind existing commands; dual-write and compare projections in tests.
3. Implement explicit adapters around current Go dispatchers.
4. Route one vertical slice: init -> plan -> build one task -> verify -> resume.
5. Make state transition results authoritative; render existing outputs from them.
6. Port `.aether/ts-host` to the adapter API or retire it if direct Go orchestration proves sufficient.
7. Quarantine/delete `control-ts` stubs and duplicate event code.
8. Thin wrappers to platform invocation and UX only.
9. Migrate Queen/context/session/memory projections.
10. Remove dual writes only after N-1 upgrade and rollback tests pass.

### Proof Of Decision

The ADR is accepted only after the release acceptance suite in `09-release-acceptance-suite.md` passes for one real adapter and the deterministic fake adapter, with no false completion, no state loss on interruption, and a successful N-1 migration.
