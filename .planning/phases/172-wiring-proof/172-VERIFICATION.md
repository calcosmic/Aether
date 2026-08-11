---
phase: 172-wiring-proof
verified: 2026-08-11T16:56:42Z
status: gaps_found
score: 2/4 roadmap success criteria fully verified (criteria 1 and 2); criteria 3 and 4 each have a confirmed, reproduced defect
overrides_applied: 0
gaps:
  - truth: "Success criterion 3 — a test enumerates every `aether …` invocation in the in-scope `.aether` markdown corpus and fails naming the file, the line and the offending flag when an instruction names a flag the binary does not register"
    status: failed
    reason: >
      `.aether/docs/command-playbooks/continue-full.md` — squarely inside the audited corpus
      (`auditedCorpora`, in scope since before this phase, required by D-06) — contains a
      malformed fence closer at line 1194 (`  --ttl "30d"` immediately followed by the closing
      ``` fence marker on the SAME line, no newline). `extractDocumentedCalls`'s fence toggle
      (`strings.HasPrefix(strings.TrimSpace(line), "```")`) does not fire on that line because
      the line does not START with the marker, so it desyncs in-fence/out-of-fence parity for
      the rest of the file (confirmed: 149 fence-marker lines total, an odd count; parser state
      entering line 1243 is `inFence=false` when it should be `true`). The line at 1243,
      `midden_result=$(aether midden-recent-failures 50 2>/dev/null || echo '{"count":0,...}')`,
      passes a bare positional `50` to `midden-recent-failures`, whose real contract is
      `Args: cobra.NoArgs` plus `--limit` (`cmd/midden_cmds.go:20,486`) — the exact bug class
      (and, in fact, the exact same command) that plan 172-00 fixed at four sibling call sites
      in build-full.md, build-wave.md (x2) and continue-advance.md. This one occurrence was
      never fixed and is invisible to `TestCommandCallsMatchCobraContracts`, which still
      reports "audited 954 documented invocations across 6 corpora" with zero violations.
      Independently reproduced: calling `extractDocumentedCalls` directly on the file extracts
      zero calls at line 1243. This is not a hypothetical — it is a live, present-tense,
      unaudited, unfixed violation of exactly the shape success criterion 3 exists to catch,
      inside the corpus the criterion names. The phase's own 172-00-SUMMARY.md documents the
      identical bug shape (glued fence closer, same consequence) in the sibling file
      `continue-advance.md` and explicitly defers it (`deferred-items.md`) — but that
      documented finding was never generalized to check other files in the same corpus for the
      same defect, so this second, still-live instance was never found or disclosed by any
      plan in this phase.
    artifacts:
      - path: "cmd/command_call_audit_test.go"
        issue: "extractDocumentedCalls's fence-toggle test only recognizes a closing fence marker when it is the first non-whitespace content on its own line; a marker glued to the end of a content line does not toggle, silently desyncing fence parity for the remainder of the file and hiding every subsequent full-line invocation from parseFencedInvocation."
      - path: ".aether/docs/command-playbooks/continue-full.md"
        issue: "Line 1194 glues the closing ``` fence marker to the preceding content (`--ttl \"30d\"````), desyncing fence parity. Line 1243 (`midden-recent-failures 50`, a bare positional against a cobra.NoArgs + --limit command) is a live, unfixed, unaudited violation as a direct consequence."
    missing:
      - "Fix line 1194 in continue-full.md so the closing fence marker is on its own line (matching the fix pattern already applied to the sibling file continue-advance.md's line 503 per 172-00's deviation record), or make extractDocumentedCalls tolerant of a marker glued to trailing content."
      - "Fix .aether/docs/command-playbooks/continue-full.md:1243 to pass --limit 50 instead of the bare positional 50, matching the four sibling fixes already made by 172-00."
      - "Add a pinning test asserting the extractor does not desync fence parity when a closing marker is glued to content on the same line — the class of defect that let this specific violation hide, and that the phase's own deferred-items.md already flagged once for a sibling file without generalizing the fix."
      - "Sweep the rest of the audited corpus (auditedCorpora + auditedFiles) for the same glued-fence-marker shape to rule out further hidden violations before closing this gap."
  - truth: "Success criterion 4 — the ratchet and the flag test run in the same CI command the release gate already runs, verified by deleting a caller and observing the gate go red, not by reading the workflow file"
    status: partial
    reason: >
      The specific proof recorded in 172-05-SUMMARY.md (delete skill-create's only caller,
      run the named CI step's command verbatim, observe it fail naming skill-create, restore,
      observe it pass) is genuine and independently reproduced by both the orchestrator and
      this verification (TestDeletingACallerMakesTheRatchetNameIt passes with the same
      skill-create/skill-create.yaml pairing). That half of criterion 4 is solid.
      However, TestWiringGateStepRunsEveryWiringTest's own stated second purpose — "or if the
      blanket `go test ./...` release-gate step it sits alongside has been narrowed or
      removed" (the function's own doc comment) — is not actually enforced. Its check is
      `strings.Contains(workflow, "go test ./... -count=1 -timeout 900s")` applied to the
      WHOLE workflow file text, not scoped to a specific, failing step. That exact substring
      also appears inside the "Test summary" step (ci.yml:105), which runs `if: always()` and
      pipes its `go test` invocation through `grep -c '--- PASS' || echo 'unknown'` — a
      construction that cannot itself fail. Independently reproduced: with the real, blocking
      "Run Go tests" step (ci.yml:41-42) deleted entirely from the workflow file (and only that
      step deleted — the harmless "Test summary" step and its decoy substring left in place),
      `go test ./cmd -run TestWiringGateStepRunsEveryWiringTest -count=1 -v` still PASSES.
      This directly contradicts 172-05-SUMMARY.md's own threat-register claim ("T-172-18 ...
      Mitigated by the assertion that the blanket step is still present") — the assertion does
      not detect this exact narrowing. The change was reverted immediately after the
      reproduction (`git checkout HEAD -- .github/workflows/ci.yml`); working tree confirmed
      clean and the pre-existing 115-line `git status --porcelain` count was unchanged before
      and after.
    artifacts:
      - path: "cmd/ci_wiring_gate_test.go"
        issue: "TestWiringGateStepRunsEveryWiringTest's blanket-step-presence check is a bare substring match against the entire workflow file, satisfiable by an unrelated line inside a step that is structurally incapable of failing (if: always(), piped through || echo 'unknown'). It does not verify the blanket step is present AS A GATE, only that the string exists SOMEWHERE in the file."
    missing:
      - "Scope the blanket-step-presence check to the named step whose run: line is exactly `go test ./... -count=1 -timeout 900s` (matching the precision already used to locate the named wiring step by its `- name:` value), so a decoy occurrence elsewhere in the file cannot satisfy it."
      - "Alternatively, assert the containing step lacks `if: always()` and does not pipe its exit status through a fallback that can never fail, so the check verifies the step can actually gate the pipeline."
human_verification: []
---

# Phase 172: Wiring Proof Verification Report

**Phase Goal:** A capability added by this milestone cannot ship without a caller. The orphan
ratchet exists, runs in CI, and blocks — before any of the capabilities it constrains are
built, so it is shaped by the standard rather than by whatever shipped.

**Verified:** 2026-08-11T16:56:42Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP.md § Phase 172 Success Criteria, verbatim)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Registering a new cobra subcommand with no caller makes `TestNoRegisteredSubcommandIsUnreferenced` fail, naming the command; allowlist seeded from real scanner output (278 orphans, 6 tagged `owner_phase: "178"`); shrink-only against a committed baseline | ✓ VERIFIED | Independently re-ran `TestNoRegisteredSubcommandIsUnreferenced`, `TestOrphanAllowlistOnlyShrinks`, `TestRatchetDetectsASyntheticOrphan`, `TestDeletingACallerMakesTheRatchetNameIt` — all pass. Log line confirms "enumerated 405 registered commands, found 278 orphans", matching the SUMMARY's claim exactly. `TestDeletingACallerMakesTheRatchetNameIt` names `skill-create` / `.aether/commands/skill-create.yaml` at runtime, matching the SUMMARY's own transcript. |
| 2 | `aether spawn-can-spawn 5 --enforce` — the exact string `.aether/workers.md:292` documents — exits 0 | ✓ VERIFIED | Ran `go run ./cmd/aether spawn-can-spawn 5 --enforce` directly: `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0. Also confirmed the flag-form playbook invocation (`--depth 3`) still works. `.aether/workers.md:292` itself is byte-identical to before (per 172-03's own `git diff` confirmation, cross-checked by reading the current file). |
| 3 | A test enumerates every `aether …` invocation in `.aether/*.md` and fails naming file/line/flag when an instruction names a flag the binary does not register | ✗ FAILED | `.aether/docs/command-playbooks/continue-full.md:1243` is a live, unfixed, unaudited `midden-recent-failures 50` positional-argument violation (same bug class 172-00 fixed at four sibling sites) sitting inside the audited corpus, invisible because a malformed fence closer at line 1194 desyncs the extractor's fence-parity state for the rest of the file. `TestCommandCallsMatchCobraContracts` reports green (954 invocations, 0 violations) despite this. Independently reproduced by direct extraction (see below). |
| 4 | The ratchet and the flag test run in the same CI command the release gate already runs — verified by deleting a caller and observing the gate go red, not by reading the workflow file | ⚠️ PARTIAL | The specific behavioural proof (delete `skill-create`'s caller, run the named step's exact command, observe red, restore, observe green) is genuine and independently reproduced. However the companion test that is supposed to keep this guarantee durable over time (`TestWiringGateStepRunsEveryWiringTest`) does not actually detect deletion of the blanket `go test ./...` gate step — a decoy substring inside a non-blocking `if: always()` step satisfies its check. Independently reproduced: deleting the real gate step leaves the guard test green. |

**Score:** 2/4 fully verified; 1 failed; 1 partial (base proof solid, self-protecting durability test defective).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/spawn.go` | `spawn-can-spawn` accepts positional depth + `--enforce` with real deny semantics | ✓ VERIFIED | Confirmed by direct execution; `spawnCanSpawnDecision` seam present and reachable by test. |
| `cmd/spawn_enforce_test.go` | Tests for documented invocation + deny-to-exit wiring | ✓ VERIFIED | `TestSpawnCanSpawnAcceptsDocumentedInvocation`, `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` both pass. |
| `cmd/subcommand_reachability_ratchet_test.go` | Orphan ratchet + shrink-only guard + self-tests | ✓ VERIFIED | All 7 tests pass; 278-entry allowlist confirmed. |
| `cmd/testdata/orphan_allowlist.json` / `_baseline.json` | Shrink-only allowlist, seeded honestly | ✓ VERIFIED | 278 entries, byte-identical live/baseline, 6 tagged `owner_phase: "178"` confirmed via direct `python3 -c` count. |
| `cmd/command_call_audit_test.go` | Extractor sees `$(aether …)` calls; `.aether` corpus audited; catches unregistered flags | ⚠️ **HOLLOW in one confirmed spot** | `normalizeShellToken`/`isShellOperator` work correctly for the `$(` case they were built for (independently confirmed via unit probe). But the pre-existing fence-toggle logic this file relies on has a live blind spot (see gap 1) that lets a real violation inside the declared corpus go unaudited. |
| `.aether/docs/orphan-allowlist-policy.md` | Plain-English policy naming all four guarded files | ✓ VERIFIED | `TestAllowlistPolicyNamesEveryGuardedFile` passes; states 278 / 6 matching the real JSON. |
| `.github/workflows/ci.yml` | Named step running the wiring/flag guards alongside the blanket suite | ⚠️ **Guard test protecting it is incomplete** | Named step present and correctly filtered (21 tests). But `TestWiringGateStepRunsEveryWiringTest`'s blanket-step-presence assertion is satisfiable by a decoy substring (see gap 2). |
| `cmd/ci_wiring_gate_test.go` | `TestWiringGateStepRunsEveryWiringTest` keeps the named step's filter from falling behind | ⚠️ Partially effective | The filter-coverage half works correctly (independently reproduced: removing a test name from the regex fails the guard). The blanket-step-narrowing half does not (see gap 2). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `.aether/workers.md:292` | `cmd/spawn.go spawnCanSpawnCmd` | cobra resolution + `--enforce` flag | ✓ WIRED | Confirmed by direct execution and by `TestSpawnCanSpawnAcceptsDocumentedInvocation` reading the manual at runtime. |
| `cmd/subcommand_reachability_ratchet_test.go` | `rootCmd` | recursive `Commands()` walk | ✓ WIRED | 405 commands enumerated, matches SUMMARY. |
| `cmd/subcommand_reachability_ratchet_test.go` | `cmd/testdata/orphan_allowlist_baseline.json` | set-membership diff | ✓ WIRED | Independently proved shrink-only by re-running the existing self-tests (did not re-mutate the JSON files; relied on the phase's own recorded red/green transcripts plus a live pass of the guard tests). |
| `.github/workflows/ci.yml` named step | guard test files | `-run` regex, AST-enumerated | ✓ WIRED (coverage half) / ✗ **NOT WIRED (blanket-step-narrowing half)** | Coverage half independently reproduced red-then-green. Narrowing-detection half independently reproduced to silently pass when the real gate step is deleted. |

### Data-Flow Trace (Level 4)

Not applicable in the conventional sense — this phase produces static-analysis test infrastructure and CI configuration, not a UI or a data-rendering path. The equivalent trace performed here is the extractor's real read-path over real files, covered above and in the gaps.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `spawn-can-spawn 5 --enforce` exits 0 | `go run ./cmd/aether spawn-can-spawn 5 --enforce` | `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0 | ✓ PASS |
| `spawn-can-spawn --depth 3` still works | `go run ./cmd/aether spawn-can-spawn --depth 3` | `{"ok":true,"result":{"can_spawn":true,"depth":3}}`, exit 0 | ✓ PASS |
| Ratchet fires on synthetic orphan / real caller deletion | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestRatchetDetectsASyntheticOrphan\|TestDeletingACallerMakesTheRatchetNameIt'` | All PASS, 278 orphans, skill-create correctly named when its caller is suppressed | ✓ PASS |
| Extractor sees a real invocation inside the declared corpus | direct call to `extractDocumentedCalls` on `continue-full.md` | Returns 34 calls total but **zero at line 1243** (`midden-recent-failures 50`) — the exact defect class this phase exists to catch, still live | ✗ **FAIL** |
| Wiring-gate guard detects a deleted CI gate step | delete "Run Go tests" step from `ci.yml`, run `TestWiringGateStepRunsEveryWiringTest` | Test still **PASSES** | ✗ **FAIL** |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase or this repo; this is a Go-native testing project, not a migration/tooling phase with shell probes. `Step 7c` is SKIPPED (no runnable probe scripts declared or discovered).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| WIRE-01 | 172-02, 172-04 | A test fails when a registered subcommand has no caller outside its own definition, seeded with today's known orphans in an allowlist that may only shrink | ✓ SATISFIED | Ratchet exists, fires (synthetic + real deletion), shrink-only against baseline (both lists), all independently re-verified. |
| WIRE-02 | 172-01 | The documented invocation in `.aether/workers.md` matches a flag that exists — running the documented command succeeds rather than erroring on `--enforce` | ✓ SATISFIED | Directly executed; exits 0; deny path independently reachable via `spawnCanSpawnDecision` seam and its tests. |
| WIRE-03 | 172-00, 172-03, 172-04 | A test fails when a `.aether/*.md` instruction names a CLI flag the binary does not register | ✗ **BLOCKED** | The audit is not actually blind to the flags it was built to catch in principle, but it IS blind, today, to a live violation of exactly this shape inside the declared corpus (`continue-full.md:1243`), because of an unrelated but real fence-parsing bug. A requirement that "a test fails when ... a flag the binary does not register" is named is not satisfied while a real such instance sits undetected. |

REQUIREMENTS.md lines 139-141 list all three as "Pending" status in that table (not yet flipped to a completion marker), which is consistent with this verification's outcome — WIRE-03 should not be marked complete.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.aether/docs/command-playbooks/continue-full.md` | 1194, 1243 | Malformed fence closer glued to content; unfixed `midden-recent-failures 50` bare-positional call hidden by it | 🛑 Blocker | Live, unaudited violation of the exact class WIRE-03 exists to catch. |
| `cmd/ci_wiring_gate_test.go` | 67 | `strings.Contains(workflow, ...)` unscoped substring check, satisfiable by a decoy line in a step that cannot fail | 🛑 Blocker | Undermines the durability half of criterion 4 / D-15's stated guarantee (T-172-18), though the base behavioural proof for criterion 4 is genuine. |
| `cmd/cli_flag_audit_test.go` | 111-115 | `TestCLIFlagAudit`'s `os.ReadDir(dir)` error path silently `continue`s with no anti-vacuity `t.Fatal`, unlike every other guard test this phase added | ⚠️ Warning | Pre-existing (predates phase 172); not exploited today (directories exist and are read), but is now load-bearing for the shrink-only skip-list guard built on top of it in 172-04. Worth hardening for consistency with the phase's own stated anti-vacuity discipline. |
| `cmd/command_call_audit_test.go` | 130-142 (`openedSubstitution`) | Only recognises `$(` as an opener for trailing-`)` trimming; a bare `(aether …)` subshell form is silently dropped entirely (independently reproduced: `ok=false`), and a backtick-substitution form (`` x=`aether cmd` ``) is recognised but its last token retains a stray trailing backtick | ⚠️ Warning | Confirmed via direct probe. Not currently exploited — a repo-wide grep for both shapes across every audited corpus found zero real occurrences today — but it is a real, latent completeness gap in the extractor this phase is centrally about. |
| `cmd/subcommand_reachability_ratchet_test.go` | 1176-1180 (`TestWiringGuardsHaveNoRuntimeEscapeHatch`) | `guardFiles` lists only 3 of the 5 guard files this phase created (`spawn_enforce_test.go` and `ci_wiring_gate_test.go` are absent), and `forbiddenRe` does not cover `os.LookupEnv`, `os.Environ`, or `testing.Short()` | ℹ️ Info | Disclosed by the phase's own 172-05-SUMMARY.md as a known, narrow gap. Confirmed both missing files are currently clean of any escape-hatch pattern. Not currently exploited. |
| `cmd/subcommand_reachability_ratchet_test.go` + `cmd/cli_flag_audit_test.go` | — | Two shrink-only set-membership diffs (orphan allowlist, flag skip-list) implemented independently rather than via a shared comparator | ℹ️ Info | Disclosed by 172-04-SUMMARY.md as a deliberate, documented follow-up. Not a functional defect today — both were independently red/green-tested. |

No `TBD`, `FIXME`, or `XXX` markers found in any file this phase modified.

### Human Verification Required

None. Every claim above was checked by running the actual code and tests, not by reading documentation or trusting SUMMARY narration.

### Gaps Summary

Two of the four roadmap success criteria are not fully met, and both failures are independently
reproduced against the live codebase, not inferred from the code review:

1. **Criterion 3 is false as stated.** The flag/call audit does not, in fact, "enumerate every
   `aether …` invocation" in its declared corpus — `.aether/docs/command-playbooks/continue-full.md`
   contains a malformed fence closer that silently desyncs the parser's fence-tracking state,
   and a live, unfixed `midden-recent-failures 50` positional-argument violation sits invisible
   inside that desynced region. This is not a theoretical edge case: it is the exact bug class
   (and the exact command) that plan 172-00 fixed at four sibling call sites in the same
   milestone, in a sibling file with the identical fence-parity defect that was found and
   explicitly deferred — but never checked for elsewhere in the same corpus. The audit reports
   green while a real instance of the bug it exists to catch sits inside its own declared scope.

2. **Criterion 4's durability guarantee is false as claimed.** The behavioural proof recorded in
   172-05-SUMMARY.md (delete a caller, watch the named CI step go red) is real and reproduces
   cleanly. But the test written to keep that guarantee from eroding over time —
   `TestWiringGateStepRunsEveryWiringTest`, whose own doc comment states it exists partly to
   catch the blanket release-gate step being "narrowed or removed" — does not actually detect
   that removal. Its check is an unscoped substring search across the whole workflow file, and
   an identical substring exists inside a step (`Test summary`) that is structurally incapable
   of failing. Deleting the real, blocking "Run Go tests" step leaves this guard green.

Both gaps are grouped by a common root cause worth flagging to the closure plan: this phase's
guard tests are, in several places, doing exactly what CLAUDE.md's "Definition of Done" warns
against — a check that reads as protection but is satisfiable without the thing it claims to
verify actually being true. The phase caught and fixed several instances of this pattern in the
*target* documentation (the four `midden-recent-failures` fixes, the `swarm-display-update`
fixes) but two instances of the same pattern survive inside the *guards themselves*.

Neither gap is deferred to a later milestone phase — no phase in 173-179 names markdown
fence-parsing correctness or CI-gate self-protection as in scope, so both remain live, actionable
gaps for this phase (or an immediate follow-up plan) rather than items a later phase will pick up.

---

_Verified: 2026-08-11T16:56:42Z_
_Verifier: Claude (gsd-verifier)_
