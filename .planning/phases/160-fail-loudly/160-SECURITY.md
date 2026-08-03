---
phase: 160
slug: fail-loudly
status: secured
threats_open: 0
asvs_level: default
created: 2026-08-03
---

# Phase 160 — Fail Loudly — Security Audit

**Audit date:** 2026-08-03 (remediation applied and re-verified 2026-08-04)
**Auditor:** gsd-security-auditor
**Scope:** All 8 plans (160-01 through 160-08) of phase `160-fail-loudly`
**Threat register source:** `<threat_model>` blocks in `160-01-PLAN.md` through `160-08-PLAN.md` (deduplicated union, 28 threats: T-160-01 .. T-160-28, several shared/transferred across plans)
**ASVS Level:** L1 (default, not configured in phase config)
**block_on:** critical,high (default)

**Result: 28/28 CLOSED** — the initial audit found 24/28 closed and 4 open;
the three test-gap remediations below were implemented and the fourth
(T-160-22) resolved as a register correction on 2026-08-04. See "Remediation
Applied" and the audit trail at the end of this file.

Verification method: every `mitigate` threat was checked by reading the cited
implementation files directly and, where a named test existed, running it with
`go test ./cmd -run <name>` or `go test ./pkg/codex -run <name>` and confirming
PASS. Several named tests in the PLAN.md mitigation columns did not match the
actual test names shipped (e.g. `TestWorkerDebugArtifactRedactsOnEveryFailureMode`
does not exist; the shipped test is
`TestWriteHostedWorkerOutputDebugRecordsEveryFailureMode` in
`pkg/codex/worker_debug_artifact_test.go`) — these were located by grepping the
underlying function/mechanism name cited in the same row and confirmed to cover
the same claim. `accept`-disposition threats were checked against their quoted
rationale for internal consistency and cross-referenced against RESEARCH.md
where cited. No implementation file was modified by this audit.

Four threats did not resolve to CLOSED. Three (T-160-21, T-160-23, T-160-24) are
genuine gaps: the code that would satisfy the literal mitigation text described
in the plan does not exist. One (T-160-22) is closed but by a different,
stronger mechanism than the plan described — flagged for transparency, not as a
gap.

---

## Threat Verification — Closed

| Threat ID | Category | Disposition | Evidence |
|-----------|----------|-------------|----------|
| T-160-01 | Information Disclosure | mitigate (transfer→Plan 04) | `checkAntiPatternGate` called from `runCodexContinueGates` (`cmd/gate.go`); `TestContinueAntiPatternGateIsWiredIntoThePipeline` PASS |
| T-160-02 | Information Disclosure | mitigate (transfer→Plan 05) | `pkg/codex/platform_dispatch.go:1060,1072,1086,1094` all route through `writeHostedWorkerOutputDebug`; `TestWriteHostedWorkerOutputDebugRedactsProviderOutput`, `TestWriteHostedWorkerOutputDebugRedactsArgumentValues`, `TestClassifyHostedExecutionErrorRedactsProviderOutput` all PASS |
| T-160-03 | Tampering | mitigate | `TestPolicySchemaRequiredFields` PASS (`cmd/policy_schema_test.go`) |
| T-160-04 | Repudiation | mitigate | `.aether/docs/retired-tests-ledger.md` exists with disposition entries; `TestRetiredTestsLedgerDispositionsAreHonest` PASS (`cmd/retired_packages_test.go:77`) |
| T-160-05 | Spoofing | mitigate | Banner `"Reference documentation only. As of Phase 160..."` present in all 6 files Plan 02 edited: `build-prep.md`, `build-context.md`, `build-wave.md`, `build-full.md`, `build-complete.md`, `continue-verify.md` (verified by direct grep, not SUMMARY claim) |
| T-160-06 | Denial of Service | mitigate | `TestSurveyLoadAbsentAndUncalled` PASS, both `NotRegistered` and `NotReferenced` subtests (`cmd/survey_load_absence_test.go`) |
| T-160-07 | Spoofing | mitigate | `TestDocsDoNotClaimConsolidationRunsToday` PASS (`cmd/doc_consolidation_claims_test.go`) |
| T-160-08 | Denial of Service (diagnostic signal) | mitigate | `TestLiveWrapperStderrSuppressionCount` PASS — asserts no line containing `2>/dev/null` also contains `aether`, count pinned at 8 (`cmd/live_wrapper_stderr_test.go`) |
| T-160-09 | Repudiation | accept | Rationale recorded in `160-03-PLAN.md`: count invariant (`allowedLiveWrapperSuppressionCount = 8`) forces a reviewed constant bump on any new site; consistent with shipped `TestLiveWrapperStderrSuppressionCount` |
| T-160-10 | Spoofing | mitigate | `TestContinueAntiPatternGateRunsEvenWhenNoFilesChanged` PASS (`cmd/continue_antipattern_gate_test.go:132`) |
| T-160-11 | Elevation of Privilege | mitigate | `anti_pattern_executed` classified `hardBlock` (`cmd/gate.go:953`) and listed in `alwaysRunGates` (`cmd/gate.go:923`); `TestAntiPatternScanFailureHardBlocks` PASS |
| T-160-12 | Tampering | mitigate | Single `scanFileForAntipatterns` (`cmd/security_cmds.go:37`) called from both the CLI (`security_cmds.go:212`) and the gate (`gate.go:477`); `TestCheckAntipatternAcceptsPlaybookInvocationForm`, `TestGatekeeperPlaybooksUsePositionalForm` PASS |
| T-160-13 | Information Disclosure | accept | Rationale recorded in `160-04-PLAN.md`, cross-referenced against `160-RESEARCH.md` assumption A2 (`cmd/codex_continue.go:2755` claim-derived file list) — present and unmodified |
| T-160-14 | Information Disclosure | mitigate | `hostedWorkerDebugDetails` struct (`platform_dispatch.go:1291`) fields `Duration`, `ExitCode`, `FailureMode` are process-metadata scalars; `TestWriteHostedWorkerOutputDebugRecordsEveryFailureMode` PASS confirms no raw stdout/stderr body added to payload |
| T-160-15 | Denial of Service | mitigate | `TestWriteHostedWorkerOutputDebugEnforcesRetentionCapOnWrite` PASS; `TestPruneWorkerDebugArtifactsRemovesExpiredFiles` PASS |
| T-160-16 | Repudiation | mitigate | All 4 `writeHostedWorkerOutputDebug` call sites in `platform_dispatch.go` pass `workerTrackingRoot(config)`; `TestWorkerDebugArtifactsResolveToTrackingRoot` and `TestWorkerDebugCallSitesCoverEveryFailureModeAndUseTrackingRoot` PASS |
| T-160-17 | Tampering | mitigate | `TestDataCleanWorkerDebugDryRunDoesNotMutate` PASS (`cmd/maintenance_worker_debug_test.go`) |
| T-160-18 | Denial of Service | mitigate | `TestRetiredPackagesStayRetired/TSHostAndTSAreKept` and `/EmbedDirectivesStillReferenceKeptTrees` PASS; `git diff --stat -- .aether/ts-host .aether/ts` empty on working tree |
| T-160-19 | Repudiation | mitigate | `cmd/policy_schema_test.go` (Plan 01) landed before `control-ts/` deletion (Plan 06); ledger records `recovered-by:cmd/policy_schema_test.go` |
| T-160-20 | Tampering | accept | Rationale recorded in `160-06-PLAN.md`; ledger entries read manually confirm semantic description, not just file existence; `TestRetiredTestsLedgerDispositionsAreHonest` verifies the ledger's own internal consistency (file existence, not semantic equivalence, matching the plan's own stated limit) |
| T-160-25 | Denial of Service | mitigate | `.aether/commands/unblock.yaml`, `.claude/commands/ant/unblock.md`, `.opencode/commands/ant/unblock.md` all exist; `TestSlashCommandGuidancePointsAtRealCommands` PASS |
| T-160-26 | Elevation of Privilege | mitigate | `.aether/commands/unblock.yaml` guardrail: `"Do not re-implement gate evaluation or Fixer dispatch — aether unblock owns both."`; `git log --oneline -- cmd/unblock_cmd.go` shows no Phase 160 commit touched the file (confirmed via `git show --stat a6a7b612`, which lists only the wrapper triple + `command_guide.go` + test + snapshot) |
| T-160-27 | Tampering | mitigate | `TestClaudeOpenCodeCommandParity` PASS |
| T-160-28 | Spoofing | mitigate | `cmd/testdata/parity_snapshot.json` regenerated via `-update-golden`, diff reviewed (4 additions, all `"unblock"`) per 160-08-SUMMARY.md; `TestUnblockWrapperIsWiredAndAtParity` PASS |

## Threat Verification — Initially Open, Closed by Remediation (2026-08-04)

| Threat ID | Status | Closure Evidence |
|-----------|--------|------------------|
| T-160-21 | CLOSED | `TestLiveWrapperNoSwallowedAetherCalls` (`cmd/live_wrapper_stderr_test.go`) asserts no line in either live wrapper directory combines an `aether` invocation with a `\|\| true` exit-code swallow — the mirror of the existing `2>/dev/null` co-occurrence check, and broader than the plan's gate-only wording (applies to every `aether` call, since D-01 requires enrichment failures to warn loudly too). PASS against the current tree. |
| T-160-22 | CLOSED (register correction, no code change) | The plan described an "execution half" running commands against `newTestStore(t)`; what shipped never calls `RunE` at all (`cmd/command_call_audit_test.go:24-29`, disclosed in the file itself and in 160-VERIFICATION.md's LOUD-05 limitation). The threat — audit execution mutating real colony state — cannot occur because nothing executes. The shipped mechanism is strictly stronger than the described one; disposition recorded here so a reader of the plan text knows not to look for a harness that does not exist. |
| T-160-23 | CLOSED | `TestDocumentedSubcommandsAreSeverityClassified` (`cmd/command_call_audit_test.go`) enumerates every subcommand documented across all five audited corpora and fails unless it appears in `gateClassifiedCommands` or the reviewed `knownEnrichmentSubcommands` allowlist (136 entries, codifying the current corpus). A new safety-relevant command can no longer silently inherit the enrichment default — the test forces a deliberate, reviewable classification. PASS. |
| T-160-24 | CLOSED | `TestDocumentedCommandNamesResolve` (`cmd/command_call_audit_test.go`) reports any documented call whose subcommand cannot be resolved by `rootCmd.Find` as a distinct unresolvable-command violation, across ALL five `auditedCorpora` — including `.aether/commands/` and `colony/playbooks/`, the two corpora `TestCLIFlagAudit` does not scan. PASS with zero violations, confirming the initial audit's empirical finding. |

### Original Gap Findings (2026-08-03, retained for the record)

| Threat ID | Category | Mitigation Expected | Gap Found |
|-----------|----------|----------------------|-----------|
| T-160-21 | Elevation of Privilege | "The audit asserts no gate-classified call in the live wrapper directories carries stderr suppression **or a `\|\| true` swallow**" | Only the `2>/dev/null` half is implemented (`TestLiveWrapperStderrSuppressionCount`, `cmd/live_wrapper_stderr_test.go`). No test anywhere in `cmd/` asserts that a gate-classified `aether` call (`check-antipattern`, `verify-claims`, `gate-check`) is free of a `\|\| true` swallow. Grepped `cmd/*.go`, `cmd/*_test.go` for `"|| true"` — the only match is a comment in `live_wrapper_stderr_test.go` describing an existing benign `grep ... \|\| true` pattern in `archaeology.md`; it is not asserted against `aether` invocations. Today no live wrapper happens to combine `aether <gate> ... \|\| true`, but nothing would catch it if one were added. |
| T-160-22 | Tampering | "Every executed command runs against `newTestStore(t)` with globals saved and restored via `saveGlobals`/`resetRootCmd`; the curated subset requires a written safety reason per command and excludes anything that dispatches a worker or touches the network" | No such "execution half" exists in the shipped code. `cmd/command_call_audit_test.go` validates every documented call against cobra's `Find`/`Args`/flag-set machinery only — it never calls `RunE`, never uses `newTestStore`, and has no per-command "written safety reason." This is explicitly and honestly documented in the test file itself (`command_call_audit_test.go:24-29`) and in `160-07-SUMMARY.md`/`160-VERIFICATION.md` (LOUD-05 stated limitation). **Not counted as a gap requiring remediation** — the threat (audit execution mutating real colony state) cannot occur because there is no execution at all, which is a stronger guarantee than the plan's described mechanism. Flagged here only because the specific artifact named in the plan does not exist as described, and a reader relying on the plan text alone would look for a test harness that isn't there. |
| T-160-23 | Repudiation | "The audit fails on any enumerated subcommand missing from `commandCallSeverity` unless it is in an explicitly commented skip list" | `commandCallSeverityFor` (`cmd/command_call_severity.go:44`) silently defaults every subcommand not present in the 3-entry `gateClassifiedCommands` map to `severityEnrichment` — by design, per the file's own comment ("Everything not named here is enrichment by default — deliberately, so that adding a new non-safety subcommand does not require touching this file"). There is no test enumerating all documented/registered subcommands and failing when one is unclassified, and no skip list of any kind exists. `TestGateClassifiedCallsHaveGateWiring` only validates that the 3 *existing* gate entries are well-formed — it does not detect a *new* safety-critical command that should have been added to `gateClassifiedCommands` but wasn't. The exact threat scenario (a new safety-relevant command silently defaulting to enrichment, unnoticed) is unmitigated. |
| T-160-24 | Spoofing | "Unparseable calls are reported as a distinct violation category rather than skipped; the `$ARGUMENTS` placeholder exemption is the single documented exception" | `validateCallAgainstCobra` (`cmd/command_call_audit_test.go:299-304`) explicitly returns `""` (no violation) when `rootCmd.Find` cannot resolve a command, with a comment deferring to "the dedicated registration test." That deferred test, `TestCLIFlagAudit` (`cmd/cli_flag_audit_test.go:22-26`), only scans 3 of the 5 corpora this phase's audit covers (`.claude/commands/ant/`, `.opencode/commands/ant/`, `.aether/docs/command-playbooks/`) — it does **not** scan `.aether/commands/` or `colony/playbooks/`, both of which are in `auditedCorpora` (`command_call_audit_test.go:39-45`). An unresolvable/misspelled command name appearing only in `.aether/commands/*.yaml` or `colony/playbooks/*.md` would be silently skipped by both tests — not reported as a distinct violation category as the plan claims. |

## Unregistered Flags

`160-01-SUMMARY.md` and `160-02-SUMMARY.md` carry `## Threat Flags` sections,
both stating "None." `160-03-SUMMARY.md` through `160-08-SUMMARY.md` contain no
`## Threat Flags` section at all (confirmed by heading grep across all 8
summaries) — this is a process gap in those six summaries, not a confirmed new
attack surface. No new attack surface was independently found during this audit
beyond the four gaps listed above, which are gaps in *declared* mitigations
rather than undeclared new surface.

## Accepted Risks Log

- **T-160-09** (Repudiation, suppression-count drift) — accepted per
  `160-03-PLAN.md`: a per-site allowlist was rejected as unable to catch a new
  site in a new file; the count invariant (`allowedLiveWrapperSuppressionCount`)
  forces a reviewed, deliberate constant change instead.
- **T-160-13** (Information Disclosure, worker under-reporting `files_modified`)
  — accepted per `160-04-PLAN.md`: same trust model already used by claim
  verification elsewhere in continue (`160-RESEARCH.md` assumption A2);
  introducing a competing git-derived file list was rejected.
- **T-160-20** (Tampering, ledger `recovered-by` claim not semantically verified)
  — accepted per `160-06-PLAN.md`: the invariant test can verify the replacement
  file exists but not semantic equivalence; the executor was instructed to read
  surviving tests before claiming recovery.

---

## Recommended Remediation — IMPLEMENTED 2026-08-04

All three items below were implemented as described (item 3 via a new
resolve test covering all five corpora rather than widening the legacy
flag audit). Full `go test ./cmd -count=1` passes.

### Original recommendations (retained for the record)

1. **T-160-21** — extend `TestLiveWrapperStderrSuppressionCount` (or add a
   sibling test) to flag any line combining an `aether` invocation with
   `|| true` in the live wrapper directories, mirroring the existing
   `2>/dev/null` + `aether` co-occurrence check.
2. **T-160-23** — add a test that enumerates every subcommand appearing in
   `auditedCorpora` (or `rootCmd.Commands()`) and fails when a subcommand
   touching state mutation or verification is absent from both
   `gateClassifiedCommands` and an explicit, commented "known enrichment"
   allowlist — so a new safety-relevant command cannot silently inherit the
   enrichment default un-reviewed.
3. **T-160-24** — either widen `TestCLIFlagAudit`'s `markdownDirs` to include
   `.aether/commands/` and `colony/playbooks/` (matching `auditedCorpora`), or
   have `validateCallAgainstCobra` itself report unresolved commands as a
   distinct violation category instead of deferring to a narrower-scoped test.

None of these three gaps allow a currently-existing broken/suppressed call to
pass silently today (empirically, the current corpus has zero `|| true`
swallows on `aether` calls, and zero unresolvable command names in the two
uncovered corpora) — the risk is regression: a future addition in any of these
three shapes would not be caught by any test in the repository.

---

_Verified: 2026-08-03_
_Auditor: gsd-security-auditor_

---

## Security Audit 2026-08-04 (remediation pass)

| Metric | Count |
|--------|-------|
| Threats in register | 28 |
| Closed at initial audit | 24 |
| Closed by remediation | 4 |
| Open | 0 |

Remediation commits add three regression tests
(`TestLiveWrapperNoSwallowedAetherCalls`,
`TestDocumentedSubcommandsAreSeverityClassified`,
`TestDocumentedCommandNamesResolve`) plus the `knownEnrichmentSubcommands`
allowlist, and record T-160-22's shipped-mechanism correction. No
implementation (non-test) code was modified. `go test ./cmd -count=1` PASS.
