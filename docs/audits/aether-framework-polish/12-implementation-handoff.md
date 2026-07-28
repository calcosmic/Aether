# Aether Framework Polish: Implementation Handoff

Handoff date: 2026-07-22

Checkout: `ef12c596` (`v1.24`) with a large, intentional, unreleased working tree

## Resume Objective

Continue turning Aether into a dependable AI-development framework. Preserve the
verified Go core and finish iterative planning proof, parallel reconciliation,
knowledge trust, and released-artifact proof before calling Aether polished.

For beginners: the source-tree engine now proves that workers did real work and
can recover after interruption, revise unfinished plans without erasing accepted
work, enforce the supported caste permissions, and bind build workers to one
durable run. The next job is to prove that real research changes real plans,
without adding another planning subsystem.

## Safety Rules For The Next Session

- Do not reset, clean, checkout, or otherwise discard the current working tree.
- Do not assume every dirty file belongs to one change; the checkout was already
  dirty and contains work from several Aether repair passes.
- Do not publish, tag, push, bump a version, or run a real release without explicit
  user approval.
- Do not re-enable automatic Hive retrieval or promotion until its trust model is
  redesigned and tested.
- Treat `docs/audits/aether-framework-polish/00-executive-verdict.md` as the audit
  baseline and `11-core-lifecycle-truth-implementation.md` as the current repair
  ledger.

## Completed And Proven

### Staged Release Coherence

- `.aether/version.json` is the source for local/CI release-candidate versions.
- GoReleaser snapshots accept that version through `AETHER_RELEASE_VERSION` instead
  of inheriting the nearest historical tag.
- CI and the release workflow assemble archives plus checksums, assert the binary's
  exact version, pack npm, and install the actual current-platform archive through
  that package before any real release step.
- The npm bootstrap verifies both the archive checksum and the extracted binary's
  own version before activation.
- Failed checksums and valid-but-wrong-version archives preserve the old binary.
- A retry restores `.previous` state left by interrupted activation.
- `aether migrate-state --rollback <backup>` performs locked, atomic, byte-exact
  restoration and keeps a pre-rollback safety backup.

### Lifecycle Truth

- Provider crashes, timeouts, malformed results, partial writes, and implementation
  no-ops cannot commit a successful build.
- Newly accepted plans require `evidence_policy: bound_v1` and explicit evidence
  bindings. Old plans are classified as `legacy_unbound`; migration does not invent
  missing proof.
- Discovery phases may use durable read-only worker reports. Implementation phases
  still require project artifacts.
- Criterion verification is tied to current artifact fingerprints, so deleted or
  changed evidence cannot advance a phase.

Primary files:

- `pkg/colony/colony.go`
- `cmd/criterion_evidence.go`
- `cmd/codex_plan.go`
- `cmd/codex_plan_finalize.go`
- `cmd/provenance.go`
- `cmd/state_load.go`

### Durable Build Recovery

- Every build attempt has a durable journal and immutable manifest binding.
- Wrapper-generated completion packets are accepted from temporary storage, staged
  under the Go-owned attempt path, and finalized only from that exact path.
- Resume can replay a terminal staged completion without dispatching workers again.
- Exact replay is idempotent; changed or stale packets are rejected.
- A completed but unsealed colony now resumes to `aether seal`, not `aether entomb`.

Primary files:

- `cmd/build_attempt.go`
- `cmd/codex_build_finalize.go`
- `cmd/finalizer_completion_contract.go`
- `cmd/context.go`
- `cmd/recovery_snapshot.go`
- `.aether/ts-host/src/host.ts`
- `.aether/commands/build.yaml`
- Claude/OpenCode build wrappers and the Codex build-cycle skill

### Versioned Plan Revision

- Initial and revised plans have immutable revision snapshots while `Plan.Phases`
  remains the only current execution view.
- `plan --refresh` preserves the completed prefix exactly and replaces only the
  unfinished suffix, with deterministic phase/task/dependency renumbering.
- Every revision records its parent, rationale, evidence paths and bytes, planning
  evidence, plan hash, and preserved/superseded/replacement phase IDs.
- Plan-only packets bind the active revision, mutable plan-state hash, and revision
  evidence-content hash. Changed state or evidence is rejected without mutation.
- Build packets bind the active plan revision, preventing superseded future work
  from finalizing or completed tasks from being selected again.
- Oracle and failed verification expose explicit, non-automatic revision options;
  status, resume, context, wrappers, Codex skills, and the TypeScript host carry
  the same revision contract.

Primary files:

- `cmd/plan_revision.go`
- `cmd/codex_plan.go`
- `cmd/codex_plan_finalize.go`
- `cmd/codex_build.go`
- `cmd/codex_build_finalize.go`
- `pkg/colony/colony.go`
- `.aether/ts-host/src/command-registry.ts`
- `.aether/ts-host/src/host.ts`

### Hive Quarantine

- Cross-project worker retrieval and seal-time automatic promotion are off by
  default.
- `AETHER_HIVE_POLICY=read` permits worker retrieval.
- `AETHER_HIVE_POLICY=promote` permits automatic promotion.
- Manual explicit `hive-promote` remains available; internal automatic calls use
  `hive-promote --automatic` and honor the policy.

Primary files:

- `cmd/hive_policy.go`
- `cmd/hive.go`
- `cmd/context_weighting.go`
- `.aether/ts-host/src/hive-injector.ts`
- `.aether/docs/command-playbooks/continue-advance.md`

### Platform Honesty

- `pkg/codex/platform_contract.go` defines typed capability levels: `proven`,
  `limited`, `unavailable`, and `test_only`.
- `command-guide` exposes the selected platform contract.
- Claude is primary; typed read-only/workspace profiles now use plan mode or
  fail-closed native sandboxing without `bypassPermissions`.
- OpenCode has indirect named routing, but now enters through an attested
  task-only router and explicit external-directory denial.
- Codex is secondary and has no slash-command surface; the Go adapter now selects
  read-only/workspace sandbox mode per caste.
- Narrow test/ledger/survey/documentation scopes remain behavioral and are not
  counted as host-enforced permission isolation.

### Production Subprocess Owner

- `cmd/internal_worker_adapter.go` is the hidden Go-owned boundary for provider
  preflight and one typed worker dispatch.
- `.aether/ts-host` keeps asynchronous wave coordination but delegates provider
  selection, prompt assembly, launch, and result parsing to that Go boundary.
- `AETHER_WORKER_PLATFORM` is a hard pin; an unavailable pinned provider cannot
  fall back to another installed CLI.
- Malformed, nonterminal, or worker/task-mismatched results fail closed.
- `control-ts` is private/retired, returns no success, and writes only to
  package-quarantine or test-temporary state.
- Platform-native Task/subagent panels remain an explicitly selected wrapper
  adapter mode; command guidance forbids dispatching the Go path too.

Primary files:

- `cmd/internal_worker_adapter.go`
- `pkg/codex/platform_dispatch.go`
- `.aether/ts-host/src/go-bridge.ts`
- `.aether/ts-host/src/worker-dispatch.ts`
- `.aether/ts-host/src/host.ts`
- `control-ts/src/retired.ts`
- `.aether/docs/ARCHITECTURE_BOUNDARY.md`

### Durable Build Run Identity

- Every bound build has a random run ID, immutable manifest digest, workspace and
  branch fingerprint, attempt ID, and execution owner.
- Wrapper and native worker launches receive that exact binding plus a unique
  provider-run ID.
- Terminal results are hashed into the attempt journal before the wrapper responds
  or the native wave completes. Repeated wrapper requests return the cached result.
- Provider PID persistence now uses cross-process storage locking; parallel adapter
  processes cannot overwrite one another.
- Force redispatch cancels only the superseded run and refuses to start if exact
  cancellation fails.
- Status/resume expose run, manifest, worker, provider, and recovery evidence.

Primary files:

- `pkg/codex/execution_binding.go`
- `pkg/codex/process_tracker.go`
- `cmd/build_attempt.go`
- `cmd/build_worker_run.go`
- `cmd/internal_worker_adapter.go`
- `cmd/codex_build_worktree.go`
- `.aether/ts-host/src/worker-dispatch.ts`
- `docs/audits/aether-framework-polish/15-durable-build-run-identity.md`

### Compiled Black-Box Proof

`cmd/blackbox_harness_test.go` builds the actual CLI plus a separate deterministic
provider process and proves:

- isolated source-candidate install;
- `lay-eggs -> init -> discuss -> Oracle -> bound plan`;
- real external Builder and Watcher processes across all generated phases;
- build/continue evidence enforcement, resume, and seal;
- update preservation of project Queen content, custom skills, and runtime state;
- byte-exact migration backup and explicit rollback guidance;
- legacy plan classification and installed platform-agent assets;
- source-candidate binary/hub version agreement;
- research-invalidated future-plan replacement, exact completed-phase preservation,
  restart restoration, next-build revision binding, and stale packet rejection.

## Verification At Pause

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | Pass |
| `go test ./... -race -count=1` | Pass |
| `go vet ./...` | Pass |
| Native Go build | Pass |
| Windows amd64 cross-build | Pass |
| TypeScript `npm run typecheck` | Pass |
| TypeScript `npm test` | Pass, 516 tests |
| Retired control `npm test` | Pass, 133 tests; live colony-state SHA-256 unchanged |
| TypeScript `npm run build` | Pass |
| npm bootstrap `npm test` | Pass, 11 tests |
| npm bootstrap `npm pack --dry-run` | Pass, four intended files |
| `goreleaser check` | Pass |
| `AETHER_RELEASE_VERSION=1.0.42 goreleaser release --snapshot --clean` | Pass; archives and checksums produced for every configured target |
| Actual GoReleaser archive installed through packed npm | Pass |
| GoReleaser snapshot build after plan-revision repair | Pass; six target binaries built with configured version `1.0.42`; native snapshot smoke reports `1.0.42` |
| `git diff --check` | Pass |

Latest full rerun after durable build run identity: `go test ./... -count=1`,
`go test ./... -race -count=1`, `go vet ./...`, native build, Windows amd64
cross-build, TypeScript typecheck/516 tests/build, retired control 133 tests with
live-state isolation, npm 11 tests/package dry-run,
GoReleaser snapshot build, snapshot-binary version smoke, and `git diff --check`
all pass. The snapshot release produced all six configured archives and checksums,
and staged archive installation through packed npm passed in this checkpoint.

The first race run for durable identity found a real concurrent progress-observer
write. Provider callback delivery is now serialized. The next full race run found
only host-speed-dependent workflow durations in a golden test; both observed and
stored durations are normalized at the comparison boundary. The final complete
race suite passes (`cmd` 322.688s; `pkg/codex` 16.690s).

The first full Go run found only a stale command-catalog golden. Inspection showed
the intended additions `hive-promote --automatic`, `hive-read --for-worker`, and
the existing `reconcile` command. `cmd/testdata/command_catalog.json` was refreshed,
then the complete normal and race suites passed.

The first full CLI run after the plan-revision repair found only the intended
`plan` command-catalog and visual-output snapshot changes. Those snapshots now
record the new revision flags and active revision line; the complete normal and
race suites passed afterward.

## Current Blocking Finding

The local staged release set is now coherent at `1.0.42`. The already-published npm
and GitHub artifacts have not been replaced or retested from their public URLs, and
no release was published in this session. Public release qualification still needs
an approved tag/publish operation followed by clean-machine registry download tests.

Relevant files:

- `.goreleaser.yml`
- `.github/workflows/release.yml`
- `.aether/version.json`
- `npm/package.json`
- `npm/lib/bootstrap.js`
- `cmd/root.go`

## Next Work, In Order

1. ~~Prove plan revision through real Oracle/Scout/Route-Setter provider execution
   and add typed assumption/decision impact links without creating another plan
   store.~~ Done 2026-07-24: see `16-provider-backed-plan-revision.md`. The
   compiled journey proved the full question-to-verification loop through real
   provider processes; typed links were deliberately not added because the
   journey exposed no missing contract.
2. ~~Replace first-accepted worktree conflict handling with declared ownership and
   an atomic whole-wave reconciliation decision.~~ Done 2026-07-24: see
   `17-declared-worktree-ownership.md`. Tasks declare owned paths up front,
   same-wave declared overlap fails pre-dispatch, cross-wave declared overlap is
   legal (later waves inherit earlier output), and each wave reconciles as one
   atomic decision with the journal written after the decision.
3. Redesign Hive knowledge around evidence IDs, stable repository identity,
   contradiction, revocation, decay, locks, and opt-in retrieval before enabling it.
4. Complete released-artifact Journeys A-H: greenfield, brownfield, difficult bug,
   interruption, research-invalidated plan, memory isolation, parallel conflict,
   and upgrade.
5. With explicit approval, publish one candidate tag and test GitHub plus npm from
   clean machines; do not infer public success from the local staging server.
6. Only after these gates pass, consolidate the beginner command surface and update
   public positioning from alpha toolkit to dependable framework.

## Suggested Resume Commands

```bash
git status --short
sed -n '1,240p' docs/audits/aether-framework-polish/12-implementation-handoff.md
sed -n '1,240p' docs/audits/aether-framework-polish/11-core-lifecycle-truth-implementation.md
go test ./cmd -run 'TestCLICompiled(InstallToSealJourney|InstallUpdateMigrationContract)' -count=1
go test ./cmd -run '^TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild$' -count=1
git diff --check
```

Run the full normal and race suites again after production changes. They take
several minutes because the black-box tests compile binaries and create isolated
colonies.

## Completed Checkpoint: Production Subprocess Owner

Direct and TS-host subprocess execution now uses the Go adapter boundary. Explicit
platform pins are hard, unavailable providers fail honestly, production TS-host
sources cannot import the legacy provider launcher, and `control-ts` is
private/fail-closed with isolated state. Platform-native wrapper panels remain an
explicit alternative launch mode, never a second launch for the same manifest.

## Completed Checkpoint: Typed Permission Profiles

Build and planning manifests now carry canonical typed permission profiles. Go
rejects missing, stale, or broadened adapter requests. Scout and Includer receive
`repository_read_only`; current write-capable castes receive `workspace_write`.
Codex selects the matching sandbox, Claude uses plan or fail-closed native
sandboxing without `bypassPermissions`, and OpenCode uses an attested task-only
primary router plus explicit external-directory denial.

`scoped_write` and `test_write` are recognized but rejected. Probe test-only,
survey-only, documentation-only, and review-ledger-only rules remain visible
behavioral restrictions inside the workspace boundary, not security claims.

## Completed Checkpoint: Durable Build Run Identity

Build manifests, worker requests, provider processes, and terminal results now
share one immutable execution binding. Exact-run cancellation, cached wrapper
results, concurrency-safe PID persistence, native terminal journaling, and
status/resume diagnostics are implemented. The honest limits are recorded in
`15-durable-build-run-identity.md`: native aggregate commit reconstruction and
general orphaned-stream reattachment are not implemented.

## Completed Checkpoint: Provider-Backed Plan Revision

The research-to-plan loop is proven through real provider execution:
`TestCLIProviderBackedPlanRevisionJourney` drives the compiled CLI plus the
deterministic provider fixture through Oracle evidence production, real Scout
and Route-Setter planning (`dispatch_mode: real`, `plan_source:
worker-artifact`), immutable research-bound revision activation with exact
completed-prefix preservation, restart recovery, provider-backed build, and
criteria-enforced verification to `COMPLETED`. Typed assumption/decision
impact links were not added: the journey exposed no missing contract in the
existing revision record, and the checkpoint rule forbids speculative
structure. One behavioral finding is recorded in
`16-provider-backed-plan-revision.md`: keyword phase-mode inference turns
planner prose containing words like "research" into discovery phases, which
then dispatch research workers instead of builders.

## Completed Checkpoint: Declared Worktree Ownership And Atomic Wave Reconciliation

Worktree-mode builds now declare per-task path ownership from evidence
artifacts and single-path hints. Same-wave declared overlap fails before any
worker runs; cross-wave declared overlap is legal because later worktrees
inherit earlier synced output. Each wave reconciles as one atomic decision:
a conflict-free wave syncs in deterministic dispatch order, and any
ownership violation rejects the whole wave — nothing syncs, violators fail
with paths and owners named, innocent workers are blocked, and all wave
worktrees are preserved as orphan evidence. Terminal results are journaled
only after the decision. See `17-declared-worktree-ownership.md`.

## Definition Of The Next Safe Checkpoint

Redesign Hive knowledge around evidence IDs, stable repository identity,
contradiction, revocation, decay, locks, and opt-in retrieval before
enabling automatic retrieval or promotion. The current quarantine
(`AETHER_HIVE_POLICY=off` by default) stays in place until the trust model
is designed and tested.

No tag, publish, release, or intentional version bump was performed before
this handoff.
