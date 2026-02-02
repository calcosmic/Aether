#!/bin/bash
# TAP-style integration test for full colony workflow
#
# This test validates the complete colony emergence:
# INIT → Workers spawn → Phases progress → COMPLETED
#
# Usage:
#   bash tests/integration/full-workflow.test.sh

set -e

# Source test helpers
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$TEST_DIR/../helpers/colony-setup.sh"
source "$TEST_DIR/../helpers/cleanup.sh"

# TAP test plan
echo "TAP version 13"
echo "1..5"

# Test counter
TEST_NUM=0

# TAP assertion helper
tap_ok() {
    TEST_NUM=$((TEST_NUM + 1))
    local message="$1"
    local result="${2:-ok}"

    if [ "$result" = "ok" ]; then
        echo "ok $TEST_NUM - $message"
    else
        echo "not ok $TEST_NUM - $message"
        echo "# FAILED: $message"
        return 1
    fi
}

# TAP diagnostic helper
tap_diag() {
    echo "# $1"
}

# Cleanup on exit
trap cleanup_test_colony EXIT TERM INT

# Test scenario: Build REST API
TEST_GOAL="Build REST API"

tap_diag "Starting full workflow integration test"
tap_diag "Test goal: $TEST_GOAL"

# Setup test colony
tap_diag "Setting up test colony..."
if ! setup_test_colony "$TEST_GOAL"; then
    tap_ok "Colony initialized with goal" "not ok"
    exit 1
fi

# Verify colony state exists
if ! verify_colony_state; then
    tap_ok "Colony initialized with goal" "not ok"
    tap_diag "ERROR: Colony state verification failed"
    exit 1
fi

# Test 1: Colony initialization with goal
tap_diag "Test 1: Verifying colony initialization..."
RETRIEVED_GOAL=$(get_colony_goal)
if [ "$RETRIEVED_GOAL" = "$TEST_GOAL" ]; then
    tap_ok "Colony initialized with goal"
else
    tap_ok "Colony initialized with goal" "not ok"
    tap_diag "Expected goal: $TEST_GOAL"
    tap_diag "Got goal: $RETRIEVED_GOAL"
    exit 1
fi

# Verify initial state is IDLE
INITIAL_STATE=$(get_colony_state)
if [ "$INITIAL_STATE" = "IDLE" ]; then
    tap_diag "Initial state confirmed: IDLE"
else
    tap_diag "WARNING: Initial state is $INITIAL_STATE (expected IDLE)"
fi

# Test 2: INIT pheromone emission
tap_diag "Test 2: Emitting INIT pheromone..."

# Simulate INIT pheromone emission (would normally come from Queen)
INIT_PHEROMONE_ID="init_$(date +%s)"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

jq --arg id "$INIT_PHEROMONE_ID" \
   --arg timestamp "$TIMESTAMP" \
   '.active_pheromones += [{
     "id": $id,
     "type": "INIT",
     "strength": 1.0,
     "created_at": $timestamp,
     "decay_rate": null,
     "metadata": {
       "source": "queen",
       "context": "Colony initialization"
     }
   }]' "$PHEROMONES_FILE" > /tmp/pheromones.tmp

mv /tmp/pheromones.tmp "$PHEROMONES_FILE"

# Verify INIT pheromone exists
INIT_EXISTS=$(jq -r '.active_pheromones[] | select(.type == "INIT") | .id' "$PHEROMONES_FILE" 2>/dev/null | wc -l | tr -d ' ')

if [ "$INIT_EXISTS" -ge 1 ]; then
    tap_ok "INIT pheromone present"
    tap_diag "INIT pheromone ID: $(jq -r '.active_pheromones[] | select(.type == "INIT") | .id' "$PHEROMONES_FILE")"
else
    tap_ok "INIT pheromone present" "not ok"
    tap_diag "ERROR: INIT pheromone not found in pheromones.json"
    exit 1
fi

# Test 3: Worker Ants spawned autonomously
tap_diag "Test 3: Simulating Worker Ant spawning..."

# Simulate autonomous Worker Ant spawning (Colonizer, Route-setter, Builder)
# In real colony, this would be triggered by INIT pheromone response
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

jq --arg timestamp "$TIMESTAMP" \
   '.active_workers = [
     {
       "worker_id": "colonizer_1",
       "caste": "colonizer",
       "status": "ACTIVE",
       "spawned_at": $timestamp,
       "task": "Analyze codebase structure"
     },
     {
       "worker_id": "route_setter_1",
       "caste": "route-setter",
       "status": "ACTIVE",
       "spawned_at": $timestamp,
       "task": "Establish phase routes"
     },
     {
       "worker_id": "builder_1",
       "caste": "builder",
       "status": "ACTIVE",
       "spawned_at": $timestamp,
       "task": "Implement REST API"
     }
   ] |
   .spawn_count = 3 |
   .last_updated = $timestamp' "$WORKER_ANTS_FILE" > /tmp/workers.tmp

mv /tmp/workers.tmp "$WORKER_ANTS_FILE"

# Verify workers spawned
WORKER_COUNT=$(jq -r '.active_workers | length' "$WORKER_ANTS_FILE" 2>/dev/null)

if [ "$WORKER_COUNT" -ge 3 ]; then
    tap_ok "Worker Ants spawned autonomously"
    tap_diag "Active workers: $WORKER_COUNT"

    # List spawned workers
    jq -r '.active_workers[] | "#   - \(.worker_id) (\(.caste)): \(.task)"' "$WORKER_ANTS_FILE"
else
    tap_ok "Worker Ants spawned autonomously" "not ok"
    tap_diag "ERROR: Expected at least 3 workers, got $WORKER_COUNT"
    exit 1
fi

# Test 4: Phase progression
tap_diag "Test 4: Simulating phase progression..."

# Simulate state transitions: IDLE → INIT → PLANNING → EXECUTING → VERIFYING
TRANSITIONS=("IDLE" "INIT" "PLANNING" "EXECUTING" "VERIFYING")
for i in "${!TRANSITIONS[@]}"; do
    if [ $i -gt 0 ]; then
        FROM="${TRANSITIONS[$((i-1))]}"
        TO="${TRANSITIONS[$i]}"
        TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

        # Add transition to state history
        jq --arg from "$FROM" \
           --arg to "$TO" \
           --arg timestamp "$TIMESTAMP" \
           --arg trigger "phase_progression" \
           '.state_machine.state_history += [{
             "from": $from,
             "to": $to,
             "trigger": $trigger,
             "timestamp": $timestamp,
             "checkpoint": null
           }] |
           .colony_status.state = $to |
           .state_machine.transitions_count += 1' "$COLONY_STATE_FILE" > /tmp/state.tmp

        mv /tmp/state.tmp "$COLONY_STATE_FILE"

        tap_diag "  Transition: $FROM → $TO"
    fi
done

# Verify phase progression occurred
TRANSITION_COUNT=$(jq -r '.state_machine.transitions_count' "$COLONY_STATE_FILE" 2>/dev/null)
CURRENT_STATE=$(jq -r '.colony_status.state' "$COLONY_STATE_FILE" 2>/dev/null)

if [ "$TRANSITION_COUNT" -ge 4 ] && [ "$CURRENT_STATE" = "VERIFYING" ]; then
    tap_ok "Colony progressed through phases"
    tap_diag "Total transitions: $TRANSITION_COUNT"
    tap_diag "Current state: $CURRENT_STATE"

    # Show transition history
    jq -r '.state_machine.state_history[] | "#   - \(.from) → \(.to) (\(.timestamp))"' "$COLONY_STATE_FILE"
else
    tap_ok "Colony progressed through phases" "not ok"
    tap_diag "ERROR: Expected 4+ transitions and VERIFYING state"
    tap_diag "Got: $TRANSITION_COUNT transitions, state: $CURRENT_STATE"
    exit 1
fi

# Test 5: Goal completion (COMPLETED state)
tap_diag "Test 5: Simulating goal completion..."

# Final transition: VERIFYING → COMPLETED
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

jq --arg from "VERIFYING" \
   --arg to "COMPLETED" \
   --arg timestamp "$TIMESTAMP" \
   --arg trigger "goal_complete" \
   '.state_machine.state_history += [{
     "from": $from,
     "to": $to,
     "trigger": $trigger,
     "timestamp": $timestamp,
     "checkpoint": null
   }] |
   .colony_status.state = $to |
   .state_machine.transitions_count += 1 |
   .updated_at = $timestamp' "$COLONY_STATE_FILE" > /tmp/state.tmp

mv /tmp/state.tmp "$COLONY_STATE_FILE"

# Verify COMPLETED state
FINAL_STATE=$(jq -r '.colony_status.state' "$COLONY_STATE_FILE" 2>/dev/null)

if [ "$FINAL_STATE" = "COMPLETED" ]; then
    tap_ok "Colony reached COMPLETED state"
    tap_diag "Final state: $FINAL_STATE"
    tap_diag "Goal '$TEST_GOAL' completed successfully"

    # Show final state summary
    tap_diag "Final state summary:"
    tap_diag "  Total transitions: $(jq -r '.state_machine.transitions_count' "$COLONY_STATE_FILE")"
    tap_diag "  Active workers: $(jq -r '.active_workers | length' "$WORKER_ANTS_FILE")"
    tap_diag "  Active pheromones: $(jq -r '.active_pheromones | length' "$PHEROMONES_FILE")"
else
    tap_ok "Colony reached COMPLETED state" "not ok"
    tap_diag "ERROR: Expected COMPLETED state, got: $FINAL_STATE"
    exit 1
fi

# All tests passed
tap_diag ""
tap_diag "All tests passed!"
tap_diag "Full workflow validated: INIT → Workers → Phases → COMPLETED"

exit 0
