---
phase: 195-coherent-jobs
plan: 09
subsystem: orchestration
tags: [parity, wrappers, codex-guide, command-yaml, coherent-jobs, task-receipts, team-checkin]

# Dependency graph
requires:
  - phase: 195-coherent-jobs
    provides: "195-03's coherent-job planner (job_name/job_reason/job_source, covered_task_ids, job_decisions), 195-04/195-06's two-stage task-receipt trust boundary, 195-05's decideBuildCheckin/--checkin/checkin_summary, 195-07's parent-linked recovery job, and 195-08's worktree admission/sync/finalization"
provides:
  - "One shared, executable definition of the build coherent-job contract (buildCoherentJobFieldAnchors / buildCoherentJobAuthorityAnchors / buildCoherentJobForbiddenAnchors) used by three separate guards"
  - "TestCommandGuideBuildCoherentJobsContract: the Codex build guide carries the contract on codex, claude, and opencode"
  - "TestBuildCommandYAMLCoherentJobsParity: .aether/commands/build.yaml and the Codex build-cycle skill carry it, and the drift guard still names every surface"
  - "TestLifecycleWrappersCarryCoherentJobContract: all three build wrapper copies carry it and stay byte-identical"
  - "Retirement of the pre-195 wrapper claim that a false checkin_requested means --no-checkin was passed"
affects: [195-10]

# Actuals (#2632)
actuals:
  tokens: 13000
  tasks: 1
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One shared anchor definition per cross-surface contract: three guards import the same vars instead of each re-typing its own list, so the guards cannot drift from each other"
    - "Verbatim-sentence authority anchors compared byte-for-byte across five surfaces, because a paraphrase on one platform is exactly how this repo previously shipped four surfaces each describing a slightly different contract"

key-files:
  created: []
  modified:
    - .aether/commands/build.yaml
    - .aether/skills/colony/aether-colony-build-cycle/SKILL.md
    - .claude/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .opencode/commands/ant/build.md
    - cmd/command_guide.go
    - cmd/command_guide_test.go
    - cmd/parity_test.go
    - cmd/lifecycle_wrapper_contract_test.go

key-decisions:
  - "The contract is defined ONCE (cmd/command_guide_test.go) and asserted by three guards over five surfaces. Each guard re-typing its own anchor list would reproduce the exact drift the guards exist to catch."
  - "Three authority statements are compared verbatim rather than by keyword, because the failure mode this plan closes is five surfaces that each say something slightly different about who owns completion credit."
  - "The forbidden-anchor list is the load-bearing half: '(`--no-checkin` was passed)' was live wrapper text and is now refused by name, so the one-worker fast path cannot be silently read as 'skip the stage'."
  - "cmd/cli_flag_audit_test.go was listed in files_modified but needed no edit -- it is a scanner, and the new --job-proposal / --checkin wrapper examples were written in a form it actually parses so it validates them against the real Go flag set."
  - "Landed as ONE commit, per the plan's must_have that no intermediate commit may expose an updated canonical contract while any shipped surface is stale. RED was captured and quoted before any surface changed rather than committed separately."

patterns-established:
  - "Cross-surface parity guard: shared anchor vars + a single assert helper + one guard per surface family (guide / .aether canonical sources / shipped wrappers)."

requirements-completed: [JOBS-01, JOBS-02, JOBS-03, JOBS-04]

coverage:
  - id: D1
    description: "The Codex build guide names every coherent-job, receipt, recovery and check-in contract item and the three verbatim Go-authority statements, on codex, claude, and opencode."
    requirement: "JOBS-01"
    verification:
      - kind: unit
        ref: "cmd/command_guide_test.go#TestCommandGuideBuildCoherentJobsContract"
        status: pass
    human_judgment: false
  - id: D2
    description: ".aether/commands/build.yaml and the aether-colony-build-cycle Codex skill carry the same contract, and the YAML drift guard still names the Claude/OpenCode wrappers, the Codex skill, and cmd/command_guide.go."
    requirement: "JOBS-01"
    verification:
      - kind: unit
        ref: "cmd/parity_test.go#TestBuildCommandYAMLCoherentJobsParity"
        status: pass
    human_judgment: false
  - id: D3
    description: "All three build wrapper copies carry the contract and remain byte-identical; the byte-identity assertion was proven load-bearing by mutation."
    requirement: "JOBS-04"
    verification:
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleWrappersCarryCoherentJobContract"
        status: pass
      - kind: integration
        ref: "cmp -s .claude/commands/ant/build.md .claude/commands/ant-build.md && cmp -s .claude/commands/ant/build.md .opencode/commands/ant/build.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "No surface tells an agent that a false checkin_requested only means --no-checkin was passed; each consumes checkin_reason and renders the runtime's compact summary on the one-worker fast path."
    requirement: "JOBS-03"
    verification:
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleWrappersCarryCoherentJobContract (forbidden anchor \"(`--no-checkin` was passed)\")"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every --job-proposal and --checkin flag reference written into the wrappers resolves against the real Go flag set, and no CLI surface changed."
    requirement: "JOBS-01"
    verification:
      - kind: integration
        ref: "cmd/cli_flag_audit_test.go#TestCLIFlagAudit"
        status: pass
      - kind: integration
        ref: "cmd/parity_test.go#TestPlatformParityGolden"
        status: pass
      - kind: integration
        ref: "go run ./cmd/aether contract-schema --check"
        status: pass
    human_judgment: false
  - id: D6
    description: "The prose each platform will actually read is accurate to the shipped runtime and understandable without opening the repository."
    verification: []
    human_judgment: true
    rationale: "Wording quality and plain-English readability are owner judgement; the tests prove the facts are present and correct, not that the sentences read well."

# Metrics
duration: 35min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 09: Cross-Surface Coherent-Job Parity Summary

**Every shipped build surface — the YAML command source, the Codex guide, the Codex build-cycle skill, and all three byte-identical wrappers — now describes the same runtime-owned coherent-job, task-receipt, recovery, worktree and check-in contract, enforced by three guards reading one shared definition.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-27
- **Tasks:** 1
- **Files modified:** 9

## Accomplishments

- Defined the build coherent-job contract **once** — 15 real runtime flag/field names, 3 verbatim Go-authority sentences, and 5 forbidden instructions (`cmd/command_guide_test.go`) — and asserted it from three separate guards, so the guards themselves cannot drift apart.
- Updated `.aether/commands/build.yaml` with three new `wrapper_additions` blocks (`coherent_jobs`, `task_receipts`, `team_checkin`) and an `ownership.coherent_jobs` line naming the Go-authority boundary.
- Updated the Codex build guide (`cmd/command_guide.go`) with three coherent-job/check-in PreSteps, two task-receipt PreSteps, and two recovery/worktree PostSteps.
- Updated the Codex `aether-colony-build-cycle` skill's Build Flow with steps 5a, 5b, 13a, 16 and 17 covering grouping, check-in, receipts, recovery, and worktree reconciliation.
- Rewrote the wrapper's **Team Check-In** entry condition: `checkin_requested`/`checkin_reason` are now the runtime's decision with all five reason codes spelled out, `checkin_summary` is rendered on the one-worker fast path, and the stale "(`--no-checkin` was passed)" claim is gone and refused by name.
- Added a new **Coherent Jobs** wrapper stage with the full stage skeleton, plus receipt requirements in Worker Spawning and recovery/worktree reading in Finalize — then copied the canonical wrapper byte-for-byte into the two mirror copies.

## Task Commits

The plan's `must_haves` state that "no intermediate commit can expose an updated canonical contract while leaving any shipped wrapper or Codex surface stale", so the single task landed as a single commit covering the tests and all five surfaces together.

1. **Task 1: Synchronize YAML, guide, skill, and wrapper triplet atomically** — `a7b00742` (docs)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/command_guide_test.go` — shared anchor vars, `assertBuildCoherentJobContract`, `TestCommandGuideBuildCoherentJobsContract`
- `cmd/parity_test.go` — `TestBuildCommandYAMLCoherentJobsParity` (YAML + Codex skill + drift-guard surface list)
- `cmd/lifecycle_wrapper_contract_test.go` — `buildWrapperTripletPaths`, `TestLifecycleWrappersCarryCoherentJobContract` (contract + byte-identity)
- `cmd/command_guide.go` — build entry: 5 new PreSteps, 2 new PostSteps
- `.aether/commands/build.yaml` — `ownership.coherent_jobs` plus `coherent_jobs` / `task_receipts` / `team_checkin` wrapper additions
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — Build Flow steps 5a, 5b, 13a, 16, 17
- `.claude/commands/ant/build.md` — new `## Coherent Jobs` stage; rewritten Team Check-In entry condition; receipt bullets in Worker Spawning; recovery/worktree reading in Finalize
- `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md` — byte-identical copies of the canonical wrapper

## Verification Commands and Results

Every command below was run; the real output is reported.

- Plan `<verify>` (exact): `go test ./cmd -run 'Test(CommandGuideBuildCoherentJobsContract|BuildCommandYAMLCoherentJobsParity|LifecycleWrapper.*CoherentJob|CLIFlagAudit|PlatformParity)' -count=1 -timeout 10m` → `ok github.com/calcosmic/Aether/cmd 0.639s`
- `cmp -s .claude/commands/ant/build.md .claude/commands/ant-build.md && cmp -s .claude/commands/ant/build.md .opencode/commands/ant/build.md` → both succeeded (`cmp both OK`)
- `go run ./cmd/aether contract-schema --check` → `{"ok":true,"result":{"drift":false,"path":".aether/schemas/completion-packet.schema.json"}}` — the generated schema was not modified by this plan
- Broader regression sweep over every guard that reads these files: `go test ./cmd -run 'Test(BuildWrapper|LifecycleWrapper|Lifecycle|CommandGuide|Codex.*Skill|CodexLifecycle|BriefPath|YAMLWrapperContract|AllYamlHaveWrappersAndGuide|NoPhantomCommands|FlagParityAcrossSurfaces|ExecutionPathAudit|IntelligentWrappers|WrapperSources|AuditCatalogGolden|BuildMdOwnership|SpecialistCommandSurfaces)' -count=1 -timeout 10m` → `ok github.com/calcosmic/Aether/cmd 0.767s`
- `go build ./...` → clean. `go vet ./cmd/` → clean. `gofmt -l cmd/ pkg/` → empty.
- `TestAuditCatalogGolden` passes with no golden update: this plan added no CLI flag and no subcommand, it only documented flags 195-05 and 195-03 already registered.

## RED Evidence

The task is `type="auto"` (not `tdd="true"`), and the plan's atomicity `must_have` forbids an intermediate commit, so the tests were **written and run before any surface changed** but committed together with the fix. The real pre-change output:

```
--- FAIL: TestCommandGuideBuildCoherentJobsContract/codex
    command-guide build --platform codex: missing coherent-job contract item "--job-proposal"
    command-guide build --platform codex: missing coherent-job contract item "job_decisions"
    ... (all 15 field anchors) ...
    command-guide build --platform codex: missing verbatim Go-authority statement:
      Go owns accepted groups, completion credit, retry, worktree reconciliation, and check-in policy; the wrapper proposes, renders, spawns, and submits.
--- FAIL: TestBuildCommandYAMLCoherentJobsParity
    .aether/commands/build.yaml: missing coherent-job contract item "--job-proposal"
    .aether/skills/colony/aether-colony-build-cycle/SKILL.md: missing coherent-job contract item "task_receipts"
--- FAIL: TestLifecycleWrappersCarryCoherentJobContract/.claude/commands/ant/build.md
    .claude/commands/ant/build.md: still carries a forbidden instruction "(`--no-checkin` was passed)"
--- FAIL: TestLifecycleWrappersCarryCoherentJobContract/.claude/commands/ant-build.md
    .claude/commands/ant-build.md: still carries a forbidden instruction "(`--no-checkin` was passed)"
--- FAIL: TestLifecycleWrappersCarryCoherentJobContract/.opencode/commands/ant/build.md
    .opencode/commands/ant/build.md: still carries a forbidden instruction "(`--no-checkin` was passed)"
FAIL	github.com/calcosmic/Aether/cmd	0.693s
```

All five surfaces failed by name before the change and pass after it. The forbidden-instruction failure is the load-bearing one: that sentence was live shipped wrapper text, made untrue by 195-05's one-worker fast path.

The one assertion that was **new rather than newly-satisfied** — the triplet byte-identity subtest — was proven load-bearing by mutation (append one comment line to the OpenCode copy, run, restore from a captured copy; `grep -c MUTATION` = 0 and `cmp` clean afterwards):

```
--- FAIL: TestLifecycleWrappersCarryCoherentJobContract/triplet_is_byte_identical
    build wrapper copies drifted: .claude/commands/ant/build.md (30932 bytes) != .opencode/commands/ant/build.md (30951 bytes)
    — the three copies are hand-maintained and must be byte-identical
```

## Decisions Made

See `key-decisions` in the frontmatter. The load-bearing one: the contract is written down once and read by three guards. A guard that re-types its own copy of the list is a fourth surface that can go stale, which is precisely the failure this plan exists to prevent.

## Deviations from Plan

### Auto-fixed Issues

None — no bug, missing-critical, or blocking issue was encountered.

### Scoping decisions (documented, not Rule 1-4 fixes)

**1. `cmd/cli_flag_audit_test.go` was listed in `files_modified` but not modified.**
- **Found during:** Task 1.
- **Reason:** `TestCLIFlagAudit` is a scanner, not a fixture list — it extracts `aether <subcommand> --flag` calls from the wrapper corpus and checks each flag against the live Cobra registration. It needed no edit; it needed the wrapper's new flag examples written in a form it actually parses. Its regex only captures flags that immediately follow the subcommand, so both new examples were written as `aether build --job-proposal '<json>' $ARGUMENTS --plan-only` and `aether build --checkin $ARGUMENTS --plan-only` rather than the usual `aether build $ARGUMENTS --plan-only …` shape. Both flags are now genuinely audited against the Go runtime, which a cosmetic edit to the test file would not have achieved.
- **Verification:** `TestCLIFlagAudit` passes and reports the two flags resolving; deleting either flag from `cmd/codex_workflow_cmds.go` would now fail it by name.

**2. Verbatim authority sentences forced un-wrapped lines in the Codex skill.**
- **Found during:** Task 1.
- **Reason:** The three authority statements are asserted byte-for-byte, and markdown line-wrapping broke two of them inside `SKILL.md`, producing a real failure. They are now single long lines in that file. This is deliberate: the alternative — a keyword-based assertion — would accept five surfaces that each paraphrase the ownership rule differently, which is the exact drift the plan targets.

**3. Wrapper copies were synchronized with `cp`, not hand-edited three times.**
- **Reason:** The three copies have no generator and are byte-identical by policy. Editing the canonical file and copying it is the only method that cannot produce a near-miss; the new `triplet_is_byte_identical` subtest now fails if anyone edits a mirror directly.

---

**Total deviations:** 0 auto-fixed. 3 documented scoping decisions, none affecting scope or correctness.
**Impact on plan:** None. Every `<acceptance_criteria>` command and the plan-level `<verification>` commands were run and passed.

## Issues Encountered

- The plan's `wrapper additions` had to avoid the existing method-to-envelope proportion guard (`TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob`, minimum 3:1) and the singular wrapper-host-contract pointer rule. Resolved by writing the new stage as method prose (`Read job_decisions …`) rather than envelope-parsing prose (`Parse result.…`) and adding no second contract pointer; the wrapper's ratio moved from 37:2 to 40:2, comfortably clear.
- Nothing else. No CLI surface, no generated schema, and no runtime behaviour changed in this plan — it is a documentation-parity commit guarded by executable tests.

## Known Stubs

None — every anchor asserted by the three guards is present on all five surfaces, and each guard was seen to fail before the change (or, for the one newly-added assertion, by mutation).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 195-10 owns the remaining documentation work: `CLAUDE.md`'s "Team Check-In" section still describes the pre-195 unconditional pause, and the folded one-worker todo still needs closing. This plan deliberately did not touch either, per its own `<action>`.
- The three guards are now the mechanism that will catch 195-10 if it updates `CLAUDE.md` in a way that contradicts the shipped surfaces.
- No blockers.

## Self-Check: PASSED

- All 9 declared modified files confirmed changed in `a7b00742` (`git diff --stat HEAD~1 HEAD`: 438 insertions, 6 deletions).
- Commit `a7b00742` confirmed present in `git log --oneline`.
- `git diff --diff-filter=D HEAD~1 HEAD` reports no deletions; no untracked files were produced by this plan.
- Every `<acceptance_criteria>` command and both plan-level `<verification>` commands were run with real output quoted above.
- `go build ./...`, `go vet ./cmd/`, `gofmt -l cmd/ pkg/` all clean.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
