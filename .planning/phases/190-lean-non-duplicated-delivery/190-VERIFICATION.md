---
phase: 190-lean-non-duplicated-delivery
verified: 2026-08-20T23:50:20Z
status: gaps_found
score: 6/7 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 6/7
  gaps_closed:
    - "D-190-03-A: the native/direct build dispatch path (executeCodexBuildDispatches) and 6 sibling native dispatch functions (continue review, continue watcher, plan, colonize, seal, swarm) independently double-delivered active pheromone signals via ContextCapsule + a separate PheromoneSection. Closed by 190-05. Revert-tested in this pass: re-adding `PheromoneSection: resolvePheromoneSection()` to executeCodexBuildDispatches's WorkerDispatch literal made TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule and TestEightCommandsDeliverPheromoneExactlyOnce/build_native fail with the exact pre-fix symptom (\"assembled prompt has 2 Pheromone Signals headings\"); restoring the file returned both to green."
    - "D-190-05-A: continue's native review/watcher dispatch paths (plus colonize, plan, seal, swarm) independently double-delivered prior-worker handoffs via ContextCapsule + a separate HandoffSection. Closed by 190-06 (distinct heading, not deletion, since the two channels carry materially different workflow-scoped content). Revert-tested in this pass: reverting plannedContinueReviewDispatches/plannedContinueWatcherDispatch to call renderWorkerHandoffSection instead of renderRelatedWorkflowHandoffSection made TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading, TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading, and TestNineCommandsDeliverHandoffExactlyOnce/{continue_review,continue_watcher} fail with the exact pre-fix symptom (\"2 Previous Worker Handoffs headings\"); restoring the file returned all to green."
  gaps_remaining:
    - "The phase's own unqualified goal sentence ('No context section delivered twice') is still false in the codebase -- NOT for the previously-named native build-path reason (that instance is now closed and mutation-confirmed), but for a newly discovered instance on a different route this verification pass found: continue's classic-ceremony/heavy-review WRAPPER flow (codexContinuePlanManifest) delivers an active pheromone signal to every reviewer/watcher prompt twice, via ContextCapsule + a separate manifest-level PheromoneSection field the wrapper prose also concatenates. See gaps below."
  regressions: []
gaps:
  - truth: "No context section is delivered twice on ANY dispatch path this repo spawns a worker from -- native or wrapper-mediated, for any command"
    status: failed
    reason: >
      NEWLY DISCOVERED by this verification pass -- not logged in deferred-items.md as D-190-03-A
      or D-190-05-A, and not part of either prior gap. continue's plan-only/classic-ceremony
      wrapper manifest (codexContinuePlanManifest, cmd/codex_continue_plan.go:149-167) sets BOTH
      ContextCapsule: resolveCodexWorkerContext() (cmd/codex_continue_plan.go:163 -- unconditionally
      renders "## Pheromone Signals" whenever a signal is active, the same shared capsule renderer
      190-05 fixed at 7 other call sites) AND PheromoneSection: resolvePheromoneSection()
      (cmd/codex_continue_plan.go:164 -- independently renders the SAME active signal's text under
      its own "### Active Pheromone Signals" heading) -- both fields manifest-level, resolved once.
      Both .claude/commands/ant/continue.md and .opencode/commands/ant/continue.md (byte-identical,
      md5 031949b27c91ed343ba0ea778dc60587) instruct the wrapper, at lines 134, 170 and 178: "Each
      reviewer's prompt = `continue_manifest.context_capsule` + `continue_manifest.pheromone_section`
      ... prepended VERBATIM ahead of each dispatch's own runtime-provided `brief`." This is the
      exact D-190-01-A shape (build's wrapper flow, fixed by 190-03) recurring on continue's wrapper
      flow, unfixed. It is not a rare edge case: this path triggers on every `aether continue
      --classic-ceremony` or `--verification-depth heavy` invocation, and CLAUDE.md's own
      Queen-Owned Orchestration section documents heavy depth as the ordinary choice for
      production/final/security-named phases, not an exception.

      DEMONSTRATED, not inferred: a throwaway probe (seedActiveSignal + runCodexContinuePlanOnly,
      deleted after use, tree confirmed clean via `git status --porcelain` before and after) showed
      the seeded marker text present once in `plan.ContextCapsule` (under "## Pheromone Signals")
      and once in `plan.PheromoneSection` (under "### Active Pheromone Signals") -- 2 occurrences
      total in the wrapper-concatenated string, simulating exactly what .claude/commands/ant/continue.md
      lines 170/178 instruct the wrapper to assemble.

      This field WAS examined during 190-05's own per-caller audit (190-05-SUMMARY.md, "Also
      audited and found inert" section, referencing cmd/codex_continue_plan.go's
      codexContinuePlanManifest.PheromoneSection field) but incorrectly cleared: the audit checked
      only whether PheromoneSection duplicated PER-DISPATCH (it does not -- codexContinueExternalDispatch
      has no such field, and TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection's own
      reflection check confirms this), never whether the two CO-RESIDENT manifest-level fields
      (ContextCapsule and PheromoneSection) duplicate against EACH OTHER when the wrapper
      concatenates both -- which is the actual D-190-01-A-shaped bug. Build's equivalent wrapper
      prose (.claude/commands/ant/build.md:259) already reads "`context_capsule` ... is the SOLE
      source of pheromone signals and prior worker handoffs" with zero `pheromone_section`
      references anywhere in that file (confirmed by grep) -- continue's prose was never brought in
      line with that same 190-03 fix.
    artifacts:
      - path: "cmd/codex_continue_plan.go"
        issue: "codexContinuePlanManifest sets both ContextCapsule (line 163) and PheromoneSection (line 164) at manifest-construction time inside runCodexContinuePlanOnly; both render the same active-signal text under two different headings, and nothing dedups them before the manifest is returned to the wrapper."
      - path: ".claude/commands/ant/continue.md"
        issue: "Lines 134, 170, 178 instruct the wrapper to prepend continue_manifest.context_capsule AND continue_manifest.pheromone_section, both verbatim, ahead of every classic-ceremony/heavy-review dispatch's brief."
      - path: ".opencode/commands/ant/continue.md"
        issue: "Byte-identical to the Claude Code file (md5 031949b27c91ed343ba0ea778dc60587) -- carries the identical instruction, so both primary platforms are affected equally."
    missing:
      - "Stop setting PheromoneSection on codexContinuePlanManifest (mirroring 190-05's fix at the 7 native call sites -- ContextCapsule is already this flow's sole channel for pheromones, matching what 190-03 already established for build's own wrapper manifest)."
      - "Remove the pheromone_section references from .claude/commands/ant/continue.md and .opencode/commands/ant/continue.md (lines 134, 170, 178), mirroring build.md:259's 'context_capsule ... is the SOLE source of pheromone signals' language."
      - "A permanent test mirroring TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule's shape, but for codexContinuePlanManifest: seed an active signal, call runCodexContinuePlanOnly, assert the signal's own text appears in the union of ContextCapsule+PheromoneSection exactly once, not twice. The existing TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection (Phase 189) only proves both fields are independently non-empty -- that assertion gap is exactly what let this ship undetected through Phase 189 and all of Phase 190's plans to date."
---

# Phase 190: Lean, Non-Duplicated Delivery — Verification Report

**Phase Goal:** No context section delivered twice; briefs stop transiting the orchestrator byte-for-byte.
**Verified:** 2026-08-20T23:50:20Z
**Status:** gaps_found
**Re-verification:** Yes — third pass, after 190-05 and 190-06 gap-closure plans

## Summary For The Developer

Two real bugs were closed since the last pass, and both closures are confirmed by breaking them on
purpose and watching the right test fail, not by trusting the summaries. But this pass found a
**third, previously-undetected instance of the identical bug shape**, on a route none of the six
plans in this phase ever touched: `aether continue --classic-ceremony` / `--verification-depth
heavy` still delivers an active pheromone signal to every reviewer's prompt twice. The score stays
6/7 — not because nothing improved, but because the one truth that was failing before is still
failing now, for a different, smaller, well-understood reason.

## Goal Achievement

Every claim below marked **DEMONSTRATED** was proven by running or mutating code in this session
(Go test execution, revert-and-restore mutation tests, live throwaway probes with `git status
--porcelain` confirmed empty before and after) — not by reading SUMMARY.md prose. The working tree
was restored to byte-identical HEAD (`270dad6c`) after every mutation and probe; final confirmation
recorded below.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Plan-only manifests carry `brief_path` to files on disk; the wrapper passes paths; build.md prose describes `brief_path` as routine | ✓ VERIFIED | DEMONSTRATED (re-run this pass). `TestBuildPlanOnlyManifestOmitsInlineBriefWhenBriefPathPresent` passed, logging the same measurement as the prior pass: "7 dispatches, 6848 bytes would have shipped inline... 364 bytes actually ship... 94.7% reduction." `TestDispatchEntryCarriesBriefPath` passed. `git diff --stat` from the previous verification's HEAD (`1430acf4`) to current HEAD confirms zero files touched under `.claude/commands/`, `.opencode/commands/`, `.aether/commands/`, or `cmd/command_guide.go` — this criterion's surface is untouched by 190-05/190-06. |
| 2 | SC2 — `--print-brief` asserts zero duplicated sections; pheromones and handoffs get one home each **within the scope `--print-brief` inspects** (the wrapper plan-only build flow) | ✓ VERIFIED | DEMONSTRATED (re-run this pass). `TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs`, `TestPrintBriefFailsOnDuplicatedSection`, `TestPlanOnlyDispatchesCarryNoHandoffSection`, `TestNativeDispatchHandoffStaysExactlyOnceViaCapsule` all passed. `cmd/build_review_190_findings_test.go` (the file carrying these locks) is byte-for-byte unmodified since the previous pass (confirmed via `git diff --stat`, 0 changes) — 190-05/190-06 did not touch `--print-brief` or its detectors at all. |
| 3 | SC3 — TS-host hive double-injection removed (build path and continue dry-run `hive_section`) | ✓ VERIFIED | DEMONSTRATED (re-run this pass). `TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly` passed. `git diff --stat` confirms zero `.aether/ts-host/` files touched by 190-05/190-06. `grep -rn "HiveSection\|hive_section" cmd/*.go pkg/codex/*.go` (non-test) still returns zero hits — no new hive channel was introduced by the pheromone/handoff fixes. |
| 4 | SC4 — Orchestrator-relay byte count measurably drops | ✓ VERIFIED | DEMONSTRATED (same test run as truth 1, same 94.7%-reduction figure, computed live by the test itself, not a hardcoded string). `codexBuildManifest.WorkerBriefs` unaffected — 190-05/190-06 never touched `cmd/codex_build.go`'s manifest-serialization code, only `executeCodexBuildDispatches`'s native `codex.WorkerDispatch` construction (a different function in the same file; confirmed by `git diff cmd/codex_build.go` showing only the `PheromoneSection`-removal hunk plus its doc comment, +34/-? lines total for the whole file across both plans). |
| 5 | **The phase's own goal sentence, unqualified: "No context section delivered twice," on ANY dispatch path this repo spawns a worker from** — native or wrapper-mediated, for any command | ✗ FAILED | The previously-failing instance (native build dispatch, D-190-03-A) is CLOSED — DEMONSTRATED via revert test (see Re-verification Detail below). A second instance found by the same 190-05 audit (continue's native handoff duplication, D-190-05-A) is also CLOSED — DEMONSTRATED via revert test. But this verification pass found a **third, new, previously-undetected instance**: continue's plan-only/classic-ceremony WRAPPER flow still delivers an active pheromone signal twice. See `gaps` in frontmatter for full detail and DEMONSTRATED probe evidence. |
| 6 | Convergence risk (two parallel 190-04 fixes reconciled) left no franken-state, and this remains true after two more merged worktree plans | ✓ VERIFIED | Re-checked this pass: `go build ./...` clean, `go vet ./cmd/...` clean. No duplicate function definitions across any 190-05/190-06-touched file (`grep -c "^func <name>"` returns exactly 1 for `executeCodexBuildDispatches`, `renderWorkerHandoffSection`, `renderRelatedWorkflowHandoffSection`, `resolvePheromoneSection`). Both plans' own SUMMARYs document their `chore: merge executor worktree` commits landed cleanly (`git log` confirms `8a598648`, `95007482`), and `git status --porcelain` is empty at HEAD. |
| 7 | D-190-R-A (hive cross-domain coverage narrowing, WR-03) is an honestly-recorded, defensible scope call, not a phase-190-goal violation | ✓ VERIFIED | Unaffected by 190-05/190-06 (neither plan touches hive wisdom). Re-confirmed this pass: still recorded in `deferred-items.md` as an explicit owner/product decision ("whether hard exclusion is the right final behaviour is a product decision... The record is now honest either way"), not a duplication defect. No later phase (191 "Dead Wood" is config/dead-code deletion; 192 "Final Showdown" is a benchmark gate) claims this — it stays an open, named, non-blocking product question, consistent with the previous pass's treatment. |

**Score:** 6/7 truths verified

### Re-verification Detail: What Was Actually Broken And Fixed (Not Just Claimed)

**D-190-03-A (native pheromone duplication) — revert-tested this pass:**
Re-added `PheromoneSection: resolvePheromoneSection(),` to `executeCodexBuildDispatches`'s
`codex.WorkerDispatch{}` literal (`cmd/codex_build.go`, immediately after `ContextCapsule: capsule,`).
Result: `TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule` failed with `"build native: assembled
prompt has 2 \"Pheromone Signals\" headings, want exactly 1"`; `TestEightCommandsDeliverPheromoneExactlyOnce/build_native`
failed identically; the other 8 breadth sub-tests (continue×2, plan, colonize, seal, swarm, quick,
oracle) correctly stayed green, since only the `build_native` case was mutated. File restored from
backup, `git diff --stat` confirmed zero diff, both tests re-run and passed clean.

**D-190-05-A (continue native handoff duplication) — revert-tested this pass:**
Reverted both `plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch`
(`cmd/codex_continue.go`) from `renderRelatedWorkflowHandoffSection("continue", ...)` back to
`renderWorkerHandoffSection("continue", ...)` (the pre-190-06 call). Result:
`TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading` and
`TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading` both failed with `"assembled prompt has 2
\"## Previous Worker Handoffs\" headings, want exactly 1"`; `TestNineCommandsDeliverHandoffExactlyOnce/{continue_review,continue_watcher}`
failed identically; the other 7 breadth sub-tests stayed green. File restored from backup, `git diff
--stat` confirmed zero diff, all tests re-run and passed clean.

**Conclusion:** both fixes are real, load-bearing, and correctly scoped — not decorative tests that
would pass regardless of the source.

### Adversarial Scrutiny: 190-06's "Distinct Heading, Not Deletion" Decision

The task asked whether giving the dedicated handoff channel its own heading (instead of deleting it,
as 190-05 did for pheromones) could let the SAME text reach a prompt twice under two different
headings — satisfying a heading-count test while still violating the goal's spirit.

**Probed empirically, not just reasoned about.** A throwaway probe seeded a "build"-workflow handoff
record and a "continue"-workflow handoff record with the SAME literal `NextWorkerInstructions` text,
then called `plannedContinueReviewDispatches` and assembled the real prompt. Result: the identical
text appeared **twice** — once under "## Previous Worker Handoffs" (capsule, build-workflow), once
under "## Related Worker Handoffs" (dedicated field, continue-workflow). This confirms the scenario
is mechanically reachable, not merely theoretical.

**Judgment: this is not a phase-goal violation**, for four reasons:

1. **Different bug class.** Every duplicate this phase has fixed (D-190-01-A, D-190-03-A, D-190-05-A)
   was the SAME underlying data, computed by two independent code paths, delivered twice. This
   scenario requires two DIFFERENT, independently-authored handoff records (different IDs, different
   workflow tags, different worker names) that happen to share text — a coincidence of content, not a
   redundant computation. Structurally, `record.Workflow` is a single field set once at persist time
   from the producing dispatch's own `Workflow` value (`cmd/codex_dispatch_contract.go:867`); a build-tagged
   record and a continue-tagged record can never be the same record.
2. **Consistent with this codebase's own established convention.** `--print-brief`'s own duplication
   detector (`duplicatedBriefSections`, since 190-01) has always counted HEADINGS, not content hashes.
   190-06's own source comment (`cmd/codex_dispatch_contract.go`, on `renderRelatedWorkflowHandoffSection`)
   states this explicitly: "this repo's own convention treats a repeated HEADING as 'the same section
   delivered twice' regardless of whether the body content is identical." This is a documented,
   consistent choice, not a newly-discovered loophole.
3. **The distinct headings carry real information.** They tell the reading worker WHERE each note came
   from (cross-phase build carryover vs. same-workflow sibling relay) — collapsing them to avoid a
   contrived content collision would destroy that distinction for every normal case where the content
   legitimately differs (the overwhelming majority).
4. **A content-hash alternative would be strictly worse.** Suppressing "duplicate-looking" text across
   channels risks dropping genuinely distinct information that merely shares some words (e.g. two
   workers both correctly reporting the same changed file).

This is examined and resolved, not left open — flagged here for transparency since it was explicitly
in scope for this pass's scrutiny, not because it changes the verdict.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/codex_build.go` (`executeCodexBuildDispatches`) | No `PheromoneSection` set; capsule is sole channel | ✓ VERIFIED | Confirmed by reading + revert test (see above). |
| `cmd/build_pheromone_190_05_test.go` | 9-case breadth lock for pheromone once-ness | ✓ VERIFIED | `TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule` + `TestEightCommandsDeliverPheromoneExactlyOnce` (9 sub-tests) all pass; mutation-confirmed to catch the reintroduced defect. |
| `cmd/codex_dispatch_contract.go` (`renderRelatedWorkflowHandoffSection`, `renderHandoffSectionNamed`) | Distinct-heading relay for materially-different handoff content | ✓ VERIFIED | Exists, wired at 6 call sites (continue review, continue watcher, colonize, plan, seal, swarm); mutation-confirmed via revert test. |
| `cmd/build_handoff_190_06_test.go` | 9-case breadth lock for handoff once-ness (both channels) | ✓ VERIFIED | `TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading`, `TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading`, `TestNineCommandsDeliverHandoffExactlyOnce` (9 sub-tests) all pass; mutation-confirmed. |
| `deferred-items.md` | D-190-03-A and D-190-05-A marked RESOLVED with pointers to closing SUMMARYs | ✓ VERIFIED | Both entries read in full; resolution text matches the actual code (cross-checked, not just trusted). |
| `cmd/codex_continue_plan.go` (`codexContinuePlanManifest`) | Should not independently duplicate pheromones against its own `ContextCapsule` | ✗ **NOT VERIFIED — NEW GAP** | Both `ContextCapsule` and `PheromoneSection` are set (lines 163-164) and both are consumed by wrapper prose. See gaps in frontmatter. |
| `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md` | Should read like build.md's fixed prose ("capsule is the SOLE source of pheromone signals") | ✗ **NOT VERIFIED — NEW GAP** | Still instructs concatenation of `context_capsule` + `pheromone_section` (lines 134, 170, 178 in both, byte-identical). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `executeCodexBuildDispatches` | `codex.WorkerDispatch{ContextCapsule}` only | `PheromoneSection` no longer set | WIRED, FIXED | Revert-tested. |
| `plannedContinueReviewDispatches` / `plannedContinueWatcherDispatch` | `renderRelatedWorkflowHandoffSection` | distinct `"## Related Worker Handoffs"` heading | WIRED, FIXED | Revert-tested. |
| `dispatchRealSurveyorsWithTimeout` / `dispatchRealPlanningWorkersWithIterationContext` / `plannedSealFinalReviewDispatches` / `invokeSwarmWorker` | `renderRelatedWorkflowHandoffSection` | same mechanism, 4 more commands | WIRED, FIXED | Confirmed via `TestNineCommandsDeliverHandoffExactlyOnce`'s per-command sub-tests, all passing. |
| `runCodexContinuePlanOnly` | `codexContinuePlanManifest{ContextCapsule, PheromoneSection}` | both fields set unconditionally when a signal is active | **WIRED, BUT DUPLICATING** | **NEW GAP.** Confirmed by direct trace (`cmd/codex_continue_plan.go:163-164`) and live probe (marker text present once in each field). |
| `.claude/commands/ant/continue.md` (classic-ceremony reviewer prompt assembly) | `continue_manifest.context_capsule` + `continue_manifest.pheromone_section` | both concatenated verbatim | **INSTRUCTED TO DUPLICATE** | **NEW GAP.** Lines 170, 178; byte-identical in `.opencode/commands/ant/continue.md`. |
| `pkg/codex/prompt.go` (`AssemblePrompt`/`AssembleHostedPrompt`) | 5 independent parts (context/handoff/skill/pheromone/brief) | unchanged since 190-01 | WIRED, UNCHANGED | Re-read this pass; confirms the fix strategy throughout is "stop setting the redundant field at the caller," never "dedup in the assembler" — consistent across all three closed instances and applicable to the new one too. |
| `pkg/codex/dispatch.go` (`WorkerConfig` construction) | Forwards `ContextCapsule`/`PheromoneSection`/`HandoffSection` unfiltered | lines 175, 180-181 | WIRED, UNCHANGED | Re-confirmed this pass — the duplication-prevention burden sits entirely at the caller layer, by design; this is exactly why the new gap sits at `codex_continue_plan.go`, not in this shared forwarding code. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `TestEightCommandsDeliverPheromoneExactlyOnce` / `TestNineCommandsDeliverHandoffExactlyOnce` | captured `codex.WorkerConfig`/`codex.WorkerDispatch` | Spy `WorkerInvoker` capturing the REAL value each production dispatch function hands to the invoker | Yes | ✓ FLOWING — not a parallel computation; the spy proves the actual wiring. |
| `codexContinuePlanManifest.ContextCapsule` / `.PheromoneSection` | seeded pheromone signal | `pheromones.json` via `store.SaveJSON`, read back through `resolveCodexWorkerContext()`/`resolvePheromoneSection()` in production code | Yes (and duplicated) | ⚠ FLOWING TWICE — this is the new gap; both variables independently source from the same underlying signal. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Pheromone breadth lock (9 cases) | `go test ./cmd/ -run TestEightCommandsDeliverPheromoneExactlyOnce -v -count=1` | 9/9 PASS | ✓ PASS |
| Handoff breadth lock (9 cases) | `go test ./cmd/ -run TestNineCommandsDeliverHandoffExactlyOnce -v -count=1` | 9/9 PASS | ✓ PASS |
| Criteria 1/2/3/4 targeted regression (8 named tests) | `go test ./cmd/ -run '<8 test names>' -v -count=1` | 8/8 PASS | ✓ PASS |
| `go vet ./cmd/...` | `go vet ./cmd/...` | exit 0, no output | ✓ PASS |
| `go build ./...` | `go build ./...` | exit 0, no output | ✓ PASS |
| D-190-03-A mutation (re-add `PheromoneSection` to native build dispatch) | revert, `go test -run TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule\|TestEightCommandsDeliverPheromoneExactlyOnce` | FAIL as expected (2/9 sub-cases), then reverted, then PASS | ✓ PASS (fix is real) |
| D-190-05-A mutation (revert continue handoff calls to plain `renderWorkerHandoffSection`) | revert, `go test -run TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading\|TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading\|TestNineCommandsDeliverHandoffExactlyOnce` | FAIL as expected (4 failures), then reverted, then PASS | ✓ PASS (fix is real) |
| NEW GAP probe: continue plan-only manifest cross-field pheromone duplication | throwaway probe: seed active signal, `runCodexContinuePlanOnly`, count marker in `ContextCapsule` + `PheromoneSection` | marker present 1× in each field, 2× in the wrapper-concatenated string | ✗ FAIL — confirms the gap |
| Adversarial probe: 190-06 same-text-under-two-headings scenario | throwaway probe: seed identical text in a build-tagged AND a continue-tagged handoff, assemble via `plannedContinueReviewDispatches` | text present 2×, once under each of two distinct headings | Reachable but judged NOT a goal violation (see Adversarial Scrutiny above) |
| Working tree clean after all probes and mutations | `git status --porcelain` | empty | ✓ PASS |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` files exist in this repository and none is named in this
phase's PLAN/SUMMARY/REVIEW files. This is a Go/TS phase verified via `go test` directly plus live
mutation/probe testing in this session (see Behavioral Spot-Checks above), not a probe-script-based
migration/tooling phase.

### Requirements Coverage

N/A. No `requirements:` IDs are declared in any Phase 190 plan's frontmatter, and `grep -n "190"
.planning/REQUIREMENTS.md` returns zero matches — no REQ-IDs map to Phase 190. No orphaned
requirements. Unchanged from the previous pass.

### Anti-Patterns Found

None in the files 190-05/190-06 modified. Scanned `cmd/codex_build.go`, `cmd/codex_build_finalize.go`,
`cmd/codex_colonize.go`, `cmd/codex_continue.go`, `cmd/codex_dispatch_contract.go`, `cmd/codex_plan.go`,
`cmd/seal_final_review.go`, `cmd/swarm_cmd.go`, `cmd/build_pheromone_190_05_test.go`,
`cmd/build_handoff_190_06_test.go` for `TBD`/`FIXME`/`XXX`, `TODO`/`HACK`/`PLACEHOLDER`,
"not yet implemented"/"placeholder"/"coming soon", and empty-return stub patterns. The only hits were
pre-existing, unrelated `codex_colonize.go` lines that implement the surveyor's OWN tech-debt scanner
(literal strings `"TODO"`/`"FIXME"` used as detection patterns for codebases Aether analyzes, not
debt markers in Aether's own code) and pre-existing `"placeholder"` occurrences inside
`ResultCollectionPolicy` documentation strings describing conflict-resolution semantics — neither is
new, neither is a stub.

### Human Verification Required

None. This remains a backend CLI/wire-format phase (no UI, no visual rendering, no real-time
behavior, no external service integration) — every claim in this report was verifiable, and was
verified, by direct execution.

### Gaps Summary

**What is genuinely fixed, confirmed by breaking it and watching it break:** the native/direct build
dispatch path no longer double-delivers pheromone signals (7 commands' worth, closed by 190-05), and
continue's native review/watcher dispatch paths — plus four more commands the same audit
discipline surfaced — no longer double-deliver prior-worker handoffs (closed by 190-06). Both fixes
were verified in this session by temporarily reverting the source and watching the exact named
regression test fail with the exact pre-fix symptom text, then restoring and watching it pass again.
The working tree is clean at HEAD (`270dad6c`) with no leftover mutations or probe files.

**What is not yet true:** the phase's own literal goal — "No context section delivered twice" — still
does not hold everywhere. This pass found a route none of the six 190-0X plans exercised:
`aether continue --classic-ceremony` (equivalently, `--verification-depth heavy`) still delivers an
active pheromone signal to every spawned reviewer and watcher's prompt twice, via
`codexContinuePlanManifest`'s `ContextCapsule` and `PheromoneSection` fields, both of which the
wrapper prose (identical across Claude Code and OpenCode) explicitly concatenates. This is the exact
shape of bug 190-03 already fixed for build's equivalent wrapper flow — build.md was updated to make
the capsule the sole channel; continue.md never was. It was looked at during 190-05's audit and
cleared on an incomplete check (per-dispatch duplication only), not fixed. The fix is small,
well-precedented (this phase has now applied the identical "stop setting the redundant field" pattern
three times), and does not require new architecture — but it is not done as of this HEAD.

This is a genuine BLOCKER for the phase's own stated goal, not a scope question: it is squarely a
duplication defect (the same class this entire phase exists to close), on a route this phase's own
scope statement covers ("No context section delivered twice" — unqualified by command), triggered by
ordinary use (heavy-depth continue runs), and demonstrated live in this session, not inferred.

---

_Verified: 2026-08-20T23:50:20Z_
_Verifier: Claude (gsd-verifier)_
