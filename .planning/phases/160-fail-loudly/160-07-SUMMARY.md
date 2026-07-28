# Plan 160-07 — Summary

**Status:** Complete (3/3 tasks)
**Requirements:** LOUD-04, LOUD-05 (+ D-01 classification)
**Completed:** 2026-07-27

> **Execution note:** run inline by the orchestrator after the account spend limit stopped
> worktree executors.

## The blind spot this closes

`cmd/cli_flag_audit_test.go` matches `--flag` tokens with a regex and **never consults
cobra's `Args` validator**. A call that passes a bare positional to a `cobra.NoArgs` command
is therefore structurally invisible to it. All seven of the phase's confirmed-broken calls
were exactly that shape. A regex that only knows about flags cannot see them.

## What was built

**`cmd/command_call_severity.go`** (D-01) — the gate/enrichment split in **non-test** code,
because D-01 requires it "encoded where the audit can test it" and a table in a markdown file
cannot fail a build. Safety gates (`check-antipattern`, `verify-claims`, `gate-check`) halt
when they cannot execute; everything else defaults to enrichment (warn loudly, continue
degraded). Each gate entry records *why* it halts, since that judgement is what a future
reader has to re-make.

**`cmd/command_call_audit_test.go`** — five tests:

| Test | Requirement | What it does |
|---|---|---|
| `TestCommandCallsMatchCobraContracts` | LOUD-03/05 | Resolves **804 documented invocations** across 5 corpora against the real cobra tree and validates each with cobra's own `Find` + `Args` |
| `TestAuditDetectsPositionalDrift` | LOUD-04 | Self-test: feeds a positional to a registered `NoArgs` command and asserts the audit flags it |
| `TestCommandCallExtractorSeesRealInvocationsAndSkipsProse` | — | Guards the extractor against rotting into vacuous success |
| `TestGateClassifiedCallsHaveGateWiring` | D-01 | Gate-classified names must be real commands with recorded rationale |

**What "by execution, not regex" means here, precisely.** The audit runs cobra's real
resolution and validation machinery, not text matching. It deliberately stops short of
invoking `RunE`: executing 804 documented invocations for real would mutate colony state,
delete data, and publish to the hub. **What it cannot catch is a runtime failure inside
`RunE` given a well-formed command line.** That limit is stated in the test file rather than
left implied.

## What the audit found — beyond the original seven

The audit surfaced **41 genuinely broken documented calls that were not among the seven this
phase set out to fix**, all confirmed by running the commands:

| Broken form | Count | Why it fails | Fixed to |
|---|---|---|---|
| `gate-results-write --passed false` | 36 | `--passed` is a **bool** flag; `false` becomes a stray positional | `--passed=false` |
| `skill-parse-frontmatter <path>` | 2 | takes `--file`, not a positional | `--file <path>` |
| `flag-check-blockers {phase_number}` | 2 | accepts no positional arguments at all | argument removed |
| `generate-commit-message "contextual" {phase_id} …` | 1 | flag-based, not positional | `--type/--scope/--subject/--body` |

**Two of these were in the LIVE wrapper corpus** (`.claude/commands/ant/skill-create.md` and
its OpenCode mirror) — those would fail for a real user running `/ant-skill-create` today.
The other 39 are in the dead playbook corpus: documentation correctness, no user-visible
change, but a future phase reconnecting that corpus would have inherited every one of them.

## Iteration honesty

The audit's first run reported **138 violations, and essentially all of them were bugs in
the audit, not in the docs.** The extractor was discarding `<phase>` and `"$ARGUMENTS"` as
if they were shell redirects, could not resolve two-level subcommands (`host build`), and
validated `aether host *` against cobra even though those commands set
`DisableFlagParsing: true` and forward raw to the TypeScript host.

Corrections applied, each narrowing false positives without weakening detection:
1. `rootCmd.Find` on the full token list, so multi-level subcommand paths resolve correctly.
2. Skip commands under `DisableFlagParsing` — cobra is not the contract owner there.
3. `$ARGUMENTS` treated as **variadic (0..n)**; `<x>`/`{x}` as **exactly one**. Conflating
   these reported a violation on every correct wrapper.
4. A quote- and placeholder-aware tokenizer (`<temp completion>` is one argument, not two).
5. Bare `aether focus` in prose is a *reference*, not an invocation; zero-argument calls are
   only audited inside fenced code blocks.

138 → 73 → 49 → 43 → 41 real, all fixed → **0**.

This is recorded because an audit that reports confident, wrong findings is worse than no
audit: it trains readers to ignore it.

## Deliberate-regression proof

`--passed={true/false}` reverted to `--passed {true/false}` in `continue-verify.md`:

```
--- FAIL: TestCommandCallsMatchCobraContracts
    1 documented CLI call(s) violate the command's real argument contract
```

Reverted after. Note: a first attempt at this proof used a `sed` form that silently failed to
match, and the test "passed" — which momentarily looked like the audit had gone blind. The
proof above was re-run with a verified mutation. **Confirm the mutation landed before
trusting a green regression check.**

## Verification

```
go build ./cmd/aether   → OK
go vet ./cmd            → clean
go test ./cmd -count=1  → ok (804 invocations audited, 0 violations)
```

## Files

- `cmd/command_call_severity.go` (created)
- `cmd/command_call_audit_test.go` (created — 5 tests)
- `.claude/commands/ant/skill-create.md`, `.opencode/commands/ant/skill-create.md` (fixed — live corpus)
- `.aether/docs/command-playbooks/{continue-gates,continue-verify,continue-advance,continue-full,continue-finalize,build-full,build-prep,build-wave}.md` (fixed — dead corpus)
