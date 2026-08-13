---
phase: 173-delegation-guard
reviewed: 2026-08-13T12:33:42Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - .aether/workers.md
  - .claude/commands/ant-patrol.md
  - .claude/commands/ant/patrol.md
  - .claude/settings.json
  - .github/workflows/ci.yml
  - .opencode/commands/ant/patrol.md
  - cmd/ci_wiring_gate_test.go
  - cmd/command_call_audit_test.go
  - cmd/hook_cmds_test.go
  - cmd/hook_cmds.go
  - cmd/internal_cmds.go
  - cmd/spawn_ancestor_test.go
  - cmd/spawn_ancestor.go
  - cmd/spawn_budget_test.go
  - cmd/spawn_budget.go
  - cmd/spawn_enforce_test.go
  - cmd/spawn_failclosed_test.go
  - cmd/spawn_reap_test.go
  - cmd/spawn_reap.go
  - cmd/spawn_runs.go
  - cmd/spawn_tree_view_test.go
  - cmd/spawn.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/parity_snapshot.json
  - cmd/testdata/regression_snapshot.json
  - cmd/visual_writer_discipline_test.go
  - cmd/write_cmds_test.go
  - pkg/agent/spawn_tree_test.go
  - pkg/agent/spawn_tree.go
  - pkg/colony/colony_test.go
  - pkg/colony/colony.go
findings:
  critical: 1
  warning: 7
  info: 6
  total: 14
status: issues_found
---

# Phase 173: Code Review Report

**Reviewed:** 2026-08-13T12:33:42Z
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

Phase 173 (delegation guard) adds a depth cap, a whole-run spawn budget, an
ancestor-cycle check, a stale-spawn reaper, a hook-level dispatch deterrent,
and fail-closed behavior across every guard. The core guard logic in
`cmd/spawn.go`, `cmd/spawn_budget.go`, `cmd/spawn_ancestor.go`, and
`cmd/spawn_reap.go` is well-structured, and the red-proof test discipline is
genuinely strong (deny paths, negative controls, byte-identical no-trace
assertions, purity hashing). `go build`, `go vet`, and every new test were
executed during this review and pass.

However, the review found one critical issue and several warnings. The
critical finding is that the hook payload capture mechanism added in Wave 0
(`captureRawHookPayload` + the `~/.aether/hook-capture-path` sentinel) is an
unvalidated write primitive that any agent in a session can activate and aim
at protected colony state — including `spawn-tree.txt` itself, where crafted
payload content parses as spawn-tree completion lines. That inverts the trust
model this exact phase was built to enforce. The warnings cluster around
residue the phase's own tests miss (a stale depth-3 spawn-capability block in
workers.md), a two-sided weakness in the hook's `agent_type` heuristic, an
inspection command that mutates state at the budget ceiling, a check-then-act
race in `spawn-log`, and wrapper/spec drift in the patrol command.

## Critical Issues

### CR-01: Hook payload capture is an unvalidated write primitive that can forge spawn-tree records and corrupt protected colony state

**File:** `cmd/hook_cmds.go:335-381` (`hookCapturePath`, `captureRawHookPayload`); enforcement gap in `protectedHookWriteReason` at `cmd/hook_cmds.go:417-444`
**Issue:** `captureRawHookPayload` appends the raw hook stdin bytes to whatever
path is named by `AETHER_HOOK_CAPTURE_FILE` or by the first line of
`~/.aether/hook-capture-path`, with **no validation of the destination**. The
comment claims "Both switches are files/variables only the operator can
create" — this is false. `~/.aether/hook-capture-path` is an ordinary file in
the user's home directory; `protectedHookWriteReason` does not protect it (it
guards `/.aether/data/`, `/.aether/dreams/`, `.env*`, `.codex/config.toml`,
and `.github/workflows/` only), so any worker with the Write tool can create
it. Once created, **every subsequent hook invocation** (the hook fires on all
Write/Edit/Agent/Task calls per `.claude/settings.json`) appends the raw JSON
payload to the named path. Consequences:

1. **Protected-state corruption:** pointing the sentinel at
   `.aether/data/COLONY_STATE.json` or `pheromones.json` appends garbage JSON
   lines, corrupting the files the Write/Edit branch of this same hook exists
   to protect. The hook's own writes bypass the hook's own guard.
2. **Spawn-tree record forgery:** pointing it at `.aether/data/spawn-tree.txt`
   lets an agent inject parseable lines. `parseSpawnTreeBytes` validates only
   that field 6 of a 7-field line is numeric, and a 4-field completion line
   needs only a recognized status in field 3. A crafted tool argument such as
   a `file_path` of `x|VictimWorker|abandoned|done` produces a captured line
   that splits into a valid completion record — marking a live worker
   abandoned, releasing its budget slot, and falsifying the exact records
   `deriveSpawnDepth` and the whole-run budget treat as authority.
3. **Silent exfiltration/append to arbitrary user files** as a lesser variant.

The capture was a Wave-0 evidence instrument (173-HOOK-FINDINGS.md is already
recorded), yet it ships permanently in the production hook path.
**Fix:**
```go
// In hookCapturePath / captureRawHookPayload:
// 1. Refuse any destination that resolves inside an .aether data tree or the
//    current repo:
resolved := resolveHookPathSymlinks(filepath.Clean(path))
if strings.Contains(filepath.ToSlash(resolved), "/.aether/") ||
    !strings.HasPrefix(resolved, filepath.Join(home, ".aether-captures")+string(os.PathSeparator)) {
    return "" // capture destination must live under ~/.aether-captures/
}
```
Better: remove the sentinel-file fallback entirely now that the findings are
captured (keep only the env var, which an agent cannot set for the hook
process), or gate capture behind a build tag / hidden debug flag. At minimum,
add `~/.aether/hook-capture-path` (and the configured capture destination) to
the protected-path deny list so the switch cannot be flipped by a worker.

## Warnings

### WR-01: workers.md still ships the old three-level spawn convention inside the child-prompt template — and the guard test cannot see it

**File:** `.aether/workers.md:364-369`
**Issue:** The Step 4 child-prompt template still reads:
```
--- SPAWN CAPABILITY ---
You are at depth {your_depth + 1}.
{if depth < 3: "You MAY spawn sub-workers if you encounter genuine surprise (3x complexity)."}
{if depth >= 3: "You are at max depth. Complete all work inline, no spawning."}

Spawn limits: Depth 1→4, Depth 2→2, Depth 3→0
```
This tells a depth-2 helper it MAY spawn (`depth < 3`), states a depth-2
sub-spawn allowance of 2, and references a depth 3 that line 272-273 of the
same file says cannot exist ("A helper cannot spawn anyone — there is no
depth 3"). This is exactly the residue SPAWN-06 set out to remove, sitting in
the one block workers literally paste into child prompts.
`TestWorkersMdStatesOneDepthConvention` (cmd/spawn_enforce_test.go:784)
checks only markdown table rows, `spawn-log --depth 0` lines, the "global
cap" phrase, and the substring "20" — none of which match this prose block,
so the test passes while the contradiction ships.
**Fix:** Rewrite the block to the two-level convention (depth 1 may spawn
helpers; depth 2 completes inline) and delete the `Depth 1→4, Depth 2→2,
Depth 3→0` line. Extend `TestWorkersMdStatesOneDepthConvention` to fail on
`"depth < 3"`, `"Depth 3"`, and any `Depth 2→N` allowance where N > 0.

### WR-02: workers.md documents a spawn-can-spawn return shape the runtime does not produce

**File:** `.aether/workers.md:304`
**Issue:** The line updated by this phase at 303 is immediately followed by a
stale contract claim:
`# Returns: {"can_spawn": true/false, "depth": N, "max_spawns": N, "current_total": N}`.
The actual command returns `can_spawn`, `depth`, `authoritative`, and (on
deny) `reason`/`detail` (cmd/spawn.go:368-376). `max_spawns` and
`current_total` do not exist; a worker parsing them gets null. CLAUDE.md's own
corollary: "A documentation claim about runtime behaviour must be testable or
removed."
**Fix:** Replace with the real shape:
`# Returns: {"can_spawn": true/false, "depth": N, "authoritative": true/false}` (+ `reason`/`detail` on deny). Consider having
`TestSpawnCanSpawnAcceptsDocumentedInvocation` also assert the documented
return keys are a subset of the actual result keys.

### WR-03: spawn-can-spawn (an inspection command) mutates midden.json when the budget is exhausted

**File:** `cmd/spawn_budget.go:107-117` (`spawnTreeBudgetReason` →
`spawnTreeBudgetCeilingToMidden`), reached from `cmd/spawn.go:357`
**Issue:** `spawnTreeBudgetCeilingToMidden` fires from inside
`spawnTreeBudgetReason`, which runs on **every** `spawnCanSpawnDecision` call
— including the purely advisory `spawn-can-spawn` report path (no
`--enforce`, no spawn attempted). Once the run is at the ceiling, every
advisory check appends a midden entry. This violates the repo's own
Definition-of-Done corollary ("an inspection or --dry-run command must not
mutate state" — the same defect class as the `consolidation-*-dry-run` bug
CLAUDE.md documents), and a worker polling `spawn-can-spawn` in a loop at the
ceiling floods midden.json with duplicate entries.
`TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` only exercises the
spawn-log path, so this is untested.
**Fix:** Move the midden write out of `spawnTreeBudgetReason` and into
spawn-log's deny branch (the actual refused spawn attempt), or pass an
`attempting bool` through `spawnDecisionInput` and write midden only when
true. Add a purity test: `spawn-can-spawn` at the ceiling changes no file
hashes.

### WR-04: The hook's agent_type heuristic fails both ways — an aether-*-typed second-tier helper bypasses it, and legitimate non-Aether nested dispatches are blocked

**File:** `cmd/hook_cmds.go:177-201` (`hookSpawnDenyReason`)
**Issue:** The requester-depth resolution keys entirely on the `agent_type`
prefix:
```go
case strings.HasPrefix(strings.ToLower(agentType), hookAetherAgentTypePrefix):
    requesterDepth = 1
```
(a) **Under-block:** nothing forces a first-tier worker to dispatch its
helper with `subagent_type="general-purpose"`. If it dispatches with any
registered `aether-*` type (all 27 exist in `.claude/agents/ant/`), the
helper's own subsequent dispatch carries `agent_type: "aether-builder"`, is
classified depth 1, and the hook allows a **third-level platform dispatch**.
`spawn-log` will refuse to record it, but the depth-3 subagent still runs —
the invisible-runaway blind spot this phase exists to close. The function's
bounded-claims comment does not name this bypass, and no test covers an
aether-*-typed second-tier requester.
(b) **Over-block:** every non-`aether-*` subagent in this repo that
legitimately dispatches its own subagent (including non-Aether dev tooling —
e.g. GSD workflow agents — and any user-defined agent) is denied with
"cannot resolve who is asking to delegate", because the hook is installed
repo-wide via `.claude/settings.json` for all Agent/Task calls, not only
Aether colony dispatches.
**Fix:** For (a), treat an `aether-*` agent_type as depth 1 only when the
dispatch **target** (`tool_input.subagent_type`) is `general-purpose` or
non-aether; deny when an aether-typed requester dispatches another
aether-typed agent, or record/consult a session-scoped depth marker. At
minimum, document the bypass in the bounded-claims comment and add a test
pinning current behavior for an aether-*-typed requester. For (b), consider
scoping the deny to sessions with an active colony (state file present) so
non-Aether tooling in the repo is not collateral.

### WR-05: spawn-log's guard decision and RecordSpawn are not atomic — concurrent spawns can exceed the whole-run budget

**File:** `cmd/spawn.go:94-121`
**Issue:** `deriveSpawnDepth`, `spawnCanSpawnDecision` (budget + ancestor
checks), and `st.RecordSpawn` each re-read `spawn-tree.txt` /
`spawn-runs.json` independently; no lock spans check-then-record. Parallel
first-tier workers each running `aether spawn-log` for their helpers is the
system's normal operating mode (separate processes). Two processes that both
observe `Consumed == 19` both pass `spawnTreeBudgetReason` and both record,
yielding 21+ entries — the "limit that gets raised under pressure" D-02
explicitly forbids, raised by race instead of by config. The same window
exists for the ancestor-cycle check.
**Fix:** Perform the decision inside the same `store.UpdateFile` critical
section that writes the entry (compute budget/ancestors from the `existing`
bytes passed to the update callback and return an error to abort the write),
or take the store's file lock across decision + record. A test spawning N
concurrent spawn-log invocations at `Consumed == Max-1` and asserting exactly
one succeeds would pin it.

### WR-06: Patrol wrapper markdown edited directly while its declared YAML source was not updated — the spawn-orphans instruction can be silently dropped

**File:** `.aether/commands/patrol.yaml` (unchanged) vs
`.claude/commands/ant-patrol.md:12`, `.claude/commands/ant/patrol.md:12`,
`.opencode/commands/ant/patrol.md:12`
**Issue:** All three patrol wrappers gained the "also run `aether
spawn-orphans`" step, and each carries the header "Aether-managed: runtime
spec at `.aether/commands/patrol.yaml`. Synced by aether update." — but
`patrol.yaml` (the declared source of truth per CLAUDE.md's YAML Source
Chain) says nothing about spawn-orphans. Any regeneration or sync from the
YAML spec will clobber the wrappers and silently remove the orphan-listing
step, which is the only user-facing surfacing of the reaper this phase built.
This is the exact "documented but not wired" decay mode the repo's audits
keep finding.
**Fix:** Add the spawn-orphans guidance to `patrol.yaml` (e.g., a `runtime`
follow-up command or a guardrail line) so spec and wrappers agree, or update
the wrapper header if wrappers are now canonical for patrol.

### WR-07: spawnAncestorChain silently truncates on a dangling mid-chain parent instead of failing closed

**File:** `cmd/spawn_ancestor.go:69-77`
**Issue:** The walk breaks silently when an ancestor's parent name is not in
the tree (`parent, ok := byName[current.ParentName]; if !ok ... break`) and
the loop only terminates cleanly on a coordinator sentinel. An unresolvable
*startName* denies (D-19 fail-closed), but an unresolvable *ancestor* — the
same "corrupted or hand-edited tree" hazard, e.g. a parent whose line was
corrupted and skipped by `parseSpawnTreeBytes` — yields a shortened chain and
an allow, with no signal that the guard examined less than the full ancestry.
The two branches apply opposite policies to the same class of unreadable
evidence, and neither the function comment nor
`TestSpawnAncestorCheckFailsClosedOnUnreadableTree` covers the mid-chain
case.
**Fix:** Distinguish "reached a root sentinel" (complete chain, allow) from
"parent named but unresolvable" (incomplete chain): return an error for the
latter so `spawnAncestorCycleReason` denies with the existing "ancestor chain
unreadable" message, and add a test with a corrupted middle ancestor.

## Info

### IN-01: "73 workers" arithmetic counts the coordinator as a spawned worker

**File:** `.aether/workers.md:280-283`; `cmd/spawn_budget.go:19-20`
**Issue:** "8 workers wide and 2 levels deep — 73 workers in total" is 1
coordinator + 8 workers + 64 helpers. The budget's own rule (D-03) counts
only spawned helpers — 72 — and the coordinator "is never recorded in the
spawn tree". Both the doc and the code comment call all 73 "workers".
**Fix:** Say "72 spawned helpers (73 agents counting the coordinator)".

### IN-02: spawn-orphans --clear reports partial success as pure failure

**File:** `cmd/spawn_reap.go:277-281`
**Issue:** `spawnReapStaleEntries` deliberately continues past per-entry
failures and returns both the reaped names and a joined error, but the
command's `--clear` path discards the names on any error and reports only
"reap encountered an error" — entries were mutated and budget released
without the output naming what changed.
**Fix:** On partial failure, still render the reaped list and budget delta
alongside the error.

### IN-03: Midden entry IDs can collide, and ceiling refusals grow midden.json unboundedly

**File:** `cmd/spawn_budget.go:148`
**Issue:** `midden_%d_%d` (unix seconds + pid) collides for two refusals in
the same second in one process, and every refused attempt at the ceiling
appends a new entry with no dedup — a retry loop at the ceiling floods the
failure log (compounded by WR-03).
**Fix:** Add a nanosecond or counter component to the ID; dedup consecutive
identical ceiling entries (or reinforce a single entry, as pheromones do).

### IN-04: SessionID is decoded but never used

**File:** `cmd/hook_cmds.go:34`
**Issue:** `claudeHookInput.SessionID` is populated from the payload but no
code path reads it.
**Fix:** Drop the field, or note the intended future use where it is
declared.

### IN-05: Sentinel-coverage test walks gitignored local data, making it environment-sensitive

**File:** `cmd/spawn_enforce_test.go:696-748`
**Issue:** `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent`
walks all of `.aether/`, which includes the gitignored, local-only
`.aether/data/` tree. A developer's local artifacts containing a
`spawn-log --parent "..."` line (e.g. captured briefs or handoffs) can fail
the test locally while CI stays green.
**Fix:** Skip `.aether/data/`, `.aether/dreams/`, and `.aether/oracle/`
during the walk.

### IN-06: Capture appends a bare newline when stdin is empty

**File:** `cmd/hook_cmds.go:369-381`
**Issue:** `captureRawHookPayload` is called with `raw == nil` when stdin is
a TTY or empty; with capture enabled it still opens the file and writes a
blank line per invocation.
**Fix:** `if path == "" || len(raw) == 0 { return }`.

---

_Reviewed: 2026-08-13T12:33:42Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
