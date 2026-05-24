#!/usr/bin/env bash
# scripts/audit-hardcoded.sh
# Audit Go source for hardcoded prompt/agent/phase text.
#
# Usage: bash scripts/audit-hardcoded.sh
# Exit: 0 if all checks pass, 1 if any check fails
set -euo pipefail

PASS_COUNT=0
FAIL_COUNT=0

pass() {
    echo "PASS: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo "FAIL: $1"
    echo "  $2"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

echo "=== Hardcoded Behaviour Audit ==="
echo ""

# --- Check 1: No 'You are' agent directive strings in non-test Go files ---
echo "--- Check 1: No 'You are' agent directive strings in non-test Go files ---"
YOU_ARE_MATCHES=$(grep -rn '"You are' cmd/ --include="*.go" | grep -v '_test.go' | grep -v 'fallback\|error\|fmt.Errorf' || true)
if [[ -z "$YOU_ARE_MATCHES" ]]; then
    pass "No 'You are' directive strings found in non-test Go files"
else
    MATCH_COUNT=$(echo "$YOU_ARE_MATCHES" | grep -c '^' || true)
    fail "Found ${MATCH_COUNT} 'You are' directive string(s) in non-test Go files" "$(echo "$YOU_ARE_MATCHES" | head -5)"
fi

# --- Check 2: No hardcoded ceremony/ritual strings in non-test Go files ---
echo ""
echo "--- Check 2: No hardcoded ceremony/ritual strings in non-test Go files ---"
CEREMONY_FILES=$(grep -rn 'ceremony\|ritual\|playbook' cmd/ --include="*.go" -l | grep -v '_test.go' || true)
if [[ -z "$CEREMONY_FILES" ]]; then
    pass "No ceremony/ritual/playbook strings found in non-test Go files"
else
    FILE_COUNT=$(echo "$CEREMONY_FILES" | grep -c '^' || true)
    fail "Found ${FILE_COUNT} file(s) with ceremony/ritual/playbook strings" "$(echo "$CEREMONY_FILES" | head -5)"
fi

# --- Check 3: No phase directive strings (e.g. 'Your task is to') in non-test Go files ---
echo ""
echo "--- Check 3: No phase directive strings in non-test Go files ---"
TASK_MATCHES=$(grep -rn '"Your task is to' cmd/ --include="*.go" | grep -v '_test.go' | grep -v 'fallback\|error\|fmt.Errorf' || true)
if [[ -z "$TASK_MATCHES" ]]; then
    pass "No 'Your task is to' directive strings found in non-test Go files"
else
    MATCH_COUNT=$(echo "$TASK_MATCHES" | grep -c '^' || true)
    fail "Found ${MATCH_COUNT} 'Your task is to' directive string(s) in non-test Go files" "$(echo "$TASK_MATCHES" | head -5)"
fi

# --- Check 4: Fallback and error text is allowed (informational) ---
echo ""
echo "--- Check 4: Fallback and error text policy ---"
echo "INFO: Fallback strings, error messages, and fmt.Errorf text are permitted"
echo "      per the boundary contract. Only agent directive / behaviour strings"
echo "      are forbidden in non-test Go files."
pass "Fallback and error text policy acknowledged"

echo ""
TOTAL_CHECKS=$((PASS_COUNT + FAIL_COUNT))
if [[ $FAIL_COUNT -eq 0 ]]; then
    echo "=== Hardcoded behaviour audit passed (${PASS_COUNT}/${TOTAL_CHECKS}) ==="
    exit 0
else
    echo "=== Hardcoded behaviour audit FAILED (${PASS_COUNT}/${TOTAL_CHECKS} passed, ${FAIL_COUNT} failed) ==="
    exit 1
fi
