# Aether End-to-End Test Suite

This directory contains end-to-end tests for the Aether colony sync system, verifying that `aether install`, `aether update`, and `aether update --all` work correctly.

## Test Files

- `test-install.sh` - Tests the `aether install` command
- `test-update.sh` - Tests the `aether update` command (single repo)
- `test-update-all.sh` - Tests the `aether update --all` command (all repos)
- `run-all.sh` - Runs all test suites and generates a summary report

## Running Tests

### Run All Tests

```bash
cd tests/e2e
./run-all.sh
```

### Run Individual Test Suites

```bash
cd tests/e2e
./test-install.sh
./test-update.sh
./test-update-all.sh
```

## Test Coverage

### test-install.sh

Tests that `aether install`:

1. Creates `~/.aether/` directory
2. Creates `~/.aether/system/` directory
3. Creates `~/.aether/commands/claude/` directory
4. Creates `~/.aether/commands/opencode/` directory
5. Creates `~/.aether/agents/` directory
6. Copies claude commands to global hub
7. Creates `~/.aether/version.json`
8. Creates `~/.aether/registry.json`
9. Creates `~/.aether/manifest.json`
10. Copies aether-utils.sh to system directory
11. Is idempotent (safe to run multiple times)
12. Shell scripts have executable bit

### test-update.sh

Tests that `aether update`:

1. Checks for hub existence
2. Checks for `.aether/` in target repo
3. Copies system files from hub
4. Updates `.aether/version.json`
5. Syncs claude commands
6. Removes stale files
7. Preserves colony data
8. Uses hash comparison to prevent unnecessary writes
9. Detects up-to-date repos
10. `--dry-run` doesn't modify files

### test-update-all.sh

Tests that `aether update --all`:

1. Updates all registered repos
2. Syncs commands to all repos
3. Removes stale files from all repos
4. Handles non-existent repos gracefully
5. Preserves colony data in all repos
6. `--dry-run` doesn't modify files
7. `--list` shows all registered repos
8. Updates registry timestamps

## Key Features Tested

### Install Command

- **Global hub setup**: Creates `~/.aether/` with all required directories
- **Command sync**: Copies commands from repo to global `~/.claude/commands/ant/`
- **Metadata files**: Creates version.json, registry.json, manifest.json
- **Idempotency**: Running install multiple times is safe

### Update Command

- **Hub to repo sync**: Copies newer files from hub to repo's `.aether/`
- **Hash comparison**: Only copies files when content changes (preserves timestamps)
- **Orphan cleanup**: Removes files that no longer exist in hub
- **Colony data preservation**: Never touches `.aether/data/`

### Update All Command

- **Multi-repo sync**: Updates all repos registered in `~/.aether/registry.json`
- **Batch operations**: Efficiently processes multiple repos
- **Non-existent repo handling**: Prunes repos that no longer exist

## Test Environment

All tests use isolated temporary environments:

- `$HOME` is set to a temporary directory for each test run
- Test repos are created in temporary directories
- All temporary files are cleaned up after tests complete
- Tests don't modify your actual Aether installation

## Prerequisites

- `node` - Node.js runtime
- `jq` - JSON processor for parsing version.json files
- `shasum` or `sha256sum` - File hash verification (on macOS/Linux)

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed
