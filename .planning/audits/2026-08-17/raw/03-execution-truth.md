# Raw report 03 — Execution Truth Audit

*Verbatim output of the "Audit execution truth" investigation, 2026-08-17, branch `oracle-reinstate`.*

Overall: workers and orchestrators are LLMs whose reports are inherently unverifiable prose, but the runtime wraps them in a layered evidence system: deterministic shell verification it runs itself, existence-checked file claims, SHA-256 artifact snapshots, atomic packet rejection, and hard-block gates. Honest summary: **task "done" status originates from LLM self-report, but phase advancement is runtime-enforced against independently executed tests and hashed artifacts.** Residual honor-system surface: the *content* of reviews and the *authorship* of file claims.

## 1. Task completion — who writes "completed"

- The wrapper LLM (Queen) assembles the completion packet (build.md:22-26, :286-295). Raw per-worker `status:"completed"` is LLM self-report.
- The runtime is the only writer of task state: `runCodexBuildFinalize` (codex_build_finalize.go:379) validates, then `reconcileCompletedBuildTasks` (codex_build.go:2111-2130) flips `Task.Status = TaskCompleted` after the packet survived the validation chain. State commit atomic (:584-596) with durable attempt journal and completion SHA-256 binding (:422-433, 499); a different packet can never replace a committed build (:717-735).
- Status vocabulary normalized (:1209-1232); `"manual"`/`"manually_reconciled"` treated as success-equivalent (:1439, codex_continue_finalize.go:764,866). Bypassable: nothing stops a wrapper labeling a worker manually-reconciled without any human reconciling.
- Continue worker flow: empty status defaults to "completed" (codex_continue.go:3566-3571) — a small fail-open. *(Note: the hostile review later REFUTED this as a practical fail-open — all ingestion paths validate terminal status first.)*

Classification: partially enforced.

## 2. File-modification verification

- **Existence checked; authorship not.** Every claimed path must resolve to a real, regular, non-symlink file inside the repo or the packet is rejected (`validateAndNormalizeClaimPathToRoot` :1629-1676, os.Lstat at :1701-1724). No git-diff reconciliation: `discoverChangedFilesFromGit` (:1539-1547) is used only as a fallback to FILL empty claims (:1309-1311), never as a cross-check. A worker can claim a pre-existing file it never touched.
- Phantom-build rejection: `validateBuildProvenance` (cmd/provenance.go:14-33, SAFE-01/02) — fails if no implementation worker completed, or all completed workers claim zero files. One real claim satisfies it. Discovery/verification-only phases get softer rules (:35-75).
- Post-hoc tamper detection real: `attachBuildArtifactEvidence`/`snapshotBuildArtifact` (criterion_evidence.go:283-346) SHA-256-hash every claimed artifact at finalize; continue re-hashes and blocks on mismatch unless verification re-ran green (:455-495, `verificationReRunProvesArtifacts` :648-663 — refuses to count never-executed "passed" steps).
- Handoffs validated shape, stored prose: `ValidateWorkerHandoff` (pkg/codex/handoff.go:38-51) checks only the verification_status enum; changed_files path-normalized without existence checks (pkg/codex/worker.go:1060-1083). Hard rule: completed worker with EMPTY handoff fails finalize (:1145-1147). Handoff verification_status "pass" never checked against anything.
- `build-reconcile` (cmd/build_reconcile.go:44-182) sweeps git status into a synthetic packet — marked `"synthetic": true`, dry-run honored (:146-149) — but :156 records deleted files as `tests_written` ("for now"), a mislabel in durable claims.

Classification: partially enforced (existence + hash-freeze runtime; authorship honor-system).

## 3. Tests — does the runtime run them?

**Yes — the Go runtime executes the suite itself, on both continue paths.** `runVerificationStep` (codex_continue.go:2901-2975) shells out (`sh -c`, :3914-3916): direct path :1548-1553, external path via `runCodexContinueVerificationSnapshot` (codex_continue_plan.go:227-241, invoked at codex_continue_finalize.go:191). The Watcher is an additional reviewer, not the source of test truth (continue.md:44-46).

Failing tests → blocker → gate fails (:3167-3185); `tests_pass` is compile-time hard_block (gate.go:999, map read-only :991-1002). **`aether continue` has no --force flag.** Phase advance only inside the gated advance (codex_continue_finalize.go:510-521, 1097; codex_continue.go:923).

Bypasses (adversarial list):
- Environment-classed failures downgrade to warnings for non-production phases (:1627-1640).
- No resolvable test command → step "skipped-passed" (:2913-2919, :2957-2965) — unless a criterion names `tests` (hard block :2903-2911). A repo outside detected ecosystems with an unbound plan can advance on claims alone with --skip-watchers.
- `--skip-watchers` stamps the watcher passed (:1581-1582; finalize :193-194).
- Watcher auto-skip after 3 consecutive failures (:1593-1609, threshold :174).
- `ExpectFailingTests` inverts the tests gate for planned-RED phases (:2879-2899) — plan-declared.
- Seal `--force` (commit 07f8997d): cmd/codex_workflow_cmds.go:380-424. Honestly recorded: requires `--reason` (:418-421; :242-244; :425-427), writes a sealed_forced event with counts and reason (:577-587), surfaces force_sealed/unverified_phases in result (:658-663) and CROWNED-ANTHILL.md (:610-623). `validateSealReady` (seal_final_review.go:355-377) blocks unforced seals on incomplete phases and blocker flags. The most honestly recorded bypass in the codebase.

Classification: runtime-enforced, with enumerated, mostly-documented bypasses.

## 4. Watcher / Gatekeeper / Auditor / Probe — gates or advisory?

- Gate tiers hard-coded (gate.go:994-1014): hard_block = gatekeeper, watcher_veto, flags, tests_pass, no_critical_flags, anti_pattern_executed, charter_compliance_executed. Soft_block gates auto-resolved by the Queen when non-critical (codex_continue_finalize.go:298-303). Phase-mode relaxation: gatekeeper/auditor advisory in discovery, soft in prototype/maintenance (gate.go:1037-1074); production full strictness.
- **Verdict content is LLM honor system.** Watcher gate passes iff wrapper-reported status is completed/manually-reconciled (`attachExternalContinueWatcher`, :762-768). Reviewer blocking comes from the reviewer's own structured findings (:866-890 — Blocking/CRITICAL stop the line); nothing verifies the reviewer inspected anything.
- **Concrete bypass:** a missing reviewer result is synthesized as "timeout" (`mergeExternalContinueResults` :608-620); review timeouts are warnings (:896-898); missing/timeout watcher is advisory when the runtime's own shell checks passed (:773-780). An orchestrator that omits every review result advances if deterministic checks are green. `--skip-missing` widens this (:906-912). Deterministic shell verification is the intended backstop.

Classification: blocking in structure, LLM-honor-system in content, backstopped by runtime checks; timeout path bypassable.

## 5. Acceptance criteria

Genuinely machine-checkable for new plans. Criteria must be bound to evidence: artifacts and/or checks from {build, types, lint, tests, claims, watcher} (criterion_evidence.go:22-29); new plans rejected if any criterion unbound (`validateNewPlanEvidenceContract` :111-128); finalize refuses malformed bindings (codex_build_finalize.go:444). `evaluatePhaseCriterionEvidence` (:377-521): artifact claimed + hash-matching (or amended-with-green-rerun), required checks actually executed and passed (`evaluateCriterionCheck` :601-628 — a skipped required check fails). Enforced criteria block continue on both paths (codex_continue.go:1660-1664; codex_continue_plan.go:267-271 — comment records this was previously dead on the external path and was wired). Caveats: legacy plans with unbound criteria are Enforced=false — prose only (:387-390); the watcher check inherits §4's honor-system verdict; `--read-only-artifact` escape hatch is operator-authorized and hash-attested (:437-448; :149-157).

Classification: runtime-enforced (bound-v1) / LLM-honor-system (legacy).

## 6. Per-phase commits

Runtime runs git itself (cmd/phase_commit.go:56-62), strictly after the atomic advance (codex_continue_finalize.go:528), staging ONLY handoff-reported paths that exist or are tracked deletions (:141-152), explicit pathspec (:171-197, no `git add -A`), never pushes (test-locked :36), verifies via rev-parse (:199). Deliberately non-fatal: failed commit never blocks advance but writes `uncommitted-changes.marker` which pauses autopilot (:186-196; autopilot.go marker check). Off switch: `phase-commits set off`. The LLM is never asked to commit.

## 7. Hallucinated-completion defenses (found)

- Phantom-build rejection (provenance.go:14-33; continue-side :132-152 — "Per D-03: rejection causes halt. There is no warn-and-allow path.")
- Whole-packet atomic rejection (D-05/D-06): structural schema validation of raw submitted bytes (:249-330); claim-path violations, duplicate/missing/identity/non-terminal/invalid-handoff violations accumulate and reject before any state mutation (:918-938; :977-1072).
- Claimed-path existence checks; `.aether/data` claim prohibition with sanctioned prefixes (:1588-1594, 1641-1647).
- Manifest freshness/root/colony-mode/plan-revision/attempt binding/digest idempotency (:391-437, 655-674).
- Empty-handoff rejection (:1145-1147 — "what makes the wrapper's mandatory-handoff instruction enforceable rather than prose").
- Artifact SHA freeze + re-hash; read-only evidence can never come from a worker packet (criterion_evidence.go:36-48).
- Dead-gate hygiene: always-pass operational_evidence gate deleted with rationale (codex_continue.go:3204-3211); anti_pattern_executed/charter_compliance_executed exist so "a scan which could not run must never be reported as passing" (gate.go:1001-1002).

Remaining honor-system residue: spawn-tree bookkeeping wrapper-driven (build.md:269-273); review findings content; handoff prose fields; file-claim authorship.

## Dry-run / report-command honesty

- consolidation-phase-end/seal --dry-run locked non-mutating by cmd/consolidation_dryrun_test.go:134,138.
- build-reconcile --dry-run returns before any write (:146-149). Without the flag it is an intentional, marked mutator; deleted-files-as-tests_written (:156) is the one dishonest field found.
- One non-gate mutation inside finalize: `collectPendingSuggestions` runs suggest-analyze post-commit, explicitly non-blocking (:600-653).

## Verdict table

| Enforcement point | Classification |
|---|---|
| Worker task → "completed" | LLM self-report, runtime-written after packet validation |
| File claims exist in repo | Runtime-enforced |
| File claims authored by worker | LLM honor system (no diff reconciliation) |
| Handoff content | Stored prose; shape-validated; empty rejected |
| Tests at continue | Runtime-executed, hard-blocking; bypasses: env-class downgrade, unresolvable-command skip, --skip-watchers |
| Reviewer verdicts | Blocking in structure, LLM-judged content; omitted-result→timeout→warning bypass |
| Acceptance criteria (bound-v1) | Runtime-enforced (hashes + executed checks); legacy prose-only |
| Phase advance | Runtime-enforced, no force flag |
| Seal | Runtime-gated; --force + mandatory --reason, honestly recorded |
| Per-phase commit | Runtime-executed, non-fatal, autopilot-gated |
| Dry-run flags | Honest, test-locked for historical offenders |
