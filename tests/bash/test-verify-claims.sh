#!/usr/bin/env bash
# Verify Claims Integration Tests
# Tests verify-claims subcommand for detecting fabricated worker claims (QUAL-08)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with builder claims
# ============================================================================
setup_verify_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"
    mkdir -p "$tmp_dir/src"

    # Copy aether-utils and dependencies
    cp "$AETHER_UTILS" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Create a minimal COLONY_STATE.json (some subcommands may reference it)
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "Test verify-claims",
  "state": "EXECUTING",
  "current_phase": 1
}
EOF

    echo "$tmp_dir"
}

cleanup_verify_env() {
    rm -rf "$1"
}

# ============================================================================
# Test 1: All files exist + test exit 0 + watcher passed = "passed"
# ============================================================================
test_start "verify-claims: clean pass (all files exist, tests pass, watcher agrees)"

TMP=$(setup_verify_env)

# Create real files that the builder "claims" to have created
touch "$TMP/src/auth.ts"
touch "$TMP/src/types.ts"

# Write builder claims referencing those files
cat > "$TMP/.aether/data/last-build-claims.json" << EOF
{
  "files_created": ["$TMP/src/auth.ts"],
  "files_modified": ["$TMP/src/types.ts"],
  "build_phase": 1,
  "timestamp": "2026-01-01T00:00:00Z"
}
EOF

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/last-build-claims.json" \
  '{"verification_passed":true}' \
  0 2>/dev/null)

if assert_ok_true "$OUTPUT" && \
   assert_json_field_equals "$OUTPUT" ".result.verification_status" "passed" && \
   assert_json_field_equals "$OUTPUT" ".result.blocked" "false" && \
   assert_json_field_equals "$OUTPUT" ".result.checks_run" "2"; then
    test_pass
else
    test_fail "verification_status=passed, blocked=false" "$(echo "$OUTPUT" | jq '.result')"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Test 2: Missing file + test exit 0 + watcher passed = "blocked"
# ============================================================================
test_start "verify-claims: missing file blocks verification"

TMP=$(setup_verify_env)

# Only create one of two claimed files
touch "$TMP/src/auth.ts"

cat > "$TMP/.aether/data/last-build-claims.json" << EOF
{
  "files_created": ["$TMP/src/auth.ts", "$TMP/src/nonexistent.ts"],
  "files_modified": [],
  "build_phase": 1,
  "timestamp": "2026-01-01T00:00:00Z"
}
EOF

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/last-build-claims.json" \
  '{"verification_passed":true}' \
  0 2>/dev/null)

if assert_ok_true "$OUTPUT" && \
   assert_json_field_equals "$OUTPUT" ".result.verification_status" "blocked" && \
   assert_json_field_equals "$OUTPUT" ".result.blocked" "true" && \
   assert_contains "$(echo "$OUTPUT" | jq -r '.result.mismatches[0].type')" "missing_file"; then
    test_pass
else
    test_fail "verification_status=blocked with missing_file mismatch" "$(echo "$OUTPUT" | jq '.result')"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Test 3: All files exist + test exit 1 + watcher passed true = "blocked"
# ============================================================================
test_start "verify-claims: test exit code mismatch blocks verification"

TMP=$(setup_verify_env)

touch "$TMP/src/auth.ts"

cat > "$TMP/.aether/data/last-build-claims.json" << EOF
{
  "files_created": ["$TMP/src/auth.ts"],
  "files_modified": [],
  "build_phase": 1,
  "timestamp": "2026-01-01T00:00:00Z"
}
EOF

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/last-build-claims.json" \
  '{"verification_passed":true}' \
  1 2>/dev/null)

if assert_ok_true "$OUTPUT" && \
   assert_json_field_equals "$OUTPUT" ".result.verification_status" "blocked" && \
   assert_json_field_equals "$OUTPUT" ".result.blocked" "true" && \
   assert_contains "$(echo "$OUTPUT" | jq -r '.result.mismatches[0].type')" "test_mismatch"; then
    test_pass
else
    test_fail "verification_status=blocked with test_mismatch" "$(echo "$OUTPUT" | jq '.result')"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Test 4: All files exist + test exit 0 + watcher passed false = "passed"
# (watcher saying fail when tests pass is conservative, not fabrication)
# ============================================================================
test_start "verify-claims: conservative watcher (tests pass but watcher says fail) is not blocked"

TMP=$(setup_verify_env)

touch "$TMP/src/auth.ts"

cat > "$TMP/.aether/data/last-build-claims.json" << EOF
{
  "files_created": ["$TMP/src/auth.ts"],
  "files_modified": [],
  "build_phase": 1,
  "timestamp": "2026-01-01T00:00:00Z"
}
EOF

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/last-build-claims.json" \
  '{"verification_passed":false}' \
  0 2>/dev/null)

if assert_ok_true "$OUTPUT" && \
   assert_json_field_equals "$OUTPUT" ".result.verification_status" "passed" && \
   assert_json_field_equals "$OUTPUT" ".result.blocked" "false"; then
    test_pass
else
    test_fail "verification_status=passed (conservative watcher not a problem)" "$(echo "$OUTPUT" | jq '.result')"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Test 5: No builder claims file = graceful handling (pass, not crash)
# ============================================================================
test_start "verify-claims: no builder claims file does not crash or block"

TMP=$(setup_verify_env)

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/nonexistent-claims.json" \
  '{"verification_passed":true}' \
  0 2>/dev/null)

if assert_ok_true "$OUTPUT" && \
   assert_json_field_equals "$OUTPUT" ".result.verification_status" "passed" && \
   assert_json_field_equals "$OUTPUT" ".result.blocked" "false"; then
    test_pass
else
    test_fail "verification_status=passed (graceful no-claims handling)" "$(echo "$OUTPUT" | jq '.result')"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Test 6: Summary message is a plain one-liner (locked decision)
# ============================================================================
test_start "verify-claims: summary is a plain one-liner on failure"

TMP=$(setup_verify_env)

cat > "$TMP/.aether/data/last-build-claims.json" << EOF
{
  "files_created": ["$TMP/src/does-not-exist.ts"],
  "files_modified": [],
  "build_phase": 1,
  "timestamp": "2026-01-01T00:00:00Z"
}
EOF

OUTPUT=$(cd "$TMP" && bash .aether/aether-utils.sh verify-claims \
  "$TMP/.aether/data/last-build-claims.json" \
  '{"verification_passed":true}' \
  0 2>/dev/null)

SUMMARY=$(echo "$OUTPUT" | jq -r '.result.summary')

# Summary should be a single line (no newlines)
LINECOUNT=$(echo "$SUMMARY" | wc -l | tr -d ' ')
# Summary should contain "Blocked" when there are mismatches
if [[ "$LINECOUNT" -eq 1 ]] && assert_contains "$SUMMARY" "Blocked"; then
    test_pass
else
    test_fail "single-line summary containing 'Blocked'" "lines=$LINECOUNT summary=$SUMMARY"
fi

cleanup_verify_env "$TMP"

# ============================================================================
# Results
# ============================================================================
test_summary
