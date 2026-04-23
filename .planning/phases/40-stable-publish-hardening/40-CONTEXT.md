# Phase 40: Stable Publish Hardening - Context

**Gathered:** 2026-04-23
**Status:** Ready for planning
**Source:** Operations guide, existing install flow, user discussion

<domain>
## Phase Boundary

Ensure stable publish atomically syncs binary and hub to the same version — no more 1.0.20 binary with 1.0.19 hub. This phase introduces a dedicated `aether publish` command and verifies version agreement at publish time.

**What this phase delivers:**
- New `aether publish` command that wraps binary build + hub sync + version stamping
- Hub version.json always matches binary version after publish
- Auto-check verifies agreement after publish completes
- `aether version --check` flag for manual downstream verification
- Existing 1.0.19/1.0.20 mismatch reproduced and fixed

</domain>

<decisions>
## Implementation Decisions

### Atomicity Strategy (LOCKED)

- **Hub follows binary** — When publish runs, hub `version.json` is updated to match whatever the binary version is. If binary is v1.0.20, hub becomes v1.0.20. Warns if hub was behind.
- The version source is the same `resolveVersion()` logic already in `cmd/root.go` — ldflags first, then git tags, then fallback.
- Publish writes hub `version.json` AFTER successful companion sync, not before.

### Publish Entry Point (LOCKED)

- **New `aether publish` command** — dedicated command for the source-to-hub publishing flow.
- Wraps the existing `install --package-dir` hub sync logic into a proper publish operation.
- Replaces the ad-hoc `aether install --package-dir "$PWD"` pattern shown in the operations guide.
- The operations guide (Section 5, Step C) should be updated to use `aether publish` instead.
- `aether install --package-dir` continues to work for backward compatibility but `aether publish` is the documented path.
- Publish command should also handle binary building via `go build` with ldflags (or accept a pre-built binary path).
- The `--channel` flag from the existing install flow carries through: `aether publish` for stable, `aether publish --channel dev` for dev (dev channel isolation is Phase 41, but the flag should exist).

### Verification Approach (LOCKED)

- **Auto-check after publish** — The publish command automatically verifies version agreement after writing. If hub and binary don't agree after publish, it fails loudly.
- **`aether version --check` flag** — Manual downstream verification that `aether version` and `~/.aether/system/version.json` agree. Returns non-zero if they don't.
- Phase 43 covers the full integrity chain (source → binary → hub → downstream). This phase only handles binary ↔ hub agreement.

### Claude's Discretion

- Exact `aether publish` command flag names and defaults
- Whether `aether publish` runs `go build` automatically or requires a pre-built binary
- Error message wording for version mismatch
- Test structure and organization

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Publish Pipeline
- `AETHER-OPERATIONS-GUIDE.md` — The complete operations guide defining the publish workflow (Sections 5, 6, 7)
- `cmd/install_cmd.go` — Existing install/publish logic including `syncDirToHub` and `version.json` writing
- `cmd/root.go` — `resolveVersion()` function and `normalizeVersion()`
- `cmd/update_cmd.go` — Downstream update flow (consumes hub)

### Version Management
- `.aether/version.json` — Source version file
- `npm/package.json` — npm version (must match for releases)
- `cmd/root.go:28-43` — Version resolution chain (ldflags → git tags → fallback)

### Existing Tests
- `cmd/e2e_install_setup_update_test.go` — E2E test patterns for install/update flow
- `cmd/install_cmd_test.go` — Unit test patterns for install command

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `resolveVersion()` in `cmd/root.go:28` — Already resolves version from ldflags/git tags. Publish command should reuse this.
- `syncDirToHub()` in `cmd/install_cmd.go:660` — Hub sync logic already handles file comparison and stale file removal.
- `version.json` writing in `cmd/install_cmd.go:644-652` — Already writes hub version during install.
- `normalizeVersion()` in `cmd/root.go` — Strips `v` prefix, ensures consistent format.

### Established Patterns
- `--package-dir` flag pattern for specifying source location
- `--channel` flag for stable vs dev routing
- Hub directory at `~/.aether/system/` (stable) or `~/.aether-dev/system/` (dev)
- `version.json` format: `{"version":"1.0.20","updated_at":"now"}`

### Integration Points
- `aether update` reads hub `version.json` to determine what version downstream repos get
- `aether version` reads both binary version and hub version
- `aether install --package-dir` is the current (unofficial) publish path

</code_context>

<specifics>
## Specific Requirements

1. Reproduce the current 1.0.19/1.0.20 mismatch — verify it exists, then prove it's fixed
2. `aether publish` should work as documented in the operations guide
3. Update operations guide to use `aether publish` instead of `aether install --package-dir`
4. After publish, `aether version` and `~/.aether/system/version.json` must agree

</specifics>

<deferred>
## Deferred Ideas

- Dev channel isolation (Phase 41)
- Downstream stale-publish detection (Phase 42)
- Full integrity chain check (Phase 43)
- Auto-version bumping from git tags (could be future improvement)

---
*Phase: 40-stable-publish-hardening*
*Context gathered: 2026-04-23*
