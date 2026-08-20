---
phase: 190-lean-non-duplicated-delivery
reviewed: 2026-08-20T10:58:30Z
depth: standard
files_reviewed: 28
files_reviewed_list:
  - .aether/commands/build.yaml
  - .aether/skills/colony/aether-colony-build-cycle/SKILL.md
  - .aether/ts-host/dist/hive-injector.d.ts
  - .aether/ts-host/dist/hive-injector.js
  - .aether/ts-host/dist/host.d.ts
  - .aether/ts-host/dist/host.js
  - .aether/ts-host/dist/types.d.ts
  - .aether/ts-host/dist/worker-dispatch.js
  - .aether/ts-host/src/hive-injector.ts
  - .aether/ts-host/src/host.ts
  - .aether/ts-host/src/types.ts
  - .aether/ts-host/src/worker-dispatch.ts
  - .aether/ts-host/test/hive-injector.test.ts
  - .aether/ts-host/test/host-integration.test.ts
  - .aether/ts-host/test/milestone-audit.test.ts
  - .aether/ts-host/test/worker-dispatch.test.ts
  - .claude/commands/ant-build.md
  - .claude/commands/ant/build.md
  - .opencode/commands/ant/build.md
  - cmd/build_manifest_brief_composition_test.go
  - cmd/build_print_brief.go
  - cmd/build_print_brief_test.go
  - cmd/codex_build.go
  - cmd/codex_build_test.go
  - cmd/command_guide.go
  - cmd/internal_worker_adapter.go
  - cmd/internal_worker_adapter_test.go
  - cmd/phase_baseline_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 190: Code Review Report

**Reviewed:** 2026-08-20T10:58:30Z
**Depth:** standard
**Files Reviewed:** 28
**Status:** issues_found

## Summary

Phase 190 claims three things: (01) plan-only build briefs move from inline JSON to files on disk,
with `--print-brief` failing on duplicated sections; (02) the TS host's independent hive-wisdom
computation is deleted so hive wisdom arrives exactly once; (03) pheromone signals and prior-worker
handoffs move to the shared context capsule as their one home, closing a real duplicate the 190-01
detector caught.

`go build ./...`, `go vet ./cmd/...`, the full targeted Go test set (`Build|PrintBrief|Brief|Hive|
InternalWorkerAdapter`, 100+ tests), `tsc --noEmit`, and the touched TS test files (69 tests) all
pass. The TS-side hive removal is genuinely complete — `hive_section` is gone from both `src/*.ts`
and the committed `dist/*.js`/`.d.ts`, dist was rebuilt in the same commit as the source change, and
the regression-lock tests assert on wire content, not function names. The "flipped" plan-only test
was a legitimate update, not a broken read-only guarantee — plan-only's COLONY_STATE.json immutability
is separately, still asserted byte-for-byte. D-190-03-A (the native/direct dispatch path's own,
separately deferred pheromone double-channel) was independently re-traced by this review and holds up
exactly as documented: real, pre-existing, untouched by this phase's diff, and honestly deferred.

However, plan 03's central claim — "prior-worker handoffs now have ONE home" — is false for one
specific, provable case. `attachBuildDispatchContext` still unconditionally populates the per-dispatch
`HandoffSection` struct field with the same "## Previous Worker Handoffs" content the manifest-level
`context_capsule` already carries, and that field ships, un-gated, in both `result.dispatches[]` and
`result.dispatch_manifest.dispatches[]` for every plan-only dispatch. 190-03 gated the brief's own
copy of this content; it did not gate this separate, adjacent field, and no test in the diff inspects
`handoff_section`'s content, so the duplication shipped unnoticed. Three further, lower-severity gaps
are documented below: stale worker-brief files accumulate on repeated `--plan-only` calls for the same
phase (the cleanup helper that exists for the direct path is never called on the plan-only path), the
`--print-brief` duplication gate can detect a section repeating but not a section silently dropping to
zero, and the hive-channel-removal claim of "the SAME wisdom, delivered twice" glosses over a real
difference in domain-matching semantics between the two channels.

## Critical Issues

### CR-01: `dispatch.handoff_section` still duplicates the capsule's "Previous Worker Handoffs" content — plan 03's "one home" claim is false for this field

**File:** `cmd/codex_build.go:3264` (population), `cmd/codex_build.go:67` (struct field, also serialized directly into `dispatch_manifest.dispatches[]`), `cmd/codex_build.go:2045` (`codexBuildDispatchMaps` copies it into `result.dispatches[]`)

**Issue:** Plan 190-03's fix gates only the **brief's own** rendering of prior-worker handoffs
(`composeBuildManifestBrief`'s `includeSteeringSections` flag, `codex_build.go:3324`). It never
touches the separate `HandoffSection` struct field, which `attachBuildDispatchContext` still sets
unconditionally on every dispatch:

```go
dispatches[i].HandoffSection = renderWorkerHandoffSection("build", phase.ID, dispatches[i].Name)
```

`codexBuildDispatchMaps` then copies this into `entry["handoff_section"]` whenever non-empty
(`codex_build.go:2044-2046`), and because `codexBuildDispatch.HandoffSection` carries a
`json:"handoff_section,omitempty"` tag, it is *also* serialized automatically into every entry of
`result.dispatch_manifest.dispatches[]` (the typed manifest is marshaled directly). Meanwhile
`manifest.ContextCapsule` (`resolveCodexWorkerContext()`) already renders the identical
`## Previous Worker Handoffs` section from the same source (`renderWorkerHandoffSection("build",
state.CurrentPhase, "")`, `cmd/colony_prime_context.go:695`) — the two calls differ only in the
`workerName` exclusion argument, which for a freshly-generated dispatch name essentially never
excludes anything, so the two renders are the same content in practice.

Verified by execution (throwaway probe, run then deleted, tree left clean): seeded one stored
worker-handoff record, ran `aether build 1 --plan-only`, and inspected the JSON response. Every
single dispatch in `result.dispatches[]` carried a non-empty `handoff_section` (165 bytes) containing
both the `## Previous Worker Handoffs` heading and the exact sentinel text already present in
`dispatch_manifest.context_capsule`. No test anywhere in this diff or the pre-existing suite inspects
`handoff_section`'s content (`grep -rn '"handoff_section"\]\|\.HandoffSection\b' cmd/*_test.go` finds
nothing), which is why this shipped unnoticed alongside the (correctly fixed) brief-level duplicate.

This is not neutralized by the wrapper prose: `.claude/commands/ant/build.md` correctly does *not*
instruct the wrapper to read `dispatch.handoff_section` (its "nothing else, nothing invented" list is
`context_capsule` + `brief`/`brief_path` + `skill_section`), so a wrapper that follows the doc exactly
never surfaces this duplicate to a worker's actual prompt today. But the field is still in the wire
contract, contradicts this phase's own stated and tested invariant ("one home each"), doubles the
response payload for every dispatch for no live benefit, and is a standing trap for any
less-than-perfectly-compliant wrapper implementation (a reasonable reading of "here's this dispatch's
handoff context, plus its skill context" is "use both").

**Fix:** Stop populating `HandoffSection` in `attachBuildDispatchContext` (its only caller, per
`grep -n "dispatches\[i\].HandoffSection ="`), mirroring the reasoning already applied to `Brief`:

```go
// includeSteeringSections=false path already omits this from the brief; the
// manifest-level capsule (resolveCodexWorkerContext()) is the sole channel
// for this caller (plan-only wrapper flow, --print-brief simulation) --
// leaving this field populated ships the same "## Previous Worker Handoffs"
// content a second time in result.dispatches[] and
// result.dispatch_manifest.dispatches[], contradicting 190-03's own "one
// home" claim.
dispatches[i].HandoffSection = ""
```

Then extend `duplicatedBriefSections`'s input in `printWorkerBriefs` (`cmd/build_print_brief.go:91`)
to include `single[0].HandoffSection` in the assembled text it checks, so a future regression here
trips the same detector 190-01 built for exactly this class of bug — today it silently skips this
field. Add a test asserting `handoff_section` is absent/empty in a plan-only response when a stored
handoff exists and the capsule already carries it (the mirror of
`TestPlanOnlyManifestBriefCarriesPheromones`'s existing "brief_path file must NOT also carry it"
assertion, applied to this field instead of the brief file).

## Warnings

### WR-01: Stale worker-brief files accumulate on repeated `--plan-only` calls for the same phase

**File:** `cmd/codex_build.go:288` (call site, no cleanup before it), `cmd/codex_build.go:580`
(the only call to `cleanupStaleBuildAttemptArtifacts`, direct path only), `cmd/codex_build.go:2834`
(`cleanupStaleBuildAttemptArtifacts`, which does `os.RemoveAll(.../worker-briefs)`)

**Issue:** `runCodexBuildWithOptions` (the direct/native path) calls
`cleanupStaleBuildAttemptArtifacts`, which wipes `build/phase-N/worker-briefs/` before writing fresh
brief files. `runCodexBuildPlanOnlyWithOptions` — the path this phase's Plan 01 specifically added
brief-file writing to — never calls it. Worker-brief filenames are `{dispatch.Name}.md`, and dispatch
names are deterministic hashes of `phase:task-index:task-goal-text`, so any change in a task's wording,
a `--selected-tasks` filter, or a caste/coalescing decision between two `--plan-only` calls for the
same phase produces different names, leaving the prior run's files on disk, unreferenced by the new
manifest.

Re-running `--plan-only` for the same phase without `--force` is an explicitly supported, ordinary
flow (`runCodexBuildPlanOnlyWithOptions`'s own "idle plan-only" auto-supersede logic exists
specifically so this does not require `--force`; the code comment there says blocking it "made an
aborted /ant-build jam every following one"). Verified by execution (throwaway probe, run then
deleted): a colony state with one task, run through `--plan-only` (5 brief files written), the task's
goal text edited, then `--plan-only` re-run with no other flags — the worker-briefs directory grew
from 5 to 6 files, with the original run's file (matching the *old* task text's dispatch name) still
present alongside the new run's file for the *new* text.

Nothing else ever clears this directory either — `clearActiveColonyRuntimeFiles` (entomb/abandon) and
`aether init`'s cleanup sweep both leave `.aether/data/build/` untouched.

**Fix:** Call `cleanupStaleBuildAttemptArtifacts(phaseNum)` (or a plan-only-safe equivalent that
doesn't touch `verification.json`/`gates.json`/`continue.json`/`review.json`, which the direct path's
version also clears and which may not apply here) before `writeBuildWorkerBriefFiles` in
`runCodexBuildPlanOnlyWithOptions`, e.g. immediately before `cmd/codex_build.go:288`:

```go
cleanupStaleBuildAttemptArtifacts(phaseNum)
briefPaths, dispatches, err := writeBuildWorkerBriefFiles(root, phase, buildDirRel, dispatches, generatedAt, true)
```

### WR-02: `--print-brief`'s duplication gate can detect "twice" but never "zero"

**File:** `cmd/build_print_brief.go:308-338` (`duplicatedBriefSections`), `cmd/build_print_brief.go:92`
(the hard-fail call site, whose own error text says "must appear exactly once")

**Issue:** `duplicatedBriefSections` only records a finding when a heading's count is `> 1`
(`cmd/build_print_brief.go:332`, `if n > 1`) or when the handoff-schema anchor's count is `> 1`
(`cmd/build_print_brief.go:325`). A section that is present **zero** times anywhere in the assembled
`capsule + brief + skill_section` text produces an empty result, so `printWorkerBriefs` exits 0 (no
error) exactly as it would on a healthy build. The call site's own error message, though, promises
more than the code checks: `"each owned section and the handoff schema must appear exactly once"` —
the code only ever enforces "not more than once."

Verified directly (throwaway unit-level probe, run then deleted): calling `duplicatedBriefSections`
on assembled text containing no `## Pheromone Signals` heading at all returns an empty slice — no
failure signal.

In practice this means: if a future change silently stops delivering an active pheromone signal or a
stored handoff to *either* channel (capsule or brief), `--print-brief`'s exit code stays 0. The
default checklist rendering (`renderBriefChecklist`) does mark the row `ABSENT` for a human reading
the text, but nothing converts that into a non-zero exit code, and `--full` mode has no `ABSENT`
marker at all for a raw-prompt inspection — a script or CI step that gates on exit code alone (the
natural way to wire this into automation) would not catch a silent-drop-to-zero regression, only a
duplication.

**Fix:** Either (a) tighten the check-site language to accurately describe what is verified ("must
never appear more than once" rather than "must appear exactly once"), or (b) extend
`duplicatedBriefSections` (or add a sibling check) that also fails when a section known to be
*expected* — e.g., an active pheromone signal exists in `pheromones.json`, or a fresh worker-handoff
record exists — is absent from the assembled text. Given this phase's own framing ("no context
section delivered twice") explicitly treats "exactly once" as the goal, option (b) closes the gap the
name of the check already implies it covers.

### WR-03: The hive-channel removal narrows cross-domain coverage, not just deduplicates identical content

**File:** `.aether/ts-host/src/hive-injector.ts` (deleted; see `git show
b60a0075caabf0a078a03d131b20b0cdc50be592:.aether/ts-host/src/hive-injector.ts`),
`cmd/context_weighting.go:78-97` (`filterHiveWisdomEntriesByDomain`, pre-existing, untouched by this
phase)

**Issue:** The deleted TS-side `readHiveWisdom` called `hive-read --for-worker --min-confidence 0.5`
**without** a `--domain` filter (fetching every entry regardless of domain), then applied its own
relevance scoring: an exact domain match kept full confidence, and any *other* domain was still
included but at a 0.5x confidence discount before ranking and taking the top N. Go's own,
pre-existing, unmodified-by-this-phase capsule builder
(`readHiveWisdomEntriesForDomains`/`filterHiveWisdomEntriesByDomain`, called from
`cmd/colony_prime_context.go:714`) does the opposite: when the repo has registered domain tags, any
entry whose domain is not an exact match (or `"general"`/empty) is **excluded entirely** — no
discount, no partial inclusion.

The phase's own documentation repeatedly frames this as removing "the SAME hive wisdom" delivered
twice (e.g. `milestone-audit.test.ts`'s new comment: "That module independently recomputed and
attached the SAME hive wisdom Go's colony-prime capsule already delivers"). That framing is verified
only at the *header* level (both channels render a `## HIVE WISDOM (Cross-Colony Patterns)` heading;
190-02's own before/after proof used two deliberately different marker strings to demonstrate the
*count* dropped from 2 to 1, not that the two channels selected the same entries). For a colony whose
registered domain tags don't span every domain present in `~/.aether/hive/wisdom.json`, the two
channels were **not** delivering identical content — the TS channel was the only one surfacing
cross-domain wisdom (at reduced confidence) at all. Removing it is a genuine behavior/coverage change
for that case, not pure deduplication, and it is not called out anywhere in the phase's summaries or
deferred-items log.

**Fix:** This is a documentation/verification-precision issue more than a functional defect (the
prior *duplicate* delivery was real and correctly removed at the header level). At minimum, correct
the phase record to note the selection-semantics difference rather than claiming identical content. If
the softer cross-domain, discounted-confidence behavior was intentional and valuable, port it into
`filterHiveWisdomEntriesByDomain` (include non-matching-domain entries at a reduced effective
confidence rather than excluding them outright) as a follow-up; if the hard exclusion is the intended,
final behavior, say so explicitly rather than implying no coverage changed.

## Info

### IN-01: D-190-03-A independently re-verified — confirmed accurate, correctly deferred

**File:** `cmd/codex_build.go:1694-1720` (`executeCodexBuildDispatches`), `pkg/codex/prompt.go:58-87`
(`AssemblePrompt`/`AssembleHostedPrompt`)

Per this review's mandate to verify claims by execution rather than trust the trace: this review
independently traced `executeCodexBuildDispatches` and confirmed `deferred-items.md`'s `D-190-03-A`
entry is accurate. The native/direct dispatch path (no `--plan-only`, e.g. autopilot builds) passes
both `ContextCapsule: capsule` (which already renders its own `## Pheromone Signals` section) and a
separately-resolved `PheromoneSection: pheromoneSection` into `codex.WorkerDispatch`, and
`AssemblePrompt`/`AssembleHostedPrompt` join both as independent, both-included parts — a real,
currently-shipping double-delivery of pheromone signals on that path. This is unaffected by anything
in this phase's diff (none of the touched files include `executeCodexBuildDispatches`'s own lines,
`pkg/codex/prompt.go`, or `cmd/colony_prime_context.go`), matches the "twice, never zero" shape the
phase's own probe questions asked about, and is honestly logged with concrete file/line evidence in
`deferred-items.md`. No action needed from this review; noted for the record only.

### IN-02: Minor doc-comment overclaim in `writeBuildWorkerBriefFiles`

**File:** `cmd/codex_build.go:2095-2124`

The function's doc comment states that on a write failure, "the caller's slice keeps that dispatch's
`.Brief` populated and `.BriefPath` empty" — implying a partially-populated slice is returned. The
code actually returns `nil, nil, err` on any `store.AtomicWrite` failure (`cmd/codex_build.go:2106`),
discarding the whole dispatches slice built so far, not returning it partially populated. This is
harmless in practice — both call sites (`runCodexBuildPlanOnlyWithOptions`,
`writeCodexBuildArtifacts`) treat any non-nil error as fatal and never inspect the returned dispatches
— but the comment describes more graceful degradation than the code implements. Worth a one-line
comment correction next time this function is touched.

---

_Reviewed: 2026-08-20T10:58:30Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
