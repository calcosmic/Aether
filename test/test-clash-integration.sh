#!/usr/bin/env bash
# Test: Clash detection integration with aether-utils.sh
# Tests that the full wiring works: sourcing, dispatch, hook install, merge driver.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

PASS=0
FAIL=0

pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

echo "=== Clash Detection Integration Tests ==="
echo ""

# --- Test 1: clash-detect.sh is sourced by aether-utils.sh ---
echo "1. _clash_detect function is available after sourcing aether-utils.sh"
if bash -c "source '$AETHER_UTILS' 2>/dev/null; type _clash_detect" &>/dev/null; then
    pass "_clash_detect function available"
else
    fail "_clash_detect function not found after sourcing aether-utils.sh"
fi

# --- Test 2: worktree.sh is sourced by aether-utils.sh ---
echo "2. _worktree_create function is available after sourcing aether-utils.sh"
if bash -c "source '$AETHER_UTILS' 2>/dev/null; type _worktree_create" &>/dev/null; then
    pass "_worktree_create function available"
else
    fail "_worktree_create function not found after sourcing aether-utils.sh"
fi

# --- Test 3: clash-detect dispatch returns valid JSON ---
echo "3. clash-detect dispatch returns valid JSON"
result=$(bash "$AETHER_UTILS" clash-detect --file some-file.txt 2>/dev/null)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "clash-detect returns valid JSON with no conflict"
else
    fail "clash-detect returned unexpected output: $result"
fi

# --- Test 4: clash-check alias returns valid JSON ---
echo "4. clash-check alias dispatch returns valid JSON"
result=$(bash "$AETHER_UTILS" clash-check --file some-file.txt 2>/dev/null)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "clash-check alias returns valid JSON"
else
    fail "clash-check alias returned unexpected output: $result"
fi

# --- Test 5: clash-setup --install writes hook to settings.json ---
echo "5. clash-setup --install writes hook to settings.json"
TMP_SETTINGS=$(mktemp -t clash-integration-settings)
echo '{}' > "$TMP_SETTINGS"
result=$(CLASH_SETTINGS_PATH="$TMP_SETTINGS" bash "$AETHER_UTILS" clash-setup --install 2>/dev/null)
if echo "$result" | jq -e '.ok == true and .result.hook_installed == true' >/dev/null 2>&1; then
    pass "clash-setup --install returns success"
else
    fail "clash-setup --install failed: $result"
fi
# Verify settings.json has the hook with correct structure
hook_count=$(jq '[.hooks.PreToolUse[]?.hooks[]?.command | select(contains("clash-pre-tool-use"))] | length' "$TMP_SETTINGS" 2>/dev/null)
if [[ "$hook_count" -ge 1 ]]; then
    pass "Hook entry found in settings.json"
else
    fail "Hook entry not found in settings.json (count=$hook_count)"
fi
rm -f "$TMP_SETTINGS"

# --- Test 6: clash-setup --uninstall removes hook ---
echo "6. clash-setup --uninstall removes hook from settings.json"
TMP_SETTINGS=$(mktemp -t clash-integration-settings)
cat > "$TMP_SETTINGS" << 'SETTINGS_EOF'
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "node .aether/utils/hooks/clash-pre-tool-use.js",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
SETTINGS_EOF
result=$(CLASH_SETTINGS_PATH="$TMP_SETTINGS" bash "$AETHER_UTILS" clash-setup --uninstall 2>/dev/null)
if echo "$result" | jq -e '.ok == true and .result.hook_installed == false' >/dev/null 2>&1; then
    pass "clash-setup --uninstall returns success"
else
    fail "clash-setup --uninstall failed: $result"
fi
rm -f "$TMP_SETTINGS"

# --- Test 7: worktree-create validates required args ---
echo "7. worktree-create validates required --branch argument"
result=$(bash "$AETHER_UTILS" worktree-create 2>&1)
if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
    pass "worktree-create returns error without --branch"
else
    fail "Expected error JSON, got: $result"
fi

# --- Test 8: worktree-cleanup validates required args ---
echo "8. worktree-cleanup validates required --branch argument"
result=$(bash "$AETHER_UTILS" worktree-cleanup 2>&1)
if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
    pass "worktree-cleanup returns error without --branch"
else
    fail "Expected error JSON, got: $result"
fi

# --- Test 9: .aether/data/ is allowlisted ---
echo "9. .aether/data/ files bypass clash detection"
result=$(bash "$AETHER_UTILS" clash-detect --file ".aether/data/pheromones.json" 2>/dev/null)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass ".aether/data/ files bypass clash detection"
else
    fail "Expected no conflict for .aether/data/ files: $result"
fi

# --- Test 10: merge driver script exists and works ---
echo "10. merge-driver-lockfile.sh is functional"
MERGE_DRIVER="$REPO_ROOT/.aether/utils/merge-driver-lockfile.sh"
if [[ -f "$MERGE_DRIVER" ]]; then
    if bash "$MERGE_DRIVER" /dev/null /dev/null /dev/null >/dev/null 2>&1; then
        pass "merge-driver-lockfile.sh runs and exits 0"
    else
        fail "merge-driver-lockfile.sh failed"
    fi
else
    fail "merge-driver-lockfile.sh not found at $MERGE_DRIVER"
fi

# --- Test 11: .gitattributes has lockfile merge driver ---
echo "11. .gitattributes registers lockfile merge driver"
if grep -q 'package-lock.json merge=lockfile' "$REPO_ROOT/.gitattributes" 2>/dev/null; then
    pass ".gitattributes has package-lock.json merge=lockfile"
else
    fail ".gitattributes missing lockfile merge driver entry"
fi

# --- Test 12: PreToolUse hook script exists ---
echo "12. clash-pre-tool-use.js hook script exists"
HOOK_SCRIPT="$REPO_ROOT/.aether/utils/hooks/clash-pre-tool-use.js"
if [[ -f "$HOOK_SCRIPT" ]]; then
    pass "clash-pre-tool-use.js exists"
else
    fail "clash-pre-tool-use.js not found at $HOOK_SCRIPT"
fi

echo ""
echo "=== RESULTS ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
    echo "STATUS: SOME TESTS FAILED"
    exit 1
else
    echo "STATUS: ALL TESTS PASSED"
    exit 0
fi
