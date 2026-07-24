# Core Lifecycle Truth: Implementation Record

**Date:** 2026-07-21  
**Scope:** First implementation slice from Stage 0 and Stage 1 of the polish roadmap.

## Plain-English Result

Aether now refuses to call a phase successful in five cases that previously undermined trust: a worker failed or timed out, a real provider reported success without changing anything in an implementation phase, required verification evidence is missing or stale, every verification command was skipped, or two isolated workers changed the same path. CLI commands that render an error also return a failing process exit code.

In plain English: a worker saying "done" is no longer enough. For evidence-aware plans, Aether records exactly which files and checks prove each criterion, fingerprints claimed files at build time, and checks those fingerprints again before advancing. If the CLI is killed, the next session can distinguish the abandoned attempt from a build still running in another terminal and give a recovery command that the runtime accepts. Wrapper builds now receive the same durable attempt identity before workers run, and replaying the exact completion cannot commit the phase twice. This is a credibility repair, not completion of the roadmap: old plans without evidence bindings remain compatible but are reported as `legacy_unbound`, and interrupted direct runs are safely redispatched rather than selectively resumed worker by worker.

## Implemented Invariants

### Process Result Agreement

- `cmd/helpers.go` records the error code whenever `outputError` renders an error envelope.
- `cmd/root.go` resets and reads that command-scoped marker around Cobra execution.
- A rendered error can no longer be followed by process exit `0` merely because a legacy handler returned `nil`.
- The existing root renderer still owns final formatting, so returned errors do not produce a second JSON envelope.

### Truthful Build Finalization

- `cmd/codex_build.go` validates every terminal dispatch result before committing `BUILT` or emitting `build_completed`.
- `failed`, `blocked`, and `timeout` results stop finalization and preserve the pre-build lifecycle state.
- Real Claude, OpenCode, and Codex implementation dispatches must show observed file changes. A completed provider response with no changes is treated as a no-op failure.
- Verification-only manifests may complete without production mutation.
- Synthetic and unknown test invokers retain compatibility while still being subject to terminal-status validation.
- Observed failed-worktree records survive lifecycle rollback as orphan evidence.

### Minimum Verification Evidence

- `cmd/codex_continue.go` counts deterministic checks that actually execute.
- If every candidate command is unresolved or skipped, aggregate verification fails with an explicit blocker.
- Individual optional checks may still be skipped when at least one fresh deterministic check ran.

### Criteria-Bound Evidence

- `pkg/colony/colony.go` adds optional `evidence_requirements` to phase and task success criteria.
- Each requirement binds the exact criterion text to repository-relative artifacts and/or named checks: `build`, `types`, `lint`, `tests`, `claims`, or `watcher`.
- `cmd/codex_plan.go` preserves these bindings through worker-plan normalization, canonical colony state, and plan-artifact round trips.
- `cmd/codex_build.go` rejects partial or malformed binding sets and snapshots SHA-256 plus size for each claimed regular file at build finalization.
- Runtime files under `.aether/data`, absolute paths, traversal paths, symlink artifacts, and paths resolving outside the repository cannot count as product evidence.
- Task-level artifacts must be claimed by the matching task, not merely by the aggregate build.
- `cmd/codex_continue.go` compares the current criterion contract to the build manifest and rechecks artifact hashes before advancement. Deleted, changed, unclaimed, or unhashed artifacts block the phase.
- Required named checks must exist and execute successfully. A skipped required check does not pass; `watcher` requires a real non-skipped Watcher result.
- Artifact-only maintenance phases may advance without an arbitrary shell command when their fresh claimed artifacts satisfy every bound criterion.
- Existing plans with criteria but no bindings are classified as `legacy_unbound`. They retain compatibility but do not claim criterion-level enforcement.

### Durable Build Attempts And Interrupted Resume

- `cmd/build_attempt.go` stores one canonical record per attempt under `.aether/data/build/phase-N/attempts/`, plus a small latest-attempt pointer.
- The record includes phase, process ID, owner, selected tasks, checkpoint/manifest/claims paths, original-state SHA-256, planned and terminal dispatches, artifact-fingerprinted claims, failure reason, recovery command, and ordered transition history.
- Direct builds persist `prepared` before projecting `EXECUTING`, `dispatching` before worker launch, terminal worker results before projecting `BUILT`, and finally `built` after the atomic colony-state commit.
- `build-finalize` applies the same terminal-before-state rule to wrapper-mediated external task results.
- Failed commands persist `failed` with `aether build N --force`; force redispatch marks any prior active attempt `interrupted` without deleting its evidence.
- A hard process kill leaves the last durable status active instead of manufacturing a failure or success. `resume` checks the recorded PID: a live process routes to `aether watch`, while a dead process routes to `aether build N --force`.
- `status`, `resume`, and `resume-dashboard` expose the newest relevant attempt even after lifecycle rollback resets `CurrentPhase` to zero.
- Interrupted output remains in the working tree for inspection. A successful retry receives a distinct attempt ID and must still satisfy normal build and criterion evidence rules.

### Bound Wrapper Attempts And Exact-Once Finalization

- `build --plan-only` now creates a durable `awaiting_external` attempt and embeds `attempt_id` plus `attempt_path` in the immutable dispatch manifest without advancing `COLONY_STATE.json`.
- The journal stores a canonical JSON digest and an embedded copy of the issued manifest. A changed, partial-identity, stale-state, or superseded manifest is rejected before claims or lifecycle mutation.
- A second plan-only dispatch is blocked while an attempt is active. Explicit `--force` marks the old attempt `interrupted`, preserves it, and issues a distinct attempt.
- `build-finalize` binds one canonical completion digest to that attempt. Replaying the same packet after a successful commit returns `idempotent:true` without adding state events or journal transitions; a changed packet is rejected.
- Terminal external results can be finalized again after interruption without rerunning workers. If `BUILT` was committed but the last journal update failed, replay reconciles the journal only when the final manifest, persisted claims, attempt identity, and completion digest all agree.
- The TypeScript host keeps the Go-issued manifest immutable while enriching cloned worker briefs with playbooks, Hive context, and iteration feedback.
- Confidence iterations may dispatch workers multiple times, but the host invokes `build-finalize` exactly once with the final completion packet. Earlier temporary packets are cleaned up only after the next iteration exists.
- Active Orchestrator boundary guidance prevents both attempt creation and TypeScript worker dispatch; the enforced next action remains `aether discuss`.

### Worktree Ownership

- `cmd/codex_build_worktree.go` maintains one path-ownership registry across all waves in a build.
- A worker must claim its touched paths before synchronization into the root checkout.
- The second worker to claim an already-owned path fails with the path and current owner in its blocker.
- The losing worktree is retained as orphan evidence and its content is not synchronized.

### Initial Compiled-CLI Harness

- `cmd/blackbox_harness_test.go` builds and invokes the actual `aether` executable in a disposable repository and HOME.
- Runtime roots, cache roots, output mode, and user configuration are isolated.
- The harness asserts stdout, stderr, process exit status, JSON envelope, state mutation, and source-checkout stability.
- The first regression proves `flag-add` without a title exits nonzero while `version` exits zero.
- A test-only external adapter executable exercises the production Codex dispatcher, subprocess, prompt, result-file, parsing, timeout, and finalization path without calling a model.

### Deterministic External Adapter Matrix

`cmd/testdata/adapter-fixture` provides six controlled process modes:

| Mode | Fixture behavior | Required Aether result |
| --- | --- | --- |
| `success` | Builder writes and claims `app.txt`; review workers return valid empty claims | Exit 0 and phase reaches `BUILT` |
| `no-op` | Every worker reports completed without project mutation | Nonzero exit and phase does not reach `BUILT` |
| `crash` | Worker process exits 17 | Nonzero exit and phase does not reach `BUILT` |
| `malformed` | Worker writes invalid final claims | Nonzero exit and phase does not reach `BUILT` |
| `timeout` | Worker sleeps beyond its deadline | Nonzero exit and phase does not reach `BUILT` |
| `partial` | Worker changes a project file and exits 18 | Nonzero exit, phase does not reach `BUILT`, partial file remains inspectable |

The adapter records process IDs outside the project repository. This proves a separate executable ran while preventing test instrumentation from being mistaken for worker output.

## Second Remediation Slice

### Mandatory New-Plan Evidence And Legacy Migration

- `Plan.evidence_policy` distinguishes `not_required`, `legacy_unbound`, and `bound_v1`.
- Every newly accepted real, external-finalized, or explicit synthetic plan must bind every phase and task criterion to supported evidence before canonical state changes.
- Existing colonies are normalized to an explicit legacy policy. `migrate-state` persists that classification, keeps the original state bytes exactly under `backups/`, and emits a concrete rollback command.

### Crash-Safe Wrapper Completion

- `build-completion-stage` validates the Go-issued attempt and immutable manifest, writes the accepted packet under the attempt path, records its digest/path, and refuses a changed restage.
- The TypeScript host and Claude/OpenCode build instructions stage before `build-finalize` and use only the returned Go-owned path.
- `resume` detects a staged packet and offers exact finalization instead of redispatching already-finished workers.

### Safe Hive Default

- Automatic cross-project retrieval and promotion are off by default.
- Worker retrieval must use policy-aware `hive-read --for-worker`.
- Wrapper/playbook promotion must use `hive-promote --automatic`; the runtime skips it unless `AETHER_HIVE_POLICY=promote`.
- Manual inspection and explicitly invoked manual promotion remain available. This is quarantine, not proof that the Hive trust model is sound.

### Honest Platform Contract

- `pkg/codex/platform_contract.go` publishes versioned capability levels and mechanisms for state lifecycle, worker dispatch, named caste routing, structured completion, progress, permissions, and native command surfaces.
- `command-guide` returns the contract for the requested platform.
- The contract states that Claude caste permissions are not enforced under `bypassPermissions`, OpenCode delegation is indirect, and Codex lacks a native slash-command surface and per-caste sandbox.

### Compiled Core And Upgrade Journeys

- `TestCLICompiledInstallToSealJourney` proves source-candidate install, setup, discussion, Oracle artifacts, bound planning, provider-backed workers, criterion verification, resume, and seal through real process boundaries.
- `TestCLICompiledInstallUpdateMigrationContract` proves update preservation, migration classification, byte-exact backup, rollback guidance, installed platform assets, and binary/hub version agreement.
- These journeys exposed and repaired two additional lifecycle contradictions: read-only Discovery phases were impossible under the implementation no-op gate, and unsealed `COMPLETED` colonies resumed to entomb instead of seal.

## Third Remediation Slice: Versioned Iterative Planning

### Immutable Revision History With One Execution View

- `Plan.Phases` remains the single current execution view; `Plan.ActiveRevisionID` and `Plan.Revisions` add immutable accepted-plan snapshots rather than a second mutable planner store.
- An initial accepted plan records revision 1. A pre-revision colony receives one `legacy_import` baseline when its first revision is accepted; migration does not invent earlier history.
- A revision preserves the completed prefix exactly, including nil/empty JSON distinctions, and replaces only the unfinished suffix. Replacement phase/task IDs and internal dependency references are renumbered after the preserved prefix.
- Completed phases after unfinished work are rejected as inconsistent. A fully completed plan must be sealed or replaced by a new goal, not rewritten.
- The revision records parent ID, reason type, reason, evidence paths, evidence-content hash, planning evidence hash, planning run, plan hash, and preserved/superseded/replacement phase IDs.

### Atomic And Stale-Safe Finalization

- Planning manifests bind the active revision ID and full mutable plan-state hash before workers run.
- Research and verification-failure revisions require at least one repository-relative regular evidence file. Symlink escape, traversal, directory, missing-file, and absolute-path inputs are rejected.
- The manifest also fingerprints evidence contents. If the plan, revision, or evidence changes while Scout/Route-Setter work is in flight, finalization rejects the packet without mutating `COLONY_STATE.json` or spawn history.
- The final accepted plan, active revision, revision history, current phase, lifecycle state, and event are committed through one locked `UpdateJSONAtomically` transition.
- An active build attempt blocks revision. Build manifests bind the current plan revision and plan-state hash; a packet from a superseded revision cannot finalize.

### Research, Verification, Resume, And Platform Wiring

- Completed Oracle runs expose an explicit, non-automatic revision option using the real `.aether/oracle/synthesis.md` artifact.
- Blocked verification and review expose an explicit revision option linked to their reports. Aether does not automatically rewrite the plan merely because a gate failed.
- `status`, `resume`, context capsules, colony-prime context, and plan ceremony show the active revision and reason.
- `--revision-type`, `--revision-reason`, and repeatable `--revision-evidence` flow through Cobra, the TypeScript host, command guidance, Claude/OpenCode wrappers, and the Codex build-cycle skill.
- Planning workers receive the immutable completed boundary and required evidence paths, while Go remains the only component allowed to activate the replacement plan.

## Regression Evidence

Added or strengthened tests cover:

- Worker `failed`, `blocked`, and `timeout` results cannot advance a phase.
- A provider-backed no-op implementation cannot advance a phase.
- All-skipped verification cannot pass.
- A verification path with explicit runnable build/test commands still passes without model review when the intended deterministic checks succeed.
- Two worktree workers changing the same untracked path cause an ownership conflict, preserve one accepted result, retain the losing worktree, and leave the phase unbuilt.
- A real compiled CLI error has an error envelope and nonzero process exit.
- A compiled CLI build accepts a valid external write and rejects external no-op, crash, malformed-result, timeout, and partial-write outcomes.
- Planner-produced phase and task evidence requirements survive normalization and canonical round trips.
- A bound artifact that is fresh, claimed, and unchanged passes criterion verification.
- A missing, post-build-modified, unclaimed, unhashed, runtime-state, or symlink-escaped artifact cannot satisfy a criterion.
- A required skipped check blocks its criterion, while a valid artifact-only maintenance phase can pass without shell-command ceremony.
- A compiled CLI `build` plus `continue` journey completes only while `app.txt` matches its build-time fingerprint; deletion or modification leaves the phase unadvanced with a criterion-specific blocker.
- Build-attempt history preserves `prepared -> dispatching -> terminal -> built` with claims and artifact hashes.
- A compiled CLI hard-kill journey preserves partial worker output, leaves lifecycle state unadvanced, exposes the interrupted attempt on resume, recommends the accepted force route, preserves the old attempt as `interrupted`, and records the retry as a distinct `built` attempt.
- Resume refuses to recommend redispatch while the recorded build PID is still alive and routes the user to `aether watch` instead.
- Plan-only manifests persist an `awaiting_external` attempt, block accidental duplicate dispatch, support explicit supersession, and reject stale completion packets.
- Wrapper manifests remain byte-equivalent at the completion boundary even though cloned dispatch briefs receive Hive, playbook, and iteration context.
- Two or three confidence iterations produce exactly one `build-finalize` call.
- Exact completion replay is idempotent; changed completion replay is rejected.
- A terminal external journal resumes from the saved completion without worker redispatch, and a `BUILT` state with a lagging terminal journal reconciles only from matching final evidence.
- A compiled CLI plan-only/finalize journey proves attempt binding, exact replay, and tamper rejection through real process boundaries.
- A compiled revision journey preserves completed phase JSON exactly, accepts a research-bound replacement, restores the active revision in a new process, binds the next build to it, and proves the completed task is not selected again.
- External finalization rejects a revision after its base plan changes and leaves canonical state byte-identical.
- Revision validation rejects evidence that changes after planning dispatch.

## Verification Run

| Command | Result |
| --- | --- |
| Focused lifecycle regressions with `-count=1` | Pass |
| Focused lifecycle regressions with `-race -count=1` | Pass |
| `go test ./cmd -count=1` | Pass |
| `go test ./... -count=1` | Pass |
| `go test ./... -race -count=1` | Pass |
| `go build ./cmd/aether` | Pass |
| Windows amd64 cross-build of `./cmd/aether` | Pass |
| `go vet ./...` | Pass |
| `goreleaser check` | Pass |
| `AETHER_RELEASE_VERSION=1.0.42 goreleaser release --snapshot --clean` | Pass; six archives plus `checksums.txt` report `1.0.42` |
| Snapshot binary smoke (`aether version`) | Pass; reports `1.0.42` |
| Actual GoReleaser archive installed through packed npm | Pass; package, metadata, archive, checksum, binary, and hub agree |
| `git diff --check` | Pass |
| TypeScript host `npm run typecheck` | Pass |
| TypeScript host `npm test` | Pass (515 tests) |
| TypeScript host `npm run build` | Pass |
| npm bootstrap `npm test` | Pass (11 tests) |
| npm bootstrap `npm pack --dry-run` | Pass (4 intended files, version `1.0.42`) |
| GoReleaser snapshot build after plan-revision repair | Pass; six target binaries built as `1.0.42` |

Latest revision-specific rerun on 2026-07-22: the compiled restart journey,
changed-base and changed-evidence rejection, `go test ./... -count=1`,
`go test ./... -race -count=1`, `go vet ./...`, native and Windows amd64 builds,
TypeScript typecheck/515 tests/build, npm 11 tests/package dry-run, GoReleaser
snapshot build, snapshot-binary version smoke, and `git diff --check` all pass.

### Memory Consolidation Follow-Up

The previously failing `pkg/memory/TestPipeline_Consolidation` used a fixed `2026-02-25` timestamp while describing it as 40 days old. Git history shows an earlier repair replaced an older expired date with that value, guaranteeing the failure would recur as wall-clock time advanced. The fixture now calculates a real relative 40-day age.

Production behavior was not weakened: a new regression proves a 365-day-old instinct is archived and cannot become Queen-eligible even when it has high stored confidence and four legacy applications. The complete uncached and race suites now pass.

## Deliberate Limits

- Build rollback restores lifecycle state but does not undo partial writes already made in the shared root checkout.
- Worktree conflict handling is first-accepted-writer wins. It detects and preserves the conflicting result; it does not pre-allocate ownership or atomically reconcile a whole wave.
- No-op enforcement remains as a coarse backstop for recognized production platform adapters. Bound plans now add artifact- and check-specific proof, but legacy unbound plans still rely on aggregate verification.
- Evidence requirements match exact files. Directory trees, globs, generated bundles, semantic UI behavior, and other richer evidence types are not yet modeled.
- Newly accepted real and synthetic plans require complete `bound_v1` evidence bindings. Pre-existing plans remain explicitly `legacy_unbound`; migration does not invent evidence they never had.
- Interrupted direct builds are safely redispatched as a new attempt; Aether does not yet reuse their terminal results to auto-finalize or resume only unfinished workers.
- Wrapper plan-only dispatch is journaled before external agents run. The host stages the accepted completion under the Go-owned attempt path before finalization, so resume can replay the exact packet without redispatch.
- The compiled-CLI fixture now covers source-candidate install, discussion, one Oracle pass, bound planning, all build/continue phases, resume, seal, update preservation, migration backup/rollback guidance, platform assets, binary/hub agreement, and deterministic research-driven plan revision. The revision journey uses synthetic replacement planning; real Oracle/Scout/Route-Setter provider execution remains an acceptance gap.
- Revisions currently preserve an immutable completed prefix and replace the unfinished suffix. They do not yet model typed assumption/decision IDs or a sparse affected-node graph, so the planner may replace more future work than strictly necessary.
- Snapshot versioning now takes the explicit source-manifest version through `AETHER_RELEASE_VERSION`; CI and release workflows assert the binary version and install the staged archive through the packed npm bootstrap. This proves a local release candidate, not the already-published registry assets.
- Hive is quarantined rather than repaired: automatic worker retrieval and seal promotion are off by default. Evidence provenance, contradiction, stable repository identity, revocation, decay, locking, and retrieval relevance still require redesign.
- Platform contracts now disclose differences, but Claude permissions remain bypassed, OpenCode named routing remains indirect, and Codex has no native slash-command surface or caste-specific sandbox.
- The command-scoped exit marker contains legacy handlers safely, but the long-term design remains typed errors returned through Cobra rather than hidden rendering side effects.

## Next Implementation Slice

1. Select one production orchestration owner, quarantine duplicate success stubs, and move all model/process behavior behind the typed adapter contract.
2. Replace prompt-only caste permission claims with host-enforced profiles or visibly mark the capability unavailable.
3. Prove the revision boundary with real Oracle/Scout/Route-Setter providers and add typed assumption/decision impact links without creating another plan store.
4. Replace first-accepted worktree conflict handling with declared ownership and one atomic whole-wave reconciliation decision.
5. Redesign project/Hive knowledge around evidence IDs, stable repository identity, contradiction, revocation, decay, locking, and opt-in retrieval before re-enabling promotion.
