# Phase 160: Fail Loudly - Context

**Gathered:** 2026-07-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Every playbook and wrapper CLI call either succeeds or visibly fails — no call
silently swallowed. `control-ts/` deleted as a standalone first commit;
`.aether/ts-host/` and `.aether/ts/` explicitly kept. Docs stop describing
behaviour that does not happen. Plus (folded 2026-07-27): worker debug
artifacts become reliable evidence — written on every failure mode, capped,
and surviving worktree cleanup.

**Reconciliation note (verified by execution, 2026-07-27):** a large part of
this phase's original scope was completed by the "Close the Loops" work
(commits `281dd34a`..`758dbd85`) and the dispatch-fix session (`b2b41486`).
The plan for this phase must START from the reconciliation table below — the
remaining work is smaller than the requirement list suggests, but each
already-done item still needs its verification command per the Definition of
Done (a command that fails when the requirement is unmet).

| Requirement | Verified state 2026-07-27 |
|---|---|
| LOUD-01 (survey-load) | Done differently: command deleted, zero references remain. Verify: absence + no stale callers |
| LOUD-02 (check-antipattern gate) | Done: positional form fixed, pinned by `cmd/security_gate_drift_test.go` |
| LOUD-03 (five remaining calls) | All exist and error loudly when misused; needs the execution audit to formally pin |
| LOUD-04 (positional drift test) | NOT verified — `cmd/cli_flag_audit_test.go` positional coverage unknown |
| LOUD-05 (execution audit) | NOT done — no run-the-commands audit exists |
| LOUD-06 (stderr suppression) | Mostly done: only 4 benign `2>/dev/null` sites remain (grep fallbacks in archaeology.md, git call in dream.md) |
| LOUD-07 (docs claim consolidation runs) | Almost: structural-learning-stack.md + AGENTS.md corrected; `CLAUDE.md:838` still claims "phase-end at /ant-continue" while `CLAUDE.md:37` says the opposite |
| LOUD-08 (/ant-unblock) | NOT done — `cmd/unblock_cmd.go:126` still points at nonexistent command |
| RETIRE-01 (delete control-ts) | NOT done — directory exists |
| RETIRE-02/03 (keep ts-host, ts) | Confirmed live and load-bearing (ts-host preflight cap + dist rebuild proven necessary 2026-07-27) |
| RETIRE-04 (deleted-test ledger) | NOT done; `control-ts/tests/schemas/policy.schema.test.ts` replacement constraint stands. Note: `.aether/ts-host/test/playbook-loader.test.ts` was already deleted (commit `b2b41486`) as orphaned — record it in the ledger retroactively |

</domain>

<decisions>
## Implementation Decisions

### Failure behavior (what "fail loudly" means at runtime)
- **D-01:** Severity depends on the call. Safety gates (security scan,
  verification, claims checks) HALT the run when they cannot execute — a build
  must not "pass" without its gate having run. Context/enrichment calls
  (survey, skill matching, progress rendering) show a loud, visible warning
  and the run continues degraded. The plan must classify every audited call
  into gate vs enrichment and encode the classification where the audit can
  test it.

### Unblock guidance (LOUD-08's either/or)
- **D-02:** Build the missing `/ant-unblock` wrapper on Claude Code and
  OpenCode (YAML source + generated wrappers, same chain as other commands),
  so `cmd/unblock_cmd.go`'s guidance becomes true instead of being reworded.
  Fits the milestone theme: switch on what exists — the Fixer dispatch flow
  (`aether unblock --dispatch`, modes full/propose/advise) already works.

### Folded findings from 2026-07-27 dispatch investigation (debug-file trio)
- **D-03:** Debug artifacts are written on EVERY worker failure mode — parse
  failure (exists today), timeout, and non-zero exit — and include exit code,
  duration, and provider session id. Evidence that vanishes is the quietest
  silent failure there is.
- **D-04:** Debug artifacts get a retention cap (suggested: 50 files or 14
  days, pruned on write) and `worker-debug` is wired into `aether data-clean`.
- **D-05:** In worktree mode, debug artifacts are written to the tracking
  root (not the worktree), so they survive `git worktree remove` and the
  `(debug: <path>)` hint in errors resolves from the user's cwd.

### Claude's Discretion
- Ledger format/location for RETIRE-04, drift-audit mechanics (LOUD-04/05),
  gate-vs-enrichment classification details, wrapper wording for /ant-unblock.
- CLAUDE.md:838 fix is a one-line deletion/rewrite — fold into LOUD-07 work.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §LOUD-01..08, §RETIRE-01..04 — the phase's requirement text
- `.planning/ROADMAP.md` — Phase 160 section, incl. build.md ownership note (only narrow call-argument fixes here; structural rewrite belongs to Phase 165)

### Evidence from the 2026-07-27 sessions (why the folded decisions exist)
- `.planning/SESSION-HANDOFF.md` — v1.25 Close the Loops completion state
- `pkg/codex/platform_dispatch.go` — `writeHostedWorkerOutputDebug` (only called on parse failure today), `safeHostedWorkerArgs` (identity-based redaction, added 2026-07-27), `runHostedProviderPreflight`
- `pkg/codex/testdata/m4l_scout_payload_20260727.txt` — the real failing payload, already a fixture
- `cmd/codex_build_worktree.go` — sync-back path that currently loses `.aether/data/` artifacts (`snapshotGitStatus` ~line 870, `finalizeBuildWorktree` ~line 452)
- `cmd/unblock_cmd.go:126` — the guidance string LOUD-08 targets
- `cmd/security_gate_drift_test.go` — the drift-test shape to generalize for LOUD-04/05

### Constraints
- `control-ts/tests/schemas/policy.schema.test.ts` — the only `model-routing.yaml` schema validation; replacement must exist before/alongside deletion (RETIRE-04)
- `.aether/ts-host/` — KEEP; embedded via `cmd/embedded_assets.go:13`; deleting breaks go build, publish, integrity
- Editing `.aether/ts-host/src/*.ts` requires `npm --prefix .aether/ts-host run build` or dist stays stale (proven 2026-07-27)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/security_gate_drift_test.go` — pins exit code + stderr + regex for one command; the pattern to generalize into the execution audit
- `writeHostedWorkerOutputDebug` — single chokepoint for debug artifacts; D-03/D-04/D-05 all land in/around it
- Command YAML → wrapper generation chain (`.aether/commands/*.yaml`) — the path for the new `/ant-unblock` wrapper

### Established Patterns
- Tolerant-decode pattern established 2026-07-27: `stringList` (pkg/codex) and `planScore` (cmd) — one loose field must not discard a whole artifact; tolerance never becomes silence (garbage still errors)
- Tests assert invariants/proportions, not named sections (Definition of Done corollary)
- Hub-published files (agents, commands) need `aether publish` after edit; agent bodies must stay Claude/OpenCode parity-identical (`TestClaudeOpenCodeAgentContentParity`)

### Integration Points
- `aether data-clean` (`cmd/maintenance.go`) — where worker-debug retention wires in
- Goldens `cmd/testdata/{command_catalog,parity_snapshot,regression}` need `-update-golden` when the /ant-unblock command is added

</code_context>

<specifics>
## Specific Ideas

- The user experienced the silent-failure cost firsthand on 2026-07-27 (three
  failed /ant-plan runs whose real cause was invisible). "Fail loudly" should
  be judged against that bar: could an operator have diagnosed it from the
  terminal + debug file alone, without hunting session transcripts?

</specifics>

<deferred>
## Deferred Ideas

- **Route-setter permission contradiction** (`pkg/codex/permission_profile.go:96`
  says "without repository writes" while the brief orders writing
  phase-plan.json; same class: `cmd/phase_research.go:124` vs read-only scout
  profile) — **Phase 163 (Context Reaches Workers)**
- **`artifacts` sub-schema only accepts `{}`** (`pkg/codex/worker.go:921-926`
  makes the response contract's escape hatch unusable) — **Phase 163**
- **Pull visual work (Phase 168) earlier** — raised, not selected for
  discussion; roadmap order stands unless the user revisits
- Preflight env knob (`AETHER_PREFLIGHT_TIMEOUT` following the
  `AETHER_CONTINUE_VERIFICATION_TIMEOUT` pattern) and deleting the dead TS
  preflight in `platform-dispatcher.ts` — nice-to-haves, any phase
- Stale worktree copies under `cmd/.aether/worktrees/` pollute repo-wide
  greps — cleanup candidate, no phase assigned

</deferred>

---

*Phase: 160-Fail Loudly*
*Context gathered: 2026-07-27*
