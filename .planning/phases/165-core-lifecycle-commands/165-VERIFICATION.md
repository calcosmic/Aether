---
phase: 165-core-lifecycle-commands
verified: 2026-08-03T00:00:00Z
status: gaps_found
score: 9/10 must-haves verified
overrides_applied: 0
gaps:
  - truth: "init.md's Shelf Backlog stage never instructs a hand-write to protected state (session.json / COLONY_STATE.json), per the plan's own CRITICAL D-2 fence"
    status: failed
    reason: >
      .claude/commands/ant/init.md:257 (identical in .opencode/commands/ant/init.md:257
      and the flat mirror .claude/commands/ant-init.md:257) instructs: "Promoted items
      become todos: append `[shelf:{category}] {text}` to `active_todos` in the session
      file or colony state." This directly contradicts the same file's own <read_only>
      block ("This wrapper never writes these files by hand ... session.json ...
      COLONY_STATE.json"), the guardrail in .aether/commands/init.yaml, and the
      165-05-PLAN.md Task 2 action's explicit CRITICAL D-2 fence, which forbids writing
      or describing writes to session.json / COLONY_STATE.json. Independently confirmed
      the runtime provides no alternative: cmd/shelf_init.go's shelfEntryToTodo (which
      produces exactly this string) has zero call sites, and promoteShelfEntry only
      flips shelf status. Following this instruction either hand-mutates protected
      state (the exact Frankenstein-state corruption class this project has already
      hit once) or the promoted item silently never becomes a todo. This also evades
      the phase's own regression fence: TestInitWrapperCeremonyContract's forbidden
      list does not match this phrasing. Confirmed present in the current working tree
      (git status clean on init.md since the review that first found this).
    artifacts:
      - path: ".claude/commands/ant/init.md"
        issue: "Line 257 instructs a hand-append to `active_todos` in the session file or colony state"
      - path: ".opencode/commands/ant/init.md"
        issue: "Same instruction, byte-identical location"
      - path: ".claude/commands/ant-init.md"
        issue: "Flat mirror carries the same instruction"
      - path: "cmd/init_wrapper_ceremony_test.go"
        issue: "forbidden slice does not include this phrasing ('append `[shelf:' / 'to `active_todos`'), so this regression evades the phase's own fence"
    missing:
      - "Remove the hand-append instruction from all three init.md surfaces (.claude, .opencode, flat mirror), or route it through a real runtime command (e.g. aether shelf-promote-batch or aether init appends the todo using the existing but currently-unused shelfEntryToTodo helper)"
      - "Add the phrase to cmd/init_wrapper_ceremony_test.go's forbidden slice so this class of regression fails a test going forward, per the code reviewer's suggested fix"
---

# Phase 165: Core Lifecycle Commands Verification Report

**Phase Goal:** `init`, `plan`, `build`, and `continue` wrappers are rewritten to carry engineering method — stage purpose, files to read, spawn choreography, stop conditions — instead of protocol instructions whose primary job is parsing `result.manifest.dispatch_manifest`. Phase 165 is the sole structural owner of `build.md` for this milestone.
**Verified:** 2026-08-03
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Reading build.md/plan.md/continue.md/init.md shows stage purpose, files-to-read guidance, spawn choreography, and stop conditions (ROADMAP SC1, CMD-01) | ✓ VERIFIED | `**Purpose:**` present 10x (build), 8x (plan), 8x (init), 3x (continue); `**Reads:**`/`**Stop conditions:**` present throughout; `TestBuildWrapperStageSkeletonAndParity`, `TestContinueWrapperStageSkeletonAndParity`, `TestPlanWrapperStageSkeleton`, `TestInitWrapperStageSkeletonAndParity` all PASS (verified by direct `go test` run, not SUMMARY claim) |
| 2 | Grepping the four wrappers for envelope-parsing/manifest-file-writing as primary job returns effectively zero (method markers ≥3x envelope markers; contract doc referenced exactly once per wrapper, zero for init) (ROADMAP SC2, CMD-02) | ✓ VERIFIED | `grep -c wrapper-host-contract.md` = 1 for build/plan/continue, 0 for init (confirmed directly); `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob` PASSES including `contract_pointer_is_singular` and the negative-control subtest |
| 3 | A user reading build.md can describe each stage's purpose and worker inputs without opening Go source (ROADMAP SC3, CMD-03) | ✓ VERIFIED | Recorded human checkpoint in 165-VALIDATION.md: dated 2026-08-03, verdict "Approved," all nine stages stated in one sentence each; this was a blocking `checkpoint:human-verify` gate (165-06 Task 3), not a self-reported claim |
| 4 | `/ant-chaos`, `/ant-archaeology`, `/ant-dream`, `/ant-oracle`, `/ant-swarm`, `/ant-sage`, `/ant-colonize`, `/ant-council` continue to work unchanged (ROADMAP SC4, CMD-04) | ✓ VERIFIED | `TestSpecialistCommandSurfacesUnchanged` PASSES (17-file SHA-256 ledger + count assertion + command-guide reachability); independently confirmed via `git log` that none of the 8 surfaces (14 wrapper files + 3 sage agent files) have any commit in the phase's working period |
| 5 | build.md's structural rewrite is committed by Phase 165 alone; ownership/merge order is traceable in the file (ROADMAP SC5, CMD-05) | ✓ VERIFIED | `.claude/commands/ant/build.md:7` carries the `PHASE-160:`/`PHASE-165`/`PHASE-168:` ownership comment; line 262 (last non-empty line) is exactly `<!-- PHASE-168: visual-guidance trailer appends below this line -->`; `TestBuildMdOwnershipHandshake` PASSES |
| 6 | continue.md's runtime-owned context-clear line stays out of the wrapper (fast + heavy paths) | ✓ VERIFIED | `grep -c "safe to clear your context"` / `grep -c ant-resume` return 0 across `.claude`, `.opencode`, and the flat mirror; `TestContinueWrapperStageSkeletonAndParity/context_clear_stays_runtime_owned` PASSES |
| 7 | plan.md's two-decision-moment contract (Phase 164) survives the rewrite | ✓ VERIFIED | `## Decision Moment 1`/`## Decision Moment 2` both present; `plan-research-approve --approve-all` and `--auto` each appear exactly once; `TestPlanWrapperCardsParity` and `TestPlanWrapperCeremonyContract` PASS unchanged |
| 8 | init.md's 👑 intention-setting beat is restored at approval, in current vocabulary | ✓ VERIFIED | `.claude/commands/ant/init.md:282`: `👑 Queen has set the colony's intention — "{refined_goal}"` |
| 9 | init.md carries zero envelope-parsing prose (it has no host-manifest step) | ✓ VERIFIED | `grep -c 'result\.manifest\.\|dispatch_manifest'` on init.md returns 0; `TestInitWrapperStageSkeletonAndParity/no_envelope_parsing_prose` PASSES |
| 10 | init.md never instructs a hand-write to protected state (session.json / COLONY_STATE.json), consistent with its own `<read_only>` block and the plan's CRITICAL D-2 fence | ✗ FAILED | Line 257 instructs "append `[shelf:{category}] {text}` to `active_todos` in the session file or colony state" — a direct hand-mutation instruction contradicting the file's own guardrails; see Gaps below |

**Score:** 9/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.aether/docs/wrapper-host-contract.md` | "Manifest and Completion Packet Shapes" + terminal-result ruling | ✓ VERIFIED | Both headings present exactly once; 9 pre-existing + new `## ` sections intact |
| `cmd/lifecycle_wrapper_contract_test.go` | Shared toolkit + flat-mirror/retired-vocab invariants | ✓ VERIFIED | Compiles, all 3 Plan-01 tests + Plan-06 additions pass |
| `.claude/commands/ant-build.md`, `ant-init.md` | Resynced flat mirrors | ✓ VERIFIED | `diff` against canonical produces no output |
| `.claude/commands/ant/build.md`, `.opencode/...` | Rewritten build wrapper, byte-parity | ✓ VERIFIED | `diff` identical; `TestBuildWrapperCeremonyContract` etc. pass |
| `.claude/commands/ant/continue.md`, `.opencode/...` | Rewritten continue wrapper, byte-parity | ✓ VERIFIED | `diff` identical |
| `.claude/commands/ant/plan.md`, `.opencode/...` | Rewritten plan wrapper, byte-parity | ✓ VERIFIED | `diff` identical |
| `.claude/commands/ant/init.md`, `.opencode/...` | Rewritten init wrapper, sanctioned 1-line platform delta | ✓ VERIFIED (content, but see gap) | `diff` shows exactly the AskUserQuestion/Ask line; content itself carries the CR-01 defect (Truth 10) |
| `cmd/init_wrapper_ceremony_test.go` | First dedicated init ceremony test | ✓ VERIFIED | Exists, exports `TestInitWrapperCeremonyContract`, `TestInitWrapperStageSkeletonAndParity`, both pass — but does not fence the CR-01 phrasing |
| `.aether/commands/{build,continue,plan,init}.yaml` | Guardrails updated in lockstep | ✓ VERIFIED | `stage_skeleton`/`ownership_chain`/`termination_conditions`/`intention_beat` keys present per plan; `depth_proposal_card`/`research_batch_card` preserved |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/lifecycle_wrapper_contract_test.go` | `.claude/commands/ant-{init,plan,build,continue}.md` | byte-equality vs canonical | ✓ WIRED | `TestLifecycleFlatMirrorsMatchCanonical` passes for all four verbs |
| `cmd/*_wrapper_ceremony_test.go` | canonical wrapper files | `assertStageSkeletonDensity` / `assertOrderedHeadingParity` | ✓ WIRED | Verified directly with `go test -v`, all subtests pass |
| `.claude/commands/ant/{build,plan,continue}.md` | `.aether/docs/wrapper-host-contract.md` | single contract pointer | ✓ WIRED | Exactly one occurrence each, confirmed by direct grep |
| `cmd/lifecycle_wrapper_contract_test.go` | 17 specialist/delight surfaces | SHA-256 content ledger | ✓ WIRED | `TestSpecialistCommandSurfacesUnchanged` passes; independently cross-checked with `git log` on all 8 named surfaces |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full test suite compiles and passes for all lifecycle/ceremony tests | `go test ./cmd/... -run '<19 test names from all 6 plans>'` | All PASS, 0 failures | ✓ PASS |
| `go vet ./...` clean | `go vet ./...` | no output | ✓ PASS |
| `go build ./cmd/aether` succeeds | `go build ./cmd/aether` | exit 0 | ✓ PASS |
| No debt markers (TBD/FIXME/XXX) in phase-modified files | `grep -nE 'TBD|FIXME|XXX'` across all rewritten wrappers + test files | 0 matches | ✓ PASS |
| Retired depth vocabulary absent from all 8 canonical + flat-mirror files | `grep colony_depth` | 0 matches | ✓ PASS |
| D-10 item 9 git-mutation fence holds | `grep -E 'git stash|git add -A|git commit'` on build/continue canonical + mirrors | 0 matches | ✓ PASS |
| Context-clear fence holds on continue.md (all 3 surfaces) | `grep 'safe to clear your context\|ant-resume'` | 0 matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CMD-01 | 165-02..05, 165-06 | Wrappers carry engineering method (stage purpose, reads, spawn choreography, stop conditions) | ✓ SATISFIED | Stage-skeleton density tests pass on all 4 wrappers; structured blocks present in all 8 canonical files |
| CMD-02 | 165-01, 165-06 | No wrapper's primary job is envelope-parsing | ✓ SATISFIED | Ratio invariant (≥3:1) + singular contract pointer, tested and independently confirmed |
| CMD-03 | 165-06 | build.md describable stage-by-stage without opening Go source | ✓ SATISFIED | Dated, recorded human verdict via blocking checkpoint |
| CMD-04 | 165-06 | 8 specialist/delight commands keep working unchanged | ✓ SATISFIED | SHA-256 ledger + git history confirm zero touches |
| CMD-05 | 165-02 | build.md sole structural owner, merge order traceable | ✓ SATISFIED | Ownership handshake comment + trailer marker + dedicated test |

**Note:** All five CMD IDs pass their literal test coverage. However, Truth 10 (a phase-goal-adjacent regression of the project's own documented D-2 "Frankenstein-state" corruption class, explicitly fenced as CRITICAL in 165-05-PLAN.md's own task action) fails independently of the five CMD IDs' literal wording, because none of CMD-01..05 as worded happens to prohibit a hand-state-write instruction. This is flagged as a phase-level gap regardless of formal requirement-ID satisfaction, per this project's Definition of Done (a documentation/protocol claim must be testable, and a known regression class re-appearing silently is exactly the failure pattern CLAUDE.md's own history calls out).

REQUIREMENTS.md's traceability table still shows CMD-01..05 as `[ ]` / "Pending" — this reflects that the master requirements ledger has not yet been updated post-phase (consistent with RESEARCH-01..10 for the already-verified Phase 164, which are `[x]`/"Complete" only after that phase's own verification concluded). Not treated as a gap in itself; it is expected to be updated once this phase is signed off.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.claude/commands/ant/init.md` (+ `.opencode`, flat mirror) | 257 | Hand-write instruction to protected state (`active_todos` in session/colony state) | 🛑 Blocker | See Truth 10 / gaps above (code review CR-01, independently confirmed) |
| `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md` | build:50-52 vs 255; continue:190-192 vs 205 | `<read_only>` block says "may read" while Guardrails say "Do NOT read" — internal contradiction, drifts from own YAML source wording | ⚠️ Warning | Ambiguous instruction to the model; does not block the phase goal but degrades clarity (review WR-01) |
| `.claude/commands/ant/continue.md` | 200-209 | Several guardrails present in `continue.yaml` source are missing from the generated wrapper (`--synthetic` fence, "parse visual output" fence, etc.) | ⚠️ Warning | Guardrail drift within a single deliverable (review WR-02) |
| `cmd/build_wrapper_ceremony_test.go` | 21-54 | `build-completion-stage` / `result.completion_path` staging step has no required-string fence | ⚠️ Warning | A future edit could silently delete the safety-critical staging step (review WR-03) |
| `.claude/commands/ant/plan.md` | 50-78 | Boundary-guidance clarification gate runs after Decision Moment 2 has already mutated research-approval state | ⚠️ Warning | Wasted interaction / approvals recorded against a manifest about to be discarded (review WR-04) |
| `.claude/commands/ant/init.md` | 283-284 | Pheromones written before `aether init` succeeds — ordering risk vs. stated "write nothing on cancel/failure" claim | ⚠️ Warning | Possible partial-persistence on init failure (review WR-05) |
| `.claude/commands/ant/init.md` | 16, 284 | Fragile shell-quoting in `--goal`/`--charter-json` templates (unescaped quotes) | ⚠️ Warning | Command could break or mis-execute on ordinary user/AI-composed text (review WR-06) |
| `.claude/commands/ant/build.md` | 44-45 vs 184 | Partial-wave-failure policy stated two different ways | ⚠️ Warning | Ambiguous stop-condition semantics (review WR-07) |
| `cmd/continue_wrapper_ceremony_test.go` | 270-289 | Dead helper `sliceBetweenMarkers`, no call sites | ℹ️ Info | Dead test code (review IN-01) |
| `cmd/lifecycle_wrapper_contract_test.go` | 267 vs 3 other files | Ratio constant `3` duplicated as magic number instead of using `wrapperRatioOK` | ℹ️ Info | Maintainability only (review IN-03) |
| `.aether/commands/continue.yaml` / `.claude/commands/ant/build.md` | continue.yaml:5, build.md:237 | Minor doc-quality nits (duplicate flag risk, `--light` description mismatch) | ℹ️ Info | review IN-04, IN-07 |

## Human Verification Required

None outstanding. CMD-03's manual read-through requirement was already discharged by a recorded, dated, blocking human-verify checkpoint during plan 165-06 (see Truth 3). No new user-facing behavior in this phase requires fresh human testing.

## Gaps Summary

One blocker: **init.md instructs the LLM to hand-append a todo string to `active_todos` in the session file or colony state** (line 257, mirrored in `.opencode` and the flat installed mirror). This is not a hypothetical risk — it is the exact class of bug (LLM reconstructs/hand-mutates protected JSON state, causing Frankenstein-state corruption) that this project has already suffered once and that Plan 165-05's own task action explicitly fenced as CRITICAL ("Do not write, and do not describe writing, `COLONY_STATE.json`, `session.json`, ..."). The instruction contradicts the same file's `<read_only>` block and guardrail bullet, and there is no runtime command that performs this write today (`shelfEntryToTodo` has zero callers in `cmd/shelf_init.go`), so an LLM following this exact instruction has only two paths: violate the state-write boundary, or silently drop the promised "promoted items become todos" behavior. The phase's own regression fence (`TestInitWrapperCeremonyContract`'s forbidden list) does not catch this specific phrasing, so nothing currently prevents it from shipping or recurring.

Seven warnings (WR-01 through WR-07) and several info items from the code review were independently confirmed in the codebase; none of them individually block the phase's core stage-skeleton/ceremony/ownership deliverables, but WR-01 (contradictory read/write instructions), WR-04 (ordering hazard around boundary guidance), and WR-05 (pheromone-write-before-init-success ordering) are worth closing alongside the blocker since they touch the same class of internal-consistency and state-safety concerns this phase was meant to harden.

Everything else — the four wrappers' stage skeletons, the CMD-02 method/envelope ratio, the CMD-03 human read-through, the CMD-04 specialist-command fence, and the CMD-05 build.md ownership handshake — is verified directly against the current codebase (not SUMMARY claims): tests were re-run in this verification pass, `diff`/`grep` checks were run independently, and git history was checked for the specialist-command files.

---

_Verified: 2026-08-03_
_Verifier: Claude (gsd-verifier)_
