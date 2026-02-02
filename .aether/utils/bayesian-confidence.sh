#!/bin/bash
# Aether Bayesian Confidence Library
# Implements Beta distribution confidence calculation for meta-learning
#
# Bayesian inference with Beta(α,β) distribution:
# - Prior: Beta(1,1) represents uniform distribution (no prior knowledge)
# - Success: α_new = α_old + 1 (increment alpha)
# - Failure: β_new = β_old + 1 (increment beta)
# - Confidence (posterior mean): μ = α / (α + β)
#
# Sample size weighting prevents overconfidence from small samples:
# - Weight = min(1.0, (α + β - 2) / 10)
# - Full weight at 10+ samples, reduced weight for fewer samples
#
# This replaces Phase 6's simple +0.1/-0.15 arithmetic with
# mathematically principled Bayesian inference.
#
# Usage:
#   source .aether/utils/bayesian-confidence.sh
#   result=$(update_bayesian_parameters 1 1 "success")  # Returns "2 1"
#   confidence=$(calculate_confidence 2 1)              # Returns "0.666667"
#   weighted=$(calculate_weighted_confidence 2 1)       # Returns "0.370000"
#   stats=$(get_confidence_stats 2 1)                   # Returns JSON object
#   prior=$(initialize_bayesian_prior)                  # Returns '{"alpha":1,"beta":1,...}'

# Source required utilities
# Find Aether root by walking up from the script location or current directory
if [ -n "${BASH_SOURCE[0]}" ]; then
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"
else
    AETHER_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fi

# Try to source from AETHER_ROOT, fallback to relative paths
if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
else
    source ".aether/utils/atomic-write.sh"
fi

if [ -f "$AETHER_ROOT/.aether/utils/file-lock.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/file-lock.sh"
else
    source ".aether/utils/file-lock.sh"
fi

# Configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"
LOCK_FILE="$AETHER_ROOT/.aether/locks/bayesian_confidence.lock"

# Constants
PRIOR_ALPHA=1
PRIOR_BETA=1
MIN_SAMPLES_FOR_WEIGHT=10
BC_SCALE=6

# Update Bayesian parameters based on outcome
# Arguments: alpha, beta, outcome ("success" or "failure")
# Returns: "new_alpha new_beta" (space-separated)
# Uses bc for arithmetic to handle any size integers
update_bayesian_parameters() {
    local alpha="$1"
    local beta="$2"
    local outcome="$3"

    local new_alpha
    local new_beta

    if [ "$outcome" = "success" ]; then
        # Increment alpha (successes)
        new_alpha=$(echo "$alpha + 1" | bc)
        new_beta=$beta
    elif [ "$outcome" = "failure" ]; then
        # Keep alpha, increment beta (failures)
        new_alpha=$alpha
        new_beta=$(echo "$beta + 1" | bc)
    else
        echo "Error: outcome must be 'success' or 'failure', got '$outcome'" >&2
        return 1
    fi

    echo "$new_alpha $new_beta"
}

# Calculate confidence from alpha and beta parameters
# Arguments: alpha, beta
# Returns: confidence as float with 6 decimal places
# Formula: confidence = alpha / (alpha + beta)
# Uses bc with scale=6 for precision
calculate_confidence() {
    local alpha="$1"
    local beta="$2"

    # Use bc for floating-point division with 6 decimal places
    echo "scale=$BC_SCALE; $alpha / ($alpha + $beta)" | bc
}

# Calculate weighted confidence with sample size adjustment
# Arguments: alpha, beta
# Returns: weighted confidence as float with 6 decimal places
# Weight formula: weight = min(1.0, (alpha + beta - 2) / 10)
# Weighted formula: weighted = raw * (0.5 + 0.5 * weight)
# This prevents overconfidence from small samples
calculate_weighted_confidence() {
    local alpha="$1"
    local beta="$2"

    # Calculate raw confidence first
    local raw_confidence
    raw_confidence=$(calculate_confidence "$alpha" "$beta")

    # Calculate sample size (subtract prior counts of 1+1=2)
    local sample_size
    sample_size=$(echo "$alpha + $beta - 2" | bc)

    # Calculate sample size weight (0.0 at 0 samples, 1.0 at 10+ samples)
    local weight
    weight=$(echo "scale=$BC_SCALE; $sample_size / $MIN_SAMPLES_FOR_WEIGHT" | bc)

    # Cap weight at 1.0 using awk (bc doesn't have ternary operator)
    weight=$(awk -v w="$weight" 'BEGIN { print (w > 1.0 ? 1.0 : w) }')

    # Apply weight: weighted = raw * (0.5 + 0.5 * weight)
    # This ensures even with 0 weight, confidence stays at 50% of raw
    local weighted
    weighted=$(echo "scale=$BC_SCALE; $raw_confidence * (0.5 + 0.5 * $weight)" | bc)

    echo "$weighted"
}

# Get comprehensive confidence statistics as JSON
# Arguments: alpha, beta
# Returns: JSON object with all statistics
# Includes: alpha, beta, confidence, weighted_confidence, total_spawns,
#           successful_spawns, failed_spawns, sample_size_weight
get_confidence_stats() {
    local alpha="$1"
    local beta="$2"

    # Calculate all statistics
    local confidence
    local weighted_confidence
    local total_spawns
    local successful_spawns
    local failed_spawns
    local sample_size
    local sample_size_weight

    confidence=$(calculate_confidence "$alpha" "$beta")
    weighted_confidence=$(calculate_weighted_confidence "$alpha" "$beta")
    total_spawns=$(echo "$alpha + $beta - 2" | bc)
    successful_spawns=$(echo "$alpha - 1" | bc)
    failed_spawns=$(echo "$beta - 1" | bc)
    sample_size=$total_spawns

    # Calculate sample size weight for display
    sample_size_weight=$(echo "scale=$BC_SCALE; $sample_size / $MIN_SAMPLES_FOR_WEIGHT" | bc)
    sample_size_weight=$(awk -v w="$sample_size_weight" 'BEGIN { print (w > 1.0 ? 1.0 : w) }')

    # Output JSON object
    cat <<EOF
{
  "alpha": $alpha,
  "beta": $beta,
  "confidence": $confidence,
  "weighted_confidence": $weighted_confidence,
  "total_spawns": $total_spawns,
  "successful_spawns": $successful_spawns,
  "failed_spawns": $failed_spawns,
  "sample_size_weight": $sample_size_weight
}
EOF
}

# Initialize Bayesian prior (uniform distribution)
# Arguments: none
# Returns: JSON string with alpha=1, beta=1, confidence=0.5
# Beta(1,1) represents uniform prior - no prior knowledge
initialize_bayesian_prior() {
    local confidence
    confidence=$(calculate_confidence $PRIOR_ALPHA $PRIOR_BETA)

    cat <<EOF
{
  "alpha": $PRIOR_ALPHA,
  "beta": $PRIOR_BETA,
  "confidence": $confidence
}
EOF
}

# Export functions for use by spawn-outcome-tracker.sh and other scripts
export -f update_bayesian_parameters calculate_confidence calculate_weighted_confidence
export -f get_confidence_stats initialize_bayesian_prior
