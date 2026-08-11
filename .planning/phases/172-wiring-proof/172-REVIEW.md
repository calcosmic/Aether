---
phase: 172-wiring-proof
reviewed: 2026-08-11T16:47:07Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - cmd/spawn.go
  - cmd/spawn_enforce_test.go
  - cmd/command_call_audit_test.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/cli_flag_audit_test.go
  - cmd/ci_wiring_gate_test.go
  - cmd/testdata/orphan_allowlist.json
  - cmd/testdata/orphan_allowlist_baseline.json
  - cmd/testdata/flag_audit_skiplist_baseline.json
  - cmd/testdata/command_catalog.json
  - .github/workflows/ci.yml
  - .aether/workers.md
  - .aether/docs/orphan-allowlist-policy.md
  - .aether/docs/command-playbooks/build-full.md
  - .aether/docs/command-playbooks/build-wave.md
  - .aether/docs/command-playbooks/continue-advance.md
  - .aether/docs/command-playbooks/continue-gates.md
findings:
  critical: 7
  warning: 13
  info: 4
  total: 24
status: issues_found
---

# Phase 172: Code Review Report

**Reviewed:** 2026-08-11T16:47:07Z
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found

## Summary

The whole suite is green (`go test ./cmd -run '<the CI regex>'` passes in 2.3s), and the
structural design is sound: both shrink-only comparators `t.Fatalf` on a missing or
unparseable data file, so neither can be disabled by deleting a file; neither compares counts,
so a one-out-one-in swap fails in both; and the CI-gate guard genuinely derives its expected
test list from the guard files' ASTs rather than from a second hand-written copy.

It is still defeatable in seven distinct ways, and the most serious is not theoretical. **The
exact bug class this phase exists to catch survives, live, in the audited corpus, with the
audit reporting green:** `.aether/docs/command-playbooks/continue-full.md:1243` still reads
`aether midden-recent-failures 50` — a bare positional handed to a `cobra.NoArgs` command,
byte-identical in shape to the four calls this phase fixed in `build-full.md`,
`build-wave.md` (×2) and `continue-advance.md`. It survives because the extractor's code-fence
tracker is inverted from line 1194 onward by a closing fence glued to the end of a content
line (`--ttl "30d"` + fence). A guard that misses a live instance of its own founding bug is
the phase's central claim failing on its own terms.

The other blockers cluster in three places: (a) guard-on-guard checks that can be satisfied by
text that cannot fail (`ci.yml`'s `Test summary` step), (b) the escape-hatch scan being narrower
than the set of guard files and narrower than the set of ways to read an environment variable,
and (c) two of the phase's own remediations to shipped instruction files (`workers.md`,
`continue-gates.md`) that make the audit green while leaving the instruction unable to work —
which is worse than the broken call, because the audit can no longer see it.

Verified explicitly and found **correct** (recorded so they are not re-litigated): the two
shrink-only comparators are behaviourally equivalent on the swap case; `--enforce` really does
reach a non-zero process exit (`outputError` → `markRenderedCommandError` →
`Execute`/`renderedErrorExit`), and its presence changes nothing on the allow path;
`repoRootForCommandSourceTest` fails loudly rather than silently mis-scoping. The known
duplication of the shrink comparator is out of scope per the brief and is not reported.

## Critical Issues

### CR-01: A live instance of the phase's founding bug survives the audit, unseen

**File:** `cmd/command_call_audit_test.go:169-184` (fence tracking); evidence at
`.aether/docs/command-playbooks/continue-full.md:1194` and `:1243`

**Issue:** `extractDocumentedCalls` toggles `inFence` only when a line *starts* with a fence
marker:

```go
if strings.HasPrefix(strings.TrimSpace(line), "```") {
    inFence = !inFence
    continue
}
```

`continue-full.md:1194` closes a block with the marker glued to the end of a content line
(`  --ttl "30d"` immediately followed by the fence). That line does not start with the marker,
so the fence never closes and every fence-parity decision from line 1194 to EOF is inverted.
Two files in the audited corpus have an odd fence count for exactly this reason
(`continue-advance.md`, 49 markers; `continue-full.md`, 149 markers — the latter has ~600
inverted lines).

The consequence is a proven false negative. At `continue-full.md:1243` the extractor computes
`inFence == false`, the line carries no backticks, so `backtickCallRe` does not match and the
call is never extracted at all:

```
midden_result=$(aether midden-recent-failures 50 2>/dev/null || ...)
```

`midden-recent-failures` is `Args: cobra.NoArgs` (`cmd/midden_cmds.go:20`). This is the same
positional-to-NoArgs shape the phase's own comment calls "the exact shape of all seven of the
phase's confirmed-broken calls" — and `TestCommandCallsMatchCobraContracts` passes.

**Fix:** treat a fence marker anywhere on a line as a delimiter, and split the line at it, so
a glued close still closes:

```go
trimmed := strings.TrimSpace(line)
if idx := strings.Index(trimmed, "```"); idx >= 0 {
    if idx > 0 && inFence {
        // content before a glued closing fence is still a fenced line
        if c, ok := parseFencedInvocation(path, i+1, trimmed[:idx]); ok {
            calls = append(calls, c)
        }
    }
    inFence = !inFence
    continue
}
```

Add a regression assertion that fence parity is even for every audited file, so an unbalanced
fence fails loudly instead of silently halving coverage:

```go
if inFence {
    t.Errorf("%s ends inside an unclosed code fence — every fence-parity decision after the "+
        "unbalanced marker is inverted and invocations are silently dropped", path)
}
```

Then fix `continue-full.md:1243` to `--limit 50` alongside the four already corrected.

---

### CR-02: The "CI coverage was not narrowed" guard is satisfied by a step that cannot fail

**File:** `cmd/ci_wiring_gate_test.go:67-71`; evidence at `.github/workflows/ci.yml:42` and
`:105`

**Issue:** The guard asserts the blanket release-gate step still exists by substring:

```go
if !strings.Contains(workflow, "go test ./... -count=1 -timeout 900s") {
```

Two lines in `ci.yml` contain that substring. Line 42 is the real gate. Line 105 is the
`Test summary` step:

```yaml
      - name: Test summary
        if: always()
        run: |
          echo "Go tests: $(go test ./... -count=1 -timeout 900s -v 2>&1 | grep -c '--- PASS' || echo 'unknown') passed"
```

That step runs under `if: always()`, its exit status is `echo`'s, and the `|| echo 'unknown'`
swallows any failure — it can never fail a build. So deleting line 42 (the actual release
gate) leaves this guard passing. The guard's stated purpose — "it must never be mistaken for a
replacement that narrows CI coverage" — does not hold.

**Fix:** parse the workflow structurally and require a *failing-capable* step, not a substring.
Minimum viable fix: locate the `- name: Run Go tests` step and assert its own `run:` line,
and assert the summary step is excluded:

```go
runGoTests, err := stepRunLine(workflow, "Run Go tests")
if err != nil || !strings.Contains(runGoTests, "go test ./... -count=1 -timeout 900s") {
    t.Fatalf("the `Run Go tests` release-gate step no longer runs the blanket suite: %v / %q", err, runGoTests)
}
if strings.Contains(stepBlock(workflow, "Run Go tests"), "if: always()") {
    t.Fatal("the blanket release gate must be able to fail the build; it must not be `if: always()`")
}
```

(`stepRunLine` can reuse the existing `extractWiringGateRunArg` scanning logic, parameterised
on the step name.)

---

### CR-03: The escape-hatch guard misses `os.LookupEnv` and does not cover two of the five guard files

**File:** `cmd/subcommand_reachability_ratchet_test.go:1175-1210`

**Issue:** Two independent holes in the same test.

1. The forbidden pattern is `os\.Getenv|t\.Skip|t\.SkipNow`. `os.LookupEnv("AETHER_SKIP_WIRING")`
   is the idiomatic alternative and is **not** matched; neither is `os.Environ()`,
   `syscall.Getenv`, `testing.Short()`, or a `runtime.GOOS` bail-out. The test's own failure
   message claims it catches "an environment-variable read"; it catches one spelling of one.
2. `guardFiles` lists three files:

```go
guardFiles := []string{
    "subcommand_reachability_ratchet_test.go",
    "command_call_audit_test.go",
    "cli_flag_audit_test.go",
}
```

   `ci_wiring_gate_test.go` and `spawn_enforce_test.go` — both added by this phase, both listed
   as guard files by `wiringGateGuardFiles` in `cmd/ci_wiring_gate_test.go:36-42` — are not
   scanned. `ci_wiring_gate_test.go` is the test that keeps the CI `-run` filter honest, and it
   is the one guard here that can be switched off with a `t.Skip` and nothing will notice.

**Fix:** widen the pattern and share one file list:

```go
forbiddenRe := regexp.MustCompile(`os\.Getenv|os\.LookupEnv|os\.Environ|syscall\.Getenv|t\.Skip|t\.SkipNow|testing\.Short`)

for _, f := range wiringGateGuardFiles { // the single, shared inventory
    ...
}
```

`wiringGateGuardFiles` already exists in the same package; use it in both places and delete
the local `guardFiles` slice (see WR-06).

---

### CR-04: `TestCLIFlagAudit` passes vacuously when its input directories are absent

**File:** `cmd/cli_flag_audit_test.go:67-72`, `:111-115`, `:190-192`

**Issue:** Every other guard in this phase has an anti-vacuity assertion. This one has none.

```go
markdownDirs := []string{
    "../.claude/commands/ant/",
    "../.opencode/commands/ant/",
    "../.aether/docs/command-playbooks/",
}
...
entries, err := os.ReadDir(dir)
if err != nil {
    continue // directory may not exist in test environment
}
```

If all three directories are renamed, moved, or removed (the ratchet's own comments
contemplate exactly this: "corpus removed by a later phase"), the loop reads nothing,
`mismatches` stays empty, and the test **passes**. The only signal is a `t.Logf` nobody reads.
The paths are also `..`-relative rather than resolved through `repoRootForCommandSourceTest()`,
so any change to the test working directory silently disables it too.

This test is named in `.aether/docs/orphan-allowlist-policy.md:90` as one of the tests
enforcing the policy, and it is one of the tests the CI wiring step exists to make legible. A
guard that no-ops when its input is missing is a guard that can be disabled by moving a
directory.

**Fix:**

```go
root, err := repoRootForCommandSourceTest()
if err != nil {
    t.Fatalf("resolve repo root: %v", err)
}
markdownDirs := []string{
    filepath.Join(root, ".claude", "commands", "ant"),
    filepath.Join(root, ".opencode", "commands", "ant"),
    filepath.Join(root, ".aether", "docs", "command-playbooks"),
}
...
scannedFiles := 0   // increment per file actually read
...
if scannedFiles == 0 || len(foundSubcommands) == 0 {
    t.Fatalf("the flag audit read %d file(s) and found %d subcommand(s) — it is not looking at "+
        "anything, which would pass vacuously forever", scannedFiles, len(foundSubcommands))
}
```

---

### CR-05: `openedSubstitution` recognises one of the three openers `normalizeShellToken` accepts

**File:** `cmd/command_call_audit_test.go:111-142`, use sites `:229-235` and `:288-294`

**Issue:** `normalizeShellToken` strips three substitution openers:

```go
case strings.HasPrefix(t, "$("): t = t[2:]
case strings.HasPrefix(t, "`"):  t = t[1:]
case strings.HasPrefix(t, "("):  t = t[1:]
```

`openedSubstitution` — the function that decides whether to trim the matching closing
delimiter off the last token — recognises only `$(`:

```go
return strings.HasPrefix(t, "$(")
```

So for the other two openers the closing delimiter is never trimmed. Verified against a
standalone reproduction of both functions:

| Input | Extracted | Result |
|---|---|---|
| `` x=`aether status` `` | name `` status` `` | fails the `^[a-z][a-z0-9-]*$` shape check → **call silently dropped** |
| `(aether status)` | name `status)` | **call silently dropped** |
| `(aether status --json)` | name `status`, args `[--json)]` | **false positive**: reported as `unknown flag --json)` |

The doc comment on `normalizeShellToken` explicitly claims the two call sites "cannot drift
apart again". They have not drifted from each other — but `normalizeShellToken` and
`openedSubstitution` have, and they are two halves of the same decision. No live instance
exists in the corpus today, which is precisely why this will not be noticed until it produces
a wrong answer.

**Fix:** make the two functions symmetric by construction:

```go
// substitutionOpener returns the opener tok begins with (after an optional
// VAR= prefix) and the closing delimiter that matches it, or "" for neither.
func substitutionOpener(tok string) (open, closeDelim string) {
	t := tok
	if loc := assignmentPrefixRe.FindStringIndex(t); loc != nil {
		t = t[loc[1]:]
	}
	switch {
	case strings.HasPrefix(t, "$("):
		return "$(", ")"
	case strings.HasPrefix(t, "`"):
		return "`", "`"
	case strings.HasPrefix(t, "("):
		return "(", ")"
	}
	return "", ""
}
```

and trim `closeDelim` (not a hardcoded `")"`) at both use sites. Add table-driven cases for
`` `aether status` ``, `(aether status)` and `(aether status --json)` to
`TestCommandCallExtractorSeesRealInvocationsAndSkipsProse`.

---

### CR-06: The `workers.md` remediation makes the audit green with a call that can only ever fail

**File:** `.aether/workers.md:329` and `:376`; contract at `cmd/swarm.go:261-287`

**Issue:** This phase rewrote both `swarm-display-update` calls from a positional form (which
the audit correctly rejected) to:

```bash
aether swarm-display-update --agent "{child_name}" --id "{your_name}" --status "excavating"
```

`--id` is not the caller's identity. It is the **swarm id**, used as a storage path segment
(`cmd/swarm.go:283`):

```go
path := fmt.Sprintf("swarms/%s/display.json", id)
if err := store.LoadJSON(path, &df); err != nil {
    outputError(1, fmt.Sprintf("display not found for swarm %s: %v", id, err), nil)
```

Passing `{your_name}` (a worker name like `Mason-67`) makes the runtime look for
`swarms/Mason-67/display.json`. Worse, `swarm-display-init` — the only thing that creates that
file — appears **nowhere** in the entire `.claude`/`.opencode`/`.aether` corpus (verified by
grep: the only two hits for `swarm-display-*` in the whole tree are these two `workers.md`
lines). Every execution of this documented step therefore exits non-zero with
`display not found for swarm {your_name}`.

The rewrite also silently dropped the caste, task summary, tool counts, progress and chamber
that the previous form carried — those are not flags on this command at all, so the
information has nowhere to go.

Net effect: the call went from "detectably broken" to "undetectably broken". The audit checks
arity and flag existence; it has no way to see that the value is the wrong identity. This is
the failure mode CLAUDE.md's Definition of Done section describes.

**Fix:** either supply the real swarm id and initialise the display, or remove the step:

```bash
# only valid inside a swarm, where {swarm_id} was created by swarm-display-init
aether swarm-display-update --id "{swarm_id}" --agent "{child_name}" --status "excavating"
```

If the generic spawn protocol has no swarm id (it does not — spawns are logged via
`spawn-log`), delete both lines rather than leaving an instruction that always errors.

---

### CR-07: The `continue-gates.md` remediation reads the wrong JSON level and can only return `{}`

**File:** `.aether/docs/command-playbooks/continue-gates.md:875`

**Issue:** The phase rewrote a broken `aether state-read '.build_synthesis.watcher'` (a
positional to a `cobra.NoArgs` command) into:

```bash
watcher_result=$(aether state-read 2>/dev/null | jq -c '.build_synthesis.watcher // {}' 2>/dev/null || echo "{}")
quality_score=$(echo "$watcher_result" | jq -r '.quality_score // 0')
critical_count=$(echo "$watcher_result" | jq '[.issues_found[]? | select(.severity == "CRITICAL")] | length')
```

`state-read` emits the standard envelope (`cmd/state_cmds.go:883` → `outputOK(state)` →
`{"ok":true,"result":{…}}`). Verified against the real binary:

```
$ aether state-read | jq -c 'keys'
["ok","result"]
```

`.build_synthesis.watcher` at the top level is structurally always `null`, so `// {}` always
yields `{}`, `quality_score` is always `0` and `critical_count` is always `0`. The quality gate
that reads Watcher results now reports a clean pass unconditionally. (`.result.build_synthesis`
does not exist either — `build_synthesis` appears nowhere in the Go sources — so the state key
itself is stale, but that is a separate, pre-existing problem; the introduced defect is the
missing `.result` level in a newly written expression.)

Like CR-06, the audit now reports this line as correct, so the breakage is no longer visible to
the machinery built to find it.

**Fix:** use the purpose-built subcommand, which exists (`cmd/state_cmds.go:889`):

```bash
watcher_result=$(aether state-read-field --field 'build_synthesis.watcher' 2>/dev/null | jq -c '.result.value // {}' || echo "{}")
```

and separately confirm whether `build_synthesis` is written by anything; if it is not, the whole
step is dead and should be removed rather than repaired.

---

## Warnings

### WR-01: Only the first `aether` invocation on a line is ever extracted

**File:** `cmd/command_call_audit_test.go:193-200` and `:260-267`

**Issue:** Both binary-detection loops `break` at the first match, so
`aether continue && aether build 2` yields one call (`continue`, with `&& aether build 2` as
args, truncated at `&&`). The second invocation is never contract-checked, and in
`singleFileCallerNames` it gets no caller credit either — `credit()` stops at the `&&`
operator. The `.sh`/`.js` branch of `singleFileCallerNames` (`:408-423`) does **not** break and
handles multiple invocations per line, so the markdown and shell paths disagree.

**Fix:** replace the `break` with a scan that yields every binary token on the line, and have
`validateCallAgainstCobra` truncate each at its own operator boundary.

---

### WR-02: Quoted command substitution `X="$(aether …)"` is invisible

**File:** `cmd/command_call_audit_test.go:111-128`

**Issue:** `tokenizeShellLike` keeps quoted runs together, so `RESULT="$(aether state-read --json)"`
is a single token beginning with `"`. `normalizeShellToken` strips `VAR=`, `$(`, `` ` `` and
`(` — but not a leading quote — so the token never normalises to `aether` and the invocation is
dropped entirely (verified by reproduction). This is the same blind-spot class 172-00 claims to
have closed; the fix was applied to the unquoted form only. No live instance in the corpora
today.

**Fix:** strip leading `"`/`'` in `normalizeShellToken` before the opener loop, and add
`RESULT="$(aether skill-detect)"` to the extractor's fixture.

---

### WR-03: A commented-out invocation in a shell script counts as caller evidence

**File:** `cmd/subcommand_reachability_ratchet_test.go:401-423`

**Issue:** The `.sh`/`.js` branch tokenises every line with no comment handling, so a `#`-prefixed
line credits its command. Live instances:

```
scripts/smoke-daily-driver.sh:7  #   1. `aether update --force` in a repo with foreign (GSD) Claude settings
scripts/smoke-daily-driver.sh:10 #   3. `aether host plan --dry-run` runs the TS host without any
```

These credit `update`, `host` and `plan` as "having a caller". The markdown branch correctly
rejects the equivalent (a fenced `# aether foo` is rejected by the preceding-token check at
`command_call_audit_test.go:271-280`); the shell branch does not. The policy document promises
"code that actually invokes it" — a comment is not code. No command is masked by this today
(all three have real callers), but the mechanism is live and the corpus is small enough that one
new script comment could clear a genuine orphan.

**Fix:** skip lines whose first token starts with `#` (and JS `//` / `/*`) in the `.js`/`.sh`
branch of `singleFileCallerNames`.

---

### WR-04: `credit()` grants caller credit to positional values that happen to be command names

**File:** `cmd/subcommand_reachability_ratchet_test.go:374-388`

**Issue:** After crediting the command, `credit` walks forward crediting every following token
that matches `^[a-z][a-z0-9-]*$`, stopping only at a flag, placeholder or operator. So
`aether export pheromones` credits both `export` and `pheromones` — and because
`computeOrphanNames` is keyed by bare name (`:713-739`), the top-level `pheromones` command is
cleared by a call that only ever reaches `export pheromones`. Any documented
`aether <parent> <word>` where `<word>` is also a top-level command name has the same effect.

**Fix:** credit multi-word paths as paths, not as independent names — resolve
`[command, args…]` through `rootCmd.Find` and credit the resolved command's full path, so a
leaf only clears the parent it actually sits under.

---

### WR-05: Menu-entry evidence recognises only a field literally named `command:`, and only the first one

**File:** `cmd/subcommand_reachability_ratchet_test.go:85`, `:274-300`

**Issue:** `yamlRuntimeCommandRe` is `^\s*command:\s*"(.*)"\s*$` and
`extractYAMLRuntimeCommand` uses `FindStringSubmatch` (first match only). The two flagship menu
specs do not use that field name:

```yaml
# .aether/commands/build.yaml:5-6
  manifest_command:  "aether build $ARGUMENTS --plan-only"
  finalizer_command: "aether build-finalize $ARGUMENTS --completion-file <…>"
```

`continue.yaml` is the same shape. So D-03's "a menu entry counts as a caller" contributes zero
evidence for `build`, `build-finalize`, `continue` and `continue-finalize` — they are saved only
by incidental backticked mentions in the wrapper markdown. Any spec that grows a second
`command:` field also loses the second one.

**Fix:** match `^\s*[a-z_]*command:\s*"(.*)"\s*$` and use `FindAllStringSubmatch`, crediting
every hit.

---

### WR-06: Three overlapping, hand-maintained "guard file" inventories that can drift apart

**Files:** `cmd/subcommand_reachability_ratchet_test.go:1176-1180` (3 entries),
`cmd/ci_wiring_gate_test.go:36-42` (5 entries), `cmd/cli_flag_audit_test.go:43-48` (4 paths)

**Issue:** The same concept — "the files this phase's guards live in" — is written down three
times with three different memberships, and nothing asserts the three are consistent or
complete. Adding a sixth guard file is invisible to all three: its tests are not required in the
CI `-run` filter, it is not escape-hatch scanned, and it need not appear in the policy document.
This is the drift vector the phase exists to eliminate, reproduced inside the phase's own code.
CR-03 is the first concrete consequence.

**Fix:** declare one exported-within-package inventory (e.g. `wiringGuardFiles`) and derive all
three uses from it; assert it covers every `*_test.go` in `cmd/` that declares a test named in
the CI step's `-run` regex.

---

### WR-07: The policy document hardcodes counts nothing asserts, which go stale on the first success

**File:** `.aether/docs/orphan-allowlist-policy.md:56-66`

**Issue:** "The unused-command list currently holds **278 commands** … **6** are tagged … The
flag-check exceptions list holds exactly 2 entries." `TestAllowlistPolicyNamesEveryGuardedFile`
checks only that four *paths* appear in the document; the numbers are unchecked prose. The first
time the list shrinks — the entire point of the ratchet — the document becomes wrong and nothing
fails. CLAUDE.md: "A documentation claim about runtime behaviour must be testable or removed."

**Fix:** either assert the numbers, or remove them:

```go
live := loadOrphanAllowlist(t, "testdata/orphan_allowlist.json")
if !strings.Contains(doc, fmt.Sprintf("**%d commands**", len(live))) {
    t.Errorf("policy doc states a stale entry count; the live list holds %d", len(live))
}
```

---

### WR-08: The policy document's absolutist guarantees are stronger than the guards

**File:** `.aether/docs/orphan-allowlist-policy.md:29-38`

**Issue:** Two claims overstate what is enforced:

- *"A name can never be added to either list."* It can: edit `orphan_allowlist.json` **and**
  `orphan_allowlist_baseline.json` in the same commit and `TestOrphanAllowlistOnlyShrinks`
  passes. The design intent (D-11) is that this is a *visible* edit, not an impossible one — but
  the document states impossibility.
- *"There is no override switch, no setting to turn the check off."* `-update-orphan-allowlist`
  (`cmd/subcommand_reachability_ratchet_test.go:39`, `:902-906`) causes
  `TestNoRegisteredSubcommandIsUnreferenced` to write the live list and `return` **before any
  assertion runs**. It is a Go test flag rather than an environment variable, and
  `TestOrphanAllowlistOnlyShrinks` still catches the additions afterwards — but "no switch" is
  not accurate.

**Fix:** restate honestly: "the only way a name is added is by editing both checked-in files in
the same commit, which shows up in review", and name the regeneration flag and what it does and
does not bypass.

---

### WR-09: The skill-lifecycle tag covers 6 names, the code says 8, and the D-08 loop is a no-op for the other 2

**File:** `cmd/subcommand_reachability_ratchet_test.go:97-106`, `:936-942`;
`.aether/docs/orphan-allowlist-policy.md:60`

**Issue:** `skillLifecycleOrphanCandidates` holds 8 names and the comment calls it "the set
Phase 178's success criterion measures reaching zero". Only 6 are in the allowlist —
`skill-parse-frontmatter` and `skill-cache-rebuild` have callers. The D-08 verification loop only
fires for names it *finds* in the allowlist, so for those two it does nothing at all, silently.
The policy doc says 6, the code comment says 8. Phase 178's finish line is therefore ambiguous.

**Fix:** assert the intended relationship rather than only checking present entries:

```go
for name := range skillLifecycleOrphanCandidates {
    entry := findAllowlistEntry(allowlist, name)
    if entry == nil {
        continue // already has a caller — record it, don't silently skip
    }
    if entry.OwnerPhase != "178" { t.Errorf(...) }
}
t.Logf("skill-lifecycle set: %d of %d still orphaned", n, len(skillLifecycleOrphanCandidates))
```

and reconcile the two documents on one number.

---

### WR-10: `workers.md` documents a `spawn-can-spawn` payload the command does not emit, and a deny flow `--enforce` makes impossible

**File:** `.aether/workers.md:292-296`; `cmd/spawn.go:200-216`

**Issue:** Line 293 states:

```
# Returns: {"can_spawn": true/false, "depth": N, "max_spawns": N, "current_total": N}
```

The command returns only `can_spawn` and `depth`. `max_spawns` and `current_total` are never
emitted by any code path.

Line 296 then says "If `can_spawn` is false, complete the work inline." With `--enforce` — which
line 292 now passes — a deny writes an error envelope to **stderr** and nothing to stdout, so
`result` is empty and the worker cannot read `can_spawn` at all. Under a `set -e` shell the
capture aborts the script. The documented read-the-answer flow and the enforce-and-exit flow are
mutually exclusive, and the manual describes both.

The phase's own test only asserts `can_spawn` and `depth`, so the documented-but-absent fields
are not caught.

**Fix:** delete `max_spawns`/`current_total` from line 293 (or emit them), and rewrite 294-296 to
describe the enforce contract: non-zero exit means stop, there is no payload to read. Extend
`TestSpawnCanSpawnAcceptsDocumentedInvocation` to assert the documented key set matches the
emitted key set exactly.

---

### WR-11: `spawn-can-spawn` fails open on a missing or negative depth

**File:** `cmd/spawn.go:189-198`, `:406`

**Issue:** `--depth` is registered with help text "(required)" but nothing enforces it. With no
positional and no flag, `mustGetInt` returns the zero value and the command answers for depth
`0` — the most permissive value — with no indication the caller forgot the argument. A negative
depth is likewise accepted silently. Today the decision seam returns `true` unconditionally so
nothing observable changes; the moment Phase 173 implements a real cap, a caller that omits the
argument gets an allow rather than an error.

**Fix:** require the depth explicitly:

```go
if len(args) == 0 && !cmd.Flags().Changed("depth") {
    outputError(1, "depth is required: pass it positionally (`spawn-can-spawn 3`) or as --depth", nil)
    return nil
}
if depth < 0 {
    outputError(1, fmt.Sprintf("invalid depth %d: must be >= 0", depth), nil)
    return nil
}
```

---

### WR-12: The command-call audit's anti-vacuity guard is pinned to one file, not to each corpus

**File:** `cmd/command_call_audit_test.go:311-315`, `:548-565`

**Issue:** `collectDocumentedCalls` silently `continue`s past any missing corpus, and the
anti-vacuity check requires only that (a) total calls > 0 and (b) `.aether/workers.md`
contributed at least one. Deleting or renaming `.claude/commands/ant` — the live surface where a
broken call reaches a user today — silently removes it from the audit while both checks still
pass. The comment at `:548-555` states exactly why this matters ("a corpus that is listed but
never actually read produces a test that passes…") and then applies the remedy to one file.

**Fix:** track per-corpus contribution and fail on any corpus that is present-but-silent or
absent:

```go
if contributed[corpus] == 0 {
    t.Fatalf("corpus %s contributed zero extracted calls — listed but not read", corpus)
}
```

---

### WR-13: The midden playbook lines were arity-fixed but their consumers still cannot read the output

**File:** `.aether/docs/command-playbooks/build-full.md:982`,
`build-wave.md:511`, `:803`, `continue-advance.md:151`, `:553`

**Issue:** The phase corrected `midden-recent-failures 50` → `--limit 50` on five lines. The very
next line in each block is unchanged and cannot read the result. Verified against the real
binary:

```
$ aether midden-recent-failures --limit 3
{"ok":true,"result":{"entries":[],"total":0}}
$ … | jq '.count // 0'
0
```

The playbooks read `.count` and `.failures[]`; the real fields are `.result.total` and
`.result.entries`. `midden_count` is therefore `0` on every run, and all four midden-threshold
blocks (the auto-REDIRECT learning loop) are dead. The `|| echo '{"count":0,"failures":[]}'`
fallback makes the dead path indistinguishable from a genuinely empty midden.

This parsing bug predates the phase, but the phase edited these exact lines and left them in a
state where the call succeeds and the caller still gets nothing — the "wired but dead" pattern.

**Fix:** on each of the five sites:

```bash
midden_result=$(aether midden-recent-failures --limit 50 2>/dev/null | jq -c '.result // {"entries":[],"total":0}' || echo '{"entries":[],"total":0}')
midden_count=$(echo "$midden_result" | jq '.total // 0')
```

and update the downstream `.failures[]` references to `.entries[]`.

---

## Info

### IN-01: `--` terminator is not honoured in the cobra validator

**File:** `cmd/command_call_audit_test.go:482`

`case a == "--": continue` skips the terminator but keeps treating subsequent `--x` tokens as
flags. Everything after `--` should be classified as a positional. No live instance today.

---

### IN-02: `tokenizeShellLike` swallows the rest of a line on an unmatched `<` or `{`

**File:** `cmd/command_call_audit_test.go:425-430`

`<` sets `quote = '>'` and `{` sets `quote = '}'` with no fallback, so
`aether phase <n --json` produces the single token `<n --json`. Documented placeholders are the
motivation and are handled; an unbalanced metacharacter merges arbitrary arguments.

---

### IN-03: A test writes into the checked-in source tree

**File:** `cmd/subcommand_reachability_ratchet_test.go:769-787`

`writeOrphanAllowlist` does `os.WriteFile("testdata/orphan_allowlist.json", …)` with a hardcoded
relative path from inside a running test. It is opt-in behind a flag and D-11 justifies the
design, but a test that mutates tracked files deserves a guard that the path resolves under the
repo root it was asked about.

---

### IN-04: Minor hygiene in `cli_flag_audit_test.go`

**File:** `cmd/cli_flag_audit_test.go:120`, `:157`, `:291`

`dir + entry.Name()` relies on every entry in `markdownDirs` carrying a trailing slash (use
`filepath.Join`); `flagRe` is `regexp.MustCompile`d inside the innermost loop rather than at
package scope; and `../.aether/docs/orphan-allowlist-policy.md` is `..`-relative while the rest
of the phase resolves through `repoRootForCommandSourceTest()`. The same in-loop
`regexp.MustCompile(`^[a-z][a-z0-9-]*$`)` appears twice in
`cmd/command_call_audit_test.go:236` and `:295`; `subcommandNameShapeRe` already exists for this.

---

_Reviewed: 2026-08-11T16:47:07Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
