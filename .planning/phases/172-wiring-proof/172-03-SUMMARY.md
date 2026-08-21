---
phase: 172-wiring-proof
plan: "03"
subsystem: command-call-audit
tags: [testing, static-analysis, cli, wiring-proof, aether-workers-md]
dependency-graph:
  requires: [172-00, 172-01]
  provides:
    - "auditedFiles — a second, non-recursive corpus in cmd/command_call_audit_test.go covering the top-level .aether/*.md files"
    - "anti-vacuity assertion that .aether/workers.md actually contributed an extracted call"
    - "TestAetherCorpusCatchesAnUnregisteredFlag — permanent proof the corpus+extractor+validator chain catches an unregistered flag"
    - ".aether/workers.md with every documented invocation matching the binary's real contract"
  affects:
    - cmd/command_call_audit_test.go
    - .aether/workers.md
tech-stack:
  added: []
  patterns:
    - "Disjoint non-recursive file-list corpus alongside the existing recursive-directory auditedCorpora, since filepath.Walk cannot express 'this one directory, non-recursively'"
    - "Function-local cobra fixture command (rootCmd.AddCommand + defer rootCmd.RemoveCommand) mirroring a superseded pre-fix contract, fed the real extractor's own output rather than a hand-built struct"
key-files:
  created: []
  modified:
    - cmd/command_call_audit_test.go
    - .aether/workers.md
decisions:
  - "auditedFiles is a package-level []string of exact top-level .aether/*.md paths, walked by a small dedicated loop in collectDocumentedCalls rather than folded into auditedCorpora (which filepath.Walk always treats as a directory tree)"
  - ".aether/HANDOFF.md excluded — gitignored (.gitignore:87), so auditing it would make pass/fail depend on a session-local file's presence, which must never differ between CI and a local run"
  - "swarm-display-update's swarm identifier (--id) mapped to {your_name} (the parent worker), since no swarm_id variable exists anywhere in the general spawn protocol (that concept belongs to the separate /ant-swarm bug-investigation feature) — task_summary, tool-use counts, progress, and location positionals were dropped rather than inventing flags for them"
  - "The anti-vacuity manual check could not literally follow 'rename the file, run the test' as written: .aether/workers.md is go:embed'd by the root aetherassets package (embedded_assets.go:12), so removing it breaks the build before the test can run. Verified the assertion instead by emptying the file's content (keeping it present so the embed succeeds) — this proves the same claim (a present-but-unread/empty file is caught) via a technique the embed constraint actually permits."
  - "The seed transcript required reverting both cmd/spawn.go and cmd/spawn_enforce_test.go (172-01 created the latter and it references spawnCanSpawnDecision, which only exists post-fix) to get the pre-172-01 state to compile — not just cmd/spawn.go as the plan's literal git-stash instruction implies, because spawn.go was already committed by 172-01 (nothing to stash) and its own test file is coupled to the fix"
metrics:
  duration: "~65 minutes"
  completed: 2026-08-11
---

# Phase 172 Plan 03: WIRE-03 — Bring `.aether/workers.md` Into the Flag Audit Summary

The flag audit now reads the four top-level `.aether/*.md` files (not just `.aether/commands/` and the retired playbooks), proved it actually opened `.aether/workers.md` rather than merely listing it, fixed the two real `swarm-display-update` positional-argument violations that corpus addition surfaced, and replaced the "seeded to fail today" criterion with a permanent fixture test that reproduces the pre-172-01 `--enforce` bug on every CI run.

## What Was Built

**Task 1 — the corpus and the drift it surfaced.**

- `auditedFiles []string` in `cmd/command_call_audit_test.go`: a second, disjoint,
  **non-recursive** corpus listing the four git-tracked top-level `.aether/*.md` files
  (`CONTEXT.md`, `CROWNED-ANTHILL.md`, `QUEEN.md`, `workers.md`). `.aether/HANDOFF.md` is
  excluded — it is gitignored, so auditing it would make the test's result depend on whether a
  session-local file happens to exist locally versus in a fresh CI checkout.
- `collectDocumentedCalls` extended with a small loop over `auditedFiles` (mirroring the
  existing missing-corpus `os.Stat` tolerance), run in addition to the five directory-tree
  walks over `auditedCorpora`.
- An anti-vacuity assertion inside `TestCommandCallsMatchCobraContracts`: at least one
  extracted call must have a `File` path ending in `.aether/workers.md`, or the test
  `t.Fatal`s naming that file. The existing `len(calls) == 0` guard could not catch a corpus
  that silently contributed nothing while four other corpora kept the total non-zero — this
  closes exactly that gap.
- Fixed the two real violations the new corpus surfaced: `.aether/workers.md:329` and `:376`
  called `aether swarm-display-update` with nine positional arguments against a real contract
  (`cmd/swarm.go:261-264`) of `Args: cobra.NoArgs` and exactly three flags — `--agent`, `--id`,
  `--status`. Rewrote both to `aether swarm-display-update --agent "{child_name}" --id
  "{your_name}" --status "excavating"` (and `"completed"` for the second), dropping the task
  summary, per-tool-call counts, progress percentage, and location positionals — the command
  has no parameter for any of them, and inventing flags to keep them would recreate exactly
  the drift this phase exists to catch.
- `.aether/workers.md:292` (`result=$(aether spawn-can-spawn {your_depth} --enforce)`) is
  byte-identical before and after — confirmed by `git diff` — and appears among the validated
  invocations with zero violations, proving 172-00's extractor fix and 172-01's `--enforce`
  registration are both live in this test's path.
- `swarm-display-update`, newly visible to the audit for the first time, had no D-01
  gate/enrichment classification; added to `knownEnrichmentSubcommands` (a display-only,
  non-blocking status update, the same tier as `pheromone-display` and `swarm-timing-start`).

**Task 2 — the permanent proof and the transcripts.**

- `TestAetherCorpusCatchesAnUnregisteredFlag`: registers a function-local fixture command
  (`audit-selftest-preenforce-spawn-can-spawn`, `Args: cobra.NoArgs`, `--depth` registered,
  **no** `--enforce`) via `rootCmd.AddCommand` + `defer rootCmd.RemoveCommand` — mirroring the
  pre-172-01 `spawn-can-spawn` contract exactly. It then runs the real `extractDocumentedCalls`
  over the real `.aether/workers.md`, finds the live `spawn-can-spawn` call, swaps the command
  name to the fixture's via `strings.Replace`, re-parses that text through the real
  `parseFencedInvocation` (not a hand-built `documentedCall` — the point is proving the corpus,
  extractor and validator together, which a hand-built struct could not do), and asserts
  `validateCallAgainstCobra` returns a violation naming `--enforce`. `t.Fatal`s if the extractor
  finds no `spawn-can-spawn` call at all, so a silently-broken extractor can never make this
  test pass vacuously.
- Both transcripts recorded below, produced by temporarily reverting `cmd/spawn.go` (and the
  172-01-created `cmd/spawn_enforce_test.go`, which the pre-fix state cannot compile without
  reverting too) to their pre-172-01 commit, running the target test, then restoring both files
  byte-identical (confirmed via `git diff --stat` showing no change) before continuing.

## Task Commits

1. **Task 1: Add the top-level `.aether/*.md` corpus and fix the drift it surfaces** —
   `92803b37` (feat)
2. **Task 2: Make the `--enforce` seed permanent, and record the red-then-green proof** —
   `2ebba038` (test)

## Files Modified

- `cmd/command_call_audit_test.go` — `auditedFiles` corpus, anti-vacuity assertion,
  `swarm-display-update` classification, `TestAetherCorpusCatchesAnUnregisteredFlag`.
  `normalizeShellToken`, `isShellOperator`, and `tokenizeShellLike` (172-00's wave-1 work) are
  untouched — confirmed via `git diff --stat`.
- `.aether/workers.md` — lines 329 and 376 rewritten to the registered flag form; line 292
  byte-identical.

## Verification

```
$ go test ./cmd -run 'TestCommandCallsMatchCobraContracts|TestDocumentedCommandNamesResolve|TestCommandCallExtractorSeesRealInvocationsAndSkipsProse|TestDocumentedSubcommandsAreSeverityClassified|TestAetherCorpusCatchesAnUnregisteredFlag|TestAuditDetectsPositionalDrift' -count=1 -v
=== RUN   TestCommandCallsMatchCobraContracts
    command_call_audit_test.go:586: audited 954 documented invocations across 6 corpora
--- PASS: TestCommandCallsMatchCobraContracts (0.08s)
=== RUN   TestAuditDetectsPositionalDrift
--- PASS: TestAuditDetectsPositionalDrift (0.00s)
=== RUN   TestAetherCorpusCatchesAnUnregisteredFlag
--- PASS: TestAetherCorpusCatchesAnUnregisteredFlag (0.00s)
=== RUN   TestCommandCallExtractorSeesRealInvocationsAndSkipsProse
--- PASS: TestCommandCallExtractorSeesRealInvocationsAndSkipsProse (0.00s)
=== RUN   TestDocumentedCommandNamesResolve
--- PASS: TestDocumentedCommandNamesResolve (0.07s)
=== RUN   TestDocumentedSubcommandsAreSeverityClassified
--- PASS: TestDocumentedSubcommandsAreSeverityClassified (0.07s)
PASS
```

- `go vet ./cmd` — clean.
- `go run ./cmd/aether swarm-display-init --id Y` then `swarm-display-update --agent X --id Y
  --status excavating` — both succeed (`{"ok":true,...}`), confirming the rewritten form
  matches the real command's argument contract.
- `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced -count=1` — `ok ... [no tests to
  run]`. That test is 172-02's (concurrent wave-2 plan) and does not exist yet in this
  worktree; it passes vacuously here and will run for real once the waves merge. No file this
  plan touched creates a registered-but-uncalled command, so nothing in this plan's diff can
  trip it once it exists.
- `git diff --stat cmd/command_call_audit_test.go` shows no change to `normalizeShellToken`,
  `isShellOperator`, or `tokenizeShellLike`.
- `git diff .aether/workers.md` — line 292 unchanged; only the two `swarm-display-update` lines
  altered.
- `go test ./cmd/... -count=1 -timeout 900s` — **PASS**, `ok github.com/calcosmic/Aether/cmd
  281.627s`, run after both tasks were committed.

### Anti-vacuity manual check (Task 1 acceptance criterion)

The plan's literal instruction ("temporarily rename the file, run the test, confirm it fails
naming `.aether/workers.md`") does not work as written: `.aether/workers.md` is `go:embed`'d by
the root `aetherassets` package (`embedded_assets.go:12`), which `cmd/install_cmd.go` imports.
Renaming the file breaks compilation (`pattern .aether/workers.md: no matching files found`)
before the test can even run — a stronger failure mode than the one being tested for, but not
the one the criterion asked to observe. Verified the same underlying claim instead by keeping
the file present (satisfying the embed) but replacing its content with text containing no
`aether` invocations:

```
$ mv .aether/workers.md .aether/workers.md.bak && go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1 -v
# github.com/calcosmic/Aether/cmd
embedded_assets.go:12:482: pattern .aether/workers.md: no matching files found
FAIL	github.com/calcosmic/Aether/cmd [setup failed]

$ echo "# Worker spawn protocol (no invocations)" > .aether/workers.md && go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1 -v
=== RUN   TestCommandCallsMatchCobraContracts
    command_call_audit_test.go:564: .aether/workers.md contributed zero extracted calls — the corpus is listed but not being read, which is the exact vacuous-pass failure mode this test exists to catch
--- FAIL: TestCommandCallsMatchCobraContracts (0.08s)
FAIL
```

File restored from a backup copy immediately after; `git diff .aether/workers.md` afterward
showed the plan's intended two-line change only, confirming a clean restore.

### Seed transcript (Task 2 acceptance criterion)

`cmd/spawn.go` was already committed by 172-01 (`ad57e8b3`), so `git stash push cmd/spawn.go`
as the plan literally specifies stashes nothing (no uncommitted changes to stash). Reproduced
the pre-172-01 state instead by checking out `cmd/spawn.go` from `ad57e8b3^` and additionally
moving aside `cmd/spawn_enforce_test.go` (created new by 172-01, references
`spawnCanSpawnDecision`, which only exists in the post-fix `spawn.go` — without moving it aside
the package does not compile at all). Both files were restored byte-identical afterward.

**RED** — pre-172-01 `cmd/spawn.go`, `cmd/spawn_enforce_test.go` absent:
```
$ go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1 -v
=== RUN   TestCommandCallsMatchCobraContracts
    command_call_audit_test.go:580: 1 documented CLI call(s) violate the command's real argument contract:
          .aether/workers.md:292: `result=$(aether spawn-can-spawn {your_depth} --enforce)` — unknown flag --enforce
    command_call_audit_test.go:586: audited 954 documented invocations across 6 corpora
--- FAIL: TestCommandCallsMatchCobraContracts (0.08s)
FAIL
```

**GREEN** — both files restored:
```
$ go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1 -v
=== RUN   TestCommandCallsMatchCobraContracts
    command_call_audit_test.go:586: audited 954 documented invocations across 6 corpora
--- PASS: TestCommandCallsMatchCobraContracts (0.08s)
PASS
```

`git diff --stat cmd/spawn.go cmd/spawn_enforce_test.go` after restoring: empty — confirmed
byte-identical to the pre-transcript committed state.

## Violations Fixed and the Rule Applied to Each

Only violation category surfaced by the corpus addition: `swarm-display-update` positional
calls (2 occurrences, `.aether/workers.md:329` and `:376`). Rule applied per the plan's own
"which side" instruction: the binary already implements the capability the instruction
describes (updating an agent's display status) via three flags — `--agent`, `--id`,
`--status` — so the **instruction** was fixed, not the binary. The task summary, per-tool-call
counts, progress percentage, and swarm location the old positional call passed have no
corresponding real parameter anywhere in `swarmDisplayUpdateCmd`; those were dropped rather
than given invented flags, per the plan's explicit instruction.

`.aether/workers.md:292` (`spawn-can-spawn`) and line 11
(`ant_name=$(aether generate-ant-name "builder" | jq -r '.result')`) validated cleanly with no
changes needed, as predicted in the plan's `<corpus_scope>` section.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - missing classification] `swarm-display-update` had no D-01 severity
classification**
- **Found during:** Task 1, running `TestDocumentedSubcommandsAreSeverityClassified` after
  fixing the positional-argument violations.
- **Issue:** `swarm-display-update` was invisible to the audit before this plan (the corpus
  that mentions it — `.aether/workers.md` — was not yet audited), so it had never been reviewed
  for the gate-vs-enrichment split `TestDocumentedSubcommandsAreSeverityClassified` requires.
- **Fix:** Added `"swarm-display-update": true` to `knownEnrichmentSubcommands` — it is a
  non-blocking status-display update with no halt-worthy failure mode, the same tier as the
  other `swarm-*` display/timing commands already classified there.
- **Files modified:** `cmd/command_call_audit_test.go`
- **Commit:** `92803b37` (folded into the Task 1 commit, discovered while verifying Task 1's
  own acceptance criteria)

---

**Total deviations:** 1 auto-fixed (Rule 2 — missing classification, directly downstream of
this plan's own corpus addition).
**Impact on plan:** Necessary for `go test ./cmd/... -count=1 -timeout 900s` to pass, which is
both Task 1's and Task 2's own acceptance criterion. No scope creep — the classification only
covers the one command this plan's corpus change made visible for the first time.

## Self-Check: PASSED

- FOUND: `cmd/command_call_audit_test.go`
- FOUND: `.aether/workers.md`
- FOUND commit: `92803b37` (Task 1)
- FOUND commit: `2ebba038` (Task 2)
