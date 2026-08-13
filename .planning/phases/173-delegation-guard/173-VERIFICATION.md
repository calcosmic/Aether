---
phase: 173-delegation-guard
verified: 2026-08-13T12:55:31Z
status: gaps_found
score: 5/6 ROADMAP success criteria fully verified; 1 falsified with a demonstrated exploit
overrides_applied: 0
gaps:
  - truth: "Every delegation guard fails closed when its inputs are unreadable (ROADMAP Phase 173 success criterion 4, and the phase's own goal statement: 'every guard fails closed')"
    status: failed
    reason: >
      Empirically demonstrated: a worker (or any process with Bash access — no Write/Edit
      tool needed, since the PreToolUse hook only matches Write|Edit|Agent|Task, not Bash)
      can defeat the whole-run spawn budget (SPAWN-03) entirely by overwriting
      .aether/data/spawn-tree.txt with a single line of non-pipe-format text. After
      exhausting the real 20-helper budget and confirming the 21st spawn is correctly
      refused, corrupting spawn-tree.txt with `echo "garbage" > spawn-tree.txt` and
      retrying the identical spawn succeeds and records normally, reporting
      budget_consumed:1 as if the run had never happened. pkg/agent/spawn_tree.go's
      parseFile() swallows every read error AND silently discards unparseable lines,
      always returning a nil error — so a corrupted (not merely absent) tree is
      indistinguishable from a fresh one to spawnTreeBudgetState(), and the recording
      chokepoint (spawn-log, "the one chokepoint an LLM cannot route around" per the
      phase's own goal) allows and writes. The depth cap and ancestor-cycle checks are
      NOT bypassed by this same technique for non-sentinel-parented spawns (they
      correctly deny "unknown parent" once the tree is emptied), but any Queen-parented
      spawn only needs the budget check to pass, and that check has no corruption
      detection at all — only a directory-at-path check exists (cmd/internal_cmds.go's
      spawn-can-spawn-swarm fix from plan 03), and it was never applied to the
      spawn-log/spawn-can-spawn path the budget actually gates.
    artifacts:
      - path: "pkg/agent/spawn_tree.go"
        issue: "parseFile() (line 328) treats every store.ReadFile error identically to 'file absent, no history' and always returns a nil error; parseSpawnTreeBytes silently skips any line that fails to parse rather than surfacing a partial-parse signal. No caller of Parse()/parseFile() can distinguish 'legitimately empty colony' from 'corrupted file'."
      - path: "cmd/spawn_budget.go"
        issue: "spawnTreeBudgetState() (line 58) computes Consumed purely from EntriesForRun()'s returned slice; a corrupted spawn-tree.txt yields an empty slice with no error, so the whole-run ceiling silently resets to 0 instead of denying per D-19's own stated fail-closed intent."
      - path: "cmd/spawn.go"
        issue: "deriveSpawnDepth's sentinel branch (spawnParentIsRoot) never reads the tree at all, so a Queen/Prime-1/Swarm-parented spawn's depth derivation cannot be affected by corruption either way — meaning the budget check is the ONLY guard standing between a corrupted tree and unlimited depth-1 spawning, and it is exactly the one guard proven not to fail closed on this input."
    missing:
      - "A corruption-detection layer at the Parse()/parseFile() boundary that distinguishes 'file absent (0 bytes / not found)' — the T-173-24 justified exception — from 'file present but content does not parse as valid spawn-tree pipe format' — which must propagate as an error."
      - "spawnTreeBudgetReason (and, for defense in depth, spawnAncestorCycleReason) must turn that propagated error into a deny, matching the D-19 posture already implemented for spawn-runs.json corruption and for a directory obstructing spawn-tree.txt's path."
      - "A regression test proving that a spawn-log call against a corrupted-but-regular-file spawn-tree.txt is refused, mirroring TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable's existing shape but targeting spawn-tree.txt's content instead of spawn-runs.json's."
human_verification:
  - test: "Run a real /ant-build that hits the whole-run spawn budget or depth cap, and read the operator-visible output (not raw JSON) to confirm the refusal reason is visible without opening a file."
    expected: "The refusal names the helper, its would-be parent, and the reason, inside the /ant-build or /ant-continue ceremony narration a non-technical operator actually reads."
    why_human: "173-RESIDUE.md residue 8 states explicitly this is unproven: the Go runtime's Detail field is proven correct, but no plan in this phase touches build.md/continue.md, so whether it surfaces in the wrapper ceremony (as opposed to a raw CLI error) has never been observed."
---

# Phase 173: Delegation Guard Verification Report

**Phase Goal:** Recursive delegation is bounded by the runtime at the one chokepoint an LLM
cannot route around — the spawn-recording call. Depth is derived from the parent's recorded
entry rather than asserted by the caller, a whole-tree budget bounds what depth alone cannot,
every guard fails closed, and the operator can watch the tree while it grows. Nothing gains the
ability to delegate in this phase; enforcement lands before capability because parent/depth
linkage is recorded at spawn time and cannot be retrofitted to past runs.

**Verified:** 2026-08-13T12:55:31Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Summary For The Owner (plain English)

This phase built five separate checks that are supposed to stop an AI helper from spawning
too many other helpers, or spawning helpers too many levels deep, or asking a helper to redo
work a level above it already started (a loop that would never stop on its own). Four of those
five checks hold up under real testing, including me deliberately trying to break them. The
fifth — the one that limits the *total number* of helpers a single run can create — can be
switched off by any helper simply overwriting one internal record file with one line of garbage
text, something a helper can already do with an ordinary shell command (no special access
required). After that one write, the "you've used all 20 helpers you're allowed" limit resets to
zero and stays broken for the rest of the run. I proved this by actually doing it: I ran a
program up to its real limit, watched it correctly refuse the 21st helper, then overwrote the
record file, and watched the exact same request succeed as if nothing had happened.

The team that built this phase was unusually honest about a related, narrower version of this
same weak spot — they wrote it down in their own residue notes before I ever tested it. What I
found goes further than what they wrote down: I turned their disclosed limitation into a working
demonstration that the count-based limit can be reset to zero on demand, not just "undercounted
in an edge case." That is worth fixing, or worth a deliberate, written decision to accept the
risk, before this phase is called done.

## Goal Achievement

### Observable Truths (ROADMAP Phase 173 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `spawn-can-spawn --depth 99` denies; a spawn one level past the cap exits non-zero and writes no spawn-tree entry | VERIFIED | `go test ./cmd -run 'TestSpawnCanSpawnDeniesPastDepthCap\|TestSpawnLogRefusesPastCapAndWritesNoEntry'` passes. Manually reproduced: `spawn-can-spawn --depth 99` → `can_spawn:false`, reason `depth`. Built a 2-level tree, attempted a 3rd-level spawn → exit 1, entry count unchanged (2 before, 2 after). |
| 2 | `spawn-tree-depth` reports depth 2 for a three-level tree built entirely with `--depth 0`, because depth is derived from the parent's own entry | VERIFIED | `go test ./cmd -run 'TestSpawnTreeDepthReportsTwoForAThreeLevelTree\|TestSpawnLogDerivesDepthFromRecordedParent\|TestWorkersMdStatesOneDepthConvention'` passes. Manually reproduced: built Queen→L1→L2 all with `--depth 0`; `spawn-tree-depth` reported `max_depth:2`. |
| 3 | With the wave cap at 8 and the tree budget at 20, a run never exceeding 8 per wave is refused at helper 21; wave-1 consumption is not restored in wave 2 | VERIFIED (for the non-adversarial case the wording describes) | `go test ./cmd -run 'TestSpawnTreeBudgetRefusesTheTwentyFirstHelper\|TestSpawnTreeBudgetIsNotRestoredBetweenWaves\|TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities'` passes. Manually reproduced: two waves of 8 (16 total), budget reported 17/20 after the 17th; a third wave of 4 reached 20/20; the 21st was refused by name. **See the criterion-4 gap below: this same budget is trivially resettable to 0 by any process that can write a single line to `spawn-tree.txt`, which undermines the "bounds what depth alone cannot" framing this criterion sits inside of, even though the criterion's own literal wording (about wave-shape independence) holds under normal operation.** |
| 4 | With `COLONY_STATE.json` and the spawn tree made unreadable, every delegation guard denies and exits non-zero, and the hook denies an unresolvable requester | **FAILED** | See Gap 1 below. Three of four guards never read `COLONY_STATE.json` (this is disclosed, bounded residue, not new). The hook's own unresolved-requester deny is VERIFIED (`TestHookPreToolUseDeniesUnresolvedRequesterDepth` passes; the hook doesn't touch either file). `spawn-can-spawn-swarm` correctly denies on both a corrupted `COLONY_STATE.json` and a directory obstructing `spawn-tree.txt`'s path (manually reproduced, exit 1 both times). **But the authoritative recorder, `spawn-log`, and the advisory `spawn-can-spawn`, do NOT deny when `spawn-tree.txt` is a corrupted-but-regular file** — they silently treat it as an empty, fresh tree and allow. This is not a directory-obstruction edge case; it is a plain `echo "garbage" > spawn-tree.txt`, and it demonstrably resets the whole-run budget to 0 (see Gap 1). |
| 5 | A spawn whose (caste, normalised task) already appears in its own ancestor chain is refused, naming the ancestor | VERIFIED | `go test ./cmd -run 'TestSpawnCanSpawnDeniesAncestorCycle\|TestSpawnAncestorCheckAllowsDifferentTaskSameCaste\|TestSpawnAncestorCheckAllowsSameTaskDifferentCaste'` passes. Manually reproduced: A1 (builder, "fix the login form") spawns C1 with the identical caste+task → denied, message names A1 by depth. Also confirmed the ancestor check correctly denies (does not silently allow) when the tree is corrupted and the direct, non-sentinel parent can no longer be resolved. |
| 6 | `spawn-tree-active` renders the tree indented by depth with parent attribution while a run is in progress, and mutates nothing | VERIFIED | `go test ./cmd -run 'TestSpawnTreeActiveRendersIndentedByDepthMidRun\|TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON\|TestSpawnTreeActiveMutatesNothing\|TestSpawnTreeActiveKeepsItsJSONContract'` passes. Manually reproduced against a live (uncompleted) two-level tree: correct two-space-per-level indentation, English caste labels, "sent here by" parent attribution; file hashes identical before/after two consecutive runs. |

**Score:** 5/6 criteria hold as literally worded; criterion 4 is falsified by a reproducible exploit that also undercuts the spirit of criterion 3 and the phase's own goal statement ("every guard fails closed").

### The Decisive Reproduction (Gap 1)

Run from a clean scratch colony store, using the phase's own built binary:

```
# Legitimately exhaust the 20-helper whole-run budget
for i in 1..20: aether spawn-log --parent Queen --caste builder --name "W$i" --task "task $i" --depth 0

# Confirm the 21st is correctly refused
$ aether spawn-log --parent Queen --caste builder --name W21 --task "..." --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned
in this run; Queen may not spawn another","code":1}
exit: 1

# A worker corrupts the record with one shell line -- no Write/Edit tool needed,
# since .claude/settings.json's PreToolUse hook only matches Write|Edit|Agent|Task,
# never Bash:
$ echo "garbage not pipe format" > .aether/data/spawn-tree.txt

# The identical, already-refused request now succeeds:
$ aether spawn-log --parent Queen --caste builder --name W22-BYPASS --task "..." --depth 0
{"ok":true,"result":{"budget_consumed":1,"budget_max":20,"depth":1,"depth_source":"derived",
"name":"W22-BYPASS","parent":"Queen","recorded":true, ...}}
exit: 0
```

This is not the same finding as 173-RESIDUE.md's own disclosed residue #7 (which frames the gap
as "the fail-closed proof is narrower than the criterion's literal wording, per-guard"). This
reproduction goes further: it demonstrates that the disclosed gap is not merely a proof-scope
limitation but a working, low-effort exploit path with a concrete, damaging effect — the
whole-run budget (the mechanism the phase's goal statement calls out by name as bounding "what
depth alone cannot") can be reset to zero, repeatedly, by any process able to write one line to
one file, at any point during a run.

**This looks unintentional**, unlike CR-01 (which the team caught and fixed same-day) — no
plan text, review finding, or residue entry names this specific reset-to-zero consequence,
though 173-03-SUMMARY.md's "Bounded Residue" section and 173-RESIDUE.md residue #7 both name the
underlying mechanism (`Parse()` cannot distinguish absent from corrupted) that makes it possible.
To accept this as a scoped, honestly-narrowed limitation rather than close it, add an override to
this file's frontmatter recording that decision explicitly; otherwise this is a build gap for a
follow-up plan.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/spawn.go` | Depth cap, decision chokepoint, sentinel resolution | VERIFIED | `spawnMaxDelegationDepth=2`, `spawnCanSpawnDecision` real 3-check chain, `deriveSpawnDepth` parent-lookup authority — all present, tested, wired |
| `cmd/spawn_budget.go` | Whole-run budget of 20, midden write on ceiling only, 75% warning | VERIFIED (mechanism); gap noted above (integrity of its input) | Present, tested; `spawnTreeBudgetState` has no corruption detection (Gap 1) |
| `cmd/spawn_ancestor.go` | Ancestor-chain cycle check | VERIFIED | Present, tested, correctly fails closed on an unresolvable non-sentinel chain |
| `cmd/spawn_reap.go` | Staleness scan, reap mutation, `spawn-orphans` command | VERIFIED | Present, tested, wired into `beginRuntimeSpawnRun` and `/ant-patrol` |
| `cmd/hook_cmds.go` | Fail-closed PreToolUse Agent/Task deny path; env-var-only capture (CR-01 fix) | VERIFIED | `hookSpawnDenyReason` present and tested; `TestHookCaptureHasNoFileBasedSwitch` passes, confirming CR-01's fix holds |
| `.planning/phases/173-delegation-guard/173-HOOK-FINDINGS.md` | Dated empirical verdict | VERIFIED | Contains `VERDICT: HOOK_FIRES_IN_SUBAGENT`, raw payloads, named residue |
| `.aether/workers.md` | States one depth convention agreeing with D-01/D-05 | PARTIAL — see WR-01 below | The behaviour table and both `spawn-log` examples were corrected (verified); the separate "SPAWN CAPABILITY" child-prompt template block (lines 364-369) still tells a depth-2 helper it MAY spawn and cites a nonexistent "Depth 3" — contradicts D-05/D-01 in the exact block workers paste into child prompts |
| `.planning/phases/173-delegation-guard/173-RESIDUE.md` | Named, bounded residue mapped to success criteria and requirements | VERIFIED | Present; nine residues named; residue 7 explicitly anticipates the question this verification resolves against criterion 4 |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `.claude/settings.json` | `aether hook-pre-tool-use` | `Agent\|Task` PreToolUse matcher | WIRED | Matcher present; confirmed the hook does NOT also match `Bash`, which is load-bearing for Gap 1's reproduction (a worker doesn't need Write/Edit to corrupt `spawn-tree.txt`) |
| `cmd/spawn.go spawnLogCmd` | `spawnCanSpawnDecision` | pre-record enforcement | WIRED | Confirmed: a denied decision never reaches `RecordSpawn` |
| `cmd/ci_wiring_gate_test.go wiringGateGuardFiles` | `.github/workflows/ci.yml` `-run` filter | `TestWiringGateStepRunsEveryWiringTest` | WIRED | All five of this phase's new guard test files, and every test function inside them, are present in the CI filter (`go test` confirms the ratchet passes) |
| `.claude/commands/ant/patrol.md` | `aether spawn-orphans` | patrol health-check bullet | WIRED (but see WR-06) | The wrapper markdown calls it; the declared YAML source (`patrol.yaml`) does not mention it, so a future regeneration from the YAML spec would silently drop this the only user-facing call site for the reaper |

### Data-Flow Trace (Level 4)

Not applicable in the conventional sense (this phase is CLI/guard logic, not a UI rendering
pipeline) — the equivalent trace performed here is the exploit reproduction above, which follows
the data from a corrupted on-disk record through `Parse()` → `spawnTreeBudgetState()` →
`spawnTreeBudgetReason()` → `spawnCanSpawnDecision` → `spawnLogCmd.RunE` → `RecordSpawn`, and
confirms the corruption is silently absorbed at the first step and never surfaces as a deny at
any later one.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Depth cap denies at 99 | `spawn-can-spawn --depth 99` | `can_spawn:false`, reason `depth` | PASS |
| 3rd-level spawn refused, writes nothing | `spawn-log` past 2-level cap | exit 1, entry count unchanged | PASS |
| `spawn-tree-depth` derives true depth | 3-level tree, all `--depth 0` | `max_depth:2` | PASS |
| Whole-run budget refuses at 21, wave-independent | 3 waves (8/8/4), then a 21st | 21st refused by name; 17/20 correctly reported after wave 2 | PASS |
| Ancestor cycle refused, ancestor named | A1 repeats its own caste+task one level down | denied, names A1 and its depth | PASS |
| `spawn-tree-active` renders live, mutates nothing | mid-run 2-level tree | correct indentation/attribution; file hashes identical across two runs | PASS |
| `spawn-can-spawn-swarm` fails closed | corrupted `COLONY_STATE.json`; directory at `spawn-tree.txt` | both denied, exit 1 | PASS |
| **Recorder fails closed on tree corruption** | `spawn-log`/`spawn-can-spawn` against a garbled (regular-file) `spawn-tree.txt` | **allowed; budget silently reset to 0** | **FAIL — Gap 1** |
| CR-01 regression lock | `TestHookCaptureHasNoFileBasedSwitch` | passes | PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared by this phase's PLAN/SUMMARY files, and none
exist under that convention for this phase's subject area. Skipped: no runnable probe artifacts
found (`find scripts -path '*/tests/probe-*.sh'` returns nothing relevant; this phase's own
verification method is Go's test runner plus manual CLI reproduction, both exercised above).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| SPAWN-01 | 173-04 | Refuse at a configured depth | SATISFIED | Verified above; unaffected by Gap 1 (the exploit does not defeat the depth cap for non-sentinel parents, and sentinel-parented spawns are always depth 1 regardless) |
| SPAWN-02 | 173-02 | A spawned child records its true depth | SATISFIED | Verified above |
| SPAWN-03 | 173-05 | Whole-tree budget bounds what depth alone cannot | **BLOCKED** | The budget mechanism is correct under normal operation but is not a genuine bound against an adversarial or buggy worker — see Gap 1. A "bound" that any process can reset to zero with one shell write does not satisfy the requirement's own framing ("bounds what depth alone cannot") |
| SPAWN-04 | 173-01, 173-03, 173-07, 173-10 | PreToolUse hook denies before platform acts; fails closed on unresolved requester | SATISFIED | The hook itself is fully verified and does not depend on either corrupted file; `spawn-can-spawn-swarm`'s named D-19 defect is fixed and tested |
| SPAWN-05 | 173-06 | Ancestor-chain repetition detected and refused | SATISFIED | Verified above, including under tree corruption (denies, does not silently allow, for non-sentinel-parented spawns) |
| SPAWN-06 | 173-02, 173-10 | Decision: what depth 0 means | SATISFIED (decision recorded); PARTIAL on its documentation consequence | D-05 recorded in 173-CONTEXT.md; `TestWorkersMdStatesOneDepthConvention` passes, but see WR-01 below — the child-prompt template block was not covered by that test and still contradicts the recorded convention |
| SPAWN-07 | 173-08 | Operator watches the tree grow | SATISFIED | Verified above, including operator sign-off recorded in 173-08-SUMMARY.md |
| SPAWN-08 | 173-09 | Abandoned child reaped, budget released, operator command | SATISFIED | Verified above via tests; not directly exercised manually but automated coverage is thorough and specific (later-timestamp rule, threshold configurability, mutation purity) |

No orphaned requirements: REQUIREMENTS.md lists SPAWN-01 through SPAWN-08 for Phase 173, and every
one is claimed by at least one plan's frontmatter `requirements:` field (cross-referenced above).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `pkg/agent/spawn_tree.go` | 328-339 | Silent error-swallowing (`parseFile` always returns nil error) | 🛑 Blocker (via Gap 1's exploit chain) | Root cause of the whole-run budget bypass |
| `.aether/workers.md` | 364-369 | Stale/contradictory prose (WR-01, open) | ⚠️ Warning | Child-prompt template still tells a depth-2 helper it MAY spawn and cites a "Depth 3" the recorded convention says cannot exist. Runtime still denies the attempt (fail-closed holds), but a worker following this text wastes a turn and the SPAWN-06 decision is not honestly reflected everywhere in the one document workers actually read from |
| `.aether/workers.md` | 304 | Stale documented return shape (WR-02, open) | ⚠️ Warning | Documents `max_spawns`/`current_total` keys that do not exist in `spawn-can-spawn`'s real JSON output |
| `cmd/spawn_budget.go` | 99-117 | Inspection command mutates state (WR-03, open) | ⚠️ Warning | `spawn-can-spawn` (no `--enforce`, purely advisory) writes to `midden.json` when called at the budget ceiling, violating CLAUDE.md's own Definition-of-Done corollary that inspection/dry-run commands must not mutate state. Confirmed by code reading: `spawnCanSpawnCmd`'s `RunE` calls `spawnCanSpawnDecision` unconditionally, which calls `spawnTreeBudgetReason`, which calls `spawnTreeBudgetCeilingToMidden` on every call once the ceiling is reached, regardless of `--enforce` |
| `cmd/hook_cmds.go` | 177-201 | Two-sided heuristic weakness (WR-04, open) | ⚠️ Warning | Named, bounded, and disclosed by the phase's own team; not independently exploited further in this verification since `spawn-log` remains the authoritative backstop even when the hook's heuristic is fooled |
| `cmd/spawn.go` | 94-121 | Check-then-act race across processes (WR-05, open) | ⚠️ Warning | Not independently reproduced here (requires genuine concurrent processes); code-confirmed still present. Compounds Gap 1's severity: even without deliberate corruption, concurrent spawns near the ceiling can already exceed it by race |
| `cmd/spawn_ancestor.go` | 69-77 | Silent truncation on unresolvable mid-chain ancestor (WR-07, open) | ⚠️ Warning | Named, bounded, disclosed; not independently reproduced here |
| `.aether/commands/patrol.yaml` vs 3 wrapper files | — | Spec/wrapper drift (WR-06, open) | ⚠️ Warning | Confirmed: `patrol.yaml` (the declared source of truth) does not mention `spawn-orphans`; all three wrapper markdown files do |

No unreferenced `TBD`/`FIXME`/`XXX` markers found in the files this phase modified.

### Human Verification Required

**1. Refusal reason visible to the non-technical operator inside the real ceremony**

**Test:** Run a real `/ant-build` (or `/ant-continue`) that hits either the depth cap or the
whole-run budget ceiling, and read the output as the owner would — not the raw JSON, the
narration shown in the terminal.
**Expected:** The refusal names the helper, its would-be parent, and the reason, in plain
English, inside the ceremony narration.
**Why human:** 173-RESIDUE.md residue 8 states this explicitly as unproven: no plan in this
phase touches `build.md`/`continue.md`. The Go-level `Detail` string is proven correct and
present in the CLI's own JSON/error output, but whether it reaches the wrapper-level narration a
non-technical operator actually reads has never been observed.

### Gaps Summary

Five of the six ROADMAP success criteria for this phase hold under both the phase's own
extensive automated test suite (all named tests re-run here and passing) and independent manual
CLI reproduction performed during this verification, including several deliberate attempts to
break each guard. The depth cap, the ancestor-cycle check, the reap/orphan system, the live tree
view, and the PreToolUse hook's own fail-closed behavior are all genuinely built and working —
this is unusually well-tested code, and the CR-01 critical finding from the phase's own code
review is fixed and locked by a passing regression test (`TestHookCaptureHasNoFileBasedSwitch`).

The one gap is serious: the whole-run spawn budget — the mechanism the phase's own goal
statement singles out as bounding "what depth alone cannot" — can be reset to zero at any point
during a run by any process that writes one line of non-conforming text to
`.aether/data/spawn-tree.txt`. This does not require the Write or Edit tool (which the hook
guards); an ordinary Bash redirect is sufficient, and Bash is not covered by the
`PreToolUse` `Write|Edit|Agent|Task` matcher. The phase's own team came close to finding this —
`pkg/agent/spawn_tree.go`'s error-swallowing `parseFile()` is named as residue in both
173-03-SUMMARY.md and 173-RESIDUE.md's residue #7 — but the residue was framed as a proof-scope
limitation ("fail-closed is proven per-guard, against inputs it actually reads"), not as the
demonstrated, working budget-reset exploit this verification reproduces.

The depth cap and ancestor-cycle checks are NOT defeated by the same technique for
non-sentinel-parented spawns (they correctly deny "unknown parent" once the tree is emptied by
corruption) — only the budget check, and only for the common Queen-parented case, since depth
derivation for a coordinator sentinel never needs to read the tree at all.

Given this project's own precedent for exactly this situation (Phase 172's CR-06, and this
phase's own RESIDUE.md explicitly leaving criterion 4 "to be judged against evidence at
verification time"), the two honest paths forward are: (a) close the gap with a scoped fix — make
`parseFile()`/`Parse()` distinguish "absent" from "corrupted" and have `spawnTreeBudgetReason`
deny on the latter, or (b) accept this as a named, documented risk via an explicit override
recorded in this file's frontmatter with a human's sign-off. This verification does not make that
call — it surfaces the evidence needed to make it.

---

_Verified: 2026-08-13T12:55:31Z_
_Verifier: Claude (gsd-verifier)_
