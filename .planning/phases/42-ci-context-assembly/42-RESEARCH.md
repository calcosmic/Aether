# Phase 42: CI Context Assembly - Research

**Researched:** 2026-03-31
**Domain:** Shell scripting (bash), JSON context assembly, subcommand architecture
**Confidence:** HIGH

## Summary

Phase 42 builds the `pr-context` subcommand that produces machine-readable JSON colony context for CI agents. The design doc at `.aether/docs/ci-context-assembly-design.md` is comprehensive and authoritative -- it specifies the output schema, fallback chains, cache layer, token budgets, edge cases, and integration points. The implementation is primarily a bash function (`_pr_context`) in `pheromone.sh` that reuses existing functions from `colony-prime` (which lives in the same file) and adds three additions from the context discussion: midden data inclusion, a TTL-based cache system, and extraction of a shared `_budget_enforce()` function.

The reference implementation (`_colony_prime`, pheromone.sh lines 737-1553) provides the exact pattern to follow. pr-context mirrors colony-prime's 10-section assembly but outputs structured JSON instead of a `prompt_section` string, uses softer budgets (6K/3K vs 8K/4K), and never hard-fails. The budget trimming logic (lines 1388-1492) is the most intricate part and will be extracted into a shared `_budget_enforce()` function that both callers use.

**Primary recommendation:** Follow the design doc as the spec. Extract `_budget_enforce()` first (D-07), then build `_pr_context()` with the cache layer and midden section, keeping colony-prime's output unchanged throughout. Test with isolated tmpdir environments following the established `setup_pheromone_env()` pattern.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** pr-context output includes a `midden` section with recent failure entries and cross-PR pattern data
- **D-02:** Midden section is classified as VOLATILE (always read fresh from `.aether/data/midden/midden.json`)
- **D-03:** Midden entries in pr-context output are bounded (top 10 most recent, or all entries from last 7 days -- whichever is smaller)
- **D-04:** Implement the full TTL-based cache system as specified in design doc Section 4. Cache at `.aether/data/pr-context-cache.json` (gitignored, branch-local)
- **D-05:** Cache writes use `acquire_lock` from file-lock.sh. Cache reads are lock-free
- **D-06:** TTL values: QUEEN.md 1 hour, hive/eternal 2 hours. Evict stale entries on each pr-context call
- **D-07:** Extract `_budget_enforce()` shared function from colony-prime (pheromone.sh lines 1388-1492)
- **D-08:** Trim order stays identical to colony-prime (rolling-summary first, blockers never)
- **D-09:** Refactor must not change colony-prime's output. Existing colony-prime tests must pass unchanged
- **D-10:** pr-context NEVER hard-fails. Every source has a fallback chain
- **D-11:** All fallbacks are logged in `fallbacks_used` output array and `warnings` array
- **D-12:** Output matches design doc Section 3.2 schema with addition: `midden` section
- **D-13:** Structured signal arrays (redirects, focus, feedback) as typed JSON
- **D-14:** Wire pr-context into `/ant:continue` and `/ant:run`. CI pipeline integration out of scope
- **D-15:** Dispatch entry: `pr-context) _pr_context "$@" ;;` in aether-utils.sh

### Claude's Discretion
- Exact midden entry format in JSON output (count + recent items, or structured categories)
- Cache eviction granularity (per-entry vs full-cache clear)
- Whether to add `--section` flag for requesting specific sections only
- Exact placement of `_budget_enforce()` extraction (inline in pheromone.sh vs separate utils/ module)

### Deferred Ideas (OUT OF SCOPE)
- CI pipeline workflow files (GitHub Actions) -- Phase 44
- `--section` flag for requesting specific pr-context sections only
- pr-context for OpenCode agents
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CI-01 | `aether pr-context` outputs valid JSON with sections: colony_state, pheromones, phase_context, blockers, hive_wisdom | Design doc Section 3.2 defines complete schema; colony-prime reference implementation in pherone.sh lines 737-1553 provides assembly pattern; midden section added per D-01 |
| CI-02 | When a source file is missing or corrupt, pr-context returns partial data with missing section marked as `null` -- never hard-fails | Design doc Section 5 defines fallback chain per source; D-10/D-11 require soft-fail everywhere with logging; colony-prime's existing soft-fail patterns (pheromone.sh lines 973-986, 1139-1171) provide reference |
| CI-03 | Normal mode output stays under 6,000 characters; compact mode under 3,000 characters; trim order follows colony-prime | Budget enforcement algorithm at pheromone.sh lines 1388-1492; D-07 extracts into `_budget_enforce()`; design doc Section 6 defines budgets and allocation |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| bash | 5.x | Runtime environment | Project standard; all utils are bash scripts |
| jq | 1.7+ | JSON parsing and construction | Required by all aether-utils.sh subcommands; used via `jq -n --arg/--argjson` pattern |
| git | 2.x | Branch resolution, worktree detection | Used by `--branch` flag default (`git rev-parse --abbrev-ref HEAD`) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| file-lock.sh | in-tree | `acquire_lock`/`release_lock` for cache writes | Cache file write operations (D-05) |
| error-handler.sh | in-tree | `json_ok`/`json_err` output formatting | All subcommand output formatting |
| state-api.sh | in-tree | `_state_mutate` for COLONY_STATE.json reads | Colony state section assembly |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Inline `_budget_enforce` in pheromone.sh | Separate utils/budget.sh module | Inline keeps it near colony-prime context; separate module adds import overhead for a ~100-line function used by only 2 callers. Inline is recommended for proximity. |

**Installation:**
No new packages required. All dependencies are in-tree bash scripts.

## Architecture Patterns

### Recommended Project Structure
```
.aether/utils/pheromone.sh
├── _budget_enforce()          # NEW: extracted shared function (~line 730, before _colony_prime)
├── _colony_prime()             # EXISTING: lines 737-1553, MODIFIED to call _budget_enforce()
├── _pr_context()               # NEW: main pr-context implementation (after _colony_prime)
├── _cache_read()               # NEW: cache layer helper
├── _cache_write()              # NEW: cache layer helper
└── _extract_wisdom()           # EXISTING: reused as-is by pr-context

.aether/aether-utils.sh
├── dispatch case entry         # NEW: pr-context) _pr_context "$@" ;;
└── help JSON entry             # NEW: {"name": "pr-context", "description": "..."}

.aether/data/pr-context-cache.json    # NEW: cache file (gitignored)
```

### Pattern 1: Subcommand Registration
**What:** New subcommand follows the established pattern in aether-utils.sh
**When to use:** Any new subcommand
**Example:**
```bash
# In aether-utils.sh case statement (near line 3908):
pr-context) _pr_context "$@" ;;

# In help JSON sections (near line 1254):
{"name": "pr-context", "description": "Generate CI-ready colony context as structured JSON"}
```

### Pattern 2: Isolated Test Environment
**What:** Tests create tmpdir with full aether copy, set AETHER_ROOT override
**When to use:** All pr-context tests
**Example:**
```bash
setup_pr_context_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"
    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    cp -r "$(dirname "$AETHER_UTILS")/utils" "$tmpdir/.aether/"
    # ... create test data files ...
    echo "$tmpdir"
}
# Run: AETHER_ROOT="$tmpdir" bash "$tmpdir/.aether/aether-utils.sh" pr-context
```

### Pattern 3: Budget Enforcement Extraction
**What:** Extract colony-prime's inline budget trimming into a shared function
**When to use:** Both colony-prime and pr-context call this
**Example:**
```bash
# _budget_enforce takes section variables and max_chars, returns trimmed sections
# Input: named section variables (sec_rolling, sec_learnings, etc.) and max_chars
# Output: trims in-place, sets budget_trimmed_list, recalculates final_prompt
_budget_enforce() {
    local -n _be_max_chars=$1
    local -n _be_final_prompt=$2
    # ... trim order: rolling, learnings, decisions, hive, capsule, user_prefs, queen_global, queen_local, signals
    # ... NEVER trim blockers
    # ... preserve REDIRECTs even when signals section trimmed
}
```
**Note on nameref:** Bash 4.3+ supports `local -n` for nameref variables. macOS ships bash 3.2 by default, but this project requires bash 5.x (already the case per existing test patterns using namerefs). However, if nameref proves fragile, the fallback is passing section names as positional args and using eval for variable access -- which is the pattern colony-prime currently uses (inline, no function call).

### Pattern 4: Soft-Fail Fallback Chain
**What:** Every data source read is wrapped with fallback and warning logging
**When to use:** All 8+ source reads in pr-context
**Example:**
```bash
# QUEEN.md fallback (different from colony-prime which hard-fails)
pc_queen_global_data="{}"
pc_fallbacks=()
if [[ -f "$HOME/.aether/QUEEN.md" ]]; then
    pc_queen_global_data=$(_extract_wisdom "$HOME/.aether/QUEEN.md") || {
        pc_fallbacks+=("queen_global: file read failed")
        pc_queen_global_data="{}"
    }
else
    pc_fallbacks+=("queen_global: no file found")
fi
```

### Anti-Patterns to Avoid
- **Hard-failing on missing files:** colony-prime exits 1 on no QUEEN.md (line 924-929). pr-context must NOT do this. Every missing source returns empty/null with a warning.
- **Duplicating colony-prime logic:** pr-context should call `_extract_wisdom()`, `pheromone-prime`, `hive-read`, and `context-capsule` directly -- not re-implement them.
- **Changing colony-prime output during extraction:** D-09 requires existing colony-prime tests pass unchanged. The `_budget_enforce()` extraction must be a pure refactor -- same variable names, same trim order, same output.
- **Locking on cache reads:** D-05 specifies lock-free reads. Only cache writes acquire a lock.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| QUEEN.md parsing | Custom section parser | `_extract_wisdom()` in pheromone.sh | Handles v1/v2 format detection, 4-section and 6-section variants |
| Wisdom entry filtering | Regex-based line filtering | `_filter_wisdom_entries()` in pheromone.sh | Strips boilerplate, keeps actual entries and phase headers |
| Pheromone signal assembly | Direct pheromones.json read | `pheromone-prime` subcommand via aether-utils.sh | Handles signal counting, instinct extraction, decay calculation |
| Hive wisdom retrieval | Direct wisdom.json read | `hive-read` subcommand via aether-utils.sh | Domain scoping, confidence filtering, access tracking, eternal fallback |
| Colony state snapshot | Direct COLONY_STATE.json read | `context-capsule` subcommand via aether-utils.sh | Bounded output, compact mode, handles missing file |
| File locking for cache | Custom lock mechanism | `acquire_lock`/`release_lock` from file-lock.sh | Stale lock detection, PID tracking, timeout handling |
| JSON output formatting | echo/printf | `json_ok()` from error-handler.sh | Consistent `{"ok":true,"result":...}` envelope |
| Midden cross-PR analysis | Custom analysis logic | `midden-cross-pr-analysis` subcommand | Already computes systemic/critical classifications |

**Key insight:** pr-context is primarily an orchestrator that calls existing subcommands and assembles their output into a structured JSON envelope. The assembly logic itself is straightforward; the complexity is in the fallback chains, budget enforcement, and cache layer.

## Common Pitfalls

### Pitfall 1: Budget Extraction Breaking colony-prime
**What goes wrong:** Extracting `_budget_enforce()` changes variable scoping or nameref behavior, causing colony-prime to produce different output
**Why it happens:** Bash nameref (`local -n`) has subtle scoping rules, especially when calling functions from within functions that share variable names
**How to avoid:** Extract as a pure refactor first. Run `test-colony-prime-budget.sh` before and after extraction. Compare output character-for-character. Consider keeping budget logic inline as nested helper within `_colony_prime` scope rather than using namerefs -- the safest extraction is to move the block into a function that reads/writes the same global-prefixed variables (e.g., `cp_sec_*` and `cp_budget_trimmed_list`).
**Warning signs:** Any change in colony-prime test output after extraction

### Pitfall 2: Cache Mtime Comparison on macOS
**What goes wrong:** macOS `stat` uses different flags than Linux (`stat -f %m` vs `stat -c %Y`)
**Why it happens:** The codebase already handles this (existing cross-platform patterns in context-capsule), but cache mtime extraction needs the same treatment
**How to avoid:** Use `date -r "$file" +%s` for mtime extraction (works on both macOS and Linux with coreutils), or use the existing `_cross_platform_stat` pattern from the codebase
**Warning signs:** Cache always invalidates on macOS, or returns stale data on Linux

### Pitfall 3: JSON Construction with Special Characters
**What goes wrong:** Content from QUEEN.md, pheromones, or midden entries contains quotes, newlines, or backslashes that break `jq -n --arg` construction
**Why it happens:** Shell variable interpolation into JSON is fragile. The codebase uses `jq -Rs '.'` for raw string escaping and `jq -n --arg` for safe insertion
**How to avoid:** Always use `jq -Rs '.'` to escape raw text content before inserting into JSON. Never interpolate shell variables directly into JSON strings. The existing `json_ok` + `jq -n --arg` pattern handles this correctly.
**Warning signs:** JSON parse errors in pr-context output for entries with special characters

### Pitfall 4: Concurrent Cache Writes from Parallel CI Runs
**What goes wrong:** Two CI runs on the same branch both write cache simultaneously, corrupting the file
**Why it happens:** PR-based workflow may trigger multiple agents on the same branch
**How to avoid:** Use `acquire_lock` on the cache file before writing (D-05). Read is lock-free (safe for concurrent reads). The existing lock mechanism handles stale locks and timeouts.
**Warning signs:** Cache file contains invalid JSON after concurrent CI runs

### Pitfall 5: Midden Section Token Budget Overrun
**What goes wrong:** Midden entries with long descriptions push total output over budget
**Why it happens:** Midden data is volatile (D-02) and cannot be predictably bounded at assembly time
**How to avoid:** Bound midden output per D-03 (top 10 recent or last 7 days, whichever is smaller). Include midden in the trim order between phase-learnings and key-decisions priority (it is "nice to have" context for CI agents). Truncate individual midden entry descriptions to 160 chars (matching context-capsule pattern at aether-utils.sh line 4254).
**Warning signs:** pr-context output exceeds budget consistently when midden has many entries

## Code Examples

### Budget Enforcement Reference (from pheromone.sh lines 1388-1492)
```bash
# This is the EXACT pattern to extract into _budget_enforce()
# Colony-prime uses cp_ prefix; pr-context will use pc_ prefix
# Both call the same logic with different max_chars

# Trim order (first = trimmed first):
# 1. rolling-summary
# 2. phase-learnings
# 3. key-decisions
# 4. hive-wisdom
# 5. context-capsule
# 6. user-prefs
# 7. queen-wisdom-global
# 8. queen-wisdom-local
# 9. pheromone-signals (preserve REDIRECTs)
# 10. blockers: NEVER trimmed

if [[ "$cp_budget_len" -gt "$cp_max_chars" ]]; then
  # Step 1: trim rolling-summary
  if [[ "$cp_budget_len" -gt "$cp_max_chars" && -n "$cp_sec_rolling" ]]; then
    cp_sec_rolling=""
    cp_budget_trimmed_list="rolling-summary"
    cp_final_prompt="$cp_sec_queen_global$cp_sec_queen_local$cp_sec_user_prefs$cp_sec_hive$cp_sec_capsule$cp_sec_learnings$cp_sec_decisions$cp_sec_blockers$cp_sec_rolling$cp_sec_signals"
    cp_budget_len=${#cp_final_prompt}
  fi
  # ... repeat for each section ...
fi
```

### Cache Read Pattern (per design doc Section 4)
```bash
_cache_read() {
    local cache_file="$COLONY_DATA_DIR/pr-context-cache.json"
    local source_name="$1"  # e.g., "queen_global"
    local source_path="$2"  # e.g., "$HOME/.aether/QUEEN.md"
    local ttl_seconds="$3"  # e.g., 3600

    if [[ ! -f "$cache_file" ]]; then
        echo "null"  # Cache miss
        return
    fi

    # Get current mtime of source file
    local current_mtime
    current_mtime=$(stat -f %m "$source_path" 2>/dev/null || date -r "$source_path" +%s 2>/dev/null || echo "0")

    # Read cached entry
    local cached_mtime cached_at cached_data
    cached_mtime=$(jq -r --arg src "$source_name" '.entries[$src].mtime // 0' "$cache_file" 2>/dev/null)
    cached_data=$(jq -r --arg src "$source_name" '.entries[$src].data // null' "$cache_file" 2>/dev/null)

    # Validate: mtime unchanged and not past TTL
    if [[ "$cached_mtime" == "$current_mtime" && "$cached_data" != "null" ]]; then
        echo "$cached_data"
        return
    fi

    echo "null"  # Cache miss (mtime changed or TTL expired)
}
```

### Soft-Fail JSON Output Pattern
```bash
# pr-context NEVER hard-fails. Output is always valid JSON.
# Missing sources produce null/empty sections with warnings.

pc_warnings=()
pc_fallbacks=()

# Example: colony_state section
pc_colony_state="null"
if [[ -f "$COLONY_DATA_DIR/COLONY_STATE.json" ]]; then
    pc_goal=$(jq -r '.goal // "No goal set"' "$COLONY_DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "No goal set")
    pc_state=$(jq -r '.state // "UNKNOWN"' "$COLONY_DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "UNKNOWN")
    pc_colony_state=$(jq -nc --arg goal "$pc_goal" --arg state "$pc_state" \
        '{exists:true, goal:$goal, state:$state}')
else
    pc_fallbacks+=("colony_state: COLONY_STATE.json missing")
    pc_colony_state='{"exists":false,"goal":"No goal set","state":"UNKNOWN","current_phase":0,"total_phases":0,"phase_name":""}'
fi
```

### Midden Section Assembly (per D-01/D-02/D-03)
```bash
# Midden is VOLATILE -- always fresh read, never cached
pc_midden='{"count":0,"entries":[],"systemic_categories":[]}'
if [[ -f "$COLONY_DATA_DIR/midden/midden.json" ]]; then
    # Bound: top 10 recent OR last 7 days (whichever is smaller)
    local midden_cutoff
    midden_cutoff=$(date -u -v-"7d" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "")

    pc_midden=$(jq --arg cutoff "$midden_cutoff" '
        [.entries // [] | .[] |
            select(if ($cutoff | length) > 0 then .timestamp >= $cutoff else true end)
        ] | .[:10]  # Cap at 10
        | {count: length, entries: .}
    ' "$COLONY_DATA_DIR/midden/midden.json" 2>/dev/null || echo '{"count":0,"entries":[]}')

    # Add cross-PR analysis (non-blocking)
    local cross_pr
    cross_pr=$(bash "$SCRIPT_DIR/aether-utils.sh" midden-cross-pr-analysis --window 14 2>/dev/null || echo '{}')
    pc_midden=$(echo "$pc_midden" | jq --argjson cross "$cross_pr" '. + {cross_pr_analysis: $cross}' 2>/dev/null || echo "$pc_midden")
else
    pc_fallbacks+=("midden: midden.json missing")
fi
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| colony-prime hard-fail on missing QUEEN.md | pr-context soft-fail with empty wisdom | Phase 42 (this phase) | CI agents get partial context instead of nothing |
| Inline budget enforcement in colony-prime | Shared `_budget_enforce()` function | Phase 42 (this phase) | Two callers use same trim logic with different budgets |
| Formatted text signals in prompt_section | Structured typed JSON signal arrays | Phase 42 (this phase) | CI agents can programmatically parse redirect/focus/feedback |
| No caching of context sources | TTL-based mtime cache for QUEEN.md/hive/eternal | Phase 42 (this phase) | Repeated CI calls avoid re-reading unchanged files |

**Deprecated/outdated:**
- Hard-fail on missing QUEEN.md: colony-prime still does this (correct for interactive use), but pr-context must NOT

## Open Questions

1. **Budget enforce extraction strategy**
   - What we know: The existing code uses inline bash with `cp_sec_*` variable naming and string concatenation. Nameref (`local -n`) requires bash 4.3+.
   - What's unclear: Whether the simplest extraction is (a) a function that takes prefixed variable names and uses eval, (b) a function using namerefs, or (c) a heredoc/template that both callers instantiate.
   - Recommendation: Use approach (a) -- pass the prefix string ("cp_" or "pc_") and use indirect variable access (`eval`) to read/trim sections. This is the pattern already used elsewhere in the codebase for similar dynamic variable access.

2. **Midden section placement in trim order**
   - What we know: Midden is "nice to have" per the design doc's priority classification. D-03 bounds it to keep it small.
   - What's unclear: Exact position between other "nice to have" sections (phase-learnings, key-decisions, rolling-summary).
   - Recommendation: Place midden at position 2.5 (between rolling-summary and phase-learnings). It is less critical than learnings but similarly volatile.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| bash 5.x | All subcommands | Verified | 5.x | -- |
| jq | JSON parsing/construction | Verified | 1.7+ | -- |
| git | Branch resolution | Verified | 2.x | -- |
| file-lock.sh | Cache write locking | In-tree | -- | -- |
| midden.sh | Midden data reading | In-tree | -- | -- |
| pheromone.sh | Signal assembly, wisdom extraction | In-tree | -- | -- |
| state-api.sh | Colony state reads | In-tree | -- | -- |

**Missing dependencies with no fallback:**
None -- all dependencies are in-tree bash scripts or standard CLI tools.

**Missing dependencies with fallback:**
None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Bash + test-helpers.sh (custom) |
| Config file | None -- each test file is self-contained |
| Quick run command | `bash tests/bash/test-pr-context.sh -x` |
| Full suite command | `npm test` (runs all 80 test files) |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CI-01 | Valid JSON output with all required sections | unit | `bash tests/bash/test-pr-context.sh::test_output_has_required_sections` | Wave 0 |
| CI-01 | Midden section present with bounded entries | unit | `bash tests/bash/test-pr-context.sh::test_midden_section_bounded` | Wave 0 |
| CI-01 | Cache status included in output | unit | `bash tests/bash/test-pr-context.sh::test_cache_status` | Wave 0 |
| CI-02 | Missing COLONY_STATE.json returns partial with null | unit | `bash tests/bash/test-pr-context.sh::test_missing_colony_state` | Wave 0 |
| CI-02 | Missing pheromones.json returns partial with warning | unit | `bash tests/bash/test-pr-context.sh::test_missing_pheromones` | Wave 0 |
| CI-02 | Missing QUEEN.md returns empty wisdom (no hard fail) | unit | `bash tests/bash/test-pr-context.sh::test_no_queen_md_soft_fail` | Wave 0 |
| CI-02 | Corrupt JSON files handled gracefully | unit | `bash tests/bash/test-pr-context.sh::test_corrupt_json_fallback` | Wave 0 |
| CI-03 | Normal mode under 6,000 characters | unit | `bash tests/bash/test-pr-context.sh::test_normal_mode_budget` | Wave 0 |
| CI-03 | Compact mode under 3,000 characters | unit | `bash tests/bash/test-pr-context.sh::test_compact_mode_budget` | Wave 0 |
| CI-03 | Trim order matches colony-prime | unit | `bash tests/bash/test-pr-context.sh::test_trim_order` | Wave 0 |
| CI-03 | Blockers never trimmed | unit | `bash tests/bash/test-pr-context.sh::test_blockers_never_trimmed` | Wave 0 |
| D-09 | colony-prime output unchanged after extraction | regression | `bash tests/bash/test-colony-prime-budget.sh` | Existing |

### Sampling Rate
- **Per task commit:** `bash tests/bash/test-pr-context.sh`
- **Per wave merge:** `npm test`
- **Phase gate:** Full suite green + colony-prime budget tests green

### Wave 0 Gaps
- [ ] `tests/bash/test-pr-context.sh` -- covers CI-01, CI-02, CI-03 (new file)
- [ ] Existing `tests/bash/test-colony-prime-budget.sh` must pass unchanged (regression guard for D-09)

## Sources

### Primary (HIGH confidence)
- `.aether/docs/ci-context-assembly-design.md` -- Complete specification (835 lines), verified against codebase
- `.aether/utils/pheromone.sh` lines 737-1553 -- Reference implementation (colony-prime)
- `.aether/utils/pheromone.sh` lines 1388-1492 -- Budget enforcement logic to extract
- `.aether/aether-utils.sh` lines 4172-4371 -- context-capsule subcommand reference
- `.aether/utils/midden.sh` lines 526-605, 831-944 -- midden-collect and midden-cross-pr-analysis

### Secondary (MEDIUM confidence)
- `.aether/utils/file-lock.sh` -- acquire_lock/release_lock pattern
- `.aether/docs/state-contract-design.md` -- Branch-local vs hub-global state rules
- `tests/bash/test-colony-prime-budget.sh` -- Existing test pattern (807 lines)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All dependencies are in-tree bash scripts, verified present
- Architecture: HIGH - Design doc is comprehensive; colony-prime is direct reference implementation
- Pitfalls: HIGH - Based on direct code analysis of existing patterns and known bash portability issues
- Budget extraction: MEDIUM - Nameref/eval extraction in bash has edge cases; recommend testing thoroughly

**Research date:** 2026-03-31
**Valid until:** 2026-04-30 (stable -- bash/jq ecosystem, in-tree code)
