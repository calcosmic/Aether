# Phase 104: Release Integrity & Regression Test Suite — Context

**Gathered:** 2026-05-07
**Status:** Ready for planning
**Source:** ROADMAP.md requirements + prior phase deliverables

## Phase Boundary

Phase 104 is the penultimate audit phase. It verifies the release pipeline operates as one coherent system and creates regression tests that freeze all verified contracts from Phases 100-104 so future drift fails CI.

**What this phase delivers:**
- End-to-end release pipeline verification test (publish → hub sync → install/update cycle)
- Structural snapshot regression tests freezing all six audit dimensions
- Review ledger persistence verification (DATA-03)

**What this phase does NOT deliver:**
- No changes to the publish/update/install runtime code (read-only verification)
- No remediation of findings (Phase 105 handles that)
- No new CLI commands or runtime features

## Implementation Decisions

### Locked Decisions
- **Test-only phase**: All deliverables are tests and verification artifacts — no runtime code changes
- **Golden snapshot pattern**: Replicate the Phase 100/102/102 test pattern (golden JSON + report cross-reference)
- **Read-only audit**: Tests verify existing behavior, do not modify publish/update/install logic
- **Mock filesystem for E2E**: Publish/install/update tests use temp directories to simulate hub and source checkout without touching real ~/.aether/

### Architecture Decisions
- **Structural snapshot scope**: Cover all six audit dimensions: command contracts (Phase 100), wrapper parity (Phase 101), worker economy (Phase 102), data flow (Phase 103), release integrity (Phase 104), gate classifications
- **Review ledger test approach**: Verify ledger accumulates across phases by reading existing .aether/data/reviews/ structure and testing write/read round-trip
- **Release pipeline E2E test**: Mock the full cycle — build binary, sync to mock hub, install from mock hub, update from mock hub, verify no stale files

## Canonical References

### Prior Phase Deliverables (MUST be frozen by regression tests)
- `cmd/testdata/command_catalog.json` — Phase 100 golden snapshot (377 commands)
- `cmd/audit_catalog_test.go` — Phase 100 catalog verification tests
- `cmd/contract_validate_test.go` — Phase 100 contract structure tests
- `cmd/testdata/worker_economy_snapshot.json` — Phase 102 golden snapshot
- `cmd/worker_economy_test.go` — Phase 102 worker economy tests
- `cmd/testdata/data_flow_snapshot.json` — Phase 103 golden snapshot
- `cmd/data_flow_audit_test.go` — Phase 103 data flow tests

### Release Pipeline Source Code
- `cmd/publish_cmd.go` — `aether publish` (build binary, sync companion files, verify version)
- `cmd/update_cmd.go` — `aether update` (sync companion files from hub, stale file cleanup)
- `cmd/install_cmd.go` — `aether install` (copy commands/agents/docs to hub)
- `cmd/platform_sync.go` — Sync pair definitions for companion file copy
- `cmd/version.go` — `aether version --check` (binary/hub version agreement)

### Review Ledger System
- `cmd/review_ledger.go` — Domain review ledger CRUD
- `cmd/review_ledger_test.go` — Existing ledger tests
- `cmd/colony_prime_prior_reviews_test.go` — Prior reviews section tests

### Existing Tests to Build Upon
- `cmd/publish_cmd_test.go` — Existing publish tests (mock source checkout, version mismatch detection)
- `cmd/update_cmd_test.go` — Existing update tests

## Specific Ideas

### Regression Test Structure
```
cmd/regression_test.go              # Master regression suite
cmd/testdata/regression_snapshot.json  # Golden snapshot with:
  - command_count: 377 (from Phase 100)
  - lifecycle_contracts: 16 (from Phase 100)
  - caste_count: 26 (from Phase 102)
  - colony_prime_sections: 16 (from Phase 103)
  - artifact_count: 33 (from Phase 103)
  - parity_surfaces: 5 (from Phase 101, when complete)
```

### Release Pipeline E2E Test
Simulate the full cycle in a temp directory:
1. Create mock source checkout (go.mod, .aether/commands/, .claude/commands/)
2. Run publish logic to mock hub
3. Verify hub has correct files, version matches
4. Run update --force from mock hub
5. Verify no stale files, version agreement
6. Run install to fresh mock home
7. Verify all companion files present

### Review Ledger Persistence Test
1. Write review entries for multiple domains
2. Verify entries survive simulated session reset (new store instance reading same file)
3. Verify cross-phase accumulation (entries from phase N still present when phase N+1 writes)

## Deferred Ideas

- Phase 101 parity verification is incomplete (0/2 plans) — Phase 104 regression tests should reference Phase 101 findings but not depend on them being complete
- Binary download from GitHub Releases is out of scope for automated tests (network dependency) — mock the download layer
- Real `go build` in publish test is slow — use `--skip-build-binary` or mock binary

---

*Phase: 104-release-integrity-regression-test-suite*
*Context gathered: 2026-05-07 via ROADMAP-derived context*
