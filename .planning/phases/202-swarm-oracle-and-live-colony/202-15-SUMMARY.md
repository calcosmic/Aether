---
phase: 202-swarm-oracle-and-live-colony
plan: "15"
subsystem: docs-and-contract
tags: [wrappers, claude-md, classic-contract, corpus, watch, swarm, oracle]

# Dependency graph
requires:
  - phase: 202-06
    provides: "cmd/watch_dashboard.go's live cockpit (current wave in depth, header line, ticker) this plan's watch wrapper and CLAUDE.md section describe."
  - phase: 202-07
    provides: "cmd/swarm_repair_checkpoint.go's checkpoint/verify/rollback transaction this plan's swarm wrapper and CLAUDE.md section describe."
  - phase: 202-09
    provides: "cmd/watch_replay.go's replay-backed idle summary and the three-way live/replay/idle branch this plan's watch wrapper describes."
  - phase: 202-11
    provides: "cmd/oracle_live.go's live round emission this plan's oracle wrapper and CLAUDE.md section describe."
  - phase: 202-12
    provides: "cmd/oracle_synthesis.go's recommendation-first synthesis ordering this plan's oracle wrapper and CLAUDE.md section describe."
  - phase: 202-14
    provides: "cmd/episode_index.go's status/history lineage the corpus cases and CLAUDE.md section reference for durable discoverability."
provides:
  - "Nine updated wrapper files (three .aether/commands/*.yaml sources plus six .claude/.opencode markdown copies) describing the restored watch three-screen model, Swarm's four-lens diagnosis with checkpointed repair and three-strike case, and Oracle's one-clarification/shared-preset/recommendation-first shape -- Claude and OpenCode copies stay byte-identical, and every existing ceremony contract string is unchanged and in order."
  - "18 new Phase 202 corpus cases in cmd/testdata/classic-contract/v1/cases.json (2-4 per group across the six V-202-* groups, at least one negative per group) plus cmd/classic_contract_202_test.go's TestClassicContractPhase202Cases proving schema validity, full decision/group coverage, go_test_symbol resolution against the parsed cmd package, and both required negative fixtures."
  - "A new 'Live Colony, Swarm, and Oracle (v1.28)' section in CLAUDE.md naming a Go test beside every behavioral claim, plus cmd/claudemd_live_colony_test.go's TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest guard (with its own negative-fixture proof) that fails by symbol and by citing sentence when a cited test stops existing."
affects: []

# Actuals (#2632)
actuals:
  tokens: 20505
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Structural CLAUDE.md claim-guard, generalized: TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest follows TestEveryLearningClaimInCLAUDEMDNamesALiveTest's precedent (scan sections for the Test-name shape, verify each against a live-parsed cmd/pkg symbol set) and extends it to also report the citing sentence, via a coarse sentence splitter -- reused rather than a third guard shape."
    - "Corpus case shape reused verbatim from Phase 201's V-201-* cases: positive cases assert `.aether/data/COLONY_STATE.json` changed; negative (rejected-shortcut) cases assert a named, never-written forbidden_artifacts receipt path -- no new case shape introduced for Phase 202."
    - "Package-wide Go test symbol resolution (classicContractCmdTestSymbols) generalizes the existing single-cited-file resolver (classicPhase200GoTestSymbolResolves) to parse every cmd/*_test.go file once, so a case's go_test_symbol need not also appear in its own source_citations to be found -- a stricter, not weaker, proof."

key-files:
  created:
    - cmd/classic_contract_202_test.go
    - cmd/claudemd_live_colony_test.go
  modified:
    - .aether/commands/watch.yaml
    - .aether/commands/swarm.yaml
    - .aether/commands/oracle.yaml
    - .claude/commands/ant/watch.md
    - .claude/commands/ant/swarm.md
    - .claude/commands/ant/oracle.md
    - .opencode/commands/ant/watch.md
    - .opencode/commands/ant/swarm.md
    - .opencode/commands/ant/oracle.md
    - .claude/commands/ant-watch.md
    - .claude/commands/ant-swarm.md
    - .claude/commands/ant-oracle.md
    - cmd/testdata/classic-contract/v1/cases.json
    - cmd/classic_contract_test.go
    - cmd/lifecycle_wrapper_contract_test.go
    - CLAUDE.md

key-decisions:
  - "Extended validateClassicContractCorpus's front-door-journey exemption (previously `SYN-200-` and `SYN-201-` only) to also skip `SYN-202-` cases. Without this, no Phase 202 case could ever validate -- the exemption existed for exactly this reason on the two prior phases and Phase 202 needed the identical extension, not a new mechanism."
  - "The 18 new corpus cases split 3-positive/1-negative or 2-positive/1-negative per group (never a flat 2-per-group) so every one of the twelve SYN-202 decisions individually gets its own case, rather than doubling up two decisions onto one case to hit the two-per-group floor with fewer total cases."
  - "The Go-test-symbol resolver for Phase 202 cases (classicContractCmdTestSymbols) parses the whole cmd package once rather than only each case's own cited _test.go file (the Phase 200 precedent's narrower approach) -- a case's go_test_symbol must exist in the package regardless of which file its source_citations happen to name, which is the stricter and more accurate proof of 'this test exists.'"
  - "Task 3's full-suite sweep surfaced two regressions from Task 1's wrapper edits that Task 1's own scoped verify list could not have caught: a stale CMD-04 hash ledger entry (TestSpecialistCommandSurfacesUnchanged) and stale flat .claude/commands/ant-{watch,swarm,oracle}.md mirrors (TestLifecycleFlatMirrorsMatchCanonical). Both are fixed in this plan's Task 3 commit rather than deferred, since they are directly caused by this plan's own Task 1 edits, not pre-existing drift."

patterns-established:
  - "A plan whose own commit touches three hand-maintained markdown copies of one command should always run the full package test suite once at the end, not only the plan's own listed verify commands -- CMD-04-style hash ledgers and flat-mirror-parity checks live in test files the touched wrapper's own commit never lists."

requirements-completed: [LIVE-02, LIVE-03, LIVE-06, CEC-05]

coverage:
  - id: D1
    description: "The watch, Swarm and Oracle wrappers on both platforms describe the runtime behaviour this phase actually restored (three watch screens, four Swarm lenses with checkpointed repair and three-strike escalation, Oracle's one clarification and recommendation-first answer), Claude and OpenCode stay byte-identical, and every existing ceremony-contract string survives unchanged and in order."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/command_parity_test.go#TestClaudeOpenCodeCommandParity"
        status: pass
      - kind: unit
        ref: "cmd/swarm_wrapper_ceremony_test.go#TestSwarmWrapperCeremonyContract"
        status: pass
      - kind: unit
        ref: "cmd/command_source_hygiene_test.go#TestCommandSourceHygiene"
        status: pass
      - kind: unit
        ref: "cmd/agent_doc_paths_test.go#TestOracleYamlWrapperRoleMatchesRuntimeOwnership"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestSpecialistCommandSurfacesUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleFlatMirrorsMatchCanonical"
        status: pass
    human_judgment: false
  - id: D2
    description: "Phase 202's twelve synthesis decisions are registered as 18 executable corpus cases across the six Phase 202 groups (each with at least one negative case proving a rejected shortcut never returned), every case's go_test_symbol resolves in the parsed cmd package, and a fixture naming a nonexistent symbol or an unregistered decision fails by name."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/classic_contract_202_test.go#TestClassicContractPhase202Cases"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractSchema"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicMechanismCoverage"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase202MechanismRegistry"
        status: pass
      - kind: other
        ref: "python3 group/negative-case/126-unchanged count check (embedded in the plan's own <verify> block)"
        status: pass
    human_judgment: false
  - id: D3
    description: "CLAUDE.md gains a section describing the live colony view, four-lens Swarm and iterative Oracle in plain English with colony terms translated inline, every behavioral claim names a Go test that proves it, and a guard fails by name when a cited test stops existing -- proven able to fail by its own negative fixture. The module builds, vets clean, and every test touching a file this plan changed passes, including under -race."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/claudemd_live_colony_test.go#TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_live_colony_test.go#TestClaudeMDLiveColonyGuardCatchesAMissingSymbol"
        status: pass
      - kind: unit
        ref: "cmd/phase_end_hive_test.go#TestEveryLearningClaimInCLAUDEMDNamesALiveTest"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_verification_depth_test.go#TestCLAUDEMDVerificationDepthClaims"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDStatesCoherentJobContract"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false
---

# Phase 202 Plan 15: Wrappers, Corpus, and Guide Say What the Runtime Now Does Summary

**Updated `/ant-watch`, `/ant-swarm`, and `/ant-oracle` on both platforms, registered 18 executable Phase 202 corpus cases across all twelve SYN-202 decisions, and added a CLAUDE.md section where every claim names its own proving test.**

## Performance

- **Duration:** ~75 min
- **Started:** 2026-09-11T14:16Z (session start; this plan's own work began after reading context)
- **Completed:** 2026-09-11T15:31Z
- **Tasks:** 3
- **Files modified:** 18 (2 created, 16 modified)

## Accomplishments

- The watch, Swarm and Oracle wrapper triplets (`.aether/commands/*.yaml` plus the Claude/OpenCode markdown pairs) now describe the runtime's restored behaviour -- watch's three screens (live, replay, idle) with the runtime alone resolving which; Swarm's four investigators, comparison card, checkpointed repair and three-strike case; Oracle's one clarification, the shared Fast/Balanced/Deep/Exhaustive picker with each preset's own target/cap, watchable rounds, and recommendation-first answers.
- Phase 202's twelve synthesis decisions are executable contract: 18 new corpus cases (144 total, the pre-existing 126 unchanged) across all six `V-202-*` groups, each group carrying at least one negative case proving a rejected shortcut -- a second event transport, an invented liveness row, four cosmetically identical lenses, an uncheckpointed repair, a prefix-based deletion, a sources-first synthesis -- never returned.
- CLAUDE.md's new "Live Colony, Swarm, and Oracle (v1.28)" section names a real Go test beside every behavioural claim it makes; `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest` fails by symbol and by citing sentence if any of those 18 cited tests stops existing, and its own negative fixture proves the guard can fail.
- Along the way, fixed two real regressions Task 1's edits introduced that Task 1's own scoped verify list never covered: a stale CMD-04 command-hash ledger entry, and three stale flat `.claude/commands/ant-*.md` mirrors.

## Task Commits

Each task was committed atomically:

1. **Task 1: Update the watch, Swarm and Oracle wrapper triplets on both platforms** - `300b914a` (feat)
2. **Task 2: Register Phase 202's behaviours as executable corpus cases** - `74d1571d` (test)
3. **Task 3: Make every shipped claim name its test, then sweep the phase** - `d25b4475` (docs; includes the two regression fixes discovered during the sweep)

**Plan metadata:** (this commit)

_Note: Task 3's commit folds in the two regression fixes it discovered during its own full-suite sweep, rather than a separate follow-up commit, since both are directly caused by Task 1's edits in this same plan._

## Files Created/Modified

- `.aether/commands/watch.yaml` - three-screen behaviour description, `--once`/`--interval` documented
- `.aether/commands/swarm.yaml` - four-lens investigation, comparison card, checkpointed repair, three-strike case
- `.aether/commands/oracle.yaml` - one-clarification, shared depth picker, watchable rounds, recommendation-first answer
- `.claude/commands/ant/{watch,swarm,oracle}.md`, `.opencode/commands/ant/{watch,swarm,oracle}.md` - matching markdown prose, Claude/OpenCode byte-identical
- `.claude/commands/ant-{watch,swarm,oracle}.md` - flat installed-consumer mirrors resynced to the canonical nested sources
- `cmd/testdata/classic-contract/v1/cases.json` - 18 new Phase 202 cases appended; 126 pre-existing cases untouched
- `cmd/classic_contract_test.go` - `validateClassicContractCorpus`'s Phase 200/201 exemption extended to `SYN-202-`
- `cmd/classic_contract_202_test.go` - new: `TestClassicContractPhase202Cases` and its symbol-resolution/coverage/schema fixtures
- `cmd/lifecycle_wrapper_contract_test.go` - CMD-04 hash ledger entries for `oracle.md`/`swarm.md` updated with a stated reason
- `CLAUDE.md` - new "Live Colony, Swarm, and Oracle (v1.28)" section
- `cmd/claudemd_live_colony_test.go` - new: the section's own removal-proof guard plus its negative fixture

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Extended the Classic corpus's front-door-journey exemption to SYN-202- cases**
- **Found during:** Task 2, first run of `TestClassicContractPhase202Cases`
- **Issue:** `validateClassicContractCorpus` only skipped its "case ID must end with .claude/.opencode" front-door-journey check for `SYN-200-` and `SYN-201-` prefixed cases. Every new `SYN-202-` case failed this check, since no Phase 202 case is part of a required front-door journey.
- **Fix:** Added `|| strings.HasPrefix(testCase.SynthesisDecision, "SYN-202-")` to the existing skip condition -- the exact extension the prior two phases already established, applied a third time.
- **Files modified:** `cmd/classic_contract_test.go`
- **Verification:** `TestClassicContractPhase202Cases` and the full `TestClassicContract*` suite (104s, all green) pass; `TestClassicContractCorpusRequiredCategories`/`RejectsMissingFrontDoorCase` (front-door journey requirements) still pass unchanged.
- **Committed in:** `74d1571d` (Task 2 commit)

**2. [Rule 1 - Bug] Updated the stale CMD-04 hash ledger for oracle.md/swarm.md**
- **Found during:** Task 3's full-suite sweep (`TestSpecialistCommandSurfacesUnchanged` failing on `.claude/commands/ant/oracle.md`, `.claude/commands/ant/swarm.md`, and their OpenCode copies)
- **Issue:** Task 1 edited the content of `oracle.md` and `swarm.md` on both platforms without updating the SHA-256 hashes `specialistCommandSurfaceHashes` pins them to -- a "deliberately brittle" ledger whose own doc comment says a legitimate edit must update the recorded hash in the same commit with a stated reason.
- **Fix:** Recomputed both files' SHA-256 and updated all four ledger entries (Claude + OpenCode, oracle + swarm), with a comment naming this plan and what changed.
- **Files modified:** `cmd/lifecycle_wrapper_contract_test.go`
- **Verification:** `TestSpecialistCommandSurfacesUnchanged` passes for all 17 tracked surfaces.
- **Committed in:** `d25b4475` (Task 3 commit)

**3. [Rule 1 - Bug] Resynced stale flat .claude/commands/ant-{watch,swarm,oracle}.md mirrors**
- **Found during:** Task 3's full-suite sweep (`TestLifecycleFlatMirrorsMatchCanonical` failing for `watch`/`swarm`/`oracle`)
- **Issue:** `.claude/commands/ant-<verb>.md` is a byte-identical flat mirror of the canonical nested `.claude/commands/ant/<verb>.md` source, copied (never hand-edited) by `aether install`/`aether update`. Task 1 edited the three canonical nested sources but never touched the flat mirrors, leaving them stale.
- **Fix:** Copied the updated canonical content into the three flat mirrors.
- **Files modified:** `.claude/commands/ant-watch.md`, `.claude/commands/ant-swarm.md`, `.claude/commands/ant-oracle.md`
- **Verification:** `TestLifecycleFlatMirrorsMatchCanonical` passes for all mirrors, including `watch`/`swarm`/`oracle`.
- **Committed in:** `d25b4475` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking corpus-validation gap, 2 bugs caused by this plan's own Task 1 edits and caught only by the full-suite sweep Task 3 required).
**Impact on plan:** All three were necessary for correctness and for the plan's own success criteria (a fully green module). No scope creep -- each fix is directly downstream of this plan's own tasks, not unrelated work.

## Issues Encountered

A full unscoped `go test ./... -count=1` run hit two conditions unrelated to this plan's changes:

1. **A documented machine-specific suite ceiling.** This repo's `cmd` package spawns isolated child processes per test lane; on this machine those lanes intermittently exceed their per-lane deadline under full parallel load (`isolated process child hub setup deadline: context deadline exceeded`). This matches a pre-existing, previously-documented condition on this development machine and is not caused by any file this plan touches.
2. **~15 pre-existing test failures in files this plan never touches**, confirmed by tracing each failure to its source file: `TestNextActionNeverHardcoded` (`cmd/swarm_cmd.go:runSwarmDestroy`), `TestPhase199GateReceipt` (a Phase 199 protected-path fingerprint unrelated to any Phase 202 file), `TestQueenChoiceReachesTheDispatchList`/`TestNoWorkerWithoutStatedReason` (Queen orchestration), `TestCurrentVocabulary199` (a Phase 199 vocabulary inventory), `TestHumanFacingOutputGoesThroughWriteVisualOutput` (`cmd/watch_live.go:runColonyLiveRefreshLoop` -- a pre-existing direct-stdout-write finding, not part of this plan's `<files>` list), `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`, `TestAuditCatalogGolden`, `TestBuildStartLegacyHelpersRetired200`, `TestGoSourceHintsMatchCobraContracts`, `TestCompletionPacketSchemaMatchesStructs`, `TestPlanningAdversarial200`, `TestSkillManifestReadEmpty`, `TestSkillManifestReadFromHub`, `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`, `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestNoSecondAutomaticFixAttempt`.

Per the scope boundary rule, these are out of scope for this plan -- none trace to a file in this plan's `<files>` list or to any file this plan's tasks edited. They are logged to `.planning/WINDOWS.md` (kind `unrun-verify`) rather than fixed here.

What **was** verified clean: `go build ./...`, `go vet ./...`, every test in the plan's own `<verify>` blocks, both regression fixes above, and (as a scoped proxy for the full-module race sweep the plan's own `<verify>` block could not complete within this machine's ceiling) `go test ./pkg/... -race -count=1` (all 17 `pkg/` packages pass) plus `go test ./cmd -race -count=1` scoped to every test touching a file this plan changed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 202 (Swarm, Oracle and Live Colony) is now fully planned and executed: 15/15 plans have summaries.
- The 15 pre-existing unrelated test failures logged above are pending investigation outside this plan's scope -- ready for `/gsd-verify-work` or a dedicated Swarm/Oracle investigation to pick up.
- Ready for phase-level verification (`/gsd-verify-work 202`) and, once cleared, `/gsd-plan-phase 203`.

## Self-Check: PASSED

All 17 created/modified key files verified present on disk (`[ -f ]`); all 3 task commit hashes (`300b914a`, `74d1571d`, `d25b4475`) verified present in `git log --oneline --all`.

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
