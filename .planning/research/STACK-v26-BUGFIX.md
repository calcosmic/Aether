# Stack Research: v2.6 Bugfix & Hardening

**Domain:** Bash/jq shell script hardening -- no new dependencies
**Researched:** 2026-03-29
**Confidence:** HIGH (grounded in direct codebase inspection of ~5,200 lines aether-utils.sh, 10 domain modules, 616+ tests, and shellcheck 0.11.0 analysis)

---

## Executive Summary

This is a hardening milestone, not a stack expansion. The research covers three bug categories identified in PROJECT.md v2.6 scope: (1) unescaped variables in grep patterns and JSON output, (2) cross-colony state isolation via LOCK_DIR mutation, and (3) JSON escaping gaps in helper functions. No new dependencies are needed. The fixes require adopting established bash idioms (`grep -F --`, `jq --arg`), hardening the existing `json_ok`/`json_err` helpers, and refactoring `LOCK_DIR` from a mutable global into a function-parameter pattern.

The existing codebase already has good patterns in places -- `grep -q --` with `--` separator is used correctly in signature-check and antipattern-check (lines 1859, 1949). But these patterns are inconsistent: `grep "$caste_filter"` on line 1606 passes a user variable as a regex without escaping, and `json_ok "{\"updated\":true,\"ant\":\"$ant_name\"}"` on line 570 injects `$ant_name` directly into a JSON string without quoting. The fix is standardization, not invention.

---

## Recommended Patterns (No New Dependencies)

### 1. Grep Safety: Three-Tier Escaping Strategy

The codebase uses `grep` in three distinct contexts, each requiring different escaping:

| Context | Current Pattern | Fix | Applies To |
|---------|----------------|-----|-----------|
| **Literal string search** (user-provided text) | `grep "$var" file` | `grep -F -- "$var" file` | ant_name, caste_filter, file names |
| **Regex search** (code-controlled patterns) | `grep -E "$pattern" file` | Keep as-is (pattern is code, not user input) | antipattern-check, signature-check regex patterns |
| **Existence test** (known literal) | `grep -q "string" file` | `grep -qF -- "string" file` (minor hardening) | changelog headers, known markers |

**The key distinction:** If the variable comes from user input, colony state, or an ant name, it MUST use `grep -F`. If the pattern is a code-defined regex (like `'^## \[.*\]'` on line 682), `grep -E` is fine.

**Specific fixes needed:**

| File | Line | Current | Fix |
|------|------|---------|-----|
| aether-utils.sh | 1606 | `grep "$caste_filter" "$log_file"` | `grep -F -- "$caste_filter" "$log_file"` |
| aether-utils.sh | 703 | `grep -q "^## ${date_str}$"` | `grep -qF -- "## ${date_str}"` (date_str is YYYY-MM-DD, no regex chars, but -F is defensive) |
| suggest.sh | 84 | `grep -qE "$exclude_pattern"` | Keep (pattern is code-controlled regex) |
| suggest.sh | 109, 128, 146, 165 | Various `grep` with file names | `grep -F --` where variable holds file name |
| pheromone-xml.sh | 280 | `grep '<signal' "$xml_file"` | Keep (pattern is code-controlled literal) |
| learning.sh | 1253 | `grep -q "${escaped_content}"` | `grep -qF -- "$unescaped_content"` (the variable is already escaped upstream -- double-escaping is a bug risk) |

**Why `grep -F` and not regex escaping:**
- `grep -F` (a.k.a. `fgrep`) treats the pattern as a literal fixed string. No escaping needed.
- `printf '%q'` escapes for the shell, NOT for regex. Using it before `grep` is wrong -- it would add backslashes that `grep -F` would then treat as literal characters.
- A `sed`-based regex escaper (`sed 's/[.[\*^$()+?{|\\]/\\&/g'`) is fragile and unnecessary when `-F` is available.
- `--` separator prevents variable values starting with `-` from being interpreted as grep flags.

### 2. JSON Escaping: Adopt `jq --arg` Consistently

The codebase has two JSON output patterns, one safe and one unsafe:

**Unsafe pattern (string interpolation into JSON):**
```bash
# Line 570 -- $ant_name could contain quotes, backslashes, newlines
json_ok "{\"updated\":true,\"action\":\"worker-spawn\",\"ant\":\"$ant_name\"}"
```

**Safe pattern (jq --arg for variable passing):**
```bash
# Line 1296 -- $srf_raw is safely passed through jq's escaping
json_ok "$(jq -n --arg v "$srf_raw" '$v')"
```

**The fix:** Replace all `json_ok "{...\"$var\"...}"` with `json_ok "$(jq -n --arg name "$var" '{...name: $name...}')"` for string values, or use `--argjson` only for variables that already contain valid JSON (like `$combined`, `$crit_json`).

**Specific high-risk instances (user-derived strings):**

| Line | Variable | Risk | Fix |
|------|----------|------|-----|
| 570 | `$ant_name` | HIGH -- ant names can contain special chars | `jq -n --arg ant "$ant_name"` |
| 581 | `$ant_name` | HIGH -- same as above | `jq -n --arg ant "$ant_name"` |
| 413 | `$ctx_file` | MEDIUM -- file paths with spaces/quotes | `jq -n --arg file "$ctx_file"` |
| 456 | `$cmd` | MEDIUM -- command names | `jq -n --arg cmd "$cmd"` |
| 468 | `$safe` | LOW -- boolean-ish, but still | `jq -n --arg safe "$safe"` |
| 619 | `$status` | LOW -- known enum values | `jq -n --arg status "$status"` |
| 796 | `$date_str`, `$phase`, `$plan` | LOW -- date/numbers, but pattern is fragile | `jq -n --arg date "$date_str" --arg phase "$phase" --arg plan "$plan"` |
| 2262, 2284, 2298 | `$message`, `$body` | HIGH -- free-form text from builders | `jq -n --arg msg "$message" --arg body "$body"` |
| 2664 | `$model`, `$caste` | MEDIUM -- could contain unexpected chars | `jq -n --arg model "$model" --arg caste "$caste"` |
| 2697 | `$proxy_status` | LOW -- known enum | `jq -n --arg status "$proxy_status"` |

**Low-risk instances (can defer):** Lines where the variable is a jq-produced JSON value (like `$combined`, `$crit_json`, `$all_pass`) -- these are already valid JSON and correctly passed as-is. Lines where the variable is a controlled enum or number.

**The `error-handler.sh` escaping fix (lines 74-75, 107):**

Current:
```bash
escaped_message=$(echo "$message" | sed 's/"/\\"/g' | tr '\n' ' ')
```

This is incomplete -- it does not escape backslashes (so `\"` becomes `\\"` which is wrong), tabs, control characters, or unicode. The fix:

```bash
escaped_message=$(printf '%s' "$message" | jq -Rs '.')  # jq handles ALL JSON escaping
```

Or better yet, refactor `json_err` and `json_warn` to use `jq -n` internally:

```bash
json_err() {
  local code="${1:-$E_UNKNOWN}"
  local message="${2:-An unknown error occurred}"
  local details="${3:-null}"
  local recovery="${4:-}"
  local timestamp
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  if [[ -z "$recovery" ]]; then
    recovery=$(_get_recovery "$code")
  fi

  # Use jq for correct JSON escaping of all fields
  jq -n --arg code "$code" --arg message "$message" \
    --argjson details "$details" --argjson recovery "$recovery" --arg ts "$timestamp" \
    '{"ok":false,"error":{"code":$code,"message":$message,"details":$details,"recovery":$recovery,"timestamp":$ts}}' >&2
}
```

### 3. Cross-Colony State Isolation: LOCK_DIR Mutation

**The problem:** `file-lock.sh` sets `LOCK_DIR` as a global variable (line 20). `hive.sh` temporarily mutates this global to lock `~/.aether/hive/wisdom.json` (lines 47, 134, 328). If a colony is running concurrently with a hive operation, and a colony operation calls `acquire_lock` between the mutation and the restore, the colony lock file is created in `~/.aether/hive/` instead of `$AETHER_ROOT/.aether/locks/`.

**Current mitigation:** hive.sh saves and restores `LOCK_DIR`:
```bash
hv_saved_lock_dir="$LOCK_DIR"
LOCK_DIR="$hv_hive_dir"
acquire_lock "$hs_wisdom_file" || { LOCK_DIR="$hv_saved_lock_dir"; json_err ... }
# ... later ...
LOCK_DIR="${hv_saved_lock_dir:-$LOCK_DIR}"
```

**Why this is fragile:**
1. If `acquire_lock` succeeds and then any code path between success and the `release_lock` call throws an error (triggering `trap ERR`), the LOCK_DIR restore may be skipped. The `trap ERR` handler calls `error_handler`, which does NOT restore LOCK_DIR.
2. In bash 3.2, there are no `local` variables for lock state. The save/restore pattern uses 3 separate copies (`hv_saved_lock_dir`, `hs_saved_lock_dir`, `hr_saved_lock_dir`) because each function needs its own.
3. If a future developer adds a new function that mutates LOCK_DIR, they must remember the save/restore pattern.

**Recommended fix: Add a `--lock-dir` parameter to `acquire_lock`/`release_lock`**

```bash
acquire_lock() {
    local file_path="$1"
    local lock_dir_override="${2:-}"  # NEW: optional lock directory override
    local effective_lock_dir="${lock_dir_override:-$LOCK_DIR}"
    local lock_file="${effective_lock_dir}/$(basename "$file_path").lock"
    # ... rest of function uses $effective_lock_dir ...
}
```

This way, hive.sh calls `acquire_lock "$wisdom_file" "$HOME/.aether/hive"` instead of mutating a global. No save/restore needed. The global `LOCK_DIR` remains untouched.

**Alternative (simpler, less disruptive):** Keep the save/restore pattern but add it to the `trap ERR` cleanup. The `error_handler` function in `error-handler.sh` could restore `LOCK_DIR` from a `_SAVED_LOCK_DIR` global. However, this is more invasive and fragile than the parameter approach.

### 4. Shellcheck Integration

ShellCheck 0.11.0 is already installed. Current lint config (package.json line 32):

```json
"lint:shell": "shellcheck --severity=error .aether/aether-utils.sh bin/generate-commands.sh .aether/utils/file-lock.sh .aether/utils/atomic-write.sh .aether/utils/colorize-log.sh .aether/utils/watch-spawn-tree.sh"
```

**Gaps:**
1. Only 6 files are linted. The utils/ directory has 29+ scripts but only 3 are in the lint target.
2. `--severity=error` catches only the worst issues. The unescaped variable problems are mostly `warning` level (SC2086: "Double quote to prevent globbing and word splitting").
3. No `.shellcheckrc` file exists to configure project-wide settings.

**Recommended shellcheck rules for this milestone:**

| Rule | Severity | Description | Relevant Lines |
|------|----------|-------------|----------------|
| SC2086 | warning | Double quote to prevent globbing/word splitting | grep with unquoted vars |
| SC2061 | warning | Quote the grep pattern parameter | grep "$var" without -F |
| SC2295 | info | Expansions inside ${} can't be checked | `grep "${date_str}"` style |
| SC2312 | info | Consider invoking this command separately | complex pipes with grep |

**Recommended approach:**
1. Add a `.shellcheckrc` with severity level set to `warning` for the utils/ directory
2. Add all `.sh` files in `.aether/` to the lint target
3. Add `# shellcheck disable=SCXXXX` annotations for intentional patterns (many already exist as `# SUPPRESS:OK`)
4. Fix all genuine SC2086/SC2061 findings rather than suppressing them

---

## Existing Codebase Patterns to Standardize

The codebase already has some correct patterns that should be the standard:

### Good patterns already in use:
```bash
# grep -F with -- separator (signature-check, line 1859)
grep -q -- "$pattern_string" "$target_file"

# jq -Rs for safe string-to-JSON conversion (activity-read, line 1610)
json_ok "$(echo "$content" | jq -Rs '.')"

# jq --arg for safe variable passing (state-save, line 1344)
echo "$vc_mm_arr" | jq --arg f "$vc_mm_f" '. + [{"file":$f}]'

# @base64 for safe iteration of complex objects (queen.sh line 737)
jq -r '.[] | @base64'
```

### Anti-patterns to eliminate:
```bash
# String interpolation into JSON (used ~25 times in aether-utils.sh)
json_ok "{\"ant\":\"$ant_name\"}"       # BROKEN if ant_name has quotes
# Replace with:
json_ok "$(jq -n --arg ant "$ant_name" '{"ant":$ant}')"

# User variable as grep regex (line 1606)
grep "$caste_filter" "$log_file"        # BROKEN if caste_filter has regex chars
# Replace with:
grep -F -- "$caste_filter" "$log_file"

# Incomplete JSON escaping via sed (error-handler.sh line 75)
escaped_message=$(echo "$message" | sed 's/"/\\"/g' | tr '\n' ' ')
# Replace with:
escaped_message=$(printf '%s' "$message" | jq -Rs '.')
```

---

## What NOT to Do

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `printf '%q'` before `grep` | Escapes for the SHELL, not for regex. The escaped backslashes become literal characters in grep. | `grep -F --` (fixed string, no escaping needed) |
| Custom `escape_regex()` function via sed | Fragile, does not handle all edge cases, adds maintenance burden | `grep -F --` (eliminates the need entirely) |
| `--argjson` with user input | Parses the value as JSON, so malformed input causes errors and crafted input could inject JSON structures | `--arg` (always treats value as a string) |
| Adding a `json_escape()` bash function | Reinventing what jq already does perfectly | `jq -Rs '.'` or `jq -n --arg var "$value"` |
| Changing `json_ok` signature | Too many call sites (~40+). Breaking the API for all callers. | Keep `json_ok` as-is, fix callers to pass pre-escaped values |
| Replacing all `grep -E` with `grep -F` | Many patterns are intentionally regex (antipattern detection, signature matching) | Only add `-F` where the variable is user-derived |

---

## Installation

No new dependencies. All fixes use existing tools:

```bash
# Verify available versions (already present):
bash --version    # 3.2+ (macOS default)
jq --version      # 1.6+ (currently 1.8.1)
shellcheck --version  # 0.10+ (currently 0.11.0)

# No npm installs needed
# No new bash scripts needed
```

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `grep -F --` | Custom regex escaping function | `-F` is POSIX-standard, available everywhere, zero maintenance. Custom escaping is fragile and error-prone. |
| `jq --arg` for all string values | Keep string interpolation with `sed` escaping | `sed 's/"/\\"/g'` misses backslashes, tabs, control chars, unicode. `jq --arg` handles everything correctly. |
| `acquire_lock` with `--lock-dir` parameter | Keep global `LOCK_DIR` mutation with save/restore | The save/restore pattern has 3 copies in hive.sh already, is fragile under `trap ERR`, and is easy to forget in new code. A parameter is cleaner. |
| `.shellcheckrc` config file | Inline `# shellcheck` directives only | A config file applies to all files uniformly. Inline directives are fine for specific exceptions but cannot set the base severity level. |
| Severity `warning` for shellcheck | Keep `severity=error` | The unescaped variable bugs are at `warning` level. `error`-only would miss them entirely. |

---

## Stack Patterns by Variant

**If the variable is a controlled enum (phase numbers, known actions, etc.):**
- String interpolation into JSON is acceptable: `json_ok "{\"phase\":$phase}"`
- `grep -E` with known patterns is acceptable: `grep -qE '^## \[.*\]'`
- These are code-controlled values with no user input

**If the variable is user-derived (ant names, messages, file paths, search terms):**
- ALWAYS use `jq --arg` for JSON output
- ALWAYS use `grep -F --` for grep patterns
- These can contain any character

**If the variable is jq-produced JSON (combined results, parsed state, etc.):**
- Pass directly to `json_ok` as-is: `json_ok "{\"pass\":$all_pass}"`
- Use `--argjson` when passing to a `jq` invocation that expects JSON
- These are already valid JSON by construction

---

## Version Compatibility

| Tool | Available Version | Minimum Required | Notes |
|------|-------------------|-----------------|-------|
| bash | 3.2.57 (macOS) | 3.2+ | `printf %q` works in 3.2. `grep -F` and `--` are POSIX. |
| jq | 1.8.1 | 1.6+ | `--arg` available since jq 1.5. `@base64`, `@json` available since 1.5. |
| shellcheck | 0.11.0 | 0.10+ | SC2086, SC2061 available in all modern versions. |
| grep | macOS BSD grep | POSIX | `-F` flag is POSIX-standard. `--` separator is POSIX. |

**bash 3.2 caveat:** `jq --arg` works fine because it does not depend on bash features -- it is a jq feature invoked as an external command. The `local` keyword is available in bash 3.2 (it is not POSIX sh but is in bash 3.2+).

---

## Sources

- HIGH: Direct codebase inspection of `aether-utils.sh` (5,200+ lines, 40+ json_ok call sites, 30+ grep instances)
- HIGH: Direct codebase inspection of `error-handler.sh` (json_err/json_warn escaping on lines 56-117)
- HIGH: Direct codebase inspection of `file-lock.sh` (LOCK_DIR global pattern, 192 lines)
- HIGH: Direct codebase inspection of `hive.sh` (LOCK_DIR mutation pattern, 3 save/restore instances)
- HIGH: Direct codebase inspection of `suggest.sh`, `learning.sh`, `pheromone-xml.sh`, `spawn.sh` (grep patterns)
- HIGH: `package.json` line 32 (current shellcheck configuration)
- HIGH: Verified tool versions on the target machine: bash 3.2.57, jq 1.8.1, shellcheck 0.11.0
- MEDIUM: ShellCheck documentation on SC2086 (double quote to prevent globbing) and SC2061 (quote grep pattern) -- from training data, not verified with live docs (web search rate-limited)
- MEDIUM: `jq --arg` safety properties -- from training data, well-established behavior
- HIGH: `grep -F` POSIX specification -- standard POSIX behavior, verified working on macOS

---
*Stack research for: Aether v2.6 Bugfix & Hardening*
*Researched: 2026-03-29*
