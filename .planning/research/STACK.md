# Stack Research: Aether Production Hardening

**Domain:** Multi-agent CLI orchestration / bash+Node.js production hardening
**Researched:** 2026-03-23
**Confidence:** HIGH (recommendations derived from codebase analysis, Oracle audit findings, and verified production patterns)

## Context: What Exists and What This Research Covers

Aether is a shipped v2.0.0 product distributed via npm. The Oracle audit surfaced concrete issues requiring production hardening. This STACK.md covers **patterns, approaches, and tooling changes** -- not new libraries. The fundamental stack (bash + jq + Node.js + JSON files) stays.

| Layer | Current | Status | This Research |
|-------|---------|--------|---------------|
| Core logic | `aether-utils.sh` (11,272 lines, 178 subcommands) | Staying | Modularization patterns, error handling triage |
| Utility scripts | 22 scripts in `.aether/utils/` (5,237 lines) | Staying | Extraction targets from monolith |
| CLI entry | Node.js + Commander.js v12 | Staying | Process hardening, exit code discipline |
| JSON processing | `jq` (system dependency, 619 invocations) | Staying | Type coercion patterns, validation gates |
| File safety | `file-lock.sh` + `atomic-write.sh` | Staying | Temp file uniqueness fix, lock scope audit |
| Testing (JS) | AVA v6 + sinon + proxyquire | Staying | Coverage expansion patterns |
| Testing (bash) | Custom test harness (41 test files) | Staying | Structured assertion patterns |
| State storage | JSON files (COLONY_STATE.json, pheromones.json, etc.) | Staying | Backup/recovery patterns, schema validation |
| Distribution | npm package with postinstall | Staying | Package validation hardening |
| Linting | ShellCheck (severity=error) | Staying | Severity escalation, `.shellcheckrc` addition |

**The job is hardening what exists, not replacing it.**

---

## Recommended Stack Patterns (No New Dependencies)

The Oracle audit found 338 error-swallowing patterns, 76 dead subcommands, state desync risks, and memory pipeline fragility. Each recommendation below is a **pattern or practice change**, not a library addition.

### 1. Error Handling Triage Pattern

**Problem:** 338 error-suppression instances (`2>/dev/null || true`, `2>/dev/null || echo ""`, etc.) across `aether-utils.sh`. Not all are bugs -- some are correct fallback behavior. The danger is that correct and incorrect suppressions are indistinguishable.

**Pattern: Three-category error triage**

| Category | Marker | Action | Example |
|----------|--------|--------|---------|
| **Intentional** | `# SUPPRESS:OK -- <reason>` | Keep, document | `mkdir -p "$dir" 2>/dev/null || true  # SUPPRESS:OK -- dir may already exist` |
| **Lazy** | No marker = suspect | Replace with proper error handling | `jq '.foo' "$file" 2>/dev/null || echo ""` --> `jq '.foo' "$file" || json_err "$E_JSON_INVALID" "Failed to parse $file"` |
| **Dangerous** | On state-mutation paths | Critical fix -- add error propagation | `atomic_write ... 2>/dev/null || true` --> `atomic_write ... || { json_err "$E_BASH_ERROR" "State write failed for $file"; }` |

**Why this pattern:** Attempting to fix all 338 at once is impractical and risky. Triaging by category lets you fix dangerous ones first (state mutations, ~40 instances), lazy ones second (JSON parsing, ~200 instances), and mark intentional ones as reviewed (~100 instances). The comment markers create a grep-auditable record.

**Priority targets (from Oracle audit):**
1. `suggest-analyze` ERR trap gap (lines 10236-10427) -- 200 lines with error trapping disabled
2. Any suppression on `atomic_write` or `COLONY_STATE.json` mutation calls
3. `memory-capture` caller sites (build-wave.md line 383, build-verify.md lines 330/346/386, build-complete.md line 58, continue-advance.md line 66)
4. Conditional module sourcing pattern (`[[ -f ]] && source`) -- add fallback error if required module missing

**Confidence:** HIGH -- based on direct codebase analysis of 418 `2>/dev/null` instances and 104 `|| true` instances.

### 2. Bash Monolith Modularization Pattern

**Problem:** `aether-utils.sh` is 11,272 lines with 178 subcommands in a single case statement. 76 subcommands (43%) are dead code. The file is sourced entirely on every invocation, including all 9 utility modules.

**Pattern: Domain-grouped extraction following existing precedent**

The codebase already has a proven modularization pattern: `hive.sh` (561 lines), `skills.sh` (502 lines), and `midden.sh` (260 lines) were extracted from the monolith into `utils/` and sourced at startup. Continue this pattern for the next tier of extractable domains.

| Extraction Target | Subcommand Count | Est. Lines | Priority | Rationale |
|-------------------|-----------------|------------|----------|-----------|
| **Dead code removal** | 76 subcommands | ~2,000-2,500 | P0 | 43% of subcommands never called. Removing them shrinks the file by ~20% with zero functional impact. |
| `pheromone.sh` | ~15 subcommands | ~800 | P1 | Pheromone read/write/display/prime/expire/export are a cohesive domain. Already has lock discipline. |
| `colony-prime.sh` | ~5 subcommands | ~600 | P1 | Context assembly, trimming, budget enforcement. Most complex read path in the system. |
| `learning.sh` | ~10 subcommands | ~500 | P2 | Learning observe/promote/inject/memory-capture. The memory pipeline (Oracle Rec 8). |
| `spawn.sh` | ~8 subcommands | ~400 | P2 | Worker spawning, depth tracking, completion logging. |
| `flag.sh` | ~6 subcommands | ~300 | P3 | Flag add/resolve/check/list. Simple extract. |

**Extraction protocol:**
1. Move functions to `utils/<name>.sh`
2. Source from `aether-utils.sh` using existing `[[ -f ]] && source` pattern (but add fallback error for required modules -- see Pattern 1)
3. Keep case-statement dispatch in `aether-utils.sh` (maintains single entry point)
4. Verify all existing tests pass after extraction (no behavioral change)
5. Add a sourcing verification function that checks all required modules loaded

**What NOT to do:**
- Do NOT break the single-entry-point pattern (`bash .aether/aether-utils.sh <subcommand>`). Every slash command and playbook depends on this interface. Changing it would cascade across 43+ command files.
- Do NOT use dynamic dispatch or autoloading. The Oracle audit confirmed the case-statement dispatch is safe and all callers are statically determinable via grep.
- Do NOT extract into separate scripts that are called as subprocesses instead of sourced. The startup cost of re-parsing 5K+ lines of sourced utilities would become per-invocation overhead instead of once-per-session.

**Confidence:** HIGH -- extraction pattern proven by existing `hive.sh`, `skills.sh`, `midden.sh` extractions. Dead code identified by Oracle audit cross-referencing all command/playbook callers.

### 3. State Protection Pattern

**Problem:** COLONY_STATE.json has 219 references across 38 of 43 slash commands, dual access paths (jq + subcommands), and no backup before mutation. A mid-autopilot corruption causes total loss.

**Pattern: Checkpoint-before-mutate with bounded rotation**

```bash
# In any function that mutates COLONY_STATE.json:
_checkpoint_state() {
    local phase="${1:-unknown}"
    local checkpoint_dir="$DATA_DIR/checkpoints"
    mkdir -p "$checkpoint_dir"

    local state_file="$DATA_DIR/COLONY_STATE.json"
    [[ -f "$state_file" ]] || return 0

    # Atomic copy with phase label
    cp "$state_file" "$checkpoint_dir/COLONY_STATE.phase-${phase}.bak"

    # Rotate: keep last 3 checkpoints
    ls -t "$checkpoint_dir"/COLONY_STATE.phase-*.bak 2>/dev/null \
        | tail -n +4 \
        | xargs rm -f 2>/dev/null || true  # SUPPRESS:OK -- cleanup is best-effort
}
```

**Where to call it:**
- `build-prep` (before state -> "EXECUTING")
- `continue-advance` (before phase advancement)
- `autopilot-update` (before each auto-advance)

**Why 3 checkpoints:** Balances disk usage (~30KB per checkpoint) against rollback depth. Three phases of history covers the "subtly bad work" detection window identified in the Oracle audit.

**Related: Git checkpoint tags for autopilot**

Before each autopilot build phase, create a lightweight git tag: `aether/pre-phase-N`. Combined with state checkpoints, this provides full code + state rollback. The existing `autofix-checkpoint` subcommand (line 2409) already implements git stashing -- extend it.

**Confidence:** HIGH -- directly implements Oracle Rec 1 and Rec 7. Feasibility confirmed by audit: build-prep already acquires lock and writes state.

### 4. Memory Pipeline Circuit Breaker Pattern

**Problem:** The `memory-capture` pipeline is a 5-step sequential chain where step 1 failure (corrupted `learning-observations.json`) kills all downstream steps silently. Callers wrap with `2>/dev/null || true`, making the pipeline death invisible.

**Pattern: Detect-recover-log at step 1 boundary**

```bash
# In learning-observe (step 1 of memory-capture):
# Replace: jq validation that exits on failure
# With: validation + recovery + audit trail

_validate_or_reset_observations() {
    local obs_file="$1"
    local template="$SCRIPT_DIR/templates/learning-observations.template.json"

    if [[ ! -f "$obs_file" ]]; then
        cp "$template" "$obs_file"
        return 0
    fi

    if ! jq -e . "$obs_file" > /dev/null 2>&1; then
        # Log corruption event to midden before recovery
        local timestamp
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        _midden_write_entry "memory" \
            "learning-observations.json corrupted — reset from template" \
            "medium" "$timestamp" 2>/dev/null || true  # SUPPRESS:OK -- midden is best-effort

        # Recover from template
        cp "$template" "$obs_file"
    fi
}
```

**Why this specific pattern:** The Oracle audit identified this as the only "detection without remediation" gap in the system. The template file already exists (`learning-observations.template.json`). The reset preserves existing behavior (fresh observations file) while adding an audit trail (midden entry) and preventing permanent pipeline death.

**Do NOT:**
- Make memory-capture callers propagate errors (would cascade across 6+ playbook sites and risk halting builds on non-critical memory failures)
- Remove the `2>/dev/null || true` from callers (memory capture should remain non-blocking for builds)
- Add retry logic (if the file is corrupted, retrying won't help -- reset is the correct action)

**Confidence:** HIGH -- directly implements Oracle Rec 8. The ~15-line fix transforms a permanent silent failure into a recoverable event.

### 5. jq Type Safety Pattern

**Problem:** String-typed confidence values (`"0.8"` vs `0.8`) silently exclude wisdom from `hive-read` results. Confirmed bug via REDIRECT signal and midden evidence.

**Pattern: Defensive type coercion at all numeric comparison boundaries**

```bash
# Before (fails silently on string-typed numbers):
jq --argjson min_conf "$threshold" \
    '[.[] | select(.confidence >= $min_conf)]'

# After (handles both string and number types):
jq --argjson min_conf "$threshold" \
    '[.[] | select((.confidence | tonumber? // 0) >= $min_conf)]'
```

**Audit scope:** Apply `tonumber? // <default>` at all `--argjson` comparison sites. The codebase has 619 jq invocations -- focus on comparison operations (`>=`, `<=`, `>`, `<`, `==` with numeric args).

**Why `tonumber?` not `tonumber`:** The `?` operator suppresses errors from non-numeric values (e.g., `null`). The `// 0` provides a safe default. Without the `?`, a `null` confidence value would halt the entire jq pipeline.

**Confidence:** HIGH -- confirmed 5-line fix. Oracle Rec 4 with exact line identification.

### 6. Midden Temp File Race Fix

**Problem:** Concurrent lockless midden writes both use `$mw_midden_file.tmp` as the temp path. Two simultaneous failures cause data loss.

**Pattern: PID-scoped temp files (already used in atomic-write.sh)**

```bash
# Before (shared temp path):
jq ... > "$mw_midden_file.tmp" && mv "$mw_midden_file.tmp" "$mw_midden_file"

# After (PID-scoped temp path, matching atomic-write.sh pattern):
local _mw_tmp="${mw_midden_file}.${$}.$(date +%s%N).tmp"
jq ... > "$_mw_tmp" && mv "$_mw_tmp" "$mw_midden_file"
rm -f "$_mw_tmp" 2>/dev/null || true  # SUPPRESS:OK -- cleanup on mv failure
```

**Why:** The `atomic-write.sh` module already uses `$$` (PID) + timestamp + `$RANDOM` for temp file uniqueness. The midden lockless fallback path should follow the same pattern. This eliminates the race condition without changing the lock/fallback architecture.

**Confidence:** HIGH -- directly from Oracle Q2 finding 7. Pattern already proven in `atomic-write.sh` line 58.

### 7. ShellCheck Severity Escalation

**Problem:** Current ShellCheck runs at `--severity=error` only, catching 4 of the 12 severity levels. Warnings and info-level issues (unquoted variables, useless cat, incorrect test syntax) pass silently.

**Pattern: Graduated severity with `.shellcheckrc` for project-wide configuration**

Create `.shellcheckrc` at repo root:
```
# Aether ShellCheck configuration
# Target severity: warning (catches unquoted variables, incorrect tests)
# Plan: escalate to info in a future phase after warning-level fixes

# Disabled checks with rationale:
disable=SC2034   # Unused variables -- false positives on exported functions
disable=SC2155   # Declare and assign separately -- too noisy for existing code
disable=SC1091   # Not following sourced files -- we use dynamic paths
disable=SC2086   # Double-quote to prevent globbing -- audit separately (high volume)
```

**Escalation plan:**
1. **Now:** Add `.shellcheckrc`, keep `--severity=error` in CI
2. **Phase 1:** Fix all `--severity=warning` issues (estimated ~50-80 instances based on typical bash codebases of this size)
3. **Phase 2:** Escalate CI to `--severity=warning`
4. **Phase 3:** Fix `--severity=info` issues for newly written code only

**Expand lint scope:** Current `lint:shell` only checks 6 specific files. Expand to all `.sh` files:
```json
"lint:shell": "shellcheck --severity=error .aether/aether-utils.sh .aether/utils/*.sh bin/*.sh"
```

**Confidence:** MEDIUM -- severity levels and `.shellcheckrc` syntax verified via ShellCheck documentation. Specific disable codes based on common patterns in this codebase, but exact warning counts need validation.

### 8. Test Reliability Patterns

**Problem:** 41 bash test files use a custom harness. The harness works but lacks test isolation -- tests share temp directories, can leave artifacts, and have no setup/teardown lifecycle.

**Pattern: Strengthen the existing harness rather than migrate**

The custom test harness (`test-helpers.sh`) already provides `setup_isolated_env`, `assert_json_valid`, `assert_json_has_field`, and proper exit code reporting. Migrating to bats-core would require rewriting 41 test files for marginal benefit.

**Specific improvements:**

a) **Isolated temp dirs per test (not per file):**
```bash
# Add to test-helpers.sh:
test_setup() {
    TEST_TMPDIR=$(mktemp -d)
    export AETHER_ROOT="$TEST_TMPDIR"
    export DATA_DIR="$TEST_TMPDIR/.aether/data"
    mkdir -p "$DATA_DIR"
}

test_teardown() {
    [[ -d "${TEST_TMPDIR:-}" ]] && rm -rf "$TEST_TMPDIR"
    unset AETHER_ROOT DATA_DIR TEST_TMPDIR
}
```

b) **Timeout enforcement:**
```bash
# Add to test-helpers.sh:
run_with_timeout() {
    local timeout="${1:-30}"
    shift
    timeout "$timeout" "$@" || {
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            test_fail "completed within ${timeout}s" "timed out"
        fi
        return $exit_code
    }
}
```

c) **Parallel-safe test execution:** Ensure tests can run in parallel by eliminating shared state between test files (no shared temp dirs, no hardcoded port numbers, no global state mutations).

**For Node.js (AVA) tests:** AVA already runs tests in parallel with isolated workers. Focus on:
- Mocking filesystem operations to avoid cross-test contamination
- Using `t.teardown()` for cleanup in every test that creates temp files
- Adding timeout per test (already configured at 30s in package.json)

**Confidence:** HIGH -- based on direct analysis of `test-helpers.sh` and the existing test patterns.

### 9. Package Distribution Hardening

**Problem:** npm's `postinstall` hook runs `node bin/cli.js install --quiet`. If this fails, the package appears installed but the hub is not set up. Users encounter confusing errors later.

**Pattern: Validate postinstall success with diagnostic output on failure**

The existing `validate-package.sh` checks file presence before publishing. Add a corresponding post-install validation:

```javascript
// In bin/cli.js install command:
// After hub setup, verify critical files exist:
const criticalFiles = [
    path.join(HUB_DIR, 'QUEEN.md'),
    path.join(HUB_DIR, 'version.json'),
    path.join(HUB_SYSTEM_DIR, 'aether-utils.sh'),
];

const missing = criticalFiles.filter(f => !fs.existsSync(f));
if (missing.length > 0 && !globalQuiet) {
    console.error('WARNING: Hub setup incomplete. Missing files:');
    missing.forEach(f => console.error(`  - ${f}`));
    console.error('Run: aether install --force');
}
```

**Additional hardening:**
- The `preinstall` script currently swallows all errors (`2>/dev/null || true`). This is correct for optional validation but should log failures to stderr for debugging.
- Add `--force` flag to `aether install` that re-creates the hub from scratch (for recovering from partial installs).

**Confidence:** HIGH -- based on direct analysis of `package.json` and `cli.js` install flow.

### 10. Continue-Advance State Write Protection

**Problem:** `continue-advance` instructs the LLM to write COLONY_STATE.json via the Write tool -- not through a bash subcommand, so no bash-level lock is held. If a slow builder's spawn-complete fires during this window, the LLM's full-file overwrite destroys the spawn-complete event.

**Pattern: Migrate state writes to bash subcommands with lock protection**

Instead of the LLM doing `Write tool -> COLONY_STATE.json`, the playbook should call:
```bash
bash .aether/aether-utils.sh state-advance --phase N --status "completed"
```

This routes through the existing lock infrastructure. The subcommand `state-advance` already exists and acquires the lock.

**Why this matters:** The LLM Write tool does a full-file overwrite with no concurrency awareness. The bash subcommands use acquire_lock -> read -> modify -> atomic_write -> release_lock. Routing state mutations through bash eliminates the race window.

**What NOT to do:**
- Do NOT add locking to the LLM Write tool (it's a Claude Code primitive, not under Aether's control)
- Do NOT remove the LLM's ability to write files generally (only restrict state file mutations)

**Confidence:** HIGH -- directly from Oracle Q2 finding 8. The `state-advance` subcommand exists and is already locked.

---

## What NOT to Do (Anti-Patterns to Avoid)

| Avoid | Why | What to Do Instead |
|-------|-----|-------------------|
| **Full rewrite of aether-utils.sh** | 11K lines, 178 subcommands, 572+ tests depend on current interface. Rewrite risk massively exceeds hardening risk. | Incremental extraction using existing `utils/` pattern. Dead code removal first. |
| **Migrate bash to Node.js** | The bash layer handles 178 subcommands of domain logic. Node.js is only the CLI entry point (installation, config). The split is correct. Migrating would double the Node.js surface area while adding zero reliability. | Keep the split. Harden each layer independently. |
| **Add SQLite for state storage** | Adds a native dependency to a CLI tool distributed via npm. Users need different SQLite binaries per platform. JSON files are sufficient at Aether's scale (dozens of entries, not millions). | Keep JSON files. Add schema validation at write boundaries. |
| **TypeScript migration** | The Node.js layer is 16 modules of thin CLI plumbing. TypeScript adds a build step, breaks existing tests, and provides minimal benefit for a codebase where the real logic is in bash/markdown. | Keep CJS JavaScript. Add JSDoc annotations where type clarity matters. |
| **bats-core migration** | 41 existing test files would need rewriting. The custom harness works. Migration effort provides marginal testing benefit. | Strengthen the existing harness with isolation, timeouts, and parallel safety. |
| **Add a database for lock management** | Over-engineering for a single-user CLI tool. File-based locks with PID tracking are the correct abstraction. | Fix the specific bugs (midden temp path, type coercion) rather than replacing the mechanism. |
| **Event sourcing for state changes** | COLONY_STATE.json is read-modify-write, not an event log. Event sourcing would require replaying events to reconstruct state, adding massive complexity for no benefit. | Checkpoint-before-mutate pattern (simple `cp` before writes). |

---

## Tooling Changes (Development Workflow)

### Dead Code Detection Approach

No tool exists for bash dead code detection. Use grep-based audit:

```bash
# List all subcommand names from the case statement:
grep -oE '^\s{2}[a-z][-a-z]*\)' .aether/aether-utils.sh | sed 's/)//' | sed 's/^\s*//' | sort > /tmp/all-subcommands.txt

# List all subcommand invocations across commands and playbooks:
grep -rohE 'aether-utils\.sh\s+[a-z][-a-z]+' .claude/ .opencode/ .aether/docs/ | awk '{print $2}' | sort -u > /tmp/used-subcommands.txt

# Diff to find dead code:
comm -23 /tmp/all-subcommands.txt /tmp/used-subcommands.txt
```

The Oracle audit already identified 76 dead subcommands. Categories: semantic search engine (6), swarm display/timing (10), view state management (6), learning display/selection (8), spawning diagnostics (5), suggest advanced (4), error/security advanced (5), miscellaneous (32).

### ShellCheck CI Integration

Expand from 6 files to all shell scripts:

```json
{
  "lint:shell": "shellcheck --severity=error .aether/aether-utils.sh .aether/utils/*.sh bin/validate-package.sh bin/generate-commands.sh",
  "lint:shell:strict": "shellcheck --severity=warning .aether/aether-utils.sh .aether/utils/*.sh"
}
```

The `lint:shell:strict` target enables gradual adoption of stricter checks without breaking CI.

### JSON Schema Validation at Write Boundaries

Use `jq` schema validation (no new dependencies) at state mutation boundaries:

```bash
# Validate COLONY_STATE.json structure after write:
_validate_colony_state() {
    local state_file="$1"
    local required_fields='["version","current_phase","events"]'

    local missing
    missing=$(jq -r --argjson req "$required_fields" \
        '$req[] as $f | if has($f) then empty else $f end' "$state_file" 2>/dev/null)

    if [[ -n "$missing" ]]; then
        json_err "$E_VALIDATION_FAILED" "COLONY_STATE.json missing fields: $missing"
    fi
}
```

This follows the existing `validate-state` subcommand pattern but adds it to the write path, not just the read path.

---

## Version Compatibility Constraints

| Dependency | Constraint | Reason |
|------------|-----------|--------|
| bash | 3.2+ | macOS ships bash 3.2 (GPLv2). Avoid bash 4+ features: no associative arrays, no `${var,,}` lowercasing, no `mapfile`. |
| jq | 1.5+ | Most systems have 1.6+. `tonumber?` (try-catch) available since 1.5. |
| Node.js | >=16.0.0 | Set in `package.json` engines. AVA v6 requires Node 20+, but the CLI itself works on 16+. Consider bumping to >=20 to match AVA requirement. |
| ShellCheck | 0.8+ | `--severity` flag available since 0.7. `.shellcheckrc` support since 0.7. |

### Node.js Engine Recommendation

The `package.json` declares `"node": ">=16.0.0"` but AVA v6 requires Node 20+. This means `npm test` fails on Node 16-19 even though the CLI itself works. Two options:

1. **Bump engines to `>=20.0.0`** -- aligns with test requirements, drops support for EOL Node versions
2. **Keep `>=16.0.0` and separate test requirements** -- users can run the tool on Node 16+ but can't run tests

**Recommendation:** Bump to `>=20.0.0`. Node 16 and 18 are both EOL. Users running Claude Code or OpenCode (Aether's target audience) will have Node 20+ installed.

**Confidence:** HIGH for the recommendation. Node 16 EOL was September 2023, Node 18 EOL was April 2025.

---

## Implementation Order

Based on dependency analysis and Oracle audit priority matrix:

| Order | Pattern | Effort | Impact | Dependencies |
|-------|---------|--------|--------|--------------|
| 1 | jq type coercion (#5) | ~5 lines | Fixes confirmed bug | None |
| 2 | Midden temp file race (#6) | ~3 lines | Fixes confirmed race | None |
| 3 | Memory pipeline circuit breaker (#4) | ~15 lines | Prevents permanent silent failure | None |
| 4 | State protection / checkpoints (#3) | ~20 lines | Prevents total loss | None |
| 5 | Continue-advance lock (#10) | ~10 lines playbook change | Closes race window | None |
| 6 | Dead code removal (from #2) | Large audit | -20% file size | None |
| 7 | Error handling triage (#1) | Large audit | Addresses root cause | Pattern established by #3, #4 |
| 8 | Monolith extraction (#2) | Medium per module | Maintainability | Dead code removal first |
| 9 | ShellCheck escalation (#7) | Medium | Catches latent bugs | Ideally after error triage |
| 10 | Test reliability (#8) | Medium | Prevents flaky tests | None (can parallelize) |
| 11 | Package distribution (#9) | Small | Better install UX | None |

Items 1-5 are independent, low-effort, high-impact fixes. Items 6-11 are larger efforts that benefit from items 1-5 being complete.

---

## Sources

- Oracle audit synthesis (`/.aether/oracle/synthesis.md`) -- 55 findings across 5 questions, 85% multi-source trust ratio (HIGH confidence)
- Oracle gaps analysis (`/.aether/oracle/gaps.md`) -- all questions answered at analytical ceiling (HIGH confidence)
- Direct codebase analysis: `aether-utils.sh` (11,272 lines), `utils/*.sh` (5,237 lines), `bin/lib/*.js` (16 modules) (HIGH confidence)
- [Bash Scripting Best Practices for Reliable Automation](https://oneuptime.com/blog/post/2026-02-13-bash-best-practices/view) -- `set -euo pipefail` patterns, trap cleanup (MEDIUM confidence)
- [Shell Scripting Best Practices for Production Systems](https://oneuptime.com/blog/post/2026-02-13-shell-scripting-best-practices/view) -- Error handling discipline (MEDIUM confidence)
- [How to Handle Error Handling with set -e in Bash](https://oneuptime.com/blog/post/2026-01-24-bash-set-e-error-handling/view) -- `set -e` pitfalls and workarounds (MEDIUM confidence)
- [ShellCheck severity documentation](https://shellcheck.net/wiki/severity) -- severity levels and `.shellcheckrc` configuration (HIGH confidence)
- [ShellCheck integration wiki](https://www.shellcheck.net/wiki/Integration) -- CI pipeline integration patterns (HIGH confidence)
- [Node.js error handling patterns](https://oneuptime.com/blog/post/2026-01-22-nodejs-error-handling-patterns/view) -- Structured error handling (MEDIUM confidence)
- [JSON file corruption prevention (Node.js)](https://github.com/nodejs/help/issues/2346) -- Atomic write patterns for JSON (HIGH confidence)
- [BATS-core documentation](https://bats-core.readthedocs.io/) -- Bash testing framework reference (HIGH confidence)
- [NPM Security Best Practices after Shai Hulud](https://snyk.io/articles/npm-security-best-practices-shai-hulud-attack/) -- postinstall script security context (MEDIUM confidence)
- Previous STACK.md (2026-03-19) for pheromone milestone -- confirms existing stack decisions still valid (HIGH confidence)

---
*Stack research for: Aether production hardening milestone*
*Researched: 2026-03-23*
