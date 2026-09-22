---
phase: 206-screens-reach-the-owner
verified: 2026-09-22T00:00:00Z
status: passed
score: 10/10 must-haves verified
behavior_unverified: 0
overrides_applied: 0
resolved_after_verification:
  - "WR-03 resolved 2026-09-22 by the orchestrator: CLAUDE.md gained the section 'A permanent status line (v1.29, Phase 206)' naming all seven locking tests, and TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest now guards that section too (proven to fail on a fake test name). No human item remains."
human_verification_original:
  - test: "Decide whether WR-03 (the permanent status line has zero mentions in CLAUDE.md — no section, no cited locking test) needs a documentation fix before shipping, or can ship as a tracked follow-up."
    expected: "Either CLAUDE.md gains a status-line section naming its seven locking tests (mirroring the direct-route section), or the owner accepts the gap as a tracked, non-blocking follow-up."
    why_human: "This is a documentation-completeness judgment call, not a code defect — the status line itself is fully built, tested (7 tests, including a race-clean concurrency test and an AST guard proving it spells no command of its own) and registered. CLAUDE.md's own stated policy governs false claims ('a documentation claim about runtime behaviour must be testable or removed'), not omitted ones, so this doesn't mechanically fail any rule — but the project's own code review (206-REVIEW.md WR-03) flagged the asymmetry with its sibling feature (the direct route) as worth a human decision rather than silent ratification."
---

# Phase 206: Screens Reach the Owner — Verification Report

**Phase Goal:** The owner sees the program's own screens in the chat, every time, unchanged.
**Verified:** 2026-09-22
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (roadmap success criterion) | Status | Evidence |
|---|---|---|---|
| 1 | Every menu command that draws a screen tells the chat to show it unchanged, and a test fails if any command loses that instruction (UED-01) | ✓ VERIFIED | `go test ./cmd -run TestEveryWrapperThatDrawsAScreenRelaysIt` — PASS. Pre-existing from release 1.0.88, re-confirmed unchanged by this phase. |
| 2 | A reply that hides the screen is sent back once, never twice, and the check changes nothing on disk (UED-02) | ✓ VERIFIED | `go test ./cmd -run 'TestStopHookSendsBackAReplyThatHidTheScreen|TestStopHookNeverBlocksTwice|TestStopHookScreenCheckDoesNotMutate'` — all PASS. |
| 3 | A screen printed for a chat carries no colour escape codes (UED-03) | ✓ VERIFIED | `go test ./cmd -run TestPipedVisualOutputCarriesNoEscapeCodes` — PASS. |
| 4 | Start-up, code survey and helper cards are in the guarded look (UED-04) | ✓ VERIFIED | `go test ./cmd -run TestEveryOrdinaryScreenIsMeasuredForVoice` — PASS (eleven screen families measured). |
| 5 | The program hands its screen to Claude Code to show directly (UED-05, delivery mechanism) | ✓ VERIFIED | `go test ./cmd -run 'TestDirectRoute...'` — all 13 direct-route tests PASS, including the code-review fix `TestDirectRouteRegistrationRequiresABashMatcher`. `cmd/hook_direct_screen.go` implements `hookPostToolUseCmd` → `directScreenDelivery` → `emitDirectScreen` (systemMessage), registered in `init()` (`cmd/hook_cmds.go:1098`) and shipped in `.claude/settings.json` (`PostToolUse`, matcher `Bash`, `aether hook-post-tool-use`, timeout 10 — confirmed live in the repo's own settings file). |
| 6 | The Stop-hook backstop composes honestly with the direct route (one shared decision, never demanded twice, still demanded when partial) | ✓ VERIFIED | `TestDirectRouteAndTheBackstopUseOneDecision`, `TestBackstopStaysQuietWhenTheScreenAlreadyArrived`, `TestBackstopStillFiresWhenOnlyPartOfTheScreenArrived` — all PASS. Code inspection of `screenRelayBlockReason` (`cmd/hook_cmds.go:359-382`) confirms the guard requires both `deliver && complete` from `directScreenDelivery` AND `directScreenRouteRegistered(input.Cwd)` before staying silent. |
| 7 | The owner has looked at it on his own screen and said yes or no (UED-05 judgment) | ✓ VERIFIED | `.planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md` records the owner's verbatim answer `keep-on`, dated 2026-09-22, with the exact question put to him. `.claude/settings.json` reflects the verdict: `PostToolUse` entry for `aether hook-post-tool-use` is present and registered (matches "keep-on"), matching Task 4's own acceptance check. |
| 8 | A permanent status line shows phase, task and next command (UED-06) | ✓ VERIFIED | `go test ./cmd -run 'TestStatusLine...'` — all 7 tests PASS, including `TestStatusLineIsSafeUnderConcurrentReads` under `-race`. `.claude/settings.json` carries the shipped `statusLine` key (`aether status-line`, confirmed live). Code inspection of `statusLineText` confirms it routes every command through `translateHintCommandsForPlatform` (the one existing translator) and an AST guard (`TestStatusLineComesFromTheSharedDecision`'s second half) proves no command string is spelled inside `status_line.go`. |
| 9 | Proven in real chats in a scratch project, not only in tests (UED-05/06, success criterion 5) | ✓ VERIFIED | `scripts/proof-screens-reach-the-owner.sh` exists, is executable, `bash -n` passes, and structurally implements the five gates the plan specifies (install-path assertion, banner-line extraction, a real capped `claude -p` chat, a delivery check that explicitly excludes `tool_result` echo text before searching for banner lines, and a status-line check) with `--max-turns 6` and a `timeout 300` wall-clock cap. Per this verification's explicit instructions, the script was **not** re-run (it spends real money); the two real runs recorded in `206-02-SUMMARY.md` (2026-09-21 and 2026-09-22, both exit 0, "7/7 banner lines proven delivered as an informational message distinct from the tool's own output") are accepted as the evidence for this truth, consistent with the phase's own stated two-live-run budget. |
| 10 | Requirement traceability — UED-01 through UED-06 are all accounted for with no orphans | ✓ VERIFIED | `.planning/REQUIREMENTS.md` marks all six `[x]`; each cites a real, passing test. Plan 01 frontmatter claims `[UED-01, UED-02, UED-03, UED-04, UED-05]`; plan 02 claims `[UED-05, UED-06]`. No requirement mapped to Phase 206 in REQUIREMENTS.md is absent from a plan's `requirements` field. |

**Score:** 10/10 truths verified, 0 present-but-behavior-unverified.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/hook_direct_screen.go` | Direct route implementation | ✓ VERIFIED | 246 lines; exists, builds, contains `hookPostToolUseCmd`, `directScreenDelivery`, `directScreenMessageWithinCap`, `directScreenRouteRegistered`, `directScreenMatcherCoversBash`, `emitDirectScreen`. |
| `cmd/hook_direct_screen_test.go` | Direct route tests | ✓ VERIFIED | Exists; 13 named tests all PASS, including the post-review fix test. |
| `cmd/claudemd_direct_screen_test.go` | CLAUDE.md claim-locking guard | ✓ VERIFIED | `TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest` PASSES. |
| `cmd/testdata/post-tool-use/status-screen-payload.json` | Real captured fixture | ✓ VERIFIED | Exists, parses as JSON, `tool_response.stdout` is a genuine 6,386-byte `aether status` screen containing banner-bar lines. |
| `.claude/settings.json` | Shipped hook + status-line registration | ✓ VERIFIED | Live-read from the repo: `PostToolUse` entry matcher `Bash` → `aether hook-post-tool-use` (timeout 10); top-level `statusLine` → `aether status-line`. |
| `cmd/status_line.go` | Status line implementation | ✓ VERIFIED | 152 lines; `statusLineCmd`, `statusLineText`, `statusLineTaskLabel`, `shortenStatusLineGoal`, registered in `init()`. |
| `cmd/status_line_test.go` | Status line tests | ✓ VERIFIED | 7 named tests, all PASS (including `-race`). |
| `scripts/proof-screens-reach-the-owner.sh` | Real-chat proof script | ✓ VERIFIED | Executable, `bash -n` clean, contains turn cap, wall-clock cap, five gates. Not re-executed per verification instructions (costs real money); two prior real runs documented in SUMMARY. |
| `.planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md` | Recorded owner decision | ✓ VERIFIED | Exists, dated, verbatim `keep-on` answer. |
| CLAUDE.md — "The screen reaches the owner directly (v1.29, Phase 206)" | Documentation for the direct route | ✓ VERIFIED | Section exists (line 280), names every locking test, closes with the owner's dated verdict sentence. |
| CLAUDE.md — status line documentation | Documentation for `aether status-line` | ✗ MISSING | Zero mentions of `status-line`/`statusLine` anywhere in CLAUDE.md (confirmed by case-insensitive grep). Flagged by code review as WR-03, still open per the task brief. Not a required artifact of any plan's `must_haves`, and does not contradict any existing documented claim — routed to human verification rather than treated as a blocking gap. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `hookPostToolUseCmd` | `directScreenDelivery` | direct call in RunE | ✓ WIRED | Confirmed by code read and by `TestDirectRouteAndTheBackstopUseOneDecision` (AST-asserted, both call sites reach the one function). |
| `screenRelayBlockReason` | `directScreenDelivery` + `directScreenRouteRegistered` | composed guard before per-line comparison | ✓ WIRED | `cmd/hook_cmds.go:371` — both conditions required (`deliver && complete` and registration) before staying silent. |
| `.claude/settings.json` `PostToolUse` entry | `hookPostToolUseCmd` | cobra command registration | ✓ WIRED | `rootCmd.Find("hook-post-tool-use")` resolves; `TestDirectRouteHookIsRegistered` PASSES. |
| `.claude/settings.json` `statusLine` key | `statusLineCmd` | cobra command registration | ✓ WIRED | `TestStatusLineIsRegistered` PASSES; `rootCmd.AddCommand(statusLineCmd)` confirmed at `cmd/hook_cmds.go:1102`. |
| `statusLineText` | `resolveNextAction` | the one shared what-next decision | ✓ WIRED | `TestStatusLineComesFromTheSharedDecision` PASSES (both the content assertion and the AST no-rival-command guard). |
| `statusLineText` command segment | `translateHintCommandsForPlatform` | render-time translation | ✓ WIRED | Confirmed by code read (`cmd/status_line.go:118`) and by a documented, verified deviation from the plan's literal wording (SUMMARY key-decisions) that was the correct fix, proven by `go run ./cmd/aether status-line` printing `/ant-resume` in this repo rather than the untranslated `aether resume`. |
| `directScreenRouteRegistered` | project's own `.claude/settings.json` / `.claude/settings.local.json` | live read, Bash-matcher gated | ✓ WIRED | Post-review fix (commit `9225272e`) added `directScreenMatcherCoversBash`; `TestDirectRouteRegistrationRequiresABashMatcher` (6 subtests) PASSES. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Build and vet clean | `go build ./cmd/aether && go vet ./cmd && go vet ./...` | exit 0, no output | ✓ PASS |
| Direct-route + backstop group | `go test ./cmd -run 'TestDirectRoute\|TestBackstop\|TestStopHook\|TestBannerPredicate\|TestEveryDirectScreenClaim' -count=1` | ok, all subtests PASS | ✓ PASS |
| Status-line group | `go test ./cmd -run 'TestStatusLine' -count=1` | ok, 7/7 PASS | ✓ PASS |
| Status-line concurrency | `go test ./cmd -run TestStatusLineIsSafeUnderConcurrentReads -race -count=1` | ok | ✓ PASS |
| UED-01/03/04 sibling proof tests | `go test ./cmd -run 'TestEveryWrapperThatDrawsAScreenRelaysIt\|TestPipedVisualOutputCarriesNoEscapeCodes\|TestEveryOrdinaryScreenIsMeasuredForVoice'` | ok, all PASS | ✓ PASS |
| Reachability ratchet | `go test ./cmd -run 'TestAuditCatalogGolden\|TestCatalogCompleteness\|TestNoRegisteredSubcommandIsUnreferenced'` | `TestNoRegisteredSubcommandIsUnreferenced` FAILS on `aether codex-native-worker context-ack` only | ⚠ Pre-existing, unrelated (confirmed against `deferred-items.md` and commit `1f905181`, Phase 204.2 in-progress work — not introduced by this phase) |
| Full suite | Not re-run (per explicit instruction) | `FULL-SUITE FAIL discovered=5908 executed=5908` per `206-02-SUMMARY.md`, 30 pre-existing/known-red names, zero `TestDirectRoute*`/`TestBackstop*`/`TestStatusLine*` failures | Accepted as documented evidence, not independently re-executed |
| Real-chat proof script | Not re-run (per explicit instruction — spends real money) | Two real runs recorded 2026-09-21/22, both exit 0 | Accepted as documented evidence, not independently re-executed |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| UED-01 | (pre-existing, re-confirmed by 206-01) | Every menu command relays its screen | ✓ SATISFIED | `TestEveryWrapperThatDrawsAScreenRelaysIt` |
| UED-02 | (pre-existing, re-confirmed by 206-01) | Reply sent back once, never twice, no mutation | ✓ SATISFIED | `TestStopHookSendsBackAReplyThatHidTheScreen`, `TestStopHookNeverBlocksTwice`, `TestStopHookScreenCheckDoesNotMutate` |
| UED-03 | (pre-existing, re-confirmed by 206-01) | No colour escape codes | ✓ SATISFIED | `TestPipedVisualOutputCarriesNoEscapeCodes` |
| UED-04 | (pre-existing, re-confirmed by 206-01) | Guarded look, eleven families | ✓ SATISFIED | `TestEveryOrdinaryScreenIsMeasuredForVoice` |
| UED-05 | 206-01, 206-02 | Direct route + owner judgment | ✓ SATISFIED | Direct-route test group, `206-OWNER-VERDICT.md`, proof script (documented runs) |
| UED-06 | 206-02 | Permanent status line | ✓ SATISFIED | Status-line test group, shipped settings key |

No orphaned requirements: every ID REQUIREMENTS.md maps to Phase 206 appears in a plan's `requirements` field, and vice versa.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| (none) | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER, no empty-return stubs, no hardcoded-empty-props found in `cmd/hook_direct_screen.go`, `cmd/status_line.go`, their test files, or the proof script | — | Clean |
| `CLAUDE.md` | — | Status line feature entirely undocumented (WR-03, code review, still open) | ⚠ Warning | Documentation-completeness gap, not a false claim; routed to human verification below |
| `cmd/testdata/post-tool-use/status-screen-payload.json:18` | — | Real captured fixture embeds a raw internal state token (`between_commands_boundary`) that CLAUDE.md's own Classic Visual Voice section says must never reach the owner (IN-02, code review) | ℹ Info | Pre-existing bug in a different renderer (`aether status`'s "what changed" line), outside this phase's diff scope; does not affect any test in this phase; tracked as a follow-up, not a Phase 206 defect |
| `scripts/proof-screens-reach-the-owner.sh:170-176` | — | Comment misattributes the platform's own message prefix to Aether's code (IN-01, code review) | ℹ Info | Cosmetic; does not affect script correctness or gate logic |

## Human Verification Required (resolved — see frontmatter)

### 1. WR-03 — status line undocumented in CLAUDE.md

**Test:** Decide whether `aether status-line` needs its own CLAUDE.md section (mirroring the direct-route section's structure, citing its seven locking tests) before this phase is considered fully closed, or whether the gap is accepted as a tracked follow-up.
**Expected:** Either CLAUDE.md gains the section, or the owner explicitly accepts the asymmetry as intentional/deferred.
**Why human:** This is a documentation-completeness judgment, not a mechanically verifiable defect — the feature itself is fully built, tested, and wired; nothing behaves incorrectly. The project's own code review already flagged it as a real (if non-critical) gap and it remains open per this verification's own task brief, so it is surfaced here for an explicit decision rather than silently waved through or silently treated as a blocker.

## Gaps Summary

No blocking gaps. Every roadmap success criterion and every plan `must_haves.truths` entry for both 206-01 and 206-02 is backed by a passing, targeted test or direct code/settings inspection. The two code-review warnings with real correctness implications (WR-01: matcher-blind registration check; WR-02: mid-word truncation contradicting its own doc comment) were both fixed in commit `9225272e` and are now locked by `TestDirectRouteRegistrationRequiresABashMatcher` and an honest doc comment, respectively — verified directly in this pass, not merely trusted from the SUMMARY. The one remaining open review item (WR-03, status line undocumented in CLAUDE.md) is a documentation-completeness gap with no behavioral consequence, routed to human verification rather than blocking. The two Info-level findings (IN-01, IN-02) are cosmetic/out-of-scope and require no action from this phase. The one reachability-ratchet failure (`aether codex-native-worker context-ack`) is a confirmed pre-existing orphan from Phase 204.2's in-progress work, untouched by this phase's changes, and logged in `deferred-items.md`.

---

_Verified: 2026-09-22_
_Verifier: Claude (gsd-verifier)_
