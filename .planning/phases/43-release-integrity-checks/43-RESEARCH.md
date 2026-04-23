# Phase 43: Release Integrity Checks and Diagnostics - Research

**Researched:** 2026-04-23
**Domain:** Go CLI / Cobra command development, release pipeline validation, diagnostic integration
**Confidence:** HIGH

## Summary

Phase 43 is the capstone of the publish pipeline hardening work (Phases 40-42). It introduces a new first-class CLI command `aether integrity` that validates the entire release chain: source version, binary version, hub version, companion-file surfaces, and downstream update simulation. The command must work in both the Aether source repo and consumer repos (auto-detected), support both visual and JSON output modes, and integrate into `aether medic --deep`.

The implementation is straightforward because nearly all the building blocks already exist in the codebase. The stale-publish detection logic (`checkStalePublish`), version resolution (`resolveVersion`, `readHubVersion`), companion-file counting (`countEntriesInDir`), visual rendering (`renderBanner`, `renderStalePublishBanner`, `outputWorkflow`), and channel-aware pathing (`resolveHubPathForHome`) are all battle-tested from Phases 40-42. The primary work is composing these into a new Cobra command with context-aware branching and wiring the result into the medic deep-scan pipeline.

**Primary recommendation:** Implement `aether integrity` as a new `cmd/integrity_cmd.go` file with a single `runIntegrity` function that branches on `isSourceRepo()`. Reuse `checkStalePublish` for the downstream simulation check. Integrate into medic by calling `runIntegrity` from `performHealthScan` when `opts.Deep` is true. Test with the established E2E patterns from `cmd/e2e_stale_publish_test.go`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Version resolution | Go runtime (`cmd/root.go`) | — | Binary, repo, and hub versions are all read by Go functions |
| Companion-file counting | Go runtime (`cmd/update_cmd.go`) | — | `countEntriesInDir` and expected constants live in the runtime |
| Stale-publish classification | Go runtime (`cmd/update_cmd.go`) | — | `checkStalePublish` is the authoritative logic |
| Visual/JSON output | Go runtime (`cmd/codex_visuals.go`) | — | `outputWorkflow` and banner renderers are runtime-owned |
| Medic health scan | Go runtime (`cmd/medic_scanner.go`) | — | `performHealthScan` orchestrates all deep checks |
| Source vs consumer context | Go runtime (`cmd/integrity_cmd.go`) | — | Auto-detection is a runtime concern |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.22+ | Runtime language | Existing Aether binary is Go [VERIFIED: go.mod] |
| Cobra | v1.8.0 | CLI framework | Already used for all 80+ subcommands [VERIFIED: go.mod] |
| spf13/cobra/pflag | v1.0.5 | Flag parsing | Bundled with Cobra, used throughout cmd/ [VERIFIED: codebase grep] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | JSON output mode (`--json`) | Always — output envelope is JSON by default |
| path/filepath | stdlib | Cross-platform path construction | All hub/repo path resolution |
| os/exec | stdlib | Git tag resolution fallback in `resolveVersion` | When binary is built from source without ldflags |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Cobra | urfave/cli | Cobra is already the project's CLI framework; switching is out of scope |
| New `integrity` command | Extend `version --check` | CONTEXT.md D-01 explicitly rejected this — the pipeline deserves its own first-class command |
| Custom semver parser | blang/semver | `compareVersions` in `update_cmd.go` already handles the project's simple 3-segment semver needs; adding a dependency is unnecessary |

**Installation:** No new dependencies required. All functionality uses the existing Go module.

**Version verification:**
```bash
cd /Users/callumcowie/repos/Aether && go version
# go version go1.22.x darwin/arm64
cat go.mod | grep cobra
# github.com/spf13/cobra v1.8.0
```

## Architecture Patterns

### System Architecture Diagram

```
+----------------------------------+
|  User runs: aether integrity     |
+----------------------------------+
                |
                v
+----------------------------------+
|  Cobra: integrityCmd             |
|  Flags: --json, --channel,       |
|         --source                 |
+----------------------------------+
                |
                v
+----------------------------------+
|  runIntegrity(cmd, args)         |
|  1. Detect channel (binary/env)  |
|  2. Detect context (source vs    |
|     consumer repo)               |
+----------------------------------+
                |
    +-----------+-----------+
    |                       |
    v                       v
+---------------+   +------------------+
| Source Repo   |   | Consumer Repo    |
| (5 checks)    |   | (4 checks)       |
+---------------+   +------------------+
| 1. Source ver |   | 1. Binary ver    |
| 2. Binary ver |   | 2. Hub ver       |
| 3. Hub ver    |   | 3. Local files   |
| 4. Hub files  |   | 4. Downstream    |
| 5. Downstream |   |    simulation    |
+---------------+   +------------------+
                |
                v
+----------------------------------+
|  Aggregate results               |
|  - Classification (ok/info/      |
|    warning/critical)             |
|  - Recovery commands per failure |
+----------------------------------+
                |
    +-----------+-----------+
    |                       |
    v                       v
+---------------+   +------------------+
| Visual mode   |   | JSON mode        |
| (default)     |   | (--json flag)    |
+---------------+   +------------------+
| renderBanner  |   | integrityResult  |
| per-check     |   | -> JSON envelope |
| summary +     |   | with check list  |
| recovery cmds |   | and exit code    |
+---------------+   +------------------+
```

### Recommended Project Structure

```
cmd/
├── integrity_cmd.go          # NEW: aether integrity command
├── integrity_cmd_test.go     # NEW: unit + E2E tests
├── update_cmd.go             # EXISTING: checkStalePublish, countEntriesInDir
├── medic_scanner.go          # EXISTING: performHealthScan (integration point)
├── medic_wrapper.go          # EXISTING: scanHubPublishIntegrity
├── root.go                   # EXISTING: resolveVersion, readHubVersion
├── codex_visuals.go          # EXISTING: renderBanner, outputWorkflow
├── runtime_channel.go        # EXISTING: channel resolution
└── install_cmd.go            # EXISTING: isAetherSourceCheckout
```

### Pattern 1: Context-Aware Command Execution
**What:** The command auto-detects whether it is running in the Aether source repo or a consumer repo, then runs a different set of checks.
**When to use:** Any command that behaves differently in source vs consumer context.
**Example:**
```go
// Source: cmd/install_cmd.go (line 832)
func isAetherSourceCheckout(packageDir string) bool {
    root := findAetherModuleRoot(packageDir)
    if root == "" {
        return false
    }
    if _, err := os.Stat(filepath.Join(root, "cmd", "aether", "main.go")); err != nil {
        return false
    }
    return true
}
```
Adapted for integrity: check `cmd/aether/main.go` + `.aether/version.json` in the current working directory.

### Pattern 2: Reusable Stale-Publish Check
**What:** `checkStalePublish` performs version comparison and companion-file completeness classification.
**When to use:** Any code that needs to validate hub freshness against a binary version.
**Example:**
```go
// Source: cmd/update_cmd.go (lines 385-448)
func checkStalePublish(hubDir, hubVersion, binaryVersion string, channel runtimeChannel, syncDetails []map[string]interface{}) stalePublishResult {
    // Returns stalePublishResult with Classification, Message, Components, RecoveryCommand
}
```
For integrity, call this with the hub directory and versions; the `syncDetails` can be empty since we are only validating, not syncing.

### Pattern 3: Dual Output Mode (Visual vs JSON)
**What:** Commands check `AETHER_OUTPUT_MODE` env var or `--json` flag to decide between human-readable banners and structured JSON.
**When to use:** All user-facing commands that may run in CI.
**Example:**
```go
// Source: cmd/codex_visuals.go (lines 213-222)
func outputWorkflow(result interface{}, visual string) {
    if shouldRenderVisualOutput(stdout) {
        writeVisualOutput(stdout, visual)
        return
    }
    outputOK(result)
}
```

### Pattern 4: Medic Deep-Scan Integration
**What:** `performHealthScan` conditionally runs additional scanners when `opts.Deep` is true.
**When to use:** Adding new diagnostic checks to the medic command.
**Example:**
```go
// Source: cmd/medic_scanner.go (lines 166-170)
if opts.Deep {
    allIssues = append(allIssues, scanWrapperParity(fc)...)
    allIssues = append(allIssues, scanHubPublishIntegrity()...)
    allIssues = append(allIssues, scanCeremonyIntegrity(fc)...)
}
```
Add `scanIntegrity()` here that calls the integrity check logic and converts results to `[]HealthIssue`.

### Anti-Patterns to Avoid
- **Duplicating stale-check logic:** Do not rewrite version comparison or companion-file counting. Use `checkStalePublish` and `countEntriesInDir` directly.
- **Hard-coding hub paths:** Always use `resolveHubPathForHome(homeDir, channel)` — never string-construct `~/.aether` manually.
- **Suppressing errors in version reads:** If `readHubVersion` returns "unknown", treat it as a failure (info-level at minimum) rather than silently continuing.
- **Mixing visual and JSON output:** Never print raw text when `--json` is set. Route everything through `outputWorkflow` or equivalent.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Semver comparison | Custom parser beyond 3 segments | `compareVersions` in `update_cmd.go` | Already handles the project's versioning scheme; adding a library adds dependency weight for no gain |
| Companion-file counting | `filepath.Walk` or `os.ReadDir` loops | `countEntriesInDir` in `update_cmd.go` | Already tested, handles filters, returns 0 on missing dirs gracefully |
| Version resolution chain | Inline `os.ReadFile` + `exec.Command("git")` | `resolveVersion` in `root.go` | Encapsulates ldflags -> repo -> git tag -> hub -> fallback chain |
| JSON output envelope | Ad-hoc `json.Marshal` structs | `outputWorkflow` / `outputError` | Ensures consistent `ok`/`error`/`code` envelope shape across all commands |
| Source repo detection | Checking for `.git` only | `isAetherSourceCheckout` in `install_cmd.go` | Verifies go.mod contains the Aether module path AND `cmd/aether/main.go` exists |
| Visual banner rendering | `fmt.Printf` with manual ANSI | `renderBanner`, `renderStageMarker` | Consistent styling, emoji lookup, ANSI guard for non-TTY |

**Key insight:** This phase is almost entirely composition of existing, well-tested functions. The risk is not in building new complex logic but in wiring the pieces together correctly and testing the integration.

## Runtime State Inventory

> This phase does not involve rename, rebrand, or migration. No runtime state inventory required.

## Common Pitfalls

### Pitfall 1: Source Repo Detection Fails in Subdirectories
**What goes wrong:** `isAetherSourceCheckout` uses `findAetherModuleRoot` which walks up from the given directory. If the user runs `aether integrity` from a subdirectory of the Aether repo, `findAetherModuleRoot(os.Getwd())` will still find the repo root because it walks up to the go.mod. This is correct behavior.
**Why it happens:** The function is designed to work from any directory within the module tree.
**How to avoid:** Use `os.Getwd()` as the starting point for detection. Do not assume the user is at repo root.
**Warning signs:** Tests that only call `integrity` from repo root may pass while real usage from subdirectories fails.

### Pitfall 2: Channel Mismatch Between Binary and Hub
**What goes wrong:** The user has `aether-dev` binary but `~/.aether` (stable) hub, or vice versa. The integrity check must use the channel inferred from the binary name (or `--channel` flag) to select the correct hub path.
**Why it happens:** Dev and stable channels are isolated (Phase 41). Mixing them produces confusing version mismatches.
**How to avoid:** Always call `runtimeChannelFromFlag(cmd.Flags())` or `resolveRuntimeChannel()` before resolving the hub path. Never assume stable.
**Warning signs:** Integrity reports "hub version unknown" when the hub actually exists under a different channel directory.

### Pitfall 3: `checkStalePublish` Returns `staleOK` for Missing Hub
**What goes wrong:** If `hubVersion` is "unknown", `checkStalePublish` returns `staleInfo` (not critical), which may be misleading if the hub is completely missing.
**Why it happens:** The function is designed for update flows where the hub existence is already checked. In integrity, we need to surface missing hub as a distinct failure.
**How to avoid:** In `runIntegrity`, check `os.Stat(hubVersionFile)` before calling `checkStalePublish`. If missing, report a dedicated critical issue: "Hub not installed".
**Warning signs:** Integrity passes with info-level "unknown version" while the real problem is a missing installation.

### Pitfill 4: Medic Integration Produces Duplicate Issues
**What goes wrong:** `scanHubPublishIntegrity` in `medic_wrapper.go` already checks hub file counts. If the integrity check also checks hub counts, medic `--deep` may report the same problem twice.
**Why it happens:** Both scanners validate companion-file completeness.
**How to avoid:** The integrity check should focus on the **version chain** (source -> binary -> hub -> downstream) while `scanHubPublishIntegrity` focuses on **wrapper parity** (repo surfaces vs hub surfaces). In medic, call the integrity check for the chain and keep `scanHubPublishIntegrity` for wrapper parity. Deduplicate by category prefix ("integrity" vs "publish").
**Warning signs:** Medic output shows two critical issues with the same message for the same missing files.

### Pitfall 5: Exit Code Inconsistency with `--json`
**What goes wrong:** When `--json` is set, the command must still return the correct exit code (0, 1, or 2) even though visual output is suppressed.
**Why it happens:** `outputWorkflow` returns after writing JSON, so `runIntegrity` must explicitly `return fmt.Errorf(...)` or `return nil` to set the exit code.
**How to avoid:** Always `return` an error for exit code 1 or 2, and `return nil` for exit code 0. Do not rely on `outputWorkflow` to influence the exit code.
**Warning signs:** CI passes (exit 0) even though JSON output contains failed checks.

## Code Examples

### Integrity Result Struct
```go
// Pattern derived from stalePublishResult (cmd/update_cmd.go)
type integrityCheck struct {
    Name     string `json:"name"`
    Status   string `json:"status"`   // "pass", "fail", "skip"
    Message  string `json:"message"`
    Details  map[string]interface{} `json:"details,omitempty"`
}

type integrityResult struct {
    Context          string           `json:"context"`          // "source" or "consumer"
    Channel          string           `json:"channel"`
    Checks           []integrityCheck `json:"checks"`
    Overall          string           `json:"overall"`          // "ok", "warning", "critical"
    RecoveryCommands []string         `json:"recovery_commands,omitempty"`
}
```

### Context Detection
```go
// Source: adapted from isAetherSourceCheckout (cmd/install_cmd.go)
func detectIntegrityContext() string {
    cwd, err := os.Getwd()
    if err != nil {
        return "consumer"
    }
    root := findAetherModuleRoot(cwd)
    if root == "" {
        return "consumer"
    }
    if _, err := os.Stat(filepath.Join(root, "cmd", "aether", "main.go")); err != nil {
        return "consumer"
    }
    if _, err := os.Stat(filepath.Join(root, ".aether", "version.json")); err != nil {
        return "consumer"
    }
    return "source"
}
```

### Running the Downstream Simulation
```go
// Source: adapted from runUpdate (cmd/update_cmd.go)
func runDownstreamSimulation(hubDir, hubVersion, binaryVersion string, channel runtimeChannel) stalePublishResult {
    // Reuse the existing stale detection logic with empty sync details
    return checkStalePublish(hubDir, hubVersion, binaryVersion, channel, []map[string]interface{}{})
}
```

### Visual Integrity Report
```go
// Pattern derived from renderMedicReport and renderStalePublishBanner
func renderIntegrityVisual(result integrityResult) string {
    var b strings.Builder
    b.WriteString(renderBanner(commandEmoji("integrity"), "Release Integrity"))
    b.WriteString(visualDivider)
    b.WriteString(fmt.Sprintf("Context: %s repo\n", result.Context))
    b.WriteString(fmt.Sprintf("Channel: %s\n\n", result.Channel))

    for _, check := range result.Checks {
        icon := "✓"
        if check.Status != "pass" {
            icon = "✗"
        }
        b.WriteString(fmt.Sprintf("%s %s: %s\n", icon, check.Name, check.Status))
        if check.Message != "" {
            b.WriteString(fmt.Sprintf("  %s\n", check.Message))
        }
    }

    if len(result.RecoveryCommands) > 0 {
        b.WriteString("\nRecovery\n")
        for _, cmd := range result.RecoveryCommands {
            b.WriteString(fmt.Sprintf("  %s\n", cmd))
        }
    }
    return b.String()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `aether version --check` only compared binary vs hub | `aether integrity` validates full chain including source version and downstream simulation | Phase 43 (this phase) | Single command gives complete pipeline health picture |
| `aether medic --deep` only checked wrapper parity and hub surfaces | Also runs integrity chain validation | Phase 43 (this phase) | Medic now flags incomplete publishes with exact recovery commands |
| Stale publish detection only in `aether update` | Reusable `checkStalePublish` callable from integrity | Phase 42 (2026-04-23) | Logic is decoupled from sync flow and reusable for read-only checks |

**Deprecated/outdated:**
- `version --check` as a comprehensive release validator: it only compares binary and hub versions. It remains useful for quick checks but is superseded by `integrity` for pipeline validation.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `checkStalePublish` can be safely called with empty `syncDetails` for a read-only downstream simulation | Code Examples | If the function ever mutates `syncDetails` or expects non-nil, it could panic. Reviewed source — it only reads the slice. |
| A2 | `isAetherSourceCheckout` pattern (go.mod + cmd/aether/main.go) is sufficient to distinguish source repo from consumer repo | Architecture Patterns | A consumer repo that happens to have a go.mod with the Aether module path and a `cmd/aether/main.go` file would be misdetected. This is extremely unlikely in practice. |
| A3 | The expected companion-file counts (50/50/25/25/29) will not change during this phase | Standard Stack | If counts change, the integrity check will falsely report failures. Counts are stable as of v1.0.20. |

## Open Questions

1. **Should the integrity check verify the installed binary path?**
   - What we know: `resolveVersion` reads the binary version. The binary path is not currently tracked.
   - What's unclear: Whether knowing the binary path (e.g., `/usr/local/bin/aether`) adds diagnostic value.
   - Recommendation: Skip for initial implementation. Add if user feedback requests it.

2. **How should medic display integrity failures?**
   - What we know: Medic uses `HealthIssue` structs with severity/category/message/file/fixable fields.
   - What's unclear: Whether to map each integrity check to a separate `HealthIssue`, or aggregate them into one.
   - Recommendation: Map each failed check to a `HealthIssue` with category="integrity" so they appear alongside other deep-scan findings.

3. **Should `--source` flag be exposed in medic too?**
   - What we know: `aether integrity` has `--source` to force source-repo checks.
   - What's unclear: Whether `aether medic --deep --source` should also force source context.
   - Recommendation: No — medic runs in the current repo context. If the user wants source-repo integrity in medic, they can run `aether integrity --source` directly.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build + test | Yes | 1.22+ | — |
| Cobra | CLI framework | Yes | v1.8.0 | — |
| Git | `resolveVersion` git tag fallback | Yes | Any | Fallback to hub version or "0.0.0-dev" |
| Aether hub (`~/.aether`) | Hub version + companion-file checks | Assumed present | — | Report "not installed" if missing |
| Aether dev hub (`~/.aether-dev`) | Dev channel checks | Assumed present | — | Report "not installed" if missing |

**Missing dependencies with no fallback:**
- None — all core dependencies are compile-time.

**Missing dependencies with fallback:**
- Missing hub: integrity check reports critical failure with install instructions.
- Missing git: version resolution falls back to hub version or dev default.

## Validation Architecture

> `workflow.nyquist_validation` is absent from `.planning/config.json`; treating as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./cmd -run TestIntegrity -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REL-01 | Integrity check validates source, binary, hub, companion files, and downstream result | E2E | `go test ./cmd -run TestE2EIntegrity -v` | No — Wave 0 |
| REL-01 | Source repo context runs all 5 checks | Unit | `go test ./cmd -run TestIntegritySourceContext -v` | No — Wave 0 |
| REL-01 | Consumer repo context runs 4 checks | Unit | `go test ./cmd -run TestIntegrityConsumerContext -v` | No — Wave 0 |
| REL-01 | JSON output mode produces structured results | Unit | `go test ./cmd -run TestIntegrityJSONOutput -v` | No — Wave 0 |
| REL-01 | Exit code 0 when all checks pass | E2E | `go test ./cmd -run TestE2EIntegrityExitCode -v` | No — Wave 0 |
| REL-01 | Exit code 1 when any check fails | E2E | `go test ./cmd -run TestE2EIntegrityExitCodeFail -v` | No — Wave 0 |
| REL-02 | Medic --deep includes integrity findings | Unit | `go test ./cmd -run TestMedicDeepIncludesIntegrity -v` | No — Wave 0 |
| REL-02 | Integrity failures in medic include recovery commands | Unit | `go test ./cmd -run TestMedicIntegrityRecovery -v` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd -run TestIntegrity -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/integrity_cmd_test.go` — does not exist; must be created
- [ ] `cmd/integrity_cmd.go` — does not exist; must be created
- [ ] `performHealthScan` integration — no `scanIntegrity()` function exists yet

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | N/A — no auth in this phase |
| V3 Session Management | No | N/A — no sessions |
| V4 Access Control | No | N/A — local CLI only |
| V5 Input Validation | Yes | Path inputs (`--channel`, cwd) must not traverse outside intended directories. Use `filepath.Join` with validated channel names only (`stable`, `dev`). |
| V6 Cryptography | No | N/A — no crypto operations |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via `--channel` flag | Tampering | Whitelist channel to `channelStable`/`channelDev`; reject unknown values |
| Information disclosure via JSON output | Information Disclosure | JSON mode is intentional for CI; no sensitive data (tokens, keys) is included in integrity output |
| Command injection in recovery commands | Tampering | Recovery commands are hardcoded strings, not constructed from user input |

## Sources

### Primary (HIGH confidence)
- `cmd/update_cmd.go` — `checkStalePublish`, `stalePublishResult`, `countEntriesInDir`, `compareVersions`, expected count constants
- `cmd/root.go` — `resolveVersion`, `readInstalledHubVersion`, `readRepoVersion`, `normalizeVersion`, `findAetherModuleRoot`
- `cmd/medic_scanner.go` — `performHealthScan`, `HealthIssue`, `ScannerResult`, deep-scan integration pattern
- `cmd/medic_wrapper.go` — `scanHubPublishIntegrity`, `scanWrapperParity`, expected count constants
- `cmd/medic_cmd.go` — `MedicOptions`, `runMedic`, flag wiring, JSON/visual rendering
- `cmd/codex_visuals.go` — `renderBanner`, `renderStalePublishBanner`, `outputWorkflow`, `renderStageMarker`
- `cmd/runtime_channel.go` — `runtimeChannel`, `resolveHubPathForHome`, `runtimeChannelFromFlag`
- `cmd/install_cmd.go` — `isAetherSourceCheckout`, `findAetherModuleRoot`
- `cmd/version.go` — `versionCmd`, `--check` flag behavior
- `cmd/e2e_stale_publish_test.go` — E2E test patterns for stale detection
- `cmd/update_cmd_test.go` — `createHubWithExpectedCounts` helper, test setup patterns
- `cmd/medic_cmd_test.go` — Medic command test patterns
- `.planning/phases/43-release-integrity-checks/43-CONTEXT.md` — All locked decisions (D-01 through D-15)
- `.planning/REQUIREMENTS.md` — REL-01 (R062), REL-02 (R063)

### Secondary (MEDIUM confidence)
- `AETHER-OPERATIONS-GUIDE.md` — Section 10 (Exact Verification Commands) for expected counts
- `.aether/docs/publish-update-runbook.md` — Publish/update workflow context

### Tertiary (LOW confidence)
- None — all claims verified against codebase or CONTEXT.md

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are already in use, versions verified from go.mod
- Architecture: HIGH — patterns are established and reusable from existing commands
- Pitfalls: HIGH — derived from direct code review and prior phase experience (40-42)

**Research date:** 2026-04-23
**Valid until:** 2026-05-23 (stable stack, low churn expected)
