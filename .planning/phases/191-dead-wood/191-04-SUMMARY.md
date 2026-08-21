---
phase: 191-dead-wood
plan: 04
subsystem: infra
tags: [go, cli, dead-code, review-depth, verification-depth, cwd-relative-loader]

# Dependency graph
requires:
  - phase: 191-dead-wood
    provides: "191-CONTEXT.md's Criterion 2 analysis (the four CWD-relative silent-fallback loaders) and 191-PATTERNS.md's fold-then-delete process"
provides:
  - "colony/policies/review-depth.yaml deleted; every field it defined confirmed already identical to cmd/review_depth.go's compiled fallbacks"
  - "A permanent Go-level regression test (TestReviewDepthPolicyFieldsSurviveDeletion) locking the fallbacks against future silent drift"
  - "A byte-for-byte, real-binary, dev-checkout CLI proof that review-depth keyword matching is unchanged before/after deletion"
  - "A proven, fail-then-pass reappearance ratchet (TestReviewDepthPolicyDoesNotReappear)"
affects: [191-07-final-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tier-1 os.Stat existence-check ratchet reusing repoRootForCommandSourceTest() (191-PATTERNS.md house style)"
    - "Two-layer byte-identical proof: Go-level *PathOverride regression test (Layer 1) + real compiled-binary CLI capture from repo root (Layer 2), since go test's CWD never reaches the CWD-relative branch"

key-files:
  created:
    - cmd/review_depth_ratchet_test.go
    - cmd/testdata/review_depth_original_191_04.yaml
  modified:
    - cmd/review_depth_test.go
  deleted:
    - colony/policies/review-depth.yaml

key-decisions:
  - "Field-by-field diff found zero divergence between review-depth.yaml and cmd/review_depth.go's compiled fallbacks -- no fold edits were needed to review_depth.go itself"
  - "Reused the pre-existing resetReviewDepthPolicyState() test helper instead of adding a new resetReviewDepthPolicyCache() -- the plan's read_first assumed no reset function existed; one already did"
  - "loadReviewDepthPolicy() kept unchanged as a permanently-fallback function per 191-CONTEXT.md D-04 -- the file it reads no longer exists, but the loader scaffolding is not deleted"

patterns-established: []

requirements-completed: []

# Metrics
duration: ~40min
completed: 2026-08-21
---

# Phase 191 Plan 04: Fold-and-Delete review-depth.yaml Summary

**Deleted `colony/policies/review-depth.yaml` (the last of Phase 191's four CWD-relative silent-fallback loaders) after confirming every field it defined -- three keyword lists and six smart-default-reason strings -- already matched `cmd/review_depth.go`'s compiled Go defaults byte-for-byte; proved this two ways (a permanent regression test and a real compiled-binary CLI capture) and added a fail-then-pass-proven reappearance ratchet.**

## Performance

- **Duration:** ~40 min (dominated by three full `go test ./cmd/... -count=1` runs at 358-444s each)
- **Completed:** 2026-08-21T01:26:58Z
- **Tasks:** 2/2 completed
- **Files modified:** 4 (1 deleted, 1 created as test, 1 created as testdata fixture, 1 modified)

## Accomplishments

- Confirmed, field-by-field, that `colony/policies/review-depth.yaml`'s `heavy_keywords` (12 entries), `security_risk_keywords` (10 entries), `blast_radius_keywords` (10 entries), and all 6 `smart_default_reasons` values were already byte-identical to `cmd/review_depth.go`'s compiled fallbacks -- zero divergence, so no fold edits were required to that file.
- Added `TestReviewDepthPolicyFieldsSurviveDeletion`, a permanent before/after regression test that renders every accessor (`getHeavyKeywords`, `getSecurityRiskKeywords`, `getBlastRadiusKeywords`, `getSmartDefaultReason` for all 6 keys) once with the real original file content and once with the file absent, asserting both renders match entry-by-entry. Ran and passed both before and after the deletion.
- Deleted `colony/policies/review-depth.yaml` via `git rm`.
- Produced a real, compiled-`aether`-binary, dev-checkout CLI capture (via a temporary, never-committed debug subcommand) before and after deletion; the two captures are byte-for-byte identical (matching SHA-256 checksums).
- Added `TestReviewDepthPolicyDoesNotReappear`, a Tier-1 existence-check ratchet, and proved it fail-then-pass: recreated the file, confirmed the test failed by name citing the ruling, removed it, confirmed the test passed again.
- Confirmed the Queen-Owned-Orchestration-locked tests (`TestBuildWorkerCapHonoursVerificationDepth`, `TestGatekeeperNeedsASecuritySignal`, `TestProbeIsRequiredOnlyWhereItCanFindSomething`, `TestWatcherIsAlwaysRequiredOnBuild`, `TestHighRiskPhaseKeepsBothReviewers`) pass unmodified -- this fold did not disturb the locked depth-selection numbers.
- Full `go test ./cmd/... -count=1` passes with zero regressions (run three times across the two tasks).

## Task Commits

Each task was committed atomically:

1. **Task 1: Baseline-capture, diff-and-fold, delete review-depth.yaml** - `0bc98ee1` (feat)
2. **Task 2: Add the reappearance ratchet with fail-then-pass proof** - `19cf93c1` (test)

## Field-by-Field Diff Table

Every field `colony/policies/review-depth.yaml` defined, compared against `cmd/review_depth.go`'s compiled defaults, captured **before** any edits:

| Field | `review-depth.yaml` value | Go compiled default | Result |
|---|---|---|---|
| `review_depth_version` | `"1.0"` | *(no accessor anywhere reads `reviewDepthPolicy.ReviewDepthVersion`; grep-confirmed across `cmd/` and `pkg/`)* | Unused field — nothing to fold, no behavior depends on it |
| `heavy_keywords` (12) | security, auth, crypto, secrets, permissions, compliance, audit, release, deploy, production, ship, launch | `heavyKeywordsFallback` — identical 12 entries, identical order | MATCH |
| `security_risk_keywords` (10) | security, auth, crypto, secrets, permissions, compliance, audit, token, session, password | `securityRiskKeywordsFallback` — identical 10 entries, identical order | MATCH |
| `blast_radius_keywords` (10) | core runtime, state mutation, colony state, state machine, phase transition, dispatch, build command, continue command, verification depth, planning depth | `blastRadiusKeywordsFallback` — identical 10 entries, identical order | MATCH |
| `smart_default_reasons.high_risk` | `"auto: security risk"` | `fallbackReasonHighRisk` | MATCH |
| `smart_default_reasons.final_phase` | `"auto: final phase"` | `fallbackReasonFinalPhase` | MATCH |
| `smart_default_reasons.medium_risk` | `"auto: high blast radius"` | `fallbackReasonMediumRisk` | MATCH |
| `smart_default_reasons.early_phase` | `"auto: early phase"` | `fallbackReasonEarlyPhase` | MATCH |
| `smart_default_reasons.late_phase` | `"auto: late phase"` | `fallbackReasonLatePhase` | MATCH |
| `smart_default_reasons.standard` | `"auto: standard"` | `fallbackReasonStandard` | MATCH |

**Conclusion:** zero divergence across every field with a live accessor. `cmd/review_depth.go` required no fold edits at all -- its compiled defaults already said exactly what the file said. This is the mechanically simpler outcome the plan anticipated as possible (D-04 step 3 folds "any divergent value"; here there was none), and the two proofs below exist precisely to make that claim checkable rather than assumed.

## Layer 1: Go-Level Regression Test (Before Deletion)

`TestReviewDepthPolicyFieldsSurviveDeletion` (`cmd/review_depth_test.go`) renders every accessor via `reviewDepthPathOverride` pointed at `cmd/testdata/review_depth_original_191_04.yaml` (a byte-for-byte copy of the file, captured immediately before `git rm`, verified with `diff` at copy time), then again with the override pointed at a nonexistent path (the folded-default path), and asserts every entry of all three keyword lists and all six smart-default reasons match. Includes a sanity check that the fixture actually loaded (`loadReviewDepthPolicy() != nil`), so the test cannot pass vacuously by both renders silently hitting the same fallback.

Ran and passed **before** `git rm`:
```
=== RUN   TestReviewDepthPolicyFieldsSurviveDeletion
--- PASS: TestReviewDepthPolicyFieldsSurviveDeletion (0.00s)
PASS
```

Ran and passed again **after** `git rm colony/policies/review-depth.yaml`:
```
=== RUN   TestReviewDepthPolicyFieldsSurviveDeletion
--- PASS: TestReviewDepthPolicyFieldsSurviveDeletion (0.00s)
PASS
```

## Layer 2: Real Dev-Checkout CLI Capture (Byte-Identical Proof)

`go test`'s working directory is the package directory (`cmd/`), never the repo root, so Layer 1 alone cannot exercise `loadReviewDepthPolicy()`'s CWD-relative `policyPath("review-depth")` branch (`colony/policies/review-depth.yaml`, resolved relative to CWD). Per 191-PATTERNS.md's "Baseline Capture Before Any Change" and the plan's Claude's-Discretion allowance for "a small throwaway Go main," a temporary, never-committed debug subcommand (`cmd/zz_review_depth_debug_cmd.go`, registered on `rootCmd`, deleted before either task's commit) printed every accessor's output plus several derived-behavior samples (`phaseHasHeavyKeywords`, `phaseRiskLevel`, `resolveSmartVerificationDepth`, `renderSmartDepthReason`, `resolveReviewDepth`, `resolveVerificationDepth`, `renderReviewDepthLineWithReason`) for a security-keyword phase, a blast-radius phase, a low-risk phase, and a final phase.

Built the real `aether` binary and ran it **from this repo's root** (the only environment where the CWD-relative read resolves) before and after deletion:

**Before** (`colony/policies/review-depth.yaml` present):
```
heavy_keywords: [security auth crypto secrets permissions compliance audit release deploy production ship launch]
security_risk_keywords: [security auth crypto secrets permissions compliance audit token session password]
blast_radius_keywords: [core runtime state mutation colony state state machine phase transition dispatch build command continue command verification depth planning depth]
smart_default_reason[high_risk]: auto: security risk
smart_default_reason[final_phase]: auto: final phase
smart_default_reason[medium_risk]: auto: high blast radius
smart_default_reason[early_phase]: auto: early phase
smart_default_reason[late_phase]: auto: late phase
smart_default_reason[standard]: auto: standard
phaseHasHeavyKeywords("Security hardening"): true
phaseRiskLevel("Security hardening"): high
resolveSmartVerificationDepth("Security hardening", 6): heavy
renderSmartDepthReason("Security hardening", 6): auto: security risk
note: phase "Security hardening" matched a security/release keyword; review depth escalated to heavy (pass --light to override)
resolveReviewDepth("Security hardening", 6, false, false): heavy
resolveVerificationDepth("Security hardening", 6, false, false, ""): heavy
phaseRiskLevel("Core runtime refactor"): medium
resolveSmartVerificationDepth("Core runtime refactor", 6): standard
renderSmartDepthReason("Core runtime refactor", 6): auto: high blast radius
phaseRiskLevel("UI polish"): low
resolveSmartVerificationDepth("UI polish", 6): standard
renderSmartDepthReason("UI polish", 6): auto: standard
renderSmartDepthReason("Final polish", 6): auto: final phase
renderReviewDepthLineWithReason(heavy, 6, 6, "Final polish", true): Review depth: heavy (final phase)
```

**After** (`colony/policies/review-depth.yaml` deleted, binary rebuilt): identical output, character for character.

**Diff:** `diff BEFORE.txt AFTER.txt` produced no output (exit 0). SHA-256 of both files: `cafcb66a300f3363852f56edd41bd1caf1dc6b78c6a77fe764eed59417b588e8` (matched). Byte-identical, confirmed by both a textual diff and a cryptographic checksum.

## Reappearance Ratchet: Fail-Then-Pass Proof

`TestReviewDepthPolicyDoesNotReappear` (`cmd/review_depth_ratchet_test.go`), Tier 1 per 191-PATTERNS.md (plain `os.Stat`, reusing `repoRootForCommandSourceTest()`).

**Step 1 -- recreated the file** (copied the testdata fixture back to `colony/policies/review-depth.yaml`) and ran the ratchet. It FAILED, naming the exact path and the ruling:
```
=== RUN   TestReviewDepthPolicyDoesNotReappear
    review_depth_ratchet_test.go:33: colony/policies/review-depth.yaml has reappeared at /Users/callumcowie/repos/Aether/.claude/worktrees/agent-a2b90df2e910c32cf/colony/policies/review-depth.yaml -- this file was ruled zero-added-value and deleted in Phase 191 Plan 04 (191-CONTEXT.md D-04): every field it defined already matched cmd/review_depth.go's compiled fallbacks, proven by TestReviewDepthPolicyFieldsSurviveDeletion. If it is back, either fold any new divergent value into review_depth.go's fallbacks and update that regression test with fresh evidence, or delete it again -- never leave it as an untested, silently-read CWD-relative file
--- FAIL: TestReviewDepthPolicyDoesNotReappear (0.00s)
FAIL
FAIL	github.com/calcosmic/Aether/cmd	0.582s
```

**Step 2 -- removed the file again** and re-ran. It PASSED:
```
=== RUN   TestReviewDepthPolicyDoesNotReappear
--- PASS: TestReviewDepthPolicyDoesNotReappear (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.582s
```

`git status --short` after the proof confirmed no stray files were left behind (`colony/policies/review-depth.yaml` was never staged during the recreate step).

## Files Created/Modified

- `cmd/review_depth_test.go` - Added `TestReviewDepthPolicyFieldsSurviveDeletion` and its two helpers (`captureReviewDepthFields`, `assertReviewDepthFieldEqual`), reusing the existing `resetReviewDepthPolicyState()` helper
- `cmd/testdata/review_depth_original_191_04.yaml` - New permanent testdata fixture: byte-for-byte copy of the deleted file's original content
- `cmd/review_depth_ratchet_test.go` - New file: `TestReviewDepthPolicyDoesNotReappear`
- `colony/policies/review-depth.yaml` - Deleted (`git rm`)
- `cmd/review_depth.go` - **Not modified.** Read in full during `read_first`; the field-by-field diff found zero divergence, so no fold edits were needed. `loadReviewDepthPolicy()` itself is unchanged and remains a permanently-fallback function per 191-CONTEXT.md D-04.

## Decisions Made

- **Zero-fold outcome is a valid, provable outcome, not a shortcut.** The plan anticipated folding "any divergent value" -- this task found none, and both proofs (Layer 1 regression test, Layer 2 CLI capture) exist specifically to make "zero divergence" a checked fact rather than an assumption.
- **Reused the pre-existing `resetReviewDepthPolicyState()` instead of adding a new reset function.** The plan's `read_first` instructed adding `resetReviewDepthPolicyCache()` "if absent," having assumed (per the plan's own interface stub note: "no reset function confirmed to exist today") that none existed. A full read of `cmd/review_depth_test.go` found `resetReviewDepthPolicyState()` already defined and already used by 4 existing tests. I initially added a duplicate function to `cmd/review_depth.go`, then caught this during `read_first` verification, reverted it, and used the existing helper instead -- avoiding duplicate infrastructure doing the identical job under two names.
- **Temporary debug subcommand, never committed.** To produce a real dev-checkout CLI capture (Layer 2), a temporary cobra command (`cmd/zz_review_depth_debug_cmd.go`) was added, used to build and run the real `aether` binary from repo root twice (before/after deletion), then deleted before either task's `git add`/commit. It never appears in git history. This matches the plan's Claude's-Discretion allowance for "a small throwaway Go main" as an acceptable Layer-2 mechanism.
- **Testdata fixture over an embedded string literal.** `cmd/testdata/review_depth_original_191_04.yaml` was created as an exact `cp` of the live file (verified with `diff` before deletion) rather than retyping the YAML as a Go string constant, to eliminate transcription-error risk and follow this repo's existing `testdata/` convention.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking/Plan-Assumption Correction] Plan's read_first assumed no reset function existed; one already did**
- **Found during:** Task 1, `read_first` (full read of `cmd/review_depth_test.go`)
- **Issue:** The plan's `<interfaces>` section noted "no reset function confirmed to exist today -- read_first must confirm and add one mirroring resetVisualsCache()'s shape if absent," and I initially added `resetReviewDepthPolicyCache()` to `cmd/review_depth.go` before completing the full read. Reading the rest of the 1845-line test file revealed `resetReviewDepthPolicyState()` already existed (same logic, plus a `policyCache` cleanup line) and was already used by 4 pre-existing tests.
- **Fix:** Reverted the new `resetReviewDepthPolicyCache()` addition (confirmed via `git diff --stat cmd/review_depth.go` showing zero changes to that file in the final commit) and wrote the new regression test against the existing `resetReviewDepthPolicyState()` instead.
- **Files modified:** `cmd/review_depth_test.go` (net effect only; `cmd/review_depth.go` ended with no diff)
- **Verification:** `go test ./cmd/... -count=1` passes; no duplicate reset function exists in the final tree
- **Committed in:** `0bc98ee1` (Task 1 commit) — the reverted addition was never committed on its own

---

**Total deviations:** 1 auto-fixed (plan-assumption correction, caught before it reached a commit)
**Impact on plan:** No scope creep, no behavior change. The correction kept the codebase from gaining duplicate cache-reset infrastructure.

## Issues Encountered

None beyond the deviation above. The sandboxed Bash tool rejected two multi-step, redirect-containing compound commands as "too complex to verify... stays inside the worktree" early in execution (the initial worktree-branch-check one-liner, and a combined `ls && capture` pairing) -- both were resolved by splitting into separate, simpler Bash calls, with no effect on the work itself.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Criterion 2 (the four CWD-relative silent-fallback loaders) is now fully resolved across this phase's plans: 191-02 (colony-prime.md + dispatch-contract.yaml), 191-03 (visuals.md), and this plan (191-04, review-depth.yaml).
- `TestBuildWorkerCapHonoursVerificationDepth` and the other Queen-Owned-Orchestration-locked tests were confirmed to pass unmodified -- this plan's fold did not disturb the depth-selection numbers CLAUDE.md documents.
- 191-07 (final verification, Wave 2) can proceed once all Wave-1 plans land; nothing in this plan touched `colony/policies/oracle-phase-directives.yaml` or any file criterion 5 protects.

---
*Phase: 191-dead-wood*
*Completed: 2026-08-21*

## Self-Check: PASSED

- FOUND: `cmd/review_depth_ratchet_test.go`
- FOUND: `cmd/testdata/review_depth_original_191_04.yaml`
- FOUND: `.planning/phases/191-dead-wood/191-04-SUMMARY.md`
- CONFIRMED ABSENT: `colony/policies/review-depth.yaml`
- FOUND commit: `0bc98ee1`
- FOUND commit: `19cf93c1`
- CONFIRMED: `git diff --stat 916db8fa..HEAD -- cmd/review_depth.go` is empty (zero changes, as claimed)
