# Phase 46: Stuck-Plan Investigation and Release Decision - Research

**Researched:** 2026-04-24
**Domain:** Go CLI runtime, colony planning pipeline, release integrity
**Confidence:** HIGH

## Summary

This phase has two deliverables: (1) investigate whether `aether plan` still hangs in freshly updated downstream repos, and (2) run a full milestone audit and make the v1.6 release decision.

The stuck-plan investigation is narrow in scope. D-01 says to test only in a freshly updated downstream repo. The code analysis reveals several plausible "stuck" failure modes, but the most likely root cause was stale hub state (binary v1.0.20 with hub v1.0.19) which would cause version-dependent logic in `resolveVersion()` to return different values across calls, potentially creating inconsistent state that blocks plan execution. Phases 40-43 pipeline hardening (atomic publish, stale publish detection, integrity checks) should have resolved this class of problem.

The milestone audit follows the established `/gsd-audit-milestone` workflow: read all VERIFICATION.md files from Phases 39-46, cross-reference REQUIREMENTS.md, check requirements coverage, and produce `v1.6-MILESTONE-AUDIT.md`. The v1.5 release decision (Phase 36) provides a clear template for how release decisions work in this project.

**Primary recommendation:** Structure this phase as two sequential tasks -- stuck-plan reproduction first (quick, either fixes a bug or documents it as stale), then milestone audit (systematic, reads all VERIFICATION.md files).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Test stuck `aether plan` in a freshly updated downstream repo only. If it works, document the issue as stale-install fallout resolved by Phases 40-43 pipeline hardening. No need to test in the original problematic repo.
- **D-02:** Run a full milestone audit (`/gsd-audit-milestone`) before shipping v1.6. Standard checks (go tests pass, version agreement via `aether version --check`, E2E regression tests pass) plus all phases reviewed against original intent.

### Claude's Discretion
- Exact steps for stuck-plan reproduction (which commands to run, what output to check)
- How to structure the milestone audit report
- Whether to include a version bump commit as part of this phase

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EVD-02 (R067) | Verify whether stuck `aether plan` issue still reproduces in freshly updated stable and dev repos; if yes, fix with regression test | Plan command flow fully mapped; reproduction procedure identified; fallback logic documented |

All other v1.6 requirements (PUB-01 through REL-04) are assigned to prior phases. Phase 46's job is to confirm EVD-02 and audit the milestone as a whole.
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Plan execution (stuck investigation) | API / Backend (Go CLI runtime) | -- | `aether plan` is a Go command that reads colony state, dispatches workers, and synthesizes planning artifacts. Entirely server-side. |
| Worker dispatch | API / Backend (Go CLI runtime) | -- | `dispatchRealPlanningWorkersWithTimeout` invokes platform workers via the codex invoker, with timeout and fallback logic. |
| Version resolution | API / Backend (Go CLI runtime) | -- | `resolveVersion()` reads ldflags, then `.aether/version.json`, then `git describe --tags`. |
| Milestone audit | Documentation / Verification | -- | Reads VERIFICATION.md files, cross-references REQUIREMENTS.md, produces audit report. No runtime code changes. |
| Release decision | Documentation / Verification | -- | Version bump, changelog, tag -- same pattern as Phase 36. |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go (stdlib) | 1.26.1 | Runtime, testing, CLI | Project is Go-native; cobra for CLI, testing package for tests |
| Cobra | v1.9.0 | CLI framework | Used for all `aether` subcommands |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gsd-sdk | installed | Milestone audit orchestration | Used by `/gsd-audit-milestone` to query phase data, find phase directories, and run integration checks |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `/gsd-audit-milestone` | Manual audit (read each VERIFICATION.md) | GSD audit is automated and follows the project's 3-source cross-reference protocol. Manual audit misses the structured format and integration checker. Use GSD. |

**Installation:**
```bash
# No new packages needed -- this is a test-and-audit phase
go test ./... -count=1
```

**Version verification:** Go 1.26.1 installed and confirmed. No new dependencies required.

## Architecture Patterns

### System Architecture Diagram

```
Stuck-Plan Investigation:
==========================

  Fresh Downstream Repo
  ┌─────────────────────────┐
  │ 1. aether update --force │ ──> Syncs companion files from hub
  │ 2. aether init "test goal"│ ──> Creates COLONY_STATE.json
  │ 3. aether plan            │ ──> Reads state, dispatches workers, synthesizes plan
  └─────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────┐
  │              runCodexPlanWithOptions      │
  │                                           │
  │  loadActiveColonyState() ──> COLONY_STATE │
  │       │                                   │
  │       ├── store == nil? ──> ERROR          │
  │       ├── no COLONY_STATE? ──> ERROR       │
  │       ├── empty goal? ──> ERROR            │
  │       │                                   │
  │       ▼                                   │
  │  existing plan? (no --refresh) ──> return  │
  │       │                                   │
  │       ▼                                   │
  │  beginRuntimeSpawnRun()                    │
  │  loadCodexSurveyContext()                  │
  │  create planning directories               │
  │       │                                   │
  │       ▼                                   │
  │  newCodexWorkerInvoker()                   │
  │       │                                   │
  │       ├── FakeInvoker ──> synthetic plan   │
  │       ├── unavailable ──> fallback plan    │
  │       ├── available ──> real dispatch      │
  │       │        │                          │
  │       │        ▼                          │
  │       │  dispatchRealPlanningWorkersWithTimeout()
  │       │    (15m timeout per worker)        │
  │       │    ├── scout wave 1                │
  │       │    └── route_setter wave 2         │
  │       │        │                          │
  │       │    timeout/error? ──> fallback     │
  │       │                                   │
  │       ▼                                   │
  │  Synthesize planning artifacts            │
  │  Save COLONY_STATE.json                    │
  │  Return plan JSON                          │
  └───────────────────────────────────────────┘


Milestone Audit:
=================

  /gsd-audit-milestone
         │
         ▼
  ┌──────────────────────────┐
  │ 1. List all v1.6 phases  │
  │    (39-46, including     │
  │     44.1, 44.2)          │
  └──────────────────────────┘
         │
         ▼
  ┌──────────────────────────┐
  │ 2. Read each VERIF.md    │
  │    Extract: status,      │
  │    gaps, requirements    │
  └──────────────────────────┘
         │
         ▼
  ┌──────────────────────────┐
  │ 3. Spawn integration     │
  │    checker (gsd subagent)│
  └──────────────────────────┘
         │
         ▼
  ┌──────────────────────────┐
  │ 4. 3-source cross-ref   │
  │    VERIF + SUMMARY +     │
  │    REQUIREMENTS.md       │
  └──────────────────────────┘
         │
         ▼
  ┌──────────────────────────┐
  │ 5. v1.6-MILESTONE-AUDIT │
  └──────────────────────────┘
```

### Recommended Task Structure

```
Task 1: Reproduce stuck `aether plan` in fresh downstream repo
  - Create temp directory
  - Run `aether update --force` (uses current hub)
  - Run `aether init "test stuck plan investigation"`
  - Run `aether plan` with a timeout (e.g., 60s)
  - Check: does it produce output? Does it return an error? Does it hang?
  - If it works: document as "stale-install fallout resolved by pipeline hardening"
  - If it hangs: diagnose the blocking point (likely in worker dispatch or version resolution)

Task 2: Milestone audit and release decision
  - Run `/gsd-audit-milestone` for v1.6
  - Review audit results, address any gaps_found
  - Run standard checks: `go test ./...`, `aether version --check`, E2E regression tests
  - Update version files, changelog, and tag (if shipping v1.6)
  - Produce VERIFICATION.md for Phase 46
```

### Pattern 1: Stuck-Plan Reproduction
**What:** Create a clean downstream repo, update from hub, init a colony, and run plan. This tests the full downstream flow that users experience.
**When to use:** Whenever investigating a reported runtime issue in downstream repos.
**Example:**
```bash
# In a fresh temp directory
TMPDIR=$(mktemp -d)
cd "$TMPDIR"
aether update --force          # Sync from hub (must work first)
aether init "test goal"        # Create colony state
timeout 60 aether plan         # Should produce plan JSON within 60s
echo $?                        # 0 = success, non-zero = error, 124 = timeout (stuck)
```

### Pattern 2: Milestone Audit (from Phase 36 template)
**What:** Read all VERIFICATION.md files, cross-reference requirements, produce audit report.
**When to use:** Final phase of any milestone.
**Example:** See `.planning/phases/36-release-decision/36-VERIFICATION.md` for the v1.5 release decision pattern.

### Anti-Patterns to Avoid
- **Testing in the Aether source repo:** D-01 explicitly says to test in a downstream repo only. The source repo has development state that masks downstream issues.
- **Skipping the milestone audit:** D-02 requires a full audit. The audit catches orphaned requirements and cross-phase integration gaps that individual phase verifications miss.
- **Running plan without init:** `aether plan` requires a colony to be initialized first. Running it without `aether init` will produce "no colony initialized" error, not a hang. This is expected behavior, not the bug.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Milestone audit workflow | Manual VERIFICATION.md parsing and cross-referencing | `/gsd-audit-milestone` | The GSD audit workflow handles 3-source cross-reference, integration checking, orphan detection, and produces structured output automatically |
| Stuck-plan reproduction framework | Custom shell script with timeouts | `timeout` command + `aether` CLI | Simple timeout + CLI invocation is sufficient. No framework needed for a single reproduction test. |
| Version agreement check | Custom script comparing version files | `aether version --check` | Built-in command that exits 0 on match, non-zero on mismatch |

**Key insight:** This phase is primarily an investigation and audit phase. The only code change that might be needed is a fix if `aether plan` is still stuck. Everything else is verification and documentation.

## Common Pitfalls

### Pitfall 1: Confusing "no output" with "stuck"
**What goes wrong:** `aether plan` produces JSON output to stdout. If stdout is captured but stderr shows an error, the test might think it hung when it actually failed fast.
**Why it happens:** The plan command returns structured JSON on success and error messages on stderr on failure. Both paths terminate quickly.
**How to avoid:** Check both stdout and stderr. Use `timeout` with a reasonable limit (60s). The plan command should complete in under 5 seconds in synthetic/fallback mode.
**Warning signs:** If `timeout 60 aether plan` exits with code 124, that is a genuine hang. If it exits with code 1 and has stderr output, that is a fast failure (not the bug).

### Pitfall 2: Testing in source repo instead of downstream
**What goes wrong:** Running `aether plan` in the Aether source repo will work because it has development state, COLONY_STATE.json, and a local `.aether/` directory. The bug only manifests in downstream repos.
**Why it happens:** Source repos always have up-to-date companion files. Downstream repos depend on hub sync.
**How to avoid:** Always test in a fresh temp directory, as D-01 specifies.

### Pitfall 3: Assuming `aether plan` hangs on all platforms
**What goes wrong:** The plan command uses `SelectPlatformInvoker()` which checks for Claude Code, OpenCode, or Codex CLI availability. If none are available, it falls back to `FakeInvoker` which completes instantly.
**Why it happens:** The "stuck" behavior requires a real platform invoker to be available AND to hang. In a fresh repo with no platform configured, the fallback should kick in immediately.
**How to avoid:** Understand that "stuck" only happens when a real invoker is selected but blocks. In a test environment with no AETHER_WORKER_PLATFORM or AETHER_REAL_DISPATCH set, the plan uses FakeInvoker and cannot hang.

### Pitfall 4: Missing VERIFICATION.md for in-progress phases
**What goes wrong:** The milestone audit reads VERIFICATION.md from all phases. Phases 39, 42, 44, 44.1, and 46 itself may not have VERIFICATION.md files yet.
**Why it happens:** Phases get VERIFICATION.md after they are executed and verified. Some v1.6 phases are still in progress.
**How to avoid:** The audit workflow handles missing VERIFICATION.md by flagging the phase as "unverified" -- this is expected for Phase 46 (the phase being audited). Check which phases actually have VERIFICATION.md before running the audit.

## Code Examples

### How `aether plan` decides its execution path

```go
// Source: cmd/codex_plan.go lines 106-150
func runCodexPlanWithOptions(root string, opts codexPlanOptions) (map[string]interface{}, error) {
    // 1. Check store is initialized
    if store == nil {
        return nil, fmt.Errorf("no store initialized")
    }

    // 2. Load colony state (fast: reads COLONY_STATE.json)
    state, err := loadActiveColonyState()
    if err != nil {
        return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
    }

    // 3. If plan already exists and not refreshing, return existing plan
    if len(state.Plan.Phases) > 0 && !opts.Refresh {
        // Fast return -- no worker dispatch
        return map[string]interface{}{"planned": true, "existing_plan": true, ...}, nil
    }

    // 4. Otherwise, proceed with planning...
    // This is where the "stuck" could occur if worker dispatch hangs
    if !opts.Synthetic {
        invoker := newCodexWorkerInvoker()
        // ... dispatch real workers with 15m timeout per worker ...
    }
}
```

### How `NewWorkerInvoker` decides which invoker to use

```go
// Source: pkg/codex/worker.go lines 620-634
func NewWorkerInvoker() WorkerInvoker {
    switch strings.ToLower(strings.TrimSpace(os.Getenv(envRealDispatch))) {
    case "0", "false", "fake":
        return &FakeInvoker{}         // Explicitly fake
    case "1", "true", "real":
        return SelectPlatformInvoker(context.Background())  // Explicitly real
    }
    if normalizePlatform(os.Getenv(envWorkerPlatform)) == PlatformFake {
        return &FakeInvoker{}         // Platform env says fake
    }
    if runningInGoTest() {
        return &FakeInvoker{}         // Test environment
    }
    return SelectPlatformInvoker(context.Background())  // Default: try real platform
}
```

### How `resolveVersion` works (the version chain)

```go
// Source: cmd/root.go lines 28-58
func resolveVersion(dir ...string) string {
    // Priority 1: ldflags version (set at build time)
    if Version != "0.0.0-dev" {
        return normalizeVersion(Version)
    }

    // Priority 2: .aether/version.json in repo root
    if repoVersion := readRepoVersion(gitDir); repoVersion != "" {
        return repoVersion
    }

    // Priority 3: git describe --tags
    out, err := exec.Command("git", args...).Output()
    // ...

    // Priority 4: fallback to "0.0.0-dev"
    return "0.0.0-dev"
}
```

### Downstream repo test pattern (from E2E tests)

```go
// Source: cmd/e2e_regression_test.go lines 14-73 (pattern)
// Step 1: Create mock source checkout
sourceDir := createMockSourceCheckout(t, "1.0.99-test")

// Step 2: Publish to stable hub (or use existing hub)
rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary"})
rootCmd.Execute()

// Step 3: Create downstream repo
repoDir := t.TempDir()
os.Chdir(repoDir)

// Step 4: Update from hub
rootCmd.SetArgs([]string{"update", "--force"})
rootCmd.Execute()

// Step 5: Verify downstream files exist
repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
os.Stat(repoWorkers) // should exist
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `aether install --package-dir "$PWD"` | `aether publish` | Phase 40 (v1.6) | `publish` includes atomic version agreement verification. `install` still works but without verification. |
| No stale publish detection | `aether update --force` blocks on stale hub | Phase 42 (v1.6) | Downstream repos now get actionable error when hub is behind binary. |
| No integrity checking | `aether integrity` validates full chain | Phase 43 (v1.6) | Single command checks source -> binary -> hub -> downstream. |
| No E2E pipeline tests | 4 E2E regression tests | Phase 45 (v1.6) | Stable/dev publish/update, stale detection, channel isolation all tested. |

**Known v1.6 requirements status (from REQUIREMENTS.md):**
- PUB-01 (R059): Pending (Phase 40 marked COMPLETE in ROADMAP but checkbox not ticked in REQUIREMENTS.md)
- PUB-02 (R060): Pending (Phase 41 marked COMPLETE in ROADMAP but checkbox not ticked in REQUIREMENTS.md)
- PUB-03 (R061): Complete (Phase 44.2)
- PUB-04 (R061): Pending (Phase 42)
- REL-01 (R062): Complete (Phase 44.2)
- REL-02 (R063): Complete (Phase 43)
- REL-03 (R064): Pending (Phase 44)
- REL-04 (R065): Pending (Phase 45 -- tests exist but REQUIREMENTS.md not updated)
- EVD-01 (R066): Pending (Phase 44)
- EVD-02 (R067): Pending (Phase 46 -- this phase)
- OPN-01 (R068): Pending (Phase 39)

**Note:** REQUIREMENTS.md checkboxes appear stale. Many phases are marked COMPLETE in ROADMAP.md but their checkboxes in REQUIREMENTS.md are still unchecked. The milestone audit (3-source cross-reference) should catch this.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The stuck `aether plan` issue was caused by stale hub state (binary v1.0.20, hub v1.0.19), not a code bug | Summary, Pattern 1 | If wrong, the issue may still reproduce and require a code fix + regression test |
| A2 | In a fresh downstream repo with no AETHER_WORKER_PLATFORM or AETHER_REAL_DISPATCH env vars, `aether plan` uses FakeInvoker and cannot hang | Common Pitfalls | If wrong, the reproduction test design needs adjustment |
| A3 | Phases 39, 42, 44, 44.1 have not been executed yet (no VERIFICATION.md files) | Common Pitfalls 4 | If they have been executed, the audit scope changes |

## Open Questions

1. **Which phases actually have VERIFICATION.md files?**
   - What we know: Phase 40, 41, 43, 44.2, 45 all have VERIFICATION.md. Phases 39, 42, 44, 44.1 do not.
   - What's unclear: Whether Phases 39, 42, 44, 44.1 were executed but just not verified, or never executed at all.
   - Recommendation: Before running the milestone audit, explicitly check which phases are missing VERIFICATION.md. Missing VERIFICATION.md = unverified phase = potential blocker for audit.

2. **Will the stuck-plan reproduction use the real aether binary or `go run`?**
   - What we know: `aether update --force` requires an installed binary. `aether init` and `aether plan` also require it.
   - What's unclear: Whether the current installed binary is fresh enough to include all v1.6 fixes.
   - Recommendation: Build a fresh binary first (`go build ./cmd/aether`), install it, then run the reproduction test.

3. **What version will v1.6 ship as?**
   - What we know: Current source version is 1.0.20. v1.6 is a milestone label, not a product version.
   - What's unclear: Whether v1.6 ships as v1.0.21 or stays at v1.0.20.
   - Recommendation: This is a Claude's Discretion item. Recommend bumping to v1.0.21 to distinguish from the v1.5 release (which was also v1.0.20).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Binary build, tests | Yes | 1.26.1 | -- |
| gsd-sdk | Milestone audit | Yes | installed | Manual audit |
| aether binary | Reproduction test | Yes | v1.0.20 (source) | `go build ./cmd/aether` |
| timeout | Reproduction test | Yes | macOS built-in | -- |

**Missing dependencies with no fallback:**
- None.

**Missing dependencies with fallback:**
- None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing package |
| Config file | none |
| Quick run command | `go test ./cmd/ -run "TestE2ERegression|TestPlan" -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EVD-02 (R067) | aether plan works in fresh downstream repo | manual + unit | `timeout 60 aether plan` (in fresh repo) | No -- Wave 0 if fix needed |
| EVD-02 (R067) | aether plan produces valid JSON output | unit | `go test ./cmd/ -run "TestPlan" -count=1` | Yes -- `cmd/codex_plan_test.go` |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestE2ERegression|TestPlan" -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- None for existing tests. If `aether plan` is reproducibly stuck, a new regression test will be needed (the reproduction test itself becomes the regression test).

## Security Domain

> This phase has no security-relevant changes. It is an investigation and audit phase. No new attack surfaces are introduced. If a code fix is needed for the stuck-plan bug, it would be in the planning dispatch path which already has timeout guards (15m per worker).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | no | -- |
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V6 Cryptography | no | -- |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_plan.go` -- Full plan command flow, 1565 lines, read in full
- `cmd/codex_plan_test.go` -- 934 lines, 15 test functions covering plan behavior
- `cmd/e2e_regression_test.go` -- 4 E2E tests for publish/update pipeline
- `cmd/publish_cmd.go` -- Publish command with version verification
- `cmd/integrity_cmd.go` -- Full integrity chain validation
- `cmd/version.go` -- Version check command
- `cmd/dispatch_runtime.go` -- Worker dispatch with timeout
- `cmd/codex_dispatch_contract.go` -- Timeout constants (15m per planning worker)
- `pkg/codex/worker.go` -- Worker invoker selection logic
- `cmd/state_load.go` -- Colony state loading
- `cmd/root.go` -- Version resolution chain
- `.aether/version.json` -- Current source version: 1.0.20
- `.planning/REQUIREMENTS.md` -- v1.6 requirements and traceability
- `.planning/ROADMAP.md` -- v1.6 phase status
- `.planning/phases/36-release-decision/36-VERIFICATION.md` -- v1.5 release decision template
- `.planning/phases/45-e2e-regression-coverage/45-VERIFICATION.md` -- Most recent verification
- `.aether/docs/publish-update-runbook.md` -- Publish/update operational guide
- `/gsd-audit-milestone` workflow -- Milestone audit procedure

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` -- Project decisions and history (note: some entries may be stale)
- `.planning/config.json` -- No nyquist_validation override (absent = enabled)

### Tertiary (LOW confidence)
- None -- all findings verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Go 1.26.1 verified, no new dependencies needed
- Architecture: HIGH - Plan command flow fully read and traced from source code
- Pitfalls: HIGH - All failure modes identified from code analysis

**Research date:** 2026-04-24
**Valid until:** 30 days (stable codebase, investigation/audit phase)
