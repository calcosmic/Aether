# Phase 32: Wire QUEEN.md into Commands - Research

**Researched:** 2026-02-20
**Domain:** Worker priming, colony wisdom injection, two-level file loading
**Confidence:** HIGH

## Summary

Phase 32 requires wiring existing queen-* commands into slash commands (init.md, build.md) to automatically provide workers with colony wisdom. The implementation involves creating a unified `colony-prime()` function that combines queen-read (wisdom), pheromone-prime (signals), and instincts into a single worker context. Key technical decisions include handling the two-level QUEEN.md architecture (global + local), fail-graceful error handling, and integrating into existing build.md flow.

**Primary recommendation:** Implement `colony-prime()` as a unified function in aether-utils.sh that internally calls `queen-read` and `pheromone-prime`, with build.md calling this single function rather than multiple separate calls.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **All three sources:** Workers receive wisdom (QUEEN.md) + pheromones (FOCUS/REDIRECT/FEEDBACK) + instincts
- **Format:** Mixed format — QUEEN.md stays markdown, pheromones stay XML, instincts in markdown
- **QUEEN.md content:** Categories only (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees) — metadata and evolution log excluded from worker context
- **Two-level architecture:** Global ~/.aether/QUEEN.md loads first, then local .aether/QUEEN.md (like CLAUDE.md pattern)
- **Instincts:** Dynamic per colony, stored as a section within QUEEN.md
- **colony-prime() function:** New unified function that internally calls queen-read + pheromone-prime
- **build.md integration:** Call colony-prime() once for unified worker context (not multiple separate calls)
- **Fail gracefully:** If sub-functions fail, log warnings but don't crash the build
- **QUEEN.md missing:** Fail hard — stop build with clear error message requiring user to run /ant:init
- **pheromones.json missing:** Silently continue with warning — don't block the build, workers just won't receive pheromone signals
- **Template creation:** init.md creates default QUEEN.md from template (no fallback to running without it)
- **Metadata format:** JSON inside HTML comment (`<!-- ... -->`)
- **Metadata fields:** version, thresholds (philosophy:5, pattern:3, redirect:2, stack:1, decree:0), stats (counts per category)
- **Metadata storage:** Inline in QUEEN.md HTML comment block (not separate file, not in colony state)

### Claude's Discretion
- Exact placement of colony-prime() call in build.md
- Error message wording for QUEEN.md missing
- How to handle partial template corruption
- Default instincts content for new QUEEN.md files

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| QUEEN-01 | QUEEN.md file structure with 5 wisdom categories (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees) | Existing structure verified in .aether/docs/QUEEN.md |
| QUEEN-02 | queen-init command creates QUEEN.md from template (called by init.md) | Already implemented in aether-utils.sh |
| QUEEN-03 | queen-read command returns wisdom as JSON for worker priming (called by build.md) | Already implemented, returns structured JSON |
| QUEEN-05 | Metadata block with version, stats, thresholds (in HTML comment) | Already exists in QUEEN.md template |
| INT-01 | init.md calls queen-init after bootstrap (QUEEN.md created) | Already wired in init.md Step 1.6 |
| INT-02 | build.md calls queen-read before spawning workers (wisdom injected) | Already in build.md Step 4.1 |
| PHER-EVOL-01 | Pheromones automatically injected at key workflow points | Already in build.md Step 4 and Step 4.1.6 |
| PHER-EVOL-04 | Pheromone history tracking in colony state | Already stored in pheromones.json |
| PRIME-01 | colony-prime function combines wisdom + signals + instincts | NEW - needs implementation |
| PRIME-02 | build.md uses colony-prime for unified worker context | NEW - needs implementation |
| PRIME-03 | Workers receive structured colony context (not multiple overlapping calls) | NEW - needs implementation |
| META-03 | Stats block tracks counts per category | Already in QUEEN.md METADATA block |

**Gap Analysis:** The core infrastructure exists (queen-read, pheromone-prime, init.md, build.md integration points). The missing piece is the unified `colony-prime()` function that combines these into a single call for cleaner worker context injection.
</phase_requirements>

## Standard Stack

### Core (Already Existing)
| Component | Version | Purpose | Status |
|-----------|---------|---------|--------|
| queen-read | Existing | Extract QUEEN.md wisdom as JSON | Implemented |
| queen-init | Existing | Create QUEEN.md from template | Implemented |
| pheromone-prime | Existing | Combine signals + instincts into prompt | Implemented |
| pheromones.json | Existing | Store active pheromone signals | Implemented |
| COLONY_STATE.json | Existing | Store colony state and instincts | Implemented |

### Implementation Needed
| Component | Purpose | Notes |
|-----------|---------|-------|
| colony-prime() | Unified function combining queen-read + pheromone-prime | New function in aether-utils.sh |
| Two-level loading | Global + local QUEEN.md | Uses AETHER_ROOT pattern |

### Installation
No new packages required - all functionality exists in aether-utils.sh.

## Architecture Patterns

### Pattern 1: Unified Worker Priming (colony-prime)

**What:** Single function that combines all worker context sources into one call

**When to use:** Before spawning any worker in build.md

**Implementation approach:**
```
colony-prime()
  ├─ queen-read (wisdom from QUEEN.md)
  ├─ pheromone-prime (signals + instincts)
  └─ Format combined output for worker injection
```

**Code structure in aether-utils.sh:**
```bash
colony-prime)
    # Load global QUEEN.md first (~/.aether/QUEEN.md)
    # Load local QUEEN.md second (.aether/docs/QUEEN.md)
    # Combine wisdom sections
    # Call pheromone-prime for signals+instincts
    # Return unified JSON with:
    #   - wisdom object (combined from both levels)
    #   - signals object (pheromone-prime output)
    #   - prompt_section (formatted markdown)
    ;;
```

### Pattern 2: Two-Level QUEEN.md Loading

**What:** Global QUEEN.md loads first, then project-local QUEEN.md overrides/extends

**When to use:** In queen-read and colony-prime functions

**Load order:**
1. Global: `~/.aether/QUEEN.md` (system-wide wisdom)
2. Local: `$AETHER_ROOT/.aether/docs/QUEEN.md` (project-specific wisdom)

**Merge strategy:** Local wisdom extends/overrides global - same categories, local entries added after global

### Pattern 3: Build.md Integration Point

**Current state:** build.md calls:
- Step 4: pheromone-prime (signals + instincts)
- Step 4.1: queen-read (wisdom)
- Step 4.1.6: pheromone-read (separate signals)

**Target state:** Single call to colony-prime() in Step 4 area, replacing all three separate calls

**Recommended placement:** After Step 4 (Load Constraints), before Step 4.0 (Territory Survey)

## Common Pitfalls

### Pitfall 1: Multiple Overlapping Context Injections
**What goes wrong:** Workers receive duplicated context from multiple function calls
**Why it happens:** build.md currently calls pheromone-prime, queen-read, and pheromone-read separately
**How to avoid:** Single colony-prime() call, formatted output injected once
**Warning signs:** Worker prompts showing duplicate sections

### Pitfall 2: QUEEN.md Missing Error Handling
**What goes wrong:** Build fails silently or with unhelpful error
**Why it happens:** queen-read returns error but build.md doesn't handle it properly
**How to avoid:** According to CONTEXT.md - fail HARD with clear message to run /ant:init
**Warning signs:** Generic "command failed" messages

### Pitfall 3: Pheromone Failure Blocking Build
**What goes wrong:** Missing pheromones.json crashes the build
**Why it happens:** Different error handling than QUEEN.md
**How to avoid:** According to CONTEXT.md - silently continue with warning, don't block build
**Warning signs:** Build stops when pheromones.json doesn't exist

### Pitfall 4: Metadata Exposed to Workers
**What goes wrong:** METADATA block (JSON) gets injected into worker prompts
**Why it happens:** queen-read extracts full QUEEN.md content including METADATA comment
**How to avoid:** According to CONTEXT.md - only categories (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees) go to workers, metadata and evolution log excluded

## Code Examples

### Example 1: queen-read Output Format (Verified)
Source: `.aether/aether-utils.sh:3446-3476`

```bash
# Returns JSON with structure:
{
  "metadata": { "version": "1.0.0", "stats": {...}, "thresholds": {...} },
  "wisdom": {
    "philosophies": "markdown content",
    "patterns": "markdown content",
    "redirects": "markdown content",
    "stack_wisdom": "markdown content",
    "decrees": "markdown content"
  },
  "priming": {
    "has_philosophies": true/false,
    "has_patterns": true/false,
    ...
  }
}
```

### Example 2: pheromone-prime Output Format (Verified)
Source: `.aether/aether-utils.sh:4539-4577`

```bash
# Returns JSON with structure:
{
  "signal_count": 2,
  "instinct_count": 3,
  "prompt_section": "--- ACTIVE SIGNALS ---\nFOCUS: ...\nREDIRECT: ...\n--- INSTINCTS ---\n...",
  "log_line": "Primed: 2 signals, 3 instincts"
}
```

### Example 3: Worker Prompt Injection Template (Verified)
Source: `.claude/commands/ant/build.md:618-642`

```markdown
--- QUEEN WISDOM (Eternal Guidance) ---
📜 Philosophies:
{queen_philosophies}
🧭 Patterns:
{queen_patterns}
⚠️ Redirects (AVOID these):
{queen_redirects}
🔧 Stack Wisdom:
{queen_stack_wisdom}
🏛️ Decrees:
{queen_decrees}
--- END QUEEN WISDOM ---
```

### Example 4: Two-Level File Loading Pattern
Source: Based on CLAUDE.md pattern (global + local)

```bash
# Pseudo-code for two-level loading
global_queen="$HOME/.aether/QUEEN.md"
local_queen="$AETHER_ROOT/.aether/docs/QUEEN.md"

# Load global first
if [[ -f "$global_queen" ]]; then
  global_wisdom=$(queen-read "$global_queen")
fi

# Load local second (overrides/extends global)
if [[ -f "$local_queen" ]]; then
  local_wisdom=$(queen-read "$local_queen")
fi

# Merge wisdom (local entries added to global)
combined_wisdom=$(merge_wisdom "$global_wisdom" "$local_wisdom")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No worker priming | queen-read called in build.md Step 4.1 | Phase 28 | Workers now receive wisdom |
| Pheromones separate | pheromone-prime in build.md Step 4 | Phase 28 | Workers receive signals+instincts |
| Multiple overlapping calls | Unified colony-prime() needed | Phase 32 | Cleaner worker context |

**Current gap:** Multiple separate calls (pheromone-prime, queen-read, pheromone-read) in build.md need consolidation into single colony-prime() call.

**Deprecated/outdated:**
- Separate pheromone-read call (Step 4.1.6) - will be replaced by colony-prime
- Multiple wisdom/signal injections - will be consolidated

## Open Questions

1. **Two-level loading implementation details**
   - What we know: Global first, then local. Local extends/overrides global.
   - What's unclear: How to handle duplicate entries? Same category entries from both files?
   - Recommendation: Append local entries after global entries, don't deduplicate

2. **colony-prime() error handling strategy**
   - What we know: queen-read should fail hard, pheromone-prime should fail gracefully
   - What's unclear: If queen-read succeeds but pheromone-prime fails, what gets returned?
   - Recommendation: Return wisdom even if signals fail, with warning in log_line

3. **Metadata in combined output**
   - What we know: Workers shouldn't receive metadata, only categories
   - What's unclear: Should colony-prime() return metadata for display purposes?
   - Recommendation: Include metadata in JSON response but not in prompt_section

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` - queen-read (lines 3419-3483), pheromone-prime (lines 4460-4580)
- `.claude/commands/ant/build.md` - Current integration points (Steps 4, 4.1, 4.1.6)
- `.claude/commands/ant/init.md` - queen-init call (Step 1.6)
- `.aether/docs/QUEEN.md` - Current template structure

### Secondary (MEDIUM confidence)
- Phase requirements in `.planning/REQUIREMENTS.md` - Requirement IDs and descriptions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All components exist, only unification needed
- Architecture: HIGH - Clear patterns from existing code
- Pitfalls: HIGH - Identified from current build.md issues

**Research date:** 2026-02-20
**Valid until:** 90 days (stable implementation, no fast-moving changes)
