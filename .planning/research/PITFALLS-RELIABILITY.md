# Reliability Pitfalls: Aether v1.23 Daily Driver Restoration

**Domain:** Aether colony framework -- known reliability failure modes causing state leakage, silent work loss, and artifact collection failures across colony lifecycles.
**Researched:** 2026-05-20
**Confidence:** HIGH (root causes confirmed against source code and incident records)

---

## Critical Pitfalls

These failure modes have been observed in production use and have root causes confirmed in the codebase. Each one can cause real user work to be silently lost or corrupted.

---

### Pitfall 1: Frankenstein State from Full-File JSON Reconstruction

**What goes wrong:** An LLM reconstructs the entire COLONY_STATE.json from its context window and writes it via `state-write` or the Write tool. Because the LLM's context contains stale data from a prior colony, a `/clear`, or a compacted conversation, the reconstructed file mixes old and new data. The result is a valid JSON file that claims to be the current colony but contains plan data, pheromone signals, or phase progress from a different colony or an earlier session.

**Why it happens:** Playbooks instruct the LLM to "Write COLONY_STATE.json" after making state changes. The LLM reads the full file, mentally edits a few fields, then writes the whole thing back. If its context has been cleared, compacted, or is from a different conversation, the fields it does not actively edit get filled with stale values from its training or cache. This was first observed when animation repo donation-page plan data merged into an animation colony during `continue`.

**Root cause locations:**
- `build-full.md` line 251: `Write COLONY_STATE.json` (no `state-mutate` guard)
- `continue-full.md` line 1149: `Write COLONY_STATE.json` (no `state-mutate` guard)
- `continue-advance.md` line 303: Explicitly warns against this but only covers the advance playbook itself

**Consequences:** Corrupted colony state that is structurally valid but factually wrong. Phases appear completed that were not. Plans reference tasks from other colonies. Decisions from prior sessions leak into the current one.

**Prevention:** Every COLONY_STATE.json write must use `state-mutate` with targeted jq expressions. The `state-mutate` command performs locked read-modify-write, eliminating the LLM-context-stale-data vector entirely. Audit all playbooks: any instruction that says "Write COLONY_STATE.json" must be replaced with specific `state-mutate` calls for each field change. The continue-advance playbook already demonstrates the correct pattern (lines 305-327).

**Detection:** Grep all playbooks for `Write.*COLONY_STATE\.json` or `state-write`. Any match outside of init (which runs once and writes initial state) is a bug. Add a CI check that blocks commits containing `Write COLONY_STATE.json` in playbook files.

**Warning signs:** Colony state contains plan data or task IDs that do not match the current goal. Phase progress seems inconsistent with what was actually built. `/ant-status` shows phases completed that the user did not run.

**Phase to address:** RUNTIME-01 -- eliminate stale decision/session leakage between colonies.

---

### Pitfall 2: Silent Failure Pipeline (2>/dev/null || true)

**What goes wrong:** Over 80 playbook instructions call CLI subcommands with `2>/dev/null || true` appended, which means every failure is silently swallowed. The pheromone system, midden (failure tracking), memory/learning pipeline, spawn tracking, activity logging, and flag/blocker creation all fail silently. The LLM worker has no idea these calls failed, so it reports the build as successful while critical side-effects never happen.

**Why it happens:** The playbooks were written against an imagined API. The `--worker`, `--source`, `--reason`, `--ttl`, `--section`, `--key`, `--content`, `--path`, `--goal`, `--tags`, `--id`, `--description`, `--summary`, and `--caste` flags did not exist when the playbooks were written. The `|| true` was added to prevent worker failures on these calls from blocking builds. The Go CLI was later updated to add these flags (per the decision "fix Go CLI to match markdown"), but the `|| true` patterns were never removed.

**Root cause locations (representative sample):**
- `build-verify.md` lines 365, 370, 384, 418, 423: midden-write and pheromone-write calls with `2>/dev/null || true`
- `build-wave.md` lines 666, 671, 730, 735: same pattern
- `build-full.md` lines 834, 898, 1346, 1386: same pattern
- `continue-full.md` line 1176: pheromone-write with `2>/dev/null || true`
- `continue-advance.md` line 106: memory-capture with `2>/dev/null || true`
- `continue-gates.md` lines 16, 25, 104, 174, 203, 357, 484, 646, 704, 792, 865, 875, 990, 1000: should-skip-gate and other calls

**Consequences:** The learning pipeline is hollow. Midden never records failures. Pheromone signals are never written. Memory captures are lost. Workers report pheromone and memory operations as completed when they silently failed. Over time, the colony accumulates no learned behavior.

**Prevention:** Remove `2>/dev/null || true` from all playbook calls where failure matters. For calls where graceful degradation is acceptable (e.g., optional telemetry, nice-to-have pheromone signals), add explicit fallback handling: capture the exit code, log a warning, and proceed. The key distinction: `|| true` says "I do not care if this fails." The playbooks should say "if this fails, log it and continue" instead.

**Detection:** Grep playbooks for `2>/dev/null || true` and `2>/dev/null || echo`. Each match needs a decision: (a) remove suppression entirely (failure should propagate), or (b) add explicit fallback handling with a warning. Add a CI check that flags new `|| true` additions to playbook files.

**Warning signs:** `/ant-midden` shows no failures even after builds that had errors. `/ant-pheromones` shows only manually-created signals. `/ant-memory-details` shows zero learnings despite multiple phases completed.

**Phase to address:** RUNTIME-02 -- worker artifact/result collection must preserve completed work. Also relevant to QUEEN-01/02.

---

### Pitfall 3: Worker Artifact Result Loss During Build Interrupt

**What goes wrong:** A build worker completes its task and produces files, but the build is interrupted before the build packet is finalized. The colony remains on an active phase with no usable worker manifest. Because the worker ran in a subprocess (Claude Code agent, OpenCode session, or Codex process), its result JSON was never written to the completion file. The completed code changes exist on disk but are not recorded in colony state.

**Why it happens:** The build-finalize flow requires workers to write a completion JSON to a temp path, then `aether build-finalize` ingests it. If the worker process is killed (user presses Ctrl+C, terminal closes, timeout fires), the completion JSON is never written. The Go runtime correctly detects the missing packet (known-issues.md line 18-21), but the recovery requires `--force` to redispatch, which means the already-completed work must be re-done or manually reconciled.

**Root cause locations:**
- `cmd/codex_build_finalize.go` line 148+: finalizer requires `--completion-file` with a manifest
- `cmd/finalizer_completion_contract.go` line 24-36: validates completion file path but has no fallback for interrupted workers
- `known-issues.md` lines 17-21: documents the issue with mitigations but no code fix

**Consequences:** Workers complete real work (write code, pass tests) but that work is not recorded. The user sees "build packet missing" and must either re-run the entire phase or manually reconcile. In worktree mode, the completed work exists on a branch that is never merged back.

**Prevention:**
1. Workers should write their completion JSON incrementally (start-of-work snapshot + partial results) rather than all-at-once at the end. A partial completion file is better than no completion file.
2. Add a `build-reconcile` command that scans the filesystem for unrecorded worker changes (via git diff, artifact snapshots) and creates a synthetic build packet from what it finds.
3. The build playbook should include a pre-dispatch checkpoint of all tracked files (git status hash) so post-interrupt reconciliation can determine what changed.

**Detection:** After any build interrupt, `aether status` should report "unreconciled worker changes detected" if git shows modifications that are not in the build packet. The existing `recover` scanner (7 stuck-state classes) should add an 8th: "completed worker artifacts without build packet."

**Warning signs:** `aether continue` shows "blocked recovery guidance" about missing build packet. `git status` shows uncommitted changes from workers that colony state does not acknowledge.

**Phase to address:** RUNTIME-02 -- worker artifact/result collection preserves completed work.

---

### Pitfall 4: Pending Decisions Leaking Across Colonies

**What goes wrong:** `pending-decisions.json` is scoped to the `.aether/data/` directory (repo-local), but its entries are not scoped to the current colony goal or session. When a user starts a new colony in the same repo, decisions from the previous colony appear as active. The `filterPendingDecisionFileForScope` function (discuss.go line 125) separates "active" from "stale" decisions, but the stale decisions remain in the file and can confuse downstream commands.

**Why it happens:** Each decision entry has a `session_id` and `goal` field for scoping, but the filtering only happens in `discuss`. Other commands that read `pending-decisions.json` (flag commands, plan commands) may not filter by scope. When a plan reads unresolved decisions from a prior colony, it incorporates stale constraints into the new plan.

**Root cause locations:**
- `cmd/discuss.go` lines 125-126: `filterPendingDecisionFileForScope` correctly separates active vs stale
- `cmd/flag_cmds.go` lines 96, 179, 229, 286, 338: reads pending-decisions without scope filtering
- `cmd/pending_decision.go` line 167: rejects stale decisions on resolve but does not clean them up
- `cmd/discuss.go` lines 590-602: `filterPendingDecisionFileForScope` splits but never deletes stale entries

**Consequences:** New colony plans incorporate constraints from old colonies. A plan for "add user authentication" might carry forward a decision about "don't use Redis for caching" from a previous infrastructure colony. The plan appears more constrained than it should be, or includes irrelevant decisions.

**Prevention:** Every command that reads `pending-decisions.json` must filter by current session scope, not just `discuss`. Add a `prune-stale-decisions` step to `aether init` that archives or removes decisions from previous sessions. Alternatively, scope the file itself by session (e.g., `pending-decisions-{session_id}.json`) so files from old sessions are simply not read.

**Detection:** After `aether init`, running `aether pending-decision --list` should show zero unresolved decisions (not decisions from the previous colony). Add a test that starts two colonies in sequence and verifies the second colony sees no decisions from the first.

**Warning signs:** `aether discuss` shows "ignored N stale clarifications from a prior goal/session" (the existing notice at discuss.go line 175). This notice is itself a warning sign that decisions are leaking.

**Phase to address:** RUNTIME-01 -- stale decision/session leakage between colonies eliminated.

---

### Pitfall 5: Worktree Branch Orphaning

**What goes wrong:** Build waves spawn parallel agents into git worktrees. Each agent commits to its own branch. But when a build is interrupted or the build-complete step does not run merge-back, those worktree branches accumulate with genuinely valuable code that was never merged to the target branch. The work exists but is invisible to subsequent builds and continues.

**Why it happens:** The worktree merge-back is a single step in the continue-advance playbook (Step 2.0.4, continue-advance.md line 333). It runs only during `continue`, not during `build-complete` or on interrupt. If the build phase completes but the user never runs `continue`, the merge-back never happens. Additionally, the worktree merge-back is marked NON-BLOCKING, meaning failures create blockers but do not halt the flow.

**Root cause locations:**
- `continue-advance.md` lines 333-380: Step 2.0.4 worktree merge-back
- `cmd/codex_build_worktree.go`: worktree creation but no guaranteed merge-back path
- Memory: `project_worktree_merge_gap.md` documents 13 orphaned worktree branches with valuable Go code

**Consequences:** Valuable code (queen V2, graph BFS/cycle detection) lived on orphaned branches that were nearly lost. New builds start from the target branch and cannot see worktree-only changes. After `continue` advances the phase, the user has no indication that unmerged worktree branches exist.

**Prevention:**
1. Add worktree merge-back to `build-complete` (not just `continue-advance`), so it runs when the phase finishes regardless of whether the user runs continue.
2. Add a pre-build check that scans for existing worktree branches from prior builds and either merges them or warns the user.
3. The `aether status` command should report orphaned worktree branches as a health issue.

**Detection:** Before any `aether build`, check `git worktree list` for branches not merged into the target. Report them. The existing `scanDirtyWorktrees` in `recover_scanner.go` already detects dirty worktrees but may not detect clean-but-unmerged ones.

**Warning signs:** `git worktree list` shows branches not present in `git log --oneline` for the target branch. `git branch` shows branches with commit messages referencing colony phases.

**Phase to address:** RUNTIME-02 -- worker artifact/result collection preserves completed work.

---

### Pitfall 6: Provider API Failures Misreported as Parse Errors

**What goes wrong:** When a real worker process is dispatched to Claude Code, OpenCode, or Codex, and the provider returns an auth/API failure (rate limit, auth expired, model unavailable), the runtime sees non-JSON output where it expects worker claims JSON. The error is reported as "parse worker output: no JSON found in output" rather than "provider auth failure" or "API rate limited."

**Why it happens:** The finalizer's worker-result parsing runs before provider error classification. The completion file contains raw terminal output from the provider's error response, which is not valid worker claims JSON. The error message tells the user to check their worker, but the real problem is their provider credentials or API access.

**Root cause locations:**
- `known-issues.md` lines 35-38: documents the issue
- `cmd/codex_build_finalize.go` line 210+: `mergeExternalBuildResults` parses JSON before checking for provider errors
- `cmd/result_validation.go` line 23+: `validateWorkerResultIdentity` checks field match but does not check for provider error payloads

**Consequences:** Users waste time debugging their worker configuration when the real problem is an expired API key or rate limit. Recovery steps point to worker files and debug artifacts rather than provider health.

**Prevention:** Before parsing worker results, check the output for known provider error patterns (e.g., "AUTH_ERROR", "rate limit", "401", "403", "model not found", "insufficient_quota"). If a provider error pattern is detected, report it as a provider/API issue rather than a parse failure. The known-issues.md already identifies this as the next hardening priority.

**Detection:** Add test cases where the completion file contains provider error payloads instead of worker claims. Verify the error message mentions the provider, not the parser.

**Warning signs:** Build-finalize reports "parse worker output: no JSON found" when the worker definitely ran (it appears in the process list). The raw output file contains HTML or plain-text error messages from the provider.

**Phase to address:** QUEEN-01/02 -- deterministic commands and Queen execution policy should handle provider failures gracefully.

---

## Moderate Pitfalls

---

### Pitfall 7: Session File Stale Between Colonies

**What goes wrong:** When a colony is sealed or entombed, `session.json` should be cleaned up. But if the seal/entomb is interrupted or the user starts a new colony without properly closing the old one, the session file from the previous colony persists. Resume commands detect this (session_cmds.go line 37-73 checks age and git HEAD match) but the detection only fires on explicit resume, not on init or plan.

**Root cause locations:**
- `cmd/session_flow_cmds.go` lines 37-73: `sessionVerifyFresh` checks age and git HEAD
- `cmd/entomb_cmd.go` line 520: entomb clears session.json but only if entomb completes
- `cmd/init_cmd.go`: init may not clear session.json from a prior colony

**Consequences:** New colony starts with stale session data. Context capsule may load stale state. Session-verify-fresh checks in playbooks may report stale sessions unexpectedly during normal workflows.

**Prevention:** `aether init` should always clear `session.json` before writing a new one. Add session cleanup to `aether seal` as a finalizer step (not just entomb). The 24-hour staleness threshold should also apply to init -- if a session is older than 24 hours, init should warn and offer to clear it.

**Phase to address:** RUNTIME-01 -- stale decision/session leakage.

---

### Pitfall 8: Completion File Temp Path Race Conditions

**What goes wrong:** Build completion files are written to temp paths like `${TMPDIR}/aether-build-<run>/build-completion.json`. Multiple concurrent builds (unlikely but possible in worktree mode) could collide on temp paths if run IDs are not sufficiently unique. The finalizer reads from this path and expects exactly one completion file.

**Root cause locations:**
- `cmd/finalizer_completion_contract.go` line 11: `finalizerCompletionTempPattern = "${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json"`

**Consequences:** Build-finalize reads the wrong completion file if two builds share a temp directory. Worker results get mixed between builds.

**Prevention:** Ensure the `<run>` component includes the session ID, timestamp, or PID for uniqueness. The current pattern relies on callers generating unique run IDs, but there is no enforcement.

**Phase to address:** RUNTIME-02 -- worker artifact collection.

---

### Pitfall 9: Handoff Section Injection Without Worker Data

**What goes wrong:** Colony-prime injects a "Previous Worker Handoffs" section into worker context even when no handoff data exists. The `renderWorkerHandoffSection` function (worker_handoff_test.go line 70) returns a section regardless of whether any entries match the requested workflow and phase. An empty or stale handoff section wastes context budget tokens.

**Root cause locations:**
- `cmd/codex_dispatch_contract.go` line 16: `workerHandoffsPath = "handoffs/worker-handoffs.json"`
- `cmd/worker_handoff_test.go` line 70: renders section content from matching entries

**Prevention:** Only inject the handoff section when there are matching entries for the current workflow and phase. Skip entirely when the handoffs file is empty or has no relevant entries. This saves context budget tokens and avoids confusing workers with irrelevant handoff data.

**Phase to address:** QUEEN-01/02 -- Queen execution policy.

---

### Pitfall 10: Event Array Unbounded Growth

**What goes wrong:** COLONY_STATE.json has an `events` array that gets appended to on every state transition. While continue-advance caps it at 100 entries (line 300), build-full.md line 247 also appends events during build state update but does not enforce the cap in the same write. If builds run without continues (e.g., multiple consecutive builds without advancing), the events array grows without bound.

**Root cause locations:**
- `build-full.md` line 247: "Append to `events`" without cap
- `build-full.md` line 251: "Write COLONY_STATE.json" without `state-mutate`
- `continue-advance.md` line 300: "Keep max 100 events" (only runs during continue)

**Prevention:** The events cap should be enforced in `state-mutate` itself (as a post-write validation step) rather than relying on every caller to remember to cap it. Alternatively, add the cap to the build-full state update step.

**Phase to address:** RUNTIME-01 -- state mutation safety.

---

## Minor Pitfalls

---

### Pitfall 11: Context Capsule Budget Wasted on Stale Data

**What goes wrong:** The context capsule loads sections (QUEEN.md wisdom, hive wisdom, pheromone signals, phase learnings) that may be stale from a prior colony. The freshness scoring (context.go line 52-63) helps rank sections, but stale sections are still loaded and trimmed, consuming the budget that should go to relevant current-colony data.

**Prevention:** Scope context capsule sections to the current session ID or goal. On colony init, clear or archive context capsule data from prior sessions.

**Phase to address:** RUNTIME-01.

---

### Pitfall 12: Git Checkpoint Without Automatic Restore

**What goes wrong:** Build-playbook Step 3 creates a git checkpoint for rollback capability, but there is no automated restore path. The user must manually `git reset` to the checkpoint if the build fails. The checkpoint exists but is never referenced by recovery commands.

**Prevention:** Wire git checkpoints into `aether recover` as a "restore to pre-build state" option. The checkpoint path is deterministic (`checkpoints/pre-build-phase-N.json`).

**Phase to address:** RUNTIME-02.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `2>/dev/null \|\| true` on CLI calls | Build does not fail when a subcommand is missing or broken | Learning pipeline hollow; failures never recorded; pheromone system non-functional | Never in production playbooks |
| Full JSON reconstruction via Write tool | Simpler playbook instructions (LLM writes what it reads) | Frankenstein state corruption from stale context | Only in `aether init` (writes initial state once) |
| Non-blocking worktree merge-back | Continue flow never halts due to merge issues | Worktree branches silently accumulate, valuable code orphaned | Short-term only; must add merge-back to build-complete |
| Event array cap only in continue | Simpler build playbooks | Unbounded growth if user runs builds without continues | Never; cap should be enforced by state-mutate |
| Session scoping only in discuss | Simpler flag and pending-decision commands | Stale decisions leak into new colonies | Never; all readers must scope by session |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical reliability pieces.

- [ ] **state-mutate migration:** `build-full.md` still says "Write COLONY_STATE.json" on line 251. Verify all 3 playbooks use `state-mutate` exclusively.
- [ ] **Silent failure removal:** Grep for `2>/dev/null \|\| true` in playbooks. Verify each has been either removed or replaced with explicit fallback handling. Zero `|| true` on midden-write, pheromone-write, or memory-capture calls.
- [ ] **Worktree merge-back on build-complete:** Verify that `build-complete` step (not just `continue-advance` step 2.0.4) merges worktree branches.
- [ ] **Pending decision scoping:** Verify that `flag_cmds.go` and `pending_decision.go` filter by current session scope, not just `discuss.go`.
- [ ] **Provider error classification:** Verify that `mergeExternalBuildResults` checks for provider error patterns before JSON parsing.
- [ ] **Session cleanup on init:** Verify that `aether init` clears `session.json` from prior colonies.
- [ ] **Handoff section conditional injection:** Verify colony-prime skips the handoff section when no matching entries exist.
- [ ] **Event cap in build path:** Verify events are capped at 100 during build state updates, not just during continue.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Frankenstein state | HIGH | Restore from checkpoint (`checkpoints/pre-build-phase-N.json`). If no checkpoint, manually reconstruct from git log and worker handoffs. Prevention (state-mutate) is far cheaper than recovery. |
| Silent failure accumulation | LOW | Re-run failed calls manually. Check midden for missing entries. Run `aether data-clean` to reset, then re-run builds to regenerate. |
| Worker artifact loss on interrupt | MEDIUM | `git status` shows uncommitted changes. `git diff` shows worker modifications. Run `aether build N --force` to redispatch. For worktree mode, merge orphan branches manually. |
| Pending decision leakage | LOW | `aether pending-decision --list` shows stale decisions. `aether discuss` reports stale count. Archive stale entries and re-run plan. |
| Worktree branch orphaning | MEDIUM | `git worktree list` identifies orphan branches. `git log branch-name --oneline` shows commits. Merge manually into target branch. If branches were deleted, `git reflog` may recover them. |
| Provider error misreport | LOW | Check raw output file for provider error text. Fix auth/credentials. Retry the build. |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Frankenstein state (Pitfall 1) | RUNTIME-01 | Grep playbooks for `Write.*COLONY_STATE\.json` -- zero matches in build/continue playbooks. Add CI gate. |
| Silent failure pipeline (Pitfall 2) | RUNTIME-02 | Grep playbooks for `\|\| true` -- zero matches on midden-write, pheromone-write, memory-capture. End-to-end test: build, then verify midden has entries. |
| Worker artifact loss (Pitfall 3) | RUNTIME-02 | Interrupt a build mid-phase, then run `aether status`. Should report unreconciled changes. `aether build-reconcile` should create synthetic build packet. |
| Pending decision leakage (Pitfall 4) | RUNTIME-01 | Start colony A, create a decision, seal. Start colony B. `aether pending-decision --list` should show zero entries. |
| Worktree branch orphaning (Pitfall 5) | RUNTIME-02 | Run build in worktree mode, interrupt before continue. `git worktree list` should show branches. `aether status` should report them. |
| Provider error misreport (Pitfall 6) | QUEEN-01/02 | Simulate provider auth failure in completion file. Verify error message mentions provider, not parser. |
| Session file staleness (Pitfall 7) | RUNTIME-01 | Seal colony, start new colony. `session.json` should contain only new session data. |
| Completion file race (Pitfall 8) | RUNTIME-02 | Verify temp path includes session ID. No collisions possible. |
| Handoff section injection (Pitfall 9) | QUEEN-01/02 | Start fresh colony with no prior builds. Colony-prime output should not include "Previous Worker Handoffs" section. |
| Event array growth (Pitfall 10) | RUNTIME-01 | Run 5 consecutive builds without continue. Events array should never exceed 100 entries. |

---

## Sources

- **HIGH confidence** -- Direct source code inspection:
  - `cmd/codex_build_finalize.go` lines 148-229: finalizer completion contract and validation
  - `cmd/codex_dispatch_contract.go` line 16: worker handoffs path
  - `cmd/finalizer_completion_contract.go` lines 11-57: temp path pattern and freshness validation
  - `cmd/discuss.go` lines 125-126, 590-602: pending decision scope filtering
  - `cmd/pending_decision.go` lines 157-193: pending decision resolution
  - `cmd/flag_cmds.go` lines 96, 179, 229, 286, 338: pending decision reads without scope filtering
  - `cmd/recover_scanner.go` lines 17-42: 7 stuck-state detectors
  - `cmd/session_flow_cmds.go` lines 37-73: session freshness verification
  - `cmd/result_validation.go`: worker result identity validation
  - `cmd/codex_worker_artifacts.go`: artifact snapshot and preservation logic

- **HIGH confidence** -- Playbook source inspection:
  - `.aether/docs/command-playbooks/build-full.md` lines 240-251: full JSON write (no state-mutate)
  - `.aether/docs/command-playbooks/continue-full.md` line 1149: full JSON write (no state-mutate)
  - `.aether/docs/command-playbooks/continue-advance.md` lines 303-327: correct state-mutate pattern
  - `.aether/docs/command-playbooks/continue-advance.md` lines 333-380: worktree merge-back step

- **HIGH confidence** -- Incident records:
  - `.claude/projects/-Users-callumcowie-repos-Aether/memory/project_state_corruption_bug.md`: Frankenstein state evidence
  - `.claude/projects/-Users-callumcowie-repos-Aether/memory/project_cli_flag_mismatch.md`: Silent failure evidence
  - `.claude/projects/-Users-callumcowie-repos-Aether/memory/project_worktree_merge_gap.md`: Worktree orphaning evidence
  - `.aether/docs/known-issues.md`: Provider error misreporting (lines 35-38), build interrupt recovery (lines 17-21)

- **MEDIUM confidence** -- Inferred from code patterns:
  - Session file staleness between colonies (code shows cleanup in entomb but not init)
  - Handoff section unconditional injection (test shows rendering but no empty-skip check)
  - Event array cap gap in build path (cap exists in continue but not build)

---

*Reliability pitfalls research for: Aether v1.23 Daily Driver Reliability*
*Researched: 2026-05-20*
