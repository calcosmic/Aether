# Raw report 02 — State Correctness & Recovery Correctness

*Verbatim output of the "Audit state and recovery" investigation, 2026-08-17, branch `oracle-reinstate`.*

## Part A — State correctness

### A1. Schema — WORKS END-TO-END (with caveats)

`ColonyState` (pkg/colony/colony.go:326-381): goal, charter (:296-305), scope, mode, state machine, current phase, plan with immutable revisions (`PlanRevision` :437-455 — every accepted replacement snapshotted with reason + evidence hash), per-phase/task status, success criteria + `EvidenceRequirements` (:638-680), memory, gate results, worktree entries, run ID, paused flags.

Reset-test answers: what/why from goal + charter + ResearchDocs. Phase/completed/failed from state plus per-phase durable evidence dir `.aether/data/build/phase-N/` (manifest.json, attempts/*.json, result-collection.json, worker-reports/*.md, verification.json, gates.json, continue.json, review.json — layout built in cmd/build_attempt.go:95, cmd/codex_build_finalize.go). Files changed: three records — `last-build-claims.json` (last build only, cmd/codex_build.go:883-889); per-phase attempt journal claims with `ArtifactEvidence` — runtime-computed SHA-256/size/mtime snapshots, nonexistent claims silently dropped (cmd/criterion_evidence.go:283-309, attach at codex_build_finalize.go:487, build_attempt.go:726); worker handoffs (cmd/codex_dispatch_contract.go:17), repo-relative (cmd/worker_handoff_test.go:64-67). Verification evidence: genuinely runtime-produced — verification.json stores actual commands run with captured output; gate-results-N.json per phase (cmd/gate.go:1304-1315). What next: `nextCommandFromState` (cmd/recovery_snapshot.go:198) derives it deterministically; session.json stores `suggested_next`.

Caveat: worker-reports/*.md and wave summaries are LLM prose; Events is a flat []string. Load-bearing artifacts are runtime-written.

### A2. Frankenstein-state prevention — WORKS, one bypass

- `state-mutate`: jq-style expressions, typed field mode with validated transitions (cmd/state_cmds.go:262-270 uses colony.Transition), and `--guard task-complete:<id>` / `phase-advance:<N>` preconditions that actually run the repo's test command (state_cmds.go:120-146, gate.go:174-207).
- Every wrapper checked carries a do-not-hand-edit guardrail (build.md:352, continue.md:264, init.md:319, resume.md:12). Zero wrappers instruct direct JSON writes.
- Structural enforcement: `aether hook-pre-tool-use` registered as PreToolUse on Write|Edit and Agent|Task (.claude/settings.json:2-23); `protectedHookWriteReason` blocks writes under .aether/data/ except four sanctioned prefixes (cmd/hook_cmds.go:404-438).
- **Bypass:** hook covers Write|Edit and Agent|Task only — a Bash shell redirection is not intercepted. Present but not airtight.

### A3. Resume — WORKS END-TO-END

`/ant-resume` and `/ant-resume-colony` are thin wrappers over one runtime command (resume aliases resume-colony, cmd/session_flow_cmds.go:172-175). Path (:176-325): freshness check — session age vs 24h + git HEAD vs stored baseline_commit (:19, :37-73); state load fallback chain: COLONY_STATE.json → exact JSON snapshot fenced in HANDOFF.md (`aether-colony-state` fence, :426-443, runtime-written by pause via renderHandoffStateSnapshot, cmd/recovery_snapshot.go:658 — pause→resume round-trips exact state) → lossy legacy markdown parse (:443+, goal/phase/tasks only, last resort). Stale sessions: spawn state cleared, fresh run_id (:216-232); orphaned-worktree GC runs (:261); stale FOCUS pheromones detected (:264-270); handoff consumed and deleted (:308-317).

## Part B — Recovery correctness

### B1. Process dies mid-build — WORKS (in-repo); see B4 for worktree

Partial-dispatch record exists: plan-only writes a durable attempt record (status awaiting) BEFORE any worker spawns (cmd/codex_build.go:247-374, build_attempt.go:75-135); wrapper logs spawn-log/spawn-complete into the spawn tree. Completion bound to attempt by manifest SHA + execution binding (build_attempt.go:458-500). After a kill: state stays EXECUTING; `nextCommandFromState` recommends `aether build N --force`, verified accepted (cmd/interrupted_build_recovery_test.go:10-36). Fresh plan-only supersedes a dangling idle attempt (codex_build.go:247-257). Uncollected file changes recoverable via `build-reconcile` (cmd/build_reconcile.go:44-100), surfaced by /ant-status (cmd/status.go:327, locked by status_test.go:1810) and /ant-recover (recover_repair.go:792-801 — its "not yet implemented" comment is stale; command exists). Genuinely lost on kill: in-flight worker results/handoffs (persisted only at finalize) — files on disk survive and reconcile catches them. Tests: phase_recovery_test.go:13-133, e2e_recovery_test.go:226-538 (10 scenarios incl. compound + destructive).

### B2. Worker timeout/error — PARTIAL; midden claim in CLAUDE.md is FALSE for this path

- Codex-native path: 10-min default timeout, status timeout (pkg/codex/worker.go:19-22, 394-525); terminal statuses persisted (codex_build_finalize.go:1227); failed dispatches → structured redispatch instructions (:766-795).
- Claude path: no runtime timeout. Hung worker leaves a live spawn-tree entry; SPAWN-08 reaper marks abandoned after spawn_reap_threshold_minutes (default 120) at next run start (cmd/spawn_runs.go:24, pkg/colony/colony.go:340-359); /ant-recover scans stale workers (recover_scanner.go:76, recover_test.go:200-273).
- **Midden: nothing on the build/continue failure path writes it.** midden-write is a manual CLI (cmd/midden_cmds.go:400); only chaos.md references it besides display. CLAUDE.md's "Failures are logged during: Build failures (build.md)" is stale. The immune auto-detector reads midden (cmd/immune.go:252-257) and mostly reads an empty file. Present but not wired.

### B3. Context reset mid-phase — WORKS; the documented per-command pattern is mostly dead

Real mechanism is resume-internal (age > 24h or HEAD mismatch ⇒ stale ⇒ spawn state cleared, new run_id, intervention logged; :216-232, tested). `session-verify-fresh` CLI (cmd/session_cmds.go:358-441) is weaker than advertised: without a session_start_time argument every existing file counts as fresh (~:421); it only reports, never clears; only medic.md references it (.claude/commands/ant/medic.md:28). CLAUDE.md's "Session Freshness Detection — Pattern: capture SESSION_START..." describes a workflow current wrappers do not perform. CLI: present but not wired.

### B4. Worktree never merged back — PRESENT BUT UNRELIABLE; one destructive hazard, one hard bug

- Merge-back exists and is gated: `mergePhaseWorktrees` runs tests in the worktree, clash detection, merges; called from build-finalize, failure BLOCKS finalize (cmd/codex_build_worktree.go:1096-1160, codex_build_finalize.go:569-574). Dirty worktrees detected by recover with confirmation (recover_scanner.go:257, recover_repair.go:610, recover_test.go:431-477).
- **Hazard 1 (data loss):** `gcOrphanedWorktrees` — run unconditionally by resume-colony (session_flow_cmds.go:261), init (init_cmd.go:201), and continue (codex_continue.go:540) — removes every Allocated/InProgress/Orphaned worktree via `git worktree remove --force` + `git branch -D` (codex_build_worktree.go:1048-1090, removeGitWorktree :819-837). A build that died between dispatch and finalize with unmerged work: the next resume silently destroys it before any diagnosis. No unmerged-commit or dirty check precedes removal.
- **Hazard 2 (functional bug):** merge gate hard-codes `go test ./...` (:1124). In any non-Go project every branch lands failed and build-finalize aborts "worktree merge-back blocked" — worktree mode cannot complete a build outside a Go repo. `resolveTestCommand()` exists (cmd/gate.go:176) and is not used here.
- Test coverage: dispatch isolation, overlap rejection, pheromone sync-back tested (codex_build_worktree_test.go:145-706), but `mergePhaseWorktrees` has only an empty-case test (:881) — both hazards untested.

### B5. State says complete, git says nothing — PARTIAL

No single invariant, three converging checks: (1) artifact evidence snapshotted from disk at finalize; claimed-but-absent files fail bound criteria during continue (codex_continue.go:2024-2025, codex_continue_finalize.go:155). (2) phase-advance guard re-runs project tests and requires all tasks completed (state_cmds.go:120-146, gate.go:174/400). (3) per-phase auto-commit on durable advance, default on (`commitPhaseAdvance` cmd/phase_commit.go:125, called at codex_continue.go:1000); `aether reconcile` warns on baseline_commit vs HEAD divergence (cmd/reconcile.go:313-314). Gap: no retrospective SHA re-verification of completed phases at resume; phase commit is best-effort, failure doesn't roll back the advance.

### B6. /ant-recover and /ant-medic — WORK END-TO-END, well tested

Recover: scans stuck state, missing packet, stale workers, partial phase, bad manifest, dirty worktrees, broken survey, missing agent files, unreconciled changes (recover_scanner.go:17-494); destructive repairs need confirmation/--force (recover_repair.go:25-155). 30+ tests + e2e compound/destructive scenarios. Medic: JSON validity across state/session/pheromones/JSONL, staleness, integrity; read-only default; `--fix` snapshots a checkpoint first, `autofix-rollback --checkpoint-id` restores (cmd/autofix.go:13-106); truncated-JSON salvage (`findLastValidJSON`, medic_repair.go:789).

## Classification summary

| Capability | Verdict |
|---|---|
| State schema answers reset-test | works end-to-end |
| Files-changed record per phase | works (journal + SHA evidence) |
| Verification evidence per phase | works — runtime-run commands with captured output |
| Frankenstein prevention | works; Bash-redirect hook bypass |
| Resume | works; legacy-markdown fallback lossy |
| Mid-build death (in-repo) | works (attempt journal + --force + reconcile) |
| Worker timeout recording | partial — midden not wired, CLAUDE.md stale |
| Context-reset freshness | works via resume internals; session-verify-fresh CLI vacuous, unwired |
| Worktree reconciliation | present but unreliable — GC data loss; go-test gate; untested merge path |
| Phase-complete vs git | partial — no retrospective re-verification |
| recover/medic | work end-to-end |

**Two findings worth fixing first:** resume-time worktree GC destroying unmerged work (codex_build_worktree.go:1048 + 824/830) and the hard-coded go-test merge gate (:1124) — both on the worktree recovery path, both untested; the first turns a survivable crash into permanent data loss.
