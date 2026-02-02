#!/bin/bash
# Colony Setup Helper for Tests
# Provides test colony initialization functionality
#
# Usage:
#   source tests/helpers/colony-setup.sh
#   setup_test_colony "Build REST API"

# Get git root directory for path resolution
get_git_root() {
    git rev-parse --show-toplevel 2>/dev/null || echo "$PWD"
}

# Colony state file paths
GIT_ROOT=$(get_git_root)
COLONY_STATE_FILE="${GIT_ROOT}/.aether/data/COLONY_STATE.json"
PHEROMONES_FILE="${GIT_ROOT}/.aether/data/pheromones.json"
WORKER_ANTS_FILE="${GIT_ROOT}/.aether/data/worker_ants.json"

# Setup test colony with goal
# Args: goal_string
# Returns: 0 on success, 1 on failure
# Outputs: Colony path for test access
setup_test_colony() {
    local goal="$1"

    if [ -z "$goal" ]; then
        echo "# Error: Goal argument required for setup_test_colony" >&2
        return 1
    fi

    echo "# Setting up test colony with goal: $goal"

    # Ensure data directory exists
    local data_dir="${GIT_ROOT}/.aether/data"
    if [ ! -d "$data_dir" ]; then
        mkdir -p "$data_dir" || {
            echo "# Error: Failed to create data directory: $data_dir" >&2
            return 1
        }
    fi

    # Initialize COLONY_STATE.json with minimal colony state
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local colony_id="test_colony_$(date +%s)"

    cat > "$COLONY_STATE_FILE" <<EOF
{
  "colony_id": "$colony_id",
  "goal": "$goal",
  "colony_status": {
    "state": "IDLE",
    "current_phase": 1,
    "queen_checkin": null
  },
  "state_machine": {
    "valid_states": ["IDLE", "INIT", "PLANNING", "EXECUTING", "VERIFYING", "COMPLETED", "FAILED"],
    "last_transition": "$timestamp",
    "transitions_count": 0,
    "state_history": []
  },
  "phases": {
    "current_phase": 1,
    "roadmap": []
  },
  "worker_ants": {
    "colonizer": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    },
    "route-setter": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    },
    "builder": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    },
    "watcher": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    },
    "scout": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    },
    "architect": {
      "status": "IDLE",
      "current_task": null,
      "spawned_subagents": 0
    }
  },
  "pheromones": [],
  "memory": {
    "working": [],
    "short_term": [],
    "long_term": []
  },
  "meta_learning": {
    "total_spawns": 0,
    "successful_spawns": 0,
    "failed_spawns": 0,
    "active_specialist_types": [],
    "deprecated_specialist_types": [],
    "specialist_confidence": {},
    "spawn_outcomes": [],
    "last_updated": "$timestamp"
  },
  "resource_budgets": {
    "max_spawns_per_phase": 10,
    "current_spawns": 0,
    "max_spawn_depth": 3,
    "circuit_breaker_trips": 0,
    "circuit_breaker_cooldown_until": null
  },
  "spawn_tracking": {
    "depth": 0,
    "total_spawns": 0,
    "spawn_history": [],
    "failed_specialist_types": [],
    "cooldown_specialists": [],
    "circuit_breaker_history": []
  },
  "performance_metrics": {
    "successful_spawns": 0,
    "failed_spawns": 0,
    "avg_spawn_duration_seconds": 0
  },
  "checkpoints": {
    "checkpoint_count": 0,
    "latest_checkpoint": null
  },
  "verification": {
    "votes": [],
    "issues": [],
    "outcome": "pending"
  },
  "created_at": "$timestamp",
  "updated_at": "$timestamp"
}
EOF

    if [ $? -ne 0 ]; then
        echo "# Error: Failed to create COLONY_STATE.json" >&2
        return 1
    fi

    echo "# Colony state initialized at: $COLONY_STATE_FILE"

    # Initialize pheromones.json
    cat > "$PHEROMONES_FILE" <<EOF
{
  "active_pheromones": [],
  "metadata": {
    "last_updated": "$timestamp",
    "total_pheromones": 0
  }
}
EOF

    if [ $? -ne 0 ]; then
        echo "# Error: Failed to create pheromones.json" >&2
        return 1
    fi

    echo "# Pheromones initialized at: $PHEROMONES_FILE"

    # Initialize worker_ants.json
    cat > "$WORKER_ANTS_FILE" <<EOF
{
  "active_workers": [],
  "worker_registry": {
    "colonizer": {
      "caste": "colonizer",
      "status": "IDLE",
      "capabilities": [
        "codebase_analysis",
        "semantic_indexing",
        "structure_mapping"
      ]
    },
    "route-setter": {
      "caste": "route-setter",
      "status": "IDLE",
      "capabilities": [
        "phase_planning",
        "route_establishment",
        "coordination"
      ]
    },
    "builder": {
      "caste": "builder",
      "status": "IDLE",
      "capabilities": [
        "code_generation",
        "implementation",
        "testing"
      ]
    },
    "watcher": {
      "caste": "watcher",
      "status": "IDLE",
      "capabilities": [
        "verification",
        "quality_assurance",
        "validation"
      ]
    },
    "scout": {
      "caste": "scout",
      "status": "IDLE",
      "capabilities": [
        "research",
        "information_gathering",
        "context_analysis"
      ]
    },
    "architect": {
      "caste": "architect",
      "status": "IDLE",
      "capabilities": [
        "knowledge_synthesis",
        "memory_management",
        "pattern_extraction"
      ]
    }
  },
  "specialist_mappings": {
    "capability_to_caste": {
      "database": "scout",
      "security": "watcher",
      "testing": "watcher",
      "api": "route_setter",
      "frontend": "builder",
      "backend": "builder",
      "performance": "architect",
      "documentation": "scout",
      "infrastructure": "builder",
      "research": "scout",
      "planning": "route_setter",
      "memory": "architect"
    }
  },
  "spawn_count": 0,
  "last_updated": "$timestamp"
}
EOF

    if [ $? -ne 0 ]; then
        echo "# Error: Failed to create worker_ants.json" >&2
        return 1
    fi

    echo "# Worker ants initialized at: $WORKER_ANTS_FILE"

    # Initialize memory.json with three-layer memory structure
    MEMORY_FILE="${GIT_ROOT}/.aether/data/memory.json"
    cat > "$MEMORY_FILE" <<EOF
{
  "working_memory": {
    "max_capacity_tokens": 200000,
    "current_tokens": 0,
    "items": []
  },
  "short_term_memory": {
    "max_sessions": 10,
    "current_sessions": 0,
    "sessions": []
  },
  "long_term_memory": {
    "patterns": []
  },
  "metrics": {
    "total_compressions": 0,
    "average_compression_ratio": 0,
    "working_memory_evictions": 0,
    "short_term_evictions": 0,
    "total_pattern_extractions": 0
  }
}
EOF

    if [ $? -ne 0 ]; then
        echo "# Error: Failed to create memory.json" >&2
        return 1
    fi

    echo "# Memory initialized at: $MEMORY_FILE"

    # Initialize watcher_weights.json
    WATCHER_WEIGHTS_FILE="${GIT_ROOT}/.aether/data/watcher_weights.json"
    cat > "$WATCHER_WEIGHTS_FILE" <<EOF
{
  "watcher_weights": {
    "security": 1.0,
    "performance": 1.0,
    "quality": 1.0,
    "test_coverage": 1.0
  },
  "weight_bounds": {
    "min": 0.1,
    "max": 3.0
  },
  "last_updated": "$timestamp"
}
EOF

    if [ $? -ne 0 ]; then
        echo "# Error: Failed to create watcher_weights.json" >&2
        return 1
    fi

    echo "# Watcher weights initialized at: $WATCHER_WEIGHTS_FILE"
    echo "# Test colony setup complete"
    echo "# Colony ID: $colony_id"

    return 0
}

# Verify colony state exists
# Returns: 0 if colony state exists, 1 if not
verify_colony_state() {
    if [ ! -f "$COLONY_STATE_FILE" ]; then
        echo "# Error: Colony state file not found: $COLONY_STATE_FILE" >&2
        return 1
    fi

    if [ ! -f "$PHEROMONES_FILE" ]; then
        echo "# Error: Pheromones file not found: $PHEROMONES_FILE" >&2
        return 1
    fi

    if [ ! -f "$WORKER_ANTS_FILE" ]; then
        echo "# Error: Worker ants file not found: $WORKER_ANTS_FILE" >&2
        return 1
    fi

    if [ ! -f "${GIT_ROOT}/.aether/data/memory.json" ]; then
        echo "# Error: Memory file not found" >&2
        return 1
    fi

    return 0
}

# Get colony goal from state
# Returns: Goal string or empty if not set
get_colony_goal() {
    if [ -f "$COLONY_STATE_FILE" ]; then
        jq -r '.goal // empty' "$COLONY_STATE_FILE" 2>/dev/null
    fi
}

# Get colony state
# Returns: State string (IDLE, INIT, etc.)
get_colony_state() {
    if [ -f "$COLONY_STATE_FILE" ]; then
        jq -r '.colony_status.state // "IDLE"' "$COLONY_STATE_FILE" 2>/dev/null
    else
        echo "IDLE"
    fi
}

# Export functions for use in tests
export -f get_git_root setup_test_colony verify_colony_state get_colony_goal get_colony_state
export GIT_ROOT COLONY_STATE_FILE PHEROMONES_FILE WORKER_ANTS_FILE
