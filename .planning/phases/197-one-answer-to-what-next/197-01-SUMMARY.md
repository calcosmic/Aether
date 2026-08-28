---
phase: 197-one-answer-to-what-next
plan: 01
subsystem: cli
tags: [go, cobra, next-action, resolver, availability-gate]

requires:
  - phase: 193-the-queen-decides
    provides: the Queen-owned decision pattern this resolver follows — one place decides, tests assert the real output rather than an intermediate record
provides:
  - "resolveNextAction — one pure function that answers all eight of NEXT-01's questions from the saved state"
  - "nextActionCandidates — the single enumerable set holding every command the resolver may say"
  - "availableCommand — criterion 6's gate, resolving every emitted command against the live cobra tree"
  - "loadNextActionInput — the read-only loader that feeds the resolver from a real project"
affects: [197-02 card renderer, 197-03 session-start hint, closing-message ratchet]

actuals:
  tokens: 18459
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Pure resolver plus thin impure loader: every decision branch is testable with no filesystem"
    - "Enumerable candidate set walked BEFORE gating, so a rename fails the build instead of being absorbed by the fallback"
    - "Read-only state read that skips the two repair paths which persist"

key-files:
  created:
    - cmd/next_action.go
    - cmd/next_action_input.go
    - cmd/next_action_test.go
    - cmd/next_action_availability_test.go
    - cmd/next_action_neutrality_test.go
  modified: []

key-decisions:
  - "A finished-but-unsigned-off project is told to sign off, not to archive — resolving the drift between the five existing deciders"
  - "The availability gate landed in task 1 rather than task 2, because the resolver routes every emitted command through it and could not be written without it"
  - "The loader does not use loadActiveColonyState: both of its repair paths persist, and an inspection must not write"
  - "A command read out of a saved recovery report is data, not a literal, so it is gated but not required to be in the candidate set"

patterns-established:
  - "Candidate-set-before-gate: prove the set resolves against the live tree before any fallback can mask a rename"
  - "Comment-stripped source scanning: purity assertions parse the file so a comment about the store does not trip the check"
  - "Content-hashed recursive data-directory fingerprint as the read-only assertion"

requirements-completed: [NEXT-01, NEXT-06]

coverage:
  - id: D1
    description: "One pure function, given the project's saved state, returns all eight things the owner needs: where things stand, what changed, open flags and signals, the recommendation, the exact command, two to four alternatives, whether it is safe to close the chat, and any paused or blocked state."
    requirement: NEXT-01
    verification:
      - kind: unit
        ref: "cmd/next_action_test.go#TestResolveNextActionCoversEveryLifecycleState"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestResolveNextActionIsPure"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestNextActionAlternativesAreWellFormed"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestNextActionAnswerFieldsAreAllSerialisable"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestResolverIsFreeOfImpureReferences"
        status: pass
    human_judgment: false
  - id: D2
    description: "The recommended command, and every alternative, resolves against the live command tree. A command that is not registered can never leave the resolver, and a rename fails the build rather than being silently absorbed by the fallback."
    requirement: NEXT-06
    verification:
      - kind: unit
        ref: "cmd/next_action_availability_test.go#TestEveryResolverCandidateResolves"
        status: pass
      - kind: unit
        ref: "cmd/next_action_availability_test.go#TestUnregisteredCommandIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/next_action_availability_test.go#TestRecommendedCommandIsAlwaysAvailable"
        status: pass
      - kind: unit
        ref: "cmd/next_action_availability_test.go#TestNoCommandIsSpelledInlineAtABranch"
        status: pass
      - kind: manual_procedural
        ref: "rename demonstration: cmd/codex_workflow_cmds.go:313 Use: \"continue\" -> \"continue-renamed-for-proof\", TestEveryResolverCandidateResolves FAILED, rename reverted"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every command value the resolver produces is the platform-neutral runtime form; no platform's slash spelling is ever baked in."
    verification:
      - kind: unit
        ref: "cmd/next_action_neutrality_test.go#TestResolverCommandsArePlatformNeutral"
        status: pass
      - kind: unit
        ref: "cmd/next_action_neutrality_test.go#TestCandidateSetIsPlatformNeutral"
        status: pass
    human_judgment: false
  - id: D4
    description: "The resolver can be fed from a real project, and feeding it changes nothing on disk."
    verification:
      - kind: integration
        ref: "cmd/next_action_test.go#TestLoadNextActionInputGathersSavedState"
        status: pass
      - kind: integration
        ref: "cmd/next_action_test.go#TestLoadNextActionInputDoesNotMutate"
        status: pass
      - kind: integration
        ref: "cmd/next_action_test.go#TestLoadNextActionInputDistinguishesNoColony"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestLoaderContainsNoCommandDecision"
        status: pass
    human_judgment: false
  - id: D5
    description: "Safe-to-close is claimable only when the handoff document is verifiably on disk — the existing rule in renderContextClearGuidanceForPlatform, preserved rather than weakened when it moved into the resolver."
    verification:
      - kind: unit
        ref: "cmd/next_action_test.go#TestNextActionContextHealthIsSafeOnlyWithAHandoffOnDisk"
        status: pass
      - kind: unit
        ref: "cmd/next_action_test.go#TestNextActionMidBuildIsNeverSafeToClose"
        status: pass
    human_judgment: false
  - id: D6
    description: "The recommendation prose reads as plain English for someone who has never opened a file in this repository, translating every repo-invented word inline."
    verification: []
    human_judgment: true
    rationale: "Whether a sentence lands for a non-technical reader is a judgment no test asserts. The tests only prove the prose is non-empty and names the phase number."

duration: 48 min
completed: 2026-08-28
status: complete
---

# Phase 197 Plan 01: One Answer to What Next Summary

**A pure eight-field resolver over the saved colony state, plus a live-cobra-tree availability gate whose enumerable candidate set is walked before any fallback can mask a rename.**

## Performance

- **Duration:** 48 min
- **Tasks:** 3
- **Files created:** 5
- **Files modified:** 0

## Accomplishments

- `resolveNextAction` merges the branches that `workflowSuggestionsForState`, `nextCommandFromState`, `nextCommandForHookState`, `nextUpSuggestionsForState` and `closeoutNextCommand` each covered partially, most specific first. None of those five was changed — collapsing them is plan 197-02's job and would have put this plan's file ownership across `codex_visuals.go`.
- Every command the resolver may say is declared exactly once, in `nextActionCandidates`. No command name is spelled inline at a branch, and `TestNoCommandIsSpelledInlineAtABranch` sweeps the whole state table to keep it that way.
- `availableCommand` resolves a candidate's leading verb path against the live cobra tree two ways that must agree: an exact walk by name-or-alias (the anti-fuzzy guard, so a prefix can never pass for a real command) and `rootCmd.Find` (the house pattern from the orphan reachability ratchet). Arguments survive untouched, so `aether build 7` is checked on `build` and keeps its `7`.
- The read-only loader gathers state, open flags, the planning blocker, the live recovery report, handoff presence, signals and outstanding task goals — and provably writes nothing.
- Context health preserves `renderContextClearGuidanceForPlatform`'s rule exactly: safe is claimable only when the handoff file is on disk, and a running build is never safe regardless.

## The rename demonstration

The plan required proof that criterion 6's gate can actually refuse a real rename, because at runtime the gate DROPS a failing candidate and falls back to one that resolves — which means a sweep over emitted commands would silently measure the fallback and stay green.

**What was done:** `cmd/codex_workflow_cmds.go:313` was edited from `Use: "continue"` to `Use: "continue-renamed-for-proof"`, with nothing else touched.

**Result — the two tests behaved exactly as the plan predicted:**

```
=== TestEveryResolverCandidateResolves (must FAIL) ===
--- FAIL: TestEveryResolverCandidateResolves (0.00s)
    next_action_availability_test.go:87: candidate "continue" spells "aether continue",
    which does not resolve to a registered command.
        Either the command was renamed or removed, or this entry was never real.
        Fix the entry -- do not delete this test.
FAIL	github.com/calcosmic/Aether/cmd

=== TestRecommendedCommandIsAlwaysAvailable (the sweep that CANNOT catch it) ===
ok  	github.com/calcosmic/Aether/cmd	0.615s
```

The sweep over emitted commands stayed green on the fallback. Only the walk over the enumerable candidate set, performed BEFORE any gating, turned the rename into a red build. That is the whole reason the plan was revised to require the set.

**Restored:** `git checkout -- cmd/codex_workflow_cmds.go`; the file is byte-identical to its committed state (`git status --short` shows it clean) and all availability tests pass again.

## Task Commits

1. **Task 1 RED: failing tests for the pure resolver** — `809da59f` (test)
2. **Task 1 GREEN: the eight-field answer and the resolver** — `df01d363` (feat)
3. **Task 2: three proofs the availability gate refuses** — `35b96819` (test)
4. **Task 3 RED: failing tests for the read-only loader** — `76444a52` (test)
5. **Task 3 GREEN: the loader** — `50c9ee57` (feat)

Recorded RED output for each TDD task:

- Task 1 RED: `undefined: nextActionInput` (×10 sites), `FAIL github.com/calcosmic/Aether/cmd [build failed]`
- Task 2: `TestNoCommandIsSpelledInlineAtABranch/an_interrupted_build_that_never_started_is_told_to_restart_it: command "aether build 2 --force" (shape "aether build --force") is not drawn from nextActionCandidates` — a real failure in the test's own shape-normalisation, fixed before commit
- Task 3 RED: `undefined: loadNextActionInput` (×6 sites), `FAIL github.com/calcosmic/Aether/cmd [build failed]`

## Files Created

- `cmd/next_action.go` — the eight-field `nextAction`, the `nextActionCandidates` set, the `availableCommand` gate, and the pure `resolveNextAction`
- `cmd/next_action_input.go` — the read-only loader and `readColonyStateWithoutWriting`
- `cmd/next_action_test.go` — the lifecycle table, purity, alternatives, context health, serialisability, and the loader tests
- `cmd/next_action_availability_test.go` — criterion 6's three proofs plus the recovery-command-from-disk and argument-preservation cases
- `cmd/next_action_neutrality_test.go` — S-01 locked across every platform value the runtime recognises

## Decisions Made

**1. A finished-but-unsigned-off project is told to sign off, not to archive.**
The five existing deciders disagreed here. `workflowSuggestionsForState` advised archiving for a `COMPLETED` project; `nextCommandFromState`, `nextCommandForHookState` and `nextUpSuggestionsForState` all advised signing off. Signing off is what sets the final milestone that the archive step looks for, so advising archiving first tells the owner to skip the step that records what was learned. The resolver signs off, and reaches archiving only once that milestone is actually set. This is the drift the phase was opened to end.

**2. A `READY` project whose current phase is already finished starts the next unfinished phase.**
`nextCommandFromState` jumped straight to signing off in this case; `nextCommandForHookState` scanned for the next ready phase. The resolver takes the more specific branch — start the next unfinished phase, and sign off only when there is genuinely none left.

**3. A command read out of a saved recovery report is data, not a literal.**
It cannot live in a static set because it is written by a previous run. It is gated identically to any candidate; when it fails, the resolver falls back to a command that is itself gated and records the substitution in `Notes`, so the owner is never silently redirected. Covered by `TestRecoveryCommandFromDiskIsGated`, both directions.

**4. Context health is an enumeration plus a reason code, never prose.**
`KEEP` / `SAFE` / `CLEAR_RECOMMENDED` with `handoff_not_on_disk`, `build_in_progress`, `at_a_natural_break`, `handoff_saved`. The sentence belongs to plan 197-02's card.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking] The availability gate had to land in task 1, not task 2**

- **Found during:** Task 1
- **Issue:** The plan assigns the gate to task 2, but `resolveNextAction` routes every command it emits through the gate before returning — the resolver's own return path depends on it. Writing task 1 without the gate would have meant writing a resolver that emits ungated commands and then rewriting its return path in task 2.
- **Fix:** `availableCommand`, `gateChoice` and `gateAlternatives` were implemented in task 1's GREEN commit. Task 2 remained the plan's real deliverable: the three proofs, the neutrality lock, and the rename demonstration.
- **Consequence for the RED:** task 2's tests passed on first run rather than failing, so they are proofs over existing code rather than a red-to-green cycle. The plan's own stated purpose for those tests — proving the gate can refuse — is satisfied instead by the rename demonstration above, which was mandatory regardless and which genuinely turned the build red.
- **Files:** `cmd/next_action.go`
- **Commit:** `df01d363`

**2. [Rule 2 — Missing critical] The loader cannot use `loadActiveColonyState`**

- **Found during:** Task 3
- **Issue:** The plan's read list names `loadActiveColonyState`. Reading it revealed that it calls `loadColonyStateWithCompatibilityRepair` (which persists a legacy-numeric-field repair, `cmd/state_load.go:63`) and `repairMissingPlanFromArtifacts` (which persists a recovered plan, `cmd/state_repair.go:53`). Both write to `COLONY_STATE.json`. Using it would have made the loader a writer, directly contradicting the plan's own read-only requirement and CLAUDE.md's corollary about inspection commands that quietly write.
- **Fix:** `readColonyStateWithoutWriting` reads the state itself and applies the legacy-numeric repair **in memory only**, never persisting it. The behaviour an old state file needs is preserved; the write is not.
- **Verification:** `TestLoadNextActionInputDoesNotMutate` fingerprints every file under the project's data directory by modification time, size and SHA-256 before and after two loads. It fails if the loader is made to write.
- **Files:** `cmd/next_action_input.go`
- **Commit:** `50c9ee57`

**3. [Rule 1 — Bug] The purity assertion tripped on its own explanatory comment**

- **Found during:** Task 1
- **Issue:** `TestResolverIsFreeOfImpureReferences` scanned raw source text, so the comment "touches no package store" matched the very pattern it was forbidding. A word-boundary regexp alone would not have fixed it — the comment genuinely contains the word.
- **Fix:** The test now parses `next_action.go` with `go/parser` (comments discarded) and re-prints it, then scans the comment-free code. Word-boundary matching is retained so ordinary prose containing "restore" is not mistaken for a store reference.
- **Files:** `cmd/next_action_test.go`
- **Commit:** `df01d363`

**4. [Rule 1 — Bug] Shape normalisation in `TestNoCommandIsSpelledInlineAtABranch`**

- **Found during:** Task 2
- **Issue:** The test reduced the candidate template by deleting `%d` from the string, producing a double space (`aether build  --force`), while it reduced the emitted command by dropping whole fields (`aether build --force`). The two shapes never matched and the test reported a false positive against `aether build 2 --force`.
- **Fix:** Both sides now go through the same field-wise normaliser, which drops `%d` and any numeric field.
- **Files:** `cmd/next_action_availability_test.go`
- **Commit:** `35b96819`

**5. [Rule 3 — Blocking] `snapshotDataDir` name collision**

- **Found during:** Task 3
- **Issue:** `cmd/boundary_contract_test.go:22` already declares `snapshotDataDir`.
- **Fix:** Renamed the new helper to `snapshotProjectDataTree`. It is deliberately not a reuse of the existing one: the existing helper is shallow (`os.ReadDir`, top level only) and compares size and modification time. The new one walks recursively and adds a content hash, which is what the read-only claim actually needs — a same-size rewrite would slip past the old one.
- **Files:** `cmd/next_action_test.go`
- **Commit:** `76444a52`

---

**Total deviations:** 5 auto-fixed (2 blocking, 2 bugs, 1 missing critical)
**Impact on plan:** No scope creep. Deviation 1 moved work one task earlier without changing what was built; deviation 2 strengthened the read-only guarantee the plan asked for. All five were necessary for correctness.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None. Nothing renders from the resolver yet, which is the plan's deliberate design — a card built over a resolver that could still emit a dead command would be a confidently wrong instruction — not a stub. Plan 197-02 owns the renderers.

## Threat Flags

None. This plan adds no network endpoint, no auth path, no file-access pattern and no schema at a trust boundary. It reads project-local state that the runtime already reads, and writes nothing.

## Verification Run

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(ResolveNextActionCoversEveryLifecycleState\|ResolveNextActionIsPure\|NextActionAlternativesAreWellFormed)' -count=1` | ok — 30 sub-tests across 10 lifecycle branches |
| `go test ./cmd -run 'Test(RecommendedCommandIsAlwaysAvailable\|UnregisteredCommandIsRefused\|EveryResolverCandidateResolves\|ResolverCommandsArePlatformNeutral)' -count=1` | ok |
| `go test ./cmd -run 'Test(LoadNextActionInputGathersSavedState\|LoadNextActionInputDoesNotMutate\|LoadNextActionInputDistinguishesNoColony)' -count=1` | ok |
| `go test ./cmd -run 'Test(OrphanAllowlistOnlyShrinks\|NoRegisteredSubcommandIsUnreferenced\|AuditCatalogGolden\|PlatformParityGolden\|RegressionSnapshot)' -count=1` | ok — all five untouched, as the plan required |
| `go test ./cmd -run 'Test(HumanFacingOutputGoesThroughWriteVisualOutput\|DocumentedSubcommandsAreSeverityClassified\|WrapperCommandNamesMatchCanonicalCorpus\|WiringGuardsHaveNoRuntimeEscapeHatch)' -count=1` | ok |
| `go build ./...` | clean |
| `go build ./cmd/aether` | clean |
| `go vet ./cmd/` | clean |
| `gofmt -l cmd/ pkg/` | no output |

No CLI flag or subcommand was added or removed, so `cmd/testdata/command_catalog.json` needed no refresh — confirmed by the catalog and parity goldens passing untouched.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Plan 197-02 can render from `resolveNextAction` directly. Three things it inherits:

- `nextAction` is fully json-tagged, so the whole struct marshals into the machine-readable envelope with no further work.
- Every command value is platform-neutral runtime form. The card renderer applies `translateHintCommandsForPlatform` / `platformCommandName` at the visual writer; JSON envelopes must stay raw.
- `ContextHealth` is an enum plus reason code. The owner-facing sentence is 197-02's to write.

One thing 197-02 must be careful about: the five existing deciders are all still live and still called from ~200 sites. Collapsing them is 197-02's job, and it owns `codex_visuals.go` for that reason.

## Self-Check: PASSED

**Files claimed as created — all present on disk:**

| File | Present |
|---|---|
| `cmd/next_action.go` | yes |
| `cmd/next_action_input.go` | yes |
| `cmd/next_action_test.go` | yes |
| `cmd/next_action_availability_test.go` | yes |
| `cmd/next_action_neutrality_test.go` | yes |
| `.planning/phases/197-one-answer-to-what-next/197-01-SUMMARY.md` | yes |

**Commits claimed — all present in `git log`:** `809da59f`, `df01d363`, `35b96819`, `76444a52`, `50c9ee57`.

**Rename demonstration reverted:** `git status --short` is empty and `cmd/codex_workflow_cmds.go:313` reads `Use:   "continue",`. No diff against HEAD for that file.

**STATE.md and ROADMAP.md:** not modified (parallel-executor mode; the orchestrator owns them).

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-28*
