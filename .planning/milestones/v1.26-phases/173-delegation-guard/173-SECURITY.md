---
phase: 173
slug: delegation-guard
status: verified
threats_open: 0
threats_total: 83
asvs_level: 1
created: 2026-08-13
---

# Phase 173 — Security (Delegation Guard)

> Per-phase security contract: threat register, accepted risks, and audit trail.
>
> This document covers all 13 plans of Phase 173 (`173-01` through `173-13`, including gap-closure
> plans `173-11`/`173-12`/`173-13`), plus two post-plan events surfaced by code review
> (`173-REVIEW.md`): CR-01 (a worker-writable capture switch, fixed and locked same-day) and WR-08
> (a third whole-run-budget reset route, fixed and locked same-day). Threat IDs collide across
> plans (e.g. `T-173-50` through `T-173-60` are reused with different meanings in plans 09/10/11/12);
> every row below is keyed by **(plan, threat ID)**, never merged across plans.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| Claude Code platform → `aether hook-pre-tool-use` stdin | Untrusted JSON of unverified shape; field presence/naming is the platform's choice | `agent_id`, `agent_type`, `session_id`, `tool_name`, `tool_input` |
| `AETHER_HOOK_CAPTURE_FILE` env value → local filesystem write | Operator-supplied path becomes a file the hook appends to on every tool call | Raw hook payload bytes |
| Worker-supplied `--parent`/`--name`/`--depth`/`--task`/`--caste` flags → `spawn-log` → `spawn-tree.txt` | An LLM chooses these values; the recorded tree is the security-relevant artifact every later guard reads | Spawn records |
| Any process with shell access → `.aether/data/spawn-tree.txt` and `.aether/data/spawn-runs.json` | The `PreToolUse` matcher covers `Write\|Edit\|Agent\|Task`, never `Bash` — a plain shell redirect reaches these files unguarded | Ledger / run-record bytes |
| `COLONY_STATE.json` and `spawn-tree.txt` on disk → guard decisions | Written concurrently by other processes; a mid-write, truncated, or corrupt read is a normal operating condition | Colony state, spawn entries |
| Wall-clock elapsed time → a decision to mark another process's work dead | The only signal available; the judged process may still be running | Spawn timestamps |
| `SpawnReapThresholdMinutes` in colony state → reaping aggressiveness | Operator-editable value controlling how readily live work is killed | Config value |
| The phase's own written record (`173-RESIDUE.md`) → the next phase's planner | A false claim here is inherited as a guarantee | Documentation |

---

## Threat Register

Status legend: **CLOSED** = mitigation/acceptance verified in code or docs; **OPEN** = declared
mitigation absent. `threats_open: 0` — every row below is CLOSED.

### Plan 173-01 — Hook payload capture (Wave 0 observation)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-01 | Information Disclosure | mitigate | `captureRawHookPayload` (`cmd/hook_cmds.go:354`) short-circuits when `AETHER_HOOK_CAPTURE_FILE` is empty; opens with mode `0o600`; `TestHookPreToolUseCapturesRawPayloadOnlyWhenCaptureFileIsSet` passes. Path-outside-repo requirement enforced by Task 3 process, not code — acceptable per plan (operator-controlled env var, see T-173-03). | CLOSED |
| T-173-02 | Denial of Service | mitigate | Every error path in `captureRawHookPayload` returns silently (`cmd/hook_cmds.go:358-364`); no `stdout`/decision emitted regardless of capture outcome. | CLOSED |
| T-173-03 | Tampering | accept | Operator sets the env var in their own shell — no privilege boundary crossed. **Acceptance restored 2026-08-13** after a same-day regression (CR-01, see below) briefly reintroduced a worker-writable file-based switch; fix + lock verified (`TestHookCaptureHasNoFileBasedSwitch`, `cmd/hook_cmds_test.go:349`, PASS). | CLOSED (accepted) |
| T-173-04 | Spoofing | mitigate | `173-HOOK-FINDINGS.md` `## Named residue` section states no `agent_id`→`AgentName` bridge exists; plan 01 only records fields, does not treat them as identity. | CLOSED |
| T-173-05 | Elevation of Privilege | accept | Deliberate: plan 01's matcher observes only, deny path is plan 07 (`depends_on: ["173-01"]`); stated in plan 01's own objective. | CLOSED (accepted) |

### Plan 173-02 — Derived depth at the recorder

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-06 | Elevation of Privilege | mitigate | `cmd/spawn.go:92,137`: `claimedDepth` read from `--depth` but never passed to `RecordSpawn` (derived `depth` is used); reported separately as `claimed_depth`. `TestSpawnLogIgnoresCallerSuppliedDepth` exists (`cmd/spawn_enforce_test.go`). | CLOSED |
| T-173-07 | Elevation of Privilege | accept, bounded | `deriveSpawnDepth` (`cmd/spawn.go:45-53`) requires a recorded parent or sentinel — cannot invent a parent, but can name a real shallower one. Residue #2 in `173-RESIDUE.md`. | CLOSED (accepted, bounded) |
| T-173-08 | Tampering | mitigate | `latestSpawnEntryByName` returns nil on parse error (`cmd/spawn.go:230-237`), routing into `deriveSpawnDepth`'s deny branch. `TestSpawnLogRefusesAnUnknownParentAndWritesNothing` present (`cmd/spawn_enforce_test.go:382`). | CLOSED |
| T-173-09 | Denial of Service | mitigate | Deny branch in `spawnLogCmd` returns before `RecordSpawn` and before the ceremony publish (`cmd/spawn.go:96-99`). Byte-identity asserted by the same test above. | CLOSED |
| T-173-10 | Repudiation | mitigate | Success payload carries `claimed_depth`, `depth`, `depth_source` (`cmd/spawn.go:136-138`). | CLOSED |
| T-173-11 | Spoofing | mitigate | `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent` present (`cmd/spawn_enforce_test.go:690`) — scans the live corpus rather than a frozen list. | CLOSED |

### Plan 173-03 — `spawn-can-spawn-swarm` fail-closed

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-12 | Elevation of Privilege | mitigate | `cmd/internal_cmds.go:551-560`: colony-state load error now returns `outputError` with `can_spawn:false`, `reason:"state-unreadable"`; `"No state = no spawns"` comment removed (confirmed via `grep`). Test: `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableColonyState`. | CLOSED |
| T-173-13 | Elevation of Privilege | mitigate | Spawn-tree read moved to `agent.NewSpawnTree(...).Parse()` (`cmd/internal_cmds.go:598-608`); parse error denies. Test: `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree`. | CLOSED |
| T-173-14 | Denial of Service | mitigate | Negative control `TestSpawnCanSpawnSwarmAllowsWhenStateAndTreeAreBothReadable` present (`cmd/spawn_failclosed_test.go:173`), executed and PASS in this audit. | CLOSED |
| T-173-15 | Tampering | mitigate | Hand-rolled pipe parser deleted; counting now goes through `agent.SpawnTree.Parse()` (confirmed by code read). | CLOSED |

### Plan 173-04 — Depth cap + widened decision seam

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-16 | Elevation of Privilege | accept, bounded | `spawnCanSpawnCmd` without `--name` reports on a self-claimed depth (`DepthIsAuthoritative=false`); only `spawn-log` and `--name`-resolved calls are authoritative. Residue #1 in `173-RESIDUE.md`. Proven jointly by `TestSpawnCanSpawnNameOverridesClaimedDepth` + `TestSpawnLogRefusesPastCapAndWritesNoEntry`. | CLOSED (accepted, bounded) |
| T-173-17 | Elevation of Privilege | mitigate | `spawnCanSpawnDecision(...)` called from exactly two sites in `cmd/spawn.go` (`spawnLogCmd` line 106, `spawnCanSpawnCmd` line 357) — confirmed by `grep -n 'spawnCanSpawnDecision('`. | CLOSED |
| T-173-18 | Denial of Service | mitigate | Deny branch returns before `RecordSpawn`/ceremony publish in both call sites (code read, `cmd/spawn.go:96-99`, `113-116`). | CLOSED |
| T-173-19 | Tampering | mitigate | Seam widened in place (`var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult`, `cmd/spawn.go:289`); both new checks (`spawnTreeBudgetReason`, `spawnAncestorCycleReason`) called from within it. | CLOSED |
| T-173-20 | Denial of Service | mitigate | No `CONTRACT-STUB-PLAN-05`/`-06` markers survive: `TestNoDelegationGuardContractStubSurvives` executed in this audit, PASS. | CLOSED |
| T-173-21 | Repudiation | mitigate | `spawnDecisionResult.Detail` names requester, depth, and cap (`cmd/spawn.go:296-303`), embedded in `--enforce`'s error message. | CLOSED |

### Plan 173-05 — Whole-run tree budget (20)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-22 | Denial of Service | mitigate | `spawnTreeBudgetMax = 20` (`cmd/spawn_budget.go:24`), counts every helper via `spawnTreeBudgetState`; refusals write no entry (D-09, shared with T-173-09/18). | CLOSED |
| T-173-23 | Elevation of Privilege | mitigate | Every error path in `spawnTreeBudgetReason` returns non-empty deny (`cmd/spawn_budget.go:210-216`). `TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable` present (`cmd/spawn_budget_test.go:280`), has its own negative control. | CLOSED |
| T-173-24 | Denial of Service | mitigate | "No run yet" exception narrowed by plan 12 to exactly absent/zero-byte/whitespace ledger (`cmd/spawn_budget.go:67-91,113-120`) — strictly narrower/stronger than the original, still allows a genuinely fresh colony (`TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn`, executed, PASS). | CLOSED |
| T-173-25 | Tampering | accept, bounded | `filterEntriesForRun` windows by wall-clock timestamp, not a transactional link. Residue #4 in `173-RESIDUE.md`. (Partially superseded in practice by plan 12/WR-08's whole-ledger floor, which narrows — not widens — this bound.) | CLOSED (accepted, bounded) |
| T-173-26 | Denial of Service | mitigate | `grep -rn 'queenSpawnBudget\|MaxWorkers' cmd/spawn_budget.go` returns nothing; `TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities` present (`cmd/spawn_budget_test.go:224`). | CLOSED |
| T-173-27 | Information Disclosure | mitigate | `spawnTreeBudgetCeilingToMidden` has exactly one call site (`cmd/spawn_budget.go:219`, inside the exhausted branch only). `TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` present (`cmd/spawn_budget_test.go:340`). | CLOSED |

### Plan 173-06 — Ancestor-cycle detection

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-28 | Denial of Service | mitigate | `spawnAncestorCycleReason` (`cmd/spawn_ancestor.go:93-120`) refuses a repeated caste+normalised-task pair, names the ancestor. `TestSpawnCanSpawnDeniesAncestorCycle` present. | CLOSED |
| T-173-29 | Denial of Service | mitigate | `spawnAncestorChain` (`cmd/spawn_ancestor.go:44-79`) uses a visited-name set, terminates on repeat; `st.Parse()` called once per invocation. | CLOSED |
| T-173-30 | Elevation of Privilege | mitigate | Error path returns non-empty deny (`cmd/spawn_ancestor.go:100-104`); `TestSpawnAncestorCheckFailsClosedOnUnreadableTree` present (`cmd/spawn_ancestor_test.go:202`), has its own negative control. | CLOSED |
| T-173-31 | Denial of Service | mitigate | `normalizeSpawnTask` applies exactly 4 rules (`cmd/spawn_ancestor.go:24-27`); `TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop` present (`cmd/spawn_ancestor_test.go:164`); false-positive controls `TestSpawnAncestorCheckAllowsDifferentTaskSameCaste`/`SameTaskDifferentCaste` exist. | CLOSED |
| T-173-32 | Tampering | mitigate | `sanitizeSpawnField` strips `\|`, CR, LF on write (`pkg/agent/spawn_tree.go`, confirmed referenced in plan interfaces and unchanged). | CLOSED |
| T-173-33 | Elevation of Privilege | accept, bounded | Textual match, not semantic — reworded identical work is not caught. Residue #5 in `173-RESIDUE.md`. | CLOSED (accepted, bounded) |

### Plan 173-07 — PreToolUse hook fail-closed deny path

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-34 | Elevation of Privilege | mitigate | `hookSpawnDenyReason`'s unresolved branch returns non-empty (`cmd/hook_cmds.go:196-201`). `TestHookPreToolUseDeniesUnresolvedRequesterDepth` present and executed (via combined `TestHookPreToolUse*` run), PASS. | CLOSED |
| T-173-35 | Spoofing | accept, bounded | Platform fields unauthenticated, no identity bridge. Residue #3 in `173-RESIDUE.md`, quoting `173-HOOK-FINDINGS.md`'s dated verdict. Authoritative enforcement remains `spawn-log`. | CLOSED (accepted, bounded) |
| T-173-36 | Denial of Service | mitigate | Absent identifier = main session, allowed with comment (`cmd/hook_cmds.go:172-178`). `TestHookPreToolUseAllowsMainSessionDispatch` / `AllowsFirstTierWorkerDispatch` present, executed via combined run, PASS. | CLOSED |
| T-173-37 | Tampering | mitigate | `grep -n 'spawnMaxDelegationDepth' cmd/hook_cmds.go` shows shared constant in use; no second cap literal. | CLOSED |
| T-173-38 | Repudiation | mitigate | `tracer.LogIntervention(..., "hook.pre-tool-use.spawn-deny", ...)` at `cmd/hook_cmds.go:109`, called before `emitHookBlock`. | CLOSED |
| T-173-39 | Elevation of Privilege | accept, bounded | Hook coverage determined empirically (`173-HOOK-FINDINGS.md`), bounded and dated. `spawn-log` enforces the cap regardless of hook coverage. Residue #3 in `173-RESIDUE.md`. | CLOSED (accepted, bounded) |

### Plan 173-08 — Live tree rendering (`spawn-tree-active`)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-40 | Tampering | mitigate | No write/create/publish/update on `spawnTreeActiveCmd`'s path (code read, `cmd/spawn.go:410-465`, `480-529`). `TestSpawnTreeActiveMutatesNothing` present, executed via combined `TestSpawnTreeActive` run, PASS. | CLOSED |
| T-173-41 | Denial of Service | mitigate | Reads through `Active()`/`Store.ReadFile` (shared read lock); no new locking scheme introduced. | CLOSED |
| T-173-42 | Information Disclosure | accept | Task text already stored plaintext and already rendered by existing ceremony output — no new exposure surface (plan 08's own threat-model text). | CLOSED (accepted) |
| T-173-43 | Tampering | mitigate | `sanitizeSpawnField` strips CR/LF/pipe on write; `colorizeCaste` emits only fixed, known ANSI codes (`cmd/codex_visuals.go:3848`). | CLOSED |
| T-173-44 | Repudiation | mitigate | `spawnTreeActiveParentPhrase` (`cmd/spawn.go:537-542`) still names a completed/non-active parent by its recorded string rather than dropping it. | CLOSED |

### Plan 173-09 — Orphan reaping

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-45 | Denial of Service | mitigate | `SpawnStatusAbandoned` excluded from consumed budget (`cmd/spawn_budget.go:124`,`143`,`171`); reaper auto-runs at `beginRuntimeSpawnRun` (`cmd/spawn_runs.go:31`, before any budget decision). `TestSpawnReapReleasesBudgetForStaleEntryOnly` present. | CLOSED |
| T-173-46 | Denial of Service | mitigate | `spawnReapEntryLastActivity` uses the LATER of Timestamp/ActivityTimestamp (`cmd/spawn_reap.go:49-65`); default threshold 120 min. `TestSpawnReapNeverTouchesAnEntryInsideTheThreshold` / `RespectsRefreshedActivity` present. | CLOSED |
| T-173-47 | Tampering | mitigate | `UpdateStatusPreserveActivity` used exclusively in `spawnReapStaleEntries` (`cmd/spawn_reap.go:124`); `grep -n 'UpdateStatus('` inside that function shows none. | CLOSED |
| T-173-48 | Tampering | mitigate | `spawn-orphans` without `--clear` calls `spawnReapCandidates` only (read-only, `cmd/spawn_reap.go:244-262`). `TestSpawnOrphansListingMutatesNothing` present. | CLOSED |
| T-173-49 | Elevation of Privilege | mitigate | `spawnReapThresholdMinutes` rejects `<1` and falls back to default (`cmd/spawn_reap.go:36-38`); unreadable state also falls back rather than reaping on a guess. | CLOSED |
| T-173-50 (plan 09) | Repudiation | mitigate | Each reap writes a plain-English summary (`cmd/spawn_reap.go:120-123`); `spawn-orphans --clear` reports names + budget before/after (`renderSpawnOrphansClearText`). | CLOSED |
| T-173-51 (plan 09) | Spoofing | mitigate | No liveness signal exists; comments/output consistently avoid "detects abandonment" language (confirmed by code read, e.g. `cmd/spawn_reap.go:15-19`, `99-108`). Residue #6 in `173-RESIDUE.md`. | CLOSED |

### Plan 173-10 — CI ratchet, cross-guard proof, residue, ROADMAP correction

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-52 (plan 10) | Tampering | mitigate | 5 new guard files added to `wiringGateGuardFiles` (`cmd/ci_wiring_gate_test.go:87-99`); `TestWiringGateStepRunsEveryWiringTest` fails in both directions. Executed in this audit via full-filter CI runs, PASS. | CLOSED |
| T-173-53 (plan 10) | Elevation of Privilege | mitigate | `TestNoDelegationGuardContractStubSurvives` executed in this audit, PASS. | CLOSED |
| T-173-54 (plan 10) | Elevation of Privilege | mitigate | `TestDelegationGuardTableCoversEveryGuardCommand` enumerates registered commands via `rootCmd.Commands()`, not a hardcoded list. Executed in this audit, PASS. | CLOSED |
| T-173-55 (plan 10) | Repudiation | mitigate | `cmd/testdata/orphan_allowlist.json` diff is removals-only (confirmed: zero removals earned this phase, `git diff` shows no change — verified as a deliberate, tested outcome, not an addition). | CLOSED |
| T-173-56 (plan 10) | Spoofing | mitigate | `173-RESIDUE.md` names residues in bounded language (confirmed present, read in full). | CLOSED |
| T-173-57 (plan 10) | Tampering | mitigate | Blanket `go test ./...` release-gate step verified byte-identical to pre-phase state via `blanketReleaseGateProblem`/`blanketGateRunCommand` (`cmd/ci_wiring_gate_test.go:34-71`), confirmed present and unmodified. | CLOSED |
| T-173-58 (plan 10) | Elevation of Privilege | mitigate | `TestDelegationGuardTableCoversEveryGuardCommand` fails on zero/only-non-exercisable axes (code read + PASS execution). | CLOSED |
| T-173-59 (plan 10) | Repudiation | mitigate | Same test AST-parses `cmd/spawn.go`, `cmd/spawn_budget.go`, `cmd/spawn_ancestor.go` for a `COLONY_STATE.json` string literal (excludes comments); executed, PASS — confirms no guard has since gained a colony-state read. | CLOSED |
| T-173-60 (plan 10) | Tampering | mitigate | ROADMAP criterion 2 corrected to `2 for a three-level tree`; criterion 4 confirmed byte-identical to its "every" wording (both grepped and present in current `.planning/ROADMAP.md`). | CLOSED |

### Plan 173-11 — Corruption detection at the parse boundary (gap closure)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-50 (plan 11) | Tampering | mitigate | `parseFile`/`parseSpawnTreeBytes` wrap `ErrSpawnTreeCorrupt` (`pkg/agent/spawn_tree.go:16-23`,`367-481`). `TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError` executed, PASS. | CLOSED |
| T-173-51 (plan 11) | Tampering | mitigate | `RecordSpawn`/`updateStatus` write closures abort on parse error (code read + `TestSpawnTreeRefusesToRewriteACorruptLedger`, executed via `go test ./pkg/agent`, PASS). | CLOSED |
| T-173-52 (plan 11) | Denial of Service | mitigate | Absent/zero-byte/whitespace still parse to empty with nil error (`pkg/agent/spawn_tree.go:360-378`); `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn` executed, PASS. | CLOSED |
| T-173-53 (plan 11) | Information Disclosure | mitigate | Error strings name only line number + field count (`pkg/agent/spawn_tree.go:439`,`481`); `LEDGER-CONTENT-CANARY` non-leak assertion present in tests, and independently re-verified live by `173-VERIFICATION.md`'s manual reproduction. | CLOSED |
| T-173-54 (plan 11) | Elevation of Privilege | mitigate | `grep -rn 'AETHER_.*SPAWN_TREE\|SkipSpawnTreeValidation\|lenient' pkg/agent/spawn_tree.go` returns nothing (confirmed). | CLOSED |
| T-173-66 | Denial of Service | mitigate | Rule set accepts every shape `formatSpawnTreeLines` emits (writer round-trip test `TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces`, executed via `go test ./pkg/agent`, PASS). | CLOSED |
| T-173-67 | Tampering | accept | Shape-based classification cannot detect a well-formed but forged line. Residue #10 in `173-RESIDUE.md`, explicitly and repeatedly disclosed (also referenced by plans 12/13). | CLOSED (accepted) |
| T-173-69 | Tampering | mitigate | `loadRunStateLocked` returns an error for an obstructed/unreadable `spawn-runs.json`, distinct from absent (`pkg/agent/spawn_tree.go:634-645`). `TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne` executed, PASS. | CLOSED |

### Plan 173-12 — Budget denies on unreadable ledger/run-record (gap closure)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-55 (plan 12) | Elevation of Privilege | mitigate | `st.Parse()` called before `CurrentRun()` in `spawnTreeBudgetState` (`cmd/spawn_budget.go:100-112`, integrity-first ordering confirmed by line order). | CLOSED |
| T-173-56 (plan 12) | Tampering | mitigate | Refusal leaves tampered bytes untouched — confirmed both by code (deny returns before any write) and by `173-VERIFICATION.md`'s live manual reproduction (`cat spawn-tree.txt` shows garbage preserved). | CLOSED |
| T-173-57 (plan 12) | Information Disclosure | mitigate | Deny message names budget + file, never content; `LEDGER-CONTENT-CANARY` non-leak assertion in `cmd/spawn_budget_test.go`. | CLOSED |
| T-173-58 (plan 12) | Repudiation | mitigate | `spawnTreeBudgetCeilingToMidden` still the sole midden call site, reached only on genuine exhaustion (D-11 preserved through the gap-closure edit). | CLOSED |
| T-173-59 (plan 12) | Denial of Service | mitigate | `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn` executed in this audit, PASS (three states: no file, zero-byte, no-run-small-ledger). | CLOSED |
| T-173-60 (plan 12) | Tampering | mitigate | All 3 new plan-12 tests registered in `.github/workflows/ci.yml`'s `-run` filter (confirmed via `grep`), plus the later plan-13/WR-08 additions. | CLOSED |
| T-173-68 | Repudiation | mitigate | `requester-not-recorded` axis re-pointed at a readable-but-incomplete ledger (`cmd/spawn_failclosed_test.go`), confirmed via executed subtest `spawn-can-spawn/requester-not-recorded`, PASS, still asserting `ancestor chain unreadable`. | CLOSED |
| T-173-70 | Elevation of Privilege | mitigate | Whole-ledger counting when no run resolves against a non-empty ledger (`cmd/spawn_budget.go:113-133`). `TestErasingTheRunRecordDoesNotResetTheWholeRunBudget` executed, PASS. | CLOSED |
| T-173-71 | Denial of Service | mitigate | No-run case counted, not refused (code read); third case of `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn` executed, PASS. | CLOSED |

### Plan 173-13 — Cross-guard proof breadth + stale-claim correction (gap closure)

| Threat ID | Category | Disposition | Evidence | Status |
|---|---|---|---|---|
| T-173-61 | Repudiation | mitigate | Repo-wide grep for the 6 stale phrasings (`never return a non-nil error`, `cannot itself be made to return`, `can never itself return`, `cannot be made to error`, `cannot tell the two apart`, `swallows a missing file`, `skipped entirely by parseSpawnTreeBytes`) returns nothing — re-executed in this audit (`grep -rn --include="*.go"`), confirmed empty. | CLOSED |
| T-173-62 | Tampering | mitigate | 3 `ledger-corrupt` axes present in `delegationGuardFaultTable` (`spawn-can-spawn`, `spawn-log`, `spawn-can-spawn-swarm`); all 3 subtests executed in this audit, PASS. | CLOSED |
| T-173-63 | Repudiation | mitigate | `173-RESIDUE.md` records exploit, second route, closure, and un-closed limits (read in full; residues #10-13 present). | CLOSED |
| T-173-64 | Elevation of Privilege | accept | `recordCodexBuildDispatches` bypasses `spawnCanSpawnDecision` — named as residue #11 in `173-RESIDUE.md`, not fixed (explicitly out of gap-closure scope). Spawns via this path are still counted once recorded. | CLOSED (accepted) |
| T-173-65 | Denial of Service | accept | `spawn-tree-active` goes silent (not error) on a tampered ledger — named as residue #12 in `173-RESIDUE.md`. Enforcement path unaffected. | CLOSED (accepted) |
| T-173-72 | Tampering | mitigate | `run-state-obstructed` axis added to `spawn-log` entry (`cmd/spawn_failclosed_test.go`); executed subtest `spawn-log/run-state-obstructed`, PASS. | CLOSED |

---

## Post-Plan Events (surfaced by code review, `173-REVIEW.md`)

These are not part of the 83-row plan-authored register but were found during this session's code
review and are fixed, tested, and locked — verified in this audit by execution, not by reading the
review report's claim.

| ID | Description | Fix | Lock (verified) |
|---|---|---|---|
| CR-01 | A briefly-lived sentinel-file fallback (`~/.aether/hook-capture-path`) let a worker's own Write tool redirect hook-payload capture, reopening T-173-03's boundary (an operator-only switch). | Removed same day (commit `833719da`); `AETHER_HOOK_CAPTURE_FILE` is now the only switch. | `TestHookCaptureHasNoFileBasedSwitch` (`cmd/hook_cmds_test.go:349`) — **executed in this audit, PASS.** |
| WR-08 | The whole-ledger safety net (T-173-70/71) covered only the "no run resolves" case; a run record naming an **ended** run with a stale window still let a full, valid 25-entry ledger report `budget_consumed:0`. Reproduced live by the code reviewer against a freshly built binary. | Non-active run's window is now untrusted (whole-ledger count applies); an active run's live entries outside its window raise the count to a whole-ledger floor (`cmd/spawn_budget.go:149-196`, commit `1f9ca5a1`). | `TestAnEndedRunRecordDoesNotResetTheWholeRunBudget` (`cmd/spawn_budget_test.go:748`) — **executed in this audit, PASS**, and confirmed registered in `.github/workflows/ci.yml`'s `-run` filter. |

---

## Accepted Risks Log

*Accepted risks do not resurface in future audit runs unless a new plan reopens them.*

| Risk ID | Threat Ref | Rationale | Documented In | Accepted By | Date |
|---|---|---|---|---|---|
| AR-01 | T-173-03 | Operator sets `AETHER_HOOK_CAPTURE_FILE` in their own shell; no privilege boundary is crossed by an operator who can already write the file directly. | Plan `173-01` `<threat_model>`; regression-locked by CR-01 fix (`TestHookCaptureHasNoFileBasedSwitch`). | Plan 173-01 threat model | 2026-08-13 |
| AR-02 | T-173-05 | Plan 01's new `Agent\|Task` matcher deliberately observes only; enforcement is plan 07's explicit responsibility (`depends_on`). | Plan `173-01` objective and `<threat_model>`. | Plan 173-01 threat model | 2026-08-13 |
| AR-03 | T-173-07 | Deriving depth from a recorded parent closes "invent a parent"; it does not close "name a real, shallower ancestor." Identity verification of the caller is out of scope. | `173-RESIDUE.md` residue #2; plan `173-02-SUMMARY.md`. | Plan 173-02 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-04 | T-173-16 | `spawn-can-spawn` without `--name` is advisory by design (`spawnDecisionInput.DepthIsAuthoritative`); only `spawn-log` (and `--name`-resolved calls) are authoritative. | `173-RESIDUE.md` residue #1; `cmd/spawn.go` doc comment on `spawnDecisionInput`. | Plan 173-04 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-05 | T-173-25 | Run membership is a wall-clock activity window, not a transactional link; two runs starting within the same second, or clock skew, can misattribute a spawn. | `173-RESIDUE.md` residue #4. | Plan 173-05 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-06 | T-173-33 | Ancestor-cycle matching is textual after normalisation, not semantic; reworded but identical work is not caught. | `173-RESIDUE.md` residue #5. | Plan 173-06 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-07 | T-173-35 | Platform-supplied `agent_id`/`agent_type` fields are not authenticated; an agent could present a first-tier subagent type it is not. Authoritative enforcement remains `spawn-log`. | `173-RESIDUE.md` residue #3, quoting `173-HOOK-FINDINGS.md`'s dated verdict. | Plan 173-07 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-08 | T-173-39 | Hook coverage (which dispatch levels the `PreToolUse` hook fires for) is bounded to what `173-HOOK-FINDINGS.md` empirically observed on 2026-08-13; `spawn-log` enforces the depth cap regardless. | `173-RESIDUE.md` residue #3. | Plan 173-07 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-09 | T-173-42 | Worker task text is already stored in plaintext in `spawn-tree.txt` and already rendered by existing ceremony output; the new live-tree view adds no new exposure surface. | Plan `173-08` `<threat_model>`. | Plan 173-08 threat model | 2026-08-13 |
| AR-10 | T-173-67 | Shape-based ledger classification cannot detect a well-formed but forged line; closing this needs integrity data (append-only log or signature) the file format does not carry — a storage-format change, out of scope for this phase. | `173-RESIDUE.md` residue #10; first named in `173-11-SUMMARY.md`. | Plan 173-11 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-11 | T-173-64 | `recordCodexBuildDispatches` and sibling direct recorders (build-finalize, plan, colonize, continue paths) call `RecordSpawn` directly and never consult `spawnCanSpawnDecision`; their spawns are counted once recorded but never asked permission first. Routing every one of those call sites through the decision chokepoint is a behaviour change to the build path, out of scope for a gap-closure plan. | `173-RESIDUE.md` residue #11. | Plan 173-13 threat model / 173-RESIDUE.md | 2026-08-13 |
| AR-12 | T-173-65 | `spawn-tree-active` renders no header/helpers/error on a tampered ledger — enforcement fails closed, the operator's live view fails silent instead of naming the problem. Fixing this is view work (a plain-English line), not a guard change. | `173-RESIDUE.md` residue #12. | Plan 173-13 threat model / 173-RESIDUE.md | 2026-08-13 |

*Also carried in `173-RESIDUE.md` but not tied to a specific threat-model row (structural bounds, not
STRIDE threats): residue #6 (no liveness signal — reaper detects elapsed time, not abandonment, tied
to T-173-51/plan 09), residue #8 (refusal reason proven in runtime output, not proven in
`/ant-build`/`/ant-continue` ceremony narration — open human-verification item, see below), residue
#9 (allowlist earned zero removals, with reasons), residue #13 (concurrent check-then-act race,
WR-05, open — see Known Non-Blocking Findings below).*

---

## Known Non-Blocking Findings (carried forward from `173-REVIEW.md`, not part of the phase's declared threat register)

These were surfaced by code review, not by the plans' own `<threat_model>` blocks, and every
`SUMMARY.md` `## Threat Flags` section for this phase (plans 07, 10, 11 — the only three that
include the section) reports **"None"** — no plan flagged new, unmapped attack surface during its
own implementation. The items below are pre-existing/adjacent code-quality and defense-in-depth
observations disclosed by the human-facing code review, already tracked in `173-REVIEW.md`, and are
reported here for completeness per the adversarial-audit stance. None of them contradict a CLOSED
row above; none are declared threats this audit could mark OPEN.

| ID | Summary | Status |
|---|---|---|
| WR-01 | `.aether/workers.md`'s child-prompt template (Step 4 "SPAWN CAPABILITY" block) still states the pre-phase three-level convention (`Depth 1→4, Depth 2→2, Depth 3→0`), contradicting the two-level convention this phase enforces at runtime. Runtime still denies correctly (fail-closed holds) — this is a documentation drift, not a guard failure. | Open, disclosed, non-blocking |
| WR-02 | `.aether/workers.md` documents a `spawn-can-spawn` return shape (`max_spawns`, `current_total`) the runtime does not produce. | Open, disclosed, non-blocking |
| WR-03 | The advisory `spawn-can-spawn` (no `--enforce`) still writes a midden entry when polled at the budget ceiling, via the same `spawnTreeBudgetReason`→`spawnTreeBudgetCeilingToMidden` path spawn-log's genuine refusal uses. This does not contradict T-173-27's declared claim (midden write is reachable *only* from the budget-exhausted branch, which remains true), but it is a state mutation from what CLAUDE.md's Definition of Done calls an inspection-style command. No task text leaks (message contains only requester name and counts). | Open, disclosed, non-blocking |
| WR-04 | The hook's `agent_type` heuristic (T-173-35/39's accepted bound) fails both ways in specific scenarios: an `aether-*`-typed requester dispatching another `aether-*`-typed agent under-blocks; a non-`aether-*` subagent legitimately dispatching its own helper over-blocks. Already named as accepted/bounded residue. | Open, disclosed, non-blocking |
| WR-05 | `spawn-log`'s guard decision and `RecordSpawn` are not atomic; two concurrent processes can both pass the budget check at `Consumed==19` and both record, exceeding 20. Named explicitly as still-open in `173-RESIDUE.md` residue #13. | Open, disclosed, non-blocking |
| WR-06 | `.aether/commands/patrol.yaml` (the YAML source) does not document the `spawn-orphans` step that 3 wrapper markdown files carry — spec/wrapper drift, not a guard defect. | Open, disclosed, non-blocking |
| WR-07 | `spawnAncestorChain` silently truncates (returns an incomplete-but-allowed chain) on a *well-formed* ledger missing a mid-chain ancestor, versus denying on an unresolvable *start* name. A corrupt mid-chain line now denies the whole walk (post-plan-11), but a merely-incomplete one does not. This is adjacent to, but distinct from, T-173-29/30 (which are about corrupted/unreadable input, both CLOSED). | Open, disclosed, non-blocking |
| IN-01 – IN-06 | Minor: arithmetic phrasing, partial-failure reporting, midden ID collision risk, an unused struct field, a test walking gitignored data, a bare-newline capture edge case. None security-relevant beyond what is already named above. | Open, disclosed, non-blocking |

---

## Human Verification Required (carried from `173-VERIFICATION.md`, unaffected by this audit)

| Item | Why it is not closeable by this audit | Status |
|---|---|---|
| Whether a refusal's plain-English reason (proven present in the Go runtime's own JSON/error output — `spawnDecisionResult.Detail`) actually renders inside the `/ant-build`/`/ant-continue` ceremony narration a non-technical operator reads, versus only in a raw CLI error a person would have to go looking for. | No plan in this phase touches `build.md`/`continue.md`; this is a runtime-vs-presentation question only a human watching a real build can answer, and it does not correspond to any STRIDE row in the threat register — it is `173-RESIDUE.md` residue #8. | Open — human observation needed, non-blocking for this audit |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-13 | 83 (+ 2 post-plan events: CR-01, WR-08) | 83 | 0 | Claude (gsd-security-auditor) |

**Verification method:** For each of the 83 threat rows, the declared mitigation/acceptance was
checked against the actual implementation — not against plan text or SUMMARY.md claims. For
`mitigate` rows, the named function, test, or CI registration was located by `grep`/`Read` in the
cited file and, wherever the row named a specific test, that test (or an equivalent aggregate `-run`
filter covering it) was **executed** in this session:

- `go build ./cmd/aether` — PASS
- `go vet ./...` — clean
- `go test ./cmd -run 'TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs|TestDelegationGuardTableCoversEveryGuardCommand|TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAnEndedRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn|TestHookCaptureHasNoFileBasedSwitch|TestNoDelegationGuardContractStubSurvives' -count=1 -v` — **all PASS**, including all 16 cross-guard subtests (`ledger-corrupt` ×3, `run-state-obstructed` ×1, plus the pre-existing axes and the all-valid control)
- `go test ./cmd -run 'TestHookPreToolUse|TestSpawn|TestBudget|TestNormalizeSpawnTask|TestWiringGate|TestNoDelegationGuardContractStubSurvives|TestOrphanAllowlist|TestNoRegisteredSubcommandIsUnreferenced' -count=1` — PASS
- `go test ./pkg/agent -run 'TestSpawnTree|TestSpawnRunState' -count=1` — PASS
- Repo-wide grep for the 6 stale "impossible" phrasings named in T-173-61 — confirmed empty
- `git log --oneline` confirmed both post-plan fix commits exist: `833719da` (CR-01) and `1f9ca5a1` (WR-08)

For `accept`/`accept, bounded` rows, the acceptance was verified present either in the originating
plan's own `<threat_model>` mitigation-plan text, or in `173-RESIDUE.md`'s eleven-item "What this
phase does NOT prove" section (residues #1–#13, all read in full) — both are locations this audit's
governing instructions name as acceptable acceptance documentation.

No implementation file was modified by this audit.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer) — no `transfer` rows exist in this phase
- [x] Accepted risks documented in Accepted Risks Log (12 rows, AR-01 through AR-12)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-13
