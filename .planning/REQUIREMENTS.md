# Requirements: Aether Colony Orchestration System

**Defined:** 2026-04-04
**Core Value:** The system must reliably interpret a user request, decompose it into executable work, verify outputs, and ship correct work with minimal back-and-forth.

## v5.5 Requirements

Requirements for Go Binary Release milestone. Each maps to roadmap phases.

### Release Pipeline

- [x] **REL-01**: User can trigger a cross-platform release by pushing a `v*` git tag, producing binaries for darwin/linux/windows on amd64/arm64
- [x] **REL-02**: User gets checksums.txt alongside release binaries for integrity verification
- [x] **REL-03**: goreleaser config check runs in existing CI to catch config drift before release

### Binary Download

- [ ] **BIN-01**: User receives the Go binary automatically when running `npm install -g aether-colony`
- [ ] **BIN-02**: User receives the correct platform binary (OS + architecture detected automatically)
- [ ] **BIN-03**: System verifies binary integrity via SHA-256 checksum before installing
- [ ] **BIN-04**: Binary installs atomically (download to temp, verify, rename) so a failed download never corrupts the existing binary

### Update Flow

- [x] **UPD-01**: User gets an updated binary when running `aether update` if the released binary is newer than installed one
- [x] **UPD-02**: Binary update failure does not block the rest of the update flow (file sync, YAML refresh still complete)

### Version Gate

- [ ] **GATE-01**: System checks three conditions before routing to the Go binary: binary exists at expected path, is executable, and reports a version matching the npm package
- [ ] **GATE-02**: Version comparison works without external npm dependencies (custom semver logic)

### npm Shim

- [ ] **SHM-01**: User's `aether` command delegates to the Go binary when the version gate passes, falling back to Node.js CLI when it does not
- [ ] **SHM-02**: Commands that must run in Node.js (install, update, setupHub) always stay in Node.js regardless of binary availability

## v5.6 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Distribution Extras

- **BREW-01**: User can install via Homebrew tap
- **SIGN-01**: Binaries are code-signed for macOS and Windows (avoids Gatekeeper/SmartScreen warnings)

### Advanced Binary Management

- **SELF-01**: User can update the binary independently via `aether update-self`
- **ROLL-01**: System keeps `aether.old` backup for rollback after binary update

## Out of Scope

| Feature | Reason |
|---------|--------|
| PATH profile injection | npm shim delegates to `~/.aether/bin/aether` directly -- no PATH change needed; profile injection is fragile across shells/platforms |
| Remove shell fallback | Shell fallback is insurance against binary issues; keep indefinitely |
| `go:embed` for self-contained binary | Separate architecture decision, not a distribution concern |
| Bundled binaries in npm package | Would make npm package 100MB+; download-on-demand is better |
| Cosign/SBOM | Oversized for current project scale |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| REL-01 | 48 | Complete |
| REL-02 | 48 | Complete |
| REL-03 | 48 | Complete |
| BIN-01 | 49 | Pending |
| BIN-02 | 49 | Pending |
| BIN-03 | 49 | Pending |
| BIN-04 | 49 | Pending |
| UPD-01 | 50 | Complete
| UPD-02 | 50 | Complete |
| GATE-01 | 51 | Pending |
| GATE-02 | 51 | Pending |
| SHM-01 | 51 | Pending |
| SHM-02 | 51 | Pending |

**Coverage:**
- v5.5 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0

---
*Requirements defined: 2026-04-04*
*Last updated: 2026-04-04 after roadmap creation*
