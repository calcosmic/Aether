---
phase: 205-owner-acceptance-and-restoration-seal
plan: 12
subsystem: testing
tags: [owner-preparation, preservation, release-readiness, documentation]
requires:
  - phase: 205-02
    provides: Review handoff format and filtering of unsafe learned text
  - phase: 205-03
    provides: Finish/archive and resume handoff repairs
  - phase: 205-04
    provides: Accurate review-team descriptions and chat relay instructions
  - phase: 205-05
    provides: Refusal of criteria naming checks that never ran
  - phase: 205-11
    provides: Reconciled capability evidence and its limits
provides:
  - Named snapshots of owner experiment work and clean session branches
  - One-page owner brief containing ten tasks and no commands
  - Readiness decision supported by the completed parent gate and durable evidence
affects: [205-13, 205-17, owner-walkthrough, PROOF-05]
actuals:
  tokens: 3780
  tasks: 3
  commits: 5
tech-stack:
  added: []
  patterns:
    - Compare full preservation inventories before and after branch preparation
    - Keep raw test failures separate from an agreed baseline readiness decision
key-files:
  created:
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-BRIEF.md
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-PRE-SESSION-READINESS.md
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-12-SUMMARY.md
  modified:
    - /Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.git/info/exclude
key-decisions:
  - Preserve the nested repository under the same named snapshot in its own repository
  - Carry the two pre-existing context records onto the clean session branch unchanged
  - Preserve exactly three existing local cache exclusions without touching cache bytes
  - Use the completed parent gate at 8391fef9 against the unchanged 17-name baseline
  - Leave PROOF-05 pending until the actual owner walkthrough and acceptance
patterns-established:
  - Cite durable log and preservation receipt paths with their SHA-256 hashes
requirements-completed: []
duration: "Two sessions; active duration not measured separately from parent-gate waiting"
completed: 2026-09-16
status: complete
---

# Phase 205 Plan 12: Owner-session preparation summary

**The owner's experiment is saved under named branches, the clean session has a ten-task brief, and the completed parent gate supports readiness against the unchanged 17-failure baseline.** This finishes preparation. The owner has not performed or accepted the walkthrough, so `PROOF-05` remains pending.

## Performance and scope

- **Tasks:** 3 of 3 complete, across preparation and a later gate-finalization session.
- **Recorded preparation inventory:** 2026-09-15 21:35:22 UTC; session preservation verified at 21:39:52 UTC.
- **Final gate/register/disk review:** 2026-09-16 00:59:11 +02:00; readiness committed at 01:06:46 +02:00.
- **Aether artifacts:** three documents, including this summary; no production source changed by this executor.
- **Actuals basis:** 15,118 characters in the final brief and readiness document, divided by four and rounded up, gives 3,780 estimated tokens. This excludes this metadata summary and the owner's pre-existing experiment, which was preserved rather than authored. Five task commits comprise three target-project commits and two Aether commits; the separate summary commit is metadata.

Work stayed in the assigned Aether worktree `/Users/callumcowie/repos/Aether/.claude/worktrees/agent-p205-12-codex-20260915`, branch `worktree-agent-p205-12-codex-20260915`. Its starting base was `7508fc73d48370ef1a41ee42c7cd2dae988f558d`. Finalization did not merge or switch this worktree to the parent's newer tested revision. Shared `STATE.md`, `ROADMAP.md`, and `REQUIREMENTS.md` were not changed by this executor.

## Accomplishments

- Saved every modified, untracked nonignored, and deleted experiment entry, including the nested research work, under named local snapshot branches.
- Prepared clean session branches while preserving all project-record and ignored-file bytes and metadata.
- Wrote the [owner brief](205-BRIEF.md), retaining the exact goal, ten ordered tasks, three ten-minute limits, and the stuck-means-failure rule.
- Completed the [readiness report](205-PRE-SESSION-READINESS.md) from the parent's full normal and race results, live defect register, disk measurement, and checksum-verified durable evidence.

## Task commits

All commits used normal hooks. Nothing was pushed.

| Task | Repository | Commit | Result |
|---|---|---|---|
| 1 | Nested research repository | `f09a3a27e7a81f3e50bb5e79b86d0d97c9a0f61f` | Named snapshot of all 18 research files |
| 1 | Owner's main project | `e8941d6f5e1f3f1834357d4746e7ffcf0f77b962` | Complete experiment snapshot, including the nested snapshot reference |
| 1 | Owner's main project | `363fd07630d9028d2d1a18edf3f5cc21e7dece2d` | Clean session branch carrying the two pre-existing context records |
| 2 | Assigned Aether worktree | `72fd2768c90fc50c75e7cd3d40617c2d347fafb2` | Original one-page brief |
| 3 and brief correction | Assigned Aether worktree | `bee656d1bc9c25d27c983171f7dda13f3f69961b` | Final readiness evidence and reported-cost wording |

**Plan metadata:** this summary is committed separately after task 3 is resolved. The parent-owned repair `10e5251d8e86f2ab5c0bd3c21ca93836d4e6dd7b` is not an executor task commit.

## Task 1: Saved work and clean session

Target project: `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System`. The original main branch `codex/skills-as-commands` remains at its original commit `ccf182644fd57eea8f063466002cd9b062d9be45`.

### Exact pre-change counts

Both status inventories disable rename detection. Grouped status counts an untracked directory once; expanded status lists its individual files.

| Inventory | Modified | Untracked | Deleted | Total |
|---|---:|---:|---:|---:|
| Main project, grouped status | 352 | 231 entries | 181 | 764 |
| Main project, expanded status | 352 | 1,373 files | 181 | 1,906 |
| Nested repository, separately | 4 | 14 files | 0 | 18 |

The main 1,906 entries include one modified gitlink, Git's recorded reference to the nested repository; the other 1,905 entries are ordinary changed files. The snapshot diff is exactly **352 modified, 1,373 added, 181 deleted**. The complete path set, contents and Git modes were compared with the pre-change inventory, not just the totals. Deleted files are represented by their absence in the snapshot and remain available in the unchanged original history.

### Named branches and commits

| Repository | Snapshot branch and commit | Session branch and commit |
|---|---|---|
| Main project | `p205-owner-work-snapshot-20260915` at `e8941d6f5e1f3f1834357d4746e7ffcf0f77b962` | `p205-owner-session-20260915` at `363fd07630d9028d2d1a18edf3f5cc21e7dece2d` |
| Nested research | `p205-owner-work-snapshot-20260915` at `f09a3a27e7a81f3e50bb5e79b86d0d97c9a0f61f` | `p205-owner-session-20260915` at `285a68a9bfa9a3899ac3fd0a03fde516c6e871fc` |

The nested path is `MaxForLive_Vault/MDS-System-Archive/03-Research/pattern-verification-full`. Its original `master` branch remains at `285a68a9bfa9a3899ac3fd0a03fde516c6e871fc`. Its snapshot verifies all 18 files; its clean original session has four files and zero ignored files. The outer snapshot points to the nested snapshot commit. **Recovering the whole experiment requires restoring the named snapshot in both repositories.** Neither snapshot depends on an unnamed stash. The snapshot name was also written to `/tmp/aether-205-snapshot-name` as required.

The main snapshot tree contains **8,317 entries: 8,316 file blobs and one gitlink**. The main session tree contains **7,125 entries: 7,124 file blobs and one gitlink**. Both session working trees were verified clean in the original preparation receipts; no target branch switch or snapshot was repeated during finalization.

The main session commit is a child of the original main HEAD and differs only in `.aether/CONTEXT.md` and `.aether/HANDOFF.md`. Those records already had edits before preparation, so their original working bytes were carried onto the session branch. Their contents, permissions and modification times were not rewritten.

### Project records and ignored files preserved

Before, snapshot and session receipts agree on all **383 `.aether/data/` files** (22 tracked) and all **1,305 extended project-record files**. The extended scope is every `.aether/` file except program source under `skills/` and `ts-host/`. Exact path sets, bytes, permissions and modification times match throughout.

| Preserved content | Unchanged SHA-256 |
|---|---|
| `.aether/data/` inventory | `eaa524f4d970a461be6aee8444bca96e3da611e0c8c1ab39aa3bcd55ac38e868` |
| Extended project-record inventory | `0b4abea16f4b74f3798aae04cfdfd3ae143066118ce62b5a6d0ce25c9801dd88` |
| `.aether/data/COLONY_STATE.json` | `883c58156730f89d329acf97aeb1ce954ba96efb3c6b5996ccf3ec2a83cbaa0d` |
| `.aether/CONTEXT.md` | `033144fcd319a64a1476ada09f3a3cfc57fa2c2908558a3af2bba8d730a05774` |
| `.aether/HANDOFF.md` | `4d32350065a7a5269d573acacb43eb842bd09af3fdb6bff53c5f7709a2519a03` |

The aggregate hashes cover compact, sorted JSON mapping each path to its content SHA-256, size and Git mode. Each file was also compared individually, including its metadata. The preserved colony is **COMPLETED, 3/3 phases and 20/20 tasks**, with no `seal/outcome.json`. Preparation did not seal or archive it: the owner still has that opening task to perform.

All **12,605 ignored files** retain their exact path sets, bytes, permissions and modification times. Returning to the original source tree exposed three files whose ignore rules belonged to the saved experiment. Only these exact local patterns were added to the main project's `.git/info/exclude` to preserve their previously ignored status:

```text
/MaxForLive_Vault/.cache/cycling74-docs.sqlite
/MaxForLive_Vault/.cache/cycling74-web.sqlite
/MaxForLive_Vault/.local/cycling74-web-state.json
```

The exclude file's original SHA-256 is `6671fe83b7a07c8932ee89164d1f2793b2318058eb8b98dc5c06ee0a5a3b0ec1`; after those additions it is `b92bc278b3e11d7a2cbf4d4dfab43a6c917de5a90ed3191247486f4f881d3e64`. The original file is preserved at durable relative path `owner-preparation/plan12-git-info-exclude-before`; the receipt is `owner-preparation/plan12-ignore-preservation.json`. No cache contents were changed. These local exclusions are explicitly part of the preparation record.

## Task 2: Brief checks

The final brief is **463 words, 29 lines**, retaining the compact one-page form. The original 456-word brief was changed only from the promise that all ten tasks record time and cost to: “All ten are timed. Use only costs the program reports; note missing values.” Final SHA-256: `807db7b28acc71dcba88e0edc5bd6790376ceaca6c7fd889fb282af973936c04`.

The prose audit verified:

- The owner's exact goal from the context, ten numbered tasks in the required order, and both named branches.
- No command, invocation or flag; the only file path is the target project.
- A stuck task is marked failed, the owner notes where and moves on without help.
- Tasks **2, 3 and 7** each have a **ten-minute** ask-to-built-and-checked limit; task 7 starts at the resume request. Other tasks are timed without a duration failure.
- Ten verdict lines and an overall yes/no; only the owner's overall yes accepts the work.
- Explicit conditional “not exercised” wording when no check failure or harmful learned change occurs.

Task 3 in the brief uses the existing step-count validator on the clean original project. It does not depend on the experiment-only MIDI-channel option. The brief asks for no reimplementation of the already delivered VAL-41..44 rules.

## Task 3: Completed parent gate

**Fit for the agreed local pre-session release at `8391fef9cfc7ba122d8834d83009c4006ed5af3e`, against the unchanged 17-name baseline.** Both full suites still exited **1**; this is not an all-green result. The owner walkthrough and acceptance are pending.

The parent ran the serialized gate in `/private/tmp/aether-phase205-execution/repaired-checkout`, pinned to that commit, and confirmed it was clean afterward. This executor ran **no Go command**, and did not duplicate the build, static checks or tests.

| Parent command | Raw exit | Completed evidence |
|---|---:|---|
| `go build ./cmd/aether` | 0 | Passed |
| `go vet ./...` | 0 | Passed |
| `go test ./... -count=1 -timeout 90m` | 1 | Complete; 17 known failures only |
| `go test ./... -race -count=1 -timeout 90m` | 1 | Complete; the same 17 failures only |

Each log reports `FULL-SUITE FAIL discovered=5573 executed=5573 lanes=59`. **The 5,573 counts cover the `cmd` package only.** Every test group also reports equal planned/executed counts. Separately, all **18 other packages containing tests** passed both runs; the root package and `cmd/aether` report no test files. The readiness report lists those other packages individually. There were **zero unexpected top-level failures, zero data-race warnings and no timeouts**.

Every following name failed in both normal and race runs and is classified as **pre-existing, approved baseline**:

1. `TestAuditCatalogGolden`
2. `TestBuildStartLegacyHelpersRetired200`
3. `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`
4. `TestCompletionPacketSchemaMatchesStructs`
5. `TestCurrentVocabulary199`
6. `TestFailedCheckSendsExactlyOneBuilderFixAttempt`
7. `TestFixAttemptIsCountedSeparately`
8. `TestFixAttemptNeverOverwritesTheFirstResult`
9. `TestGoSourceHintsMatchCobraContracts`
10. `TestGoldenBuildVisualOutput`
11. `TestGoldenContinueVisualOutput`
12. `TestHumanFacingOutputGoesThroughWriteVisualOutput`
13. `TestNoSecondAutomaticFixAttempt`
14. `TestPhase199GateReceipt`
15. `TestPlanningAdversarial200`
16. `TestPlanningPublicPaths200`
17. `TestResolveTestCommand_GoProject`

This exact set matches the owner's 2026-09-14 record at `/Users/callumcowie/.claude/projects/-Users-callumcowie-repos-Aether/memory/project_known_red_baseline_2026_09_14.md` and the durable `baseline.json`. The apparent additional `TestGateProbe` output is a deliberately failing child test; its enclosing `TestGateProbeCatchesEveryKnownGateNeutering` passes in both runs. It neither enlarges the baseline nor supplies an ignored new top-level failure.

The earlier `7508fc73` gate is **superseded**. Its complete normal run had the 17 baseline failures plus `TestDocumentedCommandNamesResolve` and `TestResolveAetherRoot_GitFallback`; its race run was deliberately interrupted. The parent repaired the built-in help initialization and checkout-name-dependent test fixture in `10e5251d8e86f2ab5c0bd3c21ca93836d4e6dd7b`, integrated at the new tested revision. Both extra failures are absent from the completed final runs. The baseline was never enlarged, and production code was unchanged by those repairs. The newer revision differs from this worktree's base only in the two verifier files, the repair note and parent state documentation.

### Live defect register and disk

The actual `.planning/WINDOWS.md` was reread at **2026-09-16 00:59:11 +02:00**. Its JSON, table and header agree on **48 entries: 34 open, 14 fixed, 0 waived**. All these identifiers have the literal status **`open`**:

**2, 4, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 25, 26, 27, 28, 29, 30, 31, 34, 35, 36, 40, 42, 43, 44, 47, 48.**

The register matches the tested revision byte for byte, SHA-256 `488ba7301c22c8907be07277f0347091e429949ccc8b90958ef4896dd75f0bda`. Reopened/deferred descriptions do not change their recorded `open` status. Entry 48 explains why the brief records only available cost values.

Live free space on the assigned worktree's filesystem was **13,361,221,632 bytes (12.44 GiB)** at the same timestamp, measured with `shutil.disk_usage('.')`. The parent reports cache cleanup before the first gate. This finalizer cleared no caches and deleted no user files. The disk value is a dated measurement.

The required summaries **205-02, 205-03, 205-04, 205-05 and 205-11** were read; their relevant outcomes are recorded in the readiness report. Their scoped checks and requirement metadata do not substitute for the complete gate or complete `PROOF-05`.

## Durable evidence

All relative evidence paths below resolve beneath [/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/](/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/). The parent archived the original evidence, repair evidence and final gate there; the original preservation receipts are retained verbatim.

| Durable artifact | SHA-256 |
|---|---|
| `repaired-gate/final-review.json` | `9aba08e0135f861126c5902eaf7cd600e0781a7741df8424295bd104ad4beb23` |
| `repaired-gate/results.json` | `57d4599d7b7765a6a53b48740c70726927321b508ba91c5efb2191c8068f41c0` |
| `repaired-gate/full.log` | `1ea79e6ef0a7a075eb41b35c655b3b2adf30245de30ba54086114aac77f55917` |
| `repaired-gate/race.log` | `c31097b4bfdd3505305d8b663965946a6f95d7c7014c1549ac6d84647c278abc` |
| `repaired-gate/archive-manifest.json` | `3dc408d6086e08263b061da64e771a869447bb38a6ba20ec4750b46b6ccee4f4` |
| `baseline.json` | `5ce2d98b291f41586df99f841abe316da169de65b7d2be008023e0c6be071f8d` |
| `gate-repair/archive-manifest.json` | `acd316e4a5209534e0eb9579b9f6cbd3022e5f61f0cfb07409f03ef25a39a135` |
| `owner-preparation/plan12-inventory-before.json` | `5eac1af751b072860bb2dfa408f8c3679cd9bcd33abc4b8c6b9c602bc1582ee9` |
| `owner-preparation/plan12-verify-before.json` | `c5051bca059bf4f95b6b68560b9f7e9be22de86d0ecac4ffe9bb31d8d45b5ec5` |
| `owner-preparation/plan12-verify-snapshot.json` | `3c21231c1f0cc7eb5a328471db3fed5bef4c74bffb22b9dae6204858aba5002c` |
| `owner-preparation/plan12-verify-session.json` | `52a8516ca42db0c60009e0037aed885098c800d77e9d553e3658d737d0e83500` |
| `owner-preparation/plan12-ignore-preservation.json` | `61ab66c87253da3262083a5104b28fdf78487b8cb2edd7fa8f06ca4ae7cccbdf` |
| `owner-preparation/archive-manifest.json` | `6e50fdfc51b76cdbcb1ab54297a19c4ca84f08016483e92816c34f469cbe4f29` |

All **7 repaired-gate, 25 gate-repair and 12 owner-preparation artifacts** were checked against their manifests' sizes and hashes. Archived final review and results match the final parent copies; result/log hash references agree. Raw logs were independently checked for complete equal counts, the exact 17-name failure set, passing other packages and absence of race/timeout output.

The archived `owner-preparation/plan12-progress.json` is the historical waiting checkpoint, deliberately retained unchanged. The current scratch progress at `/tmp/aether-phase205-execution/plan12-progress.json` is updated to completed after the summary commit; it does not replace those durable source receipts.

## Decisions and deviations

1. **Nested repository:** the experiment included an independently versioned research repository. A separate named snapshot was necessary to preserve its 18 files. The outer snapshot records its commit; recovery uses both named branches.
2. **Pre-existing context edits:** `.aether/data/` had clean tracked status, but the two context documents already differed from the original commit. A session commit carries those bytes forward while setting aside the source experiment.
3. **Three local ignore entries:** the original tree lacked three experiment ignore rules. Exact local exclusions preserved the prior ignored classification without altering or broadly hiding files; the original exclude file and before/after hashes are recorded above.
4. **Parent-owned verification and repair:** task 3 used the authorized parent's serialized gate. The first complete normal run exposed two verifier defects; the parent repaired them and ran a fresh complete normal/race gate at `8391fef9`. The interrupted old race was never treated as completion.
5. **Cost wording:** the finalizer made the requested narrow correction because some cost values are absent. Timing applies to all tasks; missing reported costs must be noted.

These changes preserve the preparation's scope. No experiment code was discarded, no colony was pre-closed, and no known-failure allowance was expanded.

## Issues and limitations

- The initial parent gate was a dependency checkpoint, now resolved by its final review and complete results.
- The release decision is relative to the recorded 17-failure baseline. Both raw test exits remain 1, and the 34 open defect-register entries remain open.
- Snapshot/session cleanliness and record preservation are established by the original preparation receipts; finalization did not repeat target-project mutations.
- Actual owner behavior, the ten timed journeys, available cost reporting, musical results and the overall acceptance judgment remain unobserved.

## User setup required

No external-service configuration was added. The prepared project and brief are available for the separately scheduled owner session.

## Next phase readiness

The publish plan can read the completed readiness artifact and its pinned gate evidence. Publication itself remains a separate assignment. This executor did not publish, push, seal, archive or claim phase/owner acceptance. **Preparation is complete; `PROOF-05` still requires the actual owner walkthrough and yes/no acceptance.**

---

*Phase: 205-owner-acceptance-and-restoration-seal*
*Plan 12 completed: 2026-09-16*
