#!/bin/bash
# Aether Colony Spawn Helper with Model Assignment
# Usage: spawn-with-model.sh <caste> <task_description> [project_root]
#
# This script sets up the correct environment variables for model routing
# through the LiteLLM proxy before spawning Claude Code.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Arguments
CASTE="${1:-}"
TASK="${2:-}"
PROJECT_ROOT="${3:-$AETHER_ROOT}"

# Validate arguments
[[ -z "$CASTE" ]] && { echo "Usage: spawn-with-model.sh <caste> <task_description> [project_root]" >&2; exit 1; }
[[ -z "$TASK" ]] && { echo "Usage: spawn-with-model.sh <caste> <task_description> [project_root]" >&2; exit 1; }

# DEPRECATED: Per-caste model routing is now handled by agent frontmatter `model:` field.
# This script is preserved for backward compatibility but will be removed in a future version.
# See: .aether/workers.md "Model Selection" section for current routing approach.
echo "[DEPRECATED] spawn-with-model.sh is superseded by agent frontmatter model: field" >&2
echo "  Per-caste routing is now handled by Claude Code natively." >&2
echo "  This script will be removed in a future version." >&2

# Get slot name for this caste from YAML
slot_info=$(bash "$AETHER_ROOT/.aether/aether-utils.sh" model-profile get "$CASTE" 2>/dev/null || echo '{"ok":true,"result":{"model":"inherit"}}')
slot=$(echo "$slot_info" | jq -r '.result.model // "inherit"')

# Resolve slot name to concrete model via model_slots section in YAML
case "$slot" in
  opus|sonnet|haiku)
    resolved=$(node -e "const p=require('./bin/lib/model-profiles').loadModelProfiles('.'); console.log(p.model_slots?.['$slot']||'glm-5-turbo')" 2>/dev/null)
    model="${resolved:-glm-5-turbo}"
    ;;
  inherit|*)
    model="glm-5-turbo"
    ;;
esac

# Log the spawn with model
ant_name=$(bash "$AETHER_ROOT/.aether/aether-utils.sh" generate-ant-name "$CASTE" 2>/dev/null || echo "${CASTE}-$(date +%s)")
bash "$AETHER_ROOT/.aether/aether-utils.sh" spawn-log "Queen" "$CASTE" "$ant_name" "$TASK" 2>/dev/null || true
bash "$AETHER_ROOT/.aether/aether-utils.sh" activity-log "SPAWN" "$ant_name ($CASTE)" "Model: $model - $TASK" 2>/dev/null || true

# Export environment for Claude Code
export ANTHROPIC_BASE_URL="${ANTHROPIC_BASE_URL:-http://localhost:4000}"
export ANTHROPIC_AUTH_TOKEN="${ANTHROPIC_AUTH_TOKEN:-sk-litellm-local}"
export ANTHROPIC_MODEL="$model"

echo "[$(date '+%H:%M:%S')] Spawning $ant_name ($CASTE) with model: $model"
echo "  Task: $TASK"
echo "  Project: $PROJECT_ROOT"

# Check proxy health
if curl -s http://localhost:4000/health | grep -q "healthy" 2>/dev/null; then
    echo "  Proxy: healthy"
else
    echo "  Proxy: unavailable (will use default routing)"
fi

# Start Claude Code with the environment set
# Note: This assumes claude is in PATH
if command -v claude &> /dev/null; then
    claude --cwd "$PROJECT_ROOT"
else
    echo "Claude Code not found in PATH. Environment prepared:"
    echo "  ANTHROPIC_BASE_URL=$ANTHROPIC_BASE_URL"
    echo "  ANTHROPIC_MODEL=$ANTHROPIC_MODEL"
    exit 0
fi
