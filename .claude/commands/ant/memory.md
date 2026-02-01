---
name: ant:memory
description: Colony memory operations (search, status, verify, compress)
---

<objective>
Provide Queen with comprehensive memory operations:
- Search across all three memory layers with relevance ranking
- View memory statistics and usage
- Verify 200k token limit enforcement
- Trigger manual compression of Working Memory
</objective>

<process>
You are the **Queen Ant Colony** managing colony memory.

## Step 1: Parse Subcommand

The user provides a subcommand and optional arguments:
```bash
subcommand="$1"  # search, status, verify, compress
arg1="$2"        # query for search, or empty
arg2="$3"        # optional limit for search
```

## Step 2: Execute Subcommand

### Subcommand: search
Search all three memory layers for query:
```bash
# Source memory search utilities
source .aether/utils/memory-search.sh

# Get query argument
query="$1"
limit="${2:-20}"

# Validate query provided
if [ -z "$query" ]; then
  echo "Usage: /ant:memory search \"<query>\" [limit]"
  exit 1
fi

# Search across all layers
search_memory "$query" "$limit"
```

### Subcommand: status
Show memory statistics:
```bash
# Source memory search utilities
source .aether/utils/memory-search.sh

# Get memory status
get_memory_status
```

### Subcommand: verify
Verify 200k token limit enforcement:
```bash
# Source memory search utilities
source .aether/utils/memory-search.sh

# Verify token limit
verify_token_limit
exit_code=$?

echo ""
echo "Max capacity tokens: 200000"
echo "Compression triggers at 80% (160000 tokens) to prevent overflow"
exit $exit_code
```

### Subcommand: compress
Trigger manual compression of Working Memory:
```bash
# Source memory compression utilities
source .aether/utils/memory-compress.sh

# Get current phase from COLONY_STATE.json
current_phase=$(jq -r '.colony_status.current_phase // "1"' .aether/data/COLONY_STATE.json)

# Prepare compression data
temp_file=$(prepare_compression_data "$current_phase")

if [ $? -eq 0 ]; then
  echo "Working Memory data prepared for compression: $temp_file"
  echo ""
  echo "NOTE: Manual compression requires Architect Ant (LLM) to apply DAST compression."
  echo "The prepared file contains Working Memory items ready for compression."
  echo ""
  echo "To complete compression:"
  echo "1. Architect Ant reads: $temp_file"
  echo "2. Architect Ant applies DAST compression (2.5x ratio)"
  echo "3. Architect Ant outputs compressed JSON"
  echo "4. Run: trigger_phase_boundary_compression $current_phase '<compressed_json>'"
else
  echo "Working Memory is empty or below compression threshold. No compression needed."
fi
```

## Step 3: Present Results

Format output based on subcommand:

**For search:**
```json
[
  {
    "id": "wm_...",
    "content": "matching content",
    "layer": "working_memory",
    "relevance": 0.7,
    ...
  },
  ...
]
```

**For status:**
```
MEMORY STATUS

Working Memory:
  Items: {count}
  Tokens: {current} / {max} (200,000) ({percent}%)
  Eviction Threshold: {threshold} tokens (80%)
  Max Capacity: 200,000 tokens (hard limit)

Short-term Memory:
  Sessions: {current} / {max} ({percent}%)
  Compression Ratio: 2.5x target

Long-term Memory:
  Patterns: {count}
  Types: success={n}, failure={n}, preference={n}, constraint={n}

Metrics:
  Total Compressions: {n}
  Average Compression Ratio: {ratio}
  Working Memory Evictions: {n}
  Short-term Evictions: {n}
  Pattern Extractions: {n}
```

**For verify:**
```
TOKEN LIMIT VERIFICATION
Max Capacity: 200000 tokens
Current Usage: {current} tokens ({percent}%)
Compression Threshold: 160000 tokens (80%)
Status: PASS - Current usage within safe limits
```

**For compress:**
```
Working Memory data prepared for compression: /tmp/working_memory_for_compression_{phase}.json

NOTE: Manual compression requires Architect Ant (LLM) to apply DAST compression.
The prepared file contains Working Memory items ready for compression.

To complete compression:
1. Architect Ant reads: /tmp/working_memory_for_compression_{phase}.json
2. Architect Ant applies DAST compression (2.5x ratio)
3. Architect Ant outputs compressed JSON
4. Run: trigger_phase_boundary_compression {phase} '<compressed_json>'
```

## Step 4: Handle Errors

**Invalid subcommand:**
```bash
echo "Error: Unknown subcommand '$subcommand'"
echo ""
echo "Available subcommands:"
echo "  search \"<query>\" [limit]  - Search all memory layers"
echo "  status                      - Show memory statistics"
echo "  verify                      - Verify 200k token limit"
echo "  compress                    - Trigger manual compression"
exit 1
```

**Missing arguments:**
```bash
case "$subcommand" in
  search)
    if [ -z "$arg1" ]; then
      echo "Error: search requires a query"
      echo "Usage: /ant:memory search \"<query>\" [limit]"
      exit 1
    fi
    ;;
esac
```

</process>

<context>
# AETHER TRIPLE-LAYER MEMORY

## Memory Architecture

The colony uses three memory layers:

1. **Working Memory** - Immediate context, 200k token capacity
2. **Short-term Memory** - Compressed sessions, max 10 sessions
3. **Long-term Memory** - Persistent patterns, associative links

## 200k Token Limit

Working Memory has a **hard limit of 200,000 tokens**:
- Max capacity: 200,000 tokens
- Compression threshold: 160,000 tokens (80%)
- When threshold is reached, compression is triggered automatically
- Compression prevents overflow by moving data to Short-term Memory

This is enforced by the compression system:
- `auto_compress_if_needed()` checks threshold before adding items
- Compression at 80% ensures we never exceed 200k tokens

## Search Ranking

Search results are ranked by:
1. **Layer priority**: Working Memory (0) > Short-term (1) > Long-term (2)
2. **Relevance score**: exact match = 1.0, contains match = 0.7
3. **Recency**: more recent items appear first

## Compression

Compression uses **DAST (Discriminative Abstractive Summarization Technique)**:
- Preserves: key_decisions, outcomes, learned_patterns, blockers_encountered, solutions_found
- Discards: intermediate_steps, failed_attempts, redundant_context
- Target ratio: 2.5x compression

Compression triggers:
1. **Phase boundary**: When phase completes
2. **Token threshold**: When Working Memory exceeds 80% (160k tokens)
3. **Manual**: Via `/ant:memory compress`

</context>

<reference>
# Memory Search Functions

The following functions are available in `.aether/utils/memory-search.sh`:

## Cross-layer Search
- `search_memory(query, [limit])` - Search all three layers, combine and rank results
- `search_working_memory(query, [limit])` - Search Working Memory, update access metadata
- `search_short_term_memory(query, [limit])` - Search Short-term Memory sessions
- `search_long_term_memory(query, [limit])` - Search Long-term Memory patterns

## Status and Verification
- `get_memory_status()` - Display formatted memory statistics
- `verify_token_limit()` - Verify 200k token limit enforcement

## Memory Compression

The following functions are available in `.aether/utils/memory-compress.sh`:

- `prepare_compression_data(phase)` - Prepare Working Memory for compression
- `trigger_phase_boundary_compression(phase, compressed_json)` - Process compressed result
- `create_short_term_session(phase, compressed_json)` - Store compressed session
- `clear_working_memory()` - Clear Working Memory after compression

</reference>

<allowed-tools>
Bash
Write
Read
</allowed-tools>
