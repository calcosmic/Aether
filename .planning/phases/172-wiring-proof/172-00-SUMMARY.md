---
phase: 172-wiring-proof
plan: "00"
subsystem: command-call-audit
tags: [testing, static-analysis, cli, wiring-proof]
dependency-graph:
  requires: []
  provides:
    - normalizeShellToken
    - command-substitution-aware invocation extraction
    - redirection-aware operator detection
  affects:
    - cmd/command_call_audit_test.go
    - 172-02 (orphan ratchet — depends on this extractor being honest)
    - 172-03 (flag-audit corpus extension — depends on this extractor being honest)
tech-stack:
  added: []
  patterns:
    - "Shared shell-token normalisation helper applied at every binary-detection call site"
    - "Paren-depth scan (substitutionDepth) to distinguish a top-level pipe from a nested one inside an unrelated command substitution"
key-files:
  created: []
  modified:
    - cmd/command_call_audit_test.go
    - .aether/docs/command-playbooks/build-full.md
    - .aether/docs/command-playbooks/build-wave.md
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-gates.md
decisions:
  - "Trailing-`)` trim must happen before the command-name regex check, not after — a zero-argument substitution call's name would otherwise fail the regex and be silently dropped"
  - "Newly-visible arg-contract violations (midden-recent-failures, state-read) were fixed in the docs, not by widening the Go commands' Args validators — the flags they need already exist"
  - "Nested-pipeline rejection implemented via paren-depth counting across the token slice preceding the aether token, not via a literal string comparison — the naive one-token lookback cannot distinguish a top-level pipe from a nested one"
metrics:
  duration: "~90 minutes"
  completed: 2026-08-11
---

# Phase 172 Plan 00: Command-Substitution-Aware Invocation Extraction Summary

One shared `normalizeShellToken` helper taught the repo's only markdown/YAML invocation
extractor to see `result=$(aether ...)` calls — previously invisible to both binary-detection
call sites — while a paren-depth check keeps nested pipelines rejected and a widened
`isShellOperator` stops `2>/dev/null` from posing as a positional argument.

## What Was Built

**Task 1 — the extractor fix.** `cmd/command_call_audit_test.go` gained:

- `normalizeShellToken(tok string) string` — strips a leading `VAR=` assignment prefix and then
  any leading run of `$(`, a backtick, or a bare `(`. Both `parseFencedInvocation` and the
  backtick branch of `extractDocumentedCalls` call this ONE helper to locate the `aether`
  binary token, replacing the dead `prev != "$("` comparison (a token `tokenizeShellLike` can
  never produce on its own) that let the two branches drift apart in the first place.
- `openedSubstitution(tok string) bool` — reports whether a token itself opens a substitution,
  used to decide whether the call's last token needs its trailing `)` trimmed
  (`--enforce)` → `--enforce`).
- `substitutionDepth(toks []string) int` — counts `(` minus `)` across the tokens preceding a
  candidate `aether` token. A positive depth means an EARLIER token opened a substitution that
  is still open, so `aether` is nested inside someone else's pipeline
  (`echo $(foo | aether cmd)`) even though the token immediately before it (`|`) looks like an
  ordinary, accepted pipe. This is what makes the negative nested-pipeline test case actually
  fail extraction, rather than merely documenting an intent that the single-token-lookback
  guard could not enforce.
- `redirectionRe` (`^[0-9]*[<>]`) extending `isShellOperator`, gated behind `isPlaceholder` so a
  `<phase>`-shaped documentation placeholder (which also starts with `<`) is never misread as a
  redirection.

**Task 2 — pinning tests.** Six new cases in
`TestCommandCallExtractorSeesRealInvocationsAndSkipsProse`: a backticked `x=$(aether cmd)`
(backtick branch), a fenced `NAME=$(aether cmd --flag)` (fenced branch), the verbatim
`.aether/workers.md:292` shape (`result=$(aether spawn-can-spawn 5 --enforce)` →
`Command == "spawn-can-spawn"`, `Args == ["5", "--enforce"]`), a redirected substitution
(`$(aether skill-index 2>/dev/null)` — `isShellOperator("2>/dev/null")` true,
`validateCallAgainstCobra` clean), and the negative nested-pipeline case.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Trailing-`)` trim ran after the name-regex check, silently dropping
zero-argument substitution calls**
- **Found during:** Task 2, while designing the pinning test for a zero-arg case
  (`x=$(aether skill-detect)`).
- **Issue:** In both `extractDocumentedCalls` and `parseFencedInvocation`, the command-name
  shape check (`^[a-z][a-z0-9-]*$`) ran BEFORE the trailing-`)` trim. A call like
  `$(aether status)` produced `name = "status)"`, which fails the regex and is silently
  dropped — the opposite of Task 1's own stated goal.
- **Fix:** Reordered both functions to trim the trailing `)` before the regex check.
- **Files modified:** `cmd/command_call_audit_test.go`
- **Commit:** 037a4b1f (folded into the Task 2 commit, since it was discovered while writing
  Task 2's own pinning tests)
- **Effect:** raised the audited-invocation count from 933 to 935 by un-blinding two more
  real, previously-invisible invocations: `skill-detect` and `survey-verify`, both classified
  as enrichment-tier in `knownEnrichmentSubcommands`.

**2. [Rule 1 - Bug] Five documented invocations, newly visible after the extractor fix, violate
their command's real argument contract**
- **Found during:** Task 1, running `TestCommandCallsMatchCobraContracts` after the tokenizer
  fix landed.
- **Issue:** Widening the extractor to see command-substitution calls surfaced five real,
  previously-invisible bugs in `.aether/docs/command-playbooks/` — the same corpus and the same
  class of bug (`.aether/workers.md:292`) this whole phase exists to catch:
  - `build-full.md:982`, `build-wave.md:511`, `build-wave.md:803`, `continue-advance.md:151` —
    `midden-recent-failures N` passing `N` as a bare positional; the command only accepts
    `--limit N` (`cmd/midden_cmds.go:486`).
  - `continue-gates.md:875` — `state-read '.build_synthesis.watcher'` passing a jq-style path as
    a positional to a `cobra.NoArgs` command (`cmd/state_cmds.go:872`) that returns the ENTIRE
    state and has no field-path parameter at all (`state-read-field` only accepts a fixed set of
    top-level field names, not a nested path).
- **Fix:** Corrected the four `midden-recent-failures` calls to use `--limit N`. Rewrote the
  `state-read` call to pipe the full state through `jq -c '.build_synthesis.watcher // {}'`
  instead — the runtime effect is identical, and no Go code changed.
- **Files modified:** `.aether/docs/command-playbooks/build-full.md`,
  `.aether/docs/command-playbooks/build-wave.md` (×2),
  `.aether/docs/command-playbooks/continue-advance.md` (×2 — the second occurrence at line 553
  is not currently audited at all; see Deferred Issues),
  `.aether/docs/command-playbooks/continue-gates.md`
- **Commit:** 3a7495f9
- **Why in scope:** the plan's own acceptance criteria requires
  `TestCommandCallsMatchCobraContracts` to pass after Task 1, and no other plan in this phase
  (172-01 through 172-05) references or owns these files. Fixing the runtime `Args` validators
  instead was rejected as broader than necessary — the flags these commands need already exist.

**3. [Rule 2 - missing classification] 17 subcommands newly visible to
`TestDocumentedSubcommandsAreSeverityClassified` had no D-01 gate/enrichment classification**
- **Found during:** Task 1 and Task 2's bugfix, running the severity-classification test.
- **Issue:** `knownEnrichmentSubcommands` is a reviewed allowlist; every subcommand the audit
  sees must be classified once, deliberately. The wider extractor surfaced 15 (Task 1) + 2
  (Task 2's ordering fix) previously-invisible subcommands with no entry.
- **Fix:** Reviewed each and added it to `knownEnrichmentSubcommands`: `colony-name`,
  `gate-recovery-template`, `gate-results-read`, `generate-progress-bar`,
  `learning-check-promotion`, `learning-extract-fallback`, `learning-promote-auto`,
  `midden-collect`, `midden-cross-pr-analysis`, `pheromone-merge-back`,
  `queen-write-learnings`, `should-skip-gate`, `skill-index`, `suggest-analyze`,
  `worktree-list`, `skill-detect`, `survey-verify`. All are read/report/non-blocking-sync
  commands — none halt a run on failure by design (several already have `|| echo <fallback>`
  wrapping in their documented invocations), so none belong in `gateClassifiedCommands`.
- **Files modified:** `cmd/command_call_audit_test.go`
- **Commits:** 3a7495f9, 037a4b1f

## Audited-Invocation Count (before / after)

| | Count | How measured |
|---|---|---|
| Before | **886** | `go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1 -v`, run against `f49eb459` (this plan's parent commit) via `git stash push -- cmd/command_call_audit_test.go` before any Task 1 edit |
| After | **935** | Same command, run against the final state of this plan (both tasks committed) |

935 − 886 = **49** previously-invisible invocations are now audited for the first time.

## Newly-Visible Commands Inside the Three Permitted Caller Corpora

Planning measured this as **none** — every command-substitution invocation lives in
`.aether/docs/command-playbooks/*.md` or `.aether/workers.md`, and the three corpora that count
as caller evidence for 172-02's orphan ratchet
(`.claude/commands/ant`, `.opencode/commands/ant`, `.aether/commands`) contain zero
`$(aether …)` invocations.

Re-verified after implementation: confirmed unchanged. Grepping `\$(` across
`.claude/commands/ant/`, `.opencode/commands/ant/`, and `.aether/commands/` at planning time and
again after this plan's changes both return only two hits, both in `lay-eggs.md`
(`.claude/` and `.opencode/` mirrors) and both are arithmetic expansions
(`files=$((files + 1))`), not command substitutions. 172-02 can seed its orphan baseline against
this extractor with no correction needed for this concern.

## Deferred Issues (not fixed — out of scope for this task)

- **`.aether/docs/command-playbooks/continue-advance.md` — malformed fence closer near line
  503.** `--ttl "30d"` is followed directly by a closing triple-backtick fence marker on the
  SAME line (no newline), which the fence-detection logic
  (`strings.HasPrefix(strings.TrimSpace(line), "```")`) cannot see. This desyncs
  in-fence/out-of-fence parity for the rest of the file, so a sixth `midden-recent-failures 50`
  call at line 553 is invisible to the audit entirely (fixed anyway, for consistency with the
  other four `--limit`-flag corrections, but the audit still cannot confirm it). Pre-existing,
  unrelated to command-substitution recognition — logged to
  `.planning/phases/172-wiring-proof/deferred-items.md` rather than fixed here.

## Self-Check: PASSED

- FOUND: `cmd/command_call_audit_test.go`
- FOUND: `.aether/docs/command-playbooks/build-full.md`
- FOUND: `.aether/docs/command-playbooks/build-wave.md`
- FOUND: `.aether/docs/command-playbooks/continue-advance.md`
- FOUND: `.aether/docs/command-playbooks/continue-gates.md`
- FOUND: `.planning/phases/172-wiring-proof/172-00-SUMMARY.md`
- FOUND: `.planning/phases/172-wiring-proof/deferred-items.md`
- FOUND commit: `3a7495f9` (Task 1)
- FOUND commit: `037a4b1f` (Task 2)
