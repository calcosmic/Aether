---
phase: 172-wiring-proof
verified: 2026-08-11T22:10:00Z
status: gaps_found
score: 2/4 roadmap success criteria fully verified (criteria 2 and 3); criterion 1 has a newly confirmed, live counterexample; criterion 4's durability half remains defeatable by three independently reproduced mutations
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 2/4 (previous run: criteria 1 and 2 verified; criteria 3 and 4 failed/partial)
  gaps_closed:
    - "Success criterion 3 (fence-parsing blind spot hiding a live `midden-recent-failures 50` positional-argument violation at continue-full.md:1243) — closed by plan 172-06: extractDocumentedCalls now tolerates a marker glued to trailing content, the two hidden invocations are fixed, and TestAuditedCorpusHasNoGluedFenceMarkers sweeps the whole corpus. Independently re-verified below."
  gaps_remaining:
    - "Success criterion 4's durability half (the guard that is supposed to keep the CI gate durable over time) — plan 172-07 replaced the whole-file substring check with a step-scoped one, closing the specific decoy this phase's own prior verification had reproduced, but the replacement is itself a substring/first-match check with three new, independently reproduced bypasses (`; true` / `|| exit 0` / extra `-run` / `| cat` / commented-out run line; a commented-out step in a minimal workflow; first-`-run`-wins vs go test's last-`-run`-wins). The property criterion 4 asks for — that the gate cannot be silently narrowed without the guard going red — is still false."
  regressions:
    - "Success criterion 1 — previously marked VERIFIED. A code review completed after that verification, and independently reproduced here, demonstrates that the ratchet's caller-evidence map is keyed by bare command name with no parent path, and that two real, currently registered commands (top-level `aether colonize` and `aether closeout`) have zero direct callers anywhere in the audited corpus today, yet are not reported as orphans — confirmed absent from the live-generated 278-entry `cmd/testdata/orphan_allowlist.json`. The criterion's central claim (a subcommand with no caller makes the test fail, naming the command) is false for these two real commands right now, not hypothetically."
gaps:
  - truth: "Success criterion 1 — registering a new cobra subcommand with no caller outside its own definition file makes TestNoRegisteredSubcommandIsUnreferenced fail, naming the command"
    status: failed
    reason: >
      The ratchet's caller-evidence collection (`credit()` / `singleFileCallerNames`,
      `cmd/subcommand_reachability_ratchet_test.go:365-426`) records evidence keyed by
      bare leaf command name only — no parent command path. `computeOrphanNames`
      (same file, lines 713-739) looks evidence up the same way: `evidence[c.Name]`.
      The registered cobra tree has commands that share a leaf name across different
      parents. Two are live, present-tense, unreviewed false negatives today:
      `aether colonize` (top-level, registered at `cmd/codex_workflow_cmds.go:29`,
      not hidden, real functioning surveyor command) and `aether closeout` (top-level,
      registered at `cmd/ceremony_cmd.go:118`). Every documented invocation in the
      three audited corpora (`.claude/commands/ant`, `.opencode/commands/ant`,
      `.aether/commands`) calls only `aether host colonize` (a distinct command
      registered under the `host` parent, `cmd/host_cmd.go:32`) and
      `aether ceremony closeout` (registered under the `ceremony` parent,
      `cmd/ceremony_cmd.go`) — confirmed by grep across all three corpora: zero
      occurrences of a bare `aether colonize` or `aether closeout` invocation exist
      anywhere in them. Because `credit()` records only the bare token `colonize` /
      `closeout`, crediting `host colonize` also (wrongly) clears the unrelated
      top-level `colonize`, and crediting `ceremony closeout` also clears the
      unrelated top-level `closeout`. Independently reproduced by inspecting the
      live-generated `cmd/testdata/orphan_allowlist.json` (278 entries, produced by
      the ratchet's own `-update-orphan-allowlist` regeneration path): `colonize` and
      `closeout` are both absent from it. These are exactly the "works, and nothing
      calls it" commands the ratchet's own file header says it exists to catch, and
      it does not catch them. TestRatchetDetectsASyntheticOrphan (a synthetic,
      uniquely-named fixture) still passes, so the narrow fixture-level guarantee
      holds — but the criterion's actual wording is about the general property
      ("no caller ... makes the test fail, naming the command"), and that general
      property is demonstrably false for two real commands in the codebase today.
    artifacts:
      - path: "cmd/subcommand_reachability_ratchet_test.go"
        issue: "credit()/singleFileCallerNames record caller evidence keyed by bare command name with no parent path; computeOrphanNames looks evidence up the same way. Any two commands sharing a leaf name under different parents share evidence, so a call to one silently clears orphan status for the other."
    missing:
      - "Key caller evidence by resolved command path (cobra CommandPath(), e.g. 'aether host colonize' vs 'aether colonize'), not by bare leaf name — resolve each documented invocation through rootCmd.Find (already used by the sibling audit in command_call_audit_test.go:533) before crediting it."
      - "Update computeOrphanNames and enumerateRegisteredCommands to carry and compare the full command path, not just Name/Aliases."
      - "Regenerate cmd/testdata/orphan_allowlist.json after the fix and add real callers or explicit baseline entries for aether colonize and aether closeout (and audit the other name-collision pairs the review lists as currently benign: build, continue, plan, seal, swarm, watch, oracle, pheromones/registry/wisdom export-vs-import, get/set across colony-depth/parallel-mode/plan-granularity) before this can be called closed."
  - truth: "Success criterion 4 — the ratchet and the flag test run in the same CI command the release gate already runs, verified by deleting a caller and observing the gate go red, not by reading the workflow file"
    status: failed
    reason: >
      The specific proof this phase's own summaries rely on (delete skill-create's
      only caller, run the named CI step's exact command, observe it fail naming
      skill-create, restore, observe it pass) is genuine and re-confirmed here
      (TestDeletingACallerMakesTheRatchetNameIt passes on the current tree, naming
      skill-create / skill-create.yaml). Plan 172-07 also genuinely fixed the specific
      decoy this phase's own prior verification had reproduced (a `Test summary` step
      with `if: always()` piping the blanket command's text through
      `|| echo 'unknown'` used to satisfy a whole-file substring check even with the
      real `Run Go tests` step deleted; `blanketReleaseGateProblem` now scopes to the
      named step's block and that specific bypass is closed — reconfirmed here).
      However, the replacement check is itself built from substring/first-match logic
      over one line of YAML rather than a structural parse, and three new, independent
      bypasses of the *replacement* were reproduced directly against the current
      `blanketReleaseGateProblem` and `extractWiringGateRunArg` functions using
      synthetic workflow strings (not hypothetical — run against the actual current
      source in this repo):
      (1) `blanketReleaseGateProblem` returns nil (passes) for a `Run Go tests` step
      whose run line is `go test ./... -count=1 -timeout 900s; true`, or
      `... || exit 0`, or `... -run TestNothing` (the release gate then executes zero
      tests and exits 0), or `... | cat`, or a shell-commented-out command
      (`echo skip # go test ./... -count=1 -timeout 900s`) — none of these are among
      the three specific fallback strings the function checks for (`|| true`,
      `|| echo`, `|| :`), and it never rejects a `#`-commented run line or a bare
      trailing `; true`/extra `-run`.
      (2) `blanketReleaseGateProblem` returns nil against a minimal synthetic workflow
      whose only "Run Go tests" line is commented out (`# - name: Run Go tests`) —
      `stepBlock`'s marker match does not check that the matched text is not preceded
      only by a comment-prefix indent, so a commented-out step is read as a present,
      capable-of-failing step.
      (3) `extractWiringGateRunArg` uses `runFlagArgRe.FindStringSubmatch`, which
      returns the *first* `-run '<regex>'` on the line; `go test` itself honours the
      *last*. Given a run line with two `-run` flags
      (`go test ./cmd -run '<real filter>' -count=1 -timeout 900s -v -run 'TestNothingAtAll'`),
      the guard extracts and validates the first (real) filter and passes, while the
      step itself would actually run only `TestNothingAtAll` — a filter matching
      nothing, and the wiring step would report success while running zero guard
      tests.
      All three reproductions were run directly against the current
      `cmd/ci_wiring_gate_test.go` functions with synthetic input strings (the same
      technique the phase's own `TestBlanketGateCheckRejectsADecoyStep` uses), not
      against a mutated copy of the real ci.yml file, and were not persisted to disk.
      The property criterion 4 requires — that the gate cannot be silently narrowed
      without the durability guard going red — remains false, one round of
      gap-closure after the same underlying substring-matching root cause was first
      identified.
    artifacts:
      - path: "cmd/ci_wiring_gate_test.go"
        issue: "blanketReleaseGateProblem checks strings.Contains(runLine, blanketGateRunSubstring) and only three named exit-status fallback strings, rather than requiring the run line to equal (not merely contain) the exact command; and stepBlock's marker match does not verify the matched '- name:' text is not itself inside a comment. extractWiringGateRunArg's runFlagArgRe.FindStringSubmatch returns the first -run match on a line where go test honours the last, and never asserts there is exactly one -run flag or that the run line invokes 'go test ./cmd'."
    missing:
      - "Require the blanket gate step's run line to equal blanketGateRunSubstring exactly (after trimming), not merely contain it, and explicitly reject a line whose trimmed content starts with '#'."
      - "Anchor stepBlock's step-name match to require the character(s) before '- name:' on its line to be pure whitespace, so a commented-out step (# - name: ...) cannot satisfy it."
      - "Use FindAllStringSubmatch for the -run extraction in extractWiringGateRunArg, fail if more than one -run is present on the line, and assert the run line actually invokes 'go test ./cmd'."
      - "Extend TestBlanketGateCheckRejectsADecoyStep with table rows for each of the five new bypasses (; true, || exit 0, extra -run, | cat, commented-out run line, commented-out step) so the fix is pinned rather than asserted once by a verifier."
deferred: []
human_verification: []
---

# Phase 172: Wiring Proof Verification Report

**Phase Goal:** A capability added by this milestone cannot ship without a caller. The orphan
ratchet exists, runs in CI, and blocks — before any of the capabilities it constrains are
built, so it is shaped by the standard rather than by whatever shipped.

**Verified:** 2026-08-11T22:10:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 172-06/07/08, and after independently checking a code review (172-REVIEW.md, 4 critical findings) completed against the post-gap-closure state.

## Goal Achievement

### Observable Truths (ROADMAP.md § Phase 172 Success Criteria, verbatim)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Registering a new cobra subcommand with no caller makes `TestNoRegisteredSubcommandIsUnreferenced` fail, naming the command; allowlist seeded from real scanner output (278 orphans, 6 tagged `owner_phase: "178"`); shrink-only against a committed baseline | ✗ FAILED (regression) | Re-ran `TestNoRegisteredSubcommandIsUnreferenced` — still logs "enumerated 405 registered commands, found 278 orphans" and passes, and the count/shrink-only mechanics are unchanged and correct. But independently reproduced CR-04 from the code review: `computeOrphanNames`/`credit()` key caller evidence by bare command name with no parent path. `aether colonize` (top-level) and `aether closeout` (top-level) are real, non-hidden, currently registered commands with zero direct callers anywhere in the three audited corpora — confirmed by grep (only `aether host colonize` and `aether ceremony closeout` appear, never the bare top-level forms) — yet both are absent from the live-generated `cmd/testdata/orphan_allowlist.json` (278 entries), meaning the ratchet currently believes they have callers. This is the exact failure mode the criterion exists to prevent, live in the codebase today. |
| 2 | `aether spawn-can-spawn 5 --enforce` — the exact string `.aether/workers.md:292` documents — exits 0 | ✓ VERIFIED | Re-ran `go run ./cmd/aether spawn-can-spawn 5 --enforce` directly: `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0. |
| 3 | A test enumerates every `aether …` invocation in `.aether/*.md` and fails naming file/line/flag when an instruction names a flag the binary does not register | ✓ VERIFIED | Gap closed by plan 172-06. Re-ran `TestExtractorDoesNotDesyncOnGluedFenceMarker`, `TestAuditedCorpusHasNoGluedFenceMarkers` (scans 215 files, passes), `TestCommandCallsMatchCobraContracts` (audited 988 documented invocations, up from 954 — the fourteen previously-hidden glued-marker regions are now visible), `TestCLIFlagAudit` (scanned 141 files across 3 corpora, coverage 126 subcommands) — all pass. The specific live violation this gap was opened for (`continue-full.md:1243`'s bare positional to `midden-recent-failures`) is fixed to `--limit 50` and now visible to the audit. |
| 4 | The ratchet and the flag test run in the same CI command the release gate already runs — verified by deleting a caller and observing the gate go red, not by reading the workflow file | ✗ FAILED (gap not closed, defect class recurred) | The specific decoy this phase's prior verification reproduced (a non-blocking `Test summary` step's text satisfying a whole-file substring check) is genuinely closed by plan 172-07's `blanketReleaseGateProblem`/`stepBlock`, reconfirmed here. But three new, independently reproduced bypasses of the *replacement* check exist against the current source: (a) `; true`, `|| exit 0`, an appended `-run TestNothing`, `| cat`, or a `#`-commented run line all make `blanketReleaseGateProblem` return nil against a `Run Go tests` step carrying them; (b) a `# - name: Run Go tests` commented-out step in a minimal synthetic workflow also returns nil; (c) `extractWiringGateRunArg` extracts the *first* `-run` argument on a line where `go test` honours the *last*, so a run line with two `-run` flags validates the real filter while the step itself would run a different (or empty-matching) one. All three reproduced directly against the live functions with synthetic input, not asserted from the code review's prose. |

**Score:** 2/4 fully verified (criteria 2, 3); 2 failed (criterion 1 — regression discovered by review, independently confirmed; criterion 4 — gap-closure attempt did not close the underlying substring-matching defect class, independently confirmed with three new bypasses).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| WIRE-01 | 172-00, 172-02, 172-04, 172-05, 172-07, 172-08 | A test fails when a registered subcommand has no caller outside its own definition, seeded with today's known orphans in a shrink-only allowlist | ✗ BLOCKED | Same defect as truth 1: the ratchet does not fail for `aether colonize` / `aether closeout`, which have no caller today. REQUIREMENTS.md marks WIRE-01 `[x]` in its checklist prose but `Pending` in its traceability table (lines 32 and 139) — the traceability table is the more accurate of the two today. |
| WIRE-02 | 172-01, 172-05, 172-07 | The documented `.aether/workers.md` invocation matches a real flag; running it succeeds rather than erroring on `--enforce` | ✓ SATISFIED | Confirmed by direct execution above. |
| WIRE-03 | 172-00, 172-03, 172-04, 172-05, 172-06, 172-07, 172-08 | A test fails when a `.aether/*.md` instruction names a CLI flag the binary does not register | ✓ SATISFIED | Confirmed via re-run of the full named guard-test set (below) and the closed fence-parsing gap. |

No orphaned requirements: WIRE-01/02/03 are the only IDs REQUIREMENTS.md maps to Phase 172, and all three appear in at least one plan's `requirements` field.

### Full Named CI Guard-Test Set (re-run against current tree)

The exact `-run` filter from `.github/workflows/ci.yml`'s "Verify subcommand wiring and CLI flag contracts" step (24 test names) was re-run directly:

```
go test ./cmd -run '<24-name filter copied verbatim from ci.yml:100>' -count=1 -timeout 900s -v
```

All 24 pass. This confirms the phase's own summaries did not fabricate green test output — the guard suite is genuinely green. The finding in this report is that green does not equal correct: the code review's four critical issues are defects in what the guard tests *check*, not claims that the tests fail today.

### Independent Reproductions (this verification, not inherited from 172-REVIEW.md)

| Claim (from 172-REVIEW.md) | Reproduction method | Result |
|---|---|---|
| CR-01: blanket-gate substring check defeatable by `; true` / `\|\| exit 0` / extra `-run` / `\| cat` / `#`-commented run line | Called `blanketReleaseGateProblem` directly (package-internal test, synthetic workflow strings, no files mutated) with each of the five mutated run lines | All five returned `nil` (pass) — confirmed |
| CR-02: a commented-out `Run Go tests` step satisfies the check against a minimal workflow | Called `blanketReleaseGateProblem` with a workflow containing only a `# - name: Run Go tests` / `#   run: ...` block, no other steps | Returned `nil` (pass) — confirmed |
| CR-03: `extractWiringGateRunArg` reads the first `-run`, `go test` honours the last | Called `extractWiringGateRunArg` with a synthetic run line carrying two `-run` flags | Extracted the first (real) filter, `err=nil` — confirmed; the step itself would actually run the second filter |
| CR-04: caller evidence keyed by bare name credits `aether host colonize` as proof `aether colonize` (top-level) is used, and `aether ceremony closeout` as proof `aether closeout` (top-level) is used | Confirmed both are separately registered (`cmd/codex_workflow_cmds.go:29` vs `cmd/host_cmd.go:32`; `cmd/ceremony_cmd.go:118` vs its `ceremony` parent), confirmed by grep that only the parented forms appear anywhere in the three audited corpora, and confirmed both bare names are absent from the live-generated 278-entry `orphan_allowlist.json` | Confirmed on all three points |

All reproductions used synthetic in-memory strings passed directly to the unexported functions (the same technique `TestBlanketGateCheckRejectsADecoyStep` uses) or read-only inspection of committed files. No repository file was mutated during this verification; `git status --porcelain cmd/ .github/` was empty before and after.

### Anti-Patterns / Additional Findings (not blocking the 4 roadmap criteria, but load-bearing for the phase's stated purpose)

| File | Finding | Severity | Impact |
|---|---|---|---|
| `cmd/subcommand_reachability_ratchet_test.go:39,902-906` | `-update-orphan-allowlist` flag returns before the unallowed-orphan and D-08 owner-phase assertions run; the escape-hatch scanner does not look for `flag.Bool`-style overrides | ⚠️ Warning (WR-01 in review) | A command-line flag inside a guard file can suppress the very assertion the guard exists to make |
| `cmd/ci_wiring_gate_test.go` | Named wiring step is never checked for `if:`/`continue-on-error`/swallowed exit status (only the blanket step is) | ⚠️ Warning (WR-02) | Legibility loss only — the blanket step still provides coverage |
| `.github/workflows/ci.yml` | Nothing asserts the workflow's `on:` block or job-level `if:` — a job that never runs would still pass every step-scoped guard | ⚠️ Warning (WR-03) | A gate the workflow never reaches is not a gate |
| `.aether/docs/command-playbooks/continue-full.md:1244-1246` | `midden-recent-failures --limit 50`'s arity is now correct, but the surrounding `jq '.count'` / `.failures[]` parsing does not match the command's real `{"result":{"entries":...,"total":...}}` response shape — `midden_count` is always 0 | ⚠️ Warning (WR-08) | Outside the audit's stated scope (arity, not response-shape), but the described auto-REDIRECT block is dead code as written |

These are recorded here for traceability but do not change the pass/fail status of the four roadmap success criteria — none of them is the specific mechanism those criteria name.

### Gaps Summary

Two of the four roadmap success criteria are not met on the current tree, independently confirmed by direct reproduction against the live source (not inherited from SUMMARY.md or 172-REVIEW.md claims):

1. **Criterion 1 (orphan ratchet correctness) has a live, present-tense counterexample.** `aether colonize` and `aether closeout` (both real, registered, non-hidden top-level commands) have zero direct callers in the audited corpus today, and the ratchet does not flag either — because caller evidence is credited by bare command name, so a call to the differently-parented `aether host colonize` / `aether ceremony closeout` wrongly clears the top-level command's orphan status. This was VERIFIED in the prior verification pass; it is now FAILED, having been discovered by code review and independently reproduced here against the current code.

2. **Criterion 4 (CI-gate durability) is still not met**, one gap-closure round after the same underlying defect class (a substring match standing in for a structural check) was first identified. Plan 172-07 closed the exact bypass this phase's own prior verification reproduced, but the replacement check is built from the same category of weak match (`strings.Contains`, first-match-wins) and three new, independently reproduced bypasses exist against the code as it stands right now.

Both gaps are precisely the shape the phase's own goal statement describes wanting to prevent: "a capability added ... cannot ship without a caller," and a gate "shaped by the standard rather than by whatever shipped." The phase's own two most safety-critical guards are themselves currently shaped by what a substring-matching implementation happened to catch, not by the structural property the roadmap criteria describe.

---

_Verified: 2026-08-11T22:10:00Z_
_Verifier: Claude (gsd-verifier)_
