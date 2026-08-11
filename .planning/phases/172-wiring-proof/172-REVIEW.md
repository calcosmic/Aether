---
phase: 172-wiring-proof
reviewed: 2026-08-11T00:00:00Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - .aether/docs/command-playbooks/build-complete.md
  - .aether/docs/command-playbooks/build-full.md
  - .aether/docs/command-playbooks/build-verify.md
  - .aether/docs/command-playbooks/build-wave.md
  - .aether/docs/command-playbooks/continue-advance.md
  - .aether/docs/command-playbooks/continue-finalize.md
  - .aether/docs/command-playbooks/continue-full.md
  - .aether/docs/command-playbooks/continue-gates.md
  - .aether/docs/orphan-allowlist-policy.md
  - .aether/workers.md
  - .github/workflows/ci.yml
  - cmd/ci_wiring_gate_test.go
  - cmd/cli_flag_audit_test.go
  - cmd/command_call_audit_test.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/testdata/flag_audit_skiplist_baseline.json
  - cmd/testdata/orphan_allowlist_baseline.json
  - cmd/testdata/orphan_allowlist.json
findings:
  critical: 4
  warning: 14
  info: 5
  total: 23
status: issues_found
---

# Phase 172: Code Review Report

**Reviewed:** 2026-08-11
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

This phase's claim is that Aether's wiring guards are actually wired into CI and
cannot pass vacuously. The guard suite is green (`go test ./cmd -run '<the CI
filter>'` passes, 24 tests). It is not, however, undefeatable, and it does not
establish the property it advertises.

Four defects are demonstrated, not inferred. Three of them let the CI gate be
switched off while every guard still reports green; I reproduced all three
against the live `.github/workflows/ci.yml` by feeding mutated copies of the
real file to the phase's own checking function. The fourth is a false negative
in the orphan ratchet — its headline claim ("nothing calls it") is already wrong
today for at least two registered commands, which I identified by re-running the
phase's own evidence collector with the ambiguity removed.

The common root cause across the CI findings is that `blanketReleaseGateProblem`
and `extractWiringGateRunArg` assert on **substrings of a line of YAML** rather
than on the structure of the command. The phase's own file comment says the
step must be checked "structurally … not merely have the command's text appear
somewhere in the file". It replaced a whole-file substring check with a
step-scoped substring check; the substring class of bug survived the fix.

Plain-English version, for the owner: the safety check that is supposed to prove
"the tests really run in CI" only reads the text of the command in the settings
file. Anyone can leave the text exactly as it is and add three characters after
it that throw away the result — the check still says everything is fine. And the
"is anything actually using this command?" check matches commands by name only,
ignoring which menu they sit under, so `aether host colonize` is counted as proof
that the unrelated top-level `aether colonize` is in use. It isn't.

## Critical Issues

### CR-01: The blanket CI gate check is a substring test — `; true` or an extra `-run` neuters the gate and the guard still passes

**File:** `cmd/ci_wiring_gate_test.go:214-251` (`blanketReleaseGateProblem`)

**Issue:** The function checks only that the step's single `run:` line
*contains* `go test ./... -count=1 -timeout 900s`, has no `-race`, no `if:`, no
`continue-on-error`, and none of exactly three fallbacks (`|| true`, `|| echo`,
`|| :`). Anything appended after the required substring is invisible to it.

Reproduced by passing mutated copies of the real `.github/workflows/ci.yml` to
`blanketReleaseGateProblem`:

| Mutation of the live gate step's run line | `blanketReleaseGateProblem` returns |
|---|---|
| `… -timeout 900s; true` | `nil` (passes) |
| `… -timeout 900s \|\| exit 0` | `nil` (passes) |
| `… -timeout 900s -run TestNothing` | `nil` (passes) |
| `… -timeout 900s \| cat` | `nil` (passes) |
| `run: echo skip # go test ./... -count=1 -timeout 900s` | `nil` (passes) |

The `-run TestNothing` case is the worst: the blanket release gate then executes
zero tests and exits 0, while the guard that exists to prove the gate is intact
reports it intact. The shell-comment case passes because the required text is
matched inside a `#` comment while the actual command is something else.

**Fix:** Stop matching a substring; parse and constrain the whole command.
Minimum viable hardening:

```go
// Require the run line to BE the command, not merely contain it.
cmd := strings.TrimSpace(strings.TrimPrefix(trimmedRunLine, "run:"))
if cmd != blanketGateRunSubstring {
    return fmt.Errorf("CI step %q's run line must be exactly %q (no appended "+
        "arguments, redirections, or fallbacks); found: %s",
        blanketGateStepName, blanketGateRunSubstring, cmd)
}
// Reject shell comments outright — the command must not be commented out.
if strings.HasPrefix(cmd, "#") {
    return fmt.Errorf("CI step %q's run line is commented out: %s", blanketGateStepName, cmd)
}
```

Then extend `TestBlanketGateCheckRejectsADecoyStep` with table rows for
`; true`, `|| exit 0`, `| cat`, a trailing `-run`, and a `#`-commented command.
Those five rows are the cases the existing six-row table does not cover.

---

### CR-02: A commented-out blanket gate step satisfies the check

**File:** `cmd/ci_wiring_gate_test.go:156-190` (`stepBlock`)

**Issue:** `stepBlock` locates the step by searching for the raw text
`- name: Run Go tests` anywhere in the file and requires only that it be
followed by end-of-line. It never checks that the marker begins the line (after
whitespace). A YAML comment satisfies it:

```yaml
      # - name: Run Go tests
      #   run: go test ./... -count=1 -timeout 900s
```

Verified against a minimal synthetic workflow: `blanketReleaseGateProblem`
returns `nil` — the release gate has been deleted and the guard passes.

Against the *current* `ci.yml` this mutation happens to fail, but only by
accident and with a wrong diagnosis: because no later `- name:` exists at the
matched (comment) indent, the block runs to end-of-file and picks up the
unrelated `Test summary` step's `if: always()`. The reported error is
`CI step "Run Go tests" has a conditional if: always()`, which is false — that
step does not exist at all. Deleting or reordering `Test summary` removes even
this accidental protection.

**Fix:** Anchor the marker to the start of the line's content, and reject a
match whose line prefix is not pure whitespace:

```go
lineStart := strings.LastIndexByte(workflow[:absIdx], '\n') + 1
indent := workflow[lineStart:absIdx]
if strings.TrimSpace(indent) != "" {
    // e.g. a "# " prefix — a commented-out step is not a step.
    searchFrom = afterEnd
    continue
}
```

Also bound the block: if no next `- name:` at the same indent is found, stop at
the first line whose indentation is less than the step's, rather than running to
EOF — so the `if:`/`continue-on-error` scan can never inspect an unrelated
later step.

---

### CR-03: The named wiring step's `-run` filter is read from the first `-run`, but `go test` honours the last

**File:** `cmd/ci_wiring_gate_test.go:67, 357-373` (`runFlagArgRe`, `extractWiringGateRunArg`)

**Issue:** `runFlagArgRe.FindStringSubmatch(runLine)` returns the **first**
`-run '<regex>'` on the line. `go test` uses the **last** `-run` it is given.
Appending a second flag to the live wiring step:

```yaml
run: go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced|…' -count=1 -timeout 900s -v -run 'TestNothingAtAll'
```

`extractWiringGateRunArg` returns the full 947-character original filter and
`TestWiringGateStepRunsEveryWiringTest` passes, while the step itself runs zero
guard tests. Verified against the live `ci.yml`.

The same class applies to the whole line: nothing asserts the run line even
begins with `go test ./cmd`, so `run: echo "-run '…'"` also passes.

**Fix:** Use `FindAllStringSubmatch` and fail if more than one `-run` is
present; assert the command shape as well:

```go
ms := runFlagArgRe.FindAllStringSubmatch(runLine, -1)
if len(ms) == 0 {
    return "", fmt.Errorf("CI step %q's run line has no `-run '<regex>'`: %s", wiringGateStepName, strings.TrimSpace(runLine))
}
if len(ms) > 1 {
    return "", fmt.Errorf("CI step %q's run line carries %d `-run` flags; go test honours only the last, so this guard would validate a filter that never executes: %s",
        wiringGateStepName, len(ms), strings.TrimSpace(runLine))
}
if !strings.Contains(runLine, "go test ./cmd ") {
    return "", fmt.Errorf("CI step %q's run line does not invoke `go test ./cmd`: %s", wiringGateStepName, strings.TrimSpace(runLine))
}
return ms[0][1], nil
```

---

### CR-04: Caller evidence is keyed by bare command name with no parent path — `aether host colonize` credits the unrelated top-level `colonize`

**File:** `cmd/subcommand_reachability_ratchet_test.go:365-426` (`singleFileCallerNames`), `713-739` (`computeOrphanNames`)

**Issue:** `credit()` records `names[command] = true` plus every following
bareword that looks like a command name, and `computeOrphanNames` asks only
`evidence[c.Name]`. Neither carries the parent path. The registered cobra tree
has **20 leaf names that exist at more than one path**, so a documented call to
one silently clears the other.

Measured on the live tree and the live caller corpora:

| Name | Registered at | Distinct commands? | A direct `aether <name>` call exists? |
|---|---|---|---|
| `colonize` | `aether colonize`, `aether host colonize` | yes | **no** |
| `closeout` | `aether closeout`, `aether ceremony closeout` | yes | **no** |

`.claude/commands/ant/colonize.md:16` documents `aether host colonize
$ARGUMENTS`; that credits the token `colonize`, which clears the top-level
`aether colonize` command. Nothing in any permitted corpus invokes
`aether colonize` or `aether closeout` directly, neither is in
`cmd/testdata/orphan_allowlist.json`, and the ratchet reports both as wired.
These are exactly the "works, and nothing calls it" commands the file's own
header (lines 3-12) says it exists to catch.

Other collisions currently benign but latent: `build`, `continue`, `plan`,
`seal`, `swarm`, `watch`, `oracle` (top-level vs `host …`), `pheromones` /
`registry` / `wisdom` (`export …` vs `import …`), `get` / `set`
(`colony-depth` vs `parallel-mode` vs `plan-granularity`). Any of these becoming
orphaned at one path will be invisible.

`computeOrphanNames`'s own comment acknowledges the name-collision problem for
*reporting* ("they collapse into one allowlist entry") but the same collision in
*crediting* is not handled.

**Fix:** Key evidence by the resolved command path, not the leaf name. Resolve
each documented invocation through `rootCmd.Find` (which the sibling audit in
`command_call_audit_test.go:533` already does) and record the `CommandPath()` of
the target; then compare against `CommandPath()` when computing orphans:

```go
// credit: resolve the token run through cobra rather than trusting the name.
if target, _, err := rootCmd.Find(append([]string{command}, args...)); err == nil &&
    target != nil && target != rootCmd {
    names[target.CommandPath()] = true
}
```

and in `computeOrphanNames`, walk with the parent path so `c` is identified as
`aether host colonize` vs `aether colonize`. Regenerating
`testdata/orphan_allowlist.json` afterwards will surface the newly-visible
orphans; each needs a real caller or a baseline entry added in the same review.
`enumerateRegisteredCommands` must also start carrying the path — today
`registeredCommandInfo` has only `Name`, `Aliases`, `Hidden`.

## Warnings

### WR-01: `-update-orphan-allowlist` is an override switch inside a guard file, and the escape-hatch scan does not look for flags

**File:** `cmd/subcommand_reachability_ratchet_test.go:39, 902-906`; `.aether/docs/orphan-allowlist-policy.md:34-38, 95-97`

**Issue:** `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced
-update-orphan-allowlist` regenerates the live list and `return`s **before** the
unallowed-orphan assertion and before the D-08 owner-phase assertion. That is a
command-line flag in a guard file that turns the guard's assertions off.
`TestWiringGuardsHaveNoRuntimeEscapeHatch`'s regex covers `os.Getenv`,
`os.LookupEnv`, `os.Environ`, `syscall.Getenv`, `t.Skip`, `t.SkipNow`,
`testing.Short` — not `flag.Bool`. The policy document states "There is no
override switch, no setting to turn the check off" and that the escape-hatch
test "fails if any of the guard files above ever grow a setting, **flag**, or
environment variable that could switch the check off." Both claims are false as
written. CLAUDE.md's own rule: "A documentation claim about runtime behaviour
must be testable or removed."

**Fix:** Either (a) make the update path assert first and write second, so the
flag cannot suppress a failure, or (b) narrow the doc to describe what the test
actually checks and add `flag.Bool|flag.String|flag.Int` to `forbiddenRe` with
an explicit, commented exemption for `updateOrphanAllowlist` — so a *second*
such flag fails loudly.

### WR-02: The named wiring step is never checked for `if:`, `continue-on-error`, or a swallowed exit status

**File:** `cmd/ci_wiring_gate_test.go:100-108`

**Issue:** `blanketReleaseGateProblem` hardens the blanket step; the named wiring
step gets only `extractWiringGateRunArg`, which reads the `-run` argument and
nothing else. `continue-on-error: true` on the wiring step passes the guard.
Coverage is preserved by the blanket step, so this is a legibility loss rather
than a coverage loss — but the phase's stated purpose for that step is
legibility.

**Fix:** Factor the structural checks out of `blanketReleaseGateProblem` into a
`stepCanFailTheBuild(block, stepName) error` helper and call it for both steps.

### WR-03: Nothing asserts the workflow runs at all

**File:** `.github/workflows/ci.yml:3-11`; `cmd/ci_wiring_gate_test.go`

**Issue:** Every guard inspects steps. None inspects `on:` (currently
`pull_request`/`push` on `main`) or job-level keys. Changing `on:` to
`workflow_dispatch` only, or adding `if: false` to the `go` job, disables all of
it while `TestWiringGateStepRunsEveryWiringTest` and
`TestBlanketGateCheckRejectsADecoyStep` stay green. A gate the workflow never
reaches is not a gate.

**Fix:** Assert in the same test that the workflow's `on:` block contains both
`pull_request:` and `push:`, and that the `go:` job has no `if:` key.

### WR-04: The `-run` ↔ guard-test check is one-directional; a stale name in the CI filter is undetected

**File:** `cmd/ci_wiring_gate_test.go:110-139`

**Issue:** The test proves every guard test is matched by the filter. It does not
prove every alternative in the filter matches a guard test. `go test -run` exits
0 when the pattern matches nothing, so deleting or renaming
`TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` while leaving its name in the
filter silently removes it from the named step with no red test. (Checked: no
alternative is stale today — 24 alternatives, all resolving.)

**Fix:** After the `uncovered` loop, split `runArg` on `|` and fail for any
alternative that matches none of `testNames`.

### WR-05: `TestCallerEvidenceCreditsCommandSubstitution/half_b` fails when the phase's own goal is reached

**File:** `cmd/subcommand_reachability_ratchet_test.go:974-979`

**Issue:** `if len(allowlist) == 0 { t.Fatal("allowlist is empty …") }`. The
allowlist shrinking to zero is the stated objective of the policy document and of
Phase 178. Reaching it turns this guard red. A guard whose failure condition
includes success is a guard that will be edited under pressure at the worst time.

**Fix:** Build the synthetic fixture from a name that is *registered but not
allowlisted* (or a throwaway command registered inside the test body, as
`TestRatchetDetectsASyntheticOrphan` already does at line 1059), so the assertion
is independent of allowlist contents.

### WR-06: `TestAllowlistPolicyNamesEveryGuardedFile` breaks two of the phase's own rules

**File:** `cmd/cli_flag_audit_test.go:44-49, 340-359`

**Issue:** Three problems in one test:
1. It reads `"../.aether/docs/orphan-allowlist-policy.md"` — the exact
   `..`-relative pattern the same file's comment at lines 74-79 says must not be
   used ("a `..`-relative path silently mis-scopes … and cannot be told apart
   from 'directory legitimately moved' (T-172-38)"). Every other test in this
   phase resolves through `repoRootForCommandSourceTest()`.
2. `guardedAllowlistFiles` is a hand-maintained list. A fifth guarded file added
   later is silently exempt from the policy-naming rule — the same "the list
   must be updated by hand" failure the phase attacks elsewhere.
3. The assertion is `strings.Contains(doc, path)`. The document could name every
   path while describing the opposite policy and still pass.

**Fix:** (1) resolve via `repoRootForCommandSourceTest()`; (2) derive the guarded
file list from the testdata directory listing plus the files that literally read
them, or at minimum add a floor assertion (`len(guardedAllowlistFiles) >= 4`)
mirroring the `wiringGateGuardFiles` floor at
`subcommand_reachability_ratchet_test.go:1187`; (3) accept that (3) is inherent
and say so in the test's doc comment rather than implying behavioural coverage.

### WR-07: The policy document's numbers are unguarded and will go stale by design

**File:** `.aether/docs/orphan-allowlist-policy.md:56-66, 29-32`

**Issue:** The doc records "278 commands", "6 … skill-related", "272 … wider
backlog", "exactly 2 entries" — verified accurate today. Nothing asserts them,
and the entire purpose of the ratchet is that 278 goes down. This document is
guaranteed to become wrong. Separately, "A name can never be added to either
list … the automated check … fails immediately, by name" is not true of the two
*baseline* files: adding a name to `orphan_allowlist_baseline.json` and to the
live list passes every test. The doc concedes this two paragraphs later
("The only way either list changes is by editing the saved copy"), so the page
contradicts itself on its own headline rule.

**Fix:** Either assert the counts from the JSON in
`TestAllowlistPolicyNamesEveryGuardedFile` (making the doc fail loudly when it
drifts), or replace the hard numbers with a pointer to the file. Reword line
29-32 to state plainly that the baseline is human-reviewed, not machine-enforced.

### WR-08: `continue-full.md`'s midden call had its arity fixed but still cannot work — the response shape is wrong

**File:** `.aether/docs/command-playbooks/continue-full.md:1244-1246`

**Issue:** This phase changed `aether midden-recent-failures 50` to
`aether midden-recent-failures --limit 50`, which satisfies the cobra contract
audit. The surrounding shell was not changed:

```bash
midden_result=$(aether midden-recent-failures --limit 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')
… jq -r '[.failures[] | .category] …'
```

`cmd/midden_cmds.go:45` emits `{"entries": …, "total": N}`, wrapped by
`outputOK` (`cmd/helpers.go:27`) as `{"ok":true,"result":{…}}`. There is no
`.count` and no `.failures` at any level. `midden_count` is always `0`, so the
entire PHER-02 auto-REDIRECT block is dead. The `|| echo '{"count":0,…}'`
fallback guarantees the failure is silent. This is precisely the class of bug
the phase exists to eliminate, one line away from the line it fixed — and the
audit is structurally unable to see it (`command_call_audit_test.go:26-30`
states this limitation honestly).

**Fix:**

```bash
midden_result=$(aether midden-recent-failures --limit 50 2>/dev/null || echo '{"ok":true,"result":{"entries":[],"total":0}}')
midden_count=$(echo "$midden_result" | jq '.result.total // 0')
recurring_categories=$(echo "$midden_result" | jq -r '[.result.entries[] | .category] | group_by(.) | …')
```

### WR-09: The `generate-commit-message` disclaimer contradicts the five lines under it

**File:** `.aether/docs/command-playbooks/continue-full.md:1535-1541, 1560-1580`

**Issue:** The new line reads "This command returns `message` only — the other
fields below are not produced by this command." It is immediately followed by
"Parse the returned JSON to extract:" listing `message`, `body`,
`files_changed`, `subsystem`, `scope` — and by downstream steps that interpolate
them for real: `git commit -m "{message}" -m "{body}"` and
`Committed: {message} ({files_changed} files)`. Confirmed against
`cmd/generate_cmds.go:118-121`: the command emits `{"message": …}` and folds
`--body` into `message`. Following the doc produces a commit with a duplicated
or empty body. A disclaimer that the next paragraph overrides is worse than no
disclaimer.

**Fix:** Delete the four unproduced bullets, delete the disclaimer, and change
the commit step to `git commit -m "{message}"` (the body is already inside
`message`). Replace `{files_changed}` in the display with the value already
captured from `git diff --stat` two steps earlier.

### WR-10: The fence repair fixed marker placement but left the mangled shell inside the same blocks

**File:** `.aether/docs/command-playbooks/continue-full.md:1274, 1281`

**Issue:** Plan 172-06 repaired glued triple-backtick markers so the audit could
see the blocks. Inside two of the blocks it made visible, statements are still
glued onto continuation lines:

```
        --ttl "30d"      emit_count=$((emit_count + 1))
        --content "Recurring error pattern: $category ($count occurrences)"    fi
```

Both are pre-existing (present at `642e39b3`), but both sit in blocks this phase
touched, and both are unrunnable shell: the `\` continuations were lost and the
next statement absorbed into the flag value. `TestAuditedCorpusHasNoGluedFenceMarkers`
cannot detect them — it only looks for the ``` marker. The corpus sweep therefore
certifies a file whose documented commands do not parse.

**Fix:** Repair both lines (restore the `\` continuation and put `emit_count=…`
/ `fi` on their own lines) while in the file. Longer term, consider extending
the sweep to flag a line inside a fenced block where a `--flag "value"` is
followed by further non-flag content with no `\`.

### WR-11: The AST enumeration floor is 8 against 24 actual guard tests

**File:** `cmd/ci_wiring_gate_test.go:119-123`

**Issue:** `if len(testNames) < 8` guards against a silent AST walk. The real
count across the five guard files is 24 (7 + 9 + 4 + 2 + 2). Two-thirds of the
guard tests could be deleted and the floor would still pass. Compare the sibling
floors, which are set just under the measured value: `scannedFiles < 120` for a
measured 141 (`cli_flag_audit_test.go:218`), `len(paths) < 200` for a measured
215 (`command_call_audit_test.go:1329`).

**Fix:** Raise to `< 20` and record the measured 24 in the message, matching the
convention the other two floors already use.

### WR-12: `extractYAMLRuntimeCommand` reads only the first `command:` field per file

**File:** `cmd/subcommand_reachability_ratchet_test.go:85, 279-281`

**Issue:** `yamlRuntimeCommandRe.FindStringSubmatch` returns the first match. A
`.aether/commands/*.yaml` that declares more than one `command:` line (a
multi-step spec, or a nested key that happens to be named `command`) credits only
the first, so a real menu-driven caller is dropped and its command can be
reported as an orphan. Fails in the safe direction, but produces a false alarm
that a future editor is likely to "fix" by adding an allowlist entry.

**Fix:** Use `FindAllStringSubmatch` and credit every match.

### WR-13: The escape-hatch scan is line-based over five files and is evadable

**File:** `cmd/subcommand_reachability_ratchet_test.go:1199-1228`

**Issue:** `forbiddenRe` is applied per line to comment-stripped source. `os.\n\tGetenv("X")`
(legal, gofmt-stable Go) evades it, as does any helper defined in a
non-guard file, as does any early `return` that makes an assertion unreachable.
The test's doc comment claims "A guard that can be switched off at runtime is
not a guard", which overstates what a five-file line regex can establish.

**Fix:** No cheap complete fix exists; the honest move is to say so in the doc
comment ("this catches the common spellings; it is not a proof") so a future
reader does not over-trust it. Optionally strengthen by scanning the parsed AST
for calls to `os.Getenv`/`t.Skip` rather than raw text, which removes the
line-splitting evasion.

### WR-14: `buildCommandDefinitionIndex` misses cobra commands declared after the first spec of a grouped `var (…)` block

**File:** `cmd/subcommand_reachability_ratchet_test.go:168-265` via `declNameAndBody` (`cmd/visual_writer_discipline_test.go`)

**Issue:** `declNameAndBody` returns on the **first** `ValueSpec` of a `GenDecl`.
For

```go
var (
    aCmd = &cobra.Command{Use: "a"}
    bCmd = &cobra.Command{Use: "b"}   // never walked
)
```

`bCmd` is invisible to the definition index. Two consequences: (a) the command is
reported as "unattributed" in `TestNoRegisteredSubcommandIsUnreferenced`
(fails loudly — acceptable); (b) the self-reference exclusion in
`collectGoSelfInvocationCallers:512` compares `defIndex[name]` against the file
basename, so with an empty index a command that only ever invokes *itself* from
its own definition file would be credited as having an external caller. No such
declaration exists today, so this is latent.

**Fix:** Iterate all `ValueSpec`s. Since `declNameAndBody` is shared, add a
local `allDeclBodies(decl) []ast.Node` in the ratchet file rather than changing
the shared helper's signature.

## Info

### IN-01: The flag audit's regex cannot see a zero-flag invocation at end of line

**File:** `cmd/cli_flag_audit_test.go:89`

**Issue:** `aether\s+([\w][\w-]*)\s+(…)` requires whitespace after the
subcommand name. `aether backup-prune-global` at end of line does not match, so
it never reaches `foundSubcommands` or the registration check. (It is still
covered by the cobra contract audit in the sibling file.)

**Fix:** Change the trailing `\s+` to `(?:\s+|$)`.

### IN-02: `substitutionDepth` counts parentheses inside quoted strings

**File:** `cmd/command_call_audit_test.go:169-175`

**Issue:** `strings.Count(tok, "(")` runs over whole tokens including quoted
values. `aether pheromone-write --content "recurring (3 times" && aether status`
leaves depth at 1, so the second, genuine invocation is discarded as "nested".
A false negative in the audit.

**Fix:** Strip quoted spans before counting, or count only on tokens that
`substitutionOpener` recognises.

### IN-03: Duplicated rationale comment in `knownEnrichmentSubcommands`

**File:** `cmd/command_call_audit_test.go:1051-1056, 1194-1199`

**Issue:** The same four-line justification is written twice — once above
`backup-prune-global` covering both commands, once above `temp-clean` covering
one. Two copies of a rationale drift.

**Fix:** Keep one, and reference it from the second entry.

### IN-04: Two path conventions for the one shared guard-file inventory

**File:** `cmd/subcommand_reachability_ratchet_test.go:1202` vs `cmd/ci_wiring_gate_test.go:112`

**Issue:** `wiringGateGuardFiles` holds bare basenames. One consumer does
`os.ReadFile(f)` (cwd-relative), the other `filepath.Join(repoRoot, "cmd", f)`.
Both work only because `go test` sets cwd to the package directory — the same
assumption `TestCLIFlagAudit`'s comment (lines 74-79) explicitly refuses to make.

**Fix:** Resolve both through `repoRootForCommandSourceTest()`.

### IN-05: `stepBlock` runs to EOF when the target step is last, producing wrong diagnoses

**File:** `cmd/ci_wiring_gate_test.go:182-186`

**Issue:** When no next `- name:` at the same indent exists, the block is the
remainder of the file, so the `if:` and `continue-on-error` scans read unrelated
later steps. This is what produced the false message
`CI step "Run Go tests" has a conditional if: always()` in CR-02, for a step
that had been deleted. Misleading failure text on a safety guard costs debugging
time at exactly the wrong moment.

**Fix:** Covered by the CR-02 fix (bound the block by indentation).

---

_Reviewed: 2026-08-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
