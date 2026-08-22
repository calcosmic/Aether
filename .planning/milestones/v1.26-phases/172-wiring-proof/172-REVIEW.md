---
phase: 172-wiring-proof
reviewed: 2026-08-12T13:20:00Z
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
  critical: 1
  warning: 8
  info: 8
  total: 17
status: issues_found
---

# Phase 172: Code Review Report (re-review after 172-12 / 172-13, round 4)

**Reviewed:** 2026-08-12T13:20:00Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

172-12 and 172-13 did what they claimed. I independently re-derived the
arithmetic, re-hashed the frozen anchor, re-ran the guard suite, and — since
the sandbox in this session refuses to touch `.github/workflows/ci.yml`
directly (the same restriction 172-12's own SUMMARY records) — reproduced
two of the five prior Criticals against `auditWorkflowShape` in isolation,
via a throwaway `_test.go` file that mutates the hermetic
`testShapeBaseWorkflow` string rather than the live file, then deleted it and
confirmed `git status --porcelain` was empty again:

```
job "go" key "continue-on-error" ... is not on the reviewed whitelist gateJobAllowedKeys [runs-on steps]
step "Run Go tests" key "env" ... is not on the reviewed whitelist gateStepAllowedKeys [name run]
```

That confirms CR-01 and CR-02 fire on the exact mechanism the SUMMARY
describes, and reading `auditWorkflowShape` end to end confirms the same
default-deny structure closes CR-03 (`releaseTriggerAllowedKeys = []string{"branches"}`
rejects `paths-ignore` by absence, not by name). I independently recomputed
CR-04's full-path accounting (272 trivial + 12 non-trivial across 6 leaves =
284; 284 + 9 = 293 — matches both the plan and the checked-in JSON), and
re-hashed the frozen anchor myself (`shasum -a 256` → `873cad5a...45e0e`,
matching `preMigrationSnapshotSHA256` exactly). `go test ./cmd -run
'<the guard chain>'` is green.

**But this round introduces one new Critical of its own, and it is the same
shape of gap the prior three rounds kept producing: the guard checks the
*shapes of tampering it enumerated* — a literal YAML `env:` key at three
scopes — and leaves an adjacent, more deniable route to the identical outcome
completely unaudited.** `auditWorkflowShape` and `gateStepAllowedKeys` only
ever inspect the two named gate steps' own YAML. Every other step in the same
`go` job — Checkout, Setup Go, Setup Node, Install goreleaser, Validate
goreleaser config, Build, Vet, and eight more — runs arbitrary shell on the
same persistent runner before the gate step does, with `run:` content that
is not on any whitelist, not pinned to any equality check, and not scanned by
anything in this phase. Any one of them can set `GOFLAGS`/`GOTOOLCHAIN` for
every later step in the job by writing to `$GITHUB_ENV` — a standard,
documented GitHub Actions mechanism, not an exploit — or overwrite the `go`
binary on `$PATH` outright, since steps in a job share one filesystem. Detail
in CR-06 below. This is not a defeat of any of the three named mechanisms
(the pin, the execution harness, the shape whitelist) — it sits outside all
three by construction, because all three were built to inspect the *content
of the gate step itself*, and this class of tampering never touches that
step's text at all.

Every prior Warning and Info this phase-4 gap-closure round did not target
is unchanged. WR-06 (policy doc arithmetic/headline claim) is now genuinely
true and closed. WR-04 is closed on its comment-correction half. Everything
else — WR-01, WR-02, WR-03, WR-05, WR-07, WR-08, WR-09, WR-10, and all eight
Info items — is untouched, confirmed by direct inspection of the current
source at the cited lines.

---

## Adjudication of Prior Findings

| ID | Verdict | Mechanism / Evidence |
|---|---|---|
| **CR-01** (job-level `continue-on-error` invisible) | **CLOSED** | `gateJobAllowedKeys = []string{"runs-on", "steps"}` (ci_wiring_gate_test.go:1182) is consulted by `auditWorkflowShape`'s rule at ci_wiring_gate_test.go:1400-1406, which rejects any job key absent from the list. Independently reproduced in isolation (hermetic mutation of `testShapeBaseWorkflow`, not the live file): `job "go" key "continue-on-error" ... is not on the reviewed whitelist gateJobAllowedKeys [runs-on steps]`. Also covered by `TestWorkflowShapeWhitelistRejectsUnenumeratedKeys/known_defect:_job-level_continue-on-error` (ci_wiring_gate_test.go:1628-1634). |
| **CR-02** (job/step/workflow `env: GOFLAGS`) | **CLOSED** (all 3 scopes) | `env` is absent from `workflowRootAllowedKeys` (ci_wiring_gate_test.go:1173), `gateJobAllowedKeys` (1182), and `gateStepAllowedKeys` (1207) — rejected by omission at each of the three scopes (rules at 1293-1297, 1402-1406, 1447-1452). Independently reproduced the step-level spelling in isolation: `step "Run Go tests" key "env" ... is not on the reviewed whitelist gateStepAllowedKeys [name run]`. 172-12's SUMMARY additionally reproduces all three spellings live against `.github/workflows/ci.yml` with mutate/restore transcripts. Doc comments at ci_wiring_gate_test.go:783-799 now state plainly that the whitelist, not the execution harness, is what closes this route. **However, see new finding CR-06**: this closes the *YAML-key* spelling of an env-var attack, not the *runtime env-file* spelling from an adjacent step — a materially different route to the same outcome. |
| **CR-03** (`paths-ignore` under `on:`) | **CLOSED** | `releaseTriggerAllowedKeys = []string{"branches"}` (ci_wiring_gate_test.go:1196) is the only key permitted on a trigger block (rule e, 1358-1362) — `paths-ignore`, `paths`, `types`, `branches-ignore` are all rejected by absence, not by name. Covered by `TestWorkflowShapeWhitelistRejectsUnenumeratedKeys/known_defect:_paths-ignore_under_pull_request` and by four further generality rows (`types`, `branches-ignore`, `paths`, `branches value is not exactly [main]`) that were never part of CR-03's own enumeration. |
| **CR-04** (last-word tolerance let a same-leaf newcomer through) | **CLOSED** | `TestPathMigrationDidNotWidenTolerance` (subcommand_reachability_ratchet_test.go:1589) was rewritten to compare full command paths via `pathMigrationToleratedPaths`/`pathMigrationWideningProblems` (1516-1570), driven by a frozen, reviewed `pathMigrationExpansion` map (1477-1484) plus two pinned counts (`preMigrationSnapshotEntryCount=278`, `pathMigrationExpandedPathCount=284`). Re-derived the arithmetic independently: 272 trivial 1:1 leaves + 12 paths from 6 non-trivial leaves (`get`→3, `set`→3, `registry`→2, `wisdom`→2, `archive`→1, `lifecycle`→1) = 284; 284 + 9 reviewed exemptions = 293, matching both JSON files exactly. `TestPathMigrationRejectsASameLeafNewcomer` (1634) proves this by construction: the review's own `aether newthing get` example, the verifier's live `aether colony-depth setup` reproduction, and a generality loop over all 278 frozen leaves at once — confirmed green by direct test run (`go test ./cmd -run TestPathMigration...` → PASS, `278 newcomer paths ... 278 reported as widening problems`). |
| **CR-05** (frozen anchor had no integrity pin) | **CLOSED** | `preMigrationSnapshotSHA256` (subcommand_reachability_ratchet_test.go:214) + `TestPreMigrationSnapshotIsFrozen` (220-234) hash the file at test time and fail on any mismatch. Independently re-hashed the file myself: `shasum -a 256 cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` → `873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e`, byte-identical to the pinned constant. Wired into the CI `-run` filter (ci.yml:100). |
| WR-01 (wiring step held to a weaker standard than the blanket step) | **STILL OPEN** | `extractWiringGateRunArg` (ci_wiring_gate_test.go:612-636) still only requires the run line to `strings.Contains(runLine, "go test ./cmd ")` and carry exactly one `-run` flag — a trailing `|| true`, `; true`, `\| cat`, or `&&` after the filter still satisfies it. `stepCanFailTheBuild`, now also applied to the wiring step (line 140), still checks only `if:`/`continue-on-error`, not shell control characters in the run line. `auditWorkflowShape`'s `gateStepAllowedKeys` closes the *key*-level version of this (an `env:` block) but does not inspect the *content* of the wiring step's `run:` value at all — so this specific gap (legibility-defeating shell suffixes on the wiring step) is unaddressed by either of this round's mechanisms. Out of scope for 172-12/172-13 per their own stated scope. |
| WR-02 (`-update-orphan-allowlist` appended to CI skips core assertions) | **STILL OPEN** | `if *updateOrphanAllowlist { ... return }` still sits inside `TestNoRegisteredSubcommandIsUnreferenced` (subcommand_reachability_ratchet_test.go:1164), before the D-08 `owner_phase` assertions. Unchanged by either plan. |
| WR-03 (`sh -c` executes an attacker-controllable string from ci.yml) | **STILL OPEN** | `runGateCommand` (ci_wiring_gate_test.go:801-828) still runs `exec.CommandContext(ctx, "sh", "-c", command)` with only a `strings.Contains(command, "go test")` sanity floor at the caller. Unchanged. |
| WR-04 (execution proof's comments overstated coverage) | **CLOSED** (comment-correction half; underlying scope unchanged) | The two flagged comments (blanketGateRunCommand's doc comment, and the sanity-floor comment in `TestReleaseGateCommandFailsATreeWithAFailingTest`) now explicitly state the pin *is* load-bearing for the narrowing-to-probe-only class, rather than disclaiming it as "NOT the proof" without qualification (ci_wiring_gate_test.go:60-70, 883-896). The new block comment at 689-704 names all three mechanisms and their exact non-overlapping scopes. Grepped confirmed: the overstated phrase "whatever the sixth mutation nobody has enumerated yet" no longer appears (0 occurrences). The underlying architectural fact WR-04 originally raised — the harness proves discrimination on a synthetic probe, not this repo's own tests — is unchanged, but it is no longer misrepresented, and the fix's own "correct the comments" option was explicit in the original recommendation. |
| WR-05 (`..`-relative path banned 270 lines earlier, used anyway) | **STILL OPEN** | `os.ReadFile("../.aether/docs/orphan-allowlist-policy.md")` still at cli_flag_audit_test.go:356, unchanged, in the same file whose lines 81-90 ban exactly this pattern. |
| WR-06 (policy doc arithmetic wrong; headline "can never be added" claim false) | **CLOSED** | `.aether/docs/orphan-allowlist-policy.md:142-153` now states the real 274(272+2 relocated)+10=284, 284+9=293 accounting, explicitly naming the leaf-splitting 172-09 did (contradicting the review's old "278 carried forward unchanged" claim, which is gone). The headline claim at lines 43-46 ("A name can never be added to either list ... fails immediately, by name") is now actually true for the orphan list given CR-04's fix: a same-leaf newcomer added to both live/baseline files no longer passes silently, since `TestPathMigrationDidNotWidenTolerance` compares full paths against the frozen anchor. |
| WR-07 (parent/grouping cobra nodes counted as permanently-unfixable orphans) | **STILL OPEN** | `enumerateRegisteredCommands` unchanged; `aether host` and similar menu nodes remain in the allowlist tagged `path-collision-revealed`/`RECLAIM` with no exclusion rule for pure-grouping nodes. |
| WR-08 (anti-vacuity floor tolerates a third of guard tests deleted) | **STILL OPEN, MARGIN UNCHANGED IN PROPORTION** | Floor is still `len(testNames) < 20` (ci_wiring_gate_test.go:171), now against a measured 34 (was 30) — up to 14 of 34 guard tests (41%) can be deleted without tripping it, versus 10 of 30 (33%) before. Not tightened by either plan. |
| WR-09 (prefix-shadowed `-run` alternatives can go stale undetected) | **STILL OPEN** | The stale-alternative check (ci_wiring_gate_test.go:198-220) still uses unanchored `regexp.MatchString`, unchanged. |
| WR-10 (escape-hatch scan: hand-maintained 5-file inventory, one spelling of "skip") | **STILL OPEN** | `wiringGateGuardFiles` (ci_wiring_gate_test.go:87-93) is still exactly 5 files; the receiver-name assumption and the missing transitive-helper coverage are unchanged. |
| IN-01 (stale "the four files" comment) | **STILL OPEN** | cli_flag_audit_test.go:38 still says "the four files"; `guardedAllowlistFiles` still holds 5. |
| IN-02 (redundant duplicated parameter in `check(t, path, listPath)`) | **STILL OPEN** | `check(t, "testdata/orphan_allowlist.json", "testdata/orphan_allowlist.json")` still calls with the same string twice (subcommand_reachability_ratchet_test.go:1455-1457). |
| IN-03 (dead alias-matching branch, unused field) | **STILL OPEN** | Not touched by either plan; unrelated code region. |
| IN-04 (hardcoded timeout literal in error message) | **STILL OPEN (technically accurate, still un-DRY)** | ci_wiring_gate_test.go:817 still hardcodes "120s" in the message rather than formatting `gateCommandTimeout`; the value still happens to match (120 * time.Second), so this remains a maintainability nit rather than a live bug. |
| IN-05 (`runLineOf` matches any key ending in `run:`) | **STILL OPEN** | `strings.Index(block, "run:")` unchanged at ci_wiring_gate_test.go:336. |
| IN-06 (commented-out following step absorbed into current step's block) | **STILL OPEN** | `blockEnd` (ci_wiring_gate_test.go:297-330) still treats a `# - name: ...` line as neither a same-indent marker nor a shallower indent, so it is absorbed into the preceding block. Unchanged. |
| IN-07 (documented-unreachable dead code, silent `continue`) | **STILL OPEN** | Unrelated code region, not touched. |
| IN-08 (CRLF-sensitivity, inconsistent first/last-wins in `TestReleaseGateWorkflowActuallyRuns`) | **STILL OPEN** | `strings.TrimRight(line, " ")` at ci_wiring_gate_test.go:1058 still does not strip `\r`; this test now runs alongside the new, structurally equivalent `auditWorkflowShape`, making it redundant coverage rather than the sole mechanism, but the bug itself (a confusing failure message on a CRLF checkout) is unchanged. |

---

## New Findings

### CR-06: The gate step's environment and toolchain are wide open to any of the other ~18 unaudited steps in the same job — a route the whitelist, the pin, and the execution harness were each built to inspect the wrong text for

**Files:** `.github/workflows/ci.yml` (all steps other than "Run Go tests" and
"Verify subcommand wiring and CLI flag contracts"); `cmd/ci_wiring_gate_test.go`
(`gateStepAllowedKeys`, `workflowRootAllowedKeys`, `gateJobAllowedKeys`,
`auditWorkflowShape`)

**Issue:** All three mechanisms this phase has built — the exact-equality pin
on the gate step's own run line, the execution harness that runs that exact
command against a probe module, and 172-12's `auditWorkflowShape` key
whitelist — inspect only the *text of the two named gate steps themselves*.
Nothing in this phase inspects, whitelists, or restricts the other ~18 steps
in the same `go:` job: Checkout, Setup Go, Setup Node, Install goreleaser,
Validate goreleaser config, Build, Vet, and everything after the gate steps.

GitHub Actions jobs run every step sequentially **on the same persistent
runner and filesystem**, and a step can propagate environment changes to
every later step in the job by writing to the file named in the
`$GITHUB_ENV` environment variable — this is standard, documented Actions
behaviour (`echo "GOFLAGS=-run=TestNothingZZZ" >> "$GITHUB_ENV"`), not an
exploit of a bug. A single line added to, say, the "Setup Node" step's
existing `run:` block:

```yaml
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: "22"
```
→
```yaml
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: "22"
      - name: Setup Node (env)
        run: echo "GOFLAGS=-run=TestNothingZZZ" >> "$GITHUB_ENV"
```

sets `GOFLAGS` for the "Run Go tests" step that runs afterward, exactly
reproducing CR-02's empirically-verified outcome (`go test ./... -run
=TestNothingZZZ` exits 0 regardless of test failures) — but:

- `auditWorkflowShape` does not see it: this new step is not one of the two
  named gate steps, so `gateStepAllowedKeys` never inspects it, and its
  `run:` value is free-form text, not a YAML `env:` key.
- `blanketGateRunCommand`'s exact-equality pin still matches: the "Run Go
  tests" step's own run line is untouched.
- The execution harness still discriminates correctly on the synthetic
  probe module, because it never sees the workflow's own environment at all
  (`cmd.Env` is left nil — this is documented and intentional per CR-02's
  fix, but it means this harness was never capable of catching this class
  either).

A more direct variant needs no environment variable at all: any earlier step
can overwrite the `go` binary itself (`sudo ln -sf /bin/true
"$(command -v go)"`, or write a fake `go` script earlier on `$PATH` via
`$GITHUB_PATH`), since every step in the job shares one filesystem. This
defeats `go test`, `go vet`, and `go build` identically and is invisible to
every guard in this phase for the same reason.

**This is exactly the failure mode CLAUDE.md's Definition of Done and this
phase's own STOP-RULE describe: a guard that checks the *shapes of
disabling it has enumerated* (a literal `env:` YAML key on the two named
steps) while leaving an adjacent, less conspicuous route (a `run:` script
on any of the eighteen other steps) to the identical runtime outcome
completely unchecked.** It is materially easier to hide than CR-01/02/03,
since it does not require adding a recognisable top-level key to a
short, two-key-wide whitelist — it only requires one line inside an
already-multi-line `run:` block on a step nobody is looking at.

**Fix:** Two complementary options, neither mutually exclusive:

1. **Shrink the audited surface to match the trusted surface.** Move the
   release-gate step to its own job (or a workflow with only Checkout +
   Setup Go + the gate step before it), so there is no earlier step left
   that could plant an env-file write or a toolchain override. This is the
   only fix that removes the gap rather than chasing its next spelling.

2. **If the steps must stay co-located, extend the audit to scan every
   `run:` block in the job — not just the two named gate steps — for the
   patterns that can propagate to a later step:**

```go
// forbiddenEnvPropagationRe matches writes to the two GitHub Actions
// environment-propagation files from ANY step's run: block, not just the
// two named gate steps — a step anywhere earlier in the same job can set
// GOFLAGS/GOTOOLCHAIN or prepend a fake `go` to PATH for every step after
// it, and neither the equality pin nor the execution harness can see this
// because it never touches the gate step's own text.
var forbiddenEnvPropagationRe = regexp.MustCompile(`>>\s*"?\$GITHUB_(ENV|PATH)"?`)

for _, stepNode := range stepsNode.Content {
	// ... extract this step's run: text (all steps, not just the two gate steps) ...
	if forbiddenEnvPropagationRe.MatchString(runText) {
		errs = append(errs, fmt.Errorf(
			"step %q writes to $GITHUB_ENV or $GITHUB_PATH — this changes what every LATER step in the same job sees, "+
				"including the release gate, and none of this phase's other guards can see it because it never touches the gate step's own text", stepName))
	}
}
```

Option 2 is a blocklist of the two known Actions propagation mechanisms and
will itself eventually need the same "adjacent sibling" scrutiny this phase
has repeatedly required of its other guards (e.g. a step could still `sudo`
overwrite the toolchain directly without touching either env file) — option
1 is the only fix that does not leave a residual class.

---

## Verdict

Two follow-up rounds of work closed all five of the serious problems flagged
last time — I checked the arithmetic myself, re-hashed the frozen file
myself, and reproduced two of the five bypasses in an isolated test to
confirm the fix actually works, not just that the transcript in the summary
says it does. That work is solid and the previous five most urgent findings
are genuinely fixed.

But this round's own new mechanism has the same blind spot as every round
before it: it only looks at the two specific steps in the CI checklist that
run the tests, and not at any of the roughly eighteen other steps that run
*before* them in the same automated setup. Those other steps — installing
Node, installing a release tool, building the app — are allowed to run
anything at all, and on GitHub's systems a step is allowed to leave a note
for every step after it (an official, documented feature, not a hack) or
even replace the testing tool on disk outright. A single line hidden inside
one of those other steps' existing setup instructions can quietly tell the
test run to skip everything and report success, and none of the three
safety nets built in this phase would notice, because none of them ever
look at those other steps at all. The most durable fix is to move the
actual test-running step into its own separate, minimal checklist with
nothing ahead of it that could interfere — everything else is a narrower
patch that this phase's own history shows tends to get worked around again.

---

_Reviewed: 2026-08-12T13:20:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
