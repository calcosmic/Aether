# Phase 205 release-gate test repair

Bounded repair for plan 205-12, task 3. The two checks now measure the intended
contracts without depending on another test having run or the checkout's name.
**The repaired full plain and race gates are still pending the parent executor.**
These focused results do not establish release readiness.

## Original full-run accounting

- Tested commit: `7508fc73d48370ef1a41ee42c7cd2dae988f558d`.
- Parent checkout: `/private/tmp/aether-phase205-execution/gate-checkout`.
- `go build ./cmd/aether` and `go vet ./...` exited 0.
- `go test ./... -count=1 -timeout 90m` exited 1 after 1,378 seconds.
  Its controller reported `FULL-SUITE FAIL discovered=5573 executed=5573 lanes=59`:
  complete execution, **not a green run**.
- There were 19 distinct top-level failures: all 17 names in the unchanged
  `/tmp/aether-phase205-execution/baseline.json`, plus
  `TestDocumentedCommandNamesResolve` and `TestResolveAetherRoot_GitFallback`.
  `TestGateProbe` in the results' raw name list is deliberate nested fixture
  output; its enclosing `TestGateProbeCatchesEveryKnownGateNeutering` passed.
  It is neither an eighteenth baseline failure nor another repair target.
- The parent interrupted the superseded race run. It supplies no pass/fail
  conclusion; see the preserved interruption receipt below.

Evidence directory for this repair: `/tmp/aether-phase205-execution/gate-repair`
(`/private/tmp/...` is the same location on this Mac). Original artifacts were
copied here before the parent rerun can replace them:

| Preserved artifact | Original path relative to `/tmp/aether-phase205-execution/` | SHA-256 |
| --- | --- | --- |
| `original-full.log` | `gates/full.log` | `adcb5ba3b7d3850f61f9566dab015e45de8f86e7c2039c1c3f4a8338dcf980ee` |
| `original-results.json` | `gates/results.json` | `48711db7ec7efc08800540115cb86dea4ec5b8f8bee2a5bf217c629db41e6b8b` |
| `baseline.json` | `baseline.json` | `5ce2d98b291f41586df99f841abe316da169de65b7d2be008023e0c6be071f8d` |
| `original-race-interrupted.json` | `gates/race-interrupted.json` | `29bac83ad29c4ae7437fe0ac7ece85cc7247264da77d2e7d391cbae9fb74df44` |

## Causes and changes

1. `cmd/command_call_audit_test.go`: the name audit calls `rootCmd.Find` before
   Cobra lazily registers its built-in `help`. Explicitly call
   `rootCmd.InitDefaultHelpCmd()`, matching `TestCLIFlagAudit` and
   `TestCLIFlagAuditSubcommandsRegistered`. The three documented help calls
   remain audited, and the unresolved-command checks remain intact.
2. `pkg/storage/paths_test.go`: replace the checkout-dependent capitalized
   `Aether` assertion with a disposable `root-resolver-*` Git repository. Enter
   `nested/child` and require the exact repository root. `os.MkdirTemp` avoids
   embedding the test's own `Aether` name, `filepath.EvalSymlinks` handles macOS
   physical paths, setup errors fail the test, and cleanup restores the working
   directory before removing the fixture. Environment-precedence tests remain.
3. This evidence note is the only other changed file.

These are verifier bug fixes under GSD deviation Rule 1, explicitly assigned
within task 3. Production registration and path-resolution code are unchanged;
no baseline exception was added. No STATE, ROADMAP, REQUIREMENTS, numbered plan
SUMMARY, owner project, hub, main checkout, or other worktree was changed.
No full suite or publication was run by this repair executor.

## Before/after evidence

Repair worktree:
`/Users/callumcowie/repos/Aether/.claude/worktrees/agent-p205-gatefix-codex-20260915`,
branch `worktree-agent-p205-gatefix-codex-20260915`, starting at
`dbc8c55d7129317a3cb56e188a6ea673fb7ae28f`. The two test files were identical to
the original tested revision before editing.

Reproduction used a separate clone made from this assigned worktree, detached
at `7508fc73d48370ef1a41ee42c7cd2dae988f558d`, under the evidence directory's
`baseline-checkout/`. It remained clean and unmodified before and after testing.
The active parent gate checkout was not used. Clone commands, from this worktree:

```sh
git clone --shared --no-checkout . /tmp/aether-phase205-execution/gate-repair/baseline-checkout
git -C /tmp/aether-phase205-execution/gate-repair/baseline-checkout checkout --detach 7508fc73d48370ef1a41ee42c7cd2dae988f558d
```

The same command failed both targets in that baseline clone, then passed both
in the repair worktree. Running the name audit alone in its package ensures
the sibling flag audits cannot initialize help on its behalf:

```sh
go test ./cmd ./pkg/storage -run '^(TestDocumentedCommandNamesResolve|TestResolveAetherRoot_GitFallback)$' -count=1 -timeout 90m -v
```

Remaining checks ran in the repair worktree, uncached with the same timeout.
The audit pattern covers all 13 tests in the two command audit files:

```sh
GATE_REPAIR_AUDITS='^(TestCommandCallsMatchCobraContracts|TestAuditDetectsPositionalDrift|TestAetherCorpusCatchesAnUnregisteredFlag|TestCommandCallExtractorSeesRealInvocationsAndSkipsProse|TestExtractorDoesNotDesyncOnGluedFenceMarker|TestDocumentedCommandNamesResolve|TestDocumentedSubcommandsAreSeverityClassified|TestGateClassifiedCallsHaveGateWiring|TestAuditedCorpusHasNoGluedFenceMarkers|TestCLIFlagAudit|TestCLIFlagAuditSubcommandsRegistered|TestFlagAuditSkipListOnlyShrinks|TestAllowlistPolicyNamesEveryGuardedFile)$'
go test ./cmd -run "$GATE_REPAIR_AUDITS" -count=1 -timeout 90m -v
go test ./cmd -run "$GATE_REPAIR_AUDITS" -race -count=1 -timeout 90m -v
go test ./pkg/storage -count=1 -timeout 90m -v
go test ./pkg/storage -race -count=1 -timeout 90m -v
go test -overlay /tmp/aether-phase205-execution/gate-repair/cwd-only-overlay.json ./pkg/storage -run '^TestResolveAetherRoot_GitFallback$' -count=1 -timeout 90m -v
```

The final command is an expected-failure check. The Go overlay substitutes a
temporary copy of `paths.go` that returns the current directory after honoring
`AETHER_ROOT`; it does not edit production source. The repaired test rejects
`.../root-resolver-27156091/nested/child` instead of the required repository root.
The overlay and replacement source are retained with the evidence.

Each log below has a same-stem `.json` recording its exact expanded command,
working directory, revision, source hashes, exit code, duration, and clean/dirty
status. All paths are relative to the evidence directory above.

| Log | Outcome | SHA-256 |
| --- | --- | --- |
| `before.log` | Exit 1; both assigned failures reproduced | `3f6230702c4a982b272063b58f7c9e1bfc9df55b3194d68b7b037da7fefedcde` |
| `after-targets.log` | Exit 0; 2/2 targets pass | `ff5f89fa47c08bd18f0e24c2f5c961c960e2157f357f504ec354c6fde5b98b36` |
| `audits-normal.log` | Exit 0; 13/13 audits pass | `8fbeb0c1a28309658984ef116575ae7526c2c7f92cee7ac59467cadd5dec0705` |
| `audits-race.log` | Exit 0; 13/13 audits pass | `4e0dd25489a317c9d37775c4be9ea315d531b7a51cd71fcfd2f3ab61d3f093ae` |
| `storage-normal.log` | Exit 0; 47/47 top-level tests pass | `b5bcb88a266224a0adb4c15f09a5398737bcd6bca35c6463b1cfe937b6ac9f37` |
| `storage-race.log` | Exit 0; 47/47 top-level tests pass | `bb60d2b8eed76632499e05316800c59cfb51fc2ac08c9e02fbf3aff0c96d3402` |
| `cwd-only-negative.log` | Expected exit 1; nested-directory shortcut rejected | `d4facffe8dfc52d890ad452ebf390ac5717ed421552c0e307b9025ba7c0aeafc` |

Parent handoff: review/merge this repair, then rerun the complete plain and race
gates against the merged revision, preserving discovered/executed accounting
and comparing failures with the same 17-name baseline.
