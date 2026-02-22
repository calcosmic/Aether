#!/bin/bash
# Semantic CLI - Shell interface to Python semantic layer
#
# Provides semantic search and indexing capabilities for Aether colony.
# Uses Python semantic_layer.py for embeddings and similarity search.
#
# Usage:
#   source .aether/utils/semantic-cli.sh
#   semantic-init           # Initialize semantic store
#   semantic-index "text"   # Add text to index
#   semantic-search "query" # Find similar entries
#   semantic-rebuild        # Rebuild from all sources

# Only set strict mode when executed directly, not when sourced
if [[ "${BASH_SOURCE[0]:-$0}" == "${0}" ]]; then
    set -euo pipefail
fi

# Get script directory for relative paths
# Fallback for when BASH_SOURCE isn't set (non-interactive sourcing)
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    SEMANTIC_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
    # Try to find .aether directory from current location
    if [[ -d ".aether" ]]; then
        SEMANTIC_CLI_DIR="$(pwd)/.aether/utils"
    elif [[ -d "../.aether" ]]; then
        SEMANTIC_CLI_DIR="$(cd .. && pwd)/.aether/utils"
    else
        # Last resort - use aether-utils.sh's detected AETHER_DIR if available
        SEMANTIC_CLI_DIR="${AETHER_DIR:-$HOME/.aether/system}/utils"
    fi
fi
AETHER_DIR="$(dirname "$SEMANTIC_CLI_DIR")"
PROJECT_ROOT="$(dirname "$AETHER_DIR")"

# Data directory for semantic store
SEMANTIC_DATA_DIR="${AETHER_DIR}/data/semantic"
SEMANTIC_EMBEDDINGS_FILE="${SEMANTIC_DATA_DIR}/embeddings.json"

# Check if Python dependencies are available
semantic-check-deps() {
    python3 -c "import sys; sys.path.insert(0, '$AETHER_DIR'); from semantic_layer import EmbeddingModel; print('ok')" 2>/dev/null
}

# Initialize semantic store directory
semantic-init() {
    mkdir -p "$SEMANTIC_DATA_DIR"

    if [[ ! -f "$SEMANTIC_DATA_DIR/index.json" ]]; then
        cat > "$SEMANTIC_DATA_DIR/index.json" << 'EOF'
{
  "version": "1.0",
  "entries": [],
  "last_updated": null,
  "stats": {
    "total_entries": 0,
    "by_source": {}
  }
}
EOF
    fi

    if [[ ! -f "$SEMANTIC_EMBEDDINGS_FILE" ]]; then
        echo '{"embeddings":{}}' > "$SEMANTIC_EMBEDDINGS_FILE"
    fi
}

# Index a single text entry
# Usage: semantic-index <text> <source> [entry_id]
semantic-index() {
    local text="${1:-}"
    local source="${2:-unknown}"
    local entry_id="${3:-}"

    if [[ -z "$text" ]]; then
        semantic_json_err 1 "semantic-index requires text argument"
        return 1
    fi

    semantic-init

    # Generate entry ID if not provided
    if [[ -z "$entry_id" ]]; then
        entry_id="${source}_$(date +%s)_$((RANDOM % 10000))"
    fi

    local timestamp
    timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # Call Python to compute embedding and save everything
    python3 << PYTHON_SCRIPT 2>/dev/null
import sys
import json
sys.path.insert(0, '$AETHER_DIR')
from semantic_layer import EmbeddingModel

# Compute embedding
model = EmbeddingModel()
embedding = model.encode('''$text''')

# Load existing embeddings
try:
    with open('$SEMANTIC_EMBEDDINGS_FILE', 'r') as f:
        emb_store = json.load(f)
except:
    emb_store = {'embeddings': {}}

# Store embedding
emb_store['embeddings']['$entry_id'] = {
    'embedding': embedding,
    'text': '''$text''',
    'source': '$source',
    'indexed_at': '$timestamp'
}

# Save embeddings
with open('$SEMANTIC_EMBEDDINGS_FILE', 'w') as f:
    json.dump(emb_store, f)

# Update index
try:
    with open('$SEMANTIC_DATA_DIR/index.json', 'r') as f:
        index = json.load(f)
except:
    index = {'version': '1.0', 'entries': [], 'stats': {}}

index['entries'].append({
    'id': '$entry_id',
    'source': '$source',
    'text_preview': '''${text:0:200}''',
    'indexed_at': '$timestamp'
})
index['last_updated'] = '$timestamp'
index['stats']['total_entries'] = len(index['entries'])

by_source = {}
for e in index['entries']:
    src = e.get('source', 'unknown')
    by_source[src] = by_source.get(src, 0) + 1
index['stats']['by_source'] = by_source

with open('$SEMANTIC_DATA_DIR/index.json', 'w') as f:
    json.dump(index, f, indent=2)

print(json.dumps({
    'ok': True,
    'entry_id': '$entry_id',
    'embedding_dim': len(embedding),
    'source': '$source'
}))
PYTHON_SCRIPT
}

# Search for similar entries
# Usage: semantic-search <query> [top_k] [threshold] [source_filter]
semantic-search() {
    local query="${1:-}"
    local top_k="${2:-5}"
    local threshold="${3:-0.5}"
    local source_filter="${4:-}"

    if [[ -z "$query" ]]; then
        semantic_json_err 1 "semantic-search requires query argument"
        return 1
    fi

    if [[ ! -f "$SEMANTIC_EMBEDDINGS_FILE" ]]; then
        semantic_json_ok '{"results":[]}' "No semantic index found. Run semantic-init first."
        return 0
    fi

    # Call Python to search
    python3 << PYTHON_SCRIPT 2>/dev/null
import sys
import json
import math
sys.path.insert(0, '$AETHER_DIR')
from semantic_layer import EmbeddingModel

def cosine_similarity(v1, v2):
    dot = sum(a*b for a,b in zip(v1, v2))
    norm1 = math.sqrt(sum(a*a for a in v1))
    norm2 = math.sqrt(sum(b*b for b in v2))
    if norm1 == 0 or norm2 == 0:
        return 0.0
    return dot / (norm1 * norm2)

# Compute query embedding
model = EmbeddingModel()
query_emb = model.encode('''$query''')

# Load stored embeddings
with open('$SEMANTIC_EMBEDDINGS_FILE', 'r') as f:
    emb_store = json.load(f)

# Search
results = []
for entry_id, data in emb_store.get('embeddings', {}).items():
    # Source filter
    source = data.get('source', '')
    if '$source_filter' and source != '$source_filter':
        continue

    stored_emb = data.get('embedding', [])
    if not stored_emb:
        continue

    similarity = cosine_similarity(query_emb, stored_emb)

    if similarity >= $threshold:
        results.append({
            'id': entry_id,
            'text': data.get('text', ''),
            'source': source,
            'similarity': round(similarity, 3),
            'indexed_at': data.get('indexed_at', '')
        })

# Sort by similarity
results.sort(key=lambda x: x['similarity'], reverse=True)
results = results[:$top_k]

print(json.dumps({
    'ok': True,
    'query': '''$query''',
    'count': len(results),
    'results': results
}))
PYTHON_SCRIPT
}

# Find similar entries to check for duplicates
semantic-find-duplicate() {
    local text="${1:-}"
    local threshold="${2:-0.85}"

    if [[ -z "$text" ]]; then
        semantic_json_err 1 "semantic-find-duplicate requires text argument"
        return 1
    fi

    local result
    result=$(semantic-search "$text" 3 "$threshold")

    local count
    count=$(echo "$result" | jq -r '.count // 0')

    if [[ "$count" -gt 0 ]]; then
        echo "$result" | jq '. + {"is_duplicate": true}'
    else
        semantic_json_ok "[]" "No duplicates found"
    fi
}

# Rebuild entire index from all Aether data sources
semantic-rebuild() {
    echo "Rebuilding semantic index from all sources..."

    # Reset
    rm -rf "$SEMANTIC_DATA_DIR"/*
    semantic-init

    local count=0

    # Index flags
    if [[ -f "$AETHER_DIR/data/flags.json" ]]; then
        echo "  Indexing flags..."
        local flags_json
        flags_json=$(jq -c '.flags[]' "$AETHER_DIR/data/flags.json" 2>/dev/null || echo "")

        while IFS= read -r flag; do
            [[ -z "$flag" ]] && continue
            local flag_id flag_title flag_desc
            flag_id=$(echo "$flag" | jq -r '.id // empty')
            flag_title=$(echo "$flag" | jq -r '.title // empty')
            flag_desc=$(echo "$flag" | jq -r '.description // empty')

            if [[ -n "$flag_title" ]]; then
                semantic-index "$flag_title: $flag_desc" "flags" "$flag_id" >/dev/null 2>&1 || true
                ((count++)) || true
            fi
        done <<< "$flags_json"
    fi

    # Index dreams
    if [[ -d "$AETHER_DIR/dreams" ]]; then
        echo "  Indexing dreams..."
        for dream in "$AETHER_DIR/dreams"/*.md; do
            [[ -f "$dream" ]] || continue
            local dream_id dream_content
            dream_id=$(basename "$dream" .md)
            dream_content=$(head -100 "$dream" | tr '\n' ' ')

            if [[ -n "$dream_content" ]]; then
                semantic-index "$dream_content" "dreams" "$dream_id" >/dev/null 2>&1 || true
                ((count++)) || true
            fi
        done
    fi

    # Index pheromones
    if [[ -f "$AETHER_DIR/data/pheromones.json" ]]; then
        echo "  Indexing pheromones..."
        local signals_json
        signals_json=$(jq -c '.signals[]' "$AETHER_DIR/data/pheromones.json" 2>/dev/null || echo "")

        while IFS= read -r signal; do
            [[ -z "$signal" ]] && continue
            local sig_id sig_type sig_content
            sig_id=$(echo "$signal" | jq -r '.id // empty')
            sig_type=$(echo "$signal" | jq -r '.type // empty')
            sig_content=$(echo "$signal" | jq -r '.content.text // .content // empty')

            if [[ -n "$sig_content" ]]; then
                semantic-index "[$sig_type] $sig_content" "pheromones" "$sig_id" >/dev/null 2>&1 || true
                ((count++)) || true
            fi
        done <<< "$signals_json"
    fi

    # Index QUEEN.md if exists
    if [[ -f "$AETHER_DIR/data/QUEEN.md" ]]; then
        echo "  Indexing QUEEN.md..."
        local queen_content
        queen_content=$(cat "$AETHER_DIR/data/QUEEN.md" | tr '\n' ' ')
        semantic-index "$queen_content" "queen" "queen-wisdom" >/dev/null 2>&1 || true
        ((count++)) || true
    fi

    echo "✅ Indexed $count entries"
    jq -c '.stats' "$SEMANTIC_DATA_DIR/index.json"
}

# Get context relevant to a task (for worker injection)
semantic-get-context() {
    local task="${1:-}"
    local max_results="${2:-3}"

    if [[ -z "$task" ]]; then
        echo ""
        return 0
    fi

    if ! semantic-check-deps >/dev/null 2>&1; then
        echo ""
        return 0
    fi

    local result
    result=$(semantic-search "$task" "$max_results" 0.5 2>/dev/null || echo '{"results":[]}')

    local count
    count=$(echo "$result" | jq -r '.count // 0')

    if [[ "$count" -eq 0 ]]; then
        echo ""
        return 0
    fi

    echo "---"
    echo "## Relevant Context (semantic search)"
    echo ""
    echo "$result" | jq -r '.results[] | "### \(.source // "unknown") (similarity: \(.similarity))\n\(.text[:200] // .text)\n"'
    echo "---"
}

# Check semantic layer status
semantic-status() {
    if [[ ! -f "$SEMANTIC_DATA_DIR/index.json" ]]; then
        semantic_json_ok '{"initialized": false, "message": "Run semantic-init to initialize"}'
        return 0
    fi

    local deps_ok
    if semantic-check-deps >/dev/null 2>&1; then
        deps_ok="true"
    else
        deps_ok="false"
    fi

    local entry_count
    entry_count=$(jq '.entries | length' "$SEMANTIC_DATA_DIR/index.json" 2>/dev/null || echo "0")

    jq -n --arg deps_ok "$deps_ok" --arg entries "$entry_count" \
        '{"initialized": true, "dependencies_ok": ($deps_ok == "true"), "total_entries": ($entries | tonumber)}'
}

# Helper: Output JSON OK response
semantic_json_ok() {
    local result="${1:-}"
    local message="${2:-}"
    jq -n --argjson result "$result" --arg message "$message" \
        '{"ok": true, "result": $result, "message": $message}'
}

# Helper: Output JSON error response
semantic_json_err() {
    local code="${1:-1}"
    local message="${2:-Unknown error}"
    jq -n --arg code "$code" --arg message "$message" \
        '{"ok": false, "error": {"code": $code, "message": $message}}'
}

# Export functions
export -f semantic-init
export -f semantic-index
export -f semantic-search
export -f semantic-find-duplicate
export -f semantic-rebuild
export -f semantic-get-context
export -f semantic-status
export -f semantic-check-deps
export -f semantic_json_ok
export -f semantic_json_err
