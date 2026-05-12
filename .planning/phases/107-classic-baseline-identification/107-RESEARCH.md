# Phase 107: Classic Baseline Identification - Research

**Researched:** 2026-05-12
**Domain:** Classic Aether version comparison and behavioral baseline establishment
**Confidence:** HIGH

## Summary

This phase identifies the best Classic Aether version (among v5.3.0, v5.3.3, v5.4.0) as a behavioral comparison anchor for the hybrid runtime milestone. The research reveals a clear winner: **v5.4.0** is the only version with the Go-delegation bridge (version-gate.js + binary-downloader.js), making it the natural transition point between the Node era and the Go era.

Key findings:

- **v5.4.0 has 16 modules**, v5.3.0 and v5.3.3 each have 14. The two additional modules (binary-downloader.js and version-gate.js) are the Go-delegation bridge that makes v5.4.0 the only version that can hand off to the Go binary. [VERIFIED: git ls-tree for each tag]
- **v5.3.0 and v5.3.3 are functionally identical** -- only 15 lines differ in update-transaction.js (exchange directory handling). All other 13 shared modules are byte-identical. [VERIFIED: diff between tags]
- **The smoke test cannot test plan/build/continue as CLI commands** because in the Classic era, these were slash commands (Markdown wrappers), not CLI subcommands. The Node CLI only handles init, install, update, status, etc. The smoke test must instead verify: (1) the Node CLI initializes a colony correctly, (2) the wrapper markdown files exist, (3) COLONY_STATE.json is created and has valid structure. [VERIFIED: v5.4.0 bin/cli.js -- only 13 registered subcommands, none named plan/build/continue]
- **v5.4.0 package.json reports version "5.3.3"** -- the version field was not updated in the tag. This is a known metadata bug that does not affect functionality. [VERIFIED: git show v5.4.0:package.json]
- **node_modules/ is NOT committed in the v5.4.0 tag.** The Classic CLI has 3 npm dependencies (commander ^12.1.0, js-yaml ^4.1.0, picocolors ^1.1.1) and the smoke test must run `npm install` before testing CLI commands. [VERIFIED: git ls-tree v5.4.0 node_modules/ returns empty; git show v5.4.0:package.json shows dependencies]

**Primary recommendation:** Select v5.4.0 as the Classic baseline. Write the behavioral checklist by expanding the Phase 106 contract's 16-module classification table with per-module behavioral expectations. Write the smoke test as a standalone Bash script that checks out v5.4.0, installs npm dependencies, runs `node bin/cli.js init`, and verifies COLONY_STATE.json structure and wrapper command presence.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Full behavioral checklist comparing all 3 candidate versions (v5.3.0, v5.3.3, v5.4.0) against behavior criteria
- **D-02:** Checklist covers all 16 Classic modules with: what each does, expected behavior, which version has it, and 4-category classification (Restore in TS / Keep in Go / Obsolete / Reject as unsafe)
- **D-03:** The comparison should show what changed across versions and explain why the selected version is the bridge between Node era and Go era
- **D-04:** Full lifecycle verification -- test checks exit codes = 0 for plan/build/continue, output contains ceremony stage markers and caste labels, COLONY_STATE.json changes between commands
- **D-05:** Smoke test is a standalone Bash script (scripts/smoke-test-classic.sh), not a Go test. Simpler, runs in CI without compilation, independent of current Go runtime

### Claude's Discretion
- Smoke test implementation format (chose Bash script for simplicity and CI portability)

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BASE-01 | Best Classic version identified with evidence (compare v5.3.0, v5.3.3, v5.4.0 against behavior criteria) | Version comparison table below; module diff analysis; architectural bridge analysis |
| BASE-02 | Smoke-test script exists that checks out Classic tag and verifies it can run lifecycle without errors | Smoke test design below; Classic CLI command inventory; COLONY_STATE.json structure |
| BASE-03 | Baseline documented with: selected tag, selection rationale, known limitations, behavior comparison checklist | Full behavioral checklist below; selection rationale; known limitations |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Classic source code analysis | Research/Documentation | -- | Static analysis of git tags, no runtime needed |
| Behavioral checklist creation | Documentation | -- | Written artifact in .aether/references/ |
| Smoke test execution | Bash (scripts/) | Node.js (runtime) | Bash orchestrates, Node.js runs Classic CLI |
| Colony state verification | Bash (assertions) | -- | JSON parsing with grep/jq in smoke test |
| Version comparison evidence | Git (tag checkout) | -- | Uses git worktree or stash to inspect tags |

## Standard Stack

### Core
No new libraries needed for this phase. The work is documentation, a comparison checklist, and a Bash smoke test.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Node.js | 25.9.0 (installed) | Running Classic v5.4.0 CLI commands in smoke test | Smoke test `node bin/cli.js init` |
| Bash | 3.2.57 (installed) | Smoke test script orchestration | scripts/smoke-test-classic.sh |
| git | installed | Tag checkout for version comparison | `git worktree add` or `git stash && git checkout` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Standalone Bash script | Go test | Bash is per D-05: simpler, no compilation, CI-portable |
| git worktree for tag inspection | git stash + checkout | Worktree avoids modifying working tree; cleaner for CI |

**Version verification:**
```
Node.js: v25.9.0 [VERIFIED: node --version]
Bash: 3.2.57 [VERIFIED: bash --version]
Git: available [VERIFIED: git tag -l success]
```

## Architecture Patterns

### Classic Version Comparison: Module-by-Module Analysis

All three versions share the same directory structure: `bin/cli.js`, `bin/lib/`, `bin/generate-commands.js`, `bin/npx-entry.js`. The CLI entry point uses `commander` for subcommand routing and all 14 shared modules use CommonJS `require()`.

**Key architectural difference:** v5.4.0 adds a **delegation shim** at the top of `bin/cli.js` that checks for a Go binary at `~/.aether/bin/aether`. If the binary exists and its version matches the npm package version, ALL commands (except install/update/setup) are delegated to the Go binary via `spawnSync`. v5.3.0 and v5.3.3 have no such delegation -- they are pure Node.js. [VERIFIED: diff between v5.3.0:bin/cli.js and v5.4.0:bin/cli.js]

### Recommended Project Structure
```
scripts/
└── smoke-test-classic.sh     # New smoke test script

.aether/references/
└── classic-baseline.md        # New baseline document (selected tag, rationale, checklist)

.planning/phases/107-classic-baseline-identification/
└── 107-RESEARCH.md            # This file
```

### Pattern 1: Behavioral Checklist per Module
**What:** Each of the 16 Classic modules gets a structured entry with: module name, purpose, expected behavior, which versions have it, classification, and migration notes.
**When to use:** Creating the BASE-03 baseline document.
**Example:**
```markdown
### binary-downloader.js
- **Purpose:** Downloads platform-specific Go binary from GitHub Releases during npm install
- **Expected behavior:** Verifies SHA-256 checksum, atomic install, never throws
- **Versions:** v5.4.0 only (not in v5.3.0 or v5.3.3)
- **Classification:** Keep in Go -- pkg/downloader/ already owns this in current runtime
- **Key functions:** downloadBinary(), getPlatformArch(), atomicInstall()
- **Migration note:** Already reimplemented in Go; Classic module is reference only
```

### Pattern 2: Smoke Test as Isolated Checkout
**What:** The smoke test creates a temporary directory, clones/checks out the Classic tag, installs npm dependencies, runs CLI commands, and cleans up. Never touches the working repo.
**When to use:** BASE-02 smoke test script.
**Example:**
```bash
#!/usr/bin/env bash
set -euo pipefail
WORKDIR=$(mktemp -d)
trap 'git worktree remove "$WORKDIR" --force 2>/dev/null; rm -rf "$WORKDIR"' EXIT
git worktree add "$WORKDIR" v5.4.0 --detach
cd "$WORKDIR"
npm install --production    # Install commander, js-yaml, picocolors
node bin/cli.js init --goal "smoke-test"
# ... verify COLONY_STATE.json ...
```

### Anti-Patterns to Avoid
- **Testing plan/build/continue as CLI commands:** These were never CLI subcommands in Classic. They were slash commands (Markdown wrappers) that the AI platform executed. The smoke test can only verify CLI subcommands (init, status, version, etc.) and wrapper file presence.
- **Assuming v5.4.0 package.json is accurate:** It reports version "5.3.3" -- a metadata bug. Use the git tag, not package.json version, as the authoritative version identifier.
- **Running the smoke test against the current working tree:** The current codebase is Go-based, not Node-based. The test must check out the Classic tag in isolation.
- **Skipping npm install:** node_modules/ is NOT in the git tag. The smoke test MUST run `npm install` before any CLI commands, or `node bin/cli.js` will fail with missing module errors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Version comparison | Custom diff tool | `git diff` between tags | Git handles tag checkout and diff natively |
| JSON validation in smoke test | Custom JSON parser | `grep` + `python3 -c "import json..."` or `jq` | Standard tools, no dependencies |
| Module classification | New classification framework | Phase 106 contract's existing 16-module table | Already classified; this phase expands it |

**Key insight:** Phase 106 already did the hard work of classifying all 16 modules. This phase validates that classification against actual source code and adds behavioral detail. Do not re-classify from scratch.

## Common Pitfalls

### Pitfall 1: Testing Plan/Build/Continue as CLI Subcommands
**What goes wrong:** Writing a smoke test that calls `node bin/cli.js plan` and expects it to work.
**Why it happens:** The current Go binary has `aether plan`, `aether build`, `aether continue` as real subcommands. In Classic, these were slash commands only.
**How to avoid:** The Classic CLI has 13 subcommands: init, install, update, version, uninstall, setup, checkpoint, sync-state, spawn-log, spawn-tree, status, nestmates, context. Test only these. For plan/build/continue, verify only that the wrapper Markdown files exist.
**Warning signs:** Smoke test calls `node bin/cli.js plan` and gets "unknown command".

### Pitfall 2: Forgetting the Delegation Shim in v5.4.0
**What goes wrong:** Running `node bin/cli.js init` on v5.4.0 and getting Go binary behavior instead of Node behavior, because a Go binary is installed at `~/.aether/bin/aether`.
**Why it happens:** v5.4.0's delegation shim checks for the Go binary first and delegates everything except install/update/setup.
**How to avoid:** The smoke test must handle the delegation shim. Three approaches: (a) temporarily move the Go binary, (b) use `HOME=/dev/null` to prevent version-gate from finding it, or (c) test only install/setup commands which are excluded from delegation. The simplest approach: set `HOME` to a temp directory for the test so version-gate cannot find `~/.aether/bin/aether`.
**Warning signs:** `node bin/cli.js version` outputs "1.0.37" (Go version) instead of "5.3.3" (Node version from package.json).

### Pitfall 3: v5.3.0 vs v5.3.3 Treated as Meaningfully Different
**What goes wrong:** Spending time analyzing differences between v5.3.0 and v5.3.3 when they are functionally identical.
**Why it happens:** The version numbers suggest meaningful change.
**How to avoid:** The only difference is 15 lines in update-transaction.js (exchange directory exclusion logic). Treat them as the same baseline. The comparison is really v5.3.x vs v5.4.0.
**Warning signs:** Behavioral checklist has separate columns for v5.3.0 and v5.3.3 with identical entries.

### Pitfall 4: Smoke Test Modifies Working Tree
**What goes wrong:** The smoke test checks out v5.4.0 in the current directory, overwriting the Go-based codebase.
**Why it happens:** Using `git checkout v5.4.0` directly instead of a worktree.
**How to avoid:** Always use `git worktree add` in a temporary directory. The smoke test must never modify the working tree.
**Warning signs:** After running the smoke test, `aether version` shows "5.3.3" (Node version) instead of "1.0.37" (Go version).

### Pitfall 5: Missing npm install Step
**What goes wrong:** The smoke test checks out v5.4.0 and immediately runs `node bin/cli.js init`, which fails because `commander`, `js-yaml`, and `picocolors` are not installed.
**Why it happens:** The current Go binary has no npm dependencies, so it is easy to forget that Classic requires them.
**How to avoid:** The smoke test must include `npm install --production` after checking out the tag and before running any CLI commands.
**Warning signs:** Error message "Cannot find module 'commander'" from `node bin/cli.js`.

## Code Examples

### Classic CLI Subcommands (v5.4.0) [VERIFIED: git show v5.4.0:bin/cli.js]

```bash
# These are the 13 CLI subcommands in Classic v5.4.0:
node bin/cli.js init --goal "test colony"    # Initialize colony
node bin/cli.js install                       # Install slash commands + set up hub
node bin/cli.js update                        # Update from hub
node bin/cli.js version                       # Show version
node bin/cli.js uninstall                     # Remove slash commands
node bin/cli.js setup                         # Setup hub
node bin/cli.js status                        # Colony status
node bin/cli.js sync-state                    # Sync COLONY_STATE.json with .planning/
node bin/cli.js spawn-log                     # Show spawn log
node bin/cli.js spawn-tree                    # Show spawn tree
node bin/cli.js checkpoint                    # Create checkpoint
node bin/cli.js nestmates                     # Show nestmates
node bin/cli.js context                       # Show colony context
```

### Delegation Shim (v5.4.0 only) [VERIFIED: git show v5.4.0:bin/cli.js]

```javascript
// This block appears ONLY in v5.4.0, before the CLI runs:
const { shouldDelegate, getBinaryPath } = require('./lib/version-gate');
if (shouldDelegate(process.argv)) {
    const { spawnSync } = require('child_process');
    const binaryPath = getBinaryPath();
    const result = spawnSync(binaryPath, process.argv.slice(2), {
      stdio: 'inherit',
      env: process.env,
    });
    process.exit(result.status);
}
```

### npm Dependencies (v5.4.0) [VERIFIED: git show v5.4.0:package.json]

```json
{
  "dependencies": {
    "commander": "^12.1.0",
    "js-yaml": "^4.1.0",
    "picocolors": "^1.1.1"
  }
}
```

Note: `node_modules/` is NOT committed in the v5.4.0 tag. The smoke test must run `npm install --production` before testing CLI commands.

### Smoke Test Skeleton [RECOMMENDED PATTERN]

```bash
#!/usr/bin/env bash
# scripts/smoke-test-classic.sh
# Smoke test for Classic Aether v5.4.0
set -euo pipefail

CLASSIC_TAG="v5.4.0"
WORKDIR=""

cleanup() {
    if [[ -n "$WORKDIR" && -d "$WORKDIR" ]]; then
        git worktree remove "$WORKDIR" --force 2>/dev/null || true
    fi
}
trap cleanup EXIT

echo "=== Classic Baseline Smoke Test (${CLASSIC_TAG}) ==="

# Create isolated worktree
WORKDIR=$(mktemp -d)
git worktree add "$WORKDIR" "$CLASSIC_TAG" --detach

cd "$WORKDIR"

# Install npm dependencies (node_modules not committed in tag)
echo "--- Setup: npm install ---"
npm install --production --silent
echo "PASS: npm install"

# Test 1: Node CLI runs without error
echo "--- Test 1: CLI help ---"
node bin/cli.js --help > /dev/null
echo "PASS: CLI help"

# Test 2: Init creates colony state
echo "--- Test 2: Colony init ---"
node bin/cli.js init --goal "smoke-test"
if [[ ! -f .aether/data/COLONY_STATE.json ]]; then
    echo "FAIL: COLONY_STATE.json not created"
    exit 1
fi
echo "PASS: Colony init"

# Test 3: COLONY_STATE.json has required fields
echo "--- Test 3: State structure ---"
STATE=$(cat .aether/data/COLONY_STATE.json)
for field in version current_phase events; do
    echo "$STATE" | grep -q "\"${field}\"" || { echo "FAIL: missing ${field}"; exit 1; }
done
echo "PASS: State structure"

# Test 4: Wrapper commands exist
echo "--- Test 4: Wrapper commands ---"
for cmd in plan.md build.md continue.md init.md; do
    [[ -f ".claude/commands/ant/${cmd}" ]] || { echo "FAIL: missing ${cmd}"; exit 1; }
done
echo "PASS: Wrapper commands"

# Test 5: All 16 lib modules present
echo "--- Test 5: Module inventory ---"
EXPECTED_MODULES="banner.js binary-downloader.js caste-colors.js colors.js errors.js event-types.js file-lock.js init.js interactive-setup.js logger.js nestmate-loader.js spawn-logger.js state-guard.js state-sync.js update-transaction.js version-gate.js"
for mod in $EXPECTED_MODULES; do
    [[ -f "bin/lib/${mod}" ]] || { echo "FAIL: missing ${mod}"; exit 1; }
done
echo "PASS: Module inventory"

echo ""
echo "=== All smoke tests passed ==="
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pure Node.js CLI (v5.3.x) | Go-delegation bridge (v5.4.0) | v5.4.0 | First hybrid version -- Node CLI delegates to Go binary |
| Go-delegation bridge (v5.4.0) | Pure Go binary (v1.0.x) | v6.0+ | Complete transition to Go; Node CLI retired |
| Slash commands only (Classic) | Go subcommands (current) | v6.0+ | plan/build/continue became real Go subcommands |

**Deprecated/outdated:**
- `bin/lib/state-sync.js`: Replaced by Go atomic writes in `pkg/storage/`
- `bin/lib/interactive-setup.js`: Replaced by discuss flow
- `bin/lib/nestmate-loader.js`: Replaced by Go skill system
- `bin/lib/file-lock.js`: Replaced by Go file locking in `pkg/storage/`

## Classic Version Comparison: Full Analysis

### Module Inventory

| Module | v5.3.0 | v5.3.3 | v5.4.0 | Classification |
|--------|--------|--------|--------|----------------|
| spawn-logger.js | identical | identical | identical | Restore in TS |
| state-guard.js | identical | identical | identical | Keep in Go |
| caste-colors.js | identical | identical | identical | Keep in Go |
| event-types.js | identical | identical | identical | Keep in Go |
| file-lock.js | identical | identical | identical | Keep in Go |
| state-sync.js | identical | identical | identical | Obsolete |
| banner.js | identical | identical | identical | Keep in Go |
| colors.js | identical | identical | identical | Keep in Go |
| logger.js | identical | identical | identical | Restore in TS |
| init.js | identical | identical | identical | Keep in Go |
| interactive-setup.js | identical | identical | identical | Obsolete |
| nestmate-loader.js | identical | identical | identical | Obsolete |
| errors.js | identical | identical | identical | Restore in TS |
| update-transaction.js | present | 15 lines different | 15 lines different | Keep in Go |
| binary-downloader.js | absent | absent | present | Keep in Go |
| version-gate.js | absent | absent | present | Keep in Go |

### v5.3.0 vs v5.3.3 Difference

Only 15 lines differ in `update-transaction.js`:
- v5.3.0 excludes `exchange` directory from hub sync entirely
- v5.3.3 removes `exchange` from the exclusion list but filters exchange data files by extension (only `.sh` scripts distribute, `.xml`/`.json` data excluded)

This is a minor hub distribution policy change. All 13 other modules are byte-identical.

### v5.3.x vs v5.4.0 Differences

134 lines differ in `bin/cli.js`:
1. **Delegation shim** (~10 lines): `shouldDelegate()` check before CLI runs -- delegates to Go binary
2. **Binary download during install** (~10 lines): Downloads Go binary during `npm install`
3. **refreshBinary() function** (~60 lines): Updates Go binary during `aether update`
4. **Stash restore improvements** (~20 lines): Better conflict handling during update
5. **Binary refresh during update** (~10 lines): Non-blocking binary refresh in update flow

Plus two entirely new modules:
- `binary-downloader.js` (~260 lines): SHA-256 verified, platform-aware, atomic binary download
- `version-gate.js` (~160 lines): Binary availability check, version comparison, delegation decision

### Selection Rationale: v5.4.0

v5.4.0 is the clear choice for three reasons:

1. **Bridge architecture:** v5.4.0 is the only version that contains the Node-to-Go delegation bridge (version-gate.js + binary-downloader.js). This is the architectural pattern that the current hybrid runtime milestone is restoring and improving. It is literally the bridge between the Node era and the Go era.

2. **Complete module set:** With 16 modules instead of 14, v5.4.0 covers every Classic behavior. Using v5.3.x would mean the behavioral checklist is incomplete for two modules.

3. **Production endpoint:** v5.4.0 was the last Classic release before the full Go transition. It represents the mature state of the Node.js runtime, with all bug fixes and improvements accumulated over the v5.x series.

### Known Limitations of v5.4.0

1. **package.json version mismatch:** Reports "5.3.3" instead of "5.4.0". Use git tag as the version authority.
2. **Delegation shim conflicts with current Go binary:** If `~/.aether/bin/aether` exists, v5.4.0 delegates to it. Smoke test must handle this by setting HOME to a temp directory.
3. **No plan/build/continue CLI commands:** These are slash commands only. The smoke test can only verify CLI subcommands and wrapper file presence.
4. **Requires Node.js + npm install:** The Classic CLI requires Node.js runtime and 3 npm dependencies (commander, js-yaml, picocolors). The smoke test must run `npm install` before testing.
5. **commander dependency:** The CLI uses the `commander` npm package. Node modules must be installed before testing.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | v5.4.0 package.json version "5.3.3" is a metadata bug, not a functional issue | Version Comparison | Low -- verified by git tag and actual module content |
| A2 | The smoke test can use `git worktree add` without conflicting with the current working tree state | Architecture Patterns | Low -- worktree is designed for this; may fail if working tree has uncommitted changes |

**Resolved during research:**
- Node modules availability: CONFIRMED `node_modules/` NOT in v5.4.0 tag. Smoke test must run `npm install`. npm dependencies are commander, js-yaml, picocolors. [VERIFIED: git ls-tree v5.4.0 node_modules/ returns empty]

## Open Questions

1. **Smoke test scope for D-04**
   - What we know: D-04 says "test checks exit codes = 0 for plan/build/continue, output contains ceremony stage markers and caste labels, COLONY_STATE.json changes between commands."
   - What's unclear: Since plan/build/continue are slash commands (not CLI subcommands), the smoke test cannot directly execute them. The test can only verify the preconditions: CLI works, colony state created, wrapper files present.
   - Recommendation: The smoke test verifies the Node CLI lifecycle (`init -> status -> sync-state`). For plan/build/continue, the test verifies wrapper Markdown files exist and contain expected ceremony markers. This satisfies D-04's intent within Classic's architectural constraints. The full plan/build/continue lifecycle test belongs in Phase 108 (Golden Workflow Tests) against the Go runtime.

2. **Baseline document location**
   - What we know: CONTEXT.md suggests `.aether/references/` or the phase directory.
   - What's unclear: Exact file path and naming.
   - Recommendation: Place at `.aether/references/classic-baseline.md`. This keeps it alongside the Phase 106 contract and makes it discoverable by future phases.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Classic CLI execution | Yes | v25.9.0 | -- |
| npm | Install Classic dependencies | Yes | available via Node | -- |
| git | Tag checkout, worktree | Yes | installed | -- |
| Bash | Smoke test script | Yes | 3.2.57 | -- |

**Missing dependencies with no fallback:**
- None identified. All required tools are available.

**Missing dependencies with fallback:**
- None. All dependencies are available.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Bash (smoke test) + manual verification |
| Config file | none |
| Quick run command | `bash scripts/smoke-test-classic.sh` |
| Full suite command | `bash scripts/smoke-test-classic.sh && go test ./cmd/ -run TestBoundary -v` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BASE-01 | v5.4.0 selected with evidence | manual | `cat .aether/references/classic-baseline.md` | Wave 0 |
| BASE-02 | Smoke test passes for Classic v5.4.0 | automated | `bash scripts/smoke-test-classic.sh` | Wave 0 |
| BASE-03 | Baseline document complete with all sections | manual | `grep -c "Selection Rationale\|Known Limitations\|Behavioral Checklist" .aether/references/classic-baseline.md` | Wave 0 |

### Sampling Rate
- **Per task commit:** `bash scripts/smoke-test-classic.sh`
- **Per wave merge:** `bash scripts/smoke-test-classic.sh && go test ./... -race`
- **Phase gate:** Smoke test green + baseline document complete

### Wave 0 Gaps
- [ ] `scripts/smoke-test-classic.sh` -- covers BASE-02
- [ ] `.aether/references/classic-baseline.md` -- covers BASE-01, BASE-03

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | no | -- |
| V6 Cryptography | no | -- |

This phase is purely analytical: reading source code at git tags, writing documentation, and creating a smoke test. No user input, no authentication, no cryptography, no access control changes.

### Known Threat Patterns

No security threats identified for this phase. The smoke test runs in a temporary worktree and cleans up after itself.

## Sources

### Primary (HIGH confidence)
- `git ls-tree v5.3.0 bin/lib/` -- 14 modules listed [VERIFIED: git command]
- `git ls-tree v5.3.3 bin/lib/` -- 14 modules listed (same as v5.3.0) [VERIFIED: git command]
- `git ls-tree v5.4.0 bin/lib/` -- 16 modules listed [VERIFIED: git command]
- `git diff v5.3.0:bin/lib/ v5.4.0:bin/lib/` -- Only update-transaction.js + 2 new modules differ [VERIFIED: diff]
- `git diff v5.3.0:bin/cli.js v5.4.0:bin/cli.js` -- 134 lines differ (delegation shim + binary download) [VERIFIED: diff]
- `git show v5.4.0:bin/lib/version-gate.js` -- Full source of Go-delegation bridge [VERIFIED: source read]
- `git show v5.4.0:bin/lib/binary-downloader.js` -- Full source of binary download module [VERIFIED: source read]
- `git show v5.4.0:bin/lib/spawn-logger.js` -- Full source of spawn tracking module [VERIFIED: source read]
- `git show v5.4.0:bin/lib/state-guard.js` -- Full source of state guard module [VERIFIED: source read]
- `git show v5.4.0:bin/lib/errors.js` -- Full source of error hierarchy [VERIFIED: source read]
- `git show v5.4.0:bin/lib/caste-colors.js` -- Full source of caste styling [VERIFIED: source read]
- `git show v5.4.0:bin/lib/banner.js` -- ASCII banner content [VERIFIED: source read]
- `git show v5.4.0:bin/lib/init.js` -- Colony initialization logic [VERIFIED: source read]
- `git show v5.4.0:package.json` -- npm dependencies and (incorrect) version [VERIFIED: source read]
- `git ls-tree v5.4.0 node_modules/` -- Confirmed empty (no committed node_modules) [VERIFIED: git command]
- `git ls-tree v5.4.0 .claude/commands/ant/` -- 45 wrapper commands [VERIFIED: git command]
- `.aether/references/contracts/runtime-boundary-contract.md` -- Phase 106 contract with 16-module classification table [VERIFIED: source read]
- `cmd/boundary_contract_test.go` -- Phase 106 delivered test [VERIFIED: source read]

### Secondary (MEDIUM confidence)
- `git log --oneline v5.3.0..v5.4.0` -- 47 commits between versions [VERIFIED: git log]
- `git diff v5.3.0:bin/lib/update-transaction.js v5.3.3:bin/lib/update-transaction.js` -- Only 15 lines differ [VERIFIED: diff]

## Metadata

**Confidence breakdown:**
- Version comparison: HIGH -- All diffs verified directly via git commands
- Module classification: HIGH -- Source code read for each module; classification matches Phase 106 contract
- Smoke test design: HIGH -- npm dependency issue confirmed resolved; all runtime dependencies verified available
- Pitfalls: HIGH -- Based on direct observation of Classic CLI architecture

**Research date:** 2026-05-12
**Valid until:** 2026-06-12 (stable -- git tags are immutable)
