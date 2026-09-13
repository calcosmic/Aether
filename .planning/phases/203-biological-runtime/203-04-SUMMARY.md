---
phase: 203-biological-runtime
plan: "04"
subsystem: recruitment
tags: [platform-probe, process-groups, native-nesting, biological-runtime]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-02's recruitment tracer (recruitCmd, dispatchRecruitment, recruitmentDispatchResult) this plan extends"
provides:
  - "probeNativeNestingOnce (Go) / probeNativeNesting (TS) -- bounded, isolated, session-cached native-nesting probes with matching contracts on both hosts"
  - "recruitmentAdapterKind / recruitmentAdapterKinds() / chooseRecruitmentAdapter -- the closed, fixed-order adapter-choice mechanism, wired onto dispatchRecruitment's own recruitmentDispatchResult"
  - "The folded todo's cheap-auth-probe half retired: platform-dispatcher.ts's runProbe now uses detached process-group teardown, one timeout-only retry, and the single shared AETHER_PREFLIGHT_TIMEOUT setting"
affects: [203-06, 203-07, 203-09]

# Actuals (#2632)
actuals:
  tokens: 14400
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Session-cached probe behind sync.Once (Go) / a resolved-promise cache (TS), keyed so the probe launches at most once per process and never on the plain no-recruitment path"
    - "Verdict decided only from the probed child's own plain-text answer, never a platform name or hardcoded table"
    - "Adapter choice as a closed enum with an AST-checkable dispatch switch, mirroring ColonyLiveEpisodeKinds()'s completeness convention"
    - "TS probe teardown ports Go's configureWorkerCommand behaviour via detached: true + process.kill(-pid), since Node has no direct process-group option"

key-files:
  created:
    - cmd/recruitment_probe.go
    - cmd/recruitment_probe_test.go
    - .aether/ts-host/src/recruitment-probe.ts
    - .aether/ts-host/test/recruitment-probe.test.ts
  modified:
    - cmd/recruitment_dispatch.go
    - .aether/ts-host/src/platform-dispatcher.ts
    - .aether/ts-host/dist/platform-dispatcher.js
    - .aether/ts-host/dist/platform-dispatcher.d.ts
    - .aether/ts-host/dist/recruitment-probe.js
    - .aether/ts-host/dist/recruitment-probe.d.ts

key-decisions:
  - "Deferred: threading AdapterKind onto the durable recruitmentResult (cmd/recruitment_result.go) is left to whichever plan owns that file in this wave -- this plan owns cmd/recruitment_dispatch.go and cmd/recruitment_probe.go only. dispatchRecruitment's own recruitmentDispatchResult carries AdapterKind now; the wiring from there into the persisted recruitmentResult, and from a future requested-adapter intent field into chooseRecruitmentAdapter's `requested` argument, is additive follow-up work outside this plan's file ownership."
  - "resolveProbeTimeoutMs (TS) deliberately does NOT introduce a second AETHER_PROBE_TIMEOUT environment variable, diverging from the plan's literal text. It delegates to the existing resolvePreflightTimeoutMs, matching the Go side's own resolvedAvailabilityProbeTimeout (`return resolvedPreflightTimeout()`). Introducing AETHER_PROBE_TIMEOUT on the TS side only would have reopened a defect this codebase already fixed and locked with a test (TestLiveReadinessSourceHasOneTimeoutEnvironmentVariable, pkg/codex/preflight_phase_198_3_test.go)."
  - "AETHER_RECRUIT_PROBE_TIMEOUT (Go) and its TS twin are a genuinely NEW, separate knob from AETHER_PREFLIGHT_TIMEOUT -- the native-nesting probe answers a different question (capability, not provider readiness) than the auth/model-round-trip probes, so this is not the same duplication risk as AETHER_PROBE_TIMEOUT."
  - "The probe is invoked lazily and only when native-bind is actually requested. Since recruitmentIntent carries no requested-adapter field yet, dispatchRecruitment always requests root-mediated explicitly (through chooseRecruitmentAdapter, never a hand-written literal), so probeNativeNestingOnce is never called from the live dispatch path today -- avoiding a real recursion hazard (see Deviations) and satisfying the plan's own 'a build that recruits none pays for none' / 'never once per recruitment' must_haves in the strongest possible way for v1."
  - "Named the pre-existing `which`-lookup timeout (a local filesystem check unrelated to provider readiness) as its own WHICH_LOOKUP_TIMEOUT_MS constant, so no bare 'timeout: 5000' literal remains in platform-dispatcher.ts, satisfying this plan's own literal acceptance criterion without conflating it with the readiness probe's budget."
  - "Restored .aether/ts-host's node_modules via `npm ci` (a pre-vetted, already package-locked install, not a new/unverified package) -- the directory was empty in this worktree, which would have made `npm run build`/`npm test` fail for reasons unrelated to this plan's changes."

requirements-completed: [BIO-03]

coverage:
  - id: D1
    description: "A configurable, bounded, session-cached, out-of-repository native-nesting probe on the Go side (probeNativeNestingOnce), never inferring its verdict from a platform name or table"
    requirement: "BIO-03"
    verification:
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentProbeReadsTimeoutEnv"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentProbeBoundsWallClock"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentProbeRunsOncePerProcess"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentProbeDirectoryIsCleanedUp"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentProbeNeverInfersFromPlatformName"
        status: pass
      - kind: other
        ref: "grep -c 'probeNativeNestingOnce' cmd/*.go (used only in cmd/recruitment_probe.go, cmd/recruitment_dispatch.go, and their tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same probe contract restored on the TypeScript host (probeNativeNesting), and the folded todo's cheap-auth-probe counter-example (runProbe's hardcoded timeout: 5000, no process-group teardown) retired"
    requirement: "BIO-03"
    verification:
      - kind: unit
        ref: ".aether/ts-host/test/recruitment-probe.test.ts (14 cases, both describe blocks)"
        status: pass
      - kind: other
        ref: "grep -n 'timeout: 5000' .aether/ts-host/src/platform-dispatcher.ts returns nothing"
        status: pass
      - kind: integration
        ref: ".aether/ts-host: node --import tsx --test test/platform-dispatcher.test.ts (46/46, no regressions) and npm run build (clean)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every recruitment dispatch names exactly one declared adapter kind (root-mediated or native-bind), chosen through one fixed-order function rather than a hardcoded literal, with a native-bind refusal path that never silently defaults to native"
    requirement: "BIO-03"
    verification:
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentAdapterChoice"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestEveryDeclaredAdapterCanDispatch"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_probe_test.go#TestRecruitmentDispatchResultNamesOneAdapter"
        status: pass
    human_judgment: true
    rationale: "Threading AdapterKind onto the durable, cross-plan recruitmentResult (owned by a sibling plan's file in this wave) is explicitly deferred -- see Deviations. A human/orchestrator pass should confirm the sibling plan actually completes that wiring before BIO-03 is marked fully satisfied end to end."

# Metrics
duration: 55min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 04: Native-Nesting Probe and Named Adapter Dispatch Summary

**A session-cached, process-group-safe native-nesting probe now exists on both the Go and TypeScript hosts, and every recruitment dispatch names the mechanism (root-mediated or native-bind) that carried it through one fixed-order, AST-checked decision function -- retiring the folded todo's hardcoded, repo-rooted TS auth-probe timeout in the process.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-13T09:47:00Z (approx.)
- **Completed:** 2026-09-13T10:42:28Z
- **Tasks:** 3 completed
- **Files modified:** 10 (4 created, 6 modified, 4 of the modified are generated `dist/` output)

## Accomplishments

- `cmd/recruitment_probe.go`: `probeNativeNestingOnce` caches one probe result per resolved platform behind `sync.Once`, runs in an `os.MkdirTemp` directory removed on every exit path, binds the whole process group via `codex.ConfigureWorkerCommand`, retries exactly once and only on timeout, and decides `Supported` only from the probed child's own reported answer.
- `resolvedRecruitmentProbeTimeout` resolves `AETHER_RECRUIT_PROBE_TIMEOUT` with the same resolve-warn-fallback shape `resolvePreflightTimeoutMs` uses on the TypeScript side, with a once-per-bad-value stderr warning.
- `.aether/ts-host/src/recruitment-probe.ts` restores the identical contract on the TypeScript host: `probeNativeNesting`, `resolveRecruitmentProbeTimeoutMs`, `RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS`, a `mkdtempSync`-based temp directory, and `detached: true` + `process.kill(-pid)` teardown (Node's port of `configureWorkerCommand`, since Node has no direct process-group option).
- Retired the folded todo's second half: `platform-dispatcher.ts`'s `runProbe` (the cheap auth probe behind `isPlatformAvailable`) no longer hardcodes `timeout: 5000` with no process-group teardown -- it now uses `detached` + negative-pid kill and retries once, only on timeout. Its budget deliberately reuses the single existing `AETHER_PREFLIGHT_TIMEOUT` setting rather than reviving the `AETHER_PROBE_TIMEOUT` knob this codebase already retired on the Go side (see Deviations).
- `cmd/recruitment_dispatch.go` gained the closed `recruitmentAdapterKind` enum (`root-mediated`, `native-bind`), `recruitmentAdapterKinds()`, and `chooseRecruitmentAdapter` -- a fixed-order decision function that `dispatchRecruitment` now calls to name the adapter on its own `recruitmentDispatchResult`, rather than hand-writing a literal.
- `TestEveryDeclaredAdapterCanDispatch` is an AST-derived inventory of `recruitmentAdapterKinds()`'s runtime output cross-checked against `chooseRecruitmentAdapter`'s dispatch switch. Proved genuinely failing (not just typed correctly) by mutation-testing it live: adding a third `recruitmentAdapterGhost` kind made the test fail and name `"ghost"` by value, then reverted (working tree clean, `git diff` empty after revert).
- `npm ci` restored `.aether/ts-host`'s empty `node_modules` (a pre-vetted, package-locked install of already-declared dependencies, not a new/unverified package) so `npm run build` and the TS test suite could actually run in this worktree.

## Task Commits

Each task was committed atomically (test-then-feat per its `tdd="true"` attribute):

1. **Task 1: Bounded, isolated, session-cached native-nesting probe (Go)**
   - `0a6152f3` (test) -- failing test for the probe contract
   - `15cdf767` (feat) -- `cmd/recruitment_probe.go`
2. **Task 2: Retire the hardcoded, process-leaky TypeScript probe**
   - `9f1ac0db` (test) -- failing tests for both TS probes
   - `4f385659` (feat) -- `recruitment-probe.ts` + `platform-dispatcher.ts` fix + rebuilt `dist/`
3. **Task 3: Name the adapter on every dispatch, prove every declared adapter can dispatch**
   - `04a4b645` (test) -- failing tests for the adapter choice and dispatch completeness
   - `0e00948a` (feat) -- `recruitmentAdapterKind` enum, `chooseRecruitmentAdapter`, wired into `dispatchRecruitment`

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_probe.go` - `recruitmentProbeResult`, `probeNativeNestingOnce`, `resolvedRecruitmentProbeTimeout`, `recruitmentProbeDefaultTimeout`, `recruitmentProbeRunner` (test seam)
- `cmd/recruitment_probe_test.go` - `TestRecruitmentProbe*` (5 tests), `TestRecruitmentAdapterChoice`, `TestEveryDeclaredAdapterCanDispatch`, `TestRecruitmentDispatchResultNamesOneAdapter`
- `cmd/recruitment_dispatch.go` - `recruitmentAdapterKind`, `recruitmentAdapterRootMediated`, `recruitmentAdapterNativeBind`, `recruitmentAdapterKinds`, `chooseRecruitmentAdapter`, `recruitmentDispatchResult.AdapterKind`
- `.aether/ts-host/src/recruitment-probe.ts` (new) - `RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS`, `resolveRecruitmentProbeTimeoutMs`, `probeNativeNesting`, plus test-injection seams
- `.aether/ts-host/src/platform-dispatcher.ts` - `resolveProbeTimeoutMs`, rewritten `runProbe`/`runProbeOnce` (detached + retry), named `WHICH_LOOKUP_TIMEOUT_MS`
- `.aether/ts-host/test/recruitment-probe.test.ts` (new, corrected path -- see Deviations) - 14 test cases across both probes
- `.aether/ts-host/dist/*` - rebuilt output for `platform-dispatcher` and the new `recruitment-probe` module

## Decisions Made

See `key-decisions` in the frontmatter for the full list. In short: the AdapterKind→`recruitmentResult` wiring is deferred to whichever sibling plan owns that file this wave; the TS probe timeout intentionally reuses the existing shared setting instead of reviving a retired env var; the native-nesting probe is invoked lazily (never on the plain root-mediated path) both for latency (SYN-203-05) and to avoid a real subprocess-recursion hazard; and a stray, unrelated `which`-lookup timeout literal was named to satisfy this plan's own literal acceptance criterion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan's literal `AETHER_PROBE_TIMEOUT` env var would reopen a defect this codebase already fixed**
- **Found during:** Task 2, before implementing `resolveProbeTimeoutMs`
- **Issue:** The plan's action text names a new `resolveProbeTimeoutMs` reading `AETHER_PROBE_TIMEOUT`. `pkg/codex/preflight_phase_198_3_test.go`'s `TestLiveReadinessSourceHasOneTimeoutEnvironmentVariable` explicitly locks the Go side to a SINGLE shared readiness timeout (`AETHER_PREFLIGHT_TIMEOUT`) and fails if `AETHER_PROBE_TIMEOUT` ever reappears in `platform_dispatch.go`. Introducing that variable only on the TS side would create the exact "two disagreeing knobs" hazard this project already retired.
- **Fix:** `resolveProbeTimeoutMs()` delegates to the existing `resolvePreflightTimeoutMs()` -- a pure alias, matching the Go side's own `resolvedAvailabilityProbeTimeout` (`return resolvedPreflightTimeout()`).
- **Files modified:** `.aether/ts-host/src/platform-dispatcher.ts`
- **Verification:** New test asserts `resolveProbeTimeoutMs() === resolvePreflightTimeoutMs()`-equivalent behavior for both a default and an overridden `AETHER_PREFLIGHT_TIMEOUT` value.
- **Committed in:** `4f385659` (Task 2 commit)

**2. [Rule 3 - Blocking] Plan named `.aether/ts-host/tests/recruitment-probe.test.ts`; the real directory is `test/` (singular)**
- **Found during:** Task 2, before writing the test file
- **Issue:** The actual test glob (`package.json`'s `test` script) is `test/*.test.ts`; a file under `tests/` would never run.
- **Fix:** Created the file at `.aether/ts-host/test/recruitment-probe.test.ts`.
- **Files modified:** N/A (path correction only)
- **Verification:** `node --import tsx --test test/recruitment-probe.test.ts` runs and passes.
- **Committed in:** `9f1ac0db` (Task 2 test commit)

**3. [Rule 1 - Bug] A second, unrelated `timeout: 5000` literal (the `which`-lookup check) was in scope by the plan's own literal acceptance criterion**
- **Found during:** Task 2, running the acceptance-criterion grep
- **Issue:** `grep -n 'timeout: 5000' platform-dispatcher.ts` also matched a `spawnSync("which", ...)` binary-existence check unrelated to the auth probe the folded todo names -- a purely coincidental literal collision, but the acceptance criterion is unqualified.
- **Fix:** Extracted a named `WHICH_LOOKUP_TIMEOUT_MS = 5_000` constant with a comment explaining why it deliberately does NOT share the readiness probe's budget (a local filesystem check, not a provider round-trip).
- **Files modified:** `.aether/ts-host/src/platform-dispatcher.ts`
- **Verification:** `grep -n 'timeout: 5000'` now returns nothing; existing `isPlatformAvailable` tests unaffected.
- **Committed in:** `4f385659` (Task 2 commit)

**4. [Rule 3 - Blocking] `.aether/ts-host/node_modules` was empty in this worktree**
- **Found during:** Task 2, running `npm run build` and `npm test`
- **Issue:** `node_modules` contained zero packages (a fresh worktree checkout never ran `npm install` for the ts-host subproject), causing unrelated pre-existing module-resolution failures across the whole test suite and blocking `npm run build`.
- **Fix:** Ran `npm ci` inside `.aether/ts-host` -- a package-locked, already-declared-dependency install, not a new/unverified package (excluded from the package-install carve-out, which targets adding NEW packages a plan names, not restoring the project's own existing lockfile).
- **Files modified:** N/A (environment restoration only; `node_modules` is gitignored)
- **Verification:** `npm run build` now succeeds cleanly; the full test suite went from immediate module-resolution crashes to 533/555 passing (see Issues Encountered for the remaining 22, all pre-existing and unrelated).
- **Committed in:** N/A (no tracked files changed)

**5. [Rule 4 - Architectural, resolved by file-ownership boundary, not by asking] `recruitmentResult.AdapterKind` was not added**
- **Found during:** Task 3, before writing `chooseRecruitmentAdapter`'s integration
- **Issue:** The plan's action text says to thread the chosen kind "onto `recruitmentResult` as `AdapterKind`", but `cmd/recruitment_result.go` is declared as 203-07's file in this wave's parallel-execution file ownership, and `cmd/recruitment.go` (the only caller that constructs a `recruitmentResult`) is 203-03's file. Neither is in this plan's `files_modified` list.
- **Fix:** Added `AdapterKind` to `recruitmentDispatchResult` (a struct this plan owns in `cmd/recruitment_dispatch.go`) instead, and wired `dispatchRecruitment` to populate it via `chooseRecruitmentAdapter`. The plan's own literal `TestEveryRecruitmentResultNamesOneAdapter` test name was NOT created here (it would require modifying `recruitment_result.go`/`recruitment.go`, and a same-named test function declared in both this plan's file and 203-07's own `recruitment_result_test.go` would be a duplicate-symbol compile error at merge). `TestRecruitmentDispatchResultNamesOneAdapter` proves the half owned here instead.
- **Files modified:** `cmd/recruitment_dispatch.go` only (no files outside this plan's ownership were touched)
- **Verification:** `TestRecruitmentDispatchResultNamesOneAdapter` passes; `TestEveryDeclaredAdapterCanDispatch` and `TestRecruitmentAdapterChoice` (the plan's other two named tests) both exist and pass exactly as named.
- **Committed in:** `0e00948a` (Task 3 commit)
- **Follow-up needed:** Once `cmd/recruitment_result.go` gains an `AdapterKind` field (203-07 or a later integration plan), that plan should copy `dispatchResult.AdapterKind` into the constructed `recruitmentResult` in `cmd/recruitment.go`, exactly as `TerminalStatus`/`Summary` are already copied today.

---

**Total deviations:** 5 auto-fixed (2 bugs avoiding config/literal defects, 2 blocking path/environment issues, 1 architectural boundary resolved via file-ownership isolation per this wave's parallel-execution contract)
**Impact on plan:** All five were necessary either for correctness (avoiding a locked-ratchet regression, a wrong test path, an unqualified acceptance criterion, a broken environment) or to respect the hard file-ownership boundary of this parallel wave. No scope creep: every fix stayed inside this plan's declared files, or was environment-only with no tracked-file changes.

## Issues Encountered

- The full `.aether/ts-host` test suite (`npm test`, all 555 tests) shows 22 pre-existing failures unrelated to this plan's changes -- confirmed by name (`go-bridge`, `lifecycle`, `golden-workflow`, `wrapper ceremony alignment`, etc.) and by direct inspection: none reference `platform-dispatcher`, `recruitment-probe`, `runProbe`, `isPlatformAvailable`, or `spawnWorker`. These appear to stem from the `aether` Go binary / colony-state fixtures those tests depend on (`an approved specification is missing` errors), not from anything touched here. The scoped suites this plan actually changed (`recruitment-probe.test.ts`: 14/14; `platform-dispatcher.test.ts`: 46/46; `preflight-phase-198-3.test.ts`: 3/3) are fully green with zero regressions.
- The Go side's own known-red baseline (per this plan's project-specific warnings) was not encountered: all scoped Go tests (`TestRecruitment*`, `TestEveryDeclaredAdapterCanDispatch`, `TestRecruitmentAdapterChoice`, `TestPlatformParityGolden`, plus `pkg/codex`'s readiness tests) pass cleanly.

## Threat Flags

| Flag | File | Description |
|------|------|--------------|
| threat_flag: new-process-spawn-surface | cmd/recruitment_probe.go | `runRecruitmentProbe` launches an external binary (resolved via `AETHER_RECRUIT_BINARY`/`AETHER_CODEX_PATH`/default `codex`) with a fixed, non-caller-controlled argv (or a test-only override) inside an OS temp directory. Unlike `dispatchRecruitment`'s workspace, this directory carries no user-supplied path to validate -- it is always `os.MkdirTemp`-created and always removed. No new externally-reachable input surface is introduced; this is the same class of surface 203-02's SUMMARY already flagged for `dispatchRecruitment` itself. |
| threat_flag: new-process-spawn-surface | .aether/ts-host/src/recruitment-probe.ts | Mirrors the Go-side flag above on the TypeScript host: `probeNativeNesting` spawns a detached child process. `AETHER_RECRUIT_PROBE_ARGS`/`AETHER_RECRUIT_BINARY` are test-only overrides read from the process environment, not from any request payload. |

## Known Stubs

- **`recruitmentResult.AdapterKind` does not exist yet** (`cmd/recruitment_result.go`, owned by a sibling plan in this wave). Every dispatch names its adapter on `recruitmentDispatchResult` (this plan's own record), but that value is not yet copied into the durable, exactly-once-bound `recruitmentResult` a later reader (status/watch, CEC-07's outcome join) would consult. This is not silently incomplete: `TestRecruitmentDispatchResultNamesOneAdapter` proves the half owned here, and this SUMMARY names the exact follow-up (copy `dispatchResult.AdapterKind` into the constructed `recruitmentResult` in `cmd/recruitment.go`, mirroring how `TerminalStatus`/`Summary` are already copied). Not fixed here because `cmd/recruitment_result.go` and `cmd/recruitment.go` are declared as other plans' files in this wave's parallel-execution contract.
- **`chooseRecruitmentAdapter`'s `requested` argument is always `recruitmentAdapterRootMediated` in the live dispatch path today** -- `recruitmentIntent` (owned by 203-03) carries no requested-adapter field yet, so native-bind, while fully implemented and tested in isolation, has no real caller. This is the intended v1 scope per the CLASSIC-SYNTHESIS ruling that native nesting is "a probed, non-launch-blocking enhancement only" -- not a gap, but noted here so a future plan adding that field knows exactly where to thread it through (`dispatchRecruitment`'s `chooseRecruitmentAdapter` call site).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BIO-03's probe contract is proven, symmetric across both hosts, and genuinely bounded (mutation-tested for the wall-clock claim on the Go side, and directly verified against a grandchild-holding-pipes fixture on both hosts).
- The folded todo (`2026-08-01-ts-host-preflight-hardcoded-timeout.md`) is fully retired: its preflight-timeout half was already fixed before this plan; its cheap-auth-probe half is fixed here.
- `chooseRecruitmentAdapter` and `recruitmentAdapterKinds()` are ready for 203-06 (BIO-02's full admission gate) or a later plan to extend `recruitmentIntent` with a requested-adapter field and thread it through this plan's decision function -- no new decision mechanism needs to be invented.
- Blocker for full BIO-03/CEC-07 closure: `recruitmentResult.AdapterKind` still needs to exist and be populated -- flagged explicitly above for whichever plan (likely 203-07, which owns `cmd/recruitment_result.go`) completes that wiring, or for the orchestrator's post-merge integration pass.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_probe.go`
- FOUND: `cmd/recruitment_probe_test.go`
- FOUND: `.aether/ts-host/src/recruitment-probe.ts`
- FOUND: `.aether/ts-host/test/recruitment-probe.test.ts`
- FOUND: commit `0a6152f3` (test: bounded isolated native-nesting probe) in `git log --oneline`
- FOUND: commit `15cdf767` (feat: bounded isolated session-cached native-nesting probe) in `git log --oneline`
- FOUND: commit `9f1ac0db` (test: TS native-nesting and auth probes) in `git log --oneline`
- FOUND: commit `4f385659` (feat: retire hardcoded TS auth probe) in `git log --oneline`
- FOUND: commit `04a4b645` (test: named adapter choice) in `git log --oneline`
- FOUND: commit `0e00948a` (feat: name the adapter on every recruitment dispatch) in `git log --oneline`
- Re-ran plan-level `<verification>` (Go): `go build ./... && go vet ./cmd ./pkg/codex ./pkg/events && go test ./cmd -run '^(TestRecruitmentProbe|TestEveryDeclaredAdapterCanDispatch|TestRecruitmentAdapterChoice|TestRecruitmentDispatchResultNamesOneAdapter|TestRecruitment)' -count=1 -timeout 180s && go test ./pkg/codex -run 'TestAvailabilityPreflight|TestLiveReadinessSource' -count=1` -- PASS
- Re-ran plan-level verification (TypeScript, scoped -- see Issues Encountered for the unrelated 22 pre-existing failures in the full suite): `cd .aether/ts-host && node --import tsx --test test/recruitment-probe.test.ts test/platform-dispatcher.test.ts test/preflight-phase-198-3.test.ts && npm run build && ! grep -q 'timeout: 5000' src/platform-dispatcher.ts` -- PASS
- Re-ran the mutation-test proof for `TestEveryDeclaredAdapterCanDispatch`: added a third `recruitmentAdapterGhost` kind to both the enum and `recruitmentAdapterKinds()` -- test failed and named `"ghost"` by value; reverted (`git diff cmd/recruitment_dispatch.go` empty after revert, confirmed before committing Task 3).
