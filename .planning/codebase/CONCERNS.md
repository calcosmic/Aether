# Codebase Concerns

**Analysis Date:** 2026-08-01

## Tech Debt

### State Corruption via Full-File Reconstruction

**Issue:** Colony state mutations in playbooks and commands reconstruct the entire COLONY_STATE.json in LLM context, then write via `state-mutate`. When context is stale (from previous colonies or conversation context), this produces "Frankenstein" state mixing old and new data.

**Files:** `cmd/codex_continue.go` (state advance logic), `.claude/commands/ant/continue.md` (orchestration), `.claude/commands/ant/init.md` (colony initialization)

**Impact:** Colony state data loss, phase tracking corruption, cross-colony data contamination. Documented instance: donation page plan data merged into animation colony state.

**Fix approach:** Replace full-file reconstruction with targeted `state-mutate '<jq_expression>'` calls for each field change. The mutation function exists in state-api.sh and does atomic locked read-modify-write, eliminating LLM-context-stale-data vector.

### Learning Pipeline Never Executes

**Issue:** `consolidation-phase-end` and `consolidation-seal` commands exist, are documented, and are tested, but have no real callers. The entire wisdom consolidation pipeline (Observe→Promote→Queen→Consolidate) only runs inside these two isolated commands; no lifecycle command invokes them.

**Files:** `cmd/graph_consolidation_cmds.go` (command definitions at line 205 and 270), `pkg/memory/pipeline.go` (pipeline construction at line 217, 303), `cmd/doc_consolidation_claims_test.go` (test that verifies they have no callers — this is intentional)

**Impact:** Colony learning disabled by default. Workers cannot extract instincts or promote patterns. The Queen never receives learned preferences. Cross-colony hive brain remains empty. Claimed features (automatic pattern capture, phase learning) are non-functional.

**Fix approach:** Phase 162 work: wire real callers into `/ant-continue` and `/ant-seal` lifecycle. Update all documentation claims in CLAUDE.md, AGENTS.md, and `.aether/docs/structural-learning-stack.md` when wiring completes.

### Machinery Exists But Is Never Wired

**Issue:** Verified by specialist review July 2026 — Aether capability is rarely deleted; it is built, tested, documented, and never called. Confirmed examples:

- `pkg/colony/policies/` (7 of 10 policy files unread)
- `colony-vital-signs` health computation (0 readers — unplugged from `/ant-status`)
- Approved charter governance detection (zero reader references in `cmd/colony_prime_context.go`)
- Check-antipattern Gatekeeper security scan (playbook redirects stderr to `/dev/null`, silent failures)

**Files:** Multiple; see `.claude/projects/-Users-callumcowie-repos-Aether/memory/project_orphaned_playbooks.md` for verified examples with line citations

**Impact:** Users get partial or non-functional features because the calling machinery was never wired. Static analysis (grep, linting) cannot catch this; only execution reveals wiring gaps.

**Fix approach:** When triaging feature gaps, grep for **callers**, not just definitions. Execute commands before concluding they work. Treat the Go runtime in `cmd/` and Codex guides (`AGENTS.md`, `.codex/CODEX.md`) as authoritative; verify against execution.

## Known Bugs

### Interrupted Phase Builds Strand Worktree Branches

**Issue:** GSD's `execute-phase` merges each wave's worktree branches back and deletes them at phase end. When a run is interrupted (context exhaustion, crash, spend limit), branch cleanup never runs. Orphaned branches accumulate with no owner and are never reaped.

**Files:** `.aether/data/` worktree refs, `cmd/worktree*.go` cleanup logic

**Impact:** Dead branches accumulate indefinitely (audit 2026-07-27 found 10 accumulated since May 2026). While cleanup is needed, the risk of data loss is overstated by commit counts: `git rev-list --count main..<branch>` reports diverged lineage, not real lost changes.

**Fix approach:** Audit before deletion using: (1) grep main for branch feature headlines, (2) verify all files the branch ADDED exist in main, (3) for genuinely new source files, grep main for branch symbols (scaffold often splits across files). Record tip SHAs in ledger before deletion; `git branch <name> <sha>` restores until gc.

### Untracked Plan Files Lost on Worktree Merge-Back

**Issue:** GSD executor agents write phase plan files (PLAN.md, SUMMARY.md, RESEARCH.md, VALIDATION.md) but do not commit them before the orchestrator force-removes the worktree. These untracked files vanish without recovery.

**Files:** `.planning/` directory (tracked but gitignored), GSD executor agents' file output

**Impact:** Planning artifacts and summaries are lost, breaking post-phase analysis and carry-forward context.

**Fix approach:** Require GSD executor agents to commit planning artifacts with `git add -f .planning/` before returning (files need force-add due to gitignore). Confirm with `git log --stat -1` that SUMMARY.md is in the commit before returning.

### .planning is Gitignored but Tracked

**Issue:** `.planning/` matches a `.gitignore` entry (line 123) while ~71 files under it are tracked. Already-tracked files commit normally; any **new** file fails `git add` with "paths are ignored by one of your .gitignore files" and is silently omitted from commits.

**Files:** `.gitignore` line 123: `.planning/`, GSD planning operations

**Impact:** New phase plans, research summaries, and validations created by GSD workflows fail to commit. The check `gsd-sdk query commit` returns `committed: false` for these paths.

**Fix approach:** Use `git add -f .planning/<filename>` for all new files in `.planning/`. Consider removing `.planning/` from gitignore and letting the hub publish job manage cleanup, or use a prefix-pattern (e.g., gitignore only `.planning/*.tmp`).

### Version String Republish Defeats Freshness Check

**Issue:** v1.0.43 was published at least twice (2026-07-28, then again 2026-07-30 after phase-163 fixes) without bumping the version number. Downstream binary reporting "1.0.43, hub aligned" can be missing fixes that are inside v1.0.43. Only version agreement check runs; it silently passes when the binary is stale.

**Files:** Release tag logic, `aether version --check` command, `aether publish` workflow

**Impact:** Fixes to a release version go undeployed to users running downstream colonies. Version agreement is the only downstream freshness check, and same-number republish silently defeats it.

**Fix approach:** **Always bump version before `aether publish` when behavior changes.** When triaging downstream bugs, verify fix commits are ancestors of what was actually published, not just compare version strings.

## Security Considerations

### Permission Profile Write Guardrail Contradiction

**Issue:** Platform rules mark `.aether/data/` as protected ("never modify programmatically"), but Aether's own planning workers must write artifacts there (`planning/phase-plan.json`, `SCOUT.md`, iteration state). The permission profile says `repository_read_only` yet the worker brief orders writing phase-plan.json. Workers must work around the guardrail.

**Files:** `pkg/codex/permission_profile.go` line 96 (profile enforcement), `.claude/rules/aether-colony.md` (states protected paths), `.aether/docs/known-issues.md` line 52–68 (documented contradiction)

**Impact:** A guardrail that the system's own workers must evade trains workers to bypass security restrictions. Neither respected nor effective — worse than no guardrail.

**Fix approach:** Phase 163 scope (Context Reaches Workers). Carve out `.aether/data/planning/`, `.aether/data/phase-research/`, and `.aether/data/survey/` as declared-writable for workers, or route artifact writes through CLI. Update rules file to match actual system intent.

### Provider Availability Preflight Is Not Post-Launch Guarantee

**Issue:** Before worker launch, Aether checks whether a platform CLI exists and appears authenticated (categories: `binary_missing`, `auth_probe_failed`, `auth_inactive`, `invalid_auth_output`, `credentials_missing`, `probe_skipped`). This does not prove the later worker request will be accepted by provider API, account, proxy, or upstream.

**Files:** `pkg/codex/platform_dispatch.go` (provider preflight), `cmd/codex_build.go` (dispatch logic)

**Impact:** Post-launch provider/API/auth failures can look like worker parse failures if the provider returns error payload instead of worker claims JSON. Users may see generic "parse worker output: no JSON found" instead of a clear setup problem.

**Fix approach:** Classify post-launch provider/API/auth payloads before worker-result parsing. Surface sanitized provider problem description before generic JSON-parse errors.

### Multiline Pheromone Content Injection Risk

**Issue:** Pheromone signals are now sanitized (XML tags rejected, angle brackets escaped, shell injection patterns blocked, 500-char cap), but sanitization runs **after** prompt injection sanitization logic. Attack surface is reduced but not eliminated.

**Files:** `cmd/pheromone_cmds.go` (sanitization logic), `cmd/colony_prime_context.go` (injection into prompts)

**Impact:** LLM instruction override attempts are detected and rejected, but edge cases around escaping and multiline payloads may persist.

**Fix approach:** Run sanitization **before** content acceptance, not after prompts are composed. Add integration tests for common injection patterns (multiline "system:" prefixes, escaped quotes in bash contexts).

## Performance Bottlenecks

### Large Orchestrator Files Limit Readability and Testability

**Issue:** Core orchestrators are large monolithic files:
- `cmd/codex_continue.go` (3908 lines)
- `cmd/codex_build.go` (2888 lines)
- `cmd/oracle_loop.go` (3654 lines)
- `cmd/codex_visuals.go` (3857 lines)

**Files:** `cmd/codex_continue.go`, `cmd/codex_build.go`, `cmd/oracle_loop.go`, `cmd/codex_visuals.go`

**Impact:** Single-file changes require scanning thousands of lines. Testing individual functions requires understanding large initialization chains. Refactoring is risky; impacts are hard to trace.

**Fix approach:** Split large files by concern (e.g., `codex_continue_verification.go`, `codex_continue_gates.go`, `codex_visuals_caste.go`). Maintain single entry point but allow independent testing of sub-functions.

### No Deterministic Verification Command Fallback Is Silent

**Issue:** Verification step retrieval can fail to resolve a deterministic command. When no verification command exists for a phase, execution falls back silently. Workers then see no test command and cannot verify their work.

**Files:** `cmd/codex_continue.go` (verification resolution), verification step data structures

**Impact:** Phases with no explicit verification command get no verification. Watcher cannot assess work quality. Gate evaluation proceeds with missing data.

**Fix approach:** Make verification failure visible (warning in report, not silent fallback). If no deterministic command exists, surface to user before phase dispatch or require explicit watcher override.

## Fragile Areas

### Slash Command Documentation Parity Drifts

**Issue:** `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md` describe platform-specific UX but can drift from the Go CLI surface when commands change. No automated sync keeps mirrors aligned.

**Files:** `.claude/commands/ant/` (60 commands), `.opencode/commands/ant/` (60 commands), `cmd/` (Go implementations)

**Impact:** Users follow outdated command examples. Platform wrappers claim to expose options that no longer exist or miss new flags.

**Fix approach:** Treat Go runtime in `cmd/` and Codex guides (`AGENTS.md`, `.codex/CODEX.md`) as authoritative first. Run periodic (monthly) command-doc sweeps to audit mirror parity. Consider generating wrappers from YAML source (`aether/commands/*.yaml`) to reduce manual sync burden.

### Visual Output Depends on Terminal Mode

**Issue:** Caste colors, live previews, progress bars only render in visual/TTY mode (`codex_visuals.go`). Non-interactive terminals or JSON mode disable visuals silently.

**Files:** `cmd/codex_visuals.go` (rendering logic), environment checks (`AETHER_FORCE_VISUAL=1`)

**Impact:** Users in CI/scripts get no visual feedback. Formatting decisions (colors, emojis, progress) are hidden without explanation.

**Fix approach:** Log visual-mode decisions when `AETHER_FORCE_VISUAL` is set or terminal detection fails. Offer fallback ASCII-art mode for non-TTY environments.

### Worker Artifact Contracts Are Behavioral, Not Enforced

**Issue:** Build/plan workers are expected to follow artifact contracts (phase-plan.json format, survey structure, claims schema), but Aether cannot enforce contract adherence. If a worker ignores the contract, Aether falls back to local synthesis silently.

**Files:** `cmd/codex_plan.go` (plan artifact handling), `cmd/codex_build.go` (claims handling), `pkg/codex/dispatch.go` (worker result parsing)

**Impact:** Workers trained to expect enforcement but getting silent fallback. Plan/claims data may be stale synthesis rather than real worker output.

**Fix approach:** Make artifact contract violations visible: log actual vs. expected schema, emit warnings. Require explicit worker override to proceed with fallback synthesis.

## Scaling Limits

### Hive Wisdom Capped at 200 Entries

**Issue:** Cross-colony hive brain `wisdom.json` caps at 200 entries with LRU eviction. Large projects or long-running repositories quickly exceed this cap and lose older patterns.

**Files:** `pkg/memory/` (wisdom storage), `~/.aether/hive/wisdom.json` (200-entry cap)

**Impact:** As repository ages or colony count grows, general wisdom is evicted before being widely useful. Cross-colony pattern sharing degrades over time.

**Fix approach:** Consider tiered wisdom (hot cache 200, archive 1000+, searchable by tag/domain). Implement configurable cap and eviction strategy.

### Event Bus TTL May Drop Events During Long Phases

**Issue:** Event bus stores JSONL events with TTL cleanup. Long-running phases or delayed continue operations may lose events before consolidation runs.

**Files:** `pkg/events/` (event bus implementation), `cmd/codex_continue.go` (event usage during verification)

**Impact:** Learning and pattern observation may be incomplete if events expire before consolidation.

**Fix approach:** Extend TTL for in-flight phase events (do not clean up events from the current phase until it completes). Implement event retention policy tied to phase lifecycle.

## Dependencies at Risk

### TypeScript Host Coupling

**Issue:** Go runtime embeds TypeScript host assets (`//go:embed` in `embedded_assets.go:13`). TypeScript host holds the only playbook loader, confidence loop, and dashboard/swarm display. Deleting TS host breaks Go build.

**Files:** `cmd/embedded_assets.go` line 13, `.aether/ts-host/` (embedded), Go import dependencies

**Impact:** Tight coupling makes architecture changes risky. Removing or updating the TS host requires coordinating Go build changes. No way to test Go-only scenarios.

**Fix approach:** Separate embed from import; embed should be optional. Consider moving dashboard and playbook loading into Go for full runtime ownership.

### Playbook Execution Silently Fails

**Issue:** Playbook commands like `check-antipattern` (Gatekeeper security scan) redirect stderr to `/dev/null`. Execution failures are silent; no error propagates to verify gates.

**Files:** Playbook CLI invocations (pattern: `command 2>/dev/null`), `cmd/gate.go`

**Impact:** Security/quality gate can pass despite command failures. Users believe gate ran when execution failed.

**Fix approach:** Remove stderr redirect for production (`/dev/null` useful for dev but not for release). Capture and report stderr when gate commands fail. Test gate behavior with intentional command failures.

## Missing Critical Features

### No Automatic Feature Flagging

**Issue:** Feature capabilities (curation ants, hive wisdom, consolidation pipeline) are built but have no feature flags. System toggles between "capability fully disabled" (no caller) or "fully enabled" (when wired), with no gradual rollout path.

**Files:** Various — no feature flag infrastructure present

**Impact:** When consolidation is wired, all deployments immediately start running expensive pipeline. No way to roll out gradually or test at scale.

**Fix approach:** Implement feature flags for new capabilities. Allow opt-in testing before making features default.

### No Cross-Platform CLI Sync

**Issue:** `aether update` in other repos only syncs from the hub. No automated check ensures Claude/OpenCode command specs match current Go CLI flags or new options.

**Files:** `cmd/publish_cmd.go`, `aether update` logic

**Impact:** Platform-specific commands drift independently from Go CLI. Users on different platforms see different UX for the same logical operation.

**Fix approach:** Generate wrapper command specs from YAML source (`aether/commands/*.yaml`) instead of maintaining three separate mirrors. Regenerate on each publish.

## Test Coverage Gaps

### Playbook CLI Execution Not Tested End-to-End

**Issue:** Playbook commands are called from within verify gates and orchestrators but are not tested as executed commands. Only flag presence is validated, not actual behavior.

**Files:** `cmd/cli_flag_audit_test.go` (checks flag names only), playbook integration points

**Impact:** Broken playbooks only fail in production when a user's phase runs verification. No pre-release validation of command behavior.

**Fix approach:** Add integration tests that actually execute sample playbook commands. Verify both success and failure paths. Make test failures block release.

### Consolidation Pipeline Wiring Never Tested Live

**Issue:** The learning pipeline is built and unit-tested but has no end-to-end test that verifies it actually runs during a full `/ant-continue` → `/ant-seal` lifecycle.

**Files:** `cmd/doc_consolidation_claims_test.go` (documents this intentionally), `cmd/codex_continue.go`, `cmd/seal_cmd.go`

**Impact:** When consolidation wiring is added, no existing test will catch if integration was incomplete. Risk of another "machinery built but never called" incident.

**Fix approach:** Add explicit test verifying that consolidation-phase-end and consolidation-seal execute inside the continue/seal lifecycle (even if just a smoke test that they don't error).

### CI Oracle Heartbeat Test Is Flaky

**Issue:** Oracle deep-research loop has a CI heartbeat test that fails intermittently. Rerunning once often passes (test noise), but twice-in-a-row indicates real issue.

**Files:** Test related to oracle loop in CI

**Impact:** Flaky test reduces confidence in test suite. Developers learn to rerun flaky tests instead of investigating root cause.

**Fix approach:** Investigate the oracle loop timeout/resource contention. Increase timeout if legitimate, fix race condition if not. Lock with deterministic test that fails predictably.

---

*Concerns audit: 2026-08-01*
