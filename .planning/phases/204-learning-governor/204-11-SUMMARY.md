---
phase: 204-learning-governor
plan: "11"
subsystem: learning
tags: [claude-md, classic-contract, defect-register, plain-english, learning-governor, closing-plan]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md's twelve SYN-204 dispositions and rulings (a)-(f), the mechanism registry entries this plan's contract cases cite"
  - phase: 204-02
    provides: "recordPhaseApplicationCredit, learningVerifiedEntries/learningUnverifiedEntries -- cited by this plan's CLAUDE.md claims and contract cases"
  - phase: 204-03
    provides: "the memory-schema/provenance contract and the owner's field-census decision (instinct.related_instincts retired; midden.acknowledge_reason/learn.parent_id/pheromone.scope knowingly empty) -- recorded here in the authority doc and WINDOWS.md"
  - phase: 204-04
    provides: "cmd/episode_ledger.go's durable outcome ledger -- cited by this plan's CLAUDE.md claims and contract cases, and the swarm/recovery coverage gap recorded as a new open limit"
  - phase: 204-05
    provides: "the fixture bank -- cited by this plan's CLAUDE.md claims and contract cases, and its unguarded-fixture floor recorded as a new open limit"
  - phase: 204-06
    provides: "the nine-state guidance application ledger and the helpful-application promotion gate -- cited by this plan's CLAUDE.md claims"
  - phase: 204-07
    provides: "the seven named eval gates -- cited by this plan's CLAUDE.md claims and contract cases"
  - phase: 204-08
    provides: "pkg/shadow's sealed evaluator and cmd/shadow_cmds.go -- cited by this plan's CLAUDE.md claims and contract cases, and the placeholder-grader/settings-only-scope limit recorded as a new open limit"
  - phase: 204-09
    provides: "cmd/promotion_gate.go's authority-refusal boundary and cmd/rollback.go's atomic rollback -- cited by this plan's CLAUDE.md claims, contract cases, and the authority-doc decision record"
  - phase: 204-10
    provides: "cmd/improvement_report.go's two-figure report and cmd/source_proposal.go's propose-only boundary -- cited by this plan's CLAUDE.md claims and contract cases, and the missing-caller/no-intervention-vocabulary limit recorded as a new open limit"
provides:
  - "CLAUDE.md's '## Learning Governor (v1.28, Phase 204)' section, citing 22 distinct live tests across ten claims, closing with the two-promotable/nine-retained authority scope and a plain-English summary"
  - "cmd/claudemd_learning_governor_test.go: the removal-proof guard for that section, plus the phase's one new invented word (\"canary\") extended into a locally-scoped plain-English check"
  - "12 new executable Classic contract cases (cmd/testdata/classic-contract/v1/cases.json), one per registered SYN-204 mechanism, and TestClassicContractPhase204Cases proving full coverage plus two negative fixtures"
  - "The canary authority-boundary decision in .aether/docs/learning-system-authority.md: two promotable scopes, nine retained-authority scopes, and the structural (code-path) enforcement argument"
  - "The condensed Learning Governor section in both rule-file copies, byte-identical"
  - "Six new WINDOWS.md open-limit entries (40-45) plus one migrated pre-existing data-integrity fix (entry 46, closing a stray duplicate row)"
affects: []

# Actuals (#2632) -- chars/4 over the realized diff (9edbdecb..HEAD), never a harness token count.
actuals:
  tokens: 18190
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Section-scoped CLAUDE.md removal-proof guard (readLearningGovernorSection + citedTestNameRe), an exact structural copy of the Biological Runtime section's own guard, applied to a new heading rather than generalized into a shared helper -- matching this file's own established one-guard-per-phase-section precedent."
    - "Locally-scoped plain-English vocabulary extension (learningGovernorInventedWords): a superset of the shared repoInventedWords table plus the phase's one new invented word (\"canary\"), defined in the new test file rather than editing the shared table in an out-of-scope file."
    - "Behavior-only Classic contract cases follow the same shape SYN-200/201/202 already established (an 'existing-go-test-fixture' citing a real, already-passing Go test, plus a generic COLONY_STATE.json causal state assertion) rather than re-deriving a new case shape for SYN-204."

key-files:
  created:
    - cmd/claudemd_learning_governor_test.go
  modified:
    - CLAUDE.md
    - cmd/classic_contract_test.go
    - cmd/testdata/classic-contract/v1/cases.json
    - .aether/rules/aether-colony.md
    - .claude/rules/aether-colony.md
    - .aether/docs/learning-system-authority.md
    - .planning/WINDOWS.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "Every test cited in CLAUDE.md's new section and in the 12 new contract cases was chosen from cmd-package-only tests (never pkg/shadow, even though pkg/shadow/isolation_test.go's TestCandidateCannotAlterItsEvaluatorDigest is the most on-the-nose citation for the shadow-isolation claim) -- the removal-proof guard mirrors Biological Runtime's exactly and only globs *_test.go inside cmd/, so a pkg/-only citation would silently fail that guard. cmd/shadow_cmds_test.go's TestCandidateStoreIsAppendOnlyAndRefusesEdits/TestComparisonResultReachesTheDurableLedger/TestHoldoutResolutionHappensOnlyInTheCommandLayer stand in for the same causal guarantee from the cmd-package command layer instead."
  - "learningGovernorInventedWords is a local superset defined in the new test file rather than an edit to the shared repoInventedWords table in cmd/next_action_card_test.go (outside this plan's declared files_modified) -- widening the shared table would add a check with nothing to catch (no rendered next-action card ever says \"canary\"), while the local copy keeps detection scoped to the section that actually introduces the word and still fails, not passes vacuously, if a later edit drops its inline translation."
  - "The 12 new contract cases follow SYN-200/201/202's 'existing-go-test-fixture' shape (citing an already-passing Go test plus a generic .aether/data/COLONY_STATE.json causal state assertion) rather than the SYN-199 required-journey shape or a fresh causal-execution harness -- Task 2's own acceptance criteria name only TestClassicContractSchema/TestClassicMechanismCoverage/TestClassicContractPhase204Cases/TestClassicContractPhase204MechanismRegistry, not a causal-execution test, so no TestClassicContractPhase204CausalExecution was added."
  - "Nine of the twelve SYN-204 mechanism disposition rows carry no Classic ancestor (per 204-CLASSIC-SYNTHESIS.md's own comparative matrix); their cases' historical_anchors honestly cite 'no-classic-ancestor:<synthesis-doc-anchor>' rather than inventing a plausible-looking OLD- id the evidence file does not actually support."
  - "The retained-authority list's nine categories are each mapped to the owner as the retaining authority in the .aether/docs/learning-system-authority.md decision record -- none is mapped to an 'independent reviewer' instead, because 204-CLASSIC-SYNTHESIS.md's own routed requirement (LEARN-07) never names an independent-review path for any of the nine, only owner approval through the existing tick-to-approve queue."

patterns-established:
  - "A phase-closing plan (the '-11' / '-15' style closer other Phase 20x's used) writes the CLAUDE.md section, the contract cases, and the WINDOWS.md open-limit ledger entries as three separably-committed tasks, each independently verified, rather than one combined documentation commit."

requirements-completed: [LEARN-01, LEARN-07, LEARN-08]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "CLAUDE.md's '## Learning Governor (v1.28, Phase 204)' section: every claim about runtime behaviour cites a live test, a citation naming a nonexistent test fails the guard by name, a section citing none fails, and the plain-English mandate holds including the phase's one new invented word."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/claudemd_learning_governor_test.go#TestCLAUDEMDLearningGovernorClaimsCiteLiveTests"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_learning_governor_test.go#TestCLAUDEMDLearningGovernorSectionIsPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_biological_runtime_test.go#TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests (regression, still passes)"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_biological_runtime_test.go#TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish (regression, still passes)"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_classic_voice_test.go#TestEveryVoiceClaimInCLAUDEMDNamesALiveTest (regression, still passes)"
        status: pass
    human_judgment: false
  - id: D2
    description: "12 executable Classic contract cases, one per registered SYN-204 mechanism, each in its own V-204-* group with a real command and a causal state assertion; every registered mechanism has at least one case; a case citing an unregistered decision or an unknown group fails by name."
    requirement: "SYNTH-06"
    verification:
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase204Cases"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase204MechanismRegistry (regression, still passes)"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractSchema (regression, still passes)"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicMechanismCoverage (regression, still passes)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both rule-file copies carry the condensed section and are byte-identical; the authority record names the two promotable and nine retained-authority scopes plus the tests enforcing the boundary; every limit this phase leaves open is recorded in WINDOWS.md with what would close it, and the register's own front-matter counts match its table."
    requirement: "LEARN-07"
    verification:
      - kind: unit
        ref: "cmd/current_vocabulary_docs_199_test.go#TestCurrentVocabularyDocs199"
        status: pass
      - kind: other
        ref: "diff .aether/rules/aether-colony.md .claude/rules/aether-colony.md"
        status: pass
      - kind: other
        ref: "python3 WINDOWS.md front-matter-vs-table-count check (plan Task 3 <verify>)"
        status: pass
    human_judgment: false

# Metrics
duration: 130min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 11: Learning Governor Closing Documentation Summary

**Wrote the phase's one CLAUDE.md section (22 distinct cited tests across ten claims), registered all twelve SYN-204 mechanisms as executable Classic contract cases, updated both rule-file copies and the authority record with the canary boundary decision, and recorded six new open limits in WINDOWS.md -- plus fixed a pre-existing data-integrity bug in WINDOWS.md discovered while verifying this plan's own acceptance criterion.**

## Performance

- **Duration:** ~130 min
- **Started:** ~2026-09-14T13:35:00Z (estimated from session context)
- **Completed:** 2026-09-14T15:43:29Z
- **Tasks:** 3/3 completed
- **Files modified:** 9 (1 created, 8 modified, including REQUIREMENTS.md)

## Accomplishments

- `CLAUDE.md` gained a `## Learning Governor (v1.28, Phase 204)` section citing 22 distinct live tests across the ten claims the plan's own action text specified, closing with the two-promotable/nine-retained-authority scope table in prose and a plain-English "For dummies" paragraph that translates every invented word inline, including the phase's one new term ("canary").
- `cmd/claudemd_learning_governor_test.go` is the removal-proof guard: `TestCLAUDEMDLearningGovernorClaimsCiteLiveTests` fails naming any cited test that doesn't resolve to a real `cmd`-package function (and fails if the section cites fewer than 10 distinct tests); `TestCLAUDEMDLearningGovernorSectionIsPlainEnglish` reuses the shared `repoInventedWords` translation rule, extended locally with "canary".
- 12 new cases in `cmd/testdata/classic-contract/v1/cases.json`, one per registered `SYN-204-01` through `SYN-204-12`, each declaring its `V-204-*` group, a real `command` (`continue`/`patrol`/`seal`), a causal `.aether/data/COLONY_STATE.json` state assertion, and a citation to the real, already-passing Go test proving the behaviour. `TestClassicContractPhase204Cases` (new) loads the corpus through the existing strict loader, asserts full mechanism/group coverage, and proves by negative fixture that an unregistered `SYN-204` decision and an unknown group are each refused by name.
- `.aether/rules/aether-colony.md` (canonical) and `.claude/rules/aether-colony.md` (generated) both gained the condensed Learning Governor section and are byte-identical (`diff` confirms).
- `.aether/docs/learning-system-authority.md` gained "Decision 6," recording the canary authority boundary: two promotable scopes (project knowledge, routing) and nine retained-authority scopes (preferences, skills, workflows, source, security, deletion, permissions, verification, external actions), each mapped to the owner, with the structural (code-path, not environmental) enforcement argument and the tests proving it.
- `.planning/WINDOWS.md` gained six new open-limit entries (40-45, covering the shadow-compare placeholder grader, the swarm/recovery episode-ledger coverage gap, the fixture-bank guard floor, the 204-03 owner field-census decision, the missing automatic learning-validation path, and `proposeSourceImprovement`'s missing caller) plus one migrated-and-fixed entry (46) that closes a pre-existing data-integrity bug discovered while verifying this task's own front-matter/table-count acceptance criterion.
- `LEARN-01`, `LEARN-07`, and `LEARN-08` — all shared across multiple Phase 204 plans — are now ready (every declaring plan complete) and marked complete in `REQUIREMENTS.md`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the Learning Governor section, with a test beside every claim** - `0e8f44cb` (docs)
2. **Task 2: Register this phase's behaviours as executable contract cases** - `d2a72b93` (test)
3. **Task 3: Update the rule copies and the authority record, and write down what is still open** - `b6b3b6b1` (docs)

**Plan metadata:** committed alongside this SUMMARY (worktree mode -- STATE.md/ROADMAP.md excluded, handled by the orchestrator).

## Files Created/Modified

- `cmd/claudemd_learning_governor_test.go` - `TestCLAUDEMDLearningGovernorClaimsCiteLiveTests`, `TestCLAUDEMDLearningGovernorSectionIsPlainEnglish`, `learningGovernorInventedWords`, `untranslatedLearningGovernorWords`
- `CLAUDE.md` - New `## Learning Governor (v1.28, Phase 204)` section
- `cmd/classic_contract_test.go` - `TestClassicContractPhase204Cases`; widened `validateClassicContractCorpus`'s SYN-200/201/202 exclusion to also exclude SYN-203/SYN-204
- `cmd/testdata/classic-contract/v1/cases.json` - 12 new Phase 204 cases
- `.aether/rules/aether-colony.md` - Condensed `## Learning Governor (v1.28, Phase 204)` section
- `.claude/rules/aether-colony.md` - Byte-identical copy
- `.aether/docs/learning-system-authority.md` - "Decision 6 -- Phase 204's canary authority boundary"
- `.planning/WINDOWS.md` - Entries 40-46 (six new open limits, one migrated-and-fixed pre-existing bug); stray duplicate row removed
- `.planning/REQUIREMENTS.md` - `LEARN-01`, `LEARN-07`, `LEARN-08` marked complete

## Decisions Made

See `key-decisions` in frontmatter above for the full account of the four substantive ones (cmd-package-only test citations, the locally-scoped invented-word extension, the existing-go-test-fixture case shape, and the honest "no-classic-ancestor" historical anchors).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `validateClassicContractCorpus` had no exclusion for SYN-203-/SYN-204- decisions**
- **Found during:** Task 2, first run of `TestClassicContractPhase204Cases`
- **Issue:** `cmd/classic_contract_test.go`'s `validateClassicContractCorpus` only excludes `SYN-200-`/`SYN-201-`/`SYN-202-` prefixed cases from its Phase-199-shaped "required journey" validation (fixed `.claude`/`.opencode` platform pairs, mandatory `PROOF-01` citation). This plan is the first to add `SYN-204-` cases to the corpus (no `SYN-203-` cases exist either), so without widening the exclusion, every new case would have been wrongly bucketed as an unregistered Phase 199 journey and failed validation.
- **Fix:** Widened the exclusion condition to also skip `SYN-203-` and `SYN-204-` prefixed cases, documented inline with why this plan is the one that surfaced the gap.
- **Files modified:** `cmd/classic_contract_test.go`
- **Verification:** `TestClassicContractPhase204Cases` and the full Task 2 named test group pass; SYN-199 through SYN-202 case counts unchanged (verified by direct count before/after).
- **Committed in:** `d2a72b93`

**2. [Rule 1 - Bug] Pre-existing stray duplicate row in `.planning/WINDOWS.md`, discovered by this task's own acceptance criterion**
- **Found during:** Task 3, running the plan's own front-matter-vs-table-count `<verify>` check
- **Issue:** A row with `id=14` (phase `202.1`, `pkg/codex/platform_dispatch.go`, describing a real `AETHER_WORKER_NAME` wiring fix) sat on the file's own final line, below the closing ```` ```` ```` fence of the JSON ledger block — outside the ledger tool's managed data entirely. It collided with the genuine, JSON-backed entry `14` (phase 203, `pkg/codex/worker.go`), making the visible markdown table carry one more `fixed` row (10) than the front-matter's `fixed_count` (9) recorded, failing this task's own acceptance criterion. 204-05-SUMMARY.md had already flagged this exact stray-row phenomenon as a reason to parse the JSON block rather than the table, but had not fixed it.
- **Fix:** Migrated the stray row's content into the ledger properly via `gsd-tools windows append` (new entry 46, preserving its original description verbatim and noting the original `2026-09-12T21:30:00.000Z` timestamps in the description text since the tool re-stamps `recorded_at`/`resolved_at`), marked it `fixed` via `gsd-tools windows fixed 46`, then removed the now-redundant orphaned raw markdown line.
- **Files modified:** `.planning/WINDOWS.md`
- **Verification:** The plan's own Task 3 python front-matter/table-count check now passes (`open_count=36` matches 36 open rows, `fixed_count=10` matches 10 fixed rows, `waived_count=0` matches 0 waived rows); the JSON ledger block re-parses cleanly with 46 entries, last id 46.
- **Committed in:** `b6b3b6b1`

---

**Total deviations:** 2 auto-fixed (1 Rule 3 blocking, 1 Rule 1 bug — both required to satisfy this plan's own stated `<acceptance_criteria>`). **Impact on plan:** None on any machine-checkable gate — every `<acceptance_criteria>` line and every task's `<verify>` command passes; both fixes strengthened existing machinery (the corpus validator now correctly scopes to the cases it actually validates; the defect register's own counts are now honest) rather than working around it.

## Issues Encountered

None beyond the deviations documented above. `go build ./...` and `go vet ./cmd/...` are clean; `gofmt -l` reports nothing for either file this plan created or touched in Go source (`cmd/claudemd_learning_governor_test.go`, `cmd/classic_contract_test.go`) — the five files `gofmt -l` reports elsewhere in `cmd/` (`codex_build.go`, `lifecycle_wrapper_contract_test.go`, `pheromone_approval_test.go`, `swarm_cmd.go`, `work_repair_test.go`) are pre-existing and untouched by this plan, out of scope per the deviation rules' scope boundary.

## User Setup Required

None -- no external service configuration required.

## Known Stubs

None. This plan's own deliverables (a documentation section, contract cases, rule-file updates, and defect-register entries) have no runtime data path that could be stubbed.

## Threat Flags

None found. This plan touches no new network endpoint, auth path, file-access pattern, or schema change at a trust boundary — it is documentation, test-data, and defect-register work over already-shipped code.

## Next Phase Readiness

Phase 204 (Learning Governor) is fully documented and its behaviours are registered as executable Classic contract cases. `LEARN-01`, `LEARN-07`, and `LEARN-08` are all marked complete. Six genuinely open limits from this phase are now visible in `.planning/WINDOWS.md` (entries 40-45) for the acceptance phase to weigh, alongside the two the orchestrator already recorded mid-wave (38, 39). No plan in this phase's own scope claims any of these six as closed — each names what would close it, per the plan's own Assumption X.

No blockers.

## Self-Check: PASSED

- All 8 key files (1 created, 8 modified including REQUIREMENTS.md) confirmed present via `git status`/`git log`.
- All 3 task commits (`0e8f44cb`, `d2a72b93`, `b6b3b6b1`) confirmed present via `git log --oneline`.
- `grep -c '^## Learning Governor (v1.28, Phase 204)$' CLAUDE.md` returns 1; `git diff --stat CLAUDE.md` shows insertions only, confined to the new section.
- `diff .aether/rules/aether-colony.md .claude/rules/aether-colony.md` returns no output (byte-identical).
- `grep -q 'retained' .aether/docs/learning-system-authority.md` succeeds.
- Full named test set for all three tasks re-run together and passes: `go test ./cmd -run '^(TestCLAUDEMDLearningGovernorClaimsCiteLiveTests|TestCLAUDEMDLearningGovernorSectionIsPlainEnglish|TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests|TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish|TestEveryVoiceClaimInCLAUDEMDNamesALiveTest|TestClassicContractSchema|TestClassicMechanismCoverage|TestClassicContractPhase204Cases|TestClassicContractPhase204MechanismRegistry|TestCurrentVocabularyDocs199)$' -count=1 -timeout 10m` -> `ok`.
- `go build ./...` and `go vet ./cmd/...` both clean.
- `python3` front-matter-vs-table-count check on `.planning/WINDOWS.md` passes (exit 0).
- `LEARN-01`/`LEARN-07`/`LEARN-08` confirmed marked complete in `.planning/REQUIREMENTS.md` via `requirements.mark-complete` (ready-ids reported 3/3 ready before marking).

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
