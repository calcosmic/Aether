---
phase: 203-biological-runtime
part: a
reviewed: 2026-09-13T21:30:32Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - cmd/recruitment.go
  - cmd/recruitment_intent.go
  - cmd/recruitment_admission.go
  - cmd/recruitment_dispatch.go
  - cmd/recruitment_result.go
  - cmd/recruitment_recovery.go
  - cmd/recruitment_subtree.go
  - cmd/recruitment_lane.go
  - cmd/recruitment_probe.go
  - cmd/recruitment_credit.go
  - cmd/agency_contract.go
  - cmd/spawn.go
  - pkg/codex/platform_dispatch.go
  - pkg/codex/worker.go
  - pkg/codex/claimed_files.go
  - .aether/ts-host/src/platform-dispatcher.ts
  - .aether/ts-host/src/recruitment-probe.ts
  - .aether/ts-host/src/spawn-orchestrator.ts
  - .aether/ts-host/src/command-registry.ts
  - .aether/ts-host/src/host.ts
status: issues
critical: 2
warning: 3
info: 0
---

# Phase 203 (Biological Runtime) — Code Review, Part A

**Reviewed:** 2026-09-13T21:30:32Z
**Depth:** standard
**Files Reviewed:** 15 primary files (+ matching test files read for fail-ability)
**Status:** issues_found

## Summary

Part A implements the recruitment core: intent validation, the admission gate,
dispatch, exactly-once result binding, recovery classification, the governed
subtree projection, the in-repo build-lane bridge, and the native-nesting
probe (both Go and TypeScript sides). Most of the mechanical discipline this
phase advertises is genuinely present and enforced: `bindRecruitmentResult`'s
replay/conflict/generation logic is sound and its content-snapshot fields
match its diff-check fields 1:1; the manifest/intent durability ordering
("validate, record, then decide") is real; `dispatchRecruitment`'s workspace
containment and process-group-safe timeout are correctly wired; and
`cmd/recruitment_lane.go`'s in-repo build-lane bridge is genuinely called
from `cmd/codex_build.go`'s dispatch loop (not an orphan).

However, two BLOCKER-level issues were found, both matching this project's
own named failure modes. The first is a **documented-but-unproven claim of
cross-lane parity**: the phase's own synthesis document requires "a host-lane
recruitment and a native-lane recruitment against the same ledger state
produce the same allow/deny answer," and `spawn-orchestrator.ts`'s own header
comment repeats that claim verbatim — but the admission dimensions BIO-02
added (permission, path, cost, duplicate, parent-authority) are structurally
wired to apply to `spawnOriginRecruit` only, and the host/autopilot lane
calls `spawn-can-spawn` under `spawnOriginSpawnCanSpawn`, which is explicitly
mapped to an *empty* check set. The one test with a matching name
(`TestBothLanesUseOneReasonVocabulary`) proves the opposite of parity — that
the five dimensions are unreachable on that origin — so the acceptance
criterion is asserted in the synthesis doc but not enforced by any runtime
check. The second is the previously-flagged (WINDOWS.md #18) fail-open
depth-cap bypass via the unauthenticated coordinator-sentinel name match,
confirmed still present in the current code and now also reachable through
`aether recruit --parent Queen` in addition to `spawn-can-spawn`.

## Critical Issues

### CR-01: Host/autopilot recruitment lane bypasses four of BIO-02's five admission dimensions, contradicting the phase's own documented parity guarantee

**File:** `.aether/ts-host/src/spawn-orchestrator.ts:1-20, 108-142`, `cmd/recruitment_admission.go:39-57`, `cmd/recruitment_lane_test.go:298-360`

**Issue:** `spawn-orchestrator.ts`'s own header comment states: "every claim is
decided by the same Go ledger the interactive `aether recruit` lane already
consults, so a host-lane recruitment and a native-lane recruitment against
the same spawn-tree state produce the same allow/deny answer, with the same
reason vocabulary." `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md:206`
states the same thing as `SYN-203-02`'s acceptance test: "A fixture proves a
host-lane recruitment and a native-lane recruitment against the SAME
simulated ledger state produce the SAME allow/deny answer."

This is not what the code does. `processClaims` (spawn-orchestrator.ts:108-116)
calls `aether spawn-can-spawn --name --depth --caste --task --workspace`,
which runs with `Origin: spawnOriginSpawnCanSpawn`. `recruitmentAdmissionChecks`
(cmd/recruitment_admission.go:47-57) maps `spawnOriginSpawnCanSpawn` to an
**empty** check list — none of `recruitmentReasonParent`,
`recruitmentReasonPermission`, `recruitmentReasonPath`, `recruitmentReasonCost`,
or `recruitmentReasonDuplicate` ever apply on that origin. Only `spawnOriginRecruit`
(the native `aether recruit` command, and the in-repo lane in
`cmd/recruitment_lane.go`) gets all eight admission dimensions. The Go test
whose name most resembles a parity check, `TestBothLanesUseOneReasonVocabulary`
(cmd/recruitment_lane_test.go:298-360), does not assert parity at all — it
asserts the opposite: it fails if `spawnOriginSpawnCanSpawn`'s check list is
ever non-empty, and explicitly lists the five recruitment-only reasons as
"unreachable" for that origin. `203-09-SUMMARY.md:51` confirms this is a
deliberate design decision ("Origin stays spawn-can-spawn deliberately...none
of BIO-02's five recruitment-only dimensions apply"), but that decision was
never reconciled with the synthesis document's own written acceptance
criterion, and no fixture anywhere proves the "same allow/deny answer"
sentence that both the synthesis doc and the shipped source comment assert.

**Concrete failure scenario:** A worker dispatched through the TS/autopilot
host lane (`aether host build` / `aether run`) emits a spawn claim for a
caste whose resolved permission profile is repository-read-only. On the
native lane, `aether recruit --caste <that-caste> ...` is refused with
`reason: "permission"` (`recruitmentPermissionReason`, cmd/recruitment_admission.go:144-163).
On the host lane, the identical claim is admitted (`can_spawn: true`) because
`recruitmentPermissionReason` is never invoked for `spawnOriginSpawnCanSpawn`.
The same divergence applies to a workspace outside the colony root (path
containment is never checked on this lane beyond the fixed `cwd` value
supplied), a per-request cost-slot count that would exceed the remaining
whole-run budget under BIO-02's own rule, a duplicate concurrent request in
the same subtree, and a parent recorded with terminal status. Depth,
whole-run budget count, and ancestor-cycle *are* shared correctly across both
lanes — this is a partial, not total, divergence, but it means the guarantee
this phase's own governing document requires is false for four of the five
new dimensions BIO-02 exists to add.

**Fix:** Either (a) extend `spawn-can-spawn`'s advisory contract to also
apply the five recruitment-only checks when the caller supplies the data
they need (permission/workspace/cost/intent-id), with a new origin or an
opt-in flag rather than silently skipping them, and add a real cross-lane
fixture test that feeds the identical claim through both `aether recruit`
and `aether spawn-can-spawn` against the same seeded ledger state and asserts
identical `allowed`/`reason`; or (b) correct both the synthesis document and
the `spawn-orchestrator.ts` header comment to state plainly that only
depth/budget/ancestor-cycle are shared today, and that the host lane does
not yet enforce permission/path/cost/duplicate/parent-authority — so a
future reader does not rely on a guarantee that was written down but never
built.

---

### CR-02: Fail-open depth-cap bypass via unauthenticated coordinator-sentinel name — confirmed still present (tracks WINDOWS.md #18)

**File:** `cmd/spawn.go:18-34`, `cmd/recruitment.go:79-88`, `cmd/recruitment_admission.go:85-90`

**Issue:** `spawnParentIsRoot` (cmd/spawn.go:27-34) matches an unauthenticated,
caller-supplied `--parent` string against the fixed sentinel list
`spawnRootParentNames = {"Queen", "Prime-1", "Swarm"}` case-insensitively, with
no verification that the calling process is genuinely the coordinator. In
`recruitCmd` (cmd/recruitment.go:79-88), any `--parent` value that matches one
of these three names is granted `depth = 0` and `DepthIsAuthoritative = true`
purely from the string match — the same fail-open exemption already
identified for `aether spawn-can-spawn --name Queen` before this phase, now
also reachable through the new `aether recruit` command.
`validateRecruitmentIntent` (cmd/recruitment_intent.go:153) explicitly
special-cases the sentinel ("!spawnParentIsRoot(...) && !DepthIsAuthoritative"
only denies a *non*-sentinel unauthoritative parent), and
`recruitmentParentAuthorityReason` (cmd/recruitment_admission.go:86-90) does
the same — the sentinel is treated as "always authoritative" with no spawn-
tree entry required, in both the intent-validation layer and the admission
layer.

This is not a new discovery — `.planning/WINDOWS.md` entry 18 (opened
2026-09-13, same day as this review, routed to plan 203-06, currently
`open`) already documents this exact mechanism against `spawn-can-spawn` and
notes it is "PRE-EXISTING since phase 173, surfaced... by a commit security
review of cmd/recruitment.go." This review confirms, against the current
code on disk, that the bypass remains unfixed and that `cmd/recruitment.go`
(added by this phase) inherits it rather than closing it.

**Concrete failure scenario:** A helper already recruited at depth 2 (the
maximum allowed by `spawnMaxDelegationDepth`) runs
`aether recruit --parent Queen --caste builder --objective "..." --reason "..."`.
Because the literal string `"Queen"` matches the sentinel list regardless of
who is actually invoking the command, `deriveSpawnDepth`'s equivalent logic
in `recruitCmd` resolves depth 0 with `DepthIsAuthoritative = true`, and the
prospective child is admitted at depth 1 — even though the real caller is
already two levels deep. Repeating this pattern lets a chain of helpers
recruit indefinitely, defeating the two-hop cap
(`spawnMaxDelegationDepth = 2`) that exists specifically as "the runaway-
spawn and cost control" (WINDOWS.md's own wording).

**Fix:** As WINDOWS.md #18 already recommends, do not accept a bare-string
sentinel match as proof of coordinator identity. Bind the sentinel names to
something the calling process cannot self-assert (e.g., a value only the
true top-level orchestrator process can present — a launch-scoped token,
environment variable set once by the actual entry point and never forwarded
to spawned children's environments, or a recorded "this run's coordinator"
identity written once at `aether init`/build-start time and checked against).
This is tracked as owned by plan 203-06 per WINDOWS.md; this review's
purpose is to confirm it is still open in the code actually shipped for
this phase and reachable via the new `aether recruit` surface, not only the
pre-existing `spawn-can-spawn`.

## Warnings

### WR-01: Wrong reason code on recruitment-intent durable-write failure

**File:** `cmd/recruitment.go:142-152`

**Issue:** When `recordRecruitmentIntent` fails (a storage/durability error,
not a validation problem), the refusal is reported with
`"reason": recruitmentReasonScope`. `recruitmentReasonScope` is the fixed
vocabulary value for "too many declared paths" (see
`cmd/recruitment_intent.go:312-321`) and is not part of
`recruitmentAdmissionReasons()`'s declared admission vocabulary
(`cmd/recruitment_admission.go:31-37`) at all. The structurally analogous
branch a few lines later — `amendRecruitmentManifest` failing
(cmd/recruitment.go:243-253) — correctly uses `recruitmentReasonUnresolved`
for the same class of problem ("could not durably record this..."). No test
exercises the `recordRecruitmentIntent`-failure branch (confirmed: no test
blocks `recruitment/intents.json`'s path the way
`TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure`
(cmd/recruitment_admission_test.go:662) blocks the manifest path), so this
mislabeling has no coverage.

**Concrete failure scenario:** A colony whose `recruitment/intents.json` path
is briefly unwritable (disk full, permission error, concurrent lock) causes
`aether recruit` to report `reason: "scope"` — a caller or future tooling
that branches on the `reason` field (e.g. `next_action` guidance, or a
dashboard grouping refusals by declared vocabulary) would misclassify a
storage outage as "too many declared paths were named."

**Fix:**
```go
outputOK(map[string]interface{}{
    "admitted": false,
    "parent":   intent.ParentName,
    "reason":   recruitmentReasonUnresolved, // was recruitmentReasonScope
    "detail":   detail,
    "message":  fmt.Sprintf("%s -- carry on with the task alone", detail),
})
```
Add a test mirroring `TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure`
but blocking `recruitment/intents.json`'s own path instead, asserting
`reason == recruitmentReasonUnresolved`.

### WR-02: TypeScript native-nesting probe (`recruitment-probe.ts`) has no caller anywhere in the host

**File:** `.aether/ts-host/src/recruitment-probe.ts` (entire file, 255 lines)

**Issue:** `probeNativeNesting` and every other exported symbol in this file
are not imported by any other file in `.aether/ts-host/src/`. Confirmed by
grepping every `.ts` source file and the compiled `dist/*.js` bundle for a
reference to `recruitment-probe`: the only match is the file compiling
itself; no other module imports or calls `probeNativeNesting`. This mirrors
the Go-side situation, where `probeNativeNestingOnce` (cmd/recruitment_probe.go)
is likewise never called from production code — but the Go side's dispatch
path (`cmd/recruitment_dispatch.go:215-223`) explicitly documents this as a
deliberate, disclosed deferral ("a plain recruitment pays nothing for a probe
it never needed... recruitmentIntent carries no requested-adapter field yet
-- that is a future plan's job"). The TypeScript file carries no equivalent
disclosure that it is unwired; its own header comment describes it purely as
mirroring the Go contract, without noting that, unlike the Go side, nothing
in the TS host ever calls it at all — not even in an advisory, non-blocking
capacity.

**Fix:** Either wire a real call site (e.g., have the host log or surface
`probeNativeNesting()`'s result once per session the way the Go CLI could via
a future `--status`/diagnostic surface), or add the same explicit disclosure
the Go side has ("not yet wired to any dispatch decision; available for a
future plan") so a future reader does not mistake this for load-bearing code
the way this project's own named failure mode (an orphan shipped as if
reachable) has repeatedly happened before in this codebase.

### WR-03: `recruitmentEvidenceAltered` reads evidence file paths from disk with no containment check

**File:** `cmd/recruitment_recovery.go:133-153`

**Issue:** `recruitmentEvidenceAltered` calls `os.ReadFile(source)` directly
on `evidence.Source`, a path taken verbatim from a stored `recruitmentResult`'s
`Evidence` field, with no call to `validateSpendContainedPath` or any other
containment boundary — unlike `dispatchRecruitment`'s workspace path
(cmd/recruitment_dispatch.go:163-168) and `recruitmentPathReason`
(cmd/recruitment_admission.go:169-184), both of which validate a workspace
path against the colony root before use. Today this is not exploitable in
practice: neither `recruitCmd` (cmd/recruitment.go) nor
`dispatchOneInRepoRecruitment` (cmd/recruitment_lane.go) ever populates
`recruitmentResult.Evidence` when constructing a result, so `Evidence` is
always empty in the current production write paths, and this read path is
never reached with attacker-influenced data today.

**Concrete failure scenario (future-facing):** If a later plan wires a
recruited child's own reported evidence (e.g., a file path the child names
in its own completion report) into `recruitmentResult.Evidence` without
adding a containment check at that write site, `classifyRecruitmentRecovery`
(`aether recruit --status`) would read an arbitrary path off disk — including
outside the colony root — merely by classifying recovery state, an operation
this file's own doc comment describes as read-only and safe to run "at any
time."

**Fix:** Add the same containment check this codebase already applies
elsewhere before dereferencing a stored path from a durable record:
```go
if _, err := validateSpendContainedPath(repoRootFromStore(store), source, "recruitment evidence"); err != nil {
    return false, fmt.Errorf("recruitment evidence %q: %w", source, err)
}
```
placed before `os.ReadFile(source)` in `recruitmentEvidenceAltered`, so the
containment discipline is enforced at the one place evidence paths are ever
dereferenced, rather than depending on every future writer of `Evidence` to
remember to validate it first.

## Info

None beyond the items above at Info severity — no further findings.

---

_Reviewed: 2026-09-13T21:30:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
