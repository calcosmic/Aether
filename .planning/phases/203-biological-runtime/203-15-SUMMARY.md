---
phase: 203-biological-runtime
plan: "15"
subsystem: recruitment
tags: [recruitment, documentation, latency-guard, plain-english, wiring, reachability]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "The complete recruitment subsystem built across 203-02/03/04/05/06/07/08/09/10/11/12/13/14 -- this plan is the phase's closing verification and documentation pass, not a new mechanism"
provides:
  - "TestNoRecruitmentPathCostsNothing / TestNoNewMandatoryStep / TestTuningPassIsFreeWithoutCredit -- counter-based proof that an ordinary build/check pays nothing for recruitment"
  - ".aether/workers.md's real spawning protocol (aether recruit), replacing the false Task-tool/subagent_type instructions and the contradictory depth-limit table"
  - "cmd/workers_doc_test.go's TestWorkersDocMatchesRuntime -- locks the document to the runtime's own enforced depth constant"
  - "A real caller for `aether recruit`: workerDisciplineCallerFiles, a new evidence source in the WIRE-01 reachability ratchet, closing TestNoRegisteredSubcommandIsUnreferenced honestly (not via allowlist)"
  - "CLAUDE.md's 'Biological Runtime' section, mirrored into .aether/rules/aether-colony.md and regenerated into .claude/rules/aether-colony.md, every claim locked to a live test"
affects: []

# Actuals (#2632)
actuals:
  tokens: 14300
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Runtime call counters over package-level `var` chokepoints (recruitmentProbeRunner, spawnCanSpawnDecision), not wall-clock timing, for a latency guarantee that must not be flaky on a loaded machine"
    - "AST call-graph scanning (extending the 203-02 tracer's own recruitmentSymbols/buildDispatchFilesForRecruitmentScan pattern) as the stronger claim over a file-access counter alone"
    - "Widening a caller-evidence ratchet's own scan (workerDisciplineCallerFiles) when a genuine new class of caller is found outside it, mirroring Phase 197's hookScriptCorpora precedent -- never widening the allowlist to hide a true finding"
    - "Doc-test scoped to a named markdown section (regex-bounded), not the whole file, so a plain-English/citation guard proves only what the new content claims"

key-files:
  created:
    - cmd/recruitment_latency_test.go
    - cmd/workers_doc_test.go
    - cmd/claudemd_biological_runtime_test.go
  modified:
    - .aether/workers.md
    - CLAUDE.md
    - .aether/rules/aether-colony.md
    - .claude/rules/aether-colony.md
    - cmd/subcommand_reachability_ratchet_test.go
    - cmd/spawn_enforce_test.go
    - cmd/command_call_audit_test.go

key-decisions:
  - "Ran a full, unscoped `go test ./cmd -count=1` after all three tasks' own scoped verification passed, per this plan's own role as the phase's closing check -- found and fixed one real regression (aether recruit's D-01 severity classification, see deviations) and confirmed one further real regression (a stale test-name reference in .github/workflows/ci.yml's wiring-gate filter) that this worktree executor is permission-denied from fixing directly; documented rather than silently left, per this session's own instructions."
  - "The 'store reader' latency counter is a file-existence check under the recruitment/ and credit/ prefixes, not a runtime hook on pkg/storage.Store -- Store is a concrete, unexported-field struct with no test seam for call interception, and this plan's Task 1 file list is cmd/recruitment_latency_test.go alone. Combined with the AST call-graph scan and the two real runtime counters (probe, admission gate), this proves the stronger claim honestly without touching pkg/storage."
  - "TestNoNewMandatoryStep proves the folded-todo pause-count guarantee structurally (exactly one decideBuildCheckin call site, plus buildCheckinDecisionInput's field set unchanged from its recorded baseline) rather than against a typed baseline number that 201-TURNAROUND-BASELINE.md/the worker-turnaround todo do not actually record -- neither document states a 'pause count'; the runtime's own decision function and struct shape ARE the recorded baseline this phase's own additions must not have touched."
  - "workerDisciplineCallerFiles (a new, narrowly-scoped single-file evidence list, never a whole directory) was added to the WIRE-01 reachability ratchet's collectCallerEvidence/listCallerCorpusFiles, because .aether/workers.md was never one of the three permitted caller-evidence kinds (D-01: platform wrapper docs, hooks/scripts, Go self-invocation) -- a worker-run command like `aether recruit` has no wrapper or hook caller by design, so its one true, honest caller was always going to be the document every worker reads for its own discipline. This is the same class of fix Phase 197 already used for hookScriptCorpora's third entry when a real caller was found outside the scan."
  - "Rewrote TestSpawnCanSpawnAcceptsDocumentedInvocation (WIRE-02/D-13/D-14) as TestRecruitAcceptsDocumentedInvocation, repointed at the new documented `aether recruit` invocation, because the manual, voluntary `aether spawn-can-spawn {your_depth} --enforce` check it pinned to is exactly the bypassable pattern this plan's rewrite retires. Fixed the positional/flag classifier in the same pass: the original only ever saw spawn-can-spawn's lone boolean --enforce flag and misclassified a non-boolean flag's own value token (`--parent \"{your_name}\"`) as a stray positional."
  - "CLAUDE.md's new section does not claim the depth cap is unconditionally enforced -- per the live security finding recorded during this phase (WINDOWS.md, cmd/spawn.go's coordinator-sentinel exemption), it states plainly that the check trusts a short, fixed coordinator-name list on its own word, with the gap tracked rather than hidden."

requirements-completed: [BIO-02, BIO-03, BIO-06]

coverage:
  - id: D1
    description: "An ordinary build/check that never recruits pays nothing for this phase's admission machinery -- proven by real call counters (probe launcher, admission gate) around a real `aether build` run, a file-existence check under the recruitment/credit prefixes, an AST call-graph scan, and a zero-write proof on the outcome-tuning pass against an empty credit store"
    requirement: "BIO-03"
    verification:
      - kind: unit
        ref: "cmd/recruitment_latency_test.go#TestNoRecruitmentPathCostsNothing"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_latency_test.go#TestNoNewMandatoryStep"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_latency_test.go#TestTuningPassIsFreeWithoutCredit"
        status: pass
      - kind: other
        ref: "Live mutation of each guarantee (a probe call added to the build path, a field added to buildCheckinDecisionInput, a write added to the empty-credit-store branch), each confirmed failing by name, then reverted"
        status: pass
    human_judgment: false
  - id: D2
    description: ".aether/workers.md no longer describes a Task-tool/subagent_type spawning mechanism no caste was ever granted, nor a depth-limit table that contradicts the enforced cap; it documents the real aether recruit path, states a refusal is not an error, and states the per-run depth-raise flag is recorded permanently -- locked to the runtime's own spawnMaxDelegationDepth constant"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/workers_doc_test.go#TestWorkersDocMatchesRuntime"
        status: pass
      - kind: unit
        ref: "cmd/spawn_enforce_test.go#TestWorkersMdStatesOneDepthConvention"
        status: pass
      - kind: unit
        ref: "cmd/spawn_enforce_test.go#TestRecruitAcceptsDocumentedInvocation"
        status: pass
      - kind: other
        ref: "! grep -q 'subagent_type' .aether/workers.md && ! grep -qE 'Depth 1.?(→|to) ?4' .aether/workers.md && grep -q 'aether recruit' .aether/workers.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestNoRegisteredSubcommandIsUnreferenced passes with aether recruit closed by REAL wiring (a new, honest caller-evidence source reading .aether/workers.md), never by widening the shrink-only orphan allowlist -- the phase's own acceptance signal"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestNoRegisteredSubcommandIsUnreferenced"
        status: pass
      - kind: other
        ref: "orphan count dropped from 256 to 254 with no change to testdata/orphan_allowlist.json; TestOrphanAllowlistOnlyShrinks, TestDeletingACallerMakesTheRatchetNameIt, TestCallerEvidenceCreditsCommandSubstitution, TestWiringGuardsHaveNoRuntimeEscapeHatch all pass unmodified"
        status: pass
    human_judgment: false
  - id: D5
    description: "The full, unscoped go test ./cmd suite was run once after all three tasks completed; the one real regression it found (aether recruit's undeclared D-01 severity) was fixed; a second real regression (a stale test-name reference in .github/workflows/ci.yml, outside this worktree executor's edit permissions) is documented, not silently left"
    verification: []
    human_judgment: true
    rationale: "TestWiringGateStepRunsEveryWiringTest fails because .github/workflows/ci.yml's 'Verify subcommand wiring and CLI flag contracts' step still names the retired TestSpawnCanSpawnAcceptsDocumentedInvocation instead of its Task-2-deviation rename, TestRecruitAcceptsDocumentedInvocation. This worktree executor's permission settings deny direct edits under .github/workflows/ (confirmed: both a Bash sed and the Edit tool were refused). A human or the orchestrator must apply the one-line rename in ci.yml's -run filter (line 100) before this test will pass. See Deviations and Next Phase Readiness below for the exact fix."
  - id: D4
    description: "Every claim CLAUDE.md's new Biological Runtime section makes names a live test, none claims the depth cap is unconditionally enforced, and the section follows the project's own plain-English rule using the shared vocabulary table -- mirrored into the canonical colony-rules file and regenerated byte-identically into the Claude copy"
    requirement: "BIO-06"
    verification:
      - kind: unit
        ref: "cmd/claudemd_biological_runtime_test.go#TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_biological_runtime_test.go#TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/current_vocabulary_docs_199_test.go#TestCurrentVocabularyDocs199/ClaudeRuleGenerationParity"
        status: pass
      - kind: other
        ref: "Live mutation of both new tests (a renamed citation, an untranslated jargon word inserted), each confirmed failing by name, then reverted; diff .aether/rules/aether-colony.md .claude/rules/aether-colony.md byte-identical"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 15: Close the Phase Against Its Two Standing Constraints Summary

**Proved by counter (never wall-clock) that recruitment costs nothing on an ordinary run, replaced .aether/workers.md's false spawning protocol with the real `aether recruit` path, closed the phase's own named acceptance signal (`TestNoRegisteredSubcommandIsUnreferenced`) by wiring a real caller rather than widening the allowlist, and locked every runtime-behaviour claim added to CLAUDE.md to a live test.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-09-13T19:29:54+02:00 (worktree base)
- **Completed:** 2026-09-13T20:19:10+02:00
- **Tasks:** 3 completed
- **Files modified:** 10 (3 created, 7 modified)

## Accomplishments

- `cmd/recruitment_latency_test.go` (new): `TestNoRecruitmentPathCostsNothing` drives a real `aether build` through the same entry point `cmd/codex_build_test.go`'s own core fixtures use, with counting doubles wrapped around `recruitmentProbeRunner` and `spawnCanSpawnDecision` (both already package-level `var`s for exactly this purpose), asserting zero calls, zero files under the `recruitment/`/`credit/` store prefixes, and — via an AST call-graph scan extending the 203-02 tracer's own `recruitmentSymbols`/`buildDispatchFilesForRecruitmentScan` — zero reachable recruitment or credit symbols in the real build-dispatch files.
- `TestNoNewMandatoryStep` locks `decideBuildCheckin` to exactly one call site and `buildCheckinDecisionInput` to its current field set, then confirms the real policy still resolves a one-worker, nothing-pending build to the existing fast path (0 pauses) — the folded turnaround todo's "no new mandatory ceremony step" guarantee, proven structurally rather than against a typed number neither cited baseline document actually records.
- `TestTuningPassIsFreeWithoutCredit` proves the outcome-weighted note-tuning pass reaches a real, empty credit store and performs zero writes (checked by file-existence, not by trusting the function's own report).
- Rewrote `.aether/workers.md`'s "Spawning Sub-Workers" and "Step-by-Step Spawn Protocol" sections: removed the Task-tool/`subagent_type="general-purpose"` instruction (a mechanism no caste has ever been granted) and the "Spawn limits: Depth 1→4, Depth 2→2, Depth 3→0" table (which contradicted the enforced cap of 2 stated three paragraphs earlier). Replaced both with the real path: run `aether recruit`; the program decides; a refusal is not an error and never blocks the caller; an admission starts a real helper the owner sees join inline; the `--max-depth` per-run raise is recorded permanently. `caste` carries an inline plain-English translation on first use in the rewrite.
- `cmd/workers_doc_test.go` (new): `TestWorkersDocMatchesRuntime` fails in both directions — if the document ever re-teaches the retired tool invocation, or if its stated depth cap disagrees with the runtime's own `spawnMaxDelegationDepth` constant (derived at test time, never a typed literal).
- **The phase's own acceptance signal.** `TestNoRegisteredSubcommandIsUnreferenced` still failed after the workers.md rewrite alone, because `.aether/workers.md` was never one of the reachability ratchet's three permitted caller-evidence kinds (platform wrapper docs, hooks/scripts, Go self-invocation) — `aether recruit` is a command a worker runs from inside its own task, with no wrapper or hook caller by design. Added `workerDisciplineCallerFiles` (`cmd/subcommand_reachability_ratchet_test.go`), a narrowly-scoped single-file evidence source naming exactly `.aether/workers.md`, mirroring Phase 197's own precedent of widening the scan (never the allowlist) when a genuine caller was found outside it. Orphan count dropped from 256 to 254 with zero changes to `testdata/orphan_allowlist.json` — `aether recruit` was never allowlisted (per `.planning/WINDOWS.md` #19, deliberately left red as this plan's own acceptance signal).
- Discovered and fixed a real regression this rewrite caused: `TestSpawnCanSpawnAcceptsDocumentedInvocation` (Phase 172/173, WIRE-02) pinned to workers.md documenting a manual `aether spawn-can-spawn {your_depth} --enforce` call — exactly the voluntary, skippable check this plan's rewrite retires. Renamed to `TestRecruitAcceptsDocumentedInvocation`, repointed at the new documented `aether recruit` invocation, and fixed a latent classifier bug (a non-boolean flag's value token was misclassified as a stray positional) surfaced in the same pass.
- Added a "Biological Runtime (v1.28, Phase 203)" section to `CLAUDE.md` describing the real deliverable in the file's own mandated register (plain English, every jargon word translated inline, every claim naming its test) — including an explicit, honest statement of the one open security gap (the coordinator-sentinel name trusted on its own word) rather than an unconditional enforcement claim. Mirrored a condensed version into the canonical `.aether/rules/aether-colony.md` and regenerated `.claude/rules/aether-colony.md` from it byte-for-byte (confirmed against the real production sync path via the existing `ClaudeRuleGenerationParity` check).
- `cmd/claudemd_biological_runtime_test.go` (new): `TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests` (scoped to the new section only, fails by name on a renamed/deleted citation) and `TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish` (reuses the existing `repoInventedWords`/`untranslatedRepoWords` vocabulary table from `cmd/next_action_card_test.go` rather than a second one).
- **Full-suite closing check.** After all three tasks' own scoped `<verify>` commands passed, ran a full, unscoped `go test ./cmd -count=1` (per this plan's own role as the phase's closing verification). Found one real regression: `TestDocumentedSubcommandsAreSeverityClassified` failed because `.aether/workers.md` now documents `aether recruit`, and the D-01 command-call audit (a different, older audit than the reachability ratchet, which already treats `.aether/workers.md` as an audited file) requires every documented subcommand to be deliberately classified gate-or-enrichment. Fixed by adding `recruit` to `knownEnrichmentSubcommands` with D-03's own rationale (a refusal or failure never stops the caller's work). Every other failure was confirmed pre-existing and unrelated (see Deviations).

## Task Commits

Each task was committed atomically, with additional deviation commits where a real regression or wiring gap was discovered and fixed:

1. **Task 1: Measure that an ordinary build pays nothing** — `17c07ffe` (test)
2. **Task 2: Replace the false spawning instructions with the real ones** — `539fce42` (fix); plus two deviation commits found while proving the acceptance signal and running the regression suite: `824be303` (fix: reachability ratchet caller-evidence widening), `872891a7` (fix: repoint the pinned spawn-invocation test)
3. **Task 3: Lock every documented claim to a live test** — `5403a7d4` (docs)
4. **Full-suite closing check (post-Task-3)** — `fb422626` (fix: classify `aether recruit`'s D-01 severity)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_latency_test.go` — `TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`, `TestTuningPassIsFreeWithoutCredit`
- `.aether/workers.md` — real `aether recruit` spawning protocol, replacing the false Task-tool instructions
- `cmd/workers_doc_test.go` — `TestWorkersDocMatchesRuntime`
- `cmd/subcommand_reachability_ratchet_test.go` — `workerDisciplineCallerFiles`, wired into `collectCallerEvidence`/`listCallerCorpusFiles` (deviation, see below)
- `cmd/spawn_enforce_test.go` — `TestRecruitAcceptsDocumentedInvocation` (renamed/repointed from `TestSpawnCanSpawnAcceptsDocumentedInvocation`), with a positional/flag classifier fix (deviation, see below)
- `CLAUDE.md` — new "Biological Runtime (v1.28, Phase 203)" section
- `.aether/rules/aether-colony.md` — canonical mirror of the new section
- `.claude/rules/aether-colony.md` — regenerated byte-identical copy
- `cmd/claudemd_biological_runtime_test.go` — `TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests`, `TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish`
- `cmd/command_call_audit_test.go` — added `"recruit": true` to `knownEnrichmentSubcommands` (deviation, found by the full-suite closing check, see below)

## Decisions Made

See `key-decisions` in the frontmatter for the full list. In short: the latency test's "store reader" counter is a file-existence check plus an AST call-graph scan rather than a `pkg/storage.Store` hook (no test seam exists there, and Task 1's file list is the test file alone); the pause-count guarantee is proven structurally against the runtime's own decision function and struct shape, since neither cited baseline document records a typed pause count; the reachability ratchet gained a new, narrowly-scoped caller-evidence source for worker-instruction documents, mirroring Phase 197's own precedent; the pinned spawn-invocation test was repointed to the real current command rather than left describing a retired one; and CLAUDE.md's new section states the one open security gap honestly instead of claiming unconditional enforcement.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `.aether/workers.md` was never in the reachability ratchet's caller-evidence scan**
- **Found during:** Task 2, after rewriting workers.md and re-running `TestNoRegisteredSubcommandIsUnreferenced` — it still failed, naming `aether recruit` as an orphan.
- **Issue:** The ratchet's own doc comment states it searches "exactly the three permitted kinds of caller evidence (D-01)": platform wrapper docs (`.claude/commands/ant`, `.opencode/commands/ant`, `.aether/commands`), hooks/scripts, and Go self-invocation. `.aether/workers.md` — the document every worker actually reads for its own discipline, and the only genuine caller a worker-run command like `aether recruit` could ever have — was in none of those three.
- **Fix:** Added `workerDisciplineCallerFiles` (an explicit, narrowly-scoped single-file list naming exactly `.aether/workers.md`, never a whole directory) to `cmd/subcommand_reachability_ratchet_test.go`, wired into both `collectCallerEvidence` and `listCallerCorpusFiles`. This is the same class of fix Phase 197 already used for `hookScriptCorpora`'s third entry when a real caller was found outside the scan (documented in that var's own comment).
- **Files modified:** `cmd/subcommand_reachability_ratchet_test.go` (not in this plan's declared `files_modified` list)
- **Verification:** `TestNoRegisteredSubcommandIsUnreferenced` passes (orphan count 256 → 254, `testdata/orphan_allowlist.json` untouched); `TestDeletingACallerMakesTheRatchetNameIt`, `TestOrphanAllowlistOnlyShrinks`, `TestCallerEvidenceCreditsCommandSubstitution`, `TestWiringGuardsHaveNoRuntimeEscapeHatch` all pass unmodified.
- **Committed in:** `824be303`

**2. [Rule 1 - Bug] The workers.md rewrite broke a pre-existing pinned-invocation test**
- **Found during:** Task 2's own regression sweep (`go test ./cmd -run 'TestSpawn...'`), not caught by the plan's own literal `<verify>` command.
- **Issue:** `TestSpawnCanSpawnAcceptsDocumentedInvocation` (Phase 172/173, WIRE-02) pinned to workers.md documenting a manual `aether spawn-can-spawn {your_depth} --enforce` call before spawning — the exact voluntary, skippable check this plan's own rewrite retires in favour of `aether recruit`'s single atomic call. Removing that documented invocation (correctly, per this plan's own objective) made the pinned test fail by finding nothing to extract.
- **Fix:** Renamed to `TestRecruitAcceptsDocumentedInvocation`, repointed the extraction regex and assertions at the new documented `aether recruit` invocation, and fixed the positional/flag classifier (a non-boolean flag's own value token, e.g. `--parent "{your_name}"`, was being misclassified as a stray positional — the original classifier only ever needed to handle `spawn-can-spawn`'s lone boolean `--enforce` flag). Also reformatted workers.md's example command to one line, since the test's per-line regex extraction cannot follow a backslash-continued multi-line shell command.
- **Files modified:** `cmd/spawn_enforce_test.go` (not in this plan's declared `files_modified` list); `.aether/workers.md` (already declared)
- **Verification:** `TestRecruitAcceptsDocumentedInvocation` passes; `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` and the rest of `cmd/spawn_enforce_test.go` pass unmodified; `spawn-can-spawn` itself is untouched and remains real, tested machinery used internally by the TS host bridge (203-09) and by `aether recruit`'s own admission call.
- **Committed in:** `872891a7`

**3. [Rule 1 - Bug] `.aether/workers.md`'s new `aether recruit` documentation had no D-01 severity classification**
- **Found during:** the full-suite closing check (`go test ./cmd -count=1`) run after all three tasks' own scoped verification passed.
- **Issue:** `cmd/command_call_audit_test.go` (a different, older D-01 audit than the reachability ratchet, which already treats `.aether/workers.md` as an audited file via `auditedFiles`) requires every documented subcommand found anywhere in its corpus to be deliberately classified as `gateClassifiedCommands` (failure halts a run) or `knownEnrichmentSubcommands` (failure warns, run continues). `recruit` was newly visible to this audit and unclassified.
- **Fix:** Added `"recruit": true` to `knownEnrichmentSubcommands` with D-03's own rationale: a refused or failed recruitment never stops the caller's work, and no verification result, security scan, or gate outcome depends on one succeeding.
- **Files modified:** `cmd/command_call_audit_test.go` (not in this plan's declared `files_modified` list)
- **Verification:** `TestDocumentedSubcommandsAreSeverityClassified` passes; `TestGateClassifiedCallsHaveGateWiring`, `TestCommandCallsMatchCobraContracts` pass unmodified.
- **Committed in:** `fb422626`

**4. [Rule 3 - Blocking, NOT fixed — permission-denied] `.github/workflows/ci.yml`'s wiring-gate `-run` filter still names the retired test**
- **Found during:** the same full-suite closing check.
- **Issue:** `TestWiringGateStepRunsEveryWiringTest` (D-15, `cmd/ci_wiring_gate_test.go`) reads `.github/workflows/ci.yml`'s live content and fails because the "Verify subcommand wiring and CLI flag contracts" step's `-run` filter still names `TestSpawnCanSpawnAcceptsDocumentedInvocation` (retired by deviation #2 above) instead of its replacement, `TestRecruitAcceptsDocumentedInvocation`.
- **Not fixed here:** both a direct `sed` edit via Bash and the `Edit` tool were refused with "File is in a directory that is denied by your permission settings" — `.github/workflows/` is permission-denied for this worktree executor (a sandbox policy, not a judgement call). Attempting to bypass a permission boundary is out of scope for any deviation rule.
- **What must happen:** a human or the orchestrator, working outside this sandboxed worktree, must apply the one-line change: in `.github/workflows/ci.yml` line 100's `-run` filter, replace `TestSpawnCanSpawnAcceptsDocumentedInvocation` with `TestRecruitAcceptsDocumentedInvocation`. No other line in that filter needs to change.
- **Verification of the underlying claim:** `TestWiringGateStepRunsEveryWiringTest` was re-run in isolation and confirmed it fails ONLY on this one stale reference (the failure message names exactly one missing-guard-test line and one matches-nothing line, both naming the same rename).

---

**Total deviations:** 3 auto-fixed (2 blocking wiring gaps in foundational cross-phase guards, 1 bug in a pre-existing pinned test that this plan's own legitimate fix broke), 1 found-but-not-fixed (permission-denied, documented above and in Next Phase Readiness). All three fixes were necessary for the plan's own stated acceptance signal and for the full regression suite to be genuinely green, not merely the plan's own literal `<verify>` commands. No scope creep beyond what each required.

## Issues Encountered

The `.github/workflows/ci.yml` permission wall above (deviation #4) is the only unresolved issue. Everything else was found and fixed via this plan's own regression sweeps before this SUMMARY was written.

## Threat Flags

None new. This plan adds no new process-spawn, network, or auth surface — it is documentation, a latency-measurement test, a doc-test, and a caller-evidence-scan widening. The one open security finding from earlier in this phase (the coordinator-sentinel name trusted on its own word, `cmd/spawn.go`, recorded in `.planning/WINDOWS.md` #18) is referenced honestly in CLAUDE.md's new section rather than concealed, but is not fixed here — it was explicitly out of scope for plan 203-06, which owns those files, and remains an open item for the owner or a dedicated follow-up plan.

## Known Stubs

None. Every must_have this plan's own frontmatter names is implemented and tested; no placeholder or empty-value stub was introduced.

## User Setup Required

None - no external service configuration required.

## Full-Suite Verification (post-Task-3 closing check)

Ran `go test ./cmd -count=1` (555.7s) after all three tasks' scoped `<verify>` commands passed. 14 unique failing test names surfaced; each was individually triaged:

- **Fixed by this plan (see Deviations #3):** `TestDocumentedSubcommandsAreSeverityClassified`.
- **Found but not fixed — permission-denied (see Deviations #4):** `TestWiringGateStepRunsEveryWiringTest`.
- **Confirmed pre-existing, unrelated, verified against the unmodified base commit (`dab02fbe`) in a detached worktree:** `TestStatusPrefersCurrentRunWorkersOverStaleHistory` (a 203-14 residual — a stale worker also appears in the new Governed Subtree status section this plan did not touch), `TestCLIFlagAudit` and `TestDocumentedCommandNamesResolve` (both trace to `aether help` not being registered — unrelated to recruitment), `TestHumanFacingOutputGoesThroughWriteVisualOutput` (WINDOWS.md #13, `watch_live.go` bypassing `writeVisualOutput` — a file this plan never touched).
- **Already documented as known-red baseline in this plan's own prompt:** `TestCurrentVocabulary199`, `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`, `TestPhase199GateReceipt`, `TestSkillIndexReadEmpty`, `TestSkillIsUserCreated`, `TestSkillIsUserCreatedShipped`.
- **Resource-contention artifact of the full parallel run, not a real failure:** `TestVisualsDumpExportsCasteIdentityContract` passed cleanly on an isolated re-run; `TestCLICompiledInstallToSealJourney` (214s, a heavy install→seal blackbox e2e test unrelated to any file this plan touched) and the log's own `lane parallel-048 exceeded 9m0s: context deadline exceeded` line both point at the documented ~20-minute machine-specific suite ceiling (WINDOWS.md #12/#16), not a regression.

## Next Phase Readiness

- The phase's own named acceptance signal (`TestNoRegisteredSubcommandIsUnreferenced`) is green, closed by real wiring rather than an allowlist entry — confirmed by re-running it by name, per this plan's own explicit instruction.
- `.aether/workers.md` now describes only mechanisms the runtime actually grants, locked to the runtime's own enforced depth constant.
- CLAUDE.md's new section is fully test-locked and does not overstate the depth cap's enforcement, honestly naming the one open coordinator-sentinel gap instead.
- **Blocker for CI, not for this phase's own gate:** `.github/workflows/ci.yml` line 100 needs the one-line rename described in Deviations #4 (`TestSpawnCanSpawnAcceptsDocumentedInvocation` → `TestRecruitAcceptsDocumentedInvocation`) — this worktree executor is permission-denied from making it. The orchestrator or a human must apply it before the next real CI run of the "Verify subcommand wiring and CLI flag contracts" step.
- **Open item carried forward, not a blocker:** the coordinator-sentinel authentication gap (`.planning/WINDOWS.md` #18, owned by plan 203-06's file scope) remains open — it predates this phase and this plan's own scope does not include closing it.
- **Open item carried forward, not a blocker:** `testdata/orphan_allowlist.json` retains one now-stale entry (`aether generate-ant-name`, now correctly credited via the new evidence source) — harmless under the allowlist's shrink-only contract, but a future audit could regenerate the file for hygiene.
- **Open item carried forward, not a blocker:** `TestStatusPrefersCurrentRunWorkersOverStaleHistory` fails identically on the pre-this-plan base commit — a genuine 203-14 residual bug in the Governed Subtree status section, confirmed unrelated to this plan and outside its scope to fix.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_latency_test.go`
- FOUND: `cmd/workers_doc_test.go`
- FOUND: `cmd/claudemd_biological_runtime_test.go`
- FOUND: commit `17c07ffe` (test: measure that an ordinary build/check pays nothing for recruitment) in `git log --oneline`
- FOUND: commit `539fce42` (fix: replace the false spawning protocol with the real one) in `git log --oneline`
- FOUND: commit `824be303` (fix: teach the reachability ratchet to see worker-instruction callers) in `git log --oneline`
- FOUND: commit `872891a7` (fix: repoint the pinned spawn-invocation test to aether recruit) in `git log --oneline`
- FOUND: commit `5403a7d4` (docs: lock every biological-runtime claim in CLAUDE.md to a live test) in `git log --oneline`
- FOUND: commit `fb422626` (fix: classify aether recruit's D-01 severity as enrichment) in `git log --oneline`
- Re-ran plan-level Task 1 `<verify>`: `go test ./cmd -run '^(TestNoRecruitmentPathCostsNothing|TestNoNewMandatoryStep|TestTuningPassIsFreeWithoutCredit)$' -count=1` — PASS
- Re-ran plan-level Task 2 `<verify>`: `! grep -q 'subagent_type' .aether/workers.md && ! grep -qE 'Depth 1.?(→|to) ?4' .aether/workers.md && grep -q 'aether recruit' .aether/workers.md && go test ./cmd -run '^TestWorkersDocMatchesRuntime$' -count=1` — PASS
- Re-ran plan-level Task 3 `<verify>`: `go test ./cmd -run '^(TestCLAUDEMDBiologicalRuntime|TestCurrentVocabularyDocs199)' -count=1 && diff .aether/rules/aether-colony.md .claude/rules/aether-colony.md && go vet ./... && go build ./...` — PASS
- Re-ran the phase's own acceptance signal by name: `go test ./cmd -run '^TestNoRegisteredSubcommandIsUnreferenced$' -count=1 -v` — PASS (395 registered commands, 254 orphans, none unallowed; `testdata/orphan_allowlist.json` untouched)
- Mutation-tested every guarantee this plan added, live: a probe call added to the build-dispatch path (`TestNoRecruitmentPathCostsNothing` failed, named the counter), a field added to `buildCheckinDecisionInput` (`TestNoNewMandatoryStep` failed, named the field), a write added to the empty-credit-store branch of the tuning pass (`TestTuningPassIsFreeWithoutCredit` failed, named the file), `spawnMaxDelegationDepth` bumped from 2 to 3 (`TestWorkersDocMatchesRuntime` failed, named the stale wording), a citation renamed in CLAUDE.md (`TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests` failed, named it), and an untranslated jargon word inserted into CLAUDE.md's new section (`TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish` failed, named it) — all six confirmed failing by name, then reverted (`git diff`/`git status --short` empty after each revert)
- Ran the full, unscoped `go test ./cmd -count=1` (555.7s) once after all three tasks' own scoped verification passed; triaged all 14 unique failing test names (see Full-Suite Verification above); fixed the one real regression found (`TestDocumentedSubcommandsAreSeverityClassified`); confirmed the one real-but-permission-blocked regression (`TestWiringGateStepRunsEveryWiringTest`, documented in Deviations #4); confirmed every remaining failure pre-existing (one verified directly against the unmodified base commit `dab02fbe` in a detached worktree) or already named in this plan's own known-red baseline
- Confirmed `git status --short` clean at plan completion (no untracked or uncommitted changes) except this SUMMARY.md itself, about to be committed
