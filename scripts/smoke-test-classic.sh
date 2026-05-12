#!/usr/bin/env bash
# scripts/smoke-test-classic.sh
# Smoke test for Classic Aether v5.4.0
#
# Checks out v5.4.0 in an isolated git worktree, installs npm dependencies,
# runs CLI subcommands, and verifies colony state creation, state mutation
# across commands, module presence, and wrapper file content.
#
# Usage: bash scripts/smoke-test-classic.sh
# Exit: 0 on success, 1 on any test failure
set -euo pipefail

CLASSIC_TAG="v5.4.0"
WORKDIR=""
FAKE_HOME=""
PASS_COUNT=0

cleanup() {
    if [[ -n "$FAKE_HOME" && -d "$FAKE_HOME" ]]; then
        rm -rf "$FAKE_HOME" 2>/dev/null || true
    fi
    if [[ -n "$WORKDIR" && -d "$WORKDIR" ]]; then
        git worktree remove "$WORKDIR" --force 2>/dev/null || true
    fi
}
trap cleanup EXIT

pass() {
    echo "PASS: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo "FAIL: $1"
    echo "  $2"
    exit 1
}

echo "=== Classic Baseline Smoke Test (${CLASSIC_TAG}) ==="
echo ""

# --- Setup: Create isolated worktree ---
WORKDIR=$(mktemp -d)
echo "--- Setup: git worktree add ${CLASSIC_TAG} ---"
git worktree add "$WORKDIR" "$CLASSIC_TAG" --detach

# Isolate HOME to prevent v5.4.0's delegation shim from finding ~/.aether/bin/aether.
# The version-gate.js module checks for a Go binary at $HOME/.aether/bin/aether and
# delegates all commands (except install/update/setup) to it. By setting HOME to a
# temp directory, we force Classic to use its own Node.js implementations.
FAKE_HOME=$(mktemp -d)
export HOME="$FAKE_HOME"

cd "$WORKDIR"

# Install npm dependencies (node_modules not committed in the v5.4.0 tag).
# Dependencies: commander ^12.1.0, js-yaml ^4.1.0, picocolors ^1.1.1
echo "--- Setup: npm install ---"
npm install --production --silent 2>&1
echo "PASS: npm install"
PASS_COUNT=$((PASS_COUNT + 1))

# --- Test 1: CLI help ---
echo ""
echo "--- Test 1: CLI help ---"
if node bin/cli.js --help > /dev/null 2>&1; then
    pass "CLI help exits 0"
else
    fail "CLI help" "node bin/cli.js --help returned non-zero exit code"
fi

# --- Test 2: Colony init ---
echo ""
echo "--- Test 2: Colony init ---"
if node bin/cli.js init --goal "smoke-test" > /dev/null 2>&1; then
    if [[ -f .aether/data/COLONY_STATE.json ]]; then
        pass "Colony init creates COLONY_STATE.json"
    else
        fail "Colony init" ".aether/data/COLONY_STATE.json not created after init"
    fi
else
    fail "Colony init" "node bin/cli.js init --goal 'smoke-test' returned non-zero exit code"
fi

# --- Test 3: State structure ---
echo ""
echo "--- Test 3: State structure ---"
STATE_FILE=".aether/data/COLONY_STATE.json"
MISSING_FIELDS=()
for field in version current_phase events; do
    if ! grep -q "\"${field}\"" "$STATE_FILE"; then
        MISSING_FIELDS+=("$field")
    fi
done
if [[ ${#MISSING_FIELDS[@]} -eq 0 ]]; then
    pass "State structure has all required fields (version, current_phase, events)"
else
    fail "State structure" "Missing fields: ${MISSING_FIELDS[*]}"
fi

# --- Test 4: State mutation across commands (per D-04) ---
echo ""
echo "--- Test 4: State mutation (sync-state) ---"
# Capture SHA-256 hash of COLONY_STATE.json before sync-state
HASH_BEFORE=$(shasum -a 256 "$STATE_FILE" | awk '{print $1}')

if node bin/cli.js sync-state > /dev/null 2>&1; then
    # Capture hash after sync-state
    HASH_AFTER=$(shasum -a 256 "$STATE_FILE" | awk '{print $1}')

    if [[ "$HASH_BEFORE" != "$HASH_AFTER" ]]; then
        echo "  Before: ${HASH_BEFORE}"
        echo "  After:  ${HASH_AFTER}"
        pass "COLONY_STATE.json mutated by sync-state"
    else
        fail "State mutation" "COLONY_STATE.json hash unchanged after sync-state (before=${HASH_BEFORE}, after=${HASH_AFTER})"
    fi
else
    fail "State mutation" "node bin/cli.js sync-state returned non-zero exit code"
fi

# --- Test 5: Wrapper commands with ceremony content (per D-04) ---
# D-04 DEVIATION ACKNOWLEDGMENT:
# The smoke test cannot execute plan/build/continue as CLI lifecycle commands because
# Classic never implemented them as CLI subcommands -- they were slash commands only
# (Markdown wrappers executed by the AI platform). This is an architectural limitation
# of Classic, not a test gap. The test verifies the maximum that Classic can provide:
# (1) CLI init creates and mutates state, (2) sync-state causes observable state
# mutation between CLI invocations, (3) wrapper files exist with ceremony/caste
# content markers. The full plan/build/continue lifecycle test with ceremony stage
# markers in actual output belongs in Phase 108 (Golden Workflow Tests) against the
# Go runtime, where these are real subcommands.
#
# For ceremony content verification: build.md and continue.md are the lifecycle
# commands that D-04 focuses on -- they contain worker/caste/stage ceremony content.
# plan.md and init.md are verified for file existence; they use different ceremony
# patterns (plan.md references "worker" and "caste" via phase planning context;
# init.md uses "Queen" and "Colony" terminology but not the specific lifecycle
# ceremony markers).
echo ""
echo "--- Test 5: Wrapper commands ---"

# 5a: Verify all four wrapper files exist
WRAPPER_MISSING=()
for cmd in plan.md build.md continue.md init.md; do
    if [[ ! -f ".claude/commands/ant/${cmd}" ]]; then
        WRAPPER_MISSING+=("$cmd")
    fi
done
if [[ ${#WRAPPER_MISSING[@]} -eq 0 ]]; then
    pass "All 4 wrapper files exist (plan, build, continue, init)"
else
    fail "Wrapper commands" "Missing wrapper files: ${WRAPPER_MISSING[*]}"
fi

# 5b: Verify ceremony content in lifecycle commands (build, continue)
CEREMONY_PASS=true
for cmd in build.md continue.md; do
    FILE=".claude/commands/ant/${cmd}"
    FOUND_MARKERS=""
    for marker in "Stage" "worker" "caste" "Builder"; do
        if grep -qi "$marker" "$FILE"; then
            FOUND_MARKERS="${FOUND_MARKERS}${marker} "
        fi
    done
    if [[ -n "$FOUND_MARKERS" ]]; then
        echo "  ${cmd}: markers found [${FOUND_MARKERS}]"
    else
        echo "  ${cmd}: NO ceremony markers found"
        CEREMONY_PASS=false
    fi
done
if $CEREMONY_PASS; then
    pass "Lifecycle wrappers contain ceremony markers"
else
    fail "Wrapper commands" "build.md or continue.md missing ceremony markers"
fi

# --- Test 6: Module inventory ---
echo ""
echo "--- Test 6: Module inventory ---"
EXPECTED_MODULES=(
    "banner.js"
    "binary-downloader.js"
    "caste-colors.js"
    "colors.js"
    "errors.js"
    "event-types.js"
    "file-lock.js"
    "init.js"
    "interactive-setup.js"
    "logger.js"
    "nestmate-loader.js"
    "spawn-logger.js"
    "state-guard.js"
    "state-sync.js"
    "update-transaction.js"
    "version-gate.js"
)
MODULE_MISSING=()
for mod in "${EXPECTED_MODULES[@]}"; do
    if [[ ! -f "bin/lib/${mod}" ]]; then
        MODULE_MISSING+=("$mod")
    fi
done
if [[ ${#MODULE_MISSING[@]} -eq 0 ]]; then
    pass "All 16 lib modules present"
else
    fail "Module inventory" "Missing modules: ${MODULE_MISSING[*]}"
fi

# --- Test 7: Version output ---
echo ""
echo "--- Test 7: Version output ---"
VERSION_OUTPUT=$(node bin/cli.js version 2>&1)
if [[ $? -eq 0 && -n "$VERSION_OUTPUT" ]]; then
    echo "  Version: ${VERSION_OUTPUT}"
    pass "Version command outputs content"
else
    fail "Version output" "node bin/cli.js version returned non-zero or empty output"
fi

# --- Test 8: Status command ---
echo ""
echo "--- Test 8: Status command ---"
if node bin/cli.js status > /dev/null 2>&1; then
    pass "Status command exits 0 (colony initialized)"
else
    fail "Status command" "node bin/cli.js status returned non-zero exit code"
fi

# --- Summary ---
echo ""
echo "=== All smoke tests passed (${PASS_COUNT}/8) ==="
