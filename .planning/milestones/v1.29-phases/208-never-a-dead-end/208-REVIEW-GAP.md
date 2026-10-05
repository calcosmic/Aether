---
phase: 208-never-a-dead-end
reviewed: 2026-09-23T14:22:32Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - cmd/unattended_session.go
  - cmd/refusal.go
  - cmd/journey.go
  - cmd/journey_live_test.go
  - cmd/unattended_refusal_test.go
  - cmd/refusal_printed_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/testdata/refusals/host-relay-refusal.txt
  - .aether/commands/colonize.yaml
  - .claude/commands/ant/colonize.md
  - .claude/commands/ant-colonize.md
  - .opencode/commands/ant/colonize.md
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
resolved:
  warning: 2
  info: 1
  outstanding: 0
resolved_at: 2026-09-23T15:05:00Z
status: clean
---

# Phase 208 (gap-closure plans 208-09/208-10): Code Review Report

**Reviewed:** 2026-09-23T14:22:32Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** clean -- all 3 findings fixed and locked by named tests (see Fix Status)

## Summary

Scope is exactly `git diff 0079ff73..HEAD` over the twelve listed paths -- plan 208-09's "act when
alone, ask when you're there" fix (`sessionHasNoOneToAsk`, the shared guidance sentence on both
refusal lanes, the host-relayed printed-refusal extractor, the four hand-kept colonize wrapper
copies) and plan 208-10 (documentation only; no code changed, confirmed by `git diff` and by
208-10-PLAN.md's own task list).

The mechanism is sound and its own claims were checked against the running code, not just the
doc comments:

- **Attended output is genuinely byte-for-byte unchanged.** Both `Error()` and `renderRefusal`
  gate the new sentence on `r.ProtectsWork && next != "" && sessionHasNoOneToAsk()`; with the
  fact unset (`sessionHasNoOneToAsk()` false), both functions produce exactly the pre-existing
  format strings. Verified by reading the code, and by running
  `TestAttendedRefusalTextIsUnchanged`, `TestOnlyAWorkProtectingStopCarriesTheGuidance`, and
  `TestUnattendedRefusalNamesTheWayPastOnBothLanes` locally (all pass).
- **No command-injection risk in the new relayed-form extraction path.** The subprocess runner
  it feeds (`journeyRunPrintedNextCommands`, unchanged by this diff but the consumer of
  `journeyPrintedRefusals`'s new output) splits the runtime command with `strings.Fields` and
  execs via `exec.CommandContext(ctx, "aether", fields[1:]...)` -- never a shell -- and
  independently re-checks `fields[0] == "aether"` before running. `printedCommandToRuntimeCommand`
  only maps text that already starts with `aether ` or names a verb registered in
  `wrapperCommandNames`; everything else is refused with `ok=false`, and the relayed-form
  extractor (`journeyPrintedRefusalRelayedLineRe` + the `printedCommandToRuntimeCommand` gate in
  `journeyPrintedRefusals`) silently discards an unmappable tail rather than treating it as a
  match, matching the "an unmappable tail must yield nothing" requirement.
- **The trust boundary (user-role `tool_result` blocks only) is intact for the new form.** The
  relayed-line regex is applied inside the exact same `entry.Message.Role != "user"` /
  `b.Type != "tool_result"` filter the pre-existing block-form regex already used -- confirmed by
  reading the loop body, and the pre-existing WR-05 guard against trusting the assistant's own
  text (`TestPrintedRefusalExtractorIgnoresAssistantOwnText`) is untouched by this diff and still
  passes.
- **`cmd/testdata/refusals/host-relay-refusal.txt` is genuinely captured, not typed.** Its
  provenance is recorded in 208-09-SUMMARY.md (line 65 of a real, still-on-disk journey
  transcript, copied byte-for-byte) and the fixture's own content is consistent with the
  documented `sanitizeBridgeMessage`/`formatGoCommand` relay shape in
  `.aether/ts-host/src/go-bridge.ts`. The test that consumes it treats it as an opaque blob read
  from disk and asserts only on the extractor's derived output -- it is not a fixture shaped by
  hand to fit the assertion.
- **All four hand-kept colonize command copies carry the identical rule text**, verified both by
  `TestColonizeWrapperCarriesTheActWhenAloneRule` (phrase-based, passing) and independently by
  computing SHA-256 over `.claude/commands/ant/colonize.md`, `.opencode/commands/ant/colonize.md`,
  and `.claude/commands/ant-colonize.md` directly (all three identical,
  `7c8b3918aeace20d166691154e7d1991cceabe054aecad0b2971d59db0f05ce9`, matching the updated
  `specialistCommandSurfaceHashes` entries in `cmd/lifecycle_wrapper_contract_test.go`).

Two things fall short of what they claim, both in the "one reader" and "trust boundary" areas
this review was specifically asked to stress-test:

1. `TestTheIsAnyoneHereFactHasOneReader` is documented, and referenced from CLAUDE.md, as
   enforcing that `AETHER_UNATTENDED` is read in exactly one place, "structurally." It is a plain
   substring grep over one directory's file text, not a call-shape-aware (AST) check, and a
   second reader written with a trivially different call shape passes it silently. Reproduced
   below.
2. The new relayed-form marker (`— next: `) is a generic three-word substring, not a
   distinctively-shaped block like the backtick-delimited `Next: \`...\`` line. Genuine,
   unrelated tool output that happens to contain that substring followed by something that maps
   through `printedCommandToRuntimeCommand` would be treated as a real printed refusal and
   actually executed by the journey harness.

Both are described in more detail below. Neither is a live defect today (no second reader
exists; no observed false-positive has occurred in the one real journey walk's transcripts), but
both are exactly the class of "guard that doesn't structurally guarantee what it claims" pattern
this project's own CLAUDE.md repeatedly calls out as having caused real incidents.

## Critical Issues

None found in this diff.

## Warnings

### WR-01: The "one reader" guard is a literal-string grep, not a structural check, and is trivially bypassed

**File:** `cmd/unattended_refusal_test.go:150-183` (guard), `cmd/unattended_session.go:39-47` (the
property it is meant to protect)

**Issue:** `TestTheIsAnyoneHereFactHasOneReader` builds `needle := os.Getenv("AETHER_UNATTENDED")`
as a literal string and does `strings.Contains(fileText, needle)` over every non-test `.go` file
directly inside `cmd/` (via `os.ReadDir(".")`, non-recursive). This has two independent gaps:

- **Call-shape bypass.** A second reader written as `os.Getenv(unattendedEnvVar)` (the variable
  instead of the literal), or `os.LookupEnv("AETHER_UNATTENDED")`, or anything routed through a
  helper/alias, does not contain the exact needle substring and is invisible to the guard. I
  reproduced this directly: adding a scratch file `cmd/zz_review_probe_second_reader.go`
  containing both `os.Getenv(unattendedEnvVar)` and `os.LookupEnv("AETHER_UNATTENDED")` as real,
  functioning second reads of the fact, `go test ./cmd -run TestTheIsAnyoneHereFactHasOneReader`
  passes cleanly. (The file was removed immediately after the probe; nothing was left behind.)
- **Scope bypass.** `os.ReadDir(".")` only lists files directly in the `cmd` package's own
  directory. `cmd/aether/main.go` (a separate `package main`, the actual binary entrypoint) and
  every file under `pkg/` are never scanned at all, by directory scope alone, regardless of call
  shape.

This matters because `AETHER_UNATTENDED` is the one fact that flips a work-protecting refusal
from "wait for the owner" to "run the recovery command yourself" -- exactly the class of signal
D-01's own reasoning treats as safety-relevant ("a wrong 'nobody is here' reading could replace
an owner's real work... without ever checking first"). A second, differently-scoped or
differently-thresholded reader added later (e.g. one that treats any non-empty value as "true"
instead of requiring the exact string `"1"`) would silently reintroduce exactly the drift
CLAUDE.md's own retrospective calls out ("a second reader is how two surfaces in this repository
have drifted apart before") -- and this guard would not catch it.

**Fix:** Either tighten the guard to be call-shape-agnostic (e.g. grep for the bare string
`"AETHER_UNATTENDED"` combined with any `os.Getenv`/`os.LookupEnv`/`os.Environ` call anywhere in
its vicinity, or better, an AST-based check that walks `*ast.CallExpr` nodes for `os.Getenv` /
`os.LookupEnv` with that literal argument -- the codebase already has precedent for AST-based
structural guards elsewhere), or scope it explicitly across the whole module (`filepath.Walk`
from the module root, skipping `vendor`/`node_modules`/`.git`) rather than one non-recursive
directory, and soften the doc comment's claim from "structurally" to what the check actually is.
At minimum, widen the scan to include `cmd/aether/` and `pkg/`.

### WR-02: The relayed-form marker is a generic substring that can turn unrelated tool output into an executed command

**File:** `cmd/journey.go:290-303` (`journeyPrintedRefusalRelayedLineRe`), `cmd/journey.go:392-403`
(the match/validate loop in `journeyPrintedRefusals`)

**Issue:** The block-form marker (`^Next: \`...\``) is distinctive: it requires a dedicated line
starting with a literal label and a backtick-delimited command, a shape vanishingly unlikely to
occur by coincidence in unrelated captured output. The new relayed-form marker is just
`— next: ` (an em dash, the word "next:", a space) matched anywhere inline via
`(?m)— next: (.+)$`. The only additional gate is that the text after the marker must map through
`printedCommandToRuntimeCommand` -- i.e. start with `aether ` or `/ant-<a real wrapper verb>`.
That gate stops arbitrary prose from being treated as a command, but it does not stop a
*coincidental* occurrence of the marker inside genuine, unrelated tool output (build logs,
linter output, npm/pip warnings, a commit message echoed by `git log`) that happens to be
followed by text shaped like a real `aether`/`/ant-` invocation from being treated as a real,
Aether-emitted refusal and then actually executed by `journeyRunPrintedNextCommands` as a
subprocess inside the practice project.

This is bounded (no shell is ever invoked; only `aether`/`ant-<verb>` commands can be reached at
all; it runs inside a disposable, harness-built practice project, not a production or
user-controlled environment) so this is not an arbitrary-code-execution risk. But it is exactly
the kind of accidental trigger the WR-05 fix (restricting extraction to genuine `tool_result`
blocks) was written to prevent one layer up: a false match here would silently run an
unintended `aether` command and could misclassify an otherwise-passing journey step as failed
(or mask a real failure) for a reason that has nothing to do with an actual refusal.

**Fix:** Require the block-form's own stricter shape as a precondition for accepting a relayed
match too -- e.g. only accept a relayed match when the same `tool_result` text also contains the
fixed `"Fatal: Go command failed:"` (or a similarly Aether-specific) prefix somewhere before the
marker on the same line, the way `formatGoCommand`'s actual host-relay output always does per
`.aether/ts-host/src/go-bridge.ts`. That keeps the "no delimiter" tradeoff documented in
`journeyPrintedRefusals`'s comment while anchoring the match to something only Aether's own
error path would plausibly produce.

## Info

### IN-01: Relayed-command punctuation stripping only handles a trailing period

**File:** `cmd/journey.go:392-395`

**Issue:** After extracting a relayed-form candidate, the code does
`strings.TrimSpace(strings.TrimSuffix(candidate, "."))` -- it strips exactly one trailing `.` if
present, but nothing else. `cmd/refusal_printed_test.go`'s
`TestPrintedRefusalExtractorFindsAHostRelayedRefusal` explicitly asserts the extracted command
never ends in `.` or `"`, but no code path actually strips a trailing `"` (or `'`, `)`, `,`,
etc.) -- the assertion currently passes only because the one real fixture happens not to exercise
that case. If a future relayed message is embedded inside quoted prose (plausible, given the
second real journey walk in 208-JOURNEY-RUN.md shows the runtime's own text quoted inline inside
the assistant's own reasoning), a trailing quote character would ride along into the command
Aether was told to map, and `printedCommandToRuntimeCommand` would (correctly) refuse it rather
than silently mis-running it -- so this degrades to "silently finds nothing" rather than a wrong
execution, but it's worth tightening for the same reason WR-02 above is: the fewer coincidences
required to produce a wrong answer, the better.

**Fix:** Either broaden the trim to a small fixed set of trailing punctuation
(`strings.TrimRight(candidate, ".\"'),;")`) or, better, address this the same way as WR-02 by
anchoring relayed matches to a following/preceding Aether-specific fragment rather than trying to
enumerate punctuation edge cases.

---

_Reviewed: 2026-09-23T14:22:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

## Fix Status

All three findings were fixed in the same session the review ran, each proved by a test that
fails without the fix (CLAUDE.md, "a test must be able to fail").

| ID | Fix | Proof it can fail |
|---|---|---|
| WR-01 | `TestTheIsAnyoneHereFactHasOneReader` is now a real structural check: it parses every non-test `.go` file in the whole module (`filepath.Walk` from the repo root, skipping `.git`/`vendor`/`node_modules`/`testdata`/`.planning`/`dist`) and refuses two things outside `cmd/unattended_session.go` -- the name as a string literal, and any use of the `unattendedEnvVar` identifier whatever it is passed to. It also fails if the walk finds implausibly few files or never visits the sole reader, so it cannot pass vacuously. The doc comment now states honestly the one case it does not catch (a name assembled at run time from pieces). | Both bypasses the review reproduced were re-run against the new check and are caught by name: `os.Getenv(unattendedEnvVar)` in a scratch `cmd/` file fails with "uses the unattendedEnvVar identifier"; `os.LookupEnv("AETHER_UNATTENDED")` in a scratch `pkg/zzprobe` package -- a directory the old check never scanned at all -- fails with "names AETHER_UNATTENDED as a string literal". Both scratch files removed. |
| WR-02 | `journeyPrintedRefusalRelayedLineRe` now requires the TypeScript host's own fixed `Go command failed:` prefix earlier on the same line (new `journeyRelayedRefusalPrefix` constant). This loses no genuine refusal: that relay is the only route by which `refusal.Error()`'s one-line form ever reaches a transcript, because a refusal reaching a terminal directly is matched by `errors.As` in `ExitWithError` (`cmd/root.go`) and rendered as the full drawn block, which the block-form pattern already finds. | Two cases added to `TestPrintedRefusalExtractorIgnoresRelayedProse`: a runnable `aether` tail with no relay prefix, and a relay prefix on a different line from the marker. Reverting the anchor makes the first fail by name, showing the unrelated line being returned as a refusal the harness would have executed. |
| IN-01 | The trailing-punctuation trim is now `strings.TrimRight` over a named set (`journeyRelayedCommandTrailingPunctuation`), not a single `TrimSuffix(".")`. | New `TestRelayedCommandLosesTrailingPunctuation` exercises seven trailing shapes with the command derived from a real registry row. Restoring the old single-period trim fails it by name on the quote cases -- which is exactly the vacuous-assertion gap IN-01 identified. |

Verified after the fixes: `go build ./...`, `go vet ./cmd/` and `go build -tags=journey ./cmd/aether`
are clean, and the whole affected test family passes, including the pre-existing refusal, voice,
wrapper-parity and journey-gate guarantees.
