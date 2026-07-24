# Release Acceptance Suite

## Gate Question

> Does this exact released set of Aether artifacts install, perform its advertised core workflow through real and deterministic adapters, preserve truthful state through failure, and upgrade without loss?

The suite runs against a release candidate tarball/manifest and npm package, not the source tree's embedded assets by default. A source-only pass cannot promote a release.

## Implemented Source-Candidate Slice

The repository now has a compiled black-box harness in `cmd/blackbox_harness_test.go`. It builds the actual CLI and a separate deterministic provider executable, isolates HOME/repository/cache state, and currently proves:

- error envelope and process exit agreement;
- provider write, no-op, crash, malformed output, timeout, and partial-write handling;
- criterion-bound fresh artifact verification and changed/deleted evidence rejection;
- hard-kill recovery with distinct force-redispatch attempts;
- bound wrapper manifest finalization, exact replay, and tamper rejection;
- complete source-candidate install through Oracle, plan, all build/continue phases, resume, and seal;
- install/update preservation, byte-exact migration backup, explicit rollback command, legacy evidence classification, platform-agent installation, and binary/hub version agreement;
- a packed npm candidate installing the actual current-platform GoReleaser archive from the staged release set;
- checksum and embedded-binary version rejection without replacing the previous binary;
- recovery from the filesystem state left by interrupted binary activation;
- N-1 update, migration, executable rollback, and preservation of project Queen and custom skill content; and
- a compiled research-invalidated revision that preserves completed phase bytes, records hashed evidence and rationale, survives restart, binds the next build, and rejects stale state/evidence packets.
- a compiled Go adapter boundary that owns subprocess provider selection/launch,
  rejects malformed or mismatched terminal results, honors hard provider pins,
  and is called asynchronously by the TS host; and
- a retired control-plane suite that fails closed and leaves the live colony
  state byte-identical.

This is necessary but not public-release qualification. It does not download from an actual GitHub release, install through the public npm registry, exercise signatures/attestations, hard-kill every activation boundary, prove narrow test/ledger/survey write scopes, or run one live smoke per release-qualified adapter. Typed repository-read-only/workspace-write profiles are now deterministic and tested locally.

## Harness Architecture

Each test gets:

- A fresh temporary HOME, Git repository, Aether hub, binary directory, adapter config, and network fixture.
- A release manifest containing binary/companion/npm/schema versions and digests.
- A deterministic contract adapter that can write controlled files, emit malformed results, delay, time out, crash, or ignore cancellation.
- At least one live supported adapter smoke job with a bounded token/time budget.
- A filesystem monitor that fails any write outside the fixture root.
- A process registry and kill hooks for each lifecycle boundary.
- Captured stdout, stderr, exit status, ordered events, state revisions, Git fingerprints, file hashes, worker transcripts/claims, and verification evidence.

Release CI must use `go test ./... -count=1`, `go test ./... -race -count=1`, TypeScript typecheck/tests in isolated roots, `go vet`, GoReleaser validation/build, archive member inspection, checksum verification, npm pack/install, and the black-box matrix.

## Required Fixtures

1. **Greenfield notes**: ambiguous goal, no files.
2. **Brownfield Go CLI**: Cobra command patterns, tests, history, deliberate technical debt.
3. **Difficult selection bug**: seeded provider-pin bug with a known failing regression test.
4. **Research contradiction**: initial dependency assumption contradicted by local evidence/source fixture.
5. **Memory pair**: tabs-only and spaces-only projects plus unrelated third project.
6. **Parallel overlap**: two tasks share tracked and untracked paths.
7. **Upgrade N-1**: initialized colony with active/completed phases, local Queen, local skill, signals, Oracle artifacts, dirty unrelated file.
8. **Malicious repository**: prompt-injection docs, symlink/path traversal, fake secrets, shell metacharacters, untrusted skill.

## Black-Box Test Groups

### A. Release Set Integrity

1. Build every supported OS/arch archive.
2. Verify archive name and internal binary member.
3. Verify checksum/signature/attestation.
4. Pack exact npm candidate and install it in a clean environment.
5. Assert npm, release tag, binary, companions, schema and manifest versions agree.
6. Interrupt before and during atomic activation; previous version must remain runnable.
7. Roll back after post-install smoke failure.

### B. Initialization And Planning

1. Initialize greenfield and brownfield fixtures idempotently.
2. Capture goal, typed clarifications, decisions, unknowns and workspace baseline.
3. Require colonization citations to actual files/tests/history.
4. Run bounded research and require typed conclusion/decision link.
5. Accept a plan where every node links rationale, dependency, artifacts, criteria and verification.
6. Interrupt planner after artifact write; resume and accept without data loss.

### C. Worker Execution And Evidence

1. Assert every visible worker has a process/run ID and adapter record.
2. Assert explicit platform pin launches only that adapter.
3. Assert permission profile is enforced or dispatch is blocked.
4. Test success with expected changes and fresh tests.
5. Test positive worker prose with zero changes.
6. Test irrelevant changes, missing files, stale output and malformed claims.
7. Test docs-only/test-only/external/human-review phase policies.

### D. Failure, Recovery, And Resume

Kill at:

- install download, extract, migrate, activate;
- plan worker start, artifact write, result collection, finalizer;
- builder before write, after partial write, after write before claims;
- verification command and model review;
- state commit and projection rebuild.

For each kill, assert no false success/advance, idempotent retry, preserved evidence, no duplicate completed work, and exact next action. Corrupt canonical snapshot and rebuild from journal; no plan/decision loss is allowed.

### E. Replanning

After completing an unaffected phase, introduce evidence contradicting a future assumption. Require a new plan revision, preserved completed evidence, superseded affected tasks, updated dependencies, reason/source, and no re-execution of completed nodes.

### F. Knowledge And Signals

- Duplicate, contradict, validate, stale, revoke and decay project observations.
- Attempt fake multi-repo confirmation and incompatible-project retrieval.
- Concurrently promote/revoke to test locking.
- Inject conflicting FOCUS/REDIRECT and require visible resolution/pause.
- Expire a signal without housekeeping and require all readers to agree.
- Confirm consumed signal appears in selection/decision explanation.

### G. Parallelism

- Non-overlapping changes merge and preserve both hashes.
- Same tracked file, same untracked file, rename/delete and generated-file overlaps pause with both results preserved.
- Killed worker leaves recoverable worktree and no accepted merge.
- Shared-tree mode is reported serial for writing workers.

### H. Cross-Platform Contract

Run the same manifest through deterministic adapter fixtures for Claude, OpenCode and Codex, plus a live smoke for every release-qualified platform. Compare required invariants, not output prose. Unsupported native subagent/read-only/cancel capabilities must be explicit and block guarantees that depend on them.

## Polished Release Criteria

| # | User criterion | Automated test | Integration test | Manual acceptance | Required evidence | Current status | Blocking defects |
| ---: | --- | --- | --- | --- | --- | --- | --- |
| 1 | Install documented path | npm pack/archive/version/checksum matrix | Clean npm and direct binary install | Follow published quickstart on clean machine | Release manifest, hashes, exit 0, matching versions | Fail | npm latest broken; version skew; no atomic set |
| 2 | Initialize greenfield/brownfield | Idempotent scaffold/state schema | Both fixtures | Inspect files and next action | Baseline fingerprint, typed goal, no false dirty credit | Limited | stale topology docs; setup counted as changes |
| 3 | State imperfect goal | Goal/clarification schema | Ambiguous notes fixture | Questions are product-relevant | Decisions/unknowns with provenance | Partial | generic discuss; answers become signals |
| 4 | Reach clear editable plan | Plan graph validation | Real and deterministic planner | Human can edit/accept rationale and criteria | Accepted revision and source artifacts | Fail | timeout loses plan; generic synthetic fallback |
| 5 | Research unresolved questions | Source/confidence/stop validation | Contradiction fixture | Inspect source and conclusion | Typed question->source->decision link | Limited | plan consumption optional |
| 6 | Build one phase | Worker/process/result assertions | Real adapter writes expected feature | Inspect diff | Run ID, manifest hash, owned diff, claims | Fail | failed/cancelled build can return success |
| 7 | See actual workers | Event-to-process bijection | Live watch during build | Compare UI to process table | PID/session/adapter/start/terminal result | Partial | narrated spawned/completed states |
| 8 | Inspect what changed | Workspace ownership/diff test | Manual pre-existing and worker changes | Review grouped diff | Base/tree hashes and provenance | Partial | pre-existing/generated/worker evidence blur |
| 9 | Verify explicit criteria | Criterion-evidence graph | No-op/irrelevant/stale/docs/test-only cases | Review evidence per criterion | Non-empty required commands/artifacts/reviews | Fail | skipped checks pass; semantic link absent |
| 10 | Recover failed/interrupted phase | Fault injection matrix | Kill real adapter mid-write | Resume without cleanup | Paused journal event; idempotent next action | Fail | false `BUILT`; handoff can lose plan |
| 11 | Resume in new model session | Blind-context sufficiency test | New process/HOME session | User does not restate project | Current traceable capsule and plan/decision state | Partial | projections conflict/omit data |
| 12 | Correct colony with signals | Conflict/expiry/consumption tests | Mid-phase user correction | Inspect why behavior changed | Scoped signal and decision link | Partial | prompt-only precedence; no conflict detection |
| 13 | Replan on evidence change | Plan revision graph tests | Completed unaffected + contradicted future | Inspect preserved work/rationale | Supersession and evidence links | Fail | refresh forbidden after completion |
| 14 | Preserve useful project knowledge | Knowledge provenance/contradiction suite | Two incompatible projects | Inspect/revoke lesson | Evidence, scope, validation, expiry | Fail | memory test fails; Hive contamination |
| 15 | Complete and seal | Seal policy/evidence manifest | End-to-end greenfield and brownfield | Review residual risk/final summary | All criteria evidence, unresolved warnings, versions | Fail | false COMPLETED state can reach seal |
| 16 | Upgrade without corruption | N-1 migration/rollback matrix | Active colony with custom content | Compare before/after and rollback | Versioned migration log, hashes, preserved custom data | Fail/blocked | public install broken; mixed version allowed |

## Security Acceptance

Release blocks unless:

- Read-only roles cannot write through platform tools or shell escape.
- Explicit write scopes reject path traversal, absolute out-of-root targets, and symlink escapes.
- Repository prompt injection cannot grant new tools/permissions or become an unlabelled system instruction.
- External research and skills retain source/trust labels and cannot execute as commands without policy approval.
- Secrets are absent from logs, context projections, claims, events, and release artifacts.
- Downloads fail closed on checksum/signature mismatch and preserve the old install.
- Global knowledge writes are locked and project identity is stable.
- Concurrent state mutations serialize or produce a conflict, never last-write loss.
- `govulncheck` (or documented Go vulnerability equivalent), npm audit policy, secret scan and SBOM/provenance checks pass according to a versioned release policy.

## Evidence Bundle

Every candidate uploads a machine-readable acceptance bundle:

```text
release-manifest.json
environment.json
test-summary.json
journeys/<id>/commands.jsonl
journeys/<id>/events.jsonl
journeys/<id>/state-revisions/
journeys/<id>/workspace-before-after.json
journeys/<id>/worker-results/
journeys/<id>/verification-evidence.json
journeys/<id>/screens-or-terminal-capture/
security/
cross-platform-contract.json
upgrade-migration-report.json
```

Raw model transcripts may be redacted for secrets, but hashes and structured results remain. Redaction itself is logged.

## Gate Policy

- Any P0 journey failure blocks release.
- A skipped required test is a failure, not a pass.
- Flaky reruns do not erase the first failure; both attempts enter the bundle.
- Live-adapter nondeterminism may produce a documented retry policy, but state safety invariants permit no retry exception.
- Snapshot and prompt-substring tests are supporting evidence only.
- Two consecutive release candidates must pass the complete deterministic suite before advanced features are re-enabled.
