#!/bin/bash
# Aether Metrics Tracking Utility
# Tracks performance metrics: timing, file I/O, subprocess spawns, token usage
#
# Usage:
#   source tests/performance/metrics-tracking.sh
#   start_op=$(date +%s.%N)
#   # ... perform operation ...
#   end_op=$(date +%s.%N)
#   track_metrics "operation_name" "$start_op" "$end_op"
#   generate_report baseline.json current.json

# Get git root directory
get_git_root() {
    git rev-parse --show-toplevel 2>/dev/null || echo "$PWD"
}

# Metrics storage
GIT_ROOT=$(get_git_root)
RESULTS_DIR="${GIT_ROOT}/.planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/performance/results"
HISTORY_FILE="${RESULTS_DIR}/history.jsonl"

# Ensure results directory exists
ensure_results_dir() {
    if [ ! -d "$RESULTS_DIR" ]; then
        mkdir -p "$RESULTS_DIR" || {
            echo "Error: Failed to create results directory: $RESULTS_DIR" >&2
            return 1
        }
    fi
}

# Count file I/O operations in .aether/data
# Returns: Number of JSON files
count_file_io() {
    local data_dir="${GIT_ROOT}/.aether/data"
    if [ -d "$data_dir" ]; then
        find "$data_dir" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ' || echo "0"
    else
        echo "0"
    fi
}

# Estimate token usage from .aether/data JSON files
# Heuristic: 4 characters per token
# Returns: Estimated token count
estimate_tokens() {
    local data_dir="${GIT_ROOT}/.aether/data"
    if [ -d "$data_dir" ]; then
        local total_chars=$(find "$data_dir" -name "*.json" -type f -exec wc -c {} + 2>/dev/null | awk '{sum += $1} END {print sum}')
        # Default to 0 if no files found
        total_chars=${total_chars:-0}
        # Use bc for floating point division
        echo "scale=0; $total_chars / 4" | bc 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

# Get memory footprint (RSS in KB)
# Returns: Resident set size in KB
get_memory_footprint() {
    # Get current bash process RSS
    if command -v ps >/dev/null 2>&1; then
        ps -o rss= -p $$ | tr -d ' ' || echo "0"
    else
        echo "0"
    fi
}

# Detect hardware information
# Returns: JSON object with hardware info
detect_hardware() {
    local cpu="Unknown"
    local ram="Unknown"
    local disk="Unknown"

    # Detect hardware on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Get CPU info
        if command -v sysctl >/dev/null 2>&1; then
            local cpu_info=$(sysctl -n machdep.cpu.brand_string 2>/dev/null)
            cpu="${cpu_info:-Unknown}"
        fi

        # Get RAM info
        if command -v sysctl >/dev/null 2>&1; then
            local ram_kb=$(sysctl -n hw.memsize 2>/dev/null)
            if [ -n "$ram_kb" ]; then
                local ram_gb=$((ram_kb / 1024 / 1024 / 1024))
                ram="${ram_gb}GB"
            fi
        fi

        # Disk type (assume SSD for modern Macs)
        disk="SSD"
    fi

    # Detect hardware on Linux
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Get CPU info
        if [ -f /proc/cpuinfo ]; then
            cpu=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | xargs || echo "Unknown")
        fi

        # Get RAM info
        if [ -f /proc/meminfo ]; then
            local ram_kb=$(grep "^MemTotal:" /proc/meminfo | awk '{print $2}')
            if [ -n "$ram_kb" ]; then
                local ram_gb=$((ram_kb / 1024 / 1024))
                ram="${ram_gb}GB"
            fi
        fi

        # Disk type
        disk=$(lsblk -d -o name,rota 2>/dev/null | awk 'NR==2 {if ($2=="0") print "SSD"; else print "HDD"}')
    fi

    # Output as JSON
    jq -n \
        --arg cpu "$cpu" \
        --arg ram "$ram" \
        --arg disk "$disk" \
        '{cpu: $cpu, ram: $ram, disk: $disk}'
}

# Track metrics for an operation
# Arguments: operation_name, start_time, end_time
# Returns: JSON object with metrics
track_metrics() {
    local operation_name="$1"
    local start_time="$2"
    local end_time="$3"

    if [ -z "$operation_name" ] || [ -z "$start_time" ] || [ -z "$end_time" ]; then
        echo "Error: operation_name, start_time, and end_time are required" >&2
        return 1
    fi

    # Calculate duration using bc
    local duration=$(echo "scale=6; $end_time - $start_time" | bc 2>/dev/null || echo "0")

    # Collect metrics
    local file_io_count=$(count_file_io)
    local token_estimate=$(estimate_tokens)
    local memory_kb=$(get_memory_footprint)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Output JSON
    jq -n \
        --arg op "$operation_name" \
        --arg ts "$timestamp" \
        --argjson dur "$duration" \
        --argjson file_io "$file_io_count" \
        --argjson tokens "$token_estimate" \
        --argjson memory "$memory_kb" \
        '{
            operation: $op,
            timestamp: $ts,
            duration_s: $dur,
            file_io_count: $file_io,
            token_estimate: $tokens,
            memory_kb: $memory
        }'
}

# Save metrics to history file (JSONL format)
# Arguments: operation_name, metrics_json
save_metrics() {
    local operation_name="$1"
    local metrics_json="$2"

    ensure_results_dir

    # Append to history file
    echo "$metrics_json" >> "$HISTORY_FILE"

    return 0
}

# Generate comparison report between two baseline files
# Arguments: baseline_file, current_file
# Returns: Formatted table with comparison
generate_report() {
    local baseline_file="$1"
    local current_file="$2"

    if [ ! -f "$baseline_file" ]; then
        echo "Error: Baseline file not found: $baseline_file" >&2
        return 1
    fi

    if [ ! -f "$current_file" ]; then
        echo "Error: Current file not found: $current_file" >&2
        return 1
    fi

    # Print header
    printf "\n%-22s | %-10s | %-10s | %-10s | %-10s\n" "Operation" "Baseline" "Current" "Delta" "Change"
    printf "%-22s-+-%-10s-+-%-10s-+-%-10s-+-%-10s\n" "----------------------" "----------" "----------" "----------" "----------"

    # Get operations from both files
    local operations=$(jq -r '.operations | keys[]' "$baseline_file" 2>/dev/null | sort)

    for op in $operations; do
        # Get baseline timing
        local baseline_time=$(jq -r ".operations.${op}.median_s // 0" "$baseline_file" 2>/dev/null)
        # Get current timing
        local current_time=$(jq -r ".operations.${op}.median_s // 0" "$current_file" 2>/dev/null)

        # Calculate delta and percent change
        local delta=$(echo "scale=6; $current_time - $baseline_time" | bc 2>/dev/null || echo "0")
        local percent_change=0

        if [ "$baseline_time" != "0" ]; then
            percent_change=$(echo "scale=1; ($delta / $baseline_time) * 100" | bc 2>/dev/null || echo "0")
        fi

        # Format delta sign
        local delta_sign="+"
        if [ $(echo "$delta < 0" | bc -l 2>/dev/null || echo "0") -eq 1 ]; then
            delta_sign=""
        fi

        # Determine color coding (improvement vs regression)
        local indicator=""
        local change_str="${delta_sign}${delta}s (${percent_change}%)"

        # >5% faster = improvement, >5% slower = regression
        if [ $(echo "$percent_change < -5" | bc -l 2>/dev/null || echo "0") -eq 1 ]; then
            indicator="✓"  # Improvement
        elif [ $(echo "$percent_change > 5" | bc -l 2>/dev/null || echo "0") -eq 1 ]; then
            indicator="✗"  # Regression
        fi

        # Print row
        printf "%-22s | %-10s | %-10s | %-10s | %-10s\n" "$op" "${baseline_time}s" "${current_time}s" "${delta_sign}${delta}s" "${indicator}${change_str}"
    done

    printf "\n"

    # Identify bottlenecks (slowest 3 operations from current file)
    echo "Bottlenecks (slowest operations):"
    local slowest=$(jq -r '.operations | to_entries | sort_by(.value.median_s) | reverse | .[0:3][] | "\(.key): \(.value.median_s)s"' "$current_file" 2>/dev/null)
    echo "$slowest" | while read -r line; do
        echo "  - $line"
    done
    echo ""
}

# Compare two historical baseline files
# Arguments: file1, file2
# Returns: Comparison showing trend over time
compare_baselines() {
    local file1="$1"
    local file2="$2"

    echo "Comparing baselines:"
    echo "  File 1: $file1"
    echo "  File 2: $file2"
    echo ""

    # Get timestamps
    local ts1=$(jq -r '.timestamp // "Unknown"' "$file1" 2>/dev/null)
    local ts2=$(jq -r '.timestamp // "Unknown"' "$file2" 2>/dev/null)

    echo "Timestamps:"
    echo "  File 1: $ts1"
    echo "  File 2: $ts2"
    echo ""

    # Use generate_report for comparison
    generate_report "$file1" "$file2"
}

# Plot metrics over time (optional, ASCII plot if gnuplot available)
# Arguments: operation_name, output_file
plot_metrics() {
    local operation_name="$1"
    local output_file="${2:-/dev/stdout}"

    ensure_results_dir

    # Check if history file exists
    if [ ! -f "$HISTORY_FILE" ]; then
        echo "Error: History file not found: $HISTORY_FILE" >&2
        return 1
    fi

    # Extract metrics for operation from history
    local data=$(jq -r --arg op "$operation_name" 'select(.operation == $op) | "\(.timestamp)|\(.duration_s)"' "$HISTORY_FILE" 2>/dev/null)

    if [ -z "$data" ]; then
        echo "No data found for operation: $operation_name" >&2
        return 1
    fi

    # Count data points
    local count=$(echo "$data" | wc -l | tr -d ' ')

    echo "Performance trend for '$operation_name' ($count data points):"
    echo ""

    # Simple ASCII plot (table format)
    echo "Timestamp                    | Duration (s)"
    echo "-----------------------------|-------------"
    echo "$data" | while IFS='|' read -r ts dur; do
        printf "%-28s | %s\n" "$ts" "$dur"
    done
    echo ""

    # If gnuplot is available, generate actual plot
    if command -v gnuplot >/dev/null 2>&1; then
        local plot_file="${output_file}.png"
        local temp_data=$(mktemp)

        # Prepare data for gnuplot
        echo "$data" | awk -F'|' '{print NR, $2}' > "$temp_data"

        # Create plot
        gnuplot -e "
            set terminal png size 800,400
            set output '$plot_file'
            set title 'Performance Trend: $operation_name'
            set xlabel 'Run Number'
            set ylabel 'Duration (seconds)'
            plot '$temp_data' with linespoints title 'Duration'
        " 2>/dev/null

        rm -f "$temp_data"

        if [ -f "$plot_file" ]; then
            echo "Plot saved to: $plot_file"
        fi
    else
        echo "Note: gnuplot not available, showing table only"
    fi
}

# Export functions
export -f get_git_root ensure_results_dir count_file_io estimate_tokens
export -f get_memory_footprint detect_hardware track_metrics save_metrics
export -f generate_report compare_baselines plot_metrics
export GIT_ROOT RESULTS_DIR HISTORY_FILE
