# Aether Test Suite Analysis

**Generated:** 2026-02-16
**Analyzed by:** Oracle (comprehensive test audit)

---

## Executive Summary

The Aether test suite consists of **40 test files** across 4 categories, using **2 primary frameworks** (AVA for JavaScript, custom bash framework). While many tests pass, there are significant quality issues including **failing tests**, **tests with unclear purpose**, and **gaps in coverage** for critical components.

### Key Findings

| Metric | Value |
|--------|-------|
| Total Test Files | 40 |
| Unit Tests (JS) | 24 |
| Bash Tests | 9 |
| E2E Tests | 5 |
| Integration Tests | 2 |
| Tests Passing | ~85% |
| Tests Failing | ~10% |
| Tests with Issues | ~5% |

---

## Test Frameworks

### 1. AVA (JavaScript Unit Tests)
- **Configuration:** `package.json` - timeout 30s
- **Location:** `tests/unit/**/*.test.js`
- **Features:**
  - Parallel test execution
  - Built-in assertion library
  - Promise/async support
  - Test isolation via separate processes

### 2. Custom Bash Test Framework
- **Location:** `tests/bash/test-helpers.sh`
- **Features:**
  - JSON assertion helpers (`assert_json_valid`, `assert_ok_true`)
  - Test environment setup/teardown
  - Colored output (pass/fail)
  - Exit code validation
  - Temporary directory management

### 3. E2E Shell Tests
- **Location:** `tests/e2e/*.sh`
- **Features:**
  - Full workflow testing
  - Isolated temp environments
  - Hub/registry simulation
  - Multi-repo scenarios

---

## Test File Inventory

### Unit Tests (24 files)

| File | Purpose | Status | Notes |
|------|---------|--------|-------|
| `cli-hash.test.js` | SHA-256 hashing utilities | PASS | 9 tests, all passing |
| `cli-manifest.test.js` | Manifest generation/validation | PASS | 7 tests, all passing |
| `cli-override.test.js` | Model profile selection | **FAIL** | 10 tests, 8 failing - aether-utils.sh path issue |
| `cli-sync.test.js` | Directory sync with cleanup | PASS | 17 tests, all passing |
| `cli-telemetry.test.js` | Telemetry display | PASS | 16 tests, all passing |
| `colony-state.test.js` | COLONY_STATE.json validation | PASS | 11 tests, all passing |
| `file-lock.test.js` | File locking mechanism | PASS | 12 tests, all passing |
| `init.test.js` | Repository initialization | PASS | 10 tests, all passing |
| `model-profiles.test.js` | Model profile loading | PASS | 25 tests, all passing |
| `model-profiles-overrides.test.js` | User override handling | PASS | Tests caste-to-model mapping |
| `model-profiles-task-routing.test.js` | Task-based routing | PASS | Tests keyword matching |
| `namespace-isolation.test.js` | Command namespace isolation | **BROKEN** | Calls `process.exit()`, crashes test runner |
| `oracle-regression.test.js` | Oracle command regression | PASS | Validates research workflow |
| `spawn-tree.test.js` | Spawn tree operations | PASS | 10 tests, validates depth tracking |
| `state-guard.test.js` | State transition guards | PASS | Iron Law enforcement tests |
| `state-guard-events.test.js` | Event audit trail | PASS | Event recording/retrieval |
| `state-loader.test.js` | State persistence | PASS | Load/save operations |
| `state-sync.test.js` | State synchronization | PASS | Cross-repo state sync |
| `sync-dir-hash.test.js` | Hash-based sync | PASS | Content-addressable sync |
| `telemetry.test.js` | Telemetry collection | PASS | Event logging |
| `update-transaction.test.js` | Update with rollback | **MIXED** | 1 test failing (verifyIntegrity) |
| `update-errors.test.js` | Error handling | **MIXED** | Several tests failing |
| `user-modification-detection.test.js` | Detect user edits | PASS | Hash comparison |
| `validate-state.test.js` | State validation | PASS | Schema validation |

### Bash Tests (9 files)

| File | Purpose | Status | Notes |
|------|---------|--------|-------|
| `test-aether-utils.sh` | aether-utils.sh subcommands | **MIXED** | 14 tests, 1 failing (bootstrap-system) |
| `test-generate-commands.sh` | Command generation | PASS | Validates Claude/OpenCode sync |
| `test-helpers.sh` | Test framework utilities | PASS | Shared helpers |
| `test-phase3-xml.sh` | Phase 3 XML conversion | PASS | JSON to XML |
| `test-pheromone-xml.sh` | Pheromone XML export | PASS | Signal serialization |
| `test-session-freshness.sh` | Session freshness detection | PASS | 21/21 tests passing |
| `test-xinclude-composition.sh` | XInclude merging | PASS | Document composition |
| `test-xml-security.sh` | XML security checks | PASS | XXE prevention |
| `test-xml-utils.sh` | XML utility functions | PASS | 20/20 tests passing |

### E2E Tests (5 files)

| File | Purpose | Status | Notes |
|------|---------|--------|-------|
| `checkpoint-update-build.test.js` | Full workflow | PASS | Integration of checkpoint, update, build |
| `test-install.sh` | Install command | PASS | Hub setup verification |
| `test-update.sh` | Update command | PASS | Single repo update |
| `test-update-all.sh` | Update --all | PASS | Multi-repo update |
| `update-rollback.test.js` | Rollback behavior | PASS | Transaction rollback |

### Integration Tests (2 files)

| File | Purpose | Status | Notes |
|------|---------|--------|-------|
| `file-lock-integration.test.js` | Concurrent locking | PASS | Multi-process safety |
| `state-guard-integration.test.js` | State transitions | PASS | Phase advancement flow |

---

## Failing Tests Analysis

### Critical Failures (Block CI)

#### 1. `cli-override.test.js` - Model Profile Selection
**Failure:** `bash: .aether/aether-utils.sh: No such file or directory`

**Root Cause:** Tests run from repo root but look for `.aether/aether-utils.sh` in current working directory. When tests create temp directories, the relative path breaks.

**Affected Tests:**
- model-profile select returns task-routing default when no keyword match
- model-profile select returns CLI override when provided
- model-profile select returns task-routing model when no CLI override
- model-profile select returns user override when no CLI override
- model-profile select CLI override takes precedence over user override
- model-profile validate returns valid:true for known models
- model-profile validate returns valid:false for unknown models
- integration: end-to-end model selection with all override types
- integration: verify JSON output structure

**Fix Required:** Use absolute paths or copy aether-utils.sh to temp directories.

#### 2. `namespace-isolation.test.js` - Test Runner Crash
**Failure:** `Error: Unexpected process.exit()`

**Root Cause:** Test explicitly calls `process.exit()` at line 333, which AVA intercepts as a crash.

**Fix Required:** Refactor to return exit codes instead of calling process.exit().

### Medium Priority Failures

#### 3. `test-aether-utils.sh` - Bootstrap System
**Failure:** `E_HUB_NOT_FOUND: unbound variable`

**Root Cause:** Missing error code constant in aether-utils.sh when hub doesn't exist.

**Fix Required:** Define `E_HUB_NOT_FOUND` constant or use string error codes.

#### 4. `update-transaction.test.js` - verifyIntegrity
**Failure:** `verifyIntegrity detects missing files`

**Root Cause:** Test expects specific behavior when files are missing but implementation may have changed.

#### 5. `update-errors.test.js` - Multiple Failures
**Affected Tests:**
- detectDirtyRepo identifies modified files
- validateRepoState throws UpdateError with E_REPO_DIRTY
- detectPartialUpdate finds missing files
- detectPartialUpdate finds corrupted files with hash mismatch
- detectPartialUpdate finds corrupted files with size mismatch
- E_REPO_DIRTY recovery commands include cd to repo path
- verifySyncCompleteness throws E_PARTIAL_UPDATE on partial files
- E_PARTIAL_UPDATE error includes retry command

**Root Cause:** Tests may be using outdated mocks or testing implementation details that changed.

---

## Coverage Gaps

### Untested Critical Components

1. **Pheromone System**
   - Signal creation and propagation
   - Priority-based routing
   - Expiration handling
   - Cross-colony signal sharing

2. **Caste System**
   - Actual model routing (configuration exists but execution unverified)
   - Worker spawn with specific castes
   - Caste-based task assignment

3. **Command Sync**
   - `bin/generate-commands.sh` has tests but coverage is shallow
   - OpenCode command generation not fully tested
   - Command validation (only sync check, not behavior)

4. **Oracle/Research Loop**
   - RALF (Research-Analyze-Learn-Format) loop
   - Progress tracking
   - Research.json evolution

5. **Chamber/Archive System**
   - Colony sealing
   - Chamber creation
   - Entombment workflow

6. **Hook System**
   - Pre-commit hooks
   - Auto-format
   - Path protection

### Partially Tested

1. **Session Freshness** - Well tested (21 tests, all passing)
2. **XML Utilities** - Well tested (20 tests, all passing)
3. **State Guard** - Good coverage but some edge cases missing
4. **File Lock** - Good coverage including integration tests

---

## Test Quality Issues

### 1. Tests Testing Mocks, Not Reality

**File:** `cli-telemetry.test.js`

**Issue:** Tests create mock data structures and test those mocks, not actual telemetry collection:

```javascript
const mockSummary = createMockSummary({ totalSpawns: 0, totalModels: 0 });
const summary = mockSummary; // Just assigns mock to variable
t.is(summary.total_spawns, 0); // Tests mock, not real code
```

**Impact:** Tests pass but don't verify actual telemetry behavior. The real telemetry collection code is untested.

### 2. Tests with Unclear Purpose

**Files:**
- `cli-telemetry.test.js` - Tests mock data structures
- `user-modification-detection.test.js` - Tests file hashing but not integration with colony

### 3. Duplicate Test Logic

**Pattern:** Multiple test files test similar functionality:
- `state-guard.test.js` and `state-guard-integration.test.js` overlap
- `file-lock.test.js` and `file-lock-integration.test.js` overlap

### 4. Platform-Specific Tests

**File:** `test-xml-utils.sh`

**Issue:** Tests skip when tools unavailable (xmllint, xmlstarlet) but don't fail, giving false confidence.

---

## Recommendations

### Immediate Actions (P0)

1. **Fix cli-override.test.js** - Use absolute paths to aether-utils.sh
2. **Fix namespace-isolation.test.js** - Remove process.exit() call
3. **Fix E_HUB_NOT_FOUND** - Add missing constant to aether-utils.sh

### Short Term (P1)

4. **Add pheromone system tests** - Core feature with zero coverage
5. **Add caste routing integration tests** - Verify model routing actually works
6. **Refactor cli-telemetry tests** - Test real code, not mocks
7. **Add chamber/seal/entomb tests** - Lifecycle management untested

### Long Term (P2)

8. **Consolidate duplicate tests** - Merge overlapping test files
9. **Add performance benchmarks** - No performance tests exist
10. **Add chaos engineering tests** - Random failure injection
11. **Improve E2E coverage** - Only 5 E2E tests for complex system

---

## Test Execution Summary

### Command Reference

```bash
# Run all tests
npm test

# Run only unit tests
npm run test:unit

# Run only bash tests
npm run test:bash

# Run individual bash test
bash tests/bash/test-session-freshness.sh

# Run linting
npm run lint:shell
npm run lint:json
npm run lint:sync
```

### Current Test Results

| Category | Files | Tests | Passing | Failing | Status |
|----------|-------|-------|---------|---------|--------|
| Unit (JS) | 24 | ~200 | ~170 | ~20 | Yellow |
| Bash | 9 | ~100 | ~95 | ~5 | Green |
| E2E | 5 | ~30 | ~30 | 0 | Green |
| Integration | 2 | ~20 | ~20 | 0 | Green |
| **Total** | **40** | **~350** | **~315** | **~25** | **Yellow** |

---

## Appendix: File Locations

### Source Files Referenced

- `/Users/callumcowie/repos/Aether/package.json` - Test configuration
- `/Users/callumcowie/repos/Aether/tests/bash/test-helpers.sh` - Bash test framework
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - Main utility script (test target)
- `/Users/callumcowie/repos/Aether/bin/lib/` - JavaScript modules (test targets)

### Test Directories

- `/Users/callumcowie/repos/Aether/tests/unit/` - 24 JavaScript test files
- `/Users/callumcowie/repos/Aether/tests/bash/` - 9 bash test files
- `/Users/callumcowie/repos/Aether/tests/e2e/` - 5 E2E test files
- `/Users/callumcowie/repos/Aether/tests/integration/` - 2 integration test files

---

*Analysis complete. Recommend prioritizing failing test fixes before adding new coverage.*
