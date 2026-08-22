---
phase: 191-dead-wood
verified: 2026-08-21T09:29:39Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 2
overrides:
  - must_have: "Dead code removed (pkg/trace/cost.go + unconstructed pool path, session-verify-fresh, newLearningValidator, unreferenced skill-lifecycle commands) — ROADMAP criterion 3"
    reason: "session-verify-fresh has a live, current caller in the /ant-medic wrapper on all three platforms (.claude/commands/ant/medic.md:28, .claude/commands/ant-medic.md:28, .opencode/commands/ant/medic.md:28), .aether/commands/medic.yaml, and .aether/skills/colony/context-management/SKILL.md:33 — deleting it would break /ant-medic. The ROADMAP text predates that wiring (CHANGELOG.md:191 documents the wiring landing separately). This is the phase's own D-07 correction (191-CONTEXT.md), not an unauthorized deviation. Independently re-verified by the verifier: sessionVerifyFreshCmd is still registered (cmd/session_cmds.go:361,636), and `go run ./cmd/aether session-verify-fresh --command oracle` was executed live and returned a correct, well-formed JSON result."
    accepted_by: "gsd-verifier"
    accepted_at: "2026-08-21T09:29:39Z"
  - must_have: "191-02-PLAN.md must_have: every field colony-prime.md or dispatch-contract.yaml defined that diverged from its Go compiled default has been folded into that default before deletion"
    reason: "2 of dispatch-contract.yaml's 13 modeled fields (fallback_behaviors.planning, fallback_visibility.planning) were found to diverge from the Go compiled default and were deliberately NOT folded, because the divergence ran the other way: 3 independent pre-existing tests (TestPlanIncludesDispatchContract, TestPlanVisualOutputShowsDispatchContractDetails, TestGoldenPlanVisualOutput against a committed golden fixture) proved the Go default is the current, tested, fail-closed, shipped design and the YAML text was itself stale ('synthesize planning artifacts locally' — a real regression to a weaker, non-fail-closed behavior). Folding the stale YAML text, as the literal instruction says, would have silently shipped that regression to every installed Aether user. The plan's own execution caught this via a failing test run, reverted the accidental fold, and rewrote the regression test to assert byte-identity for every field except these two, with the two checked against the tested fail-closed text instead. Independently re-verified by the verifier: both fallback constants (cmd/codex_dispatch_contract.go:93,106) currently hold the fail-closed text, not the stale YAML text; all 3 dependent tests plus TestDispatchContractYamlDeletionProducesByteIdenticalOutput were re-run live and pass."
    accepted_by: "gsd-verifier"
    accepted_at: "2026-08-21T09:29:39Z"
---

# Phase 191: Dead Wood Verification Report

**Phase Goal:** Nothing remains that claims to define behaviour while defining nothing. Delivers ROSTER-01/02 and SKILL-01 as rulings-by-deletion.
**Verified:** 2026-08-21T09:29:39Z
**Status:** passed
**Re-verification:** No — initial verification

## Environment Note (read before the evidence below)

This worktree (`agent-adb5037b9ffa84dc9`) was, at verification start, checked out at commit `6577f51c` (an unrelated, much older point in history — the merge-base with `main`), with no phase-191 code or planning artifacts present. Phase 191's actual committed work (`916db8fa` "begin phase" through `fe084409` "update tracking after final verification plan", 30 commits) lives on a separate line not yet merged to `main`, checked out in a different, locked sibling worktree. To perform this verification I created a new local branch, `phase-191-verification`, from `fe084409` inside this worktree (`git checkout -b phase-191-verification fe084409` — a non-destructive operation; the prior branch tip was itself an ancestor of `main` with no unique work) and did all verification there. This is disclosed for auditability; it does not affect the findings below, all of which were produced by direct execution against the phase's real, committed code.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Zero-reader configs deleted (39 files: `colony/agents/*.yaml` ×27, `colony/phases/*.yaml` ×9, `model-routing.yaml`, `autopilot.yaml`, `memory-rules.yaml`) with grep-ratchets against reappearance | VERIFIED | DEMONSTRATED. `colony/agents/` and `colony/phases/` directories confirmed absent; `colony/policies/` listing confirmed the 3 named files absent (5 files remain: the protected file + 4 out-of-scope D-02 keeps). Restored `colony/agents/queen.yaml` from git history (`916db8fa`) and re-ran `TestZeroReaderColonyConfigsDoNotReappear` myself: it FAILED, naming exactly that file and citing ROSTER-01/02. Deleted it again, `git status --short colony/` empty, re-ran: PASS. `TestZeroReaderColonyConfigsListIsComplete` (the anti-vacuous-guard: a hardcoded, shape-checked list rather than a directory scan, since the source dirs no longer exist) also passes. |
| 2 | The four CWD-relative silent-fallback loaders resolved: diffed, folded, deleted — dev-checkout output byte-identical before/after | VERIFIED (2 fields via override — see frontmatter) | DEMONSTRATED for two loaders independently, in full: **review-depth.yaml** — every value in the frozen original YAML (`cmd/testdata/review_depth_original_191_04.yaml`) manually diffed by the verifier against `cmd/review_depth.go`'s compiled fallback slices/constants: exact match, item-for-item, for all 12 heavy keywords, 10 security-risk keywords, 10 blast-radius keywords, and 6 smart-default-reason strings — nothing needed folding. **visuals.md** — independently re-ran `TestVisualsConfigOriginalFileHadPreexistingParseDefect` (proves the original file never actually parsed, due to a pre-existing YAML indentation defect — it was always inert), `TestVisualsConfigFoldedDefaultsMatchOriginalFile` (file-present vs file-absent renders byte-identical) and `TestVisualsConfigOriginalFileValuesMatchCompiledDefaults` (a hand-repaired parseable copy proves the values would have matched anyway): all 3 PASS. **colony-prime.md + dispatch-contract.yaml** — `TestColonyPrimeAndDispatchContractDoNotReappear` and `TestDispatchContractYamlDeletionProducesByteIdenticalOutput` re-run and PASS; the 2 deliberately-unfolded fields independently confirmed (see override #2). |
| 3 | Dead code removed (`pkg/trace/cost.go` + unconstructed pool path, `newLearningValidator`, unreferenced skill-lifecycle commands) — `session-verify-fresh` correctly KEPT, not deleted, per the phase's own D-07 correction of stale ROADMAP text | VERIFIED (session-verify-fresh via override — see frontmatter) | DEMONSTRATED. `pkg/trace/cost.go` absent; `go build ./...` succeeds repo-wide; `grep -rn "NewPool(" cmd/` returns zero (confirms the "unconstructed pool path" claim independently). `newLearningValidator`: zero matches anywhere in the repo. The 8 skill-lifecycle CLI commands (`skill-index/-detect/-match/-inject/-list/-diff/-parse-frontmatter/-cache-rebuild`): zero `Use:` string matches in `cmd/skills.go`, and `init()` no longer registers any of them. `matchSkillsForWorkflow`/`resolveSkillMatchInput`/`renderSkillInjectResult` still exist unmodified and are traced live into the real interactive build path (see Key Link Verification). `session-verify-fresh`: still registered (`cmd/session_cmds.go:361,636`), still referenced by the medic wrapper on all 3 platforms + `SKILL.md`, and **executed live** by the verifier (`go run ./cmd/aether session-verify-fresh --command oracle` → correct JSON). |
| 4 | False docs corrected (`workers.md:825`, CLAUDE.md trim-order and host-build claims) | VERIFIED | DEMONSTRATED. `.aether/workers.md:825-830` now names `pkg/learn`'s `captureContinueLearning()` at `cmd/codex_continue_finalize.go:1458` called from `cmd/codex_continue.go:962` and `cmd/codex_continue_finalize.go:521` — the verifier independently grepped all three exact line numbers and function signatures: all match exactly. CLAUDE.md's Trim order list (9 items + "blockers never trimmed") matches `cmd/context.go:1027-1038`'s real `trimOrder` slice verbatim, including the QUEEN wisdom global/local split. CLAUDE.md's Command Playbooks section now describes two distinct flows (interactive wrapper vs. autopilot/host-driven); independently cross-checked against `.claude/commands/ant/build.md:348` ("Do NOT run `aether host build` from this wrapper; the TS host hop is off the interactive build path") — exact match. |
| 5 | `oracle-phase-directives.yaml` and all auxiliary commands untouched — `/ant-oracle` and `/ant-dream` smoke-pass | VERIFIED | DEMONSTRATED. `git diff 916db8fa..HEAD -- colony/policies/oracle-phase-directives.yaml` is empty (byte-identical). `git diff --stat 916db8fa..HEAD` across all 18 auxiliary wrapper/agent files (oracle, dream, chaos, archaeology, swarm, council — Claude nested + flat-mirror, OpenCode, Codex TOML) is empty — zero touched. Verifier **independently executed** `go run ./cmd/aether oracle selftest`: real result, `"passed":9,"failed":0"` (dispatcher, research round, artifacts written, findings recorded, etc. — a real spawned research round, not a `--help` check). Verifier **independently executed** the dream wrapper's one Go touch point (`aether activity-log --command "DREAM" ...`) in an isolated scratch colony outside this worktree: produced a correctly-structured JSONL entry. |
| 6 | Live skill matching/injection (`matchSkillsForWorkflow`) remains wired into the real build path after the CLI-surface deletion — this is the phase's highest-risk edit | VERIFIED | DEMONSTRATED via direct call-chain trace (not the SUMMARY's claim alone): `resolveSkillSectionResultForWorkflow` (`cmd/codex_build.go:3267-3273`) calls `matchSkillsForWorkflow` at line 3268. That function is called by `attachBuildDispatchContext` (line 3299-3312, sets `dispatches[i].SkillSection` for every dispatch), which is called by `runCodexBuildPlanOnlyWithOptions` (line 207/281) — the real `aether build $ARGUMENTS --plan-only` entrypoint the interactive wrapper uses. `dispatches[i].SkillSection` is a manifest field (`json:"skill_section,omitempty"`) that reaches the wrapper. (Minor note: 191-CONTEXT.md's own shorthand attributed the line-3268 call site to `composeBuildManifestBrief` — it is actually one helper-layer removed, inside `resolveSkillSectionResultForWorkflow`; the wiring itself is real and verified, only the planning doc's naming was imprecise.) |
| 7 | No regression introduced — full test suite green, including race detection, for every package this phase touched | VERIFIED | DEMONSTRATED. `go build ./...` and `go vet ./...`: clean, zero output. `go test ./cmd/... -count=1`: ok (379s). `go test ./cmd/... -race -count=1`: ok (418s, exit 0). `go test ./pkg/agent/... ./pkg/trace/... ./pkg/codex/... -race -count=1`: ok, all 4 packages. See "Anti-Patterns" section for one transient flake observed on an early truncated run, not reproduced on any subsequent clean run. |

**Score:** 7/7 truths verified (5 outright, 2 via documented override — see frontmatter)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/colony_zero_reader_config_ratchet_test.go` | Existence-check ratchet for the 39 deleted zero-reader configs | VERIFIED | `TestZeroReaderColonyConfigsDoNotReappear` + `TestZeroReaderColonyConfigsListIsComplete` both pass; fail-then-pass independently demonstrated by the verifier (see Truth 1) |
| `cmd/colony_prime_and_dispatch_contract_ratchet_test.go` | Reappearance ratchet for `colony-prime.md` + `dispatch-contract.yaml` | VERIFIED | `TestColonyPrimeAndDispatchContractDoNotReappear` passes |
| `cmd/visuals_config_ratchet_test.go` | Reappearance ratchet for `visuals.md` | VERIFIED | `TestVisualsConfigDoesNotReappear` passes; byte-identity independently re-proven (Truth 2) |
| `cmd/review_depth_ratchet_test.go` | Reappearance ratchet for `review-depth.yaml` | VERIFIED | `TestReviewDepthPolicyDoesNotReappear` passes; values independently hand-diffed (Truth 2) |
| `cmd/skills.go` (`matchSkillsForWorkflow`, `resolveSkillMatchInput`, `renderSkillInjectResult`) | Preserved unmodified; 8 CLI wrappers removed | VERIFIED | Functions present at lines 560/544/831; `init()` (line 1163-1169) registers only `skillIndexReadCmd`/`skillManifestReadCmd`/`skillIsUserCreatedCmd` — none of the 8 dead ones |
| `cmd/subcommand_reachability_ratchet_test.go` (`assertSkillLifecycleCommandsStayDeleted`) | D-08 block rewritten to record the ruling-by-deletion | VERIFIED | Called from `TestNoRegisteredSubcommandIsUnreferenced` (line 1217); independently re-run, passes ("enumerated 398 registered commands, found 271 orphans") |
| `pkg/trace/cost.go` | Deleted | VERIFIED | Absent; `pkg/agent/pool.go`'s `OnComplete` no longer references it; `go build ./...` clean |
| `cmd/helpers.go` (`newLearningValidator`) | Deleted | VERIFIED | Zero matches repo-wide |
| `.aether/workers.md` | Corrected Wisdom Pipeline description at line ~825 | VERIFIED | Exact function/line/caller claims independently confirmed |
| `CLAUDE.md` | Corrected Trim order list + Command Playbooks host-build description | VERIFIED | Both independently cross-checked against `cmd/context.go` and `.claude/commands/ant/build.md` |
| `colony/policies/oracle-phase-directives.yaml` | Untouched, byte-identical | VERIFIED | `git diff 916db8fa..HEAD` empty |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `cmd/codex_build.go` `runCodexBuildPlanOnlyWithOptions` → `attachBuildDispatchContext` | `cmd/skills.go:560` `matchSkillsForWorkflow` | `resolveSkillSectionResultForWorkflow` (in-process call at `cmd/codex_build.go:3268`) | WIRED | Traced end-to-end by the verifier; `dispatches[i].SkillSection` is a manifest JSON field reaching the wrapper on the real `aether build --plan-only` path |
| `.claude/commands/ant/medic.md:28` (+ OpenCode + flat mirror) | `cmd/session_cmds.go` `sessionVerifyFreshCmd` | `aether session-verify-fresh --command <name>` | WIRED | Executed live by the verifier; real JSON result returned |
| `cmd/oracle_loop.go` `oraclePhaseDirectivePaths()` | `colony/policies/oracle-phase-directives.yaml` (+ repo-root + hub fallback) | unchanged 3-path search | WIRED, unchanged | Byte-diff empty; `aether oracle selftest` executed live, 9/9 |
| `cmd/install_cmd.go` `setupInstallHub` | `colony/policies/oracle-phase-directives.yaml` only | `isOraclePhaseDirectivesFile` basename filter (`cmd/platform_sync.go:362-364`) syncing to `hub/system/colony/policies/` | WIRED, correctly scoped | Destination path independently confirmed to be exactly `oraclePhaseDirectivePaths()`'s 3rd (hub) candidate — the phase's own 191-07 finding that `install_cmd.go` does reference `colony/` (contradicting 191-CONTEXT.md's D-13 grep-only claim) is real, but is a correctly-scoped, protected-file-only sync path, not a bug |

### Data-Flow Trace (Level 4)

Not applicable — this phase deletes dead files/code and corrects documentation; it does not add UI or dashboard components rendering dynamic data. The closest analog (skill-matching data flowing into worker briefs) is covered under Truth 6 / Key Link Verification above via direct call-chain trace, which is the appropriate check for this phase's shape.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Reappearance ratchet fires on a real re-introduced file, names it precisely | Restored `colony/agents/queen.yaml` from `916db8fa`, ran `go test ./cmd/... -run TestZeroReaderColonyConfigsDoNotReappear -v` | FAILED, naming `colony/agents/queen.yaml has reappeared -- ... ROSTER-01/ROSTER-02` | PASS (ratchet works as designed) |
| Same ratchet passes again once the restored file is removed | `rm colony/agents/queen.yaml && rmdir colony/agents`, re-ran the same test | PASS, `git status --short colony/` empty | PASS |
| `session-verify-fresh` executes for real | `go run ./cmd/aether session-verify-fresh --command oracle` | `{"ok":true,"result":{"command":"oracle",...}}` | PASS |
| `/ant-oracle` smoke-passes for real | `go run ./cmd/aether oracle selftest` | `"passed":9,"failed":0` (real spawned research round) | PASS |
| `/ant-dream`'s one Go touch point executes for real | `aether init` + `aether activity-log --command "DREAM" --details "..."` in an isolated scratch colony (outside this worktree) | Correctly-structured JSONL entry appended to `activity-log.jsonl` | PASS |
| `agent.NewPool` genuinely has zero `cmd/` callers | `grep -rn "NewPool(" cmd/` | zero matches | PASS |
| `skillMatchesWorkspace` (a D-05/deferred-items.md finding) genuinely has zero callers | `grep -rn "skillMatchesWorkspace(" cmd/` | 1 match (its own definition) | PASS (confirms the ledger entry is accurate) |
| Full build + vet, whole repo | `go build ./...`, `go vet ./...` | clean, zero output | PASS |
| Full `cmd` package suite | `go test ./cmd/... -count=1` | `ok  cmd  379.376s` | PASS |
| Full `cmd` package suite, race detector | `go test ./cmd/... -race -count=1` | `ok  cmd  417.954s`, exit 0 | PASS |
| Touched `pkg/` packages, race detector | `go test ./pkg/agent/... ./pkg/trace/... ./pkg/codex/... -race -count=1` | `ok` × 4 packages | PASS |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` convention exists in this repo (`find scripts -path '*/tests/probe-*.sh'` returns nothing), and neither the PLAN files nor SUMMARY files declare probe scripts for this phase. This is a code-deletion/doc-correction phase; verification was performed via direct `go test`/live-execution instead (see Behavioral Spot-Checks).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ROSTER-01 | 191-01 | `colony/agents/*.yaml` zero-reader contradiction resolved by explicit ruling (wired-or-deleted) | SATISFIED | Deleted with the compiled registry (27 `.claude/agents/ant/*.md` + `.opencode/agents/*.md` + `.codex/agents/*.toml`, none of which read `colony/agents/`) documented as authoritative; ratcheted and fail-then-pass proven by the verifier |
| ROSTER-02 | 191-01 | No file remains that claims to define agents and defines nothing, incl. `model-routing.yaml` | SATISFIED | `colony/agents/*.yaml`, `colony/phases/*.yaml`, and the 3 named `colony/policies/*.yaml` files all confirmed absent and ratcheted |
| SKILL-01 | 191-05 | Unreferenced skill-lifecycle commands (8 of 9) either have a caller or are removed | SATISFIED | The 8 confirmed-uncalled CLI wrappers removed; the 9th (`skill-create`) confirmed to have a real caller (`/ant-skill-create`) and correctly left alone; underlying matching logic confirmed still load-bearing (Truth 6) |

No orphaned requirements: `grep -n "Phase 191" .planning/REQUIREMENTS.md` returns exactly these 3 rows, and all 3 are claimed by a plan's `requirements:` frontmatter (191-01, 191-01, 191-05 respectively).

**Note:** REQUIREMENTS.md's own status column still reads "Pending" and the checkboxes are unchecked for all three as of this commit — this is expected pre-verification-closure bookkeeping (the same pattern as ROADMAP.md, whose Phase 191 progress row the phase's own last commit already updated to "7/7 Complete"; REQUIREMENTS.md's per-ID status flip is evidently a separate, later step outside this phase's own plans' declared scope). Not treated as a gap given the independently-verified evidence above; flagged here so the next workflow step updates it.

### Anti-Patterns Found

None. Scanned all 38 non-planning files changed in the phase's commit range (916db8fa..HEAD) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and for `placeholder|coming soon|will be here|not yet implemented|not available` (case-insensitive): the only hit is a documentation line in `.aether/docs/command-playbooks/build-context.md` literally describing a code-review scan step ("Note TODO/FIXME/HACK markers") — not a marker itself. No debt markers, no stub returns, no empty-implementation patterns introduced.

**One transparency note (Confirmation Bias Counter):** an early, self-inflicted-truncated (`| tail -150`) run of `go test ./cmd/...` by the verifier ended in `FAIL` with the specific failing test name cut off by the truncation. A full, untruncated re-run of the identical command immediately afterward passed cleanly (`ok`, 379s), as did a subsequent `-race` run (418s, exit 0). This matches 191-07-SUMMARY.md's own disclosed finding of one transient, non-reproducing flake in `pkg/codex` (`TestCodexReadOnlyProfileSelectsReadOnlySandbox`, confirmed unrelated to this phase's changes) — consistent with, though not itself proof of, the same flake. Disclosed per this repo's own "not reproduced = not a defect" house rule and in the interest of not silently omitting a FAIL the verifier's own commands produced.

### Human Verification Required

None. Every must-have in this phase was independently verifiable by direct execution (file existence, ratchet fail-then-pass, byte-diff regression tests, `go build`/`go vet`/`go test`/`go test -race`, and live execution of `session-verify-fresh`, `oracle selftest`, and the dream wrapper's `activity-log` touch point). No visual, real-time, or external-service-dependent behavior is in scope for this phase.

### The Sweep: Does Anything Still "Claim to Define Behaviour While Defining Nothing"?

Explicit ruling, as requested: **the phase GOAL, as operationalized by the ROADMAP's own 5 enumerated success criteria, is satisfied.**

The phase's own planning pass (191-CONTEXT.md, decisions D-02 and D-14) and execution (`deferred-items.md`) discovered additional zero-reader/unreferenced candidates beyond the literal, named scope of ROADMAP criteria 1 and 3:

- 4 more zero-reader `colony/policies/*.yaml` files (`pheromone-lifecycle.yaml`, `safety-gates.yaml`, `signal-rules.yaml`, `skill-creation.yaml`) — confirmed present and unmodified (verifier re-confirmed via `ls colony/policies/`)
- 27 zero-reader `colony/prompts/*.md` per-agent files (everything except `colony-prime.md`) — confirmed present, 27 files, unmodified
- `colony/playbooks/*.md` (7 files, ambiguous test-only readers, not fully traced) — confirmed present, only `build.md` content-changed (1 line, an in-scope fix removing a reference to the now-deleted `aether skill-inject`)
- The 8 CRUD-authoring skill-lifecycle commands (`skill-patch`, `skill-archive`, `skill-pin`, `skill-list-lifecycle`, `skill-promote`, `skill-view`, `skill-curator-run`, `skill-recover`) — a different, unreviewed subsystem per the existing reachability test's own comment; confirmed still registered
- `cmd/skills.go`'s `skillMatchesWorkspace` (zero callers, discovered incidentally during 191-05) — confirmed zero callers by the verifier, correctly untouched
- 4 `command_catalog.json` entries missing classification metadata (pre-existing, unrelated to this phase) — logged in `deferred-items.md`, not this phase's concern

Every one of these is present, correctly ledgered (either in 191-CONTEXT.md's D-02/D-14 decisions or in `deferred-items.md`), and none was silently dropped or silently expanded into scope. The ROADMAP's own text is explicit that this phase "**Delivers ROSTER-01/02 and SKILL-01**" — three specific requirement IDs with exactly-enumerated file/command sets — not an unbounded sweep of the whole repository. REQUIREMENTS.md independently confirms ROSTER-03..08 were shelved to Future Requirements by owner decision, and no requirement ID maps any of the above items to this phase. A conservative "keep and log for a future ruling" disposition on discovered-but-unlisted items is the textbook-correct execution of the phase's own explicit planning discretion ("when in doubt, KEEP a file and log it") and CLAUDE.md's own warning against a "cleanup" phase silently expanding its own blast radius. This is a defensible, correctly-bounded reading of the goal text, not a dodge of it.

### Gaps Summary

No gaps found. All 5 ROADMAP success criteria and all cross-plan must-haves verified by direct execution, not by trusting SUMMARY.md narrative. Two documented, evidence-backed overrides apply (see frontmatter) — both are cases where the executing plans caught and correctly resolved a place where blindly following literal instruction text would have introduced a real regression or broken a live wrapper; both were independently re-derived and confirmed correct by the verifier, not merely accepted on the plan's word.

---

_Verified: 2026-08-21T09:29:39Z_
_Verifier: Claude (gsd-verifier)_
