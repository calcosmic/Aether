# Technology Stack: v1.1 Bug Fixes

**Project:** Aether Colony System v1.1
**Researched:** 2026-02-14
**Confidence:** HIGH (based on existing codebase patterns and Node.js ecosystem standards)

## Overview

This document specifies the technology stack needed to fix four critical issues in v1.1:
1. Phase advancement logic causing AI loops
2. npm package update system for cross-repo sync
3. Build checkpoint stashing user data
4. Missing package-lock.json for deterministic builds
5. Unit tests for core sync functions

The existing stack (Node.js CLI with commander.js, AVA for testing, bash utilities) is sound. These recommendations focus on additions and fixes to address specific bugs.

---

## Core Technologies (No Changes)

| Technology | Version | Purpose | Status |
|------------|---------|---------|--------|
| Node.js | >=16.0.0 | Runtime | Keep - meets requirements |
| commander | ^12.1.0 | CLI argument parsing | Keep - already in use |
| picocolors | ^1.1.1 | Colored output | Keep - already in use |
| AVA | ^6.0.0 | Unit testing | Keep - already configured |

---

## Recommended Additions for v1.1

### 1. Deterministic Builds: package-lock.json

**What:** npm's lock file for deterministic dependency installation

**Why Required:**
- Without package-lock.json, `npm install` pulls latest matching versions
- Breaking changes in dependencies can slip in silently
- CI builds may differ from local builds
- Required for reproducible releases

**Implementation:**
```bash
# Generate package-lock.json
npm install

# Commit to repository
git add package-lock.json
git commit -m "Add package-lock.json for deterministic builds"

# In CI, use:
npm ci  # Uses package-lock.json exactly
```

**Confidence:** HIGH - Industry standard practice

---

### 2. Unit Testing: sinon for Mocking

**What:** Standalone test spies, stubs, and mocks for JavaScript

**Why Required:**
- Current tests (AVA) test bash utilities via execSync
- Need to test JavaScript functions (syncDirWithCleanup, updateRepo) in isolation
- File system operations need mocking for fast, deterministic tests
- CLI functions depend on external state (hub, repos, git)

**Version:** ^17.0.0 (latest stable)

**Installation:**
```bash
npm install --save-dev sinon
```

**Usage Pattern for syncDirWithCleanup:**
```javascript
const test = require('ava');
const sinon = require('sinon');
const fs = require('fs');
const proxyquire = require('proxyquire');

// Mock fs for isolated tests
const mockFs = {
  existsSync: sinon.stub(),
  readdirSync: sinon.stub(),
  copyFileSync: sinon.stub(),
  mkdirSync: sinon.stub(),
  readFileSync: sinon.stub(),
  unlinkSync: sinon.stub(),
  rmdirSync: sinon.stub(),
  chmodSync: sinon.stub()
};

// Load module with mocked dependencies
const { syncDirWithCleanup } = proxyquire('../bin/cli.js', {
  fs: mockFs
});

test.beforeEach(() => {
  sinon.resetHistory();
});

test('syncDirWithCleanup skips files with matching hashes', t => {
  mockFs.existsSync.returns(true);
  mockFs.readdirSync.returns([{ name: 'file.txt', isDirectory: () => false }]);
  mockFs.readFileSync.returns(Buffer.from('content'));

  const result = syncDirWithCleanup('/src', '/dest');

  t.is(result.skipped, 1);
  t.is(result.copied, 0);
  t.false(mockFs.copyFileSync.called);
});
```

**Confidence:** HIGH - Sinon is the de facto standard for JS mocking

---

### 3. Module Testing: proxyquire

**What:** Override dependencies during testing

**Why Required:**
- CLI functions in bin/cli.js use require('fs') directly
- Need to inject mocks without modifying source
- Enables testing of internal functions not exported

**Version:** ^2.1.3

**Installation:**
```bash
npm install --save-dev proxyquire
```

**Confidence:** HIGH - Standard for testing modules with dependencies

---

### 4. Temporary File Testing: tmp

**What:** Temporary file and directory creation for tests

**Why Required:**
- Integration tests need real file system operations
- Must clean up after tests to avoid pollution
- Handles OS-specific temp directory locations

**Version:** ^0.2.1

**Installation:**
```bash
npm install --save-dev tmp
```

**Usage:**
```javascript
const tmp = require('tmp');
const fs = require('fs');
const path = require('path');

test('updateRepo syncs files correctly', t => {
  const tempDir = tmp.dirSync({ unsafeCleanup: true });
  const hubDir = path.join(tempDir.name, 'hub');
  const repoDir = path.join(tempDir.name, 'repo');

  // Setup test files...

  t.teardown(() => tempDir.removeCallback());
});
```

**Confidence:** HIGH - Well-maintained, widely used

---

## Bug-Specific Stack Recommendations

### Bug 1: Phase Advancement Loops

**Root Cause:** Commands check state but don't verify phase progression logic

**Stack Addition:** State machine validation

```javascript
// Add to bin/lib/validators.js
function validatePhaseTransition(currentPhase, requestedPhase, state) {
  // Prevent loops: cannot build same phase twice without completion
  if (requestedPhase === currentPhase && state === 'BUILDING') {
    return { valid: false, reason: 'Phase already in progress' };
  }

  // Prevent backwards movement without explicit reset
  if (requestedPhase < currentPhase && state !== 'COMPLETED') {
    return { valid: false, reason: 'Cannot regress to earlier phase' };
  }

  return { valid: true };
}
```

**No new dependencies needed** - use existing JSON state files

---

### Bug 2: Update System Cross-Repo Sync

**Root Cause:** updateRepo function lacks comprehensive tests

**Stack Addition:** Test fixtures for multi-repo scenarios

```javascript
// tests/fixtures/multi-repo-setup.js
function createMultiRepoFixture(tempDir) {
  return {
    hubDir: createHub(tempDir),
    repo1: createRepo(tempDir, 'repo1'),
    repo2: createRepo(tempDir, 'repo2'),
    registry: createRegistry(tempDir)
  };
}
```

**Dependencies:** tmp (already recommended above)

---

### Bug 3: Checkpoint Stashing User Data

**Root Cause:** git stash operates on all dirty files

**Stack Addition:** Git parsing utilities

```javascript
// bin/lib/git-utils.js
const { execSync } = require('child_process');

function getDirtyFilesInDirs(repoPath, targetDirs) {
  // Already exists in cli.js - extract to module
  const args = targetDirs.filter(d => fs.existsSync(path.join(repoPath, d)));
  if (args.length === 0) return [];

  const result = execSync(
    `git status --porcelain -- ${args.map(d => `"${d}"`).join(' ')}`,
    { cwd: repoPath, encoding: 'utf8', stdio: 'pipe' }
  );
  return result.trim().split('\n').filter(Boolean).map(line => line.slice(3));
}

function stashSpecificFiles(repoPath, files) {
  // Use git stash push with explicit file list
  const fileArgs = files.map(f => `"${f}"`).join(' ');
  execSync(`git stash push -m "aether-update-backup" -- ${fileArgs}`, {
    cwd: repoPath,
    stdio: 'pipe'
  });
}
```

**No new dependencies** - use existing Node.js built-ins

---

### Bug 4: package-lock.json Missing

**Already covered above** - generate and commit package-lock.json

---

### Bug 5: Unit Tests for syncDirWithCleanup

**Dependencies Required:**
- sinon (^17.0.0) - for mocking fs
- proxyquire (^2.1.3) - for dependency injection
- tmp (^0.2.1) - for temp directories

**Test Structure:**
```javascript
// tests/unit/sync-dir.test.js
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const tmp = require('tmp');
const fs = require('fs');
const path = require('path');

// Unit tests with mocked fs
require('./sync-dir.unit.test.js');

// Integration tests with real fs
require('./sync-dir.integration.test.js');
```

---

## Installation Summary

```bash
# Add package-lock.json (run once, commit result)
npm install

# Add testing dependencies
npm install --save-dev sinon@^17.0.0 proxyquire@^2.1.3 tmp@^0.2.1

# Verify all dependencies
npm ls
```

---

## Updated package.json Structure

```json
{
  "name": "aether-colony",
  "version": "1.1.0",
  "dependencies": {
    "commander": "^12.1.0",
    "picocolors": "^1.1.1"
  },
  "devDependencies": {
    "ava": "^6.0.0",
    "sinon": "^17.0.0",
    "proxyquire": "^2.1.3",
    "tmp": "^0.2.1"
  },
  "scripts": {
    "test": "npm run test:unit && npm run test:bash",
    "test:unit": "ava",
    "test:bash": "bash tests/bash/test-aether-utils.sh",
    "lint": "npm run lint:shell && npm run lint:json && npm run lint:sync"
  }
}
```

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Jest | Heavy, opinionated, slower than AVA | Keep AVA - already configured |
| mock-fs | Less flexible than sinon + proxyquire | sinon + proxyquire |
| nock | HTTP mocking not needed | None - no HTTP calls in CLI |
| tap | Different test style than existing | Keep AVA |

---

## Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| package-lock.json | HIGH | Standard npm practice |
| AVA testing | HIGH | Already in use, proven |
| Sinon mocking | HIGH | Industry standard |
| proxyquire | HIGH | Standard for DI testing |
| tmp | HIGH | Widely used, stable |
| Phase loop fix | MEDIUM-HIGH | Logic fix, no new deps |

---

## Sources

- Existing codebase: `/Users/callumcowie/repos/Aether/package.json` - Current stack validated
- Existing codebase: `/Users/callumcowie/repos/Aether/bin/cli.js` - syncDirWithCleanup implementation
- Existing codebase: `/Users/callumcowie/repos/Aether/test/sync-dir-hash.test.js` - Test patterns established
- npm documentation: package-lock.json for deterministic installs
- Sinon.js: Standard mocking library for JavaScript

---

*Stack research for Aether Colony v1.1 bug fixes*
*Researched: 2026-02-14*
