---
phase: 190-lean-non-duplicated-delivery
verified: 2026-08-20T12:23:29Z
status: gaps_found
score: 6/7 must-haves verified
overrides_applied: 0
gaps:
  - truth: "No context section is delivered twice on ANY build dispatch path (the phase's own literal goal sentence, not just the --print-brief-inspected wrapper flow named in Success Criterion 2)"
    status: failed
    reason: >
      The native/direct build dispatch path (aether build <phase> with no --plan-only -- the
      path autopilot /ant-run uses) independently passes both ContextCapsule (which already
      renders "## Pheromone Signals") and a separately-resolved PheromoneSection into
      codex.WorkerDispatch. AssemblePrompt/AssembleHostedPrompt join both as two separate,
      both-included prompt parts, never deduplicated. Confirmed by direct code trace (not
      inference): cmd/codex_build.go:1699-1725 computes and sets both fields unconditionally;
      pkg/codex/prompt.go:64-70 and :79-85 list "context" and "pheromone" as two independent
      promptPart entries; pkg/codex/dispatch.go:175,180 forwards both fields unfiltered into
      WorkerConfig. This is the exact "enforcement covering one form of an equivalent pair"
      failure mode this project's own history names: --print-brief (the wrapper-facing form)
      is protected by the 190-01 detector; the native form is not, and no test anywhere
      inspects executeCodexBuildDispatches' own assembled AssemblePrompt/AssembleHostedPrompt
      output for pheromone duplication (TestNativeDispatchHandoffStaysExactlyOnceViaCapsule
      only covers HandoffSection on this path, not PheromoneSection). The phase's own process
      found this, verified it empirically, and honestly deferred it (D-190-03-A in
      deferred-items.md, independently re-confirmed by 190-REVIEW.md's IN-01) -- it is not a
      surprise, but "documented as deferred" is not the same as "the stated goal is true."
      Success Criterion 2's literal wording ("--print-brief asserts zero duplicated sections")
      is satisfied and unaffected by this finding -- --print-brief never inspects this path.
      The gap is specifically against the phase's broader, unqualified goal sentence.
    artifacts:
      - path: "cmd/codex_build.go"
        issue: "executeCodexBuildDispatches (lines 1699-1725) sets both ContextCapsule and PheromoneSection on every native-dispatch codex.WorkerDispatch; the capsule already contains the pheromone content."
      - path: "pkg/codex/prompt.go"
        issue: "AssemblePrompt (lines 58-74) and AssembleHostedPrompt (lines 76-87) each list \"context\" and \"pheromone\" as independent, both-included promptPart entries with no dedup check between them."
    missing:
      - "A fix scoped to executeCodexBuildDispatches alone (stop setting PheromoneSection when ContextCapsule already carries the pheromone content on this one call site) OR the broader 8-call-site architectural decision deferred-items.md's D-190-03-A already recommends."
      - "A test that asserts count(\"## Pheromone Signals\") == 1 in executeCodexBuildDispatches' own assembled native prompt with an active signal -- mirroring TestNativeDispatchHandoffStaysExactlyOnceViaCapsule's shape but for PheromoneSection instead of HandoffSection."
---

# Phase 190: Lean, Non-Duplicated Delivery — Verification Report

**Phase Goal:** No context section delivered twice; briefs stop transiting the orchestrator byte-for-byte.
**Verified:** 2026-08-20T12:23:29Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

Every claim below marked **DEMONSTRATED** was proven by running code in this session (Go
tests, TS tests, mutation tests with revert, a live TypeScript rebuild-and-diff) — not by
reading SUMMARY.md prose. Claims marked **inferred** were established by direct source
reading with no live execution. The working tree was restored to clean (`git status
--porcelain` empty) after every mutation probe; final confirmation is recorded below.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Plan-only manifests carry `brief_path` to files on disk; the wrapper passes paths; build.md prose describes `brief_path` as routine | ✓ VERIFIED | DEMONSTRATED. `TestBuildPlanOnlyManifestOmitsInlineBriefWhenBriefPathPresent` ran and passed, logging the real measurement: "7 dispatches, 6848 bytes would have shipped inline... 364 bytes actually ship... 94.7% reduction." `TestDispatchEntryCarriesBriefPath` (direct path, unaffected) passed. All 6 prose surfaces (`.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md`, `.aether/commands/build.yaml`, `cmd/command_guide.go`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`) independently grepped and confirmed to say `brief_path` is "the routine channel" / inline `brief` is "the rare case." Wrapper triplet confirmed byte-identical via `md5` (all three: `a530d99f5e4fdd51fe52310a16e1a804`). |
| 2 | SC2 — `--print-brief` asserts zero duplicated sections; pheromones and handoffs get one home each **within the scope `--print-brief` inspects** (the wrapper plan-only flow) | ✓ VERIFIED | DEMONSTRATED. `TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs` (real active pheromone + real stored handoff) passed cleanly. `TestPrintBriefFailsOnDuplicatedSection` (duplication direction) passed. `TestPrintBriefGateCatchesAbsentSectionNotJustDuplicated` + `TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker` (absence direction, WR-02) passed. Mutation-tested: reverting CR-01's fix (repopulating `HandoffSection`) makes `TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs` FAIL with `"dispatch Dash-21 delivers duplicated context: Previous Worker Handoffs"` — confirming the gate is live, not decorative. Reverting WR-02's absence gate makes `TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker` FAIL. All mutations reverted; tree left clean. |
| 3 | SC3 — TS-host hive double-injection removed (build path and continue dry-run `hive_section`) | ✓ VERIFIED | DEMONSTRATED. `grep` of tracked `.ts` source (`host.ts`, `worker-dispatch.ts`, `types.ts`) found zero `hive_section`/`hiveSection`/`HiveSection` references. `grep HiveSection cmd/*.go` found zero non-test references. Ran `npm run build` (real `tsc -p tsconfig.build.json`) inside `.aether/ts-host` and diffed the output against the committed `dist/`: **zero-byte diff** (`git status --porcelain .aether/ts-host/dist/` empty) — dist is not stale and carries no residual hive computation. Ran the full TS suite: 537/537 tests pass, including the 5 named Phase-190 regression locks ("build/plan/continue runner does not call hive-read and never attaches hive_section", "dry-run... never contains a hive_section key") and `TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly` (Go side). `pkg/codex/*.go` grepped for any Go-side hive field: none exists outside a doc comment — hive wisdom has exactly one channel (`ContextCapsule`) on both the native and internal-worker-adapter paths. |
| 4 | SC4 — Orchestrator-relay byte count measurably drops | ✓ VERIFIED | DEMONSTRATED (same run as truth 1). The 94.7%-reduction measurement is emitted by the test itself at run time (`codex_build_test.go:909`), not asserted as a hardcoded string — a regression that shrank the saving would show a different number. `codexBuildManifest.WorkerBriefs` (`json:"worker_briefs"`) independently confirmed to hold `briefPaths` (path strings), not brief content, at both call sites (`cmd/codex_build.go:307,2162`) — no adjacent field silently reintroduces the verbatim body. |
| 5 | **The phase's own goal sentence, unqualified: "No context section delivered twice," on ANY build dispatch path** — not just the one `--print-brief` inspects | ✗ FAILED | See `gaps` in frontmatter. DEMONSTRATED false by direct code trace: `executeCodexBuildDispatches` (native/direct path, no `--plan-only` — the path autopilot `/ant-run` uses) sets both `ContextCapsule` and `PheromoneSection` on every dispatch (`cmd/codex_build.go:1699-1725`); `AssemblePrompt`/`AssembleHostedPrompt` (`pkg/codex/prompt.go:58-87`) join both as separate, non-deduplicated parts. A live FOCUS/REDIRECT/FEEDBACK signal reaches a native-dispatched worker's prompt twice, under two different headings, right now. This is real, not hypothetical — it is the phase's own honestly-logged `D-190-03-A`, independently re-confirmed by `190-REVIEW.md`'s IN-01, and independently re-confirmed a third time by this verification's own trace of `pkg/codex/prompt.go` and `pkg/codex/dispatch.go`. |
| 6 | Convergence risk (two parallel 190-04 fixes reconciled) left no franken-state | ✓ VERIFIED | Checked for duplicate function definitions across the touched files: none found (each function defined exactly once). Checked for a leftover `rescue-190-04` branch/ref: none found (confirmed deleted, matching 190-04-SUMMARY.md's claim). Checked for dead/single-caller flags: `includeSteeringSections` and `clearInlineBrief` both have two live, distinct, exercised call sites each (confirmed by reading, not just by the plan's own claim). `cleanupStaleWorkerBriefs` is called from two different, non-overlapping paths (plan-only directly, direct path via `cleanupStaleBuildAttemptArtifacts`) — not a double-call on the same path. |
| 7 | D-190-R-A (hive cross-domain coverage narrowing, WR-03) is an honestly-recorded, defensible scope call, not a phase-190-goal violation | ✓ VERIFIED | This finding is about coverage *narrowing* (a domain-matching semantics change), not duplication — it sits outside "no context section delivered twice" and "briefs stop transiting the orchestrator byte-for-byte" by its own nature. `milestone-audit.test.ts`'s comment was confirmed corrected (states the selection-semantics difference rather than claiming identical content) and the review's own resolution table marks it "Record corrected," not "functional defect." No later-phase deferral needed — it's closed as a documentation fix, not an open gap. |

**Score:** 6/7 truths verified

### Deferred Items

None. `deferred-items.md`'s own `D-190-03-A` (the finding underlying gap #5) was checked against
Phase 191, 185, and 192's ROADMAP goal/success-criteria text — none names pheromone
double-delivery, the native build dispatch path, or `codex.WorkerDispatch`'s `PheromoneSection`
field. Per the conservative-matching rule, this is kept as an active gap rather than deferred to
a named later phase.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/codex_build.go` (`writeBuildWorkerBriefFiles`) | Single brief-writing helper, `clearInlineBrief`-distinguished | ✓ VERIFIED | Exists, substantive, wired into both `runCodexBuildPlanOnlyWithOptions` (true) and `writeCodexBuildArtifacts` (false); both call sites' tests pass. |
| `cmd/codex_build.go` (`composeBuildManifestBrief`) | `includeSteeringSections`-flagged composer, two live callers | ✓ VERIFIED | `attachBuildDispatchContext` passes `false`; `writeBuildWorkerBriefFiles`'s fallback passes `true`. Both paths covered by passing, mutation-confirmed tests. |
| `cmd/codex_build.go` (`cleanupStaleWorkerBriefs`) | Brief-only cleanup, called before every plan-only write | ✓ VERIFIED | Called unconditionally at `codex_build.go:292`, ahead of `writeBuildWorkerBriefFiles`. Mutation-tested: removing the call site makes `TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs` FAIL. Confirmed untouched by `clearActiveColonyRuntimeFiles` (entomb/abandon), `aether init`'s sweep, and both worktree-GC functions (`cleanupBuildWorktrees`, `gcOrphanedWorktrees` — neither touches `.aether/data/build/`, both act only on git worktree checkouts). |
| `cmd/build_print_brief.go` (`duplicatedBriefSections`, `absentBriefSections`) | Detects both "twice" and "zero" | ✓ VERIFIED | Both wired into `printWorkerBriefs` ahead of checklist/`--full`. Mutation-tested both directions (see truth 2). |
| `cmd/build_review_190_findings_test.go` | Regression locks for CR-01/WR-01/WR-02 + ported native-handoff test | ✓ VERIFIED | All 5 tests in this file ran and passed; 3 independently mutation-confirmed to catch a reintroduced defect. |
| `cmd/internal_worker_adapter.go` | No `HiveSection` field | ✓ VERIFIED | Confirmed via grep; `TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly` passed. |
| `.aether/ts-host/src/hive-injector.ts` | Deleted | ✓ VERIFIED | Absent from tracked source and dist. |
| `.aether/ts-host/dist/*` | Rebuilt, reflects source | ✓ VERIFIED | DEMONSTRATED — live `npm run build` reproduced the committed dist byte-for-byte (zero diff). |
| Wrapper build.md triplet + build.yaml + command_guide.go + Codex skill | `brief_path`-routine, capsule-sole-source prose | ✓ VERIFIED | All 6 surfaces grepped directly; correct language confirmed in each. |
| `pkg/codex/prompt.go`, `cmd/codex_build.go` (native dispatch) | Not modified by this phase (correctly, per plan scope) — but consequently still duplicates pheromones | ⚠️ Pre-existing gap, confirmed live | See gap #5. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `runCodexBuildPlanOnlyWithOptions` | `writeBuildWorkerBriefFiles` | `clearInlineBrief=true` | WIRED | Confirmed by reading + passing/mutation tests. |
| `writeCodexBuildArtifacts` | `writeBuildWorkerBriefFiles` | `clearInlineBrief=false` | WIRED | `TestWorkerBriefFileHoldsComposedBrief`, `TestDispatchEntryCarriesBriefPath` pass unmodified. |
| `printWorkerBriefs` | `duplicatedBriefSections` | assembled `capsule+brief+skill+handoff_section` text | WIRED | Confirmed the check runs ahead of both checklist and `--full` branches; mutation-confirmed. |
| `printWorkerBriefs` | `absentBriefSections` | `expectedBriefSections(state)` against capsule | WIRED | Mutation-confirmed. |
| `attachBuildDispatchContext` | `composeBuildManifestBrief` | `includeSteeringSections=false` | WIRED | `dispatches[i].HandoffSection = ""` on the same path, mutation-confirmed via CR-01 test AND cross-confirmed to trip the live `--print-brief` gate when reverted. |
| `.aether/ts-host/src/host.ts` (4 runners) | Go `internal-worker-adapter` | no `hive_section` attached | WIRED (absence) | 5 TS regression-lock tests pass; live dry-run stdout confirmed to never contain the string `hive_section`. |
| `executeCodexBuildDispatches` (native path) | `codex.WorkerDispatch{ContextCapsule, PheromoneSection}` | both fields set unconditionally | **WIRED, BUT DUPLICATING** | This is gap #5 — confirmed by direct trace of `cmd/codex_build.go:1699-1725` and `pkg/codex/prompt.go:58-87`/`dispatch.go:175,180`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `--print-brief` checklist/duplication gate | `capsule`, `expectedBriefSections` | `resolveCodexWorkerContext()`, `pheromones.json` / `worker-handoffs.json` via `filterSignalsForPrompt` / `renderWorkerHandoffSection` | Yes | ✓ FLOWING — tests seed real persisted state (`store.SaveJSON`, `persistDispatchWorkerHandoff`) and read it back through production code, not mocks. |
| Plan-only manifest `brief_path` | `dispatches[i].BriefPath` | `writeBuildWorkerBriefFiles` → `store.AtomicWrite` | Yes | ✓ FLOWING — file content verified to match the composed brief exactly. |
| TS dry-run manifest JSON | dispatch objects | live `aether host build --dry-run` stdout, captured verbatim (not re-derived) | Yes | ✓ FLOWING. |

### Behavioral Spot-Checks

This is a Go/TS CLI phase with no server or UI to curl/click; the equivalent spot-checks are
direct test execution plus mutation testing (revert-the-fix-and-watch-it-fail), run in this
session.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full targeted regression suite (12 named tests) | `go test ./cmd/ -run '...' -v -count=1` | 12/12 PASS, 2.242s | ✓ PASS |
| Broad phase-190 test surface | `go test ./cmd/ -run 'Build\|PrintBrief\|Brief\|Hive\|InternalWorkerAdapter' -count=1` | ok, 87.287s | ✓ PASS |
| `go vet ./cmd/...` | `go vet ./cmd/...` | exit 0, no output | ✓ PASS |
| `go build ./...` | `go build ./...` | exit 0, no output | ✓ PASS |
| CR-01 mutation (repopulate `HandoffSection`) | revert, `go test -run TestPlanOnlyDispatchesCarryNoHandoffSection` | FAIL as expected, then reverted | ✓ PASS (fix is real) |
| CR-01 mutation cross-check | same revert, `go test -run TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs` | FAIL as expected — the live `--print-brief` gate independently caught it | ✓ PASS (defense-in-depth confirmed) |
| WR-01 mutation (remove `cleanupStaleWorkerBriefs` call site) | revert, `go test -run TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs` | FAIL as expected, then reverted | ✓ PASS (fix is real) |
| WR-02 mutation (disable absence gate) | revert, `go test -run TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker` | FAIL as expected, then reverted | ✓ PASS (fix is real) |
| TS host rebuild-and-diff | `npm run build` inside `.aether/ts-host`, `git status --porcelain dist/` | zero diff | ✓ PASS |
| Full TS suite | `npm test` inside `.aether/ts-host` | 537/537 pass, 0 fail | ✓ PASS |
| Working tree clean after all probes | `git status --porcelain` | empty | ✓ PASS |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` files exist in this repository and none is named in
this phase's PLAN/SUMMARY/REVIEW files. This is a Go/TS phase verified via `go test` / `npm test`
directly (see Behavioral Spot-Checks above), not a probe-script-based migration/tooling phase.

### Requirements Coverage

N/A. No `requirements:` IDs are declared in either PLAN.md's frontmatter (`requirements: []` in
both 190-01 and 190-02), and `grep -n "190" .planning/REQUIREMENTS.md` returns zero matches — no
REQ-IDs are mapped to Phase 190. No orphaned requirements.

### Anti-Patterns Found

None. Scanned all Go/TS/markdown files this phase modified (11 files across 190-01, 13 across
190-02, 10 across 190-03, plus `cmd/build_review_190_findings_test.go` from the convergence fix)
for `TBD`/`FIXME`/`XXX`, `TODO`/`HACK`/`PLACEHOLDER`, "not yet implemented"/"placeholder"/"coming
soon", and stub-return patterns (`return null`, `return {}`, `return []`, `=> {}`). Zero hits.
The two `return []...` matches found are legitimate multi-line slice literals, not stubs.

Note (INFO, not a phase defect): `.aether/ts-host/src/*.js` and `.aether/ts-host/test/*.js`
contain ~81 stale, **gitignored** (`.gitignore:114-116`), **untracked** compiled files dated
2026-05-20 — three months before this phase — including a stale `hive-injector.js` and an old
`host-integration.test.js` that still asserts `hive_section` *presence*. Confirmed inert: (a)
`dist/*.js` resolves its own relative imports within `dist/`, never into `src/`, and the runtime
loads only `dist/host.js`; (b) `npm test`'s script globs `test/*.test.ts` only, never
`test/*.test.js`, so the stale file never executes. A naive `grep -r hive_section
.aether/ts-host` (without an extension filter) would produce a false positive from this debris —
worth a `git clean -ndx .aether/ts-host/src .aether/ts-host/test` sweep for hygiene, but it
predates this phase, is not part of what shipped, and does not affect any of the truths above.

### Human Verification Required

None. This is a backend CLI/wire-format phase (no UI, no visual rendering, no real-time
behavior, no external service integration) — every claim was verifiable, and was verified, by
direct execution.

### Gaps Summary

Four of the five things this phase set out to do are genuinely, demonstrably done — verified by
running code in this session, not by trusting SUMMARY.md: plan-only briefs move to disk with a
94.7% measured byte reduction (SC1/SC4), the `--print-brief` inspector fails both when a section
repeats and when one silently drops to zero on the flow it inspects (SC2), and the TS-host's
independent hive-wisdom computation is fully removed with a live rebuild proving `dist/` carries
no residue (SC3). Three follow-on review findings (CR-01, WR-01, WR-02) were closed with tests
this verification independently confirmed are real — not name-checked — by reverting each fix and
watching the specific named test fail.

One gap remains, and it is real: the phase's own goal sentence says "No context section delivered
twice" without qualification, but the native/direct build dispatch path — the one `/ant-run`
autopilot uses, and the one nothing in this phase's diff touches — still delivers pheromone
signals twice whenever a FOCUS/REDIRECT/FEEDBACK signal is active. This was found, proven, and
honestly logged by the phase's own process (`D-190-03-A`) and independently re-confirmed by both
the code review (IN-01) and this verification's own trace of `pkg/codex/prompt.go`. It does not
contradict Success Criterion 2's literal wording (which is scoped to what `--print-brief`
inspects), but it does contradict the phase's own broader, stated goal. The fix is scoped and
named already — either gate `PheromoneSection` on this one call site the way `HandoffSection` was
gated in `attachBuildDispatchContext`, or make the larger 8-call-site architectural decision
`deferred-items.md` describes — but as of this HEAD (`7361e991`), it is not done.

**This looks like a legitimate scope boundary, not an oversight.** If the developer judges that
"no context section delivered twice" was always intended to mean "on the wrapper-facing flow
`--print-brief` inspects" (i.e., Success Criterion 2's literal scope is the actual contract, not
the broader goal prose), that is a defensible reading. To accept it, add to this file's
frontmatter:

```yaml
overrides:
  - must_have: "No context section is delivered twice on ANY build dispatch path"
    reason: "Success Criterion 2 scopes duplication-freedom to the --print-brief-inspected wrapper flow; native-path pheromone duplication is real but tracked separately as D-190-03-A for a future phase"
    accepted_by: "{your name}"
    accepted_at: "{ISO timestamp}"
```

Then re-run verification to apply it. Absent that override, this phase's own stated goal is not
yet fully true in the codebase.

---

_Verified: 2026-08-20T12:23:29Z_
_Verifier: Claude (gsd-verifier)_
