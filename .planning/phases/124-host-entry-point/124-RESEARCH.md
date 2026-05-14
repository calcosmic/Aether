# Phase 124: Host Entry Point — Research

**Researched:** 2026-05-14
**Domain:** Go CLI (Cobra) + TypeScript host integration + publish/update distribution
**Confidence:** HIGH

## Summary

The TypeScript orchestration host at `.aether/ts-host/` is fully built and functional, but there is **no Go CLI entry point** to invoke it. The `aether host <workflow>` command does not exist. Publish and update flows do not distribute TS host assets. This phase must bridge that gap.

The TS host already implements `plan`, `build`, `continue`, and `lifecycle` commands that call Go `--plan-only` manifests and finalizers. The Go side needs a `host` command that discovers Node, validates the built TS host exists, and delegates with inherited TTY. Publish must build and copy TS assets to the hub. Update must sync them to downstream repos and ensure `npm ci` + `npm run build` run when needed.

**Primary recommendation:** Add `cmd/host_cmd.go` with Cobra subcommands, extend publish/update to treat `ts-host/` as a first-class companion asset, and add clear fallback messaging for missing Node or unbuilt assets.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI entry point (`aether host`) | Go CLI (API/Backend) | — | Go owns all CLI commands and subcommand routing |
| Workflow orchestration (plan/build/continue/lifecycle) | TypeScript Host | Go (manifests/finalizers) | TS host dispatches workers; Go owns state mutation |
| Asset distribution (publish/update) | Go CLI | — | Go owns install/update/publish hub sync |
| Node/dependency detection | Go CLI | — | Go validates environment before spawning TS |
| Fallback messaging | Go CLI | — | Go surfaces errors when TS host cannot run |
| Oracle workflow stub | TypeScript Host | Go (iteration commands in Phase 126) | TS host will orchestrate; Go will own state |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Cobra | v1.8.0 (via go.mod) | CLI command framework | Already used for all 80+ `aether` subcommands [VERIFIED: cmd/root.go] |
| Go stdlib (`os/exec`) | 1.22 | Subprocess invocation | No external dependency needed for Node spawn |
| Node.js | >= 20 | TS host runtime | Specified in `.aether/ts-host/package.json` engines [VERIFIED: package.json] |
| TypeScript | 5.9.3 | TS host compilation | Specified in `devDependencies` [VERIFIED: package.json] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `npm ci` | bundled with npm | Dependency install | During publish and update when `node_modules` is missing |
| `tsc` | 5.9.3 | Build `dist/` from `src/` | During publish and update when `dist/` is missing |

### Installation (Node side)
```bash
# In .aether/ts-host/
npm ci
npm run build
```

**Version verification:**
- Node: `v25.9.0` installed on this machine (exceeds >= 20 requirement) [VERIFIED: `node --version`]
- npm: `11.12.1` [VERIFIED: `npm --version`]
- TypeScript: `5.9.3` in package.json [VERIFIED: `npm view typescript version` matches]

## Architecture Patterns

### System Architecture Diagram

```
User runs: aether host lifecycle
                |
                v
        +---------------+
        |  Go CLI host  |
        |   cmd (Cobra) |
        +---------------+
                |
    +-----------+-----------+
    |                       |
    v                       v
+--------+          +------------------+
| Node   |          | Fallback message |
| found? |--NO----> | (stderr + exit 1)|
+--------+          +------------------+
    | YES
    v
+--------+          +------------------+
| dist/  |          | Build hint       |
| found? |--NO----> | (src/ exists?)   |
+--------+          +------------------+
    | YES
    v
+---------------------------------------+
| exec.Command(node, host.js, lifecycle)|
| stdin/stdout/stderr inherited         |
+---------------------------------------+
                |
                v
        +---------------+
        | TS host       |
        | host.ts       |
        +---------------+
                |
    +-----------+-----------+
    |                       |
    v                       v
+--------+          +------------------+
| Go CLI |          | Worker dispatch  |
| --plan |<-------->| (platform spawn) |
| -only  |          |                  |
+--------+          +------------------+
    |
    v
+--------+          +------------------+
| Go     |          | TS writes        |
|finalizer|<--------| completion file  |
+--------+          +------------------+
```

### Recommended Project Structure (Go side)
```
cmd/
├── host_cmd.go          # NEW: aether host <workflow>
├── host_cmd_test.go     # NEW: tests for host command
├── publish_cmd.go       # MODIFY: build + sync ts-host assets
├── update_cmd.go        # MODIFY: sync ts-host from hub + build
└── ...
```

### Pattern 1: Node Discovery and Version Validation
**What:** Check PATH for `node`, parse `--version`, enforce minimum version.
**When to use:** Before any TS host subprocess invocation.
**Example:**
```go
// Source: cmd/narrator_launcher.go (existing pattern)
nodePath, err := exec.LookPath("node")
if err != nil {
    return nil // fallback: no narrator
}
```

### Pattern 2: Subprocess Delegation with TTY Inheritance
**What:** Run Node process with inherited stdin/stdout/stderr so dashboard and interactive output work.
**When to use:** All `aether host <workflow>` subcommands.
**Example:**
```go
// Source: cmd/narrator_launcher.go (existing pattern)
cmd := exec.CommandContext(ctx, nodePath, runtimePath, "--visuals", visualPath)
stdin, _ := cmd.StdinPipe()
// ... pipes set up, then cmd.Start()
```

### Pattern 3: Hub Asset Sync
**What:** Copy directory trees from source checkout to `~/.aether/system/` during publish, and from hub to repo during update.
**When to use:** Publish and update flows for TS host assets.
**Example:**
```go
// Source: cmd/install_cmd.go (existing pattern)
hubSyncResult := syncDirToHub(srcAether, systemDir)
```

### Anti-Patterns to Avoid
- **Blocking publish on TS host build failure:** TS host is best-effort; Go binary and companion files are the critical path. Warn, don't fail.
- **Parsing TS host visual output:** The TS host may emit ANSI. Go must never parse it. Use JSON mode for programmatic communication.
- **Writing to `.aether/data/` from TS host directly:** Already enforced by boundary contract. All state through Go finalizers.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Node version parsing | Regex on `node --version` | `semver` package or simple `strings.TrimPrefix` + `strconv.Atoi` | Node version format is stable (`vX.Y.Z`) |
| Directory tree copy | Custom walk + copy | `syncDirToHub` / `syncDir` from `cmd/install_cmd.go` | Already tested, handles exclusions, validation, cleanup |
| Subprocess exit code forwarding | Manual `Wait` + `ExitError` inspection | `cmd.Run()` + return `err` directly | Cobra already handles `RunE` error propagation |

## Runtime State Inventory

This phase is greenfield (new command + asset distribution). No rename/refactor/migration of stored data.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — new command, no data migration | None |
| Live service config | None | None |
| OS-registered state | None | None |
| Secrets/env vars | None | None |
| Build artifacts | `.aether/ts-host/dist/` exists and is built | Ensure publish copies to hub; update copies to repo |

## Common Pitfalls

### Pitfall 1: Node Not in PATH
**What goes wrong:** `aether host` fails with obscure "file not found" error.
**Why it happens:** Users may not have Node installed or it may not be on PATH.
**How to avoid:** Explicit `exec.LookPath("node")` check with clear fallback message before attempting subprocess.
**Warning signs:** `exec.Command` returns `*exec.Error` with `Err == exec.ErrNotFound`.

### Pitfall 2: TS Host Built Assets Out of Sync
**What goes wrong:** `dist/` exists but is stale (older than `src/` changes).
**Why it happens:** Developers edit TS source but forget to rebuild.
**How to avoid:** During publish, always run `npm run build` from clean state. During update, check `mtime` of `src/` vs `dist/` or always rebuild.
**Warning signs:** Runtime behavior doesn't match source code.

### Pitfall 3: npm ci Failure Blocks Publish
**What goes wrong:** Network issue or registry problem causes `npm ci` to fail, killing the entire publish.
**Why it happens:** Treating TS host build as critical path instead of best-effort.
**How to avoid:** Log warning, continue with Go binary and companion file publish. TS host can be built later.
**Warning signs:** Publish fails with npm error even though Go binary built successfully.

### Pitfall 4: Missing node_modules in Downstream Repo
**What goes wrong:** Update copies `dist/` but not `node_modules`, and user tries to run `aether host`.
**Why it happens:** `node_modules` is excluded from hub sync (correctly — see `hubExcludeDirs`).
**How to avoid:** Update flow must run `npm ci` in the target repo after syncing TS host assets.
**Warning signs:** `aether host` fails with "Cannot find module" from Node.

## Code Examples

### Node Discovery with Version Check
```go
// Source: inferred from cmd/narrator_launcher.go + semver best practice
func discoverNode() (string, error) {
    nodePath, err := exec.LookPath("node")
    if err != nil {
        return "", fmt.Errorf("Node.js >= 20 is required. Install from https://nodejs.org/")
    }
    out, err := exec.Command(nodePath, "--version").Output()
    if err != nil {
        return "", fmt.Errorf("failed to check node version: %w", err)
    }
    version := strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
    major, _ := strconv.Atoi(strings.Split(version, ".")[0])
    if major < 20 {
        return "", fmt.Errorf("Node.js %s found, but >= 20 is required", version)
    }
    return nodePath, nil
}
```

### TS Host Path Resolution
```go
// Source: inferred from narratorRuntimePath pattern
func resolveTsHostPath(root string) (string, bool) {
    candidates := []string{}
    if root != "" {
        candidates = append(candidates, filepath.Join(root, ".aether", "ts-host", "dist", "host.js"))
    }
    if hub := resolveHubPath(); hub != "" {
        candidates = append(candidates, filepath.Join(hub, "system", "ts-host", "dist", "host.js"))
    }
    for _, c := range candidates {
        if info, err := os.Stat(c); err == nil && !info.IsDir() {
            return c, true
        }
    }
    return "", false
}
```

### Subcommand Delegation
```go
// Source: inferred from cmd/serve.go + narrator_launcher.go patterns
var hostLifecycleCmd = &cobra.Command{
    Use:   "lifecycle [phase]",
    Short: "Run full plan->build->continue lifecycle via TS host",
    RunE: func(cmd *cobra.Command, args []string) error {
        nodePath, err := discoverNode()
        if err != nil {
            return err // fallback message already in error
        }
        tsHostPath, ok := resolveTsHostPath("")
        if !ok {
            return fmt.Errorf("TS host assets not found. Run `aether update --force` to install them.")
        }
        tsArgs := []string{tsHostPath, "lifecycle"}
        if len(args) > 0 {
            tsArgs = append(tsArgs, args[0])
        }
        c := exec.Command(nodePath, tsArgs...)
        c.Stdin = os.Stdin
        c.Stdout = os.Stdout
        c.Stderr = os.Stderr
        return c.Run() // exit code forwarded automatically
    },
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bash/Node orchestration (Classic v5.x) | Go CLI + TS host hybrid (v1.16+) | 2026-04 | Go owns safety; TS owns orchestration |
| Direct `.aether/data/` writes from TS | Go finalizer commands only | Phase 109 | Boundary contract enforced |
| Narrator-only Node subprocess (`narrator.js`) | Full orchestration host (`host.js`) | Phase 109 | TS host now drives lifecycle, not just visuals |
| Manual Go command chains in wrappers | `aether host <workflow>` delegation | Phase 124 (this phase) | Wrappers become thin, host becomes thick |

**Deprecated/outdated:**
- `.aether/ts/` (old narrator-only TypeScript): superseded by `.aether/ts-host/` [VERIFIED: `.aether/ts/package.json` has no host commands]
- Direct wrapper Go command chains: being replaced by `aether host` delegation per v1.19 roadmap

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Node >= 20 is available on target machines for TS host users | Standard Stack | Users without Node cannot use `aether host`; fallback messaging mitigates |
| A2 | `npm ci` is sufficient for dependency install (no need for `npm install`) | Technical Approach | `npm ci` requires `package-lock.json`; if missing, falls back to `npm install` |
| A3 | `syncDirToHub` and related functions in `cmd/install_cmd.go` can be reused or adapted for TS host asset sync | Architecture Patterns | If exclusions differ, may need custom sync logic |
| A4 | TS host `oracle` command can be a stub in Phase 124; real implementation in Phase 127 | Technical Approach | `aether host oracle` will be non-functional until Phase 127, but entry point exists |

## Open Questions

1. **Should publish fail or warn if `npm` is missing?**
   - What we know: Go binary and companion files are the critical publish path.
   - What's unclear: Whether TS host assets are considered required or optional at this stage.
   - Recommendation: Warn, don't fail. TS host is new in v1.19; not all users need it immediately.

2. **Should update run `npm run build` automatically or prompt the user?**
   - What we know: `npm ci` + `tsc` can take 10-30 seconds.
   - What's unclear: Whether silent auto-build during `aether update` is acceptable UX.
   - Recommendation: Auto-build with stderr progress output. Update already runs other sync operations silently; build is analogous.

3. **How should the safety invariant test (`cmd/safety_invariant_test.go`) treat the new `host` command?**
   - What we know: `TestInstallPureGo` forbids `ts-host` strings in install/update/publish commands.
   - What's unclear: Whether the `host` command itself (which explicitly references TS host) needs exemption.
   - Recommendation: The `host` command is explicitly the bridge — it is expected to reference TS host. The safety invariant should only apply to install/update/publish, not to the host command itself. Update the test comment to clarify.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS host runtime | Yes | v25.9.0 | None — fallback messaging |
| npm | TS host dependency install | Yes | 11.12.1 | None — fallback messaging |
| Go | CLI build | Yes | 1.22+ | None |
| `aether` binary | Go CLI commands | Yes | v1.0.38 | `go run ./cmd/aether` |

**Missing dependencies with no fallback:** None on this machine.

**Missing dependencies with fallback:**
- Node/npm missing on user machine: Clear error message with install instructions.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — standard `go test` |
| Quick run command | `go test ./cmd -run TestHost -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HEP-01 | `aether host lifecycle` runs TS host | unit | `go test ./cmd -run TestHostLifecycle -v` | NEW |
| HEP-02 | `aether host plan` delegates to TS | unit | `go test ./cmd -run TestHostPlan -v` | NEW |
| HEP-03 | `aether host build` delegates to TS | unit | `go test ./cmd -run TestHostBuild -v` | NEW |
| HEP-04 | `aether host continue` delegates to TS | unit | `go test ./cmd -run TestHostContinue -v` | NEW |
| HEP-05 | `aether host oracle` delegates to TS | unit | `go test ./cmd -run TestHostOracle -v` | NEW |
| HEP-06 | Publish builds and syncs TS assets | unit | `go test ./cmd -run TestPublishTsHost -v` | NEW |
| HEP-07 | Fallback messaging when Node missing | unit | `go test ./cmd -run TestHostFallback -v` | NEW |

### Sampling Rate
- **Per task commit:** `go test ./cmd -run TestHost -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/host_cmd_test.go` — covers HEP-01 through HEP-07
- [ ] `cmd/host_cmd.go` — host command implementation
- [ ] Publish TS host build step — modify `cmd/publish_cmd.go`
- [ ] Update TS host sync step — modify `cmd/update_cmd.go`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | No | — |
| V5 Input Validation | Yes | Validate phase number is integer; validate Node path is absolute; prevent path traversal in TS host resolution |
| V6 Cryptography | No | — |

### Known Threat Patterns for Go + Node Subprocess Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via args | Tampering | Use `exec.Command` with slice args (no shell); validate positional args before passing |
| Path traversal to wrong Node binary | Tampering | `exec.LookPath` finds first `node` on PATH; document that users should secure PATH |
| npm dependency confusion | Tampering | `npm ci` uses `package-lock.json`; verify lockfile exists before install |

## Sources

### Primary (HIGH confidence)
- `cmd/narrator_launcher.go` — Node subprocess invocation, TTY pipe handling, runtime path resolution
- `cmd/install_cmd.go` — `syncDirToHub`, `setupInstallHub`, `hubExcludeDirs`, publish flow
- `cmd/publish_cmd.go` — Publish command structure, version verification, hub sync
- `cmd/update_cmd.go` — Update flow, repo sync pairs, stale publish detection
- `cmd/root.go` — Cobra root command, `Execute()`, `Version`
- `cmd/helpers.go` — `outputOK`, `outputError`, JSON envelope format
- `.aether/ts-host/package.json` — Node version requirement, build scripts, dependencies
- `.aether/ts-host/src/host.ts` — TS host CLI arg parsing, command dispatch
- `.aether/ts-host/src/go-bridge.ts` — `discoverGoBinary()`, `callGoJSON()`, boundary enforcement
- `.aether/references/contracts/runtime-boundary-contract.md` — Go/TS ownership boundary

### Secondary (MEDIUM confidence)
- `.aether/docs/publish-update-runbook.md` — Publish/update workflow documentation
- `.aether/docs/migration-map.md` — Milestone A/B/C migration plans, Oracle lifecycle pattern
- `cmd/safety_invariant_test.go` — `TestInstallPureGo` forbids TS host references in install/update/publish
- `cmd/platform_sync.go` — `installSyncPairs()`, `repoSyncPairs()`, sync pair structures

### Tertiary (LOW confidence)
- `cmd/testdata/command_catalog.json` — Contains `"host"` as a known command name (may be from `aether serve` host flag, not this command)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against registry and local environment
- Architecture: HIGH — existing patterns in narrator_launcher.go and install_cmd.go are clear
- Pitfalls: MEDIUM — based on observed patterns, no prior production TS host distribution exists

**Research date:** 2026-05-14
**Valid until:** 2026-06-14 (stable stack, low churn expected)
