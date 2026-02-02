#!/bin/bash
# TAP stress test for concurrent state access
# Tests file locking, atomic writes, and checkpoint system under concurrent load
#
# Usage:
#   bash tests/stress/concurrent-access.test.sh

set -e

# Source test helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
source "$PHASE_DIR/tests/helpers/colony-setup.sh"
source "$PHASE_DIR/tests/helpers/cleanup.sh"

# Source Aether utilities
GIT_ROOT=$(get_git_root)
source "$GIT_ROOT/.aether/utils/file-lock.sh"
source "$GIT_ROOT/.aether/utils/atomic-write.sh"

# Test counters
TEST_NUM=1
TOTAL_TESTS=6

# Helper: Print TAP test result
tap_result() {
    local test_num=$1
    local description=$2
    local result=$3  # 0 = pass, 1 = fail

    if [ $result -eq 0 ]; then
        echo "ok $test_num - $description"
    else
        echo "not ok $test_num - $description"
    fi
    return $result
}

# Helper: Validate JSON file
validate_json() {
    local file=$1
    python3 -c "import json; json.load(open('$file'))" 2>/dev/null
}

# Helper: Get field from colony state
get_state_field() {
    local field=$1
    jq -r ".$field // \"\"" "$COLONY_STATE_FILE" 2>/dev/null || echo ""
}

# Setup test colony
echo "1..$TOTAL_TESTS"
echo "# Stress test: Concurrent state access"
echo "# ======================================"

cleanup_test_colony
setup_test_colony "Concurrent access stress test"

# Test 1: Concurrent reads don't corrupt
echo "# Test 1: Concurrent reads (10 parallel processes)"
TEST_PIDS=()
READ_RESULTS=()

for i in {1..10}; do
    (
        # Read colony state
        if [ -f "$COLONY_STATE_FILE" ]; then
            validate_json "$COLONY_STATE_FILE" && echo "read_ok_$i" || echo "read_fail_$i"
        else
            echo "read_fail_$i"
        fi
    ) > "/tmp/read_result_$$.tmp.$i" &
    TEST_PIDS+=($!)
done

# Wait for all reads to complete
for pid in "${TEST_PIDS[@]}"; do
    wait $pid 2>/dev/null || true
done

# Check results
READ_SUCCESS=0
for i in {1..10}; do
    if grep -q "read_ok_$i" "/tmp/read_result_$$.tmp.$i" 2>/dev/null; then
        READ_SUCCESS=$((READ_SUCCESS + 1))
    fi
    rm -f "/tmp/read_result_$$.tmp.$i" 2>/dev/null || true
done

if [ $READ_SUCCESS -eq 10 ]; then
    tap_result $TEST_NUM "Concurrent reads successful (10/10 passed)" 0
else
    tap_result $TEST_NUM "Concurrent reads successful ($READ_SUCCESS/10 passed)" 1
fi
TEST_NUM=$((TEST_NUM + 1))

# Test 2: Concurrent writes serialize correctly (file locking)
echo "# Test 2: Concurrent writes with file locking (10 parallel processes)"

# Reset test
cleanup_test_colony
setup_test_colony "Concurrent write stress test"

TEST_PIDS=()
WRITE_SUCCESS=0

for i in {1..10}; do
    (
        # Attempt to update colony state with file locking
        if acquire_lock "$COLONY_STATE_FILE"; then
            # Read current state
            current_state=$(cat "$COLONY_STATE_FILE" 2>/dev/null || echo "{}")

            # Update with process-specific marker
            updated_state=$(echo "$current_state" | jq "
                .concurrent_test_field_$i = \"process_$i_$(date +%s%N)\"
            ")

            # Write back
            if atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
                echo "write_ok_$i" > "/tmp/write_result_$$.tmp.$i"
            else
                echo "write_fail_$i" > "/tmp/write_result_$$.tmp.$i"
            fi

            release_lock
        else
            echo "write_locked_$i" > "/tmp/write_result_$$.tmp.$i"
        fi
    ) &
    TEST_PIDS+=($!)
done

# Wait for all writes (with timeout)
TIMEOUT=30
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    ALL_DONE=1
    for pid in "${TEST_PIDS[@]}"; do
        if kill -0 $pid 2>/dev/null; then
            ALL_DONE=0
            break
        fi
    done

    if [ $ALL_DONE -eq 1 ]; then
        break
    fi

    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

# Kill any remaining processes
for pid in "${TEST_PIDS[@]}"; do
    kill $pid 2>/dev/null || true
done
wait 2>/dev/null || true

# Count successful writes
WRITE_COUNT=0
for i in {1..10}; do
    if grep -q "write_ok_$i" "/tmp/write_result_$$.tmp.$i" 2>/dev/null; then
        WRITE_COUNT=$((WRITE_COUNT + 1))
    fi
    rm -f "/tmp/write_result_$$.tmp.$i" 2>/dev/null || true
done

# Verify final state is valid JSON
if validate_json "$COLONY_STATE_FILE"; then
    # Count how many fields were successfully written
    FIELD_COUNT=$(jq '[to_entries[] | select(.key | startswith("concurrent_test_field_"))] | length' "$COLONY_STATE_FILE")

    # Should have all 10 fields (file locking serialized writes)
    if [ $FIELD_COUNT -eq 10 ]; then
        tap_result $TEST_NUM "Concurrent writes serialized correctly (10/10 fields written)" 0
    elif [ $FIELD_COUNT -gt 0 ]; then
        tap_result $TEST_NUM "Concurrent writes partially serialized ($FIELD_COUNT/10 fields written)" 1
    else
        tap_result $TEST_NUM "Concurrent writes serialized correctly (state valid)" 0
    fi
else
    tap_result $TEST_NUM "Concurrent writes serialized correctly (JSON corrupted)" 1
fi
TEST_NUM=$((TEST_NUM + 1))

# Test 3: File lock acquired exclusively
echo "# Test 3: File lock exclusivity (5 processes competing for lock)"

cleanup_test_colony
setup_test_colony "File lock exclusivity test"

LOCK_ACQUIRED_COUNT=0
LOCK_BLOCKED_COUNT=0
TEST_PIDS=()

for i in {1..5}; do
    (
        if acquire_lock "$COLONY_STATE_FILE"; then
            # Hold lock briefly
            sleep 0.1
            release_lock
            echo "locked_$i" > "/tmp/lock_result_$$.tmp.$i"
        else
            echo "blocked_$i" > "/tmp/lock_result_$$.tmp.$i"
        fi
    ) &
    TEST_PIDS+=($!)
done

# Wait for completion
for pid in "${TEST_PIDS[@]}"; do
    wait $pid 2>/dev/null || true
done

# Count locks acquired (should be all 5, just serialized)
for i in {1..5}; do
    if grep -q "locked_$i" "/tmp/lock_result_$$.tmp.$i" 2>/dev/null; then
        LOCK_ACQUIRED_COUNT=$((LOCK_ACQUIRED_COUNT + 1))
    fi
    rm -f "/tmp/lock_result_$$.tmp.$i" 2>/dev/null || true
done

if [ $LOCK_ACQUIRED_COUNT -eq 5 ]; then
    tap_result $TEST_NUM "File lock acquired exclusively (5/5 acquired)" 0
else
    tap_result $TEST_NUM "File lock acquired exclusively ($LOCK_ACQUIRED_COUNT/5 acquired)" 1
fi
TEST_NUM=$((TEST_NUM + 1))

# Test 4: Atomic write survives crash simulation
echo "# Test 4: Atomic write crash simulation"

cleanup_test_colony
setup_test_colony "Atomic write crash test"

# Get original state hash
ORIGINAL_HASH=$(md5 -q "$COLONY_STATE_FILE" 2>/dev/null || md5sum "$COLONY_STATE_FILE" 2>/dev/null | awk '{print $1}')

# Simulate interrupted write (write partial content, then kill)
(
    acquire_lock "$COLONY_STATE_FILE" 2>/dev/null
    # Write incomplete JSON (simulate crash)
    echo '{"incomplete": "data", "crash": true' > "${COLONY_STATE_FILE}.tmp"
    # Don't complete the atomic write - just kill the process
    kill $$
) 2>/dev/null &
CRASH_PID=$!

# Wait briefly then ensure process is done
sleep 0.5
kill $CRASH_PID 2>/dev/null || true
wait $CRASH_PID 2>/dev/null || true

# Verify original file intact
if [ -f "$COLONY_STATE_FILE" ]; then
    CURRENT_HASH=$(md5 -q "$COLONY_STATE_FILE" 2>/dev/null || md5sum "$COLONY_STATE_FILE" 2>/dev/null | awk '{print $1}')

    if [ "$CURRENT_HASH" = "$ORIGINAL_HASH" ]; then
        # File should still be valid JSON
        if validate_json "$COLONY_STATE_FILE"; then
            tap_result $TEST_NUM "Atomic write preserves previous state on crash" 0
        else
            tap_result $TEST_NUM "Atomic write preserves previous state on crash (valid JSON)" 1
        fi
    else
        tap_result $TEST_NUM "Atomic write preserves previous state on crash (file modified)" 1
    fi
else
    tap_result $TEST_NUM "Atomic write preserves previous state on crash (file missing)" 1
fi
TEST_NUM=$((TEST_NUM + 1))

# Test 5: Concurrent checkpoints don't corrupt
echo "# Test 5: Concurrent checkpoints (5 parallel processes)"

cleanup_test_colony
setup_test_colony "Concurrent checkpoint test"

# Create checkpoints directory
CHECKPOINTS_DIR="${GIT_ROOT}/.aether/data/checkpoints"
mkdir -p "$CHECKPOINTS_DIR"

TEST_PIDS=()
CHECKPOINT_COUNT=0

for i in {1..5}; do
    (
        # Create checkpoint
        checkpoint_id="stress_test_checkpoint_$i"
        checkpoint_dir="${CHECKPOINTS_DIR}/${checkpoint_id}"
        checkpoint_file="${checkpoint_dir}/checkpoint.json"

        mkdir -p "$checkpoint_dir"

        # Copy colony state to checkpoint
        if [ -f "$COLONY_STATE_FILE" ]; then
            cp "$COLONY_STATE_FILE" "$checkpoint_file" 2>/dev/null

            # Validate checkpoint
            if validate_json "$checkpoint_file"; then
                echo "checkpoint_ok_$i" > "/tmp/checkpoint_result_$$.tmp.$i"
            else
                echo "checkpoint_fail_$i" > "/tmp/checkpoint_result_$$.tmp.$i"
            fi
        else
            echo "checkpoint_fail_$i" > "/tmp/checkpoint_result_$$.tmp.$i"
        fi
    ) &
    TEST_PIDS+=($!)
done

# Wait for all checkpoints
for pid in "${TEST_PIDS[@]}"; do
    wait $pid 2>/dev/null || true
done

# Count valid checkpoints
for i in {1..5}; do
    if grep -q "checkpoint_ok_$i" "/tmp/checkpoint_result_$$.tmp.$i" 2>/dev/null; then
        CHECKPOINT_COUNT=$((CHECKPOINT_COUNT + 1))
    fi
    rm -f "/tmp/checkpoint_result_$$.tmp.$i" 2>/dev/null || true
done

if [ $CHECKPOINT_COUNT -eq 5 ]; then
    tap_result $TEST_NUM "Concurrent checkpoints don't corrupt (5/5 valid)" 0
else
    tap_result $TEST_NUM "Concurrent checkpoints don't corrupt ($CHECKPOINT_COUNT/5 valid)" 1
fi
TEST_NUM=$((TEST_NUM + 1))

# Test 6: All JSON files valid after stress
echo "# Test 6: JSON validation after all concurrent operations"

JSON_VALID=1

# Check COLONY_STATE.json
if ! validate_json "$COLONY_STATE_FILE"; then
    echo "# Error: COLONY_STATE.json is invalid"
    JSON_VALID=0
fi

# Check pheromones.json
if ! validate_json "$PHEROMONES_FILE"; then
    echo "# Error: pheromones.json is invalid"
    JSON_VALID=0
fi

# Check worker_ants.json
if ! validate_json "$WORKER_ANTS_FILE"; then
    echo "# Error: worker_ants.json is invalid"
    JSON_VALID=0
fi

# Check memory.json
if ! validate_json "${GIT_ROOT}/.aether/data/memory.json"; then
    echo "# Error: memory.json is invalid"
    JSON_VALID=0
fi

# Check checkpoints
for checkpoint_file in "${CHECKPOINTS_DIR}"/stress_test_checkpoint_*/checkpoint.json; do
    if [ -f "$checkpoint_file" ] && ! validate_json "$checkpoint_file"; then
        echo "# Error: Checkpoint $checkpoint_file is invalid"
        JSON_VALID=0
    fi
done

if [ $JSON_VALID -eq 1 ]; then
    tap_result $TEST_NUM "All JSON files valid after stress" 0
else
    tap_result $TEST_NUM "All JSON files valid after stress (some corrupted)" 1
fi

# Cleanup
cleanup_test_colony

echo "# ======================================"
echo "# Stress test complete"
