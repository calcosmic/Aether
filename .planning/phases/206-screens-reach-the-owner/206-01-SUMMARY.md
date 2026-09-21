---
phase: 206-screens-reach-the-owner
plan: 01
subsystem: cli-hooks
tags: [go, cobra, claude-code-hooks, posttooluse, stop-hook, settings-merge]

requires:
  - phase: 205-screens-reach-the-owner-part-c
    provides: the Stop-hook finish-check backstop (screenRelayBlockReason, owedScreenBanners, isAetherBannerLine, isAetherVisualCommand) this plan extends and composes with
provides:
  - "aether hook-post-tool-use: hands a drawn Aether screen to Claude Code as a systemMessage the owner sees directly, no model turn spent"
  - "directScreenDelivery: the one decision function shared by the direct route and the Stop-hook backstop, including the 10,000-byte length rule"
  - "directScreenRouteRegistered: reads a project's own .claude/settings.json (or settings.local.json) to know whether the direct route is installed"
  - "screenRelayBlockReason composes with the direct route: stays quiet only when the whole screen provably already arrived AND the route is registered"
  - "a real captured PostToolUse fixture (cmd/testdata/post-tool-use/status-screen-payload.json) for future direct-route work to build on"
affects: [206-02-status-line-and-real-chat-proof]

actuals:
  tokens: 16000
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "One decision function (directScreenDelivery) shared structurally (AST-asserted) between two independent call sites, so a length-rule change can never happen in only one of them"
    - "Real hook-payload fixtures captured live from a genuine running Claude Code session rather than hand-typed, redacting only what identifies the capturing machine/session"

key-files:
  created:
    - cmd/hook_direct_screen.go
    - cmd/hook_direct_screen_test.go
    - cmd/claudemd_direct_screen_test.go
    - cmd/testdata/post-tool-use/status-screen-payload.json
  modified:
    - cmd/hook_cmds.go
    - cmd/stop_hook_screen_test.go
    - .claude/settings.json
    - CLAUDE.md

key-decisions:
  - "The committed fixture represents a genuine top-level owner chat turn (no agent_id/agent_type), matching the key-absence already proven in this repo's own prior real Stop-hook fixture, rather than the spawned-subagent shape this session's own tool calls actually carry."
  - "The length rule (directScreenMessageWithinCap) is the ONLY function in the package referencing directScreenMessageCapBytes, enforced structurally by TestDirectRouteAndTheBackstopUseOneDecision, so a second copy of the cap can never silently grow."
  - "screenRelayBlockReason's new exemption requires BOTH directScreenDelivery reporting complete delivery AND directScreenRouteRegistered reading true from this project's own settings -- an unregistered project keeps the pre-206 backstop behaviour byte-for-byte."

patterns-established:
  - "A hook fixture too specific to capture from a top-level session (this executor runs as a spawned subagent) is captured for real anyway, then the minimal, evidence-backed field removal needed to represent the intended context is documented inline rather than guessed."

requirements-completed: [UED-01, UED-02, UED-03, UED-04, UED-05]

coverage:
  - id: D1
    description: "aether hook-post-tool-use hands a drawn screen to the owner directly via systemMessage, with no model turn spent"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteHandsTheScreenToTheOwner"
        status: pass
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteStaysQuietWhenThereIsNoScreen"
        status: pass
    human_judgment: false
  - id: D2
    description: "An over-cap screen arrives as its last whole lines behind a truncation notice, never a fragment of a line"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteNeverCutsALineInHalf"
        status: pass
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteMessageIsValidJSONForEveryScreen"
        status: pass
    human_judgment: false
  - id: D3
    description: "The Stop-hook backstop stops asking only once the whole screen provably arrived via a registered direct route; an unregistered project or a partial screen still blocks"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/stop_hook_screen_test.go#TestBackstopStaysQuietWhenTheScreenAlreadyArrived"
        status: pass
      - kind: unit
        ref: "cmd/stop_hook_screen_test.go#TestBackstopStillFiresWhenOnlyPartOfTheScreenArrived"
        status: pass
    human_judgment: false
  - id: D4
    description: "The direct route and the backstop reach one shared decision function structurally, never a second copy"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteAndTheBackstopUseOneDecision"
        status: pass
    human_judgment: false
  - id: D5
    description: "The route is off for Aether's own helpers, off under AETHER_SCREEN_RELAY=off, and writes nothing anywhere unless capture is explicitly switched on"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteIsSkippedForHelpersAndWorkers"
        status: pass
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteOffSwitchStopsIt"
        status: pass
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteWritesNothing"
        status: pass
    human_judgment: false
  - id: D6
    description: "The shipped settings register the route against Bash with a timeout, naming a real command, and installing it into a downstream project never disturbs that project's own hooks"
    requirement: "UED-05"
    verification:
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteHookIsRegistered"
        status: pass
      - kind: unit
        ref: "cmd/hook_direct_screen_test.go#TestDirectRouteInstallsWithoutDisturbingOtherHooks"
        status: pass
    human_judgment: false
  - id: D7
    description: "CLAUDE.md's new claims about the direct route each name a live test, and the four behaviours released in 1.0.88 (UED-01..04) still hold unchanged"
    requirement: "UED-01"
    verification:
      - kind: unit
        ref: "cmd/claudemd_direct_screen_test.go#TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest"
        status: pass
      - kind: unit
        ref: "TestEveryWrapperThatDrawsAScreenRelaysIt, TestStopHookSendsBackAReplyThatHidTheScreen, TestStopHookNeverBlocksTwice, TestStopHookScreenCheckDoesNotMutate, TestPipedVisualOutputCarriesNoEscapeCodes, TestEveryOrdinaryScreenIsMeasuredForVoice"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-09-21
status: complete
---

# Phase 206 Plan 01: Direct Screen Route Summary

**When a Bash command draws one of Aether's screens, Claude Code's own PostToolUse hook now hands that screen straight to the owner as a `systemMessage`, at zero model-turn cost, with the pre-existing Stop-hook backstop staying as the safety net for anything the new route could not deliver.**

## Performance

- **Duration:** ~40min (including live fixture capture)
- **Started:** 2026-09-21T23:00Z (approx, plan-phase commit)
- **Completed:** 2026-09-21T23:33:08+02:00
- **Tasks:** 3/3 completed
- **Files modified:** 8

## Accomplishments

- Built `aether hook-post-tool-use` (new hidden cobra command) and `directScreenDelivery` (the one decision function it shares with the Stop-hook backstop), proven end-to-end against a **real, live-captured** PostToolUse payload rather than a hand-typed one.
- Added the 10,000-byte length rule (`directScreenMessageWithinCap`): an over-cap screen arrives as its last whole lines behind a plain truncation notice, never a fragment of a line or a split multi-byte character, and reports itself incomplete so the backstop still asks for the rest.
- Composed the direct route with the existing Stop-hook backstop: `screenRelayBlockReason` now stays quiet only when the whole screen provably already arrived through a *registered* direct route (read live from the project's own `.claude/settings.json`), leaving every other case -- unregistered project, partial screen -- blocking exactly as it did in 1.0.88.
- Registered the route in the shipped `.claude/settings.json` (`PostToolUse` → `aether hook-post-tool-use` on `Bash`, timeout 10) and proved the merge that installs it downstream never disturbs a project's own foreign hooks.
- Documented the whole feature in CLAUDE.md with every claim naming its locking test, guarded by a removal-proof AST test, and re-confirmed the six proof tests for the four behaviours already released in 1.0.88 (UED-01..04) still pass unchanged.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — one real command's screen reaches the owner directly** - `96e49c14` (feat)
2. **Task 2: The length rule, and one decision shared with the finish check** - `d1f26b73` (test, RED) then `ec543cf3` (feat, GREEN)
3. **Task 3: Install it downstream, write the claim, confirm what already shipped** - `4ca84280` (docs)
4. Follow-up: `714c861b` (docs) — fixed two stale doc-comment references to the pre-rename `owedScreenBanners` left over from task 2's rename to `owedScreenFrom`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed stale doc comments after the owedScreenBanners → owedScreenFrom rename**
- **Found during:** post-task-3 sweep
- **Issue:** Two doc comments in `cmd/hook_cmds.go` still named the pre-rename function.
- **Fix:** Updated both comments to name `owedScreenFrom`.
- **Files modified:** `cmd/hook_cmds.go`
- **Commit:** `714c861b`

### Fixture capture: real, but from a spawned subagent session

This plan's Task 1 precondition required capturing the fixture "in a real Claude Code session." This executor *is* a real, live Claude Code session — but it runs as a spawned `gsd-executor` subagent (dispatched via the Task tool), not the owner's own top-level chat. I captured the fixture for real: I temporarily shimmed the installed `aether` binary (backed up first, restored after) so that only the `hook-post-tool-use` invocation was intercepted and its raw stdin bytes written to a scratch file, then ran a genuine `AETHER_OUTPUT_MODE=visual aether status` Bash call. The captured payload is byte-real in every field's shape (`tool_response` as `{"stdout": ...}`, `tool_input.command`, `session_id`, `prompt_id`, etc.).

Because the capturing session was a subagent, the platform attached `agent_id`/`agent_type` identifying it as one — fields the plan's own test design (Task 2: "another subtest setting agent_id in the payload") requires the *base* committed fixture to lack, since it must represent a delivering, top-level owner turn. I removed those two fields to represent a genuine top-level turn. This is not a guess: this repo's own prior real top-level capture (`cmd/testdata/stop-hook/menu-command-hide-screen-stop-payload.json`, captured for the Phase 205 Stop-hook work) shows the identical key is simply *absent* for a top-level session — I checked this directly before deciding. Every other edit to the fixture is the path redaction the plan specifies (home-directory absolute paths, and the capture-shim's temporarily-renamed binary path in `tool_input.command`, restored to the plain `aether` invocation actually intended).

I recorded this reasoning inline in `cmd/hook_direct_screen_test.go`'s doc comment on `TestDirectRouteHandsTheScreenToTheOwner` and in the fixture-generation comment in `cmd/hook_cmds.go`'s `ToolResponse` field, so the provenance is visible without reading this SUMMARY.

### Break-proofs performed (and reverted)

1. **Task 1 tracer:** Temporarily forced `directScreenDelivery` to `return "", false, false` unconditionally, reran `TestDirectRouteHandsTheScreenToTheOwner`, confirmed it failed ("expected exactly one line of stdout carrying the screen, got none"), then reverted and reran to confirm green.
2. **Task 2 backstop composition:** Temporarily forced `directScreenRouteRegistered` to `return true` unconditionally, reran `TestBackstopStaysQuietWhenTheScreenAlreadyArrived`, confirmed its "direct route not registered: still blocks" subtest failed, then reverted and reran to confirm green.

No stubs, overridable hooks, or test-only seams were left in production code for either break-proof — both were done by temporarily editing the source file directly and restoring it, matching the plan's own instruction.

## Known Stubs

None. Every code path implemented in this plan is real and reachable: the cobra command is registered, the settings entry is shipped, and the backstop composition is wired into the real `hookStopCmd` path.

## Self-Check: PASSED

- `cmd/hook_direct_screen.go` — FOUND
- `cmd/hook_direct_screen_test.go` — FOUND
- `cmd/claudemd_direct_screen_test.go` — FOUND
- `cmd/testdata/post-tool-use/status-screen-payload.json` — FOUND
- `git log --oneline --all | grep 96e49c14` — FOUND
- `git log --oneline --all | grep d1f26b73` — FOUND
- `git log --oneline --all | grep ec543cf3` — FOUND
- `git log --oneline --all | grep 4ca84280` — FOUND
- `git log --oneline --all | grep 714c861b` — FOUND
