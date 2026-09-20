---
phase: 172-wiring-proof
plan: "08"
subsystem: testing
tags: [go, static-analysis, ci-gate, wiring-proof, gap-closure]

# Dependency graph
requires:
  - phase: 172-04
    provides: "the flag audit's shrink-only skip-list guard (D-12), which this plan makes rest on a non-vacuous TestCLIFlagAudit"
  - phase: 172-06
    provides: "the fence-tolerant extractor and its exact-count fixture, which this plan extends with three more substitution-form cases"
provides:
  - "substitutionOpener(tok string) (open, closeDelim string): one function owning both the substitution-opener recognition and its matching closing delimiter, replacing the narrower openedSubstitution which recognised only \\$("
  - "Three pinned fixture cases (backtick-substitution assignment, bare-subshell call, bare-subshell call with a trailing flag value) in TestCommandCallExtractorSeesRealInvocationsAndSkipsProse, with a mandatory red proof showing all three failing before the fix"
  - "TestCLIFlagAudit resolved through repoRootForCommandSourceTest() instead of \`..\`-relative paths, with loud per-directory/per-file failures and a scannedFiles/foundSubcommands anti-vacuity floor (>= 120 files, > 0 subcommands)"
  - "flag-audit mismatch failures name a filepath.Rel-derived repo-relative path instead of a bare basename"
affects: [wiring-proof, "172-*", any future phase reading TestCLIFlagAudit as a non-vacuous guard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One function owning both halves of a decision (opener recognition + its matching closer) rather than two functions that can independently drift, mirroring 172-06's shared processDocumentedCallLine pattern"
    - "Anti-vacuity floor pattern from cmd/subcommand_reachability_ratchet_test.go (TestNoRegisteredSubcommandIsUnreferenced's \"enumerated only N registered commands\" guard) applied to a second guard (TestCLIFlagAudit)"
    - "Per-item t.Errorf (not t.Fatalf) inside a multi-directory scan loop, so every declared directory is attempted and every failure is visible, with a separate terminal t.Fatalf for the aggregate anti-vacuity floor — see Deviations"

key-files:
  created: []
  modified:
    - cmd/command_call_audit_test.go
    - cmd/cli_flag_audit_test.go

key-decisions:
  - "Per-directory and per-file read failures in TestCLIFlagAudit use t.Errorf (continue scanning) rather than the plan action text's literal t.Fatalf, with a separate t.Fatalf for the post-loop anti-vacuity floor. This is a deliberate deviation from the plan's literal wording, made to satisfy the plan's own two mandatory red proofs, which are structurally incompatible with a single per-directory t.Fatalf (Go's t.Fatalf halts the test goroutine at the first call, so an all-three-missing scenario could never reach or display the anti-vacuity floor's scannedFiles/foundSubcommands counts if the first missing directory always aborted the test immediately). See Deviations for the full reasoning."

requirements-completed: [WIRE-01, WIRE-03]

# Metrics
duration: 25min
completed: 2026-08-11
---

# Phase 172 Plan 08: Close the Two Anti-Vacuity Findings Summary

**Unified the extractor's substitution-opener recognition with its closing-delimiter trim into one `substitutionOpener` function (fixing a silent drop of backtick and bare-subshell invocations), and made `TestCLIFlagAudit` fail loudly instead of vacuously passing when its declared corpus directories are unreadable or empty.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-11T21:15:00+02:00 (approx, first read)
- **Completed:** 2026-08-11T21:36:14+02:00
- **Tasks:** 2
- **Files modified:** 2 (excludes this SUMMARY)

## Accomplishments

- `normalizeShellToken` strips three substitution openers (`$(`, backtick, bare `(`), but the trim-the-closing-delimiter decision (`openedSubstitution`) recognised only `$(` — so a backtick substitution's command name carried a stray trailing backtick and a bare-subshell call's trailing token carried a stray trailing `)`, both silently dropped or misreported (CR-05). `substitutionOpener` now returns the opener AND its matching closer as one decision; both trim call sites use it; the narrower `openedSubstitution` is gone entirely.
- Three new fixture cases in `TestCommandCallExtractorSeesRealInvocationsAndSkipsProse` pin all three forms: a backtick-substitution assignment naming `skill-list`, a bare-subshell call to `colony-name`, and a bare-subshell call to `midden-recent-failures --limit 5` (also asserted clean against `validateCallAgainstCobra`, since that command really does accept `--limit`). The audited live-corpus invocation count is unchanged at 988 — the fix touches only forms the corpus does not currently use.
- `TestCLIFlagAudit` no longer declares its three corpus directories as `..`-relative strings and no longer silently `continue`s past an unreadable directory or file. It resolves through `repoRootForCommandSourceTest()`, fails loudly (naming the directory/file and the error) on any unreadable declared input, and carries a new anti-vacuity floor (`scannedFiles >= 120`, `foundSubcommands` non-empty against a measured 141 files / 126 subcommands) so the test cannot pass while reading nothing — closing the exact gap 172-04's shrink-only skip-list guard was resting on.
- Flag-audit mismatches now report a `filepath.Rel`-derived repo-relative path (e.g. `.claude/commands/ant/status.md`) instead of a bare basename, so a violation in `build.md` — which exists in both `.claude/commands/ant/` and `.opencode/commands/ant/` — identifies exactly one file.

## Task Commits

1. **Task 1: Make the substitution opener and its closing delimiter one decision, and pin all three forms** — `d4e85a07` (feat)
2. **Task 2: Make the flag audit fail when it is not looking at anything, and name files unambiguously** — `2578f3b1` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/command_call_audit_test.go` — Replaced `openedSubstitution(tok string) bool` with `substitutionOpener(tok string) (open, closeDelim string)`, adjacent to `normalizeShellToken` with a comment explaining the two-halves-of-one-decision relationship and naming CR-05. Updated both trim call sites (the backtick branch and `parseFencedInvocation`) to trim the returned `closeDelim`. Extended `TestCommandCallExtractorSeesRealInvocationsAndSkipsProse`'s fixture with three new lines and matching assertions; updated the exact-count assertion from 6 to 9.
- `cmd/cli_flag_audit_test.go` — Added `path/filepath` import. `TestCLIFlagAudit` now resolves `markdownDirs` via `repoRootForCommandSourceTest()` + `filepath.Join`; turned the two silent `continue`s into `t.Errorf`-and-continue with named directory/file and error; added a `scannedFiles` counter and a post-loop anti-vacuity `t.Fatalf`; switched `mismatch.file` from `entry.Name()` to a `filepath.Rel`-derived repo-relative path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in the plan's own literal wording] `t.Errorf` used for per-directory/per-file failures instead of the action text's literal `t.Fatalf`, with the anti-vacuity floor as a separate terminal `t.Fatalf`**

- **Found during:** Task 2, while implementing the two mandatory red proofs.
- **Issue:** The plan's action text says "if `os.ReadDir` fails on a declared directory, `t.Fatalf` naming the directory and the error." Taken literally, a per-directory `t.Fatalf` halts the Go test goroutine at the very first failing directory (`runtime.Goexit()` semantics) — it can never reach later code, including the anti-vacuity floor check. This makes the plan's own Red Proof 2 ("replace all three entries with non-existent directories and confirm the failure names the scanned-file count and the subcommand count — the anti-vacuity floor, distinct from red proof 1's per-directory fatal") structurally unproduceable: with pure per-directory `t.Fatalf`, mutating all three directories would just produce the identical per-directory-fatal shape as mutating one, on whichever directory is first in the slice, and the "distinct" floor message could never be shown.
- **Fix:** Used `t.Errorf` (which fails the test but does not halt the goroutine) for each unreadable directory/file, so the loop continues to attempt every declared directory. Added a separate, terminal `t.Fatalf` after the loop for the anti-vacuity floor (`scannedFiles < 120 || len(foundSubcommands) == 0`). This produces two genuinely distinct, verifiable failure shapes: Red Proof 1 (one directory missing, the other two still yield >= 120 files) shows only the per-directory `t.Errorf`, no floor failure; Red Proof 2 (all three missing) shows all three per-directory `t.Errorf` messages *plus* the distinct floor `t.Fatalf` naming `scannedFiles=0`/`foundSubcommands=0`. Both are "loud failures" per the plan's stated intent ("Turn both silent continues into loud failures") — the deviation is from the literal API name, not from the intent.
- **Verified fix:** Both red proofs run and produce exactly the required, distinguishable shapes (transcripts below).
- **Files modified:** `cmd/cli_flag_audit_test.go`
- **Commit:** `2578f3b1`

## Mandatory Red Proofs (from plan's `<output>` requirement)

### Task 1 — fixture cases added and count assertion updated to 9, BEFORE `substitutionOpener` existed

Command: `go test ./cmd -run TestCommandCallExtractorSeesRealInvocationsAndSkipsProse -count=1 -v`

```
=== RUN   TestCommandCallExtractorSeesRealInvocationsAndSkipsProse
    command_call_audit_test.go:861: extractor missed the backtick-substitution invocation `RESULT=`aether skill-list``
    command_call_audit_test.go:881: extractor missed the bare-subshell invocation `(aether colony-name)`
    command_call_audit_test.go:895: midden-recent-failures args = [--limit 5)], want ["--limit" "5"] (trailing `)` must not survive)
    command_call_audit_test.go:899: midden-recent-failures arg "5)" retains a stray closing paren
    command_call_audit_test.go:908: extracted 7 calls, want 9 (...)
--- FAIL: TestCommandCallExtractorSeesRealInvocationsAndSkipsProse (0.00s)
FAIL
```

Fails naming all three: the missing `skill-list` call, the missing `colony-name` call, and a `midden-recent-failures` argument list that is not exactly `--limit 5`, as required.

After implementing `substitutionOpener` and updating both trim call sites, re-run:

```
=== RUN   TestCommandCallExtractorSeesRealInvocationsAndSkipsProse
--- PASS: TestCommandCallExtractorSeesRealInvocationsAndSkipsProse (0.00s)
PASS
```

### Task 2, Red Proof 1 — one directory (command-playbooks) replaced with a non-existent path

Command: `go test ./cmd -run TestCLIFlagAudit -count=1 -v`

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:136: read declared corpus directory /.../.aether/docs/command-playbooks-DOES-NOT-EXIST-172-08-REDPROOF1: open .../command-playbooks-DOES-NOT-EXIST-172-08-REDPROOF1: no such file or directory — a declared input directory that cannot be read is a loud failure for this audit, not a silent skip
    cli_flag_audit_test.go:240: Audit coverage: 77 unique subcommands found in markdown
    cli_flag_audit_test.go:241: Registered subcommands in Go runtime: 373
    cli_flag_audit_test.go:242: scanned 126 files across 3 corpora
--- FAIL: TestCLIFlagAudit (0.01s)
FAIL
```

Fails naming that directory. Note: the pre-change code (silent `continue`) passed this exact mutation without any failure at all — the whole point of this task. The remaining two corpora (63 + 63 = 126 files) stay above the 120 anti-vacuity floor, so this failure is purely the per-directory message, distinct from Red Proof 2 below. Restored, re-ran:

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:240: Audit coverage: 126 unique subcommands found in markdown
    cli_flag_audit_test.go:241: Registered subcommands in Go runtime: 373
    cli_flag_audit_test.go:242: scanned 141 files across 3 corpora
--- PASS: TestCLIFlagAudit (0.01s)
PASS
```

### Task 2, Red Proof 2 — all three directories replaced with non-existent paths

Command: `go test ./cmd -run TestCLIFlagAudit -count=1 -v`

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:136: read declared corpus directory .../.claude/commands/ant-DOES-NOT-EXIST-172-08-REDPROOF2: ... no such file or directory — ...
    cli_flag_audit_test.go:136: read declared corpus directory .../.opencode/commands/ant-DOES-NOT-EXIST-172-08-REDPROOF2: ... no such file or directory — ...
    cli_flag_audit_test.go:136: read declared corpus directory .../.aether/docs/command-playbooks-DOES-NOT-EXIST-172-08-REDPROOF2: ... no such file or directory — ...
    cli_flag_audit_test.go:219: flag audit read too little to trust: scannedFiles=0 (want >= 120), foundSubcommands=0 — a guard reading nothing passes forever, silently, the moment its declared corpus directories move, are renamed, or come back empty
--- FAIL: TestCLIFlagAudit (0.00s)
FAIL
```

Confirms the failure names both the scanned-file count and the subcommand count — the anti-vacuity floor, visibly distinct from Red Proof 1's shape (which showed no floor message at all). Restored, re-ran:

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:240: Audit coverage: 126 unique subcommands found in markdown
    cli_flag_audit_test.go:241: Registered subcommands in Go runtime: 373
    cli_flag_audit_test.go:242: scanned 141 files across 3 corpora
--- PASS: TestCLIFlagAudit (0.01s)
PASS
```

### Task 2, Failure-path naming proof — an unregistered flag injected into `.claude/commands/ant/status.md`

Inserted `` `aether status --nonexistent-flag-172-08` `` on a new line (line 10) inside `.claude/commands/ant/status.md`.

Command: `go test ./cmd -run TestCLIFlagAudit -count=1 -v`

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:235: CLI flag audit found 1 unique mismatches:
          .claude/commands/ant/status.md:10: subcommand "status" missing flag --nonexistent-flag-172-08
    cli_flag_audit_test.go:240: Audit coverage: 127 unique subcommands found in markdown
    cli_flag_audit_test.go:241: Registered subcommands in Go runtime: 373
    cli_flag_audit_test.go:242: scanned 141 files across 3 corpora
--- FAIL: TestCLIFlagAudit (0.01s)
FAIL
```

Confirms the failure names the path as `.claude/commands/ant/status.md` — repo-relative, not the bare `status.md` two corpora share — the line number (10), and the flag (`--nonexistent-flag-172-08`). This is ROADMAP criterion 3's "naming the file, the line, and the offending flag" demonstrated rather than asserted. Restored `status.md` with a path-scoped edit (`git diff --stat .claude/commands/ant/status.md` showed no diff after restoration), re-ran:

```
=== RUN   TestCLIFlagAudit
    cli_flag_audit_test.go:240: Audit coverage: 126 unique subcommands found in markdown
    cli_flag_audit_test.go:241: Registered subcommands in Go runtime: 373
    cli_flag_audit_test.go:242: scanned 141 files across 3 corpora
--- PASS: TestCLIFlagAudit (0.01s)
PASS
```

## Audited Invocation Count / Scanned File Count

- `TestCommandCallsMatchCobraContracts` logs `audited 988 documented invocations across 6 corpora` — unchanged from 172-06, confirming the `substitutionOpener` change touched no live corpus behavior.
- `TestCLIFlagAudit` logs `scanned 141 files across 3 corpora` and `Audit coverage: 126 unique subcommands found in markdown` on a clean tree — matching the plan's measured corpus size (63 + 63 + 15 = 141) and comfortably above the 120-file anti-vacuity floor.

## Verification Performed

- `go vet ./cmd` — clean, both after Task 1 and after Task 2.
- `gofmt -l cmd/cli_flag_audit_test.go cmd/command_call_audit_test.go` — clean.
- `grep -c 'openedSubstitution' cmd/command_call_audit_test.go` → 0; `grep -c 'substitutionOpener' cmd/command_call_audit_test.go` → 5 (declaration + comment reference removed, both call sites, plus the doc-comment reference).
- `grep -c 'TrimSuffix(args\[len(args)-1\], ")")' cmd/command_call_audit_test.go` → 0 — no hardcoded closing delimiter survives at either call site.
- `grep -c '"\.\./\.claude' cmd/cli_flag_audit_test.go` → 0; `grep -c 'directory may not exist in test environment' cmd/cli_flag_audit_test.go` → 0.
- `go test ./cmd -run 'TestCommandCallExtractorSeesRealInvocationsAndSkipsProse|TestCommandCallsMatchCobraContracts|TestExtractorDoesNotDesyncOnGluedFenceMarker|TestAuditedCorpusHasNoGluedFenceMarkers|TestAuditDetectsPositionalDrift|TestAetherCorpusCatchesAnUnregisteredFlag|TestDocumentedCommandNamesResolve|TestDocumentedSubcommandsAreSeverityClassified|TestGateClassifiedCallsHaveGateWiring' -count=1 -v` — all pass.
- `go test ./cmd -run 'TestCLIFlagAudit|TestCLIFlagAuditSubcommandsRegistered|TestFlagAuditSkipListOnlyShrinks|TestAllowlistPolicyNamesEveryGuardedFile' -count=1 -v` — all pass, scanned-file count logged at 141 (>= 120).
- The named CI step's command (`Verify subcommand wiring and CLI flag contracts`, copied verbatim from `.github/workflows/ci.yml`) run locally — all 23 named tests pass.
- `go test ./... -count=1 -timeout 900s` — full release-gate command, all packages green (`cmd` 336.3s, all `pkg/*` packages passing).
- `git status --porcelain` after each red-proof restoration — clean (no residue from any mutation; `status.md` verified byte-identical to its pre-mutation state via `git diff --stat`).
- `git status --porcelain` before the first commit and after the last commit of this plan — clean at both points (working tree fully committed, no stray files).

## Known Stubs

None — this plan is entirely test infrastructure.

## Threat Flags

None — this plan's threat model (T-172-36 through T-172-40) is fully addressed by the work above. T-172-40 (information disclosure) was explicitly accepted with no mitigation in the plan's own threat register, since this plan edits two test files only.

## Issues Encountered

None beyond the documented `t.Errorf`-vs-`t.Fatalf` deviation above, which was required to make the plan's own two mandatory Task 2 red proofs producible as specified.

## User Setup Required

None.

## Next Phase Readiness

- Both anti-vacuity findings the verifier logged as ⚠️ (CR-05's extractor asymmetry, and the flag audit's silent-continue vacuity) are closed by commands that fail when the property does not hold, matching CLAUDE.md's Definition of Done.
- D-12's shrink-only skip-list guard now rests on a `TestCLIFlagAudit` that cannot pass while reading zero files — the guard it was built on top of is provably non-vacuous.
- ROADMAP success criterion 3's exact wording ("naming the file, the line, and the offending flag") is demonstrated by the failure-path naming proof, not merely asserted by code review.
- Per the plan's own `depends_on: ["172-04", "172-06"]` and its gap-closure framing, this closes the two remaining ⚠️ findings from `172-VERIFICATION.md`; no further gap-closure plans are indicated by this plan's own scope.

---
*Phase: 172-wiring-proof*
*Completed: 2026-08-11*

## Self-Check: PASSED

- FOUND: `.planning/phases/172-wiring-proof/172-08-SUMMARY.md`
- FOUND commit: `d4e85a07` (Task 1)
- FOUND commit: `2578f3b1` (Task 2)
