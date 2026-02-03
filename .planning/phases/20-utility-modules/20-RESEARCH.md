# Phase 20: Utility Modules - Research

**Researched:** 2026-02-03
**Domain:** Bash shell scripting, jq JSON processing, floating-point math, state validation
**Confidence:** HIGH

## Summary

Phase 20 implements 18 subcommands across 4 utility modules (pheromone math, state validation, memory ops, error tracking) inside the existing `aether-utils.sh` scaffold from Phase 19. All computation uses `jq` -- which at version 1.8.1 on this macOS system supports `exp`, `fromdate`, `todate`, `group_by`, `now`, and custom function definitions. No `awk`, `bc`, or Python is needed for any operation.

The pheromone decay formula is already established across 9 command files as `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`. The base-e exponential decay with ln(2) = 0.693147... is canonical. Null `half_life_seconds` means the signal persists forever. The threshold for "expired" is `current_strength < 0.05`. The "combine" operation for conflicting signals uses a net-effect subtraction: `max(0, signal1 - signal2)`.

The tight 300-line budget (with 47 lines already used by the scaffold) requires maximally compact code. The key strategy is: jq does all the heavy lifting as inline expressions in each case branch, shell handles only argument validation, file existence checks, and atomic writes. Estimated total is 240-260 lines.

**Primary recommendation:** Implement all 18 subcommands as inline jq expressions within the existing case-dispatch pattern. Use jq for ALL computation (decay math, schema validation, token counting, date handling). Do not use awk, bc, or Python.

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| bash | 3.2+ | Shell scripting, argument handling | Already the scaffold language, macOS default |
| jq | 1.8.1 | ALL computation: math, JSON manipulation, dates, validation | Verified: has `exp`, `fromdate`, `now`, `group_by`, custom functions |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| date | Generate ISO-8601 timestamps for new records | error-add, pheromone-cleanup |
| head + od | Generate random hex for IDs | error-add (4-char random hex) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| jq for math | awk | awk needs separate invocation; jq already loaded for JSON ops |
| jq `now` for epoch | `date +%s` | jq `now` works but returns float; `date +%s` is cleaner for shell variables |
| jq `fromdate` | manual date parsing | fromdate handles ISO-8601 natively, no parsing needed |
| Python for validation | jq type checks | jq `has()`, `type`, custom functions handle all validation needs |

## Architecture Patterns

### Recommended Project Structure
```
.aether/
  aether-utils.sh           # Single file, ~250-300 lines, ALL 18 subcommands
  utils/
    file-lock.sh             # Existing, sourced by aether-utils.sh
    atomic-write.sh          # Existing, sourced by aether-utils.sh
  data/
    COLONY_STATE.json        # Validated by validate-state colony
    pheromones.json           # Read by pheromone-batch, modified by pheromone-cleanup
    errors.json               # Modified by error-add, read by error-pattern-check/summary/dedup
    memory.json               # Read by memory-token-count/search, modified by memory-compress
    events.json               # Validated by validate-state events
```

### Pattern 1: Inline jq Computation in Case Branches
**What:** Each subcommand is a case branch containing argument validation, a jq expression, and output formatting.
**When to use:** Every subcommand.
**Example:**
```bash
pheromone-decay)
  [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
  json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" \
    '($s|tonumber) * ((-0.693147180559945 * ($e|tonumber) / ($h|tonumber)) | exp) | {strength: (. * 1000000 | round / 1000000)}')"
  ;;
```

### Pattern 2: File Read + jq Transform + Atomic Write
**What:** Subcommands that modify state files read the file, pipe through jq, and write atomically.
**When to use:** pheromone-cleanup, error-add, error-dedup, memory-compress.
**Example:**
```bash
pheromone-cleanup)
  [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
  local now; now=$(date -u +%s)
  local result; result=$(jq --arg now "$now" '
    .signals |= map(select(
      .half_life_seconds == null or
      (.strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | fromdate)) / .half_life_seconds) | exp)) >= 0.05
    ))
  ' "$DATA_DIR/pheromones.json") || json_err "Failed to process pheromones.json"
  atomic_write "$DATA_DIR/pheromones.json" "$result"
  json_ok '{"cleaned":true}'
  ;;
```

### Pattern 3: Schema Validation via jq Type Checks
**What:** Validation subcommands use jq `has()` and `type` to check field presence and types.
**When to use:** All validate-state subcommands.
**Example:**
```bash
# Reusable validation helper (defined once at top of script)
validate_json() {
  local file="$1" check="$2"
  [[ -f "$file" ]] || { echo "missing"; return 1; }
  jq -e "$check" "$file" >/dev/null 2>&1 && echo "pass" || { jq "$check" "$file" 2>&1; return 1; }
}
```

### Pattern 4: Argument Validation Gate
**What:** Each subcommand validates its arguments before any computation.
**When to use:** Every subcommand that takes arguments.
**Example:**
```bash
error-add)
  [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description>"
  [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
  ;;
```

### Anti-Patterns to Avoid
- **Using awk or bc for math:** jq handles all floating-point computation natively with `exp`, `log`, arithmetic operators.
- **Separate module files:** Do NOT split into pheromone.sh, validation.sh, etc. The 300-line budget assumes a single file. Separate files add source overhead and complexity.
- **Validating categories in error-add:** The success criteria example uses `build` as a category, but the error schema defines 12 specific categories that do NOT include `build`. Accept any string as category -- let validate-state catch invalid categories during validation runs rather than rejecting at add time.
- **Complex jq functions defined as shell variables:** Keep jq expressions inline in each case branch. Defining reusable jq functions as shell variables makes code harder to read and debug.
- **Python or Node for any operation:** The constraint is shell + jq only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Exponential decay math | Shell arithmetic or bc | jq `exp()` function | jq 1.8.1 has native `exp`; avoids shell float limitations |
| ISO-8601 date parsing | Manual string parsing | jq `fromdate` | Handles ISO-8601 natively, returns epoch seconds |
| Current epoch time | Multiple `date` calls | jq `now` or single `date -u +%s` | Consistent timestamp within a jq expression |
| JSON field type checking | grep/sed on JSON | jq `type`, `has()` | Type-safe, handles null correctly |
| Array grouping/counting | Shell loops over JSON | jq `group_by`, `length` | Single jq call vs N shell iterations |
| Atomic file writes | Manual temp+mv | `atomic_write` from atomic-write.sh | Already sourced, handles backup + validation |
| File locking | Manual lock files | `acquire_lock`/`release_lock` from file-lock.sh | Already sourced, handles stale locks |
| Random hex generation | $RANDOM formatting | `head -c 2 /dev/urandom \| od -An -tx1 \| tr -d ' '` | Cryptographic randomness, 4-char hex |

**Key insight:** jq 1.8.1 is essentially a complete computation engine for this use case. Every operation -- math, dates, string manipulation, type checking, grouping, filtering -- can be expressed as a jq pipeline. Shell is only the dispatch layer.

## Common Pitfalls

### Pitfall 1: jq `fromdate` Requires Exact ISO-8601 Format
**What goes wrong:** jq `fromdate` fails on timestamps with fractional seconds like `2026-02-03T12:00:00.123Z`.
**Why it happens:** jq's `fromdate` expects `%Y-%m-%dT%H:%M:%SZ` exactly.
**How to avoid:** Strip fractional seconds before parsing: `sub("\\.[0-9]+Z$"; "Z") | fromdate`. OR ensure all timestamps are written without fractional seconds (which the existing commands already do).
**Warning signs:** jq errors like `date "2026-02-03T12:00:00.123Z" does not match format`.

### Pitfall 2: The 300-Line Budget Is Tight
**What goes wrong:** Implementation exceeds 300 lines, violating the constraint.
**Why it happens:** 18 subcommands + scaffold + helpers + comments in 300 lines is approximately 14 lines per subcommand average.
**How to avoid:** Keep each subcommand to 5-10 lines. Use single-line jq expressions where possible. Minimize comments (the research doc serves as documentation). Share validation helpers. The estimated breakdown:
- Scaffold (existing): 47 lines
- Help text updates: 5 lines
- Shared helpers (validate_json): 5 lines
- Pheromone module (5 cmds): ~40 lines
- Validation module (6 cmds): ~60 lines
- Memory module (3 cmds): ~30 lines
- Error module (4 cmds): ~50 lines
- Total: ~237 lines (63 lines of headroom)
**Warning signs:** Any single subcommand exceeding 15 lines.

### Pitfall 3: Category Validation Conflict
**What goes wrong:** error-add rejects `build` as an invalid category, but success criteria #4 uses `error-add build high "Test failure in auth module"`.
**Why it happens:** The error schema in build.md defines 12 categories (syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security). `build` is not among them.
**How to avoid:** Do NOT validate category in error-add. Accept any string. The validate-state errors subcommand can optionally flag non-standard categories, but error-add should be permissive.
**Warning signs:** Success criteria #4 failing because "build" is rejected.

### Pitfall 4: Atomic Write Passes Content via Shell Variable
**What goes wrong:** JSON content with special characters (quotes, newlines) breaks when passed through shell variable expansion.
**Why it happens:** `atomic_write "$file" "$content"` uses echo internally, which may mangle content.
**How to avoid:** For subcommands that modify files, pipe jq output to a temp file and use `atomic_write_from_file` instead. OR write directly (jq output > temp file, then mv). Given the existing atomic-write.sh uses `echo "$content"`, test with complex JSON to ensure correctness.
**Warning signs:** Corrupted JSON after write operations.

### Pitfall 5: macOS `date` Nanoseconds
**What goes wrong:** `date +%s%N` returns `%N` literally on macOS (BSD date does not support nanoseconds).
**Why it happens:** The existing temp file pattern uses `$(date +%s%N)` which works on Linux but not macOS.
**How to avoid:** For unique ID generation, use `$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')` (epoch + random hex). This is already the pattern used for error/event IDs in the commands.
**Warning signs:** Temp files with literal `%N` in the name.

### Pitfall 6: `set -e` and jq Exit Codes
**What goes wrong:** jq exits non-zero on `null` output with `-e` flag, or on invalid input. Under `set -e`, this kills the script.
**Why it happens:** `set -euo pipefail` means any non-zero exit terminates the script immediately.
**How to avoid:** Always capture jq output in a variable with `|| json_err "message"` pattern. Do NOT pipe jq directly to atomic_write without error checking. Use `jq ... "$file"` (not `cat "$file" | jq`).
**Warning signs:** Script exits silently with no JSON error output.

### Pitfall 7: memory.json `patterns` Array
**What goes wrong:** Forgetting to validate the `patterns` array in memory.json schema.
**Why it happens:** The init.md schema shows `{"phase_learnings":[],"decisions":[],"patterns":[]}` -- three arrays. But the requirements only mention `phase_learnings` and `decisions` in the description.
**How to avoid:** The canonical schema has 3 arrays: `phase_learnings`, `decisions`, `patterns`. Validate all three. memory-compress should operate on `phase_learnings` (cap 20) and `decisions` (cap 30) independently.
**Warning signs:** validate-state memory fails because `patterns` array missing.

## Code Examples

### Example 1: Pheromone Decay (Pure Computation)
```bash
# Source: Verified with jq 1.8.1 on macOS -- exp() produces correct results
# Formula: strength * e^(-ln(2) * elapsed / half_life) = strength * 2^(-elapsed / half_life)
pheromone-decay)
  [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
  json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" \
    '{strength: (($s|tonumber) * ((-0.693147180559945 * ($e|tonumber) / ($h|tonumber)) | exp) | . * 1000000 | round / 1000000)}')"
  ;;
```
Test: `aether-utils pheromone-decay 1.0 3600 3600` -> `{"ok":true,"result":{"strength":0.5}}`

### Example 2: Pheromone Batch (File Read + Computed Fields)
```bash
# Source: Decay formula from status.md, build.md, continue.md (9 commands use it)
pheromone-batch)
  [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
  local now; now=$(date -u +%s)
  json_ok "$(jq --arg now "$now" '
    .signals | map(. + {
      current_strength: (
        if .half_life_seconds == null then .strength
        else .strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | fromdate)) / .half_life_seconds) | exp)
        end | . * 1000 | round / 1000)
    })' "$DATA_DIR/pheromones.json")" || json_err "Failed to read pheromones.json"
  ;;
```

### Example 3: State Validation (jq Type Checking)
```bash
# Source: Schema from init.md -- canonical field names and types
validate-state)
  case "${1:-}" in
    colony)
      [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
      json_ok "$(jq '
        def chk(f;t): if has(f) then if (.[f]|type) as $a | t | index($a) then "pass" else "fail: \(f) is \(.[f]|type), want \(t)" end else "fail: missing \(f)" end;
        {file: "COLONY_STATE.json", checks: [
          chk("goal";["null","string"]),
          chk("state";["string"]),
          chk("current_phase";["number"]),
          chk("workers";["object"]),
          chk("spawn_outcomes";["object"])
        ]} | . + {pass: (.checks | all(. == "pass"))}
      ' "$DATA_DIR/COLONY_STATE.json")"
      ;;
```

### Example 4: Error Add (Record Construction + Append)
```bash
# Source: Error schema from build.md lines 247-257
error-add)
  [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description>"
  [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
  local id; id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
  local ts; ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  local updated; updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --arg ts "$ts" '
    .errors += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:null, task_id:null, timestamp:$ts}] |
    if (.errors|length) > 50 then .errors = .errors[-50:] else . end
  ' "$DATA_DIR/errors.json") || json_err "Failed to update errors.json"
  atomic_write "$DATA_DIR/errors.json" "$updated"
  json_ok "\"$id\""
  ;;
```

### Example 5: Memory Token Count (Word Count * 1.3)
```bash
# Source: MEM-01 requirement -- word count * 1.3 approximation
memory-token-count)
  [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
  json_ok "$(jq '{tokens: ([.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor)}' "$DATA_DIR/memory.json")"
  ;;
```

### Example 6: Error Pattern Check (Group By + Filter)
```bash
# Source: ERR-02 requirement -- 3+ errors of same category triggers flag
error-pattern-check)
  [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
  json_ok "$(jq '
    .errors | group_by(.category) | map(select(length >= 3) |
      {category: .[0].category, count: length, first_seen: (sort_by(.timestamp) | first.timestamp), last_seen: (sort_by(.timestamp) | last.timestamp)})
  ' "$DATA_DIR/errors.json")"
  ;;
```

### Example 7: Pheromone Combine (Net Effect for Conflicting Signals)
```bash
# Source: Worker specs define combination as behavioral (FOCUS vs REDIRECT)
# Numeric computation: net_effect = max(0, signal1 - signal2)
# This represents the dominant signal's remaining influence after opposition
pheromone-combine)
  [[ $# -ge 2 ]] || json_err "Usage: pheromone-combine <signal1_strength> <signal2_strength>"
  json_ok "$(jq -n --arg s1 "$1" --arg s2 "$2" '{
    net_effect: ((($s1|tonumber) - ($s2|tonumber)) | if . < 0 then 0 else . end | . * 1000 | round / 1000),
    dominant: (if ($s1|tonumber) >= ($s2|tonumber) then "signal1" else "signal2" end),
    ratio: (if ($s2|tonumber) == 0 then null else (($s1|tonumber) / ($s2|tonumber)) | . * 1000 | round / 1000 end)
  }')"
  ;;
```

## Exact JSON Schemas (Canonical)

These are the schemas that validate-state must check. All verified from init.md and command files.

### COLONY_STATE.json
```json
{
  "goal": "<string|null>",
  "state": "<string: IDLE|READY|PLANNING|EXECUTING>",
  "current_phase": "<number>",
  "session_id": "<string|null>",
  "initialized_at": "<string|null>",
  "workers": {
    "<caste>": "<string: idle|active|error>"
  },
  "spawn_outcomes": {
    "<caste>": {
      "alpha": "<number>",
      "beta": "<number>",
      "total_spawns": "<number>",
      "successes": "<number>",
      "failures": "<number>"
    }
  }
}
```
Required top-level fields: `goal` (null|string), `state` (string), `current_phase` (number), `workers` (object), `spawn_outcomes` (object).
Optional: `session_id`, `initialized_at`.

### pheromones.json
```json
{
  "signals": [
    {
      "id": "<string>",
      "type": "<string: INIT|FOCUS|REDIRECT|FEEDBACK>",
      "content": "<string>",
      "strength": "<number: 0.0-1.0>",
      "half_life_seconds": "<number|null>",
      "created_at": "<string: ISO-8601>"
    }
  ]
}
```
Required: `signals` (array). Each signal requires: `id`, `type`, `content`, `strength`, `created_at`. `half_life_seconds` can be null.
Optional signal fields: `source`, `auto` (added by auto-emit in continue.md).

### errors.json
```json
{
  "errors": [
    {
      "id": "<string>",
      "category": "<string>",
      "severity": "<string: critical|high|medium|low>",
      "description": "<string>",
      "root_cause": "<string|null>",
      "phase": "<number|null>",
      "task_id": "<string|null>",
      "timestamp": "<string: ISO-8601>"
    }
  ],
  "flagged_patterns": [
    {
      "category": "<string>",
      "count": "<number>",
      "first_seen": "<string: ISO-8601>",
      "last_seen": "<string: ISO-8601>",
      "flagged_at": "<string: ISO-8601>",
      "description": "<string>"
    }
  ]
}
```
Required top-level: `errors` (array), `flagged_patterns` (array).
Retention limit: 50 errors max.
12 known categories: syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security.
4 severity levels: critical, high, medium, low.

### memory.json
```json
{
  "phase_learnings": [
    {
      "id": "<string>",
      "phase": "<number>",
      "phase_name": "<string>",
      "learnings": ["<string>"],
      "errors_encountered": "<number>",
      "timestamp": "<string: ISO-8601>"
    }
  ],
  "decisions": [
    {
      "id": "<string>",
      "type": "<string: focus|redirect|feedback>",
      "content": "<string>",
      "context": "<string>",
      "phase": "<number>",
      "timestamp": "<string: ISO-8601>"
    }
  ],
  "patterns": []
}
```
Required top-level: `phase_learnings` (array, cap 20), `decisions` (array, cap 30), `patterns` (array).

### events.json
```json
{
  "events": [
    {
      "id": "<string>",
      "type": "<string>",
      "source": "<string>",
      "content": "<string>",
      "timestamp": "<string: ISO-8601>"
    }
  ]
}
```
Required top-level: `events` (array). Retention limit: 100 events max.
Each event requires: `id`, `type`, `source`, `content`, `timestamp`.

## Pheromone Math: Definitive Formulas

### Decay Formula (PHER-01)
```
current_strength = strength * e^(-ln(2) * elapsed_seconds / half_life_seconds)
```
Where `ln(2) = 0.693147180559945`.

This is mathematically equivalent to `strength * 2^(-elapsed_seconds / half_life_seconds)`.

At exactly one half-life (`elapsed = half_life`), `current_strength = strength * 0.5`.

If `half_life_seconds` is null, the signal persists at original strength forever.

Verified in: status.md, build.md, plan.md, continue.md, pause-colony.md, resume-colony.md, colonize.md (9 command files use this exact formula).

### Effective Signal (PHER-02)
```
effective_signal = sensitivity * current_strength
```
Sensitivity values per caste (from worker specs, Phase 16 research):

| Caste | INIT | FOCUS | REDIRECT | FEEDBACK |
|-------|------|-------|----------|----------|
| colonizer | 1.0 | 0.7 | 0.3 | 0.5 |
| route-setter | 1.0 | 0.5 | 0.8 | 0.7 |
| builder | 0.5 | 0.9 | 0.9 | 0.7 |
| watcher | 0.3 | 0.8 | 0.5 | 0.9 |
| scout | 0.7 | 0.9 | 0.4 | 0.5 |
| architect | 0.2 | 0.4 | 0.3 | 0.6 |

Threshold (from Phase 16 research): effective > 0.5 = act, 0.3-0.5 = note, < 0.3 = ignore.

### Combine (PHER-05)
```
net_effect = max(0, signal1_strength - signal2_strength)
dominant = signal1 if signal1 >= signal2, else signal2
```
This represents opposing signals (e.g., FOCUS pulling toward an area while REDIRECT pushes away). The dominant signal wins, but its effective strength is reduced by the opposing signal.

### Cleanup Threshold (PHER-04)
Remove any signal where `current_strength < 0.05` (from continue.md Step 5).

## Line Budget Analysis

Total budget: 300 lines for `aether-utils.sh`.

| Section | Lines | Notes |
|---------|-------|-------|
| Existing scaffold (header, sourcing, helpers, dispatch) | 47 | Already written |
| Help text update (add new commands to listing) | 5 | Update help case |
| Pheromone module (5 subcommands) | 40 | decay:5, effective:4, batch:8, cleanup:10, combine:6 + case overhead |
| Validation module (6 subcommands + helper) | 60 | Helper:5, colony:10, pheromones:10, errors:10, memory:10, events:8, all:7 |
| Memory module (3 subcommands) | 30 | token-count:5, compress:15, search:8 |
| Error module (4 subcommands) | 50 | add:12, pattern-check:8, summary:8, dedup:15 |
| **Estimated total** | **232** | **68 lines of headroom** |

The budget is achievable with disciplined coding. Each subcommand averages ~8 lines.

## State of the Art

| Old Approach (v3 commands) | New Approach (v4 utilities) | Impact |
|----------------------------|-----------------------------|--------|
| LLM computes decay math in prompts | jq computes decay deterministically | Exact same formula, reproducible results |
| LLM validates JSON by reading it | jq type-checks against schema | Field-level error reporting |
| LLM estimates token counts | jq word-count * 1.3 | Consistent, not hallucinated |
| LLM manually counts error categories | jq group_by + count | Exact, no missed patterns |
| No dedup mechanism | jq timestamp comparison | Prevents duplicate error entries |

## Plan Grouping Recommendation

The 18 requirements naturally group into 4 plans matching the 4 modules:

### Plan 20-01: Pheromone Math (PHER-01 through PHER-05)
5 subcommands, all pure computation except batch/cleanup which read pheromones.json. No cross-module dependencies.
- pheromone-decay (pure math)
- pheromone-effective (pure math)
- pheromone-batch (reads pheromones.json)
- pheromone-cleanup (reads + writes pheromones.json via atomic_write)
- pheromone-combine (pure math)

### Plan 20-02: State Validation (VALID-01 through VALID-06)
6 subcommands, all read-only. Needs exact schemas documented above.
- validate-state colony
- validate-state pheromones
- validate-state errors
- validate-state memory
- validate-state events
- validate-state all (calls the above 5)

### Plan 20-03: Memory Operations (MEM-01 through MEM-03)
3 subcommands. Token counting is read-only, compress modifies file, search is read-only.
- memory-token-count (reads memory.json)
- memory-compress (reads + writes memory.json)
- memory-search (reads memory.json)

### Plan 20-04: Error Tracking (ERR-01 through ERR-04)
4 subcommands. All operate on errors.json.
- error-add (appends to errors.json)
- error-pattern-check (reads errors.json)
- error-summary (reads errors.json)
- error-dedup (reads + writes errors.json)

These 4 plans are independent -- they modify different files and can be developed in any order. However, validation (20-02) benefits from being second so it can verify pheromone math output schemas.

## Open Questions

1. **pheromone-combine semantics:** The requirement says "combination effect for conflicting signals" but doesn't specify the formula. The worker specs describe combination effects behaviorally (tables of signal pairs and behaviors), not numerically. I've recommended `max(0, s1 - s2)` as the numeric implementation (net effect after opposition), with dominant signal identification and ratio. This is a reasonable interpretation but could be revisited.
   - What we know: Worker specs show FOCUS + REDIRECT = conflict resolution behavior
   - What's unclear: Whether `pheromone-combine` should output a single number or a behavioral recommendation
   - Recommendation: Output a JSON object with `net_effect`, `dominant`, and `ratio` fields. The calling command can decide how to interpret.

2. **error-add category validation:** Success criteria uses `build` as a category, but the schema defines 12 specific categories that don't include `build`. Recommend accepting any string (no validation at add time) to match success criteria. Flag in this research for planner awareness.
   - Recommendation: Do not validate categories in error-add. Validate categories in validate-state errors as warnings, not errors.

3. **atomic_write with large JSON:** The existing `atomic_write` function passes content via `echo "$content"`. For large JSON (50 errors + patterns), shell expansion could be slow. Not a blocker for current scale but worth noting.
   - Recommendation: Use `atomic_write` as-is. If performance issues arise, switch to temp file + `atomic_write_from_file`.

4. **validate-state field-level errors:** Success criteria #2 says "specific field-level errors for any violations." The jq validation pattern returns per-field pass/fail. But how verbose should violations be? I've shown `"fail: field is type, want [types]"` format.
   - Recommendation: Return array of check results, each with field name and pass/fail string. Failed checks include the actual vs expected type.

## Sources

### Primary (HIGH confidence)
- jq 1.8.1 verified on local machine: `exp`, `fromdate`, `todate`, `now`, `group_by`, custom `def` all work
- All 13 command .md files in `.claude/commands/ant/` read for schema and formula verification
- `.aether/data/COLONY_STATE.json` and `.aether/data/pheromones.json` read for current schemas
- `.planning/REQUIREMENTS.md` read for all 18 requirements
- `.planning/ROADMAP.md` read for success criteria
- `.planning/phases/19-audit-fixes-utility-scaffold/19-RESEARCH.md` read for existing infrastructure
- `.aether/aether-utils.sh` read for current scaffold (47 lines)
- `.aether/utils/atomic-write.sh` and `.aether/utils/file-lock.sh` read for shared infrastructure
- 6 worker spec files in `.aether/workers/` read for pheromone sensitivity values and combination effects
- Phase 16 research read for sensitivity table and thresholds
- Prototype testing of jq expressions in bash (all examples verified locally)

### Secondary (MEDIUM confidence)
- Line budget estimates based on prototype implementations (tested but not final)
- pheromone-combine formula (`max(0, s1-s2)`) is a reasonable interpretation but not explicitly specified

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- jq 1.8.1 verified locally, all functions tested
- Architecture: HIGH -- single-file pattern established by Phase 19, all patterns prototyped
- Schemas: HIGH -- all 5 JSON schemas verified from init.md and command files
- Formulas: HIGH -- decay formula verified across 9 command files, tested in jq
- Pitfalls: HIGH -- all pitfalls discovered through actual prototype testing
- Line budget: MEDIUM -- estimates based on prototypes, final count may vary +/- 20 lines
- pheromone-combine: MEDIUM -- formula is interpretation of behavioral specs, not explicitly defined

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable codebase, jq version unlikely to change)
