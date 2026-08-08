---
phase: 172
slug: wiring-proof
status: planned
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-08
---

# Phase 172 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `172-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` package (no third-party test framework in `cmd/`) |
| **Config file** | none — Go tests need no config; CI invocation lives in `.github/workflows/ci.yml` ("Run Go tests" step) |
| **Quick run command** | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestOrphanAllowlist\|TestCommandCallsMatchCobraContracts\|TestSpawnCanSpawn' -v` |
| **Full suite command** | `go test ./... -count=1 -timeout 900s` |
| **Estimated runtime** | ~15s quick · ~300–600s full suite |

---

## Sampling Rate

- **After every task commit:** Run the quick run command above
- **After every plan wave:** `go test ./cmd/... -count=1 -timeout 900s -v` — the whole `cmd`
  package, because these tests mutate/read the shared `rootCmd` global and ordering side
  effects must surface early
- **Before `/gsd-verify-work`:** Full suite green AND the new named CI step(s) from D-15 green
- **Max feedback latency:** 15 seconds for the quick command

---

## Per-Task Verification Map

Task IDs are `{plan}.{task}` — e.g. `01.2` is plan `172-01-PLAN.md`, task 2.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01.1 | 172-01 | 1 | WIRE-02 (D-13, D-14) | T-172-01, T-172-02, T-172-03 | `aether spawn-can-spawn 5 --enforce` executes; an unparseable positional depth fails loudly instead of defaulting to 0; the deny path exits non-zero via `outputError`, never `os.Exit` | integration (CLI) | `go build ./cmd/aether && go run ./cmd/aether spawn-can-spawn 5 --enforce` | ❌ W0 — `cmd/spawn.go` | ⬜ pending |
| 01.2 | 172-01 | 1 | WIRE-02 (D-13) | T-172-01, T-172-03 | The documented invocation is read from `.aether/workers.md` at test time, not copied; `--enforce` deny exits non-zero and its absence does not gate | unit + integration | `go test ./cmd -run TestSpawnCanSpawn -count=1 -v` | ❌ W0 — `cmd/spawn_enforce_test.go` | ⬜ pending |
| 02.1 | 172-02 | 1 | WIRE-01 (D-01..D-05, D-07, D-08, D-09) | T-172-07, T-172-08, T-172-09 | Only the three permitted caller kinds count; token-boundary matching so `skill-list-lifecycle` cannot clear `skill-list`; the allowlist is seeded from real scanner output with queryable reason tags; the ratchet never reads the regenerated catalog | unit (AST + corpus static analysis) | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestRatchetDoesNotConsultTheRegeneratedCatalog' -count=1 -v` | ❌ W0 — `cmd/subcommand_reachability_ratchet_test.go`, `cmd/testdata/orphan_allowlist{,_baseline}.json` | ⬜ pending |
| 02.2 | 172-02 | 1 | WIRE-01 (D-10, D-11) | T-172-05, T-172-06, T-172-08 | The allowlist may only shrink by set membership (a one-out-one-in swap fails); a synthetic orphan is detected and named; suppressing a command's only caller makes the ratchet name it; no guard can be switched off at runtime | unit (baseline diff + fixture + suppression) | `go test ./cmd -run 'TestOrphanAllowlistOnlyShrinks\|TestRatchetDetectsASyntheticOrphan\|TestDeletingACallerMakesTheRatchetNameIt\|TestWiringGuardsHaveNoRuntimeEscapeHatch' -count=1 -v` | ❌ W0 — same new file | ⬜ pending |
| 03.1 | 172-03 | 2 | WIRE-03 | T-172-11 | Command-substitution invocations (`x=$(aether …)`) are audited; shell redirections terminate an invocation instead of posing as positional arguments | unit (extractor) | `go test ./cmd -run 'TestCommandCallExtractorSeesRealInvocationsAndSkipsProse\|TestAuditDetectsPositionalDrift' -count=1 -v` | ✅ extends `cmd/command_call_audit_test.go` | ⬜ pending |
| 03.2 | 172-03 | 2 | WIRE-03 (D-06, D-14) | T-172-10, T-172-12 | `.aether/workers.md` is proven to be read, not merely listed; every invocation in it matches the binary's real contract; line 292 is untouched; a fixture proves the corpus catches an unregistered flag forever | unit (corpus + fixture) | `go test ./cmd -run 'TestCommandCallsMatchCobraContracts\|TestAetherCorpusCatchesAnUnregisteredFlag' -count=1 -v` | ✅ extends `cmd/command_call_audit_test.go`; ❌ W0 — `.aether/workers.md` drift fixes | ⬜ pending |
| 04.1 | 172-04 | 2 | WIRE-01 (D-12), WIRE-03 | T-172-14 | The flag audit's `skipSubcommands` map is shrink-only against a committed baseline, so exemption pressure cannot relocate to an unguarded list | unit (baseline diff) | `go test ./cmd -run 'TestCLIFlagAudit\|TestFlagAuditSkipListOnlyShrinks' -count=1 -v` | ❌ W0 — `cmd/testdata/flag_audit_skiplist_baseline.json` | ⬜ pending |
| 04.2 | 172-04 | 2 | WIRE-01 | T-172-15 | The written policy names every guarded file, derived from the Go constants rather than from prose, and states the real seeded counts | unit (doc/code invariant) | `go test ./cmd -run TestAllowlistPolicyNamesEveryGuardedFile -count=1 -v` | ❌ W0 — `.aether/docs/orphan-allowlist-policy.md` | ⬜ pending |
| 05.1 | 172-05 | 3 | WIRE-01, WIRE-02, WIRE-03 (D-15) | T-172-17, T-172-18 | The named CI step's `-run` filter is asserted to match every guard test that exists, and the blanket `go test ./...` step is still present | unit (workflow/code invariant) | `go test ./cmd -run TestWiringGateStepRunsEveryWiringTest -count=1 -v` | ❌ W0 — `.github/workflows/ci.yml`, `cmd/ci_wiring_gate_test.go` | ⬜ pending |
| 05.2 | 172-05 | 3 | WIRE-01 (criterion 4) | T-172-19 | Deleting a caller and running the verbatim CI step command turns it red naming the command; the tree is restored; the phase records state the real seeded counts | scripted behavioural proof (transcript) + records check | `go test ./... -count=1 -timeout 900s` plus the ROADMAP/VALIDATION consistency check in `172-05-PLAN.md` task 2 | ❌ W0 — `.planning/ROADMAP.md`, this file | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/subcommand_reachability_ratchet_test.go` — new file carrying
      `TestNoRegisteredSubcommandIsUnreferenced`, the shrink-only allowlist assertion, and a
      fixture self-test proving the ratchet detects a synthetic orphan. Mirror the existing
      function-local `AddCommand` / `defer RemoveCommand` pattern from
      `TestAuditDetectsPositionalDrift` so the fixture cannot permanently trip the ratchet it tests.
- [ ] `cmd/testdata/orphan_allowlist.json` — committed baseline, **seeded from the scanner's
      real output**, not from the assumed count (D-07). Research found the true orphan count is
      likely 6, not 8: `skill-parse-frontmatter` and `skill-cache-rebuild` are genuinely invoked
      from `.claude/commands/ant/skill-create.md:220-242`.
- [ ] Extend `cmd/cli_flag_audit_test.go` — shrink-only guard over the currently-unguarded
      2-entry `skipSubcommands` map (D-12).
- [ ] `cmd/spawn.go` — register `--enforce`, relax `Args` from `cobra.NoArgs` to accept the
      documented positional depth (D-14), wire deny → non-zero exit using the existing
      `outputError` deferred-exit-code convention already used by sibling `spawnLogCmd`.
- [ ] `cmd/command_call_audit_test.go` — extend `auditedCorpora` to bring `.aether` markdown
      into scope (see Open Scope Call below).
- [ ] `.github/workflows/ci.yml` — named step(s) per D-15, near the existing
      "Verify command catalog classification" step.
- [ ] Framework install: **none** — `go test` is already the toolchain.

---

## Resolved Scope Call

Resolved by the user before planning. The flag audit's `.aether` markdown corpus is:
the four git-tracked **top-level** `.aether/*.md` files (`CONTEXT.md`, `CROWNED-ANTHILL.md`,
`QUEEN.md`, `workers.md` — `HANDOFF.md` is gitignored and excluded), **plus**
`.aether/docs/command-playbooks/*.md`, which D-06 locks into scope and which
`auditedCorpora` already covers. It is **not** recursive over `.aether/**/*.md` (~250 files);
the recursive scope and the staged-allowlist variant were both explicitly rejected. No
exceptions list is added for this corpus — the chosen scope is expected to be clean once
WIRE-02 lands.

Planning also found a second blocker the research did not: the extractor cannot see
command-substitution invocations at all, so `result=$(aether spawn-can-spawn {your_depth}
--enforce)` is skipped even with the corpus in scope. Plan `172-03` fixes that first; without
it, extending the corpus produces a test that passes while the bug sits inside its scope.

---

## Manual-Only Verifications

**None.** The one entry previously listed here — "the release gate actually goes red when a
caller disappears" — has been split into two automated pieces:

| Half of criterion 4 | How it is now covered | Where |
|---------------------|------------------------|-------|
| The ratchet detects a vanished caller and names the command | `TestDeletingACallerMakesTheRatchetNameIt` recomputes caller evidence with one corpus file suppressed and asserts the command flips to orphan. Runs on every CI run, hermetically, and never leaves the branch red. | task 02.2 |
| The gate that runs in CI actually goes red | Task 05.2 deletes a real caller, runs the named CI step's command **verbatim**, records the red output naming the command, restores, and records the green run. The transcript is the evidence; `git status --porcelain` proves the tree was restored. | task 05.2 |

The plan for 05.2 is a scripted verification an executor performs and reverts, not a
checkpoint requiring the operator. Nothing in this phase asks the user to run a command.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s for the quick command
- [ ] The allowlist baseline was seeded from real scanner output, not the assumed count
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** approved by planner 2026-08-08 — every task carries an `<automated>` verify; no three consecutive tasks lack one; Wave 0 gaps are each owned by a named task.
