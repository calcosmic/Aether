#!/usr/bin/env bash
# scripts/audit-extraction.sh
# Verify no agent behaviour exists only in Go — colony assets are the source of truth.
#
# Usage: bash scripts/audit-extraction.sh
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

echo "=== Extraction Audit ==="
echo ""

# --- Check 1: Agent YAML files exist ---
echo "--- Check 1: Agent YAML files exist ---"
AGENT_COUNT=$(find colony/agents -maxdepth 1 -name "*.yaml" | wc -l | tr -d ' ')
if [[ "$AGENT_COUNT" -ge 8 ]]; then
    pass "Found ${AGENT_COUNT} agent YAML files (>= 8)"
else
    fail "Insufficient agent YAML files" "Found ${AGENT_COUNT}, expected >= 8"
fi

# --- Check 2: Prompt Markdown files exist ---
echo ""
echo "--- Check 2: Prompt Markdown files exist ---"
PROMPT_COUNT=$(find colony/prompts -maxdepth 1 -name "*.md" | wc -l | tr -d ' ')
if [[ "$PROMPT_COUNT" -ge 8 ]]; then
    pass "Found ${PROMPT_COUNT} prompt Markdown files (>= 8)"
else
    fail "Insufficient prompt Markdown files" "Found ${PROMPT_COUNT}, expected >= 8"
fi

# --- Check 3: Phase YAML files exist ---
echo ""
echo "--- Check 3: Phase YAML files exist ---"
PHASE_COUNT=$(find colony/phases -maxdepth 1 -name "*.yaml" | wc -l | tr -d ' ')
if [[ "$PHASE_COUNT" -ge 6 ]]; then
    pass "Found ${PHASE_COUNT} phase YAML files (>= 6)"
else
    fail "Insufficient phase YAML files" "Found ${PHASE_COUNT}, expected >= 6"
fi

# --- Check 4: No agent personality strings only in Go ---
echo ""
echo "--- Check 4: No agent personality strings only in Go ---"
PERSONALITY_MATCHES=$(grep -rn '"You are a.*builder\|"You are a.*watcher\|"You are a.*scout\|"You are a.*queen\|"You are a.*oracle\|"You are a.*gatekeeper\|"You are a.*auditor\|"You are a.*probe' cmd/ --include="*.go" | grep -v '_test.go' || true)
if [[ -z "$PERSONALITY_MATCHES" ]]; then
    pass "No agent personality strings found only in Go"
else
    MATCH_COUNT=$(echo "$PERSONALITY_MATCHES" | grep -c '^' || true)
    fail "Found ${MATCH_COUNT} agent personality string(s) in Go" "$(echo "$PERSONALITY_MATCHES" | head -5)"
fi

# --- Check 5: Extraction audit doc is current ---
echo ""
echo "--- Check 5: Extraction audit doc is current ---"
AUDIT_DOC=".planning/phases/152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md"
if [[ -f "$AUDIT_DOC" ]]; then
    # Count lines that look like classified symbols (contain | and a classification)
    SYMBOL_COUNT=$(grep -cE '\|\s*(KEEP_IN_GO|MOVE_TO_TS|MOVE_TO_YAML|SPLIT|MIXED|REMOVE)\s*\|' "$AUDIT_DOC" || true)
    if [[ "$SYMBOL_COUNT" -ge 90 ]]; then
        pass "Audit doc contains ${SYMBOL_COUNT} classified symbols (>= 90)"
    else
        fail "Audit doc has insufficient classified symbols" "Found ${SYMBOL_COUNT}, expected >= 90"
    fi
else
    fail "Extraction audit doc missing" "$AUDIT_DOC not found"
fi

echo ""
TOTAL_CHECKS=$((PASS_COUNT + FAIL_COUNT))
if [[ $FAIL_COUNT -eq 0 ]]; then
    echo "=== Extraction audit passed (${PASS_COUNT}/${TOTAL_CHECKS}) ==="
    exit 0
else
    echo "=== Extraction audit FAILED (${PASS_COUNT}/${TOTAL_CHECKS} passed, ${FAIL_COUNT} failed) ==="
    exit 1
fi
