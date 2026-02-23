#!/usr/bin/env bash
# Weekly Colony Health Audit
# Run: bash .aether/scripts/weekly-audit.sh

set -euo pipefail

AETHER_ROOT="${AETHER_ROOT:-$(pwd)}"
DATA_DIR="$AETHER_ROOT/.aether/data"
UTILS="$AETHER_ROOT/.aether/aether-utils.sh"

echo "# Colony Audit Report - $(date -u +%Y-%m-%d)"
echo ""

state_size=$(wc -c < "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo 0)
pheromone_size=$(wc -c < "$DATA_DIR/pheromones.json" 2>/dev/null || echo 0)
observations_size=$(wc -c < "$DATA_DIR/learning-observations.json" 2>/dev/null || echo 0)
echo "## Memory Sizes"
echo "- COLONY_STATE.json: $state_size bytes"
echo "- pheromones.json: $pheromone_size bytes"
echo "- learning-observations.json: $observations_size bytes"
echo ""

signal_count=$(jq '.signals | length' "$DATA_DIR/pheromones.json" 2>/dev/null || echo 0)
expired_count=$(jq '[.signals[]? | select(.active == false)] | length' "$DATA_DIR/pheromones.json" 2>/dev/null || echo 0)
echo "## Pheromone Health"
echo "- Total signals: $signal_count"
echo "- Expired signals: $expired_count"
echo ""

spawn_eff=$(bash "$UTILS" spawn-efficiency 2>/dev/null | jq -c '.result // {}' 2>/dev/null || echo '{}')
total_spawned=$(echo "$spawn_eff" | jq -r '.total // 0')
completed_spawned=$(echo "$spawn_eff" | jq -r '.completed // 0')
efficiency_pct=$(echo "$spawn_eff" | jq -r '.efficiency_pct // 0')
echo "## Spawn Efficiency"
echo "- Total spawned: $total_spawned"
echo "- Completed: $completed_spawned"
echo "- Efficiency: ${efficiency_pct}%"
echo ""

blocker_count=$(jq '[.flags[]? | select(.type == "blocker" and (.resolved_at == null))] | length' "$DATA_DIR/flags.json" 2>/dev/null || echo 0)
echo "## Gate Failures"
echo "- Unresolved blockers: $blocker_count"
echo ""

if [[ -f "$DATA_DIR/midden/midden.json" ]]; then
  oracle_avg=$(jq '
    [
      (.entries[]? | select(.category == "oracle") | (.iterations // 0)),
      (.signals[]? | select(.type == "oracle") | (.iterations // 0))
    ] | flatten | if length > 0 then (add / length) else null end
  ' "$DATA_DIR/midden/midden.json" 2>/dev/null || echo "null")
  echo "## Oracle Metrics"
  if [[ "$oracle_avg" == "null" ]]; then
    echo "- Average iterations: N/A"
  else
    echo "- Average iterations: $oracle_avg"
  fi
  echo ""
fi

entropy=$(bash "$UTILS" entropy-score 2>/dev/null | jq -r '.result.score // "N/A"' 2>/dev/null || echo "N/A")
echo "## Entropy Score"
echo "- Current: $entropy"
echo "- Threshold: 75 (organize required if exceeded)"
echo ""

echo "## Recommendations"
if [[ "$efficiency_pct" =~ ^[0-9]+$ ]] && [[ "$efficiency_pct" -lt 70 ]]; then
  echo "- [HIGH] Spawn efficiency below 70% - review worker task decomposition"
fi
if [[ "$signal_count" =~ ^[0-9]+$ ]] && [[ "$signal_count" -gt 20 ]]; then
  echo "- [MEDIUM] High pheromone count - consider consolidation or expiry acceleration"
fi
if [[ "$blocker_count" =~ ^[0-9]+$ ]] && [[ "$blocker_count" -gt 3 ]]; then
  echo "- [HIGH] Multiple unresolved blockers - run /ant:swarm for auto-repair"
fi
if [[ "$entropy" =~ ^[0-9]+$ ]] && [[ "$entropy" -gt 75 ]]; then
  echo "- [CRITICAL] High entropy - run /ant:organize before next build"
fi
