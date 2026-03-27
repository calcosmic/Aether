# Phase 34: Add User Approval UX - Research

**Researched:** 2026-02-20
**Domain:** Interactive CLI UX / Bash User Input / Learning Promotion Workflow
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Approval Interaction:**
- Checkbox style — Display uses `[ ]` brackets for unchecked items
- Selection method — User types numbers to select (e.g., "1 3 5" to tick boxes)
- Post-selection flow — Summary count ("3 proposals selected") then promote — no per-item confirmation
- Zero selection — If no selections made, keep all proposals for next run (don't discard)
- Threshold override — Users CAN approve proposals below threshold; show warning but allow selection

**Proposal Display:**
- Grouping — Proposals grouped by wisdom type (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees)
- Detail level — Minimal: one line per proposal
  - Format: `[ ] 1. Philosophy: "Keep functions small" ●●●●● (5/5)`
- Checkbox visual — Bracket style `[ ]` for unchecked, `[x]` for selected
- Threshold indicator — Progress bar with filled circles (e.g., `●●●○○ 3/5`)
- Verbose flag — `--verbose` shows full content for proposals that need more context

**Batch vs Individual:**
- Approval mode — Batch approve all selected in one action
- Error handling — Stop on first error; show which succeeded before failure
- Success feedback — List each promoted item: `"✓ Promoted Pattern: Use colony-prime() for context"`
- Undo — Immediate undo prompt after promotion: `"Undo these promotions? (y/n)"`

**Rejected Proposals:**
- Default behavior — Defer to later, not discard
- Storage — learning-deferred.json (separate file, same format as observations)
- Auto-represent — Never auto-show deferred items in regular continue.md
- Manual review — `/ant:continue --deferred` shows deferred proposals with same approval UX

### Claude's Discretion
- Exact wording of summary/feedback messages
- Progress bar character choices (filled/unfilled)
- Error message format when promotion fails
- Whether to show "no proposals to review" message vs silent exit

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PHER-EVOL-03 | Tick-to-approve UX for proposed pheromones (user selects which to promote) | Checkbox-style UI pattern, bash read for input parsing, number selection to index mapping |
</phase_requirements>

## Summary

Phase 34 implements an interactive CLI approval flow for promoting observations to permanent wisdom in QUEEN.md. This builds on Phase 33's observation tracking (`learning-observe`, `learning-check-promotion`) and Phase 32's QUEEN.md structure.

**Primary recommendation:** Implement a bash-based interactive selection system using `read` for input, number-based selection mapping to array indices, and visual checkbox state with `[ ]`/`[x]` toggles. The system must handle threshold visualization, batch operations, deferred storage, and undo functionality.

The UX pattern mirrors Git's interactive staging (`git add -p`) which users already know. Progress bars use Unicode circles (●/○) for threshold visualization. All state changes are atomic with JSON file operations via the existing `aether-utils.sh` infrastructure.

## Standard Stack

### Core
| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| Bash `read` | Capture user input | Native, no dependencies, works in all shells |
| `learning-check-promotion` | Get proposals meeting thresholds | Already implemented in Phase 33 |
| `queen-promote` | Execute promotions | Already implemented in Phase 33 |
| `learning-observations.json` | Source data for proposals | Existing observation storage |
| `learning-deferred.json` | Store rejected proposals | New file, same schema as observations |

### Supporting
| Component | Purpose | When to Use |
|-----------|---------|-------------|
| `jq` | JSON manipulation | Parse proposals, update deferred file |
| `aether-utils.sh` | Utility functions | Lock acquisition, atomic writes, activity logging |
| Unicode circles (●/○) | Progress bars | Threshold visualization |
| Bracket notation `[ ]` | Checkbox state | Familiar from Git interactive mode |

### Data Schema

**learning-deferred.json** (new file):
```json
{
  "deferred": [
    {
      "content_hash": "sha256:...",
      "content": "Always validate inputs",
      "wisdom_type": "pattern",
      "observation_count": 2,
      "threshold": 3,
      "first_seen": "2026-02-20T19:50:43Z",
      "deferred_at": "2026-02-20T20:15:00Z",
      "colonies": ["colony-a", "colony-b"]
    }
  ]
}
```

**Thresholds** (from QUEEN.md METADATA):
| Type | Threshold | Visual Indicator |
|------|-----------|------------------|
| philosophy | 5 | ●●●●● or ●●●○○ |
| pattern | 3 | ●●● or ●●○ |
| redirect | 2 | ●● or ●○ |
| stack | 1 | ● |
| decree | 0 | (immediate) |

## Architecture Patterns

### Recommended Flow Structure

```
propose-promotions/
├── 1. Load proposals from learning-check-promotion
├── 2. Load deferred if --deferred flag
├── 3. Display grouped proposals with checkboxes
├── 4. Capture user selection (space-separated numbers)
├── 5. Show summary and confirm
├── 6. Execute queen-promote for each selected
├── 7. Move unselected to deferred
├── 8. Offer undo
└── 9. Log activity
```

### Pattern 1: Number-to-Index Selection Mapping

**What:** User inputs space-separated numbers (1 3 5), mapped to array indices (0 2 4).

**When to use:** Any multi-select CLI interface.

**Example:**
```bash
# Display items 1-indexed for users
# [ ] 1. Pattern: "Always validate inputs"
# [ ] 2. Philosophy: "Keep it simple"

# Parse selection
read -r selection
selected_indices=()
for num in $selection; do
    # Convert 1-indexed to 0-indexed
    idx=$((num - 1))
    selected_indices+=("$idx")
done
```

### Pattern 2: Visual Checkbox State

**What:** Toggle `[ ]` to `[x]` based on selection.

**When to use:** When showing selection state before confirmation.

**Example:**
```bash
# Initial display
# [ ] 1. Pattern: "Always validate inputs" ●●● (3/3)

# After user selects "1"
# [x] 1. Pattern: "Always validate inputs" ●●● (3/3)
```

### Pattern 3: Threshold Progress Bar

**What:** Unicode circles showing observation count vs threshold.

**When to use:** Visual indicator of "ready" vs "early" promotions.

**Example:**
```bash
generate_threshold_bar() {
    local count=$1
    local threshold=$2
    local bar=""

    for ((i=0; i<threshold; i++)); do
        if [[ $i -lt $count ]]; then
            bar+="●"  # Filled
        else
            bar+="○"  # Empty
        fi
    done

    echo "$bar ($count/$threshold)"
}

# Output: ●●●○○ (3/5) for pattern with 3 observations, threshold 5
```

### Pattern 4: Grouped Display

**What:** Group proposals by wisdom type with headers.

**When to use:** When items have categories and order doesn't matter.

**Example:**
```
📜 Philosophies (threshold: 5)
  [ ] 1. "Test-driven development ensures quality" ●●●●● (5/5)

🧭 Patterns (threshold: 3)
  [ ] 2. "Always validate inputs" ●●● (3/3)
  [ ] 3. "Use jq for JSON" ●●○ (2/3) ⚠️ Below threshold
```

### Pattern 5: Batch Operation with Partial Success

**What:** Execute multiple operations, track successes, stop on first error but report what succeeded.

**When to use:** When multiple independent promotions happen together.

**Example:**
```bash
promoted_count=0
failed_item=""

for idx in "${selected_indices[@]}"; do
    proposal=${proposals[$idx]}
    if queen-promote "$proposal"; then
        ((promoted_count++))
        echo "✓ Promoted: $proposal"
    else
        failed_item="$proposal"
        break
    fi
done

if [[ -n "$failed_item" ]]; then
    echo "⚠️ Stopped after $promoted_count promotions"
    echo "Failed on: $failed_item"
fi
```

### Pattern 6: Deferred Storage

**What:** Move unselected items to separate file for later review.

**When to use:** When user rejects items but they shouldn't be lost.

**Example:**
```bash
# For each unselected proposal, add to deferred array
deferred_proposals=()
for proposal in "${all_proposals[@]}"; do
    if [[ ! " ${selected_indices[*]} " =~ " ${proposal.index} " ]]; then
        deferred_proposals+=("$proposal")
    fi
done

# Write to learning-deferred.json
jq -n --argjson deferred "$deferred_proposals" '{deferred: $deferred}' \
    > "$DATA_DIR/learning-deferred.json"
```

### Pattern 7: Undo Prompt

**What:** Immediate undo option after destructive operation.

**When to use:** When user might regret batch action.

**Example:**
```bash
# After promotions complete
echo ""
echo "Undo these promotions? (y/n)"
read -r undo_response

if [[ "$undo_response" =~ ^[Yy]$ ]]; then
    # Remove promoted entries from QUEEN.md
    # This requires tracking what was added
    revert_promotions "$promoted_items"
fi
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON manipulation | Custom parsing | `jq` | Handles escaping, nesting, arrays correctly |
| File locking | flock manually | `aether-utils.sh acquire_lock` | Already integrated with error handling |
| Atomic writes | `echo > file` | `aether-utils.sh atomic_write` | Prevents corruption on crash |
| Input validation | Regex from scratch | Bash parameter expansion | `${var//[^0-9 ]/}` removes non-digits |
| Array contains check | Loop with break | Bash pattern matching | `[[ " $array " =~ " $item " ]]` |

**Key insight:** The existing `aether-utils.sh` already provides robust file operations, locking, and JSON handling. Use these primitives rather than reimplementing.

## Common Pitfalls

### Pitfall 1: Off-by-One in Selection
**What goes wrong:** User selects "1" but code uses index 1 instead of 0, selecting the wrong item.
**Why it happens:** Forgetting to convert between 1-indexed display and 0-indexed arrays.
**How to avoid:** Always convert: `idx=$((user_input - 1))` then validate `[[ $idx -ge 0 && $idx -lt ${#items[@]} ]]`.
**Warning signs:** Wrong items promoted, array index out of bounds errors.

### Pitfall 2: Empty Selection Handling
**What goes wrong:** Pressing Enter with no input causes script to hang or error.
**Why it happens:** Not checking for empty string before processing selection.
**How to avoid:** `[[ -z "$selection" ]] && { echo "No selection made. Proposals deferred."; exit 0; }`
**Warning signs:** Script hangs waiting for input, jq errors on empty arrays.

### Pitfall 3: Concurrent File Access
**What goes wrong:** Two colonies running simultaneously corrupt learning-deferred.json.
**Why it happens:** No lock acquisition during read-modify-write cycle.
**How to avoid:** Always use `acquire_lock "$DATA_DIR/learning-deferred.json"` before updating.
**Warning signs:** JSON parse errors, missing deferred items.

### Pitfall 4: Unicode Display Issues
**What goes wrong:** Progress bar circles (●/○) display as squares or question marks in some terminals.
**Why it happens:** Terminal doesn't support Unicode, or locale not set to UTF-8.
**How to avoid:** Check `[[ ${LANG:-} =~ UTF-8 ]]` and fall back to ASCII `[=---]` bars if needed.
**Warning signs:** Garbled output, user confusion about threshold status.

### Pitfall 5: Threshold Override Confusion
**What goes wrong:** User approves below-threshold item but doesn't realize it's "early" promotion.
**Why it happens:** Warning not visible enough or threshold indicator unclear.
**How to avoid:** Use explicit warning: `⚠️ Below threshold (2/5) — early promotion` in red/yellow.
**Warning signs:** User surprise at "immature" wisdom in QUEEN.md.

### Pitfall 6: Deferred File Growth
**What goes wrong:** learning-deferred.json grows indefinitely with old rejected proposals.
**Why it happens:** No cleanup mechanism for stale deferred items.
**How to avoid:** Add TTL to deferred entries, auto-expire after 30 days.
**Warning signs:** Large JSON file, slow parsing, cluttered --deferred view.

### Pitfall 7: Undo After Session Clear
**What goes wrong:** User clears context, then wants to undo promotions from previous session.
**Why it happens:** Undo state not persisted, only held in memory.
**How to avoid:** Write undo log to file with timestamp, allow `/ant:continue --undo` within time window.
**Warning signs:** User frustration, manual QUEEN.md editing to revert.

## Code Examples

### Example 1: Display Proposals with Checkboxes

```bash
#!/bin/bash
# Source: CONTEXT.md decisions + aether-utils.sh patterns

display_proposals() {
    local proposals_json="$1"
    local verbose="${2:-false}"

    echo "🧠 Promotion Proposals"
    echo "====================="
    echo ""

    # Group by wisdom type
    local types=("philosophy" "pattern" "redirect" "stack" "decree")
    local type_emojis=("📜" "🧭" "⚠️" "🔧" "🏛️")
    local type_names=("Philosophies" "Patterns" "Redirects" "Stack Wisdom" "Decrees")

    local idx=1
    for i in "${!types[@]}"; do
        local type="${types[$i]}"
        local proposals=$(echo "$proposals_json" | jq --arg t "$type" '[.[] | select(.wisdom_type == $t)]')

        [[ "$proposals" == "[]" ]] && continue

        echo "${type_emojis[$i]} ${type_names[$i]}"

        echo "$proposals" | jq -c '.[]' | while read -r proposal; do
            local content=$(echo "$proposal" | jq -r '.content')
            local count=$(echo "$proposal" | jq -r '.observation_count')
            local threshold=$(echo "$proposal" | jq -r '.threshold')

            # Truncate content if not verbose
            if [[ "$verbose" != "true" && ${#content} -gt 40 ]]; then
                content="${content:0:37}..."
            fi

            # Generate threshold bar
            local bar=""
            for ((j=0; j<threshold; j++)); do
                if [[ $j -lt $count ]]; then
                    bar+="●"
                else
                    bar+="○"
                fi
            done

            # Warning for below-threshold
            local warning=""
            [[ $count -lt $threshold ]] && warning=" ⚠️"

            printf "  [ ] %d. \"%s\" %s (%d/%d)%s\n" "$idx" "$content" "$bar" "$count" "$threshold" "$warning"
            ((idx++))
        done
        echo ""
    done

    echo "Enter numbers to select (e.g., '1 3 5'), or press Enter to defer all:"
}
```

### Example 2: Parse User Selection

```bash
#!/bin/bash
# Source: CONTEXT.md decisions

parse_selection() {
    local input="$1"
    local max_index="$2"
    local -n result_array=$3  # Nameref for output

    result_array=()

    # Empty input = defer all
    [[ -z "$input" ]] && return 1

    # Normalize: remove extra spaces, keep only digits and spaces
    input=$(echo "$input" | tr -s ' ' | tr -cd '0-9 ')

    # Parse each number
    local seen=""
    for num in $input; do
        # Validate range
        if [[ "$num" -lt 1 || "$num" -gt "$max_index" ]]; then
            echo "Invalid selection: $num (valid: 1-$max_index)" >&2
            continue
        fi

        # Deduplicate
        [[ "$seen" == *" $num "* ]] && continue
        seen+=" $num "

        # Convert to 0-indexed
        result_array+=("$((num - 1))")
    done

    return 0
}
```

### Example 3: Execute Batch Promotions

```bash
#!/bin/bash
# Source: CONTEXT.md decisions + aether-utils.sh queen-promote

execute_promotions() {
    local -n proposals=$1        # Array of proposal objects
    local -n indices=$2          # Selected indices
    local colony_name="$3"

    local promoted=()
    local failed=""

    echo ""
    echo "Promoting ${#indices[@]} observation(s)..."
    echo ""

    for idx in "${indices[@]}"; do
        local proposal="${proposals[$idx]}"
        local type=$(echo "$proposal" | jq -r '.wisdom_type')
        local content=$(echo "$proposal" | jq -r '.content')

        # Call queen-promote
        if result=$(bash .aether/aether-utils.sh queen-promote "$type" "$content" "$colony_name" 2>&1); then
            echo "✓ Promoted ${type^}: \"$content\""
            promoted+=("$proposal")
        else
            echo "✗ Failed to promote: $content"
            echo "  Error: $result"
            failed="$content"
            break
        fi
    done

    # Log activity
    if [[ ${#promoted[@]} -gt 0 ]]; then
        bash .aether/aether-utils.sh activity-log "PROMOTED" "Queen" "Promoted ${#promoted[@]} observation(s) to QUEEN.md"
    fi

    # Return failed item (if any) for caller to handle
    echo "$failed"
}
```

### Example 4: Store Deferred Proposals

```bash
#!/bin/bash
# Source: CONTEXT.md decisions

store_deferred() {
    local -n all_proposals=$1
    local -n selected_indices=$2
    local deferred_file="$DATA_DIR/learning-deferred.json"

    # Build array of unselected proposals
    local deferred_json="["
    local first=true

    for i in "${!all_proposals[@]}"; do
        # Skip if selected
        [[ " ${selected_indices[*]} " =~ " $i " ]] && continue

        local proposal="${all_proposals[$i]}"

        # Add deferred_at timestamp
        local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        proposal=$(echo "$proposal" | jq --arg ts "$ts" '. + {deferred_at: $ts}')

        # Add to JSON array
        [[ "$first" == "true" ]] || deferred_json+=","
        first=false
        deferred_json+="$proposal"
    done

    deferred_json+="]"

    # Acquire lock and write atomically
    acquire_lock "$deferred_file"

    # Merge with existing deferred (if any)
    if [[ -f "$deferred_file" ]]; then
        local merged=$(jq -s '.[0].deferred + .[1] | {deferred: .}' "$deferred_file" - <<< "$deferred_json")
        echo "$merged" > "$deferred_file.tmp"
    else
        echo "{\"deferred\": $deferred_json}" > "$deferred_file.tmp"
    fi

    mv "$deferred_file.tmp" "$deferred_file"
    release_lock
}
```

### Example 5: Undo Prompt Implementation

```bash
#!/bin/bash
# Source: CONTEXT.md decisions

prompt_undo() {
    local -n promoted_items=$1
    local colony_name="$2"

    # Store undo info temporarily
    local undo_file="$DATA_DIR/.promotion-undo.json"
    echo "{\"promoted\": $(printf '%s\n' "${promoted_items[@]}" | jq -R . | jq -s .), \"timestamp\": \"$(date -u +%s)\"}" > "$undo_file"

    echo ""
    echo "Undo these promotions? (y/n)"
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo "Reverting promotions..."

        for item in "${promoted_items[@]}"; do
            # Remove from QUEEN.md
            # This requires parsing QUEEN.md and removing matching entries
            revert_promotion "$item" "$colony_name"
        done

        echo "Promotions reverted."
        rm -f "$undo_file"
    else
        echo "Promotions kept."
        # Keep undo file for potential later recovery
    fi
}
```

## State of the Art

### Current Implementation (Phase 33)

| Component | Status | Location |
|-----------|--------|----------|
| `learning-observe` | Implemented | aether-utils.sh:3921 |
| `learning-check-promotion` | Implemented | aether-utils.sh:4066 |
| `queen-promote` | Implemented | aether-utils.sh:3616 |
| Proposal display in continue.md | Basic | continue.md:678-725 |
| User approval | Not implemented | — |

### What's New in Phase 34

| Feature | Old Approach | New Approach |
|---------|--------------|--------------|
| Selection | None (all or nothing) | Checkbox-style multi-select |
| Threshold display | Text only | Visual progress bar |
| Rejected items | Lost | Stored in learning-deferred.json |
| Undo | Not available | Immediate prompt |
| Batch execution | Single promote | Multi-item with partial success handling |

## Open Questions

1. **Undo Implementation Detail**
   - What we know: Undo prompt required after promotion
   - What's unclear: Whether to store undo log in file or just memory
   - Recommendation: Store in `$DATA_DIR/.promotion-undo.json` with 24h TTL

2. **Deferred Item Expiration**
   - What we know: Deferred items stored in learning-deferred.json
   - What's unclear: How long to keep deferred items
   - Recommendation: Auto-expire after 30 days, log cleanup in activity log

3. **Verbose Mode Trigger**
   - What we know: `--verbose` flag shows full content
   - What's unclear: Whether this is passed to continue.md or separate command
   - Recommendation: Support `/ant:continue --deferred --verbose` combo

4. **Error Recovery on Partial Promotion**
   - What we know: Stop on first error, show successes
   - What's unclear: Whether to auto-undo successful promotions on failure
   - Recommendation: Leave successful promotions, log failure for manual retry

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.planning/phases/34-add-user-approval-ux/34-CONTEXT.md` — User decisions and constraints
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh:3616-3918` — queen-promote implementation
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh:3921-4064` — learning-observe implementation
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh:4066-4118` — learning-check-promotion implementation
- `/Users/callumcowie/repos/Aether/.aether/docs/QUEEN.md` — QUEEN.md structure and thresholds
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/continue.md:678-725` — Existing proposal display

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md` — PHER-EVOL-03 requirement definition
- `/Users/callumcowie/repos/Aether/.aether/data/learning-observations.json` — Observation data schema

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All components exist and are tested
- Architecture: HIGH — Clear extension of existing patterns
- Pitfalls: MEDIUM-HIGH — Based on common bash scripting issues

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (stable domain, bash patterns don't change)

---

*Research complete. Ready for planning.*
