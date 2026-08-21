---
phase: 190-lean-non-duplicated-delivery
verified: 2026-08-21T00:00:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 6/7
  gaps_closed:
    - "The phase's own unqualified goal sentence ('No context section delivered twice') was still false because continue's classic-ceremony/heavy-review WRAPPER flow (codexContinuePlanManifest, cmd/codex_continue_plan.go) set both ContextCapsule and a separate PheromoneSection, and .claude/commands/ant/continue.md + .opencode/commands/ant/continue.md instructed concatenating both — double-delivering every active pheromone signal to every reviewer/watcher spawned by `aether continue --classic-ceremony` / `--verification-depth heavy`. Closed by commit b6671584 ('fix(190-07): capsule is the sole pheromone carrier on continue's wrapper flow'): PheromoneSection is no longer set on codexContinuePlanManifest (field kept, omitempty, for wire-compat only); all three wrapper prose copies (.claude/commands/ant/continue.md, .opencode/commands/ant/continue.md, and the flat installed mirror .claude/commands/ant-continue.md) now state the capsule is the SOLE source of pheromone signals and no longer instruct prepending pheromone_section. Revert-tested in THIS pass (not just re-read): temporarily restoring `PheromoneSection: resolvePheromoneSection(),` to codexContinuePlanManifest made TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection fail with 'seeded signal text must reach the manifest exactly once across capsule+pheromone_section, got 2'; restoring the file returned it to green. Separately, temporarily re-adding a forbidden concatenation line ('+ `continue_manifest.pheromone_section`') to .claude/commands/ant/continue.md made TestContinueWrapperInstructsCapsuleAndPheromoneDelivery fail with the exact forbidden-pattern message; restoring the file returned it to green. A new end-to-end throwaway probe (TestZZProbe190Pass4WrapperAssemblyDeliversSignalExactlyOnce, deleted after use) additionally confirmed, for a real HeavyFlag:true run producing 4 dispatches (1 watcher + 3 reviewers), that the FULL wrapper-assembled prompt (capsule + each dispatch's own brief + skill_section, exactly the formula continue.md now documents) carries the seeded marker exactly once per dispatch, zero times via PheromoneSection."
  gaps_remaining: []
  regressions: []
---

# Phase 190: Lean, Non-Duplicated Delivery — Verification Report

**Phase Goal:** No context section delivered twice; briefs stop transiting the orchestrator byte-for-byte.
**Verified:** 2026-08-21T00:00:00Z
**Status:** passed
**Re-verification:** Yes — fourth pass, after commit `b6671584` closed the third pass's one remaining gap (continue wrapper pheromone duplication)

## Summary For The Developer

The gap the third pass found — continue's `--classic-ceremony` / `--verification-depth heavy`
wrapper flow delivering an active pheromone signal to every reviewer and watcher twice — is fixed,
and this pass proved it by breaking the fix on purpose (twice, for both the runtime field and the
wrapper prose) and watching the exact pre-fix symptom come back, then restoring and watching it go
green again. This pass also ran a fresh end-to-end probe simulating the wrapper's own documented
assembly formula across a real heavy-depth run (4 spawned workers) rather than trusting the existing
unit test alone, and did a full-repo sweep for any other place the same bug shape could be hiding.
None was found. All 7 of this phase's must-haves are now verified. Working tree is clean at HEAD
(`b6671584`) — no leftover mutations or probe files.

## Goal Achievement

Every claim below marked **DEMONSTRATED** was proven by running or mutating code in this session
(Go test execution, revert-and-restore mutation tests against both the runtime field and the wrapper
markdown, a fresh live throwaway probe) — not by reading SUMMARY.md prose or trusting the third
pass's own record without re-running it. Claims marked **RE-CONFIRMED (regression check)** were
proven with fresh test execution this pass but were not re-derived from first principles, per the
re-verification methodology (previously-passed items get a quick regression check, not the full
Inversion + Confirmation Bias Counter treatment). Claims marked **READ-VERIFIED** are structural/
negative claims (e.g. "no other file does X") established by direct reading and grep of the actual
source, not by execution — execution cannot prove a universally-quantified absence, only exhaustive
enumeration can. The working tree was restored to byte-identical HEAD (`b6671584`) after every
mutation and probe; confirmed via `git status --porcelain` (empty) and `git diff --stat` (empty)
immediately before this report was written.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Plan-only manifests carry `brief_path` to files on disk; the wrapper passes paths; build.md prose describes `brief_path` as routine | ✓ VERIFIED | RE-CONFIRMED (regression check). `TestBuildPlanOnlyManifestOmitsInlineBriefWhenBriefPathPresent` and `TestDispatchEntryCarriesBriefPath` re-run this pass, both PASS (same 94.7%-reduction measurement, computed live by the test). `git diff --stat 270dad6c..b6671584` confirms zero files touched under `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.aether/commands/`, or `cmd/command_guide.go` since the third pass — this criterion's surface is entirely untouched by the 190-07 fix. |
| 2 | SC2 — `--print-brief` asserts zero duplicated sections; pheromones and handoffs get one home each within the scope `--print-brief` inspects (the wrapper plan-only build flow) | ✓ VERIFIED | RE-CONFIRMED (regression check). `TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs`, `TestPrintBriefFailsOnDuplicatedSection` (3 sub-cases), `TestPlanOnlyDispatchesCarryNoHandoffSection`, `TestNativeDispatchHandoffStaysExactlyOnceViaCapsule` all re-run this pass, all PASS. `cmd/build_review_190_findings_test.go` is not in the `270dad6c..b6671584` diff — untouched by the 190-07 fix. |
| 3 | SC3 — TS-host hive double-injection removed (build path and continue dry-run `hive_section`) | ✓ VERIFIED | RE-CONFIRMED (regression check). `TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly` re-run this pass, PASS. `grep -rn "HiveSection\|hive_section" cmd/*.go pkg/codex/*.go` (non-test) re-run this pass, still zero hits. |
| 4 | SC4 — Orchestrator-relay byte count measurably drops | ✓ VERIFIED | RE-CONFIRMED (same test run as truth 1, same live-computed reduction figure). `git diff --stat 270dad6c..b6671584` confirms `cmd/codex_build.go` is not in this pass's diff at all — the 190-07 fix touched only `cmd/codex_continue_plan.go` (continue's plan-only manifest), never build's manifest-serialization code. |
| 5 | **The phase's own goal sentence, unqualified: "No context section delivered twice," on ANY dispatch path this repo spawns a worker from** — native or wrapper-mediated, for any command | ✓ VERIFIED | DEMONSTRATED. See "Re-verification Detail" and "Full-Surface Sweep" below — this is the truth the third pass failed and this pass closes. |
| 6 | Convergence risk (parallel fixes reconciled) left no franken-state, and this remains true after the 190-07 fix | ✓ VERIFIED | RE-CONFIRMED this pass: `go build ./...` clean (exit 0), `go vet ./cmd/...` clean (exit 0, no output). `grep -n "^func resolveCodexWorkerContext("` returns exactly 1 definition (`cmd/colony_prime_context.go:1126`); `runCodexContinuePlanOnly`, `codexContinuePlanManifest`, `resolvePheromoneSection` each have exactly 1 definition. `git status --porcelain` empty at HEAD. |
| 7 | D-190-R-A (hive cross-domain coverage narrowing, WR-03 — also referenced as "D-190-04-A" in 190-04-SUMMARY.md, same finding under an earlier label) is an honestly-recorded, defensible scope call, not a phase-190-goal violation | ✓ VERIFIED | READ-VERIFIED this pass (see Probe 5 detail below). Re-read `deferred-items.md` in full: the duplication-level claim is confirmed already fixed by 190-02 ("every worker prompt carried `## HIVE WISDOM (Cross-Colony Patterns)` twice and now carries it once") — only the *coverage/breadth* question (hard-exclude vs. discount cross-domain entries) remains open, explicitly flagged as a product decision, not a duplication defect. Traced the same finding across three artifacts under three labels (WR-03 in 190-REVIEW.md, D-190-04-A in 190-04-SUMMARY.md, D-190-R-A in deferred-items.md's current ledger) and confirmed they are the same entity, not three different open items. |

**Score:** 7/7 truths verified

### Re-verification Detail: What Was Actually Broken And Fixed (Not Just Claimed) — Probes 1 and 4

**Probe 1 — seed a signal, run the real `runCodexContinuePlanOnly`, count the marker, revert-test the fix.**

Ran the existing regression lock fresh (not trusted from the third pass): `TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection` seeds one active FOCUS signal carrying a
distinctive marker, calls the real `runCodexContinuePlanOnly`, and asserts: `plan.ContextCapsule`
contains the marker exactly once, `plan.PheromoneSection` is empty, and `ContextCapsule + "\n" +
PheromoneSection` contains the marker exactly once. **PASS.**

Then went further than re-running the existing test — wrote a new throwaway probe,
`TestZZProbe190Pass4WrapperAssemblyDeliversSignalExactlyOnce` (`cmd/zz_probe_190_pass4_test.go`,
deleted immediately after use), that simulates continue.md's actual CURRENT documented formula —
`continue_manifest.context_capsule` + each dispatch's own `brief` + `dispatch.skill_section` — for
**every dispatch** a `HeavyFlag: true` run produces, not just the manifest-level fields the
permanent test checks. Result: 4 heavy-depth dispatches were produced (1 watcher + 3 reviewers); the
marker appeared in the full wrapper-assembled string exactly once for every one of them, and zero
times via `PheromoneSection`. Output captured: `"DEMONSTRATED: 4 heavy-depth dispatch(es) each
received the active signal exactly once via the wrapper's documented capsule+brief+skill_section
formula; plan.PheromoneSection stayed empty."` File deleted after the run; `git status --porcelain`
confirmed empty immediately after.

**Revert-test (fail-then-pass), runtime field:** Backed up `cmd/codex_continue_plan.go`, then edited
the live file to restore `PheromoneSection: resolvePheromoneSection(),` alongside `ContextCapsule`
(the exact pre-190-07 state — confirmed via `git diff` showing only that one hunk). Result:
`TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection` **FAILED** with:
```
continue_manifest.pheromone_section must stay empty ...; got "### Active Pheromone Signals\n\n**FOCUS:**\n- distinctive-continue-plan-only-pheromone-marker-3f9a"
seeded signal text must reach the manifest exactly once across capsule+pheromone_section, got 2
```
This is the exact double-delivery symptom the third pass described. Restored the file from backup;
`git diff --stat` confirmed zero diff; re-ran the test — **PASS**.

**Revert-test (fail-then-pass), wrapper prose:** Backed up `.claude/commands/ant/continue.md`, then
inserted a line containing one of the test's forbidden patterns (`` + `continue_manifest.pheromone_section` ``)
near the "Reads:" section. Result: `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery`
**FAILED** with:
```
.../continue.md still instructs prepending pheromone_section ("+ `continue_manifest.pheromone_section`")
-- the capsule is the sole carrier; concatenating both delivers every active signal twice
```
Restored the file from backup; `git diff --stat` confirmed zero diff; re-ran the test — **PASS**.

**Conclusion:** both the manifest-level fix and the wrapper-prose fix are real, load-bearing, and
correctly scoped — not decorative tests that would pass regardless of the source. This satisfies
Probe 4 (the two flipped tests are locks, not accommodations) in the same pass as Probe 1.

### Probe 2 — The heavy-depth reviewer path specifically

Traced the call chain precisely rather than assuming "heavy" is a synonym for the fixed path:

- `.claude/commands/ant/continue.md` / `.opencode/commands/ant/continue.md` instruct fetching the
  manifest via `aether host continue --dry-run --classic-ceremony $ARGUMENTS`, noting
  `--verification-depth heavy` is "equivalent for callers that already use depth flags."
- `.aether/ts-host/src/command-registry.ts`'s `continueArgs()` (read directly) always emits
  `["continue", "--plan-only", ...]`, forwarding `--verification-depth <value>` and
  `--classic-ceremony` verbatim when present — so `aether host continue --dry-run
  --verification-depth heavy` becomes `aether continue --plan-only --verification-depth heavy`.
- `cmd/codex_workflow_cmds.go`'s `continueCmd.RunE` (read directly): when `--classic-ceremony` is
  set, it forces `planOnly = true; heavyFlag = true; verificationDepth = "heavy"` (if unset); when
  `planOnly` is true (either way), it calls `runCodexContinuePlanOnly` — the exact function fixed by
  `b6671584`.
- The new throwaway probe above used `codexContinueOptions{HeavyFlag: true}` directly against
  `runCodexContinuePlanOnly` and observed 4 dispatches (1 watcher + 3 reviewers — consistent with a
  heavy-depth roster), each receiving the signal exactly once.

**DEMONSTRATED**: the heavy-depth reviewer path specifically receives the fix, not just some other
depth that happens to share code.

### Probe 3 — Full-Surface Sweep For A Fifth Instance

READ-VERIFIED (structural/negative claim — established by exhaustive enumeration and reading, not
execution, per the nature of a "nothing else exists" claim). Three consecutive prior passes each
found one more instance of the identical bug shape (native build dispatch → continue native handoffs
→ continue wrapper signals). This pass swept for a fifth on every axis the shape could recur:

1. **Every struct field named `ContextCapsule string` with a JSON tag, repo-wide** (`grep -n
   "^\s*ContextCapsule\s\+string" cmd/*.go pkg/**/*.go`, non-test): exactly 3 hits. `codexBuildManifest`
   (build wrapper, fixed 190-01/03) and `codexContinuePlanManifest` (continue wrapper, fixed this
   pass) are the only two **manifest-level** ones — both now confirmed PheromoneSection-free in live
   wiring. The third, `internalWorkerDispatchRequest` (`cmd/internal_worker_adapter.go`, the
   TS-host→Go worker-adapter boundary), forwards both `ContextCapsule` and `PheromoneSection`
   unfiltered — but tracing its only two producers (`.aether/ts-host/src/worker-dispatch.ts`'s
   `dispatchRealWorker`, which reads `dispatch.context_capsule`/`dispatch.pheromone_section` from a
   `BuildDispatch` whose Go-side counterpart, `codexBuildDispatch`, has **no such fields at all** —
   confirmed by reading its full struct declaration, so these TS reads are always `undefined`; and
   Go's own `beginDirectBuildWorkerRun`, which forwards an already-empty `PheromoneSection` from a
   `codex.WorkerDispatch` value that 190-05 already fixed) confirms this is currently dormant, not a
   live, demonstrable duplicate — no call site exists today that populates both fields for the same
   worker. This mirrors this phase's own established, accepted precedent for dormant "traps" (e.g.
   `codexWorkerDispatchesForRecovery`'s explicitly-documented inert `PheromoneSection` risk, D-190-03-A).
2. **Every `codex.WorkerDispatch{` construction site** (13 hits, `grep -rn "codex\.WorkerDispatch{"
   cmd/*.go`, non-test): all 13 read directly. Nine set `ContextCapsule` and explicitly omit
   `PheromoneSection` (commented "deliberately left unset," citing D-190-03-A/190-05); the other four
   (`persistDispatchWorkerHandoff` bookkeeping calls, `buildToWorkerDispatches`) set neither field at
   all.
3. **Every `codex.WorkerConfig{` construction site** (6 hits, non-test): `codex_build_worktree.go`
   ×2 forward an already-empty `PheromoneSection` from a `WorkerDispatch`; `command_truth.go` (quick)
   and `oracle_loop.go` (oracle) set `PheromoneSection` deliberately as their *sole* channel, because
   their own bespoke capsules (`renderQuickContextCapsule`, `renderOracleContextCapsule`) never
   render pheromones — confirmed by reading both renderers directly, and locked by
   `TestEightCommandsDeliverPheromoneExactlyOnce`'s `quick`/`oracle` sub-cases, which fail the fixture
   if this ever changes; `internal_worker_adapter.go` and `swarm_cmd.go` already covered above.
4. **The four other wrapper-facing manifests** — `codexPlanManifest` (plan/planning_manifest),
   `codexColonizeManifest`, `sealPlanManifest`, `swarmManifest` — read in full: **none has a
   manifest-level `ContextCapsule` field at all**. The specific bug shape (manifest-level capsule +
   redundant field, wrapper concatenates both) is structurally impossible on these four routes; there
   is nothing for a wrapper to concatenate.
5. **Every command wrapper source file** — `grep -rln "pheromone_section\|context_capsule"
   .claude/commands/ant/*.md .opencode/commands/ant/*.md .aether/commands/*.yaml`: only `build.md`
   (×2 platforms, `context_capsule` only, already reads "SOLE source") and `continue.md` (×2
   platforms) matched. The flat installed mirror `.claude/commands/ant-continue.md` — not covered by
   `canonicalWrapperPaths` or by the automated lock test — was independently diffed by hand
   (`diff` against its pre-fix `270dad6c` copy) and confirmed to carry the identical fix (capsule is
   the sole source; no forbidden concatenation pattern remains). `colonize.md`, `plan.md`, `seal.md`,
   `swarm.md` (both platforms) mention neither term at all.
6. **Codex platform surface**: `grep -rln "pheromone_section" .codex/` returns zero hits. By
   architecture (CLAUDE.md's "UX Architecture" table: "Codex UX | Go runtime only | No wrapper
   markdown"), Codex has no wrapper prose file that could instruct a concatenation in the first
   place — this bug class is structurally impossible there.
7. **Skill content never embedded in the capsule**: read `cmd/colony_prime_context.go` in full for
   any `## Skill` heading — none exists. Skills are injected via a wholly separate 8K-budget channel
   (`skill-inject`), never inside the colony-prime capsule, ruling out a capsule+SkillSection
   duplication vector by construction.
8. **Handoff duplication** (the other bug class this phase closed, D-190-05-A/190-06) already has
   its own 9-case breadth lock, `TestNineCommandsDeliverHandoffExactlyOnce`, re-run green this pass
   (see Truth 6/Anti-Patterns below) — covering the same 9 routes pheromones cover.

**Conclusion: no fifth instance found. The well is dry**, on the evidence enumerated above — not
because the sweep stopped early, but because every construction site of every field capable of this
bug shape, on every route, in every wrapper prose file across all three platforms, was individually
read and accounted for.

### Probe 5 — Deferred Ledger

Re-read `deferred-items.md` in full.

- **D-190-01-A**: marked RESOLVED by 190-03, with a pointer to `190-03-SUMMARY.md`. Historical
  record only.
- **D-190-03-A**: marked RESOLVED by 190-05. Historical record only; independently re-confirmed this
  pass via green regression (`TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule`,
  `TestEightCommandsDeliverPheromoneExactlyOnce`).
- **D-190-05-A**: marked RESOLVED by 190-06. Historical record only; independently re-confirmed this
  pass via green regression (`TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading`,
  `TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading`, `TestNineCommandsDeliverHandoffExactlyOnce`).
- **Stale ceremony-adapter snapshot fixtures** (from 190-02): a caste-identity rendering format drift
  (missing ant-glyph/model-tag in three committed test snapshots), proven pre-existing and unrelated
  to hive_section removal. Does not touch the phase's duplication goal.
- **Inert `hive-read` mock branches** (from 190-02): dead TS test-fixture code, cosmetic. Does not
  touch the phase's duplication goal.
- **D-190-R-A** (hive cross-domain coverage narrowing, WR-03): the task's probe instructions named
  this "D-190-04-A" — traced and confirmed this is the SAME finding under three different labels
  across three artifacts: `WR-03` in `190-REVIEW.md`, `"D-190-04-A deferred decision"` explicitly
  named in `190-04-SUMMARY.md`, and `D-190-R-A` in the current `deferred-items.md` ledger (which
  itself cites "Found during: 190-REVIEW WR-03"). Re-read its full text: the **duplication-level**
  claim is already resolved ("every worker prompt carried `## HIVE WISDOM (Cross-Colony Patterns)`
  twice and now carries it once"). What remains open is a **coverage/breadth** question — whether
  cross-domain hive entries should be hard-excluded (current behavior) or discounted-but-included
  (the deleted TS channel's old behavior) — explicitly flagged as "a product decision... not a defect
  to patch quietly." This is not a duplication defect and does not touch this phase's stated goal.

**Conclusion:** nothing else in `deferred-items.md` is a live, unresolved instance of this phase's
duplication goal. All three "D-190-0X-A" entries are RESOLVED (and independently re-verified, not
just trusted); the two remaining informational entries are format-drift and dead-test cleanup,
unrelated to duplication; D-190-R-A/D-190-04-A is a deliberately-deferred coverage/breadth product
decision that does not conflict with "no context section delivered twice."

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/codex_continue_plan.go` (`codexContinuePlanManifest`) | `PheromoneSection` no longer set; `ContextCapsule` is sole channel | ✓ VERIFIED | Read directly; confirmed by revert test (Probe 1 above). |
| `cmd/codex_continue_plan_test.go` (`TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection`) | Flipped to require capsule-sole-carrier, forbid non-empty `PheromoneSection` | ✓ VERIFIED | Read directly; mutation-confirmed to catch the reintroduced defect (Probe 1/4). |
| `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md` | Read like build.md's fixed prose ("capsule is the SOLE source of pheromone signals") | ✓ VERIFIED | Read directly, byte-identical (md5 `6e44a0a4cc48a4d7601fbd85fd5e0139`). No concatenation instruction remains; mutation-confirmed via revert test (Probe 1/4). |
| `.claude/commands/ant-continue.md` (flat installed mirror, not covered by the automated lock) | Same fix as the two nested copies | ✓ VERIFIED | Diffed directly against its pre-fix (`270dad6c`) content; confirmed identical fix applied. Not covered by `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery`'s `canonicalWrapperPaths`, so this was checked by direct read, not by an existing test. |
| `cmd/continue_wrapper_ceremony_test.go` (`TestContinueWrapperInstructsCapsuleAndPheromoneDelivery`) | Flipped to require sole-carrier language and forbid the old concatenation instructions | ✓ VERIFIED | Read directly; mutation-confirmed via revert test (Probe 1/4) using a real forbidden-pattern re-insertion. |
| `deferred-items.md` | D-190-01-A, D-190-03-A, D-190-05-A marked RESOLVED; D-190-R-A honestly scoped | ✓ VERIFIED | Read in full this pass (Probe 5); resolution text cross-checked against actual code and green regression tests, not just trusted. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `runCodexContinuePlanOnly` | `codexContinuePlanManifest{ContextCapsule}` only | `PheromoneSection` no longer set | WIRED, FIXED | Revert-tested this pass (Probe 1). |
| `.claude/commands/ant/continue.md` / `.opencode/commands/ant/continue.md` / `.claude/commands/ant-continue.md` (classic-ceremony reviewer prompt assembly) | `continue_manifest.context_capsule` only | `pheromone_section` concatenation instruction removed from all three copies | FIXED, NOT DUPLICATING | Two nested copies mutation-confirmed (Probe 1/4); flat mirror confirmed via direct diff (Probe 3, item 5). |
| `aether host continue --dry-run --verification-depth heavy` / `--classic-ceremony` | `aether continue --plan-only --verification-depth heavy` | `continueArgs()` (`.aether/ts-host/src/command-registry.ts`), read directly | WIRED, CONFIRMED | Traced the full chain to `runCodexContinuePlanOnly` (Probe 2) — the heavy-depth path is not a different, unfixed code path. |
| `executeCodexBuildDispatches` / `plannedContinueReviewDispatches` / `plannedContinueWatcherDispatch` / plan / colonize / seal / swarm | `codex.WorkerConfig`/`WorkerDispatch{ContextCapsule}` | `PheromoneSection`/redundant `HandoffSection` still absent | WIRED, UNCHANGED | RE-CONFIRMED this pass via green regression (`TestEightCommandsDeliverPheromoneExactlyOnce`, `TestNineCommandsDeliverHandoffExactlyOnce`, 9 sub-cases each). |
| `pkg/codex/prompt.go` (`AssemblePrompt`/`AssembleHostedPrompt`) | 5 independent parts (context/handoff/skill/pheromone/brief) | unchanged since 190-01 | WIRED, UNCHANGED | Fix strategy remains "stop setting the redundant field at the caller," consistent across all four now-closed instances. |
| `internalWorkerConfig` (`cmd/internal_worker_adapter.go`) | `codex.WorkerConfig{ContextCapsule, PheromoneSection}` both forwarded unfiltered from the JSON request | dormant — no current producer populates both | NOT LIVE, READ-VERIFIED | See Full-Surface Sweep item 1. Flagged for visibility, not as a gap: no call site today can trigger it, so nothing to revert-test. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection` | `plan.ContextCapsule` / `plan.PheromoneSection` | seeded pheromone signal via `store.SaveJSON`, read back through `resolveCodexWorkerContext()` in production code | Yes | ✓ FLOWING, EXACTLY ONCE |
| `TestZZProbe190Pass4WrapperAssemblyDeliversSignalExactlyOnce` (throwaway, deleted) | `plan.Dispatches[i].Brief` / `.SkillSection` combined with `plan.ContextCapsule` per the wrapper's documented formula | same seeded signal, real `runCodexContinuePlanOnly` output, real per-dispatch briefs | Yes | ✓ FLOWING, EXACTLY ONCE per dispatch (4/4) |
| `TestEightCommandsDeliverPheromoneExactlyOnce` / `TestNineCommandsDeliverHandoffExactlyOnce` | captured `codex.WorkerConfig`/`WorkerDispatch` | spy `WorkerInvoker` capturing the REAL value each production dispatch function hands to the invoker | Yes | ✓ FLOWING — re-confirmed this pass, unchanged from third pass |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Manifest-level fix (existing lock) | `go test ./cmd/ -run TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection -v -count=1` | PASS | ✓ PASS |
| Wrapper-prose fix (existing lock, 3 required + 2 forbidden patterns) | `go test ./cmd/ -run TestContinueWrapperInstructsCapsuleAndPheromoneDelivery -v -count=1` | PASS | ✓ PASS |
| New end-to-end wrapper-assembly probe (4 heavy-depth dispatches) | throwaway `TestZZProbe190Pass4WrapperAssemblyDeliversSignalExactlyOnce`, deleted after use | marker exactly once per dispatch (4/4), zero in PheromoneSection | ✓ PASS |
| Manifest-level revert-test | restore `PheromoneSection: resolvePheromoneSection()`, re-run | FAILED as expected ("got 2"), then reverted, then PASS | ✓ PASS (fix is real) |
| Wrapper-prose revert-test | re-add `` + `continue_manifest.pheromone_section` `` line, re-run | FAILED as expected (forbidden-pattern message), then reverted, then PASS | ✓ PASS (fix is real) |
| Regression: pheromone breadth lock (9 cases) | `go test ./cmd/ -run TestEightCommandsDeliverPheromoneExactlyOnce -v -count=1` | 9/9 PASS | ✓ PASS |
| Regression: handoff breadth lock (9 cases) | `go test ./cmd/ -run TestNineCommandsDeliverHandoffExactlyOnce -v -count=1` | 9/9 PASS | ✓ PASS |
| Regression: continue native handoff locks | `go test ./cmd/ -run TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading\|TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading -v -count=1` | PASS | ✓ PASS |
| Regression: criteria 1-3 named tests (7 tests) | `go test ./cmd/ -run '<7 test names>' -v -count=1` | 7/7 PASS | ✓ PASS |
| `go build ./...` | `go build ./...` | exit 0, no output | ✓ PASS |
| `go vet ./cmd/...` | `go vet ./cmd/...` | exit 0, no output | ✓ PASS |
| Flat mirror parity (`.claude/commands/ant-continue.md`) | manual `diff` against pre-fix `270dad6c` content | fix present, no forbidden pattern | ✓ PASS |
| Working tree clean after all probes and mutations | `git status --porcelain` / `git diff --stat` | both empty | ✓ PASS |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` files exist in this repository and none is named in this
phase's PLAN/SUMMARY/REVIEW files. Verified via `go test` directly plus live mutation/probe testing
in this session (see above), not a probe-script-based migration/tooling phase. Unchanged from prior
passes.

### Requirements Coverage

N/A. No `requirements:` IDs are declared in `190-05-PLAN.md`, `190-06-PLAN.md`, or any 190-07 work
(no PLAN.md exists for the orchestrator's direct fix). `grep -n "190" .planning/REQUIREMENTS.md`
returns zero matches — no REQ-IDs map to Phase 190. No orphaned requirements. Unchanged from prior
passes.

### Anti-Patterns Found

None. Scanned every file touched between the third pass and this one (`cmd/codex_continue_plan.go`,
`cmd/codex_continue_plan_test.go`, `cmd/continue_wrapper_ceremony_test.go`,
`.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`,
`.claude/commands/ant-continue.md`) for `TBD`/`FIXME`/`XXX`, `TODO`/`HACK`/`PLACEHOLDER`, and
"not yet implemented"/"placeholder"/"coming soon" language. Two incidental substring hits, both
false positives on inspection: `codex_continue_plan.go`'s pre-existing `"timeout placeholder for the
same reviewer"` (describes a result-collection design concept, not a stub) and continue.md's
pre-existing `"is not available at any cost"` (describes a security policy — skipping a credential
review is never available — not an unfinished feature). Neither is new, neither is a stub, neither is
a debt marker.

### Human Verification Required

None. This remains a backend CLI/wire-format phase (no UI, no visual rendering, no real-time
behavior, no external service integration) — every claim in this report was verifiable, and was
verified, by direct execution or direct reading of the source.

### Gaps Summary

None. The one gap the third pass found — continue's classic-ceremony/heavy-review wrapper flow
double-delivering pheromone signals — is closed, demonstrated by breaking the fix on purpose (twice:
the runtime field and the wrapper prose) and watching it fail with the exact pre-fix symptom, then
restoring and watching it pass again. A fresh end-to-end probe confirmed the fix holds across a real
4-dispatch heavy-depth run using the wrapper's own current documented assembly formula, not just the
manifest-level fields the permanent unit test checks. A full-repo sweep for a fifth instance of the
same bug shape, across every `ContextCapsule`-carrying struct, every dispatch-construction site,
every wrapper prose file on both primary platforms (including the flat installed mirror, which no
automated test covers), and the Codex platform surface, found none. The deferred ledger's one
remaining open item (D-190-R-A / D-190-04-A, hive cross-domain coverage) is a distinct, honestly-
recorded product decision about data breadth, not a duplication defect, and does not conflict with
this phase's goal. All 7 of this phase's must-haves are verified; the phase goal — "No context
section delivered twice; briefs stop transiting the orchestrator byte-for-byte" — holds in the
codebase as of HEAD (`b6671584`), not merely in a summary describing it.

---

_Verified: 2026-08-21T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
