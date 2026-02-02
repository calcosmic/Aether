#!/bin/bash
# Aether Performance Baseline Measurement Test
# Measures timing, file I/O, subprocess spawns for all colony operations
#
# Usage:
#   bash tests/performance/timing-baseline.test.sh
#
# Output:
#   - TAP format test results
#   - JSON baseline file with metrics

set -e

# Get script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASE_DIR="$(dirname "$SCRIPT_DIR")"
GIT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PHASE_DIR/../..")"

# Source helpers
HELPER_DIR="${SCRIPT_DIR}/../helpers"
if [ -f "$HELPER_DIR/colony-setup.sh" ]; then
    source "$HELPER_DIR/colony-setup.sh"
else
    echo "Error: colony-setup.sh not found at $HELPER_DIR" >&2
    exit 1
fi

if [ -f "$HELPER_DIR/cleanup.sh" ]; then
    source "$HELPER_DIR/cleanup.sh"
else
    echo "Error: cleanup.sh not found at $HELPER_DIR" >&2
    exit 1
fi

# Source metrics tracking
if [ -f "${SCRIPT_DIR}/metrics-tracking.sh" ]; then
    source "${SCRIPT_DIR}/metrics-tracking.sh"
else
    echo "Error: metrics-tracking.sh not found at ${SCRIPT_DIR}" >&2
    exit 1
fi

# Results directory
RESULTS_DIR="${GIT_ROOT}/.planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/performance/results"
BASELINE_DATE=$(date +%Y%m%d)
BASELINE_FILE="${RESULTS_DIR}/baseline-${BASELINE_DATE}.json"

# Ensure results directory exists
mkdir -p "$RESULTS_DIR"

# TAP header: 8 tests
echo "1..8"
echo "# Aether Performance Baseline Measurement"
echo "# ========================================="
echo ""

# Helper function to measure an operation 3 times and report median
# Arguments: test_number, operation_name, operation_function
measure_operation() {
    local test_number="$1"
    local operation_name="$2"
    local operation_function="$3"

    echo "# Measuring: $operation_name"

    # Run operation 3 times
    local durations=()
    local file_ios=()
    local subprocesses=()
    local tokens=()
    local memories=()

    for i in 1 2 3; do
        # Record start time
        local start_file_io=$(count_file_io)
        local start_tokens=$(estimate_tokens)

        # Measure operation timing
        local start_time=$(date +%s.%N)
        $operation_function
        local end_time=$(date +%s.%N)

        # Calculate duration
        local duration=$(echo "scale=6; $end_time - $start_time" | bc)
        durations+=("$duration")

        # Record other metrics
        local end_file_io=$(count_file_io)
        local file_io_delta=$((end_file_io - start_file_io))
        file_ios+=("$file_io_delta")

        # Estimate tokens after operation
        local end_tokens=$(estimate_tokens)
        local token_delta=$((end_tokens - start_tokens))
        tokens+=("$token_delta")

        # Memory footprint
        local memory_kb=$(get_memory_footprint)
        memories+=("$memory_kb")

        echo "#   Run $i: ${duration}s"
    done

    # Calculate median (sort and pick middle)
    local sorted_durations=$(printf '%s\n' "${durations[@]}" | sort -n)
    local median=$(echo "$sorted_durations" | awk 'NR==2')
    local min=$(echo "$sorted_durations" | head -1)
    local max=$(echo "$sorted_durations" | tail -1)

    # Take median of other metrics too
    local sorted_file_ios=$(printf '%s\n' "${file_ios[@]}" | sort -n)
    local median_file_io=$(echo "$sorted_file_ios" | awk 'NR==2')

    local sorted_tokens=$(printf '%s\n' "${tokens[@]}" | sort -n)
    local median_tokens=$(echo "$sorted_tokens" | awk 'NR==2')

    local sorted_memories=$(printf '%s\n' "${memories[@]}" | sort -n)
    local median_memory=$(echo "$sorted_memories" | awk 'NR==2')

    # Output TAP result
    echo "ok ${test_number} - ${operation_name}: ${median}s (min: ${min}s, max: ${max}s)"
    echo "#     File I/O: ${median_file_io}, Tokens: ${median_tokens}, Memory: ${median_memory}KB"

    # Save metrics for JSON output
    local op_id=$(echo "$operation_name" | tr '[:upper:]' '[:lower:]' | tr ' ' '_')

    # Store in temporary file for later JSON assembly (as complete JSON object)
    cat > "${RESULTS_DIR}/temp_metrics_${test_number}.json" << EOF
{
  "${op_id}": {
    "median_s": ${median},
    "min_s": ${min},
    "max_s": ${max},
    "file_io_count": ${median_file_io},
    "token_estimate": ${median_tokens},
    "memory_kb": ${median_memory}
  }
}
EOF
}

# === OPERATIONS TO MEASURE ===

# Operation 1: Colony initialization
operation_colony_init() {
    setup_test_colony "Performance test goal" >/dev/null 2>&1
}

# Operation 2: Pheromone emission
operation_pheromone_emit() {
    # Simulate pheromone emission by updating pheromones.json
    local pheromones_file="${GIT_ROOT}/.aether/data/pheromones.json"
    if [ -f "$pheromones_file" ]; then
        jq '.active_pheromones += [
            {
                "type": "focus",
                "signal": "performance_testing",
                "strength": 0.7,
                "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
            }
        ]' "$pheromones_file" > "${pheromones_file}.tmp" && \
        mv "${pheromones_file}.tmp" "$pheromones_file"
    fi
}

# Operation 3: State transition
operation_state_transition() {
    # Simulate state transition by updating COLONY_STATE.json
    local colony_state="${GIT_ROOT}/.aether/data/COLONY_STATE.json"
    if [ -f "$colony_state" ]; then
        jq '.colony_status.state = "INIT"' "$colony_state" > "${colony_state}.tmp" && \
        mv "${colony_state}.tmp" "$colony_state"
    fi
}

# Operation 4: Memory compression
operation_memory_compress() {
    # Simulate memory compression operation
    local memory_file="${GIT_ROOT}/.aether/data/memory.json"
    if [ -f "$memory_file" ]; then
        # Read and rewrite memory file (simulates compression operation)
        local content=$(cat "$memory_file")
        echo "$content" | jq '.metrics.total_compressions += 1' > "${memory_file}.tmp" && \
        mv "${memory_file}.tmp" "$memory_file"
    fi
}

# Operation 5: Spawn decision
operation_spawn_decision() {
    # Simulate spawn decision analysis
    # Use spawn-decision utility to analyze task requirements
    local spawn_decision="${GIT_ROOT}/.aether/utils/spawn-decision.sh"
    if [ -f "$spawn_decision" ]; then
        source "$spawn_decision" 2>/dev/null
        analyze_task_requirements "Implement REST API endpoint" > /dev/null 2>&1 || true
    fi
}

# Operation 6: Vote aggregation
operation_vote_aggregation() {
    # Simulate vote aggregation
    # Create temporary votes and aggregate
    local votes_dir=$(mktemp -d)
    echo '{"watcher": "security", "vote": "approve", "confidence": 0.9}' > "${votes_dir}/vote1.json"
    echo '{"watcher": "performance", "vote": "approve", "confidence": 0.85}' > "${votes_dir}/vote2.json"
    echo '{"watcher": "quality", "vote": "approve", "confidence": 0.88}' > "${votes_dir}/vote3.json"
    echo '{"watcher": "test_coverage", "vote": "approve", "confidence": 0.92}' > "${votes_dir}/vote4.json"

    local vote_aggregator="${GIT_ROOT}/.aether/utils/vote-aggregator.sh"
    if [ -f "$vote_aggregator" ]; then
        source "$vote_aggregator" 2>/dev/null
        aggregate_votes "$votes_dir" > /dev/null 2>&1 || true
    fi

    rm -rf "$votes_dir"
}

# Operation 7: Event publish
operation_event_publish() {
    # Simulate event publishing
    local event_bus="${GIT_ROOT}/.aether/utils/event-bus.sh"
    if [ -f "$event_bus" ]; then
        source "$event_bus" 2>/dev/null
        publish_event "test_topic" "test_type" '{"test": "data"}' "test_publisher" "test" > /dev/null 2>&1 || true
    fi
}

# Operation 8: Full workflow
operation_full_workflow() {
    # Simulate complete workflow
    setup_test_colony "Full workflow test" >/dev/null 2>&1
    operation_pheromone_emit
    operation_state_transition
    operation_memory_compress
    cleanup_test_colony >/dev/null 2>&1
}

# === RUN MEASUREMENTS ===

# Setup: Start with clean slate
echo "# Setup: Cleaning slate..."
cleanup_test_colony >/dev/null 2>&1 || true
echo "#"

# Measure each operation
measure_operation 1 "Colony init" operation_colony_init
measure_operation 2 "Pheromone emit" operation_pheromone_emit
measure_operation 3 "State transition" operation_state_transition
measure_operation 4 "Memory compress" operation_memory_compress
measure_operation 5 "Spawn decision" operation_spawn_decision
measure_operation 6 "Vote aggregation" operation_vote_aggregation
measure_operation 7 "Event publish" operation_event_publish
measure_operation 8 "Full workflow" operation_full_workflow

# === GENERATE BASELINE JSON ===

echo "#"
echo "# Generating baseline JSON..."

# Detect hardware
hardware_json=$(detect_hardware)

# Combine all metrics into single JSON
operations_json=$(cat ${RESULTS_DIR}/temp_metrics_*.json | jq -s 'add')

# Create final baseline JSON
jq -n \
    --argjson hardware "$hardware_json" \
    --argjson operations "$operations_json" \
    '{
        timestamp: "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
        hardware: $hardware,
        operations: $operations
    }' > "$BASELINE_FILE"

# Clean up temp files
rm -f ${RESULTS_DIR}/temp_metrics_*.json

echo "# Baseline saved to: $BASELINE_FILE"

# Final cleanup
cleanup_test_colony >/dev/null 2>&1 || true

echo ""
echo "# === Baseline Summary ==="
echo "# Baseline file: $BASELINE_FILE"
jq -r '.hardware' "$BASELINE_FILE" | jq -r 'to_entries | .[] | "# \(.key): \(.value)"'
echo "#"
echo "# Operations measured:"
jq -r '.operations | to_entries[] | "#   - \(.key): \(.value.median_s)s (median)"' "$BASELINE_FILE"
