---
phase: 172
slug: wiring-proof
status: gap_closure_planned
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-08
updated: 2026-08-11
---

> **Reopened 2026-08-11 by gap-closure planning.** `172-VERIFICATION.md` recorded two
> confirmed, independently reproduced gaps against ROADMAP success criteria 3 and 4. Waves 5
> and 6 (plans `172-06`, `172-07`, `172-08`) exist to close them. Waves 1–4 remain green; the
> rows for waves 5–6 below are ⬜ pending until those plans execute. The "Closed out" note at
> the foot of this file describes the wave-4 close only and no longer means the phase is done.

# Phase 172 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `172-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` package (no third-party test framework in `cmd/`) |
| **Config file** | none — Go tests need no config; CI invocation lives in `.github/workflows/ci.yml` ("Run Go tests" step) |
| **Quick run command** | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestOrphanAllowlist\|TestCallerEvidence\|TestCommandCall\|TestSpawnCanSpawn' -v` |
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

## Wave Structure

| Wave | Plans | Why this order |
|------|-------|----------------|
| 1 | `172-00` (extractor fix), `172-01` (spawn-can-spawn fix) | Nothing downstream may read the world through a blind extractor. `172-00` teaches `extractDocumentedCalls` to see `x=$(aether …)` invocations **before** `172-02` seeds a list that can never grow. `172-01` is independent. |
| 2 | `172-02` (orphan ratchet + seeded allowlist), `172-03` (`.aether` corpus + drift fixes) | Both depend on `172-00`. `172-03` additionally depends on `172-01`, so the branch is never knowingly red. No file overlap between them. |
| 3 | `172-04` (skip-list guard + policy doc) | Reuses `172-02`'s comparator and states its real seeded counts. |
| 4 | `172-05` (named CI step + criterion-4 proof + records) | Enumerates every guard test that exists, so it must run last. |
| 5 | `172-06` (fence-parity blind spot: the two live violations it hid, the tolerant toggle, the corpus sweep) | Gap 1. Must precede waves 6 because both later plans read the world through `extractDocumentedCalls`, and the audited-invocation count is only stable (988) once the blind region is visible. Content is fixed *before* the extractor gains sight, so the tree is green at the end of every task. |
| 6 | `172-07` (step-scoped release-gate check + shared guard-file inventory), `172-08` (symmetric substitution opener/closer + flag-audit anti-vacuity floor) | Gap 2 and the reinforcing hardening. Both depend on `172-06`. Disjoint `files_modified` — `172-07` owns `cmd/ci_wiring_gate_test.go` + `cmd/subcommand_reachability_ratchet_test.go` + `.github/workflows/ci.yml`; `172-08` owns `cmd/command_call_audit_test.go` + `cmd/cli_flag_audit_test.go`. |

---

## Per-Task Verification Map

Task IDs are `{plan}.{task}` — e.g. `01.2` is plan `172-01-PLAN.md`, task 2.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 00.1 | 172-00 | 1 | WIRE-03; WIRE-01 (precondition) | T-172-11, T-172-21, T-172-23 | Command-substitution invocations (`x=$(aether …)`) become visible to the one shared extractor; shell redirections terminate an invocation instead of posing as positional arguments; both binary-detection call sites share one normaliser so they cannot drift apart again | unit (extractor) | `go test ./cmd -run 'TestCommandCallsMatchCobraContracts\|TestAuditDetectsPositionalDrift\|TestDocumented\|TestGateClassified' -count=1 -v` | ✅ extends `cmd/command_call_audit_test.go` | ✅ green |
| 00.2 | 172-00 | 1 | WIRE-03 | T-172-22 | Every new extraction behaviour is pinned by a unit case, including the verbatim `result=$(aether spawn-can-spawn 5 --enforce)` shape and a negative nested-pipeline case, so the widened detection cannot rot into vacuous success | unit (extractor self-test) | `go test ./cmd -run 'TestCommandCallExtractorSeesRealInvocationsAndSkipsProse\|TestCommandCallsMatchCobraContracts' -count=1 -v` | ✅ same file | ✅ green |
| 01.1 | 172-01 | 1 | WIRE-02 (D-13, D-14) | T-172-01, T-172-02, T-172-03 | `aether spawn-can-spawn 5 --enforce` executes; an unparseable positional depth fails loudly instead of defaulting to 0; the deny path exits non-zero via `outputError`, never `os.Exit` | integration (CLI) | `go build ./cmd/aether && go run ./cmd/aether spawn-can-spawn 5 --enforce` | ❌ W0 — `cmd/spawn.go` | ✅ green |
| 01.2 | 172-01 | 1 | WIRE-02 (D-13) | T-172-01, T-172-03 | The documented invocation is read from `.aether/workers.md` at test time, not copied; `--enforce` deny exits non-zero and its absence does not gate | unit + integration | `go test ./cmd -run TestSpawnCanSpawn -count=1 -v` | ❌ W0 — `cmd/spawn_enforce_test.go` | ✅ green |
| 02.1 | 172-02 | 2 | WIRE-01 (D-01..D-05, D-07, D-08, D-09) | T-172-07, T-172-08, T-172-09, T-172-24 | Only the three permitted caller kinds count; token-boundary matching so `skill-list-lifecycle` cannot clear `skill-list`; the seeding scan is proven not blind to a `$(aether …)` caller; the allowlist is seeded from real scanner output with queryable reason tags; the ratchet never reads the regenerated catalog | unit (AST + corpus static analysis) | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestCallerEvidenceCreditsCommandSubstitution\|TestRatchetDoesNotConsultTheRegeneratedCatalog' -count=1 -v` | ❌ W0 — `cmd/subcommand_reachability_ratchet_test.go`, `cmd/testdata/orphan_allowlist{,_baseline}.json` | ✅ green |
| 02.2 | 172-02 | 2 | WIRE-01 (D-10, D-11) | T-172-05, T-172-06, T-172-08 | The allowlist may only shrink by set membership (a one-out-one-in swap fails); a synthetic orphan is detected and named; suppressing a command's only caller makes the ratchet name it; no guard can be switched off at runtime | unit (baseline diff + fixture + suppression) | `go test ./cmd -run 'TestOrphanAllowlistOnlyShrinks\|TestRatchetDetectsASyntheticOrphan\|TestDeletingACallerMakesTheRatchetNameIt\|TestWiringGuardsHaveNoRuntimeEscapeHatch' -count=1 -v` | ❌ W0 — same new file | ✅ green |
| 03.1 | 172-03 | 2 | WIRE-03 (D-06, D-14) | T-172-10, T-172-12, T-172-25 | `.aether/workers.md` is proven to be read, not merely listed, with line 292 among the validated invocations; every invocation in it matches the binary's real contract; line 292 is untouched; wave-1 extractor code is not re-opened | unit (corpus) + integration (CLI) | `go test ./cmd -run 'TestCommandCallsMatchCobraContracts\|TestDocumentedCommandNamesResolve' -count=1 -v && go run ./cmd/aether swarm-display-update --agent X --id Y --status excavating` | ✅ extends `cmd/command_call_audit_test.go`; ❌ W0 — `.aether/workers.md` drift fixes | ✅ green |
| 03.2 | 172-03 | 2 | WIRE-03 | T-172-10 | A function-local fixture reproduces the pre-fix `spawn-can-spawn` shape and proves the corpus + extractor + validator chain catches `--enforce` on every CI run, never at package scope so it cannot become a real orphan | unit (fixture) | `go test ./cmd -run 'TestAetherCorpusCatchesAnUnregisteredFlag\|TestAuditDetectsPositionalDrift' -count=1 -v` | ❌ W0 — `cmd/command_call_audit_test.go` | ✅ green |
| 04.1 | 172-04 | 3 | WIRE-01 (D-12), WIRE-03 | T-172-14 | The flag audit's `skipSubcommands` map is shrink-only against a committed baseline, so exemption pressure cannot relocate to an unguarded list | unit (baseline diff) | `go test ./cmd -run 'TestCLIFlagAudit\|TestFlagAuditSkipListOnlyShrinks' -count=1 -v` | ❌ W0 — `cmd/testdata/flag_audit_skiplist_baseline.json` | ✅ green |
| 04.2 | 172-04 | 3 | WIRE-01 | T-172-15 | The written policy names every guarded file, derived from the Go constants rather than from prose, and states the real seeded counts | unit (doc/code invariant) | `go test ./cmd -run TestAllowlistPolicyNamesEveryGuardedFile -count=1 -v` | ❌ W0 — `.aether/docs/orphan-allowlist-policy.md` | ✅ green |
| 05.1 | 172-05 | 4 | WIRE-01, WIRE-02, WIRE-03 (D-15) | T-172-17, T-172-18 | The named CI step's `-run` filter is asserted to match every guard test that exists, and the blanket `go test ./...` step is still present | unit (workflow/code invariant) | `go test ./cmd -run TestWiringGateStepRunsEveryWiringTest -count=1 -v` | ❌ W0 — `.github/workflows/ci.yml`, `cmd/ci_wiring_gate_test.go` | ✅ green |
| 05.2 | 172-05 | 4 | WIRE-01 (criterion 4) | T-172-19 | Deleting a caller and running the verbatim CI step command turns it red naming the command; the tree is restored; the phase records state the real seeded counts | scripted behavioural proof (transcript) + records check | `go test ./... -count=1 -timeout 900s` plus the ROADMAP/VALIDATION consistency check in `172-05-PLAN.md` task 2 | ❌ W0 — `.planning/ROADMAP.md`, this file | ✅ green |

### Gap-closure rows (waves 5–6)

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06.1 | 172-06 | 5 | WIRE-03 (gap 1) | T-172-26, T-172-27 | The two live violations hidden inside the desynced fence region are repaired at source (`midden-recent-failures --limit 50`; `generate-commit-message` flag form copied from its already-correct split sibling) and two commands gain their D-01 severity classification — done *before* the extractor can see them, so the audited count is provably unchanged (954) and the fix is not self-masking | unit (corpus) | `go test ./cmd -run 'TestCommandCallsMatchCobraContracts\|TestDocumentedSubcommandsAreSeverityClassified\|TestDocumentedCommandNamesResolve\|TestGateClassifiedCallsHaveGateWiring\|TestCLIFlagAudit' -count=1 -v` (asserts log reads `audited 954 …`) | ✅ extends `cmd/command_call_audit_test.go`, 7 playbook files | ⬜ pending |
| 06.2 | 172-06 | 5 | WIRE-03 (gap 1) | T-172-28, T-172-29 | A closing fence marker glued to trailing content no longer desyncs in-fence parity, so no invocation after it is invisible; pinned by `TestExtractorDoesNotDesyncOnGluedFenceMarker`, whose red proof must name `midden-recent-failures` under the old toggle **and** name `backup-prune-global` under the tempting toggle-and-skip shortcut, so one blind spot cannot be traded for another | unit (extractor, two red proofs) | `go test ./cmd -run 'TestExtractorDoesNotDesyncOnGluedFenceMarker\|TestCommandCallsMatchCobraContracts' -count=1 -v` (asserts log reads `audited 988 …`) | ✅ same file | ⬜ pending |
| 06.3 | 172-06 | 5 | WIRE-03 (gap 1) | T-172-30 | A single command fails when any file in the audited corpus gains a fence marker glued to trailing content, naming file and line — the defect class cannot return unnoticed, and the equivalence invariant (988 with the tolerant toggle *and* with repaired markdown) is asserted rather than assumed | unit (corpus sweep) + phase gate | `go test ./cmd -run 'TestAuditedCorpusHasNoGluedFenceMarkers\|TestCommandCallsMatchCobraContracts\|TestWiringGateStepRunsEveryWiringTest' -count=1 -v && go test ./... -count=1 -timeout 900s` | ✅ same file | ⬜ pending |
| 07.1 | 172-07 | 6 | WIRE-01, WIRE-02, WIRE-03 (gap 2, criterion 4) | T-172-18 (re-opened), T-172-31, T-172-32, T-172-33 | The blanket release-gate check is scoped to the step whose `run:` line is exactly `go test ./... -count=1 -timeout 900s`, resolved by an end-of-line-anchored `- name:` lookup so `Run Go tests` cannot silently resolve to `Run Go tests with race detection`, and the step is verified *capable of failing* (no `if: always()`, no `\|\| true` / `\|\| echo` fallback) — a decoy substring can no longer satisfy it | unit (workflow invariant) + hermetic fixture table | `go test ./cmd -run 'TestWiringGateStepRunsEveryWiringTest\|TestBlanketGateCheckRejectsADecoyStep' -count=1 -v` | ✅ extends `cmd/ci_wiring_gate_test.go` | ⬜ pending |
| 07.2 | 172-07 | 6 | WIRE-01 (gap 2, hardening) | T-172-34, T-172-35 | All five guard files this phase created are scanned for a runtime escape hatch from one shared `wiringGateGuardFiles` inventory (previously 3 of 5), with `os.LookupEnv`, `os.Environ`, `syscall.Getenv` and `testing.Short` added — and an anti-self-trip proof, because exempting the scanner from its own scan trades a false red for a real hole | unit (AST/source scan) + phase gate | `go test ./cmd -run 'TestWiringGuardsHaveNoRuntimeEscapeHatch\|TestWiringGateStepRunsEveryWiringTest' -count=1 -v && go test ./... -count=1 -timeout 900s` | ✅ extends `cmd/subcommand_reachability_ratchet_test.go` | ⬜ pending |
| 08.1 | 172-08 | 6 | WIRE-01, WIRE-03 (hardening) | T-172-36, T-172-37 | Substitution opener and closer become one symmetric `substitutionOpener` decision shared by both extraction call sites, so a bare `(aether …)` subshell is no longer silently dropped and a backtick form no longer carries a stray delimiter into a flag name; fixture cases are written and red *before* the implementation change | unit (extractor fixtures) | `go test ./cmd -run 'TestCommandCallExtractorSeesRealInvocationsAndSkipsProse\|TestCommandCallsMatchCobraContracts' -count=1 -v` (988 unchanged from 06.2) | ✅ extends `cmd/command_call_audit_test.go` | ⬜ pending |
| 08.2 | 172-08 | 6 | WIRE-01, WIRE-03 (hardening) | T-172-38, T-172-39, T-172-40 | The flag audit fails loudly instead of passing when a declared corpus directory cannot be read (today it silently `continue`s), enforces a scanned-file floor, and names failures by repo-relative path rather than a basename two corpora share — criterion 3's "naming the file, the line, and the offending flag" demonstrated, not asserted | unit (anti-vacuity + naming proof) + phase gate | `go test ./cmd -run 'TestCLIFlagAudit\|TestCLIFlagAuditSubcommandsRegistered\|TestFlagAuditSkipListOnlyShrinks\|TestAllowlistPolicyNamesEveryGuardedFile' -count=1 -v && go test ./... -count=1 -timeout 900s` | ✅ extends `cmd/cli_flag_audit_test.go` | ⬜ pending |

*Legend: ⬜ = pending · ✅ = green · ❌ = red · ⚠️ = flaky. Rows 00.1–05.2 are ✅ green as of 2026-08-11 (plan 172-05, wave-4 close). Rows 06.1–08.2 are ⬜ pending gap-closure execution.*

**Latency note (tasks 05.2, 06.3, 07.2, 08.2):** each of these rows ends its automated verify
chain with the full release-gate suite (`go test ./... -count=1 -timeout 900s`, ~300–600s),
well above the 15s guidance for per-task feedback. That is deliberate and matches the "Phase
gate" tier in the Sampling Rate section above, not the per-task tier: each is the closing task
of its plan, and every one of them changes a guard that the release gate itself runs — a
narrower command could not detect a guard that passes in isolation but breaks a sibling. Each
task's earlier, fast assertions (the `go test ./cmd -run …` alternations, ~10–15s) provide the
per-task feedback; the blanket suite is the gate, not the loop. Every other row completes
inside the quick-run budget.

---

## Wave 0 Requirements

- [ ] `cmd/command_call_audit_test.go` — **wave 1, plan 172-00**: teach `extractDocumentedCalls`
      and `parseFencedInvocation` to see command-substitution invocations via one shared
      `normalizeShellToken` helper, strip the closing `)`, and widen `isShellOperator` to
      attached redirections such as `2>/dev/null`. This must land **before** the allowlist is
      seeded — see the note under Resolved Scope Call.
- [ ] `cmd/subcommand_reachability_ratchet_test.go` — new file carrying
      `TestNoRegisteredSubcommandIsUnreferenced`, the shrink-only allowlist assertion,
      `TestCallerEvidenceCreditsCommandSubstitution`, and a fixture self-test proving the
      ratchet detects a synthetic orphan. Mirror the existing function-local `AddCommand` /
      `defer RemoveCommand` pattern from `TestAuditDetectsPositionalDrift` so the fixture
      cannot permanently trip the ratchet it tests.
- [ ] `cmd/testdata/orphan_allowlist.json` — committed baseline, **seeded from the scanner's
      real output**, not from the assumed count (D-07). Research found the true `skill-*` count
      is likely 6, not 8: `skill-parse-frontmatter` and `skill-cache-rebuild` are genuinely
      invoked from `.claude/commands/ant/skill-create.md:220-242`.
- [ ] Extend `cmd/cli_flag_audit_test.go` — shrink-only guard over the currently-unguarded
      2-entry `skipSubcommands` map (D-12).
- [ ] `cmd/spawn.go` — register `--enforce`, relax `Args` from `cobra.NoArgs` to accept the
      documented positional depth (D-14), wire deny → non-zero exit using the existing
      `outputError` deferred-exit-code convention already used by sibling `spawnLogCmd`.
- [ ] `cmd/command_call_audit_test.go` — extend the corpus to bring the top-level
      `.aether/*.md` files into scope (see Resolved Scope Call below).
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
--enforce)` is skipped even with the corpus in scope. Plan **`172-00` fixes that in wave 1**,
ahead of everything else in the phase. Two things depend on that ordering:

1. Extending the corpus without it (plan `172-03`) would produce a test that passes while the
   bug it exists to catch sits inside its declared scope.
2. Seeding the orphan allowlist without it (plan `172-02`) would let a command whose only
   caller is written `result=$(aether …)` be recorded as an orphan. Because the allowlist may
   only ever shrink, that wrong entry would become a permanent exemption rather than
   self-correcting through the ratchet.

Measured against the working tree at planning time: 29 distinct command names are invoked as
`$(aether …)`, but every occurrence sits in `.aether/docs/command-playbooks/` or
`.aether/workers.md`, neither of which is caller evidence. The three permitted caller corpora
plus `.aether/utils/hooks` and `scripts/` contain zero such invocations today, so the expected
effect on the seeded count is **none**. The ordering exists so the baseline is correct by
construction rather than by coincidence, and `TestCallerEvidenceCreditsCommandSubstitution`
(task 02.1) keeps it correct as those corpora change.

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
- [x] Feedback latency < 15s for the quick command (task 05.2 excepted by design — see latency note)
- [x] The extractor fix is sequenced ahead of allowlist seeding, and the ordering is asserted by `TestCallerEvidenceCreditsCommandSubstitution`
- [x] The allowlist baseline was seeded from real scanner output, not the assumed count — 278 entries (`cmd/testdata/orphan_allowlist.json`), 6 tagged `owner_phase: "178"`, confirmed both by `172-02-SUMMARY.md` and by re-running `TestNoRegisteredSubcommandIsUnreferenced` directly during this plan's execution (`enumerated 405 registered commands, found 278 orphans`)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved by planner 2026-08-08, revised 2026-08-08 after plan-check — every task
carries an `<automated>` verify; no three consecutive tasks lack one; Wave 0 gaps are each
owned by a named task; the extractor fix was resequenced into wave 1 so the shrink-only
baseline cannot be seeded by a substitution-blind scan.

**Closed out:** 2026-08-11, plan 172-05. All 12 Per-Task Verification Map rows re-run and
confirmed green during this plan's execution (not carried over from prior SUMMARYs by
assumption). Criterion 4's behavioural proof was performed against the exact CI step command,
copied verbatim out of `.github/workflows/ci.yml`, with both the red and green transcripts
recorded in `172-05-SUMMARY.md`. `git status --porcelain` on the caller file used for the proof
(`.aether/commands/skill-create.yaml`) is clean after restoration. `nyquist_compliant`,
`wave_0_complete` and `status` are all set to their completed values in this file's
frontmatter. **Waves 1–4 are complete.**

**Reopened:** 2026-08-11, gap-closure planning. `172-VERIFICATION.md` found success criterion 3
false as stated (a glued fence closer hid a live `midden-recent-failures 50` violation inside
the declared corpus) and criterion 4's durability guarantee false as claimed (a decoy substring
in a non-blocking step satisfies the blanket-gate presence check; deleting the real gate step
leaves the guard green). The wave-4 close above stands as an accurate record of what was
verified *then* — it was not, and is not, a claim that the phase goal is met. Waves 5–6
(`172-06`, `172-07`, `172-08`) close both gaps; this file returns to `status: complete` only
when rows 06.1–08.2 are green and re-verification passes.

**Latency exception, restated for the new waves:** rows 06.3, 07.2 and 08.2 each end in the
full release-gate suite for the reason given in the latency note above. Recorded here so the
exception is a decision on the record rather than an unexplained deviation from the 15s
guidance.
</content>
