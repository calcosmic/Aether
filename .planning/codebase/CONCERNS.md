---
title: Codebase Concerns
last_mapped_commit: 92252d01
---

# Codebase Concerns

**Analysis Date:** 2026-08-22

## Tech Debt

**Large monolithic lifecycle files:**
- Issue: Three core lifecycle files exceed reasonable single-file complexity: `cmd/codex_continue.go` (4155 lines), `cmd/codex_build.go` (3582 lines), `cmd/codex_plan.go` (3431 lines)
- Files: `cmd/codex_continue.go`, `cmd/codex_build.go`, `cmd/codex_plan.go`
- Impact: Difficult to review, test, and maintain. Changes to one area risk breaking distant logic. Test files are equally large (`codex_continue_test.go` = 7300 lines)
- Fix approach: Refactor into focused packages (e.g., separate verification, dispatch, state-advance logic). Start with the smallest file (`codex_plan.go`) to establish the splitting pattern

**Worker write-guard contradiction:**
- Issue: `.aether/data/` is marked protected ("never modify programmatically") in platform rules, but Aether's own planning workers must write artifacts there (`planning/phase-plan.json`, `SCOUT.md`, survey results). The Write tool guardrail blocks these paths, forcing workers to stage files elsewhere and move them
- Files: `pkg/codex/permission_profile.go:96`, `.aether/docs/known-issues.md:52–68`, worker dispatch briefs
- Impact: Workers must evade the protection mechanism that's supposed to prevent accidental mutations. Guardrails that the system's own code bypasses reduce trust in all guardrails
- Fix approach: Carve out `.aether/data/planning/`, `.aether/data/phase-research/`, `.aether/data/survey/`, `.aether/data/worker-debug/` as declared-writable in the permission profile and platform rules. These are runtime-artifact directories with reviewed workflows; they should not be protected

**Dual continue-lane parity requirement not automatically enforced:**
- Issue: Three separate continue implementations (`cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`) must stay behaviorally identical. The milestone brief (v1.27) names this as a standing concern requiring manual verification
- Files: `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`
- Impact: Changes to verification logic, gate behavior, or state advancement in one file risk silently diverging from the others. No test currently asserts end-to-end parity across the three lanes
- Fix approach: Add `TestContinueLanesParity` that runs the same fixture through all three entry points and asserts identical outcomes on blocking issues, advancement decisions, and state mutations

**293 orphaned commands in allowlist:**
- Issue: `cmd/testdata/orphan_allowlist.json` documents 293 commands with no callers. These are registered, discoverable, and unmaintained
- Files: `cmd/testdata/orphan_allowlist.json`, all referenced commands
- Impact: Users can discover and attempt to run commands that are broken, incomplete, or vestigial. The allowlist is a holding pen, not a solution; it only prevents the check from failing
- Fix approach: This is a long-term cleanup. In the short term, three things reduce friction: (1) mark high-risk orphans for deletion (not just "tolerated"); (2) route common discovery attempts (typos, aliases) to the replacement command with a helpful message; (3) run the shrink-focused audits named in `orphan-allowlist-policy.md` before each release. Phase 185 (skill-related command cleanup) targets a subset of 6

---

## Known Bugs

**Evidence gate blocks finalize when reconciliation is supplied:**
- Symptoms: An operator runs `aether continue --plan-only`, supplies `--reconcile-task`, and `aether continue-finalize` still blocks with "verification passed but no implementation evidence or reconciliation was recorded"
- Files: `cmd/codex_continue.go:3061`, `cmd/codex_continue_finalize.go`
- Trigger: Use the external wrapper path (plan-only + finalize) with task reconciliation. The direct `aether continue` path correctly accepts the reconciliation
- Workaround: Use the direct `aether continue` path instead of the plan-only + finalize lanes
- Fix: The finalize path must register `--reconcile-task` as recorded evidence in the `implementation_evidence` gate. Phase 193 analysis confirmed the gate read is present but reconciliation registration is missing

**Same phase verified twice across build and continue boundaries:**
- Symptoms: The same files are reviewed by the Watcher (verification stage) and again by the Probe/Auditor (continue review). Phase measurement on 2026-08-22 shows a one-task bug fix dispatching 8 workers (build: builder, watcher, auditor, probe, tracker; continue: watcher, auditor, probe) vs v5.4.0's 3–4. Much of the cost is duplicate review
- Files: `cmd/codex_build.go` (verification stage), `cmd/codex_continue.go` (continue review dispatch)
- Impact: Unnecessary token spend on repeated checks. The D11 decision (2026-08-22) rules that deterministic checks run on every phase, but reviewer workers should be chosen once, not twice per phase
- Fix approach: Unify verification: move all reviewer dispatch into the continue workflow. The build-side verification stage becomes deterministic checks only (types, lint, tests, "claimed files exist"). Phase 193 onwards

---

## Security Considerations

**Bare auth probe output not sanitized in error messages:**
- Risk: When provider preflight fails, the auth probe may echo raw tokens, credentials, or API responses. Known issue states these must not appear in docs, wrapper narration, debug summaries, or generated context
- Files: `cmd/provider_availability.go`, worker-output parsing paths
- Current mitigation: `sanitizeProviderOutput` should scrub these. Check that all error paths use it
- Recommendations: Add a pre-commit check that scans error output for secrets patterns; never pass raw provider stdout to templates or worker prompts; log raw output to `.aether/data/worker-debug/` only (not visible to user)

**Publish workflow does not guard dirty working tree:**
- Risk: `aether publish` builds the binary and syncs hub from the **working tree with no clean guard**. Uncommitted edits ship silently. Runbook warns to check `git status` first, but the command itself does not enforce it
- Files: `cmd/publish.go`
- Current mitigation: The runbook requires manual preflight (`git status --porcelain` must be empty); breaking CI (force push) currently blocked by `.github/workflows` permissions
- Recommendations: Add a `--force` flag gate to `aether publish` that requires explicit confirmation if the tree is dirty. Default: fail with a message pointing to the runbook. This makes "what is deployed" always traceable to a commit

**Provider availability preflight does not guarantee later success:**
- Risk: Preflight (binary exists, auth probe succeeds) is not a post-launch API guarantee. Real worker request could still fail due to rate limits, account suspension, upstream outages, or proxy failures
- Files: `cmd/provider_availability.go`, worker dispatch paths
- Current mitigation: `aether continue` blocks advancement and suggests recovery. Downstream guidance should be improved
- Recommendations: Classification of post-launch failures before worker-result parsing (Phase 1.1 in the hardening backlog) so users see "provider/account/proxy problem" instead of "JSON parse failure"

---

## Performance Bottlenecks

**TS-host preflight timeout hardcoded and not configurable:**
- Problem: `.aether/ts-host/src/platform-dispatcher.ts:155` uses a hardcoded 20-second timeout for worker platform availability probes. No `AETHER_*PREFLIGHT*` env var or config key exists to tune it
- Files: `.aether/ts-host/src/platform-dispatcher.ts:155`
- Cause: Slow provider auth (e.g., slow OAuth flow, geo-latency) can exceed the timeout and appear as a preflight failure. M4L colony measured ~$0.59 per dispatch attributable to repeated preflight attempts on timeout
- Improvement path: (1) Make timeout configurable via env var `AETHER_PREFLIGHT_TIMEOUT_MS` (default 20000); (2) log actual preflight duration to track patterns; (3) consider async preflight in parallel with worker dispatch (fire probe in background while build context loads)

**Large test files with full scenario coverage cause slow test runs:**
- Problem: `cmd/codex_continue_test.go` (7300 lines), `cmd/codex_build_test.go` (4550 lines), `cmd/codex_plan_test.go` (3131 lines) bundle comprehensive scenario coverage in single test files. Each small change triggers full file compilation and execution
- Files: Large test files listed above
- Cause: Monolithic test organization; no split between unit and integration scenarios
- Improvement path: Refactor test files to separate unit (fast, no fixtures) from integration (slow, realistic). Keep the comprehensive scenarios but run them in a separate test target (e.g., `go test -tags=integration`)

---

## Fragile Areas

**Worktree merge-back is spawned but not completed:**
- Files: `cmd/codex_build_worktree.go`, `cmd/worktree_test.go`
- Why fragile: Agents spawn isolated git worktrees for parallel work but the automatic merge-back step is incomplete. Orphaned branches with valuable code remain after worktree cleanup. Manual `aether worktree-merge-back` exists but is not called automatically
- Safe modification: (1) Add a post-build hook that automatically merges successful worktree branches back to main (with safety checks: verify merge is conflict-free, check that code passed verification); (2) add `TestWorktreeAutoMergeCompletes` asserting that the merge happens and branch is cleaned; (3) handle merge failures gracefully (preserve branch, notify operator, block phase advancement until resolved)

**Wrapper triplet byte-identity requires manual maintenance:**
- Files: `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.codex/agents/*.toml`
- Why fragile: Three separate implementations of each command must stay byte-identical or behavioral parity suffers. The test-enforcer (`TestBuildWrapperStageSkeletonAndParity` and `parity_test.go`) detects divergence but cannot fix it. Edits must be made to all three copies by hand
- Safe modification: (1) Accept that the test will catch divergence; (2) if adding a new command, generate all three from a shared template or use a pre-commit hook that enforces the triplet edit rule; (3) document in DEVELOPMENT.md that editing an ant command requires edits to `.claude/`, `.opencode/`, and `.codex/` versions

**Wrapper-runtime contract relies on documentation, not enforcement:**
- Files: `.aether/docs/wrapper-runtime-ux-contract.md`, wrapper implementations
- Why fragile: The contract says wrappers must not mutate state or duplicate verification logic, but these are guidelines, not guards. A wrapper that edits `.aether/data/` or calls a verification subcommand would only be caught in review, not by CI
- Safe modification: (1) Add a linter that scans wrapper markdown for known risky patterns (directly reading/writing `.aether/data/`, calling verification subcommands, editing pheromones without the runtime); (2) make violations a CI gate failure; (3) document the list of approved state-mutation APIs wrappers can use (if any)

**Hub publish version agreement verification depends on running extra commands:**
- Files: `cmd/publish.go`, the runbook at `.aether/docs/publish-update-runbook.md`
- Why fragile: `aether publish` should fail if binary and hub versions disagree, but the runbook warns that verification requires two separate manual commands (`aether version --check` and `aether integrity`). If forgotten, the publish silently succeeds with mismatched versions downstream
- Safe modification: (1) Make `aether publish` always run version agreement verification as part of sync (not a separate command); (2) fail the publish command if they disagree; (3) log the check result so it's visible in CI; (4) document that `aether publish --force-version-mismatch` is the only way to override (and make that flag require a recorded reason)

---

## Test Coverage Gaps

**Worktree edge cases have incomplete coverage:**
- What's not tested: Concurrent worktree creation failures, merge conflicts during merge-back, git gc racing with worktree operations, orphaned worktree recovery after kill -9
- Files: `cmd/codex_build_worktree_test.go`, `cmd/worktree_test.go`
- Risk: Real interrupted builds can leave the worktree state in an unrecoverable condition. The `worktree-reap` command attempts recovery but edge cases exist where reap itself fails or leaves branches behind
- Priority: High — worktree state affects all parallel builds and has no UI recovery path besides `aether recover`

**Provider availability edge cases:**
- What's not tested: Auth probe succeeds but actual worker request fails (post-launch failure); preflight timeout interaction with slow networks; provider availability returning different results on retry
- Files: `cmd/provider_availability_test.go`
- Risk: Users may be told a platform is ready when it's not, leading to confusing worker failures
- Priority: Medium — most users are on fast networks. Matters for slow cloud auth and proxy deployments

**Dual-verification lane parity:**
- What's not tested: The same blocking issue should be caught identically on the build-side verification stage and the continue review. No test runs a phase through both lanes and asserts identical blocking behavior
- Files: `cmd/codex_build_test.go`, `cmd/codex_continue_test.go`
- Risk: A bug in one verification path could silently pass in the other, leading to divergent behavior
- Priority: High (v1.27 concern) — this is explicitly named in the milestone brief as a dual-lane parity issue

---

## Scaling Limits

**Orphan allowlist growth unchecked:**
- Current capacity: 293 commands (as of 172-09 migration)
- Limit: When orphans exceed ~10% of total registered commands (~50 today), maintenance burden increases significantly
- Scaling path: Establish a quarterly "orphan reduction sprint" that targets 10% shrinkage per quarter. Use keyword/ownership tagging in the allowlist to group related orphans for coordinated cleanup (e.g., "skill-related": 6 commands, targeted for Phase 185)

**Very large test files slow down development iteration:**
- Current capacity: `codex_continue_test.go` at 7300 lines; test runs take >60s on a single file
- Limit: When a single test file takes >90s to run, iteration velocity drops noticeably
- Scaling path: Split test files by concern (unit vs. integration vs. scenario), run unit tests by default, integration as a separate gate. Mark slow tests with `// integration` or a build tag

---

## Scaling Limits

**State file lock contention under parallel builds:**
- Current capacity: In-repo parallel mode allows multiple workers writing to `.aether/data/COLONY_STATE.json` simultaneously with file-level locking
- Limit: Under worktree mode with 8+ parallel workers, lock contention on state reads could become a bottleneck
- Scaling path: (1) Measure state-lock contention under v1.27's expected load; (2) if needed, transition to worker-scoped state snapshots (each worker reads the phase state once, writes results to its own file, then merge at finalize); (3) consider a lightweight append-only event log instead of single shared mutable file

---

## Dependencies at Risk

**TypeScript host unpublished and maintenance-light:**
- Risk: `.aether/ts-host/` is the bridge to Claude/OpenCode dispatch but is not published independently and gets minimal test coverage. Changes to the Aether Go runtime can break ts-host integration
- Impact: If ts-host diverges from Go behavior, the wrapper path (plan-only + finalize) produces different results than the direct path
- Migration plan: (1) Publish ts-host as an npm package (aether-ts-host) so it can be versioned separately; (2) add cross-platform parity tests that run the same scenario through Go and ts-host and assert identical outcomes; (3) establish a ts-host update frequency (e.g., publish together with Go releases)

---

## Missing Critical Features

**No read-only safety mode for sensitive operations:**
- Problem: Commands like `aether publish`, `aether seal`, and `aether-entomb` mutate state directly. There's no `--dry-run` that is guaranteed not to mutate. Users cannot preview their effect
- Blocks: Safe testing of publish workflows; users lack confidence in one-way operations like seal and entomb
- Notes: Some commands have `--dry-run` but it historically mutated state (fixed in v1.25 with locked tests). A comprehensive read-only mode would require significant refactoring of the state-write paths

---

*Concerns audit: 2026-08-22*
