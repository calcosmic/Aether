#!/usr/bin/env bash
# Test: Pheromone snapshot inject and merge-back
# Tests the 4 new subcommands:
#   pheromone-snapshot-inject, pheromone-export-branch,
#   pheromone-merge-back, pheromone-merge-log
#
# Uses isolated temp directories and fake git repos to avoid
# modifying any real colony state.

set -uo pipefail
# Note: set -e disabled because ((PASS++)) returns 1 when PASS was 0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UTILS_SH="$REPO_ROOT/.aether/aether-utils.sh"

# Ensure aether-utils.sh exists
if [[ ! -f "$UTILS_SH" ]]; then
  echo "FATAL: $UTILS_SH not found"
  exit 1
fi

PASS=0
FAIL=0
SKIP=0

# --- Test harness helpers ---

# Create an isolated temp git repo with .aether/data/
setup_test_repo() {
  local tmpdir
  tmpdir=$(mktemp -d)
  cd "$tmpdir" || return 1
  git init -q
  git config user.email "test@aether.dev"
  git config user.name "Test"
  mkdir -p .aether/data .aether/locks
  # Create a dummy commit so HEAD is valid
  echo "init" > .aether/data/.gitkeep
  git add .aether/data/.gitkeep
  git commit -q -m "init"
  echo "$tmpdir"
}

# Create a pheromones.json with specified signals
# Args: output_dir, signal_json_array
create_pheromones() {
  local dir="$1"
  local signals="$2"
  local now
  now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  printf '{\n  "version": "1.0.0",\n  "colony_id": "test",\n  "generated_at": "%s",\n  "signals": %s\n}\n' \
    "$now" "$signals" > "$dir/.aether/data/pheromones.json"
}

# Run a pheromone subcommand in a test repo context
# Args: workdir, subcommand, args...
run_pheromone() {
  local workdir="$1"
  shift
  (cd "$workdir" && AETHER_ROOT="$workdir" DATA_DIR="$workdir/.aether/data" COLONY_DATA_DIR="$workdir/.aether/data" bash "$UTILS_SH" "$@")
}

# Assert a JSON value at a path equals expected
assert_json_eq() {
  local label="$1" json="$2" path="$3" expected="$4"
  local actual
  actual=$(echo "$json" | jq -r "$path" 2>/dev/null)
  if [[ "$actual" == "$expected" ]]; then
    echo "  PASS: $label"
    ((PASS++))
  else
    echo "  FAIL: $label -- expected '$expected', got '$actual'"
    ((FAIL++))
  fi
}

# Assert a JSON value at a path is greater than or equal to a number
assert_json_gte() {
  local label="$1" json="$2" path="$3" expected="$4"
  local actual
  actual=$(echo "$json" | jq "$path" 2>/dev/null)
  if [[ "$actual" =~ ^[0-9.]+$ ]] && (( $(echo "$actual >= $expected" | bc -l 2>/dev/null || echo 0) )); then
    echo "  PASS: $label"
    ((PASS++))
  else
    echo "  FAIL: $label -- expected >= $expected, got '$actual'"
    ((FAIL++))
  fi
}

# Assert command exits 0 (success)
assert_exit_ok() {
  local label="$1"
  shift
  local output
  output=$("$@" 2>&1)
  local rc=$?
  if [[ $rc -eq 0 ]] && echo "$output" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: $label"
    ((PASS++))
  else
    echo "  FAIL: $label -- exit code $rc, output: $(echo "$output" | head -3)"
    ((FAIL++))
  fi
}

# Assert command exits non-zero (error)
assert_exit_fail() {
  local label="$1"
  shift
  local output
  output=$("$@" 2>&1)
  local rc=$?
  if [[ $rc -ne 0 ]]; then
    echo "  PASS: $label"
    ((PASS++))
  else
    echo "  FAIL: $label -- expected non-zero exit, got 0"
    ((FAIL++))
  fi
}

# Run a worktree-create command in a test repo context
# Args: workdir, args...
run_worktree_create() {
  local workdir="$1"
  shift
  (cd "$workdir" && AETHER_ROOT="$workdir" DATA_DIR="$workdir/.aether/data" COLONY_DATA_DIR="$workdir/.aether/data" AETHER_TESTING=1 \
    bash "$workdir/.aether/aether-utils.sh" worktree-create "$@" 2>&1)
}

# ============================================================
echo "=== Pheromone Snapshot Inject & Merge-Back Tests ==="
echo ""

# ============================================================
echo "--- 1. pheromone-snapshot-inject: Injection filter rules ---"
echo ""

# 1a. REDIRECT from user -- should be injected
echo "1a. REDIRECT (user) should be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_1","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid X"},"content_hash":"hash_rx1","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for user REDIRECT" "$result" '.result.injected_count' "1"
else
  echo "  FAIL: command did not return ok -- output: $(echo "$result" | head -5)"
  ((FAIL++))
fi
rm -rf "$td"

# 1b. REDIRECT from worker -- should be injected
echo "1b. REDIRECT (worker:builder) should be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_2","type":"REDIRECT","priority":"high","source":"worker:builder",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid Y"},"content_hash":"hash_rx2","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for worker REDIRECT" "$result" '.result.injected_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1c. REDIRECT from system -- should be injected
echo "1c. REDIRECT (system) should be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_3","type":"REDIRECT","priority":"high","source":"system",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid Z"},"content_hash":"hash_rx3","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for system REDIRECT" "$result" '.result.injected_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1d. FOCUS from user -- should be injected
echo "1d. FOCUS (user) should be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_focus_1","type":"FOCUS","priority":"normal","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"auth"},"content_hash":"hash_f1","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for user FOCUS" "$result" '.result.injected_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1e. FOCUS from worker -- should NOT be injected
echo "1e. FOCUS (worker:builder) should NOT be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_focus_2","type":"FOCUS","priority":"normal","source":"worker:builder",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"database"},"content_hash":"hash_f2","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for worker FOCUS" "$result" '.result.injected_count' "0"
  assert_json_eq "skipped_count for worker FOCUS" "$result" '.result.skipped_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1f. FEEDBACK from user -- should be injected
echo "1f. FEEDBACK (user) should be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_1","type":"FEEDBACK","priority":"low","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"clean code"},"content_hash":"hash_b1","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for user FEEDBACK" "$result" '.result.injected_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1g. FEEDBACK from worker -- should NOT be injected
echo "1g. FEEDBACK (worker:watcher) should NOT be injected"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_2","type":"FEEDBACK","priority":"low","source":"worker:watcher",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"fast"},"content_hash":"hash_b2","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for worker FEEDBACK" "$result" '.result.injected_count' "0"
  assert_json_eq "skipped_count for worker FEEDBACK" "$result" '.result.skipped_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 1h. Mixed signals -- only correct ones injected
echo "1h. Mixed signals -- only injectable ones counted"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_u","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"r1"},"content_hash":"hr1","reinforcement_count":0},
  {"id":"sig_redirect_w","type":"REDIRECT","priority":"high","source":"worker:builder",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"r2"},"content_hash":"hr2","reinforcement_count":0},
  {"id":"sig_focus_u","type":"FOCUS","priority":"normal","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"f1"},"content_hash":"hf1","reinforcement_count":0},
  {"id":"sig_focus_w","type":"FOCUS","priority":"normal","source":"worker:builder",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"f2"},"content_hash":"hf2","reinforcement_count":0},
  {"id":"sig_feedback_u","type":"FEEDBACK","priority":"low","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"b1"},"content_hash":"hb1","reinforcement_count":0},
  {"id":"sig_feedback_w","type":"FEEDBACK","priority":"low","source":"worker:watcher",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"b2"},"content_hash":"hb2","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for mixed signals" "$result" '.result.injected_count' "4"
  assert_json_eq "skipped_count for mixed signals" "$result" '.result.skipped_count' "2"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 2. pheromone-snapshot-inject: Edge cases ---"
echo ""

# 2a. No pheromones on main -- no-op
echo "2a. No pheromones.json on main -- no-op"
td=$(setup_test_repo)
# Do not create pheromones.json
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count when no pheromones" "$result" '.result.injected_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 2b. Expired signals should not be injected
echo "2b. Expired signals should be skipped"
td=$(setup_test_repo)
# Create a signal that expired in the past
create_pheromones "$td" '[
  {"id":"sig_redirect_expired","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-01-01T00:00:00Z","expires_at":"2026-01-02T00:00:00Z","active":true,
   "strength":0.9,"reason":"test","content":{"text":"old avoid"},"content_hash":"hash_old","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for expired signal" "$result" '.result.injected_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 2c. Inactive signals should not be injected
echo "2c. Inactive signals should be skipped"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_inactive","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":false,
   "strength":0.9,"reason":"test","content":{"text":"inactive avoid"},"content_hash":"hash_inact","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "injected_count for inactive signal" "$result" '.result.injected_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 2d. Snapshot metadata written correctly
echo "2d. Snapshot metadata file created"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_snap","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"snap test"},"content_hash":"hash_snap","reinforcement_count":0}
]'
result=$(run_pheromone "$td" pheromone-snapshot-inject --from-branch main --from-commit abc123)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  snap_file="$td/.aether/data/pheromone-snapshot.json"
  if [[ -f "$snap_file" ]]; then
    assert_json_eq "snapshot schema" "$(cat "$snap_file")" '.schema' "pheromone-snapshot-v1"
    assert_json_eq "snapshot from branch" "$(cat "$snap_file")" '.snapshot_from_branch' "main"
    assert_json_eq "snapshot from commit" "$(cat "$snap_file")" '.snapshot_from_commit' "abc123"
  else
    echo "  FAIL: snapshot file not created at $snap_file"
    ((FAIL++))
  fi
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 3. pheromone-export-branch: Export eligibility ---"
echo ""

# 3a. Worker REDIRECT is eligible for merge
echo "3a. Worker REDIRECT is eligible for export"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_wrk","type":"REDIRECT","priority":"high","source":"worker:builder",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid raw SQL"},"content_hash":"hash_new_r","reinforcement_count":1}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for worker REDIRECT" "$result" '.result.eligible_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 3b. User signals are NOT eligible (already on main)
echo "3b. User REDIRECT is NOT eligible for export"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_user","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid X"},"content_hash":"hash_ur","reinforcement_count":0}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for user REDIRECT" "$result" '.result.eligible_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 3c. FEEDBACK with reinforcement_count >= 2 is eligible
echo "3c. Worker FEEDBACK with reinforcement >= 2 is eligible"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_reinforced","type":"FEEDBACK","priority":"low","source":"worker:watcher",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"test coverage thin"},"content_hash":"hash_br","reinforcement_count":3}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for reinforced FEEDBACK" "$result" '.result.eligible_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 3d. FEEDBACK with reinforcement_count < 2 is NOT eligible
echo "3d. Worker FEEDBACK with reinforcement < 2 is NOT eligible"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_weak","type":"FEEDBACK","priority":"low","source":"worker:watcher",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"minor note"},"content_hash":"hash_bw","reinforcement_count":1}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for weak FEEDBACK" "$result" '.result.eligible_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 3e. Worker FOCUS is never eligible
echo "3e. Worker FOCUS is NOT eligible for export"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_focus_wrk","type":"FOCUS","priority":"normal","source":"worker:builder",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"database schema"},"content_hash":"hash_wf","reinforcement_count":5}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for worker FOCUS" "$result" '.result.eligible_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 3f. Export file written with correct schema
echo "3f. Export file has correct schema"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_wrk2","type":"REDIRECT","priority":"high","source":"worker:builder",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"new constraint"},"content_hash":"hash_nr","reinforcement_count":1}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  export_file="$td/.aether/data/pheromone-branch-export.json"
  if [[ -f "$export_file" ]]; then
    assert_json_eq "export schema" "$(cat "$export_file")" '.schema' "pheromone-branch-export-v1"
  else
    echo "  FAIL: export file not created"
    ((FAIL++))
  fi
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 4. pheromone-merge-log: Read merge log entries ---"
echo ""

# 4a. Merge log returns empty when no log exists
echo "4a. Merge log returns empty when no log exists"
td=$(setup_test_repo)
result=$(run_pheromone "$td" pheromone-merge-log)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "merge-log entries count when empty" "$result" '.result.entries_count' "0"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 4b. Merge log returns entries after merge-back
echo "4b. Merge log returns entries after merge-back"
td=$(setup_test_repo)
# Create main pheromones (empty) and a branch export with eligible signals
create_pheromones "$td" '[]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_redirect_new",
      "type": "REDIRECT",
      "source": "worker:builder",
      "content_hash": "sha256:newhash",
      "content_text": "avoid raw SQL",
      "strength": 0.9,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": true,
      "merge_reason": "new worker REDIRECT"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  log_result=$(run_pheromone "$td" pheromone-merge-log)
  if echo "$log_result" | jq -e '.ok == true' >/dev/null 2>&1; then
    assert_json_eq "merge-log entries count after merge-back" "$log_result" '.result.entries_count' "1"
    assert_json_eq "merge-log branch name" "$log_result" '.result.entries[0].merged_from_branch' "feature/test"
  else
    echo "  FAIL: merge-log command did not return ok"
    ((FAIL++))
  fi
else
  echo "  FAIL: merge-back command did not return ok -- output: $(echo "$result" | head -5)"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 5. pheromone-merge-back: Merge eligibility ---"
echo ""

# 5a. New worker REDIRECT gets written to main
echo "5a. New worker REDIRECT merges to main pheromones"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_redirect_merge",
      "type": "REDIRECT",
      "source": "worker:builder",
      "content_hash": "sha256:mergehash",
      "content_text": "avoid raw SQL in migrations",
      "strength": 0.9,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": true,
      "merge_reason": "new worker REDIRECT"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  # Check that the signal was actually written to pheromones.json
  signal_count=$(jq '[.signals[] | select(.content_hash == "sha256:mergehash")] | length' "$td/.aether/data/pheromones.json" 2>/dev/null || echo "0")
  if [[ "$signal_count" -eq 1 ]]; then
    echo "  PASS: worker REDIRECT written to main pheromones"
    ((PASS++))
  else
    echo "  FAIL: expected signal in pheromones.json, found count=$signal_count"
    ((FAIL++))
  fi
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 5b. Ineligible signals (user signals, worker FOCUS, weak FEEDBACK) are skipped
echo "5b. Ineligible signals are skipped during merge-back"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_redirect_user_skip",
      "type": "REDIRECT",
      "source": "user",
      "content_hash": "sha256:userskip",
      "content_text": "user constraint",
      "strength": 0.9,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 0,
      "eligible_for_merge": false,
      "merge_reason": "user signal already on main"
    },
    {
      "id": "sig_focus_worker_skip",
      "type": "FOCUS",
      "source": "worker:builder",
      "content_hash": "sha256:focuswskip",
      "content_text": "worker focus",
      "strength": 0.8,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 0,
      "eligible_for_merge": false,
      "merge_reason": "worker FOCUS excluded"
    },
    {
      "id": "sig_feedback_weak_skip",
      "type": "FEEDBACK",
      "source": "worker:watcher",
      "content_hash": "sha256:weakskip",
      "content_text": "one-off note",
      "strength": 0.7,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": false,
      "merge_reason": "FEEDBACK reinforcement < 2"
    }
  ],
  "total_signals": 3,
  "eligible_count": 0,
  "ineligible_count": 3
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "new_signals_written for ineligible batch" "$result" '.result.new_signals_written' "0"
  assert_json_eq "skipped_count for ineligible batch" "$result" '.result.skipped_count' "3"
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 5c. No export file -- merge-back is a no-op
echo "5c. No export file -- merge-back is no-op"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "new_signals_written when no export" "$result" '.result.new_signals_written' "0"
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 6. pheromone-merge-back: Conflict resolution ---"
echo ""

# 6a. REDIRECT conflict -- reinforce (take max strength)
echo "6a. REDIRECT conflict resolves to reinforce"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_existing","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"avoid X"},"content_hash":"sha256:conflict1","reinforcement_count":1}
]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_redirect_branch",
      "type": "REDIRECT",
      "source": "worker:builder",
      "content_hash": "sha256:conflict1",
      "content_text": "avoid X",
      "strength": 0.95,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": true,
      "merge_reason": "new worker REDIRECT (conflict with main)"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "conflict resolution for REDIRECT" "$result" '.result.conflicts_resolved[0].resolution' "reinforced"
  # Verify strength was updated to max(0.7, 0.95) = 0.95
  new_strength=$(jq '[.signals[] | select(.content_hash == "sha256:conflict1")][0].strength' "$td/.aether/data/pheromones.json" 2>/dev/null || echo "0")
  if [[ "$new_strength" == "0.95" ]]; then
    echo "  PASS: REDIRECT strength updated to 0.95 (max)"
    ((PASS++))
  else
    echo "  FAIL: expected strength 0.95, got $new_strength"
    ((FAIL++))
  fi
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# 6b. User FOCUS conflict -- skip (main is authoritative)
echo "6b. User FOCUS conflict resolves to skip"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_focus_existing","type":"FOCUS","priority":"normal","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"security"},"content_hash":"sha256:focus_conflict","reinforcement_count":0}
]'
# This should not happen in practice (user FOCUS is not eligible for export),
# but test the conflict resolution engine directly
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_focus_branch",
      "type": "FOCUS",
      "source": "user",
      "content_hash": "sha256:focus_conflict",
      "content_text": "security",
      "strength": 0.9,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 2,
      "eligible_for_merge": true,
      "merge_reason": "test conflict resolution"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  # The merge-back should skip this signal (user FOCUS is authoritative on main)
  conflicts=$(echo "$result" | jq '.result.conflicts_resolved | length' 2>/dev/null || echo "0")
  if [[ "$conflicts" -ge 1 ]]; then
    resolution=$(echo "$result" | jq -r '.result.conflicts_resolved[0].resolution' 2>/dev/null || echo "")
    if [[ "$resolution" == "skip" ]]; then
      echo "  PASS: user FOCUS conflict resolved to skip"
      ((PASS++))
    else
      echo "  FAIL: expected 'skip', got '$resolution'"
      ((FAIL++))
    fi
  else
    echo "  FAIL: expected at least 1 conflict, got $conflicts"
    ((FAIL++))
  fi
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 7. pheromone-merge-back: FEEDBACK conflict resolution ---"
echo ""

# 7a. FEEDBACK conflict with reinforcement >= 2 -- reinforce
echo "7a. FEEDBACK conflict with reinforcement >= 2 resolves to reinforce"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_existing","type":"FEEDBACK","priority":"low","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"test coverage thin"},"content_hash":"sha256:fb_conflict","reinforcement_count":0}
]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_feedback_branch",
      "type": "FEEDBACK",
      "source": "worker:watcher",
      "content_hash": "sha256:fb_conflict",
      "content_text": "test coverage thin",
      "strength": 0.8,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 3,
      "eligible_for_merge": true,
      "merge_reason": "FEEDBACK with reinforcement >= 2"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "FEEDBACK conflict resolution" "$result" '.result.conflicts_resolved[0].resolution' "reinforced"
else
  echo "  FAIL: merge-back did not return ok -- output: $(echo "$result" | head -5)"
  ((FAIL++))
fi
rm -rf "$td"

# 7b. FEEDBACK conflict with reinforcement < 2 -- skip
echo "7b. FEEDBACK conflict with reinforcement < 2 resolves to skip"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_feedback_weak_main","type":"FEEDBACK","priority":"low","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"minor note"},"content_hash":"sha256:fb_weak","reinforcement_count":0}
]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/test",
  "branch_commit": "def456",
  "signals": [
    {
      "id": "sig_feedback_weak_branch",
      "type": "FEEDBACK",
      "source": "worker:watcher",
      "content_hash": "sha256:fb_weak",
      "content_text": "minor note",
      "strength": 0.75,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": true,
      "merge_reason": "FEEDBACK with reinforcement < 2"
    }
  ],
  "total_signals": 1,
  "eligible_count": 1,
  "ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  resolution=$(echo "$result" | jq -r '.result.conflicts_resolved[0].resolution' 2>/dev/null || echo "none")
  assert_json_eq "FEEDBACK weak conflict resolution" "$result" '.result.conflicts_resolved[0].resolution' "skip"
else
  echo "  FAIL: merge-back did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 8. pheromone-export-branch: System REDIRECT eligible ---"
echo ""

# 8a. System REDIRECT is eligible for export
echo "8a. System REDIRECT is eligible for export"
td=$(setup_test_repo)
create_pheromones "$td" '[
  {"id":"sig_redirect_sys","type":"REDIRECT","priority":"high","source":"system",
   "created_at":"2026-03-30T13:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"midden threshold pattern"},"content_hash":"hash_sys_r","reinforcement_count":1}
]'
git -C "$td" checkout -q -b feature/test-branch 2>/dev/null || true
result=$(run_pheromone "$td" pheromone-export-branch)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "eligible_count for system REDIRECT" "$result" '.result.eligible_count' "1"
else
  echo "  FAIL: command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 9. pheromone-merge-log: --last flag ---"
echo ""

# 9a. --last N returns only last N entries
echo "9a. --last 1 returns only the most recent entry"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
# Run two merge-backs with different branches
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/first",
  "branch_commit": "aaa111",
  "signals": [
    {
      "id": "sig_r_first","type": "REDIRECT","source": "worker:builder",
      "content_hash": "sha256:first","content_text": "first redirect",
      "strength": 0.9,"created_at": "2026-03-30T13:00:00Z","expires_at": "phase_end",
      "reinforcement_count": 1,"eligible_for_merge": true,"merge_reason": "test"
    }
  ],
  "total_signals": 1,"eligible_count": 1,"ineligible_count": 0
}
EXPORT_EOF
run_pheromone "$td" pheromone-merge-back >/dev/null 2>&1 || true

cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T15:00:00Z",
  "branch_name": "feature/second",
  "branch_commit": "bbb222",
  "signals": [
    {
      "id": "sig_r_second","type": "REDIRECT","source": "worker:builder",
      "content_hash": "sha256:second","content_text": "second redirect",
      "strength": 0.9,"created_at": "2026-03-30T14:00:00Z","expires_at": "phase_end",
      "reinforcement_count": 1,"eligible_for_merge": true,"merge_reason": "test"
    }
  ],
  "total_signals": 1,"eligible_count": 1,"ineligible_count": 0
}
EXPORT_EOF
run_pheromone "$td" pheromone-merge-back >/dev/null 2>&1 || true

result=$(run_pheromone "$td" pheromone-merge-log --last 1)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "merge-log --last 1 returns 1 entry" "$result" '.result.entries_count' "1"
  assert_json_eq "merge-log --last 1 returns most recent branch" "$result" '.result.entries[0].merged_from_branch' "feature/second"
else
  echo "  FAIL: merge-log command did not return ok"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 10. pheromone-merge-back: Invalid export schema ---"
echo ""

# 10a. Invalid schema is rejected
echo "10a. Invalid export schema is rejected"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
printf '{"schema":"wrong-schema","exported_at":"x","branch_name":"x","branch_commit":"x","signals":[],"total_signals":0,"eligible_count":0,"ineligible_count":0}' > "$td/.aether/data/pheromone-branch-export.json"
assert_exit_fail "merge-back rejects invalid schema" run_pheromone "$td" pheromone-merge-back
rm -rf "$td"

# ============================================================
echo ""
echo "--- 11. pheromone-merge-back: Multiple eligible signals ---"
echo ""

# 11a. Multiple eligible signals merge correctly
echo "11a. Multiple eligible signals merge in one pass"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
cat > "$td/.aether/data/pheromone-branch-export.json" <<'EXPORT_EOF'
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/multi",
  "branch_commit": "ccc333",
  "signals": [
    {
      "id": "sig_r_multi1","type": "REDIRECT","source": "worker:builder",
      "content_hash": "sha256:multi1","content_text": "avoid raw SQL",
      "strength": 0.9,"created_at": "2026-03-30T13:00:00Z","expires_at": "phase_end",
      "reinforcement_count": 1,"eligible_for_merge": true,"merge_reason": "test"
    },
    {
      "id": "sig_r_multi2","type": "REDIRECT","source": "system",
      "content_hash": "sha256:multi2","content_text": "avoid eval",
      "strength": 0.85,"created_at": "2026-03-30T13:30:00Z","expires_at": "phase_end",
      "reinforcement_count": 0,"eligible_for_merge": true,"merge_reason": "test"
    },
    {
      "id": "sig_fb_multi3","type": "FEEDBACK","source": "worker:watcher",
      "content_hash": "sha256:multi3","content_text": "coverage needs work",
      "strength": 0.7,"created_at": "2026-03-30T13:45:00Z","expires_at": "phase_end",
      "reinforcement_count": 4,"eligible_for_merge": true,"merge_reason": "FEEDBACK with reinforcement >= 2"
    }
  ],
  "total_signals": 3,"eligible_count": 3,"ineligible_count": 0
}
EXPORT_EOF
result=$(run_pheromone "$td" pheromone-merge-back)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  assert_json_eq "multiple new_signals_written" "$result" '.result.new_signals_written' "3"
  # Verify all 3 signals are in pheromones.json
  sig_count=$(jq '[.signals[] | select(.content_hash == "sha256:multi1" or .content_hash == "sha256:multi2" or .content_hash == "sha256:multi3")] | length' "$td/.aether/data/pheromones.json" 2>/dev/null || echo "0")
  if [[ "$sig_count" -eq 3 ]]; then
    echo "  PASS: all 3 signals present in pheromones.json"
    ((PASS++))
  else
    echo "  FAIL: expected 3 signals in pheromones.json, got $sig_count"
    ((FAIL++))
  fi
else
  echo "  FAIL: merge-back did not return ok -- output: $(echo "$result" | head -5)"
  ((FAIL++))
fi
rm -rf "$td"

# ============================================================
echo ""
echo "--- 12. pheromone-snapshot-inject: Requires --from-commit ---"
echo ""

# 12a. Missing --from-commit should fail
echo "12a. Missing --from-commit fails with validation error"
td=$(setup_test_repo)
create_pheromones "$td" '[]'
assert_exit_fail "snapshot-inject requires --from-commit" run_pheromone "$td" pheromone-snapshot-inject --from-branch main
rm -rf "$td"

# ============================================================
echo ""
echo "--- 13. pheromone-export-branch: No pheromones file ---"
echo ""

# 13a. Export fails when no pheromones.json exists
echo "13a. Export fails when no pheromones.json"
td=$(setup_test_repo)
# Do not create pheromones.json
assert_exit_fail "export-branch fails without pheromones.json" run_pheromone "$td" pheromone-export-branch
rm -rf "$td"

# ============================================================
echo ""
echo "--- 14. worktree-create integration: pheromone injection ---"
echo ""

# Helper: Set up a full worktree test environment with utils copied
setup_worktree_with_pheromones() {
  local tmpdir
  tmpdir=$(mktemp -d)
  cd "$tmpdir" || return 1
  git init -q
  git config user.email "test@aether.dev"
  git config user.name "Test"

  # Create full .aether structure
  mkdir -p .aether/data .aether/locks .aether/utils .aether/exchange

  # Copy aether-utils.sh (dispatcher)
  cp "$REPO_ROOT/.aether/aether-utils.sh" .aether/aether-utils.sh

  # Copy all utils
  cp -r "$REPO_ROOT/.aether/utils/." .aether/utils/ 2>/dev/null || true

  # Copy exchange if present
  if [[ -d "$REPO_ROOT/.aether/exchange" ]]; then
    cp -r "$REPO_ROOT/.aether/exchange/." .aether/exchange/ 2>/dev/null || true
  fi

  # Minimal COLONY_STATE.json
  cat > .aether/data/COLONY_STATE.json << 'CS EOF'
{
  "version": "3.0",
  "goal": "Test worktree pheromone injection",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "plan": { "phases": [{ "id": 1, "name": "Test", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [], "signals": [], "graveyards": [], "workers": [], "spawn_tree": []
}
CS EOF

  # Make an initial commit so HEAD is valid
  echo "# test" > README.md
  git add README.md
  git commit -q -m "initial commit"

  echo "$tmpdir"
}

# Helper: Run worktree-create in a test repo
run_worktree_create() {
  local workdir="$1"
  shift
  ( cd "$workdir" && AETHER_ROOT="$workdir" DATA_DIR="$workdir/.aether/data" \
    COLONY_DATA_DIR="$workdir/.aether/data" \
    bash "$workdir/.aether/aether-utils.sh" worktree-create "$@" 2>&1 )
}

# 14a. Worktree creation triggers pheromone-snapshot-inject (verified by _snapshot metadata)
echo "14a. worktree_create_injects_pheromones"
td=$(setup_worktree_with_pheromones)
create_pheromones "$td" '[
  {"id":"sig_redirect_user","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid X"},"content_hash":"hash_wt_rx","reinforcement_count":0},
  {"id":"sig_focus_user","type":"FOCUS","priority":"normal","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"security"},"content_hash":"hash_wt_f1","reinforcement_count":0}
]'
result=$(run_worktree_create "$td" --branch "test-inject-1")
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  wt_dir=$(echo "$result" | jq -r '.result.worktree_dir')
  if [[ -f "$wt_dir/.aether/data/pheromones.json" ]]; then
    # Key check: _snapshot metadata must be present (proves pheromone-snapshot-inject ran)
    snapshot_branch=$(jq -r '._snapshot.snapshot_from_branch // empty' "$wt_dir/.aether/data/pheromones.json" 2>/dev/null)
    if [[ -n "$snapshot_branch" ]]; then
      echo "  PASS: worktree_create_injects_pheromones -- snapshot metadata present (from_branch=$snapshot_branch)"
      ((PASS++))
    else
      echo "  FAIL: worktree_create_injects_pheromones -- no _snapshot metadata in worktree pheromones.json"
      ((FAIL++))
    fi
    # Also verify active signals are present
    wt_signal_count=$(jq '[.signals[] | select(.active == true)] | length' "$wt_dir/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$wt_signal_count" -ge 2 ]]; then
      echo "  PASS: worktree has $wt_signal_count active signals"
      ((PASS++))
    else
      echo "  FAIL: worktree has $wt_signal_count active signals, expected >= 2"
      ((FAIL++))
    fi
  else
    echo "  FAIL: worktree_create_injects_pheromones -- pheromones.json not found in worktree"
    ((FAIL++))
  fi
else
  echo "  FAIL: worktree_create_injects_pheromones -- command failed: $(echo "$result" | head -3)"
  ((FAIL++))
fi
# Cleanup worktree and temp dir
cd / 2>/dev/null || true
rm -rf "$td" 2>/dev/null || true

# 14b. Worktree inject includes REDIRECT and user FOCUS with injection metadata
echo "14b. worktree_create_inject_includes_redirect_and_focus"
td=$(setup_worktree_with_pheromones)
create_pheromones "$td" '[
  {"id":"sig_redirect_u","type":"REDIRECT","priority":"high","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.9,"reason":"test","content":{"text":"avoid X"},"content_hash":"hash_wt_r1","reinforcement_count":0},
  {"id":"sig_focus_u","type":"FOCUS","priority":"normal","source":"user",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.8,"reason":"test","content":{"text":"auth"},"content_hash":"hash_wt_f1","reinforcement_count":0},
  {"id":"sig_feedback_w","type":"FEEDBACK","priority":"low","source":"worker:builder",
   "created_at":"2026-03-30T12:00:00Z","expires_at":"phase_end","active":true,
   "strength":0.7,"reason":"test","content":{"text":"nice pattern"},"content_hash":"hash_wt_b1","reinforcement_count":0}
]'
result=$(run_worktree_create "$td" --branch "test-inject-2")
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  wt_dir=$(echo "$result" | jq -r '.result.worktree_dir')
  if [[ -f "$wt_dir/.aether/data/pheromones.json" ]]; then
    # Check REDIRECT is present
    has_redirect=$(jq '[.signals[] | select(.type == "REDIRECT" and .active == true)] | length' "$wt_dir/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$has_redirect" -ge 1 ]]; then
      echo "  PASS: REDIRECT present in worktree pheromones"
      ((PASS++))
    else
      echo "  FAIL: REDIRECT missing from worktree pheromones"
      ((FAIL++))
    fi
    # Check user FOCUS is present
    has_user_focus=$(jq '[.signals[] | select(.type == "FOCUS" and .source == "user" and .active == true)] | length' "$wt_dir/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$has_user_focus" -ge 1 ]]; then
      echo "  PASS: user FOCUS present in worktree pheromones"
      ((PASS++))
    else
      echo "  FAIL: user FOCUS missing from worktree pheromones"
      ((FAIL++))
    fi
    # Verify injection ran: at least one signal should have "Injected from" in reason
    injected_reason_count=$(jq '[.signals[] | select(.reason | test("Injected from"))] | length' "$wt_dir/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$injected_reason_count" -ge 1 ]]; then
      echo "  PASS: injection reason metadata present ($injected_reason_count signals)"
      ((PASS++))
    else
      echo "  FAIL: no signals have injection reason metadata"
      ((FAIL++))
    fi
  else
    echo "  FAIL: pheromones.json not found in worktree"
    ((FAIL++))
  fi
else
  echo "  FAIL: worktree-create command failed: $(echo "$result" | head -3)"
  ((FAIL++))
fi
cd / 2>/dev/null || true
rm -rf "$td" 2>/dev/null || true

# 14c. Worktree creation succeeds without pheromones.json
echo "14c. worktree_create_succeeds_without_pheromones"
td=$(setup_worktree_with_pheromones)
# Do NOT create pheromones.json
result=$(run_worktree_create "$td" --branch "test-no-phero")
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: worktree_create_succeeds_without_pheromones"
  ((PASS++))
else
  echo "  FAIL: worktree_create_succeeds_without_pheromones -- command failed: $(echo "$result" | head -3)"
  ((FAIL++))
fi
cd / 2>/dev/null || true
rm -rf "$td" 2>/dev/null || true

# ============================================================
echo ""
echo "=== RESULTS ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo "Skipped: $SKIP"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
  echo "STATUS: SOME TESTS FAILED"
  exit 1
else
  echo "STATUS: ALL TESTS PASSED"
  exit 0
fi
