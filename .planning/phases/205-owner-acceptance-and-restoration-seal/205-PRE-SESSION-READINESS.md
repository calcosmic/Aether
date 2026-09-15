# Phase 205: Pre-session readiness

**Decision: fit for the agreed local pre-session release against the unchanged
17-failure baseline, at tested revision
`8391fef9cfc7ba122d8834d83009c4006ed5af3e`.** Both complete test runs found exactly
those known failures and no additional failures. This is not an all-green test
result: both test commands exited **1**. Building and static code checks exited
**0**. The owner's walkthrough and acceptance remain pending.

Recorded on **2026-09-16, Europe/Zurich**. Evidence, the live defect register and
disk space were checked at **00:59:11 +02:00** (2026-09-15 22:59:11 UTC).

## Which revision was checked

The parent ran the complete sequence once at the repaired revision in the
isolated, pinned checkout
`/private/tmp/aether-phase205-execution/repaired-checkout`. The final parent
review confirms that checkout remained clean. The commands were serialized;
this executor reused their completed evidence and ran no Go command.

The earlier tested revision, `7508fc73d48370ef1a41ee42c7cd2dae988f558d`, is
superseded. Its normal run executed 5,573 of 5,573 command-package tests but
reported 19 failures: the 17 known failures plus
`TestDocumentedCommandNamesResolve` and `TestResolveAetherRoot_GitFallback`.
Its race run was deliberately interrupted and supplies no completion verdict.

The parent repair, `10e5251d8e86f2ab5c0bd3c21ca93836d4e6dd7b`, initializes the
command library's built-in help before auditing names and makes the root-path
test use its own repository rather than depend on the checkout's name. Neither
production code nor the 17-name baseline was changed. The tested revision differs
from this worktree's original base only in those two test files, the repair note
and parent state documentation. This executor's worktree was not merged or
switched to claim it had been tested at the newer revision.

The repair note was read from
`8391fef9:.planning/phases/205-owner-acceptance-and-restoration-seal/205-GATE-REPAIR.md`.
Its historical statement that full verification was pending is now resolved by
the final evidence below. Original failures, the interruption receipt and the
repair checks remain preserved in the durable evidence directories.

## Completed parent verification

| Check actually run by the parent | Raw exit | Duration | Result |
|---|---:|---:|---|
| `go build ./cmd/aether` | 0 | 6.7 seconds | Passed |
| `go vet ./...` | 0 | 3.4 seconds | Passed |
| `go test ./... -count=1 -timeout 90m` | 1 | 1,373.3 seconds | Complete; exactly the 17 known failures |
| `go test ./... -race -count=1 -timeout 90m` | 1 | 1,467.2 seconds | Complete; exactly the same 17 failures |

Both logs contain the exact controller headline:

```text
FULL-SUITE FAIL discovered=5573 executed=5573 lanes=59
```

**These counts belong to `github.com/calcosmic/Aether/cmd`, not the whole
repository.** In both runs all 59 test groups also report equal planned and
executed counts, summing to 5,573. Both final logs contain zero data-race warnings
and no test-timeout panic; the final review independently records no timeout.

### Other packages, reported separately

All **18 other packages containing tests** passed in both normal and race runs.
Under `github.com/calcosmic/Aether/pkg/`, they are:

`agent`, `agent/curation`, `cache`, `codegraph`, `codex`, `colony`, `downloader`,
`events`, `exchange`, `graph`, `learn`, `llm`, `memory`, `shadow`, `smoke`,
`storage`, `terminal`, and `trace`.

The repository-root package and `cmd/aether` report **no test files**. They are
not counted as tested packages or added to the 5,573 command-package count.

### Every failure and its classification

The approved list comes from the owner's 2026-09-14 known-red record,
`/Users/callumcowie/.claude/projects/-Users-callumcowie-repos-Aether/memory/project_known_red_baseline_2026_09_14.md`.
Its unchanged, durable 17-name copy is `baseline.json` in the evidence root below.
The final logs' top-level failure names were compared as exact sets, independently
of the parent's summary. Every row below failed in **both** runs.

| Failure name | Classification |
|---|---|
| `TestAuditCatalogGolden` | Pre-existing; approved baseline |
| `TestBuildStartLegacyHelpersRetired200` | Pre-existing; approved baseline |
| `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` | Pre-existing; approved baseline |
| `TestCompletionPacketSchemaMatchesStructs` | Pre-existing; approved baseline |
| `TestCurrentVocabulary199` | Pre-existing; approved baseline |
| `TestFailedCheckSendsExactlyOneBuilderFixAttempt` | Pre-existing; approved baseline |
| `TestFixAttemptIsCountedSeparately` | Pre-existing; approved baseline |
| `TestFixAttemptNeverOverwritesTheFirstResult` | Pre-existing; approved baseline |
| `TestGoSourceHintsMatchCobraContracts` | Pre-existing; approved baseline |
| `TestGoldenBuildVisualOutput` | Pre-existing; approved baseline |
| `TestGoldenContinueVisualOutput` | Pre-existing; approved baseline |
| `TestHumanFacingOutputGoesThroughWriteVisualOutput` | Pre-existing; approved baseline |
| `TestNoSecondAutomaticFixAttempt` | Pre-existing; approved baseline |
| `TestPhase199GateReceipt` | Pre-existing; approved baseline |
| `TestPlanningAdversarial200` | Pre-existing; approved baseline |
| `TestPlanningPublicPaths200` | Pre-existing; approved baseline |
| `TestResolveTestCommand_GoProject` | Pre-existing; approved baseline |

**Unexpected top-level failures: 0. Unfixed newly exposed failures: 0.**
`TestGateProbe` appears in the raw name capture because a passing test deliberately
fails a child test to prove that broken verification is detected. Its enclosing
`TestGateProbeCatchesEveryKnownGateNeutering` passed in both logs. That child output
is neither an eighteenth baseline failure nor evidence that a new failure was
waived. The two failures repaired by the parent are absent from the final
top-level failure sets.

## Live defect register and free space

The actual [defect register](../../WINDOWS.md) was reread on 2026-09-16. Its JSON
records, rendered table and header agree: **48 total, 34 open, 14 fixed, 0
waived**. Its bytes also match the file at the tested revision. SHA-256:
`488ba7301c22c8907be07277f0347091e429949ccc8b90958ef4896dd75f0bda`.

Every currently open identifier and its literal status:

| Identifier | Status |
|---|---|
| 2 | open |
| 4 | open |
| 8 | open |
| 9 | open |
| 10 | open |
| 11 | open |
| 12 | open |
| 13 | open |
| 14 | open |
| 15 | open |
| 16 | open |
| 17 | open |
| 18 | open |
| 19 | open |
| 20 | open |
| 21 | open |
| 22 | open |
| 23 | open |
| 25 | open |
| 26 | open |
| 27 | open |
| 28 | open |
| 29 | open |
| 30 | open |
| 31 | open |
| 34 | open |
| 35 | open |
| 36 | open |
| 40 | open |
| 42 | open |
| 43 | open |
| 44 | open |
| 47 | open |
| 48 | open |

Entries described as reopened or deferred still retain their recorded `open`
status. In particular, entry 48 leaves some checking-run usage and cost values
absent. The brief therefore times every task, records only costs the program
reports, and requires missing values to be noted. This preparation does not
close or waive any defect-register entry.

At the measurement time above, the filesystem containing this assigned worktree
had **13,361,221,632 bytes free (12.44 GiB)**, measured with
`shutil.disk_usage('.')`. The parent reports cache cleanup before the first gate.
No cache was cleared and no user files were deleted during this finalization.
Free space is a dated measurement, not a promise about space remaining later.

## Preparation and prerequisite review

The [one-page owner brief](205-BRIEF.md) retains the exact owner goal, ten tasks
in order, named snapshot/session branches, failure-on-confusion rule, ten-minute
limits for tasks 2, 3 and 7, conditional unexercised outcomes, and ten verdict
lines plus the overall yes/no. It contains no command, invocation or flag.

The earlier preparation receipts establish a clean session branch
`p205-owner-session-20260915` in the real M4L project. The complete experiment
is preserved on `p205-owner-work-snapshot-20260915`, including a separate named
snapshot in the nested research repository. All 383 main records, all 1,305
extended project-record files and all 12,605 ignored files were verified
unchanged. Two pre-existing context documents were carried onto the session
branch; three exact local exclusions retained the prior ignored status of local
caches. Full names, hashes, counts and those exclusions are recorded in
[the preparation summary](205-12-SUMMARY.md) and the durable receipts below.
No snapshot, branch switch or record rewrite was repeated during finalization.

The required prerequisite summaries were read:

| Plan | Relevant completed preparation |
|---|---|
| [205-02](205-02-SUMMARY.md) | Reviewers receive the required result format; unsafe learned text is filtered before reuse. |
| [205-03](205-03-SUMMARY.md) | Finish/archive handles missing handoff notes; resume refreshes the note; finalization preserves machine-readable output. |
| [205-04](205-04-SUMMARY.md) | Review-team descriptions match the program; chat instructions relay the finish/archive cards. |
| [205-05](205-05-SUMMARY.md) | A criterion naming a check that never ran is refused. |
| [205-11](205-11-SUMMARY.md) | All 72 capability records are reconciled across seven phase files, with the recorded evidence limits retained. |

These summaries describe prerequisite work. Their scoped checks and requirement
metadata do not replace the complete parent gate or the owner's actual
walkthrough. `PROOF-05` and owner acceptance remain pending.

## Durable evidence and integrity checks

All paths in this table are relative to the durable evidence root:

[/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/](/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/)

| Durable artifact | SHA-256 |
|---|---|
| `repaired-gate/final-review.json` | `9aba08e0135f861126c5902eaf7cd600e0781a7741df8424295bd104ad4beb23` |
| `repaired-gate/results.json` | `57d4599d7b7765a6a53b48740c70726927321b508ba91c5efb2191c8068f41c0` |
| `repaired-gate/full.log` | `1ea79e6ef0a7a075eb41b35c655b3b2adf30245de30ba54086114aac77f55917` |
| `repaired-gate/race.log` | `c31097b4bfdd3505305d8b663965946a6f95d7c7014c1549ac6d84647c278abc` |
| `repaired-gate/build.log` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `repaired-gate/vet.log` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `repaired-gate/archive-manifest.json` | `3dc408d6086e08263b061da64e771a869447bb38a6ba20ec4750b46b6ccee4f4` |
| `baseline.json` | `5ce2d98b291f41586df99f841abe316da169de65b7d2be008023e0c6be071f8d` |
| `gate-repair/original-full.log` | `adcb5ba3b7d3850f61f9566dab015e45de8f86e7c2039c1c3f4a8338dcf980ee` |
| `gate-repair/original-race-interrupted.json` | `29bac83ad29c4ae7437fe0ac7ece85cc7247264da77d2e7d391cbae9fb74df44` |
| `gate-repair/archive-manifest.json` | `acd316e4a5209534e0eb9579b9f6cbd3022e5f61f0cfb07409f03ef25a39a135` |
| `owner-preparation/plan12-inventory-before.json` | `5eac1af751b072860bb2dfa408f8c3679cd9bcd33abc4b8c6b9c602bc1582ee9` |
| `owner-preparation/plan12-verify-session.json` | `52a8516ca42db0c60009e0037aed885098c800d77e9d553e3658d737d0e83500` |
| `owner-preparation/plan12-ignore-preservation.json` | `61ab66c87253da3262083a5104b28fdf78487b8cb2edd7fa8f06ca4ae7cccbdf` |
| `owner-preparation/archive-manifest.json` | `6e50fdfc51b76cdbcb1ab54297a19c4ca84f08016483e92816c34f469cbe4f29` |

The finalizer verified all seven repaired-gate artifacts, all 25 repair artifacts
and all 12 owner-preparation artifacts against their archived byte sizes and
hashes. The archived final review and results match the parent's final temporary
copies exactly. The review's source-results hash matches the archived results;
every stage's log hash matches its archived log.

**Release decision scope:** task 3 is resolved under the plan's explicit
known-failure comparison. Publication remains the next separately assigned
action. No release, push, project closure, archive action or owner acceptance
was performed by this executor.
