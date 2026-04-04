# Roadmap: Aether Colony Orchestration System

## Milestones

- <details><summary>v5.4 Shell-to-Go Conversion (Phases 05-10) - SHIPPED 2026-04-04</summary>
Full shell-to-Go rewrite producing 254+ Cobra commands, 11 playbooks with 275 Go calls, test parity with 254 commands tested.
  - Phase 05: Structural Learning & Curation Ants (2/2 plans)
  - Phase 06: XML, Display & Semantic Search (2/2 plans)
  - Phase 07: Error Handling & Miscellaneous (4/4 plans)
  - Phase 08: Slash Command Wiring (2/2 plans)
  - Phase 09: Playbook Wiring (2/2 plans)
  - Phase 10: Integration Parity Tests (2/3 plans)
  </details>
- **v5.5 Go Binary Release** - Phases 48-51 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (48, 49, 50, 51): Planned milestone work
- Decimal phases (48.1, 49.1): Urgent insertions (marked with INSERTED)

- [x] **Phase 48: goreleaser Release Pipeline** - Cross-platform binary builds on tag push (completed 2026-04-04)
- [x] **Phase 49: Binary Downloader + npm Install** - Users receive Go binary on npm install (completed 2026-04-04)
- [ ] **Phase 50: Update Flow Binary Refresh** - Users get updated binary via aether update
- [ ] **Phase 51: npm Shim Delegation + Version Gate** - aether command routes to Go binary when available

## Phase Details

### Phase 48: goreleaser Release Pipeline
**Goal**: Pushing a version tag triggers a GitHub Actions workflow that produces downloadable cross-platform binaries on GitHub Releases
**Depends on**: Nothing (first phase in this milestone)
**Requirements**: REL-01, REL-02, REL-03
**Success Criteria** (what must be TRUE):
  1. Pushing a `v*` git tag triggers a GitHub Actions workflow that produces 6 platform archives (darwin/linux/windows x amd64/arm64) uploaded to a GitHub Release
  2. Each GitHub Release includes a `checksums.txt` file with SHA-256 hashes for all platform archives
  3. goreleaser config validation (`goreleaser check`) runs as part of existing CI, catching config drift before release
**Plans**: 2 plans

Plans:
- [x] 48-01-PLAN.md -- Create release workflow + fix CI Go version + add goreleaser check (original, not executed)
- [x] 48-02-PLAN.md -- Gap closure: create release workflow + fix CI Go version + add goreleaser check

### Phase 49: Binary Downloader + npm Install
**Goal**: Users receive the correct platform Go binary automatically when running npm install -g aether-colony
**Depends on**: Phase 48
**Requirements**: BIN-01, BIN-02, BIN-03, BIN-04
**Success Criteria** (what must be TRUE):
  1. Running `npm install -g aether-colony` downloads and installs the Go binary matching the user's OS and architecture to `~/.aether/bin/aether`
  2. Platform detection correctly identifies the user's OS and CPU architecture (including macOS universal binary handling) without manual configuration
  3. Every downloaded binary is verified against the published SHA-256 checksum before being placed at its final path
  4. A failed or interrupted download never leaves a corrupted binary at the install path (download-to-temp, verify, atomic rename)
**Plans**: 2 plans

Plans:
- [ ] 49-01-PLAN.md -- Create binary-downloader module with platform detection, checksum verification, and atomic install + unit tests
- [ ] 49-02-PLAN.md -- Wire downloadBinary into performGlobalInstall + add integration contract tests

### Phase 50: Update Flow Binary Refresh
**Goal**: Users get an updated Go binary when running aether update, without the update flow breaking on binary failure
**Depends on**: Phase 49
**Requirements**: UPD-01, UPD-02
**Success Criteria** (what must be TRUE):
  1. Running `aether update` downloads a new binary when the released version is newer than the installed binary
  2. If the binary download or update fails, the rest of the update flow (file sync, YAML refresh) still completes successfully
**Plans**: TBD

### Phase 51: npm Shim Delegation + Version Gate
**Goal**: The aether command delegates to the Go binary when it is present and confirmed working, falling back to Node.js when it is not
**Depends on**: Phase 49
**Requirements**: GATE-01, GATE-02, SHM-01, SHM-02
**Success Criteria** (what must be TRUE):
  1. Running any `aether` command routes to the Go binary when the version gate passes (binary exists, is executable, reports matching version)
  2. Running any `aether` command falls back to the Node.js CLI when the version gate fails (binary missing, not executable, or version mismatch)
  3. Commands that must run in Node.js (install, update, setupHub) always run in Node.js regardless of whether the Go binary is available
  4. Version comparison works using custom semver logic with no external npm dependencies
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 48 -> 49 -> 50/51 (parallel)
Phases 50 and 51 both depend only on Phase 49 and can execute in parallel.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 48. goreleaser Release Pipeline | 2/2 | Complete    | 2026-04-04 |
| 49. Binary Downloader + npm Install | 0/2 | Complete    | 2026-04-04 |
| 50. Update Flow Binary Refresh | 0/? | Not started | - |
| 51. npm Shim Delegation + Version Gate | 0/? | Not started | - |
