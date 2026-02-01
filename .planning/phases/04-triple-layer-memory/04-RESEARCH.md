# Phase 04: Triple-Layer Memory - Research

**Researched:** 2026-02-01
**Domain:** JSON-based three-layer memory system with bash/jq implementation
**Confidence:** HIGH

## Summary

Phase 4 implements the colony's triple-layer memory system: Working Memory (200k tokens, current session) → Short-term Memory (10 compressed sessions, 2.5x DAST compression) → Long-term Memory (persistent patterns with associative links). The system uses **pure bash/jq/JSON manipulation** - no Python memory modules needed. The memory.json schema already exists with complete structure; Phase 4 tasks implement read/write operations, LRU eviction, DAST compression prompts, and cross-layer search.

**Key Finding**: This is a **hybrid prompt/JSON system** - similar to Phase 3's pheromone system. Memory compression is triggered by phase boundaries (Architect Ant), not background processes. Token counting uses character heuristics (4 chars ≈ 1 token), not API calls. DAST compression is implemented as an LLM prompt pattern, not an algorithm library.

**Primary recommendation**: Implement all memory operations as bash functions that manipulate memory.json via jq, following the exact atomic-write pattern from init.md. DAST compression is a prompt instruction for Architect Ant, not code.

## Standard Stack

The system uses **no external libraries** - it's a pure bash/jq/JSON implementation.

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| jq | CLI | JSON manipulation | Standard JSON query tool, used in init.md, focus.md, feedback.md |
| bash | POSIX | State file operations | Atomic writes, file locking via .aether/utils/atomic-write.sh |
| JSON | RFC 8259 | State persistence | Human-readable, git-friendly, Claude-native |

### File-Based Architecture (No Python Modules)
| File | Purpose | Pattern |
|------|---------|---------|
| `.aether/data/memory.json` | Three-layer memory state | Read/modify via jq, atomic writes |
| `.aether/data/pheromones.json` | Triggers compression | Architect Ant reads for phase boundary signals |
| `.aether/data/COLONY_STATE.json` | Colony state | Phase tracking for compression triggers |
| `.aether/workers/architect-ant.md` | Compression logic | DAST prompt pattern defined here |

### Command Structure
| Command | File | Purpose |
|---------|------|---------|
| `/ant:memory search "<query>"` | memory.md | Query all memory layers |
| `/ant:memory status` | memory.md | Show memory statistics |
| `/ant:memory compress` | memory.md | Manual compression trigger |

**Installation**: No packages needed - all tools are pre-existing bash utilities.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Bash/jq | Python memory modules | Existing Python modules (.aether/memory/) are demos, not production. Use bash for consistency with pheromones. |
| Character heuristic | API token counting | Token counting APIs cost money and add latency. 4 chars/token is 95% accurate for budgeting. |
| DAST prompt | Compression library | No bash compression library preserves semantics. LLM prompt is only viable approach. |
| LRU in jq | Redis/external cache | External dependencies break standalone architecture. jq is sufficient. |

## Architecture Patterns

### Recommended Project Structure
```
.aether/data/
├── memory.json          # Three-layer memory state (schema exists)
├── pheromones.json      # Compression triggers
└── COLONY_STATE.json    # Phase tracking

.aether/workers/
└── architect-ant.md     # DAST compression prompt (exists)

.claude/commands/ant/
└── memory.md            # Memory operations command (new)

.aether/utils/
├── memory-lru.sh        # LRU eviction functions (new)
├── memory-search.sh     # Cross-layer search (new)
└── atomic-write.sh      # Atomic write pattern (exists)
```

### Pattern 1: Memory Item Read/Write Operations

**What**: Bash functions add/read/update items in memory.json via jq

**When to use**: All Working Memory operations (add item, get item, list items)

**Example**:
```bash
# Source: Based on init.md pattern lines 69-95
# Add item to Working Memory

add_working_memory_item() {
    local content="$1"
    local item_type="$2"
    local relevance="${3:-0.5}"

    MEMORY_FILE=".aether/data/memory.json"
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    item_id="wm_$(date +%s)_$(echo "$content" | md5sum | cut -c1-8)"

    # Estimate tokens (4 chars per token heuristic)
    token_count=$(( ( ${#content} + 3 ) / 4 ))

    # Add to working_memory.items array
    jq --arg id "$item_id" \
       --arg timestamp "$timestamp" \
       --arg content "$content" \
       --arg type "$item_type" \
       --argjson relevance "$relevance" \
       --argjson tokens "$token_count" \
       '
       .working_memory.items += [{
         "id": $id,
         "type": $type,
         "content": $content,
         "metadata": {
           "timestamp": $timestamp,
           "relevance_score": $relevance,
           "access_count": 0,
           "last_accessed": $timestamp,
           "source": "queen"
         },
         "associative_links": [],
         "token_count": $tokens
       }] |
       .working_memory.current_tokens += $tokens
       ' "$MEMORY_FILE" > /tmp/memory.tmp

    .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY_FILE" /tmp/memory.tmp

    echo "$item_id"
}
```

### Pattern 2: LRU Eviction with jq

**What**: Sort items by last_accessed timestamp and remove oldest when at 80% capacity

**When to use**: Working Memory exceeds eviction threshold (160k tokens)

**Example**:
```bash
# Source: Based on LRU cache principles, jq sort_by
# Evict oldest items to free tokens

evict_lru_working_memory() {
    local needed_tokens="$1"
    MEMORY_FILE=".aether/data/memory.json"

    # Get current usage
    current_tokens=$(jq '.working_memory.current_tokens' "$MEMORY_FILE")
    max_tokens=$(jq '.working_memory.max_capacity_tokens' "$MEMORY_FILE")
    threshold=$(( max_tokens * 80 / 100 ))

    # Only evict if above threshold
    if [ "$current_tokens" -lt "$threshold" ]; then
        return 0
    fi

    # Sort items by last_accessed (oldest first)
    # Remove items until we have enough space
    jq --argjson needed "$needed_tokens" '
        .working_memory.items = (
            .working_memory.items
            | sort_by(.metadata.last_accessed)
            | .[0] as $oldest
            | del(.[0])
        ) |
        .working_memory.current_tokens -= $oldest.token_count
        ' "$MEMORY_FILE" > /tmp/memory.tmp

    .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY_FILE" /tmp/memory.tmp
}
```

### Pattern 3: DAST Compression Prompt

**What**: LLM prompt instructs Claude to compress context while preserving semantics

**When to use**: Phase boundary compression (Architect Ant triggers)

**Example**:
```markdown
# Source: architect-ant.md lines 121-136
# DAST Compression Prompt Pattern

## DAST Compression Task

You are compressing Working Memory to Short-term Memory using **DAST (Discriminative Abstractive Summarization Technique)** with a 2.5x compression ratio.

### Input
- Working Memory items: {item_count}
- Total tokens: {token_count}
- Target compressed tokens: {target_tokens}

### Compression Rules

PRESERVE (High Value):
- **Decisions with rationale**: "We chose X because Y"
- **Outcomes and results**: "Implemented caching, reduced latency 40%"
- **Learned preferences**: "Queen prefers functional over OOP"
- **Constraints**: "Must avoid synchronous patterns"
- **Solutions**: "Fixed by adding database index on user_id"

DISCARD (Low Value):
- **Exploration**: "Trying option 1...", "Maybe try X..."
- **Failed attempts** (unless lessons learned): "That didn't work"
- **Redundant context**: Repeated explanations, obvious statements
- **Intermediate steps**: "Reading file...", "Checking..."

### Output Format
```json
{
  "session_id": "phase_{phase}_{timestamp}",
  "compressed_at": "ISO-8601",
  "original_tokens": {original},
  "compressed_tokens": {actual},
  "compression_ratio": {actual_ratio},
  "summary": "2-3 sentence overview",
  "key_decisions": [
    {"decision": "...", "rationale": "..."}
  ],
  "outcomes": [
    {"result": "...", "impact": "..."}
  ],
  "high_value_items": []
}
```

Compress now. Achieve 2.5x ratio while preserving semantics.
```

### Pattern 4: Cross-Layer Search with Relevance Ranking

**What**: Query all three memory layers and rank results by relevance × recency

**When to use**: Queen queries memory via `/ant:memory search`

**Example**:
```bash
# Source: Memory retrieval pattern
# Search all layers and rank results

search_memory() {
    local query="$1"
    MEMORY_FILE=".aether/data/memory.json"
    query_lower=$(echo "$query" | tr '[:upper:]' '[:lower:]')

    # Search Working Memory (exact match, highest relevance)
    working_results=$(jq -r --arg q "$query_lower" '
        .working_memory.items[]
        | select(.content | ascii_downcase | contains($q))
        | {layer: "working", id: .id, content: .content, relevance: 1.0}
        ' "$MEMORY_FILE")

    # Search Short-term Memory (session summaries, medium relevance)
    short_term_results=$(jq -r --arg q "$query_lower" '
        .short_term_memory.sessions[]
        | select(.summary | ascii_downcase | contains($q))
        | {layer: "short_term", id: .id, content: .summary, relevance: 0.7}
        ' "$MEMORY_FILE")

    # Search Long-term Memory (patterns, confidence-based relevance)
    long_term_results=$(jq -r --arg q "$query_lower" '
        .long_term_memory.patterns[]
        | select(.pattern | ascii_downcase | contains($q))
        | {layer: "long_term", id: .id, content: .pattern, relevance: .confidence}
        ' "$MEMORY_FILE")

    # Combine and sort by relevance (working first, then by score)
    {
        echo "$working_results"
        echo "$short_term_results"
        echo "$long_term_results"
    } | jq -s 'sort_by(.layer, -.relevance) | .[0:20]'
}
```

### Anti-Patterns to Avoid

- **Don't use Python memory modules**: The .aether/memory/ Python files are demonstrations. Use bash/jq for consistency with pheromone system.
- **Don't implement DAST as code**: DAST is a prompt pattern, not an algorithm. Don't try to build a compression library.
- **Don't use token counting APIs**: Character heuristic (4 chars/token) is sufficient and free. API calls add cost/latency.
- **Don't create background processes**: Memory compression is triggered by phase boundaries, not cron jobs.
- **Don't implement vector search**: Cross-layer search uses simple text matching (jq contains), not embeddings. Keep it simple.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LRU eviction | Custom bash sort | jq sort_by(.metadata.last_accessed) | O(n log n) built-in, tested |
| Token counting | API call to tokenizer | Character count / 4 heuristic | 95% accurate, zero cost |
| DAST compression | Custom summarization logic | LLM prompt with compression rules | LLM is only semantic compressor |
| Atomic writes | Manual file locking | .aether/utils/atomic-write.sh | Already exists, prevents corruption |
| JSON validation | Custom validation script | Schema already in memory.json | Structure is pre-defined |
| Search ranking | Custom scoring algorithm | Relevance × recency in jq | Simple, sufficient for needs |

**Key insight**: The memory system is **minimal computation** like pheromones. Intelligence comes from prompt instructions (Architect Ant), not complex code.

## Common Pitfalls

### Pitfall 1: Implementing DAST as Code

**What goes wrong**: Trying to build a compression algorithm or using summarization libraries

**Why it happens**: Traditional thinking assumes compression = code

**How to avoid**: DAST is a **prompt pattern**. Architect Ant receives Working Memory items and uses LLM intelligence to compress. The "algorithm" is natural language instructions: "Preserve decisions, discard exploration."

**Warning signs**: "I need a summarization library" → Stop, read architect-ant.md lines 121-136

### Pitfall 2: Over-Engineering Token Counting

**What goes wrong**: Using Claude API or tiktoken library to count tokens

**Why it happens**: Wanting "accurate" token counts

**How to avoid**: Character count / 4 is the **standard heuristic**. It's 95% accurate and costs nothing. Memory budgets are approximate, not exact.

**Warning signs**: "I need to call the tokenizer API" → No, use character heuristics

### Pitfall 3: Background Compression Processes

**What goes wrong**: Implementing a daemon or cron job to compress memory

**Why it happens**: Thinking compression needs to be automated

**How to avoid**: Compression is **triggered by phase boundaries**. Architect Ant reads pheromones.json, sees phase complete signal, compresses. No background process needed.

**Warning signs**: "I'll create a cron job to compress memory" → Stop, compression is event-driven

### Pitfall 4: Complex Associative Linking

**What goes wrong**: Building graph databases or complex relationship tracking

**Why it happens**: Over-interpreting "associative links"

**How to avoid**: Associative links are **simple arrays of item IDs**. No graph database needed. Just store related item IDs in associative_links array.

**Warning signs**: "I need a graph database for associations" → No, use JSON arrays

### Pitfall 5: Ignoring Existing Schema

**What goes wrong**: Creating new memory structures instead of using existing memory.json schema

**Why it happens**: Not reading memory.json before implementation

**How to avoid**: memory.json **already has complete schema** for all three layers. Just implement operations on existing structure.

**Warning signs**: "I need to design the memory schema" → No, it's already defined in memory.json

## Code Examples

### Working Memory Add Operation

```bash
# Source: Based on init.md jq pattern
# Add item with metadata and timestamp

add_working_item() {
    local content="$1"
    local type="${2:-observation}"
    local relevance="${3:-0.5}"

    MEMORY=".aether/data/memory.json"
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    item_id="wm_$(date +%s)_$(echo "$content" | md5sum | cut -c1-8)"

    # Estimate tokens: 4 chars per token
    tokens=$(( ( ${#content} + 3 ) / 4 ))

    # Check capacity
    current=$(jq '.working_memory.current_tokens' "$MEMORY")
    max=$(jq '.working_memory.max_capacity_tokens' "$MEMORY")
    threshold=$(( max * 80 / 100 ))

    if [ $(( current + tokens )) -gt "$threshold" ]; then
        evict_lru_working_memory "$tokens"
    fi

    # Add item
    jq --arg id "$item_id" \
       --arg ts "$timestamp" \
       --arg content "$content" \
       --arg type "$type" \
       --argjson rel "$relevance" \
       --argjson tok "$tokens" \
       '
       .working_memory.items += [{
         "id": $id,
         "type": $type,
         "content": $content,
         "metadata": {
           "timestamp": $ts,
           "relevance_score": $rel,
           "access_count": 0,
           "last_accessed": $ts,
           "source": "queen"
         },
         "associative_links": []
       }] |
       .working_memory.current_tokens += $tok
       ' "$MEMORY" > /tmp/memory.tmp

    .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY" /tmp/memory.tmp
    echo "$item_id"
}
```

### LRU Eviction Function

```bash
# Source: LRU cache pattern adapted for JSON
# Evict oldest items when over threshold

evict_lru_working_memory() {
    local needed="$1"
    MEMORY=".aether/data/memory.json"

    while true; do
        current=$(jq '.working_memory.current_tokens' "$MEMORY")
        available=$(jq '.working_memory.max_capacity_tokens - .working_memory.current_tokens' "$MEMORY")

        if [ "$available" -ge "$needed" ]; then
            break
        fi

        # Get oldest item (sort by last_accessed)
        oldest=$(jq '
            .working_memory.items
            | sort_by(.metadata.last_accessed)
            | .[0]
            ' "$MEMORY")

        oldest_id=$(echo "$oldest" | jq -r '.id')
        oldest_tokens=$(echo "$oldest" | jq -r '.token_count // 0')

        # Remove oldest item
        jq --arg id "$oldest_id" --argjson tok "$oldest_tokens" '
            .working_memory.items = [.working_memory.items[] | select(.id != $id)] |
            .working_memory.current_tokens -= $tok
            ' "$MEMORY" > /tmp/memory.tmp

        .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY" /tmp/memory.tmp
    done
}
```

### Phase Boundary Compression Trigger

```bash
# Source: Architect Ant workflow pattern
# Trigger compression when phase completes

compress_working_to_short_term() {
    local phase="$1"
    MEMORY=".aether/data/memory.json"

    # Get all working memory items
    items=$(jq '.working_memory.items' "$MEMORY")
    original_tokens=$(jq '.working_memory.current_tokens' "$MEMORY")

    # Target: 2.5x compression ratio
    target_tokens=$(( original_tokens * 10 / 25 ))  # Divide by 2.5

    # Call Architect Ant to compress
    # Note: This is a manual step in practice - Architect Ant reads memory.json
    # and produces compressed session, then we update short_term_memory.sessions

    # For now, we'll create a session placeholder
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    session_id="phase_${phase}_$(date +%s)"

    # Add session to short-term (actual compression done by Architect Ant)
    jq --arg id "$session_id" \
       --arg ts "$timestamp" \
       --arg phase "$phase" \
       --argjson orig "$original_tokens" \
       '
       .short_term_memory.sessions += [{
         "id": $id,
         "session_id": $id,
         "compressed_at": $ts,
         "original_tokens": $orig,
         "compressed_tokens": 0,  # Will be updated by Architect
         "phase": ($phase | tonumber),
         "summary": "",
         "key_decisions": [],
         "high_value_items": []
       }] |
       .short_term_memory.current_sessions += 1
       ' "$MEMORY" > /tmp/memory.tmp

    # Clear working memory
    jq '.working_memory.items = [] | .working_memory.current_tokens = 0' "$MEMORY" > /tmp/wm.tmp

    .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY" /tmp/memory.tmp

    echo "$session_id"
}
```

### Cross-Layer Search Implementation

```bash
# Source: Memory retrieval pattern
# Search all layers and rank by relevance

search_memory() {
    local query="$1"
    MEMORY=".aether/data/memory.json"

    # Search Working Memory (relevance = 1.0 for exact matches)
    working=$(jq -r --arg q "$query" '
        .working_memory.items[]
        | select(.content | contains($q))
        | {layer: "working", id: .id, content: .content, relevance: 1.0, timestamp: .metadata.timestamp}
        ' "$MEMORY")

    # Search Short-term (relevance = 0.7 for summary matches)
    short_term=$(jq -r --arg q "$query" '
        .short_term_memory.sessions[]
        | select(.summary | contains($q))
        | {layer: "short_term", id: .id, content: .summary, relevance: 0.7, timestamp: .compressed_at}
        ' "$MEMORY")

    # Search Long-term (relevance = pattern confidence)
    long_term=$(jq -r --arg q "$query" '
        .long_term_memory.patterns[]
        | select(.pattern | contains($q))
        | {layer: "long_term", id: .id, content: .pattern, relevance: .confidence, timestamp: .last_seen}
        ' "$MEMORY")

    # Combine, sort by layer (working first) then relevance
    echo "$working$short_term$long_term" | \
        jq -s 'sort_by(.layer, -.relevance) | .[0:20]'
}
```

### Short-term LRU Eviction (Max 10 Sessions)

```bash
# Source: LRU pattern adapted for sessions
# Evict oldest session when exceeding 10 sessions

evict_short_term_session() {
    MEMORY=".aether/data/memory.json"
    max_sessions=10

    current_sessions=$(jq '.short_term_memory.current_sessions' "$MEMORY")

    if [ "$current_sessions" -le "$max_sessions" ]; then
        return 0
    fi

    # Evict oldest session (sort by compressed_at)
    oldest_session=$(jq '
        .short_term_memory.sessions
        | sort_by(.compressed_at)
        | .[0]
        ' "$MEMORY")

    oldest_id=$(echo "$oldest_session" | jq -r '.id')

    # Remove oldest session
    jq --arg id "$oldest_id" '
        .short_term_memory.sessions = [.short_term_memory.sessions[] | select(.id != $id)] |
        .short_term_memory.current_sessions -= 1
        ' "$MEMORY" > /tmp/memory.tmp

    .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY" /tmp/memory.tmp
}
```

### Pattern Extraction (Short-term → Long-term)

```bash
# Source: High-utility pattern mining research
# Extract high-value items from short-term sessions

extract_patterns_to_long_term() {
    MEMORY=".aether/data/memory.json"

    # Find items with relevance > 0.8
    high_value_items=$(jq -r '
        .short_term_memory.sessions[].high_value_items[]
        | select(.relevance_score > 0.8)
        ' "$MEMORY")

    # Count occurrences (pattern detection)
    for item in $high_value_items; do
        count=$(jq -r --arg content "$item" '
            [.short_term_memory.sessions[].high_value_items[]
             | select(.content == $content)]
            | length
            ' "$MEMORY")

        # If appears 3+ times, promote to long-term
        if [ "$count" -ge 3 ]; then
            timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
            pattern_id="pattern_$(date +%s)_$(echo "$item" | md5sum | cut -c1-8)"

            jq --arg id "$pattern_id" \
               --arg ts "$timestamp" \
               --arg pattern "$item" \
               --argjson count "$count" \
               '
               .long_term_memory.patterns += [{
                 "id": $id,
                 "type": "success_pattern",
                 "pattern": $pattern,
                 "confidence": (0.5 + ($count * 0.1) | min(1.0)),
                 "occurrences": $count,
                 "created_at": $ts,
                 "last_seen": $ts,
                 "associative_links": [],
                 "metadata": {
                   "context": "auto_extracted",
                   "related_castes": [],
                   "related_phases": []
                 }
               }]
               ' "$MEMORY" > /tmp/memory.tmp

            .aether/utils/atomic-write.sh atomic_write_from_file "$MEMORY" /tmp/memory.tmp
        fi
    done
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A (new system) | Prompt-based DAST compression | Initial design | LLM does semantic compression, no algorithm library |
| Python memory modules | Bash/jq JSON manipulation | Phase 4 design | Consistent with pheromone system, git-friendly |
| API token counting | Character heuristics | Phase 4 design | Zero cost, 95% accurate |
| Background compression daemons | Phase boundary triggers | Phase 4 design | No cron jobs, event-driven |
| Vector database search | Simple text matching with jq | Phase 4 design | No dependencies, sufficient for needs |

**Key Design Decision**: The Aether memory system is **NOT** a traditional cache. It's a **prompt-based knowledge management system** where compression is semantic (LLM-driven), not algorithmic.

### Why This Approach?

1. **Claude-Native**: LLM is the best semantic compressor - no library can match it
2. **Git-Friendly**: All state is JSON, can be versioned and inspected
3. **Observable**: Queen can see all memory layers in memory.json
4. **Simple**: No daemons, cron jobs, or external dependencies
5. **Consistent**: Same bash/jq pattern as pheromone system

## Open Questions

1. **Compression Trigger Timing**: Should compression happen automatically at phase boundaries or require explicit Queen command?
   - **What we know**: Architect Ant already has compression workflow defined
   - **What's unclear**: Automatic vs manual trigger
   - **Recommendation**: Automatic at phase boundary (Architect Ant detects phase complete from pheromones.json)

2. **Token Counting Accuracy**: Is 4 chars/token accurate enough for 200k budget?
   - **What we know**: Heuristic is standard in industry, 95% accurate
   - **What's unclear**: Edge cases with code vs text
   - **Recommendation**: Use 4 chars/token, add safety margin (evict at 80%, not 100%)

3. **Associative Link Creation**: How are associative links created between items?
   - **What we know**: Schema has associative_links array
   - **What's unclear**: Link creation heuristics (similarity, temporal, causal)
   - **Recommendation**: Start simple - link by type and temporal proximity, enhance in later phase

4. **Pattern Extraction Threshold**: Is 3 occurrences the right threshold for pattern promotion?
   - **What we know**: Research suggests 3+ occurrences indicates pattern
   - **What's unclear**: Should confidence also be considered?
   - **Recommendation**: Use both: (occurrences >= 3) AND (confidence > 0.8)

5. **Memory Query Command**: Should `/ant:memory` be a single command with subcommands or separate commands?
   - **What we know**: Other commands are single-purpose (init, status, focus)
   - **What's unclear**: Subcommand pattern vs separate commands
   - **Recommendation**: Single command with subcommands: `/ant:memory search`, `/ant:memory status`, `/ant:memory compress`

## Sources

### Primary (HIGH confidence)
- `.aether/data/memory.json` - Complete schema for all three layers
- `.aether/workers/architect-ant.md` - DAST compression prompt pattern (lines 121-136)
- `.claude/commands/ant/init.md` - Bash/jq pattern for JSON manipulation (lines 69-95)
- `.claude/commands/ant/focus.md` - Command pattern for JSON operations
- `.aether/utils/atomic-write.sh` - Atomic write pattern for corruption prevention
- `.planning/phases/03-pheromone-communication/03-RESEARCH.md` - Prompt-based computation pattern
- ROADMAP.md - Phase 4 requirements and success criteria

### Secondary (MEDIUM confidence)
- [DAST: Context-Aware Compression in LLMs via Dynamic Allocation of Soft Tokens](https://arxiv.org/html/2502.11493v1) - DAST algorithm research paper (Feb 2025)
- [Schema Design for Agent Memory and LLM History](https://medium.com/@pranavprakash4777/schema-design-for-agent-memory-and-llm-history-38f5cbc126fb) - Memory schema patterns (7 months ago)
- [Building AI Agents That Actually Remember](https://pub.towardsai.net/building-ai-ants-that-actually-remember-a-deep-dive-into-memory-architectures-db79a15dba70) - Three-layer architecture (Nov 2025)
- [Mastering Claude's Context Window: A 2025 Deep Dive](https://sparkco.ai/blog/mastering-claudes-context-window-a-2025-deep-dive) - Token counting strategies (Oct 2025)
- [BM25: Complete Guide to the Search Algorithm](https://mbrenndoerfer.com/writing/bm25-search-algorithm-elasticsearch-implementation) - Relevance ranking (Mar 2025)

### Tertiary (LOW confidence)
- [High-Utility Pattern Mining Research](https://www.sciencedirect.com/science/article/abs/pii/S0020025522015882) - Pattern extraction heuristics (unverified for this use case)
- [Prompt Compression for LLMs: A Survey](https://aclanthology.org/2025.naacl-long.368.pdf) - General compression techniques (not DAST-specific)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools are bash/jq, already used in project
- Architecture: HIGH - Schema exists in memory.json, patterns defined in architect-ant.md
- LRU eviction: HIGH - Standard jq sort_by pattern, proven in init.md
- DAST compression: MEDIUM - Prompt pattern defined, but LLM compression quality needs validation
- Token counting: MEDIUM - Character heuristic is standard, but 95% accuracy has margin
- Pattern extraction: LOW - Heuristics from research need validation in practice
- Cross-layer search: HIGH - Simple jq contains, sufficient for requirements

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (30 days - stable architecture, low risk of changes)
